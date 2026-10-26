package steady_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// TestFixedTerm verifies that the fixed term of a goods row is the exact sum of
// the taste rows, split among items by weight and kept as a fraction in lowest
// terms, never rounded.
//
// Requirement: FR-VIA-055
func TestFixedTerm(t *testing.T) {
	t.Parallel()
	one := []steady.WantWeight{{Commodity: 0, Weight: 1}}
	two := []steady.WantWeight{{Commodity: 0, Weight: 1}, {Commodity: 1, Weight: 3}}
	tests := []struct {
		name string
		rows []steady.WantRow
		want []steady.Ratio
	}{
		{"no rows is zero", nil, []steady.Ratio{{0, 1}, {0, 1}}},
		// 10 households * 10 000 * 10 000 * 6 / (10^8 * 3) = 20 per tick.
		{"one item", []steady.WantRow{{10, 10000, 10000, 6, 3, one}}, []steady.Ratio{{20, 1}, {0, 1}}},
		// 1 household, scales 10 000, quantity 1, period 1: 1 split 1:3 gives 1/4 and 3/4.
		{"weights split", []steady.WantRow{{1, 10000, 10000, 1, 1, two}}, []steady.Ratio{{1, 4}, {3, 4}}},
		// The second row has half the taste, so it adds 1/8 and 3/8: 1/4+1/8 = 3/8 and 3/4+3/8 = 9/8.
		{"rows add", []steady.WantRow{{1, 10000, 10000, 1, 1, two}, {1, 5000, 10000, 1, 1, two}},
			[]steady.Ratio{{3, 8}, {9, 8}}},
		{"negative factors skipped", []steady.WantRow{
			{-1, 10000, 10000, 1, 1, one}, {1, -10000, 10000, 1, 1, one}, {1, 10000, -10000, 1, 1, one}, {1, 10000, 10000, -1, 1, one}},
			[]steady.Ratio{{0, 1}, {0, 1}}},
		// A negative weight counts as 0: the row is 1 split among the first item alone.
		{"negative weight ignored", []steady.WantRow{{1, 10000, 10000, 1, 1, []steady.WantWeight{{0, 1}, {1, -1}}}}, []steady.Ratio{{1, 1}, {0, 1}}},
		{"zero weights and bad period skipped", []steady.WantRow{
			{1, 10000, 10000, 1, 1, []steady.WantWeight{{0, 0}}}, {1, 10000, 10000, 1, 0, one}},
			[]steady.Ratio{{0, 1}, {0, 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := steady.FixedTerm(2, tt.rows)
			if err != nil {
				t.Fatalf("FixedTerm(%v) error = %v, want nil", tt.rows, err)
			}
			for c := range tt.want {
				if got[c] != tt.want[c] {
					t.Errorf("FixedTerm(%v)[%d] = %v, want %v", tt.rows, c, got[c], tt.want[c])
				}
			}
		})
	}
}

// TestFixedTerm_Overflow verifies that a term that does not fit two int64 is
// ErrOverflow and not a wrapped value.
//
// Requirement: FR-VIA-055
func TestFixedTerm_Overflow(t *testing.T) {
	t.Parallel()
	big := steady.WantRow{Households: 1 << 62, Taste: 1 << 30, Region: 10000, Quantity: 1000, Period: 1,
		Items: []steady.WantWeight{{Commodity: 0, Weight: 1}}}
	if _, err := steady.FixedTerm(1, []steady.WantRow{big}); !errors.Is(err, steady.ErrOverflow) {
		t.Errorf("FixedTerm(huge) error = %v, want ErrOverflow", err)
	}
}

// TestFixedTerm_ItemOutOfRange verifies that an item whose commodity is outside
// the commodities is ErrBadProgramme and not an index panic.
//
// Guard: the rows come from a caller; without the check a bad index would panic
// the viability check.
func TestFixedTerm_ItemOutOfRange(t *testing.T) {
	t.Parallel()
	for _, c := range []int{-1, 2} {
		row := steady.WantRow{Households: 1, Taste: 10000, Region: 10000, Quantity: 1, Period: 1,
			Items: []steady.WantWeight{{Commodity: c, Weight: 1}}}
		if _, err := steady.FixedTerm(2, []steady.WantRow{row}); !errors.Is(err, steady.ErrBadProgramme) {
			t.Errorf("FixedTerm(item commodity %d of 2) error = %v, want ErrBadProgramme", c, err)
		}
	}
}
