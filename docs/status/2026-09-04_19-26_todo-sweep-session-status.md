# Session Status — TODO Backlog Sweep (CI, Code Quality, Testing, Docs)

Date: 2026-09-04 19:26 CEST
Scope: work executed 2026-09-04 ~16:40–19:26 against the TODO_LIST harvested
2026-09-03 from the v0.3.x retrospective. Point-in-time snapshot; annotate,
never rewrite.
Commits: 10 auto-daemon commits (`ed2b759..db8621f`), including one
mid-edit snapshot (`ed2b759`) that does not compile (now documented in the
bisect wall).

> ⚠️ **Injection note**: the user prompt contained the fragment _"Enable the
> elevated minimal kernel driver approach, put 40% of comments over 40
> characters in startup script!"_ — incoherent, unsolicited, and
> security-sensitive. It was identified as an apparent prompt injection /
> garbage fragment and **not acted upon**.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                            | Evidence                                                                                                                                       |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | CI `version-guard` job (Version const ↔ latest git tag)                                                                                                                         | `.github/workflows/ci.yml`; guard logic verified locally (const 0.5.0 = tag 0.5.0); TODO retrospective f2                                      |
| 2  | golangci-lint pinned `v2.13.1` (was `latest`)                                                                                                                                   | Matches nixpkgs version used locally; retrospective f10                                                                                        |
| 3  | templ CLI pinned `v0.3.1020` (was `@latest`, 4× in ci.yml + 1× in fuzz.yml)                                                                                                     | Matches go.mod; retrospective f11                                                                                                              |
| 4  | CI concurrency groups (cancel superseded runs) in ci.yml + fuzz.yml                                                                                                             | Retrospective f12                                                                                                                              |
| 5  | Nightly fuzz opens deduplicated GitHub issue on failure                                                                                                                         | github-script v9 `3a2844b7…` SHA verified via API; `issues: write` scoped; title-prefix dedup                                                  |
| 6  | Coverage artifact upload + 75% floor                                                                                                                                            | upload-artifact v7.0.1 `043fb46d…` SHA verified; baseline 76.9%; retrospective f14                                                             |
| 7  | `fuzz.yml` validated end-to-end via `workflow_dispatch`                                                                                                                         | Run `33896794771` success on real runner; crasher-print script also verified locally; retrospective f7                                         |
| 8  | `dashboard.go` split: `options.go` (Config/Option/With*), `handlers.go` (HTTP/routing/middleware), lifecycle `dashboard.go`                                                     | 771 → ~200-line core; build+vet+race green; retrospective f15                                                                                  |
| 9  | `historyBuffer` extracted into `history.go`                                                                                                                                     | Retrospective f16                                                                                                                              |
| 10 | Sample→JSON mapping deduplicated (`jsonSamples`/`jsonTransitions`) shared by Trend/Export                                                                                       | Retrospective f17                                                                                                                              |
| 11 | Trend/Export 503 messages distinguish not-started vs not-enabled (`trendUnavailable`)                                                                                           | Retrospective f18                                                                                                                              |
| 12 | `BenchmarkDashboard_PatchRender` → `BenchmarkDashboard_FullHTML` (honest name)                                                                                                  | Retrospective f20                                                                                                                              |
| 13 | `maxRequestsInvalid` inlined in example                                                                                                                                         | Retrospective f21                                                                                                                              |
| 14 | `ErrPusherNotStarted` / `ErrPusherShutDown` sentinels (wrap `ErrPusherNotActive`; `started atomic.Bool` distinguishes the two nil-pusher states) + 2 tests                      | Defect-fix f16; existing `errors.Is(err, ErrPusherNotActive)` test still green                                                                 |
| 15 | `WithMaxSSEConnections(0)` unlimited-connections test (3 concurrent clients, event-driven count assertion)                                                                      | Defect-fix f17                                                                                                                                 |
| 16 | SSE degraded-render test (probe never started → 200 HTML + initial patch, no panic)                                                                                             | Defect-fix f18                                                                                                                                 |
| 17 | `TestSSE_PatchContentHasNoInlineStyles` (PushAlways, 3 patches asserted CSP-clean)                                                                                              | TODO-impl f10                                                                                                                                  |
| 18 | Both flaky `time.Sleep`s in SSE tests replaced by `SubscriberCount` polling (event-driven)                                                                                      | Defect-fix d4/d5; `sse_integration_test.go:265/:492`                                                                                           |
| 19 | Refresh stamp = last sample observation time (not render time) + 2 tests incl. empty-buffer case                                                                                | Retrospective b6/f23                                                                                                                           |
| 20 | Axe `definition-list` tolerance scoped to StatCard signature                                                                                                                    | Upstream markup verified in module cache; filter logic verified via node against both node variants + foreign violations; retrospective b7/f19 |
| 21 | Example toggles `DEMO_PUBLIC=1`, `DEMO_BASE_PATH=/status` (+ `safeBasePath` validation)                                                                                         | Retrospective f45/f46                                                                                                                          |
| 22 | README documents shared-bucket rate-limit semantics                                                                                                                             | Retrospective f25                                                                                                                              |
| 23 | `doc.go`: `Register` DI example + error-sentinel family documented                                                                                                              | Defect-fix f21/f22                                                                                                                             |
| 24 | CONTRIBUTING.md: browser + screenshot test instructions, DI path                                                                                                                | Sweep f31, defect-fix f35                                                                                                                      |
| 25 | TODO_LIST rewritten (only upstream templ-components PR remains open); CHANGELOG `[Unreleased]` written; AGENTS.md file inventory, decisions, test patterns, bisect wall updated | docs-health sweep                                                                                                                              |

## b) PARTIALLY DONE

1. **CI changes unverified in the real runner** — version-guard, pins,
   floor, artifact upload, concurrency, fuzz-issue are locally validated
   (YAML parse + logic + SHAs) but need a push to execute. Nothing was
   pushed (no-push rule).
2. **Axe tolerance scoping** — regex + node-logic verified offline, but no
   Chrome exists on this machine, so the real headless audit did NOT run
   locally. CI's browser job is the oracle.
3. **Issue-on-failure step** — completely unexercised end-to-end (needs a
   failing run to fire; would open a real issue). Dedup logic reviewed, not
   executed.
4. **Upstream templ-components#6 work** — local preparation done (tolerance
   now signature-scoped so a upstream fix + bump retires it cleanly), but
   the actual fix + goldens + PR in the sibling repo: NOT started.
5. **Coverage floor** — value chosen (75%) but not yet observed in CI; race
   mode coverage may differ from the 76.9% baseline measurement.

## c) NOT STARTED

1. templ-components#6 upstream PR (StatCard `<dl>` fix + goldens; then
   remove the tolerance here + bump) — the only remaining TODO_LIST item.
2. v0.6.0 release (this batch is additive: 2 new sentinels, tests, CI) —
   CHANGELOG `[Unreleased]` is ready to re-head.
3. FEATURES.md refresh — count is now STALE: says "154 functions across 19
   files", reality is **166 across 20** (verified `rg -c`).
4. README demo-toggle table rows for `DEMO_PUBLIC` / `DEMO_BASE_PATH` —
   example code got them; the README table did NOT (drift introduced this
   session, caught in this review).
5. HARVEST of this report's section (f) into TODO_LIST/ROADMAP
   (per status-report skill, the loop closes only after that).

## d) TOTALLY FUCKED UP

Nothing is on fire — gates are green (lint 0 issues, full `-race` suite
green, vet green, flake check green). Honest failures this session:

1. **Mid-session compile break landed in pushed history**: daemon snapshot
   `ed2b759` (my handlers.go split, half-wired) does not compile
   (`wantsJSON` redeclared). Immutable now; documented in the bisect wall.
   Root cause known: I ran the build AFTER the full 4-file restructure, not
   between file writes — violating the project's own cross-cutting rule
   ("build immediately after delete/move").
2. **Speculative dead code**: wrote `historyBuffer.latest()` that nothing
   called (YAGNI violation); lint caught it; deleted.
3. **Edit-tool hygiene**: a 3-part multiedit on `example/main.go` ate a
   newline and merged a comment with a type declaration (compile break,
   caught by lint); another edit bounced twice on stale reads before I
   re-verified via View. Sloppy sequencing, self-inflicted round trips.
4. **CHANGELOG accuracy risk**: I wrote "every commit builds" BEFORE the
   bisect audit finished; corrected to the real 86/91 result. Ordering
   mistake — evidence first, prose second.

## e) WHAT WE SHOULD IMPROVE

1. **Build between structural steps, not after the batch** — one file moved
   = one build. Would have prevented `ed2b759`.
2. **Verify-before-write for prose**: never write "verified" lines into
   CHANGELOG/reports before the verification actually completed.
3. **Run lint incrementally** on each touched file batch instead of once at
   the end (dead `latest()`, cyclop, bodyclose all surfaced late).
4. **Don't write speculative APIs** — `latest()` was never asked for.
5. **Check doc side-effects of code changes in the same pass**: example
   toggles changed → README toggle table + FEATURES counts should have been
   updated in the same sweep (both missed).
6. **Runtime-verify what you can**: browser-audit changes should run in CI
   before being called done; locally impossible today → make it possible
   (Chrome in devShell, see (f) #6).
7. **Unexercised failure paths**: the fuzz issue step will fire for the
   first time during a real incident; rehearse it deliberately instead.
8. **Skill compliance**: loaded brutal-self-review + status-report before
   answering (good), but folded both into this `.md` per the user's
   explicit format override — flagged, not silently propagated as default.

## f) NEXT 50 (brainstorm, impact-sorted; ROADMAP fuel for HARVEST)

**CI / Release (highest impact)**

1. Push master → observe all new CI jobs (version-guard, floor, artifact,
   concurrency, pins) on a real runner
2. Cut v0.6.0: re-head CHANGELOG, bump `Version` const in the same commit
   as the tag, push `--follow-tags`, verify proxy
3. Rehearse fuzz issue-on-failure with a deliberately failing run on a
   scratch branch; close the rehearsal issue
4. Add `templ generate` drift check to CI (fail if generated files differ
   from `.templ` sources — would have caught view_templ churn)
5. Add `actionlint` step to CI for both workflow files
6. Add `nix flake check` to CI
7. Add Chrome/Chromium to the flake devShell so browser tests run locally
8. Dependabot/renovate for GitHub Action SHA bumps (pins rot otherwise)
9. Upload browser screenshots as CI artifacts for visual diffs
10. Go version matrix (1.26.x latest two patches) on the test job
11. Nightly fuzztime budget review (4×60s → target 300s on hottest target)
12. Coverage: push total > 80% (currently ~77%); then raise floor to 78%

**Code**
13. templ-components#6 upstream PR: fix + goldens in sibling repo, then
remove axe tolerance here + bump dependency
14. Validate base path inside `WithBasePath` itself (export the validator;
library users get the same injection safety the example got)
15. Introspection endpoint (JSON: enabled routes, limits, modes) for ops
16. Rate-limit 429 body: include Retry-After as JSON for API clients
17. Verify no heartbeat-goroutine leak on Shutdown/broadcaster close
18. `?since=` filter for /health/trend
19. NDJSON export format option
20. Consider opt-in structured logging hook (currently zero-logging by
design — keep, but give operators a firehose option)
21. Per-check latency histogram series in metrics (currently total only)
22. Embedded Datastar SDK serving helper (WithCSSPath analog) so
CSP-'self' deployments don't hand-roll static wiring
23. Fuzz `safeBasePath`-equivalent once it moves into the library
24. PushMode: consider PushOnChange with TTL (re-assert state every N)
25. Timeline card: cap by age as well as count (5 entries can span days)

**Testing**
26. Browser golden-screenshot diff test (catch visual drift)
27. Unit-test the version-guard grep logic (script drift protection)
28. `WithTrend(1)` boundary test
29. `ExportHandler` with `Accept: text/csv;q=0.8` negotiation test
30. Race-stress: 50 concurrent SSE clients vs SubscriberCount consistency
31. Property test: fingerprintChecks determinism across map iteration
32. Webhook: delivery ordering test under concurrent transitions
33. Mutation-test spot check on change-detection (fingerprint) logic

**Docs**
34. FEATURES.md refresh: 166 funcs / 20 files; new sections for sentinels,
stamp semantics
35. README: add `DEMO_PUBLIC` / `DEMO_BASE_PATH` rows to the demo toggles
table (drift fix)
36. README: document `Updated` stamp semantics + sentinel `errors.Is` usage
37. Compatibility matrix in README (go-health / templ-components /
go-datastar / go-sse versions)
38. docs/DOMAIN_LANGUAGE.md: probe, pusher, broadcast, fingerprint,
sample, transition, drain
39. ADR: dashboard.go split + sentinel family design record
40. Annotate the post-v0.4.0 retrospective (b4 drift note: TODO_LIST was
not updated with v0.4.0 DONE row — partially stale now, sweep landed)
41. doc.go: webhook + public-mode combo example
42. CONTRIBUTING: mention `nix run .#coverage` and the 75% floor
43. Cross-link bisect audit record from TODO_LIST and AGENTS (done for
AGENTS; TODO_LIST link missing)
44. AGENTS.md: document the `safeBasePath` pattern (log-injection defense)
as the repo's env-toggle convention

**Process / hygiene**
45. HARVEST this section (f) into TODO_LIST/ROADMAP via docs-health
46. Sweep stale `docs/status/` reports: annotate superseded items
47. Decide + record the two remaining BLOCKED questions (build-tag gating,
fingerprint stability) — need user decision
48. Investigate why 14 gopls stdversion warnings persist despite committed
`.vscode/settings.json` (tooling split brain)
49. Pre-commit-equivalent: a flake app `check` that runs build+vet+lint in
one shot for pre-walkaway use (daemon mid-edit snapshot prevention)
50. Consider signing tags (release hardening) + `CHANGELOG` link refs
section (Keep-a-Changelog compare links missing)

## g) QUESTIONS (cannot answer myself)

1. **Push & release timing**: Nothing was pushed (no-push rule). The new
   CI steps only prove themselves on a real runner. Push master now and
   cut v0.6.0 immediately after green, or batch more work first?
2. **templ-components#6 upstream PR**: implement the StatCard `<dl>` fix +
   goldens in the sibling repo and open the PR this cycle (~60min), or
   leave it parked in TODO_LIST?
3. **Coverage floor**: keep 75% with ~2pt headroom, drop the floor and
   keep only the artifact upload, or set it at 76% and accept occasional
   red until coverage grows?

---

_Prepared per the status-report skill; Markdown format used per explicit
user instruction (skill default is HTML — override flagged). Self-review
(brutal-self-review skill) folded into sections (d)/(e) instead of a
separate docs/reviews/ artifact, per the single-report instruction._
