package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/backup"
	"github.com/ca-x/nowledge-mem-snap/internal/config"
	"github.com/ca-x/nowledge-mem-snap/internal/history"
	"github.com/ca-x/nowledge-mem-snap/internal/restore"
	"github.com/ca-x/nowledge-mem-snap/internal/source"
	"github.com/ca-x/nowledge-mem-snap/internal/storage"
	"github.com/ca-x/nowledge-mem-snap/version"
)

type Server struct {
	cfg             config.Config
	store           *config.Store
	auth            *Auth
	web             fs.FS
	logger          *slog.Logger
	basePath        string
	onConfigChanged func(tenant string)
	taskRuntime     func(tenant string) map[string]config.TaskRuntime
	restoreManager  *restore.Manager
}

func New(ctx context.Context, cfg config.Config, store *config.Store, web fs.FS, logger *slog.Logger, onConfigChanged func(tenant string), taskRuntime func(tenant string) map[string]config.TaskRuntime) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}
	basePath := config.NormalizeBasePath(cfg.Listen.BasePath)
	if err := config.ValidateBasePath(basePath); err != nil {
		return nil, err
	}
	cfg.Listen.BasePath = basePath
	auth, err := NewAuth(ctx, cfg.Auth, store, basePath, cfg.Listen.Port)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:             cfg,
		store:           store,
		auth:            auth,
		web:             web,
		logger:          logger,
		basePath:        basePath,
		onConfigChanged: onConfigChanged,
		taskRuntime:     taskRuntime,
		restoreManager:  restore.NewManager(logger),
	}, nil
}

func (s *Server) Handler() http.Handler {
	inner := s.innerHandler()
	if s.basePath == "" {
		return secureHeaders(inner)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc(s.basePath, func(w http.ResponseWriter, r *http.Request) {
		target := s.basePath + "/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
	mux.Handle(s.basePath+"/", http.StripPrefix(s.basePath, inner))
	return secureHeaders(mux)
}

func (s *Server) innerHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/app-config.js", s.handleAppConfig)
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
	api.HandleFunc("/api/version", s.handleVersion)
	api.HandleFunc("/api/sources/test", s.handleTestSource)
	api.HandleFunc("/api/sources/test/download", s.handleDownloadTestSource)
	api.HandleFunc("/api/targets/test", s.handleTestTarget)
	api.HandleFunc("/api/runs", s.handleRuns)
	api.HandleFunc("/api/backup/run", s.handleRunBackup)
	api.HandleFunc("/api/restore/objects", s.handleRestoreObjects)
	api.HandleFunc("/api/restore/jobs", s.handleRestoreJobs)
	api.HandleFunc("/api/restore/jobs/", s.handleRestoreJob)
	mux.Handle("/api/", s.auth.Require(api))

	static := s.staticHandler()
	mux.Handle("/assets/", static)
	mux.Handle("/logo.png", static)
	mux.Handle("/favicon.ico", static)
	mux.Handle("/login", static)
	mux.Handle("/", s.auth.Require(static))
	return mux
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
		writeJSON(w, http.StatusOK, s.withTaskRuntime(claims.Tenant, config.Redacted(cfg)))
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
		writeJSON(w, http.StatusOK, s.withTaskRuntime(claims.Tenant, config.Redacted(loaded)))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) withTaskRuntime(tenant string, cfg config.Config) config.Config {
	if s.taskRuntime != nil {
		cfg.TaskRuntime = s.taskRuntime(tenant)
	}
	return cfg
}

func (s *Server) handleSourceRoots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roots": config.AllowedDirectoryRoots()})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version":    version.Version,
		"build_time": version.BuildTime,
		"git_commit": version.GitCommit,
		"full":       version.Full(),
	})
}

func (s *Server) handleTestSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	claims, ok := s.auth.Session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req config.SourceConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if cfg, err := s.store.LoadUser(claims.Tenant); err == nil {
		if existing, ok := cfg.Source(req.Key); ok {
			req = config.MergeSourceSecrets(req, existing)
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	result := source.Test(ctx, req)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleDownloadTestSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	claims, ok := s.auth.Session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req struct {
		Source config.SourceConfig `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	sourceCfg := req.Source
	if cfg, err := s.store.LoadUser(claims.Tenant); err == nil {
		if existing, ok := cfg.Source(sourceCfg.Key); ok {
			sourceCfg = config.MergeSourceSecrets(sourceCfg, existing)
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	s.logger.Info("source test download started", "tenant", claims.Tenant, "source", sourceCfg.Key, "type", sourceCfg.Type)
	data, result := source.DownloadTest(ctx, sourceCfg, config.DefaultExportConfig())
	if !result.OK {
		s.logger.Warn("source test download failed", "tenant", claims.Tenant, "source", sourceCfg.Key, "type", sourceCfg.Type, "code", result.Code, "message", result.Message)
		writeJSON(w, http.StatusBadRequest, result)
		return
	}
	filename := fmt.Sprintf("source-test-%s.zip", time.Now().UTC().Format("20060102T150405Z"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		s.logger.Warn("source test download write failed", "tenant", claims.Tenant, "source", sourceCfg.Key, "error", err)
		return
	}
	s.logger.Info("source test download finished", "tenant", claims.Tenant, "source", sourceCfg.Key, "type", sourceCfg.Type, "bytes", len(data))
}

func (s *Server) handleTestTarget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	claims, ok := s.auth.Session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req struct {
		Target     config.TargetConfig `json:"target"`
		UploadFile bool                `json:"upload_file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	targetCfg := req.Target
	if cfg, err := s.store.LoadUser(claims.Tenant); err == nil {
		if existing, ok := cfg.Target(targetCfg.Key); ok {
			targetCfg = config.MergeTargetSecrets(targetCfg, existing)
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()
	s.logger.Info("target test started", "tenant", claims.Tenant, "target", targetCfg.Key, "type", targetCfg.Type, "upload_file", req.UploadFile)
	result := storage.Test(ctx, targetCfg, req.UploadFile)
	if !result.OK {
		s.logger.Warn("target test failed", "tenant", claims.Tenant, "target", targetCfg.Key, "type", targetCfg.Type, "upload_file", req.UploadFile, "code", result.Code, "message", result.Message)
		writeJSON(w, http.StatusOK, result)
		return
	}
	s.logger.Info("target test finished", "tenant", claims.Tenant, "target", targetCfg.Key, "type", targetCfg.Type, "upload_file", req.UploadFile, "bytes", result.Details["bytes"])
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

func (s *Server) handleRestoreObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	claims, ok := s.auth.Session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req struct {
		TargetKey string `json:"target_key"`
		Prefix    string `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, err := s.store.LoadUser(claims.Tenant)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := cfg.Target(strings.TrimSpace(req.TargetKey))
	if !ok {
		writeError(w, http.StatusNotFound, "target was not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	objects, err := s.restoreManager.ListObjects(ctx, target, req.Prefix)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"objects": objects})
}

func (s *Server) handleRestoreJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	claims, ok := s.auth.Session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req struct {
		TargetKey            string                `json:"target_key"`
		ObjectName           string                `json:"object_name"`
		DestinationSourceKey string                `json:"destination_source_key"`
		EncryptionPassword   string                `json:"encryption_password"`
		ImportOptions        restore.ImportOptions `json:"import_options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, err := s.store.LoadUser(claims.Tenant)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := cfg.Target(strings.TrimSpace(req.TargetKey))
	if !ok {
		writeError(w, http.StatusNotFound, "target was not found")
		return
	}
	destination, ok := cfg.Source(strings.TrimSpace(req.DestinationSourceKey))
	if !ok {
		writeError(w, http.StatusNotFound, "destination source was not found")
		return
	}
	job, err := s.restoreManager.Start(r.Context(), restore.StartRequest{
		Tenant:             claims.Tenant,
		Target:             target,
		Destination:        destination,
		ObjectName:         req.ObjectName,
		EncryptionPassword: req.EncryptionPassword,
		ImportOptions:      req.ImportOptions,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.logger.Info("restore job accepted", "tenant", claims.Tenant, "job", job.ID, "target", target.Key, "destination_source", destination.Key, "object", job.ObjectName)
	writeJSON(w, http.StatusAccepted, map[string]any{"job_id": job.ID})
}

func (s *Server) handleRestoreJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	claims, ok := s.auth.Session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/restore/jobs/")
	id = strings.TrimSpace(strings.Trim(id, "/"))
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "restore job was not found")
		return
	}
	job, ok := s.restoreManager.Get(id)
	if !ok || (job.Tenant != "" && job.Tenant != claims.Tenant) {
		writeError(w, http.StatusNotFound, "restore job was not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
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
		if p == "" || p == "index.html" || p == "login" || strings.HasPrefix(p, "auth/") {
			s.serveIndex(w, sub)
			return
		}
		if _, err := fs.Stat(sub, p); err != nil {
			s.serveIndex(w, sub)
			return
		}
		files.ServeHTTP(w, r)
	})
}

func (s *Server) handleAppConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	payload, err := json.Marshal(map[string]string{"basePath": s.basePath})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build app config")
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprintf(w, "window.__NMEM_SNAP_CONFIG__ = %s;\n", payload)
}

func (s *Server) serveIndex(w http.ResponseWriter, sub fs.FS) {
	data, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "web assets unavailable")
		return
	}
	body := string(data)
	base := html.EscapeString(s.baseHref())
	if strings.Contains(body, "<head>") {
		body = strings.Replace(body, "<head>", "<head>\n    <base href=\""+base+"\" />", 1)
	} else {
		body = "<base href=\"" + base + "\" />\n" + body
	}
	script := "    <script src=\"./app-config.js\"></script>"
	if strings.Contains(body, "<script type=\"module\"") {
		body = strings.Replace(body, "<script type=\"module\"", script+"\n    <script type=\"module\"", 1)
	} else if strings.Contains(body, "</body>") {
		body = strings.Replace(body, "</body>", script+"\n  </body>", 1)
	} else {
		body += "\n" + script + "\n"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(body))
}

func (s *Server) baseHref() string {
	if s.basePath == "" {
		return "/"
	}
	return s.basePath + "/"
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
