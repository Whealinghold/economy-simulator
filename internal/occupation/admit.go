package occupation

import (
	"cmp"
	"slices"
)

// The kinds of a Candidate, in the order Admit serves them (spec-spawner.md
// 6.3.0): a rung-1 switcher first, then a reviewer, then a cross-role
// replacement batch.
const (
	CandRung1 = iota
	CandReview
	CandReplace
)

// Dest is one destination of a candidate: Role indexes Limits.RoleCap and the
// used slice, and Qualifies says whether the candidate may enter it (open, a
// signal, viable or beating its own role as the kind requires). Dests come in
// the candidate's destination order, best first.
type Dest struct {
	Role      int
	Qualifies bool
}

// Candidate is an agent that wants to change role, or a batch of replacement
// households (spec-spawner.md 6.3.0, 7.2). The key is (Kind, Privation
// descending, Gap descending, Seq, Seq2, Ref), a total order (architecture.md
// 7.6). A kind 0 or 1 candidate is an agent of Households households, admitted
// whole or not at all (Ref its index, Seq its agent ID, Seq2 0). A kind 2
// candidate is the Households weight-1 replacement households of one leaver
// role (Ref and Seq its role index, Privation and Gap 0); Seq2 is 0 in
// production and a tie-break for the expanded households of a test oracle.
// A weight above 1 is Households: the caps count households.
type Candidate struct {
	Kind                                  int
	Ref                                   int
	Privation, Gap, Seq, Seq2, Households int64
	Dests                                 []Dest
}

// DestCount is the households admitted into the role Dest.
type DestCount struct {
	Dest       int
	Households int64
}

// Admission is what one candidate was granted: for kinds 0 and 1 one entry with
// the candidate's households, for a kind 2 batch the count per destination.
type Admission struct {
	Ref      int
	Admitted []DestCount
}

// Limits are the caps of spec-spawner.md 6.3, as Allowance computes them:
// RegionCap bounds the households changed in the region and RoleCap[r] the
// households entering role r (indexed like Dest.Role).
type Limits struct {
	RegionCap int64
	RoleCap   []int64
}

// Counts are the candidate households Admit refused: RegionCap by the region
// cap, RoleCap because every destination that qualified for them was full.
// A household is counted once; one with no qualifying destination and the
// region cap not reached is counted in neither (it stays, spec-spawner.md
// 6.4).
type Counts struct {
	RegionCap, RoleCap int64
}

// Admit sorts cs in place by the key of Candidate and admits them in that order
// while the region has changed fewer than lim.RegionCap households and the
// destination has taken fewer than its cap; the candidate that reaches a cap is
// admitted, so a cap is exceeded by at most one candidate's households
// (spec-spawner.md 6.3, FR-SPN-021). A candidate whose best destination is full
// takes the next qualifying one. A kind 2 batch is admitted in closed form, per
// destination max(0, min(left, region room, role room)) households, which equals
// admitting its weight-1 households one at a time; cost is O(candidates x
// destinations), never O(households).
//
// regionUsed and used (indexed by Dest.Role) are the counts so far and are
// advanced, so a second call continues the first. The admissions are appended to
// adm[:0] and their per-destination counts to dc[:0]; with enough capacity Admit
// allocates nothing. A role index outside used or lim.RoleCap is a caller error.
// Counts are bounded by the household limits of validation, so the sums cannot
// overflow.
func Admit(adm []Admission, dc []DestCount, cs []Candidate, lim Limits, regionUsed *int64, used []int64) ([]Admission, []DestCount, Counts) {
	slices.SortFunc(cs, compareCandidates)
	adm, dc = adm[:0], dc[:0]
	var c Counts
	for i := range cs {
		cand := &cs[i]
		start := len(dc)
		if cand.Kind == CandReplace {
			dc = admitBatch(dc, cand, lim, regionUsed, used, &c)
		} else {
			dc = admitWhole(dc, cand, lim, regionUsed, used, &c)
		}
		if len(dc) > start {
			adm = append(adm, Admission{Ref: cand.Ref, Admitted: dc[start:len(dc):len(dc)]})
		}
	}
	return adm, dc, c
}

func compareCandidates(a, b Candidate) int {
	return cmp.Or(
		cmp.Compare(a.Kind, b.Kind),
		cmp.Compare(b.Privation, a.Privation),
		cmp.Compare(b.Gap, a.Gap),
		cmp.Compare(a.Seq, b.Seq),
		cmp.Compare(a.Seq2, b.Seq2),
		cmp.Compare(a.Ref, b.Ref),
	)
}

func admitWhole(dc []DestCount, cand *Candidate, lim Limits, regionUsed *int64, used []int64, c *Counts) []DestCount {
	if *regionUsed >= lim.RegionCap {
		c.RegionCap += cand.Households
		return dc
	}
	qualified := false
	for _, d := range cand.Dests {
		if !d.Qualifies {
			continue
		}
		qualified = true
		if used[d.Role] < lim.RoleCap[d.Role] {
			used[d.Role] += cand.Households
			*regionUsed += cand.Households
			return append(dc, DestCount{Dest: d.Role, Households: cand.Households})
		}
	}
	if qualified {
		c.RoleCap += cand.Households
	}
	return dc
}

func admitBatch(dc []DestCount, cand *Candidate, lim Limits, regionUsed *int64, used []int64, c *Counts) []DestCount {
	left := cand.Households
	qualified := false
	for _, d := range cand.Dests {
		if !d.Qualifies {
			continue
		}
		qualified = true
		take := max(0, min(left, lim.RegionCap-*regionUsed, lim.RoleCap[d.Role]-used[d.Role]))
		if take == 0 {
			continue
		}
		used[d.Role] += take
		*regionUsed += take
		left -= take
		dc = append(dc, DestCount{Dest: d.Role, Households: take})
	}
	switch {
	case left <= 0:
	case *regionUsed >= lim.RegionCap:
		c.RegionCap += left
	case qualified:
		c.RoleCap += left
	}
	return dc
}
