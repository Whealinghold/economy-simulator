package economysim

import (
	"cmp"
	"errors"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	crd "github.com/Whealinghold/economy-sim/internal/credit"
	"github.com/Whealinghold/economy-sim/internal/insure"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// This file is the settlement of the insurance module (spec-insurance.md 8, 9.3, 11.1; the
// continuation of insurance.go, architecture.md 2.6): P1.3a for the route policies of the
// arrivals, P8.1b for the store policies of the tick, the pass of an underwriter's pot, the
// payout claim, the end of a policy and the removal step 2f, and the welfare term N_t of
// the payouts, the premiums and the principal of a payout claim. It takes no word of the
// stream: every tie is the stateless key K(t, policy ID) (FR-INS-033). A refusal of the
// ledger after the checks is a violated invariant.

// coverArrival is the arrival of an insured shipment kept for P1.3a (risk_loss.go, hook
// H-11): the shipment, its origin region, whose unit values price the loss, and the range
// of the units lost, lines first and then the traveller's account, in the module's
// amounts.
type coverArrival struct {
	ship   ShipmentID
	from   RegionID
	lo, hi int
}

// coverTotals are the cumulative totals of the module (spec-insurance.md 9.3), saved with
// its section: the policies formed, the premiums paid, the sum due, paid at once, dropped
// and cut. The shortfall formed as a claim is the ledger's Awarded.
type coverTotals struct {
	formed, premiums, due, paid, dropped, cut int64
}

// coverItem is one policy at a position of the settlement: the policy as it was, the
// regions of the two parties at the event, the loss value and the due, and what became of
// the due.
type coverItem struct {
	p                         coverPolicy
	region, underRegion       RegionID
	lossValue, due            int64
	paid, short, dropped, cut int64
	claim                     int64
	valueCapped               int64
	capped                    bool
	cause                     CoverCause
}

// coverGroup is the policies of one underwriter at a position, the slice [lo, hi) of the
// order, with its balance recorded at the entry of the position (spec-insurance.md 8.3.1).
type coverGroup struct {
	lo, hi  int
	balance int64
}

// coverRow is what CoverSettled reports for one region of a position.
type coverRow struct {
	ended, claimed, capped, valueCapped int64
	lossValue, due, paid, short         int64
	dropped, cut, shielded              int64
}

// quiet reports whether every count and sum of the row is 0.
func (r coverRow) quiet() bool { return r == coverRow{} }

// claimRow is what CoverSettled of the stage claim reports for one region.
type claimRow struct {
	ended, paid, received int64
}

// coverSettle is the scratch of the settlement, grown once and reused (NFR-INS-001).
type coverSettle struct {
	items  []coverItem
	order  []int
	groups []coverGroup
	owed   []int64
	tie    []uint64
	shares []int64
	alloc  crd.Scratch
	// values, capped and qty are the unit values of the region valueOf (ok says they are
	// set), whether each is at the cap, and the units lost by commodity of the item built.
	values, qty []int64
	capped      []bool
	valueOf     RegionID
	valuesOK    bool
	rows        []coverRow
	claims      []claimRow
}

// noteArrival keeps the units lost at the arrival of a shipment that has a route policy,
// for P1.3a (hook H-11; it is nil-safe and called with the module off or on). lost is the
// units lost of each line of the shipment and accountLost those of the traveller's
// account; both are the scratch of the arrival and are copied. It is kept whether or not
// an observer was supplied, and a held arrival never reaches it (spec-insurance.md 8.1.2).
func (in *insurance) noteArrival(s *shipment, lost []int64, accountLost []Amount) {
	if in == nil {
		return
	}
	if _, ok := in.byShipment[s.id]; !ok {
		return
	}
	lo := len(in.amounts)
	for i, ln := range s.lines {
		if i < len(lost) && lost[i] > 0 {
			in.amounts = append(in.amounts, Amount{Commodity: ln.Commodity, Quantity: Quantity(lost[i])})
		}
	}
	in.amounts = append(in.amounts, accountLost...)
	in.arrived = append(in.arrived, coverArrival{ship: s.id, from: s.from, lo: lo, hi: len(in.amounts)})
}

// p13aSettleRoutes is P1.3a (spec-insurance.md 8.1): the route policies of the shipments
// that arrived at P1.3, found by the shipment index from the arrivals the loss step kept,
// settled in ascending policy ID. It draws no word.
func (w *World) p13aSettleRoutes() {
	in := w.insurance
	if len(in.arrived) == 0 {
		return
	}
	st := &in.st
	st.items = st.items[:0]
	st.valuesOK = false
	for i := range in.arrived {
		ar := &in.arrived[i]
		id, ok := in.byShipment[ar.ship]
		if !ok {
			continue
		}
		p, found := in.route(id)
		if !found {
			invariant("a route policy of the shipment index is not in the open list")
		}
		st.items = append(st.items, w.coverItemOf(p, ar.from, in.amounts[ar.lo:ar.hi]))
	}
	in.arrived, in.amounts = in.arrived[:0], in.amounts[:0]
	slices.SortFunc(st.items, func(a, b coverItem) int { return cmp.Compare(a.p.id, b.p.id) })
	w.settleCoverItems(CoverStageArrival, false)
}

// p81bSettleStores is P8.1b (spec-insurance.md 8.1): the store policies of the tick, in
// ascending policy ID, with the units each insured lost to the strikes of P8.1a.
func (w *World) p81bSettleStores() {
	in := w.insurance
	if len(in.stores) == 0 {
		return
	}
	st := &in.st
	st.items = st.items[:0]
	st.valuesOK = false
	for i := range in.stores {
		p := &in.stores[i]
		lost, g := w.storeLost(p.insured)
		st.items = append(st.items, w.coverItemOf(p, g, lost))
	}
	w.settleCoverItems(CoverStageStore, true)
}

// storeLost returns the units an agent's stores lost at P8.1a, every class, and the region
// of the strike: the amounts of an agent's strikes are one range of the position's
// amounts (risk_loss.go). An agent that was not struck has none, in its own region.
func (w *World) storeLost(id AgentID) ([]Amount, RegionID) {
	r := w.risk
	i, found := slices.BinarySearchFunc(r.strikes, id, func(s storeStrike, id AgentID) int { return cmp.Compare(s.agent, id) })
	if !found {
		return nil, w.regionOf(&w.agents[id])
	}
	j := i
	for j+1 < len(r.strikes) && r.strikes[j+1].agent == id {
		j++
	}
	return r.amounts[r.strikes[i].lo:r.strikes[j].hi], r.strikes[i].region
}

// coverItemOf makes the item of a policy at its position: the loss value of the units lost
// at the unit values of region g at the settlement and the due, the cover share of it cut
// at the maximum payout (spec-insurance.md 8.2).
func (w *World) coverItemOf(p *coverPolicy, g RegionID, lost []Amount) coverItem {
	st := &w.insurance.st
	it := coverItem{p: *p, claim: -1}
	if len(lost) > 0 {
		w.coverValuesOf(g)
		clear(st.qty)
		for _, a := range lost {
			st.qty[a.Commodity] += int64(a.Quantity)
			if st.capped[a.Commodity] {
				it.valueCapped = addSat(it.valueCapped, int64(a.Quantity))
			}
		}
		it.lossValue, _ = insure.LossValue(st.qty, st.values, MaxMoney)
	}
	it.due, it.capped = insure.Due(p.maxPayout, p.coverBP, it.lossValue)
	return it
}

// coverValuesOf sets the unit values of region g at the prices published before the
// position (spec-insurance.md 8.2.1), once for a run of items of the same region.
func (w *World) coverValuesOf(g RegionID) {
	st := &w.insurance.st
	if st.valuesOK && st.valueOf == g {
		return
	}
	n := len(w.cfg.Commodities)
	st.qty, st.capped = coverInts(st.qty, n), coverBools(st.capped, n)
	st.values = w.unitValues(g, st.values[:0], st.capped)
	st.valueOf, st.valuesOK = g, true
}

// settleCoverItems is the settlement of the items of a position (spec-insurance.md 8.3 to 8.5):
// the pot of each underwriter from the balance recorded at the entry, the payments and the
// shortfall claims, then the ends in ascending policy ID with their events, then
// CoverSettled per region. late says the position is after P6.3, so a payment enters the
// record of its tick by amendment (spec-insurance.md 11.1.3).
func (w *World) settleCoverItems(stage CoverStage, late bool) {
	in := w.insurance
	st := &in.st
	if st.rows == nil {
		st.rows = make([]coverRow, len(w.cfg.Regions))
	}
	clear(st.rows)
	w.coverGroups()
	if len(st.groups) > 0 && w.credit.cfg.ShieldTicks > 0 {
		w.valuePrices() // the upkeep cost of the shield is valued at the prices published before the position
	}
	for g := range st.groups {
		w.payCoverGroup(&st.groups[g], late)
	}
	for i := range st.items {
		w.endCoverItem(&st.items[i], stage)
	}
	w.reportCoverSettled(stage)
}

// coverGroups orders the items with a due by underwriter then policy ID and records the
// balance of each underwriter at the entry of the position.
func (w *World) coverGroups() {
	st := &w.insurance.st
	st.order, st.groups = st.order[:0], st.groups[:0]
	for i := range st.items {
		if st.items[i].due > 0 {
			st.order = append(st.order, i)
		}
	}
	slices.SortFunc(st.order, func(a, b int) int {
		return cmp.Or(cmp.Compare(st.items[a].p.underwriter, st.items[b].p.underwriter), cmp.Compare(a, b))
	})
	for i := 0; i < len(st.order); {
		j := i
		u := st.items[st.order[i]].p.underwriter
		for j < len(st.order) && st.items[st.order[j]].p.underwriter == u {
			j++
		}
		st.groups = append(st.groups, coverGroup{lo: i, hi: j, balance: w.accountView(w.agents[u].account).Balance})
		i = j
	}
}

// payCoverGroup pays the policies of one underwriter: the pot of the recorded balance less the
// shield, shared among them pro rata by what each is due with the largest remainder, each
// share paid to its insured cut to the headroom, the rest formed as a payout claim.
func (w *World) payCoverGroup(g *coverGroup, late bool) {
	in := w.insurance
	st, t := &in.st, w.Tick()
	st.owed, st.tie = st.owed[:0], st.tie[:0]
	var total int64
	for _, k := range st.order[g.lo:g.hi] {
		it := &st.items[k]
		var ok bool
		if total, ok = checked.Add(total, it.due); !ok {
			invariant("the due of an underwriter's policies does not fit in int64")
		}
		st.owed, st.tie = append(st.owed, it.due), append(st.tie, ordering.Key(t, it.p.id))
	}
	under := &w.agents[st.items[st.order[g.lo]].p.underwriter]
	var upkeep int64
	if w.credit.cfg.ShieldTicks > 0 {
		upkeep = w.creditUpkeep(under)
	}
	pay, ok := crd.Pay(g.balance, w.credit.cfg.ShieldTicks, upkeep, total, MaxMoney)
	if !ok {
		invariant("a negative figure in the pot of an underwriter")
	}
	st.rows[w.regionOf(under)].shielded += min(g.balance, total) - pay
	if st.shares, ok = st.alloc.Allocate(st.shares, pay, st.owed, st.tie); !ok {
		invariant("the allocation of an underwriter's payment was refused")
	}
	for n, k := range st.order[g.lo:g.hi] {
		w.payCoverItem(&st.items[k], st.shares[n], late)
	}
}

// payCoverItem pays one policy its share, cut to the insured's headroom, and forms the payout
// claim for what the pot did not cover; a shortfall that cannot be recorded is dropped
// (spec-insurance.md 8.3.4, 8.4).
func (w *World) payCoverItem(it *coverItem, share int64, late bool) {
	in := w.insurance
	insured, under := &w.agents[it.p.insured], &w.agents[it.p.underwriter]
	it.paid = min(share, MaxMoney-w.accountView(insured.account).Balance)
	it.cut = share - it.paid
	if it.paid > 0 {
		mustClaim(w.ledger.Transfer(under.account, insured.account, it.paid, ledger.Payout), "a payout")
		w.coverWelfare(it.p.insured, it.paid, late)
		w.coverWelfare(it.p.underwriter, -it.paid, late)
	}
	if short := it.due - share; short > 0 {
		w.formCoverClaim(it, short, insured.account, under.account)
	}
	in.tot.due, in.tot.paid = addSat(in.tot.due, it.due), addSat(in.tot.paid, it.paid)
	in.tot.dropped, in.tot.cut = addSat(in.tot.dropped, it.dropped), addSat(in.tot.cut, it.cut)
}

// formCoverClaim records the shortfall of a policy as a payout claim of the insured on the
// underwriter, due PayoutTicks on; when the list is full it is dropped and counted
// (spec-insurance.md 8.4.2).
func (w *World) formCoverClaim(it *coverItem, short int64, insured, under int) {
	id, err := w.ledger.Obligate(insured, under, short, w.Tick()+w.insurance.cfg.PayoutTicks)
	if errors.Is(err, ledger.ErrTooManyClaims) || errors.Is(err, ledger.ErrBalanceOverflow) {
		it.dropped = short
		return
	}
	mustClaim(err, "a payout claim")
	claim, found := w.ledger.Claim(id)
	if !found {
		invariant("a formed payout claim is not in the ledger's list")
	}
	w.credit.idx.add(claim)
	it.short, it.claim = short, id
}

// endCoverItem ends the policy of an item with its cause: nothing lost, or a payout due. The
// event is emitted by report; a store policy that ended with no loss has none.
func (w *World) endCoverItem(it *coverItem, stage CoverStage) {
	in := w.insurance
	it.cause = CoverClaimed
	if it.lossValue == 0 {
		it.cause = CoverNoLoss
	}
	it.region, it.underRegion = w.regionOf(&w.agents[it.p.insured]), w.regionOf(&w.agents[it.p.underwriter])
	row := &in.st.rows[it.region]
	row.ended++
	if it.cause == CoverClaimed {
		row.claimed++
	}
	if it.capped {
		row.capped++
	}
	row.lossValue, row.due = addSat(row.lossValue, it.lossValue), addSat(row.due, it.due)
	row.paid, row.short = addSat(row.paid, it.paid), addSat(row.short, it.short)
	row.dropped, row.cut = addSat(row.dropped, it.dropped), addSat(row.cut, it.cut)
	row.valueCapped = addSat(row.valueCapped, it.valueCapped)
	in.end(&it.p)
	w.reportCoverEnded(it, stage == CoverStageStore && it.cause == CoverNoLoss)
}

// openPolicy adds a policy just formed to the lists and the totals and puts its premium in
// the welfare term N_t of both parties: a cost of the insured and an income of the
// underwriter, in the scratch of the tick (spec-insurance.md 11.1; P3.3c is before P6.3).
func (w *World) openPolicy(p coverPolicy) {
	in := w.insurance
	in.add(p)
	in.tot.formed++
	in.tot.premiums = addSat(in.tot.premiums, p.premium)
	w.coverWelfare(p.insured, -p.premium, false)
	w.coverWelfare(p.underwriter, p.premium, false)
}

// coverWelfare adds a signed amount of N_t to G_t of an agent (spec-insurance.md 11.1): to
// the scratch of the tick for a flow before P6.3 and, for one after it, to the record of
// the tick already pushed. An agent with no record is not amended, and the amount is in no
// record (11.1.3).
func (w *World) coverWelfare(id AgentID, delta int64, late bool) {
	if !late {
		w.welfare.addUnearned(int(id), delta)
		return
	}
	a := &w.agents[id]
	if a.weight < 1 {
		return
	}
	a.history.Amend(delta)
}

// dropPolicies is step 2f of a removal (spec-insurance.md 8.6): each open policy of the
// removed agent ends, in ascending policy ID, as insured_removed or underwriter_removed,
// with no money moved: the premium is not refunded and no payout is due for a loss not yet
// told. A store policy of the tick, formed and not yet settled, is among them (8.6.3). It
// is nil-safe and has no result.
func (w *World) dropPolicies(a *agent) {
	in := w.insurance
	if in == nil || in.policiesOf(a.id) == 0 && in.liabilityOf(a.id) == 0 && len(in.stores) == 0 {
		return
	}
	st := &in.st
	st.items = in.collectOf(a.id, st.items[:0])
	slices.SortFunc(st.items, func(x, y coverItem) int { return cmp.Compare(x.p.id, y.p.id) })
	for i := range st.items {
		it := &st.items[i]
		it.cause = CoverUnderwriterRemoved
		if it.p.insured == a.id {
			it.cause = CoverInsuredRemoved
		}
		it.region, it.underRegion = w.regionOf(&w.agents[it.p.insured]), w.regionOf(&w.agents[it.p.underwriter])
		in.end(&it.p)
		w.reportCoverEnded(it, false)
	}
}

// collectOf appends an item for each open policy, route and store, of an agent as the
// insured or the underwriter.
func (in *insurance) collectOf(id AgentID, dst []coverItem) []coverItem {
	for _, list := range [2][]coverPolicy{in.open, in.stores} {
		for i := range list {
			if p := &list[i]; p.insured == id || p.underwriter == id {
				dst = append(dst, coverItem{p: *p, claim: -1})
			}
		}
	}
	return dst
}

// coverClaimPaid is hook H-12 at the payment of the settlement pass: the principal paid on
// a claim of origin payout is N_t of both parties in the scratch of the tick (spec-insurance.md
// 11.1: the creditor receives it, the debtor pays it) and is counted for CoverSettled of the
// stage claim by the region of the debtor and of the creditor. A loan, and a payment of 0,
// change nothing; it is nil-safe.
func (w *World) coverClaimPaid(c *ledger.Claim, paid int64, debtor ledger.View) {
	if w.insurance == nil || c.Origin != ledger.OriginPayout || paid == 0 {
		return
	}
	rows := w.insurance.claimRows(len(w.cfg.Regions))
	rows[debtor.Region].paid = addSat(rows[debtor.Region].paid, paid)
	w.coverWelfare(AgentID(debtor.Owner), -paid, false)
	if cv := w.accountView(c.Creditor); cv.Kind == ledger.KindAgent {
		rows[cv.Region].received = addSat(rows[cv.Region].received, paid)
		w.coverWelfare(AgentID(cv.Owner), paid, false)
	}
}

// coverClaimLate is the same term for the waterfall of a removal at P7.1, which runs after
// the record of the tick was pushed: the principal paid amends the last record of both
// parties (spec-insurance.md 11.1.3). It is nil-safe.
func (w *World) coverClaimLate(v *visit, debtor ledger.View) {
	if w.insurance == nil || v.c.Origin != ledger.OriginPayout || v.paidP == 0 {
		return
	}
	w.coverWelfare(AgentID(debtor.Owner), -v.paidP, true)
	if cv := w.accountView(v.c.Creditor); cv.Kind == ledger.KindAgent {
		w.coverWelfare(AgentID(cv.Owner), v.paidP, true)
	}
}

// coverClaimEnded counts a payout claim ended repaid by the region of its debtor, for
// CoverSettled of the stage claim. It is nil-safe.
func (w *World) coverClaimEnded(c *ledger.Claim, cause Cause) {
	if w.insurance == nil || c.Origin != ledger.OriginPayout || cause != CauseRepaid {
		return
	}
	rows := w.insurance.claimRows(len(w.cfg.Regions))
	rows[w.accountView(c.Debtor).Region].ended++
}

// claimRows returns the per-region counts of the claim stage, made on first use.
func (in *insurance) claimRows(regions int) []claimRow {
	if in.st.claims == nil {
		in.st.claims = make([]claimRow, regions)
	}
	return in.st.claims
}

// reportCoverClaims delivers CoverSettled of the stage claim for each region, in
// ascending RegionID, that has a count, after CreditSettled of P1.7a, and empties the
// counts. It is nil-safe.
func (w *World) reportCoverClaims() {
	in := w.insurance
	if in == nil {
		return
	}
	t := w.Tick()
	for g, r := range in.st.claims {
		if r != (claimRow{}) && w.obs.supplied {
			w.obs.deliver(CoverSettled{Tick: t, Region: RegionID(g), Stage: CoverStageClaim, ClaimEnded: r.ended, ClaimPaid: Money(r.paid), ClaimReceived: Money(r.received)})
		}
	}
	clear(in.st.claims)
}
