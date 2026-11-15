package economysim_test

import (
	"reflect"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file is the part of the golden renderer that reads the names table and the
// module list (the continuation of render_test.go).

// table returns the names table of the typed ID type with the name, and false for an
// ID with no table (AgentID and any other is its integer).
func (n namer) table(name string) ([]string, bool) {
	switch name {
	case "RegionID":
		return n.names.Regions, true
	case "CommodityID":
		return n.names.Commodities, true
	case "RoleID":
		return n.names.Roles, true
	case "RecipeID":
		return n.names.Recipes, true
	case "JurisdictionID":
		return n.names.Jurisdictions, true
	case "SettlementClassID":
		return n.names.SettlementClasses, true
	}
	return nil, false
}

// offSettlementIDs is the skip of the typed IDs a row appends for the settlements
// module (RegionRow.Class and ClassTarget): zero is a valid class, so with the module
// off the golden writer omits them, and the files of golden/off/ do not move
// (architecture.md 2.10.5, spec-observability.md 9.5.2). With the module on they are
// written.
func offSettlementIDs(s economysim.Snapshot) func(reflect.StructField) bool {
	if slices.Contains(s.Modules, "settlements") {
		return nil
	}
	return func(f reflect.StructField) bool { return f.Name == "Class" || f.Name == "ClassTarget" }
}
