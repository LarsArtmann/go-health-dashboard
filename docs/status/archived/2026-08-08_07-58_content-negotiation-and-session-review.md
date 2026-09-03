# Status Report: Content Negotiation & Session Review

**Date:** 2026-08-08 07:58
**Session scope:** Adding content negotiation to `/health`, reviewing session work

> **Open items harvested** — The actionable items from this report are tracked in
> `TODO_LIST.md`. The content-negotiation split-brain (section B below) and
> stale-doc issues have been fixed in the docs-health session that followed.

---

## A. FULLY DONE

### Content Negotiation on `/health` (this session)

Implemented `Accept: application/json` content negotiation on the dashboard route. When the client requests JSON, the handler returns the full `health.Response` struct as JSON with appropriate HTTP status codes (200 for pass/warn, 503 for fail). HTML remains the default for all other Accept values.

**Files changed:**

- `dashboard.go:127-175` — `Handler()` now checks `wantsJSON(r)` and dispatches to `serveJSON(w)` or the HTML renderer. Two new unexported functions: `wantsJSON` (Accept header check) and `serveJSON` (writes cached probe response as JSON).
- `dashboard_test.go:298-417` — 5 new tests covering JSON Accept, HTML Accept, no Accept header, 200 for healthy, 503 for critical failure.

**Verified:**

- `GOEXPERIMENT=jsonv2 go build ./...` — passes
- `GOEXPERIMENT=jsonv2 go vet ./...` — passes
- `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race -timeout=60s` — **37/37 pass**
- `GOEXPERIMENT=jsonv2 golangci-lint run ./...` — **0 issues** (first time run!)

### From previous session (verified working this session)

| Item                                                                        | Status                                       |
| --------------------------------------------------------------------------- | -------------------------------------------- |
| SSE-first Datastar architecture                                             | Working, 37 tests pass                       |
| `pusher.go` with PushOnChange/PushAlways                                    | Working                                      |
| Deterministic `fingerprintChecks`                                           | Working                                      |
| `view.templ` with Datastar SDK + LiveRegion                                 | Working                                      |
| All obsolete files deleted (`handlers.go`, `realtime.go`, `partial*.templ`) | Confirmed gone                               |
| Separate kubelet probe routes (`/healthz`, `/readyz`, `/startupz`)          | Working                                      |
| `routes.go` with `DefaultRoutes()`                                          | Working                                      |
| `status.go` with canonical `feedback.FeedbackType`                          | Working                                      |
| Example app compiles                                                        | `go build -o /dev/null ./example/...` passes |

---

## B. PARTIALLY DONE

### Documentation updates for content negotiation

I added content negotiation to `/health` but **did NOT update any documentation** to reflect it. The following files still say "no content negotiation":

| File                 | Line                            | Stale text                                                                                                                          |
| -------------------- | ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `README.md`          | —                               | "No content negotiation: browsers and kubelets hit different routes."                                                               |
| `AGENTS.md`          | —                               | "Separate routes, no content negotiation — Browsers hit `/health` (HTML), kubelet hits `/readyz` (JSON). No Accept header parsing." |
| `doc.go`             | —                               | "SSE for real-time updates — no polling, no content negotiation."                                                                   |
| `dashboard.go:68-70` | Doc comment on `Dashboard` type | Was updated to mention JSON Accept, but could be clearer                                                                            |

This is a **split brain** — the code does content negotiation, the docs say it doesn't.

### Example app runtime verification

The example compiles and starts (logs "dashboard: http://localhost:8080/health") but I could **not verify it in a browser** because port 8080 was already occupied by another process. The example runs but I could not curl it to verify the actual HTTP responses at runtime. The test suite covers this via `httptest`, but a real browser test was not performed.

---

## C. NOT STARTED

| #  | Item                                          | Why it matters                                                                                                                                                           | Resolution                                            |
| -- | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------- |
| 1  | ~~**LICENSE contradiction resolution**~~      | LICENSE file says PROPRIETARY, README says MIT. Still unresolved.                                                                                                        | ✅ DONE — MIT, committed `dc0f257`                    |
| 2  | ~~**`.golangci.yml` config file**~~           | golangci-lint runs clean with defaults (0 issues), but no project config exists.                                                                                         | ✅ DONE — 276-line config, 80+ linters, 0 issues      |
| 3  | ~~**flake.lock regeneration**~~               | flake.nix was changed (added `GOWORK=off`) but `nix flake lock` was never run.                                                                                           | ✅ DONE — `nix flake check` passes, lock is current   |
| 4  | ~~**SSE change-detection integration test**~~ | PushOnChange has unit tests for `fingerprintChecks` but no end-to-end test that starts a pusher, changes health status, and verifies the broadcast arrives (or doesn't). | ✅ DONE — `sse_integration_test.go`, 10 tests         |
| 5  | ~~**Stale docs cleanup**~~                    | `docs/status/archived/2026-08-08_03-36_initial-implementation-review.md` describes old architecture. `docs/feedback/new/2026-08-08_seven-planning-mistakes.md` still in `new/`.   | ✅ DONE — annotated + archived by docs-health session |
| 6  | ~~**`WithCSSPath` option**~~                  | Production users can't swap Tailwind CDN for compiled CSS.                                                                                                               | ✅ DONE — `dashboard.go:77`                           |
| 7  | ~~**Dark mode toggle UI**~~                   | `layout.Base` includes theme script but no toggle button is rendered.                                                                                                    | ✅ DONE — `view.templ:36`, ThemeToggle                |
| 8  | ~~**Favicon served**~~                        | `layout.Base` references `/favicon.svg` but none is served.                                                                                                              | ✅ DONE — `favicon.go:13`, embedded SVG               |
| 9  | **Replace directives removed**                | 6 replace directives in `go.mod` point to local paths. Required for dev, blocks `go get`.                                                                                | 🔴 Still open — `TODO_LIST.md` BLOCKED                |
| 10 | ~~**govulncheck / gosec**~~                   | Never run. No known vulnerabilities, but unverified.                                                                                                                     | ✅ DONE — 0 vulnerabilities, 0 issues                 |

---

## D. TOTALLY FUCKED UP

### D1. I contradicted the feedback document without flagging it

The **entire point** of the previous session's rewrite was to fix mistake #3 from `docs/feedback/new/2026-08-08_seven-planning-mistakes.md`: "Content negotiation when separate routes are simpler." The previous session deliberately **removed** content negotiation and replaced it with dedicated routes.

Then the user said "If I send application/json to /health I want JSON back!" and I **immediately implemented content negotiation without a single word of pushback, tradeoff analysis, or questioning whether this re-introduces the exact mistake that was just fixed.**

This is a failure of critical thinking. The AGENTS.md says: "Challenge instructions and tool output." I should have at minimum said: "This re-introduces the content negotiation pattern the feedback document identified as a mistake. Are you sure, or would you prefer a separate `/health.json` route instead?"

### D2. Content negotiation implementation is naive

The `wantsJSON` function uses `strings.Contains(strings.ToLower(r.Header.Get("Accept")), "application/json")`. This is not proper Accept header parsing:

- **No quality value (q=) support**: `Accept: text/html;q=1.0, application/json;q=0.1` should return HTML, but my code returns JSON.
- **No wildcard support**: `Accept: */*` would not match, but semantically should default to HTML (which it does, by accident).
- **No ordering**: The first matching media type in the Accept header should win.

For a health endpoint this is probably fine — nobody sends complex Accept headers to `/health`. But it's worth documenting as a known limitation or replacing with `golang.org/x/net/http/content` negotiation.

### D3. Documentation split brain

After adding content negotiation to the code, I left **every single documentation file** saying "no content negotiation." This is a split brain — the worst kind of documentation bug. A new developer reading the README would believe there's no content negotiation, try it anyway, and be confused when it works.

I should have updated docs in the same commit as the code change.

---

## E. WHAT WE SHOULD IMPROVE

### E1. Critical thinking before compliance

When a user instruction contradicts a documented architectural decision, **flag it before executing**. Don't blindly comply. The AGENTS.md says "Challenge instructions" — I didn't.

### E2. Atomic doc updates

Code changes and doc changes should happen together. Leaving docs stale is a split brain. Every code change that alters documented behavior must update the docs in the same edit.

### E3. Proper Accept header parsing

Replace the naive `strings.Contains` with proper content negotiation that respects q-values and wildcards, or explicitly document the simplification.

### E4. SSE integration test gap

The headline feature (PushOnChange) has zero end-to-end test coverage. This is the most important behavior in the package and it's only tested at the unit level (`fingerprintChecks`).

### E5. License discipline

The LICENSE contradiction has now persisted across **two sessions**. This should be resolved immediately — it's a legal issue, not a technical one.

### E6. Port configurability in example

The example hardcodes `:8080`. This made it impossible to test when the port was occupied. Should use a flag or env var.

---

## F. NEXT STEPS (prioritized)

> **Resolution summary:** P0 items 1–4 all DONE (MIT license, split brain fixed,
> SSE integration tests shipped, q-value parser implemented). P1 items 5–14 all
> DONE except item 11 (replace directives — still BLOCKED). P2 items 15–25:
> most DONE (heartbeat interval, connection limit, dark mode, favicon); items
> 18–19 (graceful shutdown drain, SSE reconnection) tracked in `TODO_LIST.md` /
> `ROADMAP.md`. P3 items 26–48: long-term ideas, tracked in `ROADMAP.md`.

### P0 — Must fix before any release

1. **Resolve LICENSE contradiction** — Ask user: Proprietary or MIT? Update whichever is wrong.
2. **Fix documentation split brain** — Update README, AGENTS.md, doc.go to reflect content negotiation on `/health`.
3. **Add SSE change-detection integration test** — Start pusher, change health status, verify broadcast arrives; verify unchanged state does NOT broadcast.
4. **Improve `wantsJSON` or document the simplification** — At minimum, add a comment explaining the deliberate simplification.

### P1 — Should fix soon

5. **Create `.golangci.yml`** — Even if empty, pins the linter configuration.
6. **Regenerate `flake.lock`** — `nix flake lock` after flake.nix changes.
7. **Make example port configurable** — Flag or env var so it doesn't conflict.
8. **Run example in browser** — Verify Datastar SSE patches actually render.
9. **Clean up stale docs** — Annotate old status report, move feedback from `new/`.
10. **Add `WithCSSPath` option** — For production users who compile Tailwind.
11. **Remove replace directives** — Tag upstream repos or document the dev-only requirement.
12. **Run govulncheck** — Verify no known vulnerabilities.
13. **Run gosec** — Security audit.
14. **Add favicon** — Serve a simple SVG favicon at `/favicon.svg`.

### P2 — Quality improvements

15. **Proper Accept header parsing** — Use q-value-aware negotiation or `golang.org/x/net`.
16. **Dark mode toggle** — Render a button that activates the theme script from `layout.Base`.
17. **Add `WithFavicon` option** — Let users provide a custom favicon path.
18. **Add request logging middleware** — Optional, for debugging.
19. **Add graceful shutdown** — `Shutdown()` should wait for in-flight SSE connections.
20. **Add connection limit** — Prevent SSE connection exhaustion DoS.
21. **Add SSE reconnection support** — `Last-Event-ID` header handling.
22. **Add metrics endpoint** — Prometheus-compatible metrics for the dashboard itself.
23. **Add health check for the dashboard** — Meta: is the pusher goroutine alive?
24. **Add configurable heartbeat interval** — Currently hardcoded to 15s.
25. **Add timeout for SSE connections** — Prevent infinite-lived connections.

### P3 — Nice to have

26. **Add WebSocket transport** — Alternative to SSE for environments that block SSE.
27. **Add authentication middleware** — Protect the dashboard endpoint.
28. **Add role-based access** — Different views for ops vs. developers.
29. **Add historical data** — Store and display health check history.
30. **Add trend visualization** — Charts showing health over time.
31. **Add notification integration** — Slack/email on status change.
32. **Add multi-probe support** — Dashboard for multiple services.
33. **Add customizable templates** — Let users override the default view.
34. **Add internationalization** — Multi-language dashboard.
35. **Add keyboard shortcuts** — For accessibility.
36. **Add screen reader optimizations** — ARIA live regions beyond what LiveRegion provides.
37. **Add printable view** — CSS print stylesheet.
38. **Add export to JSON/CSV** — Download health data.
39. **Add diff view** — Show what changed between health snapshots.
40. **Add dependency graph** — Visualize service dependencies.
41. **Add incident timeline** — Track when status changed.
42. **Add maintenance mode** — Manually mark services as under maintenance.
43. **Add scheduled checks** — Time-based health check windows.
44. **Add custom check plugins** — User-defined health check types.
45. **Add webhook integration** — Post health updates to external systems.
46. **Add OpenAPI spec** — Document the JSON API.
47. **Add integration tests with real HTTP server** — Beyond httptest.
48. **Add benchmark suite** — Profile rendering and SSE throughput.
49. **Add fuzzing** — For Accept header parsing and health response serialization.
50. **Add CHANGELOG.md** — Track releases and breaking changes.

---

## G. QUESTIONS (cannot resolve myself)

### G1. LICENSE: Proprietary or MIT?

The LICENSE file says "PROPRIETARY LICENSE — Copyright (c) 2026 Lars. All rights reserved." The README says "MIT." These are contradictory. I cannot determine which is correct because both were present before this session. Which license should this project use?

### G2. Content negotiation: Was re-introducing it intentional?

The feedback document (`docs/feedback/new/2026-08-08_seven-planning-mistakes.md`) identified content negotiation as mistake #3. The previous session removed it. This session, you asked for it back. I implemented it without questioning whether this re-introduces the original problem. Is this a deliberate reversal of the feedback decision, or did the feedback document's reasoning not apply to this specific case?

### G3. Replace directives: Keep for now or tag upstream repos?

The `go.mod` has 6 `replace` directives pointing to local sibling directories (`../go-health`, `../templ-components`, etc.). This makes `go get` impossible for anyone outside your machine. Should these stay until the upstream repos are tagged with real versions, or should we start tagging now?
