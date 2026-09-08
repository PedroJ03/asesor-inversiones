# Pre-Proposal State: web-platform

- schema: gentle-ai.sdd-preproposal/v1
- revision: 4
- change: web-platform
- artifact_store: hybrid
- updated_at: 2026-09-07

## Exploration

- outcome: done — Option B (Go server-rendered pages + progressive enhancement, same repo: cmd/web + internal/web) selected
- references: openspec/changes/web-platform/exploration.md | engram topic sdd/web-platform/explore

## Research Request

- lane: Go HTML-first stack state of the art (template engine + progressive-enhancement library decision)
- classes_requested: documentation, open-web
- questions:
  1. html/template (Go stdlib) vs templ — maturity, maintenance state, Go 1.26 compatibility, component-style ergonomics, go:embed behavior.
  2. htmx 2.x vs vanilla JS vs Alpine — form-driven CRUD, search-as-you-type symbol picker, partial list re-rendering; current stable version and minified+gzipped size against the <200 KB first-load budget.
  3. Documented pitfalls with Go 1.22+ stdlib http.ServeMux method+pattern routing.

## Research Admission and Outcome

- capability: gentle-ai.sdd-research-capability/v1 (revision 3 re-entry declaration, user-approved open-web expansion)
- observed grants: documentation=[context7: Go stdlib (/golang/go), a-h/templ, bigskysoftware/htmx, alpinejs/alpine], open-web=[GitHub releases pages for the three repos; npm/jsdelivr/unpkg records for htmx.org/alpinejs; go.dev release notes]
- runtime exercisability (rev 3): documentation EXERCISED (context7); open-web DECLARED BUT NOT EXERCISABLE in the sdd-research executor runtime (no web-fetch tool there). Revision 4 closure: the open-web legs were exercised by the orchestrator-level `general` agent (a runtime WITH web-fetch capability) against exactly the declared open-web source classes; its source-cited report is incorporated into research.md revision 4 as sources W1–W13.
- outcome: done — revision 4 closed every revision-3 gap with cited evidence: templ maturity/cadence/Go-1.26 tracking (v0.3.1020, 2026-05-10, ~2–2.5-month cadence, go.mod go 1.25.0, compiler bumped to Go 1.26); htmx exact size (16,588 B min+gzip official .gz) and version line resolved (pin htmx.org@2.0.10; v4.0.0 is on the npm `next` tag, 10 days old); Alpine 3.16.3 (54,447 B uncompressed exact, ~18,659 B gz approximation); vanilla-JS baseline (0 B runtime); ServeMux conflict semantics (most-specific wins; genuinely conflicting patterns panic at registration, order-independent; `/foo/` subtree vs `/foo/{$}` exact).
- evidence references: openspec/changes/web-platform/research.md (revision 4, outcome done) | engram topic sdd/web-platform/research (revision 4 condensed mirror; OpenSpec holds the full artifact per orchestrator instruction)

## Product Decisions

- confirmed (locked by user, do not re-litigate): Architecture B — Go server-rendered + progressive enhancement in the same repo (`cmd/web` + `internal/web`), single static binary, pure-Go SQLite (`modernc.org/sqlite`); v1 scope = display-first UI + alert-rule CRUD via the frozen alert-engine contract; no real-time, drag-and-drop, or live charts; mobile-first (single column ≤720px, bottom tab bar, <200 KB first load, near-zero blocking JS); PWA basics (manifest + minimal service worker); internet-facing advisor-only behind an auth middleware seam; placeholder brand "asesor helper".
- pending: template engine (html/template vs templ) and progressive-enhancement library (htmx vs Alpine vs vanilla JS) — the two decisions this research lane feeds. The evidence is compatible with more than one pairing; selection is orchestrator/user-owned.

## proposal_ready

pending user confirmation of the two lane-fed stack decisions — the research lane itself is complete (revision 4, outcome `done`, all gaps closed with cited evidence). The orchestrator presents the two decisions (template engine: html/template vs templ; progressive-enhancement library: htmx vs Alpine vs vanilla) to the user with the evidence summary. Once both are confirmed, `sdd-propose` may run.
