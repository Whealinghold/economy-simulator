package economysim

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// This file is the fixture of the information module's internal tests: a world
// of three regions in a line with unequal distances, whose households trade grain
// at a price that rises with the tick, and a strategy that records the views it
// is given.

// The regions of infoConfig, by ID (the canonical order is by string).
const (
	infoA RegionID = iota
	infoB
	infoC
)

// infoSpeed is the information speed of the fixture, in distance units a tick.
const infoSpeed = 40

// infoPrice is the price at which grain clears in a region at a tick: the bid and
// the ask of the pair of households there carry the same limit, so the market
// publishes exactly this, and it differs for every region and every tick, so a
// rumour that reads the wrong vintage shows.
func infoPrice(region RegionID, tick int64) Price {
	return Price(1_000 + 100*int64(region) + 7*tick)
}

// seenView is what the recording strategy keeps of one call of Trade.
type seenView struct {
	agent   AgentID
	region  RegionID
	tick    int64
	rumours []Rumour
	weight  int64
}

// infoScript trades grain in pairs and records the views it is given: in each
// region the first household of a role bids and the second asks, both at
// infoPrice, so the market publishes that price. Work records the views passed
// to it.
type infoScript struct {
	trades []seenView
	works  []seenView
}

func (s *infoScript) Work(v *AgentView, _ *rand.Rand) JobRequest {
	s.works = append(s.works, seenView{v.Agent, v.Region, v.Tick.Tick, slices.Clone(v.Rumours), v.Weight})
	return JobRequest{NoJob: true}
}

func (s *infoScript) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	s.trades = append(s.trades, seenView{v.Agent, v.Region, v.Tick.Tick, slices.Clone(v.Rumours), v.Weight})
	// The odd agent of a region's pair asks and the even one bids (infoConfig).
	side := Bid
	if v.Agent%2 == 1 {
		side = Ask
	}
	p.Intents = append(p.Intents, Intent{Commodity: grain, Side: side, Quantity: 1, Limit: infoPrice(v.Region, v.Tick.Tick)})
}

func (*infoScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = (*infoScript)(nil)

// infoConfig returns the fixture: three regions a, b and c on a line with the
// edges a-b 40 both ways, b-c 80 from b to c and 120 from c to b; so, over the
// open edges, the distance from a to c is 120 and from c to a 160. Two roles
// trade there: "miller" in a and b (two households each: grain and flour are
// its sources) and "carrier" in c (two households, tradable set grain alone),
// both following the registered strategy "script". The module is on with an
// information speed of infoSpeed, an error of 0 and a horizon of 20, and both
// roles are informed. Agents: estates 0 to 2, then a: 3 and 4, b: 5 and 6,
// c: 7 and 8 (canonical order is by region, then role). The two households of a
// region are one odd ID and one even ID: the odd one asks and the even one bids.
func infoConfig() Config {
	c := tradeConfig(0)
	c.Roles[0].ID = "miller"
	c.Roles[0].Informed = true
	carrier := c.Roles[0]
	carrier.ID = "carrier"
	carrier.Recipes = nil
	carrier.StartStock = []Item{{Commodity: "grain", Quantity: 10}}
	c.Roles = append(c.Roles, carrier)
	c.Regions[0].ID = "a"
	for _, id := range []string{"b", "c"} {
		r := DefaultRegion()
		r.ID = id
		c.Regions = append(c.Regions, r)
	}
	c.Edges = []Edge{
		{From: "a", To: "b", Distance: 40}, {From: "b", To: "a", Distance: 40},
		{From: "b", To: "c", Distance: 80}, {From: "c", To: "b", Distance: 120},
	}
	c.Groups = []Group{
		{Region: "a", Role: "miller", Households: 2}, {Region: "b", Role: "miller", Households: 2},
		{Region: "c", Role: "carrier", Households: 2},
	}
	c.Modules.Information = &Information{InfoSpeed: infoSpeed, RumourErrorBP: 0, RumourMaxAge: 20}
	return c
}

// newInfoWorld builds the world of infoConfig changed by mut, with the recording
// strategy under the name "script" and the audit on, and returns the strategy.
func newInfoWorld(t *testing.T, mut func(c *Config), opts ...Option) (*World, *infoScript) {
	t.Helper()
	c := infoConfig()
	if mut != nil {
		mut(&c)
	}
	s := &infoScript{}
	w := newMill(t, c, append([]Option{WithStrategy("script", s), WithAudit()}, opts...)...)
	return w, s
}
