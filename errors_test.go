package economysim_test

import (
	"errors"
	"fmt"
	"slices"
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
		economysim.ErrNilExtension, economysim.ErrUnknownClearing, economysim.ErrUnknownSpawner, economysim.ErrUnknownStrategy, economysim.ErrInvalidTrade,
		economysim.ErrWorldFailed, economysim.ErrInStep, economysim.ErrAuditFailed,
		economysim.ErrNotViable, economysim.ErrNoUpkeep, economysim.ErrSolveLimit, economysim.ErrSolveInconsistent,
		economysim.ErrCheckpointVersion, economysim.ErrCheckpointCorrupt, economysim.ErrCheckpointConfig,
		economysim.ErrCheckpointExtension, economysim.ErrOptionContext,
		economysim.ErrSelfEdge, economysim.ErrUnknownEdge, economysim.ErrUnknownAgent, economysim.ErrClosedAccount,
		economysim.ErrNotRelocatable, economysim.ErrSameRegion, economysim.ErrOpenReservation, economysim.ErrRegionBound,
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

// TestViabilityError_Matching verifies that errors.Is(err, ErrNotViable) matches
// a *ViabilityError alone, joined and wrapped, that it matches no other sentinel,
// and that errors.As finds the rule, path, value and bound.
//
// Requirement: FR-VIA-011, FR-CFG-041
func TestViabilityError_Matching(t *testing.T) {
	t.Parallel()
	ve := &economysim.ViabilityError{
		Rule: "labour_short", Path: []string{"regions"}, Value: "5000", Bound: "10000",
	}
	joined := fmt.Errorf("validate: %w", errors.Join(errors.New("other"), ve))
	if !errors.Is(joined, economysim.ErrNotViable) {
		t.Errorf("errors.Is(%v, ErrNotViable) = false, want true", joined)
	}
	for _, s := range []error{economysim.ErrOutOfBounds, economysim.ErrNoUpkeep, economysim.ErrSolveLimit, economysim.ErrSolveInconsistent} {
		if errors.Is(joined, s) {
			t.Errorf("errors.Is(%v, %v) = true, want false", joined, s)
		}
	}
	var got *economysim.ViabilityError
	if !errors.As(joined, &got) || got.Rule != "labour_short" || got.Value != "5000" || got.Bound != "10000" || !slices.Equal(got.Path, []string{"regions"}) {
		t.Errorf("errors.As(%v) = %+v, want the labour_short error with value 5000 and bound 10000", joined, got)
	}
	var verr *economysim.ValidationError
	if errors.As(ve, &verr) {
		t.Errorf("errors.As(ViabilityError, *ValidationError) = true, want false: it is its own type")
	}
}

// TestViabilityError_Text verifies the text of a viability error: the path, value
// and bound are cut at 64 bytes and quoted as a ValidationError's are, so a
// hostile ID cannot flood a log or hold a control character, and a rule with no
// value or bound writes neither.
//
// Requirement: FR-CFG-100
func TestViabilityError_Text(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 200)
	tests := []struct {
		name string
		err  *economysim.ViabilityError
		want string
	}{
		{
			"value and bound",
			&economysim.ViabilityError{Rule: "labour_short", Path: []string{"regions"}, Value: "5000", Bound: "10000"},
			`"regions": configuration is not viable: "labour_short" (value 5000, bound 10000)`,
		},
		{
			"value only",
			&economysim.ViabilityError{Rule: "role_cannot_work", Path: []string{"roles", "idle"}, Value: "north"},
			`"roles"."idle": configuration is not viable: "role_cannot_work" (value "north")`,
		},
		{
			"no value",
			&economysim.ViabilityError{Rule: "storage_gap", Path: []string{"regions"}},
			`"regions": configuration is not viable: "storage_gap"`,
		},
		{
			"long and hostile segments",
			&economysim.ViabilityError{Rule: "storage_gap", Path: []string{"regions", long, "a\x00b\nc"}, Value: "\x1b[31m"},
			`"regions"."` + strings.Repeat("a", 64) + `"...."a\x00b\nc": configuration is not viable: "storage_gap" (value "\x1b[31m")`,
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

// TestSolveError verifies that a SolveError reports the pivots, rows and columns
// it carries and matches ErrSolveLimit.
//
// Requirement: FR-VIA-010, FR-VIA-017
func TestSolveError(t *testing.T) {
	t.Parallel()
	var err error = &economysim.SolveError{Pivots: 7, Rows: 3, Columns: 9}
	if !errors.Is(fmt.Errorf("solve: %w", err), economysim.ErrSolveLimit) {
		t.Errorf("errors.Is(SolveError, ErrSolveLimit) = false, want true")
	}
	want := "solver limit reached: 7 pivots, 3 rows, 9 columns"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	var se *economysim.SolveError
	if !errors.As(err, &se) || se.Rows != 3 || se.Columns != 9 || se.Pivots != 7 {
		t.Errorf("errors.As = %+v, want the counts back", se)
	}
}

// TestRegionBoundError_MatchesAndNames verifies that a *RegionBoundError matches
// ErrRegionBound with errors.Is, alone and wrapped, matches no other sentinel,
// is found by errors.As with its obligation, and says the obligation in its text.
//
// Requirement: FR-REG-047, FR-REG-060
func TestRegionBoundError_MatchesAndNames(t *testing.T) {
	t.Parallel()
	err := error(&economysim.RegionBoundError{Obligation: "open_shipment"})
	wrapped := fmt.Errorf("relocate: %w", err)
	for _, e := range []error{err, wrapped} {
		if !errors.Is(e, economysim.ErrRegionBound) {
			t.Errorf("errors.Is(%v, ErrRegionBound) = false, want true", e)
		}
		if errors.Is(e, economysim.ErrSameRegion) || errors.Is(e, economysim.ErrOpenReservation) {
			t.Errorf("errors.Is(%v, another refusal) = true, want false", e)
		}
	}
	var rb *economysim.RegionBoundError
	if !errors.As(wrapped, &rb) || rb.Obligation != "open_shipment" {
		t.Errorf("errors.As(%v, *RegionBoundError) = %v, want the obligation open_shipment", wrapped, rb)
	}
	if got, want := err.Error(), economysim.ErrRegionBound.Error()+": open_shipment"; got != want {
		t.Errorf("RegionBoundError.Error() = %q, want %q", got, want)
	}
}
