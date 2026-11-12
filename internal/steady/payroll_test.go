package steady_test

import (
	"context"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// The toy of spec-viability.md 5.6, worked in closed form. One region of 10
// households, commodities bread (0) and flour (1), D = 2 bread a tick, a year of
// 365 ticks and no capacity row (Starts * Duration = Year):
//
//	bake: d = 1, inputs 1 flour, outputs 2 bread   (the payer group)
//	mill: d = 1, outputs 1 flour                   (the Hireable group)
//
// At lambda 1 both run once (2 x(bake) = 2 lambda, x(mill) = x(bake)), the
// labour row binds (2 lambda <= 10, lambda* = 5), and without payroll the duals
// are p(bread) = 1/2 (2 p = 1), p(flour) = w = 1/2. With beta the payers' share
// and no own consumption, sales are 2 bread, S = 2 p(bread) = 1, L(mill) = 1 and
// sigma = beta S / L(mill) = beta, so the equations
//
//	bake: 2/2 - p(flour) - beta = w        mill: p(flour) + beta = w
//
// give w = 1/2 and p(flour) = 1/2 - beta, 0 at 5 000 bp, negative above.
func toyModel(extra ...steady.Activity) steady.Model {
	const year, factor = 365, 365 * 10_000
	acts := []steady.Activity{
		{Region: 0, Duration: 1, Starts: year, FactorSum: factor,
			Inputs:  []steady.Item{{Commodity: 1, Qty: 1}},
			Outputs: []steady.Item{{Commodity: 0, Qty: 2}}},
		{Region: 0, Duration: 1, Starts: year, FactorSum: factor,
			Outputs: []steady.Item{{Commodity: 1, Qty: 1}}},
	}
	return steady.Model{
		Commodities: 3, Year: year, Phi: steady.Ratio{Num: 1, Den: 1},
		Regions:    []steady.Region{{Households: 10, Upkeep: []int64{2, 0, 0}}},
		Activities: append(acts, extra...),
	}
}

func solveToy(t *testing.T, m steady.Model) steady.Outcome {
	t.Helper()
	o, err := steady.SolveModel(context.Background(), m, 10_000)
	if err != nil {
		t.Fatalf("SolveModel(toy) error = %v, want nil", err)
	}
	return o
}

func toyInput(bp int64, nActs int) steady.PayrollInput {
	group := make([]int, nActs)
	for i := range group {
		group[i] = -1
	}
	group[0], group[1] = 0, 1
	return steady.PayrollInput{
		Groups: []steady.PayrollGroup{{Region: 0, BP: bp}, {Region: 0, Hireable: true}},
		Group:  group,
	}
}

func eq(f steady.Frac, n, d int64) bool { return f.Cmp(steady.Rat(n, d)) == 0 }

// TestPayrollSolve_ToyClosedForm verifies the duals of the toy of the comment
// above for several shares: the wage stays 1/2, p(flour) = 1/2 - beta, the
// payers pay beta and the workers receive beta.
//
// Requirement: FR-VIA-064, FR-VIA-065, FR-VIA-067
func TestPayrollSolve_ToyClosedForm(t *testing.T) {
	t.Parallel()
	m := toyModel()
	base := solveToy(t, m)
	tests := []struct {
		name   string
		bp     int64
		flourN int64 // p(flour) = flourN / 20 000
	}{
		{"quarter share", 2_500, 5_000}, // 1/2 - 1/4 = 1/4 = 5000/20000
		{"tenth share", 1_000, 8_000},   // 1/2 - 1/10 = 2/5 = 8000/20000
		{"half share gives a free good", 5_000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := steady.PayrollSolve(m, base, toyInput(tt.bp, 2))
			if !res.Applied || res.Reason != steady.PayrollOK {
				t.Fatalf("PayrollSolve(bp %d) applied = %v reason = %v, want applied", tt.bp, res.Applied, res.Reason)
			}
			o := res.Outcome
			if !eq(o.Price[0], 1, 2) || !eq(o.Price[1], tt.flourN, 20_000) || !eq(o.Wage[0], 1, 2) {
				t.Errorf("PayrollSolve(bp %d) p(bread) = %v/%v p(flour) = %v/%v w = %v/%v, want 1/2, %d/20000, 1/2",
					tt.bp, o.Price[0].Num, o.Price[0].Den, o.Price[1].Num, o.Price[1].Den, o.Wage[0].Num, o.Wage[0].Den, tt.flourN)
			}
			beta := steady.Rat(tt.bp, 10_000)
			if res.Out[0].Cmp(beta) != 0 || res.In[1].Cmp(beta) != 0 {
				t.Errorf("PayrollSolve(bp %d) out = %v, in = %v, want both beta %v", tt.bp, res.Out[0].Num, res.In[1].Num, beta.Num)
			}
			if res.Out[1].Sign() != 0 || res.In[0].Sign() != 0 {
				t.Errorf("PayrollSolve(bp %d) worker pays or payer receives, want 0", tt.bp)
			}
			if o.Lambda.Cmp(base.Lambda) != 0 || o.SurplusBP != base.SurplusBP || o.Flow[0].Cmp(base.Flow[0]) != 0 {
				t.Errorf("PayrollSolve(bp %d) changed the primal (lambda %v, flow %v)", tt.bp, o.Lambda.Num, o.Flow[0].Num)
			}
		})
	}
}

// TestPayrollSolve_IdentityAtZero verifies that a share of 0, a region with no
// worker and a region with no payer return the payroll-free outcome untouched.
//
// Requirement: FR-VIA-068
func TestPayrollSolve_IdentityAtZero(t *testing.T) {
	t.Parallel()
	m := toyModel()
	base := solveToy(t, m)
	noWorker := toyInput(2_500, 2)
	noWorker.Groups[1].Hireable = false
	noPayer := toyInput(0, 2)
	for name, in := range map[string]steady.PayrollInput{
		"share 0": toyInput(0, 2), "no worker": noWorker, "no payer": noPayer,
	} {
		res := steady.PayrollSolve(m, base, in)
		if res.Applied || res.Reason != steady.PayrollOK {
			t.Errorf("PayrollSolve(%s) applied = %v reason = %v, want the identity", name, res.Applied, res.Reason)
		}
		for c := range base.Price {
			if res.Outcome.Price[c].Cmp(base.Price[c]) != 0 {
				t.Errorf("PayrollSolve(%s) changed price %d", name, c)
			}
		}
		if res.Out[0].Sign() != 0 || res.In[1].Sign() != 0 {
			t.Errorf("PayrollSolve(%s) paid something, want 0", name)
		}
	}
}

// TestPayrollSolve_Reasons verifies the typed reasons: a share too high for the
// chain (negative price), an inactive activity that the payroll makes
// profitable, a system with an unused non-zero dual (no unique solution), and
// the unsupported shapes.
//
// Requirement: FR-VIA-067, FR-VIA-069
func TestPayrollSolve_Reasons(t *testing.T) {
	t.Parallel()
	m := toyModel()
	base := solveToy(t, m)
	t.Run("share too high", func(t *testing.T) {
		t.Parallel()
		// p(flour) = 1/2 - beta is negative from 5 001 bp.
		res := steady.PayrollSolve(m, base, toyInput(5_001, 2))
		if res.Applied || res.Reason != steady.PayrollNegativeDual {
			t.Errorf("PayrollSolve(5001) applied = %v reason = %v, want PayrollNegativeDual", res.Applied, res.Reason)
		}
		if res.Outcome.Price[1].Cmp(base.Price[1]) != 0 {
			t.Errorf("PayrollSolve(5001) changed the duals of a rejected solution")
		}
	})
	t.Run("inactive activity becomes profitable", func(t *testing.T) {
		t.Parallel()
		// turn: d = 1, 1 flour in, 1.15 bread out (23 bread at a yield factor of
		// 500 bp). Without payroll v = 1.15/2 - 1/2 = 0.075 <= w d = 1/2, so it is
		// inactive; at 5 000 bp p(flour) = 0 and v = 0.575 > 1/2.
		turn := steady.Activity{Region: 0, Duration: 1, Starts: 365, FactorSum: 365 * 500,
			Inputs:  []steady.Item{{Commodity: 1, Qty: 1}},
			Outputs: []steady.Item{{Commodity: 0, Qty: 23}}}
		mm := toyModel(turn)
		o := solveToy(t, mm)
		if o.Active[2] {
			t.Fatalf("turn is active without payroll, want inactive")
		}
		in := toyInput(5_000, 3)
		res := steady.PayrollSolve(mm, o, in)
		if res.Applied || res.Reason != steady.PayrollInactiveProfit {
			t.Errorf("PayrollSolve(turn, 5000) applied = %v reason = %v, want PayrollInactiveProfit", res.Applied, res.Reason)
		}
		if res := steady.PayrollSolve(mm, o, toyInput(2_500, 3)); !res.Applied {
			t.Errorf("PayrollSolve(turn, 2500) reason = %v, want applied: 1.15/2 - 1/4 < 1/2", res.Reason)
		}
	})
	t.Run("unused non-zero dual", func(t *testing.T) {
		t.Parallel()
		o := base
		o.Price = append([]steady.Frac(nil), base.Price...)
		o.Price[2] = steady.Rat(1, 7) // no activity touches commodity 2: one unknown too many
		res := steady.PayrollSolve(m, o, toyInput(2_500, 2))
		if res.Applied || res.Reason != steady.PayrollNoUniqueSolution {
			t.Errorf("PayrollSolve(extra dual) applied = %v reason = %v, want PayrollNoUniqueSolution", res.Applied, res.Reason)
		}
	})
	t.Run("lanes", func(t *testing.T) {
		t.Parallel()
		ml := m
		ml.Lanes = []steady.Lane{{From: 0, To: 0, Commodity: 0}}
		res := steady.PayrollSolve(ml, base, toyInput(2_500, 2))
		if res.Applied || res.Reason != steady.PayrollUnsupported {
			t.Errorf("PayrollSolve(lanes) applied = %v reason = %v, want PayrollUnsupported", res.Applied, res.Reason)
		}
	})
	t.Run("payer and hireable", func(t *testing.T) {
		t.Parallel()
		in := toyInput(2_500, 2)
		in.Groups[0].Hireable = true
		if res := steady.PayrollSolve(m, base, in); res.Reason != steady.PayrollUnsupported {
			t.Errorf("PayrollSolve(both) reason = %v, want PayrollUnsupported", res.Reason)
		}
	})
	t.Run("share above 10000", func(t *testing.T) {
		t.Parallel()
		if res := steady.PayrollSolve(m, base, toyInput(10_001, 2)); res.Reason != steady.PayrollUnsupported {
			t.Errorf("PayrollSolve(10001) reason = %v, want PayrollUnsupported", res.Reason)
		}
	})
	t.Run("group list of the wrong length panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if recover() == nil {
				t.Errorf("PayrollSolve(short group list) did not panic")
			}
		}()
		steady.PayrollSolve(m, base, toyInput(2_500, 3))
	})
}

// TestPayrollSolve_OwnConsumption verifies that the sales of a payer are its net
// output less what its own households eat: with the baker group eating 1 of the
// 2 bread, S halves, sigma = beta / 2 and p(flour) = 1/2 - beta / 2.
//
// Requirement: FR-VIA-066
func TestPayrollSolve_OwnConsumption(t *testing.T) {
	t.Parallel()
	m := toyModel()
	base := solveToy(t, m)
	in := toyInput(5_000, 2)
	in.Groups[0].Own = []int64{1}
	res := steady.PayrollSolve(m, base, in)
	// sold = 2 - 1 = 1, S = 1/2, sigma = beta S = 1/4;
	// bake: 1 - p(flour) - 1/4 = w, mill: p(flour) + 1/4 = w -> p(flour) = 1/4, w = 1/2.
	if !res.Applied || !eq(res.Outcome.Price[1], 1, 4) || !eq(res.Outcome.Wage[0], 1, 2) {
		t.Fatalf("PayrollSolve(own 1) applied = %v p(flour) = %v/%v, want 1/4", res.Applied, res.Outcome.Price[1].Num, res.Outcome.Price[1].Den)
	}
	if !eq(res.Out[0], 1, 4) || !eq(res.In[1], 1, 4) {
		t.Errorf("PayrollSolve(own 1) out = %v in = %v, want 1/4", res.Out[0].Num, res.In[1].Num)
	}
	in.Groups[0].Own = []int64{5} // more than it makes: sales floor at 0, nothing is paid
	if r2 := steady.PayrollSolve(m, base, in); !r2.Applied || r2.Out[0].Sign() != 0 || !eq(r2.Outcome.Price[1], 1, 2) {
		t.Errorf("PayrollSolve(own 5) out = %v p(flour) = %v/%v, want 0 and 1/2", r2.Out[0].Num, r2.Outcome.Price[1].Num, r2.Outcome.Price[1].Den)
	}
}

// verify recomputes the equations of spec-viability.md 5.6 from the result alone
// and reports the first that fails: every active activity's value added, less the
// payer's burden or plus the worker's pay, equals d w; no dual is negative;
// payers pay what workers receive.
func verify(m steady.Model, res steady.PayrollResult, in steady.PayrollInput) string {
	o := res.Outcome
	rat := func(f steady.Frac) *big.Rat { return new(big.Rat).SetFrac(f.Num, f.Den) }
	for _, p := range o.Price {
		if p.Sign() < 0 {
			return "negative price"
		}
	}
	// receipts and labour by group at the new prices
	lab := make([]*big.Rat, len(in.Groups))
	rec := make([]*big.Rat, len(in.Groups))
	for g := range lab {
		lab[g], rec[g] = new(big.Rat), new(big.Rat)
	}
	sold := make([][]*big.Rat, len(in.Groups))
	for g := range sold {
		sold[g] = make([]*big.Rat, m.Commodities)
		for c := range sold[g] {
			sold[g][c] = new(big.Rat)
		}
	}
	for k, a := range m.Activities {
		g := in.Group[k]
		if g < 0 {
			continue
		}
		f := rat(o.Flow[k])
		lab[g].Add(lab[g], new(big.Rat).Mul(f, big.NewRat(a.Duration, 1)))
		for _, n := range a.Net() {
			sold[g][n.Commodity].Add(sold[g][n.Commodity], new(big.Rat).Mul(f, n.Val))
		}
	}
	pool := new(big.Rat)
	paid := new(big.Rat)
	for g, gr := range in.Groups {
		if gr.Hireable {
			pool.Add(pool, lab[g])
		}
		for c, s := range sold[g] {
			if c < len(gr.Own) {
				s.Sub(s, big.NewRat(gr.Own[c], 1))
			}
			if s.Sign() > 0 {
				rec[g].Add(rec[g], new(big.Rat).Mul(s, rat(o.Price[c])))
			}
		}
		if gr.BP > 0 {
			paid.Add(paid, new(big.Rat).Mul(big.NewRat(gr.BP, 10_000), rec[g]))
		}
	}
	sigma := new(big.Rat)
	if pool.Sign() > 0 && paid.Sign() > 0 {
		sigma.Quo(paid, pool)
	}
	for k, a := range m.Activities {
		if !o.Active[k] {
			continue
		}
		v := new(big.Rat)
		for _, n := range a.Net() {
			v.Add(v, new(big.Rat).Mul(n.Val, rat(o.Price[n.Commodity])))
		}
		d := big.NewRat(a.Duration, 1)
		if g := in.Group[k]; g >= 0 && pool.Sign() > 0 {
			switch {
			case in.Groups[g].BP > 0:
				v.Sub(v, new(big.Rat).Mul(d, new(big.Rat).Quo(new(big.Rat).Mul(big.NewRat(in.Groups[g].BP, 10_000), rec[g]), lab[g])))
			case in.Groups[g].Hireable:
				v.Add(v, new(big.Rat).Mul(d, sigma))
			}
		}
		if v.Cmp(new(big.Rat).Add(new(big.Rat).Mul(d, rat(o.Wage[0])), rat(o.Rent[k]))) != 0 {
			return "an active activity does not pay the net wage"
		}
	}
	var out, got steady.Frac
	out, got = steady.Rat(0, 1), steady.Rat(0, 1)
	for g := range in.Groups {
		out, got = out.Add(res.Out[g]), got.Add(res.In[g])
	}
	if out.Cmp(got) != 0 {
		return "payroll paid differs from payroll received"
	}
	return ""
}

// payrollChain builds a chain of n+1 commodities (0 is the basket good, each
// activity i makes commodity i from commodity i+1, the last from nothing) with
// random yields, durations and shares, and the groups: activity i is a payer,
// a worker or neither at random. It is a family of models whose payroll-free
// outcome is a clean vertex.
func payrollChain(rng *rand.Rand) (steady.Model, steady.PayrollInput) {
	n := 2 + rng.IntN(3)
	acts := make([]steady.Activity, n)
	in := steady.PayrollInput{Group: make([]int, n)}
	for i := range acts {
		starts := int64(365 - 165*rng.IntN(2)) // 200 starts a year give a capacity row (and a rent)
		a := steady.Activity{Region: 0, Duration: int64(1 + rng.IntN(3)), Starts: starts, FactorSum: starts * 10_000,
			Outputs: []steady.Item{{Commodity: i, Qty: int64(2 + rng.IntN(4))}}}
		if i+1 < n {
			a.Inputs = []steady.Item{{Commodity: i + 1, Qty: 1}}
		}
		acts[i] = a
		in.Group[i] = len(in.Groups)
		switch rng.IntN(3) {
		case 0:
			in.Groups = append(in.Groups, steady.PayrollGroup{Region: 0, BP: int64(1 + rng.IntN(6_000))})
		case 1:
			in.Groups = append(in.Groups, steady.PayrollGroup{Region: 0, Hireable: true})
		default:
			in.Groups = append(in.Groups, steady.PayrollGroup{Region: 0})
			in.Group[i] = -1
			in.Groups = in.Groups[:len(in.Groups)-1]
		}
	}
	m := steady.Model{
		Commodities: n, Year: 365, Phi: steady.Ratio{Num: 1, Den: 1},
		Regions:    []steady.Region{{Households: 50, Upkeep: append([]int64{int64(1 + rng.IntN(5))}, make([]int64, n-1)...)}},
		Activities: acts,
	}
	return m, in
}

// TestPayrollSolve_PropertyRandomChains verifies, over random chains with fixed
// seeds, that whenever PayrollSolve applies a solution it satisfies the net-wage
// equations exactly (recomputed independently in verify), keeps the primal, and
// that payroll paid equals payroll received. The failing seed is printed.
//
// Requirement: FR-VIA-064, FR-VIA-067
func TestPayrollSolve_PropertyRandomChains(t *testing.T) {
	t.Parallel()
	applied := 0
	for seed := uint64(1); seed <= 300; seed++ {
		rng := rand.New(rand.NewPCG(seed, 99))
		m, in := payrollChain(rng)
		o, err := steady.SolveModel(context.Background(), m, 100_000)
		if err != nil {
			continue
		}
		res := steady.PayrollSolve(m, o, in)
		if !res.Applied {
			continue
		}
		applied++
		if msg := verify(m, res, in); msg != "" {
			t.Fatalf("seed %d: %s", seed, msg)
		}
		if res.Outcome.Lambda.Cmp(o.Lambda) != 0 {
			t.Fatalf("seed %d: lambda changed", seed)
		}
	}
	if applied < 20 {
		t.Errorf("only %d of 300 random chains applied, want at least 20: the property test is not exercising the solver", applied)
	}
}

// FuzzPayrollSolve verifies that PayrollSolve never panics on a valid input and
// that an applied solution satisfies the equations exactly.
//
// Requirement: NFR-VIA-005
func FuzzPayrollSolve(f *testing.F) {
	for _, s := range []uint64{1, 2, 3, 7, 42} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, seed uint64) {
		rng := rand.New(rand.NewPCG(seed, 5))
		m, in := payrollChain(rng)
		o, err := steady.SolveModel(context.Background(), m, 100_000)
		if err != nil {
			return
		}
		res := steady.PayrollSolve(m, o, in)
		if res.Applied {
			if msg := verify(m, res, in); msg != "" {
				t.Fatalf("seed %d: %s", seed, msg)
			}
		}
	})
}

// swapped is the toy with the commodities swapped (flour 0, bread 1) and a
// shorter upkeep slice, so the basket good is not the first unknown and the
// normalisation reads a slice shorter than the commodity count.
func swapped() steady.Model {
	m := toyModel()
	m.Activities[0].Inputs = []steady.Item{{Commodity: 0, Qty: 1}}
	m.Activities[0].Outputs = []steady.Item{{Commodity: 1, Qty: 2}}
	m.Activities[1].Outputs = []steady.Item{{Commodity: 0, Qty: 1}}
	m.Regions[0].Upkeep = []int64{0, 2}
	return m
}

// TestPayrollSolve_SwappedCommodities verifies the toy with the basket good
// after the intermediate one in commodity order: the same duals, by commodity.
//
// Requirement: FR-VIA-065
func TestPayrollSolve_SwappedCommodities(t *testing.T) {
	t.Parallel()
	m := swapped()
	res := steady.PayrollSolve(m, solveToy(t, m), toyInput(2_500, 2))
	if !res.Applied || !eq(res.Outcome.Price[1], 1, 2) || !eq(res.Outcome.Price[0], 1, 4) {
		t.Errorf("PayrollSolve(swapped, 2500) applied = %v p = %v/%v and %v/%v, want p(flour) 1/4 and p(bread) 1/2",
			res.Applied, res.Outcome.Price[0].Num, res.Outcome.Price[0].Den, res.Outcome.Price[1].Num, res.Outcome.Price[1].Den)
	}
}

// TestPayrollSolve_CapacityRent verifies a tight capacity row: the mill can start
// 100 times a year, so it is capacity bound (about 2.74 a tick), labour is slack
// (w = 0) and the scarcity shows as a rent. Worked: p(bread) = 1/2; bake (payer):
// 2 p(bread) - p(flour) - beta = 0; mill (worker): p(flour) + beta = rent. So
// p(flour) = 1 - beta and the rent is 1 whatever the share.
//
// Requirement: FR-VIA-064, FR-VIA-065
func TestPayrollSolve_CapacityRent(t *testing.T) {
	t.Parallel()
	m := toyModel()
	m.Activities[1].Starts, m.Activities[1].FactorSum = 100, 100*10_000
	base := solveToy(t, m)
	if base.Wage[0].Sign() != 0 || base.Rent[1].Sign() <= 0 {
		t.Fatalf("toy with a capacity row: wage %v rent %v, want wage 0 and a positive rent", base.Wage[0].Num, base.Rent[1].Num)
	}
	res := steady.PayrollSolve(m, base, toyInput(2_500, 2))
	if !res.Applied || !eq(res.Outcome.Price[1], 3, 4) || !eq(res.Outcome.Rent[1], 1, 1) || res.Outcome.Wage[0].Sign() != 0 {
		t.Errorf("PayrollSolve(capacity) applied = %v p(flour) = %v/%v rent = %v/%v, want 3/4 and 1",
			res.Applied, res.Outcome.Price[1].Num, res.Outcome.Price[1].Den, res.Outcome.Rent[1].Num, res.Outcome.Rent[1].Den)
	}
}

// TestPayrollSolve_ResourceRent verifies a tight resource row: the mill draws one
// unit of a resource a start and the stock lets 1 a tick be drawn (a constant
// stock of 365 over one year), so the mill runs once a tick, labour is slack and
// the draw has a rent. Worked as the capacity case: p(flour) = 1 - beta and the
// rent of the draw is 1.
//
// Requirement: FR-VIA-064, FR-VIA-065
func TestPayrollSolve_ResourceRent(t *testing.T) {
	t.Parallel()
	m := toyModel()
	m.Activities[1].Draws = []steady.Draw{{Resource: 0, Qty: 1}}
	m.Activities[1].AbundanceBP = 10_000
	m.Resources = []steady.Endow{{Resource: 0, Region: 0, Shape: steady.Constant, Stock: 365, Horizon: 1, ScaleBP: 10_000}}
	base := solveToy(t, m)
	if base.Wage[0].Sign() != 0 || base.ResRent[0].Sign() <= 0 {
		t.Fatalf("toy with a resource row: wage %v resource rent %v, want wage 0 and a positive rent", base.Wage[0].Num, base.ResRent[0].Num)
	}
	res := steady.PayrollSolve(m, base, toyInput(2_500, 2))
	if !res.Applied || !eq(res.Outcome.Price[1], 3, 4) || !eq(res.Outcome.ResRent[0], 1, 1) {
		t.Errorf("PayrollSolve(resource) applied = %v p(flour) = %v/%v resource rent = %v/%v, want 3/4 and 1",
			res.Applied, res.Outcome.Price[1].Num, res.Outcome.Price[1].Den, res.Outcome.ResRent[0].Num, res.Outcome.ResRent[0].Den)
	}
}

// TestPayrollSolve_InactiveEdges verifies the edges of the inactive check: an
// inactive activity that earns exactly the net wage is accepted (at 5 000 bp
// p(flour) is 0, so `turn` with one bread out earns 1/2 = w d), and a payer group
// with no active activity pays nothing and divides nothing.
//
// Requirement: FR-VIA-067
func TestPayrollSolve_InactiveEdges(t *testing.T) {
	t.Parallel()
	turn := steady.Activity{Region: 0, Duration: 1, Starts: 365, FactorSum: 365 * 10_000,
		Inputs:  []steady.Item{{Commodity: 1, Qty: 1}},
		Outputs: []steady.Item{{Commodity: 0, Qty: 1}}}
	m := toyModel(turn)
	o := solveToy(t, m)
	if o.Active[2] {
		t.Fatalf("turn is active without payroll, want inactive")
	}
	if res := steady.PayrollSolve(m, o, toyInput(5_000, 3)); !res.Applied {
		t.Errorf("PayrollSolve(turn at exactly the net wage) reason = %v, want applied", res.Reason)
	}
	// turn belongs to a second payer group that has no active activity.
	in := toyInput(2_500, 3)
	in.Groups = append(in.Groups, steady.PayrollGroup{Region: 0, BP: 1_000})
	in.Group[2] = 2
	if res := steady.PayrollSolve(m, o, in); !res.Applied || res.Out[2].Sign() != 0 || res.Labour[2].Sign() != 0 {
		t.Errorf("PayrollSolve(idle payer group) applied = %v out = %v, want applied and 0", res.Applied, res.Out[2].Num)
	}
	// An activity marked active that does not pay the net wage makes the system
	// inconsistent: four equations, three unknowns, no solution.
	bad := o
	bad.Active = []bool{true, true, true}
	if res := steady.PayrollSolve(m, bad, toyInput(2_500, 3)); res.Applied || res.Reason != steady.PayrollNoUniqueSolution {
		t.Errorf("PayrollSolve(inconsistent) applied = %v reason = %v, want PayrollNoUniqueSolution", res.Applied, res.Reason)
	}
}

// TestPayrollSolve_BadInputs verifies each input that is unsupported, one at a
// time, and that the boundary values are valid.
//
// Requirement: FR-VIA-069
func TestPayrollSolve_BadInputs(t *testing.T) {
	t.Parallel()
	m := toyModel()
	base := solveToy(t, m)
	tests := []struct {
		name string
		edit func(*steady.PayrollInput)
		want steady.PayrollReason
	}{
		{"region below 0", func(in *steady.PayrollInput) { in.Groups[0].Region = -1 }, steady.PayrollUnsupported},
		{"region past the end", func(in *steady.PayrollInput) { in.Groups[0].Region = 1 }, steady.PayrollUnsupported},
		{"negative share", func(in *steady.PayrollInput) { in.Groups[0].BP = -1 }, steady.PayrollUnsupported},
		{"share of 10 000 is valid", func(in *steady.PayrollInput) { in.Groups[0].BP = 10_000 }, steady.PayrollNegativeDual},
		{"activity group past the end", func(in *steady.PayrollInput) { in.Group[0] = 2 }, steady.PayrollUnsupported},
		{"activity group in another region", func(in *steady.PayrollInput) {
			in.Groups[1].Region = 0
			in.Group[0] = 1
			in.Groups = append(in.Groups, steady.PayrollGroup{Region: 0})
			in.Group[1] = 2
		}, steady.PayrollOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := toyInput(2_500, 2)
			tt.edit(&in)
			if res := steady.PayrollSolve(m, base, in); res.Reason != tt.want {
				t.Errorf("PayrollSolve(%s) reason = %v, want %v", tt.name, res.Reason, tt.want)
			}
		})
	}
	t.Run("an activity of a group of another region", func(t *testing.T) {
		t.Parallel()
		m2 := m
		m2.Regions = append(slicesClone(m.Regions), steady.Region{Households: 10, Upkeep: []int64{0, 0, 0}})
		in := toyInput(2_500, 2)
		in.Groups[1].Region = 1 // its activity (mill) is in region 0
		if res := steady.PayrollSolve(m2, base, in); res.Reason != steady.PayrollUnsupported {
			t.Errorf("PayrollSolve(group of region 1 runs a recipe of region 0) reason = %v, want PayrollUnsupported", res.Reason)
		}
	})
	t.Run("a one group list", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if recover() == nil {
				t.Errorf("PayrollSolve(one activity too few) did not panic")
			}
		}()
		steady.PayrollSolve(m, base, toyInput(2_500, 1))
	})
}

func slicesClone(r []steady.Region) []steady.Region { return append([]steady.Region(nil), r...) }

// TestPayrollSolve_TwoRegions verifies payroll is paid within a region: region 0
// is the toy; region 1 is a copy on its own commodities (3 bread, 4 flour, 5
// unused) whose mill is not Hireable, so nothing flows there and its payer pays
// nothing, and region 0's pay is not changed by region 1's sales. The sales of
// region 0 are 2 bread, S = 2 p(bread), out = beta S, and the toy's relation
// p(flour) = p(bread) - out holds.
//
// Requirement: FR-VIA-067
func TestPayrollSolve_TwoRegions(t *testing.T) {
	t.Parallel()
	m := toyModel()
	two := m
	two.Commodities = 6
	two.Regions = []steady.Region{{Households: 10, Upkeep: []int64{2, 0, 0, 0, 0, 0}}, {Households: 10, Upkeep: []int64{0, 0, 0, 2, 0, 0}}}
	bake, mill := m.Activities[0], m.Activities[1]
	bake.Region, mill.Region = 1, 1
	bake.Inputs = []steady.Item{{Commodity: 4, Qty: 1}}
	bake.Outputs = []steady.Item{{Commodity: 3, Qty: 2}}
	mill.Outputs = []steady.Item{{Commodity: 4, Qty: 1}}
	two.Activities = append(append([]steady.Activity(nil), m.Activities...), bake, mill)
	o := solveToy(t, two)
	in := steady.PayrollInput{
		Groups: []steady.PayrollGroup{{Region: 0, BP: 2_500}, {Region: 0, Hireable: true}, {Region: 1, BP: 2_500}, {Region: 1}},
		Group:  []int{0, 1, 2, 3},
	}
	res := steady.PayrollSolve(two, o, in)
	if !res.Applied {
		t.Fatalf("PayrollSolve(two regions) reason = %v, want applied", res.Reason)
	}
	if res.Out[2].Sign() != 0 || res.In[3].Sign() != 0 {
		t.Errorf("PayrollSolve(two regions) region 1 out = %v in = %v, want 0 and 0", res.Out[2].Num, res.In[3].Num)
	}
	// The two regions' duals are not unique (one normalisation couples them), so
	// the check is relative: out = beta * 2 * p(bread) and p(flour) = p(bread) - out.
	wantOut := steady.Rat(2_500, 10_000).Times(res.Outcome.Price[0]).Mul(2, 1)
	if res.Out[0].Cmp(wantOut) != 0 || res.Out[0].Cmp(res.In[1]) != 0 || res.Out[0].Sign() <= 0 {
		t.Errorf("PayrollSolve(two regions) region 0 out = %v/%v in = %v/%v, want both beta * 2 p(bread) > 0",
			res.Out[0].Num, res.Out[0].Den, res.In[1].Num, res.In[1].Den)
	}
	if wantFlour := res.Outcome.Price[0].Add(wantOut.Mul(-1, 1)); res.Outcome.Price[1].Cmp(wantFlour) != 0 {
		t.Errorf("PayrollSolve(two regions) p(flour) = %v/%v, want p(bread) - out", res.Outcome.Price[1].Num, res.Outcome.Price[1].Den)
	}
}
