# ADR 0002: Error-sentinel family for pusher state

Date: 2026-09-04
Status: Accepted

## Context

`HealthCheck(ctx)` answered "is the dashboard's real-time pusher
alive?" with a single sentinel, `ErrPusherNotActive`. Two distinct
situations collapsed into it: the dashboard was never started, and the
dashboard was shut down after running. Operators using samber/do's
`do.HealthCheck` cascade could not tell a misconfigured service (never
started — a wiring bug, page it) from a deliberately drained one
(shutdown — usually part of a rolling deploy, don't page it). The
503 bodies of the trend endpoints had the same ambiguity.

## Decision

Introduce a two-sentinel family that wraps the existing parent:

- `ErrPusherNotStarted` — `Start` has never been called. Wraps
  `ErrPusherNotActive`.
- `ErrPusherShutDown` — `Shutdown` has completed. Wraps
  `ErrPusherNotActive`.
- `ErrPusherNotActive` stays the public parent. `errors.Is(err,
  ErrPusherNotActive)` keeps matching both children forever.

The watchdog's staleness error (`ErrPusherStale`, report-only) is
deliberately NOT part of this family: staleness means "alive but
silent", a different failure class from "not running".

## Consequences

- Existing callers keep compiling and keep matching: the wrap direction
  (child → parent) preserves every pre-existing `errors.Is` check.
- Callers can now distinguish wiring bugs from graceful drains with a
  second `errors.Is`, without new error types or error codes.
- The trend endpoints' 503 messages were aligned to the same
  distinction ("pusher is not active (call Start…)" vs "trend not
  enabled"), so humans reading HTTP responses get the same signal.
- Adding a third sentinel (e.g. for a future paused state) is additive
  and follows the same child-wraps-parent pattern.

## References

- `dashboard.go` — sentinel definitions and `HealthCheck`.
- `lifecycle_test.go` — the `errors.Is` matrix that pins the contract.
- `trend.go` — 503 message mapping (`trendUnavailable`).
