package beliefs_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
)

// TestUpdate_EvidenceMoveCases verifies the evidence move at the edges a
// gomu run found unasserted: the choice between two candidate targets, a filled
// order on the favourable side of the market, a fall by the nearer gap, and a
// gap that saturates at the int64 limit.
//
// Every case keeps the stock ratio at 10 000 (inside the band) so only the
// evidence move acts, and opens the gate (draw 0).
//
// Requirement: FR-BEL-011, FR-BEL-013, FR-BEL-006, FR-BEL-036
func TestUpdate_EvidenceMoveCases(t *testing.T) {
	t.Parallel()
	// Step 2 500 and an evidence share of 10 000 make de equal to delta (at
	// least 175 here), larger than every gap below, so the gap sets the move.
	big := specParams()
	big.Step, big.EvidenceShare = 2500, 10000
	tests := []struct {
		name   string
		p      beliefs.Params
		in     beliefs.Belief
		o      beliefs.Outcome
		q      beliefs.Quote
		wantCt int64
	}{
		// Bid, limit 800, centre 700: best ask 900 and fresh price 850 (above the
		// limit) are candidates; the smaller, 850, is the target. gL = 50,
		// gC = 150: move +min(50, 150, 175) = +50 -> 750. Taking 900 would give 800.
		{name: "bid takes the smaller of ask and price", p: big, in: beliefs.Belief{Centre: 700, Width: 2000},
			o: beliefs.Outcome{Bid: true, Limit: 800, Quantity: 10},
			q: beliefs.Quote{Price: 850, Age: 0, BestAsk: 900, HasAsk: true}, wantCt: 750},
		// Ask, limit 1 200, centre 1 300: best bid 1 050 and fresh price 1 100
		// (below the limit); the larger, 1 100, is the target. gL = -100,
		// gC = -200: move -min(100, 200, 325) = -100 -> 1 200. Taking 1 050
		// would give 1 150.
		{name: "ask takes the larger of bid and price", p: big, in: beliefs.Belief{Centre: 1300, Width: 2000},
			o: beliefs.Outcome{Bid: false, Limit: 1200, Quantity: 10},
			q: beliefs.Quote{Price: 1100, Age: 0, BestBid: 1050, HasBid: true}, wantCt: 1200},
		// Filled bid, limit 900 below the price 1 000 (the favourable side): the
		// gap limit - price = -100 is not above the threshold, so no target and
		// no move. A target of 1 000 would lift the centre 950 by min(100, 50, 9).
		{name: "filled bid below the price has no target", p: specParams(), in: beliefs.Belief{Centre: 950, Width: 2000},
			o: beliefs.Outcome{Bid: true, Limit: 900, Quantity: 4, Filled: 4},
			q: beliefs.Quote{Price: 1000, Age: 0}, wantCt: 950},
		// Ask, limit 1 200, centre 1 300, stale price, best bid 1 100: gL = -100,
		// gC = -200, de = 13 (delta 26): move -min(100, 200, 13) = -13 -> 1 287.
		// The magnitude of the nearer gap, not the signed minimum, bounds a fall.
		{name: "fall is bounded by the smaller gap", p: specParams(), in: beliefs.Belief{Centre: 1300, Width: 2000},
			o: beliefs.Outcome{Bid: false, Limit: 1200, Quantity: 10},
			q: beliefs.Quote{Price: 1000, Age: 9, BestBid: 1100, HasBid: true}, wantCt: 1287},
		// Bid with limit MaxInt64 and best ask -1: gap(-1, MaxInt64) is
		// -MaxInt64 (saturated, not MinInt64, so it can be negated). gC = -1 001,
		// de = 10 (delta 20): move -min(MaxInt64, 1001, 10) = -10 -> 990.
		{name: "gap of exactly MinInt64 saturates", p: specParams(), in: beliefs.Belief{Centre: 1000, Width: 2000},
			o: beliefs.Outcome{Bid: true, Limit: math.MaxInt64, Quantity: 10},
			q: beliefs.Quote{Price: 1000, Age: 9, BestAsk: -1, HasAsk: true}, wantCt: 990},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, res := beliefs.Update(tt.in, wideBounds, tt.o, tt.q, 8, 8, tt.p, openGate)
			if !res.Open || got.Centre != tt.wantCt {
				t.Errorf("Update(%+v, %+v, %+v) = %+v open=%v, want centre %d", tt.in, tt.o, tt.q, got, res.Open, tt.wantCt)
			}
		})
	}
}

// TestLimits_ConflictNeedsStrictExcess verifies that the conflict flag is set
// only when the cost floor is strictly above the reference band's upper end.
//
// Requirement: FR-BEL-004
func TestLimits_ConflictNeedsStrictExcess(t *testing.T) {
	t.Parallel()
	// ref 1 000 and RefBandHigh 100 000 give refHigh 10 000; unit cost 10 000
	// with CostFloor 10 000 gives costLow 10 000: equal, so no conflict.
	p := specParams()
	p.CostFloor = 10000
	for _, tt := range []struct {
		unitCost int64
		want     bool
	}{{10000, false}, {10001, true}} {
		if got := beliefs.Limits(1000, tt.unitCost, p).Conflict; got != tt.want {
			t.Errorf("Limits(1000, %d).Conflict = %v, want %v", tt.unitCost, got, tt.want)
		}
	}
}

// TestChange_BoundReadsAreExact verifies that a centre beyond a bound is not on
// it: AtFloor and AtCeiling test equality, not a side.
//
// Guard: a Change is built by the world from a clamped belief, but the reads are
// pure; a loosened comparison would count a stray centre as pinned.
func TestChange_BoundReadsAreExact(t *testing.T) {
	t.Parallel()
	bd := beliefs.Bounds{Floor: 100, Ceiling: 200}
	if c := (beliefs.Change{After: beliefs.Belief{Centre: 50}, Bounds: bd}); c.AtFloor() {
		t.Errorf("Change{After 50, floor 100}.AtFloor() = true, want false")
	}
	if c := (beliefs.Change{After: beliefs.Belief{Centre: 300}, Bounds: bd}); c.AtCeiling() {
		t.Errorf("Change{After 300, ceiling 200}.AtCeiling() = true, want false")
	}
}

// TestNextRun_Guards verifies the majority test and the every guard of the
// diagnostic run.
//
// Requirement: FR-BEL-031
func TestNextRun_Guards(t *testing.T) {
	t.Parallel()
	// 6 of 10 outcomes on a bound: 6 > 10/2 = 5, pinned. A subtraction in place
	// of the division (10 - 2 = 8) would not see it.
	if r := beliefs.NextRun(0, &beliefs.Tally{Outcomes: 10, AtFloor: 6}, 10); r.Count != 1 || r.Agents != 6 {
		t.Errorf("NextRun(0, 6 of 10 on floor, 10) = %+v, want Count 1, Agents 6", r)
	}
	// An every of 0 or below never warns; 0 must not divide by zero.
	for _, every := range []int64{0, -1} {
		if r := beliefs.NextRun(4, &beliefs.Tally{Outcomes: 2, AtFloor: 2}, every); r.Warn {
			t.Errorf("NextRun(4, pinned, %d).Warn = true, want false", every)
		}
	}
}

// TestWindow_AcceptedEdges verifies that a window reports a value as present
// after a push and that Restore accepts the extreme prices 1 and maxPrice.
//
// Requirement: FR-BEL-020
func TestWindow_AcceptedEdges(t *testing.T) {
	t.Parallel()
	w := beliefs.NewWindow(3, 100)
	w.Push(40)
	if v, ok := w.Low(); v != 40 || !ok {
		t.Errorf("Low() after Push(40) = %d, %v, want 40, true", v, ok)
	}
	if v, ok := w.High(); v != 40 || !ok {
		t.Errorf("High() after Push(40) = %d, %v, want 40, true", v, ok)
	}
	if !w.Restore([]int64{1, 100}) {
		t.Errorf("Restore([1 100]) with maxPrice 100 = false, want true")
	}
}

// TestPosition_InvertedRange verifies that a window whose high is below its low
// reads as the middle, like an equal pair.
//
// Guard: the window never produces it, but the function is pure; the rule says
// "high at most low", not "high equal to low".
func TestPosition_InvertedRange(t *testing.T) {
	t.Parallel()
	if got := beliefs.Position(5, 10, 5); got != 5000 {
		t.Errorf("Position(5, 10, 5) = %d, want 5000", got)
	}
}

// TestUpdate_PriceAtLimitIsNotACandidate verifies that a fresh market price
// equal to the limit is not a target candidate of an unfilled order: only a
// price strictly beyond the limit is.
//
// Requirement: FR-BEL-011, FR-BEL-013
func TestUpdate_PriceAtLimitIsNotACandidate(t *testing.T) {
	t.Parallel()
	// Step 2 500 and an evidence share of 10 000 make de larger than the gaps.
	p := specParams()
	p.Step, p.EvidenceShare = 2500, 10000
	tests := []struct {
		name   string
		in     beliefs.Belief
		o      beliefs.Outcome
		q      beliefs.Quote
		wantCt int64
	}{
		// Bid, limit 800, price 800, best ask 900: the target is 900 (gL 100,
		// gC 200: move +100 -> 800). Counting the price as a candidate would make
		// the target 800, gL 0 and no move.
		{name: "bid", in: beliefs.Belief{Centre: 700, Width: 2000},
			o: beliefs.Outcome{Bid: true, Limit: 800, Quantity: 10},
			q: beliefs.Quote{Price: 800, BestAsk: 900, HasAsk: true}, wantCt: 800},
		// Ask, limit 800, price 800, best bid 700: the target is 700 (move -100
		// -> 800); the price as a candidate would leave the centre at 900.
		{name: "ask", in: beliefs.Belief{Centre: 900, Width: 2000},
			o: beliefs.Outcome{Limit: 800, Quantity: 10},
			q: beliefs.Quote{Price: 800, BestBid: 700, HasBid: true}, wantCt: 800},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _ := beliefs.Update(tt.in, wideBounds, tt.o, tt.q, 8, 8, p, openGate)
			if got.Centre != tt.wantCt {
				t.Errorf("Update(%+v, %+v, %+v) centre = %d, want %d", tt.in, tt.o, tt.q, got.Centre, tt.wantCt)
			}
		})
	}
}

// TestUpdate_SingleUnitOrderAndAgeEdge verifies two boundaries: an order of
// one unit that is filled counts as fully filled (share 10 000, so the width
// narrows), and a price whose age equals MaxAge is still fresh.
//
// Requirement: FR-BEL-011, FR-BEL-012, FR-BEL-016
func TestUpdate_SingleUnitOrderAndAgeEdge(t *testing.T) {
	t.Parallel()
	p := specParams()
	p.Step, p.EvidenceShare = 2500, 10000
	tests := []struct {
		name              string
		o                 beliefs.Outcome
		q                 beliefs.Quote
		wantCt, wantWidth int64
	}{
		// Quantity 1, Filled 1: share 10 000, width 2 000 - 250.
		{name: "one unit filled narrows", o: beliefs.Outcome{Bid: true, Limit: 800, Quantity: 1, Filled: 1},
			q: beliefs.Quote{Price: 800, Age: 0}, wantCt: 700, wantWidth: 1750},
		// Unfilled bid, price 850 above the limit at age MaxAge (3): fresh, so
		// the target is 850 and the centre rises by 50.
		{name: "age at MaxAge is fresh", o: beliefs.Outcome{Bid: true, Limit: 800, Quantity: 10},
			q: beliefs.Quote{Price: 850, Age: p.MaxAge}, wantCt: 750, wantWidth: 2250},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _ := beliefs.Update(beliefs.Belief{Centre: 700, Width: 2000}, wideBounds, tt.o, tt.q, 8, 8, p, openGate)
			if got.Centre != tt.wantCt || got.Width != tt.wantWidth {
				t.Errorf("Update(%+v, %+v) = %+v, want centre %d width %d", tt.o, tt.q, got, tt.wantCt, tt.wantWidth)
			}
		})
	}
}

// TestUpdate_SumSaturatesUpward verifies that a centre or a width sum that does
// not fit int64 saturates at MaxInt64 (then clamps to the ceiling), not at
// MinInt64.
//
// Requirement: FR-BEL-036
func TestUpdate_SumSaturatesUpward(t *testing.T) {
	t.Parallel()
	p := specParams()
	p.WidthMax = math.MaxInt64
	bd := beliefs.Bounds{Floor: 1, Ceiling: math.MaxInt64}
	in := beliefs.Belief{Centre: math.MaxInt64, Width: math.MaxInt64}
	// Stock 0 against target 8: short, so the centre moves up by delta; an
	// unfilled order widens. Both sums overflow.
	got, _ := beliefs.Update(in, bd, beliefs.Outcome{Bid: true, Limit: 1, Quantity: 10}, beliefs.Quote{}, 0, 8, p, openGate)
	if got.Centre != math.MaxInt64 || got.Width != math.MaxInt64 {
		t.Errorf("Update at the int64 limit = %+v, want both at MaxInt64", got)
	}
}

// TestTally_AddCountsEveryField verifies each counter and the centre range of
// a tally over six outcomes, including a belief whose floor equals its ceiling
// (counted at both bounds) and a first outcome whose centre is above zero.
//
// Requirement: FR-BEL-030
func TestTally_AddCountsEveryField(t *testing.T) {
	t.Parallel()
	wide := beliefs.Bounds{Floor: 100, Ceiling: 1000}
	conf := beliefs.Bounds{Floor: 100, Ceiling: 1000, Conflict: true}
	pin := beliefs.Bounds{Floor: 700, Ceiling: 700}
	at := func(before, after int64, b beliefs.Bounds, open, clamped bool) beliefs.Change {
		return beliefs.Change{Before: beliefs.Belief{Centre: before}, After: beliefs.Belief{Centre: after},
			Bounds: b, Open: open, Clamped: clamped}
	}
	changes := []beliefs.Change{
		at(400, 500, wide, true, false),
		at(400, 300, conf, true, true),
		at(400, 100, conf, true, false),
		at(400, 1000, wide, false, true),
		at(400, 900, conf, true, false),
		at(700, 700, pin, false, false),
	}
	var tl beliefs.Tally
	for _, c := range changes {
		tl.Add(c)
	}
	got := [8]int64{tl.Outcomes, tl.Adjusted, tl.Raised, tl.Lowered, tl.AtFloor, tl.AtCeiling, tl.Conflicts, tl.Clamped}
	want := [8]int64{6, 4, 3, 2, 2, 2, 3, 2}
	if got != want {
		t.Errorf("counts (outcomes, adjusted, raised, lowered, floor, ceiling, conflicts, clamped) = %v, want %v", got, want)
	}
	if tl.Low() != 100 || tl.High() != 1000 {
		t.Errorf("Low, High = %d, %d, want 100, 1000", tl.Low(), tl.High())
	}
	// 500+300+100+1000+900+700 = 3500; 3500 / 6 = 583.33.
	if m, ok := tl.Mean(); !ok || m != 583 {
		t.Errorf("Mean() = %d, %v, want 583, true", m, ok)
	}
}

// TestTally_EmptyAndMeanRounding verifies the empty tally and that the mean
// rounds half to even.
//
// Requirement: FR-BEL-030
func TestTally_EmptyAndMeanRounding(t *testing.T) {
	t.Parallel()
	var empty beliefs.Tally
	if m, ok := empty.Mean(); ok || m != 0 {
		t.Errorf("empty Mean() = %d, %v, want 0, false", m, ok)
	}
	if empty.Low() != 0 || empty.High() != 0 {
		t.Errorf("empty Low, High = %d, %d, want 0, 0", empty.Low(), empty.High())
	}
	for _, tt := range []struct {
		name    string
		centres []int64
		want    int64
	}{
		{"tie goes up to even", []int64{1, 2}, 2},
		{"tie goes down to even", []int64{2, 3}, 2},
		{"negative centre counts as zero in the sum", []int64{-50, 100}, 50},
	} {
		var tl beliefs.Tally
		for _, c := range tt.centres {
			tl.Add(beliefs.Change{After: beliefs.Belief{Centre: c}})
		}
		if m, ok := tl.Mean(); !ok || m != tt.want {
			t.Errorf("%s: Mean() = %d, %v, want %d, true", tt.name, m, ok, tt.want)
		}
	}
	var neg beliefs.Tally
	neg.Add(beliefs.Change{After: beliefs.Belief{Centre: -50}})
	neg.Add(beliefs.Change{After: beliefs.Belief{Centre: 100}})
	if neg.Low() != -50 || neg.High() != 100 {
		t.Errorf("Low, High = %d, %d, want -50, 100", neg.Low(), neg.High())
	}
}

// TestNextRun_Table verifies the pinned test, the count, the bound named and
// the warn cadence of the diagnostic run.
//
// Requirement: FR-BEL-031
func TestNextRun_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		before int64
		t      beliefs.Tally
		every  int64
		want   beliefs.Run
	}{
		{"no outcome resets even with stray counts", 5, beliefs.Tally{AtFloor: 1}, 10, beliefs.Run{}},
		{"exactly half is not pinned", 5, beliefs.Tally{Outcomes: 10, AtFloor: 5}, 10, beliefs.Run{}},
		{"odd outcomes: 5 of 9 is pinned", 0, beliefs.Tally{Outcomes: 9, AtFloor: 5}, 10,
			beliefs.Run{Count: 1, Agents: 5, Outcomes: 9}},
		{"4 of 9 is not pinned", 3, beliefs.Tally{Outcomes: 9, AtFloor: 4}, 10, beliefs.Run{}},
		{"floor and ceiling add", 0, beliefs.Tally{Outcomes: 6, AtFloor: 3, AtCeiling: 1}, 10,
			beliefs.Run{Count: 1, Agents: 3, Outcomes: 6}},
		{"count advances by one", 3, beliefs.Tally{Outcomes: 4, AtFloor: 3}, 10,
			beliefs.Run{Count: 4, Agents: 3, Outcomes: 4}},
		{"count saturates", math.MaxInt64, beliefs.Tally{Outcomes: 4, AtFloor: 3}, 10,
			beliefs.Run{Count: math.MaxInt64, Agents: 3, Outcomes: 4}},
		{"ceiling with more beliefs", 0, beliefs.Tally{Outcomes: 8, AtFloor: 2, AtCeiling: 4}, 10,
			beliefs.Run{Count: 1, Ceiling: true, Agents: 4, Outcomes: 8}},
		{"tie names the floor", 0, beliefs.Tally{Outcomes: 8, AtFloor: 3, AtCeiling: 3}, 10,
			beliefs.Run{Count: 1, Agents: 3, Outcomes: 8}},
		{"floor with more beliefs", 0, beliefs.Tally{Outcomes: 8, AtFloor: 4, AtCeiling: 2}, 10,
			beliefs.Run{Count: 1, Agents: 4, Outcomes: 8}},
		{"warns at the tenth", 9, beliefs.Tally{Outcomes: 2, AtFloor: 2}, 10,
			beliefs.Run{Count: 10, Warn: true, Agents: 2, Outcomes: 2}},
		{"no warn at the ninth", 8, beliefs.Tally{Outcomes: 2, AtFloor: 2}, 10,
			beliefs.Run{Count: 9, Agents: 2, Outcomes: 2}},
		{"no warn at the eleventh", 10, beliefs.Tally{Outcomes: 2, AtFloor: 2}, 10,
			beliefs.Run{Count: 11, Agents: 2, Outcomes: 2}},
		{"warns again at the twentieth", 19, beliefs.Tally{Outcomes: 2, AtFloor: 2}, 10,
			beliefs.Run{Count: 20, Warn: true, Agents: 2, Outcomes: 2}},
		{"every of 1 warns each tick", 0, beliefs.Tally{Outcomes: 2, AtFloor: 2}, 1,
			beliefs.Run{Count: 1, Warn: true, Agents: 2, Outcomes: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := beliefs.NextRun(tt.before, &tt.t, tt.every); got != tt.want {
				t.Errorf("NextRun(%d, %+v, %d) = %+v, want %+v", tt.before, tt.t, tt.every, got, tt.want)
			}
		})
	}
}

// TestReseed_KeptWinsAndWidthInit verifies that Kept takes precedence over the
// window fields (a kept belief is not restarted even when Empty or Mean is set),
// that a restart takes its width from WidthInit rather than a constant, and that
// the bounds use Ref and UnitCost in that order.
//
// Requirement: FR-BEL-022, FR-BEL-023
func TestReseed_KeptWinsAndWidthInit(t *testing.T) {
	t.Parallel()
	p := defaults()
	p.WidthInit = 1234
	// ref 1 000, unit cost 1 600: bounds [1 120, 4 800] (see TestReseed).
	tests := []struct {
		name string
		seed beliefs.Seed
		want beliefs.Belief
	}{
		{"kept ignores Empty and Price", beliefs.Seed{Ref: 1000, UnitCost: 1600, Kept: true, Empty: true, Price: 2000, Mean: 3000, Belief: beliefs.Belief{Centre: 4000, Width: 900}}, beliefs.Belief{Centre: 4000, Width: 900}},
		{"restart width is WidthInit", beliefs.Seed{Ref: 1000, UnitCost: 1600, Mean: 2500}, beliefs.Belief{Centre: 2500, Width: 1234}},
		{"empty restart width is WidthInit", beliefs.Seed{Ref: 1000, UnitCost: 1600, Empty: true, Price: 2500, Mean: 3000}, beliefs.Belief{Centre: 2500, Width: 1234}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := beliefs.Reseed(nil, []beliefs.Seed{tt.seed}, p)
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("Reseed(%+v) = %v, want [%v]", tt.seed, got, tt.want)
			}
		})
	}
}
