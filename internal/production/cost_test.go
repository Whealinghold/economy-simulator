package production_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/production"
)

// The worked example of PRO 9.2: commodity 0 is grain (200), 1 is the tool
// (20 000), 2 is flour and 3 is bran.
var (
	exInputs = []production.Term{{Commodity: 0, Quantity: 3}}
	exTools  = []production.Wear{{Commodity: 1, WearPerUse: 1, Durability: 2000}}
	exPrices = []int64{200, 20000, 0, 0}
)

// TestUnitCost_Cases verifies the worked example of PRO 9.2 and each rule of
// 9.1.1 breaking only itself.
//
// Requirement: FR-PRO-035, FR-PRO-036
func TestUnitCost_Cases(t *testing.T) {
	t.Parallel()
	zeroGrain := []int64{0, 20000, 0, 0}
	zeroTool := []int64{200, 0, 0, 0}
	// Cost 910 as in the example. share 10 000, outQty 2, yield 10 000.
	tests := []struct {
		name                                               string
		inputs                                             []production.Term
		tools                                              []production.Wear
		prices                                             []int64
		share, outQty, yieldBP, duration, wage, rent, want int64
	}{
		// 600 + ceil(1*20 000/2 000) = 10 + 1*300 = 910; 910*10 000/(2*10 000) = 455.
		{"flour at 10 000", exInputs, exTools, exPrices, 10000, 2, 10000, 1, 300, 0, 455},
		// 910*10 000/(2*8 000) = 568.75, up to 569.
		{"flour at 8 000", exInputs, exTools, exPrices, 10000, 2, 8000, 1, 300, 0, 569},
		{"byproduct share 0", exInputs, exTools, exPrices, 0, 1, 10000, 1, 300, 0, 0},
		{"wage 0", exInputs, exTools, exPrices, 10000, 2, 10000, 1, 0, 0, 0},
		{"wage negative", exInputs, exTools, exPrices, 10000, 2, 10000, 1, -300, 0, 0},
		{"input price 0", exInputs, exTools, zeroGrain, 10000, 2, 10000, 1, 300, 0, 0},
		{"tool price 0", exInputs, exTools, zeroTool, 10000, 2, 10000, 1, 300, 0, 0},
		{"factor 0", exInputs, exTools, exPrices, 10000, 2, 0, 1, 300, 0, 0},
		{"share negative", exInputs, exTools, exPrices, -1, 2, 10000, 1, 300, 0, 0},
		// Rent adds to the cost: 910 + 90 = 1 000; 1 000/2 = 500.
		{"rent", exInputs, exTools, exPrices, 10000, 2, 10000, 1, 300, 90, 500},
		// 3*200 + 0 + 0 = 600 with no tool, no labour: 600/2 = 300.
		{"no tool, no duration", exInputs, nil, exPrices, 10000, 2, 10000, 0, 300, 0, 300},
		// Rounding up: cost 7, share 10 000, outQty 2: 3.5 up to 4 (truncation gives 3).
		{"rounding up", []production.Term{{0, 1}}, nil, []int64{7}, 10000, 2, 10000, 0, 1, 0, 4},
		// Share 5 000: 910*5 000/(2*10 000) = 227.5 up to 228.
		{"half share", exInputs, exTools, exPrices, 5000, 2, 10000, 1, 300, 0, 228},
		// Overflow of the input sum: MaxInt64 * 2 does not fit.
		{"input product overflow", []production.Term{{0, math.MaxInt64}}, nil, []int64{2}, 10000, 1, 10000, 1, 1, 0, 0},
		// Overflow of the cost sum: MaxInt64 + MaxInt64.
		{"cost sum overflow", []production.Term{{0, math.MaxInt64}, {1, 1}}, nil, []int64{1, math.MaxInt64}, 10000, 1, 10000, 1, 1, 0, 0},
		// Overflow of the share product: cost MaxInt64 * share 20 000 / 10 000 is 2 * MaxInt64.
		{"share product overflow", []production.Term{{0, math.MaxInt64}}, nil, []int64{1}, 20000, 1, 10000, 0, 1, 0, 0},
		// MaxInt64 * 10 000 / (1 * 10 000) = MaxInt64 still fits (128-bit product).
		{"large but fits", []production.Term{{0, math.MaxInt64}}, nil, []int64{1}, 10000, 1, 10000, 0, 1, 0, math.MaxInt64},
		// Labour overflow: MaxInt64 ticks * wage 2.
		{"labour overflow", nil, nil, []int64{1}, 10000, 1, 10000, math.MaxInt64, 2, 0, 0},
		// outQty * yieldBP overflow.
		{"denominator overflow", exInputs, nil, exPrices, 10000, math.MaxInt64, 10000, 1, 300, 0, 0},
		{"commodity outside prices", []production.Term{{9, 1}}, nil, exPrices, 10000, 1, 10000, 1, 300, 0, 0},
		{"negative commodity", []production.Term{{-1, 1}}, nil, exPrices, 10000, 1, 10000, 1, 300, 0, 0},
		{"tool outside prices", nil, []production.Wear{{9, 1, 10}}, exPrices, 10000, 1, 10000, 1, 300, 0, 0},
		{"durability 0", nil, []production.Wear{{1, 1, 0}}, exPrices, 10000, 1, 10000, 1, 300, 0, 0},
		{"negative rent", exInputs, exTools, exPrices, 10000, 2, 10000, 1, 300, -1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := production.UnitCost(tt.inputs, tt.tools, tt.prices, tt.share, tt.outQty, tt.yieldBP, tt.duration, tt.wage, tt.rent)
			if got != tt.want {
				t.Errorf("UnitCost = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestWearCost_Cases verifies the wear of one execution, rounded up.
//
// Requirement: FR-PRO-035
func TestWearCost_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		w      production.Wear
		price  int64
		want   int64
		wantOK bool
	}{
		{"exact", production.Wear{1, 1, 2000}, 20000, 10, true},  // 20 000 / 2 000 = 10
		{"rounds up", production.Wear{1, 1, 3}, 10, 4, true},     // 10 / 3 = 3.33, up to 4
		{"zero wear", production.Wear{1, 0, 3}, 10, 0, true},     // 0
		{"durability 0", production.Wear{1, 1, 0}, 10, 0, false}, // division by zero
		{"overflow", production.Wear{1, math.MaxInt64, 1}, math.MaxInt64, 0, false},
		{"negative price", production.Wear{1, 1, 3}, -1, 0, false},
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

// TestAnchors_Cases verifies the cost anchor of PRO 9.3: the cheaper non-zero
// unit cost wins, unknown costs and recipes that cannot run are ignored, a
// byproduct never sets an anchor and a wage of 0 gives 0 everywhere.
//
// Requirement: FR-PRO-037, FR-PRO-049
func TestAnchors_Cases(t *testing.T) {
	t.Parallel()
	flour := func(yield int64, runnable bool) production.Recipe {
		return production.Recipe{
			Inputs: exInputs, Tools: exTools, Duration: 1, YieldBP: yield, Runnable: runnable,
			// Flour (2) with share 10 000, bran (3) a byproduct with share 0.
			Outputs: []production.Out{{Commodity: 2, Quantity: 2, ShareBP: 10000}, {Commodity: 3, Quantity: 1}},
		}
	}
	// A second flour recipe with no tool and no labour: 600 / 2 = 300.
	cheap := production.Recipe{Inputs: exInputs, YieldBP: 10000, Runnable: true,
		Outputs: []production.Out{{Commodity: 2, Quantity: 2, ShareBP: 10000}}}
	// Another whose input has no price: unknown, ignored.
	unknown := production.Recipe{Inputs: []production.Term{{Commodity: 2, Quantity: 1}}, YieldBP: 10000, Runnable: true,
		Outputs: []production.Out{{Commodity: 2, Quantity: 2, ShareBP: 10000}}}
	// A discard output with a share never sets an anchor.
	discard := production.Recipe{Inputs: exInputs, YieldBP: 10000, Runnable: true,
		Outputs: []production.Out{{Commodity: 3, Quantity: 1, ShareBP: 10000, Discard: true}}}

	tests := []struct {
		name    string
		recipes []production.Recipe
		makers  [][]int
		wage    int64
		want    []int64
	}{
		// flour: 910 * 10 000 / (2 * 10 000) = 455; bran 0 (byproduct).
		{"one recipe", []production.Recipe{flour(10000, true)}, [][]int{nil, nil, {0}, {0}}, 300, []int64{0, 0, 455, 0}},
		// The cheaper wins: min(455, 300) = 300, whatever the order.
		{"cheaper wins", []production.Recipe{flour(10000, true), cheap}, [][]int{nil, nil, {0, 1}, {0}}, 300, []int64{0, 0, 300, 0}},
		{"cheaper wins reversed", []production.Recipe{cheap, flour(10000, true)}, [][]int{nil, nil, {0, 1}, {0}}, 300, []int64{0, 0, 300, 0}},
		// The cheap recipe cannot run: 455 stays.
		{"not runnable ignored", []production.Recipe{flour(10000, true), func() production.Recipe { r := cheap; r.Runnable = false; return r }()}, [][]int{nil, nil, {0, 1}, nil}, 300, []int64{0, 0, 455, 0}},
		// The cheap recipe has E 0: ignored.
		{"zero yield ignored", []production.Recipe{flour(10000, true), func() production.Recipe { r := cheap; r.YieldBP = 0; return r }()}, [][]int{nil, nil, {0, 1}, nil}, 300, []int64{0, 0, 455, 0}},
		// A recipe with an unknown cost is ignored: 455 stays.
		{"unknown cost ignored", []production.Recipe{flour(10000, true), unknown}, [][]int{nil, nil, {0, 1}, nil}, 300, []int64{0, 0, 455, 0}},
		{"no maker is 0", []production.Recipe{flour(10000, true)}, [][]int{nil, nil, nil, nil}, 300, []int64{0, 0, 0, 0}},
		{"only unknown is 0", []production.Recipe{unknown}, [][]int{nil, nil, {0}, nil}, 300, []int64{0, 0, 0, 0}},
		{"wage 0 is all 0", []production.Recipe{flour(10000, true)}, [][]int{nil, nil, {0}, {0}}, 0, []int64{0, 0, 0, 0}},
		{"discard output never anchors", []production.Recipe{discard}, [][]int{nil, nil, nil, {0}}, 300, []int64{0, 0, 0, 0}},
		{"maker index outside recipes", []production.Recipe{flour(10000, true)}, [][]int{nil, nil, {5, -1, 0}, nil}, 300, []int64{0, 0, 455, 0}},
		// A stale row is cleared: the row starts as 9s.
		{"row is overwritten", nil, [][]int{nil, nil, nil, nil}, 300, []int64{0, 0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			row := []int64{9, 9, 9, 9}
			production.Anchors(row, tt.recipes, tt.makers, exPrices, tt.wage)
			for i := range row {
				if row[i] != tt.want[i] {
					t.Errorf("Anchors row = %v, want %v", row, tt.want)
					break
				}
			}
		})
	}
}

// TestCostCount_NoAllocs verifies that the count, the cap helpers, UnitCost
// and Anchors allocate nothing.
//
// Requirement: NFR-PRO-001, NFR-PRO-003
func TestCostCount_NoAllocs(t *testing.T) {
	limits := []production.Limit{{Cap: production.NoLimit, Reason: 1}, {Cap: 3, Reason: 2}, {Cap: 0, Reason: 3}}
	recipes := []production.Recipe{{Inputs: exInputs, Tools: exTools, Duration: 1, YieldBP: 8000, Runnable: true,
		Outputs: []production.Out{{Commodity: 2, Quantity: 2, ShareBP: 10000}}}}
	makers := [][]int{nil, nil, {0}, nil}
	row := make([]int64, 4)
	var sink int64
	tests := []struct {
		name string
		fn   func()
	}{
		{"Count", func() { k, r := production.Count(10, 4, limits); sink += k + int64(r) }},
		{"Cap", func() { sink += production.Cap(7, 2) + production.CapInputAndTool(7, 2) }},
		{"YieldLimit", func() { c, b := production.YieldLimit(true, false, 0); _ = b; sink += c }},
		{"UnitCost", func() { sink += production.UnitCost(exInputs, exTools, exPrices, 10000, 2, 8000, 1, 300, 0) }},
		{"Anchors", func() { production.Anchors(row, recipes, makers, exPrices, 300); sink += row[2] }},
	}
	for _, tt := range tests {
		if got := testing.AllocsPerRun(100, tt.fn); got != 0 {
			t.Errorf("%s allocs = %v, want 0", tt.name, got)
		}
	}
	_ = sink
}

// FuzzUnitCost verifies no panic for any operands and a result never
// negative.
//
// Requirement: NFR-PRO-004, FR-PRO-036
func FuzzUnitCost(f *testing.F) {
	f.Add(encode(3, 200, 1, 20000, 1, 2000, 10000, 2, 10000, 1, 300, 0))
	f.Add(encode(math.MaxInt64, 2, 1, 1, 1, 1, 20000, 1, 10000, 1, 1, 0))
	f.Add(encode(math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		prices := []int64{word(data, 1), word(data, 3)}
		in := []production.Term{{Commodity: int(word(data, 12) % 3), Quantity: word(data, 0)}}
		tools := []production.Wear{{Commodity: 1, WearPerUse: word(data, 4), Durability: word(data, 5)}}
		got := production.UnitCost(in, tools, prices, word(data, 6), word(data, 7), word(data, 8), word(data, 9), word(data, 10), word(data, 11))
		if got < 0 {
			t.Fatalf("UnitCost(%v) = %d, want not negative", data, got)
		}
	})
}

// FuzzExecutions verifies no panic and a count never above the weight, the
// maximum when positive, or any cap, and never negative.
//
// Requirement: NFR-PRO-004, FR-PRO-009
func FuzzExecutions(f *testing.F) {
	f.Add(encode(10, 4, 7, 2, 0, 1, 5, 3))
	f.Add(encode(math.MaxInt64, math.MinInt64, math.MinInt64, math.MaxInt64, -1, 0))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		w, maxi := word(data, 0), word(data, 1)
		var limits []production.Limit
		for i := 2; i+1 < len(data)/8 && len(limits) < 8; i += 2 {
			limits = append(limits, production.Limit{Cap: production.Cap(word(data, i), word(data, i+1)), Reason: len(limits)})
		}
		k, _ := production.Count(w, maxi, limits)
		if k < 0 || k > max(w, 0) || (maxi > 0 && k > maxi) {
			t.Fatalf("Count(%d, %d, %v) = %d outside the weight and maximum", w, maxi, limits, k)
		}
		for _, l := range limits {
			if k > l.Cap && l.Cap >= 0 || (l.Cap < 0 && k != 0) {
				t.Fatalf("Count(%d, %d, %v) = %d exceeds cap %d", w, maxi, limits, k, l.Cap)
			}
		}
	})
}

// BenchmarkSite, BenchmarkCount and BenchmarkUnitCost report allocations
// (NFR-PRO-003); Count at weight 1 and at a large weight costs the same
// (NFR-PRO-002).
//
// Requirement: NFR-PRO-002, NFR-PRO-003
func BenchmarkSite(b *testing.B) {
	prof := &production.Profile{HasAll: true, All: 5000, Seasons: []production.SeasonBP{{Season: 0, BP: 1}, {Season: 2, BP: 3}}}
	b.ReportAllocs()
	var sink int64
	for b.Loop() {
		sink += production.Site(prof, true, 2)
	}
	_ = sink
}

// BenchmarkCount reports the cost and allocations of Count at the smallest and
// the largest weight, for the trend of the tick loop.
//
// Requirement: NFR-PRO-002, NFR-PRO-003
func BenchmarkCount(b *testing.B) {
	limits := []production.Limit{{Cap: production.NoLimit, Reason: 1}, {Cap: 3, Reason: 2}, {Cap: 4, Reason: 3}}
	for _, w := range []int64{1, 1 << 40} {
		b.Run(map[int64]string{1: "weight1", 1 << 40: "weightMax"}[w], func(b *testing.B) {
			b.ReportAllocs()
			var sink int64
			for b.Loop() {
				k, _ := production.Count(w, 0, limits)
				sink += k
			}
			_ = sink
		})
	}
}

// BenchmarkUnitCost reports the cost and allocations of UnitCost, for the
// trend of the cost anchor pass.
//
// Requirement: NFR-PRO-002, NFR-PRO-003
func BenchmarkUnitCost(b *testing.B) {
	b.ReportAllocs()
	var sink int64
	for b.Loop() {
		sink += production.UnitCost(exInputs, exTools, exPrices, 10000, 2, 8000, 1, 300, 0)
	}
	_ = sink
}
