package economysim

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// mkMarket is a real ledger with a few agents of one region, two commodities
// (0 does not wear, 1 wears with a full durability of 5, so it has named lots)
// and the boundary accounts of the Initial reason, plus the scratch and the
// receipt tally a world would own.
type mkMarket struct {
	t      testing.TB
	l      *ledger.Ledger
	moneyB int
	goodsB [2]int
	acct   []int
	tally  receiptTally
	sc     clearScratch
}

// newMkMarket builds the ledger with n agents. AgentID of agent i is i+10, so
// that an agent ID is never mistaken for a ledger account.
func newMkMarket(t testing.TB, n int) *mkMarket {
	t.Helper()
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{}, {Durability: 5}},
		Regions:     1,
		MaxMoney:    MaxMoney,
		MaxQuantity: MaxQuantity,
	})
	if err != nil {
		t.Fatalf("ledger.New: %v", err)
	}
	m := &mkMarket{t: t, l: l}
	open := func(id int, err error) int {
		t.Helper()
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		return id
	}
	m.moneyB = open(l.OpenBoundary(ledger.Initial, ledger.Money))
	for c := range m.goodsB {
		m.goodsB[c] = open(l.OpenBoundary(ledger.Initial, c))
	}
	for i := range n {
		m.acct = append(m.acct, open(l.Open(ledger.KindAgent, i, 0)))
	}
	return m
}

func (m *mkMarket) fund(i int, amount int64) {
	m.t.Helper()
	if err := m.l.Transfer(m.moneyB, m.acct[i], amount, ledger.Initial); err != nil {
		m.t.Fatalf("fund(%d, %d): %v", i, amount, err)
	}
}

func (m *mkMarket) stock(i, commodity int, qty int64) {
	m.t.Helper()
	if err := m.l.Move(m.goodsB[commodity], m.acct[i], commodity, qty, ledger.Initial); err != nil {
		m.t.Fatalf("stock(%d, %d, %d): %v", i, commodity, qty, err)
	}
}

// lot gives agent i qty units of commodity 1 of the lot with durability dur.
func (m *mkMarket) lot(i int, dur, qty int64) {
	m.t.Helper()
	key := ledger.Key{Commodity: 1, Durability: dur}
	if err := m.l.MoveStated(m.goodsB[1], m.acct[i], key, qty, ledger.Initial); err != nil {
		m.t.Fatalf("lot(%d, %d, %d): %v", i, dur, qty, err)
	}
}

func (m *mkMarket) agent(i int) AgentID { return AgentID(i + 10) }

// facts is the world's side of the facts of agent i: known, allowed, tradable.
func (m *mkMarket) facts(i int) orderFacts {
	return orderFacts{
		Account: m.acct[i], AgentKnown: true, SideAllowed: true,
		CommodityKnown: true, Tradable: true, Tally: &m.tally,
	}
}

func (m *mkMarket) bid(i, commodity int, price, qty int64) Order {
	return Order{Agent: m.agent(i), Commodity: CommodityID(commodity), Side: Bid, Price: Price(price), Quantity: Quantity(qty)}
}

func (m *mkMarket) ask(i, commodity int, price, qty int64) Order {
	return Order{Agent: m.agent(i), Commodity: CommodityID(commodity), Side: Ask, Price: Price(price), Quantity: Quantity(qty)}
}

func (m *mkMarket) askLot(i, commodity int, price, qty, dur int64) Order {
	o := m.ask(i, commodity, price, qty)
	o.Lot = LotKey{Named: true, Durability: dur}
	return o
}

// admitAll admits the orders of agents in the order given and returns the
// reasons ("" for admitted).
func (m *mkMarket) admitAll(b *book, orders ...Order) []Reason {
	m.t.Helper()
	var got []Reason
	for _, o := range orders {
		i := int(o.Agent) - 10
		r, ok := b.admit(m.l, m.facts(i), o)
		if ok {
			r = ""
		}
		got = append(got, r)
	}
	return got
}

// ledgerState is everything observable about the ledger, for comparing a
// ledger with an earlier copy of itself with reflect.DeepEqual.
type ledgerState struct {
	Views []ledger.View
	Lots  [][]ledger.Lot
	Flows []ledger.Flow
}

func (m *mkMarket) state() ledgerState {
	var s ledgerState
	for id := 0; id < m.l.Len(); id++ {
		v, err := m.l.Account(id)
		if err != nil {
			m.t.Fatalf("Account(%d): %v", id, err)
		}
		lots, err := m.l.AppendLots(nil, id)
		if err != nil {
			m.t.Fatalf("AppendLots(%d): %v", id, err)
		}
		s.Views = append(s.Views, v)
		s.Lots = append(s.Lots, lots)
	}
	s.Flows = m.l.AppendFlows(nil, false)
	s.Flows = append(s.Flows, m.l.AppendFlows(nil, true)...)
	return s
}

// money and units read the balance, reservation and holding of agent i.
func (m *mkMarket) money(i int) (balance, reserved int64) {
	v, err := m.l.Account(m.acct[i])
	if err != nil {
		m.t.Fatalf("Account: %v", err)
	}
	return v.Balance, v.Reserved
}

func (m *mkMarket) units(i, commodity int) (held, reserved int64) {
	lots, err := m.l.AppendLots(nil, m.acct[i])
	if err != nil {
		m.t.Fatalf("AppendLots: %v", err)
	}
	for _, lt := range lots {
		if lt.Commodity == commodity {
			held += lt.Quantity
			reserved += lt.Reserved
		}
	}
	return held, reserved
}

// avgRNG is the keyed stream of a market, as the world gives it to Clear.
func avgRNG(seed uint64, tick int64, region, commodity int64) *rand.Rand {
	s := newStream(seed)
	s.keyedPair(siteClearing, tick, region, commodity)
	return s.rnd
}

var quiet = observers{}

// mkRecorder is a recording Observer.
type mkRecorder struct{ events []Event }

func (r *mkRecorder) Observe(e Event) { r.events = append(r.events, e) }

func (r *mkRecorder) observers() *observers {
	o := newObservers([]Observer{r})
	return &o
}

// fakeClearing returns a fixed list of trades whatever it is given: the
// hand-written fake of the extension point (testing.md 6.1).
type fakeClearing struct{ list []Trade }

func (f fakeClearing) Clear(_, _ []Order, _ *rand.Rand) []Trade { return f.list }

// mkWorked builds the book of spec-clearing.md 11 on a fresh market: agents
// 0..2 are B1 12x3, B2 10x2, B3 7x4 and 3..5 are A1 6x2, A2 9x4, A3 11x1. Each
// buyer holds 100 and each seller 5 units.
func mkWorked(t testing.TB) (*mkMarket, []Order) {
	m := newMkMarket(t, 6)
	for i := range 3 {
		m.fund(i, 100)
		m.stock(i+3, 0, 5)
	}
	return m, []Order{
		m.bid(0, 0, 12, 3), m.bid(1, 0, 10, 2), m.bid(2, 0, 7, 4),
		m.ask(3, 0, 6, 2), m.ask(4, 0, 9, 4), m.ask(5, 0, 11, 1),
	}
}

// TestBook_AdmissionReservesAndRejects verifies, against a real ledger, that a
// bid reserves limit times quantity of the available cash, an ask reserves its
// units, a second order of one side is duplicate_order and reserves nothing,
// and a rejected order leaves the ledger exactly as it was.
//
// Requirement: FR-CLR-004, FR-CLR-005, FR-CLR-006, FR-CLR-030
func TestBook_AdmissionReservesAndRejects(t *testing.T) {
	t.Parallel()
	m := newMkMarket(t, 3)
	m.fund(0, 100)
	m.stock(1, 0, 10)
	b := newBook(0, 0, 0, &m.sc)

	// A bid of 7 at 12 reserves 12*7 = 84 of 100: 16 stay available.
	got := m.admitAll(b, m.bid(0, 0, 12, 7))
	if !slices.Equal(got, []Reason{""}) {
		t.Fatalf("admit bid = %v, want admitted", got)
	}
	if bal, res := m.money(0); bal != 100 || res != 84 {
		t.Errorf("after bid: balance %d reserved %d, want 100 and 84 (12*7)", bal, res)
	}
	// An ask of 4 reserves 4 of 10 units.
	m.admitAll(b, m.ask(1, 0, 5, 4))
	if held, res := m.units(1, 0); held != 10 || res != 4 {
		t.Errorf("after ask: held %d reserved %d, want 10 and 4", held, res)
	}
	before := m.state()
	tests := []struct {
		name  string
		facts func(f *orderFacts)
		o     Order
		want  Reason
	}{
		{"second bid is a duplicate, not insufficient funds", nil, m.bid(0, 0, 1, 1000), ReasonDuplicateOrder},
		{"second ask is a duplicate", nil, m.ask(1, 0, 5, 1), ReasonDuplicateOrder},
		{"unknown agent", func(f *orderFacts) { f.AgentKnown = false }, m.bid(2, 0, 1, 1), ReasonUnknownAgent},
		{"side not allowed", func(f *orderFacts) { f.SideAllowed = false }, m.bid(2, 0, 1, 1), ReasonSideNotAllowed},
		{"unknown commodity", func(f *orderFacts) { f.CommodityKnown = false }, m.bid(2, 0, 1, 1), ReasonUnknownCommodity},
		{"not tradable", func(f *orderFacts) { f.Tradable = false }, m.bid(2, 0, 1, 1), ReasonNotTradable},
		{"price zero", nil, m.bid(2, 0, 0, 1), ReasonInvalidPrice},
		{"price above max", nil, m.bid(2, 0, MaxPrice+1, 1), ReasonInvalidPrice},
		{"quantity zero", nil, m.bid(2, 0, 1, 0), ReasonInvalidQuantity},
		{"quantity above max", nil, m.bid(2, 0, 1, MaxQuantity+1), ReasonInvalidQuantity},
		{"bid beyond cash", nil, m.bid(2, 0, 1, 1), ReasonInsufficientFunds},
		{"ask beyond stock", nil, m.ask(2, 0, 1, 1), ReasonInsufficientStock},
		{"second ask after an earmark is a duplicate", nil, m.ask(1, 0, 1, 1), ReasonDuplicateOrder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := m.facts(int(tt.o.Agent) - 10)
			if tt.facts != nil {
				tt.facts(&f)
			}
			r, ok := b.admit(m.l, f, tt.o)
			if ok || r != tt.want {
				t.Errorf("admit(%+v) = %q, %v, want %q, false", tt.o, r, ok, tt.want)
			}
		})
	}
	if after := m.state(); !reflect.DeepEqual(before, after) {
		t.Errorf("a rejected order changed the ledger:\nbefore %+v\nafter  %+v", before, after)
	}
	if len(b.entries) != 2 {
		t.Errorf("book holds %d orders, want the 2 admitted", len(b.entries))
	}
}

// TestBook_AvailableStockExcludesReserved verifies an ask is checked against
// the available stock, the held units less the earmarked ones: with 10 held and
// 8 earmarked before the market (here by the ledger directly), an ordinary ask
// of 3 is refused and one of 2 admitted, and the same for a named lot.
//
// Requirement: FR-CLR-006, FR-CLR-029
func TestBook_AvailableStockExcludesReserved(t *testing.T) {
	t.Parallel()
	m := newMkMarket(t, 2)
	m.lot(0, 3, 8)
	m.lot(0, 5, 2)
	m.lot(1, 3, 4)
	if err := m.l.ReserveGoods(m.acct[0], 1, 8); err != nil {
		t.Fatalf("ReserveGoods: %v", err)
	}
	if err := m.l.ReserveLot(m.acct[1], ledger.Key{Commodity: 1, Durability: 3}, 3); err != nil {
		t.Fatalf("ReserveLot: %v", err)
	}
	b := newBook(0, 1, 0, &m.sc)
	if r, ok := b.admit(m.l, m.facts(0), m.ask(0, 1, 5, 3)); ok || r != ReasonInsufficientStock {
		t.Errorf("ordinary ask of 3 with 2 available = %q, %v, want insufficient_stock", r, ok)
	}
	if r, ok := b.admit(m.l, m.facts(0), m.ask(0, 1, 5, 2)); !ok {
		t.Errorf("ordinary ask of 2 with 2 available = %q, want admitted", r)
	}
	if r, ok := b.admit(m.l, m.facts(1), m.askLot(1, 1, 5, 2, 3)); ok || r != ReasonInsufficientStock {
		t.Errorf("named ask of 2 with 1 available = %q, %v, want insufficient_stock", r, ok)
	}
	if r, ok := b.admit(m.l, m.facts(1), m.askLot(1, 1, 5, 1, 3)); !ok {
		t.Errorf("named ask of 1 with 1 available = %q, want admitted", r)
	}
}

// TestBook_NegativeLedgerFactPanics verifies that an order whose account holds
// a negative balance is a violated programmer invariant and not an ordinary
// rejection. A boundary account is the one account that goes below 0, so
// naming it as an agent's account stands for a ledger or a caller that got an
// account wrong.
//
// Guard: admission would otherwise refuse the order as a limit rejection
// (spec-clearing.md 3.2.2), and a bug would look like a market under pressure
// in the rejection counts.
func TestBook_NegativeLedgerFactPanics(t *testing.T) {
	t.Parallel()
	for _, side := range []Side{Bid, Ask} {
		t.Run(side.String(), func(t *testing.T) {
			t.Parallel()
			m := newMkMarket(t, 1)
			m.fund(0, 100) // the money boundary account is now at -100
			b := newBook(0, 0, 0, &m.sc)
			f := m.facts(0)
			f.Account = m.moneyB
			o := Order{Agent: m.agent(0), Commodity: 0, Side: side, Price: 5, Quantity: 1}
			defer func() {
				v := recover()
				if s, ok := v.(string); !ok || !strings.Contains(s, "violated programmer invariant") {
					t.Errorf("recover() = %v, want a violated programmer invariant message", v)
				}
			}()
			b.admit(m.l, f, o)
		})
	}
	t.Run("negative receipt tally", func(t *testing.T) {
		t.Parallel()
		m := newMkMarket(t, 1)
		m.stock(0, 0, 3)
		m.tally.set(m.acct[0], -1) // a tally only ever grows from 0
		b := newBook(0, 0, 0, &m.sc)
		defer func() {
			v := recover()
			if s, ok := v.(string); !ok || !strings.Contains(s, "violated programmer invariant") {
				t.Errorf("recover() = %v, want a violated programmer invariant message", v)
			}
		}()
		b.admit(m.l, m.facts(0), m.ask(0, 0, 5, 1))
	})
}

// TestBook_MixedAsksPanic verifies that an agent that places an ordinary ask
// and a named-lot ask of one commodity in one book is a violated programmer
// invariant, with a message that says so, because the ledger would take the
// units of one for the other.
//
// Guard: only an estate names a lot and it always does, so no input reaches
// this; without the panic a mixed pair would fail later, at a release, with a
// message about the ledger.
func TestBook_MixedAsksPanic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		first, second func(m *mkMarket) Order
	}{
		{"ordinary then named", func(m *mkMarket) Order { return m.ask(0, 1, 5, 2) }, func(m *mkMarket) Order { return m.askLot(0, 1, 5, 2, 3) }},
		{"named then ordinary", func(m *mkMarket) Order { return m.askLot(0, 1, 5, 2, 3) }, func(m *mkMarket) Order { return m.ask(0, 1, 5, 2) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMkMarket(t, 1)
			m.lot(0, 3, 4)
			m.lot(0, 5, 4)
			b := newBook(0, 1, 0, &m.sc)
			m.admitAll(b, tt.first(m))
			defer func() {
				v := recover()
				if s, ok := v.(string); !ok || !strings.Contains(s, "violated programmer invariant") {
					t.Errorf("recover() = %v, want a violated programmer invariant message", v)
				}
			}()
			b.admit(m.l, m.facts(0), tt.second(m))
		})
	}
}

// TestBook_OneMarketPerBook verifies that a book belongs to one market: an
// order of another commodity is a violated programmer invariant and panics
// with a message that says so, an order of an agent of another region is the
// rejection unknown_agent and reserves nothing, and the trades of one book
// never name the other's commodity.
//
// Requirement: FR-CLR-001, FR-CLR-003
func TestBook_OneMarketPerBook(t *testing.T) {
	t.Parallel()
	// build makes a market of its own with the test it is given, so that a
	// setup failure in a subtest stops that subtest's goroutine.
	build := func(t *testing.T) (*mkMarket, *book, *book) {
		m := newMkMarket(t, 4)
		m.fund(0, 100)
		m.fund(2, 100)
		m.stock(1, 0, 5)
		m.stock(3, 1, 5)
		return m, newBook(0, 0, 0, &m.sc), newBook(0, 1, 0, &m.sc)
	}

	t.Run("other commodity panics", func(t *testing.T) {
		t.Parallel()
		m, b0, _ := build(t)
		defer func() {
			v := recover()
			s, ok := v.(string)
			if !ok || !strings.Contains(s, "violated programmer invariant") {
				t.Errorf("recover() = %v, want a violated programmer invariant message", v)
			}
		}()
		b0.admit(m.l, m.facts(0), m.bid(0, 1, 5, 1))
	})
	t.Run("other region is unknown_agent", func(t *testing.T) {
		t.Parallel()
		m, b0, _ := build(t)
		before := m.state()
		f := m.facts(0)
		f.AgentKnown = false
		if r, ok := b0.admit(m.l, f, m.bid(0, 0, 5, 1)); ok || r != ReasonUnknownAgent {
			t.Errorf("admit = %q, %v, want unknown_agent, false", r, ok)
		}
		if !reflect.DeepEqual(before, m.state()) {
			t.Errorf("the rejection changed the ledger")
		}
	})
	m, b0, b1 := build(t)
	m.admitAll(b0, m.bid(0, 0, 8, 2), m.ask(1, 0, 6, 2))
	m.admitAll(b1, m.bid(2, 1, 8, 2), m.ask(3, 1, 6, 2))
	var trades []Trade
	var err error
	_, trades, err = clearMarket(m.l, b0, AverageClearing{}, avgRNG(1, 0, 0, 0), seedReport(7), &quiet, discardLogger(), trades)
	if err != nil {
		t.Fatalf("clearMarket b0: %v", err)
	}
	_, trades, err = clearMarket(m.l, b1, AverageClearing{}, avgRNG(1, 0, 0, 1), seedReport(7), &quiet, discardLogger(), trades)
	if err != nil {
		t.Fatalf("clearMarket b1: %v", err)
	}
	// 8 and 6 meet at 7 for 2 units in each market.
	want := []Trade{
		{Commodity: 0, Buyer: m.agent(0), Seller: m.agent(1), Price: 7, Quantity: 2},
		{Commodity: 1, Buyer: m.agent(2), Seller: m.agent(3), Price: 7, Quantity: 2},
	}
	if !slices.Equal(trades, want) {
		t.Errorf("trades = %v, want %v", trades, want)
	}
	if err := m.l.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
	if m.l.HasReservation() {
		t.Errorf("HasReservation() = true after the market step, want false")
	}
}

// TestClearMarket_ScratchReuse verifies that a warmed scratch gives exactly the
// result of a fresh one: a first market with trades and unmatched orders, then
// a second of the same size with different orders, and nothing of the first is
// present in the second.
//
// Requirement: FR-CLR-008
func TestClearMarket_ScratchReuse(t *testing.T) {
	t.Parallel()
	run := func(m *mkMarket, b *book, tick int64, prev marketReport, orders ...Order) (marketReport, []Trade, []int64) {
		t.Helper()
		m.admitAll(b, orders...)
		rep, tr, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(9, tick, 0, 0), prev, &quiet, discardLogger(), nil)
		if err != nil {
			t.Fatalf("clearMarket: %v", err)
		}
		return rep, tr, slices.Clone(b.filled())
	}
	build := func() *mkMarket {
		m := newMkMarket(t, 6)
		for i := range 3 {
			m.fund(i, 1000)
			m.stock(i+3, 0, 50)
		}
		return m
	}
	first := func(m *mkMarket) []Order {
		return []Order{m.bid(0, 0, 12, 3), m.bid(1, 0, 10, 2), m.bid(2, 0, 7, 4), m.ask(3, 0, 6, 2), m.ask(4, 0, 9, 4), m.ask(5, 0, 11, 1)}
	}
	second := func(m *mkMarket) []Order {
		return []Order{m.bid(0, 0, 20, 3), m.bid(1, 0, 15, 2), m.bid(2, 0, 3, 4), m.ask(3, 0, 8, 3), m.ask(4, 0, 12, 4), m.ask(5, 0, 30, 1)}
	}
	warm := build()
	bw := newBook(0, 0, 0, &warm.sc)
	rep1, _, _ := run(warm, bw, 0, seedReport(5), first(warm)...)
	bw.reset(1)
	repW, trW, fillW := run(warm, bw, 1, rep1, second(warm)...)

	fresh := build()
	bf := newBook(0, 0, 1, &fresh.sc)
	// The fresh ledger has not seen tick 0, so its balances differ; the
	// comparison is of the second market's own outputs: trades, filled units
	// and report, which depend on the orders and keys only.
	repF, trF, fillF := run(fresh, bf, 1, rep1, second(fresh)...)
	if !slices.Equal(trW, trF) || !slices.Equal(fillW, fillF) || repW != repF {
		t.Errorf("warm scratch gave trades %v fill %v report %+v, fresh gave %v %v %+v", trW, fillW, repW, trF, fillF, repF)
	}
	if err := warm.l.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
	if warm.l.HasReservation() {
		t.Errorf("HasReservation() = true after the market step, want false")
	}
	if len(bw.entries) != 6 {
		t.Errorf("second book holds %d orders, want 6 (none of the first's)", len(bw.entries))
	}
}

// TestClearMarket_WorkedExample verifies settlement of the book of
// spec-clearing.md 11 on a real ledger: balances and stock after the trades, 8
// returned to B1 and 28 to B3, the unmatched asks' units released, money paid
// equal to money received and units sold equal to units bought, the report,
// and the carry of the price and the age through a tick with no order.
//
// Requirement: FR-CLR-008, FR-CLR-016, FR-CLR-017, FR-CLR-020, FR-CLR-021
func TestClearMarket_WorkedExample(t *testing.T) {
	t.Parallel()
	m, orders := mkWorked(t)
	b := newBook(0, 0, 0, &m.sc)
	m.admitAll(b, orders...)
	// Reserved at admission: B1 12*3 = 36, B2 10*2 = 20, B3 7*4 = 28.
	for i, want := range []int64{36, 20, 28} {
		if _, res := m.money(i); res != want {
			t.Fatalf("B%d reserved %d, want %d", i+1, res, want)
		}
	}
	rec := &mkRecorder{}
	rep, trades, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(3, 0, 0, 0), seedReport(1), rec.observers(), discardLogger(), nil)
	if err != nil {
		t.Fatalf("clearMarket: %v", err)
	}
	// Trades: B1-A1 2 at 9; B1-A2 1 at 10 (10.5 half to even); B2-A2 2 at 10
	// (9.5 half to even). Value 18 + 10 + 20 = 48; 48/5 = 9.6 rounds to 10.
	want := []Trade{
		{Commodity: 0, Buyer: m.agent(0), Seller: m.agent(3), Price: 9, Quantity: 2},
		{Commodity: 0, Buyer: m.agent(0), Seller: m.agent(4), Price: 10, Quantity: 1},
		{Commodity: 0, Buyer: m.agent(1), Seller: m.agent(4), Price: 10, Quantity: 2},
	}
	if !slices.Equal(trades, want) {
		t.Errorf("trades = %v, want %v", trades, want)
	}
	// B1 paid 2*9 + 1*10 = 28 of 36 reserved (8 released); B2 paid 20 of 20; B3
	// traded nothing (28 released). A1 got 18, A2 got 10 + 20 = 30.
	wantMoney := []int64{72, 80, 100, 18, 30, 0}
	// B1 got 3 units, B2 2; A1 sold 2 of 5, A2 3 of 5, A3 none.
	wantUnits := []int64{3, 2, 0, 3, 2, 5}
	var paid, got int64
	for i := range 6 {
		bal, res := m.money(i)
		held, resU := m.units(i, 0)
		if bal != wantMoney[i] || res != 0 || held != wantUnits[i] || resU != 0 {
			t.Errorf("agent %d: balance %d reserved %d units %d earmarked %d, want %d 0 %d 0",
				i, bal, res, held, resU, wantMoney[i], wantUnits[i])
		}
		if i < 3 {
			paid += 100 - bal
		} else {
			got += bal
		}
	}
	if paid != 48 || got != 48 {
		t.Errorf("money paid %d received %d, want 48 and 48", paid, got)
	}
	// Volume 2+1+2 = 5. Unmatched bids: B1 filled 3 of 3, B2 2 of 2, B3 0 of 4,
	// so 4. Unmatched asks: A1 sold 2 of 2, A2 3 of 4 (1 left), A3 0 of 1 (1
	// left), so 2. Best unmatched bid is B3's 7; best unmatched ask is the
	// lowest of A2's 9 and A3's 11, so 9. Price 10 is 48/5 = 9.6 rounded.
	wantRep := marketReport{
		Price: 10, Age: 0, Volume: 5, UnmatchedBid: 4, UnmatchedAsk: 2,
		BestBid: 7, BestAsk: 9, HasBestBid: true, HasBestAsk: true,
	}
	if rep != wantRep {
		t.Errorf("report = %+v, want %+v", rep, wantRep)
	}
	// Filled per order in admission order (bids then asks): B1 2+1 = 3, B2 2,
	// B3 0; A1 2, A2 1+2 = 3, A3 0.
	if want := []int64{3, 2, 0, 2, 3, 0}; !slices.Equal(b.filled(), want) {
		t.Errorf("filled() = %v, want %v", b.filled(), want)
	}
	// The trade reason is internal: money and goods moved between accounts, so
	// its flow total per region and commodity is net zero (none non-zero).
	for _, f := range m.l.AppendFlows(nil, false) {
		if f.Reason == ledger.Trade && f.Total != 0 {
			t.Errorf("flow region %d commodity %d trade total %d, want net zero", f.Region, f.Commodity, f.Total)
		}
	}
	if err := m.l.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
	if m.l.HasReservation() {
		t.Errorf("HasReservation() = true after the market step, want false")
	}

	// A tick with no order carries the price and counts the age.
	b.reset(1)
	rep2, tr2, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(3, 1, 0, 0), rep, rec.observers(), discardLogger(), nil)
	if err != nil || len(tr2) != 0 {
		t.Fatalf("empty market = %v, %v, want no trades and no error", tr2, err)
	}
	if want := (marketReport{Price: 10, Age: 1}); rep2 != want {
		t.Errorf("report of the empty market = %+v, want %+v (price carried, age 1, no best)", rep2, want)
	}
}

// TestBook_SettlementCannotOverflow verifies that admission closes the two
// overflow cases of settlement (D34): a buyer holding MaxQuantity-3 may bid 3
// and not 4 (holding_limit), and a second bid of the same agent is
// duplicate_order and not holding_limit; a seller with a balance of MaxMoney
// less 4*MaxPrice is admitted four asks over two markets, each of which fits
// alone, and the fifth is receipt_limit, and settling all of what was admitted meets no ledger refusal.
//
// Requirement: FR-CLR-007
func TestBook_SettlementCannotOverflow(t *testing.T) {
	t.Parallel()
	t.Run("bid against the buyer's holding", func(t *testing.T) {
		t.Parallel()
		m := newMkMarket(t, 2)
		m.fund(0, 1000)
		m.stock(0, 0, MaxQuantity-3)
		m.stock(1, 0, 10)
		b := newBook(0, 0, 0, &m.sc)
		if got := m.admitAll(b, m.bid(0, 0, 5, 4)); got[0] != ReasonHoldingLimit {
			t.Fatalf("bid of 4 on MaxQuantity-3 held = %q, want holding_limit", got[0])
		}
		if _, res := m.money(0); res != 0 {
			t.Errorf("rejected bid reserved %d, want 0", res)
		}
		got := m.admitAll(b, m.bid(0, 0, 5, 3), m.bid(0, 0, 5, 4), m.ask(1, 0, 5, 3))
		if !slices.Equal(got, []Reason{"", ReasonDuplicateOrder, ""}) {
			t.Fatalf("admission = %v, want admitted, duplicate_order, admitted", got)
		}
		_, tr, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 0, 0, 0), seedReport(5), &quiet, discardLogger(), nil)
		if err != nil || len(tr) != 1 {
			t.Fatalf("clearMarket = %v, %v, want one trade", tr, err)
		}
		if held, _ := m.units(0, 0); held != MaxQuantity {
			t.Errorf("buyer holds %d, want MaxQuantity %d", held, int64(MaxQuantity))
		}
		if err := m.l.Audit(); err != nil {
			t.Errorf("Audit() = %v, want nil", err)
		}
		if m.l.HasReservation() {
			t.Errorf("HasReservation() = true after the market step, want false")
		}
	})
	t.Run("asks against the seller's balance", func(t *testing.T) {
		t.Parallel()
		// Agent 0 sells; agents 1 to 4 buy. Four asks of one unit at MaxPrice
		// have a worst-case receipt of 4*MaxPrice, exactly the headroom
		// MaxMoney - (MaxMoney - 4*MaxPrice), over a market of commodity 0 and
		// one of commodity 1 (three named lots).
		m := newMkMarket(t, 5)
		m.fund(0, MaxMoney-4*MaxPrice)
		m.stock(0, 0, 1)
		for dur := int64(1); dur <= 4; dur++ {
			m.lot(0, dur, 1)
		}
		for i := 1; i <= 4; i++ {
			m.fund(i, MaxPrice)
		}
		b0 := newBook(0, 0, 0, &m.sc)
		b1 := newBook(0, 1, 0, &m.sc)
		got := m.admitAll(b0, m.ask(0, 0, MaxPrice, 1), m.bid(1, 0, MaxPrice, 1))
		got = append(got, m.admitAll(b1,
			m.askLot(0, 1, MaxPrice, 1, 1), m.askLot(0, 1, MaxPrice, 1, 2), m.askLot(0, 1, MaxPrice, 1, 3),
			m.askLot(0, 1, MaxPrice, 1, 4),
			m.bid(2, 1, MaxPrice, 1), m.bid(3, 1, MaxPrice, 1), m.bid(4, 1, MaxPrice, 1))...)
		want := []Reason{"", "", "", "", "", ReasonReceiptLimit, "", "", ""}
		if !slices.Equal(got, want) {
			t.Fatalf("admission = %v, want %v (the fourth ask of the third lot sum overflows)", got, want)
		}
		for i, b := range []*book{b0, b1} {
			_, _, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 0, 0, int64(i)), seedReport(5), &quiet, discardLogger(), nil)
			if err != nil {
				t.Fatalf("clearMarket(%d): %v", i, err)
			}
		}
		if bal, _ := m.money(0); bal != MaxMoney {
			t.Errorf("seller balance = %d, want exactly MaxMoney", bal)
		}
		if err := m.l.Audit(); err != nil {
			t.Errorf("Audit() = %v, want nil", err)
		}
		if m.l.HasReservation() {
			t.Errorf("HasReservation() = true after the market step, want false")
		}
	})
}

// TestBook_BidChecksAvailableCash verifies a bid is checked against the
// balance less the cash already reserved: with 100 held and 60 reserved
// directly in the ledger, a bid costing 50 is insufficient_funds although the
// balance alone covers it, and one costing 40 is admitted.
//
// Requirement: FR-CLR-005
func TestBook_BidChecksAvailableCash(t *testing.T) {
	t.Parallel()
	m := newMkMarket(t, 1)
	m.fund(0, 100)
	if err := m.l.ReserveMoney(m.acct[0], 60); err != nil {
		t.Fatalf("ReserveMoney: %v", err)
	}
	b := newBook(0, 0, 0, &m.sc)
	if r, ok := b.admit(m.l, m.facts(0), m.bid(0, 0, 10, 5)); ok || r != ReasonInsufficientFunds {
		t.Errorf("bid costing 50 with 100-60 = 40 available = %q, %v, want insufficient_funds", r, ok)
	}
	if r, ok := b.admit(m.l, m.facts(0), m.bid(0, 0, 10, 4)); !ok {
		t.Errorf("bid costing 40 with 40 available = %q, want admitted", r)
	}
}

// TestClearMarket_PriceAgeCarries verifies the previous report's age feeds the
// next: a market with no trade carries the price and reports prev.Age+1, and
// one that trades reports age 0 whatever the previous age.
//
// Requirement: FR-CLR-020
func TestClearMarket_PriceAgeCarries(t *testing.T) {
	t.Parallel()
	prev := marketReport{Price: 10, Age: 3}

	m := newMkMarket(t, 1)
	b := newBook(0, 0, 0, &m.sc)
	rep, _, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(3, 0, 0, 0), prev, &quiet, discardLogger(), nil)
	if err != nil {
		t.Fatalf("clearMarket (empty): %v", err)
	}
	if want := (marketReport{Price: 10, Age: 4}); rep != want {
		t.Errorf("empty market report = %+v, want %+v (age 3+1)", rep, want)
	}

	m, orders := mkWorked(t)
	b = newBook(0, 0, 0, &m.sc)
	m.admitAll(b, orders...)
	rep, _, err = clearMarket(m.l, b, AverageClearing{}, avgRNG(3, 0, 0, 0), prev, &quiet, discardLogger(), nil)
	if err != nil {
		t.Fatalf("clearMarket (trading): %v", err)
	}
	if rep.Age != 0 || rep.Volume != 5 {
		t.Errorf("trading market age %d volume %d, want 0 and 5", rep.Age, rep.Volume)
	}
}
