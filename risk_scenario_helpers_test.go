package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The helpers of scenarios S126 to S131 (RSK-S1 to RSK-S6, spec-risk.md 16): the tally of what a
// run of events shows of the module, the scales that put every row at 0, the two packs of the
// module-alone goldens and a step loop that checks the books after every tick. The scenarios keep
// counters and the events, not a snapshot per seed (plan 3-rsk, gate economics, the S113 lesson).

// riskTally is what a run of events shows of the risk module.
type riskTally struct {
	dispatched, struck     int64 // shipments dispatched, and those with a loss share above 0
	arrivals               int64
	landed, wasted         int64 // units delivered, and units spoiled, diverted or lost
	routeLost, accountLost int64 // units lost from the cargo lines and from the trip accounts
	storeEvents, storeLost int64 // StoreLoss events and the units they report
	badLine                string
}

// add tallies the events into r. A cargo line whose units do not add up (Shipped = Delivered +
// Diverted + Spoiled + Lost, spec-risk.md 7.3 item 1) is named in badLine, the first only.
func (r *riskTally) add(events []economysim.Event) {
	for _, e := range events {
		switch e := e.(type) {
		case economysim.ShipmentDispatched:
			r.dispatched++
			if e.LossBP > 0 {
				r.struck++
			}
		case economysim.ShipmentArrived:
			r.arrivals++
			r.addArrival(e)
		case economysim.StoreLoss:
			r.storeEvents++
			for _, a := range e.Lost {
				r.storeLost += int64(a.Quantity)
			}
		}
	}
}

// addArrival tallies the lines and the account losses of one arrival.
func (r *riskTally) addArrival(e economysim.ShipmentArrived) {
	for _, l := range e.Lines {
		r.routeLost += int64(l.Lost)
		r.landed += int64(l.Delivered)
		r.wasted += int64(l.Spoiled + l.Diverted + l.Lost)
		if l.Shipped != l.Delivered+l.Diverted+l.Spoiled+l.Lost && r.badLine == "" {
			r.badLine = fmt.Sprintf("tick %d shipment %d line %+v", e.Tick, e.Shipment, l)
		}
	}
	for _, a := range e.AccountLost {
		r.accountLost += int64(a.Quantity)
	}
}

// lost is the units lost by every channel of the module.
func (r riskTally) lost() int64 { return r.routeLost + r.accountLost + r.storeLost }

// riskWatch steps a world and tallies the events of each step as they come.
type riskWatch struct {
	w     *economysim.World
	rec   *recorder
	seen  int
	tally riskTally
}

// newRiskWatch builds a world of cfg with the recorder and the audit on, and the options given.
func newRiskWatch(t testing.TB, cfg economysim.Config, seed uint64, opts []economysim.Option) *riskWatch {
	t.Helper()
	rec := &recorder{}
	w := newPackWorld(t, cfg, seed, append(slices.Clone(opts), economysim.WithAudit(), economysim.WithObserver(rec))...)
	return &riskWatch{w: w, rec: rec, seen: len(rec.events)}
}

// step runs one step and tallies its events, and returns the first of the checks that fails: the
// boundary balance of the reason loss against the units the events report (spec-risk.md 7.3 items
// 2 and 3), or a cargo line that does not add up.
func (rw *riskWatch) step(t testing.TB) string {
	t.Helper()
	if err := rw.w.Step(); err != nil {
		t.Fatalf("Step() at tick %d error = %v, want nil (the audit ran)", rw.w.Tick(), err)
	}
	rw.tally.add(rw.rec.events[rw.seen:])
	rw.seen = len(rw.rec.events)
	snap, err := rw.w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() at tick %d error = %v, want nil", rw.w.Tick(), err)
	}
	if b := lossBalance(snap); b != rw.tally.lost() {
		return fmt.Sprintf("tick %d: the loss boundary holds %d, the events report %d units lost", snap.Tick, b, rw.tally.lost())
	}
	return rw.tally.badLine
}

// run steps the world for ticks and returns the first failure of step, or "".
func (rw *riskWatch) run(t testing.TB, ticks int64) string {
	t.Helper()
	for range ticks {
		if msg := rw.step(t); msg != "" {
			return msg
		}
	}
	return ""
}

// riskRouteScales returns a scale of bp for every route row of cfg, in the order of the rows.
func riskRouteScales(t testing.TB, cfg economysim.Config, bp int64) []economysim.RouteRiskScale {
	t.Helper()
	var out []economysim.RouteRiskScale
	for _, r := range cfg.Modules.Risk.Routes {
		out = append(out, economysim.RouteRiskScale{From: regionID(t, cfg, r.From), To: regionID(t, cfg, r.To), Mode: modeIDOf(t, cfg, r.Mode), BP: bp})
	}
	return out
}

// riskStoreScales returns a scale of bp for every store row of cfg, in the order of the rows.
func riskStoreScales(t testing.TB, cfg economysim.Config, bp int64) []economysim.StoreRiskScale {
	t.Helper()
	var out []economysim.StoreRiskScale
	for _, r := range cfg.Modules.Risk.Stores {
		out = append(out, economysim.StoreRiskScale{Region: regionID(t, cfg, r.Region), Class: classIDOf(t, cfg, r.Class), BP: bp})
	}
	return out
}

// regionID is the typed ID of the region named id in cfg.
func regionID(t testing.TB, cfg economysim.Config, id string) economysim.RegionID {
	t.Helper()
	i := slices.IndexFunc(cfg.Regions, func(r economysim.Region) bool { return r.ID == id })
	if i < 0 {
		t.Fatalf("region %q is not in the configuration", id)
	}
	return economysim.RegionID(i)
}

// modeIDOf is the typed ID of the transport mode named id in cfg.
func modeIDOf(t testing.TB, cfg economysim.Config, id string) economysim.ModeID {
	t.Helper()
	i := slices.IndexFunc(cfg.Modules.Transport.Modes, func(m economysim.Mode) bool { return m.ID == id })
	if i < 0 {
		t.Fatalf("mode %q is not in the configuration", id)
	}
	return economysim.ModeID(i)
}

// classIDOf is the typed ID of the store class named id in cfg.
func classIDOf(t testing.TB, cfg economysim.Config, id string) economysim.StoreClassID {
	t.Helper()
	i := slices.IndexFunc(cfg.Modules.Storage.Classes, func(c economysim.StoreClass) bool { return c.ID == id })
	if i < 0 {
		t.Fatalf("store class %q is not in the configuration", id)
	}
	return economysim.StoreClassID(i)
}

// riskPackName names a pack of the two module-alone goldens for the scenario reports: the
// scenarios file of pack two or of pack one.
func riskPackName(g riskGolden) string {
	if g.name == "two" {
		return "two"
	}
	return "one"
}

// unstruckLoss returns the first shipment that lost a unit at its arrival although it was
// dispatched with a loss share of 0 (a scale call after the dispatch must not give a share, and a
// share is the only way a unit is lost), and "" when there is none. It joins arrival to dispatch
// by shipment ID, so it holds on every seed whatever a strike rounds to: a strike may take no unit
// (a small cargo at a small share), so "struck" and "lost" are not equivalent, only "lost" implies
// "struck".
func unstruckLoss(events []economysim.Event) string {
	share := map[economysim.ShipmentID]int64{}
	for _, e := range events {
		switch e := e.(type) {
		case economysim.ShipmentDispatched:
			share[e.Shipment] = e.LossBP
		case economysim.ShipmentArrived:
			var n int64
			for _, l := range e.Lines {
				n += int64(l.Lost)
			}
			for _, a := range e.AccountLost {
				n += int64(a.Quantity)
			}
			if bp, ok := share[e.Shipment]; n > 0 && (!ok || bp == 0) {
				return fmt.Sprintf("tick %d shipment %d lost %d units with a share of %d", e.Tick, e.Shipment, n, bp)
			}
		}
	}
	return ""
}
