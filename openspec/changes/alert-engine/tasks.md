# Tasks: Motor de alertas (alert-engine)

## Review Workload Forecast

- PR1 store+engine+tests → tracker `feature/alert-engine`: ~360 lines.
- PR2 `cmd/alerts` CLI+tests → PR1 branch: ~350 lines.
- Delivery: ask-on-risk resolved — chained PRs confirmed.

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: Medium

### Suggested Work Units

| # | Goal | PR (base) | Focused test | Harness | Rollback |
|---|---|---|---|---|---|
| 1 | Rule persistence | PR1 (tracker) | `go test ./internal/store/ -run Rule` | N/A no CLI | migrations+CRUD |
| 2 | Eval persistence | PR1 (tracker) | `go test ./internal/store/` | N/A no CLI | methods |
| 3 | Evaluator | PR1 (tracker) | `go test ./internal/alert/` | N/A pure fn | delete pkg |
| 4 | `run` | PR2 (PR1) | `go test ./cmd/alerts/ -run Run` | `go run ./cmd/alerts run` | run path |
| 5 | `add` | PR2 (PR1) | `go test ./cmd/alerts/ -run Add` | `go run ./cmd/alerts add` | add path |
| 6 | list/remove/history | PR2 (PR1) | `go test ./cmd/alerts/` | `go run ./cmd/alerts list` | subcommands |

Refs — R1 persist · R2 baseline · R3 list/remove · R4 thresholds · R5 state machine · R6 guards · R7 history · R8 run · R9 add · R10 inspect · R11 report unchanged.

## Phase 0: Chain setup

- [ ] 0.1 Create tracker `feature/alert-engine` from `main`, push; commit `openspec/changes/alert-engine/` docs there (keeps child diffs code-only) — `chore: add alert-engine SDD planning artifacts`; branch PR1 `feature/alert-engine-foundation` from tracker.

## Phase 1: PR1 — store

- [ ] 1.1 Add `alert_rules`+`alerts` migrations to `internal/store/store.go` per design SQL, plus `CreateRule`+`ListRules(enabledOnly)` with validation + uniqueness error; tests: migration, valid/invalid/duplicate, list-filter (R1, R3 listing). `go test ./internal/store/`. Commit: `feat: add alert rules schema, creation, and listing`
- [ ] 1.2 Add `RemoveRule`, `LatestQuote`, `UpdateRuleState`, `InsertAlert`, `History(limit)` per contract; tests: snapshots-survive-removal, history, latest quote (R2, R3, R7). `go test ./internal/store/`. Commit: `feat: add quote lookup, rule state, and alert history`

## Phase 2: PR1 — engine

- [ ] 2.1 Create `internal/alert/engine.go` — types + `Evaluate(rules, resolve, now, maxAge)` per design contract; table-driven `engine_test.go` (R4–R6). `go test ./internal/alert/`. Commit: `feat: add pure alert evaluation engine`
- [ ] 2.2 PR1 gate: `gofmt -l .` empty; `go build ./... && go vet ./... && go test ./...`; ≤400 authored lines vs tracker; open PR1 → tracker with chain context + diagram (PR1 📍). No commit.

## Phase 3: PR2 — CLI

- [ ] 3.1 Branch `feature/alert-engine-cli` from PR1 branch; create `cmd/alerts/main.go` + `run` (`-db`, `-max-age`, 24h default) per design flow; exit semantics per R8; tests (R8). `go test ./cmd/alerts/`. Commit: `feat: add alerts run subcommand`
- [ ] 3.2 Add `add` flags per design; whitelist {yahoo, dolarapi, data912, coingecko} + watchlist validation, baseline capture via `LatestQuote` + printed feedback, missing-quote rejection; tests (R9). `go test ./cmd/alerts/ -run Add`. Commit: `feat: add alerts add subcommand with baseline capture`
- [ ] 3.3 Add `list`, `remove -id`, `history -limit`; invalid id/limit are command errors; tests: history-survives-removal (R10). `go test ./cmd/alerts/`. Commit: `feat: add alerts list, remove, and history subcommands`
- [ ] 3.4 PR2 gate: `gofmt -l .` empty; `go build ./... && go vet ./... && go test ./...`; `git diff --stat main...HEAD -- cmd/report internal/render` empty proves R11; smoke `go run ./cmd/alerts` on seeded db; open PR2 → PR1 branch. No commit.

## Phase 4: Merge order

- [ ] 4.1 Merge PR1 → tracker; retarget PR2 → tracker; merge PR2 → tracker; then tracker → `main` only.

## Guardrails

- In: SQLite rules/history, pure evaluator, `cmd/alerts` run/add/list/remove/history.
- Out: report/template changes, channels, scheduler, web UI, multi-user, hysteresis, rule editing beyond add/remove, configurable assets (#1).
- Threat matrix N/A → no RED tests; TDD disabled → tests ship in each unit.
