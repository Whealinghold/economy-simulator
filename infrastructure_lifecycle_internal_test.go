package economysim

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

// facExternal adds an external agent of the role miller in the region, failing the test on
// an error: a facility owner that RemoveAgent can remove.
func facExternal(t testing.TB, w *World, region RegionID) AgentID {
	t.Helper()
	role, ok := w.cfg.roleID("miller")
	if !ok {
		t.Fatalf("no role miller")
	}
	id, err := w.AddAgent(NewAgent{Region: region, Role: role, Money: 100})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	return id
}

// TestRemoval_Estate verifies step 2g of a removal: a site, a route and a ground facility
// of the removed agent pass to the estate of their region with FacilityChanged
// (transferred, owner_removed), a store facility is ruined with the same cause and no
// StoreChanged, in ascending ID, all before AgentBankrupted, and the estate then holds the
// facilities in the table and the lists, and the audit passes.
//
// Requirement: FR-CAP-030, FR-CAP-027
func TestRemoval_Estate(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, nil, WithObserver(obs))
	owner := facExternal(t, w, 0)
	ids, err := w.AddFacilities([]NewFacility{
		{Class: clsMill, Region: 0, Toward: NoRegion, Owner: owner},
		{Class: clsRoad, Region: 0, Toward: 1, Owner: owner},
		{Class: clsBarn, Region: 0, Toward: NoRegion, Owner: owner},
		{Class: clsDrain, Region: 0, Toward: NoRegion, Owner: owner},
	})
	if err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	obs.events = nil
	if err := w.RemoveAgent(owner); err != nil {
		t.Fatalf("RemoveAgent error = %v, want nil", err)
	}
	var order []string
	for _, e := range obs.events {
		switch v := e.(type) {
		case FacilityChanged:
			order = append(order, string(v.Kind)+":"+string(v.Cause))
		case StoreChanged:
			order = append(order, "store")
		case AgentBankrupted:
			order = append(order, "bankrupted")
		}
	}
	want := []string{"transferred:owner_removed", "transferred:owner_removed", "ruined:owner_removed", "transferred:owner_removed", "bankrupted"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("events = %v, want %v: ascending ID, no StoreChanged, before AgentBankrupted", order, want)
	}
	for _, id := range []FacilityID{ids[0], ids[1], ids[3]} {
		if st, err := w.Facility(id); err != nil || st.Owner != 0 {
			t.Errorf("Facility(%d) = %+v, %v, want it held by the estate of home", id, st, err)
		}
	}
	if _, err := w.Facility(ids[2]); !errors.Is(err, ErrUnknownFacility) {
		t.Errorf("Facility(barn) = %v, want ErrUnknownFacility: a store facility is ruined at the removal", err)
	}
	if got := w.infra.site[siteKey{0, 0, clsMill}]; got != 2 {
		t.Errorf("site capacity of the estate = %d, want 2: the mill is idle but the table follows the owner", got)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestNoRegionBound verifies N-090 (spec-infrastructure.md 7.5): the owner of a site, a
// route and a ground facility is relocated as if it held nothing, the owner of an
// installed store is refused by storage's has_store as before, and a change of weight or
// of role leaves the facilities where they are.
//
// Requirement: FR-CAP-031
func TestNoRegionBound(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	site, store := facExternal(t, w, 0), facExternal(t, w, 0)
	if _, err := w.AddFacilities([]NewFacility{
		{Class: clsMill, Region: 0, Toward: NoRegion, Owner: site},
		{Class: clsBarn, Region: 0, Toward: NoRegion, Owner: store},
	}); err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	if err := w.Relocate(site, 1); err != nil {
		t.Errorf("Relocate of a site owner = %v, want nil: a site binds nobody", err)
	}
	var rb *RegionBoundError
	if err := w.Relocate(store, 1); !errors.Is(err, ErrRegionBound) || !errors.As(err, &rb) || rb.Obligation != "has_store" {
		t.Errorf("Relocate of a store owner = %v, want ErrRegionBound has_store", err)
	}
	if st := errVal(w.Agent(site)); len(st.Facilities) != 1 || st.Facilities[0].Region != 0 || st.Region != 1 {
		t.Errorf("Agent(site) = region %d with %+v, want it in yonder holding its mill of home", st.Region, st.Facilities)
	}
}

// TestStoreSeam_StoreInUse verifies that RemoveStore is refused with ErrStoreInUse while
// the stores of the class would fall below the installed store facilities of the class,
// and that a store added by AddStore beside the facility's is removed as before.
//
// Requirement: FR-CAP-047, FR-CAP-026
func TestStoreSeam_StoreInUse(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	m := facExternal(t, w, 0)
	if _, err := w.AddFacilities([]NewFacility{{Class: clsBarn, Region: 0, Toward: NoRegion, Owner: m}}); err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	if err := w.RemoveStore(m, 0); !errors.Is(err, ErrStoreInUse) {
		t.Errorf("RemoveStore of the installed facility's store = %v, want ErrStoreInUse", err)
	}
	if err := w.AddStore(m, 0); err != nil {
		t.Fatalf("AddStore error = %v, want nil", err)
	}
	if err := w.RemoveStore(m, 0); err != nil {
		t.Errorf("RemoveStore of the store the consumer added = %v, want nil", err)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestAdjustFacilities_RuinsAStoreFacility verifies the order of the ruin of a store
// facility by a call: storage's StoreChanged (removed, facility) first and then
// FacilityChanged (ruined, consumer), the store gone from the owner, and the way out of an
// installed store that RemoveStore refuses.
//
// Requirement: FR-CAP-035, FR-CAP-026
func TestAdjustFacilities_RuinsAStoreFacility(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, nil, WithObserver(obs))
	m := facExternal(t, w, 0)
	ids := errVal(w.AddFacilities([]NewFacility{{Class: clsBarn, Region: 0, Toward: NoRegion, Owner: m}}))
	owned := w.sto.owned(m)
	obs.events = nil
	if err := w.AdjustFacilities([]FacilityAdjustment{{ID: ids[0], Delta: -800}}); err != nil {
		t.Fatalf("AdjustFacilities error = %v, want nil", err)
	}
	var seq []string
	for _, e := range obs.events {
		switch v := e.(type) {
		case StoreChanged:
			seq = append(seq, "store:"+string(v.Kind)+":"+string(v.Cause))
		case FacilityChanged:
			seq = append(seq, "facility:"+string(v.Kind)+":"+string(v.Cause))
		}
	}
	if want := []string{"store:removed:facility", "facility:ruined:consumer"}; !reflect.DeepEqual(seq, want) {
		t.Errorf("events = %v, want %v", seq, want)
	}
	if w.sto.owned(m) != owned-1 || len(w.infra.installed) != 0 {
		t.Errorf("stores = %d and installed = %v, want %d stores and none installed", w.sto.owned(m), w.infra.installed, owned-1)
	}
}

// TestFacility_CountIdentity verifies, over a run of calls and a removal, that for each
// class and region the facilities that exist are the initial ones plus those added less
// those ruined, counted from the events (spec-infrastructure.md 14.5, FR-CAP-043).
//
// Requirement: FR-CAP-043
func TestFacility_CountIdentity(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, facRows, WithObserver(obs))
	m := facExternal(t, w, 0)
	ids := errVal(w.AddFacilities([]NewFacility{
		{Class: clsMill, Region: 0, Toward: NoRegion, Owner: m}, {Class: clsMill, Region: 0, Toward: NoRegion, Owner: m},
		{Class: clsRoad, Region: 1, Toward: NoRegion, Owner: 1},
	}))
	if err := w.AdjustFacilities([]FacilityAdjustment{{ID: ids[0], Delta: -2_000}}); err != nil {
		t.Fatalf("AdjustFacilities error = %v, want nil", err)
	}
	if err := w.RemoveAgent(m); err != nil {
		t.Fatalf("RemoveAgent error = %v, want nil", err)
	}
	type cell struct {
		region RegionID
		class  FacilityClassID
	}
	want := map[cell]int{}
	for _, e := range of[FacilityChanged](obs) {
		switch e.Kind {
		case FacilityAdded, FacilityBuilt:
			want[cell{e.Region, e.Class}]++
		case FacilityRuined:
			want[cell{e.Region, e.Class}]--
		}
	}
	got := map[cell]int{}
	for _, f := range w.infra.fac {
		got[cell{f.region, f.class}]++
	}
	for c, n := range want {
		if got[c] != n {
			t.Errorf("class %d in region %d: %d facilities, the events count %d", c.class, c.region, got[c], n)
		}
	}
	if len(got) != len(wantNonZero(want)) {
		t.Errorf("cells with facilities = %v, want %v", got, wantNonZero(want))
	}
}

// wantNonZero is the cells of the counts with a count above 0.
func wantNonZero[K comparable](m map[K]int) map[K]int {
	out := map[K]int{}
	for k, n := range m {
		if n > 0 {
			out[k] = n
		}
	}
	return out
}

// plantedFacilityWorld is a world at tick 3 with facilities of the four kinds, one of them
// at a condition other than the full one and an installed store facility.
func plantedFacilityWorld(t *testing.T) *World {
	t.Helper()
	w := facWorld(t, facRows)
	step(t, w, 3)
	ids := errVal(w.AddFacilities([]NewFacility{{Class: clsMill, Region: 0, Toward: NoRegion, Owner: facMiller(w), Condition: 700}}))
	if err := w.AdjustFacilities([]FacilityAdjustment{{ID: ids[0], Delta: -100}}); err != nil {
		t.Fatalf("AdjustFacilities error = %v, want nil", err)
	}
	return w
}

// sameInfrastructure reports whether two states hold the same records, lists and tables.
func sameInfrastructure(a, b *infrastructure) bool {
	return a.next == b.next && reflect.DeepEqual(a.fac, b.fac) && reflect.DeepEqual(a.byOwner, b.byOwner) &&
		reflect.DeepEqual(a.installed, b.installed) && reflect.DeepEqual(a.site, b.site) &&
		reflect.DeepEqual(a.route, b.route) && reflect.DeepEqual(a.ground, b.ground)
}

// TestCheckpoint_Facilities verifies tag 27: a world with facilities of the four kinds
// restores to the bytes of the original with the same records, lists and tables and
// both give the same bytes after more ticks, and a restore with another infrastructure
// configuration is ErrCheckpointConfig. (That a world with the module off has no section
// is TestInfrastructure_NoDraws.)
//
// Requirement: FR-CAP-041, FR-CAP-029
func TestCheckpoint_Facilities(t *testing.T) {
	t.Parallel()
	w := plantedFacilityWorld(t)
	cp := errVal(w.Checkpoint())
	restored, err := RestoreWorld(w.Config(), cp, WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	if again := errVal(restored.Checkpoint()); !bytes.Equal(again.b, cp.b) {
		t.Errorf("checkpoint of the restored world has %d bytes and differs from the original's %d", len(again.b), len(cp.b))
	}
	if !sameInfrastructure(w.infra, restored.infra) {
		t.Errorf("the restored state differs from the original's: records, lists or tables")
	}
	step(t, w, 3)
	step(t, restored, 3)
	x, y := errVal(w.Checkpoint()), errVal(restored.Checkpoint())
	if !bytes.Equal(x.b, y.b) {
		t.Error("checkpoints of the original and the restored world differ after three more ticks")
	}
	other := w.Config()
	other.Modules.Infrastructure.Classes[2].Capacity++
	if _, err := RestoreWorld(other, cp); !errors.Is(err, ErrCheckpointConfig) {
		t.Errorf("RestoreWorld with another class = %v, want ErrCheckpointConfig", err)
	}
}

// TestCheckpoint_FacilitiesRefusals verifies every refusal of 15.3: a checkpoint taken from
// a state that breaks one rule is ErrCheckpointCorrupt on restore, each case breaking only
// that rule.
//
// Requirement: FR-CAP-041
func TestCheckpoint_FacilitiesRefusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mut  func(w *World)
	}{
		{"next ID not above the last", func(w *World) { w.infra.next = w.infra.fac[len(w.infra.fac)-1].id }},
		{"condition above durability", func(w *World) { w.infra.fac[0].condition = 99_999 }},
		{"condition 0", func(w *World) { w.infra.fac[0].condition = 0 }},
		{"owner not live", func(w *World) { w.infra.fac[0].owner = 999 }},
		{"estate of another region", func(w *World) { w.infra.fac[w.infra.index(roadID(w))].owner = 1 }},
		{"target on a site", func(w *World) { w.infra.fac[w.infra.index(millID(w))].to = 1 }},
		{"installed on a site", func(w *World) { w.infra.fac[w.infra.index(millID(w))].installed = true }},
		{"installed with no store", func(w *World) {
			f := &w.infra.fac[w.infra.index(barnID(w))]
			w.sto.take(f.owner, 0)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := plantedFacilityWorld(t)
			tt.mut(w)
			cp, err := w.Checkpoint()
			if err != nil {
				t.Fatalf("Checkpoint error = %v, want nil", err)
			}
			if _, err := RestoreWorld(w.Config(), cp); !errors.Is(err, ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld = %v, want ErrCheckpointCorrupt", err)
			}
		})
	}
}

// facilityOf is the ID of the first facility of the class, 0 when there is none.
func facilityOf(w *World, class FacilityClassID) FacilityID {
	for _, f := range w.infra.fac {
		if f.class == class {
			return f.id
		}
	}
	return 0
}

func roadID(w *World) FacilityID { return facilityOf(w, clsRoad) }
func millID(w *World) FacilityID { return facilityOf(w, clsMill) }
func barnID(w *World) FacilityID { return facilityOf(w, clsBarn) }
