#!/usr/bin/env bash
# Cold-cache behavior test for scripts/check-ui-pins.sh (M24, 2026-11-12).
#
# The pins guard must be order-independent: on a fresh CI runner or clean
# checkout its module cache is empty, so it must either self-heal (download
# the pinned module) or fail LOUDLY when download is impossible — never
# pass on missing evidence, never die on a misleading error. This script
# pins both behaviors by pointing GOMODCACHE at throwaway dirs:
#
#   leg 1 (hermetic, always runs): empty cache + GOPROXY=off → the guard
#        must exit nonzero with ::error:: output. Mutation-checked: this
#        leg fails if the bundle-missing error path is removed.
#   leg 2 (network): empty cache + working network → the guard must
#        self-heal via 'go mod download' and pass. Skips with a named
#        reason when the network cannot serve the pinned module (offline
#        desk runs stay green; CI's Hygiene job runs this leg online).
#
# Exit codes: 0 clean (leg 2 may have skipped), 1 a pinned behavior broke.
set -euo pipefail

cd "$(dirname "$0")/.."

scratch=$(mktemp -d)
cleanup() {
	trash-put "$scratch" 2>/dev/null || true
}
trap cleanup EXIT

leg1_out="$scratch/leg1.log"

# --- Leg 1: download impossible → loud failure --------------------------
# GOVCS=off is required alongside GOPROXY=off: this host carries
# GOPRIVATE=github.com/larsartmann/*, and with the proxy off the go
# command falls back to DIRECT git VCS for those modules (observed
# 2026-11-12) — without GOVCS=off the "offline" leg would secretly
# download and pass.
if GOMODCACHE="$scratch/off" GOPROXY=off GOVCS='*:off' bash scripts/check-ui-pins.sh >"$leg1_out" 2>&1; then
	echo "::error::check-ui-pins.sh PASSED with an empty GOMODCACHE and GOPROXY=off — the guard passed on missing evidence; the cold-cache loud-failure path is dead"
	exit 1
fi

if ! grep -q "::error::" "$leg1_out"; then
	echo "::error::check-ui-pins.sh failed cold+offline but emitted no ::error:: lines — it died on a raw tool error instead of failing loudly:"
	sed 's/^/       /' "$leg1_out" >&2
	exit 1
fi

echo "OK  leg 1: cold cache + GOPROXY=off fails loudly (::error:: output, nonzero exit)"

# --- Leg 2: network present → self-heal ---------------------------------
# Probe whether the pinned static module is servable into a throwaway
# cache before committing to the full run; an offline desk must skip with
# a reason, not hang or red.
heal_cache="$scratch/heal"

if ! GOMODCACHE="$heal_cache" go mod download github.com/larsartmann/go-datastar/static 2>"$scratch/netprobe.log"; then
	echo "SKIP leg 2: the pinned module cannot be downloaded here (offline desk?) — the self-heal leg runs in CI's Hygiene job"
	exit 0
fi

if ! GOMODCACHE="$heal_cache" bash scripts/check-ui-pins.sh >"$scratch/leg2.log" 2>&1; then
	echo "::error::check-ui-pins.sh FAILED with a cold-but-servable GOMODCACHE — the self-heal path is broken:"
	sed 's/^/       /' "$scratch/leg2.log" >&2
	exit 1
fi

echo "OK  leg 2: cold cache self-heals (downloads the pinned module, all pins green)"
