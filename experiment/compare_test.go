package experiment_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/experiment"
)

var (
	// worked197A and worked197B are the eight seed-matched replicas of
	// spec-experiments.md 19.7.
	worked197A = []int64{100, 104, 98, 110, 101, 99, 107, 103}
	worked197B = []int64{103, 109, 97, 118, 107, 99, 113, 104}
)

func rd(v int64) experiment.Reading { return experiment.Reading{Value: v, OK: true} }

// twoPoints assembles two points of a SameSeeds grid over a two-value knob.
func twoPoints(t testing.TB, a, b []int64) experiment.Result {
	t.Helper()
	return column(t, toySpec(len(a), experiment.Grid, knob(1, 2, 2)), [][]int64{a, b})
}

// TestCompare_Worked197 verifies every field of the worked comparison of
// spec-experiments.md 19.7.
//
// The differences d = B - A are (3, 5, -1, 8, 6, 0, 6, 1): sum 28, so MeanDiff is
// rdiv(28, 8) = 4 (3.5 rounds to the even 4); sorted they are -1, 0, 1, 3, 5, 6, 6,
// 8, whose rank 4 is 3. Six wins, one loss, one tie. The sign tail is
// 2 * (C(7,6) + C(7,7)) / 2^7 = 16/128 = 125 000 ppm. The sums of A and B are 822
// and 850: RelativeBP = rdiv(10 000 * 28, 822) = 341 (the rounded means would give
// 388, a second rounding), DiffOfMeans = rdiv(850*8 - 822*8, 64) = 4, the means
// 102.75 and 106.25 round to 103 and 106.
//
// The gain (7.4.2): the sum of squares of d is 172, so its variance is
// rdiv(8*172 - 28^2, 56) = rdiv(592, 56) = 11 and sd = isqrt_nearest(11) = 3; of A
// 84 580 and 822 give rdiv(956, 56) = 17, of B 90 658 and 850 give rdiv(2 764, 56) =
// 49; sd_indep = isqrt_nearest(66) = 8; gain = rdiv(80 000, 3) = 26 667.
//
// Requirement: FR-EXP-044, FR-EXP-045
func TestCompare_Worked197(t *testing.T) {
	t.Parallel()
	res := twoPoints(t, worked197A, worked197B)
	got, err := experiment.Compare(res, experiment.Group{Point: 0}, experiment.Group{Point: 1}, toyMeasure)
	if err != nil {
		t.Fatalf("Compare() error = %v, want nil", err)
	}
	want := experiment.Comparison{
		Pairs: 8, MeanA: rd(103), MeanB: rd(106), DiffOfMeans: rd(4), MeanDiff: rd(4), MedianDiff: rd(3),
		Wins: 6, Losses: 1, Ties: 1, SignTailPPM: rd(125000), RelativeBP: rd(341),
		PairingGain: rd(26667), SdIndep: rd(8), SdDiff: rd(3),
	}
	// Quantiles of A sorted (98, 99, 100, 101, 103, 104, 107, 110) and B sorted (97,
	// 99, 103, 104, 107, 109, 113, 118) at ranks 1, 2, 4, 6 and 8.
	for i, ab := range [][2]int64{{98, 97}, {99, 99}, {101, 104}, {104, 109}, {110, 118}} {
		want.Dist[i] = experiment.QuantilePair{A: rd(ab[0]), B: rd(ab[1]), Diff: rd(ab[1] - ab[0])}
	}
	if got != want {
		t.Errorf("Compare(19.7) =\n %+v\nwant\n %+v", got, want)
	}
}

// TestCompare_PerPointSeedsIsUnpaired verifies that two points whose replicas have
// other seeds (a PerPointSeeds experiment) are unpaired: no pairs, no wins, no
// sign test, and the growth from the exact sums of the groups.
//
// With the sums of 19.7 the growth is rdiv(10 000 * (850*8 - 822*8), 822*8) =
// rdiv(2 240 000, 6 576) = 340.6, so 341, the same as the paired one here.
//
// Requirement: FR-EXP-044
func TestCompare_PerPointSeedsIsUnpaired(t *testing.T) {
	t.Parallel()
	e := toySpec(8, experiment.Grid, knob(1, 2, 2))
	e.Seeding = experiment.PerPointSeeds
	var recs []experiment.Record
	for p, vals := range [][]int64{worked197A, worked197B} {
		for r, v := range vals {
			recs = append(recs, okRec(p, r, e.Seed0+uint64(p*8+r), v))
		}
	}
	res := assemble(t, e, recs...)
	got, err := experiment.Compare(res, experiment.Group{Point: 0}, experiment.Group{Point: 1}, toyMeasure)
	if err != nil {
		t.Fatalf("Compare() error = %v, want nil", err)
	}
	if got.Pairs != 0 || got.Wins+got.Losses+got.Ties != 0 || got.SignTailPPM.OK || got.MeanDiff.OK || got.PairingGain.OK || got.Identical {
		t.Errorf("Compare(unpaired) = %+v, want no pairs, wins, sign test, mean difference or gain", got)
	}
	if got.MeanA != rd(103) || got.MeanB != rd(106) || got.DiffOfMeans != rd(4) || got.RelativeBP != rd(341) {
		t.Errorf("Compare(unpaired) means, difference, growth = %+v %+v %+v %+v, want 103, 106, 4, 341", got.MeanA, got.MeanB, got.DiffOfMeans, got.RelativeBP)
	}
	if !got.Dist[2].Diff.OK {
		t.Errorf("Compare(unpaired) Dist = %+v, want the distributions offered for unmatched groups", got.Dist)
	}
}

// TestCompare_SignTailBeyond62 verifies that the exact sign test is offered up to
// 62 untied pairs and is undefined beyond, and that 62 and 63 pairs of wins give the
// stated tails.
//
// 62 wins and no loss: 2 * C(62,62) / 2^62 is below half a part per million, so it
// rounds to 0; 63 pairs are beyond the cap of choice.
//
// Requirement: FR-EXP-044
func TestCompare_SignTailBeyond62(t *testing.T) {
	t.Parallel()
	for _, n := range []int{62, 63} {
		a, b := make([]int64, n), make([]int64, n)
		for i := range n {
			a[i], b[i] = int64(100+i), int64(101+i)
		}
		got, err := experiment.Compare(twoPoints(t, a, b), experiment.Group{Point: 0}, experiment.Group{Point: 1}, toyMeasure)
		if err != nil {
			t.Fatalf("Compare(%d pairs) error = %v, want nil", n, err)
		}
		if got.Pairs != n || got.Wins != n {
			t.Errorf("Compare(%d pairs) Pairs, Wins = %d, %d, want %d, %d", n, got.Pairs, got.Wins, n, n)
		}
		if want := n == 62; got.SignTailPPM.OK != want || (want && got.SignTailPPM.Value != 0) {
			t.Errorf("Compare(%d pairs) SignTailPPM = %+v, want defined = %v (0 ppm at 62)", n, got.SignTailPPM, want)
		}
	}
}

// TestCompare_PairingGain verifies the pairing gain: two identical settings are
// Identical with the gain undefined (sd 0), and a known spread gives the stated
// gain.
//
// A = (10, 20, 30, 40) and B = (11, 23, 31, 43): d = (1, 3, 1, 3) has sum 8 and sum
// of squares 20, so its variance is rdiv(4*20 - 64, 12) = rdiv(16, 12) = 1 and sd = 1.
//
// Requirement: FR-EXP-045
func TestCompare_PairingGain(t *testing.T) {
	t.Parallel()
	same, err := experiment.Compare(twoPoints(t, worked197A, worked197A), experiment.Group{Point: 0}, experiment.Group{Point: 1}, toyMeasure)
	if err != nil {
		t.Fatalf("Compare(identical) error = %v, want nil", err)
	}
	if !same.Identical || same.PairingGain.OK || same.SdDiff != rd(0) || same.Ties != 8 || same.SignTailPPM.OK {
		t.Errorf("Compare(identical) = %+v, want Identical, sd 0, no gain, 8 ties and no sign test", same)
	}
	a := []int64{10, 20, 30, 40}
	b := []int64{11, 23, 31, 43}
	got, err := experiment.Compare(twoPoints(t, a, b), experiment.Group{Point: 0}, experiment.Group{Point: 1}, toyMeasure)
	if err != nil {
		t.Fatalf("Compare() error = %v, want nil", err)
	}
	// B sums to 108 with squares 121+529+961+1849 = 3 460: vB = rdiv(4*3460 - 108^2,
	// 12) = rdiv(2 176, 12) = 181 (181.3). A sums to 100 with squares 3 000: vA =
	// rdiv(2 000, 12) = 167 (166.7). sd_indep = isqrt_nearest(348) = 19 (18.65) and the
	// gain is rdiv(10 000 * 19, 1) = 190 000.
	if got.SdDiff != rd(1) || got.SdIndep != rd(19) || got.PairingGain != rd(190000) || got.Identical {
		t.Errorf("Compare(known spread) SdDiff, SdIndep, PairingGain = %+v, %+v, %+v, want 1, 19, 190000", got.SdDiff, got.SdIndep, got.PairingGain)
	}
}

// TestCompare_Errors verifies the two errors of Compare: a measure the result does
// not have and a group outside it.
//
// Requirement: FR-EXP-044
func TestCompare_Errors(t *testing.T) {
	t.Parallel()
	res := twoPoints(t, worked197A, worked197B)
	tests := []struct {
		name    string
		a, b    experiment.Group
		measure string
		want    error
	}{
		{"unknown measure", experiment.Group{Point: 0}, experiment.Group{Point: 1}, "nope", experiment.ErrUnknownMeasure},
		{"point past the result", experiment.Group{Point: 0}, experiment.Group{Point: 2}, toyMeasure, experiment.ErrUnknownRun},
		{"negative point", experiment.Group{Point: -1}, experiment.Group{Point: 1}, toyMeasure, experiment.ErrUnknownRun},
		{"variant past the point", experiment.Group{Point: 0, Variant: 1}, experiment.Group{Point: 1}, toyMeasure, experiment.ErrUnknownRun},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := experiment.Compare(res, tt.a, tt.b, tt.measure); !errors.Is(err, tt.want) {
				t.Errorf("Compare(%+v, %+v, %q) error = %v, want %v", tt.a, tt.b, tt.measure, err, tt.want)
			}
		})
	}
}

// TestCompare_BaseAtOrBelowZero verifies that the growth is undefined for a base
// at or below 0 and clamped, with Clamped set, beyond 2^31 - 1 basis points.
//
// Requirement: FR-EXP-044
func TestCompare_BaseAtOrBelowZero(t *testing.T) {
	t.Parallel()
	zero, err := experiment.Compare(twoPoints(t, []int64{0, 0}, []int64{5, 5}), experiment.Group{Point: 0}, experiment.Group{Point: 1}, toyMeasure)
	if err != nil {
		t.Fatalf("Compare() error = %v, want nil", err)
	}
	if zero.RelativeBP.OK || zero.Clamped || zero.DiffOfMeans != rd(5) {
		t.Errorf("Compare(base 0) RelativeBP, Clamped, DiffOfMeans = %+v, %v, %+v, want undefined, false, 5", zero.RelativeBP, zero.Clamped, zero.DiffOfMeans)
	}
	// 10 000 * (1 000 000 - 1) / 1 = 9.99e9 basis points, beyond the clamp.
	big, err := experiment.Compare(twoPoints(t, []int64{1}, []int64{1_000_000}), experiment.Group{Point: 0}, experiment.Group{Point: 1}, toyMeasure)
	if err != nil {
		t.Fatalf("Compare() error = %v, want nil", err)
	}
	if big.RelativeBP != rd(1<<31-1) || !big.Clamped {
		t.Errorf("Compare(huge growth) RelativeBP, Clamped = %+v, %v, want %d, true", big.RelativeBP, big.Clamped, int64(1<<31-1))
	}
	// And the same fall: 10 000 * (-999 999) / 1_000_000 is -9 999.99, not clamped.
	fall, err := experiment.Compare(twoPoints(t, []int64{1_000_000}, []int64{1}), experiment.Group{Point: 0}, experiment.Group{Point: 1}, toyMeasure)
	if err != nil {
		t.Fatalf("Compare() error = %v, want nil", err)
	}
	if fall.RelativeBP != rd(-10000) || fall.Clamped {
		t.Errorf("Compare(fall) RelativeBP = %+v clamped %v, want -10000 (-9999.99 rounds)", fall.RelativeBP, fall.Clamped)
	}
}
