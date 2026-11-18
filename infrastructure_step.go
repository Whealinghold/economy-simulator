package economysim

// This file is the effect tables of the infrastructure module (spec-infrastructure.md
// 6.4): the derived structure that every position of a tick reads, rebuilt in one pass
// over the records. The step of the facility position P8.3b is added here by the change
// that builds time (plan 3-cap Phase 6).

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
			n.site[siteKey{f.owner, f.region, f.class}] += n.eff(f)
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
