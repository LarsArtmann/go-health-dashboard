# Research — 20-source aggregate load test (F8)

|             |                                                                                                                                        |
| ----------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| **Date**    | 2026-09-10                                                                                                                             |
| **Harness** | `TestLoad_Aggregate20Sources` (`loadtest_test.go`), run with `LOADTEST=1 nix develop -c go test -run TestLoad_Aggregate20Sources -v .` |
| **Machine** | Lars's dev box (nixpkgs, go 1.26.7)                                                                                                    |

## Setup

- 20 go-health probes × 3 healthy checks each (60 namespaced checks), 100ms
  refresh interval, merged through `aggregate.New` into one dashboard.
- Dashboard: `PushAlways`, 100ms push interval, metrics + trend enabled.
- Load: 20 concurrent SSE clients streaming for 5s + 8 scrape workers × 25
  requests alternating `/health` and `/health/metrics` (200 scrapes total).

## Results

| Metric                      | Value                                                         |
| --------------------------- | ------------------------------------------------------------- |
| SSE events delivered (5s)   | 1002 (~50 per client; 10/s at the 100ms interval — zero loss) |
| Time to first SSE event     | 7.65ms                                                        |
| Scrape latency p50          | 1.05ms                                                        |
| Scrape latency p95          | 5.49ms                                                        |
| Scrape latency max          | 12.56ms                                                       |
| Subscriber ceiling observed | 20/20 concurrent without 503s (limit unset)                   |

## Reading

A 20-source aggregate page with a full browser-tab fan-out is nowhere near
the library's limits: the push loop renders ~60-check patches ten times a
second for twenty clients while scrapes stay in single-digit milliseconds.
The `WithMaxSSEConnections`/`WithRateLimit` knobs are the right tools when
this stops being true (public internet exposure), not before.

The harness is env-guarded (`LOADTEST=1`) and skipped in CI: five seconds of
twenty open streams is research time, not gate time. Re-run it after any
pusher/broadcaster change; the logged `load:`/`sse:`/`scrape latency:` lines
are the comparison baseline.

## Re-run on the evidence-era render (v0.10.1, 2026-09-23)

|             |                                                                                                                              |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------- |
| **Date**    | 2026-09-23                                                                                                                   |
| **Harness** | Same test, now env-parameterized (M92): `LOADTEST_SOURCES/CHECKS/CLIENTS/SCRAPERS/SCRAPES/DURATION/PUSH`                    |
| **Context** | The 2026-09-10 numbers predate two render additions: per-check metadata lines (v0.9.0) and the evidence strip + tooltips (v0.9.x). This re-run measures the CURRENT render, evidence log accruing per tick, under the default fixture and a shaped 30×4/40-client variant. |

### Default fixture (20×3, 20 clients, 100ms push, 5s)

| Metric                      | Value                                    |
| --------------------------- | ---------------------------------------- |
| SSE events delivered (5s)   | 1002 (~50/client — zero loss)            |
| Time to first SSE event     | 9.06ms                                   |
| Scrape latency p50          | 2.24ms                                   |
| Scrape latency p95          | 6.16ms                                   |
| Scrape latency max          | 13.22ms                                  |

### Shaped variant (30 sources × 4 checks = 120 checks, 40 SSE clients, 300 scrapes, 8s)

| Metric                      | Value                                    |
| --------------------------- | ---------------------------------------- |
| SSE events delivered (8s)   | 3216 (~80/client — zero loss)            |
| Time to first SSE event     | 45.70ms                                  |
| Scrape latency p50          | 5.75ms                                   |
| Scrape latency p95          | 30.43ms                                  |
| Scrape latency max          | 56.56ms                                  |

### Reading

- The metadata + evidence render additions moved the p50 from ~1ms to
  ~2.2ms — measurable but irrelevant at dashboard scale; p95 stays under
  10ms at the documented fixture.
- Doubling sources and clients (30×4, 40 streams) keeps p50 under 6ms;
  the tail stretches (~30–57ms) under the larger fan-out but no client
  loses events and no scrape errors.
- The evidence strip accrues observations on every pusher tick, so this
  run is the honest apples-to-apples successor of the 2026-09-10
  baseline: same harness shape, current render.
