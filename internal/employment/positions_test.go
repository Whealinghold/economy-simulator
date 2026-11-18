package employment_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/employment"
)

// TestPositions_MinOfEstablishmentStockToolsCash verifies the positions the
// default posts: the least of the establishment left, the stock, the tools and
// the cash, each at least 0.
//
// Requirement: FR-EMP-015
func TestPositions_MinOfEstablishmentStockToolsCash(t *testing.T) {
	t.Parallel()
	base := employment.Plan{
		MaxHires: 6, Hired: 1, // establishment left: 5
		Inputs: []employment.Input{{Stock: 30, Keep: 6, Qty: 3}}, // (30 - 6) / 3 = 8, less k_own 1 = 7
		Tools:  []int64{9},                                       // 9 - k_own 1 = 8
		KOwn:   1,
		Money:  200, UpkeepCost: 20, // cash 180
		Bid: 4, Duration: 3, // 12 a position: 180 / 12 = 15
	}
	tests := []struct {
		name   string
		change func(p *employment.Plan)
		want   int64
	}{
		{"the establishment binds", func(p *employment.Plan) {}, 5},
		{"the stock binds", func(p *employment.Plan) { p.MaxHires = 100; p.Tools = []int64{50}; p.Money = 10_000 }, 7},
		// tools: 9 - 1 = 8 against the establishment 5 and the stock 7 and the cash 15.
		{"the tools bind", func(p *employment.Plan) { p.MaxHires = 100; p.Inputs = nil; p.Money = 10_000 }, 8},
		// cash 180 / 12 = 15 against 100 - 1 establishment.
		{"the cash binds", func(p *employment.Plan) { p.MaxHires = 100; p.Inputs = nil; p.Tools = nil }, 15},
		{"keep above the stock gives none", func(p *employment.Plan) { p.Inputs = []employment.Input{{Stock: 5, Keep: 9, Qty: 1}} }, 0},
		{"own request takes the stock", func(p *employment.Plan) { p.KOwn = 9 }, 0},
		{"upkeep above the money gives none", func(p *employment.Plan) { p.UpkeepCost = 500 }, 0},
		{"establishment full", func(p *employment.Plan) { p.Hired = 6 }, 0},
		{"no input and no tool is not bound by them", func(p *employment.Plan) { p.Inputs = nil; p.Tools = nil; p.MaxHires = 3; p.Hired = 0 }, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := base
			tt.change(&p)
			got, ok := employment.Positions(p)
			if !ok || got != tt.want {
				t.Errorf("Positions(%+v) = %d, %v, want %d, true", p, got, ok, tt.want)
			}
		})
	}
}

// TestPositions_ZeroWhenBidBelowOne verifies that a bid below 1 posts nothing,
// and that a count or a recipe the caller should have refused is reported.
//
// Requirement: FR-EMP-015
func TestPositions_ZeroWhenBidBelowOne(t *testing.T) {
	t.Parallel()
	ok := employment.Plan{MaxHires: 4, Money: 100, Bid: 5, Duration: 2}
	zero := ok
	zero.Bid = 0
	if got, valid := employment.Positions(zero); !valid || got != 0 {
		t.Errorf("Positions(bid 0) = %d, %v, want 0, true", got, valid)
	}
	invalid := map[string]func(p *employment.Plan){
		"zero duration":    func(p *employment.Plan) { p.Duration = 0 },
		"negative hired":   func(p *employment.Plan) { p.Hired = -1 },
		"negative k own":   func(p *employment.Plan) { p.KOwn = -1 },
		"zero input qty":   func(p *employment.Plan) { p.Inputs = []employment.Input{{Stock: 1, Qty: 0}} },
		"negative stock":   func(p *employment.Plan) { p.Inputs = []employment.Input{{Stock: -1, Qty: 1}} },
		"negative tools":   func(p *employment.Plan) { p.Tools = []int64{-1} },
		"cash product big": func(p *employment.Plan) { p.Bid = math.MaxInt64; p.Duration = 2 },
	}
	for name, change := range invalid {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := ok
			change(&p)
			if got, valid := employment.Positions(p); valid {
				t.Errorf("Positions(%+v) = %d, true, want false", p, got)
			}
		})
	}
}

// FuzzPositions verifies that Positions never panics and returns a value in 0
// to the establishment left.
//
// Requirement: NFR-EMP-003
func FuzzPositions(f *testing.F) {
	f.Add(int64(6), int64(1), int64(30), int64(6), int64(3), int64(9), int64(1), int64(200), int64(20), int64(4), int64(3))
	f.Add(int64(math.MaxInt64), int64(0), int64(math.MaxInt64), int64(0), int64(1), int64(math.MaxInt64), int64(0), int64(math.MaxInt64), int64(0), int64(1), int64(1))
	f.Fuzz(func(t *testing.T, maxHires, hired, stock, keep, qty, tools, kOwn, money, upkeep, bid, duration int64) {
		p := employment.Plan{
			MaxHires: maxHires, Hired: hired,
			Inputs: []employment.Input{{Stock: stock, Keep: keep, Qty: qty}},
			Tools:  []int64{tools}, KOwn: kOwn,
			Money: money, UpkeepCost: upkeep, Bid: bid, Duration: duration,
		}
		got, ok := employment.Positions(p)
		if !ok {
			return
		}
		if got < 0 || got > max(maxHires-hired, 0) {
			t.Fatalf("Positions(%+v) = %d, outside 0 to the establishment left", p, got)
		}
	})
}
