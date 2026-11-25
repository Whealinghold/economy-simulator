package economysim_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packOneSaving is pack one with the infrastructure module and the saving module on:
// overlay/infrastructure.toml and then overlay/saving.toml (plan 3-sav Phase 7), the module-alone
// golden's configuration (spec-reference-chain.md 4.5). The saving module needs the infrastructure
// module for the build and repair legs, so the neighbour is layered under it.
var packOneSaving = append(slices.Clone(packOneInfrastructure), "overlay/saving.toml")

// savingProbe is what a run of the saving pack did that the golden must show.
type savingProbe struct {
	// Saving counts the SavingPlanned events with households saving, Builds and Repairs the
	// requests they report, Built and Repaired the FacilityChanged of kind built and repaired.
	Saving, Builds, Repairs, Built, Repaired int64
}

// savingProbeOf reads the events of a run of the saving pack.
func savingProbeOf(events []economysim.Event) savingProbe {
	var p savingProbe
	for _, e := range events {
		switch v := e.(type) {
		case economysim.SavingPlanned:
			if v.Saving > 0 {
				p.Saving++
			}
			p.Builds += v.Builds
			p.Repairs += v.Repairs
		case economysim.FacilityChanged:
			switch v.Kind {
			case economysim.FacilityBuilt:
				p.Built++
			case economysim.FacilityRepaired:
				p.Repaired++
			}
		}
	}
	return p
}

// savingPosLines is the number of positions that ran per tick with the two modules on: the 23 of
// the infrastructure set plus P3.6b, the one position of the saving module (FR-WLD-077).
const savingPosLines = 24

// TestGolden_Saving verifies the module-alone golden of the saving module: pack one with
// overlay/infrastructure.toml and overlay/saving.toml, seed 1, 400 ticks, run as a Step loop with the
// recorder and WithAudit (so the books and the identities are audited every tick), against
// testdata/golden/saving/one.txt.
//
// Requirement: FR-REF-025, FR-REF-026, FR-SAV-001, FR-SAV-033, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with -update only
// after the diff against golden/infrastructure is understood (testing.md 7.2).
func TestGolden_Saving(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSaving...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveCold(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("saving", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", savingPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "saving", "one.txt", got)
}

// TestGolden_SavingEvents verifies the golden rendering of the events of pack one with the
// module on, seed 1, one Run of 400 ticks, against testdata/golden/saving/events-one.txt, and that
// the Run and a bare Step loop end in the same snapshot. The file holds the SavingPlanned
// aggregates and the facilities built and repaired.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-SAV-029, FR-SAV-030
//
// Guard: a rule of the target, the choice or the request that changed moves the file, and the
// probe (TestGolden_SavingProbeExercisesAProject) says the file is not empty of the module.
func TestGolden_SavingEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSaving...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	if !bytes.Contains(got, []byte("SavingPlanned")) {
		t.Errorf("events of the saving pack hold no SavingPlanned, want the module's trace in the golden")
	}
	checkModuleGolden(t, "saving", "events-one.txt", got)
}

// TestPositionHash_SavingAlone verifies the per-position state hash of pack one with the module
// on, seed 1, ticks 0 to 29, against testdata/golden/saving/one.positions.txt: one line per
// position that ran, 24 per tick (P3.6b included).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-WLD-077
//
// Guard: when the golden fails, the first differing line names the tick and position at which
// the module's state diverged.
func TestPositionHash_SavingAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneSaving...), goldenSeed, posTicks)
	if want := posTicks * savingPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, savingPosLines)
	}
	checkModuleGolden(t, "saving", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// TestGolden_SavingOffLeavesNoTrace verifies the proof of testing.md 7.1.1 for the saving module:
// the three files of the off set of pack one are byte-identical to the ones the module found, the
// hash of the all-off reference configuration is the pinned one, the golden/saving set holds the
// three files and no other, and the set of golden/all and the pinned checkpoint fixtures name no
// saving file (the module joins them at Stage 3 close, not here).
//
// Requirement: FR-SAV-001, FR-REF-029
//
// Guard: an off file regenerated to absorb the module's change, or a hash that moved, fails here;
// a diff in golden/off is a failure of the change, not a golden to regenerate (architecture.md
// 2.10.5, N-217).
func TestGolden_SavingOffLeavesNoTrace(t *testing.T) {
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
	entries, err := os.ReadDir(moduleGoldenPath("saving", "."))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if want := []string{"events-one.txt", "one.positions.txt", "one.txt"}; !slices.Equal(names, want) {
		t.Errorf("files of golden/saving = %v, want %v", names, want)
	}
	for _, dir := range []string{moduleGoldenPath("all", "."), filepath.Join("testdata", "checkpoints")} {
		checkNoSavingFile(t, dir)
	}
}

// checkNoSavingFile fails for every file of dir whose name mentions the saving module.
func checkNoSavingFile(t testing.TB, dir string) {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range es {
		if strings.Contains(e.Name(), "saving") {
			t.Errorf("%s holds %s: the module joins golden/all and the checkpoint fixtures at Stage 3 close, not here", dir, e.Name())
		}
	}
}

// savingPositionsNote replays the scripted run on cfg against the recorded positions file of the
// module, so a failing golden says where the state first moved.
func savingPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("saving", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}

// TestGolden_SavingProbeExercisesAProject verifies, over seeds 1 to 4, that the pack of the golden
// makes households save for a project and carry it out: each run records a SavingPlanned with
// Saving above 0, a Builds request and a FacilityChanged of kind built by tick 400, and a repair
// request and a repaired facility as well. A pack whose households never saved or never built
// would record a golden that proves nothing about the module (plan 3-sav Phase 7, R3).
//
// Requirement: FR-SAV-019, FR-SAV-021, FR-SAV-022, FR-SAV-029
//
// Guard: an overlay whose hurdles are never passed, a collapse of the pack before any household
// has saved (N-352), or a request the world never starts fails here before the golden can be
// recorded empty. The choice of pack one over five is recorded in the plan's PR 5 result.
func TestGolden_SavingProbeExercisesAProject(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSaving...)
	for seed := uint64(1); seed <= 4; seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			res := runPack(t, cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true})
			p := savingProbeOf(res.Events)
			if p.Saving == 0 || p.Builds == 0 || p.Built == 0 || p.Repairs == 0 || p.Repaired == 0 {
				t.Errorf("seed %d: %+v, want every count above 0 (a saving household, a build request, a built facility, a repair request, a repair)", seed, p)
			}
		})
	}
}
