package economysim

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/production"
)

// yieldConfig is a canonical configuration for the yield table: the calendar
// of the bread chain of spec-production.md 12.1 declared spring, summer,
// autumn, winter (indexes 0 to 3, against the ID order autumn, spring, summer,
// winter), four regions and three recipes, and the profiles of 12.1. Regions
// sort to arid 0, far_south 1, north 2, south 3; recipes to bake_loaf 0,
// grow_grain 1, mine_iron 2.
func yieldConfig() *Config {
	c := Config{
		Calendar: Calendar{Seasons: []Season{{ID: "spring"}, {ID: "summer"}, {ID: "autumn"}, {ID: "winter"}}},
		Regions:  []Region{{ID: "north"}, {ID: "south"}, {ID: "far_south"}, {ID: "arid"}},
		Recipes:  []Recipe{{ID: "grow_grain", Sited: true}, {ID: "mine_iron", Sited: true}, {ID: "bake_loaf"}},
		Yields: []YieldProfile{
			{Region: "north", Recipe: "grow_grain", Seasons: []SeasonYield{{Season: "spring", BP: 10_000}}},
			{Region: "south", Recipe: "grow_grain", Seasons: []SeasonYield{{Season: "spring", BP: 12_000}, {Season: "autumn", BP: 7_000}}},
			{Region: "far_south", Recipe: "grow_grain", Seasons: []SeasonYield{{Season: "autumn", BP: 10_000}}},
			{Region: "north", Recipe: "mine_iron", HasAll: true, All: 10_000},
			{Region: "south", Recipe: "mine_iron", HasAll: true, All: 5_000},
			{Region: "far_south", Recipe: "mine_iron", HasAll: true, All: 15_000},
		},
	}
	cc := c.canonical()
	return &cc
}

// TestYieldTable_Site verifies the site factor and the has flag for the
// profiles of spec-production.md 12.1: a seasonal profile gives its entry for
// a listed season and 0 for the others, an all profile gives its value in
// every season, a Sited recipe with no profile (the arid region) gives 0 with
// has false, and a recipe that is not Sited with no profile runs everywhere at
// 10 000. A season of -1 (no calendar seasons) matches no entry.
//
// Requirement: FR-PRO-020, FR-PRO-028
func TestYieldTable_Site(t *testing.T) {
	t.Parallel()
	c := yieldConfig()
	tab := newYieldTable(c)
	const (
		spring, summer, autumn, winter = 0, 1, 2, 3
	)
	tests := []struct {
		name       string
		region     string
		recipe     string
		season     int
		wantFactor int64
		wantHas    bool
	}{
		{"north grain spring", "north", "grow_grain", spring, 10_000, true},
		{"north grain summer has no entry", "north", "grow_grain", summer, 0, true},
		{"north grain no season", "north", "grow_grain", -1, 0, true},
		{"south grain spring", "south", "grow_grain", spring, 12_000, true},
		{"south grain autumn", "south", "grow_grain", autumn, 7_000, true},
		{"south grain winter", "south", "grow_grain", winter, 0, true},
		{"far south grain autumn", "far_south", "grow_grain", autumn, 10_000, true},
		{"far south grain spring", "far_south", "grow_grain", spring, 0, true},
		{"arid grain has no profile and is sited", "arid", "grow_grain", spring, 0, false},
		{"north iron any season", "north", "mine_iron", winter, 10_000, true},
		{"south iron", "south", "mine_iron", summer, 5_000, true},
		{"far south iron", "far_south", "mine_iron", spring, 15_000, true},
		{"north iron with no season", "north", "mine_iron", -1, 10_000, true},
		{"arid iron has no profile and is sited", "arid", "mine_iron", spring, 0, false},
		{"unsited recipe with no profile runs everywhere", "arid", "bake_loaf", winter, 10_000, false},
		{"unsited recipe in a profiled region", "north", "bake_loaf", spring, 10_000, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			region, ok1 := c.regionID(tt.region)
			recipe, ok2 := c.recipeID(tt.recipe)
			if !ok1 || !ok2 {
				t.Fatalf("setup: region %q or recipe %q not found", tt.region, tt.recipe)
			}
			sited := c.Recipes[recipe].Sited
			got, has := tab.site(region, recipe, tt.season, sited)
			if got != tt.wantFactor || has != tt.wantHas {
				t.Errorf("site(%s, %s, season %d) = (%d, %v), want (%d, %v)",
					tt.region, tt.recipe, tt.season, got, has, tt.wantFactor, tt.wantHas)
			}
		})
	}
}

// TestYieldTable_NewRegionByProfileOnly verifies that two configurations that
// differ in one profile row and nothing else differ in the factor of that row
// only: the arid region cannot run grow_grain, and with one profile added it
// runs it, with no second recipe and no other change.
//
// Requirement: FR-PRO-028
func TestYieldTable_NewRegionByProfileOnly(t *testing.T) {
	t.Parallel()
	without := yieldConfig()
	with := Config{Calendar: without.Calendar, Regions: without.Regions, Recipes: without.Recipes,
		Yields: append(slices.Clone(without.Yields), YieldProfile{Region: "arid", Recipe: "grow_grain", HasAll: true, All: 6_000})}
	withC := with.canonical()
	arid, _ := without.regionID("arid")
	grain, _ := without.recipeID("grow_grain")
	north, _ := without.regionID("north")
	a := newYieldTable(without)
	b := newYieldTable(&withC)
	if got, has := a.site(arid, grain, 0, true); got != 0 || has {
		t.Errorf("without the row: site(arid, grow_grain) = (%d, %v), want (0, false)", got, has)
	}
	if got, has := b.site(arid, grain, 0, true); got != 6_000 || !has {
		t.Errorf("with the row: site(arid, grow_grain) = (%d, %v), want (6000, true)", got, has)
	}
	// The row beside it is untouched: the factor of a profile in another
	// region is the same in both tables.
	ga, _ := a.site(north, grain, 0, true)
	gb, _ := b.site(north, grain, 0, true)
	if ga != gb {
		t.Errorf("site(north, grow_grain) = %d with the row and %d without, want equal", gb, ga)
	}
}

// TestYieldTable_SeasonsByDeclarationOrder verifies that a calendar whose
// seasons are declared winter, spring (indexes 0 and 1, against the ID order
// spring, winter) with a profile naming both gives each season its own
// factor, and that the row holds its seasons ascending by index.
//
// Guard: Config.Yields lists a profile's seasons by ID while the calendar
// orders them by declaration. An index taken from the sorted profile would
// give winter the factor of spring and the reverse, and a one-season-per-ID
// test would not notice.
//
// Requirement: FR-PRO-020
func TestYieldTable_SeasonsByDeclarationOrder(t *testing.T) {
	t.Parallel()
	c := Config{
		Calendar: Calendar{Seasons: []Season{{ID: "winter"}, {ID: "spring"}}},
		Regions:  []Region{{ID: "r"}},
		Recipes:  []Recipe{{ID: "x", Sited: true}},
		// winter = 3 000, spring = 8 000. Canonical order lists spring first.
		Yields: []YieldProfile{{Region: "r", Recipe: "x", Seasons: []SeasonYield{{Season: "winter", BP: 3_000}, {Season: "spring", BP: 8_000}}}},
	}
	cc := c.canonical()
	tab := newYieldTable(&cc)
	for _, tt := range []struct {
		name   string
		season int
		want   int64
	}{{"winter is index 0", 0, 3_000}, {"spring is index 1", 1, 8_000}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, _ := tab.site(0, 0, tt.season, true); got != tt.want {
				t.Errorf("site(r, x, season %d) = %d, want %d", tt.season, got, tt.want)
			}
		})
	}
	want := []production.SeasonBP{{Season: 0, BP: 3_000}, {Season: 1, BP: 8_000}}
	if got := tab.rows[0].profile.Seasons; !slices.Equal(got, want) {
		t.Errorf("row seasons = %v, want %v ascending by index", got, want)
	}
}

// TestYieldShocks_Set verifies the shock slot: an absent key reads 10 000, a
// value outside [0, MaxYieldBP] is refused with ErrOutOfBounds and changes
// nothing, a value of 10 000 removes the key, the bounds themselves are
// accepted, and the ordered read is sorted by (region, recipe) whatever the
// order of the sets.
//
// Requirement: FR-PRO-023
func TestYieldShocks_Set(t *testing.T) {
	t.Parallel()
	var s yieldShocks
	if got := s.factor(1, 2); got != 10_000 {
		t.Errorf("factor of an absent key = %d, want 10000", got)
	}
	for _, bp := range []int64{-1, MaxYieldBP + 1} {
		if err := s.set(1, 2, bp); !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("set(1, 2, %d) error = %v, want ErrOutOfBounds", bp, err)
		}
	}
	if len(s.ordered()) != 0 {
		t.Errorf("slot after refused sets = %v, want empty", s.ordered())
	}
	// Set out of order: (1,0), (0,2), (0,1), then the bounds.
	for _, e := range []shockEntry{{1, 0, 3_000}, {0, 2, 0}, {0, 1, MaxYieldBP}} {
		if err := s.set(e.region, e.recipe, e.bp); err != nil {
			t.Fatalf("set(%d, %d, %d) error = %v, want nil", e.region, e.recipe, e.bp, err)
		}
	}
	want := []shockEntry{{0, 1, MaxYieldBP}, {0, 2, 0}, {1, 0, 3_000}}
	if got := s.ordered(); !slices.Equal(got, want) {
		t.Errorf("ordered = %v, want %v", got, want)
	}
	// A refused set leaves an existing key as it was.
	if err := s.set(1, 0, -5); !errors.Is(err, ErrOutOfBounds) || s.factor(1, 0) != 3_000 {
		t.Errorf("refused set over a key: error %v, factor %d, want ErrOutOfBounds and 3000", err, s.factor(1, 0))
	}
	// Overwrite, then set back to neutral: the key goes.
	if err := s.set(1, 0, 4_000); err != nil || s.factor(1, 0) != 4_000 || len(s.ordered()) != 3 {
		t.Errorf("overwrite: error %v, factor %d, len %d, want nil, 4000, 3", err, s.factor(1, 0), len(s.ordered()))
	}
	if err := s.set(0, 2, 10_000); err != nil {
		t.Fatalf("set neutral error = %v, want nil", err)
	}
	want = []shockEntry{{0, 1, MaxYieldBP}, {1, 0, 4_000}}
	if got := s.ordered(); !slices.Equal(got, want) {
		t.Errorf("ordered after neutral = %v, want %v", got, want)
	}
	// Neutral on an absent key adds nothing.
	if err := s.set(5, 5, 10_000); err != nil || len(s.ordered()) != 2 {
		t.Errorf("neutral on absent key: error %v, len %d, want nil and 2", err, len(s.ordered()))
	}
}

// TestYieldTable_Effective verifies that the shock multiplies the site factor
// of its (region, recipe) only, and the D59 split of the yield pair: a Sited
// recipe with no profile is no_site whatever the shock is, a profile of 0 or a
// shock of 0 is zero_yield, and a positive factor gives no reason.
//
// Requirement: FR-PRO-022, FR-PRO-023, FR-PRO-009
//
// Guard: no_site and zero_yield are disjoint by their cause, which is the
// profile (spec-production.md 4.2.2). The strategy sees the reason.
func TestYieldTable_Effective(t *testing.T) {
	t.Parallel()
	c := yieldConfig()
	tab := newYieldTable(c)
	north, _ := c.regionID("north")
	south, _ := c.regionID("south")
	arid, _ := c.regionID("arid")
	grain, _ := c.recipeID("grow_grain")
	bake, _ := c.recipeID("bake_loaf")
	var sh yieldShocks
	if err := sh.set(north, grain, 3_000); err != nil {
		t.Fatalf("setup: set error = %v", err)
	}
	// Shock of the arid grain is 0 and a shock on an unsited recipe with no
	// profile is 0 too: the first still reads no_site, the second zero_yield.
	if err := sh.set(arid, grain, 0); err != nil {
		t.Fatalf("setup: set error = %v", err)
	}
	if err := sh.set(north, bake, 0); err != nil {
		t.Fatalf("setup: set error = %v", err)
	}
	tests := []struct {
		name       string
		region     RegionID
		recipe     RecipeID
		season     int
		wantE      int64
		wantReason Reason
	}{
		// 10 000 * 3 000 / 10 000 = 3 000, then the neutral terms.
		{"shock multiplies its pair", north, grain, 0, 3_000, ""},
		{"other region untouched", south, grain, 0, 12_000, ""},
		{"profile with no entry for the season", north, grain, 1, 0, ReasonZeroYield},
		{"sited with no profile is no_site even with shock 0", arid, grain, 0, 0, ReasonNoSite},
		{"unsited with a shock of 0 is zero_yield", north, bake, 0, 0, ReasonZeroYield},
		{"unsited with no profile runs", south, bake, 0, 10_000, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sited := c.Recipes[tt.recipe].Sited
			e, has, ok := tab.effective(&sh, tt.region, tt.recipe, tt.season, sited)
			if !ok || e != tt.wantE {
				t.Errorf("effective = (%d, ok %v), want (%d, true)", e, ok, tt.wantE)
			}
			limit, reason := yieldRefusal(sited, has, e)
			if reason != tt.wantReason || (reason == "") != (limit != 0) {
				t.Errorf("yieldRefusal(sited %v, has %v, e %d) = (%d, %q), want reason %q", sited, has, e, limit, reason, tt.wantReason)
			}
		})
	}
}

// layoutConfig is a canonical configuration for the residue layout. Recipes
// sort to bake 0 (two outputs), grind 1 (one), mill 2 (two, one a discard),
// sow 3 (one).
func layoutConfig() *Config { return bakeryConfig() }

// TestResidueLayout verifies the residue layout of a role: one residue per
// output item of each recipe of the ascending list, discards included. A role
// with recipes of two and one outputs has offsets 0 and 2 and length 3; the
// recipe after a discard-bearing one starts after both outputs; a recipe off
// the list has no offset; an empty list has length 0.
//
// Requirement: FR-PRO-002
func TestResidueLayout(t *testing.T) {
	t.Parallel()
	rt := newRecipeTable(layoutConfig())
	tests := []struct {
		name    string
		recipes []RecipeID
		wantLen int
		offsets map[RecipeID]int // recipe -> offset; absent: not on the list
	}{
		// bake has 2 outputs, grind 1: offsets 0 and 2, length 2 + 1 = 3.
		{"two recipes", []RecipeID{0, 1}, 3, map[RecipeID]int{0: 0, 1: 2}},
		// grind 1, mill 2 (a discard and flour), sow 1: 0, 1, 3; length 4.
		{"discard counts", []RecipeID{1, 2, 3}, 4, map[RecipeID]int{1: 0, 2: 1, 3: 3}},
		{"empty role", nil, 0, map[RecipeID]int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := residueLen(tt.recipes, &rt); got != tt.wantLen {
				t.Errorf("residueLen(%v) = %d, want %d", tt.recipes, got, tt.wantLen)
			}
			for r := range RecipeID(4) {
				got, ok := residueOffset(tt.recipes, &rt, r)
				want, wantOK := tt.offsets[r]
				if got != want || ok != wantOK {
					t.Errorf("residueOffset(%v, %d) = (%d, %v), want (%d, %v)", tt.recipes, r, got, ok, want, wantOK)
				}
			}
		})
	}
}

// TestSetResidues verifies that setting every residue of a slice sets all of
// them and leaves an empty slice alone.
//
// Requirement: FR-PRO-026
func TestSetResidues(t *testing.T) {
	t.Parallel()
	got := []int64{1, 2, 3, 4}
	setResidues(got, 4_321)
	if want := []int64{4_321, 4_321, 4_321, 4_321}; !slices.Equal(got, want) {
		t.Errorf("setResidues(.., 4321) = %v, want %v", got, want)
	}
	setResidues(nil, 7) // must not panic
}

// TestJob_ReusesPending verifies the job record keeps its pending slice for
// the next start: resetting it to length 0 and appending as many quantities
// as before does not allocate.
//
// Requirement: NFR-PRO-001
func TestJob_ReusesPending(t *testing.T) {
	j := job{recipe: 2, remaining: 3, executions: 4, principal: 5, yield: 10_000, pending: make([]int64, 0, 3)}
	if j.recipe != 2 || j.remaining != 3 || j.executions != 4 || j.principal != 5 || j.yield != 10_000 {
		t.Fatalf("job fields = %+v, want recipe 2, remaining 3, executions 4, principal 5, yield 10000", j)
	}
	allocs := testing.AllocsPerRun(100, func() {
		j.pending = append(j.pending[:0], 5, 6, 7)
	})
	if allocs != 0 {
		t.Errorf("AllocsPerRun(reuse pending) = %v, want 0", allocs)
	}
}

// TestAdmit verifies the admission check: a recipe on the role's ascending
// list is admitted, one off it is not_on_role, and an empty list refuses
// everything.
//
// Requirement: FR-PRO-008, FR-PRO-048
func TestAdmit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		list   []RecipeID
		recipe RecipeID
		want   Reason
	}{
		{"on the list", []RecipeID{1, 3, 5}, 3, ""},
		{"first of the list", []RecipeID{1, 3, 5}, 1, ""},
		{"between entries", []RecipeID{1, 3, 5}, 4, ReasonNotOnRole},
		{"past the end", []RecipeID{1, 3, 5}, 9, ReasonNotOnRole},
		{"empty list", nil, 0, ReasonNotOnRole},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := admit(tt.list, tt.recipe); got != tt.want {
				t.Errorf("admit(%v, %d) = %q, want %q", tt.list, tt.recipe, got, tt.want)
			}
		})
	}
}

// TestProductionScratch_Sizing verifies that a scratch built from a
// configuration whose largest recipe has three inputs, two tools and two
// outputs has a buffer for each of those and for the number of recipes and of
// commodities, that two scratches share no backing array, and that refilling a
// buffer to its capacity with [:0] allocates nothing.
//
// Requirement: NFR-PRO-001
//
// Guard: architecture.md 2.9, the buffers are sized once and reused, so a step
// allocates nothing; a shared array between worlds would make them interfere.
func TestProductionScratch_Sizing(t *testing.T) {
	// not parallel: AllocsPerRun panics in a parallel test
	c := Config{
		Commodities: []Commodity{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}, {ID: "t1"}, {ID: "t2"}},
		Recipes: []Recipe{
			{ID: "big",
				Inputs:   []Item{{Commodity: "a", Quantity: 1}, {Commodity: "b", Quantity: 1}, {Commodity: "c", Quantity: 1}},
				Outputs:  []Output{{Commodity: "d", Quantity: 1, CostShare: 10_000}, {Commodity: "a", Quantity: 1}},
				Requires: []Tool{{Commodity: "t1", WearPerUse: 1}, {Commodity: "t2", WearPerUse: 1}}},
			{ID: "small", Outputs: []Output{{Commodity: "a", Quantity: 1, CostShare: 10_000}}},
		},
	}
	cc := c.canonical()
	rt := newRecipeTable(&cc)
	s1 := newProductionScratch(&cc, &rt)
	s2 := newProductionScratch(&cc, &rt)
	if s1.cfg != &cc || s1.recipes != &rt {
		t.Errorf("scratch references = (%p, %p), want the arguments (%p, %p)", s1.cfg, s1.recipes, &cc, &rt)
	}
	// big: 3 inputs, 2 outputs, 2 tools. The limit list is the yield pair, a
	// limit per tool and one per input: 1 + 2 + 3.
	tests := []struct {
		name     string
		got, min int
	}{
		{"terms", cap(s1.terms), 3},
		{"wears", cap(s1.wears), 2},
		{"input lines", cap(s1.inputs), 3},
		{"output lines", cap(s1.outputs), 2},
		{"limits", cap(s1.limits), 6},
		{"anchor recipes", cap(s1.anchor), 2},
		{"anchor row", len(s1.row), 6},
	}
	for _, tt := range tests {
		if tt.got < tt.min {
			t.Errorf("%s capacity = %d, want at least %d", tt.name, tt.got, tt.min)
		}
	}
	if len(s1.terms) != 0 || len(s1.wears) != 0 || len(s1.inputs) != 0 || len(s1.outputs) != 0 || len(s1.limits) != 0 || len(s1.anchor) != 0 {
		t.Errorf("buffers start with lengths (%d, %d, %d, %d, %d, %d), want all 0",
			len(s1.terms), len(s1.wears), len(s1.inputs), len(s1.outputs), len(s1.limits), len(s1.anchor))
	}
	if &s1.terms[:1][0] == &s2.terms[:1][0] || &s1.inputs[:1][0] == &s2.inputs[:1][0] ||
		&s1.limits[:1][0] == &s2.limits[:1][0] || &s1.row[0] == &s2.row[0] {
		t.Errorf("two scratches built from one config share a backing array")
	}
	allocs := testing.AllocsPerRun(100, func() {
		s1.terms = s1.terms[:0]
		for range cap(s1.terms) {
			s1.terms = append(s1.terms, production.Term{})
		}
		s1.limits = s1.limits[:0]
		for range cap(s1.limits) {
			s1.limits = append(s1.limits, production.Limit{})
		}
	})
	if allocs != 0 {
		t.Errorf("AllocsPerRun(refill to capacity) = %v, want 0", allocs)
	}
}

// TestProductionFile_Guards verifies, over the syntax tree of production.go,
// that it branches on no identifier of a commodity, recipe or role (a
// selector .ID is read only where the season index table is built, and no
// case clause or comparison names a string literal), declares no
// []production.Recipe struct field other than the scratch's one buffer and no
// package variable, and declares no function or type named for converting.
//
// Requirement: FR-PRO-001, FR-PRO-032
//
// Guard: architecture.md 9.1 and 2.9, a branch on an identifier is a defect
// and a persistent flat copy of the recipe table could drift from it.
func TestProductionFile_Guards(t *testing.T) {
	t.Parallel()
	f, err := parser.ParseFile(token.NewFileSet(), "production.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse production.go: %v", err)
	}
	var recipeBuffers []string
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			if d.Tok == token.VAR {
				t.Errorf("package variable at %v, want none (architecture.md 7.1)", d.Pos())
			}
			for _, sp := range d.Specs {
				ts, ok := sp.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if strings.Contains(strings.ToLower(ts.Name.Name), "convert") {
					t.Errorf("type %s is named for converting", ts.Name.Name)
				}
				if st, ok := ts.Type.(*ast.StructType); ok {
					for _, fld := range st.Fields.List {
						if at, ok := fld.Type.(*ast.ArrayType); ok {
							if sel, ok := at.Elt.(*ast.SelectorExpr); ok && sel.Sel.Name == "Recipe" {
								for _, n := range fld.Names {
									recipeBuffers = append(recipeBuffers, ts.Name.Name+"."+n.Name)
								}
							}
						}
					}
				}
			}
		case *ast.FuncDecl:
			if strings.Contains(strings.ToLower(d.Name.Name), "convert") {
				t.Errorf("function %s is named for converting", d.Name.Name)
			}
			idOK := d.Name.Name == "newYieldTable"
			ast.Inspect(d, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.SelectorExpr:
					if n.Sel.Name == "ID" && !idOK {
						t.Errorf("%s reads an ID at %v", d.Name.Name, n.Pos())
					}
				case *ast.CaseClause:
					for _, e := range n.List {
						if _, ok := e.(*ast.BasicLit); ok {
							t.Errorf("%s has a case on a literal at %v", d.Name.Name, e.Pos())
						}
					}
				case *ast.BinaryExpr:
					if n.Op == token.EQL || n.Op == token.NEQ {
						for _, e := range []ast.Expr{n.X, n.Y} {
							if bl, ok := e.(*ast.BasicLit); ok && bl.Kind == token.STRING {
								t.Errorf("%s compares with a string literal at %v", d.Name.Name, bl.Pos())
							}
						}
					}
				}
				return true
			})
		}
	}
	if want := []string{"productionScratch.anchor"}; !slices.Equal(recipeBuffers, want) {
		t.Errorf("[]production.Recipe fields = %v, want %v", recipeBuffers, want)
	}
}

// TestNewYieldTable_UnknownReferencePanics verifies that a profile naming a
// region, recipe or season the configuration lacks, which validation rejects,
// panics with a message that says it is a violated invariant.
//
// Requirement: FR-PRO-038
func TestNewYieldTable_UnknownReferencePanics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		y    YieldProfile
	}{
		{"region", YieldProfile{Region: "nowhere", Recipe: "x", HasAll: true, All: 1}},
		{"recipe", YieldProfile{Region: "r", Recipe: "nothing", HasAll: true, All: 1}},
		{"season", YieldProfile{Region: "r", Recipe: "x", Seasons: []SeasonYield{{Season: "never", BP: 1}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := Config{
				Calendar: Calendar{Seasons: []Season{{ID: "spring"}}},
				Regions:  []Region{{ID: "r"}},
				Recipes:  []Recipe{{ID: "x"}},
				Yields:   []YieldProfile{tt.y},
			}
			defer func() {
				msg, _ := recover().(string)
				if !strings.Contains(msg, "invariant") {
					t.Errorf("newYieldTable panic = %q, want a message with \"invariant\"", msg)
				}
			}()
			newYieldTable(&c)
		})
	}
}
