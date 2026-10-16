package beliefs_test

import (
	"encoding/binary"
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
)

// specParams are the defaults of spec-beliefs.md 11: step 200, gate 5 000,
// significance 500, evidence share 5 000, stock band 5 000 to 15 000, max age
// 3, width 200 to 5 000 in steps of 250, cost band 7 000 to 30 000 and
// reference band 1 000 to 100 000.
func specParams() beliefs.Params {
	return beliefs.Params{
		Step: 200, AdjustProb: 5000, Significance: 500, EvidenceShare: 5000,
		StockLow: 5000, StockHigh: 15000, MaxAge: 3,
		WidthInit: 2000, WidthMin: 200, WidthMax: 5000, WidthStep: 250,
		CostFloor: 7000, CostCeiling: 30000, RefBandLow: 1000, RefBandHigh: 100000,
		MaxPrice: maxPrice,
	}
}

// openGate is a draw whose gate is open at the default AdjustProb: the high
// word of 0 * 10 000 is 0, below 5 000.
const openGate = 0

// splitmix gives a spread of words from a loop seed (a fixed seed range, so a
// failure prints the seed that reproduces it).
func splitmix(s *uint64) uint64 {
	*s += 0x9e3779b97f4a7c15
	z := *s
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	return z ^ z>>31
}

var wideBounds = beliefs.Bounds{Floor: 100, Ceiling: 10000}

// TestUpdate_WorkedCases verifies the four worked cases of spec-beliefs.md 15,
// and that a Change built from each call's input and output reads the move.
//
// Requirement: FR-BEL-009, FR-BEL-010, FR-BEL-011, FR-BEL-012, FR-BEL-013, FR-BEL-014, FR-BEL-015, FR-BEL-016, FR-BEL-034, FR-BEL-036
func TestUpdate_WorkedCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		in            beliefs.Belief
		bd            beliefs.Bounds
		o             beliefs.Outcome
		q             beliefs.Quote
		stock, target int64
		want          beliefs.Belief
		raised, lower bool
		atFloor       bool
	}{
		// Case A: delta = 1 000 * 200 / 10 000 = 20, de = 10. The price is stale
		// (age 5 > 3), so T = A = 1 100: gL = 160, gC = 100, evidence +min(160,
		// 100, 10) = +10. Stock ratio 2 * 10 000 / 10 = 2 000 < 5 000: +20.
		// Centre 1 030; fill share 0: width 2 000 + 250.
		{name: "A stale price, unmatched ask pulls up", in: beliefs.Belief{Centre: 1000, Width: 2000}, bd: wideBounds,
			o: beliefs.Outcome{Bid: true, Limit: 940, Quantity: 10, Filled: 0},
			q: beliefs.Quote{Price: 1000, Age: 5, BestAsk: 1100, HasAsk: true}, stock: 2, target: 10,
			want: beliefs.Belief{Centre: 1030, Width: 2250}, raised: true},
		// Case B: significance threshold 1 000 * 500 / 10 000 = 50 < 70, T = 1 010:
		// gL = 70, gC = 10, evidence +min(70, 10, 10) = +10. Ratio 10 000: no
		// stock step. Centre 1 010; fill share 10 000: width 2 000 - 250.
		{name: "B filled ask below the market", in: beliefs.Belief{Centre: 1000, Width: 2000}, bd: wideBounds,
			o: beliefs.Outcome{Bid: false, Limit: 940, Quantity: 4, Filled: 4},
			q: beliefs.Quote{Price: 1010, Age: 0}, stock: 8, target: 8,
			want: beliefs.Belief{Centre: 1010, Width: 1750}, raised: true},
		// Case C: bounds [1 120, 4 800]; delta = 1 140 * 200 / 10 000 = 22, de =
		// 11. Candidates B = 990 and P = 1 000 (fresh, below 1 160): T = 1 000,
		// gL = -160, gC = -140, evidence -11. Ratio 20 000 > 15 000: -22. Raw
		// 1 140 - 33 = 1 107 -> floor 1 120. Width 2 000 + 250.
		{name: "C pinned at the cost floor", in: beliefs.Belief{Centre: 1140, Width: 2000}, bd: beliefs.Bounds{Floor: 1120, Ceiling: 4800},
			o: beliefs.Outcome{Bid: false, Limit: 1160, Quantity: 5, Filled: 0},
			q: beliefs.Quote{Price: 1000, Age: 0, BestBid: 990, HasBid: true}, stock: 20, target: 10,
			want: beliefs.Belief{Centre: 1120, Width: 2250}, lower: true, atFloor: true},
		// Case D: T = A = 950; gL = 950 - 880 = +70, gC = 950 - 1 000 = -50:
		// opposite signs, evidence 0. Ratio 10 000: no stock step. Centre stays;
		// width 2 000 + 250.
		{name: "D guard: opposite gaps", in: beliefs.Belief{Centre: 1000, Width: 2000}, bd: wideBounds,
			o: beliefs.Outcome{Bid: true, Limit: 880, Quantity: 10, Filled: 0},
			q: beliefs.Quote{Price: 1000, Age: 5, BestAsk: 950, HasAsk: true}, stock: 10, target: 10,
			want: beliefs.Belief{Centre: 1000, Width: 2250}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, res := beliefs.Update(tt.in, tt.bd, tt.o, tt.q, tt.stock, tt.target, specParams(), openGate)
			if got != tt.want || !res.Open {
				t.Errorf("Update(%+v, %+v) = %+v, %+v, want %+v, open", tt.in, tt.o, got, res, tt.want)
			}
			c := beliefs.Change{Before: tt.in, After: got, Bounds: tt.bd, Open: res.Open}
			if c.Raised() != tt.raised || c.Lowered() != tt.lower || c.AtFloor() != tt.atFloor || c.AtCeiling() {
				t.Errorf("Change reads = raised %v lowered %v atFloor %v atCeiling %v, want %v %v %v false",
					c.Raised(), c.Lowered(), c.AtFloor(), c.AtCeiling(), tt.raised, tt.lower, tt.atFloor)
			}
		})
	}
}

// TestUpdate_Branches verifies each target selection of 5.4 and each stock
// branch of 5.5 with a case that breaks only that rule. The belief is centre 1
// 000, width 2 000 in [100, 10 000], so delta = 20 and de = 10, and the
// significance threshold is 50, unless stated.
//
// Requirement: FR-BEL-011, FR-BEL-012, FR-BEL-013, FR-BEL-014, FR-BEL-034
func TestUpdate_Branches(t *testing.T) {
	t.Parallel()
	bidU := beliefs.Outcome{Bid: true, Limit: 940, Quantity: 10, Filled: 0}
	askU := beliefs.Outcome{Bid: false, Limit: 1060, Quantity: 10, Filled: 0}
	tests := []struct {
		name          string
		o             beliefs.Outcome
		q             beliefs.Quote
		stock, target int64
		want          int64
	}{
		// Unfilled bid. Stale means age 4 > 3, so the price is no evidence.
		{"bid unfilled, ask only", bidU, beliefs.Quote{Age: 9, BestAsk: 1100, HasAsk: true}, 10, 10, 1010},                              // gL 160, gC 100, de 10
		{"bid unfilled, no quote", bidU, beliefs.Quote{Age: 9, Price: 1050}, 10, 10, 1000},                                              // no target
		{"bid unfilled, fresh price above limit", bidU, beliefs.Quote{Age: 0, Price: 1050}, 10, 10, 1010},                               // T 1 050: gL 110, gC 50
		{"bid unfilled, smaller of ask and price", bidU, beliefs.Quote{Age: 0, Price: 1050, BestAsk: 1004, HasAsk: true}, 10, 10, 1004}, // T 1 004: gC 4 < de
		{"bid unfilled, price not above limit", bidU, beliefs.Quote{Age: 0, Price: 900}, 10, 10, 1000},                                  // 900 < 940: no candidate
		{"bid unfilled, price at limit", bidU, beliefs.Quote{Age: 0, Price: 940}, 10, 10, 1000},                                         // 940 is not > 940
		{"bid unfilled, price age at max is fresh", bidU, beliefs.Quote{Age: 3, Price: 1050}, 10, 10, 1010},                             // age 3 <= 3
		{"bid unfilled, price age past max is stale", bidU, beliefs.Quote{Age: 4, Price: 1050}, 10, 10, 1000},                           // age 4 > 3
		// Unfilled ask.
		{"ask unfilled, bid only", askU, beliefs.Quote{Age: 9, BestBid: 900, HasBid: true}, 10, 10, 990},                            // gL -160, gC -100
		{"ask unfilled, larger of bid and price", askU, beliefs.Quote{Age: 0, Price: 950, BestBid: 996, HasBid: true}, 10, 10, 996}, // T 996: gC -4
		{"ask unfilled, price above limit", askU, beliefs.Quote{Age: 0, Price: 1100}, 10, 10, 1000},                                 // 1 100 > 1 060: no candidate
		{"ask unfilled, fresh price below limit", askU, beliefs.Quote{Age: 0, Price: 950}, 10, 10, 990},                             // T 950: gL -110, gC -50
		// Filled bid: threshold 50 on limit - P.
		{"bid filled, at the threshold", beliefs.Outcome{Bid: true, Limit: 1040, Quantity: 4, Filled: 4}, beliefs.Quote{Price: 990}, 10, 10, 1000},       // 1 040 - 990 = 50, not > 50
		{"bid filled, one above the threshold", beliefs.Outcome{Bid: true, Limit: 1041, Quantity: 4, Filled: 4}, beliefs.Quote{Price: 990}, 10, 10, 990}, // 51 > 50: T 990, gL -51, gC -10
		{"bid filled, stale price", beliefs.Outcome{Bid: true, Limit: 1100, Quantity: 4, Filled: 4}, beliefs.Quote{Price: 990, Age: 4}, 10, 10, 1000},    // stale: no target
		// Filled ask: threshold 50 on P - limit.
		{"ask filled, at the threshold", beliefs.Outcome{Bid: false, Limit: 1000, Quantity: 4, Filled: 4}, beliefs.Quote{Price: 1050}, 10, 10, 1000},        // 50, not > 50
		{"ask filled, one above the threshold", beliefs.Outcome{Bid: false, Limit: 1000, Quantity: 4, Filled: 4}, beliefs.Quote{Price: 1051}, 10, 10, 1010}, // 51 > 50: gL 51, gC 51, de 10
		// Stock: no evidence (stale, no quote); ratio = stock * 10 000 / target.
		{"stock ratio at low is neutral", bidU, beliefs.Quote{Age: 9}, 5, 10, 1000},                       // 5 000, not < 5 000
		{"stock ratio below low pushes up", bidU, beliefs.Quote{Age: 9}, 4, 10, 1020},                     // 4 000 < 5 000: +20
		{"stock ratio at high is neutral", bidU, beliefs.Quote{Age: 9}, 15, 10, 1000},                     // 15 000, not > 15 000
		{"stock ratio above high pushes down", bidU, beliefs.Quote{Age: 9}, 16, 10, 980},                  // 16 000 > 15 000: -20
		{"target 0 is neutral", bidU, beliefs.Quote{Age: 9}, 0, 0, 1000},                                  // no target stock
		{"ask side uses the same stock rule", askU, beliefs.Quote{Age: 9}, 4, 10, 1020},                   // both sides: +20
		{"evidence and stock add", bidU, beliefs.Quote{Age: 9, BestAsk: 1100, HasAsk: true}, 4, 10, 1030}, // +10 + 20
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := beliefs.Belief{Centre: 1000, Width: 2000}
			got, _ := beliefs.Update(in, wideBounds, tt.o, tt.q, tt.stock, tt.target, specParams(), openGate)
			if got.Centre != tt.want {
				t.Errorf("Update(%+v, %+v, stock %d, target %d).Centre = %d, want %d", tt.o, tt.q, tt.stock, tt.target, got.Centre, tt.want)
			}
		})
	}
}

// TestUpdate_StaleMarketClosesTheGap verifies that an unfilled bid and an
// unfilled ask in a stale market are pulled toward each other's best quote by
// min(de, A - limit, A - centre) and the mirror, and neither passes its target.
//
// Requirement: FR-BEL-034
func TestUpdate_StaleMarketClosesTheGap(t *testing.T) {
	t.Parallel()
	in := beliefs.Belief{Centre: 1000, Width: 2000}
	stale := beliefs.Quote{Price: 1000, Age: 9, BestBid: 994, HasBid: true, BestAsk: 1006, HasAsk: true}
	// Bid: limit 990, A = 1 006: gL 16, gC 6, de 10: min = 6, centre 1 006.
	bid, _ := beliefs.Update(in, wideBounds, beliefs.Outcome{Bid: true, Limit: 990, Quantity: 5}, stale, 10, 10, specParams(), openGate)
	// Ask: limit 1 010, B = 994: gL -16, gC -6: -6, centre 994.
	ask, _ := beliefs.Update(in, wideBounds, beliefs.Outcome{Bid: false, Limit: 1010, Quantity: 5}, stale, 10, 10, specParams(), openGate)
	if bid.Centre != 1006 || ask.Centre != 994 {
		t.Errorf("Update bid/ask centres = %d/%d, want 1006/994", bid.Centre, ask.Centre)
	}
}

// TestGate_Edges verifies the adjustment gate at the edge of every probability
// and that Update agrees with Gate.
//
// Requirement: FR-BEL-009
func TestGate_Edges(t *testing.T) {
	t.Parallel()
	for _, k := range []int64{1, 2500, 3333, 5000, 9999} {
		t.Run("edge", func(t *testing.T) {
			t.Parallel()
			// The first closed draw is ceil(k * 2^64 / 10 000): below it the high
			// word of u * 10 000 is at most k - 1.
			num := new(big.Int).Lsh(big.NewInt(k), 64)
			edge := new(big.Int).Div(new(big.Int).Add(num, big.NewInt(9999)), big.NewInt(10000)).Uint64()
			p := specParams()
			p.AdjustProb = k
			if !beliefs.Gate(edge-1, k) || beliefs.Gate(edge, k) {
				t.Errorf("Gate edge for AdjustProb %d at u = %d: open below = %v, open at = %v, want true, false", k, edge, beliefs.Gate(edge-1, k), beliefs.Gate(edge, k))
			}
			in := beliefs.Belief{Centre: 1000, Width: 2000}
			o := beliefs.Outcome{Bid: true, Limit: 940, Quantity: 10}
			for _, u := range []uint64{edge - 1, edge} {
				_, res := beliefs.Update(in, wideBounds, o, beliefs.Quote{Age: 9}, 10, 10, p, u)
				if res.Open != beliefs.Gate(u, k) {
					t.Errorf("Update(u=%d).Open = %v, Gate = %v, want equal", u, res.Open, beliefs.Gate(u, k))
				}
			}
		})
	}
	if !beliefs.Gate(math.MaxUint64, 10000) || !beliefs.Gate(0, 10000) {
		t.Errorf("Gate(_, 10000) closed, want always open")
	}
	if beliefs.Gate(0, 0) || beliefs.Gate(math.MaxUint64, 0) {
		t.Errorf("Gate(_, 0) open, want never open")
	}
}

// TestUpdate_ClosedGateOnlyReclamps verifies that a closed gate changes
// nothing but the re-clamp of 3.3, and processes no evidence.
//
// Requirement: FR-BEL-009
func TestUpdate_ClosedGateOnlyReclamps(t *testing.T) {
	t.Parallel()
	// u = 2^63: high word 5 000, not below AdjustProb 5 000: closed.
	closed := uint64(1) << 63
	o := beliefs.Outcome{Bid: true, Limit: 940, Quantity: 10}
	q := beliefs.Quote{Age: 9, BestAsk: 1100, HasAsk: true}
	got, res := beliefs.Update(beliefs.Belief{Centre: 1000, Width: 2000}, wideBounds, o, q, 0, 10, specParams(), closed)
	if got != (beliefs.Belief{Centre: 1000, Width: 2000}) || res.Open {
		t.Errorf("Update(closed gate) = %+v, %+v, want the belief unchanged and closed", got, res)
	}
	got, _ = beliefs.Update(beliefs.Belief{Centre: 50, Width: 9999}, wideBounds, o, q, 0, 10, specParams(), closed)
	if got != (beliefs.Belief{Centre: 100, Width: 5000}) {
		t.Errorf("Update(closed gate, outside belief) = %+v, want re-clamped {100 5000}", got)
	}
}

// TestChange_Reads verifies each derived read of a Change from its fields
// only, including a belief no Update call produced.
//
// Requirement: FR-BEL-030
func TestChange_Reads(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                                 string
		c                                                    beliefs.Change
		raised, lowered, moved, atFloor, atCeiling, conflict bool
	}{
		{"above", beliefs.Change{Before: beliefs.Belief{Centre: 500}, After: beliefs.Belief{Centre: 600}, Bounds: wideBounds}, true, false, true, false, false, false},
		{"equal", beliefs.Change{Before: beliefs.Belief{Centre: 500}, After: beliefs.Belief{Centre: 500}, Bounds: wideBounds}, false, false, false, false, false, false},
		{"below", beliefs.Change{Before: beliefs.Belief{Centre: 500}, After: beliefs.Belief{Centre: 400}, Bounds: wideBounds}, false, true, true, false, false, false},
		// A closed-gate re-clamp: the stored centre 50 is below the new floor
		// 100 and the centre after is on it.
		{"re-clamp under a closed gate", beliefs.Change{Before: beliefs.Belief{Centre: 50}, After: beliefs.Belief{Centre: 100}, Bounds: wideBounds}, true, false, true, true, false, false},
		{"on the ceiling", beliefs.Change{Before: beliefs.Belief{Centre: 9000}, After: beliefs.Belief{Centre: 10000}, Bounds: wideBounds, Open: true}, true, false, true, false, true, false},
		// Reference 1 000, unit cost 10: floor = ceiling = 100, no conflict.
		{"floor equals ceiling", beliefs.Change{Before: beliefs.Belief{Centre: 100}, After: beliefs.Belief{Centre: 100}, Bounds: beliefs.Bounds{Floor: 100, Ceiling: 100}}, false, false, false, true, true, false},
		{"conflict", beliefs.Change{Before: beliefs.Belief{Centre: 140000}, After: beliefs.Belief{Centre: 140000}, Bounds: beliefs.Bounds{Floor: 140000, Ceiling: 140000, Conflict: true}}, false, false, false, true, true, true},
		// A strategy moved the centre by an amount the rule never takes.
		{"custom strategy jump", beliefs.Change{Before: beliefs.Belief{Centre: 1000}, After: beliefs.Belief{Centre: 7777}, Bounds: wideBounds, Clamped: true}, true, false, true, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.c
			got := [6]bool{c.Raised(), c.Lowered(), c.Moved(), c.AtFloor(), c.AtCeiling(), c.Conflict()}
			want := [6]bool{tt.raised, tt.lowered, tt.moved, tt.atFloor, tt.atCeiling, tt.conflict}
			if got != want {
				t.Errorf("Change %+v reads (raised, lowered, moved, atFloor, atCeiling, conflict) = %v, want %v", c, got, want)
			}
		})
	}
}

// TestUpdate_Width verifies the width step at the fill-share edge and the
// clamps at both limits.
//
// Requirement: FR-BEL-016
func TestUpdate_Width(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		width int64
		o     beliefs.Outcome
		want  int64
	}{
		// 5 * 10 000 / 10 = 5 000: at least a half, narrows.
		{"share 5000 narrows", 2000, beliefs.Outcome{Quantity: 10, Filled: 5}, 1750},
		// 4 999 * 10 000 / 10 000 = 4 999: widens.
		{"share 4999 widens", 2000, beliefs.Outcome{Quantity: 10000, Filled: 4999}, 2250},
		{"narrow clamps at min", 300, beliefs.Outcome{Quantity: 10, Filled: 10}, 200},
		{"widen clamps at max", 4900, beliefs.Outcome{Quantity: 10, Filled: 0}, 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _ := beliefs.Update(beliefs.Belief{Centre: 1000, Width: tt.width}, wideBounds, tt.o, beliefs.Quote{Age: 9}, 10, 10, specParams(), openGate)
			if got.Width != tt.want {
				t.Errorf("Update(width %d, %+v).Width = %d, want %d", tt.width, tt.o, got.Width, tt.want)
			}
		})
	}
}

// TestUpdate_OverflowAndDegenerateInputs verifies that inputs outside the
// configuration bounds never wrap or panic and are read as 5.9 says.
//
// Requirement: FR-BEL-006
func TestUpdate_OverflowAndDegenerateInputs(t *testing.T) {
	t.Parallel()
	in := beliefs.Belief{Centre: 1000, Width: 2000}
	stale := beliefs.Quote{Age: 9, BestAsk: 1100, HasAsk: true}
	bid := beliefs.Outcome{Bid: true, Limit: 940, Quantity: 10}
	tests := []struct {
		name string
		in   beliefs.Belief
		bd   beliefs.Bounds
		p    func(*beliefs.Params)
		o    beliefs.Outcome
		q    beliefs.Quote
		s, t int64
		want beliefs.Belief
	}{
		// stock * 10 000 overflows int64: read above StockHigh, so -20; no
		// evidence (stale, no quote).
		{"stock at int64 max", in, wideBounds, nil, bid, beliefs.Quote{Age: 9}, math.MaxInt64, 1, beliefs.Belief{Centre: 980, Width: 2250}},
		{"target at int64 max", in, wideBounds, nil, bid, beliefs.Quote{Age: 9}, 1, math.MaxInt64, beliefs.Belief{Centre: 1020, Width: 2250}}, // ratio 0 < 5 000
		// 10 000 * 200 / 10 000 = 200 at the largest valid price: no wrap.
		{"centre at max price", beliefs.Belief{Centre: maxPrice, Width: 2000}, beliefs.Bounds{Floor: 1, Ceiling: maxPrice}, nil, bid, beliefs.Quote{Age: 9}, 0, 10, beliefs.Belief{Centre: maxPrice, Width: 2250}},
		// Centre 20 000 * MaxInt64 / 10 000 does not fit: delta saturates at
		// MaxInt64; the stock step adds it to 20 000 with saturation and the
		// clamp brings it to the ceiling MaxPrice.
		{"delta saturates", beliefs.Belief{Centre: 20000, Width: 2000}, beliefs.Bounds{Floor: 100, Ceiling: maxPrice}, func(p *beliefs.Params) { p.Step = math.MaxInt64 }, bid, beliefs.Quote{Age: 9}, 0, 10, beliefs.Belief{Centre: maxPrice, Width: 2250}},
		// A negative sum saturates at MinInt64 and the clamp brings it to the floor.
		{"negative saturation", beliefs.Belief{Centre: 20000, Width: 2000}, beliefs.Bounds{Floor: 100, Ceiling: maxPrice}, func(p *beliefs.Params) { p.Step = math.MaxInt64 }, bid, beliefs.Quote{Age: 9}, 100, 10, beliefs.Belief{Centre: 100, Width: 2250}},
		// Evidence -160 (gL -160, gC -19 100, de huge) plus the stock step
		// -MaxInt64 is below MinInt64: it saturates and the clamp brings it to
		// the floor.
		{"move sum saturates low", beliefs.Belief{Centre: 20000, Width: 2000}, beliefs.Bounds{Floor: 100, Ceiling: maxPrice}, func(p *beliefs.Params) { p.Step = math.MaxInt64 }, beliefs.Outcome{Bid: false, Limit: 1060, Quantity: 10}, beliefs.Quote{Age: 9, BestBid: 900, HasBid: true}, 100, 10, beliefs.Belief{Centre: 100, Width: 2250}},
		// Quantity 0 and negative: fill share 0, not fully filled; the stale
		// ask 1 100 pulls up by min(160, 100, 10) = 10.
		{"quantity 0", in, wideBounds, nil, beliefs.Outcome{Bid: true, Limit: 940, Quantity: 0, Filled: 0}, stale, 10, 10, beliefs.Belief{Centre: 1010, Width: 2250}},
		{"quantity negative", in, wideBounds, nil, beliefs.Outcome{Bid: true, Limit: 940, Quantity: -3, Filled: 5}, stale, 10, 10, beliefs.Belief{Centre: 1010, Width: 2250}},
		// Filled above Quantity is Quantity: fully filled, share 10 000, narrows.
		{"filled above quantity", in, wideBounds, nil, beliefs.Outcome{Bid: true, Limit: 940, Quantity: 4, Filled: 9}, beliefs.Quote{Age: 9}, 10, 10, beliefs.Belief{Centre: 1000, Width: 1750}},
		// Filled below 0 is 0: unfilled, share 0.
		{"filled negative", in, wideBounds, nil, beliefs.Outcome{Bid: true, Limit: 940, Quantity: 4, Filled: -2}, stale, 10, 10, beliefs.Belief{Centre: 1010, Width: 2250}},
		// A negative stock is 0 (ratio 0 < 5 000: +20); a negative target is 0 (neutral).
		{"negative stock", in, wideBounds, nil, bid, beliefs.Quote{Age: 9}, -5, 10, beliefs.Belief{Centre: 1020, Width: 2250}},
		{"negative target", in, wideBounds, nil, bid, beliefs.Quote{Age: 9}, 100, -3, beliefs.Belief{Centre: 1000, Width: 2250}},
		// A price gap that overflows is above every threshold.
		{"price gap overflows", in, wideBounds, nil, beliefs.Outcome{Bid: false, Limit: 1000, Quantity: 4, Filled: 4}, beliefs.Quote{Price: math.MaxInt64}, 10, 10, beliefs.Belief{Centre: 1010, Width: 1750}},
		{"price at int64 min", in, wideBounds, nil, beliefs.Outcome{Bid: true, Limit: 1000, Quantity: 4, Filled: 4}, beliefs.Quote{Price: math.MinInt64}, 10, 10, beliefs.Belief{Centre: 990, Width: 1750}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := specParams()
			if tt.p != nil {
				tt.p(&p)
			}
			got, _ := beliefs.Update(tt.in, tt.bd, tt.o, tt.q, tt.s, tt.t, p, openGate)
			if got != tt.want {
				t.Errorf("Update(%+v, %+v, %+v, stock %d, target %d) = %+v, want %+v", tt.in, tt.o, tt.q, tt.s, tt.t, got, tt.want)
			}
		})
	}

	// A Limit below 1 is read as 1: 0, -5 and MinInt64 all give the same
	// result as 1.
	ask := beliefs.Outcome{Bid: false, Limit: 1, Quantity: 4, Filled: 4}
	q := beliefs.Quote{Price: 1051}
	want, _ := beliefs.Update(in, wideBounds, ask, q, 10, 10, specParams(), openGate)
	for _, l := range []int64{0, -5, math.MinInt64} {
		ask.Limit = l
		if got, _ := beliefs.Update(in, wideBounds, ask, q, 10, 10, specParams(), openGate); got != want {
			t.Errorf("Update(limit %d) = %+v, want %+v as for limit 1", l, got, want)
		}
	}
}

// randomCase builds a reproducible case from a seed: a belief anywhere in
// int64, valid bounds and any outcome, quote, stock, target and draw.
func randomCase(seed uint64) (beliefs.Belief, beliefs.Bounds, beliefs.Outcome, beliefs.Quote, int64, int64, uint64) {
	s := seed
	ref := int64(splitmix(&s)%2_000_000) + 1
	cost := int64(splitmix(&s) % 3_000_000)
	bd := beliefs.Limits(ref, cost, specParams())
	b := beliefs.Belief{Centre: int64(splitmix(&s)), Width: int64(splitmix(&s))}
	o := beliefs.Outcome{Bid: splitmix(&s)&1 == 0, Limit: int64(splitmix(&s)), Quantity: int64(splitmix(&s) % 50), Filled: int64(splitmix(&s) % 60)}
	flags := splitmix(&s)
	q := beliefs.Quote{
		Price: int64(splitmix(&s)), Age: int64(splitmix(&s) % 8),
		BestBid: int64(splitmix(&s)), BestAsk: int64(splitmix(&s)),
		HasBid: flags&1 == 1, HasAsk: flags&2 == 2,
	}
	return b, bd, o, q, int64(splitmix(&s)), int64(splitmix(&s)), splitmix(&s)
}

// TestUpdate_PropertyInvariant verifies over a fixed seed range that the
// invariant of 3.3 holds after every update for any inputs.
//
// Requirement: FR-BEL-002, FR-BEL-003, FR-BEL-017
func TestUpdate_PropertyInvariant(t *testing.T) {
	t.Parallel()
	p := specParams()
	for seed := uint64(1); seed <= 20000; seed++ {
		b, bd, o, q, stock, target, u := randomCase(seed)
		got, res := beliefs.Update(b, bd, o, q, stock, target, p, u)
		if got.Centre < bd.Floor || got.Centre > bd.Ceiling || got.Centre < 1 || got.Width < p.WidthMin || got.Width > p.WidthMax {
			t.Fatalf("seed %d: Update(%+v, %+v) = %+v outside bounds %+v", seed, b, o, got, bd)
		}
		if res.Open != beliefs.Gate(u, p.AdjustProb) {
			t.Fatalf("seed %d: Open = %v, Gate = %v", seed, res.Open, beliefs.Gate(u, p.AdjustProb))
		}
		if again, _ := beliefs.Update(b, bd, o, q, stock, target, p, u); again != got {
			t.Fatalf("seed %d: Update not deterministic: %+v then %+v", seed, got, again)
		}
	}
}

// TestUpdate_PropertyBoundedStep verifies over a fixed seed range that, far
// from both bounds, the centre moves by at most de + delta.
//
// Requirement: FR-BEL-015
func TestUpdate_PropertyBoundedStep(t *testing.T) {
	t.Parallel()
	p := specParams()
	bd := beliefs.Bounds{Floor: 1, Ceiling: maxPrice}
	for seed := uint64(1); seed <= 20000; seed++ {
		_, _, o, q, stock, target, u := randomCase(seed)
		s := seed
		centre := 1_000_000 + int64(splitmix(&s)%100_000_000)
		q.Price, q.BestBid, q.BestAsk = centre+int64(splitmix(&s)%2_000_000)-1_000_000, centre-int64(splitmix(&s)%2_000_000), centre+int64(splitmix(&s)%2_000_000)
		o.Limit = centre + int64(splitmix(&s)%2_000_000) - 1_000_000
		stock, target = stock%100, target%100
		delta := max(1, centre*p.Step/10000)
		de := max(1, delta*p.EvidenceShare/10000)
		got, _ := beliefs.Update(beliefs.Belief{Centre: centre, Width: 2000}, bd, o, q, stock, target, p, u)
		d := got.Centre - centre
		if d < -(de+delta) || d > de+delta {
			t.Fatalf("seed %d: centre %d moved to %d, step %d above %d", seed, centre, got.Centre, d, de+delta)
		}
	}
}

// TestUpdate_PropertyStationary verifies over a fixed seed range that the
// centre does not change with a closed gate, or with no target and a stock
// ratio in the band.
//
// Requirement: FR-BEL-009, FR-BEL-014
func TestUpdate_PropertyStationary(t *testing.T) {
	t.Parallel()
	p := specParams()
	for seed := uint64(1); seed <= 20000; seed++ {
		b, bd, o, q, stock, target, u := randomCase(seed)
		b, _ = beliefs.Clamp(b, bd, p)
		closed := u | 1<<63 // high word of u * 10 000 is at least 5 000
		if got, _ := beliefs.Update(b, bd, o, q, stock, target, p, closed); got != b {
			t.Fatalf("seed %d: closed gate moved %+v to %+v", seed, b, got)
		}
		q.HasBid, q.HasAsk, q.Age = false, false, 9 // stale, no quote: no target
		if got, _ := beliefs.Update(b, bd, o, q, 10, 10, p, 0); got.Centre != b.Centre {
			t.Fatalf("seed %d: no target, ratio 10 000 moved centre %d to %d", seed, b.Centre, got.Centre)
		}
	}
}

// TestUpdate_PropertyScaleFree verifies over a fixed seed range that scaling
// the centre, the bounds and every price by k scales the new centre by k and
// leaves the width, and that scaling the quantities, fill, stock and target by
// a weight changes nothing.
//
// Requirement: FR-BEL-018, FR-BEL-025
func TestUpdate_PropertyScaleFree(t *testing.T) {
	t.Parallel()
	p := specParams()
	for seed := uint64(1); seed <= 5000; seed++ {
		s := seed
		// A centre of 5 000 * m makes delta = 100m, de = 50m and the
		// significance threshold 250m exact, so no truncation occurs at k.
		m := int64(splitmix(&s)%1000) + 1
		centre := 5000 * m
		k := int64(splitmix(&s)%6) + 2
		price := func() int64 { return centre + int64(splitmix(&s)%uint64(centre)) - centre/2 }
		o := beliefs.Outcome{Bid: splitmix(&s)&1 == 0, Limit: price(), Quantity: int64(splitmix(&s)%20) + 1}
		o.Filled = int64(splitmix(&s)) % (o.Quantity + 1)
		if o.Filled < 0 {
			o.Filled = -o.Filled
		}
		q := beliefs.Quote{Price: price(), Age: int64(splitmix(&s) % 6), BestBid: price(), BestAsk: price(), HasBid: splitmix(&s)&1 == 0, HasAsk: splitmix(&s)&1 == 0}
		stock, target := int64(splitmix(&s)%40), int64(splitmix(&s)%20)
		bd := beliefs.Bounds{Floor: centre / 2, Ceiling: centre * 2}
		b := beliefs.Belief{Centre: centre, Width: 2000}
		base, _ := beliefs.Update(b, bd, o, q, stock, target, p, 0)

		sb := beliefs.Bounds{Floor: bd.Floor * k, Ceiling: bd.Ceiling * k}
		so, sq := o, q
		so.Limit *= k
		sq.Price, sq.BestBid, sq.BestAsk = q.Price*k, q.BestBid*k, q.BestAsk*k
		got, _ := beliefs.Update(beliefs.Belief{Centre: centre * k, Width: 2000}, sb, so, sq, stock, target, p, 0)
		if got.Centre != base.Centre*k || got.Width != base.Width {
			t.Fatalf("seed %d: scaled by %d gave %+v, want centre %d width %d", seed, k, got, base.Centre*k, base.Width)
		}

		wo := o
		wo.Quantity, wo.Filled = o.Quantity*k, o.Filled*k
		if cohort, _ := beliefs.Update(b, bd, wo, q, stock*k, target*k, p, 0); cohort != base {
			t.Fatalf("seed %d: cohort weight %d gave %+v, want %+v", seed, k, cohort, base)
		}
	}
}

// encode packs values as little-endian words, the layout FuzzUpdate decodes.
func encode(vals ...int64) []byte {
	b := make([]byte, 0, 8*len(vals))
	for _, v := range vals {
		b = binary.LittleEndian.AppendUint64(b, uint64(v))
	}
	return b
}

// FuzzUpdate verifies that Update never panics and keeps the invariant and the
// bounded step for any belief, outcome, quote, stock, target and draw. The
// bounds are made valid first, because FR-BEL-002 is for inputs within
// configuration bounds.
//
// Requirement: FR-BEL-002, FR-BEL-003, FR-BEL-006, FR-BEL-015
func FuzzUpdate(f *testing.F) {
	// Fields: centre, width, limit, quantity, filled, price, age, best bid,
	// best ask, flags (1 bid, 2 has bid, 4 has ask), stock, target, floor seed,
	// ceiling seed, draw.
	f.Add(encode(1000, 2000, 940, 10, 0, 1000, 5, 0, 1100, 5, 2, 10, 99, 9900, 0))                                                                                                                  // case A
	f.Add(encode(1000, 2000, 940, 4, 4, 1010, 0, 0, 0, 0, 8, 8, 99, 9900, 0))                                                                                                                       // case B
	f.Add(encode(1140, 2000, 1160, 5, 0, 1000, 0, 990, 0, 2, 20, 10, 1119, 3680, 0))                                                                                                                // case C
	f.Add(encode(1000, 2000, 880, 10, 0, 1000, 5, 0, 950, 5, 10, 10, 99, 9900, 0))                                                                                                                  // case D
	f.Add(encode(math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, 7, math.MinInt64, math.MinInt64, 0, 0, -1)) // extremes low
	f.Add(encode(math.MaxInt64, math.MaxInt64, math.MaxInt64, math.MaxInt64, math.MaxInt64, math.MaxInt64, 0, math.MaxInt64, math.MaxInt64, 7, math.MaxInt64, 1, 5, 5, -1))                         // extremes high
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		word := func(i int) int64 {
			if len(data) < 8*(i+1) {
				return 0
			}
			return int64(binary.LittleEndian.Uint64(data[8*i:]))
		}
		p := specParams()
		floor := 1 + int64(uint64(word(12))%uint64(p.MaxPrice))
		ceiling := floor + int64(uint64(word(13))%uint64(p.MaxPrice-floor+1))
		bd := beliefs.Bounds{Floor: floor, Ceiling: ceiling}
		b := beliefs.Belief{Centre: word(0), Width: word(1)}
		flags := word(9)
		o := beliefs.Outcome{Bid: flags&1 == 1, Limit: word(2), Quantity: word(3), Filled: word(4)}
		q := beliefs.Quote{Price: word(5), Age: word(6), BestBid: word(7), BestAsk: word(8), HasBid: flags&2 == 2, HasAsk: flags&4 == 4}
		u := uint64(word(14))

		got, res := beliefs.Update(b, bd, o, q, word(10), word(11), p, u)
		if got.Centre < bd.Floor || got.Centre > bd.Ceiling || got.Width < p.WidthMin || got.Width > p.WidthMax {
			t.Fatalf("Update(%+v, %+v) = %+v outside %+v", b, bd, got, bd)
		}
		if res.Open != beliefs.Gate(u, p.AdjustProb) {
			t.Fatalf("Open = %v, Gate = %v for u %d", res.Open, beliefs.Gate(u, p.AdjustProb), u)
		}
		// The step is bounded from the re-clamped centre.
		c0, _ := beliefs.Clamp(b, bd, p)
		delta := max(1, c0.Centre*p.Step/10000)
		de := max(1, delta*p.EvidenceShare/10000)
		if d := got.Centre - c0.Centre; d > de+delta || d < -(de+delta) {
			t.Fatalf("Update(%+v) moved centre %d to %d, step above %d", b, c0.Centre, got.Centre, de+delta)
		}
	})
}

// TestUpdateGateChange_NoAllocs verifies that the update, the gate and the
// reads of a Change allocate nothing.
//
// Requirement: NFR-BEL-001, NFR-BEL-003
func TestUpdateGateChange_NoAllocs(t *testing.T) {
	p := specParams()
	b := beliefs.Belief{Centre: 1000, Width: 2000}
	o := beliefs.Outcome{Bid: true, Limit: 940, Quantity: 10}
	q := beliefs.Quote{Price: 1000, Age: 5, BestAsk: 1100, HasAsk: true}
	c := beliefs.Change{Before: b, After: beliefs.Belief{Centre: 1010}, Bounds: wideBounds}
	var sink int64
	var flag bool
	tests := []struct {
		name string
		fn   func()
	}{
		{"Update", func() { n, _ := beliefs.Update(b, wideBounds, o, q, 2, 10, p, 0); sink += n.Centre }},
		{"Gate", func() { flag = beliefs.Gate(12345, 5000) }},
		{"Change reads", func() { flag = c.Raised() || c.Lowered() || c.Moved() || c.AtFloor() || c.AtCeiling() || c.Conflict() }},
	}
	for _, tt := range tests {
		if got := testing.AllocsPerRun(100, tt.fn); got != 0 {
			t.Errorf("%s allocs = %v, want 0", tt.name, got)
		}
	}
	_, _ = sink, flag
}

// BenchmarkUpdate reports the cost and allocations of one update (it gates
// nothing: TestUpdateGateChange_NoAllocs does).
func BenchmarkUpdate(b *testing.B) {
	p := specParams()
	in := beliefs.Belief{Centre: 1000, Width: 2000}
	o := beliefs.Outcome{Bid: true, Limit: 940, Quantity: 10}
	q := beliefs.Quote{Price: 1000, Age: 5, BestAsk: 1100, HasAsk: true}
	b.ReportAllocs()
	for b.Loop() {
		beliefs.Update(in, wideBounds, o, q, 2, 10, p, 0)
	}
}
