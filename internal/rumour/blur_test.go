package rumour_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/rumour"
)

// TestBlur_BoundsAndRounding verifies price * (10 000 + e) / 10 000 rounded
// half to even and clamped to [1, maxPrice]: the worked 370 at 5% of
// spec-information.md 12.1, the clamp to 1 for a price of 1 or 2, the clamp to
// maxPrice, and the unchanged price at e = 0.
//
// Requirement: FR-INF-015
func TestBlur_BoundsAndRounding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		price, e, maxPrice int64
		want               int64
	}{
		// 370 * 9 500 / 10 000 = 351.5, a tie, to the even 352.
		{"low tie goes to the even 352", 370, -500, 1_000_000_000, 352},
		// 370 * 10 500 / 10 000 = 388.5, a tie, to the even 388.
		{"high tie goes to the even 388", 370, 500, 1_000_000_000, 388},
		{"e = 0 keeps the price", 370, 0, 1_000_000_000, 370},
		// 368 * 9 770 / 10 000 = 359.536 rounds to 360 (the worked e = -230).
		{"the draw -230 on 368", 368, -230, 1_000_000_000, 360},
		// 1 * 5 000 / 10 000 = 0.5, a tie, to the even 0, clamped up to 1.
		{"a price of 1 clamps to 1", 1, -5_000, 1_000_000_000, 1},
		// 2 * 5 000 / 10 000 = 1.
		{"a price of 2 at -50% is 1", 2, -5_000, 1_000_000_000, 1},
		// 100 * 15 000 / 10 000 = 150, clamped to the bound 120.
		{"clamped to maxPrice", 100, 5_000, 120, 120},
		{"price at the bound stays", 120, 0, 120, 120},
		{"a price of 0 is clamped up to 1", 0, 500, 100, 1},
		{"a bound of 1 gives 1", 50, 0, 1, 1},
		// 50 * 0 / 10 000 = 0 at the factor 0 (e = -10 000), clamped up to 1.
		{"a factor of 0 gives 1", 50, -10_000, 100, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := rumour.Blur(tt.price, tt.e, tt.maxPrice)
			if !ok || got != tt.want {
				t.Errorf("Blur(%d, %d, %d) = %d, %v, want %d, true", tt.price, tt.e, tt.maxPrice, got, ok, tt.want)
			}
		})
	}
}

// TestBlur_WithinBound verifies, over a grid of prices and errors, that the
// result stays within [1, maxPrice] and that |Price - p| is at most
// ceil(p * E / 10 000) for |e| <= E.
//
// Requirement: FR-INF-015
func TestBlur_WithinBound(t *testing.T) {
	t.Parallel()
	const maxPrice = 1_000_000_000
	for _, p := range []int64{1, 2, 3, 7, 99, 370, 4_999, 123_457, maxPrice} {
		for _, e := range []int64{-5_000, -4_999, -500, -1, 0, 1, 499, 500, 5_000} {
			got, ok := rumour.Blur(p, e, maxPrice)
			bound := (p*abs(e) + 9_999) / 10_000
			if !ok || got < 1 || got > maxPrice || abs(got-p) > bound {
				t.Errorf("Blur(%d, %d, %d) = %d, %v, want within [1, %d] and %d of %d", p, e, maxPrice, got, ok, maxPrice, bound, p)
			}
		}
	}
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// TestBlur_BadInput verifies that a negative price, a bound below 1, a factor
// below zero and a product that overflows are reported as ok false.
//
// Requirement: FR-INF-015
func TestBlur_BadInput(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name               string
		price, e, maxPrice int64
	}{
		{"negative price", -1, 0, 100},
		{"bound below 1", 10, 0, 0},
		{"factor below zero", 10, -10_001, 100},
		{"error that overflows the factor", 10, math.MaxInt64, 100},
		{"product that overflows", math.MaxInt64, 5_000, math.MaxInt64},
	} {
		if got, ok := rumour.Blur(tt.price, tt.e, tt.maxPrice); ok || got != 0 {
			t.Errorf("%s: Blur(%d, %d, %d) = %d, %v, want 0, false", tt.name, tt.price, tt.e, tt.maxPrice, got, ok)
		}
	}
}

// TestBlurMean_ZeroStaysZero verifies that a mean of 0 ("no mean") is not
// blurred, and that a mean above 0 is blurred as a price with the same e.
//
// Requirement: FR-INF-015
func TestBlurMean_ZeroStaysZero(t *testing.T) {
	t.Parallel()
	for _, e := range []int64{-5_000, -1, 0, 1, 5_000} {
		if got, ok := rumour.BlurMean(0, e, 100); !ok || got != 0 {
			t.Errorf("BlurMean(0, %d, 100) = %d, %v, want 0, true", e, got, ok)
		}
	}
	// 366 * 9 500 / 10 000 = 347.7 rounds to 348; the same e as the price.
	if got, ok := rumour.BlurMean(366, -500, 1_000_000_000); !ok || got != 348 {
		t.Errorf("BlurMean(366, -500, 1e9) = %d, %v, want 348, true", got, ok)
	}
	if got, ok := rumour.BlurMean(-1, 0, 100); ok {
		t.Errorf("BlurMean(-1, 0, 100) = %d, true, want ok false", got)
	}
}

// FuzzBlur verifies, for any price, error and bound, that Blur never panics,
// that a result lies within [1, maxPrice], that for a price in [1, maxPrice]
// and |e| <= 5 000 it is within ceil(p * |e| / 10 000) of the price, and that
// it agrees with a math/big oracle whenever the inputs are valid.
//
// Requirement: NFR-INF-004
func FuzzBlur(f *testing.F) {
	f.Add(int64(370), int64(-500), int64(1_000_000_000))
	f.Add(int64(1), int64(-5_000), int64(1_000_000_000))
	f.Add(int64(math.MaxInt64), int64(5_000), int64(math.MaxInt64))
	f.Add(int64(-1), int64(0), int64(100))
	f.Add(int64(10), int64(math.MinInt64), int64(0))
	f.Fuzz(func(t *testing.T, price, e, maxPrice int64) {
		got, ok := rumour.Blur(price, e, maxPrice)
		if !ok {
			return
		}
		if got < 1 || got > maxPrice {
			t.Fatalf("Blur(%d, %d, %d) = %d, want within [1, %d]", price, e, maxPrice, got, maxPrice)
		}
		if price >= 1 && price <= maxPrice && e >= -5_000 && e <= 5_000 {
			bound := new(big.Int).Mul(big.NewInt(price), big.NewInt(abs(e)))
			bound.Add(bound, big.NewInt(9_999)).Div(bound, big.NewInt(10_000))
			if d := big.NewInt(abs(got - price)); d.Cmp(bound) > 0 {
				t.Fatalf("Blur(%d, %d, %d) = %d, off by %v, want at most %v", price, e, maxPrice, got, d, bound)
			}
		}
		// half-even oracle: q = round(price * (10 000 + e) / 10 000).
		num := new(big.Int).Mul(big.NewInt(price), big.NewInt(10_000+e))
		q, r := new(big.Int).QuoRem(num, big.NewInt(10_000), new(big.Int))
		if r.Cmp(big.NewInt(5_000)) > 0 || r.Cmp(big.NewInt(5_000)) == 0 && q.Bit(0) == 1 {
			q.Add(q, big.NewInt(1))
		}
		want := min(max(q.Int64(), 1), maxPrice)
		if got != want {
			t.Fatalf("Blur(%d, %d, %d) = %d, want %d", price, e, maxPrice, got, want)
		}
	})
}
