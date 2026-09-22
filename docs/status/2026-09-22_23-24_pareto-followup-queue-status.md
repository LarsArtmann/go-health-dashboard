# Status Report — Pareto Follow-Up Queue Execution

**Date**: 2026-09-22 23:24 CEST · **Session scope**: execution of the follow-up queue left by
`docs/status/2026-09-22_22-35_pareto-plan-execution-status.md` — desk gate, FEATURES rows, the
aggregate-path Healthz test, and both document annotations — plus one unplanned production fix
(the fleet's go-directive downgrade had reached origin).
**Result**: pushed `072c96a..f1f3241`, master synced with origin, full gate chain green.

---

## Direct answers first

**What did you forget?**

1. To verify CI on the pushed tip — the go-directive guard was presumably red on origin while the
   downgrade sat there, and I never watched it go green after the fix (`gh run watch` untouched).
2. The commit-before-wait discipline: verified edits sat uncommitted across two long buildflow
   waits, the daemon swept them into 4 heuristic commits, and the intent messages were lost until
   a soft-reset recovery re-created them.
3. To connect a hint I had already read: the deep-dive report's line 1408 ("`Aggregate.Healthz()`
   … unblocks Finding 6") appeared in my own grep output minutes before I wrote an aggregate test
   asserting the opposite premise. The test failed; the sibling source confirmed the report was
   right.

**What could you have done better?**

- Read the sibling source BEFORE writing capability assertions, not after a red test —
  verify-external-claims applies to claims about internal siblings too.
- Batch all test additions first, then update counted registries ONCE: the FEATURES count churned
  292→293→294 across three edits because I updated it mid-stream.
- Background wait scripts must make the post-wait action unconditional — my first 6-minute wait
  ended with `pgrep ... || go test`, pgrep succeeded (buildflow still running), and the test never
  ran.
- Root-cause instead of work around: the desk gate failed outside the devShell (ambient go 1.26.7);
  I re-ran it correctly but left the script trap in place.

**What could you still improve?**

- The capability-matrix discovery (aggregate forwards `Healthz`, federation does not) lives in
  FEATURES + tests but not yet in AGENTS.md's go-health dependency notes — a fresh session would
  have to rediscover it.
- Test-only changes raise the registry count but ship no CHANGELOG line; whether that's correct
  policy was never consciously decided, it just happened.

---

## a) FULLY DONE (this session run)

| # | Work                                                                                                                                                                                                                                                                                                                                                                                                                                                     | Evidence                                         |
| - | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| 1 | **Go-directive downgrade found ON ORIGIN and fixed**: daemon commit `7b6de61` had swept `go 1.27.1→1.27` onto origin (CI's new T1 guard firing for real); atomic sed+commit restore, pushed                                                                                                                                                                                                                                                              | `072c96a`; `grep go.mod` verified after          |
| 2 | **Desk gate (previously skipped) run green**: FEATURES 292/40 baseline, version const = tag, UI pins OK, changelog lint green — after discovering it must run via `nix develop -c` (ambient go 1.26.7 fails)                                                                                                                                                                                                                                             | `pre-push checks: all green` (twice, open+close) |
| 3 | **FEATURES.md inventory rows** for the three capabilities shipped by the Pareto session: Instance stat card, combined-traffic `Routes.Healthz` route, NewChecks example — registry count pinned in lockstep                                                                                                                                                                                                                                              | `c5444a9`                                        |
| 4 | **Healthz capability matrix pinned by tests (T7.3 + bonus)**: `TestHealthzPassthrough_AggregateForwardsIt` proves v0.4.0's aggregate forwards the capability AND serves its own startup-latch verdict (body-asserted, not just status); `TestFederation_HealthzCapabilityNotForwarded` proves the federation prober stays routeless (404) even with `Routes.Healthz` configured. All three Prober shapes now covered (stub/probe, aggregate, federation) | `c5444a9` — both PASS, count 294/40              |
| 5 | **Pareto plan annotated**: execution banner with per-task commit links T1–T9 (`f476f76`…`837f12e`), pointer to the execution status report, "preserved as written" note                                                                                                                                                                                                                                                                                  | `ded0537`                                        |
| 6 | **Deep-dive HTML annotated**: teal resolution callout in the verdict section + per-finding "Resolved 2026-09-22" notes naming the landing commits (`604b742`, `bfaa5c5`, `621e601`, `5e32c5a`) and the pinning byte-stability tests (names verified against CHANGELOG, not invented)                                                                                                                                                                     | `ded0537`                                        |
| 7 | **AGENTS.md auto-push caveat**: "the daemon NEVER pushes" is no longer safe — `072c96a` reached origin before the explicit push ran; discipline now requires fetch+compare before any pushed/unpushed claim                                                                                                                                                                                                                                              | `c9f1608`                                        |
| 8 | **Daemon-sweep recovery without data loss**: 4 heuristic daemon commits soft-reset (sanctioned mode), contents verified mine-only via `--stat`, re-committed as 2 intent commits                                                                                                                                                                                                                                                                         | `c5444a9`, `ded0537`                             |
| 9 | **Full gate chain + push**: build → fmt (canonical generate→fmt) → full test suite (main/hub/version ok) → lint **0 issues** → vet → desk gate green → templ fmt committed → pushed → `git fetch` confirmed sync                                                                                                                                                                                                                                         | `f1f3241` on origin                              |

## b) PARTIALLY DONE

| Item                    | Done                                                                            | Missing                                                                                                             |
| ----------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| Auto-push investigation | The BEHAVIOR is recorded (one commit auto-pushed; later daemon commits did not) | The MECHANISM is unknown (buildflow sync? git hook? fleet cron?) — not investigated                                 |
| T7 test coverage        | Plan's aggregate-path gap closed, and the federation negative added beyond plan | No test for a hypothetical EXTERNAL `Healthzer` implementor's seam (minor; stub decorator covers the internal path) |
| Session bookkeeping     | FEATURES, CHANGELOG-adjacent checks, both annotations                           | AGENTS.md dependency notes still lack the capability-matrix fact (aggregate yes / federation no)                    |
| CHANGELOG hygiene       | Verified `[Unreleased]` already covers T5/T7/T8 — no gap                        | This session's test-only additions consciously carry no entry; policy never explicitly stated                       |

## c) NOT STARTED

1. **CI verification on the pushed tip** — `gh run watch`/`gh run list` on `f1f3241` never executed.
2. **pre-push-checks.sh devShell guard** — the script still fails cryptically outside `nix develop -c`
   (bare `go: go.mod requires go >= 1.27.1 (running go 1.26.7)`); it should detect and instruct.
3. **The three carried user decisions** — v0.10.0 cut, Healthz default-route policy, buildflow
   go-mod-normalize upstream filing (see §g).
4. **All pre-existing TODO_LIST "Next Up" rows** — compose e2e, deploy README, rulesets-as-code,
   verify-dep-bump hardening, a11y/browser follow-ups, docs/metrics.md, shellcheck, etc. (§f below).
5. **README coverage check** for the new route/card — not verified this session (FEATURES got rows;
   README untouched, state unknown).

## d) TOTALLY FUCKED UP

Nothing catastrophic — no data loss, no broken main beyond the incoming downgrade (fixed), no
reverted work. The honest dishonor roll, worst first:

1. **The false-premise test** — wrote `AggregateDoesNotForwardIt` from memory of the federation
   type's shape, got a red test, and the correction inverted the test into something BETTER
   (aggregate forwards, body-pinned; federation negative added). Cost: one full fail cycle.
   Root cause: assertion written before source verification, plus ignoring a hint already on screen.
2. **The `||`-short-circuit wait** — background script `pgrep || run-test` means "run test only if
   buildflow is STILL RUNNING at poll end", which is exactly backwards; ~6 minutes burned, zero
   tests executed.
3. **Lost the commit race twice** — edits held across buildflow waits got swept into heuristic
   daemon commits; the recovery (soft-reset + re-commit) worked but is a maneuver that should
   never have been needed. "Commit beats the daemon" means commit IMMEDIATELY after green,
   not after the next wait finishes.

## e) WHAT WE SHOULD IMPROVE

1. **Verify internal claims like external ones** — sibling-source signatures before any
   capability/type assertion (would have saved the red test outright).
2. **Never park uncommitted verified work** — the auto-daemon races during waits; commit first,
   wait second.
3. **Counted registries update once, at the end** — write all tests, count once, one FEATURES edit.
4. **Background job scripts**: post-wait action unconditional; log what the wait concluded.
5. **Fix traps at root cause when hit** — the desk gate's ambient-go failure is a two-line guard;
   workaround-and-move-on leaves the trap for the next session (this is the same lesson as the
   "trivial doc staleness" rule).
6. **Grep output is context you already paid for** — reread it before asserting the opposite.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

Session-spawned (this run's observations first):

1. Watch CI green on `f1f3241` (`gh run list`/`watch`) — the pushed tip is unverified by eye.
2. Add a devShell guard to `scripts/pre-push-checks.sh`: detect ambient `go` and print
   "run via `nix develop -c`" instead of a cryptic toolchain error.
3. Record the Healthz capability matrix (aggregate yes / federation no) in AGENTS.md's go-health
   dependency notes.
4. Decide CHANGELOG policy for test-only changes (document "tests don't get entries" or add one).
5. Verify README route/feature coverage for `Routes.Healthz` + the Instance card.
6. Document the capability asymmetry for hub operators (health-hub README/deploy note: no Healthz
   route on federation-fed dashboards).
7. Investigate the auto-push mechanism (`.git/hooks`, fleet buildflow config) — one session block.
8. Consider a LOCAL pre-commit go-directive guard: CI's guard fires post-push, but the downgrade
   reached origin — the window is between local gates and push.
9. Fleet filing: buildflow `go-mod-normalize` downgrade bug (BLOCKED on your call, §g3).
10. v0.10.0 cut decision (§g1).
11. Healthz default-route policy decision (§g2).
12. Example env toggle for the new surfaces (`DEMO_INSTANCE_ID`, exercise `Routes.Healthz`) so the
    cards/routes are demoable without code.
13. Body-pin the stub-path Healthz test (assert the 503 body, not just status — parity with the
    aggregate test's identity assertion).
14. `docs/integrations.md`: load-balancer guidance using the combined-traffic route.
15. `docs/DOMAIN_LANGUAGE.md`: add "combined-traffic handler" / "capability interface" terms.

Carried TODO_LIST "Next Up" rows (premise-checked 2026-09-22, still open):

16. Boot the full compose stack e2e (live scrape, 8 panels, screenshot for deploy docs).
17. Write `deploy/README.md` (ports, provisioning, anonymous-viewer posture, teardown).
18. `scripts/set-tag-protection.sh` (rulesets-as-code, idempotent PUT, encodes the 422 quirks).
19. verify-dep-bump.sh: read the coverage floor from ci.yml (kill the split brain).
20. verify-dep-bump.sh: daemon-proof the fmt check (retry-once before trusting a failure).
21. verify-dep-bump.sh: devShell-independent `go tool cover` parse.
22. CHANGELOG per-bullet tag audit v0.3.0–v0.8.0 (diff sections against their tags).
23. Pin the `shutting_down:true` webhook payload case (true branch unpinned at wire level).
24. Check `testdata/golden/source.html` against Since-carrying tooltips (predates pairing).
25. M88: keyboard-navigation a11y smoke (tab order + focus on filter/pill/collapse).
26. M89: metrics endpoint under strict CSP in the browser suite.
27. M91: browser-suite startup latency measurement vs the 45s announce timeout.
28. M92: parameterize `loadtest_test.go` (sources/clients/cadence) and re-run with evidence.
29. M103: unit-test the version-guard grep logic.
30. M105: doc.go combo examples (webhook+public-mode, `WithBasePath`).
31. M106: amend the bisect-wall audit with the 2026-09-17 scars.
32. M107: nightly job verifying a Release page exists for every tag.
33. Mobile visual check of the metadata line in the stacked layout.
34. Dedicated dark-mode contrast measurement for the dim metadata line.
35. Re-run the aggregate load test on the v0.9.0 render.
36. Blocked-row premise triage across TODO_LIST.
37. Exercise `release-draft.yml` on the next real tag push.
38. Inspect a CI browser run's screenshot artifacts by eye.
39. Align the go.mod floor with the CI matrix's 1.26.8 testing via verify-dep-bump.sh.
40. Sweep ROADMAP v1.0 criteria against current state (annotate met/unmet).
41. pre-push-checks.sh: fuzz.yml-vs-targets consistency check (mechanize the registry rule).
42. Digest-pin prometheus/grafana images + compose dependabot.
43. Compose healthchecks (prometheus ready before grafana).
44. shellcheck for `scripts/*.sh` in CI.
45. Write `docs/metrics.md` (every metric, labels, cardinality).
46. Grafana threshold-alert example (`dashboard_health_up == 0`).

Adjacent polish surfaced while writing this report:

47. Propose upstream (go-health) documenting the Healthz capability matrix — even the audit
    mis-modeled it; a README table would spare every consumer.
48. Sweep `docs/status/` archive convention (move stale same-day reports to `archived/` if the
    docs-health flow expects it).
49. Add "check `git diff go.mod go.sum`" mechanically into the gate chain script (check-go-directive
    covers only the directive, not version-pin sweeps).
50. Re-run the docs-health VERIFY pass after the next test batch (FEATURES count is 294/40 today;
    it will drift again).

## g) THREE QUESTIONS (I cannot answer these myself)

1. **v0.10.0**: cut now (the `[Unreleased]` section carries three user-facing features + wire
   fixes) or keep accruing toward a larger minor?
2. **Healthz default route**: stay disabled-forever (kubelet owns `/healthz`, opt-in via explicit
   `Routes.Healthz`), or default to `/health/healthz` in a future minor once documented?
3. **BuildFlow filing**: file the `go-mod-normalize` go-directive downgrade (now fired THREE times:
   twice before the guard, once after — onto origin) in the BuildFlow repo, or keep absorbing it
   with the per-repo guard?

---

_Pushed `072c96a..f1f3241` · working tree clean · master == origin/master at time of writing._
