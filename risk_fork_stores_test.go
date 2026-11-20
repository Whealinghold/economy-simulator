package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// firstStoreStrike returns the tick of the first StoreLoss of a bare run of cfg, or -1.
func firstStoreStrike(t testing.TB, cfg economysim.Config, seed uint64) int {
	t.Helper()
	res := runPack(t, cfg, runSpec{Seed: seed, Ticks: 120, Drive: driveStep, Observe: true})
	for _, e := range res.Events {
		if s, ok := e.(economysim.StoreLoss); ok {
			return int(s.Tick)
		}
	}
	return -1
}

// TestFork_RiskStores verifies the fork-at-N of the store pack with the risk module on
// (FR-RSK-033, FR-RSK-035; RSK-S6): before the first step, one tick after the first strike of a
// store row (P8.1a), after a SetStoreRiskScales call that patrols the stores, and much later, the
// original and two restores give the same events, snapshot, audit, checkpoint bytes and log
// records for 20 ticks, so a restored world draws, strikes and takes the scale as the
// uninterrupted one. The cell after the call also checks the call is in the checkpoint: the
// restored world strikes nothing.
//
// Requirement: FR-RSK-033, FR-RSK-035, FR-RSK-029, FR-CHK-050
func TestFork_RiskStores(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := riskGoldenStore(t).cfg
	struck := firstStoreStrike(t, cfg, 1)
	if struck < 0 {
		t.Fatal("no StoreLoss in 120 ticks of the store pack, want a strike to fork after")
	}
	patrol := func(w *economysim.World) {
		if err := w.SetStoreRiskScales(riskStoreScales(t, cfg, 0)); err != nil {
			t.Fatalf("SetStoreRiskScales(0) error = %v, want nil", err)
		}
	}
	for _, tc := range []struct {
		name   string
		n      int
		before func(*economysim.World)
		quiet  bool
	}{
		{"before the first step", 0, nil, false},
		{"after a strike of P8.1a", struck + 1, nil, false},
		{"after a scale call", struck + 1, patrol, true},
		{"much later", 100, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "risk stores", Cfg: cfg, Seed: 1, N: tc.n, M: 20, Logs: true, Before: tc.before,
				Inspect: func(t *testing.T, restored, _ []economysim.Event) {
					hit := slices.ContainsFunc(restored, func(e economysim.Event) bool { _, ok := e.(economysim.StoreLoss); return ok })
					if tc.quiet && hit {
						t.Errorf("N %d: the restored world struck a store after the patrol, want the scale of 0 in the checkpoint", tc.n)
					}
				},
			})
		})
	}
}

// TestFork_RiskRouteScaleCall verifies the fork-at-N of the route pack across a
// SetRouteRiskScales call: the shipper dispatches at tick 1, so a call after the first step that
// patrols the routes leaves the shipments on the road with the share they were given, and the
// original and two restores of the checkpoint after the call agree for 20 ticks (the scales are
// in tag 28, the shares in tag 17).
//
// Requirement: FR-RSK-029, FR-RSK-030, FR-RSK-035, FR-CHK-050
func TestFork_RiskRouteScaleCall(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	g := riskGoldenTwo(t)
	forkAtN(t, forkCase{
		Name: "risk route scale call", Cfg: g.cfg, Seed: 1, N: 2, M: 20, Logs: true,
		Opts: func() []economysim.Option { return g.opts },
		Before: func(w *economysim.World) {
			if err := w.SetRouteRiskScales(riskRouteScales(t, g.cfg, 0)); err != nil {
				t.Fatalf("SetRouteRiskScales(0) error = %v, want nil", err)
			}
		},
		Inspect: func(t *testing.T, restored, _ []economysim.Event) {
			if lostOf(restored) == 0 {
				t.Errorf("the restored world lost no unit after the call, want the strikes made before it to be taken at the arrival")
			}
		},
	})
}
