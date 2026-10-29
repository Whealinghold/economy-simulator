package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// This file holds the role table: the roles of a configuration in typed-ID
// form, with the tradable set of each and its effective upkeep per region
// (spec-agents.md 3.3, 3.4, 5.2; spec-goods.md 6.4). The structs are in
// config.go. The table is derived at construction and is not state: it is not
// hashed and not checkpointed, and a restore rebuilds it. The tradable set and
// the base upkeep are the ones validate.go computes (tradableSets and
// effectiveUpkeep), so the validator and the world cannot disagree.

// regionUpkeep is the effective upkeep of a role in one region that has an
// extra upkeep: the base list merged with the region's, per household.
type regionUpkeep struct {
	region RegionID
	items  []itemRow
}

// roleRow is one role. recipes, trades, upkeep and tradable are in ascending
// ID order. member[c] says whether commodity c is in the tradable set, and
// regionUpkeep is ascending by region and holds a row only for a region with
// an ExtraUpkeep.
type roleRow struct {
	recipes []RecipeID
	// ownUse is the effective own-use list of the role, ascending, and jobs the
	// ascending union of recipes and ownUse (spec-subsistence.md 3.3, plan 2-10
	// D404): a job request is admitted against jobs, the view and the cost anchor
	// read recipes. With the module off ownUse is nil and jobs is recipes itself.
	ownUse       []RecipeID
	jobs         []RecipeID
	trades       []CommodityID
	upkeep       []itemRow
	tradable     []CommodityID
	member       []bool
	regionUpkeep []regionUpkeep
	// window is the effective welfare window of the role, the one
	// effectiveWindow computes for the validator (spec-welfare-and-wages.md
	// 2.3.3). Appended by the welfare row; modules join L inside that function.
	window int64
}

// roleTable holds one row per role, indexed by RoleID, and anyTradable, which
// says for each commodity whether it is in the tradable set of some role.
type roleTable struct {
	rows        []roleRow
	anyTradable []bool
}

// newRoleTable builds the table from a canonical configuration that passed
// Validate. A reference that does not resolve, or a merged upkeep quantity
// that overflows, is a programmer error and panics with a message that says so.
//
// The tradable set of a role is built in a local slice from the core terms.
// An optional module that adds items to it appends them where marked below,
// before the set is sorted and stored; the table is never written after this
// function returns, which is all "frozen" means (note N-228).
func newRoleTable(c *Config) roleTable {
	lk := newLookup(c)
	sets := tradableSets(c, lk)
	t := roleTable{
		rows:        make([]roleRow, len(c.Roles)),
		anyTradable: make([]bool, len(c.Commodities)),
	}
	for i := range c.Roles {
		r := &c.Roles[i]
		row := &t.rows[i]
		for _, id := range r.Recipes {
			rid, ok := c.recipeID(id)
			if !ok {
				// Not covered by a test: validation rejects an unknown recipe before this
				// runs, so no valid configuration reaches the panic (testing.md 9.5).
				panic("economysim: invariant violated: role " + r.ID + " names unknown recipe " + id + " after validation")
			}
			row.recipes = append(row.recipes, rid)
		}
		slices.Sort(row.recipes)
		row.ownUse, row.jobs = c.Modules.Subsistence.ownUseJobs(c, r.ID, row.recipes) // the module's join; off: nil and recipes
		w, ok := effectiveWindow(c, lk, r)
		if !ok {
			// Not covered by a test, for the same reason: the validator refuses a
			// window field or a Duration outside its bounds, which is when ok is false.
			panic("economysim: invariant violated: role " + r.ID + " has no effective welfare window after validation")
		}
		row.window = w
		for _, id := range r.Trades {
			row.trades = append(row.trades, roleCommodity(c, r.ID, id))
		}
		slices.Sort(row.trades)

		base := effectiveUpkeep(c, r)
		row.upkeep = mergeUpkeep(c, r.ID, nil, base)

		for id := range sets[i] { // map order is erased by the sort below
			row.tradable = append(row.tradable, roleCommodity(c, r.ID, id))
		}
		// A module that is on appends its items to row.tradable here, in the
		// order of the Modules pointers (none exist in Stage 1).
		slices.Sort(row.tradable)
		row.tradable = slices.Compact(row.tradable)
		row.member = make([]bool, len(c.Commodities))
		for _, id := range row.tradable {
			row.member[id] = true
			t.anyTradable[id] = true
		}

		for g := range c.Regions {
			if len(c.Regions[g].ExtraUpkeep) == 0 {
				continue
			}
			row.regionUpkeep = append(row.regionUpkeep, regionUpkeep{
				region: RegionID(g),
				items:  mergeUpkeep(c, r.ID, row.upkeep, c.Regions[g].ExtraUpkeep),
			})
		}
	}
	return t
}

// roleCommodity resolves a commodity string of a role or panics.
func roleCommodity(c *Config, role, commodity string) CommodityID {
	id, ok := c.commodityID(commodity)
	if !ok {
		panic("economysim: invariant violated: role " + role + " names unknown commodity " + commodity + " after validation")
	}
	return id
}

// mergeUpkeep returns base plus the items added, quantities of one commodity
// summed with checked arithmetic, in ascending commodity order. A sum that
// overflows is a programmer error after validation and panics.
func mergeUpkeep(c *Config, role string, base []itemRow, add []Item) []itemRow {
	out := slices.Clone(base)
	for _, it := range add {
		out = append(out, itemRow{commodity: roleCommodity(c, role, it.Commodity), quantity: it.Quantity})
	}
	slices.SortStableFunc(out, func(a, b itemRow) int { return int(a.commodity) - int(b.commodity) })
	merged := out[:0]
	for _, it := range out {
		if n := len(merged); n > 0 && merged[n-1].commodity == it.commodity {
			sum, ok := checked.Add(int64(merged[n-1].quantity), int64(it.quantity))
			if !ok {
				panic("economysim: invariant violated: upkeep of role " + role + " overflows after validation")
			}
			merged[n-1].quantity = Quantity(sum)
			continue
		}
		merged = append(merged, it)
	}
	return slices.Clip(merged)
}

// row returns the row of a role. It allocates nothing.
func (t *roleTable) row(id RoleID) *roleRow { return &t.rows[id] }

// isTradable reports whether commodity c is in the role's tradable set. A
// commodity out of range is not.
func (r *roleRow) isTradable(c CommodityID) bool {
	return int(c) >= 0 && int(c) < len(r.member) && r.member[c]
}

// upkeepIn returns the effective upkeep of one household of the role in a
// region: the merged row when the region has an extra upkeep, else the base
// list. The caller must not change it. It allocates nothing.
func (r *roleRow) upkeepIn(g RegionID) []itemRow {
	lo, hi := 0, len(r.regionUpkeep)
	for lo < hi {
		m := int(uint(lo+hi) >> 1)
		if r.regionUpkeep[m].region < g {
			lo = m + 1
		} else {
			hi = m
		}
	}
	if lo < len(r.regionUpkeep) && r.regionUpkeep[lo].region == g {
		return r.regionUpkeep[lo].items
	}
	return r.upkeep
}

// tradableSomewhere reports whether commodity c is in the tradable set of at
// least one role (spec-agents.md 3.4, spec-goods.md 6.4).
func (t *roleTable) tradableSomewhere(c CommodityID) bool {
	return int(c) >= 0 && int(c) < len(t.anyTradable) && t.anyTradable[c]
}
