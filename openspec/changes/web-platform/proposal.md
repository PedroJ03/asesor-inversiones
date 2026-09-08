# Proposal: Web Platform

## Intent

Turn the CLI report into an internet-facing, advisor-only, installable platform exposing data, alert history, and freshness without changing report generation.

## Scope

### In Scope
- Go server-rendered pages with progressive enhancement: dashboard, daily report, watchlist, asset detail, alerts, archived report, and health.
- Alert-rule create, disable, and remove flows through the alert-engine contract.
- Mobile PWA shell: ≤720px single column, bottom navigation, manifest, icons, service worker, HTTPS, <200 KB first load, and near-zero blocking JavaScript.
- `templ` at an exact v0.x pin; htmx `htmx.org@2.0.10` with SRI for CRUD and symbol selection.

### Out of Scope
- Real-time/scheduled fetching, charts, multi-client authorization, push/WhatsApp channels, dark mode, and SPA/mobile clients.
- Changes to `internal/render`, `internal/store/store.go`, `cmd/report`, or providers.

## Capabilities

### New Capabilities
- `web-pages`: Auth-seamed views for `GET /`, `/reporte`, `/reporte/{date}`, `/activos`, `/activos/{source}/{symbol}`, `/alertas`, and `/healthz`.
- `web-read-layer`: Batched latest snapshots and history reads reusable by JSON twins.
- `alert-rule-management`: Advisor rule management backed by alert-engine methods.
- `web-pwa`: Installable shell with freshness/staleness presentation.

### Modified Capabilities
- None.

## Approach

Add only `cmd/web/`, `internal/web/`, and `internal/store/web_reads.go`. Use `ServeMux`, embedded templ/assets, an identity-agnostic auth seam, and htmx HTML-over-the-wire handlers. Show `última actualización` and staleness because only `cmd/report` refreshes quotes. Keep read DTOs API-ready; design must decide whether to extract shared formatting helpers from `internal/render` without modifying it.

Consume frozen methods: `CreateRule`, `ListRules`, `RemoveRule`, `LatestQuote`, `UpdateRuleState`, `InsertAlert`, and `History`. Use seed data until the engine merges. Alert-rule tasks MUST apply afterward; its untracked OpenSpec artifacts must be committed before platform apply. Merge engine first, then rebase platform.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `cmd/web/` | New | Server entry point. |
| `internal/web/` | New | Handlers, views, assets, auth. |
| `internal/store/web_reads.go` | New | Latest/history reads. |
| `store.go`, `internal/render/`, `cmd/report/` | No change | Protected files. |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Engine dependency or collision | High | New files, frozen contract, seed data, engine-first merge. |
| Stale values appear current | High | Timestamp and staleness styling. |

## Rollback Plan

Stop the web binary and revert platform files; SQLite alert data and CLI reporting remain intact.

## Dependencies

- Merged alert-engine contract; templ pin; htmx 2.0.10 plus SRI; HTTPS-capable hosting or local fallback.

## Success Criteria

- [ ] Routes and auth seam render SQLite data with visible freshness state.
- [ ] Rule CRUD compiles and uses the frozen contract after engine merge.
- [ ] PWA assets, <200 KB first load, and near-zero blocking JS pass verification.
- [ ] `go test ./...`, `go build ./...`, and `go vet ./...` pass without protected-file edits.
