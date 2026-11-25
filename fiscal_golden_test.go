package economysim_test

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiscalPosLines is the number of positions that ran per tick of the module-alone pack: the 22 of the
// off set and the four positions of the module, P1.7b, P3.4a, P3.6c and P4.1a (spec-fiscal.md 10.1,
// spec-world.md 6).
const fiscalPosLines = 26

// solveFiscal is the steady-state report the golden prints for the fiscal pack.
func solveFiscal(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, fiveFiscal...))
	if err != nil {
		t.Fatalf("SolveSteadyState(five + polity-governed + fiscal-governed) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_Fiscal verifies the module-alone golden of the fiscal module: the warm five with
// overlay/polity-governed.toml and the two fiscal overlays, seed 1, 400 ticks, run as a Step loop with the
// recorder and WithAudit, against testdata/golden/fiscal/five.txt. The file is a new set, not a diff of the
// polity golden: the overlay seats two more treasuries, which shifts every household's agent ID, and its
// flows move money, so every later sample differs. The probe of plan 3-fis Phase 10 chose this pack: seeds 1 to
// 4 record a tax, a cut payout and a duty trade of a pseudo-agent by tick 400 (the toll is proved by S136).
//
// Requirement: FR-REF-025, FR-FIS-001, FR-FIS-002, FR-FIS-003, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with -update only after the
// diff is understood (testing.md 7.2).
func TestGolden_Fiscal(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveFiscal...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("five", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveFiscal(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("fiscal", "five.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("five.txt differs from the golden: %s (testing.md 7.1.2)", fiscalPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "fiscal", "five.txt", got)
}

// TestGolden_FiscalEvents verifies the golden rendering of the events of the fiscal five, seed 1, one Run
// of 400 ticks, against testdata/golden/fiscal/events-five.txt, and that the Run and a bare Step loop end
// in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-FIS-044, FR-FIS-045
//
// Guard: the file holds the module's two events; a golden with none would not see the module.
func TestGolden_FiscalEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveFiscal...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	for _, want := range []string{"FiscalFlowApplied", "TreasuryOrdered"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("events of the fiscal five hold no %s row, want the module's event in the golden", want)
		}
	}
	checkModuleGolden(t, "fiscal", "events-five.txt", got)
}

// TestPositionHash_FiscalGolden verifies the per-position state hash of the fiscal five, seed 1, ticks 0 to
// 29, against testdata/golden/fiscal/five.positions.txt: one line per position that ran, 26 per tick (the
// 22 of the off set and the module's four).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-FIS-040
//
// Guard: when the golden fails, the first differing line names the tick and position at which the state
// diverged.
func TestPositionHash_FiscalGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, fiveFiscal...), goldenSeed, posTicks)
	if want := posTicks * fiscalPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, fiscalPosLines)
	}
	checkModuleGolden(t, "fiscal", "five.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// fiscalPositionsNote replays the scripted run on cfg against the recorded positions file of the module,
// so a failing golden says where the state first moved.
func fiscalPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("fiscal", "five.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return "positions of ticks 0 to 29 agree with five.positions.txt: the divergence is later"
}

// TestGolden_FiscalOffUnmoved verifies that the module leaves no trace while it is off and that the
// existing goldens' inputs did not move: the fiscal pack with the module removed hashes differently from
// the pack with it, the layered files of the polity golden (the warm five and overlay/polity-governed.toml)
// still load with no fiscal module, and the all-off reference configuration still has its pinned hash
// (N-217). The files of testdata/golden/off/, polity/ and all/ are the ones of their own tests, which this
// change does not touch.
//
// Requirement: FR-FIS-001, FR-CFG-105
//
// Guard: an append that moved the off hash would break the rule of architecture.md 2.10.5; it is not a
// golden to regenerate.
func TestGolden_FiscalOffUnmoved(t *testing.T) {
	t.Parallel()
	on := loadValidPack(t, fiveFiscal...)
	off := on
	off.Modules.Fiscal = nil
	if hexHash(on) == hexHash(off) {
		t.Errorf("hash of the fiscal five with and without the module = %s both, want different", hexHash(on))
	}
	if governed := loadValidPack(t, fivePolityGoverned...); governed.Modules.Fiscal != nil || governed.Modules.Polity == nil {
		t.Errorf("the polity golden's pack has Fiscal = %v and Polity = %v, want no fiscal module and the polity", governed.Modules.Fiscal, governed.Modules.Polity)
	}
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want the pin %s", got, pinnedReferenceHash)
	}
}

// TestDeterminism_Fiscal verifies the determinism row for the fiscal five, seed 1, 400 ticks: a run split
// into Run calls, with a logger, an observer or the audit, ends in the snapshot and the state hash of the
// bare Step run, and its events equal the reference's.
//
// Requirement: NFR-FIS-004, FR-OBS-023, FR-WLD-063, FR-FIS-062
//
// Guard: the carries and the scales are state; a logger, an observer or the chunking of Run that moved one
// would fail here.
func TestDeterminism_Fiscal(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveFiscal...)
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
