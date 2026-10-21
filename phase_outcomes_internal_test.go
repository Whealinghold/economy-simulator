package economysim

import (
	"log/slog"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file tests phases 5 and 6 of the tick (phase_outcomes.go,
// plan-phase-1-11.md Phase 6) over scripted worlds. The commodity IDs are those
// of millConfig: flour 0 (reference price 3 000), grain 1 (1 000). Agent 0 is the
// estate of the one region, so the households are agents 1, 2, ... The log tests
// of the three Warns that need a scenario of a whole pack are Phase 10's; the
// runs and the lines are tested here directly.

// reviseScript is a Strategy that requests no job, trades as trade says and
// revises by lift: the belief centre plus lift. It keeps a copy of every outcome
// view it is given and the first word it takes from its own stream.
type reviseScript struct {
	trade func(v *AgentView, p *TradePlan)
	lift  Price
	seen  []OutcomeView
	words []uint64
}

func (*reviseScript) Work(*AgentView, *rand.Rand) JobRequest { return JobRequest{NoJob: true} }

func (s *reviseScript) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	if s.trade != nil {
		s.trade(v, p)
	}
}

func (s *reviseScript) Revise(o *OutcomeView, rng *rand.Rand) Belief {
	s.seen = append(s.seen, *o)
	s.words = append(s.words, rng.Uint64())
	return Belief{Centre: o.Belief.Centre + s.lift, Width: o.Belief.Width}
}

var _ Strategy = (*reviseScript)(nil)

// newScripted builds the world of c whose role follows the strategy s.
func newScripted(t *testing.T, c Config, s Strategy, opts ...Option) *World {
	t.Helper()
	return newMill(t, c, append([]Option{WithStrategy("script", s)}, opts...)...)
}

// keyOfOutcome returns the first two words of the stream of the outcome of an
// order, computed from a fresh stream: the world's word and the first word of the
// strategy's own (spec-world.md 7.3, D8).
func keyOfOutcome(seed uint64, tick int64, agent AgentID, commodity CommodityID) (first, second uint64) {
	s := newStream(seed)
	s.keyedPair(siteOutcome, tick, int64(agent), int64(commodity))
	return s.word(), s.word()
}

// TestP54_OutcomesAreDeliveredOnceInTheOrderOfTheSpec verifies P5.4 over a
// scripted world with an estate: one word per admitted order of a household in
// ascending (region, commodity) and admission order, none for the estate's
// asks; the view each strategy gets; the stored belief; and BeliefsUpdated.
//
// Arithmetic: commodity 0 is flour (3 000), commodity 1 grain (1 000). Agent 4
// gives its goods to the estate, which asks 10 flour at 3 000 and 10 grain at 1 000
// after the households. Agent 1 bids 3 grain at 800 and agent 2 asks 2 grain at 900:
// no ask is at or below a bid, so neither fills. Agent 2 asks 1 flour at 2 700 and
// agent 3 bids 1 flour at 3 300: they cross, 1 unit at the half-even midpoint
// (3 300 + 2 700) / 2 = 3 000, so both fill 1. The markets are cleared flour then
// grain and each lists the households in admission order (the estate last), so
// the outcomes come as (flour, 2), (flour, 3), (grain, 1), (grain, 2) and four
// words are taken. Stock is the units held after the trades less the intent's
// Reserve: agent 2 holds 10 - 1 = 9 flour, reserve 3, so 6; agent 3 holds 11 flour,
// reserve 2, so 9; agent 1 holds 10 grain, reserve 2, so 8; agent 2 holds 10 grain,
// no reserve, so 10. Revise lifts every centre by 10, from the seed price 3 000 to
// 3 010 for flour, inside the bounds. The flour market's two outcomes are both
// raised, none at a bound: Outcomes 2, Raised 2, mean = low = high = 3 010.
//
// Requirement: FR-WLD-049, FR-BEL-008, FR-BEL-026, FR-CLR-025, FR-OBS-022
func TestP54_OutcomesAreDeliveredOnceInTheOrderOfTheSpec(t *testing.T) {
	t.Parallel()
	s := &reviseScript{lift: 10, trade: func(v *AgentView, p *TradePlan) {
		switch v.Agent {
		case 1:
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: 3, Limit: 800, Reserve: 2, Target: 5})
		case 2:
			p.Intents = append(p.Intents,
				Intent{Commodity: flour, Side: Ask, Quantity: 1, Limit: 2_700, Reserve: 3, Target: 4, UnitCost: 2_000},
				Intent{Commodity: grain, Side: Ask, Quantity: 2, Limit: 900})
		case 3:
			p.Intents = append(p.Intents, Intent{Commodity: flour, Side: Bid, Quantity: 1, Limit: 3_300, Reserve: 2})
		}
	}}
	rec := &evLog{}
	w := newScripted(t, tradeConfig(4), s, WithObserver(rec), WithAudit())
	if _, _, err := w.estates.receive(w.ledger, w.agents[4].account, 0); err != nil {
		t.Fatalf("receive() error = %v, want nil", err)
	}
	rec.events = nil
	step(t, w, 1)
	type key struct {
		commodity CommodityID
		agent     AgentID
	}
	wantOrder := []key{{flour, 2}, {flour, 3}, {grain, 1}, {grain, 2}}
	if len(s.seen) != len(wantOrder) {
		t.Fatalf("Revise was called %d times, want %d: one for each admitted order of a household, none for the estate", len(s.seen), len(wantOrder))
	}
	for i, want := range wantOrder {
		o := s.seen[i]
		first, second := keyOfOutcome(w.state.seed, 0, want.agent, want.commodity)
		if (key{o.Commodity, o.Agent}) != want || o.Draw != first || s.words[i] != second {
			t.Errorf("outcome %d = (commodity %d, agent %d, draw %#x, own word %#x), want (%d, %d, %#x, %#x)",
				i, o.Commodity, o.Agent, o.Draw, s.words[i], want.commodity, want.agent, first, second)
		}
	}
	checkOutcomeViews(t, w, s.seen)
	flourEvent, grainEvent := beliefsOf(t, rec)
	open := 0
	for _, o := range s.seen[:2] {
		if beliefs.Gate(o.Draw, w.bp.AdjustProb) {
			open++
		}
	}
	wantFlour := BeliefsUpdated{
		Tick: 0, Region: 0, Commodity: flour, Outcomes: 2, Adjusted: int64(open), Raised: 2,
		MeanCentre: 3_010, LowCentre: 3_010, HighCentre: 3_010,
	}
	if flourEvent != wantFlour {
		t.Errorf("BeliefsUpdated of flour = %+v, want %+v", flourEvent, wantFlour)
	}
	if grainEvent.Outcomes != 2 || grainEvent.Region != 0 || grainEvent.Commodity != grain {
		t.Errorf("BeliefsUpdated of grain = %+v, want 2 outcomes", grainEvent)
	}
	if got := w.agents[3].beliefs[0]; got.Centre != 3_010 {
		t.Errorf("stored belief of agent 3 for flour = %+v, want the centre 3010 Revise returned", got)
	}
}

// beliefsOf returns the BeliefsUpdated of the flour and of the grain market of
// the one tick a recording saw, failing if they are not two, flour first.
func beliefsOf(t *testing.T, rec *evLog) (flourEvent, grainEvent BeliefsUpdated) {
	t.Helper()
	got := of[BeliefsUpdated](rec)
	if len(got) != 2 || got[0].Commodity != flour || got[1].Commodity != grain {
		t.Fatalf("BeliefsUpdated events = %+v, want one for flour then one for grain", got)
	}
	return got[0], got[1]
}

// checkOutcomeViews checks the fields of the four outcome views of
// TestP54_OutcomesAreDeliveredOnceInTheOrderOfTheSpec.
func checkOutcomeViews(t *testing.T, w *World, seen []OutcomeView) {
	t.Helper()
	bd := func(unitCost int64, commodity CommodityID) Bounds {
		b := beliefs.Limits(int64(w.cfg.Commodities[commodity].ReferencePrice), unitCost, w.bp)
		return Bounds{Floor: Price(b.Floor), Ceiling: Price(b.Ceiling)}
	}
	for i, want := range []struct {
		side           Side
		limit          Price
		qty, filled    Quantity
		stock          int64
		target         Quantity
		unitCost       Price
		bounds         Bounds
		price, age     int64
		beliefAfterRev Price
	}{
		{Ask, 2_700, 1, 1, 6, 4, 2_000, bd(2_000, flour), 3_000, 0, 3_010},
		{Bid, 3_300, 1, 1, 9, 0, 0, bd(0, flour), 3_000, 0, 3_010},
		{Bid, 800, 3, 0, 8, 5, 0, bd(0, grain), 1_000, 1, 1_010},
		{Ask, 900, 2, 0, 10, 0, 0, bd(0, grain), 1_000, 1, 1_010},
	} {
		o := seen[i]
		if o.Side != want.side || o.Limit != want.limit || o.Quantity != want.qty || o.Filled != want.filled ||
			o.Stock != want.stock || o.Target != want.target || o.UnitCost != want.unitCost || o.Bounds != want.bounds ||
			o.Market.Price != want.price || o.Market.Age != want.age {
			t.Errorf("outcome %d view = %+v, want side %v limit %d qty %d filled %d stock %d target %d unit cost %d bounds %+v price %d age %d",
				i, o, want.side, want.limit, want.qty, want.filled, want.stock, want.target, want.unitCost, want.bounds, want.price, want.age)
		}
		if o.Belief.Centre+10 != want.beliefAfterRev || o.Params != beliefTunablesOf(w.bp) {
			t.Errorf("outcome %d belief = %+v and params %+v, want the seed centre %d and the world's parameters", i, o.Belief, o.Params, want.beliefAfterRev-10)
		}
	}
}

// TestP54_AMarketWithNoOutcomeEmitsNothingAndKeepsItsRun verifies that a world
// whose households place no order has no outcome, no BeliefsUpdated and no
// word taken at P5.4, and that the run of the Warn of every market is advanced all
// the same (it stays 0).
//
// Arithmetic: no order, so Outcomes 0 for each market and NextRun resets to 0.
//
// Requirement: FR-WLD-049, FR-BEL-007, FR-OBS-094
func TestP54_AMarketWithNoOutcomeEmitsNothingAndKeepsItsRun(t *testing.T) {
	t.Parallel()
	s := &reviseScript{}
	rec := &evLog{}
	w := newScripted(t, tradeConfig(2), s, WithObserver(rec))
	w.markets.boundRun[0] = 7 // a run left by earlier ticks
	step(t, w, 1)
	if len(s.seen) != 0 || len(of[BeliefsUpdated](rec)) != 0 {
		t.Errorf("outcomes = %d and BeliefsUpdated = %d with no order, want none", len(s.seen), len(of[BeliefsUpdated](rec)))
	}
	if got := w.markets.boundRun[0]; got != 0 {
		t.Errorf("run of a market with no outcome = %d, want 0: a tick with no outcome resets it (N-230)", got)
	}
}

// TestP51_WindowIsPushedOnlyForAMarketThatTraded verifies the price loop of
// phase 5: a market publishes its seed price before any clearing, with age 0 and
// an empty window; a trade pushes the published price once; a tick with no trade
// pushes nothing and the age counts up; a market that never traded has an empty
// window.
//
// Arithmetic: agent 1 bids 1 flour and agent 2 asks 1 flour. Tick 0: 3 300 and
// 2 700 trade at 3 000; tick 1: 3 500 and 2 700 trade at 3 100; tick 2: no order.
// After tick 0 the window holds [3 000]; after tick 1 [3 000, 3 100], mean (3 000 +
// 3 100) / 2 = 3 050, low 3 000, high 3 100; after tick 2 it still holds two prices
// and the age of flour is 1, the price still 3 100. Grain never traded: its window
// is empty, its price the seed 1 000 and its age 1, 2, 3 after the ticks.
//
// Requirement: FR-WLD-058, FR-CLR-032, FR-BEL-039
func TestP51_WindowIsPushedOnlyForAMarketThatTraded(t *testing.T) {
	t.Parallel()
	bid := []Price{3_300, 3_500, 0}
	s := &reviseScript{trade: nil}
	var w *World
	s.trade = func(v *AgentView, p *TradePlan) {
		tick := w.Tick()
		if bid[tick] == 0 {
			return
		}
		if v.Agent == 1 {
			p.Intents = append(p.Intents, Intent{Commodity: flour, Side: Bid, Quantity: 1, Limit: bid[tick]})
		} else {
			p.Intents = append(p.Intents, Intent{Commodity: flour, Side: Ask, Quantity: 1, Limit: 2_700})
		}
	}
	w = newScripted(t, tradeConfig(2), s)
	fl, gr := w.markets.index(0, flour), w.markets.index(0, grain)
	if r := w.markets.reports[fl]; r.Price != 3_000 || r.Age != 0 || w.markets.windows[fl].Len() != 0 {
		t.Errorf("flour before any step = price %d age %d window %d, want the seed 3000, 0 and an empty window", r.Price, r.Age, w.markets.windows[fl].Len())
	}
	for tick, want := range []struct {
		price, age, flourLen, grainAge int64
	}{{3_000, 0, 1, 1}, {3_100, 0, 2, 2}, {3_100, 1, 2, 3}} {
		step(t, w, 1)
		r, g := w.markets.reports[fl], w.markets.reports[gr]
		if int64(r.Price) != want.price || r.Age != want.age || int64(w.markets.windows[fl].Len()) != want.flourLen ||
			int64(g.Price) != 1_000 || g.Age != want.grainAge || w.markets.windows[gr].Len() != 0 {
			t.Errorf("after tick %d: flour price %d age %d window %d, grain price %d age %d window %d; want %d, %d, %d, 1000, %d, 0",
				tick, r.Price, r.Age, w.markets.windows[fl].Len(), g.Price, g.Age, w.markets.windows[gr].Len(), want.price, want.age, want.flourLen, want.grainAge)
		}
	}
	mean, _ := w.markets.windows[fl].Mean()
	lo, _ := w.markets.windows[fl].Low()
	hi, _ := w.markets.windows[fl].High()
	if mean != 3_050 || lo != 3_000 || hi != 3_100 {
		t.Errorf("flour window mean, low, high = %d, %d, %d, want 3050, 3000, 3100", mean, lo, hi)
	}
}

// TestP53_AnUnsoldEstateLotIsMarkedDown verifies that P5.3 is called for each
// region: a lot of the estate that had an admitted ask and was not sold falls by
// the region's markdown.
//
// Arithmetic: agent 3's 10 grain go to the estate, which lists them at the
// reference price 1 000 and asks them. No household bids, so the ask is unsold and
// the price falls by trunc(1 000 * EstateMarkdown / 10 000), to 1 000 - 50 = 950
// at the default 500 basis points.
//
// Requirement: FR-EST-009, FR-WLD-042
func TestP53_AnUnsoldEstateLotIsMarkedDown(t *testing.T) {
	t.Parallel()
	w := newScripted(t, tradeConfig(3), &reviseScript{})
	if _, _, err := w.estates.receive(w.ledger, w.agents[3].account, 0); err != nil {
		t.Fatalf("receive() error = %v, want nil", err)
	}
	step(t, w, 1)
	md := w.cfg.Regions[0].EstateMarkdown
	var got []Price
	for _, l := range w.estates.listed[0] {
		if l.key.Commodity == grain {
			got = append(got, l.price)
		}
	}
	want := Price(1_000 - 1_000*md/10_000)
	if len(got) != 1 || got[0] != want {
		t.Errorf("listed price of the estate's grain after the tick = %v, want [%d] (1000 less the markdown of %d basis points)", got, want, md)
	}
}

// settledGrainTrade steps a world of three agents once in which agent 1 buys qty
// grain from agent 2 at 1 000 (limits 1 100 and 900, midpoint 1 000), with the
// purpose of the bid as given, and agent 3 does nothing.
func settledGrainTrade(t *testing.T, qty Quantity, purpose Purpose) *World {
	t.Helper()
	s := &reviseScript{trade: func(v *AgentView, p *TradePlan) {
		switch v.Agent {
		case 1:
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: qty, Limit: 1_100, Purpose: purpose})
		case 2:
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Ask, Quantity: qty, Limit: 900})
		}
	}}
	w := newScripted(t, tradeConfig(3), s, WithAudit())
	step(t, w, 1)
	return w
}

// recordsOf returns the welfare records an agent holds, oldest first.
func recordsOf(w *World, id AgentID) []welfare.Record { return w.agents[id].history.AppendRecords(nil) }

// TestP63_WelfareRecordOfAScriptedTick verifies the welfare push over a worked
// example: income from the receipts and the spend on inputs, the cost of the
// upkeep at the prices of the tick, and the working tick of a role with recipes.
//
// Arithmetic: agent 1 buys 2 grain at 1 000 for the purpose Input, so it pays
// 2 000 and its income is 0 - 2 000 = -2 000; agent 2 sells them and its income is
// the receipts 2 000; agent 3 has none. The upkeep of one household is 1 grain at
// the price the tick published for grain, 1 000 (the market traded, so the window is
// not empty), so every cost is 1 000. The role has a recipe and no one started a
// job, so nobody worked: Working false, WorkHH 0. Each ring holds one record.
//
// Requirement: FR-WEL-003, FR-WEL-017, FR-WLD-050
func TestP63_WelfareRecordOfAScriptedTick(t *testing.T) {
	t.Parallel()
	w := settledGrainTrade(t, 2, PurposeInput)
	for id, want := range map[AgentID]welfare.Record{
		1: {Income: -2_000, Cost: 1_000},
		2: {Income: 2_000, Cost: 1_000},
		3: {Income: 0, Cost: 1_000},
	} {
		if got := recordsOf(w, id); len(got) != 1 || got[0] != want {
			t.Errorf("records of agent %d = %+v, want [%+v]", id, got, want)
		}
	}
	for i := range w.welfare.slots {
		if w.welfare.slots[i] != (flows{}) {
			t.Errorf("welfare scratch of slot %d = %+v after the step, want zero: reset at P6.3", i, w.welfare.slots[i])
		}
	}
}

// TestP63_NothingClampsIncomeToKeepTheRatioAboveOne verifies, as a guard, that
// the world tops nobody up: an agent whose ratio stays far below 1 and one far
// above have a record income equal to what they received less what they spent on
// inputs, and their money moved only by the trade.
//
// Guard: if a future change clamped income to keep the ratio above 1, the buyer's
// income of -5 000 would read 0 or the cost, and its money would not be 5 000
// below the start.
//
// Arithmetic: agent 1 buys 5 grain at 1 000 for the purpose Input: 5 000 out, so
// its income is -5 000 and its ratio -5 000 / 1 000 = -5 times the upkeep. Agent 2
// sells them: income 5 000, ratio 5. Both start with 10 000 minor units, so the
// money after is 10 000 - 5 000 = 5 000 and 10 000 + 5 000 = 15 000, and agent 3 keeps
// 10 000. Upkeep is goods, so no money moves for it.
//
// Requirement: FR-WEL-011
func TestP63_NothingClampsIncomeToKeepTheRatioAboveOne(t *testing.T) {
	t.Parallel()
	w := settledGrainTrade(t, 5, PurposeInput)
	for id, want := range map[AgentID]struct{ income, money int64 }{
		1: {-5_000, 5_000}, 2: {5_000, 15_000}, 3: {0, 10_000},
	} {
		recs := recordsOf(w, id)
		if len(recs) != 1 || recs[0].Income != want.income {
			t.Errorf("record income of agent %d = %+v, want %d: no clamp, no top-up", id, recs, want.income)
		}
		if got, _ := balance(t, w, id); got != want.money {
			t.Errorf("money of agent %d = %d, want %d", id, got, want.money)
		}
	}
}

// jobScript is a Strategy that requests one execution of recipe 0 each time it is
// asked and never trades.
type jobScript struct{}

func (jobScript) Work(*AgentView, *rand.Rand) JobRequest     { return JobRequest{Recipe: 0, Executions: 1} }
func (jobScript) Trade(*AgentView, *rand.Rand, *TradePlan)   {}
func (jobScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

// TestP63_WearIsValuedAtThePriceOfTheTick verifies that the wear recorded at the
// start of a job is valued at P6.3, at the effective reference price of a tool
// that has never traded, and that a started job is a working tick.
//
// Arithmetic: each of the two agents starts one execution of mill (2 grain in, 3
// flour out) wearing the pan by 1 of durability 10 at the reference price 5 000, so
// the wear is ceil(1 * 5 000 / 10) = 500 and the income -500. The cost is 1 grain
// of upkeep at the grain price: grain never traded, so its reference price 1 000.
// One execution in a role with recipes is 1 working household-tick.
//
// Requirement: FR-WEL-003, FR-WEL-017
func TestP63_WearIsValuedAtThePriceOfTheTick(t *testing.T) {
	t.Parallel()
	c := panConfig()
	c.Roles[0].Strategy = "script"
	w := newScripted(t, c, jobScript{})
	step(t, w, 1)
	want := welfare.Record{Income: -500, Cost: 1_000, Working: true, WorkHH: 1}
	for _, id := range []AgentID{1, 2} {
		if got := recordsOf(w, id); len(got) != 1 || got[0] != want {
			t.Errorf("records of agent %d = %+v, want [%+v]", id, got, want)
		}
	}
}

// TestP63_ANeverTradedMarketIsValuedAtTheReferencePrice verifies the "never
// traded" rule of the valuation: a market with a seeded price that differs from
// the reference price and an empty window is valued at the reference price; once
// it trades, at the published price.
//
// Arithmetic: the warm start seeds grain at 1 500, the reference price is 1 000.
// Tick 0 has no trade, so the cost of the upkeep of 1 grain is 1 000 and not 1 500.
// In tick 1 agent 1 bids 1 800 and agent 2 asks 1 400 for 1 grain: the midpoint is
// (1 800 + 1 400) / 2 = 1 600, so the window holds [1 600] and the cost is 1 600.
//
// Requirement: FR-WEL-003
func TestP63_ANeverTradedMarketIsValuedAtTheReferencePrice(t *testing.T) {
	t.Parallel()
	c := shellConfig()
	c.Groups[0].Households = 2
	c.Roles[0].Strategy = "script"
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 10}}
	c.WarmStart = &WarmStart{Prices: []SeedPrice{{Region: "home", Commodity: "grain", Price: 1_500}}}
	var w *World
	s := &reviseScript{trade: func(v *AgentView, p *TradePlan) {
		if w.Tick() != 1 {
			return
		}
		if v.Agent == 1 {
			p.Intents = append(p.Intents, Intent{Commodity: 0, Side: Bid, Quantity: 1, Limit: 1_800})
		} else {
			p.Intents = append(p.Intents, Intent{Commodity: 0, Side: Ask, Quantity: 1, Limit: 1_400})
		}
	}}
	w = newScripted(t, c, s)
	if got := w.markets.reports[0].Price; got != 1_500 {
		t.Fatalf("seed price of grain = %d, want 1500", got)
	}
	step(t, w, 2)
	got := recordsOf(w, 1)
	if len(got) != 2 || got[0].Cost != 1_000 || got[1].Cost != 1_600 {
		t.Errorf("costs of agent 1 over two ticks = %+v, want 1000 then 1600", got)
	}
}

// outcomesWorld returns the world of the unfed tests: the shell's one role without
// recipes, 2 households of weight 1 each holding 2 grain, and upkeep of 1 grain.
func outcomesWorld(t *testing.T, opts ...Option) *World {
	t.Helper()
	c := shellConfig()
	c.Groups[0].Households = 2
	c.Roles[0].Strategy = "script"
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 2}}
	return newScripted(t, c, &reviseScript{}, opts...)
}

// TestP61_UpkeepUnfedHungerPrivationAndTheShortfall verifies the consumption of
// a scripted tick: the households eat what they hold up to the upkeep, an unfed
// household sets Unfed, Hunger and Privation, Privation rises 1 for each unfed tick
// and falls 1 (not below 0) for each fed tick, and UpkeepShortfall is emitted once
// for a (region, role) in each tick with an unfed household and in none else.
//
// Arithmetic: 2 households of weight 1 hold 2 grain each and eat 1 a tick. Ticks 0
// and 1: both fed, nothing emitted. Tick 2: both hold 0 and are unfed: Unfed 2,
// Agents 2, MaxHunger 1, Short 2 x 1 = 2 grain. Tick 3: Hunger 2 and Privation 2.
// Then agent 1 is given 2 grain. Tick 4: agent 1 eats (fed, Hunger 0, Privation
// 2 - 1 = 1) and agent 2 is unfed (Hunger 3, Privation 3): Unfed 1, Agents 1,
// MaxHunger 3, Short 1. Tick 5: agent 1 eats its last grain (Privation 0).
//
// Requirement: FR-WLD-050, FR-OBS-009, FR-AGT-075, FR-AGT-023
func TestP61_UpkeepUnfedHungerPrivationAndTheShortfall(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := outcomesWorld(t, WithObserver(rec), WithAudit())
	step(t, w, 2)
	if got := of[UpkeepShortfall](rec); len(got) != 0 {
		t.Fatalf("UpkeepShortfall events after two fed ticks = %+v, want none", got)
	}
	step(t, w, 2) // ticks 2 and 3
	from, ok := w.ledger.Boundary(ledger.Initial, 0)
	if !ok {
		t.Fatalf("no initial boundary account for grain")
	}
	if err := w.ledger.Move(from, w.agents[1].account, 0, 2, ledger.Initial); err != nil {
		t.Fatalf("Move() error = %v, want nil", err)
	}
	step(t, w, 2) // ticks 4 and 5
	want := []UpkeepShortfall{
		{Tick: 2, Households: 2, Unfed: 2, Agents: 2, MaxHunger: 1, Short: []ShortRow{{Commodity: 0, Units: 2}}},
		{Tick: 3, Households: 2, Unfed: 2, Agents: 2, MaxHunger: 2, Short: []ShortRow{{Commodity: 0, Units: 2}}},
		{Tick: 4, Households: 2, Unfed: 1, Agents: 1, MaxHunger: 3, Short: []ShortRow{{Commodity: 0, Units: 1}}},
		{Tick: 5, Households: 2, Unfed: 1, Agents: 1, MaxHunger: 4, Short: []ShortRow{{Commodity: 0, Units: 1}}},
	}
	got := of[UpkeepShortfall](rec)
	if len(got) != len(want) {
		t.Fatalf("UpkeepShortfall events = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i].Tick != want[i].Tick || got[i].Households != want[i].Households || got[i].Unfed != want[i].Unfed ||
			got[i].Agents != want[i].Agents || got[i].MaxHunger != want[i].MaxHunger || !slices.Equal(got[i].Short, want[i].Short) ||
			got[i].Region != 0 || got[i].Role != 0 {
			t.Errorf("UpkeepShortfall #%d = %+v, want %+v", i, got[i], want[i])
		}
	}
	a1, a2 := w.agents[1], w.agents[2]
	if a1.unfed != 0 || a1.hunger != 0 || a1.privation != 0 || a2.unfed != 1 || a2.hunger != 4 || a2.privation != 4 {
		t.Errorf("after tick 5 agent 1 = (unfed %d, hunger %d, privation %d) and agent 2 = (%d, %d, %d), want (0, 0, 0) and (1, 4, 4)",
			a1.unfed, a1.hunger, a1.privation, a2.unfed, a2.hunger, a2.privation)
	}
}

// TestP61_PrivationFallsOneForEachFedTick verifies the falling half of the leaky
// count (FR-AGT-075): after two unfed ticks Privation is 2, and each tick the
// agent is fed it falls by 1 to 0 and stays there.
//
// Arithmetic: 0 grain held at tick 0 and 1 (Privation 1, 2); 3 grain given; ticks
// 2, 3, 4 are fed: Privation 1, 0, 0 (max(0, 0 - 1) = 0).
//
// Requirement: FR-AGT-075
func TestP61_PrivationFallsOneForEachFedTick(t *testing.T) {
	t.Parallel()
	c := shellConfig()
	c.Groups[0].Households = 2
	c.Roles[0].Strategy = "script"
	c.Roles[0].StartStock = nil
	w := newScripted(t, c, &reviseScript{})
	step(t, w, 2)
	from, _ := w.ledger.Boundary(ledger.Initial, 0)
	if err := w.ledger.Move(from, w.agents[1].account, 0, 3, ledger.Initial); err != nil {
		t.Fatalf("Move() error = %v, want nil", err)
	}
	var got []int64
	for range 3 {
		step(t, w, 1)
		got = append(got, w.agents[1].privation)
	}
	if !slices.Equal(got, []int64{1, 0, 0}) || w.agents[2].privation != 5 {
		t.Errorf("privation of agent 1 over three fed ticks = %v, want [1 0 0] (after 2 at tick 1), and agent 2 still unfed at 5 (got %d)", got, w.agents[2].privation)
	}
}

// warnLines runs a world of the logger and returns the lines of one message.
func warnWorld(t *testing.T, c Config, s Strategy) (*World, capture) {
	t.Helper()
	lg := newCapture()
	w := newScripted(t, c, s, WithLogger(slog.New(lg)))
	return w, lg
}

// TestP64_HouseholdsUnfedWarnsAtTheTenthTickAndEveryTenthAfter verifies the run of
// the Warn "households unfed" (pattern B): it fires when the run reaches 10 and at
// every multiple of 10, per (region, role), with the households and the unfed.
//
// Arithmetic: 2 households with nothing to eat are unfed from tick 0, so the run
// is t + 1 at tick t and reaches 10 at tick 9 and 20 at tick 19: two lines in 20
// ticks, each households 2 and unfed 2.
//
// Requirement: FR-AGT-017, FR-OBS-070, FR-OBS-072
func TestP64_HouseholdsUnfedWarnsAtTheTenthTickAndEveryTenthAfter(t *testing.T) {
	t.Parallel()
	c := shellConfig()
	c.Groups[0].Households = 2
	c.Roles[0].Strategy = "script"
	c.Roles[0].StartStock = nil
	c.World.ExitShare = 0 // phase 7 must not remove the starving agents: the run is what is measured
	w, lg := warnWorld(t, c, &reviseScript{})
	var ticks []int64
	for range 20 {
		step(t, w, 1)
		for _, l := range lg.take("households unfed") {
			if l.level != slog.LevelWarn || l.attrs["region"] != 0 || l.attrs["role"] != 0 || l.attrs["households"] != 2 || l.attrs["unfed"] != 2 {
				t.Errorf("Warn at tick %d = %+v, want Warn region 0 role 0 households 2 unfed 2", w.Tick()-1, l)
			}
			ticks = append(ticks, w.Tick()-1)
		}
	}
	if !slices.Equal(ticks, []int64{9, 19}) {
		t.Errorf("ticks of the Warn \"households unfed\" = %v, want [9 19]", ticks)
	}
}

// TestP63_WelfareRecordClampedWarnsOnTheFirstTickAndEveryTenth verifies the run of
// the Warn "welfare record clamped" (pattern A): it fires on the first tick of a
// run and at every multiple of 10, with the households whose ring holds a field at
// the clamp.
//
// Arithmetic: a record with income MaxInt64 pushed into agent 1's ring is clamped
// to MaxInt64 / MaxWelfareWindow and stays in the ring for the window, so agent 1
// (weight 1) is at the clamp in every tick: the run is t + 1, and the lines are at
// run 1 (tick 0) and run 10 (tick 9), each with households 1.
//
// Requirement: FR-WEL-042, FR-OBS-070, FR-OBS-072
func TestP63_WelfareRecordClampedWarnsOnTheFirstTickAndEveryTenth(t *testing.T) {
	t.Parallel()
	c := shellConfig()
	c.Groups[0].Households = 2
	c.Roles[0].Strategy = "script"
	w, lg := warnWorld(t, c, &reviseScript{})
	if !w.agents[1].history.Push(welfare.Record{Income: math.MaxInt64}) {
		t.Fatalf("Push(MaxInt64 income) reported no clamp, want one")
	}
	var ticks []int64
	for range 10 {
		step(t, w, 1)
		for _, l := range lg.take("welfare record clamped") {
			if l.level != slog.LevelWarn || l.attrs["region"] != 0 || l.attrs["role"] != 0 || l.attrs["households"] != 1 {
				t.Errorf("Warn at tick %d = %+v, want Warn region 0 role 0 households 1", w.Tick()-1, l)
			}
			ticks = append(ticks, w.Tick()-1)
		}
	}
	if !slices.Equal(ticks, []int64{0, 9}) {
		t.Errorf("ticks of the Warn \"welfare record clamped\" = %v, want [0 9]", ticks)
	}
}

// TestBoundWarn_FiresAtTheTenthPinnedTickWithTheBoundAndTheCounts verifies the
// Warn "price beliefs at bound" of a market from a scripted tally: more than half
// of the outcomes on a bound for 10 consecutive ticks fires it at the tenth tick,
// naming the bound with more beliefs; a tick with no outcome resets the run.
//
// Arithmetic: 3 outcomes, 2 on the ceiling: 2 > 3 / 2 = 1, so the tick is pinned;
// the run counts 1 to 10 and the Warn at 10 names the ceiling, agents 2, outcomes 3.
// A tick with 0 outcomes then resets the run to 0.
//
// Requirement: FR-BEL-031, FR-OBS-070
func TestBoundWarn_FiresAtTheTenthPinnedTickWithTheBoundAndTheCounts(t *testing.T) {
	t.Parallel()
	w, lg := warnWorld(t, tradeConfig(2), &reviseScript{})
	var pinned beliefs.Tally
	bd := beliefs.Bounds{Floor: 500, Ceiling: 2_000}
	for _, centre := range []int64{2_000, 2_000, 1_000} {
		pinned.Add(beliefs.Change{After: beliefs.Belief{Centre: centre}, Bounds: bd})
	}
	for i := 1; i <= 10; i++ {
		w.boundWarn(1, 0, grain, &pinned)
		got := lg.take("price beliefs at bound")
		if (i == 10) != (len(got) == 1) {
			t.Fatalf("tick %d of a pinned run: %d lines, want one only at the tenth", i, len(got))
		}
		if i == 10 {
			l := got[0]
			if l.level != slog.LevelWarn || l.attrs["region"] != 0 || l.attrs["commodity"] != grain || l.attrs["agents"] != 2 || l.attrs["outcomes"] != 3 {
				t.Errorf("Warn = %+v, want Warn region 0 commodity grain agents 2 outcomes 3", l)
			}
		}
	}
	if w.markets.boundRun[1] != 10 {
		t.Errorf("run after 10 pinned ticks = %d, want 10", w.markets.boundRun[1])
	}
	w.boundWarn(1, 0, grain, &beliefs.Tally{})
	if w.markets.boundRun[1] != 0 {
		t.Errorf("run after a tick with no outcome = %d, want 0", w.markets.boundRun[1])
	}
}

// TestP12Anchors_WageFallsBackToTheSeedWageWhenThePooledWageIsUnknown verifies
// FR-VIA-052 (not latched): the cost anchor reads the starting wage of the warm
// start while the pooled wage of the region is unknown, the pooled wage once the
// rings give one, and the starting wage again when it is unknown again. The ring
// state is scripted through the unexported scratch.
//
// Arithmetic: mill takes 2 grain at 1 000 and 1 wage tick and makes 3 flour, so the
// anchor is ceil((2 000 + wage) / 3): the starting wage 100 gives ceil(2 100 / 3) =
// 700; the pooled wage (400 + 400) / (1 + 1) = 400 gives ceil(2 400 / 3) = 800;
// with the rings emptied it is 700 again.
//
// Requirement: FR-VIA-052, FR-PRO-048
func TestP12Anchors_WageFallsBackToTheSeedWageWhenThePooledWageIsUnknown(t *testing.T) {
	t.Parallel()
	c := millConfig(1, 2)
	c.WarmStart = &WarmStart{Wages: []SeedWage{{Region: "home", Wage: 100}}}
	w := newMill(t, c)
	anchor := func() int64 {
		t.Helper()
		if err := w.beginStep(); err != nil {
			t.Fatalf("beginStep() error = %v, want nil", err)
		}
		if err := w.p11Moment(); err != nil {
			t.Fatalf("p11Moment() error = %v, want nil", err)
		}
		w.p12Anchors()
		got := w.prod.unitCost(0, flour)
		if err := w.endStep(nil); err != nil {
			t.Fatalf("endStep() error = %v, want nil", err)
		}
		return got
	}
	if got := anchor(); got != 700 {
		t.Errorf("anchor with no pooled wage = %d, want 700 (the starting wage 100)", got)
	}
	for _, id := range []AgentID{1, 2} {
		w.agents[id].history.Push(welfare.Record{Income: 400, Working: true, WorkHH: 1})
	}
	if got := anchor(); got != 800 {
		t.Errorf("anchor with the pooled wage 400 = %d, want 800", got)
	}
	for _, id := range []AgentID{1, 2} {
		w.agents[id].history = newHistory(w.roles.row(w.agents[id].role))
	}
	if got := anchor(); got != 700 {
		t.Errorf("anchor with the rings emptied = %d, want 700: the starting wage again, not latched", got)
	}
}

// TestP54_DebugLinesCarryTheirKeys verifies the Debug lines of the two positions
// that write one for each agent: "belief updated" for each outcome (region,
// commodity, agent, previous, price) and "upkeep unmet" for each agent short of
// a commodity (agent, commodity, quantity).
//
// Arithmetic: agent 1 bids 1 flour at 3 300 and agent 2 asks it at 2 700: both
// outcomes, each from the seed centre 3 000 to 3 010 (lift 10). In the shell with
// nothing to eat two households are each 1 grain short of the upkeep of 1.
//
// Requirement: FR-BEL-032, FR-AGT-014
func TestP54_DebugLinesCarryTheirKeys(t *testing.T) {
	t.Parallel()
	s := &reviseScript{lift: 10, trade: func(v *AgentView, p *TradePlan) {
		if v.Agent == 1 {
			p.Intents = append(p.Intents, Intent{Commodity: flour, Side: Bid, Quantity: 1, Limit: 3_300})
		} else {
			p.Intents = append(p.Intents, Intent{Commodity: flour, Side: Ask, Quantity: 1, Limit: 2_700})
		}
	}}
	w, lg := warnWorld(t, tradeConfig(2), s)
	step(t, w, 1)
	got := lg.take("belief updated")
	if len(got) != 2 {
		t.Fatalf("\"belief updated\" lines = %d, want 2", len(got))
	}
	for i, l := range got {
		if l.level != slog.LevelDebug || l.attrs["region"] != 0 || l.attrs["commodity"] != flour || l.attrs["agent"] != int64(i+1) ||
			l.attrs["previous"] != 3_000 || l.attrs["price"] != 3_010 {
			t.Errorf("line %d = %+v, want Debug region 0 commodity 0 agent %d previous 3000 price 3010", i, l, i+1)
		}
	}
	c := shellConfig()
	c.Groups[0].Households = 2
	c.Roles[0].Strategy = "script"
	c.Roles[0].StartStock = nil
	w, lg = warnWorld(t, c, &reviseScript{})
	step(t, w, 1)
	got = lg.take("upkeep unmet")
	if len(got) != 2 {
		t.Fatalf("\"upkeep unmet\" lines = %d, want 2", len(got))
	}
	for i, l := range got {
		if l.level != slog.LevelDebug || l.attrs["agent"] != int64(i+1) || l.attrs["commodity"] != 0 || l.attrs["quantity"] != 1 {
			t.Errorf("line %d = %+v, want Debug agent %d commodity 0 quantity 1", i, l, i+1)
		}
	}
}

// TestP61_UpkeepShortfallIsCountedPerRegionAndRole verifies the cells of P6.1 and
// P6.4 on the two-region world of removalConfig when no miller holds any grain,
// the upkeep: one UpkeepShortfall for each (region, role) that has an agent, in
// ascending region and then role order, each with its own households, unfed
// households, agents, hunger and units short, and none for the weaver-less
// region's empty cell.
//
// Arithmetic: the upkeep is 1 grain per household per tick and nobody holds grain
// (the weaver's start stock is empty too), so every household goes unfed. Home
// miller: agents 2 (weight 2) and 3 (weight 1), households 3, unfed 3, agents 2,
// short 3 grain. Home weaver: agent 4, households 1, unfed 1, short 1. Yonder
// miller: agent 5 (weight 2), households 2, unfed 2, agents 1, short 2. Yonder
// weaver: no agent, no event. After the first tick every agent's hunger is 1.
//
// Requirement: FR-OBS-009, FR-WLD-058, FR-AGT-075
// Guard: the cell of a (region, role) is region * roles + role; a swapped or
// scaled index would put one region's shortfall in another's row.
func TestP61_UpkeepShortfallIsCountedPerRegionAndRole(t *testing.T) {
	t.Parallel()
	c := removalConfig()
	c.Roles[0].StartStock = nil
	rec := &evLog{}
	w := newMill(t, c, WithObserver(rec))
	rec.events = nil
	step(t, w, 1)
	want := []UpkeepShortfall{
		{Tick: 0, Region: 0, Role: 0, Households: 3, Unfed: 3, Agents: 2, MaxHunger: 1, Short: []ShortRow{{Commodity: grain, Units: 3}}},
		{Tick: 0, Region: 0, Role: 1, Households: 1, Unfed: 1, Agents: 1, MaxHunger: 1, Short: []ShortRow{{Commodity: grain, Units: 1}}},
		{Tick: 0, Region: 1, Role: 0, Households: 2, Unfed: 2, Agents: 1, MaxHunger: 1, Short: []ShortRow{{Commodity: grain, Units: 2}}},
	}
	if got := of[UpkeepShortfall](rec); !reflect.DeepEqual(got, want) {
		t.Errorf("UpkeepShortfall events = %+v, want %+v", got, want)
	}
}

// TestP63_ClampedHouseholdsAreCountedPerRegionAndRole verifies that the Warn
// "welfare record clamped" reports, for each (region, role) with households whose
// ring holds a field at the clamp, the households of that cell and no other, on the
// first tick of the run.
//
// Arithmetic: a record with an income of MaxInt64 is pushed to the ring of agent 4
// (home weaver, weight 1) and agent 5 (yonder miller, weight 2) before the step, so
// the ring is at its clamp at P6.3. The lines come in cell order: home weaver
// (region 0, role 1, 1 household), then yonder miller (region 1, role 0, 2
// households). The millers of home are not clamped and have no line.
//
// Requirement: FR-WEL-042, FR-OBS-072
func TestP63_ClampedHouseholdsAreCountedPerRegionAndRole(t *testing.T) {
	t.Parallel()
	lg := newCapture()
	w := newMill(t, removalConfig(), WithLogger(slog.New(lg)))
	for _, id := range []AgentID{4, 5} {
		w.agents[id].history.Push(welfare.Record{Income: math.MaxInt64, Cost: 1, Working: true, WorkHH: 1})
	}
	step(t, w, 1)
	var got [][3]int64
	for _, l := range lg.take("welfare record clamped") {
		got = append(got, [3]int64{l.attrs["region"], l.attrs["role"], l.attrs["households"]})
	}
	if want := [][3]int64{{0, 1, 1}, {1, 0, 2}}; !reflect.DeepEqual(got, want) {
		t.Errorf("\"welfare record clamped\" lines (region, role, households) = %v, want %v", got, want)
	}
}

// TestBoundWarn_NamesTheBoundWithMoreBeliefs verifies the bound attribute of the
// Warn "price beliefs at bound": "ceiling" when more of the pinned beliefs sit on
// the ceiling and "floor" when more sit on the floor, with the region and commodity
// of the market, at the tenth pinned tick.
//
// Arithmetic: bounds are 500 to 2 000. Three outcomes, two on the ceiling (centre
// 2 000) and one in range, are pinned (2 > 3 / 2) and name the ceiling; the mirror,
// two on the floor (centre 500), names the floor. The market of the call is
// (region 0, grain).
//
// Requirement: FR-BEL-031, FR-OBS-070
func TestBoundWarn_NamesTheBoundWithMoreBeliefs(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		centres []int64
		want    string
	}{
		{"mostly on the ceiling", []int64{2_000, 2_000, 1_000}, "ceiling"},
		{"mostly on the floor", []int64{500, 500, 1_000}, "floor"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, lg := warnWorld(t, tradeConfig(2), &reviseScript{})
			var pinned beliefs.Tally
			bd := beliefs.Bounds{Floor: 500, Ceiling: 2_000}
			for _, centre := range tt.centres {
				pinned.Add(beliefs.Change{After: beliefs.Belief{Centre: centre}, Bounds: bd})
			}
			for range 10 {
				w.boundWarn(1, 0, grain, &pinned)
			}
			got := lg.take("price beliefs at bound")
			if len(got) != 1 || got[0].strs["bound"] != tt.want || got[0].attrs["region"] != 0 || got[0].attrs["commodity"] != grain {
				t.Errorf("Warn lines = %+v, want one naming the %s for region 0 commodity grain", got, tt.want)
			}
		})
	}
}
