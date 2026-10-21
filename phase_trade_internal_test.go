package economysim

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file tests phases 3 and 4 of the tick (phase_trade.go, plan-phase-1-11.md
// Phase 5) over scripted worlds: a hand-written Strategy at the extension point
// places the orders a test names, so every expectation below is arithmetic done by
// hand. The commodity IDs are those of millConfig: flour 0, grain 1. Agent 0 is
// the estate of the one region, so the households are agents 1, 2, ...

// tradeScript is a Strategy that requests no job and trades as trade says; a nil
// trade places nothing. Revise keeps the belief.
type tradeScript struct {
	trade func(v *AgentView, p *TradePlan)
}

func (*tradeScript) Work(*AgentView, *rand.Rand) JobRequest { return JobRequest{NoJob: true} }

func (s *tradeScript) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	if s.trade != nil {
		s.trade(v, p)
	}
}

func (*tradeScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = (*tradeScript)(nil)

// tradeConfig is the chain of the trade tests: millConfig with a role that
// follows the registered strategy "script" and each household holding 10 flour and
// 10 grain and 10 000 minor units.
func tradeConfig(households int64) Config {
	c := millConfig(1, households)
	c.Roles[0].Strategy = "script"
	c.Roles[0].StartStock = []Item{{Commodity: "flour", Quantity: 10}, {Commodity: "grain", Quantity: 10}}
	return c
}

// newTrader builds the world of c whose agents trade as trade says.
func newTrader(t *testing.T, c Config, trade func(v *AgentView, p *TradePlan), opts ...Option) *World {
	t.Helper()
	return newMill(t, c, append([]Option{WithStrategy("script", &tradeScript{trade: trade})}, opts...)...)
}

// balance returns the money an agent holds and has earmarked.
func balance(t *testing.T, w *World, id AgentID) (held, reserved int64) {
	t.Helper()
	v, err := w.ledger.Account(w.agents[id].account)
	if err != nil {
		t.Fatalf("Account(agent %d) error = %v, want nil", id, err)
	}
	return v.Balance, v.Reserved
}

// reservedGoods returns the units of a commodity an agent has earmarked.
func reservedGoods(t *testing.T, w *World, id AgentID, commodity int) int64 {
	t.Helper()
	_, r, err := w.ledger.Holding(w.agents[id].account, commodity)
	if err != nil {
		t.Fatalf("Holding(agent %d, commodity %d) error = %v, want nil", id, commodity, err)
	}
	return r
}

// TestStep_ScriptedMarketClearsAtTheHalfEvenMidpoint verifies one market of two
// agents: agent 1 bids 3 grain at a limit and agent 2 asks 2 grain at a limit. A
// crossing pair trades the smaller quantity at the midpoint rounded half to even,
// the balances and stocks move by the trade, no reservation is left open, the
// events of the market are OrderPlaced twice, TradeExecuted and MarketCleared in
// that order, and their fields equal the arithmetic. A bid below the ask trades
// nothing and the market is cleared all the same.
//
// Arithmetic: 3 against 2 at 1 100 and 900 trade min(3, 2) = 2 at (1 100 + 900) /
// 2 = 1 000, pay 2 * 1 000 = 2 000. At 1 101 and 900 the midpoint is 2 001 / 2 =
// 1 000.5, which is half, so it goes to the even 1 000. At 1 103 and 900 it is 2
// 003 / 2 = 1 001.5, to the even 1 002 (pay 2 004). The unmatched bid is 3 - 2 = 1
// at the bid limit. At 800 and 900 nothing crosses: volume 0, the price stays at
// the seed 1 000 (grain's reference price), its age rises to 1, 3 bid and 2 ask
// units are unmatched and the best prices are the two limits.
//
// Requirement: FR-CLR-001, FR-CLR-004, FR-WLD-047, FR-WLD-048, FR-OBS-022
func TestStep_ScriptedMarketClearsAtTheHalfEvenMidpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		bid, ask     Price
		wantPrice    Price
		wantQty      Quantity
		wantAge      int64
		wantUnBid    Quantity
		wantUnAsk    Quantity
		wantBestBid  Price
		wantBestAsk  Price
		wantTradeSet bool
	}{
		{"cross at the midpoint", 1_100, 900, 1_000, 2, 0, 1, 0, 1_100, 0, true},
		{"half goes down to the even", 1_101, 900, 1_000, 2, 0, 1, 0, 1_101, 0, true},
		{"half goes up to the even", 1_103, 900, 1_002, 2, 0, 1, 0, 1_103, 0, true},
		{"a bid below the ask does not trade", 800, 900, 1_000, 0, 1, 3, 2, 800, 900, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &evLog{}
			w := newTrader(t, tradeConfig(2), func(v *AgentView, p *TradePlan) {
				switch v.Agent {
				case 1:
					p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: 3, Limit: tt.bid})
				case 2:
					p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Ask, Quantity: 2, Limit: tt.ask})
				}
			}, WithObserver(rec), WithAudit())
			rec.events = nil
			step(t, w, 1)
			seen := marketEvents(rec, grain)
			wantSeen := []Event{
				OrderPlaced{Tick: 0, Region: 0, Commodity: grain, Agent: 1, Side: Bid, Limit: tt.bid, Quantity: 3},
				OrderPlaced{Tick: 0, Region: 0, Commodity: grain, Agent: 2, Side: Ask, Limit: tt.ask, Quantity: 2},
			}
			if tt.wantTradeSet {
				wantSeen = append(wantSeen, TradeExecuted{Tick: 0, Region: 0, Commodity: grain, Buyer: 1, Seller: 2, Price: tt.wantPrice, Quantity: tt.wantQty})
			}
			wantSeen = append(wantSeen, MarketCleared{
				Tick: 0, Region: 0, Commodity: grain, Price: tt.wantPrice, PriceAge: tt.wantAge, Volume: tt.wantQty,
				UnmatchedBid: tt.wantUnBid, UnmatchedAsk: tt.wantUnAsk, BestBid: tt.wantBestBid, BestAsk: tt.wantBestAsk,
				HasBestBid: tt.wantBestBid > 0, HasBestAsk: tt.wantBestAsk > 0,
			})
			if len(seen) != len(wantSeen) {
				t.Fatalf("events of the grain market = %+v, want %+v", seen, wantSeen)
			}
			for i := range seen {
				if seen[i] != wantSeen[i] {
					t.Errorf("event %d of the grain market = %+v, want %+v", i, seen[i], wantSeen[i])
				}
			}
			pay := int64(tt.wantPrice) * int64(tt.wantQty)
			buyer, _ := balance(t, w, 1)
			seller, _ := balance(t, w, 2)
			if buyer != 10_000-pay || seller != 10_000+pay {
				t.Errorf("balances after the step = %d and %d, want %d and %d", buyer, seller, 10_000-pay, 10_000+pay)
			}
			// Each agent eats its upkeep of 1 grain at P6.1, so 10 +/- qty - 1.
			if g1, g2 := held(t, w, 1, grain), held(t, w, 2, grain); g1 != 9+int64(tt.wantQty) || g2 != 9-int64(tt.wantQty) {
				t.Errorf("grain held = %d and %d, want %d and %d", g1, g2, 9+int64(tt.wantQty), 9-int64(tt.wantQty))
			}
			if w.ledger.HasReservation() {
				t.Errorf("a reservation is open after the step, want none")
			}
		})
	}
}

// marketEvents returns the OrderPlaced and TradeExecuted events and the
// MarketCleared event of the market of a commodity, in the order they came.
func marketEvents(rec *evLog, commodity CommodityID) []Event {
	var seen []Event
	for _, e := range rec.events {
		switch x := e.(type) {
		case OrderPlaced, TradeExecuted:
			seen = append(seen, e)
		case MarketCleared:
			if x.Commodity == commodity {
				seen = append(seen, e)
			}
		}
	}
	return seen
}

// reserveWindow runs the positions of a step up to and including P2.8a by
// calling them one by one, as the replay of the step does, and returns with the
// step open so the caller can call P3.2 and the rest.
func reserveWindow(t *testing.T, w *World) {
	t.Helper()
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep() at tick %d error = %v, want nil", w.Tick(), err)
	}
	if err := w.p11Moment(); err != nil {
		t.Fatalf("p11Moment() error = %v, want nil", err)
	}
	w.p12Anchors()
	w.p21Complete()
	for _, f := range []func() error{w.p25Work, w.p27Start} {
		if err := f(); err != nil {
			t.Fatalf("a production position at tick %d error = %v, want nil", w.Tick(), err)
		}
	}
	w.p28aHalted()
}

// wantEarmarks checks the earmarks of the orders of TestPositions_NoReservationIsOpenAfterP41
// after P3.2: 3 300 minor units of agent 1 and 2 units of agent 2's grain.
func wantEarmarks(t *testing.T, w *World, ask Price) {
	t.Helper()
	_, money := balance(t, w, 1)
	goods := reservedGoods(t, w, 2, grain)
	if money != 3_300 || goods != 2 || !w.ledger.HasReservation() {
		t.Errorf("ask %d: after P3.2 reserved money = %d and goods = %d, want 3300 and 2", ask, money, goods)
	}
}

// TestPositions_NoReservationIsOpenAfterP41 verifies the window of reservations
// (spec-world.md 6.1.6, FR-WLD-045) directly: with orders placed, a reservation is
// open after P3.2 and after P3.4, and none is open when the P4.1 method returns
// and before P5.1, whether or not the orders traded. A check made only at a tick
// boundary would pass a reservation that a later position released.
//
// Arithmetic: agent 1 bids 3 grain at 1 100, which earmarks 3 * 1 100 = 3 300
// minor units, and agent 2 asks 2 grain, which earmarks 2 units. After P4.1 both
// remainders are released, so the earmarks are 0.
//
// Requirement: FR-WLD-045, FR-LED-020
func TestPositions_NoReservationIsOpenAfterP41(t *testing.T) {
	t.Parallel()
	for _, ask := range []Price{900, 1_500} { // crossing, then not crossing
		w := newTrader(t, tradeConfig(2), func(v *AgentView, p *TradePlan) {
			if v.Agent == 1 {
				p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: 3, Limit: 1_100})
			} else {
				p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Ask, Quantity: 2, Limit: ask})
			}
		})
		reserveWindow(t, w)
		if err := w.p32Trade(); err != nil {
			t.Fatalf("ask %d: p32Trade() error = %v, want nil", ask, err)
		}
		wantEarmarks(t, w, ask)
		if err := w.p34Estates(); err != nil {
			t.Fatalf("ask %d: p34Estates() error = %v, want nil", ask, err)
		}
		if !w.ledger.HasReservation() {
			t.Errorf("ask %d: no reservation is open after P3.4, want the orders' earmarks", ask)
		}
		if err := w.p41Clear(); err != nil {
			t.Fatalf("ask %d: p41Clear() error = %v, want nil", ask, err)
		}
		if w.ledger.HasReservation() {
			t.Errorf("ask %d: a reservation is open after P4.1, want none (spec-world.md 6.1.6)", ask)
		}
		if err := w.endStep(nil); err != nil {
			t.Fatalf("ask %d: endStep() error = %v, want nil", ask, err)
		}
	}
}

// TestStep_ReceiptsAndInputSpendEqualTheTradeFlows verifies the welfare feeds of
// P4.1 against the ledger (spec-world.md 6.1.13, FR-AGT-057 half): over a scripted
// market the receipts the scratch holds for the sellers equal the money flow of the
// trade reason for the tick, the spend by purpose Input is charged to the buyer
// only for a bid of that purpose, and both sides of the settled trade have a
// working tick. The scratch is read before P8.6 clears it.
//
// Arithmetic: 2 units at the midpoint (1 100 + 900) / 2 = 1 000 is 2 000 of money
// moved by the trade reason. The seller received 2 000. The buyer's bid had the
// purpose Input in the first case, so its input spend is 2 000, and Upkeep in the
// second, so its input spend is 0.
//
// Requirement: FR-AGT-057, FR-WEL-017, FR-WLD-058
func TestStep_ReceiptsAndInputSpendEqualTheTradeFlows(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name      string
		purpose   Purpose
		wantSpend int64
	}{
		{"a bid for an input", PurposeInput, 2_000},
		{"a bid for upkeep", PurposeUpkeep, 0},
		{"a bid for a tool is not input spend", PurposeTool, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := newTrader(t, tradeConfig(2), func(v *AgentView, p *TradePlan) {
				if v.Agent == 1 {
					p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: 2, Limit: 1_100, Purpose: tt.purpose})
				} else {
					p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Ask, Quantity: 2, Limit: 900})
				}
			})
			tickInspecting(t, w, func() {
				tradeFlow := flowOf(w, ledger.Trade, ledger.Money)
				var receipts int64
				for i := range w.welfare.slots {
					receipts += w.welfare.slots[i].receipts
				}
				if tradeFlow != 2_000 || receipts != tradeFlow {
					t.Errorf("trade flow = %d and receipts = %d, want both 2000", tradeFlow, receipts)
				}
				if got := w.welfare.slots[1].inputSpend; got != tt.wantSpend {
					t.Errorf("input spend of the buyer = %d, want %d", got, tt.wantSpend)
				}
				if got := w.welfare.slots[2].receipts; got != 2_000 {
					t.Errorf("receipts of the seller = %d, want 2000", got)
				}
				if !w.welfare.slots[1].traded || !w.welfare.slots[2].traded {
					t.Errorf("traded = %v and %v, want both true for the two sides of a trade", w.welfare.slots[1].traded, w.welfare.slots[2].traded)
				}
			})
		})
	}
}

// TestStep_APlanIsExecutedDiscardsThenOrders verifies the order of execution of
// a plan (spec-agents.md 6.5.2, FR-AGT-057): the discards go first, to the
// disposal boundary, so an ask sized to the stock left by them is admitted and one
// sized to the stock before them is rejected.
//
// Arithmetic: the agent holds 10 grain and discards 4, so 6 are left. An ask of 6
// fits and is admitted; an ask of 8 would have fitted the 10 held before the
// discard and is rejected insufficient_stock after it. Either way 4 units flow to
// disposal.
//
// Requirement: FR-AGT-057, FR-WLD-047
func TestStep_APlanIsExecutedDiscardsThenOrders(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		ask   Quantity
		wantE Event
	}{
		{"an ask that fits what the discard left", 6, OrderPlaced{Region: 0, Commodity: grain, Agent: 2, Side: Ask, Limit: 900, Quantity: 6}},
		{"an ask that needed the discarded units", 8, OrderRejected{Region: 0, Commodity: grain, Agent: 2, Side: Ask, Reason: ReasonInsufficientStock}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &evLog{}
			w := newTrader(t, tradeConfig(2), func(v *AgentView, p *TradePlan) {
				if v.Agent == 2 {
					p.Discards = append(p.Discards, Discard{Commodity: grain, Quantity: 4})
					p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Ask, Quantity: tt.ask, Limit: 900})
				}
			}, WithObserver(rec))
			rec.events = nil
			tickInspecting(t, w, func() {
				if got := flowOf(w, ledger.Disposal, grain); got != 4 {
					t.Errorf("disposal flow of grain = %d, want 4", got)
				}
			})
			var got []Event
			for _, e := range rec.events {
				switch e.(type) {
				case OrderPlaced, OrderRejected:
					got = append(got, e)
				}
			}
			if len(got) != 1 || got[0] != tt.wantE {
				t.Errorf("order events = %+v, want [%+v]", got, tt.wantE)
			}
		})
	}
}

// TestStep_ADiscardTheAgentCannotMakeIsRefusedWhole verifies that a discard of
// more than the agent holds is refused whole and logged at Debug "discard
// refused" with the agent, commodity and quantity, that the discards before it
// still go (spec-agents.md 6.5.2), and that a discard of a quantity below 1 or of a
// commodity outside the configuration is refused too and never panics.
//
// Arithmetic: 10 grain held. Discards of 6 and 6: the first leaves 4, so the second
// is refused (6 > 4) and 6 flow to disposal. A discard of 11 is refused whole and
// nothing flows. A discard of 0, of -3 and of commodity 99 move nothing.
//
// Requirement: FR-AGT-057
func TestStep_ADiscardTheAgentCannotMakeIsRefusedWhole(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name        string
		discards    []Discard
		wantFlow    int64
		wantRefused int
	}{
		{"the second of two cannot be made", []Discard{{Commodity: grain, Quantity: 6}, {Commodity: grain, Quantity: 6}}, 6, 1},
		{"more than is held", []Discard{{Commodity: grain, Quantity: 11}}, 0, 1},
		{"a quantity below 1", []Discard{{Commodity: grain, Quantity: 0}, {Commodity: grain, Quantity: -3}}, 0, 2},
		{"a commodity outside the configuration", []Discard{{Commodity: 99, Quantity: 1}, {Commodity: -1, Quantity: 1}}, 0, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lines := newCapture()
			w := newTrader(t, tradeConfig(2), func(v *AgentView, p *TradePlan) {
				if v.Agent == 1 {
					p.Discards = append(p.Discards, tt.discards...)
				}
			}, WithLogger(slog.New(lines)))
			tickInspecting(t, w, func() {
				if got := flowOf(w, ledger.Disposal, grain); got != tt.wantFlow {
					t.Errorf("disposal flow of grain = %d, want %d", got, tt.wantFlow)
				}
			})
			refused := lines.take("discard refused")
			if len(refused) != tt.wantRefused {
				t.Fatalf("\"discard refused\" lines = %d, want %d", len(refused), tt.wantRefused)
			}
			if refused[0].level != slog.LevelDebug || refused[0].attrs["agent"] != 1 {
				t.Errorf("first \"discard refused\" line = %+v, want Debug with agent 1", refused[0])
			}
		})
	}
}

// TestStep_AnIntentIsNormalisedAndItsFactsKept verifies that the clamps of
// spec-agents.md 6.5.2 apply to an intent before its order is made: a negative
// Target, a Reserve above MaxQuantity and a Purpose outside the enumeration are read
// as 0 and PurposeOther, the order is still placed, the Debug line "intent
// normalised" is written, and the world keeps the normalised Target, Reserve,
// UnitCost and Purpose at the index of the book's entry (D194).
//
// Requirement: FR-AGT-048, FR-AGT-025
func TestStep_AnIntentIsNormalisedAndItsFactsKept(t *testing.T) {
	t.Parallel()
	lines := newCapture()
	rec := &evLog{}
	w := newTrader(t, tradeConfig(2), func(v *AgentView, p *TradePlan) {
		if v.Agent == 1 {
			p.Intents = append(p.Intents,
				Intent{Commodity: grain, Side: Bid, Quantity: 1, Limit: 900, Target: -5, Reserve: MaxQuantity + 1, UnitCost: -1, Purpose: 99},
				Intent{Commodity: flour, Side: Bid, Quantity: 1, Limit: 900, Target: 7, Reserve: 2, UnitCost: 800, Purpose: PurposeInput})
		}
	}, WithLogger(slog.New(lines)), WithObserver(rec))
	rec.events = nil
	tickInspecting(t, w, func() {
		gi, fi := w.markets.index(0, grain), w.markets.index(0, flour)
		if got, want := w.markets.intents[gi], []intentRec{{purpose: PurposeOther}}; !slices.Equal(got, want) {
			t.Errorf("record of the clamped bid = %+v, want %+v (everything read as 0 and PurposeOther)", got, want)
		}
		if got, want := w.markets.intents[fi], []intentRec{{purpose: PurposeInput, target: 7, reserve: 2, unitCost: 800}}; !slices.Equal(got, want) {
			t.Errorf("record of the clean bid = %+v, want %+v (kept untouched through P4.1)", got, want)
		}
	})
	if n := len(of[OrderPlaced](rec)); n != 2 {
		t.Errorf("OrderPlaced events = %d, want 2: a clamp never rejects the order", n)
	}
	if got := lines.take("intent normalised"); len(got) != 1 || got[0].level != slog.LevelDebug {
		t.Errorf("\"intent normalised\" lines = %+v, want one at Debug (only the clamped intent)", got)
	}
}

// TestStep_AnOrderTheBookCannotAdmitIsRejectedWithItsReason verifies the
// rejections of the world and the book at P3.2: an intent of a commodity outside the
// configuration is unknown_commodity, of a side that is neither bid nor ask is
// side_not_allowed (a consumer's strategy can return either, and neither may panic),
// a commodity outside the role's tradable set is not_tradable, a second intent for a
// commodity and side is duplicate_order, a bid beyond the money is insufficient_funds
// and an ask beyond the stock is insufficient_stock. Each is an OrderRejected that
// names the agent, region, commodity and side, and none enters a book.
//
// Arithmetic: 10 000 minor units held, so a bid of 11 units at 1 000 (11 000) is
// beyond the money; 10 grain held, so an ask of 11 is beyond the stock.
//
// Requirement: FR-AGT-025, FR-AGT-026, FR-CLR-023, FR-WLD-047
func TestStep_AnOrderTheBookCannotAdmitIsRejectedWithItsReason(t *testing.T) {
	t.Parallel()
	c := tradeConfig(2)
	salt := DefaultCommodity()
	salt.ID, salt.ReferencePrice, salt.Raw = "salt", 500, true
	c.Commodities = append(c.Commodities, salt)
	other := DefaultRole()
	other.ID, other.Strategy, other.StartStock = "trader", "script", []Item{{Commodity: "salt", Quantity: 5}}
	other.Upkeep = []Item{{Commodity: "salt", Quantity: 1}}
	c.Roles = append(c.Roles, other)
	c.Groups = append(c.Groups, Group{Region: "home", Role: "trader", Households: 1})
	for _, tt := range []struct {
		name   string
		in     Intent
		want   Reason
		wantAt CommodityID
	}{
		{"unknown commodity", Intent{Commodity: 99, Side: Bid, Quantity: 1, Limit: 900}, ReasonUnknownCommodity, 99},
		{"negative commodity", Intent{Commodity: -1, Side: Ask, Quantity: 1, Limit: 900}, ReasonUnknownCommodity, -1},
		{"a side that is neither", Intent{Commodity: grain, Side: 7, Quantity: 1, Limit: 900}, ReasonSideNotAllowed, grain},
		{"outside the tradable set", Intent{Commodity: 2, Side: Bid, Quantity: 1, Limit: 900}, ReasonNotTradable, 2},
		{"a bid beyond the money", Intent{Commodity: grain, Side: Bid, Quantity: 11, Limit: 1_000}, ReasonInsufficientFunds, grain},
		{"an ask beyond the stock", Intent{Commodity: grain, Side: Ask, Quantity: 11, Limit: 900}, ReasonInsufficientStock, grain},
		{"a price below 1", Intent{Commodity: grain, Side: Bid, Quantity: 1, Limit: 0}, ReasonInvalidPrice, grain},
		{"a quantity below 1", Intent{Commodity: grain, Side: Bid, Quantity: 0, Limit: 900}, ReasonInvalidQuantity, grain},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &evLog{}
			w := newTrader(t, c, func(v *AgentView, p *TradePlan) {
				if v.Agent == 1 {
					p.Intents = append(p.Intents, tt.in)
				}
			}, WithObserver(rec), WithAudit())
			rec.events = nil
			step(t, w, 1)
			want := OrderRejected{Tick: 0, Region: 0, Commodity: tt.wantAt, Agent: 1, Side: tt.in.Side, Reason: tt.want}
			if got := of[OrderRejected](rec); len(got) != 1 || got[0] != want {
				t.Errorf("OrderRejected events = %+v, want [%+v]", got, want)
			}
			if n := len(of[OrderPlaced](rec)); n != 0 {
				t.Errorf("OrderPlaced events = %d, want 0", n)
			}
			for i, b := range w.markets.books {
				if len(b.entries) != 0 {
					t.Errorf("market %d holds %d entries after a rejected order, want 0", i, len(b.entries))
				}
			}
		})
	}
	t.Run("a second intent for a commodity and side", func(t *testing.T) {
		t.Parallel()
		rec := &evLog{}
		w := newTrader(t, c, func(v *AgentView, p *TradePlan) {
			if v.Agent == 1 {
				in := Intent{Commodity: grain, Side: Bid, Quantity: 1, Limit: 900}
				p.Intents = append(p.Intents, in, in)
			}
		}, WithObserver(rec))
		rec.events = nil
		step(t, w, 1)
		want := OrderRejected{Tick: 0, Region: 0, Commodity: grain, Agent: 1, Side: Bid, Reason: ReasonDuplicateOrder}
		if got := of[OrderRejected](rec); len(got) != 1 || got[0] != want {
			t.Errorf("OrderRejected events = %+v, want [%+v]", got, want)
		}
	})
}

// splitClearing is a Clearing that gives the first bid two trades, one with each
// of the first two asks (2 units each, at 1 000): a custom Clearing may return
// several trades for one order (D194).
type splitClearing struct{}

func (splitClearing) Clear(bids, asks []Order, _ *rand.Rand) []Trade {
	if len(bids) == 0 || len(asks) < 2 {
		return nil
	}
	b := bids[0]
	return []Trade{
		{Commodity: b.Commodity, Buyer: b.Agent, Seller: asks[0].Agent, Price: 1_000, Quantity: 2},
		{Commodity: b.Commodity, Buyer: b.Agent, Seller: asks[1].Agent, Price: 1_000, Quantity: 2},
	}
}

// TestStep_TheBuyersPurposeReachesEveryTradeOfItsBid verifies D194 over a Clearing
// that returns two trades for one bid: the purpose of the bid reaches the spend
// feed of both trades, both sellers are credited and marked as having traded, and
// the slot table is clear after P4.1, so in the next tick the same agent's bid of
// another purpose is not charged with the first tick's.
//
// Arithmetic: agent 1 bids 4 grain at 1 100 with purpose Input; agents 2 and 3 ask
// 2 each at 900. The two trades are 2 * 1 000 = 2 000 each, so the input spend is
// 4 000 and each seller received 2 000. In tick 1 the bid has the purpose Upkeep,
// so its input spend is 0 though tick 0's slot said Input.
//
// Requirement: FR-AGT-057, FR-WEL-017, FR-WLD-058
func TestStep_TheBuyersPurposeReachesEveryTradeOfItsBid(t *testing.T) {
	t.Parallel()
	c := tradeConfig(3)
	c.World.Clearing = "split"
	var w *World
	w = newTrader(t, c, func(v *AgentView, p *TradePlan) {
		switch v.Agent {
		case 1:
			purpose := PurposeInput
			if w.Tick() == 1 {
				purpose = PurposeUpkeep
			}
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: 4, Limit: 1_100, Purpose: purpose})
		default:
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Ask, Quantity: 2, Limit: 900})
		}
	}, WithClearing("split", splitClearing{}))
	for tick, wantSpend := range []int64{4_000, 0} {
		tickInspecting(t, w, func() {
			if got := w.welfare.slots[1].inputSpend; got != wantSpend {
				t.Errorf("tick %d: input spend of agent 1 = %d, want %d", tick, got, wantSpend)
			}
			for _, id := range []int{2, 3} {
				if s := w.welfare.slots[id]; s.receipts != 2_000 || !s.traded {
					t.Errorf("tick %d: agent %d receipts = %d, traded = %v, want 2000 and true", tick, id, s.receipts, s.traded)
				}
			}
			for i, v := range w.markets.slotOf {
				if v != 0 {
					t.Errorf("tick %d: slot of agent %d = %d after P4.1, want 0 (cleared)", tick, i, v)
				}
			}
		})
	}
}

// TestStep_ABuyerFindsItsOwnEntryWhenAnotherRecordSitsFirst verifies the slot of D194
// with two bids of different purposes behind an ask at entry 0: each buyer is
// charged through its own record, not the first one's.
//
// Arithmetic: agent 1 asks 4 grain at 900 (entry 0); agents 2 and 3 bid 2 each at
// 1 100, the trades are at the midpoint 1 000, so 2 000 each. Agent 2's purpose is
// Input (input spend 2 000), agent 3's is Upkeep (input spend 0).
//
// Requirement: FR-AGT-057, FR-WEL-017, FR-WLD-058
func TestStep_ABuyerFindsItsOwnEntryWhenAnotherRecordSitsFirst(t *testing.T) {
	t.Parallel()
	w := newTrader(t, tradeConfig(3), func(v *AgentView, p *TradePlan) {
		switch v.Agent {
		case 1:
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Ask, Quantity: 4, Limit: 900})
		case 2:
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: 2, Limit: 1_100, Purpose: PurposeInput})
		case 3:
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: 2, Limit: 1_100, Purpose: PurposeUpkeep})
		}
	})
	tickInspecting(t, w, func() {
		if got := w.welfare.slots[2].inputSpend; got != 2_000 {
			t.Errorf("input spend of agent 2 (entry 1) = %d, want 2000", got)
		}
		if got := w.welfare.slots[3].inputSpend; got != 0 {
			t.Errorf("input spend of agent 3 (entry 2, Upkeep) = %d, want 0", got)
		}
	})
}

// TestStep_TheRecordOfAnIntentIsParallelToTheBook verifies D194's shape after P4.1:
// in every market the records are as many as the book's entries and sit at the same
// index, an ask of a household keeps its facts, and an estate's ask has the zero
// record.
//
// Requirement: FR-AGT-049, FR-WLD-047
func TestStep_TheRecordOfAnIntentIsParallelToTheBook(t *testing.T) {
	t.Parallel()
	w := newTrader(t, tradeConfig(3), func(v *AgentView, p *TradePlan) {
		switch v.Agent {
		case 1:
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: 1, Limit: 900, Target: 3, Reserve: 1, UnitCost: 700, Purpose: PurposeInput})
		case 2:
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Ask, Quantity: 1, Limit: 1_500, Target: 9, UnitCost: 600})
		}
	})
	// Give the estate of the region agent 3's goods, so it asks in every market.
	if _, _, err := w.estates.receive(w.ledger, w.agents[3].account, 0); err != nil {
		t.Fatalf("receive() error = %v, want nil", err)
	}
	tickInspecting(t, w, func() {
		for i, b := range w.markets.books {
			if len(w.markets.intents[i]) != len(b.entries) {
				t.Errorf("market %d: %d records for %d entries, want as many", i, len(w.markets.intents[i]), len(b.entries))
			}
		}
		gi := w.markets.index(0, grain)
		for j, e := range w.markets.books[gi].entries {
			rec := w.markets.intents[gi][j]
			var want intentRec
			switch e.agent {
			case 1:
				want = intentRec{purpose: PurposeInput, target: 3, reserve: 1, unitCost: 700}
			case 2:
				want = intentRec{target: 9, unitCost: 600}
			}
			if rec != want {
				t.Errorf("entry %d (agent %d, %v) record = %+v, want %+v", j, e.agent, e.side, rec, want)
			}
		}
		// Agents 1 and 2 first, in plan order, then the estate (agent 0): 6.5.1.
		if got := []AgentID{w.markets.books[gi].entries[0].agent, w.markets.books[gi].entries[1].agent, w.markets.books[gi].entries[2].agent}; !slices.Equal(got, []AgentID{1, 2, 0}) {
			t.Errorf("owners of the entries of the grain market = %v, want [1 2 0]: the agents' orders, then the estate's ask", got)
		}
	})
}

// TestStep_AnEstateSellsThroughTheMarketAndHasNoWelfareSlot verifies the estate
// as a seller: its lots are asked at P3.4 after the agents' orders, a bid that
// crosses buys them as a trade transfer into the estate's account (FR-EST-013), a
// recording Strategy is never passed an estate (FR-AGT-032, FR-EST-011), and the
// sale credits no welfare slot, because a pseudo-agent has none, though the buyer
// has a working tick.
//
// Arithmetic: agent 3's 10 grain go to the estate. The estate asks 10 grain in a
// named lot at its listed price, the first price of the reference price 1 000: 1 000
// (a lot that does not spoil or wear). Agent 1 bids 2 at 1 500 and the trade is at
// the midpoint (1 500 + 1 000) / 2 = 1 250, so 2 * 1 250 = 2 500 reaches the estate.
//
// Requirement: FR-EST-011, FR-EST-013, FR-AGT-032, FR-WLD-042, FR-WEL-017
func TestStep_AnEstateSellsThroughTheMarketAndHasNoWelfareSlot(t *testing.T) {
	t.Parallel()
	var seen []AgentID
	w := newTrader(t, tradeConfig(3), func(v *AgentView, p *TradePlan) {
		seen = append(seen, v.Agent)
		if v.Agent == 1 {
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: 2, Limit: 1_500})
		}
	})
	if _, _, err := w.estates.receive(w.ledger, w.agents[3].account, 0); err != nil {
		t.Fatalf("receive() error = %v, want nil", err)
	}
	poolBefore := int64(w.estates.pool(w.ledger, 0))
	tickInspecting(t, w, func() {
		if got := flowOf(w, ledger.Trade, ledger.Money); got != 2_500 {
			t.Errorf("trade money flow = %d, want 2500 (2 units at 1250)", got)
		}
		if got := int64(w.estates.pool(w.ledger, 0)); got != poolBefore+2_500 {
			t.Errorf("estate balance = %d, want %d: the sale is a trade transfer into the estate", got, poolBefore+2_500)
		}
		if s := w.welfare.slots[0]; s != (flows{}) {
			t.Errorf("welfare slot of the estate = %+v, want zero: a pseudo-agent has none", s)
		}
		if !w.welfare.slots[1].traded {
			t.Errorf("the buyer of the estate's lot has no working tick, want one")
		}
	})
	for _, id := range seen {
		if id == 0 {
			t.Errorf("a strategy was passed agent 0, the estate: a pseudo-agent has no strategy")
		}
	}
	if len(seen) == 0 {
		t.Errorf("the recording strategy was never called, want a call for each household")
	}
}

// TestStep_ARoleWithNoRecipesWorksByTrading verifies, in the recipe-less role of
// the shell configuration, that an agent whose order never fills has no working
// tick and an agent whose order fills has one (spec-welfare-and-wages.md 4.3,
// FR-WEL-017): markTraded is made from the settled trades, not from the placed
// orders.
//
// Arithmetic: the shell's farmers hold 10 grain and 10 000 minor units. Agent 1 bids
// 1 at 900 against agent 2's ask of 1 at 1 100 in the first case, which does not
// cross; at 1 200 it does. workingTick for a role with no recipes is the weight (1)
// when the agent traded and 0 when it did not.
//
// Guard: if the feed were made at P3.2 an order that never fills would count as
// work, and the ratio of an idle role would read as working.
//
// Requirement: FR-WEL-017
func TestStep_ARoleWithNoRecipesWorksByTrading(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		bid     Price
		working bool
	}{
		{"the orders do not cross", 900, false},
		{"the orders cross", 1_200, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := shellConfig()
			c.Groups[0].Households = 2
			c.Roles[0].Strategy = "script"
			c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 10}}
			w := newMill(t, c, WithStrategy("script", &tradeScript{trade: func(v *AgentView, p *TradePlan) {
				if v.Agent == 1 {
					p.Intents = append(p.Intents, Intent{Commodity: 0, Side: Bid, Quantity: 1, Limit: tt.bid})
				} else {
					p.Intents = append(p.Intents, Intent{Commodity: 0, Side: Ask, Quantity: 1, Limit: 1_100})
				}
			}}))
			if len(w.roles.rows[0].recipes) != 0 {
				t.Fatalf("the shell's role lists %d recipes, want none", len(w.roles.rows[0].recipes))
			}
			tickInspecting(t, w, func() {
				for _, id := range []int{1, 2} {
					k, ownUse, traded := w.welfare.workRead(id)
					if working, workHH, _ := workingTick(false, k, ownUse, traded, false, 1); working != tt.working || (workHH == 1) != tt.working {
						t.Errorf("agent %d: working = %v, workHH = %d, want working %v", id, working, workHH, tt.working)
					}
				}
			})
		})
	}
}

// faultyClearing returns one fixed invalid list for a flour market that has a bid
// and an ask, as a faulty implementation would (spec-clearing.md 8.2).
type faultyClearing struct{ list []Trade }

func (f faultyClearing) Clear(bids, asks []Order, _ *rand.Rand) []Trade {
	if len(bids) == 0 || len(asks) == 0 || bids[0].Commodity != flour {
		return nil
	}
	return f.list
}

// checkFailedStep checks what a failed P4.1 leaves (see
// TestStep_AnInvalidTradeListFailsTheStep): the TickEnded event, no market reported,
// the failed world and its tick, and the reservations of the failed and of the later
// market.
func checkFailedStep(t *testing.T, w *World, rec *evLog) {
	t.Helper()
	if ended := of[TickEnded](rec); len(ended) != 1 || !ended[0].Failed {
		t.Errorf("TickEnded events = %+v, want one with Failed true", ended)
	}
	if n := len(of[MarketCleared](rec)) + len(of[TradeExecuted](rec)); n != 0 {
		t.Errorf("MarketCleared and TradeExecuted events = %d, want 0: nothing of a failed market is settled or reported, and the later market is not cleared", n)
	}
	if !errors.Is(w.Err(), ErrInvalidTrade) || w.Tick() != 0 {
		t.Errorf("Err() = %v, Tick() = %d, want ErrInvalidTrade and 0", w.Err(), w.Tick())
	}
	if err := w.Step(); !errors.Is(err, ErrWorldFailed) {
		t.Errorf("second Step() error = %v, want ErrWorldFailed", err)
	}
	if _, r := balance(t, w, 1); r != 0 {
		t.Errorf("money earmarked by agent 1 = %d, want 0: the failed market's reservations are released", r)
	}
	if got := reservedGoods(t, w, 2, flour); got != 0 {
		t.Errorf("flour earmarked by agent 2 = %d, want 0 (the failed market)", got)
	}
	if got := reservedGoods(t, w, 2, grain); got != 1 {
		t.Errorf("grain earmarked by agent 2 = %d, want 1 (the market after the failed one is not cleared)", got)
	}
}

// TestStep_AnInvalidTradeListFailsTheStep verifies the failure path of P4.1
// (spec-world.md 6.6.1, D151, N-239): a Clearing that returns a trade with one
// agent as buyer and seller, a price outside the limits of its orders or a buyer
// with no bid makes Step return an error that matches ErrInvalidTrade and names the
// tick, region and commodity, TickEnded has Failed true, the world is failed and
// its tick does not advance, no market is reported cleared, the reservations of the
// failed market are released and the later market, which was not cleared, keeps its.
//
// Arithmetic: agent 1 bids 2 flour at 1 000 (2 000 earmarked) and asks 2 flour at
// 900; agent 2 asks 2 flour at 900 and 1 grain at 900. The flour market is first, so
// it fails; the grain market is never cleared, so agent 2's grain ask keeps its 1
// earmarked unit.
//
// Requirement: FR-WLD-005, FR-WLD-006, FR-WLD-012, FR-WLD-048, FR-CLR-031
func TestStep_AnInvalidTradeListFailsTheStep(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		list []Trade
	}{
		{"one agent as buyer and seller", []Trade{{Commodity: flour, Buyer: 1, Seller: 1, Price: 950, Quantity: 1}}},
		{"a price outside the limits", []Trade{{Commodity: flour, Buyer: 1, Seller: 2, Price: 800, Quantity: 1}}},
		{"a buyer with no bid", []Trade{{Commodity: flour, Buyer: 99, Seller: 2, Price: 950, Quantity: 1}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tradeConfig(2)
			c.World.Clearing = "faulty"
			rec := &evLog{}
			w := newTrader(t, c, func(v *AgentView, p *TradePlan) {
				if v.Agent == 1 {
					p.Intents = append(p.Intents,
						Intent{Commodity: flour, Side: Bid, Quantity: 2, Limit: 1_000})
				}
				if v.Agent == 2 {
					p.Intents = append(p.Intents,
						Intent{Commodity: flour, Side: Ask, Quantity: 2, Limit: 900},
						Intent{Commodity: grain, Side: Ask, Quantity: 1, Limit: 900})
				}
			}, WithClearing("faulty", faultyClearing{list: tt.list}), WithObserver(rec))
			rec.events = nil
			err := w.Step()
			if !errors.Is(err, ErrInvalidTrade) || !strings.Contains(err.Error(), "tick 0 region 0 commodity 0") {
				t.Fatalf("Step() error = %v, want ErrInvalidTrade naming tick 0 region 0 commodity 0", err)
			}
			checkFailedStep(t, w, rec)
		})
	}
}

// tradeBuyers builds the world of TestStep_AMarketIsClearedFromItsOwnStream for a
// seed, runs it to the end of P3.2 and clears its markets in ascending or in reverse
// order, and returns the buyers of the trades of the flour market and then of the
// grain market.
func tradeBuyers(t *testing.T, seed uint64, reverse bool) []AgentID {
	t.Helper()
	rec := &evLog{}
	w := newTrader(t, tradeConfig(3), func(v *AgentView, p *TradePlan) {
		for _, c := range []CommodityID{flour, grain} {
			if v.Agent == 1 {
				p.Intents = append(p.Intents, Intent{Commodity: c, Side: Ask, Quantity: 1, Limit: 900})
			} else {
				p.Intents = append(p.Intents, Intent{Commodity: c, Side: Bid, Quantity: 1, Limit: 1_000})
			}
		}
	}, WithSeed(seed), WithObserver(rec))
	rec.events = nil
	reserveWindow(t, w)
	if err := w.p32Trade(); err != nil {
		t.Fatalf("seed %d: p32Trade() error = %v, want nil", seed, err)
	}
	if reverse {
		for i := len(w.markets.books) - 1; i >= 0; i-- {
			if err := w.clearOne(i, w.Tick()); err != nil {
				t.Fatalf("seed %d: clearOne(%d) error = %v, want nil", seed, i, err)
			}
		}
	} else if err := w.p41Clear(); err != nil {
		t.Fatalf("seed %d: p41Clear() error = %v, want nil", seed, err)
	}
	var buyers []AgentID
	for _, c := range []CommodityID{flour, grain} {
		for _, e := range of[TradeExecuted](rec) {
			if e.Commodity == c {
				buyers = append(buyers, e.Buyer)
			}
		}
	}
	return buyers
}

// TestStep_AMarketIsClearedFromItsOwnStream verifies that the tie-break words of a
// market come from its own keyed stream, so clearing the markets in another order
// gives the same trades (spec-world.md 7.3, FR-WLD-069 half): two worlds of one seed
// are stepped to P3.4, one clears its markets in ascending order and one in reverse,
// and the winner of every tie is the same in both.
//
// Arithmetic: in each of the two markets agents 2 and 3 bid 1 unit at the same limit
// 1 000 and agent 1 asks 1 unit at 900, so one bid trades at (1 000 + 900) / 2 = 950
// and which one is the tie-break key's. Seeds 1 to 8 are tried.
//
// Requirement: FR-WLD-069, FR-CLR-004
func TestStep_AMarketIsClearedFromItsOwnStream(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 8; seed++ {
		var buyers [2][]AgentID
		for run, reverse := range []bool{false, true} {
			buyers[run] = tradeBuyers(t, seed, reverse)
		}
		if !slices.Equal(buyers[0], buyers[1]) || len(buyers[0]) != 2 {
			t.Errorf("seed %d: buyers clearing in ascending order = %v, in reverse = %v, want equal and one trade in each market", seed, buyers[0], buyers[1])
		}
	}
}

// TestStep_DiscardsAreKeptOrRefusedAtTheirBoundaries verifies, for one scripted plan
// of one agent, which discards the world keeps and which it refuses with a Debug
// line, at the edge of each rule: a quantity of 1 is the smallest allowed and 0 is
// refused; a commodity outside the configuration is refused at -1 and at the count
// of commodities; a discard of exactly what is left unearmarked is kept and one
// unit more is refused; units already kept for the same commodity are taken off
// what is free for the next discard, and a discard of another commodity is not
// affected by them.
//
// Arithmetic: agent 1 holds 10 flour and 10 grain and nothing is earmarked at the
// time of the discards. In plan order: flour 3 kept (free 10 -> 7); flour 8 refused
// (8 > 7); flour 7 kept (7 <= 7, free 0); grain 0 refused (quantity); commodity -1
// refused; commodity 2 refused (there are 2 commodities); grain 6 kept (free 10 -> 4);
// grain 5 refused (5 > 4); grain 4 kept (free 0). So 4 discards are kept and 5
// refused; the agent is left with 0 flour and 0 grain, and the disposal boundary
// holds 10 flour and 10 grain. The grain upkeep of the tick then finds none, which
// the test does not look at.
//
// Requirement: FR-AGT-057
// Guard: a discard of units the agent does not hold, or held twice over, would
// create a negative holding or fail the ledger move for a step the consumer cannot
// see.
func TestStep_DiscardsAreKeptOrRefusedAtTheirBoundaries(t *testing.T) {
	t.Parallel()
	lg := newCapture()
	w := newTrader(t, tradeConfig(2), func(v *AgentView, p *TradePlan) {
		if v.Agent != 1 {
			return
		}
		p.Discards = append(p.Discards,
			Discard{Commodity: flour, Quantity: 3}, Discard{Commodity: flour, Quantity: 8}, Discard{Commodity: flour, Quantity: 7},
			Discard{Commodity: grain, Quantity: 0}, Discard{Commodity: -1, Quantity: 1}, Discard{Commodity: 2, Quantity: 1},
			Discard{Commodity: grain, Quantity: 6}, Discard{Commodity: grain, Quantity: 5}, Discard{Commodity: grain, Quantity: 4})
	}, WithLogger(slog.New(lg)))
	tickInspecting(t, w, func() {
		for _, c := range []int{flour, grain} {
			held, reserved, err := w.ledger.Holding(w.agents[1].account, c)
			if err != nil || held != 0 || reserved != 0 {
				t.Errorf("Holding(agent 1, commodity %d) = (%d, %d, %v), want (0, 0, nil)", c, held, reserved, err)
			}
			sink, _ := w.ledger.Boundary(ledger.Disposal, c)
			if v, err := w.ledger.Account(sink); err != nil || v.Balance != 10 {
				t.Errorf("disposal balance of commodity %d = %+v, %v, want 10", c, v, err)
			}
		}
	})
	if refused := lg.take("discard refused"); len(refused) != 5 {
		t.Errorf("Debug lines \"discard refused\" = %d, want 5", len(refused))
	}
}

// TestMarketOf_RejectsWhatNoBookCanHold verifies the world's own rejection of an
// intent before the book sees it: a side that is neither bid nor ask is
// side_not_allowed, a commodity outside the configuration is unknown_commodity at
// -1 and at the count of commodities and not at the last valid one, and the
// rejection event names the region and the commodity as ordered (the book of the
// first market of the region carries it). A side error wins over an unknown
// commodity, and the valid commodities are placed.
//
// Arithmetic: the mill world has 2 commodities (flour 0, grain 1), so the valid
// IDs are 0 and 1; -1 and 2 are the two edges just outside. Agent 1 submits six
// intents in plan order: (side 7, flour) side_not_allowed; (bid, -1)
// unknown_commodity; (bid, 2) unknown_commodity; (side 7, 2) side_not_allowed;
// (bid, grain) and (bid, flour) placed, each a bid of 1 unit at 900 for which the
// agent has 900 money.
//
// Requirement: FR-CLR-008, FR-OBS-008
func TestMarketOf_RejectsWhatNoBookCanHold(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newTrader(t, tradeConfig(2), func(v *AgentView, p *TradePlan) {
		if v.Agent != 1 {
			return
		}
		p.Intents = append(p.Intents,
			Intent{Commodity: flour, Side: 7, Quantity: 1, Limit: 900},
			Intent{Commodity: -1, Side: Bid, Quantity: 1, Limit: 900},
			Intent{Commodity: 2, Side: Bid, Quantity: 1, Limit: 900},
			Intent{Commodity: 2, Side: 7, Quantity: 1, Limit: 900},
			Intent{Commodity: grain, Side: Bid, Quantity: 1, Limit: 900},
			Intent{Commodity: flour, Side: Bid, Quantity: 1, Limit: 900})
	}, WithObserver(rec))
	rec.events = nil
	step(t, w, 1)
	var got []string
	for _, e := range of[OrderRejected](rec) {
		got = append(got, fmt.Sprintf("%d/%d/%d/%s", e.Region, e.Commodity, e.Side, e.Reason))
	}
	want := []string{"0/0/7/side_not_allowed", "0/-1/0/unknown_commodity", "0/2/0/unknown_commodity", "0/2/7/side_not_allowed"}
	if !slices.Equal(got, want) {
		t.Errorf("OrderRejected (region/commodity/side/reason) = %q, want %q", got, want)
	}
	var placed []CommodityID
	for _, e := range of[OrderPlaced](rec) {
		if e.Agent == 1 {
			placed = append(placed, e.Commodity)
		}
	}
	if !slices.Equal(placed, []CommodityID{grain, flour}) {
		t.Errorf("OrderPlaced commodities of agent 1 = %v, want [grain flour] in plan order", placed)
	}
}

// TestStep_TheFirstHouseholdIsASellerWithAWelfareSlot verifies the edge of the
// pseudo-agent rule of the welfare feeds: the first household has the lowest ID
// that is not an estate (the count of regions), and as a seller it earns receipts
// and a working tick like any other, and as the buyer of a bid for an input its
// spend is charged.
//
// Arithmetic: with one region the estate is agent 0 and the first household agent
// 1. Agent 1 asks 2 grain at 900 and agent 2 bids 2 at 1 100 for an input: the
// midpoint is (1 100 + 900) / 2 = 1 000, so 2 000 moves, agent 1 receives 2 000 and
// agent 2 spends 2 000 on an input.
//
// Requirement: FR-WEL-017, FR-WLD-058
// Guard: a comparison of the seller against the count of regions that was off by one
// would drop the first household's receipts and leave its welfare ratio too low.
func TestStep_TheFirstHouseholdIsASellerWithAWelfareSlot(t *testing.T) {
	t.Parallel()
	w := newTrader(t, tradeConfig(2), func(v *AgentView, p *TradePlan) {
		if v.Agent == 2 {
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: 2, Limit: 1_100, Purpose: PurposeInput})
		} else {
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Ask, Quantity: 2, Limit: 900})
		}
	})
	tickInspecting(t, w, func() {
		if got := w.welfare.slots[1].receipts; got != 2_000 {
			t.Errorf("receipts of agent 1, the first household and the seller = %d, want 2000", got)
		}
		if got := w.welfare.slots[2].inputSpend; got != 2_000 {
			t.Errorf("input spend of agent 2, the buyer = %d, want 2000", got)
		}
		if !w.welfare.slots[1].traded || !w.welfare.slots[2].traded {
			t.Errorf("traded = %v and %v, want both true", w.welfare.slots[1].traded, w.welfare.slots[2].traded)
		}
	})
}
