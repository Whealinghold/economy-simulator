package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s111GiveAll adds an external agent with money to the region of the endowment, which
// RemoveAgent accepts (it removes only external agents), and gives it every title of the
// endowment at the price per entry.
func s111GiveAll(t *testing.T, w *economysim.World, g economysim.RegionID, r economysim.ResourceID, price economysim.Money) economysim.AgentID {
	t.Helper()
	id, err := w.AddAgent(economysim.NewAgent{Region: g, Money: 1_000_000})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	st, err := w.Holders(g, r)
	if err != nil {
		t.Fatalf("Holders error = %v, want nil", err)
	}
	var moves []economysim.TitleTransfer
	for _, x := range st.Titles {
		moves = append(moves, economysim.TitleTransfer{Region: g, Resource: r, From: x.Holder, To: id, ShareBP: x.ShareBP, Price: price})
	}
	if err := w.TransferTitles(moves); err != nil {
		t.Fatalf("TransferTitles error = %v, want nil", err)
	}
	return id
}

// s111Titled is the share of the endowment that has a holder, the pool's and the
// households' together; the rest is the commons.
func s111Titled(t *testing.T, w *economysim.World, g economysim.RegionID, r economysim.ResourceID) int64 {
	t.Helper()
	st, err := w.Holders(g, r)
	if err != nil {
		t.Fatalf("Holders error = %v, want nil", err)
	}
	return 10_000 - st.CommonsBP
}

// s111Seed is the seed of the tests of the tenure calls that need only one world.
const s111Seed = 1

// TestTenureRemoval_PublicAPI_TitlesReachThePoolBeforeTheRemoval verifies, through the
// exported API of a world built from the files, that removing the agent that holds every
// title of an endowment gives the whole share back to the pool of the endowment's region
// (cause removal, no price), that the event comes before AgentBankrupted, that Titles
// then reports the agent as removed and the pool as the holder, and that the audit passes
// through ten more ticks.
//
// Requirement: FR-LND-023, FR-LND-027, FR-LND-029
func TestTenureRemoval_PublicAPI_TitlesReachThePoolBeforeTheRemoval(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	rec := &recorder{}
	w := mustWorld(t, s111Pack(t, 0), economysim.WithSeed(s111Seed), economysim.WithAudit(), economysim.WithObserver(rec))
	s111Steps(t, w, 5)
	g, r := s111Endowment(t, w)
	titled := s111Titled(t, w, g, r)
	target := s111GiveAll(t, w, g, r, 0)
	rec.forget()
	if err := w.RemoveAgent(target); err != nil {
		t.Fatalf("RemoveAgent error = %v, want nil", err)
	}
	moved, bankrupted, e := removalEvents(rec.events, target)
	if moved < 0 || bankrupted < 0 || moved > bankrupted {
		t.Errorf("TitleTransferred at %d, AgentBankrupted at %d, want the first before the second", moved, bankrupted)
	}
	want := economysim.TitleTransferred{Tick: e.Tick, Region: g, Resource: r, From: target, To: economysim.AgentID(g), ShareBP: titled,
		Cause: economysim.CauseRemoval, FromAfter: 0, ToAfter: titled, FromRegion: e.FromRegion, ToRegion: g, ToPool: true}
	if e != want {
		t.Errorf("removal event = %+v, want %+v: the whole titled share to the pool of the region, no price", e, want)
	}
	if _, err := w.Titles(target); !errors.Is(err, economysim.ErrAgentRemoved) {
		t.Errorf("Titles of the removed agent = %v, want ErrAgentRemoved", err)
	}
	pool, err := w.Titles(economysim.AgentID(g))
	wantPool := []economysim.TitleState{{Region: g, Resource: r, Holder: economysim.AgentID(g), ShareBP: titled}}
	if err != nil || !slices.Equal(pool, wantPool) {
		t.Errorf("Titles of the pool = %+v, %v, want %+v", pool, err, wantPool)
	}
	s111Steps(t, w, 10)
}

// removalEvents returns the positions of the TitleTransferred that takes a title from the
// agent and of its AgentBankrupted in events (-1 when absent), and that TitleTransferred.
func removalEvents(events []economysim.Event, agent economysim.AgentID) (moved, bankrupted int, title economysim.TitleTransferred) {
	moved, bankrupted = -1, -1
	for i, e := range events {
		switch e := e.(type) {
		case economysim.TitleTransferred:
			if e.From == agent {
				moved, title = i, e
			}
		case economysim.AgentBankrupted:
			if e.Agent == agent {
				bankrupted = i
			}
		}
	}
	return moved, bankrupted, title
}

// TestTenureCalls_OutsideNodeIsRefused verifies that no call of the tenure module accepts
// an outside node: the calls that name its region return ErrOutsideNode, the query of its
// estate returns ErrNotAHolder, and an estate of a node is not a holder either.
//
// Requirement: FR-LND-025, FR-LND-026, FR-LND-027
func TestTenureCalls_OutsideNodeIsRefused(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := s111Pack(t, 0)
	withOutsideNode(&cfg)
	w := mustWorld(t, cfg, economysim.WithSeed(s111Seed))
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	coast := economysim.RegionID(slices.Index(s.Names.Regions, "coast"))
	g, r := s111Endowment(t, w)
	node := economysim.AgentID(coast)
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"TransferTitles of a node", w.TransferTitles([]economysim.TitleTransfer{{Region: coast, From: economysim.NoAgent, To: node, ShareBP: 1}}), economysim.ErrOutsideNode},
		{"TransferTitles to the estate of a node", w.TransferTitles([]economysim.TitleTransfer{{Region: g, Resource: r, From: economysim.NoAgent, To: node, ShareBP: 1}}), economysim.ErrNotAHolder},
		{"SetRentScales of a node", w.SetRentScales([]economysim.EndowmentScale{{Region: coast, BP: 1}}), economysim.ErrOutsideNode},
		{"Holders of a node", errOnly(w.Holders(coast, r)), economysim.ErrOutsideNode},
		{"Rent of a node", errOnly(w.Rent(coast, r)), economysim.ErrOutsideNode},
		{"Titles of the estate of a node", errOnly(w.Titles(node)), economysim.ErrNotAHolder},
	}
	for _, tc := range cases {
		if !errors.Is(tc.err, tc.want) {
			t.Errorf("%s: error = %v, want %v", tc.name, tc.err, tc.want)
		}
	}
}

// errOnly returns the error of a call whose first result the test does not need.
func errOnly[T any](_ T, err error) error { return err }

// TestFork_TenureAfterRemoval verifies the fork matrix of the checkpoint (spec-land.md
// 14, plan 3-lnd phase 4: after a removal): on the full pack with the tenure overlay, a
// consumer moves every title to one agent with a price and then removes that agent, so the
// titles have come back to the pool; the world checkpointed at the boundary, restored and
// restored again gives the same events, snapshots and bytes as the original for the next
// ticks, at the boundary right after the calls and ten ticks later.
//
// Requirement: FR-LND-023, FR-LND-034
func TestFork_TenureAfterRemoval(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := s111Pack(t, 0)
	call := func(w *economysim.World) {
		g, r := s111Endowment(t, w)
		target := s111GiveAll(t, w, g, r, 1)
		if err := w.RemoveAgent(target); err != nil {
			t.Fatalf("RemoveAgent error = %v, want nil", err)
		}
	}
	for _, seed := range seedList(invariantSeeds(s111Seeds)) {
		for _, n := range []int{0, 10} {
			forkAtN(t, forkCase{Name: "one+resources+tenure+removal", Cfg: cfg, Seed: seed, N: n, M: forkM, Before: call})
		}
	}
}
