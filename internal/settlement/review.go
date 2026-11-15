package settlement

import "github.com/Whealinghold/economy-sim/internal/checked"

// Review returns the class a region of class c, with class floor floor and
// household count h, is given by one review (spec-settlements.md 4.3.1): up
// through every class whose enter threshold the count meets and, only if nothing
// moved up, down through every class whose leave threshold the count is below,
// never below the floor. e and l are the effective thresholds of Thresholds. The
// two directions are exclusive because l[c] < e[c] < e[c+1] for a band of at
// least 1 basis point. A c or floor outside the classes is clamped, and c below
// floor is raised to it (a state a restore refuses, so the clamp only keeps the
// function total).
func Review(c, floor int, h int64, e, l []int64) int {
	n := len(e)
	if n == 0 || len(l) != n {
		return 0
	}
	f := clampClass(floor, n)
	cur := max(clampClass(c, n), f)
	up := cur
	for up+1 < n && h >= e[up+1] {
		up++
	}
	if up != cur {
		return up
	}
	for cur > f && h < l[cur] {
		cur--
	}
	return cur
}

// Target returns the class a review at count h would give ignoring the dwell and
// whether the region is at its floor (spec-settlements.md 4.3.3, 10.5): the class
// is the floor, above the first class, and the count alone is below its leave
// threshold, so without the floor the class would be lower.
func Target(c, floor int, h int64, e, l []int64) (target int, atFloor bool) {
	target = Review(c, floor, h, e, l)
	n := len(e)
	if n == 0 || len(l) != n {
		return target, false
	}
	f := clampClass(floor, n)
	return target, f > 0 && clampClass(c, n) == f && h < l[f]
}

// Kind is the outcome of the dwell test of one review.
type Kind uint8

const (
	// Same means the review found no change due.
	Same Kind = iota
	// Changed means the class changes now.
	Changed
	// Held means a change is due and the dwell holds it.
	Held
)

// Outcome is the result of Decide: the kind and, for Held, the ticks left of the
// dwell.
type Outcome struct {
	// Kind is Same, Changed or Held.
	Kind Kind
	// TicksLeft is dwell - (t - since), at least 1, for Held; 0 otherwise.
	TicksLeft int64
}

// Decide applies the dwell to a review (spec-settlements.md 4.4.1): target is the
// class Review gave, c the current class, since the tick of the last change, t
// the tick of the review. A change is applied when t - since >= dwell and held
// otherwise; a held change leaves no state. A t below since (a state a restore
// refuses) is a held change with the full dwell left, the safe side, and at least
// 1 tick left. A negative dwell counts as 0 (validation refuses it). ok is false
// only when t - since overflows, which a validated tick cannot do.
func Decide(c, target int, since, t, dwell int64) (o Outcome, ok bool) {
	if target == c {
		return Outcome{Kind: Same}, true
	}
	dwell = max(dwell, 0)
	elapsed, ok := checked.Sub(t, since)
	if !ok {
		return Outcome{}, false
	}
	if elapsed >= dwell {
		return Outcome{Kind: Changed}, true
	}
	// elapsed < dwell here. A non-negative dwell less a non-negative elapsed cannot
	// overflow; a tick before since with a dwell of 0 is held with 1 tick left, so
	// the held count is never below 1.
	return Outcome{Kind: Held, TicksLeft: max(dwell-max(elapsed, 0), 1)}, true
}

// clampClass limits a class index to 0 through n-1 (0 for no class).
func clampClass(c, n int) int {
	return max(0, min(c, n-1))
}
