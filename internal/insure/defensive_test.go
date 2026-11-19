package insure_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/insure"
)

// TestMaxPayout_OverflowRefused verifies that a share too large for int64 is
// refused and not wrapped, and that the matching skips the offer instead of
// forming a fill with a garbage maximum payout.
//
// Requirement: FR-INS-023, FR-INS-034, NFR-INS-005
func TestMaxPayout_OverflowRefused(t *testing.T) {
	t.Parallel()
	if m, ok := insure.MaxPayout(math.MaxInt64, math.MaxInt64); ok {
		t.Errorf("MaxPayout(max, max) = %d, true, want a refusal", m)
	}
	var calls []fill
	cands := []insure.Candidate{{Ref: 1, Party: 1, Limit: 1, Sum: math.MaxInt64}}
	offers := []insure.Offer{{Ref: 0, Party: 2, CoverBP: math.MaxInt64, Room: math.MaxInt64}}
	if insure.Match(cands, offers, false, scriptedForm(nil, &calls)) || len(calls) != 0 {
		t.Errorf("Match with an overflowing maximum payout called %v, want nothing", calls)
	}
	if offers[0].Room != math.MaxInt64 {
		t.Errorf("offer room = %d, want it untouched", offers[0].Room)
	}
}

// TestMatch_NegativeSumSkipped verifies that a candidate with a negative sum
// forms nothing and an offer with negative room is never chosen.
//
// Requirement: FR-INS-034, NFR-INS-005
func TestMatch_NegativeSumSkipped(t *testing.T) {
	t.Parallel()
	var calls []fill
	form := scriptedForm(nil, &calls)
	offers := []insure.Offer{{Ref: 0, Party: 2, CoverBP: 5000, Room: -1}}
	insure.Match([]insure.Candidate{{Ref: 1, Party: 1, Limit: 10, Sum: -100}}, offers, false, form)
	insure.Match([]insure.Candidate{{Ref: 2, Party: 1, Limit: 10, Sum: 100}}, offers, false, form)
	if len(calls) != 0 {
		t.Errorf("Match called %v, want nothing", calls)
	}
}

// TestSumAccount_VehicleOutOfRange verifies that a vehicle index at or past the
// end excludes nothing and does not panic.
//
// Requirement: FR-INS-021, NFR-INS-005
func TestSumAccount_VehicleOutOfRange(t *testing.T) {
	t.Parallel()
	for _, vehicle := range []int{2, 99, math.MaxInt} {
		if got, ok := insure.SumAccount([]int64{3, 4}, []int64{10, 100}, vehicle, maxMoney); got != 430 || !ok {
			t.Errorf("SumAccount(vehicle %d) = %d, %v, want 430, true", vehicle, got, ok)
		}
	}
}

// TestZeroLimits verifies the money functions at a limit of 0 and with empty
// input: nothing positive passes, and empty input is a 0 that passes.
//
// Requirement: FR-INS-021, FR-INS-022, FR-INS-041, NFR-INS-005
func TestZeroLimits(t *testing.T) {
	t.Parallel()
	one := []int64{1}
	// Each row is one call reduced to "got, ok"; the loss value reports its
	// saturation as ok's opposite.
	tests := []struct {
		name   string
		got    func() (int64, bool)
		want   int64
		wantOK bool
	}{
		{"SumLines above 0", func() (int64, bool) { return insure.SumLines(one, one, 0) }, 0, false},
		{"SumLines nil", func() (int64, bool) { return insure.SumLines(nil, nil, 0) }, 0, true},
		{"SumStore nil", func() (int64, bool) { return insure.SumStore(nil, nil, 5000, 0) }, 0, true},
		{"Premium at limit 0", func() (int64, bool) { p, _, ok := insure.Premium(1, 10_000, 10_000, 10_000, 0); return p, ok }, 0, false},
		{"StorePremium at limit -1", func() (int64, bool) { return insure.StorePremium(1, 10_000, -1) }, 0, false},
		{"StorePremium zero", func() (int64, bool) { return insure.StorePremium(0, 0, 0) }, 0, true},
		{"LossValue nil", func() (int64, bool) { v, sat := insure.LossValue(nil, nil, 0); return v, !sat }, 0, true},
		{"LossValue past 0", func() (int64, bool) { v, sat := insure.LossValue(one, one, 0); return v, !sat }, 0, false},
		{"Due negative cover", func() (int64, bool) { d, c := insure.Due(100, -1, 100); return d, !c }, 0, true},
		{"Due negative maximum", func() (int64, bool) { d, c := insure.Due(-5, 8000, 100); return d, !c }, 0, true},
		{"UnitValue reference 0", func() (int64, bool) { return insure.UnitValue(5, 0, 12_000) }, 0, true},
	}
	for _, tt := range tests {
		if got, ok := tt.got(); got != tt.want || ok != tt.wantOK {
			t.Errorf("%s = %d, %v, want %d, %v", tt.name, got, ok, tt.want, tt.wantOK)
		}
	}
}

// TestSort_OrderIndependentOfInput verifies that the three sorts give one order
// whatever the order of the input (architecture.md 7.6), and that an empty or a
// single-element slice is left alone.
//
// Requirement: FR-INS-033, NFR-INS-002
func TestSort_OrderIndependentOfInput(t *testing.T) {
	t.Parallel()
	insure.SortRequests(nil)
	insure.SortLoadings([]insure.Offer{})
	one := []insure.Offer{{Ref: 7}}
	insure.SortStoreRates(one)
	if one[0].Ref != 7 {
		t.Fatalf("single offer changed: %+v", one)
	}
	reqs := []insure.Request{
		{Ref: 0, Party: 1, Limit: 5, Key: 1}, {Ref: 1, Party: 1, Limit: 5, Key: 1}, {Ref: 2, Party: 1, Limit: 5, Key: 1},
		{Ref: 3, Party: 2, Limit: 9, Key: 4}, {Ref: 4, Party: 1, Limit: 5, Key: 1, Seq: 1},
	}
	offs := []insure.Offer{
		{Ref: 0, Party: 1, Loading: 5, StoreRate: 5, Key: 1}, {Ref: 1, Party: 1, Loading: 5, StoreRate: 5, Key: 1},
		{Ref: 2, Party: 2, Loading: 3, StoreRate: 8, Key: 9}, {Ref: 3, Party: 1, Loading: 5, StoreRate: 5, Key: 1},
	}
	refs := func(sorted func(r []insure.Request, o []insure.Offer)) (a, b, c []int) {
		r, o := slices.Clone(reqs), slices.Clone(offs)
		sorted(r, o)
		for _, x := range r {
			a = append(a, x.Ref)
		}
		for _, x := range o {
			b = append(b, x.Ref)
		}
		o2 := slices.Clone(o)
		slices.Reverse(o2)
		insure.SortStoreRates(o2)
		for _, x := range o2 {
			c = append(c, x.Ref)
		}
		return a, b, c
	}
	wa, wb, wc := refs(func(r []insure.Request, o []insure.Offer) { insure.SortRequests(r); insure.SortLoadings(o) })
	for range 5 {
		reqs, offs = rotate(reqs), rotate(offs)
		a, b, c := refs(func(r []insure.Request, o []insure.Offer) { insure.SortRequests(r); insure.SortLoadings(o) })
		if !slices.Equal(a, wa) || !slices.Equal(b, wb) || !slices.Equal(c, wc) {
			t.Fatalf("order depends on the input: %v %v %v, want %v %v %v", a, b, c, wa, wb, wc)
		}
	}
}

// rotate returns s moved one place left, a different input order of the same rows.
func rotate[T any](s []T) []T {
	return append(slices.Clone(s[1:]), s[0])
}

// TestPremium_FactorOverflowAndLimitBoundary verifies the two edges the worked
// examples never reach: a product of the three basis-point factors that does not
// fit int64 is refused at either multiplication, and a premium exactly equal to
// the limit passes while one above it does not.
//
// Guard: the factors are multiplied before the 128-bit division (spec-insurance.md
// 6.4.1). Without the checks a wrapped k would give a small, wrong premium instead
// of the overflow outcome, and a `>` turned into `>=` would refuse a premium that
// fits.
//
// Requirement: FR-INS-022, FR-INS-024, NFR-INS-005
func TestPremium_FactorOverflowAndLimitBoundary(t *testing.T) {
	t.Parallel()
	// 2^40 * 2^40 = 2^80 overflows at the first product; 10^9 * 10^9 = 10^18 fits
	// and * 10^9 = 10^27 overflows at the second. The example of 19.1 gives 154
	// at rate 1 357, so a limit of 154 is the exact boundary.
	tests := []struct {
		name                         string
		sum, lossBP, coverBP, loadBP int64
		limit                        int64
		wantP, wantRate              int64
		wantOK                       bool
	}{
		{"first product overflows", 1, 1 << 40, 1 << 40, 1, maxMoney, 0, 0, false},
		{"second product overflows", 1, 1e9, 1e9, 1e9, maxMoney, 0, 0, false},
		{"premium equals the limit", 1134, 1305, 8000, 13000, 154, 154, 1357, true},
		{"premium one above the limit", 1134, 1305, 8000, 13000, 153, 0, 0, false},
		{"zero sum", 0, 1305, 8000, 13000, 0, 0, 1357, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, rate, ok := insure.Premium(tt.sum, tt.lossBP, tt.coverBP, tt.loadBP, tt.limit)
			if p != tt.wantP || rate != tt.wantRate || ok != tt.wantOK {
				t.Errorf("Premium(%d, %d, %d, %d, %d) = %d, %d, %v, want %d, %d, %v",
					tt.sum, tt.lossBP, tt.coverBP, tt.loadBP, tt.limit, p, rate, ok, tt.wantP, tt.wantRate, tt.wantOK)
			}
		})
	}
	if p, ok := insure.StorePremium(1000, 1000, 100); p != 100 || !ok {
		t.Errorf("StorePremium(1000, 1000) with limit 100 = %d, %v, want 100, true (1000 * 1000 / 10 000)", p, ok)
	}
}
