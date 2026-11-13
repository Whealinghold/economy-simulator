package skill_test

import (
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/skill"
)

// TestGain_SignBoundary verifies the trainer's gain output - upkeep - load *
// wTrainer / 10 000: the sign boundary, a zero load, and overflow reporting.
//
// Requirement: FR-SKL-016
func TestGain_SignBoundary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                         string
		output, upkeep, load, wTrain int64
		want                         int64
		wantOK                       bool
	}{
		// 1 000 * 2 000 / 10 000 = 200 of teaching cost; 500 - 300 - 200 = 0.
		{"exactly zero is not negative", 500, 300, 1000, 2000, 0, true},
		// 499 - 300 - 200 = -1.
		{"one below zero", 499, 300, 1000, 2000, -1, true},
		{"one above zero", 501, 300, 1000, 2000, 1, true},
		{"zero load costs no teaching time", 500, 300, 0, 2000, 200, true},
		// 1 * 9 999 / 10 000 truncates to 0 before the subtraction.
		{"the teaching cost truncates toward zero", 10, 0, 1, 9999, 10, true},
		{"product overflow", 0, 0, math.MaxInt64, math.MaxInt64, 0, false},
		{"subtraction overflow", math.MinInt64, 1, 0, 0, 0, false},
		{"second subtraction overflow", math.MinInt64, 0, 10000, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := skill.Gain(tt.output, tt.upkeep, tt.load, tt.wTrain)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Gain(%d, %d, %d, %d) = %d, %v, want %d, %v", tt.output, tt.upkeep, tt.load, tt.wTrain, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestCTrain_FlooredAtZeroAndChecked verifies C_train = ticks * surplus with the
// surplus floored at 0 and overflow reported (spec-skill-barriers.md 4.8: 2 555
// ticks at a surplus of 500 is 1 277 500).
//
// Requirement: FR-SKL-018
func TestCTrain_FlooredAtZeroAndChecked(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		ticks, surplus int64
		want           int64
		wantOK         bool
	}{
		{"worked example of 4.8", 2555, 500, 1_277_500, true},
		{"negative surplus floors at 0", 2555, -40, 0, true},
		{"zero surplus", 2555, 0, 0, true},
		{"zero ticks", 0, 500, 0, true},
		{"overflow", math.MaxInt64, 2, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := skill.CTrain(tt.ticks, tt.surplus)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("CTrain(%d, %d) = %d, %v, want %d, %v", tt.ticks, tt.surplus, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestTeach_Accumulator verifies the teaching accumulator: three trainees at a
// load of 1 000 give three teaching ticks in ten, the accumulator stays in
// [0, 9 999] whatever the loads, and a sum above 10 000 teaches every tick.
//
// Requirement: FR-SKL-015
func TestTeach_Accumulator(t *testing.T) {
	t.Parallel()
	t.Run("three loads of 1 000 over ten ticks", func(t *testing.T) {
		t.Parallel()
		// 3 000 a tick: the accumulator runs 3 000, 6 000, 9 000, 12 000 -> 2 000
		// (teaching), 5 000, 8 000, 11 000 -> 1 000 (teaching), 4 000, 7 000,
		// 10 000 -> 0 (teaching): three teaching ticks in ten.
		acc, n := int64(0), 0
		for range 10 {
			var teaching bool
			acc, teaching = skill.Teach(acc, 3000)
			if teaching {
				n++
			}
		}
		if n != 3 || acc != 0 {
			t.Errorf("ten Teach steps of 3 000 = %d teaching ticks, acc %d, want 3, 0", n, acc)
		}
	})
	tests := []struct {
		name         string
		acc, sum     int64
		want         int64
		wantTeaching bool
	}{
		{"below a full tick", 0, 9999, 9999, false},
		{"reaches 10 000 exactly", 0, 10000, 0, true},
		{"carries the remainder", 9000, 2500, 1500, true},
		{"sum of 20 000 is cut to 9 999", 5, 20000, 9999, true},
		{"sum above 20 000 does not overflow", 9999, math.MaxInt64, 9999, true},
		{"sum between 10 000 and 20 000 keeps its carry", 9999, 19999, 9999, true},
		{"negative inputs count as 0", -5, -7, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, teaching := skill.Teach(tt.acc, tt.sum)
			if got != tt.want || teaching != tt.wantTeaching {
				t.Errorf("Teach(%d, %d) = %d, %v, want %d, %v", tt.acc, tt.sum, got, teaching, tt.want, tt.wantTeaching)
			}
		})
	}
}

// FuzzTeach verifies, for any int64 pair, that Teach never panics, keeps the
// accumulator in [0, 9 999], and teaches exactly when the total reaches 10 000.
//
// Requirement: FR-SKL-015
func FuzzTeach(f *testing.F) {
	f.Add(int64(0), int64(3000))
	f.Add(int64(9999), int64(1))
	f.Add(int64(0), int64(math.MaxInt64))
	f.Add(int64(math.MinInt64), int64(math.MinInt64))
	f.Fuzz(func(t *testing.T, acc, sum int64) {
		got, teaching := skill.Teach(acc, sum)
		if got < 0 || got > skill.MaxAccumulator {
			t.Fatalf("Teach(%d, %d) acc = %d, outside [0, 9999]", acc, sum, got)
		}
		a, s := max(acc, 0), max(sum, 0)
		want := s >= skill.Basis || a+s >= skill.Basis
		if teaching != want {
			t.Fatalf("Teach(%d, %d) teaching = %v, want %v", acc, sum, teaching, want)
		}
	})
}

// TestOpen_SumsFreePlaces verifies that the open places are the sum of Max -
// Held over the trainers, a full or over-full trainer counting 0.
//
// Requirement: FR-SKL-017
func TestOpen_SumsFreePlaces(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []skill.Trainer
		want int64
	}{
		{"no trainer", nil, 0},
		{"two free", []skill.Trainer{{Held: 1, Max: 3}}, 2},
		{"full trainer", []skill.Trainer{{Held: 3, Max: 3}}, 0},
		// An institution lowered the maximum under a traineeship in progress.
		{"over-full trainer counts 0", []skill.Trainer{{Held: 3, Max: 2}, {Held: 0, Max: 3}}, 3},
		{"saturates", []skill.Trainer{{Held: 0, Max: math.MaxInt64}, {Held: 0, Max: 5}}, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := skill.Open(tt.in); got != tt.want {
				t.Errorf("Open(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

// TestLess_ApplicationTickThenID verifies the queue order of new applicants and
// that it is a total order over distinct agents.
//
// Requirement: FR-SKL-017
func TestLess_ApplicationTickThenID(t *testing.T) {
	t.Parallel()
	a := skill.Applicant{Agent: 5, Applied: 10}
	tests := []struct {
		name string
		b    skill.Applicant
		want bool
	}{
		{"later tick is behind", skill.Applicant{Agent: 1, Applied: 11}, true},
		{"earlier tick is ahead", skill.Applicant{Agent: 9, Applied: 9}, false},
		{"same tick, higher ID is behind", skill.Applicant{Agent: 6, Applied: 10}, true},
		{"same tick, lower ID is ahead", skill.Applicant{Agent: 4, Applied: 10}, false},
		{"served ticks do not matter", skill.Applicant{Agent: 6, Applied: 10, Served: 99}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := skill.Less(a, tt.b); got != tt.want {
				t.Errorf("Less(%+v, %+v) = %v, want %v", a, tt.b, got, tt.want)
			}
			if skill.Less(tt.b, a) == tt.want {
				t.Errorf("Less(%+v, %+v) and its reverse agree", a, tt.b)
			}
		})
	}
	if skill.Requeue(a, a) {
		t.Errorf("Requeue(a, a) = true, want false")
	}
	if skill.Less(a, a) {
		t.Errorf("Less(a, a) = true, want false")
	}
}

// TestRequeue_ServedThenTickThenID verifies the order of trainees returned by a
// trainer's removal: more ticks served first, then application tick, then ID.
//
// Requirement: FR-SKL-020
func TestRequeue_ServedThenTickThenID(t *testing.T) {
	t.Parallel()
	a := skill.Applicant{Agent: 5, Applied: 10, Served: 40}
	tests := []struct {
		name string
		b    skill.Applicant
		want bool // a ahead of b
	}{
		{"fewer served is behind", skill.Applicant{Agent: 1, Applied: 1, Served: 39}, true},
		{"more served is ahead", skill.Applicant{Agent: 9, Applied: 99, Served: 41}, false},
		{"equal served, later tick is behind", skill.Applicant{Agent: 1, Applied: 11, Served: 40}, true},
		{"equal served and tick, higher ID is behind", skill.Applicant{Agent: 6, Applied: 10, Served: 40}, true},
		{"equal served and tick, lower ID is ahead", skill.Applicant{Agent: 4, Applied: 10, Served: 40}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := skill.Requeue(a, tt.b); got != tt.want {
				t.Errorf("Requeue(%+v, %+v) = %v, want %v", a, tt.b, got, tt.want)
			}
		})
	}
}

// TestWait_CeilingAndNoTrainer verifies the expected wait: applicants ahead
// times ticks over places, rounded up; none with no place. The example of
// spec-skill-barriers.md 4.9 has 30 places and 60 ticks.
//
// Requirement: FR-SKL-019
func TestWait_CeilingAndNoTrainer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		ahead, ticks, max int64
		want              int64
		wantOK            bool
	}{
		// 15 * 60 / 30 = 30 exactly.
		{"exact", 15, 60, 30, 30, true},
		// 7 * 60 / 30 = 14.
		{"seven ahead", 7, 60, 30, 14, true},
		// 1 * 60 / 30 = 2; 1 * 61 / 30 = 2.03 rounds up to 3.
		{"rounds up", 1, 61, 30, 3, true},
		{"nobody ahead waits 0", 0, 60, 30, 0, true},
		{"a course of 0 ticks waits 0", 5, 0, 30, 0, true},
		{"one place", 3, 60, 1, 180, true},
		{"no trainer", 3, 60, 0, 0, false},
		{"negative places", 3, 60, -1, 0, false},
		{"negative ahead", -1, 60, 30, 0, false},
		{"negative ticks", 1, -60, 30, 0, false},
		{"overflow", math.MaxInt64, 2, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := skill.Wait(tt.ahead, tt.ticks, tt.max)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Wait(%d, %d, %d) = %d, %v, want %d, %v", tt.ahead, tt.ticks, tt.max, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestComplete_HeldDoneServing verifies the completion test at the entry step:
// serving below TrainingTicks, done at the exact boundary with headroom or no
// limit, held without headroom.
//
// Requirement: FR-SKL-015
func TestComplete_HeldDoneServing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                         string
		served, ticks, headroom, wgt int64
		noLimit                      bool
		want                         skill.Outcome
	}{
		{"one tick short", 119, 120, 5, 1, false, skill.Serving},
		{"exact boundary with headroom", 120, 120, 1, 1, false, skill.Done},
		{"exact boundary, headroom below weight", 120, 120, 0, 1, false, skill.Held},
		{"no limit overrides headroom", 120, 120, 0, 1, true, skill.Done},
		{"served beyond ticks is still done", 121, 120, 1, 1, false, skill.Done},
		{"serving ignores the limit", 5, 120, 0, 1, true, skill.Serving},
		{"headroom above weight", 120, 120, 5, 1, false, skill.Done},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := skill.Complete(tt.served, tt.ticks, tt.headroom, tt.wgt, tt.noLimit); got != tt.want {
				t.Errorf("Complete(%d, %d, %d, %d, %v) = %d, want %d", tt.served, tt.ticks, tt.headroom, tt.wgt, tt.noLimit, got, tt.want)
			}
		})
	}
}

// TestNextRun_TenthAndEveryTenth verifies that the run counter warns at the
// tenth consecutive blocked tick and every tenth after, and resets when a tick
// is not blocked.
//
// Requirement: FR-SKL-022
func TestNextRun_TenthAndEveryTenth(t *testing.T) {
	t.Parallel()
	run, warns := int64(0), []int64{}
	for tick := int64(1); tick <= 31; tick++ {
		var w bool
		run, w = skill.NextRun(run, true)
		if w {
			warns = append(warns, tick)
		}
	}
	if len(warns) != 3 || warns[0] != 10 || warns[1] != 20 || warns[2] != 30 {
		t.Errorf("warn ticks over 31 blocked ticks = %v, want [10 20 30]", warns)
	}
	if run != 31 {
		t.Errorf("run after 31 blocked ticks = %d, want 31", run)
	}
	if r, w := skill.NextRun(9, false); r != 0 || w {
		t.Errorf("NextRun(9, false) = %d, %v, want 0, false", r, w)
	}
	if r, w := skill.NextRun(math.MaxInt64, true); r != math.MaxInt64 || w {
		t.Errorf("NextRun(MaxInt64, true) = %d, %v, want MaxInt64, false", r, w)
	}
}

// TestBreakEven_WorkedExample verifies the break-even premium of
// spec-skill-barriers.md 4.8: capital 219 000 + 1 277 500 = 1 496 500, so 10 000
// + 1 496 500 * 1 000 / (250 * 2 190) = 12 733; training alone 12 333; tool alone
// 10 400.
//
// Requirement: FR-SKL-024
func TestBreakEven_WorkedExample(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                   string
		price, ctrain, capReturn, dwork, wUnsk int64
		want                                   int64
		wantOK                                 bool
	}{
		{"tool and training", 219_000, 1_277_500, 1000, 250, 2190, 12_733, true},
		{"training alone", 0, 1_277_500, 1000, 250, 2190, 12_333, true},
		{"tool alone", 219_000, 0, 1000, 250, 2190, 10_400, true},
		{"free", 0, 0, 1000, 250, 2190, 10_000, true},
		{"no working days", 1, 1, 1000, 0, 2190, 0, false},
		{"negative working days", 1, 1, 1000, -250, 2190, 0, false},
		{"zero required return", 1, 1, 0, 250, 2190, 10_000, true},
		{"negative price", -1, 1, 1000, 250, 2190, 0, false},
		{"negative training", 1, -1, 1000, 250, 2190, 0, false},
		{"negative return", 1, 1, -1, 250, 2190, 0, false},
		{"capital overflow", math.MaxInt64, 1, 1000, 250, 2190, 0, false},
		{"denominator overflow", 1, 1, 1000, math.MaxInt64, 2, 0, false},
		{"excess overflow", math.MaxInt64, 0, math.MaxInt64, 1, 1, 0, false},
		{"sum overflow", math.MaxInt64, 0, 10000, 10000, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := skill.BreakEven(tt.price, tt.ctrain, tt.capReturn, tt.dwork, tt.wUnsk)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("BreakEven(%d, %d, %d, %d, %d) = %d, %v, want %d, %v", tt.price, tt.ctrain, tt.capReturn, tt.dwork, tt.wUnsk, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestTolerance_ShareFloorAndCV verifies the tolerance of check D: 25% of the
// excess, a floor of 200 basis points, and twice the coefficient of variation
// when larger (spec-skill-barriers.md 3: excess 2 733 gives 683; a tool-only
// excess of 400 gives 100, raised to the floor of 200).
//
// Requirement: FR-SKL-024
func TestTolerance_ShareFloorAndCV(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		excess, cv2 int64
		want        int64
	}{
		{"share of 2 733", 2733, 0, 683},
		{"floor for a tool-only chain", 400, 0, 200},
		{"cv raises it", 2733, 900, 900},
		{"cv below the share is ignored", 2733, 100, 683},
		{"negative excess gives the floor", -50, 0, 200},
		{"an excess of 800 is exactly the floor", 800, 0, 200},
		{"an excess of 804 is above it", 804, 0, 201},
		{"excess of one full 10 000", 10_000, 0, 2500},
		{"excess of 10 003", 10_003, 0, 2500},
		{"huge excess does not overflow", math.MaxInt64, 0, math.MaxInt64 / 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := skill.Tolerance(tt.excess, tt.cv2); got != tt.want {
				t.Errorf("Tolerance(%d, %d) = %d, want %d", tt.excess, tt.cv2, got, tt.want)
			}
		})
	}
}

// TestPackage_NoFloatNoRoot verifies that the package imports the standard
// library and internal/checked only, and uses no floating point, no map, no
// time or random source.
//
// Requirement: FR-SKL-028
func TestPackage_NoFloatNoRoot(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"github.com/Whealinghold/economy-sim/internal/checked": true}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		for _, imp := range af.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if strings.Contains(path, ".") && !allowed[path] {
				t.Errorf("%s imports %s, want the standard library and internal/checked only", f, path)
			}
			if strings.HasPrefix(path, "math/rand") || path == "time" || path == "crypto/rand" {
				t.Errorf("%s imports %s, want no clock or random source", f, path)
			}
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range []string{"float32", "float64", "map["} {
			if strings.Contains(string(src), bad) {
				t.Errorf("%s contains %q, want no floating point or map", f, bad)
			}
		}
	}
}
