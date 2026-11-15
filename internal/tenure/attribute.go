package tenure

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Supply apportions the let units the starts took, v, among the lessors by the
// weights of their unused entitlements (spec-land.md 4.4.1) and writes each
// lessor's supply to out[part.Ref]. Part.Share is the unused entitlement U_h.
// The rule is Apportion's, and because v is at most the sum of the weights the
// supply of a lessor never exceeds its unused entitlement.
//
// ok is false, and every slot is 0, when v is above the sum of the weights, or
// for anything Apportion refuses.
func Supply(v int64, lessors []Part, out []int64) (ok bool) {
	var sum int64
	for _, l := range lessors {
		var added bool
		if sum, added = checked.Add(sum, l.Share); !added {
			zeroOut(lessors, out)
			return false
		}
	}
	if v > sum {
		zeroOut(lessors, out)
		return false
	}
	return Apportion(v, lessors, out)
}

// Taker is one tenant, or one lessor, of the attribution: an agent ID and the
// units it took (tenant) or supplies (lessor).
type Taker struct {
	ID    int64
	Units int64
}

// Triple is one pairing of the attribution: Units let by the lessor at index
// Lessor to the tenant at index Tenant of the slices as they were passed in
// (before the sort by ID, so the caller's indices stay valid). Self is true when
// tenant and lessor are one agent, whose transfer is skipped and whose units
// still count as let (spec-land.md 4.4.2, 5.3.3).
type Triple struct {
	Tenant int
	Lessor int
	Units  int64
	Self   bool
}

// Attribute walks the tenants and the lessors in ascending ID with two pointers
// and assigns the units each tenant took to the lessors' remaining supplies in
// order (spec-land.md 4.4.2). It appends the triples to dst and returns it.
//
// The tenants' units and the lessors' supplies must have the same sum. The two
// slices are not reordered: the walk runs over index slices of the scratch. ok is
// false, with dst unchanged, for a negative number, a repeated ID within a list,
// a sum beyond int64 or sums that differ.
func (s *Scratch) Attribute(dst []Triple, tenants, lessors []Taker) ([]Triple, bool) {
	var tok, lok bool
	s.ti, tok = ascending(s.ti, tenants)
	s.li, lok = ascending(s.li, lessors)
	if !tok || !lok {
		return dst, false
	}
	var want, have int64
	for _, i := range s.ti {
		var ok bool
		if want, ok = checked.Add(want, tenants[i].Units); !ok {
			return dst, false
		}
	}
	for _, i := range s.li {
		var ok bool
		if have, ok = checked.Add(have, lessors[i].Units); !ok {
			return dst, false
		}
	}
	if want != have {
		return dst, false
	}
	return s.walk(dst, tenants, lessors), true
}

// ascending returns the indices of the list in ascending ID, in idx's storage,
// and whether the IDs are distinct and the units not negative.
func ascending(idx []int, list []Taker) ([]int, bool) {
	idx = slices.Grow(idx[:0], len(list))[:len(list)]
	for i, t := range list {
		if t.Units < 0 {
			return idx, false
		}
		idx[i] = i
	}
	slices.SortFunc(idx, func(a, b int) int { return cmp.Compare(list[a].ID, list[b].ID) })
	for i := 1; i < len(idx); i++ {
		if list[idx[i-1]].ID == list[idx[i]].ID {
			return idx, false
		}
	}
	return idx, true
}

// walk is the two-pointer pass of Attribute over the validated lists.
func (s *Scratch) walk(dst []Triple, tenants, lessors []Taker) []Triple {
	left := slices.Grow(s.left[:0], len(s.li))[:len(s.li)]
	for k, i := range s.li {
		left[k] = lessors[i].Units
	}
	s.left = left
	l := 0
	for _, i := range s.ti {
		need := tenants[i].Units
		for need > 0 {
			for left[l] == 0 {
				l++
			}
			n := min(need, left[l])
			j := s.li[l]
			dst = append(dst, Triple{Tenant: i, Lessor: j, Units: n, Self: tenants[i].ID == lessors[j].ID})
			need -= n
			left[l] -= n
		}
	}
	return dst
}
