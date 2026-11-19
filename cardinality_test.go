package economysim_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// eventConstants parses event.go and returns, per Go type name (Reason, Cause,
// Kind), the string values of its constants. Go reflection cannot list them.
func eventConstants(t *testing.T) map[string]map[string]bool {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "event.go", nil, 0)
	if err != nil {
		t.Fatalf("parse event.go: %v", err)
	}
	out := map[string]map[string]bool{}
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, s := range g.Specs {
			vs := s.(*ast.ValueSpec)
			id, ok := vs.Type.(*ast.Ident)
			if !ok || len(vs.Values) != 1 {
				continue // Side's iota constants and untyped ones
			}
			lit, ok := vs.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			u, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Fatalf("unquote %s: %v", lit.Value, err)
			}
			if out[id.Name] == nil {
				out[id.Name] = map[string]bool{}
			}
			out[id.Name][u] = true
		}
	}
	for _, typ := range []string{"Reason", "Cause", "Kind", "FacilityKind"} {
		if len(out[typ]) == 0 {
			t.Fatalf("event.go declares no %s constants; the parser walk is stale", typ)
		}
	}
	return out
}

// cardChecker collects the violations of the closed sets.
type cardChecker struct {
	t interface {
		Errorf(format string, args ...any)
	} // *testing.T, or a counter in the unit test
	seed   uint64
	names  economysim.Names
	consts map[string]map[string]bool
	// flows are the IDs of the money flows of the configuration: the Reason of a
	// BoundaryFlow may be one, and the Flow of a money flow event must be (MON 4.5.2,
	// observability-and-logging.md 5.4.1).
	flows map[string]bool
	seen  map[string]int
}

// flow checks that a money flow event names a flow of the configuration.
func (c *cardChecker) flow(ev string, tick int64, id string) {
	c.seen["Flow"]++
	if !c.flows[id] {
		c.t.Errorf("seed %d tick %d %s.Flow = %q, not a flow of the configuration", c.seed, tick, ev, id)
	}
}

func (c *cardChecker) id(ev string, tick int64, field string, v, n int) {
	c.seen[field]++
	if v < 0 || v >= n {
		c.t.Errorf("seed %d tick %d %s.%s = %d, want 0 <= id < %d", c.seed, tick, ev, field, v, n)
	}
}

func (c *cardChecker) member(ev string, tick int64, typ, v string) {
	c.seen[typ]++
	if !c.consts[typ][v] {
		c.t.Errorf("seed %d tick %d %s %s = %q, not a constant of %s in event.go", c.seed, tick, ev, typ, v, typ)
	}
}

// facility checks a FacilityChanged: its region, its class, its kind and its cause.
func (c *cardChecker) facility(e economysim.FacilityChanged) {
	c.region("FacilityChanged", e.Tick, e.Region)
	c.id("FacilityChanged", e.Tick, "Class", int(e.Class), len(c.names.FacilityClasses))
	c.member("FacilityChanged", e.Tick, "FacilityKind", string(e.Kind))
	c.member("FacilityChanged", e.Tick, "Cause", string(e.Cause))
}

func (c *cardChecker) side(ev string, tick int64, s economysim.Side) {
	c.seen["Side"]++
	if s != economysim.Bid && s != economysim.Ask {
		c.t.Errorf("seed %d tick %d %s.Side = %d, want Bid or Ask", c.seed, tick, ev, int(s))
	}
}

func (c *cardChecker) region(ev string, tick int64, r economysim.RegionID) {
	c.id(ev, tick, "Region", int(r), len(c.names.Regions))
}
func (c *cardChecker) commodity(ev string, tick int64, v economysim.CommodityID) {
	c.id(ev, tick, "Commodity", int(v), len(c.names.Commodities))
}
func (c *cardChecker) role(ev string, tick int64, v economysim.RoleID) {
	c.id(ev, tick, "Role", int(v), len(c.names.Roles))
}
func (c *cardChecker) recipe(ev string, tick int64, v economysim.RecipeID) {
	c.id(ev, tick, "Recipe", int(v), len(c.names.Recipes))
}

// jurisdictions checks the rows of WorldCreated or of a snapshot: the jurisdiction is
// below len(Names.Jurisdictions) and the seat is a region (spec-polity.md 10.5, 10.7).
func (c *cardChecker) jurisdictions(ev string, tick int64, rows []economysim.JurisdictionRow) {
	for _, r := range rows {
		c.id(ev, tick, "Jurisdiction", int(r.Jurisdiction), len(c.names.Jurisdictions))
		c.region(ev, tick, r.Seat)
	}
}

// check applies the closed-set checks of OBS 6.2.1 to one event.
func (c *cardChecker) check(e economysim.Event) {
	c.checkStorage(e)
	switch e := e.(type) {
	case economysim.WorldCreated:
		c.jurisdictions("WorldCreated", e.Tick, e.Jurisdictions)
	case economysim.OrderPlaced:
		c.region("OrderPlaced", e.Tick, e.Region)
		c.commodity("OrderPlaced", e.Tick, e.Commodity)
		c.side("OrderPlaced", e.Tick, e.Side)
	case economysim.OrderRejected:
		c.region("OrderRejected", e.Tick, e.Region)
		c.commodity("OrderRejected", e.Tick, e.Commodity)
		c.side("OrderRejected", e.Tick, e.Side)
		c.member("OrderRejected", e.Tick, "Reason", string(e.Reason))
	case economysim.TradeExecuted:
		c.region("TradeExecuted", e.Tick, e.Region)
		c.commodity("TradeExecuted", e.Tick, e.Commodity)
	case economysim.MarketCleared:
		c.region("MarketCleared", e.Tick, e.Region)
		c.commodity("MarketCleared", e.Tick, e.Commodity)
	case economysim.BeliefsUpdated:
		c.region("BeliefsUpdated", e.Tick, e.Region)
		c.commodity("BeliefsUpdated", e.Tick, e.Commodity)
	case economysim.RecipeExecuted:
		c.region("RecipeExecuted", e.Tick, e.Region)
		c.recipe("RecipeExecuted", e.Tick, e.Recipe)
		for _, a := range append(append(append([]economysim.Amount{}, e.Outputs...), e.Disposed...), e.Overflow...) {
			c.commodity("RecipeExecuted", e.Tick, a.Commodity)
		}
	case economysim.RecipeRefused:
		c.region("RecipeRefused", e.Tick, e.Region)
		c.recipe("RecipeRefused", e.Tick, e.Recipe)
	case economysim.RecipeAbandoned:
		c.region("RecipeAbandoned", e.Tick, e.Region)
		c.recipe("RecipeAbandoned", e.Tick, e.Recipe)
		c.member("RecipeAbandoned", e.Tick, "Reason", string(e.Reason))
	case economysim.ProductionHalted:
		c.region("ProductionHalted", e.Tick, e.Region)
		c.role("ProductionHalted", e.Tick, e.Role)
	case economysim.UpkeepShortfall:
		c.region("UpkeepShortfall", e.Tick, e.Region)
		c.role("UpkeepShortfall", e.Tick, e.Role)
	case economysim.AgentSpawned:
		c.region("AgentSpawned", e.Tick, e.Region)
		c.role("AgentSpawned", e.Tick, e.Role)
	case economysim.AgentBankrupted:
		c.region("AgentBankrupted", e.Tick, e.Region)
		c.role("AgentBankrupted", e.Tick, e.Role)
		c.member("AgentBankrupted", e.Tick, "Cause", string(e.Cause))
	case economysim.EstateOpened:
		c.region("EstateOpened", e.Tick, e.Region)
	case economysim.WeightChanged:
		c.region("WeightChanged", e.Tick, e.Region)
		c.role("WeightChanged", e.Tick, e.Role)
		c.member("WeightChanged", e.Tick, "Cause", string(e.Cause))
	case economysim.RoleChanged:
		c.region("RoleChanged", e.Tick, e.Region)
		c.role("RoleChanged", e.Tick, e.From)
		c.role("RoleChanged", e.Tick, e.To)
		c.member("RoleChanged", e.Tick, "Cause", string(e.Cause))
	case economysim.EntryLimited:
		c.region("EntryLimited", e.Tick, e.Region)
	case economysim.BoundaryFlow:
		c.region("BoundaryFlow", e.Tick, e.Region)
		if e.Commodity != economysim.NoCommodity { // money flows carry NoCommodity
			c.commodity("BoundaryFlow", e.Tick, e.Commodity)
		}
		if c.flows[string(e.Reason)] {
			c.seen["Flow"]++
		} else {
			c.member("BoundaryFlow", e.Tick, "Reason", string(e.Reason))
		}
		c.member("BoundaryFlow", e.Tick, "Kind", string(e.Kind))
	case economysim.MoneyFlowLimited:
		c.region("MoneyFlowLimited", e.Tick, e.Region)
		c.flow("MoneyFlowLimited", e.Tick, e.Flow)
	case economysim.MoneyFlowEnded:
		c.flow("MoneyFlowEnded", e.Tick, e.Flow)
		c.member("MoneyFlowEnded", e.Tick, "Cause", string(e.Cause))
	case economysim.MoneyFlowScaled:
		c.flow("MoneyFlowScaled", e.Tick, e.Flow)
		c.member("MoneyFlowScaled", e.Tick, "Cause", string(e.Cause))
	case economysim.WearApplied:
		c.commodity("WearApplied", e.Tick, e.Commodity)
	case economysim.OutsideTraded:
		c.region("OutsideTraded", e.Tick, e.Node)
		c.region("OutsideTraded", e.Tick, e.Region)
		c.commodity("OutsideTraded", e.Tick, e.Commodity)
		c.side("OutsideTraded", e.Tick, e.Side)
	case economysim.OutsideEnded:
		c.region("OutsideEnded", e.Tick, e.Node)
		c.region("OutsideEnded", e.Tick, e.Region)
		c.commodity("OutsideEnded", e.Tick, e.Commodity)
		c.side("OutsideEnded", e.Tick, e.Side)
		c.member("OutsideEnded", e.Tick, "Cause", string(e.Cause))
	case economysim.OutsideShocked:
		c.region("OutsideShocked", e.Tick, e.Node)
		c.commodity("OutsideShocked", e.Tick, e.Commodity)
	case economysim.YieldShocked:
		c.region("YieldShocked", e.Tick, e.Region)
		c.recipe("YieldShocked", e.Tick, e.Recipe)
	case economysim.PopulationChanged:
		c.region("PopulationChanged", e.Tick, e.Region)
		c.role("PopulationChanged", e.Tick, e.Role)
	case economysim.PopulationScaled:
		c.region("PopulationScaled", e.Tick, e.Region)
	case economysim.TitleTransferred:
		c.region("TitleTransferred", e.Tick, e.Region)
		c.id("TitleTransferred", e.Tick, "Resource", int(e.Resource), len(c.names.Resources))
		c.member("TitleTransferred", e.Tick, "Cause", string(e.Cause))
	case economysim.FacilityChanged:
		c.facility(e)
	case economysim.RentScaled:
		c.region("RentScaled", e.Tick, e.Region)
		c.id("RentScaled", e.Tick, "Resource", int(e.Resource), len(c.names.Resources))
	case economysim.RentPaid:
		c.region("RentPaid", e.Tick, e.Region)
		c.id("RentPaid", e.Tick, "Resource", int(e.Resource), len(c.names.Resources))
		for _, rm := range e.From {
			c.region("RentPaid", e.Tick, rm.Region)
		}
		for _, rm := range e.To {
			c.region("RentPaid", e.Tick, rm.Region)
		}
	case economysim.FiscalFlowApplied:
		c.region("FiscalFlowApplied", e.Tick, e.Region)
		c.id("FiscalFlowApplied", e.Tick, "Jurisdiction", int(e.Authority), len(c.names.Jurisdictions))
		c.member("FiscalFlowApplied", e.Tick, "Kind", string(e.Kind))
		c.member("FiscalFlowApplied", e.Tick, "FiscalClass", string(e.Class))
	case economysim.RouteRiskScaled:
		c.region("RouteRiskScaled", e.Tick, e.From)
		c.region("RouteRiskScaled", e.Tick, e.To)
		c.id("RouteRiskScaled", e.Tick, "Mode", int(e.Mode), len(c.names.Modes))
	case economysim.StoreRiskScaled:
		c.region("StoreRiskScaled", e.Tick, e.Region)
		c.id("StoreRiskScaled", e.Tick, "Class", int(e.Class), len(c.names.StoreClasses))
	case economysim.StoreLoss:
		c.region("StoreLoss", e.Tick, e.Region)
		c.id("StoreLoss", e.Tick, "Class", int(e.Class), len(c.names.StoreClasses))
	}
}

// checkCardinality runs cfg for 400 ticks on one seed with the recorder and checks every
// event against the closed sets (see TestCardinality_EventAttributesTakeClosedValues).
func checkCardinality(t *testing.T, cfg economysim.Config, seed uint64, opts []economysim.Option) {
	t.Helper()
	res := runPack(t, cfg, runSpec{Seed: seed, Ticks: 400, Observe: true, Extra: opts})
	flows := map[string]bool{}
	if cfg.Modules.MoneySupply != nil {
		for _, f := range cfg.Modules.MoneySupply.Flows {
			flows[f.ID] = true
		}
	}
	c := &cardChecker{t: t, seed: seed, names: res.Final.Names, consts: eventConstants(t), flows: flows, seen: map[string]int{}}
	for _, e := range res.Events {
		c.check(e)
	}
	for _, f := range []string{"Region", "Commodity", "Role", "Recipe", "Side", "Reason", "Cause", "Kind"} {
		t.Logf("seed %d: %s checked %d times", seed, f, c.seen[f])
	}
	// Kind and Reason are always reachable; the others depend on collapse, so only these are required.
	for _, f := range []string{"Region", "Commodity", "Role", "Recipe", "Side", "Kind"} {
		if c.seen[f] == 0 {
			t.Errorf("seed %d: no %s value was checked; the run exercised nothing", seed, f)
		}
	}
}

// TestCardinality_EventAttributesTakeClosedValues runs pack one for 400 ticks
// on seeds 1 and 2 and checks that every event field that becomes a metric
// attribute takes a value of the configuration (typed IDs below len of the names
// table) or of a closed set (Side is Bid or Ask; Reason, Cause and Kind are
// constants of event.go). The Reason check is weaker by design: Reason is one Go
// type holding several closed sets, so a value need only be a member of any Reason
// constant; a per-event partition is not attempted.
//
// Requirement: FR-OBS-005, FR-OBS-006, FR-REF-034
// Guard: a RecipeExecuted with a recipe ID out of range, or a Reason outside event.go.
func TestCardinality_EventAttributesTakeClosedValues(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end: 2 x 400 ticks of pack one; skipped under -short")
	}
	cfg := loadValidPack(t, packOne...)
	for _, seed := range []uint64{1, 2} {
		t.Run("seed"+strconv.FormatUint(seed, 10), func(t *testing.T) {
			checkCardinality(t, cfg, seed, nil)
		})
	}
}

// TestPacks_FiveAndAllEventAttributesTakeClosedValues is the check of
// TestCardinality_EventAttributesTakeClosedValues on the warm five (every module off) and
// on the warm five with the tenure overlays and on the all-on pack, seed 1, 400 ticks: the configuration of five that N-295 left to
// Stage 2 (plan 2-24 Phase 2). The all-on row is the one that emits the events of the
// modules (transport, outside node, migration, merchants, demand). It skips under -race with
// the other 5 000-household runs, and its name starts with a prefix of gateFPrefixes.
//
// Requirement: FR-OBS-005, FR-OBS-006, FR-REF-034
// Guard: a Region, Commodity or Mode ID out of range, or a Reason outside event.go and the
// flows of the configuration, on an event of a module.
func TestPacks_FiveAndAllEventAttributesTakeClosedValues(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	for _, row := range []struct {
		pack   string
		layers []string
		opts   []economysim.Option
	}{{"five", packFiveWarm, nil}, {"five+tenure", packFiveTenure, nil}, {"all", allLayers, allOptions()}} {
		cfg := loadValidPack(t, row.layers...)
		t.Run(row.pack, func(t *testing.T) {
			t.Parallel()
			checkCardinality(t, cfg, 1, row.opts)
		})
	}
}

// TestCardinality_MoneyFlowAttributesTakeConfiguredValues runs pack one with the
// money supply module on (overlay/faucet-sunk.toml, with a limit on the stipend so
// the flow ends in the run) and checks that the Reason of a BoundaryFlow is a
// built-in reason or a flow of the configuration, and that the Flow, Region and
// Cause of the three money flow events take values of the configuration or of
// their closed sets: the flow IDs are at most MaxMoneyFlows, so the metric
// attribute economysim.reason stays bounded by the configuration.
//
// Requirement: FR-OBS-005, FR-OBS-006, FR-MON-003, FR-MON-029, FR-MON-031
//
// Guard: a flow reason that reached BoundaryFlow under a name that is not an
// ID of the configuration would be an unbounded attribute value.
func TestCardinality_MoneyFlowAttributesTakeConfiguredValues(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end: 120 ticks of pack one; skipped under -short")
	}
	cfg := loadValidPack(t, packOneFaucetSunk...)
	cfg.Modules.MoneySupply.Flows[1].Limit = 300_000 // stipend (flows are in ID order): 5 060 a tick, so it ends at tick 59
	flows := map[string]bool{}
	for _, f := range cfg.Modules.MoneySupply.Flows {
		flows[f.ID] = true
	}
	res := runPack(t, cfg, runSpec{Seed: 1, Ticks: 120, Observe: true})
	c := &cardChecker{t: t, seed: 1, names: res.Final.Names, consts: eventConstants(t), flows: flows, seen: map[string]int{}}
	ended := 0
	for _, e := range res.Events {
		c.check(e)
		if _, ok := e.(economysim.MoneyFlowEnded); ok {
			ended++
		}
	}
	if c.seen["Flow"] == 0 || ended != 1 {
		t.Errorf("flow values checked %d times and %d MoneyFlowEnded events, want some and exactly 1: the run exercised nothing", c.seen["Flow"], ended)
	}
}

// TestCardinality_ExternalAttributesTakeClosedValues runs pack one with the external
// overlay for 60 ticks and checks that the node, gate and commodity of OutsideTraded
// and OutsideEnded are IDs of the configuration, their side is Bid or Ask and their
// cause is a constant of event.go: the metric attributes economysim.node, region,
// commodity and side stay bounded by the configuration.
//
// Requirement: FR-OBS-005, FR-OBS-006, FR-EXT-036
//
// Guard: an ID out of range or a free-text cause in an outside event would be an
// unbounded attribute value.
func TestCardinality_ExternalAttributesTakeClosedValues(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end: 60 ticks of pack one with the external overlay; skipped under -short")
	}
	cfg := loadValidPack(t, packExternal()...)
	res := runPack(t, cfg, runSpec{Seed: 1, Ticks: 60, Observe: true})
	c := &cardChecker{t: t, seed: 1, names: res.Final.Names, consts: eventConstants(t), seen: map[string]int{}}
	traded := 0
	for _, e := range res.Events {
		c.check(e)
		if _, ok := e.(economysim.OutsideTraded); ok {
			traded++
		}
	}
	if traded == 0 {
		t.Error("no OutsideTraded in 60 ticks, want the module exercised")
	}
}

// TestCardinality_PolityAttributesTakeConfiguredValues runs the warm five with the
// governed polity overlay for 30 ticks and checks that the Jurisdiction and Seat of
// each row of WorldCreated and of the final snapshot are IDs of the names table: the
// metric attributes economysim.region (and a jurisdiction label an adapter derives
// from the names table) stay bounded by the configuration.
//
// Requirement: FR-OBS-005, FR-OBS-006, FR-POL-028, FR-POL-035
//
// Guard: a jurisdiction or seat index out of range in a row would be an unresolvable
// attribute value.
func TestCardinality_PolityAttributesTakeConfiguredValues(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	cfg := loadValidPack(t, fivePolityGoverned...)
	res := runPack(t, cfg, runSpec{Seed: 1, Ticks: 30, Observe: true})
	c := &cardChecker{t: t, seed: 1, names: res.Final.Names, consts: eventConstants(t), seen: map[string]int{}}
	rows := 0
	for _, e := range res.Events {
		c.check(e)
		if wc, ok := e.(economysim.WorldCreated); ok {
			rows += len(wc.Jurisdictions)
		}
	}
	c.jurisdictions("Snapshot", res.Final.Tick, res.Final.Jurisdictions)
	if rows == 0 || len(res.Final.Jurisdictions) != rows {
		t.Errorf("WorldCreated rows = %d, snapshot rows = %d, want the same non-zero count: the run exercised nothing", rows, len(res.Final.Jurisdictions))
	}
	if c.seen["Jurisdiction"] != 2*rows {
		t.Errorf("Jurisdiction ids checked %d times, want %d", c.seen["Jurisdiction"], 2*rows)
	}
}

// errCount counts the Errorf calls of a checker under test.
type errCount struct{ n int }

func (e *errCount) Errorf(string, ...any) { e.n++ }

// TestCardChecker_FlagsOutOfRangeAndOutsideSet verifies the checker itself on
// hand-built events, so a checker that accepts everything cannot hide behind a
// clean 400-tick run: a RecipeExecuted with a recipe ID at len(Recipes) or
// below 0 is flagged, one in range is not; a Reason outside event.go is flagged.
//
// Requirement: FR-OBS-005, FR-OBS-006
// Guard: without it the end-to-end cardinality test would pass if check() ever
// stopped comparing IDs to the names table.
func TestCardChecker_FlagsOutOfRangeAndOutsideSet(t *testing.T) {
	t.Parallel()
	names := economysim.Names{FacilityClasses: []string{"f"}, Regions: []string{"r"}, Commodities: []string{"c"}, Roles: []string{"o"}, Recipes: []string{"a", "b"}, Jurisdictions: []string{"j"}, Resources: []string{"x"}}
	consts := map[string]map[string]bool{"Reason": {"ok": true}, "Cause": {"removal": true, "consumer": true, "recipe": true}, "Kind": {}, "FacilityKind": {"built": true}}
	tests := []struct {
		name string
		ev   economysim.Event
		want bool // flagged
	}{
		{"recipe in range", economysim.RecipeExecuted{Recipe: 1}, false},
		{"recipe at len", economysim.RecipeExecuted{Recipe: 2}, true},
		{"recipe negative", economysim.RecipeExecuted{Recipe: -1}, true},
		{"region out of range", economysim.RecipeExecuted{Region: 1}, true},
		{"output commodity out of range", economysim.RecipeExecuted{Outputs: []economysim.Amount{{Commodity: 1}}}, true},
		{"reason in set", economysim.OrderRejected{Reason: "ok"}, false},
		{"reason outside set", economysim.OrderRejected{Reason: "nope"}, true},
		{"outside side in set", economysim.OutsideTraded{Side: economysim.Ask}, false},
		{"outside side out of set", economysim.OutsideTraded{Side: economysim.Side(2)}, true},
		{"outside node out of range", economysim.OutsideShocked{Node: 1}, true},
		{"world created row in range", economysim.WorldCreated{Jurisdictions: []economysim.JurisdictionRow{{Jurisdiction: 0, Seat: 0}}}, false},
		{"world created jurisdiction at len", economysim.WorldCreated{Jurisdictions: []economysim.JurisdictionRow{{Jurisdiction: 1}}}, true},
		{"world created jurisdiction negative", economysim.WorldCreated{Jurisdictions: []economysim.JurisdictionRow{{Jurisdiction: -1}}}, true},
		{"world created seat out of range", economysim.WorldCreated{Jurisdictions: []economysim.JurisdictionRow{{Seat: 1}}}, true},
		{"outside cause outside set", economysim.OutsideEnded{Cause: economysim.CauseUnits}, true},
		{"title cause in set", economysim.TitleTransferred{Cause: economysim.CauseRemoval}, false},
		{"title cause outside set", economysim.TitleTransferred{Cause: "gift"}, true},
		{"facility kind and cause in sets", economysim.FacilityChanged{Kind: economysim.FacilityBuilt, Cause: economysim.CauseRecipe}, false},
		{"facility kind outside set", economysim.FacilityChanged{Kind: "gifted", Cause: economysim.CauseRecipe}, true},
		{"facility cause outside set", economysim.FacilityChanged{Kind: economysim.FacilityBuilt, Cause: "gift"}, true},
		{"title resource out of range", economysim.TitleTransferred{Resource: 1, Cause: economysim.CauseConsumer}, true},
		{"rent scale resource out of range", economysim.RentScaled{Resource: 1}, true},
		{"route risk scale mode out of range", economysim.RouteRiskScaled{Mode: 1}, true},
		{"route risk scale region out of range", economysim.RouteRiskScaled{To: 1}, true},
		{"store risk scale class out of range", economysim.StoreRiskScaled{Class: 1}, true},
		{"store loss class out of range", economysim.StoreLoss{Class: 1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var probe errCount
			c := &cardChecker{t: &probe, names: names, consts: consts, seen: map[string]int{}}
			c.check(tt.ev)
			if got := probe.n > 0; got != tt.want {
				t.Errorf("check(%+v) flagged = %t, want %t", tt.ev, got, tt.want)
			}
		})
	}
}

// TestCardinality_StorageAttributesTakeConfiguredValues runs pack one with the
// storage overlay for 400 ticks and checks that the
// region and role of RoomLimited and StoreChanged are IDs of the configuration, the
// store class of StoreChanged is a class of the configuration and its kind and cause
// are constants of event.go: the metric attributes economysim.region, role and
// store_class stay bounded by the configuration (observability-and-logging.md 5.4.1).
// The room limit and the dimension are closed sets of the adapter.
//
// Requirement: FR-OBS-005, FR-OBS-006, FR-STO-035, FR-STO-036
//
// Guard: a class index out of range or a free-text cause in a storage event would be
// an unbounded attribute value.
func TestCardinality_StorageAttributesTakeConfiguredValues(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end: 400 ticks of pack one; skipped under -short")
	}
	res := runPack(t, loadValidPack(t, packOneStorage...), runSpec{Seed: 1, Ticks: 400, Observe: true})
	c := &cardChecker{t: t, seed: 1, names: res.Final.Names, consts: eventConstants(t), seen: map[string]int{}}
	limited := 0
	for _, e := range res.Events {
		c.check(e)
		if _, ok := e.(economysim.RoomLimited); ok {
			limited++
		}
	}
	if limited == 0 || c.seen["Region"] == 0 {
		t.Errorf("%d RoomLimited events and %d region values checked, want some: the run exercised nothing", limited, c.seen["Region"])
	}
}

// checkStorage applies the closed-set checks to the events of the storage module: the
// region and role of RoomLimited and StoreChanged, the class of StoreChanged (a class of
// the configuration) and its kind and cause.
func (c *cardChecker) checkStorage(e economysim.Event) {
	switch e := e.(type) {
	case economysim.RoomLimited:
		c.region("RoomLimited", e.Tick, e.Region)
		c.role("RoomLimited", e.Tick, e.Role)
	case economysim.StoreChanged:
		c.region("StoreChanged", e.Tick, e.Region)
		c.role("StoreChanged", e.Tick, e.Role)
		c.id("StoreChanged", e.Tick, "Class", int(e.Class), len(c.names.StoreClasses))
		c.member("StoreChanged", e.Tick, "Kind", string(e.Kind))
		c.member("StoreChanged", e.Tick, "Cause", string(e.Cause))
	}
}

// TestCardinality_TierAttribute verifies FR-LND-031: the attribute economysim.tier of the
// rent metrics has three values in observability-and-logging.md 5.4.1, own, commons and
// let, and RentPaid carries one unit count for each (OwnUnits, CommonsUnits, LetUnits), so
// the instrument economysim.rent.units is bounded by the configuration and by that set.
//
// Requirement: FR-LND-031
//
// Guard: a fourth tier added to the event without the closed set of the rule file, or a set
// that no field feeds, would make the attribute unbounded or empty.
func TestCardinality_TierAttribute(t *testing.T) {
	t.Parallel()
	rules := readFile(t, "doc/rules/observability-and-logging.md")
	if !strings.Contains(rules, "`economysim.tier` (`own`, `commons`, `let`)") {
		t.Errorf("observability-and-logging.md 5.4.1 does not list economysim.tier as own, commons and let")
	}
	typ := reflect.TypeFor[economysim.RentPaid]()
	for _, f := range []string{"OwnUnits", "CommonsUnits", "LetUnits"} {
		if _, ok := typ.FieldByName(f); !ok {
			t.Errorf("RentPaid has no field %s, want one unit count for each value of the attribute tier", f)
		}
	}
}
