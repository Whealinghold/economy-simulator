package verdict_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// TestSeedRules_SR1SR2SR3 verifies the three pass rules of STY 8.3 on the
// planned count n: SR1 needs every seed inside, SR2 needs ceil(9n/10), SR3
// needs the median inside; the outside seeds are listed ascending, a failed
// world is an outside seed, and n is the number planned.
//
// The needs are ceil(9n/10): 9 of 10 (90/10), 36 of 40 (360/10), 45 of 50
// (450/10), 4 of 4 (36/10 = 3.6, up to 4), 1 of 1 (0.9 up to 1).
// The worked values of STY 17.2 (spec-stylised-facts.md, worked.out line 7):
// 41, 55, 58, 60, 62, 64, 66, 70, 74, 103 against an upper bound of 90 gives
// nine inside, SR1 fails, SR2 passes, the median 63 is inside.
//
// Requirement: FR-STY-076, FR-STY-086
func TestSeedRules_SR1SR2SR3(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		n, need int
	}{{1, 1}, {2, 2}, {4, 4}, {9, 9}, {10, 9}, {11, 10}, {40, 36}, {50, 45}} {
		r := verdict.SR2(tc.n, nil)
		if r.Need != tc.need || !r.Pass {
			t.Errorf("SR2(%d, none) = need %d pass %v, want need %d pass true", tc.n, r.Need, r.Pass, tc.need)
		}
		over := make([]int64, 0, tc.n)
		for i := 0; i < tc.n-tc.need+1; i++ {
			over = append(over, int64(i+1))
		}
		if r := verdict.SR2(tc.n, over); r.Pass {
			t.Errorf("SR2(%d, %d outside) passed, want fail (need %d inside)", tc.n, len(over), tc.need)
		}
		if r := verdict.SR2(tc.n, over[:tc.n-tc.need]); !r.Pass {
			t.Errorf("SR2(%d, %d outside) failed, want pass", tc.n, tc.n-tc.need)
		}
	}

	vals := []int64{41, 55, 58, 60, 62, 64, 66, 70, 74, 103}
	var outside []int64
	for i, v := range vals {
		if v > 90 {
			outside = append(outside, int64(i+1))
		}
	}
	if r := verdict.SR1(10, outside); r.Pass || !slices.Equal(r.Outside, []int64{10}) {
		t.Errorf("SR1(10, %v) = %+v, want fail with seed 10 listed", outside, r)
	}
	if r := verdict.SR2(10, outside); !r.Pass || r.Need != 9 {
		t.Errorf("SR2(10, %v) = %+v, want pass with need 9", outside, r)
	}
	r3 := verdict.SR3(10, vals, 0, 90)
	if !r3.Pass || r3.Median != 63 || r3.Min != 41 || r3.Max != 103 {
		t.Errorf("SR3(10, 17.2 values, 0, 90) = %+v, want pass median 63 min 41 max 103", r3)
	}

	t.Run("listed ascending without repeats", func(t *testing.T) {
		got := verdict.SR2(10, []int64{7, 3, 7, 5}).Outside
		if !slices.Equal(got, []int64{3, 5, 7}) {
			t.Errorf("SR2(10, [7 3 7 5]).Outside = %v, want [3 5 7]", got)
		}
	})
	t.Run("a failed world is outside and n is the plan", func(t *testing.T) {
		// 9 values for 10 planned seeds: SR3 does not pass by having fewer
		// runs to judge (8.3.3).
		r := verdict.SR3(10, vals[:9], 0, 90)
		if r.Pass {
			t.Errorf("SR3(10, 9 values) passed, want fail: a failed world is outside")
		}
		if r := verdict.SR1(10, []int64{4}); r.Pass {
			t.Errorf("SR1(10, [4]) passed, want fail")
		}
	})
	t.Run("median of an even count is the half-to-even mean", func(t *testing.T) {
		// (3+4)/2 = 3.5 -> 4 (even); (4+5)/2 = 4.5 -> 4 (even); (-3-4)/2 = -3.5 -> -4.
		for _, tc := range []struct {
			v    []int64
			want int64
		}{{[]int64{3, 4}, 4}, {[]int64{4, 5}, 4}, {[]int64{-4, -3}, -4}, {[]int64{2, 2, 6, 10}, 4}, {[]int64{1, 2, 3}, 2}} {
			if got := verdict.SR3(len(tc.v), tc.v, -100, 100).Median; got != tc.want {
				t.Errorf("SR3 median of %v = %d, want %d", tc.v, got, tc.want)
			}
		}
	})
	t.Run("no seeds planned never passes", func(t *testing.T) {
		for _, r := range []verdict.SeedResult{verdict.SR1(0, nil), verdict.SR2(0, nil), verdict.SR3(0, nil, 0, 1), verdict.SR2(-3, nil)} {
			if r.Pass {
				t.Errorf("%s with no seeds planned passed", r.Rule)
			}
		}
	})
	t.Run("message text", func(t *testing.T) {
		got := verdict.SR2(10, []int64{7, 3}).String()
		if want := "SR2 fail n 10 need 9 outside 3 7"; got != want {
			t.Errorf("SeedResult.String() = %q, want %q", got, want)
		}
		got = verdict.SR3(2, []int64{5, 9}, 0, 10).String()
		if want := "SR3 pass n 2 need 2 outside min 5 median 7 max 9"; got != want {
			t.Errorf("SeedResult.String() = %q, want %q", got, want)
		}
	})
}

// TestSeedRules_Extremes verifies that ceil(9n/10) cannot overflow.
//
// Guard: the rule is computed as n - n/10 so a planned count at the limit of
// int does not wrap.
func TestSeedRules_Extremes(t *testing.T) {
	t.Parallel()
	const maxInt = int(^uint(0) >> 1)
	if r := verdict.SR2(maxInt, nil); !r.Pass || r.Need != maxInt-maxInt/10 {
		t.Errorf("SR2(maxInt, none) = %+v, want pass with need n - n/10", r)
	}
}

// FuzzSeedRules verifies that the seed rules never panic, that SR2 is monotone
// in the number of seeds outside and that the order of the outside list is
// irrelevant.
//
// Requirement: NFR-STY-003
func FuzzSeedRules(f *testing.F) {
	f.Add(10, []byte{3, 1, 2})
	f.Add(50, []byte{})
	f.Add(1, []byte{1, 1})
	f.Fuzz(func(t *testing.T, n int, raw []byte) {
		outside := make([]int64, len(raw))
		for i, b := range raw {
			outside[i] = int64(b)
		}
		r := verdict.SR2(n, outside)
		rev := slices.Clone(outside)
		slices.Reverse(rev)
		if r2 := verdict.SR2(n, rev); r2.Pass != r.Pass || !slices.Equal(r2.Outside, r.Outside) {
			t.Fatalf("SR2(%d, %v) depends on the order of the outside list", n, outside)
		}
		if len(outside) > 0 {
			fewer := verdict.SR2(n, outside[:len(outside)-1])
			if r.Pass && !fewer.Pass {
				t.Fatalf("SR2(%d) passes with %v outside but fails with one fewer: not monotone", n, outside)
			}
		}
		verdict.SR1(n, outside)
		verdict.SR3(n, outside, 0, 255)
	})
}

// TestSetBand_Errors verifies the sentinel errors of the band rule by
// errors.Is.
//
// Requirement: FR-STY-078
func TestSetBand_Errors(t *testing.T) {
	t.Parallel()
	ten := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, tc := range []struct {
		name string
		vals []int64
		m    int
		lim  verdict.Limit
		want error
	}{
		{"nine seeds", ten[:9], 9, verdict.Limit{}, verdict.ErrTooFewSeeds},
		{"a seed failed", ten[:9], 10, verdict.Limit{}, verdict.ErrSeedCount},
		{"above the threshold", ten, 10, verdict.Limit{Set: true, Value: 12}, verdict.ErrAboveThreshold},
	} {
		if _, err := verdict.SetBand(tc.vals, verdict.Upper, tc.m, tc.lim); !errors.Is(err, tc.want) {
			t.Errorf("SetBand(%s) error = %v, want %v", tc.name, err, tc.want)
		}
	}
}
