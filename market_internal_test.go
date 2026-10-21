package economysim

import (
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
)

// marketFixture is the table of two regions and three commodities with a price
// window of three, built from a configuration that is not validated: the table
// reads only the counts and the window capacity. seeded records the markets the
// seed function was asked for, in order.
func marketFixture() (t marketTable, seeded [][2]int) {
	cfg := &Config{Regions: make([]Region, 2), Commodities: make([]Commodity, 3), World: WorldParams{PriceWindow: 3}}
	t = newMarketTable(cfg, &clearScratch{}, func(r RegionID, c CommodityID) Price {
		seeded = append(seeded, [2]int{int(r), int(c)})
		return Price(1000 + 100*int(r) + 10*int(c))
	})
	return t, seeded
}

// TestMarketTable_OrderAndSeed verifies that the table holds one market for
// every (region, commodity) in region-major order, that the seed function is
// asked for each market once in that order, and that each market starts at its
// seed price with age 0, no volume, an empty window, a run of 0 and an empty
// book of its own pair (spec-clearing.md 5.4, FR-CLR-020; spec-beliefs.md 8.1,
// FR-BEL-039).
//
// Requirement: FR-CLR-020, FR-BEL-039
func TestMarketTable_OrderAndSeed(t *testing.T) {
	t.Parallel()
	tab, seeded := marketFixture()
	// Six markets: (0,0) (0,1) (0,2) (1,0) (1,1) (1,2); the seed of (r, c) is
	// 1000 + 100 r + 10 c, so (1,2) is 1120.
	wantOrder := [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}}
	if !slices.Equal(seeded, wantOrder) {
		t.Errorf("seed asked for %v, want %v: once each, region-major", seeded, wantOrder)
	}
	if len(tab.reports) != 6 || len(tab.windows) != 6 || len(tab.books) != 6 || len(tab.boundRun) != 6 {
		t.Fatalf("table sizes = %d, %d, %d, %d, want 6 each", len(tab.reports), len(tab.windows), len(tab.books), len(tab.boundRun))
	}
	for i := range tab.reports {
		r, c := RegionID(i/3), CommodityID(i%3)
		if got := tab.index(r, c); got != i {
			t.Errorf("index(%d, %d) = %d, want %d", r, c, got, i)
		}
		if want := seedReport(Price(1000 + 100*int(r) + 10*int(c))); tab.reports[i] != want {
			t.Errorf("report %d = %+v, want the seed report %+v", i, tab.reports[i], want)
		}
		if tab.windows[i].Len() != 0 || tab.boundRun[i] != 0 {
			t.Errorf("market %d window length %d, run %d, want an empty window and a run of 0", i, tab.windows[i].Len(), tab.boundRun[i])
		}
		if b := tab.books[i]; b.region != r || b.commodity != c || len(b.entries) != 0 || b.tick != 0 {
			t.Errorf("book %d = (%d, %d, %d entries, tick %d), want an empty book of (%d, %d) at tick 0", i, b.region, b.commodity, len(b.entries), b.tick, r, c)
		}
	}
}

// TestMarketTable_PushOnlyOnATrade verifies the push of P5.1: the published
// price of a market goes into its window only in a tick in which units traded,
// once per tick, so the window holds one price per trading tick and a market
// that did not trade is left alone; the ring keeps the last PriceWindow prices
// (spec-beliefs.md 7.4, FR-BEL-020).
//
// Requirement: FR-BEL-020
func TestMarketTable_PushOnlyOnATrade(t *testing.T) {
	t.Parallel()
	tab, _ := marketFixture()
	// Market 1 trades at 900 then 1 100 (mean (900 + 1100) / 2 = 1 000); market 4
	// has a price and a volume of 0 (no trade: age 1); market 0 never trades.
	tab.reports[1] = marketReport{Price: 900, Volume: 5}
	tab.reports[4] = marketReport{Price: 1_110, Age: 1}
	tab.pushTraded()
	tab.reports[1] = marketReport{Price: 1_100, Volume: 2}
	tab.pushTraded()
	if got, want := tab.windows[1].Len(), 2; got != want {
		t.Errorf("window of the market that traded twice has %d prices, want %d", got, want)
	}
	if mean, ok := tab.windows[1].Mean(); !ok || mean != 1_000 {
		t.Errorf("window mean = (%d, %v), want (1000, true): (900 + 1100) / 2", mean, ok)
	}
	for _, i := range []int{0, 2, 3, 4, 5} {
		if tab.windows[i].Len() != 0 {
			t.Errorf("window %d has %d prices, want 0: no unit traded there", i, tab.windows[i].Len())
		}
	}
	// A third and fourth trading tick: the ring of capacity 3 keeps the last three.
	for _, p := range []Price{1_200, 1_300} {
		tab.reports[1] = marketReport{Price: p, Volume: 1}
		tab.pushTraded()
	}
	if got, want := tab.windows[1].Values(nil), []int64{1_100, 1_200, 1_300}; !slices.Equal(got, want) {
		t.Errorf("window values = %v, want %v: the last three trading ticks, oldest first", got, want)
	}
}

// TestMarketTable_WindowRestoreRoundTrip verifies that the saved form of a
// market's window, its values oldest first, restores into a window of the same
// capacity that equals the original: the same values, the same mean, low and
// high, and the same behaviour at the next push (the write position is not part
// of its meaning).
//
// Requirement: FR-BEL-035
func TestMarketTable_WindowRestoreRoundTrip(t *testing.T) {
	t.Parallel()
	tab, _ := marketFixture()
	for _, p := range []Price{900, 1_100, 1_300, 1_500} { // four pushes into a ring of three: the first is overwritten
		tab.reports[2] = marketReport{Price: p, Volume: 1}
		tab.pushTraded()
	}
	saved := tab.windows[2].Values(nil)
	fresh, _ := marketFixture()
	if !fresh.windows[2].Restore(saved) {
		t.Fatalf("Restore(%v) = false, want true", saved)
	}
	for _, w := range []*beliefs.Window{&tab.windows[2], &fresh.windows[2]} {
		w.Push(1_700)
	}
	got, want := fresh.windows[2].Values(nil), tab.windows[2].Values(nil)
	if !slices.Equal(got, want) || !slices.Equal(want, []int64{1_300, 1_500, 1_700}) {
		t.Errorf("restored window after a push = %v, original = %v, want both [1300 1500 1700]", got, want)
	}
	gm, _ := fresh.windows[2].Mean()
	wm, _ := tab.windows[2].Mean()
	if gm != wm {
		t.Errorf("restored mean = %d, original %d, want equal", gm, wm)
	}
}

// TestMarketTable_IntentOfFindsTheBidOfABuyer verifies that, with the slots loaded
// for a market, intentOf returns the record of the bid of each buyer, that
// clearSlots undoes loadSlots, and that a buyer with no bid in the market is a
// violated invariant whether its ID lies past the slot table or inside it with a
// slot of 0.
//
// Arithmetic: market 0 holds a bid of agent 2 (entry 0), an ask of agent 3 (entry
// 1) and a bid of agent 1 (entry 2); the slot of an agent is its entry's index
// plus 1, so slotOf = [0, 3, 1, 0] after loadSlots, and the table is as long as
// the highest bid agent plus 1. Agent 3 has only an ask, so its slot is 0 inside
// the table; agent 4 is the length of the table, one past its last index.
//
// Requirement: FR-WLD-047
// Guard: a buyer found at the wrong slot would credit another household's purpose
// and target to a trade; the boundary at the table length is a read past its end.
func TestMarketTable_IntentOfFindsTheBidOfABuyer(t *testing.T) {
	t.Parallel()
	tab, _ := marketFixture()
	b := tab.books[0]
	b.entries = append(b.entries,
		bookEntry{agent: 2, side: Bid}, bookEntry{agent: 3, side: Ask}, bookEntry{agent: 1, side: Bid})
	tab.intents[0] = []intentRec{{target: 20}, {target: 30}, {target: 10}}
	tab.loadSlots(0)
	if want := []int32{0, 3, 1}; !slices.Equal(tab.slotOf, want) {
		t.Fatalf("slotOf after loadSlots = %v, want %v", tab.slotOf, want)
	}
	for _, tt := range []struct {
		buyer  AgentID
		target Quantity
	}{{1, 10}, {2, 20}} {
		if got := tab.intentOf(0, tt.buyer); got.target != tt.target {
			t.Errorf("intentOf(0, %d).target = %d, want %d", tt.buyer, got.target, tt.target)
		}
	}
	for _, buyer := range []AgentID{0, 3, 4, 9} {
		func() {
			defer func() {
				if msg, _ := recover().(string); !strings.Contains(msg, "invariant") {
					t.Errorf("intentOf(0, %d) recovered %q, want an invariant panic: the agent has no bid", buyer, msg)
				}
			}()
			tab.intentOf(0, buyer)
			t.Errorf("intentOf(0, %d) returned, want a panic: the agent has no bid", buyer)
		}()
	}
	tab.clearSlots(0)
	if want := []int32{0, 0, 0}; !slices.Equal(tab.slotOf, want) {
		t.Errorf("slotOf after clearSlots = %v, want %v", tab.slotOf, want)
	}
}
