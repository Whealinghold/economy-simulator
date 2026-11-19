package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// scaleStep is one call of the scripted series of RSK-S5: at tick (the ticks run so far) every row
// is set to bp.
type scaleStep struct{ tick, bp int64 }

// The series of RSK-S5. The shipper of the route pack sends one volley, at tick 1 (the provisioned
// shipper of the transport goldens cannot send a second: the pack collapses before a household holds
// the upkeep of another trip, N-352), so the calls are made around it: a surge before the volley
// (the probabilities doubled), its end at tick 5 (a scale of 10 000 clears the key), and a patrol
// (no strike at all) before the volley or, in the first series, after it, while the cargo is on the
// road. A shipment in flight keeps the share it was given, so the patrol that follows the surge
// takes nothing from the strikes already made (FR-RSK-030). The store pack has dispatches to act
// on all through the run, so its patrol is in the middle of it.
var (
	routeSurge  = []scaleStep{{0, 20_000}, {2, 0}, {5, 10_000}}
	routePatrol = []scaleStep{{0, 0}, {5, 10_000}}
	storeSeries = []scaleStep{{100, 0}, {200, 10_000}}
)

// scaleChanges returns the (before, after) pairs of the scale events of events, route and store
// alike, in the order they were emitted.
func scaleChanges(events []economysim.Event) (out [][2]int64) {
	for _, e := range events {
		switch e := e.(type) {
		case economysim.RouteRiskScaled:
			out = append(out, [2]int64{e.BeforeBP, e.AfterBP})
		case economysim.StoreRiskScaled:
			out = append(out, [2]int64{e.BeforeBP, e.AfterBP})
		}
	}
	return out
}

// applyScale makes the call of step on the world of rw (the routes or the stores) and returns
// the first of the checks that fails: one event per row with the scale before and after, and the
// slot of the call (Slots) holding the scales that differ from 10 000.
func applyScale(t testing.TB, rw *riskWatch, cfg economysim.Config, step scaleStep, before int64, stores bool) string {
	t.Helper()
	mark := len(rw.rec.events)
	var rows int
	if stores {
		rows = len(cfg.Modules.Risk.Stores)
		if err := rw.w.SetStoreRiskScales(riskStoreScales(t, cfg, step.bp)); err != nil {
			t.Fatalf("SetStoreRiskScales(%d) at tick %d error = %v, want nil", step.bp, step.tick, err)
		}
	} else {
		rows = len(cfg.Modules.Risk.Routes)
		if err := rw.w.SetRouteRiskScales(riskRouteScales(t, cfg, step.bp)); err != nil {
			t.Fatalf("SetRouteRiskScales(%d) at tick %d error = %v, want nil", step.bp, step.tick, err)
		}
	}
	got := scaleChanges(rw.rec.events[mark:])
	if len(got) != rows {
		return fmt.Sprintf("tick %d: %d scale events, want one for each of the %d rows", step.tick, len(got), rows)
	}
	for _, c := range got {
		if c != [2]int64{before, step.bp} {
			return fmt.Sprintf("tick %d: a scale event says %d to %d, want %d to %d", step.tick, c[0], c[1], before, step.bp)
		}
	}
	slots, err := rw.w.Slots()
	if err != nil {
		t.Fatalf("Slots() at tick %d error = %v, want nil", step.tick, err)
	}
	held := len(slots.RouteRiskScales)
	if stores {
		held = len(slots.StoreRiskScales)
	}
	if want := map[bool]int{false: rows, true: 0}[step.bp == 10_000]; held != want {
		return fmt.Sprintf("tick %d: Slots() holds %d scales, want %d", step.tick, held, want)
	}
	return ""
}

// runSeries steps rw for ticks, making each call of series when its tick is reached, and returns
// the first failure and the ticks of the dispatches and of the StoreLoss events seen.
func runSeries(t testing.TB, rw *riskWatch, cfg economysim.Config, series []scaleStep, ticks int64, stores bool) (msg string, struck, storeHits []int64) {
	t.Helper()
	before := int64(10_000)
	for i := range ticks {
		for _, s := range series {
			if s.tick == i {
				if msg = applyScale(t, rw, cfg, s, before, stores); msg != "" {
					return msg, nil, nil
				}
				before = s.bp
			}
		}
		mark := len(rw.rec.events)
		if msg = rw.step(t); msg != "" {
			return msg, nil, nil
		}
		for _, e := range rw.rec.events[mark:] {
			switch e := e.(type) {
			case economysim.ShipmentDispatched:
				if e.LossBP > 0 {
					struck = append(struck, e.Tick)
				}
			case economysim.StoreLoss:
				storeHits = append(storeHits, e.Tick)
			}
		}
	}
	return "", struck, storeHits
}

// inWindow counts the ticks of ticks in [from, to).
func inWindow(ticks []int64, from, to int64) (n int64) {
	for _, t := range ticks {
		if t >= from && t < to {
			n++
		}
	}
	return n
}

// TestScenarioRSK5_ScriptedScales is RSK-S5 (S130, spec-risk.md 16, FR-RSK-028 to FR-RSK-030): on
// the route pack a surge before the shipper's volley that a patrol follows while the cargo is on
// the road, and a patrol before the volley; on the store pack a patrol at tick 100 and its end at
// tick 200; seeds 1 to the scenario count, 400 ticks, audit on. Exact: each call emits one event
// per row with the scale before and after, Slots holds the scales that differ from 10 000, the
// books of the loss balance every tick, a unit is lost only by a dispatch that was struck
// and a shipment dispatched at 0 loses none after a later call (a shipment in flight keeps its
// share), no shipment dispatched while the routes are at 0 carries
// a loss share and no store is struck while the stores are at 0. Recorded: the strikes made.
//
// Requirement: FR-RSK-028, FR-RSK-029, FR-RSK-030, FR-RSK-007, FR-RSK-022
//
// Guard: a scale that clears the key must read as the configured probability again, and a scale
// of 0 must strike never; a shipment already on the road keeps the share it was given.
func TestScenarioRSK5_ScriptedScales(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	two, one := riskGoldenTwo(t), riskGoldenStore(t)
	techScenario(t, "two", "S130 two+risk a surge before the volley, a patrol while it is on the road, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		rw := newRiskWatch(t, two.cfg, seed, two.opts)
		msg, struck, _ := runSeries(t, rw, two.cfg, routeSurge, goldenTicks, false)
		checkExact(t, rep, "S130", "two", seed, "the calls, the slots and the books hold", msg == "", "%s", msg)
		lost := rw.tally.routeLost + rw.tally.accountLost
		checkExact(t, rep, "S130", "two", seed, "the surge strikes at least one dispatch", len(struck) > 0, "%d dispatches struck", len(struck))
		// Not "struck iff lost": a strike can take no unit (rounding), so only a loss implies a strike.
		checkExact(t, rep, "S130", "two", seed, "a unit is lost only by a dispatch that was struck", lost == 0 || len(struck) > 0, "%d dispatches struck and %d units lost", len(struck), lost)
		checkExact(t, rep, "S130", "two", seed, "a shipment struck before the patrol still loses units at its arrival", lost > 0, "%d units lost", lost)
		msgLoss := unstruckLoss(rw.rec.events)
		checkExact(t, rep, "S130", "two", seed, "a patrol after the dispatch gives no share and takes none that was given", msgLoss == "", "%s", msgLoss)
		rep.num(seed, "dispatches struck at the surge", int64(len(struck)))
		rep.num(seed, "units lost at the arrivals", lost)
	})
	techScenario(t, "two", "S130 two+risk a patrol before the volley, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		rw := newRiskWatch(t, two.cfg, seed, two.opts)
		msg, struck, _ := runSeries(t, rw, two.cfg, routePatrol, goldenTicks, false)
		checkExact(t, rep, "S130", "two", seed, "the calls, the slots and the books hold", msg == "", "%s", msg)
		checkExact(t, rep, "S130", "two", seed, "no dispatch carries a loss share while the routes are at 0", len(struck) == 0 && rw.tally.lost() == 0, "%d dispatches struck, %d units lost", len(struck), rw.tally.lost())
		rep.num(seed, "shipments dispatched", rw.tally.dispatched)
	})
	techScenario(t, "one", "S130 one+storage+risk-store a patrol and its end on the stores, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		rw := newRiskWatch(t, one.cfg, seed, nil)
		msg, _, hits := runSeries(t, rw, one.cfg, storeSeries, goldenTicks, true)
		checkExact(t, rep, "S130", "one", seed, "the calls, the slots and the books hold", msg == "", "%s", msg)
		checkExact(t, rep, "S130", "one", seed, "no store is struck while the stores are at 0", inWindow(hits, 100, 200) == 0, "%d StoreLoss in the patrol", inWindow(hits, 100, 200))
		rep.num(seed, "store strikes before the patrol", inWindow(hits, 0, 100))
		rep.num(seed, "store strikes after the patrol", inWindow(hits, 200, goldenTicks))
	})
}
