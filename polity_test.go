package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the polity module's configuration surface (spec-polity.md 8, 9):
// the failing case of each rule, their order, the two warnings and the hash. The
// tables and queries are tested in internal/polity and the neutrality in
// scenario_s112_test.go.

// polityConfig returns referenceConfig with the polity module on in the form of
// example A of spec-polity.md 12.1: five regions (capital, farm, harbour, hills,
// mill) and the reference region north added to the crown, the crown and the inland
// province governed. The jurisdictions are written in the order of the spec's file
// (crown, coast, inland, uplands), not sorted, so the given index and the
// JurisdictionID differ.
func polityConfig() economysim.Config {
	c := referenceConfig()
	for _, id := range []string{"capital", "farm", "harbour", "hills", "mill"} {
		r := economysim.DefaultRegion()
		r.ID = id
		c.Regions = append(c.Regions, r)
	}
	c.Modules.Polity = &economysim.PolityConfig{
		Jurisdictions: []economysim.Jurisdiction{
			{ID: "crown", Seat: "capital", StartMoney: 50_000},
			{ID: "coast", Parent: "crown"},
			{ID: "inland", Parent: "crown", Seat: "mill", StartMoney: 5_000},
			{ID: "uplands", Parent: "inland"},
		},
		Members: []economysim.Member{
			{Region: "capital", Jurisdiction: "crown"},
			{Region: "farm", Jurisdiction: "inland"},
			{Region: "harbour", Jurisdiction: "coast"},
			{Region: "hills", Jurisdiction: "uplands"},
			{Region: "mill", Jurisdiction: "inland"},
			{Region: "north", Jurisdiction: "crown"},
		},
	}
	return c
}

// polityOutsideConfig is polityConfig on top of the external configuration: the
// region coast is an outside node and a member of the crown, so a seat there is the
// one rule that needs another module.
func polityOutsideConfig() economysim.Config {
	c := externalConfig()
	c.Modules.Polity = &economysim.PolityConfig{
		Jurisdictions: []economysim.Jurisdiction{{ID: "realm"}},
		Members: []economysim.Member{
			{Region: "coast", Jurisdiction: "realm"}, {Region: "far", Jurisdiction: "realm"}, {Region: "north", Jurisdiction: "realm"},
		},
	}
	return c
}

// TestValidatePolity_ConfigIsValid verifies that the configurations the polity
// tests start from validate with no warning, so every failing case breaks one rule.
//
// Guard: the field cases start from polityConfig; if it did not validate, each case
// would pass for the wrong reason.
func TestValidatePolity_ConfigIsValid(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]economysim.Config{"polityConfig": polityConfig(), "polityOutsideConfig": polityOutsideConfig()} {
		if err := c.Validate(); err != nil {
			t.Errorf("%s().Validate() = %v, want nil", name, err)
		}
		for _, w := range c.Warnings() {
			if w.Rule == "jurisdiction_empty" || w.Rule == "region_unassigned" {
				t.Errorf("%s().Warnings() has %v, want none of the polity's", name, w)
			}
		}
	}
}

// polityFieldCases are the failing cases of the rows of the polity module in the
// parameter-owner table (spec-polity.md 8, 9.1, FR-CFG-092): each breaks one field
// of the valid configuration of polityConfig. buildFieldCases appends them to the
// table that the registry test reads.
func polityFieldCases() []fieldCase {
	field := func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config, m *economysim.PolityConfig)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = polityConfig()
				set(c, c.Modules.Polity)
			}}
	}
	j := func(i int, key string) []string {
		return []string{"modules", "polity", "jurisdictions[" + strconv.Itoa(i) + "]", key}
	}
	m := func(i int, key string) []string {
		return []string{"modules", "polity", "members[" + strconv.Itoa(i) + "]", key}
	}
	// chain adds n jurisdictions below uplands, each the child of the one before, so
	// the last has 3 + n ancestors.
	chain := func(n int) func(*economysim.Config, *economysim.PolityConfig) {
		return func(_ *economysim.Config, p *economysim.PolityConfig) {
			parent := "uplands"
			for i := range n {
				id := "deep" + strconv.Itoa(i)
				p.Jurisdictions = append(p.Jurisdictions, economysim.Jurisdiction{ID: id, Parent: parent})
				parent = id
			}
		}
	}
	return []fieldCase{
		field("PolityConfig.Jurisdictions", "more than MaxJurisdictions", economysim.ErrTooMany, []string{"modules", "polity", "jurisdictions"},
			"1004", "1000", func(_ *economysim.Config, p *economysim.PolityConfig) {
				for i := range economysim.MaxJurisdictions {
					p.Jurisdictions = append(p.Jurisdictions, economysim.Jurisdiction{ID: "x" + strconv.Itoa(i)})
				}
			}),
		field("PolityConfig.Members", "more than MaxRegions", economysim.ErrTooMany, []string{"modules", "polity", "members"},
			"1006", "1000", func(_ *economysim.Config, p *economysim.PolityConfig) {
				for range economysim.MaxRegions {
					p.Members = append(p.Members, economysim.Member{Region: "north", Jurisdiction: "crown"})
				}
			}),
		field("Jurisdiction.ID", "not an ID", economysim.ErrInvalidID, j(1, "id"), "Bad ID", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Jurisdictions[1].ID = "Bad ID" }),
		field("Jurisdiction.ID", "twice", economysim.ErrDuplicateID, j(1, "id"), "crown", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Jurisdictions[1].ID = "crown" }),
		field("Jurisdiction.Parent", "a jurisdiction that is not defined", economysim.ErrUnknownJurisdiction, j(1, "parent"), "nowhere", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Jurisdictions[1].Parent = "nowhere" }),
		field("Jurisdiction.Parent", "itself", economysim.ErrJurisdictionCycle, j(1, "parent"), "coast", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Jurisdictions[1].Parent = "coast" }),
		field("Jurisdiction.Parent", "a parent chain that returns", economysim.ErrJurisdictionCycle, j(0, "parent"), "uplands", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Jurisdictions[0].Parent = "uplands" }),
		// crown, inland and uplands are 3 levels; 5 more make 8 entries, which builds,
		// and a sixth (index 9, the tenth row) makes 9 and has 8 ancestors.
		field("Jurisdiction.Parent", "MaxPolityDepth or more ancestors", economysim.ErrOutOfBounds, j(9, "parent"), "deep4", "8", chain(6)),
		field("Jurisdiction.Seat", "a region that is not defined", economysim.ErrUnknownRegion, j(1, "seat"), "nowhere", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Jurisdictions[1].Seat = "nowhere" }),
		field("Jurisdiction.Seat", "a region the jurisdiction does not cover", economysim.ErrSeatNotCovered, j(1, "seat"), "capital", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Jurisdictions[1].Seat = "capital" }),
		fieldCase{field: "Jurisdiction.Seat", name: "an outside node", rules: []error{economysim.ErrSeatOutsideNode},
			paths: [][]string{{"modules", "polity", "jurisdictions[0]", "seat"}}, value: "coast",
			mutate: func(c *economysim.Config) {
				*c = polityOutsideConfig()
				c.Modules.Polity.Jurisdictions[0].Seat = "coast"
			}},
		field("Jurisdiction.StartMoney", "money without a seat", economysim.ErrStartWithoutSeat, j(1, "start_money"), "7", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Jurisdictions[1].StartMoney = 7 }),
		field("Jurisdiction.StartMoney", "above MaxMoney", economysim.ErrOutOfBounds, j(0, "start_money"),
			strconv.FormatInt(economysim.MaxMoney+1, 10), strconv.FormatInt(economysim.MaxMoney, 10),
			func(_ *economysim.Config, p *economysim.PolityConfig) {
				p.Jurisdictions[0].StartMoney = economysim.MaxMoney + 1
			}),
		field("Jurisdiction.StartMoney", "below 0", economysim.ErrOutOfBounds, j(0, "start_money"), "-1", "0",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Jurisdictions[0].StartMoney = -1 }),
		field("Jurisdiction.StartMoney", "a total above MaxMoney", economysim.ErrOverflow, []string{"modules", "polity", "jurisdictions"}, "", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) {
				p.Jurisdictions[0].StartMoney, p.Jurisdictions[2].StartMoney = economysim.MaxMoney, economysim.MaxMoney
			}),
		field("Jurisdiction.Label", "over MaxLabelBytes", economysim.ErrOutOfBounds, j(1, "label"), "257", "256",
			func(_ *economysim.Config, p *economysim.PolityConfig) {
				p.Jurisdictions[1].Label = strings.Repeat("x", 257)
			}),
		field("Member.Region", "a region that is not defined", economysim.ErrUnknownRegion, m(0, "region"), "nowhere", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Members[0].Region = "nowhere" }),
		field("Member.Region", "two rows for one region", economysim.ErrDuplicateID, m(1, "region"), "capital", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Members[1].Region = "capital" }),
		field("Member.Jurisdiction", "a jurisdiction that is not defined", economysim.ErrUnknownJurisdiction, m(0, "jurisdiction"), "nowhere", "",
			func(_ *economysim.Config, p *economysim.PolityConfig) { p.Members[0].Jurisdiction = "nowhere" }),
	}
}

// TestValidatePolity_Errors verifies each rule of spec-polity.md 9.1 with a
// configuration that breaks only that rule: the sentinel by errors.Is, the path of
// the offending row by its index in the order written, and the value.
//
// Requirement: FR-POL-002, FR-POL-003, FR-POL-004, FR-POL-005, FR-POL-014
func TestValidatePolity_Errors(t *testing.T) {
	t.Parallel()
	for _, tt := range polityFieldCases() {
		t.Run(tt.field+" "+tt.name, func(t *testing.T) {
			t.Parallel()
			var c economysim.Config
			tt.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, tt.rules[0]) {
				t.Fatalf("Validate() = %v, want %v", err, tt.rules[0])
			}
			var found bool
			for _, e := range unwrapAll(err) {
				var ve *economysim.ValidationError
				if errors.As(e, &ve) && errors.Is(ve, tt.rules[0]) && slices.Equal(ve.Path, tt.paths[0]) {
					found = true
					if tt.value != "" && ve.Value != tt.value {
						t.Errorf("Validate() value at %v = %q, want %q", ve.Path, ve.Value, tt.value)
					}
				}
			}
			if !found {
				t.Errorf("Validate() = %v, want %v at %v", err, tt.rules[0], tt.paths[0])
			}
		})
	}
}

// unwrapAll returns the errors a joined error holds, or the error itself.
func unwrapAll(err error) []error {
	if j, ok := err.(interface{ Unwrap() []error }); ok {
		return j.Unwrap()
	}
	return []error{err}
}

// TestValidatePolity_Order verifies that a configuration with two mistakes reports
// the one the order of spec-polity.md 9.1 puts first and none of the later ones: a
// parent that is not defined (parents) hides an unknown region of a member row
// (members), and so on down the order.
//
// Requirement: FR-POL-014
//
// Guard: the first error must be the same for the same mistake; a validator that
// reported both, sorted by path, would put the member row first.
func TestValidatePolity_Order(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		break_ []func(p *economysim.PolityConfig)
		first  error
		later  error
	}{
		{"duplicate ID before a parent", []func(*economysim.PolityConfig){
			func(p *economysim.PolityConfig) { p.Jurisdictions[3].ID = "crown" },
			func(p *economysim.PolityConfig) { p.Jurisdictions[1].Parent = "nowhere" },
		}, economysim.ErrDuplicateID, economysim.ErrUnknownJurisdiction},
		{"unknown parent before a cycle", []func(*economysim.PolityConfig){
			func(p *economysim.PolityConfig) { p.Jurisdictions[1].Parent = "nowhere" },
			func(p *economysim.PolityConfig) { p.Jurisdictions[0].Parent = "uplands" },
		}, economysim.ErrUnknownJurisdiction, economysim.ErrJurisdictionCycle},
		{"cycle before a member row", []func(*economysim.PolityConfig){
			func(p *economysim.PolityConfig) { p.Jurisdictions[1].Parent = "coast" },
			func(p *economysim.PolityConfig) { p.Members[0].Region = "nowhere" },
		}, economysim.ErrJurisdictionCycle, economysim.ErrUnknownRegion},
		{"member row before a seat", []func(*economysim.PolityConfig){
			func(p *economysim.PolityConfig) { p.Members[0].Region = "nowhere" },
			func(p *economysim.PolityConfig) { p.Jurisdictions[1].Seat = "nowhere" },
		}, economysim.ErrUnknownRegion, economysim.ErrSeatNotCovered},
		{"seat before the money", []func(*economysim.PolityConfig){
			func(p *economysim.PolityConfig) { p.Jurisdictions[1].Seat = "capital" },
			func(p *economysim.PolityConfig) { p.Jurisdictions[3].StartMoney = 7 },
		}, economysim.ErrSeatNotCovered, economysim.ErrStartWithoutSeat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := polityConfig()
			for _, b := range tt.break_ {
				b(c.Modules.Polity)
			}
			err := c.Validate()
			if !errors.Is(err, tt.first) || errors.Is(err, tt.later) {
				t.Errorf("Validate() = %v, want %v and not %v", err, tt.first, tt.later)
			}
		})
	}
}

// ungoverned removes the seats and the money of c, so that removing member rows
// does not break the rule that a seat is covered.
func ungoverned(c economysim.Config) economysim.Config {
	for i := range c.Modules.Polity.Jurisdictions {
		c.Modules.Polity.Jurisdictions[i].Seat, c.Modules.Polity.Jurisdictions[i].StartMoney = "", 0
	}
	return c
}

// TestValidatePolity_Warnings verifies the two warnings of spec-polity.md 9.2:
// jurisdiction_empty for a jurisdiction that covers no region, region_unassigned
// once for each region with no row when another has one (an outside node draws
// none), neither failing validation, each with its path; and none when the module
// has no member row at all.
//
// Requirement: FR-POL-026
func TestValidatePolity_Warnings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		build func() economysim.Config
		want  []string // rule@path of the polity's warnings, in order
	}{
		{"a jurisdiction that covers nothing", func() economysim.Config {
			c := polityConfig()
			c.Modules.Polity.Jurisdictions = append(c.Modules.Polity.Jurisdictions, economysim.Jurisdiction{ID: "annex"})
			return c
		}, []string{"jurisdiction_empty@modules.polity.jurisdiction.annex"}},
		{"a parent covers through its child", func() economysim.Config {
			c := ungoverned(polityConfig())
			c.Modules.Polity.Members = c.Modules.Polity.Members[:2] // capital, farm: crown and inland cover, uplands and coast do not
			return c
		}, []string{
			"jurisdiction_empty@modules.polity.jurisdiction.coast",
			"jurisdiction_empty@modules.polity.jurisdiction.uplands",
			"region_unassigned@modules.polity.member.harbour",
			"region_unassigned@modules.polity.member.hills",
			"region_unassigned@modules.polity.member.mill",
			"region_unassigned@modules.polity.member.north",
		}},
		{"no member row at all", func() economysim.Config {
			c := ungoverned(polityConfig())
			c.Modules.Polity.Members = nil
			return c
		}, []string{
			"jurisdiction_empty@modules.polity.jurisdiction.coast",
			"jurisdiction_empty@modules.polity.jurisdiction.crown",
			"jurisdiction_empty@modules.polity.jurisdiction.inland",
			"jurisdiction_empty@modules.polity.jurisdiction.uplands",
		}},
		{"an outside node draws none", func() economysim.Config {
			c := polityOutsideConfig()
			c.Modules.Polity.Members = c.Modules.Polity.Members[1:] // coast is unassigned and is the outside node
			return c
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.build()
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil (a warning never fails validation)", err)
			}
			var got []string
			for _, w := range c.Warnings() {
				if w.Rule == "jurisdiction_empty" || w.Rule == "region_unassigned" {
					got = append(got, w.Rule+"@"+strings.Join(w.Path, "."))
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Warnings() polity rows = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPolityHash verifies spec-polity.md 11 and FR-POL-022: the module off leaves
// the hash of the same configuration as it was, the module on moves it, the
// jurisdictions and members hash in canonical order whatever order they were
// written in, the label does not, an empty module that is on differs from one that
// is off, and a change of a parent, a member, a seat or a StartMoney moves it.
//
// Requirement: FR-POL-022
func TestPolityHash(t *testing.T) {
	t.Parallel()
	off := polityConfig()
	off.Modules.Polity = nil
	on := polityConfig()
	hash := func(c economysim.Config) string { return hexHash(c) }
	if hash(off) == hash(on) {
		t.Error("HashConfig of the module on equals that of the module off, want it hashed")
	}
	empty := polityConfig()
	empty.Modules.Polity = &economysim.PolityConfig{}
	if hash(empty) == hash(off) {
		t.Error("HashConfig of the empty module equals that of the module off, want a pointer that is on to be written")
	}
	reversed := polityConfig()
	slices.Reverse(reversed.Modules.Polity.Jurisdictions)
	slices.Reverse(reversed.Modules.Polity.Members)
	if hash(reversed) != hash(on) {
		t.Error("HashConfig changes with the order the rows were written in, want the canonical order")
	}
	for name, mutate := range map[string]func(p *economysim.PolityConfig){
		"label":     func(p *economysim.PolityConfig) { p.Jurisdictions[1].Label = "the coast" },
		"unlabeled": func(p *economysim.PolityConfig) { p.Jurisdictions[0].Label = "" },
	} {
		c := polityConfig()
		mutate(c.Modules.Polity)
		if hash(c) != hash(on) {
			t.Errorf("HashConfig moves with the %s, want presentation only", name)
		}
	}
	for name, mutate := range map[string]func(p *economysim.PolityConfig){
		"a parent":     func(p *economysim.PolityConfig) { p.Jurisdictions[3].Parent = "crown" },
		"a member":     func(p *economysim.PolityConfig) { p.Members[1].Jurisdiction = "crown" },
		"a seat":       func(p *economysim.PolityConfig) { p.Jurisdictions[2].Seat = "farm" },
		"a StartMoney": func(p *economysim.PolityConfig) { p.Jurisdictions[2].StartMoney++ },
		"an ID": func(p *economysim.PolityConfig) {
			p.Jurisdictions[3].ID = "highlands"
			p.Members[3].Jurisdiction = "highlands"
		},
		"a row less": func(p *economysim.PolityConfig) { p.Members = p.Members[:5] },
	} {
		c := polityConfig()
		mutate(c.Modules.Polity)
		if hash(c) == hash(on) {
			t.Errorf("HashConfig does not move with %s, want it hashed", name)
		}
	}
}

// FuzzPolityValidate verifies that Validate never panics on random jurisdiction
// and member rows, and that the polity's warnings are produced for the same input
// without a panic or a loop, whether or not the configuration is valid.
func FuzzPolityValidate(f *testing.F) {
	f.Add("crown", "", "capital", "capital", "crown", int64(5))
	f.Add("a", "a", "nowhere", "farm", "b", int64(-1))
	f.Add("x", "y", "farm", "mill", "x", int64(1)<<62)
	f.Fuzz(func(t *testing.T, id, parent, seat, region, member string, money int64) {
		c := polityConfig()
		c.Modules.Polity = &economysim.PolityConfig{
			Jurisdictions: []economysim.Jurisdiction{{ID: id, Parent: parent, Seat: seat, StartMoney: economysim.Money(money)}, {ID: parent, Parent: id}},
			Members:       []economysim.Member{{Region: region, Jurisdiction: member}},
		}
		_ = c.Validate()
		_ = c.Warnings()
	})
}
