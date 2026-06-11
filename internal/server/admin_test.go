package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

func TestAdminUsersRequireAdministrator(t *testing.T) {
	srv := newRestoreAPITestServer(t, config.Default())
	if _, err := srv.store.CreateUser("user", "password", "Regular User", "", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	req := newSessionJSONRequest(t, srv, sessionClaims{
		Subject: "user",
		Tenant:  "user",
		Expiry:  time.Now().Add(time.Hour).Unix(),
	}, http.MethodGet, "/api/admin/users", "")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func newSessionJSONRequest(t *testing.T, srv *Server, claims sessionClaims, method, path, body string) *http.Request {
	t.Helper()
	rr := httptest.NewRecorder()
	srv.auth.setSession(rr, claims)
	req := httptest.NewRequest(method, "http://example.test"+path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}
