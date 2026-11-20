package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// firstStruckDispatch returns the tick of the first dispatch with a loss share above 0 in a bare
// run of cfg, or -1.
func firstStruckDispatch(t testing.TB, cfg economysim.Config, opts []economysim.Option, seed uint64) int {
	t.Helper()
	res := runPack(t, cfg, runSpec{Seed: seed, Ticks: 120, Drive: driveStep, Observe: true, Extra: opts})
	for _, e := range res.Events {
		if d, ok := e.(economysim.ShipmentDispatched); ok && d.LossBP > 0 {
			return int(d.Tick)
		}
	}
	return -1
}

// TestScenarioRSK6_CheckpointWithStrikesInFlight is RSK-S6 (S131, spec-risk.md 16, FR-RSK-035):
// on the route pack, for seeds 1 to the scenario count, the boundary is taken one tick after the
// first dispatch that carries a loss share, so a struck shipment is on the road, and the original
// and two restores give the same events, snapshot, audit, checkpoint bytes and log records for 20
// ticks. Exact: the restored world delivers the arrival of the struck shipment that was in flight
// at the boundary, so the share it was given at the dispatch survived the checkpoint.
//
// Requirement: FR-RSK-035, FR-RSK-033, FR-CHK-050
//
// Guard: the loss share is the one field of a shipment the risk module adds to the record; a
// restore that dropped it would lose a cargo's strike or take a new draw for it.
func TestScenarioRSK6_CheckpointWithStrikesInFlight(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	g := riskGoldenTwo(t)
	for _, seed := range seedList(scenarioSeedCount()) {
		n := firstStruckDispatch(t, g.cfg, g.opts, seed)
		if n < 0 {
			t.Errorf("seed %d: no dispatch with a loss share in 120 ticks, want a struck shipment to checkpoint", seed)
			continue
		}
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			var inFlight int
			forkAtN(t, forkCase{
				Name: "risk struck in flight", Cfg: g.cfg, Seed: seed, N: n + 1, M: 20, Logs: true,
				Opts: func() []economysim.Option { return g.opts },
				Before: func(w *economysim.World) {
					s, err := w.Snapshot()
					if err != nil {
						t.Fatalf("Snapshot() error = %v, want nil", err)
					}
					inFlight = len(s.Shipments)
				},
				Inspect: func(t *testing.T, restored, _ []economysim.Event) {
					if inFlight == 0 {
						t.Errorf("seed %d: no shipment in flight at the boundary, want the struck one", seed)
					}
					if !slices.ContainsFunc(restored, func(e economysim.Event) bool { _, ok := e.(economysim.ShipmentArrived); return ok }) {
						t.Errorf("seed %d: the restored world delivered no arrival, want the struck shipment in flight to arrive", seed)
					}
				},
			})
		})
	}
}
