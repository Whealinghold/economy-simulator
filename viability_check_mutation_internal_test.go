package economysim

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// Equivalent mutants of viability_check.go that no test can kill (gomu, one-line why each):
//   - upkeepGoods `m.upkeep[r][ci] > 0 -> !=` (111), noProgramme (252), storage
//     (346) and imports (373): a demand total is a sum of non-negative products.
//   - sourced `n == 0 -> <=` and `n == 0 -> >=` (134): a length is never negative,
//     and a repeated region in a role's list is invisible, as CannotWork stops at
//     the first region with no recipe.
//   - sourced `!= -> >`, `>=` (134): canonical groups are sorted by region, so a
//     role's regions only ever ascend.
//   - sourced `len(...) > 0 -> !=` (154, both): a length is never negative.
//   - cycles `err != nil` and its body (181 to 183): steady.Cycles fails only on a
//     cancelled context (it gets Background) or a failed certificate, and the
//     overflow of the floor is the case above it.
//   - storage `shelf == 0 -> <, <=` and `len(prods[ci]) == 0 -> <=` (337): a shelf
//     life and a length are never negative, and a shelf life of 0 never exceeds.
//   - hasSource `len(a.act.Inputs) > 0 -> !=` (266): a length is never negative.
//   - autarky `&& -> ||` (400, both): a skipped solve leaves a zero outcome and a
//     missing labour share is 0, and 0 is never above the mean, so the three terms
//     are not independent.

// viaModelOf returns the model of a configuration (canonicalised, not validated).
func viaModelOf(c Config) *viaModel {
	cc := c.canonical()
	return newViaModel(&cc)
}

// viaItems makes steady items of (commodity, quantity) pairs.
func viaItems(pairs ...int64) []steady.Item {
	var out []steady.Item
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, steady.Item{Commodity: int(pairs[i]), Qty: pairs[i+1]})
	}
	return out
}

// viaAct is an activity of region 0 with the given inputs and outputs.
func viaAct(allYear bool, in, out []steady.Item) viaActivity {
	return viaActivity{allYear: allYear, act: steady.Activity{Duration: 1, Starts: 1, FactorSum: 10_000, Inputs: in, Outputs: out}}
}

// viaCustom returns a one-region model of n commodities c0.. whose activities of
// region 0 are exactly acts, numbered as recipes in order.
func viaCustom(n int, acts ...viaActivity) *viaModel {
	c := viaToy()
	c.Commodities = nil
	for i := range n {
		c.Commodities = append(c.Commodities, viaCommodity(fmt.Sprintf("c%d", i), 100, nil))
	}
	c.World.Upkeep = []Item{{Commodity: "c0", Quantity: 1}}
	c.Recipes[0].Outputs[0].Commodity = "c0"
	m := viaModelOf(c)
	m.acts[0] = nil
	for i, a := range acts {
		a.recipe = RecipeID(i)
		m.acts[0] = append(m.acts[0], a)
	}
	return m
}

// TestViabilityOutcome_FailKeepsTheFirstDefect verifies that fail keeps the first
// defect, bare when it is the certificate error and wrapped otherwise.
//
// Guard: a second defect overwriting the first, the bare sentinel being wrapped
// or the certificate error not being recorded.
//
// Requirement: FR-VIA-014
func TestViabilityOutcome_FailKeepsTheFirstDefect(t *testing.T) {
	t.Parallel()
	var first viabilityOutcome
	first.fail(errors.New("first"))
	first.fail(steady.ErrInconsistent)
	if first.fault == nil || first.fault == ErrSolveInconsistent || !strings.Contains(first.fault.Error(), "first") {
		t.Errorf("fail(first), fail(certificate) fault = %v, want the wrapped first error", first.fault)
	}
	var cert viabilityOutcome
	cert.fail(steady.ErrInconsistent)
	if cert.fault != ErrSolveInconsistent {
		t.Errorf("fail(certificate) fault = %v, want ErrSolveInconsistent itself", cert.fault)
	}
	var other viabilityOutcome
	other.fail(errors.New("tableau"))
	if other.fault == ErrSolveInconsistent || !errors.Is(other.fault, ErrSolveInconsistent) || !strings.Contains(other.fault.Error(), "tableau") {
		t.Errorf("fail(tableau) fault = %v, want ErrSolveInconsistent wrapping the cause", other.fault)
	}
}

// TestViabilityOutcome_SkippedPath verifies the rule and the path of a skip.
//
// Guard: the "viability" prefix of the path.
//
// Requirement: FR-VIA-010
func TestViabilityOutcome_SkippedPath(t *testing.T) {
	t.Parallel()
	var o viabilityOutcome
	o.skipped("v3_region", "north")
	want := []Warning{{Path: []string{"viability", "v3_region", "north"}, Rule: "viability_check_skipped"}}
	if len(o.warnings) != 1 || o.warnings[0].Rule != want[0].Rule || !slices.Equal(o.warnings[0].Path, want[0].Path) {
		t.Errorf("skipped(v3_region, north) = %v, want %v", o.warnings, want)
	}
}

// TestViaModel_RealmAndProducersLeaveDiscardOut verifies that a discarded output
// is neither made by an activity of the realm nor produced for the storage check.
//
// Guard: a discard output counted as a product.
//
// Requirement: FR-VIA-004, FR-VIA-008
func TestViaModel_RealmAndProducersLeaveDiscardOut(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Commodities = append(c.Commodities, viaCommodity("waste", 10, nil))
	c.Recipes[0].Outputs = append(c.Recipes[0].Outputs, Output{Commodity: "waste", Quantity: 1, Discard: true})
	m := viaModelOf(c)
	// grain is commodity 0 and waste 1 (sorted by ID).
	r := m.realm()
	if len(r.Makers) != 1 || !slices.Equal(r.Makers[0].Makes, []int{0}) {
		t.Errorf("realm makers = %+v, want one maker of commodity 0 only", r.Makers)
	}
	p := m.producers(0)
	if len(p[0]) != 1 || len(p[1]) != 0 {
		t.Errorf("producers = %d of grain, %d of waste, want 1 and 0", len(p[0]), len(p[1]))
	}
}

// TestViaModel_AllYearInputs verifies which inputs are marked: those of the
// all-year activities only, whatever their position.
//
// Guard: an input of a seasonal activity marked, or the scan stopping at the first
// seasonal activity.
//
// Requirement: FR-VIA-008
func TestViaModel_AllYearInputs(t *testing.T) {
	t.Parallel()
	m := viaCustom(3,
		viaAct(false, viaItems(1, 1), viaItems(0, 1)),
		viaAct(true, viaItems(2, 1), viaItems(0, 1)),
	)
	got := m.allYearInputs(0)
	if want := []bool{false, false, true}; !slices.Equal(got, want) {
		t.Errorf("allYearInputs = %v, want %v", got, want)
	}
}

// TestViaModel_UpkeepGoodsAndNoProgramme verifies the smallest demand: a region
// whose households need exactly one unit of a good has a programme and lists it,
// and one that needs nothing lists none and has none.
//
// Guard: a demand of exactly 1 counted as none.
//
// Requirement: FR-VIA-007
func TestViaModel_UpkeepGoodsAndNoProgramme(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Groups[0].Households = 1
	c.Commodities = append(c.Commodities, viaCommodity("zinc", 10, nil))
	m := viaModelOf(c)
	if m.upkeep[0][0] != 1 {
		t.Fatalf("upkeep = %v, want the 1 unit the case is about", m.upkeep[0])
	}
	if got := m.upkeepGoods(); !slices.Equal(got, []int{0}) {
		t.Errorf("upkeepGoods = %v, want [0]", got)
	}
	if m.noProgramme([]RegionID{0}) {
		t.Errorf("noProgramme(demand 1) = true, want false")
	}
	m.upkeep[0][0] = 0
	if got := m.upkeepGoods(); len(got) != 0 {
		t.Errorf("upkeepGoods(no demand) = %v, want none", got)
	}
	if !m.noProgramme([]RegionID{0}) {
		t.Errorf("noProgramme(no demand) = false, want true")
	}
}

// TestViaModel_HasSource verifies what stands behind a Raw marker: an activity
// with no inputs and a non-discard output, wherever it is in the list.
//
// Guard: an activity with a single input counted as a source, the scan stopping at
// the first activity that is not one, or a discard output counted.
//
// Requirement: FR-VIA-007
func TestViaModel_HasSource(t *testing.T) {
	t.Parallel()
	discard := []steady.Item{{Commodity: 1, Qty: 1, Discard: true}}
	tests := []struct {
		name string
		acts []viaActivity
		want bool
	}{
		{"one input is not a source", []viaActivity{viaAct(true, viaItems(0, 1), viaItems(1, 1))}, false},
		{"a source after a maker", []viaActivity{viaAct(true, viaItems(0, 1), viaItems(1, 1)), viaAct(true, nil, viaItems(1, 1))}, true},
		{"a source", []viaActivity{viaAct(true, nil, viaItems(1, 1))}, true},
		{"a discard is not a product", []viaActivity{viaAct(true, nil, discard)}, false},
		{"another commodity", []viaActivity{viaAct(true, nil, viaItems(0, 1))}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := viaCustom(2, tt.acts...)
			if got := m.hasSource([]RegionID{0}, 1); got != tt.want {
				t.Errorf("hasSource(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestViaModel_SourcedReturnsWhatFired verifies that V1 reports that a rule fired
// for each of its two rules alone, with one finding each, and that a role is
// checked in every region it has a group in.
//
// Guard: the return value being false for a single unproducible good or a single
// stuck role, and a later region of a role not being checked.
//
// Requirement: FR-VIA-004, FR-VIA-005
func TestViaModel_SourcedReturnsWhatFired(t *testing.T) {
	t.Parallel()
	t.Run("clean", func(t *testing.T) {
		t.Parallel()
		var o viabilityOutcome
		if m := viaModelOf(viaToy()); m.sourced(&o) || len(o.errs) != 0 {
			t.Errorf("sourced(toy) = errs %v, want false and none", o.errs)
		}
	})
	t.Run("one unproducible good", func(t *testing.T) {
		t.Parallel()
		c := viaToy()
		c.Commodities = append(c.Commodities, viaCommodity("stone", 10, nil))
		c.World.Upkeep = append(c.World.Upkeep, Item{Commodity: "stone", Quantity: 1})
		var o viabilityOutcome
		if !viaModelOf(c).sourced(&o) || len(o.errs) != 1 || o.errs[0].Rule != "unproducible_in_realm" || o.errs[0].Value != "stone" {
			t.Errorf("sourced = errs %v, want true and one unproducible_in_realm of stone", o.errs)
		}
	})
	t.Run("one stuck role in a later region", func(t *testing.T) {
		t.Parallel()
		c := viaToy()
		viaTwoRegions(&c)
		c.Groups = append(c.Groups, Group{Region: "south", Role: "farmer", Households: 5})
		var o viabilityOutcome
		if !viaModelOf(c).sourced(&o) || len(o.errs) != 1 || o.errs[0].Rule != "role_cannot_work" || o.errs[0].Value != "south" {
			t.Errorf("sourced = errs %v, want true and one role_cannot_work in south", o.errs)
		}
	})
}

// TestViaModel_CyclesOverflowIsAnInputSize verifies that a cycle floor out of
// int64 range is a fault of the input kind, not a solver defect, that it keeps an
// earlier fault, and that it reports no unproductive component.
//
// Arithmetic: each activity makes 2 * 10^15 for 1, so the best mix (half each) nets
// (2 * 10^15 - 1) / 2 per good, about 10^15, and 10^15 * 10 000 > 2^63 - 1.
//
// Guard: the overflow branch, its message, keeping the first fault and the return.
//
// Requirement: FR-VIA-014
func TestViaModel_CyclesOverflowIsAnInputSize(t *testing.T) {
	t.Parallel()
	const q = 2_000_000_000_000_000
	build := func() *viaModel {
		return viaCustom(2,
			viaAct(true, viaItems(0, 1), viaItems(1, q)),
			viaAct(true, viaItems(1, 1), viaItems(0, q)),
		)
	}
	var o viabilityOutcome
	if build().cycles(&o) {
		t.Errorf("cycles(overflow) = true, want false")
	}
	if !errors.Is(o.fault, ErrOverflow) || errors.Is(o.fault, ErrSolveInconsistent) || !strings.Contains(o.fault.Error(), "a cycle floor out of int64 range") {
		t.Errorf("fault = %v, want ErrOverflow naming the cycle floor", o.fault)
	}
	first := errors.New("first")
	held := viabilityOutcome{fault: first}
	if build().cycles(&held); held.fault != first {
		t.Errorf("fault after an earlier one = %v, want the earlier one kept", held.fault)
	}
}

// TestViaModel_GainWarnings verifies the cycle gain warning at the edge of the
// cost floor, with no warning and no panic at a floor of 0, one warning for two
// cycles of one lowest commodity, and the commodity a skip names.
//
// Arithmetic: a cycle of two recipes that each need 10 000 of the other's
// good (here 10 000 of c0 for 1 of c1 and 1 of c1 for 1 of c0) has the ratio 10 000;
// at a floor of 1 the gain is 1 / 10 000 * 10 000 = 1.
//
// Guard: the floor bounds 0 and 1, the dedupe by lowest commodity and comp[0].
//
// Requirement: FR-CFG-044
func TestViaModel_GainWarnings(t *testing.T) {
	t.Parallel()
	two := func() *viaModel {
		return viaCustom(2,
			viaAct(true, viaItems(0, 10_000), viaItems(1, 1)),
			viaAct(true, viaItems(1, 1), viaItems(0, 1)),
		)
	}
	three := func() *viaModel {
		return viaCustom(3,
			viaAct(true, viaItems(0, 10_000), viaItems(1, 1)),
			viaAct(true, viaItems(0, 10_000), viaItems(2, 1)),
			viaAct(true, viaItems(1, 1, 2, 1), viaItems(0, 1)),
		)
	}
	tests := []struct {
		name  string
		model func() *viaModel
		comp  []int
		floor int64
		want  int
	}{
		{"floor 1 at a gain of 1", two, []int{0, 1}, 1, 1},
		{"floor 0 draws none", two, []int{0, 1}, 0, 0},
		{"two cycles of one lowest commodity warn once", three, []int{0, 1, 2}, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.model()
			m.c.World.CostFloor = tt.floor
			var o viabilityOutcome
			m.gain(&o, tt.comp, m.candidates())
			if len(o.warnings) != tt.want {
				t.Fatalf("gain(%s) = %v, want %d warnings", tt.name, o.warnings, tt.want)
			}
			if tt.want == 1 && (o.warnings[0].Rule != "cycle_cost_gain" || !slices.Equal(o.warnings[0].Path, []string{"commodities", "c0"})) {
				t.Errorf("gain(%s) = %v, want cycle_cost_gain of c0", tt.name, o.warnings)
			}
		})
	}
	t.Run("a skip names the first commodity of the component", func(t *testing.T) {
		t.Parallel()
		m := two()
		m.maxSteps = 1
		var o viabilityOutcome
		m.gain(&o, []int{1, 0}, m.candidates())
		if len(o.warnings) != 1 || !slices.Equal(o.warnings[0].Path, []string{"viability", "cycle_gain", "c1"}) {
			t.Errorf("gain(maxSteps 1) = %v, want one skip of cycle_gain naming c1", o.warnings)
		}
	})
}

// viaStorageCase returns the model of one household that eats 1 grain a tick of a
// grain with the given shelf life that a spring-only farm makes (a gap of 273),
// after an unmade commodity with no shelf life that sorts first.
func viaStorageCase(shelf int64) (*viaModel, lookup) {
	c := viaToy()
	c.Groups[0].Households = 1
	c.Commodities[0].ShelfLife = shelf
	c.Commodities = append(c.Commodities, viaCommodity("a_salt", 10, nil))
	c.Recipes[0].Sited = true
	c.Yields = []YieldProfile{viaSpring("north", "farm")}
	cc := c.canonical()
	return newViaModel(&cc), newLookup(&cc)
}

// TestViaModel_StorageGap verifies storage_gap at the smallest demand and shelf
// life, after a commodity that is skipped, and its absence when the shelf holds.
//
// Guard: a demand of 1 counted as none, a shelf life of 1 skipped, the scan
// stopping at a skipped commodity, and a gap within the shelf life reported.
//
// Requirement: FR-VIA-008
func TestViaModel_StorageGap(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		shelf int64
		want  bool
	}{
		{"shelf below the gap", 100, true},
		{"shelf of 1", 1, true},
		{"shelf at the gap", 273, false},
		{"no shelf life", 0, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, lk := viaStorageCase(tt.shelf)
			var o viabilityOutcome
			m.storage(&o, lk)
			if got := len(o.errs) == 1 && o.errs[0].Rule == "storage_gap" && o.errs[0].Value == "273"; got != tt.want {
				t.Errorf("storage(shelf %d) = errs %v, want a storage_gap of 273 = %v", tt.shelf, o.errs, tt.want)
			}
			if !tt.want && len(o.errs) != 0 {
				t.Errorf("storage(shelf %d) = errs %v, want none", tt.shelf, o.errs)
			}
		})
	}
}

// TestViaModel_ImportsAtTheSmallestDemand verifies that a region whose single
// household needs 1 unit of a good it cannot make is told so, and the region that
// makes it is not.
//
// Guard: a demand of exactly 1 counted as none.
//
// Requirement: FR-VIA-009
func TestViaModel_ImportsAtTheSmallestDemand(t *testing.T) {
	t.Parallel()
	c := viaToy()
	viaTwoRegions(&c)
	c.Groups[1].Households = 1
	m := viaModelOf(c)
	prod := steady.Producible(m.realm())
	var o viabilityOutcome
	if m.imports(&o, 0, prod) {
		t.Errorf("imports(north) = true, want false")
	}
	if !m.imports(&o, 1, prod) || !viaWarned(o.warnings, "region_not_self_sufficient", "regions", "south", "commodities", "grain") {
		t.Errorf("imports(south) warnings = %v, want true and the south import of grain", o.warnings)
	}
}

// TestViaModel_RegionsGoOnAfterAnImporter verifies that a region that imports a
// good, which has no autarky programme, does not stop the check of the next one.
//
// Guard: the loop stopping at the first region it skips.
//
// Requirement: FR-VIA-009
func TestViaModel_RegionsGoOnAfterAnImporter(t *testing.T) {
	t.Parallel()
	c := viaToy()
	viaTwoRegions(&c)
	// a_south sorts before north, so it is region 0; north farms at 0.2: 0.8 < 1.
	c.Regions[1].ID, c.Groups[1].Region = "a_south", "a_south"
	c.Yields[0].All = 2_000
	m := viaModelOf(c)
	if m.c.Regions[0].ID != "a_south" {
		t.Fatalf("regions = %v, want a_south first", m.c.Regions)
	}
	var o viabilityOutcome
	m.regions(&o)
	if !viaWarned(o.warnings, "region_not_self_sufficient", "regions", "north") {
		t.Errorf("regions warnings = %v, want the surplus warning of north after the importer", o.warnings)
	}
}

// TestViaModel_SolveErrorsAreFaults verifies that an overflow of the programme of
// a region, in its full-labour and its nominal form, is kept as a fault with no
// finding and no later warning.
//
// Guard: the error case of autarky and of nominal.
//
// Requirement: FR-VIA-014
func TestViaModel_SolveErrorsAreFaults(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(m *viaModel, o *viabilityOutcome)
	}{
		{"autarky", func(m *viaModel, o *viabilityOutcome) { m.autarky(o, 0) }},
		{"nominal", func(m *viaModel, o *viabilityOutcome) { m.nominal(o, 0) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := viaModelOf(viaToy())
			// 9.2e15 households * 10^6 starts / 365 ticks = 2.5e19 > 2^63 - 1 in the
			// capacity row of the activity; its year of 365 is over the 0 ticks it lasts.
			m.households[0] = math.MaxInt64 / 1000
			m.acts[0][0].act.Duration, m.acts[0][0].act.Starts = 0, 1_000_000
			var o viabilityOutcome
			tt.run(m, &o)
			if !errors.Is(o.fault, ErrSolveInconsistent) || len(o.errs) != 0 || len(o.warnings) != 0 {
				t.Errorf("%s with a capacity row of 2.5e19: fault = %v, errs = %v, warnings = %v, want a fault only", tt.name, o.fault, o.errs, o.warnings)
			}
		})
	}
}

// TestViaModel_NominalOverflowRecordsNothing verifies that a work-share
// denominator that does not fit stops the nominal check with nothing recorded.
//
// Guard: the overflow guard of nominal.
//
// Requirement: FR-VIA-009
func TestViaModel_NominalOverflowRecordsNothing(t *testing.T) {
	t.Parallel()
	m := viaModelOf(viaToy())
	m.households[0] = math.MaxInt64 / 100
	var o viabilityOutcome
	m.nominal(&o, 0)
	if o.fault != nil || len(o.errs) != 0 || len(o.warnings) != 0 {
		t.Errorf("nominal(households 2^63 / 100) = fault %v, errs %v, warnings %v, want nothing", o.fault, o.errs, o.warnings)
	}
}

// TestViaModel_AutarkyErrorStopsBeforeTheNominalForm verifies that a solve error
// of the autarky programme is kept as a fault by autarky itself, not left for the
// nominal form that follows it, which a region whose sums do not fit never runs.
//
// Guard: the error case of the autarky switch (its o.fail and its return).
//
// Requirement: FR-VIA-014
func TestViaModel_AutarkyErrorStopsBeforeTheNominalForm(t *testing.T) {
	t.Parallel()
	m := viaModelOf(viaToy())
	// Same overflow as TestViaModel_SolveErrorsAreFaults; sumsOK false makes the
	// nominal form unreachable, so only the autarky case can record the fault.
	m.households[0] = math.MaxInt64 / 1000
	m.acts[0][0].act.Duration, m.acts[0][0].act.Starts = 0, 1_000_000
	m.sumsOK[0] = false
	var o viabilityOutcome
	m.autarky(&o, 0)
	if !errors.Is(o.fault, ErrSolveInconsistent) || len(o.warnings) != 0 {
		t.Errorf("autarky with a capacity row of 2.5e19 and sumsOK false: fault = %v, warnings = %v, want a fault only", o.fault, o.warnings)
	}
}
