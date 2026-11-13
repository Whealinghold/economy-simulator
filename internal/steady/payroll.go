package steady

import "math/big"

// This file is the payroll-aware duals of spec-viability.md 5.6: the employment
// module pays a share of a payer group's receipts to the households of the
// Hireable groups that worked, which moves money and no goods, so the primal of
// a solved Model is kept and only the duals (prices, wages, capacity and
// resource rents) are solved again, from a linear system in exact rationals.
// Nothing here is a float, a random draw or a map iteration that reaches a
// result.

// PayrollReason says why PayrollSolve kept the payroll-free duals. The zero
// value is none.
type PayrollReason int

// The reasons of PayrollSolve (spec-viability.md 5.6, FR-VIA-069).
const (
	// PayrollOK: the duals were re-solved, or there was nothing to pay (an
	// identity).
	PayrollOK PayrollReason = iota
	// PayrollUnsupported: the model has lanes, a group is both payer and
	// Hireable, or the input does not fit the model.
	PayrollUnsupported
	// PayrollNoUniqueSolution: the system has no solution, or more than one.
	PayrollNoUniqueSolution
	// PayrollNegativeDual: the solution has a negative price, wage or rent.
	PayrollNegativeDual
	// PayrollInactiveProfit: an inactive activity pays more than the net wage.
	PayrollInactiveProfit
)

// PayrollGroup is a (region, role) group as the payroll sees it. BP is the
// payroll share in basis points of a payer (0 for a group that is not one) and
// Hireable says its households are paid by the payers of the region. Own is the
// units of each commodity the group's own households consume in a tick (by
// commodity; a missing entry is 0): the sales of a payer are its net output less
// this.
type PayrollGroup struct {
	Region   int
	BP       int64
	Hireable bool
	Own      []int64
}

// PayrollInput gives the groups and, for each activity of the Model (in its
// order), the index of the group that runs it, or -1 for an activity of no
// payer and no Hireable group.
type PayrollInput struct {
	Groups []PayrollGroup
	Group  []int
}

// PayrollResult is the answer of PayrollSolve. Outcome is the given outcome with
// the duals replaced when Applied, and the given outcome otherwise. Out, In and
// Labour are by group, in the units of the duals at the flows of lambda 1: the
// pay a payer group pays, the pay a Hireable group receives and the labour of the
// group in household-ticks a tick. They are 0 unless Applied.
type PayrollResult struct {
	Outcome         Outcome
	Out, In, Labour []Frac
	Applied         bool
	Reason          PayrollReason
}

// payroll is the working state of one PayrollSolve.
type payroll struct {
	m   Model
	o   Outcome
	in  PayrollInput
	l   Layout
	f   []*big.Rat   // flow of each activity at lambda 1
	lab []*big.Rat   // labour of each group
	net [][]NetEntry // net vector of each activity
	// sold[g] is the units of each commodity a payer group puts on the market.
	sold [][]*big.Rat
	// pool[r] is the labour of the Hireable groups of region r.
	pool []*big.Rat
	// flowing[r] says region r pays: a payer with labour and a pool above 0.
	flowing []bool
	// unk maps a dual to its unknown, -1 for a dual held at 0: goods rows, then
	// one wage per region, then a rent per activity, then a rent per resource.
	goods, wage, rent, res []int
	byPair                 map[[2]int]int // resource row of a (resource, region) pair
	n                      int
}

// PayrollSolve re-solves the duals of the outcome o of the model m for the
// payroll in (spec-viability.md 5.6). It keeps the primal (lambda, the flows, the
// labour share), solves the equations of the net wage exactly, and returns the
// given outcome with Reason set when there is no acceptable solution. A model
// with lanes is unsupported. It panics on an input that does not fit the model
// (a programmer invariant).
func PayrollSolve(m Model, o Outcome, in PayrollInput) PayrollResult {
	res := PayrollResult{Outcome: o, Reason: PayrollOK}
	if len(in.Group) != len(m.Activities) {
		panic("steady: invariant violated: payroll group list does not match the activities")
	}
	if len(m.Lanes) > 0 || badGroups(m, in) || badShape(m, o) {
		res.Reason = PayrollUnsupported
		return res
	}
	_, l, err := Build(m)
	if err != nil {
		res.Reason = PayrollUnsupported
		return res
	}
	p := &payroll{m: m, o: o, in: in, l: l}
	p.prepare()
	res.Out, res.In, res.Labour = zeros(len(in.Groups)), zeros(len(in.Groups)), zeros(len(in.Groups))
	for g := range in.Groups {
		res.Labour[g] = fromRat(p.lab[g])
	}
	if !p.anyFlowing() {
		return res // nothing is paid: the payroll-free duals are the answer
	}
	x, reason := p.solve()
	if reason != PayrollOK {
		res.Reason = reason
		return res
	}
	res.Applied = true
	res.Outcome = p.outcome(x)
	for g, gr := range in.Groups {
		switch {
		case gr.BP > 0 && p.flowing[gr.Region]:
			res.Out[g] = fromRat(p.out(g, x))
		case gr.Hireable && p.flowing[gr.Region]:
			res.In[g] = fromRat(new(big.Rat).Mul(p.sigma(gr.Region, x), p.lab[g]))
		}
	}
	return res
}

// zeros returns n zero fractions.
func zeros(n int) []Frac {
	out := make([]Frac, n)
	for i := range out {
		out[i] = fromRat(new(big.Rat))
	}
	return out
}

// badGroups reports a group both payer and Hireable, a region out of range, a
// negative or too large share, a negative own consumption, an activity assigned
// to a group that does not exist or is of another region, or an activity of no
// duration.
func badGroups(m Model, in PayrollInput) bool {
	for _, g := range in.Groups {
		if g.Region < 0 || g.Region >= len(m.Regions) || g.BP < 0 || g.BP > 10_000 || (g.BP > 0 && g.Hireable) {
			return true
		}
		for _, own := range g.Own {
			if own < 0 {
				return true
			}
		}
	}
	for k, g := range in.Group {
		if g < -1 || g >= len(in.Groups) || (g >= 0 && in.Groups[g].Region != m.Activities[k].Region) {
			return true
		}
		if m.Activities[k].Duration < 1 {
			return true
		}
	}
	return false
}

// badShape reports an outcome whose slices do not match the model (it was solved
// for another one): the solver would index past them.
func badShape(m Model, o Outcome) bool {
	return len(o.Flow) < len(m.Activities) || len(o.Active) != len(m.Activities) ||
		len(o.Rent) != len(m.Activities) || len(o.Price) != m.goodsRows() ||
		len(o.Wage) != len(m.Regions) || len(o.ResRent) != len(m.Resources)
}

// prepare computes the flows, labour, net vectors, sales and pools, and numbers
// the unknowns.
func (p *payroll) prepare() {
	m := p.m
	p.byPair = resRows(m.Resources)
	p.f = make([]*big.Rat, len(m.Activities))
	p.net = make([][]NetEntry, len(m.Activities))
	p.lab = make([]*big.Rat, len(p.in.Groups))
	for g := range p.lab {
		p.lab[g] = new(big.Rat)
	}
	for k, a := range m.Activities {
		p.f[k] = frat(p.o.Flow[k])
		p.net[k] = a.Net()
		if g := p.in.Group[k]; g >= 0 {
			p.lab[g].Add(p.lab[g], new(big.Rat).Mul(p.f[k], big.NewRat(a.Duration, 1)))
		}
	}
	p.sold = make([][]*big.Rat, len(p.in.Groups))
	p.pool = make([]*big.Rat, len(m.Regions))
	for r := range p.pool {
		p.pool[r] = new(big.Rat)
	}
	for g, gr := range p.in.Groups {
		if gr.Hireable {
			p.pool[gr.Region].Add(p.pool[gr.Region], p.lab[g])
		}
		if gr.BP > 0 {
			p.sold[g] = p.sales(g)
		}
	}
	p.flowing = make([]bool, len(m.Regions))
	for g, gr := range p.in.Groups {
		if gr.BP > 0 && p.lab[g].Sign() > 0 && p.pool[gr.Region].Sign() > 0 {
			p.flowing[gr.Region] = true
		}
	}
	p.number()
}

// sales returns, by commodity, the units payer group g puts on the market a tick
// at lambda 1: the net output of its activities less what its households eat,
// never below 0.
func (p *payroll) sales(g int) []*big.Rat {
	out := make([]*big.Rat, p.m.Commodities)
	for c := range out {
		out[c] = new(big.Rat)
	}
	for k := range p.m.Activities {
		if p.in.Group[k] != g {
			continue
		}
		for _, e := range p.net[k] {
			out[e.Commodity].Add(out[e.Commodity], new(big.Rat).Mul(p.f[k], e.Val))
		}
	}
	own := p.in.Groups[g].Own
	for c := range out {
		if c < len(own) {
			out[c].Sub(out[c], big.NewRat(own[c], 1))
		}
		if out[c].Sign() < 0 {
			out[c].SetInt64(0)
		}
	}
	return out
}

// anyFlowing reports a region that pays.
func (p *payroll) anyFlowing() bool {
	for _, f := range p.flowing {
		if f {
			return true
		}
	}
	return false
}

// number gives an unknown to each dual that is not 0 in the payroll-free outcome.
func (p *payroll) number() {
	next := func(f Frac) int {
		if f.Num.Sign() == 0 {
			return -1
		}
		p.n++
		return p.n - 1
	}
	for _, f := range p.o.Price {
		p.goods = append(p.goods, next(f))
	}
	for _, f := range p.o.Wage {
		p.wage = append(p.wage, next(f))
	}
	for _, f := range p.o.Rent {
		p.rent = append(p.rent, next(f))
	}
	for _, f := range p.o.ResRent {
		p.res = append(p.res, next(f))
	}
}

// row returns the coefficients of activity k in the unknowns, such that the
// activity pays its net wage when row . x = 0 and pays more than it when > 0:
// v(p) - burden + pay - d w - rent - resource rents (spec-viability.md 5.6).
func (p *payroll) row(k int) []*big.Rat {
	a := p.m.Activities[k]
	v := make([]*big.Rat, p.n)
	for i := range v {
		v[i] = new(big.Rat)
	}
	add := func(u int, c *big.Rat) {
		if u >= 0 {
			v[u].Add(v[u], c)
		}
	}
	d := big.NewRat(a.Duration, 1)
	for _, e := range p.net[k] {
		add(p.goods[p.l.GoodsRow(a.Region, e.Commodity)], e.Val)
	}
	add(p.wage[a.Region], new(big.Rat).Neg(d))
	add(p.rent[k], big.NewRat(-1, 1))
	for _, dr := range a.Draws {
		add(p.res[p.byPair[[2]int{dr.Resource, a.Region}]], big.NewRat(-dr.Qty, 1))
	}
	g := p.in.Group[k]
	if g < 0 || !p.flowing[a.Region] {
		return v
	}
	gr := p.in.Groups[g]
	switch {
	case gr.BP > 0 && p.lab[g].Sign() > 0:
		// minus d * beta * S(g, p) / L(g)
		coef := new(big.Rat).Quo(new(big.Rat).Mul(d, p.beta(g)), p.lab[g])
		for c, s := range p.sold[g] {
			add(p.goods[p.l.GoodsRow(a.Region, c)], new(big.Rat).Neg(new(big.Rat).Mul(coef, s)))
		}
	case gr.Hireable:
		// plus d * sigma(p), sigma = sum of beta * S over the payers / pool
		coef := new(big.Rat).Quo(d, p.pool[a.Region])
		for h, hr := range p.in.Groups {
			if hr.BP == 0 || hr.Region != a.Region {
				continue
			}
			for c, s := range p.sold[h] {
				add(p.goods[p.l.GoodsRow(a.Region, c)], new(big.Rat).Mul(new(big.Rat).Mul(coef, p.beta(h)), s))
			}
		}
	}
	return v
}

// beta returns the share of payer group g as a fraction of 1.
func (p *payroll) beta(g int) *big.Rat { return big.NewRat(p.in.Groups[g].BP, 10_000) }

// solve builds the equations of the active activities and the normalisation,
// eliminates exactly and checks the solution.
func (p *payroll) solve() ([]*big.Rat, PayrollReason) {
	var eq [][]*big.Rat
	for k := range p.m.Activities {
		if p.o.Active[k] {
			eq = append(eq, append(p.row(k), new(big.Rat)))
		}
	}
	norm := make([]*big.Rat, p.n+1)
	for i := range norm {
		norm[i] = new(big.Rat)
	}
	for c := 0; c < p.m.Commodities; c++ {
		tot := new(big.Rat)
		for _, rg := range p.m.Regions {
			if c < len(rg.Upkeep) {
				tot.Add(tot, big.NewRat(rg.Upkeep[c], 1))
			}
		}
		if u := p.goods[c]; u >= 0 {
			norm[u].Add(norm[u], tot)
		}
	}
	norm[p.n] = big.NewRat(1, 1)
	eq = append(eq, norm)
	x, ok := eliminate(eq, p.n)
	if !ok {
		return nil, PayrollNoUniqueSolution
	}
	for _, v := range x {
		if v.Sign() < 0 {
			return nil, PayrollNegativeDual
		}
	}
	for k := range p.m.Activities {
		if p.o.Active[k] {
			continue
		}
		r := p.row(k)
		sum := new(big.Rat)
		for i, c := range r {
			sum.Add(sum, new(big.Rat).Mul(c, x[i]))
		}
		if sum.Sign() > 0 {
			return nil, PayrollInactiveProfit
		}
	}
	return x, PayrollOK
}

// eliminate solves the n unknowns of the augmented rows exactly by Gauss-Jordan
// elimination, taking the first row with a non-zero entry as the pivot (a fixed
// order). It reports false when the rank is below n or a row reads 0 = non-zero.
func eliminate(eq [][]*big.Rat, n int) ([]*big.Rat, bool) {
	r := 0
	pivotRow := make([]int, n)
	for c := 0; c < n; c++ {
		pr := -1
		for i := r; i < len(eq); i++ {
			if eq[i][c].Sign() != 0 {
				pr = i
				break
			}
		}
		if pr < 0 {
			return nil, false
		}
		eq[r], eq[pr] = eq[pr], eq[r]
		inv := new(big.Rat).Inv(eq[r][c])
		for j := c; j <= n; j++ {
			eq[r][j].Mul(eq[r][j], inv)
		}
		for i := range eq {
			if i == r || eq[i][c].Sign() == 0 {
				continue
			}
			f := new(big.Rat).Set(eq[i][c])
			for j := c; j <= n; j++ {
				eq[i][j].Sub(eq[i][j], new(big.Rat).Mul(f, eq[r][j]))
			}
		}
		pivotRow[c] = r
		r++
	}
	for i := r; i < len(eq); i++ {
		if eq[i][n].Sign() != 0 {
			return nil, false
		}
	}
	x := make([]*big.Rat, n)
	for c := 0; c < n; c++ {
		x[c] = eq[pivotRow[c]][n]
	}
	return x, true
}

// value returns the value of the dual numbered u, 0 for a dual held at 0.
func value(x []*big.Rat, u int) *big.Rat {
	if u < 0 {
		return new(big.Rat)
	}
	return x[u]
}

// receipts returns S(g, p) of payer group g.
func (p *payroll) receipts(g int, x []*big.Rat) *big.Rat {
	r := p.in.Groups[g].Region
	sum := new(big.Rat)
	for c, s := range p.sold[g] {
		sum.Add(sum, new(big.Rat).Mul(s, value(x, p.goods[p.l.GoodsRow(r, c)])))
	}
	return sum
}

// out returns the pay beta * S of payer group g.
func (p *payroll) out(g int, x []*big.Rat) *big.Rat {
	return new(big.Rat).Mul(p.beta(g), p.receipts(g, x))
}

// sigma returns the pay per working household-tick in region r.
func (p *payroll) sigma(r int, x []*big.Rat) *big.Rat {
	tot := new(big.Rat)
	for g, gr := range p.in.Groups {
		if gr.BP > 0 && gr.Region == r {
			tot.Add(tot, p.out(g, x))
		}
	}
	return tot.Quo(tot, p.pool[r])
}

// outcome returns the given outcome with the duals of x.
func (p *payroll) outcome(x []*big.Rat) Outcome {
	o := p.o
	pick := func(old []Frac, nums []int) []Frac {
		out := make([]Frac, len(old))
		for i := range old {
			out[i] = fromRat(value(x, nums[i]))
		}
		return out
	}
	o.Price = pick(p.o.Price, p.goods)
	o.Wage = pick(p.o.Wage, p.wage)
	o.Rent = pick(p.o.Rent, p.rent)
	o.ResRent = pick(p.o.ResRent, p.res)
	return o
}
