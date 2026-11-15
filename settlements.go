package economysim

import (
	"maps"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/settlement"
)

// This file is the settlements module (spec-settlements.md, SET): the size class of
// each region, derived from its households at P7.2a, and the tables of flags and
// limits a class carries. It holds the state, the construction of the class, the
// review and the two reads of a class (regionHas, regionLimit). The rules are in
// internal/settlement over primitives; the events, logs, snapshot rows and the
// checkpoint section are in settlements_report.go and the validator in
// settlements_validate.go (architecture.md 2.6, 2.10.1).

// settlements is the state of the module, nil when Modules.Settlements is nil. Per
// region (an outside node has a row that is never read) it holds the class and the
// tick of its last change, which are the saved state (spec-settlements.md 11); the
// effective thresholds, the floor and the tables are derived at construction and
// rebuilt by a restore (spec-settlements.md 4.2, 11).
type settlements struct {
	m *Settlements
	// n is the number of classes.
	n int
	// e and l are the effective enter and leave thresholds of each region, by class
	// (spec-settlements.md 4.2); floor is the index of the lowest class of each
	// region.
	e, l  [][]int64
	floor []int
	// class and since are the saved state: the class of each region as of its last
	// review and the tick of its last change, 0 at construction.
	class []int
	since []int64
	// hh is the household count of each region the review of the tick used,
	// scratch allocated once.
	hh []int64
	// tab holds the flags and limits of each class, dense (spec-settlements.md 3.3).
	tab *settlement.Table
}

// newSettlements builds the state of the module from a canonical, validated
// configuration, or returns nil when the module is off. The classes of the regions
// are 0 until found sets them (a world under construction) or a restore installs
// them.
func newSettlements(c *Config) *settlements {
	m := c.Modules.Settlements
	if m == nil {
		return nil
	}
	e, l, ok := settlementThresholds(c, m)
	if !ok {
		invariant("the effective thresholds of the settlement classes do not fit after validation")
	}
	n := len(c.Regions)
	s := &settlements{
		m: m, n: len(m.Classes), e: e, l: l, floor: make([]int, n), class: make([]int, n),
		since: make([]int64, n), hh: make([]int64, n), tab: m.table(),
	}
	at := make(map[string]int, n) // lookup only
	for i := range c.Regions {
		at[c.Regions[i].ID] = i
	}
	for _, row := range m.Regions {
		g, ok := at[row.Region]
		if !ok {
			invariant("a settlement region row names a region that does not exist after validation")
		}
		s.floor[g] = m.classIndex(row.Floor)
	}
	return s
}

// table builds the dense tables of the canonical module: the flags and the limits
// are numbered by the sorted union of the names, and each class carries the indices
// of its own (spec-settlements.md 3.3, FR-SET-005, 020).
func (m *Settlements) table() *settlement.Table {
	flags := slices.Sorted(maps.Keys(m.flagSet()))
	limits := slices.Sorted(maps.Keys(m.limitSet()))
	classes := make([]settlement.Class, len(m.Classes))
	for k := range m.Classes {
		for _, f := range m.Classes[k].Flags {
			i, _ := slices.BinarySearch(flags, f)
			classes[k].Flags = append(classes[k].Flags, i)
		}
		for _, lim := range m.Classes[k].Limits {
			i, _ := slices.BinarySearch(limits, lim.Name)
			classes[k].Limits = append(classes[k].Limits, settlement.Limit{Index: i, Value: lim.Value})
		}
	}
	t, ok := settlement.NewTable(len(limits), classes)
	if !ok {
		invariant("the flag and limit tables of the settlement classes do not fit after validation")
	}
	return t
}

// found gives each region its construction class, the highest class whose enter
// threshold the households of the region meet and not below the region's floor, with
// no band and Since 0 (spec-settlements.md 4.5.1). It runs after the agents are
// created and before the first step; a nil module does nothing.
func (s *settlements) found(w *World) {
	if s == nil {
		return
	}
	w.householdCounts(s.hh)
	for g := range s.class {
		if !w.isOutsideNode(RegionID(g)) {
			s.class[g], s.since[g] = settlement.Initial(s.hh[g], s.e[g], s.floor[g]), 0
		}
	}
}

// p72aSettlement is P7.2a, the settlement review (spec-settlements.md 5.1): for each
// region that is not an outside node in ascending RegionID, the household count is
// taken once for all regions, the class reviewed and the change applied or held by
// the dwell. It reads the weights of the tick and writes the class and Since only; it
// draws nothing and moves no household, money or goods.
func (w *World) p72aSettlement() {
	s := w.set
	w.householdCounts(s.hh)
	t := w.Tick()
	for g := range s.class {
		if !w.isOutsideNode(RegionID(g)) {
			s.review(w, RegionID(g), t)
		}
	}
}

// review is the review of region g at tick t: the target class from the count, then
// the dwell test. A change sets the class and Since and is reported; a held change
// changes nothing and is reported as held, and is not remembered (spec-settlements.md
// 4.3, 4.4).
func (s *settlements) review(w *World, g RegionID, t int64) {
	c, h := s.class[g], s.hh[g]
	target := settlement.Review(c, s.floor[g], h, s.e[g], s.l[g])
	out, ok := settlement.Decide(c, target, s.since[g], t, s.m.Dwell)
	if !ok {
		invariant("the ticks since the last change of a settlement class overflow")
	}
	switch out.Kind {
	case settlement.Changed:
		s.class[g], s.since[g] = target, t
		cause := CauseGrew
		if target < c {
			cause = CauseShrank
		}
		w.reportClassChanged(g, SettlementClassID(c), SettlementClassID(target), h, cause)
	case settlement.Held:
		w.reportClassHeld(g, SettlementClassID(c), SettlementClassID(target), h, out.TicksLeft)
	}
}

// regionHas reports whether the class of region g, as of its last review, carries
// the flag, an index into the sorted flag names of the module (spec-settlements.md
// 6.1). It is a table read that allocates nothing; a nil module has no flag.
func (w *World) regionHas(g RegionID, flag int) bool {
	s := w.set
	return s != nil && s.tab.Has(s.class[g], flag)
}

// regionLimit returns the value of the limit, an index into the sorted limit names
// of the module, for the class of region g, and whether the class has one: a class
// with no row has no limit, and a value of 0 is a closed gate (spec-settlements.md
// 3.3.2, 6.1).
func (w *World) regionLimit(g RegionID, limit int) (int64, bool) {
	s := w.set
	if s == nil {
		return 0, false
	}
	return s.tab.Limit(s.class[g], limit)
}
