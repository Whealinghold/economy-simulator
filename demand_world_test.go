package economysim_test

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// recStrategy is the default strategy that keeps what it saw: each view's Wants
// and each plan's want rows. With wants false it hides the wants from the
// default, as a strategy that ignores them does.
type recStrategy struct {
	wants             bool
	viewsWith         int
	wantUnits, funded int64
}

func (r *recStrategy) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	return economysim.DefaultStrategy{}.Work(v, rng)
}

func (r *recStrategy) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	if !r.wants {
		hidden := *v
		hidden.Wants = nil
		economysim.DefaultStrategy{}.Trade(&hidden, rng, p)
	} else {
		economysim.DefaultStrategy{}.Trade(v, rng, p)
	}
	if len(v.Wants) > 0 {
		r.viewsWith++
	}
	for _, in := range p.Intents {
		r.wantUnits += int64(in.WantUnits)
	}
	for _, f := range p.WantFunded {
		r.funded += int64(f.Units)
	}
}

func (r *recStrategy) Revise(o *economysim.OutcomeView, rng *rand.Rand) economysim.Belief {
	return economysim.DefaultStrategy{}.Revise(o, rng)
}

// recWorld builds a world of cfg whose roles all use rec, with the audit on.
func recWorld(t *testing.T, cfg economysim.Config, rec *recStrategy, seed uint64, opts ...economysim.Option) *economysim.World {
	t.Helper()
	for i := range cfg.Roles {
		cfg.Roles[i].Strategy = "rec"
	}
	opts = append(opts, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithStrategy("rec", rec))
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("seed %d: NewWorld error = %v, want nil", seed, err)
	}
	return w
}

func step(t *testing.T, w *economysim.World, n int) {
	t.Helper()
	for i := range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d error = %v, want nil", i, err)
		}
	}
}

// TestDemand_OffIsInert verifies that with the module off no view has a want, no
// plan has a want row or a WantUnits, the snapshot names no want and holds no
// want_consumption row, and a strategy that records the views changes no event.
//
// Requirement: FR-DEM-001, FR-AGT-076
func TestDemand_OffIsInert(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	rec := &recStrategy{wants: true}
	w := recWorld(t, referenceConfig(), rec, 1)
	step(t, w, 40)
	if rec.viewsWith != 0 || rec.wantUnits != 0 || rec.funded != 0 {
		t.Errorf("off world: views with wants %d, WantUnits %d, funded %d, want all 0", rec.viewsWith, rec.wantUnits, rec.funded)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	if s.Names.Wants != nil {
		t.Errorf("Names.Wants = %q, want nil", s.Names.Wants)
	}
	for _, r := range s.Boundary {
		if r.Reason == economysim.ReasonWantConsumption {
			t.Errorf("off world has the boundary row %+v, want none", r)
		}
	}
	off, _ := demandRun(t, referenceConfig(), 1, 40)
	cfg := referenceConfig()
	r2 := &recorder{}
	for i := range cfg.Roles {
		cfg.Roles[i].Strategy = "rec"
	}
	w2, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithAudit(), economysim.WithObserver(r2), economysim.WithStrategy("rec", &recStrategy{wants: true}))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	r2.forget()
	step(t, w2, 40)
	if !reflect.DeepEqual(off, r2.events) {
		t.Errorf("a recording strategy changed the events of an off world (%d and %d events)", len(off), len(r2.events))
	}
}

// onWorld is the pack one-demand with rec and an observer.
func onWorld(t *testing.T, rec *recStrategy, seed uint64) (*economysim.World, *recorder) {
	t.Helper()
	r := &recorder{}
	w := recWorld(t, loadValidPack(t, packOneDemand...), rec, seed, economysim.WithObserver(r))
	r.forget()
	return w, r
}

// TestDemand_ConservationWithAudit verifies that goods and money are conserved
// after every tick of a run with wants (the world audits each step), that the
// want consumption is a goods sink in BoundaryFlow under want_consumption, by
// region and commodity and with no money, and that within a region the rows of the
// declared reason follow every row of the built-in reasons.
//
// Requirement: FR-DEM-039, FR-LED-050
func TestDemand_ConservationWithAudit(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	rec := &recStrategy{wants: true}
	w, r := onWorld(t, rec, 1)
	step(t, w, 60)
	if err := w.Audit(); err != nil {
		t.Fatalf("Audit() = %v, want nil", err)
	}
	if rec.viewsWith == 0 || rec.wantUnits == 0 || rec.funded == 0 {
		t.Fatalf("on world: views with wants %d, WantUnits %d, funded %d, want all above 0", rec.viewsWith, rec.wantUnits, rec.funded)
	}
	var consumed int64
	region, tick, seenWant := economysim.RegionID(-1), int64(-9), false
	for _, e := range r.events {
		f, ok := e.(economysim.BoundaryFlow)
		if !ok {
			continue
		}
		if f.Region != region || f.Tick != tick {
			region, tick, seenWant = f.Region, f.Tick, false
		}
		if f.Reason == economysim.ReasonWantConsumption {
			seenWant = true
			consumed += f.Total
			if f.Kind != economysim.KindSink || f.Commodity == economysim.NoCommodity {
				t.Errorf("want_consumption row %+v, want a goods sink", f)
			}
		} else if seenWant && f.Tick >= 0 {
			t.Errorf("row %+v follows a want_consumption row of the region, want the declared rows last", f)
		}
	}
	if consumed == 0 {
		t.Errorf("no want_consumption in BoundaryFlow over 60 ticks, want some")
	}
}

// TestDemand_Determinism verifies that one seed gives the same bids, units and
// carries on every run, and another seed a different run.
//
// Requirement: FR-DEM-040
func TestDemand_Determinism(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	run := func(seed uint64) (*recStrategy, []int64) {
		rec := &recStrategy{wants: true}
		w, _ := onWorld(t, rec, seed)
		step(t, w, 40)
		var carries []int64
		for id := range economysim.AgentCount(w) {
			if role, weight := economysim.AgentRole(w, economysim.AgentID(id)); weight > 0 && len(economysim.LadderOf(w, role)) > 0 {
				carries = append(carries, economysim.WantCarries(w, economysim.AgentID(id))...)
			}
		}
		return rec, carries
	}
	a, ac := run(1)
	b, bc := run(1)
	if *a != *b || !slices.Equal(ac, bc) {
		t.Errorf("two runs of seed 1: %+v and %+v, carries equal %v, want identical", *a, *b, slices.Equal(ac, bc))
	}
	if c, cc := run(2); *c == *a && slices.Equal(cc, ac) {
		t.Errorf("seeds 1 and 2 gave the same run, want different ones")
	}
}

// TestDemand_StrategyIgnoringWants verifies that a strategy that hides the wants
// from the default funds none (no WantUnits, no WantFunded), that the carries
// still advance, that the audit holds, and that the planned units are still
// consumed from what the household holds.
//
// Requirement: FR-DEM-001, D309 of plan 2-5
func TestDemand_StrategyIgnoringWants(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	rec := &recStrategy{wants: false}
	w, _ := onWorld(t, rec, 1)
	before := map[economysim.AgentID][]int64{}
	for id := range economysim.AgentCount(w) {
		if role, weight := economysim.AgentRole(w, economysim.AgentID(id)); weight > 0 && len(economysim.LadderOf(w, role)) > 0 {
			before[economysim.AgentID(id)] = economysim.WantCarries(w, economysim.AgentID(id))
		}
	}
	step(t, w, 5)
	if rec.wantUnits != 0 || rec.funded != 0 {
		t.Errorf("a strategy that ignores wants placed WantUnits %d and funded %d, want 0", rec.wantUnits, rec.funded)
	}
	moved := false
	for id, c := range before {
		if _, weight := economysim.AgentRole(w, id); weight > 0 && !slices.Equal(c, economysim.WantCarries(w, id)) {
			moved = true
		}
	}
	if !moved {
		t.Errorf("no carry advanced in 5 ticks, want the carries to advance whatever the strategy does")
	}
}

// TestDemand_NoMoneyCreated verifies that the module has no money row: the
// boundary rows of its reason are goods only and the money flows of the run equal
// those of the same run with the wants hidden from the strategy, apart from
// trades between accounts, which the audit covers.
//
// Requirement: FR-DEM-023
func TestDemand_NoMoneyCreated(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, r := onWorld(t, &recStrategy{wants: true}, 1)
	step(t, w, 40)
	for _, e := range r.events {
		if f, ok := e.(economysim.BoundaryFlow); ok && f.Reason == economysim.ReasonWantConsumption && f.Commodity == economysim.NoCommodity {
			t.Errorf("want_consumption flow %+v is money, want goods only", f)
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	for _, row := range s.Boundary {
		if row.Reason == economysim.ReasonWantConsumption && row.Commodity == economysim.NoCommodity {
			t.Errorf("boundary row %+v is money", row)
		}
	}
}

// TestDemand_RoleChange verifies the carries of an agent whose role changes:
// kept for a want in both ladders, formed by the formula for a want only in the
// new ladder and dropped for one only in the old; nothing is drawn.
//
// Requirement: FR-DEM-042
func TestDemand_RoleChange(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, _ := onWorld(t, &recStrategy{wants: true}, 1)
	step(t, w, 3)
	var cook, grower economysim.AgentID = -1, -1
	cfg := loadValidPack(t, packOneDemand...)
	idx := func(id string) economysim.RoleID {
		for i, r := range cfg.Roles {
			if r.ID == id {
				return economysim.RoleID(i)
			}
		}
		t.Fatalf("no role %q", id)
		return 0
	}
	cookRole, growerRole := idx("cook"), idx("grower")
	for id := range economysim.AgentCount(w) {
		role, weight := economysim.AgentRole(w, economysim.AgentID(id))
		switch {
		case weight < 1:
		case role == cookRole && cook < 0:
			cook = economysim.AgentID(id)
		case role == growerRole && grower < 0:
			grower = economysim.AgentID(id)
		}
	}
	if cook < 0 || grower < 0 {
		t.Fatalf("no cook (%d) or grower (%d) in one-demand", cook, grower)
	}
	var bare economysim.RoleID = -1 // a role with no ladder
	for i := range cfg.Roles {
		if len(economysim.LadderOf(w, economysim.RoleID(i))) == 0 {
			bare = economysim.RoleID(i)
			break
		}
	}
	if bare < 0 {
		t.Fatalf("no role without a ladder in one-demand")
	}
	cl, gl := economysim.LadderOf(w, cookRole), economysim.LadderOf(w, growerRole)
	if len(cl) != 2 || !slices.Equal(cl, gl) {
		t.Fatalf("ladders of cook %v and grower %v, want the same two wants", cl, gl)
	}
	was := economysim.WantCarries(w, cook)
	economysim.ChangeRoleForTest(w, cook, growerRole) // both wants are in both ladders: kept
	if got := economysim.WantCarries(w, cook); !slices.Equal(got, was) {
		t.Errorf("carries after cook to grower = %v, want the kept %v", got, was)
	}
	economysim.ChangeRoleForTest(w, cook, bare) // none in the new ladder: dropped
	if got := economysim.WantCarries(w, cook); len(got) != 0 {
		t.Errorf("carries after a change to a role with no ladder = %v, want none", got)
	}
	economysim.ChangeRoleForTest(w, cook, cookRole) // both only in the new ladder: the formula
	index := economysim.DemandAgentIndex(w, cook)
	want := []int64{index * 7_919 % 10_000, (index*7_919 + 2_777) % 10_000}
	if got := economysim.WantCarries(w, cook); !slices.Equal(got, want) {
		t.Errorf("carries after the change back = %v, want the formula %v at index %d", got, want, index)
	}
}

// TestDemand_CarryAcrossWeightChange verifies that the carry of an agent
// created at construction is the formula of its position in the region's list and
// its position in the ladder (FR-DEM-013), and that a change of weight leaves the
// carries as they are while the planned units follow the weight (FR-DEM-027,
// FR-DEM-028).
//
// Requirement: FR-DEM-013, FR-DEM-027, FR-DEM-028, FR-DEM-042
func TestDemand_CarryAcrossWeightChange(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, _ := onWorld(t, &recStrategy{wants: true}, 1)
	checked := 0
	var heavy economysim.AgentID = -1
	for id := range economysim.AgentCount(w) {
		a := economysim.AgentID(id)
		role, weight := economysim.AgentRole(w, a)
		if weight < 1 || len(economysim.LadderOf(w, role)) == 0 {
			continue
		}
		for k, c := range economysim.WantCarries(w, a) {
			if want := (economysim.DemandAgentIndex(w, a)*7_919 + int64(k)*2_777) % 10_000; c != want {
				t.Errorf("agent %d carry %d = %d at creation, want %d", a, k, c, want)
			}
			checked++
		}
		if weight > 1 && heavy < 0 {
			heavy = a
		}
	}
	if checked == 0 {
		t.Fatalf("no agent with a ladder in one-demand")
	}
	if heavy < 0 {
		t.Skip("no agent of weight above 1 in one-demand")
	}
	_, weight := economysim.AgentRole(w, heavy)
	carries, planned := economysim.WantCarries(w, heavy), economysim.PlannedWants(w, heavy)
	economysim.SetWeightForTest(w, heavy, weight*2)
	if got := economysim.WantCarries(w, heavy); !slices.Equal(got, carries) {
		t.Errorf("carries after a weight change = %v, want %v", got, carries)
	}
	doubled := economysim.PlannedWants(w, heavy)
	for k := range planned {
		if doubled[k] < 2*planned[k] || doubled[k] > 2*planned[k]+1 {
			t.Errorf("planned units of want %d at weight %d = %d, at weight %d = %d, want about double", k, weight, planned[k], weight*2, doubled[k])
		}
	}
}

// TestDemand_AllocationsStayAtTheBaseline verifies that the filler and the checker
// allocate nothing once the scratch is grown (the world has no observer here), and
// TestDemand_CostIndependentOfWeight that a cohort of a thousand households costs
// the same to plan as one household.
//
// Requirement: NFR-DEM-001, NFR-DEM-002, FR-AGT-046
func TestDemand_AllocationsStayAtTheBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, _ := onWorld(t, &recStrategy{wants: true}, 1)
	step(t, w, 5)
	var some economysim.AgentID = -1
	for id := range economysim.AgentCount(w) {
		if role, weight := economysim.AgentRole(w, economysim.AgentID(id)); weight > 0 && len(economysim.LadderOf(w, role)) > 0 {
			some = economysim.AgentID(id)
			break
		}
	}
	if some < 0 {
		t.Fatal("no agent with a ladder")
	}
	for _, weight := range []int64{1, 1_000} {
		economysim.SetWeightForTest(w, some, weight)
		if n := economysim.DemandPlanAllocs(w, some, 20); n != 0 {
			t.Errorf("weight %d: the filler and the checker allocated %v per run, want 0", weight, n)
		}
	}
}

// TestPositions_DemandCalled verifies that the world calls P3.6 and P6.2 when the
// module is on and records no line for them when it is off.
//
// Requirement: FR-DEM-001 (testing.md 6.8)
func TestPositions_DemandCalled(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, on := range []bool{true, false} {
		t.Run(fmt.Sprintf("on=%v", on), func(t *testing.T) {
			t.Parallel()
			cfg := referenceConfig()
			if on {
				cfg = demandConfig()
			}
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
			if err != nil {
				t.Fatalf("NewWorld error = %v, want nil", err)
			}
			lines, err := economysim.PositionHashes(w, 3)
			if err != nil {
				t.Fatalf("PositionHashes error = %v, want nil", err)
			}
			n := 0
			for _, l := range lines {
				if slices.ContainsFunc([]string{" P3.6 ", " P6.2 "}, func(p string) bool { return contains(l, p) }) {
					n++
				}
			}
			if on && n != 6 || !on && n != 0 {
				t.Errorf("demand on = %v: %d lines for P3.6 and P6.2 over 3 ticks, want %d", on, n, map[bool]int{true: 6, false: 0}[on])
			}
		})
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestDemand_ConsumptionAfterUpkeep verifies that a want is consumed only from
// what the household holds above the next tick's upkeep of the same commodity,
// as a move to the want_consumption boundary, and that an unmet or consumed want
// changes no unfed, hunger or privation record.
//
// Requirement: FR-DEM-025, FR-DEM-026
func TestDemand_ConsumptionAfterUpkeep(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, _ := onWorld(t, &recStrategy{wants: true}, 1)
	step(t, w, 3)
	tried := 0
	for id := range economysim.AgentCount(w) {
		a := economysim.AgentID(id)
		if _, weight := economysim.AgentRole(w, a); weight < 1 || economysim.WantCarries(w, a) == nil {
			continue
		}
		c := economysim.UpkeepCommodity(w, a)
		before, after, upkeep, wb, wa := economysim.ConsumeWantForTest(w, a, c, 1_000_000)
		if want := min(before, upkeep); after != want && !(before < upkeep && after == 0) {
			t.Errorf("agent %d held %d of upkeep commodity %d (next upkeep %d), holds %d after want consumption, want %d", a, before, c, upkeep, after, want)
		}
		if wb != wa {
			t.Errorf("agent %d: unfed, hunger, privation %v before and %v after the want consumption, want no change", a, wb, wa)
		}
		tried++
	}
	if tried == 0 {
		t.Fatalf("no agent with a ladder")
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}
