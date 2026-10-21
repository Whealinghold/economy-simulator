package economysim_test

import (
	"os"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// TestYieldTable_TwoSettings verifies over the packs one and second, with no
// code specific to either, that the site factor of every (region, recipe,
// season) equals what the configuration's profile strings say (the season's
// entry, else All, else 0 for a profile that lists seasons, else 10 000 for a
// recipe that is not Sited and 0 for one that is), that has says whether a
// profile row exists, and that the residue layout of every role is the sum of
// its recipes' output counts with offsets that tile it.
//
// Requirement: FR-PRO-001, FR-PRO-020, FR-PRO-028, FR-PRO-002
//
// Guard: architecture.md 9.4, a table tuned to one pack's names would pass the
// reference chain and fail the other.
func TestYieldTable_TwoSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	fsys := os.DirFS("testdata/chains")
	packs := map[string][]string{
		"one":    {"base.toml", "one.toml", "warm-one.toml"},
		"second": {"second/second.toml", "second/warm-second.toml"},
	}
	for name, layers := range packs {
		raw, err := config.LoadFS(fsys, layers...)
		if err != nil {
			t.Fatalf("pack %s: LoadFS error = %v, want nil", name, err)
		}
		cfg := economysim.Canonical(raw)
		tab := economysim.NewYieldTable(&cfg)
		profiles := 0
		for gi, g := range cfg.Regions {
			for ri, r := range cfg.Recipes {
				var prof *economysim.YieldProfile
				for k := range cfg.Yields {
					if cfg.Yields[k].Region == g.ID && cfg.Yields[k].Recipe == r.ID {
						prof = &cfg.Yields[k]
						profiles++
					}
				}
				// Season -1 is "no season"; then each calendar index.
				for season := -1; season < len(cfg.Calendar.Seasons); season++ {
					want := int64(10_000)
					if r.Sited {
						want = 0
					}
					if prof != nil {
						want = 0
						switch {
						case season >= 0 && hasSeason(prof, cfg.Calendar.Seasons[season].ID):
							want = seasonBP(prof, cfg.Calendar.Seasons[season].ID)
						case prof.HasAll:
							want = prof.All
						}
					}
					got, has := economysim.YieldSite(&tab, economysim.RegionID(gi), economysim.RecipeID(ri), season, r.Sited)
					if got != want || has != (prof != nil) {
						t.Errorf("pack %s: site(%s, %s, season %d) = (%d, %v), want (%d, %v)",
							name, g.ID, r.ID, season, got, has, want, prof != nil)
					}
				}
			}
		}
		if profiles != len(cfg.Yields) {
			t.Errorf("pack %s: matched %d profiles, want %d", name, profiles, len(cfg.Yields))
		}

		rt := economysim.NewRecipeTable(&cfg)
		roles := economysim.NewRoleTable(&cfg)
		for ri, r := range cfg.Roles {
			var list []economysim.RecipeID
			for _, id := range economysim.RoleTableRow(&roles, economysim.RoleID(ri)).Recipes {
				list = append(list, economysim.RecipeID(id))
			}
			next := 0
			for _, rid := range list {
				off, ok := economysim.ResidueOffset(list, &rt, rid)
				if !ok || off != next {
					t.Errorf("pack %s role %s: offset of recipe %d = (%d, %v), want (%d, true)", name, r.ID, rid, off, ok, next)
				}
				next += len(cfg.Recipes[rid].Outputs)
			}
			if got := economysim.ResidueLen(list, &rt); got != next {
				t.Errorf("pack %s role %s: residue length = %d, want %d", name, r.ID, got, next)
			}
		}
	}
}

func hasSeason(p *economysim.YieldProfile, id string) bool {
	for _, s := range p.Seasons {
		if s.Season == id {
			return true
		}
	}
	return false
}

func seasonBP(p *economysim.YieldProfile, id string) int64 {
	for _, s := range p.Seasons {
		if s.Season == id {
			return s.BP
		}
	}
	return 0
}

// TestProductionSteps_TwoSettings runs the same script of starts, advances,
// completions, abandons, weight falls and shocks over the packs one and second
// with no code specific to either, against a real ledger: the audit and the
// conservation of goods hold after every operation, and the script reaches
// starts and completions in each pack.
//
// Requirement: FR-PRO-001, FR-PRO-028, FR-PRO-044
//
// Guard: architecture.md 9.4, steps tuned to the names or the shape of one
// pack would pass the reference chain and fail the other.
func TestProductionSteps_TwoSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	fsys := os.DirFS("testdata/chains")
	packs := []struct {
		name   string
		layers []string
	}{
		{"one", []string{"base.toml", "one.toml", "warm-one.toml"}},
		{"second", []string{"second/second.toml", "second/warm-second.toml"}},
	}
	for _, p := range packs {
		raw, err := config.LoadFS(fsys, p.layers...)
		if err != nil {
			t.Fatalf("pack %s: LoadFS error = %v, want nil", p.name, err)
		}
		cfg := economysim.Canonical(raw)
		var starts, completions int
		for seed := uint64(1); seed <= 6; seed++ {
			st, err := economysim.ProductionScript(&cfg, seed, 2_000)
			if err != nil {
				t.Fatalf("pack %s: %v", p.name, err)
			}
			starts += st.Starts
			completions += st.Completions
		}
		if starts == 0 || completions == 0 {
			t.Errorf("pack %s: script made %d starts and %d completions, want both above 0", p.name, starts, completions)
		}
	}
}
