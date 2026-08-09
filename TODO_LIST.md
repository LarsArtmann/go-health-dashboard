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

| Task                     | Status       | Why blocked                                                                                                       | Evidence                    |
| ------------------------ | ------------ | ----------------------------------------------------------------------------------------------------------------- | --------------------------- |
| Build-tag gating for SSE | 🔵 `BLOCKED` | Consumers who only want HTML shouldn't need GOEXPERIMENT=jsonv2. Requires decision: accept, fork go-sse, or gate. | `ROADMAP.md` Open Questions |

## High Impact

| Task                                          | Status    | Impact | Effort | Evidence                                        |
| --------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------- |
| Add test for `SubscriberCount()`              | 🔴 `TODO` | High   | 15min  | `dashboard.go:288` — method has no dedicated test |
| Add test for `WithHeartbeatInterval`          | 🔴 `TODO` | High   | 15min  | `dashboard.go:103` — custom interval unverified  |
| Document `WithNonceExtractor` in README       | 🔴 `TODO` | High   | 10min  | `dashboard.go:84` — v0.2.0 feature absent from README options |
| Add screenshot to README                      | 🔴 `TODO` | Med    | 30min  | No visual preview in README                      |

## Medium Impact

| Task                                     | Status    | Impact | Effort | Notes                                    |
| ---------------------------------------- | --------- | ------ | ------ | ---------------------------------------- |
| SSE reconnection support (Last-Event-ID) | 🔴 `TODO` | Med    | 60min  | See ROADMAP.md for design                |
| Embeddable dashboard mode (sub-path)     | 🔴 `TODO` | Med    | 60min  | Mount under non-root prefix              |
| Auth middleware integration              | 🔴 `TODO` | Med    | 60min  | Protect dashboard endpoint               |
| SSE nonce flow integration test          | 🔴 `TODO` | Med    | 30min  | SSE patches should carry nonce if they contain scripts |

## Low Impact / Future Work

| Task                                             | Status    | Impact | Effort | Notes                       |
| ------------------------------------------------ | --------- | ------ | ------ | --------------------------- |
| Fuzzing for Accept header parsing                | 🔴 `TODO` | Low    | 30min  | `dashboard.go:184`          |
| Fuzzing for health response serialization        | 🔴 `TODO` | Low    | 30min  |                             |
| Prometheus metrics endpoint                      | 🔴 `TODO` | Low    | 90min  | See ROADMAP.md              |
| Health history / sparkline visualization         | 🔴 `TODO` | Low    | 90min  | See ROADMAP.md              |
| UI flexibility options (WithHideStatCards, etc.) | 🔴 `TODO` | Low    | 90min  | See ROADMAP.md              |
| Headless-browser CSP test (chromedp)             | 🔴 `TODO` | Low    | 90min  | Close the runtime-CSP verification loop |
