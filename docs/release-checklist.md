# Release Checklist

Process for cutting a go-health-dashboard release. The CI version-guard job
enforces that `dashboard.Version` matches the latest git tag — bump the const
and tag in lockstep.

## Pre-flight

- [ ] `nix run .#build && nix run .#test && nix run .#lint` green
- [ ] `nix run .#test-race` green
- [ ] `nix run .#vulncheck` green
- [ ] `nix flake check` green
- [ ] `scripts/check-ui-pins.sh` green — or the movement was a dedicated,
      audited change with a green browser suite
- [ ] Full browser suite green (`nix develop -c go test -run TestBrowser`) —
      required for ANY render change; the unit suite cannot see CSP or a11y
      regressions
- [ ] `nix fmt` clean, no gopls-only findings left unassessed

## Release mechanics

- [ ] `CHANGELOG.md` section added for the new version
- [ ] `dashboard.Version` const bumped in the same commit as the tag target
- [ ] `FEATURES.md` updated to reflect shipped features (status column honest)
- [ ] `README.md` options/routes/current — no documentation of vaporware
- [ ] Screenshots refreshed when the render changed:
      `SCREENSHOT_OUTPUT=docs/screenshot.png nix develop -c go test -run 'TestCaptureREADME_Screenshot$'`
      (and `SCREENSHOT_OUTPUT_DARK=docs/screenshot-dark.png ... -run TestCaptureREADME_ScreenshotDark`)
- [ ] Commit, then annotated tag: `git tag -a vX.Y.Z -m "vX.Y.Z"` and push
      with `--follow-tags`
- [ ] Verify pkg.go.dev picked up the new version after the module proxy
      refreshes
