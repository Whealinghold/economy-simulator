package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the external module's configuration surface (spec-external-sector.md
// 3, 13, 14; plan 2-16 Phase 4): the types, the bounds, the validation errors and
// warnings, the import source and the hash. Its effect on a world is tested in
// external_internal_test.go.

// externalConfig returns referenceConfig with two more regions, coast (the
// outside node) and far (a region the node has no edge to), edges of 30 between
// north and coast in both directions, and the module on with one node: coast
// sells mineral (an import: base 837, ask 5 % above the mid, 10 units a tick)
// and buys staple (an export: base 96, bid 5 % below the mid, 20 units a tick,
// open in spring and summer only). The calendar's seasons are spring, summer,
// autumn and winter.
func externalConfig() economysim.Config {
	c := referenceConfig()
	for _, id := range []string{"coast", "far"} {
		r := economysim.DefaultRegion()
		r.ID = id
		c.Regions = append(c.Regions, r)
	}
	c.Edges = []economysim.Edge{{From: "north", To: "coast", Distance: 30}, {From: "coast", To: "north", Distance: 30}}
	ore := economysim.DefaultOutsideLine("mineral", 837)
	ore.Sell = economysim.OutsideSide{MarginBP: 500, Capacity: 10}
	grain := economysim.DefaultOutsideLine("staple", 96)
	grain.Buy = economysim.OutsideSide{MarginBP: 500, Capacity: 20, Window: []economysim.SeasonFactor{
		{Season: "autumn", BP: 0}, {Season: "winter", BP: 0},
	}}
	c.Modules.External = &economysim.External{Nodes: []economysim.OutsideNode{{
		Region: "coast", Lines: []economysim.OutsideLine{ore, grain},
	}}}
	return c
}

// externalPaths is the path of the first line of the node of externalConfig.
var (
	externalNode = []string{"modules", "external", "node", "coast"}
	externalLine = []string{"modules", "external", "node", "coast", "line", "mineral"}
)

// externalFieldCases are the failing cases of the rows of the external module in
// the parameter-owner table (spec-external-sector.md 13.1, 14.1, FR-CFG-075):
// each breaks one field of the valid configuration of externalConfig.
// buildFieldCases appends them to the table that the registry test reads.
func externalFieldCases() []fieldCase {
	line := func(field, name string, rule error, path []string, value, bound string, set func(c *economysim.Config, l *economysim.OutsideLine)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{path}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = externalConfig()
				set(c, &c.Modules.External.Nodes[0].Lines[0])
			}}
	}
	node := func(field, name string, rule error, path []string, value, bound string, set func(c *economysim.Config, n *economysim.OutsideNode)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{path}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = externalConfig()
				set(c, &c.Modules.External.Nodes[0])
			}}
	}
	out := economysim.ErrOutOfBounds
	field := func(name string) []string { return append(slices.Clone(externalLine), name) }
	side := func(side, name string) []string { return append(slices.Clone(externalLine), side, name) }
	gate := func(region, name string) []string { return append(slices.Clone(externalNode), "gate", region, name) }
	return []fieldCase{
		node("External.Nodes", "17 nodes", out, []string{"modules", "external", "node"}, "17", "16",
			func(c *economysim.Config, n *economysim.OutsideNode) {
				c.Modules.External.Nodes = slices.Repeat([]economysim.OutsideNode{*n}, economysim.MaxOutsideNodes+1)
			}),
		node("OutsideNode.Region", "an undefined region", economysim.ErrUnknownRegion, []string{"modules", "external", "node", "nowhere", "region"}, "nowhere", "",
			func(c *economysim.Config, n *economysim.OutsideNode) { n.Region = "nowhere" }),
		node("OutsideNode.Lines", "65 lines", out, append(slices.Clone(externalNode), "line"), "65", "64",
			func(c *economysim.Config, n *economysim.OutsideNode) {
				n.Lines = slices.Repeat(n.Lines[:1], economysim.MaxOutsideLines+1)
			}),
		node("OutsideNode.Label", "label over 256 bytes", out, append(slices.Clone(externalNode), "label"), "257", "256",
			func(c *economysim.Config, n *economysim.OutsideNode) { n.Label = strings.Repeat("x", 257) }),
		node("OutsideNode.Gates", "a gate row for a region with no edge to the node", economysim.ErrNotAGate, gate("far", "region"), "far", "coast",
			func(c *economysim.Config, n *economysim.OutsideNode) {
				n.Gates = []economysim.OutsideGate{economysim.DefaultOutsideGate("far")}
			}),
		node("OutsideGate.Region", "an undefined region", economysim.ErrUnknownRegion, gate("nowhere", "region"), "nowhere", "",
			func(c *economysim.Config, n *economysim.OutsideNode) {
				n.Gates = []economysim.OutsideGate{economysim.DefaultOutsideGate("nowhere")}
			}),
		node("OutsideGate.ScaleBP", "0", out, gate("north", "scale_bp"), "0", "1",
			func(c *economysim.Config, n *economysim.OutsideNode) {
				n.Gates = []economysim.OutsideGate{{Region: "north"}}
			}),
		node("OutsideGate.ScaleBP", "above MaxGateScaleBP", out, gate("north", "scale_bp"), "100001", "100000",
			func(c *economysim.Config, n *economysim.OutsideNode) {
				n.Gates = []economysim.OutsideGate{{Region: "north", ScaleBP: economysim.MaxGateScaleBP + 1}}
			}),
		line("OutsideLine.Commodity", "an undefined commodity", economysim.ErrUnknownCommodity, []string{"modules", "external", "node", "coast", "line", "ghost", "commodity"}, "ghost", "",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.Commodity = "ghost" }),
		line("OutsideLine.Base", "0, the missing key", out, field("base"), "0", "1",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.Base = 0 }),
		line("OutsideLine.Base", "above MaxPrice", out, field("base"), "1000000001", "1000000000",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.Base = economysim.MaxPrice + 1 }),
		line("OutsideLine.PriceBySeason", "an unknown season", economysim.ErrUnknownSeason, append(field("price_by_season"), "monsoon", "season"), "monsoon", "",
			func(c *economysim.Config, l *economysim.OutsideLine) {
				l.PriceBySeason = []economysim.SeasonFactor{{Season: "monsoon", BP: 12_000}}
			}),
		line("OutsideLine.TrendBP", "above MaxTrendBP", out, field("trend_bp"), "100001", "100000",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.TrendBP = economysim.MaxTrendBP + 1 }),
		line("OutsideLine.FloorBP", "0", out, field("floor_bp"), "0", "1",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.FloorBP = 0 }),
		line("OutsideLine.CeilingBP", "below 10 000", out, field("ceiling_bp"), "9999", "10000",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.CeilingBP = 9_999 }),
		line("OutsideLine.WalkStepBP", "above MaxWalkStepBP", out, field("walk_step_bp"), "5001", "5000",
			func(c *economysim.Config, l *economysim.OutsideLine) {
				l.WalkStepBP, l.WalkBoundBP = economysim.MaxWalkStepBP+1, 9_000
			}),
		line("OutsideLine.WalkRevertBP", "above 10 000", out, field("walk_revert_bp"), "10001", "10000",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.WalkRevertBP = 10_001 }),
		line("OutsideLine.WalkBoundBP", "below WalkStepBP", out, field("walk_bound_bp"), "50", "100",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.WalkStepBP, l.WalkBoundBP = 100, 50 }),
		line("OutsideLine.WalkBoundBP", "set with no step", out, field("walk_bound_bp"), "50", "0",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.WalkBoundBP = 50 }),
		line("OutsideLine.Sell", "margins that sum to 0 with a buy side", economysim.ErrCrossedQuotes, externalLine, "0", "1",
			func(c *economysim.Config, l *economysim.OutsideLine) {
				l.Sell.MarginBP = 0
				l.Buy = economysim.OutsideSide{MarginBP: 0, Capacity: 5}
			}),
		line("OutsideLine.Buy", "a margin on an absent side", out, side("buy", "margin_bp"), "5", "0",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.Buy = economysim.OutsideSide{MarginBP: 5} }),
		line("OutsideSide.MarginBP", "above MaxSellMarginBP", out, side("sell", "margin_bp"), "100001", "100000",
			func(c *economysim.Config, l *economysim.OutsideLine) {
				l.Sell.MarginBP = economysim.MaxSellMarginBP + 1
			}),
		line("OutsideSide.MarginBP", "above MaxBuyMarginBP", out, side("buy", "margin_bp"), "10000", "9999",
			func(c *economysim.Config, l *economysim.OutsideLine) {
				l.Buy = economysim.OutsideSide{MarginBP: economysim.MaxBuyMarginBP + 1, Capacity: 5}
			}),
		line("OutsideSide.Capacity", "above MaxQuantity", out, side("sell", "capacity"), "1000000001", "1000000000",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.Sell.Capacity = economysim.MaxQuantity + 1 }),
		line("OutsideSide.Window", "an unknown season", economysim.ErrUnknownSeason, append(side("sell", "window"), "monsoon", "season"), "monsoon", "",
			func(c *economysim.Config, l *economysim.OutsideLine) {
				l.Sell.Window = []economysim.SeasonFactor{{Season: "monsoon", BP: 5_000}}
			}),
		line("OutsideSide.TotalUnits", "above MaxLineTotal", out, side("sell", "total_units"), strconv.FormatInt(economysim.MaxLineTotal+1, 10), strconv.FormatInt(economysim.MaxLineTotal, 10),
			func(c *economysim.Config, l *economysim.OutsideLine) { l.Sell.TotalUnits = economysim.MaxLineTotal + 1 }),
		line("OutsideSide.TotalValue", "below 0", out, side("sell", "total_value"), "-1", "0",
			func(c *economysim.Config, l *economysim.OutsideLine) { l.Sell.TotalValue = -1 }),
		line("SeasonFactor.Season", "an unknown season", economysim.ErrUnknownSeason, append(side("sell", "window"), "monsoon", "season"), "monsoon", "",
			func(c *economysim.Config, l *economysim.OutsideLine) {
				l.Sell.Window = []economysim.SeasonFactor{{Season: "monsoon", BP: 5_000}}
			}),
		line("SeasonFactor.BP", "above 10 000 in a window", out, append(side("sell", "window"), "spring", "bp"), "10001", "10000",
			func(c *economysim.Config, l *economysim.OutsideLine) {
				l.Sell.Window = []economysim.SeasonFactor{{Season: "spring", BP: 10_001}}
			}),
	}
}

// TestExternalValidate_Valid verifies that the configuration of externalConfig,
// which the cases below break one rule at a time, validates, and that its
// defaults (DefaultOutsideLine, DefaultOutsideGate) are accepted.
//
// Requirement: FR-EXT-002, FR-CFG-075
func TestExternalValidate_Valid(t *testing.T) {
	t.Parallel()
	c := externalConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(external) = %v, want nil", err)
	}
	c.Modules.External.Nodes[0].Gates = []economysim.OutsideGate{economysim.DefaultOutsideGate("north")}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(external with a default gate row) = %v, want nil", err)
	}
	c.Modules.External = &economysim.External{}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(external with no node) = %v, want nil: a module that is on and empty is valid", err)
	}
}

// TestExternalValidate_Errors verifies the sentinels of spec-external-sector.md
// 14.1, one case each, each breaking only its rule, matched with errors.Is.
//
// Requirement: FR-EXT-002, FR-EXT-003, FR-EXT-006, FR-EXT-007, FR-EXT-008, FR-EXT-009, FR-EXT-017, FR-EXT-025, FR-EXT-040, FR-EXT-042
func TestExternalValidate_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   error
		path   []string
	}{
		{"a node in an undefined region", func(c *economysim.Config) { c.Modules.External.Nodes[0].Region = "nowhere" },
			economysim.ErrUnknownRegion, []string{"modules", "external", "node", "nowhere", "region"}},
		{"a line in an undefined commodity", func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[0].Commodity = "ghost" },
			economysim.ErrUnknownCommodity, []string{"modules", "external", "node", "coast", "line", "ghost", "commodity"}},
		{"a season the calendar lacks", func(c *economysim.Config) {
			c.Modules.External.Nodes[0].Lines[1].Buy.Window = []economysim.SeasonFactor{{Season: "monsoon", BP: 0}}
		}, economysim.ErrUnknownSeason, []string{"modules", "external", "node", "coast", "line", "staple", "buy", "window", "monsoon", "season"}},
		{"any season while the calendar has none", func(c *economysim.Config) {
			c.Calendar.Seasons = nil
		}, economysim.ErrUnknownSeason, []string{"modules", "external", "node", "coast", "line", "staple", "buy", "window", "autumn", "season"}},
		{"a second row for a node", func(c *economysim.Config) {
			n := c.Modules.External.Nodes[0]
			c.Modules.External.Nodes = append(c.Modules.External.Nodes, n)
		}, economysim.ErrDuplicateID, []string{"modules", "external", "node", "coast", "region"}},
		{"a second row for a line", func(c *economysim.Config) {
			n := &c.Modules.External.Nodes[0]
			n.Lines = append(n.Lines, n.Lines[0])
		}, economysim.ErrDuplicateID, []string{"modules", "external", "node", "coast", "line", "mineral", "commodity"}},
		{"a second row for a gate", func(c *economysim.Config) {
			c.Modules.External.Nodes[0].Gates = []economysim.OutsideGate{economysim.DefaultOutsideGate("north"), economysim.DefaultOutsideGate("north")}
		}, economysim.ErrDuplicateID, []string{"modules", "external", "node", "coast", "gate", "north", "region"}},
		{"a second row for a season", func(c *economysim.Config) {
			w := c.Modules.External.Nodes[0].Lines[1].Buy.Window
			c.Modules.External.Nodes[0].Lines[1].Buy.Window = append(w, w[0])
		}, economysim.ErrDuplicateID, []string{"modules", "external", "node", "coast", "line", "staple", "buy", "window", "autumn", "season"}},
		{"a group in the node", func(c *economysim.Config) {
			c.Groups = append(c.Groups, economysim.Group{Region: "coast", Role: c.Roles[0].ID, Households: 1})
		}, economysim.ErrOutsideNodeInhabited, nil},
		{"a warm-start price in the node", func(c *economysim.Config) {
			c.WarmStart = &economysim.WarmStart{Prices: []economysim.SeedPrice{{Region: "coast", Commodity: "staple", Price: 96}}}
		}, economysim.ErrOutsideNodeInhabited, []string{"warm_start", "price", "coast", "staple"}},
		{"a warm-start wage in the node", func(c *economysim.Config) {
			c.WarmStart = &economysim.WarmStart{Wages: []economysim.SeedWage{{Region: "coast"}}}
		}, economysim.ErrOutsideNodeInhabited, []string{"warm_start", "wage", "coast"}},
		{"a warm-start group in the node", func(c *economysim.Config) {
			c.WarmStart = &economysim.WarmStart{Groups: []economysim.WarmGroup{{Region: "coast", Role: c.Roles[0].ID}}}
		}, economysim.ErrOutsideNodeInhabited, nil},
		{"a warm-start pool in the node", func(c *economysim.Config) {
			c.WarmStart = &economysim.WarmStart{Pools: []economysim.WarmPool{{Region: "coast"}}}
		}, economysim.ErrOutsideNodeInhabited, []string{"warm_start", "pool", "coast"}},
		{"a population region row in the node", func(c *economysim.Config) {
			c.Modules.Population = &economysim.Population{Regions: []economysim.RegionPopulation{{Region: "coast"}}}
		}, economysim.ErrOutsideNodeInhabited, []string{"modules", "population", "region", "coast"}},
		{"a taste row in the node", func(c *economysim.Config) {
			c.Modules.Demand = &economysim.Demand{RegionTastes: []economysim.RegionTaste{{Region: "coast", Want: "w"}}}
		}, economysim.ErrOutsideNodeInhabited, []string{"modules", "demand", "region_taste", "coast", "w"}},
		{"a money-flow target in the node", func(c *economysim.Config) {
			c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{{
				ID: "f", Kind: economysim.FlowGrant, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Region: "coast"}, Amount: 1,
			}}}
		}, economysim.ErrOutsideNodeInhabited, []string{"modules", "money_supply", "flow", "f", "target", "region"}},
		{"crossed quotes", func(c *economysim.Config) {
			l := &c.Modules.External.Nodes[0].Lines[0]
			l.Sell.MarginBP, l.Buy = 0, economysim.OutsideSide{Capacity: 3}
		}, economysim.ErrCrossedQuotes, externalLine},
		{"a gate row for a region with no edge", func(c *economysim.Config) {
			c.Modules.External.Nodes[0].Gates = []economysim.OutsideGate{economysim.DefaultOutsideGate("far")}
		}, economysim.ErrNotAGate, []string{"modules", "external", "node", "coast", "gate", "far", "region"}},
		{"an edge between two outside nodes", func(c *economysim.Config) {
			n := c.Modules.External.Nodes[0]
			n.Region = "north"
			c.Modules.External.Nodes = append(c.Modules.External.Nodes, n)
		}, economysim.ErrNotAGate, []string{"modules", "external", "node", "coast", "edge", "north"}},
		{"a margin on an absent side", func(c *economysim.Config) {
			c.Modules.External.Nodes[0].Lines[0].Buy.MarginBP = 5
		}, economysim.ErrOutOfBounds, append(slices.Clone(externalLine), "buy", "margin_bp")},
		{"the module while the graph has no edge", func(c *economysim.Config) { c.Edges = nil },
			economysim.ErrModuleDependency, []string{"modules", "external"}},
		{"a money flow named like a reason of the module", func(c *economysim.Config) {
			c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{{
				ID: "outside_budget", Kind: economysim.FlowGrant, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds}, Amount: 1,
			}}}
		}, economysim.ErrReasonCollision, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := externalConfig()
			tt.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, tt.want) {
				t.Fatalf("Validate(%s) = %v, want %v", tt.name, err, tt.want)
			}
			vs, _ := violations(err)
			if tt.path != nil && !slices.ContainsFunc(vs, func(v *economysim.ValidationError) bool {
				return errors.Is(v, tt.want) && slices.Equal(v.Path, tt.path)
			}) {
				t.Errorf("Validate(%s) paths = %q, want one at %q", tt.name, pathsOf(vs), tt.path)
			}
		})
	}
}

// externalRules are the eleven warnings of spec-external-sector.md 14.2.
var externalRules = []string{
	"node_without_gate", "gate_edge_wrong_direction", "line_without_side", "window_all_closed",
	"quote_outside_reference_band", "walk_faster_than_beliefs", "outside_faucet_without_sink",
	"outside_drain_without_source", "outside_arbitrage", "import_only_source", "line_untraded",
}

// externalWarnings returns the warnings of c that belong to the module.
func externalWarnings(c economysim.Config) []economysim.Warning {
	var out []economysim.Warning
	for _, w := range c.Warnings() {
		if slices.Contains(externalRules, w.Rule) {
			out = append(out, w)
		}
	}
	return out
}

// TestExternalValidate_Warnings verifies the eleven warnings of
// spec-external-sector.md 14.2: each is raised by a configuration that breaks
// only its condition, names its path, and never fails validation; the valid
// configuration of externalConfig raises none.
//
// Requirement: FR-EXT-041, FR-EXT-029, FR-CFG-076
func TestExternalValidate_Warnings(t *testing.T) {
	t.Parallel()
	if got := externalWarnings(externalConfig()); len(got) != 0 {
		t.Fatalf("Warnings(externalConfig) = %v, want no external warning", got)
	}
	line := func(c *economysim.Config, i int) *economysim.OutsideLine {
		return &c.Modules.External.Nodes[0].Lines[i]
	}
	closed := func() []economysim.SeasonFactor {
		return []economysim.SeasonFactor{{Season: "spring"}, {Season: "summer"}, {Season: "autumn"}, {Season: "winter"}}
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		path   []string
	}{
		{"a node with no gate", func(c *economysim.Config) { c.Edges = []economysim.Edge{{From: "north", To: "far", Distance: 5}} },
			"node_without_gate", externalNode},
		{"a sell side with only the region-to-node edge", func(c *economysim.Config) {
			c.Edges = []economysim.Edge{{From: "north", To: "coast", Distance: 30}}
		}, "gate_edge_wrong_direction", append(slices.Clone(externalLine), "sell")},
		{"a line with no side", func(c *economysim.Config) {
			n := &c.Modules.External.Nodes[0]
			n.Lines = append(n.Lines, economysim.DefaultOutsideLine("fuel", 179))
		}, "line_without_side", []string{"modules", "external", "node", "coast", "line", "fuel"}},
		{"a window that is 0 in every season", func(c *economysim.Config) { line(c, 0).Sell.Window = closed() },
			"window_all_closed", append(slices.Clone(externalLine), "sell", "window")},
		{"a quote above the reference band", func(c *economysim.Config) { line(c, 0).Base = 837_000 },
			"quote_outside_reference_band", append(slices.Clone(externalLine), "sell", "gate", "north")},
		{"a walk faster than a belief step", func(c *economysim.Config) {
			line(c, 0).WalkStepBP, line(c, 0).WalkBoundBP = c.World.BeliefStep+1, 5_000
		}, "walk_faster_than_beliefs", append(slices.Clone(externalLine), "walk_step_bp")},
		{"buy sides and no sell side", func(c *economysim.Config) { line(c, 0).Sell = economysim.OutsideSide{} },
			"outside_faucet_without_sink", []string{"modules", "external"}},
		{"sell sides and no buy side", func(c *economysim.Config) { line(c, 1).Buy = economysim.OutsideSide{} },
			"outside_drain_without_source", []string{"modules", "external"}},
		{"a cheaper ask at one gate than a bid at another", func(c *economysim.Config) {
			c.Edges = append(c.Edges, economysim.Edge{From: "coast", To: "far", Distance: 9})
			n := &c.Modules.External.Nodes[0]
			n.Gates = []economysim.OutsideGate{{Region: "far", ScaleBP: 5_000}}
			n.Lines[0].Buy = economysim.OutsideSide{MarginBP: 500, Capacity: 5}
		}, "outside_arbitrage", []string{"commodities", "mineral"}},
		{"an upkeep good that only an import supplies", func(c *economysim.Config) {
			spice := economysim.DefaultCommodity()
			spice.ID, spice.ReferencePrice = "spice", 100
			c.Commodities = append(c.Commodities, spice)
			c.World.Upkeep = append(c.World.Upkeep, economysim.Item{Commodity: "spice", Quantity: 1})
			n := &c.Modules.External.Nodes[0]
			l := economysim.DefaultOutsideLine("spice", 100)
			l.Sell = economysim.OutsideSide{MarginBP: 100, Capacity: 3}
			n.Lines = append(n.Lines, l)
		}, "import_only_source", []string{"commodities", "spice"}},
		{"a line in a good no role trades", func(c *economysim.Config) {
			curio := economysim.DefaultCommodity()
			curio.ID, curio.ReferencePrice = "curio", 500
			c.Commodities = append(c.Commodities, curio)
			l := economysim.DefaultOutsideLine("curio", 500)
			l.Sell = economysim.OutsideSide{MarginBP: 100, Capacity: 3}
			c.Modules.External.Nodes[0].Lines = append(c.Modules.External.Nodes[0].Lines, l)
		}, "line_untraded", []string{"modules", "external", "node", "coast", "line", "curio"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := externalConfig()
			tt.mutate(&c)
			if !slices.ContainsFunc(externalWarnings(c), func(w economysim.Warning) bool {
				return w.Rule == tt.rule && slices.Equal(w.Path, tt.path)
			}) {
				t.Errorf("Warnings(%s) = %v, want %s at %q", tt.name, externalWarnings(c), tt.rule, tt.path)
			}
		})
	}
	// Every rule of 14.2 has a case: a rule added to the module without a case here
	// would be untested.
	for _, rule := range externalRules {
		if !slices.ContainsFunc(tests, func(tt struct {
			name   string
			mutate func(c *economysim.Config)
			rule   string
			path   []string
		}) bool {
			return tt.rule == rule
		}) {
			t.Errorf("no case for the warning %q", rule)
		}
	}
}

// TestExternalValidate_ImportSource verifies that a sell side makes a commodity
// sourced (spec-external-sector.md 10.1): the warning raw_without_source is gone
// for a raw good that only an import supplies, it stays when the window is 0 in
// every season, the sell side has no gate or is a buy side, and a good that the
// import is the only source of draws import_only_source.
//
// Requirement: FR-EXT-035
func TestExternalValidate_ImportSource(t *testing.T) {
	t.Parallel()
	// ore is a raw good that no recipe makes: only an import can source it.
	ore := economysim.DefaultCommodity()
	ore.ID, ore.ReferencePrice, ore.Raw = "ore", 400, true
	withOre := func(side economysim.OutsideSide, buy bool) economysim.Config {
		c := externalConfig()
		c.Commodities = append(c.Commodities, ore)
		l := economysim.DefaultOutsideLine("ore", 400)
		if buy {
			l.Buy = side
		} else {
			l.Sell = side
		}
		n := &c.Modules.External.Nodes[0]
		n.Lines = append(n.Lines, l)
		return c
	}
	rawWithout := func(c economysim.Config) bool {
		return slices.ContainsFunc(c.Warnings(), func(w economysim.Warning) bool {
			return w.Rule == "raw_without_source" && slices.Equal(w.Path, []string{"commodities", "ore"})
		})
	}
	closed := []economysim.SeasonFactor{{Season: "spring"}, {Season: "summer"}, {Season: "autumn"}, {Season: "winter"}}
	tests := []struct {
		name string
		c    economysim.Config
		warn bool
	}{
		{"a sell side with a gate and an open window", withOre(economysim.OutsideSide{MarginBP: 100, Capacity: 5}, false), false},
		{"a window that is 0 in every season", withOre(economysim.OutsideSide{MarginBP: 100, Capacity: 5, Window: closed}, false), true},
		{"a buy side only", withOre(economysim.OutsideSide{MarginBP: 100, Capacity: 5}, true), true},
		{"a sell side with a window closed in one season", withOre(economysim.OutsideSide{MarginBP: 100, Capacity: 5, Window: closed[:3]}, false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := rawWithout(tt.c); got != tt.warn {
				t.Errorf("raw_without_source for ore (%s) = %v, want %v", tt.name, got, tt.warn)
			}
		})
	}
	noGate := withOre(economysim.OutsideSide{MarginBP: 100, Capacity: 5}, false)
	noGate.Edges = []economysim.Edge{{From: "north", To: "coast", Distance: 30}}
	if !rawWithout(noGate) {
		t.Errorf("raw_without_source for ore with no node-to-region edge = false, want true: the sell side has no gate")
	}
	off := withOre(economysim.OutsideSide{MarginBP: 100, Capacity: 5}, false)
	off.Modules.External = nil
	if !rawWithout(off) {
		t.Errorf("raw_without_source for ore with the module off = false, want true")
	}
}

// TestExternalHash verifies the hash of the module (spec-config.md 8.2,
// FR-CFG-075): off leaves the pinned hash of the all-off reference configuration
// alone, on differs from off and an empty module that is on differs from both,
// the label is not hashed, every other field is, and the rows are hashed in
// canonical order whatever order they were written in.
//
// Requirement: FR-EXT-007, FR-EXT-043
func TestExternalHash(t *testing.T) {
	t.Parallel()
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(reference) = %s, want the pinned %s: the module off must leave no trace", got, pinnedReferenceHash)
	}
	on := externalConfig()
	off := externalConfig()
	off.Modules.External = nil
	empty := externalConfig()
	empty.Modules.External = &economysim.External{}
	if hexHash(on) == hexHash(off) || hexHash(empty) == hexHash(off) || hexHash(empty) == hexHash(on) {
		t.Errorf("HashConfig of on, empty and off are not three different hashes")
	}
	label := externalConfig()
	label.Modules.External.Nodes[0].Label = "the far coast"
	if hexHash(label) != hexHash(on) {
		t.Errorf("HashConfig(label changed) != HashConfig(on): the label is presentation only")
	}
	changes := map[string]func(c *economysim.Config){
		"base":        func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[0].Base++ },
		"trend":       func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[0].TrendBP = 7 },
		"floor":       func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[0].FloorBP = 7 },
		"ceiling":     func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[0].CeilingBP = 20_000 },
		"walk":        func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[0].WalkStepBP = 3 },
		"margin":      func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[0].Sell.MarginBP++ },
		"capacity":    func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[0].Sell.Capacity++ },
		"total units": func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[0].Sell.TotalUnits = 9 },
		"total value": func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[0].Sell.TotalValue = 9 },
		"window":      func(c *economysim.Config) { c.Modules.External.Nodes[0].Lines[1].Buy.Window[0].BP = 1 },
		"season table": func(c *economysim.Config) {
			c.Modules.External.Nodes[0].Lines[0].PriceBySeason = []economysim.SeasonFactor{{Season: "spring", BP: 9}}
		},
		"gate": func(c *economysim.Config) {
			c.Modules.External.Nodes[0].Gates = []economysim.OutsideGate{{Region: "north", ScaleBP: 9}}
		},
		"node region": func(c *economysim.Config) { c.Modules.External.Nodes[0].Region = "far" },
	}
	for name, mutate := range changes {
		c := externalConfig()
		mutate(&c)
		if hexHash(c) == hexHash(on) {
			t.Errorf("HashConfig(%s changed) == HashConfig(on), want different: the field is hashed", name)
		}
	}
	// The rows in another order: nodes, lines, gates and season rows sort.
	a, b := externalConfig(), externalConfig()
	for _, c := range []*economysim.Config{&a, &b} {
		c.Modules.External.Nodes[0].Gates = []economysim.OutsideGate{{Region: "far", ScaleBP: 9_000}, {Region: "north", ScaleBP: 8_000}}
		c.Modules.External.Nodes[0].Lines[1].Buy.Window = []economysim.SeasonFactor{{Season: "autumn"}, {Season: "winter"}}
		c.Modules.External.Nodes = append(c.Modules.External.Nodes, economysim.OutsideNode{Region: "far"})
	}
	n := b.Modules.External.Nodes
	n[0], n[1] = n[1], n[0]
	l := n[1].Lines
	l[0], l[1] = l[1], l[0]
	g := n[1].Gates
	g[0], g[1] = g[1], g[0]
	w := n[1].Lines[0].Buy.Window
	w[0], w[1] = w[1], w[0]
	if hexHash(a) != hexHash(b) {
		t.Errorf("HashConfig of the same rows in another order differ, want equal: the hash sorts nodes, lines, gates and seasons")
	}
}

// FuzzExternalValidate feeds Validate and Warnings a module built from fuzzed
// numbers: it never panics, a configuration that validates hashes to the same
// bytes twice, and an invalid one returns an error that is a *ValidationError.
//
// Requirement: NFR-EXT-003
func FuzzExternalValidate(f *testing.F) {
	f.Add(int64(837), int64(500), int64(10), int64(0), int64(0), int64(0), int64(10_000), int64(1_000), int64(100_000))
	f.Add(int64(0), int64(-1), int64(-5), int64(100), int64(50), int64(5_000), int64(0), int64(0), int64(0))
	f.Add(int64(1<<62), int64(1<<62), int64(1<<62), int64(1<<62), int64(1<<62), int64(1<<62), int64(1<<62), int64(1<<62), int64(1<<62))
	f.Fuzz(func(t *testing.T, base, margin, capacity, step, bound, revert, scale, floor, ceiling int64) {
		c := externalConfig()
		n := &c.Modules.External.Nodes[0]
		l := &n.Lines[0]
		l.Base, l.Sell.MarginBP, l.Sell.Capacity = economysim.Price(base), margin, capacity
		l.WalkStepBP, l.WalkBoundBP, l.WalkRevertBP = step, bound, revert
		l.FloorBP, l.CeilingBP = floor, ceiling
		n.Gates = []economysim.OutsideGate{{Region: "north", ScaleBP: scale}}
		_ = c.Warnings()
		err := c.Validate()
		if err == nil {
			again := c
			again.Modules.External = &economysim.External{Nodes: slices.Clone(c.Modules.External.Nodes)}
			if hexHash(c) != hexHash(again) {
				t.Fatalf("HashConfig of a valid configuration differs from that of its copy")
			}
			return
		}
		if _, other := violations(err); len(other) != 0 {
			t.Fatalf("Validate returned entries that are not a *ValidationError: %v", other)
		}
	})
}
