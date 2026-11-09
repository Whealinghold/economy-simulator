package economysim

import (
	"errors"
	"slices"
	"testing"
)

// The tests of the life of a journey (plan 2-14 Phase 5): the guard, the arrival,
// the skip of the strategy loops, the spawner's refusal, the removal and the
// cancel. A journey is planted through the module's own depart, as the pass will,
// because the pass is Phase 6. The fixture is infoConfig (three regions a, b and c;
// agents 3 and 4 in a, 5 and 6 in b, 7 and 8 in c) with the module on.

// migWorld builds infoConfig with the migration module on at a speed of 40 and
// the audit on, changed by mut, and returns the world, the recording strategy and
// the events.
func migWorld(t *testing.T, mut func(c *Config)) (*World, *infoScript, *evLog) {
	t.Helper()
	c := infoConfig()
	m := DefaultMigration()
	m.MoveSpeed = 40
	c.Modules.Migration = &m
	if mut != nil {
		mut(&c)
	}
	s, rec := &infoScript{}, &evLog{}
	w := newMill(t, c, WithStrategy("script", s), WithAudit(), WithObserver(rec))
	return w, s, rec
}

// regionNow returns the region of the agent's account.
func regionNow(w *World, id AgentID) RegionID { return w.regionOf(&w.agents[id]) }

// TestRefusal_OnJourneyBlocksRelocate verifies that an agent with a journey is
// refused a relocation by the obligation on_journey: the guarded path returns an
// ErrRegionBound that names it and changes nothing, and the query that skips it,
// by name or by the instance key of the agent, passes. An agent without a journey
// is not refused.
//
// Requirement: FR-MIG-047, FR-REG-047, FR-TRN-050
func TestRefusal_OnJourneyBlocksRelocate(t *testing.T) {
	t.Parallel()
	w, _, _ := migWorld(t, nil)
	w.migration.depart(3, infoB, 5)
	err := w.relocate(3, infoB, CauseConsumer)
	var rb *RegionBoundError
	if !errors.Is(err, ErrRegionBound) || !errors.As(err, &rb) || rb.Obligation != "on_journey" {
		t.Fatalf("relocate(3, b) = %v, want ErrRegionBound naming on_journey", err)
	}
	if got := regionNow(w, 3); got != infoA {
		t.Errorf("region of agent 3 after the refused relocation = %d, want %d: nothing changes", got, infoA)
	}
	for _, skip := range [][]string{{"on_journey"}, {"on_journey:3"}} {
		if reason, err := w.RelocationRefusal(3, infoB, skip...); err != nil || reason != "" {
			t.Errorf("RelocationRefusal(3, b, %v) = %q, %v, want no refusal", skip, reason, err)
		}
	}
	if reason, err := w.RelocationRefusal(3, infoB, "on_journey:4"); !errors.Is(err, ErrRegionBound) || reason != string(ReasonRegionBound) {
		t.Errorf("RelocationRefusal(3, b, on_journey:4) = %q, %v, want the refusal: the key is another agent's", reason, err)
	}
	if reason, err := w.RelocationRefusal(4, infoB); err != nil || reason != "" {
		t.Errorf("RelocationRefusal(4, b) = %q, %v, want none: agent 4 has no journey", reason, err)
	}
}

// TestArrival_RelocatesAndReseeds verifies the arrival at P1.5 (spec-migration.md
// 8.4, 8.5): at the planted tick and not before the account is in the destination
// with the cause migration and one AgentRelocated, the beliefs are those of a fresh
// seeding at the destination, RoleSince is the tick, the record is gone, and the
// role, weight, strategy and money are unchanged.
//
// Requirement: FR-MIG-029, FR-MIG-030, FR-MIG-032, FR-SPN-053
func TestArrival_RelocatesAndReseeds(t *testing.T) {
	t.Parallel()
	w, _, rec := migWorld(t, nil)
	a := &w.agents[3]
	role, weight, strategy, target, residues := a.role, a.weight, a.strategy, a.target, slices.Clone(a.residues)
	held, _ := balance(t, w, 3)
	w.migration.depart(3, infoB, 2)
	step(t, w, 2)
	if got := regionNow(w, 3); got != infoA || !w.migration.on(3) {
		t.Fatalf("after two steps agent 3 is in region %d, journey %v, want region %d and the journey: it arrives at tick 2", got, w.migration.on(3), infoA)
	}
	step(t, w, 1)
	if got := regionNow(w, 3); got != infoB || w.migration.on(3) || len(w.migration.active) != 0 {
		t.Errorf("after the arrival agent 3 is in region %d, journey %v, active %v, want region %d and none", got, w.migration.on(3), w.migration.active, infoB)
	}
	moved := of[AgentRelocated](rec)
	if len(moved) != 1 || moved[0].Agent != 3 || moved[0].From != infoA || moved[0].To != infoB || moved[0].Cause != CauseMigration || moved[0].Weight != weight {
		t.Errorf("AgentRelocated = %+v, want one for agent 3 from a to b with the cause migration", moved)
	}
	if a.roleSince != 2 || a.role != role || a.weight != weight || a.strategy != strategy {
		t.Errorf("agent 3 roleSince %d role %d weight %d strategy %d, want 2 %d %d %d", a.roleSince, a.role, a.weight, a.strategy, role, weight, strategy)
	}
	if a.target != target || !slices.Equal(a.residues, residues) || a.hunger != 0 || a.privation != 0 {
		t.Errorf("agent 3 target %d residues %v hunger %d privation %d, want %d %v 0 0: the arrival changes none of them", a.target, a.residues, a.hunger, a.privation, target, residues)
	}
	// The reseed is plan 12's reseedAtArrival: every belief has the width of a fresh seed.
	row := w.roles.row(a.role)
	probe := agent{}
	w.fillSeeds(infoB, row)
	reseedBeliefs(&probe, row, nil, slices.Clone(w.ex.seeds), w.bp)
	for i := range a.beliefs {
		if i >= len(probe.beliefs) || a.beliefs[i].Width < probe.beliefs[i].Width {
			t.Errorf("belief %d width = %d, want at least the fresh width %d", i, a.beliefs[i].Width, probe.beliefs[i].Width)
		}
	}
	if after, _ := balance(t, w, 3); after > held+1_000_000 {
		t.Errorf("money after the arrival = %d, before %d: a journey creates none", after, held)
	}
}

// TestArrival_HeldByObligation verifies that another obligation holds an arrival
// (spec-migration.md 8.4): nothing moves, the households are counted as held in the
// origin's region, the record stays, and the arrival completes in the first tick in
// which the obligation is released.
//
// Requirement: FR-MIG-031
func TestArrival_HeldByObligation(t *testing.T) {
	t.Parallel()
	w, _, _ := migWorld(t, nil)
	hold := true
	w.bounds.add("other", func(a *agent, _ []string) bool { return hold && a.id == 3 })
	w.migration.depart(3, infoB, 0)
	// The position is called alone: the pass of P7.2 reads and clears the count in the tick.
	w.p15Arrivals()
	if regionNow(w, 3) != infoA || !w.migration.on(3) {
		t.Fatalf("held arrival: region %d, journey %v, want region %d and the journey", regionNow(w, 3), w.migration.on(3), infoA)
	}
	if got := w.migration.held[infoA]; got != w.agents[3].weight {
		t.Errorf("held households of region a = %d, want %d", got, w.agents[3].weight)
	}
	hold = false
	w.p15Arrivals()
	if regionNow(w, 3) != infoB || w.migration.on(3) {
		t.Errorf("released arrival: region %d, journey %v, want region %d and none", regionNow(w, 3), w.migration.on(3), infoB)
	}
}

// TestArrival_AscendingAgentID verifies that journeys due in one tick arrive in
// ascending agent ID, whatever the order they were planted in, and that the list of
// agents on a journey is ascending.
//
// Requirement: FR-MIG-029
func TestArrival_AscendingAgentID(t *testing.T) {
	t.Parallel()
	w, _, rec := migWorld(t, nil)
	for _, id := range []AgentID{7, 3, 5} {
		w.migration.depart(id, infoB, 1)
	}
	if !slices.Equal(w.migration.active, []AgentID{3, 5, 7}) {
		t.Fatalf("active = %v, want [3 5 7]", w.migration.active)
	}
	step(t, w, 2)
	var got []AgentID
	for _, e := range of[AgentRelocated](rec) {
		got = append(got, e.Agent)
	}
	// Agent 5 is already in b, so its relocation to b is the same region: refused and held.
	if !slices.Equal(got, []AgentID{3, 7}) {
		t.Errorf("arrivals in order %v, want [3 7] (agent 5 is in b already and is held)", got)
	}
}

// TestArrival_ArrivalTickFixedAtDeparture verifies that the arrival tick is fixed
// when the journey is planted: closing the edge afterwards moves nothing (the
// journey never reads the graph again).
//
// Requirement: FR-MIG-026
func TestArrival_ArrivalTickFixedAtDeparture(t *testing.T) {
	t.Parallel()
	w, _, _ := migWorld(t, nil)
	w.migration.depart(3, infoB, 1)
	if err := w.setEdgeStatuses([]EdgeStatus{{From: infoA, To: infoB, Closed: true, FactorBP: 10_000}}, CauseConsumer); err != nil {
		t.Fatalf("setEdgeStatuses() = %v, want nil", err)
	}
	step(t, w, 2)
	if regionNow(w, 3) != infoB {
		t.Errorf("region of agent 3 after the closure = %d, want %d: the journey was fixed at its departure", regionNow(w, 3), infoB)
	}
}

// TestJourney_NoWorkNoTradeNoOrder verifies that an agent on a journey is given no
// view: Work and Trade are not called for it while it travels, no order of it
// appears, and it is called again in the destination after the arrival.
//
// Requirement: FR-MIG-027, FR-AGT-079, FR-WLD-042, FR-TRN-050
func TestJourney_NoWorkNoTradeNoOrder(t *testing.T) {
	t.Parallel()
	w, s, rec := migWorld(t, nil)
	w.migration.depart(3, infoB, 2)
	step(t, w, 2)
	for _, v := range append(slices.Clone(s.works), s.trades...) {
		if v.agent == 3 {
			t.Errorf("agent 3 was given a view at tick %d in region %d, want none on its journey", v.tick, v.region)
		}
	}
	for _, o := range of[OrderPlaced](rec) {
		if o.Agent == 3 {
			t.Errorf("OrderPlaced for agent 3 at tick %d, want none on its journey", o.Tick)
		}
	}
	step(t, w, 1)
	seen := false
	for _, v := range s.trades {
		seen = seen || (v.agent == 3 && v.region == infoB)
	}
	if !seen {
		t.Errorf("agent 3 was never given a view in region b after its arrival, want one")
	}
}

// TestJourney_LadderReliefPopulationInOrigin verifies that the households of a
// journey stay in the origin region's counts until the arrival (spec-migration.md
// 8.3): the snapshot counts them there, and the counts move at the arrival.
//
// Requirement: FR-MIG-028
func TestJourney_LadderReliefPopulationInOrigin(t *testing.T) {
	t.Parallel()
	w, _, _ := migWorld(t, nil)
	households := func() (a, b int64) {
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		return snap.Regions[infoA].Households, snap.Regions[infoB].Households
	}
	a0, b0 := households()
	w.migration.depart(3, infoB, 1)
	step(t, w, 1)
	if a, b := households(); a != a0 || b != b0 {
		t.Errorf("households a, b during the journey = %d, %d, want %d, %d: they stay in the origin", a, b, a0, b0)
	}
	step(t, w, 1)
	w1 := w.agents[3].weight
	if a, b := households(); a != a0-w1 || b != b0+w1 {
		t.Errorf("households a, b after the arrival = %d, %d, want %d, %d", a, b, a0-w1, b0+w1)
	}
}

// TestSpawner_SwitchRefusedOnJourney verifies spec-spawner.md 4: a switch of an
// agent on a journey is refused with on_journey, a removal and a reduction are
// not, and the placement of an entry chooses no such agent as its host.
//
// Requirement: FR-MIG-028, FR-SPN-054
func TestSpawner_SwitchRefusedOnJourney(t *testing.T) {
	t.Parallel()
	w, _, _ := migWorld(t, nil)
	w.migration.depart(3, infoB, 9)
	other := RoleID(0) // carrier: the roles are in ID order, so miller is 1
	tests := []struct {
		name string
		d    Decision
		want Reason
	}{
		{"a switch", Decision{Kind: DecideSwitch, Agent: 3, Role: other}, ReasonOnJourney},
		{"a switch of an agent at home", Decision{Kind: DecideSwitch, Agent: 4, Role: other}, ""},
		{"a removal", Decision{Kind: DecideRemove, Agent: 3, Cause: CauseStarved}, ""},
		{"a reduction", Decision{Kind: DecideReduce, Agent: 3, Households: 1, Cause: CauseStarved}, ""},
	}
	for _, tt := range tests {
		d := tt.d
		if got := w.refusal(infoA, &d); got != tt.want {
			t.Errorf("refusal(%s) = %q, want %q", tt.name, got, tt.want)
		}
	}
	w.gatherHosts(infoA, w.agents[3].role)
	if slices.Contains(w.ex.hostAt, 3) || !slices.Contains(w.ex.hostAt, 4) {
		t.Errorf("hosts of role miller in a = %v, want agent 4 and not agent 3 (on a journey)", w.ex.hostAt)
	}
}

// TestRemoval_DropsJourney verifies the removal's step 2a (spec-agents.md 3.6,
// FR-AGT-084): the record is deleted, the households are counted as ended in the
// origin's region, the money goes to the origin's pool, and the removal emits
// nothing for the journey.
//
// Requirement: FR-AGT-084, FR-MIG-028
func TestRemoval_DropsJourney(t *testing.T) {
	t.Parallel()
	w, _, rec := migWorld(t, nil)
	w.migration.depart(3, infoB, 9)
	weight := w.agents[3].weight
	money, _ := balance(t, w, 3)
	pool := w.estates.pool(w.ledger, infoA)
	rec.events = nil
	if err := w.removeAgent(&w.agents[3], CauseConsumer); err != nil {
		t.Fatalf("removeAgent() = %v, want nil", err)
	}
	if w.migration.on(3) || len(w.migration.active) != 0 || w.migration.ended[infoA] != weight || w.migration.ended[infoB] != 0 {
		t.Errorf("after the removal on(3) %v, active %v, ended %v, want none, none and %d in region a", w.migration.on(3), w.migration.active, w.migration.ended, weight)
	}
	if got := w.estates.pool(w.ledger, infoA); got != pool+Money(money) {
		t.Errorf("pool of region a = %d, want %d: the holdings go to the origin", got, pool+Money(money))
	}
	for _, e := range rec.events {
		switch e.(type) {
		case AgentBankrupted, EstateOpened:
		default:
			t.Errorf("the removal emitted %T, want only AgentBankrupted and EstateOpened", e)
		}
	}
}

// TestCancel_NoLedgerChange verifies that the cancel of a journey (the seam of
// INT's Relocate) drops the record and counts the households as ended without any
// change of balance, region or ledger total, and that a second cancel does nothing.
//
// Requirement: FR-MIG-048
func TestCancel_NoLedgerChange(t *testing.T) {
	t.Parallel()
	w, _, _ := migWorld(t, nil)
	w.migration.depart(3, infoB, 9)
	before := execLedgerState(w)
	money, _ := balance(t, w, 3)
	if !w.migration.cancelJourney(&w.agents[3], infoA) || w.migration.cancelJourney(&w.agents[3], infoA) {
		t.Errorf("cancelJourney twice = true then false expected: the second finds no journey")
	}
	if w.migration.on(3) || w.migration.ended[infoA] != w.agents[3].weight {
		t.Errorf("after the cancel on(3) %v, ended %v, want none and %d in a", w.migration.on(3), w.migration.ended, w.agents[3].weight)
	}
	if after, _ := balance(t, w, 3); after != money || regionNow(w, 3) != infoA || !slices.Equal(execLedgerState(w), before) {
		t.Errorf("the cancel changed the ledger: balance %d (was %d), region %d", after, money, regionNow(w, 3))
	}
}

// TestJourney_OffStageAndOffModule verifies that offStage holds for an agent on a
// journey, which is how the flag step of the subsistence module and the strategy
// loops skip it (spec-subsistence.md 5.6), and that with the module off the world
// has no state and no obligation.
//
// Requirement: FR-SUB-010, FR-TRN-050
func TestJourney_OffStageAndOffModule(t *testing.T) {
	t.Parallel()
	w, _, _ := migWorld(t, nil)
	w.migration.depart(3, infoB, 9)
	if !w.offStage(&w.agents[3]) || w.offStage(&w.agents[4]) {
		t.Errorf("offStage of agents 3 and 4 = %v, %v, want true and false", w.offStage(&w.agents[3]), w.offStage(&w.agents[4]))
	}
	off := newMill(t, infoConfig(), WithStrategy("script", &infoScript{}))
	if off.migration != nil || off.onJourney(&off.agents[3]) || off.offStage(&off.agents[3]) {
		t.Errorf("a world with the module off has migration %v, want nil and no agent off stage", off.migration)
	}
	if _, err := off.RelocationRefusal(3, infoB); err != nil {
		t.Errorf("RelocationRefusal(3, b) with the module off = %v, want none", err)
	}
}

// TestPositions_P15P72 verifies that P1.5 and P7.2 are built positions of the
// module, in the straight line, and no longer listed as not built.
//
// Requirement: FR-WLD-077
func TestPositions_P15P72(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"P1.5", "P7.2"} {
		found := slices.ContainsFunc(positionList[:], func(p position) bool { return p.id == id && p.owner == "MIG" && p.where == line })
		if !found || slices.Contains(notBuiltPositions[:], id) {
			t.Errorf("position %s built as MIG line = %v, listed not built = %v, want true and false", id, found, slices.Contains(notBuiltPositions[:], id))
		}
	}
}
