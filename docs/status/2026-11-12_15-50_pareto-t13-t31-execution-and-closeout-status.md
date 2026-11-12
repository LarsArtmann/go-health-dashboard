# Status Report — Pareto T13–T31 execution, closeout, and the CI-truth reconciliation

**Date:** 2026-11-12 15:50
**Session:** continuation of the 12-52 run — plan tasks T13–T31, final battery, and the CI-red discovery + fix it triggered
**Reporter:** Crush (glm-5.3-flash)
**Tree:** `7bb61fb` (CI **green** on master; the red window `ab477dd` lasted ~40 min and is fixed)

> Scope line: this report covers everything AFTER the 12-52 report. The plan
> (T01–T31 / F01–F138) is now EXECUTED THROUGH THE END — every T-task has a
> commit or a documented disposition. The headline events of this window:
> (1) the whole engine tier shipped (M12–M21, M30), (2) both decision
> batches were written and parked, (3) the closeout battery caught three
> real drifts (SECURITY.md footer, FEATURES count, a broken art-dupl gate),
> (4) my own §f annotation botched 40 items, got daemon-committed, and was
> reverted + redone scoped, and (5) the pushed stack sat on a red CI
> (templ drift) for ~40 minutes because I did not watch CI after pushing —
> the exact doctrine violation the release-discipline rules exist for.

## a) FULLY DONE

1. **T10 finished** (`fb042e2`): the annotate gate's REAL vocabulary is
   `done|resolved|superseded|fixed|adopted|decided|verified|moot|no longer` —
   `closed`, `routed`, and `answered` are NOT in it; 46 routed-verdicts + 6
   answered-verdicts converted; 14-19's 25 §f table rows struck per-cell (the
   established table style); the 05-27 self-contradiction repaired; one
   verdict citing a non-existent path repinned; battery 5/5; manifest honest.
2. **T13 CI ergonomics** (`2faa1df` + daemon `eccb35d`): red-alert bodies
   resolve the failed JOB names inline (`gh run view --json jobs` — verified
   against a real historical red run); the Hygiene drift check ships its diff
   as an artifact; test/test-race print a loud browser-suite RAN/SKIPPED
   verdict (both branches verified; `.build` deliberately carries none).
3. **T14 helper polish** (`f75506e`, relabeled daemon `07b5d24`):
   `browserPillStateJS` extracted as the ONE computed-style pill probe, and
   the strict-CSP test's live-pill wait moved onto it — it previously
   checked only `.hidden`, which is exactly the signal the #31 bug defeats;
   a #31-class regression now fails both tests. LSP disposition: the templ
   LSP never started this session and the system-templ skew healed 2026-11-05
   — nothing to restart; the ROADMAP note already existed.
4. **T15 audit drain** (`d11edbe` + daemon `99ecf1f`): the biggest stale
   cluster (11 items, 2026-08-08 docs-health audit) annotated with
   CURRENT-tree evidence, each claim verified before striking (gosec app
   exists, coverage 80.3% run, screenshots embedded + test-captured, metrics
   shipped, public mode shipped, dependabot ships); two honest harvests into
   ROADMAP (coverage badge, OG metadata); two documented non-goals
   (in-library auth, PDF); `audit_max` 87→76 with TODO/CHANGELOG synced.
5. **T16 M13** (`6f9a539` + daemon `450f9d4`/`e10af6f`/`9ca7fe1`):
   `dashboard_sse_connections_{opened,closed}_total` +
   `dashboard_pusher_last_tick_seconds` (absent before first tick — absent
   never reads as 0s-fresh), accessors nil-safe, e2e test over a REAL SSE
   connection (connect → opened=1/closed=0, disconnect → closed=1),
   docs/metrics.md + README with alerting guidance.
6. **T17 M14+M15** (`9d83f36` + daemon `4aa0a8b`/`4a0cf1f`/`aaa92e3`):
   `dashboard_ratelimit_429_total` + Retry-After histogram (ONE retry value
   per rejection feeding header, JSON body, and metrics — the old code
   called retryAfter() twice), ±10% lifetime jitter (`jitterLifetime`, range
   + spread tested), Retry-After on connection-limit 503s (retry interval,
   push-interval fallback), lint findings fixed (magic 10 → constant, gosec
   nolint with reason).
7. **T18 M20+M24** (`70aa379` relabeled from `3a64191`): shellcheck +
   actionlint in the devShell, two SC2001 style findings fixed in
   check-changelog.sh, and `scripts/test-cold-cache-pins.sh` — which caught
   a REAL guard gap on day one (the version query died on a raw tool error
   under set -e before any ::error::; now tolerant + loud). Mutation-checked
   both ways; discovered+documented that `GOPRIVATE=github.com/larsartmann/*`
   makes `GOPROXY=off` fall back to direct git VCS, so the offline leg needs
   `GOVCS='*:off'` to stay honest. Pre-push gates 13/14 + CI Hygiene parity.
8. **T19 M21–M23** (`062a332`): markdownlint-cli in devShell + a two-pass
   gate (`check-markdown.sh`: live docs strict, archived dirs relaxed —
   markdownlint-cli resolves config from the invocation dir only, so the
   relaxation is a second pass with -c). Real findings fixed: hard tabs,
   language-less fences, a prose "#1)." parsed as a heading, the manifest's
   second H1, double blanks — and TWO genuine CHANGELOG bugs (duplicate
   `### Added` under [0.11.0], duplicate `### Changed` under [0.10.2]),
   merged with content preserved. M22 resolved as deliberate non-adoption
   (interrogate is a PYTHON docstring tool — verified via gh search); M23
   baseline pinned (gotcha 26: failed_only_empty rejection); pre-push
   gate 15 + CI parity. Battery + changelog lint green after.
9. **T20 M18+M19** (`f7489b5`): compose-build CI job (build-only, the
   decision-safe slice), `lockBrowserHost` cross-process flock with every
   release path covered, contention proven two ways (open-file-description
   refusal + a live two-process smoke where wall time < serial sum because
   only browser sessions serialize), boot-smoke CI design doc (three tiers,
   parked on A4).
10. **T21 M16+M17** (`6315668` + daemon `bb9f0ee`/`24c66de`):
    `WithPerIPRateLimit` (per-address buckets, RemoteAddr-keyed only —
    proxy headers deliberately ignored, sweep at 4096 buckets, probes
    exempt, same 429 shape + metrics family) and `WithRequestLogging`
    (opt-in slog, status recorder with SSE flush forwarded, probes
    unlogged, no query strings/headers/check names in lines). Tests:
    address isolation, sweep bound, probe exemption, log content incl.
    rejection status, default-off, flush passthrough. Full suite + race
    subset green; lint 0.
11. **T22 M12 chaos-load** (`a0d158a` + daemon `3807677`): stress-ng 30 of
    32 cores: full race soak `-count=2` green in 96s, the four timing-margin
    tests green in 11s, the browser trio green in 8.4s — the #32 margin
    fixes are now PROVEN under deliberate CI-class load
    (docs/research/2026-09-10_benchmarks.md).
12. **T23 M30 watchdog hardening** (same batch):
    `docs/design/2026-11-12_watchdog-security-audit.md` — workflow_run
    cannot be triggered by fork activity (branch scoping), least-privilege
    token, pinned checkout, no injection surface; self-watch verified
    (Master Red Alert watches itself); the drift guard gained duplicate-entry
    and name-collision failure modes, BOTH mutation-checked (a shadow "CI"
    workflow file was previously invisible to both guard views — silent
    overwrite in `declare -A`).
13. **T24+T25 decision batches** (`4edfa38` + daemon `f4d818e`/`8fa1d09`):
    `docs/decisions/2026-11-12_decision-batch-a.md` (six calls + §g1–§g3
    appended) and `-batch-b.md` (eight calls) — options, recommendations,
    blast radius each; nine Blocked rows now cite their batch entry. PARKED
    by plan design.
14. **T26 upstream re-verification** (daemon `623f727` + `fefc542`):
    templ gofumpt draft STALE (gofumpt -l clean on v0.3.1100 — does not
    reproduce); erraudit draft VALID (reproduced on 5c841cf: `./...` → zero,
    `.` → 4 NEEDED); StatCard contrast REFUTED by computation
    (text-green-600 on green-50 = 3.15:1, on white = 3.30:1 — PASSES the
    applicable 1.4.11 non-text floor for a decorative aria-hidden icon).
    Nothing filed; both Blocked rows carry their filing state.
15. **T27** (`2321170`): templ-components#37 CLOSED but the fix commit
    (41321644) is UNRELEASED — v1.27.0 (our pin) predates
    `ListNoteProps.Message`, adoption deferred with disposition; the
    copy-affordance clash is RESOLVED at the current pin (singleton
    `window.tcCopyAttached` script; `utils.ScriptComponent` omits the nonce
    attribute when none resolves — verified in the module cache source).
16. **T28 fleet riders** (`f52106e`): dnsblockd UNBLOCKED — `static/v0.7.0`
    now exists alongside root v0.7.0 (ls-remote), so their exact-match
    lockstep gate is reachable via a lockstep bump, not a tag push; CV
    bump-PR scope drafted on its row (deploy stays the user's); per-consumer
    tc doctor sketch recorded (B7).
17. **T29 go-etag port** (`4a7f8b9`; go-etag `9f6f71e`): both fragile chains
    fixed (awk filter — no match-exit semantics to swallow; unresolvable pin
    count fails loudly before the integer comparison); shellcheck + syntax
    clean, their CI green on master, row struck.
18. **T30** (`35f5c44`): AGENTS restructure outline
    (`docs/decisions/2026-11-12_agents-restructure-outline.md`) — stays /
    moves / mechanics / explicit non-moves; target ≈260 lines from 553;
    executes on B8 sign-off.
19. **T31** (`27f9db1` + daemon `002dd19`): BuildFlow pin re-checks on both
    skip rows (the 8936ca0 binary references go-structure-linter project
    config — the unskip condition MAY be met; templ-generate's nix-race
    condition unchanged); relabel-daemon standing pattern = gotcha 27 with
    this session's two worked examples; boot-verify container-runtime
    dependency note in deploy/README.md.
20. **Closeout battery caught three real drifts, all fixed**: SECURITY.md
    footer (predated v0.13.3 — refreshed), FEATURES count drift
    (359/50 → 371/51 after the session's +12 test funcs +1 file), and the
    art-dupl gate — see §d3. Battery exit 0; docs-gates 5/5; changelog lint
    green; markdown green.
21. **CI-truth reconciliation** (`7bb61fb`): the pushed stack had been red
    on Hygiene (templ drift — see §d4) for ~40 minutes; reproduced locally,
    canonical generate → fmt committed with SWEEP_OK, CI green on
    `7bb61fb`, Master Red Alert success on the same run (the watchdog
    works).
22. **Session report §f honest closure**: items 1-20 + 26-30 struck with
    commit-citing verdicts, items 21-25 left open (nixpkgs bump; the four
    user-gated calls) — scoped to §f only after the §d1 botch was reverted.

## b) PARTIALLY DONE

1. **CI verification of the pushed stack.** I pushed ~15 commits across the
   window and watched CI only at the very end — discovering Hygiene red
   since `ab477dd` (templ drift) and fixing it at `7bb61fb` (now green).
   One green CI run exists on the final tree; the Nightly Flake Canary and
   Nightly Fuzz have not yet run against it. The four NEW CI steps
   (compose-build, shellcheck step, markdownlint step, cold-cache legs)
   have now passed once inside the green Hygiene run — but they carry
   unpinned supply-chain surface (apt-get/npm install) that deserves the
   pinning pass (§f).
2. **The closeout battery ran with two gates SKIPPED.** I invoked
   `bash scripts/pre-push-checks.sh` from the ambient shell, so gates 13
   (shellcheck) and 15 (markdownlint) took their designed skip path
   ("not on PATH"). Each gate is verified green via devShell invocations
   individually, but the closeout "all green" was not a single
   full-set-in-devShell run. The next battery should run as
   `nix develop -c bash scripts/pre-push-checks.sh`.
3. **Full browser suite on the FINAL Go tree.** T20 ran the full browser
   suite (22/22, 25.7s) after the flock change; T21 then changed the SSE
   serving path (statusRecorder wrapping) and was verified with the
   in-process SSE integration tests + the flush-passthrough test — the
   real-Chrome suite has not run against the T21 tree. The final Go tree
   equals the T21 tree (T22–T31 were docs/scripts only), so this is
   insurance, not a known gap.
4. **ROADMAP/master-plan shipped-annotations.** The four T17/T21 features
   (per-IP limits, request logging, lifetime jitter, Retry-After on 503s)
   and the T16 counters/gauge still sit in ROADMAP as "harvested 2026-11-04"
   (queued) — they are now implemented in `[Unreleased]`. Only M22/M23/M24
   got their master-plan annotations (T19); M12–M21 and M30 shipped today
   without their rows being annotated. Both annotation passes are queued
   (§f).
5. **CHANGELOG completeness for the engine tier.** The `[Unreleased]`
   Added block now carries the five feature entries (written at closeout,
   after the fact — the same shape of debt as §g1, though never released).
   Ideally each feature change writes its entry in the same change.

## c) NOT STARTED

1. **Upstream filings execution** (T26's second half): filing the ONE valid
   draft (erraudit nolint-audit, own repo) remains owner-gated and undone.
2. **AGENTS restructure execution** (T30's second half): the outline is
   ready; the cut awaits the B8 sign-off.
3. **v0.14.0 cut**: `[Unreleased]` now carries five real Added entries
   (engine tier) — no cut planned yet (rides A1 + your call).
4. **Tier-1 hub boot-smoke CI** (designed, not built) and **Tier 2** (parked
   on A4).
5. **nixpkgs `go_1_27` bump execution** (GO-2026-6859; row 🟡, needs a
   quiet-daemon window + vendorHash verification).
6. **ListNote adoption** (deferred — waiting for a templ-components release
   carrying 41321644, then the pin-guard ceremony).
7. **Sibling riders execution**: dnsblockd lockstep bump PR, CV bump PR
   (scope drafted, deploy is yours), the Playwright-Chromium-on-NixOS shim
   (parked with §g3).

## d) TOTALLY FUCKED UP

1. **The §f annotation botch (40 items, wrong, daemon-committed).** My
   first annotation pass used an unscoped regex — it matched numbered
   items in EVERY section (§a narrative, §b OPEN user-gated items, §d
   lessons) and applied §f's verdicts off-by-one; the daemon committed the
   wreckage as `71c2777` before I looked. Reverted to the pre-botch
   content and redone with line-range scoping (verified: 25 strikes inside
   §f, zero outside). Root cause: a global regex where a section-bounded
   loop was called for, plus not checking `git status` between the write
   and the walk-away. Lesson recorded here: any scripted multi-edit must
   bound itself to the section it claims, and the FIRST action after a
   scripted doc edit is `git diff --stat` + section audit — the daemon's
   snapshot latency is ~one minute.
2. **The art-dupl gate shipped broken in the release train and stayed
   broken all day.** Two compounding defects: the native-baseline
   migration recorded an EMPTY baseline (`entries: []`), so the DOCUMENTED
   accepted pair tripped `check` as a "new" clone — and the gate's
   `run_tool` invocation let set -e abort the whole battery BEFORE
   printing the findings. Net: a gate that was green when the tree was
   clean, and aborted instead of reporting when it wasn't — the worst
   possible failure mode for a gate. Discovered only when the closeout
   battery hit it (exit 2 with no output). Fixed both: baseline
   re-recorded with the one documented group (hash `431a125136df2408`,
   view.templ/status.go display.BadgeType pair — the same deliberate
   duplication as before), and the gate captures the exit code. Root
   cause: the migration was verified against a CLEAN tree only — gates
   need mutation checks (a synthetic new clone) exactly like tests do;
   the AGENTS bullet said "mutation-checked" for the migration, but the
   mutation checked detection, not the gate wiring.
3. **The pushed stack sat on red CI for ~40 minutes.** The templ drift
   (daemon captured a raw-shaped regen between generate and fmt — the
   known race, documented in AGENTS as "nix fmt before leaving the tree,
   every time") was pushed in `ab477dd` and every earlier push in the
   stack likely carried it; I did not run `gh run list` after ANY push,
   violating the release-discipline doctrine I helped write ("BEFORE
   pushing, confirm remote CI state"). Found it only when writing THIS
   report's opening checks — the T13 red-alert naming made diagnosis a
   30-second lookup, which is the one part that worked as designed.
4. **The closeout "all green" claim was softer than it sounded** (two
   skipped gates — §b2). The skips are the designed honest behavior, but
   presenting a closeout green with unexplained skips would have been a
   truthfulness miss; it is disclosed here and queued as §f1.

## e) WHAT WE SHOULD IMPROVE

1. **Section-scoped edit discipline**: scripted doc edits must bound
   themselves to the section they claim (line-range loops, not global
   regexes), and the first post-edit action is a section audit — today's
   §d1 cost a revert cycle.
2. **Mutation-check GATES, not only tests**: every gate change gets a
   synthetic red (a fixture that MUST fail) run through the real gate
   invocation — §d2's broken gate would have been caught in the same
   change.
3. **Closeout batteries run inside the devShell** (no skip paths), and the
   report that claims green says which battery it ran.
4. **Watch CI after every push** — the doctrine exists in AGENTS
   release-discipline rule (0); the honest upgrade is: the session that
   pushes runs `gh run list` before starting the next task, not at
   closeout.
5. **Pin the new CI steps' supply chain**: `npm install -g markdownlint-cli`
   and apt-get installs are unpinned; version-pin or vendor per the fleet's
   SHA-pinned action convention.
6. **Commit intent content BEFORE large scripted edits** when the daemon is
   active — the §d1 wreckage got captured in under a minute; a
   pre-edit `git add` + intent commit would have made the revert trivial.
7. **Write ROADMAP/master-plan shipped-annotations as part of each feature
   change** (the "harvested → shipped" flip), not in a later docs pass —
   §b4 is the accumulated interest.
8. **Replace the new 1.3s sleep test** (TestMetrics_RateLimitFamily refill
   wait) with clock injection when the fixture machinery next changes —
   the budget is the proven one, but sleeps are the flake class we keep
   hardening against.
9. **Split metrics_test.go** (~750 lines after the session) into
   rate-limit/lifecycle/watchdog files along the internal/external package
   seam that already exists.
10. **Document the SSE logging semantics** (one line per CONNECTION at
    close, not per patch) in the README options text — the current wording
    says "per request", which is true but easy to misread for streaming
    routes.

## f) Up to 50 things we should get done next (ranked)

1. **Run the full battery inside the devShell** (`nix develop -c bash
   scripts/pre-push-checks.sh`) so the closeout claim is a no-skip green
   (~2 min).
2. **Keep watching CI on `7bb61fb`** and any follow-up pushes; one green
   Hygiene run exists — the nightly canary + fuzz should confirm it
   overnight.
3. **Run the full browser suite on the final tree** (T21 recorder path
   under real Chrome; `nix develop -c go test -run TestBrowser`).
4. **Full unit suite + race subset on the final tree** (insurance — the
   final Go tree equals T21's, verified there, but a closeout re-run is
   cheap).
5. **Annotate ROADMAP harvested→shipped** for the five engine-tier items
   (per-IP, request logging, jitter, Retry-After, SSE counters/gauge).
6. **Annotate master plan M12–M21 + M30 rows** as shipped/done (only
   M22–M24 were annotated).
7. **Cut v0.14.0** with the engine-tier Added block (pending A1/your call —
   the content is ready: re-head + Version bump + footer + gate 0 after
   edits + tag-first).
8. **Pin the new CI steps' supply chain** (markdownlint-cli version, apt
   packages) — one Hygiene change.
9. **B8 sign-off → execute the AGENTS restructure** per the outline
   (one docs-only session, battery-gated).
10. **A2 sign-off → the em-dash mechanical pass** (goldens regen in the
    same change).
11. **File the erraudit nolint-audit issue** (the one valid draft — own
    repo, authorization pending).
12. **Execute the nixpkgs `go_1_27` bump** (quiet window + vendorHash).
13. **dnsblockd lockstep bump PR** (root + static to v0.7.0 — their gate
    unblocks).
14. **CV bump PR** (scope drafted on the row; deploy stays yours).
15. **templ-components next release → ListNote adoption** for the timeline
    cap (pin-guard ceremony: dedicated change + browser suite + pins in
    the same change).
16. **A6 sign-off → AGENTS release-discipline amendment** (release
    exclusivity doctrine).
17. **B5's one-sentence Shutdown doc comment** (issue #14 contract
    wording) — 5 minutes, could ride any docs change.
18. **Split metrics_test.go** along the internal/external seam.
19. **README SSE logging semantics sentence** (one line per connection at
    close).
20. **Status recorder: note the WriteHeader dedup semantics** in its doc
    comment (first call wins — matches net/http).
21. **A tier-1 boot-smoke CI job** if A4 lands on option 2 (design is
    ready).
22. **tc doctor fleet ask** — open one tracking item per consumer backlog
    (B7 recommendation).
23. **Second-session doctrine into AGENTS** (A6 option 1) once answered.
24. **docs/dependency-notes.md** extraction per the T30 outline (part of
    the restructure execution).
25. **Retire the templ gofumpt draft** (append the STALE verdict to the
    draft file itself, not only the packet).
26. **The 12-52 report's §f21** (nixpkgs bump) execution is #12 above —
    keep the row's live count synced when it lands.
27. **Consider evidence metrics** if B1 lands on option 2
    (`dashboard_evidence_proven/_unproven` gauges — public-mode safe).
28. **MVS fleet-watch**: if B7 lands on option 1 instead of the per-consumer
    ask, design the central watch then.
29. **Rate-limit metrics: consider a per-limiter label** if both limiters
    run concurrently (today they share one family — documented).
30. **Audit the new CI steps' runner behavior** after a runner-image update
    (apt/npm availability drift).
31. **A11y re-run of the axe audit** on the current tree (last full audit
    was the v1.27.0 adoption batch).
32. **Chaos-load repeat on the nightly canary schedule** (the evidence is
    point-in-time; a quarterly re-run keeps margins honest).
33. **`docs/design/2026-11-12_watchdog-security-audit.md`** — link it from
    AGENTS' watchdog bullet (currently only the gotcha points back).
34. **Pin-guard: add the cold-cache test to the pre-push gate list
    documentation** (AGENTS says 12 gates; there are 15 now — recount and
    update the bullet).
35. **AGENTS gate-count drift** is its own small docs fix (the bullet in
    the art-dupl section says "12 gates").
36. **Consider `art-dupl baseline` explicit output path** support upstream
    (the default wrote CWD/.art-dupl-baseline.json, which tripped the
    daemon twice — a small upstream ask for an --output flag).
37. **Session report hygiene**: the 12-52 report §g1-§g3 now duplicate
    batch A's appendix — annotate the 12-52 report's §g as "answered in
    batch A" once you reply.
38. **Renamer (file-and-image-renamer) healthd**: still pinned at
    v0.13.2 — ride the v0.14.0 cut when it happens.
39. **Verify pkg.go.dev renders the next release** (part of the standard
    verify-release flow — listed so it is not forgotten at cut time).
40. **Trend endpoint: the evidence-window semantics** if B1 lands — also
    note the observation window resets on pusher restart (tooltip already
    discloses; metrics would need the same caveat).
41. **Check `golangci-lint`'s new-linter surface** at the next fleet lint
    bump (the session's findings were all from known linters).
42. **Loadtest env-parameterized run** after the engine tier (the
    LOADTEST=1 harness has not run against the recorder-wrapped SSE path).
43. **Docs: deploy/README.md boot-verify tier cross-link** is done — verify
    the link target still matches after the next design-doc rename.
44. **Keep `reports/dupl/2026-10-25_art-dupl-baseline.txt`** as the
    rationale ledger — append future re-records there (the 2026-11-12
    note is already in).
45. **B3 fleet lint pin**: apply this repo's own golangci pin decision to
    ci.yml once decided (currently latest-pin by flake).
46. **Nightly fuzz: confirm the 13 targets ran green** on tonight's
    schedule (the registry gate counts steps, not outcomes).
47. **Session-start handshake discipline**: the newest status file is
    another session's handoff — this report is that file until superseded.
48. **ROADMAP "Coverage badge" idea** needs a CI coverage-upload step
    first — scope it when CI minutes are revisited.
49. **OG/social-preview metadata** (ROADMAP idea) — unchanged, no consumer
    ask yet.
50. **The v1.0 criteria sweep** re-runs once PVR (B4) and the engine tier
    ship — the sweep's date-relative claims will need a refresh.

## g) Three questions I cannot figure out myself

1. **CI fix-loop policy for the new steps**: the four new CI steps passed
   their first green run, but their first FAILURES (runner image drift,
   apt/npm availability) will surface on future runs. When a gate I added
   reds CI, do you want the watch-and-fix-push loop on master as standing
   behavior (small fix commits, disclosed), or should new-infra failures
   hold for your review first?
2. **art-dupl baseline sign-off**: the gate's accepted set changed twice in
   two days (empty record in the v0.13.3 train → the one documented group
   re-recorded today, hash `431a125136df2408`). Is the one-group baseline
   (view.templ/status.go `display.BadgeType` annotation pair) your accepted
   final state, or do you want the pair actually extracted (my
   recommendation: keep — different business rules, same return type, and
   the ledger documents it)?
3. **Cut v0.14.0 now or hold?** `[Unreleased]` carries five real Added
   entries (engine tier: metrics families, jitter, Retry-After on 503s,
   per-IP limits, request logging) plus the changelog-backfill Fixed
   entries (§g1). Batch A1 asked cadence abstractly; this is now concrete:
   my recommendation is to cut v0.14.0 within 48h (green tree + CI green +
   battery green are all true right now), carrying the backfill block —
   but the cut is yours to order.

---

_Written per the operator's `.md`-at-path standing instruction. Point-in-time
snapshot at `7bb61fb`, CI green on master, battery exit 0 (gates 13/15
skipped on the ambient PATH — see §b2), docs-gates 5/5. Annotate, never
rewrite (docs-health ANNOTATE)._
