package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

const (
	s04Ticks      int64 = 1400
	s04ShockFrom  int64 = 424
	s04ShockTo    int64 = 516
	s04WindowLen  int64 = 10
	s04DuringFrom int64 = s04ShockTo - s04WindowLen // the last ten ticks of the shock
)

// roleRecipes returns the RecipeIDs a role lists, by the names table.
func roleRecipes(t testing.TB, cfg economysim.Config, names economysim.Names, role string) []economysim.RecipeID {
	t.Helper()
	for _, r := range cfg.Roles {
		if r.ID != role {
			continue
		}
		var out []economysim.RecipeID
		for _, id := range r.Recipes {
			if i := indexOf(names.Recipes, id); i >= 0 {
				out = append(out, economysim.RecipeID(i))
			}
		}
		return out
	}
	t.Fatalf("role %q is not in the configuration", role)
	return nil
}

// TestScenarioS04_StapleShortage verifies scenario S04 of
// spec-reference-chain.md 7.2: packs one and one + overlay/nofloor, 1 400 ticks,
// seeds 1 to 10, grow_staple at 5 000 bp from tick 424 to 515 through the script
// and the Stage 1 seam. Exact, both packs: no published price leaves its belief
// interval (FR-STB-008). Recorded for both packs: the longest pinned run of the
// staple (more than half of the outcomes at a bound), the grower role's share of
// working ticks over the shock (the households with a grower job in the Jobs rows
// against the role's households) and the measured demand elasticity of the staple
// between ticks 414 to 423 and 506 to 515 (FR-STB-009). Asserted for pack one,
// only when the key has a value: the longest pinned run against s04.pinned_ticks
// and the absolute value of the signed elasticity against s04.elasticity_max_bp.
// The run without the floor is recorded beside it. The outcomes are written to
// testdata/chains/expected/one.scenarios.txt with -update (D181).
//
// Requirement: FR-STB-009, FR-STB-008, FR-AGT-072, FR-WEL-027, FR-WEL-021,
// FR-WEL-022, FR-WEL-023, FR-REF-020, FR-REF-021, FR-REF-022, FR-REF-024,
// FR-REF-026, FR-REF-027
//
// Guard: a belief interval that lets a price out fails here.
func TestScenarioS04_StapleShortage(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	script := loadScript(t, "s04.toml")
	for _, pk := range []struct {
		name   string
		layers []string
		bands  bool
	}{
		{"one", packOne, true},
		{"one+nofloor", append(slices.Clone(packOne), "overlay/nofloor.toml"), false},
	} {
		t.Run(pk.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, pk.layers...)
			rep := newScenReport(fmt.Sprintf("S04 %s seeds 1-%d, 1400 ticks", pk.name, scenarioSeedCount()))
			t.Run("seeds", func(t *testing.T) {
				for _, seed := range seedList(scenarioSeedCount()) {
					t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
						t.Parallel()
						s04Seed(t, cfg, script, pk.name, seed, rep, pk.bands)
					})
				}
			})
			rep.flush(t)
		})
	}
}

// s04Seed runs one seed of S04 on one pack.
func s04Seed(t *testing.T, cfg economysim.Config, script []shockStep, pack string, seed uint64, rep *scenReport, bands bool) {
	const scenario = "S04"
	var jobHH, roleHH int64
	var growerRecipes []economysim.RecipeID
	run := runScenario(t, cfg, scenarioSpec{
		Seed: seed, Ticks: s04Ticks, Script: script,
		Snap: func(tick int64) bool { return tick >= s04ShockFrom && tick < s04ShockTo },
		OnSnap: func(_ int64, s economysim.Snapshot) {
			if growerRecipes == nil {
				growerRecipes = roleRecipes(t, cfg, s.Names, "grower")
			}
			g, r := indexOf(s.Names.Regions, "north"), indexOf(s.Names.Roles, "grower")
			for _, row := range s.Roles {
				if int(row.Region) == g && int(row.Role) == r {
					roleHH += row.Households
				}
			}
			for _, j := range s.Jobs {
				if int(j.Region) == g && slices.Contains(growerRecipes, j.Recipe) {
					jobHH += j.Households
				}
			}
		},
	})
	o := run.Obs
	checkExact(t, rep, scenario, pack, seed, "market series in tick order", o.badSeq == "", "%s", o.badSeq)
	br := boundsViolations(cfg, run.Names, o, 3)
	rep.num(seed, "market-ticks outside a maker's belief interval (reported)", br.MakerOutside)
	checkExact(t, rep, scenario, pack, seed, "price within belief bounds (FR-STB-008)", len(br.Violations) == 0,
		"measure price bounds: %v", br.Violations)

	staple := marketOf(t, run, "north", "staple")
	pinned := longestRun(pinnedFlags(staple, s04Ticks))
	rep.num(seed, "longest pinned run, staple", pinned)
	if bands {
		checkBand(t, rep, scenario, pack, seed, "s04.pinned_ticks", pinned, true, func(b int64) bool { return pinned > b })
	}
	if share, ok := shareBP(jobHH, roleHH); ok {
		rep.num(seed, "grower working share bp, ticks 424-515", share)
	} else {
		rep.absent(seed, "grower working share bp, ticks 424-515")
	}
	p0, okp0 := windowMean(staple.Price, s04ShockFrom-s04WindowLen, s04ShockFrom)
	q0, okq0 := windowMean(staple.Volume, s04ShockFrom-s04WindowLen, s04ShockFrom)
	p1, okp1 := windowMean(staple.Price, s04DuringFrom, s04ShockTo)
	q1, okq1 := windowMean(staple.Volume, s04DuringFrom, s04ShockTo)
	if !(okp0 && okq0 && okp1 && okq1) {
		t.Fatalf("%s pack %s seed %d: staple series too short for the elasticity windows", scenario, pack, seed)
	}
	rep.num(seed, "staple price mean 414-423", p0)
	rep.num(seed, "staple price mean 506-515", p1)
	rep.num(seed, "staple volume mean 414-423", q0)
	rep.num(seed, "staple volume mean 506-515", q1)
	if el, ok := arcElasticityBP(q0, p0, q1, p1); ok {
		rep.num(seed, "demand elasticity bp (signed)", el)
		if bands {
			checkBand(t, rep, scenario, pack, seed, "s04.elasticity_max_bp", el, true, func(b int64) bool { return absInt(el) > b })
		}
	} else {
		rep.absent(seed, "demand elasticity bp (signed)")
	}
	rep.num(seed, "households at end", run.Final.Households)
}
