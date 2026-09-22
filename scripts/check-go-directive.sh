#!/usr/bin/env bash
# Go-directive floor guard.
#
# Fails unless go.mod's `go` directive is exactly the audited floor.
#
# History: the fleet BuildFlow runner's go-mod-normalize step keeps
# downgrading `go 1.27.1` to `go 1.27` (twice on 2026-09-22 alone), and
# every module load against go-health's `go >= 1.27.1` floor then fails
# with a bare `go: updates to go.mod needed`. The downgrade is a normalize
# artifact, not a decision — this guard turns it into a loud failure
# instead of a mysterious package-load error.
#
# BUMP POLICY: the floor is dep-driven (go-health v0.4.0 requires
# go >= 1.27.1; nixpkgs ships go_1_27 = 1.27.1, the flake builds with it,
# and CI's setup-go reads go-version-file: go.mod). When the floor moves,
# update `expected_go_directive` below and go.mod IN THE SAME CHANGE.
#
# RESTORE PATTERN (when the guard catches the downgrade locally): the
# edit + git add + commit must be one atomic tool call so the auto-daemon
# cannot sweep a torn state:
#   sed -i 's/^go 1\.27$/go 1.27.1/' go.mod && git add go.mod && git commit -m "fix(deps): restore go 1.27.1 directive"
set -euo pipefail

expected_go_directive="1.27.1"

actual=$(sed -n 's/^go \([0-9][0-9.]*\)$/\1/p' go.mod | head -n1)
if [[ "$actual" == "$expected_go_directive" ]]; then
	echo "OK  go.mod go directive: $actual"
else
	echo "::error::go.mod go directive is '${actual:-<missing>}', expected '${expected_go_directive}' — the fleet BuildFlow go-mod-normalize step downgrades the directive and breaks every module load against go-health's floor; restore with sed -i 's/^go 1\.27\$/go 1.27.1/' go.mod, then git add + commit atomically. See scripts/check-go-directive.sh"
	exit 1
fi
