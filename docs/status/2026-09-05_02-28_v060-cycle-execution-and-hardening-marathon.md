# v0.6.0 Release & Hardening Cycle — Execution Status

**Date**: 2026-09-05 02:28 CEST
**Session scope**: execute `docs/planning/2026-09-04_22-27_v060-release-and-hardening-cycle.md`
end-to-end (R/G/T/D/F/U tasks), plus everything the session discovered on the way.
**Final dashboard state**: master `0877669`, clean tree, CI run 33932180793 **success**
(all 7 jobs). Test suite: **188 test/benchmark/fuzz functions across 27 test files**,
coverage **84.7%** (library package), lint 0 issues, `nix flake check` green.
**Format note**: skill default is styled HTML; user explicitly requested `.md` —
Markdown used, override flagged.

---

## Verdict snapshot

| Category              | Count                                                                              |
| --------------------- | ---------------------------------------------------------------------------------- |
| a) Fully done         | 24 plan tasks + 4 discovered-and-fixed incidents                                   |
| b) Partially done     | 5 (PR #8 is the big one)                                                           |
| c) Not started        | 3 plan items (F5, F8, F11) + 3 user-blocked decisions                              |
| d) Totally fucked up  | 6 incidents, all recovered, 3 of them self-inflicted and visible in pushed history |
| e) Improvement themes | 8                                                                                  |
| f) Next tasks         | 50, sorted by impact                                                               |

---

## a) FULLY DONE

### Release

| Task                                      | Evidence                                                                                                                                                                      |
| ----------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| R1: push master + watch CI first real run | Run 33916788241: Test, Browser, Lint, Build, Version-guard, Vuln scan all green; coverage artifact uploaded                                                                   |
| R2: cut v0.6.0                            | Commit `26b85c5` = CHANGELOG re-head + `const Version` bump in the same commit; annotated tag; proxy resolved via `go list -m @v0.6.0`                                        |
| R2 bonus: CHANGELOG repair                | The SanitizeResponse Fixed entry, which an auto-commit had landed under v0.4.0 (it happened after v0.5.0), relocated into [0.6.0]; go-health bump bullets corrected to v0.1.3 |
| R3: GitHub Releases v0.2.0–v0.6.0         | Six pages created from CHANGELOG sections, `gh release list` verified; stale FEATURES gap row removed                                                                         |
| R4: release checklist                     | `docs/release-checklist.md` (reconcile → changelog → same-commit version bump → gates → tag → push → proxy/CI/Release verify); linked from CONTRIBUTING                       |

### Guards (the cycle's core purpose)

| Task                     | Evidence                                                                                                                                                                                                                                                                                                                                                                              |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| G1: UI pin-guard         | `scripts/check-ui-pins.sh` covers **root + sub-modules** (templ-components, templ-components/datastar, go-datastar, go-datastar/static); wired into CI Build+Test; failure path tested (temp bump → exit 1). **Caught a real fourth sweep within minutes of being written** — sub-modules had drifted while the root module looked pinned, which is exactly the hole the guard closes |
| G2: FEATURES drift guard | CI step recounts test/bench/fuzz funcs + files vs the FEATURES claim; **failed its first real run on master because the claim was actually wrong (185/20 vs 188/27)** — the guard catching its author is the strongest possible validation                                                                                                                                            |
| G3: hygiene job          | actionlint v1.7.12 on both workflows, templ generate + nix fmt + git diff drift check, `nix flake check`; green on runner (run 33919924925, 33932180793). Discovered the canonical generated-file form is generate **then** treefmt (gofumpt regroups templ's imports)                                                                                                                |

### G4 + test surge

| Task                         | Evidence                                                                                                                                  |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| G4: Chromium devShell        | `pkgs.chromium` + `GO_HEALTH_DASHBOARD_CHROME` in flake devShell; all 3 browser tests green locally for the first time ever               |
| T1: FuzzCSVExport            | 60s, 7.7M execs, 0 crashers; header/round-trip/determinism invariants                                                                     |
| T2: FuzzRecommendedCSP       | 60s, 6.3M execs; structural nonce-source assertions (my first two invariants were naive substring checks — fixed after they false-failed) |
| T3: FuzzWebhookPayload       | 60s, 5.8M execs; masking verified structurally on the decoded checks map, not substrings                                                  |
| T4: keyboard-nav smoke       | `TestBrowser_KeyboardNavigation`: real Tab walks, ≥2 distinct visible outlined targets, never dies on `<body>`                            |
| T5: metrics under strict CSP | `TestBrowser_MetricsUnderStrictCSP`: same-origin fetch + parseable exposition + silent console                                            |
| T6: coverage                 | 76.9% → **84.7%**, no function < 60%; floor 75% → 78%                                                                                     |

### Docs

| Task                    | Evidence                                                                                                                                                                                                                                                                                                |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| D1: ADRs                | `docs/adr/0001-dashboard-file-split.md`, `docs/adr/0002-error-sentinel-family.md`; cross-linked from AGENTS Key Design Decisions                                                                                                                                                                        |
| D2: gopls               | Verified empirically: warning fires identically with AND without `GOEXPERIMENT=jsonv2` on gopls v0.23.0; dismissed via `analyses.stdversion: false` (editor-only); ROADMAP carries the re-check note. A web fetch of upstream status produced contradictory claims — ignored in favor of local evidence |
| D3: AGENTS prune        | 25.3 → 23.0 KB; sweep-incident narrative, bisect SHA list (→ archived audit), and misplaced test names removed; stale facts refreshed (fuzz count, browser env, status line)                                                                                                                            |
| D4: cookbook            | `docs/cookbook-probe-options.md`: 3 recipes; caught two wrong claims in my own draft against go-health v0.1.3 source (WithAllowedMethods ⇒ WithGETOnly; WithLiveThrottle no-op under background cache)                                                                                                  |
| D5: tag signing + links | CHANGELOG footer compare links for all 8 versions (verified initial SHA `01277d3`); signing how-to inside the release checklist                                                                                                                                                                         |

### Features

| Task                             | Evidence                                                                                                                                                                                                           |
| -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| F1: introspection                | `WithIntrospection()` + `GET /health/introspect` (`Routes.Introspect`); deterministic JSON config doc; leak test against a failing probe; 404-by-default pinned; middleware + rate-limit wrapped                   |
| F2: 429 JSON + drain Retry-After | `wantsJSON` negotiation on 429 (`{"error","retry_after"}`); SSE 503 during drain carries `Retry-After` (extracted `writeSSEUnavailable` helper after cyclop flagged the inline branch)                             |
| F3: webhook metrics              | `dashboard_webhook_deliveries_total{result="ok"\|"error"}` + `dashboard_webhook_delivery_duration_seconds` histogram (shared bounds via new `renderNamed`); family only when metrics+webhook both on               |
| F4: TTL + age cap                | `WithPushOnChangeTTL(n)` (counter reset on real change; <2 ignored) + `WithTimelineMaxAge(d)` (transitions filtered; trend/export keep full history); unit-tested directly on `shouldBroadcast`/`populateHistory`  |
| F6: embedded SDK                 | `WithEmbeddedDatastarSDK()` + `GET /health/datastar.js` serving `go-datastar/static` bytes; script tag repointed after route resolution; participates in BasePath/middleware/rate-limit                            |
| F7: aggregate/webhook demos      | `DEMO_AGGREGATE=1` verified live: `api/postgres`, `api/redis`, `worker/metrics-exporter` namespaced rows, worst-of warn. `TestBrowser_AggregateCSPClean` proves namespaced rows + style-free runtime + SSE connect |
| F9: Routes + BasePath            | Prefix stored, applied once post-options; ordering test pins BOTH orders produce identical prefixed routes; `Routes()` accessor                                                                                    |
| F10: self-monitoring             | Empirically pinned: `Register`'d Dashboard appears in its own table, self-reports pass (`selfmonitor_test.go`); ROADMAP decision: feature, keep                                                                    |
| F12: leak scanner + InstanceID   | `TestPublicMode_LeakScanner` sweeps HTML+metrics for names/errors and pins the documented probe-stay-verbatim boundary; InstanceID UI deferred in ROADMAP                                                          |

### Upstream (templ-components)

| Task                    | Evidence                                                                                                                                                                                                                                                                                     |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| U1: nonce-guard PR      | [PR #8](https://github.com/LarsArtmann/templ-components/pull/8): conditional `templ.Attributes` splat (single JS source, no if/else duplication); repro test failed on master first; 2 regression tests + goldens; **Fixes #7**. On-branch status: Build/Test/Lint/CSS/CHANGELOG-warmth pass |
| U2: StatCard `<dl>` fix | **Landed on upstream master** (`63927bd` + goldens `8c87cec`, `4d1308a`): group div now wraps the dt+dd pair; structural regression test pins it                                                                                                                                             |

### Incidents survived (each fixed + documented in-repo)

1. **Fourth dependency sweep** (auto-commit `a8fa43f`) — restored; sub-module discovery hardened the guard.
2. **Raw templ output committed** (`47baa6b`) — reverted to formatted form; CI drift check now runs generate + fmt + diff (see d3).
3. **CI coverage dilution** — `./...` profile included the untested example (74.7%); scoped to the library package (84.7%), commit `0877669`.

---

## b) PARTIALLY DONE

| Item                          | State                                                                                                                        | What's missing                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **PR templ-components#8**     | Core fix green (Build/Test/Lint/CSS/CHANGELOG-warmth); branch tip `a19a9b2` pushed                                           | **Visual Regression fails on the merge preview** — master and the PR branch both modified the same binary statcard PNGs; the preview can't auto-merge binaries so CI diffed stale pairs. Needs a merge of upstream master into the branch (resolving binaries to the post-#6-fix renderings). The shared worktree is being used by a concurrent session that is actively landing goldens on master (`4d1308a`) — I stopped rather than clobber |
| **F7 aggregate/webhook demo** | Modes built, aggregate verified live, browser test green                                                                     | No demo webhook _receiver_ (a tiny httptest-style receiver that prints payloads was in the plan as F7.2); `safeWebhookURL` has **no unit test**                                                                                                                                                                                                                                                                                                |
| **D3 AGENTS prune**           | 25.3 → 23.0 KB; temporal pollution gone                                                                                      | ~18 KB target not reached; the remaining mass is load-bearing (decisions 7.5 KB, gotchas 5.1 KB). Diminishing returns — flagged, not forced                                                                                                                                                                                                                                                                                                    |
| **Living-docs sync**          | FEATURES counts fixed (188/27), TODO_LIST gained the remaining-work table, CHANGELOG [Unreleased] has F1–F4/F6/F7/F9 entries | All done piecemeal across ~15 daemon-raced commits; **no single coherence pass** over [Unreleased] as a narrative. FEATURES Known-Gaps row 158 still says "upstream #6/#7 open" — #6 is fixed upstream now (stale)                                                                                                                                                                                                                             |
| **README demo toggles**       | `/health/introspect` + `/health/datastar.js` rows added; `DEMO_PUBLIC`/`DEMO_BASE_PATH` documented earlier                   | `DEMO_AGGREGATE` / `DEMO_WEBHOOK` (and the new F4/F6 option demos) are only documented in `example/main.go`'s header, not README                                                                                                                                                                                                                                                                                                               |

---

## c) NOT STARTED

| Item                                            | Why                                                                                                                                 |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| F5: per-check latency series + NDJSON export    | Plan tail (Low impact); recorded in TODO_LIST                                                                                       |
| F8: 20-source aggregate load test               | Plan tail (Low); depends on F7 harness (now exists)                                                                                 |
| F11: `WithGrouping(BySource)` per-service cards | Med impact / 90min view-model redesign                                                                                              |
| U3: pin lift                                    | **BLOCKED** on PR #8 merging + an upstream templ-components release; the removal condition is documented in the guard script header |
| The 3 user-blocked decisions                    | Build-tag gating for SSE, fingerprint format stability, webhook HMAC/schema — intentionally never smuggled into tasks (guard #7)    |

---

## d) TOTALLY FUCKED UP

All recovered; three are permanent (harmless) scars in pushed history.

| #  | Incident                                                                                                                                            | Root cause                                                                                                                                                                                                                          | Recovery                                                                                                                                  | Lasting damage                                                                                                                                         |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| d1 | **Committed raw `templ generate` output** (`47baa6b`) → `nix flake check`'s treefmt-check failed on master AND in CI (hygiene job's first real run) | I ran generate, saw a diff vs the committed file, assumed staleness, and committed generator output without `nix fmt`. gofumpt regroups templ's single-line imports — raw output is never committable                               | Reverted to formatted form (`7ca6d0f`); drift check redesigned to generate + fmt + diff                                                   | One extra red CI run; a second truth-scar commit pair                                                                                                  |
| d2 | **Coverage floor failed on CI** (74.7% vs my 78% floor)                                                                                             | I raised the floor from a LOCAL profile (`go test .`) while CI profiled `./...` — the untested example package diluted CI's number. Two commands, two different measurements, I never compared them                                 | Scoped CI coverage to the library package (`0877669`) — the example is a demo, not library surface                                        | One red CI run; the floor was briefly unenforceable                                                                                                    |
| d3 | **My drift guard failed its first real run** (185/20 claim vs 188/27 reality)                                                                       | Across five batches I updated the function count by sed four times and the file count zero times — exactly the memory-failure the guard exists to catch                                                                             | Fixed the claim (`d918122`)                                                                                                               | None — this is the guard working, but the irony is recorded                                                                                            |
| d4 | **Python heredoc splices corrupted source repeatedly**                                                                                              | `\\n` in my command text got JSON-unescaped to a real newline before bash/python saw it; metrics.go ended up with unterminated string literals three separate times; one edit also injected a literal `Maps` typo into fuzz_test.go | Restored metrics.go from git and rebuilt the splice with `chr(92)` composition; rewrote introspect_test.go and fixed fuzz_test.go by hand | Wasted ~10 tool cycles; no pushed damage                                                                                                               |
| d5 | **U2 committed to the wrong branch** in templ-components                                                                                            | The concurrent session's branch switches + the daemon's auto-commits meant my `git add && git commit` landed on local **master**; the daemon then pushed master upstream, and an earlier stale-branch push had to be deleted        | Verified the net upstream content was correct (it was); deleted the stale remote branch; documented the two-commit split in messages      | Templ-components history has my work as scattered daemon+mine commits instead of one clean PR — for U2 specifically, no clean PR exists                |
| d6 | **Multiedit/write races with the auto-commit daemon**                                                                                               | ~6 tool calls failed with "file modified since read" and one multiedit's failed apply left a test file mangled until rewritten                                                                                                      | Adopted atomic single-pass strategies (python with chr() escapes, line-based edits, full-file rewrites via heredoc)                       | Slowed everything down; the commit-message/content split (daemon commits carrying my work, my commits carrying the message) recurs ~5 times in history |

---

## e) WHAT WE SHOULD IMPROVE

1. **Count-claims must never be hand-maintained.** The drift guard proves even the agent writing the guard drifts. Make `scripts/check-ui-pins.sh`-style guards also run as a pre-commit hook locally, and add a `nix run .#verify-docs` app that recomputes FEATURES counts and fixes them or fails.
2. **Canonical pipeline discipline.** Any generated file (`*_templ.go`, codegen output) is committed only after the full pipeline (generate → fmt) — ideally enforced by the pre-commit hook (the templ-components repo already has this pattern via `check-templ-sync.sh`; this repo should too).
3. **Local vs CI measurement parity.** Before moving any threshold (coverage floor, visual thresholds), diff the exact local command against the CI step. The two coverage commands measured different package sets.
4. **Never splice escape-heavy code via inline heredocs.** Tool-layer JSON escaping turned `\n` into real newlines three times. Use file-based splices, the edit tools, or chr()-composition — or better, avoid refactor-by-script entirely when LSP tools apply.
5. **Push smaller batches.** The daemon races large uncommitted trees; several commits landed with heuristic messages carrying my work while my message-carrying commit held only the tail. Smaller batches = message/content pairing survives.
6. **Multi-agent worktrees.** In templ-components the shared worktree switched branches under me and reached unmerged states. Any cross-repo PR work should happen in a dedicated `git worktree`, never the shared checkout.
7. **Verify claims against source, always.** The cookbook and fuzz invariants both had false statements caught only because I checked the counterparty source / ran the fuzzer. The verify-external-claims discipline also applies to my own first drafts.
8. **Session-end coherence pass.** After 15+ commits, [Unreleased], FEATURES Known Gaps, and README need one dedicated review pass — three stale spots survived to the end (pin row, README toggles, [Unreleased] narrative).

---

## f) NEXT 50 THINGS TO GET DONE

Sorted by impact. Items 1–10 are the real next cycle; 11–50 are brainstorm-grade (docs-health HARVEST should apply routing rigor).

| #  | Task                                                                                                                                         | Impact |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1  | Resolve PR #8: merge upstream master into the branch, resolve the binary statcard goldens to post-#6 renderings, get Visual Regression green | High   |
| 2  | Respond to CodeRabbit review on #8 (rate-limited at session end)                                                                             | High   |
| 3  | After #8 merges: watch for the templ-components release that carries the nonce guard                                                         | High   |
| 4  | U3 pin lift: `go get` new templ-components + re-run the go-datastar v0.5.0 decision, browser-validate, CHANGELOG                             | High   |
| 5  | Retire `scripts/check-ui-pins.sh` + its two CI steps per the documented removal condition (only after 4)                                     | High   |
| 6  | Retire the axe `definition-list` tolerance in `browser_test.go` (upstream #6 fix ships with the release)                                     | High   |
| 7  | Add pre-commit hook (or `nix run .#verify` app) running pin-guard + `nix fmt --check` + FEATURES recount locally before every push           | High   |
| 8  | v0.7.0 cut when [Unreleased] is coherent: introspection, embedded SDK, F2/F3/F4/F9, guards (follow the new checklist)                        | High   |
| 9  | F11: `WithGrouping(BySource)` per-service cards                                                                                              | Med    |
| 10 | F5: per-check latency series + `?format=ndjson` export                                                                                       | Med    |
| 11 | F8: 20-source aggregate load test; record numbers in `docs/research/`                                                                        | Med    |
| 12 | Unit tests for `safeWebhookURL` (currently untested)                                                                                         | Med    |
| 13 | Example: webhook receiver demo mode (prints payloads) — the thin half of F7.2                                                                | Med    |
| 14 | README: demo-toggle rows for `DEMO_AGGREGATE` / `DEMO_WEBHOOK`                                                                               | Med    |
| 15 | Fix the stale FEATURES Known-Gaps pin row (#6 fixed upstream; wording: "#7 in review PR #8")                                                 | Med    |
| 16 | CHANGELOG [Unreleased] coherence pass (10+ accumulated entries → one narrative)                                                              | Med    |
| 17 | Example: `DEMO_TTL` / `DEMO_TIMELINE_AGE` toggles for the new F4 options                                                                     | Med    |
| 18 | SSE connection counters (`dashboard_sse_connections_opened/closed_total`)                                                                    | Med    |
| 19 | `atCapacity` 503 (SSE connection limit) should also carry `Retry-After`                                                                      | Med    |
| 20 | Introspection: expose rate-limit requests+window (not just enabled bool)                                                                     | Med    |
| 21 | Introspection: ETag / If-None-Match support                                                                                                  | Med    |
| 22 | Introspection: include templ/go-sse/go-datastar versions                                                                                     | Low    |
| 23 | CONTRIBUTING: document the two CI guards so contributors know why they fail                                                                  | Low    |
| 24 | AGENTS.md: document `safeWebhookURL` as the env-validation convention alongside safeBasePath                                                 | Low    |
| 25 | `docs/DOMAIN_LANGUAGE.md`: introspection, nonce-strategy, delivery-stats terms                                                               | Low    |
| 26 | Amend the bisect-wall audit: the raw-templ commit (`47baa6b`) and neighbors are new mid-edit scars                                           | Low    |
| 27 | Nightly job: verify a GitHub Release page exists for every tag (catch R3-class backfill gaps)                                                | Low    |
| 28 | Test guarding CHANGELOG compare links (one per version heading)                                                                              | Low    |
| 29 | Screenshot regeneration (`SCREENSHOT_OUTPUT=docs/screenshot.png`) + dark-mode pair                                                           | Low    |
| 30 | Benchmark: compare `BenchmarkHandler_HTMLRendering` v0.6.0 vs v0.5.0; record                                                                 | Low    |
| 31 | Fuzz target: introspection marshal (deterministic JSON invariant)                                                                            | Low    |
| 32 | Fuzz target: view-model buildData (unknown statuses, hostile names)                                                                          | Low    |
| 33 | SSE fan-out load check: concurrent client count before broadcast latency degrades                                                            | Low    |
| 34 | Rate limiter: expose remaining tokens in introspection (debug mode)                                                                          | Low    |
| 35 | ROADMAP: mirror the F5/F8/F11 entries into Themes so TODO_LIST and ROADMAP agree                                                             | Low    |
| 36 | gopls stdversion re-check when nixpkgs bumps gopls (ROADMAP note exists)                                                                     | Low    |
| 37 | ErrPusherStale: expose last-stale-at timestamp in introspection                                                                              | Low    |
| 38 | Coverage artifact: include both the library profile and a full `./...` profile for context                                                   | Low    |
| 39 | Webhook delivery: structured outcome codes (timeout vs non-2xx vs network) in the metrics label                                              | Low    |
| 40 | `WithShutdownDrain` + `Retry-After`: document the interaction in README's hardening section                                                  | Low    |
| 41 | Example: smoke test that `go build ./example` + flag parsing stays valid in CI                                                               | Low    |
| 42 | InstanceID StatCard (deferred decision — only on demand signal)                                                                              | Low    |
| 43 | Webhook HMAC signing + `"schema"` version field (BLOCKED on user decision)                                                                   | Low    |
| 44 | Build-tag gating for SSE (BLOCKED on user decision)                                                                                          | Low    |
| 45 | Fingerprint format stability guarantee (BLOCKED on user decision)                                                                            | Low    |
| 46 | `RegisterNamed` multi-instance support (ROADMAP raw idea)                                                                                    | Low    |
| 47 | do child scopes: document/test dashboard registration inside scopes (ROADMAP)                                                                | Low    |
| 48 | `WithNonce` deprecation path toward `WithNonceExtractor`-only (ROADMAP)                                                                      | Low    |
| 49 | Local `nix run .#ci` mirror of the GitHub Actions steps (ROADMAP)                                                                            | Low    |
| 50 | templ-components UI follow-ups: adopt PageHeader/Stack/Dot after upstream releases (ROADMAP)                                                 | Low    |

---

## g) QUESTIONS (cannot figure out myself)

1. **PR #8 ownership**: the branch needs a merge of upstream master to fix the binary-golden Visual Regression, but the shared templ-components worktree is actively used by another session that is landing goldens on master. Should I do the merge myself in a **separate `git worktree`**, or hand #8 (and its follow-ups) entirely to the concurrent session and only own the dashboard side?

2. **v0.7.0 cadence**: cut v0.7.0 now from the current `[Unreleased]` (introspection, embedded SDK, 429/Retry-After, webhook metrics, TTL/age cap, Routes accessor, guards) — or hold the release until F5 + F11 land so the aggregate story ships complete?

3. **Branch protection**: the pin-guard and drift-guard each caught real drift on their first real runs — but only after landing on master. Do you want me to configure branch protection making Test (which runs both guards) + Hygiene **required checks**, or keep them advisory?

---

_Point-in-time snapshot. When a later session brings this current: docs-health ANNOTATE mode — inline corrections or appendix, never rewrite. Section (f) is the primary input for docs-health HARVEST into TODO_LIST.md / ROADMAP.md._
