package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// The tests of the payouts of the fiscal module at P1.7b (spec-fiscal.md 8, 12; plan
// 3-fis Phase 5). They share fiscalMill with the tax tests: the crown (treasury 0,
// seat home, 1 000) and the shire below it (treasury 1, seat yonder, 1 000), three home
// millers, two yonder millers and a weaver in each region. The cut itself is tested in
// internal/fiscal; these tests pin what the module does with it.

// stipend is a payout of the crown of amount a tick to the home millers.
func stipend(id string, amount int64) Payout {
	return Payout{ID: id, Authority: "crown", Target: homeMillers, Amount: amount}
}

// payoutWorld builds the fiscal mill with the payouts and every home miller holding
// 100.
func payoutWorld(t *testing.T, ps ...Payout) (*World, skillCast, *evLog) {
	t.Helper()
	w, c, rec := fiscalMill(t, &Fiscal{Payouts: ps}, nil)
	setBalances(t, w, millerMoney(c))
	rec.events = nil
	return w, c, rec
}

// payoutIDs lists the flows of the FiscalFlowApplied events in order.
func payoutIDs(rec *evLog) []string {
	var ids []string
	for _, e := range of[FiscalFlowApplied](rec) {
		ids = append(ids, e.Flow)
	}
	return ids
}

// TestFiscal_PayoutOrder verifies the order of the payouts of one tick: authorities in
// ascending jurisdiction ID, and within one authority money first, then the
// commodities; the flows of one group in ascending flow ID.
//
// Arithmetic: the crown pays the home millers 10 (astipend) and 3 grain (cgrain), the
// shire pays the yonder millers 10 (baid). The events come in the order astipend, the
// crown's grain cgrain, then the shire's baid, not in the order of the IDs.
//
// Requirement: FR-FIS-022
func TestFiscal_PayoutOrder(t *testing.T) {
	t.Parallel()
	grain := Payout{ID: "cgrain", Authority: "crown", Target: homeMillers, Commodity: "grain", Amount: 1}
	aid := Payout{ID: "baid", Authority: "shire", Target: FlowTarget{Kind: TargetHouseholds, Region: "yonder", Role: "miller"}, Amount: 10}
	w, _, rec := payoutWorld(t, stipend("astipend", 10), aid, grain)
	stockCrown(t, w, 9)
	w.p17bFiscal()
	got := payoutIDs(rec)
	want := []string{"astipend", "cgrain", "baid"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("payout events = %v, want %v", got, want)
	}
	auditNow(t, w)
}

// stockCrown puts qty units of grain into the crown's treasury: from the boundary to a
// miller of the shire, then by a trade, the reason whose row joins agents and
// treasuries for goods.
func stockCrown(t *testing.T, w *World, qty int64) {
	t.Helper()
	give(t, w, 7, payoutGrain(w), qty) // the home weaver, outside the class of the payouts
	if err := w.ledger.Move(w.agents[7].account, w.polity.tr[0].account, int(payoutGrain(w)), qty, ledger.Trade); err != nil {
		t.Fatalf("stock the crown with %d grain: %v", qty, err)
	}
}

// payoutGrain is the commodity "grain" of the world.
func payoutGrain(w *World) CommodityID {
	for i, c := range w.cfg.Commodities {
		if c.ID == "grain" {
			return CommodityID(i)
		}
	}
	return NoCommodity
}

// crownMoney is the money of the crown's treasury.
func crownMoney(t testing.TB, w *World) int64 { return treasuryMoney(t, w, 0) }

// TestFiscal_PayoutCut verifies the pro rata cut of a treasury that holds less than is
// due: the whole stock is spent, the extra unit goes to the recipient with the lowest
// key K(tick, agent), a cut recipient is counted in Cut, and the carry advances by the
// due, not by what was received. Money is conserved.
//
// Arithmetic: three home millers are due 500 each, 1 500 against 1 000. q =
// floor(500*1000/1500) = 333 each, the remainder 500 is equal for all, the leftover is
// 1 000 - 999 = 1, so the miller with the lowest K(0, id) receives 334 and the others
// 333. The crown ends at 0.
//
// Requirement: FR-FIS-038, FR-FIS-063
func TestFiscal_PayoutCut(t *testing.T) {
	t.Parallel()
	w, c, rec := payoutWorld(t, stipend("alms", 500))
	before := moneyTotal(w)
	w.p17bFiscal()
	first := c.homeMillers[0]
	for _, id := range c.homeMillers[1:] {
		if ordering.Key(0, int64(id)) < ordering.Key(0, int64(first)) {
			first = id
		}
	}
	for _, id := range c.homeMillers {
		want := int64(100 + 333)
		if id == first {
			want++
		}
		if got := heldMoney(t, w, id); got != want {
			t.Errorf("miller %d holds %d, want %d", id, got, want)
		}
	}
	if got := crownMoney(t, w); got != 0 {
		t.Errorf("crown holds %d, want 0 (the cut spends the whole stock)", got)
	}
	if got := moneyTotal(w); got != before {
		t.Errorf("money = %d, want %d (a payout creates and destroys none)", got, before)
	}
	want := FiscalFlowApplied{Flow: "alms", Kind: KindPayout, Authority: 0, Region: 0, Class: FiscalHouseholds,
		Commodity: NoCommodity, Due: 1_500, Applied: 1_000, Parties: 3, Cut: 3}
	if got := of[FiscalFlowApplied](rec); len(got) != 1 || got[0] != want {
		t.Errorf("events = %+v, want exactly %+v", got, want)
	}
	auditNow(t, w)
}

// TestFiscal_PayoutTopUp verifies a payout with a Level: each recipient is topped up to
// the level, one holding more gets nothing, and the flow keeps no carry.
//
// Arithmetic: Level 150, the millers hold 100, 100 and 200: dues 50, 50 and 0; the
// crown pays 100 of its 1 000.
//
// Requirement: FR-FIS-037
func TestFiscal_PayoutTopUp(t *testing.T) {
	t.Parallel()
	w, c, _ := payoutWorld(t, Payout{ID: "top", Authority: "crown", Target: homeMillers, Level: 150})
	m := millerMoney(c)
	m[c.homeMillers[2]] = 200
	setBalances(t, w, m)
	w.p17bFiscal()
	for k, want := range []int64{150, 150, 200} {
		if got := heldMoney(t, w, c.homeMillers[k]); got != want {
			t.Errorf("miller %d holds %d, want %d", k, got, want)
		}
	}
	if got := crownMoney(t, w); got != 900 {
		t.Errorf("crown holds %d, want 900", got)
	}
	if n := len(w.fiscal.flows[0].carry); n != 0 {
		t.Errorf("a top-up keeps %d carry pairs, want none", n)
	}
}

// TestFiscal_PayoutInKind verifies a payout in kind: goods move from the treasury to
// the households of the seat region, a short stock is cut pro rata, none is created or
// destroyed, and the units are counted in the unearned income of each household once
// they are valued at P6.3.
//
// Arithmetic: 2 grain each for three millers, 6 due against 5 held: q = floor(2*5/6) =
// 1 each, remainder 4, leftover 2, so the two lowest keys receive 2 and the third 1. A
// price of 7 in the region values the receipts at 14, 14 and 7.
//
// Requirement: FR-FIS-039, FR-FIS-056, FR-FIS-041
func TestFiscal_PayoutInKind(t *testing.T) {
	t.Parallel()
	p := Payout{ID: "ration", Authority: "crown", Target: homeMillers, Commodity: "grain", Amount: 2}
	w, c, rec := payoutWorld(t, p)
	stockCrown(t, w, 5)
	units := func(account int) int64 { n, _, _ := w.ledger.Holding(account, int(payoutGrain(w))); return n }
	all := func() (n int64) {
		for id := range w.agents {
			n += units(w.agents[id].account)
		}
		return n + units(w.polity.tr[0].account)
	}
	total, start := all(), map[AgentID]int64{}
	for _, id := range c.homeMillers {
		start[id] = units(w.agents[id].account)
	}
	w.p17bFiscal()
	got := map[AgentID]int64{}
	for _, id := range c.homeMillers {
		got[id] = units(w.agents[id].account) - start[id]
	}
	if after, left := all(), units(w.polity.tr[0].account); after != total || left != 0 {
		t.Errorf("grain = %d, treasury %d; want %d and 0 (nothing created or destroyed)", after, left, total)
	}
	if ones, twos := countOf(got, 1), countOf(got, 2); ones != 1 || twos != 2 {
		t.Errorf("millers received %v, want two 2s and one 1", got)
	}
	if e := of[FiscalFlowApplied](rec); len(e) != 1 || e[0].Commodity != payoutGrain(w) || e[0].Due != 6 || e[0].Applied != 5 || e[0].Cut != 1 {
		t.Errorf("events = %+v, want one for grain, due 6, applied 5, cut 1", e)
	}
	for i := range w.oc.valued {
		w.oc.valued[i] = 7
	}
	w.priceFiscalReceipts()
	for _, id := range c.homeMillers {
		if g, want := w.welfare.at(int(id)).unearned, 7*got[id]; g != want {
			t.Errorf("miller %d unearned = %d, want %d", id, g, want)
		}
	}
	if len(w.fiscal.receipts) != 0 {
		t.Errorf("receipts after P6.3 = %d, want 0", len(w.fiscal.receipts))
	}
	auditNow(t, w)
}

// countOf counts the values equal to v.
func countOf(m map[AgentID]int64, v int64) (n int) {
	for _, x := range m {
		if x == v {
			n++
		}
	}
	return n
}

// TestFiscal_PayoutToPool verifies that a payout to a pool is money to the region's
// estate account only: the pool grows and no household's balance moves.
//
// Requirement: FR-FIS-037
func TestFiscal_PayoutToPool(t *testing.T) {
	t.Parallel()
	cfg := fiscalMillConfig(&Fiscal{Payouts: []Payout{{ID: "aid", Authority: "crown", Target: FlowTarget{Kind: TargetPool, Region: "home"}, Amount: 40}}})
	cfg.Modules.Estates = &Estates{}
	w := newMill(t, cfg, WithSeed(exitSeed), WithAudit())
	c := castOf(t, w)
	setBalances(t, w, millerMoney(c))
	pool := accountMoney(t, w, w.estates.account(0))
	w.p17bFiscal()
	if got := accountMoney(t, w, w.estates.account(0)); got != pool+40 {
		t.Errorf("pool holds %d, want %d", got, pool+40)
	}
	for _, id := range c.homeMillers {
		if got := heldMoney(t, w, id); got != 100 {
			t.Errorf("miller %d holds %d, want 100 (a pool payout reaches no household)", id, got)
		}
	}
	if got := crownMoney(t, w); got != 960 {
		t.Errorf("crown holds %d, want 960", got)
	}
}

// TestFiscal_PayoutToTreasury verifies a grant-in-aid to a governed descendant.
//
// Requirement: FR-FIS-009
func TestFiscal_PayoutToTreasury(t *testing.T) {
	t.Parallel()
	w, _, _ := payoutWorld(t, Payout{ID: "grant", Authority: "crown", Target: FlowTarget{Kind: TargetTreasury, Jurisdiction: "shire"}, Amount: 100})
	w.p17bFiscal()
	if c, s := crownMoney(t, w), treasuryMoney(t, w, 1); c != 900 || s != 1_100 {
		t.Errorf("crown %d, shire %d, want 900 and 1100", c, s)
	}
	auditNow(t, w)
}

// TestFiscal_PayoutScaleZeroSkipped verifies that a payout at scale 0 moves nothing,
// advances no carry and emits nothing.
//
// Requirement: FR-FIS-001, FR-FIS-044
func TestFiscal_PayoutScaleZeroSkipped(t *testing.T) {
	t.Parallel()
	w, _, rec := payoutWorld(t, stipend("alms", 5))
	if err := w.SetFlowScales([]FlowScale{{Flow: "alms", BP: 0}}); err != nil {
		t.Fatalf("setScaleOf() error = %v, want nil", err)
	}
	w.p17bFiscal()
	if got := crownMoney(t, w); got != 1_000 || len(w.fiscal.flows[0].carry) != 0 || len(of[FiscalFlowApplied](rec)) != 0 {
		t.Errorf("crown %d, carries %d, events %d; want 1000, 0 and 0", got, len(w.fiscal.flows[0].carry), len(of[FiscalFlowApplied](rec)))
	}
}

// TestFiscal_WelfareTerms verifies the welfare terms: a tax paid is netted from the
// unearned income of the tick, a payout that is income is added, a payout that is
// NotIncome is not, and the observed wage (income less relief less unearned) is
// untouched by any of them.
//
// Arithmetic: a tax of 5 and a payout of 8 leave G_t = +3; with a second payout of 100
// that is NotIncome it is still +3. With the tax alone G_t = -5 and the income is -5, so
// income - G_t is 0.
//
// Requirement: FR-FIS-041, FR-FIS-042
func TestFiscal_WelfareTerms(t *testing.T) {
	t.Parallel()
	tax := Tax{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 5}
	w, c, _ := fiscalMill(t, &Fiscal{Taxes: []Tax{tax}}, millerMoney(skillCast{}))
	setBalances(t, w, millerMoney(c))
	w.p17bFiscal()
	id := int(c.homeMillers[0])
	if got := w.welfare.at(id).unearned; got != -5 {
		t.Errorf("unearned after the tax = %d, want -5", got)
	}
	rec := w.welfare.record(id, 1, false, 0, 0, make([]int64, len(w.cfg.Commodities)), nil)
	if rec.Income != -5 || rec.Unearned != -5 || rec.Income-rec.Relief-rec.Unearned != 0 {
		t.Errorf("record = %+v, want income -5, unearned -5 and an observed wage of 0", rec)
	}
	w.p63Welfare()
	if got := w.welfare.at(id).unearned; got != 0 {
		t.Errorf("unearned after P6.3 = %d, want 0 (the scratch is cleared)", got)
	}
	both := &Fiscal{Taxes: []Tax{tax}, Payouts: []Payout{stipend("alms", 8), {ID: "dole", Authority: "crown", Target: homeMillers, Amount: 100, NotIncome: true}}}
	w, c, _ = fiscalMill(t, both, nil)
	setBalances(t, w, millerMoney(c))
	w.p17bFiscal()
	if got := w.welfare.at(int(c.homeMillers[0])).unearned; got != 3 {
		t.Errorf("unearned with a tax of 5 and a payout of 8 = %d, want 3 (a NotIncome payout is not income)", got)
	}
}

// TestFiscal_NoOverdraft verifies that a treasury is never negative over a run with a
// payout above its income: the payout spends what there is and the audit holds.
//
// Requirement: FR-FIS-063
func TestFiscal_NoOverdraft(t *testing.T) {
	t.Parallel()
	w, _, _ := payoutWorld(t, stipend("alms", 700))
	for range 8 {
		step(t, w, 1)
		for j := range w.polity.tr {
			if got := treasuryMoney(t, w, j); got < 0 {
				t.Fatalf("treasury %d holds %d at tick %d, want at least 0", j, got, w.Tick())
			}
		}
	}
}
