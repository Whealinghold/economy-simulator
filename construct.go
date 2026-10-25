package economysim

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is steps 5 to 10 of the construction of a world (spec-world.md 5.1;
// plan-phase-1-11.md Phase 3 deliverable 2): the estate accounts, the agent ID
// space, the household agents with their accounts, money, goods and beliefs, the
// creation draws, the audit, and the logs and events. NewWorld (world.go) does
// steps 1 to 4 and 6 and calls construct. The application of the warm start is
// construct_warm.go's (D191): fundAgent and warm call it.
//
// Nothing here takes a random number before the creation draws, and the draws
// are the two words of each agent's own stream (FR-WLD-034, 062).

// liveCounts are the live counters of the world, kept so that a log line or a
// report reads them instead of scanning the agents (WLD 16.3): the agents, the
// households (the sum of their weights) and the agents of each (region, role),
// at index region * roles + role.
type liveCounts struct {
	agents, households int64
	inRole             []int64
}

// mustBuild panics when the ledger refuses a step of construction. A validated
// configuration cannot make it refuse (spec-world.md 5.1.1), so a refusal is a
// violated invariant and not an input.
func mustBuild(err error, step string) {
	if err != nil {
		invariant("the ledger refused " + step + " at construction: " + err.Error())
	}
}

// finishConstruction maps the result of the audit of step 9 to the error of
// NewWorld: nil to nil, otherwise an error that wraps ErrAuditFailed and the
// ledger's error (FR-WLD-036). It is pure, so the mapping is tested alone: the
// real audit cannot be made to fail from the root (D190).
func finishConstruction(auditErr error) error {
	if auditErr == nil {
		return nil
	}
	return fmt.Errorf("audit at construction: %w: %w", ErrAuditFailed, auditErr)
}

// builder is what construction passes between its steps. It is dropped when
// construct returns.
type builder struct {
	w     *World
	plans []groupPlan
	// seeds is the belief-seed scratch, sized for the largest tradable set.
	seeds []beliefSeed
	// spawned are the AgentSpawned events of the initial agents, in ascending
	// ID, built only when an observer was supplied (observability-and-logging.md
	// 4.5).
	spawned []AgentSpawned
}

// construct runs steps 7 to 10 of spec-world.md 5.1 on a world that has its
// ledger, its tables and its stream: the estate accounts, the agents, the warm
// start, the creation draws, the audit and the events and logs. It returns the
// error of the warm start, of the audit or of both, which are the only ways it
// can fail.
func (w *World) construct(warnings []Warning) error {
	b := w.beginConstruction()
	b.createAgents()
	if err := b.warm(); err != nil {
		return err
	}
	b.draws()
	return b.finish(warnings)
}

// beginConstruction opens the estate accounts and allocates what the agents need
// (steps 4 and 5 of spec-world.md 5.2), and returns the builder that carries the
// construction on.
func (w *World) beginConstruction() *builder {
	e, err := openEstates(w.ledger, len(w.cfg.Regions))
	mustBuild(err, "the opening of the estate accounts")
	w.estates = e
	w.allocate()
	widest := 0
	for i := range w.roles.rows {
		widest = max(widest, len(w.roles.rows[i].tradable))
	}
	return &builder{w: w, plans: groupPlans(&w.cfg), seeds: make([]beliefSeed, widest)}
}

// finish is steps 9 and 10: the audit with WithAudit, then the logs and events.
func (b *builder) finish(warnings []Warning) error {
	w := b.w
	w.finishScratch()
	var auditErr error
	if w.audit {
		auditErr = w.ledger.Audit()
	}
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err := finishConstruction(auditErr); err != nil {
		return err
	}
	b.announce(warnings)
	return nil
}

// finishScratch builds the scratch that depends on the account list and the agent
// table once both are complete: the receipt tally and the welfare flows. A restore
// builds it after the sections are loaded, as the construction does (plan 2-2 D252).
func (w *World) finishScratch() {
	w.receipts.v = make([]int64, w.ledger.Len())
	w.welfare = newWelfareScratch(&w.cfg, len(w.agents))
}

// allocate builds what the agents and the first step need and the constructor
// has not: the estates' agent slots, the market table, the counters, the
// scratch of D154 and the per-region tables.
func (w *World) allocate() {
	regions, roles := len(w.cfg.Regions), len(w.cfg.Roles)
	w.agents = make([]agent, regions) // the estate pseudo-agents have no household
	w.state.nextAgent = AgentID(regions)
	w.live = liveCounts{inRole: make([]int64, regions*roles)}
	w.bp = beliefParams(w.cfg.World)
	w.clr = &clearScratch{}
	w.prod = newProductionScratch(&w.cfg, &w.recipes)
	w.refused = newRefusalTally(regions, len(w.cfg.Recipes))
	w.reads = newReadBuf(len(w.cfg.Commodities))
	w.views = newViewScratch(&w.cfg, &w.roles, &w.recipes)
	w.tally = newEntryTally(regions)
	w.runnable = make([][]RecipeID, regions)
	w.markets = newMarketTable(&w.cfg, w.clr, seedPrices(&w.cfg))
	w.tk = newTickScratch(&w.cfg)
	w.pr = newProductionTick(&w.cfg)
	w.oc = newOutcomeTick(&w.cfg)
	w.ex = newExitTick(&w.cfg, w.obs.supplied)
}

// seedPrices returns the function that gives the seed price of a market: the
// entry of the warm start when there is one, else the commodity's effective
// reference price in the region (spec-beliefs.md 8.1, FR-BEL-024, FR-CLR-020,
// spec-regions.md 5.2.2). A warm-start price replaces the seed only.
func seedPrices(c *Config) func(RegionID, CommodityID) Price {
	warm := warmPriceTable(c)
	return func(r RegionID, k CommodityID) Price {
		if p := warm[int(r)*len(c.Commodities)+int(k)]; p > 0 {
			return p
		}
		return c.refPrice(r, k)
	}
}

// createAgents creates the household agents of the groups in canonical order
// (region, then role) and, within a group, in ascending index, so the IDs are
// the order of spec-world.md 5.3 and the accounts the order of 5.2 row 6
// (FR-WLD-032, 033). No random number is taken.
func (b *builder) createAgents() {
	for pi := range b.plans {
		p := &b.plans[pi]
		p.first = int(b.w.state.nextAgent)
		for i, weight := range p.weights {
			b.createAgent(p, i, weight)
		}
	}
}

// createAgent creates agent i of group p with the weight: its ledger account
// (the world opens it, row 7's newAgent has no ledger), its money and goods and
// its beliefs at the seed price (spec-agents.md 3.5.1, spec-world.md 5.4.2).
func (b *builder) createAgent(p *groupPlan, i int, weight int64) {
	w := b.w
	id := w.state.nextAgent
	if int(id) != len(w.agents) {
		invariant("the agent ID space is not dense")
	}
	acct, err := w.ledger.Open(ledger.KindAgent, int(id), int(p.region))
	mustBuild(err, "the opening of an agent account")
	a := newAgent(agentInit{id: id, account: acct, role: p.role, weight: weight, strategy: int(p.role)},
		w.roles.row(p.role), &w.recipes)
	money := b.fundAgent(&a, p, i)
	b.seedBeliefs(&a, p.region)
	w.agents = append(w.agents, a)
	w.state.nextAgent++
	cell := int(p.region)*len(w.cfg.Roles) + int(p.role)
	w.live.agents++
	w.live.households += weight
	w.live.inRole[cell]++
	w.state.hadAgent[cell] = true
	if w.obs.supplied {
		b.spawned = append(b.spawned, AgentSpawned{Agent: id, Region: p.region, Role: p.role, Weight: weight, StartMoney: money})
	}
}

// fundAgent gives agent i of group p its money and goods, by the reason initial,
// and returns the money (spec-agents.md 3.5.1, FR-AGT-005): weight times
// StartMoney and times each StartStock quantity with unstated lot keys, or, for
// a group with an entry in the warm start, the entry's (fundWarm, 5.6.1 item 3).
func (b *builder) fundAgent(a *agent, p *groupPlan, i int) Money {
	if p.warm != nil {
		return b.fundWarm(a, p, i)
	}
	w := b.w
	money, stock, ok := defaultEndowment(&w.cfg, &w.cfg.Roles[p.role], a.weight)
	if !ok {
		invariant("the initial endowment of an agent overflows after validation")
	}
	mustBuild(fundInitial(w.ledger, a.account, money, stock), "the initial endowment of an agent")
	return money
}

// seedBeliefs gives the agent one belief per commodity of its role's tradable
// set, at the seed price of its region's market clamped to the bounds of its
// unit cost, with the width BeliefWidthInit (spec-beliefs.md 8.2, FR-BEL-022).
// The unit cost is the cost anchor, which is unknown (0) before the first
// anchors call of P1.2. No random number is taken.
func (b *builder) seedBeliefs(a *agent, region RegionID) {
	w := b.w
	row := w.roles.row(a.role)
	seeds := b.seeds[:len(row.tradable)]
	for i, c := range row.tradable {
		seeds[i] = beliefSeed{
			Ref:      int64(w.refPrice(region, c)),
			UnitCost: w.prod.unitCost(region, c),
			Empty:    true,
			Price:    int64(w.markets.reports[w.markets.index(region, c)].Price),
		}
	}
	reseedBeliefs(a, row, nil, seeds, w.bp)
}

// draws takes the two creation words of every household agent, in ascending ID,
// each from the stream of its own agent (spec-world.md 5.7, FR-WLD-034, 066):
// the target ratio and then the residue. They are the only words construction
// takes (FR-WLD-062).
func (b *builder) draws() {
	w := b.w
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		role := &w.cfg.Roles[a.role]
		w.rnd.keyed(siteConstruction, 0, int64(a.id))
		drawCreation(w.rnd, a, role.TargetRatio, role.TargetSpread)
	}
}

// namesOf builds the names table of a canonical configuration: the string ID of
// every typed ID of the slice. Every call returns slices of its own, so an event
// or a snapshot owns what it carries (FR-OBS-002).
func namesOf(c *Config) Names {
	n := Names{
		Regions:     make([]string, len(c.Regions)),
		Commodities: make([]string, len(c.Commodities)),
		Roles:       make([]string, len(c.Roles)),
		Recipes:     make([]string, len(c.Recipes)),
		Seasons:     make([]string, len(c.Calendar.Seasons)),
	}
	for i := range c.Regions {
		n.Regions[i] = c.Regions[i].ID
	}
	for i := range c.Commodities {
		n.Commodities[i] = c.Commodities[i].ID
	}
	for i := range c.Roles {
		n.Roles[i] = c.Roles[i].ID
	}
	for i := range c.Recipes {
		n.Recipes[i] = c.Recipes[i].ID
	}
	for i := range c.Calendar.Seasons {
		n.Seasons[i] = c.Calendar.Seasons[i].ID
	}
	return n
}

// modulesOn returns the lower-case names of the modules that are on, for the
// construction summary. The slice has no module, so it is empty and not nil; a
// module's change appends its name.
func modulesOn(*Config) []string { return []string{} }

// announce is step 10 (spec-world.md 5.8): the logs, then the events. Each
// configuration warning is one Warn line, then one Info line summarises the
// world; WorldCreated and the AgentSpawned events of the initial agents, in
// ascending ID, follow, only when an observer was supplied (FR-WLD-037, FR-OBS-010,
// 076). No line is in a loop over agents.
func (b *builder) announce(warnings []Warning) {
	w := b.w
	ctx := context.Background()
	w.logWarnings(ctx, warnings)
	b.logWarm(ctx)
	if w.log.Enabled(ctx, slog.LevelInfo) {
		w.log.LogAttrs(ctx, slog.LevelInfo, "world created", w.summary()...)
	}
	if !w.obs.supplied {
		return
	}
	_, present := warmCountsOf(&w.cfg)
	w.obs.deliver(WorldCreated{Seed: w.state.seed, ConfigHash: w.hash, WarmStart: present, Names: namesOf(&w.cfg)})
	for _, e := range b.spawned {
		w.obs.deliver(e)
	}
}

// logWarnings logs each configuration warning at Warn, once, with its rule and
// path: at construction and at a restore alike.
func (w *World) logWarnings(ctx context.Context, warnings []Warning) {
	if !w.log.Enabled(ctx, slog.LevelWarn) {
		return
	}
	for _, wn := range warnings {
		w.log.LogAttrs(ctx, slog.LevelWarn, "configuration warning",
			slog.String(logkeys.Rule, wn.Rule), slog.String(logkeys.Path, strings.Join(wn.Path, ".")))
	}
}

// summary is the attributes of the Info line "world created" (spec-world.md
// 5.8.1): the seed, the first 8 bytes of the hash, the counts of the
// configuration and of the agents, the modules that are on and whether a warm
// start is present, and its counts when it is, then the edges and the components
// of the region graph (spec-regions.md 10.2).
func (w *World) summary() []slog.Attr {
	warm, present := warmCountsOf(&w.cfg)
	attrs := []slog.Attr{
		slog.Uint64(logkeys.Seed, w.state.seed),
		slog.String(logkeys.ConfigHash, hex.EncodeToString(w.hash[:8])),
		slog.Int(logkeys.Commodities, len(w.cfg.Commodities)),
		slog.Int(logkeys.Recipes, len(w.cfg.Recipes)),
		slog.Int(logkeys.Roles, len(w.cfg.Roles)),
		slog.Int(logkeys.Regions, len(w.cfg.Regions)),
		slog.Int64(logkeys.Agents, w.live.agents),
		slog.Int64(logkeys.Households, w.live.households),
		slog.Any(logkeys.Modules, modulesOn(&w.cfg)),
		slog.Bool(logkeys.WarmStart, present),
	}
	if present {
		attrs = append(attrs, slog.Int64(logkeys.Prices, warm.prices), slog.Int64(logkeys.Wages, warm.wages),
			slog.Int64(logkeys.Groups, warm.groups), slog.Int64(logkeys.Jobs, warm.jobs))
	}
	// The count the warning graph_not_strongly_connected cannot carry (a Warning
	// has a path and a rule only): 0 with one region, as Snapshot.Components is.
	components := 0
	if len(w.cfg.Regions) > 1 {
		components = w.geo.components()
	}
	attrs = append(attrs, slog.Int(logkeys.Edges, len(w.cfg.Edges)), slog.Int(logkeys.Components, components))
	return attrs
}
