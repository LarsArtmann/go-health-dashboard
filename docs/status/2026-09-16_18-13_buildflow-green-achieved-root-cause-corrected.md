# Status Report — BuildFlow Green Run Achieved, Root Cause Corrected

**When**: 2026-09-16 18:13 CEST · **Repo**: go-health-dashboard @ master (3889cda, 10 unpushed commits) ·
**Scope**: continuation of the 14:55 red→green triage session (see
`2026-09-16_14-55_buildflow-green-run-triage.md` for the earlier half — erraudit 13→0,
hadolint/shellcheck/lychee, skip gates for structure-linter/branching-flow).

**TL;DR**: The session goal is **done**: a complete, untruncated `buildflow --build-mode dev` run is
**green — 68 steps success / 0 failed, exit 0, 35.9s**. Getting there falsified the previous report's
root cause (it was never a dprint-coverage gap; it was a nix source-snapshot race) — the correction is
annotated inline in the old report and landed as a repo-level fix (templ-generate skip) with the
fleet-level fix filed as a BLOCKED row. Ancillary verifications: nolint-audit 4/4 needed after
rebuilding a stale erraudit binary, buildflow DB vacuumed 2.8→2.2 GB, browser suite green.
**Everything local is verified; nothing is on GitHub — 10 daemon commits sit unpushed, so CI has seen
none of today's work.**

---

## a) FULLY DONE (verified, not self-reported)

1. **THE GREEN RUN** — `BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode dev`, full output teed to
   `/tmp/bf-final2.log` (no head/tail filters this time): `v2 results: 68 success, 0 failed`, exit 0,
   35.9s, run ID `20260916-174318-bdeb7d2f`-family. Remaining findings are exactly the documented
   deliberate non-fixes (go-auto-upgrade 26 warnings, go-humanize-linter 3, jscpd 18). nix-build and
   treefmt-check passed; the nix-hash-fix (55×) / nix-build-verify (11×) historical-failure warnings
   reference pre-fix runs only — **0 steps failed in this run**.
2. **Root cause corrected — and proven twice.** Run 072 (the intentional re-failure) showed:
   templ-generate re-raws `view_templ.go`/`page_scripts_templ.go` at 15:19:07; **nix-fmt repaired both
   at 15:19:12** (treefmt output "traversed 154 / formatted 2"; both file mtimes = 15:19:12 — nix-fmt
   runs the project treefmt, NOT bare dprint as the 14:55 report claimed); treefmt-check still failed
   at 15:19:57 on gofumpt diffs of the *already-repaired-away* raw state. Conclusion: a nix command
   overlapping the raw window (nix-flake-update ran 15:19:04–15:19:16) ingested a `git+file` source
   snapshot mid-window and the check built from that stale snapshot. Generator-vs-nix-evaluator
   snapshot race — which also explains why DAG reshuffles changed pass/fail (04E passed, 064 failed).
3. **Repo-level fix landed**: `templ-generate` added to `.buildflow.yml` `skip_steps` with the full
   evidence chain in the rationale comment. Tree is now immutable for the whole pipeline;
   generated-file freshness stays owned by the CI hygiene job + flake apps (both regenerate
   pre-build). This made the 17:43 run deterministic-green.
4. **`erraudit nolint-audit` over the four new `//nolint:erraudit` directives**: **4 directives,
   4 needed, 0 stale, exit 0** — every suppression is live and necessary. Required rebuilding the
   stale `~/go/bin/erraudit` first (old binary failed with a bogus "go: updates to go.mod needed"
   package-load error; fresh HEAD build works).
5. **Two tool-semantics traps documented** (AGENTS.md "BuildFlow-adjacent tool traps" gotcha):
   `nolint-audit` takes a DIRECTORY (`./...` doesn't exist as a dir → best-effort walk → silent
   "No directives found"), and stale erraudit binaries fail with misleading go-mod errors.
6. **buildflow DB vacuumed**: 2.8 GB → 2.2 GB via `nix run nixpkgs#sqlite` (sqlite3 not in PATH —
   the doctor's suggested command doesn't run on this host as-is).
7. **TODO_LIST harvested**: new 🔵 BLOCKED row "Unskip templ-generate in BuildFlow (fleet DAG
   ordering)" with the corrected root cause, evidence pointers, and the tied open decision.
8. **AGENTS.md updated**: templ-skip gotcha (with the mtime-based proof narrative) + tool-traps
   gotcha (concurrent-tooling ban, nolint-audit arg semantics, stale-binary class).
9. **14:55 report annotated inline** (docs-health ANNOTATE style): TL;DR closure, b1 green-run
   evidence, b4 root-cause correction (explicitly marked ❌ WRONG with the replacement mechanism),
   c1/c2/c4 resolved, footer closed. 7/7 edits verified present.
10. **Browser suite green end-to-end**: `nix develop -c go test -run TestBrowser` → ok 11.4s in real
    headless Chrome — the erraudit refactor (writeBody, encode-before-commit, close/drain
    suppressions) didn't disturb rendering, CSP, or live SSE.
11. **Daemon-sweep integrity check**: `.buildflow.yml` (templ-generate entry) and AGENTS.md (both new
    gotchas) verified present in `HEAD:` blobs after the daemon committed them; only the report
    annotations were pending at last check (daemon will sweep).
12. **Doctor re-run classified**: the "Failed: 34" headline is host tool-availability noise (bandit,
    cargo-*, eslint, dprint … — irrelevant to this Go repo; the pipeline nix-falls-back where it
    matters, e.g. dprint and shellcheck ran fine). Recorded as environmental, not repo debt.

## b) PARTIALLY DONE

1. **CI verification of today's work** — blocked, not done: `git status -sb` revealed master is
   **ahead 10 commits** (all daemon heuristic commits since ~14:00). CI triggers on push; no runs
   exist after 11:35. Until a push happens, GitHub has zero knowledge of the erraudit fixes, the
   array-typed histogram, both skip gates, or the docs. Last observed CI on master: success at 11:35
   (3-file commit); an 11:10 failure (8-file commit, `ba6fb6d`) was the session's *starting* state.
2. **Fleet-level fix for the snapshot race** — root cause now correct, fix designed (order all
   tree-mutating generators before all nix-evaluating steps; optionally templ-generate
   skip-if-unchanged), not implemented anywhere: local patch vs upstream issue is open question 3.
3. **Doctor noise reduction** — classified (see a12) but not fixed: the few tools that would matter
   (govulncheck, go-licenses, shellcheck in PATH) are still not installed host-wide.
4. **Detect-only advisory waivers** — go-humanize-linter (3) and jscpd (18) are documented as
   deliberate non-fixes in AGENTS.md, but no machine-readable waiver exists (jscpd/humanize have no
   per-project config path in BuildFlow), so they will resurface in every summary forever.
5. **The three session questions** — unchanged and now more pointed (see g).

## c) NOT STARTED

1. **Push + CI green run on today's commits** (needs authorization).
2. **BuildFlow generator/nix ordering change** (fleet repo, needs question-3 answer).
3. **templ upstream issue**: verify current templ still emits non-gofumpt generated Go; file or
   pin-note. (Not started — today's evidence makes it *more* fileable: the raw window is the hazard.)
4. **erraudit upstream filings**: (a) `strings.Cut` blank-identifier false positive; (b) nolint-audit
   accepting `./...` while silently scanning nothing (DX trap); (c) doctor-style binary-freshness
   check for erraudit itself.
5. **`~/go/bin` stale-tool prune** (only erraudit was rebuilt; structure-linter & co unchecked).
6. **`buildflow timings --regressions` baseline** after the green run.
7. **gitleaks + codespell** on-demand runs (skipped by every build mode by design).
8. **jscpd disposition** (extract shared test scaffolding or explicit per-file waiver).
9. **README screenshot captures** (`SCREENSHOT_OUTPUT=docs/screenshot.png`, env-guarded) — not run
   since the handler refactors.
10. **Metrics histogram regression test** asserting bucket-count == len(bounds) (guards the
    array-typing fix from the earlier half).
11. **Example server shutdown-path exercise** (`DEMO_AGGREGATE` + SIGTERM → observe the new
    shutdown-report log line).
12. **Post-fix stability watch**: confirm the daemon's future sweeps don't re-fossilize raw
    `*_templ.go` now that buildflow no longer regenerates mid-run.

## d) TOTALLY FUCKED UP

1. **I ran repo tooling concurrently with an active buildflow run — twice.** First `erraudit
   nolint-audit ./...` (15:20-ish, during run 072's go-mod steps) produced a bogus "go: updates to
   go.mod needed" that I initially misattributed to a loader-env bug; the true cause (torn go.mod
   reads during buildflow's go-mod-update) only became clear when the failure repeated *after* the
   run finished. The second concurrency (nolint-audit during the background rerun) I caught before
   damage. The rule "never run Go tooling during a buildflow run" is now in AGENTS.md — earned the
   embarrassing way.
2. **I audited the wrong repository.** Running `erraudit nolint-audit .` immediately after `cd
   ~/projects/erraudit` (to rebuild it) audited erraudit's own 23 directives; I briefly read "23" vs
   the earlier "4" as a workspace-vs-plain build-behavior difference before ground-truth counting
   (`rg -c '//nolint'`) exposed that the counts came from different repos. A `pwd` check would have
   cost two seconds.
3. **The 14:55 session's root cause was wrong and I planned on it.** "nix-fmt = dprint with no Go
   coverage" was plausible, matched the log narrative, and was wrong: file mtimes proved nix-fmt
   (treefmt) repaired the files 45 seconds *before* the check failed. The evidence needed —
   `stat` on the two files — costs one command and was available from run 064's artifacts. I
   formulated fleet-fix options (Q3, DAG redesign) on top of the wrong mechanism before falsifying
   it. Lesson now written down: in any ordering race, `stat` the files before theorizing about logs.
4. **CI blindness accumulated all session unnoticed.** Ten commits piled up unpushed while I planned
   "confirm CI green on daemon commits" (f8 in the 14:55 report). The check that surfaced it
   (`git status -sb`) was one command and should have preceded any CI claims. The systemic gap is
   bigger than me: the daemon commits continuously and never pushes, so local green + remote red/gone
   can diverge indefinitely with nothing flagging it.
5. **Carried from the earlier half (still unfixed systemically)**: verification pipelines masked
   verdicts (`head -40` truncation of run 069); the daemon fossilized raw generated files twice; and
   a config-only change (skip_steps) flipped a passing step to failing without a re-verification
   ritual. Items 1–3 above are this continuation's fresh instances of the same classes.

## e) WHAT WE SHOULD IMPROVE

1. **Serialize all repo tooling against buildflow runs.** Written into AGENTS.md; should also be a
   personal checklist item: one mutation pipeline at a time, period.
2. **`pwd` before every tool invocation that takes a path** — the wrong-repo audit was pure cwd
   sloppiness.
3. **File-system evidence before log narrative.** In ordering/race investigations, `stat` the
   artifacts first; logs describe intent, mtimes describe reality.
4. **Read `--help` before first use of any subcommand** — `nolint-audit`'s directory-vs-pattern
   semantics silently no-op on `./...`.
5. **Treat fleet binaries like buildflow does**: doctor checks buildflow's own freshness; erraudit
   (and friends in `~/go/bin`) have no equivalent — a stale analyzer fails *weirdly*, not loudly.
6. **Check sync state before remote claims**: `git status -sb` (ahead/behind) before any "CI is
   green" statement.
7. **Root-cause corrections must chase their downstream dependents** — correcting b4 in the old
   report wasn't enough; the TODO row, skip rationale, and question-3 framing all had to be re-based
   on the corrected mechanism (done, but only because I re-read the report against the new evidence).
8. **Report-format override honesty**: the status-report skill's canonical output is a styled HTML
   dashboard; the user's explicit `.md` instruction wins for this report, and the divergence is
   flagged here rather than silently normalized.

## f) NEXT — up to 50, grouped (brainstorm, not commitment; bounded items are TODO_LIST fuel)

**A. Repo — close today out**

1. Push master (10 commits) once authorized; watch the CI run land green on the erraudit/metrics/
   skip-gate state. _(Needs user authorization — CRUSH forbids unrequested pushes.)_
2. Re-run `buildflow --build-mode dev` after the daemon's latest commits (report annotations landed
   post-17:43) to confirm green is stable, not a one-off.
3. Answer g-questions 1–3 (they gate five rows below).
4. Verify templ's current output against gofumpt; file the upstream issue (raw window as hazard) or
   pin-note in AGENTS.md.
5. File erraudit upstream: strings.Cut FP; nolint-audit `./...` silent no-op; freshness-check idea.
6. Prune `~/go/bin`: audit each tool against its repo HEAD (erraudit done; structure-linter & co).
7. `buildflow timings --regressions` baseline of today's fixes.
8. `buildflow -s gitleaks` + `-s codespell` once (on-demand-only steps, never yet run here).
9. jscpd 18 findings: extract or explicitly waive test scaffolding per file.
10. go-humanize-linter 3 findings: document the zero-deps relative-time helpers as the answer.
11. Install the four doctor-relevant host tools (govulncheck, go-licenses, shellcheck, dprint) or
    document per-host expectation.
12. README screenshot captures via the env-guarded suite.
13. Add the bucket-count == len(bounds) regression test.
14. Exercise the example shutdown-logging path (DEMO_AGGREGATE + SIGTERM).
15. Watch the next daemon sweeps for re-fossilized raw `*_templ.go` (should be gone with
    templ-generate skipped); revert to investigating if any appear.

**B. BuildFlow (fleet) — root-cause fixes**

16. DAG ordering: every tree-mutating generator must complete before any nix-evaluating step
    (flake-update, flake-check, build, hash-fix, vulnix…). Regression test asserting the order.
17. templ-generate: skip-if-generated-content-unchanged (kills mtime churn at the source).
18. nix source snapshot hygiene: re-ingest `git+file` sources after any step that mutated the tree
    (or snapshot once, up front, and use that tree for all steps — the deterministic design).
19. Result-cache keys must include each tool's project config file (`.go-structure-linter.yaml`
    replayed stale findings until `BUILDFLOW_NO_RESULT_CACHE=1`).
20. Surface per-project config to in-process SDK tools (go-auto-upgrade ignores
    `.go-auto-upgrade.json`; branching-flow's `.branching-flow.yml` bypassed by `RunAll`).
21. go-auto-upgrade: per-migrator include/exclude plumbing into `registry.All()`.
22. Findings-gate semantics: document whether detect-only error-severity findings gate (behavior
    drifted between binaries; the ✗-vs-warning display mismatch observed today).
23. `buildflow history`: expose per-run exit verdict separately from step percentage.
24. Auto-suggest skip lines in the summary for 100%-failing steps (nix-hash-fix class).
25. Doctor: distinguish "tool missing but nix-fallback works" from "tool missing, step will fail".

**C. go-structure-linter (fleet)**

26. Cut the release carrying `LoadProjectConfig` in `Lint()` (checkout is ~95 commits past v0.10.0).
27. Bump BuildFlow's pin; `.go-structure-linter.yaml` flips inert→effective; remove repo skip row.

**D. branching-flow (fleet)**

28. Honor `//nolint:branching-flow[:rule]` in phantom + boolblind analyzers.
29. Rule exclusions / severity overrides in `.branching-flow.yml`.
30. Route provider options through project config (kill the hardcoded `RunAll`).
31. Revisit phantom-type severities (critical/error defaults make released-API adoption impossible).

**E. Repo — public API (needs the user's versioning decision)**

32. Phantom-type campaign on the option surface as v0.9.0 (breaking) — adopt vs tune decision.
33. Bit-flag decision for Config/introspectModes bools (recommendation: keep named bools).
34. `record(ok, …)` FLAG_PARAM parameter-order fix (webhook stats).
35. statusTransition/jsonTransition composition note or mixin.
36. DO_service-locator sanction note for the example composition root.

**F. Repo — quality debt, small and bounded**

37. Link struct-size rationales (Config 32 / viewModel 29 / pusher 17 fields) to the
    branching-flow suppression decision.
38. Cross-link the three TODO Blocked rows to the fleet repos' own TODOs.
39. Close-the-loop ritual: when B16–B18 land, re-verify both skipped tools AND the ordering in one
    run, then delete all three skip entries.
40. Consider `--strict` dry-run in the pre-change checklist (would have flagged the 064-class
    regression before it shipped).
41. Decide the daemon/push gap systemically (see g-2): a push step, a reminder, or documented
    "local-only until manual push" policy.

**G. Upstream / ecosystem**

42. Fleet wiki/AGENTS entry: "templ projects in BuildFlow" — the full source-snapshot-race anatomy.
43. dprint: document that its pass can never cover Go (no plugin) so formatter claims must name
    treefmt explicitly.
44. go-auto-upgrade: a project-level "no-new-deps" signal instead of eternal warnings.
45. Re-check gopls stdversion dismissal on the next gopls upgrade (existing ROADMAP note, untouched
    today — listed so it doesn't rot).

**H. Hygiene**

46. Rotate `/tmp/bf-final.log` + `/tmp/bf-final2.log` into `docs/status/` artifacts or delete (they
    are the evidence chain for a2/a3).
47. After push: annotate AGENTS.md release-discipline with the daemon-never-pushes fact (it changes
    what "verify on CI" means locally).
48. Session-stats: confirm the doctor's binary-freshness warning class (e0dd63a binary vs BuildFlow
    HEAD drifts again with every daemon commit there — decide who rebuilds and when).
49. Template note: status reports in `.md` (user override) vs the skill's HTML default — pick one
    canonical for `docs/status/`.
50. When TODO_LIST harvests f-items, apply docs-health routing rigor: bounded → TODO_LIST; ideas →
    ROADMAP; fleet items → their repos, not this one.

## g) Questions I cannot answer myself

1. **Push authorization**: master is 10 daemon commits ahead with the entire session's verified work
   (erraudit fixes, array-typed histogram, all three skip gates, docs) and CI has seen none of it.
   Push now and verify the CI run, or is there a pre-push ritual I don't know about?
2. **The daemon's push contract** (extends the 14:55 g-2): it commits continuously but never pushes —
   is that deliberate (you push manually at milestones) or a gap? It decides whether "green locally +
   CI-blind" is an acceptable steady state or a standing incident.
3. **BuildFlow fix delivery for the snapshot race** (14:55 g-3, now with corrected mechanism): patch
   BuildFlow locally + reinstall today (fast, you own the repo, unblocks the templ-generate skip
   eventually), or file the issue upstream and hold the fleet fix for a release?

---

_Point-in-time snapshot, 18:13 CEST. Local state: all gates green, evidence teed; remote state: stale
by 10 commits. Written as `.md` per explicit user instruction (skill default is styled HTML)._

**WAITING FOR INSTRUCTIONS.**
