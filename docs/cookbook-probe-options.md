# Cookbook: probe-side option combos

Validated recipes for combining go-health probe options with the
dashboard. Option names are verified against go-health v0.1.3; run
`nix run .#example` after adapting a recipe to confirm behavior in your
setup.

## Recipe 1: Hardened public probe (`WithGETOnly` + `WithAllowedMethods` + `WithTimeout`)

When the probe endpoints are exposed beyond localhost (behind an
ingress that forwards arbitrary methods), lock the request surface down
on the probe side:

```go
probe := health.New(injector,
    health.WithCriticalServices("database"),
    health.WithRefreshInterval(100*time.Millisecond),
    health.WithGETOnly(),                                  // reject non-GET on /healthz & co
    health.WithAllowedMethods(http.MethodGet),             // explicit allow-list (empty default: GET+HEAD)
    health.WithTimeout(2*time.Second),                     // fail checks that hang longer
)
dash := dashboard.New(probe)
```

Why both options exist: the allow-list always includes GET; list extra
methods (HEAD, OPTIONS) only when infrastructure probes need them.
`WithAllowedMethods` implies `WithGETOnly`, so passing both is harmless
but redundant — plain `WithGETOnly` is the strict public posture.
`WithTimeout` bounds each check so a hung dependency flips the status
to `fail` instead of wedging the refresh tick.

Dashboard interplay: the dashboard reads `CachedResponse()` only, so
probe-side request restrictions never affect the HTML/SSE surface —
they only govern the Kubelet-style endpoints the probe itself serves.

## Recipe 2: Observability hooks (`WithEvaluationHook` + `WithHealthRecorder`)

To ship every evaluation into your own metrics/tracing pipeline without
scraping:

```go
probe := health.New(injector,
    health.WithRefreshInterval(100*time.Millisecond),
    health.WithEvaluationHook(func(resp health.Response) {
        // fire on every completed evaluation cycle
        statsd.Gauge("health.status", statusValue(resp.Status))
    }),
    health.WithHealthRecorder(recorder), // receives the same history the probe serves
)
```

Why it matters with the dashboard: the dashboard's own SSE/trend
machinery samples `CachedResponse()` at `WithPushInterval` cadence,
which is a UI decision. If your alerting needs every probe tick (not
every UI frame), put it on `WithEvaluationHook` — the two cadences are
independent by design.

## Recipe 3: Multiple replicas behind one load balancer (`WithLiveThrottle` + `WithInstanceID`)

```go
probe := health.New(injector,
    health.WithVersion(version),        // surfaced in the JSON response
    health.WithInstanceID(instanceID),  // e.g. the pod name — distinguishes replicas
    health.WithLiveThrottle(50*time.Millisecond), // min interval between expensive liveness evaluations
)
dash := dashboard.New(probe, dashboard.WithTrend(300))
```

`WithInstanceID` marks this replica in the probe's JSON so a load
balancer serving `/health` to many backends can attribute results.
`WithLiveThrottle` caps live (cache-miss) evaluations: within the
window after an evaluation, handlers serve the stored result instead of
re-running the batch — without it, live mode is a DoS amplifier on an
exposed endpoint. It only applies in live mode: once you run a
background cache (`WithRefreshInterval` > 0 plus `Probe.Start`), the
throttle is a no-op and the cache absorbs the load. Pair with a longer
dashboard `WithTrend(300)` so the trend card shows the replicas'
history, not just the last few seconds.

## Anti-patterns

- **Throttling to mask a slow check** — `WithLiveThrottle` is for read
  amplification, not for hiding a 10-second dependency. Fix the check
  or set `WithTimeout`.
- **Recording through the dashboard** — the dashboard's trend history
  is presentation state (a bounded ring). Anything you might alert on
  later belongs in `WithHealthRecorder`, not in the UI buffer.
