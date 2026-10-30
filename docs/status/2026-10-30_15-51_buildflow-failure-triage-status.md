# Status Report — BuildFlow Failure Triage (paste: templ-fmt timeout + FuzzCSVExport seed#13)

**Date**: 2026-10-30 15:51 CET
**Branch**: `master` @ `0a59af4` (fix: fuzz CSV round-trip assertion, cut v0.12.2) · tree clean
**Session scope**: Triage the two failed BuildFlow steps from the user-pasted run; verify everything green at HEAD; no code changes made or needed.

---

## Executive summary

Both pasted failures are **resolved at HEAD without new code**:

1. **`test-race` (FuzzCSVExport/seed#13, status `"\t"`)** — the pasted failure is **stale**. The pasted run executed at 03:51 today; the fix commit `0a59af4` landed at 05:04 today. The failure message itself pointed at `fuzz_test.go:382`, but at HEAD that assertion lives at line 389 — the fix (accepting the formula-neutralized `"'"+status` round-trip form; tab is a CSV formula starter per `csvFormulaStarters`, trend.go:254) moved and corrected it. Verified three independent ways: 16/16 fuzz seeds pass, `buildflow -s test-race` ✔, and an **uncached** full race suite (`go test -race -count=1 .` → ok 25.986s).
2. **`templ-fmt`** — **transient spawn starvation**, not a defect. BuildFlow itself classified it `[transient:tool.timeout]` ("killed at 1m0s, 23x its p95 of 3s … no output at all"). At failure time the host 5-min load average was 37 (it was still ~16 at session start, 19 chromium processes, 26 users). Rerun: Format Complete, count=1, errors=0, changed=0; step ✔ 1/1.
3. **Full local gate**: `buildflow --build-mode dev --exclude go-mod-update --exclude nix-hash-fix --exclude nix-build-verify` → **59/59 steps, exit 0**; working tree clean, go.mod/go.sum untouched (the trap-4 post-run check).

---

## a) FULLY DONE

| # | What | Evidence | Scope |
|---|------|----------|-------|
| 1 | Diagnosed FuzzCSVExport/seed#13 as a stale failure predating the fix commit | `git log -1 --format='%ci' 0a59af4` → 2026-10-30 05:04; failed run 20261030-035152 → 03:51; failure line 382 vs HEAD line 389 | diagnosis only, no files |
| 2 | Verified all 16 fuzz seeds pass at HEAD | `nix develop -c go test -v -run FuzzCSVExport .` → ok; 16 seed subtests ran | `fuzz_test.go` (read-only) |
| 3 | Verified the full package under the race detector, uncached | `nix develop -c go test -race -count=1 .` → **ok 25.986s** | whole package |
| 4 | Reran the failed `test-race` BuildFlow step | `buildflow -s test-race` → ✔ 6.8s, "1 success, 0 failed" | BuildFlow step |
| 5 | Reran the failed `templ-fmt` BuildFlow step (real run, then cache-confirmed) | Format Complete count=1 errors=0 changed=0; step ✔ 1/1 80ms | BuildFlow step |
| 6 | Root-caused the templ-fmt hang as host-load spawn starvation | `uptime`: 5-min load avg 37 at diagnosis; BuildFlow's own transient classification | diagnosis only |
| 7 | Ran the full local gate with the documented exclusions | run 20261030-135046: **59 steps, 59 pass, 0 fail, 100%**, exit 0 | BuildFlow dev battery |
| 8 | Post-run hygiene checks | `git status` clean; `git diff --stat go.mod go.sum` empty (AGENTS.md trap 4) | repo hygiene |

## b) PARTIALLY DONE

| # | What works | What remains | Blocker | Effort |
|---|-----------|--------------|---------|--------|
| 1 | templ-fmt transient closed by rerun (green) | No guard against the next load-spike recurrence — "rerun transients" is implicit, nowhere policy | None, decision needed | S |
| 2 | v0.12.2 cut locally in `0a59af4` | **Remote state unverified**: no `git fetch`, no tag-on-origin check, no tag-CI verdict, `scripts/verify-release.sh v0.12.2` not run by this session. Notably: NO BuildFlow gate ran between the 05:04 fix and the tag cut — my 13:50 run was the first post-fix gate. Tag-side CI is the only thing that may have verified it | None — was out of the pasted scope | S |
| 3 | Full dev-mode gate green | The run's warnings left standing: "9 tools unavailable (health check failed)" never enumerated; formatter fingerprint drift WARN (`templ-fmt v0.3.1020 -> v0.3.1020`) noticed but not investigated | None | S |
| 4 | Session findings recorded (this report) | HARVEST of section (f) into TODO_LIST.md/ROADMAP.md not run — user ordered wait-for-instructions | User instruction | S |

## c) NOT STARTED

| # | Planned | Why not started | Still wanted? |
|---|---------|-----------------|---------------|
| 1 | Investigate the regression verdict (16 steps slower than baseline: dprint-format +982%, test-compile +389%, golangci-lint +62%…) | Dismissed as host-load artifact without running `buildflow timings --regressions` | Yes — at least one judgment pass |
| 2 | Enumerate the 9 unavailable buildflow tools | Out of pasted scope | Yes |
| 3 | AGENTS.md memory update for this session's lessons (load-starvation rerun pattern; cache-served-green caveat) | Deferred to after user review of this report | Yes |
| 4 | Post-gate remote sync check (`master...origin/master`) | Not part of triage; release discipline applies to release sessions | Yes, before next push |
| 5 | Host Chromium http(s)-hang root cause (documented 2026-10-28, suite self-skips in ~12s; CI owns runtime browser verification) | Open host-level bug, needs deep netlog work | Yes, but CI-covered meanwhile |
| 6 | Root-fix system templ skew (system CLI/LSP v0.3.1020 vs go.mod pin v0.3.1070; NixOS config change outside this repo) | Different repo (nix config); only resurfaced via the drift WARN | Yes |

## d) TOTALLY FUCKED UP

**Nothing in the repo is broken by this session** — zero source changes; both failures were already dead at HEAD. What follows is radical honesty about **this session's own work quality**:

1. **I almost reported a cache-served green as verification.** My first `buildflow -s test-race` rerun passed in 6.1s — physically impossible for a real race suite (full runs take minutes). I caught the anomaly only by cross-checking `buildflow history` (7m7s step-time in the failed full run) and then forced an uncached `-count=1` proof. A less careful pass would have shipped "test-race verified" resting entirely on the result cache — the same cache that correctly served the *pre-fix* failure, meaning cache greens prove "inputs unchanged", not "logic verified". **Severity**: none (self-caught, 3 min). **Mitigation now in place**: uncached race run on record.
2. **Wasted a rerun by piping `tail -25`.** The first templ-fmt/test-race outputs were filtered down to DI shutdown noise; the step verdict was discarded. Sloppy command design; the full gate run did it right (output → file first). **Severity**: cosmetic.
3. **Paste provenance never fully reconciled.** The paste says "**2** failed" (templ-fmt AND test-race), but local history records the 03:51 run as **1** fail. Either the paste is from a different run, or BuildFlow counts a gate failure separately from the step failure. I did not chase it. **Severity**: low — both named steps are individually verified green regardless.
4. **Diagnosis order was lucky, not methodical.** I ran the seed tests before checking the fix commit's timestamp — the line-number mismatch (382 vs 389) happened to be in view. Had the fix only changed behavior without moving lines, I might have re-derived a "fix" that already existed. **Severity**: none.

## e) WHAT WE SHOULD IMPROVE

1. **Cache-aware failure verification** — When re-running a *failed* step, a green served from the result cache is weak evidence (it re-verifies inputs, not execution). Policy candidate: failure-verification reruns go uncached (`BUILDFLOW_NO_RESULT_CACHE=1`) or get a timing sanity-check against history. Impact: prevents false "fixed" claims; cost: one flag.
2. **Transient-timeout policy** — templ-fmt died of load starvation with BuildFlow's own "Rerun (transient)" advice. That's fine once; make it policy: pre-push-checks or a documented "rerun once before investigating" rule, so nobody burns an hour diagnosing a load spike.
3. **Output discipline** — background/filtered commands must land their verdict in a file or a `rg` on the verdict line, not a blind `tail`. One wasted rerun this session; cheap to never repeat.
4. **Release-session completeness** — `0a59af4` cut v0.12.2 with no local gate between fix and tag. Remote tag-CI may have covered it, but the local release-discipline checklist (gate AFTER release edits, verify-release.sh) should be confirmable from history alone. Make the sequence a checked script step, not prose.
5. **Paste-provenance habit** — the line-number mismatch was the single fastest diagnostic this session (stale-vs-broken decided in one `rg`). Formalize it: on any pasted test failure, first diff the quoted file:line against HEAD.

## f) Up to 50 things we should get done next

Ranked by impact. Source markers: **[S]** session-derived, **[K]** known-open from loaded project context (AGENTS.md/TODO_LIST), no new research done. Effort: S <30min, M 30min–2h, L >2h.

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Verify v0.12.2 remote state: `git fetch`, tag on origin, tag-CI run verdict, `bash scripts/verify-release.sh v0.12.2` [S] | Critical | S | Quality |
| 2 | HARVEST this report's section (f) into TODO_LIST.md / ROADMAP.md (docs-health HARVEST) [S] | High | S | Documentation |
| 3 | Record "cache-served greens ≠ verification" + "transient ⇒ rerun once" lessons in AGENTS.md [S] | High | S | Documentation |
| 4 | Run `buildflow timings --regressions`; judge whether the 16 slowed steps are load artifacts or real regressions [S] | Medium | S | Quality |
| 5 | Enumerate the 9 unavailable buildflow tools (`--verbose`) and classify fix vs. fleet-known [S] | Medium | S | Quality |
| 6 | Investigate the templ-fmt formatter fingerprint drift WARN (identity v0.3.1020→v0.3.1020; likely nix store path change); re-verify cached verdicts via `buildflow format` [S] | Medium | S | Quality |
| 7 | Add load-gate or retry-once for transient tool timeouts to `scripts/pre-push-checks.sh` [S] | Medium | M | Quality |
| 8 | Reconcile the paste-vs-history "2 failed vs 1 fail" discrepancy (which run produced the paste; does the gate count separately?) [S] | Low | S | Quality |
| 9 | Confirm `master...origin/master` sync and latest master CI conclusion before any push (release discipline bullet 0) [K] | High | S | Quality |
| 10 | Root-fix host Chromium http(s)-navigation hang (open since 2026-10-28; browser suite self-skips, CI carries runtime proof) [K] | High | L | Bug |
| 11 | Bump system templ package so CLI+LSP match the go.mod pin v0.3.1070 (NixOS config, outside repo; root fix of the 2026-10-25 skew gotcha) [K] | Medium | S | Cleanup |
| 12 | Make repo scripts self-select the devShell go (root-fix queued since 2026-09-22; `check-ui-pins.sh` pattern already proves the approach) [K] | Medium | M | Quality |
| 13 | Fix the release-draft workflow (never produced a draft for any tag; `gh api --jq` masks the 404) — existing TODO_LIST row [K] | High | M | Bug |
| 14 | Force uncached verification into release pre-push checks when re-running previously-failed steps [S] | Medium | S | Quality |
| 15 | Check UI pins guard covers templ formatter identity, not just module versions (the 2026-10-25 CLI/LSP skew escaped it) [S] | Medium | S | Quality |
| 16 | go-structure-linter: unskip when BuildFlow ships project-config support (fleet bump) [K] | Medium | M | Quality |
| 17 | templ-generate skip-gate: re-verify the nix snapshot race and unskip if fixed (documented unskip condition exists) [K] | Medium | M | Quality |
| 18 | Local nix-build proxy DNS refusal: root-fix DNS or document the CI-only arrangement as permanent with rationale [K] | Medium | M | Quality |
| 19 | Run the full (non-dev) build-mode battery before the next release — this session's 59/59 was dev mode [S] | High | S | Quality |
| 20 | erraudit stale-binary trap: add a `erraudit version` freshness probe to pre-push-checks (three bites on record) [K] | Low | S | Quality |
| 21 | Gates-vs-daemon race: mechanize the "retry once on dirty-tree false-fail" rule currently prose-only [K] | Medium | M | Quality |
| 22 | Mutation-check discipline: encode "probe the working tree, not a captured patch" into the pre-push script docs [K] | Low | S | Documentation |
| 23 | Nightly fuzz campaigns: confirm last fuzz.yml run green and target registry still matches fuzz_test.go (pre-push check 5 covers placement, not CI health) [S] | Low | S | Quality |
| 24 | Verify README screenshots still match post-v1.24 rendered output (env-guarded captures may predate the 23-wrapper golden drift) [S] | Medium | S | Documentation |
| 25 | v1.24 a11y follow-up: consider naming the dashboard Table regions (aria-label via Caption/AriaLabel props) instead of 23 unnamed role="region" wrappers [K] | Low | S | Feature |
| 26 | `buildflow doctor` 3 warns: enumerate and triage each [S] | Low | S | Quality |
| 27 | Host hygiene: are the 19 chromium processes leaked browser-test children or other users'? If ours, add a bounded cleanup [S] | Low | M | Cleanup |
| 28 | Consider buildflow parallelism tuning for this shared host (load 15–37 regularly) [S] | Low | M | Quality |
| 29 | Document `BUILDFLOW_NO_RESULT_CACHE=1` in docs/gotchas.md with the cache-served-green lesson [S] | Low | S | Documentation |
| 30 | gopls stdversion dismissal: re-check on next gopls upgrade (ROADMAP item, trigger approaching) [K] | Low | S | Quality |
| 31 | go-auto-upgrade samber/lo non-adoption: file the fleet-level gap upstream in buildflow (zero-deps policy currently ignored by in-process migrators) [K] | Low | S | Quality |
| 32 | branching-flow skip: re-evaluate under the v0.6.2+ severity cap (unskipped 2026-09-22; original gate condition recorded) [K] | Low | S | Quality |
| 33 | art-dupl baseline: revisit the 1-clone-group pin if production clone groups change (gate 9) [K] | Low | S | Quality |
| 34 | Detect-only advisories (bit-flags, go-humanize, jscpd): scheduled re-review that they remain deliberate non-fixes [K] | Low | S | Quality |
| 35 | Accepted-risk review: rendered duration coarseness (`durationDisplayStep`) — confirm the anti-fingerprinting rationale still holds [K] | Low | S | Quality |
| 36 | `/health` JSON `LastUpdatedTime`: leave out unless a JSON consumer asks (documented trigger — check issue tracker) [K] | Low | S | Feature |
| 37 | Load harness: periodic `LOADTEST=1 TestLoad_Aggregate20Sources` re-run against current dependency set [K] | Low | M | Quality |
| 38 | axe-core browser audit: confirm it executed (not offline-skipped) in the last green CI browser job [S] | Low | S | Quality |
| 39 | Federation has no evaluation hook (ADR-0003 seam belongs on remotes): document explicitly in cmd/health-hub package docs if not already [K] | Low | S | Documentation |
| 40 | health-hub env validation: add a smoke test exercising the `safeBasePath`-style validation for all HEALTH_HUB_* toggles [K] | Low | M | Quality |
| 41 | Fleet version-stamp kit: diff `pkg/version` against the fleet standard doc (last synced at v0.12.2?) [S] | Low | S | Quality |
| 42 | Run `nix run .#coverage` before next release; note trend vs. last recorded coverage [S] | Low | S | Quality |
| 43 | Run `nix run .#vulncheck` (govulncheck) — vulnix ran in the gate, govulncheck is a separate gate [S] | Low | S | Quality |
| 44 | Templ LSP squiggle: interim restart of templ LSP after next nix rebuild; permanent fix = item 11 [K] | Low | S | Cleanup |
| 45 | Add "check commit timestamps before diagnosing pasted failures" to the gotchas file (one line, this session's fastest win) [S] | Low | S | Documentation |
| 46 | Example demo: confirm `DEMO_ANALYTICS_URL` off-row contract still rendered after recent templ-components bumps (TestBrowser_OffRowContract is env-dependent) [K] | Low | S | Quality |
| 47 | Golden files: confirm the 7 goldens' v1.24 drift (23 role="region" wrappers) is exactly what CI currently enforces — no stale pin [K] | Low | S | Quality |
| 48 | Pre-push gate 0 after release edits: mechanize the "re-run gate 0 AFTER the cut's edits" rule (v0.12.0 red-tag precedent) [K] | Medium | S | Quality |
| 49 | `docs/research/2026-09-10_benchmarks.md` M91 Chrome-launch verdict: refresh after the Chromium 156 upgrade if launch latency shifted [K] | Low | S | Documentation |
| 50 | Decide and document which buildflow steps may legitimately serve cached greens in CI vs. must re-execute (policy follows from lesson 3) [S] | Medium | S | Quality |

**HARVEST note**: items 1–20 are the TODO_LIST candidates; the rest are ROADMAP/polish fuel. Per the status-report skill, section (f) belongs in TODO_LIST.md, not entombed here — run docs-health HARVEST when instructions resume.

## g) Questions I cannot answer myself

1. **v0.12.2 remote verification (item 1):** No BuildFlow gate ran between the 05:04 fix commit and the v0.12.2 tag cut — the only post-fix verification before this session was none. Do you want me to verify the tag's remote/CI state now (`git fetch` + `gh run list` on the tag + `verify-release.sh v0.12.2`), or is that reserved for a dedicated release session?
2. **Host load policy (items 7, 28):** The box regularly runs at load 15–37 with 26 users. Is that expected to persist (→ build load-aware retries/tuning into the gates), or was today an anomaly (→ keep "rerun transients" as manual policy)?
3. **Cached-green policy (items 14, 50):** Should previously-failed steps always be re-verified uncached (enforced in `pre-push-checks.sh`), or is cache-revalidation acceptable when a real uncached run exists elsewhere in the gate?

---

## Appendix — session command log

| Command | Verdict |
|---------|---------|
| `pgrep -fl buildflow` + `nix develop -c go test -run FuzzCSVExport .` | no active run; seeds ok 0.004s |
| `nix develop -c go test -v -run FuzzCSVExport .` | 16 seed subtests ran, all pass |
| `buildflow -s test-race --verbose` (bg) | ✔ (cache-served, 6.1s — flagged) |
| `buildflow -s test-race` | ✔ 6.8s, 1 success / 0 failed |
| `nix develop -c go test -race -count=1 .` | **ok 25.986s** (uncached proof) |
| `buildflow -s templ-fmt --verbose` | Format Complete, count=1, errors=0, changed=0 |
| `buildflow -s templ-fmt` | ✔ 1/1, 80ms |
| `buildflow --build-mode dev --exclude go-mod-update --exclude nix-hash-fix --exclude nix-build-verify` | **59/59 steps, exit 0** (run 20261030-135046) |
| `git status --short`; `git diff --stat go.mod go.sum` | clean; empty |
