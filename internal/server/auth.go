package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

const (
	sessionCookie = "nmem_snap_session"
	oidcCookie    = "nmem_snap_oidc"
)

type Auth struct {
	cfg        config.AuthConfig
	store      *config.Store
	secret     []byte
	basePath   string
	cookiePath string
	oidc       *oidcRuntime
}

type oidcRuntime struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth2   oauth2.Config
	cfg      config.OIDCConfig
}

type sessionClaims struct {
	Subject string `json:"sub"`
	Email   string `json:"email,omitempty"`
	Tenant  string `json:"tenant"`
	Expiry  int64  `json:"exp"`
}

type oidcState struct {
	State  string `json:"state"`
	Nonce  string `json:"nonce"`
	Next   string `json:"next"`
	Mode   string `json:"mode,omitempty"`
	Expiry int64  `json:"exp"`
}

type oidcClaims struct {
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Subject           string `json:"sub"`
	Name              string `json:"name"`
	Nickname          string `json:"nickname"`
	PreferredUsername string `json:"preferred_username"`
	Picture           string `json:"picture"`
}

func NewAuth(ctx context.Context, cfg config.AuthConfig, store *config.Store, basePath string, port int) (*Auth, error) {
	secret := os.Getenv(cfg.SessionSecretEnv)
	if secret == "" {
		secret = os.Getenv("NMEM_SNAP_PASSWORD")
	}
	if secret == "" {
		secret = "dev-session-secret-change-me"
	}
	basePath = config.NormalizeBasePath(basePath)
	cookiePath := "/"
	if basePath != "" {
		cookiePath = basePath
	}
	a := &Auth{
		cfg:        cfg,
		store:      store,
		secret:     []byte(secret),
		basePath:   basePath,
		cookiePath: cookiePath,
	}
	if cfg.OIDC.Enabled {
		if cfg.OIDC.RedirectURL == "" {
			if port == 0 {
				port = config.DefaultPort
			}
			cfg.OIDC.RedirectURL = fmt.Sprintf("http://localhost:%d%s/auth/oidc/callback", port, basePath)
		}
		provider, err := oidc.NewProvider(ctx, cfg.OIDC.IssuerURL)
		if err != nil {
			return nil, fmt.Errorf("initialize OIDC provider: %w", err)
		}
		oauthCfg := oauth2.Config{
			ClientID:     cfg.OIDC.ClientID,
			ClientSecret: cfg.OIDC.ClientSecret,
			RedirectURL:  cfg.OIDC.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       cfg.OIDC.Scopes,
		}
		a.oidc = &oidcRuntime{
			provider: provider,
			verifier: provider.Verifier(&oidc.Config{
				ClientID: cfg.OIDC.ClientID,
			}),
			oauth2: oauthCfg,
			cfg:    cfg.OIDC,
		}
	}
	return a, nil
}

func (a *Auth) Enabled() bool {
	return true
}

func (a *Auth) OIDCEnabled() bool {
	return a.oidc != nil
}

func (a *Auth) PasswordLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	user, ok, err := a.store.VerifyLocalUser(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	a.setSession(w, sessionClaims{
		Subject: user.Username,
		Tenant:  user.Tenant,
		Expiry:  time.Now().Add(24 * time.Hour).Unix(),
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Auth) Logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     a.cookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Auth) LoginOptions() map[string]any {
	return map[string]any{
		"password": true,
		"oidc":     a.oidc != nil,
		"username": a.cfg.Username,
	}
}

func (a *Auth) StartOIDC(w http.ResponseWriter, r *http.Request) {
	if a.oidc == nil {
		http.NotFound(w, r)
		return
	}
	next := sanitizeNext(r.URL.Query().Get("next"), a.basePath)
	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	if mode != "link" {
		mode = "login"
	}
	if mode == "link" {
		if _, ok := a.Session(r); !ok {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
	}
	state := randomToken()
	nonce := randomToken()
	encoded, err := a.signOIDCState(oidcState{
		State:  state,
		Nonce:  nonce,
		Next:   next,
		Mode:   mode,
		Expiry: time.Now().Add(10 * time.Minute).Unix(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create OIDC state")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcCookie,
		Value:    encoded,
		Path:     a.cookiePath,
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, a.oidc.oauth2.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
}

func (a *Auth) OIDCCallback(w http.ResponseWriter, r *http.Request) {
	if a.oidc == nil {
		http.NotFound(w, r)
		return
	}
	stateCookie, err := r.Cookie(oidcCookie)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing OIDC state")
		return
	}
	state, err := a.verifyOIDCState(stateCookie.Value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid OIDC state")
		return
	}
	if state.State != r.URL.Query().Get("state") {
		writeError(w, http.StatusBadRequest, "OIDC state mismatch")
		return
	}
	oauth2Token, err := a.oidc.oauth2.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "OIDC code exchange failed")
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "OIDC id_token missing")
		return
	}
	idToken, err := a.oidc.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "OIDC token verification failed")
		return
	}
	if idToken.Nonce != state.Nonce {
		writeError(w, http.StatusUnauthorized, "OIDC nonce mismatch")
		return
	}
	var claims oidcClaims
	if err := idToken.Claims(&claims); err != nil {
		writeError(w, http.StatusUnauthorized, "OIDC claims invalid")
		return
	}
	claims = a.mergeUserInfoClaims(r.Context(), oauth2Token, claims)
	if !a.emailAllowed(claims.Email) {
		writeError(w, http.StatusForbidden, "OIDC user is not allowed")
		return
	}
	profile := config.OIDCProfile{
		Issuer:            a.oidc.cfg.IssuerURL,
		Subject:           claims.Subject,
		Email:             claims.Email,
		EmailVerified:     claims.EmailVerified,
		Username:          defaultString(claims.PreferredUsername, claims.Email),
		DisplayName:       defaultString(claims.Name, claims.Nickname),
		AvatarURL:         claims.Picture,
		PreferredUsername: claims.PreferredUsername,
		Nickname:          claims.Nickname,
	}
	if state.Mode == "link" {
		session, ok := a.Session(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		linked, err := a.store.LinkOIDCIdentity(session.Tenant, profile)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		a.setSession(w, sessionClaims{
			Subject: linked.Username,
			Email:   claims.Email,
			Tenant:  linked.Tenant,
			Expiry:  time.Now().Add(24 * time.Hour).Unix(),
		})
		a.clearOIDCCookie(w)
		http.Redirect(w, r, a.withBase(sanitizeNext(state.Next, a.basePath)), http.StatusFound)
		return
	}
	user, err := a.store.UpsertOIDCUser(profile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to prepare OIDC user")
		return
	}
	a.setSession(w, sessionClaims{
		Subject: user.Username,
		Email:   claims.Email,
		Tenant:  user.Tenant,
		Expiry:  time.Now().Add(24 * time.Hour).Unix(),
	})
	a.clearOIDCCookie(w)
	http.Redirect(w, r, a.withBase(sanitizeNext(state.Next, a.basePath)), http.StatusFound)
}

func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := a.Session(r); ok {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		nextPath := sanitizeNext(r.URL.RequestURI(), a.basePath)
		http.Redirect(w, r, a.withBase("/login")+"?next="+url.QueryEscape(nextPath), http.StatusFound)
	})
}

func (a *Auth) Session(r *http.Request) (sessionClaims, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return sessionClaims{}, false
	}
	claims, err := a.verifySession(cookie.Value)
	if err != nil || claims.Expiry < time.Now().Unix() {
		return sessionClaims{}, false
	}
	return claims, true
}

func (a *Auth) setSession(w http.ResponseWriter, claims sessionClaims) {
	value, err := a.signJSON(claims)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    value,
		Path:     a.cookiePath,
		MaxAge:   int(time.Until(time.Unix(claims.Expiry, 0)).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *Auth) clearOIDCCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oidcCookie,
		Value:    "",
		Path:     a.cookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *Auth) verifySession(value string) (sessionClaims, error) {
	var claims sessionClaims
	if err := a.verifyJSON(value, &claims); err != nil {
		return sessionClaims{}, err
	}
	return claims, nil
}

func (a *Auth) signOIDCState(state oidcState) (string, error) {
	return a.signJSON(state)
}

func (a *Auth) verifyOIDCState(value string) (oidcState, error) {
	var state oidcState
	if err := a.verifyJSON(value, &state); err != nil {
		return oidcState{}, err
	}
	if state.Expiry < time.Now().Unix() {
		return oidcState{}, fmt.Errorf("state expired")
	}
	return state, nil
}

func (a *Auth) signJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig, nil
}

func (a *Auth) verifyJSON(value string, dst any) error {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid token")
	}
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(parts[0]))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}
	if !hmac.Equal(got, want) {
		return fmt.Errorf("invalid signature")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func (a *Auth) mergeUserInfoClaims(ctx context.Context, token *oauth2.Token, claims oidcClaims) oidcClaims {
	if a.oidc == nil || token == nil {
		return claims
	}
	userInfo, err := a.oidc.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return claims
	}
	var extra oidcClaims
	if err := userInfo.Claims(&extra); err != nil {
		return claims
	}
	claims.Email = defaultString(claims.Email, extra.Email)
	claims.Subject = defaultString(claims.Subject, extra.Subject)
	claims.Name = defaultString(claims.Name, extra.Name)
	claims.Nickname = defaultString(claims.Nickname, extra.Nickname)
	claims.PreferredUsername = defaultString(claims.PreferredUsername, extra.PreferredUsername)
	claims.Picture = defaultString(claims.Picture, extra.Picture)
	if !claims.EmailVerified {
		claims.EmailVerified = extra.EmailVerified
	}
	return claims
}

func (a *Auth) emailAllowed(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(a.oidc.cfg.AllowedEmails) == 0 && len(a.oidc.cfg.AllowedDomains) == 0 {
		return true
	}
	for _, allowed := range a.oidc.cfg.AllowedEmails {
		if email == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}
	domain := ""
	if at := strings.LastIndex(email, "@"); at >= 0 {
		domain = email[at+1:]
	}
	for _, allowed := range a.oidc.cfg.AllowedDomains {
		if domain == strings.ToLower(strings.TrimPrefix(strings.TrimSpace(allowed), "@")) {
			return true
		}
	}
	return false
}

func randomToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b[:])
}

func sanitizeNext(next string, basePath string) string {
	if next == "" {
		return "/"
	}
	if strings.HasPrefix(next, "http://") || strings.HasPrefix(next, "https://") || strings.HasPrefix(next, "//") {
		return "/"
	}
	if !strings.HasPrefix(next, "/") {
		return "/"
	}
	basePath = config.NormalizeBasePath(basePath)
	if basePath != "" {
		if next == basePath {
			return "/"
		}
		if strings.HasPrefix(next, basePath+"/") {
			return strings.TrimPrefix(next, basePath)
		}
	}
	return next
}

func (a *Auth) withBase(path string) string {
	if path == "" || path[0] != '/' {
		path = "/" + path
	}
	if a.basePath == "" {
		return path
	}
	if path == "/" {
		return a.basePath + "/"
	}
	return a.basePath + path
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
