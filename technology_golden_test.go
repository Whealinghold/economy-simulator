package economysim_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// technologyPosLines is the number of positions that ran per tick with the module on:
// the 22 of the off set and P8.3c.
const technologyPosLines = 23

// solveTechnology solves the steady state of pack three with the technology overlay,
// the report the module-alone golden prints.
func solveTechnology(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, technologyThree...))
	if err != nil {
		t.Fatalf("SolveSteadyState(three + technology-four) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_TechnologyAlone verifies the module-alone golden of the technology module
// (testing.md 7.1.1): pack three, warm, with the trade edges and overlay/technology-four.toml
// (learning, trend, discovery and diffusion on one field), the default strategies, seed 1,
// 400 ticks, as a Step loop with the recorder and WithAudit, against
// testdata/golden/technology/three.txt.
//
// Requirement: FR-TEC-001, FR-TEC-017, FR-TEC-024, FR-TEC-026, FR-REF-025, FR-REF-026, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with -update
// only after the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN: the files are
// new and golden/off and golden/all are untouched.
func TestGolden_TechnologyAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, technologyThree...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("three", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveTechnology(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("technology", "three.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("three.txt differs from the golden: %s (testing.md 7.1.2)", technologyPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "technology", "three.txt", got)
}

// TestGolden_TechnologyAloneEvents verifies the events of the same run, driven by Run,
// against events-three.txt: they hold TechnologyAdvanced with a discovery and with a
// catch-up add, so the golden exercises every mechanism, and the final snapshot of Run(400)
// equals the Step loop's.
//
// Requirement: FR-TEC-024, FR-TEC-009, FR-TEC-015, FR-OBS-023
//
// Guard: without a discovery and a catch-up add in the file the golden would record only
// learning and the other two mechanisms would be unpinned.
func TestGolden_TechnologyAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, technologyThree...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	if m := mechanismsSeen(res.Events); slices.Contains(m[:], false) {
		t.Errorf("TechnologyAdvanced events show learning, trend, discovery, catch-up = %v, want each true", m)
	}
	checkModuleGolden(t, "technology", "events-three.txt", renderEvents(res))
}

// TestPositionHash_TechnologyGolden verifies the per-position state hash of ticks 0 to 29
// against three.positions.txt, with P8.3c in every tick (testing.md 7.1.2).
//
// Requirement: FR-TEC-033, FR-WLD-004
func TestPositionHash_TechnologyGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, technologyThree...), goldenSeed, posTicks)
	if want := posTicks * technologyPosLines; len(lines) != want {
		t.Errorf("recorded %d position lines, want %d (%d positions per tick)", len(lines), want, technologyPosLines)
	}
	if !slices.ContainsFunc(lines, func(l string) bool { return strings.Contains(l, " P8.3c ") }) {
		t.Errorf("the positions file has no line for P8.3c, want the module's position")
	}
	checkModuleGolden(t, "technology", "three.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// technologyPositionsNote replays the scripted run against the recorded positions file,
// so a failing golden says where the state first moved.
func technologyPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("technology", "three.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return "positions of ticks 0 to 29 agree with three.positions.txt: the divergence is later"
}

// TestGolden_OffUnchangedByTechnology verifies that the files of the off set are byte for
// byte what they were (the sums of offGoldenSums), that the pinned hash of the reference
// configuration did not move, and that the technology pack with the module removed hashes
// as the pack without it: a world with the module off leaves no trace (architecture.md
// 2.10.5).
//
// Requirement: FR-TEC-001, FR-REF-025
//
// Guard: a regenerated off golden would hide a non-interference failure of the module
// (testing.md 7.1.1).
func TestGolden_OffUnchangedByTechnology(t *testing.T) {
	t.Parallel()
	checkOffSums(t)
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want the pin %s", got, pinnedReferenceHash)
	}
	on := loadValidPack(t, technologyThree...)
	on.Modules.Technology = nil
	base := slices.Clone(technologyThree[:len(technologyThree)-1])
	if got, want := hexHash(on), hexHash(loadValidPack(t, base...)); got != want {
		t.Errorf("hash of the technology pack without the module = %s, want the base pack's %s", got, want)
	}
}

// checkOffSums fails if a file of the off set differs from its recorded sum.
func checkOffSums(t *testing.T) {
	t.Helper()
	for name, want := range offGoldenSums {
		b, err := os.ReadFile(goldenPath(name))
		if err != nil {
			t.Fatalf("read off golden %s: %v", name, err)
		}
		sum := sha256.Sum256(b)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("sha256(testdata/golden/off/%s) = %s, want %s: the off set moved (testing.md 7.1.1)", name, got, want)
		}
	}
}

// mechanismsSeen reports whether some TechnologyAdvanced shows learning, a trend, a
// discovery and a catch-up add, in that order.
func mechanismsSeen(events []economysim.Event) (seen [4]bool) {
	for _, e := range events {
		if a, ok := e.(economysim.TechnologyAdvanced); ok {
			seen[0] = seen[0] || a.DoingAddedBP > 0
			seen[1] = seen[1] || a.TrendAddedBP > 0
			seen[2] = seen[2] || a.Discoveries > 0
			seen[3] = seen[3] || a.CatchUpAddedBP > 0
		}
	}
	return seen
}
