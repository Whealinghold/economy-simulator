package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestFork_Fiscal verifies FR-FIS-060 and the fork at N of CHK 8.1 through the public API
// on the governed warm five with the fiscal overlay: the original and two restores give
// the same events, snapshot, audit and checkpoint bytes after N ticks of {0, 7 (the head
// tax and the standing orders have acted and carry their remainders), 7 with a call
// between ticks that halves the scale of the purchase and pauses the payout}. The carries
// of the flows and of the standing orders, and the scales, are saved state; a paused flow
// stays paused.
//
// Requirement: FR-FIS-060, FR-CHK-048
//
// Guard: a restore that lost the carry of a standing order or of a tax, or put a paused
// flow back at 10 000, would place a different order or take a different sum at the first
// tick after it, and the events of the restored world would differ from the original's.
func TestFork_Fiscal(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	cfg := fiscalPackConfig(t, false)
	scale := func(w *economysim.World) {
		if err := w.SetFlowScales([]economysim.FlowScale{{Flow: "buy", BP: 5_000}, {Flow: "alms", BP: 0}}); err != nil {
			t.Errorf("SetFlowScales error = %v, want nil", err)
		}
	}
	for _, c := range []struct {
		name   string
		n      int
		before func(*economysim.World)
	}{
		{"before the first step", 0, nil},
		{"after the first taxes and orders", 7, nil},
		{"after a scale call between ticks", 7, scale},
	} {
		forkAtN(t, forkCase{Name: fmt.Sprintf("fiscal %s", c.name), Cfg: cfg, Seed: 1, N: c.n, M: forkM, Before: c.before,
			Opts:    func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} },
			Inspect: inspectFiscalFork(c.name)})
	}
}

// inspectFiscalFork returns the check that the run after the boundary exercised the
// module: the standing orders reported and the taxes applied, so that an equal run is not
// equal because nothing happened.
func inspectFiscalFork(name string) func(*testing.T, []economysim.Event, []economysim.Event) {
	return func(t *testing.T, restored, since []economysim.Event) {
		t.Helper()
		var ordered, taxed int
		for _, e := range restored {
			switch e := e.(type) {
			case economysim.TreasuryOrdered:
				ordered++
			case economysim.FiscalFlowApplied:
				if e.Kind == economysim.KindTax {
					taxed++
				}
			}
		}
		if ordered == 0 || taxed == 0 {
			t.Errorf("fiscal %s: the restored world delivered %d TreasuryOrdered and %d tax events in %d ticks, want both above 0", name, ordered, taxed, forkM)
		}
	}
}
