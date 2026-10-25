# Status + Brutal Self-Review — Dedup pass: 18 fixture tails → `startDashboardFixture` + pinned clone-baseline gate (2026-10-25, 17:53)

| | |
| --- | --- |
| **Session** | Triggered by user directive READ/UNDERSTAND/RESEARCH/REFLECT, execute-and-verify stepwise, "deduplciate!" — the `deduplicate-code` skill ran art-dupl, every clone group was judged, harmful duplication was extracted, and acceptance was turned into an enforced gate |
| **Window** | ~16:15–17:55 CET (single session; auto-commit daemon active throughout) |
| **Start state** | Clean Go tree at `b895cc6` (docs-only reorg staged by a prior session); CI 7/7 green per the 14:19 session handoff; art-dupl 0.9.0-1b165cd at `/run/current-system/sw/bin/` |
| **End state** | All work daemon-committed (10 heuristic commits); tree holds only this report + AGENTS/docs memory edits; **full test suite green (incl. 21 serialized browser tests), race suite green, `go vet` clean, golangci-lint 0 issues, pre-push checks 9/9 green** |
| **Net code delta** | **−235 lines** (122 insertions / 357 deletions) across 8 test files + the gate script |
| **Format note** | HTML is the status-report skill's canonical format; the user explicitly demanded `.md` at this path — honored as a one-off override, not propagated into the skill |

---

## §0 Brutal self-review

### What did I forget?

1. **I wrote wrong numbers into durable memory docs.** AGENTS.md and docs/gotchas.md said "16 hand-rolled fixture tails across 7 files" — the real count, verified by `rg -c "startDashboardFixture(t,"` at report time, is **18 sites across 8 files**. My final chat message repeated "16". Caught ~30 minutes later while writing THIS report; corrected in both docs before publishing it. One grep at extraction time would have prevented it. Lesson: counts in memory files come from `rg -c`, never from the session's sense of accomplishment.
2. **I never ran `nix flake check` or `nix build .#health-hub` this session.** The templ LSP re-rawed `view_templ.go`/`page_scripts_templ.go` mid-session (known gotcha) and I restored the canonical shape via `go tool templ generate` → `nix fmt` — but the flake packaging path consuming those files was never exercised locally. Justification (the machine's nix-build steps fail on proxy DNS per gotchas, CI owns flake builds) is real, but "justified skip" and "forgot" look identical in the handoff unless written down. This is the write-down.
3. **I re-derived an already-written policy.** The 2026-09-23 dedup session had already reserved test-scaffolding as a documented non-fix ("per-test isolation preferred") and left a triage bucket note. I didn't read that status file at handshake; I rediscovered the policy mid-analysis by grepping gotchas. The AGENTS.md handshake recipe (`ls docs/status/ | tail`) exists precisely for this and I ran it — but only looked at the newest file, not for prior sessions *on the same topic*.
4. **I left an asymmetry undocumented.** `TestMetrics_PrometheusParserConformance` builds a probe, registers routes, and starts ONLY the probe — never the dashboard. I classified it "not convertible, accept" (correct — the metrics endpoint needs no pusher) but wrote no comment at the site. The next dedup auditor will re-derive the same judgment from zero.
5. **Redundant verification spend.** The full 21-test browser suite ran to completion twice (once via `nix run .#test`, once explicitly to prove no skips). The second run was justified once (skip- paranoia); the first run's 5s root-package time was actually plausible given ~155ms Chrome launches. Harmless, but it was luck, not budgeting.

### What is something that's stupid that we do anyway?

1. **The gate I just wrote can fail open.** Check 9 prints "art-dupl not on PATH — skipping clone-baseline gate" and continues green if the binary is missing. A gate whose failure mode includes silent success is a tripwire painted on paper. art-dupl is not in the flake devShell; it only exists because the system PATH has it.
2. **~30 call sites still ritualize `s := startDashboardFixture(...)` + `defer s.cleanup()`** when the helper could register `t.Cleanup` itself. Each new fixture repeats a two-line ceremony whose omission is silent.
3. **Three documents now describe the same gate** (baseline report = evidence, docs/gotchas.md = incident, AGENTS.md = policy pointer). Intentional layering, but each retelling is a drift surface — the baseline report's embedded calibration scan already contains line numbers that rot on the next edit.

### What could you have done better?

1. **I hand-rolled a text parser for the gate when the tool ships machine output.** art-dupl has `--plumbing` and `--json` flags I never inspected; instead I grepped human-readable text, whose wording *provably varies by flag combination* ("Detected N groups, X shown" vs "Found total N"), and the first version of my gate silently killed the whole pre-push script via `grep` no-match under `set -euo pipefail`. Caught by running the gate (exit 1, zero output), fixed by counting `^found N clones:` markers with `|| true` — but the correct move was checking the machine-readable interface first.
2. **`replace_all` without counting occurrences first.** My screenshot_test.go replacement hit a third `for dash.SubscriberCount() == 0` line in a test I had deliberately NOT converted (the multi-probe aggregate test). `go vet` caught the undefined `s` within a minute and a line-scoped `sed` fixed it, but the edit tool literally asks for uniqueness and I chose force over verification.
3. **I batched conversions before verifying any.** The first full-suite run landed after only 6 of 18 conversions; green there proved the helper, not the migration. Safe in the event, but the honest sequence is convert-one → verify → convert-the-rest, and I only did it by accident of timing.
4. **Numbers from memory, not from the tree** (see forget #1). This is the same failure class the repo already has a name for: unverified claims encoded into durable surfaces.

### What could you still improve?

1. **Mutation-verify the extracted contract once**: flip the cleanup order inside `startDashboardFixture` and watch the suite catch it. The repo's own gotchas warn that mutation checks race the daemon and must probe the working tree directly — doing this properly is a 10-minute follow-up that converts "tests pass" into "the test discriminates".
2. **Promote the gate out of fail-open**: install art-dupl via the flake (or a flake app wrapper) so check 9 either runs or fails loudly; alternatively ship a `--plumbing`-parsed CI job.
3. **Close the loop cross-repo**: go-etag's `pre-release-check.sh` greps the same human summary line under `set -e`; its baseline capture happens to contain a "shown" line today, so it works — but it is the identical fragility class I just fixed here. I noticed it; noticing is not a terminal state (the 14:19 session's own confession, now mine too).

### Honesty checklist

- **Did I lie?** No. Every green claim in the closing message is reproducible: pre-push gate exit 0 (9/9), race suite `ok`, lint "0 issues", −235 from `git diff --stat`. The one inaccuracy (16 vs 18 tails) was a miscount, not a fabrication — and is corrected in the same breath as this report.
- **Ghost systems?** None created. The gate is wired into `pre-push-checks.sh` (positive AND negative tested); the baseline report is committed; the helper has 18 live callers.
- **Split brains?** Two created and flagged: the doc triple above, and the pre-correction count discrepancy between my chat message and the docs. Code-wise, `startDashboardFixture` is now the single source of the start/shutdown contract — that was the point.
- **Scope creep?** Held. The ~50 test-scaffolding groups stay accepted per the documented non-fix; `setupDashboardWithProber` and `setupSSEServer` keep their deliberately different lifecycles untouched.
- **Removed something useful?** No. Every deleted block is byte-preserved inside the helper. One behavioral delta, strictly toward correctness: all 18 fixtures now call `probe.Shutdown()` in cleanup (the evidence fixture previously didn't — that was the drift bug).
- **Tests?** Suite + race + browser green, lint/vet clean, gate double-tested. The helper itself has no direct test — it is scaffolding covered transitively by all 18 consumers, same acceptance the 09-23 session recorded.

---

## a) FULLY DONE

1. **Clone inventory**: art-dupl type-aware scan at `-t 5` — 57 groups (2 production-shown, 55 suppressed); every suspicious pair read and judged, including the five largest test blocks.
2. **`startDashboardFixture` extracted** (dashboard_test.go:67): 18 hand-rolled `probe → dashboard.New → mux → RegisterRoutes → Start×2 → cleanup` tails across 8 files now call one helper; net **−235 lines**.
3. **Real bug fixed by the unification**: `setupAllFailing` (evidence_integration_test.go) cleanup forgot `probe.Shutdown()` — the must-change-in-N-places failure class, caught because N collapsed to 1.
4. **Judged-and-accepted set documented**: `mapStatusToBadge`/`timelineBadgeType` (different business rules, same 2-token return type), per-test scaffolding (documented non-fix), `setupDashboardWithProber`/`setupSSEServer`/bench/metrics-parser sites (different lifecycle contracts).
5. **Acceptance became a gate**: `pre-push-checks.sh` check 9 — `art-dupl --type-aware --sort total-tokens -t 5 --ignore-tests .` must show EXACTLY 1 production group; baseline report committed at `reports/dupl/2026-10-25_art-dupl-baseline.txt`; gate positive- and negative-tested (wrong pin → red with actionable message, exit 1).
6. **Pipefail silent-kill bug in my first gate version found and fixed** (grep no-match under `set -euo pipefail` killed the script with no output); parser now counts `^found N clones:` markers.
7. **Verification sweep**: full suite (incl. 21 serialized browser tests) green, race green, `go vet` clean, golangci-lint 0 issues, pre-push 9/9 green.
8. **Templ generated files restored to canonical shape** after the templ LSP re-rawed them mid-session (canonical `go tool templ generate` → `nix fmt`; HEAD had captured the raw shape).
9. **Memory updated**: AGENTS.md testing-patterns bullet (helper contract + policy), AGENTS.md gotcha (gate, parser trap, pass stats), docs/gotchas.md incident section — with the 16→18/7→8 count correction applied.

## b) PARTIALLY DONE

1. **Gate coverage**: desk-only. Not in CI; silently skips when art-dupl is off PATH (fail-open branch written by me, flagged in §0).
2. **Dedup program**: production duplication pinned at exactly 1 group; the 53 full-suite test-scaffolding groups are accepted but unguarded by design — no periodic re-audit convention recorded beyond gotchas prose.
3. **Intent history**: the refactor lives in 10 `chore: auto-commit` daemon commits; no intent-labeled commit exists (harness forbids unrequested commits; amend-relabel recipe is available if you want it).
4. **Templ system skew**: I recovered the tree (canonical generate), but the root fix — NixOS system templ package v0.3.1020 vs pin v0.3.1070, which also feeds the editor LSP's persistent false squiggle — remains open from the 13:38 session.

## c) NOT STARTED

1. **HARVEST of section (f)** into `TODO_LIST.md`/`ROADMAP.md` — deliberately withheld until you instruct (you asked report-then-wait; the skill's close-the-loop rule says this report's (f) is HARVEST input).
2. **Mutation probe of the fixture contract** (cleanup-order flip → expect failures).
3. **`nix flake check` / `nix build` verification** of this session's tree (deferred to CI per machine gotcha — see §0 forget #2).
4. **Upstream art-dupl filing**: summary-line wording instability across flag combinations; confirm/adopt `--plumbing` for stable counts.
5. **go-etag parse hardening** (same fragile-grep pattern, currently working by luck of wording).
6. **Comment at `TestMetrics_PrometheusParserConformance`** recording why the dashboard is intentionally never started there.

## d) TOTALLY FUCKED UP

Nothing at disaster level. The three genuine defects of the session, all caught and fixed locally before any push:

1. **Fail-silent gate**: first gate version aborted the entire pre-push script with exit 1 and zero output (pipefail + grep no-match). Caught by running the gate; fixed; negative test added. If I had only eyeballed the diff, every future "gate green" would have meant "script died at check 9" — that is the near-miss of the session.
2. **`replace_all` over-reach** into an unconverted aggregate test (screenshot_test.go:408) — caught by `go vet`, fixed line-scoped.
3. **Wrong counts (16/7) encoded into AGENTS.md + docs/gotchas.md** — durable surfaces, ~30 minutes of lifetime, corrected at report time.

## e) WHAT WE SHOULD IMPROVE

1. **Gates must fail loud**: no skip-when-tool-missing branches in `pre-push-checks.sh`; install the tool or fail.
2. **Machine-readable over human-text parsing** for any tool with a `--plumbing`/`--json` mode; if none exists, pin the parser to stable markers and negative-test it (now done for check 9).
3. **Counts in memory docs come from `rg -c`**, never from recollection — this session produced the counterexample.
4. **Topic-aware handshake**: when a session matches a prior session's topic (dedup ← dedup), read that prior status file during handshake, not just the newest one.
5. **Single-source the gate rationale**: AGENTS.md keeps the two-line policy pointer; docs/gotchas.md keeps the incident; the baseline report keeps evidence — and the other two stop restating numbers that rot.
6. **Consider `t.Cleanup` inside `startDashboardFixture`** to delete the `defer s.cleanup()` ritual at ~30 call sites (tradeoff: implicit LIFO interleaving with test-local defers — needs a deliberate decision, not a drive-by).

## f) Up to 50 things we should get done next

*Provenance tags: [S] observed in this session, [C] carried context from AGENTS/TODO_LIST/prior-session handoffs. Sorted by impact within groups; most are small by design — a brainstorm, not a commitment list (HARVEST applies routing rigor).*

**Gate & tooling hardening**

1. [S] Install art-dupl via flake devShell (or flake app) so check 9 cannot silently skip — kills the fail-open branch.
2. [S] Repoint check 9's parser at `art-dupl --plumbing`/`--json` after verifying the schema; keep the `^found` fallback.
3. [S] Mirror check 9 as a CI job (or extend an existing lint job) so the gate holds remotely, not just at the desk.
4. [S] File upstream art-dupl: summary-line wording varies with flag combinations; request a stable count field (verify-before-filing: reproduce both wordings first).
5. [S] Harden go-etag's `pre-release-check.sh` clone parse (same grep-under-`set -e` fragility; works today only because its wording happens to match).

**Dedup continuation**

6. [S] Mutation-verify `startDashboardFixture`: flip cleanup order in the working tree, expect suite failures, restore via sed-from-HEAD (daemon-safe recipe).
7. [S] Decide `t.Cleanup` vs returned-cleanup for the fixture (deletes ~30 `defer s.cleanup()` lines; LIFO tradeoff).
8. [S] Annotate `TestMetrics_PrometheusParserConformance` with why `dash.Start` is intentionally absent.
9. [S] Record a quarterly (or per-major-refactor) full-suite clone calibration review: re-judge the 53 accepted groups, update the baseline report snapshot.
10. [S] Teach the fleet: cross-project lesson — "cleanup-contract drift is found by collapsing N copies to 1" — to crush-config `references/lessons.md` (commit there, not in-session writes).
11. [C] go-structure-linter stays skip-gated until the fleet bump lands (TODO_LIST Blocked row) — unskip the committed `flat` preset when BuildFlow ships project-config support.
12. [C] jscpd/branching-flow advisories: keep as documented non-fixes; revisit only if fleet tooling changes (standing triage-bucket row).

**Verification debt**

13. [S] Run `nix flake check` + `nix build .#health-hub` on the canonicalized generated files (locally or confirm CI green on the daemon commits).
14. [S] Confirm CI green on the exact daemon-commit chain this session produced (release discipline: never claim unpushed/unverified state — `gh run list` before anyone pushes).
15. [S] One-off: run the fuzz seed suite (`go test -run Fuzz`) after test-infra refactors as cheap regression insurance — skipped this session as unrelated, worth a policy line.
16. [C] Keep LOADTEST env-parameterized rerun on the agenda before the next release that touches pusher/history code.
17. [C] gopls stdversion re-check on next gopls upgrade (ROADMAP carried item).
18. [S] Add `check 9` output line to the release-session checklist in AGENTS.md release discipline (it now runs in pre-push; say so where release sessions read).

**Fleet / upstream filings**

19. [C] File securitymd `filesScanned: 0`-alongside-findings reporting bug (owed from 14:19 session, never filed).
20. [C] Reproduce-and-file nix-hash-fix double-WARN "stale hash not found verbatim" before self-heal (owed from 14:19 session).
21. [C] Root-fix system templ skew: bump NixOS templ package to v0.3.1070 so CLI + editor LSP match the go.mod pin (kills the view.templ:168 false squiggle permanently).
22. [C] Annotate the 13:38 templ-skew status report: its owed gates (tests/lint/flake) were paid by the 14:19 session — one line, docs-health ANNOTATE.
23. [S] Verify go-etag's gate still matches its AGENTS "Non-Obvious Behaviors" claim after any art-dupl upgrade (fleet-wide version-pin discipline for clone gates).

**Docs & docs-health**

24. [S] HARVEST this report's (f) into TODO_LIST/ROADMAP with routing rigor (most items are brainstorm-grade).
25. [S] Trim the doc triple: gotchas.md keeps the incident; AGENTS.md keeps policy; baseline report keeps evidence — remove restated numbers where they rot.
26. [S] Mark the baseline report's calibration section as a dated snapshot (it embeds line numbers that rot on the next edit).
27. [S] Decide CHANGELOG policy for internal/test-only refactors (this session added none; the structural lint neither demands nor forbids it — the policy is unwritten).
28. [C] FEATURES.md test-count row: unchanged this session (helper is not a `Test` func) — recheck after any fixture-ergonomics change (item 7) since it can alter counted helpers if naming shifts.
29. [S] Update the 09-23 dedup session's reserved triage-bucket note: bucket now partially spent (18 tails extracted; scaffolding still reserved).
30. [C] AGENTS.md dependency-notes go-health section: no changes needed this session — re-verify at next go-health bump.

**Repo hygiene**

31. [S] Relabel the 10 daemon heuristic commits covering this refactor into one intent-bearing commit (amend-relabel recipe; needs your go-ahead — see question 2).
32. [S] Remove the stale `./result` symlink (carried warn from 14:19 session; one command).
33. [S] Sweep for other gates that parse tool stdout under `set -e` without `|| true` (the check-9 bug class, applied to checks 1–8 — audit once).
34. [S] `scripts/pre-push-checks.sh`: print the art-dupl version in check 9 output (baseline + binary version drift visibility).
35. [C] Keep `--exclude nix-hash-fix --exclude nix-build-verify` in local BuildFlow runs until the machine's proxy DNS refusal is fixed (standing friction).
36. [S] Add the negative-test idiom ("run the gate against a deliberately wrong pin once") to the pre-push script's header comment so future gate authors repeat it.
37. [S] treefmt/gofmt: no formatting debt introduced; keep `nix fmt` after hand edits as standing rule (validated this session).

**Test-suite ergonomics (small, opinionated)**

38. [S] Extract `connectSSE`+`waitForSubscriber`-style pairing into one `mustConnect(t, s)` where 5+ sites repeat the pair (survey first — only if the count is real).
39. [S] Give `probeSetup` a doc comment stating the cleanup order contract (dashboard first, then probe) so callers stop re-deriving it.
40. [S] Name the fixture-builder convention in AGENTS.md: provisioning stays per-helper, lifecycle always via `startDashboardFixture` (already enforced socially; write it as rule — partially done, see item 25).
41. [C] Browser-suite serialization (`browserSerial`): revisit only if Chrome-launch latency regresses (155ms measured; fine).
42. [C] Watchdog/rate-limit wall-clock margins: leave at current budgets (two 2026-10-11 flakes documented).

**Carried backlog (context only, not researched this session)**

43. [C] v0.12.0 release cadence question from the 14:19 session (open decision).
44. [C] Second-session-during-release policy question from the 14:19 session (open decision).
45. [C] Pin-guard: next UI dependency climb still requires dedicated change + green browser suite + same-change guard updates (standing rule).
46. [C] Federation upstream gaps (no `MarkShuttingDown`, no `SourceStatuses`) — upstream proposals remain unfiled.
47. [C] `Ready()`-proves-nothing gotcha: startup-gate polling pattern is documented; no action unless a consumer hits it.
48. [C] Metrics output byte-stability tests: fine as-is (strip-histogram comparison); no work queued.
49. [S] Re-run `bash scripts/pre-push-checks.sh` as the FIRST step of the next release session (standing rule; re-stated because check 9 is new).
50. [C] Example/deploy stack: no changes this session; digest-pin + port-collision traps remain documented in deploy/README.md.

## g) Questions I cannot figure out myself

1. **Gate placement policy**: should the clone-baseline gate stay desk-only (pre-push) or become a CI job? The technical blocker is art-dupl availability in CI runners (install method/permission), but the real question is policy: is a desk gate acceptable enforcement for duplication, or must it hold remotely like lint?
2. **History relabel**: this refactor is spread across 10 `chore: auto-commit` daemon commits. Do you want me to amend-relabel them into one intent-bearing commit per the documented daemon-race recipe (rewrites unpushed history — I have not verified origin sync state), or is heuristic daemon history acceptable to you?
3. **Test-scaffolding gating philosophy**: production-only pinning (current, stable) or go-etag-style exact-count over ALL files (stricter, but every added `t.Parallel()` test turns the gate red until the baseline is deliberately updated)? I cannot infer your tolerance for gate churn from the repos alone.

---

*WAITING FOR INSTRUCTIONS.* — Report written per your explicit `.md` path demand (status-report skill's canonical format is styled HTML; overridden once, flagged here). Per harness rules I did not commit: the auto-commit daemon will pick this file up.
