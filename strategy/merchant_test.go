package strategy_test

import (
	"math/rand/v2"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// bidFor returns the bid of the plan for the commodity, and whether there is one.
func bidFor(p economysim.TradePlan, c economysim.CommodityID) (economysim.Intent, bool) {
	for _, in := range p.Intents {
		if in.Commodity == c && in.Side == economysim.Bid {
			return in, true
		}
	}
	return economysim.Intent{}, false
}

// TestMerchant_WorkedExampleA verifies the lane of spec-merchants.md 19.1 for
// agent 32, whose fee share is 6 385, in the two ticks of a trip.
//
// Prepare: one candidate (the fuel lane to the remote region by barge). The
// cautious revenue is min(310, 305) = 305, net of the cut of 500 basis points
// 305 * 9 500 / 10 000 = 289; the local price is 200. A carrying unit holds
// min(1 200 / 2, 900 / 6) = 150 fuel, so the lot is 150 and n = 1. The freight is
// 1 * 6 * 700 = 4 200 and G = 150 * (289 - 200) = 13 350; the test 4 200 * 10 000
// <= 6 385 * 13 350 holds. The break-even is 289 - ceil(4 200 * 10 000 / (6 385 *
// 150)) = 289 - 44 = 245, the offer limit 200 + 44 = 244, the haul ceiling
// 13 350 / 6 = 2 225, times 6 385 / 10 000 = 1 420. The merchant buys 150 fuel and
// 6 haul units (the money affords more than the cap, so the lot is counted as
// capped): with the belief centre of fuel at 300 (draws 270 to 330) its limit
// is lowered to 245, and with the haul centre at 2 000 (draws above 1 420) to 1 420.
//
// Send: holding 150 fuel and 6 haul units it offers 150 fuel to the remote region
// at 244 and buys nothing.
//
// Requirement: FR-MER-008, FR-MER-013, FR-MER-015, FR-MER-016, FR-MER-017
func TestMerchant_WorkedExampleA(t *testing.T) {
	t.Parallel()
	t.Run("prepare", func(t *testing.T) {
		t.Parallel()
		v := valley()
		v.Commodities[0] = row(fuel, 0, 300, 1000, 1000, 200)
		v.Commodities[1] = row(haul, 0, 2000, 1000, 5000, 700)
		p, _ := trade(t, merchantStrategy, v)
		if b, ok := bidFor(p, fuel); !ok || b.Quantity != 150 || b.Limit != 245 || b.Purpose != economysim.PurposeInput {
			t.Errorf("fuel bid = %+v (found %v), want 150 units at 245 of purpose Input (the break-even)", b, ok)
		}
		if b, ok := bidFor(p, haul); !ok || b.Quantity != 6 || b.Limit != 1420 || b.Purpose != economysim.PurposeInput {
			t.Errorf("haul bid = %+v (found %v), want 6 units at 1 420 of purpose Input (the haul ceiling)", b, ok)
		}
		want := economysim.PlanNotes{Considered: 1, Planned: 1, LotCapped: 1}
		if p.Notes != want || len(p.Offers) != 0 || len(p.Takes) != 0 || len(p.Shipments) != 0 {
			t.Errorf("notes = %+v, offers %v, takes %v, shipments %v, want %+v and nothing sent (not ready)", p.Notes, p.Offers, p.Takes, p.Shipments, want)
		}
	})
	t.Run("send", func(t *testing.T) {
		t.Parallel()
		v := valley()
		v.Commodities[0].Stock, v.Commodities[1].Stock = 150, 6
		p, _ := trade(t, merchantStrategy, v)
		wantOffer := economysim.DeliveryOffer{Mode: barge, To: remote, Commodity: fuel, Quantity: 150, Limit: 244}
		if !reflect.DeepEqual(p.Offers, []economysim.DeliveryOffer{wantOffer}) {
			t.Errorf("offers = %+v, want one: %+v", p.Offers, wantOffer)
		}
		if len(p.Intents) != 0 || len(p.Takes) != 0 {
			t.Errorf("intents %+v, takes %+v, want none: the inputs are held and the cargo is not taken", p.Intents, p.Takes)
		}
		if want := (economysim.PlanNotes{Considered: 1, Planned: 1, Offered: 1, LotCapped: 1}); p.Notes != want {
			t.Errorf("notes = %+v, want %+v", p.Notes, want)
		}
	})
}

// TestMerchant_WorkedExampleB verifies the self-carried lane of spec-merchants.md
// 19.2 for agent 18, whose fee share is 3 123: a trip of k = 3 ticks, a lot of 100
// rations (the money affords (5 905 - 105) / 58 = 100, the cap is 1 500), a cost of
// wear ceil(1 * 5 * 3 * 4 000 / 400) = 150, feed 1 * 1 * 3 * 30 = 90 and time
// 3 * 56 * 10 000 / 10 000 = 168, so C = 408 against G = 100 * (78 - 56) = 2 200:
// 408 * 10 000 = 4 080 000 <= 3 123 * 2 200 = 6 870 600. The break-even is 78 -
// ceil(4 080 000 / 312 300) = 78 - 14 = 64.
//
// Prepare: it holds the skiff and buys 100 rations and 3 cells, and no provisions
// (the trip of 3 ticks is not above the cover of 3). Send: holding the rations
// above its keep of 4 (1 + 3 cover ticks), the cells and the skiff, it requests
// the trip to itself.
//
// Requirement: FR-MER-008, FR-MER-011, FR-MER-013, FR-MER-015
func TestMerchant_WorkedExampleB(t *testing.T) {
	t.Parallel()
	t.Run("prepare", func(t *testing.T) {
		t.Parallel()
		v := colony()
		v.Commodities[1].Belief.Centre = 56
		p, _ := trade(t, merchantStrategy, v)
		b, ok := bidFor(p, ration)
		if !ok || b.Quantity < 100 || b.Limit > 64 {
			t.Errorf("ration bid = %+v (found %v), want at least the 100 of the lot at no more than the break-even 64", b, ok)
		}
		if c, ok := bidFor(p, cell); !ok || c.Quantity != 3 {
			t.Errorf("cell bid = %+v (found %v), want the 3 cells of the feed", c, ok)
		}
		if _, ok := bidFor(p, skiff); ok {
			t.Error("a bid for the skiff, want none: the vehicle is held")
		}
		if len(p.Shipments) != 0 || p.Notes.Planned != 1 || p.Notes.Considered != 1 {
			t.Errorf("shipments %+v, notes %+v, want none sent and one candidate planned", p.Shipments, p.Notes)
		}
	})
	t.Run("send", func(t *testing.T) {
		t.Parallel()
		v := colony()
		v.Commodities[0].Stock, v.Commodities[1].Stock = 3, 104
		p, _ := trade(t, merchantStrategy, v)
		want := []economysim.ShipmentRequest{{Mode: 0, To: remote, ToSelf: true}}
		if !reflect.DeepEqual(p.Shipments, want) {
			t.Errorf("shipments = %+v, want %+v", p.Shipments, want)
		}
		if p.Notes.Dispatched != 1 || len(p.Offers) != 0 {
			t.Errorf("notes %+v, offers %v, want one dispatch and no offer", p.Notes, p.Offers)
		}
	})
}

// TestMerchant_PorterFailsTheTest verifies the self-carried porter of 19.1 on the
// valley lane: a unit of (60, 40) crew 1 at speed 20 takes 3 ticks and carries
// min(60 / 2, 40 / 6) = 6 fuel, a gross margin of 6 * (305 - 200) = 630 against a
// time cost of 3 * 200 = 600, and agent 32's share of 6 385 * 630 / 10 000 = 402 is
// below it: BelowShare, no trip.
//
// Requirement: FR-MER-013
func TestMerchant_PorterFailsTheTest(t *testing.T) {
	t.Parallel()
	v := valley()
	v.Upkeep = []economysim.UpkeepRow{{Commodity: fuel, Quantity: 1}}
	v.Commodities[0] = row(fuel, 0, 200, 0, 1000, 200)
	v.Merchant.Modes = []economysim.MerchantMode{{Mode: 1, UnitMass: 60, UnitBulk: 40, Crew: 1, Vehicle: economysim.NoCommodity, Haul: economysim.NoCommodity}}
	v.Routes = []economysim.RouteRow{{To: remote, Mode: 1, Has: true, Ticks: 3}}
	v.Rumours[0].Price = 305
	v.Merchant.Params.Consigns = false
	p, _ := trade(t, merchantStrategy, v)
	if want := (economysim.PlanNotes{Considered: 1, BelowShare: 1}); p.Notes != want || len(p.Shipments) != 0 {
		t.Errorf("notes = %+v, shipments %v, want %+v and no trip", p.Notes, p.Shipments, want)
	}
}

// TestMerchant_CandidateCountedOnce verifies that each candidate is counted once,
// by the first case of spec-merchants.md 5.1.2 that applies. Eight regions, each
// built to be decided by one case: 1 has no rumour; 2 a source age of 3 (the bound
// is 2); 3 has units on the way; 4 a trip of 3 ticks for fuel with a shelf life of
// 4 (life left 4 - 1 - 1 = 2); 5 a price of 200, whose revenue 190 is not above the
// local 200; 6 a haul of 600 units a carrying unit, whose other inputs cost more
// than the money; 7 a price of 212 (revenue 201, a gross margin of 150 against the
// freight 4 200: 42 000 000 > 6 385 * 150); 8 passes. Every count is 1, Considered
// is 8 and the passing one is planned.
//
// Requirement: FR-MER-007
func TestMerchant_CandidateCountedOnce(t *testing.T) {
	t.Parallel()
	v := valley()
	v.Merchant.Goods[0].ShelfLife = 4
	v.Routes, v.Rumours = nil, nil
	for r := economysim.RegionID(1); r <= 8; r++ {
		row := economysim.RouteRow{To: r, Mode: barge, Has: true, Ticks: 2, HaulPerUnit: 6}
		switch r {
		case 4:
			row.Ticks = 3
		case 6:
			row.HaulPerUnit = 600
		}
		v.Routes = append(v.Routes, row)
		if r == 1 {
			continue
		}
		rumour := economysim.Rumour{Region: r, Commodity: fuel, Price: 310, Mean: 305, Lag: 3}
		switch r {
		case 2:
			rumour.SourceAge = 3
		case 5:
			rumour.Price, rumour.Mean = 200, 0
		case 7:
			rumour.Price, rumour.Mean = 212, 0
		}
		v.Rumours = append(v.Rumours, rumour)
	}
	v.Outbound = []economysim.OutboundRow{{To: 3, Commodity: fuel, Quantity: 5, Earliest: 4}}
	p, _ := trade(t, merchantStrategy, v)
	want := economysim.PlanNotes{Considered: 8, NoRumour: 1, Stale: 1, Locked: 1, Perishable: 1, NoMargin: 1, Unaffordable: 1, BelowShare: 1, Planned: 1, LotCapped: 1}
	if p.Notes != want {
		t.Errorf("notes = %+v, want %+v", p.Notes, want)
	}
}

// TestMerchant_StaleRumour verifies the source-age bound (spec-merchants.md 5.2.2,
// R4): a rumour is acted on when SourceAge is at most MaxSourceAge (2) and counted
// Stale above it.
//
// Requirement: FR-MER-009
func TestMerchant_StaleRumour(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		age         int64
		stale, plan int64
	}{{0, 0, 1}, {2, 0, 1}, {3, 1, 0}} {
		v := valley()
		v.Rumours[0].SourceAge = tt.age
		p, _ := trade(t, merchantStrategy, v)
		if p.Notes.Stale != tt.stale || p.Notes.Planned != tt.plan {
			t.Errorf("SourceAge %d: Stale = %d, Planned = %d, want %d and %d", tt.age, p.Notes.Stale, p.Notes.Planned, tt.stale, tt.plan)
		}
	}
}

// TestMerchant_LaneLocked verifies the lane lock (spec-merchants.md 7.1, R1): units
// of the cargo on the way to the destination lock that lane and no other, and the
// merchant plans nothing on it.
//
// Requirement: FR-MER-024
func TestMerchant_LaneLocked(t *testing.T) {
	t.Parallel()
	v := valley()
	v.Outbound = []economysim.OutboundRow{{To: remote, Commodity: fuel, Quantity: 1, Earliest: 3}}
	p, _ := trade(t, merchantStrategy, v)
	if want := (economysim.PlanNotes{Considered: 1, Locked: 1}); p.Notes != want || len(p.Offers) != 0 {
		t.Errorf("a locked lane: notes = %+v, offers %v, want %+v and no offer", p.Notes, p.Offers, want)
	}
	v.Outbound = []economysim.OutboundRow{{To: remote, Commodity: haul, Quantity: 1, Earliest: 3}, {To: 5, Commodity: fuel, Quantity: 1, Earliest: 3}}
	if p, _ = trade(t, merchantStrategy, v); p.Notes.Locked != 0 || p.Notes.Planned != 1 {
		t.Errorf("units of another commodity and of another destination on the way: notes = %+v, want the lane free", p.Notes)
	}
}

// TestMerchant_Perishable verifies the life-left rule (spec-merchants.md 5.3.1,
// 19.1): a fresh lot sent the tick after it is bought is 1 tick old, so with a
// shelf life of 3 the life left is 3 - 1 - 1 = 1 tick and the trip of 2 ticks is
// set aside; with 4 the life left is 2 and the lane passes. A commodity that never
// spoils is never set aside, and an older lot shortens the life.
//
// Requirement: FR-MER-010
func TestMerchant_Perishable(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		shelf, age int64
		perishable int64
	}{{3, -1, 1}, {4, -1, 0}, {0, 100, 0}, {5, 3, 1}, {5, 2, 0}} {
		v := valley()
		v.Merchant.Goods[0].ShelfLife, v.Merchant.Goods[0].OldestAge = tt.shelf, tt.age
		p, _ := trade(t, merchantStrategy, v)
		if p.Notes.Perishable != tt.perishable {
			t.Errorf("shelf life %d, oldest age %d: Perishable = %d, want %d", tt.shelf, tt.age, p.Notes.Perishable, tt.perishable)
		}
	}
}

// TestMerchant_ModeComparison verifies the choice among the modes of a lane
// (spec-merchants.md 6.1.1): the passing mode with the greatest G - C wins, and
// equal scores go to the lower mode. Two hired barges differ in the haul units a
// carrying unit costs: 6 and 3, so the second has the lower freight and wins; with
// the same 6 the first wins.
//
// Requirement: FR-MER-019
func TestMerchant_ModeComparison(t *testing.T) {
	t.Parallel()
	hired := economysim.MerchantMode{Hired: true, UnitMass: 1200, UnitBulk: 900, Vehicle: economysim.NoCommodity, Haul: haul, HaulSpan: 10}
	for _, tt := range []struct {
		name  string
		haul0 int64
		haul1 int64
		want  economysim.ModeID
	}{{"the cheaper freight wins", 6, 3, 1}, {"equal scores go to the lower mode", 6, 6, 0}, {"the cheaper freight wins in the other order", 3, 6, 0}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := valley()
			m0, m1 := hired, hired
			m0.Mode, m1.Mode = 0, 1
			v.Merchant.Modes = []economysim.MerchantMode{m0, m1}
			v.Routes = []economysim.RouteRow{
				{To: remote, Mode: 0, Has: true, Ticks: 2, HaulPerUnit: tt.haul0},
				{To: remote, Mode: 1, Has: true, Ticks: 2, HaulPerUnit: tt.haul1},
			}
			v.Commodities[0].Stock, v.Commodities[1].Stock = 150, 6
			p, _ := trade(t, merchantStrategy, v)
			if tt.haul0 != tt.haul1 {
				// The lot holds 6 haul units and the cheaper mode needs 3 or 6 of them: ready.
				v.Commodities[1].Stock = 6
			}
			if len(p.Offers) != 1 || p.Offers[0].Mode != tt.want {
				t.Errorf("offers = %+v, want one by mode %d", p.Offers, tt.want)
			}
			if p.Notes.Considered != 2 {
				t.Errorf("Considered = %d, want 2 (one lane, two modes)", p.Notes.Considered)
			}
		})
	}
}

// TestMerchant_OneRequestOrOffer verifies the dispatch rule (spec-merchants.md
// 6.3): a trip that is not ready sends nothing, a ready one appends exactly one
// request or offer, and a ready trip that fails the test again at the lot it holds
// sends nothing and has the surplus sold: with 6 fuel and 6 haul units the lot is 6,
// a gross margin of 6 * 89 = 534 against the freight 4 200.
//
// Requirement: FR-MER-021, FR-MER-025
func TestMerchant_OneRequestOrOffer(t *testing.T) {
	t.Parallel()
	notReady := valley()
	if p, _ := trade(t, merchantStrategy, notReady); len(p.Offers)+len(p.Shipments) != 0 {
		t.Errorf("a trip that is not ready sent offers %v, shipments %v, want none", p.Offers, p.Shipments)
	}
	ready := valley()
	ready.Commodities[0].Stock, ready.Commodities[1].Stock = 150, 6
	if p, _ := trade(t, merchantStrategy, ready); len(p.Offers) != 1 || len(p.Shipments) != 0 {
		t.Errorf("a ready trip sent offers %v, shipments %v, want exactly one offer", p.Offers, p.Shipments)
	}
	small := valley()
	small.Commodities[0].Stock, small.Commodities[1].Stock = 6, 6
	p, _ := trade(t, merchantStrategy, small)
	if len(p.Offers) != 0 {
		t.Errorf("a lot of 6 sent offers %v, want none: 534 against 4 200 fails the test", p.Offers)
	}
	sold := false
	for _, in := range p.Intents {
		sold = sold || (in.Commodity == fuel && in.Side == economysim.Ask)
	}
	if !sold {
		t.Errorf("intents = %+v, want an ask for the fuel the trip no longer needs (3.4.3)", p.Intents)
	}
}

// TestMerchant_Stays verifies the stay rules (spec-merchants.md 6.4): goods
// inbound or a job in progress plan no self-carried trip and count Waiting once,
// a hired trip is unaffected, and a merchant in the own-use mode plans no trip at
// all and posts no take.
//
// Requirement: FR-MER-022
func TestMerchant_Stays(t *testing.T) {
	t.Parallel()
	t.Run("inbound goods stop a self-carried trip", func(t *testing.T) {
		t.Parallel()
		v := colony()
		v.Inbound = []economysim.InboundRow{{Commodity: ration, Quantity: 5, Earliest: 2}}
		p, _ := trade(t, merchantStrategy, v)
		if p.Notes.Waiting != 1 || p.Notes.Considered != 0 || p.Notes.Planned != 0 {
			t.Errorf("notes = %+v, want Waiting 1 and no candidate", p.Notes)
		}
	})
	t.Run("a job stops a self-carried trip", func(t *testing.T) {
		t.Parallel()
		v := colony()
		v.HasJob = true
		if p, _ := trade(t, merchantStrategy, v); p.Notes.Waiting != 1 || p.Notes.Considered != 0 {
			t.Errorf("notes = %+v, want Waiting 1 and no candidate", p.Notes)
		}
	})
	t.Run("a hired trip is unaffected", func(t *testing.T) {
		t.Parallel()
		v := valley()
		v.HasJob = true
		v.Inbound = []economysim.InboundRow{{Commodity: fuel, Quantity: 5, Earliest: 2}}
		if p, _ := trade(t, merchantStrategy, v); p.Notes.Waiting != 0 || p.Notes.Planned != 1 {
			t.Errorf("notes = %+v, want no wait and the trip planned", p.Notes)
		}
	})
	t.Run("the own-use mode plans nothing and posts no take", func(t *testing.T) {
		t.Parallel()
		v := valley()
		v.OwnUse = true
		p, _ := trade(t, merchantStrategy, v)
		if p.Notes.Waiting != 1 || p.Notes.Considered != 0 || len(p.Takes) != 0 || len(p.Offers) != 0 {
			t.Errorf("notes = %+v, takes %v, offers %v, want Waiting 1 and nothing planned or taken", p.Notes, p.Takes, p.Offers)
		}
	})
}

// TestMerchant_Rested verifies the work share (spec-merchants.md 6.5): a merchant
// whose welfare ratio is above its target band rests, counts Rested once and plans
// no trip and no take, and still trades as the default does: it offers the fuel it
// holds as surplus.
//
// Requirement: FR-MER-023
func TestMerchant_Rested(t *testing.T) {
	t.Parallel()
	v := valley()
	v.Ratio = 20_000 // above the target of 15 000 plus the band of 500
	v.Commodities[0].Stock = 10
	p, _ := trade(t, merchantStrategy, v)
	if p.Notes.Rested != 1 || p.Notes.Considered != 0 || len(p.Offers)+len(p.Takes)+len(p.Shipments) != 0 {
		t.Errorf("notes = %+v, offers %v, takes %v, want Rested 1 and nothing planned", p.Notes, p.Offers, p.Takes)
	}
	if ask := func() bool {
		for _, in := range p.Intents {
			if in.Commodity == fuel && in.Side == economysim.Ask {
				return true
			}
		}
		return false
	}(); !ask {
		t.Errorf("intents = %+v, want the surplus fuel offered as the default does", p.Intents)
	}
	v.Hunger = 1 // the survival rule gives a whole share whatever the ratio says
	if p, _ = trade(t, merchantStrategy, v); p.Notes.Rested != 0 || p.Notes.Planned != 1 {
		t.Errorf("a hungry merchant: notes = %+v, want it to work", p.Notes)
	}
}

// TestMerchant_NoRouteNoCandidate verifies that a lane with no route is not a
// candidate (spec-merchants.md 5.1.1): a row without a route, a mode the block does
// not list and a destination with no row give no candidate at all, and so no count.
//
// Requirement: FR-MER-049
func TestMerchant_NoRouteNoCandidate(t *testing.T) {
	t.Parallel()
	v := valley()
	v.Routes[0].Has = false
	if p, _ := trade(t, merchantStrategy, v); p.Notes != (economysim.PlanNotes{}) {
		t.Errorf("a row without a route: notes = %+v, want none", p.Notes)
	}
	v = valley()
	v.Routes[0].Mode = 3
	if p, _ := trade(t, merchantStrategy, v); p.Notes != (economysim.PlanNotes{}) {
		t.Errorf("a mode the block does not list: notes = %+v, want none", p.Notes)
	}
	v = valley()
	v.Routes = nil
	if p, _ := trade(t, merchantStrategy, v); p.Notes != (economysim.PlanNotes{}) {
		t.Errorf("no routes at all: notes = %+v, want none", p.Notes)
	}
}

// TestMerchant_TakeOnlyWithConsigns verifies the take of a consignee
// (spec-merchants.md 8.2): none unless the block consigns; for each cargo commodity
// that is not the cargo of the trip planned this tick, a limit of price * (10 000 -
// TakeMarginBP) / 10 000, for the units it lacks (TakeUnits less the free stock and
// the units on the way) within its cash.
//
// In the remote region fuel costs 300, so the limit is 300 * 9 000 / 10 000 = 270
// (19.1). With 150 wanted, 40 free and 30 inbound the take is 80; with 100 000
// money and a reserve of 0 the cash affords 370.
//
// Requirement: FR-MER-028, FR-MER-029
func TestMerchant_TakeOnlyWithConsigns(t *testing.T) {
	t.Parallel()
	importer := func() *economysim.AgentView {
		v := valley()
		v.Agent, v.Region, v.Routes, v.Rumours = 61, remote, nil, nil
		v.Commodities[0] = row(fuel, 40, 300, 1000, 1000, 300)
		v.Inbound = []economysim.InboundRow{{Commodity: fuel, Quantity: 30, Earliest: 2}}
		v.Merchant.Params.Consigns = true
		return v
	}
	p, _ := trade(t, merchantStrategy, importer())
	want := []economysim.DeliveryTake{{Commodity: fuel, Limit: 270, Quantity: 80}}
	if !reflect.DeepEqual(p.Takes, want) || p.Notes.Taken != 1 {
		t.Errorf("takes = %+v, Taken = %d, want %+v and 1", p.Takes, p.Notes.Taken, want)
	}
	off := importer()
	off.Merchant.Params.Consigns = false
	if p, _ = trade(t, merchantStrategy, off); len(p.Takes) != 0 {
		t.Errorf("a block that does not consign posted takes %+v, want none", p.Takes)
	}
	poor := importer()
	poor.Money = 270 * 5
	if p, _ = trade(t, merchantStrategy, poor); len(p.Takes) != 1 || p.Takes[0].Quantity > 5 {
		t.Errorf("takes with money for 5 units = %+v, want at most 5 units", p.Takes)
	}
	full := importer()
	full.Commodities[0].Stock = 400
	if p, _ = trade(t, merchantStrategy, full); len(p.Takes) != 0 {
		t.Errorf("takes with the want already held = %+v, want none", p.Takes)
	}
	// A merchant does not buy what it is about to send.
	plans := valley()
	plans.Merchant.Params.Consigns = true
	if p, _ = trade(t, merchantStrategy, plans); len(p.Takes) != 0 || p.Notes.Planned != 1 {
		t.Errorf("takes for the cargo of the planned trip = %+v (Planned %d), want none", p.Takes, p.Notes.Planned)
	}
}

// TestMerchant_WorkReviseAreDefault verifies that Work and Revise are the default's
// (spec-merchants.md 3.3), for a view with a merchant group and one without.
//
// Requirement: FR-MER-002
func TestMerchant_WorkReviseAreDefault(t *testing.T) {
	t.Parallel()
	var def economysim.DefaultStrategy
	for _, v := range []*economysim.AgentView{valley(), colony()} {
		src1, src2 := newWords(), newWords()
		if got, want := merchantStrategy.Work(v, rand.New(src1)), def.Work(v, rand.New(src2)); got != want || src1.n != src2.n {
			t.Errorf("Work() = %+v after %d words, want the default's %+v after %d", got, src1.n, want, src2.n)
		}
	}
	o := &economysim.OutcomeView{
		Side: economysim.Bid, Limit: 100, Quantity: 10, Filled: 4, Belief: economysim.Belief{Centre: 100, Width: 1000},
		Bounds: economysim.Bounds{Floor: 1, Ceiling: 1000}, Market: economysim.MarketReport{Price: 100}, Stock: 3, Target: 10, Draw: 5,
		Params: tunables(3).Beliefs,
	}
	if got, want := merchantStrategy.Revise(o, rand.New(newWords())), def.Revise(o, rand.New(newWords())); got != want {
		t.Errorf("Revise() = %+v, want the default's %+v", got, want)
	}
}

// TestMerchant_NoBlockTradeIsDefault verifies that in a view with no merchant group
// Trade is the default's: the same plan and the same words.
//
// Requirement: FR-MER-002
func TestMerchant_NoBlockTradeIsDefault(t *testing.T) {
	t.Parallel()
	v := valley()
	v.HasMerchant = false
	got, gotN := trade(t, merchantStrategy, v)
	want, wantN := trade(t, economysim.DefaultStrategy{}, v)
	if !reflect.DeepEqual(got, want) || gotN != wantN {
		t.Errorf("Trade() = %+v after %d words, want the default's %+v after %d", got, gotN, want, wantN)
	}
}

// TestMerchant_EqualViewsEqualPlans verifies that the strategy keeps no state
// (spec-merchants.md 3.5): equal views give equal plans whatever was planned
// before, here after calls with other views of other agents and ticks.
//
// Requirement: FR-MER-005, FR-MER-026
func TestMerchant_EqualViewsEqualPlans(t *testing.T) {
	t.Parallel()
	first, _ := trade(t, merchantStrategy, valley())
	for _, v := range []*economysim.AgentView{colony(), valley()} {
		v.Tick.Tick += 7
		trade(t, merchantStrategy, v)
	}
	again, _ := trade(t, merchantStrategy, valley())
	if !reflect.DeepEqual(first, again) {
		t.Errorf("Trade() after other calls = %+v, want the plan of the first call %+v", again, first)
	}
}

// TestMerchant_DrawsEqualDefault verifies that the strategy takes the words the
// default takes for the same orders and no others (spec-merchants.md 3.1.3,
// FR-MER-005): without a trip, those of Trade; with the trip of example A, those of
// TradeWith for its two inputs.
//
// Requirement: FR-MER-005
func TestMerchant_DrawsEqualDefault(t *testing.T) {
	t.Parallel()
	src := newWords()
	var p economysim.TradePlan
	trip := []economysim.TripInput{{Commodity: fuel, Quantity: 150, Purpose: economysim.PurposeInput}, {Commodity: haul, Quantity: 6, Purpose: economysim.PurposeInput}}
	economysim.DefaultStrategy{}.TradeWith(valley(), rand.New(src), &p, trip)
	if _, got := trade(t, merchantStrategy, valley()); got != src.n || src.n != 2 {
		t.Errorf("words taken by a planned trip = %d, want the %d of TradeWith (2: fuel and haul)", got, src.n)
	}
	v := valley()
	v.Rumours = nil
	_, base := trade(t, economysim.DefaultStrategy{}, v)
	if _, got := trade(t, merchantStrategy, v); got != base {
		t.Errorf("words taken with no trip = %d, want the default's %d", got, base)
	}
}

// BenchmarkMerchantTrade reports the cost of a Trade call that plans a trip, with
// the allocations of the call: the strategy plans on its own stack and the default
// allocates what it always does.
//
// Requirement: NFR-MER-002
func BenchmarkMerchantTrade(b *testing.B) {
	b.ReportAllocs()
	v := valley()
	src := rand.NewPCG(1, 2)
	var p economysim.TradePlan
	for b.Loop() {
		p.Intents, p.Offers, p.Takes, p.Shipments, p.Notes = p.Intents[:0], p.Offers[:0], p.Takes[:0], p.Shipments[:0], economysim.PlanNotes{}
		merchantStrategy.Trade(v, rand.New(src), &p)
	}
}

// TestMerchant_EdgesOfTheLot verifies the corners of sizing and costing that the
// worked examples do not reach: the reference price when a market has no price, the
// provisions of a trip longer than the cover, a missing vehicle bought as a tool,
// money that affords nothing, a self-carried pool shared with heavy holdings, and
// arithmetic that overflows (the candidate is dropped, never wrapped).
//
// Requirement: FR-MER-011, FR-MER-018, FR-MER-020
func TestMerchant_EdgesOfTheLot(t *testing.T) {
	t.Parallel()
	t.Run("the reference price stands in for a market with no price", func(t *testing.T) {
		t.Parallel()
		v := valley()
		v.Commodities[0].Market.Price, v.Commodities[1].Market.Price = 0, 0
		want, _ := trade(t, merchantStrategy, valley())
		got, _ := trade(t, merchantStrategy, v)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("plan with reference prices = %+v, want that of the market prices %+v", got, want)
		}
	})
	t.Run("a trip longer than the cover buys provisions", func(t *testing.T) {
		t.Parallel()
		v := colony()
		v.Routes[0].Ticks = 5 // 2 ticks beyond the cover of 3: 1 * 1 * 2 rations of provisions
		v.Rumours[0].Price, v.Rumours[0].Mean = 120, 118
		base, _ := trade(t, merchantStrategy, colony())
		p, _ := trade(t, merchantStrategy, v)
		b, ok := bidFor(p, ration)
		if !ok || p.Notes.Planned != 1 || b.Quantity <= 0 || b.Quantity == 0 && base.Notes.Planned == 0 {
			t.Errorf("ration bid = %+v (found %v), notes %+v, want the lot and the provisions planned", b, ok, p.Notes)
		}
	})
	t.Run("a missing vehicle is bought as a tool", func(t *testing.T) {
		t.Parallel()
		v := colony()
		v.Money = 100_000
		v.Commodities[2].Stock = 0
		p, _ := trade(t, merchantStrategy, v)
		if b, ok := bidFor(p, skiff); !ok || b.Quantity != 1 || b.Purpose != economysim.PurposeTool {
			t.Errorf("skiff bid = %+v (found %v), want 1 vehicle of purpose Tool", b, ok)
		}
	})
	t.Run("money that affords nothing is Unaffordable", func(t *testing.T) {
		t.Parallel()
		v := valley()
		v.Money = 10
		if p, _ := trade(t, merchantStrategy, v); p.Notes.Unaffordable != 1 || p.Notes.Planned != 0 {
			t.Errorf("notes = %+v, want Unaffordable 1", p.Notes)
		}
	})
	t.Run("heavy holdings cut the pool of a self-carried mode", func(t *testing.T) {
		t.Parallel()
		v := colony()
		v.Merchant.Goods[0].Mass = 10
		v.Commodities[0].Stock = 193 // 1 930 of the 2 000 mass: room for 70 rations (G = 1 540 still passes)
		p, _ := trade(t, merchantStrategy, v)
		if p.Notes.LotCapped != 1 || p.Notes.Planned != 1 {
			t.Errorf("notes = %+v, want the lot capped by the pool and planned", p.Notes)
		}
	})
	t.Run("overflow drops the candidate", func(t *testing.T) {
		t.Parallel()
		v := valley()
		v.Merchant.Goods[0].Mass = 1 << 62
		if p, _ := trade(t, merchantStrategy, v); p.Notes.Planned != 0 || p.Notes.Unaffordable != 1 {
			t.Errorf("notes = %+v, want the candidate dropped as Unaffordable", p.Notes)
		}
		u := valley()
		u.Upkeep = []economysim.UpkeepRow{{Commodity: fuel, Quantity: 1 << 62}}
		u.Weight = 1 << 10
		trade(t, merchantStrategy, u) // an upkeep that overflows must not panic
	})
}
