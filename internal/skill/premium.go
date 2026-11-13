package skill

import "github.com/Whealinghold/economy-sim/internal/checked"

// BreakEven returns the premium, in basis points, that the chain's own tool and
// training predict at break-even (spec-skill-barriers.md 3, 4.8):
// 10 000 + (price + ctrain) * capReturn / (dwork * wUnskilled), the quotient
// truncated toward zero. ok is false when dwork * wUnskilled is not above 0,
// when an input is negative, or when a step overflows.
func BreakEven(price, ctrain, capReturn, dwork, wUnskilled int64) (bp int64, ok bool) {
	if price < 0 || ctrain < 0 || capReturn < 0 {
		return 0, false
	}
	capital, ok := checked.Add(price, ctrain)
	if !ok {
		return 0, false
	}
	den, ok := checked.Mul(dwork, wUnskilled)
	if !ok || den <= 0 {
		return 0, false
	}
	excess, ok := checked.MulDiv(capital, capReturn, den)
	if !ok {
		return 0, false
	}
	return checked.Add(Basis, excess)
}

// The rule constants of the tolerance of check D (spec-skill-barriers.md 3, both
// placeholders): a share of the predicted excess and a floor, in basis points.
const (
	// ToleranceShareBP is the share of the predicted excess over 10 000, 25%.
	ToleranceShareBP int64 = 2_500
	// ToleranceFloorBP is the least tolerance, 200 basis points.
	ToleranceFloorBP int64 = 200
)

// Tolerance returns the tolerance of check D: the largest of the share of the
// predicted excess, the floor, and cvx2, twice the observed coefficient of
// variation of the premium across seeds (spec-skill-barriers.md 3, FR-SKL-024).
// A negative excess gives a negative share, so the floor applies. The share is
// truncated toward zero.
func Tolerance(excess, cvx2 int64) int64 {
	// excess = q * 10 000 + r, so the share is exact without forming excess *
	// 2 500, which could overflow: q * 2 500 + r * 2 500 / 10 000.
	q, r := excess/Basis, excess%Basis
	share := q*ToleranceShareBP + r*ToleranceShareBP/Basis
	return max(share, ToleranceFloorBP, cvx2)
}
