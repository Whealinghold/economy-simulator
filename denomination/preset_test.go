package denomination_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/denomination"
)

// TestRPG_FormatsTheSpecTable verifies the five rows of spec-currency.md 4.3 at
// the default precision and at precision 2.
//
// Requirement: FR-CUR-017
func TestRPG_FormatsTheSpecTable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		amount int64
		p0, p2 string
	}{
		{0, "0 cp", "0.00 cp"},
		{100, "1 cp", "1.00 cp"},
		{1_460, "1 sp 4 cp", "1 sp 4.60 cp"},
		{12_345, "1 gp 2 sp 3 cp", "1 gp 2 sp 3.45 cp"},
		{-2_920, "-2 sp 9 cp", "-2 sp 9.20 cp"},
	}
	f0 := mustFormatter(t, denomination.RPG(), 0)
	f2 := mustFormatter(t, denomination.RPG(), 2)
	for _, tt := range tests {
		if got := f0.Format(tt.amount); got != tt.p0 {
			t.Errorf("RPG Format(%d) at precision 0 = %q, want %q", tt.amount, got, tt.p0)
		}
		if got := f2.Format(tt.amount); got != tt.p2 {
			t.Errorf("RPG Format(%d) at precision 2 = %q, want %q", tt.amount, got, tt.p2)
		}
	}
}

// TestRPG_LadderIsLargestFirstWithTheSpecSizes verifies the sizes, the symbols
// and that each call returns a fresh slice.
//
// Requirement: FR-CUR-017
// Guard: presets are functions, never package variables (no package-level
// mutable state); a shared slice would let one caller change another's ladder.
func TestRPG_LadderIsLargestFirstWithTheSpecSizes(t *testing.T) {
	t.Parallel()
	l := denomination.RPG()
	want := denomination.Ladder{
		{Name: "Gold", Symbol: "gp", Size: 10_000},
		{Name: "Silver", Symbol: "sp", Size: 1_000},
		{Name: "Copper", Symbol: "cp", Size: 100},
	}
	if !slices.Equal(l, want) {
		t.Errorf("RPG() = %v, want %v", l, want)
	}
	if _, err := denomination.NewFormatter(l); err != nil {
		t.Errorf("NewFormatter(RPG()) error = %v, want nil", err)
	}
	l[0].Size = 1
	if got := denomination.RPG()[0].Size; got != 10_000 {
		t.Errorf("RPG()[0].Size after mutating an earlier result = %d, want 10000", got)
	}
}

// TestSingle_FormatsEveryAmountAsACount verifies that one unit of size 1 prints
// the decimal count.
//
// Requirement: FR-CUR-018, FR-CUR-012
func TestSingle_FormatsEveryAmountAsACount(t *testing.T) {
	t.Parallel()
	f := mustFormatter(t, denomination.Single("credit", "cr"), 0)
	tests := []struct {
		amount int64
		want   string
	}{
		{0, "0 cr"},
		{1, "1 cr"},
		{-7, "-7 cr"},
		{123_456_789, "123456789 cr"},
		{math.MaxInt64, "9223372036854775807 cr"},
		{math.MinInt64, "-9223372036854775808 cr"},
	}
	for _, tt := range tests {
		if got := f.Format(tt.amount); got != tt.want {
			t.Errorf("Single Format(%d) = %q, want %q", tt.amount, got, tt.want)
		}
	}
}

// TestFormat_IrregularLadders verifies that no power of ten is assumed.
//
// Requirement: FR-CUR-019, FR-CUR-009
func TestFormat_IrregularLadders(t *testing.T) {
	t.Parallel()
	lms := denomination.Ladder{{Name: "Large", Symbol: "L", Size: 240}, {Name: "Medium", Symbol: "M", Size: 12}, {Name: "Small", Symbol: "S", Size: 1}}
	abc := denomination.Ladder{{Name: "Alpha", Symbol: "A", Size: 100}, {Name: "Beta", Symbol: "B", Size: 30}, {Name: "Gamma", Symbol: "C", Size: 7}}
	hp := denomination.Ladder{{Name: "Hour", Symbol: "H", Size: 60}, {Name: "Part", Symbol: "P", Size: 1}}
	runFormatCases(t, []formatCase{
		// 500 = 2 * 240 (480), r 20; 1 * 12, r 8; 8 * 1.
		{"lms 500", lms, 0, 500, "2 L 1 M 8 S"},
		// 239 = 0 * 240; 19 * 12 (228), r 11.
		{"lms 239", lms, 0, 239, "19 M 11 S"},
		{"lms 240", lms, 0, 240, "1 L"},
		// 250 = 2 * 100, r 50; 1 * 30, r 20; 2 * 7 (14), r 6.
		{"abc 250", abc, 0, 250, "2 A 1 B 2 C"},
		// p = 2: frac = 6 * 100 / 7 = 85.
		{"abc 250 p2", abc, 2, 250, "2 A 1 B 2.85 C"},
		// 3 725 = 62 * 60 (3 720) + 5.
		{"hp 3725", hp, 0, 3_725, "62 H 5 P"},
	})
}

// TestFormat_NeverPanicsAtTheExtremes is the always-on form of the fuzz
// properties at the corners of the input space.
//
// Requirement: NFR-CUR-001
func TestFormat_NeverPanicsAtTheExtremes(t *testing.T) {
	t.Parallel()
	big := denomination.Ladder{{Name: "Big", Symbol: "bg", Size: denomination.MaxSize}}
	one := denomination.Single("one", "o")
	runFormatCases(t, []formatCase{
		// |MinInt64| = 9 223 372 | 036 854 775 808: 9 223 372 bg, r 36 854 775 808; p = 6:
		// 36 854 775 808 * 10^6 / 10^12 = 36 854 -> 0.036854; count 9223372.
		{"max size min int64 p6", big, 6, math.MinInt64, "-9223372.036854 bg"},
		{"size one p6", one, 6, 12_345, "12345.000000 o"},
		// r = size - 1 at p = 6: 99 * 10^6 / 100 = 990 000.
		{"remainder size minus one", denomination.RPG(), 6, 99, "0.990000 cp"},
	})
}
