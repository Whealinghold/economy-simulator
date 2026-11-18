package facility

import (
	"cmp"
	"slices"
)

// Need is one facility that a repair can restore.
type Need struct {
	Ref       int   // the caller's index, returned in a Placement and the tie-break
	Condition int64 // the condition now: it orders the distribution
	Missing   int64 // units missing from its durability; 0 or less is full
}

// Distribute gives units of condition to the facilities most worn first
// (spec-infrastructure.md 4.3.2): in ascending Condition, ties to the lower
// Ref, each at most its Missing. It sorts need in place and returns one
// placement per facility that received at least one unit, in that order, reusing
// dst, and voided, the units left when every facility is full. A units below 1
// places nothing and voids nothing.
func Distribute(dst []Placement, units int64, need []Need) (placed []Placement, voided int64) {
	dst = dst[:0]
	units = max(units, 0)
	slices.SortFunc(need, func(a, b Need) int {
		return cmp.Or(cmp.Compare(a.Condition, b.Condition), cmp.Compare(a.Ref, b.Ref))
	})
	for i := range need {
		if units == 0 {
			break
		}
		give := min(units, max(need[i].Missing, 0))
		if give == 0 {
			continue
		}
		units -= give
		dst = append(dst, Placement{Ref: need[i].Ref, N: give})
	}
	return dst, units
}
