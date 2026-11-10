package strategy_test

import (
	"math/rand/v2"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// This file is the fixture of the merchant tests: views built by hand, as a
// consumer would, after the two worked examples of spec-merchants.md 19. A view is
// plain data and the strategy keeps no state, so Trade can be called with any.

// The commodities, regions and modes of the two examples, by ID.
const (
	// Example A: a river valley, the hired mode barge.
	fuel economysim.CommodityID = 0
	haul economysim.CommodityID = 1
	// Example B: a mining colony, the self-carried mode skiff.
	cell   economysim.CommodityID = 0
	ration economysim.CommodityID = 1
	skiff  economysim.CommodityID = 2

	home   economysim.RegionID = 0
	remote economysim.RegionID = 1
	barge  economysim.ModeID   = 0
)

// tunables are the household and belief tunables of the views: a cover of 3 ticks,
// reserve ticks as set by the caller, and belief widths free to the ceiling.
func tunables(reserveTicks int64) economysim.Tunables {
	return economysim.Tunables{
		CoverTicks: 3, InputJobs: 1, OutputCoverTicks: 1, FavourFloor: 2000, ReserveTicks: reserveTicks,
		SpendMaxShare: 5000, SpendHalfTicks: 10, RestBand: 500, ProduceHaltRatio: 30000,
		Beliefs: economysim.BeliefTunables{WidthMin: 0, WidthMax: 5000},
	}
}

// row is a CommodityView of the examples: stock, the belief centre and width, the
// belief ceiling and the local price.
func row(c economysim.CommodityID, stock, centre, width, ceiling, price int64) economysim.CommodityView {
	return economysim.CommodityView{
		Commodity: c, Stock: stock, Ref: economysim.Price(price),
		Belief:   economysim.Belief{Centre: economysim.Price(centre), Width: width},
		Bounds:   economysim.Bounds{Floor: 1, Ceiling: economysim.Price(ceiling)},
		Market:   economysim.MarketReport{Price: price},
		Expiring: 0,
	}
}

// valley is the view of agent 32 of example A (spec-merchants.md 19.1): a broker in
// the home region with no upkeep, 100 000 money, the barge (a carrying unit of
// 1 200 mass and 900 bulk, hauled at 6 haul units a unit over a trip of 2 ticks),
// fuel (mass 2, bulk 6) at 200 and haul units at 700, and the rumour about the
// remote region: price 310, mean 305, source age 0. The fee share of agent 32 is
// 6 385. It holds nothing, so the trip is prepared, not sent.
func valley() *economysim.AgentView {
	return &economysim.AgentView{
		Agent: 32, Region: home, Weight: 1, Money: 100_000, StartsIn: 1,
		HasRatio: true, Ratio: 10_000, TargetRatio: 15_000, Window: 10,
		Tunables:    tunables(3),
		Commodities: []economysim.CommodityView{row(fuel, 0, 200, 1000, 1000, 200), row(haul, 0, 700, 1000, 1000, 700)},
		Rumours:     []economysim.Rumour{{Region: remote, Commodity: fuel, Price: 310, Mean: 305, Lag: 3}},
		Routes:      []economysim.RouteRow{{To: remote, Mode: barge, Has: true, Ticks: 2, HaulPerUnit: 6}},
		HasMerchant: true,
		Merchant: economysim.MerchantView{
			Params:   block(),
			FeeShare: 5000,
			Modes:    []economysim.MerchantMode{{Mode: barge, Hired: true, UnitMass: 1200, UnitBulk: 900, Vehicle: economysim.NoCommodity, Haul: haul, HaulSpan: 10}},
			Goods: []economysim.MerchantGood{
				{Commodity: fuel, Mass: 2, Bulk: 6, OldestAge: -1},
				{Commodity: haul, OldestAge: -1},
			},
			Cargo: []economysim.CommodityID{fuel},
		},
	}
}

// block is the merchant block of example A by value (the modes are in the view's
// Modes): a lot of 1 carrying unit, a spread of 2 500, a rumour source age of 2,
// a time cost of 10 000, a cut of 500, and a take of 150 units at a discount of
// 1 000 that is off; a test of the takes sets Consigns.
func block() economysim.Merchant {
	b := economysim.DefaultMerchant()
	b.Modes, b.TakeUnits = nil, 150
	return b
}

// words is a source of fixed pseudo-random words that counts how many were taken,
// so a test can compare the draws of two calls.
type words struct {
	pcg *rand.PCG
	n   int
}

func newWords() *words { return &words{pcg: rand.NewPCG(7, 11)} }

func (w *words) Uint64() uint64 { w.n++; return w.pcg.Uint64() }

// trade calls Trade of the strategy on the view with the seeded source and returns
// the plan and the number of words taken.
func trade(t testing.TB, s economysim.Strategy, v *economysim.AgentView) (economysim.TradePlan, int) {
	t.Helper()
	src := newWords()
	var p economysim.TradePlan
	s.Trade(v, rand.New(src), &p)
	return p, src.n
}

// merchantStrategy is the strategy under test.
var merchantStrategy = strategy.Merchant{}

// colony is the view of agent 18 of example B (spec-merchants.md 19.2): a trader of
// weight 1 in the home region (the hub) with 5 905 money, no reserve, a ration of
// upkeep a tick, the self-carried skiff (a carrying unit of 2 000 mass and 1 500
// bulk, a vehicle of durability 400 at 4 000 that wears 5 a tick, one cell a tick
// of feed), the vehicle already held, and the rumour about the remote region (the
// pit): price 80, mean 78. The trip takes 3 ticks. The fee share of agent 18 is
// 3 123. Rations cost 56 and cells 30 here; the belief ceiling of a cell is 35 and
// the dearest draw for a ration is 58, so the money affords (5 905 - 3 * 35) / 58 =
// 100 rations.
func colony() *economysim.AgentView {
	skiffMode := economysim.MerchantMode{Mode: 0, UnitMass: 2000, UnitBulk: 1500, Crew: 1, Vehicle: skiff, WearPerTick: 5,
		Haul: economysim.NoCommodity, Feed: []economysim.Line{{Commodity: cell, Quantity: 1}}}
	return &economysim.AgentView{
		Agent: 18, Region: home, Weight: 1, Money: 5_905, StartsIn: 1,
		HasRatio: true, Ratio: 10_000, TargetRatio: 15_000, Window: 10,
		Tunables: tunables(0),
		Upkeep:   []economysim.UpkeepRow{{Commodity: ration, Quantity: 1}},
		Commodities: []economysim.CommodityView{
			row(cell, 0, 30, 1000, 35, 30), row(ration, 0, 56, 400, 1000, 56), row(skiff, 1, 4000, 0, 5000, 4000),
		},
		Rumours:     []economysim.Rumour{{Region: remote, Commodity: ration, Price: 80, Mean: 78, Lag: 4}},
		Routes:      []economysim.RouteRow{{To: remote, Mode: 0, Has: true, Ticks: 3}},
		HasMerchant: true,
		Merchant: economysim.MerchantView{
			Params: block(), FeeShare: 5000, Modes: []economysim.MerchantMode{skiffMode},
			Goods: []economysim.MerchantGood{
				{Commodity: cell, OldestAge: -1}, {Commodity: ration, Mass: 1, Bulk: 1, OldestAge: -1},
				{Commodity: skiff, Durability: 400, OldestAge: -1},
			},
			Cargo: []economysim.CommodityID{ration},
		},
	}
}
