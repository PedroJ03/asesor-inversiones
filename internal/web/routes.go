package web

import (
	"fmt"
	"net/http"

	"github.com/PedroJ03/asesor-inversiones/internal/config"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
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

	views := newViewServer(deps.Store, deps.Watchlist)
	rules := newRuleServer(deps.Store, deps.Watchlist)

	// Dashboard: exact root to avoid the catch-all subtree behavior of "/".
	mux.Handle("GET /{$}", auth.Middleware(http.HandlerFunc(views.dashboardHandler)))

	// Report: slash-less canonical plus exact slash-ful variant.
	mux.Handle("GET /reporte", auth.Middleware(http.HandlerFunc(views.reportCurrentHandler)))
	mux.Handle("GET /reporte/{$}", auth.Middleware(http.HandlerFunc(views.reportCurrentHandler)))
	mux.Handle("GET /reporte/{date}", auth.Middleware(http.HandlerFunc(views.reportDatedHandler)))

	// Watchlist: slash-less canonical plus exact slash-ful variant.
	mux.Handle("GET /activos", auth.Middleware(http.HandlerFunc(views.watchlistHandler)))
	mux.Handle("GET /activos/{$}", auth.Middleware(http.HandlerFunc(views.watchlistHandler)))
	mux.Handle("GET /activos/{source}/{symbol}", auth.Middleware(http.HandlerFunc(views.assetDetailHandler)))

	// Alerts: slash-less canonical plus exact slash-ful variant.
	mux.Handle("GET /alertas", auth.Middleware(http.HandlerFunc(rules.alertsHandler)))
	mux.Handle("GET /alertas/{$}", auth.Middleware(http.HandlerFunc(rules.alertsHandler)))
	mux.Handle("POST /alertas/reglas/{$}", auth.Middleware(http.HandlerFunc(rules.ruleCreateHandler)))
	mux.Handle("PUT /alertas/reglas/{id}", auth.Middleware(http.HandlerFunc(rules.ruleUpdateHandler)))
	mux.Handle("DELETE /alertas/reglas/{id}", auth.Middleware(http.HandlerFunc(rules.ruleDeleteHandler)))
	mux.Handle("POST /alertas/reglas/{id}", auth.Middleware(http.HandlerFunc(rules.ruleActionFallbackHandler)))

	return &Mux{handler: mux}
}

// Dependencies groups the cross-cutting services the router needs.
type Dependencies struct {
	Authorizer Authorizer
	Assets     http.Handler
	Store      *store.Store
	Watchlist  *config.Watchlist
	// RenderLogin renders the login page; it is part of the auth seam so the
	// authorizer never depends on concrete templates.
	RenderLogin func(w http.ResponseWriter, r *http.Request, err string)
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintln(w, "ok")
}


