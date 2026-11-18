# NixOS module for go-health-dashboard — the federated health-hub binary.
#
# One dashboard federating every service's existing go-health endpoint over
# HTTP. Remote checks land namespaced "name/check" (worst-of across
# remotes), one card per remote (GroupBySource); a dark remote renders a
# "name/reachable" FAIL row instead of silently freezing at its last state.
#
# Stateless: no StateDirectory — the hub is merge-on-read; its only memory
# is the in-process trend ring (opt-in via `trend`) and the HMAC-sealed
# session cookie when `oidc` is on. Binds loopback by default: go-health
# documents expose internal check names and error strings, so the raw port
# must not reach an unauthenticated network — front it with your own
# reverse proxy, or turn on the native OIDC login (`oidc`) for a public
# vhost behind TLS.
#
# Usage:
#   imports = [ inputs.go-health-dashboard.nixosModules.default ];
#   services.health-dashboard = {
#     enable = true;
#     remotes = [ "cv=http://127.0.0.1:8098/health" ];
#   oidc = {
#     issuer = "https://auth.example.com";
#     clientId = "health";
#     redirectUrl = "https://health.example.com/oidc/callback";
#     group = "viewers";              # optional membership gate
#     # Both secret files are read DIRECTLY as the static
#     # `health-dashboard` service user once at startup (no LoadCredential
#     # indirection — a DynamicUser could never read the root-owned copies,
#     # the never-green EACCES class). The deployment must grant the
#     # service user read access, 0640 root:health-dashboard being the
#     # shape (a boot-time ownership heal is the usual glue).
#     clientSecretFile = "/var/lib/secrets/health-client-secret";
#     sessionSecretFile = "/var/lib/secrets/health-session-secret";
#   };
#   };
{
  config,
  lib,
  ...
}:
let
  cfg = config.services.health-dashboard;

  # The secret files ride the deployment's own paths verbatim: the service
  # user opens them directly (read once at startup, so rotation is replace
  # + restart). A LoadCredential indirection was tried first and is WRONG
  # for root-owned secret sources: the credential copy preserves the
  # source mode (0600 root), and the unit's user — dynamic or static —
  # cannot open it (the health.artmann.tech hub crash-looped exactly there,
  # never green from day one).
  oidcEnv = lib.optionals (cfg.oidc != null) [
    "HEALTH_HUB_OIDC_ISSUER=${cfg.oidc.issuer}"
    "HEALTH_HUB_OIDC_CLIENT_ID=${cfg.oidc.clientId}"
    "HEALTH_HUB_OIDC_REDIRECT_URL=${cfg.oidc.redirectUrl}"
    "HEALTH_HUB_OIDC_GROUP=${cfg.oidc.group}"
    "HEALTH_HUB_OIDC_SESSION_TTL=${cfg.oidc.sessionTtl}"
    "HEALTH_HUB_OIDC_CLIENT_SECRET_FILE=${cfg.oidc.clientSecretFile}"
    "HEALTH_HUB_OIDC_SESSION_SECRET_FILE=${cfg.oidc.sessionSecretFile}"
  ];

  # The OIDC egress fence: when the deployment pins egress CIDRs the unit
  # allows loopback plus exactly those — a co-hosted issuer behind
  # split-horizon DNS otherwise resolves to the host's OWN public address
  # and a loopback-only default silently drops every discovery call.
  egressFence = lib.optionalAttrs (cfg.oidc != null && cfg.oidc.egressCidrs != [ ]) {
    IPAddressAllow = [ "localhost" ] ++ cfg.oidc.egressCidrs;
    IPAddressDeny = "any";
  };
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

    oidc = lib.mkOption {
      type = lib.types.nullOr (
        lib.types.submodule {
          options = {
            issuer = lib.mkOption {
              type = lib.types.str;
              description = "OIDC issuer base URL (no trailing slash; must equal the discovery document's issuer byte-for-byte).";
            };

            clientId = lib.mkOption {
              type = lib.types.str;
              description = "Confidential client id registered at the issuer.";
            };

            redirectUrl = lib.mkOption {
              type = lib.types.str;
              description = "Absolute callback URL (scheme + host + /oidc/callback) registered at the issuer.";
            };

            group = lib.mkOption {
              type = lib.types.str;
              default = "";
              description = ''
                Optional group the identity must carry (id_token groups
                claim, userinfo fallback). Empty disables the gate —
                then the issuer-side client restriction is the only
                membership check.
              '';
            };

            sessionTtl = lib.mkOption {
              type = lib.types.str;
              default = "12h";
              description = "Session lifetime (a duration string the binary parses).";
            };

            clientSecretFile = lib.mkOption {
              type = lib.types.path;
              description = ''
                Path to the confidential client secret, read directly as
                the `health-dashboard` service user once at startup
                (rotation = replace the file + restart the unit). The
                deployment must grant the service user read access —
                0640 root:health-dashboard is the shape.
              '';
            };

            sessionSecretFile = lib.mkOption {
              type = lib.types.path;
              description = ''
                Path to the cookie HMAC key, at least 16 bytes, read
                directly as the `health-dashboard` service user under the
                same group-read contract as `clientSecretFile`.
              '';
            };

            egressCidrs = lib.mkOption {
              type = lib.types.listOf lib.types.str;
              default = [ ];
              description = ''
                When non-empty, the unit's egress is fenced to loopback
                plus exactly these CIDRs (IPAddressAllow/Deny). Pin the
                issuer's address when it sits behind split-horizon DNS on
                the same host — the DNS answer is then the host's own
                public address, which a loopback-only fence silently
                drops. Kubelet probes and the wrapped remotes stay
                reachable on loopback either way.
              '';
              example = [ "203.0.113.10/32" ];
            };
          };
        }
      );
      default = null;
      description = ''
        Native OpenID Connect login. When non-null every
        dashboard-owned surface (HTML, SSE, favicon, metrics) requires a
        session cookie; /oidc/login starts the authorization-code + PKCE
        S256 dance, JSON-negotiating unauthenticated requests get a clean
        401, and the kubelet probes (/healthz, /readyz, /startupz) stay
        public. Null (default) keeps the hub auth-free — fine for a
        loopback bind behind your own reverse proxy.
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

    # Static identity: the hub reads its OIDC secret files directly, so it
    # needs a stable group the deployment can grant with (a DynamicUser
    # rotates uid/gid per start and cannot be chgrp'd for).
    users.users.health-dashboard = {
      isSystemUser = true;
      group = "health-dashboard";
      description = "go-health-dashboard federation hub";
    };
    users.groups.health-dashboard = { };

    systemd.services.health-dashboard = {
      description = "Federated go-health hub";
      wantedBy = [ "multi-user.target" ];
      startLimitBurst = 5;
      startLimitIntervalSec = 300;
      serviceConfig = {
        Type = "simple";
        # Stateless network daemon: no home, no state dir; the only
        # secrets are the OIDC secret files read directly from their
        # deployment paths (grant the group read — see the oidc options).
        User = "health-dashboard";
        Group = "health-dashboard";
        ExecStart = lib.getExe cfg.package;
        Environment = [
          "HEALTH_HUB_ADDR=${cfg.addr}"
          "HEALTH_HUB_REMOTES=${lib.concatStringsSep "," cfg.remotes}"
        ]
        ++ lib.optional cfg.trend "HEALTH_HUB_TREND=1"
        ++ oidcEnv;
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
      }
      // egressFence;
    };
  };
}
