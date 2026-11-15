package technology_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/technology"
)

// TestStep_OperationCounts verifies the cost of spec-technology.md 6.6 by counting reads,
// not by timing: a step over R regions and two fields, one with diffusion, reads the
// distance at most R*R times for that field and never for the field without it, and a
// step with no diffusion field reads it never.
//
// Requirement: NFR-TEC-004
func TestStep_OperationCounts(t *testing.T) {
	t.Parallel()
	const regions = 20
	run := func(fields []technology.Field) int {
		reads := 0
		dist := func(a, b int) (int64, bool) { reads++; return int64(10 + a + b), true }
		in := technology.Input{
			Cells:         make([]technology.Cell, regions*len(fields)),
			Counters:      make([]int64, regions*len(fields)),
			TrendCarry:    make([]int64, len(fields)),
			Fields:        fields,
			Regions:       regions,
			TicksPerYear:  365,
			MaxExperience: 1 << 50,
			Dist:          dist,
		}
		in.Cells[3].Advance = 500
		var scratch technology.Scratch
		if !technology.Step(in, &scratch, make([]technology.Result, regions*len(fields))) {
			t.Fatalf("Step() = false, want true")
		}
		return reads
	}
	still := technology.Field{CapBP: 5_000, TrendBP: 100}
	moves := technology.Field{CapBP: 5_000, CatchUpBP: 2_000, Reach: 100}
	if got := run([]technology.Field{still, still}); got != 0 {
		t.Errorf("distance reads with no diffusion field = %d, want 0", got)
	}
	if got := run([]technology.Field{still, moves}); got == 0 || got > regions*regions {
		t.Errorf("distance reads with one diffusion field = %d, want 1 to %d (R*R)", got, regions*regions)
	}
}
