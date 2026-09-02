# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, see ROADMAP.md.
> Items are ranked by impact. Status is verified, not assumed.

## Status legend

| Status           | Meaning                                                     |
| ---------------- | ----------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                   |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                   |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed.     |
| 🟢 `DONE`        | Completed. Remove from this list and log in `CHANGELOG.md`. |

## Next Up

Full breakdown: `docs/planning/2026-09-02_21-21_pareto-execution-plan-v03-cycle.md`
(Pareto tiers, 32 medium tasks, 166 fine tasks ≤12 min, execution graph).

| Task                                             | Status     | Impact | Effort | Notes                                                                    |
| ------------------------------------------------ | ---------- | ------ | ------ | ------------------------------------------------------------------------ |
| Release v0.3.1 (bump Version, date CHANGELOG, tag) | 🟢 `DONE` | High   | 45min  | v0.3.0 was a stray early tag (cached by the Go module proxy); real release retargeted to v0.3.1 |
| CI: browser-test job (chromium + env)            | 🟢 `DONE` | High   | 60min  | Plus nightly fuzz workflow and coverage totals in the test job           |
| `RecommendedCSP(nonce)` helper + tests + README  | 🟢 `DONE` | High   | 45min  | `csp.go`; invalid nonce tokens omitted so the header can never be malformed |
| Browser-test hardening (console asserts, live patch DOM check) | 🟢 `DONE` | High | 72min  | Plus axe-core a11y audit and serialized Chrome launches                  |
| Fix gopls go1.27 stdversion warning noise        | 🟢 `DONE` | Medium | 30min  | Committed `.vscode/settings.json` sets gopls build env                   |
| Fuzz targets (escape, fingerprint) + nightly fuzz | 🟢 `DONE` | Medium | 60min  | Found + fixed a real fingerprint delimiter collision                     |
| promtool conformance check for metrics           | 🟢 `DONE` | Medium | 45min  | Official prometheus/common parser test always runs; promtool lint when on PATH |
| SSE hardening (drain, lifetime, watchdog, rate limit) | 🟢 `DONE` | High | 120min | `WithShutdownDrain`, `WithMaxConnectionLifetime`, `WithRateLimit`, `ErrPusherStale` watchdog |
| Example app v2 (auth/metrics/trend env toggles)  | 🔴 `TODO` | Medium | 60min  | Onboarding for v0.3.1 features                                           |
| HARVEST plan items into this list + ROADMAP      | 🔴 `TODO` | High   | 30min  | docs-health HARVEST over the plan's section (f)                          |

Everything beyond this shortlist is triaged in the plan file (hardening, observability, polish, spikes).

## Blocked (needs user decision)

| Task                     | Status       | Why blocked                                                                                                      | Evidence                    |
| ------------------------ | ------------ | ---------------------------------------------------------------------------------------------------------------- | --------------------------- |
| Build-tag gating for SSE | 🔵 `BLOCKED` | Consumers who only want HTML shouldn't need GOEXPERIMENT=jsonv2. Requires decision: accept, fork go-sse, or gate. | `ROADMAP.md` Open Questions |

## Done (recent)

Completed in the current cycle, logged in `CHANGELOG.md`:

- Auth middleware integration (`WithMiddleware`) — protects dashboard-owned
  routes, kubelet probes stay open
- Prometheus metrics endpoint (`WithMetrics` + `MetricsHandler`)
- Health history / trend sparkline (`WithTrend`) + UI flexibility
  (`WithHideStatCards`)
- Fuzzing for Accept header parsing and health response serialization
  (`fuzz_test.go`)
- Headless-browser CSP test (`browser_test.go`, chromedp) — closes the
  runtime-CSP verification loop; found the Datastar `unsafe-eval` requirement
- Screenshot in README (`docs/screenshot.png`, captured by
  `screenshot_test.go`)
- SSE hardening options: `WithShutdownDrain`, `WithMaxConnectionLifetime`,
  `WithRateLimit` (hand-rolled token bucket), pusher staleness watchdog
  (`ErrPusherStale`)
- `RecommendedCSP(nonce)` helper, metrics conformance tests (official
  parser + promtool), browser console/CSP/live-patch/a11y verification,
  fuzz targets with nightly workflow, CI browser job, gopls env fix

Everything else worth doing is in `ROADMAP.md` as raw ideas.
