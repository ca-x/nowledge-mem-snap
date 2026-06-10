package server

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lib-x/nowledge-mem-snap/internal/backup"
	"github.com/lib-x/nowledge-mem-snap/internal/config"
	"github.com/lib-x/nowledge-mem-snap/internal/history"
	"github.com/lib-x/nowledge-mem-snap/internal/source"
)

type Server struct {
	cfg             config.Config
	store           *config.Store
	auth            *Auth
	web             embed.FS
	logger          *slog.Logger
	onConfigChanged func(tenant string)
}

func New(ctx context.Context, cfg config.Config, store *config.Store, web embed.FS, logger *slog.Logger, onConfigChanged func(tenant string)) (*Server, error) {
	auth, err := NewAuth(ctx, cfg.Auth, store)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:             cfg,
		store:           store,
		auth:            auth,
		web:             web,
		logger:          logger,
		onConfigChanged: onConfigChanged,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/auth/options", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.auth.LoginOptions())
	})
	mux.HandleFunc("/api/auth/login", s.auth.PasswordLogin)
	mux.HandleFunc("/api/auth/logout", s.auth.Logout)
	mux.HandleFunc("/api/setup/status", s.handleSetupStatus)
	mux.HandleFunc("/api/setup", s.handleSetup)
	mux.HandleFunc("/auth/oidc/start", s.auth.StartOIDC)
	mux.HandleFunc("/auth/oidc/callback", s.auth.OIDCCallback)

	api := http.NewServeMux()
	api.HandleFunc("/api/session", s.handleSession)
	api.HandleFunc("/api/profile", s.handleProfile)
	api.HandleFunc("/api/config", s.handleConfig)
	api.HandleFunc("/api/source-roots", s.handleSourceRoots)
	api.HandleFunc("/api/sources/test", s.handleTestSource)
	api.HandleFunc("/api/runs", s.handleRuns)
	api.HandleFunc("/api/backup/run", s.handleRunBackup)
	mux.Handle("/api/", s.auth.Require(api))

	static := s.staticHandler()
	mux.Handle("/login", static)
	mux.Handle("/", s.auth.Require(static))
	return secureHeaders(mux)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := s.auth.Session(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": ok,
		"user":          claims,
	})
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := s.auth.Session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		profile, err := s.store.Profile(claims.Tenant)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, profile)
	case http.MethodPut:
		var req struct {
			DisplayName string `json:"display_name"`
			AvatarURL   string `json:"avatar_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		profile, err := s.store.UpdateProfile(claims.Tenant, req.DisplayName, req.AvatarURL)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, profile)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	claims, ok := s.auth.Session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.store.LoadUser(claims.Tenant)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, config.Redacted(cfg))
	case http.MethodPut:
		var cfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if err := s.store.SaveUser(claims.Tenant, cfg); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if s.onConfigChanged != nil {
			s.onConfigChanged(claims.Tenant)
		}
		loaded, err := s.store.LoadUser(claims.Tenant)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, config.Redacted(loaded))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSourceRoots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roots": config.AllowedDirectoryRoots()})
}

func (s *Server) handleTestSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req config.SourceConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	result := source.Test(ctx, req)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	claims, ok := s.auth.Session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	cfg, err := s.store.LoadUser(claims.Tenant)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	runs, err := history.NewStoreWithRetention(s.store.Client(), claims.Tenant, cfg.HistoryLimit, cfg.HistoryRetentionDays).List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (s *Server) handleRunBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TaskKey string `json:"task_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	claims, ok := s.auth.Session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	cfg, err := s.store.LoadUser(claims.Tenant)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	historyStore := history.NewStoreWithRetention(s.store.Client(), claims.Tenant, cfg.HistoryLimit, cfg.HistoryRetentionDays)
	run, err := backup.NewRunner(cfg, historyStore, s.logger).RunTask(r.Context(), req.TaskKey)
	if err != nil {
		writeJSON(w, http.StatusAccepted, map[string]any{"run": run, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"run": run})
}

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	required, err := s.store.SetupRequired()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"setup_required": required})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	required, err := s.store.SetupRequired()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !required {
		writeError(w, http.StatusConflict, "setup has already completed")
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.store.CreateLocalUser(req.Username, req.Password, true); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, ok, err := s.store.VerifyLocalUser(req.Username, req.Password)
	if err != nil || !ok {
		writeError(w, http.StatusInternalServerError, "setup created user but login failed")
		return
	}
	s.auth.setSession(w, sessionClaims{
		Subject: user.Username,
		Tenant:  user.Tenant,
		Expiry:  time.Now().Add(24 * time.Hour).Unix(),
	})
	if s.onConfigChanged != nil {
		s.onConfigChanged(user.Tenant)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) staticHandler() http.Handler {
	sub, err := fs.Sub(s.web, "web/dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusInternalServerError, "web assets unavailable")
		})
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" || p == "login" || strings.HasPrefix(p, "auth/") {
			r.URL.Path = "/"
			files.ServeHTTP(w, r)
			return
		}
		if _, err := fs.Stat(sub, p); err != nil {
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
