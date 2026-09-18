# Status — Federation Session (go-health `health/federation` + dashboard verification)

|            |                                                                                        |
| ---------- | -------------------------------------------------------------------------------------- |
| **Date**   | 2026-09-18 09:47 CEST                                                                  |
| **Scope**  | This session only: the "health.home.lan" federation request, end to end                |
| **Repos**  | `go-health` (implementation, ahead 12) · `go-health-dashboard` (verification + docs, ahead 5) |
| **Format** | Markdown per explicit user instruction — status-report skill's canonical HTML overridden for this report |

**One-line summary:** the federation spike is now real, tested code in go-health — a hub can
render N remote go-health instances as one dashboard with `dashboard.New(federation.New(remotes),
WithGrouping(GroupBySource))` — verified end to end, gates green, committed, **unreleased**
(v0.3.0 vehicle, blocked on tag+push authorization).

---

## a) FULLY DONE

1. **`health/federation` package implemented and committed** (go-health `d07952a`, bulk swept
   into daemon commit `30170bb`). `Remote{Name, URL}`, `New(remotes, opts...)` with sentinel
   validation (`ErrNoRemotes`/`ErrInvalidRemote`/`ErrInvalidTimeout`), `WithClient`,
   `WithTimeout` (default 5s), and `Prober`: merge-on-read `CachedResponse` (one parallel fetch
   per remote per read, ctx-plumbed, 1 MiB body cap, `Accept: application/json`), worst-of via
   new `Status.Rank()`, `name/check` namespacing, shutdown overlay, max latency, synthetic
   `name/reachable` FAIL row on any unreachable/non-200/undecodable/status-invalid remote,
   `RefreshInterval()=0`, one-way per-remote startup latches, the three kubelet handlers
   (liveness fetch-free, readiness 503 on merged fail, startup fetches + names the dark
   remotes), `RegisterRoutes`, marshal seam. Evidence: `nix run .#gates` **all gates green**;
   tests pass plain and with `-race`.
2. **Untrusted-input hardening**: documents missing a status or carrying a non-pass/warn/fail
   check status are refused whole (would render as healthy downstream); pinned by test and by
   the fuzz invariant.
3. **Design note** `docs/federation-design.md` (go-health): topology, pull-not-push rationale,
   merge table, startup semantics, API, non-goals, alternatives rejected.
4. **`Status.Rank()` added to root** (fail 0 < warn 1 < pass 2, unknown ranks as pass), tested
   (`TestStatus_Rank`, commit `77dd487`), aggregate's private `statusRank` now delegates — one
   severity ordering for the module.
5. **Test suite**: 16 test functions + fuzz target. Includes the e2e wire-contract test (a real
   `NewWithHealthCheck` probe with background refresh serves its readiness handler; federated
   view preserves namespacing, statuses, errors, and `Since` verbatim), validation table,
   merge table, scalars-non-survival, wire fields decode (`since`, `duration_ns`), fetch
   timeout, concurrent reads (fetch-per-read contract counted), latch one-way semantics, four
   handler tests, `RegisterRoutes` smoke, marshal-error seam. Fuzz: 1.94M execs, 0 failures.
6. **23 lint findings → 0** (first `nix run .#lint` on the draft; fixed ctx plumbing
   (contextcheck — handlers now pass request ctx so disconnected callers cancel fetches),
   `WaitGroup.Go` modernization, perfsprint, errorlint, gocognit/cyclop splits, varnamelen,
   wsl, gci).
7. **Full-stack verification in the dashboard repo**: temporary `-replace ../go-health` +
   throwaway test: two real probes over HTTP → federation → `dashboard.New(fed,
   WithGrouping(GroupBySource))` → HTML contains `core/postgres`, `ghost/reachable`,
   "connection refused"; JSON negotiation returns 503 while a remote is dark. **PASS**, then
   every trace reverted (go.mod restored, test trashed, `git status` clean, build green).
8. **Docs updated in both repos**: go-health CHANGELOG `[Unreleased]` (federation + public
   `Status.Rank`), FEATURES (federation row, Rank note), AGENTS (package list + full federation
   paragraph) — commit `6c8eaff`; dashboard ROADMAP §2 (federation marked shipped-in-go-health
   with the hub recipe + remaining share) and AGENTS dependency notes — commit `1b2de52`.
9. **Parallel-session hygiene held**: a second session was active in go-health (benchmarks,
   etag-rejection design, ADR-005, OpenAPI 0.2.0, property tests). Their files were never
   touched; their exhaustive-linter adjustment to my `Rank` preserved semantics (my unknown-status
   test rows still pass).

## b) PARTIALLY DONE

1. **End-to-end verification is proven but not enshrined.** The dashboard-side proof was a
   throwaway test (deleted after PASS because it cannot compile against released go-health
   v0.2.0). A skeptic cannot re-run it today. What remains: the permanent integration test +
   compile-time assertion, blocked only on the v0.3.0 tag. Effort: S once released.
2. **Commit history is fragmented by the daemon.** Intent commits sit at tip (`d07952a`,
   `6c8eaff` in go-health; `1b2de52` in dashboard), but the bulk of federation code lives in
   heuristic auto-commits (`30170bb`, `4351540`, `b0bc4ca`, `e355e3d`) that raced the per-batch
   commits. Findable, not clean; folded forward per the release discipline.
3. **Docs coverage is asymmetric**: AGENTS/CHANGELOG/FEATURES/ROADMAP updated; the **OpenAPI
   spec**, **README** ("Which probe should I hit?" table), go-health **TODO_LIST**, and
   go-health's own docs/status are **not** updated for federation.
4. **health.home.lan itself**: the recipe is verified, but no runnable hub artifact exists —
   no example binary, no compose/NixOS service, no deployment. Blocked on release + your
   deployment-target answers.
5. **Benchmark coverage**: aggregate has benchmarks + FEATURES baselines (the parallel session
   added more mid-session); federation has none (network-bound, but a local-httptest benchmark
   would pin the merge/fan-out cost shape).

## c) NOT STARTED

All still wanted unless marked otherwise:

1. **Release go-health v0.3.0** (tag + push + proxy verification) — blocked on owner
   authorization AND on scoping against the parallel session's in-flight v0.3.0 prep.
2. Dashboard `go.mod` bump to v0.3.0 + full dashboard suite re-run (incl. browser CSP suite).
3. Permanent federation→dashboard integration test + `var _ dashboard.Prober =
   (*federation.Prober)(nil)` assertion in the dashboard repo.
4. Runnable federation example (`example/federation` + nix app): two mock upstreams + hub.
5. OpenAPI spec coverage for the federation handlers; README row; TODO_LIST rows.
6. Federation benchmark + FEATURES baseline row.
7. CI fuzz-enumeration check: confirm `FuzzCachedResponse_ArbitraryRemoteBodies` is picked up
   by go-health's fuzz app/weekly CI budget (dashboard's fuzz.yml enumerates targets explicitly;
   go-health's enumeration I did not verify this session).
8. M54 since-fed timeline, M55 stable-group collapse, M56 per-source staleness (designed
   2026-09-17, not implemented; M56 composes with federation's reachable rows).
9. `Remote.Headers` (auth) and `WithCacheTTL` — deliberate v1 non-goals, on-demand only.
10. Service grouping by tags/labels; `Register` auto-start (ROADMAP §2 remainder).
11. The actual hub deployment (host, TLS, process supervision).

## d) TOTALLY FUCKED UP

Radical honesty; nothing here is currently broken in the shipped tree — these are things I
broke or got wrong **during** the session and what state they left behind:

1. **I introduced a production validation bug mid-session**: the constructor rewrite dropped the
   slash-in-name rejection from the documented aggregate-name contract — `New` would have
   accepted `Name: "a/b"`, blurring the grouping axis. **Caught by my own validation-table test
   before any commit; fixed and pinned.** Root cause: hand-restructuring the validated
   constructor instead of mechanically carrying each contract item across the rewrite. Lesson
   recorded in (e).
2. **First fuzz-target design was wrong**: one `httptest.NewServer` per fuzz iteration exhausted
   listeners → a bogus "crasher" artifact and a wasted debug cycle (plus a `testdata` file I had
   to trash). Rewritten to one shared server; campaign then ran 1.9M execs clean. Root cause:
   wrote the fuzz before studying `aggregate_fuzz_test.go` closely.
3. **Two avoidable full-file rewrites**: 23 lint findings arrived only after tests passed,
   including a real quality bug (fetches ran on `context.Background()`; handlers couldn't cancel
   on client disconnect) rather than style. `nix run .#lint` on the first scaffold would have
   shrunk this to one pass.
4. **Three attempts to get the e2e `Since` test right**: I compared against a live-mode probe's
   cache (which never populates) instead of reading the live-path semantics first, then mis-designed
   the latch test (a never-answering second remote made the one-way assertion unreachable).
   Two burned cycles on guessing instead of reading `handlers.go`/probe semantics.
5. **Stale LSP diagnostics polluting the dashboard repo all session**: gopls still reports
   `federation_tmp_test.go: could not import federation` for a file that was trashed and
   verified absent (`git status` clean, `go build` green). Known LSP-cache lie; I never ran
   `lsp_restart` to clear it. Harmless, noisy, and it contradicts the "trust the CLI" rule
   every single tool output.
6. **Oversell risk on "kinda fully automatically"**: federation is zero-config *for the
   remotes*; the hub is still a hand-maintained URL list — no discovery (mDNS, Docker labels,
   Tailscale API) and no push registration. The design docs are honest about this; spoken
   summaries should be too.
7. **The session's central proof is not reproducible by others yet** (see b1): "verified end to
   end" is true of this session's run, but the enshrined test, the released dependency, and the
   runnable example do not exist. Until v0.3.0 ships and the dashboard bumps, federation is a
   well-tested unreleased capability, not a user-facing feature.

## e) WHAT WE SHOULD IMPROVE

1. **Lint the scaffold, not the finished batch**: run `nix run .#lint` on a new package's first
   skeleton (before writing the full test suite). This session's 23 findings included one real
   bug class (ctx plumbing) that earlier feedback would have surfaced before two rewrites.
2. **Contract checklists for validated constructors**: when restructuring a constructor with
   sentinel validations, enumerate the sentinels first and tick them off mechanically. The
   slash-rule drop happened exactly because the rewrite was by hand.
3. **Read house patterns for EVERY test genre before writing it**: unit style I matched;
   fuzz infra and example-Output conventions I learned by failing (server-per-iteration;
   non-hermetic example). The patterns existed in `aggregate_fuzz_test.go` /
   `aggregate/example_test.go`.
4. **Probe-mode semantics before test design**: live-mode probes never populate `CachedResponse`;
   background-refresh + `Start` is the realistic remote shape. Reading beats guessing (two
   wasted iterations).
5. **Split-brain watch on the Prober shape**: `federation_test.go` asserts conformance via a
   *copy* of the dashboard's five-method interface (import would be a cycle). If
   `dashboard.Prober` ever grows, the copy silently diverges. Fix is one line in the dashboard
   once v0.3.0 lands — a real `dashboard.Prober` assertion (follow-up #5).
6. **Shared release vehicles need explicit coordination**: v0.3.0 is now claimed by my
   federation work AND the parallel session's in-flight prep (aggregate `Healthz` design
   "deferred to v0.3.0", etag-rejection design, ADR-005, OpenAPI 0.2.0). Decide scope before
   anyone tags.
7. **Two-writers-one-tree choreography**: with a parallel session active, per-batch commits
   still got swept; commit per FILE at the moment it's verified, and re-run `git status` inside
   every commit chain (the discipline worked — nothing was lost — but the history smells).
8. **Clear LSP caches immediately** after trashing files (`lsp_restart`), or stale diagnostics
   erode trust in every later tool output.
9. **Composition gap worth a design decision**: `aggregate.Source` takes `*health.Probe`
   concretely, so a hub that wants M local probes + N remote instances today has no clean merge
   (federation only fetches URLs; aggregate only takes probes). Either document the
   boundary honestly or add a response-source seam later. Also: hub-of-hubs chaining *should*
   work naturally (a hub's readiness serves a merged document) — prove it with one test.
10. **Federation error text flows into consumer surfaces** (metrics labels, webhook payloads,
    HTML). Local errors were already escaped/masked; remote-derived error strings are untrusted
    input reaching those same seams — the dashboard's escaping exists, but a federation-fed test
    of each seam is missing.

## f) TOP 50 NEXT TASKS (impact-ranked; HARVEST food — route into TODO_LIST/ROADMAP, don't entomb)

| #  | Task                                                                                                        | Impact   | Effort | Category      |
| -- | ----------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Decide v0.3.0 scope with the parallel session (federation vs Healthz/etag/ADR-005 in-flight)                | Critical | S      | Process       |
| 2  | Release go-health v0.3.0 (tag → push tag → push master → `scripts`-style proxy/sumdb verification)          | Critical | S      | Release       |
| 3  | Dashboard: bump go.mod to go-health v0.3.0                                                                  | Critical | S      | Feature       |
| 4  | Dashboard: permanent federation→dashboard integration test (replaces deleted throwaway)                     | High     | S      | Testing       |
| 5  | Dashboard: compile-time `dashboard.Prober` assertion for `*federation.Prober` (kills the interface-copy split brain) | High | S      | Quality       |
| 6  | Dashboard: runnable federation example (2 mock upstreams + hub) + `.#example-federation` nix app            | High     | M      | Feature       |
| 7  | Push both repos at the release milestone (go-health ahead 12, dashboard ahead 5 — CI is blind until then)   | Critical | S      | Release       |
| 8  | Run FULL dashboard suite (incl. browser/CSP) against v0.3.0 after the bump                                  | High     | M      | Testing       |
| 9  | Federation-fed test of the metrics seam: untrusted remote error strings inside metric label escaping       | High     | S      | Testing       |
| 10 | go-health: add federation handlers to `docs/openapi.yaml` (reachable rows, zeroed scalars)                  | Medium   | S      | Documentation |
| 11 | go-health: README row/paragraph for federation ("Which probe should I hit?" table + quick start)            | Medium   | S      | Documentation |
| 12 | go-health: federation benchmark + FEATURES baseline row                                                     | Medium   | M      | Quality       |
| 13 | go-health: verify CI fuzz enumeration includes the new fuzz target                                          | Medium   | S      | Testing       |
| 14 | go-health: TODO_LIST rows for federation follow-ups                                                         | Medium   | S      | Documentation |
| 15 | go-health: DOMAIN_LANGUAGE.md terms (hub, remote, reachable check, merge-on-read)                           | Medium   | S      | Documentation |
| 16 | go-health: hub-of-hubs chaining test (hub's readiness as another hub's remote)                              | Medium   | S      | Testing       |
| 17 | Design decision: merging local probes + remotes in ONE hub (aggregate/federation composition seam)          | Medium   | M      | Feature       |
| 18 | Measure default interplay: 5s fetch timeout vs 2s default push cadence (slow remote delays ticks)           | Medium   | S      | Quality       |
| 19 | Dashboard: verify evidence strip counts `name/reachable` rows as non-pass observations (rendered run)       | Medium   | S      | Testing       |
| 20 | Dashboard: verify webhook + PushOnChange fingerprint over federation transitions (reachable flip)           | Medium   | M      | Testing       |
| 21 | Dashboard: verify public mode masks namespaced remote names/errors as expected                              | Medium   | S      | Testing       |
| 22 | Dashboard: SSE patch stream test over a federation-fed dashboard                                            | Medium   | S      | Testing       |
| 23 | Dashboard: trend/export (CSV/NDJSON) sanity with federation values                                          | Low      | S      | Testing       |
| 24 | Implement M56 per-source staleness ("stale?" marker + `dashboard_health_source_stale` gauge)                | High     | L      | Feature       |
| 25 | Implement M54 since-fed timeline seeding                                                                    | Medium   | M      | Feature       |
| 26 | Implement M55 stable-group collapse ("stable for 6h")                                                       | Medium   | M      | Feature       |
| 27 | Deploy health.home.lan hub (host, TLS, supervision)                                                         | Critical | L      | Feature       |
| 28 | NixOS module for the hub (remotes list → systemd unit)                                                      | High     | M      | Feature       |
| 29 | Compose service for the hub next to the existing Grafana wiring in the dashboard repo                       | Medium   | S      | Feature       |
| 30 | Grafana: scrape the hub's `/health/metrics`                                                                 | Medium   | S      | Feature       |
| 31 | Webhook receiver consuming hub-side transitions (alerting path)                                             | Medium   | M      | Feature       |
| 32 | Decide hub exposure policy (health data disclosure on LAN; auth/proxy posture)                              | Medium   | S      | Decision      |
| 33 | `Remote.Headers` (bearer/basic) when a real service needs auth                                              | Medium   | S      | Feature       |
| 34 | `WithCacheTTL` only on a concrete demand signal (revisit documented non-goal)                               | Low      | M      | Feature       |
| 35 | Per-remote last-error accessor (observability without synthesized rows)                                     | Low      | S      | Feature       |
| 36 | Public `CachedResponseContext(ctx)` variant (handlers already ctx-plumbed internally)                       | Low      | S      | Feature       |
| 37 | Naming poll: is `name/reachable` the right synthesized check name? (alternative: `upstream`)                | Low      | S      | Decision      |
| 38 | go-health ROADMAP: record push-based registration as an explicitly rejected non-goal                        | Low      | S      | Documentation |
| 39 | Commit a few discovered fuzz corpus entries as seeds                                                        | Low      | S      | Testing       |
| 40 | Root `doc.go`: link the federation package so pkg.go.dev surfaces it                                        | Medium   | S      | Documentation |
| 41 | `lsp_restart` in the dashboard repo to clear the stale `federation_tmp_test.go` diagnostic                  | Low      | S      | Cleanup       |
| 42 | Status report for the go-health repo's own `docs/status/` (its convention; this one is dashboard-scoped)    | Medium   | S      | Documentation |
| 43 | HARVEST this section into TODO_LIST.md/ROADMAP.md (docs-health) so it doesn't die in this file              | High     | S      | Documentation |
| 44 | GroupBySource card headers could surface remote freshness metadata (needs design)                           | Low      | M      | Feature       |
| 45 | Many-remote aesthetics: healthy-group collapse threshold under 20+ remotes                                  | Low      | S      | Quality       |
| 46 | Startup-latch thread-safety test with parallel handler storms under `-race`                                 | Low      | S      | Testing       |
| 47 | Document observed flap behavior (no-retry policy) in the design note's operational notes                    | Low      | S      | Documentation |
| 48 | Record the "two-writers shared release vehicle" trap in go-health AGENTS release section if absent          | Low      | S      | Documentation |
| 49 | Hosted-hub screenshot for README/website once health.home.lan is live                                       | Low      | M      | Documentation |
| 50 | Retire this session's follow-ups from AGENTS once v0.3.0 ships (docs-health VERIFY/ANNOTATE)                | Low      | S      | Documentation |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Release scoping:** Do you want go-health v0.3.0 tagged and pushed NOW with federation only,
   or held until the parallel session's in-flight v0.3.0 items (aggregate `Healthz`, etag
   design, ADR-005 follow-ups) land in the same vehicle? (I can see the collision in their
   CHANGELOG/design-note entries; only you can sequence it.)
2. **Deployment target:** Where will `health.home.lan` run — which host/runtime (NixOS box with
   a module? the dashboard repo's compose stack next to Grafana?) and roughly how many services?
   This shapes the example, the NixOS module, and the TLS/auth posture.
3. **Remote auth reality:** Do any of the services that should federate sit behind auth (bearer
   token, basic auth) or expose health on non-standard paths? That decides whether
   `Remote.Headers` must jump the v1 non-goal line before federation is actually useful to you.

---

*Point-in-time snapshot; goes stale. Section (f) is HARVEST input (docs-health), not a
commitment list. Nothing is pushed; both repos are ahead of origin pending the release decision.*
