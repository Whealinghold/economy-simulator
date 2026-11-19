package indicator_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// The window of spec-indicators.md 18.11, the lost cargo: a shipment of 10 grain leaves
// alpha with a loss share of 2 500 and arrives with 2 lost and 8 delivered, and a store
// strike takes 3 loaves at alpha, in tick 10 of the valley world.

// lossEvents are the events of that tick: the dispatch, the arrival, the contract that
// reports the same lost units, the store strike and the boundary flows of the reason
// loss, 2 grain and 3 loaves.
func lossEvents() []economysim.Event {
	return []economysim.Event{
		economysim.ShipmentDispatched{Tick: 10, Region: alpha, To: beta, LossBP: 2_500, Units: 10, Lines: []economysim.Amount{{Commodity: grain, Quantity: 10}}},
		economysim.ShipmentArrived{Tick: 10, Region: alpha, To: beta, Lines: []economysim.ArrivedLine{{Commodity: grain, Shipped: 10, Delivered: 8, Lost: 2}}},
		economysim.ContractSettled{Tick: 10, Region: alpha, To: beta, Lost: 2},
		economysim.StoreLoss{Tick: 10, Agent: 3, Region: alpha, Class: 0, Stores: 1, ShareBP: 1_800, Lost: []economysim.Amount{{Commodity: loaf, Quantity: 3}}},
		goodsFlow(10, alpha, economysim.ReasonLoss, grain, 2),
		goodsFlow(10, alpha, economysim.ReasonLoss, loaf, 3),
	}
}

// lossWindow closes a window [10, 11) over the events given.
func lossWindow(t *testing.T, events ...economysim.Event) indicator.Window {
	t.Helper()
	tl := indicator.NewTally()
	openValley(t, tl, 10, [4]int64{})
	step(tl, 10, events...)
	w, err := tl.Close(valleySnap(11, [4]int64{}))
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	return w
}

// TestLosses_UnitsValueStruckBySource verifies the worked row of spec-indicators.md
// 18.11: the units and their value by source, region and commodity at alpha's table,
// the strikes counted apart, and the total against the boundary flow.
//
// Arithmetic: Units = 2 + 3 = 5; Value = 2 * 10 (grain) + 3 * 25 (loaf) = 95; Struck is
// 1 shipment and 1 store; the boundary flow loss of the window is 2 + 3 = 5 units, so
// the gap is 0.
//
// Requirement: FR-IND-059
func TestLosses_UnitsValueStruckBySource(t *testing.T) {
	t.Parallel()
	w := lossWindow(t, lossEvents()...)
	got, err := indicator.Losses(w, valleyTable(t, w, alphaP, betaP))
	if err != nil {
		t.Fatalf("Losses() error = %v, want nil", err)
	}
	want := indicator.LossesResult{
		ConfigHash: w.ConfigHash, From: 10, To: 11,
		Rows: []indicator.LossRow{
			{Source: indicator.SourceRoute, Region: alpha, Commodity: grain, Units: 2, Value: 20},
			{Source: indicator.SourceStore, Region: alpha, Commodity: loaf, Units: 3, Value: 75},
		},
		Struck: []indicator.StruckRow{{Source: indicator.SourceRoute, Region: alpha, Count: 1}, {Source: indicator.SourceStore, Region: alpha, Count: 1}},
		Units:  5, Value: 95, Boundary: 5, Gap: 0,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Losses() = %+v, want %+v", got, want)
	}
}

// TestLosses_ContractLostNotAddedTwice verifies that the Lost of a ContractSettled,
// the same units as a ShipmentArrived line, adds nothing: the window with and without
// the contract event has the same rows.
//
// Requirement: FR-IND-059
func TestLosses_ContractLostNotAddedTwice(t *testing.T) {
	t.Parallel()
	with := lossWindow(t, lossEvents()...)
	without := lossWindow(t, slicesWithout(lossEvents(), 2)...)
	a, _ := indicator.Losses(with, valleyTable(t, with, alphaP, betaP))
	b, _ := indicator.Losses(without, valleyTable(t, without, alphaP, betaP))
	if !reflect.DeepEqual(a, b) || a.Units != 5 {
		t.Errorf("Losses with the contract event = %+v, without = %+v, want the same 5 units", a, b)
	}
}

// slicesWithout returns the events with the one at index i left out.
func slicesWithout(events []economysim.Event, i int) []economysim.Event {
	return append(append([]economysim.Event(nil), events[:i]...), events[i+1:]...)
}

// TestLosses_StrikeWithNoUnitIsInStruckNotUnits verifies that a strike that took no
// unit, a shipment whose line rounded down and a store with an empty Lost, is counted
// in Struck and adds no row and no unit.
//
// Requirement: FR-IND-059
func TestLosses_StrikeWithNoUnitIsInStruckNotUnits(t *testing.T) {
	t.Parallel()
	w := lossWindow(t,
		economysim.ShipmentDispatched{Tick: 10, Region: alpha, To: beta, LossBP: 100, Units: 1, Lines: []economysim.Amount{{Commodity: grain, Quantity: 1}}},
		economysim.ShipmentArrived{Tick: 10, Region: alpha, To: beta, Lines: []economysim.ArrivedLine{{Commodity: grain, Shipped: 1, Delivered: 1}}},
		economysim.StoreLoss{Tick: 10, Agent: 3, Region: beta, Class: 1, Stores: 2})
	got, err := indicator.Losses(w, valleyTable(t, w, alphaP, betaP))
	if err != nil {
		t.Fatalf("Losses() error = %v, want nil", err)
	}
	wantStruck := []indicator.StruckRow{{Source: indicator.SourceRoute, Region: alpha, Count: 1}, {Source: indicator.SourceStore, Region: beta, Count: 2}}
	if len(got.Rows) != 0 || got.Units != 0 || got.Value != 0 || !reflect.DeepEqual(got.Struck, wantStruck) {
		t.Errorf("Losses() = %+v, want no rows or units and the strikes %+v", got, wantStruck)
	}
}

// TestLosses_StruckSumsTheLanesOfAnOrigin verifies that the shipments struck are
// counted by origin over every destination and mode, so two lanes of alpha are one row.
//
// Requirement: FR-IND-059
func TestLosses_StruckSumsTheLanesOfAnOrigin(t *testing.T) {
	t.Parallel()
	line := []economysim.Amount{{Commodity: grain, Quantity: 1}}
	w := lossWindow(t,
		economysim.ShipmentDispatched{Tick: 10, Region: alpha, To: beta, Mode: 0, LossBP: 10, Units: 1, Lines: line},
		economysim.ShipmentDispatched{Tick: 10, Region: alpha, To: beta, Mode: 1, LossBP: 20, Units: 1, Lines: line},
		economysim.ShipmentDispatched{Tick: 10, Region: alpha, To: beta, Mode: 1, Units: 1, Lines: line})
	got, err := indicator.Losses(w, valleyTable(t, w, alphaP, betaP))
	if want := []indicator.StruckRow{{Source: indicator.SourceRoute, Region: alpha, Count: 2}}; err != nil || !reflect.DeepEqual(got.Struck, want) {
		t.Errorf("Losses() struck = %+v, %v, want %+v (the third shipment had no loss share)", got.Struck, err, want)
	}
}

// TestLosses_AccountLostIsRouteUnits verifies that what a self-carried trip lost from
// the traveller's account is a route loss at the origin, beside the lines' Lost.
//
// Arithmetic: 3 lost from the lines plus 1 from the account, both grain at alpha: 4
// units at 10.
//
// Requirement: FR-IND-059
func TestLosses_AccountLostIsRouteUnits(t *testing.T) {
	t.Parallel()
	w := lossWindow(t,
		economysim.ShipmentArrived{Tick: 10, Region: alpha, To: beta, Lines: []economysim.ArrivedLine{{Commodity: grain, Shipped: 6, Delivered: 3, Lost: 3}},
			AccountLost: []economysim.Amount{{Commodity: grain, Quantity: 1}}},
		goodsFlow(10, alpha, economysim.ReasonLoss, grain, 4))
	got, err := indicator.Losses(w, valleyTable(t, w, alphaP, betaP))
	if err != nil || got.Units != 4 || got.Value != 40 || got.Gap != 0 || len(got.Rows) != 1 {
		t.Errorf("Losses() = %+v, %v, want 4 units at 40 in one row and no gap", got, err)
	}
}

// TestLosses_TotalEqualsBoundaryFlowLoss verifies the check of row 25: the gap is the
// units of the rows less the boundary flow loss of the window, 0 when they agree and
// the difference when a flow is missing, and a window with no loss at all has none.
//
// Requirement: FR-IND-059
func TestLosses_TotalEqualsBoundaryFlowLoss(t *testing.T) {
	t.Parallel()
	events := lossEvents()
	for _, tt := range []struct {
		name          string
		events        []economysim.Event
		boundary, gap int64
	}{
		{"flows agree", events, 5, 0},
		{"a flow missing", events[:5], 2, 3},
		{"no loss", nil, 0, 0},
	} {
		w := lossWindow(t, tt.events...)
		got, err := indicator.Losses(w, valleyTable(t, w, alphaP, betaP))
		if err != nil || got.Boundary != tt.boundary || got.Gap != tt.gap {
			t.Errorf("%s: Losses() boundary %d gap %d error %v, want %d and %d", tt.name, got.Boundary, got.Gap, err, tt.boundary, tt.gap)
		}
	}
}

// TestTally_ObserveStoreLoss verifies the rows of the tally: StoreLoss summed by
// (region, class, commodity) and by (region, class), the Lost and AccountLost of an
// arrival, and the shipments struck; and that a bad amount or commodity makes the
// window refuse to close.
//
// Requirement: FR-IND-059
func TestTally_ObserveStoreLoss(t *testing.T) {
	t.Parallel()
	w := lossWindow(t,
		economysim.StoreLoss{Tick: 10, Agent: 3, Region: alpha, Class: 0, Stores: 1, Lost: []economysim.Amount{{Commodity: loaf, Quantity: 3}}},
		economysim.StoreLoss{Tick: 10, Agent: 4, Region: alpha, Class: 0, Stores: 2, Lost: []economysim.Amount{{Commodity: loaf, Quantity: 4}, {Commodity: wood, Quantity: 1}}},
		economysim.ShipmentDispatched{Tick: 10, Region: alpha, To: beta, LossBP: 1, Lines: []economysim.Amount{{Commodity: grain, Quantity: 1}}})
	wantL := []indicator.LossTally{{Region: alpha, Class: 0, Commodity: loaf, Units: 7}, {Region: alpha, Class: 0, Commodity: wood, Units: 1}}
	wantS := []indicator.StrikeTally{{Region: alpha, Class: 0, Events: 2, Stores: 3}}
	if !reflect.DeepEqual(w.StoreLosses, wantL) || !reflect.DeepEqual(w.StoreStrikes, wantS) {
		t.Errorf("StoreLosses = %+v and StoreStrikes = %+v, want %+v and %+v", w.StoreLosses, w.StoreStrikes, wantL, wantS)
	}
	if len(w.Dispatches) != 1 || w.Dispatches[0].Struck != 1 {
		t.Errorf("Dispatches = %+v, want one row with 1 struck", w.Dispatches)
	}
	for name, e := range map[string]economysim.Event{
		"negative units":    economysim.StoreLoss{Tick: 10, Region: alpha, Stores: 1, Lost: []economysim.Amount{{Commodity: loaf, Quantity: -1}}},
		"unknown commodity": economysim.StoreLoss{Tick: 10, Region: alpha, Stores: 1, Lost: []economysim.Amount{{Commodity: 9, Quantity: 1}}},
		"unknown region":    economysim.StoreLoss{Tick: 10, Region: 9, Stores: 1},
		"negative class":    economysim.StoreLoss{Tick: 10, Region: alpha, Class: -1, Stores: 1},
		"account of a bad commodity": economysim.ShipmentArrived{Tick: 10, Region: alpha, To: beta,
			Lines: []economysim.ArrivedLine{{Commodity: grain, Shipped: 1, Delivered: 1}}, AccountLost: []economysim.Amount{{Commodity: 9, Quantity: 1}}},
		"negative lost line": economysim.ShipmentArrived{Tick: 10, Region: alpha, To: beta, Lines: []economysim.ArrivedLine{{Commodity: grain, Shipped: 1, Lost: -1}}},
	} {
		tl := indicator.NewTally()
		openValley(t, tl, 10, [4]int64{})
		step(tl, 10, e)
		if _, err := tl.Close(valleySnap(11, [4]int64{})); !errors.Is(err, indicator.ErrWindowOrder) {
			t.Errorf("%s: Close() error = %v, want ErrWindowOrder", name, err)
		}
	}
}

// TestLosses_RefusesAWrongTableAndARowOutOfRange verifies the errors of the preset: a
// table of another world, a row outside the names table and sums past int64.
//
// Requirement: FR-IND-059
func TestLosses_RefusesAWrongTableAndARowOutOfRange(t *testing.T) {
	t.Parallel()
	w := lossWindow(t, lossEvents()...)
	good := valleyTable(t, w, alphaP, betaP)
	other, err := indicator.NewPrices([32]byte{1}, w.From, w.To, 2, 4, make([]economysim.Price, 8))
	if err != nil {
		t.Fatalf("NewPrices() error = %v, want nil", err)
	}
	with := func(f func(w *indicator.Window)) indicator.Window {
		c := w
		c.Arrivals = append([]indicator.ArrivalTally(nil), w.Arrivals...)
		c.StoreLosses = append([]indicator.LossTally(nil), w.StoreLosses...)
		c.StoreStrikes = append([]indicator.StrikeTally(nil), w.StoreStrikes...)
		c.Flows = append([]economysim.FlowTotal(nil), w.Flows...)
		f(&c)
		return c
	}
	for _, tt := range []struct {
		name string
		w    indicator.Window
		p    indicator.Prices
		want error
	}{
		{"a table of another world", w, other, indicator.ErrConfigMismatch},
		{"an arrival from outside the regions", with(func(c *indicator.Window) { c.Arrivals[0].Origin = 9 }), good, indicator.ErrRange},
		{"a store loss of an unknown commodity", with(func(c *indicator.Window) { c.StoreLosses[0].Commodity = 9 }), good, indicator.ErrRange},
		{"a store strike outside the regions", with(func(c *indicator.Window) { c.StoreStrikes[0].Region = 9 }), good, indicator.ErrRange},
		{"a hand-built window with no commodities", indicator.Window{Regions: 1, Arrivals: []indicator.ArrivalTally{{Lost: 1}}}, mustTable(t, 1, 0), indicator.ErrRange},
		{"a loss flow outside the regions", with(func(c *indicator.Window) { c.Flows[0].Region = 9 }), good, indicator.ErrRange},
		{"a dispatch from outside the regions", with(func(c *indicator.Window) { c.Dispatches = []indicator.DispatchTally{{Origin: 9, Struck: 1}} }), good, indicator.ErrRange},
		{"route units past int64", with(func(c *indicator.Window) { c.Arrivals[0].Lost, c.Arrivals[0].AccountLost = math.MaxInt64, 1 }), good, indicator.ErrOverflow},
		{"a value past int64", with(func(c *indicator.Window) { c.StoreLosses[0].Units = math.MaxInt64 }), good, indicator.ErrOverflow},
	} {
		if _, err := indicator.Losses(tt.w, tt.p); !errors.Is(err, tt.want) {
			t.Errorf("%s: Losses() error = %v, want %v", tt.name, err, tt.want)
		}
	}
}

// mustTable returns an empty price table of the given shape for the zero hash.
func mustTable(t *testing.T, regions, commodities int) indicator.Prices {
	t.Helper()
	p, err := indicator.NewPrices([32]byte{}, 0, 0, regions, commodities, make([]economysim.Price, regions*commodities))
	if err != nil {
		t.Fatalf("NewPrices(%d, %d) error = %v, want nil", regions, commodities, err)
	}
	return p
}

// TestDeliveredPrice_AccountLostOnlyRowIsNoLane verifies that a commodity that the
// traveller's own account lost at an arrival, and that no line of the shipment
// carried, gets no lane of the delivered price: it never moved on the lane.
//
// Fix: plan 3-rsk final review. The AccountLost row of such a commodity has Shipped 0,
// and DeliveredPrice added a lane, with the origin's price, for every row of the lane.
func TestDeliveredPrice_AccountLostOnlyRowIsNoLane(t *testing.T) {
	t.Parallel()
	w := lossWindow(t,
		economysim.ShipmentDispatched{Tick: 10, Region: alpha, To: beta, Units: 10, Lines: []economysim.Amount{{Commodity: grain, Quantity: 10}}},
		economysim.ShipmentArrived{Tick: 10, Region: alpha, To: beta, Lines: []economysim.ArrivedLine{{Commodity: grain, Shipped: 10, Delivered: 10}},
			AccountLost: []economysim.Amount{{Commodity: loaf, Quantity: 1}}})
	got, err := indicator.DeliveredPrice(w, nil, valleyTable(t, w, alphaP, betaP))
	if err != nil || len(got.Lanes) != 1 || got.Lanes[0].Commodity != grain {
		t.Errorf("DeliveredPrice().Lanes = %+v, %v, want the one lane of grain (loaf only left the account)", got.Lanes, err)
	}
}
