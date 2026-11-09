package indicator_test

import (
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/indicator"
)

// TestStat_Operations verifies each statistic of spec-indicators.md 3.6 on rows
// whose answer is worked by hand: the rows {2, 41}, {1, 9}, {1, 4} are four
// households holding 20.5, 20.5, 9 and 4 (sum 54). The mean is rdiv(54, 4) =
// 13.5, an exact half, which goes to the even 14; the quantile at 5 000 bp is
// rank 2 of 4, which is 9, and at 10 000 it is rdiv(41, 2) = 20 (the even
// integer of 20.5); two of the four households hold less than 10, so the share
// below 10 is 5 000.
//
// Requirement: FR-IND-014, FR-IND-024, FR-IND-025
func TestStat_Operations(t *testing.T) {
	t.Parallel()
	rows := []indicator.Household{{Weight: 2, Total: 41}, {Weight: 1, Total: 9}, {Weight: 1, Total: 4}}
	tests := []struct {
		name string
		op   indicator.StatOp
		arg  int64
		want indicator.StatResult
	}{
		{"sum", indicator.StatSum, 0, indicator.StatResult{OK: true, Value: 54, Households: 4}},
		{"mean", indicator.StatMean, 0, indicator.StatResult{OK: true, Value: 14, Households: 4}},
		{"median", indicator.StatQuantile, 5000, indicator.StatResult{OK: true, Value: 9, Households: 4}},
		{"maximum", indicator.StatQuantile, 10000, indicator.StatResult{OK: true, Value: 20, Households: 4}},
		{"share below 10", indicator.StatShareBelow, 10, indicator.StatResult{OK: true, Value: 5000, Households: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := indicator.Stat(rows, tt.op, tt.arg)
			if err != nil || got != tt.want {
				t.Errorf("Stat(%v, %v, %d) = %+v, %v, want %+v", rows, tt.op, tt.arg, got, err, tt.want)
			}
		})
	}
	// two households, one holding all: 5 000 bp
	if got, err := indicator.Stat([]indicator.Household{{Weight: 1, Total: 8}, {Weight: 1}}, indicator.StatGini, 0); err != nil || got != (indicator.StatResult{OK: true, Value: 5000, Households: 2}) {
		t.Errorf("Stat Gini of one holder among two = %+v, %v, want 5 000 bp", got, err)
	}
}

// TestStat_UndefinedAndErrors verifies that an empty set is undefined and not an
// error (the sum of nothing is 0), a weight below 1 and an out-of-range
// quantile are ErrRange, a negative total makes the Gini coefficient
// ErrNegativeValue while the other statistics accept it, an unknown operation is
// ErrRange, and the error names the statistic.
//
// Requirement: FR-IND-012, FR-IND-014, FR-IND-024
func TestStat_UndefinedAndErrors(t *testing.T) {
	t.Parallel()
	pair := []indicator.Household{{Weight: 1, Total: 5}, {Weight: 1, Total: -3}}
	tests := []struct {
		name   string
		rows   []indicator.Household
		op     indicator.StatOp
		arg    int64
		wantOK bool
		err    error
	}{
		{"empty sum", nil, indicator.StatSum, 0, true, nil},
		{"empty mean", nil, indicator.StatMean, 0, false, nil},
		{"empty quantile", nil, indicator.StatQuantile, 5000, false, nil},
		{"empty share", nil, indicator.StatShareBelow, 1, false, nil},
		{"empty gini", nil, indicator.StatGini, 0, false, nil},
		{"gini of nothing held", []indicator.Household{{Weight: 2}}, indicator.StatGini, 0, false, nil},
		{"zero weight", []indicator.Household{{Weight: 0, Total: 1}}, indicator.StatSum, 0, false, indicator.ErrRange},
		{"quantile above range", pair, indicator.StatQuantile, 10001, false, indicator.ErrRange},
		{"negative totals in the mean", pair, indicator.StatMean, 0, true, nil},
		{"negative totals in a quantile", pair, indicator.StatQuantile, 0, true, nil},
		{"negative totals in the gini", pair, indicator.StatGini, 0, false, indicator.ErrNegativeValue},
		{"unknown operation", pair, indicator.StatOp(99), 0, false, indicator.ErrRange},
		{"unset operation", pair, 0, 0, false, indicator.ErrRange},
		{"sum past int64", []indicator.Household{{Weight: 1, Total: 1 << 62}, {Weight: 1, Total: 1 << 62}, {Weight: 1, Total: 1 << 62}}, indicator.StatSum, 0, false, indicator.ErrOverflow},
		{"weights past int64", []indicator.Household{{Weight: 1 << 62, Total: 1}, {Weight: 1 << 62, Total: 1}, {Weight: 1 << 62, Total: 1}}, indicator.StatMean, 0, false, indicator.ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := indicator.Stat(tt.rows, tt.op, tt.arg)
			if got.OK != tt.wantOK || !errors.Is(err, tt.err) {
				t.Fatalf("Stat(%v, %v, %d) = %+v, %v, want OK %v, error %v", tt.rows, tt.op, tt.arg, got, err, tt.wantOK, tt.err)
			}
			if err != nil && !strings.Contains(err.Error(), "Stat") {
				t.Errorf("Stat error = %q, want the preset named in it", err)
			}
			if !got.OK && got.Value != 0 {
				t.Errorf("Stat(%v, %v, %d).Value = %d, want 0 when not OK", tt.rows, tt.op, tt.arg, got.Value)
			}
		})
	}
}

// TestStatOp_String verifies the names of the statistics, which the error of a
// call carries, including a value outside the closed set.
//
// Requirement: FR-IND-024
func TestStatOp_String(t *testing.T) {
	t.Parallel()
	for op, want := range map[indicator.StatOp]string{
		indicator.StatSum: "sum", indicator.StatMean: "mean", indicator.StatQuantile: "quantile",
		indicator.StatShareBelow: "share_below", indicator.StatGini: "gini", indicator.StatOp(42): "stat_op_42",
	} {
		if got := op.String(); got != want {
			t.Errorf("StatOp(%d).String() = %q, want %q", int(op), got, want)
		}
	}
}

// TestStat_RowOrderDoesNotMatter verifies that every statistic gives the same
// answer for any order of the rows, with equal values among them.
//
// Requirement: FR-IND-010
func TestStat_RowOrderDoesNotMatter(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(21, 22))
	ops := []struct {
		op  indicator.StatOp
		arg int64
	}{{indicator.StatSum, 0}, {indicator.StatMean, 0}, {indicator.StatQuantile, 2500}, {indicator.StatQuantile, 9000}, {indicator.StatShareBelow, 4}, {indicator.StatGini, 0}}
	for i := range 200 {
		rows := make([]indicator.Household, 2+r.IntN(8))
		for j := range rows {
			rows[j] = indicator.Household{Weight: 1 + r.Int64N(4), Total: r.Int64N(30)}
		}
		shuffled := slices.Clone(rows)
		r.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		for _, o := range ops {
			a, errA := indicator.Stat(rows, o.op, o.arg)
			b, errB := indicator.Stat(shuffled, o.op, o.arg)
			if a != b || errA != nil || errB != nil {
				t.Fatalf("case %d, %v: rows %v give %+v (%v), shuffled %v give %+v (%v), want equal", i, o.op, rows, a, errA, shuffled, b, errB)
			}
		}
	}
}
