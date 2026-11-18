package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// facilityCalls is the call made between ticks in the cells of TestFork_Infrastructure
// that have one: it adds a mill and a road to the estate of the north, halves a mill's
// condition, and gives the other initial mill to the estate. Every kind of change a
// consumer can make to the records (a creation, an adjustment, a transfer) is then in the
// checkpoint, and none of them is in the configuration.
func facilityCalls(t *testing.T) func(w *economysim.World) {
	return func(w *economysim.World) {
		t.Helper()
		const north = economysim.RegionID(0)
		rows, err := w.Facilities(north)
		if err != nil || len(rows) < 4 {
			t.Fatalf("Facilities(north) = %d rows, %v, want the 4 initial facilities and nil", len(rows), err)
		}
		var mills []economysim.FacilityState
		for _, f := range rows {
			if f.Class == 2 { // "mill", the third class in ID order
				mills = append(mills, f)
			}
		}
		if len(mills) != 2 {
			t.Fatalf("Facilities(north) holds %d mills, want 2 initial", len(mills))
		}
		estate := economysim.AgentID(north)
		if _, err := w.AddFacilities([]economysim.NewFacility{
			{Class: 2, Region: north, Toward: economysim.NoRegion, Owner: estate, Condition: 400},
			{Class: 3, Region: north, Toward: 1, Owner: estate},
		}); err != nil {
			t.Fatalf("AddFacilities error = %v, want nil", err)
		}
		if err := w.AdjustFacilities([]economysim.FacilityAdjustment{{ID: mills[0].ID, Delta: -500}}); err != nil {
			t.Fatalf("AdjustFacilities error = %v, want nil", err)
		}
		if err := w.TransferFacilities([]economysim.FacilityTransfer{{ID: mills[1].ID, To: estate}}); err != nil {
			t.Fatalf("TransferFacilities error = %v, want nil", err)
		}
	}
}

// facilityForkCell is one boundary of TestFork_Infrastructure: the ticks run before it
// and the call made on the original there, if any.
type facilityForkCell struct {
	name   string
	n      int
	before func(*testing.T) func(*economysim.World)
}

// facilityForkCells are the boundaries of the fork matrix of the module.
func facilityForkCells() []facilityForkCell {
	return []facilityForkCell{
		{"before the first step", 0, nil},
		{"after some ticks", 25, nil},
		{"after the calls", 25, facilityCalls},
		{"after the calls at tick 0", 0, facilityCalls},
	}
}

// TestFork_Infrastructure verifies that a world forked with facilities of the four kinds
// runs on exactly as an uninterrupted one in every event, in the snapshot (the facility
// rows), in the audit and in the checkpoint bytes, and so does a restore of the
// restore: at the first boundary (the initial rows only), after some ticks, and after
// the calls of facilityCalls made between ticks (tag 27 carries the records, the next
// ID and the installed flag; the lists and tables are rebuilt from them).
//
// Setup: infrastructureConfig (all four kinds, two initial mills), seed 7, M = 30 ticks
// after the boundary. The module has no step yet, so the records change only at a call:
// the cells prove that what a restore rebuilds does not drift from what a run keeps.
//
// Requirement: FR-CAP-041, FR-CAP-029, FR-CHK-050
func TestFork_Infrastructure(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := infrastructureConfig()
	opts := func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }
	cells := facilityForkCells()
	for _, c := range cells {
		fc := forkCase{Name: "infrastructure " + c.name, Cfg: cfg, Seed: 7, N: c.n, M: forkM, Logs: true, Opts: opts}
		if c.before != nil {
			fc.Before = c.before(t)
		}
		forkAtN(t, fc)
	}
}

// observedFacilityWorld builds the world of infrastructureConfig with the audit on and rec
// as its observer, and runs it three ticks.
func observedFacilityWorld(t *testing.T, rec *recorder) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(infrastructureConfig(), economysim.WithSeed(7), economysim.WithAudit(), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	step(t, w, 3)
	return w
}

// externalOwnerOfTwo adds an external grower to the region and gives it a mill and a road
// (the classes 2 and 3 of infrastructureConfig, the road toward region 1).
func externalOwnerOfTwo(t *testing.T, w *economysim.World, region economysim.RegionID) (economysim.AgentID, []economysim.FacilityID) {
	t.Helper()
	grower := slices.IndexFunc(w.Config().Roles, func(r economysim.Role) bool { return r.ID == "grower" })
	owner, err := w.AddAgent(economysim.NewAgent{Region: region, Role: economysim.RoleID(grower), Money: 100})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	owned, err := w.AddFacilities([]economysim.NewFacility{
		{Class: 2, Region: region, Toward: economysim.NoRegion, Owner: owner},
		{Class: 3, Region: region, Toward: 1, Owner: owner},
	})
	if err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	return owner, owned
}

// facilityChange is the kind and cause of one FacilityChanged.
type facilityChange struct {
	kind  economysim.FacilityKind
	cause economysim.Cause
}

// facilityChanges lists the kind and cause of every FacilityChanged of the events, in order.
func facilityChanges(events []economysim.Event) []facilityChange {
	var out []facilityChange
	for _, e := range events {
		if fc, ok := e.(economysim.FacilityChanged); ok {
			out = append(out, facilityChange{fc.Kind, fc.Cause})
		}
	}
	return out
}

// removalOrder returns the order of the facility and bankruptcy events of a removal
// ("facility" or "bankrupted") and the IDs of the facilities moved, and reports each
// facility event that is not a transfer to the estate with the cause owner_removed.
func removalOrder(t *testing.T, events []economysim.Event, estate economysim.AgentID) ([]string, []economysim.FacilityID) {
	t.Helper()
	var order []string
	var moved []economysim.FacilityID
	for _, e := range events {
		switch e := e.(type) {
		case economysim.FacilityChanged:
			order = append(order, "facility")
			moved = append(moved, e.Facility)
			if e.Kind != economysim.FacilityTransferred || e.Cause != economysim.CauseOwnerRemoved || e.Owner != estate {
				t.Errorf("removal FacilityChanged = (%s, %s, owner %d), want (transferred, owner_removed, estate %d)", e.Kind, e.Cause, e.Owner, estate)
			}
		case economysim.AgentBankrupted:
			order = append(order, "bankrupted")
		}
	}
	return order, moved
}

// TestFacilityCalls_ThroughThePublicAPI verifies, with an observer, the calls of a
// consumer on the records: AddFacilities and AdjustFacilities to a ruin each emit their
// FacilityChanged after the call is applied, in call order, with the kind and cause of
// 14.2; the queries agree (a ruined facility is unknown); and a refused call changes
// nothing and emits nothing.
//
// Requirement: FR-CAP-034, FR-CAP-035, FR-CAP-037
func TestFacilityCalls_ThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	rec := &recorder{}
	w := observedFacilityWorld(t, rec)
	const north = economysim.RegionID(0)
	estate := economysim.AgentID(north)
	rec.forget()

	// One mill to the estate, then a refused call (an unknown class) that must leave no trace.
	ids, err := w.AddFacilities([]economysim.NewFacility{{Class: 2, Region: north, Toward: economysim.NoRegion, Owner: estate, Condition: 100}})
	if err != nil || len(ids) != 1 {
		t.Fatalf("AddFacilities = %v, %v, want one ID and nil", ids, err)
	}
	before := len(rec.events)
	if _, err := w.AddFacilities([]economysim.NewFacility{{Class: 99, Region: north, Toward: economysim.NoRegion, Owner: estate}}); err == nil {
		t.Error("AddFacilities of an unknown class error = nil, want an error")
	}
	if err := w.AdjustFacilities([]economysim.FacilityAdjustment{{ID: 9999, Delta: 1}}); !errors.Is(err, economysim.ErrUnknownFacility) {
		t.Errorf("AdjustFacilities of an unknown ID = %v, want ErrUnknownFacility", err)
	}
	if len(rec.events) != before {
		t.Errorf("refused calls emitted %d events, want none", len(rec.events)-before)
	}

	// The 100 units left are ruined by a delta of -100: the record ends.
	if err := w.AdjustFacilities([]economysim.FacilityAdjustment{{ID: ids[0], Delta: -100}}); err != nil {
		t.Fatalf("AdjustFacilities error = %v, want nil", err)
	}
	if _, err := w.Facility(ids[0]); !errors.Is(err, economysim.ErrUnknownFacility) {
		t.Errorf("Facility(%d) after a ruin = %v, want ErrUnknownFacility", ids[0], err)
	}

	want := []facilityChange{{economysim.FacilityAdded, economysim.CauseConsumer}, {economysim.FacilityRuined, economysim.CauseConsumer}}
	if got := facilityChanges(rec.events); !slices.Equal(got, want) {
		t.Errorf("FacilityChanged (kind, cause) = %v, want %v", got, want)
	}
}

// TestFacilityRemoval_ThroughThePublicAPI verifies that the removal of an external
// owner, called through the public API, passes its site and route facilities to the
// estate of their region (step 2g of a removal) with FacilityChanged (transferred,
// owner_removed) in ascending ID, all before AgentBankrupted, that the queries then show
// the estate as the owner, and that the audit passes on the steps after.
//
// Requirement: FR-CAP-030, FR-CAP-037
func TestFacilityRemoval_ThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	rec := &recorder{}
	w := observedFacilityWorld(t, rec)
	const north = economysim.RegionID(0)
	estate := economysim.AgentID(north)
	owner, owned := externalOwnerOfTwo(t, w, north)
	rec.forget()
	if err := w.RemoveAgent(owner); err != nil {
		t.Fatalf("RemoveAgent(%d) error = %v, want nil", owner, err)
	}
	order, moved := removalOrder(t, rec.events, estate)
	if !slices.Equal(order, []string{"facility", "facility", "bankrupted"}) || !slices.Equal(moved, owned) {
		t.Errorf("removal events = %v for facilities %v, want two facility events for %v before AgentBankrupted", order, moved, owned)
	}
	for _, id := range owned {
		if f, err := w.Facility(id); err != nil || f.Owner != estate {
			t.Errorf("Facility(%d) after the removal = %+v, %v, want held by the estate %d", id, f, err, estate)
		}
	}
	step(t, w, 2)
	if err := w.Audit(); err != nil {
		t.Errorf("Audit error = %v, want nil", err)
	}
}
