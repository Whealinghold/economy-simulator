package economysim

import (
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// This file is the validation of the storage module (spec-storage.md 13; the
// continuation file of storage.go, architecture.md 2.6): the rules of 13.1 that
// Validate checks and the warnings of 13.2. The rule room_below_warm_job is checked
// at construction, not here (FR-STO-045).

// validateStorage checks the storage module (spec-storage.md 13.1, FR-STO-038). It
// reports nothing when the module is off. The rules run in phases and a phase that
// finds an error ends the run, so the first error is the same for the same
// mistake: the classes (bounds, ID, shape), the base room, the role rows (role,
// duplicates, base, stores), then the room rules, which read the rest.
func (v *validator) validateStorage() {
	m := v.c.Modules.Storage
	if m == nil {
		return
	}
	base := []string{"modules", "storage"}
	steps := []func() bool{
		func() bool { return v.storageClasses(base, m) },
		func() bool { return v.storageBase(path(base, "base"), m.Base) },
		func() bool { return v.storageRows(base, m) },
		func() bool { v.storageRoomRules(base, m); return true },
	}
	for _, step := range steps {
		if !step() {
			return
		}
	}
}

// storageClasses checks the count of classes, then each class: ID, duplicate,
// capacities, Needs and label.
func (v *validator) storageClasses(base []string, m *Storage) bool {
	if len(m.Classes) > MaxStoreClasses {
		v.rep.add(ErrOutOfBounds, path(base, "classes"), itoa(len(m.Classes)), itoa(MaxStoreClasses))
		return false
	}
	n := len(v.rep.errs)
	seen := make(map[string]bool, len(m.Classes)) // a membership test only
	for i, c := range m.Classes {
		p := path(base, "classes["+itoa(i)+"]")
		switch {
		case !validID(c.ID):
			v.rep.add(ErrInvalidID, path(p, "id"), c.ID, "")
		case seen[c.ID]:
			v.rep.add(ErrDuplicateID, path(p, "id"), c.ID, "")
		}
		seen[c.ID] = true
		v.storageClassShape(p, c)
	}
	return len(v.rep.errs) == n
}

// storageClassShape is the shape of one class: each capacity in 0 .. MaxRoom, not
// both 0, a Needs flag only with a settlement module (which is not built), a label
// within its bound.
func (v *validator) storageClassShape(p []string, c StoreClass) {
	inside := true
	for _, f := range []fieldRange{{"mass", c.Mass, 0, MaxRoom}, {"bulk", c.Bulk, 0, MaxRoom}} {
		if f.value < f.lo || f.value > f.hi {
			v.rep.add(ErrStoreShape, path(p, f.name), i64(f.value), i64(f.hi))
			inside = false
		}
	}
	if inside && c.Mass == 0 && c.Bulk == 0 {
		v.rep.add(ErrStoreShape, p, "0", "1")
	}
	if c.Needs != "" {
		v.rep.add(ErrModuleDependency, path(p, "needs"), c.Needs, "settlements")
	}
	v.label(p, "label", c.Label)
}

// storageBase checks a base room: each dimension in 0 .. MaxRoom. A nil base is
// unlimited and is fine.
func (v *validator) storageBase(p []string, b *Capacity) bool {
	if b == nil {
		return true
	}
	n := len(v.rep.errs)
	v.ranges(p, []fieldRange{{"mass", b.Mass, 0, MaxRoom}, {"bulk", b.Bulk, 0, MaxRoom}})
	return len(v.rep.errs) == n
}

// storageRows checks the role rows: a known role, one row per role, the base, and
// the stores (count, class, a limited role).
func (v *validator) storageRows(base []string, m *Storage) bool {
	n := len(v.rep.errs)
	seen := make(map[string]bool, len(m.Roles)) // a membership test only
	classes := m.classSet()
	for i, r := range m.Roles {
		p := path(base, "roles["+itoa(i)+"]")
		switch {
		case !v.role[r.Role]:
			v.rep.add(ErrUnknownRole, path(p, "role"), r.Role, "")
		case seen[r.Role]:
			v.rep.add(ErrDuplicateID, path(p, "role"), r.Role, "")
		}
		seen[r.Role] = true
		v.storageBase(path(p, "base"), r.Base)
		v.storageStores(p, r, classes)
	}
	return len(v.rep.errs) == n
}

// storageStores checks the Stores of a row: at most MaxStoresPerAgent, each a
// configured class, and none on a role whose base room is nil (the row's own).
func (v *validator) storageStores(p []string, r RoleRoom, classes map[string]bool) {
	if len(r.Stores) > MaxStoresPerAgent {
		v.rep.add(ErrOutOfBounds, path(p, "stores"), itoa(len(r.Stores)), itoa(MaxStoresPerAgent))
		return
	}
	for j, id := range r.Stores {
		if !classes[id] {
			v.rep.add(ErrUnknownStoreClass, path(p, "stores["+itoa(j)+"]"), id, "")
		}
	}
	if len(r.Stores) > 0 && r.Base == nil {
		v.rep.add(ErrNotLimited, path(p, "stores"), itoa(len(r.Stores)), "0")
	}
}

// storageBasePath is the path of the base room that Storage.baseOf resolved.
func storageBasePath(row int) []string {
	if row < 0 {
		return []string{"modules", "storage", "base"}
	}
	return []string{"modules", "storage", "roles[" + itoa(row) + "]", "base"}
}

// roomLoad is the load of a set of item lists: the sum of quantity times Mass and
// of quantity times Bulk (spec-goods.md 3.2), with checked arithmetic. A commodity
// that is not declared adds nothing (another rule reports it). It reports false
// when a sum does not fit in int64.
func roomLoad(lk lookup, lists ...[]Item) (mass, bulk int64, ok bool) {
	for _, list := range lists {
		for _, it := range list {
			cm := lk.commodity[it.Commodity]
			if cm == nil {
				continue
			}
			dm, ok1 := checked.Mul(int64(it.Quantity), cm.Mass)
			db, ok2 := checked.Mul(int64(it.Quantity), cm.Bulk)
			sm, ok3 := checked.Add(mass, dm)
			sb, ok4 := checked.Add(bulk, db)
			if !ok1 || !ok2 || !ok3 || !ok4 {
				return 0, 0, false
			}
			mass, bulk = sm, sb
		}
	}
	return mass, bulk, true
}

// storageRoomRules checks room_below_upkeep and room_below_start_stock for every
// limited role (spec-storage.md 13.1). It runs on a module that passed the
// shape rules, so every class and role it names exists.
func (v *validator) storageRoomRules(base []string, m *Storage) {
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		room, row := m.baseOf(ro.ID)
		if room == nil {
			continue
		}
		p := storageBasePath(row)
		v.roomBelowUpkeep(p, ro, room)
		v.roomBelowStartStock(p, m, ro, row, room)
	}
}

// roomBelowUpkeep is the rule room_below_upkeep: the base room per household is at
// least the load of one tick's effective upkeep at the largest upkeep of any
// region (spec-storage.md 13.1), so a household could hold what it eats.
func (v *validator) roomBelowUpkeep(p []string, ro *Role, room *Capacity) {
	mass, bulk := regionsUpkeepLoad(&v.c, v.lk, ro)
	if mass > room.Mass {
		v.rep.add(ErrRoomShape, path(p, "mass"), i64(room.Mass), i64(mass))
	}
	if bulk > room.Bulk {
		v.rep.add(ErrRoomShape, path(p, "bulk"), i64(room.Bulk), i64(bulk))
	}
}

// roomBelowStartStock is the rule room_below_start_stock: the StartStock per
// household of a limited role, with the largest weight of any group of the role,
// fits the base room of the cohort plus the stores its row gives (a store is not
// multiplied by the weight, spec-storage.md 3.3.2). The stock of a warm-start group
// is checked at construction (plan 3-sto D10), not here. Every class that a row
// gives is given here: a class that needs a settlement flag is not configurable
// yet.
func (v *validator) roomBelowStartStock(p []string, m *Storage, ro *Role, row int, room *Capacity) {
	w := v.largestWeight(ro.ID)
	if w == 0 {
		return
	}
	capMass, capBulk, ok := cohortRoom(m, row, w, room)
	if !ok {
		return
	}
	mass, bulk, ok := roomLoad(v.lk, ro.StartStock)
	massW, ok1 := checked.Mul(mass, w)
	bulkW, ok2 := checked.Mul(bulk, w)
	if !ok || !ok1 || !ok2 || massW > capMass || bulkW > capBulk {
		v.rep.add(ErrRoomShape, p, "start_stock", "room")
	}
}

// cohortRoom is the room of a cohort of weight w of the role of a row: w times the
// base room plus the capacity of the stores of the row. It is false when a figure
// does not fit in int64, which MaxRoom and MaxWeight rule out for a valid module.
func cohortRoom(m *Storage, row int, w int64, room *Capacity) (mass, bulk int64, ok bool) {
	mass, ok1 := checked.Mul(room.Mass, w)
	bulk, ok2 := checked.Mul(room.Bulk, w)
	if !ok1 || !ok2 {
		return 0, 0, false
	}
	if row < 0 {
		return mass, bulk, true
	}
	for _, id := range m.Roles[row].Stores {
		i := slices.IndexFunc(m.Classes, func(c StoreClass) bool { return c.ID == id })
		if i < 0 {
			continue
		}
		mass, ok1 = checked.Add(mass, m.Classes[i].Mass)
		bulk, ok2 = checked.Add(bulk, m.Classes[i].Bulk)
		if !ok1 || !ok2 {
			return 0, 0, false
		}
	}
	return mass, bulk, true
}

// largestWeight is the largest weight of any group of the role: the Scale of a
// group, or the world's Scale when it gives none (spec-scale.md 3.1). It is 0 for
// a role with no group.
func (v *validator) largestWeight(role string) int64 {
	var w int64
	for _, g := range v.c.Groups {
		if g.Role != role {
			continue
		}
		s := g.Scale
		if s == 0 {
			s = v.c.World.Scale
		}
		w = max(w, s)
	}
	return w
}

// storeKeyKind is the rule that the Store of a target means something only for the
// kind stores: a Store on another kind is ErrFlowShape.
func (v *validator) storeKeyKind(tp []string, t FlowTarget) {
	if t.Store != "" && t.Kind != targetStores {
		v.rep.add(ErrFlowShape, path(tp, "store"), t.Store, "")
	}
}

// unbuiltTarget checks a target of a kind that is recognised and not built: the
// stores kind has its rules (storesTarget), the treasury kind is refused until its
// modules are built (plan 2-6 D320).
func (v *validator) unbuiltTarget(tp []string, f MoneyFlow, fp []string) {
	if f.Target.Kind == targetStores {
		v.storesTarget(tp, f, fp)
		return
	}
	v.rep.add(ErrModuleDependency, path(tp, "kind"), f.Target.Kind, "")
}

// storesTarget checks a flow whose target is the kind stores (spec-money-supply.md
// 4.3.4, 12.1): the storage module must be on, the target has no role, the class is
// configured, the region is defined, and the shape is a rate grant or a fixed levy
// (no Level, no RateBP). The runtime of the kind is not built yet, so a flow that
// passes every other rule is still ErrModuleDependency, with the bound "built":
// the plan of the module removes that last rule with the flow's step (plan 3-sto
// Phase 6), because resolveTarget has no role to give such a flow.
func (v *validator) storesTarget(tp []string, f MoneyFlow, fp []string) {
	t := f.Target
	if v.c.Modules.Storage == nil {
		v.rep.add(ErrModuleDependency, path(tp, "kind"), t.Kind, "")
		return
	}
	n := len(v.rep.errs)
	if t.Role != "" {
		v.rep.add(ErrFlowShape, path(tp, "role"), t.Role, "")
	}
	if f.Level != 0 {
		v.rep.add(ErrFlowShape, path(fp, "level"), "", "")
	}
	if f.RateBP != 0 {
		v.rep.add(ErrFlowShape, path(fp, "rate_bp"), "", "")
	}
	if t.Store != "" && !v.c.Modules.Storage.classSet()[t.Store] {
		v.rep.add(ErrUnknownStoreClass, path(tp, "store"), t.Store, "")
	}
	if t.Region != "" && !v.region[t.Region] {
		v.rep.add(ErrUnknownRegion, path(tp, "region"), t.Region, "")
	}
	if len(v.rep.errs) == n {
		v.rep.add(ErrModuleDependency, path(tp, "kind"), t.Kind, "built")
	}
}

// storageWarnings are the warnings of spec-storage.md 13.2 that need no other
// module: no_limited_role, room_below_cover, room_below_recipe, store_class_unused
// and store_levy_below_upkeep. store_locked_at_start waits for the settlements
// module. A warning never fails validation and is safe on a configuration that
// does not validate: an undeclared ID is skipped.
func storageWarnings(c *Config) []Warning {
	m := c.Modules.Storage
	if m == nil {
		return nil
	}
	lk := newLookup(c)
	var out []Warning
	limited := false
	for i := range c.Roles {
		ro := &c.Roles[i]
		room, row := m.baseOf(ro.ID)
		if room == nil {
			continue
		}
		limited = true
		out = append(out, roomWarnings(c, lk, m, ro, row, room)...)
	}
	if !limited {
		out = append(out, Warning{Path: []string{"modules", "storage"}, Rule: "no_limited_role"})
	}
	out = append(out, unusedClassWarnings(c, m)...)
	return append(out, storeLevyWarnings(c, lk, m)...)
}

// roomWarnings are room_below_cover and room_below_recipe for one limited role.
func roomWarnings(c *Config, lk lookup, m *Storage, ro *Role, row int, room *Capacity) []Warning {
	var out []Warning
	p := storageBasePath(row)
	upMass, upBulk := regionsUpkeepLoad(c, lk, ro)
	cover, ok1 := checked.Mul(upMass, c.World.CoverTicks)
	coverB, ok2 := checked.Mul(upBulk, c.World.CoverTicks)
	if !ok1 || !ok2 || cover > room.Mass || coverB > room.Bulk {
		out = append(out, Warning{Path: p, Rule: "room_below_cover"})
	}
	return append(out, recipeWarnings(lk, m, ro, row, room, p, upMass, upBulk)...)
}

// recipeWarnings is room_below_recipe for one limited role: a recipe whose
// non-discard outputs, with one tick's upkeep, do not fit the room of one household
// and its stores. The sums are checked, and an overflow does not fit.
func recipeWarnings(lk lookup, m *Storage, ro *Role, row int, room *Capacity, p []string, upMass, upBulk int64) []Warning {
	capMass, capBulk, ok := cohortRoom(m, row, 1, room)
	if !ok {
		return nil
	}
	var out []Warning
	for _, id := range ro.Recipes {
		rc := lk.recipe[id]
		if rc == nil {
			continue
		}
		oMass, oBulk := outputLoad(lk, rc)
		sumMass, ok1 := checked.Add(upMass, oMass)
		sumBulk, ok2 := checked.Add(upBulk, oBulk)
		if !ok1 || !ok2 || sumMass > capMass || sumBulk > capBulk {
			out = append(out, Warning{Path: append(slices.Clone(p), "recipe", id), Rule: "room_below_recipe"})
		}
	}
	return out
}

// regionsUpkeepLoad is the largest, over the regions, of the mass and of the bulk
// load of one tick's effective upkeep of a household of the role. An overflow is
// reported as the largest int64.
func regionsUpkeepLoad(c *Config, lk lookup, ro *Role) (mass, bulk int64) {
	for i := range c.Regions {
		m, b, ok := roomLoad(lk, effectiveUpkeep(c, ro), c.Regions[i].ExtraUpkeep)
		if !ok {
			return math.MaxInt64, math.MaxInt64
		}
		mass, bulk = max(mass, m), max(bulk, b)
	}
	return mass, bulk
}

// outputLoad is the load of the non-discard outputs of one execution of a recipe
// at an effectiveness of 10 000 basis points (spec-storage.md 13.2). An overflow is
// reported as the largest int64.
func outputLoad(lk lookup, rc *Recipe) (mass, bulk int64) {
	var items []Item
	for _, o := range rc.Outputs {
		if !o.Discard {
			items = append(items, Item{Commodity: o.Commodity, Quantity: o.Quantity})
		}
	}
	mass, bulk, ok := roomLoad(lk, items)
	if !ok {
		return math.MaxInt64, math.MaxInt64
	}
	return mass, bulk
}

// unusedClassWarnings is store_class_unused: a class that no role row and no flow
// names.
func unusedClassWarnings(c *Config, m *Storage) []Warning {
	var out []Warning
	for i, cl := range m.Classes {
		used := slices.ContainsFunc(m.Roles, func(r RoleRoom) bool { return slices.Contains(r.Stores, cl.ID) })
		if !used && c.Modules.MoneySupply != nil {
			used = slices.ContainsFunc(c.Modules.MoneySupply.Flows, func(f MoneyFlow) bool {
				return f.Target.Kind == targetStores && (f.Target.Store == "" || f.Target.Store == cl.ID)
			})
		}
		if !used {
			out = append(out, Warning{Path: []string{"modules", "storage", "classes[" + itoa(i) + "]"}, Rule: "store_class_unused"})
		}
	}
	return out
}

// storeLevyWarnings is store_levy_below_upkeep: a levy that targets stores has an
// Exempt below one tick's upkeep cost, at the reference prices, of a limited role
// that has a group in a region the target covers (spec-storage.md 13.2).
func storeLevyWarnings(c *Config, lk lookup, m *Storage) []Warning {
	if c.Modules.MoneySupply == nil {
		return nil
	}
	var out []Warning
	for _, f := range c.Modules.MoneySupply.Flows {
		if f.Kind != FlowLevy || f.Target.Kind != targetStores {
			continue
		}
		if storeLevyBelow(c, lk, m, f) {
			out = append(out, Warning{Path: []string{"modules", "money_supply", "flows", f.ID}, Rule: "store_levy_below_upkeep"})
		}
	}
	return out
}

// storeLevyBelow reports whether the Exempt of a stores levy is below the upkeep
// cost of a limited role in a region of one of its groups.
func storeLevyBelow(c *Config, lk lookup, m *Storage, f MoneyFlow) bool {
	for _, g := range c.Groups {
		rg, ro := lk.region[g.Region], lk.role[g.Role]
		if rg == nil || ro == nil || (f.Target.Region != "" && f.Target.Region != g.Region) {
			continue
		}
		if room, _ := m.baseOf(ro.ID); room == nil {
			continue
		}
		if cost, ok := listCost(lk, g.Region, effectiveUpkeep(c, ro), rg.ExtraUpkeep); ok && f.Exempt < cost {
			return true
		}
	}
	return false
}
