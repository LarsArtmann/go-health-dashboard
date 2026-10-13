# ADR-0005: Watchdog Staleness Floor (`pusherStaleFloor`, 1s)

**Date:** 2026-10-11 (fixed); ADR recorded 2026-10-13
**Status:** Accepted (shipped in v0.11.0)
**Covers:** the 2026-10-11 watchdog race flake session; CHANGELOG `[0.11.0]` Fixed entry

## Context

`Dashboard.HealthCheck` implements the pusher watchdog: if the push loop has
not completed a broadcast tick for more than three push intervals, the
dashboard reports `ErrPusherStale` (report-only; never restarts the
goroutine). `resolvePushInterval` deliberately inherits the probe's refresh
interval as the push cadence when no explicit `WithPushInterval` is set, so
a probe refreshing every 20ms also pushes every 20ms — and the staleness
bound collapsed to 3×20ms = 60ms.

Under `-race` on a loaded machine, a single slow tick (a templ render on the
broadcast path) can exceed 60ms. That produced spurious `ErrPusherStale`
reports: the self-monitoring cascade marked the Dashboard `warn` although
the pusher was healthy. Two flakes on 2026-10-11 traced to exactly this
race. Any consumer with a sub-~350ms probe was exposed — a product bug, not
a test bug.

## Options considered

1. **Raise the factor globally** (e.g. 3× → 17×) — the factor that survives
   `-race` at 20ms intervals is arbitrary, and it weakens genuine
   wedged-pusher detection for normal cadences by the same amount.
   Rejected.
2. **Clamp the push cadence** (minimum push interval in
   `resolvePushInterval`) — changes the published "push at probe cadence"
   behavior for every existing consumer and hides fast probes entirely.
   Rejected.
3. **Restart the pusher on staleness** — the watchdog is report-only by
   design (ADR-0002 family): auto-restart would mask the underlying wedging
   cause and contradict the sentinel family's contract. Rejected.
4. **Floor the staleness bound**: `staleAfter := max(3×interval, floor)` —
   keeps the cadence semantics untouched, keeps 3×interval detection for
   every cadence above ~333ms (where 3×interval already exceeds 1s), and
   bounds the false-positive surface for fast cadences to "the pusher was
   silent for a full second". A wedged pusher is still detected, one second
   later. Accepted.

## Decision

**Option 4.** `pusherStaleFloor = time.Second`;
`staleAfter := max(stalenessFactor*push.interval, pusherStaleFloor)`
(`dashboard.go`). Shipped in v0.11.0.

The floor is pinned by a deterministic regression test,
`TestHealthCheck_WatchdogFloorProtectsFastPushers` (`watchdog_test.go`): a
hand-wired never-started pusher (so no live loop races the assertion) with a
20ms interval must stay healthy at a 700ms-old `lastBroadcast` and must
report `ErrPusherStale` at 2s. Deleting `pusherStaleFloor` fails the first
assertion with the exact pre-fix message ("stale after 60ms").

## Consequences

- Fast-cadence consumers (sub-~333ms probes) get wedged-pusher detection at
  1s instead of 3×interval — a detection-latency cost, not a correctness
  one; the watchdog remains report-only.
- Cadences above ~333ms behave exactly as before (3×interval > 1s).
- Wall-clock test margins documented in AGENTS.md ("the watchdog only fires
  after `max(3× push interval, 1s)`") are enforced by this test, not just
  convention.
