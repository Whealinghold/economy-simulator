package steady

import (
	"context"
	"fmt"
	"math/big"
)

// The first phase of the simplex method (N-277). A row with a negative
// right-hand side cannot start from its slack, so the row is negated, which
// makes its slack -1 and its right-hand side positive, and an artificial
// variable with coefficient 1 starts in its place. The first phase maximises
// minus the sum of the artificials; a maximum below 0 means no feasible point
// (ErrInfeasible), and a maximum of 0 leaves a basis of the original programme.

// negateForPhase1 negates row i and makes artificial column col basic in it.
func (t *tableau) negateForPhase1(i, col int) {
	for j := range t.a[i] {
		t.a[i][j].Neg(t.a[i][j])
	}
	t.a[i][col].SetInt64(1)
	t.basis[i] = col
}

// startPhase1 replaces the objective row by the first phase's, priced out
// against the artificial basis, and returns the second phase's objective row
// as it stood (integers over the common denominator 1) for endPhase1.
func (t *tableau) startPhase1() []*big.Int {
	obj := t.a[t.m]
	t.a[t.m] = make([]*big.Int, len(obj))
	for j := range t.a[t.m] {
		t.a[t.m][j] = new(big.Int)
	}
	for i, b := range t.basis {
		if b < t.n+t.m {
			continue
		}
		for j := range t.a[t.m] {
			if j != b {
				t.a[t.m][j].Sub(t.a[t.m][j], t.a[i][j])
			}
		}
	}
	return obj
}

// endPhase1 reports ErrInfeasible when the first phase ended below 0, pivots
// every artificial still in the basis (at 0) out, and restores the second
// phase's objective row, priced out against the basis. Every row has its own
// slack, so a row whose artificial is basic at 0 always has a non-zero entry in
// a slack or structural column. It returns the pivots it made; each one is
// checked against the context and against maxPivots, with done pivots already
// made, as run checks its own.
func (t *tableau) endPhase1(ctx context.Context, maxPivots, done int64, obj []*big.Int) (int64, error) {
	if t.a[t.m][t.rhs()].Sign() != 0 {
		return 0, ErrInfeasible
	}
	var pivots int64
	for r, b := range t.basis {
		if b < t.n+t.m {
			continue
		}
		c := t.firstNonzero(r)
		if c < 0 {
			return pivots, ErrInconsistent
		}
		if err := ctx.Err(); err != nil {
			return pivots, fmt.Errorf("steady: solve cancelled after %d pivots: %w", done+pivots, err)
		}
		if done+pivots >= maxPivots {
			return pivots, &LimitError{Pivots: done + pivots, Rows: t.m, Cols: t.rhs()}
		}
		if t.a[r][c].Sign() < 0 {
			for _, v := range t.a[r] {
				v.Neg(v)
			}
		}
		if err := t.pivot(r, c); err != nil {
			return pivots, err
		}
		pivots++
	}
	t.restoreObjective(obj)
	return pivots, nil
}

// firstNonzero returns the lowest structural or slack column with a non-zero
// entry in row r, or -1. A basic column is zero in every other row, so the
// column found is never basic in another row.
func (t *tableau) firstNonzero(r int) int {
	for j := 0; j < t.n+t.m; j++ {
		if t.a[r][j].Sign() != 0 {
			return j
		}
	}
	return -1
}

// restoreObjective sets the objective row to obj (valid over the denominator 1)
// scaled to the current denominator d, then zeroes the entry of every basic
// structural column by subtracting that column's cost times its row.
func (t *tableau) restoreObjective(obj []*big.Int) {
	m := t.a[t.m]
	for j := range m {
		m[j].Mul(obj[j], t.d)
	}
	f := new(big.Int)
	for i, b := range t.basis {
		if obj[b].Sign() == 0 {
			continue
		}
		for j := range m {
			m[j].Sub(m[j], f.Mul(obj[b], t.a[i][j]))
		}
	}
}
