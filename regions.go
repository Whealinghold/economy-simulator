package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/graph"
)

// edgeRules is the field phase of the edges and the overrides: the bounds of a
// distance and of a price (spec-regions.md 9.1). A path names the row by its
// index in the list as given, as a calendar month does.
func (v *validator) edgeRules() {
	for i, e := range v.c.Edges {
		v.inRange([]string{"edges[" + itoa(i) + "]", "distance"}, e.Distance, 1, MaxDistance)
	}
	for i, p := range v.c.RegionPrices {
		v.inRange([]string{"region_prices[" + itoa(i) + "]", "price"}, int64(p.Price), 1, MaxPrice)
	}
}

// graphRules is the reference phase of the edges and the overrides: both ends
// are defined, an edge is not a loop, and a pair or a key has one row
// (spec-regions.md 4.1.3, 5.3.1, 5.3.2). The sets v.region and v.commodity must
// be declared.
func (v *validator) graphRules() {
	r := &v.rep
	pairs := make(map[[2]string]bool, len(v.c.Edges))
	for i, e := range v.c.Edges {
		base := "edges[" + itoa(i) + "]"
		if !v.region[e.From] {
			r.add(ErrUnknownRegion, []string{base, "from"}, e.From, "")
		}
		if !v.region[e.To] {
			r.add(ErrUnknownRegion, []string{base, "to"}, e.To, "")
		}
		if e.From == e.To {
			r.add(ErrSelfEdge, []string{base, "to"}, e.To, "")
		}
		key := [2]string{e.From, e.To}
		if pairs[key] {
			r.add(ErrDuplicateID, []string{base}, e.From+"/"+e.To, "")
		}
		pairs[key] = true
	}
	keys := make(map[[2]string]bool, len(v.c.RegionPrices))
	for i, p := range v.c.RegionPrices {
		base := "region_prices[" + itoa(i) + "]"
		if !v.region[p.Region] {
			r.add(ErrUnknownRegion, []string{base, "region"}, p.Region, "")
		}
		if !v.commodity[p.Commodity] {
			r.add(ErrUnknownCommodity, []string{base, "commodity"}, p.Commodity, "")
		}
		key := [2]string{p.Region, p.Commodity}
		if keys[key] {
			r.add(ErrDuplicateID, []string{base}, p.Region+"/"+p.Commodity, "")
		}
		keys[key] = true
	}
}

// overrides indexes the reference-price overrides by (region, commodity) for
// the validator; the first row of a key wins, and a repeat is reported by
// graphRules. It is a lookup only.
func overrides(rows []RegionPrice) map[[2]string]Price {
	m := make(map[[2]string]Price, len(rows))
	for _, p := range rows {
		key := [2]string{p.Region, p.Commodity}
		if _, dup := m[key]; !dup {
			m[key] = p.Price
		}
	}
	return m
}

// refPrice returns the effective reference price of a commodity in a region:
// the override's price if there is a row, else the commodity's
// (spec-regions.md 5.2.2). ok is false when the commodity is not declared or
// the price is outside [1, MaxPrice]; the rule for that has spoken, and the
// rules that read the price skip it.
func (v *validator) refPrice(region, commodity string) (price int64, ok bool) {
	p, has := v.override[[2]string{region, commodity}]
	if !has {
		cm := v.lk.commodity[commodity]
		if cm == nil {
			return 0, false
		}
		p = cm.ReferencePrice
	}
	if p < 1 || p > MaxPrice {
		return 0, false
	}
	return int64(p), true
}

// refPrice returns the effective reference price of commodity k in region r of
// a canonical configuration: the override's price if there is a row, else the
// commodity's (spec-regions.md 5.2.2). It is a binary search over the sorted
// overrides and allocates nothing. The world resolves it once into a dense
// array (World.ReferencePrice); the solver, which has no world, calls this.
func (c *Config) refPrice(r RegionID, k CommodityID) Price {
	region, commodity := c.Regions[r].ID, c.Commodities[k].ID
	i, ok := slices.BinarySearchFunc(c.RegionPrices, [2]string{region, commodity}, func(p RegionPrice, key [2]string) int {
		return cmp.Or(cmp.Compare(p.Region, key[0]), cmp.Compare(p.Commodity, key[1]))
	})
	if ok {
		return c.RegionPrices[i].Price
	}
	return c.Commodities[k].ReferencePrice
}

// geometry is the region graph of a configuration read as given, for the
// warnings: the sorted region IDs, the edges that are valid (known ends, no
// loop, a distance in bounds, the first of a repeated pair) in ascending
// (From, To) order, and for each the index of its row in Edges.
type geometry struct {
	ids   []string
	edges []graph.Edge
	row   []int
}

// geometryRow is an edge of the graph with the index of its row in Edges.
type geometryRow struct {
	e graph.Edge
	i int
}

// readGeometry builds the geometry of c. It reads a configuration that may not
// validate, so it skips what is invalid, and it returns false when a count is
// over its bound, since the tables take n*n values.
func readGeometry(c *Config) (geometry, bool) {
	if len(c.Regions) > MaxRegions || len(c.Edges) > MaxEdges {
		return geometry{}, false
	}
	var g geometry
	for _, r := range c.Regions {
		g.ids = append(g.ids, r.ID)
	}
	slices.Sort(g.ids)
	g.ids = slices.Compact(g.ids)
	rows := g.validRows(c.Edges)
	for k, r := range rows {
		if k > 0 && rows[k-1].e.From == r.e.From && rows[k-1].e.To == r.e.To {
			continue
		}
		g.edges = append(g.edges, r.e)
		g.row = append(g.row, r.i)
	}
	return g, true
}

// validRows returns the valid edges of the list as rows, in ascending (From, To)
// order, a repeated pair in its written order.
func (g *geometry) validRows(edges []Edge) []geometryRow {
	index := func(id string) int {
		i, ok := slices.BinarySearch(g.ids, id)
		if !ok {
			return -1
		}
		return i
	}
	var rows []geometryRow
	for i, e := range edges {
		from, to := index(e.From), index(e.To)
		if from < 0 || to < 0 || from == to || e.Distance < 1 || e.Distance > MaxDistance {
			continue
		}
		rows = append(rows, geometryRow{graph.Edge{From: from, To: to, Length: e.Distance}, i})
	}
	slices.SortStableFunc(rows, func(a, b geometryRow) int {
		return cmp.Or(cmp.Compare(a.e.From, b.e.From), cmp.Compare(a.e.To, b.e.To))
	})
	return rows
}

// shapeWarnings are the three rules about the shape of the graph: a region with
// no edge, more than one component, and an edge longer than a detour. They need
// at least two regions (spec-regions.md 4.5.2).
func (g *geometry) shapeWarnings(add func(rule string, segs ...string)) {
	if len(g.ids) < 2 {
		return
	}
	touched := make([]bool, len(g.ids))
	for _, e := range g.edges {
		touched[e.From], touched[e.To] = true, true
	}
	for i, id := range g.ids {
		if !touched[i] {
			add("region_isolated", "regions", id)
		}
	}
	t, err := graph.Build(len(g.ids), g.edges)
	if err != nil {
		return
	}
	if t.Comps() > 1 {
		add("graph_not_strongly_connected", "edges")
	}
	for k, e := range g.edges {
		if d, ok := t.Dist(e.From, e.To); ok && d < e.Length {
			add("edge_not_shortest", "edges["+itoa(g.row[k])+"]")
		}
	}
}

// regionWarnings returns the five warnings of spec-regions.md 9.2 for c read as
// given. A world of one region and no edges draws none of the first three
// (4.5.2). The exception of region_without_agents for an outside node of the
// external module is the marked line below, which the external module fills.
func regionWarnings(c *Config) []Warning {
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }

	if g, ok := readGeometry(c); ok {
		g.shapeWarnings(add)
	}

	hasGroup := make(map[string]bool, len(c.Groups))
	for _, gr := range c.Groups {
		hasGroup[gr.Region] = true
	}
	for _, r := range c.Regions {
		// The outside node of the external module is exempt here (spec-regions.md
		// 9.2); the external module fills this line when it is built.
		if !hasGroup[r.ID] {
			add("region_without_agents", "regions", r.ID)
		}
	}

	prices := make(map[string]Price, len(c.Commodities))
	for _, cm := range c.Commodities {
		prices[cm.ID] = cm.ReferencePrice
	}
	for i, p := range c.RegionPrices {
		if def, ok := prices[p.Commodity]; ok && def == p.Price {
			add("override_equals_default", "region_prices["+itoa(i)+"]", "price")
		}
	}
	return out
}
