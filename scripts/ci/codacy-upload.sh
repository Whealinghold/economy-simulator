#!/usr/bin/env bash
# codacy-upload.sh [profile...]   (default: coverage.out in the repository root)
#
# Uploads a Go coverage profile to Codacy with the coverage reporter, pinned below. The token is
# read from the environment (CODACY_PROJECT_TOKEN) and never put on a command line. Optional:
# CODACY_COMMIT_UUID sets the commit the coverage belongs to (the reporter reads git otherwise).
#
# Known risk: Codacy publishes a SHA512SUM per release asset, but only next to the asset on its
# GitHub release, so the pinned sum below is the check against a later change of the download, not
# against a compromised release. It was copied from the release page of REPORTER_VERSION on
# 2026-11-11, and the reporter was not run when the script was written (first use: plan-2-ci Phase 6).
# Only linux amd64 is pinned: the upload runs in the `full` job on ubuntu-24.04.
set -euo pipefail

REPORTER_VERSION=14.1.3
REPORTER_SHA512=73a6c93fc5db509fd0423d6e545759c3f61d8360ee12054e32a9277daab6add0da6c6639248f6ed6c0042366440293ebc0b6144325aef52c5d3d4b230d65eb17
REPORTER_URL=https://artifacts.codacy.com/bin/codacy-coverage-reporter/$REPORTER_VERSION/codacy-coverage-reporter-linux

if [ -z "${CODACY_PROJECT_TOKEN:-}" ]; then
	echo "codacy-upload: CODACY_PROJECT_TOKEN is empty; refusing to upload" >&2
	exit 1
fi
[ "$#" -gt 0 ] || set -- "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/coverage.out"
for profile in "$@"; do
	[ -s "$profile" ] || { echo "codacy-upload: $profile is missing or empty" >&2; exit 1; }
done
profiles=$(IFS=,; echo "$*")
[ "$(uname -sm)" = "Linux x86_64" ] || { echo "codacy-upload: only linux amd64 is pinned, this is $(uname -sm)" >&2; exit 1; }

# retry <command...> runs a command up to three times, ten seconds apart.
retry() {
	local n
	for n in 1 2 3; do
		"$@" && return 0
		echo "codacy-upload: attempt $n of 3 failed: $1" >&2
		[ "$n" -eq 3 ] || sleep 10
	done
	return 1
}

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
reporter=$work/codacy-coverage-reporter

retry curl --fail --silent --show-error --location --output "$reporter" "$REPORTER_URL"
echo "$REPORTER_SHA512  $reporter" | sha512sum --check --quiet - || {
	echo "codacy-upload: checksum of the downloaded reporter does not match the pinned one" >&2
	exit 1
}
chmod +x "$reporter"

args=(report --force-coverage-parser go -r "$profiles")
[ -z "${CODACY_COMMIT_UUID:-}" ] || args+=(--commit-uuid "$CODACY_COMMIT_UUID")
retry "$reporter" "${args[@]}"
