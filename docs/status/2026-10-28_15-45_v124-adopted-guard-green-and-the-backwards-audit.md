# Status — 2026-10-28 15:45 CET

**Session scope:** finish the chromedp v0.21 + templ-components v1.24.0
thread in `go-health-dashboard` that two earlier sessions left prepared but
unapplied. Directive: keep going until everything works. One repo touched;
no pushes.

**Headline:** v1.24.0 is ADOPTED with the guard green, the goldens
regenerated, and the audit's Table direction corrected for the third and
final time (it was backwards in the record, twice). The originally pasted
`buildflow -s test-race` failure is fixed and verified green twice; the
second paste of the same failure was output from the pre-adoption window.

---

## a) FULLY DONE

### a1. The adopt-vs-revert decision: ADOPTED, applied, and green

Rationale recorded in the guard's audit history: the daemon climbs
regardless (ten unguarded sweeps), v1.24.0 is an a11y-focused release
whose changes are all improvements, and the blast radius was already
measured. Applied in one change:

- 7 goldens regenerated (`go test -run TestGoldenRender -golden-update .`)
  and the diff reviewed hunk-by-hunk.
- `scripts/check-ui-pins.sh`: four pins → v1.24.0, the "UNAUDITED DRIFT"
  block became a dated re-audit entry, and the failure message now names
  the full ceremony (audit delta, regen command, browser suite, same-change
  pins). `set -o pipefail; bash scripts/check-ui-pins.sh` → exit 0, 10 OK.
- TODO_LIST row 31 → DONE; AGENTS.md pin bullet rewritten; CHANGELOG
  `[Unreleased]` filled (adoption, chromedp v0.21 migration + honest skip,
  health-hub two-phase drain).

### a2. The v1.24.0 Table delta was recorded BACKWARDS — corrected at source

The 11:19 report said the Table wrapper GAINS `role="region"`; the 15:17
report "corrected" it to wrapper-LOSES/table-gains-aria-label and that
backwards version went into `check-ui-pins.sh`, `AGENTS.md`, TODO_LIST,
and the `a44f0fd` commit message. Both were prose. Re-derived from
evidence:

- Golden word-diff: exactly 23 × `[-tabindex="0"><table-]` →
  `{+tabindex="0" role="region"><table+}` and NOTHING else.
- Module-cache source diff (`display/table.templ` v1.23.0 vs v1.24.0): the
  wrapper GAINS `role="region"` always, plus `aria-label` only when
  AriaLabel/Caption is set (the dashboard passes neither — its entire
  rendered drift is the 23 unnamed region wrappers).

CopyButton (label span loses live semantics; sibling
`data-tc-copy-status` sr-only span; `tcAnnounceCopy` in `display/shared.go`)
and the 404 search-field submit button are real but NOT rendered by this
repo (verified by grep + the zero-diff goldens). `a44f0fd`'s commit message
stays wrong (amending it means rebasing over daemon commits; not worth it)
— the guard's audit entry now states the correction explicitly.

### a3. The pasted `buildflow -s test-race` failure: fixed and proven

- `nix develop -c go test -count=1 .` → ok (12s; 19 browser tests
  self-skip honestly, everything else runs).
- `nix develop -c go test -race -count=1 -run TestGoldenRender -v .` →
  all 7 previously-failing tests PASS.
- `buildflow -s test-race --verbose` → **EXIT=0** (`✔ test-race 6.7s`),
  twice; buildflow's own history shows test-race's last failure was the
  pre-adoption run (~15:21, before the 15:26 regen).
- `nix run .#lint` → 0 issues (first lint run any session did on the
  migrated tree).
- `nix develop -c bash scripts/pre-push-checks.sh`: round 1 failed on
  exactly one check — FEATURES.md's test-suite row claimed 47 test files,
  reality is 48 (`chromedp_eval_test.go`, added by the migration; function
  count unchanged at 346). Row recounted; round 2 re-run.

### a4. Commit hygiene under the daemon

The daemon captured the adoption as two `chore: auto-commit` commits;
HEAD was amended into `22d521e` with the full intent message (split onto
the goldens commit disclosed), per the documented recovery recipe.

---

## b) PARTIALLY DONE

### b1. chromedp v0.21 runtime verification — still not proven locally

Re-probed this session: the host's Chromium still cannot complete an http
load (`TestBrowser_CSPCleanRuntime` self-skips in 12s with the reason).
The adoption therefore records an explicit CEREMONY DEVIATION: the local
browser half could not run; CI's browser job (own Chrome) carries runtime
verification, and a red there means a one-change revert of pins + go.mod.
TODO_LIST row 32 stays BLOCKED and accurate.

### b2. TODO row 25 (dependabot `minor-and-patch-290df43455`)

chromedp v0.21.0 + cdproto v0.158.4 are already in go.mod (the migration
subsumes the branch's bump); the branch is superseded, but closing a
remote branch is a push-adjacent action — left for the owner.

---

## c) NOT STARTED / OWNED ELSEWHERE

1. DiscordSync's v1.24.0 adoption (15 snapshot regens + its 404 visual
   review) — that repo's own change; same cause, same ceremony.
2. Pushing this repo (10+ commits ahead) and verifying CI's browser job
   goes green on the adoption — awaiting authorization per release
   discipline.

---

## d) LESSONS

1. **A "corrected" prose delta is still prose.** The 15:17 session caught
   the 11:19 direction error and inverted it — into the opposite error.
   The golden word-diff + module-cache source diff are two commands and
   settle it permanently; both are now named in the guard's audit entry as
   the verification method.
2. **A pasted failure can be stale.** The second paste matched the
   pre-adoption window exactly; `buildflow history`'s "last failing" time
   plus one re-run proved the current tree green before any "repair" was
   attempted.
3. **Gates that recount reality are cheap to satisfy and hard to fool** —
   the FEATURES.md 47→48 drift was caught by the pre-push script, not by
   a human.
