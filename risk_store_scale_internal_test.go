package economysim

import (
	"errors"
	"reflect"
	"testing"
)

// This file tests the store slot of the risk module (plan 3-rsk Phase 6): the call
// SetStoreRiskScales, its errors in the order of spec-risk.md 9.2, the clearing of a
// key at 10 000, the event per changed key, the list in Slots and the read of a scale
// by the probability.

// storeScaleWorld is the room-test world with two store rows: the barn of home (500
// at 4 000, unit) and the shed of home (200 at 10 000, common).
func storeScaleWorld(t *testing.T) (*World, *evLog) {
	t.Helper()
	return riskStoreWorld(t, storeRisk("barn", 500, 4_000, false), storeRisk("shed", 200, 10_000, true))
}

// TestSetStoreRiskScales_Errors verifies the errors of spec-risk.md 9.2 in order, each
// with errors.Is, and that a refused call changes nothing and emits nothing.
//
// Requirement: FR-RSK-028
func TestSetStoreRiskScales_Errors(t *testing.T) {
	t.Parallel()
	w, rec := storeScaleWorld(t)
	off, _ := stoWorld(t, nil)
	row := StoreRiskScale{Region: regHome, Class: classBarn, BP: 5_000}
	with := func(f func(*StoreRiskScale)) []StoreRiskScale { r := row; f(&r); return []StoreRiskScale{r} }
	for _, tt := range []struct {
		name   string
		w      *World
		scales []StoreRiskScale
		want   error
	}{
		{"module off", off, []StoreRiskScale{row}, ErrModuleOff},
		{"region below 0", w, with(func(r *StoreRiskScale) { r.Region = NoRegion }), ErrUnknownRegion},
		{"region above the regions", w, with(func(r *StoreRiskScale) { r.Region = 9 }), ErrUnknownRegion},
		{"class above the classes", w, with(func(r *StoreRiskScale) { r.Class = 9 }), ErrUnknownStoreClass},
		{"class below 0", w, with(func(r *StoreRiskScale) { r.Class = -1 }), ErrUnknownStoreClass},
		{"a key with no row", w, with(func(r *StoreRiskScale) { r.Region = regFar }), ErrUnknownRiskRow},
		{"a scale below 0", w, with(func(r *StoreRiskScale) { r.BP = -1 }), ErrOutOfBounds},
		{"a scale above MaxShockBP", w, with(func(r *StoreRiskScale) { r.BP = MaxShockBP + 1 }), ErrOutOfBounds},
		{"a key twice", w, []StoreRiskScale{row, row}, ErrDuplicateID},
		{"an unknown class before a bad bound", w, with(func(r *StoreRiskScale) { r.Class, r.BP = 9, -1 }), ErrUnknownStoreClass},
		{"an unknown row before a bad bound", w, with(func(r *StoreRiskScale) { r.Region, r.BP = regFar, -1 }), ErrUnknownRiskRow},
	} {
		if err := tt.w.SetStoreRiskScales(tt.scales); !errors.Is(err, tt.want) {
			t.Errorf("%s: SetStoreRiskScales error = %v, want %v", tt.name, err, tt.want)
		}
	}
	// A bad entry after a good one refuses the whole call.
	if err := w.SetStoreRiskScales([]StoreRiskScale{row, {Region: regFar, Class: classBarn, BP: 1}}); !errors.Is(err, ErrUnknownRiskRow) {
		t.Errorf("good then unknown row error = %v, want ErrUnknownRiskRow", err)
	}
	if s, err := w.Slots(); err != nil || len(s.StoreRiskScales) != 0 || len(of[StoreRiskScaled](rec)) != 0 {
		t.Errorf("after the refusals: Slots = %+v, %v, %d events, want nothing changed", s, err, len(of[StoreRiskScaled](rec)))
	}
}

// TestSetStoreRiskScales_AtomicDefaultClearsEventPerChange verifies that a call is
// atomic and that one StoreRiskScaled is emitted for each changed key in (Region,
// Class) order whatever the order of the entries, with the list in key order.
//
// Requirement: FR-RSK-028, FR-RSK-029
func TestSetStoreRiskScales_AtomicDefaultClearsEventPerChange(t *testing.T) {
	t.Parallel()
	w, rec := storeScaleWorld(t)
	barn := StoreRiskScale{Region: regHome, Class: classBarn, BP: 5_000}
	shed := StoreRiskScale{Region: regHome, Class: classShed, BP: 0}
	if err := w.SetStoreRiskScales([]StoreRiskScale{shed, barn}); err != nil {
		t.Fatalf("SetStoreRiskScales error = %v, want nil", err)
	}
	want := []StoreRiskScaled{
		{Region: regHome, Class: classBarn, BeforeBP: 10_000, AfterBP: 5_000},
		{Region: regHome, Class: classShed, BeforeBP: 10_000, AfterBP: 0},
	}
	if got := of[StoreRiskScaled](rec); !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v, want %+v in key order", got, want)
	}
	if s, _ := w.Slots(); !reflect.DeepEqual(s.StoreRiskScales, []StoreRiskScale{barn, shed}) {
		t.Errorf("Slots().StoreRiskScales = %+v, want both in key order", s.StoreRiskScales)
	}
}

// TestSetStoreRiskScales_EqualIsSilentAndDefaultClears verifies that a value equal to
// the one in force is accepted and silent, and that 10 000 clears the key from the
// list with an event from the value it had.
//
// Requirement: FR-RSK-028, FR-RSK-029
func TestSetStoreRiskScales_EqualIsSilentAndDefaultClears(t *testing.T) {
	t.Parallel()
	w, rec := storeScaleWorld(t)
	barn := StoreRiskScale{Region: regHome, Class: classBarn, BP: 5_000}
	shed := StoreRiskScale{Region: regHome, Class: classShed, BP: 0}
	if err := w.SetStoreRiskScales([]StoreRiskScale{shed, barn}); err != nil {
		t.Fatalf("SetStoreRiskScales error = %v, want nil", err)
	}
	if err := w.SetStoreRiskScales([]StoreRiskScale{barn}); err != nil || len(of[StoreRiskScaled](rec)) != 2 {
		t.Errorf("a scale equal to the one in force: error %v, %d events, want none and 2 events in all", err, len(of[StoreRiskScaled](rec)))
	}
	shed.BP = wholeBP
	if err := w.SetStoreRiskScales([]StoreRiskScale{shed}); err != nil {
		t.Fatalf("SetStoreRiskScales(10 000) error = %v, want nil", err)
	}
	if s, _ := w.Slots(); !reflect.DeepEqual(s.StoreRiskScales, []StoreRiskScale{barn}) {
		t.Errorf("Slots().StoreRiskScales after clearing = %+v, want only the other key", s.StoreRiskScales)
	}
	if got := of[StoreRiskScaled](rec); len(got) != 3 || got[2].BeforeBP != 0 || got[2].AfterBP != wholeBP {
		t.Errorf("events = %+v, want a third from 0 to 10 000", got)
	}
}

// TestSetStoreRiskScales_NoTimerHoldsAndScalesTheProbability verifies that a scale
// holds until another call sets it, and that the probability in force is the rounded
// product with the configured one.
//
// Arithmetic: 500 * 2 000 / 10 000 = 100; 500 * 40 000 / 10 000 = 2 000.
//
// Requirement: FR-RSK-006, FR-RSK-029
func TestSetStoreRiskScales_NoTimerHoldsAndScalesTheProbability(t *testing.T) {
	t.Parallel()
	w, _ := storeScaleWorld(t)
	for _, tt := range []struct{ bp, p int64 }{{2_000, 100}, {40_000, 2_000}, {0, 0}} {
		if err := w.SetStoreRiskScales([]StoreRiskScale{{Region: regHome, Class: classBarn, BP: tt.bp}}); err != nil {
			t.Fatalf("SetStoreRiskScales(%d) error = %v, want nil", tt.bp, err)
		}
		step(t, w, 3)
		if s, _ := w.Slots(); len(s.StoreRiskScales) != 1 || w.risk.findStore(regHome, classBarn).p != tt.p {
			t.Errorf("scale %d after 3 ticks: probability %d and slots %+v, want %d and the scale held", tt.bp, w.risk.findStore(regHome, classBarn).p, s.StoreRiskScales, tt.p)
		}
	}
}

// TestEvent_RiskEventsPlainData verifies that the events of the risk module hold no
// map and no pointer, so a consumer and the golden rendering get a fixed order
// (observability-and-logging.md 4.1).
//
// Requirement: FR-RSK-037
func TestEvent_RiskEventsPlainData(t *testing.T) {
	t.Parallel()
	for _, e := range []Event{StoreLoss{}, StoreRiskScaled{}, RouteRiskScaled{}} {
		ty := reflect.TypeOf(e)
		for i := range ty.NumField() {
			switch k := ty.Field(i).Type.Kind(); k {
			case reflect.Map, reflect.Pointer, reflect.Func, reflect.Interface, reflect.Chan:
				t.Errorf("%s.%s has kind %v, want plain data", ty.Name(), ty.Field(i).Name, k)
			}
		}
	}
}

// TestSetStoreScale_AKeyWithNoRowIsAViolatedInvariant verifies that setStoreScale, whose
// callers validate the key, panics with an invariant message for a key that is no
// store row, and changes no scale.
//
// Requirement: FR-RSK-028
// Guard: a nil row would be dereferenced and the sparse list left half-edited.
func TestSetStoreScale_AKeyWithNoRowIsAViolatedInvariant(t *testing.T) {
	t.Parallel()
	w, _ := storeScaleWorld(t)
	if msg := invariantMessage(func() { w.risk.setStoreScale(storeScaleRec{region: regFar, class: classBarn}, 5_000) }); msg == "" {
		t.Error("setStoreScale of a key with no row did not panic with an invariant message")
	}
	if len(w.risk.storeScales) != 0 {
		t.Errorf("storeScales = %v after the refused set, want none", w.risk.storeScales)
	}
}
