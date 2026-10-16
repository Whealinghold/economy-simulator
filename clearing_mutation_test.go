package economysim

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// panicked runs f and returns what it panicked with, nil if it returned.
func panicked(f func()) (v any) {
	defer func() { v = recover() }()
	f()
	return nil
}

// TestResize_KeepsSpareCapacity verifies that resize reslices a slice whose
// capacity is enough, keeping its backing array, and allocates only when the
// capacity is short.
//
// Guard: the scratch is reused so that a warmed step allocates nothing
// (spec-clearing.md NFR-CLR-002); a resize that reallocated when the capacity
// was larger than needed would defeat it.
func TestResize_KeepsSpareCapacity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		n         int
		wantCap   int
		wantShare bool
	}{
		{"shorter keeps the array", 3, 10, true},
		{"exact capacity keeps the array", 10, 10, true},
		{"longer allocates", 12, 12, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := make([]int, 5, 10)
			got := resize(s, tt.n)
			if len(got) != tt.n || cap(got) != tt.wantCap {
				t.Errorf("resize(len 5 cap 10, %d) has len %d cap %d, want len %d cap %d", tt.n, len(got), cap(got), tt.n, tt.wantCap)
			}
			if share := &got[0] == &s[0]; share != tt.wantShare {
				t.Errorf("resize(len 5 cap 10, %d) shares the array = %v, want %v", tt.n, share, tt.wantShare)
			}
		})
	}
}

// TestReasonOfCode_PastTheTable verifies that a code far beyond the reason
// table has no reason and does not index the table out of range.
//
// Guard: the table is indexed by code; the bound check must hold for every
// uint8, not only for the code just after the last row.
func TestReasonOfCode_PastTheTable(t *testing.T) {
	t.Parallel()
	for _, c := range []clearing.Code{clearing.Code(len(codeReasons)), clearing.Code(len(codeReasons) + 1), 255} {
		if r, ok := reasonOfCode(c); ok || r != "" {
			t.Errorf("reasonOfCode(%d) = %q, %v, want \"\", false", c, r, ok)
		}
	}
}

// TestBook_StockLotMatching verifies that the units of a named lot available
// are read from exactly the lot whose born tick and durability both match, net
// of its reservation, that an unnamed key matches no lot, and that the
// holdings and reservations are summed over every lot of the commodity.
//
// Requirement: FR-CLR-029
func TestBook_StockLotMatching(t *testing.T) {
	t.Parallel()
	// Commodity 0 neither spoils nor wears; commodity 1 wears (durability 5) and
	// spoils (shelf life 3, so born in {0, -1, -2}).
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{}, {Durability: 5, ShelfLife: 3}},
		Regions:     1, MaxMoney: MaxMoney, MaxQuantity: MaxQuantity,
	})
	if err != nil {
		t.Fatalf("ledger.New: %v", err)
	}
	var goodsB [2]int
	for c := range goodsB {
		if goodsB[c], err = l.OpenBoundary(ledger.Initial, c); err != nil {
			t.Fatalf("OpenBoundary: %v", err)
		}
	}
	acct, err := l.Open(ledger.KindAgent, 0, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Lots of commodity 1: (born, durability, units).
	// (0,3,2) (0,5,7) (-1,3,11) (-1,5,13); total 33; 3 of (-1,5) reserved.
	for _, k := range []struct{ born, dur, qty int64 }{{0, 3, 2}, {0, 5, 7}, {-1, 3, 11}, {-1, 5, 13}} {
		key := ledger.Key{Commodity: 1, Born: k.born, Durability: k.dur}
		if err := l.MoveStated(goodsB[1], acct, key, k.qty, ledger.Initial); err != nil {
			t.Fatalf("MoveStated(%+v): %v", k, err)
		}
	}
	if err := l.ReserveLot(acct, ledger.Key{Commodity: 1, Born: -1, Durability: 5}, 3); err != nil {
		t.Fatalf("ReserveLot: %v", err)
	}
	if err := l.Move(goodsB[0], acct, 0, 5, ledger.Initial); err != nil { // another commodity, born 0 durability 0
		t.Fatalf("Move: %v", err)
	}
	var sc clearScratch
	b := newBook(0, 1, 0, &sc)
	tests := []struct {
		name      string
		lot       LotKey
		wantAvail int64
	}{
		{"born 0 durability 3", LotKey{Named: true, Born: 0, Durability: 3}, 2},
		{"born 0 durability 5", LotKey{Named: true, Born: 0, Durability: 5}, 7},
		{"born -1 durability 3", LotKey{Named: true, Born: -1, Durability: 3}, 11},
		{"born -1 durability 5 net of reservation", LotKey{Named: true, Born: -1, Durability: 5}, 10}, // 13 - 3
		{"absent lot", LotKey{Named: true, Born: -2, Durability: 4}, 0},
		{"unnamed key matches no lot", LotKey{Born: 0, Durability: 3}, 0},
		{"unnamed zero key matches no lot", LotKey{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			held, reserved, avail := b.stock(l, acct, tt.lot)
			if held != 33 || reserved != 3 || avail != tt.wantAvail {
				t.Errorf("stock(%+v) = held %d, reserved %d, lot available %d, want 33, 3, %d", tt.lot, held, reserved, avail, tt.wantAvail)
			}
		})
	}
}

// TestBook_InvariantPanics verifies that a programmer error at the boundary of
// the book panics instead of being admitted, reserved, released or settled:
// an order of another commodity (above and below the book's), a side that is
// neither bid nor ask (below and above), an unreadable ledger account, a
// reservation or release the ledger refuses, a refused settlement step and a
// clearMarket given another tick than the book's (later and earlier).
//
// Guard: every one of these branches is unreachable through admission and the
// trade-list check, so only a direct call can show that it stops loudly.
func TestBook_InvariantPanics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		do   func(m *mkMarket)
	}{
		{"commodity above the book's", func(m *mkMarket) {
			b := newBook(0, 1, 0, &m.sc)
			b.admit(m.l, m.facts(0), m.bid(0, 2, 1, 1))
		}},
		{"commodity below the book's", func(m *mkMarket) {
			b := newBook(0, 1, 0, &m.sc)
			b.admit(m.l, m.facts(0), m.bid(0, 0, 1, 1))
		}},
		{"side below bid", func(m *mkMarket) {
			o := m.bid(0, 0, 1, 1)
			o.Side = Side(-1)
			newBook(0, 0, 0, &m.sc).admit(m.l, m.facts(0), o)
		}},
		{"side above ask", func(m *mkMarket) {
			o := m.bid(0, 0, 1, 1)
			o.Side = Side(2)
			newBook(0, 0, 0, &m.sc).admit(m.l, m.facts(0), o)
		}},
		{"stock of an unreadable account", func(m *mkMarket) {
			newBook(0, 0, 0, &m.sc).stock(m.l, 9999, LotKey{})
		}},
		{"reserve a bid with no funds", func(m *mkMarket) {
			newBook(0, 0, 0, &m.sc).reserve(m.l, m.facts(0), m.bid(0, 0, 5, 1), 0)
		}},
		{"reserve an ask with no stock", func(m *mkMarket) {
			newBook(0, 0, 0, &m.sc).reserve(m.l, m.facts(0), m.ask(0, 0, 5, 1), 0)
		}},
		{"release a reservation never made", func(m *mkMarket) {
			b := newBook(0, 0, 0, &m.sc)
			b.entries = append(b.entries, bookEntry{agent: m.agent(0), account: m.acct[0], side: Ask, price: 5, qty: 3})
			b.release(m.l, nil)
		}},
		{"settlement step refused", func(*mkMarket) {
			mustSettle(errors.New("refused"), "a step")
		}},
		{"clearMarket tick after the book's", func(m *mkMarket) {
			b := newBook(0, 0, 5, &m.sc)
			clearMarket(m.l, b, AverageClearing{}, nil, seedReport(1), 6, &quiet, nil, nil)
		}},
		{"clearMarket tick before the book's", func(m *mkMarket) {
			b := newBook(0, 0, 5, &m.sc)
			clearMarket(m.l, b, AverageClearing{}, nil, seedReport(1), 4, &quiet, nil, nil)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMkMarket(t, 1)
			if v := panicked(func() { tt.do(m) }); v == nil {
				t.Errorf("%s did not panic, want a violated-invariant panic", tt.name)
			}
		})
	}
	t.Run("settle no error does not panic", func(t *testing.T) {
		t.Parallel()
		if v := panicked(func() { mustSettle(nil, "a step") }); v != nil {
			t.Errorf("mustSettle(nil) panicked with %v, want none", v)
		}
	})
}

// TestBook_BidNamesNoLot verifies that the Lot of a bid is ignored: the entry
// the book keeps for an admitted bid carries the zero LotKey.
//
// Requirement: FR-CLR-029
func TestBook_BidNamesNoLot(t *testing.T) {
	t.Parallel()
	m := newMkMarket(t, 1)
	m.fund(0, 100)
	b := newBook(0, 0, 0, &m.sc)
	o := m.bid(0, 0, 10, 2)
	o.Lot = LotKey{Named: true, Born: 3, Durability: 4}
	if r, ok := b.admit(m.l, m.facts(0), o); !ok {
		t.Fatalf("admit(bid with a lot) = %q, false, want admitted", r)
	}
	if len(b.entries) != 1 || b.entries[0].lot != (LotKey{}) {
		t.Errorf("entries = %+v, want one entry with the zero LotKey", b.entries)
	}
}

// TestListFailure_Text verifies the audit text of every failure code of
// clearing.Check, which the ErrInvalidTrade error carries to the operator.
//
// Requirement: FR-CLR-024
func TestListFailure_Text(t *testing.T) {
	t.Parallel()
	tests := []struct {
		code clearing.Code
		want string
	}{
		{clearing.BadCommodity, "a trade names another commodity than the market's"},
		{clearing.NoOrder, "a trade names a buyer with no bid or a seller with no ask"},
		{clearing.BadQuantity, "a trade has a quantity below 1"},
		{clearing.BadPrice, "a trade price is outside the limits of its orders"},
		{clearing.SameOwner, "a trade has one agent as buyer and seller"},
		{clearing.OverFilled, "the trades of an order exceed its quantity"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := listFailure(tt.code); got != tt.want {
				t.Errorf("listFailure(%d) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

// TestBook_NilLoggerIsAllowed verifies that a nil *slog.Logger is the
// no-logging case: a rejection and a market clearing with a nil logger neither
// panic nor fail.
//
// Requirement: FR-CLR-033
func TestBook_NilLoggerIsAllowed(t *testing.T) {
	t.Parallel()
	m := newMkMarket(t, 2)
	m.fund(0, 100)
	m.stock(1, 0, 5)
	b := newBook(0, 0, 0, &m.sc)
	if v := panicked(func() { b.emitRejected(&quiet, nil, m.bid(0, 0, 1, 1), ReasonOverflow) }); v != nil {
		t.Errorf("emitRejected with a nil logger panicked with %v, want none", v)
	}
	if got := m.admitAll(b, m.bid(0, 0, 10, 1), m.ask(1, 0, 5, 1)); !slices.Equal(got, []Reason{"", ""}) {
		t.Fatalf("admission = %v, want both admitted", got)
	}
	var err error
	if v := panicked(func() {
		_, _, err = clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 0, 0, 0), seedReport(1), 0, &quiet, nil, nil)
	}); v != nil || err != nil {
		t.Errorf("clearMarket with a nil logger = panic %v, error %v, want neither", v, err)
	}
}
