package economysim_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// informationPosLines is the number of positions that ran per tick with the module
// alone on: the 22 of the off set plus P5.2 and P3.6 (the emitter of RumoursServed
// makes the position run when only the information module is on).
const informationPosLines = 24

// TestPacks_InformationOverlaysLoad verifies that each information overlay loads over
// its variant and its trade overlay, validates, switches the module on at a speed of
// 50 with every role of the pack that has a group informed, and raises none of the
// module's four warnings (the cold packs have others of their own)
// (spec-reference-chain.md 4.5, 7.6).
//
// Requirement: FR-INF-001, FR-INF-045, FR-REF-025
//
// Guard: the golden and the scenarios read these three files; a pack that stopped
// being valid or lost a role would change them without a message.
func TestPacks_InformationOverlaysLoad(t *testing.T) {
	t.Parallel()
	for _, variant := range []string{"two", "three", "five"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, packInformation(variant)...)
			if in := cfg.Modules.Information; in == nil || in.InfoSpeed != 50 || in.RumourErrorBP != 500 || in.RumourMaxAge != 60 {
				t.Fatalf("pack %s: Modules.Information = %+v, want speed 50, error 500, horizon 60", variant, in)
			}
			grouped := map[string]bool{}
			for _, g := range cfg.Groups {
				grouped[g.Role] = true
			}
			for _, r := range cfg.Roles {
				if r.Informed != grouped[r.ID] {
					t.Errorf("pack %s: role %q Informed = %t, want %t (the roles with a group)", variant, r.ID, r.Informed, grouped[r.ID])
				}
			}
			for _, w := range cfg.Warnings() {
				switch w.Rule {
				case "information_without_readers", "information_without_routes", "informed_without_sources", "rumour_horizon_short":
					t.Errorf("pack %s: warning %+v, want none of the module's four", variant, w)
				}
			}
		})
	}
}

// solveInformation solves the steady state of the module-alone pack, the programme the
// golden reports.
func solveInformation(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packInformation("two")...))
	if err != nil {
		t.Fatalf("SolveSteadyState(pack two with information) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_InformationAlone verifies the module-alone golden of the information
// module: pack two with the trade and the information overlays (testing.md 7.1.1), the
// default strategies, seed 1, 400 ticks, as a Step loop with the recorder and WithAudit,
// against testdata/golden/information/two.txt; the events of a Run against
// events-two.txt, which holds RumoursServed; and the per-position state hash of ticks 0
// to 29 against two.positions.txt. The default strategy ignores rumours, so the golden
// sees the world the module leaves untouched, the record it keeps and the aggregate it
// reports. A run without the observer ends in the same snapshot.
//
// Requirement: FR-INF-046, FR-REF-025, FR-REF-026, FR-OBS-023, FR-OBS-080, FR-WLD-004
//
// Guard: a behaviour change of the module moves the files; they are regenerated with
// -update only after the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN:
// the files are new, no existing golden moves.
func TestGolden_InformationAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packInformation("two")...)
	res := runPack(t, cfg, goldenSpec(nil))
	got := renderGolden("two", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveInformation(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("information", "two.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("two.txt differs from the golden: %s (testing.md 7.1.2)", informationPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "information", "two.txt", got)

	ev := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(ev.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) observed differs from the bare Step loop's (seed %d)", goldenSeed)
	}
	events := renderEvents(ev)
	if !bytes.Contains(events, []byte("RumoursServed")) {
		t.Errorf("the events hold no RumoursServed, want the module's event in the golden")
	}
	checkModuleGolden(t, "information", "events-two.txt", events)

	lines := positionLines(t, cfg, goldenSeed, posTicks)
	if want := posTicks * informationPosLines; len(lines) != want {
		t.Errorf("recorded %d position lines, want %d (%d positions per tick)", len(lines), want, informationPosLines)
	}
	checkModuleGolden(t, "information", "two.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// informationPositionsNote replays the scripted run against the recorded positions
// file, so a failing golden says where the state first moved.
func informationPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("information", "two.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with two.positions.txt: the divergence is later", posTicks-1)
}

// offGoldenSums are the SHA-256 of the three files of testdata/golden/off/ as they
// stood when the information module began (stage-2-13 Phase 3; the files have not
// moved since commit 60944dc of stage 2-2).
var offGoldenSums = map[string]string{
	"events-one.txt":    "610f126b3f67662c1ea42f2c1bdb8675b1e8a640d715518e66aec347736b8a26",
	"one.positions.txt": "281d9b66812da906c21dd36ec410f6a7eb409d40d0b4c9f8ae2b4902caff9db9",
	"one.txt":           "6ce2e8fd1d260688695234380ac93b8d1fdd30934fada60747224d19b8795c96",
}

// TestGolden_OffUnchangedByInformation verifies that the three files of the off set
// are byte for byte what they were before the information module, and that the pinned
// hash of the reference configuration did not move (TestHashConfig_ReferencePinned
// runs beside it): a module that is off leaves no trace (architecture.md 2.10.5).
//
// Requirement: FR-INF-047, FR-REF-025
//
// Guard: a regenerated off golden would hide a non-interference failure of the module
// (testing.md 7.1.1); the sums are the proof that the off set was not touched to make
// a test pass.
func TestGolden_OffUnchangedByInformation(t *testing.T) {
	t.Parallel()
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
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want the pin %s", got, pinnedReferenceHash)
	}
}
