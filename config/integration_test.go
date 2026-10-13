package config_test

import (
	"encoding/hex"
	"os"
	"reflect"
	"slices"
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
		{"pack one", []string{"base.toml", "one.toml", "warm-one.toml"}, 7, "9e3c1357520b0e052fee1e1d773d45e49ce992a8d1cfbd7029a6a24a91038285"},
		{"pack second", []string{"second/second.toml", "second/warm-second.toml"}, 5, "3d16f1470bd2f00386962782d402fcd36258b8397b41031d6b7ab124f5b453d0"},
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
// to d5 are viability rules (spec-reference-chain.md 5) and belong to Stage 1;
// d6 gives the warning cost_floor_at_full_cost, asserted here. Each defect is
// loaded over both layer sets the reference chain names for pack one
// (spec-reference-chain.md 4.5, 5, 7.6 A2): base and one, and base, one and
// warm-one, the defect last.
//
// Requirement: FR-CFG-023, FR-CFG-007, FR-REF-016
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
	}{
		{"d1_seed_cycle", func(c economysim.Config) bool {
			i := slices.IndexFunc(c.Yields, func(y economysim.YieldProfile) bool { return y.Recipe == "gather_seed" })
			return recipeOutput(c, "grow_staple") == 30 && c.Yields[i].HasAll && c.Yields[i].All == 0
		}, "grow_staple returns 30 and gather_seed has a yield of 0"},
		{"d2_upkeep_unmade", func(c economysim.Config) bool {
			i := slices.IndexFunc(c.Yields, func(y economysim.YieldProfile) bool { return y.Recipe == "cut_fuel" })
			return c.Yields[i].HasAll && c.Yields[i].All == 0
		}, "cut_fuel has a yield of 0"},
		{"d3_idle_role", func(c economysim.Config) bool {
			i := slices.IndexFunc(c.Groups, func(g economysim.Group) bool { return g.Role == "idle" })
			return i >= 0 && c.Groups[i].Households == 10 && len(c.Roles) == 8
		}, "a role idle of 10 households"},
		{"d4_long_job", func(c economysim.Config) bool { return recipeOutput(c, "grow_staple") == 90 }, "grow_staple returns 90"},
		{"d5_storage_gap", func(c economysim.Config) bool {
			i := slices.IndexFunc(c.Commodities, func(m economysim.Commodity) bool { return m.ID == "staple" })
			return c.Commodities[i].ShelfLife == 200
		}, "staple lasts 200 ticks"},
		{"d6_floor_full", func(c economysim.Config) bool { return c.World.CostFloor == 10_000 }, "cost_floor is 10000"},
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
				// The Stage 0 rules do not see the defects: each is a viability rule
				// (D4), whose failing assertion Stage 1 adds here.
				if err := cfg.Validate(); err != nil {
					t.Errorf("Validate(%q) = %v, want nil at Stage 0", layers, err)
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
