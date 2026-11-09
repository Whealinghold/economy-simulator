package economysim_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds what the scenarios S47 to S50 of the information module share
// (plan 2-13 Phase 7): the packs with the information overlay, the recording
// strategy (a hand-written fake registered with WithStrategy, architecture.md 10.3)
// and the world builder. Every scenario runs on a reference pack with its trade
// overlay and then the information overlay (spec-reference-chain.md 4.5).

// infoName is the name the recording strategy is registered under.
const infoName = "recording"

// packInformation is the module-alone pack of the variant: the cold pack, its trade
// overlay (the edges) and its information overlay.
func packInformation(variant string) []string {
	return append(slices.Clone(packLayers[variant]), tradeOverlay(variant), "overlay/information-"+variant+".toml")
}

// seenView is what the recording strategy keeps of one call of Trade.
type seenView struct {
	tick    int64
	region  economysim.RegionID
	agent   economysim.AgentID
	rumours []economysim.Rumour
}

// recordingStrategy logs the rumours of every view it is given. With quote set it
// places no order except the quotes of the agents in quoters: one unit of the
// commodity at the price quote returns for the tick, the asker and the bidder at
// the same limit, so the market publishes exactly that price. With quote nil it
// trades as the default strategy does. It keeps no state the world needs, so a
// restore of a world that follows it needs nothing of it (architecture.md 7.7).
type recordingStrategy struct {
	seen      []seenView
	commodity economysim.CommodityID
	quoters   map[economysim.AgentID]economysim.Side
	quote     func(tick int64) (economysim.Price, bool)
}

func (s *recordingStrategy) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	return economysim.DefaultStrategy{}.Work(v, rng)
}

func (s *recordingStrategy) Revise(o *economysim.OutcomeView, rng *rand.Rand) economysim.Belief {
	return economysim.DefaultStrategy{}.Revise(o, rng)
}

func (s *recordingStrategy) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	s.seen = append(s.seen, seenView{v.Tick.Tick, v.Region, v.Agent, slices.Clone(v.Rumours)})
	if s.quote == nil {
		economysim.DefaultStrategy{}.Trade(v, rng, p)
		return
	}
	side, ok := s.quoters[v.Agent]
	if !ok {
		return
	}
	if price, ok := s.quote(v.Tick.Tick); ok {
		p.Intents = append(p.Intents, economysim.Intent{Commodity: s.commodity, Side: side, Quantity: 1, Limit: price})
	}
}

var _ economysim.Strategy = (*recordingStrategy)(nil)

// infoRun is a world built for a scenario with what it recorded.
type infoRun struct {
	w      *economysim.World
	rec    *recorder
	strat  *recordingStrategy
	cfg    economysim.Config
	region map[string]economysim.RegionID
}

// roleID returns the RoleID of a role of the configuration.
func roleID(t testing.TB, cfg economysim.Config, id string) economysim.RoleID {
	t.Helper()
	i := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == id })
	if i < 0 {
		t.Fatalf("the pack has no role %q", id)
	}
	return economysim.RoleID(i)
}

// commodityID returns the CommodityID of a commodity of the configuration.
func commodityID(t testing.TB, cfg economysim.Config, id string) economysim.CommodityID {
	t.Helper()
	i := slices.IndexFunc(cfg.Commodities, func(c economysim.Commodity) bool { return c.ID == id })
	if i < 0 {
		t.Fatalf("the pack has no commodity %q", id)
	}
	return economysim.CommodityID(i)
}

// newInfoRun loads the pack, lets mut edit the configuration, makes every role follow
// the recording strategy (built by mkStrat from the first two households of the
// grower role in the first region, the asker and the bidder), and builds the world
// with the audit on. The strategy sees the agent IDs through the construction events.
func newInfoRun(t testing.TB, layers []string, seed uint64, mut func(*economysim.Config), s *recordingStrategy) *infoRun {
	t.Helper()
	cfg := loadValidPack(t, layers...)
	if mut != nil {
		mut(&cfg)
	}
	for i := range cfg.Roles {
		cfg.Roles[i].Strategy = infoName
	}
	r := &infoRun{cfg: cfg, rec: &recorder{}, strat: s, region: map[string]economysim.RegionID{}}
	for i, reg := range cfg.Regions {
		r.region[reg.ID] = economysim.RegionID(i)
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithStrategy(infoName, s), economysim.WithObserver(r.rec), economysim.WithAudit())
	if err != nil {
		t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
	}
	r.w = w
	return r
}

// quoters returns the first two agents of the role in the region, from the events of
// construction: the first asks, the second bids.
func (r *infoRun) quoters(t testing.TB, role economysim.RoleID, region economysim.RegionID) map[economysim.AgentID]economysim.Side {
	t.Helper()
	var ids []economysim.AgentID
	for _, e := range r.rec.events {
		if a, ok := e.(economysim.AgentSpawned); ok && a.Role == role && a.Region == region {
			ids = append(ids, a.Agent)
		}
	}
	if len(ids) < 2 {
		t.Fatalf("region %d has %d agents of role %d, want 2 or more for a quote", region, len(ids), role)
	}
	return map[economysim.AgentID]economysim.Side{ids[0]: economysim.Ask, ids[1]: economysim.Bid}
}

// run steps the world n ticks.
func (r *infoRun) run(t testing.TB, n int) {
	t.Helper()
	for range n {
		if err := r.w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", r.w.Tick(), err)
		}
	}
}

// rumoursAbout returns, in tick order, the rumours about (region, commodity) that the
// views of readers in the reader region carried, with the tick of the view.
func (s *recordingStrategy) rumoursAbout(reader, region economysim.RegionID, c economysim.CommodityID) (ticks []int64, out []economysim.Rumour) {
	for _, v := range s.seen {
		if v.region != reader {
			continue
		}
		for _, ru := range v.rumours {
			if ru.Region == region && ru.Commodity == c {
				ticks, out = append(ticks, v.tick), append(out, ru)
				break
			}
		}
	}
	return ticks, out
}

// thinWorld builds the world of S47 and S50: pack two with the information overlay,
// the error set, and two growers of the north quoting staple at the price quote
// returns. Everyone else places no order, so the north staple market publishes the
// quote alone.
func thinWorld(t testing.TB, seed uint64, errorBP int64, quote func(int64) (economysim.Price, bool)) (*infoRun, economysim.CommodityID) {
	t.Helper()
	s := &recordingStrategy{quote: quote}
	r := newInfoRun(t, packInformation("two"), seed, func(c *economysim.Config) {
		c.Modules.Information.RumourErrorBP = errorBP
		// The asker sells a unit a trade tick for the whole run: its stock is large enough
		// that the upkeep reserve never stops the quote.
		i := slices.IndexFunc(c.Roles, func(r economysim.Role) bool { return r.ID == "grower" })
		for j := range c.Roles[i].StartStock {
			if c.Roles[i].StartStock[j].Commodity == "staple" {
				c.Roles[i].StartStock[j].Quantity = 1_000
			}
		}
	}, s)
	s.commodity = commodityID(t, r.cfg, "staple")
	s.quoters = r.quoters(t, roleID(t, r.cfg, "grower"), r.region["north"])
	return r, s.commodity
}

// infoCheck records the verdict of a check in the report and fails the test when it
// does not hold: the checks of S47 to S50 are exact (D181 records, it does not judge).
func infoCheck(t testing.TB, rep *scenReport, seed uint64, name string, ok bool, detail string) {
	t.Helper()
	rep.check(seed, name, ok, detail)
	if !ok {
		t.Errorf("seed %d: %s: %s", seed, name, detail)
	}
}
