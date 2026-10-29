package economysim_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file tests the sustainable draw in the viability rules through the public
// API (spec-resources.md 9.2, 11; spec-viability.md 4.4, 10): the resource_short
// error, the headroom and runway of the report, the season peak warning and the
// producibility predicate.

// wellWorld is one region "rim" of 100 households whose upkeep is 1 unit of the raw
// commodity "water" a tick, drawn from the resource "well" by the recipe "tap": no
// inputs, a duration of 1, 5 water for 2 units of the well. At the reference
// programme the households need 100 water a tick, so 20 executions and 40 units of
// the well a tick, 14 600 a year. The well holds stock units of a capacity of
// 1 000 000, and the horizon of a finite stock is horizon years.
func wellWorld(res economysim.Resource, stock, horizon int64) economysim.Config {
	world := economysim.DefaultWorldParams()
	world.Upkeep = []economysim.Item{{Commodity: "water", Quantity: 1}}
	water := economysim.DefaultCommodity()
	water.ID, water.ReferencePrice, water.Raw = "water", 1_000, true
	region := economysim.DefaultRegion()
	region.ID = "rim"
	role := economysim.DefaultRole()
	role.ID, role.Recipes = "drawer", []string{"tap"}
	res.ID, res.Capacity = "well", 1_000_000
	return economysim.Config{
		World: world, Calendar: economysim.DefaultCalendar(),
		Commodities: []economysim.Commodity{water}, Roles: []economysim.Role{role}, Regions: []economysim.Region{region},
		Groups: []economysim.Group{{Region: "rim", Role: "drawer", Households: 100}},
		Recipes: []economysim.Recipe{{
			ID: "tap", Duration: 1, TraineeYield: 5_000,
			Outputs: []economysim.Output{{Commodity: "water", Quantity: 5, CostShare: 10_000}},
			Draws:   []economysim.Draw{{Resource: "well", Quantity: 2}},
		}},
		Modules: economysim.Modules{Resources: &economysim.Resources{
			HorizonYears: horizon, Resources: []economysim.Resource{res},
			Endowments: []economysim.Endowment{{Region: "rim", Resource: "well", HasInitial: true, Initial: stock}},
		}},
	}
}

// finiteWell is a constant stock with no rate, a deposit.
func finiteWell() economysim.Resource {
	r := economysim.DefaultResource()
	r.Shape = economysim.ShapeConstant
	return r
}

// TestSolve_ResourceRowBinds verifies that a deposit too small for the horizon
// is refused as resource_short, not labour_short, with the region and the
// resource in the path, and that the report of the same programme gives a
// headroom below 1 and a rent above 0.
//
// Arithmetic: 100 households need 100 water a tick = 20 executions = 40 units a
// tick. A deposit of 73 000 over 5 years of 365 ticks is 40 a tick: exactly
// enough; over 10 years it is 20 a tick, half what is needed (lambda* = 1/2).
//
// Requirement: FR-RES-040
func TestSolve_ResourceRowBinds(t *testing.T) {
	t.Parallel()
	if err := wellWorld(finiteWell(), 73_000, 5).Validate(); err != nil {
		t.Fatalf("Validate(enough for the horizon) = %v, want nil", err)
	}
	c := wellWorld(finiteWell(), 73_000, 10)
	err := c.Validate()
	var ve *economysim.ViabilityError
	if !errors.Is(err, economysim.ErrNotViable) || !errors.As(err, &ve) {
		t.Fatalf("Validate(half enough) = %v, want ErrNotViable", err)
	}
	if ve.Rule != "resource_short" || !slices.Equal(ve.Path, []string{"regions", "rim", "resources", "well"}) || ve.Value != "5000" {
		t.Errorf("Validate(half enough) rule, path, value = %q, %v, %q, want resource_short, regions/rim/resources/well, 5000", ve.Rule, ve.Path, ve.Value)
	}
	cause := []string{"regions", "rim", "resources", "well"}
	if !slices.ContainsFunc(c.Warnings(), func(w economysim.Warning) bool {
		return w.Rule == "region_not_self_sufficient" && slices.Equal(w.Path, cause)
	}) {
		t.Errorf("Warnings(half enough) has no region_not_self_sufficient at %v, want one naming the resource", cause)
	}
	rep, err := economysim.SteadyReportUnchecked(c)
	if err != nil {
		t.Fatalf("SteadyReportUnchecked() error = %v, want nil", err)
	}
	if len(rep.Headroom) != 1 || rep.Headroom[0].BP != 5_000 {
		t.Errorf("Report.Headroom = %+v, want one row of 5000 bp: 20 a tick against 40", rep.Headroom)
	}
	if len(rep.Rent) != 1 || rep.Rent[0].Rent <= 0 {
		t.Errorf("Report.Rent = %+v, want one row above 0: the row binds", rep.Rent)
	}
}

// TestSolve_RowSlack verifies that a row with room to spare is slack: a headroom
// above 1, no rent, and the same pack passes validation.
//
// Arithmetic: 146 000 over 1 year of 365 ticks is 400 a tick against a need of 40:
// headroom 400/40 = 10. Labour allows 100 executions a tick, 5 times the 20 the
// basket needs, so the surplus factor is 5 and the row, which would allow 10, does
// not bind. (A headroom of 2.5 would still have a rent: it is the row at lambda*
// that binds, and lambda* is 5 here, above the 2.5 the row allows.)
//
// Requirement: FR-RES-040
func TestSolve_RowSlack(t *testing.T) {
	t.Parallel()
	st, err := economysim.SolveSteadyState(context.Background(), wellWorld(finiteWell(), 146_000, 1))
	if err != nil {
		t.Fatalf("SolveSteadyState() error = %v, want nil", err)
	}
	if len(st.Report.Headroom) != 1 || st.Report.Headroom[0].BP != 100_000 {
		t.Errorf("Report.Headroom = %+v, want one row of 100000 bp", st.Report.Headroom)
	}
	if len(st.Report.Rent) != 1 || st.Report.Rent[0].Rent != 0 {
		t.Errorf("Report.Rent = %+v, want one row of 0: the row is slack", st.Report.Rent)
	}
}

// TestSolve_OffUnchanged verifies that with the module off the report has no
// resource rows and a recipe with no draw is unchanged: the packs of the viability
// goldens (viability_agree_pack_test.go) are the proof for the table.
//
// Guard: the module off must leave the solver, its table and its report as they
// were.
//
// Requirement: FR-RES-039
func TestSolve_OffUnchanged(t *testing.T) {
	t.Parallel()
	st, err := economysim.SolveSteadyState(context.Background(), referenceConfig())
	if err != nil {
		t.Fatalf("SolveSteadyState(reference) error = %v, want nil", err)
	}
	if st.Report.Headroom != nil || st.Report.Rent != nil || st.Report.Runway != nil {
		t.Errorf("Report resource rows with the module off = %+v, %+v, %+v, want none", st.Report.Headroom, st.Report.Rent, st.Report.Runway)
	}
}

// TestRunway_FiniteStock verifies the run-down time of a finite stock at the
// reference draw: the starting stock over the draw a tick, and none for a stock
// that renews.
//
// Arithmetic: 146 000 units at 40 a tick last 3 650 ticks = 10 years of 365.
//
// Requirement: FR-RES-040
func TestRunway_FiniteStock(t *testing.T) {
	t.Parallel()
	st, err := economysim.SolveSteadyState(context.Background(), wellWorld(finiteWell(), 146_000, 5))
	if err != nil {
		t.Fatalf("SolveSteadyState() error = %v, want nil", err)
	}
	want := []economysim.RunwayRow{{Region: "rim", Resource: "well", Ticks: 3_650}}
	if !slices.Equal(st.Report.Runway, want) {
		t.Errorf("Report.Runway = %+v, want %+v", st.Report.Runway, want)
	}
	refill := economysim.DefaultResource()
	refill.Shape, refill.RateBP = economysim.ShapeRefill, 10_000
	st, err = economysim.SolveSteadyState(context.Background(), wellWorld(refill, 500_000, 5))
	if err != nil {
		t.Fatalf("SolveSteadyState(refill) error = %v, want nil", err)
	}
	if len(st.Report.Runway) != 0 {
		t.Errorf("Report.Runway of a stock that renews = %+v, want none", st.Report.Runway)
	}
}

// TestExistence_DrawBeyondCapacity verifies that a recipe whose draw is above the
// capacity of every endowment does not exist for the viability rules, so the
// commodity it sources is unproducible (spec-resources.md 9.2).
//
// Requirement: FR-RES-037
func TestExistence_DrawBeyondCapacity(t *testing.T) {
	t.Parallel()
	c := wellWorld(finiteWell(), 146_000, 20)
	c.Recipes[0].Draws[0].Quantity = 1_000_001 // above the capacity of 1 000 000
	err := c.Validate()
	var ve *economysim.ViabilityError
	if !errors.Is(err, economysim.ErrNotViable) || !errors.As(err, &ve) || ve.Rule != "role_cannot_work" {
		t.Errorf("Validate(a draw above the capacity) = %v, want ErrNotViable with role_cannot_work", err)
	}
}

// TestExistence_StockCannotStart verifies the second conjunct of the predicate: a
// stock of 0 that does not regrow cannot start a recipe, one that regrows can, and
// a stock above 0 can.
//
// Requirement: FR-RES-037
func TestExistence_StockCannotStart(t *testing.T) {
	t.Parallel()
	refill, logistic := economysim.DefaultResource(), economysim.DefaultResource()
	refill.Shape, refill.RateBP = economysim.ShapeRefill, 10_000
	logistic.Shape, logistic.RateBP = economysim.ShapeLogistic, 10_000
	refuge := logistic
	refuge.Refuge = 5
	tests := []struct {
		name  string
		res   economysim.Resource
		stock int64
		runs  bool
	}{
		{"finite and empty", finiteWell(), 0, false},
		{"finite and holding some", finiteWell(), 1, true},
		{"constant with a rate, empty", func() economysim.Resource { r := finiteWell(); r.RatePerYear = 100_000; return r }(), 0, true},
		{"refill, empty", refill, 0, true},
		{"refill with no rate, empty", func() economysim.Resource { r := refill; r.RateBP = 0; return r }(), 0, false},
		{"logistic, empty, no refuge", logistic, 0, false},
		{"logistic, empty, with a refuge", refuge, 0, true},
		{"logistic holding some", logistic, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := wellWorld(tt.res, tt.stock, 50).Validate()
			var ve *economysim.ViabilityError
			if tt.runs && errors.As(err, &ve) && ve.Rule == "role_cannot_work" {
				t.Errorf("Validate(%s) = %v, want the recipe to exist", tt.name, err)
			}
			if !tt.runs && !(errors.As(err, &ve) && ve.Rule == "role_cannot_work") {
				t.Errorf("Validate(%s) = %v, want role_cannot_work: the recipe cannot start", tt.name, err)
			}
		})
	}
}

// TestRawSourcing_Predicate verifies that the Raw marker needs a source that can
// start: with the only source's endowment unable to start, raw_without_source is
// raised for the commodity, and not when it can (spec-resources.md 9.3,
// spec-viability.md 10.2).
//
// Requirement: FR-RES-038
func TestRawSourcing_Predicate(t *testing.T) {
	t.Parallel()
	rules := func(c economysim.Config) []string {
		var out []string
		for _, w := range c.Warnings() {
			if w.Rule == "raw_without_source" {
				out = append(out, w.Path[len(w.Path)-1])
			}
		}
		return out
	}
	if got := rules(wellWorld(finiteWell(), 1, 50)); len(got) != 0 {
		t.Errorf("raw_without_source with a stock that can start = %v, want none", got)
	}
	if got := rules(wellWorld(finiteWell(), 0, 50)); !slices.Equal(got, []string{"water"}) {
		t.Errorf("raw_without_source with an empty finite stock = %v, want [water]", got)
	}
	off := wellWorld(finiteWell(), 0, 50)
	off.Modules.Resources, off.Recipes[0].Draws = nil, nil
	if got := rules(off); len(got) != 0 {
		t.Errorf("raw_without_source with the module off and no draw = %v, want none: the recipe is a source", got)
	}
}

// windowWell is wellWorld with a calendar of two seasons, wet (90 days) and dry
// (275), and a tap that runs in the wet season only, so its starts are confined to
// a window of 90 ticks. The well is a refill of 40 000 bp a year (4 times the gap)
// with a plan share of 2 000 and a capacity of capacity, held full at the start.
func windowWell(capacity int64) economysim.Config {
	res := economysim.DefaultResource()
	res.Shape, res.RateBP, res.PlanShareBP = economysim.ShapeRefill, 40_000, 2_000
	c := wellWorld(res, 0, 50)
	c.Calendar.Seasons = []economysim.Season{{ID: "wet", Days: 90}, {ID: "dry", Days: 275}}
	c.Modules.Resources.Resources[0].Capacity = capacity
	c.Modules.Resources.Endowments = []economysim.Endowment{{Region: "rim", Resource: "well", HasCapacity: true, Capacity: capacity}}
	c.Yields = []economysim.YieldProfile{{Region: "rim", Recipe: "tap", Seasons: []economysim.SeasonYield{{Season: "wet", BP: 10_000}}}}
	return c
}

// TestSeasonPeak_Warning verifies endowment_season_peak: a tap whose starts all
// fall in the 90 wet ticks draws a year's 14 600 units in a window, more than the
// plan stock plus the window's regrowth holds, although the annual flow suffices;
// a bigger well does not raise it. It is a warning and never an error.
//
// Arithmetic: a well of 5 000 has a plan stock of 2 000 * 5 000 / 10 000 = 1 000
// and regrows 4.0 * 4 000 = 16 000 a year, 43.8 a tick against the 40 needed (so
// the row passes), and 90/365 of it in the wet window, 3 945: the opening stock
// 4 945 against 14 600 drawn. A well of 20 000 holds 4 000 and regrows 16 000 *
// 4 = 64 000 a year, 15 781 in the window: 19 781 against 14 600, no peak.
//
// Requirement: FR-RES-040
func TestSeasonPeak_Warning(t *testing.T) {
	t.Parallel()
	peaks := func(c economysim.Config) bool {
		if err := c.Validate(); err != nil {
			t.Fatalf("Validate(window well) = %v, want nil: a warning is never an error", err)
		}
		return slices.ContainsFunc(c.Warnings(), func(w economysim.Warning) bool {
			return w.Rule == "endowment_season_peak" && slices.Equal(w.Path, []string{"regions", "rim", "resources", "well"})
		})
	}
	if !peaks(windowWell(5_000)) {
		t.Errorf("Warnings(a well of 5 000, starts in 90 ticks) has no endowment_season_peak, want one")
	}
	if peaks(windowWell(20_000)) {
		t.Errorf("Warnings(a well of 20 000, starts in 90 ticks) has endowment_season_peak, want none")
	}
	open := windowWell(5_000)
	open.Yields = nil
	if peaks(open) {
		t.Errorf("Warnings(a tap open all year) has endowment_season_peak, want none: it has no window")
	}
}
