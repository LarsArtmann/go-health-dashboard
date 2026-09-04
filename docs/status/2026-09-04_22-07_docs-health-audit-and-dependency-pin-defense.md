# Session Status — Docs-Health AUDIT & the Dependency-Pin Defense

**Date:** 2026-09-04 22:07 CEST
**Scope:** one session, two fights — (1) the full docs-health AUDIT the user ordered
("View ALL `**/2026-0*` files! Execute the docs-health SKILL!"), and (2) an
unplanned dependency-pin defense after VERIFY caught the tree red.
**Base:** `7f4763a` (session end) — clean tree, all gates green.
**Verification at time of writing:** build ✅ · vet ✅ · full `-race` suite ✅ ·
`golangci-lint` **0 issues** ✅ · `nix flake check` ✅ · repo-wide internal link
sweep ✅ (only illustrative/proposed/frozen refs remain) · test count
re-verified `rg -c` = **165 funcs / 20 files**.

> **Format override note:** the status-report skill's canonical output is a styled
> HTML dashboard; the user explicitly requested `.md` at a `.md` path — Markdown
> used, override flagged per the skill's rule.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                            | Evidence                                                                                                                                                                                              |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **Docs-health AUDIT executed per skill** — SKILL.md + references (report format, verify checklist, harvest guide, resolving-items tooling) loaded; all 36 `2026-0*` files examined | skill flow; recent reports read in full, older annotated reports triaged via targeted reads + residual scans                                                                                          |
| 2  | **Pin defense, round 1** — VERIFY found an undocumented `go get` sweep (templ-components v1.12.0, go-datastar v0.5.0) failing 3 CSP-invariant tests (`TestCSP_NoNonceRendersWithoutNonce`, `TestCSP_EmptyNonceRendersWithoutNonce`, `TestCSP_WithoutDatastarSrcUsesCDN`) | root cause verified at source in the module cache: v1.12.0 `liveRegionBusyScript` renders `<script nonce={ nonce }>` unguarded (upstream #7 unfixed); pins restored (`8cf2c62`) + nolint placement fix (`ad2447e`) |
| 3  | **Pin defense, round 2** — a concurrent sweep re-landed the same bumps (`0862981`); re-restored same session (`e607a31`). go-health v0.1.1 → v0.1.2 patch bump (`2761f76`) verified with the full race suite and **kept**             | `go.mod` state at HEAD: templ-components v1.11.0, go-datastar v0.4.0, go-health v0.1.2                                                                                                                |
| 4  | **CHANGELOG `[Unreleased]` completed** — both pin incidents documented, go-health bump, gate re-verification bullet, bisect-audit path fixed to `archived/`                        | `40a1778` + follow-up edit in `e607a31`                                                                                                                                                                |
| 5  | **AGENTS.md re-audited and de-drifted** — Status v0.4.0 → **v0.5.0**; test inventory 19 → **20 files** + `integration_test.go`; templ-components **pin gotcha** (with the twice-bitten evidence); `safeBasePath` env-toggle convention; go-health version note | `381d64e` + `fabf293`                                                                                                                                                                                  |
| 6  | **FEATURES.md rebuilt** — counts **165/20** (counting command preserved), new **Multi-Service and Integration** section (`Prober`, aggregate, cookbook), webhook rows, sentinel-family row, honest CI row (runner-unverified jobs called out), v0.5.0 Released row (proxy-resolved), Known Gaps: pin + GitHub-Releases gap | `381d64e`, `7f4763a`                                                                                                                                                                                   |
| 7  | **README.md completed** — `DEMO_PUBLIC` / `DEMO_BASE_PATH` toggle rows (drift fix from the sweep report), `Updated <time>` observation-time semantics, `HealthCheck` sentinel-family example, dependency version matrix | `381d64e`                                                                                                                                                                                              |
| 8  | **TODO_LIST.md rebuilt from HARVEST** — v0.6.0 cut (incl. watching the new CI jobs on a real runner), GitHub Releases v0.2.0–v0.5.0, upstream **#7** PR row added alongside #6; stale resolved-decision note removed; bisect-audit cross-link added | `c7e4f13`                                                                                                                                                                                              |
| 9  | **ROADMAP.md absorbed the brainstorms** — shipped multi-probe idea removed (v0.5.0 did it), resolved release-policy Open Question removed, webhook-hardening question added, Theme 2 rewritten around the shipped aggregate, **Theme 5 (Pipeline/Testing/Docs Tooling)** created with ~45 routed ideas (incl. the 04-41 DI-surface cluster, drain Retry-After, heartbeat-leak check, safeBasePath fuzz) | `c7e4f13` + follow-ups                                                                                                                                                                                 |
| 10 | **CONTRIBUTING.md** — 75% coverage floor (baseline 76.9%) + `nix run .#coverage` documented                                                                                        | this session                                                                                                                                                                                           |
| 11 | **ANNOTATE: ~240 inline verdicts** across 10 reports — 04-03, 04-21, 04-41, 04-42, v03x-execution-complete, 19-26 (full Next-50), 12-32, 09-02, post-v040, 21-30; g-questions answered inline in three reports; disposition notes at the f-sections of the three live reports | skill scripts (`annotate-prose.py`, dry-run first) + exact-line strikes; every marker carries a hash or verified evidence                                                                              |
| 12 | **ARCHIVE: 10 fully-done files moved** via `git mv` — status: 04-03, 04-21, 04-41, 04-42, v03x-execution-complete, 19-15 bisect audit; planning: Pareto plan, issue-drafts, decisions-notes, integration plan (annotated EXECUTED first) | `docs/status/` and `docs/planning/` now hold only the 6 live reports + `archived/`                                                                                                                     |
| 13 | **Duplicate test removed** — `TestHealthCheckWithContext_InterfaceSatisfied` deleted per its own 04-42 disposition; `dashboard.go:70-71` compile-time assertions are the strictly stronger check                                   | suite green after removal; count corrected 166 → 165 (`7f4763a`)                                                                                                                                       |
| 14 | **Reference graph repaired** — 6 stale refs to moved files rewritten, 5 `feedback/new/` → `feedback/archived/`, 1 missing `archived/` segment, 1 self-introduced wrong prefix fixed; full link sweep green | sweep rerun: only illustrative `...md` examples, the *proposed* `docs/release-checklist.md` filename, and one Go-generic false positive remain                                                        |
| 15 | **All gates green at session end** — build, vet, full `-race`, lint 0 issues, `nix flake check`, link sweep; working tree clean                                                    | final verification round                                                                                                                                                                               |

## b) PARTIALLY DONE

1. **CI verification of the 2026-09-04 job additions** — version-guard, coverage
   floor + artifact, concurrency, pins: locally validated (YAML/logic/SHAs) but
   never observed on a real runner; the fuzz workflow **was** dispatch-verified
   (run 33896794771) and the browser job was green (run 33763955031, 09-03).
   Observing them is folded into the v0.6.0 TODO row (nothing is pushed — no-push
   rule).
2. **Upstream templ-components work** — #6 and #7 have verified diagnoses and
   TODO_LIST rows; the PRs themselves: not started (sibling repo).
3. **AGENTS.md size** — 24.6 KB; grew again with the pin gotcha + safeBasePath
   note. Still inside the acceptable band, but the prune pass toward the 15 KB
   budget keeps not happening (third session in a row flagging it).
4. **Coverage floor** — 75% floor + artifact exist in `ci.yml`; the floor has
   never actually been evaluated by a real runner run, and race-mode coverage
   may differ from the 76.9% baseline.
5. **Two frozen dead refs** in archived history (`docs/content-negotiation-design.md`,
   `docs/feedback/new/2026-08-09_consumer-reinvents-nonce-csp-system.md`) —
   targets never existed in-tree; left as frozen history (edit value ≈ 0).
6. **gopls stdversion warnings** persist (8 visible in editor diagnostics despite
   the committed `.vscode` env); real lint is clean. Routed to ROADMAP Theme 5.
7. **Feature-count stability** — corrected twice in 24h (154→166 by the sweep,
   166→165 by my test deletion). The counting command is documented in FEATURES,
   but nothing enforces it.

## c) NOT STARTED

1. **v0.6.0 release** — CHANGELOG `[Unreleased]` is release-ready; the cut is a
   TODO_LIST row (push → watch new CI jobs → re-head → tag → proxy-verify).
2. **GitHub Releases pages** for v0.2.0–v0.5.0 (only v0.1.0 has one).
3. **templ-components#6 PR** (StatCard `<dl>` fix + goldens) and **#7 PR**
   (busy-script nonce guard) — the latter gates any UI-dep re-land.
4. **UI-dep re-land** — the bumps failed twice today; they can re-land only after
   #7 ships AND the browser suite validates the new Datastar bundle.
5. **ROADMAP Theme 5** — deliberately raw ideas; nothing actioned (by design).
6. **Webhook hardening** (HMAC `WithWebhookSecret`, `"schema":1`) — Open Question
   awaiting appetite decision.
7. **A CI pin-guard** asserting the UI-dep versions until #7 lands (new idea from
   today's double collision — see f6).

## d) TOTALLY FUCKED UP

1. **I shipped a wrong FEATURES count in the same pass that caused it.** I wrote
   "166 functions" into FEATURES and *then* deleted the redundant test in the
   same session — the count was stale within an hour and only the final verify
   round caught it (`7f4763a`). Count-then-claim is a documented lesson I
   re-violated via sequencing, not ignorance.
2. **Fabricated line numbers in the 04-41 annotation run.** I derived "line
   numbers" for items 34–50 from a *relative* `sed` range instead of absolute
   `rg -n` output; the script aborted at line 168 (a section header). The atomic
   write prevented damage, but it was a pure citation≠verification failure —
   this repo documents that exact failure mode three times over.
3. **Scripted edits with from-memory anchors.** Two bulk-edit passes
   (AGENTS pin-gotcha, post-v040 f-intro) asserted on anchor strings I typed
   from memory; both failed because the real wrapping differed. Read-before-edit,
   violated through a python script instead of the edit tool.
4. **I introduced a broken link myself** — the integration-plan resolution note
   referenced the 11-23 status report with an `archived/` prefix it doesn't
   have. My own link sweep caught it; still, adding a lie while fixing docs is
   the failure mode this skill exists to prevent.
5. **Missed the concurrent session's re-bump until the end.** The second dep
   collision (`0862981`) landed mid-session and sat red until my final race-suite
   run caught it. The post-v0.4.0 retro literally documented this collision class
   ("missed the concurrent session's dependency bump until tests broke") — and it
   happened twice today.
6. **Wrote references before the move.** CHANGELOG/AGENTS/TODO_LIST pointed at
   the bisect audit's `archived/` path before the `git mv` existed. It worked
   because I held the ordering in my head — mv-first-then-reference is the safe
   order and I didn't use it.

## e) WHAT WE SHOULD IMPROVE

1. **Re-count after every deletion, in the same pass** — and make it structural:
   a CI drift-guard comparing the FEATURES count against `rg -c` would have
   caught both the sweep's 154 and my 166 automatically.
2. **Never build script anchors from memory** — re-view the target immediately
   before any scripted bulk edit, or just use the edit tool. The two failed
   asserts cost more than the "saved" tool calls.
3. **Re-run the race suite on any suspicion of external touching.** Concurrent
   sessions are real: two dependency collisions in one day is a pattern, not
   bad luck.
4. **Serialise sessions or declare ownership windows** (recurring retro ask,
   now with a second same-day incident). Alternatively: a CI guard that fails
   when `templ-components` ≠ v1.11.0 while #7 is open — mechanical enforcement
   of the AGENTS gotcha instead of tribal memory.
5. **`git mv` first, then write references** to the new path — ordering beats
   bookkeeping.
6. **Prune AGENTS.md in the same pass that grows it** (24.6 KB; the grow-and-prune
   rule exists and I didn't apply it).
7. **Keep-a-Changelog compare links + tag signing** — unchanged backlog, still
   cheap, still unrouted to a cycle.

## f) 50 THINGS TO GET DONE NEXT (brainstorm, impact-sorted)

_Most items below are already routed: **TODO_LIST.md** holds the short-term
commitments (v0.6.0 cut, GitHub Releases, upstream #6/#7 PRs); **ROADMAP.md
Themes 3–5** hold the rest. Canonical backlog lives there, not here. New items
from this session are marked 🆕._

**Release & CI (highest impact)**

1. Push master → watch the new CI jobs (version-guard, floor+artifact, concurrency, pins) on a real runner — step one of the v0.6.0 TODO row
2. Cut v0.6.0 (TODO_LIST)
3. Create GitHub Releases v0.2.0–v0.5.0 (TODO_LIST)
4. 🆕 CI pin-guard: fail builds while `templ-components` ≠ v1.11.0 (until #7 lands) — mechanical enforcement of today's lesson
5. 🆕 CI docs drift-guard: FEATURES test/function count vs `rg -c` recount
6. templ-components#7 PR: busy-script nonce guard → lift the pin → browser-validate the bundle (TODO_LIST)
7. templ-components#6 PR: StatCard `<dl>` fix + goldens → remove axe tolerance (TODO_LIST)
8. 🆕 Decide the daemon/collision policy: serialize sessions, ownership windows, or daemon debounce during releases (user infra)
9. Rehearse the fuzz issue-on-failure path with a deliberately failing run (ROADMAP)
10. `templ generate` drift check in CI (ROADMAP)
11. `actionlint` for both workflow files (ROADMAP)
12. `nix flake check` job in CI (ROADMAP)
13. Chrome/Chromium in the devShell so the browser suite runs locally (ROADMAP)
14. Coverage: push >80%, then raise the floor to 78% (ROADMAP)
15. Dependabot/renovate for GitHub Action SHA bumps (ROADMAP)

**Testing**

16. CSV exporter fuzz target (ROADMAP)
17. `RecommendedCSP` fuzz target (ROADMAP)
18. Webhook payload-marshal fuzz target (ROADMAP)
19. Keyboard-navigation a11y smoke in the browser suite (ROADMAP)
20. Metrics endpoint under strict CSP — browser test (ROADMAP)
21. Golden-screenshot diff test (ROADMAP)
22. Race-stress: 50 concurrent SSE clients vs `SubscriberCount` (ROADMAP)
23. Shutdown-ordering test for the example (ROADMAP)
24. Boundary tests: `WithTrend(1)`, CSV `q=0.8` negotiation, `WithBasePath` edges, retry large values (ROADMAP)
25. Webhook delivery-ordering test under concurrent transitions (ROADMAP)

**Features**

26. Webhook delivery metrics (`dashboard_webhook_deliveries_total{result}` + duration) (ROADMAP)
27. HMAC signing + payload `"schema":1` — needs the Open-Question decision (ROADMAP)
28. `WithGrouping(BySource)` per-service cards (ROADMAP)
29. Introspection endpoint for ops (ROADMAP)
30. Rate-limit 429 JSON body with Retry-After (ROADMAP)
31. Retry-After on 503s during the drain window (ROADMAP)
32. Embedded Datastar SDK serving helper (ROADMAP)
33. Aggregate + webhook demo modes in `nix run .#example` (ROADMAP)
34. Browser test: aggregate-rendered page under strict CSP (ROADMAP)
35. Load test: 20-source aggregate under concurrent SSE + scrape (ROADMAP)
36. PushOnChange with TTL (ROADMAP)
37. Timeline card: cap entries by age (ROADMAP)
38. Per-check latency histogram series (ROADMAP)
39. NDJSON export format (ROADMAP)
40. Public-mode leak-scanner test + optional JSON redaction (ROADMAP)

**Docs & process**

41. ADR: options/handlers/history split + sentinel family (ROADMAP)
42. `docs/release-checklist.md` (ROADMAP)
43. AGENTS.md prune pass toward the 15 KB budget 🆕 (flagged three sessions running)
44. Screenshot regenerate one-liner + light-screenshot caption (ROADMAP)
45. "Protect probes via network policy" README note (ROADMAP)
46. doc.go: webhook + public-mode combo example; `WithBasePath` example (ROADMAP)
47. `Routes()` accessor + `BasePath` resolved after all options run (ROADMAP)
48. Self-monitoring decision: Dashboard in its own health table? (ROADMAP)
49. gopls stdversion warnings investigation (ROADMAP)
50. Tag signing + Keep-a-Changelog compare links (ROADMAP)

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Release timing:** push master and cut v0.6.0 immediately so the new CI jobs
   (version-guard, coverage floor+artifact, concurrency, pins) prove themselves on
   a real runner — or batch more work first and keep the unverified-job risk
   open longer?
2. **Dependency-collision policy:** the same UI-dep sweep landed twice today and
   was reverted twice. Do you want (a) the CI pin-guard (fail while #7 is open),
   (b) serialized sessions / daemon disabled during release-and-docs windows,
   (c) both, or (d) accept the whack-a-mole?
3. **Which blocked decision should I tee up first for you:** build-tag gating
   (accept `GOEXPERIMENT=jsonv2` / fork go-sse / build-tag gate), fingerprint
   format stability, or webhook hardening (HMAC + schema version)?

---

_Markdown used per the user's explicit `.md` instruction (skill default is HTML —
override flagged, not propagated). Point-in-time snapshot; annotate, never
rewrite. Section (f) is already HARVEST-routed: TODO_LIST/ROADMAP are canonical._
