package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestFiscal_PackDutyPosition verifies, end to end on the governed warm five with the
// fiscal file layer, that the sales duty "sales" is taken at P4.1a: in every tick in
// which it is reported it follows the MarketCleared events of every market of the tick
// and precedes TickEnded, so no trade of the tick is taxed before it settled and none
// escapes, and that at least one tick reports it with a due.
//
// Requirement: FR-FIS-033, FR-FIS-044
//
// Guard: the order of P4.1a against the clearing is a position of spec-world.md 6; a
// duty taken per market inside the clearing would pass every unit test of the step.
func TestFiscal_PackDutyPosition(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rec := &recorder{}
	w := mustWorld(t, fiscalPackConfig(t, false), economysim.WithSeed(1), economysim.WithAudit(), economysim.WithObserver(rec))
	stepBoth(t, 20, w)
	lastCleared, ended, taken := map[int64]int{}, map[int64]int{}, 0
	for i, e := range rec.events {
		switch m := e.(type) {
		case economysim.MarketCleared:
			lastCleared[m.Tick] = i
		case economysim.TickEnded:
			ended[m.Tick] = i
		}
	}
	for i, e := range rec.events {
		f, ok := e.(economysim.FiscalFlowApplied)
		if !ok || f.Kind != economysim.KindDuty {
			continue
		}
		if f.Flow != "sales" {
			t.Errorf("duty event of the flow %q, want sales", f.Flow)
		}
		if f.Due > 0 {
			taken++
		}
		if last, ok := lastCleared[f.Tick]; !ok || i < last {
			t.Errorf("duty event %+v at index %d, want it after the last MarketCleared of its tick (index %d, found %v)", f, i, last, ok)
		}
		if i > ended[f.Tick] {
			t.Errorf("duty event %+v at index %d, want it before TickEnded of its tick (index %d)", f, i, ended[f.Tick])
		}
	}
	if taken == 0 {
		t.Error("duty events with a due over 20 ticks = 0, want at least one")
	}
}
