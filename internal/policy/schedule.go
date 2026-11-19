package policy

// Step is one step of a fixed schedule: from tick At the lever Lever takes the
// value BP, closed when Closed (spec-policy.md 8.2.1). Lever is the index of the
// lever in the actor's lever list.
type Step struct {
	Lever  int
	At     int64
	BP     int64
	Closed bool
}

// ScheduleValue returns the value of lever at tick for a list of steps, normally
// sorted by (Lever, At); the result does not depend on the order. ok is false when
// the schedule holds the lever (spec-policy.md 8.2.1). With cycle 0 the value is that of the step with the largest At at or
// below tick. With a cycle, u is tick in the first cycle and tick mod cycle
// after it; a lever with no step at or below u is held in the first cycle and
// takes the value of its last step once the cycle has wrapped (the value carried
// across the wrap). A cycle below 0 is taken as none.
func ScheduleValue(steps []Step, lever int, cycle, tick int64) (bp int64, closed bool, ok bool) {
	wrapped := cycle > 0 && tick >= cycle
	u := tick
	if wrapped {
		u = tick % cycle
	}
	best, last := bounds(steps, lever, u)
	switch {
	case best != nil:
		return best.BP, best.Closed, true
	case wrapped && last != nil:
		return last.BP, last.Closed, true
	}
	return 0, false, false
}

// bounds returns the step of lever with the largest At at or below u (nil for
// none) and the step of lever with the largest At (nil for none), whatever the
// order of the steps; of two steps with one At the later in the list wins.
func bounds(steps []Step, lever int, u int64) (best, last *Step) {
	for i := range steps {
		s := &steps[i]
		if s.Lever != lever {
			continue
		}
		if last == nil || s.At >= last.At {
			last = s
		}
		if s.At <= u && (best == nil || s.At >= best.At) {
			best = s
		}
	}
	return best, last
}
