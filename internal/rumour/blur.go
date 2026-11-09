package rumour

import "github.com/Whealinghold/economy-sim/internal/checked"

// basisPoints is the denominator of a rate in basis points.
const basisPoints = 10_000

// Blur returns price * (10 000 + e) / 10 000 rounded half to even and clamped
// to [1, maxPrice], the error of a rumour (spec-information.md 5.5.1). e is
// the drawn error in basis points, from -E to E with E at most 5 000, so the
// factor is positive. ok is false when price is negative, maxPrice is below 1,
// the factor 10 000 + e is negative, or the product overflows int64.
func Blur(price, e, maxPrice int64) (blurred int64, ok bool) {
	factor, ok := checked.Add(basisPoints, e)
	if !ok || price < 0 || maxPrice < 1 || factor < 0 {
		return 0, false
	}
	q, ok := checked.MulDivHalfEven(price, factor, basisPoints)
	if !ok {
		return 0, false
	}
	return min(max(q, 1), maxPrice), true
}

// BlurMean is Blur for the window mean of an observation: a mean of 0 means
// "no mean" (spec-information.md 4.2.1) and stays 0, a mean above 0 is
// blurred by the same e as the price, so their ratio is kept to the rounding.
func BlurMean(mean, e, maxPrice int64) (blurred int64, ok bool) {
	if mean == 0 {
		return 0, true
	}
	return Blur(mean, e, maxPrice)
}
