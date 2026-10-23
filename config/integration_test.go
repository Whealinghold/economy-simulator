package config_test

import (
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// chains is the directory of the reference packs, relative to this package.
const chains = "../testdata/chains"

// TestPacks_LoadValidateAndHash verifies that the packs of testdata/chains load
// from their layers, give the same configuration and the same hash on every
// load, and pass Validate.
//
// Both packs must validate with no error (FR-REF-005, FR-REF-017).
//
// Requirement: FR-CFG-023, FR-CFG-025, FR-REF-005, FR-REF-017, FR-REF-019
//
// Guard: the pinned hashes are the earliest signal that a pack's field set or
// the hash encoding moved (compare pinnedReferenceHash in hash_test.go). A change
// to them must be on purpose and said so; stability within a run would not show
// a field that was added, dropped or re-encoded.
func TestPacks_LoadValidateAndHash(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	fsys := os.DirFS(chains)
	tests := []struct {
		name   string
		layers []string
		roles  int
		hash   string
	}{
		{"pack one", []string{"base.toml", "one.toml", "warm-one.toml"}, 7, "9abf7b8fa64bc36b9f223f7d195b81e7934fa9d88c297085ca4128d0a98275a0"},
		{"pack second", []string{"second/second.toml", "second/warm-second.toml"}, 5, "388a7d7e47361f9948027dd52d037baebd484fa4a99cdf26a07a010581d5ff8a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := config.LoadFS(fsys, tt.layers...)
			if err != nil {
				t.Fatalf("LoadFS(%q) error = %v, want nil", tt.layers, err)
			}
			if len(cfg.Roles) != tt.roles || cfg.WarmStart == nil || len(cfg.Regions) != 1 {
				t.Errorf("LoadFS(%q) = %d roles, warm start %v, %d regions, want %d roles, a warm start and 1 region",
					tt.layers, len(cfg.Roles), cfg.WarmStart != nil, len(cfg.Regions), tt.roles)
			}
			if err := cfg.Validate(); err != nil {
				t.Errorf("%s: Validate() = %v, want nil", tt.name, err)
			}
			h := economysim.HashConfig(cfg)
			if got := hex.EncodeToString(h[:]); got != tt.hash {
				t.Errorf("HashConfig(%q) = %s, want %s", tt.layers, got, tt.hash)
			}
			again, err := config.LoadFS(fsys, tt.layers...)
			if err != nil {
				t.Fatalf("LoadFS(%q) second time error = %v, want nil", tt.layers, err)
			}
			if !reflect.DeepEqual(cfg, again) || economysim.HashConfig(cfg) != economysim.HashConfig(again) {
				t.Errorf("LoadFS(%q) twice: configurations or hashes differ", tt.layers)
			}
		})
	}
}

// TestPacks_WarmLayerChangesTheHash verifies that the warm layer is part of the
// pack: the cold pack and the warm pack have different hashes, and the cold
// pack has no warm start.
//
// Requirement: FR-CFG-025
func TestPacks_WarmLayerChangesTheHash(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	fsys := os.DirFS(chains)
	cold, err1 := config.LoadFS(fsys, "base.toml", "one.toml")
	warm, err2 := config.LoadFS(fsys, "base.toml", "one.toml", "warm-one.toml")
	if err1 != nil || err2 != nil {
		t.Fatalf("LoadFS() errors = %v, %v, want nil", err1, err2)
	}
	if cold.WarmStart != nil {
		t.Errorf("cold pack WarmStart = %+v, want nil", cold.WarmStart)
	}
	if economysim.HashConfig(cold) == economysim.HashConfig(warm) {
		t.Errorf("HashConfig(cold) == HashConfig(warm), want different")
	}
}

// TestPacks_DefectOverlays verifies that the six defect overlays of
// testdata/chains/defects load over the base layers of the reference chain, and
// that each leaves the change it names in the configuration. The failures of d1
// to d5 are viability rules (spec-reference-chain.md 5): Validate rejects each with
// ErrNotViable (phase 6, Stage 1 row 10); d6 gives the warning
// cost_floor_at_full_cost, asserted here. Each defect is
// loaded over both layer sets the reference chain names for pack one
// (spec-reference-chain.md 4.5, 5, 7.6 A2): base and one, and base, one and
// warm-one, the defect last.
//
// Each of d1 to d5 is checked for the set of rules it reports (a joined error
// is walked, not read through errors.As, which finds only the first): d3, d4 and
// d5 that rule only, d1 and d2 with the rules REF 5 lists as also reported.
//
// Requirement: FR-CFG-023, FR-CFG-007, FR-REF-016, FR-REF-017, FR-CFG-041, FR-VIA-004, FR-VIA-005, FR-VIA-006, FR-VIA-007, FR-VIA-008
func TestPacks_DefectOverlays(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	fsys := os.DirFS(chains)
	recipeOutput := func(cfg economysim.Config, id string) economysim.Quantity {
		for _, r := range cfg.Recipes {
			if r.ID == id {
				return r.Outputs[0].Quantity
			}
		}
		return -1
	}
	tests := []struct {
		name  string
		check func(economysim.Config) bool
		what  string
		// rules is the set of rules Validate reports (nil for d6); single says the
		// list has one error and no more (FR-REF-016: "that rule only").
		rules  []string
		single bool
		// unproducible is the Value of the unproducible_in_realm errors, sorted.
		unproducible []string
	}{
		{"d1_seed_cycle", func(c economysim.Config) bool {
			i := slices.IndexFunc(c.Yields, func(y economysim.YieldProfile) bool { return y.Recipe == "gather_seed" })
			return recipeOutput(c, "grow_staple") == 30 && c.Yields[i].HasAll && c.Yields[i].All == 0
		}, "grow_staple returns 30 and gather_seed has a yield of 0",
			[]string{"cycle_not_productive", "unproducible_in_realm"}, false, []string{"prepared", "seed", "staple"}},
		{"d2_upkeep_unmade", func(c economysim.Config) bool {
			i := slices.IndexFunc(c.Yields, func(y economysim.YieldProfile) bool { return y.Recipe == "cut_fuel" })
			return c.Yields[i].HasAll && c.Yields[i].All == 0
		}, "cut_fuel has a yield of 0",
			[]string{"role_cannot_work", "unproducible_in_realm"}, false, []string{"fuel", "prepared", "refined", "staple", "tool_a", "tool_b"}},
		{"d3_idle_role", func(c economysim.Config) bool {
			i := slices.IndexFunc(c.Groups, func(g economysim.Group) bool { return g.Role == "idle" })
			return i >= 0 && c.Groups[i].Households == 10 && len(c.Roles) == 8
		}, "a role idle of 10 households", []string{"role_cannot_work"}, true, nil},
		{"d4_long_job", func(c economysim.Config) bool { return recipeOutput(c, "grow_staple") == 90 }, "grow_staple returns 90",
			[]string{"labour_short"}, true, nil},
		{"d5_storage_gap", func(c economysim.Config) bool {
			i := slices.IndexFunc(c.Commodities, func(m economysim.Commodity) bool { return m.ID == "staple" })
			return c.Commodities[i].ShelfLife == 200
		}, "staple lasts 200 ticks", []string{"storage_gap"}, true, nil},
		{"d6_floor_full", func(c economysim.Config) bool { return c.World.CostFloor == 10_000 }, "cost_floor is 10000", nil, false, nil},
	}
	bases := []struct {
		name   string
		layers []string
	}{
		{"cold", []string{"base.toml", "one.toml"}},
		{"warm", []string{"base.toml", "one.toml", "warm-one.toml"}},
	}
	for _, tt := range tests {
		for _, b := range bases {
			t.Run(tt.name+"/"+b.name, func(t *testing.T) {
				t.Parallel()
				layers := append(slices.Clone(b.layers), "defects/"+tt.name+".toml")
				cfg, err := config.LoadFS(fsys, layers...)
				if err != nil {
					t.Fatalf("LoadFS(%q) error = %v, want nil", layers, err)
				}
				if !tt.check(cfg) {
					t.Errorf("LoadFS(%q) does not show: %s", layers, tt.what)
				}
				if warm := cfg.WarmStart != nil; warm != (b.name == "warm") {
					t.Errorf("LoadFS(%q) has a warm start = %t, want %t", layers, warm, b.name == "warm")
				}
				// d1 to d5 are viability rules (D4, phase 6 of Validate): Validate
				// returns a joined error, so the rules are collected from every
				// *ViabilityError of the list and compared as a set.
				err = cfg.Validate()
				if want := tt.rules != nil; errors.Is(err, economysim.ErrNotViable) != want {
					t.Errorf("Validate(%q) = %v, want ErrNotViable = %t", layers, err, want)
				}
				if errors.Is(err, economysim.ErrTooManyErrors) {
					t.Errorf("Validate(%q) hit ErrTooManyErrors", layers)
				}
				if got := ruleSet(err); !slices.Equal(got, tt.rules) {
					t.Errorf("Validate(%q) rules = %q, want %q", layers, got, tt.rules)
				}
				if got := len(viabilityErrors(err)); tt.single && got != 1 {
					t.Errorf("Validate(%q) has %d viability errors, want 1 (that rule only)", layers, got)
				}
				if got := valuesOf(err, "unproducible_in_realm"); !slices.Equal(got, tt.unproducible) {
					t.Errorf("Validate(%q) unproducible_in_realm values = %q, want %q", layers, got, tt.unproducible)
				}
				if tt.name == "d2_upkeep_unmade" {
					for _, ve := range viabilityErrors(err) {
						switch ve.Rule {
						case "role_cannot_work":
							if !slices.Equal(ve.Path, []string{"roles", "forester"}) || ve.Value != "north" || ve.Bound != "" {
								t.Errorf("role_cannot_work = path %q value %q bound %q, want roles/forester, north, empty", ve.Path, ve.Value, ve.Bound)
							}
						case "unproducible_in_realm":
							if len(ve.Path) != 2 || ve.Path[0] != "commodities" || ve.Path[1] != ve.Value {
								t.Errorf("unproducible_in_realm path %q value %q, want commodities/<value>", ve.Path, ve.Value)
							}
						}
					}
				}
				if tt.name == "d6_floor_full" && err != nil {
					t.Errorf("Validate(%q) = %v, want nil: the floor at the full cost is a warning", layers, err)
				}
				// d6 is the one defect that needs no solver: its warning is a Stage 0
				// rule, so it is asserted here (FR-REF-016).
				hasWarning := slices.ContainsFunc(cfg.Warnings(), func(w economysim.Warning) bool {
					return w.Rule == "cost_floor_at_full_cost"
				})
				if want := tt.name == "d6_floor_full"; hasWarning != want {
					t.Errorf("Warnings(%q) has cost_floor_at_full_cost = %t, want %t", layers, hasWarning, want)
				}
			})
		}
	}
}

// ruleSet returns the sorted set of the rules of every *ViabilityError of err.
func ruleSet(err error) []string {
	var out []string
	for _, ve := range viabilityErrors(err) {
		out = append(out, ve.Rule)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// valuesOf returns the sorted Value of every *ViabilityError of the rule.
func valuesOf(err error, rule string) []string {
	var out []string
	for _, ve := range viabilityErrors(err) {
		if ve.Rule == rule {
			out = append(out, ve.Value)
		}
	}
	slices.Sort(out)
	return out
}

// TestPacks_DefectOverlays_WildSource verifies the d1 defect with the wild source
// of seed on: grow_staple returns 30 but gather_seed keeps its yield of 10 000,
// so the seed cycle has a source and cycle_not_productive is absent, and the
// pack fails on the labour share alone (labour_short, one error, at path
// regions, the pooled surplus as text against the bound 10 000). The pack is
// built with config.Parse over the bytes of the layers and an in-memory document
// that holds only d1's recipe (the yield row is not dropped: a sited recipe with
// no yield row keeps the source off).
//
// Requirement: FR-REF-016, FR-REF-017, FR-VIA-005
//
// Guard: a cycle rule that ignored the wild source would reject this pack with
// cycle_not_productive.
func TestPacks_DefectOverlays_WildSource(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	d1 := []byte("[recipe.grow_staple]\noutputs = [ { commodity = \"staple\", quantity = 30, cost_share = 10000 } ]\n")
	for _, tt := range []struct {
		name   string
		layers []string
	}{
		{"cold", []string{"base.toml", "one.toml"}},
		{"warm", []string{"base.toml", "one.toml", "warm-one.toml"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var docs [][]byte
			for _, l := range tt.layers {
				b, err := os.ReadFile(chains + "/" + l)
				if err != nil {
					t.Fatal(err)
				}
				docs = append(docs, b)
			}
			cfg, err := config.Parse(append(docs, d1)...)
			if err != nil {
				t.Fatalf("Parse(%q + d1 recipe) error = %v, want nil", tt.layers, err)
			}
			err = cfg.Validate()
			ves := viabilityErrors(err)
			if got := ruleSet(err); !slices.Equal(got, []string{"labour_short"}) || len(ves) != 1 {
				t.Fatalf("Validate(%q + d1 recipe) rules = %q in %d errors (%v), want labour_short only", tt.layers, got, len(ves), err)
			}
			ve := ves[0]
			surplus, perr := strconv.ParseInt(ve.Value, 10, 64)
			if perr != nil || !slices.Equal(ve.Path, []string{"regions"}) || ve.Bound != "10000" || surplus < 700 || surplus > 760 {
				t.Errorf("labour_short = path %q value %q bound %q, want regions, about 730, 10000", ve.Path, ve.Value, ve.Bound)
			}
		})
	}
}

// TestPacks_ModuleOverlaysAreRejectedUntilBuilt verifies that the fixtures that
// switch a module on are rejected at Stage 0 with the module table as the
// unknown key, because no module is built.
//
// Requirement: FR-CFG-019, FR-CFG-098
func TestPacks_ModuleOverlaysAreRejectedUntilBuilt(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	fsys := os.DirFS(chains)
	tests := []struct {
		name   string
		layers []string
		path   []string
	}{
		{"estates overlay", []string{"base.toml", "one.toml", "overlay/estates.toml"}, []string{"modules", "estates"}},
		{"demand pack", []string{"base.toml", "one.toml", "demand.toml"}, []string{"modules", "demand"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := config.LoadFS(fsys, tt.layers...)
			if !hasError(err, economysim.ErrUnknownKey, tt.path) {
				t.Errorf("LoadFS(%q) errors = %s, want ErrUnknownKey at %q", tt.layers, describe(err), tt.path)
			}
		})
	}
}
