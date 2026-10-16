package beliefs_test

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
)

// TestReseed verifies the start of a belief at the seed price, the window mean
// or a kept belief, always clamped to the bounds of the caller's unit cost.
//
// Requirement: FR-BEL-022, FR-BEL-023
func TestReseed(t *testing.T) {
	t.Parallel()
	p := defaults()
	// ref 1 000: reference band 1 000*1 000/10 000 = 100 to 1 000*100 000/10 000 =
	// 10 000. With unit cost 1 600 the cost band is 1 120 to 4 800, so the bounds
	// are [1 120, 4 800].
	tests := []struct {
		name string
		seed beliefs.Seed
		want beliefs.Belief
	}{
		{"initial agent starts at the seed price", beliefs.Seed{Ref: 1000, Empty: true, Price: 1000, Mean: 777}, beliefs.Belief{Centre: 1000, Width: 2000}},
		{"empty window uses the market price", beliefs.Seed{Ref: 1000, UnitCost: 1600, Empty: true, Price: 3000, Mean: 777}, beliefs.Belief{Centre: 3000, Width: 2000}},
		{"window mean wins over the price", beliefs.Seed{Ref: 1000, UnitCost: 1600, Mean: 2000, Price: 3000}, beliefs.Belief{Centre: 2000, Width: 2000}},
		{"mean below the cost floor is clamped to 1 120", beliefs.Seed{Ref: 1000, UnitCost: 1600, Mean: 900, Price: 900}, beliefs.Belief{Centre: 1120, Width: 2000}},
		{"price above the ceiling is clamped to 4 800", beliefs.Seed{Ref: 1000, UnitCost: 1600, Empty: true, Price: 9000}, beliefs.Belief{Centre: 4800, Width: 2000}},
		{"kept belief inside the bounds is unchanged", beliefs.Seed{Ref: 1000, UnitCost: 1600, Kept: true, Belief: beliefs.Belief{Centre: 3000, Width: 3500}, Mean: 2000}, beliefs.Belief{Centre: 3000, Width: 3500}},
		{"kept belief is re-clamped, its width kept", beliefs.Seed{Ref: 1000, UnitCost: 1600, Kept: true, Belief: beliefs.Belief{Centre: 1000, Width: 3500}}, beliefs.Belief{Centre: 1120, Width: 3500}},
		{"kept width above the maximum is cut to 5 000", beliefs.Seed{Ref: 1000, Kept: true, Belief: beliefs.Belief{Centre: 1000, Width: 6000}}, beliefs.Belief{Centre: 1000, Width: 5000}},
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

// TestReseed_OrderAndAppend verifies that the beliefs come back in the order of
// the input, appended to dst, so a role change that adds, keeps and drops a
// commodity is one call over the new tradable set.
//
// Requirement: FR-BEL-023
func TestReseed_OrderAndAppend(t *testing.T) {
	t.Parallel()
	seeds := []beliefs.Seed{
		{Ref: 1000, Kept: true, Belief: beliefs.Belief{Centre: 4000, Width: 1000}},
		{Ref: 2000, Empty: true, Price: 2000},
		{Ref: 500, Mean: 600},
	}
	prefix := []beliefs.Belief{{Centre: 1, Width: 1}}
	got := beliefs.Reseed(prefix, seeds, defaults())
	want := []beliefs.Belief{{Centre: 1, Width: 1}, {Centre: 4000, Width: 1000}, {Centre: 2000, Width: 2000}, {Centre: 600, Width: 2000}}
	if !slices.Equal(got, want) {
		t.Errorf("Reseed = %v, want %v", got, want)
	}
}

// TestReseed_DroppedCommodity verifies a role change that keeps one commodity,
// adds one and drops one: the caller passes seeds for the new tradable set only,
// so the dropped commodity has no belief in the result.
//
// Requirement: FR-BEL-023
func TestReseed_DroppedCommodity(t *testing.T) {
	t.Parallel()
	// Old role trades A and B; new role trades B and C. A is dropped.
	oldA := beliefs.Belief{Centre: 7777, Width: 1234}
	oldB := beliefs.Belief{Centre: 3000, Width: 1000}
	seeds := []beliefs.Seed{
		{Ref: 1000, Kept: true, Belief: oldB},
		{Ref: 2000, Empty: true, Price: 2500},
	}
	got := beliefs.Reseed(nil, seeds, defaults())
	want := []beliefs.Belief{oldB, {Centre: 2500, Width: 2000}}
	if !slices.Equal(got, want) {
		t.Errorf("Reseed = %v, want %v", got, want)
	}
	if slices.Contains(got, oldA) {
		t.Errorf("Reseed kept the dropped belief %+v", oldA)
	}
}

// TestWindow_Statistics verifies the ring keeps the last capacity prices, and
// the mean (half to even), low and high.
//
// Requirement: FR-BEL-020
func TestWindow_Statistics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		capacity          int
		pushes            []int64
		wantValues        []int64
		wantMean, wantLow int64
		wantHigh          int64
	}{
		// 1.5 rounds to the even 2.
		{"mean of 1 and 2 is 2", 3, []int64{1, 2}, []int64{1, 2}, 2, 1, 2},
		// 2.5 rounds to the even 2.
		{"mean of 2 and 3 is 2", 3, []int64{2, 3}, []int64{2, 3}, 2, 2, 3},
		// the oldest, 10, is overwritten: (20+30+40)/3 = 30.
		{"overwrites the oldest beyond capacity", 3, []int64{10, 20, 30, 40}, []int64{20, 30, 40}, 30, 20, 40},
		// capacity 1 keeps the last price only.
		{"capacity 1 keeps the last", 1, []int64{5, 9}, []int64{9}, 9, 9, 9},
		// 1 000 pushes of 1..1 000 then 1 more: holds 2..1 001, sum 501 500, mean 501.5 -> 502 (even).
		{"capacity 1000 wraps once", 1000, seq(1, 1001), seq(2, 1001), 502, 2, 1001},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := beliefs.NewWindow(tt.capacity, maxPrice)
			for _, v := range tt.pushes {
				w.Push(v)
			}
			mean, ok := w.Mean()
			low, _ := w.Low()
			high, _ := w.High()
			if !ok || mean != tt.wantMean || low != tt.wantLow || high != tt.wantHigh {
				t.Errorf("window after %v: mean, low, high = %d, %d, %d (ok %v), want %d, %d, %d", tt.pushes, mean, low, high, ok, tt.wantMean, tt.wantLow, tt.wantHigh)
			}
			if got := w.Values(nil); !slices.Equal(got, tt.wantValues) {
				t.Errorf("Values = %v, want %v", got, tt.wantValues)
			}
			if w.Len() != len(tt.wantValues) {
				t.Errorf("Len = %d, want %d", w.Len(), len(tt.wantValues))
			}
		})
	}
}

func seq(from, to int64) []int64 {
	var s []int64
	for v := from; v <= to; v++ {
		s = append(s, v)
	}
	return s
}

// TestWindow_EmptyReportsNone verifies the statistics are absent while the
// window is empty.
//
// Requirement: FR-BEL-020, FR-BEL-039
func TestWindow_EmptyReportsNone(t *testing.T) {
	t.Parallel()
	w := beliefs.NewWindow(4, maxPrice)
	_, m := w.Mean()
	_, l := w.Low()
	_, h := w.High()
	if m || l || h || w.Len() != 0 || len(w.Values(nil)) != 0 {
		t.Errorf("empty window: mean, low, high ok = %v, %v, %v, Len %d, want none", m, l, h, w.Len())
	}
}

// TestWindow_PushClampsOutOfRange verifies a price of 0 and of MaxPrice+1 are
// stored as 1 and MaxPrice, each in its own slot.
//
// Requirement: FR-BEL-020
//
// Guard: the entries must stay one per traded tick; ignoring the price would
// shift the window and a wrap would corrupt the mean (spec-beliefs.md 7.5).
func TestWindow_PushClampsOutOfRange(t *testing.T) {
	t.Parallel()
	w := beliefs.NewWindow(5, 1000)
	w.Push(0)
	w.Push(1001)
	w.Push(math.MinInt64)
	w.Push(math.MaxInt64)
	if got, want := w.Values(nil), []int64{1, 1000, 1, 1000}; !slices.Equal(got, want) || w.Len() != 4 {
		t.Errorf("Values = %v (Len %d), want %v", got, w.Len(), want)
	}
}

// TestWindow_RestoreIsLossless verifies a window restored from the exported
// values equals the original in content and in how the next push behaves,
// whatever the write position was.
//
// Requirement: FR-BEL-035, FR-BEL-020
func TestWindow_RestoreIsLossless(t *testing.T) {
	t.Parallel()
	a := beliefs.NewWindow(3, maxPrice)
	for _, v := range []int64{7, 8, 9, 10, 11} { // wraps: write position is not 0
		a.Push(v)
	}
	b := beliefs.NewWindow(3, maxPrice)
	if !b.Restore(a.Values(nil)) {
		t.Fatalf("Restore(%v) refused", a.Values(nil))
	}
	if !reflect.DeepEqual(a.Values(nil), b.Values(nil)) {
		t.Errorf("restored values = %v, want %v", b.Values(nil), a.Values(nil))
	}
	am, _ := a.Mean()
	bm, _ := b.Mean()
	if am != bm {
		t.Errorf("restored mean = %d, want %d", bm, am)
	}
	a.Push(12)
	b.Push(12)
	// 8,9,10,11 -> after 12: 10, 11, 12 on both.
	if want := []int64{10, 11, 12}; !slices.Equal(a.Values(nil), want) || !slices.Equal(b.Values(nil), want) {
		t.Errorf("after one push: %v and %v, want %v", a.Values(nil), b.Values(nil), want)
	}
	// An empty restore empties the window.
	if !b.Restore(nil) || b.Len() != 0 {
		t.Errorf("Restore(nil) left Len %d", b.Len())
	}
}

// TestWindow_RestoreRefuses verifies a restore with too many values or a price
// outside [1, MaxPrice] is refused and leaves the window as it was.
//
// Requirement: FR-BEL-035
func TestWindow_RestoreRefuses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		vals []int64
	}{
		{"too long", []int64{1, 2, 3, 4}},
		{"price below 1", []int64{5, 0}},
		{"negative price", []int64{-3}},
		{"price above MaxPrice", []int64{5, 1001}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := beliefs.NewWindow(3, 1000)
			w.Push(42)
			if w.Restore(tt.vals) {
				t.Errorf("Restore(%v) = true, want refused", tt.vals)
			}
			if got := w.Values(nil); !slices.Equal(got, []int64{42}) {
				t.Errorf("window after refused Restore = %v, want [42]", got)
			}
		})
	}
}

// TestPosition verifies the position of a price in the window.
//
// Requirement: FR-BEL-021
func TestPosition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		price, low, high int64
		want             int64
	}{
		{"at low", 800, 800, 1200, 0},
		{"at high", 1200, 800, 1200, 10000},
		// (1 000-800)*10 000/400 = 5 000.
		{"midpoint", 1000, 800, 1200, 5000},
		// (900-800)*10 000/400 = 2 500.
		{"quarter", 900, 800, 1200, 2500},
		// 1*10 000/3 = 3 333.33 truncated.
		{"truncates", 1, 0, 3, 3333},
		{"below the window clamps to 0", 100, 800, 1200, 0},
		{"above the window clamps to 10 000", 5000, 800, 1200, 10000},
		{"high equals low is 5 000", 900, 1000, 1000, 5000},
		{"extreme range does not overflow", math.MaxInt64 - 1, 1, math.MaxInt64, 9999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := beliefs.Position(tt.price, tt.low, tt.high); got != tt.want {
				t.Errorf("Position(%d, %d, %d) = %d, want %d", tt.price, tt.low, tt.high, got, tt.want)
			}
		})
	}
}

// TestNextRun verifies the diagnostic run: reset, the tenth and every tenth
// pinned tick warn, exactly half is not pinned, the bound named is the one with
// more beliefs and the floor on a tie.
//
// Requirement: FR-BEL-031
func TestNextRun(t *testing.T) {
	t.Parallel()
	pinned := &beliefs.Tally{Outcomes: 4, AtFloor: 3}
	t.Run("pinned for nine ticks does not warn, the tenth and every tenth after does", func(t *testing.T) {
		t.Parallel()
		var run int64
		for tick := int64(1); tick <= 30; tick++ {
			r := beliefs.NextRun(run, pinned, 10)
			run = r.Count
			if want := tick%10 == 0; r.Warn != want || run != tick {
				t.Fatalf("tick %d: NextRun = %+v, want count %d, warn %v", tick, r, tick, want)
			}
		}
	})
	tests := []struct {
		name   string
		before int64
		tally  beliefs.Tally
		want   beliefs.Run
	}{
		{"not pinned resets", 7, beliefs.Tally{Outcomes: 4, AtFloor: 1}, beliefs.Run{}},
		{"no outcomes resets", 7, beliefs.Tally{}, beliefs.Run{}},
		// Outcomes/2 = 2: a sum of 2 is not above it.
		{"exactly half is not pinned", 3, beliefs.Tally{Outcomes: 4, AtFloor: 1, AtCeiling: 1}, beliefs.Run{}},
		// a sum of 3 is above 2.
		{"one more than half is pinned", 3, beliefs.Tally{Outcomes: 4, AtFloor: 2, AtCeiling: 1}, beliefs.Run{Count: 4, Agents: 2, Outcomes: 4}},
		{"ceiling with more beliefs is named", 3, beliefs.Tally{Outcomes: 4, AtFloor: 1, AtCeiling: 3}, beliefs.Run{Count: 4, Ceiling: true, Agents: 3, Outcomes: 4}},
		{"a tie names the floor", 8, beliefs.Tally{Outcomes: 4, AtFloor: 2, AtCeiling: 2}, beliefs.Run{Count: 9, Agents: 2, Outcomes: 4}},
		// Outcomes 1, a belief with Floor == Ceiling counts in both: 2 > 0.
		{"a belief at both bounds counts twice", 9, beliefs.Tally{Outcomes: 1, AtFloor: 1, AtCeiling: 1}, beliefs.Run{Count: 10, Warn: true, Agents: 1, Outcomes: 1}},
		{"count saturates", math.MaxInt64, beliefs.Tally{Outcomes: 1, AtFloor: 1}, beliefs.Run{Count: math.MaxInt64, Agents: 1, Outcomes: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := beliefs.NextRun(tt.before, &tt.tally, 10); got != tt.want {
				t.Errorf("NextRun(%d, %+v, 10) = %+v, want %+v", tt.before, tt.tally, got, tt.want)
			}
		})
	}
}

// TestTally_Counts verifies every count and the mean, low and high of a
// scripted set of outcomes, including a closed-gate re-clamp and a belief that
// no Update produced.
//
// Requirement: FR-BEL-030
func TestTally_Counts(t *testing.T) {
	t.Parallel()
	open := beliefs.Bounds{Floor: 100, Ceiling: 10000}
	cost := beliefs.Bounds{Floor: 1120, Ceiling: 4800}
	changes := []beliefs.Change{
		// raised by 30, gate open.
		{Before: beliefs.Belief{Centre: 1000}, After: beliefs.Belief{Centre: 1030}, Bounds: open, Open: true},
		// lowered by 10, gate open.
		{Before: beliefs.Belief{Centre: 1000}, After: beliefs.Belief{Centre: 990}, Bounds: open, Open: true},
		// closed gate on the floor with a conflict: unchanged, at floor.
		{Before: beliefs.Belief{Centre: 1120}, After: beliefs.Belief{Centre: 1120}, Bounds: beliefs.Bounds{Floor: 1120, Ceiling: 4800, Conflict: true}},
		// closed gate, stored centre 900 re-clamped up to the new floor 1 120 and
		// the clamp acted: raised and at floor but not adjusted.
		{Before: beliefs.Belief{Centre: 900}, After: beliefs.Belief{Centre: 1120}, Bounds: cost, Clamped: true},
	}
	var tally beliefs.Tally
	for _, c := range changes {
		tally.Add(c)
	}
	mean, ok := tally.Mean()
	// centres after: 1 030 + 990 + 1 120 + 1 120 = 4 260; 4 260 / 4 = 1 065.
	want := beliefs.Tally{Outcomes: 4, Adjusted: 2, Raised: 2, Lowered: 1, AtFloor: 2, Conflicts: 1, Clamped: 1}
	got := tally
	if got.Outcomes != want.Outcomes || got.Adjusted != want.Adjusted || got.Raised != want.Raised ||
		got.Lowered != want.Lowered || got.AtFloor != want.AtFloor || got.AtCeiling != want.AtCeiling ||
		got.Conflicts != want.Conflicts || got.Clamped != want.Clamped {
		t.Errorf("counts = %+v, want %+v", got, want)
	}
	if !ok || mean != 1065 || tally.Low() != 990 || tally.High() != 1120 {
		t.Errorf("mean, low, high = %d (ok %v), %d, %d, want 1065, 990, 1120", mean, ok, tally.Low(), tally.High())
	}
}

// TestTally_EdgeCases verifies an empty tally, the half-to-even mean, a custom
// strategy's belief and a belief with Floor equal to Ceiling.
//
// Requirement: FR-BEL-030
func TestTally_EdgeCases(t *testing.T) {
	t.Parallel()
	at := func(centre int64) beliefs.Change {
		return beliefs.Change{Before: beliefs.Belief{Centre: 50}, After: beliefs.Belief{Centre: centre}, Bounds: beliefs.Bounds{Floor: 1, Ceiling: 1000}}
	}
	t.Run("empty tally has no mean", func(t *testing.T) {
		t.Parallel()
		var tally beliefs.Tally
		if m, ok := tally.Mean(); ok || m != 0 || tally.Outcomes != 0 || tally.Low() != 0 || tally.High() != 0 {
			t.Errorf("empty Tally: Mean = %d, %v, want 0, false", m, ok)
		}
	})
	for _, tt := range []struct {
		name    string
		centres []int64
		want    int64
	}{
		// 1.5 -> 2 (even), 2.5 -> 2 (even).
		{"mean 1.5 rounds to 2", []int64{1, 2}, 2},
		{"mean 2.5 rounds to 2", []int64{2, 3}, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var tally beliefs.Tally
			for _, c := range tt.centres {
				tally.Add(at(c))
			}
			if m, ok := tally.Mean(); !ok || m != tt.want {
				t.Errorf("Mean over %v = %d, %v, want %d", tt.centres, m, ok, tt.want)
			}
		})
	}
	t.Run("a centre no Update produced is counted as raised", func(t *testing.T) {
		t.Parallel()
		// 50 -> 999 is far more than any Update would move; the tally reads only the Change.
		var tally beliefs.Tally
		tally.Add(at(999))
		if tally.Raised != 1 || tally.Lowered != 0 {
			t.Errorf("Raised, Lowered = %d, %d, want 1, 0", tally.Raised, tally.Lowered)
		}
	})
	t.Run("Floor equal to Ceiling counts in both", func(t *testing.T) {
		t.Parallel()
		var tally beliefs.Tally
		tally.Add(beliefs.Change{Before: beliefs.Belief{Centre: 100}, After: beliefs.Belief{Centre: 100}, Bounds: beliefs.Bounds{Floor: 100, Ceiling: 100}})
		if tally.AtFloor != 1 || tally.AtCeiling != 1 || tally.Outcomes != 1 || tally.Conflicts != 0 {
			t.Errorf("tally = %+v, want AtFloor 1, AtCeiling 1, Outcomes 1, Conflicts 0", tally)
		}
	})
}

// TestState_NoAllocs verifies the per-tick path of the window, the tally and
// the run allocates nothing.
//
// Requirement: NFR-BEL-001, NFR-BEL-002
func TestState_NoAllocs(t *testing.T) {
	w := beliefs.NewWindow(50, maxPrice)
	for i := range 500 { // ten times the capacity: the ring does not grow
		w.Push(int64(100 + i))
	}
	if w.Len() != 50 {
		t.Fatalf("Len after 500 pushes = %d, want 50", w.Len())
	}
	var tally beliefs.Tally
	c := beliefs.Change{Before: beliefs.Belief{Centre: 10}, After: beliefs.Belief{Centre: 12}, Bounds: beliefs.Bounds{Floor: 1, Ceiling: 99}, Open: true}
	var sink int64
	tests := []struct {
		name string
		fn   func()
	}{
		{"Push", func() { w.Push(123) }},
		{"window reads", func() {
			m, _ := w.Mean()
			l, _ := w.Low()
			h, _ := w.High()
			sink += m + l + h + int64(w.Len())
		}},
		{"Tally.Add and reads", func() {
			tally.Add(c)
			m, _ := tally.Mean()
			sink += m + tally.Low() + tally.High()
		}},
		{"NextRun", func() { sink += beliefs.NextRun(3, &tally, 10).Count }},
		{"Position", func() { sink += beliefs.Position(5, 1, 9) }},
	}
	for _, tt := range tests {
		if got := testing.AllocsPerRun(100, tt.fn); got != 0 {
			t.Errorf("%s allocs = %v, want 0", tt.name, got)
		}
	}
	_ = sink
}
