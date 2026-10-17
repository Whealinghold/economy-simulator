package economysim

import (
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/household"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is the test-only harness of the core view filler and of the
// admission of a plan (plan-phase-1-7.md Phase 7, A5): a real ledger, the real
// role, recipe and yield tables, a production scratch and the per-market state
// of a viewCtx, with a few agents. The external tests reach it through its
// exported names; nothing here is exported from production code
// (architecture.md 10.2). The loops over agents and the clearing are the world
// row's, so the harness only fills a view and admits a plan.

// ViewHarness is a small world: tables, ledger, market state and agents.
type ViewHarness struct {
	// Cfg is the canonical configuration the harness was built from.
	Cfg    *Config
	rt     recipeTable
	roles  roleTable
	l      *ledger.Ledger
	prod   *productionScratch
	yt     yieldTable
	sh     yieldShocks
	ctx    viewCtx
	sc     *viewScratch
	agents []*agent
	v      AgentView
	clr    clearScratch
	tally  receiptTally
}

// NewViewHarness builds the harness for a configuration. Every market starts at
// its commodity's reference price with an empty price window, every belief at
// the reference price, and the tick is 0.
func NewViewHarness(t testing.TB, raw Config) *ViewHarness {
	t.Helper()
	cfg := raw.canonical()
	l, err := newLedger(cfg)
	if err != nil {
		t.Fatalf("newLedger error = %v, want nil", err)
	}
	if err := l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction error = %v, want nil", err)
	}
	h := &ViewHarness{Cfg: &cfg, l: l}
	h.rt, h.roles, h.yt = newRecipeTable(h.Cfg), newRoleTable(h.Cfg), newYieldTable(h.Cfg)
	h.prod = newProductionScratch(h.Cfg, &h.rt)
	m, err := cfg.Calendar.At(0)
	if err != nil {
		m = Moment{Season: -1}
	}
	h.ctx = viewCtx{
		cfg: h.Cfg, roles: &h.roles, recipes: &h.rt, l: l, prod: h.prod, yields: &h.yt, shocks: &h.sh,
		bp: beliefParams(cfg.World), moment: m,
	}
	nc := len(cfg.Commodities)
	for range cfg.Regions {
		for c := range nc {
			h.ctx.reports = append(h.ctx.reports, seedReport(cfg.Commodities[c].ReferencePrice))
			h.ctx.windows = append(h.ctx.windows, beliefs.NewWindow(int(cfg.World.PriceWindow), MaxPrice))
		}
	}
	h.sc = newViewScratch(h.Cfg, &h.roles, &h.rt)
	return h
}

func (h *ViewHarness) commodity(t testing.TB, name string) CommodityID {
	t.Helper()
	id, ok := h.Cfg.commodityID(name)
	if !ok {
		t.Fatalf("commodity %q not in the configuration", name)
	}
	return id
}

func (h *ViewHarness) region(t testing.TB, name string) RegionID {
	t.Helper()
	id, ok := h.Cfg.regionID(name)
	if !ok {
		t.Fatalf("region %q not in the configuration", name)
	}
	return id
}

// Spawn adds an agent of a role and weight in a region, with an empty account,
// beliefs at the reference prices, its role's target ratio and no job. It
// returns the agent's index, which is also its AgentID.
func (h *ViewHarness) Spawn(t testing.TB, role, region string, weight int64) int {
	t.Helper()
	rid, ok := h.Cfg.roleID(role)
	if !ok {
		t.Fatalf("role %q not in the configuration", role)
	}
	acct, err := h.l.Open(ledger.KindAgent, len(h.agents), int(h.region(t, region)))
	if err != nil {
		t.Fatalf("Open error = %v, want nil", err)
	}
	row := h.roles.row(rid)
	a := newAgent(agentInit{id: AgentID(len(h.agents)), account: acct, role: rid, weight: weight}, row, &h.rt)
	seeds := make([]beliefSeed, len(row.tradable))
	for i, c := range row.tradable {
		ref := int64(h.Cfg.Commodities[c].ReferencePrice)
		seeds[i] = beliefSeed{Ref: ref, Empty: true, Price: ref}
	}
	reseedBeliefs(&a, row, nil, seeds, h.ctx.bp)
	a.target = h.Cfg.Roles[rid].TargetRatio
	h.agents = append(h.agents, &a)
	return len(h.agents) - 1
}

// Give puts n units of a commodity in the agent's account from the Initial
// boundary.
func (h *ViewHarness) Give(t testing.TB, i int, commodity string, n int64) {
	t.Helper()
	c := h.commodity(t, commodity)
	from, ok := h.l.Boundary(ledger.Initial, int(c))
	if !ok {
		t.Fatalf("no Initial boundary for %s", commodity)
	}
	if err := h.l.Move(from, h.agents[i].account, int(c), n, ledger.Initial); err != nil {
		t.Fatalf("Move(%d %s) error = %v, want nil", n, commodity, err)
	}
}

// Fund gives the agent money from the Initial boundary.
func (h *ViewHarness) Fund(t testing.TB, i int, amount int64) {
	t.Helper()
	if err := fundInitial(h.l, h.agents[i].account, Money(amount), nil); err != nil {
		t.Fatalf("fundInitial error = %v, want nil", err)
	}
}

// SetHunger sets the agent's hunger and unfed households.
func (h *ViewHarness) SetHunger(i int, hunger, unfed int64) {
	h.agents[i].hunger, h.agents[i].unfed = hunger, unfed
}

// PushHistory records one welfare tick in the agent's ring.
func (h *ViewHarness) PushHistory(i int, income, cost int64, working bool) {
	h.agents[i].history.Push(welfare.Record{Income: income, Cost: cost, Working: working})
}

// SetBelief sets the agent's belief for the k-th commodity of its tradable set.
func (h *ViewHarness) SetBelief(i, k int, centre, width int64) {
	h.agents[i].beliefs[k] = beliefs.Belief{Centre: centre, Width: width}
}

// SetJob gives the agent a job in progress, by hand.
func (h *ViewHarness) SetJob(t testing.TB, i int, recipe string, remaining, executions int64) {
	t.Helper()
	rid, ok := h.Cfg.recipeID(recipe)
	if !ok {
		t.Fatalf("recipe %q not in the configuration", recipe)
	}
	h.agents[i].job = job{recipe: rid, remaining: remaining, executions: executions}
}

// SetDirective sets the agent's pending directive.
func (h *ViewHarness) SetDirective(i int, d *Directive) { h.agents[i].directive = d }

// SetReport sets the market report of a market.
func (h *ViewHarness) SetReport(t testing.TB, region, commodity string, r MarketReport) {
	t.Helper()
	rep, _ := h.ctx.market(h.region(t, region), h.commodity(t, commodity))
	*rep = marketReport{
		Price: Price(r.Price), Age: r.Age, BestBid: Price(r.BestBid), BestAsk: Price(r.BestAsk),
		HasBestBid: r.HasBestBid, HasBestAsk: r.HasBestAsk,
	}
}

// PushWindow adds a price to the price window of a market.
func (h *ViewHarness) PushWindow(t testing.TB, region, commodity string, price int64) {
	t.Helper()
	_, w := h.ctx.market(h.region(t, region), h.commodity(t, commodity))
	w.Push(price)
}

// Anchors fills the cost anchor table with one observed wage in every region, at
// the published prices, with every recipe of every role runnable everywhere. A
// wage of 0 is an unknown wage.
func (h *ViewHarness) Anchors(wage int64) {
	nc, ng := len(h.Cfg.Commodities), len(h.Cfg.Regions)
	prices, wages := make([]int64, ng*nc), make([]int64, ng)
	var run []RecipeID
	for g := range ng {
		wages[g] = wage
		for c := range nc {
			prices[g*nc+c] = int64(h.ctx.reports[g*nc+c].Price)
		}
	}
	for i := range h.roles.rows {
		run = append(run, h.roles.rows[i].recipes...)
	}
	runnable := make([][]RecipeID, ng)
	for g := range runnable {
		runnable[g] = run
	}
	h.prod.anchors(prices, wages, runnable, &h.yt, &h.sh, h.ctx.moment.Season)
}

// AnchorOf reads the cost anchor table.
func (h *ViewHarness) AnchorOf(region RegionID, c CommodityID) int64 {
	return h.prod.unitCost(region, c)
}

// Fill fills the harness's view for an agent and returns it. The view is the
// harness's, valid until the next Fill.
func (h *ViewHarness) Fill(i int) (*AgentView, error) {
	return &h.v, h.sc.fillCore(&h.v, h.agents[i], &h.ctx)
}

// Scratch returns the scratch the filler works in.
func (h *ViewHarness) Scratch() *viewScratch { return h.sc }

// Plan returns the empty, reused plan of the scratch.
func (h *ViewHarness) Plan() *TradePlan { return h.sc.tradePlan() }

// MaxExecutions reads the count a start of the recipe would use for the agent
// now, directly from the production scratch.
func (h *ViewHarness) MaxExecutions(t testing.TB, i int, recipe string) (int64, Reason) {
	t.Helper()
	rid, ok := h.Cfg.recipeID(recipe)
	if !ok {
		t.Fatalf("recipe %q not in the configuration", recipe)
	}
	a := h.agents[i]
	acct, err := h.l.Account(a.account)
	if err != nil {
		t.Fatalf("Account error = %v, want nil", err)
	}
	e, has, _ := h.yt.effective(&h.sh, RegionID(acct.Region), rid, h.ctx.moment.Season, h.Cfg.Recipes[rid].Sited)
	k, reason, err := h.prod.maxExecutions(h.l, &startReq{recipe: rid, account: a.account, principal: a.id, weight: a.weight, effective: e, hasProfile: has})
	if err != nil {
		t.Fatalf("maxExecutions error = %v, want nil", err)
	}
	return k, reason
}

// Held reads the units of a commodity the agent holds.
func (h *ViewHarness) Held(t testing.TB, i int, commodity string) int64 {
	t.Helper()
	held, _, err := h.l.Holding(h.agents[i].account, int(h.commodity(t, commodity)))
	if err != nil {
		t.Fatalf("Holding error = %v, want nil", err)
	}
	return held
}

// Balance reads the agent's balance and the money its reservations earmark.
func (h *ViewHarness) Balance(t testing.TB, i int) (balance, reserved int64) {
	t.Helper()
	v, err := h.l.Account(h.agents[i].account)
	if err != nil {
		t.Fatalf("Account error = %v, want nil", err)
	}
	return v.Balance, v.Reserved
}

// Directive reads the agent's pending directive, as stored.
func (h *ViewHarness) Directive(i int) *Directive { return h.agents[i].directive }

// Admission is the result of the admission of one intent.
type Admission struct {
	Intent Intent
	Order  Order
	Reason Reason
	OK     bool
}

// Admit normalises each intent of the plan, converts it with orderOf and admits
// it, in plan order, into a fresh book of its market over the real ledger with
// the facts the world would build (spec-agents.md 6.5.2; the Tradable flag is
// row.isTradable). The reservations stay in the ledger.
func (h *ViewHarness) Admit(t testing.TB, i int, plan *TradePlan) []Admission {
	t.Helper()
	a := h.agents[i]
	acct, err := h.l.Account(a.account)
	if err != nil {
		t.Fatalf("Account error = %v, want nil", err)
	}
	region := RegionID(acct.Region)
	row := h.roles.row(a.role)
	books := make([]*book, len(h.Cfg.Commodities))
	var out []Admission
	for _, in := range plan.Intents {
		normalise(&in)
		o := orderOf(a, in)
		known := int(in.Commodity) >= 0 && int(in.Commodity) < len(books)
		if known && books[in.Commodity] == nil {
			books[in.Commodity] = newBook(region, in.Commodity, 0, &h.clr)
		}
		var b *book
		if known {
			b = books[in.Commodity]
		} else {
			b = newBook(region, in.Commodity, 0, &h.clr)
		}
		f := orderFacts{
			Account: a.account, AgentKnown: true, SideAllowed: true, CommodityKnown: known,
			Tradable: row.isTradable(in.Commodity), Tally: &h.tally,
		}
		r, ok := b.admit(h.l, f, o)
		out = append(out, Admission{Intent: in, Order: o, Reason: r, OK: ok})
	}
	return out
}

// RequestAdmission is the first check of a job request for the agent: the
// recipe against the role's list (spec-production.md 4.1).
func (h *ViewHarness) RequestAdmission(i int, recipe RecipeID) Reason {
	return admit(h.roles.row(h.agents[i].role).recipes, recipe)
}

// Start starts the recipe for the agent with the given maximum number of
// executions, as the world does after Work, and returns the executions started
// and the reason when there are none. The agent's account is changed.
func (h *ViewHarness) Start(t testing.TB, i int, recipe RecipeID, maximum int64) (int64, Reason) {
	t.Helper()
	a := h.agents[i]
	acct, err := h.l.Account(a.account)
	if err != nil {
		t.Fatalf("Account error = %v, want nil", err)
	}
	e, has, _ := h.yt.effective(&h.sh, RegionID(acct.Region), recipe, h.ctx.moment.Season, h.Cfg.Recipes[recipe].Sited)
	row := h.roles.row(a.role)
	off, ok := residueOffset(row.recipes, &h.rt, recipe)
	if !ok {
		t.Fatalf("recipe %d is not on the role's list", recipe)
	}
	res, err := h.prod.start(h.l, &a.job, a.residues[off:off+len(h.rt.row(recipe).outputs)], &startReq{
		recipe: recipe, account: a.account, principal: a.id, weight: a.weight, maximum: maximum, effective: e, hasProfile: has,
	})
	if err != nil {
		t.Fatalf("start error = %v, want nil", err)
	}
	return res.k, res.reason
}

// Season is the season index of the harness's tick, -1 with no seasons.
func (h *ViewHarness) Season() int { return h.ctx.moment.Season }

// BeliefParamsFlat is the belief parameters of the configuration as the filler
// reads them.
func (h *ViewHarness) BeliefParamsFlat() BeliefTunables { return beliefTunablesOf(h.ctx.bp) }

// PublishedPrice is the price a market published, as the filler reads it.
func (h *ViewHarness) PublishedPrice(region RegionID, c CommodityID) Price {
	rep, _ := h.ctx.market(region, c)
	return rep.Price
}

// BeliefParams is the belief parameters as the filler holds them.
func (h *ViewHarness) BeliefParams() beliefs.Params { return h.ctx.bp }

// ResetWindow empties the price window of a market.
func (h *ViewHarness) ResetWindow(t testing.TB, region, commodity string) {
	t.Helper()
	_, w := h.ctx.market(h.region(t, region), h.commodity(t, commodity))
	*w = beliefs.NewWindow(int(h.Cfg.World.PriceWindow), MaxPrice)
}

// TradeRow is the first pass of the default strategy for one commodity that is
// a candidate for an order: the tiers of a bid, the ask quantity and the limit
// drawn, before any money is allocated.
type TradeRow struct {
	Commodity              CommodityID
	Bid                    bool
	T1, T2, T3, Ask, Limit int64
	WindowPos              int64
	UseSourced             bool // the Target comes from UseTgt, not from the output target
}

// TradeFirstPass runs the first pass of DefaultStrategy.Trade over v with the
// stream the cohort tests give Trade (seed, 1234), and returns the candidate
// rows, the planned recipe, the upkeep cost and the household parameters, so a
// test can compute the allocation of the money by its closed form.
func (h *ViewHarness) TradeFirstPass(v *AgentView, seed uint64) (rows []TradeRow, rstar int, cost int64, par household.Params) {
	par = householdParams(&v.Tunables)
	t := tradeCtx{v: v, par: par, bp: beliefParamsOf(&v.Tunables.Beliefs), rows: new(tradeScratch).reset(len(v.Commodities))}
	cost, _ = v.upkeepCost()
	t.rstar = t.plannedRecipe()
	rng := rand.New(rand.NewPCG(seed, 1234))
	for i := range v.Commodities {
		t.consider(i, rng)
	}
	for _, r := range t.rows {
		if r.kind == noOrder {
			continue
		}
		u, _, _ := t.upkeepUnits(r.commodity)
		job, out, _ := t.jobAndOutput(r.commodity)
		tg, _ := household.ComputeTargets(household.Need{Upkeep: u, Job: job, OutTgt: out}, par)
		rows = append(rows, TradeRow{Commodity: r.commodity, Bid: r.kind == bidOrder, T1: r.t1, T2: r.t2, T3: r.t3, Ask: r.ask, Limit: r.limit, WindowPos: r.windowPos, UseSourced: tg.UseTgt > 0})
	}
	return rows, t.rstar, cost, par
}
