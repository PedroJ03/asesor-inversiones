# Design: Motor de alertas: el asesor configura thresholds de valor o de porcentaje por activo y el sistema genera alertas

## Technical Approach

Add a pure `internal/alert` evaluator over immutable rules and stored quotes. `internal/store` owns SQLite schema, CRUD, latest-quote lookup, history, and state writes. `cmd/alerts` composes both; report code remains untouched. Percentage rules use `(price-baseline_price)/baseline_price*100`, with the creation baseline never edited.

## Architecture Decisions

| Decision | Choice | Rejected / rationale |
|---|---|---|
| Evaluation source | Stored latest quotes, guarded by age | Re-fetching duplicates network work; embedding in `cmd/report` violates separation. |
| Rule/history persistence | SQLite tables in the existing store | YAML cannot safely own mutable state or future web writes. |
| Percentage meaning | Immutable creation baseline, valid for every provider | Daily `ChangePct` is not available for dolarapi and has provider-specific semantics. |
| CLI boundary | Separate `cmd/alerts` with `run/add/list/remove/history` | Modifying report would couple unrelated workflows. |
| Delivery | Two chained PRs, each ≤400 authored lines | PR1 store+engine+foundational tests (~360); PR2 CLI+remaining tests (~350). |

## Data Flow

```text
alerts run → Open store → enabled rules → latest quote(source,symbol)
          → stale/missing guard → pure Evaluate → state updates + alert inserts
          → print triggered alerts and loud warnings
```

`add` validates the watchlist and source, loads the latest quote, captures `baseline_price`, and rejects missing quotes. `run` defaults to `24h`; `-max-age` overrides it. Orphaned rules are skipped with warnings.

## Interfaces / Contracts

`internal/alert` exposes `Evaluate(rules []Rule, resolve func(string,string)(Quote,bool,error), now time.Time, maxAge time.Duration) Result`, where `Result` contains `Triggered []Alert`, `Transitions []StateChange`, and `Warnings []string`. It performs no I/O. Value conditions compare price inclusively; percentage conditions compare computed drift inclusively.

`internal/store` adds `CreateRule`, `ListRules(enabledOnly bool)`, `RemoveRule`, `LatestQuote(source,symbol)`, `UpdateRuleState(id,state)`, `InsertAlert`, and `History(limit int)`. `CreateRule` receives the captured baseline and timestamps; `LatestQuote` returns the newest `fetched_at` row. CLI flags are `-db`, `-max-age` (run), and add fields `-source -symbol -kind -direction -threshold`; list/remove/history use `-id` or `-limit` as applicable.

## Data Model and State Machine

```sql
alert_rules(id INTEGER PRIMARY KEY AUTOINCREMENT, source TEXT NOT NULL, symbol TEXT NOT NULL,
 kind TEXT CHECK(kind IN ('value','pct')) NOT NULL, direction TEXT CHECK(direction IN ('above','below')) NOT NULL,
 threshold REAL NOT NULL, baseline_price REAL NOT NULL, enabled INTEGER NOT NULL DEFAULT 1,
 state TEXT CHECK(state IN ('armed','triggered')) NOT NULL DEFAULT 'armed', created_at DATETIME NOT NULL,
 UNIQUE(source,symbol,kind,direction,threshold));
alerts(id INTEGER PRIMARY KEY AUTOINCREMENT, rule_id INTEGER REFERENCES alert_rules(id) ON DELETE SET NULL,
 source TEXT NOT NULL, symbol TEXT NOT NULL, kind TEXT NOT NULL, threshold REAL NOT NULL,
 observed_price REAL, observed_change_pct REAL, baseline_price REAL, quote_fetched_at DATETIME,
 triggered_at DATETIME NOT NULL);
```

| Current state | Condition | Result |
|---|---|---|
| armed | true (including equality) | insert snapshot; triggered |
| armed | false | unchanged |
| triggered | true | unchanged; no duplicate |
| triggered | false | armed; no alert |
| either | missing/stale/error quote | unchanged; warning |

## Error Handling

Fail safe: rule-local quote errors become warnings; the CLI continues and exits successfully. Invalid input, database/migration failure, or write failure is a command error. `add` rejects missing quotes, invalid assets/kinds/directions, and non-positive baselines.

## File Changes

| File | Action |
|---|---|
| `internal/store/store.go` | Migrations and persistence methods. |
| `internal/alert/engine.go` | Pure evaluator and state machine. |
| `internal/store/store_test.go`, `internal/alert/engine_test.go` | PR1 foundational tests. |
| `cmd/alerts/main.go` | Subcommands, validation, and output. |
| `cmd/alerts/*_test.go` | PR2 CLI/history/remove tests. |
| `cmd/report/main.go`, `internal/render/*` | No changes. |

## Testing Strategy

PR1 uses table-driven engine tests for value/pct math, directions, equality, state cycles, and stale/missing warnings; store tests use `t.TempDir()` for migrations, uniqueness, latest quotes, state, and history. PR2 adds CLI validation, baseline capture, list/remove/history, and output tests; manual smoke is `go run ./cmd/alerts ...`. Run `go test ./...`, `go vet ./...`, and `go build ./...`.

## Threat Matrix

No boundary is introduced; `cmd/alerts` is directly invoked. Every matrix row is N/A: no safe/failure behavior or RED test applies.

| Boundary | Status / reason | Safe/failure behavior | RED test |
|---|---|---|---|
| Documentation-like paths | N/A — no execution | N/A | None |
| Git repository selection | N/A — no VCS automation | N/A | None |
| Commit state | N/A — no commits | N/A | None |
| Push state | N/A — no pushes | N/A | None |
| PR commands | N/A — no PR automation | N/A | None |

## Migration / Rollout

`CREATE TABLE IF NOT EXISTS` migrations follow `store.go`; existing report data is unchanged. Rollout is opt-in by invoking `cmd/alerts`; rollback is to stop invoking it and revert the two chained PRs.

## Open Questions

None; confirmed decisions are baseline drift, automatic re-arm, 24h default, separate CLI, and feature-branch chaining.
