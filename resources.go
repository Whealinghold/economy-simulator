package economysim

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/curve"
	"github.com/Whealinghold/economy-sim/internal/endowment"
)

// This file is the resources module (spec-resources.md; architecture.md 2.10.1):
// its shapes, its canonical form and the lookup of a resource by ID. The pure
// rules (regeneration, abundance, the ration) are in internal/endowment and
// internal/curve. The validator and the warnings are in resources_check.go, the
// tick positions and the take in resources_tick.go. Outside these files the
// module appears only in the catalogue files (config.go, hash.go, bounds.go,
// defaults.go, errors.go, validate.go) and at the named hooks of ADR 0010.

// The shapes of a resource (spec-resources.md 4.2): the closed set of the rules
// by which an endowment regenerates.
const (
	// ShapeConstant is a flow of RatePerYear units a year that does not depend on
	// the stock; with a rate of 0 it is a finite deposit.
	ShapeConstant = "constant"
	// ShapeRefill closes the gap between the stock and the capacity at RateBP a
	// year.
	ShapeRefill = "refill"
	// ShapeLogistic grows in proportion to the stock and to the room left, at
	// RateBP a year, with a Refuge that regrows from nothing.
	ShapeLogistic = "logistic"
)

// canonical returns a deep copy of r in canonical order (spec-config.md 3.4,
// spec-resources.md 3.3.3): resources by ID, knots by share, season entries in
// the calendar's season order (an entry the calendar lacks, which fails
// validation, goes last by name) and rows by region then resource. A nil module
// stays nil, and a module that is on stays non-nil. A row's value that its Has
// flag does not enable is zeroed, so two rows that mean the same hash the same.
func (r *Resources) canonical(calendar []Season) *Resources {
	if r == nil {
		return nil
	}
	out := &Resources{
		Resources:    sortedBy(r.Resources, func(a, b Resource) int { return cmp.Compare(a.ID, b.ID) }),
		Endowments:   sortedBy(r.Endowments, byEndowmentKey),
		HorizonYears: r.HorizonYears,
	}
	order := func(season string) int {
		return slices.IndexFunc(calendar, func(s Season) bool { return s.ID == season })
	}
	for i := range out.Resources {
		res := &out.Resources[i]
		res.Abundance = sortedBy(res.Abundance, func(a, b AbundanceKnot) int { return cmp.Compare(a.Share, b.Share) })
		res.Seasons = sortedBy(res.Seasons, func(a, b SeasonRegen) int {
			ia, ib := order(a.Season), order(b.Season)
			if ia < 0 {
				ia = len(calendar)
			}
			if ib < 0 {
				ib = len(calendar)
			}
			return cmp.Or(cmp.Compare(ia, ib), cmp.Compare(a.Season, b.Season))
		})
	}
	for i := range out.Endowments {
		e := &out.Endowments[i]
		if !e.HasCapacity {
			e.Capacity = 0
		}
		if !e.HasInitial {
			e.Initial = 0
		}
		if !e.HasRegenScale {
			e.RegenScaleBP = 0
		}
	}
	return out
}

// byEndowmentKey orders two rows by region and then resource.
func byEndowmentKey(a, b Endowment) int {
	return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Resource, b.Resource))
}

// resourceIndex indexes the resources of c by ID without validating (a repeated
// ID keeps its first definition). It is nil when the module is off. The pointers
// point into the caller's slices, so an index is read-only.
func resourceIndex(c *Config) map[string]*Resource {
	r := c.Modules.Resources
	if r == nil {
		return nil
	}
	out := make(map[string]*Resource, len(r.Resources))
	for i := range r.Resources {
		if e := &r.Resources[i]; out[e.ID] == nil {
			out[e.ID] = e
		}
	}
	return out
}

// resDraw is one draw of a recipe as the rules read it: the resource and the
// units per execution.
type resDraw struct {
	resource ResourceID
	quantity int64
}

// resRequest is the request of one agent on one endowment in a tick: recorded at
// P2.5 with the executions the agent could start ignoring draws (spec-resources.md
// 5.2 step 2), granted at P2.6 and read by the start at P2.7. cell is the dense
// index of the endowment.
type resRequest struct {
	agent      AgentID
	cell       int
	executions int64
	draw       int64
	grant      int64
}

// resourcesState is the state of the module in a world, nil when the module is
// off (spec-resources.md 3.2.2, 15). It holds what is derived from the
// configuration (the capacity of each endowment, the abundance curves, the draws
// of each recipe), the saved state (the stocks and the capacity scales) and the
// scratch of a tick (the snapshot, the requests and the counts the events
// report). Every array of endowments is dense, indexed region*resources+resource.
type resourcesState struct {
	regions, resources int
	// capacity is the configured capacity of each endowment, the row's else the
	// resource's; knots the abundance curve of each resource, nil for a flat one;
	// drawsOf the draws of each recipe in ascending resource order; roleRes the
	// resources the recipes of each role draw on, ascending. All are derived from
	// the configuration and never saved.
	capacity []int64
	knots    [][]curve.Knot
	drawsOf  [][]resDraw
	roleRes  [][]ResourceID
	// shape, rate and refuge are the regeneration rule of each resource (rate is
	// units a year for a constant stock, basis points a year for the others);
	// seasonBP its factor by season of the calendar, nil for 10 000 in every
	// season; regenRow the regeneration scale of the row of each endowment, 10 000
	// without one. ticksPerYear is the calendar's. All are derived from the
	// configuration and never saved.
	shape        []endowment.Shape
	rate, refuge []int64
	seasonBP     [][]int64
	regenRow     []int64
	ticksPerYear int64
	// stock is the units in each endowment, carry the remainder of its regeneration
	// in units of 1/D, exhaustedRun the consecutive ticks with units short, and
	// capScale and regenScale the scales of its shock slots in basis points, 10 000
	// by default. All are saved.
	stock, carry, exhaustedRun []int64
	capScale, regenScale       []int64
	// base is the stock at the last tick boundary (the end of P8.3 of the last
	// tick, or the start): the first term of the identity of spec-resources.md 6.4,
	// read by the audit. It is rebuilt from the stock at a restore.
	base []int64
	// snapStock and snapAbundance are the stock and the abundance of each
	// endowment taken at P2.3: fixed for the tick (spec-resources.md 4.7.3, 5.2).
	snapStock, snapAbundance []int64
	// requested, granted, drawn, denied and reduced count the tick for the event of
	// 14.1: the units asked, the units the ration granted, the units the starts took,
	// and the requests granted 0 and granted less than asked. regenerated is the
	// units the regeneration kept, lost the units a capacity clamp removed and
	// adjusted the signed sum of the effective adjustments; the last two include the
	// calls made between ticks since the last tick, which belong to the tick that
	// follows (spec-resources.md 6.4). They are kept whether or not an observer is
	// attached and cleared at P8.6.
	requested, granted, drawn, denied, reduced []int64
	regenerated, lost, adjusted                []int64
	// reqs are the requests of the tick in the order P2.5 made them, order the
	// indexes of reqs sorted by endowment then agent, ration the requests of one
	// endowment as internal/endowment takes them and grants its result by agent
	// slot. All keep their capacity from tick to tick.
	reqs   []resRequest
	order  []int
	ration []endowment.Request
	grants []int64
}

// newResourcesState builds the state of the module for a canonical, validated
// configuration, or returns nil when the module is off. Each stock starts at the
// row's Initial, else the capacity times InitialBP over 10 000 truncated
// (spec-resources.md 3.3, FR-RES-006); a pair with no capacity has none.
func newResourcesState(c *Config) *resourcesState {
	m := c.Modules.Resources
	if m == nil {
		return nil
	}
	regions, resources := len(c.Regions), len(m.Resources)
	cells := regions * resources
	r := &resourcesState{
		regions: regions, resources: resources,
		capacity: make([]int64, cells), stock: make([]int64, cells), capScale: make([]int64, cells),
		carry: make([]int64, cells), exhaustedRun: make([]int64, cells), regenScale: make([]int64, cells),
		base: make([]int64, cells), regenRow: make([]int64, cells), regenerated: make([]int64, cells),
		lost: make([]int64, cells), adjusted: make([]int64, cells), ticksPerYear: c.Calendar.TicksPerYear(),
		shape: make([]endowment.Shape, resources), rate: make([]int64, resources), refuge: make([]int64, resources),
		seasonBP:  make([][]int64, resources),
		snapStock: make([]int64, cells), snapAbundance: make([]int64, cells),
		requested: make([]int64, cells), granted: make([]int64, cells), drawn: make([]int64, cells),
		denied: make([]int64, cells), reduced: make([]int64, cells),
		knots:   make([][]curve.Knot, resources),
		drawsOf: make([][]resDraw, len(c.Recipes)),
		roleRes: make([][]ResourceID, len(c.Roles)),
	}
	for i := range m.Resources {
		r.knots[i] = curveKnots(m.Resources[i].Abundance)
		r.setRule(i, &m.Resources[i], c.Calendar.Seasons)
	}
	rows := make(map[[2]string]*Endowment, len(m.Endowments))
	for i := range m.Endowments {
		rows[[2]string{m.Endowments[i].Region, m.Endowments[i].Resource}] = &m.Endowments[i]
	}
	for g := range regions {
		for j := range m.Resources {
			r.start(g*resources+j, &m.Resources[j], rows[[2]string{c.Regions[g].ID, m.Resources[j].ID}])
		}
	}
	r.deriveDraws(c)
	copy(r.base, r.stock)
	return r
}

// setRule sets the regeneration rule of resource i: its shape, its rate and refuge
// and, when it has a profile, the factor of each season of the calendar (a season
// the profile does not list has 0; spec-resources.md 4.5.1).
func (r *resourcesState) setRule(i int, res *Resource, seasons []Season) {
	r.refuge[i] = res.Refuge
	switch res.Shape {
	case ShapeRefill:
		r.shape[i], r.rate[i] = endowment.Refill, res.RateBP
	case ShapeLogistic:
		r.shape[i], r.rate[i] = endowment.Logistic, res.RateBP
	default:
		r.shape[i], r.rate[i] = endowment.Constant, res.RatePerYear
	}
	if len(res.Seasons) == 0 || len(seasons) == 0 {
		return
	}
	r.seasonBP[i] = make([]int64, len(seasons))
	for _, sr := range res.Seasons {
		if k := slices.IndexFunc(seasons, func(s Season) bool { return s.ID == sr.Season }); k >= 0 {
			r.seasonBP[i][k] = sr.BP
		}
	}
}

// deriveDraws sets the draws of each recipe, in ascending resource order, and the
// resources each role's recipes draw on.
func (r *resourcesState) deriveDraws(c *Config) {
	for i := range c.Recipes {
		for _, d := range c.Recipes[i].Draws {
			id, ok := c.resourceID(d.Resource)
			if !ok {
				invariant("a draw names the resource " + d.Resource + " after validation")
			}
			r.drawsOf[i] = append(r.drawsOf[i], resDraw{resource: id, quantity: int64(d.Quantity)})
		}
	}
	for i := range c.Roles {
		for _, id := range c.Roles[i].Recipes {
			rc, _ := c.recipeID(id)
			for _, d := range r.drawsOf[rc] {
				if !slices.Contains(r.roleRes[i], d.resource) {
					r.roleRes[i] = append(r.roleRes[i], d.resource)
				}
			}
		}
		slices.Sort(r.roleRes[i])
	}
}

// start sets the capacity, the scale and the starting stock of one endowment from
// its resource and its row, which is nil when it has none.
func (r *resourcesState) start(cell int, res *Resource, row *Endowment) {
	capacity, initial, hasInitial := res.Capacity, int64(0), false
	if row != nil {
		if row.HasCapacity {
			capacity = row.Capacity
		}
		initial, hasInitial = row.Initial, row.HasInitial
	}
	r.capacity[cell], r.capScale[cell], r.regenScale[cell], r.regenRow[cell] = capacity, wholeBP, wholeBP, wholeBP
	if row != nil && row.HasRegenScale {
		r.regenRow[cell] = row.RegenScaleBP
	}
	if !hasInitial {
		var ok bool
		if initial, ok = checked.MulDiv(capacity, res.InitialBP, wholeBP); !ok {
			invariant("the starting stock of an endowment does not fit after validation")
		}
	}
	r.stock[cell] = min(initial, r.effectiveCapacity(cell))
}

// effectiveCapacity is the capacity of an endowment after its capacity scale
// (spec-resources.md 3.4); the facility layer is 0 while no module adds one.
func (r *resourcesState) effectiveCapacity(cell int) int64 {
	c, ok := endowment.EffectiveCapacity(r.capacity[cell], r.capScale[cell], 0, MaxEndowment)
	if !ok {
		invariant("the effective capacity of an endowment does not fit after validation")
	}
	return c
}

// abundanceOf is the abundance factor of an endowment holding stock units.
func (r *resourcesState) abundanceOf(cell int, stock int64) int64 {
	a, ok := endowment.Abundance(r.knots[cell%r.resources], stock, r.effectiveCapacity(cell))
	if !ok {
		invariant("the abundance of an endowment does not fit after validation")
	}
	return a
}

// cellOf is the dense index of an endowment.
func (r *resourcesState) cellOf(region RegionID, resource ResourceID) int {
	return int(region)*r.resources + int(resource)
}

// recipeAbundance is the abundance of a recipe in a region: the least over its
// draws (spec-resources.md 4.7.4), 10 000 for a recipe with none. live reads the
// stock now, as the cost anchors of P1.2 do; otherwise the snapshot of P2.3.
func (r *resourcesState) recipeAbundance(region RegionID, recipe RecipeID, live bool) int64 {
	least := int64(wholeBP)
	for _, d := range r.drawsOf[recipe] {
		cell := r.cellOf(region, d.resource)
		a := r.snapAbundance[cell]
		if live {
			a = r.abundanceOf(cell, r.stock[cell])
		}
		least = min(least, a)
	}
	return least
}

// clearTick zeroes the counts of the tick (P8.6): the requests, the grants, the
// units drawn and the requests cut, the units regenerated, lost and adjusted, and
// empties the lists of requests, and the stock becomes the base of the next tick.
// The saved state stays; the snapshot and the grants are rebuilt at P2.3 and P2.6
// and are left out of the position hash (poshash_internal_test.go).
func (r *resourcesState) clearTick() {
	clear(r.requested)
	clear(r.granted)
	clear(r.drawn)
	clear(r.denied)
	clear(r.reduced)
	clear(r.regenerated)
	clear(r.lost)
	clear(r.adjusted)
	copy(r.base, r.stock)
	r.reqs, r.order, r.ration = r.reqs[:0], r.order[:0], r.ration[:0]
}

// resourceID returns the typed ID of a resource, its index in the canonical
// Resources (spec-resources.md 3.2.3). The receiver must already be canonical.
func (c Config) resourceID(id string) (ResourceID, bool) {
	if c.Modules.Resources == nil {
		return 0, false
	}
	i, ok := slices.BinarySearchFunc(c.Modules.Resources.Resources, id, func(e Resource, t string) int { return cmp.Compare(e.ID, t) })
	return ResourceID(i), ok
}

// resourceNames returns the string ID of each ResourceID, nil with the module off.
func resourceNames(c *Config) []string {
	m := c.Modules.Resources
	if m == nil {
		return nil
	}
	out := make([]string, len(m.Resources))
	for i := range m.Resources {
		out[i] = m.Resources[i].ID
	}
	return out
}

// snapshotEndowments returns the Endowments rows of the snapshot, one per region and
// resource in that order, empty with the module off (spec-resources.md 14.4).
func (w *World) snapshotEndowments() []EndowmentRow {
	r := w.res
	if r == nil {
		return nil
	}
	rows := make([]EndowmentRow, len(r.stock))
	for cell := range rows {
		rows[cell] = EndowmentRow{
			Region: RegionID(cell / r.resources), Resource: ResourceID(cell % r.resources),
			Stock: r.stock[cell], Capacity: r.effectiveCapacity(cell), AbundanceBP: r.abundanceOf(cell, r.stock[cell]),
			ExhaustedRun: r.exhaustedRun[cell], CapacityScaleBP: r.capScale[cell], RegenScaleBP: r.regenScale[cell],
		}
	}
	return rows
}

// saveResources is tag 15 (spec-resources.md 15, spec-checkpoint.md 4.2): for each
// endowment with a configured capacity above 0, in (RegionID, ResourceID) order, the
// stock, the carry and the run of ticks with units short (a depleted stock is saved as
// 0, not restored as the starting stock); then the shock scales that are not both
// 10 000 as a count and (region, resource, capacity scale, regeneration scale)
// in the same order. The effective values, the snapshot, the grants and the counts of
// a tick are derived or cleared and are not saved.
func (w *World) saveResources(cw *codec.Writer) error {
	r := w.res
	for cell, c := range r.capacity {
		if c > 0 {
			cw.U(uint64(r.stock[cell]))
			cw.U(uint64(r.carry[cell]))
			cw.U(uint64(r.exhaustedRun[cell]))
		}
	}
	n := 0
	for cell := range r.capScale {
		if r.shocked(cell) {
			n++
		}
	}
	cw.Count(n)
	for cell := range r.capScale {
		if r.shocked(cell) {
			cw.U(uint64(cell / r.resources))
			cw.U(uint64(cell % r.resources))
			cw.U(uint64(r.capScale[cell]))
			cw.U(uint64(r.regenScale[cell]))
		}
	}
	return nil
}

// shocked reports whether either shock scale of an endowment is not the default.
func (r *resourcesState) shocked(cell int) bool {
	return r.capScale[cell] != wholeBP || r.regenScale[cell] != wholeBP
}

// loadResources reads tag 15: each stock at most MaxEndowment, each carry below
// D = 10^12 * TicksPerYear, each run at most the ticks elapsed, then the shock rows
// with the key strictly ascending, each scale at most MaxShockBP and a row not both
// default. That a stock lies inside its effective capacity, which the scales
// decide, is the cross-check (resourcesCross). The base of the identity is the
// stock restored.
func (w *World) loadResources(_ *restoreCtx, r *codec.Reader) error {
	s := w.res
	d := 1_000_000_000_000 * s.ticksPerYear
	for cell, c := range s.capacity {
		if c > 0 {
			s.stock[cell] = r.Bounded("endowment stock", MaxEndowment)
			s.carry[cell] = r.Bounded("endowment carry", d-1)
			s.exhaustedRun[cell] = r.Bounded("endowment run", w.Tick())
		}
	}
	prev := -1
	for i := range r.Count(4) {
		region, resource := r.Index(uint64(s.regions)), r.Index(uint64(s.resources))
		capacity, regen := r.Bounded("capacity scale", MaxShockBP), r.Bounded("regeneration scale", MaxShockBP)
		cell := int(region)*s.resources + int(resource)
		switch {
		case r.Err() != nil:
			return nil
		case cell <= prev:
			r.Failf("scale row %d names endowment %d after endowment %d, want strictly ascending", i, cell, prev)
			return nil
		case capacity == wholeBP && regen == wholeBP:
			r.Failf("scale row %d names endowment %d with both scales at the default, which is not saved", i, cell)
			return nil
		}
		prev, s.capScale[cell], s.regenScale[cell] = cell, capacity, regen
	}
	copy(s.base, s.stock)
	return nil
}

// resourcesCross is the module's part of the cross-checks of a restore: every stock
// lies in [0, effective capacity] under the scales just read.
func (w *World) resourcesCross() error {
	r := w.res
	for cell, st := range r.stock {
		if c := r.effectiveCapacity(cell); st > c {
			return fmt.Errorf("%w: endowment %d holds %d units, above its effective capacity %d", ErrCheckpointCorrupt, cell, st, c)
		}
	}
	return nil
}
