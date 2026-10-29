package economysim

import (
	"slices"
	"testing"
)

// subConfig is roleConfig with two sited source recipes, dig (ore, duration 1)
// and farm (staple, duration 40), the subsistence module on with the default list
// [farm, dig] and a canonical copy. Recipes sort to dig 0, farm 1, smelt 2; roles
// to idle 0, smith 1.
func subConfig(rows ...RoleOwnUse) *Config {
	c := *roleConfig()
	c.Recipes = append(slices.Clone(c.Recipes),
		Recipe{ID: "dig", Duration: 1, Sited: true, Outputs: []Output{{Commodity: "ore", Quantity: 1, CostShare: 10000}}},
		Recipe{ID: "farm", Duration: 40, Sited: true, Outputs: []Output{{Commodity: "staple", Quantity: 1, CostShare: 10000}}})
	m := DefaultSubsistence()
	m.Recipes, m.Roles = []string{"farm", "dig"}, rows
	c.Modules.Subsistence = &m
	cc := c.canonical()
	return &cc
}

// TestEffectiveList_RowReplacesDefault verifies that a role's row replaces the
// module's default list for that role alone, and the others keep the default.
//
// Requirement: FR-SUB-002
func TestEffectiveList_RowReplacesDefault(t *testing.T) {
	t.Parallel()
	m := subConfig(RoleOwnUse{Role: "smith", Recipes: []string{"farm"}}).Modules.Subsistence
	if got, want := m.ownUseList("smith"), []string{"farm"}; !slices.Equal(got, want) {
		t.Errorf("ownUseList(smith) = %v, want %v: the row replaces the default", got, want)
	}
	if got, want := m.ownUseList("idle"), []string{"dig", "farm"}; !slices.Equal(got, want) {
		t.Errorf("ownUseList(idle) = %v, want %v: the default", got, want)
	}
}

// TestEffectiveList_EmptyRowMeansNone verifies that an empty row says the role
// has no own-use recipe, which is not the same as no row.
//
// Requirement: FR-SUB-002
func TestEffectiveList_EmptyRowMeansNone(t *testing.T) {
	t.Parallel()
	c := subConfig(RoleOwnUse{Role: "smith"})
	if got := c.Modules.Subsistence.ownUseList("smith"); len(got) != 0 {
		t.Errorf("ownUseList(smith) = %v, want none", got)
	}
	tab := newRoleTable(c)
	if row := tab.row(1); row.ownUse != nil || !slices.Equal(row.jobs, row.recipes) {
		t.Errorf("smith ownUse = %v, jobs = %v, want none and its recipes %v", row.ownUse, row.jobs, row.recipes)
	}
	if got := ((*SubsistenceConfig)(nil)).ownUseList("smith"); got != nil {
		t.Errorf("ownUseList on the module off = %v, want nil", got)
	}
}

// TestEffectiveList_SortedAndBounded verifies that the effective list is sorted
// by recipe ID whatever the order given, is a copy the caller may change, and
// that the role table holds it as ascending recipe IDs with the union of the
// role's recipes in jobs.
//
// Requirement: FR-SUB-002
func TestEffectiveList_SortedAndBounded(t *testing.T) {
	t.Parallel()
	raw := &SubsistenceConfig{Recipes: []string{"z", "a", "m"}, Roles: []RoleOwnUse{{Role: "r", Recipes: []string{"y", "b"}}}}
	if got, want := raw.ownUseList("x"), []string{"a", "m", "z"}; !slices.Equal(got, want) {
		t.Errorf("ownUseList(default) = %v, want %v", got, want)
	}
	got := raw.ownUseList("r")
	got[0] = "changed"
	if raw.Roles[0].Recipes[1] != "b" || raw.ownUseList("r")[0] != "b" {
		t.Errorf("ownUseList returned the module's own slice: a change of the result reached the row")
	}
	c := subConfig()
	tab := newRoleTable(c)
	smith, idle := tab.row(1), tab.row(0)
	if want := []RecipeID{0, 1}; !slices.Equal(smith.ownUse, want) {
		t.Errorf("smith ownUse = %v, want %v (dig, farm)", smith.ownUse, want)
	}
	if want := []RecipeID{0, 1, 2}; !slices.Equal(smith.jobs, want) {
		t.Errorf("smith jobs = %v, want %v (own-use recipes and smelt, ascending)", smith.jobs, want)
	}
	if want := []RecipeID{2}; !slices.Equal(smith.recipes, want) {
		t.Errorf("smith recipes = %v, want %v: the view reads the role's own list", smith.recipes, want)
	}
	if !slices.Equal(idle.jobs, idle.ownUse) || len(idle.recipes) != 0 {
		t.Errorf("idle jobs = %v, ownUse = %v, recipes = %v, want the own-use list alone", idle.jobs, idle.ownUse, idle.recipes)
	}
}

// TestRoleTable_ModuleOffJobsAreRecipes verifies that with the module off the
// role table holds no own-use list and jobs is the role's recipes themselves, so
// the module off changes no slice.
//
// Requirement: FR-SUB-001
func TestRoleTable_ModuleOffJobsAreRecipes(t *testing.T) {
	t.Parallel()
	tab := newRoleTable(roleConfig())
	for id := range tab.rows {
		row := tab.row(RoleID(id))
		if row.ownUse != nil || !slices.Equal(row.jobs, row.recipes) || (len(row.recipes) > 0 && &row.jobs[0] != &row.recipes[0]) {
			t.Errorf("role %d: ownUse = %v, jobs = %v, recipes = %v, want no list and jobs the recipes", id, row.ownUse, row.jobs, row.recipes)
		}
	}
}

// TestRoleTable_OwnUseOutputsAreTradable verifies that the non-discard outputs of
// the effective own-use list join the tradable set of the role, so a household may
// hold and sell what it makes, and do not with the module off.
//
// Requirement: FR-SUB-010, FR-AGT-003
func TestRoleTable_OwnUseOutputsAreTradable(t *testing.T) {
	t.Parallel()
	c := subConfig()
	on, off := newRoleTable(c), newRoleTable(roleConfig())
	// idle has the world's upkeep (staple, fuel); dig makes ore.
	ore, ok := c.commodityID("ore")
	if !ok {
		t.Fatalf("commodityID(ore) not found")
	}
	if !on.row(0).isTradable(ore) {
		t.Errorf("idle isTradable(ore) with dig on its own-use list = false, want true")
	}
	if off.row(0).isTradable(ore) {
		t.Errorf("idle isTradable(ore) with the module off = true, want false")
	}
}

// TestAdmit_OwnUseBeforeAnyOtherCheck verifies that production admits a recipe on
// the role's effective own-use list through jobs, and still refuses a recipe on
// neither list with not_on_role, first of all the checks.
//
// Requirement: FR-SUB-026
func TestAdmit_OwnUseBeforeAnyOtherCheck(t *testing.T) {
	t.Parallel()
	tab := newRoleTable(subConfig())
	smith := tab.row(1)
	for _, k := range []RecipeID{0, 1, 2} {
		if r := admit(smith.jobs, k); r != "" {
			t.Errorf("admit(smith jobs, %d) = %q, want admitted", k, r)
		}
	}
	if r := admit(smith.recipes, 0); r != ReasonNotOnRole {
		t.Errorf("admit(smith recipes, own-use recipe) = %q, want %q: the role's own list does not hold it", r, ReasonNotOnRole)
	}
	if r := admit(smith.jobs, 3); r != ReasonNotOnRole {
		t.Errorf("admit(smith jobs, a recipe on neither list) = %q, want %q", r, ReasonNotOnRole)
	}
}

// TestWindow_JoinsOwnUse verifies that the welfare window of a role is at least
// twice the longest Duration on its list or its effective own-use list: idle has
// no recipe and the world's window is 30, and farm lasts 40, so the window is
// 2 * 40 = 80 with the module on and 30 with it off.
//
// Requirement: FR-WEL-009, FR-SUB-029
func TestWindow_JoinsOwnUse(t *testing.T) {
	t.Parallel()
	on, off := newRoleTable(subConfig()), newRoleTable(roleConfig())
	if got := on.row(0).window; got != 80 {
		t.Errorf("window of idle with farm (duration 40) on its own-use list = %d, want 80", got)
	}
	if got := off.row(0).window; got != 30 {
		t.Errorf("window of idle with the module off = %d, want 30", got)
	}
}

// TestDefaultSubsistence_Values verifies the placeholder defaults of
// spec-subsistence.md 12: 2, 3 and 10 and no list.
//
// Requirement: FR-SUB-007
func TestDefaultSubsistence_Values(t *testing.T) {
	t.Parallel()
	got := DefaultSubsistence()
	want := SubsistenceConfig{EnterTicks: 2, ExitTicks: 3, RetryJobs: 10}
	if got.EnterTicks != want.EnterTicks || got.ExitTicks != want.ExitTicks || got.RetryJobs != want.RetryJobs || got.Recipes != nil || got.Roles != nil {
		t.Errorf("DefaultSubsistence() = %+v, want %+v", got, want)
	}
}

// TestCanonical_SubsistenceOrder verifies the canonical order of the module: the
// default list and every row's list by recipe ID, the rows by role, and the
// receiver unchanged.
//
// Requirement: FR-SUB-002, FR-CFG-077
func TestCanonical_SubsistenceOrder(t *testing.T) {
	t.Parallel()
	c := oneRegion()
	c.Modules.Subsistence = &SubsistenceConfig{
		Recipes: []string{"c", "a"},
		Roles:   []RoleOwnUse{{Role: "q", Recipes: []string{"y", "x"}}, {Role: "p"}},
	}
	got := c.canonical().Modules.Subsistence
	if !slices.Equal(got.Recipes, []string{"a", "c"}) || got.Roles[0].Role != "p" || !slices.Equal(got.Roles[1].Recipes, []string{"x", "y"}) {
		t.Errorf("canonical subsistence = %+v, want sorted lists and rows", got)
	}
	if c.Modules.Subsistence.Recipes[0] != "c" || c.Modules.Subsistence.Roles[0].Role != "q" {
		t.Errorf("canonical changed the receiver: %+v", c.Modules.Subsistence)
	}
	if empty := (Config{Modules: Modules{Subsistence: &SubsistenceConfig{}}}).canonical(); empty.Modules.Subsistence == nil {
		t.Errorf("canonical of an empty module that is on = nil, want on")
	}
}
