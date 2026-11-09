package economysim

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file holds the benchmarks of the transport module (NFR-TRN-003): a dispatch with
// 1 line and with MaxShipmentLines lines, and the arrivals of a tick. They report; the
// limits that must hold are the allocation tests of transport_request_internal_test.go
// and the cost test of transport_arrival_internal_test.go.

// benchCargoes is the commodities the many-line dispatch ships besides grain, as many as
// MaxShipmentLines needs.
const benchCargoes = MaxShipmentLines - 1

// transportBenchWorld is the world of shipConfig with benchCargoes more raw commodities, used by a
// recipe each so that none is an orphan, and the hired coach, whose haul units cost 4 a
// carrying unit over the edge of 40.
func transportBenchWorld(b *testing.B) (*World, []Line) {
	b.Helper()
	c := shipConfig()
	lines := []Line{{Commodity: 0, Quantity: 1}} // placeholder, set below by name
	for i := range benchCargoes {
		id := fmt.Sprintf("cargo%02d", i)
		c.Commodities = append(c.Commodities, Commodity{ID: id, Mass: 1, Bulk: 1, Raw: true, ReferencePrice: 100})
		c.Recipes = append(c.Recipes, Recipe{ID: "use" + id, Duration: 1, TraineeYield: 5_000,
			Inputs:  []Item{{Commodity: id, Quantity: 1}},
			Outputs: []Output{{Commodity: "haul", Quantity: 1, CostShare: 10_000}}})
		c.Roles[0].Recipes = append(c.Roles[0].Recipes, "use"+id)
	}
	w, err := NewWorld(c, WithStrategy("script", &shipScript{}))
	if err != nil {
		b.Fatalf("NewWorld() = %v, want nil", err)
	}
	if err := w.ledger.EndConstruction(); err != nil {
		b.Fatalf("EndConstruction() = %v, want nil", err)
	}
	lines = lines[:0]
	for i := range c.Commodities {
		if c.Commodities[i].ID == "grain" || len(c.Commodities[i].ID) == 7 && c.Commodities[i].ID[:5] == "cargo" {
			lines = append(lines, Line{Commodity: CommodityID(i), Quantity: 1})
		}
	}
	if len(lines) != MaxShipmentLines {
		b.Fatalf("transportBenchWorld has %d cargo lines, want %d", len(lines), MaxShipmentLines)
	}
	return w, lines
}

// benchGive puts qty units of a commodity into the account of an agent from the initial
// boundary.
func benchGive(b *testing.B, w *World, id AgentID, commodity CommodityID, qty int64) {
	b.Helper()
	src, ok := w.ledger.Boundary(ledger.Initial, int(commodity))
	if !ok {
		b.Fatalf("no initial boundary for commodity %d", commodity)
	}
	if err := w.ledger.Move(src, w.agents[id].account, int(commodity), qty, ledger.Initial); err != nil {
		b.Fatalf("give = %v, want nil", err)
	}
}

// BenchmarkDispatch measures a hired dispatch of 1 line and of MaxShipmentLines lines.
// Every 256 dispatches the shipments are sent back to the shipper outside the timer, so
// the books stay the size of one batch.
func BenchmarkDispatch(b *testing.B) {
	for _, n := range []int{1, MaxShipmentLines} {
		b.Run(fmt.Sprintf("%d lines", n), func(b *testing.B) {
			b.ReportAllocs()
			w, all := transportBenchWorld(b)
			lines := all[:n]
			haul := commodityOfB(b, w, "haul")
			req := ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: lines}
			a := &w.agents[shipHome]
			const batch = 256
			top := func() {
				benchGive(b, w, shipHome, haul, 4*batch)
				for _, ln := range all {
					benchGive(b, w, shipHome, ln.Commodity, batch)
				}
			}
			top()
			done := 0
			for b.Loop() {
				if _, reason := w.dispatch(a, 0, &req, 0); reason != "" {
					b.Fatalf("dispatch refused %q", reason)
				}
				if done++; done%batch == 0 {
					b.StopTimer()
					w.recycleShipments(b)
					top()
					b.StartTimer()
				}
			}
		})
	}
}

// commodityOfB is commodityOf for a benchmark.
func commodityOfB(b *testing.B, w *World, id string) CommodityID {
	b.Helper()
	c, ok := w.cfg.commodityID(id)
	if !ok {
		b.Fatalf("commodity %q is not in the configuration", id)
	}
	return c
}

// recycleShipments sends every cargo in flight back to its shipper and frees its Transit
// account, as an arrival to the shipper would, without the events or the relocation.
func (w *World) recycleShipments(b *testing.B) {
	b.Helper()
	t := w.transport
	for i := range t.ships {
		s := &t.ships[i]
		for _, ln := range s.lines {
			if err := w.ledger.Move(s.account, w.agents[s.shipper].account, int(ln.Commodity), int64(ln.Quantity), ledger.TransitIn); err != nil {
				b.Fatalf("recycle = %v, want nil", err)
			}
		}
		t.free[s.from].Put(s.account)
		t.take(s.lines)
	}
	t.ships = t.ships[:0]
	t.due.Reset()
}

// BenchmarkArrivals measures P1.3: a tick with nothing due among 10 000 shipments in
// flight, which costs one look at the schedule, and a tick in which one trip to oneself
// arrives (the dispatch is outside the timer).
func BenchmarkArrivals(b *testing.B) {
	b.Run("nothing due among 10000", func(b *testing.B) {
		b.ReportAllocs()
		w, _ := transportBenchWorld(b)
		for i := range 10_000 {
			w.transport.due.Push(1_000_000+int64(i), int64(i)) // never due in the benchmark
		}
		for b.Loop() {
			w.p13Arrivals()
		}
	})
	b.Run("one arrival", func(b *testing.B) {
		b.ReportAllocs()
		w, _ := transportBenchWorld(b)
		req := ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}
		a := &w.agents[shipHome]
		for b.Loop() {
			b.StopTimer()
			if _, reason := w.dispatch(a, 0, &req, 0); reason != "" {
				b.Fatalf("dispatch refused %q", reason)
			}
			w.transport.ships[0].arrives = w.Tick()
			w.transport.due.Push(w.Tick(), int64(w.transport.ships[0].id))
			b.StartTimer()
			w.p13Arrivals()
			b.StopTimer()
			if err := w.moveAgent(a, regHome, CauseConsumer); err != nil { // back home for the next trip
				b.Fatalf("moveAgent = %v, want nil", err)
			}
			b.StartTimer()
		}
	})
}
