#!/usr/bin/env bash
# Re-run BuildFlow's failed steps with the result cache bypassed.
#
# BuildFlow cache greens re-verify inputs, not execution: during the
# 2026-10-30 FuzzCSVExport triage, a previously-failed test-race step
# "passed" in ~6s purely from cache while the real race suite takes minutes.
# A cache-served green proves inputs are unchanged since some earlier real
# run — it is NOT evidence that a fix works. Any re-run of a FAILED step
# goes through here (or sets BUILDFLOW_NO_RESULT_CACHE=1 directly); for raw
# go tests the equivalent is -count=1.
#
# Run: nix develop -c bash scripts/rerun-failed-uncached.sh [extra args...]
#   e.g. nix develop -c bash scripts/rerun-failed-uncached.sh --verbose
set -euo pipefail

export BUILDFLOW_NO_RESULT_CACHE=1
exec buildflow --failed-only "$@"
