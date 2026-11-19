package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// storeStrikeTicks returns, for the StoreLoss events of a run, how many ticks held a strike and
// how many owners were struck in all, and whether every event carried the common flag want.
func storeStrikeTicks(events []economysim.Event, want bool) (ticks, owners int64, flagOK bool) {
	flagOK = true
	last := int64(-1)
	for _, e := range events {
		s, ok := e.(economysim.StoreLoss)
		if !ok {
			continue
		}
		if s.Tick != last {
			ticks, last = ticks+1, s.Tick
		}
		owners++
		flagOK = flagOK && s.Common == want
	}
	return ticks, owners, flagOK
}

// TestScenarioRSK4_BufferStoreOwners is RSK-S4 (S129, spec-risk.md 16, 18.2): pack one with the
// storage module and the barn of the growers, struck by one store row, run twice per seed, once
// with the row a unit row (every owner draws for itself) and once with the row common (one draw a
// tick for the whole region), 400 ticks, seeds 1 to the scenario count, audit on. Exact: the
// books of the loss balance every tick, and every StoreLoss carries the common flag of its row.
// Recorded, never asserted (RSK-A2, A3): the ticks that held a strike, the owners struck, the
// units lost: a unit row spreads the strikes over the ticks, a common row puts them in lockstep.
//
// Requirement: FR-RSK-018, FR-RSK-019, FR-RSK-021, FR-RSK-022, FR-RSK-034
//
// Guard: the common flag of an event is the row's, and a common row must not draw per owner.
func TestScenarioRSK4_BufferStoreOwners(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, common := range []bool{false, true} {
		cfg := riskGoldenStore(t).cfg
		cfg.Modules.Risk.Stores[0].Common = common
		name := map[bool]string{false: "unit row", true: "common row"}[common]
		techScenario(t, "one", "S129 one+storage+risk-store the barns under a "+name+", 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
			rw := newRiskWatch(t, cfg, seed, nil)
			msg := rw.run(t, goldenTicks)
			checkExact(t, rep, "S129", "one", seed, "the loss boundary equals the events every tick", msg == "", "%s", msg)
			ticks, owners, flagOK := storeStrikeTicks(rw.rec.events, common)
			checkExact(t, rep, "S129", "one", seed, "every StoreLoss carries the common flag of its row", flagOK, "a StoreLoss differs from the row's flag %t", common)
			rep.num(seed, "ticks with a strike", ticks)
			rep.num(seed, "owners struck", owners)
			rep.num(seed, "units lost from stores", rw.tally.storeLost)
		})
	}
}
