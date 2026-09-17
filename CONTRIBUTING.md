# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

This project uses [Nix flakes](https://wiki.nixos.org/wiki/Flakes) for all build
and task automation. Enter the dev shell with `nix develop`.

All Go commands require `GOEXPERIMENT=jsonv2` (the go-sse dependency uses
`encoding/json/v2`). The Nix devShell sets this automatically.

```bash
nix run .#generate   # templ generate + go mod tidy (run after editing .templ files)
nix run .#test-race  # tests with race detector
nix run .#build      # templ generate + go build
nix run .#lint       # golangci-lint
nix run .#vulncheck  # govulncheck
nix fmt              # format code (gofumpt, goimports, golines, nixfmt)
```

CI enforces a **78% coverage floor** on the race/coverage job (baseline
83.4%). Check locally with `nix run .#coverage` before pushing.

Without Nix, prefix all Go commands with `GOEXPERIMENT=jsonv2` and run
`templ generate` before building.

## Browser and Screenshot Tests

The runtime CSP and accessibility tests drive a real headless Chrome. They
**skip automatically** when no Chrome binary is available; to run them,
point `GO_HEALTH_DASHBOARD_CHROME` at a Chrome/Chromium binary:

```bash
GO_HEALTH_DASHBOARD_CHROME=/usr/bin/google-chrome-stable \
  nix run .#test -- -run TestBrowser -v
```

Chrome startups are heavyweight and serialized internally — don't bypass the
`startHeadlessChrome` helper in new browser tests. The accessibility audit
downloads axe-core from cdnjs at setup and skips when offline.

Screenshot capture (regenerating the README images) additionally needs an
output path:

```bash
SCREENSHOT_OUTPUT=docs/screenshot.png \
  GO_HEALTH_DASHBOARD_CHROME=/usr/bin/google-chrome-stable \
  nix run .#test -- -run 'TestScreenshot' -v
```

## DI Registration Path

If your service already uses a samber/do injector (go-health requires one),
prefer `dashboard.Register(injector, probe, opts...)` over `dashboard.New`.
`Register` stores the dashboard in the container, so `do.Shutdown` and
`do.HealthCheck` cascades include it automatically — see the package
documentation and the `lifecycle_test.go` suite for the exact contracts.

## Releasing

Maintainers cut releases by following `docs/release-checklist.md` —
reconcile, re-head the CHANGELOG, bump the `Version` const in the same
commit as the annotated tag, run every gate, then push and verify the
proxy, CI, and the GitHub Release page.

## Guard Scripts

Four scripts guard contributions; CI runs them and each fails loudly with
a reason:

| Script                       | What it checks                                                                                        |
| ---------------------------- | ----------------------------------------------------------------------------------------------------- |
| `scripts/check-changelog.sh` | CHANGELOG structure: exactly one `[Unreleased]` first, semver-descending version sections             |
| `scripts/check-ui-pins.sh`   | UI dependency pins (templ-components, go-datastar) match the audited versions documented in AGENTS.md |
| `scripts/pre-push-checks.sh` | The desk gate: build, tests, lint, formatting — run before claiming work is pushable                  |
| `scripts/verify-release.sh`  | Post-release external state: tag, proxy resolution, CI, GitHub Release page (`verify-release.sh <version>`) |

Run them locally the same way CI does:

```bash
bash scripts/check-changelog.sh
bash scripts/check-ui-pins.sh
bash scripts/pre-push-checks.sh
```

Two conventions the guards assume: generated templ code is regenerated
before every build (`nix run .#build` does this), and `nix fmt` runs AFTER
the last generation — generating produces unformatted Go, so
fmt-before-generate gets undone and the CI hygiene check goes red.

## Reporting Issues

Please use GitHub Issues to report bugs or request features. Security
issues follow [SECURITY.md](SECURITY.md) instead.
