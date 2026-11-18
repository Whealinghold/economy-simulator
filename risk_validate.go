package economysim

import "slices"

// This file is the validation of the risk module (spec-risk.md 13; the continuation
// file of risk.go, architecture.md 2.6): the rules of 13.1 that Validate checks and
// the warnings of 13.2.

// riskBase is the path of the module.
var riskBase = []string{"modules", "risk"}

// riskRow is the path of a row of one of the two tables, as the other modules name a
// row: the table and the index in the order the rows were written.
func riskRow(table string, i int) []string {
	return path(riskBase, table+"["+itoa(i)+"]")
}

// validateRisk checks the module by spec-risk.md 13.1 (FR-RSK-003): the count of
// rows of each table, then each row, in the order the rows were written. A row of a
// table that needs a module that is off is reported once, for the table, as
// ErrModuleDependency. It reports nothing when the module is off.
func (v *validator) validateRisk() {
	m := v.c.Modules.Risk
	if m == nil {
		return
	}
	v.riskRoutes(m.Routes)
	v.riskStores(m.Stores)
}

// riskCount reports ErrOutOfBounds for a table with more than MaxRiskRows rows and
// says whether the rows may be read.
func (v *validator) riskCount(table string, n int) bool {
	if n > MaxRiskRows {
		v.rep.add(ErrOutOfBounds, path(riskBase, table), itoa(n), itoa(MaxRiskRows))
		return false
	}
	return true
}

// riskShape checks the two numbers of a row against 1 to 10 000 (rule row_shape).
func (v *validator) riskShape(p []string, strike, severity int64) {
	for _, f := range []fieldRange{{"strike_bp", strike, 1, wholeBP}, {"severity_bp", severity, 1, wholeBP}} {
		if f.value < f.lo || f.value > f.hi {
			v.rep.add(ErrRiskShape, path(p, f.name), i64(f.value), "1 to "+itoa(wholeBP))
		}
	}
}

// riskRegion checks the region of a row: it is defined and not an outside node
// (rules unknown_region and outside_node), and says whether it is a usable region.
func (v *validator) riskRegion(p []string, name, region string, outside map[string]bool) bool {
	switch {
	case !v.region[region]:
		v.rep.add(ErrUnknownRegion, path(p, name), region, "")
		return false
	case outside[region]:
		v.rep.add(ErrOutsideNode, path(p, name), region, "")
		return false
	}
	return true
}

// riskRoutes checks the route rows: the module dependency, the count, and for each
// row the two regions, the edge, the mode, the numbers and the key (rules
// risk_module_dependency, unknown_region, unknown_edge, unknown_mode, outside_node,
// row_shape and duplicate_id).
func (v *validator) riskRoutes(rows []RouteRisk) {
	if len(rows) == 0 {
		return
	}
	modes := modeIndex(&v.c)
	if modes == nil {
		v.rep.add(ErrModuleDependency, path(riskBase, "routes"), "transport", "")
	}
	if !v.riskCount("routes", len(rows)) {
		return
	}
	outside := outsideNodeSet(&v.c)
	edges := make(map[[2]string]bool, len(v.c.Edges)) // a membership test only
	for _, e := range v.c.Edges {
		edges[[2]string{e.From, e.To}] = true
	}
	seen := make(map[[3]string]bool, len(rows)) // a membership test only
	for i := range rows {
		r := &rows[i]
		p := riskRow("routes", i)
		from := v.riskRegion(p, "from", r.From, outside)
		to := v.riskRegion(p, "to", r.To, outside)
		if from && to && !edges[[2]string{r.From, r.To}] {
			v.rep.add(ErrUnknownEdge, p, r.From+"->"+r.To, "")
		}
		v.riskMode(p, r, modes)
		key := [3]string{r.From, r.To, r.Mode}
		if seen[key] {
			v.rep.add(ErrDuplicateID, p, r.From+"->"+r.To+" "+r.Mode, "")
		}
		seen[key] = true
	}
}

// riskMode checks the mode and the numbers of a route row (rules unknown_mode and
// row_shape). A nil set of modes is the module dependency, reported once for the
// table.
func (v *validator) riskMode(p []string, r *RouteRisk, modes map[string]*Mode) {
	if modes != nil && modes[r.Mode] == nil {
		v.rep.add(ErrUnknownMode, path(p, "mode"), r.Mode, "")
	}
	v.riskShape(p, r.StrikeBP, r.SeverityBP)
}

// riskStores checks the store rows: the module dependency, the count, and for each
// row the region, the class, the numbers and the key (rules risk_module_dependency,
// unknown_region, outside_node, unknown_store_class, row_shape and duplicate_id).
func (v *validator) riskStores(rows []StoreRisk) {
	if len(rows) == 0 {
		return
	}
	m := v.c.Modules.Storage
	if m == nil {
		v.rep.add(ErrModuleDependency, path(riskBase, "stores"), "storage", "")
	}
	if !v.riskCount("stores", len(rows)) {
		return
	}
	classes := map[string]bool{} // a membership test only
	if m != nil {
		for i := range m.Classes {
			classes[m.Classes[i].ID] = true
		}
	}
	outside := outsideNodeSet(&v.c)
	seen := make(map[[2]string]bool, len(rows)) // a membership test only
	for i := range rows {
		r := &rows[i]
		p := riskRow("stores", i)
		v.riskRegion(p, "region", r.Region, outside)
		if m != nil && !classes[r.Class] {
			v.rep.add(ErrUnknownStoreClass, path(p, "class"), r.Class, "")
		}
		v.riskShape(p, r.StrikeBP, r.SeverityBP)
		key := [2]string{r.Region, r.Class}
		if seen[key] {
			v.rep.add(ErrDuplicateID, p, r.Region+" "+r.Class, "")
		}
		seen[key] = true
	}
}

// riskWarnings are the four warnings of spec-risk.md 13.2 (FR-RSK-004), none of
// which fails validation: no_risk_rows, risk_edge_off_path, risk_store_class_unowned
// and risk_certain. It is nil with the module off and logs nothing itself.
func riskWarnings(c *Config) []Warning {
	m := c.Modules.Risk
	if m == nil {
		return nil
	}
	if len(m.Routes) == 0 && len(m.Stores) == 0 {
		return []Warning{{Path: riskBase, Rule: "no_risk_rows"}}
	}
	var out []Warning
	for i := range m.Routes {
		if r := &m.Routes[i]; r.StrikeBP == wholeBP {
			out = append(out, Warning{Path: path(riskRow("routes", i), "strike_bp"), Rule: "risk_certain"})
		}
	}
	for i := range m.Stores {
		if r := &m.Stores[i]; r.StrikeBP == wholeBP {
			out = append(out, Warning{Path: path(riskRow("stores", i), "strike_bp"), Rule: "risk_certain"})
		}
	}
	out = append(out, riskOffPath(c, m)...)
	return append(out, riskUnowned(c, m)...)
}

// riskOffPath is risk_edge_off_path: a route row whose edge no path of the
// configured graph, at the configured distances, crosses. An edge (a, b) is on the
// path of some pair exactly when NextHop(a, t) is b for some region t, because the
// path from a starts with that hop. A graph that does not build, or a row whose
// regions are not defined, gives nothing: Validate reports those.
func riskOffPath(c *Config, m *Risk) []Warning {
	g, err := NewGraph(*c)
	if err != nil {
		return nil
	}
	var out []Warning
	for i := range m.Routes {
		r := &m.Routes[i]
		from, ok1 := g.Lookup(r.From)
		to, ok2 := g.Lookup(r.To)
		if !ok1 || !ok2 {
			continue
		}
		used := false
		for t := range RegionID(g.Len()) {
			if hop, ok := g.NextHop(from, t); ok && hop == to {
				used = true
				break
			}
		}
		if !used {
			out = append(out, Warning{Path: riskRow("routes", i), Rule: "risk_edge_off_path"})
		}
	}
	return out
}

// riskUnowned is risk_store_class_unowned: a store row whose class is listed by no
// role row's Stores, so it matters only if a call gives a store of the class.
func riskUnowned(c *Config, m *Risk) []Warning {
	st := c.Modules.Storage
	if st == nil {
		return nil
	}
	var out []Warning
	for i := range m.Stores {
		class := m.Stores[i].Class
		owned := slices.ContainsFunc(st.Roles, func(r RoleRoom) bool { return slices.Contains(r.Stores, class) })
		if !owned {
			out = append(out, Warning{Path: path(riskRow("stores", i), "class"), Rule: "risk_store_class_unowned"})
		}
	}
	return out
}
