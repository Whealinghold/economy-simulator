package occupation

import (
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// ApplyParams are the inputs of ApplyOK, the application rule of
// spec-skill-barriers.md 4.5. ED and EO are the signals of the destination and
// of the agent's own role, as for CapitalOK; ToolPrice is P (0 for a role with
// no tool); CTrain is TrainingTicks times the agent's surplus (skill.CTrain);
// QWait is the expected wait q_w in ticks (skill.Wait).
type ApplyParams struct {
	ED, EO                      int64
	ToolPrice, CTrain           int64
	CapitalReturn, Upkeep, Year int64
	QWait                       int64
}

// ApplyOK is the queue-discounted capital test of spec-skill-barriers.md 4.5
// (FR-SKL-005): p_d * D_work * (10 000 - CapitalReturn * q_w / TicksPerYear) >=
// (P + C_train) * CapitalReturn, in the form of CapitalOK, where p_d * D_work *
// 10 000 is diff * ut with diff = ED - EO and ut = Upkeep * Year:
//
//	diff * ut * bracket >= (ToolPrice + CTrain) * CapitalReturn * 10 000
//
// The bracket 10 000 - CapitalReturn * QWait / Year, the division truncated
// toward zero, is floored at 0, and there is no application when it is 0. Both
// sides are exact products of three factors in 192 bits. The test is denied,
// never wrapped, for a negative input, when diff is below 0, when ut is not
// above 0 or does not fit int64, and when ToolPrice + CTrain or the discount
// overflows. With QWait 0 and CTrain 0 it equals CapitalOK, except that a
// ToolPrice of 0 passes only when diff is not below 0 and ut is above 0.
func ApplyOK(p ApplyParams) bool {
	if p.ToolPrice < 0 || p.CTrain < 0 || p.CapitalReturn < 0 || p.QWait < 0 || p.ED < p.EO {
		return false
	}
	ut, bracket, ok := applyFactors(p)
	if !ok {
		return false
	}
	capital, ok := checked.Add(p.ToolPrice, p.CTrain)
	if !ok {
		return false
	}
	// ED >= EO, so the wrapped uint64 difference is the exact diff.
	left := mul3(uint64(p.ED)-uint64(p.EO), uint64(ut), uint64(bracket))
	right := mul3(uint64(capital), uint64(p.CapitalReturn), uint64(Basis))
	for i := range left {
		if left[i] != right[i] {
			return left[i] > right[i]
		}
	}
	return true
}

// applyFactors returns ut = Upkeep * Year and the bracket 10 000 -
// CapitalReturn * QWait / Year, and ok false when ut is not above 0 or does not
// fit, the year is not above 0, the discount overflows or the bracket is not
// above 0 (the floor of 4.5).
func applyFactors(p ApplyParams) (ut, bracket int64, ok bool) {
	ut, ok = checked.Mul(p.Upkeep, p.Year)
	if !ok || ut <= 0 || p.Year <= 0 {
		return 0, 0, false
	}
	disc, ok := checked.MulDiv(p.CapitalReturn, p.QWait, p.Year)
	if !ok || disc >= Basis {
		return 0, 0, false
	}
	return ut, Basis - disc, true
}

// mul3 returns the exact product a * b * c as three 64-bit words, most
// significant first. a and b may use all 64 bits and c must be below 2^63, so
// the product is below 2^191.
func mul3(a, b, c uint64) [3]uint64 {
	hi, lo := bits.Mul64(a, b)
	h0, l0 := bits.Mul64(lo, c)
	h1, l1 := bits.Mul64(hi, c)
	mid, carry := bits.Add64(h0, l1, 0)
	return [3]uint64{h1 + carry, mid, l0}
}
