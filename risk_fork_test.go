package economysim_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// riskShipperPack is the module-alone transport pack with its scripted shipper and
// the risk module over the foot mode of both directions: a row that strikes 6 in 10
// shipments and takes the whole cargo of the strike, so that the one unit of staple
// the shipper sends is lost or arrives, and a run holds both (spec-risk.md 3.2.1).
func riskShipperPack(t testing.TB) (economysim.Config, []economysim.Option) {
	t.Helper()
	cfg, opts := shipperPack(t)
	a, b := cfg.Regions[0].ID, cfg.Regions[1].ID
	cfg.Modules.Risk = &economysim.Risk{Routes: []economysim.RouteRisk{
		{From: a, To: b, Mode: "foot", StrikeBP: 6_000, SeverityBP: 10_000},
		{From: b, To: a, Mode: "foot", StrikeBP: 6_000, SeverityBP: 10_000},
	}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(shipper pack with risk rows) error = %v, want nil", err)
	}
	return cfg, opts
}

// lostOf returns the units the arrivals among events report as lost to a strike, the
// lines of the cargo and the traveller's own account.
func lostOf(events []economysim.Event) (lost int64) {
	for _, e := range events {
		if a, ok := e.(economysim.ShipmentArrived); ok {
			for _, l := range a.Lines {
				lost += int64(l.Lost)
			}
			for _, l := range a.AccountLost {
				lost += int64(l.Quantity)
			}
		}
	}
	return lost
}

// TestFork_Risk verifies the fork-at-N of the shipper pack with the risk module on
// (FR-RSK-033, FR-RSK-035; RSK-S6): before the first dispatch, with a shipment in
// flight that carries a loss share, on the tick of an arrival, after the arrivals
// and much later, the original and two restores give the same events, snapshot,
// audit, checkpoint bytes and log records for 20 ticks, so a restored world draws,
// strikes and loses as the uninterrupted one. The cell inside a journey checks that
// the shipment in flight arrived after the boundary.
//
// Requirement: FR-RSK-033, FR-RSK-035, FR-CHK-050
func TestFork_Risk(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, opts := riskShipperPack(t)
	for _, tc := range []struct {
		name     string
		n        int
		inFlight bool
	}{{"before the first step", 0, false}, {"after a dispatch tick", 2, true}, {"on the tick of arrival", 3, true}, {"after the arrivals", 5, false}, {"much later", 30, false}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var inFlight int
			forkAtN(t, forkCase{
				Name: "risk shipper", Cfg: cfg, Seed: 1, N: tc.n, M: 20, Logs: true,
				Opts: func() []economysim.Option { return opts },
				Before: func(w *economysim.World) {
					s, err := w.Snapshot()
					if err != nil {
						t.Fatalf("Snapshot() error = %v, want nil", err)
					}
					inFlight = len(s.Shipments)
				},
				Inspect: func(t *testing.T, restored, since []economysim.Event) {
					if tc.inFlight && inFlight == 0 {
						t.Errorf("N %d: no shipment in flight at the boundary, want the cell to be inside a journey", tc.n)
					}
					if tc.inFlight && !slices.ContainsFunc(restored, func(e economysim.Event) bool { _, ok := e.(economysim.ShipmentArrived); return ok }) {
						t.Errorf("N %d: the restored world delivered no ShipmentArrived, want the shipment in flight to arrive", tc.n)
					}
				},
			})
		})
	}
}

// TestScenarioRSK_ShipperLosesGoodsAndBalances verifies, through the public API
// alone, the feature as a consumer meets it: the shipper pack with route rows that
// strike over 40 ticks delivers shipments of which some lose their cargo, the
// arrivals report the loss, the boundary account of the reason loss holds exactly the
// units reported, the audit passes at every tick, and a second run of the same seed
// ends in the same events and snapshot while the pack without rows loses nothing.
//
// Requirement: FR-RSK-005, FR-RSK-033
func TestScenarioRSK_ShipperLosesGoodsAndBalances(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, opts := riskShipperPack(t)
	run := func(c economysim.Config) ([]economysim.Event, economysim.Snapshot) {
		rec := &recorder{}
		w, err := economysim.NewWorld(c, append(slices.Clone(opts), economysim.WithObserver(rec), economysim.WithSeed(3), economysim.WithAudit())...)
		if err != nil {
			t.Fatalf("NewWorld() error = %v, want nil", err)
		}
		if _, err := w.Run(context.Background(), 40); err != nil {
			t.Fatalf("Run(40) error = %v, want nil", err)
		}
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		return rec.events, snap
	}
	events, snap := run(cfg)
	lost := lostOf(events)
	if n := arrivalsOf(events); n < 2 || lost == 0 {
		t.Fatalf("%d arrivals and %d units lost in 40 ticks, want arrivals with a strike among them", n, lost)
	}
	if b := lossBalance(snap); b != lost && b != -lost {
		t.Errorf("boundary account of loss holds %d, want the %d units the arrivals report", b, lost)
	}
	again, snapAgain := run(cfg)
	if !reflect.DeepEqual(events, again) || !reflect.DeepEqual(snap, snapAgain) {
		t.Error("a second run of the same seed differs in events or snapshot, want the same run")
	}
	off := cfg
	off.Modules.Risk = nil
	offEvents, _ := run(off)
	if got := lostOf(offEvents); got != 0 {
		t.Errorf("the pack without rows lost %d units, want 0", got)
	}
}

// arrivalsOf counts the ShipmentArrived among events.
func arrivalsOf(events []economysim.Event) (n int) {
	for _, e := range events {
		if _, ok := e.(economysim.ShipmentArrived); ok {
			n++
		}
	}
	return n
}

// lossBalance sums the boundary accounts of the reason loss of snap.
func lossBalance(snap economysim.Snapshot) (sum int64) {
	for _, b := range snap.Boundary {
		if b.Reason == economysim.ReasonLoss {
			sum += b.Balance
		}
	}
	return sum
}
