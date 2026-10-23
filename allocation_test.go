package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// stepAllocBaseline is the number of allocations of one full Step of pack one
// after 50 ticks, measured under -race (coverage adds no allocation: 70 either way) with
// the logger and observer at their no-op defaults and WithAudit off. It is a
// baseline: raising it is a reviewed change with a reason (rule 8.2).
const stepAllocBaseline = 113

// TestStep_AllocationsStayAtTheBaseline checks that a full Step of pack one
// allocates no more than the baseline. Not parallel: testing.AllocsPerRun
// panics in a parallel test. Allocation counts differ under -race, so the test
// is skipped (with the reason) without it; gate R runs it.
//
// Requirement: NFR-WLD-001, NFR-OBS-002.
// Guard: an allocation added to the per-tick path.
func TestStep_AllocationsStayAtTheBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end: builds pack one and steps it 50 ticks; skipped under -short")
	}
	if !raceEnabled {
		t.Skipf("baseline measured under -race (race=%v); counts differ without it (testing.md 5.7)", raceEnabled)
	}
	cfg := loadValidPack(t, packOne...)
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	for i := range 50 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d error = %v, want nil", i, err)
		}
	}
	got := testing.AllocsPerRun(20, func() {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	})
	t.Logf("allocs per Step = %v (baseline %d)", got, stepAllocBaseline)
	if got > stepAllocBaseline {
		t.Errorf("allocs per Step = %v, want <= %d; the constant is a baseline, raising it is a reviewed change with a reason (rule 8.2)", got, stepAllocBaseline)
	}
}
