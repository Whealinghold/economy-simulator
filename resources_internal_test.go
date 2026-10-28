package economysim

import (
	"cmp"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// oreMill is millConfig with the resources module on: one finite stock "ore" in
// home, holding stock units, of which each execution of the mill draws 3. A mill
// of weight 1 starts one execution a tick, so households agents ask 3 units each.
func oreMill(duration, households, stock int64) Config {
	c := millConfig(duration, households)
	ore := DefaultResource()
	ore.ID, ore.Capacity = "ore", 1_000
	c.Modules.Resources = &Resources{
		HorizonYears: 50, Resources: []Resource{ore},
		Endowments: []Endowment{{Region: "home", Resource: "ore", HasCapacity: true, Capacity: 1_000, HasInitial: true, Initial: stock}},
	}
	c.Recipes[0].Draws = []Draw{{Resource: "ore", Quantity: 3}}
	return c
}

// startedBy counts the agents of w that hold a job.
func startedBy(w *World) (n int64) {
	for i := range w.agents {
		if w.agents[i].hasJob() {
			n++
		}
	}
	return n
}

// TestStart_GrantCapsExecutions verifies that a scarce stock caps the starts by
// the ration and that the households refused are counted under no_endowment:
// stock 20 against ten agents asking 3 units each.
//
// Arithmetic: U = 10 * 3 = 30 > A = 20, so each floor share is 1 * 20 / 30 = 0
// with remainder 20; the 20 units left go, 3 at a time, to the agents first in the
// order of the key, 6 of them (18 units), and the 2 left are less than a draw.
// So 6 agents start, 4 are granted 0, 18 units are drawn, 2 remain, and the count
// of RecipeRefused is 4 households under no_endowment.
//
// Requirement: FR-RES-016, FR-RES-017, FR-RES-022, FR-RES-026
func TestStart_GrantCapsExecutions(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := newMill(t, oreMill(5, 10, 20), WithObserver(obs))
	// The six agents with the smallest keys of the tick get the leftover executions
	// (all remainders are equal): agents 1 to 10, the key of (tick 0, agent).
	byKey := []AgentID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	slices.SortFunc(byKey, func(a, b AgentID) int { return cmp.Compare(ordering.Key(0, int64(a)), ordering.Key(0, int64(b))) })
	tickInspecting(t, w, func() {
		r := w.res
		for _, q := range r.reqs {
			if won := slices.Contains(byKey[:6], q.agent); (q.grant == 1) != won {
				t.Errorf("agent %d: grant = %d, want %d (the six smallest keys win)", q.agent, q.grant, map[bool]int64{true: 1, false: 0}[won])
			}
		}
		got := []int64{r.requested[0], r.granted[0], r.drawn[0], r.denied[0], r.reduced[0]}
		want := []int64{30, 18, 18, 4, 0}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("counts [requested granted drawn denied reduced] = %v, want %v", got, want)
				break
			}
		}
	})
	if n := startedBy(w); n != 6 {
		t.Errorf("agents with a job = %d, want 6 (the ration granted 6 executions)", n)
	}
	for _, id := range byKey[:6] {
		if !w.agents[id].hasJob() {
			t.Errorf("agent %d has no job, want one: the ration, not the rotation, decides who starts", id)
		}
	}
	if got := w.Endowment(0, 0); got != 2 {
		t.Errorf("Endowment(home, ore) = %d, want 2 (20 - 6 * 3)", got)
	}
	refused := of[RecipeRefused](obs)
	if len(refused) != 1 || refused[0].NoEndowment != 4 || refused[0].NoInput != 0 {
		t.Errorf("RecipeRefused = %+v, want one row with NoEndowment 4", refused)
	}
	if r := w.res; r.requested[0] != 0 || len(r.reqs) != 0 || len(r.order) != 0 || len(r.ration) != 0 {
		t.Errorf("after the tick requested = %d with %d, %d and %d listed, want all clear: P8.6 clears the counts and the lists",
			r.requested[0], len(r.reqs), len(r.order), len(r.ration))
	}
}

// TestStart_NoEndowmentComesAfterNoInput verifies the place of the draw in the
// limit list of a start (spec-production.md 4.2.1): its limit is the last, after
// the input's, and with an input at hand a cap of 0 is refused no_endowment.
//
// Requirement: FR-RES-022
func TestStart_NoEndowmentComesAfterNoInput(t *testing.T) {
	t.Parallel()
	w := newMill(t, oreMill(1, 10, 20))
	a := &w.agents[1]
	q := startReq{recipe: 0, account: a.account, principal: a.id, weight: a.weight, effective: 10_000, hasProfile: false, capped: true, drawCap: 0}
	if err := w.prod.limitList(w.ledger, &q); err != nil {
		t.Fatalf("limitList() error = %v, want nil", err)
	}
	last := w.prod.limits[len(w.prod.limits)-1]
	if last.Reason != limitNoEndowment || len(w.prod.limits) < 2 || w.prod.limits[len(w.prod.limits)-2].Reason != limitNoInput {
		t.Errorf("limits = %+v, want the draw's limit last, after the input's", w.prod.limits)
	}
	k, reason, err := w.prod.maxExecutions(w.ledger, &q)
	if err != nil || k != 0 || reason != ReasonNoEndowment {
		t.Errorf("maxExecutions(cap 0) = %d, %q, %v, want 0, no_endowment, nil", k, reason, err)
	}
	q.capped = false
	if k, _, _ := w.prod.maxExecutions(w.ledger, &q); k != 1 {
		t.Errorf("maxExecutions(uncapped) = %d, want 1: a request that ignores draws is not changed", k)
	}
}

// TestSnapshot_FixedForTheTick verifies that the requests of P2.5 read the
// snapshot of P2.3 and not the live stock: with the stock emptied after the
// snapshot, as an earlier start would, every agent still asks for its execution,
// and the view's bound is the snapshot's, so the results of Work do not depend on
// the order of the calls.
//
// Requirement: FR-RES-015
func TestSnapshot_FixedForTheTick(t *testing.T) {
	t.Parallel()
	w := newMill(t, oreMill(5, 10, 20))
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep() error = %v, want nil", err)
	}
	w.p02()
	if err := w.p11Moment(); err != nil {
		t.Fatalf("p11Moment() error = %v, want nil", err)
	}
	w.p12Anchors()
	if err := w.p16Relief(); err != nil {
		t.Fatalf("p16Relief() error = %v, want nil", err)
	}
	w.p17Flows()
	w.p21Complete()
	w.p23Snapshot()
	w.res.stock[0] = 0
	if got, _ := w.res.viewCap(0, 0); got != 20/3 {
		t.Errorf("viewCap = %d, want %d: floor(snapshot 20 / draw 3)", got, 20/3)
	}
	if err := w.p25Work(); err != nil {
		t.Fatalf("p25Work() error = %v, want nil", err)
	}
	if got := len(w.res.reqs); got != 10 {
		t.Errorf("requests recorded = %d, want 10: Work ignores the live stock", got)
	}
	w.p26Ration()
	if got := w.res.granted[0]; got != 18 {
		t.Errorf("granted = %d, want 18: the ration reads the snapshot of 20", got)
	}
}

// TestStart_AllOrNothing verifies that a stock lowered between the ration and the
// starts refuses the start with no_endowment and takes nothing: the cap of the
// start is held to the live stock.
//
// Requirement: FR-RES-018
func TestStart_AllOrNothing(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := newMill(t, oreMill(5, 10, 100), WithObserver(obs))
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep() error = %v, want nil", err)
	}
	w.p02()
	if err := w.p11Moment(); err != nil {
		t.Fatalf("p11Moment() error = %v, want nil", err)
	}
	w.p12Anchors()
	if err := w.p16Relief(); err != nil {
		t.Fatalf("p16Relief() error = %v, want nil", err)
	}
	w.p17Flows()
	w.p21Complete()
	w.p23Snapshot()
	if err := w.p25Work(); err != nil {
		t.Fatalf("p25Work() error = %v, want nil", err)
	}
	w.p26Ration()
	w.res.stock[0] = 2 // below one draw of 3
	if err := w.p27Start(); err != nil {
		t.Fatalf("p27Start() error = %v, want nil", err)
	}
	if n := startedBy(w); n != 0 {
		t.Errorf("agents with a job = %d, want 0: no start fits 2 units", n)
	}
	if got, drawn := w.res.stock[0], w.res.drawn[0]; got != 2 || drawn != 0 {
		t.Errorf("stock, drawn = %d, %d, want 2, 0: nothing is taken", got, drawn)
	}
	w.p28aHalted()
	if r := of[RecipeRefused](obs); len(r) != 1 || r[0].NoEndowment != 10 {
		t.Errorf("RecipeRefused = %+v, want one row with NoEndowment 10", r)
	}
}

// TestTake_FailsPastStock verifies that a take of more than the stock, and of
// fewer than one unit, is refused with ErrInsufficientEndowment and changes
// nothing.
//
// Requirement: FR-RES-016
func TestTake_FailsPastStock(t *testing.T) {
	t.Parallel()
	w := newMill(t, oreMill(1, 10, 20))
	for _, n := range []int64{21, 0, -1} {
		if err := w.res.takeEndowment(0, n); !errors.Is(err, ErrInsufficientEndowment) {
			t.Errorf("takeEndowment(%d of 20) error = %v, want ErrInsufficientEndowment", n, err)
		}
	}
	if err := w.res.takeEndowment(0, 20); err != nil || w.res.stock[0] != 0 {
		t.Errorf("takeEndowment(20 of 20) = %v with %d left, want nil and 0: a take of the whole stock fits", err, w.res.stock[0])
	}
}

// TestEndowment_Read verifies the public read: the stock of a pair, 0 for an
// unknown region or resource, 0 with the module off, and the same on a failed
// world.
//
// Requirement: FR-RES-005, FR-RES-006
func TestEndowment_Read(t *testing.T) {
	t.Parallel()
	w := newMill(t, oreMill(1, 10, 20))
	for _, tt := range []struct {
		name     string
		region   RegionID
		resource ResourceID
		want     int64
	}{
		{"the pair", 0, 0, 20}, {"unknown region", 1, 0, 0}, {"negative region", -1, 0, 0},
		{"unknown resource", 0, 1, 0}, {"negative resource", 0, -1, 0},
	} {
		if got := w.Endowment(tt.region, tt.resource); got != tt.want {
			t.Errorf("Endowment(%s: %d, %d) = %d, want %d", tt.name, tt.region, tt.resource, got, tt.want)
		}
	}
	w.failed = errors.New("failed for the test")
	if got := w.Endowment(0, 0); got != 20 {
		t.Errorf("Endowment on a failed world = %d, want 20: it is a read", got)
	}
	if got := newMill(t, millConfig(1, 10)).Endowment(0, 0); got != 0 {
		t.Errorf("Endowment with the module off = %d, want 0", got)
	}
	// A pair with no capacity has a stock of 0 whatever the resource's start.
	c := oreMill(1, 10, 20)
	c.Modules.Resources.Endowments, c.Modules.Resources.Resources[0].Capacity = nil, 0
	if got := newMill(t, c).Endowment(0, 0); got != 0 {
		t.Errorf("Endowment of a pair with no row = %d, want 0: the default capacity is 0", got)
	}
	// The starting stock is InitialBP of the capacity, truncated, when the row gives none.
	c = oreMill(1, 10, 20)
	c.Modules.Resources.Resources[0].InitialBP = 3_333
	c.Modules.Resources.Endowments[0].HasInitial = false
	if got := newMill(t, c).Endowment(0, 0); got != 333 {
		t.Errorf("Endowment from InitialBP 3 333 of 1 000 = %d, want 333 (1 000 * 3 333 / 10 000 truncated)", got)
	}
}

// TestDraw_NotReturnedOnAbandon verifies that a job abandoned by the removal of
// its agent does not give its draw back: the units were drawn at the start.
//
// Requirement: FR-RES-023
func TestDraw_NotReturnedOnAbandon(t *testing.T) {
	t.Parallel()
	w := newMill(t, oreMill(5, 10, 100))
	step(t, w, 1)
	if got := w.Endowment(0, 0); got != 70 {
		t.Fatalf("Endowment after ten starts = %d, want 70 (100 - 10 * 3)", got)
	}
	var job *agent
	for i := range w.agents {
		if w.agents[i].hasJob() {
			job = &w.agents[i]
			break
		}
	}
	if err := w.removeAgent(job, CauseStarved); err != nil {
		t.Fatalf("removeAgent() error = %v, want nil", err)
	}
	if got := w.Endowment(0, 0); got != 70 {
		t.Errorf("Endowment after the abandon = %d, want 70: the draw is not returned", got)
	}
}

// TestDraw_RegionChangeKeepsDraw verifies that a job keeps the draw it took from
// the region it started in when its agent moves: the other region's stock is
// untouched and the job goes on.
//
// Requirement: FR-RES-023
func TestDraw_RegionChangeKeepsDraw(t *testing.T) {
	t.Parallel()
	c := oreMill(5, 10, 100)
	yonder := DefaultRegion()
	yonder.ID = "yonder"
	c.Regions = append(c.Regions, yonder)
	c.Modules.Resources.Endowments = append(c.Modules.Resources.Endowments,
		Endowment{Region: "yonder", Resource: "ore", HasCapacity: true, Capacity: 1_000, HasInitial: true, Initial: 50})
	w := newMill(t, c)
	step(t, w, 1)
	var job AgentID
	for i := range w.agents {
		if w.agents[i].hasJob() {
			job = AgentID(i)
			break
		}
	}
	if err := w.relocate(job, 1, CauseConsumer); err != nil {
		t.Fatalf("relocate() error = %v, want nil", err)
	}
	if home, away := w.Endowment(0, 0), w.Endowment(1, 0); home != 70 || away != 50 {
		t.Errorf("Endowment home, yonder = %d, %d, want 70, 50: the draw stays with the region of the start", home, away)
	}
	if !w.agents[job].hasJob() {
		t.Errorf("the job of agent %d is gone after the move, want it to go on", job)
	}
}

// TestPositions_Table verifies the places of the resources module's positions in
// the straight line (spec-world.md 6.4): P2.3 after P2.1, P2.6 after P2.5, both
// owned by RES.
//
// Requirement: FR-RES-007, FR-RES-015
func TestPositions_Table(t *testing.T) {
	t.Parallel()
	var ids []string
	for _, p := range positionList {
		switch p.id {
		case "P2.1", "P2.3", "P2.5", "P2.6", "P2.7":
			ids = append(ids, p.id)
			if (p.id == "P2.3" || p.id == "P2.6") && (p.owner != "RES" || p.where != line) {
				t.Errorf("position %s = %+v, want owner RES in the straight line", p.id, p)
			}
		}
	}
	if want := "P2.1 P2.3 P2.5 P2.6 P2.7"; len(ids) != 5 || ids[0]+" "+ids[1]+" "+ids[2]+" "+ids[3]+" "+ids[4] != want {
		t.Errorf("positions of phase 2 = %v, want %s", ids, want)
	}
}

// TestCohortGetsProportionalShare verifies, through the world, that the ration is
// weight-invariant: twenty cohorts of weight 50 and a thousand agents of weight 1
// with the same total demand get the same total, and a cohort gets its
// proportional share.
//
// Arithmetic: stock 1 500 against 1 000 executions of 3 units, U = 3 000. The
// proportional grant is w * A / U = w / 2, so a cohort asking 50 gets 50 * 1 500 /
// 3 000 = 25 exactly (no remainder) and the thousand agents of weight 1 get 500
// between them: 20 * 25 = 500 executions in both worlds, 1 500 units drawn.
//
// Requirement: FR-RES-021
func TestCohortGetsProportionalShare(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name         string
		scale, agent int64
	}{{"a thousand agents of weight 1", 1, 1_000}, {"twenty cohorts of weight 50", 50, 1}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := oreMill(1, 1_000, 1_500)
			c.Modules.Resources.Endowments[0].Capacity = 5_000
			c.World.Scale, c.World.MinAgents = tt.scale, tt.agent
			w := newMill(t, c)
			var perCohort []int64
			tickInspecting(t, w, func() {
				if w.res.drawn[0] != 1_500 {
					t.Errorf("units drawn = %d, want 1500", w.res.drawn[0])
				}
				for _, q := range w.res.reqs {
					if tt.scale == 50 {
						perCohort = append(perCohort, q.grant)
					}
				}
			})
			for _, g := range perCohort {
				if g != 25 {
					t.Errorf("grant of a cohort asking 50 = %d, want 25 (its proportional share)", g)
				}
			}
			if tt.scale == 50 && len(perCohort) != 20 {
				t.Errorf("cohorts = %d, want 20", len(perCohort))
			}
		})
	}
}

// TestRecipeRefused_CountsNoEndowment verifies that the refusal tally counts the
// new reason in its own column and flushes it as the NoEndowment count of the
// event, and that no other count moves.
//
// Requirement: FR-RES-022, FR-RES-026
func TestRecipeRefused_CountsNoEndowment(t *testing.T) {
	t.Parallel()
	tally := newRefusalTally(1, 2)
	tally.add(0, 1, ReasonNoEndowment, 7)
	tally.add(0, 1, ReasonNoInput, 2)
	var got []RecipeRefused
	tally.flush(5, func(e RecipeRefused) { got = append(got, e) })
	want := RecipeRefused{Tick: 5, Region: 0, Recipe: 1, NoInput: 2, NoEndowment: 7}
	if len(got) != 1 || got[0] != want {
		t.Errorf("flush = %+v, want [%+v]", got, want)
	}
}

// TestLedgerUntouched verifies that a draw is not a ledger flow (spec-resources.md
// 6.2.1): with draws in play the ledger opens the same accounts and records the
// same reasons as the world of the same chain with the module off, and audits.
//
// Requirement: FR-RES-024
func TestLedgerUntouched(t *testing.T) {
	t.Parallel()
	on := newMill(t, oreMill(1, 10, 1_000), WithAudit())
	off := newMill(t, millConfig(1, 10), WithAudit())
	step(t, on, 5)
	step(t, off, 5)
	if on.ledger.Len() != off.ledger.Len() {
		t.Errorf("accounts = %d with the module on, %d off, want equal", on.ledger.Len(), off.ledger.Len())
	}
	reasons := func(w *World) (out []string) {
		for _, f := range w.ledger.AppendFlows(nil, true) {
			out = append(out, string(f.Reason))
		}
		return out
	}
	if a, b := reasons(on), reasons(off); len(a) != len(b) {
		t.Errorf("flow reasons = %v with the module on, %v off, want the same set", a, b)
	}
	if err := on.ledger.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestNoRandomWord verifies that the module draws no random word: a world with
// the module on and a resource that no recipe draws on steps exactly as the world
// with the module off, in every snapshot, so the keyed streams are unchanged.
//
// Requirement: FR-RES-045
func TestNoRandomWord(t *testing.T) {
	t.Parallel()
	c := oreMill(2, 10, 1_000)
	c.Recipes[0].Draws = nil
	on := newMill(t, c, WithSeed(9))
	off := newMill(t, millConfig(2, 10), WithSeed(9))
	for i := range 12 {
		step(t, on, 1)
		step(t, off, 1)
		a, errA := on.Snapshot()
		b, errB := off.Snapshot()
		// The hash and the list of modules are what the module is.
		a.ConfigHash, a.Modules = b.ConfigHash, b.Modules
		// The rows and the names of the resources are the module's too.
		a.Endowments, a.Names.Resources = b.Endowments, b.Names.Resources
		if errA != nil || errB != nil || !reflect.DeepEqual(a, b) {
			t.Fatalf("seed 9: the snapshots differ at tick %d with the module on and no draw, want equal", i)
		}
	}
}

// TestView_OwnRegionEndowmentsOnly verifies the Endowments group of the view
// (spec-resources.md 5.5.1): the endowments that the agent's role draws on, in its
// own region, with the stock now, the capacity and the abundance of the tick; the
// other region's stock is in no view; and the view of an agent whose role draws on
// nothing has the group empty.
//
// Requirement: FR-RES-041
func TestView_OwnRegionEndowmentsOnly(t *testing.T) {
	t.Parallel()
	c := oreMill(1, 10, 20)
	yonder := DefaultRegion()
	yonder.ID = "yonder"
	c.Regions = append(c.Regions, yonder)
	c.Modules.Resources.Endowments = append(c.Modules.Resources.Endowments,
		Endowment{Region: "yonder", Resource: "ore", HasCapacity: true, Capacity: 1_000, HasInitial: true, Initial: 50})
	c.Modules.Resources.Resources[0].Abundance = []AbundanceKnot{{Share: 0, Factor: 2_000}, {Share: 10_000, Factor: 10_000}}
	w := newMill(t, c)
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep() error = %v, want nil", err)
	}
	w.p02()
	if err := w.p11Moment(); err != nil {
		t.Fatalf("p11Moment() error = %v, want nil", err)
	}
	w.p12Anchors()
	w.p21Complete()
	w.p23Snapshot()
	w.refreshViewCtx()
	var v AgentView
	if err := w.fillAgentView(&v, &w.agents[2]); err != nil {
		t.Fatalf("fillAgentView() error = %v, want nil", err)
	}
	// Share 20 * 10 000 / 1 000 = 200; factor 2 000 + 8 000 * 200 / 10 000 = 2 160.
	want := []EndowmentView{{Resource: 0, Stock: 20, Capacity: 1_000, AbundanceBP: 2_160}}
	if !reflect.DeepEqual(v.Endowments, want) {
		t.Errorf("view of a home miller: Endowments = %+v, want %+v", v.Endowments, want)
	}
	// A role whose recipes draw on nothing sees none.
	w.roles.rows[0].recipes = nil
	w.res.roleRes[0] = nil
	if err := w.fillAgentView(&v, &w.agents[2]); err != nil || len(v.Endowments) != 0 {
		t.Errorf("view of a role with no draw: Endowments = %+v, error %v, want none", v.Endowments, err)
	}
}

// TestEffectiveYield_IncludesAbundance verifies that the cost anchor of P1.2, the
// view of P2.5 and the start of P2.7 read one effective yield with the abundance
// in it (spec-production.md 6.3, FR-RES-048).
//
// Arithmetic: a stock of 500 of 1 000 is a share of 5 000; the curve (0, 2 000),
// (10 000, 10 000) gives 2 000 + 8 000 * 5 000 / 10 000 = 6 000, so a non-sited
// recipe (site 10 000) has an effective yield of 10 000 * 6 000 / 10 000 = 6 000.
//
// Requirement: FR-RES-048, FR-RES-014
func TestEffectiveYield_IncludesAbundance(t *testing.T) {
	t.Parallel()
	c := oreMill(1, 10, 500)
	c.Modules.Resources.Resources[0].Abundance = []AbundanceKnot{{Share: 0, Factor: 2_000}, {Share: 10_000, Factor: 10_000}}
	w := newMill(t, c)
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep() error = %v, want nil", err)
	}
	w.p02()
	if err := w.p11Moment(); err != nil {
		t.Fatalf("p11Moment() error = %v, want nil", err)
	}
	w.p12Anchors()
	if got := w.prod.anchor[0].YieldBP; got != 6_000 {
		t.Errorf("anchor yield at P1.2 = %d, want 6000", got)
	}
	w.p21Complete()
	w.p23Snapshot()
	w.refreshViewCtx()
	var v AgentView
	if err := w.fillAgentView(&v, &w.agents[1]); err != nil || v.Recipes[0].Effective != 6_000 {
		t.Errorf("view yield = %d, error %v, want 6000", v.Recipes[0].Effective, err)
	}
	if err := w.p25Work(); err != nil {
		t.Fatalf("p25Work() error = %v, want nil", err)
	}
	w.p26Ration()
	// A job of duration 1 completes in its start tick, so the yield is read from the
	// request that P2.7 makes.
	if q := w.startReq(&w.agents[1], 0, 0, 0); q.effective != 6_000 {
		t.Errorf("start yield = %d, want 6000", q.effective)
	}
}

// TestReplayRows_ResourcesRows verifies that the replay of the position hash has a row for
// each position of the module, and that each runs only with the module on.
//
// Guard: a position added to tick.go without a row makes the position hash skip
// it silently; checkReplayRows would fail the replay, but not name the guard.
func TestReplayRows_ResourcesRows(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"P2.3", "P2.6"} {
		found := false
		for _, r := range replayRows {
			if r.id == id {
				found = true
				if r.on == nil || r.on(&World{}) {
					t.Errorf("replay row %s runs with the module off, want it guarded by resourcesOn", id)
				}
			}
		}
		if !found {
			t.Errorf("replayRows has no row for %s", id)
		}
	}
}

// TestRation_RootBuildsDistinctRefs verifies the contract of internal/endowment
// that the Refs of a ration are distinct slots of the grants: the root gives the
// agent ID as the Ref, an agent makes at most one request on an endowment, and a
// second one is a violated invariant.
//
// Requirement: FR-RES-019, FR-RES-020
func TestRation_RootBuildsDistinctRefs(t *testing.T) {
	t.Parallel()
	w := newMill(t, oreMill(5, 10, 20))
	tickInspecting(t, w, func() {
		seen := map[int]bool{}
		for _, q := range w.res.ration {
			if seen[q.Ref] {
				t.Errorf("Ref %d is in the ration twice", q.Ref)
			}
			seen[q.Ref] = true
		}
		if len(seen) != 10 {
			t.Errorf("distinct Refs = %d, want 10 (one request for each of the ten agents)", len(seen))
		}
	})
	w.res.reqs = append(w.res.reqs[:0], resRequest{agent: 1, cell: 0, executions: 1, draw: 3}, resRequest{agent: 1, cell: 0, executions: 1, draw: 3})
	if msg := invariantMessage(w.p26Ration); msg == "" {
		t.Errorf("p26Ration with two requests of one agent on one endowment did not panic, want an invariant")
	}
}
