# Upstream draft: federation needs a two-phase drain (MarkShuttingDown)

**Status:** verified 2026-10-10 against `v0.5.1` (tag `047585f`) + a live hub run.
**Filed:** https://github.com/LarsArtmann/go-health/issues/5 (2026-10-10, gate D3 executed under the session mandate; the MANUALLY REVIEWED box ships unchecked for Lars to tick).
**Repo:** larsartmann/go-health (own repo).

---

> [!NOTE]
> This filing was drafted by GLM-5.3 via Crush from an AI-run investigation, not at my request. When the failure traces to an external report, that source is linked in the body.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

**TL;DR:** `federation.Prober` has no way to flip readiness to 503 while the
HTTP server drains, so a hub answers 200 "ready" all the way through
graceful shutdown — the exact gap `Probe.MarkShuttingDown`
(`probe.go:645`, `v0.5.1`) closes for single probes. Ask: give the
federation prober the same two-phase drain.

## Problem

Measured live on a hub built with this repo (`go-health-dashboard`
`cmd/health-hub`, go-health `v0.5.1`, tag `047585f`):

- SIGTERM → the hub's SSE clients disconnect promptly (the dashboard leads
  shutdown) — good.
- `/readyz` keeps answering **200** for the entire drain window, because
  the readiness handler does merge-on-read fetches and nothing tells it
  the process is going away. A load balancer keeps routing new
  connections to a dying hub until the listener actually closes.

Single probes don't have this hole: `Probe.MarkShuttingDown()` flips
readiness to 503 while the refresh loop keeps serving, and
`WithShutdownGracePeriod` holds that window (`probe.go:610-660`,
`v0.5.1`). The federation prober — the piece specifically deployed behind
load balancers as a hub — is the one surface without the pattern.

Source-level check at `v0.5.1`: `federation/federation.go` exposes
`CachedResponse` (:244), `StartupComplete` (:412), `ReadinessHandler`
(:441) — no shutdown/draining flag anywhere in the package; the only
options are `WithClient` (:100) and `WithTimeout` (:108).

## Goal

As an operator running a federation hub behind an LB, I want
`/readyz` to answer **503 during a deliberate drain** so the LB pulls the
hub before the listener closes — same contract a single-probe service
already gets:

|                               | `health.Probe` today | `federation.Prober` today | desired               |
| ----------------------------- | -------------------- | ------------------------- | --------------------- |
| Readiness 503 after mark      | ✅                   | ❌ (no mark exists)       | same                  |
| Fetch loop keeps serving 503s | ✅ (refresh loop)    | n/a (merge-on-read)       | ✅ (fetches continue) |
| Startup/liveness unaffected   | ✅                   | ✅                        | same                  |

## Suggested shape (not implementation dictatorship)

A `shuttingDown atomic.Bool` on `federation.Prober` plus a
`MarkShuttingDown()` method; the readiness handler overlays it the same
way the root probe's handlers overlay `shuttingDown`. Either the merge
keeps running during drain (fresh 503 bodies, mirroring the grace-period
idea), or the handler answers 503 from the flag alone without fetching —
both fix the LB lie; the first is nicer.

Out of scope for this ask: aggregate has the same hole in principle, but
an aggregate lives in-process with its sources — the caller can mark the
sources directly, which is what my demo does. The federation prober is
the one surface the caller cannot reach into.

## Why the workarounds don't cover it

- Stopping the dashboard (SSE) first: disconnects browsers, changes
  nothing about `/readyz` (measured: still 200 through the drain).
- Calling `Shutdown` on remotes from the hub: impossible — remotes are
  separate processes behind HTTP; the hub only holds URLs.
- Orchestrator `preStop` sleep: shifts the race, doesn't close it, and
  the hub still reports 200 while dying.

## Evidence commands:

```
go-health@v0.5.1: rg -n "MarkShuttingDown|draining" federation/  → 0 hits
live hub: SIGTERM with an SSE client held; /readyz polled through the
drain → all 200, listener closes at end of grace window
```

💘 Generated with Crush (Assisted-by: Crush:glm-5.3-flash)
