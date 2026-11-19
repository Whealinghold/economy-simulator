package economysim

import (
	"errors"
	"reflect"
	"testing"
)

// This file tests the route slot of the risk module (plan 3-rsk Phase 5): the call
// SetRouteRiskScales, its errors in the order of spec-risk.md 9.2, the clearing of a
// key at 10 000, the event per changed key, the list in Slots and the read of a scale
// by the probability, the expected loss and not by a shipment in flight (9.3.1).

// riskScaleWorld is the world of 18.1: two route rows on the coach, home to far (3 000
// at 2 500, unit) and far to isle (1 000 at 6 000, common).
func riskScaleWorld(t *testing.T) (*World, *evLog) {
	t.Helper()
	return newShipWorld(t, riskShip(riskCoach("a_home", "b_far", 3_000, 2_500, false), riskCoach("b_far", "c_isle", 1_000, 6_000, true)), nil)
}

// TestSetRouteRiskScales_Errors verifies the errors of spec-risk.md 9.2 in order, each
// with errors.Is, and that a refused call changes nothing and emits nothing.
//
// Requirement: FR-RSK-028
func TestSetRouteRiskScales_Errors(t *testing.T) {
	t.Parallel()
	w, rec := riskScaleWorld(t)
	off, _ := newShipWorld(t, nil, nil)
	row := RouteRiskScale{From: regHome, To: regFar, Mode: modeCoach, BP: 5_000}
	with := func(f func(*RouteRiskScale)) []RouteRiskScale { r := row; f(&r); return []RouteRiskScale{r} }
	for _, tt := range []struct {
		name   string
		w      *World
		scales []RouteRiskScale
		want   error
	}{
		{"module off", off, []RouteRiskScale{row}, ErrModuleOff},
		{"from below 0", w, with(func(r *RouteRiskScale) { r.From = NoRegion }), ErrUnknownRegion},
		{"to above the regions", w, with(func(r *RouteRiskScale) { r.To = 9 }), ErrUnknownRegion},
		{"mode above the modes", w, with(func(r *RouteRiskScale) { r.Mode = 9 }), ErrUnknownMode},
		{"mode below 0", w, with(func(r *RouteRiskScale) { r.Mode = -1 }), ErrUnknownMode},
		{"a key with no row", w, with(func(r *RouteRiskScale) { r.Mode = modeFoot }), ErrUnknownRiskRow},
		{"a scale below 0", w, with(func(r *RouteRiskScale) { r.BP = -1 }), ErrOutOfBounds},
		{"a scale above MaxShockBP", w, with(func(r *RouteRiskScale) { r.BP = MaxShockBP + 1 }), ErrOutOfBounds},
		{"a key twice", w, []RouteRiskScale{row, row}, ErrDuplicateID},
		{"an unknown mode before a bad bound", w, with(func(r *RouteRiskScale) { r.Mode, r.BP = 9, -1 }), ErrUnknownMode},
		{"an unknown row before a bad bound", w, with(func(r *RouteRiskScale) { r.Mode, r.BP = modeFoot, -1 }), ErrUnknownRiskRow},
	} {
		if err := tt.w.SetRouteRiskScales(tt.scales); !errors.Is(err, tt.want) {
			t.Errorf("%s: SetRouteRiskScales error = %v, want %v", tt.name, err, tt.want)
		}
	}
	// A bad entry after a good one refuses the whole call.
	if err := w.SetRouteRiskScales([]RouteRiskScale{row, {From: regHome, To: regFar, Mode: modeFoot, BP: 1}}); !errors.Is(err, ErrUnknownRiskRow) {
		t.Errorf("good then unknown row error = %v, want ErrUnknownRiskRow", err)
	}
	if s, err := w.Slots(); err != nil || len(s.RouteRiskScales) != 0 || len(of[RouteRiskScaled](rec)) != 0 {
		t.Errorf("after the refusals: Slots = %+v, %v, %d events, want nothing changed", s, err, len(of[RouteRiskScaled](rec)))
	}
}

// TestSetRouteRiskScales_AtomicDefaultClearsEventPerChange verifies that a call is
// atomic, that 10 000 clears a key and is not listed, that a value equal to the one
// in force is accepted and silent, and that one RouteRiskScaled is emitted for each
// changed key in (From, To, Mode) order whatever the order of the entries, with the
// scale before and after.
//
// Requirement: FR-RSK-028, FR-RSK-029
func TestSetRouteRiskScales_AtomicDefaultClearsEventPerChange(t *testing.T) {
	t.Parallel()
	w, rec := riskScaleWorld(t)
	homeFar := RouteRiskScale{From: regHome, To: regFar, Mode: modeCoach, BP: 5_000}
	farIsle := RouteRiskScale{From: regFar, To: regIsle, Mode: modeCoach, BP: 0}
	if err := w.SetRouteRiskScales([]RouteRiskScale{farIsle, homeFar}); err != nil {
		t.Fatalf("SetRouteRiskScales error = %v, want nil", err)
	}
	want := []RouteRiskScaled{
		{Tick: 0, From: regHome, To: regFar, Mode: modeCoach, BeforeBP: 10_000, AfterBP: 5_000},
		{Tick: 0, From: regFar, To: regIsle, Mode: modeCoach, BeforeBP: 10_000, AfterBP: 0},
	}
	if got := of[RouteRiskScaled](rec); !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v, want %+v in key order", got, want)
	}
	if s, _ := w.Slots(); !reflect.DeepEqual(s.RouteRiskScales, []RouteRiskScale{homeFar, farIsle}) {
		t.Errorf("Slots().RouteRiskScales = %+v, want both in key order", s.RouteRiskScales)
	}
	if err := w.SetRouteRiskScales([]RouteRiskScale{homeFar}); err != nil || len(of[RouteRiskScaled](rec)) != 2 {
		t.Errorf("a scale equal to the one in force: error %v, %d events, want none and 2 events in all", err, len(of[RouteRiskScaled](rec)))
	}
	farIsle.BP = wholeBP
	requireCleared(t, w, rec, farIsle, homeFar)
}

// requireCleared sets a key to 10 000 and fails the test unless the call lists only
// the other key afterwards and tells a third event from 0 to 10 000.
func requireCleared(t *testing.T, w *World, rec *evLog, cleared, left RouteRiskScale) {
	t.Helper()
	if err := w.SetRouteRiskScales([]RouteRiskScale{cleared}); err != nil {
		t.Fatalf("clearing error = %v, want nil", err)
	}
	if s, _ := w.Slots(); !reflect.DeepEqual(s.RouteRiskScales, []RouteRiskScale{left}) {
		t.Errorf("Slots().RouteRiskScales after clearing = %+v, want only the other key", s.RouteRiskScales)
	}
	if got := of[RouteRiskScaled](rec); len(got) != 3 || got[2].BeforeBP != 0 || got[2].AfterBP != wholeBP {
		t.Errorf("events = %+v, want a third from 0 to 10 000", got)
	}
}

// TestSetRouteRiskScales_NoTimerHolds verifies that a scale holds until another call
// sets it: ticks pass and the value is still listed and still in force.
//
// Requirement: FR-RSK-029
func TestSetRouteRiskScales_NoTimerHolds(t *testing.T) {
	t.Parallel()
	w, _ := riskScaleWorld(t)
	if err := w.SetRouteRiskScales([]RouteRiskScale{{From: regHome, To: regFar, Mode: modeCoach, BP: 2_000}}); err != nil {
		t.Fatalf("SetRouteRiskScales error = %v, want nil", err)
	}
	step(t, w, 15)
	if s, _ := w.Slots(); len(s.RouteRiskScales) != 1 || s.RouteRiskScales[0].BP != 2_000 {
		t.Errorf("Slots().RouteRiskScales after 15 ticks = %+v, want the scale of 2000 held", s.RouteRiskScales)
	}
	if got := w.risk.find(regHome, regFar, modeCoach).p; got != 600 {
		t.Errorf("effective probability = %d, want rhe(3 000 * 2 000, 10 000) = 600", got)
	}
}

// TestScale_ReadByTheProbabilityAndTheTable verifies spec-risk.md 3.4.1, 8.1.3 and
// 9.3.1: a scale of 5 000 halves the probability of 3 000, and the expected loss of the
// route is rebuilt at the call (a scale of 0 takes the row out of it); a scale of 20 000
// above the cap makes the row certain; and a shipment in flight keeps the LossBP it was
// given at its dispatch, whatever a later call sets.
//
// Requirement: FR-RSK-006, FR-RSK-025, FR-RSK-030
func TestScale_ReadByTheProbabilityAndTheTable(t *testing.T) {
	t.Parallel()
	w, _ := riskScaleWorld(t)
	set := func(bp int64) {
		t.Helper()
		if err := w.SetRouteRiskScales([]RouteRiskScale{{From: regHome, To: regFar, Mode: modeCoach, BP: bp}}); err != nil {
			t.Fatalf("SetRouteRiskScales(%d) error = %v, want nil", bp, err)
		}
	}
	row := w.risk.find(regHome, regFar, modeCoach)
	for _, tt := range []struct{ bp, p, loss int64 }{{5_000, 1_500, 375}, {0, 0, 0}, {40_000, 10_000, 2_500}, {10_000, 3_000, 750}} {
		set(tt.bp)
		if got, ok := w.RouteLoss(regHome, regFar, modeCoach); row.p != tt.p || !ok || got != tt.loss {
			t.Errorf("scale %d: probability %d and expected loss %d (%v), want %d and %d", tt.bp, row.p, got, ok, tt.p, tt.loss)
		}
	}
	set(40_000)
	haulFor(t, w, 1)
	id := askCoach(t, w, regFar)
	set(0)
	if got := w.transport.ships[id].loss; got != 2_500 {
		t.Errorf("shipment dispatched at a certain row has loss %d after a scale of 0, want the 2500 it was given", got)
	}
}

// TestScale_LowersAStrikeWithoutShiftingAnotherStream verifies spec-risk.md 11.4.1:
// under keyed streams a row at a scale of 0 takes no word, and the words of the other
// row are the words they were without the call.
//
// Requirement: FR-RSK-007
func TestScale_LowersAStrikeWithoutShiftingAnotherStream(t *testing.T) {
	t.Parallel()
	w, _ := riskScaleWorld(t)
	w2, _ := riskScaleWorld(t)
	if err := w2.SetRouteRiskScales([]RouteRiskScale{{From: regHome, To: regFar, Mode: modeCoach, BP: 0}}); err != nil {
		t.Fatalf("SetRouteRiskScales error = %v, want nil", err)
	}
	for range 6 {
		step(t, w, 1)
		step(t, w, 1) // the common row of far to isle draws the same word in both worlds
		if a, b := w.risk.routes[1].struck, w2.risk.routes[1].struck; a != b {
			t.Fatalf("tick %d: the common row struck %v in one world and %v in the other, want the same stream", w.Tick(), a, b)
		}
	}
}
