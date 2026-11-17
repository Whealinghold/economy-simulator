package economysim_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s111Seeds is the full seed count of TestScenarioS111_OneHolderCapAndFork, reduced by
// invariantSeeds under -race.
const s111Seeds = 3

// s111Pack is pack one with the resources overlay and the tenure overlay, the posted rent
// stepping (so that it is saved) inside a band, and the cap of one agent's share at capBP.
func s111Pack(t *testing.T, capBP int64) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, append(slices.Clone(packOneResources), "overlay/tenure.toml")...)
	m := *cfg.Modules.Tenure
	m.RentStepBP, m.RentFloorBP, m.RentCeilingBP, m.MaxHolderShareBP = 500, 5_000, 15_000, capBP
	cfg.Modules.Tenure = &m
	return cfg
}

// s111Endowment finds the region and the resource of the overlay's titled endowment, "arable"
// in "north", in the names table of a world.
func s111Endowment(t *testing.T, w *economysim.World) (economysim.RegionID, economysim.ResourceID) {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	g, r := slices.Index(s.Names.Regions, "north"), slices.Index(s.Names.Resources, "arable")
	if g < 0 || r < 0 {
		t.Fatalf("the overlay's endowment is not in the names table %v, %v", s.Names.Regions, s.Names.Resources)
	}
	return economysim.RegionID(g), economysim.ResourceID(r)
}

// s111MoveAll returns the entries that give every title of the endowment to the first agent
// that holds one, at the price per entry.
func s111MoveAll(t *testing.T, w *economysim.World, g economysim.RegionID, r economysim.ResourceID, price economysim.Money) (economysim.AgentID, []economysim.TitleTransfer) {
	t.Helper()
	st, err := w.Holders(g, r)
	if err != nil {
		t.Fatalf("Holders error = %v, want nil", err)
	}
	var target economysim.AgentID = economysim.NoAgent
	var moves []economysim.TitleTransfer
	for _, x := range st.Titles {
		switch {
		case x.Holder == economysim.AgentID(g):
			moves = append(moves, economysim.TitleTransfer{Region: g, Resource: r, From: x.Holder, ShareBP: x.ShareBP, Price: price})
		case target == economysim.NoAgent:
			target = x.Holder
		default:
			moves = append(moves, economysim.TitleTransfer{Region: g, Resource: r, From: x.Holder, ShareBP: x.ShareBP, Price: price})
		}
	}
	for i := range moves {
		moves[i].To = target
	}
	return target, moves
}

// s111Money is the money of every household and every pool, from the snapshot.
func s111Money(t *testing.T, w *economysim.World) (sum economysim.Money) {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	for _, g := range s.Regions {
		sum += g.MoneyHouseholds + g.MoneyPool
	}
	return sum
}

// TestScenarioS111_OneHolderCapAndFork verifies LND-S5 of spec-land.md 16.4: a consumer
// moves every title of the endowment to one agent. With the cap off the call succeeds and
// one agent holds all the agents' shares; with the cap on the same call returns ErrTitleCap
// and changes nothing; a price on each entry moves money by title_sale and creates none
// (the pools and the households together hold the same sum, and the audit passes through
// ten more ticks); and a checkpoint of the world with the titles moved and the posted rent
// stepping restores bit-identically (the fork matrix of CHK 8). Pack one with the resources
// and tenure overlays, 10 ticks before the calls, WithAudit, the seeds 1 to s111Seeds.
//
// Requirement: FR-LND-025, FR-LND-034, FR-LND-041
func TestScenarioS111_OneHolderCapAndFork(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, seed := range seedList(invariantSeeds(s111Seeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			s111Run(t, seed)
		})
	}
}

// s111Run is one seed of S111.
func s111Run(t *testing.T, seed uint64) {
	t.Helper()
	capped := mustWorld(t, s111Pack(t, 1_000), economysim.WithSeed(seed), economysim.WithAudit())
	free := mustWorld(t, s111Pack(t, 0), economysim.WithSeed(seed), economysim.WithAudit())
	for _, w := range []*economysim.World{capped, free} {
		s111Steps(t, w, 10)
	}
	g, r := s111Endowment(t, free)
	s111Cap(t, capped, g, r)
	target, moves := s111MoveAll(t, free, g, r, 0)
	money := s111Money(t, free)
	if err := free.TransferTitles(moves); err != nil {
		t.Fatalf("seed %d: TransferTitles with the cap off error = %v, want nil", seed, err)
	}
	st, _ := free.Holders(g, r)
	if len(st.Titles) != 1 || st.Titles[0].Holder != target {
		t.Errorf("seed %d: after the move the titles are %+v, want one holder", seed, st)
	}
	if got := s111Money(t, free); got != money {
		t.Errorf("seed %d: money %d after gifts, want %d", seed, got, money)
	}
	s111Steps(t, free, 10)
	s111Fork(t, seed)
}

// s111Steps runs n ticks of w.
func s111Steps(t *testing.T, w *economysim.World, n int) {
	t.Helper()
	for range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
}

// s111Cap checks that the call that moves every title to one agent is refused under the cap
// with ErrTitleCap and changes no title.
func s111Cap(t *testing.T, w *economysim.World, g economysim.RegionID, r economysim.ResourceID) {
	t.Helper()
	_, moves := s111MoveAll(t, w, g, r, 0)
	before, _ := w.Holders(g, r)
	if err := w.TransferTitles(moves); !errors.Is(err, economysim.ErrTitleCap) {
		t.Errorf("moving every title to one agent under a cap of 1000 = %v, want ErrTitleCap", err)
	}
	if after, _ := w.Holders(g, r); !slices.Equal(before.Titles, after.Titles) {
		t.Errorf("a refused call changed the titles")
	}
}

// s111Fork runs the fork matrix on the world with the titles moved and sold for a price.
func s111Fork(t *testing.T, seed uint64) {
	t.Helper()
	cfg := s111Pack(t, 0)
	call := func(w *economysim.World) {
		g, r := s111Endowment(t, w)
		_, moves := s111MoveAll(t, w, g, r, 1)
		if err := w.TransferTitles(moves); err != nil {
			t.Fatalf("seed %d: the call before the checkpoint error = %v, want nil", seed, err)
		}
		if err := w.SetRentScales([]economysim.EndowmentScale{{Region: g, Resource: r, BP: 7_000}}); err != nil {
			t.Fatalf("seed %d: SetRentScales error = %v, want nil", seed, err)
		}
	}
	for _, n := range []int{0, 10} {
		forkAtN(t, forkCase{Name: "one+resources+tenure", Cfg: cfg, Seed: seed, N: n, M: forkM, Before: call})
	}
}
