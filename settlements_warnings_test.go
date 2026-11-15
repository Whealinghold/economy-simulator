package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestWarnings_SettlementsExactlyTheRule verifies, for each of the eight rules of
// spec-settlements.md 9.2, that a configuration built to break only that rule
// draws that warning and no other settlements warning, with its count. A warning
// that fires for a neighbouring rule's case would make the one-case-each table of
// TestWarnings_Settlements pass for the wrong reason.
//
// Requirement: FR-SET-028
func TestWarnings_SettlementsExactlyTheRule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		build func(c *economysim.Config)
		want  []string // rule@path, all the settlements warnings
	}{
		{"none on the base configuration", func(*economysim.Config) {}, nil},
		{"needs_unmet_at_start", func(c *economysim.Config) {
			c.Modules.Settlements.Classes[2].Flags = nil
			c.Modules.Settlements.Classes[1].Flags = nil
		}, []string{"needs_unmet_at_start@roles.grower.needs"}},
		{"ceiling_below_start", func(c *economysim.Config) {
			c.Modules.Settlements.Classes[2].Limits[0].Value = 20
		}, []string{"ceiling_below_start@roles.refiner.ceiling"}},
		{"dwell_below_window", func(c *economysim.Config) {
			c.Modules.Settlements.Dwell = 1
		}, []string{"dwell_below_window@modules.settlements.dwell"}},
		{"flag_unused", func(c *economysim.Config) {
			c.Modules.Settlements.Classes[1].Flags = append(c.Modules.Settlements.Classes[1].Flags, "hall")
		}, []string{"flag_unused@modules.settlements.flag.hall"}},
		{"limit_unused", func(c *economysim.Config) {
			c.Modules.Settlements.Classes[1].Limits = append(c.Modules.Settlements.Classes[1].Limits, economysim.ClassLimit{Name: "berths", Value: 3})
		}, []string{"limit_unused@modules.settlements.limit.berths"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := settlementsConfig()
			tt.build(&c)
			got := settlementsRule(c)
			slices.Sort(got)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Warnings() settlements rules = %v, want exactly %v", got, tt.want)
			}
		})
	}
}

// TestWarnings_SettlementsOffAndEmpty verifies that the module off, and the module on
// with no class (which fails validation), raise no settlements warning and do not
// panic: Warnings is callable on any configuration.
//
// Requirement: FR-SET-001, FR-SET-028
func TestWarnings_SettlementsOffAndEmpty(t *testing.T) {
	t.Parallel()
	off := referenceConfig()
	if got := settlementsRule(off); len(got) != 0 {
		t.Errorf("Warnings() with the module off = %v, want no settlements warning", got)
	}
	empty := referenceConfig()
	empty.Modules.Settlements = &economysim.Settlements{}
	if got := settlementsRule(empty); len(got) != 0 {
		t.Errorf("Warnings() with the module on and no class = %v, want none", got)
	}
}

// TestWarnings_SettlementsStoreClassUsesAFlag verifies that a flag only a store class
// names (StoreClass.Needs) is not flag_unused, and that it is once the store class
// stops naming it.
//
// Requirement: FR-SET-028, FR-SET-021
func TestWarnings_SettlementsStoreClassUsesAFlag(t *testing.T) {
	t.Parallel()
	c := storageConfig()
	c.Modules.Settlements = settlementsConfig().Modules.Settlements
	c.Modules.Storage.Classes[0].Needs = "fair"
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	if got := settlementsRule(c); slices.Contains(got, "flag_unused@modules.settlements.flag.fair") {
		t.Errorf("Warnings() = %v, want fair counted as used by the store class", got)
	}
	c.Modules.Storage.Classes[0].Needs = ""
	if got := settlementsRule(c); !slices.Contains(got, "flag_unused@modules.settlements.flag.fair") {
		t.Errorf("Warnings() = %v, want flag_unused for fair once nothing names it", got)
	}
}

// TestWarnings_SettlementsSkipOutsideNodes verifies that a region that is an outside
// node has no census, so a threshold warning counts only the regions that can hold
// households.
//
// Requirement: FR-SET-028, FR-SET-006
func TestWarnings_SettlementsSkipOutsideNodes(t *testing.T) {
	t.Parallel()
	c := externalConfig()
	m := economysim.DefaultSettlements()
	m.Classes = []economysim.SettlementClass{{ID: "a"}, {ID: "b", Enter: 5}}
	c.Modules.Settlements = &m
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	want := int64(len(c.Regions) - len(c.Modules.External.Nodes))
	var got int64 = -1
	for _, w := range c.Warnings() {
		if w.Rule == "band_one_household" {
			got = w.Count
		}
	}
	if want < 1 || got != want {
		t.Errorf("band_one_household Count = %d, want %d (the regions %d less the outside nodes %d)", got, want, len(c.Regions), len(c.Modules.External.Nodes))
	}
}

// TestValidateSettlements_DistinctNamesAreBounded verifies that the union of flag
// names, and of limit names, over all classes is bounded by MaxClassFlags and
// MaxClassLimits though every single class is within it: 16 classes of 5 names each
// name 80 distinct ones, over 64.
//
// Requirement: FR-SET-004
func TestValidateSettlements_DistinctNamesAreBounded(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"flags", "limits"} {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			m := economysim.DefaultSettlements()
			for k := range economysim.MaxSettlementClasses {
				cl := economysim.SettlementClass{ID: "c" + strconv.Itoa(k), Enter: int64(k)}
				for j := range 5 {
					name := "n" + strconv.Itoa(k) + "_" + strconv.Itoa(j)
					if key == "flags" {
						cl.Flags = append(cl.Flags, name)
					} else {
						cl.Limits = append(cl.Limits, economysim.ClassLimit{Name: name, Value: 1})
					}
				}
				m.Classes = append(m.Classes, cl)
			}
			c.Modules.Settlements = &m
			err := c.Validate()
			var ve *economysim.ValidationError
			if !errors.Is(err, economysim.ErrTooMany) || !errors.As(err, &ve) ||
				strings.Join(ve.Path, ".") != strings.Join(setPath(key), ".") || ve.Value != "80" {
				t.Errorf("Validate() = %v, want ErrTooMany at %v with value 80", err, setPath(key))
			}
		})
	}
}

// TestWarnings_SettlementsUnknownFloorIsNoFloor verifies that Warnings, which runs on
// a configuration that may not validate, reads a row whose Floor names no class as a
// row with no floor, and does not panic.
//
// Guard: Warnings is documented as callable before Validate; a floor index taken
// from a failed lookup would read outside the class list.
func TestWarnings_SettlementsUnknownFloorIsNoFloor(t *testing.T) {
	t.Parallel()
	c := settlementsConfig()
	want := settlementsRule(c)
	c.Modules.Settlements.Regions[0].Floor = "nowhere"
	if err := c.Validate(); !errors.Is(err, economysim.ErrUnknownSettlementClass) {
		t.Fatalf("Validate() = %v, want ErrUnknownSettlementClass for the scenario to mean anything", err)
	}
	if got := settlementsRule(c); !slices.Equal(got, want) {
		t.Errorf("Warnings() with an unknown floor = %v, want %v (no floor)", got, want)
	}
}
