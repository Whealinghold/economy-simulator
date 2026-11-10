package reduce_test

import (
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/reduce"
)

// The ten readings of doc/specs/models/exp/reduce.out, a price range in basis
// points of the base, sorted.
var reduceOut = []int64{10100, 10200, 10300, 10400, 10500, 10600, 10700, 10900, 11800, 12100}

// TestQuantile_NearestRank verifies the rank rule max(1, ceil(bp*n/10000)): the
// five quantiles of reduce.out (rank 1, 3, 5, 8, 9 of ten), two readings (rank 1
// up to 5 000 bp and rank 2 above), one reading, and an empty slice.
//
// Requirement: FR-EXP-042
func TestQuantile_NearestRank(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []int64
		bp   int
		want int64
	}{
		// ceil(1000*10/10000) = 1, ceil(2500*10/10000) = 3, 5, 8 and 9: rank k in reduce.out.
		{"ten readings at 1 000", reduceOut, 1000, 10100},
		{"ten readings at 2 500", reduceOut, 2500, 10300},
		{"ten readings at 5 000", reduceOut, 5000, 10500},
		{"ten readings at 7 500", reduceOut, 7500, 10900},
		{"ten readings at 9 000", reduceOut, 9000, 11800},
		{"two readings at 5 000 is rank 1", []int64{4, 9}, 5000, 4},
		{"two readings at 5 001 is rank 2", []int64{4, 9}, 5001, 9},
		{"two readings at 1 000 is rank 1", []int64{4, 9}, 1000, 4},
		{"one reading at any quantile", []int64{7}, 9000, 7},
		{"zero basis points is rank 1", reduceOut, 0, 10100},
		// 3 * 6667 = 20 001 = 2 * 10 000 + 1, so the rank is ceil = 3: the value at index 2.
		{"a remainder of one rounds the rank up", ramp(6667), 3, 2},
		{"above 10 000 is the last", reduceOut, 12000, 12100},
		{"empty returns 0", nil, 5000, 0},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := reduce.Quantile(tt.in, tt.bp); got != tt.want {
				t.Errorf("Quantile(%v, %d) = %d, want %d", tt.in, tt.bp, got, tt.want)
			}
		})
	}
}

func moments(t *testing.T, vs ...int64) reduce.Moments {
	t.Helper()
	var m reduce.Moments
	for _, v := range vs {
		if !m.Add(v) {
			t.Fatalf("Add(%d) overflowed", v)
		}
	}
	return m
}

// TestMoments_MeanAndStd verifies the mean and the sample standard deviation on
// the worked cases: reduce.out (mean 10 760, std 674) and the two readings 10 400
// and 10 900 of spec-experiments.md 19.9 (mean 10 650, std 354); fewer than two
// readings have no std, none have no mean.
//
// Requirement: FR-EXP-040, FR-EXP-042, FR-EXP-047
func TestMoments_MeanAndStd(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		in            []int64
		mean, std     int64
		meanOK, stdOK bool
		count         int64
	}{
		// sum 107 600 / 10 = 10 760; variance (10*1 158 140 000 - 107 600^2)/90 = 454 266.7 -> 454 267, root 674.
		{"reduce.out", reduceOut, 10760, 674, true, true, 10},
		// mean 10 650; variance ((10 400-10 650)^2 + (10 900-10 650)^2)/1 = 125 000, root 353.55 -> 354.
		{"19.9 pair", []int64{10400, 10900}, 10650, 354, true, true, 2},
		{"one reading has a mean and no std", []int64{5}, 5, 0, true, false, 1},
		{"none has neither", nil, 0, 0, false, false, 0},
		{"equal readings have std 0", []int64{3, 3, 3}, 3, 0, true, true, 3},
		{"negative readings", []int64{-4, -2}, -3, 1, true, true, 2}, // variance 2, root 1.41 -> 1
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := moments(t, tt.in...)
			if got := m.Count(); got != tt.count {
				t.Errorf("Count() = %d, want %d", got, tt.count)
			}
			if got, ok := m.Mean(); ok != tt.meanOK || got != tt.mean {
				t.Errorf("Mean() of %v = %d, %v, want %d, %v", tt.in, got, ok, tt.mean, tt.meanOK)
			}
			if got, ok := m.Std(); ok != tt.stdOK || got != tt.std {
				t.Errorf("Std() of %v = %d, %v, want %d, %v", tt.in, got, ok, tt.std, tt.stdOK)
			}
		})
	}
}

// TestMoments_ExactAtTheBound verifies that n*SS - S^2 is exact at the bound of
// spec-experiments.md 8.3.2: 10^6 readings of 2^40 and -2^40 in equal halves
// (variance 2^80 * 10^6/(10^6 - 1) rounded), against big.Int, and that a mean of
// extreme int64 readings does not overflow.
//
// Requirement: FR-EXP-047
func TestMoments_ExactAtTheBound(t *testing.T) {
	t.Parallel()
	const n = 1_000_000
	var m reduce.Moments
	for i := range n {
		v := int64(1) << 40
		if i%2 == 1 {
			v = -v
		}
		if !m.Add(v) {
			t.Fatalf("Add overflowed at reading %d", i)
		}
	}
	// S = 0, SS = n * 2^80, variance = rdiv(n * n * 2^80, n * (n - 1)).
	num := new(big.Int).Lsh(big.NewInt(int64(n)*n), 80)
	want := refRDiv(num, big.NewInt(int64(n)*(n-1)))
	got, ok := m.Variance()
	if !ok || bigOf(got).Cmp(want) != 0 {
		t.Errorf("Variance() = %v, %v, want %v", got, ok, want)
	}
	if s, ok := m.Sum(), true; !ok || s.Sign() != 0 {
		t.Errorf("Sum() = %v, want 0", s)
	}
	big2 := moments(t, math.MaxInt64, math.MaxInt64)
	if mean, ok := big2.Mean(); !ok || mean != math.MaxInt64 {
		t.Errorf("Mean() of two MaxInt64 = %d, %v, want MaxInt64", mean, ok)
	}
}

// TestMoments_AddReportsOverflow verifies that an Add that would overflow 128
// bits returns false and leaves the accumulator as it was.
//
// Requirement: FR-EXP-040
// Guard: a wrapped sum of squares gives a small, plausible, wrong variance.
func TestMoments_AddReportsOverflow(t *testing.T) {
	t.Parallel()
	var m reduce.Moments
	// Each square is 2^126 at most; the sum of squares overflows 2^127 on the second.
	if !m.Add(math.MinInt64) {
		t.Fatalf("first Add(MinInt64) = false, want true")
	}
	before := m
	if m.Add(math.MinInt64) {
		t.Fatalf("second Add(MinInt64) = true, want false: SS reaches 2^127")
	}
	if m != before {
		t.Errorf("accumulator after a refused Add = %+v, want %+v", m, before)
	}
}

// TestSignTail_ExactAgainstFractions verifies every n from 1 to 62 and every
// split of it against the exact fraction min(1, 2 * sum_{j>=m} C(n,j) / 2^n) in
// parts per million, rounded half to even by big.Rat arithmetic; n = 63 and
// n = 0 report false; the worked 6 wins and 1 loss give 125 000.
//
// Requirement: FR-EXP-044
func TestSignTail_ExactAgainstFractions(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 62; n++ { // the literal, not MaxSignN: a mutated cap must fail
		for wins := 0; wins <= n; wins++ {
			losses := n - wins
			m := max(wins, losses)
			sum := new(big.Int)
			for j := m; j <= n; j++ {
				sum.Add(sum, new(big.Int).Binomial(int64(n), int64(j)))
			}
			num := new(big.Int).Mul(sum, big.NewInt(2_000_000))
			want := refRDiv(num, new(big.Int).Lsh(big.NewInt(1), uint(n))).Int64()
			want = min(want, 1_000_000)
			if got, ok := reduce.SignTailPPM(wins, losses); !ok || got != want {
				t.Fatalf("SignTailPPM(%d, %d) = %d, %v, want %d", wins, losses, got, ok, want)
			}
		}
	}
	// 6 wins, 1 loss: n = 7, m = 6, sum = C(7,6) + C(7,7) = 8, 2*8/128 = 0.125.
	if got, ok := reduce.SignTailPPM(6, 1); !ok || got != 125_000 {
		t.Errorf("SignTailPPM(6, 1) = %d, %v, want 125000", got, ok)
	}
	for _, c := range [][2]int{{0, 0}, {32, 31}, {-1, 3}, {3, -1}} {
		if got, ok := reduce.SignTailPPM(c[0], c[1]); ok || got != 0 {
			t.Errorf("SignTailPPM(%d, %d) = %d, %v, want 0, false", c[0], c[1], got, ok)
		}
	}
}

func ramp(n int) []int64 {
	s := make([]int64, n)
	for i := range s {
		s[i] = int64(i)
	}
	return s
}

// TestQuantile_SortedInputUnchanged documents that Quantile reads and never sorts
// or changes its input: sorting is the caller's duty.
//
// Requirement: FR-EXP-042
func TestQuantile_SortedInputUnchanged(t *testing.T) {
	t.Parallel()
	in := slices.Clone(reduceOut)
	reduce.Quantile(in, 5000)
	if !slices.Equal(in, reduceOut) {
		t.Errorf("Quantile changed its input: %v, want %v", in, reduceOut)
	}
}

// TestMoments_StdBeyondInt64 verifies the one place a standard deviation leaves
// int64: two readings a and -a with a = 6 521 908 912 666 391 106 have variance
// 2a^2 just under 2^126, whose nearest root is exactly 2^63. Std reports false
// rather than wrapping; a = ...105 has the root 2^63 - 2 and fits.
//
// Requirement: FR-EXP-047
// Guard: a root cast to int64 without a check would wrap to the minimum.
func TestMoments_StdBeyondInt64(t *testing.T) {
	t.Parallel()
	over := moments(t, 6521908912666391106, -6521908912666391106)
	if got, ok := over.Std(); ok || got != 0 {
		t.Errorf("Std() of ±6521908912666391106 = %d, %v, want 0, false: the root is 2^63", got, ok)
	}
	fits := moments(t, 6521908912666391105, -6521908912666391105)
	if got, ok := fits.Std(); !ok || got != 9223372036854775806 {
		t.Errorf("Std() of ±6521908912666391105 = %d, %v, want 9223372036854775806", got, ok)
	}
}

// TestMoments_VarianceReportsOverflow verifies that Variance returns false when
// n*SS or S^2 leaves 128 bits though every Add succeeded: 20 readings of ±2^60 in
// equal halves have SS = 20*2^120 (fits) and n*SS = 400*2^120 (does not), with
// S = 0; 20 readings of +2^60 overflow both products.
//
// Requirement: FR-EXP-047
// Guard: a wrapped n*SS gives a small, plausible, wrong variance.
func TestMoments_VarianceReportsOverflow(t *testing.T) {
	t.Parallel()
	var alt, same reduce.Moments
	for i := range 20 {
		v := int64(1) << 60
		if !same.Add(v) {
			t.Fatalf("Add(2^60) refused at reading %d", i)
		}
		if i%2 == 1 {
			v = -v
		}
		if !alt.Add(v) {
			t.Fatalf("Add(±2^60) refused at reading %d", i)
		}
	}
	for name, m := range map[string]reduce.Moments{"alternating": alt, "all positive": same} {
		if got, ok := m.Variance(); ok {
			t.Errorf("%s: Variance() = %v, true, want false (128-bit overflow)", name, got)
		}
		if got, ok := m.Std(); ok {
			t.Errorf("%s: Std() = %d, true, want false", name, got)
		}
	}
}

// TestMoments_MeanOfNothing verifies the empty accumulator: no mean and a zero
// value alongside the false.
//
// Requirement: FR-EXP-040
func TestMoments_MeanOfNothing(t *testing.T) {
	t.Parallel()
	var m reduce.Moments
	if got, ok := m.Mean(); ok || got != 0 {
		t.Errorf("Mean() of nothing = %d, %v, want 0, false", got, ok)
	}
	if got, ok := m.Variance(); ok || got.Sign() != 0 {
		t.Errorf("Variance() of nothing = %v, %v, want 0, false", got, ok)
	}
}
