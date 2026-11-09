package economysim_test

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds the scenarios S30 and S31 of the transport module (spec-transport.md
// 15, D457) and what they share with S32 (scenario_trn_hired_test.go): the pack, the
// scripted strategies and the observer. Each runs on the module-alone pack two with a
// hand-written scripted Strategy (architecture.md 10.3), not the merchant, which plan 15
// reruns as S30_WithMerchant and so on. Outcomes are recorded in
// testdata/chains/expected/two.scenarios.txt with -update; no band is invented (O8), and
// the checks that fail a test are the exact ones: conservation of what is shipped, the
// audit every tick, and the shipments in the books.

// trnWorld is the module-alone pack two (packTwoTransport) edited for a scenario, with
// the facts a scripted strategy needs. Mode IDs are coach 0 and foot 1 (modeFootID).
type trnWorld struct {
	cfg economysim.Config
	// first is, per region, the first household of the maker role: the consignee of every
	// shipment, because a household that never ships is never a traveller held by the
	// obligation of being a consignee (5.7).
	first []economysim.AgentID
	// maker, carrier and grower are roles; fuel, haul, prepared and staple are commodities.
	maker, carrier, grower       economysim.RoleID
	fuel, haul, prepared, staple economysim.CommodityID
	runHaul                      economysim.RecipeID
	strategyName                 string
}

// trnName is the name the scripted strategy is registered under.
const trnName = "trn"

// trnPack loads the pack, sets the speed of the foot mode (0 keeps the overlay's) and
// names the scripted strategy for every role; mut then edits the configuration.
func trnPack(t testing.TB, footSpeed int64, mut func(*economysim.Config)) trnWorld {
	t.Helper()
	cfg := loadValidPack(t, packTwoTransport...)
	role := func(id string) economysim.RoleID {
		i := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == id })
		if i < 0 {
			t.Fatalf("pack two has no role %q", id)
		}
		return economysim.RoleID(i)
	}
	commodity := func(id string) economysim.CommodityID {
		i := slices.IndexFunc(cfg.Commodities, func(c economysim.Commodity) bool { return c.ID == id })
		if i < 0 {
			t.Fatalf("pack two has no commodity %q", id)
		}
		return economysim.CommodityID(i)
	}
	if got := cfg.Modules.Transport.Modes[modeFootID].ID; got != "foot" {
		t.Fatalf("mode %d is %q, want foot", modeFootID, got)
	}
	w := trnWorld{
		cfg: cfg, maker: role("maker"), carrier: role("carrier"), grower: role("grower"),
		fuel: commodity("fuel"), haul: commodity("haul"), prepared: commodity("prepared"), staple: commodity("staple"),
		strategyName: trnName,
	}
	if footSpeed > 0 {
		w.cfg.Modules.Transport.Modes[modeFootID].Speed = footSpeed
	}
	if mut != nil {
		mut(&w.cfg)
	}
	rec := &recorder{}
	if _, err := economysim.NewWorld(w.cfg, economysim.WithObserver(rec)); err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	w.first = make([]economysim.AgentID, len(w.cfg.Regions))
	seen := make([]bool, len(w.cfg.Regions))
	for _, e := range rec.events {
		if a, ok := e.(economysim.AgentSpawned); ok && a.Role == w.maker && !seen[a.Region] {
			seen[a.Region], w.first[a.Region] = true, a.Agent
		}
	}
	w.runHaul = economysim.RecipeID(slices.IndexFunc(w.cfg.Recipes, func(r economysim.Recipe) bool { return r.ID == "run_haul" }))
	for i := range w.cfg.Roles {
		w.cfg.Roles[i].Strategy = trnName
	}
	return w
}

// trnBase is a Strategy that does what the default one does; a scenario's strategy
// embeds it and replaces Trade or Work.
type trnBase struct{}

func (trnBase) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	return economysim.DefaultStrategy{}.Work(v, rng)
}

func (trnBase) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	economysim.DefaultStrategy{}.Trade(v, rng, p)
}

func (trnBase) Revise(o *economysim.OutcomeView, rng *rand.Rand) economysim.Belief {
	return economysim.DefaultStrategy{}.Revise(o, rng)
}

// stockOf returns the units of a commodity the view holds.
func stockOf(v *economysim.AgentView, c economysim.CommodityID) int64 {
	for _, cv := range v.Commodities {
		if cv.Commodity == c {
			return cv.Stock
		}
	}
	return 0
}

// provisionedFor reports whether the household holds, after shipping lines, the upkeep
// of its whole weight for ticks ticks: a traveller cannot buy while away (5.4 row 22).
func provisionedFor(v *economysim.AgentView, ticks int64, lines ...economysim.Line) bool {
	for _, u := range v.Upkeep {
		stock := stockOf(v, u.Commodity)
		for _, ln := range lines {
			if ln.Commodity == u.Commodity {
				stock -= int64(ln.Quantity)
			}
		}
		if stock < u.Quantity*v.Weight*ticks {
			return false
		}
	}
	return true
}

// routeTicks returns the travel time of a route in the view, or 0 for none.
func routeTicks(v *economysim.AgentView, to economysim.RegionID, mode economysim.ModeID) int64 {
	for _, r := range v.Routes {
		if r.To == to && r.Mode == mode && r.Has {
			return r.Ticks
		}
	}
	return 0
}

// trnObs is the observer of the transport scenarios: the price of the watched
// commodity in each region, and the shipments in flight and in the books.
type trnObs struct {
	watch economysim.CommodityID
	price [2]int64
	// gapBP is, per tick, the gap between the dear and the cheap region in basis points of
	// the cheap price, and 0 when a price is not known yet.
	gapBP []int64
	// haulPrice is, per tick and region, the published price of the haul commodity.
	haul      economysim.CommodityID
	haulPrice [2][]int64
	// dispatched, arrived and haulUnits count the shipments, and shippedOf maps a
	// shipment in flight to its units; inFlight and maxInFlight are the units on the way.
	dispatched, arrived, haulUnits int64
	inFlight, maxInFlight          int64
	flight                         map[economysim.ShipmentID]int64
	// lineBroken counts arrived lines whose parts do not add up to the shipped units;
	// delivered and spoiled sum the units by what became of them.
	lineBroken, delivered, spoiled, diverted int64
	rejected                                 int64
	haulFlow                                 int64
}

func newTrnObs(watch, haul economysim.CommodityID) *trnObs {
	return &trnObs{watch: watch, haul: haul, flight: map[economysim.ShipmentID]int64{}}
}

// Observe keeps the series and the counts.
func (o *trnObs) Observe(e economysim.Event) {
	switch e := e.(type) {
	case economysim.MarketCleared:
		o.onMarket(e)
	case economysim.ShipmentDispatched:
		o.dispatched++
		o.haulUnits += e.HaulUnits
		var n int64
		for _, l := range e.Lines {
			n += int64(l.Quantity)
		}
		o.flight[e.Shipment] = n
		o.inFlight += n
		o.maxInFlight = max(o.maxInFlight, o.inFlight)
	case economysim.ShipmentArrived:
		o.arrived++
		o.inFlight -= o.flight[e.Shipment]
		delete(o.flight, e.Shipment)
		for _, l := range e.Lines {
			if l.Shipped != l.Delivered+l.Diverted+l.Spoiled {
				o.lineBroken++
			}
			o.delivered += int64(l.Delivered)
			o.diverted += int64(l.Diverted)
			o.spoiled += int64(l.Spoiled)
		}
	case economysim.ShipmentRejected:
		o.rejected++
	case economysim.BoundaryFlow:
		if e.Reason == economysim.ReasonHaulUse {
			o.haulFlow += e.Total
		}
	}
}

func (o *trnObs) onMarket(e economysim.MarketCleared) {
	if int(e.Region) >= 2 {
		return
	}
	if e.Commodity == o.watch {
		o.price[e.Region] = int64(e.Price)
	}
	if e.Commodity == o.haul {
		for int64(len(o.haulPrice[e.Region])) < e.Tick {
			o.haulPrice[e.Region] = append(o.haulPrice[e.Region], 0)
		}
		o.haulPrice[e.Region] = append(o.haulPrice[e.Region], int64(e.Price))
	}
	if e.Commodity == o.watch && e.Region == 1 { // the later of the two markets of the tick
		for int64(len(o.gapBP)) < e.Tick {
			o.gapBP = append(o.gapBP, 0)
		}
		_, _, gap, _ := o.dear()
		o.gapBP = append(o.gapBP, gap)
	}
}

// dear returns the dear and the cheap region of the watched commodity and the gap in
// basis points of the cheap price, ok false while a price is unknown.
func (o *trnObs) dear() (dear, cheap economysim.RegionID, gapBP int64, ok bool) {
	if o.price[0] < 1 || o.price[1] < 1 {
		return 0, 0, 0, false
	}
	dear, cheap = 0, 1
	if o.price[1] > o.price[0] {
		dear, cheap = 1, 0
	}
	return dear, cheap, (o.price[dear] - o.price[cheap]) * 10_000 / o.price[cheap], true
}

// trnRun builds the world with the audit and the observers, runs ticks steps under a
// slot of scenSem and returns the world.
func trnRun(t testing.TB, cfg economysim.Config, seed uint64, ticks int, strat economysim.Strategy, obs ...economysim.Observer) *economysim.World {
	t.Helper()
	scenSem <- struct{}{}
	defer func() { <-scenSem }()
	opts := []economysim.Option{economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithStrategy(trnName, strat)}
	for _, o := range obs {
		opts = append(opts, economysim.WithObserver(o))
	}
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d (seed %d) error = %v, want nil (the audit runs every tick)", w.Tick(), seed, err)
		}
	}
	return w
}

// richCarriers gives the carriers the stock to make many trips: 200 of fuel, prepared and
// staple, the goods of their upkeep and the cargo, and makes prepared keep (its shelf life
// of 4 would spoil the stock at tick 4). The whole account is the load of a self-carried
// trip (3.2.2), so the foot mode carries 100 times what the overlay's does. Households
// other than the carriers hold the upkeep of a tick or two and cannot leave for long.
func richCarriers(c *economysim.Config) {
	foot := &c.Modules.Transport.Modes[modeFootID]
	foot.UnitMass, foot.UnitBulk = 100*foot.UnitMass, 100*foot.UnitBulk
	for i := range c.Commodities {
		if c.Commodities[i].ID == "prepared" {
			c.Commodities[i].ShelfLife = 0
		}
	}
	i := slices.IndexFunc(c.Roles, func(r economysim.Role) bool { return r.ID == "carrier" })
	stock := slices.Clone(c.Roles[i].StartStock)
	for j := range stock {
		if stock[j].Commodity == "fuel" || stock[j].Commodity == "prepared" || stock[j].Commodity == "staple" {
			stock[j].Quantity = 200
		}
	}
	c.Roles[i].StartStock = stock
}

// s30Ship is the scripted shipper of S30: a carrier in the cheap region that holds the
// upkeep of the trip sends one unit of fuel to the first maker of the dear region when the
// gap exceeds the freight floor, one household in modulus at a time. A shipper that
// counts its pipeline (count) takes the units already on the way from the gap, as
// spec-transport.md 9.1 lets a strategy do with Inbound; one that does not, ignores them.
type s30Ship struct {
	trnBase
	w       trnWorld
	obs     *trnObs
	floorBP int64
	modulus int64
	count   bool
	// perUnitBP is what a unit on the way takes from the gap a counting shipper sees.
	perUnitBP int64
}

// Work leaves the carriers without a job: a household with a job in progress cannot leave
// on a trip (5.4 row 4), and the carriers' haul recipe has no demand in this scenario.
func (s s30Ship) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	if v.Role == s.w.carrier {
		return economysim.JobRequest{NoJob: true}
	}
	return economysim.DefaultStrategy{}.Work(v, rng)
}

func (s s30Ship) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	if v.Role != s.w.carrier {
		economysim.DefaultStrategy{}.Trade(v, rng, p)
		return // the carriers place no order: they live on their stock and ship it
	}
	dear, cheap, gap, ok := s.obs.dear()
	if !ok || v.Region != cheap || v.HasJob || (int64(v.Agent)+v.Tick.Tick)%s.modulus != 0 {
		return
	}
	if s.count {
		gap -= s.obs.inFlight * s.perUnitBP
	}
	line := economysim.Line{Commodity: s.w.fuel, Quantity: 1}
	k := routeTicks(v, dear, modeFootID)
	if gap < s.floorBP || k == 0 || !provisionedFor(v, k, line) {
		return
	}
	p.Shipments = append(p.Shipments, economysim.ShipmentRequest{Mode: modeFootID, To: dear, Consignee: s.w.first[dear], Lines: []economysim.Line{line}})
}

// s30Ticks is the length of a run of S30; the measures read ticks 60 to 150.
const s30Ticks = 150

// TestScenarioS30_PriceGapShipping runs S30 (TRN-A1, A2; D457): two regions with a route
// and a scripted shipper that sends fuel from the cheap region to the dear one on a
// price gap, once counting the units it already has on the way and once not, for a travel
// time k of 1, 2 and 4 ticks (the foot mode at speed 100, 50 and 25 over the edge of
// 100). It records, per seed, the shipments dispatched and arrived, the most units on the
// way, the units delivered and spoiled, and the mean and the largest gap in basis points
// over ticks 60 to 150; no band is asserted (O8). It asserts the exact facts: every line
// that arrived adds up (shipped = delivered + diverted + spoiled), the audit holds every
// tick, and the shipments dispatched are the shipments arrived plus those still in the
// snapshot.
//
// Requirement: FR-TRN-028, FR-TRN-032, FR-TRN-033, FR-TRN-049
func TestScenarioS30_PriceGapShipping(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	rep := newPackScenReport("two", "S30 price-gap shipping with a transit delay (TRN-A1, A2)")
	t.Run("cells", func(t *testing.T) {
		for _, k := range []int64{1, 2, 4} {
			for _, count := range []bool{false, true} {
				name := fmt.Sprintf("s30.k%d.%s", k, map[bool]string{false: "blind", true: "counting"}[count])
				wd := trnPack(t, 100/k, richCarriers)
				for _, seed := range seedList(scenarioSeedCount()) {
					t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
						t.Parallel()
						s30Seed(t, rep, name, wd, seed, count)
					})
				}
			}
		}
	})
	rep.flush(t)
}

// s30Seed runs one seed of one cell of S30 and records its measures and checks.
func s30Seed(t *testing.T, rep *scenReport, name string, wd trnWorld, seed uint64, count bool) {
	obs := newTrnObs(wd.fuel, wd.haul)
	st := s30Ship{w: wd, obs: obs, floorBP: 1_000, modulus: 4, count: count, perUnitBP: 50}
	w := trnRun(t, wd.cfg, seed, s30Ticks, st, obs)
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("%s seed %d: Snapshot() error = %v, want nil", name, seed, err)
	}
	var sum, peak int64
	window := obs.gapBP[min(60, len(obs.gapBP)):]
	for _, g := range window {
		sum += g
		peak = max(peak, g)
	}
	rep.num(seed, name+".dispatched", obs.dispatched)
	rep.num(seed, name+".arrived", obs.arrived)
	rep.num(seed, name+".max_units_on_the_way", obs.maxInFlight)
	rep.num(seed, name+".delivered", obs.delivered)
	rep.num(seed, name+".spoiled", obs.spoiled)
	if len(window) > 0 {
		rep.num(seed, name+".mean_gap_bp", sum/int64(len(window)))
		rep.num(seed, name+".largest_gap_bp", peak)
	} else {
		rep.absent(seed, name+".mean_gap_bp")
	}
	rep.check(seed, name+".lines_add_up", obs.lineBroken == 0, fmt.Sprintf("%d arrived lines do not add up", obs.lineBroken))
	rep.check(seed, name+".books", obs.dispatched == obs.arrived+int64(len(snap.Shipments)),
		fmt.Sprintf("dispatched %d, arrived %d, in the snapshot %d", obs.dispatched, obs.arrived, len(snap.Shipments)))
	if obs.lineBroken != 0 || obs.dispatched != obs.arrived+int64(len(snap.Shipments)) {
		t.Errorf("%s seed %d: lines that do not add up %d, dispatched %d, arrived %d, in flight %d", name, seed, obs.lineBroken, obs.dispatched, obs.arrived, len(snap.Shipments))
	}
}

// s31Ship ships one unit of prepared from every carrier to the first maker of the other
// region, at tick start.
type s31Ship struct {
	trnBase
	w     trnWorld
	start int64
}

func (s s31Ship) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	economysim.DefaultStrategy{}.Trade(v, rng, p)
	if v.Role != s.w.carrier || v.Tick.Tick != s.start || int(v.Region) >= 2 {
		return
	}
	to := economysim.RegionID(1 - int(v.Region))
	p.Shipments = append(p.Shipments, economysim.ShipmentRequest{
		Mode: modeFootID, To: to, Consignee: s.w.first[to], Lines: []economysim.Line{{Commodity: s.w.prepared, Quantity: 1}},
	})
}

// TestScenarioS31_PerishableAgainstTransitTime runs S31 (TRN-A5; D457): the commodity
// prepared, shelf life 4, shipped on foot by every carrier in the first tick it can,
// over a route of k ticks. A lot born at tick b with shelf life S is delivered if and
// only if the arrival tick is at most b + S - 1 (5.6.2); the lots are the starting stock,
// born at tick 0, shipped at tick 1, so a trip of k = 2 ticks arrives at tick 3 and is
// delivered, and one of k = 3 arrives at tick 4 and spoils whole. The warning
// route_too_slow_for_shelf_life is given for the second pack (k >= S is for a fresh lot
// of tick 4 only: it fires at k = 4) and not for the first.
//
// Arithmetic: the lot of start stock is born at tick 0; dispatched at tick 1, a trip of k
// arrives at tick 1 + k; delivered iff 1 + k <= 0 + 4 - 1, that is k <= 2. The warning is for the foot mode only: the coach
// (speed 30, 4 ticks) is also slower than the shelf life.
//
// Requirement: FR-TRN-025, FR-TRN-026, FR-TRN-038
func TestScenarioS31_PerishableAgainstTransitTime(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	rep := newPackScenReport("two", "S31 a perishable against the travel time (TRN-A5)")
	for _, tc := range []struct {
		k, speed     int64
		delivered    bool
		warnsTooSlow bool
	}{{2, 50, true, false}, {3, 40, false, false}, {4, 25, false, true}} {
		name := fmt.Sprintf("s31.k%d", tc.k)
		wd := trnPack(t, tc.speed, func(c *economysim.Config) {
			i := slices.IndexFunc(c.Roles, func(r economysim.Role) bool { return r.ID == "carrier" })
			c.Roles[i].StartStock = slices.Clone(c.Roles[i].StartStock)
			for j := range c.Roles[i].StartStock {
				if c.Roles[i].StartStock[j].Commodity == "prepared" {
					c.Roles[i].StartStock[j].Quantity = 10
				}
			}
		})
		warned := slices.ContainsFunc(wd.cfg.Warnings(), func(w economysim.Warning) bool {
			path := strings.Join(w.Path, "/")
			return w.Rule == "route_too_slow_for_shelf_life" && strings.Contains(path, "mode/foot/") && strings.HasSuffix(path, "/prepared")
		})
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				t.Parallel()
				obs := newTrnObs(wd.prepared, wd.haul)
				trnRun(t, wd.cfg, seed, 8, s31Ship{w: wd, start: 1}, obs)
				rep.num(seed, name+".dispatched", obs.dispatched)
				rep.num(seed, name+".delivered", obs.delivered)
				rep.num(seed, name+".spoiled", obs.spoiled)
				ok := obs.dispatched > 0 && obs.lineBroken == 0 && obs.diverted == 0 &&
					(tc.delivered && obs.spoiled == 0 && obs.delivered == obs.dispatched || !tc.delivered && obs.delivered == 0 && obs.spoiled == obs.dispatched)
				rep.check(seed, name+".delivered_iff_k_fits", ok,
					fmt.Sprintf("dispatched %d, delivered %d, spoiled %d, diverted %d", obs.dispatched, obs.delivered, obs.spoiled, obs.diverted))
				if !ok {
					t.Errorf("%s seed %d: dispatched %d delivered %d spoiled %d diverted %d, want %s", name, seed, obs.dispatched, obs.delivered, obs.spoiled, obs.diverted,
						map[bool]string{true: "all delivered", false: "all spoiled"}[tc.delivered])
				}
			})
		}
		rep.check(0, name+".warning_route_too_slow", warned == tc.warnsTooSlow, fmt.Sprintf("warned %t, want %t", warned, tc.warnsTooSlow))
		if warned != tc.warnsTooSlow {
			t.Errorf("%s: route_too_slow_for_shelf_life for prepared = %t, want %t", name, warned, tc.warnsTooSlow)
		}
	}
	t.Cleanup(func() { rep.flush(t) })
}
