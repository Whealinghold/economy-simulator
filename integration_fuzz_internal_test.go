package economysim

import (
	"bytes"
	"slices"
	"testing"
)

// fuzzBytes reads small numbers from the fuzz input, 0 once it runs out.
type fuzzBytes struct{ b []byte }

// next returns the next byte, or 0 when the input is spent.
func (f *fuzzBytes) next() int64 {
	if len(f.b) == 0 {
		return 0
	}
	v := int64(f.b[0])
	f.b = f.b[1:]
	return v
}

// pick maps a byte to one of the interesting values of a bound: below the range,
// at its ends, inside, above it. The ID spaces are small, so most inputs land on a
// real ID and a few on the edges.
func (f *fuzzBytes) pick(lo, hi int64) int64 {
	switch c := f.next(); c % 6 {
	case 0:
		return lo - 1
	case 1:
		return lo
	case 2:
		return hi
	case 3:
		return hi + 1
	default:
		return lo + c%(hi-lo+1)
	}
}

// directive reads a directive from the input: a job, intents, discards, holds and
// Sticky, every field over its bound and its neighbours.
func (f *fuzzBytes) directive() Directive {
	var d Directive
	if f.next()%2 == 1 {
		d.Job = &JobRequest{NoJob: f.next()%4 == 0, Recipe: RecipeID(f.pick(0, 1)), Executions: f.pick(0, 1)}
	}
	for range f.next() % 4 {
		d.Intents = append(d.Intents, Intent{Commodity: CommodityID(f.pick(0, 3)), Side: Side(f.pick(0, 1)),
			Quantity: Quantity(f.pick(1, MaxQuantity)), Limit: Price(f.pick(1, MaxPrice)), Target: Quantity(f.pick(0, MaxQuantity)),
			Reserve: Quantity(f.pick(0, MaxQuantity)), UnitCost: Price(f.pick(0, MaxPrice)), Purpose: Purpose(f.pick(0, 3))})
	}
	for range f.next() % 3 {
		d.Discards = append(d.Discards, Discard{Commodity: CommodityID(f.pick(0, 3)), Quantity: Quantity(f.pick(1, MaxQuantity))})
	}
	for range f.next() % 3 {
		d.Hold = append(d.Hold, CommodityID(f.pick(0, 3)))
	}
	d.Sticky = f.next()%2 == 1
	return d
}

// FuzzDirect fuzzes the directive of Direct: no input panics, a refused call
// changes nothing (the checkpoint bytes are those before it), and an accepted one
// leaves a mailbox that holds nothing the checks refuse (a second call with the
// mailbox's own content is accepted).
//
// Requirement: NFR-INT-005, FR-INT-002
func FuzzDirect(f *testing.F) {
	for _, s := range [][]byte{nil, {1, 0, 1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 1, 2, 1, 1}, {1, 3, 3, 3, 2, 5, 5, 5, 5, 5, 5, 5, 5, 5, 2, 1, 1, 1, 1, 2, 2, 2, 2, 1, 1}, {0, 3, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		w, _, id, _, _, _ := directWorld(t)
		d := (&fuzzBytes{data}).directive()
		before := bytesOf(t, w)
		err := w.Direct(id, d)
		switch {
		case err != nil && !bytes.Equal(bytesOf(t, w), before):
			t.Errorf("Direct(%+v) = %v, and the checkpoint changed", d, err)
		case err == nil:
			if m := w.agents[id].directive; m != nil {
				if err := w.Direct(id, *m); err != nil {
					t.Errorf("Direct(%+v) accepted, but its mailbox %+v is refused: %v", d, m, err)
				}
			}
		}
	})
}

// FuzzAddAgent fuzzes the description of AddAgent: no input panics, a refused call
// changes nothing and issues no ID, and an accepted one returns the next ID and
// leaves a world whose audit passes.
//
// Requirement: NFR-INT-005, FR-INT-002
func FuzzAddAgent(f *testing.F) {
	for _, s := range [][]byte{nil, {1, 1, 1, 1, 1, 1, 1, 1}, {3, 3, 3, 3, 3, 3, 3, 3, 3}, {2, 2, 2, 2, 2, 2, 2, 2, 2, 2}} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		w, _, _ := extWorld(t, directConfig())
		r := &fuzzBytes{data}
		n := NewAgent{Region: RegionID(r.pick(0, 1)), Role: RoleID(r.pick(0, 0)), Money: Money(r.pick(0, MaxMoney)),
			Strategy: []string{"", "directed", "default", "script", "nobody"}[r.next()%5], Mortal: r.next()%2 == 1}
		for range r.next() % 3 {
			n.Stock = append(n.Stock, StockItem{Commodity: CommodityID(r.pick(0, 3)), Quantity: Quantity(r.pick(1, MaxQuantity))})
		}
		before, next := bytesOf(t, w), w.state.nextAgent
		id, err := w.AddAgent(n)
		switch {
		case err != nil && (id != NoAgent || w.state.nextAgent != next || !bytes.Equal(bytesOf(t, w), before)):
			t.Errorf("AddAgent(%+v) = %d, %v, and the world changed (next agent %d, was %d)", n, id, err, w.state.nextAgent, next)
		case err == nil && (id != next || w.state.nextAgent != next+1 || w.Audit() != nil || !slices.Contains(w.integ.externals, int32(id))):
			t.Errorf("AddAgent(%+v) = %d, nil: next agent %d (was %d), audit %v", n, id, w.state.nextAgent, next, w.Audit())
		}
	})
}
