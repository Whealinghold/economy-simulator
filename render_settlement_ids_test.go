package economysim_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestOffSettlementIDs_OnlyRegionRowWithTheModuleOff verifies that the golden writer
// omits Class and ClassTarget of a RegionRow while settlements is off, and nothing
// else: another row's Class (StoreRow's) is written, so the golden of the storage
// module alone keeps "Class=0" and golden/off does not move.
//
// Requirement: FR-OBS-080
func TestOffSettlementIDs_OnlyRegionRowWithTheModuleOff(t *testing.T) {
	t.Parallel()
	off := economysim.Snapshot{Modules: []string{"storage"}}
	on := economysim.Snapshot{Modules: []string{"storage", "settlements"}}
	cases := []struct {
		name  string
		s     economysim.Snapshot
		row   reflect.Value
		field string
		skip  bool
	}{
		{"region class, off", off, reflect.ValueOf(economysim.RegionRow{}), "Class", true},
		{"region target, off", off, reflect.ValueOf(economysim.RegionRow{}), "ClassTarget", true},
		{"region households, off", off, reflect.ValueOf(economysim.RegionRow{}), "Households", false},
		{"region class, on", on, reflect.ValueOf(economysim.RegionRow{}), "Class", false},
		{"store class, off", off, reflect.ValueOf(economysim.StoreRow{}), "Class", false},
		{"store class, on", on, reflect.ValueOf(economysim.StoreRow{}), "Class", false},
	}
	for _, c := range cases {
		f, ok := c.row.Type().FieldByName(c.field)
		if !ok {
			t.Fatalf("%s: no field %s", c.name, c.field)
		}
		skip := offSettlementIDs(c.s, c.row)
		if got := skip != nil && skip(f); got != c.skip {
			t.Errorf("%s: skipped = %v, want %v", c.name, got, c.skip)
		}
	}
}
