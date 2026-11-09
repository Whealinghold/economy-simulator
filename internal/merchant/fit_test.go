package merchant_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// TestFit_TwoPass verifies the sizing of a lot of spec-merchants.md 6.2: the cap
// binds when the budget is large, the budget binds when it is small (example B:
// (6 000 - 105) / 58 = 101 units), a budget that does not cover one unit or a
// break-even below 1 or one that fails gives a lot of 0, and the break-even is
// asked once, at the first lot.
//
// Requirement: FR-MER-020
func TestFit_TwoPass(t *testing.T) {
	t.Parallel()
	be := func(v int64, ok bool) func(int64) (int64, bool) { return func(int64) (int64, bool) { return v, ok } }
	tests := []struct {
		name string
		in   merchant.FitInput
		be   func(int64) (int64, bool)
		want int64
	}{
		// floor((6 000 - 0 - 105) / 58) = floor(101.6) = 101, under the cap of 1 500.
		{"the budget binds", merchant.FitInput{Cap: 1_500, Money: 6_000, Other: 105, High: 58}, be(64, true), 101},
		{"the cap binds", merchant.FitInput{Cap: 150, Money: 60_000, High: 58}, be(64, true), 150},
		{"a cap of one unit", merchant.FitInput{Cap: 1, Money: 60_000, High: 58}, be(64, true), 1},
		{"a bid price of 1 buys the whole budget", merchant.FitInput{Cap: 1_000, Money: 100, High: 1}, be(64, true), 100},
		{"a budget of exactly 0", merchant.FitInput{Cap: 10, Money: 100, Reserve: 60, Other: 40, High: 1}, be(64, true), 0},
		{"the break-even of exactly 1 is a bid", merchant.FitInput{Cap: 10, Money: 6_000, High: 58}, be(1, true), 10},
		{"the reserve is kept", merchant.FitInput{Cap: 1_500, Money: 6_000, Reserve: 5_000, High: 100}, be(64, true), 10},
		{"not one unit affordable", merchant.FitInput{Cap: 1_500, Money: 57, High: 58}, be(64, true), 0},
		{"money below the reserve", merchant.FitInput{Cap: 10, Money: 100, Reserve: 200, High: 1}, be(64, true), 0},
		{"the other inputs cost more than the cash", merchant.FitInput{Cap: 10, Money: 100, Other: 200, High: 1}, be(64, true), 0},
		{"no cap", merchant.FitInput{Cap: 0, Money: 6_000, High: 58}, be(64, true), 0},
		{"no bid price", merchant.FitInput{Cap: 10, Money: 6_000, High: 0}, be(64, true), 0},
		{"the break-even is below 1", merchant.FitInput{Cap: 10, Money: 6_000, High: 58}, be(0, true), 0},
		{"the break-even fails", merchant.FitInput{Cap: 10, Money: 6_000, High: 58}, be(64, false), 0},
		{"overflow of the budget", merchant.FitInput{Cap: 10, Money: -(1 << 63) + 1, Reserve: 1 << 62, High: 1}, be(64, true), 0},
		{"overflow of the other inputs", merchant.FitInput{Cap: 10, Money: 1 << 62, Other: -(1 << 63) + 1, High: 1}, be(64, true), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := merchant.Fit(tt.in, tt.be); got != tt.want {
				t.Errorf("Fit(%+v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
	calls, lot := 0, int64(0)
	got := merchant.Fit(merchant.FitInput{Cap: 7, Money: 1_000, High: 10}, func(q int64) (int64, bool) { calls++; lot = q; return 5, true })
	if got != 7 || calls != 1 || lot != 7 {
		t.Errorf("Fit asked the break-even %d times at %d and returned %d, want once at 7 and 7", calls, lot, got)
	}
}

// TestPerUnit_And_Units verifies the capacity of a carrying unit and the units a
// lot needs (spec-merchants.md 5.4, 6.2.1): example A (1 200 mass, 900 bulk, fuel
// of 2 and 6 holds 150, a lot of 150 needs 1), the porter (6 units), a mass
// capacity of 0 left out, and the overflow of a lot.
//
// Requirement: FR-MER-020, FR-MER-011
func TestPerUnit_And_Units(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                           string
		unitMass, unitBulk, mass, bulk int64
		want                           int64
		ok                             bool
	}{
		{"example A, bulk binds", 1_200, 900, 2, 6, 150, true},
		{"the porter", 60, 40, 2, 6, 6, true},
		{"mass binds", 100, 900, 50, 6, 2, true},
		{"mass capacity 0 left out", 0, 900, 2, 6, 150, true},
		{"bulk capacity 0 left out", 100, 0, 2, 6, 50, true},
		{"a good with no mass", 100, 900, 0, 6, 150, true},
		{"too heavy for one unit", 1, 900, 2, 6, 0, true},
		{"a capacity of 1 holds a good of 1", 1, 0, 1, 0, 1, true},
		{"a bulk capacity of 1 holds a good of 1", 0, 1, 0, 1, 1, true},
		{"a negative mass capacity is left out", -5, 900, 2, 6, 150, true},
		{"a negative bulk capacity is left out", 100, -5, 2, 6, 50, true},
		{"a negative mass is left out", 100, 900, -2, 6, 150, true},
		{"a negative bulk is left out", 100, 900, 2, -6, 50, true},
		{"both left out", 0, 0, 2, 6, 0, false},
		{"both goods figures 0", 100, 900, 0, 0, 0, false},
	} {
		if got, ok := merchant.PerUnit(tt.unitMass, tt.unitBulk, tt.mass, tt.bulk); got != tt.want || ok != tt.ok {
			t.Errorf("PerUnit(%s: %d, %d, %d, %d) = %d, %v, want %d, %v", tt.name, tt.unitMass, tt.unitBulk, tt.mass, tt.bulk, got, ok, tt.want, tt.ok)
		}
	}
	for _, tt := range []struct {
		name                              string
		q, mass, bulk, unitMass, unitBulk int64
		want                              int64
		ok                                bool
	}{
		// ceil(150 * 2 / 1 200) = 1 and ceil(150 * 6 / 900) = 1.
		{"example A", 150, 2, 6, 1_200, 900, 1, true},
		// ceil(151 * 6 / 900) = ceil(1.007) = 2.
		{"one unit over", 151, 2, 6, 1_200, 900, 2, true},
		{"a small lot still needs one", 1, 2, 6, 1_200, 900, 1, true},
		{"no lot needs one", 0, 2, 6, 1_200, 900, 1, true},
		{"no capacity at all needs one", 150, 2, 6, 0, 0, 1, true},
		{"mass capacity 0 left out", 450, 2, 6, 0, 900, 3, true},
		{"overflow", 1 << 62, 1 << 62, 1, 1, 1, 0, false},
	} {
		if got, ok := merchant.Units(tt.q, tt.mass, tt.bulk, tt.unitMass, tt.unitBulk); got != tt.want || ok != tt.ok {
			t.Errorf("Units(%s: %d, %d, %d, %d, %d) = %d, %v, want %d, %v", tt.name, tt.q, tt.mass, tt.bulk, tt.unitMass, tt.unitBulk, got, ok, tt.want, tt.ok)
		}
	}
}

// TestTake_Sizing verifies the take of a consignee of spec-merchants.md 8.2: the
// limit is the local price less the margin (300 less 10% is 270, example A), the
// quantity is what it lacks counting its stock above its keep and the goods on the
// way, capped by what it can pay above the reserve and what its orders reserved,
// and no quantity is no take.
//
// Requirement: FR-MER-028
func TestTake_Sizing(t *testing.T) {
	t.Parallel()
	base := merchant.TakeInput{Price: 300, MarginBP: 1_000, Units: 150, Money: 1_000_000, MaxQty: 1_000}
	with := func(f func(in *merchant.TakeInput)) merchant.TakeInput { in := base; f(&in); return in }
	tests := []struct {
		name       string
		in         merchant.TakeInput
		limit, qty int64
		ok         bool
	}{
		{"example A", base, 270, 150, true},
		// want = 150 - max(0, 60 - 10) = 100.
		{"stock above the keep counts", with(func(in *merchant.TakeInput) { in.Stock, in.Keep = 60, 10 }), 270, 100, true},
		{"stock below the keep does not", with(func(in *merchant.TakeInput) { in.Stock, in.Keep = 5, 10 }), 270, 150, true},
		{"the pipeline counts", with(func(in *merchant.TakeInput) { in.Inbound = 120 }), 270, 30, true},
		{"nothing is lacking", with(func(in *merchant.TakeInput) { in.Inbound = 150 }), 0, 0, false},
		{"more than needed on the way", with(func(in *merchant.TakeInput) { in.Inbound = 400 }), 0, 0, false},
		// afford = floor((1 000 - 100 - 200) / 270) = floor(2.59) = 2.
		{"cash binds", with(func(in *merchant.TakeInput) { in.Money, in.Reserve, in.Reserved = 1_000, 100, 200 }), 270, 2, true},
		{"cash below the reserve", with(func(in *merchant.TakeInput) { in.Money, in.Reserve = 50, 100 }), 0, 0, false},
		{"cash exactly the reserve", with(func(in *merchant.TakeInput) { in.Money, in.Reserve, in.Price, in.MarginBP = 100, 100, 1, 0 }), 0, 0, false},
		// 270 of cash buys exactly one unit at 270.
		{"cash for exactly one unit", with(func(in *merchant.TakeInput) { in.Money = 270 }), 270, 1, true},
		{"cash one short of a unit", with(func(in *merchant.TakeInput) { in.Money = 269 }), 0, 0, false},
		{"the order cap binds", with(func(in *merchant.TakeInput) { in.MaxQty = 40 }), 270, 40, true},
		// 1 * 9 000 / 10 000 = 0.9 truncated to 0, raised to 1.
		{"the limit is at least 1", with(func(in *merchant.TakeInput) { in.Price, in.MarginBP = 1, 1_000 }), 1, 150, true},
		{"no margin", with(func(in *merchant.TakeInput) { in.MarginBP = 0 }), 300, 150, true},
		// every input above 0: want = 150 - (10 - 5) - 10 = 135, afford = (1 000 000 - 10 - 10) / 270 = 3 703.
		{"every input positive", with(func(in *merchant.TakeInput) {
			in.Stock, in.Keep, in.Inbound, in.Reserve, in.Reserved = 10, 5, 10, 10, 10
		}), 270, 135, true},
		{"the whole price taken off", with(func(in *merchant.TakeInput) { in.MarginBP = 10_000 }), 1, 150, true},
		{"a price of 0", with(func(in *merchant.TakeInput) { in.Price = 0 }), 1, 150, true},
		{"margin above 10 000", with(func(in *merchant.TakeInput) { in.MarginBP = 10_001 }), 0, 0, false},
		{"negative margin", with(func(in *merchant.TakeInput) { in.MarginBP = -1 }), 0, 0, false},
		{"negative price", with(func(in *merchant.TakeInput) { in.Price = -1 }), 0, 0, false},
		{"negative units", with(func(in *merchant.TakeInput) { in.Units = -1 }), 0, 0, false},
		{"negative pipeline", with(func(in *merchant.TakeInput) { in.Inbound = -1 }), 0, 0, false},
		{"a price beyond the cash affords nothing", with(func(in *merchant.TakeInput) { in.Price = 1 << 62; in.MarginBP = 0 }), 0, 0, false},
		{"overflow of the pipeline", with(func(in *merchant.TakeInput) { in.Units, in.Inbound = -0, 1<<62; in.Stock = -(1 << 62) - (1 << 62) }), 0, 0, false},
		{"overflow of the cash", with(func(in *merchant.TakeInput) { in.Money, in.Reserve = -(1 << 62), 1<<62 }), 0, 0, false},
		{"overflow of the reserved cash", with(func(in *merchant.TakeInput) { in.Money, in.Reserve, in.Reserved = 0, 0, -(1<<62)-(1<<62) }), 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			limit, qty, ok := merchant.Take(tt.in)
			if ok != tt.ok || limit != tt.limit || qty != tt.qty {
				t.Errorf("Take(%+v) = %d, %d, %v, want %d, %d, %v", tt.in, limit, qty, ok, tt.limit, tt.qty, tt.ok)
			}
		})
	}
}
