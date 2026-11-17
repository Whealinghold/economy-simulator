package economysim

import (
	"errors"
	"reflect"
	"testing"
)

// TestTenureConstruct_GrantsEachHouseholdOfTheRoleAndThePool verifies that the titles of
// the holdings exist at construction: each household of the role in the region holds
// weight times ShareBP, in ascending agent ID with the pool first, the commons share is
// what is left, a household of another role holds nothing, and construction emits no event
// for a title.
//
// Requirement: FR-LND-005, FR-LND-002
func TestTenureConstruct_GrantsEachHouseholdOfTheRoleAndThePool(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, nil, WithObserver(obs))
	home, yonder := tnHolders(t, w, 0), tnHolders(t, w, 1)
	if got, want := tnPairs(home), [][2]int64{{0, 1_000}, {2, 100}, {3, 100}, {4, 100}}; !reflect.DeepEqual(got, want) {
		t.Errorf("titles of home = %v, want %v", got, want)
	}
	if got, want := tnPairs(yonder), [][2]int64{{1, 500}, {6, 50}, {7, 50}}; !reflect.DeepEqual(got, want) {
		t.Errorf("titles of yonder = %v, want %v", got, want)
	}
	if home.CommonsBP != 8_700 || yonder.CommonsBP != 9_400 {
		t.Errorf("commons = %d and %d, want 8700 and 9400", home.CommonsBP, yonder.CommonsBP)
	}
	if len(of[TitleTransferred](obs)) != 0 || len(of[RentScaled](obs)) != 0 {
		t.Errorf("construction emitted a title event, want none")
	}
	if home.Posted != 4 || yonder.Posted != 0 || home.ScaleBP != wholeBP {
		t.Errorf("posted rent and scale of home = %d and %d, of yonder %d, want 4, 10000 and 0", home.Posted, home.ScaleBP, yonder.Posted)
	}
}

// TestTenureConstruct_CohortHoldsWeightTimesShare verifies that a cohort of weight w is
// one holder with a share of w times ShareBP, merged into one title, whatever its weight
// (spec-land.md 3.2.1, 6.1.1).
//
// Requirement: FR-LND-005
func TestTenureConstruct_CohortHoldsWeightTimesShare(t *testing.T) {
	t.Parallel()
	c := oreMill(5, 25, 1_000)
	c.Modules.Tenure = &TenureConfig{Holdings: []Holding{
		{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: "miller"}, ShareBP: 10},
	}}
	w := newMill(t, c)
	st := tnHolders(t, w, 0)
	var weights, shares int64
	w.eachLive(func(a *agent) {
		weights += a.weight
		if got := w.tenure.shareOf(0, a.id); got != a.weight*10 {
			t.Errorf("agent %d of weight %d holds %d, want %d", a.id, a.weight, got, a.weight*10)
		}
	})
	for _, x := range st.Titles {
		shares += x.ShareBP
	}
	if weights != 25 || shares != 250 || st.CommonsBP != 9_750 {
		t.Errorf("weights %d, shares %d, commons %d, want 25, 250 and 9750", weights, shares, st.CommonsBP)
	}
}

// TestTenureConstruct_RoleHolderCap_ErrTitleCap verifies that a role holder whose
// largest household would pass MaxHolderShareBP is refused at NewWorld.
//
// Requirement: FR-LND-041
func TestTenureConstruct_RoleHolderCap_ErrTitleCap(t *testing.T) {
	t.Parallel()
	_, err := NewWorld(tenureWorldConfig(func(m *TenureConfig) { m.MaxHolderShareBP = 99 }))
	if !errors.Is(err, ErrTitleCap) {
		t.Errorf("NewWorld with a cap of 99 under a share of 100 = %v, want ErrTitleCap", err)
	}
	if _, err := NewWorld(tenureWorldConfig(func(m *TenureConfig) { m.MaxHolderShareBP = 100 })); err != nil {
		t.Errorf("NewWorld with a cap of 100 = %v, want nil", err)
	}
}
