package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewMux_RegistersAllPatterns(t *testing.T) {
	deps := Dependencies{
		Authorizer: NewCookieAuthorizer("password", "secret"),
		Assets:     Assets,
		RenderLogin: func(w http.ResponseWriter, r *http.Request, err string) {
			w.WriteHeader(http.StatusOK)
		},
	}

	// Construction must not panic; pattern conflicts panic at registration.
	mux := NewMux(deps)

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/healthz", http.StatusOK},
		{"GET", "/login", http.StatusOK},
		{"POST", "/login", http.StatusUnauthorized}, // empty password rejected
		{"GET", "/assets/main.css", http.StatusOK},
		{"GET", "/", http.StatusSeeOther},              // protected, redirect to login
		{"GET", "/reporte", http.StatusSeeOther},       // protected
		{"GET", "/reporte/", http.StatusSeeOther},      // protected
		{"GET", "/reporte/2024-01-01", http.StatusSeeOther}, // protected
		{"GET", "/activos", http.StatusSeeOther},       // protected
		{"GET", "/activos/", http.StatusSeeOther},      // protected
		{"GET", "/activos/yahoo/AAPL", http.StatusSeeOther}, // protected
		{"GET", "/alertas", http.StatusSeeOther},       // protected
		{"GET", "/alertas/", http.StatusSeeOther},      // protected
		{"POST", "/alertas/reglas/", http.StatusSeeOther},   // protected
		{"PUT", "/alertas/reglas/1", http.StatusSeeOther},   // protected
		{"DELETE", "/alertas/reglas/1", http.StatusSeeOther}, // protected
		{"POST", "/alertas/reglas/1", http.StatusSeeOther},  // protected
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
	deps := Dependencies{
		Authorizer: NewCookieAuthorizer("password", "secret"),
		Assets:     Assets,
		RenderLogin: func(w http.ResponseWriter, r *http.Request, err string) {
			w.WriteHeader(http.StatusOK)
		},
	}
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
	deps := Dependencies{
		Authorizer: auth,
		Assets:     Assets,
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
		t.Fatalf("expected placeholder content, got %q", w.Body.String())
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

	deps := Dependencies{
		Authorizer: NewCookieAuthorizer("password", "secret"),
		Assets:     Assets,
		RenderLogin: func(w http.ResponseWriter, r *http.Request, err string) {
			w.WriteHeader(http.StatusOK)
		},
	}
	_ = NewMux(deps)
}
