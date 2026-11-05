# Status Report: Consumer CSS/CSP Incident — Diagnosis, Fixes, and Honest Self-Review

**Date:** 2026-11-04 07:55 CET
**Session scope:** "Projects that integrate us are failing CSS and Content Security policies!" — full diagnosis, library-side hardening, consumer-side fixes, documentation, and a brutal self-review of this session's own work.
**Repo state at writing:** `master` @ `546b98f`, **ahead of origin by 1** (the daemon had already pushed `dae5251`; `546b98f` — docs-only — awaits the daemon's push). Working tree clean.

---

## Scope: what this report VERIFIED vs did NOT verify

**Verified with commands/artifacts:**

- CI on the adoption commit `dae5251`: run `41783797773` — **all 7 jobs green**, including `Browser (runtime CSP)` (the runtime verification the local machine cannot perform) and `Hygiene` (templ drift check now passes).
- Full unit suite locally: green twice on the adoption content (`nix develop -c go test -count=1 .`, 48s, includes the browser-preflight self-skip).
- Lint: `buildflow -s golangci-lint --format finding` → `"findings": [], total 0` (the one step failure before it was host-load transient; rerun clean).
- Pin guard: `nix develop -c bash scripts/check-ui-pins.sh` → all OK (v1.26.0 pins + unchanged SDK sha `5d6b7794…`, size 33538).
- Canonical order: `go tool templ generate` (`updates=0` after fmt-shaped commit) → `nix fmt` (0 changed on `*_templ.go`).
- Render diff experiment: scratch module `/tmp/dash-drift` rendered `/health` under v1.24.0 vs v1.26.0 — delta is exactly the yellow→amber palette rename (plus a color-scheme meta already present in Base).
- Consumer module resolution: per-consumer `go.sum` greps (templ-components v1.26.0 in 8 consumers; go-datastar/go-sse stable at v0.6.2 everywhere).
- `tc doctor` on dynamic-markdown-site, library-policy, webphone — CSS-entry FAILs confirmed the missing/stale stylesheet class.
- webphone: rebuilt `health.css`, verified `amber-100/200` utilities present in the artifact (`rg -c` = 8 amber matches), `internal/app` tests green, their whitespace pre-commit gate clean.
- dms: new two-route CSP test green; full `internal/server` suite green under BOTH build tags (`-tags dev` and none); prod build + vet OK.

**NOT verified (and why):**

- **No local browser run of anything.** This host's Chromium cannot complete http(s) navigation (documented gotcha); ours rode CI's browser job; **dms's fix has no runtime SDK-alive check at all** — the test asserts CSP header strings, not that the SDK actually initializes under them.
- **Whether the "failing" consumers are browser-exposed in production** (deploy configs, ingress, auth in front of `/health`) — not researched for any consumer.
- **Whether ChastityAPI and CV actually mount go-health-dashboard** — their greps only surfaced their _own_ `dashboard` packages; the dependency may be indirect/policy-dogfood only. My consumer table may overcount integrators.
- **desire-secrets' actual rendering** — only a grep-level signal (no amber/yellow utilities in their `app.css`).
- `546b98f`'s CI run — commit is local-only at writing; docs-only, low risk, but unproven.
- Nothing in this report was verified against consumer _production_ deployments — all verification is source/config level.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                                                                                                                     | Evidence                                                                                 |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------- |
| 1  | **Root-cause diagnosis**: three stacked failure modes — (1) MVS drift (consumer graphs resolve newer templ-components than our tested pin), (2) CSS contract invisibility (consumer Tailwind never sees module-cache class strings), (3) consumer-side CSP/middleware breaks (headers bypassed or `unsafe-eval` missing) | session trace; `docs/gotchas.md` incident entry                                          |
| 2  | **Red CI fixed (templ fmt drift)**: raw generator output restored to the canonical fmt shape; Hygiene job green again                                                                                                                                                                                                    | commit `3f4ef11`; CI job `Hygiene ✓ 51s` on run 41783797773                              |
| 3  | **templ-components v1.24.0 → v1.26.0 deliberate adoption**: go.mod pins (root/datastar/icons/utils/htmx), all 7 goldens regenerated (yellow→amber warn palette), guard pins + audit history updated IN THE SAME CHANGE                                                                                                   | commits `34fd896` (daemon-staged mechanical half) + `dae5251` (intent); guard run all-OK |
| 4  | **`groupCountTone` yellow→amber** so the page ships ONE warning palette (its doc comment mirrors upstream BadgeWarning fills)                                                                                                                                                                                            | `status.go:48`; `status_test.go` updated; goldens yellow-free                            |
| 5  | **Contrast scan scoped honestly**: strips `aria-hidden` SVGs before applying the 4.5:1 text bans — v1.26.0's decorative Alert icon (`text-amber-600`, 3.19:1) passes WCAG 1.4.11's 3:1 non-text floor; the ban always meant status-bearing text                                                                          | `grouping_render_test.go` helper + updated out-of-scope comment; test green              |
| 6  | **go-health pseudo-bump disclosed, not absorbed**: 20261102 → 20261104, verified docs-only delta via module-cache diff                                                                                                                                                                                                   | `dae5251` go.mod; CHANGELOG entry                                                        |
| 7  | **README integration contract**: "The two integration failure signatures" (unstyled page / GenerateExpression-CSP) + dependency-graph-drift warning with the amber case                                                                                                                                                  | `README.md` new subsection after the CSP section                                         |
| 8  | **AGENTS.md eleventh-movement gotcha** (first DELIBERATE UI adoption, fleet-MVS motivation, measured delta, consumer hazard) + **CHANGELOG [Unreleased]** entries                                                                                                                                                        | both files updated; CI Version job green                                                 |
| 9  | **docs/gotchas.md long-form incident entry** incl. the `rg -E` false-negative method note                                                                                                                                                                                                                                | commit `546b98f`                                                                         |
| 10 | **TODO_LIST rows**: PR #22 templ-generator ceremony, RecommendedCSP hardening decision (go-taskqueue's local patch as evidence), fleet MVS watch (BLOCKED — needs fleet decision)                                                                                                                                        | commit `546b98f`                                                                         |
| 11 | **webphone fix**: vendored current graph, re-ran their staged `build-health-css.sh`, amber utilities verified in the committed artifact, `internal/app` tests green, vendor/ trashed (their transient convention)                                                                                                        | commit `7f7029dd` in webphone                                                            |
| 12 | **dynamic-markdown-site fix**: dashboard routes moved behind `securityHeadersMiddleware` (they had **zero** security headers before — the outer-mux bypass); dashboard-scoped CSP grants `unsafe-eval`; probe bypass intent preserved; site policy byte-unchanged elsewhere; new pinned two-route test                   | commit `9d857e6` in dms; suite green both build tags; prod build + vet OK                |
| 13 | **Render-diff proof** that v1.24→v1.26 changes only the warn palette on our surface (no new `style=`/`<style>`/script bytes)                                                                                                                                                                                             | `/tmp/dash-drift` harness, 8 diff-lines total                                            |
| 14 | **Consumer MVS survey**: templ-components resolution + go-datastar/go-sse stability per consumer                                                                                                                                                                                                                         | per-repo `go.sum` greps (documented in gotchas entry)                                    |
| 15 | **CI 7/7 green on the adoption** — the ceremony's runtime leg (browser) passed on v1.26.0                                                                                                                                                                                                                                | run `41783797773`                                                                        |

## b) PARTIALLY DONE

1. **Consumer remediation** — 2 of the identified affected repos fixed (webphone, dms). timesheets, library-policy, dnsblockd, desire-secrets remain (see c). Each outstanding repo needs its own gates run; none received any persistent artifact (issue/PR/commit) — the remediation for them lives only in this report and chat.
2. **dms dashboard CSS** — CSP/headers fixed; the page is still unstyled (no stylesheet exists there). Arguably acceptable for a dev-only surface, but it is the same "unstyled" symptom the user reported.
3. **Verification depth on the dms fix** — header-level test only; no proof the Datastar SDK actually initializes under the new policy (no browser available locally; dms has no browser CI job that I know of).
4. **Consumer verification matrix** — resolution versions are verified for all, but _mounting_ is verified for only a subset; ChastityAPI/CV mounting is UNKNOWN (own `dashboard` packages pollute the grep).
5. **Fleet MVS drift watch** — BLOCKED TODO row created, zero mechanism exists.
6. **RecommendedCSP hardening** — decision row created (go-taskqueue composes locally because `RecommendedCSP` lacks `frame-ancestors`/`form-action`/stricter `base-uri`); no implementation or API decision.
7. **Documentation of the incident in consumer-visible places** — README (ours) covers the contract going forward, but the consumers that already ship stale artifacts were not individually notified in their own repos.

## c) NOT STARTED

1. **PR #22 (dependabot)**: a-h/templ 0.3.1070 → 0.3.1100 (+ cdproto, prometheus/common) — needs the FULL templ ceremony (bump → `go tool templ generate` → `nix fmt` → commit output same change → audit output delta). The generator pin is NOT covered by any guard.
2. **Release cut** carrying the adoption (`v0.12.3` or fold into next): CHANGELOG [Unreleased] not re-headed, version const not bumped, no tag, no `verify-release.sh`.
3. **timesheets CSS pipeline**: zero `.css` files in the repo; `/health` fully unstyled; CSP wiring is otherwise exemplary.
4. **library-policy CSS/CSP decision**: their ADR-0009 wiring is minimal by design ("Dogfood is green"); styled page = a feature decision (tc init / manifest / flake tailwind job).
5. **dnsblockd manifest refresh**: `library-classes.txt` still carries the v1.24.0 badge line (`bg-yellow-100 …`); compiled `app.min.css` happens to contain amber, but the manifest is now lying about the module.
6. **desire-secrets palette check**: `app.css` contains neither palette — if their dashboard is browser-exposed, it is unstyled.
7. **ROADMAP update** — I touched TODO_LIST/gotchas/AGENTS/CHANGELOG/README but not ROADMAP (MVS-drift watch and manifest/embedded-CSS ideas belong there as raw ideas).
8. **Fleet-level `tc doctor` in consumer CI** — a BuildFlow-upstream-shaped idea, nowhere started.
9. **System templ bump** (v0.3.1020 → match the v0.3.1070 pin) — pre-existing root-fix note; dms's commit hook ran the skewed binary again this session (`updates=0`, harmless but the trap persists).
10. **Pre-existing TODO rows untouched this session**: CI failure notifications (11-day red window), remote-state pre-push gate, CI nix-build job, DevShell templ wrapper, README Installation section, screenshot content assertions, WCAG ratios from constants.
11. **TODO_LIST HARVEST**: this report's section (f) should be triaged into TODO_LIST/ROADMAP per the docs-health rule — partially done for 3 rows, not for the rest.
12. **Cleanup**: `/tmp/dash-drift` scratch module still on disk (harmless, untidy).

## d) TOTALLY FUCKED UP (this session's own failures — no varnish)

1. **A red tree got pushed to origin/master by the daemon.** I regenerated goldens and edited `status.go`/`status_test.go`/`check-ui-pins.sh` but left the whole adoption uncommitted while I wandered into consumer investigation. The auto-daemon committed the mechanical half (`34fd896`) **and pushed it** — that tree FAILED `TestRender_ContrastSafeStatusColors` (the fix in `grouping_render_test.go` was still uncommitted). Master was red on origin for the minutes until `dae5251` landed. This is exactly the release-discipline violation my own AGENTS.md warns about ("commit intent-bearing changes right after each verified batch — a long gate chain is one giant uncommitted window"). I had the rule in context and violated it anyway.
2. **Silent grep false negatives that I briefly reported as findings.** `rg -E node_modules` passes `-E` as ripgrep's _encoding_ flag, not an exclude — six consumer CSS scans came back "no amber, no yellow anywhere," which I put on the record before catching it and re-running with `-g '!…'`. Worse: the hook had already blocked me for rg-flag misuse twice earlier this session (`-rn` replace trap, "third strike on this host"), and I still fumbled flags afterward. Wrong intermediate conclusions on the transcript are worse than no conclusions.
3. **Wrong diagnosis published for dms, corrected only when my own test failed.** I first declared "their global CSP without `unsafe-eval` kills the SDK" — reality: their dashboard routes bypass the security middleware entirely (outer mux in `WithHealthProbes`), so they shipped _no_ headers and worked by accident of the missing policy. The mechanism I fixed is the right one, but my initial confident claim was wrong. A test — not my reasoning — caught it.
4. **Adoption was split across a heuristic daemon commit.** `34fd896 "chore: stage pending work before dependency update"` carries half the adoption with a meaningless message; the intent message had to come as a follow-up (`dae5251`) disclosing the split. Functional, but the history is uglier than the recipe I was following intended.
5. **Small wasted rounds**: a `for` loop missing `do`; a `git show … | sed -n '3p;25,35p'` probe that didn't show the imports block I was after; `rg -rn` hook-blocked again late in the session. Each trivial; together they signal rushing.

## e) WHAT WE SHOULD IMPROVE

1. **Commit verified batches immediately** — the pushed-red-tree incident is the single most damaging failure here and it was fully self-inflicted and foreseeable (the rule exists, in context, in AGENTS.md).
2. **Trust no grep whose flags you haven't re-read** — `-E`/`-r` traps produced one wrong conclusion and three hook blocks. A cheap habit: prefer `fd` for discovery and `-g` excludes, and re-check the first output of any new flag pattern.
3. **Sweep systematically before deep-diving**: a per-consumer matrix (mounts? CSS artifact? CSP wiring? resolved UI versions?) built with one small script would have found timesheets/desire-secrets in minute one instead of at the end.
4. **"Confirmed" means a reproducing test exists** — the dms diagnosis flip is the proof. Header assertions are necessary; runtime SDK-alive verification (browser) is the real bar for CSP work.
5. **Run `tc doctor` everywhere** — it exists precisely for this failure class and I ran it on 3 of ~15 consumers.
6. **Close the loop in the owning repo**: consumer problems ended in chat, not in their issue trackers — the buildflow rule ("file or fix, never absorb") wants each to end fixed, filed, or documented-non-fix _in the owning repo_.
7. **The library should make the contract mechanical** — README prose helps future sessions, but consumers need a checkable artifact (class manifest, testkit, or doctor gate). Prose is where this incident was born.
8. **Daemon coexistence discipline**: in a daemon environment, never hold content hostage to investigation; handshake (`git log`/`git status`) before _and_ after each batch, and amend-or-follow-up consciously (my recovery worked, but the red window was luck-of-timing).

## f) NEXT — 50 things worth getting done (brainstorm, impact-ordered; most extra items are ROADMAP fuel, not commitments)

**Consumer remediation (stops the reported pain):**

1. timesheets: add a CSS pipeline (tc init or manifest pattern) + `WithCSSPath` — the page is fully unstyled today.
2. library-policy: decide the CSS/CSP posture for the ADR-0009 reference wiring (styled page vs accepted-minimal) and either implement or record the non-fix in THEIR repo.
3. dnsblockd: regenerate `library-classes.txt` from the resolved v1.26.0 module (manifest currently lies).
4. desire-secrets: determine whether the dashboard is browser-exposed; if yes, add palette coverage to `app.css` + `WithCSSPath`.
5. ChastityAPI / CV: confirm whether they actually mount go-health-dashboard (grep ambiguity) — right-size the integrator count.
6. Build the per-consumer integration matrix (mount × CSS × CSP × resolved UI versions) with one repeatable script; run `tc doctor` on ALL consumers, keep the output as a fleet artifact.
7. File the outstanding consumer findings in each owning repo (issue or direct fix) so the loop closes there, not in chat.
8. dms: decide dashboard CSS (accept unstyled dev-only, or minimal stylesheet) — closes their half of the "CSS failing" symptom.
9. go-taskqueue: upstream-awareness check — their `withHealthHeaders` composition should survive a future `RecommendedCSP` change; leave them a comment/issue when (11) lands.
10. webphone/dnsblockd: add a canary that fails when the vendored/scanned module's rendered palette classes are absent from the artifact (the amber incident, mechanized).

**Library contract hardening (this repo — prevents recurrence):**
11. Decide + implement `RecommendedCSP` hardening (in-place v0.13 API change vs opt-in variant) — go-taskqueue's patch is the requirement spec.
12. Ship a consumer-copyable **class manifest** (`dashboard-classes.txt`) + a drift test that regenerates it from rendered goldens so palette changes can't ship silently.
13. Evaluate an **embedded prebuilt CSS option** (`WithCSSPath` defaulting to a go:embed'd build) — kills the unstyled class of failures by default; weigh the Tailwind toolchain cost in the flake.
14. Extend the pin guard to also pin the **templ generator version** (a-h/templ) — PR #22 is unguarded today.
15. Land `scripts/check-templ-stamps.sh` (stamp == go.mod tool pin) — pre-existing TODO, directly relevant to PR #22.
16. Merge PR #22 through the full ceremony (generate → fmt → audit output delta → same-change commit) or close it with rationale.
17. Cut the release carrying the adoption (re-head CHANGELOG, bump version const, tag-first push, `verify-release.sh`, re-run gate 0 after release edits).
18. Add a README **Upgrading note** for the amber rename at cut time ("recompile CSS after upgrading").
19. Add a one-line `WithCSSPath` doc-comment warning in `options.go` (recompile on any dependency bump) — contract at the API surface, not just README.
20. Consumer **testkit** idea: export a small helper (e.g. `dashboardtest`) asserting nonce-flow + CSP headers + CSS-link presence for consumer test suites.
21. Connection-pill UX when the SDK dies (CSP-blocked): distinguish "SDK blocked by policy" from "offline" (coordinate with templ-components/datastar).
22. ROADMAP entries: fleet MVS watch, class manifest, embedded CSS, RecommendedCSP hardening (raw-ideas home — I only fed TODO_LIST).
23. Introspection endpoint: surface the resolved templ-components version so consumers can self-detect drift (`WithIntrospection` already exists — extend output).
24. Golden docs: note in `golden_render_test.go` that goldens capture the FULL page (head+scripts), so upstream script/meta changes are visible there.
25. Re-verify the contrast-comment ratios (3.19:1 amber-600 etc.) from palette constants — folds into the pre-existing WCAG-constants TODO row.

**Fleet / process:**
26. Propose `tc doctor` as a BuildFlow fleet step (or per-consumer CI job) — the drift watch needs a mechanism, not a TODO row.
27. Record the cross-project lesson (consumer MVS drift invisible to library pins) in crush-config `references/lessons.md` via commit — it generalizes beyond this repo.
28. Bump the NixOS system templ package to match the fleet pin (root-fix for the v0.3.1020 CLI/LSP skew; hit again by dms's hook this session).
29. CI failure notifications for this repo (11-day red window class) — pre-existing TODO, still open.
30. Remote-state pre-push gate (fetch + ahead-count + last CI conclusion) — pre-existing TODO; would have caught today's pushed-red-tree faster.
31. CI nix-build job (vendorHash/FOD invisibility) — pre-existing TODO.
32. DevShell `templ` wrapper proxying `go tool templ` — pre-existing TODO.
33. Dependabot config: group UI deps (templ-components family) so bump PRs arrive as one reviewable ceremony instead of N singles.
34. Harvest this report's (f) into TODO_LIST/ROADMAP with docs-health routing rigor (done today for 3 rows only).
35. Delete `/tmp/dash-drift` scratch module (cleanup hygiene).
36. Verify `546b98f` actually gets pushed + CI-green (docs-only, but the loop deserves closing).

**Testing / verification depth:**
37. A browser-alive check for dms's dashboard under the new CSP (their repo needs a browser job or a manual verification pass) — the one fix in this session with no runtime proof.
38. Add a unit test pinning that SSE **patches** carry the amber palette (patch re-renders `dashboardContent`; today only full-page goldens pin it).
39. Test that the class manifest (if built, item 12) matches the SSE patch bytes too, not just the full page.
40. `nix run .#test-race` locally at least once post-adoption (CI covered it, local race never ran this session).
41. Fuzz quick-pass on export/CSV after the adoption (nightly fuzz covers it; a local 60s campaign would close the same-day loop).
42. Example binary: run it and eyeball `/health` rendered under v1.26.0 (the example is the reference consumer; nobody looked at the actual pixels this session).

**Docs / knowledge:**
43. README: add an "integration patterns" subsection citing the three real consumer wiring styles (webphone vendor-manifest, dnsblockd manifest, go-taskqueue CSP composition).
44. AGENTS.md: note the `-E`-encoding grep trap alongside the existing `-rn` trap (method-note class, third documented rg incident).
45. docs/status manifest: append this report to the status index if the convention requires it.
46. CHANGELOG: fold the consumer-incident context into the release notes wording at cut time (integrators reading notes should see the amber callout, not just "pins bumped").
47. Consider a `docs/integrations.md` collecting the per-consumer wiring inventory (the matrix from item 6, persisted).
48. Option `WithCSSPath` docs: link the README failure-signatures section from `doc.go`.
49. Upstream templ-components: propose documenting the amber rename as a BREAKING-ISH CSS-contract change in their changelog (consumers of consumers need it too).
50. Post-incident review of the daemon interaction: codify "commit before investigate" as a checklist line in AGENTS.md release-discipline (the rule exists; make the trigger explicit).

## g) Questions I cannot figure out myself

1. **Which consumer dashboards are actually browser-exposed in production, and which projects reported the failures you heard about?** Deployment/ingress/auth reality isn't in the code I can grep; it decides whether timesheets/desire-secrets/library-policy remediation is urgent or cosmetic, and whether CV/ChastityAPI even count as integrators.
2. **`RecommendedCSP` hardening: change it in place (v0.13 — every consumer's header bytes change, go-taskqueue's local patch becomes redundant) or add an opt-in hardened variant?** This is an API-compat call with fleet-wide blast radius; both designs are defensible and I need your compat posture.
3. **Do you want PR #22 (templ generator 0.3.1070→0.1100) merged now through the full ceremony as its own change, or held and bundled with the next release cut?** It's independent of the adoption but shares the release train; timing is your call.

---

**Then wait for instructions.** No further action taken after writing this report.
