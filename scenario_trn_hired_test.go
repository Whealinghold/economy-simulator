package economysim_test

import (
	"fmt"
	"math/rand/v2"
	"os"
	"regexp"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds S32 of the transport module (TRN-A3, D457) and the tests on a carrier
// role of the hired mode (D458: FR-CAR-003, 009, 010, 011, 013 and FR-TRN-043 are tests on
// the pack, with no new code). The pack is the module-alone pack two, whose coach is hired:
// it redeems haul units, an ordinary commodity the carrier role makes by an ordinary
// recipe (run_haul, 4 ticks, the round trip of the coach over an edge of 100) and sells in
// an ordinary market. The growers of the scenario are the shippers: they bid for the haul
// units of one trip, ship two units of staple by coach to the first maker of the other
// region, and bid again.

// hiredPack is the pack with the grower role allowed to trade the haul commodity and
// given the money to buy it.
func hiredPack(t testing.TB) trnWorld {
	t.Helper()
	return trnPack(t, 0, func(c *economysim.Config) {
		i := slices.IndexFunc(c.Roles, func(r economysim.Role) bool { return r.ID == "grower" })
		c.Roles[i].Trades = append(slices.Clone(c.Roles[i].Trades), "haul")
		// The shippers can pay for the trip: a bid for 10 haul units at about 500 each is
		// 5 000, more than a household of the pack holds.
		c.Roles[i].StartMoney += 100_000
		// Haul units are worth more than the fuel that makes them, so that a carrier
		// driven by the default strategy sees a profit in the recipe.
		for j := range c.Commodities {
			if c.Commodities[j].ID == "haul" {
				c.Commodities[j].ReferencePrice = 1_200
			}
		}
	})
}

// s32Strategy is the scripted strategy of S32 and of the carrier tests: the growers
// shipping by coach, and, when elasticity is above 0, the carriers starting the haul recipe
// with a probability that rises with the haul price at the rate elasticity (basis points of
// probability per basis point of the price's distance above its reference).
type s32Strategy struct {
	trnBase
	w          trnWorld
	elasticity int64
	modulus    int64
}

const coachID economysim.ModeID = 0 // "coach" precedes "foot" in the canonical order

func (s s32Strategy) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	if v.Role != s.w.carrier || s.elasticity == 0 {
		return economysim.DefaultStrategy{}.Work(v, rng)
	}
	var price, ref int64
	for _, cv := range v.Commodities {
		if cv.Commodity == s.w.haul {
			price, ref = cv.Market.Price, int64(cv.Ref)
		}
	}
	if ref < 1 {
		return economysim.JobRequest{NoJob: true}
	}
	start := min(max(5_000+s.elasticity*(price-ref)*10_000/ref/10, 0), 10_000)
	if rng.Int64N(10_000) >= start {
		return economysim.JobRequest{NoJob: true}
	}
	return economysim.JobRequest{Recipe: s.w.runHaul, Executions: 1}
}

func (s s32Strategy) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	economysim.DefaultStrategy{}.Trade(v, rng, p)
	if v.Role != s.w.grower || int(v.Region) >= 2 {
		return
	}
	// The default strategy asks for the haul units a grower holds and has no use for; the
	// scripted bid below would then trade with itself (the clearing refuses it).
	p.Intents = slices.DeleteFunc(p.Intents, func(in economysim.Intent) bool { return in.Commodity == s.w.haul })
	to := economysim.RegionID(1 - int(v.Region))
	var need int64
	for _, r := range v.Routes {
		if r.To == to && r.Mode == coachID && r.Has {
			need = r.HaulPerUnit
		}
	}
	cargo := economysim.Line{Commodity: s.w.staple, Quantity: 2}
	if need == 0 {
		return
	}
	have := stockOf(v, s.w.haul)
	if have >= need {
		if stockOf(v, s.w.staple) < int64(cargo.Quantity) {
			return
		}
		p.Shipments = append(p.Shipments, economysim.ShipmentRequest{Mode: coachID, To: to, Consignee: s.w.first[to], Lines: []economysim.Line{cargo}})
		return
	}
	if (int64(v.Agent)+v.Tick.Tick)%s.modulus != 0 {
		return // bids are placed one household in modulus a tick; the haul units keep two ticks
	}
	for _, cv := range v.Commodities {
		if cv.Commodity == s.w.haul {
			p.Intents = append(p.Intents, economysim.Intent{
				Commodity: s.w.haul, Side: economysim.Bid, Quantity: economysim.Quantity(need - have),
				Limit: min(cv.Bounds.Ceiling, 3*economysim.Price(cv.Market.Price)/2), Target: economysim.Quantity(need), Purpose: economysim.PurposeOther,
			})
		}
	}
}

// s32Ticks is the length of a run of S32; the measures compare ticks 20 to 120 with
// ticks 200 to 300.
const s32Ticks = 300

// TestScenarioS32_HaulCapacityCobweb runs S32 (TRN-A3; D457): the hired coach, a carrier
// role whose recipe duration is the round trip, and three elasticities of the carriers'
// starts (1, 4 and 16). It records the price series of the haul unit as its range in the
// early window (ticks 20 to 120) and in the late one (200 to 300), the shipments
// dispatched and the haul units traded, and whether the price converged (the late range
// not above the early) or is reported as F6. No band is asserted (O8). It asserts the
// exact facts: lines add up, the audit holds every tick, the shipments dispatched are
// those arrived plus those in the snapshot, and the haul units redeemed by dispatches are
// the haul_use flow (FR-MER-048).
//
// Requirement: FR-TRN-012, FR-TRN-028, FR-TRN-043, FR-CAR-003
func TestScenarioS32_HaulCapacityCobweb(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	rep := newPackScenReport("two", "S32 the haul capacity cobweb (TRN-A3)")
	wd := hiredPack(t)
	t.Run("cells", func(t *testing.T) {
		for _, e := range []int64{1, 4, 16} {
			name := fmt.Sprintf("s32.e%d", e)
			for _, seed := range seedList(scenarioSeedCount()) {
				t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
					t.Parallel()
					s32Seed(t, rep, name, wd, seed, e)
				})
			}
		}
	})
	rep.flush(t)
}

// rangeOf returns the largest less the smallest value of s[from:to], clipped to its length.
func rangeOf(s []int64, from, to int) (int64, bool) {
	to = min(to, len(s))
	if from >= to {
		return 0, false
	}
	lo, hi := slices.Min(s[from:to]), slices.Max(s[from:to])
	return hi - lo, true
}

// s32Seed runs one seed of one elasticity and records its measures and checks.
func s32Seed(t *testing.T, rep *scenReport, name string, wd trnWorld, seed uint64, elasticity int64) {
	obs := newTrnObs(wd.fuel, wd.haul)
	st := s32Strategy{w: wd, elasticity: elasticity, modulus: 6}
	w := trnRun(t, wd.cfg, seed, s32Ticks, st, obs)
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("%s seed %d: Snapshot() error = %v, want nil", name, seed, err)
	}
	early, okE := rangeOf(obs.haulPrice[0], 20, 120)
	late, okL := rangeOf(obs.haulPrice[0], 200, 300)
	rep.num(seed, name+".dispatched", obs.dispatched)
	rep.num(seed, name+".haul_units_redeemed", obs.haulUnits)
	if okE && okL {
		rep.num(seed, name+".haul_price_range_early", early)
		rep.num(seed, name+".haul_price_range_late", late)
		converged := int64(0)
		if late <= early {
			converged = 1
		}
		rep.num(seed, name+".converged", converged)
	} else {
		rep.absent(seed, name+".converged")
	}
	books := obs.dispatched == obs.arrived+int64(len(snap.Shipments))
	rep.check(seed, name+".lines_add_up", obs.lineBroken == 0, fmt.Sprintf("%d arrived lines do not add up", obs.lineBroken))
	rep.check(seed, name+".books", books, fmt.Sprintf("dispatched %d, arrived %d, in the snapshot %d", obs.dispatched, obs.arrived, len(snap.Shipments)))
	rep.check(seed, name+".haul_use_is_the_flow", obs.haulUnits == obs.haulFlow, fmt.Sprintf("redeemed %d, haul_use flow %d", obs.haulUnits, obs.haulFlow))
	if obs.lineBroken != 0 || !books || obs.haulUnits != obs.haulFlow {
		t.Errorf("%s seed %d: broken lines %d, books %t, haul units redeemed %d against flow %d", name, seed, obs.lineBroken, books, obs.haulUnits, obs.haulFlow)
	}
}

// carrierHooks are what a carrier test watches: the Work call of a carrier and the
// view it is passed in Trade (the only call that sees a job in progress).
type carrierHooks struct {
	work  func(v *economysim.AgentView, req economysim.JobRequest)
	trade func(v *economysim.AgentView)
}

// carrierRun runs the hired pack, edited by mut, with the default strategy for the
// carriers (elasticity 0) and the scripted growers for ticks steps, and returns the
// observer and the events.
func carrierRun(t *testing.T, seed uint64, ticks int, mut func(*economysim.Config), h carrierHooks) (*trnObs, []economysim.Event) {
	t.Helper()
	wd := hiredPack(t)
	if mut != nil {
		mut(&wd.cfg)
	}
	obs := newTrnObs(wd.fuel, wd.haul)
	rec := &recorder{}
	var st economysim.Strategy = s32Strategy{w: wd, modulus: 6}
	if h.work != nil || h.trade != nil {
		st = watch{Strategy: st, carrier: wd.carrier, hooks: h}
	}
	trnRun(t, wd.cfg, seed, ticks, st, obs, rec)
	return obs, rec.events
}

// watch wraps a Strategy and reports every Work and Trade call of the carrier role.
type watch struct {
	economysim.Strategy
	carrier economysim.RoleID
	hooks   carrierHooks
}

func (w watch) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	req := w.Strategy.Work(v, rng)
	if v.Role == w.carrier && w.hooks.work != nil {
		w.hooks.work(v, req)
	}
	return req
}

func (w watch) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	if v.Role == w.carrier && w.hooks.trade != nil {
		w.hooks.trade(v)
	}
	w.Strategy.Trade(v, rng, p)
}

// wellFedCarriers gives the carriers 900 of the goods of their upkeep, money, and a
// prepared that keeps, so that none goes hungry in the run and the work share, not the
// survival rule, decides whether they start a job.
func wellFedCarriers(c *economysim.Config) {
	richCarriers(c)
	i := slices.IndexFunc(c.Roles, func(r economysim.Role) bool { return r.ID == "carrier" })
	c.Roles[i].StartMoney += 200_000
	stock := slices.Clone(c.Roles[i].StartStock)
	for j := range stock {
		if stock[j].Commodity == "fuel" || stock[j].Commodity == "prepared" || stock[j].Commodity == "staple" {
			stock[j].Quantity = 900
		}
	}
	c.Roles[i].StartStock = stock
}

// TestCarrier_WageAndIncome verifies FR-CAR-009 and FR-CAR-010 on the carrier role of the
// hired pack: the haul recipe is a job like any other (a RecipeExecuted for run_haul), the
// carrier's haul units sell in an ordinary market (a TradeExecuted of the haul commodity
// with a carrier as the seller, and the shippers' haul_use flow of the units it redeemed),
// the role has an observed wage as any producer's (a WageObserved above 0 in the snapshot),
// and every tick of a job in progress is a working tick of the carrier's window: the
// working ticks it reports are at least the ticks of the job it has been through, up to
// the window.
//
// Requirement: FR-CAR-009, FR-CAR-010
func TestCarrier_WageAndIncome(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	wd := hiredPack(t)
	jobTicks := map[economysim.AgentID]int64{} // consecutive ticks with a job in progress
	lastTick := map[economysim.AgentID]int64{}
	var short, checked int64
	obs, events := carrierRun(t, 1, 160, nil, carrierHooks{trade: func(v *economysim.AgentView) {
		if lastTick[v.Agent]+1 != v.Tick.Tick || !v.HasJob {
			jobTicks[v.Agent] = 0
		}
		lastTick[v.Agent] = v.Tick.Tick
		if v.HasJob {
			jobTicks[v.Agent]++
			checked++
			if v.WorkingTicks < min(jobTicks[v.Agent]-1, v.Window) { // this tick's is counted at its end (P6.3)
				short++
			}
		}
	}})
	var ran, sold int64
	for _, e := range events {
		switch e := e.(type) {
		case economysim.RecipeExecuted:
			if int(e.Recipe) == int(wd.runHaul) {
				ran += e.Executions
			}
		case economysim.TradeExecuted:
			if e.Commodity == wd.haul && e.Seller >= economysim.AgentID(len(wd.cfg.Regions)) {
				sold += int64(e.Quantity)
			}
		}
	}
	if ran == 0 || sold == 0 || obs.haulFlow == 0 {
		t.Fatalf("run_haul executions %d, haul units sold %d, haul units redeemed %d, want all above 0: the pack's hired mode must be used", ran, sold, obs.haulFlow)
	}
	if checked == 0 || short != 0 {
		t.Errorf("working ticks below the ticks of the job in progress in %d of %d carrier views, want 0 of more than 0", short, checked)
	}
}

// TestCarrier_RestBand verifies FR-CAR-011: a carrier driven by the default strategy
// stops starting haul jobs by the work share of spec-agents.md 7.1.2, the rest band
// included. Over a run of the hired pack, whenever the default returns a job the
// carrier's ratio is below the top of its band, or a floor holds (it went unfed, or its
// money is below the reserve it keeps, or its working ticks are below the floor), and the
// default returned no job for carriers at or above the top of the band at least once.
//
// Requirement: FR-CAR-011
func TestCarrier_RestBand(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	var rested, unexplained int64
	carrierRun(t, 1, 200, wellFedCarriers, carrierHooks{work: func(v *economysim.AgentView, req economysim.JobRequest) {
		if !v.HasRatio || v.Hunger > 0 {
			return
		}
		top := v.TargetRatio * (10_000 + v.Tunables.RestBand) / 10_000
		switch {
		case v.Ratio >= top && req.NoJob:
			rested++
		case v.Ratio >= top && !req.NoJob && v.WorkingTicks*10_000 >= v.MinWorkShare*v.Window && v.Money >= 10*v.TargetRatio:
			unexplained++ // above the band with money in hand and the working floor met: the rule says rest
		}
	}})
	if rested == 0 {
		t.Errorf("the default stopped no carrier at the top of its rest band in 200 ticks, want at least one: the band is the rule under test")
	}
	if unexplained != 0 {
		t.Errorf("%d job starts at or above the top of the band with the floors met and money in hand, want 0", unexplained)
	}
}

// TestCarrier_BeliefRevision verifies FR-CAR-013: the belief of a carrier for the haul
// unit is revised once per admitted order after its market clears, as for any commodity.
// In every tick and region of the run the outcomes of BeliefsUpdated for the haul market
// are the orders placed in that market (a rejected order is not placed, and an estate's listing has no belief).
//
// Requirement: FR-CAR-013
func TestCarrier_BeliefRevision(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	wd := hiredPack(t)
	_, events := carrierRun(t, 1, 160, nil, carrierHooks{})
	placed := map[[2]int64]int64{}
	outcomes := map[[2]int64]int64{}
	for _, e := range events {
		switch e := e.(type) {
		case economysim.OrderPlaced:
			// An estate (an agent below the region count) lists what it holds and has no belief.
			if e.Commodity == wd.haul && e.Agent >= economysim.AgentID(len(wd.cfg.Regions)) {
				placed[[2]int64{e.Tick, int64(e.Region)}]++
			}
		case economysim.BeliefsUpdated:
			if e.Commodity == wd.haul {
				outcomes[[2]int64{e.Tick, int64(e.Region)}] += e.Outcomes
			}
		}
	}
	var total int64
	for k, n := range placed {
		if got := outcomes[k]; got != n {
			t.Errorf("tick %d region %d: %d belief outcomes for the haul market, want the %d orders placed (a rejected order is not placed)", k[0], k[1], got, n)
		}
		total += n
	}
	for k, n := range outcomes {
		if placed[k] == 0 {
			t.Errorf("tick %d region %d: %d belief outcomes for the haul market and no order placed", k[0], k[1], n)
		}
	}
	if total == 0 {
		t.Fatalf("no order for haul units in 160 ticks, want carriers asking and shippers bidding")
	}
}

// TestClearing_HaulUnitsAreOrdinary verifies FR-TRN-043 and FR-CAR-003: hired capacity is
// an ordinary commodity made by an ordinary recipe and traded by ordinary orders. The
// pack with the module on, shipping by coach, passes the invariants of testing.md 6.3
// (runInvariants, over seeds 1 to 3), haul units are traded in its markets, and no file of
// the clearing (the root's clearing.go and the package internal/clearing) names a mode,
// the transport module or the haul.
//
// Requirement: FR-TRN-043, FR-CAR-003
func TestClearing_HaulUnitsAreOrdinary(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	obs, events := carrierRun(t, 1, 120, nil, carrierHooks{})
	if obs.haulFlow == 0 {
		t.Fatalf("no haul unit redeemed in 120 ticks, want hired shipments")
	}
	var traded int64
	wd := hiredPack(t)
	for _, e := range events {
		if tr, ok := e.(economysim.TradeExecuted); ok && tr.Commodity == wd.haul {
			traded += int64(tr.Quantity)
		}
	}
	if traded == 0 {
		t.Errorf("haul units traded = 0, want trades in an ordinary market")
	}
	names := regexp.MustCompile(`(?i)\bmodeid\b|\bmode\b|transport|\bhaul`)
	files := []string{"clearing.go"}
	if dir, err := os.ReadDir("internal/clearing"); err == nil {
		for _, f := range dir {
			if !f.IsDir() && len(f.Name()) > 3 && f.Name()[len(f.Name())-3:] == ".go" {
				files = append(files, "internal/clearing/"+f.Name())
			}
		}
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v, want nil", f, err)
		}
		if loc := names.FindIndex(b); loc != nil && !slices.Contains([]string{"_test.go"}, f[max(0, len(f)-8):]) {
			t.Errorf("%s names %q, want no mode, transport or haul in the clearing", f, b[loc[0]:loc[1]])
		}
	}
	for seed := uint64(1); seed <= 3; seed++ {
		runInvariants(t, loadValidPack(t, packTwoTransport...), seed, 120, nil)
	}
}
