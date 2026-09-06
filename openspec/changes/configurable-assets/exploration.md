## Exploration: configurable-assets

### Current State

The report pipeline is provider-centric:

- `watchlist.yaml` splits assets into four provider-specific sections (`usa`, `dolares`, `bonos`, `cripto`).
- `internal/config/config.go` validates the four sections separately and exposes helpers like `LabelForSymbol`.
- Each `internal/fetch` provider receives the whole `Watchlist` and extracts its own slice.
- `cmd/report/main.go` instantiates the four providers explicitly and aggregates their quotes.
- `internal/render/render.go` and `template.html` hard-code four HTML sections (`USA`, `Dólares`, `Bonos soberanos`, `Criptomonedas`).
- The store persists raw payloads and normalized quotes, but it has no concept of a user/asset preference table.

Today the advisor can only edit `watchlist.yaml`; there is no runtime validation that a new symbol exists, no search/discovery, no per-client differentiation, and the template cannot accommodate new categories or variable-length lists cleanly.

### Affected Areas

- `watchlist.yaml` — schema must move from provider sections to a provider-tagged asset list (or keep both for backward compatibility).
- `internal/config/config.go` — `Watchlist` type, validation rules, and provider lookup need redesign.
- `cmd/report/main.go` — provider construction must be driven by the configured assets instead of a fixed list.
- `internal/fetch/{yahoo,dolarapi,data912,coingecko}.go` — constructors should accept a filtered asset slice; per-asset failure handling is needed.
- `internal/render/render.go` + `internal/render/template.html` — hard-coded sections must become a dynamic list of sections/columns.
- `internal/store/store.go` — optional `assets` or `report_config` table if preferences move from YAML to SQLite.
- `internal/config/config_test.go`, `internal/render/render_test.go` — fixtures and expectations will change with the new schema.

### Approaches

| Approach | Description | Pros | Cons | Effort |
|----------|-------------|------|------|--------|
| **A. Provider-tagged YAML watchlist** | Replace provider sections with `assets: [{symbol, provider, label}]`; keep YAML as the single source of truth. Add per-asset warnings and a dynamic template. | Low complexity; preserves current prototype; keeps provider swappability; easy to add/remove assets. | Still manual YAML editing; no client-specific lists; no symbol search. | Low |
| **B. SQLite preference store + TUI/web editor** | Store assets per user/client in SQLite; build a small CLI/TUI or web page to add/remove/search symbols. | Matches the user's client-facing intent; enables future self-service. | Requires UI, auth, and a client model; far beyond prototype scope. | High |
| **C. WhatsApp-driven configuration** | Parse WhatsApp messages (`agregar AAPL`, `quitar NVDA`) and update a client watchlist. | Natural delivery channel for Argentine advisors. | Needs WhatsApp Business API/Twilio integration, parsing, and validation; not a prototype feature. | High |
| **D. CLI override flags** | Add flags like `-usa AAPL,MSFT` and `-cripto bitcoin` for one-off reports without editing YAML. | Quick ad-hoc reports; useful for testing obscure symbols. | Not persistent; does not solve long-term configuration. | Low |

### Recommendation

Start with **Approach A (provider-tagged YAML watchlist)** as the immediate evolution, and combine it with a small slice of **Approach D (CLI `-validate`/dry-run flag)** so the advisor can test obscure symbols before committing them.

Rationale:

- The prototype is still manually run and has no web/TUI; a YAML-first change is the smallest step that unlocks the user's main request ("more assets, configurable at will").
- It keeps `fetch.Provider` swappable and avoids leaking UI/auth complexity into the current codebase.
- It makes per-asset failure handling straightforward: if a symbol does not resolve, that asset becomes a warning but the rest of the report still renders.
- Per-client configuration (Approach B/C) is a logical next phase once hosting and WhatsApp delivery are real.

### Risks

1. **Yahoo rate limits** — adding many obscure USA symbols means one HTTP call per symbol; sequential fetching can become slow or trigger blocks. Mitigation: add bounded concurrency or swap to Tiingo.
2. **Symbol validation is fuzzy** — Yahoo/CoinGecko may accept a similar symbol or return an empty/error response. Mitigation: validate by attempting a fetch and surfacing per-asset warnings, not by rejecting the whole report.
3. **Provider tag errors** — users may tag an asset with the wrong provider (e.g., a bond as `yahoo`). Mitigation: strict provider whitelist at YAML load time and clear error messages.
4. **Template brittleness** — moving from four hard-coded sections to dynamic sections may break existing golden/render tests. Mitigation: redesign `ReportData` as a slice of sections and update tests together.
5. **Schema migration fatigue** — if per-client configs arrive later, a second schema change will be needed. Mitigation: keep the new asset shape provider-agnostic so it can later be persisted to SQLite without changing fetch/render interfaces.

### Ready for Proposal

Yes. The next phase should be `sdd-propose` with a concrete plan for the provider-tagged YAML watchlist, per-asset failure handling, dynamic rendering, and a dry-run validation flag.
