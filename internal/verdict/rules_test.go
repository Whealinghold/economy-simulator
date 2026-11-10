package verdict_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// TestSeedRules_Fields verifies every field of the three seed rules, for the
// pass, the fail and the nothing-planned cases, with the boundaries of SR3's
// band inclusive, an unsorted value list, odd and even counts, and the message
// text for seeds, minimum, median and maximum of two digits.
//
// Requirement: FR-STY-076, FR-STY-086
func TestSeedRules_Fields(t *testing.T) {
	t.Parallel()
	if got, want := verdict.SR1(10, nil), (verdict.SeedResult{Rule: "SR1", Pass: true, N: 10, Need: 10}); !sameResult(got, want) {
		t.Errorf("SR1(10, none) = %+v, want %+v", got, want)
	}
	for _, tc := range []struct {
		n    int
		out  []int64
		pass bool
	}{{1, nil, true}, {1, []int64{1}, false}, {0, nil, false}, {-1, nil, false}} {
		if got := verdict.SR1(tc.n, tc.out).Pass; got != tc.pass {
			t.Errorf("SR1(%d, %v).Pass = %v, want %v", tc.n, tc.out, got, tc.pass)
		}
	}
	if got := verdict.SR2(0, nil); got.Need != 0 || got.Pass || got.Rule != "SR2" {
		t.Errorf("SR2(0, none) = %+v, want need 0, fail", got)
	}
	if got := verdict.SR2(1, nil); !got.Pass || got.Need != 1 {
		t.Errorf("SR2(1, none) = %+v, want pass, need 1", got)
	}
	if got := verdict.SR3(0, nil, 0, 1); got.Need != 0 || got.Rule != "SR3" {
		t.Errorf("SR3(0, none) = %+v, want need 0", got)
	}

	for _, tc := range []struct {
		name   string
		n      int
		vals   []int64
		lo, hi int64
		pass   bool
		median int64
	}{
		{"unsorted odd", 3, []int64{9, 1, 5}, 0, 10, true, 5},
		{"unsorted even", 4, []int64{9, 1, 5, 7}, 0, 10, true, 6},
		{"median at the lower bound", 3, []int64{4, 5, 9}, 5, 10, true, 5},
		{"median one below the lower bound", 3, []int64{3, 4, 9}, 5, 10, false, 4},
		{"median at the upper bound", 3, []int64{1, 5, 5}, 0, 5, true, 5},
		{"median one above the upper bound", 3, []int64{1, 6, 9}, 0, 5, false, 6},
		{"n of 1", 1, []int64{4}, 4, 4, true, 4},
		{"n of 2 even", 2, []int64{10, 12}, 11, 11, true, 11},
		{"more values than planned", 2, []int64{1, 2, 3}, 0, 9, true, 2},
		{"n of 0", 0, []int64{4}, 0, 9, false, 4},
	} {
		got := verdict.SR3(tc.n, tc.vals, tc.lo, tc.hi)
		if got.Pass != tc.pass || got.Median != tc.median || !got.HasStats {
			t.Errorf("SR3(%s: n %d, %v, [%d, %d]) = %+v, want pass %v median %d", tc.name, tc.n, tc.vals, tc.lo, tc.hi, got, tc.pass, tc.median)
		}
	}
	if got := verdict.SR3(2, []int64{12, 10}, 0, 99); got.Min != 10 || got.Max != 12 {
		t.Errorf("SR3 min and max of [12 10] = %d and %d, want 10 and 12", got.Min, got.Max)
	}

	// Message text with seeds and statistics of two digits (the base is ten).
	if got, want := verdict.SR2(50, []int64{11, 10, 12, 13, 14, 15}).String(), "SR2 fail n 50 need 45 outside 10 11 12 13 14 15"; got != want {
		t.Errorf("SR2(50, six outside).String() = %q, want %q", got, want)
	}
	if got, want := verdict.SR3(3, []int64{10, 21, 99}, 0, 99).String(), "SR3 pass n 3 need 3 outside min 10 median 21 max 99"; got != want {
		t.Errorf("SR3.String() = %q, want %q", got, want)
	}
}

func sameResult(a, b verdict.SeedResult) bool {
	return a.Rule == b.Rule && a.Pass == b.Pass && a.N == b.N && a.Need == b.Need && slices.Equal(a.Outside, b.Outside) &&
		a.Min == b.Min && a.Median == b.Median && a.Max == b.Max && a.HasStats == b.HasStats
}

// TestSetBand_Headroom verifies the headroom constants and the half-to-even
// quotient on cases where a one-basis-point change of h or a rounding
// direction changes the bound, and that the error cases return no value.
//
// Expected: with 50 seeds (h 2 500) and min 0, the upper bound is max plus
// max * 2 500 / 10 000 rounded half to even: max 2 gives 2 + 0.5 -> 2 + 0; max 6
// gives 6 + 1.5 -> 6 + 2; max 10 gives 10 + 2.5 -> 10 + 2; max 14 gives
// 14 + 3.5 -> 14 + 4; max 79 gives 79 + 19.75 -> 79 + 20 = 99 (under 100, kept).
// With 10 seeds (h 5 000): max 1 gives 1 + 0.5 -> 1; max 3 gives 3 + 1.5 -> 5;
// max 5 gives 5 + 2.5 -> 7.
//
// Requirement: FR-STY-077
func TestSetBand_Headroom(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		m       int
		max, ub int64
	}{
		{50, 2, 2}, {50, 6, 8}, {50, 10, 12}, {50, 14, 18}, {50, 79, 99},
		{40, 6, 8}, {39, 6, 9}, {10, 1, 1}, {10, 3, 5}, {10, 5, 7},
	} {
		got, err := verdict.SetBand(withExtremes(tc.m, 0, tc.max), verdict.Upper, tc.m, verdict.Limit{})
		if err != nil || got != tc.ub {
			t.Errorf("SetBand(m %d, 0..%d, upper) = %d, %v, want %d", tc.m, tc.max, got, err, tc.ub)
		}
		// the same range as a lower bound from min 100: 100 - add, under 100 kept.
		lb, err := verdict.SetBand(withExtremes(tc.m, 100, 100+tc.max), verdict.Lower, tc.m, verdict.Limit{})
		if want := 100 - (tc.ub - tc.max); err != nil || lb != want {
			t.Errorf("SetBand(m %d, 100..%d, lower) = %d, %v, want %d", tc.m, 100+tc.max, lb, err, want)
		}
	}
	t.Run("errors return no value", func(t *testing.T) {
		const maxI, minI = 1<<63 - 1, -1 << 63
		for _, tc := range []struct {
			name string
			vals []int64
			m    int
			want error
		}{
			{"nine seeds", withExtremes(9, 0, 1), 9, verdict.ErrTooFewSeeds},
			{"eleven values for ten seeds", withExtremes(11, 0, 1), 10, verdict.ErrSeedCount},
			{"nine values for ten seeds", withExtremes(9, 0, 1), 10, verdict.ErrSeedCount},
			{"span overflows (negative low)", withExtremes(50, -5, maxI), 50, verdict.ErrOverflow},
			{"span overflows (min and max)", withExtremes(50, minI, maxI), 50, verdict.ErrOverflow},
			{"span times h overflows", withExtremes(50, 0, maxI/2500+1), 50, verdict.ErrOverflow},
			{"upper past the top", withExtremes(50, maxI-1000000, maxI-10), 50, verdict.ErrOverflow},
			{"lower past the bottom", withExtremes(50, minI+10, minI+1000000), 50, verdict.ErrOverflow},
		} {
			side := verdict.Upper
			if tc.name == "lower past the bottom" {
				side = verdict.Lower
			}
			got, err := verdict.SetBand(tc.vals, side, tc.m, verdict.Limit{})
			if !errors.Is(err, tc.want) || got != 0 {
				t.Errorf("SetBand(%s) = %d, %v, want 0 and %v", tc.name, got, err, tc.want)
			}
		}
		// the largest span that fits does not overflow.
		if got, err := verdict.SetBand(withExtremes(50, 0, maxI/2500), verdict.Upper, 50, verdict.Limit{}); err != nil || got <= maxI/2500 {
			t.Errorf("SetBand(span max/2500) = %d, %v, want a bound above the maximum and no error", got, err)
		}
		// the rounding of the bound itself can overflow: 9.22e18 rounds up past int64.
		if got, err := verdict.SetBand(withExtremes(50, maxI-200, maxI-100), verdict.Upper, 50, verdict.Limit{}); !errors.Is(err, verdict.ErrOverflow) || got != 0 {
			t.Errorf("SetBand(upper rounding past the top) = %d, %v, want 0 and ErrOverflow", got, err)
		}
		if got, err := verdict.SetBand(withExtremes(50, minI+100, minI+200), verdict.Lower, 50, verdict.Limit{}); !errors.Is(err, verdict.ErrOverflow) || got != 0 {
			t.Errorf("SetBand(lower rounding past the bottom) = %d, %v, want 0 and ErrOverflow", got, err)
		}
	})
	t.Run("limit", func(t *testing.T) {
		v := withExtremes(10, 100, 200) // upper 200 + 50 = 250, lower 100 - 50 = 50
		for _, tc := range []struct {
			side verdict.Side
			lim  int64
			want error
		}{
			{verdict.Upper, 250, nil}, {verdict.Upper, 249, verdict.ErrAboveThreshold},
			{verdict.Lower, 50, nil}, {verdict.Lower, 51, verdict.ErrAboveThreshold},
			{verdict.Upper, 251, nil}, {verdict.Lower, 49, nil},
		} {
			if _, err := verdict.SetBand(v, tc.side, 10, verdict.Limit{Set: true, Value: tc.lim}); !errors.Is(err, tc.want) {
				t.Errorf("SetBand(side %d, limit %d) error = %v, want %v", tc.side, tc.lim, err, tc.want)
			}
		}
		// an unset limit is no limit, whatever its value.
		if _, err := verdict.SetBand(v, verdict.Upper, 10, verdict.Limit{Value: 1}); err != nil {
			t.Errorf("SetBand(unset limit) error = %v, want nil", err)
		}
	})
	t.Run("rounding to two digits", func(t *testing.T) {
		// 100 stays (under 100 means below 100), 101 and 199 round to 110 and 200 up and 100 and 190 down.
		for _, tc := range []struct {
			v    int64
			side verdict.Side
			want int64
		}{
			{99, verdict.Upper, 99}, {100, verdict.Upper, 100}, {101, verdict.Upper, 110}, {199, verdict.Upper, 200},
			{101, verdict.Lower, 100}, {199, verdict.Lower, 190}, {-99, verdict.Upper, -99}, {-100, verdict.Lower, -100},
			{-101, verdict.Upper, -100}, {-199, verdict.Lower, -200}, {2001, verdict.Upper, 2100}, {2000, verdict.Upper, 2000},
			{-2001, verdict.Lower, -2100}, {1000, verdict.Upper, 1000}, {1001, verdict.Upper, 1100}, {123456, verdict.Lower, 120000},
		} {
			got, err := verdict.SetBand(withExtremes(50, tc.v, tc.v), tc.side, 50, verdict.Limit{})
			if err != nil || got != tc.want {
				t.Errorf("SetBand(%d, side %d) = %d, %v, want %d", tc.v, tc.side, got, err, tc.want)
			}
		}
	})
}

// TestErrors_Text verifies that every sentinel error has its own message.
func TestErrors_Text(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, e := range []error{verdict.ErrTooFewSeeds, verdict.ErrSeedCount, verdict.ErrAboveThreshold, verdict.ErrOverflow, verdict.ErrNoRuns, verdict.ErrNoSeparation} {
		if e.Error() == "" || seen[e.Error()] {
			t.Errorf("error %q is empty or repeated", e)
		}
		seen[e.Error()] = true
	}
}

// TestSeparate_Edges verifies the no-value result on every error and the
// midpoint rounding at the top of the range.
//
// Requirement: FR-STY-071, FR-STY-074
func TestSeparate_Edges(t *testing.T) {
	t.Parallel()
	const maxI, minI = 1<<63 - 1, -1 << 63
	for _, tc := range []struct {
		name string
		h, p []int64
		low  bool
		want error
	}{
		{"no healthy", nil, []int64{1}, false, verdict.ErrNoRuns},
		{"no provoked", []int64{1}, nil, false, verdict.ErrNoRuns},
		{"equal", []int64{5}, []int64{5}, false, verdict.ErrNoSeparation},
		{"healthy min int64 negated", []int64{minI}, []int64{0}, true, verdict.ErrOverflow},
		{"provoked min int64 negated", []int64{0}, []int64{minI}, true, verdict.ErrOverflow},
	} {
		if got, err := verdict.Separate(tc.h, tc.p, verdict.Ticks, tc.low); !errors.Is(err, tc.want) || got != 0 {
			t.Errorf("Separate(%s) = %d, %v, want 0 and %v", tc.name, got, err, tc.want)
		}
	}
	// a provoked statistic of 1 with a healthy of 0 gives t_lo, and ticks up to 5 never leave the window.
	if got, err := verdict.Separate([]int64{0}, []int64{1}, verdict.Ticks, false); err != nil || got != 1 {
		t.Errorf("Separate([0], [1]) = %d, %v, want 1", got, err)
	}
	// negative multiples and the largest values: the rounding up stays inside the window.
	for _, tc := range []struct{ hi, lo, want int64 }{
		{maxI - 3, maxI, maxI - 1}, // ceil((2*max-3)/2) = max-1 is not a multiple of 5 and the next would pass t_lo? max-1 -> +4 overflows: t_lo
		{maxI - 8, maxI - 2, maxI - 5},
		{-12, -3, -5},
		{-7, -2, -5},
	} {
		got, err := verdict.Separate([]int64{tc.hi}, []int64{tc.lo}, verdict.Ticks, false)
		if err != nil || got < tc.hi+1 || got > tc.lo {
			t.Errorf("Separate([%d], [%d]) = %d, %v, want a value within [%d, %d]", tc.hi, tc.lo, got, err, tc.hi+1, tc.lo)
		}
	}
	if got, _ := verdict.Separate([]int64{-12}, []int64{-3}, verdict.Ticks, false); got != -5 {
		t.Errorf("Separate([-12], [-3], ticks) = %d, want -5 (ceil(-15/2) = -7, up to -5)", got)
	}
	// ceil(-9/2) = -4 rounds up to 0, above t_lo = -2: the key is t_lo.
	if got, _ := verdict.Separate([]int64{-7}, []int64{-2}, verdict.Ticks, false); got != -2 {
		t.Errorf("Separate([-7], [-2], ticks) = %d, want -2 (clamped to t_lo)", got)
	}
}

// TestRegrowTicks_Edges verifies the iteration bounds of RegrowTicks and its
// agreement with the plain per-tick iteration on a grid of small cases.
//
// Requirement: FR-STY-036, FR-STY-063
func TestRegrowTicks_Edges(t *testing.T) {
	t.Parallel()
	// With one tick a year the bound is 1 000 iterations: the level reached after
	// exactly 1 000 ticks takes 1 000, and one more takes more than the bound.
	h, carry := int64(10000), int64(0)
	for i := 0; i < 1000; i++ {
		num := h*1 + carry
		h += num / 10000
		carry = num % 10000
	}
	if got, ok := verdict.RegrowTicks(10000, h, 0, 1, 1); !ok || got != 1000 {
		t.Errorf("RegrowTicks(10000, %d, 0, 1, 1) = %d, %v, want 1000, true", h, got, ok)
	}
	if got, ok := verdict.RegrowTicks(10000, h+1, 0, 1, 1); ok || got != 0 {
		t.Errorf("RegrowTicks(10000, %d, 0, 1, 1) = %d, %v, want 0, false", h+1, got, ok)
	}
	// a forged negative band must not turn the target into an overflow.
	if got, ok := verdict.RegrowTicks(5, 1<<62, -5000, 100, 365); ok && got == 0 {
		t.Errorf("RegrowTicks(5, 2^62, -5000, ...) = 0, true, want the level unreachable")
	}
	// the longest calendar accepted is 4 096 ticks a year.
	if _, ok := verdict.RegrowTicks(820, 1000, 1000, 200, 4096); !ok {
		t.Errorf("RegrowTicks(..., 4096) unreachable, want reachable")
	}
	if got, ok := verdict.RegrowTicks(820, 1000, 1000, 200, 4097); ok || got != 0 {
		t.Errorf("RegrowTicks(..., 4097) = %d, %v, want 0, false", got, ok)
	}
	// already at or above the level, and a band above 10 000 or a negative level read as a target of 0.
	for _, tc := range []struct{ trough, level, band int64 }{{950, 1000, 1000}, {900, 1000, 1000}, {5, 1000000000, 20000}, {0, -5, 0}} {
		if got, ok := verdict.RegrowTicks(tc.trough, tc.level, tc.band, 200, 365); !ok || got != 0 {
			t.Errorf("RegrowTicks(%+v) = %d, %v, want 0, true", tc, got, ok)
		}
	}
	if got, ok := verdict.RegrowTicks(0, 5, 0, 200, 365); ok || got != 0 {
		t.Errorf("RegrowTicks(trough 0) = %d, %v, want 0, false", got, ok)
	}
	for _, tc := range []struct{ trough, level, band, rate, tpy int64 }{
		{820, 1000, 1000, 0, 365}, {820, 1000, 1000, 200, 0}, {820, 1000, 1000, 200, -1}, {820, 1000, 1000, -1, 365},
	} {
		if got, ok := verdict.RegrowTicks(tc.trough, tc.level, tc.band, tc.rate, tc.tpy); ok || got != 0 {
			t.Errorf("RegrowTicks(%+v) = %d, %v, want 0, false", tc, got, ok)
		}
	}
	// a grid against the plain iteration (hand, with the same 1 000 year bound).
	for _, trough := range []int64{3, 7, 13, 50, 211} {
		for _, level := range []int64{20, 100, 1000} {
			for _, rate := range []int64{37, 500, 9999, 20000} {
				for _, tpy := range []int64{1, 7, 52} {
					target := level * 9000 / 10000
					if trough >= target {
						continue
					}
					want, wantOK := handRegrowBounded(trough, target, rate, tpy)
					got, ok := verdict.RegrowTicks(trough, level, 1000, rate, tpy)
					if ok != wantOK || got != want {
						t.Errorf("RegrowTicks(%d, %d, 1000, %d, %d) = %d, %v, want %d, %v", trough, level, rate, tpy, got, ok, want, wantOK)
					}
				}
			}
		}
	}
}

// handRegrowBounded is the plain iteration of POP 3.8 with the 1 000 year bound.
func handRegrowBounded(h, target, rate, tpy int64) (int64, bool) {
	den := 10000 * tpy
	var carry int64
	for t := int64(1); t <= 1000*tpy; t++ {
		num := h*rate + carry
		h += num / den
		carry = num % den
		if h >= target {
			return t, true
		}
	}
	return 0, false
}

// TestNames_Exact verifies the words of the classes, the reasons and the
// detectors, which the verdict record spells as written.
//
// Requirement: FR-STY-085, FR-STY-060
func TestNames_Exact(t *testing.T) {
	t.Parallel()
	for c, want := range map[verdict.Class]string{verdict.None: "none", verdict.Detected: "detected", verdict.Unknown: "unknown"} {
		if got := c.String(); got != want {
			t.Errorf("Class(%d).String() = %q, want %q", c, got, want)
		}
	}
	reasons := []string{verdict.ReasonWindows, verdict.ReasonStale, verdict.ReasonActive, verdict.ReasonThresholdUnset, verdict.ReasonReferenceMissing, verdict.ReasonNeedsLongerRun, verdict.ReasonClamped, verdict.ReasonOutcomes}
	want := []string{"windows", "stale", "active", "threshold unset", "reference missing", "needs a longer run", "clamped", "outcomes"}
	if !slices.Equal(reasons, want) {
		t.Errorf("reasons = %q, want %q", reasons, want)
	}
	var names []string
	for _, d := range verdict.Detectors() {
		names = append(names, string(d))
	}
	if want := []string{"R1", "R2", "R3", "R4", "R5a", "R5b", "R6", "R7", "R8", "R9"}; !slices.Equal(names, want) {
		t.Errorf("Detectors() = %q, want %q", names, want)
	}
}
