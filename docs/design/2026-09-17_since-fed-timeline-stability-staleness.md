# Design — Since-Fed Timeline, Stability Collapse, Per-Source Staleness

|            |                                                                                                       |
| ---------- | ----------------------------------------------------------------------------------------------------- |
| **Date**   | 2026-09-17                                                                                            |
| **Status** | Design note — not implemented; graduates ROADMAP raw ideas (M54–M56 of the pareto plan)               |
| **Inputs** | go-health v0.2.0 `Check.Since`/`DurationNanos`; the v0.3.x history ring; the 2026-09-17 metadata work |

The v0.9.0 metadata work made the probe's state-entry time (`Check.Since`)
visible per row. Three ROADMAP ideas want to build on it. They share one
semantic foundation, so they are designed together.

## 1. Semantics of `Check.Since` (the foundation)

`Since` is **probe-observed**: it records when the probe entered the
check's current status, and it updates when the status flips. Consequences:

- **It survives dashboard restarts.** The dashboard is a reader; while the
  probe process stays up, `Since` keeps its value across dashboard
  redeploys — unlike everything in the dashboard's history ring.
- **It records unobserved flips.** A check that flipped while no dashboard
  was watching still carries the exact entry time of its current status.
- **It is lossy for middle states.** Two flips while unwatched collapse to
  the second one's entry time; the intermediate status is unrecoverable.
- **It resets when the probe restarts** (fresh process = fresh
  observation). With `aggregate.New`, only the restarting SOURCE's checks
  reset; other sources keep their stamps — per-source continuity.

## 2. Timeline fed by `Since` (M54)

Today the timeline card shows transitions derived from the pusher's ring
(dashboard-observed, bounded retention, wiped on restart — the timeline is
empty right after every redeploy).

**Design: hybrid.**

- While running, keep the observed ring for flips (it has full history the
  probe cannot offer, e.g. two flips in one refresh window).
- Seed the timeline on Start from each check's `Since`: every check enters
  the view as "<status> since <stamp>" instead of nothing. A restart no
  longer produces amnesia; the timeline shows current-state entry times
  with an explicit "observed since <restart>" divider between seeded and
  ring-recorded entries.
- Display rule: seeded entries are marked "reported" (probe-side truth);
  ring entries stay "observed". Never mix them into one undifferentiated
  sequence — they have different trust and different lossiness.

**Open question (deliberately deferred):** whether seeded entries should
age out of the timeline once the ring accrues past a window. Proposal: no
— they graduate into the stable-collapse summary (below) instead.

## 3. Stable-group collapse (M55)

The healthy group already collapses by row count (`WithHealthyGroupCollapse`).
Age-based stability is orthogonal and honest in a different way: "these
greens have been green since <time>" is provable from `Since` without any
observation window.

**Design:** the healthy group's summary line gains "· stable for 6h" when
EVERY check in the group has `now - Since >= 6h` and status `pass`. The
threshold is a new option (`WithStabilityThreshold(d)`, default 6h, 0
disables). Aggregates: a source card is "stable" only when all its checks
qualify; the worst-of status already handles the failure side.

- Restart-safe: the age derives from `Since`, not from ring data.
- Public mode: durations and ages are non-identifying (established in
  v0.9.0), so the summary renders verbatim.
- Interaction with row-count collapse: count-based collapse decides
  WHETHER the group is a `<details>`; stability decides what the summary
  SAYS. Independent axes.

## 4. Per-source staleness (M56)

A merged source whose probe stops refreshing freezes silently: its
`CachedResponse` snapshot never changes, its statuses render as healthy
(or unhealthy) forever, and the worst-of merge inherits the frozen values.

**Design: freeze detection at the dashboard.**

- The pusher already ticks every interval. For each aggregate source,
  track the last tick at which the source's contribution changed
  (statuses OR any `Since` OR durations). A source whose contribution has
  been byte-identical for `k` probe refresh intervals (configurable,
  default 5) is flagged stale.
- Surface: a "stale?" marker on the source card header + a
  `dashboard_health_source_stale{source=...}` gauge in the metrics
  exposition. The health STATUS itself is untouched (a stale source keeps
  its last real status — staleness is presentation + metrics, not a
  synthetic fail; a synthetic fail would lie about what the probe
  reported).
- Single-probe dashboards benefit too (a frozen probe is equally
  invisible today), so the mechanism keys on the response snapshot, not
  on aggregate internals.

## 5. Deliberate non-goals

- No persistence of the ring across restarts (the evidence-posture
  decision, blocked separately, covers persistence philosophy).
- No back-filling middle states for unobserved flips — impossible from
  `Since`, and pretending otherwise would fabricate history.
- No staleness-triggered status overrides — the dashboard reports what
  the probe reported, plus honest metadata about freshness.
