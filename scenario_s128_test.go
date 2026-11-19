package economysim_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// blindMerchant is the merchant strategy with the expected loss of the routes taken out of its
// view: it plans the trips as the strategy did before the risk module existed (RSK-S3, "without
// the reading of LossBP"). Every other part of the strategy is the merchant's.
type blindMerchant struct{ strategy.Merchant }

// Trade hands the merchant a view whose routes carry no loss.
func (b blindMerchant) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	w := *v
	w.Routes = slices.Clone(v.Routes)
	for i := range w.Routes {
		w.Routes[i].LossBP = 0
	}
	b.Merchant.Trade(&w, rng, p)
}

// packMerchantsRisk is the warm merchants pack of pack two with the risk rows of the route pack.
var packMerchantsRisk = append(slices.Clone(packMerchantsWarm("two")), "overlay/risk.toml")

// brokersRemoved counts the AgentBankrupted of the role named id among events.
func brokersRemoved(t *testing.T, cfg economysim.Config, id string, events []economysim.Event) (n int64) {
	t.Helper()
	i := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == id })
	if i < 0 {
		t.Fatalf("role %q is not in the configuration", id)
	}
	role := economysim.RoleID(i)
	for _, e := range events {
		if b, ok := e.(economysim.AgentBankrupted); ok && b.Role == role {
			n++
		}
	}
	return n
}

// TestScenarioRSK3_MerchantLaneReadsLoss is RSK-S3 (S128, spec-risk.md 16, FR-RSK-027): the warm
// merchants pack of pack two with the risk rows, run twice per seed, once with the merchant
// strategy as it is (the revenue of a trip net of the expected loss of its route) and once with
// the same strategy blind to the loss, 400 ticks, seeds 1 to the scenario count, audit on. Exact:
// the books of the loss balance every tick in both runs. Recorded, never asserted (the bands of
// STY will judge, RSK-A1, A4, A5): the shipments dispatched and struck, the units delivered and
// wasted, and the brokers removed in each run.
//
// Requirement: FR-RSK-027, FR-RSK-014, FR-RSK-022
//
// Guard: the strategy that reads the loss and the one that does not must both keep the books;
// the comparison is reported so a first run of the pack shows what the reading is worth.
func TestScenarioRSK3_MerchantLaneReadsLoss(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packMerchantsRisk...)
	for _, v := range []struct {
		name string
		opts []economysim.Option
	}{
		{"reading", []economysim.Option{economysim.WithStrategy(strategy.MerchantName, strategy.Merchant{})}},
		{"blind", []economysim.Option{economysim.WithStrategy(strategy.MerchantName, blindMerchant{})}},
	} {
		techScenario(t, "two", "S128 two+merchants+risk the merchant "+v.name+" the expected loss, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
			rw := newRiskWatch(t, cfg, seed, v.opts)
			msg := rw.run(t, goldenTicks)
			checkExact(t, rep, "S128", "two", seed, "the loss boundary equals the events every tick", msg == "", "%s", msg)
			rep.num(seed, "shipments dispatched", rw.tally.dispatched)
			rep.num(seed, "shipments struck", rw.tally.struck)
			rep.num(seed, "units delivered", rw.tally.landed)
			rep.num(seed, "units wasted (spoiled, diverted, lost)", rw.tally.wasted)
			rep.num(seed, "brokers removed", brokersRemoved(t, cfg, "broker", rw.rec.events))
		})
	}
}
