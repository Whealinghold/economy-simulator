package steady

import (
	"context"
	"fmt"
	"math/big"
)

// Result is a certified optimum of a programme. Every value is exact.
type Result struct {
	// Primal is the value of each structural variable, in column order.
	Primal []Frac
	// Dual is the dual value of each row of the programme as given (not as
	// scaled inside the tableau), in row order.
	Dual []Frac
	// Objective is the optimal value of Cost . x, which equals RHS . Dual.
	Objective Frac
	// Basis is the variable that is basic in each row of the final tableau:
	// 0..len(Cols)-1 for a column, len(Cols)+i for the slack of row i.
	Basis []int
	// Pivots is the number of pivots made.
	Pivots int64
	// Degenerate reports that a basic variable is 0.
	Degenerate bool
	// Alternative reports that a non-basic variable has a zero reduced cost, so
	// the optimum may not be unique.
	Alternative bool
}

// Solve maximises the programme by the simplex method with Bland's rule and
// returns the optimum only after the certificate of certify holds. The context
// is checked before every pivot, the first included, and maxPivots is the most
// pivots allowed (0 allows none): the cap gives a *LimitError and the context
// gives its wrapped error, and neither returns a partial result. The order of
// the columns is part of the input (FR-VIA-022).
func Solve(ctx context.Context, p Programme, maxPivots int64) (Result, error) {
	if err := checkProgramme(p); err != nil {
		return Result{}, err
	}
	t := newTableau(p)
	pivots, err := t.solve(ctx, maxPivots)
	if err != nil {
		return Result{}, err
	}
	res := t.result(pivots)
	if err := certify(p, res); err != nil {
		return Result{}, err
	}
	return res, nil
}

// solve runs the first phase when a right-hand side was negative and then the
// second, which is the whole solve otherwise. The pivot cap counts both phases.
func (t *tableau) solve(ctx context.Context, maxPivots int64) (int64, error) {
	if t.na == 0 {
		return t.run(ctx, maxPivots, 0)
	}
	obj := t.startPhase1()
	pivots, err := t.run(ctx, maxPivots, 0)
	if err != nil {
		return pivots, err
	}
	out, err := t.endPhase1(obj)
	if err != nil {
		return pivots, err
	}
	return t.run(ctx, maxPivots, pivots+out)
}

// run pivots until no reduced cost is negative and returns the pivot count,
// which starts at done.
func (t *tableau) run(ctx context.Context, maxPivots, done int64) (int64, error) {
	pivots := done
	for {
		if err := ctx.Err(); err != nil {
			return pivots, fmt.Errorf("steady: solve cancelled after %d pivots: %w", pivots, err)
		}
		c := t.choose()
		if c < 0 {
			return pivots, nil
		}
		if pivots >= maxPivots {
			return pivots, &LimitError{Pivots: pivots, Rows: t.m, Cols: t.n + t.m}
		}
		r := t.ratio(c)
		if r < 0 {
			return pivots, ErrUnbounded
		}
		if err := t.pivot(r, c); err != nil {
			return pivots, err
		}
		pivots++
	}
}

// choose returns the entering column by Bland's rule: the lowest index with a
// negative reduced cost, or -1 when the tableau is optimal.
func (t *tableau) choose() int {
	for j := 0; j < t.n+t.m; j++ {
		if t.a[t.m][j].Sign() < 0 {
			return j
		}
	}
	return -1
}

// ratio returns the leaving row for entering column c: the least ratio of the
// right-hand side to the positive entries, a tie going to the row whose basic
// variable has the lowest index. It returns -1 when no entry is positive.
func (t *tableau) ratio(c int) int {
	best := -1
	lhs, rhs := new(big.Int), new(big.Int)
	for i := 0; i < t.m; i++ {
		if t.a[i][c].Sign() <= 0 {
			continue
		}
		if best < 0 {
			best = i
			continue
		}
		// b_i / a_ic against b_best / a_best,c, both denominators positive.
		lhs.Mul(t.a[i][t.rhs()], t.a[best][c])
		rhs.Mul(t.a[best][t.rhs()], t.a[i][c])
		if cmp := lhs.Cmp(rhs); cmp < 0 || (cmp == 0 && t.basis[i] < t.basis[best]) {
			best = i
		}
	}
	return best
}

// newFrac returns num/den in lowest terms with a positive denominator.
func newFrac(num, den *big.Int) Frac {
	r := new(big.Rat).SetFrac(num, den)
	return Frac{Num: new(big.Int).Set(r.Num()), Den: new(big.Int).Set(r.Denom())}
}

// result reads the optimum off the final tableau, undoing the row and objective
// scaling: x is unscaled, the objective is divided by Lc, and the dual of row i
// is the reduced cost of its slack times L_i over Lc.
func (t *tableau) result(pivots int64) Result {
	res := Result{
		Primal: make([]Frac, t.n),
		Dual:   make([]Frac, t.m),
		Basis:  append([]int(nil), t.basis...),
		Pivots: pivots,
	}
	zero := new(big.Int)
	basic := make([]bool, t.n+t.m+t.na)
	for i, b := range t.basis {
		basic[b] = true
		if t.a[i][t.rhs()].Sign() == 0 {
			res.Degenerate = true
		}
	}
	for j := 0; j < t.n+t.m; j++ {
		if !basic[j] && t.a[t.m][j].Cmp(zero) == 0 {
			res.Alternative = true
		}
	}
	for j := range res.Primal {
		res.Primal[j] = newFrac(zero, big.NewInt(1))
	}
	for i, b := range t.basis {
		if b < t.n {
			res.Primal[b] = newFrac(t.a[i][t.rhs()], t.d)
		}
	}
	for i := range res.Dual {
		num := new(big.Int).Mul(t.a[t.m][t.n+i], t.rowScale[i])
		den := new(big.Int).Mul(t.d, t.objScale)
		res.Dual[i] = newFrac(num, den)
	}
	res.Objective = newFrac(t.a[t.m][t.rhs()], new(big.Int).Mul(t.d, t.objScale))
	return res
}
