package steady

import (
	"errors"
	"math/big"
)

// Errors of building a reference programme.
var (
	// ErrOverflow reports a right-hand side or an upkeep total that does not fit
	// int64 once reduced.
	ErrOverflow = errors.New("steady: value out of int64 range")
)

// Region is the data of one region of a programme: its households and the
// upkeep demand D(c, rho) of each commodity per tick (spec-viability.md 3.1).
type Region struct {
	Households int64
	Upkeep     []int64 // by commodity; a missing entry is 0
}

// Model is the data of one reference programme (spec-viability.md 3.2). One
// region is the autarky form, several regions the pooled free-trade form with
// one goods row per commodity, and Phi other than 1 the nominal form. The caller
// gives Activities in canonical order, regions ascending (FR-VIA-022); the order
// within a region is the column order.
type Model struct {
	Commodities int
	Regions     []Region
	Year        int64 // TicksPerYear
	Phi         Ratio // available work share; 1/1 for the viability test
	// Fixed is the exact fixed term W(c) of each goods row, by commodity; a
	// missing entry is 0. The goods row is  -n x + D lambda <= -W, so a positive
	// term makes the right-hand side negative and the solve start with a first
	// phase (N-277).
	Fixed      []Ratio
	Activities []Activity
}

// Layout tells where the rows and columns of a built programme are: goods rows
// 0..Commodities-1, then one labour row per region, then the capacity rows.
type Layout struct {
	Commodities int
	Regions     int
	// CapRow is the row of each activity's capacity row, or -1 when it has none
	// (J times the duration reaches the year).
	CapRow []int
	// Lambda is the column of the surplus factor, after the activities.
	Lambda int
}

// LabourRow returns the labour row of region r.
func (l Layout) LabourRow(r int) int { return l.Commodities + r }

// hasCap reports whether the activity has a capacity row: J * d < TicksPerYear.
func hasCap(a Activity, year int64) bool {
	return new(big.Int).Mul(big.NewInt(a.Starts), big.NewInt(a.Duration)).Cmp(big.NewInt(year)) < 0
}

// Size returns the rows and columns the programme of m has, before it is built:
// rows are the goods, the regions and the capacity rows; columns the activities,
// lambda and one slack per row (D126).
func Size(m Model) (rows, cols int) {
	rows = m.Commodities + len(m.Regions)
	for _, a := range m.Activities {
		if hasCap(a, m.Year) {
			rows++
		}
	}
	return rows, len(m.Activities) + 1 + rows
}

// ratio reduces num/den and reports whether both parts fit int64.
func ratio(num, den *big.Int) (Ratio, bool) {
	r := new(big.Rat).SetFrac(num, den)
	if !r.Num().IsInt64() || !r.Denom().IsInt64() {
		return Ratio{}, false
	}
	return Ratio{Num: r.Num().Int64(), Den: r.Denom().Int64()}, true
}

// checkModel panics on a Model whose activities are not in canonical order or
// name a region or commodity out of range (programmer invariants) and returns
// ErrBadProgramme for a fixed term with no goods row or a bad denominator.
func checkModel(m Model) error {
	if len(m.Fixed) > m.Commodities {
		return ErrBadProgramme
	}
	for _, f := range m.Fixed {
		if f.Den <= 0 {
			return ErrBadProgramme
		}
	}
	prev := 0
	for _, a := range m.Activities {
		if a.Region < prev {
			panic("steady: invariant violated: activities not in canonical order")
		}
		if a.Region >= len(m.Regions) {
			panic("steady: invariant violated: activity region out of range")
		}
		prev = a.Region
	}
	if m.Phi.Den <= 0 || m.Year < 1 {
		return ErrBadProgramme
	}
	return nil
}

// Build turns a model into the programme of spec-viability.md 3.2: maximise
// lambda subject to the goods rows  sum x n_c - lambda D_c >= 0 (written as
// -n x + D lambda <= 0), the labour rows  sum d x <= phi H  and the capacity
// rows  x <= H J / TicksPerYear. The columns are the activities in the order
// given, then lambda. A fixed term W(c) writes the goods row as  -n x + D lambda
// <= -W(c)  (N-277).
func Build(m Model) (Programme, Layout, error) {
	if err := checkModel(m); err != nil {
		return Programme{}, Layout{}, err
	}
	l := Layout{Commodities: m.Commodities, Regions: len(m.Regions), Lambda: len(m.Activities)}
	rows, _ := Size(m)
	p := Programme{Rows: rows, RHS: make([]Ratio, rows)}
	for i := range p.RHS {
		p.RHS[i] = Ratio{Num: 0, Den: 1}
	}
	if err := writeFixed(p.RHS, m.Fixed); err != nil {
		return Programme{}, Layout{}, err
	}
	nextCap := m.Commodities + len(m.Regions)
	if err := writeLabour(p.RHS, m, l); err != nil {
		return Programme{}, Layout{}, err
	}
	for _, a := range m.Activities {
		col := activityColumn(a, l)
		l.CapRow = append(l.CapRow, -1)
		if hasCap(a, m.Year) {
			row := nextCap
			nextCap++
			v, ok := ratio(new(big.Int).Mul(big.NewInt(m.Regions[a.Region].Households), big.NewInt(a.Starts)), big.NewInt(m.Year))
			if !ok {
				return Programme{}, Layout{}, ErrOverflow
			}
			l.CapRow[len(l.CapRow)-1] = row
			p.RHS[row] = v
			col.Entries = append(col.Entries, Entry{Row: row, Val: Ratio{Num: 1, Den: 1}})
		}
		p.Cols = append(p.Cols, col)
	}
	lam, err := lambdaColumn(m)
	if err != nil {
		return Programme{}, Layout{}, err
	}
	p.Cols = append(p.Cols, lam)
	if err := checkProgramme(p); err != nil {
		return Programme{}, Layout{}, err // a commodity out of range
	}
	return p, l, nil
}

// writeLabour writes the labour row of each region of rhs: phi * H.
func writeLabour(rhs []Ratio, m Model, l Layout) error {
	for r, rg := range m.Regions {
		num := new(big.Int).Mul(big.NewInt(rg.Households), big.NewInt(m.Phi.Num))
		v, ok := ratio(num, big.NewInt(m.Phi.Den))
		if !ok {
			return ErrOverflow
		}
		rhs[l.LabourRow(r)] = v
	}
	return nil
}

// writeFixed writes the goods rows of rhs from the fixed terms: row c is -W(c).
func writeFixed(rhs, fixed []Ratio) error {
	for c, f := range fixed {
		v, ok := ratio(new(big.Int).Neg(big.NewInt(f.Num)), big.NewInt(f.Den))
		if !ok {
			return ErrOverflow
		}
		rhs[c] = v
	}
	return nil
}

// lambdaColumn builds the column of the surplus factor: cost 1 and the total
// upkeep D(c, P) in each goods row.
func lambdaColumn(m Model) (Column, error) {
	col := Column{Cost: Ratio{Num: 1, Den: 1}}
	for c := 0; c < m.Commodities; c++ {
		total := new(big.Int)
		for _, rg := range m.Regions {
			if c < len(rg.Upkeep) {
				total.Add(total, big.NewInt(rg.Upkeep[c]))
			}
		}
		if !total.IsInt64() {
			return Column{}, ErrOverflow
		}
		if total.Sign() != 0 {
			col.Entries = append(col.Entries, Entry{Row: c, Val: Ratio{Num: total.Int64(), Den: 1}})
		}
	}
	return col, nil
}

// activityColumn builds the column of an activity without its capacity entry:
// minus its net vector in the goods rows and its duration in its region's labour
// row.
func activityColumn(a Activity, l Layout) Column {
	col := Column{Cost: Ratio{Num: 0, Den: 1}}
	for _, n := range a.Net() {
		col.Entries = append(col.Entries, Entry{Row: n.Commodity, Wide: new(big.Rat).Neg(n.Val)})
	}
	col.Entries = append(col.Entries, Entry{Row: l.LabourRow(a.Region), Val: Ratio{Num: a.Duration, Den: 1}})
	return col
}
