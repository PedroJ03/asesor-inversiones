package web

import (
	"fmt"
	"net/http"
)

// Mux holds the configured HTTP router and its dependencies.
type Mux struct {
	handler http.Handler
}

// Handler returns the root handler.
func (m *Mux) Handler() http.Handler { return m.handler }

// NewMux builds the application router. It panics on pattern conflicts so that
// route-table tests catch registration mistakes at startup.
func NewMux(deps Dependencies) *Mux {
	mux := http.NewServeMux()

	auth := deps.Authorizer
	if auth == nil {
		auth = NewCookieAuthorizer("", "")
	}
	cookieAuth, ok := auth.(*CookieAuthorizer)
	if !ok {
		cookieAuth = NewCookieAuthorizer("", "")
	}

	// Public routes.
	mux.Handle("GET /healthz", http.HandlerFunc(healthzHandler))
	mux.Handle("GET /login", &LoginHandler{Authorizer: cookieAuth, Render: deps.RenderLogin})
	mux.Handle("POST /login", &LoginHandler{Authorizer: cookieAuth, Render: deps.RenderLogin})

	// Static assets and PWA files bypass auth.
	mux.Handle("GET /assets/{file...}", deps.Assets)
	mux.Handle("GET /manifest.webmanifest", deps.Assets)
	mux.Handle("GET /sw.js", deps.Assets)

	// Dashboard: exact root to avoid the catch-all subtree behavior of "/".
	mux.Handle("GET /{$}", auth.Middleware(http.HandlerFunc(dashboardHandler)))

	// Report: slash-less canonical plus exact slash-ful variant.
	mux.Handle("GET /reporte", auth.Middleware(http.HandlerFunc(reportCurrentHandler)))
	mux.Handle("GET /reporte/{$}", auth.Middleware(http.HandlerFunc(reportCurrentHandler)))
	mux.Handle("GET /reporte/{date}", auth.Middleware(http.HandlerFunc(reportDatedHandler)))

	// Watchlist: slash-less canonical plus exact slash-ful variant.
	mux.Handle("GET /activos", auth.Middleware(http.HandlerFunc(watchlistHandler)))
	mux.Handle("GET /activos/{$}", auth.Middleware(http.HandlerFunc(watchlistHandler)))
	mux.Handle("GET /activos/{source}/{symbol}", auth.Middleware(http.HandlerFunc(assetDetailHandler)))

	// Alerts: slash-less canonical plus exact slash-ful variant.
	mux.Handle("GET /alertas", auth.Middleware(http.HandlerFunc(alertsHandler)))
	mux.Handle("GET /alertas/{$}", auth.Middleware(http.HandlerFunc(alertsHandler)))
	mux.Handle("POST /alertas/reglas/{$}", auth.Middleware(http.HandlerFunc(ruleCreateHandler)))
	mux.Handle("PUT /alertas/reglas/{id}", auth.Middleware(http.HandlerFunc(ruleUpdateHandler)))
	mux.Handle("DELETE /alertas/reglas/{id}", auth.Middleware(http.HandlerFunc(ruleDeleteHandler)))
	mux.Handle("POST /alertas/reglas/{id}", auth.Middleware(http.HandlerFunc(ruleActionFallbackHandler)))

	return &Mux{handler: mux}
}

// Dependencies groups the cross-cutting services the router needs.
type Dependencies struct {
	Authorizer Authorizer
	Assets     http.Handler
	// RenderLogin renders the login page; it is part of the auth seam so the
	// authorizer never depends on concrete templates.
	RenderLogin func(w http.ResponseWriter, r *http.Request, err string)
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintln(w, "ok")
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	renderPlaceholder(w, r, "inicio", "Inicio")
}

func reportCurrentHandler(w http.ResponseWriter, r *http.Request) {
	renderPlaceholder(w, r, "reporte", "Reporte")
}

func reportDatedHandler(w http.ResponseWriter, r *http.Request) {
	renderPlaceholder(w, r, "reporte", "Reporte histórico")
}

func watchlistHandler(w http.ResponseWriter, r *http.Request) {
	renderPlaceholder(w, r, "activos", "Activos")
}

func assetDetailHandler(w http.ResponseWriter, r *http.Request) {
	renderPlaceholder(w, r, "activos", "Detalle de activo")
}

func alertsHandler(w http.ResponseWriter, r *http.Request) {
	renderPlaceholder(w, r, "alertas", "Alertas")
}

func ruleCreateHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func ruleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func ruleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func ruleActionFallbackHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func renderPlaceholder(w http.ResponseWriter, r *http.Request, current, title string) {
	body := PlaceholderPage(title)
	if RequestIsHX(r) {
		_ = Fragment(body).Render(r.Context(), w)
		return
	}
	_ = Shell(title, current, body).Render(r.Context(), w)
}
