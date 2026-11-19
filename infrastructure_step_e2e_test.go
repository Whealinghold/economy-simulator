package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The class IDs of infrastructureConfig: the classes are numbered in the order written
// (barn, drain, mill, road).
const (
	e2eBarn economysim.FacilityClassID = iota
	e2eDrain
	e2eMill
)

// TestInfrastructureE2E_StepRuinsAndReports verifies, through the public API of a world
// built from infrastructureConfig with the audit on, what the facility step P8.3b does to
// the records a consumer added: a mill with 3 units of condition and the decay of 1 is
// ruined by decay in the third step (FacilityChanged ruined, cause decay, before 1, after
// 0, once) and then no longer exists (ErrUnknownFacility); a barn added for a grower with
// room is installed at the call, so no FacilityLimited is emitted in any step (never when
// every count is 0); the snapshot row of the barn has no pending facility; the audit
// (conservation of money and goods) holds in every step.
//
// Setup: the mill has condition 3 so the ruin lands inside five steps: 3-1 = 2, 1, then 0
// at step 3. A pending store facility cannot be made through the calls (they refuse an
// owner outside the region or with no room), so the counted Pending of FacilityLimited is
// proved in the internal tests with a planted record (TestEvents_FacilityLimited).
//
// Requirement: FR-CAP-007, FR-CAP-008, FR-CAP-026, FR-CAP-038, FR-CAP-042, FR-CAP-044
func TestInfrastructureE2E_StepRuinsAndReports(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rec := &recorder{}
	w, err := economysim.NewWorld(infrastructureConfig(), economysim.WithSeed(11), economysim.WithAudit(), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	grower := slices.IndexFunc(w.Config().Roles, func(r economysim.Role) bool { return r.ID == "grower" })
	owner, err := w.AddAgent(economysim.NewAgent{Region: 0, Role: economysim.RoleID(grower), Money: 100})
	if err != nil {
		t.Fatalf("AddAgent(grower) error = %v, want nil", err)
	}
	ids, err := w.AddFacilities([]economysim.NewFacility{
		{Class: e2eMill, Region: 0, Toward: economysim.NoRegion, Owner: owner, Condition: 3},
		{Class: e2eBarn, Region: 0, Toward: economysim.NoRegion, Owner: owner},
	})
	if err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	step(t, w, 5)

	var ruins []economysim.FacilityChanged
	for _, e := range rec.events {
		switch e := e.(type) {
		case economysim.FacilityChanged:
			if e.Kind == economysim.FacilityRuined {
				ruins = append(ruins, e)
			}
		case economysim.FacilityLimited:
			t.Errorf("FacilityLimited = %+v, want none: every count is 0 (seed 11)", e)
		}
	}
	if len(ruins) != 1 || ruins[0].Facility != ids[0] || ruins[0].Cause != economysim.CauseDecay || ruins[0].Before != 1 || ruins[0].After != 0 {
		t.Errorf("ruin events = %+v, want exactly one for facility %d with cause decay, before 1, after 0 (seed 11)", ruins, ids[0])
	}
	if _, err := w.Facility(ids[0]); !errors.Is(err, economysim.ErrUnknownFacility) {
		t.Errorf("Facility(ruined) error = %v, want ErrUnknownFacility", err)
	}
	if st, err := w.Facility(ids[1]); err != nil || !st.Installed {
		t.Errorf("Facility(barn) = %+v, %v, want an installed record", st, err)
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	for _, r := range snap.Facilities {
		if r.Pending != 0 {
			t.Errorf("Snapshot row %+v, want no pending facility", r)
		}
	}
}

// TestInfrastructureE2E_GroundLayerRaisesTheCapacity verifies that the ground facility of
// infrastructureConfig (the drain on arable, 3 000 basis points) is read by the resources
// module end to end: the effective capacity of the endowment in the snapshot is the
// capacity of the same world without the drain times 13 000 / 10 000, at the start and
// after 30 ticks, the stock never exceeds it, and the capacity shock slot is untouched.
//
// Requirement: FR-CAP-024, FR-CAP-025
//
// Guard: the layer is written by P8.3b into a table the resources module reads; if the
// table were not rebuilt, or the consumer's scale slot were written instead, the capacity
// of the world would not follow the facility.
func TestInfrastructureE2E_GroundLayerRaisesTheCapacity(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	capacityOf := func(c economysim.Config) (cap, stock, scale int64) {
		t.Helper()
		w, err := economysim.NewWorld(c, economysim.WithSeed(5), economysim.WithAudit())
		if err != nil {
			t.Fatalf("NewWorld error = %v, want nil", err)
		}
		step(t, w, 30)
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot error = %v, want nil", err)
		}
		for _, e := range snap.Endowments {
			if e.Region == 0 && e.Resource == 0 {
				return e.Capacity, e.Stock, e.CapacityScaleBP
			}
		}
		t.Fatal("no endowment row for north and arable")
		return 0, 0, 0
	}
	on := infrastructureConfig()
	off := infrastructureConfig()
	off.Modules.Infrastructure.Initial = slices.DeleteFunc(off.Modules.Infrastructure.Initial,
		func(r economysim.InitialFacility) bool { return r.Class == "drain" })
	capOn, stockOn, scaleOn := capacityOf(on)
	capOff, _, _ := capacityOf(off)
	// The drain's boost is 3 000 basis points at full condition (decay 0): the capacity is
	// raised by 13 000 / 10 000.
	if want := capOff * 13_000 / 10_000; capOn != want {
		t.Errorf("effective capacity with the drain = %d, want %d (%d without it times 13 000 / 10 000, seed 5)", capOn, want, capOff)
	}
	if stockOn > capOn {
		t.Errorf("stock = %d above the effective capacity %d, want it inside", stockOn, capOn)
	}
	if scaleOn != 10_000 {
		t.Errorf("CapacityScaleBP = %d, want 10 000: a facility never writes the consumer's slot", scaleOn)
	}
}

// TestInfrastructureE2E_DeterminismAndNonInterference verifies, for the world of
// infrastructureConfig stepped 60 ticks with the facility step active (the mills decay
// and the barn installs), that the same seed gives the same checkpoint trailer in two runs,
// and that attaching an observer and the audit changes nothing (observability does not
// change results), for several seeds; a different seed gives a different trailer.
//
// Requirement: FR-CAP-029, FR-CAP-039, FR-CAP-041
//
// Guard: the step's counters are kept only under an observer; if one of them fed a
// decision, the observed run would diverge from the quiet run.
func TestInfrastructureE2E_DeterminismAndNonInterference(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	c := infrastructureConfig()
	digests := map[string]uint64{}
	for seed := uint64(1); seed <= uint64(invariantSeeds(3)); seed++ {
		quiet := runPack(t, c, runSpec{Seed: seed, Ticks: 60, Drive: driveStep, Digest: true})
		again := runPack(t, c, runSpec{Seed: seed, Ticks: 60, Drive: driveStep, Digest: true})
		observed := runPack(t, c, runSpec{Seed: seed, Ticks: 60, Drive: driveStep, Observe: true, Audit: true, Digest: true})
		if quiet.Digest != again.Digest {
			t.Errorf("seed %d: two quiet runs give trailers %s and %s, want equal", seed, quiet.Digest, again.Digest)
		}
		if quiet.Digest != observed.Digest {
			t.Errorf("seed %d: quiet trailer %s, observed and audited trailer %s, want equal", seed, quiet.Digest, observed.Digest)
		}
		if prev, dup := digests[quiet.Digest]; dup {
			t.Errorf("seeds %d and %d give the same trailer %s, want different runs", prev, seed, quiet.Digest)
		}
		digests[quiet.Digest] = seed
	}
}
