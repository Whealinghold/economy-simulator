package economysim_test

import (
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestInvariantChecker_Balances verifies the conservation checks on
// hand-built snapshots: balanced, one unit off, and balanced with a nonzero
// pending that identity (i) must not count.
//
// Requirement: FR-REF-028, FR-OBS-042, FR-LED-045
func TestInvariantChecker_Balances(t *testing.T) {
	t.Parallel()
	const wood = economysim.CommodityID(0)
	market := func(held economysim.Quantity, pending economysim.Quantity) economysim.MarketRow {
		return economysim.MarketRow{State: economysim.MarketState{Commodity: wood}, HeldByAgents: held, Pending: pending}
	}
	snap := func(money economysim.Money, held, pending economysim.Quantity, goodsBalance, moneyBalance int64) economysim.Snapshot {
		return economysim.Snapshot{
			Tick:    1,
			Money:   money,
			Markets: []economysim.MarketRow{market(held, pending)},
			Boundary: []economysim.BoundaryRow{
				{Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: wood, Balance: goodsBalance},
				{Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: economysim.NoCommodity, Balance: moneyBalance},
			},
		}
	}
	// The faucet put 10 wood and 100 money in at construction.
	build := func() *invChecker {
		return newInvChecker(economysim.Snapshot{Construction: []economysim.FlowTotal{
			{Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: wood, Total: 10},
			{Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: economysim.NoCommodity, Total: 100},
		}})
	}
	tests := []struct {
		name string
		s    economysim.Snapshot
		want int
	}{
		{"balanced", snap(100, 10, 0, -10, -100), 0},
		{"one unit off", snap(100, 9, 0, -10, -100), 1},
		{"one coin off", snap(99, 10, 0, -10, -100), 1},
		{"pending excluded from identity (i)", snap(100, 10, 7, -10, -100), 0},
		{"balance moved without a flow", snap(101, 10, 0, -10, -101), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := build()
			c.bucketChecked = true
			got := c.balances(tt.s)
			if len(got) != tt.want {
				t.Errorf("balances = %q, want %d failure(s)", got, tt.want)
			}
		})
	}
}

// TestInvariantChecker_Flows verifies that flows of later ticks and Pending
// move the expected balance, that the Tick -1 events must equal the
// construction bucket, and that an undeclared boundary is reported.
//
// Requirement: FR-OBS-042, FR-OBS-015
func TestInvariantChecker_Flows(t *testing.T) {
	t.Parallel()
	const wood = economysim.CommodityID(0)
	pre := economysim.Snapshot{Construction: []economysim.FlowTotal{{Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: wood, Total: 10}}}
	row := func(r economysim.Reason, bal int64) economysim.BoundaryRow {
		return economysim.BoundaryRow{Reason: r, Kind: economysim.KindFaucet, Commodity: wood, Balance: bal}
	}
	flow := func(tick int64, r economysim.Reason, k economysim.Kind, n int64) economysim.Event {
		return economysim.BoundaryFlow{Tick: tick, Reason: r, Kind: k, Commodity: wood, Total: n}
	}
	c := newInvChecker(pre)
	c.events([]economysim.Event{flow(-1, economysim.ReasonInitial, economysim.KindFaucet, 10)})
	c.events([]economysim.Event{flow(0, economysim.ReasonConsumption, economysim.KindSink, 4)})
	s := economysim.Snapshot{
		Tick:     1,
		Markets:  []economysim.MarketRow{{State: economysim.MarketState{Commodity: wood}, HeldByAgents: 6}},
		Boundary: []economysim.BoundaryRow{row(economysim.ReasonInitial, -10), row(economysim.ReasonConsumption, 4)},
	}
	if got := c.balances(s); len(got) != 0 {
		t.Errorf("balances = %q, want none", got)
	}
	s.Pending = []economysim.FlowTotal{{Reason: economysim.ReasonConsumption, Kind: economysim.KindSink, Commodity: wood, Total: 1}}
	if got := c.balances(s); len(got) != 1 {
		t.Errorf("balances with an unbalanced pending = %q, want 1 failure", got)
	}
	bad := newInvChecker(pre)
	bad.events([]economysim.Event{flow(-1, economysim.ReasonInitial, economysim.KindFaucet, 9)})
	s.Pending = nil
	if got := bad.balances(s); len(got) == 0 {
		t.Error("a Tick -1 bucket that differs from Construction was not reported")
	}
	und := newInvChecker(pre)
	und.events([]economysim.Event{flow(0, economysim.ReasonDisposal, economysim.KindSink, 1)})
	if got := und.balances(s); len(got) == 0 {
		t.Error("a flow of an undeclared boundary was not reported")
	}
}

// TestInvariantChecker_Events verifies the live set rebuilt from
// AgentSpawned and AgentBankrupted, the estate exemption, and the
// price-between-limits check on constructed events: equal to the bid, equal
// to the ask, one above, one below.
//
// Requirement: FR-REF-028, FR-CLR-019
func TestInvariantChecker_Events(t *testing.T) {
	t.Parallel()
	const wood = economysim.CommodityID(0)
	order := func(a economysim.AgentID, side economysim.Side, limit economysim.Price) economysim.Event {
		return economysim.OrderPlaced{Tick: 3, Commodity: wood, Agent: a, Side: side, Limit: limit}
	}
	trade := func(buyer, seller economysim.AgentID, p economysim.Price) economysim.Event {
		return economysim.TradeExecuted{Tick: 3, Commodity: wood, Buyer: buyer, Seller: seller, Price: p, Quantity: 1}
	}
	spawn := func(a economysim.AgentID) economysim.Event { return economysim.AgentSpawned{Agent: a} }
	newC := func() *invChecker { return &invChecker{regions: 1, live: map[economysim.AgentID]bool{}} }
	tests := []struct {
		name string
		evs  []economysim.Event
		want int
	}{
		{"price equals bid", []economysim.Event{spawn(5), spawn(6), order(5, economysim.Bid, 10), order(6, economysim.Ask, 8), trade(5, 6, 10)}, 0},
		{"price equals ask", []economysim.Event{spawn(5), spawn(6), order(5, economysim.Bid, 10), order(6, economysim.Ask, 8), trade(5, 6, 8)}, 0},
		{"one above the bid", []economysim.Event{spawn(5), spawn(6), order(5, economysim.Bid, 10), order(6, economysim.Ask, 8), trade(5, 6, 11)}, 1},
		{"one below the ask", []economysim.Event{spawn(5), spawn(6), order(5, economysim.Bid, 10), order(6, economysim.Ask, 8), trade(5, 6, 7)}, 1},
		{"unknown buyer", []economysim.Event{spawn(6), trade(5, 6, 9)}, 1},
		{"bankrupt seller is gone", []economysim.Event{spawn(5), spawn(6), economysim.AgentBankrupted{Agent: 6}, trade(5, 6, 9)}, 1},
		{"estate seller needs no live entry", []economysim.Event{spawn(5), order(5, economysim.Bid, 10), trade(5, 0, 9)}, 0},
		{"estate seller above the bid", []economysim.Event{spawn(5), order(5, economysim.Bid, 10), trade(5, 0, 11)}, 1},
		{"rejected by overflow", []economysim.Event{economysim.OrderRejected{Tick: 3, Agent: 5, Reason: economysim.ReasonOverflow}}, 1},
		{"rejected by funds is fine", []economysim.Event{economysim.OrderRejected{Tick: 3, Agent: 5, Reason: economysim.ReasonInsufficientFunds}}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := newC().events(tt.evs); len(got) != tt.want {
				t.Errorf("events = %q, want %d failure(s)", got, tt.want)
			}
		})
	}
}

// TestInvariantChecker_NoMoneyFlowAfterConstruction verifies the money-flow
// check on constructed events: a money flow in a step (Tick >= 0) is flagged and
// names that tick; the construction bucket (Tick -1), a goods flow and an empty
// step are not.
//
// Requirement: FR-REF-028, FR-OBS-042
// Guard: the packs have no money faucet or sink after construction; without the
// test a check that flagged nothing would let one through.
func TestInvariantChecker_NoMoneyFlowAfterConstruction(t *testing.T) {
	t.Parallel()
	money := func(tick int64) economysim.Event {
		return economysim.BoundaryFlow{Tick: tick, Reason: "grant", Kind: economysim.KindFaucet, Commodity: economysim.NoCommodity, Total: 5}
	}
	goods := economysim.BoundaryFlow{Tick: 4, Reason: "harvest", Kind: economysim.KindFaucet, Commodity: 0, Total: 5}
	tests := []struct {
		name string
		evs  []economysim.Event
		want int
	}{
		{"no events", nil, 0},
		{"construction bucket", []economysim.Event{money(-1)}, 0},
		{"goods flow in a step", []economysim.Event{goods}, 0},
		{"money flow at tick 0", []economysim.Event{money(0)}, 1},
		{"money flow in a later step", []economysim.Event{goods, money(7)}, 1},
		{"two money flows", []economysim.Event{money(2), money(2)}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := noMoneyFlowAfterConstruction(tt.evs)
			if len(got) != tt.want {
				t.Fatalf("noMoneyFlowAfterConstruction = %q, want %d message(s)", got, tt.want)
			}
			if tt.name == "money flow in a later step" && !strings.Contains(got[0], "tick 7") {
				t.Errorf("message = %q, want it to name tick 7", got[0])
			}
		})
	}
}
