package household_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/household"
)

// FuzzTiersAndCap verifies, for any int64 inputs, that nothing panics, that the
// three tiers add to the gap, that the cap is 0 at no excess, is never above
// the excess and never decreases when the excess grows by one, and that Afford
// never spends more than the money.
//
// Requirement: NFR-AGT-004, FR-AGT-040, FR-AGT-042
func FuzzTiersAndCap(f *testing.F) {
	// Seeds from AGT 16.1 and 16.2: keep, stock, reserve, jobwant; e, upkeep
	// cost, half ticks, share; avail, limit, want.
	f.Add(int64(12), int64(4), int64(3), int64(0), int64(4505), int64(1000), int64(10), int64(5000), int64(1447), int64(310), int64(4))
	f.Add(int64(2), int64(0), int64(0), int64(2), int64(0), int64(0), int64(0), int64(0), int64(50), int64(105), int64(3))
	f.Add(int64(-1), int64(1<<62), int64(1<<62), int64(-5), int64(1<<62), int64(1<<62), int64(1<<62), int64(20000), int64(-1), int64(0), int64(1<<62))
	f.Add(int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(50), int64(105), int64(0))
	f.Fuzz(func(t *testing.T, keep, stock, reserve, job, e, uc, half, share, avail, limit, want int64) {
		gap, surplus, t1, t2, t3 := household.Tiers(keep, stock, reserve, job)
		if t1 < 0 || t2 < 0 || t3 < 0 || t1+t2+t3 != gap || (gap > 0 && surplus > 0) {
			t.Fatalf("Tiers(%d, %d, %d, %d) = %d, %d, %d, %d, %d", keep, stock, reserve, job, gap, surplus, t1, t2, t3)
		}
		p := household.Params{SpendHalfTicks: half, SpendMaxShare: share}
		got, ok := household.SpendCap(e, uc, p)
		if e <= 0 && (got != 0 || !ok) {
			t.Fatalf("SpendCap(%d, %d, %+v) = %d, %v, want 0, true", e, uc, p, got, ok)
		}
		if ok && (got < 0 || got > max(e, 0)) {
			t.Fatalf("SpendCap(%d, %d, %+v) = %d outside [0, e]", e, uc, p, got)
		}
		if next, ok2 := household.SpendCap(e+1, uc, p); ok && ok2 && e > 0 && e < 1<<62 && next < got {
			t.Fatalf("SpendCap decreases: cap(%d) = %d, cap(%d) = %d (%d, %+v)", e, got, e+1, next, uc, p)
		}
		if tg, ok := household.ComputeTargets(household.Need{Upkeep: keep, Job: job, OutTgt: stock, Want: reserve}, household.Params{CoverTicks: half}); ok && tg.Keep < tg.Reserve {
			t.Fatalf("ComputeTargets keep %d below reserve %d", tg.Keep, tg.Reserve)
		}
		if _, ok := household.ReserveMoney(uc, half); ok {
			if _, ok := checked.Mul(uc, half); !ok {
				t.Fatalf("ReserveMoney(%d, %d) ok but the product overflows", uc, half)
			}
		}
		l, q := household.Afford(avail, limit, want)
		if q < 0 || (q > 0 && (l < 1 || q > avail/l || q > want)) {
			t.Fatalf("Afford(%d, %d, %d) = %d, %d spends more than the money or the want", avail, limit, want, l, q)
		}
		household.BuyShare(avail, p)
		household.Excess(avail, limit)
		household.Fed([]int64{avail, limit}, []int64{want, e}, keep)
	})
}

// FuzzWorkRules verifies, for any int64 inputs, that nothing panics, that
// Better is antisymmetric and agrees with the exact comparison on small
// values, is transitive on three samples, that Halted is monotone in the stock,
// and that Request never returns (0, true).
//
// Requirement: NFR-AGT-004, FR-AGT-035, FR-AGT-036, FR-AGT-037
func FuzzWorkRules(f *testing.F) {
	f.Add(int64(500), int64(2), int64(200), int64(1), int64(300), int64(3), int64(5), int64(10), int64(5000))
	f.Add(int64(-1), int64(9), int64(-1), int64(2), int64(0), int64(1), int64(0), int64(0), int64(30000))
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MinInt64), int64(1), int64(1), int64(1), int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64))
	f.Fuzz(func(t *testing.T, na, da, nb, db, nc, dc, stock, target, ratio int64) {
		ab, ba := household.Better(na, da, nb, db), household.Better(nb, db, na, da)
		if ab && ba {
			t.Fatalf("Better(%d,%d,%d,%d) and its reverse are both true", na, da, nb, db)
		}
		if small := func(v int64) bool { return v > -1<<30 && v < 1<<30 }; small(na) && small(nb) && small(da) && small(db) && da > 0 && db > 0 {
			if want := na*db > nb*da; ab != want {
				t.Fatalf("Better(%d,%d,%d,%d) = %v, want %v", na, da, nb, db, ab, want)
			}
		}
		if bc := household.Better(nb, db, nc, dc); ab && bc && !household.Better(na, da, nc, dc) {
			t.Fatalf("Better is not transitive on (%d,%d) (%d,%d) (%d,%d)", na, da, nb, db, nc, dc)
		}
		if lo, hi := household.Halted(stock, target, ratio), household.Halted(stock+1, target, ratio); stock >= 0 && stock < math.MaxInt64 && lo && !hi {
			t.Fatalf("Halted(%d,%d,%d) is true but false at stock+1", stock, target, ratio)
		}
		if target < 1 && household.Halted(stock, target, ratio) {
			t.Fatalf("Halted(%d,%d,%d) with no target", stock, target, ratio)
		}
		if n, ok := household.Request(na, nb); ok != (na > 0 && nb > 0) || (ok && (n < 1 || n > na || n > nb)) || (!ok && n != 0) {
			t.Fatalf("Request(%d,%d) = %d, %v", na, nb, n, ok)
		}
		if tg, ok := household.OutputTarget(da, db, dc, stock, ratio); ok && tg < 0 {
			t.Fatalf("OutputTarget = %d, true", tg)
		}
		if c := household.KeepFoodCap(stock, target, ratio); c < 0 || (ratio > 0 && c > stock/ratio+1) {
			t.Fatalf("KeepFoodCap(%d,%d,%d) = %d", stock, target, ratio, c)
		}
		var n household.Net
		n.Output(na, da, db)
		n.Input(nb, nc)
		n.Cost(dc)
		n.Value()
		if household.Eligible(na, ratio > 0) && na <= 0 && ratio <= 0 {
			t.Fatalf("Eligible(%d, false) = true", na)
		}
	})
}
