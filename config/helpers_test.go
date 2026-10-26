package config_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode"

	economysim "github.com/Whealinghold/economy-sim"
)

// loaderSentinels are the rejections of the loader (spec-config.md 6.3).
var loaderSentinels = []error{
	economysim.ErrSyntax, economysim.ErrUnknownKey, economysim.ErrNotInteger,
	economysim.ErrWrongType, economysim.ErrMissingField, economysim.ErrUnsupportedSchema,
	economysim.ErrMergeConflict, economysim.ErrTooLarge, economysim.ErrTooManyErrors,
}

// validationSentinels are the rules of Config.Validate (errors.go).
var validationSentinels = []error{
	economysim.ErrInvalidID, economysim.ErrDuplicateID, economysim.ErrUnknownCommodity,
	economysim.ErrUnknownRole, economysim.ErrUnknownRecipe, economysim.ErrUnknownRegion,
	economysim.ErrRecipeNoOutput, economysim.ErrOrphanCommodity, economysim.ErrUnproducible,
	economysim.ErrOutOfBounds, economysim.ErrOverflow, economysim.ErrTooMany,
	economysim.ErrNoRegions, economysim.ErrModuleDependency, economysim.ErrBasketMismatch,
	economysim.ErrCalendarTick, economysim.ErrCalendarSeasons, economysim.ErrCostShares,
	economysim.ErrToolNotDurable, economysim.ErrEligibleHasTool, economysim.ErrUnknownSeason,
	economysim.ErrWarmStartPrice, economysim.ErrWarmStartJob, economysim.ErrSelfEdge,
	economysim.ErrUnknownWant, economysim.ErrWantItems, economysim.ErrWantDurable,
}

// validationErrors returns every *ValidationError of an error, in the order of
// a joined error's list.
func validationErrors(err error) []*economysim.ValidationError {
	var out []*economysim.ValidationError
	var walk func(error)
	walk = func(e error) {
		if j, ok := e.(interface{ Unwrap() []error }); ok {
			for _, c := range j.Unwrap() {
				walk(c)
			}
			return
		}
		var ve *economysim.ValidationError
		if errors.As(e, &ve) {
			out = append(out, ve)
		}
	}
	walk(err)
	return out
}

// viabilityRules are the five rules of a *ViabilityError (spec-viability.md 10.1).
var viabilityRules = []string{"unproducible_in_realm", "role_cannot_work", "cycle_not_productive", "labour_short", "storage_gap"}

// viabilityErrors returns every *ViabilityError of an error, in the order of a
// joined error's list.
func viabilityErrors(err error) []*economysim.ViabilityError {
	var out []*economysim.ViabilityError
	var walk func(error)
	walk = func(e error) {
		if j, ok := e.(interface{ Unwrap() []error }); ok {
			for _, c := range j.Unwrap() {
				walk(c)
			}
			return
		}
		var ve *economysim.ViabilityError
		if errors.As(e, &ve) {
			out = append(out, ve)
		}
	}
	walk(err)
	return out
}

// hasError reports whether err holds a ValidationError whose rule is rule and
// whose path is path.
func hasError(err error, rule error, path []string) bool {
	for _, ve := range validationErrors(err) {
		if errors.Is(ve, rule) && strings.Join(ve.Path, "\x00") == strings.Join(path, "\x00") {
			return true
		}
	}
	return false
}

// describe lists the (rule, path) of every ValidationError, for a failure
// message.
func describe(err error) string {
	var b strings.Builder
	for _, ve := range validationErrors(err) {
		fmt.Fprintf(&b, "[%v %q] ", ve.Rule, ve.Path)
	}
	if b.Len() == 0 {
		return fmt.Sprint(err)
	}
	return b.String()
}

// padded returns a valid document of exactly n bytes: the schema line followed
// by comment lines of at most 1 000 bytes.
func padded(n int) []byte {
	b := []byte("schema = 1\n")
	for len(b) < n {
		line := min(n-len(b), 1000)
		if line == 1 {
			b = append(b, '\n')
			continue
		}
		b = append(b, '#')
		b = append(b, bytes.Repeat([]byte{'x'}, line-2)...)
		b = append(b, '\n')
	}
	return b
}

// dotted returns a key of n segments, "a.a.a".
func dotted(n int) string {
	return strings.Repeat("a.", n-1) + "a"
}

// flatKeys returns n lines "k0 = 1".
func flatKeys(prefix string, n int) string {
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "%sk%d = 1\n", prefix, i)
	}
	return b.String()
}

// snake is the snake case of a Go field name by the rule of spec-config.md 6.2:
// a run of capitals is one word, so RateBP is rate_bp and ID is id.
func snake(name string) string {
	rs := []rune(name)
	var b strings.Builder
	for i, r := range rs {
		if i > 0 && unicode.IsUpper(r) {
			prevLower := unicode.IsLower(rs[i-1]) || unicode.IsDigit(rs[i-1])
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if prevLower || (unicode.IsUpper(rs[i-1]) && nextLower) {
				b.WriteByte('_')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
