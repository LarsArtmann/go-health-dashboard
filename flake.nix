{
  description = "go-health-dashboard — browser-friendly health dashboard for go-health";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    systems.url = "github:nix-systems/default";
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      treefmt-nix,
      systems,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import systems;

      imports = [
        treefmt-nix.flakeModule
      ];

      perSystem =
        { config, pkgs, ... }:
        let
          inherit (pkgs) lib;
          goPkg = pkgs.go_1_26;

          mkApp =
            name: runtimeInputs: text:
            let
              script = pkgs.writeShellApplication {
                inherit name runtimeInputs text;
              };
            in
            {
              type = "app";
              program = lib.getExe script;
            };
        in
        {
          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              gofumpt.enable = true;
              goimports.enable = true;
              golines.enable = true;
              nixfmt.enable = true;
            };
          };

          checks.format = config.treefmt.build.check self;

          devShells.default = pkgs.mkShell {
            packages = [
              goPkg
              pkgs.golangci-lint
              pkgs.gofumpt
              pkgs.golines
              pkgs.gopls
              pkgs.gotools
              pkgs.govulncheck
              pkgs.gosec
              pkgs.templ
              pkgs.trash-cli
              pkgs.chromium
            ];

            GOWORK = "off";
            GOEXPERIMENT = "jsonv2";
            GO_HEALTH_DASHBOARD_CHROME = "${pkgs.chromium}/bin/chromium";

            shellHook = ''
              echo "go-health-dashboard dev shell — $(go version)"
              echo "GOEXPERIMENT=$GOEXPERIMENT (required for go-sse dependency)"
              echo "GOWORK=off (ignore parent workspace)"
              echo "GO_HEALTH_DASHBOARD_CHROME=$GO_HEALTH_DASHBOARD_CHROME (browser suite)"
            '';
          };

          apps = {
            generate = mkApp "generate" [ goPkg pkgs.templ ] ''
              templ generate
              go mod tidy
            '';

            test = mkApp "test" [ goPkg ] ''
              templ generate
              GOEXPERIMENT=jsonv2 go test ./... -count=1 "$@"
            '';

            test-race = mkApp "test-race" [ goPkg ] ''
              templ generate
              GOEXPERIMENT=jsonv2 go test ./... -race -count=1 "$@"
            '';

            build = mkApp "build" [ goPkg pkgs.templ ] ''
              templ generate
              GOEXPERIMENT=jsonv2 go build ./...
            '';

            vet = mkApp "vet" [ goPkg ] ''
              GOEXPERIMENT=jsonv2 go vet ./...
            '';

            lint = mkApp "lint" [ pkgs.golangci-lint ] ''
              GOEXPERIMENT=jsonv2 golangci-lint run ./...
            '';

            coverage = mkApp "coverage" [ goPkg ] ''
              GOEXPERIMENT=jsonv2 go test ./... -coverprofile=coverage.out -covermode=atomic "$@"
              go tool cover -func=coverage.out
            '';

            vulncheck = mkApp "vulncheck" [ pkgs.govulncheck ] ''
              GOEXPERIMENT=jsonv2 govulncheck ./...
            '';

            security = mkApp "security" [ pkgs.gosec ] ''
              GOEXPERIMENT=jsonv2 gosec ./...
            '';

            example = mkApp "example" [ goPkg pkgs.templ ] ''
              templ generate
              GOEXPERIMENT=jsonv2 go run ./example "$@"
            '';

            clean = mkApp "clean" [ goPkg pkgs.trash-cli ] ''
              trash-put coverage.out 2>/dev/null || true
              go clean -testcache
            '';
          };
        };
    };
}
