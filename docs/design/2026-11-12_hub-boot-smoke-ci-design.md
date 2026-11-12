# Hub boot-smoke CI design (M18 residual, 2026-11-12)

Status: DESIGN ONLY — implementation is parked on the compose-stack CI
policy decision (decision batch A, 2026-11-12 plan T24). The decision-safe
slice shipped instead: the `compose-build` CI job builds the dashboard image
on every push (the break class the one boot-verification caught), and the
`nix-build` job builds the health-hub FOD on every push.

## What the smoke must prove (that build-only cannot)

1. The FOD binary BOOTS and serves loopback traffic (the 2026-11-12 desk
   boot-smoke proved this once, by hand, on NixOS via a systemd transient
   unit — CI has no systemd).
2. `nixosModules.default` still produces a valid unit: option types flow
   (`remotes` → `HEALTH_HUB_REMOTES`), the sandbox stays intact, and the
   service activates.

## Runtime-scoped design

CI runners have Docker (no systemd). Three tiers, each gated on what the
runner can actually do:

- **Tier 0 (ships today):** `nix build .#health-hub` + `docker compose
  build` — compilation proof, no runtime claim.
- **Tier 1 (cheap boot):** run the FOD binary DIRECTLY on the runner
  (no container): `HEALTH_HUB_REMOTES=http://127.0.0.1:1` with a dark
  remote, poll `/health` until the `reachable` fail row appears, assert
  the 503 JSON contract, kill. Proves the binary boots and the federation
  contract holds — no systemd, no NixOS module. ~15s.
- **Tier 2 (full module):** `nixos-container` or a QEMU VM booting
  `nixosModules.default`. Needs privileged nesting (Docker-in-Docker or a
  VM inside the runner) — GitHub hosted runners cannot run
  nixos-container without a VM layer. This tier is where the policy
  decision matters (cost ≈ 3–5 min/run, self-hosted runner solves
  nesting).

## Recommendation

Adopt Tier 1 in CI now (no policy question — it boots the released binary,
not the module), keep Tier 2 behind the decision batch. The desk boot-smoke
procedure (systemd transient unit) stays the Tier-2 evidence record until
CI can run it mechanically.
