package facility

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Fac is one site facility of the owner's class and region, offered to Assign.
type Fac struct {
	Ref        int   // the caller's index, returned in a Placement and the tie-break
	Table      int64 // the condition in the effect table: it orders the assignment
	Condition  int64 // the running condition of the record: wear is taken from it
	Cap        int64 // cap(f) of the table: executions a tick that it serves
	Served     int64 // executions that earlier starts of the tick put on it
	WearPerUse int64 // condition lost for each execution served
}

// Placement is a number of units given to the facility Ref.
type Placement struct {
	Ref int
	N   int64
}

// Assign puts k executions on the facilities most worn first (spec-infrastructure.md
// 5.1.3): in ascending Table condition, ties to the lower Ref. Each facility
// takes at most Cap less Served, adds what it took to Served and loses
// taken * WearPerUse of Condition, to a floor of 0. It sorts fac in place and
// updates Served and Condition in place; the caller reads them back by Ref.
// The order depends on the table and not on the running condition, so two
// starts of one tick give what one start of their sum gives. dst is reused for
// the placements, one per facility that took at least one execution, in
// assignment order. rest is what no facility could take; it is 0 when k does
// not exceed the room. A k below 1 places nothing.
func Assign(dst []Placement, k int64, fac []Fac) (placed []Placement, rest int64) {
	dst = dst[:0]
	k = max(k, 0)
	slices.SortFunc(fac, func(a, b Fac) int {
		return cmp.Or(cmp.Compare(a.Table, b.Table), cmp.Compare(a.Ref, b.Ref))
	})
	for i := range fac {
		if k == 0 {
			break
		}
		f := &fac[i]
		take := min(k, max(f.Cap-f.Served, 0))
		if take == 0 {
			continue
		}
		k -= take
		f.Served += take
		f.Condition = wear(f.Condition, take, f.WearPerUse)
		dst = append(dst, Placement{Ref: f.Ref, N: take})
	}
	return dst, k
}

// wear returns condition less n * perUse, to a floor of 0. A negative perUse
// is 0: wear never restores. A product that overflows int64 is more than any
// condition, so the result is 0.
func wear(condition, n, perUse int64) int64 {
	perUse = max(perUse, 0)
	w, ok := checked.Mul(n, perUse)
	if !ok {
		return 0
	}
	return max(condition-w, 0)
}
