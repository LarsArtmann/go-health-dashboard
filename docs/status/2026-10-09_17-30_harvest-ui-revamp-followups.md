# HARVEST Ledger — UI-Revamp Status Report → TODO_LIST / ROADMAP

**Date:** 2026-10-09 ~17:30 CEST
**Source:** `docs/status/2026-10-09_16-20_ui-ux-revamp-v1210-adoption-status.md` (§e/§f/§g; no pre-existing `done at` markers)
**Mode:** docs-health HARVEST (report itself left unannotated per HARVEST rule 6)
**Gates at harvest time:** `nix run .#build`, full unit suite, `nix run .#lint` (0 issues), `nix flake check` (all checks passed), coverage 84.7%, browser suite green (16:10, pre-harvest tree; harvest touched only docs + `evidence.go` copy, unit-covered).

## Ledger

Dispositions: `new row` (TODO_LIST harvest table), `existing row`, `merged`, `declined`, `done in code`, `routed to ROADMAP`.

| Source (§f# unless noted) | Disposition | Destination / reason |
| ------------------------- | ----------- | -------------------- |
| f1 Cut v0.11.0 | new row (BLOCKED) | TODO_LIST harvest table; folded in f24 (auto-push anomaly check), f44 (fuzz confirmation), f46 (release thumbnails) |
| f2 HARVEST | done in code | This pass |
| f3 Golden fixture both-cards grid | done in code | `TestGoldenRender_TrendTimeline` + `testdata/golden/trendtimeline.html` (this session) |
| f4 flake check + coverage + vulncheck | done in code | `nix flake check` all-passed; coverage 84.7%; govulncheck: 9 stdlib + 4 uncalled-import vulns (toolchain-level, Go 1.27.1 vs current DB — a toolchain-bump decision, not code; re-run before release) |
| f5 Upstream StatCard green tone | new row (BLOCKED) | TODO_LIST; filing is remote-action-gated, local `verify-before-filing` can start anytime |
| f6 Degraded screenshot verification | done in code | Viewed + verified (full-width lone trend card confirms the D1 fix); FOUND + FIXED the evidence-strip grammar bugs it exposed (`evidence.go` "has ever" → "have ever", "1 green rows are" → "1 green row is"; tests + `zeroproven.html` golden updated; all three README screenshots recaptured) |
| f7 AGENTS architecture bullet | done in code | AGENTS.md view.templ bullet now names trend/timeline grid + PageHeader/stat cards outside the patch region |
| f8 FEATURES line-ref sweep | done in code | All 29 stale `file:NN` refs converted to `file — Symbol()` anchors (lines had rotted; several cited the wrong file — With* live in options.go, SubscriberCount in handlers.go) |
| f9 README prose audit | new row | TODO_LIST |
| f10 Harness pill `[hidden]` | new row | TODO_LIST |
| f11 SSE patch payload measurement | new row | TODO_LIST |
| f12 Benchmark delta | new row | TODO_LIST |
| f13 ADR-0003 tone system | merged | TODO_LIST row (with f32) |
| f14 Timeline helper unit tests | merged | TODO_LIST row (with f37) |
| f15 ListNote timeline cap | new row | TODO_LIST |
| f16 Touch tooltips story | new row | TODO_LIST (with f33 folded) |
| f17 GroupBySource screenshot | new row | TODO_LIST |
| f18 Latency tone policy (§g Q1) | routed to ROADMAP | Open Questions — questions are not tasks |
| f19 Age-stamp policy (§g Q2) | routed to ROADMAP | Open Questions (Q2) |
| f20 BuildFlow divergent-sweep failure mode | routed to ROADMAP | Theme 6 (fleet-level) |
| f21 Browser half/half grid test | routed to ROADMAP | Theme 6 |
| f22 Axe degraded fixture | merged | TODO_LIST row (with f38) |
| f23 Endpoint-link keyboard reach | new row | TODO_LIST |
| f24 Auto-push daemon anomaly | merged | Release row checklist (f1) |
| f25 SSEErrorHandling opt-in | routed to ROADMAP | Theme 6 |
| f26 CopyButton re-vet | existing row | TODO_LIST Blocked "Copy affordance for raw check keys" — v1.21 nonce/CSP re-vet note appended |
| f27 Stat icons outside patch region (doc) | done in code | Covered by the AGENTS.md view.templ bullet rewrite (f7) |
| f28 Group count badge tone | routed to ROADMAP | Theme 6 |
| f29 Sticky table header | routed to ROADMAP | Theme 6 |
| f30 Print styles verification | routed to ROADMAP | Theme 6 |
| f31 Dark aggregate screenshot | routed to ROADMAP | Theme 6 |
| f32 Tone rule in dark-mode checklist | merged | TODO_LIST ADR row (with f13) |
| f33 Tooltip wording redundancy review | merged | TODO_LIST touch-story row (with f16) |
| f34 PolledRegion fallback | routed to ROADMAP | Theme 6 |
| f35 ROADMAP now-done sweep | done in code | This pass: templ-components follow-ups bullet updated (PageHeader/StatCard icons adopted), InstanceID decision annotated as overtaken (`c57869c`), theme 6 added |
| f36 Hub e2e vs new UI | new row | TODO_LIST |
| f37 alertSummary grouped cases | merged | TODO_LIST row (with f14) |
| f38 Axe dark re-run evidence | merged | TODO_LIST row (with f22) |
| f39 Icons adoption pattern docs | routed to ROADMAP | Theme 6 |
| f40 EmptyState filter hint | routed to ROADMAP | Theme 6 |
| f41 Badge transition animation | routed to ROADMAP | Theme 6 |
| f42 Capture wrapper script | new row | TODO_LIST (the three-capture env-var footgun hit live this session) |
| f43 /health JSON byte-identity vs v0.10.2 | done in code | Verified: zero diffs in handlers.go/webhook.go/metrics.go/trend.go since v0.10.2; dashboard.go nolint-comment only; go-health v0.4.1→v0.5.0 `Response`/`Check` wire structs + tags byte-identical (v0.5.0 adds a version endpoint + critical-name validation, no wire-shape change); status.go changes are view-model-only |
| f44 Fuzz-target confirmation | merged | Release row checklist (f1) |
| f45 Collapse-persistence exclusion docs | routed to ROADMAP | Theme 6 (initially missed in routing, caught in ledger review) |
| f46 Release-notes thumbnails | merged | Release row checklist (f1) |
| f47 Upstream `[hidden]` resilience ask | routed to ROADMAP | Theme 6 (after TODO f10) |
| f48 RefreshInterval in latency tooltip | routed to ROADMAP | Theme 6 |
| f49 Archived UI-plan sweep | routed to ROADMAP | Theme 6 |
| f50 brutal-self-review next session | declined | Process reminder, not project work; TODO_LIST stays work-only |
| §e1–e3 process improvements | declined | Design-review degenerate-state checklist, invariant-grep-first, per-batch browser runs are behavioral; AGENTS.md is over its 377-line linter budget (existing TODO row) — adding process lines works against it |
| §e4 fleet BuildFlow fix | declined (here) | Already tracked fleet-side (BuildFlow repo); not per-repo work |
| §e5 golden coverage mirrors layout | done in code | = f3 |
| §e6 harness pill rendering | new row | = f10 |
| §e7 patch payload | new row | = f11 |
| §g Q1/Q2/Q3 | routed to ROADMAP | Open Questions; surfaced to the user via the session question prompt |
