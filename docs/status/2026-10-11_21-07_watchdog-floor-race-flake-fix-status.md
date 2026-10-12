# Status Report — 2026-10-11 21:07 CEST — Watchdog floor + race-flake fixes

**Session scope:** one reported `buildflow -s test-race` failure (`TestSelfMonitoring_RegisteredDashboardAppearsInOwnTable`: want `pass`, got `warn`) → diagnosed, reproduced, root-caused, fixed, verified; two pre-existing/latent issues fixed on sight (second timing flake, securitymd gate blocker). Everything below is from this session only.

**Daemon commit map (this session's work):**

| Commit    | Files                               | Content                                           |
| --------- | ----------------------------------- | ------------------------------------------------- |
| `70d10b3` | `dashboard.go`, `hardening_test.go` | watchdog floor + rate-limit test margins          |
| `bfd1b1c` | `AGENTS.md`                         | watchdog bullet + wall-clock-margins testing rule |
| `70b3282` | `SECURITY.md`                       | Security Practices section + version-table fix    |

---

## Self-review (asked directly: forgotten / done better / still improve)

**What did I forget?**

1. **A regression unit test for the fix itself.** The watchdog floor (`max(3×interval, 1s)`) is verified only through the integration path (the flaky test passing) — no test pins "20ms pusher + 700ms-old lastBroadcast → HealthCheck returns nil". The existing watchdog tests pass with OR without the floor. My fix is the least-tested line I wrote today.
2. **Doc drift I may have created.** AGENTS.md now says "3 intervals, floored at 1s" — I did NOT verify README/FEATURES/ADR wording about the watchdog (they may still say "after 3 push intervals", which is now a lie for fast cadences).
3. **The CHANGELOG/release.** This is a behavior fix to a _published_ module (v0.10.2 carries the bug); I queued nothing.
4. **The superfluous-WriteHeader source.** I saw the log line twice, declared it "unrelated noise" from flow analysis (the failing test sends a JSON request that can't reach `handlers.go:36`), and never identified which test emits it. Honest downgrade: that was an _assumption_, not a verified attribution — it's recorded in (d).
5. **Treating the class, not the instances.** After the SECOND timing flake appeared in my own repro run, I fixed two instances and documented margins — I did not sweep for the remaining wall-clock-fragile tests (e.g. `TestRateLimit_ExceedsBudget` 1s window, browser-suite timeouts).

**What could I have done better?**

- Reproduced with the _exact_ shuffle seed from the user's paste (`-test.shuffle 1791743345037983383`) instead of `shuffle=on` — I got lucky that the failure is load-dependent, not seed-dependent.
- Quantified flake frequency pre/post fix (N-run measurement) instead of "reproduced once, verified 2+3 runs".
- Used the sanctioned daemon-recovery recipe (`git commit --amend`) to attach an intent message to `70d10b3`/`bfd1b1c`/`70b3282` — the history now reads `chore: auto-commit` ×3 and explains none of the WHY. The recipe is documented as sanctioned; I didn't use it because nothing told me the user wanted intent-bearing history for this fix.

**What could I still improve?**

- Root-fix the timing-flake class with an injected clock — the repo already has the precedent (`buildViewModelAt` takes an injected clock), so real `time.Sleep` margins in rate-limit/watchdog tests are an avoidable choice, not a necessity.
- Nightly flake canary so this class is caught by CI, not by users' buildflow runs.

**Did I lie?** No. One soft claim corrected above (WriteHeader "unrelated" was inference, not verification).

**Ghost systems / split brains?** No ghost systems introduced (floor is wired into `HealthCheck`, memory updated). One _possible_ split brain flagged in item 2 of "forgot" (AGENTS.md vs README/FEATURES watchdog wording) — unverified by choice (scope discipline), listed in (f).

---

## a) FULLY DONE

| # | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | Evidence                                                        |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- |
| 1 | **Root cause found + fixed: watchdog false-positive for fast probes.** `resolvePushInterval` inherits the probe's 20ms refresh as push cadence → staleness bound was 3×20ms=60ms; under `-race` a slow tick (templ render) exceeded it → spurious `ErrPusherStale` → probe marks Dashboard `warn`. Fix: `pusherStaleFloor = 1s`, `staleAfter := max(3×interval, floor)` (dashboard.go:284,309). Product bug, not test bug — any user with sub-~350ms probes was exposed. | `70d10b3`; pre-fix repro failed identically to the user's paste |
| 2 | **Second flake fixed: `TestRateLimit_RefillsOverTime`.** 150ms refill window had to outlast the gap between two adjacent in-process requests; under load it didn't (got 200, want 429). Widened to 1s window + 1.3s refill sleep (~7× margin).                                                                                                                                                                                                                           | `70d10b3`; targeted `-race -count=3` green                      |
| 3 | **Pre-fix reproduction** of BOTH failures in one full-package `-race` run — matching the reported failure plus the latent one.                                                                                                                                                                                                                                                                                                                                           | run at 20:45: two FAILs, package 51.8s                          |
| 4 | **Full verification chain green post-fix:** targeted tests `-race -count=3`; full package `-race -shuffle=on -count=2` (59.8s ok); `buildflow -s test-race` (57.3s); `buildflow -s golangci-lint`; `buildflow format` **45 success / 0 failed**; go.mod/go.sum verified untouched.                                                                                                                                                                                       | shell transcripts this session                                  |
| 5 | **AGENTS.md memory updated:** watchdog bullet (floor rationale) + new testing rule "wall-clock margins must survive `-race` + parallel load; don't re-tighten".                                                                                                                                                                                                                                                                                                          | `bfd1b1c`                                                       |
| 6 | **Pre-existing `securitymd` findings-gate blocker fixed** (would fail every full `buildflow` run): added fleet-standard Security Practices section + project-specific practices + footer to SECURITY.md; fixed stale version table (0.9.x → 0.10.x, current v0.10.2).                                                                                                                                                                                                    | `70b3282`; `buildflow -s securitymd` green after                |

## b) PARTIALLY DONE

| # | Item                     | What works                                                  | What remains                                                                                          | Blocker                                           | Effort |
| - | ------------------------ | ----------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------- | ------ |
| 1 | Timing-flake hardening   | 2 observed instances fixed; margins documented in AGENTS.md | Class-level sweep not done; injected-clock root fix not designed; nightly canary missing              | none — pure scoping decision                      | M      |
| 2 | Verification depth       | 2 full + 3 targeted `-race` iterations green                | Probabilistic evidence only; no pinned regression test for the floor; no CI canary                    | none                                              | S      |
| 3 | Documentation of the fix | AGENTS.md updated                                           | README/FEATURES/ADR wording unverified; CHANGELOG not started; ADR for the floor decision not written | none                                              | S      |
| 4 | SECURITY.md health       | Gate unblocked; version table current                       | "Last updated" footer + table are hand-maintained; no automated freshness check                       | none                                              | S      |
| 5 | Commit intent            | Changes are all committed (daemon)                          | Commit messages carry zero intent (`chore: auto-commit` ×3); amend-relabel not performed              | harness no-commit rule (sanctioned recipe exists) | S      |

## c) NOT STARTED

| #  | Item                                                                                | Why not started                                                                                                 | Still wanted?                                          |
| -- | ----------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| 1  | Patch release v0.10.3 (watchdog floor)                                              | Release cadence is the user's call; release discipline demands a dedicated session (`pre-push-checks.sh` first) | Yes — published v0.10.2 carries the false-positive bug |
| 2  | Regression unit test pinning the floor                                              | Not begun; discovered as gap during self-review                                                                 | Yes — Critical, S                                      |
| 3  | Injected-clock option for rate limiter/pusher tests                                 | Design decision (public API surface vs internal seam) not yet made                                              | Yes — root-fix for the class                           |
| 4  | Sweep of remaining wall-clock assertions suite-wide                                 | Not begun; item surfaced only at self-review                                                                    | Yes                                                    |
| 5  | Identify/silence superfluous WriteHeader source                                     | Declared out of scope mid-session; never picked back up                                                         | Yes                                                    |
| 6  | CHANGELOG entry + ADR-0005 (watchdog floor rationale)                               | Release-pending work                                                                                            | Yes                                                    |
| 7  | HARVEST of section (f) into TODO_LIST.md/ROADMAP.md                                 | Report written at session end per skill contract                                                                | Yes — else items die in this file                      |
| 8  | Upstream filings for federation gaps (MarkShuttingDown, SourceStatuses)             | Known upstream gaps from earlier sessions; verify-before-filing gate not yet run this session                   | Yes (Medium)                                           |
| 9  | Intent-relabel of `70d10b3`/`bfd1b1c`/`70b3282`                                     | Needs user go-ahead (touches daemon commits)                                                                    | Optional                                               |
| 10 | SECURITY.md freshness automation (version table + footer date in release checklist) | Not begun                                                                                                       | Nice-to-have                                           |

## d) TOTALLY FUCKED UP

| # | What is broken                                                                                                                                                                                                                                                                                                            | Severity                    | Root cause                                                                    | Mitigation                           |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- | ----------------------------------------------------------------------------- | ------------------------------------ |
| 1 | **Published v0.10.2 ships the watchdog false-positive.** Any deployment with a fast probe refresh (sub-~350ms) + the dashboard registered in a do injector can see spurious `warn`/cascade failures under load; with `WithCascadeProbeVerdict` enabled this can flip cascade verdicts. Master is fixed; releases are not. | High (users, until patched) | staleness bound inherited the probe cadence without a floor                   | fix is on master; ship v0.10.3 (f-3) |
| 2 | **The fix itself has no direct regression test** — deleting `pusherStaleFloor` would likely re-green the suite until load conspires again.                                                                                                                                                                                | Medium (future regressions) | I stopped at "flaky test passes now"                                          | f-1 (Critical, S)                    |
| 3 | **Fix history is intent-blind:** three `chore: auto-commit` messages, no WHY anywhere in git history. `git log` cannot tell the next session why the floor exists (only AGENTS.md can).                                                                                                                                   | Low–Medium (maintenance)    | daemon heuristic + harness no-commit rule                                     | f-10 (amend recipe is sanctioned)    |
| 4 | **The suite still has an unmeasured population of wall-clock-fragile tests.** Two flakes surfaced in two runs; nobody knows the true N.                                                                                                                                                                                   | Medium (CI trust)           | real-time dependencies with sub-second margins                                | f-7 sweep + f-8 canary               |
| 5 | **`superfluous response.WriteHeader` (handlers.go:36) recurs with no known-issue entry and no identified emitting test** — every future session re-investigates from zero.                                                                                                                                                | Low (noise)                 | HTML render fails mid-write after partial body; error path calls `http.Error` | f-9                                  |

## e) WHAT WE SHOULD IMPROVE

1. **Stop shipping real-time tests with sub-second real-sleep margins when the repo already has an injected-clock precedent.** Impact: this is the second flake class incident (selfmonitor + ratelimit; earlier sessions had their own). Fix: design the clock seam once (f-3), migrate the ~handful of offenders.
2. **Intent-bearing history for verified batches.** The daemon's heuristic messages erase WHY; the amend recipe exists but nothing reminds sessions to use it. Fix: add "relabel daemon commits for intent-bearing batches" to the release/checklist discipline or AGENTS.md gotchas.
3. **Docs that restate behavior in prose need a verify step when behavior changes.** I updated AGENTS.md but not README/FEATURES — a `docs-health` VERIFY pass keyed to changed decisions (watchdog) would catch this class. Fix: after any behavioral change, grep the doc set for the old constant/claim.
4. **Gate findings should never be discoverable by accident.** The securitymd blocker would have failed every full `buildflow` run; it surfaced only because I ran `buildflow format`. Fix: keep `-s test-race`-style drill-downs paired with a periodic full run (or note in AGENTS.md which full-gate cadence is expected).
5. **Version tables in SECURITY.md are hand-synced with releases** (0.9.x staleness proves it). Fix: fold "SECURITY.md supported-versions row" into the release checklist script.

## f) Top 50 things we should get done next

> Brainstorm, not commitment list (per skill contract): items 1–15 are the real queue; 16–50 are ROADMAP fuel for `docs-health` HARVEST to route with extra rigor.

| #  | Task                                                                                                                                                          | Impact   | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Unit regression test: pusher interval 20ms + lastBroadcast backdated 700ms → `HealthCheck` returns nil (pins the floor)                                       | Critical | S      | Bug           |
| 2  | CHANGELOG entry for the watchdog floor (behavior change: staleness detection latency ≥1s for fast cadences)                                                   | High     | S      | Documentation |
| 3  | Cut v0.10.3 via release discipline (`pre-push-checks.sh` → tag → verify-release.sh)                                                                           | High     | M      | Quality       |
| 4  | Verify README/FEATURES/ADR watchdog wording against the new floor; fix any "3 intervals" claims                                                               | High     | S      | Documentation |
| 5  | Sweep all tests for sub-second wall-clock assertions; widen margins or plan clock injection (`TestRateLimit_ExceedsBudget` 1s window, SSE waits, drain waits) | High     | M      | Quality       |
| 6  | Nightly flake canary: CI job running full package `go test -race -shuffle=on -count=5`                                                                        | High     | M      | Quality       |
| 7  | Design injected-clock seam for rate limiter + pusher ticker (public `WithClock` vs internal), migrate offenders                                               | High     | M      | Quality       |
| 8  | ADR-0005: watchdog staleness floor (fast-cadence inheritance via `resolvePushInterval`)                                                                       | Medium   | S      | Documentation |
| 9  | Identify emitting test for `superfluous response.WriteHeader`; fix (buffer HTML render) or document as known noise in AGENTS.md                               | Medium   | S      | Bug           |
| 10 | Intent-relabel daemon commits `70d10b3`/`bfd1b1c`/`70b3282` (amend recipe)                                                                                    | Medium   | S      | Cleanup       |
| 11 | HARVEST this report into TODO_LIST.md/ROADMAP.md                                                                                                              | Medium   | S      | Documentation |
| 12 | SECURITY.md freshness automation: supported-versions row + footer date in release checklist / pre-push-checks.sh                                              | Medium   | S      | Quality       |
| 13 | Upstream (verify-before-filing): federation lacks `MarkShuttingDown` — hub can't flip readiness during drain; file proposal                                   | Medium   | M      | Feature       |
| 14 | Upstream (verify-before-filing): federation lacks `SourceStatuses` (draining remote folds green); file proposal                                               | Medium   | M      | Feature       |
| 15 | Fuzz target for `rateLimiter.Allow` (burst/refill invariants) + fuzz.yml registry + AGENTS.md in the SAME change                                              | Medium   | S      | Quality       |
| 16 | Table-driven unit test for `resolvePushInterval` (configured > probe > default) if absent                                                                     | Medium   | S      | Quality       |
| 17 | Pusher-staleness observability: gauge or introspection field for last-broadcast age (stale is only visible via HealthCheck today)                             | Medium   | M      | Feature       |
| 18 | Rate-limit metrics: 429 counter + Retry-After histogram                                                                                                       | Medium   | S      | Feature       |
| 19 | `dashboard_sse_subscribers` gauge (SubscriberCount already exists as API)                                                                                     | Medium   | S      | Feature       |
| 20 | Introspection endpoint: include resolved push interval, staleAfter, last-broadcast age                                                                        | Low      | S      | Feature       |
| 21 | Tighten `TestSelfMonitoring`: replace 80ms sleep with the existing waitFor-poll pattern                                                                       | Low      | S      | Quality       |
| 22 | `TestSelfMonitoring`: also assert the Dashboard row's error field is empty (status-only today)                                                                | Low      | S      | Quality       |
| 23 | Example server: fast-cadence demo mode exercising the watchdog floor end-to-end                                                                               | Low      | S      | Documentation |
| 24 | BenchmarkHealthCheck (watchdog path cost under load)                                                                                                          | Low      | S      | Quality       |
| 25 | README options section: document continuous-refill token-bucket semantics (Retry-After ceiling)                                                               | Low      | S      | Documentation |
| 26 | Name the rate-limit test constants (refillMargin) so the 1s/1.3s numbers self-document                                                                        | Low      | S      | Cleanup       |
| 27 | `X-RateLimit-Remaining`/`Limit` headers on admitted requests                                                                                                  | Low      | S      | Feature       |
| 28 | Per-client/per-IP rate limiting option (bucket is global today)                                                                                               | Low      | L      | Feature       |
| 29 | Root-fix "repo scripts that exec go need the devShell": self-selecting toolchain probe in all scripts (queued since 2026-09-22)                               | Medium   | M      | Cleanup       |
| 30 | go-structure-linter unskip when the fleet bumps past v0.10.0 (TODO_LIST Blocked row)                                                                          | Low      | S      | Quality       |
| 31 | Re-verify templ-generate skip-gate against current BuildFlow (snapshot race)                                                                                  | Low      | S      | Quality       |
| 32 | Re-verify branching-flow skip under the current severity cap                                                                                                  | Low      | S      | Quality       |
| 33 | Spot-check the bisect wall skip-list (071c251..HEAD) still matches daemon hygiene                                                                             | Low      | M      | Cleanup       |
| 34 | Re-check gopls stdversion dismissal on next gopls upgrade (ROADMAP)                                                                                           | Low      | S      | Quality       |
| 35 | health-hub: behavior test for `HEALTH_HUB_*` env parsing (fuzz exists; happy-path table test?)                                                                | Medium   | M      | Quality       |
| 36 | Webhook notifier: test bounded in-flight behavior against a slow receiver                                                                                     | Medium   | M      | Quality       |
| 37 | Evidence log: test `maxEvidenceChecks` overflow/truncation behavior                                                                                           | Low      | S      | Quality       |
| 38 | Export CSV: formula-injection hardening test (check names containing `=`, `+`, `@`)                                                                           | Medium   | S      | Quality       |
| 39 | Metrics label escaping: property test vs fuzz corpus                                                                                                          | Low      | S      | Quality       |
| 40 | Extend check-ui-pins.sh: assert RecommendedCSP comment stays in sync with the pinned SDK version                                                              | Medium   | S      | Quality       |
| 41 | Gate-retry helper for daemon-race false failures (AGENTS.md gotcha → script)                                                                                  | Low      | M      | Cleanup       |
| 42 | go-datastar/static bundle size drift check in check-ui-pins.sh                                                                                                | Low      | S      | Quality       |
| 43 | Public mode: differential test HTML vs JSON for identifier leakage                                                                                            | Low      | M      | Quality       |
| 44 | Re-run LOADTEST evidence-era parameters post-floor (timing changed)                                                                                           | Low      | M      | Quality       |
| 45 | Release checklist item: summarize daemon commit intents before tagging                                                                                        | Low      | S      | Documentation |
| 46 | Re-measure Chrome launch latency (M91 verdict dated 2026-09-23)                                                                                               | Low      | S      | Quality       |
| 47 | Trend: exhaustive-switch guard test for `statusValue` mapping of `off`                                                                                        | Low      | S      | Quality       |
| 48 | docs-health VERIFY sweep: prune AGENTS.md gotchas that no longer reproduce                                                                                    | Low      | M      | Documentation |
| 49 | Simplify `push.interval > 0` guard in HealthCheck if resolvePushInterval guarantees positivity (dead condition?)                                              | Low      | S      | Cleanup       |
| 50 | Consider `docs/status/` archive rotation policy (this is the 4th report today)                                                                                | Low      | S      | Cleanup       |

## g) Questions I cannot figure out myself

1. **Release cadence:** master carries a behavior fix to a published module (v0.10.2 has the watchdog false-positive). Ship v0.10.3 now, or ride the next feature train?
2. **Concurrent BuildFlow session:** `buildflow doctor` warned the binary (5aaf336) is older than the BuildFlow repo's HEAD (2bd812c) — another session is committing there. Leave it alone until that session finishes, or rebuild the binary now?
3. **API appetite for the flake root-fix:** is a public clock-injection option (e.g. `WithClock`) acceptable API surface growth, or should timing tests stay margin-hardened with an internal seam only?

---

_Point-in-time snapshot — goes stale. Section (f) is HARVEST input for TODO_LIST.md/ROADMAP.md; items 1–15 first. Written per status-report skill; format is `.md` per explicit user instruction (skill default is HTML — override flagged in the closing message)._

---

## Postscript — section (g) answers (2026-10-12 12:32 CEST, session resumed)

1. **Release cadence:** "ship now" → v0.10.3 release executed immediately after this report (watchdog floor + docs).
2. **BuildFlow binary:** "the currently installed one should be fine" → leave `5aaf336` alone despite the freshness warning; no rebuild.
3. **Clock API:** "do what makes the most sense" → decision: **no public `WithClock`** for now (YAGNI — no consumer needs it; the landed margins plus the f-6 canary cover the class); revisit an internal seam only if the canary catches further timing flakes. Decision recorded here; promoted to an ADR if the seam is ever built.
