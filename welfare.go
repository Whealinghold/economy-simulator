package economysim

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is the root glue of the welfare record (spec-welfare-and-wages.md
// 9; plan-phase-1-6.md Phase 4): the history an agent holds, the scratch of a
// tick's flows and its conversion to a record at P6.3, the working tick, the
// cost of a tick, the creation of a target, the wage of a region, the runs of
// the two diagnostic Warns and the per-(region, role) summary the snapshot
// rows are built from. The arithmetic is in internal/welfare over primitive
// slices, converted inline here; this file holds no checked call and no
// result-affecting literal. It emits nothing, logs nothing and draws no random
// number beyond the one word of newTarget. Nothing here is state except what
// the world row stores in the scratch and the runs it keeps.

// history is the welfare ring an agent holds (spec-welfare-and-wages.md 13).
type history = welfare.Ring

// newHistory returns an empty history sized to the role's effective window.
// The window comes from a validated role, so a refusal is a programmer error.
func newHistory(row *roleRow) history {
	h, err := welfare.NewRing(row.window, MaxWelfareWindow)
	if err != nil {
		panic("economysim: invariant violated: role window outside the welfare bounds after validation: " + err.Error())
	}
	return h
}

// newTarget draws the target ratio of a new agent (spec-welfare-and-wages.md
// 5.1, FR-WEL-024): lo + draw(hi - lo + 1) with the range of 5.1. The caller
// has keyed s to the creation stream of the agent (spec-world.md 5.7.1), and
// this takes the first word of it whatever the spread, so the stream does not
// depend on the spread. The residue draw is the next word, the caller's.
func newTarget(s *stream, targetRatio, spread int64) int64 {
	lo, hi := welfare.TargetRange(targetRatio, spread)
	return s.drawRange(lo, hi)
}

// welfareParams are the welfare parameters of a role, read once by the caller.
type welfareParams struct {
	window     int64 // the effective window T of the role
	minShareBP int64 // MinWorkShare
	restBand   int64 // World.RestBand
	defaultBP  int64 // the role's DefaultRatio
}

// paramsOf reads the welfare parameters of a role from a canonical
// configuration and its role table.
func paramsOf(c *Config, id RoleID, t *roleTable) welfareParams {
	r := &c.Roles[id]
	return welfareParams{window: t.row(id).window, minShareBP: r.MinWorkShare, restBand: c.World.RestBand, defaultBP: r.DefaultRatio}
}

// wearRow is the fact production hands over for a tool: executions started on
// a tool of the commodity with the wear units of one use (plan 1-5 D51). It is
// valued at P6.3, not when it is recorded (plan 1-6 D66).
type wearRow struct {
	commodity  CommodityID
	executions int64
	wearPerUse int64
}

// flows are the flows of one agent in one tick, cleared after the push at P6.3.
type flows struct {
	receipts   int64 // R_t, from the settled trades
	inKind     int64 // K_t
	inputSpend int64 // S_t(Input)
	relief     int64 // F_t
	unearned   int64 // G_t
	ownReceipt int64 // K^own_t
	// wear is merged by (commodity, wearPerUse); an agent starts one job in a
	// tick, so there are at most MaxRequires tools.
	wear  [MaxRequires]wearRow
	nWear int
	// k is the executions of the one job of the tick (P2.1 or P2.7, never both,
	// FR-PRO-014) and ownUse says it is an own-use job; jobSet says markWorking
	// was called.
	k      int64
	ownUse bool
	jobSet bool
	traded bool
}

// costBuf holds the quantity and price slices that TickCost takes, so that
// forming the cost of a tick allocates nothing. Its capacity is the number of
// commodities, the most rows an upkeep list can have.
type costBuf struct{ q, p []int64 }

func newCostBuf(commodities int) costBuf {
	return costBuf{q: make([]int64, 0, commodities), p: make([]int64, 0, commodities)}
}

// tick is the cost of one tick of the upkeep rows of a weight at the prices
// (spec-welfare-and-wages.md 2.1, FR-WEL-004): the one place where the welfare
// cost multiplies a quantity by the weight (plan 1-6 A10). It saturates.
func (b *costBuf) tick(upkeep []itemRow, prices []int64, weight int64) int64 {
	b.q, b.p = b.q[:0], b.p[:0]
	for _, it := range upkeep {
		b.q = append(b.q, int64(it.quantity))
		b.p = append(b.p, prices[it.commodity])
	}
	return welfare.TickCost(b.q, b.p, weight)
}

// welfareScratch is the scratch of the flows of a tick, one entry per agent
// slot (plan 1-6 A4). The other rows add into it through the methods below.
type welfareScratch struct {
	slots      []flows
	durability []int64 // by CommodityID
	cost       costBuf
}

// newWelfareScratch builds the scratch for a canonical configuration.
func newWelfareScratch(c *Config, slots int) welfareScratch {
	s := welfareScratch{
		slots:      make([]flows, slots),
		durability: make([]int64, len(c.Commodities)),
		cost:       newCostBuf(len(c.Commodities)),
	}
	for i := range c.Commodities {
		s.durability[i] = c.Commodities[i].Durability
	}
	return s
}

// at returns the flows of a slot, growing the scratch for a new slot (an
// agent entering, outside the tick's hot path).
func (s *welfareScratch) at(slot int) *flows {
	for len(s.slots) <= slot {
		s.slots = append(s.slots, flows{})
	}
	return &s.slots[slot]
}

// satAdd adds two amounts, saturating like every sum of the tick (plan 1-6
// D63). It goes through TickIncome so that this file holds no checked call.
func satAdd(a, b int64) int64 { return welfare.TickIncome([]int64{a, b}, nil) }

// addReceipt adds the money received from a settled trade (R_t).
func (s *welfareScratch) addReceipt(slot int, amount int64) {
	f := s.at(slot)
	f.receipts = satAdd(f.receipts, amount)
}

// addUnearned adds money received that is not earned by working (G_t): a grant
// of the money supply module that counts as income (spec-money-supply.md 7.2).
func (s *welfareScratch) addUnearned(slot int, amount int64) {
	f := s.at(slot)
	f.unearned = satAdd(f.unearned, amount)
}

// addInputSpend adds the money spent on a bid with the purpose Input (S_t).
func (s *welfareScratch) addInputSpend(slot int, amount int64) {
	f := s.at(slot)
	f.inputSpend = satAdd(f.inputSpend, amount)
}

// addWear records the wear facts of executions started on a tool. Rows of the
// same commodity and wear per use merge, because the ceilings of another wear
// per use differ (plan 1-6 D66).
func (s *welfareScratch) addWear(slot int, commodity CommodityID, executions, wearPerUse int64) {
	f := s.at(slot)
	for i := range f.nWear {
		if w := &f.wear[i]; w.commodity == commodity && w.wearPerUse == wearPerUse {
			w.executions = satAdd(w.executions, executions)
			return
		}
	}
	if f.nWear == len(f.wear) {
		panic("economysim: invariant violated: more wear rows in one tick than a recipe has tools")
	}
	f.wear[f.nWear] = wearRow{commodity: commodity, executions: executions, wearPerUse: wearPerUse}
	f.nWear++
}

// markWorking sets the executions of the one job of the tick and whether it is
// an own-use job. An agent has at most one job in a tick (FR-PRO-014), seen at
// P2.1 or P2.7, so a second call for a slot in a tick is a violated invariant.
func (s *welfareScratch) markWorking(slot int, k int64, ownUse bool) {
	f := s.at(slot)
	if f.jobSet {
		panic("economysim: invariant violated: markWorking called twice for one agent in one tick")
	}
	f.k, f.ownUse, f.jobSet = k, ownUse, true
}

// markTraded says the agent traded in the tick, from the settled trades.
func (s *welfareScratch) markTraded(slot int) { s.at(slot).traded = true }

// workRead is what workingTick reads from the scratch of a slot: the executions
// of the one job, whether it is an own-use job, and whether the agent traded.
func (s *welfareScratch) workRead(slot int) (k int64, ownUse, traded bool) {
	f := s.at(slot)
	return f.k, f.ownUse, f.traded
}

// reset clears the flows of a slot after the push at P6.3.
func (s *welfareScratch) reset(slot int) { *s.at(slot) = flows{} }

// workingTick decides whether the tick was a working tick and its working and
// own-use household-ticks (spec-welfare-and-wages.md 4.3, FR-WEL-017, 018, plan
// 1-6 D70). k is the executions of the agent's one job (0 for none) and ownUse
// says it is an own-use job, so ownHH is k for an own-use job and 0 otherwise.
//
// A role with recipes works by a job in progress: workHH is k. A role without
// recipes works by trading or travelling, workHH is the weight; when it ran an
// own-use job and did not trade, workHH is k, so workHH less ownHH is 0; with
// both, workHH is the weight and ownHH the own-use k, never negative while k is
// at most the weight. inTransit is a commercial trip, which the merchant module
// sets (none in the slice); migration travel, rest and idle are not working.
func workingTick(hasRecipes bool, k int64, ownUse, traded, inTransit bool, weight int64) (working bool, workHH, ownHH int64) {
	k = max(k, 0) // a negative count of executions is none, never a negative household-tick
	if ownUse {
		ownHH = k
	}
	if hasRecipes {
		return k > 0 || inTransit, k, ownHH
	}
	switch {
	case traded || inTransit:
		return true, weight, ownHH
	case ownUse && k > 0:
		return true, k, ownHH
	}
	return false, 0, 0
}

// record forms the welfare record of the tick of a slot at P6.3 (plan 1-6 D61,
// D66): income_t by the formula of 2.1 over the flows in the scratch, with the
// wear rows valued here at the prices of the tick, and cost_t of the upkeep
// rows at the same prices. prices is by CommodityID and holds the published
// price of the tick, or the effective reference price where the market has
// never traded. Every sum saturates and there is no ok: the clamp of Push and
// the Warn report an overflow (D63). It does not clear the scratch.
func (s *welfareScratch) record(slot int, weight int64, working bool, workHH, ownHH int64, prices []int64, upkeep []itemRow) welfare.Record {
	f := s.at(slot)
	credits := [5]int64{f.receipts, f.inKind, f.ownReceipt, f.relief, f.unearned}
	var debits [1 + MaxRequires]int64
	debits[0] = f.inputSpend
	for i := range f.nWear {
		w := &f.wear[i]
		debits[1+i] = welfare.WearCost(w.wearPerUse, prices[w.commodity], s.durability[w.commodity], w.executions)
	}
	return welfare.Record{
		Income:     welfare.TickIncome(credits[:], debits[:1+f.nWear]),
		Cost:       s.cost.tick(upkeep, prices, weight),
		Relief:     f.relief,
		Unearned:   f.unearned,
		OwnReceipt: f.ownReceipt,
		Working:    working,
		WorkHH:     workHH,
		OwnHH:      ownHH,
	}
}

// roleSet is the agents of one role in one region, in ascending slot order, as
// parallel slices, so the caller chooses the set (plan 1-6 Phase 3).
type roleSet struct {
	rings   []*welfare.Ring
	weights []int64
	targets []int64 // the target ratio drawn at creation of each agent, T_lo
	free    []bool  // no job in progress when Work would be called
}

// readBuf holds the parallel slices of per-agent reads, so that a read over a
// set allocates nothing once the buffers have the size of the set.
type readBuf struct {
	nums, dens, ratios, working, sel []int64
	present                          []bool
	cost                             costBuf
}

func newReadBuf(commodities int) readBuf { return readBuf{cost: newCostBuf(commodities)} }

// parts fills the numerator and denominator of the observed wage of each ring.
func (b *readBuf) parts(rings []*welfare.Ring) {
	b.nums, b.dens = b.nums[:0], b.dens[:0]
	for _, r := range rings {
		n, d := r.WageParts()
		b.nums, b.dens = append(b.nums, n), append(b.dens, d)
	}
}

// fill fills the ratio, its presence and the working ticks of each agent of
// the set, valuing the upkeep of the agent at its weight and the prices.
func (b *readBuf) fill(set roleSet, p welfareParams, upkeep []itemRow, prices []int64) {
	b.ratios, b.working, b.present = b.ratios[:0], b.working[:0], b.present[:0]
	for i, r := range set.rings {
		u := b.cost.tick(upkeep, prices, set.weights[i])
		ratio, ok := r.Ratio(u, p.defaultBP)
		b.ratios, b.present = append(b.ratios, ratio), append(b.present, ok)
		b.working = append(b.working, r.WorkingTicks())
	}
}

// observedWage is the observed wage of a set of rings, the sum of numerators
// over the sum of working household-ticks less own-use, present whenever the
// denominator is above 0 (spec-welfare-and-wages.md 4.3, FR-WEL-037). The
// unskilled wage of a region is this over the rings of its Unskilled roles.
func (b *readBuf) observedWage(rings []*welfare.Ring) (wage int64, present bool) {
	b.parts(rings)
	return welfare.SetWage(b.nums, b.dens)
}

// regionWage is the wage the cost anchor reads for a region (4.3): the pooled
// wage over the region's rings when known and positive, raised to floorBP basis
// points of the starting wage (World.WageFloor, spec-production.md 9.3), else the
// starting wage of the warm start, else unknown. It keeps no state, so it gives
// the same answer after the rings are rebuilt from records. seed is the starting
// wage, 0 when there is none; the world row writes 0 for an unknown wage into the
// cost-anchor table.
func (b *readBuf) regionWage(rings []*welfare.Ring, seed, floorBP int64) (wage int64, known bool) {
	b.parts(rings)
	if w, ok := welfare.PooledWage(b.nums, b.dens); ok {
		return welfare.FloorWage(w, seed, floorBP), true
	}
	if seed > 0 {
		return seed, true
	}
	return 0, false
}

// seedWages returns the starting wage of each region (spec-viability.md 8.2.5),
// 0 where the warm start has none, indexed by RegionID.
func seedWages(c *Config) []int64 {
	out := make([]int64, len(c.Regions))
	if c.WarmStart == nil {
		return out
	}
	for _, w := range c.WarmStart.Wages {
		if g, ok := c.regionID(w.Region); ok {
			out[g] = int64(w.Wage)
		}
	}
	return out
}

// heldByFloor counts, in households, the free agents of the set that the work
// floor holds to work (plan 1-6 D69, FR-WEL-033): the floor condition of 5.3
// holds and the share their ratio alone gives is below 10 000. The same
// predicate serves the Warn at P2.5 and the snapshot at the boundary.
func (b *readBuf) heldByFloor(set roleSet, p welfareParams, upkeep []itemRow, prices []int64) int64 {
	b.fill(set, p, upkeep, prices)
	// Per agent, because the target T_lo is the agent's own after the spread
	// (a set shares one window and floor, not one target).
	b.sel = b.sel[:0]
	for i, w := range set.weights {
		if set.free[i] && welfare.HeldByFloor(b.ratios[i], b.present[i], b.working[i], p.window, p.minShareBP, set.targets[i], p.restBand) {
			b.sel = append(b.sel, w)
		}
	}
	return welfare.TickIncome(b.sel, nil)
}

// households is the sum of the weights of the set.
func households(set roleSet) int64 { return welfare.TickIncome(set.weights, nil) }

// clampedHouseholds counts, in households, the agents whose ring holds a field
// at the clamp (plan 1-6 D63): a constant-cost read per ring.
func (b *readBuf) clampedHouseholds(set roleSet) int64 {
	b.sel = b.sel[:0]
	for i, r := range set.rings {
		if r.AtBound() {
			b.sel = append(b.sel, set.weights[i])
		}
	}
	return welfare.TickIncome(b.sel, nil)
}

// advanceRun advances a diagnostic run of a bounded Warn by one tick
// (spec-observability.md 7.3): it resets when the condition does not hold and
// otherwise counts, saturating, and warns at every warnRun-th tick (pattern B)
// and, with first, also at the first (pattern A).
func advanceRun(before int64, active, first bool) (run int64, warn bool) {
	if !active {
		return 0, false
	}
	run = welfare.NextRun(before)
	return run, run%warnRun == 0 || (first && run == 1)
}

// floorRun advances the run of the Warn "work floor binding" of a (region,
// role) (FR-WEL-033, D69): the tick counts when strictly more than half of the
// role's households are held, 2 * held > households, written without the
// product. A role with no households has no run.
func floorRun(before, held, households int64) (run int64, warn bool) {
	return advanceRun(before, households > 0 && held > households/2, false)
}

// clampRun advances the run of the Warn "welfare record clamped" of a (region,
// role) (D63, FR-WEL-042), pattern A: the first tick of a run and every tenth.
func clampRun(before, clamped int64) (run int64, warn bool) {
	return advanceRun(before, clamped > 0, true)
}

// roleSummary is what the snapshot row of a (region, role) is built from
// (FR-WEL-020, 036, 037, plan 1-6 D62, D63, D69).
type roleSummary struct {
	Wage              int64 // the observed wage of the set, not the pooled one
	HasWage           bool
	Premium           int64
	HasPremium        bool
	Window            int64
	HeldByFloor       int64
	ClampedHouseholds int64
	MeanRatioBP       int64
	RatioHouseholds   int64
	HasMean           bool
}

// summarise reads the summary of a set at the boundary. unskilled is the
// observed wage of the region's Unskilled agents and whether it exists, from
// observedWage over their rings.
func (b *readBuf) summarise(set roleSet, p welfareParams, upkeep []itemRow, prices []int64, unskilled int64, hasUnskilled bool) roleSummary {
	sum := roleSummary{Window: p.window}
	sum.Wage, sum.HasWage = b.observedWage(set.rings)
	sum.Premium, sum.HasPremium = welfare.Premium(sum.Wage, sum.HasWage, unskilled, hasUnskilled)
	sum.HeldByFloor = b.heldByFloor(set, p, upkeep, prices) // fills the ratios
	sum.ClampedHouseholds = b.clampedHouseholds(set)
	sum.MeanRatioBP, sum.RatioHouseholds, sum.HasMean = welfare.MeanRatio(set.weights, b.ratios, b.present)
	return sum
}

// saveWelfare is tag 7 (spec-checkpoint.md 4.2; plan 2-2 D248): for each live
// agent in ascending ID the records its ring holds, oldest first, and its target
// ratio, then the sparse runs of the two Warns of welfare, the work floor and the
// clamp.
func (w *World) saveWelfare(cw *codec.Writer) error {
	var recs []welfare.Record
	w.eachLive(func(a *agent) {
		recs = a.history.AppendRecords(recs[:0])
		cw.Count(len(recs))
		for i := range recs {
			saveRecord(cw, &recs[i])
		}
		cw.U(unsigned(a.target))
	})
	writeSparse(cw, w.state.floorRun, len(w.cfg.Roles))
	writeSparse(cw, w.state.clampRun, len(w.cfg.Roles))
	return nil
}

// saveRecord writes one welfare record: the five money fields signed, the
// own-use household-ticks, the working flag and the working household-ticks.
func saveRecord(cw *codec.Writer, r *welfare.Record) {
	cw.I(r.Income)
	cw.I(r.Cost)
	cw.I(r.Relief)
	cw.I(r.Unearned)
	cw.I(r.OwnReceipt)
	cw.U(unsigned(r.OwnHH))
	cw.B(r.Working)
	cw.U(unsigned(r.WorkHH))
}

// recordMin is the fewest bytes a welfare record takes: five signed fields, the
// household-ticks, the flag and the working household-ticks, one byte each.
const recordMin = 8

// loadWelfare is tag 7: for each live agent in ascending ID the records its ring
// holds, oldest first, and its target ratio, then the sparse runs of the work floor
// and the clamp. The ring is rebuilt by welfare.FromRecords, which refuses more
// records than the window and a field beyond the clamp.
func (w *World) loadWelfare(rs *restoreCtx, r *codec.Reader) error {
	var recs []welfare.Record
	w.eachLive(func(a *agent) {
		recs = recs[:0]
		for range r.Count(recordMin) {
			recs = append(recs, loadRecord(r))
		}
		// The draw reaches TargetRatio * (10 000 + spread) / 10 000, below twice MaxBasisPoints.
		a.target = r.Bounded("target ratio", 2*MaxBasisPoints)
		if r.Err() != nil {
			return
		}
		h, err := welfare.FromRecords(recs, w.roles.row(a.role).window, MaxWelfareWindow)
		if err != nil {
			r.Failf("agent %d: %v", a.id, err)
			return
		}
		a.history = h
	})
	roles := len(w.cfg.Roles)
	readSparse(r, w.state.floorRun, len(w.cfg.Regions), roles, rs.tick)
	readSparse(r, w.state.clampRun, len(w.cfg.Regions), roles, rs.tick)
	return nil
}

// loadRecord reads one welfare record, the counterpart of saveRecord.
func loadRecord(r *codec.Reader) welfare.Record {
	return welfare.Record{
		Income: r.I(), Cost: r.I(), Relief: r.I(), Unearned: r.I(), OwnReceipt: r.I(),
		OwnHH: r.Bounded("own-use household-ticks", math.MaxInt64), Working: r.B(),
		WorkHH: r.Bounded("working household-ticks", math.MaxInt64),
	}
}
