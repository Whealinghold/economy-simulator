package production_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/production"
)

// TestWearCost_Guards verifies that each rejected operand of WearCost breaks
// only itself, and that a zero price is a valid zero wear.
//
// Requirement: FR-PRO-035
func TestWearCost_Guards(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		w      production.Wear
		price  int64
		want   int64
		wantOK bool
	}{
		// 1 * 10 / -3 would be ceil(-3.33) = -3 if the guard let it through.
		{"negative durability alone", production.Wear{Commodity: 1, WearPerUse: 1, Durability: -3}, 10, 0, false},
		// -1 * 10 / 3 would be ceil(-3.33) = -3 if the guard let it through.
		{"negative wear alone", production.Wear{Commodity: 1, WearPerUse: -1, Durability: 3}, 10, 0, false},
		// 1 * 0 / 3 = 0, valid.
		{"price 0", production.Wear{Commodity: 1, WearPerUse: 1, Durability: 3}, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := production.WearCost(tt.w, tt.price)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("WearCost(%+v, %d) = %d, %v, want %d, %v", tt.w, tt.price, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestUnitCost_Boundaries verifies the index bounds, the sign guards and the
// overflow guards of UnitCost with operands where a loosened guard would give
// a different non-zero result.
//
// Requirement: FR-PRO-035, FR-PRO-036
func TestUnitCost_Boundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                               string
		inputs                                             []production.Term
		tools                                              []production.Wear
		prices                                             []int64
		share, outQty, yieldBP, duration, wage, rent, want int64
	}{
		// cost 3*200 + 1*300 = 900; 900 * -20 000 / 20 000 = -900 would be returned.
		{"share negative large", exInputs, nil, exPrices, -20000, 2, 10000, 1, 300, 0, 0},
		// 900 * 10 000 / -20 000 = -450 would be returned.
		{"yield negative", exInputs, nil, exPrices, 10000, 2, -10000, 1, 300, 0, 0},
		// 900 * 10 000 / (-2 * 10 000) = -450 would be returned.
		{"outQty negative", exInputs, nil, exPrices, 10000, -2, 10000, 1, 300, 0, 0},
		// Commodity 1 (price 20 000) is a valid input index: 20 000 + 300.
		{"input commodity 1", []production.Term{{Commodity: 1, Quantity: 1}}, nil, exPrices, 10000, 1, 10000, 1, 300, 0, 20300},
		// Index 4 is one past the 4 prices.
		{"input commodity at len", []production.Term{{Commodity: 4, Quantity: 1}}, nil, exPrices, 10000, 1, 10000, 1, 300, 0, 0},
		// A negative price would give 3*-200 + 300 = -300.
		{"input price negative", exInputs, nil, []int64{-200, 0, 0, 0}, 10000, 1, 10000, 1, 300, 0, 0},
		// Quantity 0 is fine: 0 * 200 + 300 = 300.
		{"input quantity 0", []production.Term{{Commodity: 0, Quantity: 0}}, nil, exPrices, 10000, 1, 10000, 1, 300, 0, 300},
		// A negative quantity would give -1*200 + 300 = 100.
		{"input quantity negative", []production.Term{{Commodity: 0, Quantity: -1}}, nil, exPrices, 10000, 1, 10000, 1, 300, 0, 0},
		// Tool on commodity 0: wear 1*200/1 = 200, plus 300 labour.
		{"tool commodity 0", nil, []production.Wear{{Commodity: 0, WearPerUse: 1, Durability: 1}}, exPrices, 10000, 1, 10000, 1, 300, 0, 500},
		{"tool commodity negative", nil, []production.Wear{{Commodity: -1, WearPerUse: 1, Durability: 1}}, exPrices, 10000, 1, 10000, 1, 300, 0, 0},
		{"tool commodity at len", nil, []production.Wear{{Commodity: 4, WearPerUse: 1, Durability: 1}}, exPrices, 10000, 1, 10000, 1, 300, 0, 0},
		// MaxInt64 + wear 1 overflows; ignoring it would leave 300 of labour.
		{"tool sum overflow", []production.Term{{Commodity: 0, Quantity: 1}}, []production.Wear{{Commodity: 1, WearPerUse: 1, Durability: 1}}, []int64{math.MaxInt64, 1}, 10000, 1, 10000, 1, 300, 0, 0},
		// Labour MaxInt64 * 2 overflows; ignoring it would leave the 600 of input.
		{"labour overflow with input", exInputs, nil, exPrices, 10000, 1, 10000, math.MaxInt64, 2, 0, 0},
		// MaxInt64 + labour 1 overflows; ignoring it would leave the rent of 10.
		{"labour sum overflow", []production.Term{{Commodity: 0, Quantity: 1}}, nil, []int64{math.MaxInt64}, 10000, 1, 10000, 1, 1, 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := production.UnitCost(tt.inputs, tt.tools, tt.prices, production.Basis{Share: tt.share, OutQty: tt.outQty, YieldBP: tt.yieldBP, Duration: tt.duration, Wage: tt.wage, Rent: tt.rent})
			if got != tt.want {
				t.Errorf("UnitCost = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestAnchors_IndexAndOutputs verifies that a maker index equal to the number
// of recipes is skipped, and that only the output of the commodity itself
// anchors it.
//
// Requirement: FR-PRO-037
func TestAnchors_IndexAndOutputs(t *testing.T) {
	t.Parallel()
	// Cost 910: flour 910 * 10 000 / (2 * 10 000) = 455; bran share 3 000 of
	// one unit is 910 * 3 000 / 10 000 = 273, which must not anchor flour.
	r := production.Recipe{Inputs: exInputs, Tools: exTools, Duration: 1, YieldBP: 10000, Runnable: true,
		Outputs: []production.Out{{Commodity: 3, Quantity: 1, ShareBP: 3000}, {Commodity: 2, Quantity: 2, ShareBP: 10000}}}
	tests := []struct {
		name   string
		makers [][]int
		want   []int64
	}{
		{"index at len skipped", [][]int{nil, nil, {1, 0}, nil}, []int64{0, 0, 455, 0}},
		{"other output ignored", [][]int{nil, nil, {0}, nil}, []int64{0, 0, 455, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			row := make([]int64, 4)
			production.Anchors(row, []production.Recipe{r}, tt.makers, exPrices, 300)
			for i := range row {
				if row[i] != tt.want[i] {
					t.Fatalf("Anchors row = %v, want %v", row, tt.want)
				}
			}
		})
	}
}
