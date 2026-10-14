package ledger

import (
	"math"
	"testing"
)

// TestWearTotals verifies the selection sums: items destroyed, durability
// removed, and a refusal (not a wrap) when either product or sum leaves int64.
//
// Guard: the totals feed the flow slots; a wrapped total would corrupt the
// conservation audit silently.
func TestWearTotals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		sel           []wearSel
		destroyed, rm int64
		ok            bool
	}{
		{"empty", nil, 0, 0, true},
		// 2*3 + 1*4 = 10 removed; only the destroyed selection counts its n (1).
		{"mixed", []wearSel{{n: 2, removed: 3}, {n: 1, removed: 4, destroyed: true}}, 1, 10, true},
		{"product overflow", []wearSel{{n: 2, removed: math.MaxInt64}}, 0, 0, false},
		{"sum overflow", []wearSel{{n: 1, removed: math.MaxInt64}, {n: 1, removed: 1}}, 0, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l := &Ledger{sel: tc.sel}
			d, r, ok := l.wearTotals()
			if d != tc.destroyed || r != tc.rm || ok != tc.ok {
				t.Errorf("wearTotals() = %d, %d, %v, want %d, %d, %v", d, r, ok, tc.destroyed, tc.rm, tc.ok)
			}
		})
	}
}

// TestClaimPartsValid verifies the bounds of a payment split.
//
// Guard: a negative or zero-total part, or one above what is owed, must be
// refused before any transfer is attempted.
func TestClaimPartsValid(t *testing.T) {
	t.Parallel()
	c := &Claim{InterestOwed: 5, PrincipalOwed: 10}
	tests := []struct {
		name         string
		interest, pr int64
		want         bool
	}{
		{"both zero", 0, 0, false},
		{"negative interest", -1, 1, false},
		{"negative principal", 1, -1, false},
		{"interest above owed", 6, 0, false},
		{"principal above owed", 0, 11, false},
		{"exactly owed", 5, 10, true},
		{"interest only", 1, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := c.partsValid(tc.interest, tc.pr); got != tc.want {
				t.Errorf("partsValid(%d, %d) = %v, want %v", tc.interest, tc.pr, got, tc.want)
			}
		})
	}
}

// TestFlowsFit verifies that a loan flow total at MaxInt64 refuses any
// further positive part and that a zero part is never checked.
//
// Guard: flow totals are checked, never wrapped (architecture.md 3.5).
func TestFlowsFit(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	l := g.l
	if !l.flowsFit(0, 1, 1) {
		t.Fatalf("flowsFit(0, 1, 1) = false on empty flows, want true")
	}
	l.flows()[l.off[LoanInterest][thingMoney]] = math.MaxInt64
	if l.flowsFit(0, 1, 0) {
		t.Errorf("flowsFit(0, 1, 0) = true with interest total at MaxInt64, want false")
	}
	if !l.flowsFit(0, 0, 1) {
		t.Errorf("flowsFit(0, 0, 1) = false, want true: only the principal slot is touched")
	}
	l.flows()[l.off[LoanRepay][thingMoney]] = math.MaxInt64
	if l.flowsFit(0, 0, 1) {
		t.Errorf("flowsFit(0, 0, 1) = true with repay total at MaxInt64, want false")
	}
}
