package economysim

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
)

// The palettes a byte indexes, so that a fuzz input reaches the edges of the
// bounds (D34) and the values of a small book with few bytes.
var (
	moneyPalette = []int64{0, 1, 100, 1000, 1_000_000_000, MaxMoney - 4*MaxPrice, MaxMoney - 1, MaxMoney}
	unitPalette  = []int64{0, 1, 2, 5, 10, MaxQuantity - 3, MaxQuantity}
	lotPalette   = []int64{0, 0, 1, 2, 5}
	pricePalette = []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 20, MaxPrice, MaxPrice + 1}
	qtyPalette   = []int64{0, 1, 2, 3, 4, 5, 10, MaxQuantity - 3, MaxQuantity, MaxQuantity + 1}
)

type agentSpec struct {
	money, units int64
	lots         [5]int64 // units of commodity 1 at durability 1 to 5
}

type orderSpec struct {
	agent, commodity int
	side             Side
	price, qty       int64
	lotDur           int64 // an ask of commodity 1 naming that durability; 0 none
	unknown          bool  // the world says the agent is not in the region
}

type tradeSpec struct {
	buyer, seller, commodity int
	price, qty               int64
	lotDur                   int64
}

type bookSpec struct {
	agents []agentSpec
	orders []orderSpec
	fault  bool
	trades []tradeSpec
}

// pick returns palette[b mod len], and index the byte that selects v.
func pick(b byte, p []int64) int64 { return p[int(b)%len(p)] }

func index(v int64, p []int64) byte {
	i := slices.Index(p, v)
	if i < 0 {
		panic(fmt.Sprintf("value %d is not in the palette", v))
	}
	return byte(i)
}

// decodeSpec reads a book from a byte source; an exhausted source reads 0.
func decodeSpec(next func() byte) bookSpec {
	var s bookSpec
	n := 2 + int(next())%5
	for range n {
		a := agentSpec{money: pick(next(), moneyPalette), units: pick(next(), unitPalette)}
		for d := range a.lots {
			a.lots[d] = pick(next(), lotPalette)
		}
		s.agents = append(s.agents, a)
	}
	for range int(next()) % 17 {
		o := orderSpec{agent: int(next()) % n, commodity: int(next()) % 2, side: Side(next() % 2)}
		o.price, o.qty = pick(next(), pricePalette), pick(next(), qtyPalette)
		if d := int64(next() % 6); o.side == Ask && o.commodity == 1 {
			o.lotDur = d
		}
		o.unknown = next()%16 == 0
		s.orders = append(s.orders, o)
	}
	if s.fault = next()%2 == 1; s.fault {
		for range int(next()) % 5 {
			s.trades = append(s.trades, tradeSpec{
				buyer: int(next()) % (n + 1), seller: int(next()) % (n + 1), commodity: int(next()) % 3,
				price: pick(next(), pricePalette), qty: pick(next(), qtyPalette), lotDur: int64(next() % 6),
			})
		}
	}
	return s
}

// encode is the inverse of decodeSpec for a spec whose values are in the
// palettes: the seeds of FuzzBook are encoded, not hand-written bytes.
func (s bookSpec) encode() []byte {
	out := []byte{byte(len(s.agents) - 2)}
	for _, a := range s.agents {
		out = append(out, index(a.money, moneyPalette), index(a.units, unitPalette))
		for _, l := range a.lots {
			out = append(out, index(l, lotPalette))
		}
	}
	out = append(out, byte(len(s.orders)))
	for _, o := range s.orders {
		unk := byte(1)
		if o.unknown {
			unk = 0
		}
		out = append(out, byte(o.agent), byte(o.commodity), byte(o.side), index(o.price, pricePalette),
			index(o.qty, qtyPalette), byte(o.lotDur), unk)
	}
	if !s.fault {
		return append(out, 0)
	}
	out = append(out, 1, byte(len(s.trades)))
	for _, tr := range s.trades {
		out = append(out, byte(tr.buyer), byte(tr.seller), byte(tr.commodity),
			index(tr.price, pricePalette), index(tr.qty, qtyPalette), byte(tr.lotDur))
	}
	return out
}

func byteSource(data []byte) func() byte {
	return func() byte {
		if len(data) == 0 {
			return 0
		}
		b := data[0]
		data = data[1:]
		return b
	}
}

// runSpec builds the book of a spec on a real ledger, clears it market by
// market and returns the properties of the step that failed, one text each, for
// the test to report with the input that made them (testing.md 3.6).
func runSpec(t testing.TB, s bookSpec, seed uint64) (fails []string) {
	t.Helper()
	fail := func(format string, args ...any) {
		fails = append(fails, fmt.Sprintf(format, args...))
	}
	m := newMkMarket(t, len(s.agents))
	for i, a := range s.agents {
		if a.money > 0 {
			m.fund(i, a.money)
		}
		if a.units > 0 {
			m.stock(i, 0, a.units)
		}
		for d, q := range a.lots {
			if q > 0 {
				m.lot(i, int64(d+1), q)
			}
		}
	}
	totalMoney, totalUnits := m.totals()
	for c := range 2 {
		before := m.state()
		m.tally.reset()
		b := newBook(0, CommodityID(c), 0, &m.sc)
		admitted := map[Side]map[AgentID]Order{Bid: {}, Ask: {}}
		type lotAsk struct {
			agent AgentID
			lot   LotKey
		}
		asks := map[lotAsk]Order{}
		asksOf := map[AgentID]bool{}
		for _, os := range s.orders {
			if os.commodity != c {
				continue
			}
			o := Order{Agent: m.agent(os.agent), Commodity: CommodityID(c), Side: os.side, Price: Price(os.price), Quantity: Quantity(os.qty)}
			if os.lotDur > 0 {
				o.Lot = LotKey{Named: true, Durability: os.lotDur}
			}
			// Only an estate names a lot and it always does: an agent never
			// mixes the two in one book (admit panics on it, by design).
			mixed := false
			for _, e := range b.entries {
				mixed = mixed || e.agent == o.Agent && e.side == Ask && o.Side == Ask && e.lot.Named != o.Lot.Named
			}
			if mixed {
				continue
			}
			f := m.facts(os.agent)
			f.AgentKnown = !os.unknown
			if r, ok := b.admit(m.l, f, o); ok {
				if o.Side == Bid {
					o.Lot = LotKey{}
					admitted[Bid][o.Agent] = o
				} else {
					asks[lotAsk{o.Agent, o.Lot}] = o
					asksOf[o.Agent] = true
				}
			} else if r == "" {
				fail("rejected with no reason")
			}
		}
		var cl Clearing = AverageClearing{}
		if s.fault {
			var list []Trade
			for _, tr := range s.trades {
				x := Trade{Commodity: CommodityID(tr.commodity), Buyer: m.agent(tr.buyer), Seller: m.agent(tr.seller),
					Price: Price(tr.price), Quantity: Quantity(tr.qty)}
				if tr.lotDur > 0 {
					x.Lot = LotKey{Named: true, Durability: tr.lotDur}
				}
				list = append(list, x)
			}
			cl = fakeClearing{list}
		}
		_, trades, err := clearMarket(m.l, b, cl, avgRNG(seed, 0, 0, int64(c)), seedReport(5), &quiet, discardLogger(), nil)
		if err != nil {
			if !errors.Is(err, ErrInvalidTrade) {
				fail("market %d: error %v, want ErrInvalidTrade", c, err)
			}
			// AverageClearing may match an agent with itself only when the
			// agent holds a bid and an ask that cross, which only a treasury
			// places and then never crossed (spec-clearing.md 3.1.2).
			both := false
			for _, e := range b.entries {
				if e.side == Bid {
					_, has := asksOf[e.agent]
					both = both || has
				}
			}
			if !s.fault && !both {
				fail("market %d: AverageClearing gave an invalid list: %v", c, err)
			}
			if after := m.state(); !reflect.DeepEqual(before, after) {
				fail("market %d: a rejected list left the ledger changed", c)
			}
			continue
		}
		if err := m.l.Audit(); err != nil {
			fail("market %d: Audit() = %v", c, err)
		}
		if m.l.HasReservation() {
			fail("market %d: a reservation is open after the step", c)
		}
		var units int64
		for _, tr := range trades {
			units += int64(tr.Quantity)
			bo, ok1 := admitted[Bid][tr.Buyer]
			ao, ok2 := asks[lotAsk{tr.Seller, tr.Lot}]
			if !ok1 || !ok2 || tr.Price < ao.Price || tr.Price > bo.Price || tr.Buyer == tr.Seller || tr.Commodity != CommodityID(c) {
				fail("market %d: trade %+v breaks the rules of 8.2 (bid %+v ask %+v)", c, tr, bo, ao)
			}
		}
		var bidFill, askFill int64
		for i, f := range b.filled() {
			if f < 0 || f > int64(b.entries[i].qty) {
				fail("market %d: order %d filled %d of %d", c, i, f, b.entries[i].qty)
			}
			if b.entries[i].side == Bid {
				bidFill += f
			} else {
				askFill += f
			}
		}
		if bidFill != units || askFill != units {
			fail("market %d: units bought %d sold %d, trades %d", c, bidFill, askFill, units)
		}
		if mo, un := m.totals(); mo != totalMoney || !slices.Equal(un, totalUnits) {
			fail("market %d: money %d units %v, want %d %v conserved", c, mo, un, totalMoney, totalUnits)
		}
	}
	for i := range s.agents {
		if bal, res := m.money(i); bal < 0 || res != 0 {
			fail("agent %d balance %d reserved %d", i, bal, res)
		}
	}
	return fails
}

// totals sums the money and the units of each commodity over the accounts of
// the agents (a local sum, with no Audit).
func (m *mkMarket) totals() (money int64, units []int64) {
	units = make([]int64, 2)
	for i := range m.acct {
		bal, _ := m.money(i)
		money += bal
		for c := range units {
			held, _ := m.units(i, c)
			units[c] += held
		}
	}
	return money, units
}

// TestBook_InvariantsOverSeeds verifies, over a fixed range of seeds of
// random books, that after every market step no reservation is open, no
// balance or stock is negative, every trade price lies between the matched
// limits, goods and money are conserved by exact equality, and the ledger
// audit passes, with AverageClearing and with an arbitrary list from a fake.
//
// Requirement: FR-CLR-008, FR-CLR-014, FR-CLR-017, FR-CLR-024
func TestBook_InvariantsOverSeeds(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	for seed := range uint64(300) {
		r := rand.New(rand.NewPCG(seed, 0xB00C))
		spec := decodeSpec(func() byte { return byte(r.Uint32()) })
		for _, f := range runSpec(t, spec, seed) {
			t.Errorf("seed %d: %s", seed, f)
		}
	}
}

// FuzzBook decodes bytes to a book on a real ledger (balances and stocks
// near MaxMoney and MaxQuantity included) and to an arbitrary list of trades
// that a fake Clearing returns. A list the step accepts satisfies every
// property of spec-clearing.md 8.2 and conserves goods and money; one it
// rejects returns ErrInvalidTrade and leaves the ledger as it was before the
// step; nothing panics.
//
// Requirement: FR-CLR-007, FR-CLR-024
func FuzzBook(f *testing.F) {
	book := bookSpec{
		agents: []agentSpec{{money: 1000}, {money: 1000}, {units: 10}, {units: 10}, {money: 1000, units: 10}},
		orders: []orderSpec{
			{agent: 0, side: Bid, price: 10, qty: 3}, {agent: 1, side: Bid, price: 8, qty: 2},
			{agent: 4, side: Bid, price: 3, qty: 1}, {agent: 2, side: Ask, price: 5, qty: 2},
			{agent: 3, side: Ask, price: 6, qty: 2}, {agent: 4, side: Ask, price: 3, qty: 1},
		},
	}
	f.Add(book.encode())
	// The six failures of spec-clearing.md 8.2, each after a valid trade.
	good := tradeSpec{buyer: 0, seller: 2, price: 7, qty: 1}
	breaks := []tradeSpec{
		{buyer: 0, seller: 2, commodity: 1, price: 7, qty: 1},
		{buyer: 2, seller: 3, price: 7, qty: 1},
		{buyer: 0, seller: 2, price: 7, qty: 0},
		{buyer: 0, seller: 2, price: 11, qty: 1},
		{buyer: 4, seller: 4, price: 3, qty: 1},
		{buyer: 0, seller: 2, price: 7, qty: 3},
	}
	for _, br := range breaks {
		s := book
		s.fault, s.trades = true, []tradeSpec{good, br}
		f.Add(s.encode())
	}
	// Near the bounds of D34: a buyer one unit under MaxQuantity bids 3, a
	// seller one price under MaxMoney asks.
	edge := bookSpec{
		agents: []agentSpec{{money: 1000, units: MaxQuantity - 3}, {units: 10}, {money: MaxMoney - 4*MaxPrice, units: 1}, {money: MaxPrice}},
		orders: []orderSpec{
			{agent: 0, side: Bid, price: 5, qty: 3}, {agent: 1, side: Ask, price: 5, qty: 3},
			{agent: 2, side: Ask, price: MaxPrice, qty: 1}, {agent: 3, side: Bid, price: MaxPrice, qty: 1},
		},
	}
	f.Add(edge.encode())
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, f := range runSpec(t, decodeSpec(byteSource(data)), 1) {
			t.Errorf("input %x: %s", data, f)
		}
	})
}
