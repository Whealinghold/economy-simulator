package economysim_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// zeroScales puts every row of cfg at a scale of 0, the neutral setting of RSK-S1: the call is
// made on the world before its first step, between ticks (spec-risk.md 9.2).
func zeroScales(t testing.TB, w *economysim.World, cfg economysim.Config) {
	t.Helper()
	if len(cfg.Modules.Risk.Routes) > 0 {
		if err := w.SetRouteRiskScales(riskRouteScales(t, cfg, 0)); err != nil {
			t.Fatalf("SetRouteRiskScales(every row at 0) error = %v, want nil", err)
		}
	}
	if len(cfg.Modules.Risk.Stores) > 0 {
		if err := w.SetStoreRiskScales(riskStoreScales(t, cfg, 0)); err != nil {
			t.Fatalf("SetStoreRiskScales(every row at 0) error = %v, want nil", err)
		}
	}
}

// neutralRun runs cfg for ticks steps and returns the events other than the module's own
// (WorldCreated, which carries the rows, and the two scale events of the calls of zeroScales) and
// the snapshot with what the module on adds to it taken out. withScalesAtZero says whether the
// rows are put at 0 before the first step.
func neutralRun(t *testing.T, cfg economysim.Config, opts []economysim.Option, seed uint64, ticks int64, withScalesAtZero bool) ([]economysim.Event, economysim.Snapshot) {
	t.Helper()
	rw := newRiskWatch(t, cfg, seed, opts)
	if withScalesAtZero {
		zeroScales(t, rw.w, cfg)
	}
	for range ticks {
		if err := rw.w.Step(); err != nil {
			t.Fatalf("seed %d: Step() error = %v, want nil", seed, err)
		}
	}
	var events []economysim.Event
	for _, e := range rw.rec.events {
		switch e.(type) {
		case economysim.WorldCreated, economysim.RouteRiskScaled, economysim.StoreRiskScaled:
		default:
			events = append(events, e)
		}
	}
	snap, err := rw.w.Snapshot()
	if err != nil {
		t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
	}
	snap.ConfigHash = [32]byte{}
	if withScalesAtZero {
		snap = withoutLoss(t, snap)
	}
	return events, snap
}

// TestScenarioRSK1_NeutralAtZero is RSK-S1 (S126, spec-risk.md 16, FR-RSK-005): each pack of the
// module-alone goldens (route rows on pack two with the shipper, store rows on pack one) is run
// with the module on and every scale at 0 and with the module off, seeds 1 to the scenario count,
// 400 ticks, audit on. Exact: the events other than the module's own and the snapshot, with the
// loss boundary rows and the module's name taken out, are equal bit for bit, because a row with a
// probability of 0 takes no word and loses nothing (FR-RSK-007).
//
// Requirement: FR-RSK-005, FR-RSK-007
//
// Guard: a draw taken at a probability of 0, or a rounding that took a unit at a share of 0, would
// move the run of a world that asked for no risk.
func TestScenarioRSK1_NeutralAtZero(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, g := range []riskGolden{riskGoldenTwo(t), riskGoldenStore(t)} {
		techScenario(t, riskPackName(g), "S126 "+g.name+" the module on with every scale at 0 against the module off, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
			off := g.cfg
			off.Modules.Risk = nil
			onEvents, onSnap := neutralRun(t, g.cfg, g.opts, seed, goldenTicks, true)
			offEvents, offSnap := neutralRun(t, off, g.opts, seed, goldenTicks, false)
			checkExact(t, rep, "S126", g.name, seed, "events equal the module-off run", reflect.DeepEqual(onEvents, offEvents), "%d events on, %d off", len(onEvents), len(offEvents))
			checkExact(t, rep, "S126", g.name, seed, "snapshot equals the module-off run", reflect.DeepEqual(onSnap, offSnap), "snapshots differ")
			rep.num(seed, "events", int64(len(offEvents)))
		})
	}
}
