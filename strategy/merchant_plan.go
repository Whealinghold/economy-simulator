package strategy

import (
	"math"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/household"
	"github.com/Whealinghold/economy-sim/internal/merchant"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// wholeBP is 10 000 basis points: a work share of 1.
const wholeBP = 10_000

// state is what one call of Trade reads and computes. It lives on the stack of
// the call, so the strategy keeps nothing between calls (spec-merchants.md 3.5).
type state struct {
	v     *economysim.AgentView
	m     *economysim.MerchantView
	notes *economysim.PlanNotes
	par   household.Params
	// cost is the upkeep cost of one tick at the agent's beliefs, reserve the
	// default's reserve money and phi the agent's fee share (5.7).
	cost, reserve, phi int64
	// idle says the merchant rests or is in the own-use mode: it plans no trip and
	// posts no take (6.4, 6.5). stay says it plans no self-carried trip.
	idle, stay bool
	// best is the best passing candidate, valid when has is set; planned says the
	// trip is planned this tick (it did not fail the second test of 6.3.2).
	best    trip
	has     bool
	planned bool
	// feedBuf and tripBuf are the scratch of the carriage and of the inputs of the
	// trip, on the stack of the call.
	feedBuf [16]merchant.Feed
	tripBuf [16]economysim.TripInput
}

// trip is a candidate that passed the test of the fee share: a lane, a mode and a
// lot, with what the test left of it.
type trip struct {
	dest  economysim.RegionID
	cargo economysim.CommodityID
	mode  *economysim.MerchantMode
	// ci is the index of the cargo in the view's commodities, k the travel time and
	// s the haul units a carrying unit costs on the route.
	ci   int
	k, s int64
	// rnet is the revenue per unit net of the cut and pacq the local price.
	rnet, pacq int64
	// q is the lot and n its carrying units; res is the test at q.
	q, n int64
	res  merchant.Result
	// capped says the cap of the lot, not the money, bounded it.
	capped bool
	// ready says the merchant holds the inputs of the lot qr it can send now, which
	// passed the test again with the result rr.
	ready bool
	qr    int64
	rr    merchant.Result
}

// init reads the view: the household parameters, the upkeep cost and the reserve,
// the fee share and the work share (6.5). A merchant at its target income rests,
// and one in the own-use mode stays (6.4).
func (s *state) init(v *economysim.AgentView, notes *economysim.PlanNotes) {
	s.v, s.m, s.notes = v, &v.Merchant, notes
	t := &v.Tunables
	s.par = household.Params{
		CoverTicks: t.CoverTicks, InputJobs: t.InputJobs, OutputCoverTicks: t.OutputCoverTicks,
		FavourFloor: t.FavourFloor, ReserveTicks: t.ReserveTicks, SpendMaxShare: t.SpendMaxShare,
		SpendHalfTicks: t.SpendHalfTicks, ProduceHaltRatio: t.ProduceHaltRatio,
	}
	s.cost, s.reserve = s.upkeepCost()
	s.phi = merchant.FeeShare(s.m.FeeShare, s.m.Params.FeeShareSpreadBP, int64(v.Agent))
	switch {
	case s.workShare() == 0:
		s.idle = true
		notes.Rested++
	case v.OwnUse:
		s.idle = true
		notes.Waiting++
	}
	s.stay = v.HasJob || len(v.Inbound) > 0
}

// upkeepCost returns the cost of one tick of the agent's upkeep at its belief
// centres and the default's reserve money (spec-agents.md 7.4.2); an overflow is
// the largest value, as the default reads it, so the survival rule holds.
func (s *state) upkeepCost() (cost, reserve int64) {
	var c household.Cost
	for _, row := range s.v.Upkeep {
		total, ok := checked.Mul(s.v.Weight, row.Quantity)
		if !ok {
			return math.MaxInt64, math.MaxInt64
		}
		if i := s.at(row.Commodity); i >= 0 {
			c.Add(total, int64(s.v.Commodities[i].Belief.Centre))
		}
	}
	cost, ok := c.Value()
	if !ok {
		return math.MaxInt64, math.MaxInt64
	}
	reserve, ok = household.ReserveMoney(cost, s.par.ReserveTicks)
	if !ok {
		return cost, math.MaxInt64
	}
	return cost, reserve
}

// workShare is the work share of spec-agents.md 7.1.2 and 7.1.3, the same rule
// functions the default reads: the survival rule (hunger, or money below the
// reserve) and the floor give a whole share whatever the ratio says.
func (s *state) workShare() int64 {
	v := s.v
	if !v.HasRatio {
		return wholeBP
	}
	byRatio := welfare.WorkShare(v.Ratio, v.TargetRatio, v.Tunables.RestBand)
	if v.Hunger > 0 || v.Money < s.reserve || welfare.FloorBinds(v.WorkingTicks, v.Window, v.MinWorkShare) {
		return wholeBP
	}
	return byRatio
}

// at returns the index of the commodity in the view's tradable set, or -1. The set
// is in ascending order, and the merchant's goods are in the same order.
func (s *state) at(c economysim.CommodityID) int {
	i, ok := slices.BinarySearchFunc(s.v.Commodities, c, func(cv economysim.CommodityView, c economysim.CommodityID) int {
		return int(cv.Commodity) - int(c)
	})
	if !ok {
		return -1
	}
	return i
}

// price is the local price of the commodity at index i: the market's, and the
// effective reference price when the market has none.
func (s *state) price(i int) int64 {
	if p := s.v.Commodities[i].Market.Price; p > 0 {
		return p
	}
	return int64(s.v.Commodities[i].Ref)
}

// keep is the units of the commodity at index i the household keeps with no trip
// planned (spec-agents.md 7.3): the targets of its upkeep alone.
func (s *state) keep(i int) int64 {
	var u int64
	for _, row := range s.v.Upkeep {
		if row.Commodity == s.v.Commodities[i].Commodity {
			u, _ = checked.Mul(s.v.Weight, row.Quantity)
		}
	}
	t, ok := household.ComputeTargets(household.Need{Upkeep: u}, s.par)
	if !ok {
		return math.MaxInt64
	}
	return t.Keep
}

// free is the units of the commodity at index i above what the household keeps.
func (s *state) free(i int) int64 {
	return max(0, s.v.Commodities[i].Stock-s.keep(i))
}

// high is the dearest limit the default's draw can give for the commodity at index
// i: the draw of its belief with the largest word (spec-merchants.md 6.2.1).
func (s *state) high(i int) int64 {
	c := &s.v.Commodities[i]
	return beliefs.Draw(beliefs.Belief{Centre: int64(c.Belief.Centre), Width: c.Belief.Width},
		beliefs.Bounds{Floor: int64(c.Bounds.Floor), Ceiling: int64(c.Bounds.Ceiling)}, s.beliefParams(), math.MaxUint64)
}

// beliefParams copies the belief tunables of the view to the parameters of the
// belief rule, as the default does for its own draws.
func (s *state) beliefParams() beliefs.Params {
	t := &s.v.Tunables.Beliefs
	return beliefs.Params{
		Step: t.Step, AdjustProb: t.AdjustProb, Significance: t.Significance,
		EvidenceShare: t.EvidenceShare, StockLow: t.StockLow, StockHigh: t.StockHigh,
		MaxAge: t.MaxAge, WidthInit: t.WidthInit, WidthMin: t.WidthMin,
		WidthMax: t.WidthMax, WidthStep: t.WidthStep, CostFloor: t.CostFloor,
		CostCeiling: t.CostCeiling, RefBandLow: t.RefBandLow, RefBandHigh: t.RefBandHigh,
		MaxPrice: economysim.MaxPrice,
	}
}

// plan evaluates every candidate lane in ascending (destination, cargo, mode)
// order, counts each once by the first case that decides it (5.1.2), plans the
// passing candidate with the greatest score and, when its inputs are held, tests
// it again at the lot it can send (6.3.2). It returns whether a trip is planned;
// a trip that is ready and fails the second test is not planned (3.4.3).
func (s *state) plan() bool {
	if s.idle {
		return false
	}
	if s.stay && s.hasSelfCarried() {
		s.notes.Waiting++
	}
	rows := s.v.Routes
	for lo := 0; lo < len(rows); {
		hi := lo + 1
		for hi < len(rows) && rows[hi].To == rows[lo].To {
			hi++
		}
		s.lane(rows[lo:hi])
		lo = hi
	}
	if !s.has {
		return false
	}
	s.notes.Planned++
	if s.best.capped {
		s.notes.LotCapped++
	}
	s.planned = s.prepare()
	return s.planned
}

// hasSelfCarried reports whether the block lists a self-carried mode.
func (s *state) hasSelfCarried() bool {
	for i := range s.m.Modes {
		if !s.m.Modes[i].Hired {
			return true
		}
	}
	return false
}

// lane evaluates the candidates of one destination: the rows are its routes, one
// per mode, in mode order.
func (s *state) lane(rows []economysim.RouteRow) {
	for _, c := range s.m.Cargo {
		if ci := s.at(c); ci >= 0 {
			s.cargoLane(rows, c, ci)
		}
	}
}

// cargoLane evaluates the candidates of one destination and one cargo, a mode at
// a time: a mode with no route, one the block does not list and a self-carried mode
// of a merchant that stays are not candidates.
func (s *state) cargoLane(rows []economysim.RouteRow, c economysim.CommodityID, ci int) {
	for _, r := range rows {
		md := s.modeOf(r.Mode)
		if !r.Has || md == nil || (s.stay && !md.Hired) {
			continue
		}
		t := trip{dest: r.To, cargo: c, mode: md, ci: ci, k: r.Ticks, s: r.HaulPerUnit}
		if s.eval(&t) {
			s.keepBest(&t)
		}
	}
}

// keepBest keeps the passing candidate if it is the best so far.
func (s *state) keepBest(t *trip) {
	if !s.has || merchant.Better(s.cand(t), s.cand(&s.best)) {
		s.best, s.has = *t, true
	}
}

// cand is the place and the score of a passing trip for merchant.Better.
func (s *state) cand(t *trip) merchant.Cand {
	return merchant.Cand{Score: t.res.Score, Dest: int64(t.dest), Cargo: int64(t.cargo), Mode: int64(t.mode.Mode)}
}

// modeOf returns the merchant's mode with the ID, nil when the block does not list
// it. The modes are in mode order.
func (s *state) modeOf(id economysim.ModeID) *economysim.MerchantMode {
	for i := range s.m.Modes {
		if s.m.Modes[i].Mode == id {
			return &s.m.Modes[i]
		}
	}
	return nil
}

// locked reports whether units of the cargo are on the way to the destination
// already: the lane lock (7.1, R1).
func (s *state) locked(t *trip) bool {
	for _, o := range s.v.Outbound {
		if o.To == t.dest && o.Commodity == t.cargo && o.Quantity > 0 {
			return true
		}
	}
	return false
}

// eval decides one candidate by the first case of 5.1.2 that applies, counts it,
// and for a passing candidate sets its lot and what the test left. It returns
// whether the candidate passes.
func (s *state) eval(t *trip) bool {
	n := s.notes
	n.Considered++
	rho, ok := s.v.Rumour(t.dest, t.cargo)
	switch {
	case !ok:
		n.NoRumour++
		return false
	case rho.SourceAge > s.m.Params.MaxSourceAge:
		n.Stale++
		return false
	case s.locked(t):
		n.Locked++
		return false
	}
	g := &s.m.Goods[t.ci]
	if g.ShelfLife > 0 && merchant.Perishable(g.ShelfLife, g.OldestAge, t.k) {
		n.Perishable++
		return false
	}
	cut := int64(0)
	if t.mode.Hired {
		cut = s.m.Params.ConsigneeCutBP
	}
	rnet, ok := merchant.Net(merchant.Cautious(int64(rho.Price), int64(rho.Mean)), cut, 0)
	t.rnet, t.pacq = rnet, s.price(t.ci)
	if !ok || t.rnet <= t.pacq {
		n.NoMargin++
		return false
	}
	return s.pass(t)
}

// pass sizes the lot of the candidate and applies the test of the fee share to
// it: a lot of 0 is Unaffordable, a carriage dearer than the share of the gross
// margin is BelowShare (5.1.2 cases 6 and 7).
func (s *state) pass(t *trip) bool {
	n := s.notes
	if !s.size(t) {
		n.Unaffordable++
		return false
	}
	res, c, nu, ok := s.margin(t, t.q)
	if !ok {
		n.Unaffordable++
		return false
	}
	if !merchant.Pass(c, s.phi, res.G) {
		n.BelowShare++
		return false
	}
	t.res, t.n = res, nu
	return true
}
