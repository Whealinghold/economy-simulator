package economysim

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
)

// The tests in this file pin edges of agent.go that a mutation run (gomu,
// inside Docker) showed the other tests of the file did not reach.

// TestAgent_HasJobEdges verifies that a job of exactly one execution with one
// tick remaining is in progress, and that a negative count of either is not.
//
// Requirement: FR-AGT-053
func TestAgent_HasJobEdges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		job  job
		want bool
	}{
		{"one execution, one tick left", job{executions: 1, remaining: 1}, true},
		{"negative executions", job{executions: -1, remaining: 1}, false},
		{"negative remaining", job{executions: 1, remaining: -1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := agent{job: tt.job}
			if got := a.hasJob(); got != tt.want {
				t.Errorf("hasJob() with %+v = %v, want %v", tt.job, got, tt.want)
			}
		})
	}
}

// TestReseedBeliefs_ScratchFromAnEarlierCallIsIgnored verifies that a seed
// slice whose entries still say Kept from an earlier call does not keep a
// belief at a creation, and that seeds beyond the tradable set are ignored.
//
// Requirement: FR-AGT-004
//
// Guard: src is scratch reused by the world; a stale Kept or an extra entry
// would put another agent's belief into this one.
func TestReseedBeliefs_ScratchFromAnEarlierCallIsIgnored(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	_, row := f.role(t, "farmer")
	a := f.newAgent(t, "farmer", 0, 1)
	p := beliefParamsForTest()
	src := seedsFor(row, 100)
	for i := range src {
		src[i].Kept, src[i].Belief = true, beliefs.Belief{Centre: 999, Width: 1}
	}
	src = append(src, beliefSeed{Ref: 100, Empty: true, Price: 100}, beliefSeed{Ref: 100, Empty: true, Price: 100})
	reseedBeliefs(&a, row, nil, src, p)
	want := []beliefs.Belief{{Centre: 100, Width: 2000}, {Centre: 100, Width: 2000}, {Centre: 100, Width: 2000}}
	if !slices.Equal(a.beliefs, want) {
		t.Errorf("beliefs after a creation with stale scratch = %v, want %v", a.beliefs, want)
	}
}

// TestReseedBeliefs_KeepsALaterCommodityAfterANewOne verifies that a commodity
// new to the agent does not stop the loop: the farmer trades fuel, ore and
// staple, the old role only staple, so fuel and ore are reseeded and the stored
// staple belief (centre 77, width 500) is kept in third place.
//
// Requirement: FR-AGT-004
func TestReseedBeliefs_KeepsALaterCommodityAfterANewOne(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	_, farmer := f.role(t, "farmer")
	staple := CommodityID(f.com(t, "staple"))
	old := *farmer
	old.tradable = []CommodityID{staple}
	old.member = make([]bool, len(farmer.member))
	old.member[staple] = true
	a := f.newAgent(t, "farmer", 0, 1)
	a.beliefs = []beliefs.Belief{{Centre: 77, Width: 500}}
	reseedBeliefs(&a, farmer, &old, seedsFor(farmer, 100), beliefParamsForTest())
	want := []beliefs.Belief{{Centre: 100, Width: 2000}, {Centre: 100, Width: 2000}, {Centre: 77, Width: 500}}
	if !slices.Equal(a.beliefs, want) {
		t.Errorf("beliefs = %v, want %v", a.beliefs, want)
	}
}

// TestReseedBeliefs_BeliefsShorterThanTheOldSetPanic verifies the invariant
// panic when the stored beliefs are two or more short of the old role's set,
// not only one.
//
// Requirement: FR-AGT-004
func TestReseedBeliefs_BeliefsShorterThanTheOldSetPanic(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, false)
	_, farmer := f.role(t, "farmer")
	a := f.newAgent(t, "farmer", 0, 1)
	a.beliefs = nil // the old role has three commodities, so index 2 is two past the end
	mustPanicInvariant(t, "beliefs two short", func() {
		reseedBeliefs(&a, farmer, farmer, seedsFor(farmer, 100), beliefParamsForTest())
	})
}

// TestDefaultEndowment_Edges verifies that a role with no start stock gets an
// empty list of goods, that the money and each item are the weight times the
// start amount, and that an overflow of either product reports not ok.
//
// Requirement: FR-AGT-005
func TestDefaultEndowment_Edges(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, true)
	farmer := f.cfg.Roles[mustRole(t, f.cfg, "farmer")]
	bare := farmer
	bare.StartStock = nil
	money, stock, ok := defaultEndowment(f.cfg, &bare, 3)
	if !ok || len(stock) != 0 || int64(money) != 3*int64(farmer.StartMoney) {
		t.Errorf("defaultEndowment(no stock, 3) = %d, %v, %v, want %d, an empty list, true", money, stock, ok, 3*int64(farmer.StartMoney))
	}
	if _, _, ok := defaultEndowment(f.cfg, &farmer, math.MaxInt64); ok {
		t.Errorf("defaultEndowment(MaxInt64) ok = true, want false for an overflowing money")
	}
	free := farmer
	free.StartMoney = 0
	if _, _, ok := defaultEndowment(f.cfg, &free, math.MaxInt64); ok {
		t.Errorf("defaultEndowment(MaxInt64, no money) ok = true, want false for an overflowing item")
	}
}

// TestFundInitial_OneUnitAndAZeroBeforeIt verifies that an item of exactly one
// unit is funded, and that a zero-quantity item before it does not stop the
// rest.
//
// Requirement: FR-AGT-005
func TestFundInitial_OneUnitAndAZeroBeforeIt(t *testing.T) {
	t.Parallel()
	f := newAgentFixture(t, true)
	acct := f.open(t, "warm")
	ore, staple := CommodityID(f.com(t, "ore")), CommodityID(f.com(t, "staple"))
	if err := fundInitial(f.l, acct, 1, []itemRow{{ore, 0}, {staple, 1}}); err != nil {
		t.Fatalf("fundInitial: %v", err)
	}
	if v, _ := f.l.Account(acct); v.Balance != 1 {
		t.Errorf("balance = %d, want 1", v.Balance)
	}
	if got := f.held(t, acct, "staple"); got != 1 {
		t.Errorf("staple = %d, want 1", got)
	}
	if got := f.held(t, acct, "ore"); got != 0 {
		t.Errorf("ore = %d, want 0", got)
	}
}
