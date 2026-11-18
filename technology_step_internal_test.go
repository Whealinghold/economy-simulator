package economysim

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/production"
	tec "github.com/Whealinghold/economy-sim/internal/technology"
)

// The step, the counter and the production term (plan 3-tec, Phase 5), on Go-built
// worlds of one recipe.

// executionsIn sums the Executions of the RecipeExecuted events of rec.
func executionsIn(rec *evLog) (sum int64) {
	for _, e := range of[RecipeExecuted](rec) {
		sum += e.Executions
	}
	return sum
}

// TestCount_FoldsExecutions verifies that the executions of every completed job count in
// the cell: for a recipe of one tick (completed at P2.7) and of two ticks (completed at
// P2.1), the sum of the Executions of TechnologyAdvanced and the Experience at the end
// equal the sum of the Executions of RecipeExecuted; and that a job abandoned by the
// removal of its agent counts nothing.
//
// Requirement: FR-TEC-007
func TestCount_FoldsExecutions(t *testing.T) {
	t.Parallel()
	for _, duration := range []int64{1, 2, 3} {
		checkFolds(t, duration)
	}
}

// checkFolds runs a mill of the given recipe duration for 12 ticks and checks that the
// counted executions match the executions done, and the removed agent's job counts
// nothing.
func checkFolds(t *testing.T, duration int64) {
	t.Helper()
	rec := &evLog{}
	w := newMill(t, techMill(duration, 3, techField()), WithObserver(rec))
	step(t, w, 1)
	if duration == 3 { // every job is under way: one agent leaves with its job
		if err := w.removeAgent(&w.agents[w.firstHousehold], CauseConsumer); err != nil {
			t.Fatalf("removeAgent() = %v, want nil", err)
		}
	}
	step(t, w, 11)
	var counted, last int64
	for _, e := range of[TechnologyAdvanced](rec) {
		counted += e.Executions
		last = e.Experience
	}
	if done := executionsIn(rec); done == 0 || counted != done || last != done || w.tec.cells[0].Experience != done {
		t.Errorf("duration %d: RecipeExecuted %d, TechnologyAdvanced executions %d, experience %d (cell %d), want all equal and above 0",
			duration, done, counted, last, w.tec.cells[0].Experience)
	}
	if abandoned := of[RecipeAbandoned](rec); duration == 3 && len(abandoned) != 1 {
		t.Errorf("duration 3: %d RecipeAbandoned, want 1 (the removed agent's job)", len(abandoned))
	}
}

// TestCount_Rules verifies the counter at the unit: a recipe in no field, a field with
// every mechanism off and an outside node add nothing, a count adds, and the sum
// saturates at MaxExperience.
//
// Requirement: FR-TEC-007, FR-TEC-035
func TestCount_Rules(t *testing.T) {
	t.Parallel()
	c := techTwoByTwo()
	c.Modules.Technology.Fields[1].TrendBP = 0 // grinding has every rate 0
	w := newMill(t, c)
	tt := w.tec // fields: grinding (0, inert), milling (1); recipes: grind 0, mill 1
	tt.count(0, 0, 5)
	tt.count(0, 1, 7)
	tt.count(1, 1, 2)
	tt.count(1, 1, 3)
	if want := []int64{0, 7, 0, 5}; !slices.Equal(tt.counter, want) {
		t.Errorf("counter = %v, want %v (inert field not counted, counts added)", tt.counter, want)
	}
	tt.count(0, 1, 0)
	tt.count(0, 1, -4)
	if tt.counter[1] != 7 {
		t.Errorf("counter = %d, want 7 (a count of 0 or less adds nothing)", tt.counter[1])
	}
	tt.count(0, 1, MaxExperience)
	tt.count(0, 1, 1<<62)
	if tt.counter[1] != MaxExperience {
		t.Errorf("counter = %d, want it to saturate at %d", tt.counter[1], int64(MaxExperience))
	}
}

// TestEffective_TechTermOrder verifies that the technology term is the third factor of
// E = site x shock x tech x trainee x abundance, each division truncating: 7 777 x 9 000
// / 10 000 = 6 999, x 12 345 / 10 000 = 8 640, x 5 000 / 10 000 = 4 320, x 8 000 /
// 10 000 = 3 456; that 10 000 is neutral; and that a site factor of 0 gives 0 at any
// gain.
//
// Requirement: FR-TEC-018
func TestEffective_TechTermOrder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                  string
		site, shock, tech, trainee, abundance int64
		want                                  int64
	}{
		{"every term", 7_777, 9_000, 12_345, 5_000, 8_000, 3_456},
		{"the tech term alone", 7_777, 10_000, 12_345, 10_000, 10_000, 9_600},
		{"neutral", 7_777, 10_000, 10_000, 10_000, 10_000, 7_777},
		{"site 0", 0, 10_000, 110_000, 10_000, 10_000, 0},
	}
	for _, tt := range tests {
		got, ok := production.Effective(tt.site, tt.shock, tt.tech, tt.trainee, tt.abundance)
		if !ok || got != tt.want {
			t.Errorf("%s: Effective = %d, %v, want %d", tt.name, got, ok, tt.want)
		}
	}
	w := newMill(t, techMill(2, 2, techField()))
	e, _, ok := w.yields.effectiveAt(&w.shocks, 0, 0, 0, false, production.BP, 10_800)
	e0, _, _ := w.yields.effective(&w.shocks, 0, 0, 0, false)
	if !ok || e != 10_800 || e0 != 10_000 {
		t.Errorf("effectiveAt(tech 10 800) = %d, effective = %d, want 10 800 and 10 000 (the neutral read of the solver)", e, e0)
	}
}

// TestStart_OutputFixedAtStart verifies that a gain set at P8.3c acts on the starts of the
// next tick and that a running job keeps the yield it started with. A trend of 10 000 a
// year puts 27 basis points on the gain after tick 0 and 54 after tick 1 (10 000 / 365,
// carried); the households start at ticks 0 and 2 and rest between. The jobs started at
// tick 0 complete at tick 1 with 10 000; those started at tick 2 complete at tick 3 with
// 10 054, the gain after tick 1, though the gain was 82 when they ended.
//
// Requirement: FR-TEC-019, FR-TEC-010
func TestStart_OutputFixedAtStart(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	f := TechField{ID: "milling", Recipes: []string{"mill"}, CapBP: 5_000, TrendBP: 10_000}
	w := newMill(t, techMill(2, 2, f), WithObserver(rec))
	gains := []int64{}
	for range 4 {
		step(t, w, 1)
		gains = append(gains, w.tec.gain[0])
	}
	if want := []int64{27, 54, 82, 109}; !slices.Equal(gains, want) {
		t.Fatalf("gains after ticks 0 to 3 = %v, want %v", gains, want)
	}
	got := map[int64][]int64{}
	for _, e := range of[RecipeExecuted](rec) {
		got[e.StartTick] = append(got[e.StartTick], e.EffectiveYield)
	}
	if want := map[int64][]int64{0: {10_000, 10_000}, 2: {10_054, 10_054}}; !reflect.DeepEqual(got, want) {
		t.Errorf("effective yields by start tick = %v, want %v", got, want)
	}
}

// w0gain is the effective factor the view shows for its first recipe.
func w0gain(v *AgentView) int64 { return v.Recipes[0].Effective }

// TestGain_SameAtEveryPosition verifies that the gain changes only at P8.3c and at a
// call: the factor Work reads at P2.5 in a tick is 10 000 plus the gain the previous
// step left, for ten ticks of a world with a trend.
//
// Requirement: FR-TEC-010
func TestGain_SameAtEveryPosition(t *testing.T) {
	t.Parallel()
	var seen []int64
	s := &tapStrategy{onWork: func(v *AgentView) (JobRequest, bool) {
		seen = append(seen, w0gain(v))
		return JobRequest{}, false
	}}
	f := TechField{ID: "milling", Recipes: []string{"mill"}, CapBP: 5_000, TrendBP: 10_000, Base: 10, DoublingBP: 300}
	c := techMill(1, 2, f)
	c.Roles[0].Strategy = "tap"
	w := newMill(t, c, WithStrategy("tap", s))
	for range 10 {
		before := w.tec.gain[0]
		seen = seen[:0]
		step(t, w, 1)
		for _, e := range seen {
			if e != production.BP+before {
				t.Fatalf("tick %d: Work saw %d, want 10 000 + the gain %d of the last step", w.Tick()-1, e, before)
			}
		}
	}
	if w.tec.gain[0] == 0 {
		t.Errorf("gain after ten ticks = 0, want the trend to have moved it")
	}
}

// TestStep_PureFunctionOfInputs verifies that the step reads nothing but its input: two
// worlds with the same planted cells, counters and carries (no discovery, so no key
// is read) give the same cells after P8.3c though one of them has run 25 ticks of an
// economy, with other stocks, prices and households.
//
// Requirement: FR-TEC-033
func TestStep_PureFunctionOfInputs(t *testing.T) {
	t.Parallel()
	f := TechField{ID: "milling", Recipes: []string{"mill"}, CapBP: 5_000, Base: 100, DoublingBP: 300, TrendBP: 700, CatchUpBP: 2_000, Reach: 100}
	a, b := newMill(t, techMill(2, 2, f)), newMill(t, techMill(2, 6, f))
	step(t, b, 25)
	plant := func(w *World) {
		w.tec.cells[0] = tec.Cell{Experience: 321, Advance: 400, CatchUp: 5}
		w.tec.trendCarry[0] = 33
		w.tec.counter[0] = 17
	}
	plant(a)
	plant(b)
	a.p83cTechnology()
	b.p83cTechnology()
	if a.tec.cells[0] != b.tec.cells[0] || a.tec.trendCarry[0] != b.tec.trendCarry[0] || a.tec.gain[0] != b.tec.gain[0] {
		t.Errorf("cells after the step: %+v and %+v, want equal", a.tec.cells[0], b.tec.cells[0])
	}
	if a.tec.counter[0] != 0 {
		t.Errorf("counter after the fold = %d, want 0 (cleared where it is read)", a.tec.counter[0])
	}
}

// TestPositions_P83cListed verifies the position table: P8.3c is a TEC line entry after
// P8.3 (and after P8.3a and P8.3b, the positions of the modules that sit between) and
// before P8.4, and is not in notBuiltPositions.
//
// Requirement: FR-TEC-033
func TestPositions_P83cListed(t *testing.T) {
	t.Parallel()
	at := slices.IndexFunc(positionList[:], func(p position) bool { return p.id == "P8.3c" })
	if at < 1 || positionList[at].owner != "TEC" || positionList[at].where != line ||
		!slices.Contains([]string{"P8.3", "P8.3a", "P8.3b"}, positionList[at-1].id) || positionList[at+1].id != "P8.4" {
		t.Errorf("P8.3c at %d of positionList, want a TEC line entry after P8.3 to P8.3b and before P8.4", at)
	}
	if slices.Contains(notBuiltPositions[:], "P8.3c") {
		t.Errorf("notBuiltPositions lists P8.3c, which has a call")
	}
}

// techAll is the field with the four mechanisms on, on the recipe mill.
func techAll() TechField {
	return TechField{ID: "milling", Recipes: []string{"mill"}, CapBP: 5_000, Base: 100, DoublingBP: 300,
		TrendBP: 700, DiscoveryBP: 50_000, StepBP: 100, CatchUpBP: 2_000, Reach: 100}
}

// TestStable_TechnologyAllocNoOp verifies that, with the default logger and no observer,
// the step and the counter allocate nothing once the scratch is sized, on a world of two
// regions with every mechanism on.
//
// Requirement: NFR-TEC-003
func TestStable_TechnologyAllocNoOp(t *testing.T) {
	c := techMill(2, 4, techAll())
	away := DefaultRegion()
	away.ID = "away"
	c.Regions = append(c.Regions, away)
	c.Edges = []Edge{{From: "home", To: "away", Distance: 30}, {From: "away", To: "home", Distance: 30}}
	w := newMill(t, c)
	step(t, w, 3)
	w.tec.count(0, 0, 1)
	if got := testing.AllocsPerRun(50, func() {
		w.tec.count(0, 0, 1)
		w.p83cTechnology()
	}); got != 0 {
		t.Errorf("the counter and P8.3c allocate %v times a call, want 0", got)
	}
}

// BenchmarkP83c measures P8.3c on a world of two regions with every mechanism on.
func BenchmarkP83c(b *testing.B) {
	c := techMill(2, 4, techAll())
	away := DefaultRegion()
	away.ID = "away"
	c.Regions = append(c.Regions, away)
	c.Edges = []Edge{{From: "home", To: "away", Distance: 30}, {From: "away", To: "home", Distance: 30}}
	w := newMill(b, c)
	b.ReportAllocs()
	for b.Loop() {
		w.p83cTechnology()
	}
}

// TestAudit_TechnologyCreatesNothing verifies that the module owns no account and
// no ledger line: a world with every mechanism on has the accounts of the world with the
// module off, and runs 60 ticks with the audit on and a planted advance.
//
// Requirement: FR-TEC-032
func TestAudit_TechnologyCreatesNothing(t *testing.T) {
	t.Parallel()
	on, off := newMill(t, techMill(2, 4, techAll()), WithAudit()), newMill(t, millConfig(2, 4), WithAudit())
	if on.ledger.Len() != off.ledger.Len() {
		t.Errorf("accounts = %d with the module on, want %d (none of its own)", on.ledger.Len(), off.ledger.Len())
	}
	if err := on.AdjustTechnology([]TechnologyAdjustment{{Region: 0, Field: 0, DeltaBP: 900}}); err != nil {
		t.Fatalf("AdjustTechnology() error = %v, want nil", err)
	}
	step(t, on, 60)
	if on.tec.gain[0] <= 900 {
		t.Errorf("gain after 60 ticks = %d, want the mechanisms to have raised it above the planted 900", on.tec.gain[0])
	}
}

// TestStep_NoTableOfRegionsSquared verifies the guard of NFR-TEC-004: the module keeps no
// table sized by regions squared (a table of contact weights); the distance is read where
// it is used.
//
// Guard: a cached table of weights would grow with the square of the regions (1 000
// regions: 10^6 cells) and go stale when SetEdgeStatuses closes an edge.
//
// Requirement: NFR-TEC-004
func TestStep_NoTableOfRegionsSquared(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"technology.go", "technology_report.go"} {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		src := string(b)
		for _, bad := range []string{"regions*len(t.regions)", "len(t.regions)*len(t.regions)", "regions * regions", "weights"} {
			if strings.Contains(src, bad) {
				t.Errorf("%s contains %q, want no table of regions squared", name, bad)
			}
		}
	}
}
