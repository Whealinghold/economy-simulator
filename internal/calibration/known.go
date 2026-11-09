package calibration

// Known is the data the record checks need from the registry and from STY
// (plan 2-21 D626). The package never reads a file: the registry test fills every
// field from the parse of the registry rows and of STY 5.11.
type Known struct {
	// Knobs holds every registry row, by its ID (K-BEL12, S-CAL02).
	Knobs map[string]KnobInfo
	// Facts maps a STY fact ID without its prefix (P02) to its class (X O B R P).
	Facts map[string]byte
	// Detectors holds the regime IDs (R5a).
	Detectors map[string]bool
}

// KnobInfo is one registry row as the checks read it.
type KnobInfo struct {
	Class byte   // K, D, S or P
	Key   string // Scope.Field
	Stage string // C1 to C9: the stage that searches or sets it
	Owner string // the owner scope; it splits a stage into sub-stages
	V1x   bool   // flag v1x: no reader in version 1
	Z     bool   // flag z: a base of 0 has no grid
	Off   Off    // the off value of a D row, resolved by the caller
	Moves map[string]MoveSign
}

// MoveSign is the expected sign and its source for one Moves token of a row
// (4.4): a sign from + - 0 ~ ? and a source from S M E H.
type MoveSign struct{ Sign, Source byte }
