package production

import "math"

// NoLimit is the cap of a limit that does not bind.
const NoLimit = math.MaxInt64

// NoReason is the reason Count returns when k is 0 but no limit of the list is
// 0 (a weight of 0 or less).
const NoReason = -1

// Limit is one cap on the executions of a start and the reason the caller
// attributes to it when the cap is 0. The package does not interpret Reason.
type Limit struct {
	Cap    int64
	Reason int
}

// Count returns k, the executions of a start (PRO 4.2): the least of weight,
// the strategy's maximum (0 or less means none; a maximum above the weight
// changes nothing) and every cap of limits, taken in the caller's master
// order. When k is 0 the reason is that of the first limit whose cap is 0,
// else NoReason. A cap below 0 counts as 0 and k is never negative. A partial
// k is not an error.
func Count(weight, maximum int64, limits []Limit) (k int64, reason int) {
	k = max(weight, 0)
	if maximum > 0 {
		k = min(k, maximum)
	}
	for _, l := range limits {
		k = min(k, max(l.Cap, 0))
	}
	if k > 0 {
		return k, NoReason
	}
	for _, l := range limits {
		if l.Cap <= 0 {
			return 0, l.Reason
		}
	}
	return 0, NoReason
}

// Cap turns an available stock and a per-execution need into a cap by integer
// division: stock / need rounded down. A need of 0 or less is no limit,
// whatever the stock; a stock of 0 or less gives 0.
func Cap(stock, need int64) int64 {
	if need <= 0 {
		return NoLimit
	}
	if stock <= 0 {
		return 0
	}
	return stock / need
}

// CapInputAndTool is the defensive cap of PRO 4.2.3 (D57) for a commodity
// found in both Inputs and Requires: stock / (q + 1), the 1 being the one tool
// item an execution wears. The caller attributes it to the tool reason.
func CapInputAndTool(stock, q int64) int64 {
	if q < 0 {
		q = 0
	}
	if q == math.MaxInt64 {
		return 0 // q + 1 would exceed every stock
	}
	return Cap(stock, q+1)
}

// YieldLimit is the first limit of the list (D59). The cap is 0 when the
// effective factor is 0 or less, else NoLimit. noSite is true when the recipe
// is sited and no profile row exists for the region and recipe, whatever the
// factor; the root maps true to no_site and false to zero_yield.
func YieldLimit(sited, hasProfile bool, effective int64) (cap int64, noSite bool) {
	noSite = sited && !hasProfile
	if noSite || effective <= 0 {
		return 0, noSite
	}
	return NoLimit, false
}
