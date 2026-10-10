# Session-Detail Gotchas

> Extracted from AGENTS.md 2026-10-10 (linter-budget trim, T20): the incident
> narratives and history behind each one-line pointer there. Load-bearing
> rules stay in AGENTS.md; this file keeps the evidence and the stories so
> nothing was lost to the trim.

## Release discipline (earned during the v0.7.0 cut; all four bit or paid off)

- **(0) Commit beats the daemon — verify batch → commit immediately.** The
  auto-daemon commits continuously and NEVER pushes: master can sit
  arbitrarily ahead of origin with CI blind to it — caveat (2026-09-22
  evening): the directive-restore commit reached origin BEFORE the explicit
  push ran (push printed "Everything up-to-date"), while later daemon
  commits sat unpushed, so SOMETHING auto-pushes under unknown conditions;
  never claim unpushed state without `git fetch` + comparing
  `master...origin/master`. The moment a batch is verified (build+tests
  green), `git add` + commit in the same tool-call chain, before starting
  the NEXT batch (the v0.7.0 release commit's message was lost this way,
  `ebf52d0`; four batches in the 2026-09-17 session); run `git status -sb`
  before any "CI is green" claim, pushing deliberately at milestones (push
  needs authorization). Release sessions start with
  `bash scripts/pre-push-checks.sh` (the desk gate) — see
  `docs/release-checklist.md` §4 gate 0.
- **(1) `nix fmt` runs AFTER the last `templ generate`** — every build/test
  app regenerates `view_templ.go` in raw form, so fmt-before-generate gets
  undone and the CI hygiene drift check goes red; canonical order is
  generate → fmt (the CI hygiene job's comment says so).
- **(2) Commit intent-bearing changes immediately after each verified
  batch** — the daemon ate the v0.7.0 release commit's message (`ebf52d0`);
  on 2026-09-10 two freshly written release scripts were swept into an
  adjacent auto-commit seconds before their intent commit landed. New
  files: commit right after their first successful run, before long
  verification chains. A long gate chain is ONE GIANT uncommitted window:
  on 2026-09-17 (v0.9.0) the daemon swept the raw templ output MID-CHAIN
  (`bdcb69c`, between the build's regenerate and nix fmt), so fmt + commit
  must run back-to-back at the chain's tail and the daemon commit gets
  folded forward, never rebased away.
- **(3) Push the TAG first, then master** — the master run's
  `fetch-depth: 0` checkout then sees the tag and the version-guard job
  cannot lose a fetch race.
- **(4) External-state verification is a script, not a memory** — proxy
  hash, sumdb, clean-dir consumer, GitHub Release state, CI on the release
  commit: `bash scripts/verify-release.sh <ver>`.

## The v0.10.1 dual-cut (2026-09-22)

v0.10.0's tag commit failed two hygiene CI jobs (missing CHANGELOG footer
link; the treefmt goimports bug — see AGENTS.md) and the release-draft
workflow's `awk -v` collapsed `\\[` to a plain `[`, turning the version
into a regex char-class so both tags' draft runs died with "no section
found" (fixed by index() matching, `83962f4`). Since tags are immutable,
the recovery was the green-CI re-cut v0.10.1 (identical code) with both
pages backfilled manually; verify-release check 5 permanently reports the
red draft run on both tags — a disclosed v0.8.0-class known exception, not
a quality-gate failure.

## The templ generator pin — history (rule in AGENTS.md)

The old split-brain: the flake's FOD regenerated with nixpkgs' `templ`, an
UNPINNED version that happened to equal go.mod's `v0.3.1020` on 2026-09-19
by coincidence — any nixpkgs bump would silently change the generated
output the golden tests pin. The rule was still missed once: the go.mod
bump to v0.3.1070 landed without the regenerated output, so the committed
`view_templ.go` stayed 0.3.1020-generated until 2026-10-08 — the first
fresh `nix run .#build` then failed five golden tests on whitespace-only
HTML drift (0.3.1070 folds the inter-component `" "` WriteString calls,
e.g. `</div> <div` → `</div><div`); regenerated files + refreshed goldens
are the canonical state now.

## Bisect wall `071c251..HEAD`

5+19 auto-daemon mid-edit commits do not compile (immutable history);
`git bisect skip` them (list in the audit doc's 2026-09-23 extension). Two
classes: torn-tree carriers, and the 2026-09-22+ go-directive-sweep tears
("updates to go.mod needed"). Root cause class: the daemon snapshots
half-wired trees — run `go build ./...` before walking away. Full audit:
`docs/status/archived/2026-09-04_19-15_bisectability-audit.md` (re-run the
extension's pinned-toolchain method — the ambient go now false-fails every
post-bump commit).

## UI dependency pins — history (rule in AGENTS.md)

The 2026-09-10 v1.16.0 audit retired the axe `definition-list`/`dlitem`
tolerance in `TestBrowser_Accessibility` after upstream templ-components#6
was fixed; the audit now fails on any serious/critical violation, both
themes. Undocumented sweeps have landed eight times (v1.19.2→v1.21.0 rode
the 2026-10-08 fleet sweep, daemon commit `54cdc59`; adopted deliberately
2026-10-09). Script-emitting upstream components (CopyButton, Tooltip) are
vetted for nonce/CSP compatibility before adoption: per-element inline
scripts and unconditional `nonce=""` attributes clash with the
per-request-nonce and SSE-patch paths here. Upstream StatCard icon tiles
use `text-green-600` on the green tone — banned page-wide by
`TestRender_ContrastSafeStatusColors`, so stat-card tones stay in the
blue/purple families (contrast-measured).

## erraudit blank-identifier policy — history (rule in AGENTS.md)

RE-VERIFIED 2026-09-23 (erraudit 1c6809a): the old "strings.Cut blank
identifiers are a false positive" claim no longer reproduces — the
analyzer fires nothing on those sites, so the status.go directive was
removed (`docs/upstream/erraudit-nolint-audit-pattern-noop-issue-draft.md`
records the verification AND a real remaining bug: `nolint-audit` reports
a directive as NEEDED even when the analyzer produces no finding there,
and `nolint-audit ./...` silently answers "no directives" while `.` finds
them).

## go-structure-linter is skip-gated until the fleet bump

The step is off in `.buildflow.yml` because BuildFlow pins
go-structure-linter v0.10.0, whose SDK `Lint()` does not call
`LoadProjectConfig`: the `.go-structure-linter.yaml` `flat` preset (the
root package IS the public import path of this single-package library,
same rationale as go-datastar ADR-002) stays inert and 17
root-package-files errors gate every run. The config file is committed and
becomes effective the moment BuildFlow pins a release with project-config
support; unskip then (TODO_LIST Blocked row). Use the named preset rather
than a hand-maintained exclude list — it is the tool's own answer for
deliberately-flat packages.

## go-auto-upgrade's samber/lo suggestions are a deliberate non-adoption

stdlib2lo flags manual Filter/GroupBy loops and suggests adding
github.com/samber/lo; this module keeps zero runtime dependencies (same
policy as the hand-rolled rate limiter and metrics exposition), so those
suggestion warnings are noise, not work. BuildFlow runs the migrators
in-process and ignores `.go-auto-upgrade.json`, so there is currently no
per-project way to silence them — a fleet-level BuildFlow change, not a
per-repo script.

## branching-flow skip history (re-opened then re-gated)

UNSKIPPED 2026-09-22 under the branching-flow v0.6.2+ SDK severity cap
(findings land as warnings/observations, gate stays green). Earlier gate
condition, for the record: 24 PHANTOM_TYPE findings demanding named string
types across the PUBLIC option API (a breaking redesign needing a
versioning decision) and 2 BOOL_BLIND errors wanting Config/introspectModes
bools packed into bit flags; no scoping mechanism existed
(`.branching-flow.yml` carries only tuning knobs, BuildFlow's provider
hardcodes `analysis.RunAll` options, and the phantom/boolblind analyzers
ignore `//nolint` — only roleak/do honor it). The two INDEX_OUT_OF_RANGE
warnings it surfaced were false positives made structurally impossible
anyway (the latency histogram buckets are sized
`[len(latencyBucketBounds)]` at the type level).

## Detect-only advisories that are deliberate non-fixes

- branching-flow flags `Config`/`viewModel` field counts and bool clusters
  as bit-flag candidates: the With*-option surface is the library's public
  API and named bools beat packed flags for readability.
- go-humanize-linter suggests dustin/go-humanize (RelTime/Commaf), which
  would break the zero-runtime-deps policy. Resolution state (2026-10-08):
  the two genuine H003 matches carry reasoned `//nolint:gohumanize`
  directives (`dashboard.go` HealthCheck watchdog — millisecond precision
  beats RelTime's coarse buckets; `status.go` formatAge — the coarse stamp
  IS the anti-fingerprint accepted risk), so a manual
  `go-humanize-linter .` exits 0. Use the UNscoped `//nolint:gohumanize`
  form: golangci's nolintlint rejects the linter's own
  `//nolint:gohumanize:H003` colon-scoped syntax ("should match
  //nolint[:<linters>]"), and wsl_v5 wants no blank line between directive
  and statement. The third finding (H009 on `ExportHandler`) was an
  upstream detector bug — `getBasicLit` strips the unary minus, so
  `FormatFloat('g', -1, 64)` precision looked positive and CSV field
  delimiters looked like digit grouping — fixed in go-humanize-linter
  (sign-aware precision + `'f'`-verb check); the installed system binary
  lags until the next rebuild, so check `--version` against the repo HEAD
  before trusting its output.
- jscpd flags repeated test scaffolding, where per-test isolation is
  preferred over shared helpers.

All three report without gating — don't "fix" them into dependency
additions or API churn.

## templ-generate is skip-gated: it races nix source snapshots

The step re-raws `view_templ.go`/`page_scripts_templ.go` mid-run (templ
emits unformatted Go; canonical order is generate → fmt). BuildFlow lets
nix-evaluating steps run concurrently, and a `git+file` source snapshot
ingested during the raw window poisons the flake's treefmt-check
derivation: verified 2026-09-16, files were repaired by nix-fmt at
15:19:12 yet treefmt-check failed at 15:19:57 on gofumpt diffs against the
already-gone raw state — the check never saw the repaired tree. This made
nix-build fail nondeterministically (nix-build-verify 0/10 before the
skip, green after). With the step skipped the pipeline tree is immutable
and runs are deterministic; generated-file freshness stays owned by the CI
hygiene job and the flake apps (both regenerate pre-build). Unskip when
BuildFlow orders tree-mutating generators before all nix-evaluating steps
(fleet-level DAG change; TODO_LIST Blocked row — includes the
upstream-file vs local-patch decision).

## BuildFlow-adjacent tool traps

1. `erraudit nolint-audit` takes a DIRECTORY argument, not a package
   pattern: `./...` doesn't exist as a directory and its best-effort walk
   silently reports "No directives found" — use `.`.
2. A stale erraudit binary fails the same audit with a bogus "go: updates
   to go.mod needed" package-load error; rebuilding from HEAD fixed it
   (same stale-binary class as the buildflow doctor check — trust
   `erraudit version`/git log, not PATH).
3. Never run Go tooling (erraudit, go list, tests) while a buildflow run is
   active: its go-mod/templ steps mutate go.mod and `*_templ.go`
   mid-flight and the concurrent loader reads torn state.
4. **BuildFlow's `go-mod-update` step sweeps the UI pins** — on 2026-09-22
   it bumped templ-components 1.18.0→1.19.1, go-datastar 0.5.0→0.6.0,
   go-health →v0.3.0, go-sse →0.6.1 in one run and the auto-daemon
   committed it (`68ac162`); the golden render tests caught the drift only
   after the fact and the commit had to be reverted, and the same run's
   go-mod-normalize downgraded the `go` directive 1.27.1→1.27 (breaking
   module loading against go-health's floor). Run local gates as
   `buildflow --build-mode dev --exclude go-mod-update` and check
   `git diff go.mod go.sum` after ANY buildflow run — the
   check-ui-pins.sh guard only fires in CI (Build+Test jobs), after the
   sweep already landed locally. RESOLVED 2026-09-25: go-health v0.4.1
   lowered its floor to minor form, this repo's directive settled at
   `go 1.27` legitimately, and go-version-auto-configure v0.2.0+ gates its
   own rewrites with a dependency-floor check (verified: the dashboard
   shape comes back dep-forced and the file is left untouched) — the
   check-go-directive.sh guard was deleted as obsolete. The
   go-mod-update sweep itself remains a real trap (see the
   templ-components revert).
5. The flake's nix-build steps fail on this machine with
   `lookup proxy.golang.org ... connection refused` (FOD sandbox DNS vs
   the local DNS blocker) — chronic 100% failure, environmental, not a
   code signal; `--exclude nix-hash-fix --exclude nix-build-verify` for
   local dev gates and trust CI for flake builds.

## Parallel-session handshake (2026-09-17: two sessions, one tree)

This repo routinely runs TWO agent sessions plus the auto-daemon. Before
ANY write: `git log --oneline -10`, `ls docs/status/ | tail`, `git status`.
The newest file in `docs/status/` is another session's handoff: it lists
what they landed (cross it off, don't redo it) and what they left.
Concurrency also means gate results go stale: re-run gates on the tip
immediately before pushing, and expect `nix run .#build`'s templ
regeneration to be swept into a daemon commit — run `nix fmt` and commit
the canonical formatting yourself if it lands raw.

## Deploy stack facts

Full facts live in `deploy/README.md` (digest-pinned, healthcheck-gated,
boot-verified 2026-09-23 with screenshot). Two machine-level traps: the
provisioned alert example takes `relativeTimeRange` as integer SECONDS,
not duration strings; host ports 8080/9090/3000 collide with other
projects on this machine — remap via a compose override with `!override`
port lists (plain overrides MERGE and still bind the taken port).
