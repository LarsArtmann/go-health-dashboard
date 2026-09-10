# Release Checklist

The ritual for cutting a release, learned by scar tissue across the
v0.3.x–v0.5.x cycles. Follow top to bottom; do not skip the gates.
Version history source of truth: `CHANGELOG.md` (Keep a Changelog).

## 1. Reconcile

- [ ] `git fetch && git status` — clean tree, synced with `origin/master`.
- [ ] Dependency pins intact: `scripts/check-ui-pins.sh` exits 0
      (currently templ-components v1.13.0/v1.13.2 + go-datastar v0.5.0).
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

- [ ] `nix fmt`, then commit `chore(release): vX.Y.Z — <theme>` with a
      body naming the user-visible changes.
- [ ] Annotated tag: `git tag -a vX.Y.Z -m "<notes>"` (signed tags
      encouraged: `git config tag.gpgSign true` once, then plain
      `git tag -a` produces signed tags).
- [ ] Screenshots refreshed when the render changed:
      `SCREENSHOT_OUTPUT=docs/screenshot.png nix develop -c go test -run 'TestCaptureREADME_Screenshot$'`
      and the dark variant with `SCREENSHOT_OUTPUT_DARK`.
- [ ] `git push --follow-tags` — never push a red tree.

## 6. Verify

- [ ] Proxy: `go list -m github.com/larsartmann/go-health-dashboard@vX.Y.Z`
      resolves (may take ~1 min after push).
- [ ] CI: `gh run watch` — every job green on the release commit
      (Test, Browser, Lint, Build, Version-guard, Vulnerability Scan).
- [ ] `gh release create vX.Y.Z --title vX.Y.Z --notes-file <notes>`
      using the section extracted from `CHANGELOG.md` (historical
      releases get their pages backfilled the same way).

## 7. Close the loop

- [ ] Update `TODO_LIST.md` release rows to `DONE` with the tag/commit
      and verification evidence.
- [ ] Announce anything Compatibility-section-worthy to consumers.
