package household_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// TestFed_CountsWholeBaskets verifies that Fed is the least over the upkeep rows
// of floor(consumed / per household), capped at the weight, with no rounding up
// in favour of the household.
//
// Requirement: FR-AGT-014
func TestFed_CountsWholeBaskets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		consumed []int64
		per      []int64
		weight   int64
		want     int64
	}{
		// AGT 16.3: floor(100 / 3) = 33 and floor(40 / 1) = 40, the least is 33.
		{"example 16.3", []int64{100, 40}, []int64{3, 1}, 40, 33},
		// No upkeep: every household is fed.
		{"no rows gives the weight", nil, nil, 7, 7},
		// floor(500 / 3) = 166 is above the weight 40.
		{"capped at the weight", []int64{500}, []int64{3}, 40, 40},
		// 2 held against 3 needed: no whole basket, weight 1 gives 0.
		{"weight 1 short of one basket", []int64{2}, []int64{3}, 1, 0},
		// 40 households hold 80 against 3 each: floor(80 / 3) = 26.
		{"cohort of 40 short", []int64{80}, []int64{3}, 40, 26},
		{"a zero row is skipped", []int64{0, 9}, []int64{0, 3}, 5, 3},
		{"a negative quantity is skipped", []int64{0}, []int64{-2}, 5, 5},
		{"negative consumed counts as 0", []int64{-5}, []int64{3}, 5, 0},
		{"weight 0", []int64{9}, []int64{3}, 0, 0},
		{"negative weight", nil, nil, -3, 0},
		{"only the shorter prefix is read", []int64{9, 0}, []int64{3}, 5, 3},
		{"largest values", []int64{math.MaxInt64}, []int64{1}, math.MaxInt64, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := household.Fed(tt.consumed, tt.per, tt.weight); got != tt.want {
				t.Errorf("Fed(%v, %v, %d) = %d, want %d", tt.consumed, tt.per, tt.weight, got, tt.want)
			}
		})
	}
}

// TestNextHungerAndPrivation verifies the two counters of AGT 5.4.1: hunger
// counts consecutive unfed ticks and resets on a fed tick, privation climbs
// with every unfed tick and leaks by one on a fed tick, and both saturate.
//
// Requirement: FR-AGT-075
func TestNextHungerAndPrivation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		before, unfed   int64
		hunger, private int64
	}{
		{"unfed from zero", 0, 3, 1, 1},
		{"unfed climbs", 4, 1, 5, 5},
		{"fed resets hunger and leaks privation", 4, 0, 0, 3},
		{"fed from zero stays at zero", 0, 0, 0, 0},
		{"a negative unfed counts as fed", 2, -1, 0, 1},
		{"saturates at the maximum", math.MaxInt64, 1, math.MaxInt64, math.MaxInt64},
		{"a negative counter is floored when fed", math.MinInt64, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := household.NextHunger(tt.before, tt.unfed); got != tt.hunger {
				t.Errorf("NextHunger(%d, %d) = %d, want %d", tt.before, tt.unfed, got, tt.hunger)
			}
			if got := household.NextPrivation(tt.before, tt.unfed); got != tt.private {
				t.Errorf("NextPrivation(%d, %d) = %d, want %d", tt.before, tt.unfed, got, tt.private)
			}
		})
	}
}

// TestNextHungerAndPrivation_HungerNeverExceedsPrivation verifies, over every
// sequence of 12 fed and unfed ticks from zero, that Hunger <= Privation at
// each tick.
//
// Requirement: FR-AGT-075
func TestNextHungerAndPrivation_HungerNeverExceedsPrivation(t *testing.T) {
	t.Parallel()
	for seq := 0; seq < 1<<12; seq++ {
		var hunger, priv int64
		for i := 0; i < 12; i++ {
			unfed := int64(seq >> i & 1)
			hunger, priv = household.NextHunger(hunger, unfed), household.NextPrivation(priv, unfed)
			if hunger > priv {
				t.Fatalf("sequence %012b tick %d: Hunger %d > Privation %d", seq, i, hunger, priv)
			}
		}
	}
}

// TestCost_AccumulatesUpkeepCost verifies the upkeep cost of AGT 16.1
// (3 * 300 + 1 * 100 = 1 000), the reserve money (3 * 1 000), and that an
// overflow is sticky.
//
// Requirement: FR-AGT-038
func TestCost_AccumulatesUpkeepCost(t *testing.T) {
	t.Parallel()
	var c household.Cost
	if v, ok := c.Value(); v != 0 || !ok {
		t.Errorf("empty Cost.Value() = %d, %v, want 0, true", v, ok)
	}
	c.Add(3, 300)
	c.Add(1, 100)
	v, ok := c.Value()
	if v != 1000 || !ok {
		t.Fatalf("Cost.Value() = %d, %v, want 1000, true", v, ok)
	}
	if r, ok := household.ReserveMoney(v, 3); r != 3000 || !ok {
		t.Errorf("ReserveMoney(1000, 3) = %d, %v, want 3000, true", r, ok)
	}
	if r, ok := household.ReserveMoney(math.MaxInt64, 2); ok {
		t.Errorf("ReserveMoney(max, 2) = %d, true, want ok false", r)
	}

	for name, pair := range map[string][2]int64{
		"product": {math.MaxInt64, 2},
		"sum":     {1, math.MaxInt64},
	} {
		var o household.Cost
		o.Add(math.MaxInt64, 1)
		o.Add(pair[0], pair[1])
		o.Add(1, 1) // after the overflow, a further Add changes nothing
		if v, ok := o.Value(); ok || v != 0 {
			t.Errorf("%s overflow: Value() = %d, %v, want 0, false", name, v, ok)
		}
	}
}

// TestConsume_AllocatesNothing verifies that the consumption functions
// allocate nothing.
//
// Requirement: NFR-AGT-001
func TestConsume_AllocatesNothing(t *testing.T) {
	consumed, per := []int64{100, 40}, []int64{3, 1}
	var sink int64
	allocs := testing.AllocsPerRun(100, func() {
		var c household.Cost
		c.Add(3, 300)
		v, _ := c.Value()
		r, _ := household.ReserveMoney(v, 3)
		sink += household.Fed(consumed, per, 40) + household.NextHunger(1, 1) + household.NextPrivation(1, 0) + r
	})
	if allocs != 0 {
		t.Errorf("consumption functions allocated %v times per run, want 0 (sink %d)", allocs, sink)
	}
}
