# Status Report — Issue Triage: Pill Visibility, Nonce Stamping, CI/Fuzz Closures

**Date**: 2026-11-11 16:43 CET
**Session scope**: All 4 open GitHub issues (#31, #29, #23, #17) triaged; two code fixes implemented, verified, and committed locally; two infrastructure issues diagnosed and closed.
**HEAD**: `e3a0e9d` (fix: connection pill visibility and root data-nonce stamping) — local master **ahead 5, unpushed**.

## Scope Line — what this report VERIFIED vs NOT verified

**Verified this session (with artifact)**:

- Full unit + fuzz-seed suite: `nix develop -c go test -count=1 . ./cmd/health-hub` → green (22.1s main package).
- Browser suite: `go test -run TestBrowser -v` → **22/22 PASS** on nix Chromium 157.0.8103.38, serialized (27s wall).
- Pill browser probe strengthened: `TestBrowser_ConnectionPill` requires `getComputedStyle(el).display !== "none"` + exactly-one-visible.
- Mutation check: reverting the generated markup to the old `hidden class="inline-flex …"` makes `TestConnectionPill_MarkupAndStates` fail loudly (unit guard discriminates); restored, `-S` search confirms the mutation never got committed.
- golangci-lint: **0 issues** (`nix run .#lint`); `nix run .#vet` clean.
- Gates: `check-changelog.sh` all green; `check-templ-stamps.sh` 2/2 files match go.mod pin v0.3.1100.
- Goldens: regenerated with `-golden-update`, drift reviewed = exactly the `setPillState` script hunk (pill markup is not in goldens — fixtures lack the Datastar runtime).
- Root causes of #29 and #17 read directly from CI run logs (42477461128, 38674181389) before closing.

**NOT verified (and why)**:

- **CI on the new commits** — local master is unpushed (ahead 5); no run exists yet.
- **Datastar's behavioral no-`new Function` path under strict CSP** — the #23 fix is pinned structurally (attribute renders), not behaviorally (no browser test proves the live region connects under a CSP _without_ `unsafe-eval` once the nonce is wired).
- **cqrs-htmx's e2e pill assertions** (the #31 reporter's consumer) — sibling repo, not run.
- **Which browser version changed `[hidden]` UA handling** — measured on THIS host (Chromium 157 honors the attribute over `inline-flex`); the "UA rule changed vs your 156 measurement" claim in the #31 comment is an inference from one mutation experiment, not a sourced fact.
- Full 60s fuzz campaigns (only seed tests ran; nightly covers campaigns, green 11-10/11-11).
- The NixOS module's systemd unit behavior (d15fcb6 added it; only its fmt drift was touched this session).

---

## a) FULLY DONE

1. **#31 connection pill fix** — `page_scripts.templ`: the two hidden spans render `class="hidden min-w-24 …"` (class pair is the mechanism; the row-filter doctrine), `setPillState` toggles the class pair AND keeps the `hidden` attribute in sync (semantic marker + cqrs-htmx attribute state-machine contract). `browserStaticCSS` lost its compensating `[hidden] { display: none !important; }` rule → harness is production-honest. Probe strengthened. 22/22 browser tests green.
2. **#23 data-nonce root stamping** — `rootNonceAttrs` (status.go:718) → `layout.PageProps.HTMLDataAttrs` in `View()` (view.templ); nil nonce = byte-identical page. Pinned by `TestCSP_NonceStampedOnHTMLRoot` / `TestCSP_NoNonceOmitsRootDataAttrs`. Option 1 from the issue, as proposed.
3. **#29 closed** — root cause read from run logs: `nix fmt` drift in `nix/nixos-module.nix` from d15fcb6; already healed by 4519ee0 + green CI 42740040035 (2026-11-10). Closed with evidence-bearing comment.
4. **#17 closed** — run 38674181389 was a **build failure**, not a crasher: daemon dep sweep bumped chromedp to v0.21.0 while browser_test.go still used the v0.16 API. Migration landed 2026-10-28; Nightly Fuzz green since. Closed with evidence-bearing comment.
5. **Docs & memory** — CHANGELOG `[Unreleased]/Fixed` gained both fixes; AGENTS.md gained the pill class-pair bullet + extended nonce bullet (incl. the `.templ`-file-templ-import LSP trap). Issue comments posted on #31/#23 referencing the commit SHA.
6. **Commit hygiene** — daemon captures folded into ONE intent commit (`e3a0e9d`) via amend, per the daemon-race recipe; tree clean; all gates green post-commit.

## b) PARTIALLY DONE

1. ~~**The fixes are local-only** — ahead 5, unpushed. Consumers (cqrs-htmx per #31; file-and-image-renamer per #23) cannot consume anything. Issues #31/#23 deliberately left OPEN until origin has the fix + green CI.~~ (done at v0.13.3 2026-11-12: pushed through the cut `5e39639`, CI 8/8 green on it, #31/#23 closed with evidence comments)
2. ~~**Release gap** — v0.13.2 is still the latest tag; two consumer-facing bug fixes sit in `[Unreleased]`. No version bump, no tag, no release draft.~~ (done at v0.13.3 2026-11-12: cut `5e39639`, tag pushed first, verify-release all green, draft auto-produced by the release-draft workflow and published)
3. **#23 verification depth** — attribute stamp structurally pinned; the behavioral claim (Datastar avoids `new Function` with a root nonce) is upstream-asserted, not browser-proven here.
4. ~~**Chromium-version claim** — the "UA rule changed" inference needs a source; the comment text stands but the version boundary is unknown.~~ (resolved 2026-11-12: SOURCED — no boundary exists; the spec rule is plain display:none never !important (WebKit bug 152883 rejected it), Chromium html.css carries no [hidden] selector at all; the 157 reading was the harness compensating rule; correction comment posted on #31)

## c) NOT STARTED

1. Push master / CI verification on the 5 local commits.
2. Release cut (v0.13.3): re-head `[Unreleased]` + `Version` const bump in the same change, tag first, `verify-release.sh`, publish draft.
3. Closing #31/#23 post-push.
4. Behavioral strict-CSP browser test for #23.
5. cqrs-htmx consumer suite verification against the new pill markup.
6. Chromium `[hidden]` version-boundary research + comment/AGENTS.md correction.
7. Stale dependabot branch (`minor-and-patch-bca993d97f`, chromedp 0.21→0.22, failing CI since 2026-11-07) — untriaged.
8. Documenting the `nix run .#test` raw-regen + daemon-capture wrinkle in AGENTS.md gotchas.

## d) TOTALLY FUCKED UP

Nothing destroyed; three near-misses worth naming honestly:

1. **AGENTS.md overclaim written this session** — the pill bullet says `TestBrowser_ConnectionPill` "returns multiple:… so this bug class cannot pass again". On Chromium 157 the OLD markup does NOT trip the browser probe (the browser honors the attribute now); the **unit** test is the version-independent guard. The wording overstates the browser probe's power. One-line fix, queued.
2. **Mutation experiment put the bug back into the tree for ~2 minutes** — recovered via `git restore` from HEAD; verified via `git log -S` that the daemon never captured the mutated generated file. That was luck (daemon snapshot timing), not process. The sed+restore procedure needs the luck factor documented.
3. **Daemon captured a raw-shape `*_templ.go` (30d472d)** — `nix run .#test` regenerates raw-style output; walking away before `nix fmt` let the daemon eat it. Recovered by re-running fmt + amend. Known gotcha class ("raw templ output ≠ committed output"), but the `.#test`-app variant isn't written down.

## e) WHAT WE SHOULD IMPROVE

1. **Verify-external-claims discipline on the "UA rule changed" claim** — I applied it to my own host (mutation experiment) but let an unsourced version-boundary inference into a public issue comment.
2. **Behavioral pins over structural pins** — #23's gold verification (browser connects under strict CSP without `unsafe-eval`) is achievable and absent; structural pins don't catch upstream SDK regressions.
3. **Time-suspicious-greens as a reflex** — caught the 5.1s `.#test` green and the 1.8s cached-looking run by noticing timing; the browser suite silently skips inside the nix derivation, which briefly looked like proof. Document or make `.#test` run the browser suite.
4. **Release thinking at fix time** — two consumer-facing bugs landed without the release step being planned in the same change; the release-discipline gates exist precisely so this is mechanical.
5. **Issue-comment brevity** — the #31 comment carried good evidence but ran long for the terse-comment register.
6. **Fmt-before-walking-away** — any `nix run` app that regenerates templ output must be followed by `nix fmt` before the tree is left unattended in daemon territory.

## f) Up to 50 things to get done next

_(Brainstorm, not commitment — per status-report skill, most items below the top rows are ROADMAP fuel for docs-health HARVEST routing. Items marked ★ are session-fresh; others are repo backlog noticed this session.)_

**Release path (highest impact)**

1. ★ Push master (owner go-ahead) and watch CI (Build+Test+Hygiene+browser job).
2. ★ Cut v0.13.3: re-head `[Unreleased]` + `Version` const bump in the same change; `pre-push-checks.sh` gate 0 re-run AFTER the release edits (gate 5 lesson).
3. ★ Tag push first, then master; `verify-release.sh 0.13.3`; publish draft via `gh release edit --draft=false` after CI green.
4. ★ Close #31 + #23 once origin has the fix and CI is green. <!-- fresh:skip -->
5. ★ Confirm Release integrity workflow green post-release.

**Verification gaps**
6. ★ Browser test: strict CSP WITHOUT `unsafe-eval` + nonce wired → live region connects (proves the data-nonce behavioral effect for #23).
7. ★ Research which browser versions changed `[hidden]` UA handling; correct the #31 comment and the AGENTS.md "cannot pass again" wording (the unit test is the real guard).
8. ★ Run cqrs-htmx's e2e pill battery against the new markup (sibling repo).
9. ★ Run full 60s fuzz campaigns locally before the release (nightly covers; belt-and-suspenders).
10. ★ Run `nix run .#coverage` + `nix run .#vulncheck` before the release cut.
11. ★ Check whether `file-and-image-renamer` healthd can retire its nonce boot script once v0.13.3 ships.
12. ★ Sweep other embedded-dashboard consumers for pill/nonce exposure (MVS drift hazard list: ChastityAPI, CV, dnsblockd, timesheets, webphone, go-taskqueue, library-policy, dynamic-markdown-site).

**Repo hygiene (session-fresh)**
13. ★ Fix the AGENTS.md pill-bullet overclaim (one-line wording change).
14. ★ Record the `nix run .#test` raw-regen + daemon-capture gotcha (fmt before walking away).
15. ★ Document the browser-suite silent skip inside the `.#test` derivation, or make it run browser tests.
16. ★ Note the nix-store Chromium bump 156→157.0.8103.38 in AGENTS.md (suite re-verified on it, 27s serialized).
17. ★ Record the mutation-check "daemon-timing luck" factor in the mutation gotcha.
18. ★ Triage/close the stale dependabot chromedp 0.21→0.22 branch (failing CI since 2026-11-07; go.mod deliberately restored to v0.21.0).
19. ★ Consider a compile-only smoke workflow on push (catches dep-sweep build breaks before the nightly fuzz does — the #17 class).
20. ★ Consider a pre-commit fmt gate for `.nix` files (the d15fcb6 drift was only caught by CI Hygiene).
21. ★ Consider auto-close-on-green for the Master Red Alert workflow (currently manual close after fix).
22. ★ Restart the templ LSP to clear stale `rootNonceAttrs` diagnostics (cosmetic; lint is authoritative).

**Backlog noticed this session (from AGENTS.md/CI context)**
23. Drain the docs-gates 88-item pre-canon audit backlog (TODO_LIST row).
24. go-structure-linter: still skip-gated on the fleet bump; `.go-structure-linter.yaml` flat preset pending.
25. Watch templ releases for the a-h/parse requirement fix (a-h/parse pinned until then).
26. Next templ-components bump keeps the paired-procedure rule (bump + regen + pin-guard in one change; twelfth-movement precedent).
27. NixOS module smoke test: boot-verify the new health-hub systemd unit (added d15fcb6, never exercised here).
28. Cross-link the NixOS module from deploy/README.md (deploy stack facts doc).
29. Update AGENTS.md browser-suite timing expectation (docs said 2-3 min; healed host runs it in ~27s).
30. Re-measure Chrome launch latency on 157 (env-guarded `TestMeasureChromeLaunchLatency`, isolated run) if the number matters for CI budgets.
31. Master-red-alert issue template: include the failing job name (this session needed `--log-failed` digging).
32. `git log -S` verification should be a written step of the mutation recipe (it saved this session).
33. Fuzz workflow annotations are build errors — consider labeling compile-failure nights differently from crasher nights in the issue body.
34. CI Hygiene job: `nix fmt` drift diff was printed but no artifact attached; consider uploading the diff for one-glance fixes.
35. `.#test`/`.#build` apps: consider emitting a loud "browser suite skipped/ran" line so greens are interpretable.
36. CHANGELOG: the three `[Unreleased]` Fixed entries should survive the re-head into 0.13.3 verbatim (check at cut time).
37. `docs/status/` rotation: archive aging reports per docs-health ANNOTATE rules.
38. ROADMAP open-question sweep (templ/NixOS layering row) once the system templ pin proved stable.
39. Keep `/health` JSON byte-stability contract untouched while export grows (standing rule; no action, listed for drift-watch).
40. Watchdog/adversarial tests: no new surface this session; keep the floor test's hand-wired pusher pattern for any new watchdog work.
41. Sibling-repo `git log -S` audit: confirm no other repo vendors the old pill markup pattern (the attribute+display-class bug is copy-paste-prone).
42. Consider pinning `getComputedStyle`-based visibility assertions into the shared browser helpers (reusable for future show/hide UI).
43. EmptyState/filter chips: verify they hold on Chromium 157 + real compiled CSS (this session only verified the pill on the harness CSS).
44. Golden fixtures could add a Datastar-runtime-enabled variant (would pin pill markup in goldens too — the current fixtures never render it).
45. `check-ui-pins.sh`: consider adding the templ stamp + a-h/parse pin to the same guard (they're currently spread across two scripts).
46. Dependabot config: chromedp minor bumps will keep failing until the API-migration policy is encoded (allowlist or ignore rule).
47. AGENTS.md: the "Release discipline" section could gain the "fixes to released surfaces need a release plan in the same session" line (this session's gap).
48. Status reports: harvest this file's ★ items into TODO_LIST.md (docs-health HARVEST) if the session continues.
49. CI: consider running the browser job on the release tag too (currently master + PR matrix; tag pushes rely on master's run).
50. Optional: upstream the `[hidden]` doctrine to templ-components docs (component library consumers hit the same trap; #31's fix is dashboard-side only).

## g) Three questions I cannot figure out myself

1. **Push + release authorization**: shall I cut **v0.13.3** with these two fixes and push master now (release discipline: tag first, `pre-push-checks.sh`, `verify-release.sh`), or do you want the fixes to ride longer in `[Unreleased]`?
2. **#23 verification depth**: is the structural data-nonce pin enough, or do you want the behavioral proof — a browser test under a strict CSP _without_ `unsafe-eval`, nonce wired, asserting the live region actually connects (proving Datastar skips `new Function`)?
3. **Consumer verification**: should I run cqrs-htmx's e2e pill battery against the new markup in its own repo, or do you verify consumers separately as fleet practice?

---

_Waiting for instructions._
