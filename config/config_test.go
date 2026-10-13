package config_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// minimal is the smallest document the loader accepts: the schema line.
const minimal = "schema = 1\n"

// parseStrings runs Parse over documents given as strings.
func parseStrings(docs ...string) (economysim.Config, error) {
	b := make([][]byte, len(docs))
	for i, d := range docs {
		b[i] = []byte(d)
	}
	return config.Parse(b...)
}

// TestParse_MinimalDocumentGetsTheDefaults verifies that a document with only a
// schema line loads to a Config whose World and Calendar are the defaults and
// whose lists are empty, so the file never restates a default.
//
// Requirement: FR-CFG-007, FR-CFG-037
func TestParse_MinimalDocumentGetsTheDefaults(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(minimal)
	if err != nil {
		t.Fatalf("Parse(minimal) error = %v, want nil", err)
	}
	want := economysim.Config{World: economysim.DefaultWorld(), Calendar: economysim.DefaultCalendar()}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse(minimal) = %+v, want the defaults %+v", got, want)
	}
	if got.WarmStart != nil {
		t.Errorf("Parse(minimal).WarmStart = %+v, want nil: no [warm_start] table", got.WarmStart)
	}
}

// TestParse_EntitiesStartFromTheirDefaults verifies that an entity table
// overlays the keys present on its Default constructor, and that an absent
// key keeps the default.
//
// Requirement: FR-CFG-007
func TestParse_EntitiesStartFromTheirDefaults(t *testing.T) {
	t.Parallel()
	doc := `schema = 1
[world]
scale = 40
[commodity.grain]
reference_price = 10
[recipe.grow]
outputs = [ { commodity = "grain", quantity = 5 } ]
[role.farmer]
target_ratio = 12000
[region.north]
capital_return = 7
`
	got, err := parseStrings(doc)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	world := economysim.DefaultWorld()
	world.Scale = 40
	commodity := economysim.DefaultCommodity()
	commodity.ID, commodity.ReferencePrice = "grain", 10
	recipe := economysim.DefaultRecipe()
	recipe.ID = "grow"
	// One non-discard output with no cost_share written: 10 000 (spec-production.md 3.2).
	recipe.Outputs = []economysim.Output{{Commodity: "grain", Quantity: 5, CostShare: 10_000}}
	role := economysim.DefaultRole()
	role.ID, role.TargetRatio = "farmer", 12_000
	region := economysim.DefaultRegion()
	region.ID, region.CapitalReturn = "north", 7
	want := economysim.Config{
		World:       world,
		Calendar:    economysim.DefaultCalendar(),
		Commodities: []economysim.Commodity{commodity},
		Recipes:     []economysim.Recipe{recipe},
		Roles:       []economysim.Role{role},
		Regions:     []economysim.Region{region},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse() = %+v, want %+v", got, want)
	}
}

// TestParse_Rejections verifies that each rejection of spec-config.md 6.3
// carries its root sentinel and the key path in the file, and that a rejected
// document returns an error.
//
// Requirement: FR-CFG-008, FR-CFG-019, FR-CFG-020, FR-CFG-021, FR-CFG-098
func TestParse_Rejections(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		docs []string
		rule error
		path []string
	}{
		{"unknown top-level key", []string{minimal + "bogus = 1\n"}, economysim.ErrUnknownKey, []string{"bogus"}},
		{"unknown key in an entity", []string{minimal + "[role.a]\nbogus = 1\n"}, economysim.ErrUnknownKey, []string{"role", "a", "bogus"}},
		{"unknown key in a world", []string{minimal + "[world]\nscael = 1\n"}, economysim.ErrUnknownKey, []string{"world", "scael"}},
		{"unknown key in an item", []string{minimal + "[world]\nupkeep = [ { commodity = \"g\", quantity = 1, extra = 2 } ]\n"}, economysim.ErrUnknownKey, []string{"world", "upkeep", "[0]", "extra"}},
		{"unknown key in a yield", []string{minimal + "[region.n.yield.r]\nhas_all = true\n"}, economysim.ErrUnknownKey, []string{"region", "n", "yield", "r", "has_all"}},
		{"unknown key in a warm job", []string{minimal + "[warm_start.region.n.group.g]\nmoney = 1\njobs = [ { recipe = \"r\", in_progress_bp = 3 } ]\n"}, economysim.ErrUnknownKey, []string{"warm_start", "region", "n", "group", "g", "jobs", "[0]", "in_progress_bp"}},
		{"module table of a module not built", []string{minimal + "[modules.estates]\n"}, economysim.ErrUnknownKey, []string{"modules", "estates"}},
		{"module sub-table makes the module unknown", []string{minimal + "[modules.demand.want.comfort]\nquantity = 1\n"}, economysim.ErrUnknownKey, []string{"modules", "demand"}},
		{"old singular prefix", []string{minimal + "[module.estates]\n"}, economysim.ErrUnknownKey, []string{"module"}},
		{"bare module table", []string{minimal + "[demand]\n"}, economysim.ErrUnknownKey, []string{"demand"}},
		{"float where integer", []string{minimal + "[role.a]\ntarget_ratio = 1.5\n"}, economysim.ErrNotInteger, []string{"role", "a", "target_ratio"}},
		{"integral float where integer", []string{minimal + "[world]\nscale = 1e3\n"}, economysim.ErrNotInteger, []string{"world", "scale"}},
		{"float in an item quantity", []string{minimal + "[role.a]\nstart_stock = [ { commodity = \"g\", quantity = 2.0 } ]\n"}, economysim.ErrNotInteger, []string{"role", "a", "start_stock", "[0]", "quantity"}},
		{"string where integer", []string{minimal + "[role.a]\ntarget_ratio = \"15000\"\n"}, economysim.ErrWrongType, []string{"role", "a", "target_ratio"}},
		{"integer where string", []string{minimal + "[role.a]\nstrategy = 3\n"}, economysim.ErrWrongType, []string{"role", "a", "strategy"}},
		{"integer where bool", []string{minimal + "[role.a]\nclosed = 1\n"}, economysim.ErrWrongType, []string{"role", "a", "closed"}},
		{"date where string", []string{minimal + "[role.a]\nlabel = 1979-05-27\n"}, economysim.ErrWrongType, []string{"role", "a", "label"}},
		{"datetime where integer", []string{minimal + "[role.a]\nstart_money = 1979-05-27T07:32:00Z\n"}, economysim.ErrWrongType, []string{"role", "a", "start_money"}},
		{"table where integer", []string{minimal + "[role.a.target_ratio]\nx = 1\n"}, economysim.ErrWrongType, []string{"role", "a", "target_ratio"}},
		{"scalar where table", []string{minimal + "role = 1\n"}, economysim.ErrWrongType, []string{"role"}},
		{"integer in a list of strings", []string{minimal + "[role.a]\nrecipes = [ \"x\", 2 ]\n"}, economysim.ErrWrongType, []string{"role", "a", "recipes", "[1]"}},
		{"list where table", []string{minimal + "[role.a]\nupkeep = 3\n"}, economysim.ErrWrongType, []string{"role", "a", "upkeep"}},
		{"scalar where an entity", []string{minimal + "[recipe]\nr = 1\n"}, economysim.ErrWrongType, []string{"recipe", "r"}},
		{"integer where an item", []string{minimal + "[world]\nupkeep = [ 1 ]\n"}, economysim.ErrWrongType, []string{"world", "upkeep", "[0]"}},
		{"missing reference price", []string{minimal + "[commodity.grain]\nmass = 1\n"}, economysim.ErrMissingField, []string{"commodity", "grain", "reference_price"}},
		{"missing outputs", []string{minimal + "[recipe.grow]\nduration = 2\n"}, economysim.ErrMissingField, []string{"recipe", "grow", "outputs"}},
		{"missing group households", []string{minimal + "[region.n.group.g]\nscale = 2\n"}, economysim.ErrMissingField, []string{"region", "n", "group", "g", "households"}},
		{"missing item quantity", []string{minimal + "[world]\nupkeep = [ { commodity = \"g\" } ]\n"}, economysim.ErrMissingField, []string{"world", "upkeep", "[0]", "quantity"}},
		{"missing output commodity", []string{minimal + "[recipe.r]\noutputs = [ { quantity = 1 } ]\n"}, economysim.ErrMissingField, []string{"recipe", "r", "outputs", "[0]", "commodity"}},
		{"missing share with two outputs", []string{minimal + "[recipe.r]\noutputs = [ { commodity = \"a\", quantity = 1 }, { commodity = \"b\", quantity = 1 } ]\n"}, economysim.ErrMissingField, []string{"recipe", "r", "outputs", "[0]", "cost_share"}},
		{"missing share of a discard output", []string{minimal + "[recipe.r]\noutputs = [ { commodity = \"a\", quantity = 1 }, { commodity = \"b\", quantity = 1, discard = true } ]\n"}, economysim.ErrMissingField, []string{"recipe", "r", "outputs", "[1]", "cost_share"}},
		{"missing wear per use", []string{minimal + "[recipe.r]\noutputs = [ { commodity = \"a\", quantity = 1 } ]\nrequires = [ { commodity = \"t\" } ]\n"}, economysim.ErrMissingField, []string{"recipe", "r", "requires", "[0]", "wear_per_use"}},
		{"missing month days", []string{minimal + "[calendar]\nmonths = [ { id = \"m\" } ]\n"}, economysim.ErrMissingField, []string{"calendar", "months", "[0]", "days"}},
		{"missing warm group money", []string{minimal + "[warm_start.region.n.group.g]\nstock = []\n"}, economysim.ErrMissingField, []string{"warm_start", "region", "n", "group", "g", "money"}},
		{"missing job share", []string{minimal + "[warm_start.region.n.group.g]\nmoney = 1\njobs = [ { recipe = \"r\" } ]\n"}, economysim.ErrMissingField, []string{"warm_start", "region", "n", "group", "g", "jobs", "[0]", "in_progress"}},
		{"no schema", []string{"[world]\nscale = 2\n"}, economysim.ErrUnsupportedSchema, []string{"schema"}},
		{"schema from the future", []string{"schema = 2\n"}, economysim.ErrUnsupportedSchema, []string{"schema"}},
		{"schema as a string", []string{"schema = \"1\"\n"}, economysim.ErrUnsupportedSchema, []string{"schema"}},
		{"schema as a float", []string{"schema = 1.0\n"}, economysim.ErrUnsupportedSchema, []string{"schema"}},
		{"table then scalar", []string{minimal + "[world]\nscale = 1\n", "world = 1\n"}, economysim.ErrMergeConflict, []string{"world"}},
		{"scalar then table", []string{minimal + "world = 1\n", "[world]\nscale = 1\n"}, economysim.ErrMergeConflict, []string{"world"}},
		{"list then table", []string{minimal + "[role.a]\nrecipes = [ \"x\" ]\n", "[role.a.recipes]\nx = 1\n"}, economysim.ErrMergeConflict, []string{"role", "a", "recipes"}},
		{"nested table then scalar", []string{minimal + "[role.a.b]\nx = 1\n", "[role.a]\nb = 1\n"}, economysim.ErrMergeConflict, []string{"role", "a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(tt.docs...)
			if err == nil {
				t.Fatalf("Parse(%q) error = nil, want %v at %q", tt.docs, tt.rule, tt.path)
			}
			if !errors.Is(err, tt.rule) {
				t.Errorf("Parse(%q) error = %v, want errors.Is %v", tt.docs, err, tt.rule)
			}
			if !hasError(err, tt.rule, tt.path) {
				t.Errorf("Parse(%q) errors = %s, want %v at %q", tt.docs, describe(err), tt.rule, tt.path)
			}
		})
	}
}

// TestParse_SyntaxErrors verifies that a document that is not TOML, or whose
// values the library refuses, is ErrSyntax with the line and column and none of
// the document's text.
//
// Requirement: FR-CFG-024, FR-CFG-100
func TestParse_SyntaxErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		line string
	}{
		{"unterminated table header", "schema = 1\n[world\n", "line 2"},
		{"duplicate key", "schema = 1\nschema = 1\n", "line 2"},
		{"integer overflow", "schema = 1\n\n[world]\nscale = 9223372036854775808\n", "line 4"},
		{"redefined table", "schema = 1\n[world]\n[world]\n", "line 3"},
		{"garbage", "\xff\xfe", "line 1"},
		{"unterminated string", "schema = 1\nk = \"abc\n", "line 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(tt.doc)
			if !errors.Is(err, economysim.ErrSyntax) {
				t.Fatalf("Parse(%q) error = %v, want errors.Is ErrSyntax", tt.doc, err)
			}
			if !strings.Contains(err.Error(), tt.line) {
				t.Errorf("Parse(%q) error = %q, want it to name %q", tt.doc, err, tt.line)
			}
			if strings.Contains(err.Error(), "abc") {
				t.Errorf("Parse(%q) error = %q, want none of the document's text", tt.doc, err)
			}
		})
	}
}

// TestParse_ErrorsAreSortedAndCapped verifies that the rejections of the
// mapping are returned together, sorted by path, at most 100, followed by
// ErrTooManyErrors that says how many were left out.
//
// Requirement: FR-CFG-102
func TestParse_ErrorsAreSortedAndCapped(t *testing.T) {
	t.Parallel()
	t.Run("sorted by path", func(t *testing.T) {
		t.Parallel()
		_, err := parseStrings(minimal + "zzz = 1\n[role.b]\nbogus = 1\n[commodity.c]\nmass = 1\n[aaa]\n")
		var got [][]string
		for _, ve := range validationErrors(err) {
			got = append(got, ve.Path)
		}
		want := [][]string{{"aaa"}, {"commodity", "c", "reference_price"}, {"role", "b", "bogus"}, {"zzz"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Parse() error paths = %q, want %q", got, want)
		}
	})
	t.Run("capped at 100", func(t *testing.T) {
		t.Parallel()
		_, err := parseStrings(minimal + flatKeys("", 130))
		if n := len(validationErrors(err)); n != 100 {
			t.Errorf("Parse(130 unknown keys) reported %d errors, want 100", n)
		}
		if !errors.Is(err, economysim.ErrTooManyErrors) {
			t.Errorf("Parse(130 unknown keys) error = %v, want errors.Is ErrTooManyErrors", err)
		}
		if !strings.Contains(err.Error(), "30 more") {
			t.Errorf("Parse(130 unknown keys) error does not say 30 were left out: %q", err.Error())
		}
	})
	t.Run("exactly 100 is not cut", func(t *testing.T) {
		t.Parallel()
		_, err := parseStrings(minimal + flatKeys("", 100))
		if errors.Is(err, economysim.ErrTooManyErrors) || len(validationErrors(err)) != 100 {
			t.Errorf("Parse(100 unknown keys) = %d errors, ErrTooManyErrors %v, want 100 and false",
				len(validationErrors(err)), errors.Is(err, economysim.ErrTooManyErrors))
		}
	})
}

// TestParse_MergeRules verifies the merge of documents in order: tables merge
// by key, a scalar or a list replaces, a later document needs no schema line,
// and the order given decides.
//
// Requirement: FR-CFG-021, FR-CFG-023, FR-CFG-034
func TestParse_MergeRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		docs []string
		got  func(economysim.Config) any
		want any
	}{
		{
			"later scalar replaces",
			[]string{minimal + "[world]\nscale = 2\n", "[world]\nscale = 3\n"},
			func(c economysim.Config) any { return c.World.Scale },
			int64(3),
		},
		{
			"earlier scalar of another key stays",
			[]string{minimal + "[world]\nscale = 2\nmin_agents = 9\n", "[world]\nscale = 3\n"},
			func(c economysim.Config) any { return c.World.MinAgents },
			int64(9),
		},
		{
			"list replaces as a whole",
			[]string{
				minimal + "[world]\nupkeep = [ { commodity = \"a\", quantity = 1 }, { commodity = \"b\", quantity = 2 } ]\n",
				"[world]\nupkeep = [ { commodity = \"c\", quantity = 3 } ]\n",
			},
			func(c economysim.Config) any { return c.World.Upkeep },
			[]economysim.Item{{Commodity: "c", Quantity: 3}},
		},
		{
			"tables merge recursively by key",
			[]string{minimal + "[role.a]\nstart_money = 5\n", "[role.a]\ntarget_ratio = 11000\n[role.b]\n"},
			func(c economysim.Config) any {
				return [3]int64{int64(len(c.Roles)), int64(c.Roles[0].StartMoney), c.Roles[0].TargetRatio}
			},
			[3]int64{2, 5, 11_000},
		},
		{
			"calendar months replace as a whole",
			[]string{
				minimal + "[calendar]\nmonths = [ { id = \"a\", days = 100 }, { id = \"b\", days = 265 } ]\n",
				"[calendar]\nmonths = [ { id = \"z\", days = 365 } ]\n",
			},
			func(c economysim.Config) any { return c.Calendar.Months },
			[]economysim.Month{{ID: "z", Days: 365}},
		},
		{
			"months and seasons keep the written order",
			[]string{minimal + "[calendar]\nseasons = [ { id = \"b\", days = 1 }, { id = \"a\", days = 2 } ]\n"},
			func(c economysim.Config) any { return c.Calendar.Seasons },
			[]economysim.Season{{ID: "b", Days: 1}, {ID: "a", Days: 2}},
		},
		{
			"schema may be in a later document only",
			[]string{"[world]\nscale = 4\n", "schema = 1\n"},
			func(c economysim.Config) any { return c.World.Scale },
			int64(4),
		},
		{
			"order given decides",
			[]string{"[world]\nscale = 3\n", minimal + "[world]\nscale = 2\n"},
			func(c economysim.Config) any { return c.World.Scale },
			int64(2),
		},
		{
			"a list may replace a scalar",
			[]string{minimal + "[role.a]\nrecipes = 3\n", "[role.a]\nrecipes = [ \"x\" ]\n"},
			func(c economysim.Config) any { return c.Roles[0].Recipes },
			[]string{"x"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := parseStrings(tt.docs...)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tt.docs, err)
			}
			if got := tt.got(cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) = %v, want %v", tt.docs, got, tt.want)
			}
		})
	}
}

// TestParse_IsDeterministicAndOrderOfKeysDoesNotMatter verifies that the same
// documents load to equal configurations, and that the order of the tables in a
// document does not change the result, because every entity list is in ID
// order.
//
// Requirement: FR-CFG-005, NFR-CFG-002
func TestParse_IsDeterministicAndOrderOfKeysDoesNotMatter(t *testing.T) {
	t.Parallel()
	a := minimal + "[role.b]\n[role.a]\n[region.y.group.q]\nhouseholds = 1\n[region.y.group.p]\nhouseholds = 2\n[region.x]\n"
	b := minimal + "[region.x]\n[region.y.group.p]\nhouseholds = 2\n[region.y.group.q]\nhouseholds = 1\n[role.a]\n[role.b]\n"
	first, err1 := parseStrings(a)
	second, err2 := parseStrings(a)
	other, err3 := parseStrings(b)
	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatalf("Parse() errors = %v, %v, %v, want nil", err1, err2, err3)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("Parse(a) twice differ: %+v and %+v", first, second)
	}
	if !reflect.DeepEqual(first, other) {
		t.Errorf("Parse(a) = %+v, Parse(b, same tables reordered) = %+v, want equal", first, other)
	}
	if first.Roles[0].ID != "a" || first.Groups[0].Role != "p" {
		t.Errorf("Parse(a) roles %v groups %v, want ID order", first.Roles, first.Groups)
	}
}

// TestParse_KeyMappingExceptions verifies the rows of spec-config.md 6.2.2: the
// group, yield and warm-start layouts, the in_progress key and the arrays of
// the calendar.
//
// Requirement: FR-CFG-103, FR-CFG-034
func TestParse_KeyMappingExceptions(t *testing.T) {
	t.Parallel()
	doc := `schema = 1
[calendar]
months = [ { id = "a", days = 100 }, { id = "b", days = 265 } ]
seasons = [ { id = "wet", days = 200 }, { id = "dry", days = 165 } ]

[region.south]
[region.north.group.worker]
households = 40
scale = 4
[region.north.group.baker]
households = 5
[region.north.yield.grow]
all = 0
seasons = { winter = 2500, spring = 10000 }
[region.north.yield.bake]
seasons = { spring = 7 }
[region.north.yield.mill]
all = 9000

[warm_start.region.north]
wage = 250
pool = 90
[warm_start.region.north.price]
grain = 3
bread = 7
[warm_start.region.north.group.worker]
money = 100
stock = [ { commodity = "grain", quantity = 2 } ]
jobs = [ { recipe = "grow", in_progress = 4000 }, { recipe = "bake", in_progress = 10 } ]
[warm_start.region.south]
wage = 1
`
	cfg, err := parseStrings(doc)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	wantGroups := []economysim.Group{
		{Region: "north", Role: "baker", Households: 5},
		{Region: "north", Role: "worker", Households: 40, Scale: 4},
	}
	if !reflect.DeepEqual(cfg.Groups, wantGroups) {
		t.Errorf("Groups = %+v, want %+v (by region, then role)", cfg.Groups, wantGroups)
	}
	wantYields := []economysim.YieldProfile{
		// A written all = 0 is HasAll, unlike an absent key (spec-config.md 6.2.2).
		{Region: "north", Recipe: "bake", Seasons: []economysim.SeasonYield{{Season: "spring", BP: 7}}},
		{Region: "north", Recipe: "grow", HasAll: true, All: 0, Seasons: []economysim.SeasonYield{{Season: "spring", BP: 10_000}, {Season: "winter", BP: 2_500}}},
		{Region: "north", Recipe: "mill", HasAll: true, All: 9_000},
	}
	if !reflect.DeepEqual(cfg.Yields, wantYields) {
		t.Errorf("Yields = %+v, want %+v", cfg.Yields, wantYields)
	}
	wantWarm := &economysim.WarmStart{
		Prices: []economysim.SeedPrice{
			{Region: "north", Commodity: "bread", Price: 7},
			{Region: "north", Commodity: "grain", Price: 3},
		},
		Wages: []economysim.SeedWage{{Region: "north", Wage: 250}, {Region: "south", Wage: 1}},
		Groups: []economysim.WarmGroup{{
			Region: "north", Role: "worker", Money: 100,
			Stock: []economysim.Item{{Commodity: "grain", Quantity: 2}},
			// in_progress is JobSpec.InProgressBP; jobs keep the order written.
			Jobs: []economysim.JobSpec{{Recipe: "grow", InProgressBP: 4_000}, {Recipe: "bake", InProgressBP: 10}},
		}},
		Pools: []economysim.WarmPool{{Region: "north", Money: 90}},
	}
	if !reflect.DeepEqual(cfg.WarmStart, wantWarm) {
		t.Errorf("WarmStart = %+v, want %+v", cfg.WarmStart, wantWarm)
	}
	wantMonths := []economysim.Month{{ID: "a", Days: 100}, {ID: "b", Days: 265}}
	if !reflect.DeepEqual(cfg.Calendar.Months, wantMonths) {
		t.Errorf("Calendar.Months = %+v, want %+v", cfg.Calendar.Months, wantMonths)
	}
	if got := cfg.Calendar.Seasons[0].ID; got != "wet" {
		t.Errorf("Calendar.Seasons[0].ID = %q, want %q: a written order is kept", got, "wet")
	}
	if len(cfg.Regions) != 2 || cfg.Regions[0].ID != "north" || cfg.Regions[1].ID != "south" {
		t.Errorf("Regions = %+v, want north and south in ID order", cfg.Regions)
	}
}

// TestParse_PresenceOfWarmStart verifies that a [warm_start] table, even an
// empty one, is a non-nil WarmStart and an absent one is nil, as the hash
// tells them apart.
//
// Requirement: FR-CFG-037
func TestParse_PresenceOfWarmStart(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		want *economysim.WarmStart
	}{
		{"absent", minimal, nil},
		{"empty table", minimal + "[warm_start]\n", &economysim.WarmStart{}},
		{"region with only a wage", minimal + "[warm_start.region.n]\nwage = 5\n", &economysim.WarmStart{Wages: []economysim.SeedWage{{Region: "n", Wage: 5}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := parseStrings(tt.doc)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tt.doc, err)
			}
			if !reflect.DeepEqual(cfg.WarmStart, tt.want) {
				t.Errorf("Parse(%q).WarmStart = %+v, want %+v", tt.doc, cfg.WarmStart, tt.want)
			}
		})
	}
}

// TestParse_EmptyModulesTable verifies that an empty [modules] table is not a
// rejection: it names no module, and at Stage 0 no module is built.
//
// Requirement: FR-CFG-098
func TestParse_EmptyModulesTable(t *testing.T) {
	t.Parallel()
	if _, err := parseStrings(minimal + "[modules]\n"); err != nil {
		t.Errorf("Parse([modules]) error = %v, want nil", err)
	}
}

// TestParse_SectionScalarKeysAreTheSnakeCaseOfTheGoName verifies the key rule
// of spec-config.md 6.2: every integer, string and bool field of a section type
// is read from the snake case of its Go name, and none is dropped.
//
// Guard: the loader's tables are written by hand. A field added to a struct and
// not to its table could never be set from a file, and nothing else would
// notice, because the loader only sees the keys a file writes.
//
// Requirement: FR-CFG-103
func TestParse_SectionScalarKeysAreTheSnakeCaseOfTheGoName(t *testing.T) {
	t.Parallel()
	const outputs = `outputs = [ { commodity = "c", quantity = 1, cost_share = 10000 } ]` + "\n"
	tests := []struct {
		name   string
		typ    reflect.Type
		header string
		extra  string
		pick   func(economysim.Config) reflect.Value
	}{
		{"World", reflect.TypeFor[economysim.World](), "[world]", "", func(c economysim.Config) reflect.Value { return reflect.ValueOf(c.World) }},
		{"Calendar", reflect.TypeFor[economysim.Calendar](), "[calendar]", "", func(c economysim.Config) reflect.Value { return reflect.ValueOf(c.Calendar) }},
		{"Commodity", reflect.TypeFor[economysim.Commodity](), "[commodity.x]", "", func(c economysim.Config) reflect.Value { return reflect.ValueOf(c.Commodities[0]) }},
		{"Recipe", reflect.TypeFor[economysim.Recipe](), "[recipe.x]", outputs, func(c economysim.Config) reflect.Value { return reflect.ValueOf(c.Recipes[0]) }},
		{"Role", reflect.TypeFor[economysim.Role](), "[role.x]", "", func(c economysim.Config) reflect.Value { return reflect.ValueOf(c.Roles[0]) }},
		{"Region", reflect.TypeFor[economysim.Region](), "[region.x]", "", func(c economysim.Config) reflect.Value { return reflect.ValueOf(c.Regions[0]) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var b strings.Builder
			b.WriteString("schema = 1\n" + tt.header + "\n" + tt.extra)
			var keys []string
			for i := range tt.typ.NumField() {
				f := tt.typ.Field(i)
				if f.Name == "ID" {
					continue
				}
				var lit string
				switch f.Type.Kind() {
				case reflect.Int64:
					lit = "7"
				case reflect.String:
					lit = `"s"`
				case reflect.Bool:
					lit = "true"
				default:
					continue
				}
				keys = append(keys, f.Name)
				b.WriteString(snake(f.Name) + " = " + lit + "\n")
			}
			cfg, err := parseStrings(b.String())
			if err != nil {
				t.Fatalf("Parse(every scalar key of %s) error = %v, want nil", tt.name, err)
			}
			v := tt.pick(cfg)
			for _, name := range keys {
				f := v.FieldByName(name)
				var ok bool
				switch f.Kind() {
				case reflect.Int64:
					ok = f.Int() == 7
				case reflect.String:
					ok = f.String() == "s"
				case reflect.Bool:
					ok = f.Bool()
				}
				if !ok {
					t.Errorf("%s.%s = %v after writing key %q, want the written value", tt.name, name, f, snake(name))
				}
			}
		})
	}
}

// TestSnake_RunsOfCapitalsAreOneWord verifies the test's own snake-case rule
// against the spec's examples, so the guard above checks the right keys.
//
// Requirement: FR-CFG-103
func TestSnake_RunsOfCapitalsAreOneWord(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"TargetRatio", "target_ratio"}, {"RateBP", "rate_bp"}, {"ID", "id"},
		{"InProgressBP", "in_progress_bp"}, {"Scale", "scale"}, {"UnitLabel", "unit_label"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := snake(tt.in); got != tt.want {
				t.Errorf("snake(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
