package ledger_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestNew_RefusesOutOfRangeAndOversizedTables verifies that New refuses a
// declared reason beyond the last reason, and a table whose flow totals do not
// fit in memory, instead of panicking.
//
// Guard: the bound checks of New are exact, and never index past a table.
func TestNew_RefusesOutOfRangeAndOversizedTables(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  ledger.Config
	}{
		{name: "reason just past the last", cfg: ledger.Config{Regions: 1, MaxMoney: 1, MaxQuantity: 1, Declared: []ledger.Reason{ledger.NumReasons}}},
		{name: "reason far past the last", cfg: ledger.Config{Regions: 1, MaxMoney: 1, MaxQuantity: 1, Declared: []ledger.Reason{ledger.NumReasons + 1}}},
		// Two or more slots per region times MaxInt regions overflows int64.
		{name: "flow totals overflow", cfg: ledger.Config{Regions: math.MaxInt, MaxMoney: 1, MaxQuantity: 1, Commodities: []ledger.Commodity{{}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ledger.New(tt.cfg); !errors.Is(err, ledger.ErrInvalidConfig) {
				t.Errorf("New error = %v, want ErrInvalidConfig", err)
			}
		})
	}
}

// TestAccount_ViewAndLookups verifies the account view (an unknown ID is an
// error, a non-boundary keeps its region, a boundary reports -1), that Open
// refuses a region past the last, and that Boundary reports false for a reason
// or commodity that has no slot.
//
// Requirement: FR-LED-008
func TestAccount_ViewAndLookups(t *testing.T) {
	t.Parallel()
	l, err := ledger.New(ledger.Config{Regions: 2, MaxMoney: 100, MaxQuantity: 100, Commodities: []ledger.Commodity{{}}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	b, err := l.OpenBoundary(ledger.Initial, ledger.Money)
	if err != nil {
		t.Fatalf("OpenBoundary: %v", err)
	}
	a, err := l.Open(ledger.KindAgent, 7, 1)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if _, err := l.Account(99); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Account(99) error = %v, want ErrUnknownAccount", err)
	}
	if v, err := l.Account(a); err != nil || v.Region != 1 || v.Owner != 7 {
		t.Errorf("Account(agent) = %+v, %v, want region 1 owner 7", v, err)
	}
	if v, err := l.Account(b); err != nil || v.Region != -1 {
		t.Errorf("Account(boundary) = %+v, %v, want region -1", v, err)
	}

	// Regions is 2, so 2 and 3 are both past the last region.
	for _, region := range []int{-1, 2, 3} {
		if _, err := l.Open(ledger.KindAgent, 0, region); !errors.Is(err, ledger.ErrUnknownRegion) {
			t.Errorf("Open(region %d) error = %v, want ErrUnknownRegion", region, err)
		}
	}

	tests := []struct {
		name      string
		r         ledger.Reason
		commodity int
	}{
		{name: "reason at NumReasons", r: ledger.NumReasons, commodity: ledger.Money},
		{name: "reason past NumReasons", r: ledger.NumReasons + 1, commodity: ledger.Money},
		{name: "goods-only reason, money", r: ledger.Consumption, commodity: ledger.Money},
		{name: "money-only reason, goods", r: ledger.EstateMoney, commodity: 0},
		{name: "commodity at the count", r: ledger.Initial, commodity: 1},
		{name: "commodity past the count", r: ledger.Initial, commodity: 2},
		{name: "commodity far past the count", r: ledger.Initial, commodity: 1000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if id, ok := l.Boundary(tt.r, tt.commodity); ok {
				t.Errorf("Boundary(%d, %d) = %d, true, want false", tt.r, tt.commodity, id)
			}
		})
	}
}

// TestOpenBoundary_DuplicateAtNonZeroID verifies that a second boundary for
// the same reason and thing is refused when the first is not account 0.
//
// Guard: the duplicate check tests for "has an ID", not "has ID 0".
func TestOpenBoundary_DuplicateAtNonZeroID(t *testing.T) {
	t.Parallel()
	l, err := ledger.New(ledger.Config{Regions: 1, MaxMoney: 100, MaxQuantity: 100})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := l.Open(ledger.KindAgent, 0, 0); err != nil { // takes ID 0
		t.Fatalf("Open: %v", err)
	}
	if id, err := l.OpenBoundary(ledger.Initial, ledger.Money); err != nil || id != 1 {
		t.Fatalf("OpenBoundary = %d, %v, want 1, nil", id, err)
	}
	if _, err := l.OpenBoundary(ledger.Initial, ledger.Money); !errors.Is(err, ledger.ErrDuplicateBoundary) {
		t.Errorf("second OpenBoundary error = %v, want ErrDuplicateBoundary", err)
	}
}
