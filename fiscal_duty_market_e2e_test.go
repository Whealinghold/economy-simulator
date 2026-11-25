package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiscalMarketDoc is two duties of the root on the trades of the seat region: one of 100 bp
// on the value of what a household sells, one of 50 bp on the value of what it buys, both on
// every commodity. They stack: a trade is in the base of both.
const fiscalMarketDoc = `
[modules.fiscal.duty.sell]
authority = "root"
region = "centre"
payer = "seller"
rate_bp = 100

[modules.fiscal.duty.buy]
authority = "root"
region = "centre"
payer = "buyer"
rate_bp = 50
`

// dutyModel is what the duty events of one flow must say, rebuilt from the trades alone:
// the due of each tick and the count of trades whose payer side was a pseudo-agent.
type dutyModel struct {
	due    map[int64]economysim.Money
	exempt int64
}

// modelDuty rebuilds a duty of rateBP at the full scale from the trades of the region: the
// payer of a trade is the side payer picks, a payer below firstHousehold is exempt, and the
// due of a tick for a payer is floor(cum_t * rateBP / 10^4) - floor(cum_{t-1} * rateBP / 10^4)
// over the cumulative base cum of its trades, which is what the carry of FR-FIS-032
// makes of the per-tick truncation.
func modelDuty(trades []economysim.TradeExecuted, region economysim.RegionID, firstHousehold economysim.AgentID, rateBP int64, payer func(economysim.TradeExecuted) economysim.AgentID) dutyModel {
	m := dutyModel{due: map[int64]economysim.Money{}}
	cum, paid := map[economysim.AgentID]int64{}, map[economysim.AgentID]int64{}
	var last int64
	settle := func(tick int64) {
		for a, base := range cum {
			total := base * rateBP / 10_000
			m.due[tick] += economysim.Money(total - paid[a])
			paid[a] = total
		}
	}
	for _, tr := range trades {
		if tr.Region != region {
			continue
		}
		if tr.Tick != last {
			settle(last)
			last = tr.Tick
		}
		if p := payer(tr); p < firstHousehold {
			m.exempt++
		} else {
			cum[p] += int64(tr.Price) * int64(tr.Quantity)
		}
	}
	settle(last)
	return m
}

// dutyEvents splits the events of a run into the trades, what the duty events say of each
// flow, the money they applied and the lowest ID of an agent that spawned (the first household).
func dutyEvents(events []economysim.Event) (trades []economysim.TradeExecuted, got map[string]*dutyModel, applied economysim.Money, first economysim.AgentID) {
	got = map[string]*dutyModel{"sell": {due: map[int64]economysim.Money{}}, "buy": {due: map[int64]economysim.Money{}}}
	first = -1
	for _, e := range events {
		switch m := e.(type) {
		case economysim.TradeExecuted:
			trades = append(trades, m)
		case economysim.AgentSpawned:
			if first < 0 || m.Agent < first {
				first = m.Agent
			}
		case economysim.FiscalFlowApplied:
			if m.Kind == economysim.KindDuty {
				got[m.Flow].due[m.Tick] += m.Due
				got[m.Flow].exempt += m.Exempt
				applied += m.Applied
			}
		}
	}
	return trades, got, applied, first
}

// dutyFaults compares what the events said of one duty with the model, one message each:
// the Exempt count, the due of every tick the model has, and a due on a tick it has none.
func dutyFaults(got, want *dutyModel) (out []string) {
	if got.exempt != want.exempt {
		out = append(out, fmt.Sprintf("Exempt = %d, want %d", got.exempt, want.exempt))
	}
	for tick, d := range want.due {
		if got.due[tick] != d {
			out = append(out, fmt.Sprintf("Due at tick %d = %d, want %d", tick, got.due[tick], d))
		}
	}
	for tick, d := range got.due {
		if _, ok := want.due[tick]; !ok && d != 0 {
			out = append(out, fmt.Sprintf("Due at tick %d = %d, want none: no trade of the tick", tick, d))
		}
	}
	return out
}

// TestFiscal_PackDutyIsCollectedAtTheMarket verifies, end to end on the governed warm
// five with estates, that two stacked duties on the trades of the seat region collect
// exactly what their rule says: for each flow, the due of every tick is what the rule gives
// for the value of the trades of the tick of that payer plus the carry of the ticks before
// (rebuilt here from the TradeExecuted events alone, so the base, the rate and the carry
// across ticks are all checked), the trades whose payer is a pseudo-agent (an estate) are
// counted as Exempt and are in no base, and the root's treasury ends at its start money plus
// both duties applied, with the audit holding on every tick (a duty moves money and mints
// none).
//
// Arithmetic: the sell duty is 100 bp, so a seller whose trades total 250 over two ticks
// has paid floor(250 / 100) = 2, however the 250 splits; the buy duty is 50 bp on the
// buyer, floor(cum / 200).
//
// Requirement: FR-FIS-030, FR-FIS-031, FR-FIS-032, FR-FIS-033, FR-FIS-044
//
// Guard: the unit tests give the module the trades by hand; a settlement that handed the
// module the trades of one side only, or before a trade was cleared, or kept a
// pseudo-agent in a base, changes the dues here and in no unit test.
func TestFiscal_PackDutyIsCollectedAtTheMarket(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const seed, ticks = 1, 30
	layers := append(slices.Clone(fivePolityGoverned), "overlay/estates.toml")
	cfg := loadPackBytes(t, []byte(fiscalMarketDoc), layers...)
	rec := &recorder{}
	w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))
	s0, err := w.Snapshot()
	if err != nil || len(s0.Jurisdictions) == 0 {
		t.Fatalf("Snapshot() before the first step = %d treasuries, %v, want at least one", len(s0.Jurisdictions), err)
	}
	stepBoth(t, ticks, w)

	trades, got, applied, first := dutyEvents(rec.events)
	centre := economysim.RegionID(slices.IndexFunc(cfg.Regions, func(r economysim.Region) bool { return r.ID == "centre" }))
	want := map[string]dutyModel{
		"sell": modelDuty(trades, centre, first, 100, func(m economysim.TradeExecuted) economysim.AgentID { return m.Seller }),
		"buy":  modelDuty(trades, centre, first, 50, func(m economysim.TradeExecuted) economysim.AgentID { return m.Buyer }),
	}
	if want["sell"].exempt == 0 {
		t.Errorf("trades of a pseudo-agent in the model = 0 (seed %d), want some: the estates of the pack sell", seed)
	}
	for flow, wm := range want {
		for _, fault := range dutyFaults(got[flow], &wm) {
			t.Errorf("duty %s: %s (seed %d)", flow, fault, seed)
		}
	}
	if applied == 0 {
		t.Fatalf("duties applied over %d ticks (seed %d) = 0, want some", ticks, seed)
	}
	s1, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if got, want := s1.Jurisdictions[0].Money, s0.Jurisdictions[0].Money+applied; got != want {
		t.Errorf("treasury after %d ticks (seed %d) = %d, want %d = start %d + duties applied %d", ticks, seed, got, want, s0.Jurisdictions[0].Money, applied)
	}
}
