package indicator_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// BenchmarkRowsOf reports the cost of building the household rows of money,
// wealth and a welfare ratio from 5 000 agent states, the size of the first
// consumer's world (spec-indicators.md 21 question 1). It is reported, not
// asserted (testing.md 8.2).
func BenchmarkRowsOf(b *testing.B) {
	p, err := indicator.NewPrices([32]byte{}, 0, 0, 1, 4, []economysim.Price{10, 25, 6, 5})
	if err != nil {
		b.Fatalf("NewPrices() error = %v, want nil", err)
	}
	st := make([]economysim.AgentState, 5000)
	for i := range st {
		st[i] = economysim.AgentState{Weight: 1, Money: economysim.Money(100 + i), Ratio: int64(8000 + i), HasRatio: true,
			Holdings: []economysim.HoldingState{{Commodity: 0, Quantity: 3}, {Commodity: 1, Quantity: 1}, {Commodity: 2, Quantity: 2}}}
	}
	b.ReportAllocs()
	for b.Loop() {
		for _, col := range []indicator.Column{indicator.ColMoney, indicator.ColWealth, indicator.ColWelfareRatio} {
			if _, err := indicator.RowsOf(st, col, indicator.RowOpt{Prices: p}); err != nil {
				b.Fatalf("RowsOf() error = %v, want nil", err)
			}
		}
	}
}

// BenchmarkTallyObserve reports the cost of folding the events of one tick of an
// open window, for rows that already exist (spec-indicators.md NFR-IND-002).
func BenchmarkTallyObserve(b *testing.B) {
	tl := indicator.NewTally()
	if err := tl.Open(valleySnap(0, [4]int64{5, 5, 5, 5})); err != nil {
		b.Fatalf("Open() error = %v, want nil", err)
	}
	events := []economysim.Event{
		economysim.TickStarted{Tick: 0},
		flow(0, alpha, economysim.ReasonConsumption, grain, 3),
		economysim.MarketCleared{Tick: 0, Region: beta, Commodity: wood, Price: 8, Volume: 4},
		economysim.RecipeExecuted{Tick: 0, Region: alpha, Recipe: recGrow, Executions: 2, Outputs: []economysim.Amount{{Commodity: grain, Quantity: 4}}},
		economysim.UpkeepShortfall{Tick: 0, Region: alpha, Role: grower, Unfed: 2, MaxHunger: 3},
		economysim.TickEnded{Tick: 0},
	}
	for _, e := range events { // create every row once
		tl.Observe(e)
	}
	b.ReportAllocs()
	for b.Loop() {
		for _, e := range events {
			tl.Observe(e)
		}
	}
}
