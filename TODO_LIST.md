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

| Task                      | Status       | Why blocked                                                                                             | Evidence       |
| ------------------------- | ------------ | ------------------------------------------------------------------------------------------------------- | -------------- |
| Remove replace directives | 🔵 `BLOCKED` | Upstream repos (go-health, templ-components, etc.) untagged on GitHub. Keep for local dev until tagged. | `go.mod:21-31` |

## Recently Completed (v0.1.0 push)

All v0.1.0 TODO items are complete. See `CHANGELOG.md` for the full list.

## High Impact

| Task                                               | Status    | Impact | Effort | Evidence                                        |
| -------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------- |
| Add `WithCSSPath` option for compiled CSS          | 🟢 `DONE` | Med    | 30min  | `dashboard.go:77`, `view.templ:65-83`           |
| Add dark mode toggle button                        | 🟢 `DONE` | Med    | 30min  | `view.templ:27-35`                              |
| Add favicon endpoint                               | 🟢 `DONE` | Med    | 30min  | `favicon.go`, `routes.go:8`                     |
| Verify CSP nonce end-to-end                        | 🟢 `DONE` | Med    | 1h     | `csp_test.go` — 8 tests, all pass with `-race`  |
| Add `WithHeartbeatInterval` + SSE connection limit | 🟢 `DONE` | Med    | 1h     | `dashboard.go:84,91`, `pusher.go:41-43,143-175` |
| Add CI/CD GitHub Actions                           | 🟢 `DONE` | Med    | 30min  | `.github/workflows/ci.yml`                      |
| Add Dependabot config                              | 🟢 `DONE` | Low    | 10min  | `.github/dependabot.yml`                        |

## Low Impact / Future Work

| Task                                             | Status    | Impact | Effort | Notes                  |
| ------------------------------------------------ | --------- | ------ | ------ | ---------------------- |
| Build-tag gating for SSE code                    | 🔴 `TODO` | Med    | 90min  | Blocked on D3 decision |
| Embeddable dashboard mode (sub-path mounting)    | 🔴 `TODO` | Med    | 60min  |                        |
| Auth middleware integration                      | 🔴 `TODO` | Med    | 60min  |                        |
| Prometheus metrics endpoint                      | 🔴 `TODO` | Low    | 90min  |                        |
| Health history / sparkline visualization         | 🔴 `TODO` | Low    | 90min  |                        |
| SSE reconnection support (Last-Event-ID)         | 🔴 `TODO` | Med    | 60min  |                        |
| UI flexibility options (WithHideStatCards, etc.) | 🔴 `TODO` | Low    | 90min  |                        |
| Fuzzing for Accept header parsing                | 🔴 `TODO` | Low    | 30min  |                        |
