# TODO List

> Short-term, actionable, bounded work items, verified against the actual
> code (docs-health HARVEST passes 2026-09-03, 2026-09-04, and 2026-09-10 —
> closed items live in `CHANGELOG.md`, never here). For long-term vision and
> unrefined ideas, see ROADMAP.md.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |

## Next Up

### Correctness & guards (High impact)

| Task                                                                  | Status    | Impact | Effort | Notes                                                                                                                                                                                                                                     |
| --------------------------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/health/introspect`: expose `HealthyGroupCollapseThreshold` + `PersistCollapse` | 🔴 `TODO` | High   | S      | Introspection doc is incomplete relative to `Config` since the 0.7.0 UI work (`options.go` vs `introspect` mapping — report `2026-09-09_23-26` c5)                                                                                         |
| PersistCollapse browser E2E: toggle → SSE patch → state re-applied    | 🔴 `TODO` | High   | S      | Markup/script presence tested; the patch re-apply path has no browser proof (report b5)                                                                                                                                                   |
| Interplay tests: `RetryAlways` × `WithMaxSSEConnections`              | 🔴 `TODO` | High   | M      | Do rejected clients retry forever and hammer a 503-ing server? (report f17)                                                                                                                                                               |
| Interplay test: `RetryAlways` × rate limiting (429 + Retry-After respected by SDK?) | 🔴 `TODO` | High | M | (report f18)                                                                                                                                                                                                                              |
| Interplay test: `RetryAlways` × shutdown drain (drained clients reconnect after restart) | 🔴 `TODO` | High | M | (report f19)                                                                                                                                                                                                                              |
| Verify CV's CSP middleware accepts pill/persistence scripts (read-only, no deploy) | 🔴 `TODO` | High | S | Per-request nonce via extractor path in `~/projects/CV` (report f28)                                                                                                                                                                      |
| `wantsJSON` regression guard for the new UI work                      | 🔴 `TODO` | Low    | S      | Confirm content negotiation untouched; add guard if none (report f44)                                                                                                                                                                     |
| Public-mode filter test: masked names searchable + non-identifying    | 🔴 `TODO` | Medium  | S      | (report f16)                                                                                                                                                                                                                              |
| Coverage: run `nix run .#coverage`, record baseline + floor comparison | 🔴 `TODO` | High   | S      | Never measured during the 0.7.0 session; CI floor is 75% (report b1)                                                                                                                                                                      |

### Features (v0.6.0-cycle leftovers, verified still unimplemented 2026-09-10)

| Task                                                  | Status    | Impact | Effort | Notes                                                                                                                       |
| ----------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------------------------------------------------------------------------- |
| F5: per-check latency series + NDJSON export          | 🔴 `TODO` | Low    | M      | Per-check latency histogram labels in `metrics.go` + `?format=ndjson` on the export endpoint (`trend.go`)                    |
| F8: 20-source aggregate load test                     | 🔴 `TODO` | Low    | M      | F7 dependency is satisfied (example `DEMO_AGGREGATE` exists); record numbers in `docs/research/`                              |
| F11: `WithGrouping(BySource)` per-service cards       | 🔴 `TODO` | Med    | L      | View-model grouping for aggregate pages (`status.go` `groupChecks`)                                                           |
| Mobile: design decision + true row stacking (not just overflow containment) | 🔴 `TODO` | Medium | M | Plan T9's original intent; visual design decision first (report b2)                                                          |
| Example: `DEMO_` toggles for collapse threshold, persistence, embedded-SDK filter | 🔴 `TODO` | Medium | S | Dogfoods the 0.7.0 options (report f12); folds in config.yaml idea (f45)                                                      |
| Storage-sync listener for collapse persistence (cross-tab) | 🔴 `TODO` | Low   | S      | `view.templ` `collapsePersistence` script (report f30)                                                                        |
| `aria-live` announcements for filter result counts    | 🔴 `TODO` | Low    | M      | Screen-reader UX for narrowing (report f40)                                                                                  |
| Pill polish: equal-width states (no layout shift), title attr with last transition time | 🔴 `TODO` | Low | S   | `view.templ` `connectionPill` (report f23)                                                                                    |
| `shortDisplayName` decision for generics (`pkg.Type[T]`) + real-world example | 🔴 `TODO` | Low | S    | `status.go` (report f29)                                                                                                     |
| Patch-safe copy affordance for raw check keys         | 🔵 `BLOCKED` | Medium | M  | Product decision pending (report g Q3)                                                                                       |

### Quality & polish

| Task                                                                    | Status    | Impact | Effort | Notes                                                                                                |
| ----------------------------------------------------------------------- | --------- | ------ | ------ | ---------------------------------------------------------------------------------------------------- |
| Axe re-run on the filtered DOM state                                    | 🔴 `TODO` | High   | S      | Filter test asserts narrowing; axe not re-run post-filter (plan C5 leftover, report b4)               |
| Aggregate-mode browser test for the new UI (2+ merged probes)           | 🔴 `TODO` | Medium  | M      | Only the old aggregate CSP test covers aggregate (report c7)                                           |
| Keyboard pass: jump link + export/trend/metrics links reachable, labeled | 🔴 `TODO` | Medium | S      | Extend `TestBrowser_KeyboardNewControls` (report f27)                                                 |
| Real fuzz session: `-fuzztime 60s` on `FuzzShortDisplayName` + friends  | 🔴 `TODO` | Medium  | S      | CI exercises seed corpora only (report f26)                                                            |
| Re-baseline `BenchmarkHandler_HTMLRendering` after render growth        | 🔴 `TODO` | Medium  | S      | Filter expressions + pill + persistence script landed since last baseline (report f21)                 |
| Golden-file render snapshot test for `view.templ`                       | 🔴 `TODO` | Medium  | M      | Makes accidental markup churn reviewable (report f49)                                                  |
| Zero-count edge: groups never render empty (property test)              | 🔴 `TODO` | Low    | S      | Badge pluralization helper unreachable at 0 (report f33)                                               |
| Dark-mode contrast pass on pill/badge/link colors                       | 🔴 `TODO` | Low    | S      | Local browser pass; live-page pass rides the CV rollout (report f34)                                   |
| Verify `prefers-reduced-motion` for the chevron rotation end-to-end     | 🔴 `TODO` | Low    | S      | Upstream handles it; verify in our render (report f42)                                                 |
| Consolidate duplicated browser-test boilerplate (static handlers, chrome lifecycle) | 🔴 `TODO` | Medium | M | Helpers in `browser_test.go` (report f20)                                                              |
| Extract the three inline page scripts (pill, persistence, theme) into one nonce'd bootstrap | 🔴 `TODO` | Low | M | Cuts duplicate script tags (report f22); CSP tests must stay green                                     |
| `view.templ` split for pill/persistence if the file keeps growing       | 🔴 `TODO` | Low    | M      | See ADR-0001 file-split rationale (report f25) — conditional on growth                                 |
| Degraded screenshot fixture (failing/warning services) for docs         | 🔴 `TODO` | Medium  | S      | Healthy light+dark captured; degraded missing (report b6)                                              |
| Screenshot perms: normalize 0600 vs 0644 in the capture helper          | 🔴 `TODO` | Low    | S      | `screenshot*_test.go` (report f41)                                                                     |
| `scripts/pre-push-checks.sh`: encode drift-guard formulas + version check | 🔴 `TODO` | Medium | S     | Session-closing local gate (report f48)                                                                |
| CI: confirm the browser job runs on PRs, not just pushes               | 🔴 `TODO` | Medium  | S      | `.github/workflows/ci.yml` triggers (report f47)                                                       |

### Documentation & upstream

| Task                                                                         | Status    | Impact | Effort | Notes                                                                                                     |
| ---------------------------------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------------------------------------------------------- |
| ~~File the upstream go-health `Check.Since`/`Duration` issue~~            | 🟢 `DONE` | Medium  | S      | Filed 2026-09-10 as go-health#2 after re-verifying claims against v0.1.3; also blocks F5's per-check latency labels until it ships |
| `docs/DOMAIN_LANGUAGE.md`: dashboard terms now load-bearing in UI copy       | 🔴 `TODO` | Low     | M      | check, group, probe, source/check, collapsing (report f31)                                                 |
| Annotate the UI/UX Pareto plan: phases A-D executed, stale v1.11.0/v0.4.0 pin refs superseded | 🔴 `TODO` | Low | S | `docs/planning/2026-09-09_19-21_dashboard-ui-ux-pareto.md` (report f32, f39)                              |
| CHANGELOG `[Unreleased]` convention note (ship-or-fold discipline)           | 🔴 `TODO` | Low     | S      | So the next bumper doesn't rediscover the layout (report f43)                                              |
| Decide + document `LastUpdatedTime`/age in JSON responses (currently HTML-only by design) | 🔴 `TODO` | Low | S | `handlers.go` `serveJSON` (report f50)                                                                     |
| Harvest CV review items H1-H7: mark what the 0.7.0 release addresses         | 🔴 `TODO` | Medium  | S      | Cross-ref `~/projects/CV` review docs (report f46)                                                         |

## Blocked (needs user decision)

| Task                                                     | Status       | Why blocked                                                                                                                             | Evidence                                  |
| -------------------------------------------------------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| Tag `v0.7.0` + push `--follow-tags` (greens version-guard) | 🔵 `BLOCKED` | Release decision + publish act; also decides whether `[Unreleased]` route-ergonomics ships folded or separately (report g Q1)              | CI version-guard red; no `v0.7.0` tag yet |
| CV-side adoption: bump go.mod, deploy, verify live page  | 🔵 `BLOCKED` | Deploy pipeline + rollout order are the user's call (report g Q2). Verified 2026-09-10 (read-only): CV serves a CSP-safe mini-client, not the Datastar SDK — a safe bump needs `dashboard.WithNoDatastarRuntime()` (added in `[Unreleased]`) or the filter/pill would render dead | CV `internal/di/health_dashboard.go` pins v0.6.1 |
| Pin-guard keep/drop sign-off                             | 🔵 `BLOCKED` | Guard rewritten to v1.16.0 pins 2026-09-10 per the keep-the-guard decision; sign-off still pending (report b7)                            | `scripts/check-ui-pins.sh` header          |
| Build-tag gating for SSE                                 | 🔵 `BLOCKED` | Consumers who only want HTML shouldn't need GOEXPERIMENT=jsonv2. Requires decision: accept, fork go-sse, or gate.                        | `ROADMAP.md` Open Questions                |
| Fingerprint format stability                             | 🔵 `BLOCKED` | Length-prefix fix changed fingerprint values; documented as accepted in CHANGELOG pending a versioning decision.                         | `ROADMAP.md` Open Questions                |

Closed in this harvest (verified against code/CHANGELOG 2026-09-10): U3 pin
lift (#7 shipped upstream in v1.13.2, ceremony `e8a7033`), CI pin-guard G1
(exists, now pins v1.16.0), upstream LiveRegion nonce-guard PR (templ-
components#8 merged 2026-09-05), upstream StatCard `<dl>` PR (templ-
components#6 closed completed — fix is in v1.16.0; axe tolerance retired the
same day), and report f11 (comment on #6 — obsolete, issue closed). The
v1.16.0 sweep ceremony itself (browser suite green, guard re-pinned, axe
tolerance dropped) landed in `[Unreleased]`. Everything else from the v0.3.x
brainstorms was closed with a reason in the annotated reports under
`docs/status/` (fully-executed reports move to `archived/`), or lives in
ROADMAP.md as raw ideas. Known-broken-commit SHAs for `git bisect skip`: see
AGENTS.md and `docs/status/archived/2026-09-04_19-15_bisectability-audit.md`.
