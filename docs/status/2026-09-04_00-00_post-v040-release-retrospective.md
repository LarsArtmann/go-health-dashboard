# Post-v0.4.0 Release — Retrospective & Status

Date: 2026-09-04 00:00 CEST
Scope: work after the first retrospective
(`2026-09-03_12-32_v03x-cycle-retrospective.md`) up to v0.4.0
(`8f63d85`, proxy-verified). Covers the three user questions, their
execution, and the incidents found on the way.

---

## a) FULLY DONE

| Item                                                      | Evidence                                                                                                                                                                                                                                                                                                |
| --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **v0.4.0 released**                                       | CHANGELOG re-headed with blurb + new **Compatibility** section; `Version = "0.4.0"`; AGENTS status bumped; tagged `v0.4.0`; pushed; proxy `.info` verified → `8f63d85`                                                                                                                                  |
| Fingerprint compat documented (Q2 = accept)               | "Compatibility" paragraph in the 0.4.0 notes: delimiter → length-prefixed change, one spurious change detection if persisted                                                                                                                                                                            |
| go-sse claim **verified false** (Q3)                      | Source: `stream.go:5` imports `encoding/json/v2` unconditionally, zero build tags in the module. Empirical: fresh module, `GOEXPERIMENT= go build github.com/larsartmann/go-sse` fails — "build constraints exclude all Go files in encoding/json/v2". Dual-build without touching go-sse is impossible |
| templ-components v1.12.0 regression **root-caused**       | New LiveRegion busy-script (`tcLiveBusyAttached`) emits `nonce=""` verbatim; source-level: `live_region.templ:66` calls the script unconditionally, line 81 renders the attribute without an empty-guard (ThemeScript guards; this doesn't)                                                             |
| Empirical bisect                                          | Pin v1.11.0 → all three broken CSP tests pass. Minimal Base-only repro rendered identically in both versions (ruled Base out honestly) before locating the real culprit via full-script dump                                                                                                            |
| LiveRegion nonce propagation fix                          | `view.templ:56`: `BaseProps.Nonce: data.DatastarNonce` — correct with both upstream versions, fixes extractor flows                                                                                                                                                                                     |
| Pin v1.11.0                                               | go.mod pinned (all 3 sibling modules) so v0.4.0 ships with green CSP invariants                                                                                                                                                                                                                         |
| Upstream issue #7 filed                                   | templ-components#7 — source lines, minimal repro, rendered HTML, suggested guard, consumer workaround                                                                                                                                                                                                   |
| Upstream issue #6 (from previous segment) confirmed filed | templ-components#6                                                                                                                                                                                                                                                                                      |
| FEATURES count corrected                                  | "154 test/fuzz/benchmark funcs across 19 test files" (was "140+ across 14")                                                                                                                                                                                                                             |
| Gates at release                                          | test -count=1 green, lint 0 issues, fmt clean; tag push confirmed by proxy                                                                                                                                                                                                                              |

## b) PARTIALLY DONE

1. **Q3 itself** — not implemented: the premise ("go-sse supports both")
   is false, so "we should too" has no direct implementation. Escalated
   back with three real options (accept / fork / upstream request). This
   is correct behavior, but the item is open, not done.
2. **templ-components #6 and #7** — filed with verified diagnoses; no
   upstream PRs carrying the fixes yet; the v1.12.0 pin and the axe
   `definition-list` tolerance both stay until those land.
3. **The previous retrospective's 50-item list** — only item 5 (fingerprint
   compat note) and item 4's count fix were touched this segment; the
   other ~46 remain open and are carried forward below.
4. **Release hygiene vs. the concurrent session** — the v0.4.0 release
   notes document the pin, but TODO_LIST was not updated with a v0.4.0
   DONE row and the older completion report still reads as v0.3.1-era.
   Point-in-time docs, minor drift.

## c) NOT STARTED (carried, still open)

- Verify CI in the actual runner: ci.yml browser job green on a push;
  fuzz.yml via workflow_dispatch. Both workflows exist but have never
  been observed executing.
- `nix run .#coverage` baseline + CI floor; `nix run .#vulncheck` after
  the prometheus/common + chromedp dependency additions.
- Version-const vs. git-tag guard (stale-const bug has bitten twice).
- Document the broken-bisect commit `72783fc` in AGENTS.md (confirmed
  non-compiling; pushed history, can't rewrite).
- README drift: dark screenshot embed, `WithDescription`/`WithPublicMode`
  option rows, histogram mention in the Prometheus section.
- AGENTS.md file inventory: csp.go, ratelimit.go, trend.go, metrics.go
  and the new test files are missing from the architecture list.
- Fresh TODO_LIST cycle plan (list is all-DONE; next session needs a plan).
- Decision-record doc for the Q3 finding (the false premise + the three
  real options) so it survives into the next cycle.

## d) TOTALLY FUCKED UP

1. **Repeated the exact bug I had just documented.** The first retrospective
   lists "raw `\n` inside heredoc-built python strings" as a lesson; minutes
   later the 0.4.0 changelog blurb and Compatibility section were written
   with literal `\n` text in the file. Same root cause (backslashes through
   the tool→bash→python layers), same fix needed (chr()-style construction),
   and it happened while writing the very section that documents it.
2. **Forensic detour before empirical bisect.** I spent ~10 tool calls
   diffing base_templ.go / theme_templ.go / datastar dirs across upstream
   versions — all identical — before simply pinning v1.11.0 and rerunning
   (one command, conclusive). Empirical bisect should have been step one;
   source archaeology only after.
3. **Missed the concurrent session's dependency bump until tests broke.**
   Another session bumped templ-components to v1.12.0 (and touched
   go.mod/go.sum/flake.lock + docs) while this session worked. I started
   release prep without re-checking go.mod/fetch state, and only found the
   bump through failing nonce tests. A pre-release `git status` + go.mod
   review would have surfaced it immediately.
4. **Debug artifact nearly shipped.** `csp_debug_dump_test.go` sat in the
   tree during the daemon's snapshot window (one auto-commit carried its
   broken-import intermediate). Deleted before the release commit, but the
   pattern "add debug test → forget it" is one daemon tick away from
   shipping.
5. **Daemon-snapshot pollution continues.** This segment's pin/nonce fixes
   landed as three "chore: auto-commit" messages; the release commit is
   clean, but the pin + fix history is unreadable. Known issue, still
   unsolved, now polluting a release boundary.

## e) WHAT WE SHOULD IMPROVE

1. **Empirical bisect before source archaeology.** A pin-and-run costs one
   command and bounds the problem; reading six versions of templ files
   bounds nothing.
2. **Kill the heredoc class of bugs for good**: never construct file edits
   with backslash sequences in inline python; use chr()-built strings or
   the write tool. Add this as a hard rule in AGENTS.md (it is now
   demonstrably a repeat offense).
3. **Pre-release reconciliation ritual**: `git fetch && git status && git
   diff origin/master -- go.mod go.sum flake.lock` before any tag —
   catches concurrent-session dependency changes that invalidate
   invariants.
4. **Serialise sessions or declare ownership windows** — two agents
   committing to one repo (one bumping a UI dependency, one cutting a
   release) will keep colliding; the collision this time broke three
   security-invariant tests.
5. **UI-dependency bumps must run the browser suite before landing** —
   v1.12.0 broke three render-invariant tests; a plain unit gate was
   insufficient because the concurrent session never ran the browser
   tests.
6. **Mirror templ-components' release-checklist idea** (their
   `docs/release-checklist.md` exists; ours doesn't) — reconcile,
   changelog, version, tag, push, proxy-verify as a checklist file.
7. **Lint+build before every commit** — the previous retrospective said
   it; this segment complied for the release commit but the daemon still
   snapshotted intermediates. Consider disabling the daemon during
   release prep or committing within seconds of every edit.

## f) 50 THINGS TO DO NEXT (carried + new, deduplicated)

Release & history

1. Watch the v0.4.0 CI run end-to-end (browser job included) and fix what
   breaks on the runner.
2. Dispatch fuzz.yml once via workflow_dispatch to validate the nightly.
3. Verify bisectability of 071c251..HEAD; document `72783fc` as a known
   non-building commit in AGENTS.md.
4. Version-const vs. git-tag guard in CI (stale-const bug: twice bitten).
5. Refresh TODO_LIST: v0.4.0 DONE row + harvest a new "Next Up".
6. Write the Q3 decision record (false premise + options) into
   docs/planning/.

Upstream
7. PR the fix for templ-components#7 (guard in liveRegionBusyScript) —
same owner, fastest path to lifting the pin.
8. PR the fix for templ-components#6 (StatCard `<dl>` structure + goldens).
9. After #7 lands: bump templ-components, drop the pin, note it in the
CHANGELOG.
10. After #6 lands: remove the axe `definition-list` tolerance.
11. go-sse: decide and execute — request upstream build-tag/dual-mode
support, fork, or formally accept jsonv2 (document the decision).
12. Keep templ-components bumps blocked on the browser suite (policy from
this incident).

Tests & verification
13. Run `nix run .#coverage`; record baseline; add CI coverage floor.
14. Run `nix run .#vulncheck` (post prometheus/common + chromedp adds).
15. Fuzz target for the CSV exporter.
16. Fuzz target for `RecommendedCSP` (injection attempts).
17. Public-mode leak-scanner test (grep rendered HTML for real service
names).
18. Keyboard-navigation a11y smoke in the browser suite.
19. Browser-test the metrics endpoint under strict CSP.
20. Scope the axe `definition-list` tolerance to specific nodes, not the
whole rule (until #10).
21. Add a test that `Version` matches the latest git tag in CI (same as 4,
implementation side).

Code quality
22. Split dashboard.go (config/options vs lifecycle vs handlers).
23. Extract historyBuffer into history.go.
24. Deduplicate sample→JSON mapping in TrendHandler/ExportHandler.
25. Fix TrendHandler 503 message ("not started" vs "not enabled").
26. Rename `BenchmarkDashboard_PatchRender` honestly.
27. Simplify `maxRequestsInvalid` in the example.

Features & polish
28. Refresh stamp: observation time (last sample At), not render time.
29. Rate-limit response headers: X-RateLimit-Limit/Remaining/Reset.
30. Rate limiter: document shared-bucket semantics in README; optional
per-route buckets.
31. Drain: Retry-After on 503s during the drain window.
32. MaxConnectionLifetime: optional reconnect jitter.
33. `dashboard_pusher_last_tick_seconds` gauge (watchdog observability).
34. Optional opt-in pusher auto-restart hook.
35. `dashboard_build_info{version}` gauge.
36. `dashboard_health_checks_total` counter.
37. Trend JSON `?since=` incremental polling.
38. Export ETag/If-None-Match.
39. `WithTrendWindow(duration)` alternative to sample count.
40. Public mode: redact-JSON option + loud docs that /health JSON stays
verbatim.
41. Example: `DEMO_PUBLIC=1` and `DEMO_BASE_PATH=/status` toggles.

Docs & process
42. Embed `docs/screenshot-dark.png` in the README Dark Mode section.
43. README options snippet: add `WithDescription`/`WithPublicMode` rows.
44. README Prometheus section: histogram mention + scrape snippet.
45. AGENTS.md file inventory: add csp.go, ratelimit.go, trend.go,
metrics.go and new test files.
46. AGENTS.md: record the two new process lessons (empirical bisect
first; no backslash-built heredocs).
47. Create docs/release-checklist.md (reconcile → changelog → version →
tag → push → proxy-verify → CI watch).
48. Adopt a lint+build pre-commit gate (lefthook/pre-commit) to end the
lint-churn commits.
49. Pin golangci-lint and templ versions in CI (currently `latest`).
50. Add a CI concurrency group; decide a Renovate/Dependabot policy that
requires the browser suite for UI-library bumps.

## g) QUESTIONS (cannot be answered from the repo)

1. **Session coordination:** another session was concurrently bumping
   dependencies and running docs passes while this one released. Is that
   session still active, and should releases be serialized behind an
   owner/lock — or is the daemon-committed free-for-all acceptable?
2. **go-sse direction:** given the verified false premise — formally
   accept the jsonv2 requirement (status quo), request build-tag support
   upstream in go-sse, or fork? My recommendation is "request upstream
   support, accept jsonv2 meanwhile", but it touches your sibling-repo
   roadmap.
3. **Upstream fixes (#6/#7):** want me to open PRs against
   templ-components myself (same owner, fastest), or do you/another
   session handle that repo's code changes?
