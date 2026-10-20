package economysim_test

import (
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// Equivalent mutants of errors.go that no test can kill (gomu, one-line why each):
//   - ValidationError.Error and ViabilityError.Error `Value != "" -> >` and
//     `Bound != "" -> >` (190, 193, 281, 284): for strings, s > "" is s != "".
//   - quotePath `i == MaxNestingDepth -> >=` (211): the loop breaks at the first
//     i == MaxNestingDepth, so i never exceeds it.
//   - quotePath `i > 0 -> !=` (215): i is never negative.
//   - quoteValue `&& -> ||` (228, both): the mutated operator is the one between
//     `err == nil` and the length test; a failed parse has n 0 and the canonical
//     test then fails or the result is the same as the original.
//   - quoteValue `20 -> 21` (228): a canonical int64 is at most 20 bytes, so the
//     length test never decides alone.
//   - quoteValue `FormatInt == s -> <=` (228): a non-canonical spelling differs
//     only by a leading '+' or '0', which sort below the canonical text, so <=
//     is never true where == is false.
//   - quoteSegment `n > 0 -> !=` (241): n never goes below 0.
//   - quoteSegment `n--` removed or its block emptied (241, 242): an infinite
//     loop; go test -timeout reports it, gomu records TIMED_OUT.

// TestErrorsMutation_SentinelText verifies the exact message of every sentinel,
// as the text of an error is what a log or a caller shows.
//
// Guard: replacing a message by "" would leave the sentinels distinct only by
// pointer, and the other tests compare them by identity.
func TestErrorsMutation_SentinelText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		err  error
		want string
	}{
		{economysim.ErrInvalidID, "invalid ID"},
		{economysim.ErrDuplicateID, "duplicate ID"},
		{economysim.ErrUnknownCommodity, "unknown commodity"},
		{economysim.ErrUnknownRole, "unknown role"},
		{economysim.ErrUnknownRecipe, "unknown recipe"},
		{economysim.ErrUnknownRegion, "unknown region"},
		{economysim.ErrRecipeNoOutput, "recipe has no output"},
		{economysim.ErrOrphanCommodity, "orphan commodity"},
		{economysim.ErrUnproducible, "commodity cannot be produced"},
		{economysim.ErrOutOfBounds, "value out of bounds"},
		{economysim.ErrOverflow, "derived value overflows"},
		{economysim.ErrTooMany, "too many entities"},
		{economysim.ErrNoRegions, "no regions"},
		{economysim.ErrModuleDependency, "module depends on a module that is off"},
		{economysim.ErrBasketMismatch, "reference basket does not match the upkeep"},
		{economysim.ErrCalendarTick, "calendar tick does not divide the year"},
		{economysim.ErrCalendarSeasons, "seasons do not cover the year"},
		{economysim.ErrCostShares, "output cost shares do not add up"},
		{economysim.ErrToolNotDurable, "tool is not durable"},
		{economysim.ErrEligibleHasTool, "trainee-eligible recipe requires a tool"},
		{economysim.ErrUnknownSeason, "unknown season"},
		{economysim.ErrWarmStartPrice, "warm start price out of bounds"},
		{economysim.ErrWarmStartJob, "warm start job is invalid"},
		{economysim.ErrSyntax, "syntax error"},
		{economysim.ErrUnknownKey, "unknown key"},
		{economysim.ErrNotInteger, "not an integer"},
		{economysim.ErrWrongType, "wrong type"},
		{economysim.ErrMissingField, "missing required field"},
		{economysim.ErrUnsupportedSchema, "missing or unsupported schema"},
		{economysim.ErrMergeConflict, "table and scalar conflict"},
		{economysim.ErrTooLarge, "input too large"},
		{economysim.ErrTooManyErrors, "too many errors"},
		{economysim.ErrNilExtension, "nil extension"},
		{economysim.ErrUnknownClearing, "unknown clearing"},
		{economysim.ErrUnknownSpawner, "unknown spawner"},
		{economysim.ErrUnknownStrategy, "unknown strategy"},
		{economysim.ErrInvalidTrade, "invalid trade list"},
		{economysim.ErrNotViable, "configuration is not viable"},
		{economysim.ErrNoUpkeep, "no group has an upkeep"},
		{economysim.ErrSolveLimit, "solver limit reached"},
		{economysim.ErrSolveInconsistent, "solver certificate failed"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestErrorsMutation_QuoteValue verifies which values are written bare and
// which quoted.
//
// Arithmetic: MaxInt64 is 9223372036854775807 (19 bytes) and MinInt64 is
// -9223372036854775808 (20 bytes), the longest canonical integer.
//
// Guard: the int64 width 64 (not 63), the length bound 20 (not <, not 19), and
// the canonical-form check: "007" and "+5" parse but must be quoted, and
// "abc" does not parse and must be quoted.
func TestErrorsMutation_QuoteValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, value, want string
	}{
		{"max int64", "9223372036854775807", "9223372036854775807"},
		{"min int64 (20 bytes)", "-9223372036854775808", "-9223372036854775808"},
		{"above 2^62", "4611686018427387904", "4611686018427387904"},
		{"leading zeros", "007", `"007"`},
		{"plus sign", "+5", `"+5"`},
		{"text", "abc", `"abc"`},
		{"overflow", "9223372036854775808", `"9223372036854775808"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &economysim.ValidationError{Path: []string{"p"}, Rule: economysim.ErrOutOfBounds, Value: tt.value}
			want := `"p": value out of bounds (value ` + tt.want + `)`
			if got := e.Error(); got != want {
				t.Errorf("Error() = %q, want %q", got, want)
			}
		})
	}
}

// TestErrorsMutation_QuoteSegmentCut verifies the cut of a segment at
// MaxIDLength.
//
// Arithmetic: 64 bytes is not cut; 65 bytes is cut to 64 plus "...". A run of
// 100 continuation bytes has no rune start, so the cut walks back to 0 and
// writes an empty string; stopping at 1 would keep one byte.
//
// Guard: `len(s) > MaxIDLength` against >=; the walk-back bound `n > 0` against
// >= (which would index s[-1]) and the start 0 against 1.
func TestErrorsMutation_QuoteSegmentCut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, seg, want string
	}{
		{"exactly 64", strings.Repeat("a", 64), `"` + strings.Repeat("a", 64) + `"`},
		{"65 cut", strings.Repeat("a", 65), `"` + strings.Repeat("a", 64) + `"...`},
		{"no rune start", strings.Repeat("\x80", 100), `""...`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &economysim.ValidationError{Path: []string{tt.seg}}
			want := tt.want + ": invalid configuration"
			if got := e.Error(); got != want {
				t.Errorf("Error() = %q, want %q", got, want)
			}
		})
	}
}
