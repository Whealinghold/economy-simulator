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

// TestFiscal_PayoutNothingToPay verifies that a treasury with nothing of the resource
// pays nothing and does not fail: an empty store of an in-kind payout and a treasury
// with no money both leave every recipient as it was, count the whole due as cut, and
// keep the audit.
//
// Arithmetic: three home millers are due 2 grain, or 10 money, against a treasury that
// holds none: 6 and 30 due, 0 applied, and all three recipients cut.
//
// Requirement: FR-FIS-063
//
// Guard: the cut refuses a negative stock; the payout reads the stock floored at 0, so
// an empty or fully reserved treasury must not reach that refusal.
func TestFiscal_PayoutNothingToPay(t *testing.T) {
	t.Parallel()
	kind := Payout{ID: "ration", Authority: "crown", Target: homeMillers, Commodity: "grain", Amount: 2}
	for _, tt := range []struct {
		name string
		p    Payout
		due  int64
		cash bool
	}{
		{"an empty store", kind, 6, false},
		{"a treasury with no money", stipend("alms", 10), 30, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, rec := payoutWorld(t, tt.p)
			if tt.cash {
				if err := w.ledger.Transfer(w.polity.tr[0].account, w.agents[7].account, crownMoney(t, w), ledger.Trade); err != nil {
					t.Fatalf("drain the crown: %v", err)
				}
			}
			before := moneyTotal(w)
			w.p17bFiscal()
			e := of[FiscalFlowApplied](rec)
			if len(e) != 1 || e[0].Due != Money(tt.due) || e[0].Applied != 0 || e[0].Cut != 3 {
				t.Errorf("events = %+v, want one with due %d, applied 0, cut 3", e, tt.due)
			}
			if got := moneyTotal(w); got != before {
				t.Errorf("money = %d, want %d", got, before)
			}
			auditNow(t, w)
		})
	}
}

// TestNewFiscal_PayoutUnknownCommodityPanics verifies that a payout that names an
// undefined commodity is refused at construction as a violated invariant, instead of
// being read as a payout of money.
//
// Guard: the validator rejects the name (ErrUnknownCommodity), and the lookup of the
// commodity index answers -1 for it, which is also the index of money.
func TestNewFiscal_PayoutUnknownCommodityPanics(t *testing.T) {
	t.Parallel()
	w, _, _ := payoutWorld(t, Payout{ID: "ration", Authority: "crown", Target: homeMillers, Commodity: "grain", Amount: 2})
	cfg := w.cfg
	fiscal := *cfg.Modules.Fiscal
	fiscal.Payouts = []Payout{{ID: "ration", Authority: "crown", Target: homeMillers, Commodity: "nowhere", Amount: 2}}
	cfg.Modules.Fiscal = &fiscal
	defer func() {
		if recover() == nil {
			t.Error("newFiscal() did not panic, want a violated invariant")
		}
	}()
	newFiscal(&cfg, w.polity)
}
