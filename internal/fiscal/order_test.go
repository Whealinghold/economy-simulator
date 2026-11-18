package fiscal_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

// TestLimit_Table verifies the limit price of spec-fiscal.md 9.2: reference 50
// with 8 000 bp and 12 000 bp gives 40 and 60, a product below 1 is clamped to 1
// and one above the bound to maxPrice.
//
// Requirement: FR-FIS-050
func TestLimit_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		ref, bp, max, want int64
	}{
		{"purchase at 80 percent", 50, 8_000, 1_000, 40},
		{"release at 120 percent", 50, 12_000, 1_000, 60},
		{"truncates toward zero", 7, 8_000, 1_000, 5}, // 56000 / 10000 = 5.6
		{"clamped up to 1", 1, 1_000, 1_000, 1},
		{"clamped down to the bound", 900, 20_000, 1_000, 1_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := fiscal.Limit(tt.ref, tt.bp, tt.max); !ok || got != tt.want {
				t.Errorf("Limit(%d, %d, %d) = %d, %v; want %d, true", tt.ref, tt.bp, tt.max, got, ok, tt.want)
			}
		})
	}
	for _, c := range [][3]int64{{-1, 1, 1}, {1, -1, 1}, {1, 1, 0}, {math.MaxInt64, math.MaxInt64, 1}} {
		if got, ok := fiscal.Limit(c[0], c[1], c[2]); ok {
			t.Errorf("Limit(%d, %d, %d) = %d, true; want ok false", c[0], c[1], c[2], got)
		}
	}
}

// TestWish_Carry verifies the wish of spec-fiscal.md 9.3.1: a quantity of 1 at
// scale 5 000 wants one unit every second tick, and the carry is (qty*scale +
// carry) mod 10^4.
//
// Requirement: FR-FIS-051
func TestWish_Carry(t *testing.T) {
	t.Parallel()
	var carry, total int64
	var got []int64
	for range 6 {
		w, next, ok := fiscal.Wish(1, 5_000, carry)
		if !ok {
			t.Fatalf("Wish(1, 5000, %d) not ok", carry)
		}
		got, carry, total = append(got, w), next, total+w
	}
	if total != 3 || got[0] != 0 || got[1] != 1 || got[2] != 0 || got[3] != 1 {
		t.Errorf("six wishes of 1 at scale 5000 = %v, want 0 1 0 1 0 1", got)
	}
	for _, c := range [][3]int64{{-1, 1, 0}, {1, -1, 0}, {1, 1, -1}, {1, 1, 10_000}} {
		if w, _, ok := fiscal.Wish(c[0], c[1], c[2]); ok {
			t.Errorf("Wish(%d, %d, %d) = %d, true; want ok false", c[0], c[1], c[2], w)
		}
	}
}

// TestWish_Saturates verifies that a wish above int64 saturates at MaxInt64, as a
// due of internal/moneyflow does, instead of wrapping; a validated Quantity is far
// below it (MaxQuantity), so the case is a bound violation only.
//
// Requirement: FR-FIS-051, NFR-FIS-003
func TestWish_Saturates(t *testing.T) {
	t.Parallel()
	if w, _, ok := fiscal.Wish(math.MaxInt64, math.MaxInt64, 0); !ok || w != math.MaxInt64 {
		t.Errorf("Wish(max, max, 0) = %d, %v; want the saturated %d, true", w, ok, int64(math.MaxInt64))
	}
}

// TestPairOK_EqualLimits verifies the pair rule of spec-fiscal.md 9.4.1 on the
// integer limits: a reference price of 50 with 8 000 and 8 100 bp gives 40 and
// 40 (50 * 8 100 / 10 000 = 40.5, truncated), which cross, so the pair is
// refused; 8 000 and 12 000 give 40 and 60, which do not.
//
// Requirement: FR-FIS-053
func TestPairOK_EqualLimits(t *testing.T) {
	t.Parallel()
	buy, _ := fiscal.Limit(50, 8_000, 1_000)
	equal, _ := fiscal.Limit(50, 8_100, 1_000)
	sell, _ := fiscal.Limit(50, 12_000, 1_000)
	if fiscal.PairOK(buy, equal) {
		t.Errorf("PairOK(%d, %d) = true, want false: equal integer limits cross", buy, equal)
	}
	if !fiscal.PairOK(buy, sell) {
		t.Errorf("PairOK(%d, %d) = false, want true", buy, sell)
	}
	if fiscal.PairOK(60, 40) {
		t.Errorf("PairOK(60, 40) = true, want false: a bid above the ask crosses")
	}
}

// TestCapPurchase verifies the caps of a purchase in the order of
// spec-fiscal.md 9.3.2: the cash first, floor(cash / limit), then the hold,
// each counted; a hold of 0 means none; a limit below 1 buys nothing.
//
// Requirement: FR-FIS-051
func TestCapPurchase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                          string
		wish, cash, limit, hold, held int64
		want                          fiscal.Capped
	}{
		{"uncapped", 5, 1_000, 40, 0, 0, fiscal.Capped{Qty: 5}},
		{"cash cuts", 5, 100, 40, 0, 0, fiscal.Capped{Qty: 2, CutCash: 3}},
		{"hold cuts", 5, 1_000, 40, 10, 8, fiscal.Capped{Qty: 2, CutHold: 3}},
		{"cash then hold", 10, 200, 40, 3, 0, fiscal.Capped{Qty: 3, CutCash: 5, CutHold: 2}},
		{"hold already passed", 5, 1_000, 40, 10, 12, fiscal.Capped{Qty: 0, CutHold: 5}},
		{"limit below 1", 5, 1_000, 0, 0, 0, fiscal.Capped{Qty: 0, CutCash: 5}},
		{"negative wish", -3, 1_000, 40, 0, 0, fiscal.Capped{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fiscal.CapPurchase(tt.wish, tt.cash, tt.limit, tt.hold, tt.held); got != tt.want {
				t.Errorf("CapPurchase(%d, %d, %d, %d, %d) = %+v, want %+v", tt.wish, tt.cash, tt.limit, tt.hold, tt.held, got, tt.want)
			}
		})
	}
}

// TestCapRelease verifies the cap of a release: the unreserved stock, held less
// reserve, never negative.
//
// Requirement: FR-FIS-051
func TestCapRelease(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		wish, held, reserve int64
		want                fiscal.Capped
	}{
		{"uncapped", 5, 20, 10, fiscal.Capped{Qty: 5}},
		{"stock cuts", 15, 20, 10, fiscal.Capped{Qty: 10, CutStock: 5}},
		{"reserve above stock", 4, 6, 10, fiscal.Capped{Qty: 0, CutStock: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fiscal.CapRelease(tt.wish, tt.held, tt.reserve); got != tt.want {
				t.Errorf("CapRelease(%d, %d, %d) = %+v, want %+v", tt.wish, tt.held, tt.reserve, got, tt.want)
			}
		})
	}
}
