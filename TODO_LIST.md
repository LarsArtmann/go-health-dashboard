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

## High Impact

| Task                                                              | Status    | Impact | Effort | Evidence                                                                                                                                                                    |
| ----------------------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Add SSE change-detection integration test                         | 🔴 `TODO` | High   | 1h     | `pusher.go:116-133` — PushOnChange has only unit tests for `fingerprintChecks`, no E2E test verifying broadcast arrives on status change and does NOT arrive when unchanged |
| Improve `wantsJSON` Accept parsing or document the simplification | 🔴 `TODO` | High   | 30min  | `dashboard.go:154-158` — uses naive `strings.Contains`, no q-value sorting or wildcard support                                                                              |

## Medium Impact

| Task                                                   | Status    | Impact | Effort | Evidence                                                                          |
| ------------------------------------------------------ | --------- | ------ | ------ | --------------------------------------------------------------------------------- |
| Add `WithCSSPath` option                               | 🔴 `TODO` | Med    | 30min  | `view.templ:69` — Tailwind CDN hardcoded, no way to swap for compiled CSS         |
| Add `.golangci.yml` config                             | 🔴 `TODO` | Med    | 15min  | Linter runs clean (0 issues) but no project config to pin enabled linters         |
| Make example app port configurable                     | 🔴 `TODO` | Med    | 15min  | `example/main.go:61` — hardcodes `:8080`, conflicts with other servers            |
| Add dark mode toggle button                            | 🔴 `TODO` | Med    | 30min  | `view.templ` — `layout.Base` includes theme script but no toggle button rendered  |
| Add favicon endpoint                                   | 🔴 `TODO` | Med    | 15min  | `layout.Base` references `/favicon.svg` but none is served                        |
| Verify CSP nonce end-to-end                            | 🔴 `TODO` | Med    | 1h     | `view.templ:65-74` — nonce attributes present in templ but untested with real CSP |
| Document or improve SSE handler context test fragility | 🔴 `TODO` | Med    | 30min  | `dashboard_test.go:458,489` — 200ms timeout context, could flake on slow CI       |
