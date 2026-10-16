package checked_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// refAcc is the oracle: the exact sum with math/big, divided half to even, and
// whether the sum fits 128 bits and the quotient fits int64.
func refAcc(pairs [][2]int64, c int64) (q int64, sumOK, divOK bool) {
	sum := new(big.Int)
	for _, p := range pairs {
		sum.Add(sum, new(big.Int).Mul(big.NewInt(p[0]), big.NewInt(p[1])))
	}
	if sum.BitLen() > 128 {
		return 0, false, false
	}
	if c < 1 {
		return 0, true, false
	}
	bc := big.NewInt(c)
	bq, br := new(big.Int).QuoRem(sum, bc, new(big.Int))
	twice := new(big.Int).Lsh(br, 1)
	if cmp := twice.Cmp(bc); cmp > 0 || (cmp == 0 && bq.Bit(0) == 1) {
		bq.Add(bq, big.NewInt(1))
	}
	if !bq.IsInt64() {
		return 0, true, false
	}
	return bq.Int64(), true, true
}

func runAcc(pairs [][2]int64, c int64) (q int64, sumOK, divOK bool) {
	var a checked.Acc
	for _, p := range pairs {
		if !a.AddMul(p[0], p[1]) {
			return 0, false, false
		}
	}
	q, divOK = a.DivHalfEven(c)
	return q, true, divOK
}

// TestAcc_AgainstBig checks the accumulator against math/big at the edges of
// 128 bits, in rounding ties, in the zero divisor and in quotient overflow.
//
// Requirement: FR-CLR-019
func TestAcc_AgainstBig(t *testing.T) {
	t.Parallel()
	m := int64(math.MaxInt64)
	tests := []struct {
		name  string
		pairs [][2]int64
		c     int64
	}{
		{"empty", nil, 1},
		{"tie to even down", [][2]int64{{5, 1}}, 2},                 // 2.5 -> 2
		{"tie to even up", [][2]int64{{7, 1}}, 2},                   // 3.5 -> 4
		{"worked example", [][2]int64{{2, 9}, {1, 10}, {2, 10}}, 5}, // 48/5 = 9.6 -> 10
		{"max product", [][2]int64{{m, m}}, m},
		{"two max products", [][2]int64{{m, m}, {m, m}}, m},
		{"quotient overflows", [][2]int64{{m, m}}, 1},
		{"quotient 2^63-1 fits", [][2]int64{{m, 4}}, 4},                                                  // (2^63-1)*4/4
		{"quotient 2^63-1 rounds down", [][2]int64{{m, 4}, {1, 1}}, 4},                                   // remainder 1 of 4
		{"quotient 2^63 by round up", [][2]int64{{m, 4}, {3, 1}}, 4},                                     // remainder 3 of 4 -> 2^63
		{"quotient 2^63 by tie to even", [][2]int64{{m, 2}, {1, 1}}, 2},                                  // 2^63-1 odd, tie -> 2^63
		{"quotient exactly 2^63", [][2]int64{{1 << 62, 6}}, 3},                                           // 2^63*3/3
		{"quotient 2^63*7/7 does not fit", [][2]int64{{1 << 62, 14}}, 7},                                 // exactly 2^63
		{"quotient 2^63 with remainder rounds down, does not fit", [][2]int64{{1 << 62, 14}, {3, 1}}, 7}, // 2^63 rem 3 of 7
		{"quotient 2^63-1 with remainder rounds up, does not fit", [][2]int64{{m, 7}, {4, 1}}, 7},        // 2^63-1 rem 4 of 7 -> 2^63
		{"quotient 2^63-1 with remainder rounds down, fits", [][2]int64{{m, 7}, {3, 1}}, 7},              // rem 3 of 7
		{"zero divisor", [][2]int64{{3, 3}}, 0},
		{"negative divisor", [][2]int64{{3, 3}}, -1},
		{"sum overflows 128 bits", [][2]int64{{m, m}, {m, m}, {m, m}, {m, m}, {m, m}}, m},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wq, wsum, wdiv := refAcc(tt.pairs, tt.c)
			gq, gsum, gdiv := runAcc(tt.pairs, tt.c)
			if gsum != wsum || gdiv != wdiv || gq != wq {
				t.Errorf("Acc(%v) / %d = (%d, %v, %v), want (%d, %v, %v)", tt.pairs, tt.c, gq, gsum, gdiv, wq, wsum, wdiv)
			}
		})
	}
	t.Run("negative operand refused and sum unchanged", func(t *testing.T) {
		t.Parallel()
		var a checked.Acc
		a.AddMul(2, 3)
		if a.AddMul(-1, 5) || a.AddMul(5, -1) {
			t.Error("AddMul(-1, 5) or AddMul(5, -1) = true, want false for a negative operand")
		}
		if q, ok := a.DivHalfEven(1); !ok || q != 6 {
			t.Errorf("DivHalfEven(1) after AddMul(2, 3) = (%d, %v), want (6, true)", q, ok)
		}
	})
	t.Run("random against big", func(t *testing.T) {
		t.Parallel()
		r := rand.New(rand.NewPCG(1, 2))
		for i := 0; i < 2000; i++ {
			pairs := make([][2]int64, r.IntN(4))
			for j := range pairs {
				pairs[j] = [2]int64{r.Int64N(m), r.Int64N(m)}
			}
			c := r.Int64N(m) + 1
			wq, wsum, wdiv := refAcc(pairs, c)
			gq, gsum, gdiv := runAcc(pairs, c)
			if gsum != wsum || gdiv != wdiv || gq != wq {
				t.Fatalf("iteration %d: Acc(%v) / %d = (%d, %v, %v), want (%d, %v, %v)", i, pairs, c, gq, gsum, gdiv, wq, wsum, wdiv)
			}
		}
	})
}

// TestAcc_HiEqualsDivisor asserts that a sum whose high word equals the
// divisor is refused without a bits.Div64 panic: 2^62 * 4 = 2^64, so hi == 1,
// and dividing by 1 would need a 65-bit quotient.
//
// Requirement: FR-CLR-019
func TestAcc_HiEqualsDivisor(t *testing.T) {
	t.Parallel()
	var acc checked.Acc
	if !acc.AddMul(1<<62, 4) {
		t.Fatal("AddMul(2^62, 4) = false, want true")
	}
	if q, ok := acc.DivHalfEven(1); ok {
		t.Errorf("DivHalfEven(1) = (%d, true), want ok false", q)
	}
}

// TestAcc_RoundUpOverflow asserts that a quotient of 2^64-1 that must round up
// is refused rather than wrapped to 0.
//
// Requirement: FR-CLR-019
//
// Guard: the round-up increment is the one place the quotient can wrap; without
// the check a mean price of 0 would be published for a huge trade.
func TestAcc_RoundUpOverflow(t *testing.T) {
	t.Parallel()
	var acc checked.Acc
	// 2*(2^63-1) twice is 2^65-4; adding 3 gives 2^65-1 = 2*(2^64-1) + 1, a
	// tie at divisor 2 with an odd quotient, so it rounds up to 2^64.
	for _, p := range [][2]int64{{math.MaxInt64, 2}, {math.MaxInt64, 2}, {3, 1}} {
		if !acc.AddMul(p[0], p[1]) {
			t.Fatalf("AddMul(%d, %d) = false, want true", p[0], p[1])
		}
	}
	if q, ok := acc.DivHalfEven(2); ok {
		t.Errorf("DivHalfEven(2) = (%d, true), want ok false", q)
	}
}

// FuzzAcc fuzzes the accumulator against math/big over up to three products
// and a divisor.
//
// Requirement: FR-CLR-019
//
// Fix: the corpus file testdata/fuzz/FuzzAcc/seed-worked-example pins the worked
// example of CLR 11 (2*9 + 1*10 + 2*10 over 5); it is a hand-written seed, not
// a crash, and the same input is given with f.Add.
func FuzzAcc(f *testing.F) {
	f.Add(int64(2), int64(9), int64(1), int64(10), int64(2), int64(10), int64(5))
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64), int64(0), int64(0), int64(math.MaxInt64))
	f.Add(int64(5), int64(1), int64(0), int64(0), int64(0), int64(0), int64(2))
	f.Add(int64(1<<62), int64(4), int64(0), int64(0), int64(0), int64(0), int64(1))         // hi == divisor
	f.Add(int64(math.MaxInt64), int64(4), int64(3), int64(1), int64(0), int64(0), int64(4)) // rounds up to 2^63
	f.Add(int64(math.MaxInt64), int64(4), int64(0), int64(0), int64(0), int64(0), int64(4)) // 2^63-1 fits
	f.Add(int64(1<<62), int64(6), int64(0), int64(0), int64(0), int64(0), int64(3))         // exactly 2^63
	f.Add(int64(1<<62), int64(14), int64(0), int64(0), int64(0), int64(0), int64(7))        // 2^63*7/7
	f.Add(int64(1<<62), int64(14), int64(3), int64(1), int64(0), int64(0), int64(7))        // 2^63 rem 3
	f.Add(int64(math.MaxInt64), int64(7), int64(4), int64(1), int64(0), int64(0), int64(7)) // 2^63-1 rounds up
	f.Fuzz(func(t *testing.T, a, b, c, d, e, g, div int64) {
		pairs := [][2]int64{{a, b}, {c, d}, {e, g}}
		// The accumulator refuses negatives; keep the oracle to its domain.
		for _, p := range pairs {
			if p[0] < 0 || p[1] < 0 {
				var acc checked.Acc
				if acc.AddMul(p[0], p[1]) {
					t.Fatalf("AddMul(%d, %d) accepted a negative operand", p[0], p[1])
				}
				return
			}
		}
		wq, wsum, wdiv := refAcc(pairs, div)
		gq, gsum, gdiv := runAcc(pairs, div)
		if gsum != wsum || gdiv != wdiv || gq != wq {
			t.Errorf("Acc(%v) / %d = (%d, %v, %v), want (%d, %v, %v)", pairs, div, gq, gsum, gdiv, wq, wsum, wdiv)
		}
	})
}
