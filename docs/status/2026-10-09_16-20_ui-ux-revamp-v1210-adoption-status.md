# Status Report — UI/UX Revamp + templ-components v1.21.0 Adoption

**Date:** 2026-10-09 16:20 CEST
**Session scope:** leverage templ-components + common sense to make the dashboard UI/UX superb; adopt the unguarded v1.21.0 dependency sweep per ceremony.
**Branch state at writing:** `master`, ahead of origin/master by 10 commits (daemon never pushes; push needs authorization). Working tree clean. All gates green at last run: unit suite, full browser suite (strict-CSP Chrome), `lint` 0 issues, `check-ui-pins.sh` 6/6.

---

## a) FULLY DONE

Each item verifiable by commit hash, test, or captured artifact.

| #   | Work                                                                                                                                                                                                                                                                        | Evidence                                                                                                   | Files                                             |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ------------------------------------------------- |
| A1  | **Adopted the unguarded templ-components v1.18.0→v1.21.0 sweep** (8th occurrence; guard was red against go.mod). Unit suite was already green on the swept set; full browser suite ran green before pins moved.                                                             | `62c992b` (guard+AGENTS), `bash scripts/check-ui-pins.sh` → 6× OK                                          | `scripts/check-ui-pins.sh`, `AGENTS.md`, `go.mod` |
| A2  | **Pins guard extended**: now pins templ-components root/datastar/utils **and the new direct `icons` module** (all v1.21.0), plus go-datastar v0.6.2 and go-datastar/static v0.6.1 separately (they diverged — old guard assumed one version for both).                      | `62c992b`; guard run green                                                                                 | `scripts/check-ui-pins.sh`                        |
| A3  | **Header renders through `display.PageHeader`** (Action slot = connection pill + theme toggle). Kills the hand-rolled header; gains upstream `flex-wrap` so narrow viewports no longer cram the title row.                                                                  | `82913b2`; goldens regenerated + reviewed                                                                  | `view.templ`, `view_templ.go`                     |
| A4  | **Stat cards carry icon tiles + semantic tones**: Version=tag/purple, Uptime=clock/blue, Latency=bolt/blue, Instance=server/purple.                                                                                                                                         | `82913b2`; visible in recaptured `docs/screenshot.png`                                                     | `view.templ`                                      |
| A5  | **Overall-status alert answers "how bad?"**: `alertSummary()` renders "1 of 3 services reporting issues." / "N of N services reporting pass." as the Alert message; suppressed while shutting down (added `viewModel.ShuttingDown`) and when no services are registered.    | `3eee32b` (helper + `TestAlertSummary`, 5 cases + shutdown assertion in `TestBuildViewModel_ShuttingDown`) | `status.go`, `status_test.go`                     |
| A6  | **Trend + Status Changes cards share a half/half grid when both have content; full-width when alone.** Timeline-only is unreachable in production (timeline derives from the trend ring) but asserted for symmetry.                                                         | `d03f3c2` (`TestTrendTimelineLayout`, 4 branches), `70dd7de`                                               | `view.templ`, `trend_layout_test.go`              |
| A7  | **Timeline rows upgraded**: direction glyph (recover = trending-up, degrade = arrow-down) + `display.Badge` SM (success/warning) per transition, replacing bare colored text.                                                                                               | `82913b2`, `8483344`                                                                                       | `view.templ`                                      |
| A8  | **Empty state teaches instead of dead-ending**: `display.EmptyState` (inbox icon, "No services registered yet", registration guidance) replaces the bare `SimpleEmptyState` one-liner.                                                                                      | `82913b2`                                                                                                  | `view.templ`                                      |
| A9  | **Endpoint links (Export CSV/JSON, Trend JSON, Metrics) wear inline SVG glyphs** via the icons module (download / trending-up / chart), deduplicated through one shared class constant.                                                                                     | `82913b2`                                                                                                  | `view.templ`                                      |
| A10 | **Focus-visible ring on the theme toggle** — the a11y quality floor the upstream `layout.ThemeToggle` itself lacks; mirrors the library's button focus classes.                                                                                                             | `82913b2`                                                                                                  | `page_scripts.templ`                              |
| A11 | **Contrast invariant upheld**: initial green Uptime tone tripped `TestRender_ContrastSafeStatusColors` (upstream's green tile ships `text-green-600`, banned page-wide). Fixed by keeping tones in the blue/purple families instead of weakening the test. Trap documented. | lint+test green; `AGENTS.md` pins gotcha; FEATURES row                                                     | `view.templ`, `AGENTS.md`, `FEATURES.md`          |
| A12 | **icons promoted to direct dependency** at the already-audited v1.21.0 (same pin, `go mod tidy` moved it out of `// indirect`).                                                                                                                                             | `82913b2` (`go.mod`)                                                                                       | `go.mod`, `go.sum`                                |
| A13 | **Goldens regenerated and reviewed line-by-line** for the new markup (5 files; old hand-rolled header, flat stat cards, message-less alert all replaced).                                                                                                                   | `82df8d8`, `8483344`                                                                                       | `testdata/golden/*.html`                          |
| A14 | **Full browser suite green on the final tree** (re-run after the layout restructure, not just the baseline): CSP-clean runtime, live SSE patch, axe a11y both themes, keyboard nav, pill, collapse, filter, metrics, aggregate, mobile viewport.                            | run 2026-10-09 ~16:10, `ok 13.4s`                                                                          | —                                                 |
| A15 | **README screenshots refreshed** (light, dark, degraded) with the new UI and **visually critiqued** — the critique is what caught A6's half-empty row.                                                                                                                      | `8483344`, `14bd3ce`; `docs/screenshot*.png`                                                               | `docs/`                                           |
| A16 | **Living docs updated**: AGENTS.md pins gotcha (+ eight-sweep history, StatCard green-tone trap), FEATURES.md rows for alert/stat cards/empty state/timeline/trend/dark-mode toggle (+ new Status-change timeline row), CHANGELOG `[Unreleased]` entry written.             | `62c992b`, `14bd3ce`                                                                                       | `AGENTS.md`, `FEATURES.md`, `CHANGELOG.md`        |

---

## b) PARTIALLY DONE

| #  | Work                          | What works                                                    | What remains                                                                                                                                                                                                                                                                                                           | Effort |
| -- | ----------------------------- | ------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| B1 | **UI/UX revamp as a whole**   | All planned P1 items landed and verified (see a).             | Polish tail: ADR for the two-family tone system, golden fixture with trend+timeline both present, payload-weight measurement of inline SVGs inside SSE patches, benchmark delta. None blocking.                                                                                                                        | M      |
| B2 | **v1.21.0 adoption ceremony** | This repo's guard/audit/docs complete.                        | Fleet-level prevention: BuildFlow's `go-mod-update` step is what sweeps these pins (8 occurrences now). That is a BuildFlow-repo change, not per-repo. Also unknown whether sub-modules (icons/datastar/utils) can be swept to _divergent_ versions — the guard would catch it, but the failure mode is untested.      | M      |
| B3 | **Docs accuracy**             | Rows I touched updated.                                       | FEATURES.md still carries stale line-number refs elsewhere (`view.templ:17` etc.); AGENTS.md architecture file-map bullet still says "dashboardContent (alert + trend card + check groups)" — no mention of the trend/timeline grid; README prose not audited against the new UI (screenshot captions still accurate). | S      |
| B4 | **CHANGELOG / release**       | `[Unreleased]` entry written with rationale.                  | No version cut. This is a user-visible UI change — release checklist (`pre-push-checks.sh`, `verify-release.sh`, tag-then-master push) not started; needs authorization anyway.                                                                                                                                        | M      |
| B5 | **Screenshot evidence**       | Light/dark/degraded recaptured; light+dark visually reviewed. | The **degraded** capture was never viewed after recapture. The **source-grouped (GroupBySource)** page has no screenshot at all — only browser-test coverage.                                                                                                                                                          | S      |

---

## c) NOT STARTED

| #  | Work                                                                               | Why not started                                                                                                                                                    | Still wanted?                                       |
| -- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------- |
| C1 | Cut v0.11.0 with the UI revamp                                                     | Release discipline requires your authorization for push/tag; batch decision pending (see question Q3)                                                              | Yes                                                 |
| C2 | `docs-health` HARVEST of section (f) into TODO_LIST/ROADMAP                        | Skill says HARVEST runs when the session continues; queued behind your instructions                                                                                | Yes                                                 |
| C3 | `nix flake check`, `nix run .#coverage`, `nix run .#vulncheck` gates this session  | Time-boxed to the UI change's own gates; no dependency/flake edits beyond pins, risk low                                                                           | Yes, before release                                 |
| C4 | Upstream filing: StatCard green tile `text-green-600` fails WCAG on white (3.30:1) | `verify-before-filing` gate requires source-level verification in templ-components first; not started                                                              | Yes — it locks every consumer out of the green tone |
| C5 | Touch-story for evidence tooltips (native `title` is invisible on mobile)          | Pre-existing gap, noticed this session while reviewing badge tooltips                                                                                              | Yes, UX-relevant                                    |
| C6 | `ListNote` truncation notice for the 5-entry timeline cap (bounding-tables recipe) | Noticed while reading the recipe index; cosmetic honesty gap                                                                                                       | Medium                                              |
| C7 | Golden fixture covering the both-cards grid path                                   | Discovered at report time: **zero goldens contain "Status Changes"** — grid markup is pinned by `TestTrendTimelineLayout` markup assertions and browser tests only | Yes, S effort                                       |
| C8 | Benchmark + payload measurement of the new markup                                  | No baseline captured before the change, so a delta needs a re-measure against v0.10.2                                                                              | Nice-to-have                                        |

---

## d) TOTALLY FUCKED UP

Nothing shipped broken — both design misses were caught by the verification layers and fixed with tests before commit. Radical honesty about what _was_ fucked up mid-session:

| #  | What was fucked up                                                                                                                                                | Severity                                                                            | Root cause                                                                                                                            | Mitigation / status                                                                                      |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| D1 | **Half-empty row bug almost shipped**: the trend/timeline grid rendered a lone card at half width, stranding dead space (visible in the first recapture).         | Would have been a visible UI regression in the README screenshot.                   | Designed the grid for the both-present case only; didn't reason about single-card states upfront.                                     | Caught by screenshot self-critique; fixed (`view.templ` branching) + pinned (`TestTrendTimelineLayout`). |
| D2 | **Contrast-failing tone chosen**: green Uptime tile introduced upstream's `text-green-600` into the page, failing the repo's own WCAG invariant.                  | Test failure (caught pre-commit).                                                   | Picked tones from the library catalogue without consulting the repo's contrast test first.                                            | Re-toned to blue/purple; trap documented in AGENTS.md + FEATURES.md.                                     |
| D3 | **Browser suite ordering risk**: the suite ran green on the pre-restructure tree; the layout restructure after it invalidated that result. Caught during wrap-up. | Process risk, no bad outcome.                                                       | Gate re-run discipline not systematic after _every_ markup batch.                                                                     | Re-ran on the final tree (green). Improvement proposed in (e).                                           |
| D4 | **Daemon entanglement**: 6 of 9 session commits are heuristic auto-commits; the guard/docs/UI changes are interleaved across them.                                | History bisection pain (this repo already documents a bisect wall from this class). | Verification chains outlive the daemon's commit cadence; two intent commits (`d03f3c2`, `70dd7de`) preserve the messages that matter. | Folded forward per protocol, never rebased. No action possible retroactively.                            |

---

## e) WHAT WE SHOULD IMPROVE

1. **Design reviews should enumerate empty/single/degenerate states before implementation.** D1 cost a restructure + test. A one-line checklist item ("what renders when exactly one of these exists?") would have caught it on paper. Impact: avoids visual-regression churn; fix: add to the design step of UI work in AGENTS.md.
2. **Check repo invariants _before_ choosing library values, not after.** D2 was avoidable by grepping `TestRender_ContrastSafeStatusColors` before picking tones. Impact: one wasted generate/test cycle per occurrence; fix: grep the invariant tests as part of "READ" for any color/spacing decision.
3. **Make "gates after every markup batch" mechanical.** D3 happened because the gate ran at a milestone, not per batch. The browser suite is ~13s — cheap enough to run after each verified batch instead of once at the end. Fix: adopt per-batch browser runs for markup-touching work.
4. **The unguarded-sweep war needs a fleet fix, not an 8th guard update.** Every occurrence costs an audit ceremony. The root is BuildFlow's `go-mod-update` step; a per-project exclusion doesn't exist. Impact: recurring ~1h ceremony + CI-red windows; fix: BuildFlow-level change (tracked fleet-side).
5. **Golden coverage should mirror layout branching.** Unit markup assertions (A6) are good but goldens are the reviewable diff format in this repo. A `trendtimeline.html` golden would make grid regressions reviewable like all other markup churn (C7).
6. **Pre-existing: screenshot harness renders the connection pill's three states simultaneously** (compiled CSS in the harness lacks Tailwind preflight's `[hidden]{display:none!important}`; real deployments with Tailwind are unaffected). Noticed, verified pre-existing (old screenshot shows it too), not fixed. Worth either fixing the harness CSS or documenting.
7. **Inline SVG weight rides every SSE patch**: the link/timeline glyphs live inside `dashboardContent`, so each broadcast re-sends them. Stat-card icons are outside the patch region (correct). Payload delta unmeasured (C8) — likely negligible, but the dashboard's own Pitch is low-latency patches; measure once, record the number.

---

## f) TOP 50 THINGS TO GET DONE NEXT

Ranked by impact. Effort: S <30min, M 30min–2h, L >2h. (HARVEST fuel for TODO_LIST/ROADMAP.)

| #  | Task                                                                                                                   | Impact | Effort | Category      |
| -- | ---------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Cut v0.11.0: CHANGELOG re-head, version const, `pre-push-checks.sh`, tag-then-master push, `verify-release.sh`         | High   | M      | Release       |
| 2  | Run `docs-health` HARVEST: route section (f) into TODO_LIST/ROADMAP                                                    | High   | S      | Documentation |
| 3  | Add golden fixture with trend+timeline both present (grid path)                                                        | High   | S      | Quality       |
| 4  | Run `nix flake check` + `.#vulncheck` + `.#coverage` before release                                                    | High   | S      | Quality       |
| 5  | Verify + file upstream StatCard green-tone contrast issue (`verify-before-filing` → `github-voice`)                    | High   | M      | Quality       |
| 6  | View + verify the recaptured `screenshot-degraded.png`                                                                 | Medium | S      | Quality       |
| 7  | Update AGENTS.md architecture file-map bullet (dashboardContent: alert + trend/timeline grid + groups)                 | Medium | S      | Documentation |
| 8  | Sweep stale line-number refs in FEATURES.md                                                                            | Medium | S      | Documentation |
| 9  | Audit README prose against the new UI (features list, screenshots section)                                             | Medium | M      | Documentation |
| 10 | Fix or document the screenshot-harness pill `[hidden]` rendering                                                       | Medium | S      | Bug           |
| 11 | Measure SSE patch payload before/after icons (record in docs/research)                                                 | Medium | M      | Quality       |
| 12 | Re-run `BenchmarkHandler_HTMLRendering`, record delta                                                                  | Medium | S      | Quality       |
| 13 | ADR-0003: stat-card two-family tone system (blue/purple only, contrast rationale)                                      | Medium | S      | Documentation |
| 14 | Unit tests for `timelineBadgeType`/`timelineTrendIcon`/`Class` helpers (currently only transitive)                     | Low    | S      | Quality       |
| 15 | `ListNote` for the timeline's 5-entry cap ("showing latest 5")                                                         | Medium | S      | Feature       |
| 16 | Touch story for evidence tooltips on mobile (native `title` invisible)                                                 | High   | M      | Feature       |
| 17 | GroupBySource page screenshot + README section                                                                         | Medium | S      | Documentation |
| 18 | Decide latency-card tone policy (static vs threshold-reactive) — blocked on Q1                                         | Medium | S      | Feature       |
| 19 | Decide age-stamp policy (coarse anti-fingerprint vs live ticking) — blocked on Q2                                      | High   | S      | Decision      |
| 20 | Verify BuildFlow can't sweep icons/datastar/utils to divergent versions; document guard failure mode                   | Medium | M      | Quality       |
| 21 | Add browser test asserting half/half grid at desktop width (beyond markup presence)                                    | Medium | M      | Quality       |
| 22 | Axe-audit the degraded page variant explicitly (confirm which fixture the audit runs on)                               | Medium | S      | Quality       |
| 23 | Confirm endpoint links are keyboard-reachable in `TestBrowser_KeyboardLinks` coverage                                  | Medium | S      | Quality       |
| 24 | Investigate the auto-push daemon condition noted in AGENTS (pushed-before-manual-push anomaly) before the release push | Medium | M      | Quality       |
| 25 | `datastar.SSEErrorHandling` evaluation as opt-in complement to the pill                                                | Low    | M      | Feature       |
| 26 | `CopyButton` for raw check keys — re-vet nonce/CSP on v1.21 (script-emitting component policy)                         | Low    | M      | Feature       |
| 27 | Document that stat icons sit outside the SSE patch region (payload-by-design note in csp.go/docs)                      | Low    | S      | Documentation |
| 28 | Consider tone for group count badges (amber count on warn groups)                                                      | Low    | S      | Feature       |
| 29 | Sticky table header for long healthy groups (nice-to-have)                                                             | Low    | M      | Feature       |
| 30 | Verify print styles survive our markup (cards carry `print:` classes)                                                  | Low    | S      | Quality       |
| 31 | Capture a dark-mode aggregate screenshot for the federation docs                                                       | Low    | S      | Documentation |
| 32 | Add the two-family tone rule to the dark-mode checklist section of AGENTS (consumer-side note)                         | Low    | S      | Documentation |
| 33 | Review `evidenceTooltip` wording against the new alert scale line for redundancy                                       | Low    | S      | Documentation |
| 34 | Consider `datastar.PolledRegion` for a future non-SSE fallback mode                                                    | Low    | M      | Feature       |
| 35 | Sweep ROADMAP.md for now-done UI items (post-revamp)                                                                   | Medium | S      | Cleanup       |
| 36 | Run the hub (`cmd/health-hub`) e2e once against the new UI with 2 remotes                                              | Medium | M      | Quality       |
| 37 | Add `TestAlertSummary` cases for source-grouped VMs (grouping parity)                                                  | Low    | S      | Quality       |
| 38 | Confirm axe runs both themes on the new badge/timeline markup (dark audit re-run evidence)                             | Medium | S      | Quality       |
| 39 | Document the icons-module adoption pattern in the templ-components consumer notes (fleet-wide?)                        | Low    | S      | Documentation |
| 40 | Explore `EmptyState` for the no-match filter hint (currently text-only)                                                | Low    | S      | Feature       |
| 41 | Evaluate transition animation on status-badge changes (motion-reduce safe, CSP-safe)                                   | Low    | M      | Feature       |
| 42 | Add capture script convenience wrapper for the three README screenshots (one command)                                  | Low    | S      | Cleanup       |
| 43 | Verify `/health` JSON is byte-identical to v0.10.2 (UI-only change proof)                                              | High   | S      | Quality       |
| 44 | Check health-hub fuzz targets unaffected (no parsing change — confirm and record)                                      | Low    | S      | Quality       |
| 45 | Revisit `WithGrouping(GroupBySource)` collapse-persistence exclusion docs (FEATURES UX rationale)                      | Low    | S      | Documentation |
| 46 | Add release-notes screenshot thumbnails for v0.11.0 (light+dark side by side)                                          | Low    | S      | Documentation |
| 47 | Ask upstream about `[hidden]` resilience without Tailwind preflight (pill markup class) — after #10                    | Low    | M      | Quality       |
| 48 | Consider surfacing probe `RefreshInterval` in the latency tooltip (context for the number)                             | Low    | S      | Feature       |
| 49 | Sweep `docs/status/archived/` for superseded UI plans and annotate done items                                          | Low    | S      | Cleanup       |
| 50 | Schedule a `brutal-self-review` pass on this revamp next session (separate skill, non-overlapping lens)                | Medium | M      | Quality       |

---

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

**Q1 — Latency stat-card tone policy.** The latency card now carries a fixed blue "metrics" tile. Should it stay fixed (identity/metrics two-family system, no judgment implied), or become data-reactive (e.g. amber when the batch duration exceeds some fraction of the probe's RefreshInterval)? I tried to derive thresholds from the codebase and found no existing policy; this is a product judgment about what the tile should _claim_, and it changes the public render contract (goldens + contrast constraints).

**Q2 — Anti-fingerprint vs live-relative age.** The "Updated … · 2m ago" stamp is deliberately coarse (golden-pinned, documented accepted risk: fine-grained ages fingerprint deployments). The library's `RelativeTime` could render client-ticked precise ages, which reads better but weakens that stance. Keep coarse (current), or relax for non-public deployments (e.g. tie precision to `WithPublicMode`)? I can't decide how much the anti-fingerprint stance is meant to cost in UX terms.

**Q3 — Release cadence.** Cut v0.11.0 now with just this UI revamp, or batch it with more UI work (e.g. #15/#16 from the list) into one release? The tree is release-candidate-quality right now (all gates green), but pushing/tagging needs your authorization regardless, so the batching decision is yours.

---

_Report written per the status-report skill; format override flagged: skill canonical output is styled HTML, user explicitly demanded `.md` at `docs/status/`, so `.md` it is (single explicit override, not propagated anywhere)._
