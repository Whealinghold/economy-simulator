package merchant_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// TestCarriage_Hired verifies the cost of a hired trip, n * s * the price of a haul
// unit, at example A (1 * 6 * 700 = 4 200), and that a lot of no carrying unit or
// an overflow is refused.
//
// Requirement: FR-MER-011
func TestCarriage_Hired(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		c    merchant.Carriage
		want int64
		ok   bool
	}{
		{"example A", merchant.Carriage{N: 1, K: 2, Hired: true, S: 6, HaulPrice: 700}, 4_200, true},
		{"three carrying units", merchant.Carriage{N: 3, K: 2, Hired: true, S: 6, HaulPrice: 700}, 12_600, true},
		{"the wear and the feed of a self-carried mode are not charged", merchant.Carriage{N: 1, K: 2, Hired: true, S: 6, HaulPrice: 700, WearPerTick: 5, Durability: 10, VehiclePrice: 9, UpkeepCost: 9, TimeCostBP: 9_999}, 4_200, true},
		{"no carrying unit", merchant.Carriage{N: 0, K: 2, Hired: true, S: 6, HaulPrice: 700}, 0, false},
		{"a travel time of 0", merchant.Carriage{N: 1, K: 0, Hired: true, S: 6, HaulPrice: 700}, 4_200, true},
		{"overflow", merchant.Carriage{N: 1 << 40, K: 2, Hired: true, S: 1 << 40, HaulPrice: 1}, 0, false},
		{"overflow of the price", merchant.Carriage{N: 1, K: 2, Hired: true, S: 1 << 40, HaulPrice: 1 << 40}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tt.c.Cost(); got != tt.want || ok != tt.ok {
				t.Errorf("Carriage%+v.Cost() = %d, %v, want %d, %v", tt.c, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestCarriage_SelfCarried verifies the cost of a self-carried trip, wear + feed +
// time, at example B: wear ceil(1 * 5 * 3 * 4 000 / 400) = 150, feed 1 * 1 * 3 * 30
// = 90, time 3 * 56 * 10 000 / 10 000 = 168, in all 408; the wear rounded up, the
// time truncated, no vehicle no wear, and the overflow of each term.
//
// Requirement: FR-MER-011
func TestCarriage_SelfCarried(t *testing.T) {
	t.Parallel()
	skiff := merchant.Carriage{N: 1, K: 3, WearPerTick: 5, VehiclePrice: 4_000, Durability: 400,
		Feed: []merchant.Feed{{Quantity: 1, Price: 30}}, UpkeepCost: 56, TimeCostBP: 10_000}
	with := func(f func(c *merchant.Carriage)) merchant.Carriage {
		c := skiff
		c.Feed = append([]merchant.Feed(nil), skiff.Feed...)
		f(&c)
		return c
	}
	tests := []struct {
		name string
		c    merchant.Carriage
		want int64
		ok   bool
	}{
		{"example B", skiff, 408, true},
		// wear = ceil(1 * 5 * 3 * 4 001 / 400) = ceil(150.04) = 151: rounded up.
		{"wear rounded up", with(func(c *merchant.Carriage) { c.VehiclePrice = 4_001 }), 409, true},
		{"no vehicle, no wear", with(func(c *merchant.Carriage) { c.Durability = 0 }), 258, true},
		// time = 3 * 56 * 5 000 / 10 000 = 84.
		{"half the time cost", with(func(c *merchant.Carriage) { c.TimeCostBP = 5_000 }), 324, true},
		// time = 3 * 55 * 9 999 / 10 000 = 164.98, truncated to 164.
		{"time truncated", with(func(c *merchant.Carriage) { c.UpkeepCost, c.TimeCostBP = 55, 9_999 }), 150 + 90 + 164, true},
		{"two feed entries", with(func(c *merchant.Carriage) { c.Feed = append(c.Feed, merchant.Feed{Quantity: 2, Price: 10}) }), 408 + 60, true},
		{"two carrying units", with(func(c *merchant.Carriage) { c.N = 2 }), 2*150 + 2*90 + 168, true},
		{"overflow of the wear", with(func(c *merchant.Carriage) { c.WearPerTick = 1 << 62 }), 0, false},
		{"overflow of the wear times the time", with(func(c *merchant.Carriage) { c.N, c.WearPerTick = 1<<31, 1<<31 }), 0, false},
		{"overflow of the price of the vehicle", with(func(c *merchant.Carriage) { c.VehiclePrice, c.Durability = 1<<62, 1 }), 0, false},
		{"overflow of the feed", with(func(c *merchant.Carriage) { c.Feed[0].Quantity = 1 << 62 }), 0, false},
		{"overflow of the feed price", with(func(c *merchant.Carriage) { c.Feed[0].Price = 1 << 62 }), 0, false},
		{"overflow of the feed sum", with(func(c *merchant.Carriage) {
			c.Feed = []merchant.Feed{{Quantity: 1, Price: 1 << 61}, {Quantity: 1, Price: 1 << 61}, {Quantity: 1, Price: 1 << 61}, {Quantity: 1, Price: 1 << 61}}
			c.K = 1
		}), 0, false},
		{"overflow of the time", with(func(c *merchant.Carriage) { c.UpkeepCost = 1 << 62 }), 0, false},
		{"overflow of the time times the share", with(func(c *merchant.Carriage) { c.K, c.UpkeepCost, c.TimeCostBP = 1<<31, 1<<31, 1<<40 }), 0, false},
		{"overflow of the sum", with(func(c *merchant.Carriage) { c.Durability, c.UpkeepCost, c.TimeCostBP, c.Feed[0].Price = 0, 0, 0, 1<<62 }), 0, false},
		{"a negative travel time", with(func(c *merchant.Carriage) { c.K = -1 }), 0, false},
		{"a travel time of 0 costs nothing", with(func(c *merchant.Carriage) { c.K = 0 }), 0, true},
		// wear 2^62 and feed 2^62 sum to 2^63.
		{"overflow of the wear plus the feed", merchant.Carriage{N: 1, K: 1, WearPerTick: 1, VehiclePrice: 1 << 62, Durability: 1, Feed: []merchant.Feed{{Quantity: 1, Price: 1 << 62}}}, 0, false},
		{"overflow of the units times the wear", with(func(c *merchant.Carriage) { c.N, c.WearPerTick = 1<<32, 1<<32 }), 0, false},
		{"overflow of the units times the time", with(func(c *merchant.Carriage) { c.N, c.K, c.Durability = 1<<32, 1<<32, 0 }), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tt.c.Cost(); got != tt.want || ok != tt.ok {
				t.Errorf("Carriage%+v.Cost() = %d, %v, want %d, %v", tt.c, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestCarriage_Toll verifies that, with the fiscal module on, every mode adds n
// times the toll of a carrying unit to the cost (spec-merchants.md 5.4), and that
// the toll overflow is refused.
//
// Requirement: FR-MER-012
func TestCarriage_Toll(t *testing.T) {
	t.Parallel()
	hired := merchant.Carriage{N: 2, K: 2, Hired: true, S: 6, HaulPrice: 700, Toll: 25}
	if got, ok := hired.Cost(); !ok || got != 2*6*700+2*25 {
		t.Errorf("hired Cost() with a toll = %d, %v, want %d", got, ok, 2*6*700+2*25)
	}
	self := merchant.Carriage{N: 2, K: 3, UpkeepCost: 56, TimeCostBP: 10_000, Toll: 25}
	if got, ok := self.Cost(); !ok || got != 3*56+2*25 {
		t.Errorf("self-carried Cost() with a toll = %d, %v, want %d", got, ok, 3*56+2*25)
	}
	self.Toll = 1 << 62
	if got, ok := self.Cost(); ok {
		t.Errorf("Cost() with a toll that overflows = %d, true, want ok false", got)
	}
	self.Toll = 1 << 62 // 2^62 of toll plus 2^62 of time is 2^63
	self.N = 1
	self.UpkeepCost = 1 << 62
	self.K = 1
	if got, ok := self.Cost(); ok {
		t.Errorf("Cost() with a base and a toll that overflow = %d, true, want ok false", got)
	}
}
