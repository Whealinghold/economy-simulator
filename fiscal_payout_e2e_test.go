package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiscalPayoutDoc is a fiscal overlay for the governed warm five: the head tax of
// the pack on the cooks of the seat region, and a payout to the same cooks that the
// treasury cannot meet (the root starts with 50 000 and the tax brings one per cook a
// tick), so the pro rata cut falls from the first tick.
const fiscalPayoutDoc = `
[modules.fiscal.tax.poll]
authority = "root"
target = { kind = "households", region = "centre", role = "cook" }
amount = 1

[modules.fiscal.payout.alms]
authority = "root"
target = { kind = "households", region = "centre", role = "cook" }
amount = 100000
`

// fiscalInKindDoc pays the same cooks one unit of good_a a tick in kind from a
// treasury that holds none: PR 4 builds no standing order, so no rule gives a treasury
// stock, and the payout has nothing to pay.
const fiscalInKindDoc = `
[modules.fiscal.tax.poll]
authority = "root"
target = { kind = "households", region = "centre", role = "cook" }
amount = 1

[modules.fiscal.payout.ration]
authority = "root"
target = { kind = "households", region = "centre", role = "cook" }
commodity = "good_a"
amount = 1
`

// fiscalFlowSums splits the events of a run into the money of taxes and duties that
// moved (taxed) and of money payouts, and the payout events, so a test can rebuild the
// treasury from them.
func fiscalFlowSums(evs []economysim.FiscalFlowApplied) (taxed, paid economysim.Money, payouts []economysim.FiscalFlowApplied) {
	for _, e := range evs {
		switch e.Kind {
		case economysim.KindTax, economysim.KindDuty:
			taxed += e.Applied
		case economysim.KindPayout:
			payouts = append(payouts, e)
			if e.Commodity == economysim.NoCommodity {
				paid += e.Applied
			}
		}
	}
	return taxed, paid, payouts
}

// cutEventFault returns what is wrong with e as the event of the money payout alms, or "".
func cutEventFault(e economysim.FiscalFlowApplied) string {
	switch {
	case e.Flow != "alms":
		return "want the flow alms"
	case e.Applied < 0 || e.Applied > e.Due:
		return "want 0 <= Applied <= Due"
	case e.Headroom != 0:
		return "want Headroom 0: no recipient is near MaxMoney"
	case e.Cut < 0 || e.Cut > e.Parties:
		return "want 0 <= Cut <= Parties"
	}
	return ""
}

// emptyStoreFault returns what is wrong with e as the event of an in-kind payout of
// good from a treasury that holds none, or "".
func emptyStoreFault(e economysim.FiscalFlowApplied, good economysim.CommodityID) string {
	switch {
	case e.Commodity != good:
		return "want the payout's commodity"
	case e.Due <= 0 || e.Applied != 0:
		return "want Due > 0 and Applied 0"
	case e.Cut != e.Parties || e.Headroom != 0:
		return "want every recipient cut (Cut = Parties) and Headroom 0"
	}
	return ""
}

// TestFiscal_PackPayoutIsCutProRataAndConservesMoney verifies, end to end through the
// public API on the governed warm five, that a payout larger than the treasury is cut
// and reported as such, and that money is conserved: the treasury ends at its start
// money plus the taxes applied less the payouts applied, with no tolerance, on a run
// whose audit checks the ledger every tick.
//
// Arithmetic: the root holds 50 000 and the payout is 100 000 a cook a tick, so on the
// first tick Due exceeds what the treasury holds and Applied is at most that: Applied
// < Due and Cut counts the recipients that got less. Headroom stays 0, since no
// recipient is near MaxMoney.
//
// Requirement: FR-FIS-039, FR-FIS-063, FR-FIS-044
//
// Guard: a payout that paid in full from an empty treasury would mint money (the audit
// fails); one that cut and did not report it would leave Due = Applied and the cut
// invisible to every consumer of the event.
func TestFiscal_PackPayoutIsCutProRataAndConservesMoney(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const seed, ticks = 1, 10
	cfg := loadPackBytes(t, []byte(fiscalPayoutDoc), fivePolityGoverned...)
	w, evs, start := fiscalEvents(t, cfg, seed, ticks)
	taxed, paid, payouts := fiscalFlowSums(evs)
	if len(payouts) == 0 {
		t.Fatalf("payout events over %d ticks (seed %d) = 0, want one a tick", ticks, seed)
	}
	for _, e := range payouts {
		if fault := cutEventFault(e); fault != "" {
			t.Errorf("payout event %+v (seed %d): %s", e, seed, fault)
		}
	}
	if !slices.ContainsFunc(payouts, func(e economysim.FiscalFlowApplied) bool { return e.Cut > 0 && e.Applied < e.Due }) {
		t.Errorf("payout events %+v (seed %d), want at least one with Cut > 0 and Applied < Due", payouts, seed)
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if want, got := start+taxed-paid, snap.Jurisdictions[0].Money; got != want || got < 0 {
		t.Errorf("treasury after %d ticks (seed %d) = %d, want %d = start %d + taxed %d - paid %d, and not negative", ticks, seed, got, want, start, taxed, paid)
	}
}

// TestFiscal_PackInKindPayoutOfAnEmptyTreasuryPaysNothing verifies, end to end, that an
// in-kind payout from a treasury with no stock of the commodity moves no goods and no
// money: every event names the commodity, applies 0 of a due above 0, counts every
// recipient as cut, and counts none at the headroom (that count is for a recipient with
// no store room, not for an empty treasury), and the treasury's money rises by the tax
// alone.
//
// Requirement: FR-FIS-039, FR-FIS-063
//
// Guard: a payout of a good the treasury lacks must not fall back to paying money (the
// fix of b0b769b) and must not count its shortfall as headroom (the stage 5 correction
// of N-504: Headroom is store room or MaxMoney, Cut is a short treasury).
func TestFiscal_PackInKindPayoutOfAnEmptyTreasuryPaysNothing(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const seed, ticks = 1, 10
	cfg := loadPackBytes(t, []byte(fiscalInKindDoc), fivePolityGoverned...)
	goodA := commodityID(t, cfg, "good_a")
	w, evs, start := fiscalEvents(t, cfg, seed, ticks)
	taxed, _, payouts := fiscalFlowSums(evs)
	if len(payouts) == 0 {
		t.Fatalf("payout events over %d ticks (seed %d) = 0, want one a tick", ticks, seed)
	}
	for _, e := range payouts {
		if fault := emptyStoreFault(e, goodA); fault != "" {
			t.Errorf("payout event %+v (seed %d): %s", e, seed, fault)
		}
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if got, want := snap.Jurisdictions[0].Money, start+taxed; got != want {
		t.Errorf("treasury after %d ticks (seed %d) = %d, want %d = start %d + taxed %d: an in-kind payout moves no money", ticks, seed, got, want, start, taxed)
	}
}
