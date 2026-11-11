#!/usr/bin/env bash
# Job `arm64`: build and vet every module, then the quick run in the core and go test -short
# elsewhere. Runs natively on an arm64 host. ARM_EMULATE=1 is the fallback for an amd64 host:
# GOARCH=arm64 with the tests run under qemu-user (slow; the native runner is the decision, D1).
set -uo pipefail
# shellcheck source=common.sh
. "$(dirname "${BASH_SOURCE[0]}")/common.sh"

test_exec=()
if [ "${ARM_EMULATE:-}" = 1 ]; then
	command -v qemu-aarch64-static >/dev/null || { sudo apt-get install -y qemu-user-static || exit 1; }
	export GOARCH=arm64
	test_exec=(-exec qemu-aarch64-static)
fi

arm_module() {
	local m=$1
	run_in "$m" go build ./... || return 1
	run_in "$m" go vet ./... || return 1
	if [ "$m" = . ]; then
		run_in "$m" go test "${test_exec[@]}" -timeout 60m -run "$QUICK_PATTERN" ./...
	else
		run_in "$m" go test "${test_exec[@]}" -short ./...
	fi
}

echo "arm64: $(go env GOOS)/$(go env GOARCH)"
failed=()
for m in $(modules); do
	arm_module "$m" || failed+=("$m")
done
if [ "${#failed[@]}" -ne 0 ]; then
	echo "arm64: failed modules: ${failed[*]}" >&2
	exit 1
fi
echo "arm64: all modules passed"
