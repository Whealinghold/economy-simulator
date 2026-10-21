package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// flowKey is the key of a boundary account in the test: a reason and a thing.
type flowKey struct {
	reason    economysim.Reason
	commodity economysim.CommodityID
}

// ledgerView is what the events of a run rebuild: the agents and households per
// (region, role), the signed boundary balance per account, and the last
// MarketCleared of each market.
type ledgerView struct {
	nr, nc      int
	agents, hh  []int64
	balance     map[flowKey]int64
	lastCleared map[[2]int]economysim.MarketCleared
	changes     int // spawns, removals and changes seen after construction
}

func newLedgerView(nr, nregions, nc int) *ledgerView {
	return &ledgerView{
		nr: nr, nc: nc, agents: make([]int64, nr*nregions), hh: make([]int64, nr*nregions),
		balance: map[flowKey]int64{}, lastCleared: map[[2]int]economysim.MarketCleared{},
	}
}

// Observe applies one event: the spawns, removals, weight and role changes to the
// counts; every BoundaryFlow to the boundary balances (a faucet is negative, a
// sink positive, as the account is); each MarketCleared to the last-report table.
func (v *ledgerView) Observe(e economysim.Event) {
	switch e := e.(type) {
	case economysim.AgentSpawned:
		i := int(e.Region)*v.nr + int(e.Role)
		if e.Tick > 0 {
			v.changes++
		}
		v.agents[i]++
		v.hh[i] += e.Weight
	case economysim.AgentBankrupted:
		v.changes++
		i := int(e.Region)*v.nr + int(e.Role)
		v.agents[i]--
		v.hh[i] -= e.Weight
	case economysim.WeightChanged:
		v.hh[int(e.Region)*v.nr+int(e.Role)] += e.To - e.From
	case economysim.RoleChanged:
		from, to := int(e.Region)*v.nr+int(e.From), int(e.Region)*v.nr+int(e.To)
		v.agents[from]--
		v.agents[to]++
		v.hh[from] -= e.Weight
		v.hh[to] += e.Weight
	case economysim.BoundaryFlow:
		sign := int64(-1)
		if e.Kind == economysim.KindSink {
			sign = 1
		}
		v.balance[flowKey{e.Reason, e.Commodity}] += sign * e.Total
	case economysim.MarketCleared:
		v.lastCleared[[2]int{int(e.Region), int(e.Commodity)}] = e
	}
}

// TestSnapshot_RebuiltFromEvents verifies, over a range of seeds and the packs
// one, second and two, at every tick boundary: the agents and households by
// role and region rebuilt from AgentSpawned, AgentBankrupted, WeightChanged and
// RoleChanged equal the snapshot (FR-OBS-040); the money of the households and
// the pool equals the net of the money faucets and sinks of the events plus the
// construction bucket plus Pending (FR-OBS-041; escrow is 0 in Stage 1); the
// boundary balance of each reason equals Construction plus the BoundaryFlow
// rows plus Pending (FR-OBS-042); each MarketRow.State equals the last
// MarketCleared (FR-OBS-045 half).
//
// Arithmetic: a faucet of f units leaves its boundary account at -f and a sink of
// k units at +k, so the net money that entered is the negative of the sum of the
// money boundary balances; before the first step the bucket is Construction,
// after it the same totals are the BoundaryFlow events of tick -1.
//
// Requirement: FR-OBS-040, FR-OBS-041, FR-OBS-042, FR-OBS-045, FR-EST-016
func TestSnapshot_RebuiltFromEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	packs := []struct {
		name string
		cfg  func(*testing.T) economysim.Config
	}{
		{"one", func(t *testing.T) economysim.Config { return loadPack(t, "base.toml", "one.toml") }},
		{"second", func(t *testing.T) economysim.Config { return loadPack(t, "second/second.toml") }},
		{"two", func(t *testing.T) economysim.Config { return loadPack(t, "base.toml", "two.toml") }},
		{"famine", famine},
	}
	for _, pk := range packs {
		for seed := uint64(1); seed <= 3; seed++ {
			t.Run(fmt.Sprintf("%s/seed%d", pk.name, seed), func(t *testing.T) {
				t.Parallel()
				cfg := pk.cfg(t)
				c := economysim.Canonical(cfg)
				v := newLedgerView(len(c.Roles), len(c.Regions), len(c.Commodities))
				w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(v))
				if err != nil {
					t.Fatalf("NewWorld error = %v, want nil", err)
				}
				for tick := 0; tick <= 12; tick++ {
					checkBoundary(t, fmt.Sprintf("%s seed %d tick %d", pk.name, seed, tick), snapshotOf(t, w), v)
					if tick < 12 {
						if err := w.Step(); err != nil {
							t.Fatalf("seed %d: Step at tick %d error = %v, want nil", seed, tick, err)
						}
					}
				}
				if pk.name == "second" && v.changes == 0 {
					t.Errorf("second seed %d: no population change in 12 ticks, so the rebuild was not exercised", seed)
				}
			})
		}
	}
}

// checkBoundary compares one snapshot with what the events rebuilt.
func checkBoundary(t *testing.T, at string, s economysim.Snapshot, v *ledgerView) {
	t.Helper()
	for i, r := range s.Roles {
		if r.Agents != v.agents[i] || r.Households != v.hh[i] {
			t.Errorf("%s: role row %d (region %d, role %d) = %d agents, %d households, events give %d and %d", at, i, r.Region, r.Role, r.Agents, r.Households, v.agents[i], v.hh[i])
		}
	}
	extra := map[flowKey]int64{}
	for _, f := range append(append([]economysim.FlowTotal(nil), s.Construction...), s.Pending...) {
		sign := int64(-1)
		if f.Kind == economysim.KindSink {
			sign = 1
		}
		extra[flowKey{f.Reason, f.Commodity}] += sign * f.Total
	}
	var moneyIn int64
	for _, b := range s.Boundary {
		k := flowKey{b.Reason, b.Commodity}
		want := v.balance[k] + extra[k]
		if b.Balance != want {
			t.Errorf("%s: boundary %s/%d balance = %d, want %d from the events, Construction and Pending", at, b.Reason, b.Commodity, b.Balance, want)
		}
		if b.Commodity == economysim.NoCommodity {
			moneyIn -= want
		}
	}
	var held int64
	for _, r := range s.Regions {
		held += int64(r.MoneyHouseholds) + int64(r.MoneyPool)
	}
	if held != int64(s.Money) || held != moneyIn {
		t.Errorf("%s: money households + pool = %d, header %d, net faucets less sinks %d, want all equal", at, held, int64(s.Money), moneyIn)
	}
	for _, m := range s.Markets {
		e, ok := v.lastCleared[[2]int{int(m.State.Region), int(m.State.Commodity)}]
		if !ok {
			continue
		}
		got := m.State
		if got.Price != e.Price || got.Age != e.PriceAge || got.Volume != e.Volume || got.UnmatchedBid != e.UnmatchedBid ||
			got.UnmatchedAsk != e.UnmatchedAsk || got.BestBid != e.BestBid || got.BestAsk != e.BestAsk ||
			got.HasBid != e.HasBestBid || got.HasAsk != e.HasBestAsk {
			t.Errorf("%s: market (%d, %d) state = %+v, last MarketCleared = %+v", at, got.Region, got.Commodity, got, e)
		}
	}
}

// TestSnapshot_JobsPendingAndSignals verifies, over the warm packs, at the
// boundaries of the first ticks, that the Jobs rows and the Markets[].Pending
// equal a hand count over the agents with a job in progress (FR-PRO-042), that
// RoleRow.ExpectedRatioBP, Viable, HasExpected, MaxPrivation and MaxHunger equal
// a second reading of the saved state, and that HeldByFloor is evaluated over
// the agents with no job in progress (FR-WEL-020, FR-SPN-046; the counts of
// EntryLimited are the event, not a snapshot field).
//
// Arithmetic: a job of e executions of a recipe with output quantities q_i and
// yield y has pending_i = e * q_i * y / 10 000 in the job's own record; the
// pending output of a (region, commodity) is the sum over the jobs of the region
// of the entries that name the commodity, and the job count of a recipe is the
// number of agents that hold such a job.
//
// Requirement: FR-PRO-042, FR-WEL-020, FR-SPN-046, FR-OBS-043
func TestSnapshot_JobsPendingAndSignals(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	for _, pk := range warmPacks {
		t.Run(pk.name, func(t *testing.T) {
			t.Parallel()
			w, cfg := buildWarm(t, pk.layers, economysim.WithSeed(5))
			sawJobs := false
			for tick := 0; tick < 4; tick++ {
				s := snapshotOf(t, w)
				sawJobs = checkJobs(t, w, &cfg, s) || sawJobs
				checkSignals(t, w, s)
				if err := w.Step(); err != nil {
					t.Fatalf("%s: Step error = %v, want nil", pk.name, err)
				}
			}
			if !sawJobs {
				t.Errorf("%s: no job in progress at any of 4 boundaries, so the job rows were not exercised", pk.name)
			}
		})
	}
}

// checkJobs compares the Jobs rows and the Pending of the markets with a hand
// count over the agents, and says whether there was any job.
func checkJobs(t *testing.T, w *economysim.World, cfg *economysim.Config, s economysim.Snapshot) bool {
	t.Helper()
	nr, nc := len(cfg.Recipes), len(cfg.Commodities)
	agents, execs, hh := make([]int64, len(cfg.Regions)*nr), make([]int64, len(cfg.Regions)*nr), make([]int64, len(cfg.Regions)*nr)
	pending := make([]int64, len(cfg.Regions)*nc)
	any := false
	for _, a := range economysim.ConstructionRows(w) {
		j := economysim.AgentJob(w, a.ID)
		if a.Weight < 1 || !j.Has {
			continue
		}
		any = true
		i := int(a.Region)*nr + int(j.Recipe)
		agents[i]++
		execs[i] += j.Executions
		hh[i] += a.Weight
		for k, o := range cfg.Recipes[j.Recipe].Outputs {
			pending[int(a.Region)*nc+commodityIndex(t, cfg, o.Commodity)] += j.Pending[k]
		}
	}
	var rows []economysim.JobRow
	for i := range agents {
		if agents[i] > 0 {
			rows = append(rows, economysim.JobRow{Region: economysim.RegionID(i / nr), Recipe: economysim.RecipeID(i % nr), Agents: agents[i], Executions: execs[i], Households: hh[i]})
		}
	}
	if !reflect.DeepEqual(append([]economysim.JobRow(nil), s.Jobs...), rows) {
		t.Errorf("tick %d: Jobs = %+v, hand count = %+v", s.Tick, s.Jobs, rows)
	}
	for i, m := range s.Markets {
		if int64(m.Pending) != pending[i] {
			t.Errorf("tick %d: market %d pending = %d, hand count = %d", s.Tick, i, m.Pending, pending[i])
		}
	}
	return any
}

// checkSignals compares the signals of each role row with RoleSignalsOf.
func checkSignals(t *testing.T, w *economysim.World, s economysim.Snapshot) {
	t.Helper()
	for _, r := range s.Roles {
		g := economysim.RoleSignalsOf(w, r.Region, r.Role)
		if r.ExpectedRatioBP != g.ExpectedRatioBP || r.Viable != g.Viable || r.HasExpected != g.HasExpected ||
			r.MaxPrivation != g.MaxPrivation || r.MaxHunger != g.MaxHunger || r.HeldByFloor != g.HeldByFloor || r.Households != g.Households {
			t.Errorf("tick %d: role row (%d, %d) = %+v, second reading = %+v", s.Tick, r.Region, r.Role, r, g)
		}
	}
}
