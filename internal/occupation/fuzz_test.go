package occupation_test

import (
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

func bi(v int64) *big.Int { return big.NewInt(v) }

// exactQuotient returns trunc(a * b / c) for c > 0 as a big integer, and
// whether it fits int64.
func exactQuotient(a, b, c int64) (*big.Int, bool) {
	q := new(big.Int).Mul(bi(a), bi(b))
	q.Quo(q, bi(c))
	return q, q.IsInt64()
}

func clamp(v, lo, hi int64) int64 { return min(max(v, lo), hi) }

// FuzzArithmetic verifies, for any int64 inputs, that nothing panics; that
// Expected and Viability equal the exact quotient with math/big wherever it
// fits int64 and saturate by the sign of the wage where it does not; that Beats
// equals the exact inequality eD * 10 000 >= eO * (10 000 + hysteresis); that
// CapitalOK equals the exact inequality where upkeep * ticksPerYear fits and
// denies where it does not; that Renewal conserves carry + removed * rate; that
// Remaining equals the exact floor and stays within [0, x]; and that Exits never
// drops more than the allowance or a weight and accounts for every household it
// wanted to shed.
//
// Requirement: FR-SPN-014, FR-SPN-017, FR-SPN-019, FR-SPN-024, FR-SPN-051, FR-SPN-058
// #lizard forgives: one fuzz body, one property per block; the 13 inputs are the committed corpus format.
func FuzzArithmetic(f *testing.F) {
	// Seeds: SPN 15.1 (signals and the band), 15.4 (the cohort), 15.7 (the carry),
	// a case in which the first quotient of E alone exceeds int64, and the extremes.
	f.Add(int64(1200), int64(250), int64(1000), int64(365), int64(9041), int64(8219), int64(1000), int64(1_825_000), int64(100), int64(40), int64(7), int64(5), int64(1000))
	f.Add(int64(1e18), int64(100_000), int64(1e14), int64(100_000), int64(0), int64(0), int64(0), int64(1), int64(1), int64(1), int64(1), int64(0), int64(0))
	f.Add(int64(-1e18), int64(100_000), int64(9e13), int64(100_000), int64(math.MinInt64), int64(math.MaxInt64), int64(10_000), int64(math.MaxInt64), int64(math.MaxInt64), int64(1_000_000), int64(1_000_000), int64(1_000_000), int64(math.MaxInt64))
	f.Add(int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0))
	f.Fuzz(func(t *testing.T, wage, expected, upkeep, tpy, eD, eO, hyst, price, cr, weight, unfed, allowance, step int64) {
		e, has := occupation.Expected(wage, expected, upkeep, tpy)
		v, hasV := occupation.Viability(wage, upkeep)
		if has != (upkeep > 0 && tpy > 0) || hasV != (upkeep > 0) {
			t.Fatalf("Expected(%d, %d, %d, %d) has = %v, Viability has = %v", wage, expected, upkeep, tpy, has, hasV)
		}
		if upkeep > 0 {
			if q, fits := exactQuotient(wage, 10_000, upkeep); fits && v != q.Int64() {
				t.Fatalf("Viability(%d, %d) = %d, want %v", wage, upkeep, v, q)
			} else if !fits && v != saturateRef(q.Sign()) {
				t.Fatalf("Viability(%d, %d) = %d, want the extreme of sign %d", wage, upkeep, v, q.Sign())
			}
		}
		if has && expected >= 0 && expected <= 100_000 {
			den := new(big.Int).Mul(bi(upkeep), bi(tpy))
			q := new(big.Int).Mul(bi(wage), bi(expected*10_000))
			q.Quo(q, den)
			want := saturateRef(q.Sign())
			if q.IsInt64() {
				want = q.Int64()
			}
			if e != want {
				t.Fatalf("Expected(%d, %d, %d, %d) = %d, want %d (exact %v)", wage, expected, upkeep, tpy, e, want, q)
			}
		}

		if hyst >= 0 {
			left := new(big.Int).Mul(bi(eD), bi(10_000))
			right := new(big.Int).Mul(bi(eO), new(big.Int).Add(bi(hyst), bi(10_000)))
			if got, want := occupation.Beats(eD, eO, hyst), left.Cmp(right) >= 0; got != want {
				t.Fatalf("Beats(%d, %d, %d) = %v, want %v", eD, eO, hyst, got, want)
			}
		}

		got := occupation.CapitalOK(eD, eO, price, cr, upkeep, tpy)
		if price >= 0 && cr >= 0 && upkeep > 0 && tpy > 0 {
			ut := new(big.Int).Mul(bi(upkeep), bi(tpy))
			left := new(big.Int).Mul(new(big.Int).Sub(bi(eD), bi(eO)), ut)
			right := new(big.Int).Mul(bi(price), bi(cr))
			want := price == 0 || (ut.IsInt64() && left.Cmp(right) >= 0)
			if got != want {
				t.Fatalf("CapitalOK(%d, %d, %d, %d, %d, %d) = %v, want %v", eD, eO, price, cr, upkeep, tpy, got, want)
			}
		}

		if n, c, ok := occupation.Renewal(clamp(price, 0, 9999), clamp(weight, 0, 1<<40), clamp(cr, 0, 1<<20)); ok {
			total := new(big.Int).Mul(bi(clamp(weight, 0, 1<<40)), bi(clamp(cr, 0, 1<<20)))
			total.Add(total, bi(clamp(price, 0, 9999)))
			if back := new(big.Int).Add(new(big.Int).Mul(bi(n), bi(10_000)), bi(c)); back.Cmp(total) != 0 || c < 0 || c >= 10_000 {
				t.Fatalf("Renewal gave %d entries, carry %d for total %v", n, c, total)
			}
		}

		w := clamp(weight, 1, 1_000_000)
		un := clamp(unfed, 0, w)
		x := max(price, 0)
		if r, ok := occupation.Remaining(x, w, un); !ok || r < 0 || r > x {
			t.Fatalf("Remaining(%d, %d, %d) = %d, %v, want a value in [0, x]", x, w, un, r, ok)
		} else if q := new(big.Int).Quo(new(big.Int).Mul(bi(x), bi(un)), bi(w)); q.Int64() != r {
			t.Fatalf("Remaining(%d, %d, %d) = %d, want %v", x, w, un, r, q)
		}

		ls := []occupation.Leaver{
			{Ref: 0, Privation: eD, Weight: w, Unfed: un, Money: price, Upkeep: upkeep},
			{Ref: 1, Privation: eD, Weight: clamp(w/2+1, 1, w+1), Unfed: clamp(un, 0, w/2+1), Money: price, Upkeep: upkeep},
			{Ref: 2, Privation: eO, Weight: 1, Unfed: 1},
		}
		wanted := int64(0)
		for _, l := range ls {
			if l.Unfed > 0 {
				wanted += min(l.Unfed, l.Weight)
			}
		}
		out, cut := occupation.Exits(nil, slices.Clone(ls), allowance, step)
		shed := int64(0)
		for _, x := range out {
			l := ls[x.Ref]
			if x.Households < 1 || x.Households > l.Weight || x.Households > l.Unfed || x.Remove != (x.Households == l.Weight) {
				t.Fatalf("Exits gave %+v for %+v", x, l)
			}
			shed += x.Households
		}
		if shed > max(allowance, 0) || cut < 0 || shed+cut > wanted {
			t.Fatalf("Exits(%+v, %d, %d) shed %d, cut %d, at most %d wanted", ls, allowance, step, shed, cut, wanted)
		}
	})
}

func saturateRef(sign int) int64 {
	switch sign {
	case 1:
		return math.MaxInt64
	case -1:
		return math.MinInt64
	}
	return 0
}
