package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// stepAllocBaseline is the number of allocations of one full Step of pack one
// after 50 ticks, measured under -race with the logger and observer at their
// no-op defaults and WithAudit off. Only gate R checks it: counts differ without
// -race. It is a baseline: raising it is a reviewed change with a reason (rule 8.2).
const stepAllocBaseline = 70

// moneyStepAllocBaseline is the same measure for pack one with the money supply
// module alone on (overlay/faucet-sunk.toml): 71, one above the off world. P1.7
// itself allocates nothing once the scratch has grown (TestP17Flows_AllocatesNothing),
// so the one is in the run the money makes (a different run: other trades and entrants),
// not traced further. Raising it is a reviewed change (rule 8.2).
const moneyStepAllocBaseline = 71

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
	for _, row := range []struct {
		name     string
		layers   []string
		baseline float64
	}{
		{"one", packOne, stepAllocBaseline},
		{"one+faucet-sunk", packOneFaucetSunk, moneyStepAllocBaseline},
	} {
		cfg := loadValidPack(t, row.layers...)
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
		if err != nil {
			t.Fatalf("%s: NewWorld error = %v, want nil", row.name, err)
		}
		for i := range 50 {
			if err := w.Step(); err != nil {
				t.Fatalf("%s: Step at tick %d error = %v, want nil", row.name, i, err)
			}
		}
		got := testing.AllocsPerRun(20, func() {
			if err := w.Step(); err != nil {
				t.Fatalf("%s: Step error = %v, want nil", row.name, err)
			}
		})
		t.Logf("%s: allocs per Step = %v (baseline %v)", row.name, got, row.baseline)
		if got > row.baseline {
			t.Errorf("%s: allocs per Step = %v, want <= %v; the constant is a baseline, raising it is a reviewed change with a reason (rule 8.2)", row.name, got, row.baseline)
		}
	}
}
