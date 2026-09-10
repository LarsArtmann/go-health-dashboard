# Status Report — TODO-List Execution Sweep (+ concurrent evidence session)

**Date**: 2026-09-10 02:56 CEST
**Scope**: THIS session only — executing the entire TODO_LIST "Next Up" (5
release-hygiene rows + 6 polish rows), the release-hygiene codification,
the v0.6.1 backfill, the dependabot investigation, and the UI batch
(mobile stacking, script consolidation, contrast pass). No new research
beyond this session's own claims and one live recon of the failing PRs.
**Verdict**: Every actionable TODO row shipped or closed with a reason.
All gates that ran are green on the combined tree — which is the
complication: a SECOND agent session landed an "evidence log" feature
into the same working tree mid-flight, so this report describes a moving
shared state. Three intent commits were eaten by the auto-daemon.

---

## Runtime state at report time

| What                      | State                                                                                                                             | Evidence                                  |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| Local master              | ahead of origin by ~12 commits, **unpushed** (release hygiene + UI batch + docs; pushing not authorized this session)             | `git status`                              |
| Working tree              | my work committed (7dacd53); one untracked file from the OTHER session (`evidence_integration_test.go`, appearing as I type)      | `git status`                              |
| Unit suite                | green, 5.0s, on the combined tree (my UI batch + their evidence refactor)                                                         | `go test .`                               |
| Browser CSP suite         | **PASS 12.4s** on the combined tree (pill, persistence, theme, filter, keyboard, axe, mobile viewport, aggregate)                 | `nix develop -c go test -run TestBrowser` |
| Lint                      | my files clean; ~10 findings remain in the OTHER session's in-flight files (`evidence.go`, `evidence_test.go`, `status.go`)       | `nix run .#lint`                          |
| pre-push-checks.sh        | all green (FEATURES 246/33 synced, version 0.7.0 == v0.7.0, pins intact, changelog lint green)                                    | run at 02:4x                              |
| GitHub Releases           | v0.6.1 backfilled (published, `--latest=false`); v0.7.0 still Latest; gap v0.6.0→v0.7.0 closed                                    | `gh release list`                         |
| Dependabot PRs #13/#12/#4 | all **10/10 green** after `gh pr update-branch` (root cause: stale branches predating the v1.16.0 pins + v0.7.0 tag); none merged | `gh pr checks`                            |
| verify-release.sh         | self-verified green against v0.7.0 (all six checks + clean exit)                                                                  | run twice (second after trap fix)         |
| Standing reds             | none observed in-repo; the evidence session's lint findings are the only open in-tree debt and it is still writing                | `nix run .#lint`                          |

---

## a) FULLY DONE

1. **`scripts/verify-release.sh <version>`** — the mechanical post-push
   tail: tag on origin (peeled commit equality), proxy `Origin.Hash` ==
   tag commit (shape confirmed empirically from the v0.7.0 `.info` —
   the proxy documents the origin hash and ref it indexed), sumdb-verified
   download in a throwaway cache, clean-dir consumer `go get` + build +
   run printing the version (GOEXPERIMENT/GOWORK contract set), GitHub
   Release published-state, CI runs on the release commit, Version-const
   check when HEAD is the release commit. Verified green against v0.7.0
   with clean exit 0.
2. **`scripts/check-changelog.sh` + CI wiring** — exactly one
   `[Unreleased]`, first-section-only, semver-descending sections with a
   duplicate check. Ordering uses an explicit zero-padded key because GNU
   `sort -V` is NOT semver-aware on this machine (probed empirically: it
   ranks `0.1.0-alpha` above `0.1.0`, and a `~`-suffix workaround failed
   the same way before the awk key landed). Verified on the real
   CHANGELOG (green) plus three broken fixtures (all correctly red).
   Wired as the first step of the CI `Build` job and into
   `pre-push-checks.sh`.
3. **Release lessons codified** — AGENTS.md release-discipline gotcha
   (fmt after last `templ generate`; commit-vs-daemon discipline, now
   with three more data points from this session; tag-first push order;
   `verify-release.sh` as the external-state loop) and
   `docs/release-checklist.md` rewritten around them (ordering rule,
   commit discipline, tag-first push, one-command verification,
   poll-loop rule for slow surfaces, `--latest=false` backfill
   convention, refreshed v1.16.0 pin wording).
4. **v0.6.1 GitHub Release backfilled** — notes in the house style from
   its CHANGELOG section, attached to the existing tag,
   `--latest=false` so v0.7.0 keeps the Latest badge; verified in
   `gh release list`.
5. **Dependabot investigation (TODO row) closed** — root-caused on PR
   #13: its branch predated BOTH the v1.16.0 pin ceremony (guard expected
   v1.13.x while go.mod resolved v1.16.0) AND the v0.7.0 tag (const 0.7.0
   vs latest reachable tag 0.6.1). Exactly the "guard tripping on its own
   branch" hypothesis. `gh pr update-branch` fixed it; the same fix was
   applied to the sibling stale reds #12 and #4; all three now 10/10.
   Merging was deliberately NOT done (not in the row's scope).
6. **Mobile row stacking** — design decision first: below the `sm`
   breakpoint (640px) the table stops laying out as a table; the header
   hides and each row becomes a labeled card (Service / Status /
   Details), labels uppercase gray-500/gray-400. Pure Tailwind variants
   on markup the dashboard owns (no stylesheet, no script) so SSE patches
   carry identical classes; the upstream-rendered `<thead>` is hidden via
   an arbitrary variant on the table element (`[&_thead]:max-sm:hidden`);
   the filter keeps winning on mobile via a `:not(.hidden)` guard that
   out-specifies the class toggle (documented as load-bearing in the
   template and pinned by `TestMobileRowStacking_Markup`). Overflow
   containment remains as the fallback under the scroll wrapper.
7. **Page scripts 3→1** — connection-pill wiring, collapse persistence,
   and theme-toggle now share ONE nonce-carried bootstrap
   (`pageBootstrap` in the new `page_scripts.templ`); two inline script
   tags removed per page load. Persistence and pill sections are
   attribute/element-gated so one script serves every configuration;
   behavior is verbatim from the previously-tested scripts. The theme
   toggle button is now rendered by the dashboard itself (identical
   markup, switch semantics, and heroicon paths copied from upstream) so
   its script could join the bootstrap without a fourth pinned
   templ-components module; the pre-paint dark-mode script in `<head>`
   (injected by `layout.Base`) is deliberately untouched — it must run
   before first paint. The dedicated CSP-suite run the TODO demanded:
   **browser suite PASS 12.4s** on the combined tree.
8. **`view.templ` split (condition met)** — the merge added a large JS
   mass, so the ADR-0001 condition genuinely triggered: scripts and pill
   markup moved to `page_scripts.templ`; view.templ 531 → 403 lines.
9. **Contrast pass** — ratios computed for the full palette on white /
   gray-800 / gray-900 / gray-100. Fixes: pill/timeline `green-600 →
   green-700` (3.30 → 5.02 on white), `amber-600 → amber-700` (3.19 →
   5.02), raw keys + dim labels `gray-400/dark-gray-500 →
   gray-500/dark-gray-400` (2.54 → 4.83 on white; 3.04 → 5.78 on dark
   cards), age-dot span simplified, theme-toggle idle gray-400 → gray-500
   (non-text 1.4.11 compliance on our own markup). Dark variants were
   already passing. Locked by `TestRender_ContrastSafeStatusColors` with
   the measured ratios in its header; upstream-owned exceptions
   (CollapsibleSection chevron, Badge/Alert/StatCard palettes)
   documented as out of scope.
10. **TODO_LIST emptied per its own convention** — both tables closed;
    shipped rows recorded in `CHANGELOG.md [Unreleased]`; the two
    no-code closes and one conditional close documented inline
    (dependabot root cause, existing rg tripwire, aria-live gate).
11. **`rg -r` habit guard row closed** — already satisfied
    machine-level by `~/.config/fish/conf.d/01-rg-replace-guard.fish`
    (2026-08-30, deliberate warn-only tripwire; replacement is a
    legitimate feature). My duplicate (a shadowed `rg.fish` function)
    was removed — two guards would have been a split brain.
12. **aria-live filter-count announcer row closed** — the row's own
    acceptance gate ("only if a screen-reader user asks") is unmet; the
    no-match hint already announces via `role="status"`. Reopen on the
    first real request.
13. **FEATURES test-count guard synced** — 246 functions / 33 files
    (mine: +2 render-contract tests; the evidence session's
    evidence_test.go accounts for the rest).

## b) PARTIALLY DONE

1. **The combined tree itself** — all gates ran green on a snapshot, but
   the evidence session was still writing when this report was prepared
   (`evidence_integration_test.go` appeared minutes ago; its lint
   findings remain). The combined state is green-as-measured, not
   frozen; whoever commits last owns re-running the gates.
2. **Contrast pass, live-page half** — the in-repo half is complete and
   tested; the manual "live page" verification still rides the CV
   rollout (the BLOCKED row stands). The axe harness cannot compute
   contrast (stub CSS), which is exactly why the numbers are now locked
   in a unit test instead.
3. **Mobile stacking, visual half** — DOM/behavior is asserted in the
   browser suite, but the harness CSS is a stub (`body { margin: 0 }`),
   so the actual Tailwind rendering of the stacking variants is
   verified by pipeline trust (classes are standard `max-sm:`/arbitrary
   variants, Tailwind ≥3.1 documented), not by pixels. Documented in the
   test header; a one-off real-CLI check would close it with evidence.
4. **CI validation of the new CI step** — the changelog-lint step in
   `ci.yml` is locally tested via the script but has not run in CI
   (nothing is pushed). actionlint also was not run locally (not on
   PATH); the CI hygiene job will cover both on the next push.
5. **Golden-file review** — regenerated for the toggle/stacking/contrast
   changes; the 280-line diff was reviewed in structured summary form
   (class-frequency extraction + spot checks of the toggle swap and a
   stacked row), not line-by-line.
6. **Intent-commit protection** — partially defeated; see d-1/d-2.

## c) NOT STARTED

1. **Pushing master** — ~12 local commits including everything above;
   pushing is explicitly gated on user instruction.
2. **Merging the three green dependabot PRs** — investigated and greened,
   deliberately not merged (out of the row's scope).
3. **Evidence-feature leftovers** — lint findings and doc surface
   (FEATURES rows, CHANGELOG entry, DOMAIN_LANGUAGE terms, possible
   golden drift) belong to the other, still-active session; untouched
   per the no-stomping rule.
4. **test-race + vulncheck on the combined tree** — not run this
   session (unit/browser/lint/gate were; the changes are markup/script/
   docs, but the full battery is the release-checklist standard).
5. **Everything previously deferred by design** — CHANGELOG historical
   audit ([0.1.0-alpha] misplaced bullets), GitHub-Release-from-CHANGELOG
   automation, tag protection, introspection version dogfood test,
   example footer version, coverage floor review, scheduled fuzz matrix,
   browser-startup latency re-measure, ROADMAP v1.0 criteria, pkg.go.dev
   badge, SECURITY.md, Upgrading section, retract runbook, public-mode
   golden fixture, load-test env knobs, RetryInterval×lifetime browser
   test, pin-guard keep sign-off, CV rollout, copy affordance — all
   unchanged from the 2026-09-10 01:39 report's lists.

## d) TOTALLY FUCKED UP

1. **The daemon race ate THREE intent commits — and I had just written
   the rule against it.** (a) The two release scripts landed as
   `d18262e chore: auto-commit 2 changed file(s)` seconds before my
   commit `541d852`, whose message then described files it does not
   contain (they are in the parent). (b) The AGENTS.md +
   release-checklist codification was swept whole into `1ae14d0` — the
   commit that documents "commit immediately vs the auto-daemon" is
   itself an auto-commit. (c) Several more heuristic commits swept UI
   work mid-session. My commit-immediately discipline was applied
   AFTER verification runs (minutes), but the daemon's latency is
   SECONDS — the rule as written was insufficiently paranoid for this
   machine. The knowledge survives in history; the messages are
   cosmetic losses, again.
2. **Four consecutive false-positive test cycles from one blind spot**:
   I wrote HTML-substring assertions knowing the bootstrap script text
   lives in the page, and still tripped over it repeatedly — (a) a JS
   comment containing a literal `<details>` broke the
   only-the-healthy-group-may-render-details count; (b) the word
   "queues" in a comment tripped the public-mode leak scanner for the
   "queue" secret (the scanner worked perfectly — my comments were the
   leak-shaped noise); (c) the persistence opt-in tests matched the
   script's SOURCE strings (`data-collapsible`,
   `data-health-collapse-persist`) instead of rendered attributes;
   (d) the NoDatastarRuntime test matched pill ids mentioned in the
   bootstrap source. Each was caught and fixed properly (attribute-form
   assertions, comment rewording), but the class was predictable from
   minute one: **when a page contains inline scripts, substring tests
   must target rendered markup forms, never bare names.**
3. **`check-changelog.sh` shipped broken twice before shipping green.**
   First version used `sort -rV` and failed on the REAL CHANGELOG
   (`0.1.0-alpha` vs `0.1.0` — this box's sort is not semver-aware in
   either direction). The `~`-suffix fix was derived from the wrong
   mental model of gnulib ordering and failed again. Only after
   empirically probing `sort -V` with the actual inputs did the
   explicit awk key land. Two red runs on a Medium-impact gate before
   it was trustworthy — I should have probed sort's semantics FIRST,
   not after.
4. **Go 101 and templ 101 build breaks**: `TableProps{Class: …}` for a
   promoted embedded field (illegal in a composite literal; needed
   `BaseProps: utils.BaseProps{Class: …}`) and a missing import block
   in the new templ file. One wasted build cycle each, both
   predictable.
5. **My contrast test over-asserted into a guaranteed-fail**: it
   required the raw-key mono class in a fixture whose plain names never
   render that line (no short display). It passed locally against my
   mental fixture and failed on the real one — after I had already
   lectured this file about fixture honesty. Fixed by asserting the
   failing class's absence (a reintroduction guard) + the safe class
   generically.
6. **I demo-ran `rg -r x y` inside the repo working tree** to test the
   existing tripwire. Output-only by design (no files touched), but it
   sprayed replacement garbage ("displax") through my own transcript
   and momentarily looked like corruption. Testing a replace-guard with
   a real replacement in a clean repo was careless theater.
7. **Tool-discipline stumbles**: appended to `grouping_render_test.go`
   via shell heredoc instead of the edit tools (triggering
   stale-read conflicts, then a python patch round); one edit rejected
   for a not-re-read file; the `nix fmt`-after-generate rule I codified
   was honored but only because the CI comment was re-read first.

## e) WHAT WE SHOULD IMPROVE

1. **The daemon needs a protocol, not more discipline.** Its latency is
   seconds; "commit immediately after verification" still loses. Real
   options: (a) commit files the moment they are written, fix up after
   gates; (b) a pause/sentinel mechanism (only the user knows if one
   exists — g-2); (c) session-scoped worktrees so parallel agents never
   share a tree. Three more eaten commits this session say this is now
   the highest-leverage process fix in the repo.
2. **Parallel agent sessions on ONE working tree are a live hazard.**
   The evidence session landed mid-flight into the same checkout; every
   edit I made after the discovery was scoped to avoid their files, and
   the shared seams (view.templ groupRows, status.go signature) flipped
   under me twice. A convention (worktrees per session, a claimed-scope
   file, or "check `git status` for foreign diffs before each batch")
   would make this safe instead of lucky.
3. **HTML-substring tests need a helper**: `htmlWithoutScripts(t, body)`
   (strip script bodies before scanning) + an attribute-form convention
   (`id="x"`, `attr="value"`), applied to the four tests adjusted this
   session. The leak-scanner incident suggests also scanning page
   script SOURCE for fixture-secret words as a unit test, so comment
   wording can never trip the scanner again.
4. **Script gates deserve fixture tests**: `check-changelog.sh`'s
   semver ordering is subtle enough (two wrong implementations before
   the right one) that its fixtures (real changelog, sandwiched, broken
   order, duplicate, pre-release order) should live as testdata and run
   in the pre-push gate, not just in this session's shell history.
5. **The contrast ratios in the test comment should be regenerable**: a
   tiny `scripts/contrast-report` (same WCAG math, committed) would let
   future palette changes re-print the table instead of trusting a
   one-off python run from this session's transcript.
6. **The browser harness's stub CSS is now load-bearing knowledge**: it
   cannot verify stacking, spacing, or contrast. Either commit a minimal
   real stylesheet for the harness (hand-maintained, contract-mirroring)
   or add a header list of "classes the harness cannot see" — currently
   that knowledge lives in scattered comments.
7. **Fixture honesty as a reflex**: the over-specific raw-key assertion
   failed because I did not re-check what the fixture renders before
   pinning it. Rule of thumb: every new substring assertion gets run
   against a render FIRST (or asserted as absence when that is the real
   invariant).
8. **Codify the new gotchas into AGENTS.md** (not done yet — see f-1):
   substring-vs-attribute-form testing, script-source-in-body, the
   sort -V non-semver trap, and the daemon's seconds-scale latency.
9. **Push hygiene**: this session produced ~12 unpushed commits; the
   release report's f-10 ("push the dangling commit with the next
   push") has now compounded into a bigger batch. Local-only green is
   invisible green.
10. **Upstream nicety**: templ-components' ThemeToggle/CollapsibleSection
    idle grays are sub-AA for their roles; a friendly upstream issue
    (after the verify-before-filing ritual) would let the dashboard drop
    its hand-rolled toggle and its documented exception.

## f) NEXT — up to 50 things (brainstorm-tiered, not commitments)

**Immediate, hours:**

1. Fix the ~10 lint findings in `evidence.go` / `evidence_test.go` /
   `status.go` — after confirming with the evidence session (or its
   absence) that the files are stable; then rerun unit + browser gates.
2. Wait for the evidence session to finish, then a combined close-out:
   unit, race, browser, lint 0, `nix flake check`, FEATURES/CHANGELOG
   sync for BOTH changesets.
3. Push master (~12 commits) once the user says go.
4. Merge the three green dependabot PRs (#13, #12, #4) — user's call,
   one command each; dependabot will rebase-or-recreate the rest.
5. actionlint on `ci.yml` (locally via nixpkgs, or let the CI hygiene
   job prove the new changelog step on the next push).
6. Write the AGENTS.md gotchas from e-8 (attribute-form assertions,
   script-source-in-body, sort -V trap, daemon latency).
7. `htmlWithoutScripts` helper + migrate the four adjusted tests (e-3).
8. Fixture-based tests for `check-changelog.sh` (e-4).
9. `scripts/contrast-report` generator (e-5).
10. Refresh README screenshots (light/dark/degraded) — the captured
    images predate the toggle, stacking, and contrast changes.
11. README audit for stale claims: dependency matrix wording, CSP
    section (bootstrap now single-script), any "three inline scripts"
    phrasing.
12. One-off real-Tailwind check of the stacking/contrast classes via a
    pinned CLI (b3), recorded in `docs/research/` — closes the
    pipeline-trust gap with evidence.
13. Run test-race + vulncheck on the combined tree (c-4).
14. CONTRIBUTING.md: mention check-changelog + verify-release + the
    fmt-after-generate order for contributors.

**Engineering, days:**

15. CHANGELOG historical audit: relocate the misplaced `[0.1.0-alpha]`
    bullets (open since the v0.7.0 report c-3).
16. CI: auto-draft the GitHub Release from the CHANGELOG section on tag
    push (removes the manual notes step and its drift risk).
17. Tag protection rule on GitHub (`v*` immutable).
18. Introspection dogfood: assert `/health/introspect` reports the
    Version const in a test.
19. Example footer renders `dashboard.Version`.
20. Coverage floor review (78% → 80+; library-scoped now).
21. Scheduled fuzz matrix (weekly `-fuzztime` job) — evidence's new
    string code is a fresh fuzz target candidate.
22. Browser-suite startup latency: re-measure serialized launches
    (45s announce timeout may shrink again).
23. ROADMAP: v1.0 criteria (API freeze, consumer count, compat policy).
24. README: pkg.go.dev / Go Reference badge.
25. SECURITY.md with a reporting contact.
26. README "Upgrading" section (WithBasePath behavior change).
27. go.mod retract runbook stub (write it before it's needed).
28. Public-mode golden fixture alongside severity/source.
29. Load-test harness env knobs beyond the 20×3 fixture.
30. `WithRetryInterval` × max-connection-lifetime browser test.
31. Dark-mode axe pass (structural a11y in dark; contrast still needs
    real CSS — see b-2/e-6).
32. Named keyboard assertions for the theme toggle + pill in the
    focus-walk test (currently covered indirectly).
33. Upgrade-protocol for page scripts: a short doc mapping each script
    section to its tests (pill → TestBrowser_ConnectionPill, etc.) so
    future edits know their verification surface.
34. Fuzz target for the new evidence summary text (once landed).
35. Benchmark re-baseline after evidence lands (FullHTML numbers move).
36. Consider `data-tc-dir`-style stability: confirm hand-rolled toggle
    keeps parity if upstream ThemeToggle changes (add an upstream-diff
    note to the pin ceremony).
37. docs/DOMAIN_LANGUAGE.md: add bootstrap/stacking/contrast terms and
    the page_scripts split (after evidence lands, one pass).
38. Archiving: move the two 09-10 reports + this one to
    `docs/status/archived/` when superseded.
39. docs-health HARVEST pass over this report (dogfooding the rule).
40. Decide and document the 0.x Latest-vs-prerelease convention fully
    (only the backfill half is written down).

**Bigger bets:**

41. Daemon protocol: pause sentinel or release-mode (with g-2's answer).
42. Parallel-session convention: per-session worktrees or a
    claimed-scope handshake (with g-1's answer).
43. Real-CSS browser harness (minimal true stylesheet or an
    npx-free compiled fixture) — upgrades axe contrast, stacking, and
    spacing verification from trust to proof.
44. Evidence + UI batch release: cut v0.8.0 via the release checklist
    (verify-release.sh now makes the tail mechanical) when the user
    says ship.
45. Upstream templ-components issue for AA idle grays (after
    verify-before-filing).
46. templ-components dependency ceremony dry-run: rehearse a bump end-
    to-end (pins → browser suite → guard update) so the next real
    sweep is routine.
47. Golden-render public-mode + dark-mode fixtures.
48. Load-test the aggregate path with evidence enabled (new allocation
    surface on the broadcast path).
49. Explore removing the GOEXPERIMENT=jsonv2 requirement via build-tag
    gating (the standing BLOCKED row — only if the user picks "gate").
50. Celebrate properly: TODO_LIST "Next Up" is EMPTY for the first
    time — the backlog now contains only blocked rows and ideas.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **The concurrent "evidence log" session — yours? Still running? And
   who owns its tail?** It landed mid-flight into this tree (status.go,
   pusher.go, handlers.go, view.templ, evidence.go, two test files, at
   least four daemon commits), and it is STILL writing. I deliberately
   did not touch its files; its lint findings and its doc surface
   (FEATURES/CHANGELOG/goldens) are open. Do you want me to adopt and
   finish it (gates, lint, docs, CHANGELOG entry), or wait for that
   session to close itself? And should parallel sessions use separate
   worktrees from now on?
2. **Can the auto-commit daemon be paused, paced, or taught intent?**
   Three intent commits were eaten this session even WITH
   commit-immediately discipline — its latency is seconds, my
   verification runs are minutes. If a pause/sentinel/release-mode
   exists, release sessions should use it as a rule; if not, I will
   switch to "commit the moment files compile, fix up after gates" —
   but that trades daemon-swept messages for possibly-unverified
   commits, which I would rather not make standard practice without
   your sign-off.
3. **Push and merge authorization**: ~12 local commits (release
   hygiene tooling, UI batch, docs closeout) plus three green
   dependabot PRs (#13 client_model bump, #12 install-nix-action bump,
   #4 golangci-lint-action bump) are waiting. Push master now, and
   merge the PRs (all 10/10 green) — or hold for the evidence session
   to land and ship everything as one reviewed batch?

---

## Self-review appendix (the questions, answered straight)

- **What did you forget?** To probe `sort -V`'s semantics before
  building on it; to predict that inline-script source text lives in
  every HTML-substring assertion's haystack; to check the fixture's
  rendered output before pinning a class assertion; to run actionlint
  and test-race/vulncheck; and that "commit immediately" is not fast
  enough for this daemon.
- **What's stupid that we do anyway?** Two agent sessions sharing one
  checkout while an unattended daemon commits every few seconds — any
  one of the three actors can invalidate the other two's invariants at
  any moment. This session worked because I froze my surface after the
  collision, not because the setup is safe.
- **Did I lie to you?** No. Every green claim maps to a command run in
  this session; the deliberately-not-done items (push, merges, evidence
  leftovers) are reported as such with reasons; the browser suite's
  blind spots (real CSS) are stated in b-3 rather than papered over.
- **Ghost systems?** One near-miss removed: my redundant `rg.fish`
  guard, deleted after the pre-existing conf.d tripwire was found. The
  debug throwaway test file was removed immediately after use.
- **Split brains?** Two live ones, both named: the rg guard (resolved
  by deletion) and the shared working tree (unresolved, g-1). Docs are
  consistent: CHANGELOG/TODO_LIST/FEATURES/AGENTS/checklist all agree
  at commit 7dacd53.
- **Scope creep?** The dependabot fix was extended from #13 to #12/#4
  (same root cause, same one-command fix, zero risk) — justified; PR
  merging resisted. The theme toggle hand-roll was a consequence of the
  requested script merge, not a redesign.
- **Tests?** Two new contract tests (mobile stacking markup, contrast
  colors), four tests re-targeted from bare substrings to rendered
  forms, goldens regenerated and reviewed. The suite's honest limits
  (stub CSS) are documented, not hidden.

**Session log**: `docs/status/2026-09-10_01-39_v0.7.0-release-session.md`
(predecessor) → this report. Fully-executed reports move to `archived/`
when superseded.
