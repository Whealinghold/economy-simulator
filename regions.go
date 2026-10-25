package economysim

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/graph"
)

// The tables of internal/graph take n*n values, so its bound must hold ours.
const _ = uint(graph.MaxRegions - MaxRegions)

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
	cm := v.lk.commodity[commodity]
	if cm == nil {
		return 0, false
	}
	p := v.lk.price(region, cm)
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
		if _, dup := prices[cm.ID]; !dup { // the first row of a repeated ID, as newLookup keeps
			prices[cm.ID] = cm.ReferencePrice
		}
	}
	for i, p := range c.RegionPrices {
		if def, ok := prices[p.Commodity]; ok && def == p.Price {
			add("override_equals_default", "region_prices["+itoa(i)+"]", "price")
		}
	}
	return out
}

// EdgeID is the index of an edge in ascending order of (From, To). It does not
// change when an edge status changes (spec-regions.md 4.1.6).
type EdgeID int

// Hop is a region and a length: the effective length of an edge in Out and In,
// the shortest distance in Within.
type Hop struct {
	// Region is the region at the other end of the edge, or the region reached.
	Region RegionID
	// Distance is the length in the pack's distance unit.
	Distance int64
}

// EdgeView is the state of one edge.
type EdgeView struct {
	// ID is the index of the edge in ascending order of (From, To).
	ID EdgeID
	// From and To are the ends of the route.
	From, To RegionID
	// Base is the configured distance.
	Base int64
	// Distance is the effective length (spec-regions.md 7.2): Base when the
	// status is the default, and 0 for a closed edge, which has none.
	Distance int64
	// Closed says the edge is closed and absent from every query but Edge and
	// EdgeAt.
	Closed bool
	// FactorBP is the length factor of the status in basis points; 10 000 when
	// the status is the default.
	FactorBP int64
}

// defaultFactorBP is the factor of an edge whose status is the default: the
// configured distance (spec-regions.md 7.1.2).
const defaultFactorBP = 10_000

// edgeState is the status of one edge. The world keeps one per edge; a nil slice
// means every status is the default.
type edgeState struct {
	closed   bool
	factorBP int64
}

// edgeRec is an edge with its status and its effective length.
type edgeRec struct {
	from, to RegionID
	base     int64
	length   int64 // the effective length; 0 when closed
	closed   bool
	factorBP int64
}

// Graph is the region graph: plain data, independent of the world and of every
// other Graph. No method changes it. It describes the world at the time it was
// built and may be read from any goroutine. Keep it and reuse it: building one
// costs a full construction (spec-regions.md 4.4.5). The zero Graph has no
// region.
type Graph struct {
	names []string // the region IDs, ascending: the index is the RegionID
	edges []edgeRec
	// first holds, per region, where its out-edges start in edges (len n+1);
	// in holds the EdgeIDs ordered by (To, From) and inFirst where each region's
	// in-edges start in it (len n+1).
	first   []int32
	in      []int32
	inFirst []int32
	t       graph.Tables
}

// NewGraph runs the graph rules of spec-regions.md 9 on cfg alone and builds the
// graph of its edges, all open at their configured distance. It returns the first
// error of those rules, as a *ValidationError, in the order Validate sorts them
// in. It runs no other rule (the fields of a region, the commodities), so it
// accepts a configuration that Validate would refuse for one of those.
func NewGraph(cfg Config) (Graph, error) {
	// The rules read the lists as given, so a path indexes the caller's list as it
	// does in Validate.
	if err := graphRulesError(cfg); err != nil {
		return Graph{}, err
	}
	c := cfg.canonical()
	return newGraph(&c, nil)
}

// graphRulesError runs the rules of the region graph alone: the counts of regions,
// edges and overrides, the region IDs, and the edge and override rules. It
// returns the first error that Validate would report.
func graphRulesError(c Config) error {
	v := &validator{c: c}
	over := func(n, bound int, name string) {
		if n > bound {
			v.rep.add(ErrTooMany, []string{name}, itoa(n), itoa(bound))
		}
	}
	over(len(c.Regions), MaxRegions, "regions")
	over(len(c.Edges), MaxEdges, "edges")
	over(len(c.RegionPrices), MaxRegionPrices, "region_prices")
	if len(v.rep.errs) == 0 {
		v.graphDeclared()
		v.edgeRules()
		v.graphRules()
	}
	err := v.rep.err()
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		return joined.Unwrap()[0]
	}
	return err
}

// graphDeclared fills the sets the graph rules read: the region and commodity IDs,
// and reports the region rules a graph needs (at least one, valid, distinct).
func (v *validator) graphDeclared() {
	c := &v.c
	if len(c.Regions) == 0 {
		v.rep.add(ErrNoRegions, []string{"regions"}, "", "")
	}
	v.region = make(map[string]bool, len(c.Regions))
	for _, r := range c.Regions {
		if !validID(r.ID) {
			v.rep.add(ErrInvalidID, []string{"regions", r.ID, "id"}, r.ID, "")
		}
		if v.region[r.ID] {
			v.rep.add(ErrDuplicateID, []string{"regions", r.ID, "id"}, r.ID, "")
		}
		v.region[r.ID] = true
	}
	v.commodity = make(map[string]bool, len(c.Commodities))
	for _, cm := range c.Commodities {
		v.commodity[cm.ID] = true
	}
}

// newGraph builds the graph of a canonical configuration whose graph rules hold,
// with the given statuses by EdgeID (nil: every edge at the default). It reports
// ErrOverflow for a length or a path that does not fit int64, which the bounds
// make unreachable.
func newGraph(c *Config, status []edgeState) (Graph, error) {
	n := len(c.Regions)
	g := Graph{names: make([]string, n), first: make([]int32, n+1)}
	for i := range c.Regions {
		g.names[i] = c.Regions[i].ID
	}
	edges, open, err := edgeRecs(c, status)
	if err != nil {
		return Graph{}, err
	}
	g.edges = edges
	for _, e := range edges {
		g.first[e.from+1]++
	}
	for r := range n {
		g.first[r+1] += g.first[r]
	}
	g.in, g.inFirst = indexIn(edges, n)
	t, err := graph.Build(n, open)
	if errors.Is(err, graph.ErrOverflow) {
		return Graph{}, fmt.Errorf("region graph: %w", ErrOverflow)
	}
	if err != nil {
		invariant("the region graph rejected a validated edge list: " + err.Error())
	}
	g.t = *t
	return g, nil
}

// edgeRecs resolves the edges of a canonical configuration to RegionIDs and
// gives each its status and effective length, and returns the open ones as the
// input of the tables, in the same order.
func edgeRecs(c *Config, status []edgeState) ([]edgeRec, []graph.Edge, error) {
	edges := make([]edgeRec, len(c.Edges))
	open := make([]graph.Edge, 0, len(c.Edges))
	for i, e := range c.Edges {
		from, fromOK := c.regionID(e.From)
		to, toOK := c.regionID(e.To)
		if !fromOK || !toOK {
			invariant("an edge names an unknown region after validation")
		}
		rec := edgeRec{from: from, to: to, base: e.Distance, factorBP: defaultFactorBP}
		if status != nil {
			rec.closed, rec.factorBP = status[i].closed, status[i].factorBP
		}
		if !rec.closed {
			length, ok := graph.Effective(rec.base, rec.factorBP)
			if !ok {
				return nil, nil, fmt.Errorf("effective length of edge %d: %w", i, ErrOverflow)
			}
			rec.length = length
			open = append(open, graph.Edge{From: int(from), To: int(to), Length: length})
		}
		edges[i] = rec
	}
	return edges, open, nil
}

// indexIn returns the in-edge index of edges: their IDs ordered by (To, From), by
// a counting sort that keeps the ascending From of the edge order within one To,
// and where each of the n regions' in-edges start in it (n+1 values).
func indexIn(edges []edgeRec, n int) (in, inFirst []int32) {
	inFirst = make([]int32, n+1)
	for _, e := range edges {
		inFirst[e.to+1]++
	}
	for r := range n {
		inFirst[r+1] += inFirst[r]
	}
	in = make([]int32, len(edges))
	next := slices.Clone(inFirst[:n])
	for i, e := range edges {
		in[next[e.to]] = int32(i)
		next[e.to]++
	}
	return in, inFirst
}

// has reports whether r is a region of the graph.
func (g Graph) has(r RegionID) bool { return r >= 0 && int(r) < len(g.names) }

// Len returns the number of regions.
func (g Graph) Len() int { return len(g.names) }

// Name returns the ID string of a region, or "" if r is not a region.
func (g Graph) Name(r RegionID) string {
	if !g.has(r) {
		return ""
	}
	return g.names[r]
}

// Lookup returns the RegionID of an ID string.
func (g Graph) Lookup(id string) (RegionID, bool) {
	i, ok := slices.BinarySearch(g.names, id)
	if !ok {
		return 0, false
	}
	return RegionID(i), true
}

// Edges returns the number of configured edges, open or closed.
func (g Graph) Edges() int { return len(g.edges) }

// view returns the EdgeView of the edge at index i.
func (g Graph) view(i int) EdgeView {
	e := &g.edges[i]
	return EdgeView{ID: EdgeID(i), From: e.from, To: e.to, Base: e.base, Distance: e.length, Closed: e.closed, FactorBP: e.factorBP}
}

// Edge returns the direct route from one region to another, closed or not.
func (g Graph) Edge(from, to RegionID) (EdgeView, bool) {
	if !g.has(from) || !g.has(to) {
		return EdgeView{}, false
	}
	lo, hi := int(g.first[from]), int(g.first[from+1])
	i, ok := slices.BinarySearchFunc(g.edges[lo:hi], to, func(e edgeRec, t RegionID) int { return cmp.Compare(e.to, t) })
	if !ok {
		return EdgeView{}, false
	}
	return g.view(lo + i), true
}

// EdgeAt returns the edge with the given ID.
func (g Graph) EdgeAt(e EdgeID) (EdgeView, bool) {
	if e < 0 || int(e) >= len(g.edges) {
		return EdgeView{}, false
	}
	return g.view(int(e)), true
}

// Out returns the open edges that leave a region, in ascending To, with their
// effective length. Each call allocates the slice, which is the caller's.
func (g Graph) Out(from RegionID) []Hop {
	if !g.has(from) {
		return nil
	}
	var out []Hop
	for _, e := range g.edges[g.first[from]:g.first[from+1]] {
		if !e.closed {
			out = append(out, Hop{Region: e.to, Distance: e.length})
		}
	}
	return out
}

// In returns the open edges that enter a region, in ascending From, with their
// effective length. Each call allocates the slice, which is the caller's.
func (g Graph) In(to RegionID) []Hop {
	if !g.has(to) {
		return nil
	}
	var in []Hop
	for _, i := range g.in[g.inFirst[to]:g.inFirst[to+1]] {
		if e := &g.edges[i]; !e.closed {
			in = append(in, Hop{Region: e.from, Distance: e.length})
		}
	}
	return in
}

// Distance returns the shortest distance between two regions over the open
// edges, and (0, true) for a region and itself.
func (g Graph) Distance(from, to RegionID) (int64, bool) { return g.t.Dist(int(from), int(to)) }

// NextHop returns the region after from on the lexicographically smallest
// shortest path to a reachable, different region.
func (g Graph) NextHop(from, to RegionID) (RegionID, bool) {
	v, ok := g.t.Next(int(from), int(to))
	return RegionID(v), ok
}

// Path returns the regions of the lexicographically smallest shortest path from
// one region to another, both included; just the region when they are equal.
func (g Graph) Path(from, to RegionID) ([]RegionID, bool) {
	p, ok := g.t.AppendPath(nil, int(from), int(to))
	if !ok {
		return nil, false
	}
	out := make([]RegionID, len(p))
	for i, r := range p {
		out[i] = RegionID(r)
	}
	return out, true
}

// Within returns every region other than from whose shortest distance from it is
// at most limit, in ascending distance and, for equal distances, ascending
// RegionID. A negative limit gives none. Each call allocates the slice, which is the caller's.
func (g Graph) Within(from RegionID, limit int64) []Hop {
	hops := g.t.AppendWithin(nil, int(from), limit)
	out := make([]Hop, len(hops))
	for i, h := range hops {
		out[i] = Hop{Region: RegionID(h.Region), Distance: h.Distance}
	}
	return out
}

// Components returns the number of strongly connected components of the open
// graph.
func (g Graph) Components() int { return g.t.Comps() }

// Component returns the number of the component of a region, numbered from 0 in
// ascending order of the smallest RegionID in each.
func (g Graph) Component(r RegionID) (int, bool) { return g.t.Comp(int(r)) }

// regionState is what the world keeps of the region graph for its own reads: the
// graph as it stands, built from the configuration and the edge statuses, and the
// statuses by EdgeID (nil while every status is the default). The tables are
// derived and never saved; the statuses are saved state (spec-regions.md 11).
type regionState struct {
	g      Graph
	status []edgeState
}

// distance returns the shortest distance between two regions over the open edges.
func (r *regionState) distance(from, to RegionID) (int64, bool) {
	return r.g.t.Dist(int(from), int(to))
}

// next returns the next hop on the shortest path between two regions.
func (r *regionState) next(from, to RegionID) (RegionID, bool) {
	v, ok := r.g.t.Next(int(from), int(to))
	return RegionID(v), ok
}

// edge returns the direct route from one region to another, closed or not.
func (r *regionState) edge(from, to RegionID) (EdgeView, bool) { return r.g.Edge(from, to) }

// out appends to dst the open edges that leave a region, in ascending To.
func (r *regionState) out(dst []Hop, from RegionID) []Hop {
	if !r.g.has(from) {
		return dst
	}
	for _, e := range r.g.edges[r.g.first[from]:r.g.first[from+1]] {
		if !e.closed {
			dst = append(dst, Hop{Region: e.to, Distance: e.length})
		}
	}
	return dst
}

// components returns the number of strongly connected components of the open graph.
func (r *regionState) components() int { return r.g.t.Comps() }

// Graph returns the region graph as it stands, built fresh from the configuration
// and the current edge statuses. It shares no storage with the world or with any
// other call, and a later change of an edge status does not change it. One call
// costs a full construction: keep the value and reuse it (spec-regions.md 4.4.5).
// It works on a failed world.
func (w *World) Graph() Graph {
	g, err := newGraph(&w.cfg, w.geo.status)
	if err != nil {
		invariant("the region graph of a validated configuration failed to build: " + err.Error())
	}
	return g
}

// ReferencePrice returns the effective reference price of a commodity in a region:
// the region's override if it has one, else the commodity's price (spec-regions.md
// 5.2.2). It is read from a table resolved at construction and allocates nothing.
// An ID outside its range gives 0. It works on a failed world.
func (w *World) ReferencePrice(r RegionID, c CommodityID) Price {
	if r < 0 || int(r) >= len(w.cfg.Regions) || c < 0 || int(c) >= len(w.cfg.Commodities) {
		return 0
	}
	return w.refPrice(r, c)
}

// refPrice is the effective reference price of a commodity in a region, read from
// the table of construction. The IDs must be in range.
func (w *World) refPrice(r RegionID, c CommodityID) Price {
	return w.refs[int(r)*len(w.cfg.Commodities)+int(c)]
}

// resolveRefPrices fills the dense table of effective reference prices, indexed
// by region and then commodity (spec-regions.md 5.3.5).
func resolveRefPrices(c *Config) []Price {
	nc := len(c.Commodities)
	out := make([]Price, len(c.Regions)*nc)
	for r := range c.Regions {
		for k := range nc {
			out[r*nc+k] = c.refPrice(RegionID(r), CommodityID(k))
		}
	}
	return out
}
