package fiscal_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

// dutyWant is the exact result of DutyDue by big integers: the quotient cut to
// maxMoney and the remainder of (base * rate * scale + carry) / 10^8.
func dutyWant(base *big.Int, rateBP, scale, carry, maxMoney int64) (due, rem *big.Int) {
	num := new(big.Int).Mul(base, new(big.Int).Mul(big.NewInt(rateBP), big.NewInt(scale)))
	num.Add(num, big.NewInt(carry))
	due, rem = new(big.Int).DivMod(num, big.NewInt(100_000_000), new(big.Int))
	if due.Cmp(big.NewInt(maxMoney)) > 0 {
		due.SetInt64(maxMoney)
	}
	return due, rem
}

// dutyInDomain reports whether DutyDue accepts the arguments other than the base.
func dutyInDomain(rateBP, scale, carry, maxMoney int64) bool {
	fits := new(big.Int).Mul(big.NewInt(rateBP), big.NewInt(scale)).IsInt64()
	return rateBP >= 0 && scale >= 0 && carry >= 0 && carry < 100_000_000 && maxMoney >= 0 && fits
}

// FuzzDutyDue verifies that DutyDue never panics and, on any input it accepts,
// equals the big-integer quotient and remainder of (base * rate * scale +
// carry) / 10^8, with the due cut to maxMoney and a carry in [0, 10^8).
//
// Requirement: FR-FIS-032, NFR-FIS-003
func FuzzDutyDue(f *testing.F) {
	f.Add(int64(530), int64(820), int64(300), int64(10_000), int64(0), int64(1)<<60)
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(1), int64(1), int64(99_999_999), int64(math.MaxInt64))
	f.Fuzz(func(t *testing.T, v0, v1, rateBP, scale, carry, maxMoney int64) {
		var b fiscal.Base
		sum := new(big.Int)
		for _, v := range []int64{v0, v1} {
			if b.Add(v) {
				sum.Add(sum, big.NewInt(v))
			}
		}
		due, newCarry, ok := fiscal.DutyDue(b, rateBP, scale, carry, maxMoney)
		if valid := dutyInDomain(rateBP, scale, carry, maxMoney); ok != valid {
			t.Fatalf("DutyDue(%v, %d, %d, %d, %d) ok = %v, want %v", sum, rateBP, scale, carry, maxMoney, ok, valid)
		}
		if !ok {
			return
		}
		q, r := dutyWant(sum, rateBP, scale, carry, maxMoney)
		if big.NewInt(due).Cmp(q) != 0 || big.NewInt(newCarry).Cmp(r) != 0 {
			t.Fatalf("DutyDue(%v, %d, %d, %d, %d) = %d, %d; want %v, %v", sum, rateBP, scale, carry, maxMoney, due, newCarry, q, r)
		}
	})
}

// FuzzLimitWish verifies that Limit stays in [1, maxPrice] and that Wish
// equals the big-integer quotient and remainder of (qty * scale + carry) /
// 10^4 whenever the quotient fits.
//
// Requirement: FR-FIS-050, FR-FIS-051, NFR-FIS-003
func FuzzLimitWish(f *testing.F) {
	f.Add(int64(50), int64(8_000), int64(100), int64(7), int64(12_500), int64(9_999))
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64), int64(0))
	f.Fuzz(func(t *testing.T, ref, bp, maxPrice, qty, scale, carry int64) {
		if p, ok := fiscal.Limit(ref, bp, maxPrice); ok && (p < 1 || p > maxPrice) {
			t.Fatalf("Limit(%d, %d, %d) = %d, outside [1, %d]", ref, bp, maxPrice, p, maxPrice)
		}
		wish, next, ok := fiscal.Wish(qty, scale, carry)
		if !ok {
			return
		}
		num := new(big.Int).Mul(big.NewInt(qty), big.NewInt(scale))
		num.Add(num, big.NewInt(carry))
		q, r := new(big.Int).DivMod(num, big.NewInt(fiscal.Basis), new(big.Int))
		if q.IsInt64() && (big.NewInt(wish).Cmp(q) != 0 || big.NewInt(next).Cmp(r) != 0) {
			t.Fatalf("Wish(%d, %d, %d) = %d, %d; want %v, %v", qty, scale, carry, wish, next, q, r)
		}
	})
}

// purchaseHolds reports whether a capped purchase loses no unit and is bought
// by the cash at the limit.
func purchaseHolds(c fiscal.Capped, wish, cash, limit int64) bool {
	if c.Qty < 0 || c.CutCash < 0 || c.CutHold < 0 || c.Qty+c.CutCash+c.CutHold != max(0, wish) {
		return false
	}
	return c.Qty == 0 || (limit >= 1 && c.Qty <= max(0, cash)/limit)
}

// FuzzCaps verifies that the caps lose no unit: the placed quantity and the
// cut units add up to the wish, a placed purchase is bought by the cash at the
// limit and stays within the hold, and a placed release is covered by the
// unreserved stock, for any (also negative) wish, cash, hold, held and reserve.
//
// Requirement: FR-FIS-051, NFR-FIS-003
func FuzzCaps(f *testing.F) {
	f.Add(int64(8), int64(300), int64(60), int64(10), int64(4), int64(2))
	f.Add(int64(math.MaxInt64), int64(-5), int64(0), int64(-1), int64(math.MinInt64), int64(math.MinInt64))
	f.Fuzz(func(t *testing.T, wish, cash, limit, hold, held, reserve int64) {
		c := fiscal.CapPurchase(wish, cash, limit, hold, held)
		if !purchaseHolds(c, wish, cash, limit) || (hold > 0 && c.Qty > max(0, hold-max(0, held))) {
			t.Fatalf("CapPurchase(%d, %d, %d, %d, %d) = %+v: units lost, cash not enough or hold passed", wish, cash, limit, hold, held, c)
		}
		r := fiscal.CapRelease(wish, held, reserve)
		if r.Qty < 0 || r.CutStock < 0 || r.Qty+r.CutStock != max(0, wish) || r.Qty > max(0, max(0, held)-max(0, reserve)) {
			t.Fatalf("CapRelease(%d, %d, %d) = %+v: units lost or stock passed", wish, held, reserve, r)
		}
	})
}
