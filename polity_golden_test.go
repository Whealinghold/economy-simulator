package economysim_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fivePolityGoverned is the warm five with the governed polity overlay layered last:
// the module-alone pack of the polity module (spec-reference-chain.md 4.4.4, 4.5).
var fivePolityGoverned = append(slices.Clone(warmFive), "overlay/polity-governed.toml")

// polityPosLines is the number of positions that ran per tick: the module has no
// position, so it is the 22 of the off set (spec-polity.md 7.1).
const polityPosLines = 22

// solvePolity is the steady-state report the golden prints for the governed pack.
func solvePolity(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, fivePolityGoverned...))
	if err != nil {
		t.Fatalf("SolveSteadyState(five + polity-governed) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_PolityAlone verifies the module-alone golden of the polity module: the
// warm five with overlay/polity-governed.toml, seed 1, 400 ticks, run as a Step loop
// with the recorder and WithAudit, against testdata/golden/polity/five.txt. The file
// is a new set, not a diff of five: a seat shifts every household's agent ID
// (POL-A1, FR-POL-015).
//
// Requirement: FR-REF-025, FR-POL-015, FR-POL-016, FR-POL-028, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_PolityAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolityGoverned...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("five", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solvePolity(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("polity", "five.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("five.txt differs from the golden: %s (testing.md 7.1.2)", polityPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "polity", "five.txt", got)
}

// TestGolden_PolityAloneEvents verifies the golden rendering of the events of the
// governed five, seed 1, one Run of 400 ticks, against
// testdata/golden/polity/events-five.txt, and that the Run and a bare Step loop end in
// the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-POL-035
//
// Guard: the file holds the WorldCreated row with the treasury's Jurisdictions row;
// without it the golden would not see the module.
func TestGolden_PolityAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolityGoverned...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	if !bytes.Contains(got, []byte("Jurisdictions")) {
		t.Errorf("events of the governed five hold no Jurisdictions row, want the module's field in the golden")
	}
	checkModuleGolden(t, "polity", "events-five.txt", got)
}

// TestPositionHash_PolityGolden verifies the per-position state hash of the governed
// five, seed 1, ticks 0 to 29, against testdata/golden/polity/five.positions.txt: one
// line per position that ran, 22 per tick (the module adds none).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-POL-021
//
// Guard: when the golden fails, the first differing line names the tick and position
// at which the state diverged.
func TestPositionHash_PolityGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, fivePolityGoverned...), goldenSeed, posTicks)
	if want := posTicks * polityPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, polityPosLines)
	}
	checkModuleGolden(t, "polity", "five.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// polityPositionsNote replays the scripted run on cfg against the recorded positions
// file of the module, so a failing golden says where the state first moved.
func polityPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("polity", "five.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return "positions of ticks 0 to 29 agree with five.positions.txt: the divergence is later"
}

// TestGolden_PolityOffUnmoved verifies that the module leaves no trace while it is
// off: the governed pack with the module removed hashes as the warm five does, and the
// all-off reference configuration still has its pinned hash (N-217). The files of
// testdata/golden/off/ are the ones of TestGolden_Off, which this module's change does
// not touch.
//
// Requirement: FR-POL-001, FR-CFG-105
//
// Guard: an append that moved the off hash would break the rule of architecture.md
// 2.10.5; it is not a golden to regenerate.
func TestGolden_PolityOffUnmoved(t *testing.T) {
	t.Parallel()
	governed := loadValidPack(t, fivePolityGoverned...)
	governed.Modules.Polity = nil
	if got, want := hexHash(governed), hexHash(loadValidPack(t, warmFive...)); got != want {
		t.Errorf("hash of the governed five without the module = %s, want the warm five's %s", got, want)
	}
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want the pin %s", got, pinnedReferenceHash)
	}
}

// TestDeterminism_PolityAlone verifies the determinism row for the governed five, seed
// 1, 400 ticks: a run split into Run calls, with a logger, an observer or the audit,
// ends in the snapshot and the state hash of the bare Step run, and its events equal
// the reference's.
//
// Requirement: NFR-POL-004, FR-OBS-023, FR-WLD-063
//
// Guard: the treasury is state; a logger, an observer or the chunking of Run that moved
// it would fail here.
func TestDeterminism_PolityAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolityGoverned...)
	seed := uint64(1)
	base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
	ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
	refEvents := renderEvents(withoutRunBrackets(ref))
	if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
		t.Fatalf("seed %d: observed Step run differs from the bare one at %s (FR-OBS-023)", seed, d)
	}
	for _, tc := range detCases(true) {
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
}

// TestFork_Polity verifies the fork at N of CHK 8.1 on the governed five (FR-POL-032):
// the original and two restores give the same events, snapshot, audit and checkpoint
// bytes after N ticks of 0, 37 (a tick inside the jobs) and 150.
//
// Requirement: FR-POL-032, FR-CHK-048
//
// Guard: a restore that did not attach the treasury, or a treasury derived from a
// different order, would differ at the first snapshot.
func TestFork_Polity(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolityGoverned...)
	for _, n := range []int{0, 37, 150} {
		forkAtN(t, forkCase{Name: fmt.Sprintf("polity at %d", n), Cfg: cfg, Seed: 3, N: n, M: forkM, Logs: true,
			Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }})
	}
}
