# Research — 20-source aggregate load test (F8)

|              |                                            |
| ------------ | ------------------------------------------ |
| **Date**     | 2026-09-10                                 |
| **Harness**  | `TestLoad_Aggregate20Sources` (`loadtest_test.go`), run with `LOADTEST=1 nix develop -c go test -run TestLoad_Aggregate20Sources -v .` |
| **Machine**  | Lars's dev box (nixpkgs, go 1.26.7)        |

## Setup

- 20 go-health probes × 3 healthy checks each (60 namespaced checks), 100ms
  refresh interval, merged through `aggregate.New` into one dashboard.
- Dashboard: `PushAlways`, 100ms push interval, metrics + trend enabled.
- Load: 20 concurrent SSE clients streaming for 5s + 8 scrape workers × 25
  requests alternating `/health` and `/health/metrics` (200 scrapes total).

## Results

| Metric                          | Value      |
| ------------------------------- | ---------- |
| SSE events delivered (5s)       | 1002 (~50 per client; 10/s at the 100ms interval — zero loss) |
| Time to first SSE event         | 7.65ms     |
| Scrape latency p50              | 1.05ms     |
| Scrape latency p95              | 5.49ms     |
| Scrape latency max              | 12.56ms    |
| Subscriber ceiling observed     | 20/20 concurrent without 503s (limit unset) |

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
