package population_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/population"
)

// TestRate_FailureAndEdges verifies the exact tuple Rate returns on each
// refusal, a zero base rate (which is valid), and a schedule longer than the
// stack buffer.
//
// Guard: a mutation that returned a stale rate or flag beside ok false, or
// refused a rate of 0, would pass the happy-path tests.
func TestRate_FailureAndEdges(t *testing.T) {
	if r, c, ok := population.Rate(nil, 0, 10_000); r != 0 || c || ok {
		t.Errorf("Rate(nil) = %d, %v, %v, want 0, false, false", r, c, ok)
	}
	if r, c, ok := population.Rate(defBirth, 0, -1); r != 0 || c || ok {
		t.Errorf("Rate(negative scale) = %d, %v, %v, want 0, false, false", r, c, ok)
	}
	if r, c, ok := population.Rate([]population.Knot{{Ratio: 0, Rate: 1 << 62}}, 0, 1<<62); r != 0 || c || ok {
		t.Errorf("Rate(overflow) = %d, %v, %v, want 0, false, false", r, c, ok)
	}
	// A rate of 0 is a valid schedule value: no event, not a refusal.
	if r, c, ok := population.Rate([]population.Knot{{Ratio: 0, Rate: 0}}, 0, 10_000); r != 0 || c || !ok {
		t.Errorf("Rate(rate 0) = %d, %v, %v, want 0, false, true", r, c, ok)
	}
	// Seventeen knots (rate = ratio / 100) take the heap path of the conversion.
	var long []population.Knot
	for i := int64(0); i < 17; i++ {
		long = append(long, population.Knot{Ratio: i * 1_000, Rate: i * 10})
	}
	if r, _, ok := population.Rate(long, 16_500, 10_000); !ok || r != 160 {
		t.Errorf("Rate(17 knots, 16500) = %d, %v, want 160, true", r, ok)
	}
	if n := testing.AllocsPerRun(100, func() { population.Rate(long[:16], 7_500, 10_000) }); n != 0 {
		t.Errorf("Rate of 16 knots allocated %v times, want 0", n)
	}
}

// TestStep_Edges verifies Step at a denominator of 1, the exact failure tuple,
// and Offset at a negative or unit denominator.
//
// Guard: the denominator guard sits on the boundary of every carry; an
// off-by-one would refuse a one-tick year or wrap a negative one.
func TestStep_Edges(t *testing.T) {
	t.Parallel()
	if e, n, ok := population.Step(0, 7, 1); e != 7 || n != 0 || !ok {
		t.Errorf("Step(0, 7, 1) = %d, %d, %v, want 7, 0, true", e, n, ok)
	}
	for _, in := range [][3]int64{{1, 1, 0}, {1, 1, -3}, {-1, 1, 10}, {1, -1, 10}, {1<<63 - 1, 1, 10}} {
		if e, n, ok := population.Step(in[0], in[1], in[2]); e != 0 || n != 0 || ok {
			t.Errorf("Step(%v) = %d, %d, %v, want 0, 0, false", in, e, n, ok)
		}
	}
	// A zero carry and a zero numerator are valid.
	if e, n, ok := population.Step(0, 0, 5); e != 0 || n != 0 || !ok {
		t.Errorf("Step(0, 0, 5) = %d, %d, %v, want 0, 0, true", e, n, ok)
	}
	if got := population.Offset(5, -100); got != 0 {
		t.Errorf("Offset(5, -100) = %d, want 0", got)
	}
	if got := population.Offset(5, 1); got != 0 {
		t.Errorf("Offset(5, 1) = %d, want 0", got)
	}
}

// TestCutAllow_Zeros verifies a zero in each input of Cut and Allow is taken as
// 0 and a negative cap as no cap.
//
// Guard: the clamps at 0 are the only defence against a negative count from a
// caller bug; each argument has its own.
func TestCutAllow_Zeros(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		due, fed, group, floor int64
		take, cut, held        int64
	}{
		{"nothing due", 0, 5, 5, 1, 0, 0, 0},
		{"nobody fed", 3, 0, 5, 1, 0, 3, 0},
		{"empty group", 3, 3, 0, 0, 0, 0, 3},
		{"no floor", 3, 3, 3, 0, 3, 0, 0},
		{"negative fed", 3, -2, 5, 1, 0, 3, 0},
		{"negative group", 3, 3, -4, 0, 0, 0, 3},
		{"negative floor", 3, 3, 3, -9, 3, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			take, cut, held := population.Cut(tt.due, tt.fed, tt.group, tt.floor)
			if take != tt.take || cut != tt.cut || held != tt.held {
				t.Errorf("Cut(%d, %d, %d, %d) = %d, %d, %d, want %d, %d, %d", tt.due, tt.fed, tt.group, tt.floor, take, cut, held, tt.take, tt.cut, tt.held)
			}
		})
	}
	allow := []struct {
		name                 string
		due, households, cap int64
		allowed, dropped     int64
	}{
		{"no households yet", 4, 0, 3, 3, 1},
		{"negative households", 4, -5, 3, 3, 1},
		{"negative cap is none", 4, 10, -1, 4, 0},
		{"negative due", -4, 0, 3, 0, 0},
		{"negative due without cap", -4, 0, 0, 0, 0},
		{"cap of 1 is a cap", 3, 0, 1, 1, 2},
	}
	for _, tt := range allow {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a, d := population.Allow(tt.due, tt.households, tt.cap)
			if a != tt.allowed || d != tt.dropped {
				t.Errorf("Allow(%d, %d, %d) = %d, %d, want %d, %d", tt.due, tt.households, tt.cap, a, d, tt.allowed, tt.dropped)
			}
		})
	}
}

// TestReplacement_Edges verifies the sign tests of the crossing: g = 0 at the
// lower knot is not a crossing, a positive pair is not one, an evaluation that
// fails and a difference that overflows are refusals with zero values.
//
// Guard: the crossing needs g(lo) < 0 <= g(hi); each comparison is one mutation
// away from reporting a ratio the population never crosses.
func TestReplacement_Edges(t *testing.T) {
	t.Parallel()
	k := func(xy ...int64) []population.Knot {
		var ks []population.Knot
		for i := 0; i < len(xy); i += 2 {
			ks = append(ks, population.Knot{Ratio: xy[i], Rate: xy[i+1]})
		}
		return ks
	}
	tests := []struct {
		name         string
		birth, death []population.Knot
	}{
		{"g is 0 then positive", k(0, 5, 10, 10), k(0, 5, 10, 5)},
		{"g is positive then larger", k(0, 60, 10, 90), k(0, 10, 10, 10)},
		{"g is positive then 0", k(0, 60, 10, 10), k(0, 10, 10, 10)},
		{"birth schedule empty", nil, k(0, 10, 10, 0)},
		{"death schedule empty", k(0, 0, 10, 10), nil},
		{"difference overflows", k(0, -(1 << 62)), k(0, 1<<62+1)},
		// g(0) = 0 - 5 = -5 and g(10) = -2^62 - (2^62 + 1) overflows: a failed difference is not a 0.
		{"difference overflows after a negative", k(0, 0, 10, -(1 << 62)), k(0, 5, 10, 1<<62+1)},
		{"step difference overflows", k(0, 0, 10, 1<<62), k(0, 1<<62, 10, 0)},
		{"ratio span overflows", k(-(1 << 62), 0, 1<<62+1, 10), k(-(1 << 62), 10, 1<<62+1, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if r, lo, hi, ok := population.Replacement(tt.birth, tt.death); r != 0 || lo != 0 || hi != 0 || ok {
				t.Errorf("Replacement(%v, %v) = %d, %d, %d, %v, want 0, 0, 0, false", tt.birth, tt.death, r, lo, hi, ok)
			}
		})
	}
}

// TestGainWarn_Boundary verifies the constant 10 000 of the condition, one unit
// above it.
//
// Guard: the constant is the one place a typo would silently move the warning.
func TestGainWarn_Boundary(t *testing.T) {
	t.Parallel()
	// 10 001 * 1 * 1 against 10 000 * 1 * 1: fires; one more unit on the right would not.
	if !population.GainWarn(10_001, 1, 1, 1, 1) {
		t.Errorf("GainWarn(10001, 1, 1, 1, 1) = false, want true")
	}
	if population.GainWarn(10_000, 1, 1, 1, 1) {
		t.Errorf("GainWarn(10000, 1, 1, 1, 1) = true, want false")
	}
}

// TestGain_Edges verifies the gain at a denominator of 1 and refuses 0 and a
// negative one with a zero value.
//
// Guard: the denominator guard decides between a gain and a division by zero.
func TestGain_Edges(t *testing.T) {
	t.Parallel()
	if g, ok := population.Gain(5, 3, 1); g != 15 || !ok {
		t.Errorf("Gain(5, 3, 1) = %d, %v, want 15, true", g, ok)
	}
	for _, d := range []int64{0, -2} {
		if g, ok := population.Gain(5, 3, d); g != 0 || ok {
			t.Errorf("Gain(5, 3, %d) = %d, %v, want 0, false", d, g, ok)
		}
	}
}

// TestMeans_Edges verifies the refusals of Means with zero values, a weight of
// 0 or 1 in the middle of the list, and a mismatch of each slice both ways.
//
// Guard: a zero-weight household must not end the scan and a short or long
// slice must not be read past its end.
func TestMeans_Edges(t *testing.T) {
	t.Parallel()
	w, b, d, p := []int64{0, 1, 3}, []int64{900, 100, 300}, []int64{0, 40, 20}, []bool{true, true, true}
	// The weight-0 entry counts for nothing; the others: b = (100 + 900) / 4 = 250, d = (40 + 60) / 4 = 25, n = (1000 - 100) / 4 = 225.
	if gb, gd, gn, ok := population.Means(w, b, d, p); gb != 250 || gd != 25 || gn != 225 || !ok {
		t.Errorf("Means = %d, %d, %d, %v, want 250, 25, 225, true", gb, gd, gn, ok)
	}
	for _, c := range []struct {
		name string
		w    []int64
		b, d []int64
		p    []bool
	}{
		{"birth short", w, b[:2], d, p},
		{"birth long", w, append(b[:3:3], 1), d, p},
		{"death short", w, b, d[:2], p},
		{"death long", w, b, append(d[:3:3], 1), p},
		{"present short", w, b, d, p[:2]},
		{"present long", w, b, d, append(p[:3:3], true)},
	} {
		if gb, gd, gn, ok := population.Means(c.w, c.b, c.d, c.p); gb != 0 || gd != 0 || gn != 0 || ok {
			t.Errorf("Means(%s) = %d, %d, %d, %v, want 0, 0, 0, false", c.name, gb, gd, gn, ok)
		}
	}
	big := int64(1) << 62
	for _, c := range []struct {
		name    string
		w, b, d []int64
	}{
		{"weights overflow", []int64{big, big}, []int64{1, 1}, []int64{1, 1}},
		{"weights overflow alone", []int64{big, big}, []int64{0, 0}, []int64{0, 0}},
		{"birth product overflows", []int64{big}, []int64{4}, []int64{1}},
		{"death product overflows", []int64{big}, []int64{1}, []int64{4}},
		{"birth sum overflows", []int64{1, 1}, []int64{big*2 - 1, 1 << 62}, []int64{0, 0}},
		{"death sum overflows", []int64{1, 1}, []int64{0, 0}, []int64{big*2 - 1, 1 << 62}},
	} {
		if gb, gd, gn, ok := population.Means(c.w, c.b, c.d, []bool{true, true}[:len(c.w)]); gb != 0 || gd != 0 || gn != 0 || ok {
			t.Errorf("Means(%s) = %d, %d, %d, %v, want 0, 0, 0, false", c.name, gb, gd, gn, ok)
		}
	}
	// A negative weight is a caller bug and is left out like an absent one.
	if gb, gd, gn, ok := population.Means([]int64{-5, 2}, []int64{900, 10}, []int64{100, 4}, []bool{true, true}); gb != 10 || gd != 4 || gn != 6 || !ok {
		t.Errorf("Means(negative weight) = %d, %d, %d, %v, want 10, 4, 6, true", gb, gd, gn, ok)
	}
	// A weight of 1 is counted.
	if gb, _, _, ok := population.Means([]int64{1}, []int64{7}, []int64{0}, []bool{true}); gb != 7 || !ok {
		t.Errorf("Means(weight 1) = %d, %v, want 7, true", gb, ok)
	}
}
