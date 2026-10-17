package welfare_test

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

const maxW = 3_650

// c is the clamp of a ring with maxWindow maxW.
const c = int64(math.MaxInt64 / maxW)

// TestFromRecords_ValidatesWhatItIsGiven verifies the restore path and
// NewRing refuse, with ErrRecords and no ring, exactly what the proof of the
// clamp does not cover, and accept the boundary values.
//
// Requirement: FR-WEL-034, FR-WEL-010
func TestFromRecords_ValidatesWhatItIsGiven(t *testing.T) {
	t.Parallel()
	mk := func(n int) []welfare.Record { return make([]welfare.Record, n) }
	tests := []struct {
		name    string
		recs    []welfare.Record
		window  int64
		maxW    int64
		refused bool
	}{
		{"income over by one", []welfare.Record{{Income: c + 1}}, 30, maxW, true},
		{"cost under by one", []welfare.Record{{Cost: -(c + 1)}}, 30, maxW, true},
		{"income exactly at C", []welfare.Record{{Income: c}}, 30, maxW, false},
		{"cost exactly at -C", []welfare.Record{{Cost: -c}}, 30, maxW, false},
		{"MinInt64 field", []welfare.Record{{OwnHH: math.MinInt64}}, 30, maxW, true},
		{"T+1 records", mk(31), 30, maxW, true},
		{"T records", mk(30), 30, maxW, false},
		{"longer than maxWindow whatever the window", mk(maxW + 1), maxW + 5, maxW, true},
		{"empty slice", nil, 30, maxW, false},
		{"window 0", nil, 0, maxW, true},
		{"negative window", nil, -3, maxW, true},
		{"window maxWindow+1", nil, maxW + 1, maxW, true},
		{"window exactly maxWindow", nil, maxW, maxW, false},
		{"maxWindow 0", nil, 1, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r, err := welfare.FromRecords(tc.recs, tc.window, tc.maxW)
			if tc.refused {
				if !errors.Is(err, welfare.ErrRecords) {
					t.Errorf("FromRecords error = %v, want ErrRecords", err)
				}
				if r.Window() != 0 || r.Len() != 0 {
					t.Errorf("a refusal returned a ring (window %d)", r.Window())
				}
			} else if err != nil {
				t.Errorf("FromRecords = %v, want accepted", err)
			}
			if tc.recs == nil { // NewRing shares the two refusals on the numbers
				_, nerr := welfare.NewRing(tc.window, tc.maxW)
				if (nerr != nil) != tc.refused || (nerr != nil && !errors.Is(nerr, welfare.ErrRecords)) {
					t.Errorf("NewRing error = %v, want refusal %v", nerr, tc.refused)
				}
			}
		})
	}
}

// TestFromRecords_SumsEqualTheSlice verifies the boundary records give sums
// recomputed from the slice, and an empty slice gives a padded DefaultRatio and
// a ring whose first Push works.
//
// Requirement: FR-WEL-034
func TestFromRecords_SumsEqualTheSlice(t *testing.T) {
	t.Parallel()
	r, err := welfare.FromRecords([]welfare.Record{
		{Income: c, Cost: -c, Working: true, WorkHH: 2, OwnHH: 1},
		{Income: -c, Cost: c, Relief: c, Working: true, WorkHH: 1},
	}, 30, maxW)
	if err != nil {
		t.Fatal(err)
	}
	// income 0, relief c, cost 0: numerator 0 - c - 0 - 0 = -c over 3 - 1 = 2.
	if num, den := r.WageParts(); num != -c || den != 2 {
		t.Errorf("WageParts = %d, %d, want %d, 2", num, den, -c)
	}
	if r.WorkingTicks() != 2 || r.Len() != 2 || !r.AtBound() {
		t.Errorf("WorkingTicks %d Len %d AtBound %v, want 2, 2, true", r.WorkingTicks(), r.Len(), r.AtBound())
	}

	e, err := welfare.FromRecords(nil, 30, maxW)
	if err != nil || e.Len() != 0 || e.Window() != 30 {
		t.Fatalf("empty ring: %v, Len %d, Window %d", err, e.Len(), e.Window())
	}
	// 30 padded ticks at upkeep 2 and 15 000: pad_income 90, pad_cost 60, 90 * 10 000 / 60.
	if got, ok := e.Ratio(2, 15_000); !ok || got != 15_000 {
		t.Errorf("empty ring Ratio = %d, %v, want 15000, true", got, ok)
	}
	e.Push(welfare.Record{Income: 5})
	if e.Len() != 1 {
		t.Errorf("Len after a first Push = %d, want 1", e.Len())
	}
}

// TestRing_ReadsEqualTheSpecFormulas verifies a hand-computed agent: a working
// tick of a job of k = 2, an own-use tick, a relief tick and a grant tick.
//
// Requirement: FR-WEL-018, FR-WEL-039, FR-WEL-040
func TestRing_ReadsEqualTheSpecFormulas(t *testing.T) {
	t.Parallel()
	r, _ := welfare.NewRing(4, maxW)
	r.Push(welfare.Record{Income: 500, Cost: 100, Working: true, WorkHH: 2})                            // job, k = 2
	r.Push(welfare.Record{Income: 120, Cost: 100, OwnReceipt: 120, Working: true, WorkHH: 1, OwnHH: 1}) // own-use
	r.Push(welfare.Record{Income: 200, Cost: 100, Relief: 200})                                         // relief tick
	r.Push(welfare.Record{Income: 300, Cost: 100, Unearned: 300})                                       // grant tick
	// income 1120, cost 400, relief 200, unearned 300, own 120, workHH 3, ownHH 1.
	if got, ok := r.Ratio(10, 10_000); !ok || got != 28_000 { // 1120 * 10 000 / 400
		t.Errorf("Ratio = %d, %v, want 28000", got, ok)
	}
	if got, ok := r.EarnedRatio(10, 10_000); !ok || got != 23_000 { // (1120 - 200) * 10 000 / 400
		t.Errorf("EarnedRatio = %d, %v, want 23000", got, ok)
	}
	// wage: (1120 - 200 - 300 - 120) / (3 - 1) = 500 / 2 = 250.
	if got, ok := r.Wage(); !ok || got != 250 {
		t.Errorf("Wage = %d, %v, want 250", got, ok)
	}
	if r.WorkingTicks() != 2 {
		t.Errorf("WorkingTicks = %d, want 2", r.WorkingTicks())
	}

	// A role with no recipes (weight 3) that traded: workHH is the weight, 3,
	// against a job of k = 2: 2. Wage over each is per household.
	a, _ := welfare.NewRing(1, maxW)
	a.Push(welfare.Record{Income: 300, Working: true, WorkHH: 3})
	b, _ := welfare.NewRing(1, maxW)
	b.Push(welfare.Record{Income: 200, Working: true, WorkHH: 2})
	wa, _ := a.Wage()
	wb, _ := b.Wage()
	if wa != 100 || wb != 100 {
		t.Errorf("wages = %d and %d, want 100 and 100 (per household, whatever the weight)", wa, wb)
	}
	// An agent that only rested has no wage: absent, not 0.
	n, _ := welfare.NewRing(2, maxW)
	n.Push(welfare.Record{Income: 50})
	if w, ok := n.Wage(); ok || w != 0 {
		t.Errorf("Wage with no working household-tick = %d, %v, want absent", w, ok)
	}
}

// TestRing_FormedValuesSaturate verifies D63 for what is formed from the
// bounded sums of one ring: the four-term numerator.
//
// Requirement: FR-WEL-010, FR-WEL-042
func TestRing_FormedValuesSaturate(t *testing.T) {
	t.Parallel()
	recs := make([]welfare.Record, maxW)
	for i := range recs {
		// income C, and relief, unearned, own-use each -C, one working household-tick.
		recs[i] = welfare.Record{Income: c, Relief: -c, Unearned: -c, OwnReceipt: -c, Working: true, WorkHH: 1}
	}
	r, err := welfare.FromRecords(recs, maxW, maxW)
	if err != nil {
		t.Fatal(err)
	}
	// numerator 4 * 3650 * C, above MaxInt64: MaxInt64; denominator 3650.
	if num, den := r.WageParts(); num != math.MaxInt64 || den != maxW {
		t.Errorf("WageParts = %d, %d, want MaxInt64, %d", num, den, maxW)
	}
	if w, ok := r.Wage(); !ok || w != math.MaxInt64/maxW {
		t.Errorf("Wage = %d, %v, want MaxInt64 / %d", w, ok, maxW)
	}
	// ratio: income 3650 * C over cost 0 and no padding: absent.
	if _, ok := r.Ratio(10, 10_000); ok {
		t.Error("Ratio with a zero denominator is present")
	}
}

// TestSetWage_ObservedAgainstPooled verifies the two set wages of D62.
//
// Requirement: FR-WEL-018, FR-WEL-019, FR-WEL-037
func TestSetWage_ObservedAgainstPooled(t *testing.T) {
	t.Parallel()
	// Per household and independent of the weights: agents (num 300, den 3)
	// and (num 200, den 2) pool to 500 / 5 = 100.
	if w, ok := welfare.PooledWage([]int64{300, 200}, []int64{3, 2}); !ok || w != 100 {
		t.Errorf("PooledWage = %d, %v, want 100", w, ok)
	}
	if w, ok := welfare.SetWage([]int64{7}, []int64{2}); !ok || w != 3 { // 3.5 truncated
		t.Errorf("SetWage = %d, %v, want 3", w, ok)
	}
	if w, ok := welfare.SetWage([]int64{-7}, []int64{2}); !ok || w != -3 { // toward zero
		t.Errorf("SetWage = %d, %v, want -3", w, ok)
	}
	// No worker: both absent.
	if _, ok := welfare.SetWage([]int64{5}, []int64{0}); ok {
		t.Error("SetWage with no working household-tick is present")
	}
	if _, ok := welfare.PooledWage(nil, nil); ok {
		t.Error("PooledWage of nothing is known")
	}
	// A negative profit: observed present and negative, pooled unknown.
	if w, ok := welfare.SetWage([]int64{-90}, []int64{3}); !ok || w != -30 {
		t.Errorf("SetWage at a loss = %d, %v, want -30, true", w, ok)
	}
	if _, ok := welfare.PooledWage([]int64{-90}, []int64{3}); ok {
		t.Error("PooledWage at a loss is known")
	}
	// Exactly 0: observed present at 0, pooled unknown (not above 0).
	if w, ok := welfare.SetWage([]int64{0}, []int64{4}); !ok || w != 0 {
		t.Errorf("SetWage of 0 = %d, %v, want 0, true", w, ok)
	}
	if _, ok := welfare.PooledWage([]int64{0}, []int64{4}); ok {
		t.Error("PooledWage of 0 is known")
	}
}

// TestSetWage_SaturatesInSliceOrder verifies the set sums saturate, in the
// caller's order.
//
// Requirement: FR-WEL-010, FR-WEL-042
func TestSetWage_SaturatesInSliceOrder(t *testing.T) {
	t.Parallel()
	const maxI = math.MaxInt64
	// Two agents at the clamp: numerators about MaxInt64 each, sum saturates.
	if w, ok := welfare.SetWage([]int64{maxI, maxI}, []int64{3, 4}); !ok || w != maxI/7 {
		t.Errorf("SetWage of two saturated numerators = %d, %v, want MaxInt64 / 7", w, ok)
	}
	if w, ok := welfare.PooledWage([]int64{maxI, maxI}, []int64{3, 4}); !ok || w != maxI/7 {
		t.Errorf("PooledWage = %d, %v, want known MaxInt64 / 7", w, ok)
	}
	// Two denominators of MaxInt64 give MaxInt64: 1.
	if w, ok := welfare.SetWage([]int64{maxI, maxI}, []int64{maxI, maxI}); !ok || w != 1 {
		t.Errorf("SetWage over saturated denominators = %d, %v, want 1", w, ok)
	}
	// Order: MaxInt64, 5, -10 -> MaxInt64 - 10; the reverse -> MaxInt64 - 5.
	if w, _ := welfare.SetWage([]int64{maxI, 5, -10}, []int64{1, 0, 0}); w != maxI-10 {
		t.Errorf("forward order = %d, want MaxInt64 - 10", w)
	}
	if w, _ := welfare.SetWage([]int64{-10, 5, maxI}, []int64{0, 0, 1}); w != maxI-5 {
		t.Errorf("reverse order = %d, want MaxInt64 - 5", w)
	}
}

// TestMeanRatio_WeightedAndRoundedTowardZero verifies FR-WEL-036.
//
// Requirement: FR-WEL-036, FR-WEL-010
func TestMeanRatio_WeightedAndRoundedTowardZero(t *testing.T) {
	t.Parallel()
	// Weights 1 and 3, ratios 10 000 and 20 001: (10 000 + 60 003) / 4 = 17 500.75 -> 17 500.
	// The third agent has no ratio and is ignored.
	m, hh, ok := welfare.MeanRatio([]int64{1, 3, 50}, []int64{10_000, 20_001, 99}, []bool{true, true, false})
	if !ok || m != 17_500 || hh != 4 {
		t.Errorf("MeanRatio = %d, %d, %v, want 17500, 4, true", m, hh, ok)
	}
	// Negative: weights 1 and 1, ratios -1 and -2: -3 / 2 = -1.5 -> -1.
	if m, _, _ := welfare.MeanRatio([]int64{1, 1}, []int64{-1, -2}, []bool{true, true}); m != -1 {
		t.Errorf("negative mean = %d, want -1 (toward zero)", m)
	}
	if _, hh, ok := welfare.MeanRatio([]int64{2}, []int64{5}, []bool{false}); ok || hh != 0 {
		t.Errorf("no ratio: households %d, ok %v, want 0, absent", hh, ok)
	}
	// Saturated: two agents of weight 2 with ratio MaxInt64 -> sum saturates, 4 households.
	m, hh, ok = welfare.MeanRatio([]int64{2, 2}, []int64{math.MaxInt64, math.MaxInt64}, []bool{true, true})
	if !ok || hh != 4 || m != math.MaxInt64/4 {
		t.Errorf("saturated mean = %d, %d, %v, want MaxInt64 / 4, 4, true", m, hh, ok)
	}
}

// TestPremium_AbsenceAndExactQuotient verifies the premium.
//
// Requirement: FR-WEL-019, FR-WEL-037
func TestPremium_AbsenceAndExactQuotient(t *testing.T) {
	t.Parallel()
	if p, ok := welfare.Premium(150, true, 100, true); !ok || p != 15_000 {
		t.Errorf("Premium = %d, %v, want 15000", p, ok)
	}
	if p, ok := welfare.Premium(-150, true, 100, true); !ok || p != -15_000 {
		t.Errorf("negative role wage Premium = %d, %v, want -15000", p, ok)
	}
	for _, tc := range []struct {
		name        string
		role        int64
		rolePresent bool
		unsk        int64
		unskPresent bool
	}{
		{"no unskilled agent", 150, true, 100, false},
		{"role without working household-ticks", 150, false, 100, true},
		{"unskilled wage 0", 150, true, 0, true},
		{"unskilled wage negative", 150, true, -5, true},
	} {
		if p, ok := welfare.Premium(tc.role, tc.rolePresent, tc.unsk, tc.unskPresent); ok || p != 0 {
			t.Errorf("%s: Premium = %d, %v, want absent", tc.name, p, ok)
		}
	}
	half := int64(math.MaxInt64 / 2)
	if p, ok := welfare.Premium(half, true, 1, true); !ok || p != math.MaxInt64 {
		t.Errorf("Premium over wage 1 = %d, %v, want MaxInt64", p, ok)
	}
	want := new(big.Int).Div(new(big.Int).Mul(big.NewInt(half), big.NewInt(10_000)), big.NewInt(20_000)).Int64()
	if p, ok := welfare.Premium(half, true, 20_000, true); !ok || p != want {
		t.Errorf("Premium over wage 20000 = %d, %v, want %d (not saturated)", p, ok, want)
	}
}

// TestHeldByFloor_Predicate verifies the held predicate of D69.
//
// Requirement: FR-WEL-020, FR-WEL-027, FR-WEL-033
func TestHeldByFloor_Predicate(t *testing.T) {
	t.Parallel()
	// T 30, MinWorkShare 4 000: the threshold is 12 working ticks. lo 15 000, band 500: hi 15 750.
	tests := []struct {
		name    string
		ratio   int64
		present bool
		working int64
		minShr  int64
		want    bool
	}{
		{"floor off", 16_000, true, 0, 0, false},
		{"ratio below T_lo: share already 10 000", 14_999, true, 0, 4_000, false},
		{"ratio absent", 0, false, 0, 4_000, false},
		{"ratio at T_hi: share 0, floor holds", 15_750, true, 11, 4_000, true},
		{"ratio between: share below 10 000", 15_300, true, 11, 4_000, true},
		{"ratio far above", 40_000, true, 11, 4_000, true},
		{"one below the threshold", 20_000, true, 11, 4_000, true},
		{"exactly at the threshold", 20_000, true, 12, 4_000, false},
	}
	for _, tc := range tests {
		if got := welfare.HeldByFloor(tc.ratio, tc.present, tc.working, 30, tc.minShr, 15_000, 500); got != tc.want {
			t.Errorf("%s: HeldByFloor = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// FuzzSetFunctions asserts the set functions never panic and equal a
// math/big reference: a sum clamps after every term, a quotient is exact and
// clamped once.
//
// Requirement: NFR-WEL-002, FR-WEL-010
func FuzzSetFunctions(f *testing.F) {
	f.Add(int64(math.MaxInt64), int64(5), int64(-10), int64(3), int64(1), int64(2), int64(7))
	f.Add(int64(math.MinInt64), int64(math.MaxInt64), int64(math.MinInt64), int64(0), int64(-1), int64(1), int64(20_000))
	f.Add(int64(math.MaxInt64/2), int64(1), int64(1), int64(1), int64(1), int64(1), int64(1))
	f.Fuzz(func(t *testing.T, n1, n2, n3, d1, d2, w, u int64) {
		clamp := func(x *big.Int) int64 {
			if x.Cmp(big.NewInt(math.MaxInt64)) > 0 {
				return math.MaxInt64
			}
			if x.Cmp(big.NewInt(math.MinInt64)) < 0 {
				return math.MinInt64
			}
			return x.Int64()
		}
		fold := func(vs ...int64) int64 {
			var s int64
			for _, v := range vs {
				s = clamp(new(big.Int).Add(big.NewInt(s), big.NewInt(v)))
			}
			return s
		}
		quo := func(a, b, den int64) int64 {
			p := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
			return clamp(p.Quo(p, big.NewInt(den)))
		}
		nums, dens := []int64{n1, n2, n3}, []int64{d1, d2, d1}
		num, den := fold(nums...), fold(dens...)
		w1, ok1 := welfare.SetWage(nums, dens)
		if ok1 != (den > 0) || (ok1 && w1 != quo(num, 1, den)) {
			t.Errorf("SetWage(%v, %v) = %d, %v; sums %d, %d", nums, dens, w1, ok1, num, den)
		}
		w2, ok2 := welfare.PooledWage(nums, dens)
		if ok2 != (ok1 && w1 > 0) || (ok2 && w2 != w1) {
			t.Errorf("PooledWage = %d, %v, SetWage %d, %v", w2, ok2, w1, ok1)
		}
		ws, rs, ps := []int64{w, d2}, []int64{u, n1}, []bool{true, n2 >= 0}
		var sum, hh int64
		for i := range ws {
			if ps[i] {
				sum, hh = fold(sum, quo(ws[i], rs[i], 1)), fold(hh, ws[i])
			}
		}
		m, gh, ok := welfare.MeanRatio(ws, rs, ps)
		if gh != hh || ok != (hh > 0) || (ok && m != quo(sum, 1, hh)) {
			t.Errorf("MeanRatio(%v, %v, %v) = %d, %d, %v; want sum %d households %d", ws, rs, ps, m, gh, ok, sum, hh)
		}
		p, okp := welfare.Premium(w, true, u, true)
		if okp != (u > 0) || (okp && p != quo(w, 10_000, u)) {
			t.Errorf("Premium(%d, %d) = %d, %v", w, u, p, okp)
		}
	})
}

// Benchmarks report the ring push and read; they gate nothing.
//
// Requirement: NFR-WEL-003
func BenchmarkRing_PushAndRatio(b *testing.B) {
	r, _ := welfare.NewRing(30, maxW)
	rc := welfare.Record{Income: 100, Cost: 50, Working: true, WorkHH: 1}
	b.ReportAllocs()
	for b.Loop() {
		r.Push(rc)
		r.Ratio(10, 15_000)
	}
}
