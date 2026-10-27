package config_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// demandDoc is a [modules.demand] table written in the form of spec-demand.md
// 3.2 with the rows in the order that is not the canonical one, so the test sees
// the loader's order.
const demandDoc = minimal + `
[modules.demand.want.treat]
items = [ { commodity = "sweet_b", weight = 5000 }, { commodity = "sweet_a" } ]
quantity = 2
period = 5
rank = 2
elasticity = 20000
ceiling = 12000
substitution = 3
label = "A treat"
[modules.demand.want.comfort]
items = [ { commodity = "cloth", weight = 10000 } ]
quantity = 1
[modules.demand.taste.worker.treat]
scale = 5000
[modules.demand.taste.worker.comfort]
[modules.demand.taste.artisan.comfort]
scale = 15000
[modules.demand.region_taste.north.comfort]
scale = 15000
[modules.demand.region_taste.north.treat]
scale = 0
`

// TestLoad_DemandKeys verifies that the keys of spec-demand.md 3.2 load the
// tables of the module: [modules.demand] switches it on, want.<id>,
// taste.<role>.<want> and region_taste.<region>.<want> fill the three lists in
// canonical order, and a key that is not one of them, a missing required key
// and a value of the wrong type are rejected with the sentinel and the path of
// the key.
//
// Requirement: FR-DEM-043, FR-CFG-060, FR-CFG-098, FR-CFG-019, FR-CFG-008
func TestLoad_DemandKeys(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(demandDoc)
	if err != nil {
		t.Fatalf("Parse(demandDoc) error = %v, want nil", err)
	}
	want := &economysim.Demand{
		Wants: []economysim.Want{
			{ID: "comfort", Items: []economysim.WantItem{{Commodity: "cloth", Weight: 10_000}}, Quantity: 1, Period: 1,
				Elasticity: 15_000, Ceiling: 15_000, Substitution: 2},
			// The items are in the order written: canonical order is the world's.
			{ID: "treat", Items: []economysim.WantItem{{Commodity: "sweet_b", Weight: 5_000}, {Commodity: "sweet_a", Weight: 10_000}},
				Quantity: 2, Period: 5, Rank: 2, Elasticity: 20_000, Ceiling: 12_000, Substitution: 3, Label: "A treat"},
		},
		Tastes: []economysim.Taste{
			{Role: "artisan", Want: "comfort", Scale: 15_000},
			{Role: "worker", Want: "comfort", Scale: 10_000},
			{Role: "worker", Want: "treat", Scale: 5_000},
		},
		RegionTastes: []economysim.RegionTaste{
			{Region: "north", Want: "comfort", Scale: 15_000},
			{Region: "north", Want: "treat", Scale: 0},
		},
	}
	if !reflect.DeepEqual(got.Modules.Demand, want) {
		t.Errorf("Parse(demandDoc).Modules.Demand = %+v, want %+v", got.Modules.Demand, want)
	}
	if on, _ := parseStrings(minimal + "[modules.demand]\n"); on.Modules.Demand == nil || !reflect.DeepEqual(*on.Modules.Demand, economysim.Demand{}) {
		t.Errorf("Parse([modules.demand]).Modules.Demand = %+v, want a module that is on and empty", on.Modules.Demand)
	}
	if off, _ := parseStrings(minimal); off.Modules.Demand != nil {
		t.Errorf("Parse(minimal).Modules.Demand = %+v, want nil: the module is off", off.Modules.Demand)
	}
	const wantOK = "items = [ { commodity = \"a\" } ]\nquantity = 1\n"
	tests := []struct {
		name string
		docs []string
		rule error
		path []string
	}{
		{"unknown key in the module table", []string{minimal + "[modules.demand]\nbogus = 1\n"}, economysim.ErrUnknownKey, []string{"modules", "demand", "bogus"}},
		{"unknown key in a want", []string{minimal + "[modules.demand.want.w]\n" + wantOK + "bogus = 1\n"}, economysim.ErrUnknownKey, []string{"modules", "demand", "want", "w", "bogus"}},
		{"unknown key in an item", []string{minimal + "[modules.demand.want.w]\nquantity = 1\nitems = [ { commodity = \"a\", bogus = 1 } ]\n"}, economysim.ErrUnknownKey, []string{"modules", "demand", "want", "w", "items", "[0]", "bogus"}},
		{"unknown key in a taste", []string{minimal + "[modules.demand.taste.r.w]\nbogus = 1\n"}, economysim.ErrUnknownKey, []string{"modules", "demand", "taste", "r", "w", "bogus"}},
		{"unknown key in a region taste", []string{minimal + "[modules.demand.region_taste.g.w]\nweight = 1\n"}, economysim.ErrUnknownKey, []string{"modules", "demand", "region_taste", "g", "w", "weight"}},
		{"wants in the plural", []string{minimal + "[modules.demand.wants.w]\n" + wantOK}, economysim.ErrUnknownKey, []string{"modules", "demand", "wants"}},
		{"bare demand table", []string{minimal + "[demand]\n"}, economysim.ErrUnknownKey, []string{"demand"}},
		{"a module that is not built", []string{minimal + "[modules.credit]\n"}, economysim.ErrUnknownKey, []string{"modules", "credit"}},
		{"missing quantity", []string{minimal + "[modules.demand.want.w]\nitems = [ { commodity = \"a\" } ]\n"}, economysim.ErrMissingField, []string{"modules", "demand", "want", "w", "quantity"}},
		{"missing items", []string{minimal + "[modules.demand.want.w]\nquantity = 1\n"}, economysim.ErrMissingField, []string{"modules", "demand", "want", "w", "items"}},
		{"missing item commodity", []string{minimal + "[modules.demand.want.w]\nquantity = 1\nitems = [ { weight = 5 } ]\n"}, economysim.ErrMissingField, []string{"modules", "demand", "want", "w", "items", "[0]", "commodity"}},
		{"float quantity", []string{minimal + "[modules.demand.want.w]\nitems = [ { commodity = \"a\" } ]\nquantity = 1.5\n"}, economysim.ErrNotInteger, []string{"modules", "demand", "want", "w", "quantity"}},
		{"string scale", []string{minimal + "[modules.demand.taste.r.w]\nscale = \"5\"\n"}, economysim.ErrWrongType, []string{"modules", "demand", "taste", "r", "w", "scale"}},
		{"scalar where a want table", []string{minimal + "[modules.demand.taste.r]\nw = 1\n"}, economysim.ErrWrongType, []string{"modules", "demand", "taste", "r", "w"}},
		{"integer where the module table", []string{minimal + "[modules]\ndemand = 1\n"}, economysim.ErrWrongType, []string{"modules", "demand"}},
		{"table then scalar across documents", []string{minimal + "[modules.demand]\n", "[modules]\ndemand = 1\n"}, economysim.ErrMergeConflict, []string{"modules", "demand"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(tt.docs...)
			if err == nil || !errors.Is(err, tt.rule) || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Parse(%q) errors = %s, want %v at %q", tt.docs, describe(err), tt.rule, tt.path)
			}
		})
	}
}

// TestLoad_DemandDefaults verifies that the loader starts each row from its
// constructor (spec-demand.md 3.2.2b, 9): an absent scale or weight is 10 000, an
// absent period is 1, an absent elasticity and ceiling 15 000, an absent
// substitution 2, and a scale of 0 that the file gives is kept, as a row that
// removes the want.
//
// Requirement: FR-CFG-007, FR-DEM-041, FR-DEM-043
func TestLoad_DemandDefaults(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(minimal + `
[modules.demand.want.w]
items = [ { commodity = "a" } ]
quantity = 3
[modules.demand.taste.r.w]
[modules.demand.taste.s.w]
scale = 0
[modules.demand.region_taste.g.w]
[modules.demand.region_taste.h.w]
scale = 0
`)
	if err != nil {
		t.Fatalf("Parse error = %v, want nil", err)
	}
	d := got.Modules.Demand
	if want := (economysim.Want{ID: "w", Items: []economysim.WantItem{{Commodity: "a", Weight: 10_000}}, Quantity: 3,
		Period: 1, Elasticity: 15_000, Ceiling: 15_000, Substitution: 2}); !reflect.DeepEqual(d.Wants[0], want) {
		t.Errorf("want w = %+v, want %+v", d.Wants[0], want)
	}
	scales := []int64{d.Tastes[0].Scale, d.Tastes[1].Scale, d.RegionTastes[0].Scale, d.RegionTastes[1].Scale}
	if want := []int64{10_000, 0, 10_000, 0}; !reflect.DeepEqual(scales, want) {
		t.Errorf("scales of tastes r, s and region tastes g, h = %v, want %v: absent is 10 000, an explicit 0 is kept", scales, want)
	}
}
