package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The tests of the legal prices of the institutions module in the book (spec-institutions.md
// 6.3): the clamp of an order's limit in admission, whoever places it, and the events that
// count it. The rules are in internal/institution and internal/clearing; these build a world.

// orgClampToy is orgToy with a control on grain in region b: the reference price of grain is
// 100, a cap of 12 000 bp is 100 * 12 000 / 10 000 = 120 and a floor of 8 000 bp is 80.
func orgClampToy(capBP, floorBP int64) Config {
	c := orgToy()
	c.Modules.Institutions.Controls = []Control{{ID: "assize", Commodity: "grain", Region: "b", CapBP: capBP, FloorBP: floorBP}}
	return c
}

// orgGive puts n units of grain in the account of the agent.
func orgGive(t *testing.T, w *World, a *agent, n int64) {
	t.Helper()
	src, ok := w.ledger.Boundary(ledger.Initial, 0)
	if !ok {
		t.Fatalf("no Initial boundary for grain")
	}
	if err := w.ledger.Move(src, a.account, 0, n, ledger.Initial); err != nil {
		t.Fatalf("Move() error = %v, want nil", err)
	}
}

// TestClamp_ReservationAndOutcomeCarryReplacedLimit verifies that an order admitted under a
// legal range carries the replaced limit everywhere it is read: OrderPlaced, the reservation
// of its cash, and the entry the clearing and the outcome read (spec-institutions.md 6.3.1),
// and that the quantity is untouched.
//
// Arithmetic: the cap of b is 120 and the floor 80. A bid of 3 units at 200 is lowered to
// 120, so it reserves 3 * 120 = 360 and not 600; an ask of 2 units at 50 is raised to 80;
// a bid at 100, inside the range, is unchanged.
//
// Requirement: FR-ORG-030, FR-ORG-032
func TestClamp_ReservationAndOutcomeCarryReplacedLimit(t *testing.T) {
	t.Parallel()
	w, rec := orgWorld(t, orgClampToy(12_000, 8_000))
	farmers := orgAgents(w, orgB, orgFarmer)
	buyer, seller := &w.agents[farmers[0]], &w.agents[orgAgents(w, orgB, orgSmith)[0]]
	orgGive(t, w, seller, 5)
	before, err := w.ledger.Account(buyer.account)
	if err != nil {
		t.Fatalf("Account() error = %v, want nil", err)
	}
	w.submit(buyer, orgB, Intent{Commodity: 0, Side: Bid, Quantity: 3, Limit: 200})
	w.submit(seller, orgB, Intent{Commodity: 0, Side: Ask, Quantity: 2, Limit: 50})
	w.submit(&w.agents[farmers[1]], orgB, Intent{Commodity: 0, Side: Bid, Quantity: 1, Limit: 100})
	after, _ := w.ledger.Account(buyer.account)
	if got := after.Reserved - before.Reserved; got != 360 {
		t.Errorf("cash reserved by the bid of 3 at 200 = %d, want 360 (3 at the cap of 120)", got)
	}
	var entries []bookEntry
	entries = append(entries, w.markets.books[w.markets.index(orgB, 0)].entries...)
	for i, want := range []struct {
		price Price
		qty   Quantity
	}{{120, 3}, {80, 2}, {100, 1}} {
		if entries[i].price != want.price || entries[i].qty != want.qty {
			t.Errorf("entry %d = %d units at %d, want %d units at %d", i, entries[i].qty, entries[i].price, want.qty, want.price)
		}
	}
	placed := of[OrderPlaced](rec)
	if len(placed) != 3 {
		t.Fatalf("OrderPlaced events = %d, want 3", len(placed))
	}
	for i, want := range []Price{120, 80, 100} {
		if placed[i].Limit != want {
			t.Errorf("OrderPlaced %d limit = %d, want %d", i, placed[i].Limit, want)
		}
	}
}

// TestClamp_NoQuantityNoRejectNoDraw verifies that the clamp cuts no quantity, rejects no
// order and takes no draw: the orders of a world under a binding cap are all admitted with
// their quantities, and the stream of the world after them is where it would be without the
// control (spec-institutions.md 6.3.4).
//
// Requirement: FR-ORG-032, FR-ORG-017
func TestClamp_NoQuantityNoRejectNoDraw(t *testing.T) {
	t.Parallel()
	on, rec := orgWorld(t, orgClampToy(5_000, 0))
	off, _ := orgWorld(t, orgToy())
	for _, w := range []*World{on, off} {
		a := &w.agents[orgAgents(w, orgB, orgFarmer)[0]]
		w.submit(a, orgB, Intent{Commodity: 0, Side: Bid, Quantity: 4, Limit: 300})
	}
	if got := of[OrderRejected](rec); len(got) != 0 {
		t.Errorf("OrderRejected = %+v, want none: the clamp rejects nothing", got)
	}
	if got := of[OrderPlaced](rec); len(got) != 1 || got[0].Quantity != 4 || got[0].Limit != 50 {
		t.Errorf("OrderPlaced = %+v, want the quantity 4 at the cap of 50", got)
	}
	if on.rnd.rnd.Uint64() != off.rnd.rnd.Uint64() {
		t.Errorf("the next word of the stream differs with and without the control, want the clamp to draw nothing")
	}
}

// TestClamp_EstateAndCounterpartyBound verifies that the clamp binds the order of an estate
// as it binds a household's (spec-institutions.md 6.3.3): the estate of b lists a lot of
// grain, and the floor of b raises its ask, while the lot's own listing is unchanged.
//
// Arithmetic: the floor of 20 000 bp is 100 * 20 000 / 10 000 = 200, above the listing price
// of the lot (about the reference price, 100), so the ask enters the book at 200.
//
// Requirement: FR-ORG-030
func TestClamp_EstateAndCounterpartyBound(t *testing.T) {
	t.Parallel()
	w, rec := orgWorld(t, orgClampToy(0, 20_000))
	src, _ := w.ledger.Boundary(ledger.Initial, 0)
	if err := w.ledger.Move(src, w.estates.account(orgB), 0, 3, ledger.Initial); err != nil {
		t.Fatalf("Move() error = %v, want nil", err)
	}
	if err := w.p34Estates(); err != nil {
		t.Fatalf("p34Estates() error = %v, want nil", err)
	}
	b := w.markets.books[w.markets.index(orgB, 0)]
	if len(b.entries) != 1 || b.entries[0].price != 200 {
		t.Fatalf("entries = %+v, want the estate's ask at the floor of 200", b.entries)
	}
	if p := of[OrderPlaced](rec); len(p) != 1 || p[0].Limit != 200 {
		t.Errorf("OrderPlaced = %+v, want the limit 200", p)
	}
}

// TestEvent_PriceControlledCounts verifies the event of P7.2b: one for each (region,
// control) with a clamp, with the orders lowered, the orders raised and the units, and not
// for an order inside the range; the counts are cleared when told.
//
// Arithmetic: with cap 120 and floor 80 in b, two bids (3 and 2 units, at 200 and 150) are
// lowered, 5 units; one ask of 4 units at 10 is raised, 4 units; a bid at 100 and an ask at
// 90 are inside. So Bids 2, Asks 1 and Units 3 + 2 + 4 = 9.
//
// Requirement: FR-ORG-041
func TestEvent_PriceControlledCounts(t *testing.T) {
	t.Parallel()
	w, rec := orgWorld(t, orgClampToy(12_000, 8_000))
	farmers, smiths := orgAgents(w, orgB, orgFarmer), orgAgents(w, orgB, orgSmith)
	orgGive(t, w, &w.agents[smiths[0]], 4)
	orgGive(t, w, &w.agents[smiths[1]], 1)
	for i, in := range []Intent{
		{Commodity: 0, Side: Bid, Quantity: 3, Limit: 200}, {Commodity: 0, Side: Bid, Quantity: 2, Limit: 150},
		{Commodity: 0, Side: Bid, Quantity: 1, Limit: 100},
	} {
		w.submit(&w.agents[farmers[i]], orgB, in)
	}
	w.submit(&w.agents[smiths[0]], orgB, Intent{Commodity: 0, Side: Ask, Quantity: 4, Limit: 10})
	w.submit(&w.agents[smiths[1]], orgB, Intent{Commodity: 0, Side: Ask, Quantity: 1, Limit: 90})
	w.p72bInstitutions()
	got := of[PriceControlled](rec)
	want := PriceControlled{Tick: w.Tick(), Region: orgB, Control: 0, Commodity: 0, Bids: 2, Asks: 1, Units: 9}
	if len(got) != 1 || got[0] != want {
		t.Errorf("PriceControlled = %+v, want [%+v]", got, want)
	}
	w.p72bInstitutions()
	if got := of[PriceControlled](rec); len(got) != 1 {
		t.Errorf("PriceControlled after a second P7.2b = %d events, want still 1: the counts are cleared when told", len(got))
	}
}

// TestAlloc_InstitutionsReadersNoOp verifies that the reads of a right and of a legal range
// allocate nothing, with no observer: the test of a right for a start, an order and a
// contract, the range of a market, the clamp of a limit and the rate cap of a request, and an
// order admitted under a binding cap (observability rule 8.2, NFR-ORG-001).
//
// Requirement: NFR-ORG-001
func TestAlloc_InstitutionsReadersNoOp(t *testing.T) {
	w := newMill(t, orgTablesToy())
	w.org.round(w)
	a := &w.agents[orgAgents(w, orgA, orgSmith)[0]]
	ord := Order{Agent: a.id, Commodity: 0, Side: Ask, Price: 200, Quantity: 1}
	reads := map[string]func(){
		"startBarred":  func() { w.org.startBarred(w, a, a, false, orgB, 0) },
		"orderBarred":  func() { w.org.orderBarred(a, orgA, ord) },
		"contract row": func() { w.org.contractRefusal(a, orgA, orgB, actSell, 0) },
		"rangeAt":      func() { w.org.rangeAt(orgB, 0) },
		"clampLimit":   func() { w.org.clampLimit(orgB, 0, true, 500) },
		"rateLimit":    func() { w.org.rateLimit(orgA, 5_000) },
		"noteClamp":    func() { w.org.noteClamp(orgB, 0, true, 3) },
	}
	for name, f := range reads {
		if n := testing.AllocsPerRun(50, f); n != 0 {
			t.Errorf("AllocsPerRun(%s) = %v, want 0", name, n)
		}
	}
}
