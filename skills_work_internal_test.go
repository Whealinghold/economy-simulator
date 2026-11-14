package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// halfYieldConfig is trainingConfig with a trainee yield of 5 000, so the output
// of a trainee is half of an execution's, rounded by the residue of the principal.
func halfYieldConfig() Config {
	c := trainingConfig(3)
	c.Recipes[1].TraineeYield = 5_000
	return c
}

// kneadOf is the recipe ID of knead, which sorts before mill in the canonical order.
func kneadOf(t testing.TB, w *World) RecipeID {
	t.Helper()
	id, ok := w.cfg.recipeID("knead")
	if !ok {
		t.Fatalf("recipe knead is not in the configuration")
	}
	return id
}

// holding returns the units of commodity c an agent holds.
func holding(t testing.TB, w *World, id AgentID, c int) int64 {
	t.Helper()
	n, _, err := w.ledger.Holding(w.agents[id].account, c)
	if err != nil {
		t.Fatalf("Holding(agent %d, commodity %d) error = %v, want nil", id, c, err)
	}
	return n
}

// replayTo27 runs the positions of a step up to the starts of phase 2.
func replayTo27(t testing.TB, w *World) {
	t.Helper()
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep error = %v, want nil", err)
	}
	w.p02()
	if err := w.p11Moment(); err != nil {
		t.Fatalf("p11Moment error = %v, want nil", err)
	}
	w.p12Anchors()
	w.p21Complete()
	w.p22Teaching()
	w.p23Snapshot()
	if err := w.p25Work(); err != nil {
		t.Fatalf("p25Work error = %v, want nil", err)
	}
	w.p26Ration()
	if err := w.p27Start(); err != nil {
		t.Fatalf("p27Start error = %v, want nil", err)
	}
}

// TestBoard_ConsumedFromTrainerAccount verifies that a trainee's upkeep is consumed
// from its trainer's account at P6.1 and not from its own, and that the trainer's
// use includes it (the Board of its view).
//
// Arithmetic: a household of either role eats 1 grain a tick and each agent holds
// 10 (the start stock). The trainer eats its own 1 and its trainee's 1, so it holds
// 10 - 2 = 8; the trainee keeps its 10.
//
// Requirement: FR-SKL-013, FR-AGT-068
func TestBoard_ConsumedFromTrainerAccount(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	trainee, trainer := c.homeMillers[0], c.homeWeaver
	w.skills.start(trainee, trainer, 1, 0, 0)
	own, boss := holding(t, w, trainee, grain), holding(t, w, trainer, grain)
	if err := w.p61Upkeep(); err != nil {
		t.Fatalf("p61Upkeep error = %v, want nil", err)
	}
	if got := holding(t, w, trainee, grain); got != own {
		t.Errorf("trainee grain = %d after P6.1, want %d: its board is the trainer's", got, own)
	}
	if got := holding(t, w, trainer, grain); got != boss-2 {
		t.Errorf("trainer grain = %d after P6.1, want %d (its own upkeep and its trainee's)", got, boss-2)
	}
	if w.agents[trainee].unfed != 0 {
		t.Errorf("trainee unfed = %d, want 0 while boarded", w.agents[trainee].unfed)
	}
	var v AgentView
	if err := w.fillAgentView(&v, &w.agents[trainer], viewWork); err != nil {
		t.Fatalf("fillAgentView error = %v, want nil", err)
	}
	if want := []UpkeepRow{{Commodity: grain, Quantity: 1}}; !slices.Equal(v.Board, want) {
		t.Errorf("Board of the trainer = %v, want %v", v.Board, want)
	}
	tc := tradeCtx{v: &v}
	v.Weight = 1
	if u, isUpkeep, ok := tc.upkeepUnits(grain); u != 2 || !isUpkeep || !ok {
		t.Errorf("upkeepUnits(grain) = %d, %v, %v, want 2, true, true (own 1 and 1 boarded)", u, isUpkeep, ok)
	}
	if u, isUpkeep, _ := tc.upkeepUnits(flour); u != 0 || isUpkeep {
		t.Errorf("upkeepUnits(flour) = %d, %v, want 0, false", u, isUpkeep)
	}
	var other AgentView
	if err := w.fillAgentView(&other, &w.agents[c.homeMillers[1]], viewWork); err != nil || len(other.Board) != 0 {
		t.Errorf("Board of an agent with no trainee = %v, %v, want none", other.Board, err)
	}
}

// TestBoard_WelfareRatioTenThousandAndNettedFromTrainer verifies that the board
// of a trainee is its in-kind receipt K_t, so its record has income equal to its
// cost and its ratio is 10 000, and that the same value is netted from the income of
// its trainer in the tick of consumption.
//
// Arithmetic: 1 grain at the reference price 1 000 is a board of 1 000 for a
// household of weight 1. The trainee's record is income 1 000, cost 1 000; the
// trainer's income is its input-style debit of 1 000 only (no receipts), -1 000.
//
// Requirement: FR-WEL-030, FR-SKL-013
func TestBoard_WelfareRatioTenThousandAndNettedFromTrainer(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	trainee, trainer := c.homeMillers[0], c.homeWeaver
	w.skills.start(trainee, trainer, 1, 0, 0)
	if err := w.p61Upkeep(); err != nil {
		t.Fatalf("p61Upkeep error = %v, want nil", err)
	}
	w.p63Welfare()
	last := func(id AgentID) welfare.Record {
		recs := w.agents[id].history.AppendRecords(nil)
		return recs[len(recs)-1]
	}
	if got := last(trainee); got.Income != 1_000 || got.Cost != 1_000 {
		t.Errorf("trainee record income %d, cost %d, want 1 000 and 1 000 (ratio 10 000)", got.Income, got.Cost)
	}
	if got := last(trainer); got.Income != -1_000 {
		t.Errorf("trainer record income = %d, want -1 000: the board netted as an input", got.Income)
	}
}

// TestTraineeWork_EligibleOnlyExecutorPrincipal verifies that a trainee asks for
// the trainer's trainee-eligible recipe and runs it with the trainer as principal:
// the input is taken from the trainer's account, the output goes to it, the yield is
// TraineeYield, and the trainee's own account and its own role's recipe are not
// touched.
//
// Arithmetic: knead takes 1 grain and makes 3 flour, trainee yield 5 000, one
// execution of duration 1: produced = (1 * 3 * 5 000 + residue) / 10 000, with the
// residue of the trainer (the principal's), which is 1 when the residue is at least
// 5 000 and 1 or 2 by the sum otherwise; the test reads it before the start.
//
// Requirement: FR-SKL-014, FR-PRO-008, FR-PRO-033, FR-PRO-034
func TestTraineeWork_EligibleOnlyExecutorPrincipal(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, halfYieldConfig(), WithSeed(exitSeed), WithAudit(), WithObserver(rec))
	step(t, w, 1)
	c := castOf(t, w)
	trainee, trainer := c.homeMillers[0], c.homeWeaver
	w.skills.start(trainee, trainer, 1, 0, 0)
	off, ok := residueOffset(w.roles.row(1).jobs, &w.recipes, kneadOf(t, w))
	if !ok {
		t.Fatalf("knead is not on the weaver's list")
	}
	residue := w.agents[trainer].residues[off]
	wantUnits := (3*5_000 + residue) / 10_000
	traineeGrain, trainerGrain, trainerFlour := holding(t, w, trainee, grain), holding(t, w, trainer, grain), holding(t, w, trainer, flour)
	before := len(rec.events)
	replayTo27(t, w)
	var ran []RecipeExecuted
	for _, e := range rec.events[before:] {
		if x, ok := e.(RecipeExecuted); ok && x.Agent == trainee {
			ran = append(ran, x)
		}
	}
	if len(ran) != 1 || ran[0].Recipe != kneadOf(t, w) || ran[0].Executions != 1 || ran[0].EffectiveYield != 5_000 {
		t.Fatalf("trainee executions = %+v, want one of knead at yield 5 000", ran)
	}
	if len(ran[0].Outputs) != 1 || int64(ran[0].Outputs[0].Quantity) != wantUnits {
		t.Errorf("trainee output = %+v, want %d flour (residue %d)", ran[0].Outputs, wantUnits, residue)
	}
	if got := holding(t, w, trainee, grain); got != traineeGrain {
		t.Errorf("trainee grain = %d, want %d: the input comes from the trainer", got, traineeGrain)
	}
	// The trainer's own job of the tick takes 1 grain too and makes 3 flour; the
	// trainee's job took 1 more grain and added wantUnits.
	ownJob := int64(0)
	for _, e := range rec.events[before:] {
		if x, ok := e.(RecipeExecuted); ok && x.Agent == trainer {
			ownJob++
		}
	}
	if got, want := holding(t, w, trainer, grain), trainerGrain-1-ownJob; got != want {
		t.Errorf("trainer grain = %d, want %d (the trainee's input and %d own job(s))", got, want, ownJob)
	}
	if got, want := holding(t, w, trainer, flour), trainerFlour+wantUnits+3*ownJob; got != want {
		t.Errorf("trainer flour = %d, want %d (the trainee's %d units and %d own job(s) of 3)", got, want, wantUnits, ownJob)
	}
}

// TestTrainee_KeepsRoleAndRunsNoOwnRecipe verifies that over several ticks a trainee
// keeps its role and runs only the trainer's eligible recipe, never a recipe of its
// own role.
//
// Requirement: FR-SKL-011, FR-SKL-014
func TestTrainee_KeepsRoleAndRunsNoOwnRecipe(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, halfYieldConfig(), WithSeed(exitSeed), WithAudit(), WithObserver(rec))
	step(t, w, 1)
	c := castOf(t, w)
	trainee := c.homeMillers[0]
	w.skills.start(trainee, c.homeWeaver, 1, 0, 0)
	before := len(rec.events)
	step(t, w, 3)
	n := 0
	for _, e := range rec.events[before:] {
		if x, ok := e.(RecipeExecuted); ok && x.Agent == trainee {
			n++
			if x.Recipe != kneadOf(t, w) {
				t.Errorf("trainee ran recipe %d, want only the trainer's eligible recipe knead", x.Recipe)
			}
		}
	}
	if n == 0 {
		t.Errorf("trainee ran no recipe in 3 ticks, want the trainer's eligible one")
	}
	if got := w.agents[trainee].role; got != 0 || !w.skills.isTrainee(trainee) {
		t.Errorf("trainee role %d, trainee %v, want its own role 0 and still a trainee", got, w.skills.isTrainee(trainee))
	}
}

// TestTeaching_BarsStartNotJob verifies the teaching tick: with three trainees at a
// load of 1 000 the accumulator makes three teaching ticks in ten, the trainer
// starts no recipe on a teaching tick and does on any other, and a job it has in
// progress is not touched (P2.1 comes first).
//
// Arithmetic: 3 * 1 000 = 3 000 a tick runs the accumulator 3 000, 6 000, 9 000,
// 12 000 -> 2 000 (teaching, tick 4), 5 000, 8 000, 11 000 -> 1 000 (teaching,
// tick 7), 4 000, 7 000, 10 000 -> 0 (teaching, tick 10).
//
// Requirement: FR-SKL-015
func TestTeaching_BarsStartNotJob(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	for _, m := range c.homeMillers {
		w.skills.start(m, c.homeWeaver, 1, 0, 0)
	}
	n, pattern := 0, make([]bool, 0, 10)
	for range 10 {
		w.p22Teaching()
		teaching := w.skills.teachingTick(c.homeWeaver)
		pattern = append(pattern, teaching)
		if teaching {
			n++
		}
	}
	if want := []bool{false, false, false, true, false, false, true, false, false, true}; !slices.Equal(pattern, want) || n != 3 {
		t.Errorf("teaching ticks over ten = %v (%d), want %v (the 4th, 7th and 10th)", pattern, n, want)
	}
	// A start is barred on a teaching tick and made on another.
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep error = %v, want nil", err)
	}
	w.p21Complete()
	w.p23Snapshot()
	for _, teaching := range []bool{true, false} {
		w.skills.trainer[c.homeWeaver].teaching = teaching
		if err := w.p25Work(); err != nil {
			t.Fatalf("p25Work error = %v, want nil", err)
		}
		if got := w.pr.asks[c.homeWeaver].set; got == teaching {
			t.Errorf("teaching %v: the trainer's request is set = %v, want %v", teaching, got, !teaching)
		}
		if w.pr.asks[c.homeMillers[0]].set != true {
			t.Errorf("teaching %v: the trainee's request is not set, want it admitted whatever the trainer does", teaching)
		}
		w.pr.asks[c.homeWeaver].set, w.pr.asks[c.homeMillers[0]].set = false, false
	}
}

// TestEntryStep_OrderCompleteFillAdvance verifies the order of P7.1(a) of
// spec-skill-barriers.md 4.2.1 and the ticks served: a place filled in the entry step
// of tick s makes the agent a trainee for ticks s + 1 to s + TrainingTicks and it
// completes in the entry step of tick s + TrainingTicks; in one step the completion
// comes first, so the place it frees is filled in the same step, and the events
// follow (TraineeshipCompleted, the RoleChanged it causes, then TraineeshipStarted).
//
// Arithmetic: TrainingTicks is 4, so the agent placed at the first call has served
// 1, 2, 3 and 4 after the first four calls and completes at the fifth, tick s + 4.
//
// Requirement: FR-SKL-015
func TestEntryStep_OrderCompleteFillAdvance(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w, c := trainingWorld(t, 1, WithObserver(rec))
	a, b := c.homeMillers[0], c.homeMillers[1]
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: a, applied: 1})
	for call := 1; call <= 4; call++ {
		if err := w.skillsEntry(0); err != nil {
			t.Fatalf("call %d: skillsEntry error = %v, want nil", call, err)
		}
		if got := w.skills.trainee[a].served; got != int64(call) || !w.skills.isTrainee(a) {
			t.Errorf("after call %d: ticks served = %d, trainee %v, want %d and a trainee", call, got, w.skills.isTrainee(a), call)
		}
	}
	// Fifth call, tick s + 4: a completes, b takes the place a held, in this order.
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: b, applied: 2})
	before := len(rec.events)
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("call 5: skillsEntry error = %v, want nil", err)
	}
	var order []string
	for _, e := range rec.events[before:] {
		switch x := e.(type) {
		case TraineeshipCompleted:
			order = append(order, "completed")
			if x.Agent != a || x.Region != 0 || x.From != 0 || x.To != 1 {
				t.Errorf("TraineeshipCompleted = %+v, want agent %d in region 0 from role 0 to role 1", x, a)
			}
		case RoleChanged:
			order = append(order, "changed:"+string(x.Cause))
		case TraineeshipStarted:
			order = append(order, "started")
		}
	}
	if want := []string{"completed", "changed:traineeship", "started"}; !slices.Equal(order, want) {
		t.Errorf("events of the fifth entry step = %v, want %v", order, want)
	}
	if w.skills.isTrainee(a) || w.agents[a].role != 1 {
		t.Errorf("agent a: trainee %v, role %d, want no trainee and the role it trained for", w.skills.isTrainee(a), w.agents[a].role)
	}
	if rec := w.skills.trainee[b]; !rec.live || rec.served != 1 {
		t.Errorf("agent b = %+v, want a trainee that has served 1 tick, placed and advanced in the step", rec)
	}
}

// TestCompletion_KeepsMoneyAndGoods verifies that a completion keeps the agent's
// account: the money and the goods its new role can trade stay, goods outside the
// new role's tradable set go to the estate whole lots at a time, the counters move
// to the new role and the trainer is left with no trainee.
//
// Arithmetic: the weaver (role 1) trades grain and flour and not the pan, so the
// pan the miller holds is the one lot listed to the estate (GoodsListed 1), and
// its money and 10 grain stay.
//
// Requirement: FR-SKL-011
func TestCompletion_KeepsMoneyAndGoods(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, trainingConfig(3), WithSeed(exitSeed), WithAudit(), WithObserver(rec))
	step(t, w, 1)
	c := castOf(t, w)
	a, trainer := c.homeMillers[0], c.homeWeaver
	w.skills.start(a, trainer, 1, 4, 1)
	acct, err := w.ledger.Account(w.agents[a].account)
	if err != nil {
		t.Fatalf("Account error = %v, want nil", err)
	}
	money, grains, pans := acct.Balance, holding(t, w, a, grain), holding(t, w, a, pan)
	if pans == 0 {
		t.Fatalf("the miller holds no pan, want one to be listed to the estate")
	}
	before := len(rec.events)
	inMiller, inWeaver := w.live.inRole[0], w.live.inRole[1]
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("skillsEntry error = %v, want nil", err)
	}
	acct, _ = w.ledger.Account(w.agents[a].account)
	if acct.Balance != money || holding(t, w, a, grain) != grains || holding(t, w, a, pan) != 0 {
		t.Errorf("after the completion: money %d, grain %d, pans %d, want %d, %d and 0", acct.Balance, holding(t, w, a, grain), holding(t, w, a, pan), money, grains)
	}
	var changed []RoleChanged
	for _, e := range rec.events[before:] {
		if x, ok := e.(RoleChanged); ok {
			changed = append(changed, x)
		}
	}
	if len(changed) != 1 || changed[0].GoodsListed != 1 || changed[0].Cause != CauseTraineeship || changed[0].From != 0 || changed[0].To != 1 {
		t.Errorf("RoleChanged = %+v, want one from role 0 to 1 by traineeship listing 1 lot", changed)
	}
	if w.live.inRole[0] != inMiller-1 || w.live.inRole[1] != inWeaver+1 {
		t.Errorf("agents per role = %d, %d, want %d, %d", w.live.inRole[0], w.live.inRole[1], inMiller-1, inWeaver+1)
	}
	if w.skills.isTrainer(trainer) || w.skills.trainer[trainer].acc != 0 {
		t.Errorf("trainer = %+v, want no trainee and a zero accumulator", w.skills.trainer[trainer])
	}
}

// TestCompletion_AbandonsJobRoleChanged verifies that a job in progress at the
// completion is abandoned with role_changed before TraineeshipCompleted, which comes
// before the RoleChanged it causes (spec-observability.md 4.4.8), and moves nothing.
//
// Requirement: FR-SKL-011, FR-PRO-018, FR-OBS-020
func TestCompletion_AbandonsJobRoleChanged(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w, c := trainingWorld(t, 3, WithObserver(rec))
	a, trainer := c.homeMillers[0], c.homeWeaver
	w.skills.start(a, trainer, 1, 4, 1)
	w.agents[a].job = job{recipe: kneadOf(t, w), remaining: 2, executions: 1, principal: trainer, yield: 5_000, pending: []int64{1}}
	before := len(rec.events)
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("skillsEntry error = %v, want nil", err)
	}
	var order []string
	for _, e := range rec.events[before:] {
		switch x := e.(type) {
		case RecipeAbandoned:
			order = append(order, "abandoned:"+string(x.Reason))
		case TraineeshipCompleted:
			order = append(order, "completed")
		case RoleChanged:
			order = append(order, "changed")
		}
	}
	if want := []string{"abandoned:role_changed", "completed", "changed"}; !slices.Equal(order, want) {
		t.Errorf("events = %v, want %v", order, want)
	}
	if w.agents[a].hasJob() {
		t.Errorf("the trainee still has a job after the completion, want none")
	}
}

// TestCompletion_HeldWithoutHeadroom verifies the held completion: a trainee at
// TrainingTicks whose target role has no headroom stays a trainee, keeps its ticks
// served at TrainingTicks and its place, and completes at the first step with
// headroom; held trainees complete in the order of application tick and then agent
// ID, each taking the headroom the one before left.
//
// Arithmetic: the planted headroom is 1 and falls by 1 with each completion, so of
// two held trainees only the one that applied first (tick 2, though the higher ID)
// completes in the first step and the other in the second.
//
// Requirement: FR-SKL-015
func TestCompletion_HeldWithoutHeadroom(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	first, second := c.homeMillers[1], c.homeMillers[0] // first applied at tick 2, second at tick 5
	w.skills.start(second, c.homeWeaver, 1, 4, 5)
	w.skills.start(first, c.homeWeaver, 1, 4, 2)
	room := int64(0)
	w.skills.headroom = func(RegionID, RoleID) (int64, bool) { return room, false }
	step := func() {
		if err := w.skillsEntry(0); err != nil {
			t.Fatalf("skillsEntry error = %v, want nil", err)
		}
	}
	step()
	if !w.skills.isTrainee(first) || !w.skills.isTrainee(second) || w.skills.trainee[first].served != 4 {
		t.Errorf("with no headroom: trainees %v, %v served %d, want both held at 4", w.skills.isTrainee(first), w.skills.isTrainee(second), w.skills.trainee[first].served)
	}
	room = 1
	w.skills.headroom = func(RegionID, RoleID) (int64, bool) { return room - int64(w.live.inRole[1]-1), false } // the role gains a household per completion
	step()
	if w.skills.isTrainee(first) || !w.skills.isTrainee(second) {
		t.Errorf("with headroom 1: trainee first %v, second %v, want first completed (applied earlier) and second held", w.skills.isTrainee(first), w.skills.isTrainee(second))
	}
	room = 2
	step()
	if w.skills.isTrainee(second) {
		t.Errorf("with headroom again: second still a trainee, want completed")
	}
}

// TestObservedPremium_ExcludesTrainees verifies that a trainee does not count among
// the target role's agents: the observed wage of the target role, and the agents it
// is counted over, are the same with and without a trainee in training for it.
//
// Requirement: FR-SKL-011
func TestObservedPremium_ExcludesTrainees(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	w.agents[c.homeWeaver].history.Push(welfare.Record{Income: 4_000, Working: true, WorkHH: 1})
	before := *w.buildRegionView(0)
	w.skills.start(c.homeMillers[0], c.homeWeaver, 1, 0, 0)
	after := *w.buildRegionView(0)
	if before.Roles[1].Agents != after.Roles[1].Agents || before.Roles[1].Wage != after.Roles[1].Wage || before.Roles[1].Households != after.Roles[1].Households {
		t.Errorf("target role summary with a trainee = %+v, without = %+v, want the same agents, households and wage", after.Roles[1], before.Roles[1])
	}
	if !after.Agents[0].InTraineeship {
		t.Errorf("the trainee's view row has InTraineeship %v, want true", after.Agents[0].InTraineeship)
	}
}

// TestBoard_FilledWithSubsistenceOn verifies that the Board of a trainer's view is
// filled when the subsistence module is on too: the module's view fill returns the
// view, so the board is set before it.
//
// Requirement: FR-AGT-068
func TestBoard_FilledWithSubsistenceOn(t *testing.T) {
	t.Parallel()
	cfg := trainingConfig(3)
	sub := DefaultSubsistence()
	cfg.Modules.Subsistence = &sub
	w := newMill(t, cfg, WithSeed(exitSeed), WithAudit())
	step(t, w, 1)
	c := castOf(t, w)
	w.skills.start(c.homeMillers[0], c.homeWeaver, 1, 0, 0)
	var v AgentView
	if err := w.fillAgentView(&v, &w.agents[c.homeWeaver], viewWork); err != nil {
		t.Fatalf("fillAgentView error = %v, want nil", err)
	}
	if want := []UpkeepRow{{Commodity: grain, Quantity: 1}}; !slices.Equal(v.Board, want) {
		t.Errorf("Board of the trainer with subsistence on = %v, want %v", v.Board, want)
	}
}
