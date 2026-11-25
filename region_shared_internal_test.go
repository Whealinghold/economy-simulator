package economysim

import (
	"math"
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

// TestRegionShared_AddToAddsTheWholeWorldParts verifies that addTo adds the journeys, the money in
// contracts and the treasury, and the employed and employing households of each role, to the row
// of region g and to its role rows only, saturating the money instead of wrapping.
//
// Arithmetic: region 1 holds 5 + 2 journeys and 7 + 3 money; its two roles (nr 2) are at slots
// 2 and 3 of the role rows, 4 + 1 and 6 + 1 employed, 8 + 2 and 9 + 2 employing. The treasury
// of 40 added to MaxInt64 saturates at MaxInt64.
//
// Requirement: FR-PLC-025, NFR-PLC-002
func TestRegionShared_AddToAddsTheWholeWorldParts(t *testing.T) {
	t.Parallel()
	const nr = 2
	sh := &regionShared{s: Snapshot{Regions: make([]RegionRow, 3), Roles: make([]RoleRow, 3*nr)}}
	sh.s.Regions[1] = RegionRow{OnJourneyFrom: 2, OnJourneyTo: 3, MoneyContract: 3, MoneyTreasury: 40}
	sh.s.Regions[0] = RegionRow{OnJourneyFrom: 100, MoneyContract: 100}
	sh.s.Roles[2] = RoleRow{Employed: 1, Employing: 2}
	sh.s.Roles[3] = RoleRow{Employed: 1, Employing: 2}
	sh.s.Roles[0] = RoleRow{Employed: 100, Employing: 100}
	s := Snapshot{Regions: make([]RegionRow, 3), Roles: make([]RoleRow, 3*nr)}
	s.Regions[1] = RegionRow{OnJourneyFrom: 5, OnJourneyTo: 2, MoneyContract: 7, MoneyTreasury: Money(math.MaxInt64)}
	s.Roles[2] = RoleRow{Employed: 4, Employing: 8}
	s.Roles[3] = RoleRow{Employed: 6, Employing: 9}
	sh.addTo(&s, 1, nr)
	type sums struct{ from, to int64 }
	if got, want := (sums{s.Regions[1].OnJourneyFrom, s.Regions[1].OnJourneyTo}), (sums{7, 5}); got != want {
		t.Errorf("addTo journeys (from, to) = %v, want %v", got, want)
	}
	if got, want := (sums{int64(s.Regions[1].MoneyContract), int64(s.Regions[1].MoneyTreasury)}), (sums{10, math.MaxInt64}); got != want {
		t.Errorf("addTo money (contract, treasury) = %v, want %v", got, want)
	}
	for i, want := range []sums{{5, 10}, {7, 11}} {
		if got := (sums{s.Roles[2+i].Employed, s.Roles[2+i].Employing}); got != want {
			t.Errorf("addTo role slot %d (employed, employing) = %v, want %v", 2+i, got, want)
		}
	}
	if s.Regions[0] != (RegionRow{}) || s.Roles[0] != (RoleRow{}) {
		t.Errorf("addTo changed another region's rows: %+v, %+v, want zero", s.Regions[0], s.Roles[0])
	}
}

// TestRegionStateOf_OneSharedPassEqualsEachRegionAlone verifies that the states built from one
// regionShared for every region equal the states Region returns, each with a regionShared of its
// own: the saving of the pass is invisible in the result.
//
// Guard: NFR-PLC-002 builds the whole-world parts once per pass. If a state read a part another
// region's build had changed, the actors would decide on a different view than Region shows, and
// only a policy that reads regions would notice.
//
// Requirement: FR-PLC-025, NFR-PLC-002
func TestRegionStateOf_OneSharedPassEqualsEachRegionAlone(t *testing.T) {
	t.Parallel()
	w := newMill(t, policyMillConfig(), WithSeed(5))
	stepUntil(t, w, 3)
	alone := make([]RegionState, len(w.cfg.Regions))
	for g := range alone {
		var err error
		if alone[g], err = w.Region(RegionID(g)); err != nil {
			t.Fatalf("Region(%d) error = %v, want nil", g, err)
		}
	}
	sh, err := w.newRegionShared()
	if err != nil {
		t.Fatalf("newRegionShared() error = %v, want nil", err)
	}
	for g := range alone {
		got, err := w.regionStateOf(RegionID(g), sh)
		if err != nil || !reflect.DeepEqual(got, alone[g]) {
			t.Errorf("regionStateOf(%d, shared) = %+v, error %v, want %+v", g, got, err, alone[g])
		}
	}
}

// TestPolicyView_PassBuildsTheSharedPartOnce verifies that the first actor of a pass that reads
// regions builds the shared part and the next actor reuses it, and that an actor that reads no
// regions builds none.
//
// Requirement: FR-PLC-025, NFR-PLC-002
func TestPolicyView_PassBuildsTheSharedPartOnce(t *testing.T) {
	t.Parallel()
	crown, shire := crownActor("hold", 1), shireActor("hold", 1)
	crown.ReadsRegions, shire.ReadsRegions = true, true
	w := newMill(t, policyMillConfig(crown, shire), WithSeed(5))
	var pass policyPass
	if _, ok := w.policyView(0, &w.policy.actors[0], &pass); !ok || pass.shared == nil {
		t.Fatalf("policyView(0) = ok %v with shared %v, want ok and a built shared part", ok, pass.shared)
	}
	first := pass.shared
	if _, ok := w.policyView(1, &w.policy.actors[1], &pass); !ok || pass.shared != first {
		t.Errorf("policyView(1) rebuilt the shared part (%p, was %p), want the one of the pass", pass.shared, first)
	}
	w.policy.actors[0].reads = false
	var none policyPass
	if _, ok := w.policyView(0, &w.policy.actors[0], &none); !ok || none.shared != nil {
		t.Errorf("policyView(0) with no regions read = ok %v, shared %v, want ok and none built", ok, none.shared)
	}
}
