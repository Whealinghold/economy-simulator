package economysim_test

import (
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the information module's configuration surface: its three
// fields, the flag of a role, the bounds, the derived checks, the defaults and
// the four warnings (spec-information.md 8, 9, FR-INF-001, 031 to 039, 044, 048,
// FR-CFG-074, 076). The module has no world effect yet.

// informationConfig returns referenceConfig with a second region, a group of
// cooks in each region, edges of 55 and 40, and the information module on at
// speed 40: the cook is informed. The lags are 1 + ceil(55 / 40) = 3 from north
// to south and 1 + ceil(40 / 40) = 2 from south to north.
func informationConfig() economysim.Config {
	c := referenceConfig()
	south := economysim.DefaultRegion()
	south.ID = "south"
	c.Regions = append(c.Regions, south)
	c.Edges = []economysim.Edge{{From: "north", To: "south", Distance: 55}, {From: "south", To: "north", Distance: 40}}
	c.Groups = append(c.Groups, economysim.Group{Region: "south", Role: "cook", Households: 136})
	m := economysim.DefaultInformation()
	m.InfoSpeed = 40
	c.Modules.Information = &m
	for i := range c.Roles {
		c.Roles[i].Informed = c.Roles[i].ID == "cook"
	}
	return c
}

// informationFieldCases are the failing cases of the rows of the information
// module in the parameter-owner table (spec-information.md 8, 9.1, FR-CFG-074):
// each breaks one field of a valid configuration. buildFieldCases appends them
// to the table that the registry test reads.
func informationFieldCases() []fieldCase {
	bounds := []error{economysim.ErrOutOfBounds}
	module := func(field, name, key string, value, bound string, set func(m *economysim.Information)) fieldCase {
		return fieldCase{field: field, name: name, rules: bounds, paths: [][]string{{"modules", "information", key}}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				m := economysim.DefaultInformation()
				m.InfoSpeed = 40
				set(&m)
				c.Modules.Information = &m
			}}
	}
	return []fieldCase{
		module("Information.InfoSpeed", "0, the missing key", "info_speed", "0", "1", func(m *economysim.Information) { m.InfoSpeed = 0 }),
		module("Information.InfoSpeed", "above MaxInfoSpeed", "info_speed", strconv.Itoa(economysim.MaxInfoSpeed+1), strconv.Itoa(economysim.MaxInfoSpeed),
			func(m *economysim.Information) { m.InfoSpeed = economysim.MaxInfoSpeed + 1 }),
		module("Information.RumourErrorBP", "above MaxRumourErrorBP", "rumour_error_bp", "5001", "5000", func(m *economysim.Information) { m.RumourErrorBP = 5_001 }),
		module("Information.RumourErrorBP", "below 0", "rumour_error_bp", "-1", "0", func(m *economysim.Information) { m.RumourErrorBP = -1 }),
		module("Information.RumourMaxAge", "below 2", "rumour_max_age", "1", "2", func(m *economysim.Information) { m.RumourMaxAge = 1 }),
		module("Information.RumourMaxAge", "above MaxRumourAge", "rumour_max_age", "10001", "10000", func(m *economysim.Information) { m.RumourMaxAge = 10_001 }),
		{field: "Role.Informed", name: "true while the module is off", rules: []error{economysim.ErrModuleDependency},
			paths: [][]string{{"roles", "cook", "informed"}}, value: "true",
			mutate: func(c *economysim.Config) {
				for i := range c.Roles {
					c.Roles[i].Informed = c.Roles[i].ID == "cook"
				}
			}},
	}
}

// grow appends n regions r0, r1, ... to c, with no edge and no group.
func grow(c *economysim.Config, n int) {
	for i := range n {
		r := economysim.DefaultRegion()
		r.ID = "r" + strconv.Itoa(i)
		c.Regions = append(c.Regions, r)
	}
}

// tradeAll makes the cook trade every commodity of c.
func tradeAll(c *economysim.Config) {
	var all []string
	for _, cm := range c.Commodities {
		all = append(all, cm.ID)
	}
	for i := range c.Roles {
		if c.Roles[i].ID == "cook" {
			c.Roles[i].Trades = all
		}
	}
}

// TestValidate_Information verifies the rules of spec-information.md 9.1 and
// 9.2 beyond the field cases: the bounds are accepted at their edges, a role
// flagged Informed with the module off is ErrModuleDependency at the role, the
// two derived products are ErrTooMany at their paths (and are not computed when
// a bound already failed), and a valid configuration validates.
//
// Requirement: FR-INF-001, FR-INF-031, FR-INF-033, FR-INF-034, FR-CFG-074
func TestValidate_Information(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   error
		path   []string
	}{
		{"a valid configuration", func(c *economysim.Config) {}, nil, nil},
		{"InfoSpeed at its bound", func(c *economysim.Config) { c.Modules.Information.InfoSpeed = economysim.MaxInfoSpeed }, nil, nil},
		{"RumourErrorBP 0", func(c *economysim.Config) { c.Modules.Information.RumourErrorBP = 0 }, nil, nil},
		{"RumourErrorBP at its bound", func(c *economysim.Config) { c.Modules.Information.RumourErrorBP = economysim.MaxRumourErrorBP }, nil, nil},
		{"RumourMaxAge 2", func(c *economysim.Config) { c.Modules.Information.RumourMaxAge = 2 }, nil, nil},
		{"RumourMaxAge at its bound", func(c *economysim.Config) { c.Modules.Information.RumourMaxAge = economysim.MaxRumourAge }, nil, nil},
		{"no informed role", func(c *economysim.Config) { c.Roles[0].Informed, c.Roles[1].Informed = false, false }, nil, nil},
		{"InfoSpeed 0", func(c *economysim.Config) { c.Modules.Information.InfoSpeed = 0 }, economysim.ErrOutOfBounds, []string{"modules", "information", "info_speed"}},
		{"Informed with the module off", func(c *economysim.Config) { c.Modules.Information = nil }, economysim.ErrModuleDependency, []string{"roles", "cook", "informed"}},
		// 450 regions with a group of cooks, who trade at least one commodity each: S is
		// above 450, and 450 * 10 000 = 4 500 000 cells is above MaxRumourCells.
		{"too many cells", func(c *economysim.Config) {
			grow(c, 450)
			for i := range 450 {
				c.Groups = append(c.Groups, economysim.Group{Region: "r" + strconv.Itoa(i), Role: "cook", Households: 136})
			}
			c.Modules.Information.RumourMaxAge = economysim.MaxRumourAge
		}, economysim.ErrTooMany, []string{"modules", "information", "rumour_max_age"}},
		// The cook trades all 9 commodities of the pack. With 600 more regions there are
		// 601 others, and 9 * 601 = 5 409 candidates is above MaxRumoursPerView; with 400
		// more it is 9 * 401 = 3 609, within the bound.
		{"too many candidates for an informed role", func(c *economysim.Config) { tradeAll(c); grow(c, 600) }, economysim.ErrTooMany, []string{"roles", "cook", "informed"}},
		{"candidates within the bound", func(c *economysim.Config) { tradeAll(c); grow(c, 400) }, nil, nil},
		// The derived products are not computed from a field that is out of bounds.
		{"bound failure hides the derived checks", func(c *economysim.Config) {
			tradeAll(c)
			grow(c, 600)
			c.Modules.Information.RumourMaxAge = 1
		}, economysim.ErrOutOfBounds, []string{"modules", "information", "rumour_max_age"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := informationConfig()
			tt.mutate(&c)
			err := c.Validate()
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("errors.Is(Validate(), %v) = false, want true; error: %v", tt.want, err)
			}
			vs, other := violations(err)
			if len(other) != 0 || len(vs) != 1 || !slices.Equal(vs[0].Path, tt.path) {
				t.Errorf("Validate() violations = %v (others %v), want one at %v", vs, other, tt.path)
			}
		})
	}
}

// TestDefaults_Information verifies the placeholder defaults of
// spec-information.md 8: an error of 500 basis points, a horizon of 60 ticks,
// no speed (so the default alone is ErrOutOfBounds), and a role that is not
// informed.
//
// Requirement: FR-INF-032
func TestDefaults_Information(t *testing.T) {
	t.Parallel()
	d := economysim.DefaultInformation()
	if want := (economysim.Information{InfoSpeed: 0, RumourErrorBP: 500, RumourMaxAge: 60}); d != want {
		t.Errorf("DefaultInformation() = %+v, want %+v", d, want)
	}
	if economysim.DefaultRole().Informed {
		t.Errorf("DefaultRole().Informed = true, want false")
	}
	c := informationConfig()
	c.Modules.Information = &d
	if err := c.Validate(); !errors.Is(err, economysim.ErrOutOfBounds) {
		t.Errorf("Validate(DefaultInformation()) = %v, want ErrOutOfBounds: InfoSpeed has no default", err)
	}
	d.InfoSpeed = 1
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(DefaultInformation() with InfoSpeed 1) = %v, want nil", err)
	}
}

// TestBounds_Information verifies the values of the five bounds of
// spec-information.md 8 and that the products they guard fit int64: the largest
// price times the largest factor, and the cells.
//
// Requirement: FR-INF-048
func TestBounds_Information(t *testing.T) {
	t.Parallel()
	for _, b := range []struct {
		name      string
		got, want int64
	}{
		{"MaxInfoSpeed", economysim.MaxInfoSpeed, 1_000_000_000},
		{"MaxRumourErrorBP", economysim.MaxRumourErrorBP, 5_000},
		{"MaxRumourAge", economysim.MaxRumourAge, 10_000},
		{"MaxRumourCells", economysim.MaxRumourCells, 4_000_000},
		{"MaxRumoursPerView", economysim.MaxRumoursPerView, 4_096},
	} {
		if b.got != b.want {
			t.Errorf("%s = %d, want %d", b.name, b.got, b.want)
		}
	}
	// 10^9 * (10 000 + 5 000) = 1.5 * 10^13, far below 9.2 * 10^18.
	if p := int64(economysim.MaxPrice) * (10_000 + economysim.MaxRumourErrorBP); p <= 0 {
		t.Errorf("MaxPrice * (10 000 + MaxRumourErrorBP) = %d, want a positive int64", p)
	}
}

// logRecordsOf builds a world over c with a capturing logger and returns the
// records it wrote.
func logRecordsOf(t *testing.T, c economysim.Config) []logLine {
	t.Helper()
	h := newCaptureHandler()
	if _, err := economysim.NewWorld(c, economysim.WithSeed(1), economysim.WithLogger(slog.New(h))); err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	return h.records()
}

// TestWarnings_Information verifies each of the four warnings of
// spec-information.md 9.3 on its condition, at its path, with the count of
// rumour_horizon_short, and that it is quiet on the case beside it. The lags are
// 3 from north to south and 2 from south to north.
//
// Requirement: FR-INF-035, FR-INF-036, FR-INF-037, FR-INF-038, FR-CFG-076
func TestWarnings_Information(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		want   []string
	}{
		{"no informed role", func(c *economysim.Config) { c.Roles[0].Informed, c.Roles[1].Informed = false, false },
			"information_without_readers", []string{"modules/information"}},
		{"an informed role", func(c *economysim.Config) {}, "information_without_readers", nil},
		{"one region", func(c *economysim.Config) {
			*c = referenceConfig()
			m := economysim.DefaultInformation()
			m.InfoSpeed = 40
			c.Modules.Information = &m
		}, "information_without_routes", []string{"modules/information"}},
		{"two regions and no edge", func(c *economysim.Config) { c.Edges = nil }, "information_without_routes", []string{"modules/information"}},
		{"two regions and edges", func(c *economysim.Config) {}, "information_without_routes", nil},
		// The lag 3 from north to south is above a horizon of 2: one ordered pair.
		{"a horizon that cuts one pair", func(c *economysim.Config) { c.Modules.Information.RumourMaxAge = 2 },
			"rumour_horizon_short", []string{"modules/information/rumour_max_age"}},
		{"a horizon that cuts none", func(c *economysim.Config) { c.Modules.Information.RumourMaxAge = 3 }, "rumour_horizon_short", nil},
		// Nothing in the south is a source of a commodity the cook trades once the south has
		// no group; the cook in the north then has no other region with a source.
		{"an informed role with no source elsewhere", func(c *economysim.Config) { c.Groups = c.Groups[:1] },
			"informed_without_sources", []string{"roles/cook/informed"}},
		{"an informed role with a source elsewhere", func(c *economysim.Config) {}, "informed_without_sources", nil},
		{"the module off", func(c *economysim.Config) {
			c.Modules.Information = nil
			c.Roles[0].Informed, c.Roles[1].Informed = false, false
		}, "information_without_readers", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := informationConfig()
			tt.mutate(&c)
			if got := warnPaths(c, tt.rule); !slices.Equal(got, tt.want) {
				t.Errorf("Warnings() paths of %s = %v, want %v", tt.rule, got, tt.want)
			}
		})
	}
	c := informationConfig()
	c.Modules.Information.RumourMaxAge = 2
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(a config that draws a warning) = %v, want nil: a warning never fails validation", err)
	}
	for _, w := range c.Warnings() {
		if w.Rule == "rumour_horizon_short" && w.Count != 1 {
			t.Errorf("Warnings() rumour_horizon_short Count = %d, want 1 pair", w.Count)
		}
	}
	// A configuration that does not validate draws no horizon warning and does not panic.
	bad := informationConfig()
	bad.Modules.Information.InfoSpeed = 0
	if got := warnPaths(bad, "rumour_horizon_short"); len(got) != 0 {
		t.Errorf("Warnings() of a config with InfoSpeed 0 gave rumour_horizon_short at %v, want none", got)
	}
}

// TestWarnings_InformationLogged verifies that the construction logs each
// warning once at Warn with its rule and path, and the count of
// rumour_horizon_short under the key pairs, and logs no count for a warning that
// counts nothing.
//
// Requirement: FR-INF-039, FR-CFG-076
func TestWarnings_InformationLogged(t *testing.T) {
	t.Parallel()
	c := informationConfig()
	c.Modules.Information.RumourMaxAge = 2
	c.Roles[0].Informed, c.Roles[1].Informed = false, false
	var horizon, readers int
	for _, l := range logRecordsOf(t, c) {
		rule, _ := l.attr("rule")
		if l.msg != "configuration warning" || l.level != slog.LevelWarn {
			continue
		}
		pairs, hasPairs := l.attr("pairs")
		switch rule.String() {
		case "rumour_horizon_short":
			horizon++
			if !hasPairs || pairs.Int64() != 1 {
				t.Errorf("rumour_horizon_short logged pairs = %v, %v, want 1", pairs, hasPairs)
			}
		case "information_without_readers":
			readers++
			if hasPairs {
				t.Errorf("information_without_readers logged pairs = %v, want no count", pairs)
			}
		}
	}
	if horizon != 1 || readers != 1 {
		t.Errorf("construction logged rumour_horizon_short %d times and information_without_readers %d times, want once each", horizon, readers)
	}
}
