package economysim

import (
	"errors"

	"github.com/Whealinghold/economy-sim/internal/checked"
	crd "github.com/Whealinghold/economy-sim/internal/credit"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// This file is the claims in birth of the credit module (the continuation of
// credit.go, architecture.md 2.6; spec-credit.md 5 and 6): the collection of the
// requests and offers of the plans, the admission, the round of P3.3b by lanes and
// the formation of a loan. It takes no word of the stream: every tie is the stateless
// key K(t, agent) (FR-CRD-062). The arithmetic and the matching are internal/credit.

// subRequest and subOffer are a request and an offer as a plan returned them, with
// the agent, kept for the round of P3.3b (the plan is reused by the next agent).
type subRequest struct {
	agent AgentID
	r     CreditRequest
}

type subOffer struct {
	agent AgentID
	o     CreditOffer
}

// loanRequest and loanOffer are an admitted request and offer, found by the Ref of
// the row of the lane that holds its principal, rate, term and key.
type loanRequest struct {
	agent  AgentID
	region RegionID
}

type loanOffer struct {
	agent  AgentID
	region RegionID
}

// crdLane is the admitted requests and offers of one region, the lane of the round
// (spec-credit.md 6.2).
type crdLane struct {
	requests, offers []crd.Row
}

// creditCount is what CreditRound reports for one region, and the counter of the
// price control of an ORG cap (spec-credit.md 5.2.3), which no event carries yet.
type creditCount struct {
	requests, offers, requestsRefused, offersRefused int64
	inArrears, exposure                              int64
	formed, principal, interest, unfilled            int64
	tooMany, overflow, priceControlled               int64
}

// quiet reports whether every count that CreditRound carries is 0.
func (c creditCount) quiet() bool {
	return c == creditCount{priceControlled: c.priceControlled}
}

// roundScratch is the storage of the round: the submitted requests and offers, the
// admitted ones, the lanes and the counts per region, reused every tick. halted says
// the claims limit ended the round and capHit that a fill was dropped for it this
// tick (the run of the Warn "claim cap reached").
type roundScratch struct {
	reqs   []subRequest
	offs   []subOffer
	rq     []loanRequest
	of     []loanOffer
	lanes  []crdLane
	counts []creditCount
	halted bool
	capHit bool
	form   func(crd.Fill) crd.Outcome
}

// reset empties the round for the next tick, keeping the storage.
func (rd *roundScratch) reset() {
	rd.reqs, rd.offs, rd.rq, rd.of = rd.reqs[:0], rd.offs[:0], rd.rq[:0], rd.of[:0]
	for i := range rd.lanes {
		rd.lanes[i].requests, rd.lanes[i].offers = rd.lanes[i].requests[:0], rd.lanes[i].offers[:0]
	}
	clear(rd.counts)
	rd.halted, rd.capHit = false, false
}

// checkCreditPlan is the plan check's part for the request and the offer of agent a
// (the checkTradePlan fan-in, hook 3 of plan 3-crd): with the module off each is
// refused at once as credit_off, a Debug line and no event, else it is kept for the
// round of P3.3b. It is nil-safe, so the fan-in holds no branch on the module.
func (w *World) checkCreditPlan(a *agent, p *TradePlan) {
	if len(p.Borrow) == 0 && len(p.Lend) == 0 {
		return
	}
	if w.credit == nil {
		for range p.Borrow {
			w.logCreditRefused("request refused", a.id, ReasonCreditOff)
		}
		for range p.Lend {
			w.logCreditRefused("offer refused", a.id, ReasonCreditOff)
		}
		return
	}
	rd := &w.credit.round
	for _, r := range p.Borrow {
		rd.reqs = append(rd.reqs, subRequest{agent: a.id, r: r})
	}
	for _, o := range p.Lend {
		rd.offs = append(rd.offs, subOffer{agent: a.id, o: o})
	}
}

// p33bRound is P3.3b (spec-credit.md 6): the admission of the requests and offers of
// the plans in agent order, the lanes in ascending region with the order and the
// matching of each, the formation of each fill, then CreditRound per region. It
// runs after every plan and takes no word. The run of the Warn "claim cap reached"
// advances every tick.
func (w *World) p33bRound() {
	cr := w.credit
	rd := &cr.round
	if len(rd.reqs)+len(rd.offs) > 0 {
		w.creditPrices()
		w.admitCreditRequests()
		w.admitCreditOffers()
		w.matchCreditLanes()
		w.reportRound()
	}
	run, warn := crd.NextRun(cr.capRun, rd.capHit)
	cr.capRun = run
	if warn {
		w.logCapReached()
	}
	rd.reset()
}

// admitCreditRequests admits or refuses each request in agent order, a refusal whole with
// its reason (spec-credit.md 5.2), and puts an admitted one in the lane of its agent
// with its limit, after the rate cap of ORG replaced a limit above it (5.2.3).
func (w *World) admitCreditRequests() {
	cr := w.credit
	rd, t := &cr.round, w.Tick()
	for i := range rd.reqs {
		s := &rd.reqs[i]
		a := &w.agents[s.agent]
		g := w.regionOf(a)
		c := &rd.counts[g]
		if reason := w.creditRequestRefusal(a, s); reason != "" {
			c.refuseRequest(reason)
			w.logCreditRefused("request refused", s.agent, reason)
			continue
		}
		limit := cr.capped(s.r.MaxRateBP, g, c)
		c.requests++
		rd.lanes[g].requests = append(rd.lanes[g].requests, crd.Row{
			Ref: len(rd.rq), Agent: int64(s.agent), Principal: s.r.Principal, Term: s.r.Term, RateBP: limit,
			Key: ordering.Key(t, int64(s.agent)),
		})
		rd.rq = append(rd.rq, loanRequest{agent: s.agent, region: g})
	}
}

// refuseRequest counts a request refused at admission.
func (c *creditCount) refuseRequest(reason Reason) {
	c.requestsRefused++
	switch reason {
	case ReasonInArrears:
		c.inArrears++
	case ReasonExposure:
		c.exposure++
	}
}

// creditRequestRefusal is the first failing check of 5.2 for a request, in the order of the
// table, or "" when it is admitted. The module is on, so credit_off is not one.
func (w *World) creditRequestRefusal(a *agent, s *subRequest) Reason {
	t := w.cfg.Roles[a.role].Credit
	if t == nil || t.Borrow == nil {
		return ReasonNotBorrower
	}
	if r := w.shapeRefusal(s.r.Principal, s.r.Term, s.r.MaxRateBP); r != "" {
		return r
	}
	if rq := w.credit.round.rq; len(rq) > 0 && rq[len(rq)-1].agent == s.agent {
		return ReasonDuplicate // the requests are in agent order, so the last admitted is the agent's
	}
	f := w.figuresOf(a.account)
	if f.Overdue > 0 {
		return ReasonInArrears
	}
	if s.r.Principal > w.creditRoomAt(t.Borrow, w.creditUpkeep(a), f.Debt) {
		return ReasonExposure // refused whole, never cut to the room (5.2.2)
	}
	return ""
}

// shapeRefusal is rows 3 to 5 of the table of 5.2 for a principal, a term and a rate:
// the bounds of the module and of the configuration.
func (w *World) shapeRefusal(principal, term, rate int64) Reason {
	switch {
	case principal < w.credit.cfg.MinPrincipal || principal > MaxMoney:
		return ReasonInvalidPrincipal
	case term < 1 || term > MaxCreditTerm:
		return ReasonInvalidTerm
	case rate < 0 || rate > MaxCreditRateBP:
		return ReasonInvalidRate
	}
	return ""
}

// admitCreditOffers admits or refuses each offer in agent order and puts an admitted one in
// its lane with its effective rate, the posted rate at the lane's scale (6.3.1).
func (w *World) admitCreditOffers() {
	cr := w.credit
	rd, t := &cr.round, w.Tick()
	for i := range rd.offs {
		s := &rd.offs[i]
		a := &w.agents[s.agent]
		g := w.regionOf(a)
		c := &rd.counts[g]
		if reason := w.creditOfferRefusal(a, s); reason != "" {
			c.offersRefused++
			w.logCreditRefused("offer refused", s.agent, reason)
			continue
		}
		rate, ok := crd.Effective(s.o.RateBP, cr.scale[g], MaxCreditRateBP)
		if !ok {
			invariant("a negative rate or scale in an admitted offer")
		}
		c.offers++
		rd.lanes[g].offers = append(rd.lanes[g].offers, crd.Row{
			Ref: len(rd.of), Agent: int64(s.agent), Principal: s.o.Principal, Term: s.o.MaxTerm, RateBP: rate,
			Key: ordering.Key(t, int64(s.agent)),
		})
		rd.of = append(rd.of, loanOffer{agent: s.agent, region: g})
	}
}

// creditOfferRefusal is the first failing check of 5.2 for an offer, or "" when it is
// admitted: the lender's available money is its balance less what its orders of the
// tick reserved (5.2.1).
func (w *World) creditOfferRefusal(a *agent, s *subOffer) Reason {
	t := w.cfg.Roles[a.role].Credit
	if t == nil || t.Lend == nil {
		return ReasonNotLender
	}
	if r := w.shapeRefusal(s.o.Principal, s.o.MaxTerm, s.o.RateBP); r != "" {
		return r
	}
	if of := w.credit.round.of; len(of) > 0 && of[len(of)-1].agent == s.agent {
		return ReasonDuplicate
	}
	if v := w.accountView(a.account); s.o.Principal > v.Balance-v.Reserved {
		return ReasonInsufficientFunds
	}
	return ""
}

// matchCreditLanes runs the order and the matching of each lane in ascending region, and
// counts what each region's admitted requests did not get. A round the claims limit
// ended leaves the later lanes unmatched and their requests unfilled.
func (w *World) matchCreditLanes() {
	cr := w.credit
	rd := &cr.round
	if rd.form == nil {
		rd.form = w.formLoan
	}
	for g := range rd.lanes {
		l := &rd.lanes[g]
		if len(l.requests) > 0 && !rd.halted {
			crd.SortOffers(l.offers)
			crd.SortRequests(l.requests)
			rd.halted = crd.Match(l.requests, l.offers, cr.cfg.MinPrincipal, rd.form)
		}
		for i := range l.requests {
			rd.counts[g].unfilled = addSat(rd.counts[g].unfilled, l.requests[i].Principal)
		}
	}
}

// addSat is a plus b for a count of money, saturating at MaxMoney: the sum over a
// region of many principals near MaxMoney must not wrap in a report.
func addSat(a, b int64) int64 {
	if s, ok := checked.Add(a, b); ok && s <= MaxMoney {
		return s
	}
	return MaxMoney
}

// formLoan forms the loan of one tentative fill (spec-credit.md 6.4), in this order:
// the claims limit ends the round, an interest or a sum above MaxMoney drops the fill,
// then the ledger lends. The ledger changes only for a loan that forms.
func (w *World) formLoan(f crd.Fill) crd.Outcome {
	cr := w.credit
	rd := &cr.round
	rq, of := rd.rq[f.Request], rd.of[f.Offer]
	c := &rd.counts[rq.region]
	if w.ledger.OpenClaims() >= cr.cfg.claimsLimit() {
		c.tooMany++
		rd.capHit = true
		return crd.Halted
	}
	interest, ok := crd.Interest(f.Principal, f.RateBP, f.Term, w.cfg.Calendar.TicksPerYear())
	owed, fits := checked.Add(f.Principal, interest)
	if !ok || !fits || owed > MaxMoney {
		c.overflow++
		return crd.Dropped
	}
	due := w.Tick() + f.Term
	id, err := w.ledger.Lend(w.agents[of.agent].account, w.agents[rq.agent].account, f.Principal, interest, f.RateBP, due)
	if errors.Is(err, ledger.ErrBalanceOverflow) {
		c.overflow++
		return crd.Dropped
	}
	mustClaim(err, "a loan")
	claim, found := w.ledger.Claim(id)
	if !found {
		invariant("a formed claim is not in the ledger's list")
	}
	cr.idx.add(claim)
	c.formed++
	c.principal = addSat(c.principal, f.Principal)
	c.interest = addSat(c.interest, interest)
	w.reportFormed(rq.region, &claim, of.agent, rq.agent)
	return crd.Formed
}

// capped returns the limit a request of a borrower in region g is admitted with, after the
// rate cap of the institutions module replaced one above it (spec-credit.md 5.2.3), and counts
// the replacement in c. A nil seam changes nothing.
func (cr *credit) capped(limit int64, g RegionID, c *creditCount) int64 {
	if cr.rateCap == nil {
		return limit
	}
	replaced, lowered := cr.rateCap(g, limit)
	if !lowered {
		return limit
	}
	if replaced < 0 {
		invariant("a negative rate cap from the rate limit seam")
	}
	c.priceControlled++
	return replaced
}
