package denomination_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/denomination"
)

// TestNewFormatter_RejectsInvalidLadders verifies that every ladder fault is
// rejected at construction with its own sentinel, each case breaking only that
// rule.
//
// Requirement: FR-CUR-008, FR-CUR-009, FR-CUR-010
func TestNewFormatter_RejectsInvalidLadders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		ladder denomination.Ladder
		want   error
	}{
		{"nil ladder", nil, denomination.ErrNoUnits},
		{"empty ladder", denomination.Ladder{}, denomination.ErrNoUnits},
		{"empty name", denomination.Ladder{{Name: "", Symbol: "a", Size: 1}}, denomination.ErrUnitName},
		{"empty symbol", denomination.Ladder{{Name: "A", Symbol: "", Size: 1}}, denomination.ErrUnitName},
		{"duplicate name", denomination.Ladder{{Name: "A", Symbol: "a", Size: 10}, {Name: "A", Symbol: "b", Size: 1}}, denomination.ErrUnitDuplicate},
		{"duplicate symbol", denomination.Ladder{{Name: "A", Symbol: "a", Size: 10}, {Name: "B", Symbol: "a", Size: 1}}, denomination.ErrUnitDuplicate},
		{"size zero", denomination.Ladder{{Name: "A", Symbol: "a", Size: 0}}, denomination.ErrUnitSize},
		{"size negative", denomination.Ladder{{Name: "A", Symbol: "a", Size: -1}}, denomination.ErrUnitSize},
		{"size above max", denomination.Ladder{{Name: "A", Symbol: "a", Size: denomination.MaxSize + 1}}, denomination.ErrUnitSize},
		{"equal sizes", denomination.Ladder{{Name: "A", Symbol: "a", Size: 10}, {Name: "B", Symbol: "b", Size: 10}}, denomination.ErrUnitOrder},
		{"increasing sizes", denomination.Ladder{{Name: "A", Symbol: "a", Size: 10}, {Name: "B", Symbol: "b", Size: 100}}, denomination.ErrUnitOrder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, err := denomination.NewFormatter(tt.ladder)
			if !errors.Is(err, tt.want) {
				t.Errorf("NewFormatter(%v) error = %v, want %v", tt.ladder, err, tt.want)
			}
			if f != nil {
				t.Errorf("NewFormatter(%v) formatter = %v, want nil", tt.ladder, f)
			}
		})
	}
}

// TestNewFormatter_AcceptsBoundaryLadders verifies the smallest and largest
// valid sizes and a ladder whose sizes do not divide each other.
//
// Requirement: FR-CUR-008, FR-CUR-009
func TestNewFormatter_AcceptsBoundaryLadders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		ladder denomination.Ladder
	}{
		{"size 1 alone", denomination.Ladder{{Name: "A", Symbol: "a", Size: 1}}},
		{"max size alone", denomination.Ladder{{Name: "A", Symbol: "a", Size: denomination.MaxSize}}},
		{"max size then 1", denomination.Ladder{{Name: "A", Symbol: "a", Size: denomination.MaxSize}, {Name: "B", Symbol: "b", Size: 1}}},
		{"sizes that do not divide", denomination.Ladder{{Name: "A", Symbol: "a", Size: 100}, {Name: "B", Symbol: "b", Size: 30}, {Name: "C", Symbol: "c", Size: 7}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, err := denomination.NewFormatter(tt.ladder)
			if err != nil || f == nil {
				t.Errorf("NewFormatter(%v) = %v, %v, want a formatter", tt.ladder, f, err)
			}
		})
	}
}

// TestNewFormatter_RejectsPrecisionOutsideRange verifies that a precision
// outside 0 to MaxPrecision is ErrPrecision and gives no formatter.
//
// Requirement: FR-CUR-010
func TestNewFormatter_RejectsPrecisionOutsideRange(t *testing.T) {
	t.Parallel()
	for _, p := range []int{-1, denomination.MaxPrecision + 1} {
		f, err := denomination.NewFormatter(denomination.RPG(), denomination.WithPrecision(p))
		if !errors.Is(err, denomination.ErrPrecision) || f != nil {
			t.Errorf("NewFormatter(RPG, WithPrecision(%d)) = %v, %v, want nil, ErrPrecision", p, f, err)
		}
	}
	for p := 0; p <= denomination.MaxPrecision; p++ {
		f, err := denomination.NewFormatter(denomination.RPG(), denomination.WithPrecision(p))
		if err != nil || f == nil {
			t.Errorf("NewFormatter(RPG, WithPrecision(%d)) = %v, %v, want a formatter", p, f, err)
		}
	}
}

// TestNewFormatter_ReportsTheFirstFaultWithItsIndex verifies the fixed
// validation order and that the error names the index of the offending unit.
//
// Guard: callers read the index and branch on the sentinel; a reordered check
// would change which sentinel they see. The index is part of the contract, so
// this is the one deliberate message-text assertion.
func TestNewFormatter_ReportsTheFirstFaultWithItsIndex(t *testing.T) {
	t.Parallel()
	// Unit 1 has a bad size and unit 2 is out of order: the size of unit 1 is first.
	l := denomination.Ladder{
		{Name: "A", Symbol: "a", Size: 100},
		{Name: "B", Symbol: "b", Size: 0},
		{Name: "C", Symbol: "c", Size: 200},
	}
	_, err := denomination.NewFormatter(l)
	if !errors.Is(err, denomination.ErrUnitSize) {
		t.Fatalf("NewFormatter(%v) error = %v, want ErrUnitSize", l, err)
	}
	if !strings.Contains(err.Error(), strconv.Itoa(1)) {
		t.Errorf("NewFormatter(%v) error = %q, want it to contain the index 1", l, err)
	}
	// A name fault on unit 0 comes before the size fault of the same unit.
	l = denomination.Ladder{{Name: "", Symbol: "a", Size: 0}}
	if _, err = denomination.NewFormatter(l); !errors.Is(err, denomination.ErrUnitName) {
		t.Errorf("NewFormatter(%v) error = %v, want ErrUnitName", l, err)
	}
}
