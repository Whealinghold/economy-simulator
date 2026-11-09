package economysim

import (
	"log/slog"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file tests the contract round of P3.3 (plan 2-15 Phase 6): admission,
// keys, the lanes, the formation with its dispatch and escrow, and the events.
// The deliverers and the payers follow the script of merchants_helpers (a fake at
// the Strategy extension point), because strategy/ imports the root (plan D517).

// The commodities of merchConfig, by ID (canonical order by string: flour, grain,
// haul, hay, perish, wagon).
const (
	cGrain  CommodityID = 1
	cHaul   CommodityID = 2
	cPerish CommodityID = 4
)

// roundPlan is what the script returns on a tick: the offers and takes of each
// agent, by agent ID. A missing agent plans nothing.
type roundPlan struct {
	offers map[AgentID][]DeliveryOffer
	takes  map[AgentID][]DeliveryTake
	ships  map[AgentID][]ShipmentRequest
}

// script returns the hook of newMerchWorld that appends the plan of the agent at
// tick 0 only.
func (r roundPlan) script() func(v *AgentView, p *TradePlan) {
	return func(v *AgentView, p *TradePlan) {
		if v.Tick.Tick != 0 {
			return
		}
		p.Offers = append(p.Offers, r.offers[v.Agent]...)
		p.Takes = append(p.Takes, r.takes[v.Agent]...)
		p.Shipments = append(p.Shipments, r.ships[v.Agent]...)
	}
}

// stock gives each broker of the list the goods a trip needs: grain to sell and
// haul units for the coach (one carrying unit of a lot of at most 400 grain needs
// 4 haul units over the 40 distance units at span 10).
func stock(t *testing.T, w *World, deliverers ...AgentID) {
	t.Helper()
	for _, id := range deliverers {
		give(t, w, id, cGrain, 1_000)
		give(t, w, id, cHaul, 40)
	}
}

// refusals returns the reasons of the "offer refused" and "take refused" lines.
func refusals(c capture) (offers, takes []string) {
	for _, l := range *c.lines {
		switch l.msg {
		case "offer refused":
			offers = append(offers, l.strs["reason"])
		case "take refused":
			takes = append(takes, l.strs["reason"])
		}
	}
	return offers, takes
}

// roundWorld builds the world of merchConfig changed by mut with the plan, a
// recording observer and a capturing logger, with the goods given to the
// deliverers.
func roundWorld(t *testing.T, mut func(c *Config), plan roundPlan, deliverers ...AgentID) (*World, *evLog, capture) {
	t.Helper()
	lines := newCapture()
	w, rec, _ := newMerchWorld(t, mut, plan.script(), WithLogger(slog.New(lines)))
	stock(t, w, deliverers...)
	return w, rec, lines
}

// TestAdmission_ChecksInOrder verifies the order of the admission table of
// spec-merchants.md 9.3: each offer or take that breaks one rule is refused with
// that rule's reason, and one that breaks two is refused with the earlier row.
//
// The broker of a_home holds 1 000 grain and 40 haul units; the broker of b_far has
// 10 000 money. Rows 2 (too_many_requests) needs the plan's own requests.
//
// Requirement: FR-MER-031
func TestAdmission_ChecksInOrder(t *testing.T) {
	t.Parallel()
	good := DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 10, Limit: 100}
	with := func(f func(o *DeliveryOffer)) DeliveryOffer { o := good; f(&o); return o }
	offers := []struct {
		name   string
		offer  DeliveryOffer
		ships  int
		reason Reason
	}{
		{"admitted", good, 0, ""},
		{"the plan already holds the most requests", good, MaxDispatchPerTick, ReasonTooManyRequests},
		{"a mode that is not configured", with(func(o *DeliveryOffer) { o.Mode = 99 }), 0, ReasonUnknownMode},
		{"a destination that is not a region", with(func(o *DeliveryOffer) { o.To = 99 }), 0, ReasonUnknownRegion},
		{"the deliverer's own region", with(func(o *DeliveryOffer) { o.To = regHome }), 0, ReasonSameRegion},
		{"a commodity that is not configured", with(func(o *DeliveryOffer) { o.Commodity = 99 }), 0, ReasonUnknownCommodity},
		{"a commodity outside the tradable set", with(func(o *DeliveryOffer) { o.Commodity = cPerish }), 0, ReasonNotTradable},
		{"a quantity of 0", with(func(o *DeliveryOffer) { o.Quantity = 0 }), 0, ReasonInvalidQuantity},
		{"a limit of 0", with(func(o *DeliveryOffer) { o.Limit = 0 }), 0, ReasonInvalidPrice},
		{"more than the stock", with(func(o *DeliveryOffer) { o.Quantity = 2_000 }), 0, ReasonInsufficientStock},
		{"two faults: the earlier row (mode) wins", with(func(o *DeliveryOffer) { o.Mode, o.Quantity = 99, 0 }), 0, ReasonUnknownMode},
		{"two faults: the earlier row (quantity) wins over the stock", with(func(o *DeliveryOffer) { o.Quantity, o.Limit = 0, 0 }), 0, ReasonInvalidQuantity},
	}
	for _, tt := range offers {
		t.Run("offer "+tt.name, func(t *testing.T) {
			t.Parallel()
			plan := roundPlan{offers: map[AgentID][]DeliveryOffer{brokerHome: {tt.offer}}}
			if tt.ships > 0 {
				plan.ships = map[AgentID][]ShipmentRequest{brokerHome: slices.Repeat([]ShipmentRequest{{Mode: 99}}, tt.ships)}
			}
			w, _, cp := roundWorld(t, nil, plan, brokerHome)
			step(t, w, 1)
			got, _ := refusals(cp)
			switch {
			case tt.reason == "" && len(got) != 0:
				t.Errorf("refusals = %v, want the offer admitted", got)
			case tt.reason != "" && !slices.Equal(got, []string{string(tt.reason)}):
				t.Errorf("refusals = %v, want [%s]", got, tt.reason)
			}
		})
	}
	goodTake := DeliveryTake{Commodity: cGrain, Quantity: 10, Limit: 100}
	takeWith := func(f func(k *DeliveryTake)) DeliveryTake { k := goodTake; f(&k); return k }
	takes := []struct {
		name   string
		takes  []DeliveryTake
		reason Reason
	}{
		{"admitted", []DeliveryTake{goodTake}, ""},
		{"a commodity that is not configured", []DeliveryTake{takeWith(func(k *DeliveryTake) { k.Commodity = 99 })}, ReasonUnknownCommodity},
		{"a commodity outside the tradable set", []DeliveryTake{takeWith(func(k *DeliveryTake) { k.Commodity = cPerish })}, ReasonNotTradable},
		{"a quantity of 0", []DeliveryTake{takeWith(func(k *DeliveryTake) { k.Quantity = 0 })}, ReasonInvalidQuantity},
		{"a limit above the maximum", []DeliveryTake{takeWith(func(k *DeliveryTake) { k.Limit = MaxPrice + 1 })}, ReasonInvalidPrice},
		{"a second take for the commodity", []DeliveryTake{goodTake, goodTake}, ReasonDuplicateTake},
		{"more than the cash", []DeliveryTake{takeWith(func(k *DeliveryTake) { k.Quantity = 1_000 })}, ReasonInsufficientFunds},
	}
	for _, tt := range takes {
		t.Run("take "+tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, cp := roundWorld(t, nil, roundPlan{takes: map[AgentID][]DeliveryTake{brokerFar: tt.takes}})
			step(t, w, 1)
			_, got := refusals(cp)
			switch {
			case tt.reason == "" && len(got) != 0:
				t.Errorf("refusals = %v, want the take admitted", got)
			case tt.reason != "" && !slices.Equal(got, []string{string(tt.reason)}):
				t.Errorf("refusals = %v, want [%s]", got, tt.reason)
			}
		})
	}
}

// TestContractsOff_RefusesAll verifies that with the merchants module absent every
// offer and take is refused as contracts_off, a Debug line each, and that no
// Escrow account is opened and neither ContractRound nor ContractFormed is
// emitted.
//
// A hired mode needs the module, so the block of this world keeps the foot.
//
// Requirement: FR-MER-030
func TestContractsOff_RefusesAll(t *testing.T) {
	t.Parallel()
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{brokerHome: {{Mode: modeFoot, To: regFar, Commodity: cGrain, Quantity: 1, Limit: 10}}},
		takes:  map[AgentID][]DeliveryTake{brokerFar: {{Commodity: cGrain, Quantity: 1, Limit: 10}}},
	}
	w, rec, cp := roundWorld(t, func(c *Config) {
		c.Modules.Merchants = nil
		b := c.Roles[len(c.Roles)-1].Merchant
		b.Modes, b.Consigns = []string{"foot"}, false
	}, plan, brokerHome)
	step(t, w, 1)
	offers, takes := refusals(cp)
	if !slices.Equal(offers, []string{"contracts_off"}) || !slices.Equal(takes, []string{"contracts_off"}) {
		t.Errorf("refusals = %v and %v, want one contracts_off each", offers, takes)
	}
	if n := len(of[ContractRound](rec)) + len(of[ContractFormed](rec)); n != 0 {
		t.Errorf("events of the module = %d, want none while it is absent", n)
	}
	for id := range w.ledger.Len() {
		if v, _ := w.ledger.Account(id); v.Kind == ledger.KindEscrow {
			t.Errorf("account %d is an Escrow account, want none while the module is absent", id)
		}
	}
}

// trio makes three brokers in each region with money for the takes of the round
// of spec-merchants.md 19.1: a_home has the deliverers 3, 4 and 5 and b_far the
// payers 8, 9 and 10 (the millers are 6 and 7, and 11 and 12).
func trio(c *Config) {
	for i := range c.Groups {
		if c.Groups[i].Role == "broker" {
			c.Groups[i].Households = 3
		}
	}
	c.Roles[len(c.Roles)-1].StartMoney = 1_000_000
}

// The deliverers and payers of trio.
const (
	d32, d35, d33 AgentID = 3, 4, 5
	p61, p62, p63 AgentID = 8, 9, 10
)

// TestRound_Spec19Table verifies the round of spec-merchants.md 19.1, with the
// agents of trio standing for 32, 35 and 33 and for 61, 62 and 63: offers of 150
// units at 244, 100 at 250 and 150 at 263 against takes of up to 200 at 270, 150 at
// 255 and 150 at 240. Offers in order of limit; the first fits the first take at
// the midpoint (244 + 270) / 2 = 257, leaving it 50; the second (100 units) does
// not fit that and goes to the second take at (250 + 255) / 2 = 252.5, half to even
// 252; the third (limit 263) finds none, since the first has 50 left and the second
// is below its limit.
//
// ContractRound for the destination: 3 offers, 3 takes, 2 matched and formed, 1
// without a taker, 250 units and 150 * 257 + 100 * 252 = 38 550 + 25 200 = 63 750.
//
// Requirement: FR-MER-034, FR-MER-035
func TestRound_Spec19Table(t *testing.T) {
	t.Parallel()
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{
			d32: {{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 150, Limit: 244}},
			d35: {{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 100, Limit: 250}},
			d33: {{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 150, Limit: 263}},
		},
		takes: map[AgentID][]DeliveryTake{
			p61: {{Commodity: cGrain, Quantity: 200, Limit: 270}},
			p62: {{Commodity: cGrain, Quantity: 150, Limit: 255}},
			p63: {{Commodity: cGrain, Quantity: 150, Limit: 240}},
		},
	}
	w, rec, _ := roundWorld(t, trio, plan, d32, d35, d33)
	step(t, w, 1)
	got := of[ContractFormed](rec)
	if len(got) != 2 {
		t.Fatalf("contracts formed = %+v, want 2", got)
	}
	want := []struct {
		deliverer, payer AgentID
		qty              Quantity
		price            Price
	}{{d32, p61, 150, 257}, {d35, p62, 100, 252}}
	for i, c := range got {
		if c.Deliverer != want[i].deliverer || c.Payer != want[i].payer || c.Quantity != want[i].qty || c.Price != want[i].price ||
			c.Escrowed != Money(int64(want[i].qty)*int64(want[i].price)) || c.Contract != ContractID(i) || c.To != regFar || c.Region != regHome {
			t.Errorf("contract %d = %+v, want deliverer %d, payer %d, %d units at %d", i, c, want[i].deliverer, want[i].payer, want[i].qty, want[i].price)
		}
	}
	rounds := of[ContractRound](rec)
	wantRound := ContractRound{Tick: 0, Region: regFar, Offers: 3, Takes: 3, Matched: 2, Formed: 2, NoTaker: 1, Units: 250, Value: 63_750}
	if len(rounds) != 1 || rounds[0] != wantRound {
		t.Errorf("ContractRound = %+v, want one: %+v", rounds, wantRound)
	}
}

// TestRound_LaneOrderAndNoOp verifies that the lanes run in ascending (destination,
// commodity) order and that a tick with no offer or take does nothing: offers to
// b_far for haul (2) and grain (1) and from b_far to a_home for grain, each with a
// take, form in the order (a_home, grain), (b_far, grain), (b_far, haul); and a tick
// with nothing emits no ContractRound and draws no word.
//
// Requirement: FR-MER-032
func TestRound_LaneOrderAndNoOp(t *testing.T) {
	t.Parallel()
	offer := func(to RegionID, c CommodityID) DeliveryOffer {
		return DeliveryOffer{Mode: modeCoach, To: to, Commodity: c, Quantity: 5, Limit: 50}
	}
	take := func(c CommodityID) DeliveryTake { return DeliveryTake{Commodity: c, Quantity: 5, Limit: 60} }
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {offer(regFar, cHaul), offer(regFar, cGrain)}, p61: {offer(regHome, cGrain)}},
		takes:  map[AgentID][]DeliveryTake{p62: {take(cHaul), take(cGrain)}, d35: {take(cGrain)}},
	}
	w, rec, _ := roundWorld(t, trio, plan, d32, p61)
	step(t, w, 2)
	var lanes [][2]int
	for _, c := range of[ContractFormed](rec) {
		lanes = append(lanes, [2]int{int(c.To), int(c.Commodity)})
	}
	if want := [][2]int{{int(regHome), int(cGrain)}, {int(regFar), int(cGrain)}, {int(regFar), int(cHaul)}}; !slices.Equal(lanes, want) {
		t.Errorf("lanes formed = %v, want %v", lanes, want)
	}
	if n := len(of[ContractRound](rec)); n != 2 {
		t.Errorf("ContractRound events in 2 ticks = %d, want 2 (one for each destination of tick 0, none at tick 1)", n)
	}
	// An empty round keys nothing: the stream is where it was.
	before, _ := w.rnd.pcg.MarshalBinary()
	w.p33Round()
	if after, _ := w.rnd.pcg.MarshalBinary(); !slices.Equal(before, after) {
		t.Error("a round with no entity moved the random stream, want no word taken")
	}
}

// TestRound_KeysPerEntity verifies the keys of the round (spec-merchants.md 9.4.2,
// FR-MER-033): equal limits are told apart by the first word of the stream
// (siteContract, tick, agent, 2 * index) for an offer, so the order of two equal
// offers is the order of those words and does not depend on the agents' IDs.
//
// Two deliverers offer 5 units at the same limit to one take of 5: the contract goes
// to the one whose key is lower.
//
// Requirement: FR-MER-033
func TestRound_KeysPerEntity(t *testing.T) {
	t.Parallel()
	o := DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 5, Limit: 50}
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {o}, d35: {o}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 5, Limit: 80}}},
	}
	w, rec, _ := roundWorld(t, trio, plan, d32, d35)
	step(t, w, 1)
	key := func(a AgentID) uint64 {
		w.rnd.keyedPair(siteContract, 0, int64(a), 0)
		return w.rnd.word()
	}
	want := d32
	if key(d35) < key(d32) {
		want = d35
	}
	got := of[ContractFormed](rec)
	if len(got) != 1 || got[0].Deliverer != want {
		t.Errorf("contracts = %+v, want one by agent %d, the lower key (%d against %d)", got, want, key(d32), key(d35))
	}
}

// TestFormation_Order verifies the order and the failures of formation (spec-
// merchants.md 9.5): too_many_contracts when MaxOpen contracts are in flight,
// payer_funds when an earlier contract of the round spent the cash a take was sized
// on, and a refusal of transport, which drops the offer, counts DispatchRefused and
// leaves the take whole for the next offer.
//
// Requirement: FR-MER-036, FR-MER-046
func TestFormation_Order(t *testing.T) {
	t.Parallel()
	o := func(c CommodityID, q Quantity, limit Price) DeliveryOffer {
		return DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: c, Quantity: q, Limit: limit}
	}
	t.Run("too many contracts", func(t *testing.T) {
		t.Parallel()
		plan := roundPlan{
			offers: map[AgentID][]DeliveryOffer{d32: {o(cGrain, 5, 50)}, d35: {o(cGrain, 5, 51)}},
			takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 80}}},
		}
		w, rec, _ := roundWorld(t, func(c *Config) { trio(c); c.Modules.Merchants.MaxOpen = 1 }, plan, d32, d35)
		step(t, w, 1)
		if r := of[ContractRound](rec); len(r) != 1 || r[0].Formed != 1 || r[0].TooMany != 1 || r[0].Matched != 2 {
			t.Errorf("ContractRound = %+v, want 2 matched, 1 formed and 1 refused as too many", r)
		}
	})
	t.Run("payer funds", func(t *testing.T) {
		t.Parallel()
		// One payer takes grain and perish at 8 000 each, each affordable alone with
		// 10 000; the first contract leaves 2 000, so the second fails.
		plan := roundPlan{
			offers: map[AgentID][]DeliveryOffer{d32: {o(cGrain, 10, 800), o(cPerish, 10, 800)}},
			takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 800}, {Commodity: cPerish, Quantity: 10, Limit: 800}}},
		}
		w, rec, _ := roundWorld(t, func(c *Config) {
			trio(c)
			c.Roles[len(c.Roles)-1].StartMoney = 10_000
			c.Roles[len(c.Roles)-1].Trades = append(c.Roles[len(c.Roles)-1].Trades, "perish")
		}, plan, d32)
		give(t, w, d32, cPerish, 20)
		step(t, w, 1)
		if r := of[ContractRound](rec); len(r) != 1 || r[0].Formed != 1 || r[0].PayerFunds != 1 {
			t.Errorf("ContractRound = %+v, want 1 formed and 1 refused for the payer's funds", r)
		}
	})
	t.Run("a refusal of transport drops the offer and leaves the take", func(t *testing.T) {
		t.Parallel()
		// d32 has no haul units (insufficient_haul); d35 does. The take of 5 survives
		// the failed first offer (the lower limit) and meets the second.
		plan := roundPlan{
			offers: map[AgentID][]DeliveryOffer{d32: {o(cGrain, 5, 40)}, d35: {o(cGrain, 5, 50)}},
			takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 5, Limit: 80}}},
		}
		w, rec, _ := roundWorld(t, trio, plan, d35)
		give(t, w, d32, cGrain, 100)
		step(t, w, 1)
		r := of[ContractRound](rec)
		if len(r) != 1 || r[0].Formed != 1 || r[0].DispatchRefused != 1 || r[0].Matched != 2 {
			t.Errorf("ContractRound = %+v, want 2 matched, 1 formed and 1 dispatch refused", r)
		}
		if rej := of[ShipmentRejected](rec); len(rej) != 1 || rej[0].Agent != d32 {
			t.Errorf("ShipmentRejected = %+v, want the refusal of the first deliverer", rej)
		}
		if f := of[ContractFormed](rec); len(f) != 1 || f[0].Deliverer != d35 {
			t.Errorf("ContractFormed = %+v, want the second deliverer's contract", f)
		}
	})
}

// TestFormation_DispatchThenEscrow verifies that a contract exists only with both
// parts (spec-merchants.md 9.5): the shipment is in flight, consigned to the payer
// and carrying the units, the Escrow account of the destination holds exactly price
// * quantity, the payer's money fell by it, and the deadline is the arrival tick of
// the shipment (dispatch tick + 2: 40 distance units at speed 30).
//
// Requirement: FR-MER-037, FR-MER-038, FR-TRN-048
func TestFormation_DispatchThenEscrow(t *testing.T) {
	t.Parallel()
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 10, Limit: 90}}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 110}}},
	}
	w, rec, _ := roundWorld(t, trio, plan, d32)
	before := w.cash(&w.agents[p61])
	step(t, w, 1)
	f := of[ContractFormed](rec)
	if len(f) != 1 {
		t.Fatalf("contracts formed = %+v, want 1", f)
	}
	s := w.shipmentOf(f[0].Shipment)
	if s.consignee != p61 || s.shipper != d32 || !s.hired || len(s.lines) != 1 || s.lines[0] != (Line{Commodity: cGrain, Quantity: 10}) {
		t.Errorf("shipment = %+v, want a hired one from agent %d to agent %d with 10 grain", s, d32, p61)
	}
	if f[0].Arrives != 2 || s.arrives != 2 {
		t.Errorf("Arrives = %d, shipment arrives %d, want tick 2 (dispatched at 0, 2 ticks)", f[0].Arrives, s.arrives)
	}
	ct := w.mer.open[0]
	view, _ := w.ledger.Account(ct.escrow)
	if view.Kind != ledger.KindEscrow || view.Balance != 1_000 || view.Region != int(regFar) {
		t.Errorf("Escrow account = %+v, want an Escrow account of b_far holding 10 * 100 = 1 000", view)
	}
	if after := w.cash(&w.agents[p61]); before-after != 1_000 {
		t.Errorf("the payer's money fell by %d, want 1 000", before-after)
	}
}

// TestEscrow_LowestFreeIDReused verifies the Escrow accounts (spec-merchants.md
// 9.6.2): a contract takes the lowest-ID open Escrow account of the destination
// that no open contract uses, or opens one, and an account is never closed. Two
// accounts of b_far are open and free; two contracts take the lower, then the
// other, and a third finds none free and opens a new one.
//
// Requirement: FR-MER-039
func TestEscrow_LowestFreeIDReused(t *testing.T) {
	t.Parallel()
	o := func(q Quantity, limit Price) DeliveryOffer {
		return DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: q, Limit: limit}
	}
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {o(1, 10), o(1, 11), o(1, 12)}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 3, Limit: 100}}},
	}
	w, rec, _ := roundWorld(t, trio, plan, d32)
	lo, err1 := w.ledger.Open(ledger.KindEscrow, 0, int(regFar))
	hi, err2 := w.ledger.Open(ledger.KindEscrow, 0, int(regFar))
	if err1 != nil || err2 != nil {
		t.Fatalf("Open() = %v, %v, want nil", err1, err2)
	}
	w.mer.free[regFar].Put(hi)
	w.mer.free[regFar].Put(lo)
	step(t, w, 1)
	if f := of[ContractFormed](rec); len(f) != 3 {
		t.Fatalf("contracts formed = %+v, want 3", f)
	}
	var got []int
	for _, ct := range w.mer.open {
		got = append(got, ct.escrow)
	}
	if len(got) != 3 || got[0] != lo || got[1] != hi || got[2] == lo || got[2] == hi {
		t.Errorf("Escrow accounts of the contracts = %v, want %d, %d, then a new one", got, lo, hi)
	}
	if w.mer.free[regFar].Len() != 0 {
		t.Errorf("free Escrow accounts = %d, want 0", w.mer.free[regFar].Len())
	}
}

// TestContractRound_Order verifies the order of the events of the round (spec-
// merchants.md 15.1.1, observability 4.3): the ShipmentDispatched and
// ContractFormed of each contract in formation order, then ContractRound, then
// MerchantsPlanned, then RumoursServed, all after the last OrderPlaced of the phase.
//
// Requirement: FR-MER-055, FR-MER-056
func TestContractRound_Order(t *testing.T) {
	t.Parallel()
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 1, Limit: 10}}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 1, Limit: 20}}},
	}
	w, rec, _ := roundWorld(t, trio, plan, d32)
	step(t, w, 1)
	last := func(match func(Event) bool) int {
		at := -1
		for i, e := range rec.events {
			if match(e) {
				at = i
			}
		}
		return at
	}
	first := func(match func(Event) bool) int {
		for i, e := range rec.events {
			if match(e) {
				return i
			}
		}
		return -1
	}
	disp := first(func(e Event) bool { _, ok := e.(ShipmentDispatched); return ok })
	formed := first(func(e Event) bool { _, ok := e.(ContractFormed); return ok })
	round := first(func(e Event) bool { _, ok := e.(ContractRound); return ok })
	planned := first(func(e Event) bool { _, ok := e.(MerchantsPlanned); return ok })
	served := first(func(e Event) bool { _, ok := e.(RumoursServed); return ok })
	orders := last(func(e Event) bool { _, ok := e.(OrderPlaced); return ok })
	if !(orders < disp && disp < formed && formed < round && round < planned && planned < served) {
		t.Errorf("event positions: last order %d, ShipmentDispatched %d, ContractFormed %d, ContractRound %d, MerchantsPlanned %d, RumoursServed %d, want them in that order",
			orders, disp, formed, round, planned, served)
	}
}

// TestConservation_RoundWithAudit verifies that a contract formed moves money and
// goods without a faucet, a sink or a new reason (spec-merchants.md 9.6.1,
// FR-MER-044): the audit runs on every tick, the money in existence is the same
// before and after, and the escrow reason is not a boundary flow.
//
// Requirement: FR-MER-044
func TestConservation_RoundWithAudit(t *testing.T) {
	t.Parallel()
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 10, Limit: 90}}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 110}}},
	}
	w, rec, _ := roundWorld(t, trio, plan, d32)
	total := func() (sum int64) {
		for id := range w.ledger.Len() {
			if v, err := w.ledger.Account(id); err == nil && v.Kind != ledger.KindBoundary {
				sum += v.Balance
			}
		}
		return sum
	}
	before := total()
	step(t, w, 3) // WithAudit is on in newMerchWorld: a violated identity fails the step
	if after := total(); after != before {
		t.Errorf("money in the accounts = %d after the round, %d before, want it conserved", after, before)
	}
	for _, f := range of[BoundaryFlow](rec) {
		if f.Reason == ReasonEscrowIn || f.Reason == ReasonEscrowOut || f.Reason == ReasonEscrowRefund {
			t.Errorf("BoundaryFlow %+v names an escrow reason, want none: the contract adds no faucet and no sink", f)
		}
	}
}

// TestContractRound_EventFields verifies every field of ContractFormed and of
// ContractRound for one contract: 10 grain from agent 3 in a_home to agent 8 in
// b_far, an offer limit of 90 and a take limit of 110, so the price is
// (90 + 110) / 2 = 100 and the escrow 1 000; the coach takes 2 ticks.
//
// Requirement: FR-MER-055
func TestContractRound_EventFields(t *testing.T) {
	t.Parallel()
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 10, Limit: 90}}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 110}}},
	}
	w, rec, _ := roundWorld(t, trio, plan, d32)
	step(t, w, 1)
	want := ContractFormed{Tick: 0, Contract: 0, Payer: p61, Deliverer: d32, Region: regHome, To: regFar, Commodity: cGrain,
		Quantity: 10, Price: 100, Escrowed: 1_000, Shipment: 0, Mode: modeCoach, Arrives: 2}
	if got := of[ContractFormed](rec); len(got) != 1 || got[0] != want {
		t.Errorf("ContractFormed = %+v, want %+v", got, want)
	}
	wantRound := ContractRound{Tick: 0, Region: regFar, Offers: 1, Takes: 1, Matched: 1, Formed: 1, Units: 10, Value: 1_000}
	if got := of[ContractRound](rec); len(got) != 1 || got[0] != wantRound {
		t.Errorf("ContractRound = %+v, want %+v", got, wantRound)
	}
}

// TestRound_DebugLines verifies the Debug lines of the round (spec-merchants.md
// 15.2): "contract formed" with the contract, the deliverer, the regions, the goods
// and the price, "offer refused" and "take refused" with the agent and the reason,
// all at Debug.
//
// Requirement: FR-MER-058
func TestRound_DebugLines(t *testing.T) {
	t.Parallel()
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 10, Limit: 90}, {Mode: 99}}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 110}, {Commodity: cGrain, Quantity: 1, Limit: 1}}},
	}
	w, _, cp := roundWorld(t, trio, plan, d32)
	step(t, w, 1)
	var formed []logRec
	for _, l := range *cp.lines {
		module := l.msg == "contract formed" || l.msg == "offer refused" || l.msg == "take refused"
		if module && l.level != slog.LevelDebug {
			t.Errorf("line %q at level %v, want Debug", l.msg, l.level)
		}
		if l.msg == "contract formed" {
			formed = append(formed, l)
		}
	}
	if len(formed) != 1 {
		t.Fatalf("contract formed lines = %d, want 1", len(formed))
	}
	a := formed[0].attrs
	if a["contract"] != 0 || a["agent"] != int64(d32) || a["region"] != int64(regHome) || a["dest"] != int64(regFar) || a["quantity"] != 10 || a["price"] != 100 || a["commodity"] != int64(cGrain) {
		t.Errorf("contract formed line = %+v, want contract 0, agent %d, region %d, dest %d, 10 grain at 100", a, d32, regHome, regFar)
	}
	offers, takes := refusals(cp)
	if !slices.Equal(offers, []string{"too_many_requests"}) && !slices.Equal(offers, []string{"unknown_mode"}) {
		t.Errorf("offer refusals = %v, want the unknown mode", offers)
	}
	if !slices.Equal(takes, []string{"duplicate_take"}) {
		t.Errorf("take refusals = %v, want the duplicate take", takes)
	}
}
