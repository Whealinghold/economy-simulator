package economysim

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ownuse"
)

// TestBlockReasonAt_UnknownReasonIsTheLast verifies that a reason outside the
// staged list reads as the last entry ("any other"), and that each listed
// reason reads as its own index.
//
// Requirement: FR-SUB-023, FR-SUB-042
func TestBlockReasonAt_UnknownReasonIsTheLast(t *testing.T) {
	t.Parallel()
	for i, r := range blockReasons {
		if got := blockReasonAt(r); got != i {
			t.Errorf("blockReasonAt(%v) = %d, want %d", r, got, i)
		}
	}
	if got, want := blockReasonAt(Reason("unlisted")), len(blockReasons)-1; got != want {
		t.Errorf("blockReasonAt(unknown) = %d, want %d", got, want)
	}
}

// TestFigures_UnformableFigureSaturates verifies that a figure that cannot be
// formed (no good, a cost beyond int64, a negative balance) reads as the
// largest, as Step reads it, and that a formable one is the cover plus what the
// money buys.
//
// Requirement: FR-SUB-011, FR-SUB-012, NFR-SUB-002
func TestFigures_UnformableFigureSaturates(t *testing.T) {
	t.Parallel()
	s := &subsistence{params: ownuse.Params{Enter: 3, Exit: 6, Max: 100}}
	ok := []ownuse.Good{{Stock: 4, Upkeep: 2, Price: 5}}
	huge := []ownuse.Good{{Stock: 4, Upkeep: 2, Price: math.MaxInt64}}
	for _, tc := range []struct {
		name      string
		goods     []ownuse.Good
		hungry    bool
		money     int64
		cover, sc int64
	}{
		{"no good", nil, false, 50, 100, 100},
		{"cost overflows", huge, false, 50, 2, 100},
		{"negative money", ok, false, -1, 2, 100},
		{"formable", ok, false, 50, 2, 7}, // 2 held + 50/10 afforded
		{"hungry affords nothing", ok, true, 50, 2, 2},
	} {
		cover, secure := s.figures(tc.goods, tc.hungry, tc.money)
		if cover != tc.cover || secure != tc.sc {
			t.Errorf("%s: figures = (%d, %d), want (%d, %d)", tc.name, cover, secure, tc.cover, tc.sc)
		}
	}
}

// TestOwnCovs_SumsOutputsAndSaturates verifies that the coverage of a good is
// the sum over the non-discard outputs of the commodity, that a sum beyond int64
// saturates, and that a discard or another commodity adds nothing.
//
// Requirement: FR-SUB-039, NFR-SUB-002
func TestOwnCovs_SumsOutputsAndSaturates(t *testing.T) {
	t.Parallel()
	r := RecipeView{Duration: 1, Effective: 10_000, Outputs: []RecipeOutput{
		{Commodity: cStaple, Quantity: 2},
		{Commodity: cStaple, Quantity: 1},
		{Commodity: cStaple, Quantity: 9, Discard: true},
	}}
	dst := make([]int64, 1)
	r.ownCovs(dst, []UpkeepRow{{Commodity: cStaple, Quantity: 1}})
	if dst[0] != 30_000 {
		t.Errorf("sum of the outputs = %d, want 30000", dst[0])
	}
	r.Outputs = []RecipeOutput{{Commodity: cStaple, Quantity: math.MaxInt64}, {Commodity: cStaple, Quantity: math.MaxInt64}}
	r.ownCovs(dst, []UpkeepRow{{Commodity: cStaple, Quantity: 1}})
	if dst[0] != math.MaxInt64 {
		t.Errorf("saturated sum = %d, want MaxInt64", dst[0])
	}
}

// TestUpkeepCommodities_BaseAndRegionAscendingOnce verifies that the commodities
// of a role's base and regional upkeep come out ascending with no repeat.
//
// Requirement: FR-SUB-010
func TestUpkeepCommodities_BaseAndRegionAscendingOnce(t *testing.T) {
	t.Parallel()
	row := &roleRow{
		upkeep:       []itemRow{{commodity: 3}, {commodity: 1}},
		regionUpkeep: []regionUpkeep{{region: 0, items: []itemRow{{commodity: 3}, {commodity: 2}}}},
	}
	got := upkeepCommodities(row)
	want := []CommodityID{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("upkeepCommodities = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("upkeepCommodities = %v, want %v", got, want)
		}
	}
}
