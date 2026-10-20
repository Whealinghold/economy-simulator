package economysim_test

import (
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// packLayers are the layers of the packs the viability rules are checked on
// (spec-reference-chain.md 4.5). The cold `five` is base, lines and five.
var packLayers = map[string][]string{
	"one":    {"base.toml", "one.toml"},
	"two":    {"base.toml", "two.toml"},
	"three":  {"base.toml", "three.toml"},
	"five":   {"base.toml", "lines.toml", "five.toml"},
	"second": {"second/second.toml"},
}

// loadPack loads the layers of testdata/chains.
func loadPack(t *testing.T, layers ...string) economysim.Config {
	t.Helper()
	cfg, err := config.LoadFS(os.DirFS("testdata/chains"), layers...)
	if err != nil {
		t.Fatalf("LoadFS(%q) error = %v, want nil", layers, err)
	}
	return cfg
}

// warned returns the paths of a rule among the warnings, joined by "/".
func warned(ws []economysim.Warning, rule string) []string {
	var out []string
	for _, w := range ws {
		if w.Rule == rule {
			out = append(out, strings.Join(w.Path, "/"))
		}
	}
	return out
}

// TestPacks_PassTheViabilityRules verifies that the packs of REF 7.3 pass V1 to
// V4 (spec-reference-chain.md 7.3), cold and with the warm start that is
// committed, and draw the viability warnings the references expect: none for
// `one` and `two`, target_above_surplus for the south of `three` (its nominal
// surplus factor is 1.411 against the target 1.5) and for the five regions of
// `five` (1.12 to 1.31 against 1.5), and for the crater of `second`.
//
// Requirement: FR-VIA-048, FR-VIA-004, FR-VIA-005, FR-VIA-006, FR-VIA-007, FR-VIA-008, FR-REF-016
func TestPacks_PassTheViabilityRules(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	tests := []struct {
		name   string
		layers []string
		target []string // regions with target_above_surplus
	}{
		{"one", packLayers["one"], nil},
		{"one warm", []string{"base.toml", "one.toml", "warm-one.toml"}, nil},
		{"two", packLayers["two"], nil},
		{"two warm", []string{"base.toml", "two.toml", "warm-two.toml"}, nil},
		{"three", packLayers["three"], []string{"regions/south"}},
		{"three warm", []string{"base.toml", "three.toml", "warm-three.toml"}, []string{"regions/south"}},
		{"five", packLayers["five"], []string{"regions/centre", "regions/east", "regions/north", "regions/south", "regions/west"}},
		{"second", packLayers["second"], []string{"regions/crater_rim"}},
		{"second warm", []string{"second/second.toml", "second/warm-second.toml"}, []string{"regions/crater_rim"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadPack(t, tt.layers...)
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil", tt.name, err)
			}
			ws := cfg.Warnings()
			if got := warned(ws, "target_above_surplus"); !slices.Equal(got, tt.target) {
				t.Errorf("target_above_surplus of %s = %v, want %v", tt.name, got, tt.target)
			}
			for _, rule := range []string{"region_not_self_sufficient", "subsistence_exceeds_expected_work", "storage_gap_input", "viability_check_skipped", "cycle_cost_gain"} {
				if got := warned(ws, rule); len(got) != 0 {
					t.Errorf("%s of %s = %v, want none", rule, tt.name, got)
				}
			}
		})
	}
}

// TestPacks_ReversedListsGiveTheSameOutcome verifies that the packs with every
// entity list in reverse order validate and warn exactly as the sorted ones do,
// which proves that phase 6 canonicalises before it builds the tables.
//
// Requirement: FR-VIA-022
//
// Guard: the tables index typed IDs into the sorted slices and panic on anything
// else, and Validate does not canonicalise its receiver; a phase 6 that read the
// caller's order would crash on a configuration built in Go.
func TestPacks_ReversedListsGiveTheSameOutcome(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for name, layers := range packLayers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := loadPack(t, layers...)
			rev := cfg
			rev.Commodities, rev.Recipes, rev.Yields = slices.Clone(cfg.Commodities), slices.Clone(cfg.Recipes), slices.Clone(cfg.Yields)
			rev.Roles, rev.Regions, rev.Groups = slices.Clone(cfg.Roles), slices.Clone(cfg.Regions), slices.Clone(cfg.Groups)
			slices.Reverse(rev.Commodities)
			slices.Reverse(rev.Recipes)
			slices.Reverse(rev.Yields)
			slices.Reverse(rev.Roles)
			slices.Reverse(rev.Regions)
			slices.Reverse(rev.Groups)
			if err := rev.Validate(); err != nil {
				t.Fatalf("Validate(%s reversed) = %v, want nil", name, err)
			}
			if !reflect.DeepEqual(cfg.Warnings(), rev.Warnings()) {
				t.Errorf("Warnings(%s) differ between the sorted and the reversed lists", name)
			}
		})
	}
}

// TestPacks_DefectsFailWithTheirRules verifies the set of errors Validate gives
// each defect overlay of spec-reference-chain.md 5, over the cold and the warm
// layers: d1 the seed cycle (cycle_not_productive, and unproducible_in_realm for
// the goods that hang on it), d2 six unproducible goods and the forester who
// cannot work, d3 only the idle role, d4 only labour_short, d5 only storage_gap,
// and d6 nothing but its warning. The V3 gate keeps labour_short off d1 and d2.
//
// Requirement: FR-REF-016, FR-VIA-004, FR-VIA-005, FR-VIA-006, FR-VIA-007, FR-VIA-008, FR-VIA-011
func TestPacks_DefectsFailWithTheirRules(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	tests := []struct {
		name string
		want []string // "rule@path", in the order of the sorted list
	}{
		{"d1_seed_cycle", []string{
			"unproducible_in_realm@commodities/prepared",
			"cycle_not_productive@commodities/seed",
			"unproducible_in_realm@commodities/seed",
			"unproducible_in_realm@commodities/staple",
		}},
		{"d2_upkeep_unmade", []string{
			"unproducible_in_realm@commodities/fuel",
			"unproducible_in_realm@commodities/prepared",
			"unproducible_in_realm@commodities/refined",
			"unproducible_in_realm@commodities/staple",
			"unproducible_in_realm@commodities/tool_a",
			"unproducible_in_realm@commodities/tool_b",
			"role_cannot_work@roles/forester",
		}},
		{"d3_idle_role", []string{"role_cannot_work@roles/idle"}},
		{"d4_long_job", []string{"labour_short@regions"}},
		{"d5_storage_gap", []string{"storage_gap@regions/north/commodities/staple"}},
		{"d6_floor_full", nil},
	}
	for _, tt := range tests {
		for _, warm := range []bool{false, true} {
			name := tt.name + "/cold"
			layers := []string{"base.toml", "one.toml"}
			if warm {
				name, layers = tt.name+"/warm", append(layers, "warm-one.toml")
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				cfg := loadPack(t, append(layers, "defects/"+tt.name+".toml")...)
				err := cfg.Validate()
				var got []string
				if err != nil {
					for _, e := range err.(interface{ Unwrap() []error }).Unwrap() {
						var ve *economysim.ViabilityError
						if !errors.As(e, &ve) {
							t.Fatalf("Validate(%s) error %v, want only viability errors", name, e)
						}
						got = append(got, ve.Rule+"@"+strings.Join(ve.Path, "/"))
					}
				}
				if !slices.Equal(got, tt.want) {
					t.Errorf("Validate(%s) = %v, want %v", name, got, tt.want)
				}
				if tt.name == "d6_floor_full" && !slices.Contains(warned(cfg.Warnings(), "cost_floor_at_full_cost"), "world/cost_floor") {
					t.Errorf("Warnings(%s) has no cost_floor_at_full_cost, want it", name)
				}
			})
		}
	}
}

// TestPacks_D2OverlayTakesTheRawMarkerOff verifies that the overlay mechanism
// honours an explicit `raw = false` over the base's `raw = true`, which d2 needs:
// a Raw good is sourced (spec-viability.md 4.2.1), so d2 gives its rule only if
// fuel is not Raw.
//
// Requirement: FR-REF-016, FR-VIA-059
func TestPacks_D2OverlayTakesTheRawMarkerOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	base := loadPack(t, "base.toml", "one.toml")
	d2 := loadPack(t, "base.toml", "one.toml", "defects/d2_upkeep_unmade.toml")
	fuel := func(c economysim.Config) bool {
		i := slices.IndexFunc(c.Commodities, func(m economysim.Commodity) bool { return m.ID == "fuel" })
		return c.Commodities[i].Raw
	}
	if !fuel(base) || fuel(d2) {
		t.Errorf("fuel is Raw: base %v, d2 %v, want true and false", fuel(base), fuel(d2))
	}
}

// TestPacks_OnlyD1ToD5AreNotViable verifies that, layered over `one` and its
// warm start, every overlay file that loads at this stage still validates, so
// the loops that skip a pack that stops validating (role_pack_test.go,
// welfare_pack_test.go) hide nothing, and that d1 to d5 are the only layered
// files that do not.
//
// Requirement: FR-REF-016
//
// Guard: those loops `continue` on a Validate error; a phase 6 that began to
// refuse a scenario overlay would silently take it out of their checks.
func TestPacks_OnlyD1ToD5AreNotViable(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	fsys := os.DirFS("testdata/chains")
	var files []string
	for _, dir := range []string{"overlay", "defects"} {
		entries, err := os.ReadDir("testdata/chains/" + dir)
		if err != nil {
			t.Fatalf("ReadDir(%s) error = %v, want nil", dir, err)
		}
		for _, e := range entries {
			files = append(files, dir+"/"+e.Name())
		}
	}
	var notViable []string
	loaded := 0
	for _, f := range files {
		cfg, err := config.LoadFS(fsys, "base.toml", "one.toml", "warm-one.toml", f)
		if err != nil {
			continue // names a module that is not built yet
		}
		loaded++
		if err := cfg.Validate(); err != nil {
			if !errors.Is(err, economysim.ErrNotViable) {
				t.Errorf("Validate(%s) = %v, want nil or ErrNotViable", f, err)
			}
			notViable = append(notViable, f)
		}
	}
	want := []string{"defects/d1_seed_cycle.toml", "defects/d2_upkeep_unmade.toml", "defects/d3_idle_role.toml", "defects/d4_long_job.toml", "defects/d5_storage_gap.toml"}
	if !slices.Equal(notViable, want) {
		t.Errorf("layered files that are not viable = %v, want %v", notViable, want)
	}
	if loaded < 10 {
		t.Errorf("%d files loaded, want at least 10 (the scan must read the overlays)", loaded)
	}
}

// TestPacks_A5SouthImportsItsStaple verifies the acceptance scenario A5 of REF 7.6
// at the level of Validate: on `two` with the overlay a5 the south cannot make
// its staple, the pooled realm passes (Validate is nil), and the warning
// region_not_self_sufficient names the goods the south imports.
//
// Requirement: FR-VIA-009, FR-VIA-004
func TestPacks_A5SouthImportsItsStaple(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPack(t, "base.toml", "two.toml", "overlay/a5.toml")
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(two + a5) = %v, want nil: the pooled realm can feed itself", err)
	}
	want := []string{"regions/south/commodities/prepared", "regions/south/commodities/staple"}
	if got := warned(cfg.Warnings(), "region_not_self_sufficient"); !slices.Equal(got, want) {
		t.Errorf("region_not_self_sufficient of two + a5 = %v, want %v", got, want)
	}
}

// TestPacks_FiveWithItsWarmStartValidates verifies that `five` with the committed
// warm-five.toml validates: the document was regenerated from the solver's table,
// whose two free goods are lifted into the reference band (51 and 52 where the old
// document held 50, under the band floors).
//
// Requirement: FR-VIA-030, FR-REF-007
func TestPacks_FiveWithItsWarmStartValidates(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPack(t, "base.toml", "lines.toml", "five.toml", "warm-five.toml")
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate(five + warm-five) = %v, want nil", err)
	}
}
