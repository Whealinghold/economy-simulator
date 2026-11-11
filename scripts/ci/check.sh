#!/usr/bin/env bash
# Job `check`: per module gofmt, vet, staticcheck, short race tests, gate R and go mod tidy.
# Runs the fast stage on every module, then gate R only if all passed (fail fast, testing.md 10.5);
# exits non-zero naming the modules that failed.
set -uo pipefail
# shellcheck source=common.sh
. "$(dirname "${BASH_SOURCE[0]}")/common.sh"

ensure_staticcheck || exit 1
"$CI_DIR/selftest.sh" || exit 1

gofmt_clean() {
	local out
	out=$(cd "$REPO_ROOT/$1" && gofmt -l .) || return 1
	[ -z "$out" ] || fail_in "gofmt -l lists files in $1:"$'\n'"$out"
}

# tidy_clean runs go mod tidy and restores go.mod and go.sum, failing if tidy changed them.
tidy_clean() {
	local dir=$REPO_ROOT/$1 tmp rc=0
	tmp=$(mktemp -d)
	cp "$dir/go.mod" "$tmp/"
	[ ! -f "$dir/go.sum" ] || cp "$dir/go.sum" "$tmp/"
	run_in "$1" go mod tidy || rc=1
	cmp -s "$dir/go.mod" "$tmp/go.mod" || rc=1
	if [ -f "$tmp/go.sum" ]; then cmp -s "$dir/go.sum" "$tmp/go.sum" || rc=1; else [ ! -f "$dir/go.sum" ] || rc=1; fi
	if [ "$rc" -ne 0 ]; then
		echo "FAIL: go mod tidy changes $1; diff against the committed files:" >&2
		diff -u "$tmp/go.mod" "$dir/go.mod" >&2
		[ ! -f "$tmp/go.sum" ] || diff -u "$tmp/go.sum" "$dir/go.sum" >&2
	fi
	cp "$tmp/go.mod" "$dir/go.mod"
	if [ -f "$tmp/go.sum" ]; then cp "$tmp/go.sum" "$dir/go.sum"; else rm -f "$dir/go.sum"; fi
	rm -rf "$tmp"
	return "$rc"
}

# check_fast is the fail-fast stage (testing.md 10.5): static checks, tidy and the short race run.
check_fast() {
	local m=$1
	echo "==> [$m] gofmt -l ."
	gofmt_clean "$m" || return 1
	tidy_clean "$m" || return 1
	run_in "$m" go vet ./... || return 1
	run_in "$m" staticcheck ./... || return 1
	run_in "$m" go test -short -race ./... || return 1
}

mods=$(modules) || exit 1
failed=()
for m in $mods; do
	check_fast "$m" || failed+=("$m")
done
# (The short run is a subset of R and so runs twice: that is the fail-fast order of testing.md 10.5.)
# Gate R runs only when every module passed the fast stage, so a cheap failure never waits for it.
if [ "${#failed[@]}" -eq 0 ]; then
	for m in $mods; do
		run_in "$m" go test -race -timeout 60m ./... || failed+=("$m")
	done
fi
if [ "${#failed[@]}" -ne 0 ]; then
	echo "check: failed modules: ${failed[*]}" >&2
	exit 1
fi
echo "check: all modules passed"
