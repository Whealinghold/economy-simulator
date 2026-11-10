package ledger_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// outsideRow is one row of the table.
type outsideRow struct {
	name     string
	reason   ledger.Reason
	goods    bool
	class    ledger.Class
	from, to ledger.Kind
}

// outsideRows are the six rows of the external module (spec-ledger.md 7.2):
// the reason, the thing it moves, its class, and the kinds it runs between.
var outsideRows = []outsideRow{
	{"outside_budget", ledger.OutsideBudget, false, ledger.Faucet, ledger.KindBoundary, ledger.KindOutside},
	{"outside_unspent", ledger.OutsideUnspent, false, ledger.Sink, ledger.KindOutside, ledger.KindBoundary},
	{"outside_receipts", ledger.OutsideReceipts, false, ledger.Sink, ledger.KindOutside, ledger.KindBoundary},
	{"outside_offer", ledger.OutsideOffer, true, ledger.Faucet, ledger.KindBoundary, ledger.KindOutside},
	{"outside_unsold", ledger.OutsideUnsold, true, ledger.Sink, ledger.KindOutside, ledger.KindBoundary},
	{"outside_purchases", ledger.OutsidePurchases, true, ledger.Sink, ledger.KindOutside, ledger.KindBoundary},
}

// outsideLedger builds a one-region ledger with the six outside reasons
// declared, one spoiling and wearing commodity, and the boundary accounts of
// the six, plus one Outside account in region 0.
func outsideLedger(t *testing.T) (l *ledger.Ledger, out, budget, offer int) {
	t.Helper()
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{ShelfLife: 3, Durability: 5}},
		Regions:     1,
		MaxMoney:    1000,
		MaxQuantity: 100,
		Declared: []ledger.Reason{ledger.OutsideBudget, ledger.OutsideUnspent, ledger.OutsideReceipts,
			ledger.OutsideOffer, ledger.OutsideUnsold, ledger.OutsidePurchases},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, r := range outsideRows {
		thing := ledger.Money
		if r.goods {
			thing = 0
		}
		id, err := l.OpenBoundary(r.reason, thing)
		if err != nil {
			t.Fatalf("OpenBoundary(%s): %v", r.name, err)
		}
		switch r.reason {
		case ledger.OutsideBudget:
			budget = id
		case ledger.OutsideOffer:
			offer = id
		}
	}
	out, err = l.Open(ledger.KindOutside, 0, 0)
	if err != nil {
		t.Fatalf("Open(KindOutside): %v", err)
	}
	return l, out, budget, offer
}

// TestRegistry_OutsideRows verifies the six rows of the external module: each
// moves only its own thing, money or goods, with the class and between the
// kinds the table gives, no other pair of kinds is allowed, trade also runs
// between an Agent or an Estate and an Outside account in both directions, and
// each reason is declarable.
//
// Requirement: FR-LED-057, FR-EXT-017
func TestRegistry_OutsideRows(t *testing.T) {
	t.Parallel()
	for _, rw := range outsideRows {
		t.Run(rw.name, func(t *testing.T) {
			t.Parallel()
			for _, goods := range []bool{false, true} {
				class, has := ledger.ClassOf(rw.reason, goods)
				if has != (goods == rw.goods) {
					t.Errorf("ClassOf(%s, goods %v) has a row = %v, want %v", rw.name, goods, has, goods == rw.goods)
				}
				if has && class != rw.class {
					t.Errorf("ClassOf(%s, goods %v) = %d, want %d", rw.name, goods, class, rw.class)
				}
				for a := ledger.KindAgent; a <= ledger.KindBoundary; a++ {
					for b := ledger.KindAgent; b <= ledger.KindBoundary; b++ {
						want := goods == rw.goods && a == rw.from && b == rw.to
						if got := ledger.Allowed(rw.reason, goods, a, b); got != want {
							t.Errorf("Allowed(%s, goods %v, kind %d, kind %d) = %v, want %v", rw.name, goods, a, b, got, want)
						}
					}
				}
			}
			if _, ok := ledger.Rank(rw.reason); !ok {
				t.Errorf("Rank(%s) is not declared: the row must be a declared one", rw.name)
			}
			if _, err := ledger.New(ledger.Config{Regions: 1, MaxMoney: 10, MaxQuantity: 10, Declared: []ledger.Reason{rw.reason}}); err != nil {
				t.Errorf("New(Declared: [%s]) = %v, want nil", rw.name, err)
			}
		})
	}
	for _, goods := range []bool{false, true} {
		for _, k := range []ledger.Kind{ledger.KindAgent, ledger.KindEstate} {
			if !ledger.Allowed(ledger.Trade, goods, k, ledger.KindOutside) || !ledger.Allowed(ledger.Trade, goods, ledger.KindOutside, k) {
				t.Errorf("Allowed(trade, goods %v) between kind %d and Outside is refused, want allowed both ways", goods, k)
			}
		}
		if ledger.Allowed(ledger.Trade, goods, ledger.KindBoundary, ledger.KindOutside) {
			t.Errorf("Allowed(trade, goods %v, Boundary, Outside) = true, want a refusal", goods)
		}
	}
	// The six sit at ranks 13 to 18 of the table, ahead of the escrow rows, in the
	// order of spec-ledger.md 7.2 (BoundaryFlow shows them in that order).
	var order []ledger.Reason
	for _, r := range ledger.DeclaredByRank(nil) {
		if slices.ContainsFunc(outsideRows, func(o outsideRow) bool { return o.reason == r }) {
			order = append(order, r)
		}
	}
	want := []ledger.Reason{ledger.OutsideBudget, ledger.OutsideUnspent, ledger.OutsideReceipts, ledger.OutsideOffer, ledger.OutsideUnsold, ledger.OutsidePurchases}
	if !slices.Equal(order, want) {
		t.Errorf("DeclaredByRank orders the outside reasons %v, want %v", order, want)
	}
}

// TestLedger_OutsideAccount verifies that an Outside account is a non-boundary
// account of its region that cannot go negative, that the audit holds while it
// is empty at a boundary of the tick and fails, naming the sectoral identity,
// when it holds money or goods, and that sweeping it back to the boundary
// under the sink reasons restores the audit.
//
// Requirement: FR-LED-058, FR-EXT-012
func TestLedger_OutsideAccount(t *testing.T) {
	t.Parallel()
	l, out, budget, offer := outsideLedger(t)
	v, err := l.Account(out)
	if err != nil || v.Kind != ledger.KindOutside || v.Region != 0 || v.Balance != 0 {
		t.Fatalf("Account(outside) = %+v, %v, want an empty Outside account of region 0", v, err)
	}
	if err := l.Audit(); err != nil {
		t.Fatalf("Audit with the outside sector at zero = %v, want nil", err)
	}
	if err := l.Transfer(out, budget, 1, ledger.OutsideUnspent); err == nil {
		t.Errorf("Transfer from an empty Outside account = nil, want a refusal (no negative balance)")
	}

	if err := l.Transfer(budget, out, 40, ledger.OutsideBudget); err != nil {
		t.Fatalf("top up with money: %v", err)
	}
	if err := l.Move(offer, out, 0, 7, ledger.OutsideOffer); err != nil {
		t.Fatalf("top up with goods: %v", err)
	}
	var ae *ledger.AuditError
	if err := l.Audit(); !errors.As(err, &ae) || ae.Identity != ledger.IdentitySectoral {
		t.Errorf("Audit with a funded Outside account = %v, want the sectoral identity to fail", err)
	}

	// The sweep: money under outside_unspent and goods under outside_unsold.
	rb, ok := l.Boundary(ledger.OutsideUnspent, ledger.Money)
	if !ok {
		t.Fatalf("no boundary account of outside_unspent")
	}
	ub, ok := l.Boundary(ledger.OutsideUnsold, 0)
	if !ok {
		t.Fatalf("no boundary account of outside_unsold for commodity 0")
	}
	if err := l.Transfer(out, rb, 40, ledger.OutsideUnspent); err != nil {
		t.Fatalf("sweep money: %v", err)
	}
	if err := l.Move(out, ub, 0, 7, ledger.OutsideUnsold); err != nil {
		t.Fatalf("sweep goods: %v", err)
	}
	if err := l.Audit(); err != nil {
		t.Errorf("Audit after the sweep = %v, want nil", err)
	}
}

// TestLedger_OutsideOfferCreatesFreshLot verifies that goods moved from the
// boundary under outside_offer make a lot born in the current tick at full
// durability, whatever the tick, so a seller's goods are never old.
//
// Requirement: FR-EXT-014
func TestLedger_OutsideOfferCreatesFreshLot(t *testing.T) {
	t.Parallel()
	l, out, _, offer := outsideLedger(t)
	l.EndTick()
	l.EndTick()
	if err := l.Move(offer, out, 0, 4, ledger.OutsideOffer); err != nil {
		t.Fatalf("Move: %v", err)
	}
	lots, err := l.AppendLots(nil, out)
	if err != nil {
		t.Fatalf("AppendLots: %v", err)
	}
	want := ledger.Lot{Commodity: 0, Quantity: 4, Born: l.Tick(), Durability: 5}
	if len(lots) != 1 || lots[0] != want {
		t.Errorf("lots of the Outside account = %+v, want [%+v] (born in tick %d, full durability)", lots, want, l.Tick())
	}
	if l.Tick() != 2 {
		t.Errorf("Tick() = %d after two EndTick, want 2: the test is not on a later tick", l.Tick())
	}
}
