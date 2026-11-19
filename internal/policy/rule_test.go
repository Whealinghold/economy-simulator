package policy_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/policy"
)

// inlandRule is the reaction of spec-policy.md 14.2: target 500 bp, deadband
// 100, span 1 000, move 2 000, the lever rises when the reading is above target.
var inlandRule = policy.Reaction{Target: 500, Deadband: 100, Span: 1_000, Move: 2_000, Raise: true}

// TestRule_StepTable verifies the decisions of the table of 14.2: error +400
// gives trunc(400 * 2 000 / 1 000) = +800, +200 gives +400, an error of 20 is
// inside the band, +1 100 gives +2 200 (the step limit cuts it later) and -200
// gives -400.
//
// Requirement: FR-PLC-062
func TestRule_StepTable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		reading  int64
		current  int64
		wantNext int64
		wantHold policy.Hold
	}{
		{"tick 6: 900", 900, 10_000, 10_800, policy.Moved},
		{"tick 36: 700", 700, 10_800, 11_200, policy.Moved},
		{"tick 66: 520 rests", 520, 11_200, 11_200, policy.InBand},
		{"tick 96: 1 600", 1_600, 11_200, 13_400, policy.Moved},
		{"tick 126: 300", 300, 12_200, 11_800, policy.Moved},
		{"band edge above rests", 600, 10_000, 10_000, policy.InBand},
		{"band edge below rests", 400, 10_000, 10_000, policy.InBand},
		{"one past the edge moves", 601, 10_000, 10_202, policy.Moved},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			next, hold := policy.RuleStep(inlandRule, tt.reading, true, tt.current)
			if next != tt.wantNext || hold != tt.wantHold {
				t.Errorf("RuleStep(reading %d, v %d) = %d, %v; want %d, %v", tt.reading, tt.current, next, hold, tt.wantNext, tt.wantHold)
			}
		})
	}
}

// TestRule_Direction verifies that Raise false lowers the lever when the reading
// is above target and that the product truncates toward zero: trunc(-3 * 1 / 2)
// is -1 (not -2), so a falling rule adds +1.
//
// Requirement: FR-PLC-062
func TestRule_Direction(t *testing.T) {
	t.Parallel()
	fall := policy.Reaction{Target: 0, Deadband: 0, Span: 2, Move: 1, Raise: false}
	tests := []struct {
		name    string
		r       policy.Reaction
		reading int64
		want    int64
	}{
		{"falling rule, reading above", fall, 4, 100 - 2},
		{"falling rule, reading below", fall, -4, 100 + 2},
		{"truncates toward zero below", fall, -3, 100 + 1},
		{"truncates toward zero above", fall, 3, 100 - 1},
		{"raising rule, reading below", policy.Reaction{Span: 2, Move: 1, Raise: true}, -4, 100 - 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if next, hold := policy.RuleStep(tt.r, tt.reading, true, 100); next != tt.want || hold != policy.Moved {
				t.Errorf("RuleStep(reading %d) = %d, %v; want %d, Moved", tt.reading, next, hold, tt.want)
			}
		})
	}
}

// TestRule_NoReading verifies that an undefined gauge returns no entry, and that
// it is counted before the band is tested.
//
// Requirement: FR-PLC-062
func TestRule_NoReading(t *testing.T) {
	t.Parallel()
	next, hold := policy.RuleStep(inlandRule, 900, false, 10_000)
	if next != 10_000 || hold != policy.NoReading {
		t.Errorf("RuleStep(undefined) = %d, %v; want 10 000, NoReading", next, hold)
	}
}

// TestRule_UndampedAccepted verifies that a rule with no deadband acts on any
// error, even 1 (the warning rule_undamped is validation's, not the rule's).
//
// Requirement: FR-PLC-064
func TestRule_UndampedAccepted(t *testing.T) {
	t.Parallel()
	r := policy.Reaction{Target: 500, Span: 1, Move: 1, Raise: true}
	if next, hold := policy.RuleStep(r, 501, true, 10); next != 11 || hold != policy.Moved {
		t.Errorf("RuleStep(501) = %d, %v; want 11, Moved", next, hold)
	}
	if _, hold := policy.RuleStep(r, 500, true, 10); hold != policy.InBand {
		t.Errorf("RuleStep(500) hold = %v, want InBand (|e| <= 0)", hold)
	}
}

// TestRule_LargestProductDoesNotWrap verifies the 128-bit product at the
// largest move (100 000 bp) and the largest error: e * Move needs 81 bits, and a
// quotient that leaves int64 is cut to the bound in its sign, as is the sum with
// the lever's value.
//
// Requirement: FR-PLC-062
func TestRule_LargestProductDoesNotWrap(t *testing.T) {
	t.Parallel()
	r := policy.Reaction{Target: 0, Deadband: 0, Span: 1, Move: 100_000, Raise: true}
	tests := []struct {
		name    string
		reading int64
		v       int64
		want    int64
	}{
		{"huge positive error saturates up", 1 << 60, 5, math.MaxInt64},
		{"huge negative error saturates down", -(1 << 60), 5, math.MinInt64 + 5},
		{"huge negative error from a low value", -(1 << 60), -10, math.MinInt64},
		{"sum with a high lever value saturates", 1_000, math.MaxInt64 - 10, math.MaxInt64},
		{"sum with a low lever value saturates", -1_000, math.MinInt64 + 10, math.MinInt64},
		{"a product wider than 64 bits that divides back", 1 << 60, 0, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, hold := policy.RuleStep(r, tt.reading, true, tt.v); got != tt.want || hold != policy.Moved {
				t.Errorf("RuleStep(%d, v %d) = %d, %v; want %d, Moved", tt.reading, tt.v, got, hold, tt.want)
			}
		})
	}
	wide := policy.Reaction{Span: 1 << 40, Move: 100_000, Raise: true}
	// (1<<60) * 100 000 / 2^40 = 2^20 * 100 000 = 104 857 600 000, exact in 128 bits.
	if got, _ := policy.RuleStep(wide, 1<<60, true, 0); got != (1<<20)*100_000 {
		t.Errorf("RuleStep with a 2^40 span = %d, want %d", got, int64(1<<20)*100_000)
	}
	// A falling rule negates the saturated value, including the lowest int64.
	fall := policy.Reaction{Span: 1, Move: 100_000}
	if got, _ := policy.RuleStep(fall, -(1 << 60), true, 0); got != math.MaxInt64 {
		t.Errorf("falling RuleStep(-2^60) = %d, want MaxInt64", got)
	}
	// A reading at the edge of int64 against a negative target does not wrap in the error.
	edge := policy.Reaction{Target: math.MinInt64, Span: 1, Move: 1, Raise: true}
	if got, hold := policy.RuleStep(edge, math.MaxInt64, true, 0); hold != policy.Moved || got != math.MaxInt64 {
		t.Errorf("RuleStep(error beyond int64) = %d, %v; want MaxInt64, Moved", got, hold)
	}
}

// TestRule_LowestErrorSaturatesTheBand verifies that an error of the lowest int64 is
// compared with the deadband as the highest int64, never negated into itself: a
// deadband below it moves the rule, a deadband of the highest int64 holds it in band.
//
// Requirement: FR-PLC-062
//
// Guard: -MinInt64 wraps to MinInt64, so a naive |e| <= Deadband would read a huge
// negative error as inside every band.
func TestRule_LowestErrorSaturatesTheBand(t *testing.T) {
	t.Parallel()
	moving := policy.Reaction{Deadband: 100, Span: 1, Move: 1, Raise: true}
	// e = MinInt64 - 0, times Move 1 over Span 1, added to 0: MinInt64.
	if got, hold := policy.RuleStep(moving, math.MinInt64, true, 0); hold != policy.Moved || got != math.MinInt64 {
		t.Errorf("RuleStep(e = MinInt64, deadband 100) = %d, %v; want MinInt64, Moved", got, hold)
	}
	widest := policy.Reaction{Deadband: math.MaxInt64, Span: 1, Move: 1, Raise: true}
	if got, hold := policy.RuleStep(widest, math.MinInt64, true, 7); hold != policy.InBand || got != 7 {
		t.Errorf("RuleStep(e = MinInt64, deadband MaxInt64) = %d, %v; want 7, InBand", got, hold)
	}
}

// TestRule_BadSpanReadsAsNoReading verifies the guard for a span below 1, which
// validation excludes.
//
// Requirement: FR-PLC-062
func TestRule_BadSpanReadsAsNoReading(t *testing.T) {
	t.Parallel()
	if _, hold := policy.RuleStep(policy.Reaction{Span: 0, Move: 1}, 5, true, 0); hold != policy.NoReading {
		t.Errorf("RuleStep(span 0) hold = %v, want NoReading", hold)
	}
}

// TestSatAdd verifies the saturating sum of the checks.
//
// Requirement: FR-PLC-041
func TestSatAdd(t *testing.T) {
	t.Parallel()
	tests := []struct{ a, b, want int64 }{
		{1, 2, 3},
		{math.MaxInt64, 1, math.MaxInt64},
		{math.MinInt64, -1, math.MinInt64},
		{math.MaxInt64, math.MinInt64, -1},
	}
	for _, tt := range tests {
		if got := policy.SatAdd(tt.a, tt.b); got != tt.want {
			t.Errorf("SatAdd(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// FuzzRule fuzzes the step: it never panics, an undefined gauge or a reading in
// the band holds, and a moving rule moves in the direction of the error (up for
// a raising rule above target, down below it) or stays when saturated. The low
// bits of flags are Raise and defined.
//
// Requirement: FR-PLC-062
func FuzzRule(f *testing.F) {
	f.Add(int64(900), int64(500), int64(100), int64(1000), int64(2000), int64(10_000), uint8(3))
	f.Add(int64(math.MaxInt64), int64(math.MinInt64), int64(0), int64(1), int64(100_000), int64(math.MaxInt64), uint8(2))
	f.Fuzz(func(t *testing.T, reading, target, deadband, span, move, v int64, flags uint8) {
		if span < 1 || deadband < 0 || move < 0 || move > 100_000 {
			t.Skip()
		}
		raise, defined := flags&1 != 0, flags&2 != 0
		r := policy.Reaction{Target: target, Deadband: deadband, Span: span, Move: move, Raise: raise}
		next, hold := policy.RuleStep(r, reading, defined, v)
		checkRuleDirection(t, next, hold, v, defined, (reading > target) == raise && reading != target)
	})
}

// checkRuleDirection asserts that a held rule leaves the value alone and that a
// moving rule does not go against the error. up says the value should not fall.
func checkRuleDirection(t *testing.T, next int64, hold policy.Hold, v int64, defined, up bool) {
	t.Helper()
	if !defined {
		if hold != policy.NoReading || next != v {
			t.Fatalf("undefined gauge gave %d, %v", next, hold)
		}
		return
	}
	if hold != policy.Moved {
		if next != v {
			t.Fatalf("hold %v changed the value to %d", hold, next)
		}
		return
	}
	if (up && next < v) || (!up && next > v) {
		t.Fatalf("rule moved against the error: %d from %d", next, v)
	}
}
