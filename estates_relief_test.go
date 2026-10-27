package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestEstates_PseudoAgentNotCounted verifies that the estate pseudo-agents are in
// no count of agents or households, in no snapshot row and in no agent event,
// with relief on and paying every tick.
//
// Arithmetic: the relief line is at its bound, so every household is below it
// and the pool of 100 000 pays relief from the first tick. The region has one
// estate pseudo-agent (agent 0), so the households are agents 1 to 10 and the
// snapshot counts 10 agents of weight 1, not 11.
//
// Requirement: FR-EST-010
func TestEstates_PseudoAgentNotCounted(t *testing.T) {
	t.Parallel()
	c := minimalConfig()
	c.Modules.Estates = &economysim.Estates{}
	c.Regions[0].ReliefBelow = economysim.MaxBasisPoints
	c.WarmStart = &economysim.WarmStart{Pools: []economysim.WarmPool{{Region: "home", Money: 100_000}}}
	var spawned, removed, households, estateEvents int64
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		switch ev := e.(type) {
		case economysim.AgentSpawned:
			spawned++
			households += ev.Weight
			if int(ev.Agent) < len(c.Regions) {
				estateEvents++
			}
		case economysim.AgentBankrupted:
			removed++
			households -= ev.Weight
			if int(ev.Agent) < len(c.Regions) {
				estateEvents++
			}
		case economysim.WeightChanged:
			households += ev.To - ev.From
			if int(ev.Agent) < len(c.Regions) {
				estateEvents++
			}
		}
	})
	w, err := economysim.NewWorld(c, economysim.WithObserver(obs))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for range 20 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if estateEvents != 0 {
		t.Errorf("agent events about an estate pseudo-agent = %d, want 0", estateEvents)
	}
	if s.Agents != spawned-removed || s.Households != households {
		t.Errorf("Snapshot agents %d and households %d, events give %d and %d, want them equal: no estate is counted",
			s.Agents, s.Households, spawned-removed, households)
	}
	var rowAgents, rowHouseholds int64
	for _, r := range s.Roles {
		rowAgents, rowHouseholds = rowAgents+r.Agents, rowHouseholds+r.Households
	}
	if rowAgents != s.Agents || rowHouseholds != s.Households {
		t.Errorf("role rows count %d agents and %d households, want the snapshot's %d and %d", rowAgents, rowHouseholds, s.Agents, s.Households)
	}
}

// reliefRun steps a minimal world of 10 households with a pool of 100 000, the
// relief line at its bound and the given module and release, for 20 ticks, and
// returns the ReliefPaid events in emission order with the tick of each
// TickStarted seen, so the order against the tick can be checked.
func reliefRun(t *testing.T, on bool, release int64) []economysim.ReliefPaid {
	t.Helper()
	c := minimalConfig()
	if on {
		c.Modules.Estates = &economysim.Estates{}
	}
	c.Regions[0].PoolRelease = release
	c.Regions[0].ReliefBelow = economysim.MaxBasisPoints
	c.WarmStart = &economysim.WarmStart{Pools: []economysim.WarmPool{{Region: "home", Money: 100_000}}}
	var got []economysim.ReliefPaid
	w, err := economysim.NewWorld(c, economysim.WithObserver(economysim.ObserverFunc(func(e economysim.Event) {
		if r, ok := e.(economysim.ReliefPaid); ok {
			got = append(got, r)
		}
	})))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for range 20 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	return got
}

// TestReliefPaid_Events verifies the event of the pool policy: with the module on
// and a pool above the line, ReliefPaid arrives once per region and tick in
// ascending tick order, with the recipients and households counted and a total
// within the release; with the module off, or with PoolRelease 0, none arrives.
//
// Arithmetic: the pool 100 000 with PoolRelease 100 releases at most 1 000 in the
// first tick; the 10 households of weight 1 are all below the line, so the
// recipients are at most 10 and the households equal the recipients.
//
// Requirement: FR-EST-020, FR-OBS-008, FR-CFG-099
func TestReliefPaid_Events(t *testing.T) {
	t.Parallel()
	got := reliefRun(t, true, 100)
	if len(got) == 0 {
		t.Fatal("ReliefPaid events = none, want relief paid with a pool and households below the line")
	}
	if first := got[0]; first.Tick != 0 || first.Region != 0 || first.Paid < 1 || first.Paid > 1_000 ||
		first.Recipients < 1 || first.Recipients > 10 || first.Households != first.Recipients {
		t.Errorf("first ReliefPaid = %+v, want tick 0, region 0, Paid in 1..1000, Recipients in 1..10 and Households equal to Recipients (weights of 1)", first)
	}
	if !slices.IsSortedFunc(got, func(a, b economysim.ReliefPaid) int { return int(a.Tick - b.Tick) }) {
		t.Errorf("ReliefPaid events are not in tick order: %+v", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i].Tick == got[i-1].Tick {
			t.Errorf("two ReliefPaid in tick %d for one region: %+v and %+v", got[i].Tick, got[i-1], got[i])
		}
	}
	for _, c := range []struct {
		name    string
		on      bool
		release int64
	}{{"module off", false, 100}, {"release of zero", true, 0}} {
		if ev := reliefRun(t, c.on, c.release); len(ev) != 0 {
			t.Errorf("%s: %d ReliefPaid events, want none: %+v", c.name, len(ev), ev[0])
		}
	}
}
