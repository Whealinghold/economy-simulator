package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// This file holds the reruns of the transport and information scenarios with the merchant
// strategy in place of the scripted shipper (plan 2-26 D700, D702, D707): S30, S31, S32 and
// S46. They run on the cold packMerchants("two"), which loses its households (N-314), so a
// price gap may never form: the checks are the exact ones (the lines add up, the books, the
// audit every tick, the module's caps) and the gaps and counts are reported, never banded.
// Outcomes go to testdata/chains/expected/two.scenarios.txt with -update.

// merOut is what a rerun keeps: the observer of the transport scenarios, the merchants'
// per-tick plan counts, the shipments dispatched and the final snapshot.
type merOut struct {
	obs        *trnObs
	planned    []economysim.MerchantsPlanned
	dispatched []economysim.ShipmentDispatched
	snap       economysim.Snapshot
}

// merCapture keeps the events merRerun reads beyond trnObs.
type merCapture struct {
	planned    []economysim.MerchantsPlanned
	dispatched []economysim.ShipmentDispatched
}

func (c *merCapture) Observe(e economysim.Event) {
	switch e := e.(type) {
	case economysim.MerchantsPlanned:
		c.planned = append(c.planned, e)
	case economysim.ShipmentDispatched:
		c.dispatched = append(c.dispatched, e)
	}
}

// merCommodity returns the commodity of the id; the test fails if the pack has none.
func merCommodity(t testing.TB, cfg economysim.Config, id string) economysim.CommodityID {
	t.Helper()
	i := slices.IndexFunc(cfg.Commodities, func(c economysim.Commodity) bool { return c.ID == id })
	if i < 0 {
		t.Fatalf("the merchants pack has no commodity %q", id)
	}
	return economysim.CommodityID(i)
}

// merRole returns the index of the role of the id.
func merRole(t testing.TB, cfg economysim.Config, id string) int {
	t.Helper()
	i := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == id })
	if i < 0 {
		t.Fatalf("the merchants pack has no role %q", id)
	}
	return i
}

// merPack loads packMerchants("two") and lets mut edit it (the broker's block is a copy, so
// a mutation never reaches the pack another test loaded).
func merPack(t testing.TB, mut func(*economysim.Config)) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, packMerchants("two")...)
	b := &cfg.Roles[merRole(t, cfg, "broker")]
	blk := *b.Merchant
	blk.Modes = slices.Clone(blk.Modes)
	b.Merchant = &blk
	if mut != nil {
		mut(&cfg)
	}
	return cfg
}

// merRerun runs cfg for ticks steps under a slot of scenSem with the audit, the merchant
// strategy and any extra option, and returns what the scenarios read.
func merRerun(t testing.TB, cfg economysim.Config, seed uint64, ticks int, extra ...economysim.Option) merOut {
	t.Helper()
	scenSem <- struct{}{}
	defer func() { <-scenSem }()
	obs := newTrnObs(merCommodity(t, cfg, "fuel"), merCommodity(t, cfg, "haul"))
	cap := &merCapture{}
	opts := append(strategy.Options(), economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(obs), economysim.WithObserver(cap))
	w, err := economysim.NewWorld(cfg, append(opts, extra...)...)
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d (seed %d) error = %v, want nil (the audit runs every tick)", w.Tick(), seed, err)
		}
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot(seed %d) error = %v, want nil", seed, err)
	}
	return merOut{obs: obs, planned: cap.planned, dispatched: cap.dispatched, snap: snap}
}

// merTotals sums the plan counts of a run.
func merTotals(ps []economysim.MerchantsPlanned) (sum economysim.MerchantsPlanned) {
	for _, p := range ps {
		sum.Planned += p.Planned
		sum.Dispatched += p.Dispatched
		sum.Offered += p.Offered
		sum.Perishable += p.Perishable
		sum.Stale += p.Stale
		sum.Considered += p.Considered
	}
	return sum
}

// merExact records and asserts the exact facts every rerun shares: the lines that arrived
// add up and the shipments dispatched are those arrived plus those in the snapshot.
func merExact(t *testing.T, rep *scenReport, name string, seed uint64, o merOut) {
	t.Helper()
	books := o.obs.dispatched == o.obs.arrived+int64(len(o.snap.Shipments))
	checkExact(t, rep, name, "two", seed, name+".lines_add_up", o.obs.lineBroken == 0, "%d arrived lines do not add up", o.obs.lineBroken)
	checkExact(t, rep, name, "two", seed, name+".books", books, "dispatched %d, arrived %d, in the snapshot %d", o.obs.dispatched, o.obs.arrived, len(o.snap.Shipments))
}

// s30MerTicks is the length of a run of S30_WithMerchant; the gap is read over ticks 60 on.
const s30MerTicks = 150

// TestScenarioS30_WithMerchant reruns S30 (TRN-A1, A2) with the merchant strategy: the
// brokers carry by foot over a route of k = 1, 2 and 4 ticks (foot speed 100, 50 and 25 over
// the edge of 100) and a fee share spread of 0 and of 2 500. It asserts the exact facts: the
// lines that arrived add up (shipped = delivered + diverted + spoiled), the audit holds every
// tick, and the shipments dispatched are those arrived plus those in the snapshot. It
// reports the shipments dispatched and arrived and the mean and largest gap of fuel in basis
// points over ticks 60 to 150 (absent while a market has no price); no band is invented.
//
// Requirement: FR-TRN-028, FR-TRN-049, FR-MER-044
func TestScenarioS30_WithMerchant(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	rep := newPackScenReport("two", "S30 with the merchant: price-gap shipping with a transit delay (TRN-A1, A2)")
	for _, k := range []int64{1, 2, 4} {
		for _, spread := range []int64{0, 2_500} {
			name := fmt.Sprintf("s30m.k%d.spread%d", k, spread)
			cfg := merPack(t, func(c *economysim.Config) {
				c.Modules.Transport.Modes[modeFootID].Speed = 100 / k
				b := &c.Roles[merRole(t, *c, "broker")]
				b.Merchant.Modes = []string{"foot"}
				b.Merchant.FeeShareSpreadBP = spread
			})
			for _, seed := range seedList(scenarioSeedCount()) {
				t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
					t.Parallel()
					o := merRerun(t, cfg, seed, s30MerTicks)
					merExact(t, rep, name, seed, o)
					rep.num(seed, name+".dispatched", o.obs.dispatched)
					rep.num(seed, name+".arrived", o.obs.arrived)
					window := o.obs.gapBP[min(60, len(o.obs.gapBP)):]
					var sum, peak int64
					for _, g := range window {
						sum += g
						peak = max(peak, g)
					}
					if len(window) == 0 {
						rep.absent(seed, name+".mean_gap_bp")
						return
					}
					rep.num(seed, name+".mean_gap_bp", sum/int64(len(window)))
					rep.num(seed, name+".largest_gap_bp", peak)
				})
			}
		}
	}
	t.Cleanup(func() { rep.flush(t) })
}

// TestScenarioS31_WithMerchant reruns S31 (TRN-A5) with the merchant strategy: the commodity
// prepared (shelf life 4) is added to the brokers' trades and they carry by foot over a
// route of k = 2 (speed 50) and k = 3 (speed 40) ticks. FR-MER-010 sets a lot aside iff
// k > S - 1 - max(1, OldestAge); for a lot of age 1 that is k > 4 - 1 - 1 = 2, so with k = 3
// a lot of prepared is Perishable and none is dispatched, and with k = 2 it passes (the
// originals' delivery rule, arrival 1 + k against 4 - 1, gives the same split). It asserts
// that with k = 3 the Perishable count is above 0 and no shipment of prepared is
// dispatched, that with k = 2 every unit of prepared that is dispatched is delivered
// (none spoils), the lines add up and the books; the shipments of prepared are reported.
//
// Requirement: FR-MER-010, FR-TRN-025, FR-TRN-026
func TestScenarioS31_WithMerchant(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	rep := newPackScenReport("two", "S31 with the merchant: a perishable against the travel time (TRN-A5)")
	for _, tc := range []struct{ k, speed int64 }{{2, 50}, {3, 40}} {
		name := fmt.Sprintf("s31m.k%d", tc.k)
		cfg := merPack(t, func(c *economysim.Config) {
			c.Modules.Transport.Modes[modeFootID].Speed = tc.speed
			b := &c.Roles[merRole(t, *c, "broker")]
			b.Merchant.Modes = []string{"foot"}
			b.Trades = append(slices.Clone(b.Trades), "prepared")
		})
		prepared := merCommodity(t, cfg, "prepared")
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				t.Parallel()
				o := merRerun(t, cfg, seed, 40)
				merExact(t, rep, name, seed, o)
				var sent int64
				for _, d := range o.dispatched {
					for _, l := range d.Lines {
						if l.Commodity == prepared {
							sent += int64(l.Quantity)
						}
					}
				}
				perishable := merTotals(o.planned).Perishable
				rep.num(seed, name+".prepared_dispatched", sent)
				rep.num(seed, name+".perishable", perishable)
				if tc.k == 3 {
					checkExact(t, rep, name, "two", seed, name+".set_aside", perishable > 0 && sent == 0,
						"perishable %d, units of prepared dispatched %d, want Perishable above 0 and none dispatched", perishable, sent)
					return
				}
				checkExact(t, rep, name, "two", seed, name+".none_spoils", o.obs.spoiled == 0, "%d units spoiled in transit, want 0 for k = 2", o.obs.spoiled)
			})
		}
	}
	t.Cleanup(func() { rep.flush(t) })
}

// TestScenarioS32_WithMerchant reruns S32 (TRN-A3) with the merchant strategy: the brokers
// hire the coach (the overlay's mode) and the carriers start the haul recipe with a
// probability that rises with the haul price at the rate of the elasticity, 1, 4 and 16,
// through the scripted Work of S32 (s32Strategy); the brokers keep the merchant. It asserts
// the exact facts: the lines add up, the books, the audit every tick and the haul units
// redeemed by the dispatches are the haul_use flow. It reports the shipments dispatched, the
// haul units redeemed and the range of the haul price in the early window (ticks 20 to 120)
// and the late one (ticks 200 to 300); the cobweb is reported only, as S70 reports
// FR-MER-048 (no band is invented).
//
// Requirement: FR-TRN-043, FR-CAR-003, FR-MER-048
func TestScenarioS32_WithMerchant(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	rep := newPackScenReport("two", "S32 with the merchant: the haul capacity cobweb (TRN-A3)")
	for _, e := range []int64{1, 4, 16} {
		name := fmt.Sprintf("s32m.e%d", e)
		cfg := merPack(t, nil)
		cfg.Roles[merRole(t, cfg, "carrier")].Strategy = trnName
		st := s32Strategy{elasticity: e, modulus: 6}
		st.w.carrier = economysim.RoleID(merRole(t, cfg, "carrier"))
		st.w.haul = merCommodity(t, cfg, "haul")
		st.w.runHaul = economysim.RecipeID(slices.IndexFunc(cfg.Recipes, func(r economysim.Recipe) bool { return r.ID == "run_haul" }))
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				t.Parallel()
				o := merRerun(t, cfg, seed, s32Ticks, economysim.WithStrategy(trnName, st))
				merExact(t, rep, name, seed, o)
				checkExact(t, rep, name, "two", seed, name+".haul_use_is_the_flow", o.obs.haulUnits == o.obs.haulFlow,
					"redeemed %d, haul_use flow %d", o.obs.haulUnits, o.obs.haulFlow)
				rep.num(seed, name+".dispatched", o.obs.dispatched)
				rep.num(seed, name+".haul_units_redeemed", o.obs.haulUnits)
				early, okE := rangeOf(o.obs.haulPrice[0], 20, 120)
				late, okL := rangeOf(o.obs.haulPrice[0], 200, 300)
				if !okE || !okL {
					rep.absent(seed, name+".haul_price_range_late")
					return
				}
				rep.num(seed, name+".haul_price_range_early", early)
				rep.num(seed, name+".haul_price_range_late", late)
			})
		}
	}
	t.Cleanup(func() { rep.flush(t) })
}

// TestScenarioS46_MerchantRumouredGap is S46 (INF-S1, the rumour-lag cells that
// spec-reference-chain.md 7.9 gives to S67: "it is S46 and S30 with the merchant"): the S67
// world with the information speed (info_speed) at 100, 50 and 25 over edges of 100 and 130,
// so the rumours lag by 2 to 7 ticks, and a fee share spread of 0, 1 000 and 2 500. It
// asserts the caps of FR-MER-025 (a region plans at most one request and one offer per
// merchant a tick), that a rumour older than MaxSourceAge is never acted on (the stale count
// is 0 with MaxSourceAge at the bound, FR-MER-009 as S69 asserts it), the lines add up, the
// books and the audit every tick; it reports the trips planned and the rumours counted
// stale per lag.
//
// Arithmetic: a lag is 1 + ceil(distance / speed) (spec-information.md 12.1): 100 over 100 is
// 2 and 130 over 25 is 1 + 6 = 7.
//
// Requirement: FR-INF-017, FR-MER-025, FR-MER-024
func TestScenarioS46_MerchantRumouredGap(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	rep := newPackScenReport("two", "S46 a merchant on a rumoured gap (INF-S1)")
	for _, speed := range []int64{100, 50, 25} {
		for _, spread := range []int64{0, 1_000, 2_500} {
			name := fmt.Sprintf("s46m.speed%d.spread%d", speed, spread)
			cfg := merPack(t, func(c *economysim.Config) {
				c.Modules.Information.InfoSpeed = speed
				b := &c.Roles[merRole(t, *c, "broker")]
				b.Merchant.FeeShareSpreadBP = spread
				b.Merchant.MaxSourceAge = economysim.MaxRumourAge
			})
			for _, seed := range seedList(scenarioSeedCount()) {
				t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
					t.Parallel()
					o := merRerun(t, cfg, seed, 60)
					merExact(t, rep, name, seed, o)
					var over int64
					for _, p := range o.planned {
						if p.Dispatched > p.Merchants || p.Offered > p.Merchants {
							over++
						}
					}
					sum := merTotals(o.planned)
					checkExact(t, rep, name, "two", seed, name+".caps", over == 0, "%d region-ticks plan more requests or offers than merchants", over)
					checkExact(t, rep, name, "two", seed, name+".none_stale", sum.Stale == 0, "%d rumours counted stale at MaxSourceAge = %d", sum.Stale, economysim.MaxRumourAge)
					rep.num(seed, name+".planned", sum.Planned)
					rep.num(seed, name+".dispatched", o.obs.dispatched)
				})
			}
		}
	}
	t.Cleanup(func() { rep.flush(t) })
}
