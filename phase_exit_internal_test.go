package economysim

import (
	"log/slog"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// exitSeed is the seed of the tests of the entry step of phase 7; the failure
// messages name it (testing.md 6.2).
const exitSeed = 20261020

// exitConfig is panConfig with a second role: miller (role 0, 3 households in
// two agents of weight 2 and 1, start money 7 000, the pan tool) and weaver
// (role 1, one agent, closed, start money 3 000, 200 expected working ticks, 4
// ticks of training). Agent IDs: 0 the estate, 1 and 2 the millers, 3 the weaver.
func exitConfig() Config {
	c := panConfig()
	c.World.MinAgents = 1
	c.Roles[0].StartMoney = 7_000
	weaver := DefaultRole()
	weaver.ID, weaver.StartMoney, weaver.ExpectedWorkTicks = "weaver", 3_000, 200
	weaver.TrainingTicks, weaver.Closed = 4, true
	c.Roles = append(c.Roles, weaver)
	c.Groups = []Group{
		{Region: "home", Role: "miller", Households: 3, Scale: 2},
		{Region: "home", Role: "weaver", Households: 1},
	}
	return c
}

// exitWorld builds the world of exitConfig with the options and seeds the pool of
// the region with 25 000, while the ledger is still in construction.
func exitWorld(t *testing.T, opts ...Option) *World {
	t.Helper()
	w := newMill(t, exitConfig(), append([]Option{WithSeed(exitSeed)}, opts...)...)
	if err := w.estates.seedPool(w.ledger, 0, 25_000); err != nil {
		t.Fatalf("seed %d: seedPool() error = %v, want nil", exitSeed, err)
	}
	return w
}

// TestBuildRegionView_EveryFieldEqualsItsSource verifies the view of a scripted
// region field by field, by exact equality (spec-spawner.md 3.2).
//
// Arithmetic: the pool is 25 000 and EntryPoolShare is 5 000 basis points, so the
// Budget is 25 000 * 5 000 / 10 000 = 12 500. One household of upkeep eats 1
// grain at the published price 1 000, so Upkeep is 1 000 for both roles. The pan
// is the one tool of miller's recipe, at its seed price 5 000, so ToolPrice is
// 5 000 for miller and 0 for weaver. The millers' rings hold one record each:
// agent 1 (weight 2) earned 1 000 over 2 working household-ticks and agent 2
// (weight 1) 300 over 1, so the observed wage of miller is (1 000 + 300) /
// (2 + 1) = 433 (truncated), and weaver, with no working tick, has none. A
// household starts with its role's StartMoney, so agent 1 holds 2 * 7 000 =
// 14 000, agent 2 7 000 and agent 3 3 000. The CapitalReturn is the region's,
// 1 000, and a role's RenewalCarry is the world's, here 42 and 7.
//
// Requirement: FR-SPN-001, FR-SPN-002
func TestBuildRegionView_EveryFieldEqualsItsSource(t *testing.T) {
	t.Parallel()
	w := exitWorld(t)
	w.agents[1].history.Push(welfare.Record{Income: 1_000, Working: true, WorkHH: 2})
	w.agents[2].history.Push(welfare.Record{Income: 300, Working: true, WorkHH: 1})
	w.state.renewalCarry[0], w.state.renewalCarry[1] = 42, 7
	a1, a3 := &w.agents[1], &w.agents[3]
	a1.unfed, a1.hunger, a1.privation, a1.roleSince = 1, 2, 3, 4
	a3.external, a3.mortal, a3.privation, a3.roleSince = true, true, 9, -2
	got := *w.buildRegionView(0)
	want := RegionView{
		Tick: 0, Region: 0, TicksPerYear: w.cfg.Calendar.TicksPerYear(), PoolMoney: 25_000, Budget: 12_500,
		ReviewTicks: 10, EntryShare: 50, RoleHysteresis: 1_000, RoleCooldown: 30, EntryRatio: 8_000,
		ChangePrivation: 3, RemovePrivation: 10, DeathStep: 1_000, ExitShare: 100, Renewal: 10_000, EntryPoolShare: 5_000,
		Roles: []RoleView{
			{Role: 0, Households: 3, Agents: 2, Wage: 433, HasWage: true, Upkeep: 1_000, ExpectedWorkTicks: 250, StartMoney: 7_000, ToolPrice: 5_000, CapitalReturn: 1_000, RenewalCarry: 42},
			{Role: 1, Closed: true, Households: 1, Agents: 1, Upkeep: 1_000, ExpectedWorkTicks: 200, StartMoney: 3_000, TrainingTicks: 4, CapitalReturn: 1_000, RenewalCarry: 7},
		},
		Agents: []RegionAgent{
			{Agent: 1, Role: 0, Weight: 2, Money: 14_000, Unfed: 1, Hunger: 2, Privation: 3, RoleSince: 4, Upkeep: 1_000},
			{Agent: 2, Role: 0, Weight: 1, Money: 7_000, Upkeep: 1_000},
			{Agent: 3, Role: 1, Weight: 1, External: true, Mortal: true, Money: 3_000, Privation: 9, RoleSince: -2, Upkeep: 1_000},
		},
	}
	if got.TicksPerYear != 365 {
		t.Errorf("seed %d: TicksPerYear = %d, want 365 of the default calendar", exitSeed, got.TicksPerYear)
	}
	got.observed, got.scratch = false, nil
	if !slices.Equal(got.Roles, want.Roles) || !slices.Equal(got.Agents, want.Agents) {
		t.Errorf("seed %d: roles = %+v, agents = %+v, want %+v and %+v", exitSeed, got.Roles, got.Agents, want.Roles, want.Agents)
	}
	got.Roles, got.Agents, want.Roles, want.Agents = nil, nil, nil, nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("seed %d: view = %+v, want %+v", exitSeed, got, want)
	}
}

// TestBuildRegionView_ObservedFlagAndRemovedAgents verifies that the view carries
// the observer flag set once at construction, and that a removed agent (weight 0)
// is not in it while the others keep ascending IDs.
//
// Requirement: FR-SPN-002, FR-SPN-035
func TestBuildRegionView_ObservedFlagAndRemovedAgents(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		opts []Option
		want bool
	}{{"no observer", nil, false}, {"observer", []Option{WithObserver(&evLog{})}, true}} {
		w := exitWorld(t, tt.opts...)
		w.agents[2] = agent{}
		v := w.buildRegionView(0)
		if v.observed != tt.want || v.scratch == nil {
			t.Errorf("%s (seed %d): observed = %v, scratch set = %v, want %v and true", tt.name, exitSeed, v.observed, v.scratch != nil, tt.want)
		}
		if len(v.Agents) != 2 || v.Agents[0].Agent != 1 || v.Agents[1].Agent != 3 {
			t.Errorf("%s (seed %d): agents = %+v, want agents 1 and 3", tt.name, exitSeed, v.Agents)
		}
	}
}

// planFake is a consumer Spawner (D142): it records what it sees at each call
// and then leaves a decision in the plan, so a plan not reset before the next
// region shows.
type planFake struct {
	calls []fakeCall
}

type fakeCall struct {
	tick       int64
	region     RegionID
	planLen    int
	word       uint64
	households int64
}

func (f *planFake) Decide(v *RegionView, rng *rand.Rand, d *Decisions) {
	f.calls = append(f.calls, fakeCall{v.Tick, v.Region, len(d.List), rng.Uint64(), v.Roles[0].Households})
	d.List = append(d.List, Decision{Kind: DecideRemove, Agent: 1})
}

// TestDecideRegion_EmptyPlanAndKeyedStream verifies that Decide is called with an
// empty plan each time and, for a consumer's spawner, with the stream keyed by
// (siteSpawner, tick, region): its first word is the first word of an independent
// stream with that key. The default takes no word, so the world does not reseed
// for it: the stream is where the previous key left it.
//
// Requirement: FR-SPN-002, FR-WLD-062
func TestDecideRegion_EmptyPlanAndKeyedStream(t *testing.T) {
	t.Parallel()
	fake := &planFake{}
	c := exitConfig()
	c.World.Spawner = "fake"
	w := newMill(t, c, WithSeed(exitSeed), WithSpawner("fake", fake))
	for range 2 {
		if got := len(w.decideRegion(0).List); got != 1 {
			t.Fatalf("seed %d: plan length after Decide = %d, want the fake's 1 decision", exitSeed, got)
		}
	}
	ref := newStream(exitSeed)
	ref.keyed(siteSpawner, 0, 0)
	want := ref.word()
	for i, call := range fake.calls {
		if call.planLen != 0 || call.word != want || call.tick != 0 || call.region != 0 || call.households != 3 {
			t.Errorf("seed %d: call %d saw %+v, want an empty plan, word %d, tick 0, region 0, 3 households", exitSeed, i, call, want)
		}
	}
	d := exitWorld(t)
	d.rnd.keyed(siteWork, 1, 1)
	d.decideRegion(0)
	ref = newStream(exitSeed)
	ref.keyed(siteWork, 1, 1)
	if got, want := d.rnd.word(), ref.word(); got != want {
		t.Errorf("seed %d: next word after the default's Decide = %d, want %d: the default takes no word and needs no reseed", exitSeed, got, want)
	}
}

// removalConfig is exitConfig with a second region "yonder" (region 1; "home" is
// region 0) holding one miller agent of weight 2. Agent IDs: 0 and 1 the
// estates, 2 and 3 the millers of home (weights 2 and 1), 4 the weaver, 5 the
// miller of yonder, whose account is in region 1.
func removalConfig() Config {
	c := exitConfig()
	r := DefaultRegion()
	r.ID = "yonder"
	c.Regions = append(c.Regions, r)
	c.Groups = append(c.Groups, Group{Region: "yonder", Role: "miller", Households: 2, Scale: 2})
	return c
}

// TestRemoveAgent_MovesEverythingToTheEstateOfItsRegionAndEmitsInOrder verifies
// one removal of the miller of yonder (agent 5, weight 2) and of the weaver of
// home (agent 4, weight 1): the money and the lots go to the estate of the
// region the account is in and to no other, the account is closed, the slot is
// the zero agent (the ID is not reused), the counters fall, the events come in the
// order of spec-world.md 6.11 item 8 with the cause as given, and the ledger
// still audits.
//
// Arithmetic: agent 5 starts with 2 * 7 000 = 14 000 money, 2 * 10 = 20 grain and
// 2 * 1 = 2 pans, which are two lots (one per commodity key: GoodsListed counts
// lots, not units). The pool of yonder is 1 000 before, so 1 000 + 14 000 =
// 15 000 after and EstateOpened.Balance reads 15 000; the pool of home does not
// move. A job of 2 executions in progress is abandoned first, by agent_removed.
// The counters fall by 1 agent and by the weight 2 households, and by 1 in the
// (yonder, miller) cell.
//
// Requirement: FR-WLD-051, FR-WLD-056, FR-WLD-059, FR-AGT-001, FR-AGT-008, FR-AGT-077, FR-EST-002, FR-OBS-020
func TestRemoveAgent_MovesEverythingToTheEstateOfItsRegionAndEmitsInOrder(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		id      AgentID
		region  RegionID
		cause   Cause
		job     bool
		money   Money
		weight  int64
		wantLot int64
	}{
		{"starved with a job, region 1", 5, 1, CauseStarved, true, 14_000, 2, 2},
		{"insolvent without a job, region 1", 5, 1, CauseInsolvent, false, 14_000, 2, 2},
		{"insolvent weaver without goods, region 0", 4, 0, CauseInsolvent, false, 3_000, 1, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &evLog{}
			cap := newCapture()
			w := newMill(t, removalConfig(), WithSeed(exitSeed), WithObserver(rec), WithLogger(slog.New(cap)))
			if err := w.estates.seedPool(w.ledger, 1, 1_000); err != nil {
				t.Fatalf("seed %d: seedPool() error = %v, want nil", exitSeed, err)
			}
			if err := w.beginStep(); err != nil {
				t.Fatalf("seed %d: beginStep() error = %v, want nil", exitSeed, err)
			}
			rec.events = nil
			a := &w.agents[tt.id]
			if tt.job {
				a.job = job{recipe: 0, remaining: 1, executions: 2, principal: a.id}
			}
			acct := a.account
			role, weight := a.role, a.weight
			poolBefore := [2]Money{w.estates.pool(w.ledger, 0), w.estates.pool(w.ledger, 1)}
			unitsOf := func(acct int) (n int64) {
				lots, err := w.ledger.AppendLots(nil, acct)
				if err != nil {
					t.Fatalf("seed %d: AppendLots(%d) error = %v, want nil", exitSeed, acct, err)
				}
				for _, lt := range lots {
					n += lt.Quantity
				}
				return n
			}
			agentUnits := unitsOf(acct)
			estUnitsBefore := [2]int64{unitsOf(w.estates.account(0)), unitsOf(w.estates.account(1))}
			agentsBefore, hhBefore := w.live.agents, w.live.households
			cell := int(tt.region)*len(w.cfg.Roles) + int(role)
			inRoleBefore := w.live.inRole[cell]
			if weight != tt.weight {
				t.Fatalf("seed %d: agent %d has weight %d, want %d", exitSeed, tt.id, weight, tt.weight)
			}
			if err := w.removeAgent(a, tt.cause); err != nil {
				t.Fatalf("seed %d: removeAgent() error = %v, want nil", exitSeed, err)
			}
			wantPool := poolBefore
			wantPool[tt.region] += tt.money
			if got := [2]Money{w.estates.pool(w.ledger, 0), w.estates.pool(w.ledger, 1)}; got != wantPool {
				t.Errorf("seed %d: pools = %v, want %v: the money goes to the estate of the account's region only", exitSeed, got, wantPool)
			}
			if v, err := w.ledger.Account(acct); err != nil || !v.Closed {
				t.Errorf("seed %d: account %d = %+v, %v, want closed", exitSeed, acct, v, err)
			}
			if w.agents[tt.id].weight != 0 || w.live.agents != agentsBefore-1 || w.live.households != hhBefore-tt.weight || w.live.inRole[cell] != inRoleBefore-1 {
				t.Errorf("seed %d: slot weight %d, counters %d agents, %d households, %d in the cell, want 0, %d, %d, %d",
					exitSeed, w.agents[tt.id].weight, w.live.agents, w.live.households, w.live.inRole[cell], agentsBefore-1, hhBefore-tt.weight, inRoleBefore-1)
			}
			if got := flowOf(w, ledger.EstateMoney, ledger.Money); got != int64(tt.money) {
				t.Errorf("seed %d: estate_money flow = %d, want %d", exitSeed, got, tt.money)
			}
			// Close refuses a non-empty account (ErrNotEmpty), so a closed account
			// proves goods and money reached the estate before the close (3.6 order).
			wantUnits := estUnitsBefore
			wantUnits[tt.region] += agentUnits
			if got := [2]int64{unitsOf(w.estates.account(0)), unitsOf(w.estates.account(1))}; got != wantUnits {
				t.Errorf("seed %d: estate units = %v, want %v: the goods go to the estate of the account's region only", exitSeed, got, wantUnits)
			}
			var goodsFlow int64
			for _, f := range w.ledger.AppendFlows(nil, false) {
				if f.Reason == ledger.EstateGoods {
					goodsFlow += f.Total
				}
			}
			if goodsFlow != agentUnits {
				t.Errorf("seed %d: estate_goods flow = %d, want %d", exitSeed, goodsFlow, agentUnits)
			}
			if err := w.ledger.Audit(); err != nil {
				t.Errorf("seed %d: Audit() = %v, want nil: a removal creates and destroys nothing", exitSeed, err)
			}
			var want []Event
			if tt.job {
				want = append(want, RecipeAbandoned{Tick: 0, Agent: tt.id, Region: tt.region, Recipe: 0, Executions: 2, Reason: ReasonAgentRemoved})
			}
			want = append(want,
				AgentBankrupted{Tick: 0, Agent: tt.id, Region: tt.region, Role: role, Weight: tt.weight, Cause: tt.cause, Money: tt.money, GoodsListed: tt.wantLot},
				EstateOpened{Tick: 0, Agent: tt.id, Region: tt.region, Balance: wantPool[tt.region], GoodsListed: tt.wantLot})
			if !reflect.DeepEqual(rec.events, want) {
				t.Errorf("seed %d: events = %+v, want %+v", exitSeed, rec.events, want)
			}
			lines := cap.take("agent bankrupt")
			if len(lines) != 1 || lines[0].level != slog.LevelDebug || lines[0].attrs["agent"] != int64(tt.id) || lines[0].attrs["role"] != int64(role) {
				t.Errorf("seed %d: Debug lines %+v, want one \"agent bankrupt\" with agent %d and role %d", exitSeed, lines, tt.id, role)
			}
		})
	}
}

// TestRemoveAgent_WithoutObserverChangesTheSameState verifies that the removal
// does the same to the ledger and the counters with no observer and no logger
// supplied: events and lines are only a report.
//
// Requirement: FR-OBS-014
func TestRemoveAgent_WithoutObserverChangesTheSameState(t *testing.T) {
	t.Parallel()
	w := newMill(t, removalConfig(), WithSeed(exitSeed))
	if err := w.beginStep(); err != nil {
		t.Fatalf("seed %d: beginStep() error = %v, want nil", exitSeed, err)
	}
	if err := w.removeAgent(&w.agents[5], CauseStarved); err != nil {
		t.Fatalf("seed %d: removeAgent() error = %v, want nil", exitSeed, err)
	}
	if got := w.estates.pool(w.ledger, 1); got != 14_000 {
		t.Errorf("seed %d: pool of yonder = %d, want the agent's 2 * 7 000 = 14 000", exitSeed, got)
	}
	if err := w.endStep(nil); err != nil {
		t.Fatalf("seed %d: endStep() error = %v, want nil", exitSeed, err)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("seed %d: Audit() = %v, want nil", exitSeed, err)
	}
	step(t, w, 1)
}

// TestRelocationRefusal_HasNoRefusalInTheSlice verifies the empty composition: no
// module is on, so no agent is refused.
//
// Requirement: FR-WLD-040
func TestRelocationRefusal_HasNoRefusalInTheSlice(t *testing.T) {
	t.Parallel()
	w := newMill(t, removalConfig(), WithSeed(exitSeed))
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		if e, refused := w.relocationRefusal(&w.agents[i], nil); refused || e != nil {
			t.Errorf("seed %d: relocationRefusal(agent %d) = %v, %v, want no refusal", exitSeed, i, e, refused)
		}
	}
}
