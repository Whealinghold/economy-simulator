package economysim_test

import (
	"log/slog"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file tests the resources module through the public API: the identity of an
// endowment over several shapes and seeds, the snapshot rows, that observation
// changes no result, and the catalogue rows of its two events. The seams of the
// unexported helpers are in export_test.go.

// oreWorld is a mill of ten households whose recipe draws 3 units of the resource
// "ore" in "home" each execution, with the resource res (its ID set here) holding
// stock units of a capacity of 1 000. The mill has grain for the inputs and the
// upkeep of the ticks the tests run.
func oreWorld(res economysim.Resource, stock int64) economysim.Config {
	world := economysim.DefaultWorldParams()
	world.Upkeep = []economysim.Item{{Commodity: "grain", Quantity: 1}}
	world.OutputCoverTicks = 0
	grain, flour := economysim.DefaultCommodity(), economysim.DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 1_000, true
	flour.ID, flour.ReferencePrice = "flour", 3_000
	region := economysim.DefaultRegion()
	region.ID = "home"
	role := economysim.DefaultRole()
	role.ID, role.Recipes = "miller", []string{"mill"}
	role.StartStock = []economysim.Item{{Commodity: "grain", Quantity: 200}}
	res.ID, res.Capacity = "ore", 1_000
	return economysim.Config{
		World: world, Calendar: economysim.DefaultCalendar(),
		Commodities: []economysim.Commodity{grain, flour}, Roles: []economysim.Role{role}, Regions: []economysim.Region{region},
		Groups: []economysim.Group{{Region: "home", Role: "miller", Households: 10}},
		Recipes: []economysim.Recipe{{
			ID: "mill", Duration: 1, TraineeYield: 5_000,
			Inputs:  []economysim.Item{{Commodity: "grain", Quantity: 2}},
			Outputs: []economysim.Output{{Commodity: "flour", Quantity: 3, CostShare: 10_000}},
			Draws:   []economysim.Draw{{Resource: "ore", Quantity: 3}},
		}},
		Modules: economysim.Modules{Resources: &economysim.Resources{
			HorizonYears: 50, Resources: []economysim.Resource{res},
			Endowments: []economysim.Endowment{{Region: "home", Resource: "ore", HasInitial: true, Initial: stock}},
		}},
	}
}

// shapes are the three shapes with a rate that a draw of the mill can move.
func shapes() map[string]economysim.Resource {
	constant, refill, logistic := economysim.DefaultResource(), economysim.DefaultResource(), economysim.DefaultResource()
	constant.Shape, constant.RatePerYear = economysim.ShapeConstant, 6*365
	refill.Shape, refill.RateBP = economysim.ShapeRefill, 20_000
	logistic.Shape, logistic.RateBP, logistic.Refuge = economysim.ShapeLogistic, 40_000, 10
	return map[string]economysim.Resource{"constant": constant, "refill": refill, "logistic": logistic}
}

// TestIdentity_EveryTick verifies, from the events and the public read alone, that
// for every endowment and tick Stock_end = Stock_start - Drawn + Regenerated - Lost +
// Adjusted, where the loss and the adjustment of a call between ticks belong to the
// tick that follows it, over the three shapes, a range of seeds and calls between
// ticks that clamp, scale and adjust; and that the world's own audit (WithAudit)
// agrees, since a failure there fails the step.
//
// Requirement: FR-RES-025
func TestIdentity_EveryTick(t *testing.T) {
	t.Parallel()
	for name, res := range shapes() {
		for seed := range uint64(scenarioSeedCount()) {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				rec := &recorder{}
				w, err := economysim.NewWorld(oreWorld(res, 700), economysim.WithSeed(seed), economysim.WithObserver(rec), economysim.WithAudit())
				if err != nil {
					t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
				}
				for tick := range 40 {
					end := w.Endowment(0, 0)
					rec.events = rec.events[:0]
					switch tick % 8 {
					case 3:
						economysim.ShockCapacity(w, 0, 0, 4_000)
					case 5:
						economysim.ShockRegen(w, 0, 0, 30_000)
						economysim.AdjustStock(w, 0, 0, -150)
					case 7:
						economysim.ShockCapacity(w, 0, 0, 10_000)
						economysim.AdjustStock(w, 0, 0, 90)
					}
					adjusted := int64(0)
					for _, e := range rec.events {
						if a, ok := e.(economysim.EndowmentAdjusted); ok {
							adjusted += a.Delta
						}
					}
					rec.events = rec.events[:0]
					if err := w.Step(); err != nil {
						t.Fatalf("%s seed %d tick %d: Step() error = %v, want nil", name, seed, tick, err)
					}
					want := end + adjusted
					for _, e := range rec.events {
						if c, ok := e.(economysim.EndowmentChanged); ok {
							want += -c.Drawn + c.Regenerated - c.Lost
						}
					}
					if got := w.Endowment(0, 0); got != want {
						t.Fatalf("%s seed %d tick %d: stock = %d, want %d (start %d, adjusted %d, events %+v)", name, seed, tick, got, want, end, adjusted, rec.events)
					}
				}
			})
		}
	}
}

// TestSnapshot_EndowmentRows verifies the snapshot rows: one per region and
// resource with the stock now, the effective capacity, the abundance of the stock,
// the run of ticks with units short and the shock scales (10 000 when not shocked),
// and the names and the module list that say the module is on.
//
// Arithmetic: capacity 1 000 at a scale of 5 000 is 500; a stock of 250 is a share
// of 250 * 10 000 / 500 = 5 000, where the curve (0, 2 000) to (10 000, 10 000) gives
// 2 000 + trunc(8 000 * 5 000 / 10 000) = 6 000.
//
// Requirement: FR-RES-047
func TestSnapshot_EndowmentRows(t *testing.T) {
	t.Parallel()
	res := economysim.DefaultResource()
	res.Abundance = []economysim.AbundanceKnot{{Share: 0, Factor: 2_000}, {Share: 10_000, Factor: 10_000}}
	w, err := economysim.NewWorld(oreWorld(res, 700))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	snap := func() economysim.Snapshot {
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		return s
	}
	s := snap()
	want := []economysim.EndowmentRow{{Stock: 700, Capacity: 1_000, AbundanceBP: 7_600, CapacityScaleBP: 10_000, RegenScaleBP: 10_000}}
	if !reflect.DeepEqual(s.Endowments, want) || !slices.Equal(s.Names.Resources, []string{"ore"}) || !slices.Contains(s.Modules, "resources") {
		t.Errorf("Snapshot = rows %+v, names %v, modules %v, want rows %+v, names [ore], resources among the modules", s.Endowments, s.Names.Resources, s.Modules, want)
	}
	economysim.ShockCapacity(w, 0, 0, 5_000)
	economysim.ShockRegen(w, 0, 0, 2_000)
	economysim.AdjustStock(w, 0, 0, -250)
	want = []economysim.EndowmentRow{{Stock: 250, Capacity: 500, AbundanceBP: 6_000, CapacityScaleBP: 5_000, RegenScaleBP: 2_000}}
	if got := snap().Endowments; !reflect.DeepEqual(got, want) {
		t.Errorf("Snapshot rows after the calls = %+v, want %+v", got, want)
	}
}

// TestObservationDoesNotChangeResults verifies that a logger at Debug and an
// observer change no stock, grant or draw: two worlds of one seed, one observed and
// logged, give the same stock after every tick and the same snapshot at the end.
//
// Requirement: FR-RES-046
func TestObservationDoesNotChangeResults(t *testing.T) {
	t.Parallel()
	res := shapes()["logistic"]
	plain, err := economysim.NewWorld(oreWorld(res, 300), economysim.WithSeed(5))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	log := newCaptureHandler()
	seen, err := economysim.NewWorld(oreWorld(res, 300), economysim.WithSeed(5), economysim.WithObserver(&recorder{}), economysim.WithLogger(slog.New(log)))
	if err != nil {
		t.Fatalf("NewWorld() with an observer error = %v, want nil", err)
	}
	for tick := range 30 {
		if err := plain.Step(); err != nil {
			t.Fatalf("tick %d: Step() error = %v, want nil", tick, err)
		}
		if err := seen.Step(); err != nil {
			t.Fatalf("tick %d: Step() with an observer error = %v, want nil", tick, err)
		}
		if a, b := plain.Endowment(0, 0), seen.Endowment(0, 0); a != b {
			t.Fatalf("tick %d: stock = %d with an observer and a logger, %d without, want equal", tick, b, a)
		}
	}
	a, _ := plain.Snapshot()
	b, _ := seen.Snapshot()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("the snapshots of the observed and the plain world differ")
	}
}

// TestCatalogue_ResourceEvents verifies that the two events are declared with their
// module: rows of the table of 4.1 with the gate resources, in the list of declared
// events and no longer in notBuiltEvents (note N-126).
//
// Requirement: FR-RES-026, FR-RES-050
func TestCatalogue_ResourceEvents(t *testing.T) {
	t.Parallel()
	rows, _ := specRows(t)
	declared := declaredNames()
	for _, name := range []string{"EndowmentChanged", "EndowmentAdjusted"} {
		if !slices.Contains(declared, name) || slices.Contains(notBuiltEvents, name) || rows[name].gate != "resources" {
			t.Errorf("%s: declared = %t, listed as not built = %t, gate %q, want declared, not listed, gate resources",
				name, slices.Contains(declared, name), slices.Contains(notBuiltEvents, name), rows[name].gate)
		}
	}
}
