# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

This project uses [Nix flakes](https://nixos.wiki/wiki/Flakes) for all build
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

CI enforces a **75% coverage floor** on the race/coverage job (baseline
76.9%). Check locally with `nix run .#coverage` before pushing.

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

## Reporting Issues

Please use GitHub Issues to report bugs or request features.
