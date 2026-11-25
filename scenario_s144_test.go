package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// savingS144Variants are the legs of S144: the spread of the hurdle by agent, the commit band and the
// repair gate of the cook, in basis points. The first leg is the overlay's own band, the last two
// have the gate at 0, which never repairs (spec 5.8).
var savingS144Variants = []struct{ spread, band, repair int64 }{
	{0, 0, 5_000}, {0, 2_000, 5_000}, {5_000, 2_000, 5_000}, {40_000, 0, 5_000}, {40_000, 2_000, 5_000}, {40_000, 2_000, 0},
}

// TestScenarioS144_AMillIsBuiltAndMended reports SAV-S3 of spec-saving-and-investment.md 14.3 on
// pack one with the infrastructure and saving overlays, 200 ticks, audit on: the cooks that hold no
// mill save for one at a hurdle spread of 0, 5 000 and 40 000, a commit band of 0 and 2 000, with
// the repair gate at the default and at 0. The identities are asserted (the audit holds; every
// facility built has a build request in the counts of the events; a gate of 0 asks for no repair and
// no facility is repaired); the rest is reported: the builds, the repairs, and the households
// barred by `NoReturn` and `Unreachable`. The standing-refusal variant of the spec (a class held at
// the bound of facilities) needs a bound the pack's one class does not reach, and is not run here.
//
// Requirement: FR-SAV-015, FR-SAV-016, FR-SAV-021, FR-SAV-022, FR-SAV-033
//
// Guard: a repair requested with the gate at 0, or a facility built with no request, fails here.
func TestScenarioS144_AMillIsBuiltAndMended(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	savingScenario(t, "S144 one+saving build and repair by spread, band and gate, 200 ticks", scenarioSeedCount(), func(t *testing.T, rep *scenReport, seed uint64) {
		for _, v := range savingS144Variants {
			cfg := savingPack(t, func(c *economysim.Config) {
				c.Modules.Saving.HurdleSpreadBP, c.Modules.Saving.CommitBandBP = v.spread, v.band
				roleOf(c, "cook").Saver.RepairBelowBP = v.repair
			})
			st := savingRun(t, cfg, seed, 200)
			name := fmt.Sprintf("spread %d band %d gate %d", v.spread, v.band, v.repair)
			rep.num(seed, name+": built", st.Built)
			rep.num(seed, name+": repaired", st.Repaired)
			rep.num(seed, name+": no return household-ticks", st.NoReturn)
			rep.num(seed, name+": unreachable household-ticks", st.Unreachable)
			checkExact(t, rep, "S144", "one", seed, name+": no facility is built without a build request",
				st.Built <= st.Builds, "%d built against %d requested", st.Built, st.Builds)
			if v.repair == 0 {
				checkExact(t, rep, "S144", "one", seed, name+": a gate of 0 never repairs", st.Repairs == 0 && st.Repaired == 0,
					"%d repair requests and %d repaired", st.Repairs, st.Repaired)
			}
		}
	})
}
