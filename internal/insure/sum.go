package insure

import "github.com/Whealinghold/economy-sim/internal/checked"

// dot returns the sum of qty[i] * value[i] over the indexes other than skip (a
// negative skip skips none), every product and every partial sum checked against
// int64 and the result against limit. ok is false for slices of different lengths,
// a negative entry, an overflow or a sum above limit (spec-insurance.md 6.3.5).
func dot(qty, value []int64, skip int, limit int64) (sum int64, ok bool) {
	if len(qty) != len(value) {
		return 0, false
	}
	for i, q := range qty {
		if i == skip {
			continue
		}
		if q < 0 || value[i] < 0 {
			return 0, false
		}
		p, ok := checked.Mul(q, value[i])
		if !ok {
			return 0, false
		}
		if sum, ok = checked.Add(sum, p); !ok {
			return 0, false
		}
	}
	if sum > limit {
		return 0, false
	}
	return sum, true
}

// SumLines returns the lines term of a sum insured, the sum of qty * value over the
// shipment's lines at the origin's unit values (spec-insurance.md 6.3.1,
// FR-INS-021). ok is false on overflow or a sum above limit, the case the round
// counts as overflow.
func SumLines(qty, value []int64, limit int64) (int64, bool) {
	return dot(qty, value, -1, limit)
}

// SumAccount returns the account term of a sum insured, the sum of held * value over
// the traveller's holdings except the index vehicle, the carrying good of the
// transport kind (spec-insurance.md 6.3.1, 3.4.2). A negative vehicle excludes none.
// ok is false on overflow or a sum above limit.
func SumAccount(held, value []int64, vehicle int, limit int64) (int64, bool) {
	return dot(held, value, vehicle, limit)
}

// SumStore returns the sum insured of a store policy, trunc(Held * shareBP / 10 000)
// with Held the sum of held * value over every holding of the insured's account and
// the product in 128 bits (spec-insurance.md 6.3.3, FR-INS-021). ok is false on an
// overflow of Held or a Held above limit, or for a share below 0 or above Basis (a
// store share is 1 to 10 000, spec-insurance.md 5.1, so the sum never exceeds Held).
func SumStore(held, value []int64, shareBP, limit int64) (sum int64, ok bool) {
	total, ok := dot(held, value, -1, limit)
	if !ok || shareBP < 0 || shareBP > Basis {
		return 0, false
	}
	return checked.MulDiv(total, shareBP, Basis)
}
