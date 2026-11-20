package economysim

import "testing"

// TestFacility_MetricsDerivableRows verifies the rest of 14.4 that TestFacility_MetricsDerivable
// leaves out: the `served` and `cut` sums of FacilityLimited are the executions the mill served
// and the ones it cut, the gauge `pending` equals the Pending of the event and of the snapshot
// row, and the gauge `condition` is the snapshot row's ConditionBP, which is the sum of
// Condition * 10 000 / Durability over the facilities read through the public Facilities call.
//
// Arithmetic: the mills at 1 000 and 500 of Durability 1 000 have capacities 2 and 1 (their
// Effect is 3, TestFacility_Snapshot), so a request of 5 serves 3 and cuts 2. The uses wear 2
// each, 3 * 2 = 6, and the decay takes 1 from each mill, 2: 1 500 - 8 = 1 492 units, 14 920
// basis points. The barn of the weaver (400 of 800, no room, so pending) is 5 000 and does not
// decay (its class has no decay).
//
// Requirement: FR-CAP-038, FR-CAP-037
func TestFacility_MetricsDerivableRows(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, nil)
	m := bldMiller(w, 0)
	bldAdd(t, w, clsMill, m, 2, 1_000, 500)
	facPlant(w, clsBarn, 0, facAgents(w, 0, "weaver")[0], 400)
	w.infra.rebuild()
	bldAsk(t, w, s, "zz_grind", 5)
	facStep(t, w)
	served, cut, pending := foldLimited(rec)
	if served != 3 || cut != 2 {
		t.Errorf("sums of FacilityLimited of the mill = Served %d, Cut %d, want 3 and 2 (seed %d)", served, cut, facSeed)
	}
	var mill, barn FacilityRow
	for _, r := range errVal(w.Snapshot()).Facilities {
		switch r.Class {
		case clsMill:
			mill = r
		case clsBarn:
			barn = r
		}
	}
	if pending != 1 || barn.Pending != pending {
		t.Errorf("pending from FacilityLimited = %d, snapshot Pending = %d, want 1 and 1", pending, barn.Pending)
	}
	if got, want := mill.ConditionBP+barn.ConditionBP, int64(14_920+5_000); got != want {
		t.Errorf("snapshot ConditionBP of the mills and the barn = %d, want %d", got, want)
	}
	if fromState := gaugeFromStates(t, w); fromState != mill.ConditionBP {
		t.Errorf("condition gauge from the facility states = %d, snapshot ConditionBP = %d, want equal", fromState, mill.ConditionBP)
	}
}

// gaugeFromStates is the condition gauge of the mills of region 0 as a consumer derives it from
// the public Facilities call: Condition * 10 000 / Durability summed over the mills.
func gaugeFromStates(t *testing.T, w *World) (sum int64) {
	t.Helper()
	states, err := w.Facilities(0)
	if err != nil {
		t.Fatalf("Facilities(0) error = %v, want nil", err)
	}
	for _, f := range states {
		if f.Class == clsMill {
			sum += f.Condition * 10_000 / w.cfg.Modules.Infrastructure.Classes[f.Class].Durability
		}
	}
	return sum
}

// foldLimited sums Served and Cut of the mill and Pending of the barn of region 0 over the
// FacilityLimited events.
func foldLimited(rec *evLog) (served, cut, pending int64) {
	for _, e := range of[FacilityLimited](rec) {
		switch {
		case e.Region == 0 && e.Class == clsMill:
			served, cut = served+e.Served, cut+e.Cut
		case e.Region == 0 && e.Class == clsBarn:
			pending += e.Pending
		}
	}
	return served, cut, pending
}

// TestFacility_MetricsDerivableVoided verifies that the `voided` sum of FacilityLimited is the
// number of executions of a build lost at the completion for lack of room, and that it is
// booked on the class of the build and in no other.
//
// Arithmetic: five executions of zz_build are started with room for 2 more mills at the
// completion (MaxFacilitiesPerAgent - 2 are held), so 5 - 2 = 3 are voided.
//
// Requirement: FR-CAP-038, FR-CAP-017
func TestFacility_MetricsDerivableVoided(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, nil)
	m := bldMiller(w, 0)
	bldAsk(t, w, s, "zz_build", 5)
	facStep(t, w)
	bldAdd(t, w, clsMill, m, MaxFacilitiesPerAgent-2)
	facStep(t, w)
	voided := map[FacilityClassID]int64{}
	for _, e := range of[FacilityLimited](rec) {
		if e.Region == 0 {
			voided[e.Class] += e.Voided
		}
	}
	if len(voided) != 1 || voided[clsMill] != 3 {
		t.Errorf("Voided by class from FacilityLimited = %v, want only the mill with 3 (seed %d)", voided, facSeed)
	}
}
