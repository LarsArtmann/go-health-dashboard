# Release Checklist

The ritual for cutting a release, learned by scar tissue across the
v0.3.x–v0.5.x cycles. Follow top to bottom; do not skip the gates.
Version history source of truth: `CHANGELOG.md` (Keep a Changelog).

## 1. Reconcile

- [ ] `git fetch && git status` — clean tree, synced with `origin/master`.
- [ ] Dependency pins intact: `scripts/check-ui-pins.sh` exits 0
      (currently templ-components v1.16.0 + v1.16.0/datastar +
      v1.16.0/utils + go-datastar v0.5.0).
      Any movement must be a dedicated, audited change with a green
      browser suite — never a silent sweep.
- [ ] `TODO_LIST.md` rows intended for this release are done or deferred.

## 2. CHANGELOG

- [ ] Every merged change has an entry under `[Unreleased]`
      (Added / Changed / Deprecated / Removed / Fixed / Security).
- [ ] Breaking or behavior-visible changes are duplicated in a
      **Compatibility** section with migration notes.
- [ ] Re-head: `## [Unreleased]` → `## [X.Y.Z] - YYYY-MM-DD`, add a
      3–6 line blurb explaining the release's theme.

## 3. Version const

- [ ] Bump `const Version` in `dashboard.go` **in the same commit as
      the tag** — the CI version-guard job fails when the const and the
      latest tag diverge.
- [ ] Update the Released row in `FEATURES.md` (and the README
      dependency matrix if `go.mod` versions moved).

## 4. Gates (all green before anything is pushed)

Ordering rule: `nix fmt` runs AFTER the last `templ generate` — every
build/test app regenerates `view_templ.go` and undoes an earlier fmt,
which is exactly what the CI hygiene drift check catches (v0.7.0 cut,
fumble d-2). Canonical tail: build → `nix fmt` → flake check.

- [ ] `nix run .#build`
- [ ] `nix run .#test-race`
- [ ] `nix run .#lint` (0 issues)
- [ ] `nix run .#vet`
- [ ] `nix flake check`
- [ ] Browser suite green after ANY render change (not just dependency
      movement — the unit suite cannot see CSP, a11y, or SDK-runtime
      regressions): `nix develop -c go test -run TestBrowser`
- [ ] `nix run .#vulncheck`

## 5. Commit, tag, push

Commit discipline: the auto-daemon commits continuously. Commit each
verified batch immediately — a multi-minute edit→commit window loses the
release commit's message to a `chore: auto-commit` heuristic (v0.7.0's
`ebf52d0` wears that scar permanently; it happened again 2026-09-10).

- [ ] `nix fmt`, then commit `chore(release): vX.Y.Z — <theme>` with a
      body naming the user-visible changes.
- [ ] Annotated tag: `git tag -a vX.Y.Z -m "<notes>"` (signed tags
      encouraged: `git config tag.gpgSign true` once, then plain
      `git tag -a` produces signed tags).
- [ ] Screenshots refreshed when the render changed:
      `SCREENSHOT_OUTPUT=docs/screenshot.png nix develop -c go test -run 'TestCaptureREADME_Screenshot$'`
      and the dark variant with `SCREENSHOT_OUTPUT_DARK`.
- [ ] Push order: TAG FIRST (`git push origin vX.Y.Z`), then master.
      The master CI run's `fetch-depth: 0` checkout then sees the tag and
      the version-guard job cannot lose a fetch race (validated on the
      v0.7.0 cut).

## 6. Verify

One command runs the whole chain (tag on origin, proxy hash == tag
commit, sumdb-verified fresh-cache download, clean-dir consumer
get/build/run, GitHub Release state, CI runs on the release commit):

- [ ] `bash scripts/verify-release.sh vX.Y.Z`

If a slow external surface (pkg.go.dev, proxy indexing) lags, poll on a
bounded loop (e.g. 30×10s) — never two fetches and a shrug (the v0.7.0
cut left pkg.go.dev dangling ~12 min for exactly that reason).

- [ ] `gh release create vX.Y.Z --title vX.Y.Z --notes-file <notes>`
      using the section extracted from `CHANGELOG.md` (historical
      releases get their pages backfilled the same way; pass
      `--latest=false` when backfilling anything older than the current
      Latest).

## 7. Close the loop

- [ ] Update `TODO_LIST.md` release rows to `DONE` with the tag/commit
      and verification evidence.
- [ ] Announce anything Compatibility-section-worthy to consumers.
