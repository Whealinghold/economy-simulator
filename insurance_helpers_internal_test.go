package economysim

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// This file is the world of the insurance tests (plan 3-ins Phases 4 and 5): the world
// of shipConfig (three regions, home and far joined, the coach hired and foot
// self-carried) with a certain route row home to far, the credit module and the
// insurance module on, the role miller an insured and a role backer an underwriter, the
// households of the miller driven by a script.

// insScript is a Strategy that requests no job and lets a function build the plan of
// each household; the function gates itself on the agent and the tick.
type insScript struct {
	plan func(v *AgentView, p *TradePlan)
}

func (*insScript) Work(*AgentView, *rand.Rand) JobRequest { return JobRequest{NoJob: true} }

func (s *insScript) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	if s.plan != nil {
		s.plan(v, p)
	}
}

func (*insScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = (*insScript)(nil)

// The households of insConfig, by agent ID (the roles are sorted by ID, so backer comes
// before miller): the backer in home, two millers in home and two in far; the estates
// are agents 0 to 2.
const (
	insBacker AgentID = 3
	insHome   AgentID = 4
	insHome2  AgentID = 5
	insFar    AgentID = 6
)

// insConfig is the configuration of the tests: the insured terms and the underwriter
// terms are the ones the test passes, nil for a role with no block.
func insConfig(ins *InsureTerms, uw *UnderwriteTerms) Config {
	c := shipConfig()
	riskShip(riskCoach("a_home", "b_far", 10_000, 2_500, false))(&c)
	c.Modules.Credit = &Credit{MinPrincipal: 1, ShieldTicks: 1, WriteOffTicks: 3, MaxClaims: 100}
	c.Modules.Insurance = &Insurance{ValueCapBP: 20_000, PayoutTicks: 1, MinSum: 10, MaxPolicies: 50, MaxCoverRequests: 4}
	b := c.Roles[0]
	b.ID = "backer"
	if ins != nil {
		c.Roles[0].Cover = &CoverTerms{Insure: ins}
	}
	if uw != nil {
		b.Cover = &CoverTerms{Underwrite: uw}
	}
	c.Roles = append(c.Roles, b)
	c.Groups = []Group{{Region: "a_home", Role: "miller", Households: 2}, {Region: "a_home", Role: "backer", Households: 1}, {Region: "b_far", Role: "miller", Households: 2}}
	return c
}

// defaultInsure and defaultUnderwrite are the terms of most tests: an insured that buys
// on any lane with a loss, at most a loading of 20 000, at least half of a loss; an
// underwriter at a fair loading, 80% cover, cap 200%, offering half of its room.
func defaultInsure() *InsureTerms {
	return &InsureTerms{MinLossBP: 1, RuinShareBP: 0, MaxLoadingBP: 20_000, MinCoverBP: 5_000}
}

func defaultUnderwrite() *UnderwriteTerms {
	return &UnderwriteTerms{LoadingBP: 10_000, StoreRateBP: 100, CoverBP: 8_000, CapBP: 20_000, ShareBP: 5_000}
}

// newInsWorld builds a world of insConfig with the audit on and a recording observer,
// whose households follow plan, then ends the construction bucket.
func newInsWorld(t *testing.T, c Config, plan func(v *AgentView, p *TradePlan), opts ...Option) (*World, *evLog) {
	t.Helper()
	rec := &evLog{}
	opts = append([]Option{WithObserver(rec), WithAudit(), WithStrategy("script", &insScript{plan: plan})}, opts...)
	w := newMill(t, c, opts...)
	if err := w.ledger.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction() = %v, want nil", err)
	}
	return w, rec
}

// viewOf builds the Trade view of an agent at the start of phase 3 as the world does.
func viewOf(t *testing.T, w *World, id AgentID) *AgentView {
	t.Helper()
	w.refreshViewCtx()
	v := &AgentView{}
	if err := w.fillAgentView(v, &w.agents[id], viewTrade); err != nil {
		t.Fatalf("fillAgentView(agent %d) error = %v, want nil", id, err)
	}
	if v.Insurance != nil { // the world's group is a scratch copy reused by the next agent
		g := *v.Insurance
		g.LaneLoss, g.UnitValue = slices.Clone(g.LaneLoss), slices.Clone(g.UnitValue)
		if g.Insure != nil {
			i := *g.Insure
			g.Insure = &i
		}
		if g.Underwrite != nil {
			u := *g.Underwrite
			g.Underwrite = &u
		}
		v.Insurance = &g
	}
	return v
}
