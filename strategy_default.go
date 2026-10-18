package economysim

import (
	"math"
	"math/rand/v2"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/household"
)

// DefaultStrategy is the strategy of a household: it works when its welfare
// ratio says so, chooses the recipe that earns the most per tick at its own
// price beliefs, buys what it needs in three tiers of priority (upkeep, the
// inputs of its next job, then a buffer that a spend cap limits) and sells what
// it holds above what it keeps. It has no fields and keeps no state; its
// tunables arrive in the view, so a consumer can call it with a view of its own
// making, and a strategy of the consumer's may hold one in a named field and
// delegate (spec-agents.md 7). It takes no random word in Work and Revise and
// one word per order it considers in Trade.
type DefaultStrategy struct{}

var _ Strategy = DefaultStrategy{}

// tradeScratch is the per-commodity working state of Trade.
type tradeScratch struct {
	rows []tradeRow
}

// reset returns n zeroed rows, growing the backing array only when it is short.
func (s *tradeScratch) reset(n int) []tradeRow {
	if cap(s.rows) < n {
		s.rows = make([]tradeRow, n)
	}
	s.rows = s.rows[:n]
	clear(s.rows)
	return s.rows
}

// The kinds of order a commodity can take in Trade (the candidates of
// spec-agents.md 7.3.5, decided by the view alone).
const (
	noOrder uint8 = iota
	bidOrder
	askOrder
)

// tradeRow is one commodity's position: the order it is a candidate for, its
// targets, its tiers and, while the money is allocated, the order's one limit
// and the quantity bought so far.
type tradeRow struct {
	kind            uint8
	bought          bool
	purpose         Purpose
	reserve, target int64
	t1, t2, t3      int64
	ask             int64
	limit, qty      int64
	windowPos       int64
	unitCost        int64
	commodity       CommodityID
}

// tradeCtx is what the passes of Trade share.
type tradeCtx struct {
	v     *AgentView
	par   household.Params
	bp    beliefs.Params
	rows  []tradeRow
	rstar int
}

// beliefParamsOf converts the view's belief group to the parameters of the
// belief rule at the call (architecture.md 2.9). MaxPrice is a root bound.
func beliefParamsOf(t *BeliefTunables) beliefs.Params {
	return beliefs.Params{
		Step: t.Step, AdjustProb: t.AdjustProb, Significance: t.Significance,
		EvidenceShare: t.EvidenceShare, StockLow: t.StockLow, StockHigh: t.StockHigh,
		MaxAge: t.MaxAge, WidthInit: t.WidthInit, WidthMin: t.WidthMin,
		WidthMax: t.WidthMax, WidthStep: t.WidthStep, CostFloor: t.CostFloor,
		CostCeiling: t.CostCeiling, RefBandLow: t.RefBandLow, RefBandHigh: t.RefBandHigh,
		MaxPrice: MaxPrice,
	}
}

// Work requests the job of the work report (7.1): none when the agent rests,
// when every recipe is halted, cannot run or is not worth running, otherwise
// the best eligible recipe and as many executions as the work share asks of the
// weight, cut to what the recipe and the food in stock allow. It takes no
// random word.
func (DefaultStrategy) Work(v *AgentView, _ *rand.Rand) JobRequest {
	buf := workReportBuf{noList: true}
	workReport(v, &buf)
	if buf.chosen < 0 {
		return JobRequest{NoJob: true}
	}
	return JobRequest{Recipe: v.Recipes[buf.chosen].Recipe, Executions: buf.executions}
}

// Revise applies the belief update of spec-beliefs.md 5 to the outcome with the
// parameters of the view and the word the world drew, and returns the new
// belief (7.6, FR-AGT-050). The adjustment gate's result is the world's own.
func (DefaultStrategy) Revise(o *OutcomeView, _ *rand.Rand) Belief {
	b, _ := beliefs.Update(
		beliefs.Belief{Centre: int64(o.Belief.Centre), Width: o.Belief.Width},
		beliefs.Bounds{Floor: int64(o.Bounds.Floor), Ceiling: int64(o.Bounds.Ceiling)},
		beliefs.Outcome{Bid: o.Side == Bid, Limit: int64(o.Limit), Quantity: int64(o.Quantity), Filled: int64(o.Filled)},
		beliefs.Quote{
			Price: o.Market.Price, Age: o.Market.Age, BestBid: o.Market.BestBid, BestAsk: o.Market.BestAsk,
			HasBid: o.Market.HasBestBid, HasAsk: o.Market.HasBestAsk,
		},
		o.Stock, int64(o.Target), beliefParamsOf(&o.Params), o.Draw)
	return Belief{Centre: Price(b.Centre), Width: b.Width}
}

// Trade appends the agent's orders to the plan, at most one per commodity and
// in canonical commodity order (7.2 to 7.5). It takes one random word for each
// order it considers, in that order and before it allocates any money, so the
// number of draws is a function of the view alone. It discards nothing.
//
// A view that makes a checked operation overflow, which a world's view cannot,
// gives no order for the commodity (or none at all if the upkeep cost
// overflows) and never a panic (D96).
func (DefaultStrategy) Trade(v *AgentView, rng *rand.Rand, p *TradePlan) {
	var local viewScratch
	sc := v.scratch
	if sc == nil {
		sc = &local
	}
	t := tradeCtx{v: v, par: householdParams(&v.Tunables), bp: beliefParamsOf(&v.Tunables.Beliefs), rows: sc.trade.reset(len(v.Commodities))}
	cost, ok := v.upkeepCost()
	if !ok {
		return
	}
	savings, ok := household.ReserveMoney(cost, t.par.ReserveTicks)
	if !ok {
		return
	}
	t.rstar = t.plannedRecipe()
	for i := range v.Commodities {
		t.consider(i, rng)
	}
	t.allocate(cost, savings)
	t.emit(p)
}

// plannedRecipe is the index of the recipe the agent expects to run next
// (7.2): the best by value per tick among those with a positive yield that are
// worth running or a source and not halted, ignoring stock; -1 if none.
func (t *tradeCtx) plannedRecipe() int {
	best, bestNet, bestDur := -1, int64(0), int64(0)
	for i := range t.v.Recipes {
		r := &t.v.Recipes[i]
		if r.Effective <= 0 || t.v.recipeHalted(r, t.par) {
			continue
		}
		net, ok := t.v.recipeNet(r)
		if !ok || !household.Eligible(net, len(r.Inputs) == 0) {
			continue
		}
		if best < 0 || household.Better(net, r.Duration, bestNet, bestDur) {
			best, bestNet, bestDur = i, net, r.Duration
		}
	}
	return best
}

// sum adds checked terms and remembers an overflow.
type sum struct {
	v   int64
	bad bool
}

func (s *sum) add(x int64, ok bool) {
	if s.bad || !ok {
		s.bad = true
		return
	}
	var fits bool
	s.v, fits = checked.Add(s.v, x)
	s.bad = !fits
}

// jobTerm returns Job_c of the commodity for the planned recipe (7.3): the
// inputs and tools of the next job when it starts next tick (StartsIn is 1).
// ok is false on overflow.
func (t *tradeCtx) jobTerm(c CommodityID) (int64, bool) {
	r, v := &t.v.Recipes[t.rstar], t.v
	if v.StartsIn != 1 {
		return 0, true
	}
	var j sum
	for _, in := range r.Inputs {
		if in.Commodity == c {
			jobs, ok1 := checked.Mul(t.par.InputJobs, v.Weight)
			n, ok2 := checked.Mul(jobs, in.Quantity)
			j.add(n, ok1 && ok2)
		}
	}
	for _, tl := range r.Tools {
		if tl.Commodity == c {
			j.add(v.Weight, true)
		}
	}
	return j.v, !j.bad
}

// outputTerm returns OutTgt_c of the commodity for the planned recipe (7.3): the
// output target of a non-discard output. ok is false on overflow.
func (t *tradeCtx) outputTerm(c CommodityID) (int64, bool) {
	r, v := &t.v.Recipes[t.rstar], t.v
	var o sum
	for _, out := range r.Outputs {
		if out.Commodity == c && !out.Discard {
			o.add(household.OutputTarget(v.Weight, out.Quantity, r.Effective, t.par.OutputCoverTicks, r.Duration))
		}
	}
	return o.v, !o.bad
}

// jobAndOutput returns Job_c and OutTgt_c of the commodity for the planned
// recipe (7.3); both are 0 when no recipe is planned. ok is false on overflow.
func (t *tradeCtx) jobAndOutput(c CommodityID) (job, out int64, ok bool) {
	if t.rstar < 0 {
		return 0, 0, true
	}
	job, ok1 := t.jobTerm(c)
	out, ok2 := t.outputTerm(c)
	return job, out, ok1 && ok2
}

// makes reports whether the role makes c: a non-discard output of one of its
// recipes (7.3.5).
func (t *tradeCtx) makes(c CommodityID) bool {
	for i := range t.v.Recipes {
		for _, o := range t.v.Recipes[i].Outputs {
			if o.Commodity == c && !o.Discard {
				return true
			}
		}
	}
	return false
}

// upkeepUnits returns U_c, the units of c the agent eats this tick, and whether
// c is an upkeep commodity. ok is false on overflow.
func (t *tradeCtx) upkeepUnits(c CommodityID) (u int64, isUpkeep, ok bool) {
	for _, row := range t.v.Upkeep {
		if row.Commodity == c {
			u, ok = checked.Mul(t.v.Weight, row.Quantity)
			return u, true, ok
		}
	}
	return 0, false, true
}

// purposeOf is the Purpose of an intent for c (7.3.6): an input of the planned
// recipe with a job to buy for, a tool of it, an upkeep commodity, else other.
func (t *tradeCtx) purposeOf(c CommodityID, job int64, isUpkeep bool) Purpose {
	if t.rstar >= 0 {
		r := &t.v.Recipes[t.rstar]
		if job > 0 {
			for _, in := range r.Inputs {
				if in.Commodity == c {
					return PurposeInput
				}
			}
		}
		for _, tl := range r.Tools {
			if tl.Commodity == c {
				return PurposeTool
			}
		}
	}
	if isUpkeep {
		return PurposeUpkeep
	}
	return PurposeOther
}

// candidate says which order the commodity is a candidate for (7.3.5): a bid
// for a gap, else an ask for a surplus or for the units that expire beyond the
// reserve, except a tool the role does not make. ask is the ask quantity.
func (t *tradeCtx) candidate(c *CommodityView, gap, surplus, expiring int64) (kind uint8, ask int64) {
	switch {
	case gap > 0:
		return bidOrder, 0
	case (surplus > 0 || expiring > 0) && (!c.Tool || t.makes(c.Commodity)):
		return askOrder, max(surplus, expiring)
	}
	return noOrder, 0
}

// consider is the first pass for the commodity at index i: its targets, its
// tiers and the order it is a candidate for, and, for a candidate, the one
// random word and the limit drawn with it (7.3.5, 7.5.3). The candidate is
// decided by the view alone, never by affordability.
func (t *tradeCtx) consider(i int, rng *rand.Rand) {
	c, r := &t.v.Commodities[i], &t.rows[i]
	// Untested not-ok branch of upkeepUnits: Trade has already returned if the
	// same weight * quantity products overflowed in upkeepCost.
	u, isUpkeep, ok1 := t.upkeepUnits(c.Commodity)
	job, out, ok2 := t.jobAndOutput(c.Commodity)
	if !ok1 || !ok2 {
		return
	}
	tg, ok := household.ComputeTargets(household.Need{Upkeep: u, Job: job, OutTgt: out}, t.par)
	if !ok {
		return
	}
	gap, surplus, t1, t2, t3 := household.Tiers(tg.Keep, c.Stock, tg.Reserve, job)
	kind, ask := t.candidate(c, gap, surplus, max(c.Expiring, 0)-tg.Reserve)
	if kind == noOrder {
		return
	}
	r.kind, r.ask = kind, ask
	word := rng.Uint64()
	if r.kind == bidOrder && t.v.Hunger > 0 && t1 > 0 {
		// The urgent bid takes the top of the range; the word was still drawn.
		word = math.MaxUint64
	}
	limit := beliefs.Draw(
		beliefs.Belief{Centre: int64(c.Belief.Centre), Width: c.Belief.Width},
		beliefs.Bounds{Floor: int64(c.Bounds.Floor), Ceiling: int64(c.Bounds.Ceiling)}, t.bp, word)
	r.commodity, r.purpose, r.reserve, r.target = c.Commodity, t.purposeOf(c.Commodity, job, isUpkeep), tg.Reserve, tg.Target
	r.t1, r.t2, r.t3, r.limit, r.unitCost = t1, t2, t3, min(limit, MaxPrice), int64(c.UnitCost)
	r.windowPos = household.EmptyPos
	if c.Window.HasWindow {
		r.windowPos = c.Window.Position
	}
}

// combineTiers spends avail on up to want units of one tier of the order, at its
// one limit, and returns the money left (7.5.3, D98). It is the whole
// cross-tier rule for tiers 1 and 2; tier 3 (buyBuffer) never lowers a limit. The first tier of the order to buy may
// have the limit lowered to the money by household.Afford; a later tier that
// cannot pay one unit at the order's limit is dropped, never lowered. The money
// charged is limit times quantity at the limit in force, so the order's
// reservation never exceeds what was charged.
func (r *tradeRow) combineTiers(avail, want int64) int64 {
	if r.limit < 1 || want < 1 {
		return avail
	}
	if r.bought {
		q := min(want, avail/r.limit)
		r.qty += q
		return avail - q*r.limit
	}
	limit, q := household.Afford(avail, r.limit, want)
	if q < 1 {
		return avail
	}
	r.limit, r.bought = limit, true
	r.qty += q
	return avail - limit*q
}

// allocate is the money stack (7.4.3): tier 1 over all bids, tier 2, then tier 3
// scaled by favourability and bounded by the spend cap, which limits the
// quantity and never the limit. savings is the reserve money.
func (t *tradeCtx) allocate(cost, savings int64) {
	avail := max(t.v.Money, 0)
	for i := range t.rows {
		if r := &t.rows[i]; r.kind == bidOrder {
			avail = r.combineTiers(avail, r.t1)
		}
	}
	for i := range t.rows {
		if r := &t.rows[i]; r.kind == bidOrder {
			avail = r.combineTiers(avail, r.t2)
		}
	}
	if spendCap, ok := household.SpendCap(household.Excess(avail, savings), cost, t.par); ok {
		t.buffer(avail, spendCap)
	}
}

// buffer is tier 3: each bid buys its buffer want scaled by the buy share, at
// its limit, bounded in quantity by the money left and by the cap that all the
// bids share (FR-AGT-042). It never lowers a limit.
func (t *tradeCtx) buffer(avail, spendCap int64) {
	for i := range t.rows {
		r := &t.rows[i]
		if r.kind != bidOrder || r.limit < 1 {
			continue
		}
		// Untested not-ok branch: the buy share is at most Basis, so the product
		// over Basis is at most t3 and fits; the check keeps the loop total.
		want, ok := checked.MulDiv(r.t3, household.BuyShare(r.windowPos, t.par), household.Basis)
		if !ok {
			continue
		}
		q := min(want, min(avail, spendCap)/r.limit)
		if q < 1 {
			continue
		}
		r.qty, r.bought = r.qty+q, true
		avail -= q * r.limit
		spendCap -= q * r.limit
	}
}

// emit appends the intents, in canonical commodity order (D86): a bid for the
// quantity bought, or an ask for the surplus or the expiring units. An order
// whose quantity or limit is out of range gets none (D96); Target and Reserve
// pass unclamped, the world reads one above MaxQuantity as 0 (FR-AGT-048).
func (t *tradeCtx) emit(p *TradePlan) {
	for i := range t.rows {
		r := &t.rows[i]
		qty := r.qty
		side := Bid
		if r.kind == askOrder {
			qty, side = r.ask, Ask
		}
		if r.kind == noOrder || qty < 1 || qty > MaxQuantity || r.limit < 1 {
			continue
		}
		p.Intents = append(p.Intents, Intent{
			Commodity: r.commodity, Side: side, Quantity: Quantity(qty), Limit: Price(r.limit),
			Target: Quantity(r.target), Reserve: Quantity(r.reserve),
			UnitCost: Price(r.unitCost), Purpose: r.purpose,
		})
	}
}
