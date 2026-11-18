package employment

import "github.com/Whealinghold/economy-sim/internal/checked"

// Input is one input of a recipe as an employer's stock holds it: Stock units
// held, Keep units the keep-the-food rule leaves alone (spec-agents.md 7.1.6)
// and Qty the units one execution needs (at least 1).
type Input struct {
	Stock, Keep, Qty int64
}

// Plan holds what bounds the positions an employer posts (spec-employment.md
// 4.3.3): the establishment MaxHires less the Hired executions already open,
// the inputs, the unreserved Tools held for each tool of the recipe, the
// executions of its own request KOwn, and its cash: Money less UpkeepCost, at
// Bid per household-tick for Duration ticks.
type Plan struct {
	MaxHires, Hired int64
	Inputs          []Input
	Tools           []int64
	KOwn            int64
	Money           int64
	UpkeepCost      int64
	Bid             int64
	Duration        int64
}

// Positions returns the number of positions the default strategy posts:
//
//	min(MaxHires - Hired, by_stock, by_tools, by_cash), each at least 0
//	by_stock = min over inputs of floor((Stock - Keep) / Qty) - KOwn
//	by_tools = min over tools of (unreserved items) - KOwn
//	by_cash  = floor((Money - UpkeepCost) / (Bid * Duration))
//
// A recipe with no inputs and no tools is not bounded by them. A Bid below 1
// posts nothing. The quotient of a negative numerator is floored and then
// clamped to 0. It reports false when a count is negative, a Qty or the
// Duration is below 1, or the cash product does not fit.
func Positions(p Plan) (int64, bool) {
	if p.MaxHires < 0 || p.Hired < 0 || p.KOwn < 0 || p.UpkeepCost < 0 || p.Duration < 1 {
		return 0, false
	}
	n, ok := stockBound(p)
	if !ok {
		return 0, false
	}
	if p.Bid < 1 {
		return 0, true
	}
	perPosition, ok := checked.Mul(p.Bid, p.Duration)
	if !ok {
		return 0, false
	}
	var cash int64
	if p.Money > p.UpkeepCost {
		cash = p.Money - p.UpkeepCost
	}
	return min(n, cash/perPosition), true
}

// stockBound returns the bound of Positions from the hiring cap, the inputs
// and the tools; it reports false when a count is negative or a Qty is below 1.
func stockBound(p Plan) (int64, bool) {
	n := max(p.MaxHires-p.Hired, 0)
	for _, in := range p.Inputs {
		if in.Stock < 0 || in.Keep < 0 || in.Qty < 1 {
			return 0, false
		}
		n = min(n, max((in.Stock-in.Keep)/in.Qty-p.KOwn, 0))
	}
	for _, items := range p.Tools {
		if items < 0 {
			return 0, false
		}
		n = min(n, max(items-p.KOwn, 0))
	}
	return n, true
}
