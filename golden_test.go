package economysim_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"reflect"
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
