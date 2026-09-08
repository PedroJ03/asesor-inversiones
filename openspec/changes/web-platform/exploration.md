# Exploration: web-platform — Independent Mobile-First Web Frontend

**Decision summary (read this first)**

| Question | Answer |
|---|---|
| Architecture (Q1) | **B — Go server-rendered pages + progressive enhancement**, with the read layer structured so JSON API twins can be exposed later without rework |
| Location (Q2) | **Same repo**: `cmd/web/` + `internal/web/`, new files only (isolation rule for the concurrent alert-engine agent) |
| API surface (Q3) | Pages-first routes (`/`, `/reporte`, `/activos`, `/alertas`) over a small internal read layer; sketched below, designed in sdd-design |
| v1 scope (Q4) | Branded shell + daily report view + watchlist view + alerts view (read-only) + freshness display + advisor-only gate + installable-PWA basics |
| Mobile-first (Q5) | Single column ≤720px, bottom tab bar, <200 KB first load, near-zero JS; PWA basics are load-bearing for the future push channel (iOS requires an installed PWA) |
| Auth fork (Q6) | **NOT decided** — advisor-only assumed for v1 behind a middleware seam; multi-client flagged as an open product decision |
| Research (Q7) | Optional only: (1) templ/htmx state of the art, (2) Go Web Push/VAPID for the channels phase. Nothing blocks the proposal |

## Exploration: web-platform

### Current State

- Go 1.26.6, single module `github.com/PedroJ03/asesor-inversiones`. Pipeline: `cmd/report` → `internal/config` (watchlist.yaml) → `internal/fetch` (Yahoo, dolarapi, data912, CoinGecko behind `fetch.Provider`) → `internal/store` (SQLite WAL: `raw_fetch`, `quotes`) → `internal/render` (embedded mobile-first Spanish HTML template) → static files in `reporte/`. No server, no HTTP surface, no JS toolchain, no Makefile/CI.
- Branding today: CSS custom properties in `internal/render/template.html` (GitHub-like palette, system fonts, 720px container, card sections, positive/negative chips, sources footer + disclaimer). **The product has no name** — the report title is generic ("Reporte diario de mercado").
- Store read surface today: only `QuotesBySource`, which returns **every** stored row for a source (a history scan, not a latest snapshot). Quote history accumulates with every `cmd/report` run, so charts/history views will have real data from day one — no backfill needed.
- Data freshness: quotes are written **only when the CLI runs** (manual today). A platform reading this DB must surface "última actualización" prominently; the alert engine's 24h staleness guard is the precedent.
- `modernc.org/sqlite` is pure Go (no CGO) → the whole product (server + templates + CSS + DB driver) can ship as one static binary with `go:embed`.
- **Frozen alert-engine contract** (concurrent change; code does not exist yet — artifacts in `openspec/changes/alert-engine/`): tables `alert_rules(id, source, symbol, kind, direction, threshold, baseline_price, enabled, state, created_at)` and `alerts` (denormalized snapshots, `rule_id` FK `ON DELETE SET NULL`); store methods `CreateRule`, `ListRules(enabledOnly)`, `RemoveRule`, `LatestQuote(source,symbol)`, `UpdateRuleState`, `InsertAlert`, `History(limit)`. The platform is a **read-only consumer** of this contract.
- **Confirmed isolation rules** (parallel development): (1) platform works in its own git worktree + feature branch; (2) contract-first — build against the frozen schema, seed data until the engine ships; (3) alert-engine owns edits to `internal/store/store.go`; platform adds **only new files**; merge order: alert-engine first, platform rebases.

### Affected Areas

| Path | Why |
|---|---|
| `cmd/web/` (NEW) | Server entry point, mirroring `cmd/report` composition style |
| `internal/web/` (NEW) | Handlers, page templates, static assets (embedded), auth middleware seam |
| `internal/store/web_reads.go` (NEW FILE in existing package) | Batched latest-snapshot and history-range reads as methods on `*Store` — same package, different file → no edit collision with alert-engine's `store.go` changes |
| `internal/config/`, `internal/fetch/fetch.go` | Reused read-only (watchlist model, `Quote` type) |
| `internal/render/` | NOT modified (report CLI owns it). Platform needs the same price/percent formatting: extract shared helpers vs duplicate — design-phase decision |
| `openspec/changes/alert-engine/*` | Frozen contract, read-only reference |
| `watchlist.yaml`, `data/asesor.db` | Runtime inputs, unchanged |

### Approaches (Q1 — architecture)

Context factors that weight the comparison: solo developer; advisor is the first user (clients uncertain, later); mobile-first is mandatory; the UI is **display-first** (tables/cards of quotes and alerts, updated a few times per day, not minute-by-minute); existing Go+SQLite backend; deployment must stay simple; the channels phase (push) is server-side regardless of frontend choice.

1. **A — Separate SPA (React/Svelte) + Go JSON API**
   - Pros: highest interactivity ceiling; clean API boundary reusable by a future mobile app; frontend skill growth; static/CDN hosting of the shell.
   - Cons: two toolchains (Node build + Go) for one person; TS types duplicating `Quote`/rule structs; CORS + token auth complexity; two deploy artifacts; SPA state machinery is dead weight for a display UI; JS payload fights the mobile budget on Argentine networks/mid-range devices; slowest time-to-value.
   - Effort: **High**

2. **B — Go server-rendered pages + progressive enhancement** (html/template or templ; htmx/vanilla-JS sprinkles only where needed)
   - Pros: one language, **one binary** (embedded assets + SQLite = the whole product); reuses existing store/fetch/config patterns and branding tokens; Go 1.22+ stdlib `ServeMux` supports method+pattern routes (`GET /reporte/{date}`) → no router dependency; session-cookie auth is trivial; naturally inside a strict mobile perf budget; the same read structs can gain JSON twins later, so A's API boundary is **born, not retrofitted** if the product ever needs it.
   - Cons: less app-like transitions without deliberate effort; Go templating ergonomics are weaker than JSX for highly dynamic UI (v1's UI is not); weaker frontend-skill pull; a future pivot to a rich SPA means partial rework of the view layer (mitigated by the read-layer seam).
   - Effort: **Low–Medium**

3. **C — Hybrid / islands** (server-rendered shell + JS islands for charts, rule editor)
   - Pros: server-rendered speed with JS exactly where needed; incremental path between B and A.
   - Cons: needs boundary discipline; islands tend to drift into a shadow SPA; largely redundant with B+htmx for v1's actual interactivity needs.
   - Effort: **Medium**

Honest tradeoff note: **A becomes right** if the product evolves into a client-facing, highly interactive experience, or if the developer explicitly wants to invest in frontend skills — that is a legitimate non-product benefit and should be said out loud. For v1 (solo dev, display-first advisor tool), B wins on every weighted context factor and does not foreclose A.

### Where the frontend lives (Q2)

**Same repo.** The single Go module, shared types (`fetch.Quote`, store records), atomic commits, and one `go test ./...` all favor it. A separate repo buys nothing for a solo developer and adds API versioning + cross-repo sync cost. Concurrency safety comes from the isolation rules (new files only; separate worktree/branch at apply time), not from repo separation. Even under option A, the SPA would live in `web/` inside this repo with its build output embedded — a separate repo is not justified in any scenario explored.

### Read-only API surface sketch (Q3 — sketch only; sdd-design owns the real design)

Pages (Spanish slugs match the product UI):

| Route | Purpose |
|---|---|
| `GET /` | Dashboard: today's snapshot, triggered-alert badge, freshness timestamp |
| `GET /reporte` | Daily report view (platform shell, same four sections) |
| `GET /reporte/{yyyy-mm-dd}` | Archived reports (data already accumulates; v1.x) |
| `GET /activos` | Watchlist grouped by section with latest quotes |
| `GET /activos/{source}/{symbol}` | Asset detail: latest quote + history/sparkline |
| `GET /alertas` | Rules with `armed`/`triggered` state (`ListRules`) + recent snapshots (`History(limit)`) |
| `GET /healthz` | Liveness |

Optional JSON twins (cheap later because handlers and JSON would share one read layer): `/api/v1/quotes/latest`, `/api/v1/quotes/{source}/{symbol}/history`, `/api/v1/watchlist`, `/api/v1/alerts/rules`, `/api/v1/alerts/history`, `/api/v1/report/latest`.

Read-layer gaps the platform must add itself (new store file): **batched latest snapshot per `(source, symbol)`** (the contract's `LatestQuote` is per-pair; a dashboard wants one query) and **history range/limit per pair**. Everything alerts-related is covered by the frozen contract. **Writes: none in v1.** Web rule management (`CreateRule`/`RemoveRule`) is v2.

### Minimum lovable v1 scope (Q4)

| v1 (lovable) | Later |
|---|---|
| Branded shell: product name, tokens evolved from `template.html` | Asset-detail charts + history browser UI |
| Daily report view (today) served live from SQLite | Web rule management (create/disable/remove via contract methods) |
| Watchlist view (four sections, latest quotes) | Multi-client views (depends on the auth fork) |
| Alerts view — read-only: rule states + history | Web Push + WhatsApp (channels phase) |
| Freshness + warnings display ("última actualización") | Scheduled/auto fetch |
| Advisor-only auth gate (mechanism TBD in design) | JSON API twins when a real consumer exists |
| Installable-PWA basics: manifest, icons, minimal service worker | Dark mode |
| HTTPS deployment story | |

Lovable ≠ minimal: v1's differentiators over today's generated HTML file are (1) a live site instead of files on disk, (2) **alerts become visible for the first time** (the engine deliberately hides them from the report), (3) installable to the home screen.

### Mobile-first implications (Q5)

- **Layout**: single column ≤720px (existing pattern); bottom tab bar for the 3–4 views (thumb reach); tables collapse to stacked cards under ~480px; extend the existing CSS custom-property system; system fonts (already in use).
- **Performance budget** (Argentine mobile networks, mid-range devices): total first load **<200 KB**; near-zero blocking JS in v1 under option B; LCP <2.5s on mid-range Android; no framework runtime. Verify with Lighthouse during design/verify.
- **PWA potential**: manifest + service worker (app-shell cache, stale-while-revalidate for pages) make it installable with server-rendered pages just as well as with an SPA. This is **load-bearing for the roadmap**: Web Push on iOS works **only** from an installed home-screen PWA (iOS ≥16.4; requires HTTPS, manifest, service worker). Building PWA basics in v1 directly de-risks the channels phase on both Android and iPhone.

### Open product decisions (flagged, NOT decided here)

1. **Auth fork (Q6)**: advisor-only now vs multi-client later. v1 assumes advisor-only, but the design MUST keep auth behind a middleware seam and the read layer identity-agnostic so multi-client can be added without rework (multi-client introduces a new domain: clients, per-client watchlists, visibility rules — significant schema impact). User decision needed first: **is v1 internet-facing or private (LAN/tailnet)?** If private, the v1 gate can be minimal and the fork defers entirely.
2. **Branding**: the platform "carries the system's branding," but the system has no name. Needs a product name + logo direction + palette evolution from existing tokens.
3. **Hosting**: where does this run (own machine, VPS, Fly.io)? HTTPS is required for PWA/push, which constrains the answer.

### Recommendation (Q7)

**Option B — Go server-rendered platform with progressive enhancement, in this repo (`cmd/web` + `internal/web`), with a read layer designed so JSON twins can be added later.** For a solo Go developer building a display-first, mobile-first advisor tool on an existing Go+SQLite backend, B minimizes toolchains, deploy artifacts, and time-to-value while staying inside the mobile budget by construction; the read-layer seam preserves A's future (SPA/mobile/push consumers) without paying A's costs now. C adds little over B+htmx for v1.

Delivery sequencing: the alerts view is a contract-first consumer — develop it against seed rows matching the frozen schema, and land its tasks **after** alert-engine merges to `main` (the roadmap already enforces this order).

**Research lanes (sdd-research, all OPTIONAL — none blocks the proposal):**
1. Go HTML-first stack state of the art (`html/template` vs `templ`; htmx 2.x vs vanilla/Alpine) — modest value; the conservative default (stdlib templates, near-zero JS) carries zero research risk.
2. Web Push from Go (VAPID libraries, subscription storage) + iOS install UX — real value, but belongs to the **channels** change, not v1.
3. Hosting/HTTPS options for a single Go+SQLite binary — this is a user decision (Open question 3), not a research question.

### Risks

- **Concurrent-agent collision in `internal/store`**: alert-engine edits `store.go`; the platform must add its reads in a NEW file (e.g. `web_reads.go`) as methods on `*Store` — same package, different file, near-zero conflict. Merge order (engine first, platform rebases) is the backstop.
- **Frozen contract artifacts are UNTRACKED in git** (`openspec/changes/alert-engine/` shows `??`): a fresh platform worktree will not contain them. They must be committed before platform apply begins, or the "contract-first" rule is unenforceable from the worktree.
- **Compile-time dependency**: platform code calling contract store methods cannot compile until alert-engine lands → sequence alerts-view tasks last or isolate them behind a seed-backed interface.
- **Stale-data trust**: quotes refresh only when the CLI runs; a live-looking site with stale numbers erodes advisor trust → prominent freshness timestamp + staleness styling (mirror the 24h guard).
- **Branding/name unresolved**: the v1 "carries branding" requirement is undesignable until the user answers.
- **Single-user assumptions leaking**: if the auth fork stays open, handlers/schema must not hardcode one user — enforce the middleware-seam rule in design review.
- **Provider fragility** (Yahoo unofficial, CoinGecko rate limits — already flagged in README): the platform surfaces the same warnings; higher visibility, no new failure mode.

### Ready for Proposal

**Yes — conditional on three user answers**: (1) confirm architecture direction B (or override toward A/C knowingly), (2) v1 exposure: internet-facing vs private, and advisor-only assumption, (3) product name/branding direction + hosting target. Optional research lanes 1–2 do not block sdd-propose.
