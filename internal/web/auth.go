// Package web provides the HTTP shell, routing, auth, and rendering for the
// web platform.
package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	sessionCookieName = "session"
	sessionExpiry     = 7 * 24 * time.Hour
)

// Authorizer is the authentication seam. It is intentionally identity-agnostic:
// callers check only whether a request is authenticated.
type Authorizer interface {
	IsAuthenticated(r *http.Request) bool
	Middleware(next http.Handler) http.Handler
}

// CookieAuthorizer guards data routes with a shared password and an
// HMAC-signed session cookie.
type CookieAuthorizer struct {
	password []byte
	secret   []byte
}

// NewCookieAuthorizer builds an authorizer from the configured password and
// HMAC secret. Both are required; empty values disable authentication, which
// is only useful for focused tests that bypass auth explicitly.
func NewCookieAuthorizer(password, secret string) *CookieAuthorizer {
	return &CookieAuthorizer{
		password: []byte(password),
		secret:   []byte(secret),
	}
}

// IsAuthenticated reports whether the request carries a valid, unexpired
// session cookie.
func (a *CookieAuthorizer) IsAuthenticated(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	value, sig, ok := strings.Cut(c.Value, ".")
	if !ok {
		return false
	}
	if !a.validMAC(value, sig) {
		return false
	}
	expiresBytes, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return false
	}
	expires, err := time.Parse(time.RFC3339, string(expiresBytes))
	if err != nil {
		return false
	}
	if time.Now().UTC().After(expires) {
		return false
	}
	return true
}

// Middleware wraps a data handler. Authenticated requests pass through;
// unauthenticated requests are redirected to /login so the seam itself never
// leaks data.
func (a *CookieAuthorizer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.IsAuthenticated(r) {
			next.ServeHTTP(w, r)
			return
		}
		// Preserve htmx behavior: clients expect either a fragment or a full
		// redirect; for the shell we redirect so the browser loads /login.
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusSeeOther)
	})
}

// validMAC reports whether sig is a valid base64 HMAC-SHA256 of value.
func (a *CookieAuthorizer) validMAC(value, sig string) bool {
	want := a.sign(value)
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	return hmac.Equal(want, got)
}

// sign returns the HMAC-SHA256 of value using the configured secret.
func (a *CookieAuthorizer) sign(value string) []byte {
	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

// IssueSession sets a signed session cookie and returns the expiry time.
func (a *CookieAuthorizer) IssueSession(w http.ResponseWriter) (time.Time, error) {
	if len(a.secret) == 0 {
		return time.Time{}, errors.New("session secret not configured")
	}
	expires := time.Now().UTC().Add(sessionExpiry).Round(time.Second)
	value := base64.RawURLEncoding.EncodeToString([]byte(expires.Format(time.RFC3339)))
	sig := base64.RawURLEncoding.EncodeToString(a.sign(value))
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    value + "." + sig,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	return expires, nil
}

// ClearSession removes the session cookie.
func (a *CookieAuthorizer) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// CheckPassword performs a constant-time comparison against the configured
// shared credential.
func (a *CookieAuthorizer) CheckPassword(password string) bool {
	return subtle.ConstantTimeCompare(a.password, []byte(password)) == 1
}

// LoginHandler serves GET /login and handles POST /login credential exchange.
type LoginHandler struct {
	Authorizer *CookieAuthorizer
	Render     func(w http.ResponseWriter, r *http.Request, loginError string)
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.Render(w, r, "")
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (h *LoginHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	password := r.FormValue("password")
	if !h.Authorizer.CheckPassword(password) {
		w.WriteHeader(http.StatusUnauthorized)
		h.Render(w, r, "Contraseña incorrecta")
		return
	}
	if _, err := h.Authorizer.IssueSession(w); err != nil {
		http.Error(w, fmt.Sprintf("session error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}
