package economysim

import fcy "github.com/Whealinghold/economy-sim/internal/facility"

// This file is the effect tables of the infrastructure module (spec-infrastructure.md
// 6.4): the derived structure that every position of a tick reads, rebuilt in one pass
// over the records, and the step of the facility position P8.3b (6.3), which decays,
// ruins, installs, rebuilds the tables and clamps the stock of a ground endowment.

// rebuild recomputes the effect tables in full from the records (6.4.2, 6.4.3): the
// capacity of the sites by (owner, region, class), the gain of the routes by directed
// pair and mode, and the gain of the ground by endowment. It allocates nothing beyond
// the tables sized at construction. The route and ground gains are capped sums, never
// products (5.2.2, 5.3.2); which pair a route gain applies to (5.2.5) is the reader's
// rule, so the table holds the gain of every directed edge the facility affects.
func (n *infrastructure) rebuild() {
	clear(n.site)
	clear(n.route)
	clear(n.ground)
	for i := range n.fac {
		f := &n.fac[i]
		switch c := &n.cls[f.class]; c.kind {
		case kindSite:
			e := n.eff(f)
			n.site[siteKey{f.owner, f.region, f.class}] += e
			f.tabCond, f.tabCap, f.served = f.condition, e, 0
		case kindRoute:
			n.addRoute(f, c)
		case kindGround:
			g := int(f.region)*n.resources + c.resource
			n.ground[g] = min(n.ground[g]+n.eff(f), MaxTotalBoostBP)
		}
	}
}

// addRoute adds the gain of a route facility to the directed edges it affects
// (spec-infrastructure.md 5.2.1): with a target, the edge from the region to it and the
// edge back, each only where the configuration has it; without one, every edge that
// leaves the region.
func (n *infrastructure) addRoute(f *facility, c *facClass) {
	e := n.eff(f)
	if f.to != NoRegion {
		n.addEdge(f.region, f.to, c, e)
		n.addEdge(f.to, f.region, c, e)
		return
	}
	for to := range n.regions {
		n.addEdge(f.region, RegionID(to), c, e)
	}
}

// addEdge adds e to the gain of the directed edge (from, to) for each mode of the class,
// when the configuration has the edge.
func (n *infrastructure) addEdge(from, to RegionID, c *facClass, e int64) {
	if !n.edge[[2]RegionID{from, to}] {
		return
	}
	base := (int(from)*n.regions + int(to)) * n.modes
	for _, m := range c.modes {
		n.route[base+m] = min(n.route[base+m]+e, MaxTotalBoostBP)
	}
}

// p83bFacilities is P8.3b (spec-infrastructure.md 6.3.2), after the regeneration of every
// endowment and before the audit: decay and ruin, install, rebuild and clamp, report, in
// that order and in ascending facility ID. It draws nothing and keeps no state of its own:
// the tables are derived, so a checkpoint, which is taken after the step, never holds a
// half-done one.
func (w *World) p83bFacilities() {
	n := w.infra
	n.decayAndRuin()
	n.installPending()
	n.settle()
	n.reportLimits()
}

// decayAndRuin is steps 1 and 2 of 6.3.2: each facility's condition falls by its class's
// Decay to a floor of 0, and a facility at 0 is ruined at once, in ascending ID. Decay emits
// nothing, so the one pass gives the events of two (the ruins in ascending ID). The cause is
// use when the condition was already 0 when the step began, worn out by the starts of the
// tick, and decay when the fall of this step took it to 0 (6.2.1). A facility built or
// repaired in the tick is in the list, so it decays at the end of its tick (6.3.3).
func (n *infrastructure) decayAndRuin() {
	for i := 0; i < len(n.fac); {
		f := &n.fac[i]
		before := f.condition
		after := max(before-n.cls[f.class].decay, 0)
		if after > 0 {
			f.condition = after
			i++
			continue
		}
		cause := CauseDecay
		if before == 0 {
			cause = CauseUse
		}
		n.world.ruinFacility(i, cause) // removes record i: the next facility is at i
	}
}

// installPending is step 3 of 6.3.2: each store facility whose store has not been given
// asks storage for it, in ascending ID. The owner's account must be in the facility's
// region (5.4.2); any refusal of the operation (away, no base room, a limit, the store
// class's flag fallen) leaves the facility pending, counted, to be retried at the next
// step. On success storage's StoreChanged comes first and then FacilityChanged (installed).
func (n *infrastructure) installPending() {
	w := n.world
	for i := range n.fac {
		f := &n.fac[i]
		c := &n.cls[f.class]
		if c.kind != kindStore || f.installed {
			continue
		}
		a := w.agentOf(f.owner)
		if w.regionOf(a) != f.region || w.addStore(a, c.store, CauseFacility) != nil {
			n.countPending(f)
			continue
		}
		n.install(i)
		w.reportFacility(f, FacilityInstalled, CauseRecipe, f.condition, 0)
	}
}

// settle rebuilds the effect tables from the records and clamps the stock of every ground
// endowment that lost capacity (6.3.2 step 4, 6.5). The calls that change a facility and
// the step share it, so every position of a tick reads one set of effects.
func (n *infrastructure) settle() {
	n.rebuild()
	n.clampGround()
}

// clampGround lowers the stock of each ground endowment, in ascending (region, resource),
// to its effective capacity when a fall of the facility layer left it above (5.3.3). The
// excess is lost: it is counted in the identity of spec-resources.md 6.4 and reported by
// EndowmentAdjusted of kind facility, only when units were lost. The stock never rises
// with the capacity.
func (n *infrastructure) clampGround() {
	w, r := n.world, n.world.res
	if n.ground == nil || r == nil {
		return
	}
	for cell := range n.ground {
		c := r.effectiveCapacity(cell)
		before := r.stock[cell]
		if before <= c {
			continue
		}
		r.stock[cell] = c
		r.lost[cell] += before - c
		w.deliverAdjusted(cell, EndowmentAdjusted{Kind: AdjustKindFacility, StockBefore: before, Lost: before - c})
	}
}

// wire sets the layers a facility writes into other modules, each a function value or a
// table of this module that the owner reads and never writes (spec-infrastructure.md 5.2,
// 5.3): the route gain into transport's dispatch and Travel, the ground gain into the
// effective capacity of an endowment. The store seam is set beside it, in construct.go.
func (n *infrastructure) wire(w *World) {
	if w.transport != nil && n.route != nil {
		w.transport.layer = n.layer
	}
	if w.res != nil && n.ground != nil {
		if len(n.ground) != len(w.res.stock) {
			invariant("the ground table of the infrastructure module does not match the endowments")
		}
		w.res.facilityBP = n.ground
	}
}

// layer is the transit factor of the directed pair after the gain of its route
// facilities (spec-infrastructure.md 5.2.2): 0 stays 0, a pair whose own direct edge is not
// its shortest path has no gain (5.2.5), and otherwise F_eff = min(MaxTransitFactorBP,
// floor(slot * (10 000 + B) / 10 000)) with B the capped sum of the table. Nothing is
// allocated.
func (n *infrastructure) layer(from, to RegionID, mode ModeID, slot int64) int64 {
	if slot == 0 || int(mode) >= n.modes {
		return slot
	}
	b := n.route[(int(from)*n.regions+int(to))*n.modes+int(mode)]
	if b == 0 || !n.ownShortest(from, to) {
		return slot
	}
	f, ok := fcy.Layer(slot, b, MaxTransitFactorBP)
	if !ok {
		invariant("the transit factor of a route facility does not fit after validation")
	}
	return f
}

// ownShortest reports whether the configuration has the open directed edge (from, to) and
// its effective length is the shortest distance of the pair (5.2.5).
func (n *infrastructure) ownShortest(from, to RegionID) bool {
	geo := &n.world.geo
	e, ok := geo.edge(from, to)
	if !ok || e.Closed {
		return false
	}
	d, ok := geo.distance(from, to)
	return ok && e.Distance == d
}
