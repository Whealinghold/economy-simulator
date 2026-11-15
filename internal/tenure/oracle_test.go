package tenure_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// oracleSeeds are the seeds of the randomised oracle tests; a failure prints the
// seed and the case index, so one call of caseGen reproduces it.
var oracleSeeds = []uint64{1, 2, 3, 4}

// oracleCases is the number of random cases per seed.
const oracleCases = 1500

// edges are the int64 values the oracle tests combine: zero, the unit, the
// basis-point scale, and the values where a product or a sum crosses MaxInt64.
var edges = []int64{0, 1, 2, 3, 9_999, 10_000, 10_001, math.MaxInt32, math.MaxInt64 / 4, math.MaxInt64 / 2, math.MaxInt64/2 + 1, math.MaxInt64 - 1, math.MaxInt64}

// caseGen draws non-negative int64 values, half from edges and half of a random
// bit length, so that sums and products land on both sides of MaxInt64.
type caseGen struct{ r *rand.Rand }

func newCaseGen(seed uint64) caseGen {
	return caseGen{rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))}
}

// val returns the next value.
func (g caseGen) val() int64 {
	if g.r.IntN(2) == 0 {
		return edges[g.r.IntN(len(edges))]
	}
	return g.r.Int64N(1<<uint(1+g.r.IntN(62))) + 0
}

// small returns a value below 2^bits, for counts a test wants to keep affordable.
func (g caseGen) small(bits uint) int64 { return g.r.Int64N(1 << bits) }

// bi is the big.Int of v.
func bi(v int64) *big.Int { return big.NewInt(v) }

// fits reports whether x is in int64 range.
func fits(x *big.Int) bool { return x.IsInt64() }

// exactTotal is the sum of Rent * Draw * max(0, k - Own - Commons) over the lines,
// computed without overflow.
func exactTotal(lines []tenure.Line, k int64) *big.Int {
	total := new(big.Int)
	for _, l := range lines {
		billed := new(big.Int).Sub(bi(k), bi(l.Own))
		billed.Sub(billed, bi(l.Commons))
		if billed.Sign() < 0 {
			billed.SetInt64(0)
		}
		billed.Mul(billed, bi(l.Draw)).Mul(billed, bi(l.Rent))
		total.Add(total, billed)
	}
	return total
}

// billedUnitsFit reports whether max(0, k - Own - Commons) * Draw fits int64 on
// every line: the intermediate product TotalRent forms before it multiplies by
// the rent.
func billedUnitsFit(lines []tenure.Line, k int64) bool {
	for _, l := range lines {
		billed := new(big.Int).Sub(bi(k), bi(l.Own))
		billed.Sub(billed, bi(l.Commons))
		if billed.Sign() < 0 {
			billed.SetInt64(0)
		}
		if !fits(billed.Mul(billed, bi(l.Draw))) {
			return false
		}
	}
	return true
}

// genLines returns one to three lines with fields from g.
func genLines(g caseGen) []tenure.Line {
	lines := make([]tenure.Line, 1+g.r.IntN(3))
	for i := range lines {
		lines[i] = tenure.Line{Rent: g.val(), Draw: g.val(), Own: g.val(), Commons: g.val()}
	}
	return lines
}

// totalMismatch returns how TotalRent's answer departs from the exact total, or
// "" when it does not.
func totalMismatch(lines []tenure.Line, k, got int64, ok bool) string {
	want := exactTotal(lines, k)
	switch {
	case ok && (!fits(want) || got != want.Int64()):
		return "want exact " + want.String()
	case !ok && fits(want) && billedUnitsFit(lines, k):
		return "refused though the exact total " + want.String() + " fits and no product overflows"
	case !ok && got != 0:
		return "want 0 on refusal"
	}
	return ""
}

// TestTotalRent_MatchesBigIntOracle verifies TotalRent against an arbitrary
// precision sum at the int64 edges: when it reports ok the value is the exact
// total, and when it refuses the exact total passes int64 or an intermediate
// product (billed executions times draw) does, so a wrapped or truncated rent
// is never returned.
//
// Requirement: FR-LND-016, NFR-LND-003
//
// Guard: the rent is charged as a ledger transfer; a wrapped total would charge
// a tenant a negative or a tiny rent and create or destroy money.
func TestTotalRent_MatchesBigIntOracle(t *testing.T) {
	t.Parallel()
	for _, seed := range oracleSeeds {
		g := newCaseGen(seed)
		for i := range oracleCases {
			lines, k := genLines(g), g.val()
			got, ok := tenure.TotalRent(lines, k)
			if bad := totalMismatch(lines, k, got, ok); bad != "" {
				t.Fatalf("seed %d case %d: TotalRent(%+v, %d) = %d, %v: %s", seed, i, lines, k, got, ok, bad)
			}
		}
	}
}

// exactCharge is the largest k in [0, other] whose exact total rent is at most
// available, found by bisection on the exact total (which is non-decreasing in
// k).
func exactCharge(lines []tenure.Line, available, other int64) int64 {
	lo, hi := int64(0), other
	for lo < hi {
		mid := lo + (hi-lo)/2 + (hi-lo)%2 // upper midpoint; hi-lo+1 would wrap at MaxInt64
		if exactTotal(lines, mid).Cmp(bi(available)) <= 0 {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// TestCharge_MatchesBigIntOracle verifies Charge returns the largest k whose
// exact rent the balance covers, and that rent, at the int64 edges, against an
// arbitrary precision bisection. Cases where an intermediate product of some k
// passes int64 are skipped: Charge treats such a k as unaffordable by contract.
//
// Requirement: FR-LND-016, NFR-LND-003
//
// Guard: k decides how many executions start; one too many spends rent the
// principal does not have, one too few refuses a start the principal can pay.
func TestCharge_MatchesBigIntOracle(t *testing.T) {
	t.Parallel()
	for _, seed := range oracleSeeds {
		g := newCaseGen(seed)
		for i := range oracleCases {
			lines, available, other := genLines(g), g.val(), g.val()
			if !billedUnitsFit(lines, other) {
				continue
			}
			k, rent, ok := tenure.Charge(available, other, lines)
			want := exactCharge(lines, available, other)
			if !ok || k != want {
				t.Fatalf("seed %d case %d: Charge(%d, %d, %+v) = %d, %d, %v, want k %d", seed, i, available, other, lines, k, rent, ok, want)
			}
			if exact := exactTotal(lines, k); !fits(exact) || rent != exact.Int64() || rent > available {
				t.Fatalf("seed %d case %d: Charge(%d, %d, %+v) rent %d, want exact %s within available", seed, i, available, other, lines, rent, exact)
			}
		}
	}
}

// exactRent is floor(posted * scale / 10 000), at least 1 when both are positive.
func exactRent(posted, scale int64) *big.Int {
	want := new(big.Int).Mul(bi(posted), bi(scale))
	want.Quo(want, bi(tenure.BasisPoints))
	if posted > 0 && scale > 0 && want.Sign() == 0 {
		want.SetInt64(1)
	}
	return want
}

// fitInt returns x when it fits int64, else 0 (what a refusal returns).
func fitInt(x *big.Int) (int64, bool) {
	if !fits(x) {
		return 0, false
	}
	return x.Int64(), true
}

// TestRentInForce_MatchesBigIntOracle verifies the rent in force is
// floor(posted * scale / 10 000), at least 1 when both are positive, at the int64
// edges, and refused exactly when the result passes int64.
//
// Requirement: FR-LND-015, NFR-LND-003
func TestRentInForce_MatchesBigIntOracle(t *testing.T) {
	t.Parallel()
	for _, seed := range oracleSeeds {
		g := newCaseGen(seed)
		for i := range oracleCases {
			posted, scale := g.val(), g.val()
			want := exactRent(posted, scale)
			got, ok := tenure.RentInForce(posted, scale)
			if wantInt, _ := fitInt(want); ok != fits(want) || got != wantInt {
				t.Fatalf("seed %d case %d: RentInForce(%d, %d) = %d, %v, want %s (fits %v)", seed, i, posted, scale, got, ok, want, fits(want))
			}
		}
	}
}

// TestLeastRent_MatchesBigIntOracle verifies LeastRent is rent times
// max(0, executions * draw - own - commons) at the int64 edges, and refused only
// when the exact result or the product executions * draw passes int64.
//
// Requirement: FR-LND-014, NFR-LND-003
//
// Guard: the hire form checks this bound; a wrapped value either lets an
// unaffordable hire form or refuses an affordable one.
func TestLeastRent_MatchesBigIntOracle(t *testing.T) {
	t.Parallel()
	for _, seed := range oracleSeeds {
		g := newCaseGen(seed)
		for i := range oracleCases {
			e, d, own, commons, rent := g.val(), g.val(), g.val(), g.val(), g.val()
			units := new(big.Int).Mul(bi(e), bi(d))
			productFits := fits(units)
			units.Sub(units, bi(own)).Sub(units, bi(commons))
			if units.Sign() < 0 {
				units.SetInt64(0)
			}
			want := units.Mul(units, bi(rent))
			got, ok := tenure.LeastRent(e, d, own, commons, rent)
			if ok && (!fits(want) || got != want.Int64()) {
				t.Fatalf("seed %d case %d: LeastRent(%d, %d, %d, %d, %d) = %d, true, want exact %s", seed, i, e, d, own, commons, rent, got, want)
			}
			if !ok && fits(want) && productFits {
				t.Fatalf("seed %d case %d: LeastRent(%d, %d, %d, %d, %d) refused, exact %s fits", seed, i, e, d, own, commons, rent, want)
			}
		}
	}
}

// exactBand is the step, floor and ceiling of the band in arbitrary precision;
// fit is false when one passes int64.
func exactBand(in tenure.StepIn) (step, floor, ceil *big.Int, fit bool) {
	scale := func(bp int64) *big.Int {
		x := new(big.Int).Mul(bi(in.Base), bi(bp))
		return x.Quo(x, bi(tenure.BasisPoints))
	}
	step, floor, ceil = scale(in.StepBP), scale(in.FloorBP), scale(in.CeilBP)
	if step.Sign() == 0 {
		step.SetInt64(1)
	}
	return step, floor, ceil, fits(step) && fits(floor) && fits(ceil)
}

// exactStep is the rent step of spec-land.md 5.5.1 computed without overflow; fit
// is false when a band value or the raised rent passes int64.
func exactStep(in tenure.StepIn) (next int64, fit bool) {
	if in.Cut > in.Idle {
		return in.Posted, false
	}
	if in.StepBP == 0 || in.Closed {
		return in.Posted, true
	}
	step, floor, ceil, _ := exactBand(in)
	if !fits(step) {
		return in.Posted, false
	}
	switch {
	case in.Unserved > 0 && in.Cut == 0:
		return exactRaise(in.Posted, step, ceil)
	case in.Idle > 0:
		return exactLower(in.Posted, step, floor)
	}
	return in.Posted, true
}

// exactRaise is min(ceil, posted + step); fit is false when a value passes int64.
func exactRaise(posted int64, step, ceil *big.Int) (int64, bool) {
	up := new(big.Int).Add(bi(posted), step)
	if !fits(up) || !fits(ceil) {
		return posted, false
	}
	if up.Cmp(ceil) > 0 {
		up = ceil
	}
	return up.Int64(), true
}

// exactLower is max(floor, posted - step); fit is false when floor passes int64.
func exactLower(posted int64, step, floor *big.Int) (int64, bool) {
	if !fits(floor) {
		return posted, false
	}
	down := new(big.Int).Sub(bi(posted), step)
	if down.Cmp(floor) < 0 {
		down = floor
	}
	return down.Int64(), true
}

// TestStep_MatchesBigIntOracle verifies the rent step against an arbitrary
// precision evaluation of the rule of spec-land.md 5.5.1 at the int64 edges,
// including the refusal when a band value or the raised rent passes int64.
//
// Requirement: FR-LND-019, NFR-LND-003
func TestStep_MatchesBigIntOracle(t *testing.T) {
	t.Parallel()
	for _, seed := range oracleSeeds {
		g := newCaseGen(seed)
		for i := range oracleCases {
			in := tenure.StepIn{
				Posted: g.val(), Base: g.val(), StepBP: g.val(), FloorBP: g.val(), CeilBP: g.val(),
				Unserved: g.small(2), Idle: g.small(2), Cut: g.small(2), Closed: g.r.IntN(8) == 0,
			}
			want, fit := exactStep(in)
			got, ok := tenure.Step(in)
			if ok != fit || got != want {
				t.Fatalf("seed %d case %d: Step(%+v) = %d, %v, want %d, %v", seed, i, in, got, ok, want, fit)
			}
		}
	}
}

// TestHolderShare_MatchesBigIntOracle verifies the share a role holder gives an
// agent, weight times the row's basis points, is exact or refused at the edges.
//
// Requirement: FR-LND-041, NFR-LND-003
func TestHolderShare_MatchesBigIntOracle(t *testing.T) {
	t.Parallel()
	g := newCaseGen(7)
	for i := range oracleCases {
		w, bp := g.val(), g.val()
		want := new(big.Int).Mul(bi(w), bi(bp))
		got, ok := tenure.HolderShare(w, bp)
		if ok != fits(want) || (ok && got != want.Int64()) || (!ok && got != 0) {
			t.Fatalf("case %d: HolderShare(%d, %d) = %d, %v, want %s", i, w, bp, got, ok, want)
		}
	}
}
