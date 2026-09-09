# Status Report — CI-Green & Backlog Pareto: Full Execution Session

|                         |                                                                                                                                                                    |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Date**                | 2026-09-10 01:15 (CEST)                                                                                                                                            |
| **Session scope**       | Full execution of the reconciled TODO list + the 50-item list from `2026-09-09_23-26_ui-ux-pareto-execution-status.md` (f), per the Pareto plan `docs/planning/2026-09-10_00-26_ci-green-and-backlog-pareto.html` |
| **Repo state**          | `master`, working tree with this session's changes (auto-daemon commits interleaved)                                                                               |
| **Gates at close**      | Unit ✅ · `-race` ✅ · lint 0 ✅ · browser suite ✅ (incl. axe with ZERO tolerances) · vulncheck ✅ · `nix flake check` ✅ · pin guard ✅ · pre-push script: only the version-vs-tag check red (user decision) |
| **Coverage**            | Library 79.8% (CI floor 78%, scoped `go test .`); `nix run .#coverage` now matches CI's scope                                                                      |

---

## a) FULLY DONE

1. **v1.16.0 pin ceremony (the session's 1% → 51%).** An unguarded
   templ-components sweep to v1.16.0 (fifth occurrence) had landed after the
   last report; the CI pin guard caught it, Build/Test jobs red. Ceremony:
   full browser suite green on v1.16.0 + go-datastar v0.5.0, guard re-pinned
   (now including `templ-components/utils`), AGENTS/CHANGELOG/FEATURES
   reconciled. Upstream #6 (StatCard `<dl>`) is FIXED in v1.16.0 — the axe
   `definition-list`/`dlitem` tolerance is **retired**;
   `TestBrowser_Accessibility` now fails on any serious/critical violation
   again.
2. **WithPersistCollapse was broken in production — found by its first E2E
   test, root-caused, and fixed.** Three stacked bugs: (a) the SDK dispatches
   `datastar-patch-*` BEFORE merging the patch, so event-driven re-apply
   always targeted the replaced-away node; (b) Datastar's inner-mode merge
   syncs attributes — removing `open` fired a `toggle` that overwrote the
   stored choice with the server default; (c) the fix's MutationObserver
   initially looped forever because an identical `setAttribute` still queues
   a mutation record (starved the renderer; diagnosed from a wedged CDP
   context). Fix: storage keys off summary clicks (user intent, pre-toggle
   state read synchronously), re-apply via a scoped MutationObserver with
   guarded sets, plus a `storage` listener for cross-tab consistency.
   Proven end-to-end by `TestBrowser_CollapsePersistInteract` (toggle →
   patches → reload all hold state).
3. **F11 `WithGrouping(GroupBySource)`**: one card per aggregate
   `source/check` prefix, worst-of status per card, Services fallback for
   plain keys; severity mode stays default. Collapse policy and persistence
   deliberately skip source mode (no single healthy group); public mode
   masks source titles (topology is identifying); unknown modes fall back
   to severity.
4. **F5 NDJSON export**: `?format=ndjson` — one sample object per line,
   `application/x-ndjson`. The per-check latency half is blocked upstream:
   go-health `Check` carries no duration (that gap is now filed).
5. **Upstream filing**: [go-health#2](https://github.com/LarsArtmann/go-health/issues/2)
   (per-check `Since`/`Duration`), claims re-verified against the v0.1.3
   module cache first. Draft annotated as filed.
6. **`WithNoDatastarRuntime()`** — born from the CV read-only verification:
   CV serves a CSP-safe mini-client, not the SDK, so a straight 0.7.x bump
   would render a dead filter box and a pill stuck on "Live". The option
   omits SDK-dependent UI; everything server-rendered is unaffected.
7. **Introspection completeness**: `healthy_group_collapse_threshold`,
   `persist_collapse`, `hide_stat_cards`, `embedded_datastar_sdk`,
   `push_on_change_ttl`, `timeline_max_age`, plus the `datastar_js` and
   `introspect` routes — the document no longer lies by omission.
8. **Interplay: RetryAlways × WithMaxSSEConnections** — new browser test
   proves a second tab rides out 503s and connects the moment the slot
   frees; the 429 path shares the SDK's non-200 retry branch (verified in
   the pinned bundle) and drain semantics are covered by composition
   (existing tests), all documented in the test header.
9. **Browser/quality batch**: axe re-run on the FILTERED DOM (plan C5
   leftover) inside the filter test; aggregate new-UI browser test
   (namespaced keys, source-prefix filtering, merged count);
   keyboard-links test (jump anchor + 4 header links reachable, labelled,
   Enter-activated); public-mode filter test (masked names stay
   searchable, no raw leak — the leak half was already swept).
10. **Gates batch**: F8 load harness (`LOADTEST=1`, 20 sources × 3 checks ×
    20 SSE clients: 1000 events/5s zero-loss, first event 8ms, scrape p95
    5.5ms — recorded in `docs/research/2026-09-10_aggregate-load-test.md`);
    fuzz session (ShortDisplayName + FingerprintChecks, ~9.7M execs each,
    no crashers); benchmark re-baseline (FullHTML ~0.2ms/71KB/574 allocs,
    recorded in `docs/research/2026-09-10_benchmarks.md`); golden render
    files (severity + source fixtures, `-golden-update` flow,
    time-bomb-proofed by zeroing the age stamp); `scripts/pre-push-checks.sh`
    (encodes the FEATURES, version-guard, and pin-guard formulas — it caught
    my own FEATURES drift twice); CI browser job confirmed to run on PRs;
    degraded screenshot fixture + README section + perms normalization
    (0600→0644); browser-test boilerplate consolidated into
    `browserStaticHandlers`/shared handlers (12 duplicated blocks removed).
11. **UX batch**: equal-width pill states (no layout shift);
    `storage`-event listener (cross-tab collapse sync); no-match hint is a
    `role="status"` region; `shortDisplayName` generics decision
    (`store.Store[string]` passes through — stripping would lie) pinned in
    the truth table; zero-count property test (no grouping mode ever emits
    an empty group).
12. **Docs batch**: DOMAIN_LANGUAGE presentation-terms section (group,
    source/check, short name, collapse, persistence, filter, pill,
    jump-to-problems, sample/transition) + route table refresh; the
    2026-09-09 Pareto plan annotated (phases A–D executed, v1.11.0/v0.4.0
    pins marked superseded); CHANGELOG `[Unreleased]` convention note;
    JSON-age parity decision recorded in AGENTS.md (HTML-only by design);
    example `DEMO_COLLAPSE`/`DEMO_PERSIST`/`DEMO_EMBEDDED_SDK`/`DEMO_GROUPING`
    toggles; flake `coverage` app scoped like CI (the example's 0% was
    diluting the number); FEATURES rows for everything new.

## b) PARTIALLY DONE

1. **Full aria-live filter counts** — the no-match hint announces (role
   status), but per-keystroke match counts need a script; parked unless a
   screen-reader user asks.
2. **Dark-mode contrast pass** — harness-verifiable parts covered by axe;
   the live-page manual pass intentionally rides the CV rollout.

## c) NOT STARTED (deliberately deferred, rationale in TODO_LIST)

1. Mobile row stacking (needs the visual design decision first).
2. Single nonce'd bootstrap script (three small tested scripts beat one
   merged mid-session refactor; revisit with a dedicated CSP-suite run).
3. `view.templ` split (conditional on growth; ~500 lines now).

## d) TOTALLY FUCKED UP

1. **I wrote a placeholder-quality first draft of the load test**
   (`sources切片` — a Chinese variable name — plus a dead `wg2`/`connectDone`
   block). Never compiled, rewritten before any test run, but it is the
   exact anti-pattern the last report flagged in itself. Caught by reading
   my own diff.
2. **Three lint-fix rounds on `nix fmt` interactions**: nolint directives
   drifted when golines wrapped `make()` calls (the directive ended up
   anchored to the wrong line), and I twice "fixed" lint findings by
   patterns that no longer matched after a daemon reformat. Cost: ~4
   verification cycles. Lesson re-learned: after ANY `nix fmt`, re-grep the
   pattern before asserting on it.
3. **A golden test time bomb, caught before it bit**: the first golden
   fixture pinned `LastUpdatedTime` to a fixed future timestamp — formatAge
   would have flipped buckets at 12:00 UTC and broken the golden
   mid-afternoon. Zeroed instead; the guard omits the age span.
4. **My own FEATURES drift, twice**: added test files without recounting;
   the pre-push script I was building caught me both times (233→236). The
   tool works.

## e) WHAT WE SHOULD IMPROVE

1. The pre-push script should run in the auto-daemon's pre-commit too, not
   just before pushes.
2. The pin ceremony now has a same-change rule in the guard header and
   AGENTS.md — the next sweep still has to obey it; watch CI.
3. Load harness numbers are machine-specific; record host context (done in
   the research doc) and re-run after pusher/broadcaster changes.

## f) Decisions that remain yours

| #  | Decision                                                                                       | Where it blocks                                  |
| -- | ---------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| 1  | Tag `v0.7.0` vs cut `v0.8.0` — `[Unreleased]` is now substantial (introspection, NDJSON, grouping, ceremony, the PersistCollapse fix). Folding it into 0.7.0 ships a much bigger tag than its CHANGELOG section describes; 0.8.0 keeps both honest. Then push `--follow-tags` (greens version-guard). | CI version-guard job; release checklist          |
| 2  | CV rollout order (report g Q2). The bump is now SAFE with `WithNoDatastarRuntime()`; deploy pipeline is yours. | `~/projects/CV` go.mod pin at v0.6.1             |
| 3  | Copy affordance for raw keys (report g Q3) — still open as a product call.                      | TODO_LIST blocked                                |
| 4  | Pin-guard keep sign-off (guard retained by decision; deviation documented).                    | `scripts/check-ui-pins.sh` header                |

## CV findings mapping (report f46, from CV's 2026-09-09 architecture review)

- **F1 "health-signal starvation"** — directly addressed by the 0.7.x UI
  (collapse, short names, filter, jump link) once CV bumps.
- **F4 "handlers/config dilute the service table"** — addressed by short
  display names (same bump).
- F2/F3/F5/F6/F7/F8/F9 are CV-internal (dead Groq check, system unification,
  DB-less degradation, swallowed errors, hardcoded version, scope shape) —
  not dashboard work.
