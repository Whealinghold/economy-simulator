package tenure

import "github.com/Whealinghold/economy-sim/internal/checked"

// StepIn is the input of the rent step of one endowment (spec-land.md 5.5.1).
// Posted is the posted rent P and Base the configured rent B. StepBP, FloorBP
// and CeilBP are the configured fields of the same names, in basis points of
// Base. Unserved is X, the units asked of the let tier at the rent in force and
// not served. Idle is Y, the let supply not drawn whatever the reason, and Cut
// is C, the let units granted and cut for rent at the start, a part of Idle.
type StepIn struct {
	Posted   int64
	Base     int64
	StepBP   int64
	FloorBP  int64
	CeilBP   int64
	Unserved int64
	Idle     int64
	Cut      int64
	Closed   bool
}

// Step returns the posted rent of the next tick (spec-land.md 5.5.1):
//
//	if X > 0 and C = 0:  P' = min(ceil, P + step)
//	elif Y > 0:          P' = max(floor, P - step)
//	else:                P' = P
//
// with step = max(1, B * StepBP / 10 000), floor = B * FloorBP / 10 000 and ceil
// = B * CeilBP / 10 000, truncated toward zero. Nothing moves with a StepBP of 0
// or a closed endowment, and a Base of 0 has a ceiling of 0. ok is false for a
// negative number, a Cut above Idle (it is a part of it), or a product beyond
// int64 in the step or in the bound the taken branch uses, with Posted returned.
func Step(in StepIn) (next int64, ok bool) {
	if !in.valid() {
		return in.Posted, false
	}
	if in.StepBP == 0 || in.Closed {
		return in.Posted, true
	}
	step, ok := checked.MulDiv(in.Base, in.StepBP, BasisPoints)
	if !ok {
		return in.Posted, false
	}
	step = max(1, step)
	switch {
	case in.Unserved > 0 && in.Cut == 0:
		return raise(in, step)
	case in.Idle > 0:
		return lower(in, step)
	}
	return in.Posted, true
}

// raise is the rent after a rise by step, held under the ceiling. A Posted above
// the ceiling falls to it, as the formula of spec-land.md 5.5.1 reads.
func raise(in StepIn, step int64) (int64, bool) {
	ceil, ok := checked.MulDiv(in.Base, in.CeilBP, BasisPoints)
	up, ok2 := checked.Add(in.Posted, step)
	if !ok || !ok2 {
		return in.Posted, false
	}
	return min(ceil, up), true
}

// lower is the rent after a fall by step, held over the floor.
func lower(in StepIn, step int64) (int64, bool) {
	floor, ok := checked.MulDiv(in.Base, in.FloorBP, BasisPoints)
	if !ok {
		return in.Posted, false
	}
	return max(floor, in.Posted-step), true // Posted and step are at least 0 and 1, and the difference cannot overflow
}

// valid reports whether no field is negative and Cut is within Idle.
func (in StepIn) valid() bool {
	return in.Cut <= in.Idle && min(in.Posted, in.Base, in.StepBP, in.FloorBP, in.CeilBP, in.Unserved, in.Idle, in.Cut) >= 0
}
