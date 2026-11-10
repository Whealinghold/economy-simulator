package economysim

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The world of these tests is extConfig: coast (region 0) is the outside node,
// lowland (region 1) holds seven households of the role trader, which trades
// grain and salt. bread (commodity 0) is the upkeep. The audit is on and the
// construction bucket has been ended, so the calls here fall in the totals of the
// tick the next step runs.
const (
	regCoast   RegionID = 0
	regLowland RegionID = 1
)

// intWorld is extWorld of extConfig with the observer log; the households are
// agents 2 to 8 (two estates, two counterparties, then the households).
func intWorld(t *testing.T, opts ...Option) (*World, *evLog) {
	t.Helper()
	w, _, rec := extWorld(t, extConfig(), opts...)
	return w, rec
}

// drawWorld is intWorld with a role that has a recipe and a spread of its target
// ratio, so that both creation words are observable: the target and the residues.
func drawWorld(t *testing.T) *World {
	t.Helper()
	c := extConfig()
	c.Roles[0].Recipes, c.Roles[0].TargetSpread = []string{"make"}, 2_000
	w, _, _ := extWorld(t, c)
	return w
}

// bytesOf is the checkpoint of w, for "changes nothing" by bytes (plan 2-17 D558).
func bytesOf(t *testing.T, w *World) []byte {
	t.Helper()
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	return slices.Clone(cp.b)
}

// TestAddAgent_Errors verifies that every check of spec-integration.md 4.1.1 has
// its sentinel, in the order of the spec, that an argument error is a
// *ValidationError with the path of the argument, and that a refused call changes
// nothing (the checkpoint bytes), emits nothing and issues no ID.
//
// Requirement: FR-INT-020, FR-INT-002, FR-INT-005
func TestAddAgent_Errors(t *testing.T) {
	t.Parallel()
	ok := NewAgent{Region: regLowland, Role: 0, Money: 100, Stock: []StockItem{{Commodity: 1, Quantity: 3}}}
	with := func(f func(n *NewAgent)) NewAgent { n := ok; n.Stock = slices.Clone(ok.Stock); f(&n); return n }
	tests := []struct {
		name string
		n    NewAgent
		want error
		path string
	}{
		{"region below range", with(func(n *NewAgent) { n.Region = NoRegion }), ErrUnknownRegion, "region"},
		{"region above range", with(func(n *NewAgent) { n.Region = 9 }), ErrUnknownRegion, "region"},
		{"outside node", with(func(n *NewAgent) { n.Region = regCoast }), ErrOutsideNode, "region"},
		{"role unknown", with(func(n *NewAgent) { n.Role = 5 }), ErrUnknownRole, "role"},
		{"strategy unknown", with(func(n *NewAgent) { n.Strategy = "nope" }), ErrUnknownStrategy, "strategy"},
		{"money below 0", with(func(n *NewAgent) { n.Money = -1 }), ErrOutOfBounds, "money"},
		{"money above MaxMoney", with(func(n *NewAgent) { n.Money = MaxMoney + 1 }), ErrOutOfBounds, "money"},
		{"quantity 0", with(func(n *NewAgent) { n.Stock[0].Quantity = 0 }), ErrOutOfBounds, "quantity"},
		{"quantity above MaxQuantity", with(func(n *NewAgent) { n.Stock[0].Quantity = MaxQuantity + 1 }), ErrOutOfBounds, "quantity"},
		{"commodity unknown", with(func(n *NewAgent) { n.Stock[0].Commodity = 7 }), ErrUnknownCommodity, "commodity"},
		{"commodity twice", with(func(n *NewAgent) { n.Stock = append(n.Stock, n.Stock[0]) }), ErrDuplicateID, "commodity"},
		// The order of 4.1.1: the region before the role before the strategy before the money.
		{"region first", NewAgent{Region: 9, Role: 5, Strategy: "nope", Money: -1}, ErrUnknownRegion, "region"},
		{"bounds before commodity", with(func(n *NewAgent) { n.Stock = []StockItem{{Commodity: 7, Quantity: 0}} }), ErrOutOfBounds, "quantity"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := intWorld(t)
			before, next, n := bytesOf(t, w), w.state.nextAgent, len(rec.events)
			id, err := w.AddAgent(tt.n)
			if !errors.Is(err, tt.want) || id != NoAgent {
				t.Fatalf("AddAgent(%+v) = %d, %v, want %d, %v", tt.n, id, err, NoAgent, tt.want)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Path[0] != "n" || ve.Path[len(ve.Path)-1] != tt.path {
				t.Errorf("AddAgent error %v is not a ValidationError with a path ending %q", err, tt.path)
			}
			if !bytes.Equal(bytesOf(t, w), before) || w.state.nextAgent != next || len(rec.events) != n {
				t.Errorf("a refused AddAgent changed the world: bytes equal %v, next agent %d (was %d), %d events (was %d)",
					bytes.Equal(bytesOf(t, w), before), w.state.nextAgent, next, len(rec.events), n)
			}
		})
	}
}

// TestAddAgent_Overflow verifies that a creation that would pass a bound of the
// ledger is refused with ErrOverflow before any account is opened: agents added
// with MaxMoney each pass the flow total of the region at the tenth, and that call
// changes nothing.
//
// Requirement: FR-INT-020, FR-INT-002
func TestAddAgent_Overflow(t *testing.T) {
	t.Parallel()
	w, _ := intWorld(t)
	var err error
	for i := 0; i < 20 && err == nil; i++ {
		before := bytesOf(t, w)
		_, err = w.AddAgent(NewAgent{Region: regLowland, Role: 0, Money: MaxMoney})
		if err != nil && !bytes.Equal(bytesOf(t, w), before) {
			t.Errorf("the refused AddAgent changed the world")
		}
	}
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("AddAgent(MaxMoney) over and over ended with %v, want ErrOverflow", err)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() after the refusal = %v, want nil", err)
	}
}

// TestAddAgent_Creates verifies what the call makes (spec-integration.md 4.1.2):
// the next ID, weight 1, External and Mortal as asked, the strategy, the account
// in the region with the stated money and goods, beliefs at the seed prices, the
// current tick as RoleSince, the counters and one AgentSpawned with External true.
//
// Requirement: FR-INT-021, FR-INT-009, FR-INT-011, FR-SCL-013
func TestAddAgent_Creates(t *testing.T) {
	t.Parallel()
	w, rec := intWorld(t)
	next, live, households := w.state.nextAgent, w.live.agents, w.live.households
	cell := int(regLowland)*len(w.cfg.Roles) + 0
	cellBefore := w.live.inRole[cell]
	id, err := w.AddAgent(NewAgent{Region: regLowland, Role: 0, Money: 2_000, Mortal: true,
		Stock: []StockItem{{Commodity: 1, Quantity: 10}, {Commodity: 2, Quantity: 4}}})
	if err != nil || id != next {
		t.Fatalf("AddAgent() = %d, %v, want %d, nil", id, err, next)
	}
	a := &w.agents[id]
	if a.weight != 1 || !a.external || !a.mortal || a.role != 0 || a.roleSince != w.Tick() || w.strategyName(a) != "directed" {
		t.Errorf("agent = weight %d external %v mortal %v role %d since %d strategy %q, want 1 true true 0 %d directed",
			a.weight, a.external, a.mortal, a.role, a.roleSince, w.strategyName(a), w.Tick())
	}
	if !isDirected(w.ext.strategies[a.strategy]) {
		t.Errorf("the strategy of an agent added with no name is not directed")
	}
	v, err := w.ledger.Account(a.account)
	if err != nil || v.Balance != 2_000 || v.Region != int(regLowland) {
		t.Errorf("account = %+v, %v, want balance 2000 in region %d", v, err, regLowland)
	}
	for c, want := range map[int]int64{1: 10, 2: 4, 0: 0} {
		if got, _, err := w.ledger.Holding(a.account, c); err != nil || got != want {
			t.Errorf("Holding(commodity %d) = %d, %v, want %d", c, got, err, want)
		}
	}
	row := w.roles.row(0)
	if len(a.beliefs) != len(row.tradable) {
		t.Fatalf("beliefs = %d, want %d (the tradable set)", len(a.beliefs), len(row.tradable))
	}
	for i, c := range row.tradable {
		if want := int64(w.markets.reports[w.markets.index(regLowland, c)].Price); a.beliefs[i].Centre != want {
			t.Errorf("belief %d centre = %d, want the market price %d (the window is empty)", i, a.beliefs[i].Centre, want)
		}
	}
	if w.live.agents != live+1 || w.live.households != households+1 || w.live.inRole[cell] != cellBefore+1 {
		t.Errorf("counters = %d agents, %d households, %d in role, want %d, %d, %d",
			w.live.agents, w.live.households, w.live.inRole[cell], live+1, households+1, cellBefore+1)
	}
	var spawned []AgentSpawned
	for _, e := range of[AgentSpawned](rec) {
		if e.External {
			spawned = append(spawned, e)
		}
	}
	want := AgentSpawned{Tick: w.Tick(), Agent: id, Region: regLowland, Role: 0, Weight: 1, StartMoney: 2_000, External: true}
	if len(spawned) != 1 || spawned[0] != want {
		t.Errorf("AgentSpawned events = %+v, want exactly %+v", spawned, want)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
	if err := w.Step(); err != nil {
		t.Errorf("Step() with the new agent = %v, want nil", err)
	}
}

// TestAddAgent_LastID_AndKeyedDraws verifies that the new agent takes the next ID,
// that its two creation words come from the stream keyed (siteAddAgent, Tick(),
// the agent) and so do not depend on the agents added before it or on the call
// order, and that no other agent's target or residue changes.
//
// Requirement: FR-INT-022, FR-INT-007, FR-INT-008
func TestAddAgent_LastID_AndKeyedDraws(t *testing.T) {
	t.Parallel()
	spec := NewAgent{Region: regLowland, Role: 0, Money: 10}
	w1 := drawWorld(t)
	type draws struct {
		target   int64
		residues []int64
	}
	snap := func(w *World) map[AgentID]draws {
		m := map[AgentID]draws{}
		w.eachLive(func(a *agent) { m[a.id] = draws{a.target, slices.Clone(a.residues)} })
		return m
	}
	others := snap(w1)
	id1, err := w1.AddAgent(spec)
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	for id, d := range others {
		if now := snap(w1)[id]; now.target != d.target || !slices.Equal(now.residues, d.residues) {
			t.Errorf("agent %d draws changed by AddAgent: %+v, want %+v", id, now, d)
		}
	}
	// A second world adds a different agent first; the agent that gets id1's ID in
	// the first world and the one that gets it here have the same words.
	w2 := drawWorld(t)
	if id2, err := w2.AddAgent(spec); err != nil || id2 != id1 {
		t.Fatalf("AddAgent() in a second world = %d, %v, want %d", id2, err, id1)
	}
	if a, b := w1.agents[id1], w2.agents[id1]; a.target != b.target || !slices.Equal(a.residues, b.residues) {
		t.Errorf("the same call in two worlds gave targets %d and %d", a.target, b.target)
	}
	// The words are those of the keyed stream, drawn here by hand.
	probe := newAgent(agentInit{id: id1, role: 0, weight: 1}, w1.roles.row(0), &w1.recipes)
	role := &w1.cfg.Roles[0]
	w1.rnd.keyed(siteAddAgent, w1.Tick(), int64(id1))
	drawCreation(w1.rnd, &probe, role.TargetRatio, role.TargetSpread)
	if got := w1.agents[id1]; got.target != probe.target || !slices.Equal(got.residues, probe.residues) {
		t.Errorf("creation words = %d %v, want those of the stream (siteAddAgent, %d, %d): %d %v",
			got.target, got.residues, w1.Tick(), id1, probe.target, probe.residues)
	}
	// A second agent in the first world is the next ID and does not move the first's words.
	first := w1.agents[id1]
	id3, err := w1.AddAgent(spec)
	if err != nil || id3 != id1+1 {
		t.Fatalf("second AddAgent() = %d, %v, want %d", id3, err, id1+1)
	}
	if now := w1.agents[id1]; now.target != first.target || !slices.Equal(now.residues, first.residues) {
		t.Errorf("a later AddAgent moved the words of agent %d", id1)
	}
}

// TestAddAgent_BoundaryFlow verifies that the money and goods of a new agent are a
// counted faucet under initial: in the construction bucket when the call is made
// before the first step, in the totals of the tick otherwise, and the audit holds.
//
// Requirement: FR-INT-009, FR-INT-011
func TestAddAgent_BoundaryFlow(t *testing.T) {
	t.Parallel()
	initial := func(l *ledger.Ledger, bucket bool) (money, grain int64) {
		for _, f := range l.AppendFlows(nil, bucket) {
			if f.Reason != ledger.Initial || f.Region != int(regLowland) {
				continue
			}
			if f.Commodity == ledger.Money {
				money = f.Total
			} else if f.Commodity == 1 {
				grain = f.Total
			}
		}
		return money, grain
	}
	spec := NewAgent{Region: regLowland, Role: 0, Money: 700, Stock: []StockItem{{Commodity: 1, Quantity: 9}}}
	for _, tt := range []struct {
		name   string
		bucket bool
		world  func(t *testing.T) *World
	}{
		{"before the first step", true, func(t *testing.T) *World {
			return newMill(t, extConfig(), WithAudit(), WithStrategy("script", &extScript{plan: map[AgentID][]Intent{}}))
		}},
		{"after construction", false, func(t *testing.T) *World { w, _ := intWorld(t); return w }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := tt.world(t)
			m0, g0 := initial(w.ledger, tt.bucket)
			if _, err := w.AddAgent(spec); err != nil {
				t.Fatalf("AddAgent() = %v, want nil", err)
			}
			m1, g1 := initial(w.ledger, tt.bucket)
			if m1-m0 != 700 || g1-g0 != 9 {
				t.Errorf("initial flow rose by %d money and %d grain, want 700 and 9", m1-m0, g1-g0)
			}
			if err := w.Audit(); err != nil {
				t.Errorf("Audit() = %v, want nil", err)
			}
		})
	}
}

// wrapped holds a Directed in a field: the world cannot see through it.
type wrapped struct{ d Directed }

func (w wrapped) Work(v *AgentView, r *rand.Rand) JobRequest { return w.d.Work(v, r) }
func (w wrapped) Trade(v *AgentView, r *rand.Rand, p *TradePlan) {
	w.d.Trade(v, r, p)
}
func (w wrapped) Revise(o *OutcomeView, r *rand.Rand) Belief { return w.d.Revise(o, r) }

// TestStrategyName_Directed verifies the names an external agent's strategy may
// have: empty means directed, default, directed and a registered name resolve, an
// unknown one is ErrUnknownStrategy, and only the library's Directed is directed.
//
// Requirement: FR-INT-023
func TestStrategyName_Directed(t *testing.T) {
	t.Parallel()
	w, _ := intWorld(t, WithStrategy("wrap", wrapped{Directed{Fallback: DefaultStrategy{}}}))
	tests := []struct {
		name, strategy string
		wantName       string
		directed       bool
		err            error
	}{
		{"empty", "", "directed", true, nil},
		{"directed", "directed", "directed", true, nil},
		{"default", "default", "default", false, nil},
		{"registered", "script", "script", false, nil},
		{"wrapper", "wrap", "wrap", false, nil},
		{"unknown", "nobody", "", false, ErrUnknownStrategy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := w.AddAgent(NewAgent{Region: regLowland, Role: 0, Strategy: tt.strategy})
			if !errors.Is(err, tt.err) {
				t.Fatalf("AddAgent(strategy %q) = %v, want %v", tt.strategy, err, tt.err)
			}
			if err != nil {
				return
			}
			a := &w.agents[id]
			if got := w.strategyName(a); got != tt.wantName {
				t.Errorf("strategyName = %q, want %q", got, tt.wantName)
			}
			if got := isDirected(w.ext.strategies[a.strategy]); got != tt.directed {
				t.Errorf("isDirected = %v, want %v", got, tt.directed)
			}
		})
	}
}

// TestStrategyTable_RolesKeepTheirIndex verifies that a household's strategy is
// still the index of its role, and that the extra part of the table is sorted and
// holds the built-in names and every registered one, used or not.
//
// Guard: the households of every golden are created with strategy = role; moving
// that index would move every strategy call.
func TestStrategyTable_RolesKeepTheirIndex(t *testing.T) {
	t.Parallel()
	w, _ := intWorld(t, WithStrategy("spare", DefaultStrategy{}))
	w.eachLive(func(a *agent) {
		if a.strategy != int(a.role) {
			t.Errorf("agent %d strategy = %d, want its role %d", a.id, a.strategy, a.role)
		}
	})
	if want := []string{"default", "directed", "script", "spare"}; !slices.Equal(w.ext.extraNames, want) {
		t.Errorf("extraNames = %v, want %v", w.ext.extraNames, want)
	}
	if len(w.ext.strategies) != len(w.cfg.Roles)+len(w.ext.extraNames) {
		t.Errorf("strategy table has %d entries, want %d", len(w.ext.strategies), len(w.cfg.Roles)+len(w.ext.extraNames))
	}
}

// TestExternal_Excluded verifies that the spawner leaves an external agent alone
// in the steps that follow its creation: a non-mortal agent with nothing to its
// name is not removed or moved to another role however hungry it is, and a
// decision on it is refused with the reason external_agent.
//
// Requirement: FR-INT-024
func TestExternal_Excluded(t *testing.T) {
	t.Parallel()
	w, rec := intWorld(t)
	id, err := w.AddAgent(NewAgent{Region: regLowland, Role: 0})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	for i := 0; i < 8; i++ {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	if w.agents[id].weight != 1 || !w.agents[id].external {
		t.Errorf("the external agent was removed or changed: %+v", w.agents[id])
	}
	for _, e := range of[AgentBankrupted](rec) {
		if e.Agent == id {
			t.Errorf("the external agent was removed by %v", e.Cause)
		}
	}
	for _, e := range of[RoleChanged](rec) {
		if e.Agent == id {
			t.Errorf("the external agent changed role: %+v", e)
		}
	}
	if got := w.refusal(regLowland, &Decision{Kind: DecideRemove, Agent: id, Cause: CauseConsumer}); got != ReasonExternalAgent {
		t.Errorf("refusal of a removal of a non-mortal external agent = %q, want %q", got, ReasonExternalAgent)
	}
}

// TestRemoveAgent_Order verifies the removal of an external agent, mortal or not:
// its money and goods go to the estate, AgentBankrupted with the cause consumer
// comes before EstateOpened, the ID is not reused, and a second removal, a
// household, a pseudo-agent and an unknown ID are refused with their own errors.
//
// Requirement: FR-INT-029, FR-INT-030
func TestRemoveAgent_Order(t *testing.T) {
	t.Parallel()
	for _, mortal := range []bool{false, true} {
		w, rec := intWorld(t)
		id, err := w.AddAgent(NewAgent{Region: regLowland, Role: 0, Mortal: mortal, Money: 500, Stock: []StockItem{{Commodity: 1, Quantity: 6}}})
		if err != nil {
			t.Fatalf("AddAgent() = %v, want nil", err)
		}
		estate := w.estates.account(regLowland)
		poolBefore := w.estates.pool(w.ledger, regLowland)
		n := len(rec.events)
		if err := w.RemoveAgent(id); err != nil {
			t.Fatalf("RemoveAgent(mortal %v) = %v, want nil", mortal, err)
		}
		got := rec.events[n:]
		if len(got) != 2 {
			t.Fatalf("RemoveAgent emitted %d events %v, want AgentBankrupted then EstateOpened", len(got), got)
		}
		bk, ok1 := got[0].(AgentBankrupted)
		eo, ok2 := got[1].(EstateOpened)
		if !ok1 || !ok2 || bk.Cause != CauseConsumer || bk.Agent != id || bk.Money != 500 || bk.GoodsListed != 1 || bk.Tick != w.Tick() || eo.Agent != id {
			t.Errorf("events = %+v, %+v, want AgentBankrupted{Agent %d, consumer, Money 500, GoodsListed 1} then EstateOpened", got[0], got[1], id)
		}
		if pool := w.estates.pool(w.ledger, regLowland); pool != poolBefore+500 {
			t.Errorf("estate pool = %d, want %d", pool, poolBefore+500)
		}
		if held, _, _ := w.ledger.Holding(estate, 1); held != 6 {
			t.Errorf("estate holds %d grain, want 6", held)
		}
		if err := w.Audit(); err != nil {
			t.Errorf("Audit() = %v, want nil", err)
		}
		if next, err := w.AddAgent(NewAgent{Region: regLowland, Role: 0}); err != nil || next != id+1 {
			t.Errorf("AddAgent() after the removal = %d, %v, want the new ID %d", next, err, id+1)
		}
	}
	w, _ := intWorld(t)
	for _, tt := range []struct {
		name string
		id   AgentID
		want error
	}{
		{"household", 4, ErrNotExternal},
		{"estate", 1, ErrUnknownAgent},
		{"never issued", 99, ErrUnknownAgent},
		{"no agent", NoAgent, ErrUnknownAgent},
	} {
		before := bytesOf(t, w)
		if err := w.RemoveAgent(tt.id); !errors.Is(err, tt.want) || errors.Is(err, ErrAgentRemoved) {
			t.Errorf("RemoveAgent(%s %d) = %v, want %v", tt.name, tt.id, err, tt.want)
		}
		if !bytes.Equal(bytesOf(t, w), before) {
			t.Errorf("RemoveAgent(%s) changed the world", tt.name)
		}
	}
	id, _ := w.AddAgent(NewAgent{Region: regLowland, Role: 0})
	if err := w.RemoveAgent(id); err != nil {
		t.Fatalf("RemoveAgent() = %v, want nil", err)
	}
	if err := w.RemoveAgent(id); !errors.Is(err, ErrAgentRemoved) || !errors.Is(err, ErrUnknownAgent) {
		t.Errorf("RemoveAgent of a removed agent = %v, want ErrAgentRemoved (and ErrUnknownAgent)", err)
	}
	if err := w.RemoveAgent(id + 50); !errors.Is(err, ErrUnknownAgent) {
		t.Errorf("RemoveAgent of an ID never issued = %v, want ErrUnknownAgent", err)
	}
}

// TestRemoveAgent_EstateOverflow verifies the probe: when the estate cannot take
// the agent's money without passing MaxMoney, the call is refused with ErrOverflow
// and changes nothing, the agent staying in the world.
//
// Requirement: FR-INT-029, FR-INT-002
func TestRemoveAgent_EstateOverflow(t *testing.T) {
	t.Parallel()
	w, _ := intWorld(t)
	id, err := w.AddAgent(NewAgent{Region: regLowland, Role: 0, Money: 1_000})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	estate := w.estates.account(regLowland)
	src, _ := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if err := w.ledger.Transfer(src, estate, int64(MaxMoney-w.estates.pool(w.ledger, regLowland))-500, ledger.Initial); err != nil {
		t.Fatalf("setup: filling the estate: %v", err)
	}
	before := bytesOf(t, w)
	if err := w.RemoveAgent(id); !errors.Is(err, ErrOverflow) {
		t.Fatalf("RemoveAgent() = %v, want ErrOverflow", err)
	}
	if !bytes.Equal(bytesOf(t, w), before) || w.agents[id].weight != 1 {
		t.Errorf("a refused RemoveAgent changed the world")
	}
}
