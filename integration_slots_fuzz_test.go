package economysim_test

import (
	"bytes"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fuzzReader reads numbers from the fuzz input, 0 once it is spent.
type fuzzReader struct{ b []byte }

func (r *fuzzReader) next() byte {
	if len(r.b) == 0 {
		return 0
	}
	c := r.b[0]
	r.b = r.b[1:]
	return c
}

// pick maps a byte to a value of [lo, hi] or one step outside it, so that most inputs
// are accepted and the bounds are met: below, at each end, above, or inside.
func (r *fuzzReader) pick(lo, hi int64) int64 {
	switch c := r.next(); c % 6 {
	case 0:
		return lo - 1
	case 1:
		return lo
	case 2:
		return hi
	case 3:
		return hi + 1
	default:
		return lo + int64(c)%(hi-lo+1)
	}
}

// FuzzSetters fuzzes the nine setters and the adjustment: for any entries, no call
// panics, a refused call changes nothing (the checkpoint bytes and the events are
// those before it), an accepted one leaves a world whose audit passes and whose
// Slots can be read, and the same call repeated is accepted.
//
// Requirement: NFR-INT-005, FR-INT-002
func FuzzSetters(f *testing.F) {
	for _, s := range [][]byte{nil, {0, 2, 1, 1, 1}, {1, 2, 0, 1, 0, 1}, {3, 1, 2, 2}, {8, 3, 1, 2, 1, 3, 1, 2}, {9, 2, 1, 1, 1, 3, 1, 1, 0}} {
		f.Add(s)
	}
	packs := [10][]string{slotsExternal, slotsExternal, slotsTransport, slotsFlows, slotsEndow, slotsEndow, slotsEndow, slotsEndow, slotsEndow, slotsExternal}
	var cfgs [10]economysim.Config
	for i, p := range packs {
		cfgs[i] = loadValidPack(f, p...)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		r := &fuzzReader{data}
		which := int(r.next() % 10)
		var events int
		w, err := economysim.NewWorld(cfgs[which], economysim.WithSeed(3), economysim.WithAudit(),
			economysim.WithObserver(economysim.ObserverFunc(func(economysim.Event) { events++ })))
		if err != nil {
			t.Fatalf("NewWorld() = %v, want nil", err)
		}
		call := fuzzCall(which, r)
		before, n := slotBytes(t, w), events
		if err := call(w); err != nil {
			if !bytes.Equal(slotBytes(t, w), before) || events != n {
				t.Errorf("setter %d refused (%v) and the world changed", which, err)
			}
			return
		}
		if err := w.Audit(); err != nil {
			t.Errorf("setter %d accepted, then Audit() = %v", which, err)
		}
		if _, err := w.Slots(); err != nil {
			t.Errorf("setter %d accepted, then Slots() = %v", which, err)
		}
		if err := call(w); err != nil {
			t.Errorf("setter %d accepted a call and refused it again: %v", which, err)
		}
	})
}

// fuzzCall reads up to three entries for the setter `which` and returns the call.
func fuzzCall(which int, r *fuzzReader) func(w *economysim.World) error {
	count := int(r.next() % 4)
	region := func() economysim.RegionID { return economysim.RegionID(r.pick(0, 1)) }
	switch which {
	case 0:
		var c []economysim.YieldShock
		for range count {
			c = append(c, economysim.YieldShock{Region: region(), Recipe: economysim.RecipeID(r.pick(0, 9)), BP: r.pick(0, economysim.MaxYieldBP)})
		}
		return func(w *economysim.World) error { return w.SetYieldShocks(c) }
	case 1:
		var c []economysim.EdgeStatus
		for range count {
			c = append(c, economysim.EdgeStatus{From: region(), To: region(), Closed: r.next()%2 == 1, FactorBP: r.pick(1, economysim.MaxEdgeFactorBP)})
		}
		return func(w *economysim.World) error { return w.SetEdgeStatuses(c) }
	case 2:
		var c []economysim.TransitFactor
		for range count {
			c = append(c, economysim.TransitFactor{From: region(), To: region(), Mode: economysim.ModeID(r.pick(0, 1)), BP: r.pick(0, economysim.MaxTransitFactorBP)})
		}
		return func(w *economysim.World) error { return w.SetTransitFactors(c) }
	case 3:
		var c []economysim.FlowScale
		for range count {
			c = append(c, economysim.FlowScale{Flow: []string{"charge", "stipend", "nope"}[r.next()%3], BP: r.pick(0, economysim.MaxFlowScale)})
		}
		return func(w *economysim.World) error { return w.SetFlowScales(c) }
	case 4, 5:
		var c []economysim.RegionScale
		for range count {
			c = append(c, economysim.RegionScale{Region: region(), BP: r.pick(0, economysim.MaxBasisPoints)})
		}
		if which == 4 {
			return func(w *economysim.World) error { return w.SetBirthScales(c) }
		}
		return func(w *economysim.World) error { return w.SetDeathScales(c) }
	case 6, 7:
		var c []economysim.EndowmentScale
		for range count {
			c = append(c, economysim.EndowmentScale{Region: region(), Resource: economysim.ResourceID(r.pick(0, 2)), BP: r.pick(0, economysim.MaxShockBP)})
		}
		if which == 6 {
			return func(w *economysim.World) error { return w.SetCapacityScales(c) }
		}
		return func(w *economysim.World) error { return w.SetRegenScales(c) }
	case 8:
		var c []economysim.EndowmentAdjustment
		for range count {
			c = append(c, economysim.EndowmentAdjustment{Region: region(), Resource: economysim.ResourceID(r.pick(0, 2)), Delta: r.pick(-economysim.MaxEndowment, economysim.MaxEndowment)})
		}
		return func(w *economysim.World) error { return w.AdjustEndowments(c) }
	}
	var c []economysim.OutsideShock
	for range count {
		c = append(c, economysim.OutsideShock{Node: region(), Commodity: economysim.CommodityID(r.pick(0, 8)), BP: r.pick(1, economysim.MaxShockBP)})
	}
	return func(w *economysim.World) error { return w.SetOutsideShocks(c) }
}
