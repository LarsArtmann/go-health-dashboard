# UI/UX Batch — Status Report (chips blocked on SDK event grammar)

**Date:** 2026-10-25 07:39 CET
**Mode:** STOP-AND-REPORT (all further work on the blocked item paused)
**Session goal:** "How can we improve the UI/UX" — break down, execute, verify one step at a time.

## Gates at stop time

- Full unit suite: green in isolation for every new/changed unit test
  (`TestRender_EvidenceDisclosureTouchAndKeyboard`,
  `TestBadgeForStatus_UnprovenMarkerAndShape`, `TestGroupCountTone`,
  `TestLatencyTooltipText`, `TestApplyLatencyTone` (extended),
  `TestRowFilterExpr`, `TestFilterEmptyExpr`, `TestStatusChipExprs`,
  goldens regenerated and green).
- Full `go test .` is currently RED because of ONE failing browser test
  (`TestBrowser_StatusFilterChips`) — see §TOTALLY FUCKED UP.
- `TestBrowser_EvidenceDisclosureFocus` (new): NOT yet verified — the run
  that would have covered it was filtered/aborted by the chips failure.
- Lint / flake check / full browser suite: NOT run this session (blocked
  behind the red chips test and the report stop).
- Auto-daemon has committed all working-tree state (`b113d2a` et al; tree
  clean at stop).

---

## A) Fully done and verified

~~1. **Pre-existing test-build breakage FIXED (root cause)** — daemon commit~~ verified this session (goldens + unit green; chromedp pins restored)
   `93d13fd` (Oct 15) bumped chromedp v0.16.0 → v0.21.0 in go.mod WITHOUT
   migrating browser_test.go/screenshot_test.go (~100 call sites; v0.21 is
   a generics rework: `Run[T]`, `Do`, `remote.NewAllocator`,
   `ListenTarget` gone). Restored chromedp v0.16.0 + its matching cdproto
   pin. The whole package's test build was broken at HEAD before this —
   nothing was verifiable until fixed.
~~2. **P1 — Touch/keyboard-accessible evidence tooltips (HIGH-impact TODO row)**~~ done this session — landed, browser-pinned later the same day (TestBrowser_EvidenceDisclosureFocus)
   — native `title` (invisible on touch) replaced by CSS-only disclosures:
   focusable span + `role="tooltip"` visual toggled by
   `group-hover`/`group-focus-within` + `sr-only` copy for screen readers;
   unproven greens now carry a **dashed provisional outline**
   (`outline-dashed`, glance-visible without any hover). CSP-clean (class
   markup only), SSE-patch-pure (no scripts), goldens pinned. The truth
   strip's tooltip got the same treatment via an info glyph
   (`icons.Information`).
~~3. **P2 — Severity-toned group count badges** — warn groups get the amber~~ done this session (groupCountTone, goldens green)
   count chip, fail groups the red one (`groupCountTone`, mirroring the
   upstream badge palette); pass/off stay neutral gray. "Only problems
   carry color" rule kept.
~~4. **P3 — Filter no-match hint upgraded to `display.EmptyState`** (Search~~ done this session (filterEmptyAttrs, polarity bug fixed below)
   icon, title, description; keeps `data-filter-empty` +
   `data-class:hidden` via `filterEmptyAttrs`).
~~5. **P4 — Latency tooltip now discloses the budget** —~~ done this session (latencyTooltipText)
   `viewModel.RefreshIntervalText` stamped in `applyLatencyTone`;
   tooltip reads "...the probe refreshes every 5s, and the value turns
   amber past half that budget" (roadmap item from theme 6).
~~6. **Pre-existing BUG FIXED (unit-pinned): inverted filter-empty polarity**~~ done this session (TestFilterEmptyExpr pins both directions)
   — old `filterEmptyExpr` hid the hint exactly when NOTHING matched and
   revealed it after clearing a query; rewritten as undefined-safe
   miss-negation (new `TestFilterEmptyExpr` pins both directions).
~~7. **Test-harness correctness** — `aggregate_sse_test` read window 4KB→16KB~~ done this session (16KB read window)
   (evidence disclosures measurably add ~600 B/pass row + ~470 B strip per
   patch; measurement recorded in the test comment — partial answer to
   TODO f11).
~~8. **Play-CDN v3 compatibility** — tooltip width class `max-w-64` (v4-only)~~ done this session (max-w-[16rem])
   swapped for `max-w-[16rem]`.

## B) Partially done

~~1. **P5 — Status filter chips (All / Problems / Pass)**~~ fixed same day — colon grammar data-on:click; TestBrowser_StatusFilterChips green (12-26 report a1)
   - DONE: Go expression builders (`rowFilterExpr`, chip class/pressed
     exprs, `statusFilterSignal`), templ markup (segmented control next to
     the filter input), unit tests, goldens.
   - BLOCKED: real interaction dead in the pinned Datastar v1.0.3 bundle —
     see §D1.

## C) Not started (planned, descoped by the stop)

~~1. P6 — docs closure for keyboard-links TODO row (VERIFIED already:~~ closed 2026-10-25 — TestBrowser_KeyboardLinks verified; TODO row closed (12-26 a3)
   `TestBrowser_KeyboardLinks` at browser_test.go:1823 asserts all four
   endpoint glyph links + accessible names + jump-link Enter navigation —
   the TODO row just needs closing with this evidence).
~~2. P7 — sticky table header: analysis done, decision = SKIP (upstream~~ SKIP recorded in ROADMAP theme 6 (overflow wrapper), 2026-10-25
   `StickyHeader` prop only works in height-constrained scroll containers;
   our tables live in `overflow-x-auto` wrappers on a page-level scroller —
   thead would never stick). Needs the TODO/ROADMAP documentation entry.
~~3. P8 — docs batch: CHANGELOG + FEATURES entries for P1-P4, TODO_LIST~~ done 2026-10-25 — CHANGELOG/FEATURES/AGENTS/gotchas/README/screenshots via the 10:45+12:26+12:27 sessions
   harvest (touch-tooltips done, f23 evidence, f11 measurement note),
   AGENTS.md templ-components pin correction (says v1.21.0, go.mod is
   v1.23.0), chromedp incident → `docs/gotchas.md`, README prose audit vs
   new UI (f9), harness `[hidden]` CSS fix (f10), GroupBySource screenshot
   (f17), screenshot recapture for the new tooltips/chips UI.
~~4. Final gates: `nix run .#lint`, `nix flake check`, full browser suite~~ green — pre-push-checks 8/8 at 5ceb5f2; CI 7/7 on e8eeefa (14-19 a6/a10)
   (serialized), screenshot capture verification.

## D) Totally fucked up (honest)

1. **The chips do not work against the real SDK — shipped dead markup.**
   Empirical findings (headless Chrome + embedded Datastar v1.0.3 bundle,
   served self-hosted):
   - `data-on-click="…"` is silently IGNORED (the v1 grammar splits plugin
     keys on `:`; "on-click" resolves to no plugin — consistent with the
     known `data-bind`/`data-class:hidden` colon-keyed gotcha).
   - `data-on:click="…"` (correct v1 grammar) throws
     `SyntaxError: Unexpected token ':'` and the handler never fires —
     even a pure side-effect expression
     (`window.__sideEffect='clicked'`) never ran, and sibling
     `data-text` attributes on the same probe elements also never
     evaluated (one failing attribute seems to abort the merge batch).
   - Result: `TestBrowser_StatusFilterChips` fails (rows never hide;
     `aria-pressed` never flips; the class-object/attr wiring itself is
     untested because the signal never changes).
   - NOT reverted (markup + unit layer are sound; the interaction seam is
     the open question). The failure is deterministic, not flaky.
2. **One new browser test is UNVERIFIED** (`TestBrowser_EvidenceDisclosureFocus`)
   — its only run was swallowed by the chips failure's output filtering.
3. **A stale templ-LSP diagnostic** ("view.templ:168 missing ','") has
   shown since session start; `templ generate` + build + tests are green,
   so it is an LSP false positive, but it masks real diagnostics in that
   file and should be dismissed/re-checked on the next LSP restart.

## E) What I would improve about my own work here

1. I wrote the chips interaction against the SDK from memory of upstream
   Datastar grammar instead of probing the pinned bundle FIRST — the
   colon-keyed gotcha was already documented in AGENTS.md; the empiricalical
   probe harness should have been step zero of P5.
2. I claimed "unit green" mid-session while `go test .` (which includes
   browser tests in the devShell) had not been re-run after the chips
   markup landed — the red chips test was discoverable one cycle earlier.
3. Debug cycles fought the templ LSP/daemon file-reload race (several
   "file modified since read" guard rejections); canonical order
   (generate → fmt → test in ONE devShell call) should be the default.

## F) Next up to 50 things (order = value density)

**Unblock P5 (pick ONE path):**

~~1. Read the pinned bundle's `on` plugin registration semantics in~~ done 2026-10-25 — colon grammar confirmed against the bundle; chips fixed (12-26 a1)
   go-datastar source (not minified) — `~/projects/go-datastar` — and find
   the sanctioned event attribute grammar + assignment expression form.
~~2. Alternative A: server-side chips (`?status=problems` handled in~~ NOT-DO — path 1 (grammar fix) won; server-side chips moot
   `buildData`; links, zero JS, crawler-friendly) — but must solve
   patch-vs-query persistence (patches re-render without the query).
~~3. Alternative B: wire chips through `pageBootstrap` vanilla JS IF the~~ NOT-DO — path 1 (grammar fix) won; vanilla-JS bridge moot
   bundle exposes any signal-write path (probably absent → likely dead end).
~~4. Alternative C: drop chips, keep the (now-fixed) hint + text filter only~~ NOT-DO — path 1 (grammar fix) won; chips kept and green
   — smallest revert; chips become a ROADMAP idea again.
~~5. Whatever path: make `TestBrowser_StatusFilterChips` green or delete it~~ done — TestBrowser_StatusFilterChips green (12-26 a1)
   with the feature.

**Verification debt from this session:**
~~6. Run `TestBrowser_EvidenceDisclosureFocus` (unverified).~~ done — PASS 1.76s (12-26 a1)
~~7. `nix run .#lint` on the whole change set.~~ done — 0 issues at the shipped tip (14-19 a6)
~~8. `nix flake check`.~~ done — all checks passed (14-19 a6)
~~9. Full serialized browser suite (`nix develop -c go test -run TestBrowser`).~~ done — 19/19 browser suite green (12-27 a; 14-19 a6)
~~10. Re-capture README screenshots (new tooltips/chips markup) —~~ done 2026-10-25 10:12 (10-45); grouped capture 48a8c11
`SCREENSHOT_OUTPUT=docs/screenshot.png` etc (f42 wrapper would help).
~~11. Re-run `BenchmarkHandler_HTMLRendering`, record delta vs v0.10.2 (f12).~~ done — 44.0-54.9µs/op vs 40.4 baseline recorded (research notes 2026-10-25)
~~12. Measure SSE patch payload old-vs-new precisely → `docs/research/` (f11;~~ done — research notes 2026-10-25 (2,438B floor, ~2.5KB/row, 36.7KB @12 checks)
my ~600 B/row figure is a spot measurement, not a benchmark).

**Docs batch (P8):**
~~13. CHANGELOG `[Unreleased]` entries for P1-P4 (+ the polarity bug fix).~~ done — [Unreleased] entries landed (12-26 docs batch)
~~14. FEATURES.md: touch disclosures, chips (if kept), tone chips, latency~~ done — FEATURES rows + counts 346/47 (12-27/12-26)
budget tooltip; update test-count drift guard numbers.
~~15. TODO_LIST harvest: close touch-tooltips row (done), close f23 row~~ done — 32 rows closed with evidence (13-20 session)
(covered by TestBrowser_KeyboardLinks:1843-1876), close f11 row with
the measurement, update f10/f17/f9 rows as still-open.
~~16. AGENTS.md: templ-components pin text v1.21.0 → v1.23.0 (go.mod truth).~~ done — AGENTS pin text v1.23.0 (10-45)
~~17. `docs/gotchas.md`: the chromedp daemon-bump incident + the~~ done — gotchas chromedp + probe-methodology sections (10-45)
"probe the SDK grammar before shipping data-on-*" lesson (companion to
the existing colon-keyed gotcha).
~~18. AGENTS.md testing-patterns note: `go test .` in the devShell INCLUDES~~ done — AGENTS browser-in-go-test note (10-45)
browser tests (Chrome env is set) — "unit green" claims need `-run`
scoping or a browser-suite pass.
~~19. ROADMAP theme 6: record sticky-header SKIP rationale (overflow wrapper).~~ done — ROADMAP theme 6 SKIP rationale (10-45)
~~20. README prose audit against the new UI (f9).~~ done — README prose audit (12-27 a) + Federation Hub section (48a8c11)
~~21. Harness `[hidden]` CSS fix (f10) + recapture.~~ done — browserStaticCSS preflight fix (5e50837) + production canvases (48a8c11)
~~22. GroupBySource screenshot + README section (f17).~~ done — 48a8c11 (TestCaptureREADME_ScreenshotGrouped + README section)

**Carried queue (unchanged, high-value):**
~~23. chromedp v0.21 migration as a deliberate task (or a fleet policy row:~~ routed — TODO_LIST chromedp row (open; pins restored v0.16.0 2026-10-25)
other repos will hit the same daemon bump).
~~24. Table-driven test for `resolvePushInterval` (zero coverage).~~ done — TestResolvePushInterval (12-26 a8)
~~25. Fuzz target for `rateLimiter.Allow` + fuzz.yml registry in SAME change.~~ done — FuzzRateLimiterAllow + fuzz.yml same change (12-26 a9)
~~26. Evidence `maxEvidenceChecks` overflow test.~~ done — TestEvidenceLog_CapsAtMaxEvidenceChecks (12-26 a13)
~~27. Export CSV formula-injection hardening test.~~ done — encoding/csv + exportCSVField + test (12-26 a6)
~~28. Public-mode differential leakage test HTML vs JSON.~~ done — TestPublicMode_DifferentialHTMLVsJSON (12-26 a14)
~~29. `statusValue` off-mapping exhaustive-switch guard test.~~ done — TestStatusValue_PinsEveryStatus (12-26 a11)
~~30. Nightly flake canary CI job (`-race -shuffle=on -count=5`).~~ done — flake-canary.yml (6153cf1)
~~31. Webhook bounded in-flight test vs slow receiver.~~ done — TestWebhook_BoundedInFlightVsSlowReceiver (12-26 a12)
~~32. Metrics label escaping property test vs fuzz corpus.~~ done — unescaped-quote containment property (12-26 a16)
~~33. health-hub `HEALTH_HUB_*` env table tests.~~ done — TestConfigFromEnv_HappyPaths (12-26 a15)
~~34. `WithOnDrop`-adjacent: `Dashboard.HealthCheck` dead-condition check~~ done — dead condition removed + pinned (12-26 a7/a8)
(`push.interval > 0`).
~~35. `check-ui-pins.sh` extension: RecommendedCSP↔SDK-version sync.~~ done — bundle sha256 + CSP sync checks (b516431)
~~36. LOADTEST re-run post-floor.~~ done — post-floor re-run in research notes (12-26 a35)
~~37. `BenchmarkHealthCheck` + small-quality batch (2026-10-11 f-21–f-26).~~ done — 9998c6f (68ns BenchmarkHealthCheck, selfmonitor, README semantics)
~~38. Consumer sweep: bump CV to v0.11.0 (Blocked row).~~ routed — TODO_LIST Blocked row (CV pins v0.10.2; sweep done 2026-10-25)
~~39. deploy/ stack bump to v0.11.0 + boot-verify (Blocked).~~ closed 2026-10-25 — BLOCKED: no container runtime on this host (12-26 addendum)
~~40. Release: cut v0.12.0 with this UI batch once green (cadence decision~~ routed — TODO_LIST Cut v0.12.0 BLOCKED row (cadence open)
pending touch-tooltips → this batch).
~~41. Upstream: StatCard green-tone contrast filing (Blocked, verification~~ routed — TODO_LIST Blocked row (draft verified, filing awaits authorization)
can start).
~~42. Upstream: templ-components ListNote custom message (filed #37).~~ filed templ-components#37; adoption Blocked row in TODO_LIST
~~43. Upstream: go-health#6 federation SourceStatuses (open).~~ filed go-health#6 (open upstream; TODO_LIST row)
~~44. ADR-0003 stat-card tone-system note (two-family rule, now also applies~~ done — ADR-0006 (5e50837)
to count chips).
~~45. Em-dash copy decision (Blocked row) — my new copy avoided them; old~~ routed — TODO_LIST Blocked row (em-dash copy decision)
strings still carry them.
~~46. `docs/status/2026-10-09` follow-ups still open: axe degraded-fixture~~ done — 48a8c11 (degraded axe both themes)
run, dark re-run evidence (f22/f38).
~~47. Screenshot one-command capture wrapper script (f42).~~ done — scripts/capture-screenshots.sh (b516431)
~~48. gopls stdversion re-check on upgrade (ROADMAP).~~ routed — ROADMAP (gopls recheck on upgrade)
~~49. Deprecate `WithNonce` in favor of `WithNonceExtractor` (ROADMAP).~~ routed — ROADMAP (deprecate WithNonce, long-term)
~~50. Browser golden-screenshot diff test (ROADMAP 5).~~ routed — ROADMAP theme 5 (golden-screenshot diff test)

## G) Questions (max 3)

~~1. **Chips direction:** A) dig the go-datastar source for the sanctioned~~ answered — path A taken; grammar fixed and chips green same day
   `on` grammar + assignment form (my preference: the SyntaxError smells
   like a probe-harness artifact; the grammar question is answerable from
   the unminified repo), B) server-side `?status=` chips, or C) drop the
   feature? (A ≈ 1-2 h, B ≈ half a day + patch-persistence design, C ≈ 10 min.)
~~2. **chromedp policy:** keep the v0.16.0 pin (current state) and file the~~ routed — TODO_LIST chromedp row (pin kept, migration deliberate)
   v0.21 migration as its own task, or migrate the browser/screenshot
   suites to v0.21 now (~100 mechanical call sites + harness rewrite for
   `remote.NewAllocator`)?
~~3. **Visual review:** I cannot view images in this session, so P1-P4 are~~ partially routed — screenshots recaptured 2026-10-25; human eyeball owed (TODO content-assertions row)
   verified by markup/goldens/browser assertions only — do you want a
   screenshot recapture + your own eyeball pass before the release cut?

_Arte in Aeternum — stopped cleanly, nothing hidden._
