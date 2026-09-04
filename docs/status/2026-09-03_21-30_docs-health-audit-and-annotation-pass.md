# Status Report — Docs-Health AUDIT: Living-Docs Rebuild + Full ANNOTATE Pass

**Date:** 2026-09-03 21:30 CEST
**Session scope:** Full docs-health pipeline (VERIFY + HARVEST + BUILD + ANNOTATE) over all
non-archived `2026-0*` files and every living doc, triggered by
_"View ALL \*\*/2026-0\* files! Execute the docs-health SKILL!"_.
**Base:** `725693b` (session start, clean tree) → HEAD (auto-daemon committed the
work incrementally in 9 heuristic commits; the final report file commits with this one).
**Verification at time of writing:** `nix flake check` ✅ · build ✅ · vet ✅ ·
full test suite `-race` ✅ (run twice: after the code fix and at session end) ·
fuzz smoke (`FuzzWantsJSON`, `FuzzFingerprintChecks`) ✅ · `nix run .#vulncheck`
— **no vulnerabilities** ✅ · coverage baseline **76.9%** ✅ · CI on master
**5/5 green including the Browser (runtime CSP) job** (`gh run view 33763955031`) ✅ ·
pkg.go.dev **fetched: v0.3.1 indexed** ✅ · internal link sweep: **0 broken refs** ✅

**Format override note:** the status-report skill's canonical output is styled
HTML; the user explicitly requested `.md` at a `.md` path, so Markdown was used.

---

## a) FULLY DONE

| #  | Work                                                                                                                                                                                                                                                                                                                                                                                                                                                    | Evidence                                                                                     |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| 1  | **docs-health skill pipeline executed properly** — SKILL.md + all 8 references (ownership, harvest, build, verify-checklist, resolving-items, annotation-placement, agents-quality, report-format) + assets read before any action                                                                                                                                                                                                                      | Skill flow followed; dry-run-before-mutate honored (and it caught a real bug — see d.2)      |
| 2  | **All 11 in-scope `2026-0*` files read in full** — 6 status reports (04-03, 04-21, 04-41, 04-42, 09-02 sweep, 09-03 execution-complete, 09-03 retrospective), 3 planning docs (Pareto plan, issue drafts, decision notes), 1 research HTML (examined → explicit LEAVE decision), plus all 7 living docs                                                                                                                                                 | The research HTML was skipped-unexamined in a prior session (04-03 c4) — that loop is closed |
| 3  | **Verification battery against code** — test/bench/fuzz counts (154 funcs / 19 files), `Version = "0.3.1"`, CI jobs, pkg.go.dev indexing, vulncheck, coverage, CI run 33763955031 green 5/5 incl. browser job                                                                                                                                                                                                                                           | commands + `gh run view` + `fetch` output in session                                         |
| 4  | **Real bug fixed: meaningless test** — `lifecycle_test.go:205` asserted `errors.Is(err, err)` (trivially true; flagged by 04-41 d1 and 04-42 f1). Now asserts `errors.Is(err, dashboard.ErrPusherNotActive)`; suite green                                                                                                                                                                                                                               | `lifecycle_test.go:205-207`, `go test -race` PASS                                            |
| 5  | **FEATURES.md de-drifted (31 edits)** — ~28 stale `file:line` citations (nearly all wrong after the cycle's refactors, e.g. `dashboard.go:184` pointed at `WithTitle`) replaced with durable symbol citations (`Handler()`/`wantsJSON()`, `groupChecks()`, `renderPatch()`, …); counts fixed to 154/19 with the counting command; Released row v0.2.0 → **v0.3.1 fetch-verified**; CI row 4/4 → 5/5 verified; missing **samber/do lifecycle row** added | `FEATURES.md`, spot-checked line-by-line before rewriting                                    |
| 6  | **README.md gaps closed (retro f41–f43 + 04-41/04-42 Register asks)** — `WithDescription`/`WithPublicMode` options rows; histogram metric lines + scrape-config snippet matching `deploy/prometheus.yml`; `docs/screenshot-dark.png` embedded with capture note; `Register()` DI note after Quick Start                                                                                                                                                 | `README.md`; dark PNG exists (86 KB)                                                         |
| 7  | **ROADMAP.md lifecycle cleanup** — 13 shipped items removed from "raw ideas" (drain, lifetime, watchdog, rate limit, histogram, timeline, timestamp, metrics, export, public mode, axe, auth, OG, RecommendedCSP, sub-path); 10 new raw ideas routed from the retro; **"stateless view layer" non-goal corrected** (trend ring is bounded view state); 2 new Open Questions (release policy, fingerprint compat); incident tracking restored to Theme 3 | `ROADMAP.md`                                                                                 |
| 8  | **CHANGELOG.md fingerprint compatibility paragraph** (retro f5) added to `[Unreleased]` — documents the one-spurious-change-on-upgrade semantics                                                                                                                                                                                                                                                                                                        | `CHANGELOG.md` Fixed section                                                                 |
| 9  | **AGENTS.md inventory de-drifted** — +3 missing source files (`csp.go`, `ratelimit.go`, `trend.go`) with one-liners; test-file list replaced with the real 19-file inventory; **bisect-wall gotcha** added (`72783fc` does not compile; `git bisect skip`; daemon mid-edit-tree lesson)                                                                                                                                                                 | `AGENTS.md` Architecture + Gotchas                                                           |
| 10 | **DOMAIN_LANGUAGE.md +10 terms and +3 routes** — Sample, History Buffer, Status Transition, Latency Histogram, Shutdown Drain, Rate Limiter, Public Mode, BasePath, RetryInterval, SubscriberCount; route table gained `/health/metrics`, `/health/trend`, `/health/export` (closes sweep f30, 04-21 item 28/e11)                                                                                                                                       | `docs/DOMAIN_LANGUAGE.md`                                                                    |
| 11 | **TODO_LIST.md rebuilt from scratch** — trophy "Done (recent)" section deleted; 100%-DONE "Next Up" table removed; **24 open items in 6 themed groups + 3 BLOCKED**, every item citing its source report AND code evidence; coverage/vulncheck/CI-browser items closed inline with today's verification results                                                                                                                                         | `TODO_LIST.md`                                                                               |
| 12 | **30 broken cross-references in archived reports fixed** — all `docs/(status\|planning)/2026-…` refs rewritten to `archived/` paths (open since 04-03 f1/Q1); every target verified to resolve. This was the "single biggest failure" of the 08-09 session — now actually fixed                                                                                                                                                                         | `rg` sweep: 0 remaining broken refs                                                          |
| 13 | **ANNOTATE pass: ~229 inline item verdicts across 8 files, zero appendix-only** — retrospective 33 · sweep report 53 (40-row table + b/c/g) · execution-complete 1 · 04-21 37 (28 script + 9 manual) · 04-03 26 (23 rows + Q1–Q3 inline) · 04-41 14 · 04-42 33 · Pareto plan 32 M-rows + resolution appendix for the 166 fine tasks                                                                                                                     | strikethrough + `done at` hashes / verified-evidence markers throughout                      |
| 14 | **HARVEST executed with routing rigor** — recent-report next-tasks verified against code before routing; shipped items closed with cycle hashes (`d453c52`, `f627164`, `dd483c2`, `bd99de0`, `be5fe4c`, `3022fbf`, `50f2bcc`, `e9f47cb`, `4e4a149`); no done-item re-harvested                                                                                                                                                                          | `TODO_LIST.md` + annotations                                                                 |
| 15 | **Docs gate, not just code gate** — post-edit link sweep over all living docs and reports: every referenced path resolves (including `.github/workflows/*`, screenshots, report-to-report refs)                                                                                                                                                                                                                                                         | session link sweep                                                                           |

## b) PARTIALLY DONE

1. ~~**CONTRIBUTING.md** — flagged by two reports (browser-test how-to, `Register`/DI
   mention). I filed it in TODO_LIST instead of doing the ~20-minute edit while I was
   already editing docs. Works: nothing — file untouched. Remaining: the edit.
   Blocker: none, pure prioritization failure. Effort: S.~~ done at `db8621f` (sweep) + coverage-floor note added 2026-09-04
2. ~~**doc.go examples** — `Register` + `ErrPusherNotActive` Quick Start additions
   (04-42 f21/f22). README got the Register note; doc.go did not. Remaining: doc.go
   edit only. Effort: S.~~ done at `db8621f`
3. **Screenshot regeneration docs (sweep f15)** — dark screenshot now embedded with a
   capture-note; the **light** screenshot still has no caption and there is no
   documented regenerate one-liner. Effort: S.
4. ~~**Coverage floor** — baseline recorded (76.9%) but no CI floor or artifact upload
   (retro f9/f14). Blocker: artifact upload wants a verified `actions/upload-artifact`
   SHA first. Effort: M.~~ done at `db8621f` (verified-SHA upload + 75% floor)
5. ~~**Harvest depth** — 24 of ~30 verified-open candidates routed into TODO_LIST/ROADMAP.
   Six were left unrouted **without a documented decision** (CSV fuzz target,
   RecommendedCSP fuzz target, keyboard-nav a11y smoke, metrics-under-strict-CSP browser
   test, "protect probes via network policy" note, unchecked `injector.Shutdown()` in
   `lifecycle_test.go`). Blocker: none — see g.3. Effort: S.~~ resolved at `c7e4f13` — all six routed to ROADMAP; the injector.Shutdown check already existed (lifecycle_test.go:119)
6. **AGENTS.md size** — grew ~1 KB in a pass that should also have pruned toward the
   15 KB budget (now ~21 KB, "acceptable" band). Remaining: prune pass. Effort: M.
7. **execution-complete annotations** — lightest touch of the six reports (one inline
   pointer). Its build-tag bullet is the standing open item and got no explicit marker
   (the retro + sweep carry the full dispositions). Effort: S.
8. ~~**04-41 residual ideas** — `BenchmarkHealthCheck`, Dashboard self-monitoring test,
   `do.Package`, shutdown-ordering test verified still open but **not routed** to
   ROADMAP. Effort: S.~~ resolved at `c7e4f13` — routed to the ROADMAP DI-surface bullet
9. **Local browser suite** — skipped in my final verification run (no
   `GO_HEALTH_DASHBOARD_CHROME` in my shell); I relied on CI's green browser job from
   today instead. Works: CI proof. Missing: local confirmation. Effort: S.

## c) NOT STARTED

_(Deliberately untouched this session — all now tracked in the rebuilt TODO_LIST /
ROADMAP, none silently dropped)_

1. ~~**Release cut** (v0.4.0 vs v0.3.2) — waiting on the release-policy question (g.1);
   `[Unreleased]` batch is fully written and ready to re-head.~~ done — v0.4.0 (`8f63d85`) and v0.5.0 (`ed650bf`) shipped; the v0.6.0 batch is a TODO_LIST row
2. ~~**CI hardening cluster** — fuzz.yml `workflow_dispatch` validation, golangci-lint pin
   (`version: latest` today), templ CLI pin (`@latest` ×4 today), concurrency group,
   issue-on-failure for nightly fuzz, coverage artifact.~~ done at `db8621f` (fuzz dispatch verified: run 33896794771)
3. ~~**Code-quality refactors** — split `dashboard.go` (~700 lines), extract
   `history.go`, dedupe trend/export JSON mapping, `TrendHandler` 503 message,
   `BenchmarkDashboard_PatchRender` rename (verified: it serves full HTML), inline
   `maxRequestsInvalid`.~~ done at `db8621f`
4. ~~**Testing-gap items** — two flaky `time.Sleep`s in `sse_integration_test.go`
   (`:265`, `:492` — verified still present), `WithMaxSSEConnections(0)` test,
   probe-not-started SSE test, `style=` patch assertion, sentinel state split.~~ done at `db8621f`
5. ~~**Upstream work** — PR to templ-components for the StatCard `<dl>` fix (+ goldens),
   then remove the repo's axe rule-level tolerance.~~ open — TODO_LIST rows for #6 and #7 (added 2026-09-04)
6. ~~**Example toggles** — `DEMO_PUBLIC=1`, `DEMO_BASE_PATH=/status`.~~ done at `db8621f`
7. ~~**Bisectability audit** of `071c251..HEAD` (the `72783fc` wall is documented in
   AGENTS.md; the full 58-commit audit is not done).~~ done 2026-09-04 — 91 commits audited, 86 build; audit archived
8. **Next Pareto planning pass** — the rebuilt TODO_LIST is the input universe.
9. ~~**Unrouted brainstorm items** (b.5) — pending the g.3 decision.~~ resolved at `c7e4f13` — routed
10. ~~**ARCHIVE moves** — deliberately none: every non-archived `2026-0*` file retains
    genuinely open items (blocked decisions, upstream PR, flaky tests), and the skill
    requires _every_ item resolved before `git mv` to `archived/`.~~ done 2026-09-04 docs-health pass — 6 status + 4 planning files archived after inline disposition

## d) TOTALLY FUCKED UP

1. **I repeated the documented heredoc/backslash trap the same day it was written down.**
   The retrospective (published this morning, d.2: "the tool's JSON layer decodes
   backslash sequences before bash sees it") describes exactly what I then did: my
   annotation scripts lost `\` characters through the tool→bash→python layers. Result:
   an arg-order crash, a regex that lost a backslash and matched nothing, then a
   "fix" that didn't even parse (SyntaxError) — **~4 wasted tool calls** before I
   built the no-regex char-loop splitter. Severity: wasted session time, no damage.
   Root cause: I didn't apply a lesson I had _read an hour earlier_. The chr(92)-
   construction trick was the documented fix and I reached for raw regex first anyway.
2. **My first 04-03 annotation run silently dropped all markers.** The script reported
   "annotated 23 rows" while a `.replace()` target never matched — the rows were
   struck through with **zero** `done at` markers, which is precisely the docs-health
   #1 failure mode (appendix-only/blank strikes). My post-write re-read caught it and
   a second pass inserted all 23 markers. Severity: would have shipped corrupted
   annotations had I trusted the script's success output. Root cause: replace-target
   string didn't match the joined-cell format; the script asserted shape (line count)
   but not marker presence.
3. **Two of my scratch scripts crashed on their own CLI contract** (arg order, then
   self-inflicted syntax error while "patching" the first mistake). Pure sloppiness
   under time pressure; the stock skill scripts were always the right first choice for
   prose lists, and only the escaped-pipe table genuinely needed a custom splitter.
4. **Inconsistent harvest discipline.** I route-or-dropped ~6 verified-open items
   without recording the decision anywhere (see b.5). A future session will re-find
   them as "new" ideas — the exact rot HARVEST exists to prevent. Severity: low but
   self-inflicted future confusion.
5. **Split brain sat in TODO_LIST for a day on my watch.** The file claimed "Status is
   verified, not assumed" while its "Next Up" table was 100% DONE and a trophy
   "Done (recent)" section duplicated CHANGELOG. I fixed it this session, but the
   morning read should have been the trigger — it took the user's push.
6. **No local golangci-lint run.** I leaned on CI's green Lint job; the v0.3.x cycle's
   own process lesson was "lint before declaring done." Flake check only covers
   treefmt, not golangci.

## e) WHAT WE SHOULD IMPROVE

1. **Marker-presence assertion for annotation scripts** — shape checks (line count)
   are not enough; a script that strikes rows must assert every struck line contains
   its marker. Would have made d.2 impossible.
2. **Inspect DRY output for structure damage, not just exit status** — the dry-run
   _did_ reveal the escaped-pipe corruption (`\|` split into two cells); the lesson is
   to treat dry-run as a diff to read, not a gate to pass.
3. **Dumbest-script-first for prose with escapes** — the no-regex char-loop splitter
   worked on the first try; regex-through-heredoc failed three ways. Default to char
   loops when backslashes are in play; reach for `chr(92)` before writing a literal.
4. **Route-or-record rule** — every verified-open item gets a destination
   (TODO_LIST/ROADMAP) or a one-line "left in report because…" note. No silent drops.
5. **Do-the-20-minute-fix bias** — when a docs pass already has the context loaded
   (CONTRIBUTING, doc.go), file-it-in-TODO_LIST is procrastination, not hygiene.
6. **Grow-and-prune in the same pass** — any doc edit that adds content should check
   the file against its size budget and prune stale content in the same edit session.
7. **Docs passes run the lint gate too** — `nix run .#lint` alongside `nix flake
   check`; "CI is green" is evidence, not a substitute for the local gate the cycle
   retro demanded.
8. **Name the commit for multi-file doc passes** — the daemon's 9 heuristic commits
   fragment this session's work; one detailed `docs:` commit (like `40ba449`) is far
   more bisectable. Commit the report + any residue with a real message at session
   end, before the daemon eats it.

## f) UP TO 50 THINGS TO DO NEXT

_Brainstorm ranked by impact; HARVEST with routing rigor before treating as
commitments. Items 1–24 already live in `TODO_LIST.md` (harvested today); 25–50 are
new observations from this session awaiting the g.3 decision or refined routing._

**Dispositioned 2026-09-04:** items 1–21 and 23–24 shipped (v0.4.0/v0.5.0 cycle +
2026-09-04 sweep — see CHANGELOG); 22 open (TODO_LIST upstream row); 25–32 done or
routed at `c7e4f13`/today (CONTRIBUTING, doc.go, route-or-record, screenshots →
ROADMAP, 04-41 residuals → ROADMAP DI bullet); 33–50 executed (the lint gate is
now standard), routed to ROADMAP, or superseded. Canonical backlog:
`TODO_LIST.md` + `ROADMAP.md`.

| #  | Task                                                                                                                                                                               | Impact | Effort | Category      |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Cut the next release: re-head CHANGELOG `[Unreleased]`, bump `Version`, tag (pending g.1)                                                                                          | High   | S      | Release       |
| 2  | CI guard: `Version` const must match the latest git tag (bitten twice)                                                                                                             | High   | S      | Quality       |
| 3  | Trigger fuzz.yml via workflow_dispatch; confirm crasher-print end-to-end                                                                                                           | Medium | S      | Quality       |
| 4  | Pin golangci-lint version (currently `version: latest` in ci.yml)                                                                                                                  | Medium | S      | Quality       |
| 5  | Pin templ CLI in CI (currently `@latest`, four install steps)                                                                                                                      | Medium | S      | Quality       |
| 6  | Add CI concurrency group to cancel superseded runs                                                                                                                                 | Low    | S      | Quality       |
| 7  | Nightly fuzz: open an issue on failure instead of only printing crashers                                                                                                           | Low    | S      | Quality       |
| 8  | Coverage artifact upload (verify actions/upload-artifact SHA first) + optional CI floor                                                                                            | Low    | M      | Quality       |
| 9  | Split `dashboard.go` (~700 lines): config/options vs lifecycle vs handlers                                                                                                         | Medium | M      | Cleanup       |
| 10 | Extract `historyBuffer` into `history.go`                                                                                                                                          | Low    | S      | Cleanup       |
| 11 | Deduplicate sample→JSON mapping shared by TrendHandler/ExportHandler                                                                                                               | Low    | S      | Cleanup       |
| 12 | Fix TrendHandler 503 message (not-started vs not-enabled cases)                                                                                                                    | Low    | S      | Bug           |
| 13 | Replace flaky `time.Sleep` in SSE tests (`:265`, `:492`) with event-driven waits                                                                                                   | Medium | S      | Bug           |
| 14 | Rename `BenchmarkDashboard_PatchRender` (verified: serves full HTML)                                                                                                               | Low    | S      | Cleanup       |
| 15 | Inline the `maxRequestsInvalid` one-liner in the example                                                                                                                           | Low    | S      | Cleanup       |
| 16 | Add `TestWithMaxSSEConnections_ZeroAllowsUnlimited`                                                                                                                                | Low    | S      | Quality       |
| 17 | Test SSE handler when the probe hasn't started (degraded render)                                                                                                                   | Low    | S      | Quality       |
| 18 | Assert no `style=` attributes in SSE patch content                                                                                                                                 | Low    | S      | Quality       |
| 19 | Distinguish not-started vs shut-down in `ErrPusherNotActive` (or add a second sentinel)                                                                                            | Low    | S      | Feature       |
| 20 | Refresh stamp: use last sample timestamp (observation time), not render time                                                                                                       | Medium | S      | Feature       |
| 21 | Scope the axe `definition-list` tolerance to StatCard nodes (currently whole-rule filter)                                                                                          | Low    | S      | Quality       |
| 22 | Upstream PR to templ-components: StatCard `<dl>` fix (+ goldens); then remove the axe tolerance                                                                                    | Low    | M      | Cleanup       |
| 23 | Example toggles: `DEMO_PUBLIC=1`, `DEMO_BASE_PATH=/status`                                                                                                                         | Low    | S      | Feature       |
| 24 | Document rate-limiter shared-bucket semantics in the README options list                                                                                                           | Low    | S      | Documentation |
| 25 | Update `doc.go`: `Register` + `ErrPusherNotActive` Quick Start examples (b.2)                                                                                                      | Low    | S      | Documentation |
| 26 | CONTRIBUTING.md: browser+screenshot test how-to; `Register` DI path (b.1)                                                                                                          | Low    | S      | Documentation |
| 27 | Audit CONTRIBUTING.md consistency end-to-end (never read this session)                                                                                                             | Low    | S      | Documentation |
| 28 | Run `nix run .#lint` locally as part of the next docs pass (d.6)                                                                                                                   | Low    | S      | Process       |
| 29 | Route-or-record the six unrouted items after g.3 (b.5): CSV fuzz, CSP fuzz, keyboard-nav smoke, metrics-under-CSP test, network-policy probe note, unchecked `injector.Shutdown()` | Low    | S      | Documentation |
| 30 | Screenshot regenerate one-liner docs + caption for the light screenshot (b.3)                                                                                                      | Low    | S      | Documentation |
| 31 | Add "protect probes via network policy" note to README (sweep f23 — currently entombed)                                                                                            | Low    | S      | Documentation |
| 32 | Route 04-41 residual ideas to ROADMAP: `BenchmarkHealthCheck`, self-monitoring test, `do.Package` (b.8)                                                                            | Low    | S      | Documentation |
| 33 | Per-route middleware sets decision spike (sweep f38)                                                                                                                               | Low    | S      | Decision      |
| 34 | Bisectability audit `071c251..HEAD`; record any non-building commits beyond `72783fc`                                                                                              | Medium | M      | Quality       |
| 35 | `WithBasePath` resolution-in-`New()` design spike (kills the ordering footgun)                                                                                                     | Low    | M      | Feature       |
| 36 | Sub-millisecond `WithRetryInterval` validation (500µs silently becomes 0)                                                                                                          | Low    | S      | Feature       |
| 37 | `BenchmarkHealthCheck` benchmark                                                                                                                                                   | Low    | S      | Quality       |
| 38 | Test whether the Dashboard appears in its own health table when registered                                                                                                         | Low    | S      | Quality       |
| 39 | Explore `do.Package` for one-call injection                                                                                                                                        | Low    | M      | Feature       |
| 40 | Test example shutdown ordering (probe → injector → cancel window)                                                                                                                  | Low    | S      | Quality       |
| 41 | Full DI lifecycle integration test: Register → Start → HTTP request → do.Shutdown                                                                                                  | Low    | M      | Quality       |
| 42 | `WithBasePath("/", "")`, `"/a/b"` edge-case tests                                                                                                                                  | Low    | S      | Quality       |
| 43 | Benchmark `renderPatch` retry-field stamping overhead                                                                                                                              | Low    | S      | Quality       |
| 44 | Remove redundant `TestHealthCheckWithContext_InterfaceSatisfied` (compile-time assertions cover it; verified still present at `lifecycle_test.go:141`)                             | Low    | S      | Cleanup       |
| 45 | Check/report the `*do.ShutdownReport` from `defer injector.Shutdown()` in `lifecycle_test.go`                                                                                      | Low    | S      | Cleanup       |
| 46 | Prune AGENTS.md toward the 15 KB budget (b.6)                                                                                                                                      | Low    | M      | Documentation |
| 47 | Fuzz target for the CSV exporter (quote/newline round-trips) — retro f37                                                                                                           | Low    | S      | Quality       |
| 48 | Fuzz target for `RecommendedCSP` (injection attempts) — retro f38                                                                                                                  | Low    | S      | Quality       |
| 49 | Upstream chromedp filing: `[::1]` DevTools binding vs 127.0.0.1 launcher poll (verify first)                                                                                       | Low    | S      | Cleanup       |
| 50 | Next Pareto planning pass over the rebuilt TODO_LIST (start of the next cycle)                                                                                                     | High   | M      | Planning      |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Release policy:** the post-v0.3.1 `[Unreleased]` batch (CSP helper, SSE
   hardening, trend/export endpoints, public mode, OG, histogram, conformance tests)
   is purely additive — semver suggests **v0.4.0**. Cut it now, or batch more and ship
   v0.3.2 later? This gates TODO_LIST item 1 and everything re-headed in the
   CHANGELOG.
2. **Build-tag gating for SSE** (standing BLOCKED, untouched by design): accept the
   `GOEXPERIMENT=jsonv2` requirement as documented status quo, fork go-sse, or
   build-tag-gate the SSE code? Everything else on the board is unblocked; this one
   has been waiting on you for three cycles.
3. **The six unrouted brainstorm items** (b.5 / f.29): CSV-export fuzz target,
   RecommendedCSP fuzz target, keyboard-navigation a11y smoke, metrics-under-strict-CSP
   browser test, "protect probes via network policy" README note, and the unchecked
   `injector.Shutdown()` cleanup in `lifecycle_test.go`. Promote them into TODO_LIST,
   park them in ROADMAP as raw ideas, or leave them entombed in the (now-annotated)
   reports? My default is TODO_LIST for the two fuzz targets and ROADMAP for the rest,
   but the backlog appetite is your call.

---

_Report is a point-in-time snapshot. Section (f) items 1–24 were already harvested
into `TODO_LIST.md` during this session (the report does not duplicate that work);
items 25–50 are HARVEST input for the next pass. Historical reports referenced here
carry inline `done at` dispositions — see docs-health ANNOTATE._
