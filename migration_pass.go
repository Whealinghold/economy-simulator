package economysim

import (
	"math"

	mig "github.com/Whealinghold/economy-sim/internal/migration"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is the pass of the migration module at P7.2 (spec-migration.md 10.2,
// MIG, a continuation of migration.go for the Lizard limits, architecture.md 2.6).
// The pass converts at the call into internal/migration (architecture.md 2.9) and
// back: the package knows no region, commodity or role.

// passTally is what the pass counts for one region, in households (the sum of
// weights), for MigrationReviewed (spec-migration.md 13.1): the agents are
// attributed to the region of their account.
type passTally struct {
	unsettled, reviewed, pushed int64
	outcome                     [mig.Outcomes]int64
	held, ended                 int64
}

// startRec is one departure, in the order of admission, for MigrationStarted
// (spec-migration.md 13.1).
type startRec struct {
	agent                     AgentID
	from, to                  RegionID
	weight, arrives, distance int64
	k, earned, home, estimate int64
	jobAbandoned              bool
}

// evalRec is a pushed agent with a ranking, waiting for admission: its ranking
// is the span [first, last) of the arena.
type evalRec struct {
	id           AgentID
	region       RegionID
	earned, home int64
	first, last  int
}

// passScratch is the scratch of one pass, kept in the module and reused, so a pass
// allocates nothing once the buffers have grown (NFR-MIG-001).
type passScratch struct {
	households, pipeline []int64 // per region
	due                  []AgentID
	dueAt                []RegionID
	price                []int64 // per (region, commodity), the mean of a rumour, 0 for none
	rumours              []Rumour
	dests                []RegionID
	cands                []mig.Candidate
	ranked               []mig.Ranked
	arena                []mig.Ranked
	evals                []evalRec
	agents               []mig.Agent
	results              []mig.Result
	admit                mig.Scratch
	caps                 mig.Caps
	lines                []mig.Line
	tools                []mig.Tool
	recipes              []mig.Recipe
	costQ, costP         []int64
	tally                []passTally
	started              []startRec
	// visited and evaluated count the work of the last pass: the agents the scan of
	// step 1 saw and the candidates step 2 evaluated (NFR-MIG-002).
	visited, evaluated int64
}

// p72Migration is P7.2, the migration pass, run after the entry step of every
// region (spec-migration.md 10.1, 10.2): the counts of step 1, the evaluation of
// the agents whose turn has come in ascending ID (step 2), their admission in the
// order of need (step 3) and their departures (step 4). A tick in which no agent
// is due costs the scan of step 1 and nothing more (NFR-MIG-002). The events of
// step 5 are migration_report.go's.
func (w *World) p72Migration() {
	m := w.migration
	p := &m.pass
	w.passCounts()
	for i, id := range p.due {
		w.review(&w.agents[id], p.dueAt[i])
	}
	w.admitAndDepart()
	for g := range p.tally {
		p.tally[g].held, p.tally[g].ended = m.held[g], m.ended[g]
		m.held[g], m.ended[g] = 0, 0
	}
	w.reportMigration()
}

// passCounts is step 1: the households of each region (the journeys included,
// their account is still there) and the households on a journey to each, and the
// agents whose turn has come, in ascending ID. The agent must be eligible by items
// 1 to 5 of spec-migration.md 3.3; whether it is settled and pushed is tested in
// review.
func (w *World) passCounts() {
	m, tick := w.migration, w.Tick()
	p := &m.pass
	clear(p.households)
	clear(p.pipeline)
	clear(p.tally)
	p.due, p.dueAt, p.started = p.due[:0], p.dueAt[:0], p.started[:0]
	p.visited, p.evaluated = 0, 0
	for i := w.firstHousehold; i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 {
			continue
		}
		g := w.regionOf(a)
		p.households[g] += a.weight
		p.visited++
		if m.on(a.id) {
			p.pipeline[m.journey[a.id].to] += a.weight
			continue
		}
		if mig.Due(tick, int64(a.id), m.cfg.MoveReviewTicks) && !a.external &&
			!w.cfg.Roles[a.role].Fixed && !w.offStage(a) {
			p.due, p.dueAt = append(p.due, a.id), append(p.dueAt, g)
		}
	}
}

// review tests one agent whose turn has come: settled (spec-migration.md 5.2, and
// counted as unsettled when the gate would have pushed it), pushed (5.3), then its
// destinations are evaluated. An agent that is not settled computes nothing else.
func (w *World) review(a *agent, g RegionID) {
	m, tick := w.migration, w.Tick()
	p := &m.pass
	t := &p.tally[g]
	row := w.roles.row(a.role)
	// The push test reads the earned ratio (relief excluded, spec-migration.md 5.3), not
	// the view's Ratio: a household kept up by relief is still pushed.
	earned, present := a.history.EarnedRatio(w.upkeepCost(a, g, row), w.cfg.Roles[a.role].DefaultRatio)
	pushed := mig.Pushed(earned, present, m.cfg.MoveBelow)
	if !mig.Settled(tick, a.roleSince, m.cfg.MoveCooldown, row.window) {
		if pushed {
			t.unsettled += a.weight
		}
		return
	}
	t.reviewed += a.weight
	if !pushed {
		return
	}
	t.pushed += a.weight
	home := w.homeFigure(a, g, row)
	ranked, outcome := w.evaluate(a, g, row, home)
	if len(ranked) == 0 {
		t.outcome[outcome] += a.weight
		w.declined(a, outcome)
		return
	}
	first := len(p.arena)
	p.arena = append(p.arena, ranked...)
	p.evals = append(p.evals, evalRec{id: a.id, region: g, earned: earned, home: home, first: first, last: len(p.arena)})
}

// upkeepCost is the cost of one tick of the upkeep of the agent's weight at the
// published prices of its region (spec-welfare-and-wages.md 2.1), as the view
// computes it for the ratio.
func (w *World) upkeepCost(a *agent, g RegionID, row *roleRow) int64 {
	p := &w.migration.pass
	p.costQ, p.costP = p.costQ[:0], p.costP[:0]
	for _, it := range row.upkeepIn(g) {
		p.costQ = append(p.costQ, int64(it.quantity))
		p.costP = append(p.costP, int64(w.markets.reports[w.markets.index(g, it.commodity)].Price))
	}
	return welfare.TickCost(p.costQ, p.costP, a.weight)
}

// localPrice is the price a household reads at home: the price window mean of the
// market when the window is not empty, else the published price (spec-migration.md
// 6.3.2).
func (w *World) localPrice(g RegionID, c CommodityID) int64 {
	i := w.markets.index(g, c)
	if mean, ok := w.markets.windows[i].Mean(); ok {
		return mean
	}
	return int64(w.markets.reports[i].Price)
}

// homeFigure is V_a of spec-migration.md 6.3.2: the observed wage over the cost
// of the household's upkeep at the local prices, in basis points; 0 when the
// window holds no working household-tick or the cost is not above 0.
func (w *World) homeFigure(a *agent, g RegionID, row *roleRow) int64 {
	wage, present := a.history.Wage()
	if !present {
		return 0
	}
	p := &w.migration.pass
	p.lines = p.lines[:0]
	for _, it := range row.upkeepIn(g) {
		p.lines = append(p.lines, mig.Line{Qty: int64(it.quantity), Price: w.localPrice(g, it.commodity)})
	}
	cost, ok := mig.Cost(p.lines)
	if !ok {
		return 0
	}
	v, ok := mig.Vhat(wage, cost)
	if !ok {
		return 0
	}
	return v
}

// evaluate is the evaluation of step 2 for one pushed agent: its candidates, their
// rumours (the stream keyed once when the role has a recipe and the error is above
// 0), the estimates, the four stages and the ranking. It returns the ranking
// (empty when none passed) and, then, the outcome.
func (w *World) evaluate(a *agent, g RegionID, row *roleRow, home int64) ([]mig.Ranked, mig.Outcome) {
	m := w.migration
	p := &m.pass
	margin, ok := mig.Margin(int64(a.id), m.cfg.MoveGain, m.cfg.MoveWidth)
	// Excluded from coverage (testing.md 9.5): the width is validated to 0 to MaxBasisPoints.
	if !ok {
		invariant("the margin of a validated migration module overflowed")
	}
	params := mig.Params{
		NoEstimate: len(row.recipes) == 0, Horizon: m.cfg.MoveHorizon, EntryRatio: w.cfg.World.EntryRatio,
		Home: home, Margin: margin,
	}
	p.cands = p.cands[:0]
	if !params.NoEstimate {
		w.candidatesOf(a, g, row)
	}
	ranked, outcome, ok := mig.Rank(p.ranked, p.cands, params)
	p.ranked = ranked
	if !ok {
		outcome = mig.NoGain // a product that cannot be formed shows no gain (12.2 makes it unreachable)
	}
	return ranked, outcome
}

// candidatesOf builds the candidates of step 2 into the scratch: the regions other
// than g with a way over the open edges (an outside node is the external module's
// and is left out there), their rumours as of tick t + 1, and for each the
// estimate. The stream of the draws is keyed once, before the first rumour.
func (w *World) candidatesOf(a *agent, g RegionID, row *roleRow) {
	m, tick := w.migration, w.Tick()
	p := &m.pass
	p.dests = p.dests[:0]
	for d := range RegionID(len(w.cfg.Regions)) {
		if _, ok := w.geo.distance(g, d); ok && d != g {
			p.dests = append(p.dests, d)
		}
	}
	if w.info.errorBP > 0 {
		w.rnd.keyed(siteMigration, tick, int64(a.id))
	}
	p.rumours = w.rumoursFor(p.rumours[:0], rumourQuery{reader: g, regions: p.dests, commodities: m.kset[a.role], asOf: tick + 1}, nil)
	commodities := len(w.cfg.Commodities)
	for _, r := range p.rumours {
		if r.Mean > 0 {
			p.price[int(r.Region)*commodities+int(r.Commodity)] = int64(r.Mean)
		}
	}
	for _, d := range p.dests {
		dist, _ := w.geo.distance(g, d)
		k, _ := mig.Ticks(dist, m.cfg.MoveSpeed)
		v, priced := w.estimateAt(row, d, tick+1+k)
		p.cands = append(p.cands, mig.Candidate{Ref: int(d), Priced: priced, K: k, Vhat: v})
		p.evaluated++
	}
	for _, r := range p.rumours {
		p.price[int(r.Region)*commodities+int(r.Commodity)] = 0
	}
}

// estimateAt is the estimate Vhat of spec-migration.md 6.3 at region d for the
// role of row, with the yields of the season of the arrival tick, and whether the
// candidate passes stage 1 (the upkeep fully priced and a recipe priced). An
// overflow makes the candidate unpriced.
func (w *World) estimateAt(row *roleRow, d RegionID, arrival int64) (vhat int64, priced bool) {
	p := &w.migration.pass
	p.lines, p.tools, p.recipes = p.lines[:0], p.tools[:0], p.recipes[:0]
	mo, err := w.cfg.Calendar.At(arrival)
	if err != nil {
		return 0, false // a tick the calendar cannot place (overflow) prices nothing
	}
	season := mo.Season
	for _, it := range row.upkeepIn(d) {
		p.lines = append(p.lines, mig.Line{Qty: int64(it.quantity), Price: w.rumoured(d, it.commodity)})
	}
	upkeep := p.lines[:len(p.lines):len(p.lines)] // the recipes below append after it
	for _, rid := range row.recipes {
		p.recipes = append(p.recipes, w.recipeAt(rid, d, season))
	}
	income, recipePriced, ok := mig.Income(p.recipes)
	cost, costOK := mig.Cost(upkeep)
	if !ok || !costOK || !recipePriced || !mig.Priced(upkeep) {
		return 0, false
	}
	v, ok := mig.Vhat(income, cost)
	return v, ok
}

// recipeAt is recipe rid as region d prices it: its inputs, tools and non-discard
// outputs at the rumoured prices and the table yield of the season.
func (w *World) recipeAt(rid RecipeID, d RegionID, season int) mig.Recipe {
	p := &w.migration.pass
	rr, rc := &w.recipes.rows[rid], &w.cfg.Recipes[rid]
	site, _ := w.yields.site(d, rid, season, rc.Sited)
	r := mig.Recipe{Duration: rc.Duration, Yield: site}
	in := len(p.lines)
	for _, it := range rr.inputs {
		p.lines = append(p.lines, mig.Line{Qty: int64(it.quantity), Price: w.rumoured(d, it.commodity)})
	}
	out := len(p.lines)
	for _, o := range rr.outputs {
		if !o.discard {
			p.lines = append(p.lines, mig.Line{Qty: int64(o.quantity), Price: w.rumoured(d, o.commodity)})
		}
	}
	r.Inputs, r.Outputs = p.lines[in:out:out], p.lines[out:len(p.lines):len(p.lines)]
	first := len(p.tools)
	for _, t := range rr.tools {
		p.tools = append(p.tools, mig.Tool{Wear: t.wearPerUse, Durability: w.cfg.Commodities[t.commodity].Durability, Price: w.rumoured(d, t.commodity)})
	}
	r.Tools = p.tools[first:]
	return r
}

// rumoured is the price of commodity c in region d from the rumours of the agent
// under evaluation: the window mean of the source, 0 for none.
func (w *World) rumoured(d RegionID, c CommodityID) int64 {
	return w.migration.pass.price[int(d)*len(w.cfg.Commodities)+int(c)]
}

// admitAndDepart is steps 3 and 4: the agents with a ranking are admitted in the
// order of need against the guard, the caps and the provisioning, the outcomes of
// those not admitted are counted, and the admitted depart in the order of
// admission.
func (w *World) admitAndDepart() {
	m := w.migration
	p := &m.pass
	p.agents = p.agents[:0]
	for i := range p.evals {
		e := &p.evals[i]
		a := &w.agents[e.id]
		p.agents = append(p.agents, mig.Agent{
			Ref: i, ID: int64(e.id), Weight: a.weight, EarnedBP: e.earned, Origin: int(e.region),
			Bound: w.guarded(a, p.arena[e.first]), HoldTicks: w.holdTicks(a, e.region),
			Ranked: p.arena[e.first:e.last],
		})
	}
	w.setCaps()
	p.results = mig.Admit(p.results, p.agents, p.caps, &p.admit)
	for _, r := range p.results {
		e := &p.evals[r.Ref]
		a := &w.agents[e.id]
		if r.Outcome != mig.Started {
			p.tally[e.region].outcome[r.Outcome] += a.weight
			w.declined(a, r.Outcome)
			continue
		}
		p.tally[e.region].outcome[mig.Started] += a.weight
		w.depart(a, e, RegionID(r.Dest), r.K)
	}
	p.evals, p.arena = p.evals[:0], p.arena[:0]
}

// guarded reports whether the guard (spec-migration.md 7.2) refuses the agent for
// the head of its ranking: a region_bound obligation ends the consideration. Any
// other refusal cannot occur for a live account, and is a defect.
func (w *World) guarded(a *agent, head mig.Ranked) bool {
	reason, err := w.relocateRefusal(a.id, RegionID(head.Ref), nil)
	if err == nil {
		return false
	}
	// Excluded from coverage (testing.md 9.5): the other refusals of 6.1.3 cannot occur for a
	// live account in a region with no reservation, so the branch is a violated invariant.
	if reason != ReasonRegionBound {
		invariant("the guard of the migration pass refused with " + string(reason) + ", which a live account cannot meet")
	}
	return true
}

// holdTicks is the longest journey, in ticks, that the stock of the agent
// provisions (spec-migration.md 7.4): the least over the upkeep commodities of its
// region of floor(held / (weight * quantity)); the largest tick count when the
// upkeep is empty.
func (w *World) holdTicks(a *agent, g RegionID) int64 {
	ticks := int64(math.MaxInt64)
	for _, it := range w.roles.row(a.role).upkeepIn(g) {
		held, _, err := w.ledger.Holding(a.account, int(it.commodity))
		// Excluded from coverage (testing.md 9.5): the holding of a live account is readable.
		if err != nil {
			invariant("the ledger refused the holding of a live agent: " + err.Error())
		}
		if n, ok := mig.Hold(held, a.weight, int64(it.quantity)); ok {
			ticks = min(ticks, n)
		}
	}
	return ticks
}

// setCaps fills the caps of the pass from the counts of step 1: the outflow cap
// of each region as an origin, the pipeline cap as a destination and the households
// already on the way to it.
func (w *World) setCaps() {
	m := w.migration
	p := &m.pass
	for g := range p.households {
		out, ok := mig.CapOf(p.households[g], m.outShare[g])
		in, ok2 := mig.CapOf(p.households[g], m.inShare[g])
		// Excluded from coverage (testing.md 9.5): validation (12.2) bounds households times share.
		if !ok || !ok2 {
			invariant("the caps of a validated migration module overflowed")
		}
		p.caps.Out[g], p.caps.In[g] = out, in
	}
	copy(p.caps.Pipeline, p.pipeline)
}

// depart is step 4 for one admitted agent (spec-migration.md 8.2): the job
// abandoned with region_changed, the floor dropped, the journey recorded, and the
// departure kept for MigrationStarted.
func (w *World) depart(a *agent, e *evalRec, to RegionID, k int64) {
	m, tick := w.migration, w.Tick()
	jobAbandoned := a.hasJob()
	if jobAbandoned {
		w.abandonJob(a, e.region, ReasonRegionChanged)
	}
	if w.sub != nil {
		w.sub.drop(a.id)
	}
	arrives := tick + 1 + k
	m.depart(a.id, to, arrives)
	dist, _ := w.geo.distance(e.region, to)
	veff := int64(0)
	for _, r := range m.pass.arena[e.first:e.last] {
		if RegionID(r.Ref) == to {
			veff = r.VEff
		}
	}
	m.pass.started = append(m.pass.started, startRec{
		agent: a.id, from: e.region, to: to, weight: a.weight, arrives: arrives, distance: dist,
		k: k, earned: e.earned, home: e.home, estimate: veff, jobAbandoned: jobAbandoned,
	})
}
