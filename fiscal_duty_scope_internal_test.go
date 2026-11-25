package economysim

import (
	"testing"
)

// The tests of the scope of a duty (spec-fiscal.md 6.1.1, 6.4, plan 3-fis Phase 6): a
// tariff and an export duty at a gate, on the world of extShipConfig (the outside node
// c_isle, gated by a_home) with a polity of one governed jurisdiction, the realm, over
// the three regions.

// tariffWorld builds the world of extShipConfig under the realm with the duties, every
// household holding 1 000. The agents: estates 0 to 2, the outside buyer 3 and seller 4,
// the realm's treasury 5, the households from 6: two in a_home, then two in b_far.
func tariffWorld(t *testing.T, ds ...Duty) (*World, *evLog) {
	t.Helper()
	rec := &evLog{}
	c := shipConfig()
	extShipConfig(&c)
	c.Modules.Polity = &PolityConfig{
		Jurisdictions: []Jurisdiction{{ID: "realm", Seat: "a_home", StartMoney: 1_000}},
		Members:       []Member{{Region: "a_home", Jurisdiction: "realm"}, {Region: "b_far", Jurisdiction: "realm"}, {Region: "c_isle", Jurisdiction: "realm"}},
	}
	c.Modules.Fiscal = &Fiscal{Duties: ds}
	w := newMill(t, c, WithSeed(exitSeed), WithAudit(), WithObserver(rec), WithStrategy("script", &shipScript{}))
	money := map[AgentID]int64{}
	for i := w.firstHousehold; i < len(w.agents); i++ {
		money[AgentID(i)] = 1_000
	}
	setBalances(t, w, money)
	rec.events = nil
	return w, rec
}

// TestFiscal_DutyScopes verifies the tariff and the export duty of spec-fiscal.md 18.4
// and the rule that a domestic trade is taxed by neither: a duty of scope import takes
// from the buyer of a trade whose seller is a counterparty, a duty of scope export from
// the seller of a trade whose buyer is one, and a trade between two households is in
// the base of neither and is not counted exempt.
//
// Arithmetic: the tariff is 1 500 basis points on the buyer of flour: a counterparty
// sells 5 at 200 (1 000) to household 6, which pays 150. The export levy is 1 000 basis
// points on the seller of grain: household 7 sells 20 at 55 (1 100) to the outside buyer
// and pays 110. A flour trade of 3 at 100 between the two households pays nothing.
//
// Requirement: FR-FIS-030, FR-FIS-031
func TestFiscal_DutyScopes(t *testing.T) {
	t.Parallel()
	tariff := Duty{ID: "tariff", Authority: "realm", Region: "a_home", Commodity: "flour", Payer: DutyBuyer, Scope: ScopeImport, RateBP: 1_500}
	levy := Duty{ID: "levy", Authority: "realm", Region: "a_home", Commodity: "grain", Payer: DutySeller, Scope: ScopeExport, RateBP: 1_000}
	w, rec := tariffWorld(t, levy, tariff)
	h1, h2 := AgentID(w.firstHousehold), AgentID(w.firstHousehold+1)
	flour, grain := commodityOf(t, w, "flour"), commodityOf(t, w, "grain")
	w.fiscal.noteDutyTrades(w, 0, []Trade{
		{Commodity: flour, Buyer: h1, Seller: extSeller, Price: 200, Quantity: 5},
		{Commodity: grain, Buyer: extBuyer, Seller: h2, Price: 55, Quantity: 20},
		{Commodity: flour, Buyer: h2, Seller: h1, Price: 100, Quantity: 3},
		{Commodity: flour, Buyer: extBuyer, Seller: h1, Price: 100, Quantity: 3},
	})
	w.p41aDuties()
	if got, want := heldMoney(t, w, h1), int64(1_000-150); got != want {
		t.Errorf("importer holds %d, want %d (a tariff of 150)", got, want)
	}
	if got, want := heldMoney(t, w, h2), int64(1_000-110); got != want {
		t.Errorf("exporter holds %d, want %d (an export duty of 110)", got, want)
	}
	if got := treasuryMoney(t, w, 0); got != 1_260 {
		t.Errorf("realm holds %d, want 1260", got)
	}
	want := []FiscalFlowApplied{
		{Flow: "levy", Kind: KindDuty, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 110, Applied: 110, Parties: 1},
		{Flow: "tariff", Kind: KindDuty, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 150, Applied: 150, Parties: 1},
	}
	got := of[FiscalFlowApplied](rec)
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("events = %+v, want %+v", got, want)
	}
}

// TestFiscal_DutyCounterpartyNeverPays verifies that a counterparty never pays a duty:
// a duty of scope any on the seller takes nothing from the outside seller, counts its
// trade exempt, and takes from a household.
//
// Arithmetic: 5 000 basis points on sellers. The outside seller sells 4 at 10 to a
// household: exempt. The household sells 4 at 10 to the outside buyer: due 20.
//
// Requirement: FR-FIS-031, FR-FIS-044
func TestFiscal_DutyCounterpartyNeverPays(t *testing.T) {
	t.Parallel()
	w, rec := tariffWorld(t, Duty{ID: "any", Authority: "realm", Payer: DutySeller, RateBP: 5_000})
	h := AgentID(w.firstHousehold)
	g := commodityOf(t, w, "grain")
	w.fiscal.noteDutyTrades(w, 0, []Trade{
		{Commodity: g, Buyer: h, Seller: extSeller, Price: 10, Quantity: 4},
		{Commodity: g, Buyer: extBuyer, Seller: h, Price: 10, Quantity: 4},
	})
	w.p41aDuties()
	want := FiscalFlowApplied{Flow: "any", Kind: KindDuty, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 20, Applied: 20, Parties: 1, Exempt: 1}
	if got := of[FiscalFlowApplied](rec); len(got) != 1 || got[0] != want {
		t.Errorf("events = %+v, want exactly %+v", got, want)
	}
	for _, cp := range w.external.cps {
		if got := accountMoney(t, w, cp.account); got != 0 {
			t.Errorf("counterparty %d holds %d, want 0: a duty touches only domestic accounts", cp.agent, got)
		}
	}
}
