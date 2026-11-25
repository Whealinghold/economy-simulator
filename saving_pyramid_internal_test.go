package economysim

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

// upkeepBid is the quantity the plan bids for upkeep goods (tier 1, PurposeUpkeep).
func upkeepBid(p *TradePlan) int64 {
	var q int64
	for _, in := range p.Intents {
		if in.Side == Bid && in.Purpose == PurposeUpkeep {
			q += int64(in.Quantity)
		}
	}
	return q
}

// TestTarget_OnlyTheCapReadsIt verifies FR-SAV-003 at the plan: the goal enters the cap on
// discretionary spending and nothing else. At a money of 100 (above the reserve of 90, far
// below S* = 690) the household's tier 1 bid, the PurposeUpkeep quantity, is the same with the
// group and without it, and the survival rule reads the flat reserve, so the household is not
// barred. (At larger monies the cap trims the upkeep stock-up beyond tier 1, which is its job.)
//
// Requirement: FR-SAV-003, FR-SAV-005
func TestTarget_OnlyTheCapReadsIt(t *testing.T) {
	t.Parallel()
	for _, money := range []int64{100} {
		saver, plain := millView(money), millView(money)
		plain.Saving = nil
		ps, _ := runTrade(saver, nil)
		pp, _ := runTrade(plain, nil)
		if got, want := upkeepBid(&ps), upkeepBid(&pp); got != want || want < 1 {
			t.Errorf("Trade(money %d) tier 1 upkeep bid = %d with the group, want %d without it (at least 1)", money, got, want)
		}
	}
	if got := SavingTarget(millView(100), 0); got.Bar == BarSurvival {
		t.Errorf("SavingTarget(money 100) bar = %q, want the survival bar read against the flat reserve 90 (not S*)", got.Bar)
	}
}

// TestReach_NZeroOnlyNeedZero verifies FR-SAV-045 for an empty ring: with no record only a
// household that needs nothing more than it holds reaches its project (a funded one,
// money 690 = floor 90 + outlay 600), and one that lacks money has the Unreachable bar.
//
// Requirement: FR-SAV-045
func TestReach_NZeroOnlyNeedZero(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		money int64
		goal  int64
		bar   SavingBar
	}{
		{"funded needs nothing", 690, 600, ""},
		{"short of the outlay", 100, 0, BarUnreachable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := millView(tt.money)
			v.Saving.Records, v.Saving.Surplus = 0, 0
			if got := SavingTarget(v, 0); got.Goal != tt.goal || got.Bar != tt.bar {
				t.Errorf("SavingTarget(money %d, empty ring) goal, bar = %d, %q, want %d, %q", tt.money, got.Goal, got.Bar, tt.goal, tt.bar)
			}
		})
	}
}

// TestReach_UnreachableBuysBuffers verifies that an unreachable project sets no goal and the
// target is the floor alone, which is the greater of the reserve and the precaution buffer:
// the household still builds the buffer a deep drawdown asks for.
//
// Guard: a regression that kept the goal of an unreachable project would save for a thing
// the household can never fund, locking its money for no return.
func TestReach_UnreachableBuysBuffers(t *testing.T) {
	t.Parallel()
	v := millView(100)
	v.Saving.Surplus, v.Saving.Drawdown = 60, 5_000
	got := SavingTarget(v, 0)
	if got.Bar != BarUnreachable || got.Goal != 0 || got.Target != got.Floor {
		t.Fatalf("SavingTarget(unreachable) = %+v, want bar unreachable, no goal and S* = floor", got)
	}
	if want := max(int64(90), got.Precaution); got.Floor != want {
		t.Errorf("SavingTarget(unreachable) floor = %d, want max(reserve 90, precaution %d) = %d", got.Floor, got.Precaution, want)
	}
}

// TestView_SavingFacilitiesMatchTheWorld verifies that the facilities the Saving group lists
// are the agent's own at the time of the call: after each tick of a run in which the miller
// builds its mills, the Trade view of the same tick lists as many facilities as the world
// holds for it, so a facility completed this tick is seen at once and no project is weighed
// against a stale count (FR-SAV-046).
//
// Requirement: FR-SAV-046, FR-SAV-028
func TestView_SavingFacilitiesMatchTheWorld(t *testing.T) {
	t.Parallel()
	w, m := richMiller(t, nil)
	built := 0
	for i := range 14 {
		facStep(t, w)
		v := savView(t, w, m, viewTrade)
		got, want := len(v.Saving.Facilities), ownedMills(w, m)
		if got != want {
			t.Errorf("tick %d: Saving.Facilities = %d, world holds %d mills for the miller (seed %d)", i, got, want, facSeed)
		}
		built = max(built, want)
	}
	if built == 0 {
		t.Errorf("mills built over 14 ticks = 0, want at least 1: the test would pass with nothing built")
	}
}

// TestSaving_RepairsItsWornMill verifies example 16.2 end to end through the default alone:
// a saver miller that holds a worn mill (condition 100 of 1 000, below the gate of 5 000
// basis points) asks for the repair and the mill's condition rises above where it was, while
// the same world with no block leaves it to decay (FR-CAP-018: the default repairs nothing
// for a role with no block).
//
// Requirement: FR-SAV-022, FR-SAV-041
func TestSaving_RepairsItsWornMill(t *testing.T) {
	t.Parallel()
	condition := func(w *World) int64 {
		var c int64
		for _, f := range w.infra.fac {
			if f.class == clsMill {
				c += f.condition
			}
		}
		return c
	}
	run := func(mut func(c *Config)) int64 {
		w, m := richMiller(t, mut)
		bldAdd(t, w, clsMill, m, 1, 100)
		for range 6 {
			facStep(t, w)
		}
		return condition(w)
	}
	on := run(nil)
	off := run(func(c *Config) { c.Modules.Saving, c.Roles[0].Saver = nil, nil })
	if on <= off {
		t.Errorf("total mill condition after 6 ticks = %d with the block, %d without, want more with it: the saver repairs (seed %d)", on, off, facSeed)
	}
}

// savingSeeds is the number of seeds of the multi-seed tests: a run is 14 ticks of one household
// group, so all of them run under -race too.
const savingSeeds = 4

// spendSpy is the default strategy that records a violation of the spend bound: a plan whose
// bids reserve more than the household's money.
type spendSpy struct {
	DefaultStrategy
	over *[]string
}

func (s spendSpy) Trade(v *AgentView, r *rand.Rand, p *TradePlan) {
	s.DefaultStrategy.Trade(v, r, p)
	if got := spendOf(p); got > v.Money {
		*s.over = append(*s.over, "bids reserve more than the money")
	}
}

// TestInvariants_SavingSpendWithinMoneyOverSeeds verifies FR-SAV-005 end to end over seeds: a
// saver run for 14 ticks, whatever the seed, never has a plan whose bids reserve more than its
// money at the call, so the target never raises a spend bound.
//
// Requirement: FR-SAV-005
func TestInvariants_SavingSpendWithinMoneyOverSeeds(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= savingSeeds; seed++ {
		var over []string
		w, _ := richMiller(t, func(c *Config) { c.Roles[0].Strategy = "spy" }, WithSeed(seed), WithStrategy("spy", spendSpy{over: &over}))
		for range 14 {
			facStep(t, w)
		}
		if len(over) != 0 {
			t.Errorf("seed %d: %d plans broke the spend bound: %v", seed, len(over), over[0])
		}
	}
}

// TestDeterminism_SavingTwoRunsOverSeeds verifies NFR-SAV-001 end to end: two worlds of the
// same seed and configuration with a saver role give equal snapshots after 14 ticks, for each
// seed, so the module's choices depend on nothing but the seed.
//
// Requirement: NFR-SAV-001, FR-SAV-002
func TestDeterminism_SavingTwoRunsOverSeeds(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= savingSeeds; seed++ {
		a, _ := richMiller(t, nil, WithSeed(seed))
		b, _ := richMiller(t, nil, WithSeed(seed))
		for range 14 {
			facStep(t, a)
			facStep(t, b)
		}
		sa, errA := a.Snapshot()
		sb, errB := b.Snapshot()
		if errA != nil || errB != nil {
			t.Fatalf("seed %d: Snapshot() errors = %v, %v, want nil", seed, errA, errB)
		}
		if !reflect.DeepEqual(sa, sb) {
			t.Errorf("seed %d: two runs give different snapshots after 14 ticks", seed)
		}
	}
}
