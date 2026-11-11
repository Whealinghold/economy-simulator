package economysim

import (
	"context"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	mig "github.com/Whealinghold/economy-sim/internal/migration"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// The tests of the migration pass at P7.2 (plan 2-14 Phase 6). The fixture is
// infoConfig (three regions a, b and c on a line; agents 3 and 4 millers in a, 5 and
// 6 millers in b, 7 and 8 carriers in c) whose households trade grain and flour in
// pairs at a fixed price of each region, so every market publishes it and the
// rumours of the other regions carry it. A warm-up of a few ticks fills the price
// windows and the record; then the gate is opened (every household is pushed) and
// the pass is called by hand at the tick the test names.

// pairPrice is the price at which a commodity clears in a region, 0 where the
// region trades none. Grain is 1 000 in a, 1 500 in b and 800 in c; flour 3 300
// in a and 4 500 in b. A miller's recipe turns 2 grain into 3 flour in one tick.
func pairPrice(region RegionID, commodity CommodityID) Price {
	switch {
	case commodity == grain && region == infoA:
		return 1_000
	case commodity == grain && region == infoB:
		return 1_500
	case commodity == grain && region == infoC:
		return 800
	case commodity == flour && region == infoA:
		return 3_300
	case commodity == flour && region == infoB:
		return 4_500
	}
	return 0
}

// pairScript trades one unit of grain and of flour a tick in each region where
// pairPrice is above 0: the odd agent asks and the even agent bids at that price
// (both limits equal, so the market publishes exactly it). It runs no job.
type pairScript struct{}

func (pairScript) Work(*AgentView, *rand.Rand) JobRequest { return JobRequest{NoJob: true} }

func (pairScript) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	side := Bid
	if v.Agent%2 == 1 {
		side = Ask
	}
	for _, c := range []CommodityID{flour, grain} {
		if price := pairPrice(v.Region, c); price > 0 {
			p.Intents = append(p.Intents, Intent{Commodity: c, Side: side, Quantity: 1, Limit: price})
		}
	}
}

func (pairScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = pairScript{}

// passWorld builds the fixture with the module on, a review every tick and the
// gate shut (MoveBelow 0, so nobody is pushed while the markets warm up), changed
// by mut, and steps it warm ticks. Every agent is settled long ago.
func passWorld(t *testing.T, warm int, mut func(c *Config), opts ...Option) (*World, *evLog) {
	t.Helper()
	rec := &evLog{}
	w := newMill(t, passConfig(mut), append([]Option{WithStrategy("script", pairScript{}), WithAudit(), WithObserver(rec)}, opts...)...)
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		w.agents[i].roleSince = -1_000
	}
	step(t, w, warm)
	return w, rec
}

// passConfig is the configuration of passWorld, changed by mut.
func passConfig(mut func(c *Config)) Config {
	c := infoConfig()
	m := DefaultMigration()
	m.MoveSpeed, m.MoveReviewTicks, m.MoveBelow = infoSpeed, 1, 0
	c.Modules.Migration = &m
	for i := range c.Roles {
		c.Roles[i].StartStock = []Item{{Commodity: "grain", Quantity: 1_000}, {Commodity: "flour", Quantity: 1_000}}
		c.Roles[i].StartMoney = 1_000_000
	}
	if mut != nil {
		mut(&c)
	}
	return c
}

// open raises the gate to its bound, so the earned ratio of any agent that has an
// upkeep is below it and every household is pushed.
func open(w *World) { w.migration.cfg.MoveBelow = MaxBasisPoints }

// TestPass_EligibilityAndRotation verifies spec-migration.md 3.3 and 5.1: over one
// period every agent is due exactly once, an agent of a Fixed role, an external agent
// and an agent on a journey are never due, and the scan counts every agent.
//
// Requirement: FR-MIG-005, FR-MIG-007, FR-MIG-045
func TestPass_EligibilityAndRotation(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 0, func(c *Config) {
		c.Modules.Migration.MoveReviewTicks = 5
		for i := range c.Roles {
			c.Roles[i].Fixed = c.Roles[i].ID == "carrier"
		}
	})
	w.agents[4].external = true
	w.migration.depart(5, infoA, 1_000)
	count := map[AgentID]int{}
	for range 5 {
		step(t, w, 1)
		if w.migration.pass.visited != 6 {
			t.Errorf("tick %d: the scan visited %d agents, want 6", w.Tick()-1, w.migration.pass.visited)
		}
		for _, id := range w.migration.pass.due {
			count[id]++
		}
	}
	// 3 is due once; 4 is external, 5 is on a journey, 7 and 8 are of a Fixed role; 6 is due once.
	if want := map[AgentID]int{3: 1, 6: 1}; !reflect.DeepEqual(count, want) {
		t.Errorf("due counts over one period of 5 = %v, want %v", count, want)
	}
}

// TestPass_CandidatesAndK verifies spec-migration.md 6.1 and 6.4: the candidates of
// a household in a are the regions with a way over the open edges, ascending, each
// with its journey time ceil(distance / speed) (40 to b: 1; 120 to c: 3), and a
// closed edge takes its region off the list. The work counter counts the candidates
// evaluated.
//
// Requirement: FR-MIG-011, FR-MIG-046, FR-MIG-016
func TestPass_CandidatesAndK(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 6, nil)
	row := w.roles.row(w.agents[3].role)
	cands := func() (refs, ks []int64) {
		w.migration.pass.cands = w.migration.pass.cands[:0]
		w.candidatesOf(&w.agents[3], infoA, row)
		for _, c := range w.migration.pass.cands {
			refs, ks = append(refs, int64(c.Ref)), append(ks, c.K)
		}
		return refs, ks
	}
	if refs, ks := cands(); !slices.Equal(refs, []int64{1, 2}) || !slices.Equal(ks, []int64{1, 3}) {
		t.Errorf("candidates of agent 3 = regions %v, journey times %v, want regions [1 2] and [1 3]", refs, ks)
	}
	if err := w.setEdgeStatuses([]EdgeStatus{{From: infoB, To: infoC, Closed: true, FactorBP: 10_000}}, CauseConsumer); err != nil {
		t.Fatalf("setEdgeStatuses() = %v, want nil", err)
	}
	if refs, _ := cands(); !slices.Equal(refs, []int64{1}) {
		t.Errorf("candidates of agent 3 with b-c closed = regions %v, want [1]", refs)
	}
}

// TestPass_EstimateAndRanking verifies the estimate of spec-migration.md 6.3 and
// 6.4 on prices the rumours carry. A miller in a prices region b at grain 1 500 and
// flour 4 500: the recipe nets 3 * 4 500 - 2 * 1 500 = 10 500 a tick, the upkeep of
// one grain costs 1 500, so Vhat = 10 500 * 10 000 / 1 500 = 70 000, discounted by
// the journey of 1 tick of a horizon of 180: 70 000 * 179 / 180 = 69 611. Region c
// has no flour, so no recipe is priced there: stage 1. A miller in b prices a at
// 3 * 3 300 - 2 * 1 000 = 7 900 over 1 000: 79 000, and 79 000 * 179 / 180 = 78 561.
//
// Requirement: FR-MIG-012, FR-MIG-014, FR-MIG-016, FR-MIG-017, FR-MIG-018
func TestPass_EstimateAndRanking(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 6, nil)
	open(w)
	tests := []struct {
		agent  AgentID
		region RegionID
		veff   int64
		dest   RegionID
	}{
		{3, infoA, 69_611, infoB},
		{5, infoB, 78_561, infoA},
	}
	for _, tt := range tests {
		a := &w.agents[tt.agent]
		row := w.roles.row(a.role)
		ranked, outcome := w.evaluate(a, tt.region, row, 0)
		if len(ranked) != 1 || ranked[0].Ref != int(tt.dest) || ranked[0].VEff != tt.veff || ranked[0].K != 1 {
			t.Errorf("evaluate(agent %d) = %+v (outcome %d), want one candidate, region %d, V_eff %d, k 1", tt.agent, ranked, outcome, tt.dest, tt.veff)
		}
	}
	// A carrier has no recipe: no estimate, no ranking.
	if ranked, outcome := w.evaluate(&w.agents[7], infoC, w.roles.row(w.agents[7].role), 0); len(ranked) != 0 || outcome != 1 {
		t.Errorf("evaluate(agent 7) = %+v, outcome %d, want none and no_estimate (1)", ranked, outcome)
	}
	// Under the gate of stage 4 a home figure above the estimate keeps the household.
	if ranked, outcome := w.evaluate(&w.agents[3], infoA, w.roles.row(w.agents[3].role), 100_000); len(ranked) != 0 || outcome != 6 {
		t.Errorf("evaluate(agent 3, home 100 000) = %+v, outcome %d, want none and no_gain (6)", ranked, outcome)
	}
}

// TestPass_RumourDrops verifies spec-migration.md 6.2: a destination whose market
// has not traded within its window has a mean of 0 and is no_rumour, as is a
// destination reached before its news (the pass reads as of tick t + 1, and the
// lag from b to a is 2).
//
// Requirement: FR-MIG-012
func TestPass_RumourDrops(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 0, nil)
	open(w)
	a := &w.agents[3]
	ranked, outcome := w.evaluate(a, infoA, w.roles.row(a.role), 0)
	if len(ranked) != 0 || outcome != 3 {
		t.Errorf("evaluate(agent 3) at tick 0 = %+v, outcome %d, want none and no_rumour (3): no market has traded", ranked, outcome)
	}
}

// TestPass_DrawsInOrder verifies spec-migration.md 6.2.1 and D11: with an error
// above 0 the pass keys (siteMigration, tick, agent) once for a pushed agent with a
// recipe and the rumours it reads are those of a reference stream keyed the same way;
// with the error 0 nothing is drawn; a world whose gate stays shut draws nothing, so
// its events are those of a world without the module.
//
// Requirement: FR-MIG-013, FR-INF-051
func TestPass_DrawsInOrder(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 6, func(c *Config) { c.Modules.Information.RumourErrorBP = 500 })
	open(w)
	a := &w.agents[3]
	w.candidatesOf(a, infoA, w.roles.row(a.role))
	got := slices.Clone(w.migration.pass.rumours)
	tick := w.Tick()
	w.rnd.keyed(siteMigration, tick, int64(a.id))
	want := w.rumoursFor(nil, rumourQuery{reader: infoA, regions: []RegionID{infoB, infoC}, commodities: w.migration.kset[a.role], asOf: tick + 1}, nil)
	if len(got) == 0 || !reflect.DeepEqual(got, want) {
		t.Errorf("rumours read by the pass = %+v, want those of the stream keyed (siteMigration, %d, %d): %+v", got, tick, a.id, want)
	}

	offRec := &evLog{}
	off := newMill(t, infoConfigFor(false), WithStrategy("script", pairScript{}), WithObserver(offRec))
	on, onRec := passWorld(t, 0, nil)
	step(t, off, 8)
	step(t, on, 8)
	if got, want := of[TradeExecuted](onRec), of[TradeExecuted](offRec); len(want) == 0 || !reflect.DeepEqual(got, want) {
		t.Errorf("trades of a world whose gate is shut = %d, those of a world without the module = %d, want the same trades", len(got), len(want))
	}
}

// infoConfigFor is passWorld's configuration without or with the module.
func infoConfigFor(module bool) Config {
	c := infoConfig()
	for i := range c.Roles {
		c.Roles[i].StartStock = []Item{{Commodity: "grain", Quantity: 1_000}, {Commodity: "flour", Quantity: 1_000}}
		c.Roles[i].StartMoney = 1_000_000
	}
	if module {
		m := DefaultMigration()
		m.MoveSpeed = infoSpeed
		c.Modules.Migration = &m
	}
	return c
}

// TestPass_HomeFigure verifies spec-migration.md 6.3.2: the home figure is the
// observed wage of the window over the cost of the upkeep at the local prices, in
// basis points, and 0 when the window holds no working household-tick. With 4
// working ticks of income 2 000 each the wage is 2 000, the cost of one grain at
// 1 000 (the window mean of a) gives 2 000 * 10 000 / 1 000 = 20 000.
//
// Requirement: FR-MIG-015
func TestPass_HomeFigure(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 6, nil)
	a := &w.agents[3]
	row := w.roles.row(a.role)
	a.history = newHistory(row) // the warm-up's trades are not work: a fresh window
	if got := w.homeFigure(a, infoA, row); got != 0 {
		t.Errorf("homeFigure with no working tick = %d, want 0", got)
	}
	for range 4 {
		a.history.Push(welfare.Record{Income: 2_000, Cost: 1_000, Working: true, WorkHH: 1})
	}
	if got := w.homeFigure(a, infoA, row); got != 20_000 {
		t.Errorf("homeFigure after 4 working ticks of income 2 000 = %d, want 20 000", got)
	}
	// Before any trade the window is empty and the published price (the seed 1 000) is read.
	fresh, _ := passWorld(t, 0, nil)
	b := &fresh.agents[3]
	for range 4 {
		b.history.Push(welfare.Record{Income: 2_000, Cost: 1_000, Working: true, WorkHH: 1})
	}
	if got := fresh.homeFigure(b, infoA, fresh.roles.row(b.role)); got != 20_000 {
		t.Errorf("homeFigure with an empty price window = %d, want 20 000 at the seed price", got)
	}
}

// TestPass_AdmissionAndOutcomes verifies spec-migration.md 7 and 7.6 on the fixture:
// with every household pushed and the caps at 1 (ceil(2 * 50 / 10 000) out of a and
// of b, ceil(2 * 100 / 10 000) into each), the poorer of the two millers of a and
// of b starts and the other is held by the outflow cap, the carriers of c end in
// no_estimate, and the outcomes of each region add to its pushed households.
//
// Requirement: FR-MIG-018, FR-MIG-019, FR-MIG-021, FR-MIG-025
func TestPass_AdmissionAndOutcomes(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 6, nil)
	open(w)
	w.p72Migration()
	tally := w.migration.pass.tally
	for g, want := range map[RegionID][mig.Outcomes]int64{
		infoA: {0: 1, 8: 1}, infoB: {0: 1, 8: 1}, infoC: {1: 2},
	} {
		var sum int64
		for _, n := range tally[g].outcome {
			sum += n
		}
		if tally[g].outcome != want || sum != tally[g].pushed || tally[g].reviewed != 2 {
			t.Errorf("region %d: outcomes %v pushed %d reviewed %d, want outcomes %v adding to pushed 2 and reviewed 2", g, tally[g].outcome, tally[g].pushed, tally[g].reviewed, want)
		}
	}
	if len(w.migration.active) != 2 || len(w.migration.pass.started) != 2 {
		t.Fatalf("journeys %v, departures %d, want two of each", w.migration.active, len(w.migration.pass.started))
	}
	for _, st := range w.migration.pass.started {
		mate := mateOf(st.agent)
		if st.earned > w.pushedEarned(mate) {
			t.Errorf("agent %d started with earned ratio %d, but its mate %d has %d: the poorer goes first", st.agent, st.earned, mate, w.pushedEarned(mate))
		}
	}
}

// mateOf is the other miller of the region: 3 and 4 are in a, 5 and 6 in b.
func mateOf(id AgentID) AgentID {
	if id%2 == 1 {
		return id + 1
	}
	return id - 1
}

// pushedEarned is the earned ratio the pass reads for an agent now.
func (w *World) pushedEarned(id AgentID) int64 {
	a := &w.agents[id]
	e, _ := a.history.EarnedRatio(w.upkeepCost(a, w.regionOf(a), w.roles.row(a.role)), w.cfg.Roles[a.role].DefaultRatio)
	return e
}

// TestPass_DepartureOrder verifies spec-migration.md 8.2: a household with a job in
// progress abandons it with region_changed, then the record is set with the arrival
// tick t + 1 + k, and nothing else about the agent moves.
//
// Requirement: FR-MIG-026, FR-MIG-034
func TestPass_DepartureOrder(t *testing.T) {
	t.Parallel()
	w, rec := passWorld(t, 6, nil)
	open(w)
	// Both millers of a have a job in progress; one of them is admitted (the outflow cap of a is 1).
	for _, id := range []AgentID{3, 4} {
		w.agents[id].job = job{recipe: 0, executions: 1, remaining: 2}
	}
	rec.events = nil
	tick := w.Tick()
	w.p72Migration()
	started := w.migration.pass.started
	var mover *startRec
	for i := range started {
		if started[i].from == infoA {
			mover = &started[i]
		}
	}
	if mover == nil || !mover.jobAbandoned || w.agents[mover.agent].hasJob() {
		t.Fatalf("departures %+v: want one from a that abandoned a job, job in progress after: %v", started, mover != nil && w.agents[mover.agent].hasJob())
	}
	abandoned := of[RecipeAbandoned](rec)
	if len(abandoned) != 1 || abandoned[0].Agent != mover.agent || abandoned[0].Reason != ReasonRegionChanged {
		t.Errorf("RecipeAbandoned = %+v, want one for agent %d with region_changed", abandoned, mover.agent)
	}
	if j := w.migration.journey[mover.agent]; !j.live || j.to != infoB || j.arrives != tick+2 || mover.arrives != tick+2 {
		t.Errorf("journey of agent %d = %+v, want to b arriving at %d (tick %d + 1 + k 1)", mover.agent, j, tick+2, tick)
	}
	if other := w.agents[mateOf(mover.agent)]; !other.hasJob() {
		t.Errorf("the miller that stayed lost its job, want it kept")
	}
	if got := w.regionOf(&w.agents[mover.agent]); got != infoA {
		t.Errorf("region of the migrant after the departure = %d, want %d: it stays until it arrives", got, infoA)
	}
}

// TestPass_NoMoneyNoGoodsMove verifies spec-migration.md 9: the pass moves neither
// money nor goods, so the ledger is the same before and after it, and the audit holds.
//
// Requirement: FR-MIG-033
func TestPass_NoMoneyNoGoodsMove(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 6, nil)
	open(w)
	before := execLedgerState(w)
	w.p72Migration()
	if after := execLedgerState(w); !slices.Equal(after, before) || len(w.migration.active) == 0 {
		t.Errorf("ledger changed by the pass or no household left: journeys %v", w.migration.active)
	}
	if err := w.ledger.Audit(); err != nil {
		t.Errorf("Audit() after the pass = %v, want nil", err)
	}
	step(t, w, 4) // the journeys end and the household arrives; the audit runs each step
}

// TestPass_Deterministic verifies spec-migration.md 14 and FR-MIG-042: who moves,
// where and when does not depend on the order of the regions, edges, groups and
// roles in the configuration.
//
// Requirement: FR-MIG-042
func TestPass_Deterministic(t *testing.T) {
	t.Parallel()
	run := func(permute bool) []Event {
		w, rec := passWorld(t, 0, func(c *Config) {
			c.Modules.Migration.MoveBelow = MaxBasisPoints
			c.Modules.Migration.MoveReviewTicks = 3
			if permute {
				slices.Reverse(c.Edges)
				slices.Reverse(c.Groups)
				slices.Reverse(c.Roles)
				slices.Reverse(c.Regions)
			}
		})
		step(t, w, 14)
		return rec.events
	}
	a, b := run(false), run(true)
	moved := 0
	for _, e := range a {
		if _, ok := e.(AgentRelocated); ok {
			moved++
		}
	}
	if moved == 0 || len(a) != len(b) {
		t.Fatalf("events %d and %d, relocations %d, want the same number and some relocations", len(a), len(b), moved)
	}
	if got, want := of[AgentRelocated](&evLog{events: b}), of[AgentRelocated](&evLog{events: a}); !reflect.DeepEqual(got, want) {
		t.Errorf("relocations with the configuration permuted = %+v, want %+v", got, want)
	}
}

// TestPass_WorkIsCounted verifies NFR-MIG-002: a tick in which no agent is due costs
// the scan and evaluates no candidate, and a tick in which agents are due and pushed
// evaluates their candidates, no more.
//
// Requirement: NFR-MIG-002
func TestPass_WorkIsCounted(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 6, func(c *Config) { c.Modules.Migration.MoveReviewTicks = 1_000 })
	open(w)
	for w.migration.pass.due = w.migration.pass.due[:0]; ; step(t, w, 1) {
		if len(w.migration.pass.due) == 0 && w.Tick() > 6 {
			break
		}
		if w.Tick() > 7 {
			t.Fatalf("every tick had an agent due; the period of 1 000 should leave quiet ticks")
		}
	}
	if p := &w.migration.pass; p.visited != 6 || p.evaluated != 0 {
		t.Errorf("a quiet tick visited %d agents and evaluated %d candidates, want 6 and 0", p.visited, p.evaluated)
	}
	w.migration.cfg.MoveReviewTicks = 1
	w.p72Migration()
	if p := &w.migration.pass; p.visited != 6 || p.evaluated != 8 {
		// 4 millers with 2 candidates each; the carriers have no recipe and are not evaluated.
		t.Errorf("a tick with every agent due visited %d and evaluated %d candidates, want 6 and 8", p.visited, p.evaluated)
	}
}

// TestPass_PushIsEarnedRatio verifies spec-migration.md 5.3 and 5.4: relief is out
// of the trigger and out of the pull. Two worlds that differ only in a relief of 3 000
// paid to agent 3 in each tick of its window (its income is higher by that much) give
// the same earned ratio, the same home figure and the same tally.
//
// Requirement: FR-MIG-008, FR-MIG-009
func TestPass_PushIsEarnedRatio(t *testing.T) {
	t.Parallel()
	run := func(relief int64) (earned, home int64, tally passTally) {
		w, _ := passWorld(t, 6, nil)
		open(w)
		a := &w.agents[3]
		a.history = newHistory(w.roles.row(a.role))
		for range 5 {
			a.history.Push(welfare.Record{Income: 2_000 + relief, Relief: relief, Cost: 1_000, Working: true, WorkHH: 1})
		}
		w.p72Migration()
		return w.pushedEarned(3), w.homeFigure(a, infoA, w.roles.row(a.role)), w.migration.pass.tally[infoA]
	}
	e0, h0, t0 := run(0)
	e1, h1, t1 := run(3_000)
	if e0 != e1 || h0 != h1 || t0 != t1 {
		t.Errorf("with relief 3 000: earned %d, home %d, tally %+v; without: %d, %d, %+v; want equal", e1, h1, t1, e0, h0, t0)
	}
}

// TestPass_ReadsOnlyAllowedState verifies spec-migration.md 4.4 and N-157: another
// region's stock, household count and money leave the candidates, the estimates and
// the ranking of an agent unchanged. Prices are the only news of a region.
//
// Requirement: FR-MIG-010
func TestPass_ReadsOnlyAllowedState(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 6, nil)
	open(w)
	a := &w.agents[3]
	row := w.roles.row(a.role)
	snapshot := func() ([]mig.Ranked, []mig.Candidate) {
		ranked, _ := w.evaluate(a, infoA, row, 0)
		return slices.Clone(ranked), slices.Clone(w.migration.pass.cands)
	}
	r0, c0 := snapshot()
	give(t, w, 5, grain, 5_000)
	give(t, w, 6, flour, 5_000)
	w.agents[5].weight, w.agents[6].weight = 40, 50
	r1, c1 := snapshot()
	if !reflect.DeepEqual(r0, r1) || !reflect.DeepEqual(c0, c1) {
		t.Errorf("ranking %+v and candidates %+v after changing region b's stock and households, want %+v and %+v", r1, c1, r0, c0)
	}
}

// TestPass_ReadsNewsAsOfTheNextTick verifies the ruling C17 of spec-migration.md 5.1:
// the pass of tick t reads the rumours as of t + 1. The news of b reaches a in 2 ticks
// (1 + ceil(40 / 40)), so at tick 1 the observation of tick 0 is offered (lag 2 is not
// above 1 + 1) and a miller of a prices region b; read as of tick 1 it would be too
// early (lag 2 is above 1) and the household would see no rumour.
//
// Requirement: FR-MIG-012, FR-INF-051
func TestPass_ReadsNewsAsOfTheNextTick(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 1, nil)
	open(w)
	a := &w.agents[3]
	ranked, outcome := w.evaluate(a, infoA, w.roles.row(a.role), 0)
	if len(ranked) != 1 || ranked[0].Ref != int(infoB) {
		t.Errorf("evaluate(agent 3) at tick 1 = %+v, outcome %d, want region b ranked from the observation of tick 0", ranked, outcome)
	}
}

// TestPass_UnsettledCountedNotEvaluated verifies spec-migration.md 5.2.3: a household
// whose turn has come but that took up its place less than a window ago, and that the
// gate would push, is counted as unsettled and nothing else is computed for it: no
// candidate is evaluated, no stream is keyed and no outcome is recorded. Once settled
// it is reviewed.
//
// Requirement: FR-MIG-006, FR-MIG-007
func TestPass_UnsettledCountedNotEvaluated(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 6, nil)
	open(w)
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		w.agents[i].roleSince = w.Tick() // took up its place now
	}
	w.p72Migration()
	p := &w.migration.pass
	var unsettled, reviewed, pushed int64
	for _, tl := range p.tally {
		unsettled, reviewed, pushed = unsettled+tl.unsettled, reviewed+tl.reviewed, pushed+tl.pushed
	}
	if unsettled != 6 || reviewed != 0 || pushed != 0 || p.evaluated != 0 || len(p.started) != 0 {
		t.Errorf("unsettled %d reviewed %d pushed %d evaluated %d departures %d, want 6 0 0 0 0", unsettled, reviewed, pushed, p.evaluated, len(p.started))
	}
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		w.agents[i].roleSince = -1_000
	}
	w.p72Migration()
	if p.tally[infoA].unsettled != 0 || p.tally[infoA].reviewed != 2 {
		t.Errorf("settled: region a unsettled %d reviewed %d, want 0 and 2", p.tally[infoA].unsettled, p.tally[infoA].reviewed)
	}
}

// TestPass_BoundByAnObligation verifies the guard of spec-migration.md 7.2: a household
// that another region-bound obligation holds ends in bound, uses none of the outflow
// cap, and the other miller of its region is admitted in its place.
//
// Requirement: FR-MIG-020, FR-MIG-024
func TestPass_BoundByAnObligation(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 6, nil)
	open(w)
	w.bounds.add("open_shipment", func(a *agent, _ []string) bool { return a.id == 4 })
	w.p72Migration()
	tl := w.migration.pass.tally[infoA]
	if tl.outcome[mig.Bound] != 1 || tl.outcome[mig.Started] != 1 || !w.migration.on(3) || w.migration.on(4) {
		t.Errorf("region a: outcomes %v, journeys of 3 and 4 %v %v, want one bound (agent 4) and one started (agent 3)", tl.outcome, w.migration.on(3), w.migration.on(4))
	}
}

// TestPass_WorkedExampleBJourneyTimes verifies the table of journey times of
// spec-migration.md 15.2 on its three regions (belt a, orbital b, surface c) and its
// edges, at a speed of 3 000: k = ceil(D / 3 000) is 1, 2 from the belt, 1, 1 from the
// orbital and 5, 4 from the surface, and the candidates of a household are the other two
// regions in ascending order.
//
// Requirement: FR-MIG-016, FR-MIG-046
func TestPass_WorkedExampleBJourneyTimes(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 0, func(c *Config) {
		c.Edges = []Edge{
			{From: "a", To: "b", Distance: 3_000}, {From: "a", To: "c", Distance: 5_000},
			{From: "b", To: "a", Distance: 3_000}, {From: "b", To: "c", Distance: 2_000},
			{From: "c", To: "a", Distance: 12_400}, {From: "c", To: "b", Distance: 9_400},
		}
		c.Modules.Migration.MoveSpeed = 3_000
	})
	for _, tt := range []struct {
		agent  AgentID
		region RegionID
		want   [2]int64
	}{
		{3, infoA, [2]int64{1, 2}}, // belt to orbital, belt to surface
		{5, infoB, [2]int64{1, 1}}, // orbital to belt, orbital to surface
		{7, infoC, [2]int64{5, 4}}, // surface to belt, surface to orbital
	} {
		a := &w.agents[tt.agent]
		w.migration.pass.cands = w.migration.pass.cands[:0]
		w.candidatesOf(a, tt.region, w.roles.row(a.role))
		cands := w.migration.pass.cands
		if len(cands) != 2 || cands[0].K != tt.want[0] || cands[1].K != tt.want[1] || cands[0].Ref >= cands[1].Ref {
			t.Errorf("candidates of agent %d in region %d = %+v, want two in region order with journey times %v", tt.agent, tt.region, cands, tt.want)
		}
	}
}

// BenchmarkPass reports the cost and the allocations of the pass of P7.2 with a
// review every tick and the gate open. It only reports:
// TestPass_WorkIsCounted enforces the limit of NFR-MIG-002.
//
// Requirement: NFR-MIG-002
func BenchmarkPass(b *testing.B) {
	c := passConfig(func(c *Config) { c.Modules.Migration.MoveReviewTicks = 1 })
	w, err := NewWorld(c, WithStrategy("script", pairScript{}))
	if err != nil {
		b.Fatal(err)
	}
	if _, err := w.Run(context.Background(), 6); err != nil {
		b.Fatal(err)
	}
	open(w)
	b.ReportAllocs()
	for b.Loop() {
		w.p72Migration()
	}
}
