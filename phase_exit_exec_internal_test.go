package economysim

import (
	"log/slog"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/occupation"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file tests the executor of the spawner's plan (P7.1(d), phase_exit.go)
// through a consumer Spawner fake that scripts the plan (D142): one case per kind
// and cause, the refusals, the pool, the budget order and the counts. Every case
// is seeded with execSeed, which the failure messages name (testing.md 6.2).

// execSeed is the seed of the executor tests.
const execSeed = 20261021

// scriptSpawner is a consumer Spawner that appends what plan returns for the
// view it is given, and lets a test look at the view first.
type scriptSpawner struct {
	plan   func(v *RegionView) []Decision
	onCall func(v *RegionView)
	calls  int
}

// Decide records the call, runs the hook and appends the scripted plan.
func (s *scriptSpawner) Decide(v *RegionView, _ *rand.Rand, d *Decisions) {
	s.calls++
	if s.onCall != nil {
		s.onCall(v)
	}
	if s.plan != nil {
		d.List = append(d.List, s.plan(v)...)
	}
}

// regionPlans scripts, for every tick, the plan of each region by its index.
func regionPlans(plans ...[]Decision) func(*RegionView) []Decision {
	return func(v *RegionView) []Decision {
		if int(v.Region) < len(plans) {
			return plans[v.Region]
		}
		return nil
	}
}

// tune applies to an exit configuration what the executor tests share: the whole
// pool as the entry budget (EntryPoolShare 10 000), a longer recipe so the miller's
// welfare window (2 cycles of 20 ticks = 40) is longer than the weaver's (30), a
// weaver whose target ratio differs (20 000, spread 2 000) and the scripted spawner.
func tune(c Config) Config {
	c.World.EntryPoolShare = 10_000
	c.World.Spawner = "script"
	c.Recipes[0].Duration = 20
	c.Roles[1].TargetRatio = 20_000
	c.Roles[0].TargetSpread, c.Roles[1].TargetSpread = 2_000, 2_000
	return c
}

// execConfig is exitConfig tuned: one region "home"; agent IDs 0 the estate, 1 and
// 2 the millers (weights 2 and 1, 7 000 start money, 10 grain and 1 pan per
// household), 3 the weaver (weight 1, 3 000, no goods).
func execConfig() Config { return tune(exitConfig()) }

// execConfig2 is removalConfig tuned: regions "home" (0) and "yonder" (1); IDs 0
// and 1 the estates, 2 and 3 the millers of home, 4 the weaver, 5 the miller of yonder.
func execConfig2() Config { return tune(removalConfig()) }

// execWorld builds the world of c with the scripted spawner and the pool of region
// 0 at pool, an observer and a capturing logger.
func execWorld(t *testing.T, c Config, pool Money, sp *scriptSpawner, opts ...Option) (*World, *evLog, capture) {
	t.Helper()
	rec, lg := &evLog{}, newCapture()
	opts = append([]Option{WithSeed(execSeed), WithSpawner("script", sp), WithObserver(rec), WithLogger(slog.New(lg))}, opts...)
	w := newMill(t, c, opts...)
	if pool > 0 {
		if err := w.estates.seedPool(w.ledger, 0, pool); err != nil {
			t.Fatalf("seed %d: seedPool() error = %v, want nil", execSeed, err)
		}
	}
	return w, rec, lg
}

// runEntry begins a step, forgets the events of its entry and runs P7.1 alone.
func runEntry(t *testing.T, w *World, rec *evLog) {
	t.Helper()
	if err := w.beginStep(); err != nil {
		t.Fatalf("seed %d: beginStep() error = %v, want nil", execSeed, err)
	}
	rec.events = nil
	if err := w.p71Entry(); err != nil {
		t.Fatalf("seed %d: p71Entry() error = %v, want nil", execSeed, err)
	}
}

// wantEvents compares the events a world emitted with the expected ones.
func wantEvents(t *testing.T, name string, rec *evLog, want ...Event) {
	t.Helper()
	if !reflect.DeepEqual(rec.events, want) {
		t.Errorf("%s (seed %d): events = %+v, want %+v", name, execSeed, rec.events, want)
	}
}

// poolOf returns the pool of region r.
func poolOf(w *World, r RegionID) Money { return w.estates.pool(w.ledger, r) }

// checkCounts rebuilds the agent count from AgentSpawned and AgentBankrupted and
// the household count also from WeightChanged (FR-SPN-034), starting from the counts
// before the events, and compares them with the world's counters and with the
// agent table.
func checkCounts(t *testing.T, name string, w *World, rec *evLog, agents0, hh0 int64) {
	t.Helper()
	agents, hh := agents0, hh0
	for _, e := range rec.events {
		switch e := e.(type) {
		case AgentSpawned:
			agents, hh = agents+1, hh+e.Weight
		case AgentBankrupted:
			agents, hh = agents-1, hh-e.Weight
		case WeightChanged:
			hh += e.To - e.From
		}
	}
	var tableAgents, tableHH int64
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 {
			tableAgents, tableHH = tableAgents+1, tableHH+a.weight
		}
	}
	if agents != w.live.agents || hh != w.live.households || tableAgents != agents || tableHH != hh {
		t.Errorf("%s (seed %d): rebuilt %d agents, %d households; counters %d, %d; table %d, %d; want all equal",
			name, execSeed, agents, hh, w.live.agents, w.live.households, tableAgents, tableHH)
	}
}

// auditOK fails the test when the ledger does not audit (FR-SPN-045).
func auditOK(t *testing.T, name string, w *World) {
	t.Helper()
	if err := w.ledger.Audit(); err != nil {
		t.Errorf("%s (seed %d): Audit() = %v, want nil: no decision creates or destroys anything", name, execSeed, err)
	}
}

// execLedgerState flattens the balance and the holdings of every account, and the
// flows of the tick, so two states can be compared by equality.
func execLedgerState(w *World) []int64 {
	var out []int64
	for i := range w.ledger.Len() {
		v, err := w.ledger.Account(i)
		if err != nil {
			out = append(out, -1)
			continue
		}
		out = append(out, v.Balance)
		for c := range w.cfg.Commodities {
			held, _, _ := w.ledger.Holding(i, c)
			out = append(out, held)
		}
	}
	for _, f := range w.ledger.AppendFlows(nil, false) {
		out = append(out, int64(f.Reason), int64(f.Commodity), f.Total)
	}
	return out
}

// tableState is the part of the agent table a decision may change, by slot.
func tableState(w *World) [][8]int64 {
	out := make([][8]int64, len(w.agents))
	for i := range w.agents {
		a := &w.agents[i]
		out[i] = [8]int64{int64(a.id), int64(a.account), int64(a.role), a.weight, int64(a.strategy), a.roleSince, a.target, a.job.executions}
	}
	return out
}

// withJob gives agent id a job of 2 executions of recipe 0 with the pending
// output 4, for the cases that need one in progress.
func withJob(w *World, id AgentID) {
	w.agents[id].job = job{recipe: 0, remaining: 1, executions: 2, principal: id, pending: []int64{4}}
}

// TestExecute_ReduceMovesTheDifferenceToTheEstate verifies a reduction of agent 1
// (weight 2, with a job in progress) by 1 household, for both causes the slice
// allows: the money and each good fall to floor(x * w' / w), the difference goes
// to the estate by the ledger reason starvation, the job is re-weighed, no account
// is opened or closed and WeightChanged carries the signed money and the units.
//
// Arithmetic: agent 1 holds 2 * 7 000 = 14 000 money, 2 * 10 = 20 grain and 2 pans;
// w' = 1, so it keeps floor(14 000 * 1 / 2) = 7 000, floor(20 / 2) = 10 grain and
// floor(2 / 2) = 1 pan, and 7 000 money, 10 grain and 1 pan (11 units) go to the
// estate: the pool is 25 000 + 7 000 = 32 000 and WeightChanged is From 2, To 1,
// Money -7 000, Units 11. The job of 2 executions and pending output 4 becomes 1
// execution and floor(4 * 1 / 2) = 2. The households fall from 4 to 3.
//
// Requirement: FR-SPN-004, FR-SPN-005, FR-SPN-044, FR-SPN-045, FR-AGT-008, FR-OBS-019, FR-LED-031
func TestExecute_ReduceMovesTheDifferenceToTheEstate(t *testing.T) {
	t.Parallel()
	for _, cause := range []Cause{CauseStarved, CauseInsolvent} {
		t.Run(string(cause), func(t *testing.T) {
			t.Parallel()
			sp := &scriptSpawner{plan: regionPlans([]Decision{{Kind: DecideReduce, Agent: 1, Households: 1, Cause: cause}})}
			w, rec, _ := execWorld(t, execConfig(), 25_000, sp)
			withJob(w, 1)
			accounts, agents0, hh0 := w.ledger.Len(), w.live.agents, w.live.households
			runEntry(t, w, rec)
			wantEvents(t, "reduce "+string(cause), rec,
				WeightChanged{Tick: 0, Agent: 1, Region: 0, Role: 0, From: 2, To: 1, Cause: cause, Money: -7_000, Units: 11},
				EntryLimited{Tick: 0, Region: 0, Removed: 1})
			if m, _ := balance(t, w, 1); m != 7_000 || held(t, w, 1, grain) != 10 || held(t, w, 1, 2) != 1 {
				t.Errorf("seed %d: agent 1 holds %d money, %d grain, %d pans, want 7000, 10 and 1", execSeed, m, held(t, w, 1, grain), held(t, w, 1, 2))
			}
			if got := poolOf(w, 0); got != 32_000 {
				t.Errorf("seed %d: pool = %d, want 25 000 + 7 000", execSeed, got)
			}
			if f, g, p := flowOf(w, ledger.Starvation, ledger.Money), flowOf(w, ledger.Starvation, grain), flowOf(w, ledger.Starvation, 2); f != 7_000 || g != 10 || p != 1 {
				t.Errorf("seed %d: starvation flows money %d, grain %d, pans %d, want 7000, 10, 1", execSeed, f, g, p)
			}
			if j := &w.agents[1].job; j.executions != 1 || !slices.Equal(j.pending, []int64{2}) || w.agents[1].weight != 1 {
				t.Errorf("seed %d: job executions %d pending %v weight %d, want 1, [2], 1", execSeed, j.executions, j.pending, w.agents[1].weight)
			}
			if got := w.ledger.Len(); got != accounts {
				t.Errorf("seed %d: accounts = %d, want %d: a reduction opens and closes none (FR-SPN-044)", execSeed, got, accounts)
			}
			checkCounts(t, "reduce "+string(cause), w, rec, agents0, hh0)
			auditOK(t, "reduce "+string(cause), w)
		})
	}
}

// TestExecute_WorkedReductionFloorsWhatTheAgentKeeps is the worked example of
// spec-spawner.md 4.1: an agent of weight 4 with 10 money and 6 units of one good,
// reduced to 3, keeps floor(10 * 3 / 4) = 7 money and floor(6 * 3 / 4) = 4 units,
// so 3 money and 2 units go to the estate, and WeightChanged has From 4, To 3,
// Money -3, Units 2.
//
// Arithmetic: the agent starts with 4 * 10 = 40 money, 4 * 2 = 8 grain and 4
// pans; 30 money, 2 grain and the 4 pans are moved to the pool first (reason
// starvation, which the ledger allows both ways), leaving 10, 6 and 0. The pool is
// then 25 000 + 30 and ends at 25 030 + 3 = 25 033.
//
// Requirement: FR-SPN-005, FR-OBS-019, FR-LED-035
func TestExecute_WorkedReductionFloorsWhatTheAgentKeeps(t *testing.T) {
	t.Parallel()
	c := execConfig()
	c.Groups[0] = Group{Region: "home", Role: "miller", Households: 4, Scale: 4}
	c.Roles[0].StartMoney = 10
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 2}, {Commodity: "pan", Quantity: 1}}
	sp := &scriptSpawner{plan: regionPlans([]Decision{{Kind: DecideReduce, Agent: 1, Households: 1, Cause: CauseStarved}})}
	w, rec, _ := execWorld(t, c, 25_000, sp)
	a, est := &w.agents[1], w.estates.account(0)
	if err := w.ledger.Transfer(a.account, est, 30, ledger.Starvation); err != nil {
		t.Fatalf("seed %d: Transfer() error = %v, want nil", execSeed, err)
	}
	for _, mv := range []struct{ commodity, qty int }{{grain, 2}, {2, 4}} {
		if err := w.ledger.Move(a.account, est, mv.commodity, int64(mv.qty), ledger.Starvation); err != nil {
			t.Fatalf("seed %d: Move() error = %v, want nil", execSeed, err)
		}
	}
	runEntry(t, w, rec)
	wantEvents(t, "worked reduction", rec,
		WeightChanged{Tick: 0, Agent: 1, Region: 0, Role: 0, From: 4, To: 3, Cause: CauseStarved, Money: -3, Units: 2},
		EntryLimited{Tick: 0, Region: 0, Removed: 1})
	if m, _ := balance(t, w, 1); m != 7 || held(t, w, 1, grain) != 4 || poolOf(w, 0) != 25_033 {
		t.Errorf("seed %d: agent holds %d money and %d grain, pool %d, want 7, 4 and 25 033", execSeed, m, held(t, w, 1, grain), poolOf(w, 0))
	}
	auditOK(t, "worked reduction", w)
}

// TestExecute_RemoveAndReductionToTheWholeWeight verifies that a removal, and a
// reduction by the whole weight, take the one removal path: the events are
// RecipeAbandoned (agent_removed, if a job), AgentBankrupted with the cause of the
// decision as given (the executor never recomputes it) and EstateOpened, never
// WeightChanged, and the account is closed. A removal with the cause consumer is
// valid.
//
// Arithmetic: agent 1 (weight 2) holds 14 000 money and the lots of 20 grain and 2
// pans (2 lots, whatever the units), so the pool is 25 000 + 14 000 = 39 000; agent
// 3 (the weaver, weight 1) holds 3 000 and no goods, so the pool is 28 000. The
// EntryLimited of the tick counts the households removed: 2 and 1.
//
// Requirement: FR-SPN-005, FR-SPN-015, FR-SPN-034, FR-WLD-051, FR-WLD-056, FR-AGT-001, FR-OBS-020, FR-EST-002
func TestExecute_RemoveAndReductionToTheWholeWeight(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		d       Decision
		job     bool
		money   Money
		weight  int64
		lots    int64
		balance Money
	}{
		{"remove, starved", Decision{Kind: DecideRemove, Agent: 1, Cause: CauseStarved}, true, 14_000, 2, 2, 39_000},
		{"reduce by the whole weight, starved", Decision{Kind: DecideReduce, Agent: 1, Households: 2, Cause: CauseStarved}, true, 14_000, 2, 2, 39_000},
		{"reduce by the whole weight, insolvent", Decision{Kind: DecideReduce, Agent: 1, Households: 2, Cause: CauseInsolvent}, false, 14_000, 2, 2, 39_000},
		{"remove the weaver, insolvent", Decision{Kind: DecideRemove, Agent: 3, Cause: CauseInsolvent}, false, 3_000, 1, 0, 28_000},
		{"remove with the cause consumer", Decision{Kind: DecideRemove, Agent: 3, Cause: CauseConsumer}, false, 3_000, 1, 0, 28_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec, _ := execWorld(t, execConfig(), 25_000, &scriptSpawner{plan: regionPlans([]Decision{tt.d})})
			if tt.job {
				withJob(w, 1)
			}
			acct, role := w.agents[tt.d.Agent].account, w.agents[tt.d.Agent].role
			agents0, hh0 := w.live.agents, w.live.households
			runEntry(t, w, rec)
			var want []Event
			if tt.job {
				want = append(want, RecipeAbandoned{Tick: 0, Agent: 1, Region: 0, Recipe: 0, Executions: 2, Reason: ReasonAgentRemoved})
			}
			want = append(want,
				AgentBankrupted{Tick: 0, Agent: tt.d.Agent, Region: 0, Role: role, Weight: tt.weight, Cause: tt.d.Cause, Money: tt.money, GoodsListed: tt.lots},
				EstateOpened{Tick: 0, Agent: tt.d.Agent, Region: 0, Balance: tt.balance, GoodsListed: tt.lots},
				EntryLimited{Tick: 0, Region: 0, Removed: tt.weight})
			wantEvents(t, tt.name, rec, want...)
			if v, err := w.ledger.Account(acct); err != nil || !v.Closed {
				t.Errorf("seed %d: account %d = %+v, %v, want closed", execSeed, acct, v, err)
			}
			if got := poolOf(w, 0); got != tt.balance || w.agents[tt.d.Agent].weight != 0 {
				t.Errorf("seed %d: pool %d, slot weight %d, want %d and 0", execSeed, got, w.agents[tt.d.Agent].weight, tt.balance)
			}
			checkCounts(t, tt.name, w, rec, agents0, hh0)
			auditOK(t, tt.name, w)
		})
	}
}

// TestExecute_SwitchKeepsTheAccountAndMovesTheGoodsOutsideTheNewSet verifies a role
// change of agent 1 from miller to weaver (with a job to abandon, goods outside the
// new tradable set and a longer welfare window) and of agent 3 from weaver to
// miller (new beliefs, a residue appears): the account and the money stay, the
// lots outside the new set go whole to the estate and are counted in lots, the job
// is abandoned with role_changed before RoleChanged, the history is resized, the
// residues reset, the target carried by its offset, the beliefs kept or reseeded
// from the window mean or the price, the strategy index switched and RoleSince the
// tick.
//
// Arithmetic: the weaver trades only grain, the miller flour, grain and pan. Agent
// 1 holds 20 grain and 2 pans: the pans (1 lot) go to the estate, the grain stays,
// so GoodsListed is 1 and the pool does not move (25 000). Its 14 000 money stay.
// The target moves with its offset: the weaver's range is 20 000 * (10 000 -+
// 2 000) / 10 000 = [16 000, 24 000], and 20 000 + (15 500 - 15 000) = 20 500 is
// inside. Agent 3 (target 24 000, offset +4 000 from the weaver's 20 000) goes to
// the miller's range [12 000, 18 000]: 15 000 + 4 000 = 19 000 is clamped to
// 18 000. Its flour belief starts at the mean of its window, 2 800, its pan belief
// at the published price 5 000 (the window is empty) and its grain belief is kept.
//
// Requirement: FR-SPN-006, FR-SPN-033, FR-SPN-044, FR-SPN-048, FR-AGT-008, FR-AGT-077, FR-OBS-019
func TestExecute_SwitchKeepsTheAccountAndMovesTheGoodsOutsideTheNewSet(t *testing.T) {
	t.Parallel()
	t.Run("miller to weaver", func(t *testing.T) {
		t.Parallel()
		sp := &scriptSpawner{plan: regionPlans([]Decision{{Kind: DecideSwitch, Agent: 1, Role: 1, Cause: CauseDistress}})}
		w, rec, _ := execWorld(t, execConfig(), 25_000, sp)
		withJob(w, 1)
		a := &w.agents[1]
		a.target, a.roleSince, a.residues[0] = 15_500, -5, 77
		accounts, agents0, hh0, grainBefore := w.ledger.Len(), w.live.agents, w.live.households, held(t, w, 1, grain)
		if a.history.Window() != 40 {
			t.Fatalf("seed %d: miller window = %d, want 2 cycles of 20 ticks = 40", execSeed, a.history.Window())
		}
		runEntry(t, w, rec)
		wantEvents(t, "switch miller to weaver", rec,
			RecipeAbandoned{Tick: 0, Agent: 1, Region: 0, Recipe: 0, Executions: 2, Reason: ReasonRoleChanged},
			RoleChanged{Tick: 0, Agent: 1, Region: 0, From: 0, To: 1, Weight: 2, Cause: CauseDistress, GoodsListed: 1},
			EntryLimited{Tick: 0, Region: 0, Changed: 2})
		if m, _ := balance(t, w, 1); m != 14_000 || held(t, w, 1, grain) != grainBefore || held(t, w, 1, 2) != 0 || poolOf(w, 0) != 25_000 {
			t.Errorf("seed %d: agent holds %d money, %d grain, %d pans, pool %d, want 14 000, %d, 0, 25 000", execSeed, m, held(t, w, 1, grain), held(t, w, 1, 2), poolOf(w, 0), grainBefore)
		}
		if flowOf(w, ledger.EstateGoods, 2) != 2 || flowOf(w, ledger.EstateGoods, grain) != 0 {
			t.Errorf("seed %d: estate_goods flow of pans %d, of grain %d, want 2 and 0", execSeed, flowOf(w, ledger.EstateGoods, 2), flowOf(w, ledger.EstateGoods, grain))
		}
		if a.hasJob() || a.role != 1 || a.strategy != 1 || a.roleSince != 0 || a.target != 20_500 || a.weight != 2 {
			t.Errorf("seed %d: job %v role %d strategy %d roleSince %d target %d weight %d, want no job, 1, 1, 0, 20500, 2", execSeed, a.hasJob(), a.role, a.strategy, a.roleSince, a.target, a.weight)
		}
		if a.history.Window() != 30 || len(a.residues) != 0 || len(a.beliefs) != 1 {
			t.Errorf("seed %d: history window %d, residues %d, beliefs %d, want 30, 0, 1", execSeed, a.history.Window(), len(a.residues), len(a.beliefs))
		}
		cell := func(r RoleID) int64 { return w.live.inRole[int(r)] }
		if cell(0) != 1 || cell(1) != 2 || w.ledger.Len() != accounts {
			t.Errorf("seed %d: agents in miller %d, weaver %d, accounts %d, want 1, 2, %d", execSeed, cell(0), cell(1), w.ledger.Len(), accounts)
		}
		checkCounts(t, "switch miller to weaver", w, rec, agents0, hh0)
		auditOK(t, "switch miller to weaver", w)
	})
	t.Run("weaver to miller", func(t *testing.T) {
		t.Parallel()
		sp := &scriptSpawner{plan: regionPlans([]Decision{{Kind: DecideSwitch, Agent: 3, Role: 0, Cause: CauseReview}})}
		w, rec, _ := execWorld(t, execConfig(), 25_000, sp)
		a := &w.agents[3]
		a.target = 24_000
		w.markets.windows[w.markets.index(0, flour)].Push(2_800)
		kept := a.beliefs[0]
		runEntry(t, w, rec)
		wantEvents(t, "switch weaver to miller", rec,
			RoleChanged{Tick: 0, Agent: 3, Region: 0, From: 1, To: 0, Weight: 1, Cause: CauseReview},
			EntryLimited{Tick: 0, Region: 0, Changed: 1})
		if a.target != 18_000 || len(a.residues) != 1 || a.residues[0] != 0 || a.strategy != 0 || a.history.Window() != 40 {
			t.Errorf("seed %d: target %d residues %v strategy %d window %d, want 18000, [0], 0, 40", execSeed, a.target, a.residues, a.strategy, a.history.Window())
		}
		if len(a.beliefs) != 3 || a.beliefs[0].Centre != 2_800 || a.beliefs[1] != kept || a.beliefs[2].Centre != 5_000 || a.beliefs[0].Width != w.bp.WidthInit || a.beliefs[2].Width != w.bp.WidthInit {
			t.Errorf("seed %d: beliefs %+v, want [flour 2800, grain %+v kept, pan 5000] with the initial width %d", execSeed, a.beliefs, kept, w.bp.WidthInit)
		}
		if got := poolOf(w, 0); got != 25_000 {
			t.Errorf("seed %d: pool = %d, want 25 000: a switch without goods moves no money", execSeed, got)
		}
		auditOK(t, "switch weaver to miller", w)
	})
}

// enterPlan is the plan of n households of the miller role, caused by renewal.
func enterPlan(n int64) Decision {
	return Decision{Kind: DecideEnter, Role: 0, Households: n, Cause: CauseRenewal}
}

// TestExecute_EnterIntoAnExistingAgentAndIntoANewAgent verifies the placement and
// the funding of entries: into the agent of the role with the smallest weight
// below the group's Scale (reason births, WeightChanged after the transfer, no
// account opened), else a new agent of weight 1 (reason start_money, an account,
// the two creation draws of D10, beliefs from the price, no goods, AgentSpawned
// with the money).
//
// Arithmetic: the miller group has Scale 2 and agents 2 and 3 of weights 2 and 1; a
// household brings 7 000. One household goes to agent 3 (weight 1 < 2, room 1):
// births 7 000, WeightChanged 1 -> 2, pool 25 000 - 7 000 = 18 000. Three households:
// the first fills agent 3 (births 7 000), the second finds no host below 2 and is a
// new agent 4 (start_money 7 000), the third joins agent 4 (births 7 000): the pool
// is 25 000 - 21 000 = 4 000, flows births 14 000 and start_money 7 000, the
// households rise from 4 to 7 and the agents from 3 to 4.
//
// Requirement: FR-SPN-007, FR-SPN-008, FR-SPN-027, FR-SPN-034, FR-SPN-044, FR-SPN-045, FR-AGT-005, FR-AGT-007, FR-LED-036
func TestExecute_EnterIntoAnExistingAgentAndIntoANewAgent(t *testing.T) {
	t.Parallel()
	t.Run("into an existing agent", func(t *testing.T) {
		t.Parallel()
		w, rec, _ := execWorld(t, execConfig(), 25_000, &scriptSpawner{plan: regionPlans([]Decision{enterPlan(1)})})
		accounts, agents0, hh0 := w.ledger.Len(), w.live.agents, w.live.households
		runEntry(t, w, rec)
		wantEvents(t, "enter 1", rec,
			WeightChanged{Tick: 0, Agent: 2, Region: 0, Role: 0, From: 1, To: 2, Cause: CauseRenewal, Money: 7_000},
			EntryLimited{Tick: 0, Region: 0, Entered: 1})
		if m, _ := balance(t, w, 2); m != 14_000 || poolOf(w, 0) != 18_000 || flowOf(w, ledger.Births, ledger.Money) != 7_000 || flowOf(w, ledger.StartMoney, ledger.Money) != 0 {
			t.Errorf("seed %d: agent 2 holds %d, pool %d, births %d, start_money %d, want 14 000, 18 000, 7 000, 0",
				execSeed, m, poolOf(w, 0), flowOf(w, ledger.Births, ledger.Money), flowOf(w, ledger.StartMoney, ledger.Money))
		}
		if w.ledger.Len() != accounts || held(t, w, 2, grain) != 10 {
			t.Errorf("seed %d: accounts %d, agent 2 grain %d, want %d and 10: no account and no goods", execSeed, w.ledger.Len(), held(t, w, 2, grain), accounts)
		}
		checkCounts(t, "enter 1", w, rec, agents0, hh0)
		auditOK(t, "enter 1", w)
	})
	t.Run("into a new agent", func(t *testing.T) {
		t.Parallel()
		w, rec, _ := execWorld(t, execConfig(), 25_000, &scriptSpawner{plan: regionPlans([]Decision{enterPlan(3)})})
		accounts, agents0, hh0 := w.ledger.Len(), w.live.agents, w.live.households
		runEntry(t, w, rec)
		wantEvents(t, "enter 3", rec,
			WeightChanged{Tick: 0, Agent: 2, Region: 0, Role: 0, From: 1, To: 2, Cause: CauseRenewal, Money: 7_000},
			AgentSpawned{Tick: 0, Agent: 4, Region: 0, Role: 0, Weight: 1, StartMoney: 7_000},
			WeightChanged{Tick: 0, Agent: 4, Region: 0, Role: 0, From: 1, To: 2, Cause: CauseRenewal, Money: 7_000},
			EntryLimited{Tick: 0, Region: 0, Entered: 3})
		if poolOf(w, 0) != 4_000 || flowOf(w, ledger.Births, ledger.Money) != 14_000 || flowOf(w, ledger.StartMoney, ledger.Money) != 7_000 {
			t.Errorf("seed %d: pool %d, births %d, start_money %d, want 4 000, 14 000, 7 000", execSeed, poolOf(w, 0), flowOf(w, ledger.Births, ledger.Money), flowOf(w, ledger.StartMoney, ledger.Money))
		}
		if w.ledger.Len() != accounts+1 || w.live.agents != agents0+1 || w.live.households != hh0+3 || len(w.agents) != 5 {
			t.Errorf("seed %d: accounts %d, agents %d, households %d, slots %d, want %d, %d, %d, 5", execSeed, w.ledger.Len(), w.live.agents, w.live.households, len(w.agents), accounts+1, agents0+1, hh0+3)
		}
		ref := newStream(execSeed)
		ref.keyed(siteCreation, 0, 4)
		role := &w.cfg.Roles[0]
		wantTarget := newTarget(ref, role.TargetRatio, role.TargetSpread)
		wantResidue := ref.draw(residueBound)
		n := &w.agents[4]
		if n.target != wantTarget || len(n.residues) != 1 || n.residues[0] != wantResidue || n.roleSince != 0 || n.hunger != 0 || n.privation != 0 || n.strategy != 0 {
			t.Errorf("seed %d: new agent target %d residues %v roleSince %d hunger %d privation %d strategy %d, want the D10 draws %d and [%d], 0, 0, 0, 0",
				execSeed, n.target, n.residues, n.roleSince, n.hunger, n.privation, n.strategy, wantTarget, wantResidue)
		}
		if held(t, w, 4, grain) != 0 || len(n.beliefs) != 3 || n.beliefs[0].Centre != 3_000 || n.beliefs[1].Centre != 1_000 || n.beliefs[2].Centre != 5_000 {
			t.Errorf("seed %d: new agent grain %d beliefs %+v, want no goods and the published prices 3000, 1000, 5000", execSeed, held(t, w, 4, grain), n.beliefs)
		}
		checkCounts(t, "enter 3", w, rec, agents0, hh0)
		auditOK(t, "enter 3", w)
	})
}

// TestExecute_EntryNeverOverdrawsThePool verifies the funding of an Enter
// (FR-SPN-008, FR-EST-014, FR-EST-022): the funded number is min(n, floor(budget /
// StartMoney)), the rest are counted Unfunded and neither created nor deferred,
// two entries of a plan share the one budget, and the pool never goes negative.
//
// Arithmetic: a household brings 7 000. A pool of 17 500 (2.5 households) funds
// floor(17 500 / 7 000) = 2 of 3 households and ends at 17 500 - 14 000 = 3 500;
// a pool of 0 funds none; a pool of 25 000 funds 3 households (floor(25 000 /
// 7 000)) in two entries of 2: the first is funded in full (14 000, budget 11 000
// left) and the second gets floor(11 000 / 7 000) = 1 (one unfunded), so the pool
// ends at 25 000 - 21 000 = 4 000.
//
// Requirement: FR-SPN-008, FR-SPN-045, FR-EST-014, FR-EST-022
func TestExecute_EntryNeverOverdrawsThePool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		pool         Money
		plan         []Decision
		entered      int64
		unfunded     int64
		wantPool     Money
		wantNewAgent bool
	}{
		{"pool of 2.5 households", 17_500, []Decision{enterPlan(3)}, 2, 1, 3_500, true},
		{"pool of 0", 0, []Decision{enterPlan(3)}, 0, 3, 0, false},
		{"two entries share one budget", 25_000, []Decision{enterPlan(2), enterPlan(2)}, 3, 1, 4_000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec, lg := execWorld(t, execConfig(), tt.pool, &scriptSpawner{plan: regionPlans(tt.plan)})
			agents0, hh0 := w.live.agents, w.live.households
			runEntry(t, w, rec)
			var lim []EntryLimited
			for _, e := range rec.events {
				if l, ok := e.(EntryLimited); ok {
					lim = append(lim, l)
				}
			}
			if want := []EntryLimited{{Tick: 0, Region: 0, Entered: tt.entered, Unfunded: tt.unfunded}}; !reflect.DeepEqual(lim, want) {
				t.Errorf("%s (seed %d): EntryLimited = %+v, want %+v", tt.name, execSeed, lim, want)
			}
			if got := poolOf(w, 0); got != tt.wantPool || got < 0 {
				t.Errorf("%s (seed %d): pool = %d, want %d and never negative", tt.name, execSeed, got, tt.wantPool)
			}
			if got := len(of[AgentSpawned](rec)) > 0; got != tt.wantNewAgent {
				t.Errorf("%s (seed %d): a new agent was created = %v, want %v", tt.name, execSeed, got, tt.wantNewAgent)
			}
			if lines := lg.take("entry dropped"); len(lines) != 1 || lines[0].level != slog.LevelDebug || lines[0].strs["reason"] != string(ReasonUnfunded) || lines[0].attrs["role"] != 0 {
				t.Errorf("%s (seed %d): Debug lines %+v, want one \"entry dropped\" for role 0 with the reason unfunded (the one entry that was short)", tt.name, execSeed, lines)
			}
			checkCounts(t, tt.name, w, rec, agents0, hh0)
			auditOK(t, tt.name, w)
		})
	}
}

// TestExecute_UnfundedHouseholdsAreNotDeferred verifies that the households the
// pool could not fund are not created later: the plan of the next tick does not
// see them (it is empty here), and nothing enters then.
//
// Arithmetic: a pool of 17 500 funds 2 of 3 households at tick 0, and the tick 1
// plan is empty, so the households stay 4 + 2 = 6 and no AgentSpawned,
// WeightChanged or EntryLimited carries tick 1.
//
// Requirement: FR-SPN-008
func TestExecute_UnfundedHouseholdsAreNotDeferred(t *testing.T) {
	t.Parallel()
	sp := &scriptSpawner{plan: func(v *RegionView) []Decision {
		if v.Tick == 0 {
			return []Decision{enterPlan(3)}
		}
		return nil
	}}
	w, rec, _ := execWorld(t, execConfig(), 17_500, sp)
	step(t, w, 2)
	for _, e := range rec.events {
		switch e := e.(type) {
		case AgentSpawned:
			if e.Tick != 0 {
				t.Errorf("seed %d: %+v at tick %d, want nothing entering after tick 0", execSeed, e, e.Tick)
			}
		case WeightChanged:
			if e.Tick != 0 {
				t.Errorf("seed %d: %+v at tick %d, want nothing entering after tick 0", execSeed, e, e.Tick)
			}
		case EntryLimited:
			if e.Tick != 0 {
				t.Errorf("seed %d: %+v at tick %d, want no EntryLimited when nothing acted", execSeed, e, e.Tick)
			}
		}
	}
	if w.live.households != 6 {
		t.Errorf("seed %d: households = %d, want 4 + 2 funded", execSeed, w.live.households)
	}
}

// TestExecute_UnfundedWarnIsBounded verifies the Warn "entry unfunded" (pattern B,
// FR-SPN-036): a region whose pool funds no entrant for ten consecutive ticks while
// entrants are due warns at the tenth tick, with the region and the households
// unfunded, and not before.
//
// Arithmetic: a pool of 0 and 3 households due every tick: the run is t + 1 at
// tick t and reaches 10 at tick 9, so one Warn in 10 ticks, households 3.
//
// Requirement: FR-SPN-036, FR-OBS-070
func TestExecute_UnfundedWarnIsBounded(t *testing.T) {
	t.Parallel()
	w, _, lg := execWorld(t, execConfig(), 0, &scriptSpawner{plan: regionPlans([]Decision{enterPlan(3)})})
	var ticks []int64
	for range 10 {
		step(t, w, 1)
		for _, l := range lg.take("entry unfunded") {
			if l.level != slog.LevelWarn || l.attrs["region"] != 0 || l.attrs["households"] != 3 {
				t.Errorf("seed %d: Warn at tick %d = %+v, want Warn region 0 households 3", execSeed, w.Tick()-1, l)
			}
			ticks = append(ticks, w.Tick()-1)
		}
	}
	if !slices.Equal(ticks, []int64{9}) {
		t.Errorf("seed %d: ticks of the Warn \"entry unfunded\" = %v, want [9]", execSeed, ticks)
	}
}

// TestExecute_TheBudgetIsTakenOnceAfterTheDeaths verifies the order of the entry
// budget (FR-SPN-026, FR-WLD-056, ruling C15): the budget PoolMoney * EntryPoolShare
// / 10 000 is taken at the first switch or entry of the plan, so the estates of the
// removals before it fund the births of the same tick, and it is taken once.
//
// Arithmetic (EntryPoolShare 5 000, a household 7 000, the pool 25 000): removal
// first: agent 1 adds 14 000, the pool is 39 000, the budget 19 500 and 3 households
// fund floor(19 500 / 7 000) = 2 (1 unfunded), the pool ending at 39 000 - 14 000 =
// 25 000. Entry first: the budget is 12 500 and funds 1 (2 unfunded). Taken once:
// remove 1 (39 000, budget 19 500), enter 1 (7 000, 12 500 left, the pool 32 000),
// remove the weaver (+3 000, the pool 35 000), enter 3: the budget is still 12 500
// and funds 1 (a budget retaken would be 17 500 and fund 2), so 2 are unfunded.
//
// Requirement: FR-SPN-026, FR-SPN-008, FR-WLD-056
func TestExecute_TheBudgetIsTakenOnceAfterTheDeaths(t *testing.T) {
	t.Parallel()
	rm := func(id AgentID) Decision { return Decision{Kind: DecideRemove, Agent: id, Cause: CauseStarved} }
	tests := []struct {
		name     string
		plan     []Decision
		entered  int64
		unfunded int64
	}{
		{"deaths then births", []Decision{rm(1), enterPlan(3)}, 2, 1},
		{"births then deaths", []Decision{enterPlan(3), rm(1)}, 1, 2},
		{"taken once", []Decision{rm(1), enterPlan(1), rm(3), enterPlan(3)}, 2, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := execConfig()
			c.World.EntryPoolShare = 5_000
			w, rec, _ := execWorld(t, c, 25_000, &scriptSpawner{plan: regionPlans(tt.plan)})
			agents0, hh0 := w.live.agents, w.live.households
			runEntry(t, w, rec)
			lim := of[EntryLimited](rec)
			if len(lim) != 1 || lim[0].Entered != tt.entered || lim[0].Unfunded != tt.unfunded {
				t.Errorf("%s (seed %d): EntryLimited = %+v, want Entered %d, Unfunded %d", tt.name, execSeed, lim, tt.entered, tt.unfunded)
			}
			checkCounts(t, tt.name, w, rec, agents0, hh0)
			auditOK(t, tt.name, w)
		})
	}
}

// TestExecute_RefusalsChangeNothingAndAreCounted verifies, one case per reason the
// slice builds, that a refused decision changes nothing (ledger balances, holdings
// and flows, the account count and the agent table are equal before and after),
// is counted only in EntryLimited.Refused (the other counts do not move) and is
// logged at Debug with the agent and the reason (FR-SPN-004, FR-SPN-010, D196).
//
// Arithmetic: the world has two regions; agent 5 is the miller of yonder, agent 3
// the miller of home with weight 1, agent 2 the one with weight 2, agent 4 the
// weaver. Every plan is that one decision, so the EntryLimited of the tick is
// exactly Refused 1. A reduction of agent 3 (weight 1) by 1 would be a removal, and
// with the cause consumer it is refused before the removal path is reached.
//
// Requirement: FR-SPN-004, FR-SPN-010, FR-SPN-035, FR-SPN-045, FR-OBS-014
func TestExecute_RefusalsChangeNothingAndAreCounted(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		setup  func(w *World)
		d      Decision
		reason Reason
	}{
		{"unknown agent", nil, Decision{Kind: DecideReduce, Agent: 99, Households: 1, Cause: CauseStarved}, ReasonUnknownAgent},
		{"estate pseudo-agent", nil, Decision{Kind: DecideRemove, Agent: 0, Cause: CauseStarved}, ReasonUnknownAgent},
		{"negative agent", nil, Decision{Kind: DecideRemove, Agent: -1, Cause: CauseStarved}, ReasonUnknownAgent},
		{"agent of another region", nil, Decision{Kind: DecideRemove, Agent: 5, Cause: CauseStarved}, ReasonUnknownAgent},
		{"unknown role in a switch", nil, Decision{Kind: DecideSwitch, Agent: 3, Role: 9, Cause: CauseReview}, ReasonUnknownRole},
		{"unknown role in an entry", nil, Decision{Kind: DecideEnter, Role: 9, Households: 1, Cause: CauseRenewal}, ReasonUnknownRole},
		{"negative role in an entry", nil, Decision{Kind: DecideEnter, Role: -1, Households: 1, Cause: CauseRenewal}, ReasonUnknownRole},
		{"external agent that is not mortal", func(w *World) { w.agents[4].external = true }, Decision{Kind: DecideRemove, Agent: 4, Cause: CauseStarved}, ReasonExternalAgent},
		{"switch to the agent's own role", nil, Decision{Kind: DecideSwitch, Agent: 3, Role: 0, Cause: CauseReview}, ReasonSameRole},
		{"reduce by 0", nil, Decision{Kind: DecideReduce, Agent: 2, Households: 0, Cause: CauseStarved}, ReasonBadHouseholds},
		{"reduce by more than the weight", nil, Decision{Kind: DecideReduce, Agent: 3, Households: 2, Cause: CauseStarved}, ReasonBadHouseholds},
		{"enter 0 households", nil, Decision{Kind: DecideEnter, Role: 0, Households: 0, Cause: CauseRenewal}, ReasonBadHouseholds},
		{"enter a negative number", nil, Decision{Kind: DecideEnter, Role: 0, Households: -2, Cause: CauseRenewal}, ReasonBadHouseholds},
		{"reduce with the cause consumer", nil, Decision{Kind: DecideReduce, Agent: 2, Households: 1, Cause: CauseConsumer}, ReasonUnsupportedCause},
		{"reduce to the whole weight with the cause consumer", nil, Decision{Kind: DecideReduce, Agent: 3, Households: 1, Cause: CauseConsumer}, ReasonUnsupportedCause},
		{"a kind the slice does not know", nil, Decision{Kind: DecisionKind(9), Agent: 2}, reasonUnknownKind},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec, lg := execWorld(t, execConfig2(), 25_000, &scriptSpawner{plan: regionPlans([]Decision{tt.d})})
			if tt.setup != nil {
				tt.setup(w)
			}
			ledgerBefore, tableBefore, accounts := execLedgerState(w), tableState(w), w.ledger.Len()
			agents0, hh0 := w.live.agents, w.live.households
			runEntry(t, w, rec)
			wantEvents(t, tt.name, rec, EntryLimited{Tick: 0, Region: 0, Refused: 1})
			if !slices.Equal(ledgerBefore, execLedgerState(w)) || !reflect.DeepEqual(tableBefore, tableState(w)) || w.ledger.Len() != accounts {
				t.Errorf("%s (seed %d): the ledger or the agent table changed, want a refused decision to change nothing", tt.name, execSeed)
			}
			lines := lg.take("decision refused")
			if len(lines) != 1 || lines[0].level != slog.LevelDebug || lines[0].attrs["agent"] != int64(tt.d.Agent) || lines[0].strs["reason"] != string(tt.reason) {
				t.Errorf("%s (seed %d): Debug lines %+v, want one \"decision refused\" with agent %d and reason %s", tt.name, execSeed, lines, tt.d.Agent, tt.reason)
			}
			if got := refusalReasonOf(w, &tt.d); got != tt.reason {
				t.Errorf("%s (seed %d): refusal = %q, want %q", tt.name, execSeed, got, tt.reason)
			}
			checkCounts(t, tt.name, w, rec, agents0, hh0)
			auditOK(t, tt.name, w)
		})
	}
}

// refusalReasonOf is what the executor says of a decision in region 0 now.
func refusalReasonOf(w *World, d *Decision) Reason { return w.refusal(0, d) }

// TestExecute_ARefusedDecisionDoesNotStopThePlan verifies that the decisions
// around a refused one are executed, and as if it were not there: a plan of an
// entry, a refused removal of an unknown agent and a second entry gives the events
// and the ledger of the plan without the refusal, and EntryLimited counts the
// refusal. A mortal external agent is accepted where a non-mortal one is refused.
//
// Arithmetic: each entry of 1 household is funded (7 000 of 25 000): the first goes
// to agent 3 (weight 1 below Scale 2), the second finds no host and is a new agent
// 4 (start_money), so Entered is 2 and Refused 1.
//
// Requirement: FR-SPN-004, FR-SPN-010, FR-SPN-035
func TestExecute_ARefusedDecisionDoesNotStopThePlan(t *testing.T) {
	t.Parallel()
	bad := Decision{Kind: DecideRemove, Agent: 99, Cause: CauseStarved}
	run := func(plan ...Decision) (*World, *evLog) {
		w, rec, _ := execWorld(t, execConfig(), 25_000, &scriptSpawner{plan: regionPlans(plan)})
		runEntry(t, w, rec)
		return w, rec
	}
	with, recWith := run(enterPlan(1), bad, enterPlan(1))
	without, recWithout := run(enterPlan(1), enterPlan(1))
	wantEntry := func(refused int64) EntryLimited {
		return EntryLimited{Tick: 0, Region: 0, Entered: 2, Refused: refused}
	}
	if n := len(recWith.events); n == 0 || !reflect.DeepEqual(recWith.events[n-1], wantEntry(1)) || !reflect.DeepEqual(recWithout.events[len(recWithout.events)-1], wantEntry(0)) {
		t.Errorf("seed %d: EntryLimited with %+v, without %+v, want Entered 2 with Refused 1 and 0", execSeed, recWith.events, recWithout.events)
	}
	if !reflect.DeepEqual(recWith.events[:len(recWith.events)-1], recWithout.events[:len(recWithout.events)-1]) || !slices.Equal(execLedgerState(with), execLedgerState(without)) {
		t.Errorf("seed %d: events %+v and ledger differ from the plan without the refusal %+v", execSeed, recWith.events, recWithout.events)
	}
	for _, mortal := range []bool{false, true} {
		w, rec, _ := execWorld(t, execConfig(), 25_000, &scriptSpawner{plan: regionPlans([]Decision{{Kind: DecideRemove, Agent: 3, Cause: CauseStarved}})})
		w.agents[3].external, w.agents[3].mortal = true, mortal
		runEntry(t, w, rec)
		if got := len(of[AgentBankrupted](rec)) == 1; got != mortal {
			t.Errorf("seed %d: external agent, mortal %v: removed = %v, want %v", execSeed, mortal, got, mortal)
		}
	}
}

// TestExecute_ATickWithNothingEmitsNoEntryLimited verifies FR-SPN-035: a tick whose
// plan is empty emits no EntryLimited and no other event of the entry step.
//
// Requirement: FR-SPN-035
func TestExecute_ATickWithNothingEmitsNoEntryLimited(t *testing.T) {
	t.Parallel()
	w, rec, _ := execWorld(t, execConfig(), 25_000, &scriptSpawner{})
	runEntry(t, w, rec)
	wantEvents(t, "an empty plan", rec)
}

// TestExecute_DecisionKindsAreTheKindsOfTheRefusalRules verifies that the root's
// DecisionKind and internal/occupation's Kind have the same values, since the
// executor converts one to the other.
//
// Requirement: FR-SPN-004
func TestExecute_DecisionKindsAreTheKindsOfTheRefusalRules(t *testing.T) {
	t.Parallel()
	for _, p := range []struct {
		d DecisionKind
		k occupation.Kind
	}{{DecideRemove, occupation.KindRemove}, {DecideReduce, occupation.KindReduce}, {DecideSwitch, occupation.KindSwitch}, {DecideEnter, occupation.KindEnter}} {
		if int(p.d) != int(p.k) {
			t.Errorf("DecisionKind %d, occupation kind %d, want equal", p.d, p.k)
		}
	}
}

// guardSpawner is the default spawner with a recorder: what it saw of the carry
// of each role and what it planned.
type guardSpawner struct {
	carry                     []int64
	entries, noViable, capped int64
}

// Decide runs the default's Decide and records the carry the view held and the
// households it planned to enter, drop or cut.
func (g *guardSpawner) Decide(v *RegionView, rng *rand.Rand, d *Decisions) {
	DefaultSpawner{}.Decide(v, rng, d)
	g.carry = g.carry[:0]
	for i := range v.Roles {
		g.carry = append(g.carry, v.Roles[i].RenewalCarry)
	}
	g.entries, g.noViable, g.capped = 0, d.noViableRole, d.regionCap+d.roleCap
	for _, x := range d.List {
		if x.Kind == DecideEnter && x.Cause == CauseRenewal {
			g.entries += x.Households
		}
	}
}

// TestGuardN265_PlannedRenewalEqualsTheRuleOnTheHouseholdsRemoved is the Guard of
// N-265 (spec-spawner.md 7.1): over a seed range, with the default spawner and a
// positive Renewal, the households Decide plans to enter (or drops or cuts by a
// cap) in a region equal the sum over roles of occupation.Renewal(carry before,
// the households the executor actually removed, Renewal), the carry the executor
// stores is that rule's carry, and a refusal of a default decision fails it.
//
// Arithmetic (Renewal 5 000, so a carry of c and r removed give floor((c + r *
// 5 000) / 10 000) entries and the remainder as the new carry): carry 0 and 3
// removed give 1 entry and carry 5 000; carry 5 000 and 1 removed give 1 and carry
// 0. The agents' privation, unfed and money are drawn from the seed, so removals,
// reductions of both causes and the carries vary over the range.
//
// Requirement: FR-SPN-004, FR-SPN-005, FR-SPN-015, FR-SPN-043
//
// Guard: the default's plan and the executor apply one rule twice (7.1), and only
// this test shows them drifting apart.
func TestGuardN265_PlannedRenewalEqualsTheRuleOnTheHouseholdsRemoved(t *testing.T) {
	t.Parallel()
	var removedTotal, plannedTotal int64
	for seed := uint64(1); seed <= 40; seed++ {
		c := exitConfig()
		c.World.Renewal, c.World.ExitShare, c.World.EntryPoolShare = 5_000, 10_000, 10_000
		c.Groups[0].Households = 8
		c.Groups[0].Scale = 2
		g := &guardSpawner{}
		c.World.Spawner = "guard"
		rec := &evLog{}
		w := newMill(t, c, WithSeed(seed), WithSpawner("guard", g), WithObserver(rec))
		if err := w.estates.seedPool(w.ledger, 0, 1_000_000); err != nil {
			t.Fatalf("seed %d: seedPool() error = %v, want nil", seed, err)
		}
		if err := w.beginStep(); err != nil {
			t.Fatalf("seed %d: beginStep() error = %v, want nil", seed, err)
		}
		rng := rand.New(rand.NewPCG(seed, 265))
		for round := range 4 {
			scriptPrivation(t, w, rng, seed)
			carries := slices.Clone(w.state.renewalCarry)
			rec.events = nil
			if err := w.p71Entry(); err != nil {
				t.Fatalf("seed %d round %d: p71Entry() error = %v, want nil", seed, round, err)
			}
			removed := make([]int64, len(c.Roles))
			for _, e := range rec.events {
				switch e := e.(type) {
				case AgentBankrupted:
					removed[e.Role] += e.Weight
				case WeightChanged:
					if e.Cause == CauseStarved || e.Cause == CauseInsolvent {
						removed[e.Role] += e.From - e.To
					}
				case EntryLimited:
					if e.Refused != 0 {
						t.Errorf("seed %d round %d: EntryLimited.Refused = %d, want 0: a refusal of a default decision is a bug", seed, round, e.Refused)
					}
				}
			}
			var want int64
			for r := range removed {
				n, carry, ok := occupation.Renewal(carries[r], removed[r], c.World.Renewal)
				if !ok || w.state.renewalCarry[r] != carry || g.carry[r] != carries[r] {
					t.Errorf("seed %d round %d: role %d carry before %d (view %d), removed %d: stored carry %d, rule %d (ok %v), want equal", seed, round, r, carries[r], g.carry[r], removed[r], w.state.renewalCarry[r], carry, ok)
				}
				want += n
				removedTotal += removed[r]
			}
			if got := g.entries + g.noViable + g.capped; got != want {
				t.Errorf("seed %d round %d: planned %d entries + %d dropped + %d cut = %d, want the rule's %d households", seed, round, g.entries, g.noViable, g.capped, got, want)
			}
			plannedTotal += g.entries
			auditOK(t, "guard", w)
		}
	}
	if removedTotal == 0 || plannedTotal == 0 {
		t.Errorf("removed %d and planned %d households over the seeds, want both above 0 so the guard is not vacuous", removedTotal, plannedTotal)
	}
}

// scriptPrivation draws, from rng, a state for every household agent that makes
// the default spawner remove or reduce some of them: unfed households, a privation
// at the removal line, a role taken long ago, and for a third of them no money; and
// it gives each miller a working record so the role has a wage and is viable.
func scriptPrivation(t *testing.T, w *World, rng *rand.Rand, seed uint64) {
	t.Helper()
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 {
			continue
		}
		a.unfed = 1 + rng.Int64N(a.weight)
		a.privation = w.cfg.World.RemovePrivation + rng.Int64N(3)
		a.roleSince = -100
		if rng.IntN(3) == 0 {
			m, _ := balance(t, w, a.id)
			if err := w.ledger.Transfer(a.account, w.estates.account(0), m, ledger.Starvation); err != nil && m > 0 {
				t.Fatalf("seed %d: Transfer() error = %v, want nil", seed, err)
			}
		}
		if a.role == 0 {
			a.history.Push(welfare.Record{Income: 3_000 * a.weight, Working: true, WorkHH: a.weight})
		}
	}
}

// TestExecute_CountsHoldAtEveryTickOfAScriptedRun verifies FR-SPN-034 over a run
// of Steps in which the plan changes the agent set every tick: an entry that
// creates a new agent, a reduction and a removal. The agent count rebuilt from
// AgentSpawned and AgentBankrupted and the household count also from WeightChanged
// equal the counters after every tick, the ledger audits, and the new agents flow
// through the later phases of the next ticks (their scratch grows).
//
// Arithmetic: the run starts with 3 agents and 4 households. Each tick the plan
// enters 2 households into the miller role (7 000 each from a pool of 1 000 000),
// and at tick 3 reduces agent 1 by 1 and removes agent 3.
//
// Requirement: FR-SPN-034, FR-SPN-045, FR-WLD-051, FR-AGT-001
func TestExecute_CountsHoldAtEveryTickOfAScriptedRun(t *testing.T) {
	t.Parallel()
	sp := &scriptSpawner{plan: func(v *RegionView) []Decision {
		plan := []Decision{enterPlan(2)}
		if v.Tick == 3 {
			plan = append(plan, Decision{Kind: DecideReduce, Agent: 1, Households: 1, Cause: CauseStarved}, Decision{Kind: DecideRemove, Agent: 3, Cause: CauseInsolvent})
		}
		return plan
	}}
	w, rec, _ := execWorld(t, execConfig(), 1_000_000, sp, WithAudit())
	for range 8 {
		step(t, w, 1)
		checkCounts(t, "scripted run", w, rec, 0, 0)
	}
	if w.live.agents <= 3 || w.agents[3].weight != 0 || w.state.nextAgent != AgentID(len(w.agents)) {
		t.Errorf("seed %d: agents %d, slot 3 weight %d, next ID %d of %d slots, want entrants, a removed slot 3 and dense IDs", execSeed, w.live.agents, w.agents[3].weight, w.state.nextAgent, len(w.agents))
	}
}

// TestExecute_TheWorldStepsThroughPhase7WithTheDefaultSpawner verifies the
// position: a run of Steps with the default spawner, ExitShare at its bound and a
// household with nothing to eat, removes the starving agent by the one removal
// path (AgentBankrupted with the cause of the decision, EstateOpened after it) and
// keeps the counts and the audit at every tick.
//
// Arithmetic: the weaver has no goods and no grain to buy, so it is unfed from
// tick 0; its privation is t + 1 after tick t and the ladder removes it once it is
// at RemovePrivation 10 and 10 ticks past RoleSince 0, that is at tick 9 or later
// (the Privation set at P6.1 of the same tick is read at P7.1).
//
// Requirement: FR-SPN-005, FR-SPN-034, FR-WLD-051, FR-WLD-053
func TestExecute_TheWorldStepsThroughPhase7WithTheDefaultSpawner(t *testing.T) {
	t.Parallel()
	c := exitConfig()
	c.World.ExitShare = 10_000
	rec := &evLog{}
	w := newMill(t, c, WithSeed(execSeed), WithAudit(), WithObserver(rec))
	for range 30 {
		step(t, w, 1)
		checkCounts(t, "default spawner", w, rec, 0, 0)
	}
	gone := of[AgentBankrupted](rec)
	if len(gone) == 0 {
		t.Fatalf("seed %d: no agent was removed in 30 ticks, want the starving ones removed by the default spawner", execSeed)
	}
	opened := of[EstateOpened](rec)
	if len(opened) != len(gone) || (gone[0].Cause != CauseStarved && gone[0].Cause != CauseInsolvent) {
		t.Errorf("seed %d: %d AgentBankrupted (cause %s), %d EstateOpened, want one EstateOpened each and a cause of the ladder", execSeed, len(gone), gone[0].Cause, len(opened))
	}
}

// TestExecute_RegionsInterleave verifies that the entrants of region r are
// created before region r + 1's Decide (spec-world.md 6.9.1): the spawner fake
// records, at each region's Decide, the agent count and the AgentSpawned events
// already emitted. The order is observed from the fake: the keyed streams move no
// word with the order of creation.
//
// Arithmetic: region 0's plan enters 3 households (a household brings 7 000 from
// 25 000): one fills agent 3 (weight 1, Scale 2) and one is a new agent that the
// third fills, so one AgentSpawned. Region 0's Decide sees the 4 initial agents
// and the construction's AgentSpawned events, region 1's sees 5 agents and one
// more AgentSpawned.
//
// Requirement: FR-SPN-002, FR-WLD-056
func TestExecute_RegionsInterleave(t *testing.T) {
	t.Parallel()
	type seen struct {
		region           RegionID
		agents, spawned  int64
		millerHouseholds int64
	}
	var log []seen
	var rec *evLog
	var w *World
	sp := &scriptSpawner{plan: regionPlans([]Decision{enterPlan(3)})}
	sp.onCall = func(v *RegionView) {
		log = append(log, seen{v.Region, w.live.agents, int64(len(of[AgentSpawned](rec))), v.Roles[0].Households})
	}
	w, rec, _ = execWorld(t, execConfig2(), 25_000, sp)
	initial := int64(len(of[AgentSpawned](rec)))
	runEntry(t, w, rec)
	if len(log) != 2 || log[0].region != 0 || log[1].region != 1 || log[0].agents != 4 || log[1].agents != 5 || log[0].spawned != 0 || log[1].spawned != 1 || log[1].millerHouseholds != 2 || initial != 4 {
		t.Errorf("seed %d: Decide saw %+v after %d initial spawns, want region 0 with 4 agents and none new, region 1 with 5 agents and 1 AgentSpawned", execSeed, log, initial)
	}
}

// TestExecute_ARemovalInPhase7ReplacesTheAgentWithNewIDs verifies a removal and
// its replacement in the region the account is in (two regions): the removed
// agent's job, money and goods are in the estate of its own region, its ID is not
// reused, the replacement takes the next ID with D10's two draws, and the events
// of the region come in order: RecipeAbandoned, AgentBankrupted, EstateOpened,
// then the entry's AgentSpawned and WeightChanged, then EntryLimited.
//
// Arithmetic: agent 5 (the miller of yonder, weight 2) holds 14 000 and has a job;
// the pool of yonder is 1 000, so it is 15 000 after the removal, the budget (share
// 10 000) 15 000 and the two households fund floor(15 000 / 7 000) = 2: a new agent
// 6 (start_money 7 000, the ID 5 is not reused) and agent 6 again (births 7 000), so
// the pool ends at 15 000 - 14 000 = 1 000. Home's pool does not move.
//
// Requirement: FR-WLD-051, FR-WLD-056, FR-WLD-059, FR-AGT-001, FR-OBS-020, FR-EST-002, FR-SPN-027
func TestExecute_ARemovalInPhase7ReplacesTheAgentWithNewIDs(t *testing.T) {
	t.Parallel()
	home := []Decision(nil)
	yonder := []Decision{{Kind: DecideRemove, Agent: 5, Cause: CauseStarved}, enterPlan(2)}
	w, rec, _ := execWorld(t, execConfig2(), 25_000, &scriptSpawner{plan: regionPlans(home, yonder)})
	if err := w.estates.seedPool(w.ledger, 1, 1_000); err != nil {
		t.Fatalf("seed %d: seedPool() error = %v, want nil", execSeed, err)
	}
	withJob(w, 5)
	agents0, hh0 := w.live.agents, w.live.households
	runEntry(t, w, rec)
	wantEvents(t, "removal and replacement", rec,
		RecipeAbandoned{Tick: 0, Agent: 5, Region: 1, Recipe: 0, Executions: 2, Reason: ReasonAgentRemoved},
		AgentBankrupted{Tick: 0, Agent: 5, Region: 1, Role: 0, Weight: 2, Cause: CauseStarved, Money: 14_000, GoodsListed: 2},
		EstateOpened{Tick: 0, Agent: 5, Region: 1, Balance: 15_000, GoodsListed: 2},
		AgentSpawned{Tick: 0, Agent: 6, Region: 1, Role: 0, Weight: 1, StartMoney: 7_000},
		WeightChanged{Tick: 0, Agent: 6, Region: 1, Role: 0, From: 1, To: 2, Cause: CauseRenewal, Money: 7_000},
		EntryLimited{Tick: 0, Region: 1, Entered: 2, Removed: 2})
	if poolOf(w, 1) != 1_000 || poolOf(w, 0) != 25_000 || w.agents[5].weight != 0 || w.agents[6].id != 6 || w.state.nextAgent != 7 {
		t.Errorf("seed %d: pools %d and %d, slot 5 weight %d, new ID %d, next %d, want 25 000, 1 000, 0, 6, 7", execSeed, poolOf(w, 0), poolOf(w, 1), w.agents[5].weight, w.agents[6].id, w.state.nextAgent)
	}
	ref := newStream(execSeed)
	ref.keyed(siteCreation, 0, 6)
	role := &w.cfg.Roles[0]
	wantTarget := newTarget(ref, role.TargetRatio, role.TargetSpread)
	if w.agents[6].target != wantTarget || w.agents[6].residues[0] != ref.draw(residueBound) {
		t.Errorf("seed %d: the replacement's draws are %d and %v, want the two words of the stream keyed (creation, 0, 6)", execSeed, w.agents[6].target, w.agents[6].residues)
	}
	checkCounts(t, "removal and replacement", w, rec, agents0, hh0)
	auditOK(t, "removal and replacement", w)
	if err := w.endStep(nil); err != nil {
		t.Fatalf("seed %d: endStep() error = %v, want nil", execSeed, err)
	}
	step(t, w, 2)
}

// TestExecute_ExtinctAndBelowFloorWarns verifies the two role Warns of the entry
// step (FR-SPN-042, FR-SPN-047, pattern A: the first tick of a run): "role
// extinct" for a role that had an agent and has none, and "group below floor" for
// a group of weight above 1 whose agent count is below MinAgents.
//
// Arithmetic: removing the weaver (agent 3, the only one of its role) leaves the
// role with 0 agents after it had 1, so the Warn names region 0 and role 1. With
// MinAgents 3 the miller group of 3 households is 3 agents of weight 1; removing
// one leaves weight 2 in 2 agents, below the floor of 3, so the Warn names region
// 0, role 0 and agents 2.
//
// Requirement: FR-SPN-042, FR-SPN-047, FR-SPN-036
func TestExecute_ExtinctAndBelowFloorWarns(t *testing.T) {
	t.Parallel()
	t.Run("extinct", func(t *testing.T) {
		t.Parallel()
		w, rec, lg := execWorld(t, execConfig(), 25_000, &scriptSpawner{plan: regionPlans([]Decision{{Kind: DecideRemove, Agent: 3, Cause: CauseStarved}})})
		runEntry(t, w, rec)
		lines := lg.take("role extinct")
		if len(lines) != 1 || lines[0].level != slog.LevelWarn || lines[0].attrs["region"] != 0 || lines[0].attrs["role"] != 1 {
			t.Errorf("seed %d: lines %+v, want one Warn \"role extinct\" for region 0 role 1", execSeed, lines)
		}
	})
	t.Run("below floor", func(t *testing.T) {
		t.Parallel()
		c := execConfig()
		c.World.MinAgents = 3
		w, rec, lg := execWorld(t, c, 25_000, &scriptSpawner{plan: regionPlans([]Decision{{Kind: DecideRemove, Agent: 1, Cause: CauseStarved}})})
		if w.live.inRole[0] != 3 {
			t.Fatalf("seed %d: miller agents = %d, want 3 of weight 1", execSeed, w.live.inRole[0])
		}
		runEntry(t, w, rec)
		lines := lg.take("group below floor")
		if len(lines) != 1 || lines[0].level != slog.LevelWarn || lines[0].attrs["region"] != 0 || lines[0].attrs["role"] != 0 || lines[0].attrs["agents"] != 2 {
			t.Errorf("seed %d: lines %+v, want one Warn \"group below floor\" for region 0 role 0 agents 2", execSeed, lines)
		}
	})
}

// TestExecute_TheCountersOfEachRegionAndRoleMoveTogether verifies, on the
// two-region world, that a switch in one region and an entry in the other move the
// per-(region, role) counters of their own cell and no other: the agents of a role,
// the flag that the role has had an agent, the pool of the region and the events of
// each region in order.
//
// Arithmetic: the cells are region * 2 + role (roles miller 0, weaver 1), so
// inRole = [home miller, home weaver, yonder miller, yonder weaver] = [2, 1, 1, 0]
// before. Home plan: agent 3 (home miller) becomes a weaver: [1, 2, 1, 0]. Yonder
// plan: agent 5 (yonder miller, weight 2) becomes a weaver: [1, 2, 0, 1]; then 3
// households enter as millers, with a scale of 2 per agent: agent 6 takes the first
// (weight 1), the second joins it (weight 2) and agent 7 takes the third: [1, 2, 2, 1].
// The yonder weaver cell had no agent and now has one, so hadAgent[3] turns true.
// The yonder pool of 25 000 pays 3 households at 7 000: 25 000 - 21 000 = 4 000, and
// the home pool is untouched.
//
// Requirement: FR-SPN-006, FR-SPN-033, FR-SPN-034, FR-SPN-044
// Guard: every cell index is region * roles + role; a scaled or divided index would
// credit a switch in yonder to home's counters and hide it from the spawner's signals.
func TestExecute_TheCountersOfEachRegionAndRoleMoveTogether(t *testing.T) {
	t.Parallel()
	sp := &scriptSpawner{plan: regionPlans(
		[]Decision{{Kind: DecideSwitch, Agent: 3, Role: 1, Cause: CauseReview}},
		[]Decision{{Kind: DecideSwitch, Agent: 5, Role: 1, Cause: CauseReview}, enterPlan(3)})}
	w, rec, _ := execWorld(t, execConfig2(), 25_000, sp)
	if err := w.estates.seedPool(w.ledger, 1, 25_000); err != nil {
		t.Fatalf("seed %d: seedPool() error = %v, want nil", execSeed, err)
	}
	if !slices.Equal(w.live.inRole, []int64{2, 1, 1, 0}) || !slices.Equal(w.state.hadAgent, []bool{true, true, true, false}) {
		t.Fatalf("seed %d: before: inRole %v, hadAgent %v, want [2 1 1 0] and [true true true false]", execSeed, w.live.inRole, w.state.hadAgent)
	}
	agents0, hh0 := w.live.agents, w.live.households
	runEntry(t, w, rec)
	wantEvents(t, "switch in home, switch and entry in yonder", rec,
		RoleChanged{Tick: 0, Agent: 3, Region: 0, From: 0, To: 1, Weight: 1, Cause: CauseReview, GoodsListed: 1},
		EntryLimited{Tick: 0, Region: 0, Changed: 1},
		RoleChanged{Tick: 0, Agent: 5, Region: 1, From: 0, To: 1, Weight: 2, Cause: CauseReview, GoodsListed: 1},
		AgentSpawned{Tick: 0, Agent: 6, Region: 1, Role: 0, Weight: 1, StartMoney: 7_000},
		WeightChanged{Tick: 0, Agent: 6, Region: 1, Role: 0, From: 1, To: 2, Cause: CauseRenewal, Money: 7_000},
		AgentSpawned{Tick: 0, Agent: 7, Region: 1, Role: 0, Weight: 1, StartMoney: 7_000},
		EntryLimited{Tick: 0, Region: 1, Changed: 2, Entered: 3})
	if !slices.Equal(w.live.inRole, []int64{1, 2, 2, 1}) || !slices.Equal(w.state.hadAgent, []bool{true, true, true, true}) {
		t.Errorf("seed %d: after: inRole %v, hadAgent %v, want [1 2 2 1] and all true", execSeed, w.live.inRole, w.state.hadAgent)
	}
	if poolOf(w, 0) != 25_000 || poolOf(w, 1) != 4_000 {
		t.Errorf("seed %d: pools = %d and %d, want 25 000 and 4 000", execSeed, poolOf(w, 0), poolOf(w, 1))
	}
	for id, want := range map[AgentID]RegionID{3: 0, 5: 1, 6: 1, 7: 1} {
		if got := w.regionOf(&w.agents[id]); got != want {
			t.Errorf("seed %d: agent %d is in region %d, want %d", execSeed, id, got, want)
		}
	}
	checkCounts(t, "two regions", w, rec, agents0, hh0)
	auditOK(t, "two regions", w)
}

// TestScans_SkipAnAgentThatWasRemoved verifies that an agent removed in phase 7
// keeps its slot with weight 0 and its closed account, and that every scan of the
// agents afterwards (the region view, the snapshot, the lookup of a decision's
// agent, the next tick) leaves it out without reading its account.
//
// Arithmetic: removing agent 3 (home miller, weight 1) leaves 2 agents and 3
// households in home (agent 2 with weight 2 and the weaver 4 with weight 1) and a
// dead slot; yonder is unchanged with 1 agent and 2 households.
//
// Requirement: FR-AGT-001, FR-SPN-032
// Guard: a removed agent's account is closed; a scan that read it would be an
// unreadable-account invariant on the tick after any removal.
func TestScans_SkipAnAgentThatWasRemoved(t *testing.T) {
	t.Parallel()
	sp := &scriptSpawner{}
	w, rec, _ := execWorld(t, execConfig2(), 25_000, sp)
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep() error = %v, want nil", err)
	}
	if err := w.removeAgent(&w.agents[3], CauseStarved); err != nil {
		t.Fatalf("removeAgent() error = %v, want nil", err)
	}
	if w.agents[3].weight != 0 {
		t.Fatalf("removed agent weight = %d, want 0", w.agents[3].weight)
	}
	if got := len(w.buildRegionView(0).Agents); got != 2 {
		t.Errorf("agents in the region view of home = %d, want 2 (the removed one is left out)", got)
	}
	if got := len(w.buildRegionView(1).Agents); got != 1 {
		t.Errorf("agents in the region view of yonder = %d, want 1", got)
	}
	if a := w.agentIn(0, 3); a != nil {
		t.Errorf("agentIn(home, 3) = %+v, want nil for a removed agent", a)
	}
	if a := w.agentIn(0, 2); a == nil || a.id != 2 {
		t.Errorf("agentIn(home, 2) = %v, want agent 2", a)
	}
	if a := w.agentIn(1, 2); a != nil {
		t.Errorf("agentIn(yonder, 2) = %v, want nil: agent 2 is in home", a)
	}
	rec.events = nil
	if err := w.endStep(nil); err != nil {
		t.Fatalf("endStep() error = %v, want nil", err)
	}
	s, err := w.Snapshot()
	if err != nil || s.Agents != 3 || s.Households != 5 || s.Regions[0].Agents != 2 || s.Regions[0].Households != 3 || s.Regions[1].Agents != 1 {
		t.Errorf("Snapshot() = agents %d, households %d, home %d/%d, err %v, want 3, 5, 2/3", s.Agents, s.Households, s.Regions[0].Agents, s.Regions[0].Households, err)
	}
	step(t, w, 2)
}

// TestRefusal_PassesWhatTheRulesAllowAtTheirEdges verifies the other side of the
// refusal table: the decisions that sit just inside each rule are not refused, and
// the ones just outside are, so a comparison off by one shows. The role IDs are
// 0 and 1 (two roles), the agent table has 6 slots (IDs 0 to 5, the estates 0 and
// 1), and a reduction is supported for the causes starved and insolvent only.
//
// Requirement: FR-SPN-004, FR-SPN-010
func TestRefusal_PassesWhatTheRulesAllowAtTheirEdges(t *testing.T) {
	t.Parallel()
	w, _, _ := execWorld(t, execConfig2(), 25_000, &scriptSpawner{})
	last := AgentID(len(w.agents) - 1)
	for _, tt := range []struct {
		name string
		d    Decision
		want Reason
	}{
		{"entry into the last role", Decision{Kind: DecideEnter, Role: 1, Households: 1, Cause: CauseRenewal}, ""},
		{"entry into the role past the last", Decision{Kind: DecideEnter, Role: 2, Households: 1, Cause: CauseRenewal}, ReasonUnknownRole},
		{"switch to the role past the last", Decision{Kind: DecideSwitch, Agent: 3, Role: 2, Cause: CauseReview}, ReasonUnknownRole},
		{"reduction by starvation", Decision{Kind: DecideReduce, Agent: 2, Households: 1, Cause: CauseStarved}, ""},
		{"reduction by insolvency", Decision{Kind: DecideReduce, Agent: 2, Households: 1, Cause: CauseInsolvent}, ""},
		{"reduction by the consumer", Decision{Kind: DecideReduce, Agent: 2, Households: 1, Cause: CauseConsumer}, ReasonUnsupportedCause},
		{"removal for any cause", Decision{Kind: DecideRemove, Agent: 2, Cause: CauseConsumer}, ""},
		{"switch for any cause", Decision{Kind: DecideSwitch, Agent: 3, Role: 1, Cause: CauseConsumer}, ""},
		{"removal of the last agent of the home region's table", Decision{Kind: DecideRemove, Agent: last, Cause: CauseStarved}, ReasonUnknownAgent},
		{"removal of the agent just past the table", Decision{Kind: DecideRemove, Agent: last + 1, Cause: CauseStarved}, ReasonUnknownAgent},
		{"removal of the first household", Decision{Kind: DecideRemove, Agent: 2, Cause: CauseStarved}, ""},
		{"removal of the last estate", Decision{Kind: DecideRemove, Agent: 1, Cause: CauseStarved}, ReasonUnknownAgent},
		{"the first kind", Decision{Kind: DecideRemove, Agent: 2, Cause: CauseStarved}, ""},
		{"the last kind", Decision{Kind: DecideEnter, Role: 0, Households: 1, Cause: CauseRenewal}, ""},
		{"the kind past the last", Decision{Kind: DecideEnter + 1, Agent: 2}, reasonUnknownKind},
		{"the kind before the first", Decision{Kind: DecideRemove - 1, Agent: 2}, reasonUnknownKind},
	} {
		if got := w.refusal(0, &tt.d); got != tt.want {
			t.Errorf("%s: refusal(%+v) = %q, want %q", tt.name, tt.d, got, tt.want)
		}
	}
}

// TestExecute_TwoEntriesShareOneBudget verifies that the budget of a region is
// taken by each entry in turn: a second entry in the same tick finds what the first
// left, not the whole budget again, so the two together never create more households
// than the budget buys, and the households the second could not buy are counted
// unfunded and logged as dropped.
//
// Arithmetic: the pool is 42 000 and EntryPoolShare is 5 000, so the budget is 21 000,
// half of the pool, and a household costs the miller's StartMoney, 7 000: 3 households.
// The first entry asks 2 and is funded (14 000 spent, 7 000 of the budget left); the
// second asks 2 and is funded 1 (7 000 left buys 1): Entered 3, Unfunded 1, the pool
// 42 000 - 21 000 = 21 000. A budget not reduced by the first entry would fund the
// second one in full (14 000 of a 21 000 pool share) and enter 4.
//
// Requirement: FR-SPN-005, FR-SPN-008, FR-EST-014
// Guard: a budget reset by each entry would let a plan spend the pool twice and the
// pool would go negative, which the ledger refuses.
func TestExecute_TwoEntriesShareOneBudget(t *testing.T) {
	t.Parallel()
	sp := &scriptSpawner{plan: regionPlans([]Decision{enterPlan(2), enterPlan(2)})}
	c := execConfig()
	c.World.EntryPoolShare = 5_000
	w, rec, lg := execWorld(t, c, 42_000, sp)
	runEntry(t, w, rec)
	var limited []EntryLimited
	for _, e := range rec.events {
		if l, ok := e.(EntryLimited); ok {
			limited = append(limited, l)
		}
	}
	if len(limited) != 1 || limited[0].Entered != 3 || limited[0].Unfunded != 1 {
		t.Errorf("seed %d: EntryLimited = %+v, want one with Entered 3 and Unfunded 1", execSeed, limited)
	}
	if got := poolOf(w, 0); got != 21_000 {
		t.Errorf("seed %d: pool = %d, want 21 000 after 3 households at 7 000 out of 42 000", execSeed, got)
	}
	if dropped := lg.take("entry dropped"); len(dropped) != 1 || dropped[0].strs["reason"] != string(ReasonUnfunded) {
		t.Errorf("seed %d: \"entry dropped\" lines = %+v, want one with the reason unfunded", execSeed, dropped)
	}
}

// TestLogNoViable_OneLinePerRoleThatLostRenewals verifies that the Debug lines
// "entry dropped" for renewals the default found no viable role for are written for
// the roles with a count above 0 and not for the roles with 0, one each, in role
// order, with the reason no_viable_role.
//
// Arithmetic: counts [0, 2] for the two roles give one line, for role 1; [3, 0]
// gives one line for role 0; [0, 0] gives none.
//
// Requirement: FR-SPN-008, FR-OBS-014
func TestLogNoViable_OneLinePerRoleThatLostRenewals(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		counts []int64
		roles  []int64
	}{{[]int64{0, 2}, []int64{1}}, {[]int64{3, 0}, []int64{0}}, {[]int64{0, 0}, nil}, {[]int64{1, 1}, []int64{0, 1}}} {
		w, _, lg := execWorld(t, execConfig(), 0, &scriptSpawner{})
		w.logNoViable(&Decisions{noViableByRole: tt.counts})
		var got []int64
		for _, l := range lg.take("entry dropped") {
			if l.strs["reason"] != string(ReasonNoViableRole) {
				t.Errorf("counts %v: a line with the reason %q, want no_viable_role", tt.counts, l.strs["reason"])
			}
			got = append(got, l.attrs["role"])
		}
		if !slices.Equal(got, tt.roles) {
			t.Errorf("counts %v: lines for roles %v, want %v", tt.counts, got, tt.roles)
		}
	}
}

// TestRegionView_FieldsAreReadFromTheirOwnRegionAndRole verifies four things of the
// view the spawner is given that a one-region world cannot show: the prices are those
// of the region viewed, the role's renewal carry is that of its own (region, role)
// cell, the tool price counts a tool once however many recipes need it, and the
// configured cells are the (region, role) pairs that have a group.
//
// Arithmetic: the groups of removalConfig are (home, miller), (home, weaver) and
// (yonder, miller), so configured = [true, true, true, false] over region * 2 + role.
// The prices of flour are 3 000 at home and set to 3 111 in yonder. The carry 77 is
// set on the yonder weaver (cell 3) only. The miller has two recipes that both need
// the pan, priced 5 000, so its tool price is 5 000 and not 10 000, and asking
// twice gives the same answer.
//
// Requirement: FR-SPN-003, FR-SPN-004
// Guard: each of the four is indexed by region * roles + role, or by commodity; an
// index that ignored the region would show one region's market to another's spawner.
func TestRegionView_FieldsAreReadFromTheirOwnRegionAndRole(t *testing.T) {
	t.Parallel()
	c := execConfig2()
	c.Recipes = append(c.Recipes, Recipe{
		ID: "mill2", Duration: 1, TraineeYield: 5_000,
		Inputs:   []Item{{Commodity: "grain", Quantity: 1}},
		Outputs:  []Output{{Commodity: "flour", Quantity: 1, CostShare: 10_000}},
		Requires: []Tool{{Commodity: "pan", WearPerUse: 1}},
	})
	c.Roles[0].Recipes = []string{"mill", "mill2"}
	w, _, _ := execWorld(t, c, 25_000, &scriptSpawner{})
	if want := []bool{true, true, true, false}; !slices.Equal(w.ex.configured, want) {
		t.Errorf("configured = %v, want %v", w.ex.configured, want)
	}
	w.markets.reports[w.markets.index(1, flour)].Price = 3_111
	w.state.renewalCarry[3] = 77
	// The view is the world's one scratch, so what is read from it is read before the
	// next call.
	var homeFlour, yonderFlour Price
	var homeCarry, yonderCarry [2]Money
	for g, flourOf := range []*Price{&homeFlour, &yonderFlour} {
		v := w.buildRegionView(RegionID(g))
		*flourOf = Price(w.ex.prices[flour])
		carry := &homeCarry
		if g == 1 {
			carry = &yonderCarry
		}
		for r := range carry {
			carry[r] = Money(v.Roles[r].RenewalCarry)
		}
	}
	if homeFlour != 3_000 || yonderFlour != 3_111 {
		t.Errorf("price of flour in the view of home and of yonder = %d and %d, want 3000 and 3111", homeFlour, yonderFlour)
	}
	if yonderCarry != [2]Money{0, 77} || homeCarry != [2]Money{0, 0} {
		t.Errorf("renewal carry of the roles in yonder %v and in home %v, want [0 77] and [0 0]", yonderCarry, homeCarry)
	}
	for range 2 {
		if got := w.toolPrice(0); got != 5_000 {
			t.Errorf("toolPrice(miller) = %d, want 5000: the pan counted once for two recipes, on every call", got)
		}
	}
	if got := w.toolPrice(1); got != 0 {
		t.Errorf("toolPrice(weaver) = %d, want 0: the weaver needs no tool", got)
	}
}

// TestExecute_EntrantsGoToTheHostsOfTheirOwnRegionAndScale verifies the placement of
// an entry on the two-region world: with the scale of yonder's group at 2 its one
// agent (weight 2) is full, so one household enters as a new agent of yonder, and
// the millers of home, who have room, are not hosts; with the scale at 4 the
// household joins agent 5 instead (weight 2 to 3), while home's scale of 2 is not
// the one used. An entry into an empty (region, role) cell sets its has-had-an-agent
// flag and its count.
//
// Arithmetic: scale 2, weight 2: no room, new agent 6 of weight 1. Scale 4, weight 2:
// room 2, so 1 household joins: weight 3 and the pool pays 7 000 by births. The
// yonder weaver cell (index 3) starts with no agent and no flag; one weaver household
// enters at 3 000 and the cell has 1 agent.
//
// Requirement: FR-SPN-006, FR-SPN-007, FR-SPN-033, FR-SPN-044
// Guard: a host list that took agents of another region would put a household's money
// in the wrong region's account.
func TestExecute_EntrantsGoToTheHostsOfTheirOwnRegionAndScale(t *testing.T) {
	t.Parallel()
	enterYonder := func(c Config, d Decision) (*World, *evLog) {
		w, rec, _ := execWorld(t, c, 25_000, &scriptSpawner{plan: regionPlans(nil, []Decision{d})})
		if err := w.estates.seedPool(w.ledger, 1, 25_000); err != nil {
			t.Fatalf("seed %d: seedPool() error = %v, want nil", execSeed, err)
		}
		runEntry(t, w, rec)
		return w, rec
	}
	t.Run("the host is full", func(t *testing.T) {
		t.Parallel()
		w, rec := enterYonder(execConfig2(), enterPlan(1))
		wantEvents(t, "a full host", rec,
			AgentSpawned{Tick: 0, Agent: 6, Region: 1, Role: 0, Weight: 1, StartMoney: 7_000},
			EntryLimited{Tick: 0, Region: 1, Entered: 1})
		if w.regionOf(&w.agents[6]) != 1 {
			t.Errorf("seed %d: the entrant is in region %d, want 1", execSeed, w.regionOf(&w.agents[6]))
		}
	})
	t.Run("the host has room under the scale of its own group", func(t *testing.T) {
		t.Parallel()
		c := execConfig2()
		c.Groups[2].Scale = 4 // yonder miller: the groups are in canonical order, home first
		w, rec := enterYonder(c, enterPlan(1))
		wantEvents(t, "a host with room", rec,
			WeightChanged{Tick: 0, Agent: 5, Region: 1, Role: 0, From: 2, To: 3, Cause: CauseRenewal, Money: 7_000},
			EntryLimited{Tick: 0, Region: 1, Entered: 1})
		if w.agents[5].weight != 3 || poolOf(w, 1) != 18_000 {
			t.Errorf("seed %d: weight %d, pool %d, want 3 and 18 000", execSeed, w.agents[5].weight, poolOf(w, 1))
		}
	})
	t.Run("an empty cell gets its flag and its count", func(t *testing.T) {
		t.Parallel()
		w, rec := enterYonder(execConfig2(), Decision{Kind: DecideEnter, Role: 1, Households: 1, Cause: CauseRenewal})
		wantEvents(t, "an empty cell", rec,
			AgentSpawned{Tick: 0, Agent: 6, Region: 1, Role: 1, Weight: 1, StartMoney: 3_000},
			EntryLimited{Tick: 0, Region: 1, Entered: 1})
		if w.live.inRole[3] != 1 || !w.state.hadAgent[3] {
			t.Errorf("seed %d: yonder weaver agents %d, hadAgent %v, want 1 and true", execSeed, w.live.inRole[3], w.state.hadAgent[3])
		}
	})
}

// TestSwitch_WritesItsDebugLine verifies the Debug line "role changed" of a role
// change of the weaver, agent 3, to the miller (role 0): the agent, its new role and
// the cause as the reason.
//
// Requirement: FR-SPN-006, FR-OBS-014
func TestSwitch_WritesItsDebugLine(t *testing.T) {
	t.Parallel()
	sp := &scriptSpawner{plan: regionPlans([]Decision{{Kind: DecideSwitch, Agent: 3, Role: 0, Cause: CauseReview}})}
	w, rec, lg := execWorld(t, execConfig(), 25_000, sp)
	runEntry(t, w, rec)
	lines := lg.take("role changed")
	if len(lines) != 1 || lines[0].level != slog.LevelDebug || lines[0].attrs["agent"] != 3 || lines[0].attrs["role"] != 0 || lines[0].strs["reason"] != string(CauseReview) {
		t.Errorf("seed %d: \"role changed\" lines = %+v, want one Debug line for agent 3, role 0, reason review", execSeed, lines)
	}
}

// TestSpawnEntrant_TheIDSpaceMustBeDense verifies that creating an entrant whose next
// ID is not the next slot of the agent table is a violated invariant and changes
// nothing.
//
// Requirement: FR-SPN-007
func TestSpawnEntrant_TheIDSpaceMustBeDense(t *testing.T) {
	t.Parallel()
	for _, next := range []AgentID{5, 99} {
		w, _, _ := execWorld(t, execConfig(), 25_000, &scriptSpawner{})
		n, accounts := len(w.agents), w.ledger.Len()
		w.state.nextAgent = next
		msg := invariantMessage(func() { _ = w.spawnEntrant(0, Decision{Kind: DecideEnter, Role: 0, Households: 1}) })
		if !strings.Contains(msg, "the agent ID space is not dense") || len(w.agents) != n || w.ledger.Len() != accounts {
			t.Errorf("seed %d: next ID %d over %d slots: panic %q, agents %d, accounts %d, want the dense-space invariant and nothing opened", execSeed, next, n, msg, len(w.agents), w.ledger.Len())
		}
	}
}
