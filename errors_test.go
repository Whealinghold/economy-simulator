package economysim_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestValidationError_Matching verifies that errors.Is matches the sentinel of
// the rule through the typed error, also when joined and wrapped, and that
// errors.As finds the typed error with its fields.
//
// Requirement: FR-CFG-009
func TestValidationError_Matching(t *testing.T) {
	t.Parallel()
	ve := &economysim.ValidationError{
		Path:  []string{"roles", "worker_a", "target_ratio"},
		Rule:  economysim.ErrOutOfBounds,
		Value: "5",
		Bound: "10000",
	}
	joined := fmt.Errorf("validate: %w", errors.Join(errors.New("other"), ve))
	if !errors.Is(joined, economysim.ErrOutOfBounds) {
		t.Errorf("errors.Is(%v, ErrOutOfBounds) = false, want true", joined)
	}
	if errors.Is(joined, economysim.ErrOverflow) {
		t.Errorf("errors.Is(%v, ErrOverflow) = true, want false", joined)
	}
	var got *economysim.ValidationError
	if !errors.As(joined, &got) || got.Value != "5" || got.Bound != "10000" || len(got.Path) != 3 {
		t.Errorf("errors.As(%v) = %+v, want the error with value 5, bound 10000 and 3 path segments", joined, got)
	}
}

// TestValidationError_Text verifies the text of an error: every path segment
// and string value is quoted and cut at 64 bytes on a rune boundary with three
// dots outside the quotes, an integer is a decimal, at most 16 segments are
// written, and no control character gets through.
//
// Requirement: FR-CFG-100
func TestValidationError_Text(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 200)
	// 63 one-byte letters then a three-byte rune: the rune starts at byte 63 and
	// ends at byte 66, so a cut at byte 64 must drop all of it.
	straddle := strings.Repeat("a", 63) + "€" + "tail"
	tests := []struct {
		name string
		err  *economysim.ValidationError
		want string
	}{
		{
			"plain path and integers",
			&economysim.ValidationError{Path: []string{"roles", "worker_a", "target_ratio"}, Rule: economysim.ErrOutOfBounds, Value: "5", Bound: "10000"},
			`"roles"."worker_a"."target_ratio": value out of bounds (value 5, bound 10000)`,
		},
		{
			"string value is quoted",
			&economysim.ValidationError{Path: []string{"roles"}, Rule: economysim.ErrInvalidID, Value: "Bad ID"},
			`"roles": invalid ID (value "Bad ID")`,
		},
		{
			// 70 zeros parse as 0 but are not canonical, so they are quoted and cut at 64 bytes.
			"long run of zeros is cut",
			&economysim.ValidationError{Path: []string{"roles"}, Rule: economysim.ErrInvalidID, Value: strings.Repeat("0", 70)},
			`"roles": invalid ID (value "` + strings.Repeat("0", 64) + `"...)`,
		},
		{
			"numeric-looking string is quoted",
			&economysim.ValidationError{Path: []string{"roles"}, Rule: economysim.ErrInvalidID, Value: "+12"},
			`"roles": invalid ID (value "+12")`,
		},
		{
			"long segment is cut",
			&economysim.ValidationError{Path: []string{"roles", long}, Rule: economysim.ErrInvalidID},
			`"roles"."` + strings.Repeat("a", 64) + `"...: invalid ID`,
		},
		{
			"cut on a rune boundary",
			&economysim.ValidationError{Path: []string{straddle}, Rule: economysim.ErrInvalidID},
			`"` + strings.Repeat("a", 63) + `"...: invalid ID`,
		},
		{
			"control characters are escaped",
			&economysim.ValidationError{Path: []string{"a\x00b\nc\x1b[31m"}, Rule: economysim.ErrInvalidID},
			`"a\x00b\nc\x1b[31m": invalid ID`,
		},
		{
			"no rule",
			&economysim.ValidationError{Path: []string{"world"}},
			`"world": invalid configuration`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() of %+v = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

// TestValidationError_TextIsBounded verifies that a deep path and a huge value
// give a message of bounded length with no control character, whatever the
// fields hold: the fields keep the truth and only Error cuts.
//
// Requirement: FR-CFG-100
func TestValidationError_TextIsBounded(t *testing.T) {
	t.Parallel()
	deep := make([]string, 100)
	for i := range deep {
		deep[i] = strings.Repeat("\x07é", 1000)
	}
	ve := &economysim.ValidationError{
		Path: deep, Rule: economysim.ErrTooMany,
		Value: strings.Repeat("\x1b", 1<<20), Bound: strings.Repeat("é", 1<<20),
	}
	got := ve.Error()
	// 16 segments of at most 64 bytes each quoted (a byte takes at most 4
	// characters of escape), the dots, a value and a bound of the same form.
	if limit := 16*(4*64+5) + 16 + 2*(4*64+5) + 100; len(got) > limit {
		t.Errorf("len(Error()) = %d, want at most %d", len(got), limit)
	}
	if !strings.Contains(got, `"...`) {
		t.Errorf("Error() = %.80q..., want a cut segment marked with three dots after its quotes", got)
	}
	if !utf8.ValidString(got) {
		t.Errorf("Error() is not valid UTF-8")
	}
	for _, r := range got {
		if unicode.IsControl(r) {
			t.Errorf("Error() holds the control character %U, want none", r)
			break
		}
	}
	if len(ve.Path) != 100 || len(ve.Value) != 1<<20 {
		t.Errorf("fields after Error() = %d segments and %d value bytes, want them unchanged", len(ve.Path), len(ve.Value))
	}
}

// TestValidationError_PathCap verifies the exact cut of a long path: the first
// 16 segments are written, each cut at 64 bytes and quoted, and a longer path
// ends in three dots straight after the 16th segment; a path of exactly 16
// segments has no marker, and a segment past the 16th never appears.
//
// Requirement: FR-CFG-100
func TestValidationError_PathCap(t *testing.T) {
	t.Parallel()
	// A segment of control characters, 100 bytes: cut at 64 bytes, each escaped
	// as \a by strconv.Quote, and marked with three dots outside the quotes.
	oversized := strings.Repeat("\a", 100)
	wantOversized := `"` + strings.Repeat(`\a`, 64) + `"...`
	build := func(n int) (path []string, quoted []string) {
		for i := 0; i < n; i++ {
			s := fmt.Sprintf("s%02d", i)
			q := `"` + s + `"`
			if i == 5 || i == 16 || i == 30 {
				s, q = oversized, wantOversized
			}
			path, quoted = append(path, s), append(quoted, q)
		}
		return path, quoted
	}
	tests := []struct {
		name     string
		segments int
		marker   string
	}{
		{"one under the cap", 15, ""},
		{"exactly the cap", 16, ""},
		{"one over the cap", 17, "..."},
		{"forty segments", 40, "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path, quoted := build(tt.segments)
			want := strings.Join(quoted[:min(tt.segments, 16)], ".") + tt.marker + ": too many entities"
			got := (&economysim.ValidationError{Path: path, Rule: economysim.ErrTooMany}).Error()
			if got != want {
				t.Errorf("Error() of a path of %d segments = %q, want %q", tt.segments, got, want)
			}
		})
	}
}

// TestSentinels_AreDistinctAndMatch verifies that every structural sentinel of
// the root package has a message of its own and is matched by errors.Is
// through a ValidationError that carries it, and no other sentinel is.
//
// Guard: spec-config.md 7.3 names one sentinel per structural rule; two with one
// message, or one that does not round-trip, would make a rule unmatchable.
func TestSentinels_AreDistinctAndMatch(t *testing.T) {
	t.Parallel()
	sentinels := []error{
		economysim.ErrInvalidID, economysim.ErrDuplicateID, economysim.ErrUnknownCommodity,
		economysim.ErrUnknownRole, economysim.ErrUnknownRecipe, economysim.ErrUnknownRegion,
		economysim.ErrRecipeNoOutput, economysim.ErrOrphanCommodity, economysim.ErrUnproducible,
		economysim.ErrOutOfBounds, economysim.ErrOverflow, economysim.ErrTooMany,
		economysim.ErrNoRegions, economysim.ErrModuleDependency,
		economysim.ErrNilExtension, economysim.ErrUnknownClearing, economysim.ErrInvalidTrade,
	}
	seen := map[string]bool{}
	for _, s := range sentinels {
		if seen[s.Error()] {
			t.Errorf("sentinel message %q is used twice, want a distinct message", s.Error())
		}
		seen[s.Error()] = true
		err := error(&economysim.ValidationError{Path: []string{"x"}, Rule: s})
		for _, other := range sentinels {
			if got := errors.Is(err, other); got != (other == s) {
				t.Errorf("errors.Is(ValidationError{%v}, %v) = %v, want %v", s, other, got, other == s)
			}
		}
	}
}
