package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestSnapshot_WantBoundaryRowsFollowBuiltIn verifies that a world with the
// demand module on lists, after the 1 + 7C built-in boundary rows, one
// want_consumption sink row per commodity in ascending order at balance 0 (the
// module posts nothing yet), and that a world with the module off lists none.
//
// Requirement: FR-LED-050, FR-DEM-001
func TestSnapshot_WantBoundaryRowsFollowBuiltIn(t *testing.T) {
	t.Parallel()
	rows := func(cfg economysim.Config) []economysim.BoundaryRow {
		w, err := economysim.NewWorld(cfg)
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		return s.Boundary
	}
	off := rows(referenceConfig())
	on := rows(demandConfig())
	c := len(referenceConfig().Commodities)
	if len(off) != 1+7*c || len(on) != 1+8*c {
		t.Fatalf("snapshot boundary rows = %d off, %d on, want %d and %d", len(off), len(on), 1+7*c, 1+8*c)
	}
	for i := range off {
		if off[i] != on[i] {
			t.Errorf("built-in row %d = %+v with the module on, %+v off, want equal", i, on[i], off[i])
		}
		if off[i].Reason == economysim.ReasonWantConsumption {
			t.Errorf("row %d of the module-off world is want_consumption", i)
		}
	}
	for i, r := range on[len(off):] {
		want := economysim.BoundaryRow{Reason: economysim.ReasonWantConsumption, Kind: economysim.KindSink, Commodity: economysim.CommodityID(i)}
		if r != want {
			t.Errorf("module row %d = %+v, want %+v", i, r, want)
		}
	}
}
