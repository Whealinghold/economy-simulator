package economysim_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds the five-sized rows of plan 2-24 D689: the golden files of the warm
// `five` with every module off (testdata/golden/off/five*.txt) and of the all-on pack
// (testdata/golden/all/), the determinism cases, and the fork-at-200. Both worlds hold
// 5 000 households, so every test here skips under -race and runs in gate F; the names
// start with a prefix of gateFPrefixes.

// fiveSizedPosTicks is the length of the positions golden of the two five-sized worlds: the
// hash after each position of a 5 000-household state took about 90 s for 30 ticks, and the
// first ticks name the position that diverged as well as 30 do (plan 2-24 D689).
const fiveSizedPosTicks = 10

// fiveWorld is a world the five-sized rows run: its layers, the layers the solver reads
// (the cold set) and the options the world needs besides the seed.
type fiveWorld struct {
	name   string
	layers []string
	cold   []string
	opts   []economysim.Option
	// golden is the path of a file of its golden set.
	golden func(file string) string
}

// allGoldenPath is the path of a file of the all-on set (testing.md 7.1.1).
func allGoldenPath(name string) string { return filepath.Join("testdata", "golden", "all", name) }

var (
	fiveOff = fiveWorld{"five", packFiveWarm, packLayers["five"], nil, goldenPath}
	allOn   = fiveWorld{"all", allLayers, allColdLayers(), allOptions(), allGoldenPath}
)

// config loads and validates the layers of the world.
func (f fiveWorld) config(t testing.TB) economysim.Config { return loadValidPack(t, f.layers...) }

// solve is the solver report of the cold layers, as the header of the golden shows it.
func (f fiveWorld) solve(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, f.cold...))
	if err != nil {
		t.Fatalf("SolveSteadyState(cold %s) error = %v, want nil", f.name, err)
	}
	return newSolveReport(ss)
}

// check compares got with the file of the golden set, or writes it with -update.
func (f fiveWorld) check(t testing.TB, file string, got []byte) {
	t.Helper()
	if err := compareGolden(f.golden(file), got, *updateFlag); err != nil {
		t.Error(err)
	}
}

// goldenRun is the run of the main golden of the world: seed 1, 400 ticks, a Step loop
// with the recorder, the audit and the digest line.
func (f fiveWorld) goldenRun(t testing.TB, cfg economysim.Config) result {
	return runPack(t, cfg, goldenSpec(f.opts))
}

// positions is the per-position hash of ticks 0 to fiveSizedPosTicks-1.
func (f fiveWorld) positions(t testing.TB, cfg economysim.Config) []string {
	return positionLines(t, cfg, goldenSeed, fiveSizedPosTicks, f.opts...)
}

// positionsNote names where a failing golden first diverged.
func (f fiveWorld) positionsNote(t testing.TB, cfg economysim.Config) string {
	b, err := os.ReadFile(f.golden(f.name + ".positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, f.positions(t, cfg)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with %s.positions.txt: the divergence is later", fiveSizedPosTicks-1, f.name)
}

// goldenMain is the body of TestGolden_FiveOff and TestGolden_AllOn.
func (f fiveWorld) goldenMain(t *testing.T) {
	t.Helper()
	cfg := f.config(t)
	res := f.goldenRun(t, cfg)
	got := renderGolden(f.name, goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), f.solve(t), cfg.Warnings())
	if want, err := os.ReadFile(f.golden(f.name + ".txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("%s.txt differs from the golden: %s (testing.md 7.1.2)", f.name, f.positionsNote(t, cfg))
	}
	f.check(t, f.name+".txt", got)
}

// goldenEvents is the body of TestGolden_FiveEvents and TestGolden_AllEvents: the events
// of one Run of 400 ticks, and that the Run and a bare Step loop end in one snapshot.
func (f fiveWorld) goldenEvents(t *testing.T) {
	t.Helper()
	cfg := f.config(t)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true, Extra: f.opts})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Extra: f.opts})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	f.check(t, "events-"+f.name+".txt", renderEventsFull(res, -1))
}

// goldenPositions is the body of TestGolden_FivePositions and TestGolden_AllPositions.
func (f fiveWorld) goldenPositions(t *testing.T, perTick int) {
	t.Helper()
	lines := f.positions(t, f.config(t))
	if want := fiveSizedPosTicks * perTick; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, perTick)
	}
	f.check(t, f.name+".positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// skipFiveSized skips the test of a 5 000-household world with -short and under -race.
func skipFiveSized(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	if raceEnabled {
		t.Skip("5 000 households: gate F runs this test, never -race (plan 2-24 Phase 2)")
	}
}

// TestGolden_FiveOff verifies the end-to-end golden of the warm `five` with every module
// off, seed 1, 400 ticks, as a Step loop with the recorder and WithAudit (REF 8.5 items 1
// to 6, the digest line included), against testdata/golden/off/five.txt.
//
// Requirement: FR-REF-029, FR-REF-025, FR-CHK-056
//
// Guard: DELIBERATE NEW GOLDEN, a new file of the off set; golden/off/one*.txt do not
// move. A change of the rebalanced five moves it once, with the explanation of
// testing.md 7.2 (plan 2-24 R1).
func TestGolden_FiveOff(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	fiveOff.goldenMain(t)
}

// TestGolden_FiveEvents verifies the golden rendering of the events of the warm `five`,
// seed 1, one Run of 400 ticks, against testdata/golden/off/events-five.txt, and that the
// Run and a bare Step loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-REF-029
func TestGolden_FiveEvents(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	fiveOff.goldenEvents(t)
}

// TestGolden_FivePositions verifies the per-position state hash of the warm `five`, seed 1,
// ticks 0 to 9, against testdata/golden/off/five.positions.txt: 22 positions per tick, as
// the off set of pack one.
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-REF-029
//
// Guard: when the golden fails, the first differing line names the tick and position.
func TestGolden_FivePositions(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	fiveOff.goldenPositions(t, 22)
}

// allPosLines is the number of positions that run per tick with every module on: the 22 of
// the off set and the 20 of the modules (the hash of a 5 000-household state is slow, so the
// positions golden of these two worlds covers fiveSizedPosTicks ticks, not posTicks).
const allPosLines = 42

// TestGolden_AllOn verifies the all-on golden: the all-on pack of allLayers, seed 1, 400
// ticks, as a Step loop with the recorder and WithAudit, against testdata/golden/all/all.txt.
//
// Requirement: FR-REF-029, FR-REF-025, FR-OBS-023, FR-CHK-056, testing.md 7.1.1
//
// Guard: DELIBERATE NEW GOLDEN, the first file of the all set. A module added in Stage 3
// appends a layer and moves it once, with the explanation of testing.md 7.2.
func TestGolden_AllOn(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	allOn.goldenMain(t)
}

// TestGolden_AllEvents verifies the golden rendering of the events of the all-on pack,
// seed 1, one Run of 400 ticks, against testdata/golden/all/events-all.txt, and that the Run
// and a bare Step loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-REF-029
func TestGolden_AllEvents(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	allOn.goldenEvents(t)
}

// TestGolden_AllPositions verifies the per-position state hash of the all-on pack, seed 1,
// ticks 0 to 9, against testdata/golden/all/all.positions.txt.
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-REF-029
//
// Guard: when the golden fails, the first differing line names the tick and position at
// which the state of a module diverged.
func TestGolden_AllPositions(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	allOn.goldenPositions(t, allPosLines)
}

// fiveDetCases are the cases of detCases(true) that TestDeterminism_FiveAndAll runs, by
// name: the observed Step run (the bare and observed of REF 8.2), the single Run, the
// split Run and the audit. The other cases of the list would be too slow on 5 000
// agents (plan 2-24 D689).
var fiveDetCases = []string{"(a) second observed run", "(b) Run(400) bare", "(b) Run(1)+Run(399) observed", "(d) audit on"}

// TestDeterminism_FiveAndAll verifies, on the warm `five` and on the all-on pack, seeds 1
// and 2, 400 ticks, the cases of fiveDetCases: a run split into Run calls, observed, or
// with the audit on, ends in the snapshot, state hash and config hash of the bare Step
// run, its events equal the reference's, and a second seed ends elsewhere.
//
// Requirement: FR-REF-030, FR-OBS-023, FR-WLD-063, NFR-WLD-004
//
// Guard: observation, the audit and the chunking of Run must not move a world that holds
// every module; the cases are taken from detCases by name, so a rename fails here.
func TestDeterminism_FiveAndAll(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	byName := map[string]detCase{}
	for _, c := range detCases(true) {
		byName[c.name] = c
	}
	for _, w := range []fiveWorld{fiveOff, allOn} {
		t.Run(w.name, func(t *testing.T) {
			t.Parallel()
			cfg := w.config(t)
			var ends [2]economysim.Snapshot
			// The "seeds" group returns when its parallel subtests have ended, so ends is complete after it.
			t.Run("seeds", func(t *testing.T) {
				for i, seed := range []uint64{1, 2} {
					t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
						t.Parallel()
						// The bare run takes only the snapshots of REF 8.5: a snapshot of 5 000 agents
						// at every tick is the cost of this test.
						base := runPack(t, cfg, runSpec{Seed: seed, Ticks: detTicks, Drive: driveStep, Snapshots: snapSample, Extra: w.opts})
						ends[i] = base.Final
						for _, name := range fiveDetCases {
							tc, ok := byName[name]
							if !ok {
								t.Fatalf("detCases(true) has no case %q, want it (renamed?)", name)
							}
							spec := tc.spec()
							spec.Seed, spec.Ticks, spec.Extra = seed, detTicks, append(spec.Extra, w.opts...)
							if len(spec.Drive.chunks) == 0 {
								spec.Snapshots = snapSample
							}
							got := runPack(t, cfg, spec)
							if !reflect.DeepEqual(got.Final, base.Final) {
								t.Errorf("%s seed %d, case %q: final snapshot differs from the bare run", w.name, seed, name)
							}
							if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
								t.Errorf("%s seed %d, case %q: state hash of the world differs from the bare run", w.name, seed, name)
							}
							if got.World.ConfigHash() != base.World.ConfigHash() {
								t.Errorf("%s seed %d, case %q: ConfigHash differs from the bare run", w.name, seed, name)
							}
							if len(spec.Drive.chunks) == 0 {
								if d := firstTickDiff(base.Samples, got.Samples); d != "" {
									t.Errorf("%s seed %d, case %q: snapshot differs from the bare run at %s", w.name, seed, name, d)
								}
							}
						}
					})
				}
			})
			if reflect.DeepEqual(ends[0], ends[1]) {
				t.Errorf("%s: seeds 1 and 2 end in the same snapshot, want different", w.name)
			}
		})
	}
}
