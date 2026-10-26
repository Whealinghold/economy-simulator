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

// s04DemandTicks are the ticks at which S04 with and without the wants records
// the households, the cooks' share of money and the staple price.
var s04DemandTicks = []int64{100, 300, 600}

// TestScenarioS04_WithAndWithoutDemand records, for N-314 and D317 of plan 2-5,
// what the demand module does to the money trap of the reference packs: pack one
// (the module off) and pack one-demand (the module on, its own census), seeds 1
// to 3, 600 ticks, no shock. At ticks 100, 300 and 600 it records the households,
// the cooks' share of all the money held by agents in basis points and the staple
// price, for both packs. Exact: no price leaves its belief interval in either
// pack (FR-STB-008) and the cooks' share lies in [0, 10 000]. The numbers are
// written to testdata/chains/expected/one-demand.scenarios.txt with -update and
// asserted on nothing else: D317 predicts that the cooks' hoard still grows with
// the module on, so the share is a measurement, not a pass condition.
//
// Requirement: FR-DEM-023, FR-STB-008
//
// Guard: a share outside [0, 10 000] means the per-role money probe is wrong.
func TestScenarioS04_WithAndWithoutDemand(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S04", "one-demand"
	last := s04DemandTicks[len(s04DemandTicks)-1]
	rep := newPackScenReport(pack, fmt.Sprintf("S04 with and without the wants seeds 1-%d, %d ticks", min(3, scenarioSeedCount()), last))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(min(3, scenarioSeedCount())) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				for _, pk := range []struct {
					name   string
					layers []string
				}{{"wants off (pack one)", packOne}, {"wants on", packOneDemand}} {
					cfg := loadValidPack(t, pk.layers...)
					var cooks int
					households := map[int64]int64{}
					shares := map[int64]int64{}
					run := runScenario(t, cfg, scenarioSpec{
						Seed: seed, Ticks: last,
						Probe: func(tick int64, w *economysim.World) {
							at := tick + 1
							if !slices.Contains(s04DemandTicks, at) {
								return
							}
							s, err := w.Snapshot()
							if err != nil {
								t.Fatalf("Snapshot after tick %d error = %v, want nil", tick, err)
							}
							cooks = indexOf(s.Names.Roles, "cook")
							money := economysim.RoleMoneyForTest(w, len(s.Names.Roles))
							var total int64
							for _, m := range money {
								total += m
							}
							households[at] = s.Households
							if share, ok := shareBP(money[cooks], total); ok {
								shares[at] = share
							} else {
								shares[at] = -1
							}
						},
					})
					checkExact(t, rep, scenario, pack, seed, "price within belief bounds (FR-STB-008)", len(boundsViolations(cfg, run.Names, run.Obs, 3).Violations) == 0,
						"%s: a price left its belief interval", pk.name)
					staple := marketOf(t, run, "north", "staple")
					for _, at := range s04DemandTicks {
						rep.num(seed, fmt.Sprintf("households at tick %d, %s", at, pk.name), households[at])
						checkExact(t, rep, scenario, pack, seed, "cooks' share of money in [0, 10 000]", shares[at] >= -1 && shares[at] <= 10_000,
							"%s tick %d: share %d", pk.name, at, shares[at])
						if shares[at] < 0 {
							rep.absent(seed, fmt.Sprintf("cooks' share of money bp at tick %d, %s", at, pk.name))
						} else {
							rep.num(seed, fmt.Sprintf("cooks' share of money bp at tick %d, %s", at, pk.name), shares[at])
						}
						rep.num(seed, fmt.Sprintf("staple price at tick %d, %s", at, pk.name), staple.Price[at-1])
					}
				}
			})
		}
	})
	rep.flush(t)
}
