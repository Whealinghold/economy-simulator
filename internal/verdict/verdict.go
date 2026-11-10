// Package verdict judges a run: the failure-regime detectors, the seed rules,
// the band-setting rule and the separation protocol of spec-stylised-facts.md
// (STY), as pure functions over plain integer series.
//
// It imports the standard library only, takes no world, snapshot or event,
// draws nothing, reads no clock and logs nothing: a verdict is a return value
// (observability-and-logging.md 2.3). The caller copies the results of the
// indicator presets into the plain series of this package inline
// (architecture.md 2.9).
package verdict

import "strconv"

// Point is one value of a series at a tick. A window series carries the first
// tick of the window.
type Point struct {
	Tick, Value int64
}

// Class is the class of a verdict (STY 6.1.2).
type Class uint8

// The three verdict classes. Unknown is not None: a rule that could not be
// evaluated says so, and silence is not health.
const (
	// None means the series was long enough, its blind-spot fields were inside
	// their bounds and the rule did not fire.
	None Class = iota
	// Detected means the rule fired.
	Detected
	// Unknown means the rule could not be evaluated; the verdict carries a
	// Reason.
	Unknown
)

// String returns the word of the class as the verdict record spells it.
func (c Class) String() string {
	switch c {
	case None:
		return "none"
	case Detected:
		return "detected"
	case Unknown:
		return "unknown"
	}
	return "invalid"
}

// The closed set of reasons of an unknown verdict (STY 6.1.2, 4.3). They are
// rule constants: a record spells them as written here.
const (
	// ReasonWindows: fewer windows (or ticks) than the rule needs.
	ReasonWindows = "windows"
	// ReasonStale: the stale share is above stale_max_bp.
	ReasonStale = "stale"
	// ReasonActive: the active share is below active_min_bp.
	ReasonActive = "active"
	// ReasonThresholdUnset: a key the rule needs has no value.
	ReasonThresholdUnset = "threshold unset"
	// ReasonReferenceMissing: the series or the reference the rule needs is absent.
	ReasonReferenceMissing = "reference missing"
	// ReasonNeedsLongerRun: the run ends before the rule's horizon.
	ReasonNeedsLongerRun = "needs a longer run"
	// ReasonClamped: an index or ratio the rule reads is clamped.
	ReasonClamped = "clamped"
	// ReasonOutcomes: a share of nothing: no outcomes in the window.
	ReasonOutcomes = "outcomes"
)

// Detector names one failure regime of STY 6.2.
type Detector string

// The ten detectors: nine regimes, R5 in two parts.
const (
	R1  Detector = "R1"  // runaway inflation
	R2  Detector = "R2"  // hoarding
	R3  Detector = "R3"  // collapse
	R4  Detector = "R4"  // stagnation
	R5a Detector = "R5a" // pinned market
	R5b Detector = "R5b" // frozen market
	R6  Detector = "R6"  // cobweb blow-up
	R7  Detector = "R7"  // synchronisation
	R8  Detector = "R8"  // offset
	R9  Detector = "R9"  // deflation with distress
)

// Detectors returns the ten detectors in the order of STY 6.2.
func Detectors() []Detector {
	return []Detector{R1, R2, R3, R4, R5a, R5b, R6, R7, R8, R9}
}

// Verdict is the result of one detector.
type Verdict struct {
	Detector Detector
	Class    Class
	// Onset is the tick of the window or tick at which the rule completed
	// (Detected only); it is a tick of a point of the input.
	Onset int64
	// Value is the measured statistic at the onset and Threshold the key it
	// was compared with (Detected only).
	Value, Threshold int64
	// Reason is one of the Reason constants (Unknown only).
	Reason string
}

// String renders the verdict as the verdict record does: integers and words.
func (v Verdict) String() string {
	switch v.Class {
	case Detected:
		return string(v.Detector) + " detected onset " + strconv.FormatInt(v.Onset, 10) + " value " + strconv.FormatInt(v.Value, 10) + " threshold " + strconv.FormatInt(v.Threshold, 10)
	case Unknown:
		return string(v.Detector) + " unknown (" + v.Reason + ")"
	}
	return string(v.Detector) + " " + v.Class.String()
}
