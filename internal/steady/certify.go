package steady

import "math/big"

// rat returns the exact value of r.
func rat(r Ratio) *big.Rat { return big.NewRat(r.Num, r.Den) }

// frat returns the exact value of f.
func frat(f Frac) *big.Rat { return new(big.Rat).SetFrac(f.Num, f.Den) }

// certify checks res against the original programme p, not against the
// tableau: every primal value is non-negative and every row holds (primal
// feasibility), every dual value is non-negative and every column's reduced
// cost has the right sign (dual feasibility), and the primal objective, the dual
// objective and res.Objective are equal (FR-VIA-013, FR-VIA-014). It returns
// ErrInconsistent on the first failure: a pivot that went wrong cannot pass.
func certify(p Programme, res Result) error {
	if len(res.Primal) != len(p.Cols) || len(res.Dual) != p.Rows {
		return ErrInconsistent
	}
	x, okX := nonNegative(res.Primal)
	y, okY := nonNegative(res.Dual)
	if !okX || !okY || !primalFeasible(p, x) {
		return ErrInconsistent
	}
	primal, dual, ok := objectives(p, x, y)
	if !ok || primal.Cmp(dual) != 0 || primal.Cmp(frat(res.Objective)) != 0 {
		return ErrInconsistent
	}
	return nil
}

// nonNegative returns the exact values of fs and whether each is a well-formed
// fraction that is not negative.
func nonNegative(fs []Frac) ([]*big.Rat, bool) {
	out := make([]*big.Rat, len(fs))
	for i, f := range fs {
		if f.Num == nil || f.Den == nil || f.Den.Sign() <= 0 {
			return nil, false
		}
		if out[i] = frat(f); out[i].Sign() < 0 {
			return nil, false
		}
	}
	return out, true
}

// primalFeasible reports whether every row A x <= RHS holds.
func primalFeasible(p Programme, x []*big.Rat) bool {
	use := make([]*big.Rat, p.Rows)
	for i := range use {
		use[i] = new(big.Rat)
	}
	tmp := new(big.Rat)
	for j, c := range p.Cols {
		for _, e := range c.Entries {
			use[e.Row].Add(use[e.Row], tmp.Mul(e.rat(), x[j]))
		}
	}
	for i, b := range p.RHS {
		if use[i].Cmp(rat(b)) > 0 {
			return false
		}
	}
	return true
}

// objectives returns Cost . x and RHS . y, and whether every column is dual
// feasible: sum over rows of y_i * a_ij is at least Cost_j.
func objectives(p Programme, x, y []*big.Rat) (primal, dual *big.Rat, ok bool) {
	primal, dual = new(big.Rat), new(big.Rat)
	tmp := new(big.Rat)
	ok = true
	for j, c := range p.Cols {
		primal.Add(primal, tmp.Mul(rat(c.Cost), x[j]))
		reduced := new(big.Rat)
		for _, e := range c.Entries {
			reduced.Add(reduced, tmp.Mul(e.rat(), y[e.Row]))
		}
		if reduced.Cmp(rat(c.Cost)) < 0 {
			ok = false
		}
	}
	for i, b := range p.RHS {
		dual.Add(dual, tmp.Mul(rat(b), y[i]))
	}
	return primal, dual, ok
}
