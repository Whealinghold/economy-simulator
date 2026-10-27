package economysim_test

import (
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
