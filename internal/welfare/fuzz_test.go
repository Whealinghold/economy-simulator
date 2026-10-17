package welfare_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// ref is the exact reference of the fuzz test: math/big values clamped to
// int64 once per named value of the spec's formulas, so a wrapped value and a
// saturate-then-divide both differ from it.
var (
	lo64 = big.NewInt(math.MinInt64)
	hi64 = big.NewInt(math.MaxInt64)
)

func bi(v int64) *big.Int { return big.NewInt(v) }

func fits(x *big.Int) bool { return x.Cmp(lo64) >= 0 && x.Cmp(hi64) <= 0 }

func clamp(x *big.Int) int64 {
	switch {
	case x.Cmp(hi64) > 0:
		return math.MaxInt64
	case x.Cmp(lo64) < 0:
		return math.MinInt64
	}
	return x.Int64()
}

func add(a, b int64) int64 { return clamp(new(big.Int).Add(bi(a), bi(b))) }
func sub(a, b int64) int64 { return clamp(new(big.Int).Sub(bi(a), bi(b))) }
func mul(a, b int64) int64 { return clamp(new(big.Int).Mul(bi(a), bi(b))) }

// quoBig is p / c rounded toward zero, or toward positive infinity if ceil.
func quoBig(p *big.Int, c int64, ceil bool) *big.Int {
	q, r := new(big.Int).QuoRem(p, bi(c), new(big.Int))
	// A remainder means the exact quotient is not whole; the truncated quotient
	// is below it when the exact quotient is positive.
	if ceil && r.Sign() != 0 && p.Sign()*sgn(c) > 0 {
		q.Add(q, bi(1))
	}
	return q
}

func sgn(v int64) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	}
	return 0
}

// quo is a * b / c truncated toward zero, clamped once; c is not 0.
func quo(a, b, c int64) int64 {
	return clamp(quoBig(new(big.Int).Mul(bi(a), bi(b)), c, false))
}

// quoCeil is the same rounded toward positive infinity.
func quoCeil(a, b, c int64) int64 {
	return clamp(quoBig(new(big.Int).Mul(bi(a), bi(b)), c, true))
}

// calibOK returns the value of p1a * p1b * x / (p2a * p2b) and whether a
// calibration formula should report ok: the two products fit int64 (the
// documented limit), the divisor is not 0 and the quotient fits.
func calibOK(p1a, p1b, x, p2a, p2b int64, ceil bool) (int64, bool) {
	n := new(big.Int).Mul(bi(p1a), bi(p1b))
	d := new(big.Int).Mul(bi(p2a), bi(p2b))
	if !fits(n) || !fits(d) || d.Sign() == 0 {
		return 0, false
	}
	q := quoBig(n.Mul(n, bi(x)), d.Int64(), ceil)
	if !fits(q) {
		return 0, false
	}
	return q.Int64(), true
}

// FuzzArithmetic verifies, for arbitrary int64 operands, that no function of
// the package panics, that every saturating function equals the exact math/big
// value clamped once per named value, that the calibration formulas report ok
// false exactly where the exact value or one of the two products they form
// leaves int64, that the work share lies in [0, 10 000], and that the ratio of
// a window of equal records equals the ratio of one record.
//
// Requirement: NFR-WEL-002, FR-WEL-010, FR-WEL-042
//
// Guard: the files in testdata/fuzz/FuzzArithmetic pin the cases a wrapping
// or a saturate-then-divide implementation gets wrong: a quotient in range
// from a product above int64, and the minimum operand.
func FuzzArithmetic(f *testing.F) {
	f.Add(int64(450), int64(300), int64(3), int64(10), int64(15_000), int64(7))
	f.Add(int64(math.MaxInt64/2), int64(20_000), int64(0), int64(1), int64(10_000), int64(5))
	f.Add(int64(math.MinInt64), int64(math.MaxInt64), int64(-1), int64(0), int64(math.MinInt64), int64(-3))
	f.Add(int64(-5), int64(math.MinInt64), int64(40), int64(5_000), int64(500), int64(4))
	f.Fuzz(func(t *testing.T, a, b, c, d, e, g int64) {
		// Ratio and EarnedRatio: a income, b cost, c n, d upkeep, e default; g relief.
		padCost := mul(c, d)
		padIncome := quo(padCost, e, 10_000)
		den := add(b, padCost)
		gotR, okR := welfare.Ratio(a, b, c, d, e)
		if den == 0 {
			if okR || gotR != 0 {
				t.Errorf("Ratio(%d, %d, %d, %d, %d) = %d, %v, want absent", a, b, c, d, e, gotR, okR)
			}
		} else if want := quo(add(a, padIncome), 10_000, den); !okR || gotR != want {
			t.Errorf("Ratio(%d, %d, %d, %d, %d) = %d, %v, want %d, true", a, b, c, d, e, gotR, okR, want)
		}
		gotE, okE := welfare.EarnedRatio(a, g, b, c, d, e)
		if den == 0 {
			if okE {
				t.Errorf("EarnedRatio(%d, %d, %d, %d, %d, %d) present with no denominator", a, g, b, c, d, e)
			}
		} else if want := quo(add(sub(a, g), padIncome), 10_000, den); !okE || gotE != want {
			t.Errorf("EarnedRatio(%d, %d, %d, %d, %d, %d) = %d, %v, want %d, true", a, g, b, c, d, e, gotE, okE, want)
		}

		// Calibration: a ratio, b basket, c year, d work days / wage.
		wantW, wantOK := calibOK(a, b, c, 10_000, d, false)
		if got, ok := welfare.Wage(a, b, c, d); ok != wantOK || got != wantW {
			t.Errorf("Wage(%d, %d, %d, %d) = %d, %v, want %d, %v", a, b, c, d, got, ok, wantW, wantOK)
		}
		wantF, wantOK := calibOK(a, d, 10_000, b, c, false)
		if got, ok := welfare.RatioForWage(a, d, b, c); ok != wantOK || got != wantF {
			t.Errorf("RatioForWage(%d, %d, %d, %d) = %d, %v, want %d, %v", a, d, b, c, got, ok, wantF, wantOK)
		}
		wantD, wantOK := calibOK(a, b, c, 10_000, d, true)
		if d <= 0 {
			wantD, wantOK = 0, false
		} else if wantOK {
			wantD = min(wantD, c)
		}
		if got, ok := welfare.DaysWorked(a, b, c, d); ok != wantOK || got != wantD {
			t.Errorf("DaysWorked(%d, %d, %d, %d) = %d, %v, want %d, %v", a, b, c, d, got, ok, wantD, wantOK)
		}

		// Work share, executions, floor, target range.
		hi := quo(a, add(10_000, c), 10_000)
		var wantS int64
		switch {
		case b < a:
			wantS = 10_000
		case b >= hi:
			wantS = 0
		default:
			wantS = quo(sub(hi, b), 10_000, sub(hi, a))
		}
		gotS := welfare.WorkShare(b, a, c)
		if gotS != wantS || gotS < 0 || gotS > 10_000 {
			t.Errorf("WorkShare(%d, %d, %d) = %d, want %d in [0, 10000]", b, a, c, gotS, wantS)
		}
		if got, want := welfare.Executions(a, b), quoCeil(a, b, 10_000); got != want {
			t.Errorf("Executions(%d, %d) = %d, want %d", a, b, got, want)
		}
		wantB := e != 0 && c < quo(e, d, 10_000)
		if got := welfare.FloorBinds(c, d, e); got != wantB {
			t.Errorf("FloorBinds(%d, %d, %d) = %v, want %v", c, d, e, got, wantB)
		}
		gotLo, gotHi := welfare.TargetRange(a, b)
		if wantLo, wantHi := quo(a, sub(10_000, b), 10_000), quo(a, add(10_000, b), 10_000); gotLo != wantLo || gotHi != wantHi {
			t.Errorf("TargetRange(%d, %d) = %d, %d, want %d, %d", a, b, gotLo, gotHi, wantLo, wantHi)
		}

		// Wear cost (a wear, b price, c durability, d executions) and the tick sums.
		if c != 0 {
			if got, want := welfare.WearCost(a, b, c, d), mul(d, quoCeil(a, b, c)); got != want {
				t.Errorf("WearCost(%d, %d, %d, %d) = %d, want %d", a, b, c, d, got, want)
			}
		}
		wantC := add(add(0, mul(mul(a, c), g)), mul(mul(b, d), g))
		if got := welfare.TickCost([]int64{a, b}, []int64{c, d}, g); got != wantC {
			t.Errorf("TickCost([%d %d], [%d %d], %d) = %d, want %d", a, b, c, d, g, got, wantC)
		}
		wantI := sub(add(add(a, b), c), add(d, e))
		if got := welfare.TickIncome([]int64{a, b, c}, []int64{d, e}); got != wantI {
			t.Errorf("TickIncome([%d %d %d], [%d %d]) = %d, want %d", a, b, c, d, e, got, wantI)
		}

		// A window of k equal records reads the ratio of one record (no padding).
		const bound = 1_000_000_000
		ri, rc, k := a%bound, b%bound, c%3650
		if k < 0 {
			k = -k
		}
		k++
		one, okOne := welfare.Ratio(ri, rc, 0, 0, 10_000)
		many, okMany := welfare.Ratio(ri*k, rc*k, 0, 0, 10_000)
		if one != many || okOne != okMany {
			t.Errorf("Ratio of %d records of (%d, %d) = %d, %v, want the single record's %d, %v", k, ri, rc, many, okMany, one, okOne)
		}
	})
}
