# TODO-List Execution Session — Status Report

**Date:** 2026-10-25 12:26 CET
**Session goal:** "GET SHIT DONE! The WHOLE TODO LIST!" — execute every 🔴 TODO row
from `TODO_LIST.md` (2026-10-13 harvest + UI-revamp harvest + pareto-plan remnants),
verify each step, commit per verified batch.
**Starting point:** HEAD `390bf2d`; the morning session (07:39 report) had left
`TestBrowser_StatusFilterChips` RED and a P8 docs batch open; a PARALLEL session
was simultaneously landing UI-batch closeout work (README, screenshots, AGENTS).
**Session commits:** `edf5717` (hardening batch), `5a48289` (timing sweep),
`b516431`+`5034088` (scripts batch), `6153cf1` (flake canary), `5e50837` (docs
batch, amend-relabeled daemon carrier), `9998c6f` (measurement-code batch).

## Gates at stop time

- Root package `go test . ./cmd/health-hub`: GREEN (18.7s, browser suite included)
- `go test -race -shuffle=on -count=2 .` (canary-mode sanity): GREEN (34.4s)
- `bash scripts/check-ui-pins.sh` (with new bundle-sha + CSP-sync checks): GREEN
- `nix run .#lint` / `pre-push-checks.sh`: **NOT RUN YET** — the final-gates step
  was not reached; stale LSP diagnostics (gocognit/gci on main_test.go) are
  unconfirmed against the real linter
- Working tree: clean at report time (auto-daemon committed everything)

---

## a) FULLY DONE (verified this session)

### Unblocked the red suite
1. **`TestBrowser_StatusFilterChips` red → GREEN** — a daemon commit after the
   07:39 report had already re-wired the chips to the colon-keyed
   `data-on:click` grammar (the templ comment documents the empirical probe).
   Verified PASS (1.20s) together with the previously-unverified
   `TestBrowser_EvidenceDisclosureFocus` (1.76s). Full-suite baseline green.

### Rows verified as ALREADY DONE (closed with evidence, no code needed)
2. T10 (ADR-0003 evaluation-sink), T11 (Observe + WithEvaluationHook wiring +
   `evaluation_sink_test.go`), T12 (ADR-0004 + `WithCascadeProbeVerdict`),
   T14 (hub transport knobs: `HEALTH_HUB_CLIENT_TLS_CA` + `HEALTH_HUB_HTTP_PROXY`
   with validate-before-use), T19 (AGENTS daemon-race recipes at AGENTS.md:185).
3. UI-revamp f23 (keyboard links): `TestBrowser_KeyboardLinks`
   (browser_test.go:1999) covers all four endpoint glyph links — row closes.
4. UI-revamp "touch story for evidence tooltips": P1 disclosures landed by the
   morning/parallel sessions; browser-verified green this session.
5. UI-revamp "transitive-only helper coverage": `TestTimelineEntryMappers_OffIsNeutral`
   + grouped-VM `TestAlertSummary` cases already exist in status_test.go.

### Hardening/test batch (commit `edf5717`)
6. **CSV export formula-injection hardening + test** (trend.go): `exportCSVField`
   neutralizes `= + - @ \t \r` starters with a leading quote, RFC-4180 quoting
   moved to `encoding/csv`; `TestExportHandler_CSVFormulaInjectionNeutralized`
   pins it (stub-prober adversarial statuses through the public `Observe` seam,
   goroutine-free).
7. **Dead condition removed**: `push.interval > 0` in `HealthCheck` —
   `resolvePushInterval` (called in New) guarantees positivity; now pinned by:
8. **`TestResolvePushInterval`** — 6-case table (configured > probe > default,
   negative handling) + the positivity invariant the watchdog relies on.
9. **`FuzzRateLimiterAllow`** + fuzz.yml registry step in the SAME change;
   burst-grant/empty-deny/retryAfter-nonnegative invariants with a pinned 1h
   window making them deterministic.
10. **`ratelimit_internal_test.go`** (new file): deterministic refill/ceiling
    (clock backdating), constructor clamps, concurrent-Allow race safety.
11. **`TestStatusValue_PinsEveryStatus`** — pins the exhaustive switch incl.
    off=1 (pass-tier) and unknown→trendFailValue.
12. **`TestWebhook_BoundedInFlightVsSlowReceiver`** — 40 transitions against a
    hanging receiver: peak concurrency ≤ 8 (webhookMaxInFlight), excess dropped,
    bound recovers post-drain; race-clean.
13. **`TestEvidenceLog_CapsAtMaxEvidenceChecks`** — 10k cap: new names stop
    accruing, existing entries keep updating, capped-out names report unproven.
14. **`TestPublicMode_DifferentialHTMLVsJSON`** — one dashboard, one response:
    HTML masked, negotiated JSON byte-verbatim (and no masked names leak into JSON).
15. **`TestConfigFromEnv_HappyPaths`** (cmd/health-hub) — full env contract table
    (defaults / every knob / empty-knobs), with extracted `wantConfigDefaults` /
    `wantConfigEveryKnob` helpers to stay under gocognit.
16. **Label-escape fuzz strengthened** — unescaped-quote-containment property
    (no raw `"`/`\` survives inside `label="..."`).

### Timing-flake sweep (commit `5a48289` + part of `9998c6f`)
17. `TestRateLimit_ExceedsBudget` + `TestRateLimit_429JSONBody`: refill windows
    1s/2s → 1h — the 429s are now budget facts, killing the 2026-10-11
    "200-instead-of-429" race class.
18. `TestRateLimit_RefillsOverTime`: event-driven (exhaust-until-429 →
    poll-until-200); the 1300ms magic sleep is gone (also closes the
    "name the rate-limit constants" f-27 row — premise dissolved).
19. Drain waits audited: lower-bound-only assertions are load-immune (left
    alone); poll loops audited sound.

### Scripts batch (commits `5034088`/`b516431`)
20. **check-ui-pins.sh extended**: pinned SDK-bundle sha256 (`5d6b7794…`,
    cross-checked against upstream's own checksum test) + byte size, and
    CSP↔SDK sync checks (`'unsafe-eval'` presence + rationale comment while the
    pinned SDK needs it). All green.
21. **verify-release.sh**: self-selecting toolchain (`GO` array; probe-then-
    devShell fallback) — the root-fix for the "scripts that exec go" gotcha.
22. **pre-push-checks.sh check 7**: SECURITY.md freshness (supported row vs
    latest-tag minor, footer date vs tag date) — caught LIVE rot on first run:
    SECURITY.md still said 0.10.x against v0.11.0; fixed (0.11.x, 2026-10-25).
23. **scripts/capture-screenshots.sh** (new): one-command wrapper for the three
    README captures (light/dark/degradable subsets, loud failure on missing PNGs).

### CI batch (commit `6153cf1`)
24. **.github/workflows/flake-canary.yml** (new): nightly
    `go test -race -shuffle=on -count=5` full package (Chrome installed, seed
    surfaced on failure) — the agreed post-no-WithClock guardrail. Local
    count=2 sanity run green.

### Docs batch (commit `5e50837`; parallel-session overlap credited)
25. **ADR-0006** (`docs/adr/0006-statcard-tone-system.md`): two-family stat-card
    tone decision with the measured WCAG ratios and the page-wide green-tile ban.
26. **release-checklist**: §7 pre-tag verifier expectations (v0.11.0 d-4
    sha-order lesson, anti-pattern ban), §8 close-the-loop renumber, condensed
    daemon-intent recovery recipes, de-hardcoded pin versions, screenshot
    wrapper step.
27. **T09 audit-report quality pass**: typo confirmed fixed, all 7 internal
    anchors resolve, no empty sections, citations spot-checked (2 moved
    file:line refs match the audit-time disclaimer), dated re-check addendum
    scopes the 78/100 score to v0.5.1 and lists the since-adopted findings.
28. **Browser harness `[hidden]` fix** (browser_test.go): `browserStaticCSS`
    gains preflight's `[hidden]{display:none!important}` — the connection pill
    stops rendering three states at once in browser tests (f10; real
    deployments load full Tailwind). Pill/filter/chips tests green.
29. **README token-bucket semantics** (f-26): continuous refill, denied requests
    consume nothing, `Retry-After ≈ ceil(window/budget)` when empty.

### Measurement batch — numbers CAPTURED (commit `9998c6f` for the code)
30. **`TestSelfMonitoring` restructured**: poll-until-pass instead of the 80ms
    sleep — the poll EXPOSED that the row legitimately ARRIVES as warn
    ("Start has not been called") before dash.Start lands in the probe cache;
    now asserts steady-state pass + empty error. 5×race green.
31. **`BenchmarkHealthCheck`** (new): 68.10 ns/op, 66 B/op, 0 allocs/op steady
    state (do-cascade hot path).
32. **HTML rendering benchmarks re-run** (3×2s): Handler 44.0–54.9 µs/op
    (Sept HEAD was 40.4 µs — the disclosure/chips UI added ~4–10 µs);
    FullHTML 74.3–118 µs; FullHTMLWithTrend 120–131 µs.
33. **SSE patch payload measured** (throwaway `cmd/measure-patch`, deleted
    after): 2,438 B/event static overhead (0 checks), ≈2.5 KB per check row
    steady-state (12 checks → 36.7 KB/event); page HTML 17.2 KB (0 checks) →
    50.5 KB (12).
34. **Chrome launch latency re-measured**: 85.45 ms avg / 90.3 ms max over 5
    launches (was ~155 ms in the Sept M91 measurement; chromium 155 here).
35. **LOADTEST re-run post-floor**: 20×3 checks, 20 clients, 1,003 events/5s
    (~50/client), first event 10.9 ms, scrape p50 1.25 ms / p95 7.44 ms /
    max 16.3 ms — PASS, healthy.

### External checks
36. **pkg.go.dev spot-check for v0.11.0**: ✅ rendered (Oct 12, 2026), Latest,
    full symbol diff shown (Observe, DroppedPatchCount, SourceStatuser,
    WithCascadeProbeVerdict…).
37. **Consumer sweep**: 11 sibling consumers found. Already on v0.11.0 (7):
    DiscordSync, company-master-data, file-and-image-renamer, go-taskqueue,
    library-policy, timesheets, webphone. Stragglers: nsfw-classifier (v0.10.2),
    dnsblockd (v0.10.1), CV/cv (v0.10.2 — the standing BLOCKED row).

## b) PARTIALLY DONE

1. **nsfw-classifier bump to v0.11.0**: baseline `go build` green; `go test ./...`
   printed a bare `FAIL` tail after two green packages — the failing package
   name was truncated in the captured output and NOT yet diagnosed
   (pre-existing red vs dashboard-related: unknown). Bump not applied yet.
2. **Measurement documentation**: all numbers captured (see a-30..35) but NOT
   yet written into `docs/research/` — the research-note file is the missing
   half of f11/f12/f-45 and the Chrome-latency refresh.
3. **Daemon-race discipline**: the daemon won the edit→commit race 2× this
   session (hardening batch, docs batch); both recovered via the
   verify-HEAD-then-amend recipe with disclosure in the intent message — but
   the races still cost a retry cycle each.

## c) NOT STARTED

1. dnsblockd bump (v0.10.1 → v0.11.0) — baseline not yet run (no vendor dir,
   flake present).
2. deploy/ demo stack bump to v0.11.0 + boot-verify — Docker availability
   unprobed; ports collide per AGENTS (needs compose override with `!override`).
3. Git-hygiene row: relabel daemon commits `70d10b3`/`bfd1b1c`/`70b3282` —
   feasibility not yet checked; they are 2 weeks buried under hundreds of
   daemon commits, so amend requires history rewrite + force push (banned
   without approval) — likely closes as "not amendable, premise dissolved".
4. Browser-evidence rows: hub e2e vs the reworked UI with 2 remotes (f36),
   axe audit on the degraded variant + dark re-run evidence (f22/f38),
   GroupBySource page screenshot + README section (f17).
5. Final harvest: TODO_LIST.md row closures (this report's §f feeds it),
   CHANGELOG `[Unreleased]` entries for everything in §a, FEATURES.md
   test-count drift-guard update (I added ~15 test functions + 1 file + 1
   benchmark — the count guard WILL fail until updated).
6. Final gates: `nix run .#lint`, `bash scripts/pre-push-checks.sh` (now 8
   checks incl. the new SECURITY gate), full `nix run .#test-race`.
7. Upstream filings (templ gofumpt output; erraudit nolint-audit; StatCard
   green-tone contrast) — remain 🔵 BLOCKED on filing authorization.

## d) TOTALLY FUCKED UP (honest)

1. **I deadlocked the whole package for 10 minutes**: the first
   `TestEvidenceLog_Caps…` version called `snapshot()` (which takes
   `evidence.mu`) while HOLDING the lock — `go test` burned its full 600s
   timeout. I should have read the lock contract of every method I called
   under a held mutex before writing the test.
2. **Webhook test v1 deadlocked against itself**: I waited on `handled`
   (incremented at handler RETURN) before closing the `release` channel that
   lets the handler return. Fixed by waiting on the entry-side `inHandler`
   counter. Same lesson: counter semantics (entry vs exit) matter when the
   test controls the unblock.
3. **`t.Setenv` × `t.Parallel` panics, twice** (subtest, then parent) before I
   landed the non-parallel shape. This is a known Go contract; I wrote the
   parallel forms reflexively from the repo's house style.
4. **Three type errors in one test** (`health.Status` vs `string` in the CSV
   test) — a first draft written without compiling mentally against the
   actual types.
5. **I wrote a factually wrong README claim** ("Retry-After reads 1 whenever
   the bucket is empty") and only caught it by re-deriving the formula from
   the code (`ceil(window/budget)`). First-draft prose about numeric behavior
   needs the derivation FIRST.
6. **Ambiguous red baseline accepted**: the nsfw-classifier baseline FAIL tail
   was not chased to the failing package before the session paused — the bump
   decision now depends on facts I did not capture.
7. **Stale LSP noise never resolved against reality**: gocognit/gci/intrange
   warnings on main_test.go/fuzz_test.go persisted in diagnostics all session;
   my helpers-extraction SHOULD have fixed them, but `nix run .#lint` was
   never run to confirm — the gates step would have caught it.

## e) WHAT WE SHOULD IMPROVE

1. **Read the concurrency contract before testing a synchronized type** —
   2 of my 3 self-inflicted failures were lock/counter misuse in tests.
2. **Compile-check test drafts against real types immediately** (`go vet`
   after each append) instead of stacking type errors across a heredoc.
3. **Chase ambiguous reds to the failing package BEFORE moving on** — a
   truncated FAIL tail is not a baseline, it's a question.
4. **Run the cheap gate (lint) once per batch**, not only at session end —
   it would have settled the stale-diagnostic question hours ago.
5. **Parallel-session coordination worked, barely**: the other session's
   README/screenshot/AGENTS work was discovered by reading daemon commits
   before duplicating it — `git log` before every doc edit stays mandatory.
6. **The daemon won 2 of 8 commit races** — the amend-relabel recipe is now
   routine (zero damage), but intent commits should fire on the FIRST green
   tool call, not after the follow-up edit.

## f) Up to 50 things we should get done next (order = value density)

**Finish the session's own tail (highest value, mostly minutes):**
1. Diagnose the nsfw-classifier baseline FAIL (which package, pre-existing?),
   then bump to v0.11.0 (additive per pkg.go.dev diff) + `go mod vendor` +
   test + commit (go-ecosystem-upgrade protocol, already loaded).
2. dnsblockd v0.10.1 → v0.11.0 bump, same protocol (no vendor dir).
3. Write `docs/research/2026-10-25_research-notes.md`: HTML-render benchmark
   deltas, SSE patch-payload table (2.4 KB static / ~2.5 KB per row),
   BenchmarkHealthCheck 68 ns, Chrome latency 85 ms, LOADTEST post-floor
   numbers — closes f11/f12/f-24/f-25/f-45 documentation halves.
4. TODO_LIST.md harvest: close all §a rows with evidence links (incl. the
   rows verified-already-done: T10-T14/T19, f23, touch-tooltips, helper
   coverage, auto-daemon row → deliberate non-fix with rationale).
5. CHANGELOG `[Unreleased]`: hardening batch (CSV injection, dead condition),
   tests, scripts (pin-guard extensions, SECURITY gate, wrapper), flake
   canary, ADR-0006, checklist addendum, harness fix, README semantics.
6. FEATURES.md: drift-guard counts (+~15 tests, +1 file, +1 benchmark) + new
   capability lines (CSV hardening, canary, pin-guard bundle-sha).
7. `nix run .#lint` + settle the stale gocognit/gci/intrange diagnostics.
8. `bash scripts/pre-push-checks.sh` — first full run with the new check 7
   (SECURITY freshness) + check 1 (will fail until #6 lands; fix order: 6 → 8).
9. Git-hygiene row: verify `70d10b3`/`bfd1b1c`/`70b3282` are pushed ancestors
   → close row as not-amendable-without-force-push (premise dissolved).
10. deploy/ stack bump probe: `docker version` availability; if present, bump
    digests to v0.11.0 + boot-verify with remapped ports (compose override).

**Browser-evidence debt:**
11. Hub e2e vs the reworked UI with 2 real remotes (f36).
12. Axe audit on the degraded variant + dark re-run evidence for the new
    badge/timeline/chips markup (f22/f38).
13. GroupBySource page screenshot + README section (f17); consider extending
    capture-screenshots.sh with a `grouped` target.
14. Re-verify the three README screenshots show the NEW tooltips/chips UI
    (recaptured by the parallel session at 10:14 — eyeball pass still owed).

**Release train:**
15. Decision: cut v0.12.0 with the UI batch + this hardening train (question
    g-2) — then release-checklist §1-8 with the NEW pre-tag verifier section.
16. SECURITY.md freshness automation proved itself — mirror the row+date
    check into the release-draft workflow so CI catches it too.
17. Post-release: `verify-release.sh vX` + pkg.go.dev spot-check (now routine).

**Consumer/fleet follow-ups:**
18. CV/cv v0.10.2 → v0.11.0 (BLOCKED row — needs the deploy/rollout decision).
19. chromedp v0.16 → v0.21 migration as a deliberate dedicated task (or a
    fleet-policy row; the daemon-bump class will hit other repos).
20. Fleet script audit for the d-4 `sort | tail -1` sha anti-pattern
    (2026-10-12 f-47 — one instance found, siblings unverified).
21. Upstream filing authorization batch: templ gofumpt output, erraudit
    nolint-audit, StatCard green-tone contrast (drafts verified+ready).

**Quality backlog carried from the f-lists (not yet routed):**
22. Pusher-staleness observability: last-broadcast-age gauge (f-18).
23. Rate-limit metrics: 429 counter + Retry-After histogram (f-19).
24. `dashboard_sse_subscribers` gauge (f-20).
25. Introspection: resolved push interval, staleAfter, last-broadcast age (f-21).
26. Example server: fast-cadence demo mode exercising the watchdog floor (f-24
    of the 10-11 report).
27. Post-release API watch: triage feedback on Observe/SourceStatuser/
    DroppedPatchCount (f-48).
28. Fleet lint-pin policy + own-repo filing banner (standing BLOCKED §g row).
29. Evidence strip machine contract + persistence (standing BLOCKED row).
30. Compose-stack CI policy: at least the build-only job the e2e argues for.
31. Em-dash copy decision for shipped strings (standing BLOCKED row).
32. `ListNote` custom-message hook (templ-components#37 — upstream's move).

## g) Questions I cannot figure out myself

1. **CV deployment**: the BLOCKED row says the go.mod bump + live rollout are
   your call (CV pins v0.10.2; a safe bump needs `WithNoDatastarRuntime()` or
   the filter/pill renders dead). Authorize the bump+rollout now, or does CV
   stay on v0.10.2 until its next planned deploy?
2. **v0.12.0 cadence**: the UI batch (tooltips, chips, tone chips, latency
   budget) + this session's hardening train are all sitting in `[Unreleased]`.
   Cut v0.12.0 once the tail (§f 1-14) is green, or hold for more?
3. **Upstream filings**: three verified drafts are ready to file
   (templ-components: gofumpt output + StatCard contrast; erraudit:
   nolint-audit NEEDED verdict). Filing is a remote action under your
   identity — authorize filing all three, some, or none?
