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

Everything else worth doing is in `ROADMAP.md` as raw ideas.
