#!/usr/bin/env bash
# Shared by the CI job scripts; sourced, not run. Every job is one script call from the workflow,
# and each script also runs on a developer machine from any directory.

CI_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$CI_DIR/../.." && pwd)

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
		if [ "$dir" = "$REPO_ROOT" ]; then echo .; else echo "${dir#"$REPO_ROOT"/}"; fi
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
		PATH="$(go env GOPATH)/bin:$PATH"
		export PATH
	fi
}
