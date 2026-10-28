package economysim_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds the tests of spec-currency.md that Stage 1 owed (plan 2-23
// raised them, plan 2-8 D364 lists them): no wage field, and the reference
// basket as the upkeep at the reference prices. They are cheap and independent of
// the rebalance, so they run on whatever packs exist.

// wagePaths walks the type t and returns the path of every field whose name or
// whose type name contains "wage", looking through pointers, slices and arrays.
func wagePaths(t reflect.Type, path string, seen map[reflect.Type]bool) []string {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || seen[t] {
		return nil
	}
	seen[t] = true
	defer delete(seen, t)
	var out []string
	for i := range t.NumField() {
		f := t.Field(i)
		p := path + "." + f.Name
		ft := f.Type
		for ft.Kind() == reflect.Pointer || ft.Kind() == reflect.Slice || ft.Kind() == reflect.Array {
			ft = ft.Elem()
		}
		if strings.Contains(strings.ToLower(f.Name), "wage") || strings.Contains(strings.ToLower(ft.Name()), "wage") {
			out = append(out, p)
		}
		out = append(out, wagePaths(ft, p, seen)...)
	}
	return out
}

// TestConfig_NoWageField verifies that the configuration holds no wage: no field
// of Config or of its sections is named or typed as a wage, so a wage is what the
// market pays and the simulation measures (spec-currency.md 2.6). The one
// exception is the warm start's table of starting wages (SeedWage), which is the
// solver's output used as the pooled observed wage until a measured one exists
// (FR-VIA-050, FR-WEL-015), not a wage the simulation pays or reads as a rule;
// the exception is listed and must stay in the type.
//
// Requirement: FR-CUR-005
//
// Guard: a wage added to a role or a region as a configured rule would make the
// market's pay an input and break the measure the welfare and the spawner read.
func TestConfig_NoWageField(t *testing.T) {
	t.Parallel()
	allowed := []string{".WarmStart.Wages", ".WarmStart.Wages.Wage"}
	got := wagePaths(reflect.TypeOf(economysim.Config{}), "", map[reflect.Type]bool{})
	for _, p := range got {
		if !slices.Contains(allowed, p) {
			t.Errorf("Config has a wage field at %s, want none (FR-CUR-005: wages are measured, not configured)", p)
		}
	}
	for _, p := range allowed {
		if !slices.Contains(got, p) {
			t.Errorf("allowed wage field %s is not in Config any more: delete it from the list", p)
		}
	}
}

// basketCost restates FR-CUR-020 with no call to production code: the baseline
// upkeep (the world's list plus the region's extra upkeep) at the effective
// reference prices of the region, which the world reports.
func basketCost(w *economysim.World, names economysim.Names, cfg economysim.Config, region int) int64 {
	upkeep := slices.Concat(cfg.World.Upkeep, cfg.Regions[region].ExtraUpkeep)
	var sum int64
	for _, it := range upkeep {
		c := slices.Index(names.Commodities, it.Commodity)
		sum += int64(it.Quantity) * int64(w.ReferencePrice(economysim.RegionID(region), economysim.CommodityID(c)))
	}
	return sum
}

// TestBasket_IsUpkeepTimesReferencePrices verifies, on every pack, that the
// reference basket of each region is the baseline upkeep times the effective
// reference prices of the region (FR-CUR-020), within the BasketTolerance that
// validation allows (FR-CUR-022): the packs state their basket in round figures
// (1 000 for an upkeep of 999 on `one`), so the difference is at most
// tolerance * B_r / 10 000. A basket moved to twice its cost fails Validate with
// ErrBasketMismatch, so the cost the test restates is the cost validation reads.
//
// Requirement: FR-CUR-020, FR-CUR-022
//
// Guard: a rebalance that changes an upkeep line or a reference price without the
// basket (plan 2-8 R5) fails here before it fails a run.
func TestBasket_IsUpkeepTimesReferencePrices(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	packs := map[string][]string{"one-demand": packOneDemandCold}
	for name, layers := range packLayers {
		packs[name] = layers
	}
	for name, layers := range packs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, layers...)
			w, err := economysim.NewWorld(cfg)
			if err != nil {
				t.Fatalf("NewWorld(%s) error = %v, want nil", name, err)
			}
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot error = %v, want nil", err)
			}
			for i, r := range cfg.Regions {
				cost := basketCost(w, s.Names, cfg, i)
				diff := max(int64(r.ReferenceBasket)-cost, cost-int64(r.ReferenceBasket))
				if diff*10_000 > cfg.World.BasketTolerance*int64(r.ReferenceBasket) {
					t.Errorf("%s %s: reference basket = %d, upkeep at the reference prices = %d, want within %d bp", name, r.ID, r.ReferenceBasket, cost, cfg.World.BasketTolerance)
				}
			}
			bad := loadValidPack(t, layers...)
			bad.Regions[0].ReferenceBasket *= 2
			if err := bad.Validate(); !errors.Is(err, economysim.ErrBasketMismatch) {
				t.Errorf("Validate(%s with the first basket doubled) = %v, want ErrBasketMismatch", name, err)
			}
		})
	}
}

// TestBasket_WithinToleranceChangesNothing verifies that no simulation decision
// reads the reference basket: pack `one` with its basket moved 5% up and 5% down,
// inside the tolerance of 10% and validating, gives the same events and the same
// final snapshot over 50 ticks as the pack as it is. The configuration hash is in
// WorldCreated and in the snapshot and does differ (the basket is in the hash),
// so those two are left out of the comparison. The per-position state hash of
// testing.md 7.1.2 walks the configuration too, so it differs at tick 0 and cannot
// be used here.
//
// Requirement: FR-CUR-023
//
// Guard: a rule that read B_r (a wage target, a scale) would make a basket in
// round figures change prices, and the tolerance a lie.
func TestBasket_WithinToleranceChangesNothing(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	const ticks = 50
	run := func(basket func(int64) int64) ([]economysim.Event, economysim.Snapshot) {
		cfg := loadValidPack(t, packOne...)
		cfg.Regions[0].ReferenceBasket = economysim.Money(basket(int64(cfg.Regions[0].ReferenceBasket)))
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate(basket %d) = %v, want nil: the move must stay inside the tolerance", cfg.Regions[0].ReferenceBasket, err)
		}
		res := runPack(t, cfg, runSpec{Seed: 1, Ticks: ticks, Drive: driveStep, Observe: true})
		var events []economysim.Event
		for _, e := range res.Events {
			if _, created := e.(economysim.WorldCreated); !created {
				events = append(events, e)
			}
		}
		res.Final.ConfigHash = [32]byte{}
		return events, res.Final
	}
	baseEvents, baseFinal := run(func(b int64) int64 { return b })
	if len(baseEvents) == 0 {
		t.Fatalf("the unmoved run recorded no event, want the events of %d ticks to compare", ticks)
	}
	for name, move := range map[string]func(int64) int64{
		"up 5%":   func(b int64) int64 { return b * 105 / 100 },
		"down 5%": func(b int64) int64 { return b * 95 / 100 },
	} {
		events, final := run(move)
		if !reflect.DeepEqual(events, baseEvents) {
			t.Errorf("basket moved %s: the %d events of %d ticks differ from the unmoved run's %d, want them equal", name, len(events), ticks, len(baseEvents))
		}
		if !reflect.DeepEqual(final, baseFinal) {
			t.Errorf("basket moved %s: the snapshot after %d ticks differs from the unmoved run's, want it equal", name, ticks)
		}
	}
}
