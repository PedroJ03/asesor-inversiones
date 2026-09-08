# Research: web-platform — Go HTML-First Stack State of the Art

- schema: gentle-ai.sdd-research/v1
- revision: 4 (supersedes revision 3 `partial`; revision 1 was blocked on empty capability grants)
- change: web-platform
- lane: go-html-first-stack
- outcome: done
- date: 2026-09-07
- store_mode: hybrid (OpenSpec file is the full artifact; Engram topic `sdd/web-platform/research` carries a condensed mirror per orchestrator instruction)

## Selected Request (retained intent — unchanged since revision 1)

**Lane**: Go HTML-first stack state of the art — feeding exactly one design decision: template engine + progressive-enhancement library for a server-rendered, mobile-first platform.

**Questions**:
1. `html/template` (Go stdlib) vs `templ`: current maturity, maintenance state, compatibility with Go 1.26, ergonomics for component-style pages, behavior with `go:embed`.
2. `htmx` 2.x vs vanilla JS vs Alpine for: form-driven CRUD (alert rule management), a search-as-you-type symbol picker, partial list re-rendering. Current stable version, minified+gzipped size (must fit a <200 KB total first-load budget on mid-range Android over Argentine mobile networks).
3. Documented pitfalls combining these with Go 1.22+ stdlib `http.ServeMux` pattern routing (method+pattern routes).

**Requested source classes**: documentation, open-web.

## Admission

- capability: gentle-ai.sdd-research-capability/v1 (revision 3 declaration, issued under explicit user approval to close revision-2 gaps)
- declared grants:
  - `documentation`: [context7 library documentation for: Go standard library (`html/template`, `net/http` ServeMux), templ (`a-h/templ`), htmx (`bigskysoftware/htmx`), Alpine.js (`alpinejs/alpine`)]
  - `open-web`: [GitHub releases/commits/tags pages for `a-h/templ`, `bigskysoftware/htmx`, `alpinejs/alpine`; npm/jsdelivr/unpkg package and size records for `htmx.org` and `alpinejs`; go.dev release notes]
- revision 3 runtime note: the sdd-research executor runtime has no web-fetch tool, so open-web legs were DECLARED BUT NOT EXERCISABLE there.
- revision 4 gap-closure method: the open-web legs were exercised by the orchestrator-level `general` agent (a runtime WITH web-fetch capability) against exactly the granted open-web source classes, and its source-cited evidence report is incorporated here as sources W1–W13. No claim derives from training memory; every open-web claim cites a fetched URL.

## Sources

S1–S26 carried from revisions 2–3 (context7 documentation channel, accessed 2026-09-07). W1–W13 are new in revision 4 (open-web channel, accessed 2026-09-07, gathered by the orchestrator-level general agent per the gap-closure method above).

### Documentation sources (revisions 2–3, carried)

| ID | Class | Title | Publisher | URL | Excerpt |
|----|-------|-------|-----------|-----|---------|
| S1 | documentation | html/template.ParseFS API (go1.26.7) | Go project | https://github.com/golang/go/blob/go1.26.7/api/go1.16.txt | `func ParseFS(fsys fs.FS, patterns ...string) (*Template, error)` |
| S2 | documentation | embed.FS.Open API + go:embed (go1.26.7) | Go project | https://github.com/golang/go/blob/go1.26.7/api/go1.16.txt | `method (FS) Open(name string) (fs.File, error)`; embed.FS populated via `//go:embed` |
| S3 | documentation | templ — build HTML with Go (docs index) | a-h/templ | https://github.com/a-h/templ/blob/main/docs/docs/index.md | components compiled to "performant Go code... type-safe"; fragments compose into pages; SSR + static rendering |
| S4 | documentation | Creating an HTTP server with templ | a-h/templ | https://github.com/a-h/templ/blob/main/docs/docs/05-server-side-rendering/01-creating-an-http-server-with-templ.md | `templ.Handler(component)` is an http.Handler; `component.Render(r.Context(), w)` inside any handler |
| S5 | documentation | templ script/static-asset serving | a-h/templ | https://github.com/a-h/templ/blob/main/docs/docs/03-syntax-and-usage/13-script-templates.md | `mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(...)))` alongside `templ.Handler(...)` |
| S6 | documentation | htmx README (master) | bigskysoftware/htmx | https://github.com/bigskysoftware/htmx/blob/master/README.md | CDN include `htmx.org@2.0.10/dist/htmx.min.js` with SRI; motivation: support "more HTTP methods beyond GET and POST" |
| S7 | documentation | htmx site index | bigskysoftware/htmx | https://github.com/bigskysoftware/htmx/blob/master/www/content/_index.md | "htmx is small (~16k min.gz'd), dependency-free..." |
| S8 | documentation | hx-trigger attribute reference | bigskysoftware/htmx | https://github.com/bigskysoftware/htmx/blob/master/www/content/attributes/hx-trigger.md | `hx-get="/search" hx-trigger="input changed delay:1s" hx-target="#search-results"` |
| S9 | documentation | htmx example: update other content | bigskysoftware/htmx | https://github.com/bigskysoftware/htmx/blob/master/www/content/examples/update-other-content.md | `<form hx-post="/contacts" hx-target="#table-and-form">` re-renders list + form fragment |
| S10 | documentation | Alpine state (x-data) | alpinejs/alpine | https://github.com/alpinejs/alpine/blob/main/packages/docs/src/en/essentials/state.md | `<div x-data="{ open: false }">` local reactive state |
| S11 | documentation | Alpine start-here: live search filter | alpinejs/alpine | https://github.com/alpinejs/alpine/blob/main/packages/docs/src/en/start-here.md | x-data + x-model + x-for client-side filtering of an in-memory list |
| S12 | documentation | Alpine x-init directive | alpinejs/alpine | https://github.com/alpinejs/alpine/blob/main/packages/docs/src/en/directives/init.md | `x-init="posts = await (await fetch('/posts')).json()"` — hand-written fetch |
| S13 | documentation | Alpine x-on (.prevent) | alpinejs/alpine | https://github.com/alpinejs/alpine/blob/main/packages/docs/src/en/directives/on.md | `<form @submit.prevent="...">` — request handling is hand-written JS |
| S14 | documentation | net/http ServeMux pattern docs | Go project | https://github.com/golang/go/blob/master/src/net/http/server.go | Method-qualified patterns; `{NAME}`/`{NAME...}`/`{$}`; "the method GET matches both GET and HEAD"; wildcards are full path segments; values via `Request.PathValue` |
| S15 | documentation | ServeMux trailing-slash redirection | Go project | https://github.com/golang/go/blob/master/src/net/http/server.go | `/images/` registered → `/images` redirects to `/images/` "unless `/images` has been registered separately" |
| S16 | documentation | ServeMux pattern grammar + Go 1.21 fallback | Go project | https://github.com/golang/go/blob/master/src/net/http/pattern.go | `[METHOD] [HOST]/[PATH]` grammar (Go 1.22); `GODEBUG=httpmuxgo121=1` restores the frozen pre-1.22 mux |
| S17 | documentation | ServeMux routing tree internals | Go project | https://github.com/golang/go/blob/master/src/net/http/routing_tree.go | Tree ordered host → method → path segments; wildcard segments under empty-string keys |
| S18 | documentation | templ installation — Go version requirement | a-h/templ | https://github.com/a-h/templ/blob/main/docs/docs/02-quick-start/01-installation.md | "Requires Go 1.24 or greater" for the templ CLI |
| S19 | documentation | templ SECURITY.md — supported versions | a-h/templ | https://github.com/a-h/templ/blob/main/SECURITY.md | "The latest version of templ is supported." (security fixes land on latest only) |
| S20 | documentation | context7 index metadata for /a-h/templ | context7 (index of a-h/templ) | resolve-library-id output | Indexed Versions: `v0.3.906`; 1655 snippets; Source Reputation: High |
| S21 | documentation | hx-delete attribute reference | bigskysoftware/htmx | https://github.com/bigskysoftware/htmx/blob/master/www/content/attributes/hx-delete.md | hx-delete "triggers a DELETE request to the specified URL. The response HTML is then swapped into the DOM according to the hx-swap strategy." |
| S22 | documentation | hx-put attribute reference | bigskysoftware/htmx | https://github.com/bigskysoftware/htmx/blob/master/www/content/attributes/hx-put.md | hx-put on an element sends a PUT request to a specified URL; controls swap target and strategy |
| S23 | documentation | htmx docs — AJAX section | bigskysoftware/htmx | https://github.com/bigskysoftware/htmx/blob/master/www/content/docs.md | "Attributes like `hx-get`, `hx-post`, `hx-put`, `hx-patch`, and `hx-delete` specify the HTTP method and the URL for the request." |
| S24 | documentation | htmx essay: "HTML: The Bad Parts" | bigskysoftware/htmx | https://github.com/bigskysoftware/htmx/blob/master/www/content/essays/spa-alternative.md | Native HTML constraints: only `<a>`/`<form>` make requests, only click/submit trigger them, "only GET and POST HTTP methods being widely available" |
| S25 | documentation | context7 index metadata for /bigskysoftware/htmx | context7 (index of bigskysoftware/htmx) | resolve-library-id output | Indexed Versions: `v1.9.12`, `v2.0.4`, `v4.0.0`; 1968 snippets; Source Reputation: High |
| S26 | documentation | Alpine installation — CDN version pinning | alpinejs/alpine | https://github.com/alpinejs/alpine/blob/main/packages/docs/src/en/essentials/installation.md | Pinned example `<script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.16.3/dist/cdn.min.js">`; docs advise pinning a specific version |

### Open-web sources (revision 4, gap closure)

| ID | Class | Title | Publisher | URL | Excerpt |
|----|-------|-------|-----------|-----|---------|
| W1 | open-web | templ releases | GitHub (a-h/templ) | https://github.com/a-h/templ/releases | Latest: v0.3.1020, released 10 May 2026; prior: v0.3.1001 (28 Feb 2026), v0.3.977 (31 Dec 2025), v0.3.960 (15 Oct 2025) — cadence ≈ every 2–2.5 months |
| W2 | open-web | templ commits on main | GitHub (a-h/templ) | https://github.com/a-h/templ/commits/main | Most recent commit 15 Jul 2026 (deps bump + LSP feature work) — active maintenance |
| W3 | open-web | templ go.mod | GitHub (a-h/templ) | https://raw.githubusercontent.com/a-h/templ/main/go.mod | `module github.com/a-h/templ — go 1.25.0` |
| W4 | open-web | templ v0.3.1001 changelog | GitHub (a-h/templ) | https://github.com/a-h/templ/releases | "chore: bump compiler to Go 1.26" — project tracks current Go toolchains (repo: 10.5k stars, 366 forks, 19 open issues) |
| W5 | open-web | jsDelivr metadata for htmx.org@2.0.10 | jsDelivr data API | https://data.jsdelivr.com/v1/packages/npm/htmx.org@2.0.10 | `"name": "htmx.min.js", "size": 51238`; `"name": "htmx.min.js.gz", "size": 16588` — official pre-gzipped file: **16,588 B (≈16.2 KiB) min+gzip** is authoritative |
| W6 | open-web | jsDelivr metadata for alpinejs@3.16.3 | jsDelivr data API | https://data.jsdelivr.com/v1/packages/npm/alpinejs@3.16.3 | `"name": "cdn.min.js", "size": 54447` — uncompressed bytes exact; no .gz ships in package |
| W7 | open-web | Bundlephobia size for alpinejs@3.16.3 | Bundlephobia API | https://bundlephobia.com/api/size?package=alpinejs@3.16.3 | `"gzip":18659, "size":53187, "version":"3.16.3"` — ≈18.7 KB gz (re-bundled approximation of entry point, not an exact measurement of cdn.min.js) |
| W8 | open-web | npm dist-tags for htmx.org | npm registry | https://registry.npmjs.org/-/package/htmx.org/dist-tags | `{"latest":"2.0.10","next":"4.0.0"}` |
| W9 | open-web | npm latest record for htmx.org | npm registry | https://registry.npmjs.org/htmx.org/latest | `"_id":"htmx.org@2.0.10"` — `npm install htmx.org` resolves to 2.0.10 |
| W10 | open-web | htmx releases | GitHub (bigskysoftware/htmx) | https://github.com/bigskysoftware/htmx/releases | GitHub "Latest" = v4.0.0, released 28 Aug 2026 (10 days before fetch); v2.0.10 remains the npm `latest` and README-pinned line |
| W11 | open-web | Go 1.22 release notes — enhanced routing | Go project | https://go.dev/doc/go1.22 | "If two patterns overlap in the requests that they match, then the more specific pattern takes precedence. If neither is more specific, the patterns conflict." |
| W12 | open-web | Go blog — routing enhancements | Go project | https://go.dev/blog/routing-enhancements | "Registering both of them (in either order!) will panic." — conflicting patterns panic at registration time; "/foo/ is a subtree match, /foo/{$} is exact" |
| W13 | open-web | htmx README quick-start pin | bigskysoftware/htmx | https://raw.githubusercontent.com/bigskysoftware/htmx/master/README.md | `<script src="https://cdn.jsdelivr.net/npm/htmx.org@2.0.10/dist/htmx.min.js"` — upstream pins 2.0.10; "~14k min.gz'd" claim is stale marketing vs the authoritative 16,588 B record (W5) |

## Validated Claims

### Q1 — html/template vs templ

- C1 [S1, S2]: In the Go 1.26.7 source tree, `html/template` provides `ParseFS(fsys fs.FS, ...)` and `embed.FS` implements `fs.FS`; stdlib templates load from `//go:embed` assets.
- C2 [S3]: templ compiles HTML-in-Go components into type-safe Go code; fragments compose into full pages; SSR and static rendering documented.
- C3 [S4, S5]: templ serves through stdlib `net/http` unmodified — `templ.Handler` yields an `http.Handler`, `Render(ctx, w)` works inside any handler, static assets use ordinary `ServeMux` + `FileServer`/`StripPrefix`.
- C4 (CLOSED in revision 4) [S18, S19, W1, W2, W3, W4]: templ maturity is now evidenced: minimum toolchain Go 1.24 (templ's go.mod tracks Go 1.25.0); security support covers latest version only; **latest release v0.3.1020 (10 May 2026), release cadence ≈ every 2–2.5 months, main-branch commits through 15 Jul 2026**; the project explicitly bumped its compiler to Go 1.26 (v0.3.1001 changelog). Repo traction: 10.5k stars, 366 forks, 19 open issues. Residual caveat: still **v0.x semver** — pin exact versions and read changelogs on upgrade.
- C5 [S1, S2]: html/template's cited API is present in the go1.26.7 source tree (stdlib Go 1.26 compatibility evidenced). html/template carries zero dependency/maintenance risk (stdlib), but has weaker component ergonomics and no compile-time type checking of templates.

### Q2 — htmx 2.x vs vanilla JS vs Alpine

- C6 (CLOSED in revision 4) [W8, W9, W10, W13]: htmx version line resolved — **the correct pin is `htmx.org@2.0.10`** (npm `latest`, upstream README-endorsed). v4.0.0 is real (GitHub "Latest", 28 Aug 2026) but sits on the npm `next` tag and was 10 days old at fetch time — a deliberate bleeding-edge choice, not the stable line.
- C7 (CLOSED in revision 4) [W5]: htmx 2.0.10 exact size evidenced — **minified 51,238 B; gzipped 16,588 B (≈16.2 KiB)** from the package's official pre-gzipped file. That is ~8% of the 200 KB first-load budget.
- C8 [S8]: htmx's documented search-as-you-type pattern is `hx-get` + `hx-trigger="input changed delay:1s"` + `hx-target` — server round-trip returning HTML fragments. Matches the symbol-picker use case.
- C9 [S6, S9]: htmx's documented form-CRUD pattern posts forms via `hx-post` and re-renders partial fragments via `hx-target`/`hx-swap`.
- C10 [S21, S22, S23, S24]: `hx-put`/`hx-delete` issue real PUT/DELETE AJAX requests from markup; the browser-native GET/POST form limitation does not apply; no `_method` override convention is needed.
- C11 [S10–S13]: Alpine provides client-side reactive state (`x-data`, `x-model`, `x-for`) sufficient for local search filtering and form control, but server synchronization is hand-written `fetch` rather than attribute-driven HTML-over-the-wire.
- C12 (CLOSED in revision 4) [S26, W6, W7]: Alpine 3.16.3 is the docs-pinned stable line; exact uncompressed `cdn.min.js` is 54,447 B; gzipped ≈18,659 B (Bundlephobia approximation — treat as close, not exact). Fits the budget.
- C13 (CLOSED in revision 4): Vanilla JS requires no library — 0 bytes of runtime. It is the baseline against which htmx (16.6 KB gz) or Alpine (~18.7 KB gz) are the entire library cost. No fetch required.

### Q3 — Go 1.22+ ServeMux pitfalls

- C14 [S14, S16]: Patterns are `[METHOD] [HOST]/[PATH]` with `{name}`, `{name...}`, `{$}` wildcards; wildcard values via `Request.PathValue`; a `GET` pattern also matches `HEAD`.
- C15 [S14, S15]: Pitfall — trailing slash acts as a subtree match; ServeMux redirects `/images` → `/images/` unless `/images` is registered separately.
- C16 [S14]: Pitfall — wildcards must be full path segments; `/b_{bucket}` is invalid.
- C17 [S16]: Pitfall — pre-1.22 routing restorable via `GODEBUG=httpmuxgo121=1`; semantics differ across that boundary.
- C18 (CLOSED in revision 4) [W11, W12]: ServeMux conflict semantics evidenced — overlapping patterns resolve to the **most specific**; when neither is more specific, **registering both panics at registration time (startup), order-independent**. Fail-fast at boot, not a runtime request-time failure.

## Contradictions / Uncertainty / Freshness

- **htmx version-line tension: RESOLVED** (C6). npm `latest` = 2.0.10, `next` = 4.0.0; GitHub "Latest" tag points at the 10-day-old v4.0.0 while npm and the README pin 2.0.10. Pin 2.0.10 + SRI; revisit 4.x only after it promotes to `latest`.
- **Alpine gzip is an approximation**: W7's 18,659 B re-bundles the package entry point rather than measuring `cdn.min.js` itself (uncompressed 54,447 B exact per W6). For a 200 KB budget the ~35 KB combined htmx+Alpine figure has ample headroom either way.
- **Freshness**: htmx sources are from `master`; ServeMux documentation sources S14–S17 are from `master` (may drift ahead of go1.26.x) but C18's conflict semantics come from the Go 1.22 release notes + official blog (stable); S1–S2 are pinned to the `go1.26.7` tag.
- No contradictions between sources on mechanics.

## Product Choices

None. Template-engine and progressive-enhancement selection remains orchestrator/user-owned. This artifact deliberately makes no recommendation.

## Outcome

done — all revision-3 gaps closed with cited open-web evidence: templ maturity/cadence/Go-1.26 tracking (C4), htmx exact size + version line (C6, C7), Alpine version + size (C12), vanilla baseline (C13), ServeMux conflict/panic semantics (C18). Every claim cites a fetched source; nothing derives from training memory.

- proposal_ready: pending user confirmation of the two stack decisions (template engine, progressive-enhancement library) — the evidence lane itself is complete.
