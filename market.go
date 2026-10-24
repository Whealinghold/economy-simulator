package economysim

import (
	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/codec"
)

// This file is the record of the markets (spec-clearing.md 5.4, 16;
// spec-beliefs.md 7): for every (region, commodity) the report the market
// published at its last clearing, the price window, the run of the Warn "price
// beliefs at bound" and the book. The clearing and the beliefs rows left the
// holding of these to the world (plan-phase-1-11.md Phase 3). The report and the
// window are saved state; the run is a diagnostic (D145, D156); the book is
// scratch, kept until the outcomes of the tick are delivered (D154).
//
// The table is region-major, the order of (RegionID, CommodityID), and holds one
// market for every commodity of every region: the slice has no outside node, so
// no region is left out (spec-world.md 6.1.11 leaves an outside node out when
// that module is built). The slices are parallel and are the ones viewCtx
// reads (reports, windows), so the view needs no copy.

// marketTable is the record of every market of a world.
type marketTable struct {
	commodities int
	// reports and windows are the market report and the price window of every
	// (region, commodity), at index region * commodities + commodity.
	reports []marketReport
	windows []beliefs.Window
	// boundRun is the run of the Warn "price beliefs at bound" of each market:
	// the consecutive ticks on a bound (beliefs.Run.Count, spec-beliefs.md 12.2).
	boundRun []int64
	// books is the book of each market for the tick in progress.
	books []*book
	// intents is, per market, the facts of each admitted order that the book does
	// not keep, parallel to the book's entries (same index): D194. An estate's ask
	// has the zero record and no outcome.
	intents [][]intentRec
	// slotOf maps a buyer to the entry of its bid in the market being cleared, as
	// entry index + 1 (0 is none). It is filled for one market at P4.1 and cleared
	// when that market is done, so one table serves every market (D194).
	slotOf []int32
	// asks and trades are the scratch of P3.4 and P4.1.
	asks   []Order
	trades []Trade
}

// intentRec is what the world keeps of an admitted intent for P4.1 (the purpose
// of a bid, for the welfare feeds) and P5.4 (the facts of the outcome): the
// intent after normalise (spec-agents.md 6.5.2). Order and bookEntry carry none
// of it (D194).
type intentRec struct {
	purpose  Purpose
	target   Quantity
	reserve  Quantity
	unitCost Price
}

// newMarketTable builds the table for a canonical configuration: each report
// is the seed report of the price seed gives (spec-clearing.md 5.4, FR-CLR-020),
// each window is empty with the capacity PriceWindow and the bound MaxPrice
// (spec-beliefs.md 7.1), each run is 0 and each book is an empty book of tick 0
// on the scratch sc. seed is called once for each market, in table order.
func newMarketTable(cfg *Config, sc *clearScratch, seed func(RegionID, CommodityID) Price) marketTable {
	nc, n := len(cfg.Commodities), len(cfg.Regions)*len(cfg.Commodities)
	t := marketTable{
		commodities: nc,
		reports:     make([]marketReport, n),
		windows:     make([]beliefs.Window, n),
		boundRun:    make([]int64, n),
		books:       make([]*book, n),
		intents:     make([][]intentRec, n),
	}
	for i := range n {
		region, commodity := RegionID(i/nc), CommodityID(i%nc)
		t.reports[i] = seedReport(seed(region, commodity))
		t.windows[i] = beliefs.NewWindow(int(cfg.World.PriceWindow), MaxPrice)
		t.books[i] = newBook(region, commodity, 0, sc)
	}
	return t
}

// index returns the position of the market (region, commodity) in the table.
// The pair must be in the configuration.
func (t *marketTable) index(region RegionID, commodity CommodityID) int {
	return int(region)*t.commodities + int(commodity)
}

// pushTraded is the push of P5.1 (spec-beliefs.md 7.4): the published price of
// every market in which units traded in the tick goes into its window, once, in
// table order. A market that did not trade has age above 0 and is left alone, so
// the window holds one price per tick in which units traded. It allocates
// nothing: a window is a ring allocated at construction.
func (t *marketTable) pushTraded() {
	for i := range t.reports {
		if t.reports[i].Volume > 0 {
			t.windows[i].Push(int64(t.reports[i].Price))
		}
	}
}

// refreshViewCtx assembles the context the view builder reads from what the
// world holds at this moment (plan-phase-1-11.md D165): the tables, the market
// reports and windows (the published prices left by P4.1 of the previous tick, or
// the seed price at tick 0), the cost anchors of P1.2 and the moment of P1.1. It
// is called at the start of P2.5 and again at the start of P3.2, and the context
// is never kept across positions. It points at the world's own slices and copies
// none.
func (w *World) refreshViewCtx() {
	w.tk.vctx = viewCtx{
		cfg: &w.cfg, roles: &w.roles, recipes: &w.recipes, l: w.ledger, prod: w.prod,
		yields: &w.yields, shocks: &w.shocks, bp: w.bp, moment: w.tk.moment,
		reports: w.markets.reports, windows: w.markets.windows,
	}
}

// reset empties every book and its intent records for the tick, keeping the
// storage. It is the start of P3.2, after the reservations of the last tick are
// released (spec-world.md 6.1.6).
func (t *marketTable) reset(tick int64) {
	for i, b := range t.books {
		b.reset(tick)
		t.intents[i] = t.intents[i][:0]
	}
}

// record appends the record of the order just admitted to the book of market i,
// so that it sits at the index of the book's entry (D194).
func (t *marketTable) record(i int, r intentRec) { t.intents[i] = append(t.intents[i], r) }

// loadSlots writes the slot of every bid of the book of market i, before its
// clearing. A buyer has at most one bid in a market (duplicate_order), so the
// slot of a (market, buyer) pair is unique.
func (t *marketTable) loadSlots(i int) {
	for j := range t.books[i].entries {
		e := &t.books[i].entries[j]
		if e.side != Bid {
			continue
		}
		for int(e.agent) >= len(t.slotOf) {
			t.slotOf = append(t.slotOf, 0)
		}
		t.slotOf[e.agent] = int32(j) + 1
	}
}

// clearSlots undoes loadSlots for market i.
func (t *marketTable) clearSlots(i int) {
	for j := range t.books[i].entries {
		if e := &t.books[i].entries[j]; e.side == Bid {
			t.slotOf[e.agent] = 0
		}
	}
}

// intentOf returns the record of the bid of buyer in market i, found through the
// slot loaded for the clearing. A trade names a buyer with a bid (the check of
// the trade list), so a buyer with no slot is a violated invariant.
func (t *marketTable) intentOf(i int, buyer AgentID) intentRec {
	if int(buyer) >= len(t.slotOf) || t.slotOf[buyer] == 0 {
		invariant("a settled trade names a buyer with no bid in the market")
	}
	return t.intents[i][t.slotOf[buyer]-1]
}

// saveMarkets is tag 4 (spec-checkpoint.md 4.2; plan 2-2 D248): for each market in
// table order, the report of its last clearing, its price window oldest first and
// its run of the Warn "price beliefs at bound", dense.
func (w *World) saveMarkets(cw *codec.Writer) error {
	t := &w.markets
	var prices []int64
	for i := range t.reports {
		saveReport(cw, &t.reports[i])
		prices = t.windows[i].Values(prices[:0])
		cw.Count(len(prices))
		for _, p := range prices {
			cw.U(unsigned(p))
		}
		cw.U(unsigned(t.boundRun[i]))
	}
	return nil
}

// saveReport writes the market report: the price, the age, the volume, the
// unmatched units of each side and each best unmatched price as an optional value.
func saveReport(cw *codec.Writer, r *marketReport) {
	cw.U(unsigned(r.Price))
	cw.U(unsigned(r.Age))
	cw.U(unsigned(r.Volume))
	cw.U(unsigned(r.UnmatchedBid))
	cw.U(unsigned(r.UnmatchedAsk))
	cw.B(r.HasBestBid)
	if r.HasBestBid {
		cw.U(unsigned(r.BestBid))
	}
	cw.B(r.HasBestAsk)
	if r.HasBestAsk {
		cw.U(unsigned(r.BestAsk))
	}
}
