package economysim

import "testing"

// forageField is the forage world with the resources module on: gather draws one
// unit of the stock "field" an execution, the stock holds stock units, and only
// the own-use list names gather, so the role's own recipes draw on nothing.
func forageField(stock int64) func(*Config) {
	return func(c *Config) {
		field := DefaultResource()
		field.ID, field.Capacity = "field", 1_000
		c.Modules.Resources = &Resources{
			HorizonYears: 50, Resources: []Resource{field},
			Endowments: []Endowment{{Region: "home", Resource: "field", HasCapacity: true, Capacity: 1_000, HasInitial: true, Initial: stock}},
		}
		c.Recipes[0].Draws = []Draw{{Resource: "field", Quantity: 1}}
	}
}

// TestEndowments_OwnUseDrawsInGroup verifies that the Endowments group of a view
// holds the endowments an own-use recipe of the role draws on, although the
// role's own recipes draw on none (plan 2-10 D413).
//
// Requirement: FR-SUB-036
func TestEndowments_OwnUseDrawsInGroup(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, forageField(100))
	w.refreshViewCtx()
	w.p23Snapshot()
	var v AgentView
	if err := w.fillAgentView(&v, &w.agents[1]); err != nil {
		t.Fatalf("fillAgentView error = %v, want nil", err)
	}
	if len(v.Endowments) != 1 || v.Endowments[0].Stock != 100 {
		t.Errorf("Endowments = %+v, want the one endowment field with stock 100", v.Endowments)
	}
}

// TestWarnings_EndowmentUnusedCountsOwnUse verifies that an endowment only an
// own-use recipe draws on is not reported as unused, and is when the recipe is
// off the list.
//
// Requirement: FR-SUB-036
func TestWarnings_EndowmentUnusedCountsOwnUse(t *testing.T) {
	t.Parallel()
	unused := func(c Config) bool {
		for _, w := range c.Warnings() {
			if w.Rule == "endowment_unused" {
				return true
			}
		}
		return false
	}
	c := forageConfig(1)
	forageField(100)(&c)
	if unused(c) {
		t.Errorf("endowment_unused raised with the own-use recipe drawing on the endowment, want none")
	}
	c.Modules.Subsistence.Recipes = nil
	if !unused(c) {
		t.Errorf("endowment_unused not raised with no recipe drawing on the endowment, want it")
	}
}

// TestRation_OwnUseNoPriority_BlockedCounted verifies that the own-use requests
// enter the ration like any other and the households refused are counted as
// blocked at P2.7 (case b): a stock of 2 against four households asking one
// unit each grants two, and the two others start nothing and are counted.
//
// Requirement: FR-SUB-023, FR-SUB-036
func TestRation_OwnUseNoPriority_BlockedCounted(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := newForage(t, 4, forageField(2), WithObserver(obs))
	if err := w.Step(); err != nil {
		t.Fatalf("Step error = %v, want nil", err)
	}
	if got := of[OwnUseChanged](obs); len(got) != 1 || got[0].Started != 2 || got[0].Blocked != 2 || got[0].Entered != 4 {
		t.Errorf("OwnUseChanged = %+v, want one event with started 2, blocked 2 and entered 4", got)
	}
	for i := 1; i <= 4; i++ {
		if !w.sub.ownUse[i] {
			t.Errorf("household %d left the mode after a blocked tick, want the flag to stay", i)
		}
	}
}

// TestWork_NoInterruption verifies that an agent with a job in progress is not
// offered the mode's request: workAgent returns before the view and Work, so the
// free households of the cell are not counted and no request is recorded.
//
// Requirement: FR-SUB-049
func TestWork_NoInterruption(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, func(c *Config) { c.Recipes[0].Duration = 5 })
	for range 2 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	a := &w.agents[1]
	if !a.hasJob() || !w.sub.ownUse[a.id] {
		t.Fatalf("agent job %v, flag %v, want a running own-use job and the flag", a.hasJob(), w.sub.ownUse[a.id])
	}
	jobs, remaining := w.sub.ownUseJobs[a.id], a.job.remaining
	w.refreshViewCtx()
	w.buildRotation()
	if err := w.workAgent(a, w.regionOf(a)); err != nil {
		t.Fatal(err)
	}
	if w.sub.ownUseJobs[a.id] != jobs || a.job.remaining != remaining || w.pr.asks[a.id].set {
		t.Errorf("a busy agent was offered work: counter %d (was %d), job remaining %d (was %d), request set %v", w.sub.ownUseJobs[a.id], jobs, a.job.remaining, remaining, w.pr.asks[a.id].set)
	}
}

// TestMode_DoesNotTouchHungerPrivation verifies that the mode does not alter how
// Hunger and Privation come from the unfed count: a household flagged and one
// not, with the same unfed count, get the same hunger and privation from the
// same consumption step.
//
// Requirement: FR-SUB-033
func TestMode_DoesNotTouchHungerPrivation(t *testing.T) {
	t.Parallel()
	w := newForage(t, 2, nil)
	w.sub.grow(len(w.agents))
	w.sub.ownUse[1] = true
	for _, i := range []int{1, 2} {
		w.agents[i].unfed = 1
		w.agents[i].hunger = 2
	}
	w.refreshViewCtx()
	if err := w.p61Upkeep(); err != nil {
		t.Fatalf("p61Upkeep error = %v, want nil", err)
	}
	a, b := &w.agents[1], &w.agents[2]
	if a.hunger != b.hunger || a.privation != b.privation {
		t.Errorf("flagged (hunger %d, privation %d) and unflagged (hunger %d, privation %d), want equal", a.hunger, a.privation, b.hunger, b.privation)
	}
}
