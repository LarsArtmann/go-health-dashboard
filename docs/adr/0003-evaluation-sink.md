# ADR-0003: Probe-Cadence Evaluation Sink (`Dashboard.Observe`)

**Date:** 2026-10-10
**Status:** Accepted (decision gate D2 of the 2026-10-10 leverage plan, resolved by
the executing session under Lars's blanket "execute the whole plan" mandate)
**Covers:** session-status items 7, 31; plan tasks T10/T11

## Context

The dashboard's trend ring and evidence log ingest at **pusher cadence**
(every `PushInterval` tick reads `CachedResponse`). The probe evaluates at
**its own cadence** (`WithRefreshInterval`) and offers
`WithEvaluationHook(func(Response))` — a synchronous post-evaluation
callback. A flap that starts and clears entirely between two push ticks is
therefore invisible to trend and evidence: the health-washing window.

The hook belongs to **probe construction**, but the dashboard receives an
**already-constructed prober** (the consumer-side `Prober` interface is the
library's core design, ADR-0002-era). The dashboard cannot register the
hook itself.

## Options considered

1. **`WithEvaluationSink(func(health.Response))` option** — the dashboard
   would expose an option, and the consumer... still has to forward hook
   fires into it by hand, because the option cannot reach the
   pre-constructed probe. Config surface without removing the forwarding
   burden.
2. **Public ingest method `(*Dashboard).Observe(health.Response)`** — one
   method; the consumer wires `health.WithEvaluationHook(dash.Observe)` (or
   a forwarding closure when the probe is built before the dashboard).
   Matches the constructed-prober reality exactly.
3. **Documented consumer pattern only** (defer code to go-health v0.6) —
   leaves every consumer hand-rolling ingest into private internals, which
   is impossible without library support. Rejected.

## Decision

**Option 2 — ship `Observe` now** (gate D2 resolved: library API now, not a
documented pattern, not an option). `Observe` ingests one response into the
trend ring and the evidence log, mutex-safe by the buffers' own locks,
nil-safe when `WithTrend` is unset. The history buffer and evidence log
move from the pusher to the `Dashboard` so observations accrue from the
first hook fire — including before `Start` — and survive
`Shutdown`-and-restart of the pusher.

The pusher keeps its own tick ingest. Wiring the hook therefore makes
trend/evidence sample at **probe cadence** (denser history — the point);
values agree because both paths read the same evaluated state. Adjacent
duplicate samples are accepted and documented: the ring is capacity-bounded
history, not an event log, and evidence counts are idempotent under
repetition.

## Limitations (documented, not hidden)

- `federation.Prober` has no evaluation hook — hub operators wire
  `WithEvaluationHook` on the **remotes**, not the hub. The hub cannot
  observe between its own merge-on-read reads.
- The hook fires synchronously inside the probe's refresh; a blocking
  `Observe` would stall evaluations (the buffers' O(1) mutex ops make this
  a non-issue; still documented for upstream contract reasons —
  `WithEvaluationHook` panics are recovered as an "evaluation-hook"
  synthetic error by go-health).

## Consequences

- Example wires the hook via a forwarding closure (the probe is built
  before the dashboard; `atomic.Pointer` holder, nil-safe).
- Sub-interval flaps become visible end to end: a unit test drives
  `Observe` directly and asserts the trend samples and the rendered
  evidence strip.
- The `/health` JSON, webhook, and metrics contracts are untouched —
  Observe feeds presentation-state (HTML-only surfaces), never the wire
  shapes.
