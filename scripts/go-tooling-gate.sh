#!/usr/bin/env bash
# Go-tooling gate — refuse Go tooling while a buildflow run is active in
# THIS repository. Buildflow steps mutate go.mod and *_templ.go mid-flight,
# so any Go invocation in that window reads or writes torn state; the rule
# was attention-enforced and failed twice in two consecutive sessions
# (2026-11-04/05: `go vet` during an active run, then an amend that swept
# buildflow-authored drift). This script makes the rule mechanical.
#
# Attribution is by /proc/PID/cwd: a buildflow run rooted in ANOTHER repo
# (the common case under parallel sessions) is informational only and does
# NOT block here — those runs mutate their own trees. A run whose cwd is
# inside this repo tree exits 1 naming the blocking PIDs. A matched PID
# whose cwd cannot be read is assumed blocking (fail-closed); one whose
# /proc entry vanished between pgrep and the read is skipped (exited).
#
# All output goes to stderr so `nix develop -c <tool>` stdout stays
# parseable (devShell-banner convention).
#
# Wiring: prepended to every flake app (mkApp), the devShell templ wrapper,
# the devShell `go` shell function, and pre-push check 0.
#
# Run: scripts/go-tooling-gate.sh
# Exit codes: 0 = no repo-local buildflow run; 1 = run active (or cwd
#             unreadable for a matched PID).
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)

matched=$(pgrep -f buildflow || true)
if [ -z "$matched" ]; then
	echo "go-tooling-gate: clear — no buildflow processes at all" >&2
	exit 0
fi

blocking=""
other=""
for pid in $matched; do
	cwd=$(readlink "/proc/$pid/cwd" 2>/dev/null) || {
		if [ -d "/proc/$pid" ]; then
			echo "go-tooling-gate: PID $pid matches 'buildflow' but its cwd is unreadable — assuming blocking (fail-closed)" >&2
			blocking="$blocking $pid"
		fi
		continue
	}
	case "$cwd" in
	"$repo_root" | "$repo_root"/*)
		blocking="$blocking $pid"
		;;
	*)
		other="$other $pid:$cwd"
		;;
	esac
done

if [ -n "$other" ]; then
	echo "go-tooling-gate: buildflow in OTHER repos (informational, not blocking):" >&2
	for entry in $other; do
		echo "  $entry" >&2
	done
fi

if [ -z "$blocking" ]; then
	echo "go-tooling-gate: clear — no buildflow run rooted in $repo_root" >&2
	exit 0
fi

echo "::error::go-tooling-gate: buildflow run is ACTIVE in this repository — Go tooling would read/write torn go.mod and *_templ.go state" >&2
for pid in $blocking; do
	ps -p "$pid" -o pid=,etime=,args= 2>/dev/null | cut -c1-160 >&2 ||
		echo "  $pid (exited during reporting)" >&2
done
echo "Wait for the run to finish ('buildflow history' shows state) or do collision-light work meanwhile. Docs, scripts, and git reads are safe; nothing that execs go/golangci-lint/templ is." >&2
exit 1
