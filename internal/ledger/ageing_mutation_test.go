package ledger

import (
	"errors"
	"math"
	"testing"
)

// spoilBoundary is the ID of commodity c's Spoilage boundary account.
func spoilBoundary(l *Ledger, c int) int {
	return l.boundary[l.off[Spoilage][thingGoods]+c]
}

// TestAge_SumPassRefusesWhatWouldOverflowBeforeMoving verifies that when the
// expired units of one (region, commodity) across accounts do not fit int64,
// Age returns ErrBalanceOverflow and moves nothing.
//
// Requirement: FR-LED-006, FR-LED-007
func TestAge_SumPassRefusesWhatWouldOverflowBeforeMoving(t *testing.T) {
	t.Parallel()
	l, agents := iledger(t)
	other, err := l.Open(KindAgent, 2, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Both lots expired at tick -5 and sit in region 0: MaxInt64 + 1 overflows.
	a0, a1 := &l.accounts[agents[0]], &l.accounts[other]
	a0.lots = []Lot{{Commodity: 0, Quantity: math.MaxInt64, Born: -7}}
	a1.lots = []Lot{{Commodity: 0, Quantity: 1, Born: -7}}
	a0.hint, a0.hasHint = -5, true
	a1.hint, a1.hasHint = -5, true
	if got := l.Age(); !errors.Is(got, ErrBalanceOverflow) {
		t.Fatalf("Age() = %v, want ErrBalanceOverflow", got)
	}
	if len(a0.lots) != 1 || len(a1.lots) != 1 {
		t.Errorf("lots after refused Age = %v, %v, want both untouched", a0.lots, a1.lots)
	}
}

// TestAge_SumPassReadsOnlyDueAccountsAndExpiredLots verifies, through the
// overflow check that the sum pass feeds, that it counts the expired lots of
// the accounts whose hint is reached and nothing else: not an account with a
// hint in the future or with no hint, and not a lot that has not expired.
//
// Guard: the sum pass must agree with the move pass on which lots are due, or
// a refusal is raised for units that would never move (or missed for ones that
// would). Each case sits the Spoilage balance of commodity 0 at the edge.
func TestAge_SumPassReadsOnlyDueAccountsAndExpiredLots(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		hint     int64
		hasHint  bool
		lots     []Lot
		headroom int64 // units the Spoilage balance can still take
		wantErr  error
	}{
		// Expired lot of 2 but the balance has room for 1: refused.
		{"due account overflows", -5, true, []Lot{{Commodity: 0, Quantity: 2, Born: -7}}, 1, ErrBalanceOverflow},
		// The same lot behind a hint in the future is not read.
		{"future hint is not read", 1000, true, []Lot{{Commodity: 0, Quantity: 2, Born: -7}}, 1, nil},
		// The same lot with no hint is not read.
		{"no hint is not read", 0, false, []Lot{{Commodity: 0, Quantity: 2, Born: -7}}, 1, nil},
		// Expired 2 fits the room of 2; the unexpired 5 (born 0, expires at 2) must not be counted.
		{"unexpired lot is not counted", -5, true, []Lot{{Commodity: 0, Quantity: 2, Born: -7}, {Commodity: 0, Quantity: 5, Born: 0}}, 2, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l, agents := iledger(t)
			a := &l.accounts[agents[0]]
			a.lots = append([]Lot(nil), tt.lots...)
			a.hint, a.hasHint = tt.hint, tt.hasHint
			l.accounts[spoilBoundary(l, 0)].balance = math.MaxInt64 - tt.headroom
			if got := l.Age(); !errors.Is(got, tt.wantErr) {
				t.Errorf("Age() = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

// TestAge_OnlyTheCommodityThatSpoilsNeedsABoundary verifies that a commodity
// with nothing expired needs no Spoilage boundary, and that a boundary account
// whose ID is 0 is a valid one.
//
// Guard: a commodity with no expired unit must not demand a Spoilage boundary,
// and account ID 0 must not be read as "no account".
func TestAge_OnlyTheCommodityThatSpoilsNeedsABoundary(t *testing.T) {
	t.Parallel()
	l, err := New(Config{
		Commodities: []Commodity{{ShelfLife: 3}, {ShelfLife: 3}},
		Regions:     1,
		MaxMoney:    1000,
		MaxQuantity: 100,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// The Spoilage boundary of commodity 0 is the first account: ID 0. Commodity
	// 1 has none, and nothing of it has expired.
	b, err := l.OpenBoundary(Spoilage, 0)
	if err != nil || b != 0 {
		t.Fatalf("OpenBoundary = %d, %v, want 0, nil", b, err)
	}
	id, err := l.Open(KindAgent, 0, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	a := &l.accounts[id]
	a.lots = []Lot{{Commodity: 0, Quantity: 4, Born: -7}}
	a.hint, a.hasHint = -5, true
	if got := l.Age(); got != nil {
		t.Fatalf("Age() = %v, want nil", got)
	}
	if got := l.accounts[b].balance; got != 4 {
		t.Errorf("Spoilage balance = %d, want 4", got)
	}
}

// TestExpiring_NoHintMeansNothingExpires verifies that an account without a
// hint reports 0 even if a lot is planted that would be due, and that the same
// lot with a reached hint is reported.
//
// Requirement: FR-LED-049
func TestExpiring_NoHintMeansNothingExpires(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		hasHint bool
		want    int64
	}{
		{"no hint", false, 0},
		{"hint reached", true, 3}, // 3 units, none reserved
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l, agents := iledger(t)
			a := &l.accounts[agents[0]]
			a.lots = []Lot{{Commodity: 0, Quantity: 3, Born: -7}}
			a.hint, a.hasHint = -5, tt.hasHint
			got, err := l.Expiring(agents[0], 0, 0)
			if err != nil {
				t.Fatalf("Expiring() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Expiring(agent, 0, 0) = %d, want %d", got, tt.want)
			}
		})
	}
}
