package economysim

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// Equivalent mutants of ledger.go that no test can kill (gomu, one-line why each):
//   - reasonName `reasonNames[r] == ""` -> `<=` is the same test on a string;
//     -> `<` is never true, which differs only for an in-range empty row, and
//     the table has none below LoanOut (TestReasonNames_RoundTrip).
//   - reasonName and ledgerReason `return "", false` / `return 0, false`
//     (lines 25, 60): the replacement is the same zero literal.
//   - newLedger `err != nil -> false` and `return nil, err -> nil` (lines 89,
//     90): openBuiltInBoundaries cannot fail on the fresh ledger newLedger has
//     just built, because the count it walks is the table's own length.
//   - openBuiltInBoundaries `thing != Money -> >` (105): thing starts at Money,
//     so it is never below it. `r < numBuiltInReasons -> <=` and `-> !=` (106):
//     r starts at 0 and the row at numBuiltInReasons (LoanOut) is Internal for
//     money and absent for goods, so it is skipped either way.
//     `class != Internal -> >` (107): Class is a uint8, so != 0 is > 0.
//   - boundaryFlows `class == Internal -> <=` (132): same uint8 reason.
//     `class == Sink -> >=` (140): Sink is the largest Class, and Internal is
//     already skipped. `if goods -> true` (143): a money flow has Commodity
//     == Money == -1 == NoCommodity, so CommodityID(f.Commodity) is the same.
//   - boundaryFlows `!ok -> false` (136): the panic needs a faucet or a sink
//     with no Reason name, and every faucet and sink of the table has one.
//   - wearApplied `units > 0 -> !=` (170): the ledger's wear counters only
//     grow, so units is never negative.

// TestOpenBuiltInBoundaries_RefusesDuplicate verifies that a second walk over
// a ledger that already holds the built-in boundary accounts returns the
// ledger's duplicate error, wrapped with the reason and thing it was opening.
//
// Guard: both the closure and the loop must pass the error up, so a swallowed
// error would leave a half-opened group that the caller believed complete.
//
// Requirement: FR-LED-052
func TestOpenBuiltInBoundaries_RefusesDuplicate(t *testing.T) {
	t.Parallel()
	l := mustLedger(t, Config{Commodities: []Commodity{{ID: "a"}}, Regions: []Region{{ID: "r"}}})
	err := openBuiltInBoundaries(l, 1)
	if !errors.Is(err, ledger.ErrDuplicateBoundary) {
		t.Fatalf("second openBuiltInBoundaries error = %v, want ErrDuplicateBoundary", err)
	}
	// The first row in the order is the initial money (reason 1, thing Money).
	want := fmt.Sprintf("open boundary account %d of thing %d:", ledger.Initial, ledger.Money)
	if !strings.HasPrefix(err.Error(), want) {
		t.Errorf("error = %q, want prefix %q", err.Error(), want)
	}
}

// TestOpenBuiltInBoundaries_NegativeCountOpensNothing verifies that a
// commodity count below zero opens no account, not even the money rows.
//
// Guard: the loop bound is a less-than, so a count that is never reached by
// counting up must not run the loop at all.
//
// Requirement: FR-LED-052
func TestOpenBuiltInBoundaries_NegativeCountOpensNothing(t *testing.T) {
	t.Parallel()
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{}},
		Regions:     1,
		MaxMoney:    MaxMoney,
		MaxQuantity: MaxQuantity,
	})
	if err != nil {
		t.Fatalf("ledger.New: %v", err)
	}
	if err := openBuiltInBoundaries(l, -2); err != nil {
		t.Fatalf("openBuiltInBoundaries(-2) error = %v, want none", err)
	}
	if id, ok := l.Boundary(ledger.Initial, ledger.Money); ok {
		t.Errorf("Boundary(Initial, Money) = %d, true, want no account", id)
	}
}

// TestBoundaryFlows_MoneyIsReadFromTheCommodity verifies that the class lookup
// is by money (the commodity is Money) or goods (any other index): a money
// flow of a goods-only reason has no row and is not reported, and an index
// other than Money is looked up as goods.
//
// Guard: reading the test as an ordering (>, >=) instead of an inequality
// would flip one of the two cases.
//
// Requirement: FR-LED-031
func TestBoundaryFlows_MoneyIsReadFromTheCommodity(t *testing.T) {
	t.Parallel()
	money := []ledger.Flow{{Region: 0, Reason: ledger.RecipeInput, Commodity: ledger.Money, Total: 5}}
	if got := boundaryFlows(nil, money, 0, nil); len(got) != 0 {
		t.Errorf("boundaryFlows(money flow of a goods-only reason, nil) = %+v, want none", got)
	}
	goods := []ledger.Flow{{Region: 0, Reason: ledger.RecipeInput, Commodity: -2, Total: 5}}
	want := []BoundaryFlow{{Tick: 3, Region: 0, Reason: ReasonRecipeInput, Kind: KindSink, Commodity: -2, Total: 5}}
	got := boundaryFlows(nil, goods, 3, nil)
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("boundaryFlows(index below Money, nil) = %+v, want %+v", got, want)
	}
}

// TestWearApplied_NegativeCountReportsNothing verifies that a commodity count
// below zero reports no row and does not walk the table.
//
// Guard: the loop bound is a less-than, so a count that counting up never
// reaches must not run the loop, which would panic past the table's end.
//
// Requirement: FR-LED-013
func TestWearApplied_NegativeCountReportsNothing(t *testing.T) {
	t.Parallel()
	l := mustLedger(t, Config{Commodities: []Commodity{{ID: "a"}, {ID: "b"}}, Regions: []Region{{ID: "r"}}})
	if got := wearApplied(nil, l, -1, 0, false); len(got) != 0 {
		t.Errorf("wearApplied(count -1) = %+v, want none", got)
	}
}

// TestWearApplied_UnknownCommodityPanics verifies the programmer-error panic
// of a count above the ledger's table: its value names the commodity and
// carries the ledger's error.
//
// Requirement: FR-LED-013
func TestWearApplied_UnknownCommodityPanics(t *testing.T) {
	t.Parallel()
	l := mustLedger(t, Config{Commodities: []Commodity{{ID: "a"}, {ID: "b"}}, Regions: []Region{{ID: "r"}}})
	want := "economysim: invariant violated (programmer error): wear units of commodity 2: " + ledger.ErrUnknownCommodity.Error()
	defer func() {
		if got := recover(); got != want {
			t.Errorf("wearApplied panic = %#v, want %q", got, want)
		}
	}()
	wearApplied(nil, l, 3, 0, false)
}
