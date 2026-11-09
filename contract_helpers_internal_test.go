package economysim

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// This file is the fixture of the merchants module's internal tests (plan 2-15):
// shipConfig with the information and merchants modules and a merchant role
// "broker" of one household in each of the two regions, whose strategy is the
// hand-written script below, a fake at the Strategy extension point (testing.md
// 6.1, plan D517; strategy/ imports the root, so it cannot be used here).

// merchSeen is what the script keeps of one call of Trade: the view's merchant
// group, deep-copied because a view is valid for the call only.
type merchSeen struct {
	agent    AgentID
	region   RegionID
	tick     int64
	has      bool
	merchant MerchantView
	outbound []OutboundRow
}

// merchScript records the views it is given and lets a test append to the plan.
// Its Work asks for no job.
type merchScript struct {
	seen []merchSeen
	// plan, when set, is called with each view and the plan after the record.
	plan func(v *AgentView, p *TradePlan)
}

func (*merchScript) Work(*AgentView, *rand.Rand) JobRequest { return JobRequest{NoJob: true} }

func (s *merchScript) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	m := v.Merchant
	m.Modes = slices.Clone(m.Modes)
	m.Goods = slices.Clone(m.Goods)
	s.seen = append(s.seen, merchSeen{v.Agent, v.Region, v.Tick.Tick, v.HasMerchant, m, slices.Clone(v.Outbound)})
	if s.plan != nil {
		s.plan(v, p)
	}
}

func (*merchScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = (*merchScript)(nil)

// merchConfig returns shipConfig with the information module (speed 40, no
// error, horizon 20), the merchants module and the role "broker": no recipe, an
// informed role that trades grain and the haul commodity, carries by coach and
// consigns, one household in each region. Agents: estates 0 to 2, then in
// canonical order (region, then role) the broker of a_home 3 and its millers 4 and
// 5, the broker of b_far 6 and its millers 7 and 8.
func merchConfig() Config {
	c := shipConfig()
	c.Modules.Information = &Information{InfoSpeed: 40, RumourErrorBP: 0, RumourMaxAge: 20}
	m := DefaultMerchants()
	c.Modules.Merchants = &m
	r := DefaultRole()
	r.ID, r.Strategy, r.Informed = "broker", "script", true
	r.Trades = []string{"grain", "haul"}
	r.MerchantFeeShare = DefaultMerchantFeeShare
	b := DefaultMerchant()
	b.Modes, b.Consigns, b.TakeUnits = []string{"coach"}, true, 10
	r.Merchant = &b
	c.Roles = append(c.Roles, r)
	c.Groups = append(c.Groups,
		Group{Region: "a_home", Role: "broker", Households: 1},
		Group{Region: "b_far", Role: "broker", Households: 1})
	return c
}

// The brokers of merchConfig.
const (
	brokerHome AgentID = 3
	brokerFar  AgentID = 6
)

// newMerchWorld builds the world of merchConfig changed by mut, with a recording
// observer, the audit on, and the script under the name "script" for every role.
func newMerchWorld(t *testing.T, mut func(c *Config), plan func(v *AgentView, p *TradePlan), opts ...Option) (*World, *evLog, *merchScript) {
	t.Helper()
	c := merchConfig()
	if mut != nil {
		mut(&c)
	}
	rec, s := &evLog{}, &merchScript{plan: plan}
	opts = append([]Option{WithObserver(rec), WithAudit(), WithStrategy("script", s)}, opts...)
	w := newMill(t, c, opts...)
	return w, rec, s
}
