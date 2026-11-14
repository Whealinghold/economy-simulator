package economysim

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The world of the trade tests is extConfig (the households are agents 4 to 10, traders
// of lowland, region 1; the counterparties are agents 2 and 3) with the storage module:
// every trader has a base room of (20, 20), and bread (the upkeep), grain and salt each
// have the size (1, 1), so a unit of any is a load of (1, 1). Orders are submitted
// directly into the books, after the books of the tick are cleared as P3.2 does, and
// the markets are cleared by P4.1.
const (
	stoBuyer  AgentID = 4
	stoSeller AgentID = 5
)

// stoTradeWorld builds that world with the observer log and the audit on.
func stoTradeWorld(t *testing.T) (*World, *evLog) {
	t.Helper()
	c := extConfig()
	c.Modules.Storage = &Storage{Base: &Capacity{Mass: 20, Bulk: 20}}
	w, _, rec := extWorld(t, c)
	w.receipts.reset()
	w.markets.reset(w.Tick())
	return w, rec
}

// bid submits a bid of agent id for qty units of salt at limit.
func stoBid(w *World, id AgentID, qty Quantity, limit Price) {
	w.submit(&w.agents[id], regLowland, Intent{Commodity: 2, Side: Bid, Quantity: qty, Limit: limit})
}

// TestAdmit_NoRoomAfterInsufficientStock verifies the row no_room of admission: a bid of a
// limited account whose load exceeds its free room is rejected with the reason, an ask
// is never tested, and the row is the last, so a bid that fails the funds reports them.
//
// Arithmetic: a trader holds nothing of salt and has a room of (20, 20); 21 salt are
// (21, 21), above it; 20 fit. Another household has 11 000 of cash and bids 21 at 600,
// which is 12 600: insufficient_funds, though 21 units would not fit either.
//
// Requirement: FR-STO-008
func TestAdmit_NoRoomAfterInsufficientStock(t *testing.T) {
	t.Parallel()
	w, rec := stoTradeWorld(t)
	fund(t, w, stoBuyer, 100_000)
	stoBid(w, stoBuyer, 21, 50)
	fund(t, w, stoSeller, 1_000)
	stoBid(w, stoSeller, 21, 600)                                                                // 12 600 of cash over the 10 000 the household starts with and the 1 000 given
	w.submit(&w.agents[6], regLowland, Intent{Commodity: 2, Side: Ask, Quantity: 21, Limit: 40}) // no stock: insufficient_stock
	give(t, w, 7, 2, 20)
	w.submit(&w.agents[7], regLowland, Intent{Commodity: 2, Side: Ask, Quantity: 20, Limit: 40}) // a full room: not tested
	var got []Reason
	for _, e := range of[OrderRejected](rec) {
		got = append(got, e.Reason)
	}
	if want := []Reason{ReasonNoRoom, ReasonInsufficientFunds, ReasonInsufficientStock}; !reflect.DeepEqual(got, want) {
		t.Errorf("reasons of the rejected orders = %v, want %v", got, want)
	}
	if n := len(of[OrderPlaced](rec)); n != 1 {
		t.Errorf("orders placed = %d, want 1 (the ask of the full account)", n)
	}
	if w.sto.counts[w.cellOf(regLowland, &w.agents[stoBuyer])].bidsRejected != 1 {
		t.Errorf("bidsRejected = %d, want 1", w.sto.counts[w.cellOf(regLowland, &w.agents[stoBuyer])].bidsRejected)
	}
}

// TestAdmit_PseudoAgentExempt verifies that the estate, which has no room, asks and is
// admitted whatever the sizes (its ask is never tested), in the same tick as a bid of a
// limited account for what it sells.
//
// Requirement: FR-STO-008
func TestAdmit_PseudoAgentExempt(t *testing.T) {
	t.Parallel()
	w, rec := stoTradeWorld(t)
	give(t, w, stoSeller, 2, 15)
	if _, err := w.estates.acceptGoods(w.ledger, w.agents[stoSeller].account, regLowland, ledger.EstateGoods, nil); err != nil {
		t.Fatalf("acceptGoods() = %v, want nil", err)
	}
	if err := w.p34Estates(); err != nil {
		t.Fatalf("p34Estates() = %v, want nil", err)
	}
	if got := len(of[OrderPlaced](rec)); got == 0 {
		t.Errorf("orders placed after the estate's asks = %d, want its lot listed", got)
	}
	if got := of[OrderRejected](rec); len(got) != 0 {
		t.Errorf("rejected orders = %+v, want none: a pseudo-agent has no room to test", got)
	}
}

// TestBid_ReservesRoomWithCash verifies that an admitted bid reserves its room with its
// cash (FR-STO-009) and that the reservation is released at P4.1 with the money of the
// part that did not trade.
//
// Arithmetic: 5 salt of (1, 1) are (5, 5) of room; at limit 30 they hold 150 of cash; no
// ask matches (the counterparties ask 42 and more), so all of both is released.
//
// Requirement: FR-STO-009
func TestBid_ReservesRoomWithCash(t *testing.T) {
	t.Parallel()
	w, _ := stoTradeWorld(t)
	fund(t, w, stoBuyer, 10_000)
	stoBid(w, stoBuyer, 5, 30)
	rm, err := w.ledger.RoomOf(w.agents[stoBuyer].account)
	if err != nil || rm.ReservedMass != 5 || rm.ReservedBulk != 5 {
		t.Fatalf("room after the bid = %+v, %v, want (5, 5) reserved", rm, err)
	}
	if v, _ := w.ledger.Account(w.agents[stoBuyer].account); v.Reserved != 150 {
		t.Errorf("money reserved = %d, want 150", v.Reserved)
	}
	if err := w.p41Clear(); err != nil {
		t.Fatalf("p41Clear() = %v, want nil", err)
	}
	rm, _ = w.ledger.RoomOf(w.agents[stoBuyer].account)
	if rm.ReservedMass != 0 || rm.ReservedBulk != 0 || rm.LoadMass != 0 {
		t.Errorf("room after the market cleared with no trade = %+v, want nothing reserved and nothing loaded", rm)
	}
}

// TestSettlement_DrawsReservedNeverFails verifies that the goods of a trade are
// credited from the buyer's reserved room, in an ordinary ask and in the named lot of
// an estate, that the unmatched part is released, and that the audit holds after.
//
// Arithmetic: the buyer bids 8 salt at 60 for (8, 8) of room; the seller asks 5 at 40: 5
// are credited from the reserved room, and the other 3 are released: load (5, 5),
// reserved (0, 0). The estate case does the same with a lot of 4.
//
// Requirement: FR-STO-009, FR-STO-016
func TestSettlement_DrawsReservedNeverFails(t *testing.T) {
	t.Parallel()
	for _, estate := range []bool{false, true} {
		name := map[bool]string{false: "an ordinary ask", true: "the lot of an estate"}[estate]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w, rec := stoTradeWorld(t)
			fund(t, w, stoBuyer, 10_000)
			give(t, w, stoSeller, 2, 5)
			if estate {
				if _, err := w.estates.acceptGoods(w.ledger, w.agents[stoSeller].account, regLowland, ledger.EstateGoods, nil); err != nil {
					t.Fatalf("acceptGoods() = %v, want nil", err)
				}
				if err := w.p34Estates(); err != nil {
					t.Fatalf("p34Estates() = %v, want nil", err)
				}
			} else {
				w.submit(&w.agents[stoSeller], regLowland, Intent{Commodity: 2, Side: Ask, Quantity: 5, Limit: 40})
			}
			stoBid(w, stoBuyer, 8, 60)
			if err := w.p41Clear(); err != nil {
				t.Fatalf("p41Clear() = %v, want nil: settlement of an admitted bid cannot fail for room", err)
			}
			if got := len(of[TradeExecuted](rec)); got == 0 {
				t.Fatalf("trades = %d, want the ask matched", got)
			}
			rm, _ := w.ledger.RoomOf(w.agents[stoBuyer].account)
			held, _, _ := w.ledger.Holding(w.agents[stoBuyer].account, 2)
			if rm.ReservedMass != 0 || rm.ReservedBulk != 0 || rm.LoadMass != held || rm.LoadBulk != held {
				t.Errorf("room after settlement = %+v with %d salt held, want nothing reserved and the load of the units held", rm, held)
			}
			if err := w.ledger.Audit(); err != nil {
				t.Errorf("ledger Audit() = %v, want nil", err)
			}
		})
	}
}

// TestReserved_ZeroAtEveryBoundary verifies FR-STO-010: over steps in which the traders
// bid below every ask, so that no bid trades and every reservation is released, the
// audit, which asks that the reserved room be 0 at a boundary, passes each tick.
//
// Requirement: FR-STO-010
func TestReserved_ZeroAtEveryBoundary(t *testing.T) {
	t.Parallel()
	c := extConfig()
	c.Modules.Storage = &Storage{Base: &Capacity{Mass: 20, Bulk: 20}}
	w, s, _ := extWorld(t, c)
	for id := AgentID(4); id < 11; id++ {
		fund(t, w, id, 10_000)
		s.plan[id] = []Intent{{Commodity: 2, Side: Bid, Quantity: 4, Limit: 30}}
	}
	for range 3 {
		step(t, w, 1)
		for id := AgentID(4); id < 11; id++ {
			if rm, _ := w.ledger.RoomOf(w.agents[id].account); rm.ReservedMass != 0 || rm.ReservedBulk != 0 {
				t.Fatalf("agent %d at tick %d: reserved room (%d, %d), want 0 at a boundary", id, w.Tick(), rm.ReservedMass, rm.ReservedBulk)
			}
		}
	}
}

// The transport tests use stoWorld (shipConfig with the storage module): the coach is a
// hired mode (haul units of commodity haul, one carrying unit of (1 200, 900) for 4 haul
// units over the 40 distance), the foot a self-carried one.

// stoShip dispatches the hired request of agent id as its request number idx, giving it
// the 4 haul units one carrying unit needs (the coach takes 3 ticks to reach far: 40 over
// a speed of 30 is 2).
func stoShip(t *testing.T, w *World, from, to AgentID, grains Quantity) (ShipmentID, Reason) {
	t.Helper()
	give(t, w, from, commodityOf(t, w, "haul"), 4)
	return ask(w, from, 0, ShipmentRequest{Mode: stoCoach, To: regFar, Consignee: to, Lines: []Line{{Commodity: commodityOf(t, w, "grain"), Quantity: grains}}})
}

// TestShip_ConsigneeFullAfterElsewhere verifies row 11a: a consignee whose free room less
// its upkeep room is below the load of the lines is refused consignee_full, after
// consignee_elsewhere (a consignee in the shipper's own region is that reason first),
// and counted in the cell of the shipper.
//
// Arithmetic: the consignee holds 10 grain (30, 20) in a room of (60, 40): free (30, 20),
// less its upkeep room (3, 2): (27, 18). 5 grain are (15, 10), within it; 10 grain are
// (30, 20), above it in both dimensions.
//
// Requirement: FR-STO-014, FR-STO-046
func TestShip_ConsigneeFullAfterElsewhere(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	if _, reason := stoShip(t, w, shipHome, shipFar, 10); reason != ReasonConsigneeFull {
		t.Errorf("shipment of 10 grain to a consignee with room for 9 = %q, want consignee_full", reason)
	}
	if got := w.sto.counts[w.cellOf(regHome, &w.agents[shipHome])].shipmentsRefused; got != 1 {
		t.Errorf("shipmentsRefused of the shipper's cell = %d, want 1", got)
	}
	if _, reason := stoShip(t, w, shipHome, shipHome2, 5); reason != ReasonConsigneeElsewhere {
		t.Errorf("shipment to a consignee in the shipper's region = %q, want consignee_elsewhere before consignee_full", reason)
	}
	if _, reason := stoShip(t, w, shipHome, shipFar, 9); reason != "" {
		t.Errorf("shipment of 9 grain = %q, want it accepted: (27, 18) is within (27, 18)", reason)
	}
}

// TestShip_PledgesOnDispatch_ArrivalCreditsFromPledge verifies that a dispatched
// shipment pledges the load of its lines on the consignee, that the arrival credits the
// lines from the pledge and lowers it by the whole line, and that the audit's identity (the
// pledge is the load of the lines in flight) holds throughout.
//
// Arithmetic: 5 grain are (15, 10); the coach takes 2 ticks.
//
// Requirement: FR-STO-014, FR-STO-015
func TestShip_PledgesOnDispatch_ArrivalCreditsFromPledge(t *testing.T) {
	t.Parallel()
	w, rec := stoWorld(t, nil)
	if _, reason := stoShip(t, w, shipHome, shipFar, 5); reason != "" {
		t.Fatalf("dispatch = %q, want accepted", reason)
	}
	if st := roomOfAgent(t, w, shipFar); st.Pledge != (Capacity{Mass: 15, Bulk: 10}) {
		t.Errorf("pledge of the consignee after the dispatch = %+v, want (15, 10)", st.Pledge)
	}
	if err := w.Audit(); err != nil {
		t.Fatalf("Audit() after the dispatch = %v, want nil", err)
	}
	for range 3 {
		step(t, w, 1)
	}
	arr := of[ShipmentArrived](rec)
	if len(arr) != 1 || arr[0].Diverted || len(arr[0].Lines) != 1 || arr[0].Lines[0].Delivered != 5 {
		t.Fatalf("ShipmentArrived = %+v, want one arrival with 5 grain delivered, not diverted", arr)
	}
	if st := roomOfAgent(t, w, shipFar); st.Pledge != (Capacity{}) {
		t.Errorf("pledge of the consignee after the arrival = %+v, want (0, 0)", st.Pledge)
	}
}

// TestArrival_DivertsWhatDoesNotFit verifies the backstop of 5.4.4: a cut of the
// consignee's room while the cargo travels (a forced removal of its store) leaves the
// room left below the line, the part that fits is credited, whole units, the rest goes
// to the estate of the destination, and the pledge falls by the whole line.
//
// Arithmetic: the consignee has a barn (100, 120) over its (60, 40) and holds 18 grain
// (54, 36) after the fixture gives it 8; 5 grain (15, 10) are dispatched (its free room
// (106, 124) holds them). The barn is removed (forced; the load (54, 36) fits the base
// room). It eats 1 grain a tick, so at the arrival, two ticks later, it holds 16 grain
// (48, 32) and the room left is (12, 8): 4 grain fit by mass (12 / 3) and by bulk (8 / 2), so 4
// are delivered and 1 is diverted.
//
// Requirement: FR-STO-015
func TestArrival_DivertsWhatDoesNotFit(t *testing.T) {
	t.Parallel()
	w, rec := stoWorld(t, nil)
	stoAdd(t, w, shipFar, 0)
	stoGrain(t, w, shipFar, 8)
	if _, reason := stoShip(t, w, shipHome, shipFar, 5); reason != "" {
		t.Fatalf("dispatch = %q, want accepted", reason)
	}
	if _, err := w.removeStore(&w.agents[shipFar], 0, CauseFacility, true); err != nil {
		t.Fatalf("removeStore(forced) = %v, want nil", err)
	}
	for range 3 {
		step(t, w, 1)
	}
	arr := of[ShipmentArrived](rec)
	if len(arr) != 1 || !arr[0].Diverted || arr[0].Lines[0].Delivered != 4 || arr[0].Lines[0].Diverted != 1 {
		t.Fatalf("ShipmentArrived = %+v, want 4 delivered and 1 diverted", arr)
	}
	if st := roomOfAgent(t, w, shipFar); st.Pledge != (Capacity{}) {
		t.Errorf("pledge after a diverted arrival = %+v, want (0, 0): it falls by the whole line", st.Pledge)
	}
}

// TestShip_HasStoreAfterTravellerAway verifies row 3a: a self-carried request of an
// agent that owns a store is refused has_store, after traveller_away, and a hired one is
// not refused for it.
//
// Requirement: FR-STO-031
func TestShip_HasStoreAfterTravellerAway(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	stoAdd(t, w, shipHome, 0)
	foot := ShipmentRequest{Mode: stoFootID, To: regFar, ToSelf: true}
	if _, reason := ask(w, shipHome, 0, foot); reason != ReasonHasStore {
		t.Errorf("trip of an owner of a store = %q, want has_store", reason)
	}
	w.transport.setAway(shipHome, true)
	if _, reason := ask(w, shipHome, 0, foot); reason != ReasonTravellerAway {
		t.Errorf("trip of an owner of a store that is away = %q, want traveller_away first", reason)
	}
	w.transport.setAway(shipHome, false)
	if _, reason := stoShip(t, w, shipHome, shipFar, 5); reason != "" {
		t.Errorf("hired shipment of an owner of a store = %q, want it accepted: goods move, the owner stays", reason)
	}
}

// TestTrip_OwnerOfStoreRefused verifies that a world with the module off refuses no
// trip for a store, and with it on an agent that owns none travels: the row is empty
// until a store is owned.
//
// Requirement: FR-STO-031
func TestTrip_OwnerOfStoreRefused(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	if _, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: stoFootID, To: regFar, ToSelf: true}); reason != "" {
		t.Errorf("trip of an agent that owns no store = %q, want accepted", reason)
	}
	off, _ := newShipWorld(t, nil, nil)
	if _, reason := ask(off, shipHome, 0, ShipmentRequest{Mode: stoFootID, To: regFar, ToSelf: true}); reason != "" {
		t.Errorf("trip with the module off = %q, want accepted", reason)
	}
	_ = errors.Is
}
