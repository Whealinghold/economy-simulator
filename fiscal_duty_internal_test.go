package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The tests of the duties of the fiscal module at P4.1a (spec-fiscal.md 6, plan 3-fis
// Phase 6). They plant the trades the clearing would hand over (noteDutyTrades, the
// hook of clearOne), call the position, and read the ledger and the events: the
// arithmetic of a due is tested in internal/fiscal.

// sales is a duty of the crown on the sales of every commodity in every region it
// covers, paid by the seller, of the rate in basis points.
func sales(id string, rateBP int64) Duty {
	return Duty{ID: id, Authority: "crown", Payer: DutySeller, RateBP: rateBP}
}

// dutyMill builds the fiscal mill with the duties; every home miller holds 1 000 and
// the home weaver 5 000 (agents 4 to 6 and 7, see fiscalMillConfig).
func dutyMill(t testing.TB, ds ...Duty) (*World, skillCast, *evLog) {
	t.Helper()
	w, c, rec := fiscalMill(t, &Fiscal{Duties: ds}, nil)
	money := map[AgentID]int64{c.homeWeaver: 5_000}
	for _, id := range c.homeMillers {
		money[id] = 1_000
	}
	setBalances(t, w, money)
	rec.events = nil
	return w, c, rec
}

// trade is a trade of grain between two agents.
func trade(w *World, buyer, seller AgentID, price, qty int64) Trade {
	g, _ := w.cfg.commodityID("grain")
	return Trade{Commodity: g, Buyer: buyer, Seller: seller, Price: Price(price), Quantity: Quantity(qty)}
}

// settle hands the trades of the home market to the module and runs P4.1a.
func settle(w *World, trades ...Trade) {
	w.fiscal.noteDutyTrades(w, 0, trades)
	w.p41aDuties()
}

// carryOf is the carry of the account on the flow at position i, 0 for no pair.
func carryOf(w *World, i int, id AgentID) int64 {
	for _, p := range w.fiscal.flows[i].carry {
		if p.account == w.agents[id].account {
			return p.carry
		}
	}
	return 0
}

// TestFiscal_Example_C verifies the sales duty of spec-fiscal.md 18.3 exactly: three
// trades of two sellers in one tick, then a second tick, with the carry.
//
// Arithmetic: seller A sells 10 at 53 and 2 at 410, a base of 530 + 820 = 1 350; seller
// B sells 4 at 52, a base of 208. A: 1 350 * 300 * 10^4 = 4.05 * 10^9, due 40, carry
// 5 * 10^7; B: 208 * 300 * 10^4 = 6.24 * 10^8, due 6, carry 2.4 * 10^7. In tick 1 A
// sells 100 (a base of 100): 100 * 300 * 10^4 + 5 * 10^7 = 3.5 * 10^8, due 3, carry
// 5 * 10^7. A paid 43 of the exact 0.03 * 1 450 = 43.5, with the half unit in its carry.
//
// Requirement: FR-FIS-030, FR-FIS-032, FR-FIS-044
func TestFiscal_Example_C(t *testing.T) {
	t.Parallel()
	w, c, rec := dutyMill(t, sales("sales", 300))
	a, b, buyer := c.homeMillers[0], c.homeMillers[1], c.homeWeaver
	settle(w, trade(w, buyer, a, 53, 10), trade(w, buyer, b, 52, 4), trade(w, buyer, a, 410, 2))
	if got := []int64{heldMoney(t, w, a), heldMoney(t, w, b), treasuryMoney(t, w, 0)}; !slices.Equal(got, []int64{960, 994, 1_046}) {
		t.Errorf("A, B and the crown hold %v after tick 0, want [960 994 1046]", got)
	}
	if got := []int64{carryOf(w, 0, a), carryOf(w, 0, b)}; !slices.Equal(got, []int64{50_000_000, 24_000_000}) {
		t.Errorf("carries of A and B = %v, want [50000000 24000000]", got)
	}
	settle(w, trade(w, buyer, a, 1, 100))
	if got, carry := heldMoney(t, w, a), carryOf(w, 0, a); got != 957 || carry != 50_000_000 {
		t.Errorf("A holds %d with carry %d after tick 1, want 957 (paid 43 in all) and 50000000", got, carry)
	}
	want := []FiscalFlowApplied{
		{Flow: "sales", Kind: KindDuty, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 46, Applied: 46, Parties: 2},
		{Flow: "sales", Kind: KindDuty, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 3, Applied: 3, Parties: 1},
	}
	if got := of[FiscalFlowApplied](rec); !slices.Equal(got, want) {
		t.Errorf("events = %+v, want %+v", got, want)
	}
	auditNow(t, w)
}

// TestFiscal_DutyBase verifies the base of a duty: the sum of price times quantity per
// payer, over the trades its commodity, region, payer side and window take, and no
// others, and that the buyer pays a purchase tax.
//
// Arithmetic: a duty of 10 000 basis points takes the whole value, so a due is a base.
// The weaver buys 3 grain at 7 (21) and 2 at 5 (10) and a miller buys 4 at 2 (8): the
// buyer-paid duty on grain takes 31 from the weaver and 8 from the miller; a duty on
// flour takes nothing from them.
//
// Requirement: FR-FIS-030
func TestFiscal_DutyBase(t *testing.T) {
	t.Parallel()
	buy := Duty{ID: "buy", Authority: "crown", Payer: DutyBuyer, Commodity: "grain", RateBP: 10_000}
	flour := Duty{ID: "flour", Authority: "crown", Payer: DutyBuyer, Commodity: "flour", RateBP: 10_000}
	shire := Duty{ID: "shire", Authority: "shire", Payer: DutyBuyer, RateBP: 10_000}
	w, c, _ := dutyMill(t, buy, flour, shire)
	seller, miller := c.homeMillers[0], c.homeMillers[1]
	settle(w, trade(w, c.homeWeaver, seller, 7, 3), trade(w, miller, seller, 2, 4), trade(w, c.homeWeaver, seller, 5, 2))
	if got, want := heldMoney(t, w, c.homeWeaver), int64(5_000-31); got != want {
		t.Errorf("weaver holds %d, want %d (a purchase tax of 31)", got, want)
	}
	if got, want := heldMoney(t, w, miller), int64(1_000-8); got != want {
		t.Errorf("miller holds %d, want %d (a purchase tax of 8)", got, want)
	}
	if got := treasuryMoney(t, w, 0); got != 1_000+39 {
		t.Errorf("crown holds %d, want 1039: the flour duty and the shire's take nothing in the home market", got)
	}
	if got := treasuryMoney(t, w, 1); got != 1_000 {
		t.Errorf("shire holds %d, want 1000: its region is yonder, not home", got)
	}
}

// TestFiscal_DutyPseudoExempt verifies that a pseudo-agent never pays a duty: a trade
// whose payer is an estate or a treasury is in no base and is counted in Exempt, and
// the duty still takes its share of the trade of a household.
//
// Arithmetic: a duty of 5 000 basis points on sellers. The estate (agent 0) sells 4 at
// 10 and the crown's treasury (agent 2) sells 1 at 10: two exempt trades. A miller sells
// 3 at 10 (value 30): due 15.
//
// Requirement: FR-FIS-031, FR-FIS-044
func TestFiscal_DutyPseudoExempt(t *testing.T) {
	t.Parallel()
	w, c, rec := dutyMill(t, sales("sales", 5_000))
	estate, crown := AgentID(0), w.polity.tr[0].agent
	settle(w, trade(w, c.homeWeaver, estate, 10, 4), trade(w, c.homeWeaver, crown, 10, 1), trade(w, c.homeWeaver, c.homeMillers[0], 10, 3))
	want := FiscalFlowApplied{Flow: "sales", Kind: KindDuty, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 15, Applied: 15, Parties: 1, Exempt: 2}
	if got := of[FiscalFlowApplied](rec); len(got) != 1 || got[0] != want {
		t.Errorf("events = %+v, want exactly %+v", got, want)
	}
	if got := treasuryMoney(t, w, 0); got != 1_015 {
		t.Errorf("crown holds %d, want 1015", got)
	}
}

// TestFiscal_DutyOnlyExemptStillReports verifies that a duty whose only trades have a
// pseudo-agent as payer still reports the region, with a due of 0 and the exempt count,
// and that a duty with no trade at all reports nothing (FR-FIS-044).
//
// Requirement: FR-FIS-044
func TestFiscal_DutyOnlyExemptStillReports(t *testing.T) {
	t.Parallel()
	w, c, rec := dutyMill(t, sales("sales", 300))
	settle(w)
	if got := of[FiscalFlowApplied](rec); len(got) != 0 {
		t.Errorf("events with no trade = %+v, want none", got)
	}
	settle(w, trade(w, c.homeWeaver, 0, 10, 4))
	want := FiscalFlowApplied{Flow: "sales", Kind: KindDuty, Class: FiscalHouseholds, Commodity: NoCommodity, Exempt: 1}
	if got := of[FiscalFlowApplied](rec); len(got) != 1 || got[0] != want {
		t.Errorf("events = %+v, want exactly %+v", got, want)
	}
}

// TestFiscal_DutyOrderIndependence verifies that the totals and the carries do not
// depend on the order of the trades: the same trades in two orders leave the same
// money and the same carries.
//
// Requirement: FR-FIS-032
func TestFiscal_DutyOrderIndependence(t *testing.T) {
	t.Parallel()
	run := func(reverse bool) ([]int64, int64) {
		w, c, _ := dutyMill(t, sales("sales", 300))
		a, b := c.homeMillers[0], c.homeMillers[1]
		trades := []Trade{trade(w, c.homeWeaver, a, 53, 10), trade(w, c.homeWeaver, b, 52, 4), trade(w, c.homeWeaver, a, 410, 2), trade(w, c.homeWeaver, b, 3, 9)}
		if reverse {
			slices.Reverse(trades)
		}
		settle(w, trades...)
		return []int64{carryOf(w, 0, a), carryOf(w, 0, b)}, treasuryMoney(t, w, 0)
	}
	carries, crown := run(false)
	rcarries, rcrown := run(true)
	if !slices.Equal(carries, rcarries) || crown != rcrown {
		t.Errorf("carries %v and crown %d in order, %v and %d reversed, want equal", carries, crown, rcarries, rcrown)
	}
}

// TestFiscal_DutyStack verifies that two duties on one trade each act for themselves in
// ascending flow ID, so the later finds less and a short payer is forgiven and counted,
// with the carry advancing by the due.
//
// Arithmetic: the seller holds 100 and sells 100 at 1; duty a takes 6 000 basis points
// (60) and duty b the same (60 due): the seller holds 100 - 60 = 40 when b acts, so b
// collects 40 and is short by 20, which is forgiven.
//
// Requirement: FR-FIS-024, FR-FIS-032
func TestFiscal_DutyStack(t *testing.T) {
	t.Parallel()
	w, c, rec := dutyMill(t, sales("a", 6_000), sales("b", 6_000))
	money := map[AgentID]int64{c.homeMillers[0]: 100}
	setBalances(t, w, money)
	rec.events = nil
	settle(w, trade(w, c.homeWeaver, c.homeMillers[0], 1, 100))
	if got := heldMoney(t, w, c.homeMillers[0]); got != 0 {
		t.Errorf("seller holds %d, want 0", got)
	}
	want := []FiscalFlowApplied{
		{Flow: "a", Kind: KindDuty, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 60, Applied: 60, Parties: 1},
		{Flow: "b", Kind: KindDuty, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 60, Applied: 40, Parties: 1, Short: 1},
	}
	if got := of[FiscalFlowApplied](rec); !slices.Equal(got, want) {
		t.Errorf("events = %+v, want %+v", got, want)
	}
	auditNow(t, w)
}

// TestFiscal_DutyHeadroomCut verifies that a collection that would take the treasury
// past MaxMoney is cut to the headroom pro rata, as a tax's is, and counted in Headroom.
//
// Arithmetic: the crown holds MaxMoney - 5; three millers sell 5 each at a duty of
// 10 000 basis points, due 5 each (15), headroom 5: shares 2, 2, 1 as in the tax test.
//
// Requirement: FR-FIS-025
func TestFiscal_DutyHeadroomCut(t *testing.T) {
	t.Parallel()
	w, c, rec := dutyMill(t, sales("sales", 10_000))
	boundary, ok := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		t.Fatal("Boundary(initial, money) is not open")
	}
	if err := w.ledger.Transfer(boundary, w.polity.tr[0].account, MaxMoney-5-treasuryMoney(t, w, 0), ledger.Initial); err != nil {
		t.Fatalf("Transfer(initial) error = %v, want nil", err)
	}
	var trades []Trade
	for _, id := range c.homeMillers {
		trades = append(trades, trade(w, c.homeWeaver, id, 5, 1))
	}
	settle(w, trades...)
	if got := treasuryMoney(t, w, 0); got != MaxMoney {
		t.Errorf("crown holds %d, want MaxMoney %d", got, int64(MaxMoney))
	}
	want := FiscalFlowApplied{Flow: "sales", Kind: KindDuty, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 15, Applied: 5, Parties: 3, Headroom: 3}
	if got := of[FiscalFlowApplied](rec); len(got) != 1 || got[0] != want {
		t.Errorf("events = %+v, want exactly %+v", got, want)
	}
}

// TestFiscal_DutyScaleAndWindow verifies that a duty at scale 0 or outside its window
// is skipped whole (it keeps no trade, moves nothing, emits nothing and keeps its
// carry), and that the scale multiplies the due.
//
// Arithmetic: a base of 1 000 at 300 basis points is 30 at the full scale and
// 1 000 * 300 * 5 000 / 10^8 = 15 at 5 000.
//
// Requirement: FR-FIS-006, FR-FIS-044
func TestFiscal_DutyScaleAndWindow(t *testing.T) {
	t.Parallel()
	later := sales("later", 300)
	later.Window = Window{Start: 5}
	w, c, rec := dutyMill(t, later, sales("sales", 300))
	seller, buyer := c.homeMillers[0], c.homeWeaver
	for _, scale := range []int64{0, 5_000, 10_000} {
		if err := w.setFiscalScale(1, scale, CauseConsumer); err != nil {
			t.Fatalf("setFiscalScale(%d) error = %v, want nil", scale, err)
		}
		before := heldMoney(t, w, seller)
		settle(w, trade(w, buyer, seller, 10, 100))
		if got, want := before-heldMoney(t, w, seller), map[int64]int64{0: 0, 5_000: 15, 10_000: 30}[scale]; got != want {
			t.Errorf("duty paid at scale %d = %d, want %d", scale, got, want)
		}
	}
	if got := of[FiscalFlowApplied](rec); len(got) != 2 {
		t.Errorf("events = %d, want 2: none at scale 0, none for the duty before its window", len(got))
	}
	if n := len(w.fiscal.flows[0].duty.trades) + len(w.fiscal.flows[1].duty.trades); n != 0 {
		t.Errorf("trades kept after P4.1a = %d, want 0", n)
	}
}

// TestFiscal_DutyNetsIncome verifies that a duty paid is netted from the payer's income
// of the tick (X_t) and is not in its observed wage (spec-fiscal.md 12.1).
//
// Requirement: FR-FIS-041, FR-FIS-042
func TestFiscal_DutyNetsIncome(t *testing.T) {
	t.Parallel()
	w, c, _ := dutyMill(t, sales("sales", 300))
	settle(w, trade(w, c.homeWeaver, c.homeMillers[0], 10, 100))
	if got := w.welfare.at(int(c.homeMillers[0])).unearned; got != -30 {
		t.Errorf("unearned of the seller = %d, want -30 (a duty of 30)", got)
	}
	if got := w.welfare.at(int(c.homeWeaver)).unearned; got != 0 {
		t.Errorf("unearned of the buyer = %d, want 0 (a sales duty is the seller's)", got)
	}
}

// TestFiscal_DutyAllocs verifies that the hand-over and P4.1a allocate nothing once
// their scratch has grown (NFR-FIS-001): with no observer and the default logger, ten
// trades of three payers settled over and over cost no allocation.
//
// Requirement: NFR-FIS-001
func TestFiscal_DutyAllocs(t *testing.T) {
	w := newMill(t, fiscalMillConfig(&Fiscal{Duties: []Duty{sales("sales", 300)}}), WithSeed(exitSeed))
	c := castOf(t, w)
	money := map[AgentID]int64{c.homeWeaver: 5_000}
	var trades []Trade
	for _, id := range c.homeMillers {
		money[id] = 1_000
		trades = append(trades, trade(w, c.homeWeaver, id, 7, 3), trade(w, c.homeWeaver, id, 2, 5), trade(w, c.homeWeaver, 0, 1, 1))
	}
	setBalances(t, w, money)
	settle(w, trades...)
	if n := testing.AllocsPerRun(20, func() { settle(w, trades...) }); n != 0 {
		t.Errorf("AllocsPerRun(settle) = %v, want 0", n)
	}
}
