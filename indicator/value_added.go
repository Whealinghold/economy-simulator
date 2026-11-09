package indicator

import (
	"cmp"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// ClassRow is a value for a class of commodities in a region. The class "" is
// every commodity once (a commodity of two classes is in each class's row and
// once in the all row, spec-indicators.md 4.4.1).
type ClassRow struct {
	Region economysim.RegionID
	Class  string
	Value  int64
}

// RecipeRow is the job value added of a recipe in a region on the completion
// basis: the value of the units its executions created, gross of what was
// disposed of, less the value of the inputs of those executions
// (spec-indicators.md 7.2.4).
type RecipeRow struct {
	Region     economysim.RegionID
	Recipe     economysim.RecipeID
	Executions int64
	Outputs    int64
	Inputs     int64
	Value      int64
}

// ValueAddedRegion reconciles the by-recipe table of a region with its GDP by
// the production approach: Completed is the sum of its by-recipe values (jobs
// completed), Accrual the value added of 7.2.1 (jobs started) and WIP the named
// difference, Sum p * (In - InCompleted - dPend). Mismatch is Completed -
// Accrual - WIP, which is 0 when the recipe events and the boundary flows
// agree.
type ValueAddedRegion struct {
	Region                            economysim.RegionID
	Completed, Accrual, WIP, Mismatch int64
}

// ValueAddedResult is the output and intermediate use by class and the value
// added by recipe, with the WIP row (spec-indicators.md 9.1, row 2). A class has
// no value added of its own from output and input (7.2.3): ByClass is the
// by-recipe table grouped by the cost shares of the outputs.
type ValueAddedResult struct {
	ConfigHash           [32]byte
	From, To             int64
	Output, Intermediate []ClassRow
	ByRecipe             []RecipeRow
	ByClass              []ClassRow
	Regions              []ValueAddedRegion
}

// mulChecked returns a * b and false when it does not fit int64.
func mulChecked(a, b int64) (int64, bool) { return measure.ToInt64(measure.MulInt64(a, b)) }

// ValueAdded returns the output and the intermediate use by class, the value
// added by recipe on the completion basis (grouped to classes by cost share) and
// the WIP row that reconciles it with GDP. cfg is the configuration of the
// window's world as World.Config returns it; open and closing bound the window.
// The value of a discard output is in Outputs (a gross output) and shown as a
// loss by GDP's Written, not hidden.
func ValueAdded(w Window, cfg economysim.Config, open, closing economysim.Snapshot, p Prices, cl Classes) (ValueAddedResult, error) {
	in, err := loadAccounts("ValueAdded", w, open, closing, p)
	if err != nil {
		return ValueAddedResult{}, err
	}
	if len(cfg.Recipes) != w.Recipes || len(cfg.Commodities) != w.Commodities {
		return ValueAddedResult{}, wrap("ValueAdded", "config", ErrConfigMismatch)
	}
	res := ValueAddedResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	if res.Output, err = classValues("ValueAdded", w, in.u.out, p, cl); err != nil {
		return ValueAddedResult{}, err
	}
	if res.Intermediate, err = classValues("ValueAdded", w, in.u.in, p, cl); err != nil {
		return ValueAddedResult{}, err
	}
	ids := commodityIDs(cfg)
	for r := 0; r < w.Regions; r++ {
		if err := res.addRegion(in, w, cfg, ids, economysim.RegionID(r), p, cl); err != nil {
			return ValueAddedResult{}, err
		}
	}
	res.ByClass = collapseClasses(res.ByClass)
	return res, nil
}

// addRegion adds the by-recipe rows, the class rows and the WIP row of a region.
func (res *ValueAddedResult) addRegion(in accounts, w Window, cfg economysim.Config, ids map[string]economysim.CommodityID, region economysim.RegionID, p Prices, cl Classes) error {
	inCompleted := make([]int64, w.Commodities)
	var completed acc
	for k := range cfg.Recipes {
		row, shares, err := recipeRow(w, cfg, ids, region, economysim.RecipeID(k), p, inCompleted)
		if err != nil {
			return err
		}
		if row.Executions == 0 && row.Outputs == 0 {
			continue
		}
		res.ByRecipe = append(res.ByRecipe, row)
		completed.addInt(measure.FromInt64(row.Value))
		res.ByClass = addShares(res.ByClass, region, cl, shares)
	}
	reg, err := in.reconcile(w, region, p, inCompleted, completed)
	res.Regions = append(res.Regions, reg)
	return err
}

// commodityIDs maps a commodity's string ID to its CommodityID. The map is only
// looked up.
func commodityIDs(cfg economysim.Config) map[string]economysim.CommodityID {
	ids := make(map[string]economysim.CommodityID, len(cfg.Commodities))
	for i, c := range cfg.Commodities {
		ids[c.ID] = economysim.CommodityID(i)
	}
	return ids
}

// classValues returns, for the all row and each class, the value of a dense
// table of units at the prices of each region.
func classValues(preset string, w Window, units []int64, p Prices, cl Classes) ([]ClassRow, error) {
	names := append([]string{""}, cl.Names()...)
	var out []ClassRow
	for r := 0; r < w.Regions; r++ {
		for _, name := range names {
			set := cl.All()
			if name != "" {
				var err error
				if set, err = cl.Of(name); err != nil {
					return nil, err
				}
			}
			var a acc
			for _, c := range set {
				a.add(units[r*w.Commodities+int(c)], p.At(economysim.RegionID(r), c))
			}
			v, err := a.int64(preset, "class")
			if err != nil {
				return nil, err
			}
			out = append(out, ClassRow{Region: economysim.RegionID(r), Class: name, Value: v})
		}
	}
	return out, nil
}

// share is what one output of a recipe adds to the value added of its class: its
// value less the input cost allocated to it.
type share struct {
	commodity economysim.CommodityID
	value     int64
}

// outputsOf returns the output rows of a (region, recipe), in commodity order.
func outputsOf(w Window, r economysim.RegionID, k economysim.RecipeID) []OutputTally {
	i, _ := slices.BinarySearchFunc(w.Outputs, OutputTally{Region: r, Recipe: k, Commodity: -1}, outputKey)
	j := i
	for j < len(w.Outputs) && w.Outputs[j].Region == r && w.Outputs[j].Recipe == k {
		j++
	}
	return w.Outputs[i:j]
}

// recipeRow computes the job value added of one recipe in one region, adds the
// inputs of its executions to inCompleted, and returns the shares of its outputs
// for the grouping by class.
func recipeRow(w Window, cfg economysim.Config, ids map[string]economysim.CommodityID, r economysim.RegionID, k economysim.RecipeID, p Prices, inCompleted []int64) (RecipeRow, []share, error) {
	rec := cfg.Recipes[k]
	row := RecipeRow{Region: r, Recipe: k, Executions: w.Recipe[int(r)*w.Recipes+int(k)].Executions}
	var inputs, outputs acc
	for _, it := range rec.Inputs {
		q, ok := mulChecked(row.Executions, int64(it.Quantity))
		if !ok {
			return RecipeRow{}, nil, wrap("ValueAdded", "inputs", ErrOverflow)
		}
		inputs.add(q, p.At(r, ids[it.Commodity]))
		inCompleted[ids[it.Commodity]] += q
	}
	values := map[economysim.CommodityID]int64{} // looked up by the cfg's outputs, never iterated
	for _, o := range outputsOf(w, r, k) {
		outputs.add(o.Output, p.At(r, o.Commodity))
		v, ok := mulChecked(o.Output, int64(p.At(r, o.Commodity)))
		if !ok {
			return RecipeRow{}, nil, wrap("ValueAdded", "outputs", ErrOverflow)
		}
		values[o.Commodity] = v
	}
	var err error
	if row.Inputs, err = inputs.int64("ValueAdded", "inputs"); err != nil {
		return RecipeRow{}, nil, err
	}
	if row.Outputs, err = outputs.int64("ValueAdded", "outputs"); err != nil {
		return RecipeRow{}, nil, err
	}
	row.Value = row.Outputs - row.Inputs
	return row, allocate(rec, ids, values, row.Inputs), nil
}

// allocate splits the input cost of a recipe over its outputs by cost share,
// each rounded half to even, the rounding remainder going to the output with the
// largest share (ties to the lower CommodityID) (spec-indicators.md 7.2.4).
func allocate(rec economysim.Recipe, ids map[string]economysim.CommodityID, values map[economysim.CommodityID]int64, inputs int64) []share {
	out := make([]share, len(rec.Outputs))
	best, left := 0, inputs
	for i, o := range rec.Outputs {
		c := ids[o.Commodity]
		q, _ := measure.RoundDiv(measure.MulInt64(o.CostShare, inputs), measure.FromInt64(10_000))
		a, _ := measure.ToInt64(q)
		out[i] = share{commodity: c, value: values[c] - a}
		left -= a
		if o.CostShare > rec.Outputs[best].CostShare || (o.CostShare == rec.Outputs[best].CostShare && c < ids[rec.Outputs[best].Commodity]) {
			best = i
		}
	}
	if len(out) > 0 {
		out[best].value -= left
	}
	return out
}

// addShares adds the shares of a recipe to the class rows of a region: to the
// all row and to each class the commodity is in.
func addShares(rows []ClassRow, r economysim.RegionID, cl Classes, shares []share) []ClassRow {
	for _, s := range shares {
		rows = append(rows, ClassRow{Region: r, Class: "", Value: s.value})
		for _, name := range cl.Names() {
			if set, err := cl.Of(name); err == nil && slices.Contains(set, s.commodity) {
				rows = append(rows, ClassRow{Region: r, Class: name, Value: s.value})
			}
		}
	}
	return rows
}

// collapseClasses sums the rows of the same (region, class), in region then class
// order.
func collapseClasses(rows []ClassRow) []ClassRow {
	slices.SortStableFunc(rows, func(a, b ClassRow) int {
		return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Class, b.Class))
	})
	var out []ClassRow
	for _, r := range rows {
		if n := len(out); n > 0 && out[n-1].Region == r.Region && out[n-1].Class == r.Class {
			out[n-1].Value += r.Value
			continue
		}
		out = append(out, r)
	}
	return out
}

// reconcile builds the WIP row of a region.
func (in accounts) reconcile(w Window, r economysim.RegionID, p Prices, inCompleted []int64, completed acc) (ValueAddedRegion, error) {
	var accrual, wip acc
	for c := 0; c < w.Commodities; c++ {
		i, pr := int(r)*w.Commodities+c, p.At(r, economysim.CommodityID(c))
		accrual.add(in.u.out[i]-in.u.in[i]+in.st.pend[i]-in.op.pend[i], pr)
		wip.add(in.u.in[i]-inCompleted[c]-(in.st.pend[i]-in.op.pend[i]), pr)
	}
	row := ValueAddedRegion{Region: r}
	var err error
	if row.Completed, err = completed.int64("ValueAdded", "completed"); err != nil {
		return row, err
	}
	if row.Accrual, err = accrual.int64("ValueAdded", "accrual"); err != nil {
		return row, err
	}
	if row.WIP, err = wip.int64("ValueAdded", "wip"); err != nil {
		return row, err
	}
	row.Mismatch = row.Completed - row.Accrual - row.WIP
	return row, nil
}
