package steady

import (
	"context"
	"errors"
	"math"
	"math/big"
	"slices"
	"testing"
)

// TestErrorTextsMutation pins the text of every sentinel error of the package.
//
// Guard: gomu string_literal survivors in program.go:12, 15 and tableau.go:69-78.
func TestErrorTextsMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		err  error
		want string
	}{
		{ErrOverflow, "steady: value out of int64 range"},
		{ErrBadProgramme, "steady: malformed programme"},
		{ErrInfeasible, "steady: programme is infeasible"},
		{ErrUnbounded, "steady: programme is unbounded"},
		{ErrInconsistent, "steady: certificate failed"},
		{ErrLimit, "steady: pivot limit reached"},
		{&LimitError{Pivots: 1, Rows: 2, Cols: 3}, "steady: pivot limit reached after 1 pivots (2 rows, 3 columns)"},
	}
	for _, tc := range tests {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
	}
}

// TestCheckProgrammeMutation pins each rejection of a malformed programme.
//
// Guard: gomu survivors tableau.go:113, 117, 135, 139.
func TestCheckProgrammeMutation(t *testing.T) {
	t.Parallel()
	one := Ratio{Num: 1, Den: 1}
	tests := []struct {
		name string
		p    Programme
		want error
	}{
		{"valid", Programme{Rows: 1, RHS: []Ratio{one}, Cols: []Column{{Cost: one, Entries: []Entry{{Row: 0, Val: one}}}}}, nil},
		{"rhs longer than rows", Programme{Rows: 1, RHS: []Ratio{one, one}}, ErrBadProgramme},
		{"rhs shorter than rows", Programme{Rows: 2, RHS: []Ratio{one}}, ErrBadProgramme},
		{"negative rows", Programme{Rows: -1}, ErrBadProgramme},
		{"negative rhs denominator", Programme{Rows: 1, RHS: []Ratio{{Num: 1, Den: -1}}}, ErrBadProgramme},
		{"zero rhs denominator", Programme{Rows: 1, RHS: []Ratio{{Num: 1, Den: 0}}}, ErrBadProgramme},
		{"negative rhs is a programme", Programme{Rows: 1, RHS: []Ratio{{Num: -1, Den: 1}}}, nil},
		{"negative cost denominator", Programme{Rows: 1, RHS: []Ratio{one}, Cols: []Column{{Cost: Ratio{Num: 1, Den: -1}}}}, ErrBadProgramme},
		{"zero cost denominator", Programme{Rows: 1, RHS: []Ratio{one}, Cols: []Column{{Cost: Ratio{Num: 1, Den: 0}}}}, ErrBadProgramme},
		{"zero entry denominator", Programme{Rows: 1, RHS: []Ratio{one}, Cols: []Column{{Cost: one, Entries: []Entry{{Row: 0, Val: Ratio{Num: 1, Den: 0}}}}}}, ErrBadProgramme},
		{"negative entry denominator", Programme{Rows: 1, RHS: []Ratio{one}, Cols: []Column{{Cost: one, Entries: []Entry{{Row: 0, Val: Ratio{Num: 1, Den: -1}}}}}}, ErrBadProgramme},
		{"wide entry needs no denominator", Programme{Rows: 1, RHS: []Ratio{one}, Cols: []Column{{Cost: one, Entries: []Entry{{Row: 0, Wide: big.NewRat(1, 2)}}}}}, nil},
		{"row below range", Programme{Rows: 1, RHS: []Ratio{one}, Cols: []Column{{Cost: one, Entries: []Entry{{Row: -1, Val: one}}}}}, ErrBadProgramme},
		{"row at range", Programme{Rows: 1, RHS: []Ratio{one}, Cols: []Column{{Cost: one, Entries: []Entry{{Row: 1, Val: one}}}}}, ErrBadProgramme},
	}
	for _, tc := range tests {
		if got := checkProgramme(tc.p); !errors.Is(got, tc.want) {
			t.Errorf("%s: checkProgramme = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestPivotMutation pins the exact-division step of a pivot and its error on an
// inexact one, positive and negative remainders alike.
//
// Guard: gomu survivors tableau.go:223-229.
func TestPivotMutation(t *testing.T) {
	t.Parallel()
	mk := func(d int64, rows ...[]int64) *tableau {
		tb := &tableau{m: len(rows) - 1, n: 1, d: big.NewInt(d), basis: []int{1}}
		for _, r := range rows {
			var row []*big.Int
			for _, v := range r {
				row = append(row, big.NewInt(v))
			}
			tb.a = append(tb.a, row)
		}
		return tb
	}
	t.Run("exact", func(t *testing.T) {
		t.Parallel()
		// Objective entry on the rhs: (0*2 - (-4)*4) / 2 = 8; d becomes the pivot 2.
		tb := mk(2, []int64{2, 0, 4}, []int64{-4, 0, 0})
		if err := tb.pivot(0, 0); err != nil {
			t.Fatalf("pivot error = %v", err)
		}
		if got := tb.a[1][2].Int64(); got != 8 {
			t.Errorf("a[1][2] = %d, want 8", got)
		}
		if tb.d.Int64() != 2 || tb.basis[0] != 0 {
			t.Errorf("d = %v, basis = %v, want 2, [0]", tb.d, tb.basis)
		}
	})
	for _, v := range []int64{1, -1} {
		// (v*1 - 0) / 2 leaves the remainder v, of either sign.
		tb := mk(2, []int64{1, 1, 1}, []int64{0, v, 0})
		if err := tb.pivot(0, 0); !errors.Is(err, ErrInconsistent) {
			t.Errorf("pivot with remainder %d: error = %v, want ErrInconsistent", v, err)
		}
	}
}

// TestBuildMutation pins the checks and rows of Build.
//
// Guard: gomu survivors program.go:56, 86, 95, 100, 136, 162, 172.
func TestBuildMutation(t *testing.T) {
	t.Parallel()
	base := func() Model {
		return Model{Commodities: 1, Regions: []Region{{Households: 1}}, Year: 10, Phi: Ratio{Num: 1, Den: 1}}
	}
	capped := func() Activity {
		a := full(nil, [][2]int64{{0, 1}})
		a.Duration = 1
		return a
	}
	t.Run("fixed terms", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			fixed []Ratio
			want  error
		}{{[]Ratio{{0, 1}}, nil}, {[]Ratio{{5, 1}}, nil}, {[]Ratio{{-5, 2}}, nil}, {[]Ratio{{0, 1}, {0, 1}}, ErrBadProgramme}, {[]Ratio{{1, 0}}, ErrBadProgramme}, {[]Ratio{{1, -1}}, ErrBadProgramme}, {[]Ratio{{math.MinInt64, 1}}, ErrOverflow}} {
			m := base()
			m.Fixed = tc.fixed
			if _, _, err := Build(m); !errors.Is(err, tc.want) {
				t.Errorf("Build(Fixed %v) error = %v, want %v", tc.fixed, err, tc.want)
			}
		}
	})
	t.Run("model bounds", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			edit func(m *Model)
			want error
		}{
			{"negative phi denominator", func(m *Model) { m.Phi.Den = -1 }, ErrBadProgramme},
			{"zero phi denominator", func(m *Model) { m.Phi.Den = 0 }, ErrBadProgramme},
			{"year of one tick", func(m *Model) { m.Year = 1 }, nil},
			{"year of none", func(m *Model) { m.Year = 0 }, ErrBadProgramme},
		} {
			m := base()
			tc.edit(&m)
			if _, _, err := Build(m); !errors.Is(err, tc.want) {
				t.Errorf("%s: Build error = %v, want %v", tc.name, err, tc.want)
			}
		}
	})
	t.Run("region at range panics", func(t *testing.T) {
		t.Parallel()
		m := base()
		a := capped()
		a.Region = 1
		m.Activities = []Activity{a}
		if got := panicText(func() { _, _, _ = Build(m) }); got != "steady: invariant violated: activity region out of range" {
			t.Errorf("panic = %q", got)
		}
	})
	t.Run("capacity rows are distinct", func(t *testing.T) {
		t.Parallel()
		m := base()
		m.Activities = []Activity{capped(), capped()}
		_, l, err := Build(m)
		if err != nil || !slices.Equal(l.CapRow, []int{2, 3}) {
			t.Errorf("CapRow = %v, %v, want [2 3]", l.CapRow, err)
		}
	})
	t.Run("hasCap", func(t *testing.T) {
		t.Parallel()
		// J * d = 3 * 2 = 6: below 10 caps, equal to 6 and above does not.
		a := Activity{Starts: 3, Duration: 2}
		for year, want := range map[int64]bool{10: true, 6: false, 4: false} {
			if got := hasCap(a, year); got != want {
				t.Errorf("hasCap(year %d) = %v, want %v", year, got, want)
			}
		}
	})
	t.Run("lambda column", func(t *testing.T) {
		t.Parallel()
		// Upkeep -2, 0, 3 and a stray 7 past the commodities (row 3 is a labour row).
		m := Model{Commodities: 3, Regions: []Region{{Upkeep: []int64{-2, 0, 3, 7}}}}
		col, err := lambdaColumn(m)
		if err != nil {
			t.Fatalf("lambdaColumn error = %v", err)
		}
		want := []Entry{{Row: 0, Val: Ratio{Num: -2, Den: 1}}, {Row: 2, Val: Ratio{Num: 3, Den: 1}}}
		if !slices.EqualFunc(col.Entries, want, func(a, b Entry) bool { return a.Row == b.Row && a.Val == b.Val }) {
			t.Errorf("Entries = %v, want %v", col.Entries, want)
		}
	})
}

// TestSolveLimitMutation pins the pivot cap for a negative cap and the size
// reported in the error.
//
// Guard: gomu survivors simplex.go:63, 64.
func TestSolveLimitMutation(t *testing.T) {
	t.Parallel()
	_, err := Solve(context.Background(), twoVar(), -1)
	if !errors.Is(err, ErrLimit) {
		t.Errorf("Solve(cap -1) error = %v, want ErrLimit", err)
	}
	_, err = Solve(context.Background(), twoVar(), 0)
	var le *LimitError
	if !errors.As(err, &le) || le.Pivots != 0 || le.Rows != 3 || le.Cols != 5 {
		t.Errorf("Solve(cap 0) error = %v, want LimitError{0 pivots, 3 rows, 5 columns}", err)
	}
}

// TestBlandTieMutation pins the leaving row on a tied ratio: the row whose basic
// variable has the lowest index.
//
// Guard: gomu survivors simplex.go:105.
func TestBlandTieMutation(t *testing.T) {
	t.Parallel()
	one := func(n int64) Ratio { return Ratio{Num: n, Den: 1} }
	tests := []struct {
		name string
		p    Programme
		want []int
	}{
		// max x with x <= 1 twice: the rows tie, the lower slack (1) leaves, so x
		// is basic in row 0 and the second slack (2) stays in row 1.
		{"equal basis order", Programme{Rows: 2, RHS: []Ratio{one(1), one(1)}, Cols: []Column{
			{Cost: one(1), Entries: []Entry{{Row: 0, Val: one(1)}, {Row: 1, Val: one(1)}}},
		}}, []int{0, 2}},
		// max x + 2y with x + 2y <= 2 and x + y <= 1. x enters and row 1 leaves
		// (1 < 2). Then y enters and rows 0 and 1 tie at 1; row 1 holds x (index 0)
		// and row 0 the slack (index 2), so row 1 leaves: basis [2, 1].
		{"later row with the lower basis", Programme{Rows: 2, RHS: []Ratio{one(2), one(1)}, Cols: []Column{
			{Cost: one(1), Entries: []Entry{{Row: 0, Val: one(1)}, {Row: 1, Val: one(1)}}},
			{Cost: one(2), Entries: []Entry{{Row: 0, Val: one(2)}, {Row: 1, Val: one(1)}}},
		}}, []int{2, 1}},
	}
	for _, tc := range tests {
		res, err := Solve(context.Background(), tc.p, 100)
		if err != nil || !slices.Equal(res.Basis, tc.want) {
			t.Errorf("%s: Basis = %v, %v, want %v", tc.name, res.Basis, err, tc.want)
		}
	}
}

// TestCertifyMutation pins the length, sign and objective checks of the
// certificate on corruptions the table-driven test does not make.
//
// Guard: gomu survivors certify.go:18, 27, 38, 41, 61, 62.
func TestCertifyMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(res *Result)
	}{
		{"primal too long", func(r *Result) { r.Primal = append(r.Primal, fracOf(0)) }},
		{"dual too long", func(r *Result) { r.Dual = append(r.Dual, fracOf(0)) }},
		{"claimed objective below", func(r *Result) { r.Objective = fracOf(10) }},
		{"negative denominator", func(r *Result) { r.Primal[0] = Frac{Num: big.NewInt(-3), Den: big.NewInt(-1)} }},
		{"negative value", func(r *Result) { r.Dual[0] = fracOf(-1) }},
		{"row exceeded by one", func(r *Result) { r.Primal[0] = fracOf(5) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, res := solved(t)
			tc.mutate(&res)
			if err := certify(p, res); !errors.Is(err, ErrInconsistent) {
				t.Errorf("certify = %v, want ErrInconsistent", err)
			}
		})
	}
	t.Run("nonNegative", func(t *testing.T) {
		if _, ok := nonNegative([]Frac{fracOf(1), fracOf(-1)}); ok {
			t.Error("nonNegative accepted a negative value")
		}
		if _, ok := nonNegative([]Frac{{Num: big.NewInt(1), Den: big.NewInt(-1)}}); ok {
			t.Error("nonNegative accepted a negative denominator")
		}
	})
}

// TestOutcomeRoundMutation pins the int64 bound of the surplus and the labour
// share, and the flow of a zero surplus.
//
// Guard: gomu survivors results.go:10, 101, 113, 114.
func TestOutcomeRoundMutation(t *testing.T) {
	t.Parallel()
	t.Run("surplus at the bound", func(t *testing.T) {
		t.Parallel()
		// lambda = maxInt64 / 10 000, so floor(lambda * 10 000) = maxInt64.
		o := Outcome{Lambda: Frac{Num: big.NewInt(maxInt64), Den: big.NewInt(10000)}}
		if err := o.round(); err != nil || o.SurplusBP != maxInt64 {
			t.Errorf("round = %v, SurplusBP = %d, want nil, %d", err, o.SurplusBP, int64(maxInt64))
		}
	})
	t.Run("surplus past the bound", func(t *testing.T) {
		t.Parallel()
		o := Outcome{Lambda: Frac{Num: new(big.Int).Add(big.NewInt(maxInt64), big.NewInt(1)), Den: big.NewInt(10000)}}
		if err := o.round(); !errors.Is(err, ErrOverflow) {
			t.Errorf("round = %v, want ErrOverflow", err)
		}
	})
	t.Run("zero lambda flow", func(t *testing.T) {
		t.Parallel()
		got := flow(big.NewRat(3, 1), new(big.Rat))
		if got.Num.Sign() != 0 || got.Den.Int64() != 1 {
			t.Errorf("flow = %v/%v, want 0/1", got.Num, got.Den)
		}
	})
}
