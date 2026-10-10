# ADR-0004: Opt-In Cascade Verdict Forwarding (`WithCascadeProbeVerdict`)

**Date:** 2026-10-10
**Status:** Accepted (decision gate D1 of the 2026-10-10 leverage plan, resolved by
the executing session under Lars's blanket "execute the whole plan" mandate)
**Covers:** session-status item 11; plan task T12

## Context

`Dashboard.HealthCheck` (the `do.HealthcheckerWithContext` implementation)
reports the **real-time surface's** health: pusher started, not shut down,
not stale. go-health probes carry their **own** cascade verdict —
`Probe.HealthCheck(ctx)` wraps `ErrProbeUnhealthy` when the cached roll-up
is `fail`. A consumer that registers the dashboard via
`dashboard.Register(injector, probe, ...)` hears only the pusher in
container-wide `do.HealthCheck` cascades: the probe's verdict never
reaches the container.

Both semantics are legitimate, and they diverge on purpose:

- **Pusher-scoped** (current default): "can this instance render and push
  live updates?" A federation hub whose **remotes are dark** is still a
  healthy hub — remote health is the dashboard's *content*, and restarting
  the hub over a remote blip would cause a restart cascade (the same
  reasoning as federation's liveness handler, which never fetches).
- **Probe-forwarding**: "is the fleet this instance watches healthy?" What
  a single-service deployment usually wants from container health.

## Options considered

1. **Always forward** when the prober implements the verdict capability —
   a silent semantic break for every existing consumer; a hub with a
   failing remote would start failing `do.HealthCheck` and get restarted
   by orchestrators. Rejected as default.
2. **Opt-in option `WithCascadeProbeVerdict()`** — forwards when (and only
   when) the prober implements `HealthCheck(ctx) error`; default off
   preserves the pusher-scoped contract. Additive, no breaking change.
3. **Never forward** — leaves the audit's gap open; consumers hand-roll a
   second registration for the probe (double registration in the injector,
   confusing ownership). Rejected.

## Decision

**Option 2.** `WithCascadeProbeVerdict()` opts the cascade into the probe's
roll-up verdict. Ordering inside `HealthCheck`: pusher-state errors
(not-started / shut-down / stale) are returned **first** — a shut-down
dashboard must never masquerade as healthy just because the probe would
pass. Only when the real-time surface is healthy does the probe's verdict
(decision) apply, via a narrow local capability interface so any prober
with the `do.HealthcheckerWithContext` method shape forwards, and probers
without one (federation, test stubs) fall back to pusher-scoped silently.

## Consequences

- Default behavior is byte-for-byte unchanged; `errors.Is(err,
  ErrPusherNotActive)` semantics intact.
- Single-service deployments that want "cascade = fleet health" add one
  option; hub deployments keep the default.
- The federation gap is not papered over: forwarding on a hub forwards
  federation's own `HealthCheck` — which federation does not implement —
  so hubs are pusher-scoped unless a future federation release adds the
  verdict (tracked with the upstream proposals).
