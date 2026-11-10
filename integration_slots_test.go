package economysim_test

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The slots of spec-integration.md 5 are tested through the public calls, on the
// packs that have the module of each: the mechanism of a slot is its owner's and
// is tested in the owner's plan; what is tested here is the call (the order of the
// checks, atomicity, the event, the default that clears, the list Slots reads back).

// slotLayers are the packs of the slot tests. The regions of the first four are
// north (0) and a second region (1), an outside node in the external pack; the
// recipes are sorted, so cut_fuel is 0 and grow_staple 3; the commodities are
// sorted, so fuel is 0 and staple 5.
var (
	slotsNone      = packOne
	slotsExternal  = packExternal()
	slotsTransport = packTwoTransport
	slotsFlows     = packOneFaucetSunk
	slotsEndow     = append(slices.Clone(packOne), "overlay/resources.toml", "overlay/population.toml")
)

const (
	slotNorth, slotOther economysim.RegionID    = 0, 1
	slotFuel, slotStaple economysim.CommodityID = 0, 5
	slotGrowStaple       economysim.RecipeID    = 3
)

// slotWorld builds the world of a pack with a recorder; the audit is on.
func slotWorld(t testing.TB, layers []string) (*economysim.World, *[]economysim.Event) {
	t.Helper()
	var events []economysim.Event
	w, err := economysim.NewWorld(loadValidPack(t, layers...), economysim.WithSeed(7), economysim.WithAudit(),
		economysim.WithObserver(economysim.ObserverFunc(func(e economysim.Event) { events = append(events, e) })))
	if err != nil {
		t.Fatalf("NewWorld(%q) = %v, want nil", layers, err)
	}
	return w, &events
}

// slotBytes is the checkpoint of w, for "changes nothing" by bytes (plan 2-17 D558).
func slotBytes(t testing.TB, w *economysim.World) []byte {
	t.Helper()
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	b, err := cp.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() = %v, want nil", err)
	}
	return b
}

// slotRow is one setter of the conformance table: the pack that has its module, a
// call that sets two keys (or one) to non-default values, the call that sets them
// back, the failing calls with the sentinel each returns, and what reads the held
// keys back and what the events are.
type slotRow struct {
	name   string
	layers []string
	// off is a pack with the module off, nil for a slot that needs no module.
	off []string
	// module is the name ErrModuleOff must carry.
	set, clear, empty func(w *economysim.World) error
	// held is the number of keys of the SlotState list this setter fills.
	held func(s economysim.SlotState) int
	bad  []slotBad
	// events says an event is one this setter emits, keys the ordering key of one.
	events func(e economysim.Event) (key []int64, ok bool)
	// reversed is a call that gives the keys in descending order (nil for a row of
	// one key).
	reversed func(w *economysim.World) error
	// keys is how many events a call to set emits.
	keys int
	// oneOff says the call is an adjustment, not a held value: calling it again
	// changes the stock again.
	oneOff bool
}

// slotBad is a failing call.
type slotBad struct {
	name string
	call func(w *economysim.World) error
	want error
	// plain says the error is not a *ValidationError (a module that is off).
}

func slotRows() []slotRow {
	big := int64(economysim.MaxShockBP) + 1
	yield := func(c ...economysim.YieldShock) func(*economysim.World) error {
		return func(w *economysim.World) error { return w.SetYieldShocks(c) }
	}
	edge := func(c ...economysim.EdgeStatus) func(*economysim.World) error {
		return func(w *economysim.World) error { return w.SetEdgeStatuses(c) }
	}
	transit := func(c ...economysim.TransitFactor) func(*economysim.World) error {
		return func(w *economysim.World) error { return w.SetTransitFactors(c) }
	}
	flow := func(c ...economysim.FlowScale) func(*economysim.World) error {
		return func(w *economysim.World) error { return w.SetFlowScales(c) }
	}
	birth := func(c ...economysim.RegionScale) func(*economysim.World) error {
		return func(w *economysim.World) error { return w.SetBirthScales(c) }
	}
	death := func(c ...economysim.RegionScale) func(*economysim.World) error {
		return func(w *economysim.World) error { return w.SetDeathScales(c) }
	}
	capacity := func(c ...economysim.EndowmentScale) func(*economysim.World) error {
		return func(w *economysim.World) error { return w.SetCapacityScales(c) }
	}
	regen := func(c ...economysim.EndowmentScale) func(*economysim.World) error {
		return func(w *economysim.World) error { return w.SetRegenScales(c) }
	}
	adjust := func(c ...economysim.EndowmentAdjustment) func(*economysim.World) error {
		return func(w *economysim.World) error { return w.AdjustEndowments(c) }
	}
	outside := func(c ...economysim.OutsideShock) func(*economysim.World) error {
		return func(w *economysim.World) error { return w.SetOutsideShocks(c) }
	}
	return []slotRow{
		{
			name: "SetYieldShocks", layers: slotsExternal, keys: 2,
			set:      yield(economysim.YieldShock{Region: slotNorth, Recipe: 0, BP: 3_000}, economysim.YieldShock{Region: slotNorth, Recipe: slotGrowStaple, BP: 5_000}),
			reversed: yield(economysim.YieldShock{Region: slotNorth, Recipe: slotGrowStaple, BP: 5_000}, economysim.YieldShock{Region: slotNorth, Recipe: 0, BP: 3_000}),
			clear:    yield(economysim.YieldShock{Region: slotNorth, Recipe: 0, BP: 10_000}, economysim.YieldShock{Region: slotNorth, Recipe: slotGrowStaple, BP: 10_000}),
			empty:    yield(),
			held:     func(s economysim.SlotState) int { return len(s.YieldShocks) },
			events: func(e economysim.Event) ([]int64, bool) {
				y, ok := e.(economysim.YieldShocked)
				return []int64{int64(y.Region), int64(y.Recipe)}, ok
			},
			bad: []slotBad{
				{"region below range", yield(economysim.YieldShock{Region: economysim.NoRegion, BP: 1}), economysim.ErrUnknownRegion},
				{"region above range", yield(economysim.YieldShock{Region: 9, BP: 1}), economysim.ErrUnknownRegion},
				{"outside node", yield(economysim.YieldShock{Region: slotOther, BP: 1}), economysim.ErrOutsideNode},
				{"recipe unknown", yield(economysim.YieldShock{Region: slotNorth, Recipe: 99, BP: 1}), economysim.ErrUnknownRecipe},
				{"above MaxYieldBP", yield(economysim.YieldShock{Region: slotNorth, BP: economysim.MaxYieldBP + 1}), economysim.ErrOutOfBounds},
				{"below 0", yield(economysim.YieldShock{Region: slotNorth, BP: -1}), economysim.ErrOutOfBounds},
				{"key twice", yield(economysim.YieldShock{Region: slotNorth, BP: 1}, economysim.YieldShock{Region: slotNorth, BP: 2}), economysim.ErrDuplicateID},
				// The first failure in the order given: the repeat at entry 1 comes before the bound at entry 2.
				{"repeat before a later bound", yield(economysim.YieldShock{BP: 1}, economysim.YieldShock{BP: 2}, economysim.YieldShock{Recipe: 1, BP: -1}), economysim.ErrDuplicateID},
				{"a bad entry after good ones", yield(economysim.YieldShock{BP: 1}, economysim.YieldShock{Recipe: 1, BP: -1}), economysim.ErrOutOfBounds},
			},
		},
		{
			name: "SetEdgeStatuses", layers: slotsExternal, keys: 2,
			set: edge(economysim.EdgeStatus{From: slotNorth, To: slotOther, Closed: true, FactorBP: 10_000},
				economysim.EdgeStatus{From: slotOther, To: slotNorth, FactorBP: 20_000}),
			reversed: edge(economysim.EdgeStatus{From: slotOther, To: slotNorth, FactorBP: 20_000},
				economysim.EdgeStatus{From: slotNorth, To: slotOther, Closed: true, FactorBP: 10_000}),
			clear: edge(economysim.EdgeStatus{From: slotNorth, To: slotOther, FactorBP: 10_000},
				economysim.EdgeStatus{From: slotOther, To: slotNorth, FactorBP: 10_000}),
			empty: edge(),
			held:  func(s economysim.SlotState) int { return len(s.EdgeStatuses) },
			events: func(e economysim.Event) ([]int64, bool) {
				c, ok := e.(economysim.EdgeStatusChanged)
				return []int64{int64(c.From), int64(c.To)}, ok
			},
			bad: []slotBad{
				{"no such edge", edge(economysim.EdgeStatus{From: slotNorth, To: slotNorth, FactorBP: 10_000}), economysim.ErrUnknownEdge},
				{"region unknown", edge(economysim.EdgeStatus{From: 9, To: slotNorth, FactorBP: 10_000}), economysim.ErrUnknownEdge},
				{"factor 0", edge(economysim.EdgeStatus{From: slotNorth, To: slotOther}), economysim.ErrOutOfBounds},
				{"factor above MaxEdgeFactorBP", edge(economysim.EdgeStatus{From: slotNorth, To: slotOther, FactorBP: economysim.MaxEdgeFactorBP + 1}), economysim.ErrOutOfBounds},
				{"edge twice", edge(economysim.EdgeStatus{From: slotNorth, To: slotOther, FactorBP: 10_000}, economysim.EdgeStatus{From: slotNorth, To: slotOther, Closed: true, FactorBP: 10_000}), economysim.ErrDuplicateID},
			},
		},
		{
			name: "SetTransitFactors", layers: slotsTransport, off: slotsNone, keys: 2,
			set:      transit(economysim.TransitFactor{From: 0, To: 1, Mode: 0, BP: 20_000}, economysim.TransitFactor{From: 1, To: 0, Mode: 1, BP: 0}),
			reversed: transit(economysim.TransitFactor{From: 1, To: 0, Mode: 1, BP: 0}, economysim.TransitFactor{From: 0, To: 1, Mode: 0, BP: 20_000}),
			clear:    transit(economysim.TransitFactor{From: 0, To: 1, Mode: 0, BP: 10_000}, economysim.TransitFactor{From: 1, To: 0, Mode: 1, BP: 10_000}),
			empty:    transit(),
			held:     func(s economysim.SlotState) int { return len(s.TransitFactors) },
			events: func(e economysim.Event) ([]int64, bool) {
				c, ok := e.(economysim.TransitFactorChanged)
				return []int64{int64(c.From), int64(c.To), int64(c.Mode)}, ok
			},
			bad: []slotBad{
				{"region unknown", transit(economysim.TransitFactor{From: 9, To: 1, BP: 1}), economysim.ErrUnknownRegion},
				{"to unknown", transit(economysim.TransitFactor{From: 0, To: economysim.NoRegion, BP: 1}), economysim.ErrUnknownRegion},
				{"same region", transit(economysim.TransitFactor{From: 1, To: 1, BP: 1}), economysim.ErrSameRegion},
				{"mode unknown", transit(economysim.TransitFactor{From: 0, To: 1, Mode: 9, BP: 1}), economysim.ErrUnknownMode},
				{"above MaxTransitFactorBP", transit(economysim.TransitFactor{From: 0, To: 1, BP: economysim.MaxTransitFactorBP + 1}), economysim.ErrOutOfBounds},
				{"below 0", transit(economysim.TransitFactor{From: 0, To: 1, BP: -1}), economysim.ErrOutOfBounds},
				{"key twice", transit(economysim.TransitFactor{From: 0, To: 1, BP: 1}, economysim.TransitFactor{From: 0, To: 1, BP: 2}), economysim.ErrDuplicateID},
			},
		},
		{
			name: "SetFlowScales", layers: slotsFlows, off: slotsNone, keys: 2,
			set:      flow(economysim.FlowScale{Flow: "charge", BP: 0}, economysim.FlowScale{Flow: "stipend", BP: 20_000}),
			reversed: flow(economysim.FlowScale{Flow: "stipend", BP: 20_000}, economysim.FlowScale{Flow: "charge", BP: 0}),
			clear:    flow(economysim.FlowScale{Flow: "charge", BP: 10_000}, economysim.FlowScale{Flow: "stipend", BP: 10_000}),
			empty:    flow(),
			held:     func(s economysim.SlotState) int { return len(s.FlowScales) },
			events: func(e economysim.Event) ([]int64, bool) {
				c, ok := e.(economysim.MoneyFlowScaled)
				return []int64{int64(len(c.Flow)), int64(c.Flow[0])}, ok
			},
			bad: []slotBad{
				{"flow unknown", flow(economysim.FlowScale{Flow: "nope", BP: 1}), economysim.ErrUnknownFlow},
				{"above MaxFlowScale", flow(economysim.FlowScale{Flow: "charge", BP: economysim.MaxFlowScale + 1}), economysim.ErrOutOfBounds},
				{"below 0", flow(economysim.FlowScale{Flow: "charge", BP: -1}), economysim.ErrOutOfBounds},
				{"flow twice", flow(economysim.FlowScale{Flow: "charge", BP: 1}, economysim.FlowScale{Flow: "charge", BP: 2}), economysim.ErrDuplicateID},
			},
		},
		{
			name: "SetBirthScales", layers: slotsEndow, off: slotsNone, keys: 1,
			set:   birth(economysim.RegionScale{Region: slotNorth, BP: 20_000}),
			clear: birth(economysim.RegionScale{Region: slotNorth, BP: 10_000}),
			empty: birth(),
			held:  func(s economysim.SlotState) int { return len(s.BirthScales) },
			events: func(e economysim.Event) ([]int64, bool) {
				c, ok := e.(economysim.PopulationScaled)
				return []int64{int64(c.Region)}, ok && c.Which == economysim.ScaleBirth
			},
			bad: []slotBad{
				{"region unknown", birth(economysim.RegionScale{Region: 9, BP: 1}), economysim.ErrUnknownRegion},
				{"above MaxBasisPoints", birth(economysim.RegionScale{Region: slotNorth, BP: economysim.MaxBasisPoints + 1}), economysim.ErrOutOfBounds},
				{"below 0", birth(economysim.RegionScale{Region: slotNorth, BP: -1}), economysim.ErrOutOfBounds},
				{"region twice", birth(economysim.RegionScale{Region: slotNorth, BP: 1}, economysim.RegionScale{Region: slotNorth, BP: 2}), economysim.ErrDuplicateID},
			},
		},
		{
			name: "SetDeathScales", layers: slotsEndow, off: slotsNone, keys: 1,
			set:   death(economysim.RegionScale{Region: slotNorth, BP: 0}),
			clear: death(economysim.RegionScale{Region: slotNorth, BP: 10_000}),
			empty: death(),
			held:  func(s economysim.SlotState) int { return len(s.DeathScales) },
			events: func(e economysim.Event) ([]int64, bool) {
				c, ok := e.(economysim.PopulationScaled)
				return []int64{int64(c.Region)}, ok && c.Which == economysim.ScaleDeath
			},
			bad: []slotBad{
				{"region unknown", death(economysim.RegionScale{Region: economysim.NoRegion, BP: 1}), economysim.ErrUnknownRegion},
				{"above MaxBasisPoints", death(economysim.RegionScale{Region: slotNorth, BP: economysim.MaxBasisPoints + 1}), economysim.ErrOutOfBounds},
				{"region twice", death(economysim.RegionScale{Region: slotNorth, BP: 1}, economysim.RegionScale{Region: slotNorth, BP: 2}), economysim.ErrDuplicateID},
			},
		},
		{
			name: "SetCapacityScales", layers: slotsEndow, off: slotsNone, keys: 2,
			set:      capacity(economysim.EndowmentScale{Region: slotNorth, Resource: 0, BP: 5_000}, economysim.EndowmentScale{Region: slotNorth, Resource: 2, BP: 20_000}),
			reversed: capacity(economysim.EndowmentScale{Region: slotNorth, Resource: 2, BP: 20_000}, economysim.EndowmentScale{Region: slotNorth, Resource: 0, BP: 5_000}),
			clear:    capacity(economysim.EndowmentScale{Region: slotNorth, Resource: 0, BP: 10_000}, economysim.EndowmentScale{Region: slotNorth, Resource: 2, BP: 10_000}),
			empty:    capacity(),
			held:     func(s economysim.SlotState) int { return len(s.CapacityScales) },
			events: func(e economysim.Event) ([]int64, bool) {
				c, ok := e.(economysim.EndowmentAdjusted)
				return []int64{int64(c.Region), int64(c.Resource)}, ok && c.Kind == economysim.AdjustKindShock
			},
			bad: endowmentBad(capacity),
		},
		{
			name: "SetRegenScales", layers: slotsEndow, off: slotsNone, keys: 2,
			set:      regen(economysim.EndowmentScale{Region: slotNorth, Resource: 0, BP: 5_000}, economysim.EndowmentScale{Region: slotNorth, Resource: 2, BP: 0}),
			reversed: regen(economysim.EndowmentScale{Region: slotNorth, Resource: 2, BP: 0}, economysim.EndowmentScale{Region: slotNorth, Resource: 0, BP: 5_000}),
			clear:    regen(economysim.EndowmentScale{Region: slotNorth, Resource: 0, BP: 10_000}, economysim.EndowmentScale{Region: slotNorth, Resource: 2, BP: 10_000}),
			empty:    regen(),
			held:     func(s economysim.SlotState) int { return len(s.RegenScales) },
			events: func(e economysim.Event) ([]int64, bool) {
				c, ok := e.(economysim.EndowmentAdjusted)
				return []int64{int64(c.Region), int64(c.Resource)}, ok && c.Kind == economysim.AdjustKindShock
			},
			bad: endowmentBad(regen),
		},
		{
			name: "AdjustEndowments", layers: slotsEndow, off: slotsNone, keys: 2, oneOff: true,
			set:      adjust(economysim.EndowmentAdjustment{Region: slotNorth, Resource: 0, Delta: -10}, economysim.EndowmentAdjustment{Region: slotNorth, Resource: 2, Delta: -10}),
			reversed: adjust(economysim.EndowmentAdjustment{Region: slotNorth, Resource: 2, Delta: -10}, economysim.EndowmentAdjustment{Region: slotNorth, Resource: 0, Delta: -10}),
			// An adjustment holds no value: there is nothing to clear and nothing is listed.
			empty: adjust(),
			held:  func(economysim.SlotState) int { return 0 },
			events: func(e economysim.Event) ([]int64, bool) {
				c, ok := e.(economysim.EndowmentAdjusted)
				return []int64{int64(c.Region), int64(c.Resource)}, ok && c.Kind == economysim.AdjustKindAdjust
			},
			bad: append(endowmentBad(func(c ...economysim.EndowmentScale) func(*economysim.World) error {
				adj := make([]economysim.EndowmentAdjustment, len(c))
				for i, e := range c {
					adj[i] = economysim.EndowmentAdjustment{Region: e.Region, Resource: e.Resource, Delta: -1}
				}
				return func(w *economysim.World) error { return w.AdjustEndowments(adj) }
			})[:4:4],
				slotBad{"delta above MaxEndowment", adjust(economysim.EndowmentAdjustment{Region: slotNorth, Delta: economysim.MaxEndowment + 1}), economysim.ErrOutOfBounds},
				slotBad{"delta below -MaxEndowment", adjust(economysim.EndowmentAdjustment{Region: slotNorth, Delta: -economysim.MaxEndowment - 1}), economysim.ErrOutOfBounds}),
		},
		{
			name: "SetOutsideShocks", layers: slotsExternal, off: slotsNone, keys: 2,
			set:      outside(economysim.OutsideShock{Node: slotOther, Commodity: slotFuel, BP: 20_000}, economysim.OutsideShock{Node: slotOther, Commodity: slotStaple, BP: 5_000}),
			reversed: outside(economysim.OutsideShock{Node: slotOther, Commodity: slotStaple, BP: 5_000}, economysim.OutsideShock{Node: slotOther, Commodity: slotFuel, BP: 20_000}),
			clear:    outside(economysim.OutsideShock{Node: slotOther, Commodity: slotFuel, BP: 10_000}, economysim.OutsideShock{Node: slotOther, Commodity: slotStaple, BP: 10_000}),
			empty:    outside(),
			held:     func(s economysim.SlotState) int { return len(s.OutsideShocks) },
			events: func(e economysim.Event) ([]int64, bool) {
				c, ok := e.(economysim.OutsideShocked)
				return []int64{int64(c.Node), int64(c.Commodity)}, ok
			},
			bad: []slotBad{
				{"region unknown", outside(economysim.OutsideShock{Node: 9, Commodity: slotFuel, BP: 1}), economysim.ErrUnknownRegion},
				{"commodity unknown", outside(economysim.OutsideShock{Node: slotOther, Commodity: 99, BP: 1}), economysim.ErrUnknownCommodity},
				{"ordinary region has no line", outside(economysim.OutsideShock{Node: slotNorth, Commodity: slotFuel, BP: 1}), economysim.ErrUnknownLine},
				{"node has no line for the commodity", outside(economysim.OutsideShock{Node: slotOther, Commodity: 1, BP: 1}), economysim.ErrUnknownLine},
				{"factor 0", outside(economysim.OutsideShock{Node: slotOther, Commodity: slotFuel}), economysim.ErrOutOfBounds},
				{"above MaxShockBP", outside(economysim.OutsideShock{Node: slotOther, Commodity: slotFuel, BP: big}), economysim.ErrOutOfBounds},
				{"line twice", outside(economysim.OutsideShock{Node: slotOther, Commodity: slotFuel, BP: 1}, economysim.OutsideShock{Node: slotOther, Commodity: slotFuel, BP: 2}), economysim.ErrDuplicateID},
			},
		},
	}
}

// endowmentBad is the failing calls of the two scales of an endowment (and, cut to
// its first four, of the adjustment), built over the call that takes the scales.
func endowmentBad(call func(...economysim.EndowmentScale) func(*economysim.World) error) []slotBad {
	return []slotBad{
		{"region unknown", call(economysim.EndowmentScale{Region: 9, Resource: 0, BP: 1}), economysim.ErrUnknownRegion},
		{"resource unknown", call(economysim.EndowmentScale{Region: slotNorth, Resource: 9, BP: 1}), economysim.ErrUnknownResource},
		{"resource below range", call(economysim.EndowmentScale{Region: slotNorth, Resource: -1, BP: 1}), economysim.ErrUnknownResource},
		{"key twice", call(economysim.EndowmentScale{Region: slotNorth, Resource: 0, BP: 1}, economysim.EndowmentScale{Region: slotNorth, Resource: 0, BP: 2}), economysim.ErrDuplicateID},
		{"above MaxShockBP", call(economysim.EndowmentScale{Region: slotNorth, Resource: 0, BP: economysim.MaxShockBP + 1}), economysim.ErrOutOfBounds},
		{"below 0", call(economysim.EndowmentScale{Region: slotNorth, Resource: 0, BP: -1}), economysim.ErrOutOfBounds},
	}
}

// TestSlots_Conformance verifies, for every setter of spec-integration.md 5.1, the
// checks of 5.1.3 in their order: each failing call returns its sentinel (and a
// *ValidationError naming the argument), and changes nothing, emits nothing and
// draws nothing, shown by the checkpoint bytes before and after; a module that is
// off is ErrModuleOff; an empty slice is accepted and does nothing.
//
// Requirement: FR-INT-002, FR-INT-003, FR-INT-005, FR-INT-006, FR-INT-007
func TestSlots_Conformance(t *testing.T) {
	t.Parallel()
	for _, row := range slotRows() {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			w, events := slotWorld(t, row.layers)
			before, n := slotBytes(t, w), len(*events)
			for _, bad := range row.bad {
				err := bad.call(w)
				if !errors.Is(err, bad.want) {
					t.Errorf("%s: %s = %v, want %v", row.name, bad.name, err, bad.want)
					continue
				}
				var ve *economysim.ValidationError
				if !errors.As(err, &ve) || len(ve.Path) == 0 {
					t.Errorf("%s: %s error %v is not a ValidationError with a path", row.name, bad.name, err)
				}
				if !bytes.Equal(slotBytes(t, w), before) || len(*events) != n {
					t.Errorf("%s: %s changed the world or emitted an event", row.name, bad.name)
				}
			}
			if err := row.empty(w); err != nil || !bytes.Equal(slotBytes(t, w), before) || len(*events) != n {
				t.Errorf("%s with an empty slice = %v, want nil and no change", row.name, err)
			}
			if row.off == nil {
				return
			}
			off, offEvents := slotWorld(t, row.off)
			offBefore, offN := slotBytes(t, off), len(*offEvents)
			if err := row.set(off); !errors.Is(err, economysim.ErrModuleOff) {
				t.Errorf("%s with the module off = %v, want ErrModuleOff", row.name, err)
			}
			if !bytes.Equal(slotBytes(t, off), offBefore) || len(*offEvents) != offN {
				t.Errorf("%s with the module off changed the world or emitted an event", row.name)
			}
		})
	}
}

// TestSlots_EventsCanonicalOrder verifies that a call emits one event per changed
// key, in the ascending order of the key whatever the order of the slice, with the
// Tick the next step runs, before that step's TickStarted; and that a call that
// repeats the held values emits nothing.
//
// Requirement: FR-INT-004, FR-OBS-017
func TestSlots_EventsCanonicalOrder(t *testing.T) {
	t.Parallel()
	for _, row := range slotRows() {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			w, events := slotWorld(t, row.layers)
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
			*events = (*events)[:0]
			call := row.set
			if row.reversed != nil {
				call = row.reversed
			}
			if err := call(w); err != nil {
				t.Fatalf("%s() = %v, want nil", row.name, err)
			}
			var keys [][]int64
			for _, e := range *events {
				if k, ok := row.events(e); ok {
					keys = append(keys, k)
				}
			}
			if len(keys) != row.keys {
				t.Fatalf("%s emitted %d events of its kind (%d events in all), want %d", row.name, len(keys), len(*events), row.keys)
			}
			if !slices.IsSortedFunc(keys, slices.Compare) && row.name != "SetFlowScales" {
				t.Errorf("%s events came in key order %v, want ascending", row.name, keys)
			}
			if row.name == "SetFlowScales" {
				var flows []string
				for _, e := range *events {
					if c, ok := e.(economysim.MoneyFlowScaled); ok {
						flows = append(flows, c.Flow)
					}
				}
				if !slices.IsSorted(flows) {
					t.Errorf("MoneyFlowScaled events came for flows %v, want ascending flow ID order", flows)
				}
			}
			if tick := w.Tick(); tick != 1 {
				t.Fatalf("Tick() = %d, want 1 after one step", tick)
			}
			n := len(*events)
			if err := call(w); !row.oneOff && (err != nil || len(*events) != n) {
				t.Errorf("the same %s again = %v with %d new events, want nil and none: an entry equal to the held value is no change", row.name, err, len(*events)-n)
			}
			*events = (*events)[:0]
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
			if _, started := (*events)[0].(economysim.TickStarted); !started {
				t.Errorf("first event of the next step is %T, want TickStarted: the call's events precede it", (*events)[0])
			}
		})
	}
}

// TestSlots_EventTicks verifies that the events of a call carry the Tick the next
// step runs (the world's Tick), for every setter that emits.
//
// Requirement: FR-INT-004
func TestSlots_EventTicks(t *testing.T) {
	t.Parallel()
	tickOf := func(e economysim.Event) (int64, bool) {
		switch e := e.(type) {
		case economysim.YieldShocked:
			return e.Tick, true
		case economysim.EdgeStatusChanged:
			return e.Tick, true
		case economysim.TransitFactorChanged:
			return e.Tick, true
		case economysim.MoneyFlowScaled:
			return e.Tick, true
		case economysim.PopulationScaled:
			return e.Tick, true
		case economysim.EndowmentAdjusted:
			return e.Tick, true
		case economysim.OutsideShocked:
			return e.Tick, true
		}
		return 0, false
	}
	for _, row := range slotRows() {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			w, events := slotWorld(t, row.layers)
			for range 3 {
				if err := w.Step(); err != nil {
					t.Fatalf("Step() = %v, want nil", err)
				}
			}
			*events = (*events)[:0]
			if err := row.set(w); err != nil {
				t.Fatalf("%s() = %v, want nil", row.name, err)
			}
			for _, e := range *events {
				if tick, ok := tickOf(e); ok && tick != w.Tick() {
					t.Errorf("%T has Tick %d, want %d, the tick the next step runs", e, tick, w.Tick())
				}
			}
			if len(*events) == 0 {
				t.Errorf("%s emitted no event", row.name)
			}
		})
	}
}

// TestSlots_DefaultClears verifies that a key set to its default is cleared: after
// the set and the clear the list Slots reads back is empty and the checkpoint is
// that of a world never shocked, and between them the list holds the keys.
//
// Requirement: FR-INT-048, FR-INT-032
func TestSlots_DefaultClears(t *testing.T) {
	t.Parallel()
	for _, row := range slotRows() {
		if row.clear == nil {
			continue
		}
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			w, _ := slotWorld(t, row.layers)
			fresh := slotBytes(t, w)
			if s, err := w.Slots(); err != nil || row.held(s) != 0 {
				t.Fatalf("Slots() of a new world = %+v, %v, want nothing held", s, err)
			}
			if err := row.set(w); err != nil {
				t.Fatalf("%s() = %v, want nil", row.name, err)
			}
			if s, err := w.Slots(); err != nil || row.held(s) != row.keys {
				t.Errorf("Slots() after %s holds %d keys (error %v), want %d", row.name, row.held(s), err, row.keys)
			}
			if bytes.Equal(slotBytes(t, w), fresh) {
				t.Errorf("the checkpoint after %s is that of a world never shocked", row.name)
			}
			if err := row.clear(w); err != nil {
				t.Fatalf("clearing %s = %v, want nil", row.name, err)
			}
			if s, err := w.Slots(); err != nil || row.held(s) != 0 {
				t.Errorf("Slots() after clearing %s holds %d keys (error %v), want none", row.name, row.held(s), err)
			}
			if !bytes.Equal(slotBytes(t, w), fresh) {
				t.Errorf("the checkpoint after setting and clearing %s differs from a world never shocked", row.name)
			}
		})
	}
}

// TestSlots_ErrorCallsName verifies that an error carries the name of the call
// (architecture.md 8.2) and the index of the entry in the path of the argument.
//
// Requirement: FR-INT-005
func TestSlots_ErrorCallsName(t *testing.T) {
	t.Parallel()
	w, _ := slotWorld(t, slotsExternal)
	err := w.SetYieldShocks([]economysim.YieldShock{{Region: slotNorth, BP: 1}, {Region: 9, BP: 1}})
	var ve *economysim.ValidationError
	if !errors.As(err, &ve) || len(ve.Path) != 2 || ve.Path[0] != "changes[1]" || ve.Path[1] != "region" {
		t.Fatalf("SetYieldShocks error = %v, want a ValidationError with the path changes[1], region", err)
	}
	if want := "SetYieldShocks: "; len(err.Error()) < len(want) || err.Error()[:len(want)] != want {
		t.Errorf("SetYieldShocks error text = %q, want it to begin %q", err.Error(), want)
	}
}

// TestSlots_RestoredEqualsOriginal verifies that a world restored from a checkpoint
// reports the same slots as the original: the owners save the lists Slots reads.
//
// Requirement: FR-INT-032, FR-INT-070
func TestSlots_RestoredEqualsOriginal(t *testing.T) {
	t.Parallel()
	for _, row := range slotRows() {
		if row.clear == nil {
			continue
		}
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			w, _ := slotWorld(t, row.layers)
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
			if err := row.set(w); err != nil {
				t.Fatalf("%s() = %v, want nil", row.name, err)
			}
			cp, err := w.Checkpoint()
			if err != nil {
				t.Fatalf("Checkpoint() = %v, want nil", err)
			}
			r, err := economysim.RestoreWorld(loadValidPack(t, row.layers...), cp)
			if err != nil {
				t.Fatalf("RestoreWorld() = %v, want nil", err)
			}
			want, _ := w.Slots()
			got, err := r.Slots()
			if err != nil || fmt.Sprintf("%+v", got) != fmt.Sprintf("%+v", want) {
				t.Errorf("restored Slots() = %+v, %v, want %+v", got, err, want)
			}
		})
	}
}

// TestSetYieldShocks verifies the mechanism through the public call
// (spec-production.md 6.8): a job that starts after the call reads the shock at
// P2.7 (cut_fuel starts and completes at tick 0 with the effective yield 3 000),
// and a job in progress keeps its pending output (make_tool_a, started before tick 0
// by the warm start, completes at tick 1 at 10 000). The event carries the factor
// before and after.
//
// Requirement: FR-INT-040
func TestSetYieldShocks(t *testing.T) {
	t.Parallel()
	const cutFuel, makeTool economysim.RecipeID = 0, 4
	w, events := slotWorld(t, slotsNone)
	if err := w.SetYieldShocks([]economysim.YieldShock{{Region: slotNorth, Recipe: cutFuel, BP: 3_000}, {Region: slotNorth, Recipe: makeTool, BP: 3_000}}); err != nil {
		t.Fatalf("SetYieldShocks() = %v, want nil", err)
	}
	var shocked []economysim.YieldShocked
	for _, e := range *events {
		if y, ok := e.(economysim.YieldShocked); ok {
			shocked = append(shocked, y)
		}
	}
	want := []economysim.YieldShocked{
		{Tick: 0, Region: slotNorth, Recipe: cutFuel, BeforeBP: 10_000, AfterBP: 3_000},
		{Tick: 0, Region: slotNorth, Recipe: makeTool, BeforeBP: 10_000, AfterBP: 3_000},
	}
	if !slices.Equal(shocked, want) {
		t.Errorf("YieldShocked events = %+v, want %+v", shocked, want)
	}
	for range 2 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	yields := map[[2]int64]int64{}
	for _, e := range *events {
		if r, ok := e.(economysim.RecipeExecuted); ok {
			yields[[2]int64{int64(r.Recipe), r.StartTick}] = r.EffectiveYield
		}
	}
	if got, ok := yields[[2]int64{int64(cutFuel), 0}]; !ok || got != 3_000 {
		t.Errorf("effective yield of cut_fuel started at tick 0 = %d (ran %v), want 3000: a job that starts after the call reads the shock", got, ok)
	}
	if got, ok := yields[[2]int64{int64(makeTool), -1}]; !ok || got != 10_000 {
		t.Errorf("effective yield of make_tool_a started before tick 0 = %d (ran %v), want 10000: a job in progress keeps its pending output", got, ok)
	}
}

// TestSetEdgeStatuses_Public verifies the public call: the graph shows the closed
// edge and the longer one at once, EdgeStatusChanged has the cause consumer, and the
// call refuses nothing it should accept (an entry that opens an edge says Closed
// false and FactorBP 10 000).
//
// Requirement: FR-INT-041
func TestSetEdgeStatuses_Public(t *testing.T) {
	t.Parallel()
	w, events := slotWorld(t, slotsExternal)
	if err := w.SetEdgeStatuses([]economysim.EdgeStatus{{From: slotNorth, To: slotOther, Closed: true, FactorBP: 10_000}, {From: slotOther, To: slotNorth, FactorBP: 20_000}}); err != nil {
		t.Fatalf("SetEdgeStatuses() = %v, want nil", err)
	}
	g := w.Graph()
	if e, ok := g.Edge(slotNorth, slotOther); !ok || !e.Closed {
		t.Errorf("edge north to outer after the call = %+v (found %v), want closed", e, ok)
	}
	if e, ok := g.Edge(slotOther, slotNorth); !ok || e.Closed || e.FactorBP != 20_000 {
		t.Errorf("edge outer to north after the call = %+v (found %v), want open with a factor of 20 000", e, ok)
	}
	for _, e := range *events {
		if c, ok := e.(economysim.EdgeStatusChanged); ok && c.Cause != economysim.CauseConsumer {
			t.Errorf("EdgeStatusChanged cause = %q, want %q", c.Cause, economysim.CauseConsumer)
		}
	}
}

// TestSetTransitFactors_Public verifies the public call on the transport pack: the
// snapshot lists the factors that are not 10 000, TransitFactorChanged has the cause
// consumer and the factors before and after, and 0 is accepted (it blocks new
// dispatches).
//
// Requirement: FR-INT-042
func TestSetTransitFactors_Public(t *testing.T) {
	t.Parallel()
	w, events := slotWorld(t, slotsTransport)
	if err := w.SetTransitFactors([]economysim.TransitFactor{{From: 1, To: 0, Mode: 1, BP: 0}, {From: 0, To: 1, Mode: 0, BP: 20_000}}); err != nil {
		t.Fatalf("SetTransitFactors() = %v, want nil", err)
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	want := []economysim.TransitFactorRow{{From: 0, To: 1, Mode: 0, BP: 20_000}, {From: 1, To: 0, Mode: 1, BP: 0}}
	if !slices.Equal(snap.TransitFactors, want) {
		t.Errorf("Snapshot().TransitFactors = %+v, want %+v", snap.TransitFactors, want)
	}
	var got []economysim.TransitFactorChanged
	for _, e := range *events {
		if c, ok := e.(economysim.TransitFactorChanged); ok {
			got = append(got, c)
		}
	}
	if len(got) != 2 || got[0].AfterBP != 20_000 || got[0].BeforeBP != 10_000 || got[1].AfterBP != 0 || got[0].Cause != economysim.CauseConsumer {
		t.Errorf("TransitFactorChanged events = %+v, want two from 10 000, to 20 000 and 0, cause consumer", got)
	}
}

// TestSetFlowScales_Public verifies the public call on the money pack: 0 pauses a
// flow and the scale shows in the flow's row of the snapshot, MoneyFlowScaled has the
// cause consumer, and the default 10 000 puts the row back.
//
// Requirement: FR-INT-043
func TestSetFlowScales_Public(t *testing.T) {
	t.Parallel()
	w, events := slotWorld(t, slotsFlows)
	scale := func() int64 {
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		for _, f := range snap.Flows {
			if f.Flow == "charge" {
				return f.ScaleBP
			}
		}
		t.Fatal("no row for the flow charge")
		return 0
	}
	if got := scale(); got != 10_000 {
		t.Fatalf("scale of charge in a new world = %d, want 10000", got)
	}
	if err := w.SetFlowScales([]economysim.FlowScale{{Flow: "charge", BP: 0}}); err != nil {
		t.Fatalf("SetFlowScales(0) = %v, want nil", err)
	}
	if got := scale(); got != 0 {
		t.Errorf("scale of charge after the call = %d, want 0", got)
	}
	if err := w.SetFlowScales([]economysim.FlowScale{{Flow: "charge", BP: 10_000}}); err != nil || scale() != 10_000 {
		t.Errorf("SetFlowScales(10000) = %v, scale %d, want nil and 10000", err, scale())
	}
	n := 0
	for _, e := range *events {
		if c, ok := e.(economysim.MoneyFlowScaled); ok {
			n++
			if c.Cause != economysim.CauseConsumer || c.Flow != "charge" {
				t.Errorf("MoneyFlowScaled = %+v, want flow charge and cause consumer", c)
			}
		}
	}
	if n != 2 {
		t.Errorf("%d MoneyFlowScaled events, want 2 (set and cleared)", n)
	}
}

// TestSetBirthDeathScales verifies the default of a birth or death scale: it is the
// region's configured value, not 10 000, so a scale equal to the configured one
// clears the key (Slots empty) and the event reports the configured value as the
// factor before. The two scales are separate keys.
//
// Requirement: FR-INT-044
func TestSetBirthDeathScales(t *testing.T) {
	t.Parallel()
	w, events := slotWorld(t, slotsEndow)
	// The population overlay has no row for the region, so the configured scale is the
	// default 10 000; the test would be vacuous if a row set another value.
	configured := int64(10_000)
	if rows := w.Config().Modules.Population.Regions; len(rows) != 0 {
		t.Fatalf("the population overlay has %d region rows, want none: the configured scale is then 10000", len(rows))
	}
	if err := w.SetBirthScales([]economysim.RegionScale{{Region: slotNorth, BP: configured + 1}}); err != nil {
		t.Fatalf("SetBirthScales() = %v, want nil", err)
	}
	if s, _ := w.Slots(); len(s.BirthScales) != 1 || len(s.DeathScales) != 0 {
		t.Errorf("Slots() births %+v deaths %+v, want one birth scale and no death scale", s.BirthScales, s.DeathScales)
	}
	if err := w.SetBirthScales([]economysim.RegionScale{{Region: slotNorth, BP: configured}}); err != nil {
		t.Fatalf("SetBirthScales(configured) = %v, want nil", err)
	}
	if s, _ := w.Slots(); len(s.BirthScales) != 0 {
		t.Errorf("Slots() after setting the configured scale = %+v, want none: the configured value is the default", s.BirthScales)
	}
	var got []economysim.PopulationScaled
	for _, e := range *events {
		if c, ok := e.(economysim.PopulationScaled); ok {
			got = append(got, c)
		}
	}
	want := []economysim.PopulationScaled{
		{Region: slotNorth, Which: economysim.ScaleBirth, BeforeBP: configured, AfterBP: configured + 1},
		{Region: slotNorth, Which: economysim.ScaleBirth, BeforeBP: configured + 1, AfterBP: configured},
	}
	if !slices.Equal(got, want) {
		t.Errorf("PopulationScaled events = %+v, want %+v", got, want)
	}
}

// TestSetCapacityRegenScales_Public verifies the capacity scale at the call: the
// effective capacity is recomputed and a stock above it is clamped at once, the
// excess lost and reported by the event (Lost), while the regeneration scale leaves
// the stock alone.
//
// Requirement: FR-INT-045
func TestSetCapacityRegenScales_Public(t *testing.T) {
	t.Parallel()
	w, events := slotWorld(t, slotsEndow)
	stock := w.Endowment(slotNorth, 0)
	if stock <= 0 {
		t.Fatalf("Endowment(north, 0) = %d, want a stock above 0 in the resources pack", stock)
	}
	if err := w.SetRegenScales([]economysim.EndowmentScale{{Region: slotNorth, Resource: 0, BP: 0}}); err != nil {
		t.Fatalf("SetRegenScales() = %v, want nil", err)
	}
	if got := w.Endowment(slotNorth, 0); got != stock {
		t.Errorf("Endowment after a regeneration scale = %d, want %d: the stock is not touched", got, stock)
	}
	if err := w.SetCapacityScales([]economysim.EndowmentScale{{Region: slotNorth, Resource: 0, BP: 1_000}}); err != nil {
		t.Fatalf("SetCapacityScales() = %v, want nil", err)
	}
	after := w.Endowment(slotNorth, 0)
	if after >= stock {
		t.Errorf("Endowment after a capacity scale of 10%% = %d, want below %d: the stock is clamped to the new capacity", after, stock)
	}
	var lost int64
	for _, e := range *events {
		if c, ok := e.(economysim.EndowmentAdjusted); ok && c.Kind == economysim.AdjustKindShock && c.CapacityBP == 1_000 {
			lost = c.Lost
		}
	}
	if lost != stock-after {
		t.Errorf("EndowmentAdjusted.Lost = %d, want %d, the units above the new capacity", lost, stock-after)
	}
}

// TestAdjustEndowments_Public verifies the adjustment: the stock changes now by the
// delta, is clamped to [0, the capacity], and the event reports the change that took
// effect; the call holds no value, so Slots lists nothing for it.
//
// Requirement: FR-INT-046
func TestAdjustEndowments_Public(t *testing.T) {
	t.Parallel()
	w, events := slotWorld(t, slotsEndow)
	stock := w.Endowment(slotNorth, 0)
	if err := w.AdjustEndowments([]economysim.EndowmentAdjustment{{Region: slotNorth, Resource: 0, Delta: -economysim.MaxEndowment}}); err != nil {
		t.Fatalf("AdjustEndowments(-MaxEndowment) = %v, want nil", err)
	}
	if got := w.Endowment(slotNorth, 0); got != 0 {
		t.Errorf("Endowment after the largest negative adjustment = %d, want 0: clamped at the floor", got)
	}
	if err := w.AdjustEndowments([]economysim.EndowmentAdjustment{{Region: slotNorth, Resource: 0, Delta: 7}}); err != nil {
		t.Fatalf("AdjustEndowments(7) = %v, want nil", err)
	}
	if got := w.Endowment(slotNorth, 0); got != 7 {
		t.Errorf("Endowment after +7 = %d, want 7", got)
	}
	var deltas []int64
	for _, e := range *events {
		if c, ok := e.(economysim.EndowmentAdjusted); ok && c.Kind == economysim.AdjustKindAdjust {
			deltas = append(deltas, c.Delta)
		}
	}
	if want := []int64{-stock, 7}; !slices.Equal(deltas, want) {
		t.Errorf("EndowmentAdjusted deltas = %v, want %v: the change that took effect", deltas, want)
	}
	if s, _ := w.Slots(); s.CapacityScales != nil || s.RegenScales != nil {
		t.Errorf("Slots() after an adjustment = %+v, want no scale: an adjustment holds no value", s)
	}
}

// TestSetOutsideShocks_Public verifies the public call on the external pack: the
// event carries the factor before and after, and Slots lists the shock until it is
// set back to 10 000.
//
// Requirement: FR-INT-047
func TestSetOutsideShocks_Public(t *testing.T) {
	t.Parallel()
	w, events := slotWorld(t, slotsExternal)
	if err := w.SetOutsideShocks([]economysim.OutsideShock{{Node: slotOther, Commodity: slotStaple, BP: 25_000}}); err != nil {
		t.Fatalf("SetOutsideShocks() = %v, want nil", err)
	}
	var got []economysim.OutsideShocked
	for _, e := range *events {
		if c, ok := e.(economysim.OutsideShocked); ok {
			got = append(got, c)
		}
	}
	want := []economysim.OutsideShocked{{Tick: 0, Node: slotOther, Commodity: slotStaple, From: 10_000, To: 25_000}}
	if !slices.Equal(got, want) {
		t.Errorf("OutsideShocked events = %+v, want %+v", got, want)
	}
	if s, _ := w.Slots(); !slices.Equal(s.OutsideShocks, []economysim.OutsideShock{{Node: slotOther, Commodity: slotStaple, BP: 25_000}}) {
		t.Errorf("Slots().OutsideShocks = %+v, want the one shock", s.OutsideShocks)
	}
}

// inStepCalls is every public mutating call of INT and the reader Slots, made with
// arguments that would be accepted between steps, so that only the guard can refuse
// them.
func inStepCalls(w *economysim.World) map[string]error {
	_, addErr := w.AddAgent(economysim.NewAgent{Region: slotNorth})
	_, slotsErr := w.Slots()
	return map[string]error{
		"AddAgent":          addErr,
		"RemoveAgent":       w.RemoveAgent(0),
		"Direct":            w.Direct(0, economysim.Directive{}),
		"SetYieldShocks":    w.SetYieldShocks([]economysim.YieldShock{{Region: slotNorth, BP: 1}}),
		"SetEdgeStatuses":   w.SetEdgeStatuses([]economysim.EdgeStatus{{From: slotNorth, To: slotOther, FactorBP: 1}}),
		"SetTransitFactors": w.SetTransitFactors(nil),
		"SetFlowScales":     w.SetFlowScales(nil),
		"SetBirthScales":    w.SetBirthScales(nil),
		"SetDeathScales":    w.SetDeathScales(nil),
		"SetCapacityScales": w.SetCapacityScales(nil),
		"SetRegenScales":    w.SetRegenScales(nil),
		"AdjustEndowments":  w.AdjustEndowments(nil),
		"SetOutsideShocks":  w.SetOutsideShocks([]economysim.OutsideShock{{Node: slotOther, Commodity: slotFuel, BP: 1}}),
		"Slots":             slotsErr,
	}
}

// TestInStep_Calls verifies that every public call of INT made from inside a step
// (here, an observer on TickStarted) returns ErrInStep, before any other check
// (spec-integration.md 3.1.2), and changes nothing: the world ends the step in the
// state of a twin that was not called.
//
// Requirement: FR-INT-001
func TestInStep_Calls(t *testing.T) {
	t.Parallel()
	called := map[string]error{}
	var w *economysim.World
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		if _, ok := e.(economysim.TickStarted); ok && len(called) == 0 {
			for name, err := range inStepCalls(w) {
				called[name] = err
			}
		}
	})
	cfg := loadValidPack(t, slotsExternal...)
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(7), economysim.WithObserver(obs))
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	twin, err := economysim.NewWorld(cfg, economysim.WithSeed(7))
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	for _, x := range []*economysim.World{w, twin} {
		if err := x.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	if len(called) != 14 {
		t.Fatalf("the observer made %d calls, want 14", len(called))
	}
	for name, err := range called {
		if !errors.Is(err, economysim.ErrInStep) {
			t.Errorf("%s from inside a step = %v, want ErrInStep", name, err)
		}
	}
	if !bytes.Equal(slotBytes(t, w), slotBytes(t, twin)) {
		t.Error("the checkpoint of the world called from inside the step differs from that of its twin")
	}
}
