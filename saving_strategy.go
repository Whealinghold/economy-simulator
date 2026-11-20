package economysim

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/household"
	"github.com/Whealinghold/economy-sim/internal/ordering"
	"github.com/Whealinghold/economy-sim/internal/production"
	sav "github.com/Whealinghold/economy-sim/internal/saving"
)

// This file is the rules of the saving module (spec-saving-and-investment.md 3.2, 5): the
// savings target, the choice of a project, and the three guarded calls the default strategy
// makes. Everything is a pure function of the view, so the exported pair returns what the
// default uses (FR-SAV-042), and nothing here draws, logs or allocates. The arithmetic is in
// internal/saving and the conversions are inline at each call (architecture.md 2.9).

// ProjectChoice is the project that passes the test of 5.5 and is the best of those that do
// (5.5.4), with the figures the household acts on.
type ProjectChoice struct {
	// Recipe is the project's recipe, Class the facility class it builds or repairs and Repair
	// says which.
	Recipe RecipeID
	Class  FacilityClassID
	Repair bool
	// Executions is m: 1 for a build, the planned executions of a repair (5.8.1).
	Executions int64
	// Benefit and Cost are the figures of the test (5.3, 5.4) and Outlay the money the
	// household needs in hand for it (5.6.1), in minor units.
	Benefit, Cost, Outlay int64
	// Held is the value at the beliefs of the inputs and tools of the project the household
	// already holds (5.5.3), and Funded says it holds enough money for the outlay (5.6.2).
	Held   int64
	Funded bool
}

// SavingTargetResult is the savings target of an agent (spec 3.2.2).
type SavingTargetResult struct {
	// Precaution is the buffer of 4.2 and Floor the greater of it and the reserve.
	Precaution, Floor int64
	// Goal is the money to set aside for the chosen project, net of the plan's tier 2 spend,
	// or the keep of a job in progress, or 0; Target is S* = Floor + Goal.
	Goal, Target int64
	// Project is the chosen project, valid when Chosen is true.
	Project ProjectChoice
	Chosen  bool
	// Building says a job of a project recipe is in progress, so no project is chosen and
	// the goal is its keep (5.5.6).
	Building bool
	// Bar says why no project is chosen, empty when one is, when a job is in progress and
	// when the role has no project (5.5, 12.3).
	Bar SavingBar
}

// SavingTarget returns the floor, the goal and the target S* of the agent (spec 3.2.2,
// FR-SAV-042). tier2 is the money the plan of this tick has committed to the chosen
// project's own inputs and tools as tier 2 (5.6.3); a read of saved state passes 0. A view
// with no Saving group has the flat reserve as its target.
func SavingTarget(v *AgentView, tier2 int64) SavingTargetResult {
	if v == nil {
		return SavingTargetResult{}
	}
	reserve, upkeep := v.savingBasis()
	pk := pickProject(v, reserve, upkeep)
	return pk.target(tier2)
}

// ChooseProject returns the project that passes the test of 5.5 and is the best of those
// that do, and whether there is one, with its benefit, cost, outlay and whether the
// household is funded for it (spec 7.2, FR-SAV-042). It is a pure function of the view.
func ChooseProject(v *AgentView) (ProjectChoice, bool) {
	if v == nil {
		return ProjectChoice{}, false
	}
	reserve, upkeep := v.savingBasis()
	pk := pickProject(v, reserve, upkeep)
	return pk.choice, pk.chosen
}

// savingBasis is the flat reserve and the upkeep cost of one tick the rules read. An upkeep
// cost that overflows is read as the largest value, as reserveMoney reads the reserve, so the
// outlay saturates and no project is funded; the default's Trade makes no plan at all then.
func (v *AgentView) savingBasis() (reserve, upkeep int64) {
	cost, ok := v.upkeepCost()
	if !ok {
		return math.MaxInt64, math.MaxInt64
	}
	return v.reserveMoney(householdParams(&v.Tunables)), cost
}

// savingPick is what the choice of a project settles for one view: the figures of the
// floor, the chosen project or the bar that applies, and the keep of a job in progress.
type savingPick struct {
	reserve, precaution, floor, keep int64
	choice                           ProjectChoice
	chosen, building                 bool
	bar                              SavingBar
	// first is the first project recipe weighed or barred and standing the first standing
	// reason of 4.2.2 seen, for the Debug line of a declined project.
	first    RecipeID
	standing Reason
}

// target is the savings target of the pick (spec 3.2.2): the floor plus the goal of the
// chosen project net of tier2, or the keep of a job in progress, or 0.
func (pk *savingPick) target(tier2 int64) SavingTargetResult {
	res := SavingTargetResult{
		Precaution: pk.precaution, Floor: pk.floor, Project: pk.choice, Chosen: pk.chosen,
		Building: pk.building, Bar: pk.bar,
	}
	switch {
	case pk.chosen:
		res.Goal = sav.Goal(pk.choice.Outlay, pk.choice.Held, tier2)
	case pk.building:
		res.Goal = pk.keep
	}
	res.Target = sav.Target(pk.floor, res.Goal)
	return res
}

// pickProject is the rule of 5.5: the bars, the job in progress and the best project that
// passes and is reachable. reserve is the flat reserve and upkeep the upkeep cost of the
// agent (AGT 7.4.2). A view with no group has the reserve as its floor and no project.
func pickProject(v *AgentView, reserve, upkeep int64) savingPick {
	pk := savingPick{reserve: reserve, floor: reserve, first: -1}
	sv := v.Saving
	if sv == nil {
		return pk
	}
	pk.first = v.firstProject()
	pk.precaution = sav.Precaution(sv.Block.PrecautionShareBP, sv.Block.PrecautionCapTicks, upkeep, sv.Drawdown)
	pk.floor = sav.Floor(reserve, pk.precaution)
	if v.Hunger > 0 || v.Money < reserve || reserve == math.MaxInt64 || v.OwnUse { // the largest reserve is an overflow: survival binds
		pk.bar = BarSurvival
		return pk
	}
	if v.HasJob && v.projectJob() {
		pk.building, pk.keep = true, sav.Keep(upkeep, v.Remaining, sv.SafetyTicks)
		return pk
	}
	e := projectEval{v: v, sv: sv, floor: pk.floor, upkeep: upkeep,
		hurdle: sav.Hurdle(sv.Block.HurdleBP, sv.HurdleSpreadBP, ordering.Weyl32(int64(v.Agent)))}
	e.pick(&pk)
	return pk
}

// firstProject is the first recipe of the view with a Builds or Repairs clause that is not an
// own-use recipe, or -1: the recipe a Debug line names.
func (v *AgentView) firstProject() RecipeID {
	for i := range v.Recipes {
		if r := &v.Recipes[i]; !r.OwnUse && (r.Builds.On || r.Repairs.On) {
			return r.Recipe
		}
	}
	return -1
}

// projectJob reports whether the job in progress is of a recipe with a Builds or Repairs
// clause.
func (v *AgentView) projectJob() bool {
	for i := range v.Recipes {
		if r := &v.Recipes[i]; r.Recipe == v.Recipe {
			return r.Builds.On || r.Repairs.On
		}
	}
	return false
}

// projectState is the verdict on one project.
type projectState uint8

const (
	projectStanding projectState = iota + 1
	projectNoReturn
	projectUnreachable
	projectPassed
)

// projectTally counts the verdicts of the projects of one view and keeps the best that
// passed and is reachable.
type projectTally struct {
	standing, noReturn, unreachable, projects int
	best                                      ProjectChoice
	found                                     bool
}

// take adds the verdict on a project to the tally (5.5.4: the best by benefit over cost,
// a repair before a build, then the lower recipe ID).
func (t *projectTally) take(ch ProjectChoice, st projectState) {
	t.projects++
	switch st {
	case projectStanding:
		t.standing++
	case projectNoReturn:
		t.noReturn++
	case projectUnreachable:
		t.unreachable++
	case projectPassed:
		cand := sav.Candidate{Benefit: ch.Benefit, Cost: ch.Cost, Repair: ch.Repair, ID: int64(ch.Recipe)}
		cur := sav.Candidate{Benefit: t.best.Benefit, Cost: t.best.Cost, Repair: t.best.Repair, ID: int64(t.best.Recipe)}
		if !t.found || sav.Better(cand, cur) {
			t.best, t.found = ch, true
		}
	}
}

// bar is the bar of a household none of whose projects was chosen: a passing project that
// was not reachable first, then one weighed and not passing, then standing refusals only,
// and none for a role with no project (5.5, 12.3).
func (t *projectTally) bar() SavingBar {
	switch {
	case t.unreachable > 0:
		return BarUnreachable
	case t.noReturn > 0:
		return BarNoReturn
	case t.standing > 0:
		return BarStanding
	}
	return ""
}

// projectEval is what the weighing of the projects of one view shares.
type projectEval struct {
	v      *AgentView
	sv     *SavingView
	floor  int64
	upkeep int64
	hurdle int64
}

// pick weighs every project recipe of the view, in ascending recipe ID, and sets the
// chosen project or the bar in pk (5.5.1 to 5.5.4).
func (e *projectEval) pick(pk *savingPick) {
	var t projectTally
	for i := range e.v.Recipes {
		r := &e.v.Recipes[i]
		if r.OwnUse || !(r.Builds.On || r.Repairs.On) {
			continue
		}
		if r.Standing != "" && pk.standing == "" {
			pk.standing = r.Standing
		}
		t.take(e.weigh(r))
	}
	pk.choice, pk.chosen = t.best, t.found
	if !t.found {
		pk.bar = t.bar()
	}
}

// weigh returns the verdict on one project recipe and, for a project that passed, its
// figures. A project with a standing refusal is not weighed (5.5.1).
func (e *projectEval) weigh(r *RecipeView) (ProjectChoice, projectState) {
	if r.Standing != "" {
		return ProjectChoice{}, projectStanding
	}
	if len(r.Inputs) > MaxRecipeItems || len(r.Tools) > MaxRecipeItems {
		return ProjectChoice{}, projectNoReturn // a validated recipe never is; a hand-built view can be
	}
	class := r.Builds.Class
	if r.Repairs.On {
		class = r.Repairs.Class
	}
	cl := e.sv.class(class)
	if cl == nil {
		return ProjectChoice{}, projectNoReturn
	}
	hd := heldOf(e.sv.Facilities, class)
	v, ok := e.value(r, cl, hd)
	if !ok {
		return ProjectChoice{}, projectNoReturn
	}
	return e.price(r, cl, hd, v)
}

// projectValue is the value side of a project: the executions m, the benefit and the ticks
// of service the benefit counts (T_p for a build, T_rep for a repair).
type projectValue struct {
	m, benefit, service int64
}

// value is m, the benefit and the service time of a project (5.3, 5.8.1), and false when the
// project has no value: a repair of nothing, or one below the gate.
func (e *projectEval) value(r *RecipeView, cl *SavingClass, hd classHeld) (projectValue, bool) {
	if !r.Repairs.On {
		b, tp := e.buildBenefit(cl, hd)
		return projectValue{m: 1, benefit: b, service: tp}, true
	}
	if hd.n == 0 || !sav.RepairDue(hd.lowestBP, e.sv.Block.RepairBelowBP) {
		return projectValue{}, false
	}
	m := sav.MPlan(hd.missing, r.Repairs.Units)
	restored := sav.Restored(hd.missing, m, r.Repairs.Units)
	b, trep := e.repairBenefit(cl, hd, restored)
	return projectValue{m: m, benefit: b, service: trep}, true
}

// buildBenefit is the benefit of one new facility of the class and its service time T_p
// (5.3.1, 5.3.2). A class that is not a site and has no Worth row has none.
func (e *projectEval) buildBenefit(cl *SavingClass, hd classHeld) (benefit, tp int64) {
	w, payback := e.v.Weight, e.sv.Block.PaybackTicks
	if row := e.sv.worth(cl.Name); row != nil {
		tp = sav.ServiceTime(cl.Durability, sav.WorthRate(cl.Decay, cl.WearPerUse, w, cl.Capacity), payback)
		return sav.WorthBenefit(row.PerTick, tp, hd.n, row.Count, MaxMoney), tp
	}
	if cl.Kind != kindSite {
		return 0, 0
	}
	pv := e.jobPVs(cl.Class)
	after := hd.capacity + cl.Capacity
	dJ := sav.BuildGain(sav.JobValue(pv.b, pv.a, w, hd.capacity), sav.JobValue(pv.b, pv.a, w, after))
	tp = sav.ServiceTime(cl.Durability, sav.Rate(cl.Decay, cl.WearPerUse, min(w, after), pv.durB), payback)
	return sav.SiteBenefit(dJ, tp, MaxMoney), tp
}

// repairBenefit is the benefit of restoring what is missing and its service time T_rep
// (5.8.1); a Worth row's PerTick replaces the computed gain and the row's Count does not
// limit a repair.
func (e *projectEval) repairBenefit(cl *SavingClass, hd classHeld, restored int64) (benefit, trep int64) {
	w, payback := e.v.Weight, e.sv.Block.PaybackTicks
	if row := e.sv.worth(cl.Name); row != nil {
		rate := sav.WorthRate(cl.Decay, cl.WearPerUse, w, cl.Capacity)
		trep = sav.RepairTime(restored, rate, payback, hd.n)
		return sav.WorthBenefit(row.PerTick, trep, 0, 1, MaxMoney), trep
	}
	if cl.Kind != kindSite {
		return 0, 0
	}
	pv := e.jobPVs(cl.Class)
	dJ := sav.BuildGain(sav.JobValue(pv.b, pv.a, w, max(0, hd.capacity-cl.Capacity)), sav.JobValue(pv.b, pv.a, w, hd.capacity))
	rate := sav.Rate(cl.Decay, cl.WearPerUse, min(w, hd.capacity), pv.durB)
	trep = sav.RepairTime(restored, rate, payback, hd.n)
	return sav.SiteBenefit(dJ, trep, MaxMoney), trep
}

// price is the cost side: the cost and the test (5.4, 5.5), then the outlay, the funds and
// the reach (5.6). It returns the figures of a project that passes and is reachable.
func (e *projectEval) price(r *RecipeView, cl *SavingClass, hd classHeld, pv projectValue) (ProjectChoice, projectState) {
	ln := e.lines(r, pv.m)
	pvs := e.jobPVs(cl.Class)
	opp := sav.Opportunity(sav.JobValue(pvs.b, pvs.a, e.v.Weight, hd.capacity), r.Duration)
	cost := sav.Cost(ln.inputs, opp, MaxMoney)
	hurdle := e.hurdle
	if sav.Committed(ln.held, ln.inputs) {
		hurdle = sav.Lowered(hurdle, e.sv.CommitBandBP)
	}
	if !sav.Passes(pv.benefit, cost, hurdle, MaxMoney) {
		return ProjectChoice{}, projectNoReturn
	}
	outlay := sav.Outlay(ln.inputs, ln.toolBuy, e.upkeep, r.Duration, e.sv.SafetyTicks)
	need := sav.Need(outlay, ln.held, e.floor, e.v.Money)
	if !sav.Reach(need, e.sv.Records, e.sv.Surplus, pv.service) {
		return ProjectChoice{}, projectUnreachable
	}
	return ProjectChoice{
		Recipe: r.Recipe, Class: cl.Class, Repair: r.Repairs.On, Executions: pv.m,
		Benefit: pv.benefit, Cost: cost, Outlay: outlay, Held: ln.held,
		Funded: sav.Funded(e.v.Money, ln.held, outlay, e.floor),
	}, projectPassed
}

// projectLines is the money terms of the lines of a recipe for m executions: the inputs and
// the wear of the tools as cost_r (7.1.4), the price of the tools the household does not
// hold, and the value of what it holds (5.5.3, 5.6.1).
type projectLines struct {
	inputs, toolBuy, held int64
}

// lines converts the lines of the recipe to the primitives of internal/saving at the call
// (architecture.md 2.9). The arrays are on the stack: a recipe has at most MaxRecipeItems
// lines on each side.
func (e *projectEval) lines(r *RecipeView, m int64) projectLines {
	var qty, centre, stock [2*MaxRecipeItems + 1]int64
	var tq, tc, ts [MaxRecipeItems]int64
	n := 0
	for _, in := range r.Inputs {
		qty[n], centre[n], stock[n] = in.Quantity, e.v.centreOf(in.Commodity), e.v.stockOf(in.Commodity)
		n++
	}
	nt := 0
	for _, t := range r.Tools {
		c := e.v.centreOf(t.Commodity)
		wear, ok := production.WearCost(production.Wear{Commodity: int(t.Commodity), WearPerUse: t.WearPerUse, Durability: t.Durability}, c)
		if !ok {
			wear = math.MaxInt64 // a wear that cannot be priced is a cost no project passes
		}
		qty[n], centre[n] = 1, wear // the wear of a tool is a term of the cost; it is not held stock
		n++
		tq[nt], tc[nt], ts[nt] = 1, c, e.v.stockOf(t.Commodity)
		nt++
	}
	qty[n], centre[n] = 1, r.Rent
	n++
	return projectLines{
		inputs:  sav.Inputs(m, qty[:n], centre[:n]),
		toolBuy: sav.ToolBuy(m, ts[:nt], tc[:nt]),
		held:    sav.HeldValue(m, qty[:len(r.Inputs)], stock[:len(r.Inputs)], centre[:len(r.Inputs)]) + sav.HeldValue(m, tq[:nt], ts[:nt], tc[:nt]),
	}
}

// jobPV is the value per tick of the best valued recipe that needs a class (b) and of the
// best that does not and whose other needs the agent holds (a), and the duration of the
// recipe that gives b (5.3.1).
type jobPV struct{ b, a, durB int64 }

// jobPVs is pv_b and pv_a of 5.3.1 for the class. A recipe is valued when it has a positive
// yield, is eligible by 7.2 ignoring stock and is not halted.
func (e *projectEval) jobPVs(class FacilityClassID) jobPV {
	var out jobPV
	par := householdParams(&e.v.Tunables)
	for i := range e.v.Recipes {
		r := &e.v.Recipes[i]
		if r.OwnUse || r.Effective <= 0 || e.v.recipeHalted(r, par) {
			continue
		}
		net, ok := e.v.recipeNet(r)
		if !ok || !household.Eligible(net, len(r.Inputs) == 0) {
			continue
		}
		pv := sav.PerTick(net, r.Duration)
		switch {
		case needsClass(r, class):
			if pv > out.b {
				out.b, out.durB = pv, r.Duration
			}
		case e.holdsNeeds(r):
			out.a = max(out.a, pv)
		}
	}
	return out
}

// holdsNeeds reports whether the agent holds a capacity above 0 of every class the recipe
// needs.
func (e *projectEval) holdsNeeds(r *RecipeView) bool {
	for _, c := range r.Needs {
		if heldOf(e.sv.Facilities, c).capacity <= 0 {
			return false
		}
	}
	return true
}

// needsClass reports whether the recipe needs the class.
func needsClass(r *RecipeView, class FacilityClassID) bool {
	for _, c := range r.Needs {
		if c == class {
			return true
		}
	}
	return false
}

// classHeld is what the agent holds of one class in its region: the facilities, the sum of
// cap(f), the condition units missing and the lowest share of durability left, in basis
// points (5.3.1, 5.8.1). lowestBP is meaningful only when n is above 0.
type classHeld struct {
	n, capacity, missing, lowestBP int64
}

// heldOf sums the facilities of the class.
func heldOf(fs []FacilityView, class FacilityClassID) classHeld {
	h := classHeld{lowestBP: household.Basis}
	for i := range fs {
		f := &fs[i]
		if f.Class != class {
			continue
		}
		h.n++
		h.capacity += f.Capacity
		h.missing += max(0, f.Durability-f.Condition)
		h.lowestBP = min(h.lowestBP, sav.ConditionBP(f.Condition, f.Durability))
	}
	return h
}

// savingTarget is the savings target the default's spend cap reads instead of the flat
// reserve (spec 3.2.2): the floor plus the goal of the chosen project, net of the tier 2 spend
// of the plan when the project is the planned recipe (5.6.3). In any other plan the tier 2 is
// the inputs of another recipe, none of them the project's, and is not netted from the goal.
// The pick was made once for the view, by planProject.
func (t *tradeCtx) savingTarget(tier2 int64) int64 {
	if !t.projectPlanned {
		tier2 = 0
	}
	return t.pk.target(tier2).Target
}

// planProject picks the project of the agent once for the Trade call and, when the
// household is funded and the plan has no trip, makes the project's recipe the planned
// recipe with m executions of inputs (spec 5.6.4, FR-SAV-020).
func (t *tradeCtx) planProject(reserve, upkeep int64) {
	t.pk = pickProject(t.v, reserve, upkeep)
	if !t.pk.chosen || !t.pk.choice.Funded || t.trip != nil {
		return
	}
	for i := range t.v.Recipes {
		if t.v.Recipes[i].Recipe == t.pk.choice.Recipe {
			t.rstar, t.jobs, t.projectPlanned = i, t.pk.choice.Executions, true
			return
		}
	}
}

// savingRequest is the request of a household with a project chosen (spec 5.7): one
// execution of a build and MaxExecutions of a repair, whatever the work share, and only
// when the project's recipe can run now. ok is false when the default's request stands.
func savingRequest(v *AgentView) (JobRequest, bool) {
	if v.HasJob {
		return JobRequest{}, false
	}
	reserve, upkeep := v.savingBasis()
	pk := pickProject(v, reserve, upkeep)
	if !pk.chosen {
		return JobRequest{}, false
	}
	for i := range v.Recipes {
		r := &v.Recipes[i]
		if r.Recipe != pk.choice.Recipe || r.MaxExecutions < 1 {
			continue
		}
		if pk.choice.Repair {
			return JobRequest{Recipe: r.Recipe, Executions: r.MaxExecutions}, true
		}
		return JobRequest{Recipe: r.Recipe, Executions: 1}, true
	}
	return JobRequest{}, false
}
