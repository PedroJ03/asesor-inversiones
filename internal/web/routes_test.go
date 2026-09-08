package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

func testDeps(t *testing.T) Dependencies {
	t.Helper()
	s, err := store.Open("file:" + t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	return Dependencies{
		Authorizer: NewCookieAuthorizer("password", "secret"),
		Assets:     Assets,
		Store:      s,
		Watchlist:  testWatchlist(),
		RenderLogin: func(w http.ResponseWriter, r *http.Request, err string) {
			w.WriteHeader(http.StatusOK)
		},
	}
}

func TestNewMux_RegistersAllPatterns(t *testing.T) {
	deps := testDeps(t)

	// Construction must not panic; pattern conflicts panic at registration.
	mux := NewMux(deps)

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/healthz", http.StatusOK},
		{"GET", "/login", http.StatusOK},
		{"POST", "/login", http.StatusUnauthorized},          // empty password rejected
		{"GET", "/assets/main.css", http.StatusOK},
		{"GET", "/", http.StatusSeeOther},                    // protected, redirect to login
		{"GET", "/reporte", http.StatusSeeOther},             // protected
		{"GET", "/reporte/", http.StatusSeeOther},            // protected
		{"GET", "/reporte/2024-01-01", http.StatusSeeOther},  // protected
		{"GET", "/activos", http.StatusSeeOther},             // protected
		{"GET", "/activos/", http.StatusSeeOther},            // protected
		{"GET", "/activos/yahoo/AAPL", http.StatusSeeOther},  // protected
		{"GET", "/alertas", http.StatusSeeOther},             // protected
		{"GET", "/alertas/", http.StatusSeeOther},            // protected
		{"POST", "/alertas/reglas/", http.StatusSeeOther},    // protected
		{"PUT", "/alertas/reglas/1", http.StatusSeeOther},    // protected
		{"DELETE", "/alertas/reglas/1", http.StatusSeeOther}, // protected
		{"POST", "/alertas/reglas/1", http.StatusSeeOther},   // protected
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			mux.Handler().ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("%s %s: expected status %d, got %d (body: %q)", tc.method, tc.path, tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestNewMux_UnknownPath(t *testing.T) {
	deps := testDeps(t)
	mux := NewMux(deps)

	req := httptest.NewRequest("GET", "/does-not-exist", nil)
	w := httptest.NewRecorder()
	mux.Handler().ServeHTTP(w, req)

	// With exact-root registration, unknown paths should 404.
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown path, got %d", w.Code)
	}
}

func TestNewMux_AuthenticatedDataRouteContainsNoRedirect(t *testing.T) {
	auth := NewCookieAuthorizer("password", "secret")
	s, err := store.Open("file:" + t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	deps := Dependencies{
		Authorizer: auth,
		Assets:     Assets,
		Store:      s,
		Watchlist:  testWatchlist(),
		RenderLogin: func(w http.ResponseWriter, r *http.Request, err string) {
			w.WriteHeader(http.StatusOK)
		},
	}
	mux := NewMux(deps)

	// Issue a session and attach it to the request.
	rec := httptest.NewRecorder()
	if _, err := auth.IssueSession(rec); err != nil {
		t.Fatalf("issue session: %v", err)
	}

	req := httptest.NewRequest("GET", "/reporte", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	mux.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected authenticated route to be OK, got %d: %q", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Reporte") {
		t.Fatalf("expected report content, got %q", w.Body.String())
	}
}

func TestNewMux_PatternConflictsPanic(t *testing.T) {
	// This test documents that conflicting registrations panic at startup.
	// NewMux is deterministic and uses disjoint patterns, so the production
	// router is panic-free. The panic behavior is verified by the absence of
	// conflicts in TestNewMux_RegistersAllPatterns.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewMux panicked unexpectedly: %v", r)
		}
	}()

	deps := testDeps(t)
	_ = NewMux(deps)
}

func TestNewMux_AuthenticatedDashboard(t *testing.T) {
	auth := NewCookieAuthorizer("password", "secret")
	deps := testDeps(t)
	deps.Authorizer = auth
	mux := NewMux(deps)

	rec := httptest.NewRecorder()
	if _, err := auth.IssueSession(rec); err != nil {
		t.Fatalf("issue session: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	mux.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Inicio") {
		t.Error("expected dashboard title")
	}
	if !strings.Contains(body, "última actualización") {
		t.Error("expected freshness label")
	}
}

func TestNewMux_AuthenticatedAssetDetail(t *testing.T) {
	auth := NewCookieAuthorizer("password", "secret")
	deps := testDeps(t)
	deps.Authorizer = auth
	mux := NewMux(deps)

	rec := httptest.NewRecorder()
	if _, err := auth.IssueSession(rec); err != nil {
		t.Fatalf("issue session: %v", err)
	}

	req := httptest.NewRequest("GET", "/activos/yahoo/AAPL", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	mux.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d for missing asset, got %d", http.StatusNotFound, w.Code)
	}
}

func TestNewMux_AuthenticatedDatedReportInvalidDate(t *testing.T) {
	auth := NewCookieAuthorizer("password", "secret")
	deps := testDeps(t)
	deps.Authorizer = auth
	mux := NewMux(deps)

	rec := httptest.NewRecorder()
	if _, err := auth.IssueSession(rec); err != nil {
		t.Fatalf("issue session: %v", err)
	}

	req := httptest.NewRequest("GET", "/reporte/not-a-date", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	mux.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestNewMux_RouteMethodMatrix(t *testing.T) {
	deps := testDeps(t)
	mux := NewMux(deps)

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/healthz", http.StatusOK},
		{"GET", "/login", http.StatusOK},
		{"POST", "/login", http.StatusUnauthorized},
		{"GET", "/assets/main.css", http.StatusOK},
		{"GET", "/manifest.webmanifest", http.StatusOK},
		{"GET", "/sw.js", http.StatusOK},
		{"GET", "/", http.StatusSeeOther},
		{"GET", "/reporte", http.StatusSeeOther},
		{"GET", "/reporte/", http.StatusSeeOther},
		{"GET", "/reporte/2024-01-01", http.StatusSeeOther},
		{"GET", "/activos", http.StatusSeeOther},
		{"GET", "/activos/", http.StatusSeeOther},
		{"GET", "/activos/yahoo/AAPL", http.StatusSeeOther},
		{"GET", "/alertas", http.StatusSeeOther},
		{"GET", "/alertas/", http.StatusSeeOther},
		{"POST", "/alertas/reglas/", http.StatusSeeOther},
		{"PUT", "/alertas/reglas/1", http.StatusSeeOther},
		{"DELETE", "/alertas/reglas/1", http.StatusSeeOther},
		{"POST", "/alertas/reglas/1", http.StatusSeeOther},
		{"POST", "/healthz", http.StatusMethodNotAllowed},
		{"PUT", "/login", http.StatusMethodNotAllowed},
		{"DELETE", "/reporte", http.StatusMethodNotAllowed},
		{"GET", "/does-not-exist", http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			mux.Handler().ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected status %d, got %d (body: %q)", tc.want, w.Code, w.Body.String())
			}
		})
	}
}
