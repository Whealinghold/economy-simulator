package otelobserver_test

import (
	"maps"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

func tick(h *harness, n int64, es ...economysim.Event) {
	h.send(economysim.TickStarted{Tick: n})
	h.send(es...)
	h.send(economysim.TickEnded{Tick: n})
}

func created() economysim.Event { return economysim.WorldCreated{Names: miniNames()} }

// TestWorldCreated_BuildsNames verifies that the adapter copies the names of
// the event: changing the event's slice afterwards changes no attribute value.
//
// Requirement: FR-OBS-052
// Guard: an event may be kept by another observer, and the adapter must not read
// through a slice the world or the consumer can still change.
func TestWorldCreated_BuildsNames(t *testing.T) {
	h := newHarness(t)
	n := miniNames()
	h.send(economysim.WorldCreated{Names: n})
	n.Commodities[1] = "changed"
	tick(h, 0, economysim.TradeExecuted{Commodity: 1, Quantity: 5})
	if v, ok := h.collect().get("market.volume", "commodity", "c1"); !ok || v != 5 {
		t.Errorf("market.volume{c1} = %d, %v after the event's names changed, want 5, true", v, ok)
	}
}

// TestAttributes_DropOutOfRange verifies that an ID outside its table, a side
// that is neither bid nor ask and a rejection reason beyond the cap record
// nothing and do not panic.
//
// Requirement: FR-OBS-052
func TestAttributes_DropOutOfRange(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	es := []economysim.Event{
		economysim.OrderPlaced{Commodity: 3, Region: 0},
		economysim.OrderPlaced{Commodity: -1, Region: 0},
		economysim.OrderPlaced{Commodity: 0, Region: 3},
		economysim.OrderPlaced{Commodity: 0, Region: -1},
		economysim.OrderPlaced{Commodity: 0, Region: 0, Side: 7},
		economysim.OrderRejected{Commodity: 3, Reason: "x"},
		economysim.OrderRejected{Commodity: -2, Reason: "x"},
		economysim.TradeExecuted{Commodity: 9, Quantity: 1},
		economysim.TradeExecuted{Commodity: 0, Quantity: 0},
		economysim.TradeExecuted{Commodity: 0, Quantity: -4},
	}
	for i := 0; i < 40; i++ { // 40 distinct reasons: 32 fit, 8 are dropped
		es = append(es, economysim.OrderRejected{Commodity: 0, Reason: economysim.Reason(string(rune('A' + i)))})
	}
	tick(h, 0, es...)
	got := h.collect()
	for _, name := range []string{"market.orders", "market.trades", "market.volume", "estate.sold"} {
		if v := got.sum(name); v != 0 {
			t.Errorf("%s = %d, want 0: every sample was out of range", name, v)
		}
	}
	if v := got.sum("market.orders.rejected"); v != 32 {
		t.Errorf("market.orders.rejected = %d, want 32: the 40 reasons are cut at 32", v)
	}
}

// TestVolume_OncePerSeriesPerTick verifies that 10 000 orders, 10 000
// rejections and 10 000 trades over 6 series each call the SDK at most once per
// non-zero series, and not at all in a tick with no event; that market.volume
// equals the volume the markets reported; and that trades and orders add up.
//
// 2 commodities x 3 regions x 2 sides = 12 order series; 2 commodities x 3 reasons = 6
// rejection series; 2 commodities = 2 trade and volume series; the same 6
// (commodity, region) series for the estate's sales, where Seller is below 3.
//
// Requirement: NFR-OBS-005, FR-OBS-045
func TestVolume_OncePerSeriesPerTick(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	es, cleared := busyTick()
	tick(h, 0, es...)
	limits := map[string]int{
		"economysim.market.orders": 12, "economysim.market.orders.rejected": 6,
		"economysim.market.trades": 2, "economysim.market.volume": 2, "economysim.estate.sold": 6,
	}
	for name, max := range limits {
		if h.calls[name] == 0 || h.calls[name] > max {
			t.Errorf("SDK calls for %s = %d, want between 1 and %d (one per series)", name, h.calls[name], max)
		}
	}
	got := h.collect()
	for name, want := range map[string]int64{"market.volume": cleared, "market.orders": 10000, "market.trades": 10000} {
		if v := got.sum(name); v != want {
			t.Errorf("%s = %d, want %d", name, v, want)
		}
	}
	before := maps.Clone(h.calls)
	tick(h, 1)
	delete(before, "economysim.world.step.duration") // an empty step still has a duration
	delete(h.calls, "economysim.world.step.duration")
	if !maps.Equal(before, h.calls) {
		t.Errorf("SDK calls after an empty tick = %v, want %v: none", h.calls, before)
	}
}

// busyTick is 10 000 orders, rejections and trades over the series of the
// harness' names, and the MarketCleared events that report the traded volume,
// which it returns.
func busyTick() (es []economysim.Event, volume int64) {
	reasons := []economysim.Reason{"a", "b", "c"}
	var cleared [2]int64
	for i := 0; i < 10000; i++ {
		c, r := economysim.CommodityID(i%2), economysim.RegionID(i%3)
		es = append(es,
			economysim.OrderPlaced{Commodity: c, Region: r, Side: economysim.Side(i % 2)},
			economysim.OrderRejected{Commodity: c, Reason: reasons[i%3]},
			economysim.TradeExecuted{Commodity: c, Region: r, Seller: economysim.AgentID(i % 5), Quantity: 2},
		)
		cleared[c] += 2
	}
	es = append(es,
		economysim.MarketCleared{Commodity: 0, Volume: economysim.Quantity(cleared[0])},
		economysim.MarketCleared{Commodity: 1, Volume: economysim.Quantity(cleared[1])})
	return es, cleared[0] + cleared[1]
}

// TestEstateSold_SellerBelowRegionCount verifies that estate.sold sums the
// quantity of the trades whose Seller is below the region count (an estate's
// account number is its region's, N-310 item 2), and nothing else.
//
// Sellers 0 and 2 are estates (3 regions), seller 3 and 90 are agents: 4 + 6 = 10.
//
// Requirement: FR-OBS-050
func TestEstateSold_SellerBelowRegionCount(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0,
		economysim.TradeExecuted{Commodity: 0, Region: 0, Seller: 0, Quantity: 4},
		economysim.TradeExecuted{Commodity: 0, Region: 2, Seller: 2, Quantity: 6},
		economysim.TradeExecuted{Commodity: 0, Region: 1, Seller: 3, Quantity: 9},
		economysim.TradeExecuted{Commodity: 0, Region: 1, Seller: 90, Quantity: 9})
	got := h.collect()
	if v := got.sum("estate.sold"); v != 10 {
		t.Errorf("estate.sold = %d, want 10", v)
	}
	if v, ok := got.get("estate.sold", "commodity", "c0", "region", "r2"); !ok || v != 6 {
		t.Errorf("estate.sold{c0, r2} = %d, %v, want 6, true", v, ok)
	}
	if v := got.sum("market.volume"); v != 28 {
		t.Errorf("market.volume = %d, want 28: every trade counts", v)
	}
}

// TestOrders_AttributesAndSettleOutsideStep verifies the attributes of an order
// series and that an event outside a step is flushed at once, without a
// TickEnded.
//
// Requirement: FR-OBS-052, FR-OBS-055
func TestOrders_AttributesAndSettleOutsideStep(t *testing.T) {
	h := newHarness(t)
	h.send(created(),
		economysim.OrderPlaced{Commodity: 2, Region: 1, Side: economysim.Ask},
		economysim.OrderRejected{Commodity: 1, Reason: economysim.ReasonOverflow})
	got := h.collect()
	if v, ok := got.get("market.orders", "commodity", "c2", "region", "r1", "side", "ask"); !ok || v != 1 {
		t.Errorf("market.orders{c2, r1, ask} = %d, %v, want 1, true: %v", v, ok, got["economysim.market.orders"])
	}
	if v, ok := got.get("market.orders.rejected", "commodity", "c1", "reason", "overflow"); !ok || v != 1 {
		t.Errorf("market.orders.rejected{c1, overflow} = %d, %v, want 1, true: %v", v, ok, got["economysim.market.orders.rejected"])
	}
}

// TestStepDuration_FakeClock verifies that a step lasts the time between the
// two clock reads and that the clock is read exactly twice per step.
//
// Requirement: FR-OBS-058
func TestStepDuration_FakeClock(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0)
	tick(h, 1)
	if h.clock.reads != 4 {
		t.Errorf("clock reads after 2 steps = %d, want 4", h.clock.reads)
	}
	if v, ok := h.collect()["economysim.world.step.duration"][""]; !ok || v != 2 {
		t.Errorf("step.duration count = %d, %v, want 2, true", v, ok)
	}
}

// TestStepFailed_NoDurationOneFailed verifies that a failed step records no
// duration and adds one to step.failed, so failed plus durations equals the
// TickEnded count, and that it still flushes what it saw.
//
// Requirement: FR-OBS-058
func TestStepFailed_NoDurationOneFailed(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0)
	h.send(economysim.TickStarted{Tick: 1}, economysim.OrderPlaced{Commodity: 0, Region: 0}, economysim.TickEnded{Tick: 1, Failed: true})
	tick(h, 2)
	got := h.collect()
	if v := got["economysim.world.step.duration"][""]; v != 2 {
		t.Errorf("durations = %d, want 2", v)
	}
	if v := got["economysim.world.step.failed"][""]; v != 1 {
		t.Errorf("step.failed = %d, want 1", v)
	}
	if v := got.sum("market.orders"); v != 1 {
		t.Errorf("market.orders = %d, want 1: a failed step flushes what it saw", v)
	}
}

// TestWorldCreated_ForgetsThePreviousWorld verifies that a second WorldCreated
// with a different names table rebuilds the attributes.
//
// Guard: the accumulators are sized from the names, so keeping the old ones would
// index a smaller or larger table.
func TestWorldCreated_ForgetsThePreviousWorld(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	h.send(economysim.WorldCreated{Names: economysim.Names{Commodities: []string{"z"}, Regions: []string{"y"}}})
	tick(h, 0, economysim.OrderPlaced{Commodity: 0, Region: 0, Side: economysim.Bid}, economysim.OrderPlaced{Commodity: 1, Region: 0})
	if v, ok := h.collect().get("market.orders", "commodity", "z", "region", "y", "side", "bid"); !ok || v != 1 {
		t.Errorf("market.orders{z, y, bid} = %d, %v, want 1, true", v, ok)
	}
}

// TestSeries_EveryCellHasItsAttributes verifies that each cell of the dense
// accumulators is recorded under its own commodity, region, side and reason: it
// sends a different count to each of the 3 x 3 x 2 order series, the 3 x 3
// rejection series and the 3 x 3 estate series, and reads each back by its
// attributes.
//
// Guard: a cell is decoded back to attributes by mixed radix, and a wrong radix
// moves a count to another series without changing any total.
func TestSeries_EveryCellHasItsAttributes(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, cellEvents()...)
	got := h.collect()
	for c := 0; c < 3; c++ {
		cn := "c" + string(rune('0'+c))
		for r := 0; r < 3; r++ {
			rn := "r" + string(rune('0'+r))
			for s := 0; s < 2; s++ {
				want := int64(1 + c*6 + r*2 + s)
				if v, _ := got.get("market.orders", "commodity", cn, "region", rn, "side", cellSides[s]); v != want {
					t.Errorf("market.orders{%s, %s, %s} = %d, want %d", cn, rn, cellSides[s], v, want)
				}
			}
			want := int64(1 + c*3 + r)
			if v, _ := got.get("market.orders.rejected", "commodity", cn, "reason", cellReasons[r]); v != want {
				t.Errorf("market.orders.rejected{%s, %s} = %d, want %d", cn, cellReasons[r], v, want)
			}
			if v, _ := got.get("estate.sold", "commodity", cn, "region", rn); v != want {
				t.Errorf("estate.sold{%s, %s} = %d, want %d", cn, rn, v, want)
			}
		}
	}
}

var (
	cellReasons = []string{"a", "b", "c"}
	cellSides   = []string{"bid", "ask"}
)

// cellEvents sends a different count to each series: 1+6c+2r+s orders for
// (c, r, side s), and 1+3c+r rejections and estate trades of quantity 1.
func cellEvents() (es []economysim.Event) {
	for c := 0; c < 3; c++ {
		for r := 0; r < 3; r++ {
			cid, rid := economysim.CommodityID(c), economysim.RegionID(r)
			for s := 0; s < 2; s++ {
				for n := 0; n < 1+c*6+r*2+s; n++ {
					es = append(es, economysim.OrderPlaced{Commodity: cid, Region: rid, Side: economysim.Side(s)})
				}
			}
			for n := 0; n < 1+c*3+r; n++ {
				es = append(es,
					economysim.OrderRejected{Commodity: cid, Reason: economysim.Reason(cellReasons[r])},
					economysim.TradeExecuted{Commodity: cid, Region: rid, Seller: economysim.AgentID(r), Quantity: 1})
			}
		}
	}
	return es
}
