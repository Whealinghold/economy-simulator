package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the risk module's configuration surface: its rows, bounds, shape
// rules, warnings and canonical order (spec-risk.md 3.2, 12, 13, FR-RSK-001 to
// 004, FR-CFG-089). The module has no world effect yet. The hash is in
// risk_hash_internal_test.go and the loader in config/risk_test.go.

// riskConfig returns transportConfig (two regions, edges of 40 in both directions
// and the self-carried mode foot) with the storage module of storageConfig (the
// classes bin and shed, both listed by the growers) and the risk module with two
// route rows, one unit and one common, and two store rows. It raises no risk
// warning: both edges lie on a path and both classes are owned.
func riskConfig() economysim.Config {
	c := transportConfig()
	c.Modules.Storage = storageConfig().Modules.Storage
	c.Modules.Risk = &economysim.Risk{
		Routes: []economysim.RouteRisk{
			{From: "north", To: "south", Mode: "foot", StrikeBP: 3_000, SeverityBP: 2_500},
			{From: "south", To: "north", Mode: "foot", StrikeBP: 1_000, SeverityBP: 6_000, Common: true},
		},
		Stores: []economysim.StoreRisk{
			{Region: "north", Class: "bin", StrikeBP: 500, SeverityBP: 4_000},
			{Region: "north", Class: "shed", StrikeBP: 200, SeverityBP: 10_000, Common: true},
		},
	}
	return c
}

// riskOutsideNode adds the outside node coast to c, with its edges to north, and keeps
// the edges between north and south that the rows of riskConfig need.
func riskOutsideNode(c *economysim.Config) {
	withOutsideNode(c)
	c.Edges = append(c.Edges, economysim.Edge{From: "north", To: "south", Distance: 40}, economysim.Edge{From: "south", To: "north", Distance: 40})
}

// riskCase builds one failing case from a mutation of riskConfig.
type riskCase func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase

// riskAt is the path of a field of a row of one of the tables.
func riskAt(table string, i int, key ...string) []string {
	return append([]string{"modules", "risk", table + "[" + strconv.Itoa(i) + "]"}, key...)
}

// riskFieldCases are the failing cases of the rows of the risk module in the
// parameter-owner table (spec-risk.md 12.2, 13.1, FR-RSK-003): each breaks one
// field of the valid configuration of riskConfig. buildFieldCases appends them to
// the table that the registry test reads.
func riskFieldCases() []fieldCase {
	risk := func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = riskConfig()
				set(c)
			}}
	}
	var cs []fieldCase
	cs = append(cs, riskTableCases(risk)...)
	cs = append(cs, riskRouteCases(risk)...)
	return append(cs, riskStoreCases(risk)...)
}

// riskTableCases are the cases of the two tables: the module dependency and the
// bound on the rows.
func riskTableCases(risk riskCase) []fieldCase {
	return []fieldCase{
		risk("Risk.Routes", "without the transport module", economysim.ErrModuleDependency, []string{"modules", "risk", "routes"}, "transport", "",
			func(c *economysim.Config) { c.Modules.Transport = nil }),
		risk("Risk.Routes", "more than MaxRiskRows", economysim.ErrOutOfBounds, []string{"modules", "risk", "routes"}, "4097", "4096",
			func(c *economysim.Config) {
				c.Modules.Risk.Routes = slices.Repeat(c.Modules.Risk.Routes[:1], economysim.MaxRiskRows+1)
			}),
		risk("Risk.Stores", "without the storage module", economysim.ErrModuleDependency, []string{"modules", "risk", "stores"}, "storage", "",
			func(c *economysim.Config) { c.Modules.Storage = nil }),
		risk("Risk.Stores", "more than MaxRiskRows", economysim.ErrOutOfBounds, []string{"modules", "risk", "stores"}, "4097", "4096",
			func(c *economysim.Config) {
				c.Modules.Risk.Stores = slices.Repeat(c.Modules.Risk.Stores[:1], economysim.MaxRiskRows+1)
			}),
	}
}

// riskRouteCases are the cases of the fields of a route row.
func riskRouteCases(risk riskCase) []fieldCase {
	row := func(c *economysim.Config) *economysim.RouteRisk { return &c.Modules.Risk.Routes[0] }
	return []fieldCase{
		risk("RouteRisk.From", "an unknown region", economysim.ErrUnknownRegion, riskAt("routes", 0, "from"), "nowhere", "",
			func(c *economysim.Config) { row(c).From = "nowhere" }),
		risk("RouteRisk.From", "an outside node", economysim.ErrOutsideNode, riskAt("routes", 0, "from"), "coast", "",
			func(c *economysim.Config) { riskOutsideNode(c); row(c).From, row(c).To = "coast", "north" }),
		risk("RouteRisk.To", "an unknown region", economysim.ErrUnknownRegion, riskAt("routes", 0, "to"), "nowhere", "",
			func(c *economysim.Config) { row(c).To = "nowhere" }),
		risk("RouteRisk.To", "a pair that is not an edge", economysim.ErrUnknownEdge, riskAt("routes", 0), "north->north", "",
			func(c *economysim.Config) { row(c).To = "north" }),
		risk("RouteRisk.Mode", "an unknown mode", economysim.ErrUnknownMode, riskAt("routes", 0, "mode"), "coach", "",
			func(c *economysim.Config) { row(c).Mode = "coach" }),
		risk("RouteRisk.Mode", "a key twice", economysim.ErrDuplicateID, riskAt("routes", 1), "north->south foot", "",
			func(c *economysim.Config) { c.Modules.Risk.Routes[1] = *row(c) }),
		risk("RouteRisk.StrikeBP", "below 1", economysim.ErrRiskShape, riskAt("routes", 0, "strike_bp"), "0", "1 to 10000",
			func(c *economysim.Config) { row(c).StrikeBP = 0 }),
		risk("RouteRisk.StrikeBP", "above 10 000", economysim.ErrRiskShape, riskAt("routes", 0, "strike_bp"), "10001", "1 to 10000",
			func(c *economysim.Config) { row(c).StrikeBP = 10_001 }),
		risk("RouteRisk.SeverityBP", "below 1", economysim.ErrRiskShape, riskAt("routes", 0, "severity_bp"), "0", "1 to 10000",
			func(c *economysim.Config) { row(c).SeverityBP = 0 }),
		risk("RouteRisk.SeverityBP", "above 10 000", economysim.ErrRiskShape, riskAt("routes", 0, "severity_bp"), "10001", "1 to 10000",
			func(c *economysim.Config) { row(c).SeverityBP = 10_001 }),
	}
}

// riskStoreCases are the cases of the fields of a store row.
func riskStoreCases(risk riskCase) []fieldCase {
	row := func(c *economysim.Config) *economysim.StoreRisk { return &c.Modules.Risk.Stores[0] }
	return []fieldCase{
		risk("StoreRisk.Region", "an unknown region", economysim.ErrUnknownRegion, riskAt("stores", 0, "region"), "nowhere", "",
			func(c *economysim.Config) { row(c).Region = "nowhere" }),
		risk("StoreRisk.Region", "an outside node", economysim.ErrOutsideNode, riskAt("stores", 0, "region"), "coast", "",
			func(c *economysim.Config) { riskOutsideNode(c); row(c).Region = "coast" }),
		risk("StoreRisk.Class", "an unknown store class", economysim.ErrUnknownStoreClass, riskAt("stores", 0, "class"), "vault", "",
			func(c *economysim.Config) { row(c).Class = "vault" }),
		risk("StoreRisk.Class", "a key twice", economysim.ErrDuplicateID, riskAt("stores", 1), "north bin", "",
			func(c *economysim.Config) { c.Modules.Risk.Stores[1] = *row(c) }),
		risk("StoreRisk.StrikeBP", "below 1", economysim.ErrRiskShape, riskAt("stores", 0, "strike_bp"), "0", "1 to 10000",
			func(c *economysim.Config) { row(c).StrikeBP = 0 }),
		risk("StoreRisk.StrikeBP", "above 10 000", economysim.ErrRiskShape, riskAt("stores", 0, "strike_bp"), "10001", "1 to 10000",
			func(c *economysim.Config) { row(c).StrikeBP = 10_001 }),
		risk("StoreRisk.SeverityBP", "below 1", economysim.ErrRiskShape, riskAt("stores", 0, "severity_bp"), "-1", "1 to 10000",
			func(c *economysim.Config) { row(c).SeverityBP = -1 }),
		risk("StoreRisk.SeverityBP", "above 10 000", economysim.ErrRiskShape, riskAt("stores", 0, "severity_bp"), "10001", "1 to 10000",
			func(c *economysim.Config) { row(c).SeverityBP = 10_001 }),
	}
}

// TestValidate_Risk_ValidConfigurationPasses verifies that the configuration the
// failing cases start from is valid, so each case breaks one rule, and that the
// empty module, and one with rows of one table only, validate.
//
// Requirement: FR-RSK-001, FR-RSK-002
func TestValidate_Risk_ValidConfigurationPasses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		set  func(c *economysim.Config)
	}{
		{"the rows of riskConfig", func(*economysim.Config) {}},
		{"an empty module", func(c *economysim.Config) { c.Modules.Risk = &economysim.Risk{} }},
		{"an empty module with the other modules off", func(c *economysim.Config) {
			c.Modules.Risk = &economysim.Risk{}
			c.Modules.Transport, c.Modules.Storage = nil, nil
		}},
		{"routes only, storage off", func(c *economysim.Config) {
			c.Modules.Risk.Stores = nil
			c.Modules.Storage = nil
		}},
		{"stores only, transport off", func(c *economysim.Config) {
			c.Modules.Risk.Routes = nil
			c.Modules.Transport = nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := riskConfig()
			tt.set(&c)
			if err := c.Validate(); err != nil {
				t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
			}
		})
	}
}

// TestValidate_Risk_StrikeAtTheBounds verifies that 1 and 10 000 are in bound for
// both numbers of a row, a certain row with a total severity included (it warns,
// it does not fail).
//
// Requirement: FR-RSK-003, FR-RSK-004
func TestValidate_Risk_StrikeAtTheBounds(t *testing.T) {
	t.Parallel()
	for _, v := range []int64{1, 10_000} {
		c := riskConfig()
		c.Modules.Risk.Routes[0].StrikeBP, c.Modules.Risk.Routes[0].SeverityBP = v, v
		c.Modules.Risk.Stores[0].StrikeBP, c.Modules.Risk.Stores[0].SeverityBP = v, v
		if err := c.Validate(); err != nil {
			t.Errorf("Validate(rows at %d) = %v, want nil", v, err)
		}
	}
}

// TestValidate_Risk_DuplicateID verifies that a key twice in a table built in Go is
// ErrDuplicateID at the second row, for each table, and that a different mode or
// class makes a different key.
//
// Requirement: FR-RSK-002, FR-RSK-003
func TestValidate_Risk_DuplicateID(t *testing.T) {
	t.Parallel()
	c := riskConfig()
	m := economysim.Mode{ID: "cart", Speed: 20, UnitMass: 40, UnitBulk: 30, Crew: economysim.DefaultCrew}
	c.Modules.Transport.Modes = append(c.Modules.Transport.Modes, m)
	c.Modules.Risk.Routes = append(c.Modules.Risk.Routes, c.Modules.Risk.Routes[0])
	c.Modules.Risk.Routes[2].Mode = "cart"
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(the same edge in two modes) = %v, want nil: a mode is part of the key", err)
	}
	c.Modules.Risk.Routes[2].Mode = "foot"
	if err := c.Validate(); !errors.Is(err, economysim.ErrDuplicateID) {
		t.Errorf("Validate(a route key twice) = %v, want ErrDuplicateID", err)
	}
}

// TestWarnings_Risk verifies the four warnings of spec-risk.md 13.2, one case each,
// that none fails validation and that the overlay-like configuration is silent.
//
// Requirement: FR-RSK-004
func TestWarnings_Risk(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		set      func(c *economysim.Config)
		wantRule string
		wantPath []string
	}{
		{"no rows", func(c *economysim.Config) { c.Modules.Risk = &economysim.Risk{} }, "no_risk_rows", []string{"modules", "risk"}},
		{"an edge on no path", func(c *economysim.Config) {
			// north to south is 100 long and every path from north goes by mid in 40 + 40.
			mid := economysim.DefaultRegion()
			mid.ID = "mid"
			c.Regions = append(c.Regions, mid)
			c.Edges = []economysim.Edge{
				{From: "north", To: "south", Distance: 100}, {From: "north", To: "mid", Distance: 40},
				{From: "mid", To: "south", Distance: 40}, {From: "south", To: "north", Distance: 40},
				{From: "south", To: "mid", Distance: 40}, {From: "mid", To: "north", Distance: 40},
			}
			c.Modules.Risk.Routes[1] = economysim.RouteRisk{From: "north", To: "mid", Mode: "foot", StrikeBP: 100, SeverityBP: 100}
			c.Modules.Risk.Routes[0].StrikeBP = 100
		}, "risk_edge_off_path", []string{"modules", "risk", "routes[0]"}},
		{"a class no role owns", func(c *economysim.Config) {
			c.Modules.Storage.Classes = append(c.Modules.Storage.Classes, economysim.StoreClass{ID: "vault", Mass: 10, Bulk: 10})
			c.Modules.Risk.Stores[0].Class = "vault"
		}, "risk_store_class_unowned", []string{"modules", "risk", "stores[0]", "class"}},
		{"a certain route row", func(c *economysim.Config) { c.Modules.Risk.Routes[0].StrikeBP = 10_000 },
			"risk_certain", []string{"modules", "risk", "routes[0]", "strike_bp"}},
		{"a certain store row", func(c *economysim.Config) { c.Modules.Risk.Stores[1].StrikeBP = 10_000 },
			"risk_certain", []string{"modules", "risk", "stores[1]", "strike_bp"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := riskConfig()
			tt.set(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil: a warning never fails", tt.name, err)
			}
			var got []economysim.Warning
			for _, w := range c.Warnings() {
				if len(w.Path) > 1 && w.Path[0] == "modules" && w.Path[1] == "risk" {
					got = append(got, w)
				}
			}
			if len(got) == 0 {
				t.Fatalf("Warnings(%s) has no risk warning, want %s", tt.name, tt.wantRule)
			}
			if got[0].Rule != tt.wantRule || !slices.Equal(got[0].Path, tt.wantPath) {
				t.Errorf("Warnings(%s) first risk warning = %v %q, want %v %q", tt.name, got[0].Path, got[0].Rule, tt.wantPath, tt.wantRule)
			}
		})
	}
}

// TestWarnings_RiskRowWithoutEdgeIsNotOffPath verifies that a route row whose two
// regions are defined but whose edge is not configured raises no
// risk_edge_off_path: Validate reports it as ErrUnknownEdge, and a second report
// for the same row would be wrong, because "off path" is a claim about a
// configured edge.
//
// Requirement: FR-RSK-003, FR-RSK-004
//
// Fix: plan 3-rsk review 1. riskOffPath looked only for a hop from the row's origin
// to its destination, so a row with no edge between two regions that a path joins
// by way of a third was warned as off path as well as refused.
func TestWarnings_RiskRowWithoutEdgeIsNotOffPath(t *testing.T) {
	t.Parallel()
	c := riskConfig()
	// north and south are joined only by way of mid: no edge north to south.
	mid := economysim.DefaultRegion()
	mid.ID = "mid"
	c.Regions = append(c.Regions, mid)
	c.Edges = []economysim.Edge{
		{From: "north", To: "mid", Distance: 40}, {From: "mid", To: "south", Distance: 40},
		{From: "south", To: "mid", Distance: 40}, {From: "mid", To: "north", Distance: 40},
	}
	c.Modules.Risk = &economysim.Risk{
		Routes: []economysim.RouteRisk{{From: "north", To: "south", Mode: "foot", StrikeBP: 100, SeverityBP: 100}},
	}
	if err := c.Validate(); !errors.Is(err, economysim.ErrUnknownEdge) {
		t.Fatalf("Validate(a route row with no edge) = %v, want ErrUnknownEdge", err)
	}
	for _, w := range c.Warnings() {
		if w.Rule == "risk_edge_off_path" {
			t.Errorf("Warnings(a route row with no edge) = %v %q, want no risk_edge_off_path", w.Path, w.Rule)
		}
	}
}

// TestWarnings_AbsentRowSilent verifies that the module off, and the module on with
// rows that touch every edge and class, raise no risk warning, and that absence of
// a row is no warning (spec-risk.md 3.2.3).
//
// Requirement: FR-RSK-002, FR-RSK-004
func TestWarnings_AbsentRowSilent(t *testing.T) {
	t.Parallel()
	off := riskConfig()
	off.Modules.Risk = nil
	for name, c := range map[string]economysim.Config{"module off": off, "module on": riskConfig()} {
		for _, w := range c.Warnings() {
			if len(w.Path) > 1 && w.Path[0] == "modules" && w.Path[1] == "risk" {
				t.Errorf("Warnings(%s) = %v %q, want no risk warning", name, w.Path, w.Rule)
			}
		}
	}
}

// TestSentinels_Risk verifies that the two new sentinels are distinct from each
// other and from the ones they are close to, so errors.Is tells them apart.
//
// Requirement: FR-CFG-089
func TestSentinels_Risk(t *testing.T) {
	t.Parallel()
	all := []error{
		economysim.ErrRiskShape, economysim.ErrUnknownRiskRow, economysim.ErrOutOfBounds,
		economysim.ErrUnknownEdge, economysim.ErrUnknownMode, economysim.ErrUnknownStoreClass,
	}
	for i, a := range all {
		for j, b := range all {
			if (i == j) != errors.Is(a, b) {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", a, b, i != j, i == j)
			}
		}
	}
	if economysim.MaxRiskRows != 4_096 {
		t.Errorf("MaxRiskRows = %d, want 4096 (spec-risk.md 12.2)", economysim.MaxRiskRows)
	}
}

// TestCanonical_RiskRowsSortedByIndex verifies, through the hash, that the order the
// rows are written in changes nothing: route rows sort by (From, To, Mode) and store
// rows by (Region, Class), the order of the indices of a canonical configuration.
//
// Requirement: FR-RSK-002, FR-CFG-089
func TestCanonical_RiskRowsSortedByIndex(t *testing.T) {
	t.Parallel()
	a := riskConfig()
	b := riskConfig()
	slices.Reverse(b.Modules.Risk.Routes)
	slices.Reverse(b.Modules.Risk.Stores)
	if economysim.HashConfig(a) != economysim.HashConfig(b) {
		t.Error("HashConfig of the rows reversed differs from the hash of the rows in order, want equal")
	}
	c := riskConfig()
	c.Modules.Risk.Routes[0].StrikeBP++
	if economysim.HashConfig(a) == economysim.HashConfig(c) {
		t.Error("HashConfig after a change of StrikeBP equals the hash before, want different")
	}
}

// FuzzValidateRisk verifies that Validate never panics on rows with any numbers and
// any IDs, and that a configuration it accepts has every number in 1 to 10 000.
//
// Requirement: FR-RSK-003, FR-CFG-089
func FuzzValidateRisk(f *testing.F) {
	f.Add("north", "south", "foot", int64(3_000), int64(2_500), true)
	f.Add("", "", "", int64(0), int64(-1), false)
	f.Fuzz(func(t *testing.T, from, to, name string, strike, sev int64, common bool) {
		c := riskConfig()
		c.Modules.Risk = &economysim.Risk{
			Routes: []economysim.RouteRisk{{From: from, To: to, Mode: name, StrikeBP: strike, SeverityBP: sev, Common: common}},
			Stores: []economysim.StoreRisk{{Region: from, Class: name, StrikeBP: strike, SeverityBP: sev, Common: common}},
		}
		if err := c.Validate(); err == nil && (strike < 1 || strike > 10_000 || sev < 1 || sev > 10_000) {
			t.Fatalf("Validate accepted strike %d and severity %d, want ErrRiskShape", strike, sev)
		}
		_ = c.Warnings()
	})
}
