package experiment

// The bounds of an experiment (spec-experiments.md 13.2). They are rule
// constants of this package, not configuration (architecture.md 9.6), and
// placeholders until a measurement exists (N-015). Every product of two of them
// is computed with a checked multiplication, and an overflow is ErrOversize.
const (
	// MaxFactors is the most factors of one experiment.
	MaxFactors = 16
	// MaxLevels is the most levels of one factor (Steps of a range factor).
	MaxLevels = 1000
	// MaxPoints is the most points of a design, counted before any is generated.
	MaxPoints = 100000
	// MaxReplicas is the most replicas of one point and variant.
	MaxReplicas = 10000
	// MaxVariants is the most variants of a branch.
	MaxVariants = 64
	// MaxContinuations is the most reseeded continuations of a variant.
	MaxContinuations = 1024
	// MaxRuns is points times variants times replicas times max(1, continuations).
	MaxRuns = 1000000
	// MaxMeasures is the most measures of one experiment.
	MaxMeasures = 256
	// MaxWindows is the most sample windows of a run, Ticks over Every.
	MaxWindows = 10000
	// MaxCells is runs times measures times (windows + 1) with the series, and
	// runs times measures without it.
	MaxCells = 100000000
	// MaxReading is the largest magnitude of a reading: it keeps the sums of a
	// group below 2^126 for 10^6 used runs.
	MaxReading = int64(1) << 40
	// MaxErrors is the most errors Validate reports before ErrTooManyErrors.
	MaxErrors = 100
)
