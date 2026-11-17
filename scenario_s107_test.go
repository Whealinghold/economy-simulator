package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s107Seeds is the full seed count of TestScenarioS107_TitlesOnARefillingEndowment, reduced
// by invariantSeeds under -race.
const s107Seeds = 3

// s107Ticks is the length of a run: enough ticks for the stock of the titled endowment to
// be drawn and to regenerate several times.
const s107Ticks = 60

// TestScenarioS107_TitlesOnARefillingEndowment verifies LND-S1 of spec-land.md 16.4: titles
// on a refilling endowment (pack one with the resources and tenure overlays: the growers
// hold 20 basis points each, a mix of agents of weight 1 and cohorts, and the pool of north
// 1 000), seeds 1 to s107Seeds, with the audit on. Every tick of every titled endowment
// satisfies the invariants of the tiers: the units drawn never pass the stock at the start
// of the tick (stock after, plus drawn, less regenerated, plus lost), the grants never
// pass the requests, and nothing is drawn from a stock that is not there; the audit of
// P8.4 holds money and goods (rent moves money between accounts and creates none). The
// reference pack collapses (N-352), so what it draws and whether any unit is let is
// reported and not asserted (spec-reference-chain.md 4.5, D12 of plan-3-lnd.md).
//
// Requirement: FR-LND-007, FR-LND-011, FR-LND-017, FR-LND-037, FR-LND-045
func TestScenarioS107_TitlesOnARefillingEndowment(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, seed := range seedList(invariantSeeds(s107Seeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			s107Run(t, seed)
		})
	}
}

// s107Run is one seed of S107.
func s107Run(t *testing.T, seed uint64) {
	t.Helper()
	cfg := loadValidPack(t, append(slices.Clone(packOneResources), "overlay/tenure.toml")...)
	rec := &recorder{}
	w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))
	for range s107Ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step error = %v, want nil (the audit failed or the step did)", seed, err)
		}
	}
	var drawn, short int64
	var changed int
	for _, e := range rec.events {
		c, ok := e.(economysim.EndowmentChanged)
		if !ok {
			continue
		}
		changed++
		start := c.Stock + c.Drawn - c.Regenerated + c.Lost
		if c.Drawn > start || c.Drawn < 0 {
			t.Errorf("seed %d, tick %d: %d units drawn from a stock of %d at the start", seed, c.Tick, c.Drawn, start)
		}
		if c.Granted > c.Requested || c.Drawn > c.Granted {
			t.Errorf("seed %d, tick %d: requested %d, granted %d, drawn %d, want drawn <= granted <= requested", seed, c.Tick, c.Requested, c.Granted, c.Drawn)
		}
		drawn += c.Drawn
		short += c.Short
	}
	t.Logf("seed %d: %d EndowmentChanged events, %d units drawn, %d units short over %d ticks (reported, not asserted: N-352)", seed, changed, drawn, short, s107Ticks)
}
