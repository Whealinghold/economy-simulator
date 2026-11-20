package economysim

import (
	"errors"
	"slices"
	"testing"
)

// The tests of the two slots of the institutions module (plan 3-org Phase 8,
// spec-institutions.md 7.1, 7.2): what a call does to the resolved tables and the events,
// on the world of orgTablesToy, whose controls are, by ControlID, assize 0 (grain in the
// crown: cap 12 000, floor 8 000), assize_a 1 (grain in a: cap 10 000), usury 2 (rate cap
// 1 200 in the crown) and usury_a 3 (rate cap 900 in a), and whose rights are, by RightID,
// ban_a 0, farm_b 1, sell_crown 2, sell_inland 3 and train_c 4. Grain has the reference
// price 100.

// orgSlotWorld builds the world of orgTablesToy with an event log.
func orgSlotWorld(t *testing.T) (*World, *evLog) {
	t.Helper()
	return orgWorld(t, orgTablesToy())
}

// orgWantRange fails the test unless region g has the cap and floor of grain and the rate cap.
func orgWantRange(t *testing.T, o *institutions, g RegionID, capPrice, floor, rate int64) {
	t.Helper()
	gotCap, gotFloor, _, _ := o.rangeOf(g, 0)
	if gotCap != capPrice || gotFloor != floor || o.legal.rate[g] != rate {
		t.Errorf("region %d: range (%d, %d) and rate cap %d, want (%d, %d) and %d", g, gotCap, gotFloor, o.legal.rate[g], capPrice, floor, rate)
	}
}

// orgWantRight fails the test unless the right in force for the activity on the subject in
// region g is want, or the activity is open there when open is true.
func orgWantRight(t *testing.T, o *institutions, activity, subject int, g RegionID, want RightID, open bool) {
	t.Helper()
	id, ok := o.rightIn(activity, subject, g)
	if ok == open || (ok && id != want) {
		t.Errorf("rightIn(activity %d, subject %d, region %d) = %d, %v, want %d with the key open = %v", activity, subject, g, id, ok, want, open)
	}
}

// TestSetLegalLevels_AtomicAndRebuilds verifies that a call rebuilds the ranges of the
// controls it changes and leaves the others, that the new levels are listed by Slots and
// the snapshot, that the event of each changed control comes after the whole call in
// ascending ControlID, and that an entry equal to the level in force changes nothing.
//
// Arithmetic: assize to (15 000, 5 000) gives b and c the cap 100 * 15 000 / 10 000 = 150
// and the floor 100 * 5 000 / 10 000 = 50; usury_a to 600 gives a the rate cap 600; assize_a
// is untouched, so a keeps its cap 100.
//
// Requirement: FR-ORG-036, FR-ORG-038, FR-ORG-042
func TestSetLegalLevels_AtomicAndRebuilds(t *testing.T) {
	t.Parallel()
	w, rec := orgSlotWorld(t)
	o := w.org
	err := w.SetLegalLevels([]LegalLevel{{Control: 3, CapBP: 600}, {Control: 0, CapBP: 15_000, FloorBP: 5_000}, {Control: 1, CapBP: 10_000}})
	if err != nil {
		t.Fatalf("SetLegalLevels() error = %v, want nil", err)
	}
	orgWantRange(t, o, orgA, 100, 0, 600)
	orgWantRange(t, o, orgB, 150, 50, 1_200)
	orgWantRange(t, o, orgC, 150, 50, 1_200)
	want := []LegalLevelSet{
		{Tick: 0, Control: 0, CapBefore: 12_000, CapAfter: 15_000, FloorBefore: 8_000, FloorAfter: 5_000},
		{Tick: 0, Control: 3, CapBefore: 900, CapAfter: 600},
	}
	if got := of[LegalLevelSet](rec); !slices.Equal(got, want) {
		t.Errorf("LegalLevelSet events = %+v, want %+v (assize_a is equal to its level: no event)", got, want)
	}
	s, err := w.Slots()
	if err != nil {
		t.Fatalf("Slots() error = %v, want nil", err)
	}
	if want := []LegalLevel{{Control: 0, CapBP: 15_000, FloorBP: 5_000}, {Control: 3, CapBP: 600}}; !slices.Equal(s.LegalLevels, want) {
		t.Errorf("Slots().LegalLevels = %+v, want %+v", s.LegalLevels, want)
	}
	if err := w.SetLegalLevels([]LegalLevel{{Control: 0, CapBP: 15_000, FloorBP: 5_000}}); err != nil {
		t.Fatalf("SetLegalLevels(the level in force) error = %v, want nil", err)
	}
	if got := of[LegalLevelSet](rec); len(got) != 2 {
		t.Errorf("LegalLevelSet events after a call that changes nothing = %d, want still 2", len(got))
	}
}

// TestSetLegalLevels_EntryEqualToConfiguredRestores verifies that an entry equal to the
// configured level puts the configured range back and leaves no row in the slot state or
// the snapshot (spec-institutions.md 7.1.2).
//
// Requirement: FR-ORG-036, FR-ORG-038
func TestSetLegalLevels_EntryEqualToConfiguredRestores(t *testing.T) {
	t.Parallel()
	w, rec := orgSlotWorld(t)
	if err := w.SetLegalLevels([]LegalLevel{{Control: 0, CapBP: 5_000}}); err != nil {
		t.Fatalf("SetLegalLevels(5 000) error = %v, want nil", err)
	}
	if capPrice, floor, _, _ := w.org.rangeOf(orgB, 0); capPrice != 50 || floor != 0 {
		t.Fatalf("rangeOf(b, grain) after the call = (%d, %d), want (50, 0)", capPrice, floor)
	}
	if err := w.SetLegalLevels([]LegalLevel{{Control: 0, CapBP: 12_000, FloorBP: 8_000}}); err != nil {
		t.Fatalf("SetLegalLevels(the configured level) error = %v, want nil", err)
	}
	if capPrice, floor, _, _ := w.org.rangeOf(orgB, 0); capPrice != 120 || floor != 80 {
		t.Errorf("rangeOf(b, grain) after the restore = (%d, %d), want (120, 80)", capPrice, floor)
	}
	if s, _ := w.Slots(); s.LegalLevels != nil {
		t.Errorf("Slots().LegalLevels after the restore = %+v, want none", s.LegalLevels)
	}
	if got := of[LegalLevelSet](rec); len(got) != 2 {
		t.Errorf("LegalLevelSet events = %d, want 2 (the change and the restore, each a change of the level in force)", len(got))
	}
}

// TestSetLegalLevels_RefusedCallChangesNothing verifies that a call with a valid entry and
// an invalid one is refused whole: no range moves, no event is emitted and the slot state is
// empty (spec-institutions.md 7.1.1, atomic).
//
// Requirement: FR-ORG-036
func TestSetLegalLevels_RefusedCallChangesNothing(t *testing.T) {
	t.Parallel()
	w, rec := orgSlotWorld(t)
	err := w.SetLegalLevels([]LegalLevel{{Control: 0, CapBP: 5_000}, {Control: 2, CapBP: MaxCreditRateBP + 1}})
	if !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("SetLegalLevels(a rate cap above MaxCreditRateBP) error = %v, want ErrOutOfBounds", err)
	}
	if capPrice, floor, _, _ := w.org.rangeOf(orgB, 0); capPrice != 120 || floor != 80 {
		t.Errorf("rangeOf(b, grain) after a refused call = (%d, %d), want (120, 80)", capPrice, floor)
	}
	if got := of[LegalLevelSet](rec); len(got) != 0 {
		t.Errorf("LegalLevelSet events after a refused call = %+v, want none", got)
	}
	if s, _ := w.Slots(); s.LegalLevels != nil {
		t.Errorf("Slots().LegalLevels after a refused call = %+v, want none", s.LegalLevels)
	}
}

// TestSetRightSuspensions_RebuildsTable verifies that a call rebuilds the resolved rights:
// a suspended right leaves its key open and no outer right is consulted, lifting it puts the
// right back, and the event of each changed right comes in ascending RightID.
//
// Arithmetic: the sale of grain in a is held by sell_inland (3), in b by sell_crown (2).
// Suspending sell_inland opens the sale in a (no fall back to sell_crown); b is untouched.
//
// Requirement: FR-ORG-015, FR-ORG-037, FR-ORG-042
func TestSetRightSuspensions_RebuildsTable(t *testing.T) {
	t.Parallel()
	w, rec := orgSlotWorld(t)
	o := w.org
	if err := w.SetRightSuspensions([]RightSuspension{{Right: 3, Suspended: true}, {Right: 0, Suspended: false}, {Right: 1, Suspended: true}}); err != nil {
		t.Fatalf("SetRightSuspensions() error = %v, want nil", err)
	}
	orgWantRight(t, o, actSell, 0, orgA, 0, true)
	orgWantRight(t, o, actSell, 0, orgB, 2, false)
	orgWantRight(t, o, actProduce, int(orgRecipe(t, w, "farm")), orgB, 0, true)
	want := []RightSuspensionSet{{Tick: 0, Right: 1, Suspended: true}, {Tick: 0, Right: 3, Suspended: true}}
	if got := of[RightSuspensionSet](rec); !slices.Equal(got, want) {
		t.Errorf("RightSuspensionSet events = %+v, want %+v (right 0 was not suspended: no event)", got, want)
	}
	if s, _ := w.Slots(); !slices.Equal(s.RightSuspensions, []RightSuspension{{Right: 1, Suspended: true}, {Right: 3, Suspended: true}}) {
		t.Errorf("Slots().RightSuspensions = %+v, want rights 1 and 3", s.RightSuspensions)
	}
	if err := w.SetRightSuspensions([]RightSuspension{{Right: 3, Suspended: false}, {Right: 1}}); err != nil {
		t.Fatalf("SetRightSuspensions(lift) error = %v, want nil", err)
	}
	orgWantRight(t, o, actSell, 0, orgA, 3, false)
	if s, _ := w.Slots(); s.RightSuspensions != nil {
		t.Errorf("Slots().RightSuspensions after the lift = %+v, want none", s.RightSuspensions)
	}
}

// TestAgentState_Membership verifies the membership group of an agent: a member shows its
// body, a household that waits for a place and one in no body show the zero value, and a
// world with the module off shows it for every agent.
//
// Arithmetic: guild has 5 places for the 8 smiths of b, so 5 are admitted at the first
// round (the order of the key decides which, the test reads the members from the module) and 3 wait; the farmers of b are in no body.
//
// Requirement: FR-ORG-010, FR-INT-032
func TestAgentState_Membership(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgToy())
	step(t, w, 1)
	members := orgMembers(w, 0)
	var shown int
	for _, id := range orgAgents(w, orgB, orgSmith) {
		st, err := w.Agent(AgentID(id))
		if err != nil {
			t.Fatalf("Agent(%d) error = %v, want nil", id, err)
		}
		want := Membership{}
		if slices.Contains(members, id) {
			want = Membership{Member: true, Body: 0}
			shown++
		}
		if st.Membership != want {
			t.Errorf("Agent(smith %d of b).Membership = %+v, want %+v", id, st.Membership, want)
		}
	}
	if shown != 5 {
		t.Errorf("smiths of b that show a membership = %d, want 5 (the places of guild)", shown)
	}
	if st, _ := w.Agent(AgentID(orgAgents(w, orgB, orgFarmer)[0])); st.Membership != (Membership{}) {
		t.Errorf("Agent(farmer of b).Membership = %+v, want the zero value (no body)", st.Membership)
	}
	off := orgToy()
	off.Modules.Institutions = nil
	w2 := newMill(t, off)
	step(t, w2, 1)
	if st, _ := w2.Agent(AgentID(w2.firstHousehold)); st.Membership != (Membership{}) {
		t.Errorf("Agent().Membership with the module off = %+v, want the zero value", st.Membership)
	}
}
