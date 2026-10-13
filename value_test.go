package economysim_test

import (
	"fmt"
	"math"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestMoney_AddSub verifies checked addition and subtraction of money at zero,
// at negative values and at the int64 bounds, where a wrap would be silent.
//
// Requirement: FR-CUR-001, FR-CUR-003
func TestMoney_AddSub(t *testing.T) {
	t.Parallel()
	const max, min = economysim.Money(math.MaxInt64), economysim.Money(math.MinInt64)
	tests := []struct {
		name          string
		a, b          economysim.Money
		sum, diff     economysim.Money
		sumOK, diffOK bool
	}{
		{"zero", 0, 0, 0, 0, true, true},
		{"small", 5, 3, 8, 2, true, true},
		{"negative", -5, 3, -2, -8, true, true},
		{"max plus one", max, 1, 0, max - 1, false, true},
		{"max minus negative one", max, -1, max - 1, 0, true, false},
		{"min minus one", min, 1, min + 1, 0, true, false},
		{"min plus negative one", min, -1, 0, min + 1, false, true},
		{"max plus min", max, min, -1, 0, true, false},
		{"zero minus min", 0, min, min, 0, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tt.a.Add(tt.b); ok != tt.sumOK || (ok && got != tt.sum) || (!ok && got != 0) {
				t.Errorf("Money(%d).Add(%d) = %d, %v, want %d, %v", tt.a, tt.b, got, ok, tt.sum, tt.sumOK)
			}
			if got, ok := tt.a.Sub(tt.b); ok != tt.diffOK || (ok && got != tt.diff) || (!ok && got != 0) {
				t.Errorf("Money(%d).Sub(%d) = %d, %v, want %d, %v", tt.a, tt.b, got, ok, tt.diff, tt.diffOK)
			}
		})
	}
}

// TestQuantity_AddSub verifies checked addition and subtraction of quantities
// at the int64 bounds.
//
// Requirement: FR-CUR-003
func TestQuantity_AddSub(t *testing.T) {
	t.Parallel()
	const max, min = economysim.Quantity(math.MaxInt64), economysim.Quantity(math.MinInt64)
	tests := []struct {
		name          string
		a, b          economysim.Quantity
		sum, diff     economysim.Quantity
		sumOK, diffOK bool
	}{
		{"small", 7, 4, 11, 3, true, true},
		{"below zero", 4, 7, 11, -3, true, true},
		{"max plus one", max, 1, 0, max - 1, false, true},
		{"min minus one", min, 1, min + 1, 0, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tt.a.Add(tt.b); ok != tt.sumOK || got != tt.sum {
				t.Errorf("Quantity(%d).Add(%d) = %d, %v, want %d, %v", tt.a, tt.b, got, ok, tt.sum, tt.sumOK)
			}
			if got, ok := tt.a.Sub(tt.b); ok != tt.diffOK || got != tt.diff {
				t.Errorf("Quantity(%d).Sub(%d) = %d, %v, want %d, %v", tt.a, tt.b, got, ok, tt.diff, tt.diffOK)
			}
		})
	}
}

// TestPrice_Times verifies that price times quantity is exact when it fits in
// int64 and reports overflow, with a zero result, when it does not.
//
// Requirement: FR-CUR-001, FR-CUR-003
func TestPrice_Times(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		p    economysim.Price
		q    economysim.Quantity
		want economysim.Money
		ok   bool
	}{
		{"zero quantity", 120, 0, 0, true},
		{"ordinary", 120, 25, 3000, true}, // 120 * 25
		{"negative quantity", 10, -3, -30, true},
		{"largest exact", math.MaxInt32, math.MaxInt32, math.MaxInt32 * math.MaxInt32, true},
		{"just over", math.MaxInt64/2 + 1, 2, 0, false},
		{"min times minus one", math.MinInt64, -1, 0, false},
		{"max times one", math.MaxInt64, 1, math.MaxInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tt.p.Times(tt.q); ok != tt.ok || got != tt.want {
				t.Errorf("Price(%d).Times(%d) = %d, %v, want %d, %v", tt.p, tt.q, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestValueTypes_Formatting verifies that Money, Price and Quantity format as
// a plain decimal count, with no denomination name or symbol, for both
// Stringer and the verbs of fmt.
//
// Requirement: FR-CUR-001
func TestValueTypes_Formatting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"money zero", economysim.Money(0).String(), "0"},
		{"money positive", economysim.Money(1460).String(), "1460"},
		{"money negative", economysim.Money(-2920).String(), "-2920"},
		{"money min", economysim.Money(math.MinInt64).String(), "-9223372036854775808"},
		{"price", economysim.Price(50).String(), "50"},
		{"quantity max", economysim.Quantity(math.MaxInt64).String(), "9223372036854775807"},
		{"money in Sprint", fmt.Sprint(economysim.Money(12345)), "12345"},
		{"price in Sprintf %d", fmt.Sprintf("%d", economysim.Price(7)), "7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Errorf("formatting %s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestIDs_Sentinels verifies the values of the "no" sentinels: -1, below
// every valid ID, so a lookup by it is outside [0, n).
//
// Guard: spec-integration.md 4 and spec-regions.md 3.3 rely on a sentinel that
// no index can equal; a zero value would collide with the first region.
func TestIDs_Sentinels(t *testing.T) {
	t.Parallel()
	if economysim.NoCommodity != -1 || economysim.NoRegion != -1 || economysim.NoAgent != -1 {
		t.Errorf("NoCommodity, NoRegion, NoAgent = %d, %d, %d, want -1 each",
			economysim.NoCommodity, economysim.NoRegion, economysim.NoAgent)
	}
	var zeroRegion economysim.RegionID
	var zeroAgent economysim.AgentID
	if zeroRegion == economysim.NoRegion || zeroAgent == economysim.NoAgent {
		t.Errorf("zero RegionID or AgentID equals its sentinel")
	}
}
