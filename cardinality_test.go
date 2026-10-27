package economysim_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
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
	for _, typ := range []string{"Reason", "Cause", "Kind"} {
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

// check applies the closed-set checks of OBS 6.2.1 to one event.
func (c *cardChecker) check(e economysim.Event) {
	switch e := e.(type) {
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
	}
}

// TestCardinality_EventAttributesTakeClosedValues runs pack one for 400 ticks
// on seeds 1 and 2 and checks that every event field that becomes a metric
// attribute takes a value of the configuration (typed IDs below len of the
// names table) or of a closed set (Side is Bid or Ask; Reason, Cause and Kind
// are constants of event.go). The Reason check is weaker by design: Reason is
// one Go type holding several closed sets, so a value need only be a member of
// any Reason constant; a per-event partition is not attempted.
//
// Requirement: FR-OBS-005, FR-OBS-006, FR-REF-034 (the configuration of five waits for Stage 2, N-295).
// Guard: a RecipeExecuted with a recipe ID out of range, or a Reason outside event.go.
func TestCardinality_EventAttributesTakeClosedValues(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end: 2 x 400 ticks of pack one; skipped under -short")
	}
	consts := eventConstants(t)
	cfg := loadValidPack(t, packOne...)
	for _, seed := range []uint64{1, 2} {
		t.Run("seed"+strconv.FormatUint(seed, 10), func(t *testing.T) {
			res := runPack(t, cfg, runSpec{Seed: seed, Ticks: 400, Observe: true})
			c := &cardChecker{t: t, seed: seed, names: res.Final.Names, consts: consts, seen: map[string]int{}}
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
	names := economysim.Names{Regions: []string{"r"}, Commodities: []string{"c"}, Roles: []string{"o"}, Recipes: []string{"a", "b"}}
	consts := map[string]map[string]bool{"Reason": {"ok": true}, "Cause": {}, "Kind": {}}
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
