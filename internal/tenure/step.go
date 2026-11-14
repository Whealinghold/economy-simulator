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
// negative number or a product beyond int64, with Posted returned.
func Step(in StepIn) (next int64, ok bool) {
	if !in.valid() {
		return in.Posted, false
	}
	if in.StepBP == 0 || in.Closed {
		return in.Posted, true
	}
	step, floor, ceil, ok := band(in)
	if !ok {
		return in.Posted, false
	}
	switch {
	case in.Unserved > 0 && in.Cut == 0:
		up, ok := checked.Add(in.Posted, step)
		if !ok {
			return in.Posted, false
		}
		return min(ceil, up), true
	case in.Idle > 0:
		return max(floor, in.Posted-step), true // Posted and step are at least 0 and 1, and the difference cannot overflow
	}
	return in.Posted, true
}

// valid reports whether no field is negative.
func (in StepIn) valid() bool {
	return min(in.Posted, in.Base, in.StepBP, in.FloorBP, in.CeilBP, in.Unserved, in.Idle, in.Cut) >= 0
}

// band returns the step, the floor and the ceiling of the posted rent.
func band(in StepIn) (step, floor, ceil int64, ok bool) {
	step, ok1 := checked.MulDiv(in.Base, in.StepBP, BasisPoints)
	floor, ok2 := checked.MulDiv(in.Base, in.FloorBP, BasisPoints)
	ceil, ok3 := checked.MulDiv(in.Base, in.CeilBP, BasisPoints)
	return max(1, step), floor, ceil, ok1 && ok2 && ok3
}
