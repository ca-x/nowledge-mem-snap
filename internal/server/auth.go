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

	"github.com/lib-x/nowledge-mem-snap/internal/config"
)

const (
	sessionCookie = "nmem_snap_session"
	oidcCookie    = "nmem_snap_oidc"
)

type Auth struct {
	cfg    config.AuthConfig
	store  *config.Store
	secret []byte
	oidc   *oidcRuntime
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
	Expiry int64  `json:"exp"`
}

func NewAuth(ctx context.Context, cfg config.AuthConfig, store *config.Store) (*Auth, error) {
	secret := os.Getenv(cfg.SessionSecretEnv)
	if secret == "" {
		secret = os.Getenv("NMEM_SNAP_PASSWORD")
	}
	if secret == "" {
		secret = "dev-session-secret-change-me"
	}
	a := &Auth{
		cfg:    cfg,
		store:  store,
		secret: []byte(secret),
	}
	if cfg.OIDC.Enabled {
		if cfg.OIDC.RedirectURL == "" {
			cfg.OIDC.RedirectURL = fmt.Sprintf("http://localhost:%d/auth/oidc/callback", config.DefaultPort)
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
		Path:     "/",
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
	next := sanitizeNext(r.URL.Query().Get("next"))
	state := randomToken()
	nonce := randomToken()
	encoded, err := a.signOIDCState(oidcState{
		State:  state,
		Nonce:  nonce,
		Next:   next,
		Expiry: time.Now().Add(10 * time.Minute).Unix(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create OIDC state")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcCookie,
		Value:    encoded,
		Path:     "/",
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
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Subject       string `json:"sub"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		writeError(w, http.StatusUnauthorized, "OIDC claims invalid")
		return
	}
	if !a.emailAllowed(claims.Email) {
		writeError(w, http.StatusForbidden, "OIDC user is not allowed")
		return
	}
	tenant := tenantFromIdentity(claims.Subject, claims.Email)
	user, err := a.store.UpsertExternalUser(tenant, defaultString(claims.Email, claims.Subject), defaultString(claims.Name, claims.Email), claims.Picture)
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
	http.SetCookie(w, &http.Cookie{
		Name:     oidcCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, sanitizeNext(state.Next), http.StatusFound)
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
		http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusFound)
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
		Path:     "/",
		MaxAge:   int(time.Until(time.Unix(claims.Expiry, 0)).Seconds()),
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

func tenantFromIdentity(subject, email string) string {
	if tenant := config.TenantKey(subject); tenant != "" {
		return tenant
	}
	if tenant := config.TenantKey(email); tenant != "" {
		return tenant
	}
	return "unknown"
}

func sanitizeNext(next string) string {
	if next == "" {
		return "/"
	}
	if strings.HasPrefix(next, "http://") || strings.HasPrefix(next, "https://") || strings.HasPrefix(next, "//") {
		return "/"
	}
	if !strings.HasPrefix(next, "/") {
		return "/"
	}
	return next
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
