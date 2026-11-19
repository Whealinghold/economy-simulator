package insure_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/insure"
)

// maxMoney stands for the root's MaxMoney (1e18); the package takes it as a limit.
const maxMoney int64 = 1_000_000_000_000_000_000

// TestUnitValue_CapAndPublished verifies that a unit is valued at the published
// price, capped at ValueCapBP of the reference.
//
// The numbers are spec-insurance.md 19.1 step 2 with a cap of 12 000: grain
// min(12, 12 000 * 10 / 10 000 = 12) = 12, cloth min(90, 96) = 90, rope 30, tar
// min(25, 24) = 24; at settlement grain's published 14 is cut to the cap 12.
//
// Requirement: FR-INS-020
func TestUnitValue_CapAndPublished(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		published, ref, capBP int64
		want                  int64
		wantOK                bool
	}{
		{"grain at the cap", 12, 10, 12_000, 12, true},
		{"cloth below the cap", 90, 80, 12_000, 90, true},
		{"rope", 30, 25, 12_000, 30, true},
		{"tar cut to the cap", 25, 20, 12_000, 24, true},
		{"grain after a shortage", 14, 10, 12_000, 12, true},
		{"cap truncates", 100, 7, 10_001, 7, true},
		{"zero reference", 5, 0, 12_000, 0, true},
		{"negative published", -1, 10, 12_000, 0, false},
		{"negative reference", 1, -10, 12_000, 0, false},
		{"negative cap", 1, 10, -1, 0, false},
		{"quotient overflows", 5, 1<<63 - 1, 1<<63 - 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := insure.UnitValue(tt.published, tt.ref, tt.capBP)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("UnitValue(%d, %d, %d) = %d, %v, want %d, %v", tt.published, tt.ref, tt.capBP, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestSum_ByShape verifies the lines term, the account term with the vehicle left
// out, and the overflow outcomes.
//
// 19.1: 40 * 12 + 5 * 90 + 2 * 30 + 6 * 24 = 480 + 450 + 60 + 144 = 1 134. The
// account term skips index 1 (the carrying good): 3 * 10 + 4 * 99 + 5 * 2 = 40 with
// the 99 skipped is 3 * 10 + 5 * 2 = 40.
//
// Requirement: FR-INS-021, FR-INS-024
func TestSum_ByShape(t *testing.T) {
	t.Parallel()
	value := []int64{12, 90, 30, 24}
	t.Run("lines term", func(t *testing.T) {
		t.Parallel()
		got, ok := insure.SumLines([]int64{40, 5, 2, 6}, value, maxMoney)
		if got != 1134 || !ok {
			t.Errorf("SumLines = %d, %v, want 1134, true", got, ok)
		}
	})
	t.Run("account term skips the vehicle", func(t *testing.T) {
		t.Parallel()
		got, ok := insure.SumAccount([]int64{3, 4, 5}, []int64{10, 99, 2}, 1, maxMoney)
		if got != 40 || !ok {
			t.Errorf("SumAccount(vehicle 1) = %d, %v, want 40, true", got, ok)
		}
	})
	t.Run("a negative vehicle skips none", func(t *testing.T) {
		t.Parallel()
		got, ok := insure.SumAccount([]int64{3, 4, 5}, []int64{10, 99, 2}, -1, maxMoney)
		if got != 436 || !ok { // 30 + 396 + 10
			t.Errorf("SumAccount(no vehicle) = %d, %v, want 436, true", got, ok)
		}
	})
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		got, ok := insure.SumLines(nil, nil, maxMoney)
		if got != 0 || !ok {
			t.Errorf("SumLines(nil) = %d, %v, want 0, true", got, ok)
		}
	})
}

// TestSum_Refusals verifies that a sum above the limit, a product that overflows,
// a negative entry and slices of different lengths are refused.
//
// Requirement: FR-INS-024, NFR-INS-005
func TestSum_Refusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		qty, value []int64
		limit      int64
	}{
		{"above the limit", []int64{2}, []int64{6}, 11},
		{"sum of two passes the limit", []int64{1, 1}, []int64{6, 6}, 11},
		{"product overflows", []int64{1 << 62}, []int64{4}, maxMoney},
		{"partial sum overflows", []int64{1, 1}, []int64{1 << 62, 1 << 62}, 1<<63 - 1},
		{"negative quantity", []int64{-1}, []int64{5}, maxMoney},
		{"negative value", []int64{1}, []int64{-5}, maxMoney},
		{"different lengths", []int64{1, 2}, []int64{5}, maxMoney},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := insure.SumLines(tt.qty, tt.value, tt.limit); ok {
				t.Errorf("SumLines(%v, %v, %d) = %d, true, want a refusal", tt.qty, tt.value, tt.limit, got)
			}
		})
	}
}

// TestSum_StoreHeldTimesShare verifies the store sum, trunc(Held * ShareBP / 10 000).
//
// 19.2: Held = 250 * 11 + 50 * 70 = 6 250; at 5 000 bp the sum is 3 125; at 3 333 bp
// it is trunc(6 250 * 3 333 / 10 000) = trunc(2 083.125) = 2 083.
//
// Requirement: FR-INS-021
func TestSum_StoreHeldTimesShare(t *testing.T) {
	t.Parallel()
	held, value := []int64{250, 50}, []int64{11, 70}
	tests := []struct {
		name    string
		share   int64
		want    int64
		wantOK  bool
		heldCap int64
	}{
		{"half", 5_000, 3_125, true, maxMoney},
		{"truncates", 3_333, 2_083, true, maxMoney},
		{"all", 10_000, 6_250, true, maxMoney},
		{"negative share", -1, 0, false, maxMoney},
		{"held above the limit", 5_000, 0, false, 6_249},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := insure.SumStore(held, value, tt.share, tt.heldCap)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("SumStore(share %d, limit %d) = %d, %v, want %d, %v", tt.share, tt.heldCap, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestPremium_Example19_1 verifies the premium, the rate and the maximum payout of
// the worked route policy.
//
// Sum 1 134, lane loss 1 305, cover 8 000, loading 13 000: rate
// halfEven(1 305 * 8 000 * 13 000 / 10^8) = halfEven(1 357.2) = 1 357; premium
// halfEven(1 134 * 1 305 * 8 000 * 13 000 / 10^12) = halfEven(153.9) = 154; maximum
// halfEven(8 000 * 1 134 / 10 000) = halfEven(907.2) = 907.
//
// Requirement: FR-INS-022, FR-INS-023
func TestPremium_Example19_1(t *testing.T) {
	t.Parallel()
	p, rate, ok := insure.Premium(1134, 1305, 8000, 13000, maxMoney)
	if p != 154 || rate != 1357 || !ok {
		t.Errorf("Premium(1134, 1305, 8000, 13000) = %d, %d, %v, want 154, 1357, true", p, rate, ok)
	}
	if m, ok := insure.MaxPayout(8000, 1134); m != 907 || !ok {
		t.Errorf("MaxPayout(8000, 1134) = %d, %v, want 907, true", m, ok)
	}
}

// TestPremium_StoreExample19_2 verifies the store premium and maximum of the second
// worked example: sum 3 125 at 150 bp is halfEven(46.875) = 47, and the maximum
// halfEven(8 000 * 3 125 / 10 000) = 2 500.
//
// Requirement: FR-INS-022, FR-INS-023
func TestPremium_StoreExample19_2(t *testing.T) {
	t.Parallel()
	if p, ok := insure.StorePremium(3125, 150, maxMoney); p != 47 || !ok {
		t.Errorf("StorePremium(3125, 150) = %d, %v, want 47, true", p, ok)
	}
	if m, ok := insure.MaxPayout(8000, 3125); m != 2500 || !ok {
		t.Errorf("MaxPayout(8000, 3125) = %d, %v, want 2500, true", m, ok)
	}
}

// TestPremium_HalfToEven verifies the rounding rule on exact ties: the quotient goes
// to the even integer, so 0.5 is 0, 1.5 is 2 and 2.5 is 2.
//
// Requirement: FR-INS-022
func TestPremium_HalfToEven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		sum  int64
		want int64
	}{{5, 0}, {15, 2}, {25, 2}, {35, 4}}
	for _, tt := range tests {
		got, ok := insure.StorePremium(tt.sum, 1000, maxMoney) // sum * 1 000 / 10 000
		if got != tt.want || !ok {
			t.Errorf("StorePremium(%d, 1000) = %d, %v, want %d, true", tt.sum, got, ok, tt.want)
		}
	}
	// A route tie: loss 1, cover 10 000, loading 10 000 give 10^8 over 10^12, so the
	// premium is sum / 10 000: 5 000 is 0.5 and goes to 0, 15 000 is 1.5 and goes to 2,
	// 25 000 is 2.5 and goes to 2.
	for _, c := range []struct{ sum, want int64 }{{5_000, 0}, {15_000, 2}, {25_000, 2}} {
		p, _, ok := insure.Premium(c.sum, 1, 10_000, 10_000, maxMoney)
		if p != c.want || !ok {
			t.Errorf("Premium(%d, 1, 10000, 10000) = %d, %v, want %d, true", c.sum, p, ok, c.want)
		}
	}
}

// TestRhe_LargestCaseNoOverflow verifies that the largest product of the bounds
// (MaxMoney * 10^4 * 10^4 * 10^5, about 10^31) is formed in 128 bits and its quotient
// checked, not wrapped.
//
// MaxMoney 10^18 * 10^4 * 10^4 * 10^5 / 10^12 = 10^19 is above int64 and the limit,
// so it is refused; with loading 10^4 the quotient is 10^18 and fits.
//
// Requirement: FR-INS-022, NFR-INS-005
func TestRhe_LargestCaseNoOverflow(t *testing.T) {
	t.Parallel()
	if p, _, ok := insure.Premium(maxMoney, 10_000, 10_000, 100_000, maxMoney); ok {
		t.Errorf("Premium(MaxMoney, 1e4, 1e4, 1e5) = %d, true, want a refusal (quotient 1e19)", p)
	}
	p, rate, ok := insure.Premium(maxMoney, 10_000, 10_000, 10_000, maxMoney)
	if p != maxMoney || rate != 10_000 || !ok {
		t.Errorf("Premium(MaxMoney, 1e4, 1e4, 1e4) = %d, %d, %v, want 1e18, 10000, true", p, rate, ok)
	}
}

// TestPremium_Refusals verifies the domain checks: a negative input and a premium
// above the limit are refused, never wrapped.
//
// Requirement: FR-INS-024, NFR-INS-005
func TestPremium_Refusals(t *testing.T) {
	t.Parallel()
	for _, in := range [][4]int64{{-1, 1, 1, 1}, {1, -1, 1, 1}, {1, 1, -1, 1}, {1, 1, 1, -1}} {
		if p, r, ok := insure.Premium(in[0], in[1], in[2], in[3], maxMoney); ok {
			t.Errorf("Premium(%v) = %d, %d, true, want a refusal", in, p, r)
		}
	}
	if p, _, ok := insure.Premium(1134, 1305, 8000, 13000, 153); ok {
		t.Errorf("Premium with limit 153 = %d, true, want a refusal (premium 154)", p)
	}
	if p, ok := insure.StorePremium(-1, 1, maxMoney); ok {
		t.Errorf("StorePremium(-1, 1) = %d, true, want a refusal", p)
	}
	if p, ok := insure.StorePremium(1000, 1000, 99); ok {
		t.Errorf("StorePremium(1000, 1000) with limit 99 = %d, true, want a refusal", p)
	}
	if m, ok := insure.MaxPayout(-1, 5); ok {
		t.Errorf("MaxPayout(-1, 5) = %d, true, want a refusal", m)
	}
}

// TestLossValue_Example19_1 verifies the loss value and the due of the worked route
// policy: 9 * 12 + 1 * 90 + 0 * 30 + 2 * 24 = 246, and
// min(907, halfEven(8 000 * 246 / 10 000) = halfEven(196.8) = 197) = 197.
//
// Requirement: FR-INS-041, FR-INS-042
func TestLossValue_Example19_1(t *testing.T) {
	t.Parallel()
	v, sat := insure.LossValue([]int64{9, 1, 0, 2}, []int64{12, 90, 30, 24}, maxMoney)
	if v != 246 || sat {
		t.Errorf("LossValue = %d, %v, want 246, false", v, sat)
	}
	if d, capped := insure.Due(907, 8000, 246); d != 197 || capped {
		t.Errorf("Due(907, 8000, 246) = %d, %v, want 197, false", d, capped)
	}
}

// TestDue_StoreCapped verifies the cut at the maximum payout of the second example:
// loss value 133 * 11 + 27 * 70 = 3 353, share halfEven(8 000 * 3 353 / 10 000) =
// 2 682, due min(2 500, 2 682) = 2 500, capped. A share equal to the maximum is not
// a cut.
//
// Requirement: FR-INS-042
func TestDue_StoreCapped(t *testing.T) {
	t.Parallel()
	v, _ := insure.LossValue([]int64{133, 27}, []int64{11, 70}, maxMoney)
	if v != 3353 {
		t.Fatalf("LossValue = %d, want 3353", v)
	}
	tests := []struct {
		name             string
		maxPayout, cover int64
		loss             int64
		want             int64
		wantCapped       bool
	}{
		{"cut at the maximum", 2500, 8000, 3353, 2500, true},
		{"equal to the maximum", 2682, 8000, 3353, 2682, false},
		{"no loss", 2500, 8000, 0, 0, false},
		{"rounds to nothing", 2500, 1, 1, 0, false},
		{"share overflows int64", 7, 10_000, 1<<63 - 1, 7, true},
		{"no maximum", 0, 8000, 100, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, capped := insure.Due(tt.maxPayout, tt.cover, tt.loss)
			if got != tt.want || capped != tt.wantCapped {
				t.Errorf("Due(%d, %d, %d) = %d, %v, want %d, %v", tt.maxPayout, tt.cover, tt.loss, got, capped, tt.want, tt.wantCapped)
			}
		})
	}
}

// TestLossValue_Saturates verifies that the loss value saturates at the limit
// instead of wrapping, on a product, on a sum, and past the limit alone.
//
// Requirement: FR-INS-041
func TestLossValue_Saturates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		lost, value []int64
		limit       int64
		want        int64
		wantSat     bool
	}{
		{"product overflows", []int64{1 << 62}, []int64{4}, 1000, 1000, true},
		{"sum overflows", []int64{1, 1}, []int64{1<<63 - 1, 5}, 1<<63 - 1, 1<<63 - 1, true},
		{"above the limit", []int64{3, 3}, []int64{2, 2}, 11, 11, true},
		{"at the limit", []int64{3, 3}, []int64{2, 2}, 12, 12, false},
		{"negative and zero entries count as 0", []int64{-1, 0, 2}, []int64{9, 9, 3}, 100, 6, false},
		{"read over the shorter", []int64{1, 1, 1}, []int64{5}, 100, 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, sat := insure.LossValue(tt.lost, tt.value, tt.limit)
			if got != tt.want || sat != tt.wantSat {
				t.Errorf("LossValue(%v, %v, %d) = %d, %v, want %d, %v", tt.lost, tt.value, tt.limit, got, sat, tt.want, tt.wantSat)
			}
		})
	}
}

// TestSortRequests_LimitThenKey verifies the order of the requests: limit
// descending, then Key, Party and Seq ascending, and Ref as the last tie-break.
//
// Requirement: FR-INS-033
func TestSortRequests_LimitThenKey(t *testing.T) {
	t.Parallel()
	rows := []insure.Request{
		{Ref: 0, Party: 5, Limit: 100, Key: 9},
		{Ref: 1, Party: 4, Limit: 200, Key: 9},
		{Ref: 2, Party: 3, Limit: 100, Key: 1},
		{Ref: 3, Party: 2, Limit: 100, Key: 9, Seq: 1},
		{Ref: 4, Party: 2, Limit: 100, Key: 9, Seq: 0},
		{Ref: 5, Party: 2, Limit: 100, Key: 9, Seq: 0},
	}
	insure.SortRequests(rows)
	var got []int
	for _, r := range rows {
		got = append(got, r.Ref)
	}
	// 200 first; then limit 100 by Key 1 (ref 2), then Key 9 by Party 2 (Seq 0: refs 4, 5; Seq 1: ref 3), then Party 5.
	if want := []int{1, 2, 4, 5, 3, 0}; !slices.Equal(got, want) {
		t.Errorf("SortRequests order = %v, want %v", got, want)
	}
}

// TestSortOffers_PriceThenKey verifies the two offer orders: Loading ascending for
// the lane requests and StoreRate ascending for the store requests, equal prices by
// Key, Party and Ref, and that a re-sort keeps the room.
//
// Requirement: FR-INS-033
func TestSortOffers_PriceThenKey(t *testing.T) {
	t.Parallel()
	offers := []insure.Offer{
		{Ref: 0, Party: 1, Loading: 300, StoreRate: 10, Key: 5, Room: 11},
		{Ref: 1, Party: 2, Loading: 100, StoreRate: 30, Key: 5, Room: 12},
		{Ref: 2, Party: 3, Loading: 100, StoreRate: 20, Key: 2, Room: 13},
		{Ref: 3, Party: 4, Loading: 100, StoreRate: 20, Key: 2, Room: 14},
	}
	refs := func() (out []int) {
		for _, o := range offers {
			out = append(out, o.Ref)
		}
		return out
	}
	insure.SortLoadings(offers)
	if got, want := refs(), []int{2, 3, 1, 0}; !slices.Equal(got, want) {
		t.Errorf("SortLoadings order = %v, want %v", got, want)
	}
	insure.SortStoreRates(offers)
	if got, want := refs(), []int{0, 2, 3, 1}; !slices.Equal(got, want) {
		t.Errorf("SortStoreRates order = %v, want %v", got, want)
	}
	if offers[3].Room != 12 {
		t.Errorf("offer 1 room after the re-sort = %d, want 12 (rows move whole)", offers[3].Room)
	}
}
