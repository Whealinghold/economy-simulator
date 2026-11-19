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
	// needs and ceil are, per role in canonical order, the index of the flag its
	// Needs names and of the limit its Ceiling names, or -1 for none; the indices are
	// into the sorted names of tab (spec-settlements.md 6.1).
	needs, ceil []int
	// flags are the sorted flag names of the module: the index of a flag in them is
	// the index regionHas takes.
	flags []string
	// cnt is the households of each role in region cntRegion at the first read of
	// the entry step of cntTick, kept exact by noteRoleMove between two reads; it is
	// scratch, never saved (spec-settlements.md 6.4.2, plan 3-set SET11 b).
	cnt       []int64
	cntTick   int64
	cntRegion RegionID
	cntOK     bool
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
	s.flags = slices.Sorted(maps.Keys(m.flagSet()))
	s.needs, s.ceil = m.roleReaders(s.flags, c.Roles)
	s.cnt = make([]int64, len(c.Roles))
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

// roleReaders resolves, per role in canonical order, the flag its Needs names and the
// limit its Ceiling names to indices into the sorted flag and limit names, or -1 for
// none (spec-settlements.md 6.1).
func (m *Settlements) roleReaders(flags []string, roles []Role) (needs, ceil []int) {
	limits := slices.Sorted(maps.Keys(m.limitSet()))
	needs, ceil = make([]int, len(roles)), make([]int, len(roles))
	for i := range roles {
		needs[i], ceil[i] = nameIndex(flags, roles[i].Needs), nameIndex(limits, roles[i].Ceiling)
	}
	return needs, ceil
}

// nameIndex is the index of name in the sorted names, or -1 for the empty name.
func nameIndex(sorted []string, name string) int {
	if i, ok := slices.BinarySearch(sorted, name); ok && name != "" {
		return i
	}
	return -1
}

// fillRoleGates is the reader of H1 and H2 (spec-settlements.md 6.4.1, 6.4.2): for
// each role of the view of region g, Closed is also set when the role's Needs flag is
// not carried by the region's class, and Headroom and NoLimit say what the class limit
// of its Ceiling leaves, less the households of the role at the start of the entry
// step. The households stay: nothing here removes one (R3).
func (w *World) fillRoleGates(g RegionID) {
	s := w.set
	for r := range w.ex.view.Roles {
		rv := &w.ex.view.Roles[r]
		if f := s.needs[r]; f >= 0 && !w.regionHas(g, f) {
			rv.Closed = true
		}
		if l := s.ceil[r]; l >= 0 {
			if limit, ok := w.regionLimit(g, l); ok {
				rv.Headroom, rv.NoLimit = settlement.Headroom(limit, rv.Households), false
			}
		}
	}
}

// roleHeadroom is the class half of the headroom of role r in region g for the
// completion of a trainee, the slot skills.headroom (spec-institutions.md 4.5.2, plan
// 3-skl SK9): no limit for a role with no ceiling in the class, else the limit less the
// households of the role, which a completion has already moved, so two held trainees
// share the room in order.
func (w *World) roleHeadroom(g RegionID, r RoleID) (headroom int64, noLimit bool) {
	s := w.set
	if s.ceil[r] < 0 {
		return 0, true
	}
	limit, ok := w.regionLimit(g, s.ceil[r])
	if !ok {
		return 0, true
	}
	return settlement.Headroom(limit, s.roleCount(w, g, r)), false
}

// roleCount is the households of role r in region g. One scan of the agents per
// (tick, region) fills it at the first read (O(A), NFR-SET-001); noteRoleMove keeps it
// exact after.
func (s *settlements) roleCount(w *World, g RegionID, r RoleID) int64 {
	if t := w.Tick(); !s.cntOK || s.cntTick != t || s.cntRegion != g {
		clear(s.cnt)
		for i := w.firstHousehold; i < len(w.agents); i++ {
			a := &w.agents[i]
			if rg, ok := w.householdRegion(a); ok && rg == g {
				s.cnt[a.role] = satAdd(s.cnt[a.role], a.weight)
			}
		}
		s.cntOK, s.cntTick, s.cntRegion = true, t, g
	}
	return s.cnt[r]
}

// noteRoleMove is called after a completed traineeship moved weight households of
// region g from one role to another, so a count read before it stays exact for the
// next held trainee (hook H-14). It does nothing when no count is held for the region.
func (s *settlements) noteRoleMove(g RegionID, from, to RoleID, weight int64) {
	if s.cntOK && s.cntRegion == g {
		s.cnt[from] = max(s.cnt[from]-weight, 0) // a count never goes below 0, so headroom is never inflated
		s.cnt[to] = satAdd(s.cnt[to], weight)
	}
}

// storeClassGate is the gate of STO (spec-storage.md 3.2.3): a store class may be
// given in a region whose class carries the flag its Needs names. It is nil when no
// store class has a Needs, which leaves every class open.
func (w *World) storeClassGate() func(RegionID, int) bool {
	m := w.cfg.Modules.Storage
	if m == nil {
		return nil
	}
	flags := make([]int, len(m.Classes))
	any := false
	for i := range m.Classes {
		flags[i] = nameIndex(w.set.flags, m.Classes[i].Needs)
		any = any || flags[i] >= 0
	}
	if !any {
		return nil
	}
	return func(g RegionID, class int) bool { return flags[class] < 0 || w.regionHas(g, flags[class]) }
}

// wireSlots fills the slots of the modules built before this one: the headroom of a
// completion (hook H-10) and the gate of the store classes (spec-storage.md 3.2.3). A
// restore calls it too, after allocation, so a restored world holds the slots the
// uninterrupted one holds (N-462); construction calls it after the initial agents, which the
// gate would otherwise refuse before lockStores counts them. It does nothing without the module (the wrap of the institutions
// module is then wired by allocation).
func (w *World) wireSlots() {
	if w.set == nil {
		return
	}
	if w.skills != nil {
		w.skills.headroom = w.roleHeadroom
	}
	if w.sto != nil {
		w.sto.gate = w.storeClassGate()
	}
	w.wireOrgHeadroom() // last, so the least of the two limits wraps the slot just set
}

// wireSeams takes from the initial agents the stores a class lacks, once the classes of
// the regions exist: the stores of the initial agents a class lacks are not given, and
// each such agent's AgentSpawned says how many (spec-storage.md 3.2.3), after wireSlots.
func (b *builder) wireSeams() {
	b.w.wireSlots()
	if b.w.set != nil && b.w.sto != nil {
		b.lockStores()
	}
}

// lockStores takes from each initial agent the stores of its role's row whose class
// the region lacks the flag of, and records the count for its AgentSpawned. Nothing
// has moved yet, no room is set (the storage module defers it) and no event exists.
func (b *builder) lockStores() {
	w := b.w
	s := w.sto
	if s.gate == nil {
		return
	}
	for i := w.firstHousehold; i < len(w.agents); i++ {
		a := &w.agents[i]
		var locked int64
		for _, c := range s.given[a.role] {
			if !s.gate(w.regionOf(a), c) {
				s.take(a.id, c)
				locked++
			}
		}
		if k := i - w.firstHousehold; k < len(b.spawned) {
			b.spawned[k].StoresLocked = locked
		}
	}
}
