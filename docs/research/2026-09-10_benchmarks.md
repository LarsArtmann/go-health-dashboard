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
