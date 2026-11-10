package merchant_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// TestCautious_And_Net verifies the cautious revenue and the net revenue of
// spec-merchants.md 5.2.1: the price when there is no mean, the lower of price and
// mean otherwise, the consignee's cut and the expected loss truncated in turn, and
// a share outside 0 to 10 000 refused.
//
// Requirement: FR-MER-008
func TestCautious_And_Net(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ price, mean, want int64 }{
		{310, 305, 305}, {300, 305, 300}, {310, 0, 310}, {0, 0, 0},
	} {
		if got := merchant.Cautious(tt.price, tt.mean); got != tt.want {
			t.Errorf("Cautious(%d, %d) = %d, want %d", tt.price, tt.mean, got, tt.want)
		}
	}
	for _, tt := range []struct {
		name            string
		rhat, cut, loss int64
		want            int64
		ok              bool
	}{
		// 305 * 9 500 / 10 000 = 289.75, truncated to 289 (example A).
		{"hired", 305, 500, 0, 289, true},
		{"self-carried", 305, 0, 0, 305, true},
		// 305 * 9 500 / 10 000 = 289, then 289 * 9 000 / 10 000 = 260.1, truncated to 260.
		{"loss after the cut", 305, 500, 1_000, 260, true},
		{"the whole revenue to the consignee", 305, 10_000, 0, 0, true},
		{"the whole revenue lost", 305, 0, 10_000, 0, true},
		{"cut above 10 000", 305, 10_001, 0, 0, false},
		{"loss above 10 000", 305, 0, 10_001, 0, false},
		{"cut below 0", 305, -1, 0, 0, false},
		{"loss below 0", 305, 0, -1, 0, false},
		{"overflow", 1 << 62, 0, 0, 1 << 62, true},
	} {
		got, ok := merchant.Net(tt.rhat, tt.cut, tt.loss)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Net(%s: %d, %d, %d) = %d, %v, want %d, %v", tt.name, tt.rhat, tt.cut, tt.loss, got, ok, tt.want, tt.ok)
		}
	}
}

// TestPerishable_Boundary verifies the life left and the perishable test of
// spec-merchants.md 5.3.1 at the boundary of example A: a fresh lot bought at t
// and sent at t + 1 has age 1, so a shelf life of 3 leaves 1 tick (a trip of 2 is
// set aside) and a shelf life of 4 leaves 2 (it passes); an age of -1 (no lot) is
// read as 1; a shelf life of 0 never spoils.
//
// Requirement: FR-MER-010
func TestPerishable_Boundary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		shelf, age, k int64
		life          int64
		want          bool
	}{
		{"shelf 3, trip 2", 3, 1, 2, 1, true},
		{"shelf 4, trip 2", 4, 1, 2, 2, false},
		{"trip equal to the life", 4, 1, 2, 2, false},
		{"trip one above the life", 4, 1, 3, 2, true},
		{"an older lot", 5, 3, 2, 1, true},
		{"no lot held reads as age 1", 4, -1, 2, 2, false},
		{"age 0 reads as age 1", 4, 0, 2, 2, false},
		{"never spoils", 0, 50, 1_000, -51, false},
		{"a shelf life of 1 spoils on any trip", 1, 1, 1, -1, true},
		{"a trip shorter than the life", 4, 1, 1, 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.shelf > 0 {
				if got := merchant.LifeLeft(tt.shelf, tt.age); got != tt.life {
					t.Errorf("LifeLeft(%d, %d) = %d, want %d", tt.shelf, tt.age, got, tt.life)
				}
			}
			if got := merchant.Perishable(tt.shelf, tt.age, tt.k); got != tt.want {
				t.Errorf("Perishable(%d, %d, %d) = %v, want %v", tt.shelf, tt.age, tt.k, got, tt.want)
			}
		})
	}
}

// TestPass_Inequality verifies the test of the fee share, c * 10 000 <= phi * g,
// at the four brokers of example A (cost 4 200, gross margin 13 350: phi 6 385,
// 4 476 and 5 655 pass, 2 566 fails), exactly at the boundary, and for a gross
// margin that is not positive or a negative cost.
//
// Requirement: FR-MER-013
func TestPass_Inequality(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		c, phi, g int64
		want      bool
	}{
		// 42 000 000 <= 6 385 * 13 350 = 85 239 750.
		{"agent 32", 4_200, 6_385, 13_350, true},
		// 42 000 000 <= 4 476 * 13 350 = 59 754 600.
		{"agent 33", 4_200, 4_476, 13_350, true},
		// 42 000 000 > 2 566 * 13 350 = 34 256 100.
		{"agent 34", 4_200, 2_566, 13_350, false},
		{"exactly equal", 5_000, 10_000, 5_000, true},
		{"one unit short", 5_001, 10_000, 5_000, false},
		{"a zero cost, share and margin", 0, 0, 5, true},
		{"a zero margin and a zero cost", 0, 5, 0, false},
		{"a negative share", 5, -1, 5, false},
		// 2^62 * 10 000 against 2^62 * 20 000: the high words differ and the left is lower.
		{"a wide product, the share double", 1 << 62, 20_000, 1 << 62, true},
		// the left is higher.
		{"a wide product, the share half", 1 << 62, 5_000, 1 << 62, false},
		{"zero cost", 0, 1, 1, true},
		{"no margin", 1, 10_000, 0, false},
		{"negative margin", 1, 10_000, -5, false},
		{"negative cost", -1, 10_000, 5, false},
		{"a product beyond 64 bits", 1 << 62, 10_000, 1 << 62, true},
		{"a product beyond 64 bits, short", 1<<62 + 1, 10_000, 1 << 62, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := merchant.Pass(tt.c, tt.phi, tt.g); got != tt.want {
				t.Errorf("Pass(%d, %d, %d) = %v, want %v", tt.c, tt.phi, tt.g, got, tt.want)
			}
		})
	}
}

// TestBreakEven_AskLimit verifies the margin needed, the break-even and the ask
// limit of spec-merchants.md 5.5.3 against the worked examples: the four brokers
// of example A (break-even 245, 226, 179, 239 and offer limit 244, 263, 310, 250)
// and the trader of example B (break-even 64), and that the score is G - C.
//
// Requirement: FR-MER-015
func TestBreakEven_AskLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                      string
		c, phi, q, rnet, pacq     int64
		g, score, needed, be, ask int64
	}{
		// needed = ceil(4 200 * 10 000 / (6 385 * 150)) = ceil(43.85) = 44.
		{"agent 32", 4_200, 6_385, 150, 289, 200, 13_350, 9_150, 44, 245, 244},
		// ceil(42 000 000 / 671 400) = ceil(62.56) = 63.
		{"agent 33", 4_200, 4_476, 150, 289, 200, 13_350, 9_150, 63, 226, 263},
		// ceil(42 000 000 / 384 900) = ceil(109.12) = 110.
		{"agent 34", 4_200, 2_566, 150, 289, 200, 13_350, 9_150, 110, 179, 310},
		// ceil(42 000 000 / 848 250) = ceil(49.51) = 50.
		{"agent 35", 4_200, 5_655, 150, 289, 200, 13_350, 9_150, 50, 239, 250},
		// ceil(408 * 10 000 / (3 123 * 100)) = ceil(13.06) = 14.
		{"trader, example B", 408, 3_123, 100, 78, 56, 2_200, 1_792, 14, 64, 70},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := merchant.Margin(tt.c, tt.phi, tt.q, tt.rnet, tt.pacq)
			want := merchant.Result{G: tt.g, Score: tt.score, Needed: tt.needed, BreakEven: tt.be, AskLimit: tt.ask}
			if !ok || got != want {
				t.Errorf("Margin(%d, %d, %d, %d, %d) = %+v, %v, want %+v, true", tt.c, tt.phi, tt.q, tt.rnet, tt.pacq, got, ok, want)
			}
		})
	}
}

// TestHaulCeiling verifies the ceiling of the bid for a haul unit of
// spec-merchants.md 5.6: the expected margin per haul unit G / (n * s) truncated
// (2 225 in example A), times the fee share truncated (1 420, 995, 570 and 1 258
// for the four brokers), the toll subtracted from the margin, and a lot with no
// carrying unit refused.
//
// Requirement: FR-MER-016, FR-CAR-004
func TestHaulCeiling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		g, toll, n, s, phi int64
		want               int64
		ok                 bool
	}{
		{"agent 32", 13_350, 0, 1, 6, 6_385, 1_420, true},
		{"agent 33", 13_350, 0, 1, 6, 4_476, 995, true},
		{"agent 34", 13_350, 0, 1, 6, 2_566, 570, true},
		{"agent 35", 13_350, 0, 1, 6, 5_655, 1_258, true},
		// (13 350 - 2 * 600) / (2 * 6) = 12 150 / 12 = 1 012 (1 012.5 truncated), times 5 000 / 10 000 = 506.
		{"two carrying units with a toll", 13_350, 600, 2, 6, 5_000, 506, true},
		{"a toll above the margin is negative", 100, 600, 1, 1, 5_000, -250, true},
		{"no carrying unit", 13_350, 0, 0, 6, 5_000, 0, false},
		{"one carrying unit, one haul unit", 100, 0, 1, 1, 5_000, 50, true},
		{"a share of 0 gives a ceiling of 0", 100, 0, 1, 1, 0, 0, true},
		{"a negative share", 100, 0, 1, 1, -1, 0, false},
		{"no haul unit", 13_350, 0, 1, 0, 5_000, 0, false},
		{"overflow of the haul units", 1, 0, 1 << 40, 1 << 40, 5_000, 0, false},
		{"overflow of the toll", 1, 1 << 40, 1 << 40, 1, 5_000, 0, false},
		{"overflow of the margin less the toll", -(1 << 62) - (1 << 61), 1 << 62, 1, 1, 5_000, 0, false},
		{"overflow of the ceiling", 1 << 62, 0, 1, 1, 1 << 62, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := merchant.HaulCeiling(tt.g, tt.toll, tt.n, tt.s, tt.phi)
			if got != tt.want || ok != tt.ok {
				t.Errorf("HaulCeiling(%d, %d, %d, %d, %d) = %d, %v, want %d, %v", tt.g, tt.toll, tt.n, tt.s, tt.phi, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestRank_BestScoreTiesLow verifies that the best candidate is the greatest
// score and that a tie goes to the lower destination, then cargo, then mode
// (spec-merchants.md 5.1.2).
//
// Requirement: FR-MER-014
func TestRank_BestScoreTiesLow(t *testing.T) {
	t.Parallel()
	a := merchant.Cand{Score: 10, Dest: 2, Cargo: 3, Mode: 1}
	tests := []struct {
		name string
		b    merchant.Cand
		want bool
	}{
		{"a lower score loses", merchant.Cand{Score: 9, Dest: 0, Cargo: 0, Mode: 0}, true},
		{"a higher score wins", merchant.Cand{Score: 11, Dest: 9, Cargo: 9, Mode: 9}, false},
		{"a tie goes to the lower destination", merchant.Cand{Score: 10, Dest: 3, Cargo: 0, Mode: 0}, true},
		{"a tie goes to the lower cargo", merchant.Cand{Score: 10, Dest: 2, Cargo: 4, Mode: 0}, true},
		{"a tie goes to the lower mode", merchant.Cand{Score: 10, Dest: 2, Cargo: 3, Mode: 2}, true},
		{"the other way round, a higher destination loses", merchant.Cand{Score: 10, Dest: 1, Cargo: 9, Mode: 9}, false},
		{"the other way round, a higher cargo loses", merchant.Cand{Score: 10, Dest: 2, Cargo: 2, Mode: 9}, false},
		{"the other way round, a higher mode loses", merchant.Cand{Score: 10, Dest: 2, Cargo: 3, Mode: 0}, false},
		{"equal is not better", a, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := merchant.Better(a, tt.b); got != tt.want {
				t.Errorf("Better(%+v, %+v) = %v, want %v", a, tt.b, got, tt.want)
			}
		})
	}
}

// TestMargin_OverflowDrops verifies that a lot whose arithmetic does not fit, or
// that is not a lot, returns ok false and not a wrapped figure, so the candidate
// is dropped (spec-merchants.md 5.8), and that Gross reports an overflow.
//
// Requirement: FR-MER-018
func TestMargin_OverflowDrops(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                  string
		c, phi, q, rnet, pacq int64
	}{
		{"gross margin overflows", 1, 5_000, 1 << 40, 1 << 40, 0},
		{"price difference overflows", 1, 5_000, 1, 1 << 62, -(1 << 62) - (1 << 62)},
		// G = 1 * (-2^62 - 2^62) = -2^63, and G - 1 does not fit.
		{"score overflows", 1, 5_000, 1, -(1 << 62), 1 << 62},
		{"fee share times lot overflows", 1, 1 << 62, 1 << 40, 10, 5},
		{"no lot", 1, 5_000, 0, 10, 5},
		{"no share", 1, 0, 1, 10, 5},
		{"negative cost", -1, 5_000, 1, 10, 5},
		{"break-even overflows", 1 << 62, 1, 1, -(1 << 62), -(1 << 62) - 1},
		{"margin needed overflows", 1 << 62, 1, 1, 10, 5},
		{"ask limit overflows", 1, 10_000, 1, 1<<63 - 1, 1<<63 - 1},
	} {
		if got, ok := merchant.Margin(tt.c, tt.phi, tt.q, tt.rnet, tt.pacq); ok {
			t.Errorf("Margin(%s: %d, %d, %d, %d, %d) = %+v, true, want ok false", tt.name, tt.c, tt.phi, tt.q, tt.rnet, tt.pacq, got)
		}
	}
	if g, ok := merchant.Gross(1<<40, 1<<40, 0); ok || g != 0 {
		t.Errorf("Gross(2^40, 2^40, 0) = %d, %v, want 0, false", g, ok)
	}
	if g, ok := merchant.Gross(2, 1<<62, -(1 << 62)); ok || g != 0 {
		t.Errorf("Gross(2, 2^62, -2^62) = %d, %v, want 0, false: the price difference overflows", g, ok)
	}
	// The lowest figures that are a lot: q 1, a share of 1, a cost of 0.
	if r, ok := merchant.Margin(0, 1, 1, 10, 5); !ok || r != (merchant.Result{G: 5, Score: 5, Needed: 0, BreakEven: 10, AskLimit: 5}) {
		t.Errorf("Margin(0, 1, 1, 10, 5) = %+v, %v, want G 5, Score 5, Needed 0, BreakEven 10, AskLimit 5", r, ok)
	}
}

// TestRounding_AgainstMerchant verifies that the roundings of spec-merchants.md 5.8
// go against the merchant: the needed margin is rounded up (so the break-even is
// not dearer and the ask limit not cheaper than the exact figure), and the net
// revenue and the haul ceiling are truncated down.
//
// Requirement: FR-MER-018
func TestRounding_AgainstMerchant(t *testing.T) {
	t.Parallel()
	// C * 10 000 / (phi * q) = 10 000 / (5 000 * 3) = 0.667: rounded up to 1, not down to 0.
	r, ok := merchant.Margin(1, 5_000, 3, 100, 50)
	if !ok || r.Needed != 1 || r.BreakEven != 99 || r.AskLimit != 51 {
		t.Errorf("Margin(1, 5000, 3, 100, 50) = %+v, %v, want Needed 1, BreakEven 99, AskLimit 51", r, ok)
	}
	// An exact quotient is not rounded up: 5 000 * 10 000 / (5 000 * 1 000) = 10.
	if r, _ := merchant.Margin(5_000, 5_000, 1_000, 100, 50); r.Needed != 10 {
		t.Errorf("Margin(5000, 5000, 1000, 100, 50).Needed = %d, want exactly 10", r.Needed)
	}
	// 101 * 9 999 / 10 000 = 100.99, truncated to 100.
	if got, _ := merchant.Net(101, 1, 0); got != 100 {
		t.Errorf("Net(101, 1, 0) = %d, want 100 (truncated)", got)
	}
	// 7 / 2 = 3 per haul unit, times 9 999 / 10 000 = 2.9997, truncated to 2.
	if got, _ := merchant.HaulCeiling(7, 0, 1, 2, 9_999); got != 2 {
		t.Errorf("HaulCeiling(7, 0, 1, 2, 9999) = %d, want 2 (truncated twice)", got)
	}
}
