package economysim_test

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// idleStrategy does nothing: no job, no order. A household under it only eats its upkeep
// and watches its goods spoil, so the goods of two worlds that differ in nothing but the
// storage module can be compared tick by tick.
type idleStrategy struct{}

func (idleStrategy) Work(*economysim.AgentView, *rand.Rand) economysim.JobRequest {
	return economysim.JobRequest{NoJob: true}
}
func (idleStrategy) Trade(*economysim.AgentView, *rand.Rand, *economysim.TradePlan) {}
func (idleStrategy) Revise(o *economysim.OutcomeView, _ *rand.Rand) economysim.Belief {
	return o.Belief
}

// holdingsOf returns the holdings of every household of region 0 after a step.
func holdingsOf(t *testing.T, w *economysim.World) [][]economysim.HoldingState {
	t.Helper()
	ids, err := w.Agents(0)
	if err != nil {
		t.Fatalf("Agents(0) error = %v, want nil", err)
	}
	var out [][]economysim.HoldingState
	for _, id := range ids {
		st, err := w.Agent(id)
		if err != nil {
			t.Fatalf("Agent(%d) error = %v, want nil", id, err)
		}
		out = append(out, st.Holdings)
	}
	return out
}

// TestScenarioS92_GoodsSpoilOnTheirOwnClock is the ageing leg of STO-S2 (spec-storage.md
// 16, FR-STO-028): two worlds of pack one that differ only in the storage module, every
// household idle, so nothing is bought or made and the goods only spoil and are eaten. The
// holdings of every household of the region are equal after every one of 20 ticks, and the
// short-lived commodity (4 ticks) of the start stock is gone from some household by then:
// a limited account, with a store or without, ages its goods exactly as an account that
// has no room. Seeds 1 to 10, audit on.
//
// Requirement: FR-STO-028
//
// Guard: a store that slowed or stopped ageing would be a preservation, which the module
// does not have (spec-storage.md open question 2).
func TestScenarioS92_GoodsSpoilOnTheirOwnClock(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	on := loadValidPack(t, packOneStorage...)
	off := loadValidPack(t, packOne...)
	for _, c := range []*economysim.Config{&on, &off} {
		for i := range c.Roles {
			c.Roles[i].Strategy = "idle"
		}
	}
	stoScenario(t, "one", "S92 one+storage goods spoil on their own clock, 20 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		opts := []economysim.Option{economysim.WithAudit(), economysim.WithStrategy("idle", idleStrategy{})}
		a, b := newPackWorld(t, on, seed, opts...), newPackWorld(t, off, seed, opts...)
		same, spoiled, first := true, false, ""
		start := holdingsOf(t, a)
		for i := range 20 {
			for _, w := range []*economysim.World{a, b} {
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, i, err)
				}
			}
			ha, hb := holdingsOf(t, a), holdingsOf(t, b)
			if !reflect.DeepEqual(ha, hb) && same {
				same, first = false, fmt.Sprintf("tick %d", i)
			}
			spoiled = spoiled || !slices.EqualFunc(start, ha, func(x, y []economysim.HoldingState) bool { return len(x) == len(y) })
		}
		checkExact(t, rep, "S92", "one", seed, "the holdings with and without the module are equal every tick", same, "first differ at %s", first)
		checkExact(t, rep, "S92", "one", seed, "a commodity of the start stock spoiled away", spoiled, "no household lost a commodity in 20 ticks")
	})
}

// TestScenarioS92_BufferOwnerFillsAndReleases is the directed leg of STO-S2: a household
// of pack one with the storage module owns a store and is directed, sticky, to bid for
// staple for 100 ticks (a plenty season) and then to ask it for 100 (a lean one), seeds 1 to
// 10, audit on. Exact: the load of its account is within its room after every tick, the
// room it has is the base room plus the store. Recorded: the most staple it held (above the
// base room if the store was used) and what it held at the end.
//
// Requirement: FR-STO-004, FR-STO-005, FR-STO-028
func TestScenarioS92_BufferOwnerFillsAndReleases(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packOneStorage...)
	staple := economysim.CommodityID(slices.IndexFunc(cfg.Commodities, func(c economysim.Commodity) bool { return c.ID == "staple" }))
	stoScenario(t, "one", "S92 one+storage a buffer store owner fills and releases, 200 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		rec := &recorder{}
		w := newPackWorld(t, cfg, seed, economysim.WithAudit(), economysim.WithObserver(rec))
		owner := addBufferOwner(t, w, cfg)
		peak, within := runBufferOwner(t, w, owner, staple)
		st, _ := w.Agent(owner)
		var noRoom int64
		for _, e := range rec.events {
			if e, ok := e.(economysim.OrderRejected); ok && e.Agent == owner && e.Reason == economysim.ReasonNoRoom {
				noRoom++
			}
		}
		rep.num(seed, "orders of the owner rejected for room", noRoom)
		rep.num(seed, "most staple held", peak)
		rep.num(seed, "room of the owner (mass)", st.Room.Room.Mass)
		checkExact(t, rep, "S92", "one", seed, "the load of the owner is within its room at every tick", within, "load above room")
	})
}

// addBufferOwner adds the external household of the directed leg of S92, a cook of region
// 0 with money and one store of the first class, and returns its ID.
func addBufferOwner(t *testing.T, w *economysim.World, cfg economysim.Config) economysim.AgentID {
	t.Helper()
	cook := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == "cook" })
	owner, err := w.AddAgent(economysim.NewAgent{Region: 0, Role: economysim.RoleID(cook), Strategy: "directed", Money: 10_000_000})
	if err != nil {
		t.Fatalf("AddAgent() error = %v, want nil", err)
	}
	if err := w.AddStore(owner, 0); err != nil {
		t.Fatalf("AddStore() error = %v, want nil", err)
	}
	return owner
}

// runBufferOwner directs the owner for 200 ticks, to bid for 400 units of staple a tick for
// the first 100 and to ask 400 a tick for the rest, and returns the most staple it held and
// whether its load was within its room after every tick.
func runBufferOwner(t *testing.T, w *economysim.World, owner economysim.AgentID, staple economysim.CommodityID) (peak int64, within bool) {
	t.Helper()
	within = true
	for i := range 200 {
		side, limit := economysim.Bid, economysim.Price(2_000)
		if i >= 100 {
			side, limit = economysim.Ask, 1
		}
		d := economysim.Directive{Intents: []economysim.Intent{{Commodity: staple, Side: side, Quantity: 400, Limit: limit}}, Sticky: true}
		if err := w.Direct(owner, d); err != nil {
			t.Fatalf("Direct() at tick %d error = %v, want nil", i, err)
		}
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", i, err)
		}
		st, err := w.Agent(owner)
		if err != nil {
			t.Fatalf("Agent() error = %v, want nil", err)
		}
		within = within && st.Room.Load.Mass <= st.Room.Room.Mass && st.Room.Load.Bulk <= st.Room.Room.Bulk
		for _, h := range st.Holdings {
			if h.Commodity == staple {
				peak = max(peak, int64(h.Quantity))
			}
		}
	}
	return peak, within
}
