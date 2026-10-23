package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

const (
	s07Ticks        int64 = 400
	s07DefaultFloor int64 = 7000
)

// TestScenarioS07_PerishableGlut verifies scenario S07 of
// spec-reference-chain.md 7.2: pack one + overlay/glut-prepared (736 households,
// twice the cooks), 400 ticks, seeds 1 to 10, with cost_floor 10 000, 9 000,
// 7 000 (the default) and 5 000, each set by a one-key document layered last
// (loadPackBytes). Measures, per floor: the share of ticks with a trade in
// prepared, the share of prepared made that spoiled (the spoilage BoundaryFlow
// over the outputs of RecipeExecuted) and the longest freeze (no trade while
// both sides have unmatched orders, F5). Exact at the default floor: no
// published price leaves its belief interval (FR-STB-008). Asserted at the
// default floor only when the key has a value: s07.freeze_ticks and
// s07.spoiled_bp. The curve of the four floors is recorded (the outcomes are
// written to testdata/chains/expected/one.scenarios.txt with -update, D181).
//
// Requirement: FR-STB-011, FR-BEL-038, FR-BEL-004, FR-GDS-009, FR-GDS-010,
// FR-REF-020, FR-REF-022, FR-REF-024, FR-REF-026, FR-REF-027
//
// Guard: a belief interval that lets a price out at the default floor fails here.
func TestScenarioS07_PerishableGlut(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration scenario, skipped with -short")
	}
	layers := append(slices.Clone(packOne), "overlay/glut-prepared.toml")
	rep := newScenReport(fmt.Sprintf("S07 one+glut-prepared seeds 1-%d, 400 ticks, cost_floor 10000/9000/7000/5000", scenarioSeedCount()))
	t.Run("floors", func(t *testing.T) {
		for _, floor := range []int64{10000, 9000, s07DefaultFloor, 5000} {
			cfg := loadPackBytes(t, []byte(fmt.Sprintf("[world]\ncost_floor = %d\n", floor)), layers...)
			if cfg.World.CostFloor != floor {
				t.Fatalf("S07: cost_floor = %d after the layer, want %d", cfg.World.CostFloor, floor)
			}
			t.Run(fmt.Sprintf("floor %d", floor), func(t *testing.T) {
				for _, seed := range seedList(scenarioSeedCount()) {
					t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
						t.Parallel()
						s07Seed(t, cfg, floor, seed, rep)
					})
				}
			})
		}
	})
	rep.flush(t)
}

// s07Seed runs one seed of S07 at one floor.
func s07Seed(t *testing.T, cfg economysim.Config, floor int64, seed uint64, rep *scenReport) {
	const scenario = "S07"
	pack := fmt.Sprintf("one+glut-prepared floor %d", floor)
	tag := fmt.Sprintf("floor %d ", floor)
	run := runScenario(t, cfg, scenarioSpec{Seed: seed, Ticks: s07Ticks})
	o := run.Obs
	br := boundsViolations(cfg, run.Names, o, 3)
	if floor == s07DefaultFloor {
		checkExact(t, rep, scenario, pack, seed, "market series in tick order", o.badSeq == "", "%s", o.badSeq)
		checkExact(t, rep, scenario, pack, seed, "price within belief bounds at the default floor (FR-STB-008)", len(br.Violations) == 0,
			"measure price bounds: %v", br.Violations)
	} else {
		rep.num(seed, tag+"price-bound violations (recorded)", int64(len(br.Violations)))
	}
	prepared := marketOf(t, run, "north", "prepared")
	var traded int64
	freeze := make([]bool, len(prepared.Price))
	for i := range prepared.Price {
		if prepared.Volume[i] > 0 {
			traded++
		}
		freeze[i] = prepared.Volume[i] == 0 && prepared.UnmatchedBid[i] > 0 && prepared.UnmatchedAsk[i] > 0
	}
	share, _ := shareBP(traded, int64(len(prepared.Price)))
	rep.num(seed, tag+"trade share bp, prepared", share)
	longest := longestRun(freeze)
	rep.num(seed, tag+"longest freeze, prepared", longest)
	c := indexOf(run.Names.Commodities, "prepared")
	made, spoiled := run.Obs.made[economysim.CommodityID(c)], run.Obs.spoiled[economysim.CommodityID(c)]
	rep.num(seed, tag+"prepared made", made)
	if spoilShare, ok := shareBP(spoiled, made); ok {
		rep.num(seed, tag+"spoiled share bp, prepared", spoilShare)
		if floor == s07DefaultFloor {
			checkBand(t, rep, scenario, pack, seed, "s07.spoiled_bp", spoilShare, true, func(b int64) bool { return spoilShare > b })
		}
	} else {
		rep.absent(seed, tag+"spoiled share bp, prepared")
	}
	if floor == s07DefaultFloor {
		checkBand(t, rep, scenario, pack, seed, "s07.freeze_ticks", longest, true, func(b int64) bool { return longest > b })
	}
	rep.num(seed, tag+"households at end", run.Final.Households)
}
