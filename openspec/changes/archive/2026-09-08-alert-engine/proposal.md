# Proposal: Motor de alertas: el asesor configura thresholds de valor o de porcentaje por activo y el sistema genera alertas

## Intent

Give the advisor threshold alerts over stored quotes; report rendering remains unchanged.

## Scope

### In Scope
- SQLite rules/history keyed by `(source, symbol)`.
- Baseline-drift value/percentage evaluation with persisted state and automatic re-arm.
- `cmd/alerts` developer CLI: `run`, `add`, `list`, `remove`, `history`.

### Out of Scope
- Report/template changes, channels, scheduling, web UI, multi-user data, hysteresis, or broader rule editing.
- Configurable assets (deferred under issue #1); add validates `watchlist.yaml`.

## Capabilities

### New Capabilities
- `alert-rules`: Persist threshold rules.
- `alert-evaluation`: Evaluate quotes, deduplicate, re-arm, and persist history.
- `alert-cli`: Operate alerts through `cmd/alerts`.

### Modified Capabilities
- None; report behavior remains unchanged.

## Approach

Add `alert_rules(id, source, symbol, kind, direction, threshold, baseline_price, enabled, state, created_at)` with checks and unique identity. Add `alerts(id, rule_id nullable FK ON DELETE SET NULL, source, symbol, kind, threshold, observed_price, observed_change_pct, baseline_price, quote_fetched_at, triggered_at)`; snapshots survive removal. `add` captures the latest quote, rejects missing quotes, and makes baselines immutable (delete/recreate).

Evaluate latest SQLite quotes only. Value compares price inclusively; percentage computes `(price-baseline_price)/baseline_price*100`, valid for all providers including dolarapi. Stale/missing quotes (default 24h, `-max-age` override) warn loudly and do not change state. Armed→triggered inserts once; a false condition silently re-arms.

Flags include database path, `-max-age`, and add fields for source, symbol, kind, direction, and threshold.

## Delivery and Review Workload Forecast

Use **two chained PRs / feature-branch-chain**: `feature/alert-engine` tracker; PR1 targets it, PR2 targets PR1, and only the tracker merges to `main`.

- PR1: schema/store (~140), engine (~120), foundational tests (~100): **~360 lines**.
- PR2: CLI (~150), remaining tests (~200): **~350 lines**.

Decision needed before apply: No  
Chained PRs recommended: Yes  
400-line budget risk: Medium

## Testing Strategy

Offline table-driven tests cover matching, baseline math, dedupe/re-arm, and stale/missing guards. `t.TempDir()` SQLite tests cover schema, CRUD, uniqueness, latest quotes, history, and state updates. Verify with `go test ./...`, `go vet ./...`, and `go build ./...`; CLI smoke is manual.

## Risks and Rollback

| Risk | Mitigation |
|---|---|
| Baseline captured from stale quote | Require a quote and report its timestamp. |
| Provider quote semantics differ | Persist observed values/timestamps and display warnings. |
| Watchlist drift | Skip orphaned rules with warnings, never crash. |

Rollback: stop invoking `cmd/alerts`, revert the chained PRs, and retain/archive alert tables; report data/rendering are untouched.

## Success Criteria

- [ ] Rules and auditable alerts persist in SQLite with immutable baselines.
- [ ] Inclusive thresholds, staleness safety, dedupe, and automatic re-arm are verified offline.
- [ ] Both chained PRs remain within the 400-line review budget and `go test ./...` passes.
