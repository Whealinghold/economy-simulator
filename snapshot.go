package economysim

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file is the snapshot (spec-observability.md 5; plan-phase-1-11.md Phase 9):
// plain data, a deep copy, a function of the saved state (5.2.1). Every slice is
// in the canonical order of its key (5.1.4); a field at its off value is zero.
// The slice declares the rows it has state for (D146); a module appends its own
// rows and fields at the end. Each field's comment names its source.

// Snapshot is the standing state of a world at a tick boundary. Agents and
// Households are separate counts (a household count is the sum of weights); the
// estates are in neither.
type Snapshot struct {
	// Tick is the tick the next step will run (World.Tick).
	Tick int64
	// Seed is the effective seed (the world's saved state).
	Seed uint64
	// ConfigHash is the configuration hash (World.ConfigHash).
	ConfigHash [32]byte
	// Moment is Calendar.At(Tick), a pure function of the tick.
	Moment Moment
	// Names is the names table, derived from the configuration.
	Names Names
	// Modules lists the modules that are on; empty in the slice.
	Modules []string
	// Agents is the live agents and Households the sum of their weights.
	Agents, Households int64
	// Money is the money of the non-boundary accounts: the households and the
	// estates (escrow and treasuries are 0 in the slice).
	Money Money
	// Regions has one row per region, by RegionID.
	Regions []RegionRow
	// Boundary has one row per boundary account, money first, then goods by
	// commodity, the reasons in registry order: the signed balances of the ledger.
	Boundary []BoundaryRow
	// Pending is the totals that calls between ticks added since the last tick
	// end, in the order of BoundaryFlow.
	Pending []FlowTotal
	// Construction is the construction bucket, and only before the first step
	// (spec-observability.md 5.2.4).
	Construction []FlowTotal
	// Roles has one row per (region, role), by RegionID then RoleID.
	Roles []RoleRow
	// Markets has one row per (region, commodity), in that order.
	Markets []MarketRow
	// Jobs has one row per (region, recipe) with at least one job in progress.
	Jobs []JobRow
	// Edges has one row per edge whose status is not the default, by EdgeID
	// (spec-regions.md 10.5).
	Edges []EdgeRow
	// Components is the number of strongly connected components of the graph as it
	// is now, and 0 when the world has one region, so that the header of an off
	// golden does not move (architecture.md 2.10.5).
	Components int64
}

// EdgeRow reports an edge whose status is not the default.
type EdgeRow struct {
	// From and To are the ends of the edge.
	From, To RegionID
	// Closed says the edge is closed.
	Closed bool
	// FactorBP is the length factor of the status in basis points.
	FactorBP int64
}

// RegionRow reports one region. MoneyContract and MoneyTreasury are appended
// with the modules that own them (D146).
type RegionRow struct {
	// Region is the region; Outside is true for an outside node (none in the
	// slice).
	Region  RegionID
	Outside bool
	// Agents and Households are the agents of the region and their weights.
	Agents, Households int64
	// MoneyHouseholds is the money of the agents of the region, from the ledger.
	MoneyHouseholds Money
	// MoneyPool is the money of the region's estate account (FR-EST-013).
	MoneyPool Money
	// EntryBudget is MoneyPool * EntryPoolShare / 10 000: advisory.
	EntryBudget Money
}

// BoundaryRow is the balance of one boundary account: how much has entered
// (negative, a faucet) or left (a sink). Commodity is NoCommodity for money.
type BoundaryRow struct {
	Reason    Reason
	Kind      Kind
	Commodity CommodityID
	Balance   int64
}

// FlowTotal is one non-zero flow total of a region, a reason and a thing.
type FlowTotal struct {
	Region    RegionID
	Reason    Reason
	Kind      Kind
	Commodity CommodityID
	Total     int64
}

// RoleRow reports the agents of one role in one region: the counts, the
// welfare and spawner signals, each a scan of the saved agent state.
type RoleRow struct {
	Region RegionID
	Role   RoleID
	// Agents and Households are the agents and their weights.
	Agents, Households int64
	// Unfed is the sum of the agents' Unfed (households, not weights);
	// UnfedAgents the agents with Unfed above 0.
	Unfed, UnfedAgents int64
	// MaxHunger and MaxPrivation are the largest over the agents.
	MaxHunger, MaxPrivation int64
	// HeldByFloor is the households the work floor holds to work at the boundary,
	// over the agents with no job in progress (FR-WEL-020).
	HeldByFloor int64
	// MeanRatioBP is the household-weighted mean welfare ratio over the
	// RatioHouseholds whose ratio exists; HasMeanRatio is false if none.
	MeanRatioBP     int64
	HasMeanRatio    bool
	RatioHouseholds int64
	// WageObserved and PremiumBP are the observed wage and premium of the role.
	WageObserved Money
	HasWage      bool
	PremiumBP    int64
	HasPremium   bool
	// Window is the role's effective welfare window in ticks.
	Window int64
	// ExpectedRatioBP, Viable and HasExpected are the spawner's signals.
	ExpectedRatioBP int64
	Viable          bool
	HasExpected     bool
	// ClampedHouseholds is the households whose welfare ring holds a field at
	// the clamp; appended last.
	ClampedHouseholds int64
}

// MarketState is the report of one market and its price window: the type the
// snapshot and the Stage 2 query Market share, so they cannot disagree. It is
// saved state.
type MarketState struct {
	Region    RegionID
	Commodity CommodityID
	// Price is the published price, Age its age in ticks, Volume the units
	// traded and UnmatchedBid and UnmatchedAsk the unfilled units of the last
	// clearing.
	Price                      Price
	Age                        int64
	Volume                     Quantity
	UnmatchedBid, UnmatchedAsk Quantity
	// BestBid and BestAsk are valid when HasBid and HasAsk.
	BestBid, BestAsk Price
	HasBid, HasAsk   bool
	// WindowLen is the prices in the window; the mean, lowest and highest are 0
	// when it is empty.
	WindowLen                         int64
	WindowMean, WindowLow, WindowHigh Price
}

// MarketRow reports one (region, commodity).
type MarketRow struct {
	State MarketState
	// HeldByAgents and HeldByEstate are the stock in the households' and the
	// estate's accounts.
	HeldByAgents, HeldByEstate Quantity
	// Lots is the lots of the commodity those accounts hold.
	Lots int64
	// Pending is the pending output of the jobs in progress in the region.
	Pending Quantity
	// EstateAtFloor is the units of the estate's listings priced at the
	// resolution floor, where a markdown cannot lower them (spec-estates.md
	// 2.7.3); at most HeldByEstate. Derived from the saved listings; appended last.
	EstateAtFloor Quantity
	// Beliefs is the beliefs of the region's agents for the commodity; the
	// centres are the mean (half to even), lowest and highest; AtFloor and
	// AtCeiling count the beliefs at the bounds of their range.
	Beliefs                           int64
	MeanCentre, LowCentre, HighCentre Price
	AtFloor, AtCeiling                int64
}

// JobRow reports the jobs in progress of one recipe in one region.
type JobRow struct {
	Region RegionID
	Recipe RecipeID
	// Agents is the agents with a job of the recipe, Executions the sum of
	// their executions and Households the sum of their weights.
	Agents, Executions, Households int64
}

// beliefAcc accumulates the beliefs of one market.
type beliefAcc struct {
	n, atFloor, atCeiling, low, high int64
	sum                              checked.Acc
}

// snapAcc is what one region's agents add up, by commodity and by recipe.
type snapAcc struct {
	held, estate, atFloor, lots, pending []int64
	jobs, execs, hh                      []int64
	bel                                  []beliefAcc
}

func newSnapAcc(commodities, recipes int) snapAcc {
	return snapAcc{
		held: make([]int64, commodities), estate: make([]int64, commodities), atFloor: make([]int64, commodities),
		lots: make([]int64, commodities), pending: make([]int64, commodities),
		jobs: make([]int64, recipes), execs: make([]int64, recipes), hh: make([]int64, recipes),
		bel: make([]beliefAcc, commodities),
	}
}

func (a *snapAcc) reset() {
	clear(a.held)
	clear(a.estate)
	clear(a.atFloor)
	clear(a.lots)
	clear(a.pending)
	clear(a.jobs)
	clear(a.execs)
	clear(a.hh)
	clear(a.bel)
}

// Snapshot returns a deep copy of the standing state at the tick boundary. It
// returns ErrInStep from inside a step and ErrWorldFailed on a failed world; it
// draws no random number and changes no saved state (spec-observability.md 5.2); it
// rewrites only the scratch that P1.2 of the next step rewrites first.
func (w *World) Snapshot() (Snapshot, error) {
	if err := w.guard(); err != nil {
		return Snapshot{}, err
	}
	moment, err := w.cfg.Calendar.At(w.Tick())
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return Snapshot{}, fmt.Errorf("snapshot at tick %d: %w", w.Tick(), err)
	}
	// The cost anchors are derived scratch and are not saved. Derive them from the
	// saved state as P1.2 of the next step will, so that the range a belief is
	// counted against is a function of the saved state and a restored world reports
	// what the original does (FR-CHK-053).
	w.tk.moment = moment
	w.p12Anchors()
	nr, nc := len(w.cfg.Roles), len(w.cfg.Commodities)
	s := Snapshot{
		Tick: w.Tick(), Seed: w.state.seed, ConfigHash: w.hash, Moment: moment,
		Names: namesOf(&w.cfg), Modules: modulesOn(&w.cfg),
		Agents: w.live.agents, Households: w.live.households,
		Regions:  make([]RegionRow, len(w.cfg.Regions)),
		Roles:    make([]RoleRow, len(w.cfg.Regions)*nr),
		Markets:  make([]MarketRow, len(w.cfg.Regions)*nc),
		Jobs:     make([]JobRow, 0, len(w.cfg.Regions)*len(w.cfg.Recipes)),
		Boundary: w.boundaryRows(),
	}
	s.Edges, s.Components = w.edgeRows(), w.componentCount()
	s.Pending = flowTotals(w.ledger.AppendFlows(nil, false))
	if w.ledger.InConstruction() {
		s.Construction = flowTotals(w.ledger.AppendFlows(nil, true))
	}
	acc, makes := newSnapAcc(nc, len(w.cfg.Recipes)), w.makesTable()
	for g := range w.cfg.Regions {
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err := w.snapRegion(&s, RegionID(g), &acc, makes); err != nil {
			return Snapshot{}, fmt.Errorf("snapshot at tick %d: %w", w.Tick(), err)
		}
		s.Money = satMoney(s.Money, s.Regions[g].MoneyHouseholds, s.Regions[g].MoneyPool)
	}
	return s, nil
}

// satMoney adds two amounts to m, saturating.
func satMoney(m, a, b Money) Money { return Money(satAdd(satAdd(int64(m), int64(a)), int64(b))) }

// boundaryRows reads the balance of every boundary account that was opened, in
// the group order of spec-ledger.md 3.4.2: the built-in group (money, then each
// commodity, the reasons in registry order), then the declared module rows by
// the rank of their registry row, each with money before goods and then
// ascending commodity.
func (w *World) boundaryRows() []BoundaryRow {
	declared := ledger.DeclaredByRank(nil)
	rows := make([]BoundaryRow, 0, (1+len(w.cfg.Commodities))*(numBuiltInReasons+len(declared)))
	for thing := ledger.Money; thing < len(w.cfg.Commodities); thing++ {
		for r := ledger.Reason(0); int(r) < numBuiltInReasons; r++ {
			rows = w.appendBoundaryRow(rows, r, thing)
		}
	}
	for _, r := range declared {
		for thing := ledger.Money; thing < len(w.cfg.Commodities); thing++ {
			rows = w.appendBoundaryRow(rows, r, thing)
		}
	}
	return rows
}

// appendBoundaryRow appends the row of the boundary account of reason r and a
// thing (Money or a commodity index), or nothing when the reason has no row for
// the thing or no account was opened.
func (w *World) appendBoundaryRow(rows []BoundaryRow, r ledger.Reason, thing int) []BoundaryRow {
	class, ok := ledger.ClassOf(r, thing != ledger.Money)
	id, open := w.ledger.Boundary(r, thing)
	if !ok || !open {
		return rows
	}
	v, err := w.ledger.Account(id)
	if err != nil {
		invariant("a boundary account is unreadable: " + err.Error())
	}
	name, _ := reasonName(r)
	row := BoundaryRow{Reason: name, Kind: KindFaucet, Commodity: NoCommodity, Balance: v.Balance}
	if class == ledger.Sink {
		row.Kind = KindSink
	}
	if thing != ledger.Money {
		row.Commodity = CommodityID(thing)
	}
	return append(rows, row)
}

// flowTotals converts the totals of the ledger to the rows of a snapshot,
// faucets and sinks only, in the order of the ledger.
func flowTotals(flows []ledger.Flow) []FlowTotal {
	bf := boundaryFlows(nil, flows, 0)
	out := make([]FlowTotal, 0, len(bf))
	for _, f := range bf {
		out = append(out, FlowTotal{Region: f.Region, Reason: f.Reason, Kind: f.Kind, Commodity: f.Commodity, Total: f.Total})
	}
	return out
}

// makesTable says, per role and commodity, whether a recipe of the role makes
// the commodity: the rule under which a belief's bounds use the cost anchor.
func (w *World) makesTable() [][]bool {
	t := make([][]bool, len(w.cfg.Roles))
	for r := range t {
		t[r] = make([]bool, len(w.cfg.Commodities))
		for _, rid := range w.roles.row(RoleID(r)).recipes {
			for _, o := range w.recipes.row(rid).outputs {
				t[r][o.commodity] = t[r][o.commodity] || !o.discard
			}
		}
	}
	return t
}

// snapRegion fills the rows of region g from the view the spawner reads (its
// scans are the classes of spec-observability.md 5.5) and one pass over the
// agents.
func (w *World) snapRegion(s *Snapshot, g RegionID, acc *snapAcc, makes [][]bool) error {
	v := w.buildRegionView(g)
	acc.reset()
	nr := len(w.cfg.Roles)
	rr := &s.Regions[g]
	*rr = RegionRow{Region: g, MoneyPool: v.PoolMoney, EntryBudget: v.Budget, Agents: int64(len(v.Agents))}
	for r := range nr {
		w.roleRow(&s.Roles[int(g)*nr+r], g, v, r)
	}
	for i := range v.Agents {
		ra := &v.Agents[i]
		rr.Households = satAdd(rr.Households, ra.Weight)
		rr.MoneyHouseholds = Money(satAdd(int64(rr.MoneyHouseholds), int64(ra.Money)))
		addRoleAgent(&s.Roles[int(g)*nr+int(ra.Role)], ra)
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err := w.snapAgent(g, &w.agents[ra.Agent], acc, makes); err != nil {
			return err
		}
	}
	floor, _ := resolutionFloor(w.cfg.World)
	for _, l := range w.estates.heldLots(w.ledger, g) {
		acc.estate[l.Commodity] = satAdd(acc.estate[l.Commodity], l.Quantity)
		acc.lots[l.Commodity]++
		if w.estates.atFloor(g, l, floor) {
			acc.atFloor[l.Commodity] = satAdd(acc.atFloor[l.Commodity], l.Quantity)
		}
	}
	w.marketRows(s, g, acc)
	s.Jobs = appendJobs(s.Jobs, g, acc)
	return nil
}

// roleRow sets the signals of role r of the view of region g: the welfare
// summary read by fillRoles and the spawner's expected ratio.
func (w *World) roleRow(row *RoleRow, g RegionID, v *RegionView, r int) {
	sum := &w.ex.sums[r]
	*row = RoleRow{
		Region: g, Role: RoleID(r), HeldByFloor: sum.HeldByFloor, MeanRatioBP: sum.MeanRatioBP,
		HasMeanRatio: sum.HasMean, RatioHouseholds: sum.RatioHouseholds, WageObserved: Money(sum.Wage),
		HasWage: sum.HasWage, PremiumBP: sum.Premium, HasPremium: sum.HasPremium, Window: sum.Window,
		ClampedHouseholds: sum.ClampedHouseholds,
	}
	row.ExpectedRatioBP, _, row.HasExpected, row.Viable = roleSignals(&v.Roles[r], v.TicksPerYear, v.EntryRatio)
}

// addRoleAgent adds agent ra to the counts of its role row.
func addRoleAgent(row *RoleRow, ra *RegionAgent) {
	row.Agents++
	row.Households = satAdd(row.Households, ra.Weight)
	row.Unfed = satAdd(row.Unfed, ra.Unfed)
	if ra.Unfed > 0 {
		row.UnfedAgents++
	}
	row.MaxHunger, row.MaxPrivation = max(row.MaxHunger, ra.Hunger), max(row.MaxPrivation, ra.Privation)
}

// snapAgent adds the lots, the beliefs and the job of agent a to acc.
func (w *World) snapAgent(g RegionID, a *agent, acc *snapAcc, makes [][]bool) error {
	lots, err := w.ledger.AppendLots(w.lotBuf[:0], a.account)
	w.lotBuf = lots
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return fmt.Errorf("lots of agent %d: %w", a.id, err)
	}
	for _, l := range lots {
		acc.held[l.Commodity] = satAdd(acc.held[l.Commodity], l.Quantity)
		acc.lots[l.Commodity]++
	}
	for i, id := range w.roles.row(a.role).tradable {
		var unitCost int64
		if makes[a.role][id] {
			unitCost = w.prod.unitCost(g, id)
		}
		bd := beliefs.Limits(int64(w.refPrice(g, id)), unitCost, w.bp)
		acc.bel[id].add(a.beliefs[i].Centre, bd)
	}
	if !a.hasJob() {
		return nil
	}
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if !w.prod.addPending(&a.job, acc.pending, acc.jobs) {
		return fmt.Errorf("pending output of agent %d: %w", a.id, ErrOverflow)
	}
	acc.execs[a.job.recipe] = satAdd(acc.execs[a.job.recipe], a.job.executions)
	acc.hh[a.job.recipe] = satAdd(acc.hh[a.job.recipe], a.weight)
	return nil
}

// add counts one belief with the given centre against the bounds of its range.
func (b *beliefAcc) add(centre int64, bd beliefs.Bounds) {
	b.n++
	if b.n == 1 {
		b.low, b.high = centre, centre
	}
	b.low, b.high = min(b.low, centre), max(b.high, centre)
	b.sum.AddMul(max(centre, 0), 1)
	if centre == bd.Floor {
		b.atFloor++
	}
	if centre == bd.Ceiling {
		b.atCeiling++
	}
}

// marketRows fills the market rows of region g from its markets and acc.
func (w *World) marketRows(s *Snapshot, g RegionID, acc *snapAcc) {
	nc := len(w.cfg.Commodities)
	for c := range nc {
		i := w.markets.index(g, CommodityID(c))
		row, b := &s.Markets[i], &acc.bel[c]
		row.State = w.marketState(i, g, CommodityID(c))
		row.HeldByAgents, row.HeldByEstate = Quantity(acc.held[c]), Quantity(acc.estate[c])
		row.Lots, row.Pending = acc.lots[c], Quantity(acc.pending[c])
		row.EstateAtFloor = Quantity(acc.atFloor[c])
		row.Beliefs, row.LowCentre, row.HighCentre = b.n, Price(b.low), Price(b.high)
		row.AtFloor, row.AtCeiling = b.atFloor, b.atCeiling
		mean, _ := b.sum.DivHalfEven(b.n)
		row.MeanCentre = Price(mean)
	}
}

// marketState is the state of market i: its report and its window.
func (w *World) marketState(i int, g RegionID, c CommodityID) MarketState {
	r, win := &w.markets.reports[i], &w.markets.windows[i]
	st := MarketState{
		Region: g, Commodity: c, Price: r.Price, Age: r.Age, Volume: r.Volume,
		UnmatchedBid: r.UnmatchedBid, UnmatchedAsk: r.UnmatchedAsk, BestBid: r.BestBid, BestAsk: r.BestAsk,
		HasBid: r.HasBestBid, HasAsk: r.HasBestAsk, WindowLen: int64(win.Len()),
	}
	mean, _ := win.Mean()
	low, _ := win.Low()
	high, _ := win.High()
	st.WindowMean, st.WindowLow, st.WindowHigh = Price(mean), Price(low), Price(high)
	return st
}

// appendJobs appends the job rows of region g, by recipe.
func appendJobs(rows []JobRow, g RegionID, acc *snapAcc) []JobRow {
	for r, n := range acc.jobs {
		if n > 0 {
			rows = append(rows, JobRow{Region: g, Recipe: RecipeID(r), Agents: n, Executions: acc.execs[r], Households: acc.hh[r]})
		}
	}
	return rows
}
