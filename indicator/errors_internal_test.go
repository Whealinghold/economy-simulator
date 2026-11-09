package indicator

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/measure"
)

// TestErrors_Sentinels verifies that each of the six sentinels of
// spec-indicators.md 3.5 is returned wrapped with the preset's name and the
// field, that errors.Is still finds it, and that the sentinels are distinct.
//
// Requirement: FR-IND-006, FR-IND-029
func TestErrors_Sentinels(t *testing.T) {
	t.Parallel()
	sentinels := []error{ErrConfigMismatch, ErrWindowOrder, ErrUnknownClass, ErrRange, ErrNegativeValue, ErrOverflow}
	for i, s := range sentinels {
		t.Run(s.Error(), func(t *testing.T) {
			t.Parallel()
			err := wrap("GDP", "open", s)
			if !errors.Is(err, s) {
				t.Errorf("errors.Is(wrap(GDP, open, %v), sentinel) = false, want true", s)
			}
			if !strings.Contains(err.Error(), "GDP") || !strings.Contains(err.Error(), "open") {
				t.Errorf("wrap(GDP, open, %v) = %q, want the preset and the field in it", s, err)
			}
			for j, other := range sentinels {
				if i != j && errors.Is(err, other) {
					t.Errorf("wrapped %v also matches %v, want distinct sentinels", s, other)
				}
			}
		})
	}
	if wrap("GDP", "open", nil) != nil {
		t.Errorf("wrap(GDP, open, nil) != nil, want nil")
	}
}

// TestFromMeasure_MapsTheSentinels verifies the mapping from the errors of
// internal/measure to those of this package.
//
// Requirement: FR-IND-009, FR-IND-014
func TestFromMeasure_MapsTheSentinels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   error
		want error
	}{
		{nil, nil},
		{measure.ErrNegative, ErrNegativeValue},
		{measure.ErrRange, ErrRange},
		{measure.ErrOverflow, ErrOverflow},
	}
	for _, tt := range tests {
		if got := fromMeasure(tt.in); !errors.Is(got, tt.want) || (tt.want == nil && got != nil) {
			t.Errorf("fromMeasure(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// TestNet_SumsAreCheckedNeverWrapped verifies that net returns the exact sum and
// less of int64 terms, and ErrOverflow where an exact result does not fit,
// including the negation of the smallest int64.
//
// Requirement: FR-IND-009
func TestNet_SumsAreCheckedNeverWrapped(t *testing.T) {
	t.Parallel()
	const maxInt, minInt = int64(1<<63 - 1), int64(-1 << 63)
	tests := []struct {
		name  string
		plus  []int64
		minus []int64
		want  int64
		err   error
	}{
		{"empty sum is 0", nil, nil, 0, nil},
		{"plus less minus", []int64{7, 5}, []int64{2}, 10, nil},
		{"two maxima past int64", []int64{maxInt, maxInt}, nil, 0, ErrOverflow},
		{"wide intermediate that fits", []int64{maxInt, maxInt}, []int64{maxInt}, maxInt, nil},
		{"negated minimum past int64", nil, []int64{minInt}, 0, ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := net("Test", "field", tt.plus, tt.minus...)
			if got != tt.want || !errors.Is(err, tt.err) {
				t.Errorf("net(%v, %v) = %d, %v, want %d, %v", tt.plus, tt.minus, got, err, tt.want, tt.err)
			}
		})
	}
}
