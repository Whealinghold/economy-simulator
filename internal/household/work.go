package household

import (
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Net accumulates the net value of one execution of a recipe, value minus cost
// (spec-agents.md 7.1.4, D89). The caller adds the terms one at a time, so the
// tick builds no slice. The zero value is an empty sum. Once an operation
// overflows, ok stays false.
type Net struct {
	value, cost int64
	bad         bool
}

// Output adds floor(q * effective * centre / 10 000) to the value, for a
// non-discard output of q units, the effective yield effective in basis points
// and the agent's belief centre for the commodity. The caller does not add a
// discard output.
func (n *Net) Output(q, effective, centre int64) {
	if n.bad {
		return
	}
	qe, ok := checked.Mul(q, effective)
	if ok {
		var v int64
		if v, ok = checked.MulDivFloor(qe, centre, Basis); ok {
			n.value, ok = checked.Add(n.value, v)
		}
	}
	n.bad = !ok
}

// Input adds q * centre to the cost, for an input of q units.
func (n *Net) Input(q, centre int64) {
	if n.bad {
		return
	}
	p, ok := checked.Mul(q, centre)
	n.Cost(p)
	n.bad = n.bad || !ok
}

// Cost adds an amount already formed to the cost: the wear of a tool
// (production.WearCost, D90) and the rent of a draw.
func (n *Net) Cost(amount int64) {
	if n.bad {
		return
	}
	var ok bool
	n.cost, ok = checked.Add(n.cost, amount)
	n.bad = !ok
}

// Value returns value minus cost and false if any operation overflowed.
func (n *Net) Value() (net int64, ok bool) {
	if n.bad {
		return 0, false
	}
	return checked.Sub(n.value, n.cost)
}

// Better reports whether netA / durA is greater than netB / durB, by
// cross-multiplication in 128 bits with the sign handled. A tie is false: the
// caller iterates the recipes in ascending ID and replaces its choice only on a
// strict improvement, so the lower ID wins (spec-agents.md 7.1.4). A duration
// below 1 gives false.
func Better(netA, durA, netB, durB int64) bool {
	if durA < 1 || durB < 1 {
		return false
	}
	sa, sb := sign(netA), sign(netB)
	if sa != sb {
		return sa > sb
	}
	if sa == 0 {
		return false
	}
	// Same sign: compare |netA| * durB with |netB| * durA; for negative nets the
	// smaller magnitude is the greater value.
	ah, al := bits.Mul64(abs(netA), uint64(durB))
	bh, bl := bits.Mul64(abs(netB), uint64(durA))
	greater := ah > bh || (ah == bh && al > bl)
	less := ah < bh || (ah == bh && al < bl)
	if sa > 0 {
		return greater
	}
	return less
}

// Eligible reports whether a recipe may be chosen: its net is above 0 or it is
// a source, a recipe with no inputs (FR-AGT-035).
func Eligible(net int64, source bool) bool {
	return net > 0 || source
}

// OutputTarget returns ceil(weight * q * effective * outputCoverTicks / (10 000
// * duration)), the stock of one output that the halt compares with
// (spec-agents.md 7.1.5). A negative factor counts as 0. ok is false when the
// duration is below 1 or a product overflows. The result may exceed
// MaxQuantity for a valid configuration, since validation bounds the keep and
// not the output target; the intent's normalise cuts the target then.
func OutputTarget(weight, q, effective, outputCoverTicks, duration int64) (int64, bool) {
	if duration < 1 {
		return 0, false
	}
	wq, ok := checked.Mul(max(weight, 0), max(q, 0))
	if !ok {
		return 0, false
	}
	wqe, ok := checked.Mul(wq, max(effective, 0))
	if !ok {
		return 0, false
	}
	den, ok := checked.Mul(Basis, duration)
	if !ok {
		return 0, false
	}
	return checked.MulDivCeil(wqe, max(outputCoverTicks, 0), den)
}

// Halted reports whether an output is overstocked, stock * 10 000 >= haltRatio *
// outputTarget in 128 bits (spec-agents.md 7.1.5, FR-AGT-036). An outputTarget
// of 0 or below never halts: the literal inequality at a target of 0 is always
// true, which would halt every recipe whose OutputCoverTicks is 0, and a recipe
// with no target has nothing to be over. The caller applies it to outputs with
// a positive CostShare only; a byproduct never halts.
func Halted(stock, outputTarget, haltRatio int64) bool {
	if outputTarget < 1 {
		return false
	}
	lh, ll := bits.Mul64(uint64(max(stock, 0)), uint64(Basis))
	rh, rl := bits.Mul64(uint64(max(haltRatio, 0)), uint64(outputTarget))
	return lh > rh || (lh == rh && ll >= rl)
}

// KeepFoodCap returns floor((stock - upkeepTotal) / q), the most executions an
// input that is also an upkeep commodity allows, and 0 if stock <= upkeepTotal
// or q is below 1 (spec-agents.md 7.1.6, FR-AGT-037). A negative upkeepTotal
// counts as 0.
func KeepFoodCap(stock, upkeepTotal, q int64) int64 {
	upkeepTotal = max(upkeepTotal, 0)
	if q < 1 || stock <= upkeepTotal {
		return 0
	}
	return (stock - upkeepTotal) / q
}

// Request returns min(executions, kcap), the executions to request for a
// recipe, the caller passing welfare.Executions(weight, share) and the least of
// the view's MaxExecutions and the KeepFoodCap of its inputs. It never returns
// (0, true): a request of 0 means "the most the agent can run"
// (spec-agents.md 6.3) and production.Count reads it as no maximum, so a zero
// cap would start the whole job and eat the food (FR-AGT-037). With kcap or
// executions below 1 it returns ok false, the recipe is ineligible and the
// caller tries the next best, returning NoJob only if none is left.
func Request(executions, kcap int64) (n int64, ok bool) {
	if executions < 1 || kcap < 1 {
		return 0, false
	}
	return min(executions, kcap), true
}

func sign(v int64) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// abs returns |v| as a uint64, exact for MinInt64.
func abs(v int64) uint64 {
	if v < 0 {
		return -uint64(v)
	}
	return uint64(v)
}
