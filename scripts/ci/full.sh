#!/usr/bin/env bash
# Job `full`: gate F per module (all seeds, atomic coverage, no race), then the coverage check.
# F_TIMEOUT sets the go test timeout (default: the rule's 120m, testing.md 9.1). FULL_MODULES
# restricts the modules (space-separated, relative to the repository root; for a smoke test).
# Writes <module>/coverage.out. The core and otelobserver are gated by coverage-check.sh; the
# examples are reported, not gated (no threshold is set for them).
set -uo pipefail
# shellcheck source=common.sh
. "$(dirname "${BASH_SOURCE[0]}")/common.sh"

: "${F_TIMEOUT:=120m}"
# shellcheck disable=SC2086 # FULL_MODULES is a space-separated list
mods=${FULL_MODULES:-$(modules)} || exit 1

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

failed=()
for m in $mods; do
	case $m in . | otelobserver) gated_rows_present "$m" || { failed+=("$m thresholds"); continue; } ;; esac
	rm -f "$REPO_ROOT/$m/coverage.out" # a stale profile must never be checked or uploaded
	if ! run_in "$m" go test -covermode=atomic -coverpkg=./... -coverprofile=coverage.out -timeout "$F_TIMEOUT" ./...; then
		failed+=("$m")
		continue
	fi
	case $m in
		. | otelobserver)
			"$CI_DIR/coverage-check.sh" "$REPO_ROOT/$m/coverage.out" "$CI_DIR/coverage-thresholds.txt" "$REPO_ROOT" || failed+=("$m coverage")
			;;
		*) (cd "$REPO_ROOT/$m" && echo "==> [$m] $(go tool cover -func=coverage.out | tail -n 1) (reported, not gated)") ;;
	esac
done
if [ "${#failed[@]}" -ne 0 ]; then
	echo "full: failed: ${failed[*]}" >&2
	exit 1
fi
echo "full: all modules passed"
