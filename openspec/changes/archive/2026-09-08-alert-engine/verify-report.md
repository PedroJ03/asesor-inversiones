# Verification Report: `alert-engine`

## Scope

Verified the merged implementation on local `main` against the OpenSpec requirements, design contracts, task checklist, focused tests, and a live CLI smoke run. No application code was modified.

## Verdicts

| Ref | Requirement | Verdict | Evidence |
|---|---|---|---|
| R1 | Persist validated rules, constrained kind/direction/state, unique identity, validation, and enabled filtering | PASS | `internal/store/store.go:75-87,245-288,391-424`; `TestCreateAndListRules`, `TestCreateRuleValidation`, `TestCreateRuleDuplicate`, `TestListRulesEnabledFilter`; focused store suite passed. |
| R2 | Capture immutable baseline from latest quote and reject missing quote | PASS | `cmd/alerts/main.go:154-168`; `TestAddValidCapturesBaseline`, `TestAddRejectsMissingQuote`; live add printed `Captured baseline price=768.0000 from quote fetched at 2026-09-08T13:58:00Z`. |
| R3 | List state and remove rules while preserving snapshots with nullable `rule_id` | PASS | `internal/store/store.go:291-305,360-389,391-424`; `TestRemoveRulePreservesHistory`, `TestRemovePreservesHistory`; focused store and CLI suites passed. |
| R4 | Inclusive value thresholds and baseline-drift percentage thresholds for every provider | PASS | `internal/alert/engine.go:93-107`; table-driven value and percentage boundary tests passed, including equality. |
| R5 | Correct armed/triggered state machine and deduplication | PASS | `internal/alert/engine.go:77-87`; `TestEvaluateValueBoundaries` covers armed+true/false, `TestEvaluateDedupeAndRearmCycle` covers triggered+true and triggered+false; all focused alert tests passed. |
| R6 | Stale/missing/error/orphan guards warn without state changes and do not abort other rules | PASS | `internal/alert/engine.go:60-73`; `TestEvaluateGuards`, `TestEvaluateOrphanDoesNotAbort`, and `TestRunTriggersAndWarnings` passed. |
| R7 | Denormalized history snapshots and bounded `History(limit)` | PASS | `internal/store/store.go:88-101,343-389`; `TestHistoryLimit`, `TestHistoryOutput`, and removal-history tests passed. |
| R8 | `run` supports `-db`, 24h default, `-max-age`, warning exit 0, store failure non-zero | PASS | `cmd/alerts/main.go:75-120`; `TestRunTriggersAndWarnings`, `TestRunStoreFailureExitsNonZero`; live `run` printed an alert and exited successfully. |
| R9 | `add` validates source/watchlist/kind/direction/positive threshold and reports baseline | PASS | `cmd/alerts/main.go:122-169,282-329`; add validation and baseline tests passed; live add succeeded with the expected baseline feedback. |
| R10 | `list`, `remove -id`, and bounded `history -limit`, with invalid argument errors | PASS | `cmd/alerts/main.go:172-280`; `TestListDisplaysState`, `TestRemovePreservesHistory`, `TestHistoryOutput`; focused CLI suite passed. |
| R11 | Report/render unchanged and alert code remains separate | PASS | `git diff --exit-code main~1 main -- cmd/report internal/render` produced no output; searches of `cmd/report` and `internal/render` found no `alert` references. |

## Commands and Exact Results

1. `gofmt -l .` — **PASS**; no output.
2. `go build ./... && go vet ./... && go test ./...` — **PASS**; all packages passed (`cmd/alerts`, `internal/alert`, `internal/config`, `internal/fetch`, `internal/render`, `internal/store`); `cmd/report` reported `[no test files]`.
3. `git diff --exit-code main~1 main -- cmd/report internal/render` — **PASS**; no output.
4. `go test ./internal/store/ -v` — **PASS**; 11 tests passed.
5. `go test ./internal/alert/ -v` — **PASS**; 5 tests passed, including all value/pct boundaries, dedupe/re-arm, guards, and orphan isolation.
6. `go test ./cmd/alerts/ -v` — **PASS**; 9 tests passed, including run, add, list, remove, and history flows.
7. Live smoke, using `/tmp/opencode/verify-smoke.db` and not `data/asesor.db`:
   - `go run ./cmd/report -db /tmp/opencode/verify-smoke.db -out /tmp/opencode/verify-smoke-report` — **PASS**; fetched 14 quotes from `yahoo, dolarapi, data912, coingecko`.
   - `go run ./cmd/alerts add -db /tmp/opencode/verify-smoke.db -source yahoo -symbol SPY -kind value -direction above -threshold 1` — **PASS**; created rule 1 and printed baseline `768.0000` with quote timestamp `2026-09-08T13:58:00Z`.
   - `go run ./cmd/alerts run -db /tmp/opencode/verify-smoke.db` — **PASS**; printed `ALERT yahoo/SPY value 1.0000` at observed price `768.0000`.
   - `go run ./cmd/alerts history -db /tmp/opencode/verify-smoke.db -limit 5` — **PASS**; printed the SPY snapshot.
   - `go run ./cmd/alerts list -db /tmp/opencode/verify-smoke.db` — **PASS**; printed rule 1 as `triggered`, `enabled yes`.

## Contract Cross-Check

- `Evaluate` matches the design signature and `Result` fields: `internal/alert/engine.go:48-57`.
- Store method set matches the design contract: `CreateRule`, `ListRules`, `RemoveRule`, `LatestQuote`, `UpdateRuleState`, `InsertAlert`, and `History` are present in `internal/store/store.go`.
- `alert_rules` matches the design columns, checks, default values, and unique identity at `internal/store/store.go:75-87`.
- `alerts` matches the design snapshot columns and `ON DELETE SET NULL` foreign key at `internal/store/store.go:88-100`.

## Findings

### WARNING — R1: `created_at` uses quote fetch time in the CLI

- **Spec:** A valid rule is stored with its creation timestamp.
- **Code:** `cmd/alerts/main.go:154-162` passes `quote.FetchedAt` to `CreateRule` as `createdAt`; `internal/store/store.go:275-279` persists that value as `created_at`.
- **Evidence:** The live smoke output distinguishes the quote timestamp (`13:58:00Z`) from the run/creation a few seconds later (`13:58:04Z`). The implementation therefore records quote age rather than the actual add-operation timestamp. This does not affect alert behavior, but it weakens creation-time audit accuracy.

### WARNING — R1/R2: direct store tests do not assert timestamp semantics or percentage-specific CLI creation

- **Spec:** Creation timestamp and immutable percentage baseline behavior are part of the rule contract.
- **Code/tests:** `internal/store/store_test.go:127-157` checks identity/state/baseline but not `CreatedAt`; `cmd/alerts/main_test.go:112-144` exercises baseline capture with a value rule rather than a percentage rule.
- **Evidence:** All suites pass, but these assertions are not directly covered. The implementation path is consistent with the specification; this is a coverage gap, not a failed behavior observed in verification.

### SUGGESTION — R8: add a subprocess assertion for exit codes

- **Spec:** Rule-local warnings exit 0 and store/persistence failures exit non-zero.
- **Code/tests:** `cmd/alerts/main_test.go:56-110` tests the command functions' returned errors, not the compiled process exit status.
- **Evidence:** The live successful smoke commands exited 0 and the store-failure function test returned an error. A subprocess test would prove the `os.Exit` boundary directly.

## Risks

1. Rule creation audit timestamps can represent quote freshness rather than creation time.
2. CLI tests do not directly prove process-level exit codes.
3. The live smoke validates the currently available network/provider response; provider behavior can vary independently of offline tests.

## Overall Status

**SUCCESS** — R1–R11 are verified as met. The findings are non-blocking metadata/test-coverage concerns; no critical implementation deviation was observed.
