package economysim

import (
	"reflect"
	"testing"
)

// TestView_MerchantGroup verifies the merchant group of the view (spec-merchants.md
// 4.1): it is present in the view passed to Trade of an agent whose role has a
// block and in no other, it carries the block by value (with no Modes: they are
// in Modes), the fee share, the modes of the block in mode order and the goods of
// the tradable set in canonical order with their definitions, and its slices are
// capped so a strategy that appends changes nothing.
//
// The broker trades grain and haul: grain has Mass 3 and Bulk 2 and haul none
// (shipConfig), neither spoils. The coach is the first mode (ModeID 0) and hauls
// "haul".
//
// Requirement: FR-MER-027
func TestView_MerchantGroup(t *testing.T) {
	t.Parallel()
	w, _, s := newMerchWorld(t, nil, nil)
	step(t, w, 2)
	if len(s.seen) == 0 {
		t.Fatal("the script saw no view")
	}
	haul, _ := w.cfg.commodityID("haul")
	for _, v := range s.seen {
		isBroker := v.agent == brokerHome || v.agent == brokerFar
		if v.has != isBroker {
			t.Errorf("tick %d agent %d: HasMerchant = %v, want %v (only the brokers have a block)", v.tick, v.agent, v.has, isBroker)
		}
		if !isBroker {
			if !reflect.DeepEqual(v.merchant, MerchantView{}) {
				t.Errorf("agent %d: Merchant = %+v, want the zero value for an agent with no block", v.agent, v.merchant)
			}
			continue
		}
		m := v.merchant
		want := DefaultMerchant()
		want.Modes, want.Consigns, want.TakeUnits = nil, true, 10
		if !reflect.DeepEqual(m.Params, want) || m.FeeShare != DefaultMerchantFeeShare {
			t.Errorf("agent %d: Params = %+v, FeeShare = %d, want %+v and %d", v.agent, m.Params, m.FeeShare, want, DefaultMerchantFeeShare)
		}
		if len(m.Modes) != 1 || m.Modes[0].Mode != modeCoach || !m.Modes[0].Hired || m.Modes[0].Haul != haul || m.Modes[0].HaulSpan != 10 || m.Modes[0].UnitMass != 1_200 {
			t.Errorf("agent %d: Modes = %+v, want the coach (hired, hauling %d, span 10, unit mass 1 200)", v.agent, m.Modes, haul)
		}
		grain, _ := w.cfg.commodityID("grain")
		if len(m.Goods) != 2 || m.Goods[0].Commodity != min(grain, haul) || m.Goods[1].Commodity != max(grain, haul) {
			t.Fatalf("agent %d: Goods = %+v, want grain and haul in canonical order", v.agent, m.Goods)
		}
		for _, g := range m.Goods {
			if g.Commodity == grain && (g.Mass != 3 || g.Bulk != 2 || g.ShelfLife != 0) {
				t.Errorf("grain row = %+v, want Mass 3, Bulk 2, ShelfLife 0", g)
			}
		}
	}
}

// TestView_MerchantGroupOldestAge verifies OldestAge of a good (spec-merchants.md
// 4.1): -1 when the agent holds none, the ticks since the born tick of the oldest
// lot of a commodity that spoils, and 0 for a held commodity that never spoils.
//
// "perish" has a shelf life of 3; the broker is given 2 units at tick 0 and 1
// unit at tick 1, so at tick 2 the oldest lot is 2 ticks old.
//
// Requirement: FR-MER-027
func TestView_MerchantGroupOldestAge(t *testing.T) {
	t.Parallel()
	w, _, s := newMerchWorld(t, func(c *Config) {
		broker := &c.Roles[len(c.Roles)-1]
		broker.Trades = []string{"grain", "haul", "perish"}
	}, nil)
	perish, _ := w.cfg.commodityID("perish")
	grain, _ := w.cfg.commodityID("grain")
	give(t, w, brokerHome, grain, 4)
	step(t, w, 1)
	give(t, w, brokerHome, perish, 2)
	step(t, w, 1)
	give(t, w, brokerHome, perish, 1)
	step(t, w, 1)
	ages := map[int64]map[CommodityID]int64{}
	for _, v := range s.seen {
		if v.agent != brokerHome {
			continue
		}
		ages[v.tick] = map[CommodityID]int64{}
		for _, g := range v.merchant.Goods {
			ages[v.tick][g.Commodity] = g.OldestAge
		}
	}
	// Tick 0: nothing of perish; tick 1: the 2 units born at 1 are 0 old; tick 2:
	// the oldest lot was born at tick 1, so it is 1 old.
	for _, tt := range []struct {
		tick int64
		c    CommodityID
		want int64
	}{{0, perish, -1}, {1, perish, 0}, {2, perish, 1}, {0, grain, 0}} {
		if got := ages[tt.tick][tt.c]; got != tt.want {
			t.Errorf("OldestAge of commodity %d at tick %d = %d, want %d", tt.c, tt.tick, got, tt.want)
		}
	}
}

// TestView_MerchantGroupIsScratch verifies that the slices of the group have
// their capacity capped and that a strategy that writes to a good or a mode
// changes no later view: the goods are a scratch copy, refilled for each call.
//
// Guard: the filler copies the static table of the role into the scratch each
// call; if it handed out the table itself, one strategy's write would corrupt
// every later merchant's view.
//
// Requirement: FR-MER-027
func TestView_MerchantGroupIsScratch(t *testing.T) {
	t.Parallel()
	w, _, s := newMerchWorld(t, nil, func(v *AgentView, _ *TradePlan) {
		if !v.HasMerchant {
			return
		}
		m := &v.Merchant
		if cap(m.Goods) != len(m.Goods) || cap(m.Modes) != len(m.Modes) {
			t.Errorf("capacities = %d of %d goods and %d of %d modes, want them capped to the length", cap(m.Goods), len(m.Goods), cap(m.Modes), len(m.Modes))
		}
		m.Goods[0].Mass, m.Goods[0].OldestAge = 999, 999
	})
	step(t, w, 3)
	for _, v := range s.seen {
		if v.has && v.merchant.Goods[0].Mass == 999 {
			t.Errorf("tick %d agent %d: a write to a good of an earlier view reached this one", v.tick, v.agent)
		}
	}
}

// notesOf is the plan notes the scripts below report: a distinct count for each
// field, scaled by k, so a field summed into the wrong one shows.
func notesOf(k int64) PlanNotes {
	return PlanNotes{Considered: 1 * k, NoRumour: 2 * k, Stale: 3 * k, Locked: 4 * k, Perishable: 5 * k, NoMargin: 6 * k,
		Unaffordable: 7 * k, BelowShare: 8 * k, Planned: 9 * k, Dispatched: 10 * k, Offered: 11 * k, Taken: 12 * k,
		Rested: 13 * k, Waiting: 14 * k, LotCapped: 15 * k}
}

// plannedAt returns the MerchantsPlanned events of a tick, in emission order.
func plannedAt(rec *evLog, tick int64) []MerchantsPlanned {
	var out []MerchantsPlanned
	for _, e := range of[MerchantsPlanned](rec) {
		if e.Tick == tick {
			out = append(out, e)
		}
	}
	return out
}

// TestPlanNotes_SummedPerRegion verifies that the world adds the plan notes of the
// agents with a merchant block into one MerchantsPlanned per region: two brokers
// in a_home report k = 1 and k = 2 (the fixture gives agents a counter) and one in
// b_far reports k = 4, so the event of a_home carries 3 times each count and that
// of b_far 4 times, and the miller's notes, which has no block, are ignored.
//
// Requirement: FR-MER-055, FR-MER-056
func TestPlanNotes_SummedPerRegion(t *testing.T) {
	t.Parallel()
	calls := map[RegionID]int64{}
	w, rec, _ := newMerchWorld(t, func(c *Config) { c.Groups[len(c.Groups)-2].Households = 2 }, func(v *AgentView, p *TradePlan) {
		switch {
		case !v.HasMerchant:
			p.Notes = notesOf(100) // a role with no block: not counted
		case v.Region == regHome:
			calls[v.Region]++
			p.Notes = notesOf(calls[v.Region]) // 1, then 2 in the second broker
		default:
			p.Notes = notesOf(4)
		}
	})
	step(t, w, 1)
	got := plannedAt(rec, 0)
	if len(got) != 2 {
		t.Fatalf("MerchantsPlanned events at tick 0 = %+v, want one for each of the two regions", got)
	}
	want := func(region RegionID, merchants, k int64) MerchantsPlanned {
		n := notesOf(k)
		return MerchantsPlanned{Tick: 0, Region: region, Merchants: merchants,
			Considered: n.Considered, NoRumour: n.NoRumour, Stale: n.Stale, Locked: n.Locked, Perishable: n.Perishable,
			NoMargin: n.NoMargin, Unaffordable: n.Unaffordable, BelowShare: n.BelowShare, Planned: n.Planned,
			Dispatched: n.Dispatched, Offered: n.Offered, Taken: n.Taken, Rested: n.Rested, Waiting: n.Waiting, LotCapped: n.LotCapped}
	}
	if w0 := want(regHome, 2, 3); got[0] != w0 {
		t.Errorf("event for a_home = %+v, want %+v", got[0], w0)
	}
	if w1 := want(regFar, 1, 4); got[1] != w1 {
		t.Errorf("event for b_far = %+v, want %+v", got[1], w1)
	}
}

// TestMerchantsPlanned_Emitted verifies the event's emission (spec-merchants.md
// 15.1.1, spec-world.md 6 P3.3a): once per region in ascending RegionID per tick,
// after the last order of phase 3 and before RumoursServed (P3.6) and the markets
// of phase 4, also with the merchants module absent, and never when no role has a
// merchant block.
//
// Requirement: FR-MER-055, FR-MER-056, FR-MER-057
func TestMerchantsPlanned_Emitted(t *testing.T) {
	t.Parallel()
	script := func(v *AgentView, p *TradePlan) { p.Notes = PlanNotes{Considered: 1} }
	tests := []struct {
		name string
		mut  func(c *Config)
		want int
	}{
		{"module on", nil, 2},
		{"module absent", func(c *Config) {
			// A hired mode needs the module (contracts), so the block keeps the foot.
			c.Modules.Merchants = nil
			b := c.Roles[len(c.Roles)-1].Merchant
			b.Modes, b.Consigns = []string{"foot"}, false
		}, 2},
		{"no merchant block", func(c *Config) {
			c.Roles[len(c.Roles)-1].Merchant = nil
			c.Modules.Merchants = nil
		}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec, _ := newMerchWorld(t, tt.mut, script)
			step(t, w, 3)
			if got := len(of[MerchantsPlanned](rec)); got != 3*tt.want {
				t.Fatalf("MerchantsPlanned events in 3 ticks = %d, want %d", got, 3*tt.want)
			}
			if tt.want == 0 {
				return
			}
			for tick := range int64(3) {
				evs := plannedAt(rec, tick)
				if evs[0].Region != regHome || evs[1].Region != regFar {
					t.Errorf("tick %d: regions = %d, %d, want ascending %d, %d", tick, evs[0].Region, evs[1].Region, regHome, regFar)
				}
			}
			lastOrder, firstPlanned, firstServed, firstCleared := -1, -1, -1, -1
			for i, e := range rec.events {
				switch e := e.(type) {
				case OrderPlaced:
					if e.Tick == 1 {
						lastOrder = i
					}
				case MerchantsPlanned:
					if e.Tick == 1 && firstPlanned < 0 {
						firstPlanned = i
					}
				case RumoursServed:
					if e.Tick == 1 && firstServed < 0 {
						firstServed = i
					}
				case MarketCleared:
					if e.Tick == 1 && firstCleared < 0 {
						firstCleared = i
					}
				}
			}
			if firstPlanned < 0 || firstServed < 0 || firstPlanned > firstServed || firstServed > firstCleared || firstPlanned < lastOrder {
				t.Errorf("event positions at tick 1: last order %d, MerchantsPlanned %d, RumoursServed %d, first MarketCleared %d, want them in that order",
					lastOrder, firstPlanned, firstServed, firstCleared)
			}
		})
	}
}

// TestMerchantsPlanned_NoObserverNoAlloc verifies that with the no-op observer the
// module's per-agent calls allocate nothing once their scratch has grown: the
// filler of the merchant group, the plan check and the P3.3a position.
//
// Requirement: NFR-MER-004
func TestMerchantsPlanned_NoObserverNoAlloc(t *testing.T) {
	w := newMill(t, merchConfig(), WithStrategy("script", &merchScript{}))
	step(t, w, 2)
	a := &w.agents[brokerHome]
	var v AgentView
	var p TradePlan
	run := func() {
		w.fillMerchantView(&v, a)
		w.checkMerchantPlan(a, &p)
		w.p33aPlanned()
	}
	run()
	if got := testing.AllocsPerRun(100, run); got != 0 {
		t.Errorf("allocations per call of the filler, the check and P3.3a = %v, want 0", got)
	}
}

// TestMetrics_Derivable verifies that the metrics of FR-MER-059 can be computed
// from the events alone (spec-merchants.md 15.3): the eleven decision counts of
// MerchantsPlanned are fields named for the eleven constants of LaneDecision, so
// economysim.decision takes exactly those values, and the other four counts of the
// event (Considered, Dispatched, Offered, Taken) are not decisions.
//
// Requirement: FR-MER-059
func TestMetrics_Derivable(t *testing.T) {
	t.Parallel()
	constants := []LaneDecision{DecisionNoRumour, DecisionStale, DecisionLocked, DecisionPerishable, DecisionNoMargin,
		DecisionUnaffordable, DecisionBelowShare, DecisionPlanned, DecisionRested, DecisionWaiting, DecisionLotCapped}
	wantValues := []string{"no_rumour", "stale", "locked", "perishable", "no_margin", "unaffordable", "below_share", "planned", "rested", "waiting", "lot_capped"}
	for i, c := range constants {
		if string(c) != wantValues[i] {
			t.Errorf("decision constant %d = %q, want %q", i, c, wantValues[i])
		}
	}
	// Each constant has a field of the event named for it in CamelCase.
	typ := reflect.TypeOf(MerchantsPlanned{})
	camel := func(s string) string {
		out, up := []byte{}, true
		for i := range len(s) {
			switch {
			case s[i] == '_':
				up = true
			case up:
				out, up = append(out, s[i]-'a'+'A'), false
			default:
				out = append(out, s[i])
			}
		}
		return string(out)
	}
	for _, c := range constants {
		if f, ok := typ.FieldByName(camel(string(c))); !ok || f.Type.Kind() != reflect.Int64 {
			t.Errorf("MerchantsPlanned has no int64 field %q for decision %q", camel(string(c)), c)
		}
	}
	// 4 context fields, 4 counts that are not decisions and the 11 decisions.
	if got, want := typ.NumField(), 3+1+len(constants)+4-1; got != want {
		t.Errorf("MerchantsPlanned has %d fields, want %d (Tick, Region, Merchants and the 15 counts of PlanNotes)", got, want)
	}
}

// TestObserver_DoesNotChangeResults verifies that a run with an observer and a run
// without one give the same snapshot, with notes reported in every plan: the notes
// and the tally influence nothing (spec-merchants.md 15.1.2, observability 1.4).
//
// Requirement: FR-MER-057
func TestObserver_DoesNotChangeResults(t *testing.T) {
	t.Parallel()
	script := func(v *AgentView, p *TradePlan) { p.Notes = notesOf(v.Tick.Tick + 1) }
	run := func(observed bool) Snapshot {
		c := merchConfig()
		w := newMill(t, c, WithAudit(), WithStrategy("script", &merchScript{plan: script}))
		if observed {
			w = newMill(t, c, WithAudit(), WithObserver(&evLog{}), WithStrategy("script", &merchScript{plan: script}))
		}
		step(t, w, 6)
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		return snap
	}
	if with, without := run(true), run(false); !reflect.DeepEqual(with, without) {
		t.Errorf("the snapshot after 6 ticks differs with an observer attached: %+v against %+v", with, without)
	}
}

// TestTradePlan_ZeroForDefault verifies that the plan the world hands to Trade has
// no offer, no take and zero notes, also when the scratch plan held some on the
// previous call, and that the default strategy leaves all three empty.
//
// Requirement: FR-MER-005
func TestTradePlan_ZeroForDefault(t *testing.T) {
	t.Parallel()
	var sc viewScratch
	p := sc.tradePlan()
	p.Offers = append(p.Offers, DeliveryOffer{Quantity: 1})
	p.Takes = append(p.Takes, DeliveryTake{Quantity: 1})
	p.Notes = PlanNotes{Considered: 3}
	p = sc.tradePlan()
	if len(p.Offers) != 0 || len(p.Takes) != 0 || p.Notes != (PlanNotes{}) {
		t.Errorf("tradePlan() after a used plan = %+v, want no offer, no take and zero notes", *p)
	}
	got, _ := runTrade(exampleView(2_500), exampleWords())
	if len(got.Offers) != 0 || len(got.Takes) != 0 || got.Notes != (PlanNotes{}) {
		t.Errorf("DefaultStrategy.Trade left offers %v, takes %v, notes %+v, want none", got.Offers, got.Takes, got.Notes)
	}
}
