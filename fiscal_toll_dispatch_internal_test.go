package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The tests of the toll at dispatch (spec-fiscal.md 7.4, plan 3-fis Phase 7): the check
// of row 22a, step 3a, the total on the event, the aggregate of P3.6c, and the module off.
// They use tollWorld: a self-carried trip on foot from b_far to c_isle leaves inland
// (ferry 4) and enters the coast (pier 7), 11 for the one carrying unit it takes.

// tollShipper is the first household of b_far in tollWorld (inlandBorder has two
// treasuries, so the households start at agent 5, two in a_home first).
func tollShipper(w *World) AgentID { return AgentID(w.firstHousehold + 2) }

// footToIsle is the self-carried trip on foot from b_far to c_isle.
var footToIsle = ShipmentRequest{Mode: modeFoot, To: regIsle, ToSelf: true}

// TestFiscal_TollCharged verifies step 3a: the shipper pays each toll to the treasury of
// its authority, the total is on ShipmentDispatched, the money of the world is the same,
// and the toll is netted from the shipper's income of the tick.
//
// Arithmetic: ferry 4 + pier 7 = 11 for the one unit the trip takes. The shipper falls
// from 100 to 89, inland rises from 1 000 to 1 004 and the coast from 1 000 to 1 007.
//
// Requirement: FR-FIS-036, FR-FIS-002, FR-FIS-041
func TestFiscal_TollCharged(t *testing.T) {
	t.Parallel()
	w, rec := tollWorld(t, inlandBorder, ferryAndGate()...)
	shipper := tollShipper(w)
	total := moneyTotal(w)
	if _, reason := ask(w, shipper, 0, footToIsle); reason != "" {
		t.Fatalf("dispatch() reason = %q, want none", reason)
	}
	if got := heldMoney(t, w, shipper); got != 89 {
		t.Errorf("shipper holds %d, want 89", got)
	}
	if coast, inland := treasuryMoney(t, w, 0), treasuryMoney(t, w, 1); coast != 1_007 || inland != 1_004 {
		t.Errorf("coast and inland hold %d and %d, want 1007 and 1004", coast, inland)
	}
	if got := moneyTotal(w); got != total {
		t.Errorf("money after the dispatch = %d, want %d (a toll creates and destroys none)", got, total)
	}
	if got := w.welfare.at(int(shipper)).unearned; got != -11 {
		t.Errorf("unearned of the shipper = %d, want -11", got)
	}
	if d := of[ShipmentDispatched](rec); len(d) != 1 || d[0].Toll != 11 {
		t.Errorf("ShipmentDispatched = %+v, want one with Toll 11", d)
	}
	auditNow(t, w)
}

// TestFiscal_TollRefused verifies row 22a: a shipper whose available money is below the
// total toll is refused whole with toll_unaffordable and nothing changes, and one that
// holds exactly the total is not refused.
//
// Arithmetic: the total is 11. A shipper holding 10, or 100 of which 90 is reserved
// (available 10), is refused; one holding 11 pays all of it and ends at 0.
//
// Requirement: FR-FIS-035
func TestFiscal_TollRefused(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		money    int64
		reserve  int64
		wantErr  Reason
		wantLeft int64
	}{
		{"holding 10", 10, 0, ReasonTollUnaffordable, 10},
		{"holding 100 with 90 reserved", 100, 90, ReasonTollUnaffordable, 100},
		{"holding exactly the total", 11, 0, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := tollWorld(t, inlandBorder, ferryAndGate()...)
			shipper := tollShipper(w)
			setBalances(t, w, map[AgentID]int64{shipper: tt.money})
			if tt.reserve > 0 {
				if err := w.ledger.ReserveMoney(w.agents[shipper].account, tt.reserve); err != nil {
					t.Fatalf("ReserveMoney() error = %v, want nil", err)
				}
			}
			rec.events = nil
			inland := treasuryMoney(t, w, 1)
			_, reason := ask(w, shipper, 0, footToIsle)
			if reason != tt.wantErr {
				t.Fatalf("dispatch() reason = %q, want %q", reason, tt.wantErr)
			}
			if got := heldMoney(t, w, shipper); got != tt.wantLeft {
				t.Errorf("shipper holds %d, want %d", got, tt.wantLeft)
			}
			if tt.wantErr != "" {
				wantUntouched(t, w, rec, shipper, inland)
			}
		})
	}
}

// wantUntouched checks that a refused toll left the world as it was: the inland
// treasury at inland, no shipment, the shipper at home, one ShipmentRejected with the
// reason toll_unaffordable and no toll event at P3.6c.
func wantUntouched(t *testing.T, w *World, rec *evLog, shipper AgentID, inland int64) {
	t.Helper()
	if got := treasuryMoney(t, w, 1); got != inland || len(w.transport.ships) != 0 || w.transport.isAway(shipper) {
		t.Errorf("a refused shipment changed the world: inland %d (was %d), %d shipments, away %v", got, inland, len(w.transport.ships), w.transport.isAway(shipper))
	}
	if r := of[ShipmentRejected](rec); len(r) != 1 || r[0].Reason != ReasonTollUnaffordable {
		t.Errorf("ShipmentRejected = %+v, want one with toll_unaffordable", r)
	}
	w.p36cTolls()
	if got := of[FiscalFlowApplied](rec); len(got) != 0 {
		t.Errorf("toll events after a refusal = %+v, want none", got)
	}
}

// TestFiscal_Events_P36c verifies the aggregate of P3.6c: one FiscalFlowApplied for each
// toll flow that charged, in ascending flow ID, with the shipper's region and the
// authority paid, the counters cleared after it, and nothing for a toll that did not act.
//
// Arithmetic: two trips from b_far: ferry 4 + pier 7 each, so ferry is due 8 and pier 14
// over two shippers (Parties 2); gate acts on no edge of b_far to c_isle and is not
// reported.
//
// Requirement: FR-FIS-044, FR-FIS-036
func TestFiscal_Events_P36c(t *testing.T) {
	t.Parallel()
	w, rec := tollWorld(t, inlandBorder, ferryAndGate()...)
	first := tollShipper(w)
	for _, id := range []AgentID{first, first + 1} {
		if _, reason := ask(w, id, 0, footToIsle); reason != "" {
			t.Fatalf("dispatch of agent %d reason = %q, want none", id, reason)
		}
	}
	w.p36cTolls()
	want := []FiscalFlowApplied{
		{Flow: "ferry", Kind: KindToll, Authority: 1, Region: regFar, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 8, Applied: 8, Parties: 2},
		{Flow: "pier", Kind: KindToll, Authority: 0, Region: regFar, Class: FiscalHouseholds, Commodity: NoCommodity, Due: 14, Applied: 14, Parties: 2},
	}
	if got := of[FiscalFlowApplied](rec); !slices.Equal(got, want) {
		t.Errorf("events = %+v, want %+v", got, want)
	}
	rec.events = nil
	w.p36cTolls()
	if got := of[FiscalFlowApplied](rec); len(got) != 0 {
		t.Errorf("events at the next P3.6c = %+v, want none (the counters are cleared)", got)
	}
}

// TestFiscal_TollOffUnchanged verifies that with no toll configured, or with the module
// off, a dispatch charges nothing and ShipmentDispatched carries a Toll of 0, and that a
// toll at scale 0 charges nothing either.
//
// Requirement: FR-FIS-001, FR-FIS-044
func TestFiscal_TollOffUnchanged(t *testing.T) {
	t.Parallel()
	off, rec := newShipWorld(t, nil, nil)
	if _, reason := ask(off, shipHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}); reason != "" {
		t.Fatalf("dispatch() with the module off reason = %q, want none", reason)
	}
	if d := of[ShipmentDispatched](rec); len(d) != 1 || d[0].Toll != 0 {
		t.Errorf("ShipmentDispatched with the module off = %+v, want one with Toll 0", d)
	}
	w, rec := tollWorld(t, inlandBorder, ferryAndGate()...)
	for i := range w.fiscal.flows {
		if err := w.setFiscalScale(i, 0, CauseConsumer); err != nil {
			t.Fatalf("setFiscalScale(%d) error = %v, want nil", i, err)
		}
	}
	shipper := tollShipper(w)
	if _, reason := ask(w, shipper, 0, footToIsle); reason != "" {
		t.Fatalf("dispatch() at scale 0 reason = %q, want none", reason)
	}
	if got := heldMoney(t, w, shipper); got != 100 {
		t.Errorf("shipper holds %d at scale 0, want 100", got)
	}
	if d := of[ShipmentDispatched](rec); len(d) != 1 || d[0].Toll != 0 {
		t.Errorf("ShipmentDispatched at scale 0 = %+v, want one with Toll 0", d)
	}
}

// TestFiscal_TollAllocs verifies that the check, the charge and P3.6c allocate nothing
// once the scratch has grown (NFR-FIS-001), with no observer and the default logger.
//
// Requirement: NFR-FIS-001
func TestFiscal_TollAllocs(t *testing.T) {
	w, _ := tollWorld(t, inlandBorder, ferryAndGate()...)
	w.obs.supplied = false
	c := &w.transport.ctx
	*c = shipCtx{w: w, a: &w.agents[tollShipper(w)], r: &footToIsle, t: w.transport, region: regFar, n: 1}
	run := func() {
		c.toll = 0
		if reason := ckToll(c); reason != "" {
			t.Fatalf("ckToll() = %q, want none", reason)
		}
		w.chargeToll(c)
		w.p36cTolls()
	}
	run()
	if n := testing.AllocsPerRun(5, run); n != 0 {
		t.Errorf("AllocsPerRun(toll) = %v, want 0", n)
	}
}

// TestFiscal_TollTreasuryFull verifies that a shipment whose toll a treasury cannot take
// without passing MaxMoney is refused whole as overflow, with nothing changed, and that
// a toll that fills the treasury exactly is charged.
//
// Arithmetic: the trip pays inland the ferry 4 and the coast the pier 7. With inland
// at MaxMoney - 3 the headroom is 3, below 4: refused. At MaxMoney - 4 the headroom is
// 4: the ferry fits and inland ends at MaxMoney.
//
// Guard: the transfer of step 3a would be refused by the ledger, which send treats as
// a violated invariant (a panic); the check of row 22a has to refuse first.
func TestFiscal_TollTreasuryFull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		headroom int64
		want     Reason
	}{
		{"headroom below the toll", 3, ReasonOverflow},
		{"headroom equal to the toll", 4, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := tollWorld(t, inlandBorder, ferryAndGate()...)
			shipper := tollShipper(w)
			boundary, ok := w.ledger.Boundary(ledger.Initial, ledger.Money)
			if !ok {
				t.Fatal("Boundary(initial, money) is not open")
			}
			if err := w.ledger.Transfer(boundary, w.polity.tr[1].account, MaxMoney-tt.headroom-treasuryMoney(t, w, 1), ledger.Initial); err != nil {
				t.Fatalf("Transfer(initial) error = %v, want nil", err)
			}
			rec.events = nil
			_, reason := ask(w, shipper, 0, footToIsle)
			if reason != tt.want {
				t.Fatalf("dispatch() reason = %q, want %q", reason, tt.want)
			}
			wantInland, wantShipper := int64(MaxMoney)-tt.headroom, int64(100)
			if tt.want == "" {
				wantInland, wantShipper = MaxMoney, 89
			}
			if got := treasuryMoney(t, w, 1); got != wantInland {
				t.Errorf("inland holds %d, want %d", got, wantInland)
			}
			if got := heldMoney(t, w, shipper); got != wantShipper {
				t.Errorf("shipper holds %d, want %d", got, wantShipper)
			}
			if r := of[ShipmentRejected](rec); (tt.want != "") != (len(r) == 1 && r[0].Reason == tt.want) {
				t.Errorf("ShipmentRejected = %+v, want one with %q only for a refusal", r, tt.want)
			}
		})
	}
}
