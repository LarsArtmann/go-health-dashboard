# Feedback: go-health-dashboard Planning — Seven Architectural Mistakes in One Plan

**Date:** 2026-08-08
**Project:** go-health-dashboard
**Task:** Plan the go-health-dashboard composition layer (go-health + templ-components + go-datastar)
**Outcome:** v1 plan shipped with 7 architectural mistakes across dependency analysis, real-time strategy, routing design, API correctness, and architecture simplification. User caught them on review ("why are we not using go-datastar, what else is fucked up?"). Full rewrite required.

> **All 7 mistakes resolved.** The v2 rewrite
> (`docs/status/2026-08-08_07-22_sse-rewrite-fixes-seven-mistakes.md`) fixed every
> issue: SSE-first architecture, separate routes, canonical `FeedbackType`,
> single-mode design, broadcaster demoted to internal detail, verified
> composability with go-health. Archived — kept as a learning artifact.

---

## What I Did

1. Researched all four libraries (go-health, templ-components, go-datastar, go-sse) via sub-agents
2. Designed a two-mode architecture: HTMX polling (default) + go-datastar SSE (optional, tier 100%)
3. Designed content negotiation via Accept headers on a shared `/health` endpoint
4. Planned a prerequisite task (P1) to add `CachedResponse()` and `RefreshInterval()` to go-health
5. Wrote a 16-task, 9.5h plan with a mermaid execution graph
6. Committed and pushed as v1
7. User asked "why are we not using go-datastar, what else is fucked up?"
8. I re-read all four codebases and found 7 mistakes
9. Rewrote the plan as v2 (12 tasks, 7.5h, SSE-first)

---

## Where the Plan Failed

### 1. Designed a prerequisite task for code that already existed

**What happened:** P1 in the v1 plan says:

> go-health's Probe type currently exposes 10 methods but lacks cache accessors needed by the dashboard

I then designed a 30-minute task (4 subtasks) to add `CachedResponse()` and `RefreshInterval()` to go-health, including tests and AGENTS.md updates.

**The actual reality:** go-health exports **12** methods, not 10. Both `CachedResponse() Response` and `RefreshInterval() time.Duration` already exist with full godocs, tests, and shutdown-overlay logic. They were added in a prior session (commits `608ff6d` and `ded3fd5`).

**Root cause:** My sub-agent reported the Probe's exported methods but I didn't verify against the actual source code. The sub-agent's search was based on a prior conversation's knowledge ("10 exported methods") rather than reading `probe.go` line by line. I trusted a stale count instead of verifying.

**Impact:** 30 minutes of planned wasted work. More critically, it would have sent me into go-health to "add" methods that already exist, potentially creating duplicate definitions or conflicting implementations.

**Suggested fix for planning skill:** Add a verification gate:

> Before planning any prerequisite work on an external package, `grep` the actual source for the exact function signature. If the function exists, delete the prerequisite task. Never plan work based on a count or summary — always verify against source.

---

### 2. Made HTMX polling the default and SSE the "optional" tier

**What happened:** The v1 plan's Pareto breakdown:

> **1% that delivers 51%:** dashboard.Handler() with content negotiation + templ page
> **4% that delivers 64%:** HTMX polling auto-refresh
> **Remaining 20% to reach 100%:** Real-time SSE push + polish

SSE was dead last — tier 100%, marked "Low*" impact, "Optional — only for NOC monitors needing sub-second updates."

**The actual reality:** The user built go-sse and go-datastar specifically for this kind of real-time UI. templ-components already ships `datastar.LiveRegion`, `datastar.SDKScript`, and `datastar.Indicator`. The entire ecosystem is SSE-first. Making polling the default is like buying a Tesla and using the hand crank.

**Root cause:** I imported HTMX mental models from generic web development instead of reading the actual ecosystem. The templ-components `datastar` package exists specifically because Lars chose Datastar over HTMX for real-time. I saw `htmx.PolledRegion` existed and defaulted to it because polling is the "safe, simple" choice in general web dev — without asking "what did the person who built this ecosystem intend?"

**Impact:** The v1 plan would have shipped a dashboard with 2-5 second latency on a health monitoring tool where you want to see failures the instant they happen. It also doubled the architecture: two templates (full page + partial), two endpoints (page + partial), two test matrices. The v2 plan collapses this to one template, one SSE endpoint, one test surface.

**Suggested fix for planning skill:** Add an ecosystem-awareness check:

> Before choosing a real-time strategy, check what the dependency ecosystem is optimized for. If the UI library ships dedicated components for pattern A (`datastar.LiveRegion`) and only generic components for pattern B (`htmx.PolledRegion`), default to pattern A. The ecosystem builder's choice is stronger signal than generic best practices.

---

### 3. Designed content negotiation when separate routes are simpler

**What happened:** The v1 plan put the dashboard on the SAME route as readiness (`/health`) and designed Accept-header-based content negotiation:

> The dashboard lives at the same path as the readiness endpoint (`/health`). The Accept header determines representation.

I then designed q-value parsing, wildcard handling (`*/*`), missing-header defaults, and a 45-minute task (P5) with 5 subtasks just for the negotiation logic.

**The actual reality:** Kubelet and browsers are different consumers hitting different paths. There is zero reason to serve both from one endpoint. Kubelet hits `/readyz` (JSON from go-health, unchanged). Browsers hit `/health` (HTML from dashboard). No Accept header parsing needed. No q-values. No wildcards. No negotiation.

**Root cause:** I overengineered this by applying a REST API design pattern (content negotiation) to a problem that doesn't have multiple consumers on the same route. I was thinking "proper HTTP semantics" instead of "what is the simplest thing that works?" The content negotiation pattern is correct when you have one resource with multiple representations requested by different clients on the same URL. Here, the clients are already separated by route convention (`/healthz` vs `/health`).

**Impact:** 45 minutes of planned complexity for zero user benefit. The negotiation logic would have been a source of bugs (q-value edge cases, proxy header mangling) and test surface area — all to solve a problem that doesn't exist.

**Suggested fix for planning skill:** Add a routing-simplicity check:

> Before adding content negotiation, ask: "Are the consumers already separated by route?" If yes (kubelet uses `/readyz`, browsers use `/health`), use separate routes. Content negotiation is for when multiple consumers MUST share one URL. If they don't have to, don't force them to.

---

### 4. Used deprecated API types throughout

**What happened:** The v1 plan's status mapping table:

| `health.Status` | `feedback.AlertType` |
| --------------- | -------------------- |
| `StatusPass`    | `AlertSuccess`       |

And the SSE code example:

```go
datastar.WithMode(datastar.MergeInner)
```

**The actual reality:**

- `feedback.AlertType` is explicitly deprecated — it's a type alias for `feedback.FeedbackType`. The correct types are `FeedbackSuccess`, `FeedbackWarning`, `FeedbackError`.
- `datastar.WithMode(datastar.MergeInner)` doesn't compile. The correct API is `datastar.WithModeInner()` (sugar constructor) or `datastar.WithMode(datastar.ElementPatchModeInner)`.
- `datastar.ElementsFromTempl` returns `(ElementsPatch, error)`, not a bare `ElementsPatch`. The v1 code example ignores the error.

**Root cause:** I relied on sub-agent summaries of the templ-components and go-datastar APIs instead of reading the actual Go source. The sub-agent reported type names that were close-but-wrong (`AlertType` exists but is deprecated; `MergeInner` looks right but isn't a exported constant). I didn't run `grep` or read the actual `.go` files to verify the exact exported names.

**Impact:** If someone had executed the v1 plan verbatim, the first `templ generate` + `go build` would have failed on deprecated type usage and non-existent constants. Every code example in the plan was wrong.

**Suggested fix for planning skill:** Add an API-verification gate:

> Every type name, function name, and constant in a plan must be verified against actual source code before the plan is committed. Run `grep -r "AlertType" --include="*.go"` in the dependency repo. If the type is deprecated, note it and use the replacement. If a constant doesn't exist, find the correct name. Plans with wrong API names are worse than no plan — they send implementers on a debugging goose chase.

---

### 5. Designed a two-mode architecture that doubled everything

**What happened:** The v1 plan supported both HTMX polling AND go-datastar SSE:

- Two templ files: `view.templ` (full page) + `partial.templ` (HTMX partial)
- Two endpoint types: page handler + partial handler + SSE handler
- Two refresh strategies: polling (default) + SSE (opt-in via `WithSSEPush()`)
- Two test matrices: polling tests + SSE tests
- A `RefreshMode` config option to switch between them

**The actual reality:** One mode suffices. The v2 plan has one `card.templ` component that serves both initial page load (rendered server-side) and real-time updates (sent as Datastar SSE patches via `ElementsFromTempl`). One template, one endpoint, one test surface.

**Root cause:** I defaulted to "support both" because it felt like giving the user flexibility. But flexibility has a cost: every feature must be implemented twice, tested twice, documented twice. For a health dashboard, SSE is strictly better than polling — lower latency, less bandwidth, better DOM efficiency. There is no use case where polling is preferred over SSE in this ecosystem.

**Impact:** The two-mode architecture would have added ~2 hours of implementation, testing, and documentation for a mode nobody would use. It also creates a maintenance burden: every future feature (status history, sparklines, custom grouping) must work in both modes.

**Suggested fix for planning skill:** Add a YAGNI check for architecture modes:

> Before designing a multi-mode architecture, ask: "Is there a concrete user who needs mode B?" If the answer is "some hypothetical user who can't use SSE" — that's not concrete. Design for the users you know about. If a real user shows up needing polling, add it then. One excellent mode beats two mediocre modes.

---

### 6. Presented an internal implementation detail as an architecture decision

**What happened:** The v1 plan's SSE section led with the Broadcaster pattern:

> `realtime.go`: SSE pusher goroutine, broadcaster, `ElementsFromTempl` patch, SSE endpoint handler

The mermaid graph showed the broadcaster as a major architectural component. The technical decisions section discussed the broadcaster's trade-offs.

**The actual reality:** The broadcaster is an internal optimization: one shared ticker reads `probe.CachedResponse()` and fans out to N SSE connections via `sse.Broadcaster[sse.Event]`. Without it, each SSE connection would independently tick and read the cache — redundant but functionally correct. The broadcaster is an implementation detail of `pusher.go`, not an architecture decision the user configures or cares about.

**Root cause:** I elevated an internal mechanism to architectural prominence because it was the most technically interesting part of the design. But interesting-to-implement is not the same as important-to-decide. The user's decision is "SSE or polling?" — not "broadcaster or per-connection ticking?"

**Impact:** The v1 plan made the SSE mode look more complex than it is, which reinforced the (wrong) decision to make it optional. If the broadcaster had been presented as a 3-line internal detail, the SSE mode would have looked as simple as it actually is.

**Suggested fix for planning skill:** Add an architecture-vs-implementation check:

> Before presenting something as an architecture decision, ask: "Does the user need to know about this to use the library?" If no, it's an implementation detail — mention it briefly in the implementation section, not in the architecture diagram or technical decisions. Architecture diagrams should show what the user wires together, not what happens inside the wiring.

---

### 7. Didn't verify composability before planning the integration

**What happened:** The v1 plan's "Prerequisite" section assumed go-health needed changes:

> go-health's Probe type currently exposes 10 methods but lacks cache accessors needed by the dashboard

The entire plan was built on the assumption that the dashboard needed to modify go-health to read cached responses.

**The actual reality:** go-health already exports everything the dashboard needs:

| Dashboard needs        | go-health exports                       | Status         |
| ---------------------- | --------------------------------------- | -------------- |
| Cached health snapshot | `Probe.CachedResponse() Response`       | Already exists |
| Live evaluation        | `Probe.Evaluate(ctx) Response`          | Already exists |
| Refresh cadence        | `Probe.RefreshInterval() time.Duration` | Already exists |
| JSON handlers          | `Probe.ReadinessHandler()` etc.         | Already exists |
| Data model             | `Response`, `Status`, `Check` types     | Already exists |

Zero changes needed. The dashboard is a pure consumer.

**Root cause:** I planned the integration before verifying the integration surface. I assumed the dashboard would need things go-health didn't provide, then designed tasks to add them. If I had started by listing "what does the dashboard need?" and checking each item against go-health's actual exports, I would have discovered immediately that everything exists.

**Impact:** The v1 plan's P1 task was entirely dead weight. More subtly, the framing "go-health needs changes" made the dashboard feel like a heavier integration than it is. The dashboard takes a `*health.Probe` and reads from it. It never mutates probe state. It's a pure consumer — the lightest possible coupling.

**Suggested fix for planning skill:** Add a composability-first check:

> Before planning any prerequisite work on upstream packages, build a dependency matrix: list every capability the new package needs, then check each one against the upstream package's actual exports. If all cells are green, zero prerequisite work exists. Only plan changes for cells that are red. This takes 5 minutes and prevents the most embarrassing planning mistake: designing work that's already done.

---

## What Worked Well

- **Separate repo decision** — Keeping go-health-dashboard in its own module (not inside go-health) was correct. go-health's single-dependency principle is preserved. This decision survived the v1→v2 rewrite unchanged.
- **Sub-agent research** — Using sub-agents to read all four codebases in parallel was effective for the initial research pass. The failure was in verification, not in the research approach itself.
- **Mermaid execution graph** — The dependency visualization caught several ordering issues and made the plan reviewable at a glance. This technique survived into v2.
- **Status mapping design** — Mapping `health.Status` directly to `FeedbackType`/`BadgeType` constants (not via string maps) was correct and survived the rewrite.
- **Honest revision** — When the user asked "what's fucked up?", I re-read everything from scratch and found all 7 mistakes myself rather than defending the v1 plan. The v2 plan is materially better.

---

## Summary: The One Check That Would Have Prevented All Seven

If the planning process had a **verification gate** that ran before committing the plan:

> **Before committing any plan, verify every factual claim against actual source code:**
>
> 1. For every "X doesn't exist" claim: `grep` the source. Does it exist?
> 2. For every type/function/constant name: `grep` the source. Is the name exact? Is it deprecated?
> 3. For every "needs prerequisite work" claim: does the upstream already export what you need?
> 4. For every architecture mode: is there a concrete user who needs the alternative?
> 5. For every technical decision: is this user-facing or internal?
>
> If any check fails, revise the plan before committing.

...all seven mistakes would have been caught before the plan shipped. Every single failure was a verification failure — assuming instead of checking. The research was thorough; the verification was absent.
