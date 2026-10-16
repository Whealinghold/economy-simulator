package economysim

// This file is the test-only seam of the root package (architecture.md 10.2):
// it reaches the unexported tables from the external test package, so nothing
// is exported from production code. Each row of Stage 1 appends its own
// aliases and wrappers below; none replaces the file.

// Canonical returns a deep copy of c in canonical order, which the table
// constructors take (config.go canonical).
func Canonical(c Config) Config { return c.canonical() }

// DeriveCalendar exposes the derivation of calendar.go.
func DeriveCalendar(c Calendar) (days, ticks int64, err error) { return c.derive() }

// DayTable is the day-of-year table of calendar.go.
type DayTable = dayTable

// NewDayTable builds the day-of-year table of a calendar.
func NewDayTable(c Calendar) DayTable { return newDayTable(c) }

// DayTableDay reads the table at a zero-based day of the year.
func DayTableDay(t DayTable, dayOfYear int64) (month int, dayOfMonth int64, season int) {
	return t.day(dayOfYear)
}

// RecipeTable is the recipe table of recipe.go.
type RecipeTable = recipeTable

// NewRecipeTable builds the recipe table of a canonical configuration.
func NewRecipeTable(c *Config) RecipeTable { return newRecipeTable(c) }

// RecipeMakers reads the makers of a commodity.
func RecipeMakers(t *RecipeTable, id CommodityID) []RecipeID { return t.makers(id) }

// RecipeRowView is a recipe row flattened to primitives, for the external
// test, so the unexported row types need no alias.
type RecipeRowView struct {
	Inputs, Tools, Outputs []int
	Source                 bool
}

// RecipeRow flattens the row of a recipe to the commodity IDs of its lists.
func RecipeRow(t *RecipeTable, id RecipeID) RecipeRowView {
	r := t.row(id)
	var v RecipeRowView
	for _, in := range r.inputs {
		v.Inputs = append(v.Inputs, int(in.commodity))
	}
	for _, tl := range r.tools {
		v.Tools = append(v.Tools, int(tl.commodity))
	}
	for _, o := range r.outputs {
		v.Outputs = append(v.Outputs, int(o.commodity))
	}
	v.Source = r.source
	return v
}

// RoleTable is the role table of role.go.
type RoleTable = roleTable

// NewRoleTable builds the role table of a canonical configuration.
func NewRoleTable(c *Config) RoleTable { return newRoleTable(c) }

// TradableSets returns the validator's tradable sets of a canonical
// configuration, the one implementation the role table must agree with.
func TradableSets(c *Config) []map[string]bool { return tradableSets(c, newLookup(c)) }

// RoleRowView is a role row flattened to primitives for the external test.
type RoleRowView struct {
	Recipes, Trades, Tradable []int
	Member                    []bool
}

// RoleRow flattens the row of a role.
func RoleRow(t *RoleTable, id RoleID) RoleRowView {
	r := t.row(id)
	var v RoleRowView
	for _, x := range r.recipes {
		v.Recipes = append(v.Recipes, int(x))
	}
	for _, x := range r.trades {
		v.Trades = append(v.Trades, int(x))
	}
	for _, x := range r.tradable {
		v.Tradable = append(v.Tradable, int(x))
	}
	v.Member = r.member
	return v
}

// RoleTradableSomewhere reads the union flag of a commodity.
func RoleTradableSomewhere(t *RoleTable, c CommodityID) bool { return t.tradableSomewhere(c) }

// BeliefParams exposes the construction of the belief parameters of beliefs.go
// as a flat view, so the external test needs no internal type.
func BeliefParams(w World) (flat [16]int64) {
	p := beliefParams(w)
	return [16]int64{p.Step, p.AdjustProb, p.Significance, p.EvidenceShare, p.StockLow, p.StockHigh,
		p.MaxAge, p.WidthInit, p.WidthMin, p.WidthMax, p.WidthStep, p.CostFloor, p.CostCeiling,
		p.RefBandLow, p.RefBandHigh, p.MaxPrice}
}
