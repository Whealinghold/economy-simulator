package economysim

import "fmt"

// This file holds the fan-ins of the world: compositions with one entry point
// each, a fixed-order sequence of calls into the module files and nothing else
// (architecture.md 2.10.4). The slice has no module, so a fan-in is the core
// call alone, and a module's change appends its call after it.

// fillAgentView fills v for agent a: the core groups by the one view builder
// (strategy_view.go), from the context that refreshViewCtx assembled (market.go)
// at the start of the position that calls it, then the module fillers, none in
// the slice. It is the same builder at every use: P2.5 and P3.2 each refresh the
// context first. A module's filler is appended below the core call.
func (w *World) fillAgentView(v *AgentView, a *agent) error {
	if err := w.views.fillCore(v, a, &w.tk.vctx); err != nil {
		return err
	}
	if w.demand != nil {
		w.fillWants(v, a)
	}
	return nil
}

// checkTradePlan checks the plan a strategy returned for agent a, in place and
// in a fixed order (spec-agents.md 6.5.2): the clamps of each intent's belief
// inputs, then the discards, each refused whole when the agent does not hold
// the units. The conversion of an intent to an order and the tradable set are
// the book's admission and run as the orders are submitted. A module appends its
// checks (shipments, offers and takes, credit and cover requests) below the
// core calls. Nothing here changes the ledger.
func (w *World) checkTradePlan(a *agent, p *TradePlan) {
	w.normalisePlan(a, p)
	w.checkDiscards(a, p)
	if w.demand != nil {
		w.checkWants(a, p)
	}
}

// removeAgent is the one removal path of an agent (spec-agents.md 3.6,
// architecture.md 2.10.4): the job abandoned with agent_removed, the steps of the
// modules (2b to 2g: none in the slice), the money and the lots to the estate of
// the region the account is in (steps 3 and 4, counted in lots), the account
// closed, the events in the order of spec-world.md 6.11 item 8 (RecipeAbandoned,
// then AgentBankrupted, then EstateOpened) and the counters. The cause is the
// caller's and is never recomputed here. The error is the estate's balance
// overflow, which fails the step; every other refusal is a violated invariant.
// A module's step is appended below its slot, and the events stay last.
func (w *World) removeAgent(a *agent, cause Cause) error {
	tick, region := w.Tick(), w.regionOf(a)
	if a.hasJob() {
		w.abandonJob(a, region, ReasonAgentRemoved)
	}
	money, lots, err := w.estates.receive(w.ledger, a.account, region)
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return fmt.Errorf("remove agent %d at tick %d: %w", a.id, tick, err)
	}
	mustClose(w.ledger.Close(a.account))
	if w.obs.supplied {
		w.obs.deliver(AgentBankrupted{Tick: tick, Agent: a.id, Region: region, Role: a.role, Weight: a.weight, Cause: cause, Money: money, GoodsListed: lots})
		w.obs.deliver(EstateOpened{Tick: tick, Agent: a.id, Region: region, Balance: w.estates.pool(w.ledger, region), GoodsListed: lots})
	}
	if w.demand != nil {
		w.demand.dropAgent(a)
	}
	w.logRemoval(a, cause)
	w.forget(a, region)
	return nil
}

// agentJoined is the first agent-lifecycle fan-in (architecture.md 2.10.4, ADR
// 0010): the steps of the modules for an agent that has just been created, called
// from createAgent, spawnEntrant and the integration add after the agent is in
// the world and the live counters are updated. Nothing is drawn.
func (w *World) agentJoined(a *agent, region RegionID) {
	if w.demand != nil {
		w.joinedDemand(a, region)
	}
}

// agentRoleChanged is the second agent-lifecycle fan-in: the steps of the modules
// for an agent whose role has just changed from the given one, called from
// takeRole once the agent holds its new role. Nothing is drawn.
func (w *World) agentRoleChanged(a *agent, from RoleID, region RegionID) {
	if w.demand != nil {
		w.roleChangedDemand(a, from, region)
	}
}

// relocationRefusal is the one entry point of the region_bound refusals that
// keep an agent from being moved to another region (spec-regions.md 6.1.3 row 7,
// 6.5.2): the first obligation that the agent holds and that skip does not name
// is returned. The table is built at construction by the modules that are on, in
// module order, and each module adds its obligation there (regionBounds.add), so
// this stays one call; with no module on the table is empty and nothing refuses.
func (w *World) relocationRefusal(a *agent, skip []string) (*RegionBoundError, bool) {
	return w.bounds.refusal(a, skip)
}

// mustClose panics when the ledger refuses to close the account of a removed
// agent: the intake emptied it, so a refusal is a violated invariant.
func mustClose(err error) {
	if err != nil {
		invariant("the ledger refused the closing of a removed agent's account: " + err.Error())
	}
}

// tradableItems is the fifth fan-in (architecture.md 2.10.4): a fixed-order
// sequence of guarded calls into the module files, each adding its items to the
// tradable set of one role before the role table is frozen (spec-agents.md 3.4,
// FR-DEM-006). It is called from tradableSets (validate.go), the one function the
// role table, the validator and the warnings share, so they cannot disagree. A
// module's change appends its guarded call below the last.
func tradableItems(c *Config, lk lookup, role *Role, set map[string]bool) {
	if c.Modules.Demand != nil {
		demandItems(lk, role, set)
	}
}
