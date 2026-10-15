package economysim

// This file holds the recipe table: the recipes of a configuration in typed-ID
// form, and the index of the recipes that make a commodity (spec-production.md
// 3.1.1, 3.1.2, 3.2.2, 9.3). The structs are in config.go. The table is derived
// at construction and is not state: it is not hashed and not checkpointed, and
// a restore rebuilds it. Duration, Sited, TraineeEligible and TraineeYield stay
// in the configuration and are read there by RecipeID, so there is no second
// copy to drift.

// itemRow is one (commodity, quantity) pair by typed ID: a recipe input, or an
// entry of an upkeep list.
type itemRow struct {
	commodity CommodityID
	quantity  Quantity
}

// outputRow is one output of a recipe by typed ID. costShare is in basis
// points; discard says the output is created and disposed of in one step.
type outputRow struct {
	commodity CommodityID
	quantity  Quantity
	costShare int64
	discard   bool
}

// toolRow is one tool of a recipe by typed ID: the wear units one execution
// puts on it.
type toolRow struct {
	commodity  CommodityID
	wearPerUse int64
}

// recipeRow is one recipe. Every list is in canonical commodity order
// (ascending CommodityID). source is true when the recipe has no inputs
// (spec-production.md 3.1.2).
type recipeRow struct {
	inputs  []itemRow
	outputs []outputRow
	tools   []toolRow
	source  bool
}

// recipeTable holds one row per recipe, indexed by RecipeID, and byOutput, the
// makers of each commodity indexed by CommodityID.
type recipeTable struct {
	rows     []recipeRow
	byOutput [][]RecipeID
}

// newRecipeTable builds the table from a canonical configuration that passed
// Validate. A reference that does not resolve is a programmer error and
// panics. byOutput lists, for each commodity, the recipes with it among their
// non-discard outputs, in ascending RecipeID (spec-production.md 9.3); a
// byproduct is an output like any other (3.2.2).
func newRecipeTable(c *Config) recipeTable {
	t := recipeTable{
		rows:     make([]recipeRow, len(c.Recipes)),
		byOutput: make([][]RecipeID, len(c.Commodities)),
	}
	for i := range c.Recipes {
		r := &c.Recipes[i]
		id := RecipeID(i)
		row := &t.rows[i]
		row.source = len(r.Inputs) == 0
		row.inputs = make([]itemRow, len(r.Inputs))
		for j, in := range r.Inputs {
			row.inputs[j] = itemRow{commodity: recipeCommodity(c, r.ID, in.Commodity), quantity: in.Quantity}
		}
		row.outputs = make([]outputRow, len(r.Outputs))
		for j, o := range r.Outputs {
			cid := recipeCommodity(c, r.ID, o.Commodity)
			row.outputs[j] = outputRow{commodity: cid, quantity: o.Quantity, costShare: o.CostShare, discard: o.Discard}
			if o.Discard {
				continue
			}
			// Recipes are visited in ascending ID, so appending keeps the
			// makers ascending; a repeated output names the recipe once.
			if m := t.byOutput[cid]; len(m) == 0 || m[len(m)-1] != id {
				t.byOutput[cid] = append(m, id)
			}
		}
		row.tools = make([]toolRow, len(r.Requires))
		for j, tl := range r.Requires {
			row.tools[j] = toolRow{commodity: recipeCommodity(c, r.ID, tl.Commodity), wearPerUse: tl.WearPerUse}
		}
	}
	return t
}

// recipeCommodity resolves a commodity string of a recipe or panics.
func recipeCommodity(c *Config, recipe, commodity string) CommodityID {
	id, ok := c.commodityID(commodity)
	if !ok {
		panic("economysim: invariant violated: recipe " + recipe + " names unknown commodity " + commodity + " after validation")
	}
	return id
}

// row returns the row of a recipe. It allocates nothing.
func (t *recipeTable) row(id RecipeID) *recipeRow { return &t.rows[id] }

// makers returns the recipes that make a commodity as a non-discard output, in
// ascending RecipeID; the list is empty when no recipe makes it. The caller
// must not change it.
func (t *recipeTable) makers(id CommodityID) []RecipeID { return t.byOutput[id] }
