package economysim

import "github.com/Whealinghold/economy-sim/internal/beliefs"

// Belief is the price belief of one agent for one commodity (spec-beliefs.md
// 3.1): the Centre, its best guess, and the Width, its half-width in basis
// points of the centre. A belief the world reads or stores holds the invariant
// of spec-beliefs.md 3.3: the centre lies between the belief floor and the
// belief ceiling and the width between the configured minimum and maximum.
type Belief struct {
	Centre Price
	Width  int64
}

// Bounds is the interval a belief's centre is kept in (spec-beliefs.md 3.2):
// Floor is the lowest centre allowed and Ceiling the highest. Floor never
// exceeds Ceiling, and both lie in [1, MaxPrice].
type Bounds struct {
	Floor, Ceiling Price
}

// beliefParams returns the parameters the belief rule reads, built once at
// world construction. PriceWindow is not among them: it is the capacity of a
// market's window.
func beliefParams(w World) beliefs.Params {
	return beliefs.Params{
		Step: w.BeliefStep, AdjustProb: w.BeliefAdjustProb, Significance: w.BeliefSignificance,
		EvidenceShare: w.BeliefEvidenceShare,
		StockLow:      w.BeliefStockLow, StockHigh: w.BeliefStockHigh, MaxAge: w.BeliefMaxAge,
		WidthInit: w.BeliefWidthInit, WidthMin: w.BeliefWidthMin,
		WidthMax: w.BeliefWidthMax, WidthStep: w.BeliefWidthStep,
		CostFloor: w.CostFloor, CostCeiling: w.CostCeiling,
		RefBandLow: w.RefBandLow, RefBandHigh: w.RefBandHigh,
		MaxPrice: MaxPrice,
	}
}

// beliefReason classifies one outcome into the reason of the "belief updated"
// Debug line, with the precedence of spec-beliefs.md 12.2: a closed gate, then
// the floor, then the ceiling, then a moved centre, then held.
func beliefReason(c beliefs.Change) Reason {
	switch {
	case !c.Open:
		return ReasonGateClosed
	case c.AtFloor():
		return ReasonAtFloor
	case c.AtCeiling():
		return ReasonAtCeiling
	case c.Moved():
		return ReasonMoved
	}
	return ReasonHeld
}

// beliefsEvent maps the tally of a market's tick to its event (spec-beliefs.md
// 12.1). The tally must hold at least one outcome.
func beliefsEvent(tick int64, region RegionID, commodity CommodityID, t *beliefs.Tally) BeliefsUpdated {
	mean, _ := t.Mean()
	return BeliefsUpdated{
		Tick: tick, Region: region, Commodity: commodity,
		Outcomes: t.Outcomes, Adjusted: t.Adjusted, Raised: t.Raised, Lowered: t.Lowered,
		AtFloor: t.AtFloor, AtCeiling: t.AtCeiling, Conflicts: t.Conflicts, Clamped: t.Clamped,
		MeanCentre: Price(mean), LowCentre: Price(t.Low()), HighCentre: Price(t.High()),
	}
}
