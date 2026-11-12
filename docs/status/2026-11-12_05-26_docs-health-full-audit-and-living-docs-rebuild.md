# Status Report — Full Docs-Health Audit: All 2026-0* Files Read, Six Living Docs Rebuilt, One Report Archived, Gates 5/5 Green

**Date**: 2026-11-12 05:26 CET
**Session scope**: The user ordered a full docs-health AUDIT (BUILD + HARVEST + VERIFY + ANNOTATE) over every non-archived `**/2026-0*` file plus superb living docs (TODO_LIST, CHANGELOG, AGENTS, README, ROADMAP, FEATURES). This session touched **only documentation** — zero `.go`/`.templ` files — and ended with the docs-gates battery **5/5 PASS**.
**HEAD**: `06beb0d` (daemon commit capturing the audit edits) + one pending file (the `fresh:skip` marker on today's 16-43 report, daemon will capture).

## Scope Line — what this report VERIFIED vs NOT verified

**Verified this session (with artifact)**:

- Full docs-gates battery: `nix run github:Artmann-Technologies/docs-gates -- run .` → **freshness PASS, audit-all PASS, annotate PASS, routing PASS, cadence PASS** (matrix printed; started FAIL/FAIL/PASS/PASS/PASS).
- Every claim re-derived from code, not trusted from docs: go.mod pins (go-health v0.10.0, templ-components v1.27.0, go-datastar v0.6.2/static v0.6.1, go-sse v0.6.2, chromedp v0.21.0, cdproto v0.159.1), `Version = "0.13.2"` (dashboard.go:20), test recount (`rg -c '^func (Test|Benchmark|Fuzz)'` → **359/49 root + 14 cmd/pkg**), fuzz recount (**13** targets, 13 `-fuzz` invocations in fuzz.yml), `.docs-gates.json` (`audit_max: 87`, not 88), `RecommendedCSPStrict` at csp.go:85 (shipped v0.13.1, absent from README until this session), `WithStabilityThreshold` absent (design note's "not implemented" claim still true), `nix/nixos-module.nix` option surface (`services.health-dashboard`).
- Live GitHub state pulled fresh: issue **#32 OPEN (master CI red)**, #31/#23 open, #29/#17 closed; CI run 43061375783 failing jobs named (`TestShutdown_StopsPusherLoopGoroutine` on the race job, `TestBrowser_CollapseInteract` on the browser job); `git status -sb` shows master **in sync** with origin (the 16-43 report's "ahead 5, unpushed" is stale — the push happened ~16:45).
- The 7 non-archived `2026-0*` files read in full or sampled (3 HTML deep-dives head+tail): 4 markdown research/design files are fresh and carry living-source annotations — untouched; the 2026-08-09 HTML deep-dive already carried a supersession banner; **the 09-04 and 09-22 go-health HTML audits did not** — banners added this session.
- Archive gates on the annotated report BEFORE the `git mv`: `check-rows --strict` → "1 file(s) complete", `bare-rows` → "1 file(s) clean" (both initially FAILED and caught 3 rows I had missed — see §d).
- CHANGELOG structural lint: `bash scripts/check-changelog.sh` → all green after editing `[Unreleased]`.
- `nix fmt` → 0 changed (all hand edits formatter-clean).
- Daemon interleaving verified per the recovery doctrine: three daemon commits (`6ca9605`, `4dd07ff`, `621a250` + later `90bff75`/`06beb0d`) captured the living-doc edits mid-pass; `git show` confirmed each capture holds the intended content (v0.13.2 stamp, audit_max 87, Tagged v0.13.2 lead, rewritten TODO_LIST).
- Internal-link check over the eight living docs: two hits, both **false positives** (generic Go syntax inside code spans: `do.HealthCheck[*Dashboard](injector)` README:94, `Run[T](ctx, action)` AGENTS:167).

**NOT verified (and why)**:

- **CI green on the pushed fixes** — master is red (#32) on two tests that ran green locally at the identical Go tree 30 minutes before the red runs; flake-vs-real discrimination needs the diagnosis session, not docs work.
- **The NixOS module's runtime behavior** — no container/VM harness on this host; the module evaluates and the flake builds, nothing more. FEATURES now carries it as 🟡PARTIALLY_FUNCTIONAL for exactly this reason.
- **The Chromium `[hidden]` UA version boundary** — the "UA rule changed" inference in the #31 comment remains unsourced; research left open (TODO row).
- **cqrs-htmx's e2e pill battery** — sibling repo, not run (`repo:` tagged TODO row).
- **The full 98-item audit-all corpus** — I drained the 12 items whose credit my TODO purge deleted (verified green) but did NOT sweep the remaining 86 pre-canon items; they sit under the ratchet pin as the standing doctrine prescribes.
- CONTRIBUTING.md, SECURITY.md, docs/gotchas.md, docs/integrations.md, docs/metrics.md — existence-checked and link-checked only; no full VERIFY pass (out of this session's ordered scope).

---

## a) FULLY DONE

1. **All 7 non-archived `2026-0*` files viewed** (the other ~64 glob matches are `archived/` closed history, skipped per docs-health doctrine): benchmarks, dark-contrast, aggregate-load-test, the since-fed design note (read fully, all claims still true), the three HTML deep-dives (sampled; supersession state assessed and fixed where missing).
2. **All six living docs rebuilt to current truth**:
   - **AGENTS.md**: Status → v0.13.2; NixOS module added to the file map (`d15fcb6`, boot-verify pending); browser bullet updated (Chromium 157.0.8103.38, 22 tests, ~27s, `.#test`-skips-browser truth); pill-bullet overclaim softened (the unit test is the version-independent guard — the browser probe is host-Chromium-dependent); three new gotchas (`nix run .#test` regen + daemon capture, mutation-check `git log -S` closer, `.#test` greens are non-browser greens); Release discipline gained rule (7) "fixes to released surfaces plan their release in the same change".
   - **README.md**: version matrix rewritten from go.mod; `RecommendedCSPStrict` documented in the CSP section (the go-taskqueue sentence now names the shipped variant); health-hub section gained the NixOS deployment path.
   - **FEATURES.md**: Known Gaps pin row → v1.27.0/v0.21.0 truth; test row → 359/49 with a 2026-11-11 recount stamp; fuzz row → 13; CSP row names the Strict variant; Released row re-led to v0.13.2 with v0.13.1/v0.13.0 in the chain; **new NixOS module row (honest 🟡PARTIALLY_FUNCTIONAL)**.
   - **TODO_LIST.md**: full rebuild — 10 DONE rows purged (per its own header + docs-health doctrine), 3 resolved Blocked rows deleted, master-plan/docs-gates/row-47/row-59 evidence refreshed, and the **harvest landed**: CI-red #32 (top), cut v0.13.3, close #31/#23 (Blocked), behavioral strict-CSP test, cqrs-htmx (`repo:` tag), dependabot 0.21→0.22 branch, NixOS smoke + deploy/README cross-link, Chromium research, AGENTS-size decision (Blocked), MVS watch row restored with version tokens.
   - **ROADMAP.md**: shipped ideas annotated (federation → `cmd/health-hub` + v0.10.0 pin; HMAC → `WithWebhookHMAC` v0.13.2), decided Open Questions annotated (fingerprint DECIDED 2026-11-05, webhook appetite HALF-ANSWERED), fuzz budget 11→13, three new §5 raw ideas harvested from the 16-43 report + the drain.
   - **CHANGELOG.md** `[Unreleased]`: NixOS module Added entry written; audit_max text corrected 88→87.
3. **ANNOTATE + ARCHIVE pass**: `docs/status/2026-10-25_13-38_templ-cli-version-skew.md` — all 11 §f table rows + 6 §b/§c prose items struck with evidence-bearing verdicts (stamp guard, devShell wrapper, NixOS bump, gotchas history — every blocker landed 2026-11-04/05); gates green; `git mv` to `archived/`; archive manifest moved the entry out of "Kept in place" into the archived table with the deciding reason. Cadence headroom bought: 18/18 → **17/18**.
4. **Supersession banners** added to `docs/research/2026-09-04_go-health-deep-dive.html` and `2026-09-22_go-health-deep-dive.html`, mirroring the 2026-08-09 convention (POINT-IN-TIME comment naming the 2026-10-09 audit + AGENTS.md dependency notes as successors).
5. **Master plan annotations**: M01–M11 SHIPPED (with commit hashes), M27's #23 item half-resolved, M31 RESOLVED (host healed 2026-11-06 + 22/22 on 157) — the plan TODO cites as "authoritative detail" now agrees with reality.
6. **The 12-item audit-all drain**: my TODO purge un-credited 12 historical report items (86 → 98 open, battery red); all 12 annotated in their six source reports with honest verdicts (done/routed/decided-against + evidence) → battery green again. `freshness` PASS after the `<!-- fresh:skip -->` exemption on today's token-poor item 4 (the TODO row exists; the matcher cannot lexically credit it — residual noted in the health report).
7. **Final health report printed inline** (not filed): as-found Accuracy 4.75/10, Fitness 8.35/10, full findings table with per-finding fixes.

## b) PARTIALLY DONE

1. **The audit's own follow-through** — the gates are green, but the *code* side of the release path is untouched: #32 red, v0.13.3 unc ut, #31/#23 open. The docs now say exactly that (that's the job), but the underlying work is a later session's.
2. **Living-doc coverage of `docs/` sub-surface** — gotchas.md, integrations.md, metrics.md, release-checklist.md, DOMAIN_LANGUAGE.md, ADRs were existence- and link-checked only; a full VERIFY pass over them was out of scope and not attempted.
3. **The AGENTS.md pill-bullet correction** — wording fixed, but the underlying claim in the #31 GitHub comment (unsourced "UA rule changed" boundary) is corrected NOWHERE yet except this repo's AGENTS; the comment itself needs the sourced follow-up.
4. **Today's 16-43 report** — its §f items are now harvested (freshness PASS), but the report file itself is unannotated by design (freshest report = harvest source, not annotate target); it stays unarchived until its release-path items resolve.
5. **Daemon-capture disclosure** — five daemon commits carry my edits under `chore:` labels; content verified but the intent (docs-health audit) lives only in this report + the pending fresh:skip edit, not in an intent commit. The daemon-race recipe says amend/disclose at the next intent commit.

## c) NOT STARTED

1. Fix master CI red (#32): diagnose `TestShutdown_StopsPusherLoopGoroutine` (race job) + `TestBrowser_CollapseInteract` (browser job) — flake-under-load vs real regression, at a Go tree identical from `e3a0e9d`..`2ca9eca`.
2. Cut v0.13.3 (re-head `[Unreleased]` + Version bump same change; gate 0 after release edits; tag first; verify-release; publish draft).
3. Close #31 + #23 once the fix is released with green CI.
4. Behavioral strict-CSP browser test for #23 (live region connects WITHOUT `unsafe-eval` once the root nonce is wired).
5. Run cqrs-htmx's e2e pill battery against the class-pair markup (`repo:` cqrs-htmx).
6. Triage/close stale dependabot branch `minor-and-patch-bca993d97f` (chromedp 0.21→0.22, red since 2026-11-07).
7. NixOS module boot smoke test + cross-link from deploy/README.md.
8. Chromium `[hidden]` UA version-boundary research + sourced correction of the #31 comment.
9. The docs-gates 86-item pre-canon backlog drain (standing ratcheted row).
10. AGENTS.md size decision (537 lines vs fleet 220 budget) — standing user question since 2026-11-06.
11. Everything else in the rebuilt TODO_LIST (M12+ of the master plan, watched-list single-sourcing, go-etag grep port, …).

## d) TOTALLY FUCKED UP

Nothing destroyed — three real process failures, all caught and recovered in-session:

1. **The TODO_LIST full-file rewrite silently dropped one live row** — the Consumer-side MVS drift watch (original row 27, 🔵BLOCKED on a fleet decision) vanished in the rewrite. Caught in the VERIFY phase only because the audit-all gate's annotation specs re-referenced it; restored with refreshed version tokens. Root cause: wholesale `write` of a dense table instead of row-by-row diff discipline. A rewrite that silently deletes memory is a loss, not an update — the memory rule I know and briefly failed.
2. **I trusted truncated tool output for the archive annotation** — `rg | head -40` and `emit-keys | head -12` showed 8 table rows; the table had 11. I specced 8 verdicts believing the set complete. The moved-file gates (`check-rows --strict`, `bare-rows`) FAILED loudly and caught rows 9–11 before the `git mv` — the gate-before-move doctrine is the only reason the archive is honest. Never pipe gate-relevant tool output through `head`.
3. **My own gate conditional was inverted — twice** — `grep -L '~~' file && echo "ok"` : `grep -L` exits 0 when it LISTS files (no matches), so the success branch printed on a file with zero markers, and I read that as passing on the first run. Caught on the second run when the same construct printed FAIL. The authoritative per-row gates were right both times; my shell wrapper lied to me and I believed it once. Direct `grep -c` settles it (19 markers in the archived file).

## e) WHAT WE SHOULD IMPROVE

1. **Full-file rewrites need a row-by-row before/after diff** — the dropped MVS row (§d1) is exactly the class the "no duplication → no loss" memory rule targets. For dense tables: `diff <(git show HEAD:TODO_LIST.md) TODO_LIST.md` as a written step after any rewrite, not optional diligence.
2. **Never eyeball gate semantics through conditionals** — run the gate, read its raw output, THEN wrap. `grep -L` exit semantics (and art-dupl's summary-line drift, already a documented gotcha) are trap one-liners; the wrapper around them is where false greens are born.
3. **`head`-truncation is a correctness hazard on emit-keys/rg output** — the skill's own doctrine says run emit-keys BEFORE writing specs so refusals surface on the read path; truncated reading defeated exactly that. Written rule: emit-keys output is consumed in full or not at all.
4. **The freshness matcher's generic-token escape (`fresh:skip`) will recur** — token-poor items ("close X + Y once CI is green") are structurally unmatchable. Either the TODO row learns to carry one distinctive noun from the item, or the skip marker is the standing answer; today I used the skip and disclosed the residual. A one-line note in the docs-gates config/README about the pattern would save the next session the same 20-minute diagnosis.
5. **Harvest-at-fix-time** — the 16-43 report sat unharvested for ~13 hours with the freshness gate able to prove it; today's audit closed the gap, but the ritual ("harvest the report's ★ items before walking away") belongs at report-writing time, not next-docs-pass time.
6. **Daemon-commit disclosure batch** — five `chore:` daemon commits carry this audit's substance; at the next intent commit, the message should disclose the mapping (the recovery recipe's member (1) pattern), keeping history honest without force-push.

## f) Up to 50 things to get done next

_Brainstorm per the status-report doctrine — top rows are commitments, the tail is ROADMAP fuel for docs-health HARVEST routing. ★ = session-fresh; unmarked = repo backlog noticed this session._

**Release path (highest impact)**

1. ★ Fix master CI red (#32): reproduce both failing tests, discriminate flake-under-CI-load vs real regression, fix, re-run failed jobs UNCRYPTED, watch green, close #32.
2. ★ Cut v0.13.3: re-head `[Unreleased]` + Version bump in one change; gate 0 re-run AFTER the edits; tag first then master; `verify-release.sh 0.13.3`; publish draft after CI green.
3. ★ Close #31 + #23 post-release (Blocked until 2 lands green).
4. ★ Confirm the Release integrity workflow produced a real Draft post-cut (the fixed workflow's second live run).
5. ★ Carry the `[Unreleased]` Fixed entries verbatim through the re-head (16-43 §f36's check, still unexecuted).

**Verification gaps**

6. ★ Behavioral strict-CSP browser test for #23 (no-`unsafe-eval` + nonce → live region connects).
7. ★ cqrs-htmx e2e pill battery (repo: cqrs-htmx).
8. ★ Chromium `[hidden]` UA boundary research + sourced #31-comment correction.
9. ★ NixOS module boot smoke test (VM/host), then flip FEATURES to FULLY_FUNCTIONAL.
10. ★ Cross-link the NixOS module from deploy/README.md.
11. ★ Run `nix run .#coverage` + `nix run .#vulncheck` before the v0.13.3 cut (16-43 §f10, still open).
12. ★ Run the full 60s fuzz campaigns locally before the cut (nightly covers; belt-and-suspenders).
13. ★ `nix develop -c go test -count=1 -race .` full-package soak on the release candidate tree.
14. Verify the 22-test browser suite stays green on Chromium 157 with the regenerated goldens (post-#31 markup).

**Repo hygiene (session-fresh)**

15. ★ Add the row-by-row rewrite-diff discipline to AGENTS.md gotchas (the §d1 lesson, written down while fresh).
16. ★ Note the `grep -L` exit-semantics trap next to the art-dupl summary-line gotcha (same false-green class).
17. ★ Decide the `fresh:skip` pattern note: docs-gates README/config comment vs AGENTS line.
18. ★ Daemon-disclosure intent commit: map the five `chore:` daemon commits to this audit in one message.
19. ★ Triage/close dependabot branch `minor-and-patch-bca993d97f` (or add the dependabot ignore rule for chromedp minors pending the API-migration policy).
20. ★ Sweep sibling repos for the old attribute-hidden pill markup pattern (`git log -S` audit; 16-43 §f41).
21. ★ `getComputedStyle` visibility assertion → shared browser helper (16-43 §f42; reused by future show/hide UI).
22. ★ Make `nix run .#test`/`.#build` print a loud "browser suite skipped" line (ROADMAP §5 idea harvested today; implement or formally park).
23. ★ Master-red-alert issue template: include the failing job name (16-43 §f31).
24. ★ CI Hygiene: upload the fmt-drift diff as an artifact (16-43 §f34).
25. ★ Restart the templ LSP to clear stale `rootNonceAttrs` diagnostics (cosmetic; lint authoritative).
26. ★ Update the archived/README "Kept in place" list — the 14-19 entry still claims active-handoff items that v0.13.2 shipped (nix-build job, templ ceremony done; red-main alerting shipped 2026-11-04).
27. ★ Archive-rotation sweep: the remaining 2026-10-25/10-28 reports (14-19, 11-19, 11-34, 15-17, 15-45) are now largely resolved — ANNOTATE→ARCHIVE candidates for the next docs pass (cadence 17/18 buys one more).
28. ★ The 16-43 report's own §b residuals need their release-path closure noted once v0.13.3 ships (annotate the report then).

**Docs-health backlog (standing)**

29. Drain the docs-gates 86-item pre-canon audit backlog (TODO row; one stale cluster per docs pass, lower the pin each time).
30. AGENTS.md size decision: 537 lines vs fleet 220 budget — restructure (split gotchas/dependency notes into docs/) or accept (Blocked on user).
31. Watchdog watched-list single-sourcing (TODO row, OPEN).
32. go-etag fragile-grep port (TODO row, `repo:` go-etag).
33. markdown-lint pass over docs/ (ROADMAP §5, M21).
34. Codify the cold-cache `check-ui-pins.sh` behavior as a test (ROADMAP §5, M24).
35. devShell shellcheck + actionlint (ROADMAP §5, M20).
36. `interrogate` doc-coverage evaluation (ROADMAP §5, M22).
37. buildflow rerun-failed-uncached baseline evidence (ROADMAP §5, M23).
38. Hub boot smoke test in CI (ROADMAP §5, M18; container runtime absent locally).
39. Cross-process browser-test file-lock (ROADMAP §5/§7, M19).
40. Re-check the gopls stdversion dismissal on a future gopls upgrade (ROADMAP §5).
41. Re-sweep ROADMAP v1.0 criterion 2 (consumer signal) once CV/siblings ride v0.13.x — the sweep table cites the 2026-10-25 consumer state.

**Fleet / external (noticed, not actionable here alone)**

42. templ releases: watch for the a-h/parse requirement fix (a-h/parse pinned until then; the reverted 2026-11-06 sweep is the standing hazard).
43. templ-components docs: upstream the `[hidden]` doctrine (16-43 §f50; component consumers hit the same trap).
44. DiscordSync's own templ-components v1.24.0+ adoption (its snapshots drifted 15; its repo's row — annotated decided-against-here today).
45. CV-side bump + deploy + live verify (Blocked row; user call).
46. dnsblockd lockstep gate (Blocked row; go-datastar static tag or gate relax — owner act).
47. PVR enablement (Blocked row; repo-admin act).
48. Em-dash copy decision for shipped strings (Blocked row; ~200 sites incl. goldens).
49. Second-session-during-release policy (Blocked row; unpoliced).
50. File-and-image-renamer: check whether its healthd nonce boot script can retire once v0.13.3 ships (16-43 §f11).

## g) Three questions I cannot figure out myself

1. **Release path go-ahead**: master is red (#32) with two consumer-facing fixes stranded in `[Unreleased]`. Do you want (a) diagnose-and-green #32 first, then cut v0.13.3 (my recommendation — the fingerprint/wire pins are untouched by the failing tests, but shipping from red master violates the release-discipline gate), or (b) cut v0.13.3 immediately and let #32's diagnosis ride the next window?
2. **TODO_LIST DONE-row policy**: I hard-deleted all 13 DONE/resolved rows today per docs-health doctrine and the file's own header ("every closed item lives in CHANGELOG, never here") — but the repo has a visible habit of keeping 🟢DONE rows as a session-visible record. Is hard-delete the standing rule (my assumption now embodied in the file), or do you want a short "closed this week" grace section?
3. **AGENTS.md size**: 537 lines vs the fleet's 220-line budget — the preflight warns on every run, and today's additions pushed it further. Restructure (move the dependency notes + gotcha detail into `docs/` and keep AGENTS as the index), or formally accept the overage as the price of single-file context?

---

_Waiting for instructions._
