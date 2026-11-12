package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/steady"
)

// This file is the payroll-aware steady state of spec-viability.md 5.6 (VIA,
// ADR 0012): with Modules.Employment on, the duals of a solved programme are
// solved again so that the net wage per working tick is equal across roles. The
// arithmetic is internal/steady/payroll.go; this file reads the payroll from the
// configuration, converts inline at the call (architecture.md 2.9) and writes the
// duals back before the table is made. The primal, the census and the flows are
// not touched. The call is one guarded line in solve (viability.go); with the
// module off this file is not reached.

// The reasons of PayrollRecord.Reason: the closed set of FR-VIA-069.
const (
	payrollUnsupported    = "unsupported"
	payrollNoUnique       = "no_unique_solution"
	payrollNegativeDual   = "negative_dual"
	payrollInactiveProfit = "inactive_profit"
)

// PayrollRecord names a region whose programme kept its payroll-free duals under
// the employment module, and why (spec-viability.md 10.3).
type PayrollRecord struct {
	// Region is the ID of the region.
	Region string
	// Reason is one of "unsupported" (lanes, a nominal seed, or a recipe shared
	// with another role), "no_unique_solution", "negative_dual" and
	// "inactive_profit".
	Reason string
}

// payrollRows is what a payroll-aware programme keeps for its report: by group
// (the groups of every region of the programme, in region then Groups order) the
// pay paid, the pay received and the labour, in the units of the duals at the
// flows of lambda 1, the first group of each region, and by activity of the
// programme the pay per execution that the net-wage equation moves (positive for
// a payer, negative for a worker).
type payrollRows struct {
	base    []int
	out, in []steady.Frac
	labour  []steady.Frac
	perExec []steady.Frac
}

// payrollCtx is the input of the solve and what the root needs to read the answer.
type payrollCtx struct {
	in     steady.PayrollInput
	base   []int // the first group of each region of the programme
	shared bool  // an active recipe of a payer or Hireable group is also listed by another group
	over   bool  // a group's own consumption does not fit int64
}

// payroll re-solves the duals of programme p for the payroll of the
// configuration, when a region has a payer and a Hireable group. Nothing it does
// changes a flow, the census or the number of regions.
func (s *viaSolve) payroll(p *viaProgramme) {
	pc := s.payrollInput(p)
	regions := payrollRegions(p, pc.in)
	if len(regions) == 0 {
		return // no region has both a payer and a Hireable group: nothing is paid
	}
	if len(p.md.Lanes) > 0 || len(p.nominalSeed) > 0 || pc.shared || pc.over {
		s.payrollFailed(p, regions, payrollUnsupported)
		return
	}
	res := steady.PayrollSolve(p.md, p.pr, pc.in)
	switch res.Reason {
	case steady.PayrollOK:
	case steady.PayrollNoUniqueSolution:
		s.payrollFailed(p, regions, payrollNoUnique)
		return
	case steady.PayrollNegativeDual:
		s.payrollFailed(p, regions, payrollNegativeDual)
		return
	case steady.PayrollInactiveProfit:
		s.payrollFailed(p, regions, payrollInactiveProfit)
		return
	default:
		s.payrollFailed(p, regions, payrollUnsupported)
		return
	}
	if !res.Applied {
		return
	}
	p.out, p.pr = res.Outcome, res.Outcome
	p.pay = &payrollRows{base: pc.base, out: res.Out, in: res.In, labour: res.Labour, perExec: payrollPerExecution(p, pc.in, res)}
}

// payrollFailed lists the regions with the reason.
func (s *viaSolve) payrollFailed(p *viaProgramme, regions []int, reason string) {
	for _, k := range regions {
		s.rep.PayrollInfeasible = append(s.rep.PayrollInfeasible, PayrollRecord{Region: s.m.c.Regions[p.regions[k]].ID, Reason: reason})
	}
}

// payrollInput builds the groups of every region of the programme, in region
// then Groups order, with their share, Hireable flag and own consumption, and
// the group of each activity (-1 for an activity of no payer and no Hireable
// group).
func (s *viaSolve) payrollInput(p *viaProgramme) payrollCtx {
	c := s.m.c
	var pc payrollCtx
	var roles []*roleRow // by group
	var first []int      // by region: its first group
	for k, r := range p.regions {
		first = append(first, len(pc.in.Groups))
		for gi := range c.Groups {
			g := &c.Groups[gi]
			if gr, _ := c.regionID(g.Region); gr != r {
				continue
			}
			ro, _ := c.roleID(g.Role)
			role, row := &c.Roles[ro], s.m.roles.row(ro)
			pg := steady.PayrollGroup{Region: k, Hireable: role.Hireable, Own: make([]int64, len(c.Commodities))}
			if role.Employer != nil {
				pg.BP = role.Employer.PayrollBP
			}
			for _, it := range row.upkeepIn(r) {
				own, ok := checked.Mul(int64(it.quantity), g.Households)
				pc.over = pc.over || !ok
				pg.Own[it.commodity] = own
			}
			pc.in.Groups = append(pc.in.Groups, pg)
			roles = append(roles, row)
		}
	}
	pc.base = first
	pc.in.Group = make([]int, len(p.md.Activities))
	for k, r := range p.regions {
		end := len(pc.in.Groups)
		if k+1 < len(first) {
			end = first[k+1]
		}
		for i, a := range s.m.acts[r] {
			at := p.base[k] + i
			pc.in.Group[at] = -1
			listing := 0
			for g := first[k]; g < end; g++ {
				if !slices.Contains(roles[g].recipes, a.recipe) {
					continue
				}
				listing++
				if pc.in.Group[at] < 0 && (pc.in.Groups[g].BP > 0 || pc.in.Groups[g].Hireable) {
					pc.in.Group[at] = g
				}
			}
			if listing > 1 && pc.in.Group[at] >= 0 && p.out.Active[at] {
				pc.shared = true
			}
		}
	}
	return pc
}

// payrollRegions lists the regions of the programme that have a payer and a
// Hireable group.
func payrollRegions(p *viaProgramme, in steady.PayrollInput) []int {
	var out []int
	for k := range p.regions {
		var payer, worker bool
		for _, g := range in.Groups {
			if g.Region != k {
				continue
			}
			payer = payer || g.BP > 0
			worker = worker || g.Hireable
		}
		if payer && worker {
			out = append(out, k)
		}
	}
	return out
}

// payrollPerExecution returns, by activity of the programme, the pay per
// execution that moves the net-wage equation: d * Out / Labour of a payer's
// group, minus d * In / Labour for a worker's, 0 for the rest.
func payrollPerExecution(p *viaProgramme, in steady.PayrollInput, res steady.PayrollResult) []steady.Frac {
	out := make([]steady.Frac, len(p.md.Activities))
	for at, a := range p.md.Activities {
		out[at] = steady.Rat(0, 1)
		g := in.Group[at]
		if g < 0 || res.Labour[g].Sign() == 0 {
			continue
		}
		switch {
		case in.Groups[g].BP > 0:
			per, _ := res.Out[g].Quo(res.Labour[g])
			out[at] = per.Mul(a.Duration, 1)
		case in.Groups[g].Hireable:
			per, _ := res.In[g].Quo(res.Labour[g])
			out[at] = per.Mul(-a.Duration, 1)
		}
	}
	return out
}

// payrollBP returns the pay a household of group gi of the region pays and
// receives each tick, in basis points of its upkeep cost (CensusRow.PayrollOutBP
// and PayrollInBP): the pay of the group times the price scale, over its
// households times the upkeep cost of one at the seed prices, rounded half to
// even. Both are 0 where the programme kept its payroll-free duals or the figure
// does not fit int64.
func (rc *regionCtx) payrollBP(gi int) (outBP, inBP int64) {
	pay := rc.p.pay
	if pay == nil {
		return 0, 0
	}
	g := pay.base[rc.k] + gi
	gc := &rc.groups[gi]
	upkeep, ok := rc.upkeepCost(gc)
	den, ok2 := checked.Mul(gc.g.Households, upkeep)
	if !ok || !ok2 || den < 1 {
		return 0, 0
	}
	bp := func(f steady.Frac) int64 {
		v, fits := f.Times(rc.scale).Mul(wholeBP, den).Round(0, maxInt64)
		if !fits {
			return 0
		}
		return v
	}
	return bp(pay.out[g]), bp(pay.in[g])
}
