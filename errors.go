package economysim

import (
	"errors"
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
