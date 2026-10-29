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
	// ResRent is the dual of each resource row (Model.Resources order), in the
	// unit of Wage: the value of one more unit of draw a tick, 0 when the row is
	// slack (spec-resources.md 11.3). ResTight reports that the draw equals the
	// sustainable draw ResPhi, and ResDraw is the draw a tick at lambda = 1, the
	// executions of Flow times the quantities drawn.
	ResRent  []Frac
	ResTight []bool
	ResPhi   []Frac
	ResDraw  []Frac
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
	o.readResources(m, l, res, lam)
	return o, o.round()
}

// readResources fills the resource fields of the outcome. The draw of a row is
// the sum over the activities that draw on it of the quantity times x; a row
// written divided by its sustainable draw (Layout.ResPhi) has its dual divided by
// it again to be the dual of the row as the spec writes it.
func (o *Outcome) readResources(m Model, l Layout, res Result, lam *big.Rat) {
	use := make([]*big.Rat, len(m.Resources))
	for j := range use {
		use[j] = new(big.Rat)
	}
	byPair := resRows(m.Resources)
	for k, a := range m.Activities {
		x := frat(res.Primal[k])
		for _, d := range a.Draws {
			j := byPair[[2]int{d.Resource, a.Region}]
			use[j].Add(use[j], new(big.Rat).Mul(x, big.NewRat(d.Qty, 1)))
		}
	}
	for j := range m.Resources {
		phi := l.ResPhi[j]
		y := frat(res.Dual[l.ResRow[j]])
		if phi.Sign() > 0 {
			y = new(big.Rat).Quo(y, phi)
		}
		o.ResRent = append(o.ResRent, fromRat(y))
		o.ResTight = append(o.ResTight, use[j].Cmp(phi) == 0)
		o.ResPhi = append(o.ResPhi, fromRat(phi))
		o.ResDraw = append(o.ResDraw, flow(use[j], lam))
	}
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
