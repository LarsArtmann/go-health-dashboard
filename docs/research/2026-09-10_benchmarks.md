# Research — render benchmark re-baseline (post-0.7.x UI)

|             |                                                                                                                                                                          |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Date**    | 2026-09-10                                                                                                                                                               |
| **Command** | `nix develop -c go test -bench 'BenchmarkHandler_HTMLRendering\|BenchmarkDashboard_FullHTML' -benchtime 2s -run xxx -count 3 .`                                          |
| **Context** | First baseline after the render grew the filter expressions, connection pill, persistence script, and source grouping (plan f21: the old baseline predated the UI work). |

## Results (3 runs each)

| Benchmark                   | ns/op (min–max)   | B/op   | allocs/op |
| --------------------------- | ----------------- | ------ | --------- |
| Handler_HTMLRendering       | 29,028 – 99,236   | —      | —         |
| Dashboard_FullHTML          | 185,607 – 222,947 | 70,754 | ~574      |
| Dashboard_FullHTMLWithTrend | 183,608 – 199,296 | 70,869 | ~577      |

## Reading

- Full-page renders land at ~0.2ms and ~71KB / ~575 allocations per render —
  the filter/pill/persistence additions are noise at this scale (the page
  carries a ~60-check table in the realistic fixtures these benchmarks use).
- `Handler_HTMLRendering` is visibly bimodal across runs (29µs vs 84/99µs);
  it exercises the full HTTP path including the probe cache read, so GC and
  scheduling dominate its spread. Compare medians, not mins.
- At the measured load-test cadence (20 clients × 10 renders/s) this is
  ~40ms/s of render CPU — far from being the bottleneck (see
  `2026-09-10_aggregate-load-test.md`).

## Re-baseline — per-check metadata stamping (v0.9.0)

|             |                                                                                                                                                                                             |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Date**    | 2026-09-17                                                                                                                                                                                  |
| **Command** | `GOEXPERIMENT=jsonv2 GOWORK=off go test -run xxx -bench 'BenchmarkHandler_HTMLRendering\|BenchmarkDashboard_FullHTML' -benchtime 2s -count 3 .` at HEAD, then identically in a `v0.8.1` worktree (the last tree before `buildViewModelAt` stamped per-check metadata), plus a stamping-loop micro-benchmark (`BenchmarkBuildViewModelAt_Stamping`, `-benchtime 1s -count 3`). |
| **Context** | The go-health v0.2.0 consumer upgrade added a per-tick stamping loop (SinceText/DurationText derived once per build) and a metadata line per table row. Plan f7 asked for a before/after run before calling the cost negligible. |

### Handler-level: v0.8.1 → HEAD (medians of 3, same machine)

| Benchmark                       | v0.8.1           | HEAD             | Δ                          |
| ------------------------------- | ---------------- | ---------------- | -------------------------- |
| Handler_HTMLRendering ns/op     | 40,355           | 52,198           | +~12µs (bimodal on both)   |
| Dashboard_FullHTML ns/op        | 69,721           | 113,539          | +~44µs                     |
| Dashboard_FullHTML B/op         | 105,196          | 106,291          | +~1.1KB                    |
| Dashboard_FullHTML allocs/op    | 571              | 585              | +14                        |
| FullHTMLWithTrend ns/op         | 59,610           | 113,539          | +~54µs                     |

### Stamping loop in isolation (60-check build, 3 runs each)

| Variant        | ns/op         | B/op    | allocs/op |
| -------------- | ------------- | ------- | --------- |
| no metadata    | 9,298–9,420   | 20,664  | 72        |
| with metadata  | 21,388–22,199 | 25,508  | 432       |

### Reading

- The stamping loop itself costs **~0.21µs, 80B, and 6 allocs per row**
  (12.4µs / 60 rows) — microseconds per pusher tick against a default
  2s interval. Per-tick stamping is confirmed negligible.
- The handler-level delta is larger than the loop: each row now renders
  the metadata span (templ scaffolding + title attribute), and rows with
  probe-set `Since` (the benchmarks' real probes set it) produce the
  Sprintf'd text. ~+14 allocs on the 2-check FullHTML fixture matches
  ~6 allocs/row from the micro-benchmark.
- At the aggregate load-test cadence (200 renders/s) the added render
  CPU is roughly 5–10ms/s — not a bottleneck, but the v0.9.0 render is
  honestly ~25–60% more expensive per page than the 2026-09-10 baseline
  at these fixtures. Re-baseline future render work against THESE
  numbers, not the 2026-09-10 table.
- Cross-tree runs on a loaded shared machine inherit its noise;
  `Handler_HTMLRendering` remains documented-bimodal. Medians, not mins.
