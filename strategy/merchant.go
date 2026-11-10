package strategy

import (
	"math/rand/v2"

	economysim "github.com/Whealinghold/economy-sim"
)

// Merchant is the strategy of a merchant role (spec-merchants.md 3): the default
// strategy with a planned trip in place of a planned recipe, a ceiling on the
// bids for the cargo and the haul units, at most one shipment request or delivery
// offer a tick and the takes of a consignee. Work and Revise are the default's. It
// keeps no state, takes no random word beyond the default's and, in a view with no
// merchant group, is the default.
type Merchant struct {
	// Default is the strategy it delegates to for everything it does not change.
	Default economysim.DefaultStrategy
}

var _ economysim.Strategy = Merchant{}

// Work is the default's (spec-merchants.md 3.3): a role with no recipe requests no
// job.
func (m Merchant) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	return m.Default.Work(v, rng)
}

// Revise is the default's (spec-merchants.md 3.3).
func (m Merchant) Revise(o *economysim.OutcomeView, rng *rand.Rand) economysim.Belief {
	return m.Default.Revise(o, rng)
}

// Trade plans the trip of the tick (spec-merchants.md 5 and 6), has the default
// buy its inputs, lowers the bids for the cargo and the haul units to the ceilings
// the trip can bear, appends the shipment request or the delivery offer when the
// trip is ready and the takes of a consignee. Without a merchant group in the
// view it is the default's Trade. The plan notes count each decision once. It
// takes the words the default takes for the same orders and no other.
func (m Merchant) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	if !v.HasMerchant {
		m.Default.Trade(v, rng, p)
		return
	}
	var s state
	s.init(v, &p.Notes)
	if !s.plan() {
		m.Default.Trade(v, rng, p)
		s.takes(p)
		return
	}
	n0 := len(p.Intents)
	m.Default.TradeWith(v, rng, p, s.inputs())
	s.lower(p, n0)
	s.send(p)
	s.takes(p)
}
