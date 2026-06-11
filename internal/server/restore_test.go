package server

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

func TestRestoreEndpointsRequireAuth(t *testing.T) {
	handler := newBasePathTestServer("").Handler()

	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/api/restore/browse", body: `{}`},
		{method: http.MethodPost, path: "/api/restore/objects", body: `{}`},
		{method: http.MethodPost, path: "/api/restore/jobs", body: `{}`},
		{method: http.MethodGet, path: "/api/restore/jobs/job-id"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://example.test"+tc.path, bytes.NewBufferString(tc.body))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestRestoreObjectsRejectsMissingTarget(t *testing.T) {
	srv := newRestoreAPITestServer(t, config.Default())

	req := newAuthenticatedJSONRequest(t, srv, http.MethodPost, "/api/restore/objects", `{"target_key":"missing","prefix":"backups"}`)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestRestoreJobsRejectsNonNowledgeDestination(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NMEM_SNAP_ALLOWED_SOURCE_ROOTS", root)
	cfg := config.Default()
	cfg.Targets = []config.TargetConfig{{
		Key:     "target",
		Name:    "Target",
		Enabled: true,
		Type:    "s3",
		S3: config.S3Config{
			EndpointURL: "http://127.0.0.1:9000",
			Region:      "auto",
			BucketName:  "bucket",
			AccessKeyID: "access",
		},
	}}
	cfg.Sources = append(cfg.Sources, config.SourceConfig{
		Key:     "dir",
		Name:    "Directory",
		Enabled: true,
		Type:    "directory",
		Directory: config.DirectorySource{
			Path: root,
		},
	})
	srv := newRestoreAPITestServer(t, cfg)

	body := `{"target_key":"target","object_name":"backups/export.zip","destination_source_key":"dir"}`
	req := newAuthenticatedJSONRequest(t, srv, http.MethodPost, "/api/restore/jobs", body)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("must be nowledgemem_api")) {
		t.Fatalf("body = %s, want destination type error", rr.Body.String())
	}
}

func newRestoreAPITestServer(t *testing.T, cfg config.Config) *Server {
	t.Helper()
	store := config.NewStore(t.TempDir())
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	if err := store.CreateLocalUser("admin", "password", true); err != nil {
		t.Fatalf("CreateLocalUser: %v", err)
	}
	if err := store.SaveUser("admin", cfg); err != nil {
		t.Fatalf("SaveUser: %v", err)
	}
	srv, err := New(
		context.Background(),
		config.Default(),
		store,
		fstest.MapFS{
			"web/dist/index.html": {Data: []byte(`<!doctype html><html><head></head><body></body></html>`)},
		},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv
}

func newAuthenticatedJSONRequest(t *testing.T, srv *Server, method, path, body string) *http.Request {
	t.Helper()
	rr := httptest.NewRecorder()
	srv.auth.setSession(rr, sessionClaims{
		Subject: "admin",
		Tenant:  "admin",
		Expiry:  time.Now().Add(time.Hour).Unix(),
	})
	req := httptest.NewRequest(method, "http://example.test"+path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}
