package ledger

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
)

// wantRowCfg is a one-region, three-commodity ledger configuration with the
// given declared reasons.
func wantRowCfg(declared ...Reason) Config {
	return Config{
		Commodities: []Commodity{{}, {}, {}}, Regions: 1,
		MaxMoney: 1000, MaxQuantity: 100, Declared: declared, MaxClaims: 2,
	}
}

// TestLedger_DeclaredBitReplacesThreshold verifies that a reason is declarable
// exactly when its registry row carries the declared bit, that every other
// reason is refused with ErrInvalidConfig, and that the three loan reasons
// still have a flow slot only when they are listed, as they did under the
// threshold.
//
// Requirement: FR-LED-050, FR-LED-052
func TestLedger_DeclaredBitReplacesThreshold(t *testing.T) {
	t.Parallel()
	declarable := []Reason{LoanOut, LoanRepay, LoanInterest, WantConsumption, Relief}
	for r := Reason(0); r <= NumReasons; r++ {
		want := slices.Contains(declarable, r)
		t.Run(fmt.Sprintf("reason %d", r), func(t *testing.T) {
			t.Parallel()
			if got := isDeclared(r); got != want {
				t.Errorf("isDeclared(%d) = %v, want %v", r, got, want)
			}
			_, err := New(wantRowCfg(r))
			if got := err == nil; got != want {
				t.Errorf("New(Declared: [%d]) error = %v, want success %v", r, err, want)
			}
			if err != nil && !errors.Is(err, ErrInvalidConfig) {
				t.Errorf("New(Declared: [%d]) error = %v, want ErrInvalidConfig", r, err)
			}
		})
	}

	// A declared row has a slot only when listed; a built-in one always has.
	tests := []struct {
		name     string
		declared []Reason
		r        Reason
		commod   int
		want     bool
	}{
		{"loan listed", []Reason{LoanOut}, LoanOut, Money, true},
		{"loan not listed", nil, LoanOut, Money, false},
		{"other loan not listed", []Reason{LoanOut}, LoanRepay, Money, false},
		{"want listed", []Reason{WantConsumption}, WantConsumption, 1, true},
		{"want not listed", []Reason{LoanOut}, WantConsumption, 1, false},
		{"want has no money row", []Reason{WantConsumption}, WantConsumption, Money, false},
		{"built-in without a list", nil, Consumption, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l, err := New(wantRowCfg(tt.declared...))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			_, err = l.slotOf(tt.r, tt.commod)
			if got := err == nil; got != tt.want {
				t.Errorf("slotOf(%d, %d) error = %v, want a slot %v", tt.r, tt.commod, err, tt.want)
			}
		})
	}
	if _, err := New(wantRowCfg(WantConsumption, WantConsumption)); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("New(want_consumption twice) error = %v, want ErrInvalidConfig", err)
	}
}

// TestRank_DeclaredRowsFollowTheTable verifies that DeclaredByRank lists the
// declared reasons in the order of the table of spec-ledger.md 7.2 and not in
// the order of their numbers, that Rank agrees with the pinned code of the row,
// and that a built-in or unknown reason has no rank.
//
// Requirement: FR-LED-050
func TestRank_DeclaredRowsFollowTheTable(t *testing.T) {
	t.Parallel()
	got := DeclaredByRank(nil)
	// want_consumption is row 6 of the table, relief row 12, the loan rows 25 to
	// 27; the numbers of the first two are the highest, so the two orders differ.
	want := []Reason{WantConsumption, Relief, LoanOut, LoanRepay, LoanInterest}
	if !slices.Equal(got, want) {
		t.Errorf("DeclaredByRank(nil) = %v, want %v", got, want)
	}
	prefix := []Reason{Trade}
	if got := DeclaredByRank(prefix); len(got) != 1+len(want) || got[0] != Trade {
		t.Errorf("DeclaredByRank([Trade]) = %v, want Trade then %v", got, want)
	}
	for _, r := range want {
		rank, ok := Rank(r)
		if !ok || uint64(rank) != reasonCode(r) {
			t.Errorf("Rank(%d) = %d, %v, want the pinned code %d", r, rank, ok, reasonCode(r))
		}
	}
	for _, r := range []Reason{Trade, Consumption, Starvation, NumReasons, NumReasons + 1} {
		if rank, ok := Rank(r); ok || rank != 0 {
			t.Errorf("Rank(%d) = %d, %v, want 0, false", r, rank, ok)
		}
	}
	// Only a declared row has a rank: a built-in row's column stays 0.
	for r := Reason(0); r < NumReasons; r++ {
		for th, rw := range table[r] {
			if !rw.declared && rw.rank != 0 {
				t.Errorf("table[%d][%d].rank = %d on a built-in row, want 0", r, th, rw.rank)
			}
		}
	}
}

// TestNew_NoDeclaredReasonLeavesLayoutUnchanged verifies that a ledger built
// without a reason past the Stage 1 size has the Stage 1 layout: the fixed
// offset array keeps the Stage 1 length, the slice of later offsets is nil, and
// the stride is the count of the built-in slots; and that declaring the loan
// reasons leaves the slice nil too, while declaring want_consumption makes it
// and adds one slot per commodity.
//
// Guard: the position hash walks the arrays of the ledger. A reason appended to
// the enum must not lengthen any of them while it is undeclared, or every golden
// of a world with the module off would move.
func TestNew_NoDeclaredReasonLeavesLayoutUnchanged(t *testing.T) {
	t.Parallel()
	if got := reflect.TypeOf(Ledger{}.off).Len(); got != numStage1Reasons || numStage1Reasons != 18 {
		t.Errorf("len(Ledger.off) = %d, numStage1Reasons = %d, want both 18", got, numStage1Reasons)
	}
	// Built-in slots: money for trade, initial, estate_money, start_money, split,
	// merge, births, starvation (8); goods for trade, initial, recipe_input,
	// recipe_output, disposal, consumption, spoilage, wear, estate_goods, split,
	// merge, starvation (12 per commodity). Three commodities: 8 + 36 = 44.
	const builtIn = 8 + 12*3
	tests := []struct {
		name       string
		declared   []Reason
		wantStride int
		wantLater  bool
	}{
		{"no module", nil, builtIn, false},
		{"loan reasons", []Reason{LoanOut, LoanRepay, LoanInterest}, builtIn + 3, false},
		{"want_consumption", []Reason{WantConsumption}, builtIn + 3, true},
		{"loans and want_consumption", []Reason{LoanOut, WantConsumption}, builtIn + 1 + 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l, err := New(wantRowCfg(tt.declared...))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if l.stride != tt.wantStride || len(l.base) != tt.wantStride || len(l.boundary) != tt.wantStride {
				t.Errorf("New(%v) stride = %d, base %d, boundary %d, want %d", tt.declared, l.stride, len(l.base), len(l.boundary), tt.wantStride)
			}
			if got := l.later != nil; got != tt.wantLater {
				t.Errorf("New(%v) later = %v, want non-nil %v", tt.declared, l.later, tt.wantLater)
			}
			if got := l.offOf(NumReasons, thingGoods); got != -1 {
				t.Errorf("offOf(NumReasons, goods) = %d, want -1", got)
			}
		})
	}
}
