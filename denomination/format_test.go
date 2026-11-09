package denomination_test

import (
	"math"
	"sync"
	"testing"

	"github.com/Whealinghold/economy-sim/denomination"
)

func mustFormatter(t *testing.T, l denomination.Ladder, p int) *denomination.Formatter {
	t.Helper()
	f, err := denomination.NewFormatter(l, denomination.WithPrecision(p))
	if err != nil {
		t.Fatalf("NewFormatter(%v, WithPrecision(%d)) error = %v", l, p, err)
	}
	return f
}

type formatCase struct {
	name   string
	ladder denomination.Ladder
	p      int
	amount int64
	want   string
}

func runFormatCases(t *testing.T, tests []formatCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := mustFormatter(t, tt.ladder, tt.p)
			if got := f.Format(tt.amount); got != tt.want {
				t.Errorf("Format(%d) at precision %d = %q, want %q", tt.amount, tt.p, got, tt.want)
			}
		})
	}
}

// TestFormat_FillsFromLargestToSmallest verifies count = r / size, r = r % size
// from the largest unit down, with zero counts omitted and the largest unbounded.
//
// Requirement: FR-CUR-013, FR-CUR-014
func TestFormat_FillsFromLargestToSmallest(t *testing.T) {
	t.Parallel()
	runFormatCases(t, []formatCase{
		// 12 345 = 1 gp (10 000), r 2 345; 2 sp (2 000), r 345; 3 cp (300), r 45 (truncated).
		{"all three units", denomination.RPG(), 0, 12_345, "1 gp 2 sp 3 cp"},
		{"one unit only", denomination.RPG(), 0, 10_000, "1 gp"},
		{"zero middle omitted", denomination.RPG(), 0, 10_100, "1 gp 1 cp"},
		// 10 000 000 / 10 000 = 1000 gp: the largest is unbounded.
		{"largest is unbounded", denomination.RPG(), 0, 10_000_000, "1000 gp"},
	})
}

// TestFormat_ZeroAmount verifies that a zero amount prints 0 of the smallest unit.
//
// Requirement: FR-CUR-014, FR-CUR-016
func TestFormat_ZeroAmount(t *testing.T) {
	t.Parallel()
	runFormatCases(t, []formatCase{
		{"precision 0", denomination.RPG(), 0, 0, "0 cp"},
		{"precision 2", denomination.RPG(), 2, 0, "0.00 cp"},
		{"single unit", denomination.Single("credit", "cr"), 0, 0, "0 cr"},
	})
}

// TestFormat_Precision verifies frac = r * 10^p / size, truncated and
// zero-padded to p digits.
//
// Requirement: FR-CUR-015
func TestFormat_Precision(t *testing.T) {
	t.Parallel()
	big := denomination.Ladder{{Name: "Big", Symbol: "bg", Size: denomination.MaxSize}}
	runFormatCases(t, []formatCase{
		// 1 460 = 1 sp (1 000), r 460; 4 cp (400), r 60; frac = 60 * 10^p / 100.
		{"p0", denomination.RPG(), 0, 1_460, "1 sp 4 cp"},
		{"p1", denomination.RPG(), 1, 1_460, "1 sp 4.6 cp"},
		{"p2", denomination.RPG(), 2, 1_460, "1 sp 4.60 cp"},
		{"p6", denomination.RPG(), 6, 1_460, "1 sp 4.600000 cp"},
		// frac = 5 * 100 / 100 = 5, padded to 05.
		{"zero padded", denomination.RPG(), 2, 5, "0.05 cp"},
		// 1 499: r after cp is 99, truncated at p = 0.
		{"truncated", denomination.RPG(), 0, 1_499, "1 sp 4 cp"},
		// (10^12 - 1) * 10^6 = 999 999 999 999 000 000 < 2^64; / 10^12 = 999 999.
		{"largest product", big, 6, denomination.MaxSize - 1, "0.999999 bg"},
		// 10 000 at p = 2: smallest count and frac are zero, so cp is omitted.
		{"zero smallest omitted", denomination.RPG(), 2, 10_000, "1 gp"},
	})
}

// TestFormat_SignAndNeverNegativeZero verifies the sign rule: written only when
// a printed count or fraction is non-zero.
//
// Requirement: FR-CUR-012, FR-CUR-016
func TestFormat_SignAndNeverNegativeZero(t *testing.T) {
	t.Parallel()
	size100 := denomination.Single("x", "xp")
	size100[0].Size = 100
	runFormatCases(t, []formatCase{
		{"negative", denomination.RPG(), 0, -2_920, "-2 sp 9 cp"},
		{"negative exact", denomination.RPG(), 0, -100, "-1 cp"},
		// Only the largest unit is non-zero: the sign must still be written.
		{"negative largest only", denomination.RPG(), 0, -10_000, "-1 gp"},
		{"negative largest only p2", denomination.RPG(), 2, -10_000, "-1 gp"},
		{"minus one p0", denomination.RPG(), 0, -1, "0 cp"},
		{"minus 99 p0", denomination.RPG(), 0, -99, "0 cp"},
		{"minus one p2", denomination.RPG(), 2, -1, "-0.01 cp"},
		// D671: -5 on size 100 at p = 1: frac = 5 * 10 / 100 = 0, nothing non-zero.
		{"zero fraction no sign", size100, 1, -5, "0.0 xp"},
	})
}

// TestFormat_MinAndMaxInt64 verifies the extremes format without overflow.
//
// Requirement: FR-CUR-012, NFR-CUR-001
func TestFormat_MinAndMaxInt64(t *testing.T) {
	t.Parallel()
	// |MinInt64| = 9 223 372 036 854 775 808 = 922 337 203 685 477 gp (9 223 372 036 854 770 000)
	// + 5 808; 5 sp (5 000), r 808; 8 cp (800), r 8; p = 2: 8 * 100 / 100 = 8 -> 8.08 cp.
	runFormatCases(t, []formatCase{
		{"min p0", denomination.RPG(), 0, math.MinInt64, "-922337203685477 gp 5 sp 8 cp"},
		{"min p2", denomination.RPG(), 2, math.MinInt64, "-922337203685477 gp 5 sp 8.08 cp"},
		// MaxInt64 = ...775 807: remainder 5 807: 5 sp, r 807; 8 cp, r 7; p = 2: 7 -> 8.07 cp.
		{"max p0", denomination.RPG(), 0, math.MaxInt64, "922337203685477 gp 5 sp 8 cp"},
		{"max p2", denomination.RPG(), 2, math.MaxInt64, "922337203685477 gp 5 sp 8.07 cp"},
	})
}

// TestFormat_IsPureAndIndependentOfTheCallersSlice verifies that Format returns
// no error, is repeatable, and does not share data with the caller's slice.
//
// Requirement: FR-CUR-010, FR-CUR-011
func TestFormat_IsPureAndIndependentOfTheCallersSlice(t *testing.T) {
	t.Parallel()
	var _ func(*denomination.Formatter, int64) string = (*denomination.Formatter).Format

	l := denomination.RPG()
	f := mustFormatter(t, l, 0)
	first := f.Format(12_345)
	if again := f.Format(12_345); again != first {
		t.Errorf("Format(12345) twice = %q then %q, want equal", first, again)
	}
	l[0].Symbol = "zz"
	l[2].Size = 1
	if got := f.Format(12_345); got != first {
		t.Errorf("Format(12345) after mutating the ladder = %q, want %q", got, first)
	}
	g := mustFormatter(t, denomination.RPG(), 0)
	if got := g.Format(12_345); got != first {
		t.Errorf("Format(12345) of a second formatter = %q, want %q", got, first)
	}
}

// TestFormat_ConcurrentUse verifies that one formatter serves parallel callers.
//
// Guard: the doc comment promises concurrent safety; run under -race.
func TestFormat_ConcurrentUse(t *testing.T) {
	t.Parallel()
	f := mustFormatter(t, denomination.RPG(), 2)
	var wg sync.WaitGroup
	for i := range 8 {
		amount := int64(i)*12_345 + 7
		want := f.Format(amount)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				if got := f.Format(amount); got != want {
					t.Errorf("Format(%d) = %q, want %q", amount, got, want)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestNewFormatter_IgnoresNilOption(t *testing.T) {
	t.Parallel()
	f, err := denomination.NewFormatter(denomination.Single("Unit", "u"), nil)
	if err != nil {
		t.Fatalf("NewFormatter with a nil Option error = %v", err)
	}
	if got := f.Format(5); got != "5 u" {
		t.Errorf("Format(5) = %q, want %q", got, "5 u")
	}
}
