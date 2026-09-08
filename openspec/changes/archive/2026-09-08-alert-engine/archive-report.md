# Archive Report: `alert-engine`

**Change title**: Motor de alertas: el asesor configura thresholds de valor o de porcentaje por activo y el sistema genera alertas
**Archived**: 2026-09-08
**Final status**: **SHIPPED**
**Implementation tip at close**: `b07d1ed` (`test: add pct rule coverage for CLI add and end-to-end drift`) on `main`
**Archive commit**: the commit introducing this file (`chore: archive alert-engine change and sync main specs`)

This report is the terminal record of the SDD cycle. It describes the change AT CLOSE per the Final-State Authority hierarchy: persisted tasks artifact first, orchestrator final-state facts second, intermediate snapshots (`apply-progress`, `verify-report`) attributed as history only.

## Final Status

**SHIPPED.** All runtime objectives are complete. The full feature lives on `main` at `b07d1ed`: SQLite `alert_rules`/`alerts` persistence, the pure `internal/alert` evaluator (inclusive value + baseline-drift pct thresholds, two-state armed/triggered machine with dedupe and automatic re-arm, stale/missing/orphan guards), and the `cmd/alerts` CLI (`run`, `add`, `list`, `remove`, `history`). Report rendering is untouched (R11).

## Specs Synced to Source of Truth

All three delta specs were purely `ADDED` requirements into previously empty domains (no MODIFIED/REMOVED/RENAMED; no destructive merges — `rules.archive` warning not triggered). Main specs were created by mechanical copy, each verified byte-identical by empty `diff -r` readback:

| Domain | Action | Source of truth |
|---|---|---|
| `alert-rules` | Created (6 requirements) | `openspec/specs/alert-rules/spec.md` |
| `alert-evaluation` | Created (4 requirements) | `openspec/specs/alert-evaluation/spec.md` |
| `alert-cli` | Created (4 requirements) | `openspec/specs/alert-cli/spec.md` |

## Artifact Lineage

Hybrid store: OpenSpec files authoritative (now under this archive folder), Engram mirrors.

| Artifact | Archived file | Engram topic | Observation ID |
|---|---|---|---|
| exploration | `exploration.md` | `sdd/alert-engine/explore` | #290 |
| proposal | `proposal.md` | `sdd/alert-engine/proposal` | #292 |
| spec (3 deltas) | `specs/{alert-rules,alert-evaluation,alert-cli}/spec.md` | `sdd/alert-engine/spec` | #297 |
| design | `design.md` | `sdd/alert-engine/design` | #295 |
| tasks | `tasks.md` | `sdd/alert-engine/tasks` | #301 (upserted at archive: 4.1 reconciled) |
| apply-progress | (Engram only) | `sdd/alert-engine/apply-progress` | ID not surfaced by FTS search; existence confirmed by session summary #305 |
| verify-report | `verify-report.md` | `sdd/alert-engine/verify-report` | ID not surfaced by FTS search; existence confirmed by session summary #325 |
| state | (Engram only) | `sdd/alert-engine/state` | #291 (9 revisions) |
| archive-report | `archive-report.md` (this file) | `sdd/alert-engine/archive-report` | saved at archive time |

Session summaries on record: #294 (explore→propose pause), #293 (propose), #298 (spec), #296 (design), #305 (apply PR1), #313 (apply PR2), #325 (verify).

## Delivery Summary

Strategy: `ask-on-risk` resolved to **chained PRs / feature-branch-chain** (400-line budget risk: Medium). Forecast vs. outcome:

- **PR #2** (`feature/alert-engine-foundation` → tracker `feature/alert-engine`, merged `d4f214e`): store + evaluator + foundational tests. Authored diff came to **900 lines** (894 additions, 6 deletions) vs the 400-line review budget — the cohesive store+engine+tests unit could not be split without breaking behavior. **`size:exception` approved by the maintainer.**
- **PR #3** (`feature/alert-engine-cli` → PR1 branch, merged `b98d6f7`): `cmd/alerts` CLI + remaining tests. Initially merged into the wrong base; corrected by tracker merge **`5854c8c`** (completing the PR #3 chain into the tracker).
- **PR #5** (tracker → `main`, merged `322438b`), with corrective merge **`5d3d924`** (tracker into main: store + evaluator + CLI).
- Merge-order task 4.1 was executed by the maintainer, per plan; only the tracker merged to `main`.

## Verification Summary

Per `verify-report.md` (verification of merged `main`, 2026-09-08): **R1–R11 all PASS**, no CRITICAL findings. Gates at verification: `gofmt -l .` clean; `go build ./... && go vet ./... && go test ./...` green; `git diff --exit-code main~1 main -- cmd/report internal/render` empty (R11); live CLI smoke passed (14 quotes fetched, rule created with baseline 768.0000, trigger, history, list).

**Post-verify remediation — commit `b07d1ed`** (final-state facts outrank the verify snapshot):

- **RESOLVED** the pct CLI coverage WARNING: added `TestAddPctCapturesBaseline` (pct rule via CLI: kind pct, baseline 400 captured, armed) and `TestPctEndToEnd` (add pct threshold 5 → new quote 430 → drift asserted **exactly 7.5** in history).
- Same commit, **disclosed behavior-neutral deviation**: fixed 3 pre-existing tests whose hardcoded `2026-09-07` timestamp would eventually break the 24h staleness guard; switched to dynamic UTC time.

**Final test totals at close: 27 tests green (11 store + 5 alert + 11 CLI)** — superseding the 25 (11 + 5 + 9 CLI) recorded in the verify-report snapshot, which predates `b07d1ed`.

## Task Completion Record

`tasks.md`: **12/12 complete.** One exceptional archive-time reconciliation was performed and is recorded here as required:

- **Task 4.1** (merge order) was left unticked by `sdd-apply` because merge execution was maintainer-controlled after the apply phase closed (commit `1ba7396` marked 0.1–3.4 complete only). The checkbox was reconciled to `[x]` at archive time.
- **Reconciliation reason / proof**: git merge history — `d4f214e` (PR #2), `b98d6f7` (PR #3), `5854c8c` (corrective tracker merge), `322438b` (PR #5), `5d3d924` (tracker→main) — plus `main` tip `b07d1ed` carrying the complete feature, and the verify-report validating the merged implementation on `main`. The orchestrator's final-state facts explicitly authorized archive-time reconciliation on this evidence.

## Remaining Open Findings (carried forward, intentionally not fixed)

1. **WARNING — metadata accuracy**: `created_at` stores the quote fetch time instead of the rule creation time (`cmd/alerts/main.go` passes `quote.FetchedAt` as `createdAt`). Does not affect alert behavior; weakens creation-time audit accuracy.
2. **SUGGESTION — test coverage**: `cmd/alerts` exit codes are tested through returned errors, not compiled subprocess exit status.

These are non-blocking follow-ups for a future change.

## Ledger Lineage Note

All runtime objectives complete: the PR1 attempt failed on budget → maintainer reset → remediated by PR2 passing settle; PR2, verify, and test-pct-cli objectives all complete. Roadmap next: `web-platform` (PRs #6/#7, parallel user session — out of scope for this archive).

## Audit Notes

- Delta spec sync performed BEFORE the archive move (mechanical `cp` via mktemp + `diff -r`); change folder moved with `git mv` and verified byte-identical against a pre-move recursive snapshot (empty `diff -r`).
- `openspec/specs/.gitkeep` removed (domains now hold real specs).
- `openspec/changes/web-platform/` (parallel user work) was not touched.
