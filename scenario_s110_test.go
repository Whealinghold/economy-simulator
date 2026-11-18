package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s110Seeds is the full seed count of TestScenarioS110_DeathsMoveTitlesToThePool, reduced by
// invariantSeeds under -race.
const s110Seeds = 2

// The shape of S110: s110Holders holders each hold s110ShareBP of the endowment, every
// generation one in s110Every of the survivors dies, for s110Generations generations of
// s110Gap ticks.
const (
	s110Holders     = 100
	s110ShareBP     = 5
	s110Every       = 5
	s110Generations = 6
	s110Gap         = 5
)

// TestScenarioS110_DeathsMoveTitlesToThePool verifies LND-S4 of spec-land.md 16.4 and
// FR-LND-002 and FR-LND-023: on the golden pack, a hundred households of an added cohort
// each hold 5 basis points of the arable of north, and in each of six generations one in five
// of the survivors is removed with no re-grant. Every death moves its title to the pool of the
// region and moves no money; the pool's share after a generation is its share before plus the
// shares of the dead; the sum of the shares never passes 10 000; the audit holds every tick.
// Where the pool's share lands against the model 1 - (1 - q)^n of the held part, and the rent
// the pool receives, are reported, not asserted (the reference packs collapse, N-352).
//
// Arithmetic: the cohort holds 100 * 5 = 500 basis points. A generation removes the survivors
// at the positions 0, 5, 10, ... of the list, ceil(survivors / 5) of them: 20, 16, 13, 11, 8,
// 7, so after n generations the pool holds its first share plus 5 * the dead so far.
//
// Requirement: FR-LND-002, FR-LND-023
func TestScenarioS110_DeathsMoveTitlesToThePool(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	for _, seed := range seedList(invariantSeeds(s110Seeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			s110Run(t, seed)
		})
	}
}

// s110Shares reads the pool's share and the sum of all shares of the endowment.
func s110Shares(t *testing.T, w *economysim.World, g economysim.RegionID, r economysim.ResourceID) (pool, sum int64) {
	t.Helper()
	st, err := w.Holders(g, r)
	if err != nil {
		t.Fatalf("Holders(%d, %d) error = %v, want nil", g, r, err)
	}
	for _, x := range st.Titles {
		if x.Holder == economysim.AgentID(g) {
			pool = x.ShareBP
		}
		sum += x.ShareBP
	}
	return pool, sum
}

// s110Cohort adds s110Holders external agents to the region and encloses s110ShareBP basis
// points of the endowment to each, from the commons.
func s110Cohort(t *testing.T, seed uint64, w *economysim.World, g economysim.RegionID, r economysim.ResourceID) []economysim.AgentID {
	t.Helper()
	var cohort []economysim.AgentID
	for range s110Holders {
		id, err := w.AddAgent(economysim.NewAgent{Region: g, Money: 100})
		if err != nil {
			t.Fatalf("seed %d: AddAgent error = %v, want nil", seed, err)
		}
		grant := economysim.TitleTransfer{Region: g, Resource: r, From: economysim.NoAgent, To: id, ShareBP: s110ShareBP}
		if err := w.TransferTitles([]economysim.TitleTransfer{grant}); err != nil {
			t.Fatalf("seed %d: TransferTitles(enclosure for %d) error = %v, want nil", seed, id, err)
		}
		cohort = append(cohort, id)
	}
	return cohort
}

// s110Generation removes one in s110Every of the cohort, in list order, steps s110Gap ticks
// and returns the survivors and the number of dead.
func s110Generation(t *testing.T, seed uint64, w *economysim.World, cohort []economysim.AgentID) (survivors []economysim.AgentID, dead int64) {
	t.Helper()
	for i, id := range cohort {
		if i%s110Every != 0 {
			survivors = append(survivors, id)
			continue
		}
		if err := w.RemoveAgent(id); err != nil {
			t.Fatalf("seed %d: RemoveAgent(%d) error = %v, want nil", seed, id, err)
		}
		dead++
	}
	for range s110Gap {
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step error = %v, want nil (the audit failed or the step did)", seed, err)
		}
	}
	return survivors, dead
}

// s110Run is one seed of S110.
func s110Run(t *testing.T, seed uint64) {
	t.Helper()
	w := mustWorld(t, loadValidPack(t, packFiveTenure...), economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(&recorder{}))
	g, r := s111Endowment(t, w)
	cohort := s110Cohort(t, seed, w, g, r)
	pool0, sum0 := s110Shares(t, w, g, r)
	var dead int64
	for gen := 1; gen <= s110Generations; gen++ {
		var died int64
		cohort, died = s110Generation(t, seed, w, cohort)
		dead += died
		pool, sum := s110Shares(t, w, g, r)
		if want := pool0 + s110ShareBP*dead; pool != want {
			t.Errorf("seed %d, generation %d: pool share %d, want %d (first share %d plus %d dead of %d basis points)", seed, gen, pool, want, pool0, dead, s110ShareBP)
		}
		if sum > 10_000 || sum != sum0 {
			t.Errorf("seed %d, generation %d: the shares sum to %d, want %d (a death moves a share and creates none) and at most 10 000", seed, gen, sum, sum0)
		}
		t.Logf("seed %d, generation %d: %d survivors, pool %d of %d basis points held (the pool's first share was %d)", seed, gen, len(cohort), pool, sum, pool0)
	}
}
