#!/usr/bin/env bash
# Self-test of the CI scripts: fixture profiles through coverage-check.sh, the guard that
# the quick pattern equals doc/rules/testing.md 9.1, and the Codacy script's refusals.
# Run from anywhere; set CHECKER to test another copy of the checker.
set -uo pipefail
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)
td=$here/testdata
checker=${CHECKER:-$here/coverage-check.sh}
fails=0

# expect <name> <exit status> <output substring> <command...>
expect() {
	local name=$1 want=$2 sub=$3 out got
	shift 3
	out=$("$@" 2>&1); got=$?
	if [ "$got" -ne "$want" ] || [[ $out != *"$sub"* ]]; then
		echo "FAIL $name: exit $got (want $want), output wanted to contain '$sub':"
		printf "    %s\n" "${out//$'\n'/$'\n'    }"
		fails=$((fails + 1))
	else
		echo "ok   $name"
	fi
}
check() { "$checker" "$td/$1" "$td/thresholds.txt" "$td/src"; }

expect pass                    0 "ok   .: 90.0%"            check pass.out
expect internal-89.9           1 "FAIL internal/a: 89.9%"   check internal_low.out
expect root-under-90           1 "FAIL .: 89.0%"            check root_low.out
expect duplicate-blocks        0 "ok   .: 100.0%"           check dup.out
expect duplicate-uncovered     1 "FAIL .: 50.0%"            check dup_uncovered.out
expect unlisted-package        1 "FAIL internal/new"        check unlisted.out
expect main-and-generated      0 "ok   .: 100.0%"           check excluded.out
expect malformed-line          2 "malformed profile line 3" check malformed.out
expect malformed-count         2 "malformed profile line 2" check badcount.out
expect block-comment-package   0 "ok   .: 100.0%"           check blockcomment.out
expect block-comment-then-main  0 ""          check blockcomment_main.out
expect missing-source           2 "cannot read source"       check nosrc.out
expect empty-profile           2 "empty profile"            check empty.out
expect zero-byte-profile       2 "empty profile"            check zero.out
expect otel-79                 1 "FAIL otelobserver: 79.0%" check otel_low.out
expect otel-80                 0 "ok   otelobserver: 80.0%" check otel_ok.out

# The thresholds file must name every package directory under internal/ and the other
# scopes of testing.md 9.1, so a new package cannot pass unchecked.
for d in "$root"/internal/*/; do
	expect "threshold-row-$(basename "$d")" 0 "internal/$(basename "$d") " grep -F "internal/$(basename "$d") " "$here/coverage-thresholds.txt"
done

# The quick pattern of the arm64 job equals the one in doc/rules/testing.md 9.1.
if [ -r "$here/common.sh" ]; then
	# shellcheck source=common.sh
	. "$here/common.sh"
	expect quick-pattern-equals-testing-md 0 "" grep -qF -- "-run '$QUICK_PATTERN' ./..." "$root/doc/rules/testing.md"
fi

# The Codacy upload refuses to run without a token.
if [ -x "$here/codacy-upload.sh" ]; then
	expect codacy-empty-token 1 "CODACY_PROJECT_TOKEN is empty" env -u CODACY_PROJECT_TOKEN "$here/codacy-upload.sh" "$td/pass.out"
	expect codacy-blank-token 1 "CODACY_PROJECT_TOKEN is empty" env CODACY_PROJECT_TOKEN= "$here/codacy-upload.sh" "$td/pass.out"
	expect codacy-comma-path 1 "contains a comma" env CODACY_PROJECT_TOKEN=x "$here/codacy-upload.sh" "$td/a,b.out"
	expect codacy-bad-commit 1 "not a 40-character" env CODACY_PROJECT_TOKEN=x CODACY_COMMIT_UUID='abc; id' "$here/codacy-upload.sh" "$td/pass.out"
	expect codacy-short-commit 1 "not a 40-character" env CODACY_PROJECT_TOKEN=x CODACY_COMMIT_UUID=abc123 "$here/codacy-upload.sh" "$td/pass.out"
	expect codacy-long-commit 1 "not a 40-character" env CODACY_PROJECT_TOKEN=x CODACY_COMMIT_UUID=0123456789abcdef0123456789abcdef012345678 "$here/codacy-upload.sh" "$td/pass.out"
fi

# A failing go list must not read as "no modules, all passed".
expect modules-fail-closed 1 "go list -m failed" bash -c ". '$here/common.sh'; PATH=/nonexistent; modules"

if [ "$fails" -ne 0 ]; then echo "selftest: $fails case(s) failed"; exit 1; fi
echo "selftest: all cases passed"
