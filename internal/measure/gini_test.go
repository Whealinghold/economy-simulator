package measure_test

import (
	"errors"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/measure"
)

// randRows draws n rows of weight 1 to maxW and a total from 0 to maxT.
func randRows(r *rand.Rand, n int, maxW, maxT int64) []measure.Row {
	rows := make([]measure.Row, n)
	for i := range rows {
		rows[i] = measure.Row{Weight: 1 + r.Int64N(maxW), Total: r.Int64N(maxT + 1)}
	}
	return rows
}

// pairwiseGini returns the Gini coefficient in basis points from the pairwise
// definition, with exact integers: with P the sum over i and j of
// |T_i * w_j - T_j * w_i|, the coefficient is P / (2 * W * S) (a row of weight w
// counting as w households), and 10 000 times it is rounded half to even once.
// It is the identity 2 * N = P of spec-indicators.md 5.4.1, with no float.
func pairwiseGini(rows []measure.Row) *big.Int {
	p, w, s := new(big.Int), new(big.Int), new(big.Int)
	for _, a := range rows {
		w.Add(w, big.NewInt(a.Weight))
		s.Add(s, big.NewInt(a.Total))
		for _, b := range rows {
			d := new(big.Int).Sub(new(big.Int).Mul(big.NewInt(a.Total), big.NewInt(b.Weight)), new(big.Int).Mul(big.NewInt(b.Total), big.NewInt(a.Weight)))
			p.Add(p, d.Abs(d))
		}
	}
	den := new(big.Int).Mul(w, s)
	return halfEven(new(big.Int).Mul(p, big.NewInt(10000)), den.Lsh(den, 1))
}

// TestGini_PairwiseDefinition verifies, for two thousand random row sets, that
// the Lorenz form of the Gini coefficient equals the pairwise definition in
// exact integers, so the sort by cross multiplication and the running totals
// are right, including for rows of weight above 1 whose total does not divide.
//
// Requirement: FR-IND-014
func TestGini_PairwiseDefinition(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(11, 12))
	for i := range 2000 {
		rows := randRows(r, 1+r.IntN(9), 1+r.Int64N(6), 1+r.Int64N(1000))
		got, ok, err := measure.Gini(rows)
		var sum int64
		for _, row := range rows {
			sum += row.Total
		}
		if sum == 0 {
			if ok || err != nil {
				t.Fatalf("set %d: Gini(%v) = %d, %v, %v, want not ok", i, rows, got, ok, err)
			}
			continue
		}
		if err != nil || !ok {
			t.Fatalf("set %d: Gini(%v) error = %v, ok = %v", i, rows, err, ok)
		}
		if want := pairwiseGini(rows); big.NewInt(got).Cmp(want) != 0 {
			t.Errorf("set %d: Gini(%v) = %d, want %v from the pairwise sum", i, rows, got, want)
		}
	}
}

// TestGini_RowOfWeightEqualsRepeatedRows verifies that a row of weight w and
// total w*v gives the coefficient of w rows of weight 1 holding v each, so a
// measure over agents and over the households they stand for agree.
//
// Requirement: FR-IND-014
func TestGini_RowOfWeightEqualsRepeatedRows(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(13, 14))
	for i := range 500 {
		var cohorts, expanded []measure.Row
		for range 1 + r.IntN(5) {
			w, v := 1+r.Int64N(5), r.Int64N(50)
			cohorts = append(cohorts, measure.Row{Weight: w, Total: w * v})
			for range w {
				expanded = append(expanded, measure.Row{Weight: 1, Total: v})
			}
		}
		g1, ok1, err1 := measure.Gini(cohorts)
		g2, ok2, err2 := measure.Gini(expanded)
		if g1 != g2 || ok1 != ok2 || err1 != nil || err2 != nil {
			t.Fatalf("case %d: Gini(%v) = %d, %v, %v; Gini(%v) = %d, %v, %v, want equal", i, cohorts, g1, ok1, err1, expanded, g2, ok2, err2)
		}
	}
}

// TestGini_EqualAndOneHolder verifies the two extremes: equal households give 0,
// and one household holding everything gives (W - 1) / W, 5 000 bp for two
// households, 9 000 for ten and 9 990 for a thousand, and a single household 0.
//
// Requirement: FR-IND-015
func TestGini_EqualAndOneHolder(t *testing.T) {
	t.Parallel()
	oneHolder := func(n int) []measure.Row {
		rows := make([]measure.Row, n)
		rows[0] = measure.Row{Weight: 1, Total: 100}
		for i := 1; i < n; i++ {
			rows[i] = measure.Row{Weight: 1}
		}
		return rows
	}
	tests := []struct {
		name string
		rows []measure.Row
		want int64
	}{
		{"equal households", []measure.Row{{1, 7}, {1, 7}, {1, 7}}, 0},
		{"equal cohorts of different weight", []measure.Row{{2, 14}, {5, 35}}, 0},
		{"one household", []measure.Row{{1, 9}}, 0},
		{"one cohort", []measure.Row{{4, 9}}, 0},
		{"two households, one holds all", oneHolder(2), 5000},
		{"ten households, one holds all", oneHolder(10), 9000},
		{"a thousand households, one holds all", oneHolder(1000), 9990},
		{"one holder among a cohort of nine", []measure.Row{{1, 100}, {9, 0}}, 9000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok, err := measure.Gini(tt.rows)
			if err != nil || !ok || got != tt.want {
				t.Errorf("Gini(%v) = %d, %v, %v, want %d, true, nil", tt.rows, got, ok, err, tt.want)
			}
		})
	}
}

// TestGini_UndefinedAndNegative verifies the three outcomes that are not a
// coefficient: an empty set or a total of 0 is not ok and no error, a negative
// total is ErrNegative, a weight below 1 is ErrRange, and a product past the
// guard of 5.2.1 is ErrOverflow, never a wrapped value.
//
// Requirement: FR-IND-014, FR-IND-015, FR-IND-009
func TestGini_UndefinedAndNegative(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		rows   []measure.Row
		wantOK bool
		err    error
	}{
		{"no rows", nil, false, nil},
		{"everyone holds nothing", []measure.Row{{1, 0}, {3, 0}}, false, nil},
		{"a negative total", []measure.Row{{1, 5}, {1, -1}}, false, measure.ErrNegative},
		{"a weight of zero", []measure.Row{{0, 5}, {1, 1}}, false, measure.ErrRange},
		{"a negative weight", []measure.Row{{-2, 5}, {1, 1}}, false, measure.ErrRange},
		{"weights past uint64", []measure.Row{{math.MaxInt64, 1}, {math.MaxInt64, 1}, {2, 1}}, false, measure.ErrOverflow},
		{"10 000 * W * S past 2^127", []measure.Row{{1 << 62, 1 << 62}, {1, 1}}, false, measure.ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok, err := measure.Gini(tt.rows)
			if ok != tt.wantOK || !errors.Is(err, tt.err) || (err != nil && got != 0) {
				t.Errorf("Gini(%v) = %d, %v, %v, want ok %v, error %v", tt.rows, got, ok, err, tt.wantOK, tt.err)
			}
		})
	}
}

// TestGini_OrderOfRowsDoesNotMatter verifies that shuffling the rows leaves the
// coefficient unchanged, with equal values among them.
//
// Requirement: FR-IND-010
func TestGini_OrderOfRowsDoesNotMatter(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(15, 16))
	for i := range 300 {
		rows := randRows(r, 2+r.IntN(8), 4, 6) // small totals make equal values common
		want, wantOK, _ := measure.Gini(rows)
		r.Shuffle(len(rows), func(a, b int) { rows[a], rows[b] = rows[b], rows[a] })
		if got, ok, _ := measure.Gini(rows); got != want || ok != wantOK {
			t.Fatalf("case %d: Gini after a shuffle = %d, %v, want %d, %v", i, got, ok, want, wantOK)
		}
	}
}

// FuzzGini verifies that Gini never panics, that a coefficient is between 0 and
// (W - 1) / W in basis points, and that it equals the pairwise definition, for
// any rows.
//
// Requirement: NFR-IND-003
func FuzzGini(f *testing.F) {
	f.Add([]byte{1, 5, 1, 9, 2, 0})
	f.Add([]byte{})
	f.Add([]byte{255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		var rows []measure.Row
		var w int64
		for i := 0; i+1 < len(data) && len(rows) < 12; i += 2 {
			rows = append(rows, measure.Row{Weight: int64(data[i]%7) + 1, Total: int64(data[i+1])})
			w += rows[len(rows)-1].Weight
		}
		got, ok, err := measure.Gini(rows)
		if err != nil {
			t.Fatalf("Gini(%v) error = %v", rows, err)
		}
		if !ok {
			return
		}
		if max := (w - 1) * 10000 / w; got < 0 || got > max+1 {
			t.Errorf("Gini(%v) = %d, want between 0 and %d", rows, got, max)
		}
		if want := pairwiseGini(rows); big.NewInt(got).Cmp(want) != 0 {
			t.Errorf("Gini(%v) = %d, want %v", rows, got, want)
		}
	})
}
