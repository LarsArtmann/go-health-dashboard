# Research — measurement sweep: render cost, SSE payload, watchdog floor, Chrome latency, load (2026-10-25)

|             |                                                                                                                                                                                                                                                                                                     |
| ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Date**    | 2026-10-25                                                                                                                                                                                                                                                                                          |
| **Method**  | In-repo `go test -bench` / env-guarded load harness runs in the flake devShell (go 1.27.x, `GOEXPERIMENT=jsonv2`, `GOWORK=off`), on the v0.11.0 tree; throwaway helper binaries (`cmd/measure-patch`) used for payload capture and then trashed                                                     |
| **Context** | TODO rows f-11/f-12 (SSE payload + render delta vs v0.10.2 after the icon-glyph markup rework), the S-sized quality batch (BenchmarkHealthCheck, Chrome-latency re-measure), and the LOADTEST evidence-era re-run. Closes the "record the numbers in `docs/research/`" remainder of the payload row |

## Render cost (`go test -bench`, this tree vs the 2026-09 baselines)

| Benchmark                          | 2026-09 baseline | 2026-10-25 (v0.11.0 tree)          | Reading                                                                                                      |
| ---------------------------------- | ---------------- | ---------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `BenchmarkHandler_HTMLRendering`   | 40.4 µs/op       | 44.0–54.9 µs/op                    | +9–36% — the icon-tile stat cards, filter chips, evidence strip, and disclosure badges all render per page   |
| Full HTML page (View render)       | —                | 74.3–118 µs/op                     | includes layout shell + SDK script tag                                                                       |
| Dashboard + `WithTrend`            | —                | 120–131 µs/op                      | trend ring copy + sparkline markup                                                                           |
| `BenchmarkHealthCheck` (lifecycle) | —                | **68.10 ns/op, 66 B/op, 0 allocs** | steady-state `HealthCheck` is a pointer load + two bool reads; the watchdog re-baseline made this measurable |

The HTML delta is page-render cost only, once per page load; the per-tick SSE
path (below) is what multiplies with clients. No regression gate is implied —
these numbers restore the comparison point the markup rework lost (row f-12).

## SSE patch payload (throwaway `cmd/measure-patch`, trashed after capture)

- Static floor per `dashboardContent` patch: **~2,438 B** (banner, evidence
  strip, groups, filter hint — even with zero checks).
- Per check row: **~2.5 KB** (badge + disclosure markup + metadata line +
  error cell); a pass row with no error runs ~600 B, the evidence strip adds
  ~470 B per broadcast.
- 12-check aggregate page: **36.7 KB/event**, full HTML 17.2 KB → 50.5 KB
  with patches — the glyph rework (inline SVGs instead of icon-font classes)
  is the dominant per-row cost driver.

Operational reading: at PushAlways with a 2 s cadence and 12 checks, one
client draws ~18 KB/s; PushOnChange (the default) amortizes this to changes
only. `WithTrend` does not add payload (ring is server-side).

## Chrome launch latency (browser suite serialization cost)

- 2026-09-23 measured ~155 ms/launch (M91 evidence doc).
- 2026-10-25 re-measure (`TestMeasureChromeLaunchLatency`, isolated run):
  **85.45 ms average**. The serialized-launch budget in `browserSerial`
  stays comfortably inside the 45 s announce timeout; no action.

## Load harness (`TestLoad_Aggregate20Sources`, `LOADTEST=1`, env-parameterized)

Configuration: 1003 events observed over the 5 s window, ~50 events/client,
first event 10.9 ms after connect. Scrape leg: p50 **1.25 ms**, p95
**7.44 ms**, max 16.3 ms. Verdict: no regression vs the 2026-09-23
evidence-era run recorded in `docs/research/` — the federation hub path and
the JSON scrape path both hold sub-10 ms p95 under aggregate fan-in.

## Semantics pinned along the way (corrections worth keeping)

- **Rate-limit `Retry-After`** is `ceil((1 − tokens)/refill)` — when the
  bucket is empty it reads as the full inter-token wait, NOT always 1 (the
  README previously said "always reads 1"; corrected in this batch).
- `resolvePushInterval` guarantees a positive interval (configured > 0 →
  probe > 0 → `defaultPushInterval` 2 s), which is why the dead
  `push.interval > 0` condition was removed from `HealthCheck` — pinned by
  `TestResolvePushInterval` (positivity invariant case).
- `health.Status` rank → `statusValue`: pass=1, off=1, warn=0.5, fail=0
  (off is pass-tier, never rolls up) — pinned by
  `TestStatusValue_PinsEveryStatus`.

## Consumer sweep (v0.11.0 adoption state, verified 2026-10-25)

| Consumer                                                                                                     | Pin     | State                                                                                                                                                                                                                                                                                                                    |
| ------------------------------------------------------------------------------------------------------------ | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| DiscordSync, company-master-data, file-and-image-renamer, go-taskqueue, library-policy, timesheets, webphone | v0.11.0 | already current (no action)                                                                                                                                                                                                                                                                                              |
| nsfw-classifier                                                                                              | v0.10.2 | **bumped to v0.11.0** (`e102344e`, rides go-health v0.7.0). Baseline has two pre-existing ONNX real-model failures (`internal/face`, `internal/skin`) that do NOT import the dashboard — unrelated, recorded, not fixed here                                                                                             |
| dnsblockd                                                                                                    | v0.10.1 | **BLOCKED upstream** — their lockstep gate needs go-datastar root/static exact-match pins; upstream tagged root v0.6.2 but never `static/v0.6.2` (verified via `git ls-remote` 2026-10-25: newest static tag is v0.6.1). Unblock is an owner act (push the tag or decide sub-train tolerance), tracked as dnsblockd T331 |
| CV                                                                                                           | v0.10.2 | BLOCKED row in our TODO_LIST — user's deploy call, not bumped                                                                                                                                                                                                                                                            |

pkg.go.dev: v0.11.0 rendered and marked Latest (spot-check closed 2026-10-25).

## Tooling note

`cmd/measure-patch/` was a throwaway capture binary, created for the SSE
payload numbers and removed (`trash`) after the numbers landed in this file —
stale editor diagnostics about it are expected to linger until LSP restart.
