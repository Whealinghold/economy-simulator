package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// populationOne is pack one with the population module on at flat rates of 2 000 a
// year (about a third of a birth and of a death per tick over its 600 households)
// and a floor of 40 households, which holds every death of the digger group (30
// households) at once, so a "deaths held" run is non-zero at a fork after the
// first tick.
func populationOne(t testing.TB) economysim.Config {
	cfg := loadValidPack(t, packOne...)
	mod := economysim.DefaultPopulation()
	mod.BirthSchedule = []economysim.Knot{{Ratio: 0, Rate: 2_000}}
	mod.DeathSchedule = []economysim.Knot{{Ratio: 0, Rate: 2_000}}
	mod.MinHouseholds = 40
	cfg.Modules.Population = &mod
	return cfg
}

// TestFork_Population verifies the fork-at-N of pack one with the population module
// on: at N in {0, 37, 211} and after a call between ticks that sets a death scale,
// the original and two restores give the same events, snapshot, audit and
// checkpoint bytes for 30 ticks. The checkpoint holds the death carries, the birth
// carries, the shock scale and a non-zero run of "deaths held".
//
// Requirement: FR-POP-035
func TestFork_Population(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := populationOne(t)
	cells := []struct {
		name string
		n    int
	}{{"at the start", 0}, {"inside a job", 37}, {"inside a season", 211}, {"after a call between ticks", 90}}
	for _, cell := range cells {
		t.Run(fmt.Sprintf("seed 1 %s", cell.name), func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "one with population", Cfg: cfg, Seed: 1, N: cell.n, M: forkM,
				Before: func(w *economysim.World) {
					if !economysim.SetPopulationScale(w, 0, economysim.ScaleDeath, 30_000) {
						t.Errorf("SetPopulationScale() = false, want true")
					}
				},
			})
		})
	}
}
