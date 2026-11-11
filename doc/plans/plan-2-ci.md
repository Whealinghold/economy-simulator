# Plan: Stage 2, row `ci.yml`: the CI workflow

**Status: Proposed (2026-11-11).** Builds the roadmap row "`.github/workflows/ci.yml` (three jobs: check, full with the Codacy coverage upload, and arm64)" of `roadmap-v1.md` Stage 2. Contract: the owner's decision of 2026-10-23 (CI yes, with an arm64 job) and the decision memo `DEC-ci.md` (supervisor scratchpad, `s2/DEC-ci.md`); its picks are the Decisions below. The plan writes no module code and moves no golden.

Rule tags read when the plan was written: architecture=0052d4050fd8b5aa5325ae2e, observability=e7cfbd33b1620a09dc78dff7, testing=120890253bd2f6c110a015d2. Notes tag at writing: bb1aecc236a36f9a48ffd6a2.

Specs implemented: none. Serves `testing.md` 9.1 (CI MUST run R and F and fail below the thresholds), 7.1 and `architecture.md` 7.4 and 14.1 (two architectures; gofmt, vet, staticcheck, race, tidy in CI), NFR-WLD-004 and `vision.md` 5 item 2 (the byte tests pass on amd64 and arm64). Requirement IDs touched: NFR-WLD-004 (the CI arm64 job is its evidence; it already exists in `spec-world.md`).

Notes read, with disposition: N-019 (its Stage 0 updates ask for a CI: apply, Phase 5 closes the CI part), N-243 (the Codacy coverage upload: apply, Phase 4; the Lizard and markdownlint items are not this plan's), N-343 (arm64 never shown: the first green arm64 run is its evidence, Phase 6 updates the note; close it only then). Left open: every other ALL/CODE/REF note (this plan adds no field, event, site or name).

Decision block: D-CI-1 to D-CI-7 of the memo, restated below as D1 to D7.

## Scope

In:

- `scripts/ci/` (new): `common.sh`, `check.sh`, `arm64.sh`, `full.sh`, `coverage-check.sh`, `coverage-thresholds.txt`, `codacy-upload.sh`, `selftest.sh`, `testdata/`.
- `.github/workflows/ci.yml`, `.github/dependabot.yml` (new).
- Documents: `AGENTS.md` (Layout, Commands), `doc/rules/architecture.md` 2.6 and 14.1, `doc/rules/testing.md` 9.1, `doc/roadmap-v1.md`, `doc/vision.md` item 2, `doc/notes.md` (N-019, N-243, N-343).
- Owner actions (Phase 6): the `CODACY_PROJECT_TOKEN` secret, the label `full`, branch protection, fork-approval setting.

Out: Lizard, markdownlint and `govulncheck` in CI (Codacy runs Lizard and markdownlint; `govulncheck` is not required by any rule: tracked as a candidate in N-019's Stage 0 update); timed fuzz runs (N-019/N-243 mention them; no fuzz time is decided, add a row when it is); mutation testing (`gomu` runs only in Docker, testing.md 9.4); sharding gate F (only if Phase 6 measures it as needed, D4); the coverage threshold of `examples/` modules (testing.md 9.1 says "their own", no value is set anywhere: left as an open item, the check reports their coverage and gates none).

## Decisions

- **D1 (runners).** amd64 jobs run on `ubuntu-24.04`, arm64 on the GitHub-hosted native `ubuntu-24.04-arm` (free and unlimited, 4 vCPU, for a public repository). Rejected: QEMU emulation (9 times slower, measured, and it is not hardware); self-hosted. Seam: the `runs-on:` line of the arm64 job; `ARM_EMULATE=1` in `arm64.sh` installs binfmt for the fallback.
- **D2 (jobs).** `check` (every PR and push; `timeout-minutes: 180`): a fast stage per module (`gofmt -l` empty, `go mod tidy` with no diff, `go vet`, `staticcheck` 2026.2.2, `go test -short -race`), then R per module only if every module passed the fast stage (testing.md 10.5 fail-fast order; the short run is a subset of R and so runs twice), after `selftest.sh`. `arm64` (every PR and push): `go build` and `go vet` of every module, then the quick pattern of testing.md 9.1 (all seeds, no race, no coverage) in the core and `go test -short` in the other modules. `full`: F per module (each module needs a coverage policy: `.` and `otelobserver` are gated, `examples/*` reported, any other module fails before its tests; every package of a gated module must have a threshold row, checked before F), the coverage check, artifact, Codacy (also for a run whose coverage check failed, never for a failed or partial test run). Modules are read from `go work edit -json`, so a new example needs no edit. Each job is one script call.
- **D3 (schedule).** `full` runs on `push` to `main`, a nightly `schedule`, `workflow_dispatch`, and a `pull_request` that carries the label `full`. `check` and `arm64` skip on a `labeled` event. Concurrency: workflow level, one group per ref for PR events other than `labeled` (a newer push cancels the older run); every other run (main, nightly, dispatch, `labeled`) has a unique group, because GitHub keeps one pending run per group and a long `full` must not displace another run. `full` has its own job-level group (per ref for a PR, cancel in progress, so opened plus labeled, or labeled plus a push, runs it once). Seam: the `on:` block and the one `if:` of `full`.
- **D4 (F).** One job, `timeout-minutes: 330` (hosted cap 360), `F_TIMEOUT` defaults to the rule's `120m`. If the first measured run exceeds 280 minutes, a follow-up builds the shard; if the root package needs more than `120m`, the owner amends testing.md 9.1: CI does not diverge from the rule. Required for merge: `check` and `arm64`. `full` is the gate of record on `main` and nightly.
- **D5 (logic in scripts).** Bash scripts in `scripts/ci/`, shellcheck-clean, runnable on a developer machine; thresholds are data in `coverage-thresholds.txt` (the table of testing.md 9.1). Per-package coverage is computed from the profile with awk (block de-duplicated, covered if any count is above zero), because `go tool cover -func` does not give per-package figures.
- **D6 (supply chain).** `actions/checkout` v7.0.1 (`3d3c42e5aac5ba805825da76410c181273ba90b1`) and `actions/setup-go` v7.0.0 (`b7ad1dad31e06c5925ef5d2fc7ad053ef454303e`), pinned by SHA with a version comment; no other third-party action. Dependabot (`github-actions`, weekly). `permissions: contents: read`; `persist-credentials: false`. Go from `go-version-file: go.work` with `GOTOOLCHAIN=local`. Codacy by script, not by its action (last release 2022-01-19, composite, interpolates inputs into shell).
- **D7 (forks).** `pull_request` only, never `pull_request_target`. The token is read only in the upload step, which runs only when the event is not a pull request or the head repository is this one, and the secret is non-empty. `coverage.out` is uploaded as an artifact before the Codacy step; the upload retries three times.

## Phases

### Phase 1: the coverage checker (test first)

Goal: a script that turns a coverage profile into a pass or fail per scope of testing.md 9.1.

Files: `scripts/ci/coverage-check.sh`, `scripts/ci/coverage-thresholds.txt`, `scripts/ci/selftest.sh`, `scripts/ci/testdata/*.out`.

Steps: (1) write `selftest.sh` and its fixture profiles first: a profile that passes, one where one `internal/` package is at 89.9%, one where the root is under 90%, one with duplicate blocks (covered in one copy only), an empty profile, and a malformed line, an unreadable source file and block comments before the package clause; the test asserts exit status and the message names the scope. (2) Write `coverage-check.sh` (`coverage-check.sh <profile> <thresholds>`): thresholds file has `<path prefix> <min percent>` rows (root `.`, each `internal/*` package, `config`, `strategy`, `indicator`, `experiment`, `denomination`; `otelobserver` is checked with its own profile and 80). An unlisted package fails (so a new `internal/` package cannot slip in), except `main` packages and generated files, which testing.md 9.1 excludes (a selftest case covers both), and an empty profile fails. Integer arithmetic on counts, thresholds compared as `covered*100 >= min*total`.

Acceptance: `scripts/ci/selftest.sh` passes, and each fixture case fails when the checker is broken on purpose (flip the comparison and see two cases fail, testing.md 9.3). `shellcheck scripts/ci/*.sh` is clean.

### Phase 2: the job scripts

Goal: the three jobs exist as scripts that run on this host.

Files: `scripts/ci/common.sh`, `check.sh`, `arm64.sh`, `full.sh`.

Steps: `common.sh` holds the pinned staticcheck version, `modules()` (reads `go work edit -json`) and a `run_in()` helper that prints the module and the command. `check.sh` runs D2's check list: the fast stage on every module (a failing module does not stop the others), then R if none failed. `arm64.sh` reads the quick pattern from one variable that equals the pattern in testing.md 9.1 (a guard in `selftest.sh` greps testing.md and fails if they differ). `full.sh` runs F per module with `F_TIMEOUT`, writes `coverage.out` (core) and `otelobserver/coverage.out`, then calls `coverage-check.sh`.

Acceptance: on this host `scripts/ci/check.sh` and `scripts/ci/arm64.sh` pass (arm64.sh runs natively amd64 here; the arm64 result is Phase 6). `scripts/ci/full.sh` is run with `F_TIMEOUT` on one small module only (`examples/colony`) as a smoke test; the full F run is Phase 6's.

### Phase 3: the Codacy upload script

Files: `scripts/ci/codacy-upload.sh`.

Steps: downloads the reporter at a pinned version and runs it with `--force-coverage-parser go -r coverage.out`; refuses to run with an empty `CODACY_PROJECT_TOKEN`; retries three times. Find out whether Codacy publishes a checksum or signature for the reporter binary; if it does, verify it, if not, record the version and the missing check in the file's header and in N-243's update (a known risk).

Acceptance: with an empty token the script exits non-zero with a clear message (selftest case); with a token it is exercised only in Phase 6. shellcheck clean.

### Phase 4: the workflow

Files: `.github/workflows/ci.yml`, `.github/dependabot.yml`.

Steps: write the three jobs of D1 to D7: `permissions: contents: read` at workflow level; each job `checkout` (SHA, `persist-credentials: false`), `setup-go` (SHA; cache on `check` and `arm64` keyed on every `go.sum`, off on `full`), one script call; `full` uploads `coverage.out` as an artifact, then the Codacy step with the `if:` of D7 and the secret in that step's `env:` only. Lint: `go run github.com/rhysd/actionlint/cmd/actionlint@<pinned version>` (run by hand here, not a CI step) is clean.

Acceptance: actionlint clean; the YAML holds no logic beyond triggers, `if:` conditions, runners and script calls (review check: no `run:` of more than one line except the script call).

### Phase 5: documents (the same change as the workflow)

Files: `AGENTS.md`, `doc/rules/architecture.md`, `doc/rules/testing.md`, `doc/roadmap-v1.md`, `doc/vision.md`, `doc/notes.md`.

Steps: `AGENTS.md` Layout and `architecture.md` 2.6 list `.github/` and `scripts/ci/` with the same entries; `AGENTS.md` Commands says the scripts run the gates. `testing.md` 9.1 gains one sentence: CI runs R and the quick run on every PR, F on `main`, nightly and labelled PRs, and a PR that touches code or tests carries the label `full` (this answers testing.md 11 item 8). `roadmap-v1.md` row `planned` to `built` and the "not built" bullet loses the workflow; `vision.md` item 2 states the workflow exists; N-019, N-243 get dated updates (N-343 only a pointer until Phase 6's green arm64 run). Each edited rule file and `notes.md` gets a fresh random tag in the same change.

Acceptance: `grep` shows `AGENTS.md` and `architecture.md` 2.6 name the same entries; every edited tagged file has a new tag; markdown tables use the spaced separator.

### Phase 6: first runs and owner actions

Steps: (owner) add the secret `CODACY_PROJECT_TOKEN`, the label `full`, branch protection requiring `check` and `arm64`, and the setting "require approval for all outside collaborators". Push the branch, run `workflow_dispatch`, and record: the F wall time on the 4-vCPU runner, whether the root package stayed within `120m`, the arm64 result, and whether Codacy shows the coverage. Apply D4's rule to the measurement. Update N-343 (close it if arm64 is green, with the run URL) and `vision.md` 5 item 2.

Acceptance: a green `arm64` run on the default-branch workflow; a recorded F time; Codacy shows a coverage figure for the commit; if any of these fails the row stays `planned` and the finding goes to the notes.

## Catalogue appends

None: no Go file, no field, no event, no checkpoint tag.

## Cross-plan contract

Consumes: gates R and F and the quick pattern (`testing.md` 9.1) and `gateFPrefixes` (`scenario_helpers_test.go`) <- plan-2-1; `go.work` module list <- plan-2-22 and plan-2-24. Provides: `scripts/ci/` and the workflow -> every later row (the gate they run), and the evidence for `vision.md` 5 item 2 -> the Stage 2 sign-off (plan-2-24 D694).

## Testing

| Requirement | Test(s) |
| --- | --- |
| testing.md 9.1 thresholds | `scripts/ci/selftest.sh` fixture cases (Phase 1) |
| testing.md 9.1 quick pattern equals CI's | the `selftest.sh` guard (Phase 2) |
| NFR-WLD-004 | the first green `arm64` run (Phase 6); the byte tests themselves already exist |
| architecture.md 14.1 | `check` job |

The lowest level is a shell self-test; the pyramid's unit level here is `selftest.sh`, the end-to-end level is the first workflow run. No Go test is added, none is changed. Golden files: none change. Fuzz: none (the checker parses a trusted local file; malformed input is a selftest case). Coverage: unchanged.

## Observability

None, because no library code changes. The CI itself: every script prints the module and command it runs and exits non-zero with the failing scope named; the nightly run is the alarm for a red `main` (GitHub emails the repository's notifications); `coverage.out` is kept as an artifact.

## Security review

- Trust boundary: a PR from a fork runs `check` and `arm64` with a read-only token and no secrets (`pull_request`, not `pull_request_target`). Worst case is a hostile test burning runner minutes (free on a public repository, capped by `timeout-minutes`).
- The one secret, `CODACY_PROJECT_TOKEN`, is exposed only to the upload step, only on events with trusted code (push, schedule, dispatch, or a PR whose head repository is this one). A same-repo PR author can already push to the repository.
- Supply chain: two SHA-pinned actions, tools installed with `go install <path>@<version>` (checked by the module proxy's checksum database), the Codacy reporter at a pinned version with the checksum question of Phase 3. The reporter is the weakest point and is stated as such.
- Script injection: workflow expressions are never interpolated into `run:` text; scripts read inputs from the environment.
- Resource exhaustion: every job has `timeout-minutes`; PR runs cancel superseded runs.

## Compatibility and rollout

Adds files; changes no API. Rollout: merge the scripts and the workflow in one PR, see `check` and `arm64` green on that PR, then the owner turns on branch protection (before that, a red job blocks nothing). If `arm64` or `full` is red for a reason in the CI itself, the job is removed from branch protection, not from the file, and the fix follows. Rollback: delete `.github/workflows/ci.yml`; the scripts stay usable by hand as the commands of `AGENTS.md` are today.

## Risks and open items

1. **R1.** F on a 4-vCPU runner may exceed 120m for the root package (the dev host needed about 60 minutes); then the owner must amend testing.md 9.1 or approve a larger runner. Decided by the measurement of Phase 6, not before.
2. **R2.** PRs without the label `full` have no coverage in Codacy (MissingRequirements, N-243). Owner decision recorded in D3: opt-in by label; alternative is F on every PR.
3. **R3.** The arm64 runner is free only while the repository is public (2 vCPU and billed minutes if private).
4. **R4.** Codacy's Go path mapping from import-path profile lines is not verified; Phase 6 checks it.
5. **R5.** No threshold exists for `examples/` coverage (testing.md 9.1): reported, not gated.
6. **R6.** The CI toolchain is Go 1.26.10 (from `go.work`) while the core module declares `go 1.25` and the pinned vectors were computed on 1.25; Phase 6 must show the byte tests green on that toolchain on both architectures, and a failure there is a behaviour change to explain (architecture.md 7.4), not a CI fix.
