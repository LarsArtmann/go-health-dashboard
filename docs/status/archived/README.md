# Archive Manifest — 2026-10-25 docs-health sweep

Bulk-annotation + archive pass over every non-archived 2026-0*/2026-1* status,
planning, and harvest file (33 files). Every numbered item was resolved inline
(`~~item~~ verdict` — done at / closed / routed / NOT-DO) before the move; both
archive gates pass (`grep -L '~~'` over the moved set: empty; `check-rows`: 34/34
complete). Classification per the docs-health decision table — every file below
is ANNOTATE → ARCHIVE: each report's actionable items were executed, routed, or
consciously closed by later sessions, so a reader's question ("is this done?
where is it now?") is answered inline.

Status reports → `docs/status/archived/`:

| File (2026 prefix)                                              | Reason everything is resolved                                                                                                                   |
| --------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-09-17_08-48 docs-health audit + living-docs rebuild        | Executed by the 09-18/09-23 passes; supersessions annotated inline                                                                              |
| 2026-09-18_05-41 full-list execution + tail                     | Executed; residual rows swept 2026-09-23 (TODO_LIST sweep row)                                                                                  |
| 2026-09-18_06-12 TODO completion + infra gaps                   | Executed; swept 2026-09-23                                                                                                                      |
| 2026-09-18_09-47 federation package session                     | Shipped in go-health v0.3.0→v0.7.0 lineage; dashboard consumed v0.7.0 2026-10-11                                                                |
| 2026-09-18_13-58 audit pass 2 (pin-drift rescue)                | v1.18.0 adoption done; superseded by the v1.21→v1.23 sweeps                                                                                     |
| 2026-09-22_17-39 hub root-redirect + fleet inventory            | Landed same cycle; hub shipped in v0.10.0                                                                                                       |
| 2026-09-22_22-35 pareto plan execution                          | All tasks executed (v0.10.0 cycle)                                                                                                              |
| 2026-09-22_23-24 pareto follow-up queue                         | Executed/closed by the 09-23 sweep                                                                                                              |
| 2026-09-22_23-50 v0.10.0/v0.10.1 release                        | Released; superseded by v0.11.0 (2026-10-12)                                                                                                    |
| 2026-09-23_00-55 v0.10 release session review                   | Review of a completed release; follow-ups swept                                                                                                 |
| 2026-09-23_16-12 browser-rig dedup                              | Landed; rig is the house pattern (browserSerial/startBrowserSession)                                                                            |
| 2026-09-23_18-09 full-list execution sweep                      | This sweep closed the whole 09-18/22 queue; its own tail closed later                                                                           |
| 2026-09-23_19-30 changelog tag audit                            | Audit stable (all bullets TRUE); superseded by verify-release.sh at every cut                                                                   |
| 2026-10-08_21-58 gohumanize FP fixes                            | In-repo work landed; linter-repo items tracked in that repo                                                                                     |
| 2026-10-09_16-20 UI-revamp + v1.21.0 adoption                   | Landed; follow-ups harvested (17-30 ledger) and closed 2026-10-25                                                                               |
| 2026-10-09_17-30 HARVEST ledger                                 | Fully dispositioned routing ledger; consumption complete (ledger-exempt table)                                                                  |
| 2026-10-09_21-49 post-revamp follow-through                     | Landed; three fixes shipped in v0.10.x/v0.11.0                                                                                                  |
| 2026-10-09_22-19 go-health deep-dive session                    | Findings executed via the 10-10 leverage plan (T01-T20)                                                                                         |
| 2026-10-10_08-46 pareto plan execution                          | All 20 tasks executed, verified, pushed                                                                                                         |
| 2026-10-10_09-50 pareto execution + CI closeout                 | Pushed; CI green 2026-10-11 (5debc92)                                                                                                           |
| 2026-10-10_13-54 T16 tail (lint closeout, rebase)               | Pushed + CI green 2026-10-11; provenance closed                                                                                                 |
| 2026-10-11_18-56 CI-green closeout (release publish)            | v0.11.0 published 2026-10-12; T13 unblocked                                                                                                     |
| 2026-10-11_19-55 T13 train + filings                            | Landed (b8be72d/1473fc2); filings made (#5, #6)                                                                                                 |
| 2026-10-11_19-59 same, brutal self-review                       | Same session; items executed or routed                                                                                                          |
| 2026-10-11_21-07 watchdog floor + race-flake fixes              | Landed; ADR-0005; floor regression test shipped                                                                                                 |
| 2026-10-12_12-50 v0.11.0 release                                | Released 2026-10-12; f-list harvested (closed 2026-10-25)                                                                                       |
| 2026-10-13_18-25 adoption session (floor test, buffered render) | Landed; HARVEST rows closed 2026-10-25                                                                                                          |
| 2026-10-25_07-39 UI/UX batch (chips blocked)                    | Unblock path executed same day; chips green; 50-item f-list closed/routed                                                                       |
| 2026-10-25_10-45 closeout handoff                               | Push executed same day (872b555; c0991da/e8eeefa, CI 7/7) — inline-corrected                                                                    |
| 2026-10-25_12-26 TODO execution + hardening                     | Addendum: list complete except §g; dispositions inline                                                                                          |
| 2026-10-25_12-27 UI/UX closeout + self-review                   | Push resolved same day; 50-row queue closed/routed                                                                                              |
| 2026-10-25_13-20 TODO closeout + self-review                    | Closeout complete; decision-gated residue routed to standing Blocked rows                                                                       |
| 2026-10-25_13-38 templ CLI version skew (archived 2026-11-11)   | Every blocker landed: stamp guard + devShell wrapper (2026-11-04), NixOS templ bump (2026-11-05), gotchas history; §f 1-11 struck with verdicts |

Planning → `docs/planning/archived/`:

| File                                                 | Reason                                                                      |
| ---------------------------------------------------- | --------------------------------------------------------------------------- |
| 2026-09-17_09-04 v0.9.0 release + backlog pareto     | Executed in the v0.9.0 cycle (pre-existing strikes completed 2026-10-25)    |
| 2026-09-22_21-49 go-health superb consumption pareto | T1-T9 executed in the v0.10.0 cycle (guard later retired obsolete upstream) |
| 2026-10-10_01-44 go-health leverage pareto           | T01-T20 executed (per-item verdicts cite the TODO_LIST T-rows)              |

Kept in place (NOT archived — genuinely open work or active references):

- `docs/status/2026-10-25_14-19_buildflow-gate-and-11-day-red-ci-status.md` —
  the ACTIVE handoff: v0.12.0 cadence, red-main alerting, remote-state gate,
  dependabot branch, CI nix-build job.
- `docs/planning/2026-10-25_07-52_ui-ux-batch-closeout-and-queue.md` — active
  queue: T24 (chromedp) and T27 (v0.12.0) remain open.
- `docs/design/2026-09-17_*`, `docs/research/2026-*` — historical design and
  measurement snapshots actively cited by living docs.

---

# Archive Manifest — 2026-11-05 docs-gates repair sweep

Three reports archived while bringing the docs-gates battery green (annotate
matrix + audit-all dispositions + cadence archive ritual). Eligibility gates
ran BEFORE each move: `check-rows --strict` and `bare-rows` both pass on the
moved set (the two metadata fact-tables carry the sanctioned
`<!-- check-rows:ledger -->` exemption).

Status reports → `docs/status/archived/`:

| File (2026 prefix)                                   | Reason everything is resolved                                                                                                          |
| ---------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-10-25_17-02 docs-health full audit sweep        | Audit executed same-day (its own manifest below); item 30 closed with the live pkg.go.dev v0.13.2 check                                |
| 2026-10-25_17-53 dedup fixture + clone-baseline gate | Dedup pass landed (startDashboardFixture, gate 9); row 44 harvested to TODO_LIST (Blocked table)                                       |
| 2026-10-25_18-54 v0.12.1 release retro               | Retro of a shipped release; row 3 struck (release-cut.sh delivers the post-edit battery), row 17 harvested to TODO_LIST (go-etag port) |
| 2026-10-28_19-17 self-review v1.24.0 adoption        | Adoption landed same day (v1.24.0 guard green); loopback-reaping row moot (processes verified gone)                                    |

Kept in place (NOT archived — genuinely open work or active references): the
fifteen remaining unarchived reports, incl. the 2026-11-05 handoffs
(01-14, 02-48, 18-08).
