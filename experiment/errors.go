package experiment

import "errors"

// The errors an experiment adds (spec-experiments.md 14.1). A record carries a
// Status and the error text, not one of these; they are what Validate, Plan, Run
// and Assemble return. ErrDuplicateID and ErrTooManyErrors are the root's.
var (
	// ErrBadSchedule is a length, a window, a replica count or a step tick
	// outside its range, or a step with no function.
	ErrBadSchedule = errors.New("bad schedule")
	// ErrBadFactor is a factor with no levels and no path, or both, a bad path,
	// or too many factors or levels.
	ErrBadFactor = errors.New("bad factor")
	// ErrFactorOverlap is a path set by two factors, or twice by one level.
	ErrFactorOverlap = errors.New("two factors set one path")
	// ErrBadDesign is a design or seeding outside its closed set, or a spread of
	// no points.
	ErrBadDesign = errors.New("bad design")
	// ErrBadBranch is a branch point or a variant outside its range.
	ErrBadBranch = errors.New("bad branch")
	// ErrUnknownMeasure is a criterion that names no measure, or a measure with
	// no function.
	ErrUnknownMeasure = errors.New("unknown measure")
	// ErrOversize is an experiment over a bound of bounds.go.
	ErrOversize = errors.New("experiment over a bound")
	// ErrUnknownRun is a record whose key is outside the plan.
	ErrUnknownRun = errors.New("record outside the plan")
	// ErrDuplicateRun is two different records of one key.
	ErrDuplicateRun = errors.New("two records of one run")
	// ErrRange is a reading beyond MaxReading.
	ErrRange = errors.New("reading beyond MaxReading")
)
