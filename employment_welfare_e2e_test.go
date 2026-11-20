package economysim_test

import (
	"math/rand/v2"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// labourScript is the strategy of the small hire world below: a farm posts one position
// of the recipe at a bid of 100 and a hand offers one household at a reserve of 0, and
// neither works for itself or places an order, so the only money that moves is the
// module's. It is written through the public API and keeps no state.
type labourScript struct{ farm economysim.RoleID }

func (s labourScript) Work(v *economysim.AgentView, _ *rand.Rand) economysim.JobRequest {
	if v.Role == s.farm {
		return economysim.JobRequest{NoJob: true, Hire: economysim.HireRequest{Recipe: 0, Positions: 1, Bid: 100}}
	}
	return economysim.JobRequest{NoJob: true, Offer: economysim.LabourOffer{Households: 1}}
}

func (labourScript) Trade(*economysim.AgentView, *rand.Rand, *economysim.TradePlan) {}

func (labourScript) Revise(o *economysim.OutcomeView, _ *rand.Rand) economysim.Belief {
	return o.Belief
}

var _ economysim.Strategy = labourScript{}

// labourWorld builds the smallest world of the hire form through the public API: one
// region, a raw commodity "grain", the employer role "farm" (recipe "tend" of 3 ticks,
// two grain an execution, the hire list), and the Hireable role "hand" with no recipe,
// one agent of weight 2, so a hired household is half of the agent. There is no upkeep and
// no order, and the farm holds 10 000.
func labourWorld(t *testing.T, tape *hireTape) *economysim.World {
	t.Helper()
	world := economysim.DefaultWorldParams()
	world.OutputCoverTicks, world.Upkeep, world.MinAgents = 0, nil, 1
	grain := economysim.DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 1_000, true
	farm, hand := economysim.DefaultRole(), economysim.DefaultRole()
	farm.ID, farm.Recipes, farm.StartMoney, farm.Strategy = "farm", []string{"tend"}, 10_000, "script"
	farm.Employer = &economysim.Employer{MaxHires: 4, WorkerShareBP: 5_000, HireYieldBP: 10_000, Recipes: []string{"tend"}}
	hand.ID, hand.StartMoney, hand.Hireable, hand.Strategy = "hand", 0, true, "script"
	region := economysim.DefaultRegion()
	region.ID, region.ReferenceBasket = "r0", 0
	cfg := economysim.Config{
		World: world, Calendar: economysim.DefaultCalendar(),
		Commodities: []economysim.Commodity{grain},
		Roles:       []economysim.Role{farm, hand},
		Recipes: []economysim.Recipe{{ID: "tend", Duration: 3, TraineeYield: 5_000,
			Outputs: []economysim.Output{{Commodity: "grain", Quantity: 2, CostShare: 10_000}}}},
		Regions: []economysim.Region{region},
		Groups: []economysim.Group{{Region: "r0", Role: "farm", Households: 1},
			{Region: "r0", Role: "hand", Households: 2, Scale: 2}},
	}
	m := economysim.DefaultEmployment()
	m.Hire = true
	cfg.Modules.Employment = &m
	opts := []economysim.Option{economysim.WithSeed(1), economysim.WithAudit(), economysim.WithStrategy("script", labourScript{farm: 0})}
	if tape != nil {
		opts = append(opts, economysim.WithObserver(tape))
	}
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld(small hire world) = %v, want nil", err)
	}
	return w
}

// TestHire_MultiTickLifeAndObservedWageThroughThePublicAPI verifies, on a world built in
// Go, the whole life of a three-tick hire and what it does to the worker's welfare record
// (spec-employment.md 6, 8.1, 8.3): one household of a hand agent of weight 2 is hired at
// a wage of 0 + 100 * 5 000 / 10 000 = 50 for 3 ticks, the escrow of 150 is paid 50 a tick
// from P2.7 and held in the contract sector in between (100 after the first step, 50
// after the second), the hire ends Completed with 3 working ticks and 150 paid, and the
// observed wage of the role is 50 a working household-tick: the hired tick is k = 1
// household-tick, not the agent's weight of 2 (which would read 25) and not one tick of
// the agent.
//
// Requirement: FR-EMP-008, FR-EMP-024, FR-EMP-029, FR-EMP-036, FR-EMP-045, FR-EMP-046, FR-EMP-047
func TestHire_MultiTickLifeAndObservedWageThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	var tape hireTape
	w := labourWorld(t, &tape)
	escrow := func() economysim.Money {
		sn, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		return sn.Regions[0].MoneyContract
	}
	for i, want := range []economysim.Money{100, 50} {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() %d = %v, want nil", i, err)
		}
		if got := escrow(); got != want {
			t.Errorf("MoneyContract after step %d = %d, want %d (150 escrowed less 50 a paid tick)", i, got, want)
		}
	}
	for range 4 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	var formed []economysim.HireFormed
	var ended []economysim.HireEnded
	for _, e := range tape.events {
		switch e := e.(type) {
		case economysim.HireFormed:
			formed = append(formed, e)
		case economysim.HireEnded:
			ended = append(ended, e)
		}
	}
	if len(formed) < 2 || len(ended) < 1 {
		t.Fatalf("hires formed %d and ended %d in 6 ticks, want the first to complete and a second to form", len(formed), len(ended))
	}
	if f := formed[0]; f.Tick != 0 || f.Wage != 50 || f.Households != 1 || f.Ticks != 3 || f.Escrowed != 150 || f.Reserve != 0 || f.Bid != 100 {
		t.Errorf("first HireFormed = %+v, want tick 0, wage 50, 1 household, 3 ticks, escrowed 150, reserve 0, bid 100", f)
	}
	if e := ended[0]; e.Tick != 2 || e.Cause != economysim.CauseCompleted || e.Worked != 3 || e.Paid != 150 || e.Refunded != 0 {
		t.Errorf("first HireEnded = %+v, want tick 2, Completed, 3 worked, 150 paid, 0 refunded", e)
	}
	sn, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	var rows int
	for _, r := range sn.Roles {
		if r.Role != 1 { // the hand
			continue
		}
		rows++
		if !r.HasWage || r.WageObserved != 50 {
			t.Errorf("hand RoleRow (HasWage, WageObserved) = (%v, %d), want (true, 50): a hired tick is k = 1 household-tick", r.HasWage, r.WageObserved)
		}
	}
	if rows != 1 {
		t.Errorf("Snapshot().Roles has %d rows of the hand, want 1", rows)
	}
}
