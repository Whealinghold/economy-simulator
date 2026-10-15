package economysim_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// rolePackLayers lists the pack layers the role table is checked over: the two
// settings, the defect overlays and the scenario overlays.
func rolePackLayers(t *testing.T) map[string][]string {
	t.Helper()
	layers := map[string][]string{
		"one":    {"base.toml", "one.toml", "warm-one.toml"},
		"second": {"second/second.toml", "second/warm-second.toml"},
	}
	for _, d := range []string{"d1_seed_cycle", "d2_upkeep_unmade", "d3_idle_role", "d4_long_job", "d5_storage_gap", "d6_floor_full"} {
		layers["defect "+d] = []string{"base.toml", "one.toml", "defects/" + d + ".toml"}
	}
	entries, err := os.ReadDir("testdata/chains/overlay")
	if err != nil {
		t.Fatalf("ReadDir(overlay) error = %v, want nil", err)
	}
	for _, e := range entries {
		layers["overlay "+e.Name()] = []string{"base.toml", "one.toml", "warm-one.toml", "overlay/" + e.Name()}
	}
	return layers
}

// TestRoleTable_AgreesWithValidator verifies that over every pack and overlay
// that validates, each role's tradable set equals the one tradableSets
// returns, the membership flags match the set, a commodity is flagged
// tradable exactly when it is in some set, and the commodities in no set that
// are not discard-only are exactly those the warning commodity_untraded names.
// It also checks that the tables hold no ID out of range and that the role
// and recipe counts match the configuration.
//
// Requirement: FR-GDS-002, FR-AGT-003, FR-AGT-004
//
// Guard: the table and the validator share one implementation of AGT 3.4; if
// the table were re-derived, the world and the validator could disagree about
// which commodity is tradable and no other test would say so.
func TestRoleTable_AgreesWithValidator(t *testing.T) {
	t.Parallel()
	fsys := os.DirFS("testdata/chains")
	checked := 0
	for name, layers := range rolePackLayers(t) {
		raw, err := config.LoadFS(fsys, layers...)
		if err != nil {
			if strings.HasPrefix(name, "overlay ") {
				// An overlay may name a module that is not built yet.
				continue
			}
			t.Fatalf("pack %s: LoadFS error = %v, want nil", name, err)
		}
		if raw.Validate() != nil {
			continue
		}
		checked++
		cfg := economysim.Canonical(raw)
		tab := economysim.NewRoleTable(&cfg)
		sets := economysim.TradableSets(&cfg)
		if len(sets) != len(cfg.Roles) {
			t.Fatalf("pack %s: %d tradable sets, want %d", name, len(sets), len(cfg.Roles))
		}
		union := map[string]bool{}
		for ri := range cfg.Roles {
			v := economysim.RoleRow(&tab, economysim.RoleID(ri))
			var got []string
			for _, id := range v.Tradable {
				if id < 0 || id >= len(cfg.Commodities) {
					t.Fatalf("pack %s role %s: tradable ID %d out of range", name, cfg.Roles[ri].ID, id)
				}
				got = append(got, cfg.Commodities[id].ID)
			}
			var want []string
			for id := range sets[ri] {
				want = append(want, id)
				union[id] = true
			}
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Errorf("pack %s role %s: tradable = %v, want %v", name, cfg.Roles[ri].ID, got, want)
			}
			for ci, c := range cfg.Commodities {
				if v.Member[ci] != sets[ri][c.ID] {
					t.Errorf("pack %s role %s: member[%s] = %v, want %v", name, cfg.Roles[ri].ID, c.ID, v.Member[ci], sets[ri][c.ID])
				}
			}
			for _, id := range v.Recipes {
				if id < 0 || id >= len(cfg.Recipes) {
					t.Errorf("pack %s role %s: recipe ID %d out of range", name, cfg.Roles[ri].ID, id)
				}
			}
			if len(v.Recipes) != len(cfg.Roles[ri].Recipes) || len(v.Trades) != len(cfg.Roles[ri].Trades) {
				t.Errorf("pack %s role %s: %d recipes and %d trades, want %d and %d", name, cfg.Roles[ri].ID,
					len(v.Recipes), len(v.Trades), len(cfg.Roles[ri].Recipes), len(cfg.Roles[ri].Trades))
			}
		}
		discardOnly := map[string]bool{}
		for _, c := range cfg.Commodities {
			discardOnly[c.ID] = true
		}
		for _, r := range cfg.Recipes {
			for _, o := range r.Outputs {
				if !o.Discard {
					delete(discardOnly, o.Commodity)
				}
			}
			for _, in := range r.Inputs {
				delete(discardOnly, in.Commodity)
			}
			for _, tl := range r.Requires {
				delete(discardOnly, tl.Commodity)
			}
		}
		for _, c := range cfg.Commodities {
			// A commodity that is a discard output and appears in no recipe
			// list as input, tool or real output is exempt from the warning.
			isDiscardOut := false
			for _, r := range cfg.Recipes {
				for _, o := range r.Outputs {
					isDiscardOut = isDiscardOut || (o.Commodity == c.ID && o.Discard)
				}
			}
			untraded := !union[c.ID] && !(isDiscardOut && discardOnly[c.ID])
			warned := false
			for _, w := range cfg.Warnings() {
				warned = warned || (w.Rule == "commodity_untraded" && slices.Equal(w.Path, []string{"commodities", c.ID}))
			}
			if flag := economysim.RoleTradableSomewhere(&tab, economysim.CommodityID(slices.IndexFunc(cfg.Commodities, func(x economysim.Commodity) bool { return x.ID == c.ID }))); flag != union[c.ID] {
				t.Errorf("pack %s: tradableSomewhere(%s) = %v, want %v", name, c.ID, flag, union[c.ID])
			}
			if warned != untraded {
				t.Errorf("pack %s: commodity_untraded for %s = %v, want %v", name, c.ID, warned, untraded)
			}
		}
	}
	if checked < 2 {
		t.Errorf("validated %d packs, want at least the two settings", checked)
	}
}
