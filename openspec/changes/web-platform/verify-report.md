```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:9834c8e918388d1c0a6cb362d638c3edbfd6193398cea65f252f9619eb29acf3
verdict: fail
blockers: 3
critical_findings: 3
requirements: 8/8
scenarios: 15/18
test_command: go test ./...
test_exit_code: 0
test_output_hash: sha256:601b3afc5bd4def68be811dd343da18908730052bd05aa3e9b7f3cb4f5b68b61
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: web-platform
**Version**: Revision 2 design; Standard verification
**Mode**: Standard
**Status**: fail

### Executive summary
The implementation builds, passes the full Go test/build/vet gate, matches the revised routing table, consumes the intended store contracts, and leaves protected files untouched. Verification fails because three required spec scenarios lack a passing covering test: valid archived-report rendering, the unavailable-engine sequencing block, and stale quote rendering that proves the UI never styles stale data as current.

### Completeness
| Metric | Value |
|---|---:|
| Tasks total | 23 |
| Tasks complete | 23 |
| Tasks incomplete | 0 |
| Requirements complete | 8/8 |
| Scenarios compliant | 15/18 |

### Findings
**CRITICAL**
1. `UNTESTED` valid archived report scenario: `web-pages/spec.md:25-29` requires a valid available `YYYY-MM-DD` report to render; `internal/web/views_test.go:151-183` covers only invalid and unavailable dates, with no passing test for available dated data.
2. `UNTESTED` engine-unavailable sequencing scenario: `alert-rule-management/spec.md:31-36` requires preparation to block when the engine contract is unavailable; no test or executable guard covers this condition. The current suite only proves the merged contract path (`internal/web/rules_test.go:26-189`).
3. `UNTESTED` stale quote view scenario: `web-pwa/spec.md:31-39` requires a quote-bearing page to show a visible stale warning and never imply a live quote; `internal/web/freshness_test.go:9-55` tests state calculation, but no page test renders a stale quote and asserts the stale class/warning is present and current styling is absent.

**WARNING**
1. Offline coverage is partial: `internal/web/pwa_test.go:137-155` verifies the explicit offline marker does not fabricate `última actualización`, but does not execute service-worker cache fallback under a simulated network failure.
2. Proposal success criteria remain unchecked in `openspec/changes/web-platform/proposal.md:60-65` despite the corresponding implementation/gate evidence passing; the change metadata is stale and should not be treated as completion bookkeeping.

**SUGGESTION**
1. Add a route-registration assertion that inspects the actual method-qualified patterns, rather than relying primarily on non-conflicting construction and response status checks in `internal/web/routes_test.go:31-72`.
2. Make the first-load budget test assert the measured total in a stable logged/fixture form so the reported `68,919 B` evidence remains directly reproducible.

### Spec coverage table
| Capability / requirement | Scenario | Covering test | Result |
|---|---|---|---|
| web-pages / auth seam | Authenticated dashboard | `internal/web/routes_test.go:143-171 > TestNewMux_AuthenticatedDashboard` | COMPLIANT |
| web-pages / auth seam | Unauthenticated data request renders no data | `internal/web/auth_test.go:158-199 > TestAuthorizerMiddleware_ProtectsDataRoutes`; `routes_test.go:31-59` | COMPLIANT |
| web-pages / auth seam | Valid archived report | No covering test; `views_test.go:151-183` covers only invalid/unavailable | UNTESTED |
| web-pages / progressive HTML | JavaScript-disabled watchlist | `internal/web/views_test.go:185-211 > TestWatchlist_GroupsBySection` | COMPLIANT |
| web-pages / progressive HTML | Freshness visible | `internal/web/views_test.go:55-98 > TestDashboard_RendersFreshnessAndAlertBadge` | COMPLIANT |
| web-read-layer / batched latest | Newest pair and missing isolation | `internal/store/web_reads_test.go:60-118` | COMPLIANT |
| web-read-layer / batched latest | Empty batch performs no query | `internal/store/web_reads_test.go:120-144` | COMPLIANT |
| web-read-layer / bounded history | In-range deterministic history | `internal/store/web_reads_test.go:146-227` | COMPLIANT |
| web-read-layer / bounded history | Invalid limit/range errors | `internal/store/web_reads_test.go:229-265` | COMPLIANT |
| web-read-layer / isolation | Contract-safe platform change | `git diff main...HEAD` protected-file check; `web_reads.go` | COMPLIANT |
| alert-rule-management / frozen contract | Valid rule, armed state, percentage baseline | `internal/web/rules_test.go:26-189` | COMPLIANT |
| alert-rule-management / frozen contract | Disable/remove preserves history | `internal/web/rules_test.go:256-442` | COMPLIANT |
| alert-rule-management / frozen contract | Invalid/duplicate safe errors | `internal/web/rules_test.go:94-189` | COMPLIANT |
| alert-rule-management / sequencing | Engine contract unavailable blocks | No covering test | UNTESTED |
| web-pwa / installable shell | Manifest/icons/service worker | `internal/web/pwa_test.go:13-135` | COMPLIANT |
| web-pwa / performance | Under 200 KB and non-blocking JS | `internal/web/pwa_test.go:158-222` | COMPLIANT |
| web-pwa / offline shell | Offline marker without fabricated freshness | `internal/web/pwa_test.go:137-155` | PARTIAL |
| web-pwa / honest freshness | Stale warning and never current styling | No page-level stale rendering test; `freshness_test.go:9-55` only | UNTESTED |

### Design coherence
| Decision | Evidence | Result |
|---|---|---|
| Exact root and dual collection routes | `internal/web/routes.go:34-65` includes `GET /{$}`, both slash forms for report/assets/alerts, and no subtree catch-all patterns | PASS |
| Handler-to-store contract | `internal/web/views.go:119-284`, `rules.go:120-306`; only `CreateRule`, `ListRules`, `RemoveRule`, `SetRuleEnabled`, `LatestQuote`, `History`, and read-layer additions are used; no production SQL in `internal/web` | PASS |
| Protected files unchanged | `git diff --name-only main...HEAD -- internal/store/store.go internal/render cmd/report` returned empty | PASS |
| Auth and no-JS fallback | `internal/web/auth.go:22-87`, `rules.go:193-241` | PASS |
| 24-hour freshness boundary | `internal/web/freshness.go:43-55`; exact boundary test at `freshness_test.go:19-31` | PASS |
| PWA SRI/defer and honest offline marker | `internal/web/pwa_test.go:198-248`, `137-155` | PASS with offline-runtime coverage warning |

### Proposal success criteria
| Criterion | Proposal checkbox | Verification state |
|---|---|---|
| Routes/auth render SQLite data with visible freshness | Unchecked | Evidence passes implementation/runtime smoke; metadata stale |
| Rule CRUD uses frozen contract | Unchecked | Evidence passes tests/static inspection; metadata stale |
| PWA assets, <200 KB, near-zero blocking JS | Unchecked | Evidence passes tests; HTTPS/Lighthouse remains deployment-dependent |
| Full Go test/build/vet without protected edits | Unchecked | PASS: all three commands exit 0; protected diff empty |

### Gate results
| Command | Exit | Evidence |
|---|---:|---|
| `go test ./...` | 0 | `test_output_hash=sha256:601b3afc5bd4def68be811dd343da18908730052bd05aa3e9b7f3cb4f5b68b61` |
| `go build ./...` | 0 | `build_output_hash=sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `go vet ./...` | 0 | Empty output; same empty-output digest `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Protected diff | empty | `git diff --name-only main...HEAD -- internal/store/store.go internal/render cmd/report` |

### Runtime smoke
| Check | Result | Evidence |
|---|---|---|
| Temporary DB server startup | PASS | `WEB_AUTH_PASSWORD=test WEB_SESSION_SECRET=testsecret WEB_DB_PATH=<temp> WEB_ADDR=127.0.0.1:18080 go run ./cmd/web` |
| `GET /healthz` | PASS | `200`, body `ok` |
| Unauthenticated `GET /` | PASS | `303 See Other`, `Location: /login`, empty body |
| Login POST | PASS | `303 See Other`, `Location: /`, Secure/HttpOnly/SameSite=Lax session cookie |
| Authenticated `GET /` | PASS | `200`, HTML document containing `Inicio`, `asesor helper`, and freshness marker |
| Cleanup | PASS | Process terminated and temporary DB/log/cookie files removed |

### Final verdict
**FAIL** — implementation and runtime gate are healthy, but required scenario-level evidence is incomplete for three scenarios.

## Key Learnings

1. Full Go gates can pass while required scenario-level coverage remains incomplete.
2. Exact-root ServeMux registration prevents unknown subtree paths from reaching dashboard handlers.
3. Freshness unit tests do not substitute for page-level stale-rendering evidence.
