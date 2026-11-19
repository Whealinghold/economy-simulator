package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestFiscal_PayoutInKindStopsAtTheRecipientsRoom verifies that an in-kind payout to a
// household whose account has a store room limit gives it no more units than fit, and
// that the unapplied part stays in the treasury and is counted as headroom, instead
// of the ledger refusing the move.
//
// Arithmetic: the payout is 2 units of grain (mass 1, bulk 0) to each of three home
// millers, 6 due, and the crown holds 6, so no cut falls. The first miller has a mass
// room of 1, so it takes 1 of its 2 units: 5 applied, 1 left in the crown, 1 miller
// short of room (Headroom 1, Cut 0).
//
// Requirement: FR-FIS-039, FR-FIS-063
//
// Guard: a household of a role with stores can be the target of an in-kind payout; the
// ledger refuses a credit that does not fit with ErrNoRoom, which the payout would take
// for a violated invariant and panic on.
func TestFiscal_PayoutInKindStopsAtTheRecipientsRoom(t *testing.T) {
	t.Parallel()
	p := Payout{ID: "ration", Authority: "crown", Target: homeMillers, Commodity: "grain", Amount: 2}
	w, c, rec := payoutWorld(t, p)
	grain := payoutGrain(w)
	w.cfg.Commodities[grain].Mass = 1
	sizes := make([]ledger.Size, len(w.cfg.Commodities))
	sizes[grain] = ledger.Size{Mass: 1}
	if err := w.ledger.SetSizes(sizes); err != nil {
		t.Fatalf("SetSizes() error = %v, want nil", err)
	}
	tight := c.homeMillers[0]
	start := grainUnits(w, w.agents[tight].account)
	if err := w.ledger.SetRoom(w.agents[tight].account, true, start+1, 1); err != nil { // room for 1 more unit
		t.Fatalf("SetRoom() error = %v, want nil", err)
	}
	stockCrown(t, w, 6)
	w.p17bFiscal()
	if got := grainUnits(w, w.agents[tight].account) - start; got != 1 {
		t.Errorf("the miller with room for 1 more received %d grain, want 1", got)
	}
	if got := grainUnits(w, w.polity.tr[0].account); got != 1 {
		t.Errorf("the crown holds %d grain, want 1 (the part with no room stays)", got)
	}
	if e := of[FiscalFlowApplied](rec); len(e) != 1 || e[0].Due != 6 || e[0].Applied != 5 || e[0].Headroom != 1 || e[0].Cut != 0 {
		t.Errorf("events = %+v, want one with due 6, applied 5, headroom 1, cut 0", e)
	}
	auditNow(t, w)
}
