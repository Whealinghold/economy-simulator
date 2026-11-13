#!/usr/bin/env bash
# Shared by the gate scripts; sourced, not run. They run on the dev host from any directory
# (there is no hosted CI workflow; testing.md 9.1).

CI_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
REPO_ROOT=$(cd "$CI_DIR/../.." && pwd -P)

# Pinned: installed with go install when the one on PATH differs (checked by the module proxy).
STATICCHECK_VERSION=2026.2.2

# The quick run of doc/rules/testing.md 9.1; selftest.sh fails if the two differ.
export QUICK_PATTERN='TestScenario|TestDeterminism|TestGolden|TestInvariants|TestCheckpoint|TestFork|TestConfigOnly|TestFixture|TestPacks|TestCensus|TestExpected|TestStylised|TestCost|TestStable'

# modules prints the directory of every module of go.work, relative to the repository root.
# It fails when go list fails or lists nothing: a loop over no modules would pass every gate.
modules() {
	local dirs dir
	dirs=$(cd "$REPO_ROOT" && go list -m -f '{{.Dir}}') && [ -n "$dirs" ] || { echo "modules: go list -m failed or found no module" >&2; return 1; }
	while IFS= read -r dir; do
		if [ "$dir" = "$REPO_ROOT" ]; then dir=.; else dir=${dir#"$REPO_ROOT"/}; fi
		# the gate scripts word-split the list, so a relative path with whitespace would break them
		case $dir in *[[:space:]]*) echo "modules: directory with whitespace is not supported: $dir" >&2; return 1 ;; esac
		echo "$dir"
	done <<<"$dirs"
}

# run_in <module> <command...> prints the module and the command, then runs it there.
run_in() {
	local mod=$1
	shift
	echo "==> [$mod] $*"
	(cd "$REPO_ROOT/$mod" && "$@")
}

# fail_in <message> reports a failure and returns 1, so a step can end with `|| return 1`.
fail_in() { echo "FAIL: $*" >&2; return 1; }

# ensure_staticcheck installs the pinned staticcheck unless the one on PATH reports it.
ensure_staticcheck() {
	if ! staticcheck -version 2>/dev/null | grep -qF "staticcheck $STATICCHECK_VERSION "; then
		echo "==> installing staticcheck $STATICCHECK_VERSION"
		go install "honnef.co/go/tools/cmd/staticcheck@$STATICCHECK_VERSION" || return 1
		PATH="$(go env GOBIN):$(go env GOPATH)/bin:$PATH"
		export PATH
	fi
}

# gated_rows_present fails before the hours-long run when a package of a gated module has no row
# in the thresholds file, which coverage-check.sh would only report afterwards.
gated_rows_present() {
	local m=$1 base ip name nfiles rel rc=0 pkgs
	base=$(awk '$1 == "base" { print $2 }' "$CI_DIR/coverage-thresholds.txt")
	pkgs=$(cd "$REPO_ROOT/$m" && go list -f '{{.ImportPath}} {{.Name}} {{len .GoFiles}}' ./...) || { echo "FAIL: [$m] go list failed" >&2; return 1; }
	while read -r ip name nfiles; do
		[ "$name" != main ] && [ "$nfiles" -gt 0 ] || continue
		rel=${ip#"$base"}
		rel=${rel#/}
		awk -v r="${rel:-.}" '$1 == r { f = 1 } END { exit !f }' "$CI_DIR/coverage-thresholds.txt" || { echo "FAIL: [$m] package ${rel:-.} has no row in coverage-thresholds.txt" >&2; rc=1; }
	done <<<"$pkgs"
	return "$rc"
}
