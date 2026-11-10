package calibration

// Record is a calibration record (spec-calibration.md 8): the typed lines of the
// grammar of 8.2, in the order that grammar gives. A list holds its lines in the
// order of its subject. Closed words (a design, a decision, a result) are kept as
// the strings the grammar spells, because the parser, not this type, owns the
// closed sets. A value that the grammar writes as `-` for none is an Opt or an
// empty string.
type Record struct {
	Pack      string
	Lineage   string // 16 hex digits
	Stage     string // C0 to C9, with the sub-stages C4a to C4c and C7a, C7b
	Attempt   int
	Block     int
	Commit    string
	Parent    string // the predecessor's lineage, "" for `-`
	Base      string // config base: 16 hex digits
	Chosen    string // config chosen: 16 hex digits, or "same"
	Documents []Document

	Experiments []Experiment
	Factors     []Factor
	Unsearched  []Unsearched
	Measures    []Measure
	Criteria    []Criterion
	Screens     []Screen
	ScreenedIn  []string // knob IDs
	Dormant     []string // knob IDs
	Findings    []Finding
	Feasible    Feasible
	Pick        Pick
	Pareto      []int
	Sets        []Setting
	Confirm     ConfirmLine
	Facts       []Fact
	Detectors   []Detector
	Stability   []Stability
	Claims      []Claim
	Unmet       []Unmet
	Decision    string // stop, escalate, reentry C<n>, review or revert
	Notes       []string
}

// Opt is an integer the grammar may write as `-`.
type Opt struct {
	Has   bool
	Value int64
}

// Document is a `document` line: a document name and the digest of its bytes.
type Document struct{ Name, Digest string }

// Experiment is an `experiment` line: one step and the plain data of its Spec.
type Experiment struct {
	Step, Design string
	Points       int
	Replicas     int
	Seed0        uint64
	Seeding      string
	Ticks        int64
	Transient    int64
	Every        int64
	Salt         uint64
	Render       string // digest of the rendered Result
}

// Factor is a `factor` line: a searched key, its grid and its edge.
type Factor struct {
	Name, Key, Grid string
	Base            int64
	Lower, Upper    Opt
	Edge            string
}

// Unsearched is an `unsearched` line: why (at_off, invisible or v1x) and which knobs.
type Unsearched struct {
	Why   string
	Knobs []string
}

// Measure is a `measure` line.
type Measure struct {
	Name, Fact, Token string
	Class             byte // X, O, B, R or P
	Kind              string
}

// Criterion is a `criterion` line.
type Criterion struct {
	Name, Measure, Op string
	A, B              int64
	Target            string
}

// Screen is a `screen` line.
type Screen struct {
	Knob, Measure    string
	ElLower, ElUpper Opt
	Range, ShareBP   int64
}

// Finding is a `finding` line.
type Finding struct {
	Knob, Fact, Measure string
	Expected, Measured  byte // a sign: + - 0 ~ ?
	Status              string
}

// Feasible is the `feasible n of N` line; Has is false when it is absent.
type Feasible struct {
	Has   bool
	N, Of int
}

// Pick is the `chosen` line: the point index, or None, and its grid distance.
type Pick struct {
	Has      bool
	None     bool
	Index    int
	Distance int64
}

// Setting is a `set` line: a registry key (or the configuration path of a table
// keyed by an ID) and the value the overlay writes.
type Setting struct {
	Key   string
	Value int64
}

// ConfirmLine is the `confirm seeds a..b control yes|no` line; Has is false when absent.
type ConfirmLine struct {
	Has         bool
	First, Last uint64
	Control     bool
}

// Fact is a `fact` line.
type Fact struct {
	ID               string
	Class            byte
	Key              string
	Target           string
	Min, Median, Max int64
	Outside          []uint64
	Result           string
}

// Detector is a `detector` line.
type Detector struct {
	ID, Verdict string
	Working     bool
	Onset       Opt
}

// Stability is a `stability` line.
type Stability struct {
	Knob, Pack, Point string
	Value             int64
	Set               string // gate or holdout
	First, Last       uint64
	InSample          bool
	CBP, CNC, CWS     string
	Edge              string
}

// Claim is a `claim` line.
type Claim struct{ Knob, Pack, Word string }

// Unmet is an `unmet` line.
type Unmet struct{ Fact, Measure, Values string }
