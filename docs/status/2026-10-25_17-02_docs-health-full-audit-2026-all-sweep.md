# Status + Brutal Self-Review — docs-health FULL AUDIT: all 2026-* files, living-docs rebuild, 35-file archive sweep (2026-10-25, 17:02)

| | |
| --- | --- |
| **Session** | "View ALL \*\*/2026-0\* files! Execute the docs-health SKILL! ... TODO_LIST, CHANGELOG, AGENTS, README, ROADMAP, FEATURES must be all SUPERB ... Archive FULLY done and UPDATED (inline strikethrough) .md files" — one session, AUDIT mode (BUILD + HARVEST + VERIFY + ANNOTATE/ARCHIVE) |
| **Window** | ~14:30–17:00 CET |
| **Start state** | Clean tree at `e8eeefa` (CI 7/7); `docs/status/` held 37 non-archived dated reports (13×2026-09, 24×2026-10) + 4 planning + 1 design + research; one untracked report (14-19, the previous session's handoff) |
| **End state** | 35 files annotated inline (~900 item verdicts) and `git mv`'d to `archived/` (33 status/ledger + 3 planning, incl. the harvest ledger and both completed pareto plans); exactly 3 dated files remain unarchived (13-38, 14-19, planning 07-52 — all genuinely open); archive manifest written; living docs (TODO_LIST/FEATURES/ROADMAP/CHANGELOG/AGENTS) refreshed; all gates green |
| **Gates at stop time** | `pre-push-checks.sh` **8/8 green** (changelog lint, fuzz registry 13/13, SECURITY freshness 0.11.x, templ pin grep, **lint 0 issues**); archive completeness gate (`grep -L '~~'` over the moved set) **empty**; `check-rows` **35/35 complete**; `nix fmt` **0 changed** (no struck-row rewrap); living-doc link-check **clean**; CI 7/7 on `e8eeefa` (inherited from the 14:19 session, not re-watched this session) |
| **Commits** | None by me (no authorization); the auto-daemon swept the sweep into `269e8d0`/`f38b7e0`/`b895cc6` + successors |

## a) FULLY DONE

1. **READ leg — every non-archived 2026-\* file viewed** (37 status + 4 planning + design + research notes, full reads for today's nine, structured extraction for the rest), plus the six living docs and the skill's asset/reference contracts.
2. **VERIFY leg against code:** `Version = "0.11.0"` (dashboard.go:20), go.mod pins (go-health v0.7.0, templ-components v1.23.0 ×4, go-datastar v0.6.2/static v0.6.1, chromedp v0.16.0+cdproto), test counts **346/47 root + 14 cmd/pkg** (FEATURES matches exactly), CI green on `e8eeefa`, CHANGELOG `[Unreleased]` coverage cross-checked against the day's reports.
3. **Living docs fixed:** FEATURES "Released" row v0.10.1 → **v0.11.0** (the released-Latest claim contradicted the code — worst pre-sweep drift) + chromedp pin note; TODO_LIST: stale "Cut v0.11.0" BLOCKED row closed (release shipped 2026-10-12), CV premise v0.6.1 → v0.10.2, pin sign-off row refreshed to the ninth sweep, footer now "v0.11.0 is Latest; CI 7/7 on `e8eeefa`"; **11 new rows** harvested from the 13-38/14-19 §f queues (stamp guard, devShell templ wrapper, remote-state pre-push gate, CI nix-build job, dependabot branch review, README Installation section, Cut-v0.12.0 BLOCKED row, capture content-assertions, contrast-ratio test, small quality batch, canary-first-run check); ROADMAP: two live Open Questions added (v0.12.0 cadence, templ toolchain layering) + v1.0 criterion-2 refresh (7 siblings on v0.11.0, CV at v0.10.2) + theme-7 file-lock idea; CHANGELOG: three Fixed entries for the 14:19 session's user-visible work; AGENTS: release-discipline now requires remote CI state before push.
4. **ANNOTATE + ARCHIVE sweep — 35 files, ~900 inline verdicts:** every numbered item in every resolved report struck with a per-item disposition (`done at`/done — /routed/NOT-DO), per-item for the files with known mappings (07-39's 50-item list, 12-26/12-27/13-20, the leverage plan's T01-T20), section-bucket verdicts with concrete anchors (harvest ledger, sweep rows, CHANGELOG sections) for the superseded batches; §d/§e retrospectives left bare per canon; the 17-30 HARVEST ledger ledger-exempted (`check-rows:ledger`, it was already 100% dispositioned); the zero-item 19-30 tag audit got its completion strike; the 13:38 report's owed-gates items annotated as satisfied (the 14:19 report's f9 ask).
5. **Both archive gates pass:** `grep -L '~~'` over the 35 moved files → empty; `check-rows` → 35/35 complete; `nix fmt` re-run → 0 changed (dprint did not rewrap any struck row).
6. **Archive manifest** (`docs/status/archived/README.md`): one line per moved file with the deciding reason, the kept-file list with reasons, and the deviation notes — the bulk-archive requirement.
7. **Link-rot closed out:** every living-doc reference to a moved file repointed to `archived/` (6 exact + 1 glob-style), re-verified clean; TODO_LIST's own new evidence cells included.
8. **Kept-set discipline:** only 13-38 (templ skew: NixOS bump, stamp guard, wrapper, LSP all open), 14-19 (active handoff: v0.12.0 cadence, red-main alerting, remote gate, nix job, dependabot), and planning 07-52 (T24/T27 open) remain unarchived — each with inline annotations where later sessions paid their debts.

## b) PARTIALLY DONE

1. **Annotation depth is two-tiered.** Per-item hash-level verdicts for the 2026-10-08..25 chain (where TODO_LIST/commit evidence named the items); evidence-anchored section-bucket verdicts for the 2026-09 batch (swept by the 09-23 full-list execution). Honest and checkable, but not per-item-hash archaeology for files a month old.
2. **§d/§e sections unstruck by design** (retrospective narratives, not tasks) — consistent with the archived canon I sampled, but it means `grep -c '~~'` does not cover 100% of numbered prose in the archive.
3. **The 14-19 report's §f items 2/6/7/8/12/17-21/23/25** (red-main mechanism choice, securitymd/buildflow upstream filings, dependabot docker entries, cold-cache test codification, devShell tools, interrogate, buildflow re-run baselines, markdown-lint pass, hub boot smoke, release-draft re-verify) are tracked by the KEPT report, not minted as TODO_LIST rows — the kept file is the active queue, but TODO_LIST is the canonical backlog and these are not there.
4. **Research/design files kept in place without annotation** (benchmarks, dark-contrast, aggregate-load, three HTML deep-dives, the 09-17 design doc): historical snapshots actively cited by living docs — verified only to header/claim level, not item-level.
5. **v1.0 sweep:** only criterion 2 was refreshed with new evidence; criteria 1/3/4 keep their 2026-09-23 dating.

## c) NOT STARTED

1. **Push** — never authorized; the daemon owns commits (tip `b895cc6`).
2. **v0.12.0 cut** — cadence question parked in ROADMAP Open Questions; `[Unreleased]` is release-sized and all gates are green.
3. **Every row I minted** — that is the deliverable, not debt: stamp guard, wrapper, remote gate, nix job, dependabot, README install, content assertions, contrast test, quality batch, canary check, CI notification mechanism.
4. **AGENTS size trim** (405 vs the 220-line budget — standing, flagged again by the 14:19 report; I added one bullet and note it as fitness debt).
5. **The chromedp v0.21 migration / pin policy** and **em-dash copy decision** — standing BLOCKED rows, untouched by design.

## d) TOTALLY FUCKED UP (own errors, all recovered, zero content loss — annotate.go is atomic and every failure was pre-write)

1. **I violated the "read the checker's source first" lesson three hours after it was written.** The 14:19 report (§e4) said exactly this; I still guessed the status-items spec grammar (`key:kind:"value"` — that's rows/prose) instead of reading annotate.go's doc comment (`key<TAB>verdict`), burned a 66-spec verify run, then needed THREE more iterations on the emit-keys pairing (stdout=keys vs stderr=linemap; sequential indices, not line numbers; the key sits BEFORE the `lineno:` line) before the first file annotated. Five failed runs to learn a documented contract.
2. **genmap.sh emitted item numbers instead of file line numbers** — six files failed with MISSING KEY before I saw that awk needed `FNR`. A print-statement debug would have caught it in one run; I found it by inspecting pairing output.
3. **Two "unmatched spec" failures on lines containing literal `~~`** (the resolve() guard blunt-skips any such line) — I first misdiagnosed the map filter (`awk '$1!=10'` filtered the wrong column semantics, two more dead runs, plus a python heredoc SyntaxError from a lazy placeholder) before grepping the real FNRs (80 and 108) and striking those two lines by hand.
4. **`git mv` batch with a wrong composed target** (`$(dirname $(dirname $f))/archived/` → `docs/archived/`) — failed loudly, nothing moved, fixed with explicit targets.
5. **Edit-before-View rejections ×4** (CHANGELOG, AGENTS, 12-27, 10-45) — I had only bash-read those files; the rule and the 13:54 report's d1 lesson both say view first. Wasted four round trips.
6. **19-55 hand-built map off by two lines** (13/14 instead of 7-12) on a 27-line file — the one class of error grep -n output was sitting right there to prevent.
7. **Heavy reads late-optimized:** I full-read ten reports before switching to headers+numbered-lines extraction — the compressed mode was the right default from file ~10 and would have bought headroom for the deeper per-item verdicts I then economized on.

## e) WHAT WE SHOULD IMPROVE

1. **Read the tool's source before the first bulk invocation** — the annotate toolkit documents its grammar in the doc comment; one `sed -n '1,40p'` beats five failed runs. (Third time this lesson has been paid for today alone.)
2. **Synthetic dry-run before any new spec shape:** the skill says "ALWAYS dry-run the first spec against a new file shape"; I dry-ran against REAL files (which is not a shape test). A 3-line fixture would have exposed the grammar and pairing mistakes in one call.
3. **Compressed extraction is the default reading mode for bulk audits:** headers + numbered-item lines first; full reads only for the files whose verdicts need prose context (today's chain).
4. **Build the pairing from the tool's stderr, never positionally against assumptions** — emit-keys' two-stream output is a contract, not an accident; robust helpers beat clever one-liners (my final ann.sh survives skip-lines and stream separation).
5. **Path composition for batch git ops gets one echo dry-run** — the `dirname dirname` bug was visible in the command itself.
6. **Sibling-report claims used as gate inputs should be re-checked in-session** (`gh run list`) when the claim gates an archive decision — I leaned on the 14:19 report's CI evidence plus the local gate; cheap to verify directly.
7. **Archive sweeps must include a link-rot pass over the LIVING docs in the same change** — I caught the six broken references only because I ran the check; it should be a mechanical step of every `git mv` batch (the skill states it; now this repo has felt it).

## f) Up to 50 things to get done next (order = value density; the TODO_LIST rows are canonical, this is the session-ranked view)

**Decisions that gate everything: 1-4**

1. Cut v0.12.0 or hold (ROADMAP Open Question; `[Unreleased]` is release-sized; tree + CI + FOD green).
2. Chromedp: v0.21 migration as a dedicated change, or write the pin policy down (TODO row).
3. Red-main CI notification mechanism (GitHub-native vs webhook vs fleet; TODO row).
4. Templ toolchain layering: NixOS templ bump (external root fix) + how much in-repo belt-and-suspenders (stamp guard + wrapper rows exist).

**In-repo, small, high-signal: 5-16**

5. `scripts/check-templ-stamps.sh` + pre-push/CI wiring (13-38 f2).
6. DevShell `templ` wrapper proxying `go tool templ` (13-38 f3).
7. Remote-state pre-push gate (fetch + ahead + last CI conclusion, warning-only) (14-19 f3).
8. CI nix-build job (or periodic) for the vendorHash/FOD class (14-19 f5).
9. Review/merge dependabot `minor-and-patch-290df43455` (mind the chromedp row) (14-19 f4).
10. README "Installation" section (14-19 f13).
11. Content assertions in the four screenshot capture tests (minted row).
12. Contrast ratios computed from palette constants in one test (minted row).
13. Small quality batch (aggregate axe audit, Dockerfile base-floor check, SECURITY mirror in release-draft, gotchas harness-canvas entry, race-leg note, fleet `sort|tail` audit) (minted row).
14. Verify the flake-canary's first nightly run (minted row; first fire was tonight).
15. NixOS templ bump ≥ v0.3.1070 (external act; kills the phantom-squiggle class) (13-38 f1).
16. Editor templ LSP repoint/rebuild (13-38 f5).

**Queue items still living only in the kept 14-19 report — route or do: 17-25**

17. Reproduce + file securitymd upstream (`filesScanned: 0` beside a finding) (b1).
18. Repro + file buildflow upstream (nix-hash-fix double-WARN before self-heal) (b1).
19. Dependabot docker entries: drop vs own Dockerfiles (f12).
20. Codify the pin-guard cold-cache test so guard edits re-learn it (f17).
21. DevShell additions (shellcheck/dprint/lychee/hadolint/vulnix) vs documented fallback (f18).
22. Install or silence `interrogate` (f19).
23. Full buildflow re-run post-fixes; rebuild timing baselines (f20).
24. One `markdown-lint`/codespell/gitleaks pass over today's dprint-rewritten files (f21).
25. Health-hub boot smoke with empty `HEALTH_HUB_REMOTES` (f23).

**Release train: 26-31**

26. At cut: full checklist §1-§8 incl. §7 verifier expectations + §5 daemon recipes.
27. Recapture all four screenshots on the release commit (after item 11).
28. SECURITY row/footer pre-flight (gate makes it mechanical).
29. Post-tag consumer re-sweep (7 green siblings re-pin; nsfw already v0.11.0).
30. pkg.go.dev propagation spot-check for the new tag.
31. `verify-release.sh <ver>` full 9/9 + release-draft workflow re-verify (f25).

**Docs/system hygiene from this session's observations: 32-42**

32. Fleet-propagate the templ stamp-guard pattern (the stale-system-binary class is fleet-wide: erraudit, go-humanize-linter, templ).
33. Add a `check-rows:ledger` note to harvest ledgers in sibling repos before their next archive passes.
34. Mirror this session's archive-manifest pattern into the other repos' next docs-health sweeps (manifest-in-archive-README).
35. TODO_LIST convention call: keep closed-in-place rows (house style) or adopt the skill's delete-on-close (question g3).
36. AGENTS size trim to budget — with a mandate this time (seventh flag).
37. Em-dash copy decision + one mechanical pass (standing BLOCKED).
38. CV bump + `WithNoDatastarRuntime()` evaluation (standing BLOCKED, rollout order is yours).
39. File the three verified upstream drafts (templ-components ×2, erraudit ×1) — authorization pending.
40. Push `static/v0.6.2` in go-datastar (owner act; unblocks dnsblockd).
41. `ListNote` adoption after templ-components#37 moves.
42. Re-check gopls stdversion dismissal on the next gopls upgrade (ROADMAP standing).

**Ideas that surfaced this session (ROADMAP-fuel, not yet routed): 43-50**

43. Session-closing gate could run the archive completeness gate when `docs/status` changed (grep -L + check-rows are one command each).
44. A `docs/status/` README pointing at the manifest + the "keep ≤ N active reports" rotation rule (theme 7 idea, now with a worked example).
45. Annotate-tool pairing helper could be upstreamed into annotate.go as `status-items --from-lines` (the positional stderr/stdout pairing I hand-rolled is a recurring footgun — fleet candidate).
46. Evidence-strip machine contract + persistence (standing BLOCKED, design call).
47. Cross-process browser file-lock (minted in ROADMAP theme 7 today).
48. `dashboard_sse_subscribers` gauge + introspection additions (standing theme-7 ideas).
49. Compose-stack CI policy: at least the build-only job (standing BLOCKED).
50. Post-release API watch: triage feedback on Observe/SourceStatuser/DroppedPatchCount (standing idea).

## g) Questions I cannot figure out myself

1. **v0.12.0: cut now or hold?** Everything is green (`e8eeefa`, CI 7/7; pre-push 8/8 at this session's tip). Hold-for-chromedp/CV means the UI batch + hardening train wait behind a fleet decision; cut-now means those ride v0.13.0. Your cadence call — I did not tag anything.
2. **Annotation depth for the archived 2026-09 batch: acceptable as shipped?** Those files carry section-bucket verdicts anchored to the 09-23 sweep/CHANGELOG (not per-item hashes). Per-item hash archaeology for ~350 items a month past their context is expensive and low-yield in my judgment — confirm, or tell me to deepen a specific file.
3. **TODO_LIST convention: keep closed-in-place rows (house style, provenance-anchored) or migrate to the skill's delete-on-close rule?** I kept the house convention and scored it as a known fitness deviation; if you prefer the skill's rule, the next docs-health pass should prune closed rows instead of annotating them.

---

_Point-in-time snapshot at the post-sweep tree; CI green inherited from `e8eeefa` (not re-watched this session). Report is annotate-only history: subsequent sessions should strike my §b/§c items as they land, not rewrite them (docs-health ANNOTATE)._
