package economysim

import (
	"testing"
)

// TestBoundaryUpkeep_ReadsThePublishedPrice verifies that the upkeep of the credit
// group of Agent is valued at the published price of a market that has one, and at
// the reference price of a market that has none, and that the reading leaves the
// reports and the valued prices of the world alone.
//
// Setup: the loan world after two steps, where no order was placed, so no market has
// a price and the upkeep of a miller is 1 000 at the reference prices (see
// TestCredit_AgentState). The test then gives every market of the region a published
// price of twice the reference; the upkeep is linear in the prices, so it becomes
// 2 * 1 000 = 2 000.
//
// Guard: the first branch (a price window with a value) is reached only after the
// first trade of a market, which the scripted worlds never make; without this test
// a swapped branch would value every figure of a traded world at the reference price.
//
// Requirement: FR-CRD-050
func TestBoundaryUpkeep_ReadsThePublishedPrice(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	w, _, _ := loanWorld(t, s)
	step(t, w, 2)
	a := &w.agents[hMillA]
	if got := w.boundaryUpkeep(a); got != 1_000 {
		t.Fatalf("boundaryUpkeep with no published price = %d, want 1 000 (the reference prices)", got)
	}
	g, nc := int(w.regionOf(a)), len(w.cfg.Commodities)
	for c := range nc {
		i := g*nc + c
		w.markets.reports[i].Price = Price(2 * int64(w.refs[i]))
		w.markets.windows[i].Push(2 * int64(w.refs[i]))
	}
	valued := append([]int64(nil), w.oc.valued...)
	if got := w.boundaryUpkeep(a); got != 2_000 {
		t.Errorf("boundaryUpkeep with a published price of twice the reference = %d, want 2 000", got)
	}
	for i, v := range w.oc.valued {
		if v != valued[i] {
			t.Fatalf("boundaryUpkeep changed valued[%d] from %d to %d, want the scratch left alone", i, valued[i], v)
		}
	}
}
