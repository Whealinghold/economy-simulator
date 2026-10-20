package steady

import (
	"errors"
	"fmt"
	"math/big"
)

// Ratio is the exact rational Num/Den of a programme's data. Den must be
// positive.
type Ratio struct {
	Num, Den int64
}

// Entry is one non-zero coefficient of a column: Val in row Row. A coefficient
// that does not fit two int64 (a net vector, activity.go) is given in Wide, which
// then takes the place of Val.
type Entry struct {
	Row  int
	Val  Ratio
	Wide *big.Rat
}

// parts returns the numerator and the positive denominator of the entry.
func (e Entry) parts() (num, den *big.Int) {
	if e.Wide != nil {
		return e.Wide.Num(), e.Wide.Denom()
	}
	return big.NewInt(e.Val.Num), big.NewInt(e.Val.Den)
}

// rat returns the exact value of the entry.
func (e Entry) rat() *big.Rat {
	if e.Wide != nil {
		return e.Wide
	}
	return rat(e.Val)
}

// Column is one structural variable of a programme: its objective coefficient
// and its coefficients in the rows. Two entries for one row add.
type Column struct {
	Cost    Ratio
	Entries []Entry
}

// Programme is "maximise Cost . x subject to A x <= RHS, x >= 0" with Rows
// rows. RHS is the signed fixed term of each row; this solver needs every one
// non-negative, so that the slack basis is feasible and no first phase is
// needed (a negative one is ErrInfeasibleStart).
type Programme struct {
	Rows int
	RHS  []Ratio
	Cols []Column
}

// Frac is an exact fraction in lowest terms with a positive denominator. It is
// the only form in which a value leaves the solver before the boundary
// functions of round.go.
type Frac struct {
	Num, Den *big.Int
}

// Errors the solver returns. ErrInconsistent cannot be caused by input and is a
// defect of the solver.
var (
	// ErrBadProgramme reports data that is not a programme: a row out of range,
	// a denominator that is not positive, or an RHS of the wrong length.
	ErrBadProgramme = errors.New("steady: malformed programme")
	// ErrInfeasibleStart reports a negative fixed term, which the slack basis
	// cannot start from.
	ErrInfeasibleStart = errors.New("steady: negative right-hand side needs a first phase")
	// ErrUnbounded reports a programme whose objective can grow without limit.
	ErrUnbounded = errors.New("steady: programme is unbounded")
	// ErrInconsistent reports that the certificate failed.
	ErrInconsistent = errors.New("steady: certificate failed")
	// ErrLimit is matched by a LimitError.
	ErrLimit = errors.New("steady: pivot limit reached")
)

// LimitError is the error of a solve that reached its pivot cap. It carries the
// pivots made and the size of the tableau, and matches ErrLimit.
type LimitError struct {
	Pivots     int64
	Rows, Cols int
}

// Error implements error.
func (e *LimitError) Error() string {
	return fmt.Sprintf("steady: pivot limit reached after %d pivots (%d rows, %d columns)", e.Pivots, e.Rows, e.Cols)
}

// Unwrap returns ErrLimit.
func (e *LimitError) Unwrap() error { return ErrLimit }

// tableau holds the integer tableau a[i][j] / d. Rows 0..m-1 are the
// constraints, row m is the objective (z - c.x = 0), columns 0..n-1 the
// structural variables, n..n+m-1 the slacks and n+m the right-hand side.
type tableau struct {
	m, n     int
	a        [][]*big.Int
	d        *big.Int
	basis    []int
	rowScale []*big.Int // L_i: row i was multiplied by it, its slack is L_i * s_i
	objScale *big.Int   // Lc: the objective row was multiplied by it
}

// rhs is the index of the right-hand-side column.
func (t *tableau) rhs() int { return t.n + t.m }

// checkProgramme validates the data of p before any work.
func checkProgramme(p Programme) error {
	if p.Rows < 0 || len(p.RHS) != p.Rows {
		return ErrBadProgramme
	}
	for _, b := range p.RHS {
		if b.Den <= 0 {
			return ErrBadProgramme
		}
		if b.Num < 0 {
			return ErrInfeasibleStart
		}
	}
	for _, c := range p.Cols {
		if !validColumn(c, p.Rows) {
			return ErrBadProgramme
		}
	}
	return nil
}

// validColumn reports whether the column has positive denominators and rows in
// [0, rows).
func validColumn(c Column, rows int) bool {
	if c.Cost.Den <= 0 {
		return false
	}
	for _, e := range c.Entries {
		if e.Row < 0 || e.Row >= rows || (e.Wide == nil && e.Val.Den <= 0) {
			return false
		}
	}
	return true
}

// lcm returns the least common multiple of a and b, both positive.
func lcm(a, b *big.Int) *big.Int {
	g := new(big.Int).GCD(nil, nil, a, b)
	r := new(big.Int).Quo(a, g)
	return r.Mul(r, b)
}

// scaleTo returns v scaled to the integer v.Num * (l / v.Den) for a multiple l
// of v.Den.
func scaleTo(v Ratio, l *big.Int) *big.Int {
	return scaleParts(big.NewInt(v.Num), big.NewInt(v.Den), l)
}

// scaleParts returns num/den scaled to the integer num * (l / den) for a
// multiple l of den.
func scaleParts(num, den, l *big.Int) *big.Int {
	f := new(big.Int).Quo(l, den)
	return f.Mul(f, num)
}

// newTableau builds the starting tableau of p, which checkProgramme accepted:
// every row, and the objective, multiplied by the least common multiple of its
// denominators so that all entries are integers and the slack basis has
// coefficient 1.
func newTableau(p Programme) *tableau {
	m, n := p.Rows, len(p.Cols)
	t := &tableau{m: m, n: n, d: big.NewInt(1), basis: make([]int, m)}
	t.a = make([][]*big.Int, m+1)
	for i := range t.a {
		t.a[i] = make([]*big.Int, n+m+1)
		for j := range t.a[i] {
			t.a[i][j] = new(big.Int)
		}
	}
	t.rowScale = make([]*big.Int, m)
	for i := range t.rowScale {
		t.rowScale[i] = big.NewInt(p.RHS[i].Den)
	}
	t.objScale = big.NewInt(1)
	for _, c := range p.Cols {
		t.objScale = lcm(t.objScale, big.NewInt(c.Cost.Den))
		for _, e := range c.Entries {
			_, den := e.parts()
			t.rowScale[e.Row] = lcm(t.rowScale[e.Row], den)
		}
	}
	for j, c := range p.Cols {
		t.a[m][j].Neg(scaleTo(c.Cost, t.objScale))
		for _, e := range c.Entries {
			num, den := e.parts()
			t.a[e.Row][j].Add(t.a[e.Row][j], scaleParts(num, den, t.rowScale[e.Row]))
		}
	}
	for i := 0; i < m; i++ {
		t.a[i][n+i].SetInt64(1)
		t.a[i][n+m] = scaleTo(p.RHS[i], t.rowScale[i])
		t.basis[i] = n + i
	}
	return t
}

// pivot makes column c basic in row r (a[r][c] > 0) by an integer-preserving
// step: every other entry becomes (p*a[i][j] - a[i][c]*a[r][j]) / d, an exact
// division, and the new common denominator is p. It reports an inexact division,
// which cannot happen on a correct tableau.
func (t *tableau) pivot(r, c int) error {
	p := new(big.Int).Set(t.a[r][c])
	tmp, rem := new(big.Int), new(big.Int)
	for i := range t.a {
		if i == r {
			continue
		}
		f := new(big.Int).Set(t.a[i][c])
		for j, v := range t.a[i] {
			v.Mul(v, p)
			tmp.Mul(f, t.a[r][j])
			v.Sub(v, tmp)
			v.QuoRem(v, t.d, rem)
			if rem.Sign() != 0 {
				return ErrInconsistent
			}
		}
	}
	t.d = p
	t.basis[r] = c
	return nil
}
