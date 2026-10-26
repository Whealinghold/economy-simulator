package economysim_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// goldenTicks and goldenSeed are the length and seed REF 8.2 fixes for the golden.
const (
	goldenTicks int64  = 400
	goldenSeed  uint64 = 1
)

// solveCold solves the cold layer set (base, one: no warm start), as FR-REF-007
// solves "the layered configuration without" the warm start, and adapts the
// report to the renderer (the one place a rename of the solver report is edited).
func solveCold(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packOneCold...))
	if err != nil {
		t.Fatalf("SolveSteadyState(cold one) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// samplesOnly returns res with only the samples of REF 8.5 (a snapEvery run keeps
// every tick), so a per-tick run renders the same one.txt as a snapSample run.
func samplesOnly(res result, ticks int64) result {
	out := res
	out.Samples = map[int64]economysim.Snapshot{}
	for tick, s := range res.Samples {
		if (runSpec{Ticks: ticks, Snapshots: snapSample}).isSample(tick) {
			out.Samples[tick] = s
		}
	}
	return out
}

// oneTxt renders one.txt for a pack-one run.
func oneTxt(cfg economysim.Config, seed uint64, ticks int64, res result, report solveReport) []byte {
	return renderGolden("one", seed, ticks, samplesOnly(res, ticks), report, cfg.Warnings())
}

// TestGolden_OneOff verifies the end-to-end golden of pack one (the three layers),
// seed 1, 400 ticks, run as a Step loop with the recorder and WithAudit, against
// testdata/golden/off/one.txt (REF 8.5 items 1 to 6: item 6 is the digest of the
// checkpoint taken after the run, which is also the cross-architecture check of
// FR-CHK-056 when this test runs on arm64).
//
// Requirement: FR-REF-029, FR-REF-025, FR-REF-026, FR-OBS-023, FR-CHK-064, FR-CHK-056
//
// Guard: a behaviour change of any row moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_OneOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := oneTxt(cfg, goldenSeed, goldenTicks, res, solveCold(t))
	if want, err := os.ReadFile(goldenPath("one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", positionsNote(t, cfg))
	}
	checkGolden(t, "one.txt", got)
}

// TestGolden_OneEvents verifies the golden rendering of the events of pack one,
// seed 1, one Run of 400 ticks (RunStarted and RunEnded included), against
// testdata/golden/off/events-one.txt (OBS 9.5.3), and that this Run and a bare
// Step loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-REF-029, FR-REF-025
//
// Guard: the two runs of the golden cannot be one (Snapshot is not callable from
// an observer), so their final snapshots are compared here.
func TestGolden_OneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	var started, ended int
	for _, e := range res.Events {
		switch eventType(e) {
		case "RunStarted":
			started++
		case "RunEnded":
			ended++
		}
	}
	if started != 1 || ended != 1 {
		t.Errorf("RunStarted = %d, RunEnded = %d, want 1 and 1", started, ended)
	}
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	checkGolden(t, "events-one.txt", renderEvents(res))
}

// item1to5Sha256 is the sha256 of testdata/golden/off/one.txt as Stage 1 left it,
// before item 6 (the checkpoint digest line) was appended (plan 2-2 D257).
const item1to5Sha256 = "49dff2ef8f850eefdaa3a9546a5cfd6a82438a7d0edef164a365bc1e9b993d03"

// eventsOneSha256 and positionsOneSha256 are the sha256 of the events and the
// positions files of the off set, which a checkpoint does not move (a checkpoint
// emits no event and adds no position).
const (
	eventsOneSha256    = "610f126b3f67662c1ea42f2c1bdb8675b1e8a640d715518e66aec347736b8a26"
	positionsOneSha256 = "281d9b66812da906c21dd36ec410f6a7eb409d40d0b4c9f8ae2b4902caff9db9"
)

// sha256Hex returns the hex sha256 of b.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// TestGolden_OneOffMovedOnlyByItem6 verifies that one.txt differs from the file
// Stage 1 committed by its last line only: with the line `checkpoint_digest <hex>`
// and its newline removed, the file hashes to the old file's sha256.
//
// Requirement: FR-CHK-064, FR-REF-029
//
// Guard: the move of one.txt is explained once (testing.md 7.2, plan 2-2 D257);
// without this test a later -update could change items 1 to 5 under cover of the
// item 6 explanation. Plan 8 re-pins or retires it when it moves one.txt.
func TestGolden_OneOffMovedOnlyByItem6(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(goldenPath("one.txt"))
	if err != nil {
		t.Fatalf("read one.txt: %v", err)
	}
	// The remainder ends with the newline of the line before the digest line.
	body := bytes.TrimSuffix(b, []byte("\n"))
	i := bytes.LastIndexByte(body, '\n')
	if i < 0 || !bytes.HasPrefix(body[i+1:], []byte("checkpoint_digest ")) {
		t.Fatalf("last line of one.txt = %q, want checkpoint_digest <hex>", body[i+1:])
	}
	if got := sha256Hex(b[:i+1]); got != item1to5Sha256 {
		t.Errorf("sha256(one.txt without its last line) = %s, want %s (items 1 to 5 moved)", got, item1to5Sha256)
	}
	if hexPart := string(body[i+1+len("checkpoint_digest "):]); len(hexPart) != 2*sha256.Size {
		t.Errorf("digest %q has %d hex digits, want %d", hexPart, len(hexPart), 2*sha256.Size)
	}
}

// TestGolden_EventsAndPositionsUnchanged verifies that the events and positions
// files of the off set are byte-identical to the ones Stage 1 committed: a
// checkpoint emits no event and adds no position.
//
// Requirement: FR-REF-029, FR-OBS-080
//
// Guard: testing.md 7.1.1 forbids regenerating an off file to absorb a change; the
// hashes make a -update of either file fail here (N-217).
func TestGolden_EventsAndPositionsUnchanged(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, want string }{
		{"events-one.txt", eventsOneSha256},
		{"one.positions.txt", positionsOneSha256},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b, err := os.ReadFile(goldenPath(tt.name))
			if err != nil {
				t.Fatalf("read %s: %v", tt.name, err)
			}
			if got := sha256Hex(b); got != tt.want {
				t.Errorf("sha256(%s) = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

// demandPosLines is the number of positions that ran per tick in pack one-demand:
// the 22 of the off set and P3.6 and P6.2 of the module.
const demandPosLines = 24

// solveColdDemand solves the cold layer set of one-demand (base, demand,
// one-demand: no warm start), the programme with the wants.
func solveColdDemand(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packOneDemandCold...))
	if err != nil {
		t.Fatalf("SolveSteadyState(cold one-demand) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_OneDemand verifies the module-alone golden of the demand module:
// the pack one-demand (it carries its own census, so it is the module's pack and
// not the reference chain with an overlay: testing.md 7.1.1), seed 1, 400 ticks,
// run as a Step loop with the recorder and WithAudit, against
// testdata/golden/demand/one-demand.txt (REF 8.5 items 1 to 6, the digest line
// included).
//
// Requirement: FR-REF-025, FR-REF-026, FR-DEM-039, FR-DEM-040, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_OneDemand(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemand...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one-demand", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveColdDemand(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("demand", "one-demand.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one-demand.txt differs from the golden: %s (testing.md 7.1.2)", demandPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "demand", "one-demand.txt", got)
}

// TestGolden_OneDemandEvents verifies the golden rendering of the events of
// one-demand, seed 1, one Run of 400 ticks, against
// testdata/golden/demand/events-one-demand.txt, and that the Run and a bare Step
// loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-REF-025, FR-DEM-035
//
// Guard: the file holds WantShortfall events; without them the golden would not
// see the module's events.
func TestGolden_OneDemandEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemand...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	if !bytes.Contains(got, []byte("WantShortfall")) {
		t.Errorf("events of one-demand hold no WantShortfall, want the module's event in the golden")
	}
	checkModuleGolden(t, "demand", "events-one-demand.txt", got)
}

// TestPositionHash_OneDemandGolden verifies the per-position state hash of
// one-demand, seed 1, ticks 0 to 29, against testdata/golden/demand/
// one-demand.positions.txt: one line per position that ran, 24 per tick (P3.6
// and P6.2 run with the module on, and not in the off set).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-DEM-040
//
// Guard: when the golden fails, the first differing line names the tick and
// position at which the demand state diverged.
func TestPositionHash_OneDemandGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneDemand...), goldenSeed, posTicks)
	if want := posTicks * demandPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, demandPosLines)
	}
	checkModuleGolden(t, "demand", "one-demand.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// demandPositionsNote replays the scripted run on cfg against the recorded
// positions file of the module, so a failing golden says where the state first
// moved.
func demandPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("demand", "one-demand.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one-demand.positions.txt: the divergence is later", posTicks-1)
}

// offOneTxtSha256 is the sha256 of testdata/golden/off/one.txt as it stands,
// digest line included.
const offOneTxtSha256 = "6ce2e8fd1d260688695234380ac93b8d1fdd30934fada60747224d19b8795c96"

// TestGolden_OffUnchanged verifies the proof of testing.md 7.1.1 for the demand
// module: the three files of the off set are byte-identical to the ones the
// module found (one.txt by its full hash, the events and positions files by the
// hashes of TestGolden_EventsAndPositionsUnchanged), and the set holds no other
// file.
//
// Requirement: FR-DEM-001, FR-REF-029
//
// Guard: an off file regenerated to absorb the module's change would fail here;
// a diff in golden/off is a failure of the change, not a golden to regenerate
// (architecture.md 2.10.5).
func TestGolden_OffUnchanged(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(goldenPath("one.txt"))
	if err != nil {
		t.Fatalf("read one.txt: %v", err)
	}
	if got := sha256Hex(b); got != offOneTxtSha256 {
		t.Errorf("sha256(off/one.txt) = %s, want %s", got, offOneTxtSha256)
	}
	entries, err := os.ReadDir(filepath.Dir(goldenPath("one.txt")))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if want := []string{"events-one.txt", "one.positions.txt", "one.txt"}; !slices.Equal(names, want) {
		t.Errorf("files of golden/off = %v, want %v", names, want)
	}
}
