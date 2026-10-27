package economysim

import (
	"math"
	"slices"
	"testing"
)

// TestReservedReasons_CoverTheLedgerNames verifies that every reason name the
// world knows (reasonNames) is in reservedReasons, that the list has no
// duplicate, and that every name is a valid ID, so a flow cannot take the name
// of a reason (spec-money-supply.md 4.5.2, plan 2-6 D325).
//
// Requirement: FR-MON-036
//
// Guard: a module that declares a reason must add its name to reservedReasons,
// or a flow ID could collide with it and two reasons would share one BoundaryFlow
// row. Without this test the omission shows only when a pack names a flow so.
func TestReservedReasons_CoverTheLedgerNames(t *testing.T) {
	t.Parallel()
	for _, name := range reasonNames {
		if name != "" && !slices.Contains(reservedReasons, string(name)) {
			t.Errorf("reason %q of reasonNames is not in reservedReasons", name)
		}
	}
	seen := map[string]bool{}
	for _, name := range reservedReasons {
		if seen[name] {
			t.Errorf("reservedReasons lists %q twice", name)
		}
		seen[name] = true
		if !validID(name) {
			t.Errorf("reservedReasons entry %q is not a valid ID", name)
		}
	}
}

// TestMoneySupply_CanonicalIsACopy verifies that canonical sorts the flows by ID,
// spells an empty period as a tick, returns nil for a module that is off and
// leaves its receiver unchanged.
//
// Requirement: NFR-MON-003
func TestMoneySupply_CanonicalIsACopy(t *testing.T) {
	t.Parallel()
	var off *MoneySupply
	if off.canonical() != nil {
		t.Error("canonical of a nil module is not nil")
	}
	m := &MoneySupply{Flows: []MoneyFlow{{ID: "b"}, {ID: "a", Per: PerYear}}}
	got := m.canonical()
	if got.Flows[0].ID != "a" || got.Flows[0].Per != PerYear || got.Flows[1].ID != "b" || got.Flows[1].Per != PerTick {
		t.Errorf("canonical flows = %+v, want a (year) then b (tick)", got.Flows)
	}
	if m.Flows[0].ID != "b" || m.Flows[1].Per != PerYear || m.Flows[0].Per != "" {
		t.Errorf("canonical changed its receiver: %+v", m.Flows)
	}
}

// TestPairUnbalanced_InvalidConfigIsNotComparable verifies that Warnings, which
// runs on a configuration that does not validate, never raises pair_unbalanced
// from a wrapped or meaningless total: a household count that overflows, a
// negative amount and a calendar with no year are not comparable.
//
// Requirement: FR-MON-039
//
// Guard: an unchecked sum of households wrapped negative, and a calendar with no
// year gave a per-tick total of 0, so either made the pair look unbalanced.
func TestPairUnbalanced_InvalidConfigIsNotComparable(t *testing.T) {
	t.Parallel()
	target := FlowTarget{Kind: TargetHouseholds, Role: "r"}
	pair := func(grant, levy int64) *MoneySupply {
		return &MoneySupply{Flows: []MoneyFlow{
			{ID: "g", Kind: FlowGrant, Target: target, Amount: grant, Per: PerYear},
			{ID: "l", Kind: FlowLevy, Target: target, Amount: levy, Per: PerYear},
		}}
	}
	tests := []struct {
		name   string
		groups []Group
		ms     *MoneySupply
		want   bool
	}{
		{"unbalanced and valid", []Group{{Role: "r", Households: 10}}, pair(100, 1), true},
		{"balanced", []Group{{Role: "r", Households: 10}}, pair(5, 5), false},
		{"household sum overflows", []Group{{Role: "r", Households: math.MaxInt64}, {Role: "r", Households: 1}}, pair(100, 1), false},
		{"negative households", []Group{{Role: "r", Households: -5}}, pair(100, 1), false},
		{"negative amount", []Group{{Role: "r", Households: 10}}, pair(-100, 1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Config{Groups: tt.groups}
			if got := pairUnbalanced(c, tt.ms); got != tt.want {
				t.Errorf("pairUnbalanced = %v, want %v", got, tt.want)
			}
		})
	}
	// A per-tick pair on a zero calendar has no year to multiply by.
	c := &Config{Groups: []Group{{Role: "r", Households: 10}}}
	ms := &MoneySupply{Flows: []MoneyFlow{
		{ID: "g", Kind: FlowGrant, Target: target, Amount: 100},
		{ID: "l", Kind: FlowLevy, Target: target, Amount: 1},
	}}
	if pairUnbalanced(c, ms) {
		t.Error("pairUnbalanced on a calendar with no year = true, want false")
	}
}

// TestAvailableAbove_TakesOnlyWhatIsHeldAboveTheExemption verifies the balance
// a levy may take: the balance less weight*exempt, never negative, and nothing
// when the product does not fit in an int64.
//
// Requirement: FR-MON-015, FR-MON-020
//
// Guard: an overflowed product must leave nothing to take, not wrap into a
// negative exemption that would let a levy take more than the account holds.
func TestAvailableAbove_TakesOnlyWhatIsHeldAboveTheExemption(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		bal, weight, exempt int64
		want                int64
	}{
		{"balance above the exemption", 100, 3, 10, 70},  // 100 - 3*10
		{"balance equal to the exemption", 30, 3, 10, 0}, // 30 - 30
		{"balance below the exemption", 5, 3, 10, 0},     // 5 - 30 < 0
		{"no exemption", 7, 3, 0, 7},
		{"product overflows", 100, math.MaxInt64, 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := availableAbove(tt.bal, tt.weight, tt.exempt); got != tt.want {
				t.Errorf("availableAbove(%d, %d, %d) = %d, want %d", tt.bal, tt.weight, tt.exempt, got, tt.want)
			}
		})
	}
}

// TestBeyondTolerance_PairTotals verifies the balanced-pair comparison: totals
// that differ by more than pairTolerance basis points of the larger are beyond
// it, equal or close ones are not, and a negative total (an input that did not
// fit) or an overflowing product is never reported.
//
// Requirement: FR-MON-039
func TestBeyondTolerance_PairTotals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b int64
		want bool
	}{
		{"equal", 1000, 1000, false},
		{"exactly at the tolerance", 10_000, 9_000, false}, // 1000 of 10000 is 1000 bp
		{"one past the tolerance", 10_000, 8_999, true},
		{"order does not matter", 8_999, 10_000, true},
		{"negative a", -1, 10_000, false},
		{"negative b", 10_000, -1, false},
		{"product overflows", math.MaxInt64, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := beyondTolerance(tt.a, tt.b); got != tt.want {
				t.Errorf("beyondTolerance(%d, %d) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
