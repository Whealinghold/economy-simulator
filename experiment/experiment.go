// Package experiment runs sets of worlds of one base pack: sweeps over factors,
// replicas over seeds, branches from a checkpoint, integer reductions of the
// readings, and a canonical text of the result (spec-experiments.md). It is the
// "run again" of a tuning loop. It builds no world of its own, adds no
// configuration field and no hash, registers no event, and starts no goroutine:
// a unit of work (RunUnit) is a pure function of the experiment and a key, so a
// consumer may run the units of Plan on as many goroutines as it likes.
//
// An experiment is plain data and four function values (Prepare, Options, and
// the Fn and Do of measures and steps). It reads no file, no clock and no
// environment. The package imports the root package, config and its own
// internal helpers, and neither indicator nor internal/measure: a measure is the
// consumer's function (spec-experiments.md 3.4.0).
package experiment

import (
	"context"

	economysim "github.com/Whealinghold/economy-sim"
)

// Design is how the points of an experiment are generated from its factors.
type Design int

// The designs (spec-experiments.md 4).
const (
	// Grid is every combination of the levels, the last factor fastest.
	Grid Design = iota
	// OneAtATime is the baseline and then each factor's other levels alone.
	OneAtATime
	// Spread is a Latin hypercube of DesignN points.
	Spread
)

// Seeding is how replica seeds are chosen across points.
type Seeding int

// The seedings (spec-experiments.md 4.6).
const (
	// SameSeeds gives every point the seeds Seed0 + r: points are paired.
	SameSeeds Seeding = iota
	// PerPointSeeds gives each point its own seeds: points are independent.
	PerPointSeeds
)

// Op is how a criterion bounds its measure.
type Op int

// The criterion operators (spec-experiments.md 3.4.5).
const (
	// AtMost passes a reading of at most A.
	AtMost Op = iota
	// AtLeast passes a reading of at least A.
	AtLeast
	// Between passes a reading from A to B inclusive.
	Between
)

// Spec describes a set of runs (an experiment). The zero value is not valid.
type Spec struct {
	// Name is a label for the consumer; it is part of no key.
	Name string
	// Base is the documents of the pack, in layering order.
	Base [][]byte
	// Factors are the axes of the design; none gives one point, the base.
	Factors []Factor
	// Design is Grid, OneAtATime or Spread.
	Design Design
	// DesignN is the number of points of a Spread.
	DesignN int
	// DesignSalt makes another batch of a Spread of the same size.
	DesignSalt uint64
	// Replicas is the runs per point and variant, at least 1.
	Replicas int
	// Seed0 is the first replica seed.
	Seed0 uint64
	// Seeding is SameSeeds (the default) or PerPointSeeds.
	Seeding Seeding
	// Ticks is the length of a run.
	Ticks int64
	// Transient is the ticks before the first window opens, 0 to Ticks-1.
	Transient int64
	// Every is the ticks of one sample window; 0 means one window.
	Every int64
	// Script is the calls applied between ticks in every run.
	Script []Step
	// Measures are the readings of a run.
	Measures []Measure
	// Criteria are the bands on the final readings.
	Criteria []Criterion
	// Keep says what a record holds beyond the final readings.
	Keep Keep
	// Branch, when not nil, runs each unit's burn-in once and continues it.
	Branch *Branch
	// Prepare, when not nil, changes the resolved configuration of a point
	// before it is validated (spec-experiments.md 3.3.3).
	Prepare func(ctx context.Context, cfg economysim.Config) (economysim.Config, error)
	// Options returns fresh options and extensions for each world built. It
	// must return neither WithSeed nor WithReseed, and a new instance of every
	// stateful extension on each call (spec-experiments.md 5.5, N-129).
	Options func(Key) []economysim.Option
}

// A Factor is one axis of the design: a finite list of levels, or an integer
// range of one field path.
type Factor struct {
	// Name is a label; it is part of no key.
	Name string
	// Levels are explicit levels, at least 2; or
	Levels []Level
	// Path, Min and Max make a range factor of one field path.
	Path string
	// Min is the lowest value of a range factor.
	Min int64
	// Max is the highest value of a range factor.
	Max int64
	// Steps is the levels of a range factor in a Grid or OneAtATime, at least 2.
	Steps int
}

// A Level is one setting of a factor: integer changes, whole documents and calls.
type Level struct {
	// Label names the level for the consumer.
	Label string
	// Changes are a field path and an integer value each.
	Changes []Change
	// Docs are documents layered after the changes.
	Docs [][]byte
	// Steps are calls added to the script of every run at this level.
	Steps []Step
}

// A Change sets one field path to an integer.
type Change struct {
	// Path is the key path, for example "world.cost_floor".
	Path string
	// Value is the integer the path is set to.
	Value int64
}

// A Step is a call, or a few, applied at the boundary before a tick. It keeps no
// state between runs (spec-experiments.md 3.5.0).
type Step struct {
	// Tick is the boundary the step is applied at, in [0, Ticks).
	Tick int64
	// Label names the step for the consumer.
	Label string
	// Do makes the call; an error fails the run with the code script.
	Do func(w *economysim.World) error
}

// A Measure turns one window into one integer, or says it is undefined.
type Measure struct {
	// Name identifies the measure; criteria refer to it.
	Name string
	// Observer, when not nil, builds the observer a world of the run is given.
	Observer func(Key) economysim.Observer
	// Begin, when not nil, is called as a window opens.
	Begin func(k Key, open economysim.Snapshot, from int64, span bool)
	// Fork, when not nil, copies state from the burn-in's key to a branch's.
	Fork func(from, to Key)
	// Fn reads the window that has closed.
	Fn func(Sample) (value int64, ok bool, err error)
}

// A Sample is one closed window handed to Measure.Fn.
type Sample struct {
	// Config is the resolved configuration of the run.
	Config economysim.Config
	// Open and Close are the snapshots that bound the window.
	Open, Close economysim.Snapshot
	// From and To are the window's ticks [From, To).
	From, To int64
	// Span is true for the whole span [Transient, Ticks): the final reading.
	Span bool
	// Key is the run; a measure finds here what its Observer built.
	Key Key
}

// A Criterion is a named band on the final reading of one measure.
type Criterion struct {
	// Name identifies the criterion.
	Name string
	// Measure is a Measure.Name.
	Measure string
	// Op is AtMost, AtLeast or Between.
	Op Op
	// A is the limit, or the lower bound of Between.
	A int64
	// B is the upper bound of Between.
	B int64
}

// Keep says what a record holds beyond the final readings.
type Keep struct {
	// Series keeps the reading of each window.
	Series bool
	// Snapshot keeps the final snapshot of the run.
	Snapshot bool
}

// A Branch runs each unit's burn-in once and continues it in several ways.
type Branch struct {
	// At is the tick of the checkpoint, 1 to Ticks-1.
	At int64
	// Variants are the sets of calls applied from At, at least 1.
	Variants []Variant
	// Continuations is the reseeded continuations of each variant; 0 or 1: none.
	Continuations int
	// Salt0 is the first salt; continuation c uses Salt0 + c.
	Salt0 uint64
}

// A Variant is a set of calls applied from the branch point.
type Variant struct {
	// Label names the variant for the consumer.
	Label string
	// Steps each have Tick at or above Branch.At.
	Steps []Step
}
