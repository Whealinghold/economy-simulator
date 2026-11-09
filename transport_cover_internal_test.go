package economysim

import (
	"reflect"
	"testing"
)

// This file holds small unit tests of the transport module that the larger tests
// reach only on one path: the shipment sums of InTransit, the feed of a load that
// needs several units, and the saturation of the feed.

// TestQueries_InTransitSumsAcrossShipments verifies spec-transport.md 7.4.2: the units
// on the way of one consignee are summed per commodity over its shipments, in
// commodity order whatever the order of the shipments' lines.
//
// Requirement: FR-TRN-034
func TestQueries_InTransitSumsAcrossShipments(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, nil)
	give(t, w, shipHome, flour, 1)
	if _, r := ask(w, shipHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar,
		Lines: []Line{{Commodity: flour, Quantity: 1}, {Commodity: grain, Quantity: 2}}}); r != "" {
		t.Fatalf("first dispatch reason = %q, want none", r)
	}
	if _, r := ask(w, shipHome2, 0, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar,
		Lines: []Line{{Commodity: grain, Quantity: 3}}}); r != "" {
		t.Fatalf("second dispatch reason = %q, want none", r)
	}
	want := []Amount{{Commodity: flour, Quantity: 1}, {Commodity: grain, Quantity: 5}}
	if got := w.InTransit(shipFar); !reflect.DeepEqual(got, want) {
		t.Errorf("InTransit(consignee) = %+v, want %+v", got, want)
	}
}

// TestDispatch_SelfCarriedFeedNeedsTwoUnits verifies spec-transport.md 3.2.2 on a load
// of two units: the least n for which the load, less the feed n units eat over k
// ticks, needs at most n units. The wagon holds 480 + 20 grain (1 500, 1 000) and 100
// fodder (200, 300), 1 700 mass in all: one unit of 1 200 is not enough even after
// the 6 fodder of one unit (1 688), two units are after 12 (1 676).
//
// Requirement: FR-TRN-009, FR-TRN-014
func TestDispatch_SelfCarriedFeedNeedsTwoUnits(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, nil)
	w.agents[shipHome].weight = 4 // a crew of 2 fields two wagons
	wagon, fodder := commodityOf(t, w, "wagon"), commodityOf(t, w, "hay")
	give(t, w, shipHome, wagon, 2)
	give(t, w, shipHome, fodder, 100)
	give(t, w, shipHome, grain, 480)
	id, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeWagon, To: regFar, ToSelf: true})
	if reason != "" {
		t.Fatalf("dispatch() reason = %q, want none", reason)
	}
	if got := w.transport.ships[id].units; got != 2 {
		t.Errorf("units = %d, want 2", got)
	}
	if got := held(t, w, shipHome, int(fodder)); got != 88 {
		t.Errorf("fodder held = %d, want 100 - 3 * 2 * 2 = 88", got)
	}
}

// TestFeedUse_Saturates verifies that the feed of a trip saturates at MaxQuantity
// instead of wrapping, for a product that overflows and for one that only exceeds
// the bound.
//
// Requirement: FR-TRN-014
func TestFeedUse_Saturates(t *testing.T) {
	t.Parallel()
	c := &shipCtx{k: 2}
	for _, tt := range []struct {
		name string
		q, n int64
		want int64
	}{
		{"exact", 3, 4, 24},
		{"over the bound", MaxQuantity, 2, MaxQuantity},
		{"overflow", MaxQuantity, 1 << 62, MaxQuantity},
	} {
		if got := c.feedUse(Line{Quantity: Quantity(tt.q)}, tt.n); got != tt.want {
			t.Errorf("%s: feedUse(%d, %d) = %d, want %d", tt.name, tt.q, tt.n, got, tt.want)
		}
	}
}
