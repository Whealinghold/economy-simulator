package economysim_test

import (
	"os"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// TestBeliefParams_TwoSettings verifies that the belief parameters built from
// the validated packs one and second equal their World fields and pass the
// bounds and the ordering rules of spec-beliefs.md 11.
//
// Requirement: FR-BEL-004
//
// Guard: architecture.md 9.4, the construction must work for any setting.
func TestBeliefParams_TwoSettings(t *testing.T) {
	t.Parallel()
	fsys := os.DirFS("testdata/chains")
	packs := map[string][]string{
		"one":    {"base.toml", "one.toml", "warm-one.toml"},
		"second": {"second/second.toml", "second/warm-second.toml"},
	}
	for name, layers := range packs {
		raw, err := config.LoadFS(fsys, layers...)
		if err != nil {
			t.Fatalf("pack %s: LoadFS error = %v, want nil", name, err)
		}
		if err := raw.Validate(); err != nil {
			t.Fatalf("pack %s: Validate error = %v, want nil", name, err)
		}
		w := raw.World
		p := economysim.BeliefParams(w)
		want := [16]int64{w.BeliefStep, w.BeliefAdjustProb, w.BeliefSignificance, w.BeliefEvidenceShare,
			w.BeliefStockLow, w.BeliefStockHigh, w.BeliefMaxAge, w.BeliefWidthInit, w.BeliefWidthMin,
			w.BeliefWidthMax, w.BeliefWidthStep, w.CostFloor, w.CostCeiling, w.RefBandLow, w.RefBandHigh,
			economysim.MaxPrice}
		if p != want {
			t.Errorf("pack %s: params = %v, want %v", name, p, want)
		}
		const step, stockLow, stockHigh, wInit, wMin, wMax, cFloor, cCeil, rLow, rHigh = 0, 4, 5, 7, 8, 9, 11, 12, 13, 14
		switch {
		case p[step] < 1 || p[step] > 2_500,
			p[stockLow] >= p[stockHigh],
			p[wMin] > p[wInit] || p[wInit] > p[wMax] || p[wMax] >= 10_000,
			p[cFloor] > p[cCeil],
			p[rLow] > 10_000 || p[rHigh] < 10_000:
			t.Errorf("pack %s: params %v break the bounds of BEL 11", name, p)
		}
	}
}
