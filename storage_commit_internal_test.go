package economysim

import (
	"errors"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The world of the commitment tests is shipConfig with the storage module and the
// recipe mill of 3 ticks: two grain (3, 2) make three flour (3, 2), so an execution
// takes (6, 4) of input and pledges (9, 6) of output, a net growth of (3, 2) once the
// input is freed. A household holds 10 grain (30, 20) and eats 1 grain (3, 2) a tick,
// which is its upkeep room. The households are agents 3 and 4 in a_home (region 0)
// and 5 and 6 in b_far (region 1).
const (
	stoMill   RecipeID = 0 // mill; pack is 1
	stoCoach  ModeID   = modeCoach
	stoFootID ModeID   = modeFoot
)

// stoMiller is a Strategy that requests the recipe mill, up to max executions, whenever
// it is free, and trades nothing.
type stoMiller struct{ max int64 }

func (s stoMiller) Work(*AgentView, *rand.Rand) JobRequest {
	return JobRequest{Recipe: stoMill, Executions: s.max}
}
func (stoMiller) Trade(*AgentView, *rand.Rand, *TradePlan)   {}
func (stoMiller) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = stoMiller{}

// stoMillWorld is the world of shipConfig where the millers run the mill recipe (the
// role's list is the mill only) of three ticks under strat, with the storage module
// giving every role the base room (mass, bulk) and a class barn of (100, 120), changed
// by mutate. The audit is on, so every step checks the pledge of every account.
func stoMillWorld(t *testing.T, mass, bulk int64, strat Strategy, mutate func(c *Config)) (*World, *evLog) {
	t.Helper()
	c := shipConfig()
	c.Recipes[0].Duration = 3
	c.Roles[0].Recipes = []string{"mill"}
	c.Modules.Storage = &Storage{
		Base:    &Capacity{Mass: mass, Bulk: bulk},
		Classes: []StoreClass{{ID: "barn", Mass: 100, Bulk: 120}},
	}
	if mutate != nil {
		mutate(&c)
	}
	rec := &evLog{}
	w := newMill(t, c, WithObserver(rec), WithAudit(), WithStrategy("script", strat))
	if err := w.ledger.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction() = %v, want nil", err)
	}
	return w, rec
}

// TestStart_KIsMinK0KRoom verifies the start of a job under a room: the executions
// are the least of the request and what the room holds once the upkeep room is left
// free, a start whose room holds none is refused with no_room, consuming nothing and
// counted in RecipeRefused.NoRoom, and a start that fits pledges the exact load of its
// output.
//
// Arithmetic: the household holds 10 grain, a load of (30, 20), so the free room is
// (room - 30, room - 20) and its upkeep room is (3, 2). One execution takes 2 grain and
// gives 3 flour, so the room must hold a net growth of (9 - 6, 6 - 4) = (3, 2) beyond
// the upkeep room: a free room of (6, 4) at the least, a base room of (36, 24). The
// rows below are that room, one mass short, one bulk short, and the room (33, 22)
// whose free room (3, 2) holds the growth but leaves no upkeep room (FR-STO-046).
//
// Requirement: FR-STO-011, FR-STO-012, FR-STO-013, FR-STO-046
func TestStart_KIsMinK0KRoom(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mass, bulk int64
		started    bool
	}{
		{"room exactly", 36, 24, true},
		{"one mass short", 35, 24, false},
		{"one bulk short", 36, 23, false},
		{"growth fits the free room but not the upkeep room", 33, 22, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := stoMillWorld(t, tt.mass, tt.bulk, stoMiller{}, nil)
			step(t, w, 1)
			st := roomOfAgent(t, w, shipHome)
			agent, err := w.Agent(shipHome)
			if err != nil {
				t.Fatalf("Agent() = %v, want nil", err)
			}
			if agent.HasJob != tt.started {
				t.Errorf("HasJob after the first step = %t, want %t", agent.HasJob, tt.started)
			}
			wantPledge := Capacity{}
			if tt.started {
				wantPledge = Capacity{Mass: 9, Bulk: 6}
			}
			if st.Pledge != wantPledge {
				t.Errorf("pledge after the first step = %+v, want %+v (the exact load of 3 flour)", st.Pledge, wantPledge)
			}
			refused := int64(0)
			for _, e := range of[RecipeRefused](rec) {
				refused += e.NoRoom
			}
			// Each of the 4 households is counted twice: once at P2.5 when its view says the
			// only recipe is blocked, and once more for the request of this strategy, which
			// asks for it whatever the view says.
			if want := int64(8); !tt.started && refused != want {
				t.Errorf("RecipeRefused.NoRoom over the first step = %d, want %d (4 households, twice each)", refused, want)
			}
			if tt.started && refused != 0 {
				t.Errorf("RecipeRefused.NoRoom over the first step = %d, want 0", refused)
			}
			if !tt.started {
				if held, _, _ := w.ledger.Holding(w.agents[shipHome].account, int(commodityOf(t, w, "grain"))); held != 9 {
					t.Errorf("grain held after a refused start = %d, want 9: 10 less the 1 eaten, none consumed by the start", held)
				}
			}
		})
	}
}

// TestStart_CutCountsExecutionsAndKeepsTheRest verifies that a start sized below the
// request by the room is a partial start: k is the executions the room holds, the
// others are counted in the counters of RoomLimited, and the output of k is pledged.
//
// Arithmetic: weight 2 gives a room of 2 * (36, 24) = (72, 48), and 21 grain (63, 42)
// leave a free room of (9, 6); the upkeep room of two households is (6, 4), so (3, 2)
// is free for growth, which holds one execution (net (3, 2)) and not two.
//
// Requirement: FR-STO-011, FR-STO-012
func TestStart_CutCountsExecutionsAndKeepsTheRest(t *testing.T) {
	t.Parallel()
	w, _ := stoMillWorld(t, 36, 24, stoMiller{}, nil)
	a := &w.agents[shipHome]
	a.weight = 2
	w.weightChangedStorage(a)
	stoGrain(t, w, shipHome, 11) // 21 grain
	cell := w.cellOf(regHome, a)
	tickInspecting(t, w, func() {
		if got := w.sto.counts[cell].startsCut; got != 1 {
			t.Errorf("startsCut after the starts = %d, want 1 (2 asked, 1 held)", got)
		}
	})
	agent, err := w.Agent(shipHome)
	if err != nil || agent.Executions != 1 || agent.Room.Pledge != (Capacity{Mass: 9, Bulk: 6}) {
		t.Errorf("job of the cut start = %d executions, pledge %+v, %v, want 1 execution and the pledge (9, 6)", agent.Executions, agent.Room.Pledge, err)
	}
	if w.sto.counts[cell].startsCut != 0 {
		t.Errorf("startsCut after the step = %d, want 0: the counters are cleared at P8.6", w.sto.counts[cell].startsCut)
	}
}

// TestStart_DiscardOutputTakesNone verifies that a discard output is created and
// disposed at the completion and so takes no room and is not pledged: a recipe with a
// waste output of size (5, 5) beside the flour pledges the flour alone and starts in
// the same room.
//
// Requirement: FR-STO-012, FR-STO-013
func TestStart_DiscardOutputTakesNone(t *testing.T) {
	t.Parallel()
	w, _ := stoMillWorld(t, 36, 24, stoMiller{}, func(c *Config) {
		c.Commodities = append(c.Commodities, Commodity{ID: "waste", Mass: 5, Bulk: 5, ReferencePrice: 10})
		c.Recipes[0].Outputs = append(c.Recipes[0].Outputs, Output{Commodity: "waste", Quantity: 1, CostShare: 0, Discard: true})
	})
	step(t, w, 1)
	if st := roomOfAgent(t, w, shipHome); st.Pledge != (Capacity{Mass: 9, Bulk: 6}) {
		t.Errorf("pledge of a job with a discard output = %+v, want (9, 6): the waste takes none", st.Pledge)
	}
}

// TestComplete_CreditsFromPledge verifies that the completion credits the output from
// the pledge, which falls to what the next job pledges, and that the run passes the
// audit each tick (the identity that the pledge is what the job and the cargo imply).
//
// Arithmetic: the job of three ticks starts at tick 0 and completes at tick 2: 3 flour
// of (3, 2) are created.
//
// Requirement: FR-STO-013, FR-STO-020, FR-STO-040
func TestComplete_CreditsFromPledge(t *testing.T) {
	t.Parallel()
	w, rec := stoMillWorld(t, 120, 80, stoMiller{}, nil)
	for range 3 {
		step(t, w, 1)
	}
	done := of[RecipeExecuted](rec)
	if len(done) == 0 || done[0].Tick != 2 || len(done[0].Disposed) != 0 {
		t.Fatalf("RecipeExecuted events = %+v, want completions at tick 2 with nothing disposed", done)
	}
	flour, _, _ := w.ledger.Holding(w.agents[done[0].Agent].account, int(commodityOf(t, w, "flour")))
	if flour != 3 {
		t.Errorf("flour held by the first completer = %d, want 3", flour)
	}
}

// TestComplete_AfterCutDisposesRemainderInCommodityOrder verifies example 18.4 in
// miniature: a cut of the room while a job runs (a forced removal of a store) leaves
// the pledge above what the room can hold; the completion credits the output up to the
// room left, whole units, and creates and disposes of the rest under disposal, which
// RecipeExecuted reports as Disposed and Overflow; the pledge ends at 0.
//
// Arithmetic: the household has the base room (36, 24) and a barn (100, 120). After the
// first step it holds 7 grain (21, 14) (10, less the 2 the start took and the 1 eaten);
// 5 more make 12 grain (36, 24), which fills the base room. The barn is removed (forced,
// no goods move: the load fits). One more is eaten at tick 1, so at the completion the
// load is (33, 22) and the room left is (3, 2): one flour (3, 2) is credited, and 2 of
// the 3 are disposed.
//
// Requirement: FR-STO-013, FR-STO-017
func TestComplete_AfterCutDisposesRemainderInCommodityOrder(t *testing.T) {
	t.Parallel()
	w, rec := stoMillWorld(t, 36, 24, stoMiller{}, nil)
	stoAdd(t, w, shipHome, 0)
	step(t, w, 1)
	stoGrain(t, w, shipHome, 5)
	if _, err := w.removeStore(&w.agents[shipHome], 0, CauseFacility, true); err != nil {
		t.Fatalf("removeStore(forced) = %v, want nil", err)
	}
	if st := roomOfAgent(t, w, shipHome); st.Pledge != (Capacity{Mass: 9, Bulk: 6}) || st.Free != (Capacity{}) {
		t.Fatalf("room group after the cut = %+v, want the pledge (9, 6) kept and the free room read as 0", st)
	}
	step(t, w, 1)
	step(t, w, 1)
	var got RecipeExecuted
	for _, e := range of[RecipeExecuted](rec) {
		if e.Agent == shipHome {
			got = e
		}
	}
	flour := commodityOf(t, w, "flour")
	if len(got.Outputs) != 1 || got.Outputs[0] != (Amount{Commodity: flour, Quantity: 3}) ||
		len(got.Disposed) != 1 || got.Disposed[0] != (Amount{Commodity: flour, Quantity: 2}) ||
		len(got.Overflow) != 1 || got.Overflow[0] != (Amount{Commodity: flour, Quantity: 2}) {
		t.Errorf("RecipeExecuted of the cut completion = %+v, want 3 flour created, 2 disposed and 2 overflow", got)
	}
	if held, _, _ := w.ledger.Holding(w.agents[shipHome].account, int(flour)); held != 1 {
		t.Errorf("flour held after the cut completion = %d, want 1", held)
	}
	if st := roomOfAgent(t, w, shipHome); st.Pledge != (Capacity{}) {
		t.Errorf("pledge after the completion = %+v, want (0, 0)", st.Pledge)
	}
}

// TestAbandon_DropsPledge verifies that a job abandoned (here by a role change) leaves
// the pledge with only what the cargo in flight implies, which here is nothing.
//
// Requirement: FR-STO-013, FR-STO-020
func TestAbandon_DropsPledge(t *testing.T) {
	t.Parallel()
	w, _ := stoMillWorld(t, 120, 80, stoMiller{}, nil)
	step(t, w, 1)
	if st := roomOfAgent(t, w, shipHome); st.Pledge == (Capacity{}) {
		t.Fatalf("pledge of a running job = %+v, want its output's load", st.Pledge)
	}
	w.abandonJob(&w.agents[shipHome], regHome, ReasonRoleChanged)
	if st := roomOfAgent(t, w, shipHome); st.Pledge != (Capacity{}) {
		t.Errorf("pledge after the job is abandoned = %+v, want (0, 0)", st.Pledge)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestWeightReduction_LoadFitsPledgeMayExceed verifies spec-storage.md 5.5.3: a weight
// reduction takes the goods and the room down together, so the load stays within the
// room, the reduction is never refused, and the pledge of a job that keeps all its
// output (one execution, below the new weight) is left as it is and may exceed what
// the room can hold; the completion then cuts it.
//
// Arithmetic: weight 10 with a base of (36, 24) is a room of (360, 240). A recipe that
// makes 100 flour (300, 200) from 2 grain pledges (300, 200), which the free room
// (330, 220) less the upkeep room (30, 20) holds. Halving the weight gives (180, 120):
// the pledge (300, 200) is above it; the load (grain only, at most (15, 10)) is not.
//
// Requirement: FR-STO-017, FR-STO-013
func TestWeightReduction_LoadFitsPledgeMayExceed(t *testing.T) {
	t.Parallel()
	w, rec := stoMillWorld(t, 36, 24, stoMiller{max: 1}, func(c *Config) { c.Recipes[0].Outputs[0].Quantity = 100 })
	a := &w.agents[shipHome]
	a.weight = 10
	w.weightChangedStorage(a)
	step(t, w, 1)
	if st := roomOfAgent(t, w, shipHome); st.Pledge != (Capacity{Mass: 300, Bulk: 200}) {
		t.Fatalf("pledge of the job of weight 10 = %+v, want (300, 200)", st.Pledge)
	}
	if err := w.execReduce(regHome, a, Decision{Agent: shipHome, Households: 5, Cause: CauseStarved}); err != nil {
		t.Fatalf("execReduce() = %v, want nil: a reduction is never refused for room", err)
	}
	st := roomOfAgent(t, w, shipHome)
	if st.Room != (Capacity{Mass: 180, Bulk: 120}) || st.Pledge != (Capacity{Mass: 300, Bulk: 200}) || st.Free != (Capacity{}) {
		t.Errorf("room group after the reduction = %+v, want room (180, 120), pledge (300, 200) kept and free 0", st)
	}
	if st.Load.Mass > st.Room.Mass || st.Load.Bulk > st.Room.Bulk {
		t.Errorf("load %+v above room %+v after the reduction", st.Load, st.Room)
	}
	step(t, w, 1)
	step(t, w, 1)
	for _, e := range of[RecipeExecuted](rec) {
		if e.Agent == shipHome && len(e.Disposed) == 0 {
			t.Errorf("RecipeExecuted of the cut completion = %+v, want part of the 100 flour disposed", e)
		}
	}
}

// TestRoleChange_ExcessToEstateUnderEstateGoods verifies spec-storage.md 5.5.2: a role
// change to a limited role whose room is smaller than the load moves whole lots to the
// estate, under estate_goods and counted in RoleChanged.GoodsListed, by the rule of
// 5.5.2.1, until the load fits, and the change is not refused.
//
// Arithmetic: the household holds one lot of 10 grain (3, 2), a load of (30, 20); the
// new role has a room of (20, 10): the shortfall (10, 10) relative to the room is 0.5
// in mass and 1 in bulk, so the bulk binds; grain is the one commodity held, and its lot
// is whole, so all 10 go: one lot listed.
//
// Requirement: FR-STO-018
func TestRoleChange_ExcessToEstateUnderEstateGoods(t *testing.T) {
	t.Parallel()
	w, rec := stoWorld(t, func(c *Config) {
		tiny := DefaultRole()
		tiny.ID, tiny.Trades = "tiny", []string{"grain", "flour"}
		c.Roles = append(c.Roles, tiny)
		c.Modules.Storage.Roles = append(c.Modules.Storage.Roles, RoleRoom{Role: "tiny", Base: &Capacity{Mass: 20, Bulk: 10}})
		c.Modules.Storage = c.Modules.Storage.canonical()
	})
	grainID := commodityOf(t, w, "grain")
	estate := w.estates.accounts[regHome]
	before, _, _ := w.ledger.Holding(estate, int(grainID))
	if err := w.execSwitch(regHome, &w.agents[shipHome], Decision{Role: 2, Cause: CauseReview}); err != nil {
		t.Fatalf("execSwitch() = %v, want nil: a role change is never refused for room", err)
	}
	if held, _, _ := w.ledger.Holding(w.agents[shipHome].account, int(grainID)); held != 0 {
		t.Errorf("grain held after the change = %d, want 0: the lot went to the estate", held)
	}
	if after, _, _ := w.ledger.Holding(estate, int(grainID)); after-before != 10 {
		t.Errorf("grain the estate received = %d, want 10", after-before)
	}
	ev := of[RoleChanged](rec)
	if len(ev) != 1 || ev[0].GoodsListed != 1 {
		t.Errorf("RoleChanged events = %+v, want one with GoodsListed 1 (the lot of the excess)", ev)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestPledge_DerivedNotSaved_RecomputedAtEveryChange verifies that the pledge of the
// account is the same value whichever way the world reaches it: after a start, a
// completion, an abandon and a reduction the audit's identity (the pledge is what the
// job and the cargo in flight imply) holds, and the pledge is not a field of the
// agent.
//
// Requirement: FR-STO-020, FR-STO-040
func TestPledge_DerivedNotSaved_RecomputedAtEveryChange(t *testing.T) {
	t.Parallel()
	w, _ := stoMillWorld(t, 120, 80, stoMiller{}, nil)
	for i, what := range []func(){
		func() { step(t, w, 1) },
		func() { step(t, w, 1) },
		func() { step(t, w, 1) }, // the completion
		func() { w.abandonJob(&w.agents[shipHome], regHome, ReasonRoleChanged) },
	} {
		what()
		if err := w.Audit(); err != nil {
			t.Fatalf("Audit() after change %d = %v, want nil", i, err)
		}
	}
}

// TestCredit_NoRoomWhereCommittedPanicsOrAuditFails verifies FR-STO-016: a credit into
// a limited account that no room was committed for is refused by the ledger with
// ErrNoRoom, changing nothing, and a path that is guaranteed room (a settlement) that
// meets it is a violated invariant that panics with a message that says so, rather than
// losing goods; a deliberately broken caller stands for the defect.
//
// Arithmetic: the buyer holds 10 grain (30, 20) in a room of (36, 24), a free room of
// (6, 4); 3 grain are (9, 6), above it in both dimensions.
//
// Requirement: FR-STO-016
func TestCredit_NoRoomWhereCommittedPanicsOrAuditFails(t *testing.T) {
	t.Parallel()
	w, _ := stoMillWorld(t, 36, 24, stoMiller{}, nil)
	a, b := &w.agents[shipHome], &w.agents[shipHome2]
	grain := int(commodityOf(t, w, "grain"))
	err := w.ledger.MoveCredited(a.account, b.account, grain, 3, ledger.Trade, false, ledger.CreditFree)
	if !errors.Is(err, ledger.ErrNoRoom) {
		t.Fatalf("an unmarked credit of 3 grain into a free room of (6, 4) = %v, want ErrNoRoom", err)
	}
	defer func() {
		r, _ := recover().(string)
		if !strings.Contains(r, "invariant") {
			t.Errorf("settlement that meets ErrNoRoom: recover() = %q, want a panic that says invariant", r)
		}
	}()
	mustSettle(err, "the delivery")
}

// TestPledge_TraineeJobPledgesThePrincipal verifies that the output of a job whose principal
// is another agent (a trainee's job, spec-production.md 4.3.2) is pledged on the
// principal's account, not the executor's, and leaves it when the job does; the audit
// of the pledge agrees at each point.
//
// Arithmetic: 3 flour of (3, 2) are (9, 6).
//
// Requirement: FR-STO-013, FR-STO-020
func TestPledge_TraineeJobPledgesThePrincipal(t *testing.T) {
	t.Parallel()
	w, _ := stoMillWorld(t, 120, 80, stoMiller{}, nil)
	trainee := &w.agents[shipHome2]
	trainee.job = job{recipe: stoMill, remaining: 2, executions: 1, principal: shipHome, yield: 10_000, pending: []int64{3}}
	w.syncJob(trainee)
	if got, want := roomOfAgent(t, w, shipHome).Pledge, (Capacity{Mass: 9, Bulk: 6}); got != want {
		t.Errorf("pledge of the principal = %+v, want %+v", got, want)
	}
	if got := roomOfAgent(t, w, shipHome2).Pledge; got != (Capacity{}) {
		t.Errorf("pledge of the executor = %+v, want (0, 0)", got)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() with a trainee's job = %v, want nil", err)
	}
	trainee.job.clear()
	w.syncJob(trainee)
	if got := roomOfAgent(t, w, shipHome).Pledge; got != (Capacity{}) {
		t.Errorf("pledge of the principal after the job ends = %+v, want (0, 0)", got)
	}
}

// TestRoleChange_TrainerToLimitedRolePledgesItsTraineesOutput verifies that a trainer
// whose role has no base room, with a trainee's job pledged on it, is given the
// pledge of that job in the ledger when it changes to a limited role: the load was
// tracked while the account was unlimited and must reach the ledger with the room.
//
// Arithmetic: 3 flour of (3, 2) are (9, 6).
//
// Requirement: FR-STO-013, FR-STO-034
//
// Guard: the pledge a trainee put on an unlimited trainer is tracked but not set in
// the ledger (the account has no room). Without the repledge of the role change the
// trainer would hold a limited account with a pledge of 0 against the (9, 6) its
// trainee's job implies, and the audit would name it. (The explicit repledge of the
// agent in roleChangedStorage is covered by syncJob for an agent with no job; the test
// fails only if both go.)
func TestRoleChange_TrainerToLimitedRolePledgesItsTraineesOutput(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	trainer, trainee := &w.agents[shipHome], &w.agents[shipHome2]
	if err := w.execSwitch(regHome, trainer, Decision{Role: 1, Cause: CauseReview}); err != nil {
		t.Fatalf("execSwitch() to the unlimited role = %v, want nil", err)
	}
	trainee.job = job{recipe: stoMill, remaining: 2, executions: 1, principal: shipHome, yield: 10_000, pending: []int64{3}}
	w.syncJob(trainee)
	if st := roomOfAgent(t, w, shipHome); st.Limited || st.Pledge != (Capacity{}) {
		t.Fatalf("room group of the unlimited trainer = %+v, want unlimited with no pledge in the ledger", st)
	}
	if err := w.execSwitch(regHome, trainer, Decision{Role: 0, Cause: CauseReview}); err != nil {
		t.Fatalf("execSwitch() to the limited role = %v, want nil", err)
	}
	if got, want := roomOfAgent(t, w, shipHome).Pledge, (Capacity{Mass: 9, Bulk: 6}); got != want {
		t.Errorf("pledge of the trainer after the change = %+v, want %+v (its trainee's output)", got, want)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() after the change = %v, want nil", err)
	}
}
