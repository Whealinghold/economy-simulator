package economysim

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The tests of the tick of the external module (plan 2-16 Phase 6): the worked
// tick of spec-external-sector.md 18.1, the quote as a limit, the counterparties
// that never meet, the walk's draws, the purity of the orders, the closed edge,
// the window and the caps of 18.3, and the identities.

// extScript is a Strategy that asks no job and places the intents that plan gives
// each agent, a fake at the Strategy extension point (testing.md 6.1).
type extScript struct {
	plan map[AgentID][]Intent
}

func (*extScript) Work(*AgentView, *rand.Rand) JobRequest { return JobRequest{NoJob: true} }

func (s *extScript) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	p.Intents = append(p.Intents, s.plan[v.Agent]...)
}

func (*extScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = (*extScript)(nil)

// extConfig returns the world of example A of spec-external-sector.md 18.1: the
// regions coast (the outside node, estate agent 0) and lowland (estate 1), edges
// both ways of 30, the counterparties agent 2 (the buyer of lowland) and agent 3
// (the seller), and seven households of the role trader from agent 4, whose strategy
// is the script. The node sells salt (Base 40, ask 5 % above, 30 a tick) and buys
// grain (Base 20, bid 10 % below, 50 a tick). The upkeep is bread, which nobody
// trades, so the one tick the tests run has no other order.
func extConfig() Config {
	c := shellConfig()
	bread := DefaultCommodity()
	bread.ID, bread.ReferencePrice, bread.Raw = "bread", 1_000, true
	grain, salt := DefaultCommodity(), DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 20, true
	salt.ID, salt.ReferencePrice, salt.Raw = "salt", 40, true
	c.Commodities = []Commodity{bread, grain, salt}
	c.World.Upkeep = []Item{{Commodity: "bread", Quantity: 1}}
	c.World.BeliefStep = 500 // the least price a belief moves at is 10 000 / 500 = 20, the cheapest good of 18.1
	// A recipe that names the two goods, so that neither is an orphan commodity; no
	// role runs it.
	c.Recipes = []Recipe{{ID: "make", Duration: 1, TraineeYield: 5_000,
		Outputs: []Output{{Commodity: "grain", Quantity: 1, CostShare: 5_000}, {Commodity: "salt", Quantity: 1, CostShare: 5_000}}}}
	c.Roles[0].ID, c.Roles[0].Strategy, c.Roles[0].Trades = "trader", "script", []string{"grain", "salt"}
	coast, lowland := DefaultRegion(), DefaultRegion()
	coast.ID, lowland.ID = "coast", "lowland"
	c.Regions = []Region{coast, lowland}
	c.Groups = []Group{{Region: "lowland", Role: "trader", Households: 7}}
	c.Edges = []Edge{{From: "coast", To: "lowland", Distance: 30}, {From: "lowland", To: "coast", Distance: 30}}
	saltLine := DefaultOutsideLine("salt", 40)
	saltLine.Sell = OutsideSide{MarginBP: 500, Capacity: 30}
	grainLine := DefaultOutsideLine("grain", 20)
	grainLine.Buy = OutsideSide{MarginBP: 1_000, Capacity: 50}
	c.Modules.External = &External{Nodes: []OutsideNode{{Region: "coast", Lines: []OutsideLine{saltLine, grainLine}}}}
	return c
}

// extWorld builds the world of c with the script, a recording observer and the audit
// on, ends the construction bucket, and returns the script and the log.
func extWorld(t *testing.T, c Config, opts ...Option) (*World, *extScript, *evLog) {
	t.Helper()
	s := &extScript{plan: map[AgentID][]Intent{}}
	rec := &evLog{}
	opts = append([]Option{WithObserver(rec), WithAudit(), WithStrategy("script", s)}, opts...)
	w := newMill(t, c, opts...)
	if err := w.ledger.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction() = %v, want nil", err)
	}
	return w, s, rec
}

// fund gives an agent money from the initial boundary, as a fixture.
func fund(t *testing.T, w *World, id AgentID, amount int64) {
	t.Helper()
	src, ok := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		t.Fatal("no initial boundary for money")
	}
	if err := w.ledger.Transfer(src, w.agents[id].account, amount, ledger.Initial); err != nil {
		t.Fatalf("fund(%d to agent %d) = %v, want nil", amount, id, err)
	}
}

// boundaryOf returns the balance of the boundary account of a reason and a thing.
func boundaryOf(t *testing.T, w *World, r ledger.Reason, thing int) int64 {
	t.Helper()
	id, ok := w.ledger.Boundary(r, thing)
	if !ok {
		t.Fatalf("no boundary account for reason %d and thing %d", r, thing)
	}
	v, err := w.ledger.Account(id)
	if err != nil {
		t.Fatalf("Account(%d) = %v, want nil", id, err)
	}
	return v.Balance
}

// TestExternal_WorkedTick verifies example A of spec-external-sector.md 18.1 end to
// end. Grain, an export: the buyer is topped up with 18 * 50 = 900; the realm asks
// 15 (20 units), 17 (15), 19 (10) and 21 (30); the bid of 18 meets 20 units at the
// midpoint 16.5 -> 16 (320) and 15 at 17.5 -> 18 (270): 35 units for 590, the sweep
// returns 900 - 590 = 310 and takes 35 grain. Salt, an import: the seller receives 30
// units; the realm bids 50 (10), 45 (10) and 41 (10) against the ask 42: 10 at 46
// (460) and 10 at 43.5 -> 44 (440), 20 units for 900; the sweep takes 10 unsold salt
// and 900. The six flows are 900, 310, 900, 30, 10 and 35, the balance of trade is
// 590 - 900 = -310 and both counterparty accounts are empty.
//
// Requirement: FR-EXT-013, FR-EXT-014, FR-EXT-015, FR-EXT-019, FR-EXT-039, FR-EXT-048
func TestExternal_WorkedTick(t *testing.T) {
	t.Parallel()
	w, s, rec := extWorld(t, extConfig())
	grain, salt := commodityOf(t, w, "grain"), commodityOf(t, w, "salt")
	sellers := []struct {
		limit Price
		units Quantity
	}{{15, 20}, {17, 15}, {19, 10}, {21, 30}}
	for i, o := range sellers {
		id := w.firstHousehold + i
		give(t, w, AgentID(id), grain, 100)
		s.plan[AgentID(id)] = []Intent{{Commodity: grain, Side: Ask, Quantity: o.units, Limit: o.limit}}
	}
	bidders := []struct {
		limit Price
		units Quantity
	}{{50, 10}, {45, 10}, {41, 10}}
	for i, o := range bidders {
		id := AgentID(w.firstHousehold + len(sellers) + i)
		fund(t, w, id, 10_000)
		s.plan[id] = []Intent{{Commodity: salt, Side: Bid, Quantity: o.units, Limit: o.limit}}
	}
	if len(w.agents) != 4+7 {
		t.Fatalf("the world has %d agent slots, want 11 (2 estates, 2 counterparties, 7 households)", len(w.agents))
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	flows := []struct {
		name  string
		got   int64
		want  int64
		thing int
		r     ledger.Reason
	}{
		{"outside_budget (faucet, balance is minus the flow)", boundaryOf(t, w, ledger.OutsideBudget, ledger.Money), -900, ledger.Money, ledger.OutsideBudget},
		{"outside_unspent", boundaryOf(t, w, ledger.OutsideUnspent, ledger.Money), 310, ledger.Money, ledger.OutsideUnspent},
		{"outside_receipts", boundaryOf(t, w, ledger.OutsideReceipts, ledger.Money), 900, ledger.Money, ledger.OutsideReceipts},
		{"outside_offer salt", boundaryOf(t, w, ledger.OutsideOffer, int(salt)), -30, int(salt), ledger.OutsideOffer},
		{"outside_unsold salt", boundaryOf(t, w, ledger.OutsideUnsold, int(salt)), 10, int(salt), ledger.OutsideUnsold},
		{"outside_purchases grain", boundaryOf(t, w, ledger.OutsidePurchases, int(grain)), 35, int(grain), ledger.OutsidePurchases},
	}
	for _, f := range flows {
		if f.got != f.want {
			t.Errorf("boundary balance of %s = %d, want %d", f.name, f.got, f.want)
		}
	}
	if net := -boundaryOf(t, w, ledger.OutsideBudget, ledger.Money) - boundaryOf(t, w, ledger.OutsideUnspent, ledger.Money) - boundaryOf(t, w, ledger.OutsideReceipts, ledger.Money); net != -310 {
		t.Errorf("balance of trade (budget - unspent - receipts) = %d, want -310", net)
	}
	for i, cp := range w.external.cps {
		if v, _ := w.ledger.Account(cp.account); v.Balance != 0 {
			t.Errorf("counterparty %d holds %d, want an empty account", i, v.Balance)
		}
	}
	traded := of[OutsideTraded](rec)
	want := []OutsideTraded{
		// By commodity, the buy side before the sell side of a commodity: grain (1) then salt (2).
		{Tick: 0, Node: 0, Region: 1, Commodity: grain, Side: Bid, Mid: 20, Price: 18, Wanted: 50, Offered: 50, Traded: 35, Value: 590, ListValue: 630},
		{Tick: 0, Node: 0, Region: 1, Commodity: salt, Side: Ask, Mid: 40, Price: 42, Wanted: 30, Offered: 30, Traded: 20, Value: 900, ListValue: 840},
	}
	if !slices.Equal(traded, want) {
		t.Errorf("OutsideTraded events = %+v, want %+v", traded, want)
	}
	if n := len(of[OutsideEnded](rec)); n != 0 {
		t.Errorf("OutsideEnded events = %d, want none: no cap is set", n)
	}
	for _, e := range of[TradeExecuted](rec) {
		if e.Commodity == grain && e.Price != 16 && e.Price != 18 || e.Commodity == salt && e.Price != 46 && e.Price != 44 {
			t.Errorf("trade %+v is at a price other than the midpoints of 18.1 (16, 18, 46, 44)", e)
		}
	}
}

// TestExternal_QuoteIsALimit verifies that the quote of a counterparty is a limit and
// not a peg (spec-external-sector.md 6.7, 18.1): a realm bid of 50 against the outside
// ask of 42 trades at the midpoint 46, a realm ask of 15 against the outside bid of 18
// at 16.5 rounded half to even, 16.
//
// Requirement: FR-EXT-049
func TestExternal_QuoteIsALimit(t *testing.T) {
	t.Parallel()
	w, s, rec := extWorld(t, extConfig())
	grain, salt := commodityOf(t, w, "grain"), commodityOf(t, w, "salt")
	seller, bidder := AgentID(w.firstHousehold), AgentID(w.firstHousehold+1)
	give(t, w, seller, grain, 20)
	fund(t, w, bidder, 1_000)
	s.plan[seller] = []Intent{{Commodity: grain, Side: Ask, Quantity: 20, Limit: 15}}
	s.plan[bidder] = []Intent{{Commodity: salt, Side: Bid, Quantity: 10, Limit: 50}}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	got := map[CommodityID]Price{}
	for _, e := range of[TradeExecuted](rec) {
		got[e.Commodity] = e.Price
	}
	if got[salt] != 46 || got[grain] != 16 {
		t.Errorf("trade prices salt %d, grain %d, want 46 ((42+50)/2) and 16 ((15+18)/2 = 16.5 to even)", got[salt], got[grain])
	}
}

// TestExternal_CounterpartiesNeverMeet verifies that the buyer and the seller of one
// gate never match each other: with margins summing to the minimum spread of 1 basis
// point the ask is 20 and the bid 19, both orders are placed and nothing trades. And
// that two nodes on one gate place two orders (spec-external-sector.md 3.2.3, 6.6.1).
//
// Requirement: FR-EXT-009, FR-EXT-013, FR-EXT-048
func TestExternal_CounterpartiesNeverMeet(t *testing.T) {
	t.Parallel()
	t.Run("minimum spread", func(t *testing.T) {
		t.Parallel()
		c := extConfig()
		l := DefaultOutsideLine("grain", 20)
		l.Sell, l.Buy = OutsideSide{MarginBP: 0, Capacity: 5}, OutsideSide{MarginBP: 1, Capacity: 5}
		c.Modules.External.Nodes[0].Lines = []OutsideLine{l}
		w, _, rec := extWorld(t, c)
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
		placed := of[OrderPlaced](rec)
		if len(placed) != 2 || placed[0].Side != Bid || placed[0].Limit != 19 || placed[1].Side != Ask || placed[1].Limit != 20 {
			t.Errorf("OrderPlaced = %+v, want the bid of 19 then the ask of 20", placed)
		}
		if n := len(of[TradeExecuted](rec)); n != 0 {
			t.Errorf("TradeExecuted = %d, want none: bid 19 is below ask 20", n)
		}
	})
	t.Run("two nodes on one gate", func(t *testing.T) {
		t.Parallel()
		c := extConfig()
		delta := DefaultRegion()
		delta.ID = "delta"
		c.Regions = []Region{c.Regions[0], delta, c.Regions[1]}
		c.Edges = append(c.Edges, Edge{From: "delta", To: "lowland", Distance: 10}, Edge{From: "lowland", To: "delta", Distance: 10})
		l := DefaultOutsideLine("salt", 40)
		l.Sell = OutsideSide{MarginBP: 500, Capacity: 5}
		c.Modules.External.Nodes = []OutsideNode{c.Modules.External.Nodes[0], {Region: "delta", Lines: []OutsideLine{l}}}
		w, _, rec := extWorld(t, c)
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
		asks := 0
		for _, e := range of[OrderPlaced](rec) {
			if e.Side == Ask && e.Commodity == commodityOf(t, w, "salt") {
				asks++
			}
		}
		if asks != 2 {
			t.Errorf("asks for salt placed in lowland = %d, want 2: one for each node", asks)
		}
	})
}

// TestExternal_WalkDraws verifies draw D2 (spec-external-sector.md 5.1.1, 6.4, D523,
// D524): the drift of a line follows one word of the keyed stream of (tick, node,
// commodity) a tick, recomputed here by an independent stream; a walking line added
// before it changes nothing of its drift; and the creation draws are the same with
// the module on and off.
//
// Requirement: FR-EXT-021, FR-EXT-044
// Guard: D523, the site is keyed by node and commodity so that a line added before
// another does not shift its draws.
func TestExternal_WalkDraws(t *testing.T) {
	t.Parallel()
	walking := func(extra bool) Config {
		c := extConfig()
		g := &c.Modules.External.Nodes[0].Lines[0]
		if g.Commodity != "grain" { // the lines are written salt, grain: grain is the second
			g = &c.Modules.External.Nodes[0].Lines[1]
		}
		g.WalkStepBP, g.WalkRevertBP, g.WalkBoundBP = 100, 500, 2_000
		if extra {
			bread := DefaultOutsideLine("bread", 1_000)
			bread.WalkStepBP, bread.WalkBoundBP = 300, 3_000
			c.Modules.External.Nodes[0].Lines = append(c.Modules.External.Nodes[0].Lines, bread)
		}
		return c
	}
	grainDrift := func(w *World) int64 {
		for _, l := range w.external.lines {
			if l.cfg.Commodity == "grain" {
				return l.drift
			}
		}
		t.Fatal("no grain line")
		return 0
	}
	plain, _, _ := extWorld(t, walking(false), WithSeed(7))
	extra, _, _ := extWorld(t, walking(true), WithSeed(7))
	mirror := newStream(plain.state.seed)
	grain := commodityOf(t, plain, "grain")
	var want int64
	for tick := range int64(40) {
		mirror.keyedPair(siteOutsideWalk, tick, 0, int64(grain))
		want = outsideWalk(want, mirror.drawRange(-100, 100))
		for _, w := range []*World{plain, extra} {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() at tick %d = %v, want nil", tick, err)
			}
			if got := grainDrift(w); got != want {
				t.Fatalf("drift of grain after tick %d = %d, want %d from the keyed stream of (tick, node, commodity)", tick, got, want)
			}
		}
	}
	if want == 0 {
		t.Errorf("the drift stayed at 0 over 40 ticks, want a walk")
	}
	off := extConfig()
	off.Modules.External = nil
	offWorld, _, _ := extWorld(t, off, WithSeed(7))
	onWorld, _, _ := extWorld(t, extConfig(), WithSeed(7))
	for i := range 7 {
		a, b := onWorld.agents[onWorld.firstHousehold+i], offWorld.agents[offWorld.firstHousehold+i]
		if a.target != b.target {
			t.Errorf("household %d has target ratio %d with the module on and %d with it off, want equal: no draw at creation", i, a.target, b.target)
		}
	}
}

// outsideWalk is one step of the walk of the test line: revert 500, bound 2 000.
func outsideWalk(drift, d int64) int64 {
	drift = drift - drift*500/10_000 + d
	return min(max(drift, -2_000), 2_000)
}

// TestExternal_PureOfTheRealm verifies that the order of a counterparty does not
// depend on the realm (spec-external-sector.md 5.2, 11.3, FR-EXT-023): a world whose
// households hold much more money and goods places the same orders at the same limits
// and quantities.
//
// Requirement: FR-EXT-023
func TestExternal_PureOfTheRealm(t *testing.T) {
	t.Parallel()
	a, _, recA := extWorld(t, extConfig())
	b, _, recB := extWorld(t, extConfig())
	for i := range 7 {
		id := AgentID(b.firstHousehold + i)
		fund(t, b, id, 50_000)
		give(t, b, id, commodityOf(t, b, "salt"), 40)
		give(t, b, id, commodityOf(t, b, "grain"), 40)
	}
	for _, w := range []*World{a, b} {
		for range 3 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
		}
	}
	oa, ob := of[OrderPlaced](recA), of[OrderPlaced](recB)
	if len(oa) == 0 || !slices.Equal(oa, ob) {
		t.Errorf("OrderPlaced differ between a poor and a rich realm:\n poor %+v\n rich %+v", oa, ob)
	}
}

// TestExternal_ClosedEdge verifies that a closed edge stops its side at once and
// counts the wanted units as CutClosed, with no order and no top-up, and that the
// length factor of an open edge changes nothing (spec-external-sector.md 7.5,
// FR-EXT-032).
//
// Requirement: FR-EXT-032
func TestExternal_ClosedEdge(t *testing.T) {
	t.Parallel()
	w, _, rec := extWorld(t, extConfig())
	// coast to lowland carries the sell side (salt); lowland to coast the buy side.
	if err := w.setEdgeStatuses([]EdgeStatus{{From: 0, To: 1, Closed: true, FactorBP: 10_000}, {From: 1, To: 0, Closed: false, FactorBP: 5_000}}, CauseConsumer); err != nil {
		t.Fatalf("setEdgeStatuses() = %v, want nil", err)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	salt, grain := commodityOf(t, w, "salt"), commodityOf(t, w, "grain")
	for _, e := range of[OutsideTraded](rec) {
		switch e.Commodity {
		case salt:
			if e.Wanted != 30 || e.CutClosed != 30 || e.Offered != 0 || e.Price != 0 {
				t.Errorf("salt event = %+v, want 30 wanted all cut by the closed edge, nothing offered", e)
			}
		case grain:
			if e.Offered != 50 || e.CutClosed != 0 || e.Price != 18 {
				t.Errorf("grain event = %+v, want 50 offered at 18: the length factor of an open edge changes nothing", e)
			}
		}
	}
	for _, e := range of[OrderPlaced](rec) {
		if e.Commodity == salt {
			t.Errorf("OrderPlaced for salt = %+v, want none: the edge is closed", e)
		}
	}
	if boundaryOf(t, w, ledger.OutsideOffer, int(salt)) != 0 {
		t.Errorf("outside_offer for salt = %d, want 0: no top-up through a closed edge", boundaryOf(t, w, ledger.OutsideOffer, int(salt)))
	}
}

// capConfig returns the world of example B of spec-external-sector.md 18.3: the node
// inner_system and the gate ring, edges both ways, the lines ore (Base 120, a buy
// side with bid 20 % below, 200 a tick, TotalUnits 1 500) and reagent (Base 300, trend
// -200, a sell side with ask 8 % above, 10 a tick, TotalValue 12 000), and seven
// households of the role trader, whose script is set by the test.
func capConfig() Config {
	c := extConfig()
	ore, reagent := DefaultCommodity(), DefaultCommodity()
	ore.ID, ore.ReferencePrice, ore.Raw = "ore", 120, true
	reagent.ID, reagent.ReferencePrice, reagent.Raw = "reagent", 300, true
	c.Commodities = []Commodity{c.Commodities[0], ore, reagent}
	c.Recipes[0].Outputs = []Output{{Commodity: "ore", Quantity: 1, CostShare: 5_000}, {Commodity: "reagent", Quantity: 1, CostShare: 5_000}}
	c.Roles[0].Trades = []string{"ore", "reagent"}
	inner, ring := DefaultRegion(), DefaultRegion()
	inner.ID, ring.ID = "inner_system", "ring"
	c.Regions = []Region{inner, ring}
	c.Groups = []Group{{Region: "ring", Role: "trader", Households: 7}}
	c.Edges = []Edge{{From: "inner_system", To: "ring", Distance: 2_400}, {From: "ring", To: "inner_system", Distance: 2_400}}
	o := DefaultOutsideLine("ore", 120)
	o.Buy = OutsideSide{MarginBP: 2_000, Capacity: 200, TotalUnits: 1_500}
	r := DefaultOutsideLine("reagent", 300)
	r.TrendBP = -200
	r.Sell = OutsideSide{MarginBP: 800, Capacity: 10, TotalValue: 12_000}
	c.Modules.External = &External{Nodes: []OutsideNode{{Region: "inner_system", Lines: []OutsideLine{o, r}}}}
	return c
}

// TestExternal_WindowAndCaps verifies example B of spec-external-sector.md 18.3 and the
// window of 7.2. Ore: the bid is 96, the miners ask 90 so each trade is at 93; the
// outside takes 200 a tick, 1 400 in ticks 0 to 6, and in tick 7 the offer is cut to
// the remaining 100 (CutCap 100) and the side ends with the cause units: 1 500 units,
// list value 1 500 * 96 = 144 000. Reagent: the ask is ceil(300 * 10 800 / 10 000) = 324;
// the buyers bid 400 so each trade is at 362; 10 units in ticks 0 to 2 (list value
// 3 240 a tick), in tick 3 the remaining 12 000 - 9 720 = 2 280 allows
// floor(2 280 / 324) = 7 units (CutCap 3), the list value is 11 988 and the remaining
// 12 is below the quote, so the side ends with the cause value: 37 units for
// 37 * 362 = 13 394. Each end is one OutsideEnded, and an ended side places nothing.
// A season at 0 places nothing and counts no cut.
//
// Requirement: FR-EXT-029, FR-EXT-030, FR-EXT-031
func TestExternal_WindowAndCaps(t *testing.T) {
	t.Parallel()
	w, s, rec := extWorld(t, capConfig())
	ore, reagent := commodityOf(t, w, "ore"), commodityOf(t, w, "reagent")
	miner, buyer := AgentID(w.firstHousehold), AgentID(w.firstHousehold+1)
	s.plan[miner] = []Intent{{Commodity: ore, Side: Ask, Quantity: 200, Limit: 90}}
	s.plan[buyer] = []Intent{{Commodity: reagent, Side: Bid, Quantity: 10, Limit: 400}}
	for range 10 {
		give(t, w, miner, ore, 200)
		fund(t, w, buyer, 5_000)
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d = %v, want nil", w.Tick(), err)
		}
	}
	var oreEv, reagentEv []OutsideTraded
	for _, e := range of[OutsideTraded](rec) {
		if e.Commodity == ore {
			oreEv = append(oreEv, e)
		} else {
			reagentEv = append(reagentEv, e)
		}
	}
	if len(oreEv) != 8 || len(reagentEv) != 4 {
		t.Fatalf("OutsideTraded events for ore %d and reagent %d, want 8 (ticks 0 to 7) and 4 (ticks 0 to 3): an ended side places nothing", len(oreEv), len(reagentEv))
	}
	if e := oreEv[7]; e.Tick != 7 || e.Wanted != 200 || e.Offered != 100 || e.CutCap != 100 || e.Traded != 100 || e.Price != 96 {
		t.Errorf("ore in tick 7 = %+v, want 200 wanted, 100 offered, CutCap 100, 100 traded at 96", e)
	}
	if e := reagentEv[3]; e.Tick != 3 || e.Wanted != 10 || e.Offered != 7 || e.CutCap != 3 || e.Traded != 7 || e.Price != 324 {
		t.Errorf("reagent in tick 3 = %+v, want 10 wanted, 7 offered, CutCap 3, 7 traded at 324", e)
	}
	var paid, units int64
	for _, e := range reagentEv {
		paid += int64(e.Value)
		units += e.Traded
	}
	if paid != 13_394 || units != 37 {
		t.Errorf("reagent traded %d units for %d, want 37 for 13394 (37 * 362)", units, paid)
	}
	ended := of[OutsideEnded](rec)
	want := []OutsideEnded{
		{Tick: 3, Node: 0, Region: 1, Commodity: reagent, Side: Ask, Cause: CauseValue, Units: 37, ListValue: 11_988},
		{Tick: 7, Node: 0, Region: 1, Commodity: ore, Side: Bid, Cause: CauseUnits, Units: 1_500, ListValue: 144_000},
	}
	if !slices.Equal(ended, want) {
		t.Errorf("OutsideEnded = %+v, want %+v", ended, want)
	}
}

// TestExternal_ClosedSeasonCountsNoCut verifies that a season at 0 in the window of a
// side places nothing and counts nothing as cut (spec-external-sector.md 7.2).
//
// Requirement: FR-EXT-029
func TestExternal_ClosedSeasonCountsNoCut(t *testing.T) {
	t.Parallel()
	c := capConfig()
	m, err := c.Calendar.At(0)
	if err != nil {
		t.Fatalf("Calendar.At(0) = %v, want nil", err)
	}
	c.Modules.External.Nodes[0].Lines[0].Buy.Window = []SeasonFactor{{Season: c.Calendar.Seasons[m.Season].ID, BP: 0}}
	w, _, rec := extWorld(t, c)
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	ore := commodityOf(t, w, "ore")
	for _, e := range of[OutsideTraded](rec) {
		if e.Commodity == ore {
			t.Errorf("OutsideTraded for ore in its closed season = %+v, want none", e)
		}
	}
	for _, e := range of[OrderPlaced](rec) {
		if e.Commodity == ore {
			t.Errorf("OrderPlaced for ore in its closed season = %+v, want none", e)
		}
	}
}

// TestExternal_Identities verifies the identities of spec-external-sector.md 8.2 over
// three seeds with the audit on, 30 ticks of the realm of 18.1 re-supplied every tick:
// the units and the money in the events add up to the boundary balances of the
// reasons, exactly, and no step fails the audit (every counterparty account is empty
// at each boundary and the sectoral identity holds).
//
// Requirement: FR-EXT-019, FR-EXT-020, FR-EXT-012
func TestExternal_Identities(t *testing.T) {
	t.Parallel()
	for _, seed := range []uint64{1, 2, 3} {
		w, s, rec := extWorld(t, extConfig(), WithSeed(seed))
		grain, salt := commodityOf(t, w, "grain"), commodityOf(t, w, "salt")
		for i, o := range []struct {
			limit Price
			units Quantity
		}{{15, 20}, {17, 15}, {19, 10}, {21, 30}} {
			s.plan[AgentID(w.firstHousehold+i)] = []Intent{{Commodity: grain, Side: Ask, Quantity: o.units, Limit: o.limit}}
		}
		for i, o := range []struct {
			limit Price
			units Quantity
		}{{50, 10}, {45, 10}, {41, 10}} {
			s.plan[AgentID(w.firstHousehold+4+i)] = []Intent{{Commodity: salt, Side: Bid, Quantity: o.units, Limit: o.limit}}
		}
		for range 30 {
			for i := range 7 {
				give(t, w, AgentID(w.firstHousehold+i), commodityOf(t, w, "bread"), 1) // the upkeep, so nobody starves
			}
			for i := range 4 {
				give(t, w, AgentID(w.firstHousehold+i), grain, 30)
			}
			for i := range 3 {
				fund(t, w, AgentID(w.firstHousehold+4+i), 600)
			}
			if err := w.Step(); err != nil {
				t.Fatalf("seed %d: Step() at tick %d = %v, want nil (the audit failed or a ledger refusal)", seed, w.Tick(), err)
			}
		}
		var exportUnits, exportValue, importUnits, importValue int64
		for _, e := range of[OutsideTraded](rec) {
			if e.Side == Bid {
				exportUnits += e.Traded
				exportValue += int64(e.Value)
			} else {
				importUnits += e.Traded
				importValue += int64(e.Value)
			}
		}
		checks := []struct {
			name      string
			got, want int64
		}{
			{"export units = outside_purchases", exportUnits, boundaryOf(t, w, ledger.OutsidePurchases, int(grain))},
			{"import units = outside_offer - outside_unsold", importUnits, -boundaryOf(t, w, ledger.OutsideOffer, int(salt)) - boundaryOf(t, w, ledger.OutsideUnsold, int(salt))},
			{"export value = outside_budget - outside_unspent", exportValue, -boundaryOf(t, w, ledger.OutsideBudget, ledger.Money) - boundaryOf(t, w, ledger.OutsideUnspent, ledger.Money)},
			{"import value = outside_receipts", importValue, boundaryOf(t, w, ledger.OutsideReceipts, ledger.Money)},
		}
		for _, c := range checks {
			if c.got != c.want || c.want == 0 {
				t.Errorf("seed %d: %s: events %d, boundary %d, want equal and non-zero", seed, c.name, c.got, c.want)
			}
		}
	}
}
