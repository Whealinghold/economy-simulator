package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/ordering"
	tnr "github.com/Whealinghold/economy-sim/internal/tenure"
)

// This file is the tenure module in the production phase (spec-land.md 4, 5.3, 5.4): the
// entitlements of P2.3, the four tiers of the ration of P2.6, the cap and the reservation
// of rent at a start of P2.7 (pass 1) and the lessor supplies, the attribution and the
// transfers of rent after the last start (pass 2). The rules are internal/tenure's; this
// file converts to and from its primitive structs and holds the scratch of a tick. An
// endowment with no title is never touched: the ration of RES 5.3 runs on it unchanged
// (spec-land.md 4.1, FR-LND-006). No draw is taken; every tie goes by the stateless key.

// tierGrant is what the tiers granted to one request of the resources module, by the
// index of the request: the executions of each tier and the units the clip of the let
// demand removed.
type tierGrant struct {
	own, commons, let, pricedOut int64
}

// tenureCount are the counts of one endowment in a tick (spec-land.md 13.1, RentPaid):
// the units drawn in each tier, the let supply offered and idle, the units held by a
// closure, unserved and priced out, the tenants and lessors, the money paid and
// forfeited, the requests refused for each reason, the let units cut for rent and the
// let units drawn rent-free. They are scratch of the tick: cleared at P2.3, read by the
// rent step of P8.3a.
type tenureCount struct {
	ownUnits, commonsUnits, letUnits   int64
	offered, held                      int64
	unserved, pricedOut                int64
	tenants, lessors                   int64
	paid, forfeited                    int64
	noTitle, rentFunds                 int64
	cutUnits, exemptUnits              int64
	rentInForce, postedBefore, entitle int64
}

// idleUnits is the let supply the starts did not draw, whatever the reason: Y of
// spec-land.md 5.5.1.
func (c *tenureCount) idleUnits() int64 { return c.offered - c.letUnits }

// startLine is one draw of a start on a titled endowment: the request, the endowment and
// the line the charge reads.
type startLine struct {
	req, cell int
	exempt    bool
	line      tnr.Line
}

// rentStart is the let units one start took on one endowment, for pass 2: the executor
// that is the tenant of the attribution, the principal that pays, its account and the
// rent per unit (0 for an exempt request).
type rentStart struct {
	cell          int
	tenant, payer AgentID
	account       int
	units, rent   int64
	exempt        bool
}

// rationScratch is the scratch of the module in a tick. Nothing in it is saved: the
// entitlements are derived at P2.3, the grants at P2.6, the reservations and the starts
// at P2.7, and all are cleared at the next P2.3.
type rationScratch struct {
	// ents are the holders of each endowment with their entitlements, ascending by ID
	// (the pool first), and commons the entitlement of the commons.
	ents    [][]tnr.Holder
	commons []int64
	counts  []tenureCount
	tg      []tierGrant

	sc               tnr.Scratch
	reqs             []tnr.Request
	parts            []tnr.Part
	out              []int64
	lines            []startLine
	charge           []tnr.Line
	starts           []rentStart
	rents            []int64
	reserved         []int64
	touched          []AgentID
	tenants, lessors []tnr.Taker
	triples          []tnr.Triple
	// noTitleCell and rentCell are the endowment a refusal of the request being started
	// is counted on, -1 for none.
	noTitleCell, rentCell int
}

// titled reports whether the endowment has a title, so the tiers ration it.
func (t *tenure) titled(cell int) bool { return len(t.holds[cell]) > 0 }

// entitlements is the derived step of P2.3 (spec-land.md 4.2): for each endowment with a
// title the largest-remainder apportionment of the snapshot stock among the holders and
// the commons, the counts of the tick cleared and the reservations of the last tick
// dropped.
func (t *tenure) entitlements() {
	rt, r := &t.rt, t.world.res
	if rt.ents == nil {
		n := len(t.holds)
		rt.ents, rt.commons, rt.counts = make([][]tnr.Holder, n), make([]int64, n), make([]tenureCount, n)
	}
	clear(rt.counts)
	rt.starts = rt.starts[:0]
	for _, id := range rt.touched {
		rt.reserved[id] = 0
	}
	rt.touched = rt.touched[:0]
	tick := t.world.Tick()
	for cell, list := range t.holds {
		rt.ents[cell], rt.commons[cell] = rt.ents[cell][:0], 0
		if len(list) == 0 {
			continue
		}
		rt.parts = rt.parts[:0]
		for i, x := range list {
			rt.parts = append(rt.parts, tnr.Part{Ref: i, ID: int64(x.holder), Share: x.share, Key: ordering.Key(tick, int64(x.holder))})
		}
		rt.parts = append(rt.parts, tnr.Part{Ref: len(list), Share: wholeBP - sumTitles(list), Commons: true})
		rt.out = slices.Grow(rt.out[:0], len(list)+1)[:len(list)+1]
		if !tnr.Apportion(r.snapStock[cell], rt.parts, rt.out) {
			invariant("the apportionment of an endowment's stock failed after validation")
		}
		for i, x := range list {
			rt.ents[cell] = append(rt.ents[cell], tnr.Holder{ID: int64(x.holder), Entitled: rt.out[i]})
		}
		rt.commons[cell] = rt.out[len(list)]
		rt.counts[cell].entitle = r.snapStock[cell] - rt.commons[cell]
	}
}

// entitled is the entitlement of an agent on an endowment at the snapshot of the tick, 0
// for a holder of no title.
func (t *tenure) entitled(cell int, id AgentID) int64 {
	if cell >= len(t.rt.ents) {
		return 0
	}
	hs := t.rt.ents[cell]
	i, found := slices.BinarySearchFunc(hs, int64(id), func(h tnr.Holder, v int64) int { return cmp.Compare(h.ID, v) })
	if !found {
		return 0
	}
	return hs[i].Entitled
}

// open is the units of the stock open to a principal by rule before any competition
// (spec-land.md 4.5.1): the whole stock, or with a closed endowment only its own
// entitlement and the commons.
func (t *tenure) open(cell int, principal AgentID) int64 {
	if !t.closed[cell] {
		return t.world.res.snapStock[cell]
	}
	return t.entitled(cell, principal) + t.rt.commons[cell]
}

// spendable is the money of an agent a rent can be paid from: the balance less what the
// ledger reserves, less the rent the earlier starts of the tick reserved (spec-land.md
// 5.4.1).
func (t *tenure) spendable(id AgentID) int64 {
	v, err := t.world.ledger.Account(t.world.agents[id].account)
	if err != nil {
		invariant("the account of a principal is unreadable: " + err.Error())
	}
	return tnr.Available(v.Balance-v.Reserved, t.reservedOf(id))
}

// reservedOf is the rent the starts so far reserved against the agent.
func (t *tenure) reservedOf(id AgentID) int64 {
	if int(id) >= len(t.rt.reserved) {
		return 0
	}
	return t.rt.reserved[id]
}

// reserve adds to the rent reserved against an agent.
func (t *tenure) reserve(id AgentID, amount int64) {
	rt := &t.rt
	for len(rt.reserved) <= int(id) {
		rt.reserved = append(rt.reserved, 0)
	}
	if rt.reserved[id] == 0 {
		rt.touched = append(rt.touched, id)
	}
	rt.reserved[id] += amount
}

// exempt reports whether the request is exempt from rent: OwnUseExempt is set and its
// recipe is on the principal's own-use list (spec-land.md 5.3.4).
func (t *tenure) exempt(q *resRequest) bool {
	w := t.world
	return t.ownUseExempt && isOwn(w.roles.row(w.agents[q.principal].role), q.recipe)
}

// chargeRent is the rent per unit let to a request: the rent in force, 0 when exempt.
func (t *tenure) chargeRent(q *resRequest) int64 {
	if t.exempt(q) {
		return 0
	}
	return t.rentInForce(q.cell)
}

// ration is the ration of P2.6 for the requests of one endowment, the indexes of
// r.reqs in the order of the run: for an endowment with a title the four tiers of
// spec-land.md 4.3. It writes the grant of each request to the resources module's
// grants and its counts, and says it did; an endowment with no title is left to RES 5.3.
func (t *tenure) ration(order []int) bool {
	r, rt := t.world.res, &t.rt
	cell := r.reqs[order[0]].cell
	if !t.titled(cell) {
		return false
	}
	rt.tg = slices.Grow(rt.tg[:0], len(r.reqs))[:len(r.reqs)]
	tick := t.world.Tick()
	reqs := rt.reqs[:0]
	for _, i := range order {
		q := &r.reqs[i]
		reqs = append(reqs, tnr.Request{Ref: i, ID: int64(q.agent), Principal: int64(q.principal), Executions: q.executions,
			Draw: q.draw, Key: ordering.Key(tick, int64(q.agent)), Balance: t.spendable(q.principal), Exempt: t.exempt(q)})
	}
	rt.reqs = reqs
	p := tnr.Params{Stock: r.snapStock[cell], Commons: rt.commons[cell], Rent: t.rentInForce(cell), Closed: t.closed[cell]}
	res, ok := rt.sc.Tiers(p, rt.ents[cell], reqs)
	if !ok {
		invariant("the tiers of an endowment refused its requests after validation")
	}
	for i := range reqs {
		x := &reqs[i]
		rt.tg[x.Ref] = tierGrant{own: x.Own, commons: x.Commons, let: x.Let, pricedOut: x.PricedOut}
		r.grants[r.reqs[x.Ref].agent] = x.Own + x.Commons + x.Let
	}
	c := &rt.counts[cell]
	c.offered, c.held, c.unserved, c.pricedOut = res.Supply, res.Held, res.Unserved, res.PricedOut
	c.rentInForce, c.postedBefore = p.Rent, t.posted[cell]
	r.tallyRun(order)
	return true
}

// capRequest sets what tenure adds to the cap of a start (spec-land.md 4.5.1, 5.4.1)
// from the draws of the admitted request at records [first, first+n): noTitle when a
// draw is closed to the principal by rule, and the most executions whose rent the
// principal can pay. A draw refused by rule, or priced out of the let tier with nothing
// taken in the others, does not lower the cap of the endowment's own limit, so the
// reason is the tenure's and not no_endowment. An endowment with no title adds nothing.
func (t *tenure) capRequest(q *startReq, first, n int) {
	r, rt := t.world.res, &t.rt
	rt.lines, rt.noTitleCell, rt.rentCell = rt.lines[:0], -1, -1
	most, lifted := q.weight, false
	for i := first; i < first+n; i++ {
		d := &r.reqs[i]
		if !t.titled(d.cell) {
			most = min(most, d.grant, r.stock[d.cell]/d.draw)
			continue
		}
		g := rt.tg[i]
		byRule := r.snapStock[d.cell]/d.draw > 0 && t.open(d.cell, q.principal)/d.draw == 0
		priced := !byRule && d.grant == 0 && g.pricedOut > 0
		switch {
		case byRule:
			lifted, q.noTitle = true, true
			rt.noTitleCell = d.cell
		case priced:
			lifted = true
			rt.rentCell = d.cell
		default:
			most = min(most, d.grant, r.stock[d.cell]/d.draw)
		}
		rent := t.chargeRent(d)
		rt.lines = append(rt.lines, startLine{req: i, cell: d.cell, exempt: t.exempt(d),
			line: tnr.Line{Rent: rent, Draw: d.draw, Own: g.own, Commons: g.commons}})
	}
	if lifted {
		q.drawCap = most
	}
	t.capRent(q, lifted && rt.noTitleCell < 0)
}

// capRent sets the cap of rent of a start: the largest k at most its weight whose rent
// over all its titled draws the principal's balance, less the rent its earlier starts
// reserved, covers. It sets none when no draw pays rent. pricedOut says a draw was priced
// out with nothing granted, which is a cap of 0.
func (t *tenure) capRent(q *startReq, pricedOut bool) {
	rt := &t.rt
	rt.charge = rt.charge[:0]
	paying := pricedOut
	for _, l := range rt.lines {
		rt.charge = append(rt.charge, l.line)
		if l.line.Rent > 0 && rt.rentCell < 0 {
			rt.rentCell = l.cell
		}
		paying = paying || l.line.Rent > 0
	}
	if !paying {
		return
	}
	q.hasRentCap = true
	if pricedOut {
		return
	}
	k, _, ok := tnr.Charge(t.spendable(q.principal), q.weight, rt.charge)
	if !ok {
		invariant("the rent of a start is out of range after validation")
	}
	q.rentCap = k
}

// started is what a start that began adds: it reserves the rent of its k executions against
// its principal's later starts, counts the units drawn in each tier and the let units
// cut for rent, and records the let units for pass 2.
func (t *tenure) started(a *agent, q *startReq, first, n int, k int64) {
	rt := &t.rt
	if n == 0 || len(rt.lines) == 0 {
		return
	}
	rt.charge = rt.charge[:0]
	for _, l := range rt.lines {
		rt.charge = append(rt.charge, l.line)
	}
	rent, ok := tnr.TotalRent(rt.charge, k)
	if !ok {
		invariant("the rent of a start that began does not fit")
	}
	t.reserve(q.principal, rent)
	other := t.otherExecutions(q)
	for _, l := range rt.lines {
		c := &rt.counts[l.cell]
		own := min(k, l.line.Own)
		com := min(k-own, l.line.Commons)
		c.ownUnits += own * l.line.Draw
		c.commonsUnits += com * l.line.Draw
		units := (k - own - com) * l.line.Draw
		if q.hasRentCap && q.rentCap < other && k < other {
			c.cutUnits += letUnits(&l.line, other) - letUnits(&l.line, k)
		}
		if units == 0 {
			continue
		}
		c.letUnits += units
		rt.starts = append(rt.starts, rentStart{cell: l.cell, tenant: a.id, payer: q.principal, account: q.account,
			units: units, rent: l.line.Rent, exempt: l.exempt})
	}
}

// letUnits are the units of k executions of a line that come from let units.
func letUnits(l *tnr.Line, k int64) int64 {
	return max(0, k-l.Own-l.Commons) * l.Draw
}

// otherExecutions is the executions the start would have taken without the rent cap:
// its weight and maximum and every limit of the last count but the rent's.
func (t *tenure) otherExecutions(q *startReq) int64 {
	k := q.weight
	if q.maximum > 0 {
		k = min(k, q.maximum)
	}
	for _, l := range t.world.prod.limits {
		if l.Reason != limitRentFunds {
			k = min(k, max(l.Cap, 0))
		}
	}
	return k
}

// refused counts a request refused for a reason of tenure on the endowment that
// caused it (spec-land.md 13.1).
func (t *tenure) refused(reason Reason) {
	rt := &t.rt
	switch {
	case reason == ReasonNoTitle && rt.noTitleCell >= 0:
		rt.counts[rt.noTitleCell].noTitle++
	case reason == ReasonRentFunds && rt.rentCell >= 0:
		rt.counts[rt.rentCell].rentFunds++
	}
}

// settle is pass 2 of P2.7 (spec-land.md 5.3.1), after the last start: for each
// endowment, in (RegionID, ResourceID) order, the supply of each lessor is the
// apportionment of the let units the starts took by the unused entitlements, the
// attribution pairs tenants and lessors, and the rent is transferred with the reason
// rent, in the order of the triples.
func (t *tenure) settle() {
	rt := &t.rt
	// Stable: two draws of one start on one endowment share (cell, tenant) and keep their append order (architecture.md 7.6).
	slices.SortStableFunc(rt.starts, func(a, b rentStart) int { return cmp.Or(cmp.Compare(a.cell, b.cell), cmp.Compare(a.tenant, b.tenant)) })
	for lo, hi := 0, 0; lo < len(rt.starts); lo = hi {
		hi = lo + 1
		for hi < len(rt.starts) && rt.starts[hi].cell == rt.starts[lo].cell {
			hi++
		}
		t.settleCell(rt.starts[lo].cell, rt.starts[lo:hi])
	}
	rt.starts = rt.starts[:0]
}

// settleCell is pass 2 for one endowment: recs are its starts that took let units, in
// ascending executor ID.
func (t *tenure) settleCell(cell int, recs []rentStart) {
	rt := &t.rt
	var v int64 // the let units the starts took, V of spec-land.md 4.4.1
	rt.tenants = rt.tenants[:0]
	for _, x := range recs {
		v += x.units
		rt.tenants = append(rt.tenants, tnr.Taker{ID: int64(x.tenant), Units: x.units})
	}
	hs := rt.ents[cell]
	rt.parts = rt.parts[:0]
	tick := t.world.Tick()
	for i, h := range hs {
		rt.parts = append(rt.parts, tnr.Part{Ref: i, ID: h.ID, Share: h.Unused, Key: ordering.Key(tick, h.ID)})
	}
	rt.out = slices.Grow(rt.out[:0], len(hs))[:len(hs)]
	if !tnr.Supply(v, rt.parts, rt.out) {
		invariant("the let units the starts took pass the supply of the lessors")
	}
	rt.lessors = rt.lessors[:0]
	for i, h := range hs {
		if rt.out[i] > 0 {
			rt.lessors = append(rt.lessors, tnr.Taker{ID: h.ID, Units: rt.out[i]})
		}
	}
	var ok bool
	if rt.triples, ok = rt.sc.Attribute(rt.triples[:0], rt.tenants, rt.lessors); !ok {
		invariant("the attribution of let units to lessors failed after the supply was apportioned")
	}
	c := &rt.counts[cell]
	c.tenants, c.lessors = int64(len(rt.tenants)), int64(len(rt.lessors))
	for _, x := range recs {
		if x.exempt {
			c.exemptUnits += x.units
		}
	}
	for _, tr := range rt.triples {
		t.payRent(c, recs[tr.Tenant], AgentID(rt.lessors[tr.Lessor].ID), tr.Units)
	}
}

// payRent moves the rent of one triple from the principal of the tenant's start to the
// lessor (spec-land.md 5.3.3): nothing for an exempt request or to oneself, the part a
// payee at the balance bound cannot receive is forfeited and not charged. The payer's
// side cannot fail, the rent being reserved at the start.
func (t *tenure) payRent(c *tenureCount, x rentStart, lessor AgentID, units int64) {
	w := t.world
	if x.exempt || x.rent == 0 || x.payer == lessor {
		return
	}
	due, fits := checked.Mul(x.rent, units) // the cap checked the total rent of the start; not trusted here
	if !fits {
		invariant("the rent of a triple does not fit after the cap of the start")
	}
	to := t.account(lessor, t.regionOf(x.cell))
	v, err := w.ledger.Account(to)
	if err != nil {
		invariant("the account of a lessor is unreadable: " + err.Error())
	}
	paid, lost, ok := tnr.Credit(v.Balance, due, MaxMoney)
	if !ok {
		invariant("the balance of a lessor is out of range after validation")
	}
	c.forfeited += lost
	if paid == 0 {
		return
	}
	if err := w.ledger.Transfer(x.account, to, paid, ledger.Rent); err != nil {
		invariant("the ledger refused the rent of a start whose rent was reserved: " + err.Error())
	}
	c.paid += paid
	w.welfare.addInputSpend(int(x.payer), paid)
	if lessor >= AgentID(t.regions) {
		w.welfare.addUnearned(int(lessor), paid)
	}
}

// leastRent is the least rent a job of executions executions of draw units could owe on an
// endowment: the rent in force on the units left after the principal's own entitlement and
// the whole commons (spec-land.md 4.7.2), 0 when its own entitlement covers the draw. The
// formation of a hire (EMP) will call it as a bound; ok is false when the product does not
// fit.
func (t *tenure) leastRent(principal AgentID, cell int, executions, draw int64) (int64, bool) {
	var commons int64
	if cell < len(t.rt.commons) {
		commons = t.rt.commons[cell]
	}
	return tnr.LeastRent(executions, draw, t.entitled(cell, principal), commons, t.rentInForce(cell))
}
