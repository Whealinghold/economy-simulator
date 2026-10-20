package steady

import (
	"context"
	"math/big"
)

// maxInt64 is the largest int64, spelled without the math package, which the
// non-test files of this package do not import (FR-VIA-015).
const maxInt64 = 1<<63 - 1

// Outcome is the certified optimum of a Model, read as spec-viability.md 5 reads
// it. Every Frac is exact. The slices are indexed by commodity, region or
// activity as the Model was, so their order is the canonical order of the input.
type Outcome struct {
	// Lambda is the surplus factor lambda*.
	Lambda Frac
	// SurplusBP is floor(lambda* * 10 000).
	SurplusBP int64
	// LabourShareBP is ceil(10 000 * 10 000 / SurplusBP), meaningful only when
	// HasLabourShare: it is absent when a capacity row is tight or SurplusBP is 0
	// (FR-VIA-012).
	LabourShareBP  int64
	HasLabourShare bool
	// Price is the dual of each goods row, in units of the basket: the
	// normalisation sum p D = 1 holds when lambda* is positive.
	Price []Frac
	// Wage is the dual of each region's labour row.
	Wage []Frac
	// Rent is the dual of each activity's capacity row, 0 without one.
	Rent []Frac
	// Flow is the executions per tick of each activity at lambda = 1, x* over
	// lambda* (D120); all 0 when lambda* is 0.
	Flow []Frac
	// Active reports x > 0 for each activity: the regions in which each activity
	// is active in a pooled solution (FR-VIA-045).
	Active []bool
	// CapTight reports that an activity's capacity row holds with equality.
	CapTight []bool
	// ZeroPriced lists, ascending, the commodities whose price is 0 (FR-VIA-019).
	ZeroPriced []int
	// Degenerate and Alternative are the diagnostics of the final basis
	// (FR-VIA-019).
	Degenerate, Alternative bool
	// Pivots is the number of pivots made.
	Pivots int64
}

// SolveModel builds the programme of m, solves it with Bland's rule under the
// certificate of Solve, and reads the result. It returns no Outcome with an
// error: ErrUnbounded for an empty basket, *LimitError at the pivot cap, the
// wrapped context error, ErrInconsistent for a failed certificate, and the
// errors of Build (FR-VIA-003, 013, 014).
func SolveModel(ctx context.Context, m Model, maxPivots int64) (Outcome, error) {
	p, l, err := Build(m)
	if err != nil {
		return Outcome{}, err
	}
	res, err := Solve(ctx, p, maxPivots)
	if err != nil {
		return Outcome{}, err
	}
	return read(m, p, l, res)
}

// read turns a certified Result into an Outcome.
func read(m Model, p Programme, l Layout, res Result) (Outcome, error) {
	o := Outcome{
		Lambda:      res.Primal[l.Lambda],
		Price:       res.Dual[:l.Commodities],
		Wage:        res.Dual[l.Commodities : l.Commodities+l.Regions],
		Degenerate:  res.Degenerate,
		Alternative: res.Alternative,
		Pivots:      res.Pivots,
	}
	one := big.NewInt(1)
	for c, f := range o.Price {
		if f.Num.Sign() == 0 {
			o.ZeroPriced = append(o.ZeroPriced, c)
		}
	}
	lam := frat(o.Lambda)
	for k := range m.Activities {
		x := frat(res.Primal[k])
		o.Active = append(o.Active, x.Sign() > 0)
		o.Flow = append(o.Flow, flow(x, lam))
		rent, tight := Frac{Num: new(big.Int), Den: one}, false
		if row := l.CapRow[k]; row >= 0 {
			rent = res.Dual[row]
			tight = x.Cmp(rat(p.RHS[row])) == 0
		}
		o.Rent = append(o.Rent, rent)
		o.CapTight = append(o.CapTight, tight)
	}
	return o, o.round()
}

// flow returns x / lambda as a Frac, 0 when lambda is 0.
func flow(x, lam *big.Rat) Frac {
	if lam.Sign() == 0 {
		return Frac{Num: new(big.Int), Den: big.NewInt(1)}
	}
	q := new(big.Rat).Quo(x, lam)
	return Frac{Num: q.Num(), Den: q.Denom()}
}

// round fills SurplusBP and the labour share, the one place a value of the
// result leaves as int64 (FR-VIA-003, FR-VIA-012). It returns ErrOverflow when
// the surplus does not fit.
func (o *Outcome) round() error {
	num := new(big.Int).Mul(o.Lambda.Num, big.NewInt(10000))
	bp, ok := Floor(num, o.Lambda.Den, 0, maxInt64)
	if !ok {
		return ErrOverflow
	}
	o.SurplusBP = bp
	if bp == 0 {
		return nil
	}
	for _, t := range o.CapTight {
		if t {
			return nil
		}
	}
	share, ok := Ceil(big.NewInt(10000*10000), big.NewInt(bp), 0, maxInt64)
	o.LabourShareBP, o.HasLabourShare = share, ok
	return nil
}
