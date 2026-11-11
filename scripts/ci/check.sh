#!/usr/bin/env bash
# Job `check`: per module gofmt, vet, staticcheck, short race tests, gate R and go mod tidy.
# Runs every module and exits non-zero naming the modules that failed.
set -uo pipefail
# shellcheck source=common.sh
. "$(dirname "${BASH_SOURCE[0]}")/common.sh"

ensure_staticcheck || exit 1

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

check_module() {
	local m=$1
	echo "==> [$m] gofmt -l ."
	gofmt_clean "$m" || return 1
	run_in "$m" go vet ./... || return 1
	run_in "$m" staticcheck ./... || return 1
	run_in "$m" go test -short -race ./... || return 1
	run_in "$m" go test -race -timeout 60m ./... || return 1
	tidy_clean "$m" || return 1
}

failed=()
for m in $(modules); do
	check_module "$m" || failed+=("$m")
done
if [ "${#failed[@]}" -ne 0 ]; then
	echo "check: failed modules: ${failed[*]}" >&2
	exit 1
fi
echo "check: all modules passed"
