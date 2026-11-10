# NixOS module for go-health-dashboard — the federated health-hub binary.
#
# One dashboard federating every service's existing go-health endpoint over
# HTTP. Remote checks land namespaced "name/check" (worst-of across
# remotes), one card per remote (GroupBySource); a dark remote renders a
# "name/reachable" FAIL row instead of silently freezing at its last state.
#
# Stateless: no StateDirectory, no secrets — the hub is merge-on-read;
# its only memory is the in-process trend ring (opt-in via `trend`).
# Binds loopback by default: go-health documents expose internal check
# names and error strings, so the raw port must not reach an unauthenticated
# network — front it with your own reverse proxy for remote access.
#
# Usage:
#   imports = [ inputs.go-health-dashboard.nixosModules.default ];
#   services.health-dashboard = {
#     enable = true;
#     remotes = [ "cv=http://127.0.0.1:8098/health" ];
#   };
{
  config,
  lib,
  ...
}:
let
  cfg = config.services.health-dashboard;
in
{
  options.services.health-dashboard = {
    enable = lib.mkEnableOption "federated go-health hub dashboard";

    package = lib.mkOption {
      type = lib.types.package;
      description = "The health-hub package (inputs.go-health-dashboard.packages.\${system}.default).";
    };

    remotes = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      description = ''
        Federated upstreams as name=url pairs, fetched fresh on every
        read. Names namespace checks (name/check) and must be unique,
        non-empty, and free of "/". URLs are absolute http(s) endpoints
        answering GET with a go-health JSON document (a bare probe's
        readiness handler, or any go-health-dashboard route — the hub
        always sends Accept: application/json).
      '';
      example = [ "cv=http://127.0.0.1:8098/health" ];
    };

    addr = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1:8080";
      description = ''
        Listen address (host:port) for HEALTH_HUB_ADDR. Loopback by
        default — check names and error strings are internal.
      '';
    };

    trend = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = ''
        Enable the trend sparkline + timeline card (in-memory ring,
        ~1h of samples at the 2s push cadence). Purely additive — no
        persistence, no wire-contract change.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    assertions = [
      {
        assertion = cfg.remotes != [ ];
        message = "services.health-dashboard.remotes must list at least one name=url pair — a federation of nothing has no surface to serve (the hub fails the unit at startup anyway; this fails the eval instead).";
      }
    ];

    systemd.services.health-dashboard = {
      description = "Federated go-health hub";
      wantedBy = [ "multi-user.target" ];
      startLimitBurst = 5;
      startLimitIntervalSec = 300;
      serviceConfig = {
        Type = "simple";
        # Stateless network daemon: no home, no state dir, no secrets.
        DynamicUser = true;
        ExecStart = lib.getExe cfg.package;
        Environment = [
          "HEALTH_HUB_ADDR=${cfg.addr}"
          "HEALTH_HUB_REMOTES=${lib.concatStringsSep "," cfg.remotes}"
        ]
        ++ lib.optional cfg.trend "HEALTH_HUB_TREND=1";
        Restart = "on-failure";
        RestartSec = "5s";
        MemoryMax = lib.mkDefault "256M";
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        RestrictNamespaces = true;
        CapabilityBoundingSet = "";
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
      };
    };
  };
}
