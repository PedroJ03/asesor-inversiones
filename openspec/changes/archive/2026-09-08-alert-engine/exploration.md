# Exploration: alert-engine

**Change title**: Motor de alertas: el asesor configura thresholds de valor o de porcentaje por activo y el sistema genera alertas.

## Locked Constraints (user-confirmed, not revisitable)

1. Alerts are NOT visible in the report. `internal/render` and the report output MUST NOT change in this phase.
2. Alerts surface later in a future web platform — this phase only produces and stores them.
3. No delivery channels (no WhatsApp/push/email). Alert output = SQLite persistence + CLI visibility.
4. Roadmap: alert engine → web platform → automatic channels.

## Current State

- Pipeline: `cmd/report` loads `watchlist.yaml` → fetches 4 providers → `store.SaveRaw` + `store.SaveQuotes` → `render`.
- Assets are identified by `(source, symbol)` in the `quotes` table: `yahoo`/`SPY`, `dolarapi`/`blue`, `data912`/`AL30D`, `coingecko`/`bitcoin`.
- Quote field availability per provider (decisive for percentage rules):

| Source | Price | PrevClose | ChangePct | Meaning |
|---|---|---|---|---|
| yahoo | yes | yes | yes | daily change |
| dolarapi | yes (venta) | no (0) | no (0) | n/a — only bid/ask |
| data912 | yes | derived | yes | daily pct_change |
| coingecko | yes | no | yes | 24h change |

- `store.QuotesBySource` exists but there is no "latest quote per asset" query.
- Runs are manual (`go run ./cmd/report`); no scheduler. Baseline verified green (`go build ./...`, `go test ./...`) on 2026-09-06.

## Area 1 — Rule Model

Two threshold kinds, each with a direction; comparisons are inclusive (`>=` / `<=`):

- **value**: alert when `price >= X` (above) or `price <= X` (below).
- **pct**: alert when `change_pct >= +X` (above) or `change_pct <= -X` (below).

### Percentage semantics fork

- **(a) Daily change** — uses the already-normalized `Quote.ChangePct` (price vs prev_close or provider-reported 24h change).
- **(b) Baseline drift** — `(price - baseline) / baseline * 100` against a baseline price captured when the rule is created.

**Recommendation for this phase: (a) daily change.**

- Matches the advisor's EOD mental model ("tell me what moved hard today") and the report's own daily-change framing.
- Zero extra state: no baseline to capture, store, or re-arm; the value already exists in every quote except dolarapi.
- Baseline drift ("tell me when BTC moves 10% from today's price") is a real future need — defer it; the schema stays extensible (a nullable `baseline_price` column can be added later without migration pain).

**Consequence**: `pct` rules are only valid for providers that report change. `dolarapi` has no ChangePct → reject `kind=pct` rules for `dolarapi` at creation time with a clear error (static validation beats evaluate-time surprises).

## Area 2 — Rule Storage

| Option | Pros | Cons |
|---|---|---|
| **SQLite (recommended)** | Same DB the engine already runs on; future web platform configures rules via UI against the same tables; mutable rule state (armed/triggered) belongs in a DB, not a config file; FK into alert history | New tables + store methods |
| YAML | Consistent with watchlist | Mixing config with mutable state; awkward programmatic updates from a future UI; no FK to history |

**Recommendation: SQLite**, in the same `data/asesor.db`, as new tables in `internal/store`. Schema sketch:

```sql
CREATE TABLE alert_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,             -- yahoo|dolarapi|data912|coingecko
  symbol TEXT NOT NULL,             -- asset symbol within that provider
  kind TEXT NOT NULL CHECK (kind IN ('value','pct')),
  direction TEXT NOT NULL CHECK (direction IN ('above','below')),
  threshold REAL NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  state TEXT NOT NULL DEFAULT 'armed' CHECK (state IN ('armed','triggered')),
  created_at DATETIME NOT NULL,
  UNIQUE (source, symbol, kind, direction, threshold)
);

CREATE TABLE alerts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  rule_id INTEGER NOT NULL REFERENCES alert_rules(id),
  source TEXT NOT NULL,
  symbol TEXT NOT NULL,
  kind TEXT NOT NULL,
  threshold REAL NOT NULL,          -- snapshot: survives rule edits/deletion
  observed_price REAL,              -- set for value rules (also for pct, context)
  observed_change_pct REAL,         -- set for pct rules
  quote_fetched_at DATETIME,
  triggered_at DATETIME NOT NULL
);
```

## Area 3 — Evaluation Source

| Option | Description | Verdict |
|---|---|---|
| A. Alert run re-fetches | Duplicates network calls (doubles Yahoo per-symbol calls, CoinGecko quota); fetch orchestration lives in `cmd/report`, reuse would require refactoring the report pipeline | Rejected |
| **B. Evaluate stored quotes** | Alert binary reads latest quotes from SQLite (written by `go run ./cmd/report`), applies rules, persists alerts. Zero network, idempotent, clean separation | **Recommended** |
| C. Evaluate inside cmd/report | Guarantees fresh data but couples concerns, modifies the report binary's behavior, and violates the spirit of constraint 1 | Rejected |

**Recommendation: B**, with a **staleness guard**:

- For each active rule, fetch the latest quote for its `(source, symbol)` (`ORDER BY fetched_at DESC LIMIT 1` — one simple query per rule; rule counts are small, no clever SQL needed).
- If no quote exists, the provider failed, or `fetched_at` is older than a max age (default **24h**, overridable via flag): do NOT evaluate that rule, do NOT change its state — emit a warning instead. Fail-safe over eager.
- Advisor workflow: `go run ./cmd/report && go run ./cmd/alerts`. A future cron calls the same two commands; scheduling infra is explicitly later.

## Area 4 — Dedupe / Re-arm Semantics

Two-state machine persisted on the rule row:

```
armed ──(condition true)──► TRIGGERED: insert alert, state := triggered
triggered ──(condition still true)──► no-op (dedupe)
triggered ──(condition false)──► state := armed (auto re-arm, no alert)
armed ──(condition false)──► no-op
```

- A crossing alerts **once per arm cycle**, not every cycle.
- **Auto re-arm** when the condition resets is the default: simplest model that avoids both spam and dead rules. Manual re-arm is a future option, not this phase.
- Missing/stale quote: no state change at all (guard from Area 3).
- "Crossing" is condition-state comparison against persisted state (we evaluate latest snapshots, not intraday series) — correct and honest at EOD granularity.
- Edge cases the tests must cover: first crossing fires exactly once; repeat evaluation while true stays silent; reset re-arms silently; re-cross fires again; both `above` and `below` rules can coexist on one asset; boundary equality fires (inclusive).
- **Deferred**: hysteresis band (re-arm only after moving X% away from threshold) — flap risk is low with manual/EOD runs.

## Area 5 — Alert History

The `alerts` table (Area 2) is the persisted history: asset, rule id, kind, threshold snapshot, observed value(s), quote freshness timestamp, trigger time. Deliberately denormalized snapshots so history stays auditable after rule edits/deletion. The future web platform reads this table directly; no delivery logic attached.

## Area 6 — CLI Shape

**Recommendation: a separate `cmd/alerts` binary** (not a flag of `cmd/report`): keeps the report binary untouched (constraint 1), and cron can invoke each concern independently.

Subcommands (Go `flag` + `os.Args` switch, no new deps):

- `run` — evaluate active rules against latest quotes; print triggered alerts and warnings; exit 0 even with warnings (fail-safe).
- `add -source yahoo -symbol NVDA -kind value -direction above -threshold 200` — validates source whitelist, watchlist membership (Area 7), and the pct/dolarapi restriction.
- `list` — rules with state (armed/triggered/enabled).
- `remove -id N` and `history` — small but optional under budget pressure (Area 9).

## Area 7 — Asset Referencing

- Rules reference assets by `(source, symbol)` — exactly the `quotes` table key, no new identifier concept.
- At `add` time, validate against the current fixed `watchlist.yaml` (source whitelist + symbol membership) so orphan rules for untracked assets are rejected early.
- **Forward compatibility with configurable-assets** (deferred, GitHub issue #1): when the watchlist becomes provider-tagged/DB-backed, `(source, symbol)` remains the stable asset identifier — only the validation source changes, the alert schema does not.

## Area 8 — Testing Strategy

All offline, no network (consistent with `internal/fetch/testdata` conventions and the go-testing skill):

- **Pure engine** (new `internal/alert` package): table-driven tests for rule matching (value/pct × above/below × boundary equality), state transitions (armed→triggered→armed→triggered dedupe cycle), staleness guard, missing-quote guard. Inputs: in-memory rules + quotes + now; outputs: triggered alerts + new states + warnings. No DB, no HTTP.
- **Store additions**: table creation, rule CRUD + unique constraint, latest-quote query, alert insert + state update — `t.TempDir()` SQLite, following the existing `newTestStore` pattern in `internal/store/store_test.go`.
- **CLI**: manual smoke (`go run ./cmd/alerts add ... && go run ./cmd/alerts run`); no E2E harness exists yet.

## Area 9 — Size Sanity (400-line review budget)

| Piece | Estimate |
|---|---|
| `internal/alert/engine.go` (pure evaluation) | ~120 lines |
| `internal/store` additions (2 tables + queries) | ~140 lines |
| `cmd/alerts/main.go` (run/add/list) | ~150 lines |
| Tests (engine + store) | ~300 lines |

Production code lands at ~400 lines; tests add ~300. **This is at the budget ceiling.** Levers if the budget counts tests too:

1. Drop `remove` and `history` subcommands (delete via SQL, history read by the future platform).
2. Split into two chained PRs: (i) store + engine + tests, (ii) CLI.

**Scope creep — explicitly OUT**: report/template changes, delivery channels, scheduler/cron infra, web UI, rule editing (only add/remove), hysteresis, multi-user/client rules, baseline-drift percentage mode.

## Open Questions for the User (confirm at exploration gate)

1. **Percentage semantics**: daily change now (recommended), baseline drift deferred — confirm.
2. **CLI shape**: separate `cmd/alerts` binary — confirm.
3. **Re-arm**: automatic when condition resets (recommended) vs manual re-arm — confirm.
4. **Staleness guard**: default max quote age 24h — confirm or adjust.

## Recommendation

Proceed to `sdd-propose` with: SQLite rule/alert storage in `internal/store`, pure evaluation engine in `internal/alert` (daily-change pct, inclusive thresholds, two-state auto re-arm dedupe), staleness-guarded evaluation over stored quotes, and a `cmd/alerts` binary (run/add/list core; remove/history as budget allows). No external research needed — every building block exists in the current codebase.

## Risks

1. **Review budget overrun** — production + tests ≈ 700 lines total; mitigate with the scope trims or chained PRs from Area 9.
2. **Silent staleness** — if the advisor forgets `cmd/report`, alert runs emit warnings and fire nothing; correct fail-safe, but the CLI must make warnings loud.
3. **Per-provider ChangePct heterogeneity** — yahoo daily vs coingecko 24h vs dolarapi none; document in rule-creation errors and alert output so semantics are never ambiguous.
4. **Threshold flap** — a value oscillating around the threshold re-fires each re-arm; acceptable at EOD cadence, hysteresis deferred.
5. **Watchlist drift** — rules validated against watchlist at creation can become orphan if assets are later removed from YAML; engine must skip missing assets with a warning, not crash.
