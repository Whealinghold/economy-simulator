package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// estateRun runs pack one for ticks steps with the audit on and returns, per step,
// the events of the step and the snapshots before and after it. md replaces the
// estate markdown of the north when it is not negative.
func estateRun(t *testing.T, ticks int, md int64) (steps [][]economysim.Event, before, after []economysim.Snapshot) {
	t.Helper()
	cfg := loadValidPack(t, packOne...)
	if md >= 0 {
		cfg.Regions[0].EstateMarkdown = md
	}
	rec := &recorder{}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithAudit(), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	snap := func() economysim.Snapshot {
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot error = %v, want nil", err)
		}
		return s
	}
	rec.forget()
	for range ticks {
		before = append(before, snap())
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
		steps = append(steps, rec.events)
		rec.forget()
		after = append(after, snap())
	}
	return steps, before, after
}

// isEstate reports whether an agent ID is an estate pseudo-agent: the region count
// is 1 in pack one.
func isEstate(a economysim.AgentID) bool { return a < 1 }

// TestEstates_NoBidEverCarriesAnEstateID verifies that over a run with removals no
// OrderPlaced and no OrderRejected on the bid side names an estate: the estate
// places asks only.
//
// Requirement: FR-EST-006
func TestEstates_NoBidEverCarriesAnEstateID(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	steps, _, _ := estateRun(t, 200, -1)
	var asks, removals int
	for tick, evs := range steps {
		for _, e := range evs {
			switch e := e.(type) {
			case economysim.OrderPlaced:
				if isEstate(e.Agent) && e.Side == economysim.Bid {
					t.Errorf("tick %d: bid placed by estate %d, want none", tick, e.Agent)
				}
				if isEstate(e.Agent) {
					asks++
				}
			case economysim.OrderRejected:
				if isEstate(e.Agent) && e.Side == economysim.Bid {
					t.Errorf("tick %d: bid rejected for estate %d, want none", tick, e.Agent)
				}
			case economysim.EstateOpened:
				removals++
			}
		}
	}
	if asks == 0 || removals == 0 {
		t.Errorf("estate asks %d and removals %d in 200 ticks, want both above 0: the run proves nothing otherwise", asks, removals)
	}
}

// TestEstates_OneAskPerHeldLot verifies that the units the estate places asks for
// in a tick never exceed the units it held at the start of the tick, commodity by
// commodity (so it asks nothing for a commodity it holds none of), and that in
// some tick it asks for all it holds: its asks are its held lots, one ask per lot.
//
// Requirement: FR-EST-007
func TestEstates_OneAskPerHeldLot(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	steps, before, _ := estateRun(t, 200, -1)
	var checked int
	for tick, evs := range steps {
		asked := map[economysim.CommodityID]economysim.Quantity{}
		for _, e := range evs {
			if o, ok := e.(economysim.OrderPlaced); ok && isEstate(o.Agent) {
				asked[o.Commodity] += o.Quantity
			}
		}
		for _, m := range before[tick].Markets {
			// Rejected asks are not counted, so the placed units are at most the held ones.
			if got := asked[m.State.Commodity]; got > m.HeldByEstate {
				t.Errorf("tick %d commodity %d: estate asked %d units, held %d at the start", tick, m.State.Commodity, got, m.HeldByEstate)
			}
			if m.HeldByEstate > 0 && asked[m.State.Commodity] == m.HeldByEstate {
				checked++
			}
		}
	}
	if checked == 0 {
		t.Error("no tick in which the estate asked for all it held, want some: the run proves nothing otherwise")
	}
}

// TestEstates_MarkdownAtP53 verifies the markdown: a commodity the estate lists
// as one ask of the same size on two ticks in a row has its price fall by
// trunc(price * EstateMarkdown / 10 000) between them (or sit on the floor), and
// with a markdown of 0 it never falls.
//
// Arithmetic: a price p with a markdown of m basis points becomes p - p*m/10 000,
// truncated.
//
// Requirement: FR-EST-009
func TestEstates_MarkdownAtP53(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, md := range []int64{0, 500} {
		steps, _, _ := estateRun(t, 200, md)
		type ask struct {
			price economysim.Price
			qty   economysim.Quantity
			n     int
		}
		prev := map[economysim.CommodityID]ask{}
		var fell, same int
		for tick, evs := range steps {
			cur := map[economysim.CommodityID]ask{}
			for _, e := range evs {
				if o, ok := e.(economysim.OrderPlaced); ok && isEstate(o.Agent) {
					a := cur[o.Commodity]
					cur[o.Commodity] = ask{o.Limit, o.Quantity, a.n + 1}
				}
			}
			for c, a := range cur {
				p, ok := prev[c]
				if !ok || p.n != 1 || a.n != 1 || p.qty != a.qty {
					continue
				}
				cut := economysim.Price(int64(p.price) * md / 10_000)
				switch {
				case md == 0 && a.price != p.price:
					t.Errorf("tick %d commodity %d: price %d -> %d with a markdown of 0, want it unchanged", tick, c, p.price, a.price)
				case a.price == p.price-cut && cut > 0:
					fell++
				case a.price == p.price:
					same++
				case a.price > p.price-cut && a.price < p.price:
					same++ // the markdown stops at the resolution floor, which is above p - cut
				default:
					t.Errorf("tick %d commodity %d: price %d -> %d with a markdown of %d bp, want %d or the floor", tick, c, p.price, a.price, md, p.price-cut)
				}
			}
			prev = cur
		}
		if md > 0 && fell == 0 {
			t.Errorf("markdown %d bp: no price fell by its markdown in 200 ticks (%d unchanged), want some: the run proves nothing otherwise", md, same)
		}
	}
}

// TestEstates_ProceedsArePool verifies that the proceeds of the estate's sales are
// the pool: in every tick the pool moves by exactly the estate's trade proceeds,
// plus the money of the removals it took in, less the money it paid out (start
// money of entrants and the households of a birth), and RegionRow.MoneyPool is the
// estate account's money (the audit and the sectoral identity hold at every tick).
//
// Requirement: FR-EST-013
func TestEstates_ProceedsArePool(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	steps, before, after := estateRun(t, 200, -1)
	var sold int64
	for tick, evs := range steps {
		var proceeds, intake, paid int64
		for _, e := range evs {
			switch e := e.(type) {
			case economysim.TradeExecuted:
				if isEstate(e.Seller) {
					proceeds += int64(e.Price) * int64(e.Quantity)
				}
			case economysim.AgentBankrupted:
				intake += int64(e.Money)
			case economysim.WeightChanged:
				intake -= int64(e.Money)
			case economysim.AgentSpawned:
				paid += int64(e.StartMoney)
			}
		}
		sold += proceeds
		got := int64(after[tick].Regions[0].MoneyPool) - int64(before[tick].Regions[0].MoneyPool)
		if want := proceeds + intake - paid; got != want {
			t.Errorf("tick %d: the pool moved by %d, want %d = sales %d + removals' money %d - paid %d", tick, got, want, proceeds, intake, paid)
		}
	}
	if sold == 0 {
		t.Error("the estate sold nothing in 200 ticks, want some: the run proves nothing otherwise")
	}
}
