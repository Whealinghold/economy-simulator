package economysim

import (
	"errors"
	"reflect"
	"testing"
)

// addHolder adds an external agent in a region with money and gives it a share of the ore
// of home from the pool, failing the test on an error.
func addHolder(t testing.TB, w *World, region RegionID, money Money, share int64) AgentID {
	t.Helper()
	id, err := w.AddAgent(NewAgent{Region: region, Money: money})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	if share > 0 {
		if err := w.TransferTitles([]TitleTransfer{tr(0, 0, id, share, 0)}); err != nil {
			t.Fatalf("the gift of %d to agent %d error = %v, want nil", share, id, err)
		}
	}
	return id
}

// TestRemoval_TitlesToPoolOfEndowmentRegion_BeforeAgentBankrupted_NoMoney verifies that a
// removed agent's titles go to the pool of each endowment's region, not the region of the
// agent's account, one TitleTransferred of cause removal each in ascending (region,
// resource), before AgentBankrupted, with no price and no money moving for them.
//
// Requirement: FR-LND-023, FR-LND-029
func TestRemoval_TitlesToPoolOfEndowmentRegion_BeforeAgentBankrupted_NoMoney(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, nil, WithObserver(obs))
	id := addHolder(t, w, 1, 500, 70) // lives in yonder, holds 70 of the ore of home
	if err := w.TransferTitles([]TitleTransfer{tr(1, 1, id, 40, 0)}); err != nil {
		t.Fatalf("the gift of the ore of yonder error = %v, want nil", err)
	}
	obs.events = nil
	if err := w.RemoveAgent(id); err != nil {
		t.Fatalf("RemoveAgent error = %v, want nil", err)
	}
	var kinds []string
	for _, e := range obs.events {
		switch e.(type) {
		case TitleTransferred:
			kinds = append(kinds, "title")
		case AgentBankrupted:
			kinds = append(kinds, "bankrupted")
		}
	}
	if !reflect.DeepEqual(kinds, []string{"title", "title", "bankrupted"}) {
		t.Fatalf("event order = %v, want two titles then AgentBankrupted", kinds)
	}
	evs := of[TitleTransferred](obs)
	if evs[0].Region != 0 || evs[0].To != 0 || evs[0].ShareBP != 70 || evs[0].Cause != CauseRemoval || evs[0].Price != 0 || evs[0].ToAfter != 1_000 || evs[0].FromRegion != 1 || evs[0].ToRegion != 0 || !evs[0].ToPool {
		t.Errorf("first event = %+v, want the title of home to the pool of home, from an agent of yonder", evs[0])
	}
	if evs[1].Region != 1 || evs[1].To != 1 || evs[1].ShareBP != 40 || evs[1].ToAfter != 500 {
		t.Errorf("second event = %+v, want the title of yonder to the pool of yonder", evs[1])
	}
	if got, want := tnPairs(tnHolders(t, w, 0)), [][2]int64{{0, 1_000}, {2, 100}, {3, 100}, {4, 100}}; !reflect.DeepEqual(got, want) {
		t.Errorf("titles of home after the removal = %v, want %v", got, want)
	}
	if _, err := w.Titles(id); !errors.Is(err, ErrAgentRemoved) {
		t.Errorf("Titles of the removed agent = %v, want ErrAgentRemoved", err)
	}
}

// TestRemoval_NoTitleNoEvent verifies that the removal of an agent with no title emits no
// TitleTransferred.
//
// Requirement: FR-LND-023
func TestRemoval_NoTitleNoEvent(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, nil, WithObserver(obs))
	id := addHolder(t, w, 0, 10, 0)
	if err := w.RemoveAgent(id); err != nil {
		t.Fatalf("RemoveAgent error = %v, want nil", err)
	}
	if n := len(of[TitleTransferred](obs)); n != 0 {
		t.Errorf("%d TitleTransferred events for an agent with no title, want 0", n)
	}
}

// TestRelocation_KeepsTitles_NoRegionBound verifies that a relocation leaves the titles
// with the agent and registers no region_bound refusal (spec-land.md 6.5, 6.6).
//
// Requirement: FR-LND-024
func TestRelocation_KeepsTitles_NoRegionBound(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, nil)
	id := addHolder(t, w, 0, 10, 60)
	if err := w.Relocate(id, 1); err != nil {
		t.Fatalf("Relocate of a title holder error = %v, want nil: a title is no region_bound obligation", err)
	}
	ts, err := w.Titles(id)
	if err != nil || len(ts) != 1 || ts[0].ShareBP != 60 || ts[0].Region != 0 {
		t.Errorf("Titles after the relocation = %+v, %v, want the 60 of home", ts, err)
	}
}

// TestRemoval_CohortWeightChangeKeepsTitles verifies that a title belongs to the agent and
// not to its households: a cut in weight changes no share (spec-land.md 6.5).
//
// Requirement: FR-LND-024
func TestRemoval_CohortWeightChangeKeepsTitles(t *testing.T) {
	t.Parallel()
	c := oreMill(5, 25, 1_000)
	c.Modules.Tenure = &TenureConfig{Holdings: []Holding{
		{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: "miller"}, ShareBP: 10},
	}}
	w := newMill(t, c, WithAudit())
	before := tnPairs(tnHolders(t, w, 0))
	a := &w.agents[w.firstHousehold]
	a.weight--
	w.live.households--
	if got := tnPairs(tnHolders(t, w, 0)); !reflect.DeepEqual(got, before) {
		t.Errorf("titles after a weight change = %v, want %v unchanged", got, before)
	}
}

// TestRemoval_PoolShareKeptWhenTheAgentIsTheLastTitle verifies that the pool of a region
// above 0 keeps the share it held when the removed agent's title is the last of a list of
// two: the removal moves the agent's share onto the pool's, it does not replace it.
//
// Fix: the removal read the pool's share after deleting the agent's title in place, from the
// list header of the old length, whose last element was zeroed; the search for the pool
// missed it and the pool was left with the moved share alone.
func TestRemoval_PoolShareKeptWhenTheAgentIsTheLastTitle(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, func(m *TenureConfig) {
		m.Holdings = []Holding{
			{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindPool}, ShareBP: 1_000},
			{Region: "yonder", Resource: "ore", Holder: HolderRef{Kind: HolderKindPool}, ShareBP: 500},
		}
	})
	id := addHolder(t, w, 1, 500, 0)
	if err := w.TransferTitles([]TitleTransfer{tr(1, 1, id, 40, 0)}); err != nil {
		t.Fatalf("the gift of the ore of yonder error = %v, want nil", err)
	}
	if err := w.RemoveAgent(id); err != nil {
		t.Fatalf("RemoveAgent error = %v, want nil", err)
	}
	got, err := w.Holders(1, 0)
	if err != nil {
		t.Fatalf("Holders(1, 0) error = %v, want nil", err)
	}
	if want := [][2]int64{{1, 500}}; !reflect.DeepEqual(tnPairs(got), want) {
		t.Errorf("titles of yonder after the removal = %v, want %v", tnPairs(got), want)
	}
}
