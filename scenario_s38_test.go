package economysim_test

import (
	"fmt"
	"slices"
	"testing"
)

// s38Ticks is the length of a run of S38: the failure of s57Script (ticks 100 to
// 189) and the 110 ticks after it.
const s38Ticks = 300

// TestScenarioS38_ShockAndRecovery runs scenario S38 (POP-S2, plan 2-11 D437) on pack
// one with overlay/estates.toml (the pool and the relief) and overlay/population.toml,
// 300 ticks and the harvest failure of S14 at depth 0 from tick 100 to 189 (pack
// famine is cold and goes extinct by tick 300 with the module off, so it cannot
// carry the "no extinct Warn" check), seeds 1 to
// scenarioSeedCount(), the audit on, the shock scripted through the Stage 1 yield
// seam (the setter of plan 17 is not built, O5). It asserts the exact checks:
// conservation, no negative balance or stock, the counts of PopulationChanged add
// up, and the "region extinct" Warn does not fire. The households, their least
// value, the end mean ratio, the births and deaths and the relief-fed pool are
// reported for the F3 regime; the bands are STY's.
//
// Requirement: FR-POP-022, FR-POP-033, FR-POP-037
//
// Guard: a ladder that emptied the region under the shock, or a death that left
// the books unbalanced, fails here.
func TestScenarioS38_ShockAndRecovery(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S38", "one+estates+population"
	cfg := loadValidPack(t, append(slices.Clone(packOne), "overlay/estates.toml", "overlay/population.toml")...)
	rep := newPackScenReport("one", fmt.Sprintf("S38 %s seeds 1-%d, %d ticks", pack, scenarioSeedCount(), s38Ticks))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				st := runPopulation(t, cfg, seed, s38Ticks, s57Script)
				reportPop(rep, seed, "shock", st)
				rep.num(seed, "pool at the end", st.Pool)
				checkExact(t, rep, scenario, pack, seed, "counts of PopulationChanged add up", st.Bad == "", "first bad event %s", st.Bad)
				checkExact(t, rep, scenario, pack, seed, "no extinct-region Warn", st.Warns["region extinct"] == 0, "%d Warn lines", st.Warns["region extinct"])
			})
		}
	})
	rep.flush(t)
}
