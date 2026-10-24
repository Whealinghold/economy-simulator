package economysim

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The structural rules of the configuration (spec-config.md 7.3). Validate
// returns each as the Rule of a *ValidationError, so errors.Is matches the
// sentinel and errors.As finds the typed error.
var (
	// ErrInvalidID is an ID that does not match [a-z][a-z0-9_]* or is longer
	// than MaxIDLength bytes.
	ErrInvalidID = errors.New("invalid ID")
	// ErrDuplicateID is two entities of one kind with the same ID.
	ErrDuplicateID = errors.New("duplicate ID")
	// ErrUnknownCommodity is a reference to a commodity that is not defined.
	ErrUnknownCommodity = errors.New("unknown commodity")
	// ErrUnknownRole is a reference to a role that is not defined.
	ErrUnknownRole = errors.New("unknown role")
	// ErrUnknownRecipe is a reference to a recipe that is not defined.
	ErrUnknownRecipe = errors.New("unknown recipe")
	// ErrUnknownRegion is a reference to a region that is not defined.
	ErrUnknownRegion = errors.New("unknown region")
	// ErrRecipeNoOutput is a recipe with no output.
	ErrRecipeNoOutput = errors.New("recipe has no output")
	// ErrOrphanCommodity is a commodity that no recipe makes or uses and that
	// no upkeep, tool or boundary mentions.
	ErrOrphanCommodity = errors.New("orphan commodity")
	// ErrUnproducible is a commodity that something consumes but that cannot be
	// produced from a raw source.
	ErrUnproducible = errors.New("commodity cannot be produced")
	// ErrOutOfBounds is a value outside the bound of its field or of a constant
	// of bounds.go.
	ErrOutOfBounds = errors.New("value out of bounds")
	// ErrOverflow is a derived product that does not fit in int64.
	ErrOverflow = errors.New("derived value overflows")
	// ErrTooMany is more entities of a kind than its bound. It is checked first
	// and stops the rules that are super-linear in a count.
	ErrTooMany = errors.New("too many entities")
	// ErrNoRegions is a configuration with no region.
	ErrNoRegions = errors.New("no regions")
	// ErrModuleDependency is a module's configuration that refers to another
	// module that is off. No Stage 0 rule returns it, since no module is built;
	// the module changes that add a dependency are its first users.
	ErrModuleDependency = errors.New("module depends on a module that is off")
)

// The rules that other specs contribute to Validate (spec-config.md 7.4.1).
// Each has a failing-case test; the sentinels of the modules are appended with
// the module that returns them.
var (
	// ErrBasketMismatch is a region whose ReferenceBasket differs from the cost
	// of the baseline upkeep plus its ExtraUpkeep, at the effective reference
	// prices, by more than World.BasketTolerance basis points of the basket.
	ErrBasketMismatch = errors.New("reference basket does not match the upkeep")
	// ErrCalendarTick is a calendar whose tick is both a part of a day and a
	// number of days, or whose year is not a whole number of ticks.
	ErrCalendarTick = errors.New("calendar tick does not divide the year")
	// ErrCalendarSeasons is a season of fewer than one day, or seasons whose
	// days do not add up to the days of the year.
	ErrCalendarSeasons = errors.New("seasons do not cover the year")
	// ErrCostShares is the cost shares of a recipe's outputs that do not add to
	// 10 000, or a discard output with a share above 0.
	ErrCostShares = errors.New("output cost shares do not add up")
	// ErrToolNotDurable is a recipe that requires a commodity with a durability
	// of 0 as a tool.
	ErrToolNotDurable = errors.New("tool is not durable")
	// ErrEligibleHasTool is a trainee-eligible recipe that requires a tool.
	ErrEligibleHasTool = errors.New("trainee-eligible recipe requires a tool")
	// ErrUnknownSeason is a yield entry for a season that the calendar lacks.
	ErrUnknownSeason = errors.New("unknown season")
	// ErrWarmStartPrice is a seed price or a starting wage of the warm start
	// that is outside its bounds.
	ErrWarmStartPrice = errors.New("warm start price out of bounds")
	// ErrWarmStartJob is a job row of the warm start that names a recipe the
	// role does not list, or whose shares are outside 0 to 10 000.
	ErrWarmStartJob = errors.New("warm start job is invalid")
)

// The rejections of the file loader of package config (spec-config.md 6.3). The
// loader wraps them with the path of the key, so a consumer matches them with
// errors.Is.
var (
	// ErrSyntax is a document that is not TOML.
	ErrSyntax = errors.New("syntax error")
	// ErrUnknownKey is a key that the configuration does not have, at any depth,
	// including a module table of a module that is not built.
	ErrUnknownKey = errors.New("unknown key")
	// ErrNotInteger is a floating point literal where an integer is needed.
	ErrNotInteger = errors.New("not an integer")
	// ErrWrongType is a value of the wrong type that is not a float: a string
	// where a number is needed, an integer where a string is needed, a date.
	ErrWrongType = errors.New("wrong type")
	// ErrMissingField is a required field that is absent.
	ErrMissingField = errors.New("missing required field")
	// ErrUnsupportedSchema is a missing or unsupported schema version.
	ErrUnsupportedSchema = errors.New("missing or unsupported schema")
	// ErrMergeConflict is a key that is a table in one document and a scalar in
	// another.
	ErrMergeConflict = errors.New("table and scalar conflict")
	// ErrTooLarge is an input over a limit of the loader: documents, size,
	// line, nesting depth or keys of one table.
	ErrTooLarge = errors.New("input too large")
)

// ErrTooManyErrors ends a list of errors that was cut at 100: it says how many
// were left out. Both Validate and the loader return it.
var ErrTooManyErrors = errors.New("too many errors")

// The errors of registration and of the name check (spec-world.md 4.1.1, 4.3).
var (
	// ErrNilExtension is a registration option given a nil extension.
	ErrNilExtension = errors.New("nil extension")
	// ErrUnknownClearing is a configured World.Clearing that is empty or names
	// no registered clearing; NewWorld returns it as a *ValidationError.
	ErrUnknownClearing = errors.New("unknown clearing")
	// ErrUnknownSpawner is a configured World.Spawner that is empty or names no
	// registered spawner; NewWorld returns it as a *ValidationError.
	ErrUnknownSpawner = errors.New("unknown spawner")
	// ErrUnknownStrategy is a configured Role.Strategy that names no registered
	// strategy; NewWorld returns it as a *ValidationError.
	ErrUnknownStrategy = errors.New("unknown strategy")
)

// The errors of the market step (spec-clearing.md 8.2).
var (
	// ErrInvalidTrade is a list of trades from a Clearing that breaks a rule of
	// spec-clearing.md 8.2: the step settles none of that market's trades,
	// releases its reservations and fails with an error that wraps it.
	ErrInvalidTrade = errors.New("invalid trade list")
)

// The errors of the world's calls (spec-world.md 3.3.3, 3.5, 3.6). Each is
// matched with errors.Is and none is logged by the library.
var (
	// ErrWorldFailed is a call on a world that an earlier step failed or whose
	// step panicked. The error wraps the cause too, and Err returns it.
	ErrWorldFailed = errors.New("world failed")
	// ErrInStep is a call made from inside a running step, by an observer or an
	// extension. It changes nothing.
	ErrInStep = errors.New("call inside a step")
	// ErrAuditFailed is an audit that found a violated identity: at
	// construction with WithAudit, at the end of a step with WithAudit, or in a
	// call of Audit. The error wraps the failing check as well.
	ErrAuditFailed = errors.New("audit failed")
)

// The viability rules and the solver (spec-viability.md 10, 5.4; spec-config.md
// 7.6). Validate returns ErrNotViable inside a *ViabilityError; the other three
// are the sentinels of SolveSteadyState.
var (
	// ErrNotViable is a configuration that cannot work: a *ViabilityError carries
	// the rule and unwraps to it, so errors.Is matches every viability rule.
	ErrNotViable = errors.New("configuration is not viable")
	// ErrNoUpkeep is a solve of a configuration in which no group has an upkeep:
	// the programme is unbounded and there is nothing to be viable for.
	ErrNoUpkeep = errors.New("no group has an upkeep")
	// ErrSolveLimit is a solve that reached MaxViabilityPivots, or the option
	// that lowers it, or whose tableau is over MaxViabilityCells; a *SolveError
	// carries the counts and unwraps to it.
	ErrSolveLimit = errors.New("solver limit reached")
	// ErrSolveInconsistent is a result that failed the exact certificate. No
	// input can cause it: it is a defect of the solver.
	ErrSolveInconsistent = errors.New("solver certificate failed")
)

// The checkpoint and the option context (spec-checkpoint.md 6.3, 13.1): each is
// matched with errors.Is, the error that wraps it names what failed, and none is
// logged by the library.
var (
	// ErrCheckpointVersion is a checkpoint whose format version this library does
	// not read. The error that wraps it carries the version read and the one
	// supported.
	ErrCheckpointVersion = errors.New("checkpoint format version not supported")
	// ErrCheckpointCorrupt is bytes that are not a checkpoint: too short, a wrong
	// magic or digest, a malformed value, a section that is wrong or a state that
	// fails a cross-check or the audit. The error that wraps it names the tag or
	// the offset.
	ErrCheckpointCorrupt = errors.New("checkpoint is corrupt")
	// ErrCheckpointConfig is a checkpoint taken from a world of another
	// configuration: the hashes differ. The error that wraps it carries both.
	ErrCheckpointConfig = errors.New("checkpoint is of another configuration")
	// ErrCheckpointExtension is an extension that does not match the checkpoint,
	// or a SaveState that failed: the error that wraps it names the extension and
	// wraps the extension's own error.
	ErrCheckpointExtension = errors.New("checkpoint extension mismatch")
	// ErrOptionContext is an option that is not valid in this call: WithSeed on a
	// restore, WithReseed on NewWorld.
	ErrOptionContext = errors.New("option not valid in this call")
)

// ValidationError is one rejected rule of a configuration (spec-config.md
// 7.1.1). Rule is the sentinel, so errors.Is(err, ErrOutOfBounds) matches and
// errors.As finds the ValidationError.
//
// The fields hold the truth as given, so a value or a path segment may be long
// or hold control characters. Error cuts and quotes them: every segment and
// every string value is cut at the last rune boundary at or before MaxIDLength
// bytes and quoted, so the text holds no control character and no more than
// MaxNestingDepth segments (spec-config.md 6.3.2).
type ValidationError struct {
	// Path is the segments of the path of the Go field, in the plural list
	// names of the sections and the IDs of the entities as given, for example
	// roles, worker_a, target_ratio. It is not the key path of a file.
	Path []string
	// Rule is the sentinel of the rule that failed.
	Rule error
	// Value is the offending value as text: an integer in decimal, a string as
	// given.
	Value string
	// Bound is the bound that Value breaks, when the rule is a bound: a
	// decimal, or the name of the constant or the derived quantity.
	Bound string
}

// Error returns the path, the rule and, when set, the value and the bound.
func (e *ValidationError) Error() string {
	var b strings.Builder
	b.WriteString(quotePath(e.Path))
	b.WriteString(": ")
	if e.Rule != nil {
		b.WriteString(e.Rule.Error())
	} else {
		b.WriteString("invalid configuration")
	}
	if e.Value != "" {
		b.WriteString(" (value ")
		b.WriteString(quoteValue(e.Value))
		if e.Bound != "" {
			b.WriteString(", bound ")
			b.WriteString(quoteValue(e.Bound))
		}
		b.WriteByte(')')
	}
	return b.String()
}

// Unwrap returns the rule, so errors.Is matches its sentinel.
func (e *ValidationError) Unwrap() error { return e.Rule }

// quotePath writes the segments of a path joined by dots, each cut and quoted,
// at most MaxNestingDepth of them (spec-config.md 6.3.2). A longer path ends in
// three dots.
func quotePath(path []string) string {
	var b strings.Builder
	for i, s := range path {
		if i == MaxNestingDepth {
			b.WriteString("...")
			break
		}
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(quoteSegment(s))
	}
	return b.String()
}

// quoteValue writes an integer in canonical decimal form (no sign other than
// '-', no leading zeros, at most 20 bytes) as is and any other text as a
// quoted, cut string, so a numeric-looking ID is still quoted and nothing
// uncut can pass.
func quoteValue(s string) string {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && len(s) <= 20 && strconv.FormatInt(n, 10) == s {
		return s
	}
	return quoteSegment(s)
}

// quoteSegment cuts s at the last rune boundary at or before MaxIDLength bytes
// and writes it with strconv.Quote, followed by three dots outside the quotes
// when it was cut.
func quoteSegment(s string) string {
	cut := len(s) > MaxIDLength
	if cut {
		n := MaxIDLength
		for n > 0 && !utf8.RuneStart(s[n]) {
			n--
		}
		s = s[:n]
	}
	q := strconv.Quote(s)
	if cut {
		q += "..."
	}
	return q
}

// ViabilityError is one rejected viability rule of a configuration
// (spec-viability.md 10.1, spec-config.md 7.6). It is its own type and does not
// embed ValidationError. Unwrap returns ErrNotViable, so errors.Is(err,
// ErrNotViable) matches every viability rule and errors.As finds the rule, path,
// value and bound. Validate puts it in the one sorted list with the
// ValidationErrors.
type ViabilityError struct {
	// Rule is the name of the rule: unproducible_in_realm, role_cannot_work,
	// cycle_not_productive, labour_short or storage_gap.
	Rule string
	// Path is the segments of the path of the entity the rule is about, as a
	// ValidationError's Path is.
	Path []string
	// Value is the offending value as text.
	Value string
	// Bound is the number that decides the rule, as text; empty when none does.
	Bound string
}

// Error returns the path, the rule and, when set, the value and the bound, cut
// and quoted as ValidationError.Error does.
func (e *ViabilityError) Error() string {
	var b strings.Builder
	b.WriteString(quotePath(e.Path))
	b.WriteString(": ")
	b.WriteString(ErrNotViable.Error())
	b.WriteString(": ")
	b.WriteString(quoteSegment(e.Rule))
	if e.Value != "" {
		b.WriteString(" (value ")
		b.WriteString(quoteValue(e.Value))
		if e.Bound != "" {
			b.WriteString(", bound ")
			b.WriteString(quoteValue(e.Bound))
		}
		b.WriteByte(')')
	}
	return b.String()
}

// Unwrap returns ErrNotViable.
func (e *ViabilityError) Unwrap() error { return ErrNotViable }

// SolveError is a solve that stopped on a limit: it carries the pivots made and
// the size of the tableau, and unwraps to ErrSolveLimit.
type SolveError struct {
	// Pivots is the number of pivots made, or 0 when the tableau was over
	// MaxViabilityCells and never built.
	Pivots int64
	// Rows and Columns are the size of the tableau of the programme.
	Rows, Columns int
}

// Error returns the counts.
func (e *SolveError) Error() string {
	return fmt.Sprintf("%s: %d pivots, %d rows, %d columns", ErrSolveLimit.Error(), e.Pivots, e.Rows, e.Columns)
}

// Unwrap returns ErrSolveLimit.
func (e *SolveError) Unwrap() error { return ErrSolveLimit }
