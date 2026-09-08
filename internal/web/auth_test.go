package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCookieAuthorizer_Authentication(t *testing.T) {
	auth := NewCookieAuthorizer("secret-password", "hmac-secret")

	t.Run("valid cookie authenticates", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := auth.IssueSession(w); err != nil {
			t.Fatalf("issue session: %v", err)
		}

		req := httptest.NewRequest("GET", "/", nil)
		for _, c := range w.Result().Cookies() {
			req.AddCookie(c)
		}

		if !auth.IsAuthenticated(req) {
			t.Fatal("expected request with valid cookie to be authenticated")
		}
	})

	t.Run("missing cookie is unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		if auth.IsAuthenticated(req) {
			t.Fatal("expected missing cookie to be unauthenticated")
		}
	})

	t.Run("tampered cookie is rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		if _, err := auth.IssueSession(w); err != nil {
			t.Fatalf("issue session: %v", err)
		}

		req := httptest.NewRequest("GET", "/", nil)
		for _, c := range w.Result().Cookies() {
			// Corrupt the signature portion.
			if c.Name == sessionCookieName {
				c.Value = c.Value + "x"
			}
			req.AddCookie(c)
		}

		if auth.IsAuthenticated(req) {
			t.Fatal("expected tampered cookie to be rejected")
		}
	})

	t.Run("wrong password is rejected", func(t *testing.T) {
		if auth.CheckPassword("wrong") {
			t.Fatal("expected wrong password to be rejected")
		}
	})

	t.Run("correct password is accepted", func(t *testing.T) {
		if !auth.CheckPassword("secret-password") {
			t.Fatal("expected correct password to be accepted")
		}
	})
}

func TestCookieAuthorizer_CookieFlags(t *testing.T) {
	auth := NewCookieAuthorizer("password", "secret")
	w := httptest.NewRecorder()
	if _, err := auth.IssueSession(w); err != nil {
		t.Fatalf("issue session: %v", err)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]

	if !c.HttpOnly {
		t.Error("expected HttpOnly")
	}
	if !c.Secure {
		t.Error("expected Secure")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", c.SameSite)
	}
	if !strings.Contains(c.Value, ".") {
		t.Error("expected signed cookie value with '.' separator")
	}
}

func TestLoginHandler(t *testing.T) {
	auth := NewCookieAuthorizer("password", "secret")
	render := func(w http.ResponseWriter, r *http.Request, err string) {
		w.Header().Set("Content-Type", "text/html")
		if err != "" {
			_, _ = w.Write([]byte("error:" + err))
			return
		}
		_, _ = w.Write([]byte("login page"))
	}
	h := &LoginHandler{Authorizer: auth, Render: render}

	t.Run("GET returns login page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/login", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		if !strings.Contains(w.Body.String(), "login page") {
			t.Fatalf("expected login page body, got %q", w.Body.String())
		}
	})

	t.Run("POST with correct password sets cookie and redirects", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/login", strings.NewReader("password=password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("expected status %d, got %d", http.StatusSeeOther, w.Code)
		}
		if loc := w.Header().Get("Location"); loc != "/" {
			t.Fatalf("expected redirect to /, got %q", loc)
		}
		cookies := w.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected session cookie after login")
		}
	})

	t.Run("POST with wrong password returns unauthorized without data", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/login", strings.NewReader("password=wrong"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "error:") {
			t.Fatalf("expected error message, got %q", body)
		}
		if strings.Contains(body, "quote") || strings.Contains(body, "rule") {
			t.Fatal("login failure must not contain quote/rule data")
		}
	})
}

func TestAuthorizerMiddleware_ProtectsDataRoutes(t *testing.T) {
	auth := NewCookieAuthorizer("password", "secret")
	protected := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("sensitive quote/rule data"))
	}))

	t.Run("unauthenticated request redirects without data", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/reporte", nil)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("expected status %d, got %d", http.StatusSeeOther, w.Code)
		}
		if loc := w.Header().Get("Location"); loc != "/login" {
			t.Fatalf("expected redirect to /login, got %q", loc)
		}
		if strings.Contains(w.Body.String(), "sensitive") {
			t.Fatal("unauthenticated response must not leak protected data")
		}
	})

	t.Run("authenticated request passes through", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if _, err := auth.IssueSession(rec); err != nil {
			t.Fatalf("issue session: %v", err)
		}

		req := httptest.NewRequest("GET", "/reporte", nil)
		for _, c := range rec.Result().Cookies() {
			req.AddCookie(c)
		}
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		if !strings.Contains(w.Body.String(), "sensitive") {
			t.Fatalf("expected protected body, got %q", w.Body.String())
		}
	})
}
