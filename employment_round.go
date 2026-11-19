package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	payroll "github.com/Whealinghold/economy-sim/internal/employment"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// This file is the labour round of the employment module (spec-employment.md 4 and
// 5), EMP, at P2.5a: the offers and vacancies that Work returned are admitted in
// ascending agent ID, ordered by value and a stateless key, matched first fit in
// each region, and a pair that passes the formation checks becomes a hire at once:
// the bill moves to an Escrow account, the hire is recorded, the worker's own
// request is withdrawn and a hired request joins the ration of P2.6. The round
// draws nothing (spec-employment.md 5.3.1). The arithmetic is internal/employment;
// this file converts at the call (architecture.md 2.9).

// The reasons of the formation checks that a Check of the round returns
// (spec-employment.md 5.5), numbered from payroll.ReasonCheck in the order of the
// spec. The checks of the modules not built yet (2 not_permitted, 5 and 9 and 10
// of the infrastructure module, 8 of the tenure module, 11 of the storage module)
// are named slots: the order of the table is fixed here and the row is added where
// its module is.
const (
	lrZeroYield     = payroll.ReasonCheck + iota // check 3
	lrNoSite                                     // check 3
	lrNoTool                                     // check 4
	lrNoInput                                    // check 6
	lrNoEndowment                                // check 7
	lrEmployerFunds                              // check 12
	numLabourReasons
)

// labourReasons is the Reason of each code a vacancy can end the round with, by
// code (the table is never written).
var labourReasons = [numLabourReasons]Reason{
	payroll.ReasonTooMany:  ReasonTooManyHires,
	payroll.ReasonNoLabour: ReasonNoLabour,
	lrZeroYield:            ReasonZeroYield,
	lrNoSite:               ReasonNoSite,
	lrNoTool:               ReasonNoTool,
	lrNoInput:              ReasonNoInput,
	lrNoEndowment:          ReasonNoEndowment,
	lrEmployerFunds:        ReasonEmployerFunds,
}

// labourReq is what Work returned for an offer or a vacancy, kept from P2.5 to the
// round of P2.5a.
type labourReq struct {
	agent  AgentID
	region RegionID
	offer  LabourOffer
	hire   HireRequest
}

// labourLane is the admitted offers and vacancies of one region, with the agents
// and the facts the Refs of the round point to.
type labourLane struct {
	offers  []payroll.Offer
	workers []AgentID
	vacs    []payroll.Vacancy
	posts   []vacancyPost
}

// vacancyPost is the employer's side of one admitted vacancy: the agent and the
// recipe, the positions admitted, and what the checks read: site is the reason of
// check 3 (0 for none), committed the executions already committed to the employer
// this tick (its own request, its trainees' and the hires formed so far), money
// its available money at the start of the round and spent the bills formed.
type vacancyPost struct {
	employer  AgentID
	recipe    RecipeID
	positions int64
	site      int
	committed int64
	money     int64
	spent     int64
}

// noteLabour keeps the offer and the vacancy that a call of Work returned, for the
// round of P2.5a. With the hire form off they are refused, at Debug only
// (spec-employment.md 5.2, employment_off).
func (w *World) noteLabour(a *agent, region RegionID, req *JobRequest) {
	if req.Offer.Households == 0 && req.Hire.Positions == 0 {
		return
	}
	if !w.employment.isHire() {
		w.logLabourRefused("offer or vacancy refused", a.id, ReasonEmploymentOff)
		return
	}
	w.employment.reqs = append(w.employment.reqs, labourReq{agent: a.id, region: region, offer: req.Offer, hire: req.Hire})
}

// p25aLabour is P2.5a (spec-employment.md 5.1): the labour round. The requests are
// admitted in ascending agent ID, not in the rotation order of the calls, so the
// result does not depend on it, then each region's lane is matched in ascending
// RegionID and its hires formed.
func (w *World) p25aLabour() {
	e := w.employment
	if !e.hire || len(e.reqs) == 0 {
		return
	}
	slices.SortFunc(e.reqs, func(a, b labourReq) int { return cmp.Compare(a.agent, b.agent) })
	for i := range e.reqs {
		w.admitRequest(&e.reqs[i])
	}
	e.reqs = e.reqs[:0]
	for g := range e.lanes {
		if l := &e.lanes[g]; len(l.offers) > 0 || len(l.vacs) > 0 {
			w.runLane(RegionID(g), l)
			*l = labourLane{offers: l.offers[:0], workers: l.workers[:0], vacs: l.vacs[:0], posts: l.posts[:0]}
		}
	}
}

// admitRequest admits the offer and the vacancy of a request, each whole or
// refused with its reason (spec-employment.md 5.2, 5.2.1).
func (w *World) admitRequest(r *labourReq) {
	a := &w.agents[r.agent]
	if r.offer.Households != 0 {
		w.admitOffer(a, r)
	}
	if r.hire.Positions != 0 {
		w.admitVacancy(a, r)
	}
}

// offerFault returns the first admission reason that refuses the offer of agent a,
// in the order of spec-employment.md 5.2 (rows 2 to 6), or "".
func (w *World) offerFault(a *agent, o LabourOffer) Reason {
	switch {
	case !w.employment.hireable[a.role]:
		return ReasonNotHireable
	case w.skills != nil && w.skills.isTrainee(a.id):
		return ReasonInTraineeship
	case w.sub.on(a):
		return ReasonInOwnUse
	case o.Households < 1 || o.Households > a.weight:
		return ReasonInvalidHouseholds
	case o.Reserve < 0 || o.Reserve > MaxPrice:
		return ReasonInvalidPrice
	}
	return ""
}

// admitOffer admits the offer of a request into the lane of its region or counts
// and logs its refusal.
func (w *World) admitOffer(a *agent, r *labourReq) {
	t := w.employment.tallyOf(r.region, w.Tick())
	if reason := w.offerFault(a, r.offer); reason != "" {
		t.offersRefused++
		w.logLabourRefused("offer refused", a.id, reason)
		return
	}
	l := &w.employment.lanes[r.region]
	l.offers = append(l.offers, payroll.Offer{Ref: len(l.workers), Households: r.offer.Households,
		Reserve: int64(r.offer.Reserve), Key: ordering.Key(w.Tick(), int64(a.id))})
	l.workers = append(l.workers, a.id)
}

// vacancyFault returns the first admission reason that refuses the vacancy of the
// employer a, in the order of spec-employment.md 5.2 (rows 7 to 10), or "".
func (w *World) vacancyFault(a *agent, h HireRequest) Reason {
	rb := &w.employment.roles[a.role]
	switch {
	case !rb.on:
		return ReasonNotEmployer
	case !slices.Contains(rb.list, h.Recipe):
		return ReasonNotOnHireList
	case h.Positions < 1:
		return ReasonInvalidPositions
	case h.Bid < 1 || h.Bid > MaxPrice:
		return ReasonInvalidPrice
	}
	return ""
}

// admitVacancy admits the vacancy of a request into the lane of its region, its
// positions cut to what the establishment has left, or counts and logs its refusal
// (employer_full, row 11).
func (w *World) admitVacancy(a *agent, r *labourReq) {
	e := w.employment
	t := e.tallyOf(r.region, w.Tick())
	reason := w.vacancyFault(a, r.hire)
	room := e.roles[a.role].maxHires - e.hiredBy(a.id)
	if reason == "" && room < 1 {
		reason = ReasonEmployerFull
	}
	if reason != "" {
		t.vacanciesRefused++
		w.logLabourRefused("vacancy refused", a.id, reason)
		return
	}
	v, err := w.ledger.Account(a.account)
	if err != nil {
		invariant("the account of an agent is unreadable: " + err.Error())
	}
	l := &e.lanes[r.region]
	post := vacancyPost{employer: a.id, recipe: r.hire.Recipe, positions: min(r.hire.Positions, room),
		committed: w.committedOf(a), money: v.Balance - v.Reserved}
	post.site = w.hireSite(r.region, r.hire.Recipe, e.roles[a.role].yieldBP)
	l.vacs = append(l.vacs, payroll.Vacancy{Ref: len(l.posts), Positions: post.positions, Bid: int64(r.hire.Bid),
		Duration: w.cfg.Recipes[r.hire.Recipe].Duration, ShareBP: e.roles[a.role].shareBP, Key: ordering.Key(w.Tick(), int64(a.id))})
	l.posts = append(l.posts, post)
}

// committedOf is the executions already committed to the stock and the tools of
// the employer a at P2.5a: its own request and its trainees' (spec-employment.md
// 5.5, check 6). The hires formed in the round are added as they form.
func (w *World) committedOf(a *agent) int64 {
	var n int64
	if ask := &w.pr.asks[a.id]; ask.set {
		n += ask.k0
	}
	if w.skills != nil && w.skills.isTrainer(a.id) {
		for _, t := range w.skills.trainer[a.id].trainees {
			if ask := &w.pr.asks[t]; ask.set {
				n += ask.k0
			}
		}
	}
	return n
}

// hireSite is check 3 for the recipe in the region: the reason the effective
// factor of a hired execution is 0 (no_site or zero_yield), as a code, or 0. The
// factor carries HireYieldBP in the place of the trainee term (spec-employment.md
// 5.5). It does not depend on the size of the block.
func (w *World) hireSite(region RegionID, recipe RecipeID, yieldBP int64) int {
	cr := &w.cfg.Recipes[recipe]
	e, has, ok := w.yields.effectiveFor(&w.shocks, region, recipe, w.tk.moment.Season, cr.Sited,
		w.prod.abundance(region, recipe, false), w.prod.techFactor(region, recipe), yieldBP)
	if !ok {
		invariant("the effective yield of a hire does not fit after validation")
	}
	switch _, reason := yieldRefusal(cr.Sited, has, e); reason {
	case ReasonNoSite:
		return lrNoSite
	case ReasonZeroYield:
		return lrZeroYield
	}
	return 0
}

// runLane matches the lane of region g and forms its hires in formation order
// (spec-employment.md 5.3 to 5.5), then counts the round.
func (w *World) runLane(g RegionID, l *labourLane) {
	e := w.employment
	e.cur = l
	payroll.Round(&e.res, l.offers, l.vacs, payroll.Lane{Open: e.open[g], MaxOpen: e.maxOpen}, w.formationCheck)
	e.cur = nil
	w.countLane(g, l)
	for _, h := range e.res.Hires {
		w.formHire(g, l, h)
	}
}

// formationCheck is the Check of the round: the formation checks of
// spec-employment.md 5.5 in their order, for the offer o and the vacancy v at the
// wage. A pass counts the block as committed to the employer and its bill as spent,
// so the next pair of the same vacancy sees both; the round forms the hire at once.
func (w *World) formationCheck(o payroll.Offer, v payroll.Vacancy, wage int64, _ []payroll.Hire) (int, payroll.Close) {
	x := &w.employment.cur.posts[v.Ref]
	if x.site != 0 {
		return x.site, payroll.CloseVacancy
	}
	if reason := w.stockFault(x, o.Households); reason != 0 {
		return reason, payroll.CloseFit
	}
	bill, ok := payroll.Bill(wage, v.Duration, o.Households)
	if !ok || bill > x.money-x.spent {
		return lrEmployerFunds, payroll.CloseFit
	}
	x.committed += o.Households
	x.spent += bill
	return payroll.ReasonNone, payroll.CloseFit
}

// stockFault is checks 4, 6 and 7 for a block of m executions of the vacancy x, in
// that order: the tools, the inputs, the draws on an endowment. It returns the code
// of the first that fails, or 0.
func (w *World) stockFault(x *vacancyPost, m int64) int {
	need, row := x.committed+m, w.recipes.row(x.recipe)
	account := w.agents[x.employer].account
	for _, t := range row.tools {
		if w.unreserved(account, t.commodity) < need {
			return lrNoTool
		}
	}
	for _, in := range row.inputs {
		n, ok := checked.Mul(need, int64(in.quantity))
		if !ok || w.unreserved(account, in.commodity) < n {
			return lrNoInput
		}
	}
	if w.res != nil && w.drawFault(x, m) {
		return lrNoEndowment
	}
	return 0
}

// unreserved is the units of a commodity an account holds that no reservation
// earmarks.
func (w *World) unreserved(account int, c CommodityID) int64 {
	held, reserved, err := w.ledger.Holding(account, int(c))
	if err != nil {
		invariant("the stock of an employer is unreadable: " + err.Error())
	}
	return held - reserved
}

// drawFault is check 7: a draw of the recipe on an endowment that m executions
// could never take, even alone, from the snapshot of P2.3. It does not subtract
// the requests already made (spec-employment.md 5.5).
func (w *World) drawFault(x *vacancyPost, m int64) bool {
	r := w.res
	region := w.regionOf(&w.agents[x.employer])
	for _, d := range r.drawsOf[x.recipe] {
		n, ok := checked.Mul(m, d.quantity)
		if !ok || n > r.snapStock[r.cellOf(region, d.resource)] {
			return true
		}
	}
	return false
}

// formHire forms the hire of a pair the round matched, atomically
// (spec-employment.md 5.5): the bill moves from the employer to an Escrow account,
// the record is made with the next hire ID, the worker's own request is withdrawn
// and a hired request is made in its place, and HireFormed is emitted. Everything
// it needs was checked and the bill is held, so nothing here can fail.
func (w *World) formHire(g RegionID, l *labourLane, h payroll.Hire) {
	e := w.employment
	worker, post := &w.agents[l.workers[h.Offer]], &l.posts[h.Vacancy]
	employer := &w.agents[post.employer]
	ticks := w.cfg.Recipes[post.recipe].Duration
	bill, ok := payroll.Bill(h.Wage, ticks, h.Households)
	if !ok {
		invariant("the bill of a checked hire does not fit")
	}
	escrow := w.escrowTake(g)
	if err := w.ledger.Transfer(employer.account, escrow, bill, ledger.WageEscrow); err != nil {
		invariant("the ledger refused the escrow of a checked hire: " + err.Error())
	}
	rec := hireRec{id: e.next, region: g, employer: employer.id, worker: worker.id, recipe: post.recipe, escrow: escrow,
		wage: h.Wage, k: h.Households, ticks: ticks, formed: w.Tick(), reserve: Price(h.Reserve), bid: Price(h.Bid)}
	e.next++
	e.hires = append(e.hires, rec)
	e.grow(max(worker.id, employer.id))
	e.workerOf[worker.id] = rec.id + 1
	e.hired[employer.id] += rec.k
	e.hiredN[employer.id]++
	e.open[g]++
	w.withdrawOwnRequest(worker)
	ask := jobAsk{recipe: rec.recipe, maximum: rec.k, k0: rec.k, set: true}
	ask.first, ask.draws = w.recordDraws(worker, employer.id, g, rec.recipe, rec.k)
	w.pr.asks[worker.id] = ask
	w.reportHireFormed(&rec, worker, employer, bill)
}

// withdrawOwnRequest takes back the request the worker made for itself at P2.5: it
// does not start, and its draws on an endowment leave the ration of P2.6
// (spec-employment.md 4.1.1, 5.5).
func (w *World) withdrawOwnRequest(a *agent) {
	ask := &w.pr.asks[a.id]
	if !ask.set {
		return
	}
	if w.res != nil {
		w.res.withdraw(ask.first, ask.draws)
	}
	ask.set = false
}
