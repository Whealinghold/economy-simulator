package verdict_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// span returns m values with the given minimum and maximum.
func span(m int, lo, hi int64) []int64 {
	v := make([]int64, m)
	for i := range v {
		v[i] = lo + (hi-lo)*int64(i)/int64(m-1)
	}
	return v
}

// TestSetBand_HeadroomAndRounding verifies the band rule of STY 8.4.3: the
// headroom by the number of setting seeds, the half-to-even headroom quotient,
// the outward rounding to two significant digits and the refusals.
//
// Expected values (spec-stylised-facts.md 17.1, worked.out lines 2 to 4, and
// seeds.py round_outward asserts): 50 seeds, min 70, max 161, range 91, h 2 500
// bp: 161 + 91*2 500/10 000 = 161 + 22.75 -> 23 half to even = 184, which
// rounds up to 190; 201 up is 210; 184 down is 180; -201 as an upper bound is
// -200 and as a lower bound -210; 99 is kept. With m = 10 the headroom is
// 5 000: 10..50 gives 50 + 40*5 000/10 000 = 70, and 10 - 20 = -10 down.
//
// Requirement: FR-STY-077, FR-STY-078, FR-STY-005
func TestSetBand_HeadroomAndRounding(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		vals []int64
		side verdict.Side
		lim  verdict.Limit
		want int64
	}{
		{"17.1 upper, 50 seeds, h 2500", span(50, 70, 161), verdict.Upper, verdict.Limit{}, 190},
		{"half-to-even headroom 22.75 is 23", withExtremes(50, 70, 161), verdict.Upper, verdict.Limit{}, 190},
		{"quotient exact half goes to even", withExtremes(50, 0, 2), verdict.Upper, verdict.Limit{}, 2},
		{"201 rounds up to 210", withExtremes(50, 201, 201), verdict.Upper, verdict.Limit{}, 210},
		{"184 rounds down to 180 for a lower bound", withExtremes(50, 184, 184), verdict.Lower, verdict.Limit{}, 180},
		{"-201 upper moves toward zero", withExtremes(50, -201, -201), verdict.Upper, verdict.Limit{}, -200},
		{"-201 lower moves away from zero", withExtremes(50, -201, -201), verdict.Lower, verdict.Limit{}, -210},
		{"a value under 100 is kept", withExtremes(50, 99, 99), verdict.Upper, verdict.Limit{}, 99},
		{"100 is two digits already", withExtremes(50, 100, 100), verdict.Upper, verdict.Limit{}, 100},
		{"m 10 uses 5000: 50 + 20 = 70", withExtremes(10, 10, 50), verdict.Upper, verdict.Limit{}, 70},
		{"m 10 lower: 10 - 20 = -10", withExtremes(10, 10, 50), verdict.Lower, verdict.Limit{}, -10},
		{"m 39 still uses 5000", withExtremes(39, 10, 50), verdict.Upper, verdict.Limit{}, 70},
		{"m 40 uses 2500: 50 + 10 = 60", withExtremes(40, 10, 50), verdict.Upper, verdict.Limit{}, 60},
		{"lower bound at the threshold is allowed", withExtremes(50, 100, 200), verdict.Lower, verdict.Limit{Set: true, Value: 75}, 75},
		{"upper bound at the threshold is allowed", withExtremes(50, 100, 200), verdict.Upper, verdict.Limit{Set: true, Value: 230}, 230},
	} {
		got, err := verdict.SetBand(tc.vals, tc.side, len(tc.vals), tc.lim)
		if err != nil || got != tc.want {
			t.Errorf("SetBand(%s) = %d, %v, want %d", tc.name, got, err, tc.want)
		}
	}

	t.Run("refusals", func(t *testing.T) {
		if _, err := verdict.SetBand(withExtremes(9, 1, 2), verdict.Upper, 9, verdict.Limit{}); !errors.Is(err, verdict.ErrTooFewSeeds) {
			t.Errorf("SetBand(m 9) error = %v, want ErrTooFewSeeds", err)
		}
		// 50 + 20 = 70 is above a threshold of 69: refused; the bound is returned for review.
		got, err := verdict.SetBand(withExtremes(10, 10, 50), verdict.Upper, 10, verdict.Limit{Set: true, Value: 69})
		if !errors.Is(err, verdict.ErrAboveThreshold) || got != 70 {
			t.Errorf("SetBand(upper 70 against 69) = %d, %v, want 70 and ErrAboveThreshold", got, err)
		}
		// a lower bound below its floor: 10 - 20 = -10 under a floor of 0.
		if _, err := verdict.SetBand(withExtremes(10, 10, 50), verdict.Lower, 10, verdict.Limit{Set: true, Value: 0}); !errors.Is(err, verdict.ErrAboveThreshold) {
			t.Errorf("SetBand(lower -10 against floor 0) error = %v, want ErrAboveThreshold", err)
		}
	})
	t.Run("extremes of int64 are an error, not a wrap", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			lo   int64
			hi   int64
			side verdict.Side
		}{
			{"span overflows", math.MinInt64, math.MaxInt64, verdict.Upper},
			{"span times h overflows", 0, math.MaxInt64, verdict.Upper},
			{"upper at the top", math.MaxInt64, math.MaxInt64, verdict.Upper},
			{"lower at the bottom", math.MinInt64, math.MinInt64, verdict.Lower},
			{"upper rounding at the top", math.MaxInt64 - 5, math.MaxInt64 - 5, verdict.Upper},
			{"headroom pushes past the top", math.MaxInt64 - 1000000, math.MaxInt64 - 10, verdict.Upper},
			{"headroom pushes past the bottom", math.MinInt64 + 10, math.MinInt64 + 1000000, verdict.Lower},
		} {
			if got, err := verdict.SetBand(withExtremes(50, tc.lo, tc.hi), tc.side, 50, verdict.Limit{}); err == nil {
				t.Errorf("SetBand(%s) = %d, nil, want an overflow error", tc.name, got)
			}
		}
	})
}

// withExtremes returns m values with exactly the given minimum and maximum.
func withExtremes(m int, lo, hi int64) []int64 {
	v := make([]int64, m)
	for i := range v {
		v[i] = lo
	}
	v[m-1] = hi
	return v
}

// FuzzSetBand verifies that the outward rounding never lands inside the
// extreme it is set from, and that the rule returns an error rather than wrap
// or panic, including on the extremes of int64.
//
// Requirement: NFR-STY-003
func FuzzSetBand(f *testing.F) {
	f.Add(int64(70), int64(161), true, 50)
	f.Add(int64(math.MinInt64), int64(math.MaxInt64), false, 10)
	f.Add(int64(-201), int64(-201), true, 40)
	f.Fuzz(func(t *testing.T, lo, hi int64, upper bool, m int) {
		if lo > hi {
			lo, hi = hi, lo
		}
		if m < 10 || m > 200 {
			m = 10 + (m%190+190)%190
		}
		side := verdict.Lower
		if upper {
			side = verdict.Upper
		}
		v := withExtremes(m, lo, hi)
		got, err := verdict.SetBand(v, side, m, verdict.Limit{})
		if err != nil {
			return
		}
		if upper && got < hi || !upper && got > lo {
			t.Fatalf("SetBand(min %d max %d, upper %v, m %d) = %d, inside the extreme", lo, hi, upper, m, got)
		}
	})
}
