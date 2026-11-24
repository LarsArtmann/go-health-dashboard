# Status Report — Tooltip Isolation Fix (hover-one-all-open bug)

**2026-11-24 19:27 CET · session-scoped report · HEAD `d52da5a`**

**Scope line — what this report VERIFIED** (all commands run inside the flake
devShell this session): full test suite incl. the serialized browser suite
(`nix develop -c go test -count=1 .` → `ok … 24.249s`), `nix run .#lint`
(0 issues), `scripts/check-changelog.sh` (all green), `scripts/check-templ-stamps.sh`
(OK, v0.3.1100 == go.mod pin), the bug REPRODUCED pre-fix in a real headless
Chromium under two CSS modes (Play-CDN-style and locally compiled Tailwind
v4.3.3) via scratch chromedp runs (deleted after diagnosis), the regression
test's discrimination proven by mutation (unnamed-class markup → test fails
with "tooltips are not isolated" on hover AND focus), and the golden
regeneration diff reviewed by hand (3 files, class strings only).

**Scope line — what this report did NOT verify**: real-desktop hover under
genuine `@media (hover: hover)`-wrapped Tailwind v4 CSS (headless Chrome
reports no hover-capable pointer, so the final test mirrors the matching
semantics without the media wrapper — see Honest Gaps #1); any pixel-level /
screenshot rendering; remote CI state and push state (nothing was pushed and
no release battery was run — no release was cut); which consumer surface the
user saw the bug on (never established — see Questions); the health-hub
NixOS deployment's CSS story (not inspected — out of the session's scope);
fleet-wide consumer CSS staleness (explicitly not researched per your
instruction).

**Git-state caveat**: the fix core (view.templ, view_templ.go, goldens) was
captured by the auto-commit daemon at `eec443f`→`19c8ea0`→`d52da5a` mid-session;
the docs and test-code edits (AGENTS.md, CHANGELOG.md, browser_test.go,
scratch deletion) sat UNCOMMITTED in the working tree at report time (the
daemon will pick them up; this harness forbids manual commits without your
explicit request).

---

## The bug, in one paragraph

Hovering (or keyboard-focusing) ONE badge tooltip opened EVERY tooltip in the
healthy services section. Root cause: the disclosure flipped on Tailwind's
UNNAMED `group-hover:block` / `group-focus-within:block` variants, which
match any `.group` ancestor — and `display.CollapsibleSection` (the healthy
group's container, view.templ:258) renders `<details class="group rounded-lg">`
(needed for its `group-open:rotate-180` chevron), an ancestor of every
healthy row. Fix: NAMED group scoping — `group/badge` wrapper +
`group-hover/badge:block` + `group-focus-within/badge:block` tooltip
(view.templ:408-422), which binds the flip to the badge's own wrapper only.

---

## a) FULLY DONE

1. **Diagnosis, empirically grounded** — reproduced the exact symptom in
   headless Chromium: hovering badge 1 inside the healthy `<details.group>`
   opened 2 of 3 tooltips (Play-CDN CSS) and focusing opened 2 of 3 even
   under compiled Tailwind v4 CSS. Not a guess: DOM probes with computed
   `display` values.
2. **The fix** — named `group/badge` on `disclosureFocusClass`,
   `group-hover/badge:block group-focus-within/badge:block` on
   `disclosureTooltipClass`; doc comments record WHY (the CollapsibleSection
   ancestor hazard) in view.templ and view_templ.go (pinned generator,
   generate → fmt order respected).
3. **Regression test with teeth** — `TestBrowser_TooltipDisclosureIsolation`
   (browser_test.go): hovers badge 1 and focuses badge 2 INSIDE the section,
   asserts exactly one visible tooltip each time, asserts it is the target
   badge's own text, guards against fixture drift (fails if the badge isn't
   inside a `details`). Mutation-checked: reverting to unnamed classes fails
   it with the user's exact symptom, on both the hover and focus legs.
4. **Existing contract test updated honestly** —
   `TestBrowser_EvidenceDisclosureFocus` now asserts the
   `group-focus-within/badge:block` class (its doc comment updated too).
5. **Golden files regenerated** — 3 files (`zeroproven`, `sourceproven`,
   `publicmode`) via the sanctioned `-golden-update`; diff reviewed: class
   strings only, section `<details>` keeps its unnamed `group`.
6. **Gates green** — full suite incl. 23-test serialized browser suite,
   golangci-lint 0 issues (fixed my own nlreturn + tagliatelle violations),
   changelog lint, templ stamp guard.
7. **Knowledge wired where the fleet expects it** — CHANGELOG `[Unreleased]`
   Fixed entry (with the consumer-CSS-rebuild migration note), AGENTS.md
   design-decision sentence + a full Gotchas bullet recording the
   named-group doctrine and the headless `hover:hover` caveat.
8. **Scratch artifacts cleaned** — `scratch_tooltip_test.go` trashed after
   diagnosis; no compensating rules added to the shared `browserStaticCSS`
   harness (issue-#31 doctrine respected: the isolation CSS is test-local).

## b) PARTIALLY DONE

1. **Verification of the hover leg under REAL Tailwind v4 output** — I
   inspected the compiled rules (`grep` of a v4.3.3 build: both variants
   compile correctly scoped) and the regression test mirrors their matching
   semantics, but headless Chrome cannot execute the `@media (hover: hover)`
   -wrapped hover rule. The FOCUS leg runs under true compiled v4 CSS in the
   test; the HOVER leg runs under a media-less mirror. Tailwind's own
   emission is the remaining link in the chain — verified by inspection, not
   execution.
2. **The release decision (rule 7)** — the fix is consumer-facing and sits in
   `[Unreleased]` next to the large OIDC entry with NO cut planned. This is
   the exact shape of the 2026-11-11 gap that rule 7 was written to close.
   The changelog entry exists; the cut decision does not.
3. **Consumer migration story** — the CHANGELOG says "consumers must rebuild
   CSS" but there is no concrete check they can run (a `tc doctor` /
   library-classes.txt refresh instruction) and no fleet sweep to find who is
   now hover-less with stale CSS.
4. **War-story documentation** — the gotcha lives in AGENTS.md (complete) but
   `docs/gotchas.md` did not get the narrative entry the fleet pattern uses
   for debugging stories (the three-round repro journey incl. the two false
   leads is written nowhere).

## c) NOT STARTED (this session, correctly out of scope or deferred)

1. Any release action: pre-push-checks.sh, remote CI verification, tag, cut.
2. Fleet sweep of sibling repos for the same unnamed-`group-*`-inside-
   CollapsibleSection pattern (you forbade unrelated research).
3. Upstream templ-components report about the unnamed-`group` footgun
   (verify-before-filing gate not even started).
4. Visual verification (screenshot of the tooltip open state, dark + light).
5. TODO_LIST/ROADMAP harvest of section (f) below.

## d) TOTALLY FUCKED UP

Nothing shipped is broken — but three things in the PROCESS deserve the
harsh label:

1. **A daemon commit captured the mutation experiment.** `eec443f` holds the
   mutated (unnamed) view_templ.go — the sed-back state of the mutation
   check lived in the tree long enough for the auto-daemon to commit it
   (immutable history). `19c8ea0` superseded it with the fix; `git log -S`
   confirms HEAD is clean and nothing was pushed. This is precisely the
   documented mutation-check-races-the-daemon trap — I ran the check in a
   daemon-live tree without the probe-the-working-tree-first discipline
   (I did `rg`-verify before testing, but the daemon snapshot still fired
   between operations).
2. **Two false leads burned ~6 tool calls before empirical reproduction.**
   I theorized about Play-CDN semantics and canonical v4 rules (both turned
   out CORRECT and irrelevant) instead of reproducing against the user's
   actual DOM surface first. The first real-page probe hit the wrong target
   (info glyph OUTSIDE the section, off-viewport coordinates, no
   scrollIntoView) and produced a misleading 0-of-3 "hover doesn't work at
   all" reading. If I had trusted that first probe, I'd have wrongly closed
   the bug. Lesson applied mid-session (hover-chain instrumentation), lesson
   now recorded here.
3. **Sloppy scratch-test craftsmanship** — one written file contained a
   literal placeholder syntax error; three compile-fail cycles against
   chromedp v0.21 APIs (`MouseMoveXY`/`MousePosition`/`NewRemoteAllocator`,
   async-IIFE eval) that the AGENTS.md migration notes — IN THIS SESSION'S
   CONTEXT — should have prevented. Plus one `rg -rn` invocation that hit
   the host's replace-trap hook ("third strike of this exact trap on this
   host" — a documented repeat offense).

## e) WHAT WE SHOULD IMPROVE

1. **Reproduce-first doctrine for UI bugs**: target the EXACT DOM surface the
   user reported (badge inside the section) before building any CSS-theory
   matrix. Nearest-equivalent fixtures (info glyph, static page) gave two
   false verdicts.
2. **Read migration notes before writing against a migrated API**: chromedp
   v0.21 idioms are documented in AGENTS.md and were violated three times.
3. **Mutation checks must be daemon-proof by construction**: probe the tree
   (`rg`) immediately before AND after the test run, and prefer mutating a
   throwaway copy of the tree over the live one when feasible.
4. **The named-group doctrine deserves a mechanical guard** — a lint/grep
   rule banning unnamed `group-hover:`/`group-focus-within:` in `*.templ`
   would have caught this class at CI time. Doctrine in prose decays; gates
   don't.
5. **Hover-leg verification gap** (Honest Gap #1): decide whether an
   env-guarded real-CSS hover capture (Tailwind CLI at test time) is worth
   the CI cost.
6. **Ask for the reproduction surface earlier when it is unobservable**: one
   question ("which page did you hover?") could have skipped the Play-CDN
   fixture round entirely. Autonomy doctrine says don't ask — but
   unobservable environment facts are the legitimate exception.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

_(a brainstorm, not a commitment list — most items are ROADMAP fuel; the
first block is the real priority.)_

**Release & verification (do these first)**

1. Decide the cut: patch release for this fix vs riding the next minor with
   the OIDC work (closes the rule-7 gap).
2. When cutting: `bash scripts/pre-push-checks.sh` first; verify remote CI +
   latest master run conclusion (`gh run list`) before any push.
3. `git log -S 'shadow-lg group-hover:block'` closer already run (done —
   recorded here); nothing further needed on the mutation trail.
4. Let CI's browser job run `TestBrowser_TooltipDisclosureIsolation` once on
   a real CI Chrome (first green run = cross-environment confirmation).
5. Harvest this report's section (f) into TODO_LIST.md (docs-health HARVEST)
   and route the rest to ROADMAP.
6. Consumer-facing fixes plan their release — confirm the [Unreleased]
   section ships in the next cut or explicitly defer its contents.

**Direct follow-ups on this fix**
7. Env-guarded test that compiles REAL Tailwind v4 CSS at test time and
executes the hover flip in headless (closes Honest Gap #1).
8. Env-guarded screenshot capture of an OPEN tooltip (dark + light) for the
docs.
9. Write the `docs/gotchas.md` war story (three-round repro journey, daemon
mutation capture, false leads).
10. Add a mechanical guard: grep/lint rule banning unnamed
`group-hover:`/`group-focus-within:` (and bare `group` additions) in
`*.templ` outside the sanctioned CollapsibleSection wrapper.
11. Cross-check this repo's README for any stale tooltip/CSS-build claims.
12. CHANGELOG: link the concrete consumer rebuild recipe (`tc doctor` /
library-classes refresh) next to the migration note.
13. Verify the health-hub NixOS module / deploy stack CSS story: does
health.artmann.tech serve Play-CDN (fix arrives on rebuild) or compiled
CSS (needs regen)? Document the answer in deploy/README.md.
14. Fleet sweep: which sibling repos use CollapsibleSection together with
unnamed `group-*` utilities (ChastityAPI, CV, go-taskqueue, dnsblockd,
dynamic-markdown-site, timesheets, library-policy, webphone are known
templ-components consumers)?
15. Upstream (verify-before-filing first): propose templ-components document
or rename CollapsibleSection's `group` class as a NAMED group
(`group/section`) so consumers' unnamed variants stop colliding.
16. Consider an upstream `tc doctor` check for named-variant staleness in
consumer CSS.
17. Annotate the 2026-10-25 UI batch status report with a pointer to this
fix (docs-health ANNOTATE, when touching old reports next).
18. Extend the keyboard-navigation test to Tab THROUGH badge disclosures and
assert each opens only its own tooltip (focus leg already pinned
separately; tab-order version is belt-and-suspenders).
19. Touch/tap verification on a mobile viewport (tap focuses → focus-within
holds the tip open) — chromedp touch emulation, env-guarded.
20. Adjacent-row stacking check: focus badge A then click badge B — assert A
closes and B opens (z-index/overlap sanity).
21. Narrow-viewport clipping check: `left-1/2 -translate-x-1/2` tooltips near
screen edges with `max-w-[16rem]`.
22. Verify SSE patch behavior when a badge is FOCUSED at patch time (patch
replaces content → focus lost → tooltip closes; is that the accepted UX?
document or fix).

**Hygiene / adjacent code health (noticed this session, unrelated)**
23. Pre-existing gopls `scannererr` warning at browser_test.go:272 (bufio
Scanner without `scanner.Err()` check in `startHeadlessChrome`) — the
ONE genuinely fixable nit I touched the file but left alone.
24. Pre-existing gopls `unusedparams` infos (browser_test.go:382 `dash`,
integration_test.go:295 `name`) — trivial cleanups.
25. The eight templ `embedlit` hints in view.templ (pre-existing; evaluate
whether the suggested embedded-field removal is real or noise).
26. `latencywarn.html` golden carries no disclosures (grep-verified) — add a
golden scenario with MIXED severity + disclosures in source-grouping
mode if coverage feels thin.
27. Assert in a CLI test (not just browser) that the rendered markup carries
`group-hover/badge:block` — cheap class-contract pin outside the browser
suite (golden diff covers it today; an explicit assertion reads better).
28. Re-verify erraudit nolint sites are still minimal next lint sweep
(per fleet policy; no finding this session).
29. Consider `role="status"`/`aria-describedby` comparison with upstream
Tooltip's aria script — our sr-only pattern is audited, but record WHY
we diverge in one sentence in view.templ's doc comment (partially there
already).
30. Check `forced-colors` / `prefers-reduced-motion` rendering of the
disclosure (display flip only; likely fine — one screenshot).

**Fleet / process**
31. Record the "headless Chrome has no hover-capable pointer" fact in the
AGENTS.md Chrome gotcha cluster (it cost a false-negative diagnosis
round).
32. Same for "Play CDN (cdn.tailwindcss.com) serves v3-style scoped rules —
verified 2026-11-24" so nobody re-litigates CDN vs compiled.
33. After the cut: go-ecosystem-upgrade sweep of consumers onto the fixed
version.
34. Keep the pre-push watched-workflows + fuzz registry gates green when the
cut lands (gate 5/12 are mechanical — nothing new needed here).
35. TODO_LIST row for item 7 (real-CSS hover test) with a Blocked-by note
(Tailwind CLI in CI browser job).
36. ROADMAP raw idea: custom `tc doctor`-style guard in THIS repo that fails
CI if the pinned templ-components emits classes absent from a
reference-compiled CSS.

**Explicitly NOT worth doing (anti-list, so nobody picks these up)**
37. No wire contract changes — `/health` JSON, export, metrics untouched by
design; don't "improve" them onto this fix.
38. No Version const bump — the fleet standard ties it to the tag cut, not
to `[Unreleased]` accrual.
39. Don't migrate CollapsibleSection usage away from `<details class="group">`
— the class is upstream-owned; the fix is consumer-side and correct.
40. Don't add a custom stylesheet for the tooltip — the Play-CDN demo path
has no compiled CSS; class-only markup is the doctrine.
41. Don't chase the remaining `nlreturn`-style lint opinion beyond what the
fleet config already gates.
42. Don't re-run LOADTEST for a render-only change.
43. Don't refresh README screenshots for class-string-only diffs (pixels
unchanged).
44. Don't convert `browserStaticCSS` into a fuller Tailwind mirror — its
minimalism is the issue-#31 lesson.
45. Don't add axe runs to the isolation test — the existing axe coverage
loads its own harness; doubling it adds flake surface for no new signal.
46. Don't pin a newer chromedp for this change — the migration ceremony is
gated and unrelated.
47. Don't "fix" the `row-11`/`row-46` odd utilities seen in the accidental
v4 build output — scanner candidates, harmless.
48. Don't touch the `.crush/shell-output` dumps from this session (transient).
49. Don't relitigate PushOnChange/fingerprint contracts — untouched, pinned,
green.
50. Don't archive this report into `docs/status/archived/` until a later
session has actually harvested section (f).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Where did you see the bug?** (the example demo via `nix run .#example`,
   the health-hub deployment at health.artmann.tech, or one of your consumer
   apps with compiled CSS?) — It decides whether anything beyond this repo
   needs action: a compiled-CSS consumer now renders NO hover until its CSS
   is rebuilt, and I cannot observe which app you were hovering.
2. **Ship the fix as its own patch cut (v0.13.3) now, or ride the next
   release with the OIDC work already in `[Unreleased]`?** — Release
   discipline rule 7 requires an explicit decision in the same change as the
   fix; the business priority (exposed hub users vs batching) is yours to
   call and I won't guess a tag.
3. **Is the hover leg's verification bar met by inspection + a media-less
   runtime mirror, or do you want an env-guarded test that compiles real
   Tailwind v4 CSS and executes the hover flip?** — The honest gap is that
   headless Chrome cannot fire `@media (hover: hover)` rules; closing it
   costs a Tailwind CLI dependency in the CI browser job, and that trade is
   a policy call, not an engineering one.

---

_Point-in-time snapshot; goes stale the moment HEAD moves. Section (f) is
HARVEST fuel for TODO_LIST.md/ROADMAP.md — items 1-6 first. Waiting for your
instructions._
