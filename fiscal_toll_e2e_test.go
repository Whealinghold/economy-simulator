package economysim_test

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiscalTollDoc governs the two regions of pack two by one jurisdiction each, "alpha"
// (north) and "beta" (south), and gives beta a toll of 3 a carrying unit on every shipment
// that enters it. The jurisdictions sort alpha then beta, so row 0 of a snapshot is alpha.
func fiscalTollDoc(amount int64) string {
	return fmt.Sprintf(`
[modules.polity.jurisdiction.alpha]
seat = "north"
start_money = 5000

[modules.polity.jurisdiction.beta]
seat = "south"
start_money = 5000

[modules.polity.member]
north = "alpha"
south = "beta"

[modules.fiscal.toll.gate]
authority = "beta"
on = "enter"
amount = %d
`, amount)
}

// fiscalAllFlowsDoc adds to the toll a sales duty of beta on the seller, and a payout of
// alpha to the carriers of the north, so that one run moves money by a duty, a toll and a
// payout together.
var fiscalAllFlowsDoc = fiscalTollDoc(3) + `
[modules.fiscal.duty.sales]
authority = "beta"
region = "south"
payer = "seller"
rate_bp = 100

[modules.fiscal.payout.alms]
authority = "alpha"
target = { kind = "households", region = "north", role = "carrier" }
amount = 2
`

// tollPack loads pack two with its transport overlay and the document, and puts the scripted
// shipper on it: every carrier of a region sends a unit of staple on foot to the other one.
func tollPack(t testing.TB, doc string) (economysim.Config, []economysim.Option) {
	t.Helper()
	return shipperOnConfig(t, loadPackBytes(t, []byte(doc), packTwoTransport...))
}

// tollRun steps a world of the toll pack for ticks under the audit and returns it, the events
// it saw and the money of the treasuries before the first step, alpha first.
func tollRun(t *testing.T, doc string, seed uint64, ticks int) (*economysim.World, []economysim.Event, []economysim.Money) {
	t.Helper()
	cfg, opts := tollPack(t, doc)
	rec := &recorder{}
	w := mustWorld(t, cfg, append(opts, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))...)
	start := treasuryMoneys(t, w)
	stepBoth(t, ticks, w)
	return w, rec.events, start
}

// treasuryMoneys is the money of each treasury of w, in ascending jurisdiction ID.
func treasuryMoneys(t *testing.T, w *economysim.World) []economysim.Money {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	var out []economysim.Money
	for _, j := range s.Jurisdictions {
		out = append(out, j.Money)
	}
	return out
}

// TestFiscal_PackTollIsChargedAtDispatch verifies, end to end on pack two with the scripted
// shipper, that a toll on entering the south is charged at the dispatch of each shipment that
// enters it and of no other: every ShipmentDispatched north to south reports a Toll of 3
// for each carrying unit, every one south to north reports 0, the FiscalFlowApplied events
// of the toll add up to the tolls the dispatches reported, and the treasury of the toll's
// authority ends at its start money plus them while the other treasury has not moved.
//
// Arithmetic: the toll is 3 a carrying unit at the full scale, so a shipment of Units
// units pays 3 * Units. The audit, on every tick, is what proves that the toll moved money
// from the shipper to the treasury and created none.
//
// Requirement: FR-FIS-034, FR-FIS-036, FR-FIS-044
//
// Guard: the unit tests build the shipment by hand; a toll that was charged on a path
// other than the one the dispatch takes, or reported but not paid, shows only when the
// strategy asks for the shipment.
func TestFiscal_PackTollIsChargedAtDispatch(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const seed, ticks = 1, 40
	w, events, start := tollRun(t, fiscalTollDoc(3), seed, ticks)
	var paid, reported economysim.Money
	var entered int
	for _, e := range events {
		switch m := e.(type) {
		case economysim.ShipmentDispatched:
			want := economysim.Money(0)
			if m.To == 1 { // south: the shipment enters beta
				want, entered = economysim.Money(3*m.Units), entered+1
			}
			if m.Toll != want {
				t.Errorf("ShipmentDispatched to region %d of %d units has Toll %d, want %d (seed %d)", m.To, m.Units, m.Toll, want, seed)
			}
			paid += m.Toll
		case economysim.FiscalFlowApplied:
			if m.Kind == economysim.KindToll {
				reported += m.Applied
			}
		}
	}
	if entered == 0 || paid == 0 {
		t.Fatalf("shipments into the south = %d and tolls paid = %d over %d ticks (seed %d), want both positive", entered, paid, ticks, seed)
	}
	if reported != paid {
		t.Errorf("toll events add up to %d, dispatches reported %d (seed %d), want equal", reported, paid, seed)
	}
	end := treasuryMoneys(t, w)
	if want := []economysim.Money{start[0], start[1] + paid}; !slices.Equal(end, want) {
		t.Errorf("treasuries after %d ticks (seed %d) = %v, want %v = start %v plus the tolls %d to beta", ticks, seed, end, want, start, paid)
	}
}

// tollRefusals checks the events of a run in which every shipment into the south owes an
// unaffordable toll and returns the count of refusals and of dispatches the other way.
func tollRefusals(t *testing.T, events []economysim.Event, seed uint64) (rejected, back int) {
	t.Helper()
	for _, e := range events {
		switch m := e.(type) {
		case economysim.ShipmentRejected:
			rejected++
			if m.Region != 0 || m.To != 1 || m.Reason != economysim.ReasonTollUnaffordable {
				t.Errorf("ShipmentRejected %+v (seed %d), want only north to south with toll_unaffordable", m, seed)
			}
		case economysim.ShipmentDispatched:
			back++
			if m.To == 1 || m.Toll != 0 {
				t.Errorf("ShipmentDispatched %+v (seed %d), want none into the south and no toll", m, seed)
			}
		}
	}
	return rejected, back
}

// tollMoved is the money the toll events of a run report as applied.
func tollMoved(events []economysim.Event) (sum economysim.Money) {
	for _, e := range events {
		if f, ok := e.(economysim.FiscalFlowApplied); ok && f.Kind == economysim.KindToll {
			sum += f.Applied
		}
	}
	return sum
}

// TestFiscal_PackTollRefusedWholeWhenUnaffordable verifies, end to end, that a toll no
// shipper can pay refuses every shipment into the region it guards whole: each is a
// ShipmentRejected from the north to the south with the reason toll_unaffordable, none is
// dispatched there, no toll event reports a charge, the treasuries are at their start
// money, and the shipments the other way, which owe nothing, go on.
//
// Arithmetic: the toll is 1 000 000 000 a carrying unit, far above what a household of the
// pack holds, so the total of any trip exceeds the shipper's available money.
//
// Requirement: FR-FIS-035
//
// Guard: a refusal that charged a part of the toll before it refused would leave the
// treasury above its start money, and one that refused only the toll would still ship.
func TestFiscal_PackTollRefusedWholeWhenUnaffordable(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const seed, ticks = 1, 40
	w, events, start := tollRun(t, fiscalTollDoc(1_000_000_000), seed, ticks)
	rejected, back := tollRefusals(t, events, seed)
	if moved := tollMoved(events); moved != 0 {
		t.Errorf("toll events report %d applied (seed %d), want 0: a refusal charges nothing", moved, seed)
	}
	if rejected == 0 || back == 0 {
		t.Errorf("rejections = %d and dispatches to the north = %d over %d ticks (seed %d), want both positive", rejected, back, ticks, seed)
	}
	if end := treasuryMoneys(t, w); !slices.Equal(end, start) {
		t.Errorf("treasuries after %d ticks (seed %d) = %v, want %v: a refused shipment pays nothing", ticks, seed, end, start)
	}
}

// routeProbe is the scripted shipper that also keeps the Toll of every route row it was
// shown, by the region of the agent, the destination and the mode.
type routeProbe struct {
	shipper
	tolls map[[3]int]economysim.Money
}

func (p routeProbe) Trade(v *economysim.AgentView, rng *rand.Rand, plan *economysim.TradePlan) {
	for _, r := range v.Routes {
		if r.Has {
			p.tolls[[3]int{int(v.Region), int(r.To), int(r.Mode)}] = economysim.Money(r.Toll)
		}
	}
	p.shipper.Trade(v, rng, plan)
}

// TestFiscal_PackRouteRowShowsTheToll verifies, through the view a strategy reads, that
// the route rows into the guarded region carry the toll of one carrying unit, for each
// mode (the toll names none), and the rows the other way carry 0: the number a merchant
// subtracts from its haul ceiling (strategy/merchant_toll_test.go is the unit test of
// the subtraction).
//
// Arithmetic: one carrying unit pays the toll's 3 once, on the one edge that enters beta.
//
// Requirement: FR-FIS-037
//
// Guard: a row filled from the wrong end of the edge would show the toll on the way out and
// leave a merchant paying a toll its margin never counted.
func TestFiscal_PackRouteRowShowsTheToll(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, s := shipperFor(t, loadPackBytes(t, []byte(fiscalTollDoc(3)), packTwoTransport...))
	p := routeProbe{shipper: s, tolls: map[[3]int]economysim.Money{}}
	w := mustWorld(t, cfg, economysim.WithStrategy("ship", p), economysim.WithSeed(1), economysim.WithAudit())
	stepBoth(t, 3, w)
	for key, got := range p.tolls {
		want := economysim.Money(0)
		if key[1] == 1 && key[0] == 0 {
			want = 3
		}
		if got != want {
			t.Errorf("route row from region %d to %d by mode %d shows Toll %d, want %d", key[0], key[1], key[2], got, want)
		}
	}
	if len(p.tolls) == 0 || p.tolls[[3]int{0, 1, int(modeFootID)}] != 3 {
		t.Errorf("route rows seen = %v, want the foot row north to south with Toll 3", p.tolls)
	}
}

// moneyFlows is the money, not goods, that the duty, toll and payout events of a run report applied.
func moneyFlows(events []economysim.Event) (duty, toll, payout economysim.Money) {
	for _, e := range events {
		f, ok := e.(economysim.FiscalFlowApplied)
		if !ok || f.Commodity != economysim.NoCommodity {
			continue
		}
		switch f.Kind {
		case economysim.KindDuty:
			duty += f.Applied
		case economysim.KindToll:
			toll += f.Applied
		case economysim.KindPayout:
			payout += f.Applied
		}
	}
	return duty, toll, payout
}

// TestFiscal_PackConservationOverSeeds verifies, over the seeds of a scenario, that a
// duty, a toll and a payout together conserve money exactly: on every tick the audit
// holds (a flow moves money between sectors and mints none), and at the end each
// treasury holds its start money plus what flowed into it less what flowed out, rebuilt
// from the events alone, with no tolerance.
//
// Arithmetic: beta ends at start + the duty applied + the tolls applied, alpha at start -
// the payouts applied (the duty and the toll name beta, the payout alpha).
//
// Requirement: FR-FIS-002, FR-FIS-044, FR-FIS-036, FR-FIS-032
//
// Guard: each flow is conserved on its own in the unit tests; the sum is the one a
// consumer's books must match, and a flow that paid one treasury and reported another
// would pass each of them.
func TestFiscal_PackConservationOverSeeds(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
		w, events, start := tollRun(t, fiscalAllFlowsDoc, seed, 40)
		duty, toll, payout := moneyFlows(events)
		if duty == 0 || toll == 0 || payout == 0 {
			t.Errorf("duty = %d, toll = %d, payout = %d applied over 40 ticks (seed %d), want all positive", duty, toll, payout, seed)
		}
		want := []economysim.Money{start[0] - payout, start[1] + duty + toll}
		if got := treasuryMoneys(t, w); !slices.Equal(got, want) {
			t.Errorf("treasuries (seed %d) = %v, want %v = start %v, duty %d and toll %d into beta, payout %d out of alpha", seed, got, want, start, duty, toll, payout)
		}
	}
}

// TestFiscal_PackDutyTollPayoutDeterminism verifies that two runs of the pack with a duty,
// a toll and a payout give the same events and the same snapshot, and that a run with an
// observer attached ends in the snapshot of one without.
//
// Requirement: NFR-FIS-004, NFR-FIS-005
//
// Guard: the toll path walk and the duty bases are built from scratch in each tick; an
// order that came from a map would differ on some runs only.
func TestFiscal_PackDutyTollPayoutDeterminism(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const seed, ticks = 2, 30
	w1, ev1, _ := tollRun(t, fiscalAllFlowsDoc, seed, ticks)
	w2, ev2, _ := tollRun(t, fiscalAllFlowsDoc, seed, ticks)
	cfg, opts := tollPack(t, fiscalAllFlowsDoc)
	bare := mustWorld(t, cfg, append(opts, economysim.WithSeed(seed), economysim.WithAudit())...)
	stepBoth(t, ticks, bare)
	if !reflect.DeepEqual(ev1, ev2) {
		t.Errorf("events of two runs of seed %d differ (%d and %d), want equal", seed, len(ev1), len(ev2))
	}
	s1, _ := w1.Snapshot()
	s2, _ := w2.Snapshot()
	sb, _ := bare.Snapshot()
	if !reflect.DeepEqual(s1, s2) || !reflect.DeepEqual(s1, sb) {
		t.Errorf("snapshots of two observed runs and one bare run of seed %d differ, want equal", seed)
	}
}
