package calibration

import (
	"regexp"
	"strings"
	"unicode"
)

// This file is the one map from a configuration path to a registry (scope,
// field) (plan 2-21 D627). It serves the `in` and `chk` check against the first
// table of VIA 11 and, in Phase 6, the purity test of a calibration overlay
// (CLB 11.5): both read the same paths, spelt as EXP 11.3.1 spells them.

const star = "*" // every field of a scope and of the scopes nested in it

// viaPaths is the table-prefix rule of EXP 11.3.1 and the snake form, written
// out for each path of the first table of VIA 11. A path that VIA 11 lists and
// this map lacks fails TestRegistry_SolverFlags, so a new path is found.
func viaPaths() map[string][]pair {
	return map[string][]pair{
		"world.upkeep":                        {{"World", "Upkeep"}, {"Item", star}},
		"role.<id>.upkeep":                    {{"Role", "Upkeep"}, {"Item", star}},
		"region.<id>.extra_upkeep":            {{"Region", "ExtraUpkeep"}},
		"region.<id>.group.<role>.households": {{"Group", "Households"}},
		"recipe.<id>.duration":                {{"Recipe", "Duration"}},
		"recipe.<id>.inputs":                  {{"Recipe", "Inputs"}, {"Item", star}},
		"recipe.<id>.outputs":                 {{"Recipe", "Outputs"}, {"Output", star}},
		"recipe.<id>.requires":                {{"Recipe", "Requires"}, {"Tool", star}},
		"recipe.<id>.draws":                   {{"Recipe", "Draws"}, {"Draw", star}},
		"recipe.<id>.sited":                   {{"Recipe", "Sited"}},
		"commodity.<id>.durability":           {{"Commodity", "Durability"}},
		"commodity.<id>.shelf_life":           {{"Commodity", "ShelfLife"}},
		"commodity.<id>.raw":                  {{"Commodity", "Raw"}},
		"region.<id>.yield.<recipe>": {
			{"YieldProfile", "All"}, {"YieldProfile", "Seasons"}, {"SeasonYield", star}, {"Config", "Yields"},
		},
		"calendar.*":                                        {{"Calendar", star}, {"Month", star}, {"Season", star}},
		"role.<id>.expected_work_ticks":                     {{"Role", "ExpectedWorkTicks"}},
		"role.<id>.recipes":                                 {{"Role", "Recipes"}},
		"modules.resources.resource.<id>.*":                 {{"Resource", star}, {"AbundanceKnot", star}, {"SeasonRegen", star}},
		"modules.resources.endowment.<region>.<resource>.*": {{"Endowment", star}},
		"modules.resources.horizon_years":                   {{"Resources", "HorizonYears"}},
		"modules.demand.want.<id>.*":                        {{"Want", star}, {"WantItem", star}},
		"modules.demand.taste.<role>.<want>.*":              {{"Taste", star}},
		"modules.demand.region_taste.<region>.<want>.*":     {{"RegionTaste", star}},
		"modules.transport.mode.*":                          {{"Mode", star}},
		"region.<id>.edge.<id>.distance":                    {{"Edge", "Distance"}},
		"modules.external.node.*": {
			{"OutsideNode", star}, {"OutsideGate", star}, {"OutsideLine", star}, {"OutsideSide", star}, {"SeasonFactor", star},
		},
		"commodity.<id>.reference_price":          {{"Commodity", "ReferencePrice"}},
		"region.<id>.reference_price.<commodity>": {{"RegionPrice", "Price"}},
		"world.belief_step":                       {{"World", "BeliefStep"}},
		"world.cost_floor":                        {{"World", "CostFloor"}},
		"world.cost_ceiling":                      {{"World", "CostCeiling"}},
		"world.ref_band_low":                      {{"World", "RefBandLow"}},
		"world.ref_band_high":                     {{"World", "RefBandHigh"}},
		"role.<id>.target_ratio":                  {{"Role", "TargetRatio"}},
		"role.<id>.target_spread":                 {{"Role", "TargetSpread"}},
		"world.welfare_window":                    {{"World", "WelfareWindow"}},
		"world.window_cycles":                     {{"World", "WindowCycles"}},
		"modules.infrastructure.*":                {{"Infrastructure", star}, {"FacilityClass", star}, {"InitialFacility", star}},
		"modules.storage.*":                       {{"Storage", star}, {"RoleRoom", star}, {"StoreClass", star}, {"Capacity", star}},
		"modules.institutions.control.<id>.*":     {{"Control", star}},
		"modules.tenure.term.*":                   {{"TenureTerm", star}},
	}
}

var reSnakeParts = regexp.MustCompile(`[a-z0-9]+`)

// pascal respells a snake-case key of a configuration file as the Go field name
// (spec-config.md 6.2): the acronyms BP and ID keep their capitals, and the
// exception in_progress is InProgressBP.
func pascal(snake string) string {
	if snake == "in_progress" {
		return "InProgressBP"
	}
	var b strings.Builder
	for _, w := range reSnakeParts.FindAllString(snake, -1) {
		switch w {
		case "bp", "id":
			b.WriteString(strings.ToUpper(w))
		default:
			r := []rune(w)
			b.WriteRune(unicode.ToUpper(r[0]))
			b.WriteString(string(r[1:]))
		}
	}
	return b.String()
}
