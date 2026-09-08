# Tasks: Web Platform

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~2,600–3,400 authored, excl. generated/vendored |
| Team budget (800 lines) | Exceeded: High risk |
| Chained PRs recommended | Yes, 6 slices |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending (user decision) |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Reads | PR 1 | `go test ./internal/store/ -run Web` | temp SQLite | revert `web_reads.go` + test |
| 2 | Shell/auth | PR 2 | `go test ./internal/web/` | `go run ./cmd/web` + curl | revert `cmd/web/` + shell files |
| 3 | Views/freshness | PR 3 | `go test ./internal/web/ -run Page` | browse after `go run ./cmd/report` | revert view files |
| 4 | Rules | PR 4 | `go test ./internal/web/ -run Rule` | server; rule CRUD | revert alerts files |
| 5 | PWA | PR 5 | `go test ./internal/web/ -run PWA` | Caddy HTTPS + Lighthouse/offline | revert PWA assets |
| 6 | Docs | PR 6 | `go build ./...` | N/A (docs) | revert docs |

## Phase 1: Read Layer

- [x] 1.1 Write `internal/store/web_reads_test.go`: newest-per-pair, missing isolated, empty batch no query.
- [x] 1.2 Create `internal/store/web_reads.go`: `WebQuoteKey`, limits 100/500, `LatestSnapshots` newest row per pair.
- [x] 1.3 Add `QuoteHistory`: reject bad ranges/limits; order `fetched_at ASC, id ASC`; range/bounds/determinism tests.

## Phase 2: Shell, Routing, Auth

- [ ] 2.1 Pin templ v0.3.1020 in `go.mod`/`go.sum`; vendor htmx 2.0.10.
- [ ] 2.2 Create `internal/web/auth.go`: `Authorizer` middleware, HMAC cookie, `WEB_AUTH_PASSWORD`/`WEB_SESSION_SECRET`, login; unauth: no data.
- [ ] 2.3 Create `cmd/web/main.go`: composition, mux, shutdown, unauth `GET /healthz`.
- [ ] 2.4 Create `internal/web/routes.go`: method-qualified dual registration (slash-less canonical) per design; route-table test asserts patterns (routing threat).
- [ ] 2.5 Create `internal/web/shell_templ.go` + `internal/web/assets/`: `Shell`, `Nav`, embedded CSS, `HX-Request` fragments; commit `_templ.go`.

## Phase 3: Reading Views

- [ ] 3.1 Create `internal/web/format.go`: duplicate price/percent helpers + tests; never edit internal/render.
- [ ] 3.2 Create `internal/web/freshness.go`: 24h threshold, current/stale/missing/unavailable states + tests.
- [ ] 3.3 Dashboard `GET /` and `/reporte[/{date}]`: invalid date 400, unavailable 404, alert badge; tests.
- [ ] 3.4 Watchlist `/activos` via `LatestSnapshots`; asset detail via `LatestQuote` + `QuoteHistory`; fragment/missing tests.
- [ ] 3.5 Mobile CSS: ≤720px single column, bottom tabs (Inicio/Reporte/Activos/Alertas), cards <480px; `última actualización` prominent.

## Phase 4: Rule Management

- [ ] 4.1 Verify frozen contract; consume directly, no substitutes/seeds.
- [ ] 4.2 Alerts `/alertas`: `ListRules(false)` + bounded `History`; armed/triggered, pct baseline; tests.
- [ ] 4.3 Create rule `POST /alertas/reglas/{$}`, PRG; `ErrInvalidRule`/`ErrDuplicateRule` → safe HTML errors; tests.
- [ ] 4.4 Rule actions: `PUT`/`DELETE /alertas/reglas/{id}`, no-JS `POST` fallback (`action=deshabilitar|eliminar`), PRG; htmx fragments; test verbs, history preserved.

## Phase 5: PWA

- [ ] 5.1 Add `manifest.webmanifest` + 192/512 maskable icons, embedded.
- [ ] 5.2 Add `sw.js`: cache-first versioned shell assets, stale-while-revalidate documents, offline marker preserving timestamps/freshness.
- [ ] 5.3 htmx SRI/defer; register service worker.
- [ ] 5.4 Verify <200 KB first load, HTTPS installability (Lighthouse), JS-disabled HTML.

## Phase 6: Docs/Final Gate

- [ ] 6.1 Document run/deploy: env vars, Caddy proxy, tunnel fallback; rollback: stop binary.
- [ ] 6.2 Final gate: `go test ./...`, `go build ./...`, `go vet ./...`; zero protected store/render/report edits.
