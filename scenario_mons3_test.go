package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

const (
	// mons3Ticks is the horizon of MON-S3 on pack one. The derivation of
	// spec-money-supply.md 8.6 is 3 500 ticks of one added household a tick; pack
	// one's entry is the spawner's, so the run is short and only reports.
	mons3Ticks int64 = 400
	// mons3Grant is G of 8.6: a pool grant of 3 000 a tick, h = 1 entrant at a
	// start money of s = 3 000.
	mons3Grant = 3_000
)

// TestScenarioMONS3_PoolGrant verifies MON-S3 (spec-money-supply.md 8.6, 16) on pack
// one with the estates module on, with and without a grant of 3 000 a tick to the
// pool of the north, seeds 1 and 2 (1 under -race), 400 ticks, with the audit on.
//
// Exact: the money supply at the end is its value at the start plus the grant
// total of the flow (0 without a grant), the boundary balance of the flow is minus
// its total, and the audit holds at every tick. Reported only (the thresholds are
// placeholders, N-015, and no band exists until STY): the money of the households
// per household at ticks 0, 100, 200 and 400, the household-ticks unfed, the grant
// total, relief paid and the pool at the end.
//
// Requirement: FR-MON-046, FR-MON-019
//
// Guard: a pool grant is money entering by a named flow and by exactly what the
// flow says, and it reaches households only by the pool (entry and relief).
func TestScenarioMONS3_PoolGrant(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	base := loadValidPack(t, append(slices.Clone(packOne), "overlay/estates.toml")...)
	withGrant := base
	withGrant.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{{ID: "pool_grant", Kind: economysim.FlowGrant,
		Target: economysim.FlowTarget{Kind: economysim.TargetPool, Region: "north"}, Amount: mons3Grant}}}
	for _, leg := range []struct {
		name string
		cfg  economysim.Config
	}{{"without a grant", base}, {"with a pool grant", withGrant}} {
		for _, seed := range seedList(min(2, scenarioSeedCount())) {
			t.Run(fmt.Sprintf("%s/seed %d", leg.name, seed), func(t *testing.T) {
				t.Parallel()
				mons3Run(t, leg.cfg, seed)
			})
		}
	}
}

// mons3Run runs one leg and seed of MON-S3.
func mons3Run(t *testing.T, cfg economysim.Config, seed uint64) {
	t.Helper()
	scenSem <- struct{}{}
	defer func() { <-scenSem }()
	var unfed, relief int64
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		switch e := e.(type) {
		case economysim.UpkeepShortfall:
			unfed += e.Unfed
		case economysim.ReliefPaid:
			relief += int64(e.Paid)
		}
	})
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(obs))
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	snap := func() economysim.Snapshot {
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("seed %d: Snapshot error = %v, want nil", seed, err)
		}
		return s
	}
	perHousehold := func(s economysim.Snapshot) int64 {
		var held economysim.Money
		for _, r := range s.Regions {
			held += r.MoneyHouseholds
		}
		return int64(held) / max(s.Households, 1)
	}
	start := snap()
	series := []int64{perHousehold(start)}
	for tick := int64(1); tick <= mons3Ticks; tick++ {
		if err := w.Step(); err != nil { // the audit runs in every step
			t.Fatalf("seed %d: Step at tick %d error = %v, want nil", seed, tick-1, err)
		}
		if tick == 100 || tick == 200 || tick == mons3Ticks {
			series = append(series, perHousehold(snap()))
		}
	}
	end := snap()
	var total economysim.Money
	for _, f := range end.Flows {
		total += f.Total
	}
	if got := end.Money - start.Money; got != total {
		t.Errorf("seed %d: the money supply moved by %d, want the grant total %d", seed, got, total)
	}
	for _, b := range end.Boundary {
		if b.Reason == "pool_grant" && b.Balance != -int64(total) {
			t.Errorf("seed %d: boundary balance of the grant %d, want minus its total %d", seed, b.Balance, -int64(total))
		}
	}
	var pool economysim.Money
	for _, r := range end.Regions {
		pool += r.MoneyPool
	}
	t.Logf("seed %d: money per household at ticks 0, 100, 200, %d = %v; household-ticks unfed %d; grant total %d; relief paid %d; pool at the end %d",
		seed, mons3Ticks, series, unfed, total, relief, pool)
}
