package economysim_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	economysim "github.com/Whealinghold/economy-sim"
)

// Compile-time assertions: an event is an Event only through this package, and
// ObserverFunc is an Observer (architecture.md 4.5, 4.8).
var (
	_ economysim.Observer = economysim.ObserverFunc(nil)
	_ economysim.Event    = economysim.TickStarted{}
)

// declaredEvents is one zero value of every event type declared in event.go.
// TestEvents_DeclaredListMatchesSource keeps it complete.
var declaredEvents = []economysim.Event{
	economysim.WorldCreated{},
	economysim.RunStarted{},
	economysim.RunEnded{},
	economysim.TickStarted{},
	economysim.TickEnded{},
	economysim.OrderPlaced{},
	economysim.OrderRejected{},
	economysim.TradeExecuted{},
	economysim.MarketCleared{},
	economysim.BeliefsUpdated{},
	economysim.RecipeExecuted{},
	economysim.RecipeRefused{},
	economysim.RecipeAbandoned{},
	economysim.ProductionHalted{},
	economysim.UpkeepShortfall{},
	economysim.WantShortfall{},
	economysim.AgentSpawned{},
	economysim.AgentBankrupted{},
	economysim.EstateOpened{},
	economysim.WeightChanged{},
	economysim.RoleChanged{},
	economysim.EntryLimited{},
	economysim.BoundaryFlow{},
	economysim.WearApplied{},
	economysim.WorldRestored{},
	economysim.AgentRelocated{},
	economysim.EdgeStatusChanged{},
}

// notBuiltEvents names every row of the tables of spec-observability.md 4.1
// that event.go does not declare yet: the events of the modules, of the
// regions, checkpoint and skill stages, and of ReliefPaid, which belongs to
// the pool policy of the estates module (spec-config.md 6.2.4). The change that
// declares one deletes its entry; the catalogue test fails on a stale entry.
var notBuiltEvents = []string{
	"OwnUseChanged", "RoomLimited",
	"PopulationChanged", "TraineeshipStarted", "TraineeshipCompleted",
	"TraineeshipEnded", "StoreChanged", "ReliefPaid", "MoneyFlowLimited",
	"MoneyFlowEnded", "MoneyFlowScaled", "StoreLoss",
	"ShipmentDispatched", "ShipmentArrived",
	"ShipmentRejected", "TransitFactorChanged", "ContractFormed",
	"ContractSettled", "ContractRound", "MerchantsPlanned", "MigrationStarted",
	"MigrationReviewed", "EndowmentChanged", "EndowmentAdjusted",
	"RumoursServed", "OutsideTraded", "OutsideEnded", "OutsideShocked",
	"YieldShocked", "PopulationScaled", "RouteRiskScaled", "StoreRiskScaled",
	"HireFormed", "HireEnded", "LabourRound", "LoanFormed", "LoanEnded",
	"LoanPassed", "CreditRound", "CreditSettled", "CreditRateScaled",
	"TechnologyAdvanced", "TechnologyAdjusted", "SettlementClassChanged",
	"SettlementHeld", "RentPaid", "TitleTransferred", "RentScaled",
	"FacilityChanged", "FacilityLimited", "FiscalFlowApplied",
	"TreasuryOrdered", "MembershipChanged", "RightRefused", "PriceControlled",
	"LegalLevelSet", "RightSuspensionSet", "SavingPlanned", "PolicyDecided",
	"CoverFormed", "CoverEnded", "CoverRound", "CoverSettled",
}

// deferredCore names every event whose row of 4.1 has the gate core but that
// event.go does not declare, with the stage and file that builds it. A core
// row that is neither declared nor here is a forgotten event; an entry that is
// declared in the code is stale. The change that declares the event deletes
// its entry here and in notBuiltEvents.
var deferredCore = map[string]string{
	"YieldShocked": "Stage 2, integration.go (INT, PRO 6.8, roadmap-v1.md)",
}

// notBuiltFields names the Event.Field entries of a row of 4.1 whose module
// has not appended the field to its declared event yet (D3 of the plan of
// Stage 0). The change that appends one deletes its entry.
var notBuiltFields = []string{
	"WorldCreated.Jurisdictions", // polity module, POL 10.7
	"AgentSpawned.StoresLocked",  // storage module, STO 14.1
	"EntryLimited.RoleCeiling",   // settlements module, SET 6.4
}

// eventRow is one row of the tables of spec-observability.md 4.1.
type eventRow struct {
	gate   string
	fields []string
}

var (
	backtickRe = regexp.MustCompile("`([^`]*)`")
	withRe     = regexp.MustCompile("with `[^`]*`")
	identRe    = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
)

// stripParens removes every balanced parenthesised group from s.
func stripParens(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '(':
			depth++
		case r == ')':
			if depth > 0 {
				depth--
			}
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// rowFieldNames returns the names of the top-level fields of a "Fields" cell
// of 4.1, in the order written: the first word of each backticked span that is
// not inside parentheses and is not the row type of a nested slice.
func rowFieldNames(cell string) []string {
	cell = withRe.ReplaceAllString(stripParens(cell), "")
	var names []string
	for _, m := range backtickRe.FindAllStringSubmatch(cell, -1) {
		if strings.ContainsAny(m[1], "{};") {
			continue
		}
		words := strings.Fields(m[1])
		if len(words) > 0 && identRe.MatchString(words[0]) {
			names = append(names, words[0])
		}
	}
	return names
}

// readFile reads a file by a path relative to the package directory.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// specRows parses the tables of section 4.1 of spec-observability.md by a
// plain text parse and returns the rows by event name, and the names in the
// order of the file.
func specRows(t *testing.T) (map[string]eventRow, []string) {
	t.Helper()
	text := readFile(t, "doc/specs/spec-observability.md")
	start := strings.Index(text, "\n### 4.1 ")
	end := strings.Index(text, "\n### 4.2 ")
	if start < 0 || end < start {
		t.Fatalf("section 4.1 not found in spec-observability.md")
	}
	rows := map[string]eventRow{}
	var order []string
	nameRe := regexp.MustCompile("^`([A-Z][A-Za-z0-9]*)`")
	for _, line := range strings.Split(text[start:end], "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cols := strings.Split(line, "|")
		if len(cols) < 8 {
			continue
		}
		m := nameRe.FindStringSubmatch(strings.TrimSpace(cols[1]))
		if m == nil {
			continue
		}
		if _, dup := rows[m[1]]; dup {
			t.Fatalf("event %s is a row of 4.1 twice", m[1])
		}
		rows[m[1]] = eventRow{
			gate:   strings.TrimSpace(cols[len(cols)-3]),
			fields: rowFieldNames(cols[len(cols)-2]),
		}
		order = append(order, m[1])
	}
	return rows, order
}

// rulesEvents returns the event names that are rows of the table of
// observability-and-logging.md 4.3, by a plain text parse of its first column.
func rulesEvents(t *testing.T) map[string]bool {
	t.Helper()
	text := readFile(t, "doc/rules/observability-and-logging.md")
	start := strings.Index(text, "\n4.3. The library **MUST** emit at least")
	end := strings.Index(text, "\nThe parentheses name the main fields")
	if start < 0 || end < start {
		t.Fatalf("table of 4.3 not found in observability-and-logging.md")
	}
	names := map[string]bool{}
	for _, line := range strings.Split(text[start:end], "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cell := strings.Split(line, "|")[1]
		for _, m := range backtickRe.FindAllStringSubmatch(cell, -1) {
			names[m[1]] = true
		}
	}
	return names
}

// sourceEvents returns the names of the types of event.go that have the
// unexported method isEvent, by parsing the source.
func sourceEvents(t *testing.T) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "event.go", nil, 0)
	if err != nil {
		t.Fatalf("parse event.go: %v", err)
	}
	var names []string
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || fd.Name.Name != "isEvent" || len(fd.Recv.List) != 1 {
			continue
		}
		if id, ok := fd.Recv.List[0].Type.(*ast.Ident); ok {
			names = append(names, id.Name)
		}
	}
	slices.Sort(names)
	return names
}

// declaredNames returns the type names of declaredEvents, sorted.
func declaredNames() []string {
	var names []string
	for _, e := range declaredEvents {
		names = append(names, reflect.TypeOf(e).Name())
	}
	slices.Sort(names)
	return names
}

// TestEvents_DeclaredListMatchesSource verifies that the list of declared
// events held by these tests is exactly the set of types of event.go that
// implement Event, so that no declared event escapes the shape and catalogue
// tests.
//
// Guard: a new event added to event.go and forgotten here would be exempt from
// every rule these tests enforce.
func TestEvents_DeclaredListMatchesSource(t *testing.T) {
	t.Parallel()
	got, want := declaredNames(), sourceEvents(t)
	if !slices.Equal(got, want) {
		t.Errorf("declared events in tests = %v, want the isEvent types of event.go %v", got, want)
	}
}

// eventProblems returns what is wrong with the type of one field of an event,
// by the rules of observability-and-logging.md 4.1: no pointer, map, function,
// channel, interface, float, complex or time value, and no embedded type, in
// the field or in anything it holds.
func eventProblems(path string, t reflect.Type) []string {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.String:
		return nil
	case reflect.Array, reflect.Slice:
		return eventProblems(path+"[]", t.Elem())
	case reflect.Struct:
		if t.PkgPath() == "time" {
			return []string{path + " is a time value"}
		}
		var out []string
		for i := range t.NumField() {
			f := t.Field(i)
			if f.Anonymous {
				out = append(out, path+"."+f.Name+" is an embedded type")
			}
			out = append(out, eventProblems(path+"."+f.Name, f.Type)...)
		}
		return out
	}
	return []string{fmt.Sprintf("%s has kind %s", path, t.Kind())}
}

// TestEventProblems_RejectsForbiddenShapes verifies the checker that the shape
// test relies on: each forbidden kind is reported, each allowed one is not.
//
// Guard: a checker that accepted a map or a pointer would make the shape test
// pass for an event that breaks the golden rendering.
func TestEventProblems_RejectsForbiddenShapes(t *testing.T) {
	t.Parallel()
	type inner struct{ A int64 }
	type embedded struct {
		inner
		B int64
	}
	tests := []struct {
		name string
		typ  reflect.Type
		bad  bool
	}{
		{"int64", reflect.TypeFor[int64](), false},
		{"hash array", reflect.TypeFor[[32]byte](), false},
		{"slice of struct", reflect.TypeFor[[]inner](), false},
		{"pointer", reflect.TypeFor[*int64](), true},
		{"map", reflect.TypeFor[map[string]int64](), true},
		{"float", reflect.TypeFor[float64](), true},
		{"func", reflect.TypeFor[func()](), true},
		{"interface", reflect.TypeFor[any](), true},
		{"embedded", reflect.TypeFor[embedded](), true},
		{"slice of pointer", reflect.TypeFor[[]*inner](), true},
		{"time", reflect.TypeFor[struct{ T time.Time }](), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := eventProblems("F", tt.typ)
			if (len(got) > 0) != tt.bad {
				t.Errorf("eventProblems(%s) = %v, want a problem: %v", tt.typ, got, tt.bad)
			}
		})
	}
}

// TestEvents_Shape verifies, by reflection over every declared event, that Tick
// is the first field and an int64, that no field is a pointer, map, function,
// float or time value or an embedded type, that a field naming a configured
// entity carries the typed ID of its kind, and that a reason, a cause, a kind
// and a side are of their closed-set types.
//
// Requirement: FR-OBS-001, FR-OBS-005, FR-OBS-006
func TestEvents_Shape(t *testing.T) {
	t.Parallel()
	region := reflect.TypeFor[economysim.RegionID]()
	commodity := reflect.TypeFor[economysim.CommodityID]()
	role := reflect.TypeFor[economysim.RoleID]()
	recipe := reflect.TypeFor[economysim.RecipeID]()
	agent := reflect.TypeFor[economysim.AgentID]()
	// The type a field of this name must have, wherever it appears.
	byName := map[string]reflect.Type{
		"Region": region, "Commodity": commodity, "Role": role,
		"Recipe": recipe, "Agent": agent, "Buyer": agent, "Seller": agent,
		"Reason": reflect.TypeFor[economysim.Reason](),
		"Cause":  reflect.TypeFor[economysim.Cause](),
		"Kind":   reflect.TypeFor[economysim.Kind](),
		"Side":   reflect.TypeFor[economysim.Side](),
	}
	// From and To name two roles in RoleChanged and two weights in
	// WeightChanged (spec-observability.md 4.2.1).
	byEventField := map[string]reflect.Type{
		"RoleChanged.From": role, "RoleChanged.To": role,
		"WeightChanged.From": reflect.TypeFor[int64](), "WeightChanged.To": reflect.TypeFor[int64](),
	}
	for _, e := range declaredEvents {
		typ := reflect.TypeOf(e)
		t.Run(typ.Name(), func(t *testing.T) {
			t.Parallel()
			if typ.Kind() != reflect.Struct || typ.NumField() == 0 {
				t.Fatalf("%s is not a struct with fields", typ)
			}
			if f := typ.Field(0); f.Name != "Tick" || f.Type != reflect.TypeFor[int64]() {
				t.Errorf("%s first field = %s %s, want Tick int64", typ, f.Name, f.Type)
			}
			for i := range typ.NumField() {
				f := typ.Field(i)
				for _, p := range eventProblems(typ.Name()+"."+f.Name, f.Type) {
					t.Errorf("%s", p)
				}
				want, ok := byEventField[typ.Name()+"."+f.Name]
				if !ok {
					want, ok = byName[f.Name]
				}
				if ok && f.Type != want {
					t.Errorf("%s.%s has type %s, want %s", typ, f.Name, f.Type, want)
				}
			}
		})
	}
}

// TestEvents_CatalogueStaged verifies the staged catalogue test of
// spec-observability.md 9.1.2: every declared event is a row of 4.1 with the
// gate core and a row of the table of observability-and-logging.md 4.3; every
// row of 4.1 is declared or named by notBuiltEvents; a listed event that is
// declared is stale; and each notBuiltFields entry names a field of a row that
// the declared event does not have yet.
//
// Requirement: FR-OBS-003
func TestEvents_CatalogueStaged(t *testing.T) {
	t.Parallel()
	rows, order := specRows(t)
	rules := rulesEvents(t)
	declared := map[string]bool{}
	for _, n := range declaredNames() {
		declared[n] = true
	}
	listed := map[string]bool{}
	for _, n := range notBuiltEvents {
		if listed[n] {
			t.Errorf("notBuiltEvents lists %s twice", n)
		}
		listed[n] = true
	}

	for _, name := range order {
		switch {
		case declared[name] && listed[name]:
			t.Errorf("event %s is declared and still in notBuiltEvents: delete the stale entry", name)
		case !declared[name] && !listed[name]:
			t.Errorf("row %s of 4.1 is neither declared nor in notBuiltEvents", name)
		}
		if _, deferred := deferredCore[name]; !declared[name] &&
			strings.HasPrefix(rows[name].gate, "core") && !deferred {
			t.Errorf("row %s of 4.1 has gate core, is not declared and is not in deferredCore: a forgotten core event, or name the stage that builds it", name)
		}
	}
	for n := range declared {
		row, ok := rows[n]
		if !ok {
			t.Errorf("declared event %s is not a row of 4.1", n)
			continue
		}
		if !strings.HasPrefix(row.gate, "core") && row.gate != "demand" {
			t.Errorf("declared event %s has gate %q in 4.1, want core: a module's event is declared with its module", n, row.gate)
		}
		if !rules[n] {
			t.Errorf("declared event %s is not a row of the table of observability-and-logging.md 4.3", n)
		}
	}
	for n, stage := range deferredCore {
		switch row, ok := rows[n]; {
		case stage == "":
			t.Errorf("deferredCore entry %s names no stage", n)
		case declared[n]:
			t.Errorf("deferredCore entry %s is declared: delete the stale entry", n)
		case !listed[n]:
			t.Errorf("deferredCore entry %s is not in notBuiltEvents", n)
		case !ok || !strings.HasPrefix(row.gate, "core"):
			t.Errorf("deferredCore entry %s is not a core row of 4.1", n)
		}
	}
	for n := range listed {
		if _, ok := rows[n]; !ok {
			t.Errorf("notBuiltEvents lists %s, which is not a row of 4.1 (a typo)", n)
		}
	}

	byName := map[string]reflect.Type{}
	for _, e := range declaredEvents {
		byName[reflect.TypeOf(e).Name()] = reflect.TypeOf(e)
	}
	for _, entry := range notBuiltFields {
		ev, field, ok := strings.Cut(entry, ".")
		if !ok {
			t.Errorf("notBuiltFields entry %q is not Event.Field", entry)
			continue
		}
		row, inRow := rows[ev]
		if !inRow || !slices.Contains(row.fields, field) {
			t.Errorf("notBuiltFields entry %s is not a field of a row of 4.1 (a typo)", entry)
		}
		typ, isDeclared := byName[ev]
		if !isDeclared {
			t.Errorf("notBuiltFields entry %s names an event that is not declared", entry)
			continue
		}
		if _, has := typ.FieldByName(field); has {
			t.Errorf("notBuiltFields entry %s is declared: delete the stale entry", entry)
		}
	}
}

// TestEvents_FieldsFollowRow verifies that each declared event has exactly the
// fields of its row of 4.1, less the notBuiltFields entries, in the order of
// the row, after Tick.
//
// Requirement: FR-OBS-004
func TestEvents_FieldsFollowRow(t *testing.T) {
	t.Parallel()
	rows, _ := specRows(t)
	for _, e := range declaredEvents {
		typ := reflect.TypeOf(e)
		t.Run(typ.Name(), func(t *testing.T) {
			t.Parallel()
			row, ok := rows[typ.Name()]
			if !ok {
				t.Fatalf("%s is not a row of 4.1", typ.Name())
			}
			var want []string
			for _, f := range row.fields {
				if !slices.Contains(notBuiltFields, typ.Name()+"."+f) {
					want = append(want, f)
				}
			}
			var got []string
			for i := 1; i < typ.NumField(); i++ {
				got = append(got, typ.Field(i).Name)
			}
			if !slices.Equal(got, want) {
				t.Errorf("%s fields after Tick = %v, want the row of 4.1 less notBuilt %v", typ.Name(), got, want)
			}
		})
	}
}

// TestRowFieldNames verifies the plain text parse of a Fields cell of 4.1 on
// the shapes the table uses: a type after the name, a plus marker,
// parenthesised comments with commas and nested code spans, and a nested row
// type introduced by "with".
//
// Guard: a parse that took a word of a comment for a field would make the
// order test compare against the wrong list and hide a missing field.
func TestRowFieldNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cell string
		want []string
	}{
		{"none", " none ", nil},
		{"plain", " `Region`, `Commodity`, `Limit Price` ", []string{"Region", "Commodity", "Limit"}},
		{"plus and parens", " `Seed uint64`, `Names Names`+, `Jurisdictions []JurisdictionRow`+ (one `JurisdictionRow{Jurisdiction; Seat}` for each, empty with none) ",
			[]string{"Seed", "Names", "Jurisdictions"}},
		{"comma in parens", " `Region`, `RoleCeiling`+ (a refusal by the class, SET 6.4), `Cooldown` ", []string{"Region", "RoleCeiling", "Cooldown"}},
		{"with row type", " `Free`, `Skipped []HaltRow` with `HaltRow{Recipe RecipeID; Households int64}` ", []string{"Free", "Skipped"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := rowFieldNames(tt.cell); !slices.Equal(got, tt.want) {
				t.Errorf("rowFieldNames(%q) = %v, want %v", tt.cell, got, tt.want)
			}
		})
	}
}

// TestConstants_Values verifies the string of each reason, cause and kind
// constant of the slice against the spec that owns it, and that no two
// constants of one type share a value. The values are the values of the reason
// and cause log keys, so a change is a change of the golden output.
//
// The spec has no single parseable list of every closed set (spec-observability.md
// 4.2.3 names only the kinds and defers the rest to the owning specs), so the
// hard-coded table stays and is cross-read from the owners: each value must be
// a backticked token of its owning spec, and the table of spec-clearing.md 3.2
// must have no reason that is neither a constant nor named as deferred.
//
// Guard: the table pins the strings of the golden output; the spec read keeps
// a typo in both the table and the constant from passing.
//
// Requirement: FR-OBS-006
func TestConstants_Values(t *testing.T) {
	t.Parallel()
	// Sources: rejection reasons spec-clearing.md 3.2; abandon reasons
	// spec-production.md 14.1; start refusals spec-production.md 4.2.1;
	// boundary reasons spec-ledger.md 7.2; causes
	// spec-spawner.md 3 and 13.1.1; kinds spec-observability.md 4.2.3.
	reasons := []struct {
		got  economysim.Reason
		want string
	}{
		{economysim.ReasonUnknownAgent, "unknown_agent"},
		{economysim.ReasonSideNotAllowed, "side_not_allowed"},
		{economysim.ReasonUnknownCommodity, "unknown_commodity"},
		{economysim.ReasonNotTradable, "not_tradable"},
		{economysim.ReasonInvalidPrice, "invalid_price"},
		{economysim.ReasonInvalidQuantity, "invalid_quantity"},
		{economysim.ReasonDuplicateOrder, "duplicate_order"},
		{economysim.ReasonOverflow, "overflow"},
		{economysim.ReasonHoldingLimit, "holding_limit"},
		{economysim.ReasonReceiptLimit, "receipt_limit"},
		{economysim.ReasonInsufficientFunds, "insufficient_funds"},
		{economysim.ReasonInsufficientStock, "insufficient_stock"},
		{economysim.ReasonAgentRemoved, "agent_removed"},
		{economysim.ReasonRoleChanged, "role_changed"},
		{economysim.ReasonInitial, "initial"},
		{economysim.ReasonRecipeInput, "recipe_input"},
		{economysim.ReasonRecipeOutput, "recipe_output"},
		{economysim.ReasonDisposal, "disposal"},
		{economysim.ReasonConsumption, "consumption"},
		{economysim.ReasonSpoilage, "spoilage"},
		{economysim.ReasonWear, "wear"},
		{economysim.ReasonTrade, "trade"},
		{economysim.ReasonEstateMoney, "estate_money"},
		{economysim.ReasonEstateGoods, "estate_goods"},
		{economysim.ReasonStartMoney, "start_money"},
		{economysim.ReasonSplit, "split"},
		{economysim.ReasonMerge, "merge"},
		{economysim.ReasonBirths, "births"},
		{economysim.ReasonStarvation, "starvation"},
		{economysim.ReasonWantConsumption, "want_consumption"},
	}
	// The belief reasons of the "belief updated" Debug line, spec-beliefs.md
	// 12.2: a slice of their own, checked by the same closure below.
	beliefReasons := []struct {
		got  economysim.Reason
		want string
	}{
		{economysim.ReasonGateClosed, "gate_closed"},
		{economysim.ReasonMoved, "moved"},
		{economysim.ReasonHeld, "held"},
		{economysim.ReasonAtFloor, "at_floor"},
		{economysim.ReasonAtCeiling, "at_ceiling"},
	}
	// The refusal reasons of a start in the first slice, spec-production.md
	// 4.2.1, in the order of its master table. A slice of their own, so the
	// index ranges of the owners of reasons above are not touched.
	productionReasons := []struct {
		got  economysim.Reason
		want string
	}{
		{economysim.ReasonNotOnRole, "not_on_role"},
		{economysim.ReasonZeroYield, "zero_yield"},
		{economysim.ReasonNoSite, "no_site"},
		{economysim.ReasonNoTool, "no_tool"},
		{economysim.ReasonNoInput, "no_input"},
	}
	// The log-only reasons of the "work declined" Debug line, spec-agents.md
	// 14.3: no event carries them.
	agentReasons := []struct {
		got  economysim.Reason
		want string
	}{
		{economysim.ReasonRest, "rest"},
		{economysim.ReasonHalted, "halted"},
		{economysim.ReasonNoRecipe, "no_recipe"},
	}
	// The reasons of a spawner decision the world refuses (spec-spawner.md 4)
	// and the two "entry dropped" Debug reasons (13.3), in the order the
	// constants are declared. unknown_agent is in the first slice.
	spawnerReasons := []struct {
		got  economysim.Reason
		want string
	}{
		{economysim.ReasonUnknownRole, "unknown_role"},
		{economysim.ReasonExternalAgent, "external_agent"},
		{economysim.ReasonSameRole, "same_role"},
		{economysim.ReasonBadHouseholds, "bad_households"},
		{economysim.ReasonUnsupportedCause, "unsupported_cause"},
		{economysim.ReasonUnfunded, "unfunded"},
		{economysim.ReasonNoViableRole, "no_viable_role"},
	}
	causes := []struct {
		got  economysim.Cause
		want string
	}{
		{economysim.CauseInsolvent, "insolvent"},
		{economysim.CauseStarved, "starved"},
		{economysim.CauseConsumer, "consumer"},
		{economysim.CauseDistress, "distress"},
		{economysim.CauseReview, "review"},
		{economysim.CauseRenewal, "renewal"},
	}
	kinds := []struct {
		got  economysim.Kind
		want string
	}{
		{economysim.KindFaucet, "faucet"},
		{economysim.KindSink, "sink"},
	}
	// The strings are also read from the specs that own them, by plain text:
	// each value must be a backticked token of its owner. The reasons are
	// the rejection reasons (the first ten, spec-clearing.md 3.2), the abandon
	// reasons (the next two, spec-production.md) and the ledger reasons.
	owner := map[string]string{}
	for i, r := range reasons {
		switch {
		case i < 12:
			owner[string(r.got)] = "doc/specs/spec-clearing.md"
		case i < 14:
			owner[string(r.got)] = "doc/specs/spec-production.md"
		default:
			owner[string(r.got)] = "doc/specs/spec-ledger.md"
		}
	}
	for _, r := range beliefReasons {
		owner[string(r.got)] = "doc/specs/spec-beliefs.md"
	}
	for _, r := range productionReasons {
		owner[string(r.got)] = "doc/specs/spec-production.md"
	}
	for _, r := range agentReasons {
		owner[string(r.got)] = "doc/specs/spec-agents.md"
	}
	for _, r := range spawnerReasons {
		owner[string(r.got)] = "doc/specs/spec-spawner.md"
	}
	for _, c := range causes {
		owner[string(c.got)] = "doc/specs/spec-spawner.md"
	}
	for _, k := range kinds {
		owner[string(k.got)] = "doc/specs/spec-observability.md"
	}
	specText := map[string]string{}
	for v, file := range owner {
		if _, ok := specText[file]; !ok {
			specText[file] = readFile(t, file)
		}
		if !strings.Contains(specText[file], "`"+v+"`") {
			t.Errorf("value %q is not a backticked token of %s", v, file)
		}
	}
	// Every rejection reason of the table of spec-clearing.md 3.2 is a
	// constant, or is named here as built with its module.
	deferredReasons := []string{"not_permitted", "no_room"}
	clearing := specText["doc/specs/spec-clearing.md"]
	from := strings.Index(clearing, "### 3.2 Validation")
	to := strings.Index(clearing, "Checks run in the order of the table")
	if from < 0 || to < from {
		t.Fatalf("table of 3.2 not found in spec-clearing.md")
	}
	// The two settlement rows say which bound the order would break (3.2.2,
	// D34): holding_limit names MaxQuantity and receipt_limit names MaxMoney.
	for reason, want := range map[string]string{"holding_limit": "MaxQuantity", "receipt_limit": "MaxMoney"} {
		row := regexp.MustCompile("(?m)^\\| `" + reason + "` \\|.*$").FindString(clearing[from:to])
		if !strings.Contains(row, want) {
			t.Errorf("%s row of spec-clearing.md 3.2 = %q, want it to name %s", reason, row, want)
		}
	}
	rowRe := regexp.MustCompile("(?m)^\\| `([a-z_]+)` \\|")
	for _, m := range rowRe.FindAllStringSubmatch(clearing[from:to], -1) {
		if m[1] == "reason" { // the header row
			continue
		}
		_, isConst := owner[m[1]]
		if !isConst && !slices.Contains(deferredReasons, m[1]) {
			t.Errorf("rejection reason %q of spec-clearing.md 3.2 has no constant and is not in deferredReasons", m[1])
		}
		if isConst && slices.Contains(deferredReasons, m[1]) {
			t.Errorf("deferredReasons lists %q, which is declared: delete the stale entry", m[1])
		}
	}
	// Every reason of the master order of spec-production.md 4.2.1 that a
	// slice start can return is a constant, in the table's order, and every
	// other row is named here as built with its module.
	deferredProduction := []string{
		"not_permitted", "no_facility", "no_endowment", "no_title", "rent_funds", "not_unlocked",
		"facility_cap", "no_target", "store_refused", "nothing_to_repair", "no_room",
	}
	prod := specText["doc/specs/spec-production.md"]
	pFrom := strings.Index(prod, "4.2.1. **The master order.**")
	pTo := strings.Index(prod, "A start of the first slice (the module-free core)")
	if pFrom < 0 || pTo < pFrom {
		t.Fatalf("master order table of 4.2.1 not found in spec-production.md")
	}
	var inSlice, notInSlice []string
	masterRe := regexp.MustCompile("(?m)^\\| \\d+ \\| `([a-z_]+)` \\|.*\\| (yes|no) \\|$")
	for _, m := range masterRe.FindAllStringSubmatch(prod[pFrom:pTo], -1) {
		if m[2] == "yes" {
			inSlice = append(inSlice, m[1])
		} else {
			notInSlice = append(notInSlice, m[1])
		}
	}
	var gotSlice []string
	for _, r := range productionReasons {
		gotSlice = append(gotSlice, r.want)
	}
	if !slices.Equal(inSlice, gotSlice) {
		t.Errorf("reasons marked yes in spec-production.md 4.2.1 = %v, constants (in order) = %v", inSlice, gotSlice)
	}
	if !slices.Equal(notInSlice, deferredProduction) {
		t.Errorf("reasons marked no in spec-production.md 4.2.1 = %v, deferredProduction = %v", notInSlice, deferredProduction)
	}
	for _, d := range deferredProduction {
		if _, ok := owner[d]; ok {
			t.Errorf("deferredProduction lists %q, which is declared: delete the stale entry", d)
		}
	}
	seen := map[string]bool{}
	check := func(typ, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s constant = %q, want %q", typ, got, want)
		}
		if key := typ + "/" + got; seen[key] {
			t.Errorf("%s value %q is declared twice", typ, got)
		} else {
			seen[key] = true
		}
	}
	for _, r := range reasons {
		check("Reason", string(r.got), r.want)
	}
	for _, r := range beliefReasons {
		check("Reason", string(r.got), r.want)
	}
	for _, r := range productionReasons {
		check("Reason", string(r.got), r.want)
	}
	for _, r := range agentReasons {
		check("Reason", string(r.got), r.want)
	}
	for _, r := range spawnerReasons {
		check("Reason", string(r.got), r.want)
	}
	for _, c := range causes {
		check("Cause", string(c.got), c.want)
	}
	for _, k := range kinds {
		check("Kind", string(k.got), k.want)
	}
}

// TestSide_String verifies the strings of the two sides, which are the values
// of the side log key and of the golden rendering, that Bid is the zero value,
// and that an out-of-range value still renders.
//
// Requirement: FR-OBS-006
func TestSide_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		side economysim.Side
		want string
	}{
		{"bid", economysim.Bid, "bid"},
		{"ask", economysim.Ask, "ask"},
		{"zero value is bid", economysim.Side(0), "bid"},
		{"out of range", economysim.Side(7), "side(7)"},
		{"negative", economysim.Side(-1), "side(-1)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.side.String(); got != tt.want {
				t.Errorf("Side(%d).String() = %q, want %q", int(tt.side), got, tt.want)
			}
		})
	}
	if economysim.Bid == economysim.Ask {
		t.Errorf("Bid == Ask = true, want distinct sides")
	}
}

// metricNames returns the names in the first column of the rows of text that
// start with "| `economysim.".
func metricNames(text string) map[string]bool {
	names := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, "| `economysim.") {
			continue
		}
		cell := strings.Split(line, "|")[1]
		if m := backtickRe.FindStringSubmatch(cell); m != nil {
			names[m[1]] = true
		}
	}
	return names
}

// TestMetricCatalogue_FeedTablesAgree verifies that every instrument of the
// catalogue of observability-and-logging.md 5.3 has a row in the feed tables of
// 5.3.1 and that every row there names an instrument of the catalogue, so a
// metric added to one table cannot be missing from the other.
//
// Guard: observability-and-logging.md 5.3.1 says a new metric must be derivable
// from an event or a snapshot field, and the tables say which. A metric with no
// feed row has no stated source, and a feed row with no instrument is dead text.
func TestMetricCatalogue_FeedTablesAgree(t *testing.T) {
	t.Parallel()
	text := readFile(t, "doc/rules/observability-and-logging.md")
	a := strings.Index(text, "### 5.3 Initial catalogue")
	b := strings.Index(text, "### 5.3.1 ")
	c := strings.Index(text, "### 5.4 ")
	if a < 0 || b < a || c < b {
		t.Fatalf("sections 5.3, 5.3.1 and 5.4 not found in order in observability-and-logging.md")
	}
	catalogue, feeds := metricNames(text[a:b]), metricNames(text[b:c])
	if len(catalogue) == 0 || len(feeds) == 0 {
		t.Fatalf("parsed %d instruments and %d feed rows, want both above 0", len(catalogue), len(feeds))
	}
	for n := range catalogue {
		if !feeds[n] {
			t.Errorf("instrument %s of 5.3 has no row in 5.3.1", n)
		}
	}
	for n := range feeds {
		if !catalogue[n] {
			t.Errorf("row %s of 5.3.1 is not an instrument of 5.3", n)
		}
	}
	for _, n := range []string{"economysim.commodity.wear", "economysim.ledger.lots"} {
		if !catalogue[n] {
			t.Errorf("instrument %s is missing from 5.3", n)
		}
	}
}

// TestEntryLimited_AppendedFieldsAreCounts verifies that the fields the spawner
// appended to EntryLimited (Stranded, and the counts it redefined) are
// int64 counts, not typed IDs.
//
// Requirement: FR-SPN-061
//
// Guard: architecture.md 2.10.5 lets a field be appended to an existing event
// only as a count, an amount, a bool, a closed-set constant or a slice, because
// the golden rendering omits a field at zero and zero is a valid ID. A typed ID
// here would make a world with the module off render differently and move
// testdata/golden/off/.
func TestEntryLimited_AppendedFieldsAreCounts(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf(economysim.EntryLimited{})
	for _, name := range []string{"Stranded", "Cooldown", "Hysteresis", "ExitCap"} {
		f, ok := typ.FieldByName(name)
		if !ok {
			t.Errorf("EntryLimited has no field %s", name)
			continue
		}
		if f.Type.Kind() != reflect.Int64 || f.Type.Name() != "int64" {
			t.Errorf("EntryLimited.%s has type %v, want int64", name, f.Type)
		}
	}
}
