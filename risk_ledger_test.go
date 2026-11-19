package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestLedger_LossIsGoodsSinkOnly verifies that the risk module on opens one
// boundary account of the reason loss for each commodity and none for money, that
// every one is a sink, and that a world with the module off has none: the reason
// is posted by the module only with it on and never moves money (spec-risk.md 7.1,
// 7.4; spec-ledger.md 7.2).
//
// Requirement: FR-RSK-022
func TestLedger_LossIsGoodsSinkOnly(t *testing.T) {
	t.Parallel()
	count := func(c economysim.Config) (goods, money int, sinks bool) {
		w, err := economysim.NewWorld(c)
		if err != nil {
			t.Fatalf("NewWorld() error = %v, want nil", err)
		}
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		sinks = true
		for _, b := range snap.Boundary {
			if b.Reason != economysim.ReasonLoss {
				continue
			}
			if b.Commodity < 0 {
				money++
			} else {
				goods++
			}
			sinks = sinks && b.Kind == economysim.KindSink
		}
		return goods, money, sinks
	}
	on := riskConfig()
	goods, money, sinks := count(on)
	if want := len(on.Commodities); goods != want || money != 0 || !sinks {
		t.Errorf("module on: loss boundary rows goods=%d money=%d all sinks=%v, want goods=%d money=0 all sinks", goods, money, sinks, want)
	}
	off := riskConfig()
	off.Modules.Risk = nil
	if goods, money, _ := count(off); goods != 0 || money != 0 {
		t.Errorf("module off: loss boundary rows goods=%d money=%d, want none", goods, money)
	}
}
