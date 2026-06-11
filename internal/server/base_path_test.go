package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

func TestHandlerBasePathServesHealthzAtRootAndBasePath(t *testing.T) {
	t.Parallel()
	handler := newBasePathTestServer("/admin").Handler()

	for _, path := range []string{"/healthz", "/admin/healthz"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			resp := serveTestRequest(handler, http.MethodGet, path)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("GET %s status = %d, want %d", path, resp.StatusCode, http.StatusOK)
			}
		})
	}
}

func TestHandlerBasePathRedirectsBarePrefix(t *testing.T) {
	t.Parallel()
	handler := newBasePathTestServer("/admin").Handler()

	resp := serveTestRequest(handler, http.MethodGet, "/admin?x=1")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
	}
	if got, want := resp.Header.Get("Location"), "/admin/?x=1"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestHandlerBasePathRedirectsProtectedPageToPrefixedLogin(t *testing.T) {
	t.Parallel()
	handler := newBasePathTestServer("/admin").Handler()

	resp := serveTestRequest(handler, http.MethodGet, "/admin/")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
	if got, want := resp.Header.Get("Location"), "/admin/login?next=%2F"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestHandlerBasePathReturnsUnauthorizedForAPI(t *testing.T) {
	t.Parallel()
	handler := newBasePathTestServer("/admin").Handler()

	resp := serveTestRequest(handler, http.MethodGet, "/admin/api/config")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandlerBasePathServesRuntimeConfig(t *testing.T) {
	t.Parallel()
	handler := newBasePathTestServer("/admin").Handler()

	resp := serveTestRequest(handler, http.MethodGet, "/admin/app-config.js")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got, want := resp.Header.Get("Content-Type"), "application/javascript; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"basePath":"/admin"`) {
		t.Fatalf("runtime config body = %q, want basePath /admin", string(body))
	}
}

func TestHandlerBasePathInjectsBaseHrefIntoIndex(t *testing.T) {
	t.Parallel()
	handler := newBasePathTestServer("/admin").Handler()

	resp := serveTestRequest(handler, http.MethodGet, "/admin/login")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	if !strings.Contains(got, `<base href="/admin/" />`) {
		t.Fatalf("index body = %q, want base href", got)
	}
	if !strings.Contains(got, `<script src="./app-config.js"></script>`) {
		t.Fatalf("index body = %q, want runtime config script", got)
	}
	configIndex := strings.Index(got, `<script src="./app-config.js"></script>`)
	moduleIndex := strings.Index(got, `<script type="module"`)
	if configIndex < 0 || moduleIndex < 0 || configIndex > moduleIndex {
		t.Fatalf("index body = %q, want runtime config script before module script", got)
	}
}

func TestHandlerRootPathRuntimeConfigAndIndex(t *testing.T) {
	t.Parallel()
	handler := newBasePathTestServer("").Handler()

	configResp := serveTestRequest(handler, http.MethodGet, "/app-config.js")
	defer configResp.Body.Close()
	body, err := io.ReadAll(configResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"basePath":""`) {
		t.Fatalf("runtime config body = %q, want empty basePath", string(body))
	}

	indexResp := serveTestRequest(handler, http.MethodGet, "/login")
	defer indexResp.Body.Close()
	indexBody, err := io.ReadAll(indexResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(indexBody), `<base href="/" />`) {
		t.Fatalf("index body = %q, want root base href", string(indexBody))
	}
}

func TestAuthCookiePathUsesBasePath(t *testing.T) {
	t.Parallel()
	auth := &Auth{
		basePath:   "/admin",
		cookiePath: "/admin",
		secret:     []byte("test-secret"),
	}
	rr := httptest.NewRecorder()
	auth.setSession(rr, sessionClaims{
		Subject: "admin",
		Tenant:  "admin",
		Expiry:  1893456000,
	})
	if got := rr.Header().Get("Set-Cookie"); !strings.Contains(got, "Path=/admin") {
		t.Fatalf("Set-Cookie = %q, want Path=/admin", got)
	}
}

func TestSanitizeNextStripsBasePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		next string
		want string
	}{
		{name: "empty", next: "", want: "/"},
		{name: "external", next: "https://example.com/", want: "/"},
		{name: "protocol relative", next: "//example.com/", want: "/"},
		{name: "relative", next: "settings", want: "/"},
		{name: "base root", next: "/admin", want: "/"},
		{name: "base child", next: "/admin/settings?tab=tasks", want: "/settings?tab=tasks"},
		{name: "plain app path", next: "/settings", want: "/settings"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeNext(tt.next, "/admin"); got != tt.want {
				t.Fatalf("sanitizeNext(%q) = %q, want %q", tt.next, got, tt.want)
			}
		})
	}
}

func newBasePathTestServer(basePath string) *Server {
	basePath = config.NormalizeBasePath(basePath)
	cookiePath := "/"
	if basePath != "" {
		cookiePath = basePath
	}
	return &Server{
		web: fstest.MapFS{
			"web/dist/index.html": {
				Data: []byte(`<!doctype html><html><head><title>test</title></head><body><div id="root"></div><script type="module" src="./assets/index.js"></script></body></html>`),
			},
			"web/dist/assets/index.js": {
				Data: []byte(`console.log("test");`),
			},
			"web/dist/logo.png": {
				Data: []byte("png"),
			},
		},
		basePath: basePath,
		auth: &Auth{
			basePath:   basePath,
			cookiePath: cookiePath,
			secret:     []byte("test-secret"),
		},
	}
}

func serveTestRequest(handler http.Handler, method string, path string) *http.Response {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(method, "http://example.test"+path, nil)
	handler.ServeHTTP(rr, req)
	return rr.Result()
}
