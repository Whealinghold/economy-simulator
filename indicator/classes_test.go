package indicator_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// TestClasses_UnknownAndOverlap verifies the class table of
// spec-indicators.md 4.4: a class returns its sorted commodities, a class no
// commodity carries is ErrUnknownClass, a commodity of two classes is in both
// classes and once in All, the table is built from the configuration and so
// names no class itself, and the slices returned are copies.
//
// Requirement: FR-IND-029
func TestClasses_UnknownAndOverlap(t *testing.T) {
	t.Parallel()
	cfg := economysim.Config{Commodities: []economysim.Commodity{
		{ID: "a", Classes: []string{"x", "y"}}, // IDs 0 to 3
		{ID: "b", Classes: []string{"y"}},
		{ID: "c"},
		{ID: "d", Classes: []string{"x", "z"}},
	}}
	cl := indicator.NewClasses(cfg)
	tests := []struct {
		name string
		want []economysim.CommodityID
	}{
		{"x", []economysim.CommodityID{0, 3}},
		{"y", []economysim.CommodityID{0, 1}},
		{"z", []economysim.CommodityID{3}},
	}
	for _, tt := range tests {
		got, err := cl.Of(tt.name)
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("Of(%q) = %v, %v, want %v, nil", tt.name, got, err, tt.want)
		}
	}
	for _, name := range []string{"w", "", "X"} {
		if got, err := cl.Of(name); !errors.Is(err, indicator.ErrUnknownClass) || got != nil {
			t.Errorf("Of(%q) = %v, %v, want nil, ErrUnknownClass", name, got, err)
		}
	}
	if got, want := cl.All(), []economysim.CommodityID{0, 1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("All() = %v, want %v: every commodity once, classed or not", got, want)
	}
	if got, want := cl.Names(), []string{"x", "y", "z"}; !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
	x, _ := cl.Of("x")
	x[0] = 9
	if again, _ := cl.Of("x"); again[0] != 0 {
		t.Errorf("Of(x) after the caller changed a result = %v, want the table unchanged", again)
	}
	all := cl.All()
	all[0] = 9
	if cl.All()[0] != 0 {
		t.Errorf("All() after the caller changed a result = %v, want the table unchanged", cl.All())
	}
	var zero indicator.Classes
	if _, err := zero.Of("x"); !errors.Is(err, indicator.ErrUnknownClass) || len(zero.All()) != 0 {
		t.Errorf("zero Classes: Of error = %v, All = %v, want ErrUnknownClass and no commodity", err, zero.All())
	}
}

// TestClasses_OfThePack verifies that the classes of the reference pack come
// from its data: the classes food, fuel, material and tool, in the order of the
// commodities, with no class name in the code.
//
// Requirement: FR-IND-029
func TestClasses_OfThePack(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadOne(t)
	cl := indicator.NewClasses(cfg)
	for _, name := range cl.Names() {
		set, err := cl.Of(name)
		if err != nil || len(set) == 0 {
			t.Fatalf("Of(%q) = %v, %v, want a non-empty set", name, set, err)
		}
		for _, id := range set {
			if !slices.Contains(cfg.Commodities[id].Classes, name) {
				t.Errorf("Of(%q) holds commodity %q, which does not carry the tag", name, cfg.Commodities[id].ID)
			}
		}
	}
	if len(cl.All()) != len(cfg.Commodities) {
		t.Errorf("len(All()) = %d, want %d commodities", len(cl.All()), len(cfg.Commodities))
	}
}
