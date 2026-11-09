package population

import "github.com/Whealinghold/economy-sim/internal/checked"

// Means returns the household-weighted mean annual birth rate b, death rate d
// and natural increase n = the mean of birth - death, in basis points a year,
// over the entries whose present flag is set, each truncated toward zero
// (spec-population.md 10.4). weights, birth, death and present are parallel
// slices. With no weight present all three are 0. ok is false on overflow or
// when the slices differ in length.
func Means(weights, birth, death []int64, present []bool) (b, d, n int64, ok bool) {
	if !sameLen(len(weights), birth, death, present) {
		return 0, 0, 0, false
	}
	var total, sb, sd int64
	for i, w := range weights {
		if !present[i] || w <= 0 {
			continue
		}
		if total, ok = checked.Add(total, w); !ok {
			return 0, 0, 0, false
		}
		if sb, ok = addProduct(sb, w, birth[i]); !ok {
			return 0, 0, 0, false
		}
		if sd, ok = addProduct(sd, w, death[i]); !ok {
			return 0, 0, 0, false
		}
	}
	if total == 0 {
		return 0, 0, 0, true
	}
	// The rates are the caller's, so a negative one could overflow the difference.
	diff, ok := checked.Sub(sb, sd)
	if !ok {
		return 0, 0, 0, false
	}
	return sb / total, sd / total, diff / total, true
}

// addProduct returns acc + w*rate, ok false on overflow.
func addProduct(acc, w, rate int64) (int64, bool) {
	p, ok := checked.Mul(w, rate)
	if !ok {
		return 0, false
	}
	return checked.Add(acc, p)
}

// sameLen reports whether the three parallel slices have n entries.
func sameLen(n int, birth, death []int64, present []bool) bool {
	return len(birth) == n && len(death) == n && len(present) == n
}
