package economysim_test

import (
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// registryRow is one row of the parameter-owner table of doc/specs/README.md.
type registryRow struct {
	param  string // the field, the first backticked word of the parameter cell
	scope  string // the Go type, the words of the scope cell before a parenthesis
	option bool   // the row is an option that changes results, not a field
}

// registryInput is what checkRegistry compares: the rows, the types of the
// code by name, the staged list of what is not built, and the fields that have
// no row by rule.
type registryInput struct {
	rows     []registryRow
	types    map[string]reflect.Type
	notBuilt []string        // "Scope" or "Scope.Field", the staged form of architecture.md 9.7
	exempt   map[string]bool // "Scope.Field" that needs no row
}

// notBuilt lists the scopes (Go type names) and the fields of built scopes that
// the code does not have yet, so the registry test skips their rows
// (architecture.md 9.7, spec-config.md 9.2). The change that builds a scope or a
// field deletes its entries; the test fails when an entry exists in the code,
// and when a row is neither built nor listed.
var notBuilt = []string{
	// Sections of Config built in Stage 2 (regions graph and price overrides).
	"Config.Edges", "Config.RegionPrices", "Edge", "RegionPrice",
	// The pointers of Modules: a module's change adds its field and deletes its
	// entry here. Modules.MoneySupply is the scope token the table spells for the
	// rows of that module's flows.
	"Modules.Credit", "Modules.Employment", "Modules.Estates", "Modules.External", "Modules.Fiscal",
	"Modules.Information", "Modules.Infrastructure", "Modules.Institutions", "Modules.Insurance",
	"Modules.Merchants", "Modules.Migration", "Modules.Policy", "Modules.Polity",
	"Modules.Population", "Modules.Resources", "Modules.Risk", "Modules.Saving",
	"Modules.Settlements", "Modules.Storage", "Modules.Subsistence", "Modules.Technology",
	"Modules.Tenure", "Modules.Transport",
	// Fields that a module appends to the core types Recipe, Region and Role.
	"Recipe.Builds", "Recipe.Draws", "Recipe.Needs", "Recipe.Repairs", "Region.PoolRelease",
	"Region.ReliefBelow", "Role.Ceiling", "Role.Cover", "Role.Credit", "Role.Employer", "Role.Fixed",
	"Role.Hireable", "Role.Informed", "Role.Merchant", "Role.MerchantFeeShare", "Role.Needs",
	"Role.Saver",
	// Scopes (Go type names) of the modules that are not built.
	"AbundanceKnot", "Body", "Borrow", "Capacity", "ClassLimit", "Control", "CoverTerms", "Credit",
	"CreditTerms", "Demand", "Duty", "Employer", "Employment", "Endowment", "External",
	"FacilityBuild", "FacilityClass", "FacilityRepair", "Fiscal", "FlowTarget", "Holding",
	"Information", "Infrastructure", "InitialFacility", "Institutions", "Insurance", "Insure",
	"Jurisdiction", "Knot", "Lend", "Lever", "Member", "Merchant", "Merchants", "Migration", "Mode",
	"Modules.MoneySupply", "MoneyFlow", "OutsideGate", "OutsideLine", "OutsideNode", "OutsideSide",
	"Payout", "Policy", "PolicyActor", "Polity", "Population", "ProjectWorth", "Purchase",
	"RateLimit", "Reaction", "RegionMove", "RegionPopulation", "RegionSettlement", "RegionTaste",
	"Release", "Resource", "Resources", "Right", "Risk", "RoleOwnUse", "RoleRoom", "RouteRisk",
	"Saver", "Saving", "ScheduleStep", "SeasonFactor", "SeasonRegen", "SettlementClass",
	"Settlements", "Storage", "StoreClass", "StoreRisk", "Subsistence", "Taste", "Tax", "TechField",
	"Technology", "Tenure", "TenureTerm", "Toll", "Transport", "Underwrite", "Want", "WantItem",
	"Window",
}

// exemptFields are the fields of the built types that have no row by rule:
// the sections of Config that are containers whose fields are the rows
// (spec-config.md 9.2), the presentation-only fields (8.3) and the Has flag
// that only spells in Go what the file says by writing a key (README, Parameter
// owners).
var exemptFields = map[string]bool{
	"Config.World": true, "Config.Calendar": true, "Config.Commodities": true,
	"Config.Recipes": true, "Config.Roles": true, "Config.Regions": true,
	"Config.Groups": true, "Config.Modules": true,
	"YieldProfile.HasAll": true,
}

// readOwnerTable reads the parameter-owner table of doc/specs/README.md by a
// plain text parse: the pipe rows under the heading "Parameter owners".
func readOwnerTable(t *testing.T) []registryRow {
	t.Helper()
	raw, err := os.ReadFile("doc/specs/README.md")
	if err != nil {
		t.Fatalf("read the parameter-owner table: %v", err)
	}
	var rows []registryRow
	in := false
	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			in = strings.TrimSpace(strings.TrimPrefix(line, "## ")) == "Parameter owners"
		case in && strings.HasPrefix(line, "|"):
			cells := strings.Split(line, "|")
			if len(cells) < 4 {
				continue
			}
			param, scope := strings.TrimSpace(cells[1]), strings.TrimSpace(cells[2])
			if param == "Parameter" || strings.HasPrefix(param, "---") {
				continue
			}
			row := registryRow{option: strings.Contains(scope, "(option")}
			if open := strings.IndexByte(param, '`'); open >= 0 {
				if end := strings.IndexByte(param[open+1:], '`'); end >= 0 {
					row.param = param[open+1 : open+1+end]
				}
			}
			row.scope, _, _ = strings.Cut(scope, " (")
			row.scope = strings.TrimSpace(row.scope)
			if row.param == "" || row.scope == "" {
				t.Fatalf("owner table row %q has no parameter or no scope", line)
			}
			rows = append(rows, row)
		}
	}
	return rows
}

// typesOf returns the struct types reachable from root, by name.
func typesOf(root reflect.Type) map[string]reflect.Type {
	out := map[string]reflect.Type{}
	var visit func(reflect.Type)
	visit = func(t reflect.Type) {
		if _, ok := out[t.Name()]; ok {
			return
		}
		out[t.Name()] = t
		for i := range t.NumField() {
			ft := t.Field(i).Type
			if ft.Kind() == reflect.Slice || ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				visit(ft)
			}
		}
	}
	visit(root)
	return out
}

// checkRegistry runs checks 1 and 2 of spec-config.md 9.2 in the staged form
// and returns what is wrong, one line each, sorted:
//
//  1. every built row is a field of the type of its scope;
//  2. every field of a built type has a row, unless exempt (never staged);
//
// and the two conditions of the staged form: a listed scope or field that
// exists in the code is stale, and a row that is neither built nor listed is a
// typo or a gap.
func checkRegistry(in registryInput) []string {
	var problems []string
	listed := map[string]bool{}
	for _, e := range in.notBuilt {
		listed[e] = true
		scope, field, isField := strings.Cut(e, ".")
		typ, built := in.types[scope]
		switch {
		case isField && !built:
			problems = append(problems, fmt.Sprintf("notBuilt entry %s names a field of the scope %s, which is not built: list the scope", e, scope))
		case isField:
			if _, ok := typ.FieldByName(field); ok {
				problems = append(problems, fmt.Sprintf("notBuilt entry %s exists in the code: delete it", e))
			}
		case built:
			problems = append(problems, fmt.Sprintf("notBuilt entry %s exists in the code: delete it", e))
		}
	}

	have := map[string]bool{}
	for _, row := range in.rows {
		have[row.scope+"."+row.param] = true
		if row.option {
			continue // checked against the option, not a field (spec-config.md 9.2)
		}
		key := row.scope + "." + row.param
		if listed[row.scope] || listed[key] {
			continue
		}
		typ, built := in.types[row.scope]
		if !built {
			problems = append(problems, fmt.Sprintf("row %s: scope %s is neither built nor listed in notBuilt", key, row.scope))
			continue
		}
		if _, ok := typ.FieldByName(row.param); !ok {
			problems = append(problems, fmt.Sprintf("row %s: %s has no field %s, and the field is not listed in notBuilt", key, row.scope, row.param))
		}
	}
	for _, e := range in.notBuilt {
		used := false
		for _, row := range in.rows {
			if row.scope == e || row.scope+"."+row.param == e {
				used = true
				break
			}
		}
		if !used {
			problems = append(problems, fmt.Sprintf("notBuilt entry %s matches no row: a typo hides nothing and is removed", e))
		}
	}

	for name, typ := range in.types {
		for i := range typ.NumField() {
			key := name + "." + typ.Field(i).Name
			if !have[key] && !in.exempt[key] {
				problems = append(problems, fmt.Sprintf("field %s has no row in the parameter-owner table", key))
			}
		}
	}
	slices.Sort(problems)
	return problems
}

// TestParameterRegistry_Config verifies, over the real configuration types and
// the real table of doc/specs/README.md, that every built row is a field and
// every field has a row, with the staged list of what the code does not have
// yet.
//
// Requirement: FR-CFG-028
func TestParameterRegistry_Config(t *testing.T) {
	t.Parallel()
	rows := readOwnerTable(t)
	if len(rows) < 100 {
		t.Fatalf("read %d rows of the parameter-owner table, want the whole table (more than 100)", len(rows))
	}
	exempt := maps.Clone(exemptFields)
	for _, name := range presentationFields() {
		exempt[name] = true
	}
	problems := checkRegistry(registryInput{
		rows:     rows,
		types:    typesOf(reflect.TypeFor[economysim.Config]()),
		notBuilt: notBuilt,
		exempt:   exempt,
	})
	for _, p := range problems {
		t.Error(p)
	}
}

// presentationFields returns the allow-list of presentation-only fields, which
// need no row (spec-config.md 8.3, 9.2).
func presentationFields() []string {
	names := make([]string, 0, len(presentationOnly))
	for name := range presentationOnly {
		names = append(names, name)
	}
	return names
}

// TestParameterRegistry_StagedChecks verifies the checker itself, because a
// registry test that cannot fail proves nothing: a listed scope or field that
// exists in the code, a row that is neither built nor listed, a field with no
// row and a row with no field each fail, and a clean registry passes.
//
// Requirement: FR-CFG-028
//
// Guard: the staged form is only safe if a stale entry cannot hide a gap. If
// the test of the real table silently passed on a stale entry, a built field
// could go unchecked for ever.
func TestParameterRegistry_StagedChecks(t *testing.T) {
	t.Parallel()
	type built struct{ A, B int64 }
	types := map[string]reflect.Type{"Built": reflect.TypeFor[built]()}
	rowA := registryRow{param: "A", scope: "Built"}
	rowB := registryRow{param: "B", scope: "Built"}
	tests := []struct {
		name string
		in   registryInput
		want int // number of problems
	}{
		{"clean", registryInput{rows: []registryRow{rowA, rowB}, types: types}, 0},
		{"unbuilt scope listed", registryInput{
			rows: []registryRow{rowA, rowB, {param: "X", scope: "Later"}}, types: types, notBuilt: []string{"Later"}}, 0},
		{"unbuilt field listed", registryInput{
			rows: []registryRow{rowA, rowB, {param: "C", scope: "Built"}}, types: types, notBuilt: []string{"Built.C"}}, 0},
		{"stale scope: it exists in the code", registryInput{
			rows: []registryRow{rowA, rowB}, types: types, notBuilt: []string{"Built"}}, 1},
		{"stale field: it exists in the code", registryInput{
			rows: []registryRow{rowA, rowB}, types: types, notBuilt: []string{"Built.A"}}, 1},
		{"row of an unknown scope that is not listed", registryInput{
			rows: []registryRow{rowA, rowB, {param: "X", scope: "Typo"}}, types: types}, 1},
		{"row of an unknown field that is not listed", registryInput{
			rows: []registryRow{rowA, rowB, {param: "C", scope: "Built"}}, types: types}, 1},
		{"field with no row", registryInput{rows: []registryRow{rowA}, types: types}, 1},
		{"field with no row but exempt", registryInput{
			rows: []registryRow{rowA}, types: types, exempt: map[string]bool{"Built.B": true}}, 0},
		{"entry that matches no row", registryInput{
			rows: []registryRow{rowA, rowB}, types: types, notBuilt: []string{"Nowhere"}}, 1},
		{"field of an unbuilt scope listed alone", registryInput{
			rows: []registryRow{rowA, rowB, {param: "X", scope: "Later"}}, types: types, notBuilt: []string{"Later.X"}}, 1},
		{"option row needs no field", registryInput{
			rows: []registryRow{rowA, rowB, {param: "WithX", scope: "Built", option: true}}, types: types}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := checkRegistry(tt.in); len(got) != tt.want {
				t.Errorf("checkRegistry(%s) = %d problems %q, want %d", tt.name, len(got), got, tt.want)
			}
		})
	}
}

// noFailingCase lists the built rows whose field has no value that any rule of
// Validate rejects, each with the reason (spec-config.md 9.2, check 5). A row
// is here because no value is wrong, not because a test is missing.
var noFailingCase = map[string]string{
	"World.Spawner":  "a registered name: NewWorld checks it with the registrations (spec-config.md 7.1.4), Validate cannot",
	"World.Clearing": "a registered name: NewWorld checks it with the registrations (spec-config.md 7.1.4), Validate cannot",
	"Role.Strategy":  "a registered name: NewWorld checks it with the registrations (spec-config.md 7.1.4), Validate cannot",
	"Role.Unskilled": "a bool: both values are valid",
	"Role.Closed":    "a bool: both values are valid",
	"Recipe.Sited":   "a bool: both values are valid",
	"Output.Discard": "a bool: both values are valid",
	"Commodity.Raw":  "a bool: both values are valid",
}

// builtRows returns the rows of the owner table that are built and are not
// exempt: the rows whose scope and field exist in the code and are not in the
// staged list (spec-config.md 9.2).
func builtRows(t *testing.T) []string {
	t.Helper()
	types := typesOf(reflect.TypeFor[economysim.Config]())
	listed := map[string]bool{}
	for _, e := range notBuilt {
		listed[e] = true
	}
	var out []string
	for _, row := range readOwnerTable(t) {
		key := row.scope + "." + row.param
		if row.option || listed[row.scope] || listed[key] || exemptFields[key] || presentationOnly[key] {
			continue
		}
		if typ, ok := types[row.scope]; ok {
			if _, ok := typ.FieldByName(row.param); ok {
				out = append(out, key)
			}
		}
	}
	return out
}

// TestParameterRegistry_EveryBuiltRowHasAFailingCase verifies check 5 of
// spec-config.md 9.2: every built row of the parameter-owner table is named by a
// failing case of the validation tests, read from the case table of
// TestValidate_FieldRules, or is listed in noFailingCase with its reason; and no
// case names a field that is not a row. The presentation-only fields are
// exempt from the table and have cases for their length all the same.
//
// Requirement: FR-CFG-031
func TestParameterRegistry_EveryBuiltRowHasAFailingCase(t *testing.T) {
	t.Parallel()
	have := map[string]bool{}
	for _, c := range fieldCases {
		have[c.field] = true
	}
	for _, key := range builtRows(t) {
		if _, listed := noFailingCase[key]; !have[key] && !listed {
			t.Errorf("row %s has no failing case in fieldCases and is not in noFailingCase", key)
		}
		if _, listed := noFailingCase[key]; have[key] && listed {
			t.Errorf("row %s is in noFailingCase and has a failing case: delete its entry", key)
		}
	}
	types := typesOf(reflect.TypeFor[economysim.Config]())
	fields := slices.Sorted(maps.Keys(have))
	for key := range noFailingCase {
		fields = append(fields, key)
	}
	for _, key := range fields {
		scope, field, _ := strings.Cut(key, ".")
		typ, ok := types[scope]
		if !ok {
			t.Errorf("case or entry %s names the scope %s, which is not a type of the configuration", key, scope)
			continue
		}
		if _, ok := typ.FieldByName(field); !ok {
			t.Errorf("case or entry %s names a field that %s does not have", key, scope)
		}
	}
}

// TestParameterRegistry_DefaultsPassValidation verifies check 3 of
// spec-config.md 9.2: the default of every field, from its Default constructor,
// passes validation. It builds one configuration of the constructors' values and
// the fields they leave required, and checks that it validates; it then sets each
// field of a built scope to its default and checks that it does not move out of
// its range, by validating a configuration that differs from the minimal one in
// that field only.
//
// Requirement: FR-CFG-029
func TestParameterRegistry_DefaultsPassValidation(t *testing.T) {
	t.Parallel()
	base := minimalConfig()
	if err := base.Validate(); err != nil {
		t.Fatalf("Validate(defaults) = %v, want nil", err)
	}
	// Every integer field of a Default constructor is set on the minimal
	// configuration from a second constructor call: a default that is stateful
	// or shared would differ between the two.
	t.Run("constructors are pure", func(t *testing.T) {
		t.Parallel()
		if !reflect.DeepEqual(economysim.DefaultWorld(), economysim.DefaultWorld()) ||
			!reflect.DeepEqual(economysim.DefaultCalendar(), economysim.DefaultCalendar()) ||
			!reflect.DeepEqual(economysim.DefaultRole(), economysim.DefaultRole()) {
			t.Errorf("two calls of a Default constructor differ")
		}
		a, b := economysim.DefaultCalendar(), economysim.DefaultCalendar()
		a.Months[0].Days = 99
		if b.Months[0].Days == 99 {
			t.Errorf("DefaultCalendar shares its Months between calls")
		}
	})
	type target struct {
		name string
		set  func(c *economysim.Config)
	}
	targets := []target{
		{"World", func(c *economysim.Config) {
			u := c.World.Upkeep
			c.World = economysim.DefaultWorld()
			c.World.Upkeep = u
		}},
		{"Calendar", func(c *economysim.Config) { c.Calendar = economysim.DefaultCalendar() }},
		{"Role", func(c *economysim.Config) {
			id := c.Roles[0].ID
			c.Roles[0] = economysim.DefaultRole()
			c.Roles[0].ID = id
		}},
		{"Region", func(c *economysim.Config) {
			id := c.Regions[0].ID
			c.Regions[0] = economysim.DefaultRegion()
			c.Regions[0].ID = id
		}},
		{"Commodity", func(c *economysim.Config) {
			d := economysim.DefaultCommodity()
			d.ID, d.ReferencePrice, d.Raw = c.Commodities[0].ID, c.Commodities[0].ReferencePrice, true
			c.Commodities[0] = d
		}},
		{"Recipe", func(c *economysim.Config) {
			d := economysim.DefaultRecipe()
			d.ID = "make"
			d.Outputs = []economysim.Output{{Commodity: "grain", Quantity: 1, CostShare: 10_000}}
			c.Recipes = []economysim.Recipe{d}
		}},
	}
	for _, tt := range targets {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			tt.set(&c)
			if err := c.Validate(); err != nil {
				t.Errorf("Validate(%s set to its default) = %v, want nil", tt.name, err)
			}
		})
	}
}

// specDefault reads the default of a row of an owner spec's parameter table:
// the rows of the form `| Scope | Field | Meaning | Default |` (with a Bound
// column after). It returns the integer in the default cell, spaces removed, and
// false where the cell is not one integer (a list, a name, "none").
func specDefault(cells []string) (scope, field string, def int64, ok bool) {
	if len(cells) < 6 {
		return "", "", 0, false
	}
	scope = strings.TrimSpace(cells[1])
	field = strings.Trim(strings.TrimSpace(cells[2]), "`")
	cell := strings.ReplaceAll(strings.TrimSpace(cells[4]), " ", "")
	cell = strings.ReplaceAll(cell, "_", "")
	n, err := strconv.ParseInt(cell, 10, 64)
	return scope, field, n, err == nil
}

// TestParameterRegistry_SpecDefaultsEqualConstructors verifies check 4 of
// spec-config.md 9.2 (a SHOULD, FR-CFG-030): the default that an owner spec's
// parameter table states for a World, Role, Region or Commodity field is the
// default of its Default constructor. It reads the pipe rows of the owner specs
// by a plain text parse and compares the integers it can read.
//
// Requirement: FR-CFG-030
func TestParameterRegistry_SpecDefaultsEqualConstructors(t *testing.T) {
	t.Parallel()
	files := []string{
		"spec-agents.md", "spec-beliefs.md", "spec-spawner.md", "spec-welfare-and-wages.md",
		"spec-goods.md", "spec-production.md", "spec-scale.md", "spec-skill-barriers.md", "spec-estates.md",
	}
	ctor := map[string]reflect.Value{
		"World":     reflect.ValueOf(economysim.DefaultWorld()),
		"Role":      reflect.ValueOf(economysim.DefaultRole()),
		"Region":    reflect.ValueOf(economysim.DefaultRegion()),
		"Commodity": reflect.ValueOf(economysim.DefaultCommodity()),
		"Recipe":    reflect.ValueOf(economysim.DefaultRecipe()),
	}
	checked := 0
	for _, name := range files {
		raw, err := os.ReadFile("doc/specs/" + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if !strings.HasPrefix(line, "|") {
				continue
			}
			scope, field, want, ok := specDefault(strings.Split(line, "|"))
			if !ok {
				continue
			}
			v, known := ctor[scope]
			if !known {
				continue
			}
			f := v.FieldByName(field)
			if !f.IsValid() || f.Kind() != reflect.Int64 {
				continue
			}
			checked++
			if got := f.Int(); got != want {
				t.Errorf("%s: the default of %s.%s is %d in the table and %d in Default%s()", name, scope, field, want, got, scope)
			}
		}
	}
	t.Logf("compared %d defaults of the owner tables with the constructors", checked)
	if checked < 30 {
		t.Errorf("compared %d defaults, want at least 30: the parse of the owner tables found too few rows", checked)
	}
}
