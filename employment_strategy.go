package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/household"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is what a strategy sees and does of the employment module's hire form
// (spec-employment.md 4.3, 9), EMP: the view group of a Hireable role and of an
// employer, the default strategy's offer and vacancies, and the planned hires that
// size an employer's targets. The decisions are pure functions of the view and keep
// no state: the default draws no word for an offer or a vacancy (4.3.5).

// EmployerView is the employer group of an AgentView (spec-employment.md 9.1). It
// is present, with HasEmployer, only for an agent whose role has an employer block
// in a world in the hire form. Recipes is shared by the agents of the role and a
// strategy must treat it as read-only.
type EmployerView struct {
	// Recipes is the hire list in ascending RecipeID; MaxHires, WorkerShareBP and
	// HireYieldBP are the block's fields (spec-employment.md 3.2).
	Recipes                              []RecipeID
	MaxHires, WorkerShareBP, HireYieldBP int64
	// Hired is the executions of the hires the agent has open and Hires their number.
	Hired, Hires int64
	// Open is the hires open in the agent's region against MaxOpen, so that a strategy
	// does not post into a full lane.
	Open, MaxOpen int64
}

// fillEmployment is the filler of the employment group of AgentView (spec-employment.md
// 9.1, H9): Hireable for a role with the field, the employer group for a role with
// the block, and for each recipe of the hire list the hired yield beside the
// effective one. It reads the module's state and nothing else.
func (w *World) fillEmployment(v *AgentView, a *agent) {
	e := w.employment
	v.Hireable = e.hireable[a.role]
	rb := &e.roles[a.role]
	if !rb.on {
		return
	}
	v.HasEmployer = true
	v.Employer = EmployerView{
		Recipes: rb.list[:len(rb.list):len(rb.list)], MaxHires: rb.maxHires, WorkerShareBP: rb.shareBP,
		HireYieldBP: rb.yieldBP, Hired: e.hiredBy(a.id), Hires: e.hiredCount(a.id),
		Open: e.open[v.Region], MaxOpen: e.maxOpen,
	}
	for i := range v.Recipes {
		if r := &v.Recipes[i]; slices.Contains(rb.list, r.Recipe) {
			r.Hired = w.hiredEffective(v.Region, r.Recipe, rb.yieldBP)
		}
	}
}

// hiredEffective is the effective yield of a hired execution of the recipe in the
// region: the factor of spec-employment.md 5.5 with HireYieldBP in the place of the
// trainee term, the one a hire's output is created at.
func (w *World) hiredEffective(region RegionID, recipe RecipeID, yieldBP int64) int64 {
	cr := &w.cfg.Recipes[recipe]
	e, _, ok := w.yields.effectiveFor(&w.shocks, region, recipe, w.tk.moment.Season, cr.Sited,
		w.prod.abundance(region, recipe, false), w.prod.techFactor(region, recipe), yieldBP)
	if !ok {
		invariant("the effective yield of a hire does not fit after validation")
	}
	return e
}

// employmentWork adds the offer and the vacancy of the default strategy to the
// request the work decision made (spec-employment.md 4.3, H10). buf is the work
// report of the same view, so the offer reads the share and the chosen recipe the
// decision used.
func employmentWork(v *AgentView, buf *workReportBuf, req *JobRequest) {
	if v.Hireable {
		req.Offer = v.defaultOffer(buf)
	}
	if v.HasEmployer {
		var own int64
		if buf.chosen >= 0 {
			own = buf.executions
		}
		req.Hire = v.defaultHire(own)
	}
}

// defaultOffer is the worker's offer of 4.3.1: ceil(w * s / 10 000) households at the
// reservation of 4.3.2, none when the agent is in the own-use mode or its work share
// is 0 (a household the target-income rule rests offers nothing).
func (v *AgentView) defaultOffer(buf *workReportBuf) LabourOffer {
	if v.OwnUse || buf.share < 1 {
		return LabourOffer{}
	}
	n := min(welfare.Executions(v.Weight, buf.share), v.Weight)
	if n < 1 {
		return LabourOffer{}
	}
	return LabourOffer{Households: n, Reserve: Price(v.reservation(buf))}
}

// reservation is the value per tick of the agent's own best recipe, the one the work
// decision chose: max(0, floor(net / Duration)), and 0 with none (4.3.2). It is the
// agent's own price of a tick of its labour at its own beliefs.
func (v *AgentView) reservation(buf *workReportBuf) int64 {
	if buf.chosen < 0 {
		return 0
	}
	r := &v.Recipes[buf.chosen]
	net, ok := v.recipeNet(r)
	if !ok || net < 1 || r.Duration < 1 {
		return 0
	}
	return min(net/r.Duration, MaxPrice)
}

// hirePick is the recipe of the hire list the default would post for (4.3.3): the
// eligible one with the best net value per tick at the hired yield, ties to the lower
// ID, not halted by the production halt. net is the net value of one hired execution
// and idx its index in the view; ok is false with none.
func (v *AgentView) hirePick() (idx int, net int64, ok bool) {
	p := householdParams(&v.Tunables)
	idx = -1
	for i := range v.Recipes {
		r := &v.Recipes[i]
		if r.OwnUse || r.Hired < 1 || !slices.Contains(v.Employer.Recipes, r.Recipe) || v.recipeHalted(r, p) {
			continue
		}
		n, fits := v.recipeNetAt(r, r.Hired)
		if !fits || !household.Eligible(n, len(r.Inputs) == 0) {
			continue
		}
		if idx < 0 || household.Better(n, r.Duration, net, v.Recipes[idx].Duration) {
			idx, net = i, n
		}
	}
	return idx, net, idx >= 0
}

// hireBid is the bid of a vacancy for the recipe at index idx with net value net per
// execution: floor(net / Duration), the most a tick of a hired household is worth
// (4.3.3), cut to MaxPrice. It is below 1 for a recipe that is worth nothing. A Duration
// below 1 (a hand-built view; Validate rules it out) counts as 1, so it cannot divide by 0.
func (v *AgentView) hireBid(idx int, net int64) int64 {
	return min(max(net, 0)/max(v.Recipes[idx].Duration, 1), MaxPrice)
}

// defaultHire is the employer's vacancies of 4.3.3: the best recipe of the hire list
// at the hired yield, bid the net value per tick, positions the least of the
// establishment left and what stock, tools and cash allow after the agent's own
// request of own executions. It posts nothing when the bid is below 1, no position
// is left or the region's lane is full. It is not scaled by the work share (4.3.4).
func (v *AgentView) defaultHire(own int64) HireRequest {
	eb := &v.Employer
	if v.StartsIn != 1 || eb.Open >= eb.MaxOpen {
		return HireRequest{}
	}
	idx, net, ok := v.hirePick()
	if !ok {
		return HireRequest{}
	}
	bid := v.hireBid(idx, net)
	if bid < 1 {
		return HireRequest{}
	}
	r := &v.Recipes[idx]
	n := min(eb.MaxHires-eb.Hired, v.byCash(r, bid), v.byStock(r)-own, v.byTools(r)-own)
	if n < 1 {
		return HireRequest{}
	}
	return HireRequest{Recipe: r.Recipe, Positions: n, Bid: Price(bid)}
}

// byCash is the positions the cash allows: floor((Money - UpkeepCost) / (bid *
// Duration)), the employer leaving its tier 1 for the tick (4.3.3). It is 0 when the
// upkeep cost or the wage bill of one position does not fit.
func (v *AgentView) byCash(r *RecipeView, bid int64) int64 {
	cost, ok := v.upkeepCost()
	bill, bok := checked.Mul(bid, r.Duration)
	if !ok || !bok || bill < 1 || v.Money <= cost {
		return 0
	}
	return (v.Money - cost) / bill
}

// byStock is the executions of the recipe the stock of its inputs allows, the food
// the agent eats kept out of it (keep-the-food, spec-agents.md 7.1.6), before the
// agent's own request; unbounded for a recipe with no input.
func (v *AgentView) byStock(r *RecipeView) int64 {
	n := int64(MaxQuantity)
	for _, in := range r.Inputs {
		if in.Quantity < 1 {
			continue
		}
		k := v.stockOf(in.Commodity) / in.Quantity
		for _, row := range v.Upkeep {
			if u, fits := checked.Mul(v.Weight, row.Quantity); row.Commodity == in.Commodity && fits {
				k = min(k, household.KeepFoodCap(v.stockOf(in.Commodity), u, in.Quantity))
			}
		}
		n = min(n, k)
	}
	return n
}

// byTools is the executions the tools of the recipe allow: the items of the scarcest
// tool, before the agent's own request; unbounded for a recipe with no tool.
func (v *AgentView) byTools(r *RecipeView) int64 {
	n := int64(MaxQuantity)
	for _, t := range r.Tools {
		n = min(n, v.stockOf(t.Commodity))
	}
	return n
}

// plannedHire is the recipe r_h the default would post for next tick (an index into
// the view's Recipes) and H*, the positions it would post, for the targets of Trade
// (spec-employment.md 9.2.1). n is 0 with no plan.
type plannedHire struct {
	idx int
	n   int64
}

// plannedHire returns the plan for the agent whose tier 1 costs upkeep: bounded by the
// establishment and the cash left after tier 1, not by the stock and the tools it is
// about to buy; a busy employer plans none (9.2.2).
func (v *AgentView) plannedHire(upkeep int64) plannedHire {
	eb := &v.Employer
	if !v.HasEmployer || v.StartsIn != 1 || eb.Open >= eb.MaxOpen {
		return plannedHire{}
	}
	idx, net, ok := v.hirePick()
	if !ok {
		return plannedHire{}
	}
	bid := v.hireBid(idx, net)
	bill, fits := checked.Mul(bid, v.Recipes[idx].Duration)
	if bid < 1 || !fits || bill < 1 || upkeep < 0 || v.Money <= upkeep {
		return plannedHire{}
	}
	return plannedHire{idx: idx, n: max(min(eb.MaxHires-eb.Hired, (v.Money-upkeep)/bill), 0)}
}

// hireInputs adds to j the units of c the planned hires need for the next tick:
// InputJobs * H* * q_c of the recipe r_h (spec-employment.md 9.2.1).
func (t *tradeCtx) hireInputs(j *sum, c CommodityID) {
	if t.hire.n < 1 {
		return
	}
	for _, in := range t.v.Recipes[t.hire.idx].Inputs {
		if in.Commodity == c {
			jobs, ok1 := checked.Mul(t.par.InputJobs, t.hire.n)
			n, ok2 := checked.Mul(jobs, in.Quantity)
			j.add(n, ok1 && ok2)
		}
	}
}

// hireTools adds to j the items of c the planned hires need as tools: H* for each
// tool of the recipe r_h.
func (t *tradeCtx) hireTools(j *sum, c CommodityID) {
	if t.hire.n < 1 {
		return
	}
	for _, tl := range t.v.Recipes[t.hire.idx].Tools {
		if tl.Commodity == c {
			j.add(t.hire.n, true)
		}
	}
}

// hirePurpose is the Purpose of an order for c when the planned hires need it: an input
// with a job to buy for, or a tool, of the recipe r_h. ok is false when they do not.
func (t *tradeCtx) hirePurpose(c CommodityID, job int64) (Purpose, bool) {
	if t.hire.n < 1 {
		return PurposeOther, false
	}
	r := &t.v.Recipes[t.hire.idx]
	if job > 0 {
		for _, in := range r.Inputs {
			if in.Commodity == c {
				return PurposeInput, true
			}
		}
	}
	for _, tl := range r.Tools {
		if tl.Commodity == c {
			return PurposeTool, true
		}
	}
	return PurposeOther, false
}
