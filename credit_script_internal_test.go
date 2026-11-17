package economysim

import (
	"log/slog"
	"math/rand/v2"
	"testing"
)

// loanScript is a strategy for the tests of the credit round: its Trade runs the
// function the test gives for the agent, on an empty plan (the agent places no order
// and so reserves no money), and records the credit group of every view it is given.
// Work and Revise are the default's.
type loanScript struct {
	DefaultStrategy
	// plan is what each agent appends to its plan, by agent.
	plan map[AgentID]func(p *TradePlan)
	// seen is a copy of the credit group of the view of each agent that had one, and
	// missing the agents that were given none, at their last Trade.
	seen    map[AgentID]CreditView
	missing map[AgentID]bool
}

func newLoanScript() *loanScript {
	return &loanScript{plan: map[AgentID]func(*TradePlan){}, seen: map[AgentID]CreditView{}, missing: map[AgentID]bool{}}
}

// Trade records the view and runs the agent's script.
func (s *loanScript) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	if v.Credit != nil {
		cv := *v.Credit
		s.seen[v.Agent], s.missing[v.Agent] = cv, false
		if cv.Lend != nil {
			l := *cv.Lend
			cv.Lend = &l
			s.seen[v.Agent] = cv
		}
	} else {
		s.missing[v.Agent] = true
	}
	if f := s.plan[v.Agent]; f != nil {
		f(p)
	}
}

// The agents of loanConfig, by role and region: home has three millers, which
// borrow, and two weavers, which lend; yonder has two millers and a weaver.
const (
	hMillA, hMillB, hMillC = AgentID(2), AgentID(3), AgentID(4)
	hLendX, hLendY         = AgentID(5), AgentID(6)
	yMillA, yMillB         = AgentID(7), AgentID(8)
	yLend                  = AgentID(9)
)

// loanConfig is the world of the round tests: two regions, the millers hold a
// Borrow block and the weavers a Lend block, the minimum principal is 100 and the
// shield 1 tick. Every upkeep cost is 1 000, so a borrower's exposure room is
// 90 * 1 000 = 90 000.
func loanConfig() Config {
	c := creditConfig(1, 0)
	c.World.MinAgents = 20
	c.Groups = []Group{
		{Region: "home", Role: "miller", Households: 3, Scale: 1},
		{Region: "home", Role: "weaver", Households: 2, Scale: 1},
		{Region: "yonder", Role: "miller", Households: 2, Scale: 1},
		{Region: "yonder", Role: "weaver", Households: 1, Scale: 1},
	}
	c.Modules.Credit.MinPrincipal = 100
	c.Roles[0].Credit = &CreditTerms{Borrow: &BorrowTerms{MaxRateBP: 1500, MaxDebtTicks: 90, SlackTicks: 4}}
	c.Roles[1].Credit = &CreditTerms{Lend: &LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5000}}
	c.Roles[0].Strategy, c.Roles[1].Strategy = "script", "script"
	// Both roles hold grain enough to eat for a long run: the script places no order.
	c.Roles[0].StartStock[0].Quantity = 400
	c.Roles[1].StartStock = []Item{{Commodity: "grain", Quantity: 400}}
	return c
}

// loanBalances is the money each agent of loanConfig holds when the tests start.
var loanBalances = map[AgentID]int64{
	hMillA: 1_000, hMillB: 1_000, hMillC: 1_000, hLendX: 10_000, hLendY: 10_000,
	yMillA: 1_000, yMillB: 1_000, yLend: 10_000,
}

// loanWorld builds the world of loanConfig with the script, an observer and a
// capturing logger, and sets the balances of loanBalances.
func loanWorld(t testing.TB, s *loanScript, opts ...Option) (*World, *evLog, capture) {
	t.Helper()
	return loanWorldFor(t, loanConfig(), s, opts...)
}

// loanWorldFor is loanWorld over the configuration c.
func loanWorldFor(t testing.TB, c Config, s *loanScript, opts ...Option) (*World, *evLog, capture) {
	t.Helper()
	rec, lc := &evLog{}, newCapture()
	opts = append([]Option{WithSeed(exitSeed), WithStrategy("script", s), WithObserver(rec), WithLogger(slog.New(lc)), WithAudit()}, opts...)
	w := newMill(t, c, opts...)
	setBalances(t, w, loanBalances)
	return w, rec, lc
}
