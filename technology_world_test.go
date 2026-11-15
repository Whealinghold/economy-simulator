package economysim_test

import (
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The world-level tests of the technology module's first slice (plan 3-tec, PR A):
// the module can be configured, validated and hashed, and does nothing at run time.
// They load the overlays through the loader, build real worlds and run them. What a
// gain does to production is a test of the next slice, and nothing here depends on it.

var (
	technologyOne   = []string{"base.toml", "one.toml", "warm-one.toml", "overlay/technology.toml"}
	technologyThree = []string{"base.toml", "three.toml", "warm-three.toml", "overlay/trade-three.toml", "overlay/technology-four.toml"}
)

// TestTechnology_OverlaysBuildWorlds verifies that each technology overlay, loaded on
// its pack through the loader, validates and builds a world with the audit on, that
// the world raises none of the module's five warnings and that the injected logger
// receives none of them, so the overlays that carry the module's golden are clean
// test doubles.
//
// Requirement: FR-TEC-002, FR-TEC-029
func TestTechnology_OverlaysBuildWorlds(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for name, layers := range map[string][]string{"learning on one": technologyOne, "all four on three": technologyThree} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, layers...)
			if cfg.Modules.Technology == nil || len(cfg.Modules.Technology.Fields) != 1 {
				t.Fatalf("overlay loaded %+v, want one field", cfg.Modules.Technology)
			}
			h := newCaptureHandler()
			w, err := economysim.NewWorld(cfg, economysim.WithAudit(), economysim.WithLogger(slog.New(h)))
			if err != nil {
				t.Fatalf("NewWorld() error = %v, want nil", err)
			}
			for _, wn := range w.Warnings() {
				if isTechnologyRule(wn.Rule) {
					t.Errorf("Warnings() has %v, want none of the module's", wn)
				}
			}
			if got := technologyLoggedWarnings(t, h); len(got) != 0 {
				t.Errorf("the log has the technology warnings %v, want none", got)
			}
		})
	}
}

// TestTechnology_AllFiveWarningsAreLogged verifies, on a world that builds, that each
// of the five warnings of spec-technology.md 10.2 to 10.4 reaches the injected
// logger once, at Warn, with the rule and the dotted path of its field, and that the
// set equals the one World.Warnings returns: diffusion on in a world of one region,
// a trend past the schedule and a cap past the maximum yield of a logistic stock, a
// field with every mechanism off and a field whose recipe no role lists.
//
// Requirement: FR-TEC-029, FR-TEC-030, FR-TEC-031
func TestTechnology_AllFiveWarningsAreLogged(t *testing.T) {
	t.Parallel()
	c := resourcesConfig()
	withPopulation(&c)
	stand := resourceOf(&c, "stand")
	stand.PlanShareBP = 6_000
	stand.Abundance = []economysim.AbundanceKnot{{Share: 0, Factor: 0}, {Share: 10_000, Factor: 10_000}}
	idle := economysim.DefaultRecipe()
	idle.ID = "idle"
	idle.Outputs = slices.Clone(recipeOf(&c, "cut_fuel").Outputs)
	c.Recipes = append(c.Recipes, idle)
	c.Modules.Technology = &economysim.Technology{Fields: []economysim.TechField{
		{ID: "farming", Recipes: []string{"gather_seed", "grow_staple"}, CapBP: 5, CatchUpBP: 500, Reach: 100},
		{ID: "forestry", Recipes: []string{"cut_fuel"}, CapBP: 2_000, TrendBP: 201},
		{ID: "spare", Recipes: []string{"idle"}, CapBP: 5},
	}}
	want := []string{
		"diffusion_no_contact@modules.technology.fields[0]",
		"field_inert@modules.technology.fields[2]",
		"field_unused@modules.technology.fields[2]",
		"technology_past_maximum_yield@modules.technology.fields[1]",
		"trend_outpaces_schedule@modules.technology.fields[1]",
	}
	h := newCaptureHandler()
	w, err := economysim.NewWorld(c, economysim.WithLogger(slog.New(h)))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil (a warning never fails construction)", err)
	}
	var fromWorld []string
	for _, wn := range w.Warnings() {
		if isTechnologyRule(wn.Rule) {
			fromWorld = append(fromWorld, wn.Rule+"@"+strings.Join(wn.Path, "."))
		}
	}
	logged := technologyLoggedWarnings(t, h)
	slices.Sort(fromWorld)
	slices.Sort(logged)
	if !slices.Equal(fromWorld, want) {
		t.Errorf("Warnings() = %v, want %v", fromWorld, want)
	}
	if !slices.Equal(logged, want) {
		t.Errorf("logged warnings = %v, want %v", logged, want)
	}
}

// TestTechnology_UnvalidatedConfigWarnsWithoutPanic verifies that Warnings is safe on a
// configuration that does not validate: a field that names a recipe nobody defined, a
// field with no recipe and a duplicate field raise no warning of the stock or the
// schedule, and do not panic, so a consumer can ask for warnings before it fixes the
// errors.
//
// Guard: the warnings read tables by ID; without the skip of an undeclared ID a
// config with a typo would panic in the one call that reports the typo.
func TestTechnology_UnvalidatedConfigWarnsWithoutPanic(t *testing.T) {
	t.Parallel()
	c := resourcesConfig()
	withPopulation(&c)
	c.Modules.Technology = &economysim.Technology{Fields: []economysim.TechField{
		{ID: "typo", Recipes: []string{"nowhere"}, CapBP: 2_000, TrendBP: 9_999},
		{ID: "empty", CapBP: 2_000, TrendBP: 9_999},
		{ID: "typo", Recipes: []string{"cut_fuel"}, CapBP: 2_000},
	}}
	if err := c.Validate(); err == nil {
		t.Fatalf("Validate() = nil, want the errors of the three fields")
	}
	for _, wn := range c.Warnings() {
		if wn.Rule == "technology_past_maximum_yield" && len(wn.Path) > 2 && wn.Path[2] == "fields[0]" {
			t.Errorf("Warnings() has %v for a field whose recipe is not defined, want none", wn)
		}
	}
}

// TestTechnology_InertModuleRunsAsOff verifies, for seeds 1 to 3 over 120 ticks of the
// pack one with its warm start, that a world with the module on but doing nothing (no
// field, and one field with every mechanism off) has the same events after
// WorldCreated and the same final snapshot, apart from the configuration hash and the
// list of modules, as the world with the module off, and that the module changes the
// configuration hash.
//
// Requirement: FR-TEC-001, NFR-TEC-001
func TestTechnology_InertModuleRunsAsOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	layers := []string{"base.toml", "one.toml", "warm-one.toml"}
	off := loadValidPack(t, layers...)
	empty := off
	empty.Modules.Technology = &economysim.Technology{}
	inert := off
	inert.Modules.Technology = &economysim.Technology{Fields: []economysim.TechField{{ID: "farming", Recipes: []string{"grow_staple"}, CapBP: 3_000}}}
	for _, tt := range []struct {
		name string
		cfg  economysim.Config
	}{{"no field", empty}, {"a field with every mechanism off", inert}} {
		if economysim.HashConfig(tt.cfg) == economysim.HashConfig(off) {
			t.Errorf("%s: the hash equals the module off, want the module to be in it", tt.name)
		}
		for seed := uint64(1); seed <= 3; seed++ {
			spec := runSpec{Seed: seed, Ticks: 120, Observe: true, Audit: true}
			a, b := runPack(t, off, spec), runPack(t, tt.cfg, spec)
			if !reflect.DeepEqual(a.Events[1:], b.Events[1:]) {
				t.Errorf("%s, seed %d: the events after WorldCreated differ from the module off", tt.name, seed)
			}
			fa, fb := a.Final, b.Final
			fa.ConfigHash, fb.ConfigHash, fa.Modules, fb.Modules = [32]byte{}, [32]byte{}, nil, nil
			fa.Names, fb.Names = economysim.Names{}, economysim.Names{} // Names carries TechFields (FR-TEC-001)
			if !reflect.DeepEqual(fa, fb) {
				t.Errorf("%s, seed %d: the final snapshots differ from the module off beyond the hash and the module list", tt.name, seed)
			}
		}
	}
}

// TestTechnology_OverlayWorldsAreDeterministic verifies, for two seeds over 60 ticks,
// that each technology overlay on its pack gives the same events and the same final
// snapshot on two runs, and the same final snapshot whether the world is stepped
// tick by tick or run in two calls of 30, with the audit on and an observer attached.
//
// Requirement: NFR-TEC-001
func TestTechnology_OverlayWorldsAreDeterministic(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for name, layers := range map[string][]string{"learning on one": technologyOne, "all four on three": technologyThree} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, layers...)
			for seed := uint64(1); seed <= 2; seed++ {
				spec := runSpec{Seed: seed, Ticks: 60, Observe: true, Audit: true}
				a, b := runPack(t, cfg, spec), runPack(t, cfg, spec)
				if !reflect.DeepEqual(a.Events, b.Events) || !reflect.DeepEqual(a.Final, b.Final) {
					t.Errorf("seed %d: two runs of the same world differ", seed)
				}
				spec.Drive = driveRun(30, 30)
				if c := runPack(t, cfg, spec); !reflect.DeepEqual(a.Final, c.Final) {
					t.Errorf("seed %d: Run(30) twice gives a different final snapshot than 60 steps", seed)
				}
			}
		})
	}
}
