package economysim_test

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packOneInfrastructure is pack one with the infrastructure module alone on: overlay/infrastructure.toml (plan
// 3-cap Phase 10), the module-alone golden's configuration (spec-reference-chain.md 4.5).
var packOneInfrastructure = append(slices.Clone(packOne), "overlay/infrastructure.toml")

// infrastructurePosLines is the number of positions that ran per tick with the module on: the
// 22 of the off set plus P8.3b, the one position of the module (FR-CAP-044).
const infrastructurePosLines = 23

// TestGolden_Infrastructure verifies the module-alone golden of the infrastructure module: pack one +
// overlay/infrastructure.toml, seed 1, 400 ticks, run as a Step loop with the recorder and
// WithAudit (so the books and the claim identities are audited every tick), against
// testdata/golden/infrastructure/one.txt.
//
// Requirement: FR-REF-025, FR-REF-026, FR-CAP-001, FR-CAP-007, FR-CAP-008, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_Infrastructure(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneInfrastructure...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveCold(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("infrastructure", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", infrastructurePositionsNote(t, cfg))
	}
	checkModuleGolden(t, "infrastructure", "one.txt", got)
}

// TestGolden_InfrastructureEvents verifies the golden rendering of the events of pack one with
// the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/infrastructure/events-one.txt, and that the Run and a bare Step loop end in
// the same snapshot. The file holds the facilities added, ruined and handed to the estate and the
// per-tick limits of the run.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-CAP-038, FR-CAP-039
//
// Guard: the events of the module are the run's record of every claim; a rule of the
// decay, the ruin, the wear or the removal that changed moves the file.
func TestGolden_InfrastructureEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneInfrastructure...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	checkModuleGolden(t, "infrastructure", "events-one.txt", renderEvents(res))
}

// TestPositionHash_InfrastructureAlone verifies the per-position state hash of pack one with
// the module on, seed 1, ticks 0 to 29, against testdata/golden/infrastructure/one.positions.txt:
// one line per position that ran, 23 per tick.
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-CAP-044
//
// Guard: when the golden fails, the first differing line names the tick and position at
// which the module's state diverged.
func TestPositionHash_InfrastructureAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneInfrastructure...), goldenSeed, posTicks)
	if want := posTicks * infrastructurePosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, infrastructurePosLines)
	}
	checkModuleGolden(t, "infrastructure", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// TestGolden_InfrastructureOffUnmoved verifies the proof of testing.md 7.1.1 for the infrastructure
// module: the three files of the off set of pack one are byte-identical to the ones the
// module found, the set holds no other file than before, and the hash of the all-off
// reference configuration is the pinned one.
//
// Requirement: FR-CAP-001, FR-REF-029
//
// Guard: an off file regenerated to absorb the module's change, or a hash that moved,
// fails here; a diff in golden/off is a failure of the change, not a golden to
// regenerate (architecture.md 2.10.5, N-217).
func TestGolden_InfrastructureOffUnmoved(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]string{"one.txt": offOneTxtSha256, "events-one.txt": eventsOneSha256, "one.positions.txt": positionsOneSha256} {
		b, err := os.ReadFile(goldenPath(name))
		if err != nil {
			t.Fatalf("read off/%s: %v", name, err)
		}
		if got := sha256Hex(b); got != want {
			t.Errorf("sha256(off/%s) = %s, want %s", name, got, want)
		}
	}
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want the pinned %s", got, pinnedReferenceHash)
	}
	entries, err := os.ReadDir(moduleGoldenPath("infrastructure", "."))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if want := []string{"events-one.txt", "one.positions.txt", "one.txt"}; !slices.Equal(names, want) {
		t.Errorf("files of golden/infrastructure = %v, want %v", names, want)
	}
}

// infrastructurePositionsNote replays the scripted run on cfg against the recorded positions
// file of the module, so a failing golden says where the state first moved.
func infrastructurePositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("infrastructure", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}

// TestDeterminism_Infrastructure verifies that a run of the infrastructure pack is the same run however
// it is observed: for each seed, a run with a logger, an observer or the audit, and a run
// split into Run calls (the reduced case list, seeds 1 and 2), end in the snapshot and the
// state hash of the bare Step run, the events of the observed runs are equal, and a
// different seed gives a different run.
//
// Requirement: FR-CAP-002, FR-CAP-039, NFR-CAP-001
//
// Guard: the facility step reads state (conditions, tables, counters) and draws no
// word; a logger, an observer or the audit that changed what the module counted would
// change the run, and only a run with facilities in use can show it.
func TestDeterminism_Infrastructure(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneInfrastructure...)
	cases := detCases(true)
	seeds := seedList(2)
	finals := make([]economysim.Snapshot, len(seeds))
	for i, seed := range seeds {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
			finals[i] = base.Final
			ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
			refEvents := renderEvents(withoutRunBrackets(ref))
			if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
				t.Fatalf("seed %d: observed Step run differs from the bare one at %s", seed, d)
			}
			for _, tc := range cases {
				spec := tc.spec()
				spec.Seed, spec.Ticks = seed, detTicks
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("seed %d, case %q: final snapshot differs from the bare run", seed, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("seed %d, case %q: state hash of the world differs from the bare run", seed, tc.name)
				}
				if spec.Observe {
					if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
						t.Errorf("seed %d, case %q: events differ from the reference at %s", seed, tc.name, d)
					}
				}
			}
		})
	}
	if len(seeds) > 1 && reflect.DeepEqual(finals[0], finals[1]) {
		t.Errorf("seeds %d and %d end in the same snapshot, want different runs", seeds[0], seeds[1])
	}
}
