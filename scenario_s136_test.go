package economysim_test

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packThreeTransport is pack three with its trade and transport overlays (spec-reference-chain.md
// 7.12): north, south and far_south on a line, edges of 100.
var packThreeTransport = append(slices.Clone(packLayers["three"]), tradeOverlay("three"), "overlay/transport-three.toml")

// s136Doc is the document of FIS-S5 (spec-fiscal.md 19) on pack three: the crown governs everything,
// with the inland (the north) and the coast (the south and the far south) below it. The coast charges 3
// a carrying unit on entering it, the inland 2 on leaving it, the coast taxes the households of the south
// one a tick, and the crown takes a quarter of the coast's treasury above 1 000 by a remittance. Every number is a
// labelled placeholder (N-018, N-027). The merchants of the pack are not configured: the scripted
// shipper of the transport tests is the shipper, so the dispatches and their tolls are exact.
const s136Doc = `
[modules.polity.jurisdiction.crown]
seat = "north"
start_money = 5000

[modules.polity.jurisdiction.inland]
parent = "crown"
seat = "north"
start_money = 5000

[modules.polity.jurisdiction.coast]
parent = "crown"
seat = "south"
start_money = 5000

[modules.polity.member]
north = "inland"
south = "coast"
far_south = "coast"

[modules.fiscal.toll.gate]
authority = "coast"
on = "enter"
amount = 3

[modules.fiscal.toll.ferry]
authority = "inland"
on = "exit"
amount = 2

[modules.fiscal.tax.poll]
authority = "coast"
target = { kind = "households", region = "south" }
amount = 1

[modules.fiscal.tax.remit]
authority = "crown"
target = { kind = "treasury", jurisdiction = "coast" }
rate_bp = 2500
exempt = 1000
`

// s136Line is the position of each region of pack three on its line, and s136InCoast says which regions
// the coast covers (the inland covers the north only).
var (
	s136Line    = map[string]int{"north": 0, "south": 1, "far_south": 2}
	s136InCoast = map[string]bool{"south": true, "far_south": true}
)

// s136Toll is the toll of a shipment of units carrying units from region a to region b of pack three: 3
// for each unit for each edge that enters the coast and 2 for each edge that leaves the inland (FIS 7.2).
func s136Toll(a, b string, units int64) economysim.Money {
	step := 1
	if s136Line[b] < s136Line[a] {
		step = -1
	}
	at := map[int]string{0: "north", 1: "south", 2: "far_south"}
	var per int64
	for p := s136Line[a]; p != s136Line[b]; p += step {
		u, v := at[p], at[p+step]
		if !s136InCoast[u] && s136InCoast[v] {
			per += 3
		}
		if u == "north" && v != "north" {
			per += 2
		}
	}
	return economysim.Money(per * units)
}

// threeShipper is the scripted shipper of the transport tests on three regions: every other agent
// sends a unit of staple on foot every seventh tick, to the next region of the ring one time and to the
// one after it the next, so that the dispatches cross the coast's border in both directions (the shipper
// of the transport tests knows two regions and the carriers of pack three live in one).
type threeShipper struct{ shipper }

// Trade ships on the scripted ticks, once the household can hold the upkeep of the trip (provisioned).
func (s threeShipper) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	economysim.DefaultStrategy{}.Trade(v, rng, p)
	if v.Tick.Tick%s.shipEvery != 1 || v.Agent%2 != 0 || int(v.Region) >= len(s.first) {
		return
	}
	to := economysim.RegionID((int(v.Region) + 1 + int(v.Tick.Tick/s.shipEvery)%2) % len(s.first))
	if !s.provisioned(v, to) {
		return
	}
	p.Shipments = append(p.Shipments, economysim.ShipmentRequest{
		Mode: modeFootID, To: to, Consignee: s.first[to], Lines: []economysim.Line{{Commodity: s.staple, Quantity: 1}},
	})
}

// s136Out is what one run keeps: the dispatch count, those with a wrong toll, the sum of the tolls the
// dispatches report and the treasuries' ends.
type s136Out struct {
	dispatched, wrong, leaving int
	tolls                      economysim.Money
	start, end                 []economysim.Money
	k                          *fiscalKept
}

// tollJudge is the observer of S136: the counters of fiscalKept and the check of each dispatch.
type tollJudge struct {
	*fiscalKept
	names []string
	out   *s136Out
}

// Observe checks each dispatch against the toll of FIS 7.2 and forwards every event.
func (j *tollJudge) Observe(e economysim.Event) {
	if d, ok := e.(economysim.ShipmentDispatched); ok {
		j.out.dispatched++
		j.out.tolls += d.Toll
		if from, to := j.names[d.Region], j.names[d.To]; from == "north" && to != "north" {
			j.out.leaving++
		}
		if want := s136Toll(j.names[d.Region], j.names[d.To], d.Units); d.Toll != want {
			j.out.wrong++
		}
	}
	j.fiscalKept.Observe(e)
}

// s136World builds the world of pack three with the document and the scripted shipper; opts are the
// options that a restore needs again (the shipper), without the seed, the audit and the observer.
func s136World(t testing.TB, seed uint64) (*economysim.World, economysim.Config, []economysim.Option, *s136Out) {
	t.Helper()
	cfg, base := shipperFor(t, loadPackBytes(t, []byte(s136Doc), packThreeTransport...))
	opts := []economysim.Option{economysim.WithStrategy("ship", threeShipper{base})}
	out := &s136Out{k: newFiscalKept()}
	probe := mustWorld(t, cfg, opts...)
	s, err := probe.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	j := &tollJudge{fiscalKept: out.k, names: s.Names.Regions, out: out}
	w := mustWorld(t, cfg, append(slices.Clone(opts), economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(j))...)
	out.start = treasuryMoneyRows(t, w)
	return w, cfg, opts, out
}

// TestScenarioS136_TollsAndRemittance reports FIS-S5 of spec-fiscal.md 19 (STB-L179) on pack three with
// its trade and transport overlays, seeds 1 to 10, 150 ticks, audit on, the scripted shipper sending a
// unit every few ticks: tolls at the coast's border, a head tax of the coast and a remittance up the
// chain. Every figure is exact: each dispatch pays 3 a unit for each edge that enters the coast and 2 for
// each edge that leaves it (FIS 7.2), the dispatches' tolls add up to the toll events, the coast's
// treasury ends at its start plus the tolls and the tax less the remittance, the crown's at its start
// plus the remittance; and a world checkpointed at tick 60 and restored (the remittance carries a
// remainder) ends in the snapshot of the world that was not interrupted. The refusals for
// toll_unaffordable and the trade volume are reported, never asserted (N-027).
//
// Requirement: FR-FIS-034, FR-FIS-035, FR-FIS-036, FR-FIS-023, FR-FIS-060, FR-FIS-002
//
// Guard: a toll charged on the wrong edge, on a pass-through, or twice, or a carry lost in a checkpoint,
// shows as a wrong dispatch, a broken identity or a restored world that drifts.
func TestScenarioS136_TollsAndRemittance(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	fiscalScenario(t, "three", "S136 three+fiscal tolls, a tax and a remittance, 150 ticks", scenarioSeedCount(), func(t *testing.T, rep *scenReport, seed uint64) {
		c := fiscalChecker{t, rep, "S136", "three", seed}
		w, cfg, opts, o := s136World(t, seed)
		fiscalStepTo(t, w, 60)
		cp, err := w.Checkpoint()
		if err != nil {
			t.Fatalf("seed %d: Checkpoint() error = %v, want nil", seed, err)
		}
		fiscalStepTo(t, w, 150)
		o.end = treasuryMoneyRows(t, w)
		k := o.k
		tolls := k.applied["gate"] + k.applied["ferry"]
		c.check("shipments were dispatched", o.dispatched > 0 && o.tolls > 0, "%d dispatches, tolls %d", o.dispatched, o.tolls)
		c.check("every dispatch pays the toll of FIS 7.2", o.wrong == 0, "%d of %d dispatches paid another toll", o.wrong, o.dispatched)
		c.check("toll events add up to the dispatches' tolls", tolls == o.tolls, "events %d, dispatches %d", tolls, o.tolls)
		// Jurisdiction rows are in ascending ID: the coast (0), the crown (1) and the inland (2).
		coast, crown, inland := 0, 1, 2
		c.check("coast = start + gate + tax - remittance",
			o.end[coast] == o.start[coast]+k.applied["gate"]+k.applied["poll"]-k.applied["remit"], "end %d, start %d, gate %d, tax %d, remittance %d",
			o.end[coast], o.start[coast], k.applied["gate"], k.applied["poll"], k.applied["remit"])
		c.check("inland = start + ferry", o.end[inland] == o.start[inland]+k.applied["ferry"],
			"end %d, start %d, ferry %d", o.end[inland], o.start[inland], k.applied["ferry"])
		c.check("crown = start + remittance", o.end[crown] == o.start[crown]+k.applied["remit"],
			"end %d, start %d, remittance %d", o.end[crown], o.start[crown], k.applied["remit"])
		s136Restore(c, cfg, opts, cp, w)
		c.check("a dispatch leaves the inland", o.leaving > 0, "%d of %d dispatches leave the inland", o.leaving, o.dispatched)
		rep.num(seed, "dispatches", int64(o.dispatched))
		rep.num(seed, "tolls paid", int64(o.tolls))
		rep.num(seed, "refused for toll_unaffordable", int64(k.refused))
		rep.num(seed, "remitted", int64(k.applied["remit"]))
		rep.num(seed, "head tax collected", int64(k.applied["poll"]))
	})
}

// s136Restore restores the checkpoint of tick 60 under the same configuration, steps it to tick 150 and
// checks its snapshot is the one of the world that was not interrupted.
func s136Restore(c fiscalChecker, cfg economysim.Config, opts []economysim.Option, cp economysim.Checkpoint, w *economysim.World) {
	t, seed := c.t, c.seed
	t.Helper()
	r, err := economysim.RestoreWorld(cfg, cp, append(slices.Clone(opts), economysim.WithAudit())...)
	if err != nil {
		t.Fatalf("seed %d: RestoreWorld() error = %v, want nil", seed, err)
	}
	fiscalStepTo(t, r, 150)
	want, err := w.Snapshot()
	if err != nil {
		t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
	}
	got, err := r.Snapshot()
	if err != nil {
		t.Fatalf("seed %d: Snapshot() of the restored world error = %v, want nil", seed, err)
	}
	c.check("a restored world ends where the uninterrupted one does", reflect.DeepEqual(got, want), "the snapshots at tick 150 differ")
}
