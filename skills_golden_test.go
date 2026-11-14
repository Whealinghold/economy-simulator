package economysim_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packOneSkills is pack one, warm, with the skill overlay alone on: 120 ticks of
// training, three trainees a trainer and a teaching load of 1 000 on the two skilled
// roles. The pack carries no module (the skill barriers are core data,
// spec-config.md 3.7.2), so the module-alone golden is the reference chain with
// the overlay (testing.md 7.1.1). The warm layer is chosen by the probe of
// TestPacks_SkillCourseCompletes: the cold layer sets starve before a course ends.
var packOneSkills = append(slices.Clone(packOne), "overlay/skill.toml")

// skillsPosLines is the number of positions that ran per tick with a role that
// trains: the 22 of the off set and P2.2.
const skillsPosLines = 23

// TestPacks_SkillCourseCompletes verifies the choice of the pack of the module-alone
// golden (plan 3-skl R1, Question 5): pack one, warm, with overlay/skill.toml
// completes at least one course within 400 ticks, the length of the golden, at each
// of seeds 1 to 4, so the golden exercises the completion and not only the
// queue. The cold layer sets of one, two, three and five starve with the overlay and
// complete none in 600 ticks, which is why they are not the pack.
//
// Arithmetic: a place filled at tick s completes at s + 120 (TrainingTicks), and the
// first applicants are placed in the first ticks, so the first completion is
// expected about tick 125; the probe recorded 124 to 125 on seeds 1 to 4.
//
// Requirement: FR-SKL-010, FR-SKL-015, FR-REF-025
//
// Guard: without a completion the golden would record the placement and the ends
// of the traineeships but never the role change a course causes.
func TestPacks_SkillCourseCompletes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	for seed := uint64(1); seed <= 4; seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			var first int64 = -1
			obs := economysim.ObserverFunc(func(e economysim.Event) {
				if c, ok := e.(economysim.TraineeshipCompleted); ok && first < 0 {
					first = c.Tick
				}
			})
			w := newPackWorld(t, cfg, seed, economysim.WithObserver(obs))
			for range goldenTicks {
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d: Step() error = %v, want nil", seed, err)
				}
			}
			if first < 0 || first > 200 {
				t.Errorf("seed %d: first TraineeshipCompleted at tick %d, want one within the first 200 ticks", seed, first)
			}
		})
	}
}

// TestGolden_SkillsAlone verifies the module-alone golden of the skill barriers:
// pack one, warm, with overlay/skill.toml (testing.md 7.1.1), the default
// strategies, seed 1, 400 ticks, as a Step loop with the recorder and WithAudit,
// against testdata/golden/skills/one.txt; the events of a Run against
// events-one.txt, which holds the three Traineeship events and the final snapshot
// with the places, the queue and the trainees; and the per-position state hash of
// ticks 0 to 29 against one.positions.txt, with P2.2 in it. A run without the observer
// ends in the same snapshot.
//
// Requirement: FR-SKL-021, FR-SKL-027, FR-REF-025, FR-REF-026, FR-OBS-023, FR-WLD-004
//
// Guard: a behaviour change of the module moves the files; they are regenerated
// with -update only after the diff is understood (testing.md 7.2). DELIBERATE NEW
// GOLDEN: the files are new and golden/off is untouched.
func TestGolden_SkillsAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	res := runPack(t, cfg, goldenSpec(nil))
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveCold(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("skills", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", skillsPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "skills", "one.txt", got)

	ev := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(ev.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) observed differs from the bare Step loop's (seed %d)", goldenSeed)
	}
	events := renderEvents(ev)
	for _, want := range []string{"TraineeshipStarted", "TraineeshipCompleted", "TraineeshipEnded"} {
		if !bytes.Contains(events, []byte(want)) {
			t.Errorf("the events hold no %s, want the module's three events in the golden", want)
		}
	}
	checkModuleGolden(t, "skills", "events-one.txt", events)

	lines := positionLines(t, cfg, goldenSeed, posTicks)
	if want := posTicks * skillsPosLines; len(lines) != want {
		t.Errorf("recorded %d position lines, want %d (%d positions per tick)", len(lines), want, skillsPosLines)
	}
	if !slices.ContainsFunc(lines, func(l string) bool { return strings.Contains(l, " P2.2 ") }) {
		t.Errorf("the positions file has no line for P2.2, want the module's position")
	}
	checkModuleGolden(t, "skills", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// skillsPositionsNote replays the scripted run against the recorded positions file,
// so a failing golden says where the state first moved.
func skillsPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("skills", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}

// TestGolden_OffUnchangedBySkills verifies that the three files of the off set are
// byte for byte what they were before the skill barriers' traineeship (the sums of
// offGoldenSums), and that the pinned hash of the reference configuration did not
// move: a world with no role that trains leaves no trace (architecture.md 2.10.5).
//
// Requirement: FR-SKL-006, FR-REF-025
//
// Guard: a regenerated off golden would hide a non-interference failure of the
// module (testing.md 7.1.1).
func TestGolden_OffUnchangedBySkills(t *testing.T) {
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

// TestNonInterference_Skills verifies that a logger at Debug and an observer change
// nothing a traineeship does: pack one with the overlay, seed 1, 200 ticks (which hold
// placements, completions and removals of trainers), ends in the same snapshot and
// the same checkpoint digest bare and observed, and the observed run logs the
// Debug lines of the module.
//
// Requirement: FR-OBS-023, FR-SKL-028
func TestNonInterference_Skills(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	bare := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: 200, Drive: driveStep, Digest: true})
	h := newCaptureHandler()
	seen := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: 200, Drive: driveStep, Observe: true, Logger: h, Digest: true})
	if !reflect.DeepEqual(bare.Final, seen.Final) || bare.Digest != seen.Digest {
		t.Errorf("the observed run (logger at Debug and observer) ends in a different snapshot or checkpoint than the bare run (seed %d)", goldenSeed)
	}
	var ended, refused int
	for _, l := range h.sink.lines {
		switch l.msg {
		case "traineeship ended":
			ended++
		case "traineeship refused":
			refused++
		}
		if l.level > slog.LevelWarn {
			t.Errorf("line %q at level %v, want nothing above Warn from the module", l.msg, l.level)
		}
	}
	if ended == 0 {
		t.Errorf("no \"traineeship ended\" line in 200 ticks, want the trainers' removals logged (refused lines: %d)", refused)
	}
}
