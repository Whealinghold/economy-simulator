package economysim

import "github.com/Whealinghold/economy-sim/internal/checked"

// This file is the strategy side of the credit module (spec-credit.md 5.1, 10): the
// request and the offer a plan carries, the group of the view, the filler of that
// group and the default rules of the borrower and the lender, exported so that a
// wrapper strategy can call them (architecture.md 4.7).

// CreditRequest asks for a loan. The principal is a most: the round may fill it in
// parts, from several lenders (spec-credit.md 5.1).
type CreditRequest struct {
	// Principal is the money in minor units, at least the module's MinPrincipal.
	Principal int64
	// Term is the ticks until the whole falls due, 1 to MaxCreditTerm.
	Term int64
	// MaxRateBP is the highest yearly rate the request accepts, 0 to MaxCreditRateBP.
	MaxRateBP int64
}

// CreditOffer offers money on a term at a posted rate. The principal is a most
// (spec-credit.md 5.1).
type CreditOffer struct {
	// Principal is the money in minor units, at least the module's MinPrincipal.
	Principal int64
	// MaxTerm is the longest term the offer lends for, 1 to MaxCreditTerm.
	MaxTerm int64
	// RateBP is the posted yearly rate, 0 to MaxCreditRateBP.
	RateBP int64
}

// CreditView is the credit group of the view of an agent whose role has a credit
// block (spec-credit.md 10.1): plain values valid for the call only. It holds no
// other agent's claim and no figure of the market for credit.
type CreditView struct {
	// Lend and Borrow are a copy of the role's block; one of them is nil.
	Lend   *LendTerms
	Borrow *BorrowTerms
	// MinPrincipal is the module's smallest principal.
	MinPrincipal int64
	// UpkeepCost is the agent's upkeep cost U, weight included, at the prices of its
	// region (spec-credit.md 5.3.1).
	UpkeepCost int64
	// Debt is the principal owed as debtor, Owed the principal and interest owed,
	// Overdue the owed on the overdue claims (0 when none) and NextDue the earliest
	// due tick of its claims as debtor (-1 when it has none).
	Debt, Owed, Overdue, NextDue int64
	// CreditRoom is the exposure room of 5.3.3, 0 for a lender.
	CreditRoom int64
	// Lent is the principal owed to the agent as creditor and Receivable the
	// principal and interest owed to it.
	Lent, Receivable int64
	// ScaleBP is the rate scale of the agent's region in basis points.
	ScaleBP int64
}

// creditScratch is the storage of the group of the view: the copy of the role's
// block and the group itself, reused by every agent (a view is valid for its call).
type creditScratch struct {
	view   CreditView
	lend   LendTerms
	borrow BorrowTerms
}

// fillCreditView sets the credit group of the view passed to Trade for an agent whose
// role has a block, from the claim indexes and the prices of the tick (hook 2 of
// plan 3-crd). It is the filler of the fillTradeOnly fan-in and draws nothing.
func (w *World) fillCreditView(v *AgentView, a *agent) {
	t := w.cfg.Roles[a.role].Credit
	if t == nil {
		return
	}
	cr := w.credit
	w.creditPrices()
	f, u := w.figuresOf(a.account), w.creditUpkeep(a)
	s := &cr.vs
	s.view = CreditView{
		MinPrincipal: cr.cfg.MinPrincipal, UpkeepCost: u, Debt: f.Debt, Owed: f.Owed, Overdue: f.Overdue,
		NextDue: f.NextDue, Lent: f.Lent, Receivable: f.Receivable, ScaleBP: cr.scale[w.regionOf(a)],
	}
	if t.Lend != nil {
		s.lend, s.view.Lend = *t.Lend, &s.lend
	}
	if t.Borrow != nil {
		s.borrow, s.view.Borrow = *t.Borrow, &s.borrow
		s.view.CreditRoom = w.creditRoomAt(t.Borrow, u, f.Debt)
	}
	v.Credit = &s.view
}

// creditPrices values the prices of the tick for the upkeep cost once per tick: the
// view of every agent and the admission of the round read the same figure, and the
// valuation is a pass over every market (valuePrices is idempotent within a tick).
func (w *World) creditPrices() {
	cr := w.credit
	if cr.valuedAt != w.Tick()+1 {
		w.valuePrices()
		cr.valuedAt = w.Tick() + 1
	}
}

// DefaultCredit appends to the plan the request or the offer the default strategy
// makes for the credit block of the view (spec-credit.md 10.2, 10.3): a borrower's
// request from its cash gap, a lender's offer from the money its own bids left. It
// takes no random word. tier2 is the cost of the inputs and tools of the next job at
// the limits of the plan and duration the duration of the recipe it plans (0 for
// none); both are what the default Trade knows and the view alone does not hold.
func DefaultCredit(v *AgentView, p *TradePlan, tier2, duration int64) {
	if r, ok := DefaultBorrow(v, tier2, duration); ok {
		p.Borrow = append(p.Borrow, r)
	}
	if o, ok := DefaultLend(v, planBids(p)); ok {
		p.Lend = append(p.Lend, o)
	}
}

// DefaultBorrow is the default borrower of spec-credit.md 10.2, a pure function of
// the view and of the two figures of the default Trade: it asks for the gap between
// the money the household holds and the upkeep and inputs of the horizon of its job,
// capped by its exposure room, and asks for nothing when there is no job, no gap or
// the ask is below the minimum principal. FR-CRD-051.
func DefaultBorrow(v *AgentView, tier2, duration int64) (CreditRequest, bool) {
	cv := v.Credit
	if cv == nil || cv.Borrow == nil {
		return CreditRequest{}, false
	}
	j, late := duration, int64(0)
	switch {
	case v.HasJob:
		j = v.Remaining
	case v.Money-cv.UpkeepCost < tier2:
		late = 1 // the inputs cannot be bought at once (10.2.1.2)
	}
	if j <= 0 {
		return CreditRequest{}, false
	}
	var need sum
	var hz sum // the one is the rule constant of the phase order (10.2.1.1)
	hz.add(j, true)
	hz.add(1+late, true)
	hz.add(cv.Borrow.SlackTicks, true)
	if hz.bad {
		return CreditRequest{}, false // a hand-built view; a world's Remaining is far below
	}
	h := hz.v
	need.add(checked.Mul(cv.UpkeepCost, h))
	need.add(tier2, true)
	need.add(checked.Sub(0, v.Money))
	if need.bad {
		return CreditRequest{}, false
	}
	ask := min(need.v, cv.CreditRoom, MaxMoney)
	if ask < cv.MinPrincipal {
		return CreditRequest{}, false
	}
	return CreditRequest{Principal: ask, Term: h, MaxRateBP: cv.Borrow.MaxRateBP}, true
}

// DefaultLend is the default lender of spec-credit.md 10.3: a share of what its cash
// has left after its bids and the upkeep it keeps, offered at its block's posted
// rate. bids is the sum of limit times quantity of the bids of the plan it has just
// built. It offers nothing when the offer is below the minimum principal.
// FR-CRD-052.
func DefaultLend(v *AgentView, bids int64) (CreditOffer, bool) {
	cv := v.Credit
	if cv == nil || cv.Lend == nil {
		return CreditOffer{}, false
	}
	l := cv.Lend
	keep, ok := checked.Mul(l.KeepTicks, cv.UpkeepCost)
	if !ok {
		return CreditOffer{}, false
	}
	lendable, fits := checked.Sub(v.Money, bids)
	if fits {
		lendable, fits = checked.Sub(lendable, keep)
	}
	if !fits {
		return CreditOffer{}, false // a hand-built view whose figures wrap: nothing is lendable
	}
	lendable = max(0, lendable)
	offer, ok := checked.MulDiv(l.ShareBP, lendable, wholeBP)
	if !ok || offer < cv.MinPrincipal {
		return CreditOffer{}, false
	}
	return CreditOffer{Principal: min(offer, MaxMoney), MaxTerm: l.MaxTermTicks, RateBP: l.RateBP}, true
}

// planBids is the cash the bids of a plan reserve: the sum of limit times quantity of
// its bid intents, saturating at MaxMoney (a plan whose bids overflow leaves nothing
// to lend).
func planBids(p *TradePlan) int64 {
	var s sum
	for i := range p.Intents {
		if in := &p.Intents[i]; in.Side == Bid {
			s.add(checked.Mul(int64(in.Limit), int64(in.Quantity)))
		}
	}
	if s.bad {
		return MaxMoney
	}
	return s.v
}

// tier2Cost is the cost of the inputs and tools of the next job at the limits of the
// plan, whatever money allows (spec-credit.md 10.2): the units of tier 2 of each
// bid at its limit price, summed. It is read before the money stack lowers a limit,
// and is 0 for an agent with no credit block. An overflow reads as the largest value.
func (t *tradeCtx) tier2Cost() int64 {
	if t.v.Credit == nil {
		return 0
	}
	var s sum
	for i := range t.rows {
		if r := &t.rows[i]; r.kind == bidOrder && r.limit >= 1 {
			s.add(checked.Mul(r.t2, r.limit))
		}
	}
	if s.bad {
		return MaxMoney
	}
	return s.v
}

// plannedDuration is the duration of the recipe the agent plans to run next, 0 when
// it plans none (spec-credit.md 10.2: J of a job to start).
func (t *tradeCtx) plannedDuration() int64 {
	if t.rstar < 0 {
		return 0
	}
	return t.v.Recipes[t.rstar].Duration
}
