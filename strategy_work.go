package economysim

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/household"
	"github.com/Whealinghold/economy-sim/internal/production"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is the work decision of the default strategy (spec-agents.md 7.1,
// plan-phase-1-7.md D88): one pure function of the view, the work report, that
// DefaultStrategy.Work reads and that the world reads for its counts (the
// floor, the halt, the rest and the decline reason), so that the two cannot
// drift apart and a consumer's strategy cannot hide them.

// workReportBuf is the result of workReport. The skipped slice is reused: the
// caller passes a buffer and the report resets it. The zero value works.
type workReportBuf struct {
	// share is the work share in basis points (7.1.2, 7.1.3).
	share int64
	// floorChanged says the floor MinWorkShare held the agent to work: the floor
	// condition holds and the share the ratio alone gives is below 10 000, the
	// predicate of welfare.HeldByFloor (7.1.3).
	floorChanged bool
	// rest says the share is 0.
	rest bool
	// skipped are the recipes the production halt skips, ascending by RecipeID
	// (7.1.5). It is empty when the agent rests: the halt is counted only for a
	// household that would have worked.
	skipped []RecipeID
	// halted is the number of recipes the halt skips, counted whether or not
	// skipped is filled; noList, set by the caller before the call, leaves
	// skipped empty, which is what Work does so that it allocates nothing.
	halted int
	noList bool
	// reason is why no job is requested, empty when one is.
	reason Reason
	// blocked is the index into the view's Recipes of the first recipe that
	// cannot run, set only when reason is that recipe's production reason
	// (the halt skipped none and some recipe could not start); -1 otherwise.
	// The world counts the household as refused for it in the RecipeRefused
	// tally, since the default strategy never requests a recipe the view says
	// cannot run and so never reaches the refusal of a start (N-308).
	blocked int
	// chosen is the index into the view's Recipes of the recipe requested, -1
	// when none is; executions is the number to request.
	chosen     int
	executions int64
}

// householdParams copies the tunables the rules of internal/household read.
func householdParams(t *Tunables) household.Params {
	return household.Params{
		CoverTicks: t.CoverTicks, InputJobs: t.InputJobs, OutputCoverTicks: t.OutputCoverTicks,
		FavourFloor: t.FavourFloor, ReserveTicks: t.ReserveTicks, SpendMaxShare: t.SpendMaxShare,
		SpendHalfTicks: t.SpendHalfTicks, ProduceHaltRatio: t.ProduceHaltRatio,
	}
}

// commodityAt returns the index of c in the view's tradable set, or -1.
func (v *AgentView) commodityAt(c CommodityID) int {
	for i := range v.Commodities {
		if v.Commodities[i].Commodity == c {
			return i
		}
	}
	return -1
}

// centreOf is the agent's belief centre for c, 0 for a commodity outside the
// tradable set.
func (v *AgentView) centreOf(c CommodityID) int64 {
	if i := v.commodityAt(c); i >= 0 {
		return int64(v.Commodities[i].Belief.Centre)
	}
	return 0
}

// stockOf is the agent's stock of c, 0 for a commodity outside the tradable set.
func (v *AgentView) stockOf(c CommodityID) int64 {
	if i := v.commodityAt(c); i >= 0 {
		return v.Commodities[i].Stock
	}
	return 0
}

// upkeepCost is the upkeep cost of one tick at the agent's beliefs, the sum
// over the upkeep rows of the weight times the quantity times the centre
// (7.4.2). ok is false on overflow.
func (v *AgentView) upkeepCost() (int64, bool) {
	var c household.Cost
	for _, row := range v.Upkeep {
		total, ok := checked.Mul(v.Weight, row.Quantity)
		if !ok {
			return 0, false
		}
		c.Add(total, v.centreOf(row.Commodity))
	}
	return c.Value()
}

// reserveMoney is ReserveTicks times the upkeep cost (7.4.2). An overflow is
// read as the largest value, so the survival rule then holds and nothing is
// spent on a buffer.
func (v *AgentView) reserveMoney(p household.Params) int64 {
	cost, ok := v.upkeepCost()
	if ok {
		if r, ok := household.ReserveMoney(cost, p.ReserveTicks); ok {
			return r
		}
	}
	return math.MaxInt64
}

// recipeNet is the net value of one execution of r, value minus cost, at the
// agent's beliefs (7.1.4): the non-discard outputs at the effective yield, the
// inputs, and the wear of the tools by production.WearCost (D90). ok is false
// on overflow.
func (v *AgentView) recipeNet(r *RecipeView) (int64, bool) {
	var n household.Net
	for _, o := range r.Outputs {
		if !o.Discard {
			n.Output(o.Quantity, r.Effective, v.centreOf(o.Commodity))
		}
	}
	for _, in := range r.Inputs {
		n.Input(in.Quantity, v.centreOf(in.Commodity))
	}
	for _, t := range r.Tools {
		w, ok := production.WearCost(production.Wear{Commodity: int(t.Commodity), WearPerUse: t.WearPerUse, Durability: t.Durability}, v.centreOf(t.Commodity))
		if !ok {
			return 0, false
		}
		n.Cost(w)
	}
	return n.Value()
}

// recipeHalted reports whether the production halt skips r: an output with a
// positive CostShare is overstocked (7.1.5). A byproduct never halts.
func (v *AgentView) recipeHalted(r *RecipeView, p household.Params) bool {
	for _, o := range r.Outputs {
		if o.CostShare <= 0 {
			continue
		}
		tgt, ok := household.OutputTarget(v.Weight, o.Quantity, r.Effective, p.OutputCoverTicks, r.Duration)
		if ok && household.Halted(v.stockOf(o.Commodity), tgt, p.ProduceHaltRatio) {
			return true
		}
	}
	return false
}

// foodCap is the most executions of r the view allows: its MaxExecutions cut by
// the keep-the-food cap of every input that is also an upkeep commodity (7.1.6).
// A cap of 0 makes the recipe ineligible for the request.
func (v *AgentView) foodCap(r *RecipeView) int64 {
	k := r.MaxExecutions
	for _, in := range r.Inputs {
		for _, row := range v.Upkeep {
			if row.Commodity != in.Commodity {
				continue
			}
			u, ok := checked.Mul(v.Weight, row.Quantity)
			if !ok {
				return 0
			}
			k = min(k, household.KeepFoodCap(v.stockOf(in.Commodity), u, in.Quantity))
		}
	}
	return k
}

// workShare is the work share of 7.1.2 and 7.1.3 and whether the floor changed
// it. The ratio absent gives 10 000. The survival rule (hunger, or money below
// the reserve) and the floor give 10 000 whatever the ratio says.
func workShare(v *AgentView, reserve int64) (share int64, floorChanged bool) {
	if !v.HasRatio {
		return wholeBP, false
	}
	byRatio := welfare.WorkShare(v.Ratio, v.TargetRatio, v.Tunables.RestBand)
	floor := welfare.FloorBinds(v.WorkingTicks, v.Window, v.MinWorkShare)
	floorChanged = floor && byRatio < wholeBP
	if v.Hunger > 0 || v.Money < reserve || floor {
		return wholeBP, floorChanged
	}
	return byRatio, false
}

// offer is the net value of r and the executions to request of it, and whether
// the recipe is eligible: a net that passes household.Eligible and a request of
// at least 1 once the food cap cuts the want (7.1.4, 7.1.6).
func (v *AgentView) offer(r *RecipeView, want int64) (net, n int64, ok bool) {
	net, ok = v.recipeNet(r)
	if !ok || !household.Eligible(net, len(r.Inputs) == 0) {
		return 0, 0, false
	}
	n, ok = household.Request(want, v.foodCap(r))
	return net, n, ok
}

// recipePick is the state of the comparison of the recipes in workReport.
type recipePick struct {
	want, bestNet, bestDur int64
	cannotRun              Reason
	cannotRunAt            int
}

// consider takes recipe i into the report: the halt skips it, a recipe that
// cannot run is passed over (the first one gives the decline reason), and an
// eligible one replaces the chosen when it is better (7.1.5 to 7.1.7).
func (dst *workReportBuf) consider(v *AgentView, i int, p household.Params, pk *recipePick) {
	r := &v.Recipes[i]
	if r.MaxExecutions < 1 && pk.cannotRun == "" {
		pk.cannotRun, pk.cannotRunAt = r.Reason, i
	}
	if v.recipeHalted(r, p) {
		dst.skip(r.Recipe)
		return
	}
	if r.MaxExecutions < 1 {
		return
	}
	net, n, ok := v.offer(r, pk.want)
	if !ok {
		return
	}
	if dst.chosen < 0 || household.Better(net, r.Duration, pk.bestNet, pk.bestDur) {
		dst.chosen, dst.executions, pk.bestNet, pk.bestDur = i, n, net, r.Duration
	}
}

// skip counts a recipe the halt skips and lists it unless noList is set.
func (dst *workReportBuf) skip(r RecipeID) {
	dst.halted++
	if !dst.noList {
		dst.skipped = append(dst.skipped, r)
	}
}

// declineReason is why no recipe is chosen: halted if the halt skipped one,
// else the production reason of the first recipe that cannot run, else no_recipe.
func (dst *workReportBuf) declineReason(cannotRun Reason) Reason {
	switch {
	case dst.halted > 0:
		return ReasonHalted
	case cannotRun != "":
		return cannotRun
	}
	return ReasonNoRecipe
}

// workReport does the whole work decision for v into dst: the share, the halt,
// the eligibility (including a food cap of at least 1), the comparison of the
// recipes, and the request (7.1.2 to 7.1.7). The reason a request is declined
// has this precedence: rest (a share of 0), halted (a recipe was skipped by the
// halt and none is chosen), the production reason of the first recipe that
// cannot run, then no_recipe. It reads no random number and allocates nothing
// once dst.skipped has capacity.
func workReport(v *AgentView, dst *workReportBuf) {
	p := householdParams(&v.Tunables)
	*dst = workReportBuf{skipped: dst.skipped[:0], noList: dst.noList, chosen: -1, blocked: -1}
	dst.share, dst.floorChanged = workShare(v, v.reserveMoney(p))
	if dst.share == 0 {
		dst.rest, dst.reason = true, ReasonRest
		return
	}
	pk := recipePick{want: welfare.Executions(v.Weight, dst.share)}
	for i := range v.Recipes {
		dst.consider(v, i, p, &pk)
	}
	if dst.chosen < 0 {
		dst.reason = dst.declineReason(pk.cannotRun)
		if dst.halted == 0 && pk.cannotRun != "" {
			dst.blocked = pk.cannotRunAt
		}
	}
}
