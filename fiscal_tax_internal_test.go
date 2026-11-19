package economysim

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The tests of the taxes of the fiscal module at P1.7b (spec-fiscal.md 5, plan 3-fis
// Phase 4). They plant the balances, call the position, and read the ledger and the
// events: the arithmetic of a due is tested in internal/fiscal.

// fiscalMillConfig is skillsConfig with the fiscal module m on two governed
// jurisdictions: the crown (seat home, 1 000 of start money) and the shire below it
// (seat yonder, 1 000). The agents: estates 0 and 1, treasuries 2 (crown) and 3
// (shire), home millers 4 to 6, the home weaver 7, yonder millers 8 and 9 and the
// yonder weaver 10, every household of weight 1.
func fiscalMillConfig(m *Fiscal) Config {
	c := skillsConfig()
	c.Roles[1].TrainingTicks = 0
	c.Modules.Polity = &PolityConfig{
		Jurisdictions: []Jurisdiction{{ID: "crown", Seat: "home", StartMoney: 1_000}, {ID: "shire", Parent: "crown", Seat: "yonder", StartMoney: 1_000}},
		Members:       []Member{{Region: "home", Jurisdiction: "crown"}, {Region: "yonder", Jurisdiction: "shire"}},
	}
	c.Modules.Fiscal = m
	return c
}

// fiscalMill builds the world of fiscalMillConfig with every household holding the
// money in start (0 for an agent it does not name), and the events forgotten.
func fiscalMill(t testing.TB, m *Fiscal, start map[AgentID]int64, opts ...Option) (*World, skillCast, *evLog) {
	t.Helper()
	rec := &evLog{}
	w := newMill(t, fiscalMillConfig(m), append([]Option{WithSeed(exitSeed), WithAudit(), WithObserver(rec)}, opts...)...)
	cast := castOf(t, w)
	setBalances(t, w, start)
	rec.events = nil
	return w, cast, rec
}

// homeMillers is the target of a head tax on the home millers.
var homeMillers = FlowTarget{Kind: TargetHouseholds, Region: "home", Role: "miller"}

// accountMoney reads the money of an account.
func accountMoney(t testing.TB, w *World, account int) int64 {
	t.Helper()
	v, err := w.ledger.Account(account)
	if err != nil {
		t.Fatalf("Account(%d) error = %v, want nil", account, err)
	}
	return int64(v.Balance)
}

// treasuryMoney reads the money of the treasury of the jurisdiction at index j of the
// sorted jurisdictions (crown 0, shire 1).
func treasuryMoney(t testing.TB, w *World, j int) int64 {
	t.Helper()
	return accountMoney(t, w, w.polity.tr[j].account)
}

// millerMoney is every home miller holding 100.
func millerMoney(c skillCast) map[AgentID]int64 {
	m := map[AgentID]int64{}
	for _, id := range c.homeMillers {
		m[id] = 100
	}
	return m
}

// TestFiscal_TaxCollectsFromItsClass verifies that a head tax of 5 on the home
// millers takes 5 from each of the three of them and from nobody else (not the home
// weaver, not a yonder miller), pays it to the treasury of its authority, moves no
// money into or out of the world, and is reported once for the flow and the region
// with the counts of FR-FIS-044.
//
// Arithmetic: 3 millers of weight 1 owing 5 each is a due of 15, all paid; each ends
// at 100 - 5 = 95 and the crown at 1 000 + 15 = 1 015.
//
// Requirement: FR-FIS-002, FR-FIS-009, FR-FIS-021, FR-FIS-044
func TestFiscal_TaxCollectsFromItsClass(t *testing.T) {
	t.Parallel()
	start := map[AgentID]int64{}
	w, c, rec := fiscalMill(t, &Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 5}}}, start)
	start = millerMoney(c)
	start[c.homeWeaver], start[c.yonderMillers[0]] = 100, 100
	setBalances(t, w, start)
	before := moneyTotal(w)
	w.p17bFiscal()
	for _, id := range c.homeMillers {
		if got := heldMoney(t, w, id); got != 95 {
			t.Errorf("home miller %d holds %d, want 95", id, got)
		}
	}
	if got := heldMoney(t, w, c.homeWeaver); got != 100 {
		t.Errorf("home weaver holds %d, want 100 (not in the class)", got)
	}
	if got := heldMoney(t, w, c.yonderMillers[0]); got != 100 {
		t.Errorf("yonder miller holds %d, want 100 (not in the region)", got)
	}
	if got := treasuryMoney(t, w, 0); got != 1_015 {
		t.Errorf("crown treasury holds %d, want 1015", got)
	}
	if got := moneyTotal(w); got != before {
		t.Errorf("money after the tax = %d, want %d (a tax creates and destroys none)", got, before)
	}
	want := FiscalFlowApplied{Flow: "poll", Kind: KindTax, Authority: 0, Region: 0, Class: FiscalHouseholds,
		Commodity: NoCommodity, Due: 15, Applied: 15, Parties: 3}
	if got := of[FiscalFlowApplied](rec); len(got) != 1 || got[0] != want {
		t.Errorf("events = %+v, want exactly %+v", got, want)
	}
	auditNow(t, w)
}

// TestFiscal_ShortForgiven verifies that a payer that holds less than its due pays
// what it has, is counted in Short, and that the rest is forgiven and not carried: a
// tick later, with the money restored, it owes the due of one tick and not two.
//
// Arithmetic: the due is 5 a tick; a miller holding 2 pays 2 and is short, the two
// with 100 pay 5. Due 15, Applied 12, Short 1. In the second tick all three hold 100
// again, so the due is 15 and it is all applied.
//
// Requirement: FR-FIS-024, FR-FIS-044
func TestFiscal_ShortForgiven(t *testing.T) {
	t.Parallel()
	w, c, rec := fiscalMill(t, &Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 5}}}, nil)
	money := millerMoney(c)
	money[c.homeMillers[0]] = 2
	setBalances(t, w, money)
	w.p17bFiscal()
	setBalances(t, w, millerMoney(c))
	w.p17bFiscal()
	got := of[FiscalFlowApplied](rec)
	if len(got) != 2 || got[0].Due != 15 || got[0].Applied != 12 || got[0].Short != 1 || got[1].Due != 15 || got[1].Applied != 15 || got[1].Short != 0 {
		t.Errorf("events = %+v, want (due 15, applied 12, short 1) then (due 15, applied 15, short 0)", got)
	}
}

// TestFiscal_FixedTaxCarryStartsFromAccount verifies the carry of a tax per year: it
// starts at (account ID mod period) * 10^4, so a household's first due depends on its
// own account, advances by the due, and the pair is kept in ascending account ID.
//
// Arithmetic: Amount 1 000 a year over 365 ticks, scale 10 000: the numerator is
// 1000*10^4 + (a mod 365)*10^4 and the due is floor((1000 + a) / 365) = 2 for an
// account a below 95, with the remainder (1000 + a - 730) * 10^4 = (270 + a) * 10^4
// carried. The accounts of the home millers are all below 95.
//
// Requirement: FR-FIS-021
func TestFiscal_FixedTaxCarryStartsFromAccount(t *testing.T) {
	t.Parallel()
	w, c, rec := fiscalMill(t, &Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 1_000, Per: PerYear}}}, nil)
	setBalances(t, w, millerMoney(c))
	w.p17bFiscal()
	if got := of[FiscalFlowApplied](rec); len(got) != 1 || got[0].Due != 6 || got[0].Applied != 6 {
		t.Fatalf("events = %+v, want one with due 6 (three millers owing 2)", got)
	}
	f := &w.fiscal.flows[0]
	if len(f.carry) != 3 {
		t.Fatalf("pairs = %+v, want one for each of the 3 millers", f.carry)
	}
	for i, p := range f.carry {
		want := (270 + int64(p.account)) * 10_000
		if p.carry != want || (i > 0 && p.account <= f.carry[i-1].account) {
			t.Errorf("pair %d = %+v, want account ascending and carry %d", i, p, want)
		}
	}
}

// TestFiscal_RateTax verifies a rate tax on the balance above the exemption, with
// the remainder carried.
//
// Arithmetic: 500 basis points above an exemption of 10. A miller holding 110 has a
// base of 100: due = 100 * 500 * 10^4 / 10^8 = 5. One holding 107 has a base of 97:
// 97 * 500 * 10^4 = 4.85 * 10^8, due 4 and carry 0.85 * 10^8 = 85 000 000. One
// holding 5 is below the exemption: base 0, no due.
//
// Requirement: FR-FIS-021
func TestFiscal_RateTax(t *testing.T) {
	t.Parallel()
	w, c, rec := fiscalMill(t, &Fiscal{Taxes: []Tax{{ID: "tithe", Authority: "crown", Target: homeMillers, RateBP: 500, Exempt: 10}}}, nil)
	a, b, d := c.homeMillers[0], c.homeMillers[1], c.homeMillers[2]
	setBalances(t, w, map[AgentID]int64{a: 110, b: 107, d: 5})
	w.p17bFiscal()
	if got := of[FiscalFlowApplied](rec); len(got) != 1 || got[0].Due != 9 || got[0].Applied != 9 || got[0].Parties != 2 {
		t.Errorf("events = %+v, want one with due 9 (5 + 4), applied 9 and 2 parties", got)
	}
	for _, p := range w.fiscal.flows[0].carry {
		want := int64(0)
		if p.account == w.agents[b].account {
			want = 85_000_000
		}
		if p.carry != want {
			t.Errorf("carry of account %d = %d, want %d", p.account, p.carry, want)
		}
	}
}

// TestFiscal_TaxOrderAndRemittance verifies that the taxes on households run before the
// taxes on treasuries whatever their IDs, so a remittance is computed on the
// treasury's balance after the tick's other taxes, and that the events follow the same
// order.
//
// Arithmetic: the shire holds 1 000; "zpoll" takes 5 from each of the two yonder
// millers (shire 1 010); "alpha" (an ID that sorts first) takes 1 000 basis points of
// the shire's treasury: floor(1010 * 1000 * 10^4 / 10^8) = 101. The crown ends at
// 1 000 + 101 = 1 101, the shire at 1 010 - 101 = 909.
//
// Requirement: FR-FIS-022, FR-FIS-023, FR-FIS-009
func TestFiscal_TaxOrderAndRemittance(t *testing.T) {
	t.Parallel()
	yonder := FlowTarget{Kind: TargetHouseholds, Region: "yonder", Role: "miller"}
	m := &Fiscal{Taxes: []Tax{
		{ID: "alpha", Authority: "crown", Target: FlowTarget{Kind: TargetTreasury}, RateBP: 1_000},
		{ID: "zpoll", Authority: "shire", Target: yonder, Amount: 5},
	}}
	w, c, rec := fiscalMill(t, m, nil)
	money := map[AgentID]int64{}
	for _, id := range c.yonderMillers {
		money[id] = 100
	}
	setBalances(t, w, money)
	w.p17bFiscal()
	if got := []int64{treasuryMoney(t, w, 0), treasuryMoney(t, w, 1)}; got[0] != 1_101 || got[1] != 909 {
		t.Errorf("treasuries hold crown %d, shire %d, want 1101 and 909", got[0], got[1])
	}
	wantFiscalEvents(t, of[FiscalFlowApplied](rec), []FiscalFlowApplied{
		{Flow: "zpoll", Kind: KindTax, Authority: 1, Region: 1, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 10, Applied: 10, Parties: 2},
		{Flow: "alpha", Kind: KindTax, Authority: 0, Region: 1, Class: FiscalTreasury, Commodity: NoCommodity, Due: 101, Applied: 101, Parties: 1},
	})
	auditNow(t, w)
}

// wantFiscalEvents fails the test unless got is want, in order.
func wantFiscalEvents(t *testing.T, got, want []FiscalFlowApplied) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("events = %+v, want %+v", got, want)
	}
}

// TestFiscal_HeadroomCut verifies that a collection that would take the treasury past
// MaxMoney is cut to the headroom pro rata, with the largest remainder and the keyed
// tie, and counted in Headroom, and that the cut payers keep the rest.
//
// Arithmetic: the crown holds MaxMoney - 5; three millers owe 5 each (15) and the
// headroom is 5: floor(5 * 5 / 15) = 1 each and the remainder 25 mod 15 = 10 is equal
// for all three, so the 2 units left go to the two with the lowest keys: shares 2, 2, 1.
//
// Requirement: FR-FIS-025
func TestFiscal_HeadroomCut(t *testing.T) {
	t.Parallel()
	w, c, rec := fiscalMill(t, &Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 5}}}, nil)
	setBalances(t, w, millerMoney(c))
	boundary, ok := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		t.Fatal("Boundary(initial, money) is not open")
	}
	room := MaxMoney - 5 - treasuryMoney(t, w, 0)
	if err := w.ledger.Transfer(boundary, w.polity.tr[0].account, room, ledger.Initial); err != nil {
		t.Fatalf("Transfer(initial) error = %v, want nil", err)
	}
	w.p17bFiscal()
	if got := treasuryMoney(t, w, 0); got != MaxMoney {
		t.Errorf("crown treasury holds %d, want MaxMoney %d", got, int64(MaxMoney))
	}
	paid := map[int64]int{}
	for _, id := range c.homeMillers {
		paid[100-heldMoney(t, w, id)]++
	}
	if paid[2] != 2 || paid[1] != 1 {
		t.Errorf("amounts taken from the millers = %v, want two of 2 and one of 1", paid)
	}
	want := FiscalFlowApplied{Flow: "poll", Kind: KindTax, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 15, Applied: 5, Parties: 3, Headroom: 3}
	if got := of[FiscalFlowApplied](rec); len(got) != 1 || got[0] != want {
		t.Errorf("events = %+v, want exactly %+v", got, want)
	}
}

// TestFiscal_ScaleZeroSkipped verifies that a flow at scale 0 is skipped whole: no
// carry changes, no event, no transfer, and that at scale 10 000 again it acts. A flow
// outside its window is skipped in the same way.
//
// Requirement: FR-FIS-001, FR-FIS-006, FR-FIS-044
func TestFiscal_ScaleZeroSkipped(t *testing.T) {
	t.Parallel()
	w, c, rec := fiscalMill(t, &Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 1_000, Per: PerYear}}}, nil)
	setBalances(t, w, millerMoney(c))
	if err := w.SetFlowScales([]FlowScale{{Flow: "poll", BP: 0}}); err != nil {
		t.Fatalf("SetFlowScales(poll, 0) error = %v, want nil", err)
	}
	rec.events = nil
	w.p17bFiscal()
	if len(of[FiscalFlowApplied](rec)) != 0 || len(w.fiscal.flows[0].carry) != 0 || treasuryMoney(t, w, 0) != 1_000 {
		t.Errorf("at scale 0: events %d, pairs %d, crown %d, want none, none and 1000", len(of[FiscalFlowApplied](rec)), len(w.fiscal.flows[0].carry), treasuryMoney(t, w, 0))
	}
	if err := w.SetFlowScales([]FlowScale{{Flow: "poll", BP: wholeBP}}); err != nil {
		t.Fatalf("SetFlowScales(poll, 10000) error = %v, want nil", err)
	}
	w.p17bFiscal()
	if len(of[FiscalFlowApplied](rec)) != 1 || len(w.fiscal.flows[0].carry) != 3 {
		t.Errorf("at scale 10000: events %d, pairs %d, want 1 and 3", len(of[FiscalFlowApplied](rec)), len(w.fiscal.flows[0].carry))
	}
}

// TestFiscal_ClosedAccountCarryDropped verifies that the carry of a removed household is
// dropped at the flow's next run and is not saved, and that nothing else is kept for
// it (plan 3-fis D3: no per-agent state and no call in the removal).
//
// Requirement: FR-FIS-060
func TestFiscal_ClosedAccountCarryDropped(t *testing.T) {
	t.Parallel()
	w, c, _ := fiscalMill(t, &Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 1_000, Per: PerYear}}}, nil)
	setBalances(t, w, millerMoney(c))
	w.p17bFiscal()
	gone := c.homeMillers[1]
	if err := w.removeAgent(&w.agents[gone], CauseConsumer); err != nil {
		t.Fatalf("removeAgent error = %v, want nil", err)
	}
	if len(w.fiscal.flows[0].carry) != 3 {
		t.Fatalf("pairs after the removal = %d, want 3: nothing is called at a removal", len(w.fiscal.flows[0].carry))
	}
	if _, err := w.Checkpoint(); err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	w.p17bFiscal()
	for _, p := range w.fiscal.flows[0].carry {
		if p.account == w.agents[gone].account {
			t.Errorf("pair of the removed account %d is still kept", p.account)
		}
	}
	if len(w.fiscal.flows[0].carry) != 2 {
		t.Errorf("pairs after the next run = %d, want 2", len(w.fiscal.flows[0].carry))
	}
}

// TestFiscal_SetFlowScalesRefusals verifies the refusals of the scale call for a fiscal
// flow: an unknown ID, a scale out of bounds and a repeated ID, each with its sentinel
// and no change.
//
// Requirement: FR-FIS-010
func TestFiscal_SetFlowScalesRefusals(t *testing.T) {
	t.Parallel()
	w, _, rec := fiscalMill(t, &Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 5}}}, nil)
	for _, tt := range []struct {
		name    string
		changes []FlowScale
		want    error
	}{
		{"unknown flow", []FlowScale{{Flow: "nope", BP: 1}}, ErrUnknownFlow},
		{"above the bound", []FlowScale{{Flow: "poll", BP: MaxFlowScale + 1}}, ErrOutOfBounds},
		{"negative", []FlowScale{{Flow: "poll", BP: -1}}, ErrOutOfBounds},
		{"repeated", []FlowScale{{Flow: "poll", BP: 1}, {Flow: "poll", BP: 2}}, ErrDuplicateID},
	} {
		if err := w.SetFlowScales(tt.changes); !errors.Is(err, tt.want) {
			t.Errorf("SetFlowScales(%s) error = %v, want %v", tt.name, err, tt.want)
		}
	}
	if w.fiscal.flows[0].scale != wholeBP || len(of[MoneyFlowScaled](rec)) != 0 {
		t.Errorf("after the refusals: scale %d, %d events, want 10000 and none", w.fiscal.flows[0].scale, len(of[MoneyFlowScaled](rec)))
	}
}
