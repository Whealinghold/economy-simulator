package economysim

import (
	"reflect"
	"testing"
)

// TestKeepRows verifies the filter the region states build their shared rows with: the rows for
// which keep is true in order, in a new slice, empty (nil only for a nil input) when none is, so a state owns its rows.
//
// Requirement: FR-PLC-031, NFR-PLC-002
func TestKeepRows(t *testing.T) {
	t.Parallel()
	odd := func(n int) bool { return n%2 == 1 }
	for _, tt := range []struct {
		name string
		in   []int
		want []int
	}{
		{"none", nil, nil},
		{"all filtered out", []int{2, 4}, []int{}},
		{"some", []int{1, 2, 3, 4, 5}, []int{1, 3, 5}},
	} {
		got := keepRows(tt.in, odd)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: keepRows(%v) = %v, want %v", tt.name, tt.in, got, tt.want)
		}
	}
	in := []int{1, 3}
	got := keepRows(in, odd)
	got[0] = 9
	if in[0] != 1 {
		t.Errorf("keepRows shares its backing array: the input became %v after a write to the result", in)
	}
}

// TestRegionStateOf_SharedPartsAreNotAliased verifies that two states of one pass built from one
// regionShared own their rows: a write to the journeys of one leaves the other and the shared
// rows as they were.
//
// Requirement: FR-PLC-025, NFR-PLC-002
func TestRegionStateOf_SharedPartsAreNotAliased(t *testing.T) {
	t.Parallel()
	w := newMill(t, policyMillConfig(), WithSeed(5))
	sh, err := w.newRegionShared()
	if err != nil {
		t.Fatalf("newRegionShared() error = %v, want nil", err)
	}
	sh.s.Journeys = []JourneyRow{{From: 0, To: 1, Weight: 3}}
	a, errA := w.regionStateOf(0, sh)
	b, errB := w.regionStateOf(0, sh)
	if errA != nil || errB != nil || len(a.Journeys) != 1 {
		t.Fatalf("regionStateOf() = %d journeys, errors %v and %v, want 1 journey and nil", len(a.Journeys), errA, errB)
	}
	a.Journeys[0].Weight = 99
	if b.Journeys[0].Weight != 3 || sh.s.Journeys[0].Weight != 3 {
		t.Errorf("a write to one state's journeys changed another's (%d) or the shared rows' (%d), want 3 and 3", b.Journeys[0].Weight, sh.s.Journeys[0].Weight)
	}
}

// TestRegionStateOf_ShipmentLinesAreNotAliased verifies that two states of one pass own the
// Lines of a shipment row: a write to one leaves the other and the shared rows as they were.
//
// Requirement: FR-PLC-025, NFR-PLC-002
func TestRegionStateOf_ShipmentLinesAreNotAliased(t *testing.T) {
	t.Parallel()
	w := newMill(t, policyMillConfig(), WithSeed(5))
	sh, err := w.newRegionShared()
	if err != nil {
		t.Fatalf("newRegionShared() error = %v, want nil", err)
	}
	sh.ships = []ShipmentRow{{Region: 0, To: 1, Lines: []Amount{{Quantity: 7}}}}
	a, errA := w.regionStateOf(0, sh)
	b, errB := w.regionStateOf(1, sh)
	if errA != nil || errB != nil || len(a.Shipments) != 1 || len(b.Shipments) != 1 {
		t.Fatalf("regionStateOf() = %d and %d shipments, errors %v and %v, want 1, 1, nil, nil", len(a.Shipments), len(b.Shipments), errA, errB)
	}
	a.Shipments[0].Lines[0].Quantity = 99
	if b.Shipments[0].Lines[0].Quantity != 7 || sh.ships[0].Lines[0].Quantity != 7 {
		t.Errorf("a write to one state's shipment lines changed another's (%v) or the shared rows' (%v), want 7 and 7", b.Shipments[0].Lines[0], sh.ships[0].Lines[0])
	}
}
