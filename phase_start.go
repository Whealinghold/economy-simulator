package economysim

import (
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is the entry of a step and phase 1 (spec-world.md 6.2, 6.3;
// plan-phase-1-11.md Phase 4): the end of the construction phase and the
// construction bucket's events (P0.2a, run inside beginStep), the moment
// (P1.1) and the cost anchors (P1.2). Each position is one unexported method,
// called by one line of tick.go or by beginStep (D166). The positions of phase 1
// that belong to modules are not built, so they are not here.

// tickScratch is what the positions of the entry and of phase 1 write and the
// later positions read: the moment of the tick, the view context, the one view
// that Work is called with and the buffers of the anchors and of the bucket
// events. It is derived or scratch, allocated at construction and never saved
// (D154).
type tickScratch struct {
	moment Moment
	vctx   viewCtx
	view   AgentView
	// prices and wages are the inputs of the cost anchors: the published price of
	// every market, region-major, and the wage of every region. rings holds, per
	// region, the welfare rings of its agents.
	prices, wages []int64
	rings         [][]*welfare.Ring
	// flows, boundary and wear are the buffers of the events of the bucket.
	flows    []ledger.Flow
	boundary []BoundaryFlow
	wear     []WearApplied
}

// newTickScratch sizes the scratch for a canonical configuration.
func newTickScratch(c *Config) tickScratch {
	return tickScratch{
		prices: make([]int64, len(c.Regions)*len(c.Commodities)),
		wages:  make([]int64, len(c.Regions)),
		rings:  make([][]*welfare.Ring, len(c.Regions)),
	}
}

// regionOf returns the region of an agent's ledger account. The region lives on
// the account (spec-agents.md FR-AGT-002), so it is never stored on the agent.
// An unreadable account of a live agent is a violated invariant.
func (w *World) regionOf(a *agent) RegionID {
	v, err := w.ledger.Account(a.account)
	if err != nil {
		invariant("the account of an agent is unreadable: " + err.Error())
	}
	return RegionID(v.Region)
}

// p02aBucket is P0.2a, run inside beginStep at the entry of the first step of a
// world still in its construction phase (spec-world.md 6.2, D160): it ends the
// construction phase and then, only when an observer was supplied, delivers the
// construction bucket as BoundaryFlow events and then WearApplied events with
// Tick -1, once, before TickStarted. The bucket is read, not cleared, and no
// random number is drawn. A step after the first finds the phase over and does
// nothing.
func (w *World) p02aBucket() {
	if !w.ledger.InConstruction() {
		return
	}
	if err := w.ledger.EndConstruction(); err != nil {
		invariant("the construction phase could not be ended: " + err.Error())
	}
	if !w.obs.supplied {
		return
	}
	t := &w.tk
	t.flows = w.ledger.AppendFlows(t.flows[:0], true)
	t.boundary = boundaryFlows(t.boundary[:0], t.flows, constructionTick)
	for _, e := range t.boundary {
		w.obs.deliver(e)
	}
	t.wear = wearApplied(t.wear[:0], w.ledger, len(w.cfg.Commodities), constructionTick, true)
	for _, e := range t.wear {
		w.obs.deliver(e)
	}
}

// p11Moment is P1.1: the moment of the tick, once, from the calendar. It is
// derived and adds nothing to a checkpoint. The error is the calendar's, a tick
// whose day number does not fit int64.
func (w *World) p11Moment() error {
	m, err := w.cfg.Calendar.At(w.Tick())
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return err
	}
	w.tk.moment = m
	return nil
}

// p12Anchors is P1.2 (spec-production.md 9.3, D157): the cost anchor of every
// (region, commodity), computed once before the first call of Work from the
// published price left by P4.1 of the previous tick (the seed price at tick 0,
// not the price window) and the pooled wage of each region as it stood at the end
// of the previous tick, which falls back to the starting wage of the warm start
// and is 0 while both are unknown. The recipes that can run in a region are those
// some agent there has on its role's list. No random number is drawn.
func (w *World) p12Anchors() {
	t := &w.tk
	for i := range w.markets.reports {
		t.prices[i] = int64(w.markets.reports[i].Price)
	}
	for g := range t.rings {
		t.rings[g] = t.rings[g][:0]
	}
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 {
			g := w.regionOf(a)
			t.rings[g] = append(t.rings[g], &a.history)
		}
	}
	for g := range t.wages {
		t.wages[g], _ = w.reads.regionWage(t.rings[g], w.seedWage[g], w.cfg.World.AnchorHold)
	}
	w.setRunnable()
	w.prod.anchors(t.prices, t.wages, w.runnable, &w.yields, &w.shocks, t.moment.Season)
}

// setRunnable lists, per region, the recipes of every role that has an agent
// there, in role order. A recipe may appear twice; the anchors only mark it.
func (w *World) setRunnable() {
	roles := len(w.cfg.Roles)
	for g := range w.runnable {
		w.runnable[g] = w.runnable[g][:0]
		for r := range roles {
			if w.live.inRole[g*roles+r] > 0 {
				w.runnable[g] = append(w.runnable[g], w.roles.row(RoleID(r)).recipes...)
			}
		}
	}
}
