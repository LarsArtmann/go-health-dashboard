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

Without Nix, prefix all Go commands with `GOEXPERIMENT=jsonv2` and run
`templ generate` before building.

## Reporting Issues

Please use GitHub Issues to report bugs or request features.
