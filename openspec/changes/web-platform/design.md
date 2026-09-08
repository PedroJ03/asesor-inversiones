# Design: Web Platform

> Revision 2: correction of mobile shell, no-JS actions, routing, date errors, and engine sequencing.

## Technical Approach

Build a Go 1.26 server-rendered application in `cmd/web` and `internal/web`, using templ `v0.3.1020`, Go 1.22+ `http.ServeMux`, embedded assets, and htmx `htmx.org@2.0.10` with SRI. HTML is default; htmx receives fragments. SQLite reads use `internal/store/web_reads.go`; alert-engine is available in mainline.

## Architecture Decisions

| Decision | Choice | Rationale / tradeoff |
|---|---|---|
| Rendering | templ components and embedded assets | Type-safe pages/fragments; exact v0.x pin limits drift. |
| Enhancement | htmx, no frontend framework | HTML-over-the-wire CRUD/search, 16,588 B gzipped, no client state layer. |
| Auth | `Authorizer` middleware with shared credential and HMAC-signed cookie | Replaceable identity seam; production requires `WEB_AUTH_PASSWORD` and `WEB_SESSION_SECRET`. |
| Formatting | Duplicate small price/percent helpers in `internal/web` | Protected `internal/render` helpers are unexported; avoid refactoring it. |
| Freshness | 24-hour threshold | Matches alert-engine safety; stale data is never styled current. |
| Hosting | Caddy/free-tier VPS; tunnel fallback | HTTPS/reverse proxy; Tailscale Funnel or Cloudflare Tunnel supports local use. |

## Routing and Data Flow

Non-conflicting method-qualified patterns (`GET` also matches `HEAD`):

| Pattern | Handler |
|---|---|
| `GET /` | dashboard |
| `GET /reporte`, `GET /reporte/{$}` | current report; slash-less URL is canonical |
| `GET /reporte/{date}` | dated report; invalid format → 400; valid but unavailable → controlled 404 in the platform shell |
| `GET /activos`, `GET /activos/{$}` | watchlist; slash-less URL is canonical |
| `GET /activos/{source}/{symbol}` | asset detail |
| `GET /alertas`, `GET /alertas/{$}` | rules/history; slash-less URL is canonical |
| `POST /alertas/reglas/{$}` | create rule |
| `PUT /alertas/reglas/{id}` | disable/update state |
| `DELETE /alertas/reglas/{id}` | remove rule |
| `POST /alertas/reglas/{id}` | no-JS fallback with explicit `action=deshabilitar` or `action=eliminar` |
| `GET /healthz` | unauthenticated liveness |
| `GET /login`, `POST /login` | credential exchange |
| `GET /assets/{file...}`, `GET /manifest.webmanifest`, `GET /sw.js` | embedded assets |

Both collection spellings are registered because ServeMux redirects slash-less requests when only a slashful pattern exists. They serve directly; links and `Nav` use slash-less canonical URLs. No subtree patterns are registered. Data routes use `Authorizer`; health/assets bypass it. Native rule forms POST with PRG; htmx enhances them to PUT/DELETE fragments. Method qualification prevents conflicts.

Data flow is `handler → read DTOs → Store → templ`; handlers call only `CreateRule`, `ListRules`, `RemoveRule`, `UpdateRuleState`, `LatestQuote`, and `History`, never SQL.

### Mobile shell layout

At ≤720px the shell is single-column, evolving the existing report container pattern. `Nav` becomes a thumb-reachable bottom tab bar with 3–4 tabs: Inicio, Reporte, Activos, Alertas; it may become a top bar at a desktop breakpoint. Tables become stacked cards below approximately 480px. CSS supplies this without blocking JavaScript.

## Interfaces / Contracts

`internal/store/web_reads.go` adds only:

```go
type WebQuoteKey struct{ Source, Symbol string }
const WebHistoryDefaultLimit = 100
const WebHistoryMaxLimit = 500
func (s *Store) LatestSnapshots(pairs []WebQuoteKey) (map[WebQuoteKey]QuoteRecord, error)
func (s *Store) QuoteHistory(source, symbol string, from, to time.Time, limit int) ([]QuoteRecord, error)
```

The map distinguishes missing pairs from zero values; empty batches perform no query. History rejects invalid limits/ranges and orders by `fetched_at ASC, id ASC`.

Components are `Shell`, `Nav`, `Freshness`, `Dashboard`, `Report`, `Watchlist`, `AssetDetail`, `Alerts`, and rule fragments. `HX-Request: true` selects a fragment; otherwise the shell renders. Freshness states are `current`, `stale`, `missing`, and `unavailable`, with `última actualización`. CSS keeps `asesor helper` properties without touching `internal/render/template.html`.

PWA assets include the manifest, 192/512px maskable icons, and service worker: cache-first versioned shell assets, stale-while-revalidate documents, and an offline marker that never alters timestamps/classes.

## File Changes

| File | Action | Description |
|---|---|---|
| `cmd/web/main.go` | Create | Composition, mux, server, shutdown. |
| `internal/web/*` | Create | Auth, handlers, templ, CSS, PWA assets, and tests. |
| `internal/store/web_reads.go` | Create | Batched latest and bounded history. |
| `go.mod`, `go.sum` | Modify | templ dependency metadata. |
| `internal/store/store.go`, `internal/render/*`, `cmd/report/*` | No change | Protected files. |

## Testing Strategy

Unit-test auth, freshness, formatting, and validation; SQLite-test newest-per-pair, missing pairs, deterministic history, and bounds. Handler tests cover routes, auth isolation, 400/404 dates, fragments, no-JS POST actions, and rule errors. Browser/Lighthouse checks JavaScript-disabled HTML, HTTPS installability, SRI, and <200 KB first load.

## Threat Matrix

Routing is applicable and requires mux-registration/route-table tests. Supplied rows are N/A: documentation-like paths, Git selection, commit state, push state, and PR commands (no VCS/PR automation). No RED tests apply.

## Migration / Rollout

No data migration. Alert-engine methods/schema are available in mainline; consume them directly, without deferring for a merge or inventing seeds. Any uncommitted alert-engine OpenSpec artifact should be committed for hygiene, not as a code gate. Deploy behind Caddy with tunnel fallback; roll back by stopping the binary.

## Open Questions

None; locked decisions are recorded above.
