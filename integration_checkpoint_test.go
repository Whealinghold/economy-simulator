package economysim_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The checkpoint of a world with calls in its history (spec-integration.md 13, tag
// 22 of spec-checkpoint.md 4.2): an external agent, its mailbox, the slots and the
// rumours it was given must come back, so that a restored world resumes
// bit-identically.

// roleByID is the index of the role with the given ID in cfg.
func roleByID(t testing.TB, cfg economysim.Config, id string) economysim.RoleID {
	t.Helper()
	for i := range cfg.Roles {
		if cfg.Roles[i].ID == id {
			return economysim.RoleID(i)
		}
	}
	t.Fatalf("the configuration has no role %q", id)
	return 0
}

// playerCalls is the history of calls the fork tests share: a player with money and
// stock under a sticky directive, a second agent under a directive that is not
// sticky, a yield shock and a closed road. The pack has two regions and transport.
func playerCalls(t testing.TB, cfg economysim.Config, w *economysim.World) {
	t.Helper()
	grower := roleByID(t, cfg, "grower")
	player, err := w.AddAgent(economysim.NewAgent{
		Region: slotNorth, Role: grower, Money: 50_000,
		Stock: []economysim.StockItem{{Commodity: slotFuel, Quantity: 20}, {Commodity: slotStaple, Quantity: 10}},
	})
	if err != nil {
		t.Fatalf("AddAgent(player) = %v, want nil", err)
	}
	bot, err := w.AddAgent(economysim.NewAgent{Region: slotOther, Role: grower, Money: 1_000, Mortal: true})
	if err != nil {
		t.Fatalf("AddAgent(bot) = %v, want nil", err)
	}
	for _, c := range []struct {
		id economysim.AgentID
		d  economysim.Directive
	}{
		{player, economysim.Directive{Sticky: true, Hold: []economysim.CommodityID{slotFuel}}},
		{bot, economysim.Directive{Intents: []economysim.Intent{{Commodity: slotStaple, Side: economysim.Bid, Quantity: 3, Limit: 400}}}},
	} {
		if err := w.Direct(c.id, c.d); err != nil {
			t.Fatalf("Direct(%d) = %v, want nil", c.id, err)
		}
	}
	if err := w.SetYieldShocks([]economysim.YieldShock{{Region: slotNorth, Recipe: slotGrowStaple, BP: 6_000}}); err != nil {
		t.Fatalf("SetYieldShocks() = %v, want nil", err)
	}
	if err := w.SetEdgeStatuses([]economysim.EdgeStatus{{From: slotNorth, To: slotOther, Closed: true, FactorBP: 10_000}}); err != nil {
		t.Fatalf("SetEdgeStatuses() = %v, want nil", err)
	}
}

// TestFork_ExternalAgent verifies the fork-at-N of a world whose history holds
// calls: an external player and a mortal bot with money and stock, a sticky and a
// non-sticky directive, a yield shock and a closed edge. At N 0, 1 and 5 (inside a
// job of the pack) the original and two restores give the same events, snapshot,
// audit and checkpoint bytes for 30 ticks.
//
// Requirement: FR-INT-070, FR-INT-049, FR-CHK-050
func TestFork_ExternalAgent(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packTwoTransport...)
	for _, n := range []int{0, 1, 5} {
		t.Run(fmt.Sprintf("N %d", n), func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "external agent", Cfg: cfg, Seed: 7, N: n, M: forkM,
				Before: func(w *economysim.World) { playerCalls(t, cfg, w) },
				Inspect: func(t *testing.T, restored, since []economysim.Event) {
					if !hasEvent[economysim.AgentSpawned](since) && !hasEvent[economysim.OrderPlaced](since) {
						t.Error("the continuation holds neither a spawn nor an order, so it did not run the agents")
					}
				},
			})
		})
	}
}

// hasEvent reports whether events holds one of type E.
func hasEvent[E economysim.Event](events []economysim.Event) bool {
	return slices.ContainsFunc(events, func(e economysim.Event) bool { _, ok := e.(E); return ok })
}

// consumerRemoves is a spawner plan that removes the mortal external agents with the
// cause consumer, one tick after they are first seen, and nothing else.
func consumerRemoves(v *economysim.RegionView) []economysim.Decision {
	var d []economysim.Decision
	for _, a := range v.Agents {
		if a.External && a.Mortal && a.RoleSince < v.Tick {
			d = append(d, economysim.Decision{Kind: economysim.DecideRemove, Agent: a.Agent, Cause: economysim.CauseConsumer})
		}
	}
	return d
}

// TestScenarioS82_PlayerAndRestore is INT-S4: the fork test above on every seed of
// the race-aware helper, with a spawner of the consumer that removes the mortal
// player after it has lived a tick. The restored worlds remove it at the same tick
// and with the same events.
//
// Requirement: FR-INT-070, FR-INT-049, FR-CHK-050
func TestScenarioS82_PlayerAndRestore(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := withSpawner(loadValidPack(t, packTwoTransport...))
	for seed := range scenarioSeedCount() {
		t.Run(fmt.Sprintf("seed %d", seed+1), func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "player and restore", Cfg: cfg, Opts: scripted(consumerRemoves), Seed: uint64(seed + 1), N: 3, M: forkM,
				Before: func(w *economysim.World) { playerCalls(t, cfg, w) },
				Inspect: func(t *testing.T, restored, since []economysim.Event) {
					removed := false
					for _, e := range since {
						if b, ok := e.(economysim.AgentBankrupted); ok && b.Cause == economysim.CauseConsumer {
							removed = true
						}
					}
					if !removed {
						t.Errorf("seed %d: the continuation holds no removal with the cause consumer", seed+1)
					}
				},
			})
		})
	}
}

// informedPlayer builds the information pack of the variant two with an external
// agent of an informed role, steps it ticks times and returns the world and the
// agent. The rumours need a few ticks of price history to be offered.
func informedPlayer(t *testing.T, ticks int) (*economysim.World, economysim.Config, economysim.AgentID) {
	t.Helper()
	cfg := loadValidPack(t, packInformation("two")...)
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(7), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	id, err := w.AddAgent(economysim.NewAgent{Region: slotNorth, Role: roleByID(t, cfg, "grower"), Money: 50_000})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	return w, cfg, id
}

// TestFork_RumoursKept verifies that the rumours of the last view of an external
// agent of an informed role, and the tick of that view, survive a checkpoint: the
// restored world answers Agent as the original did, and runs on to the same bytes.
//
// Requirement: FR-INT-049, FR-CHK-052
func TestFork_RumoursKept(t *testing.T) {
	t.Parallel()
	w, cfg, id := informedPlayer(t, 14)
	before, err := w.Agent(id)
	if err != nil {
		t.Fatalf("Agent() = %v, want nil", err)
	}
	if len(before.Rumours) == 0 || before.RumoursTick != w.Tick()-1 {
		t.Fatalf("Agent().Rumours = %d, tick %d, want rumours from the view of tick %d", len(before.Rumours), before.RumoursTick, w.Tick()-1)
	}
	economysim.FillWorkView(w, id) // a Work view leaves the stored rumours and their tick alone
	if kept, _ := w.Agent(id); !reflect.DeepEqual(kept.Rumours, before.Rumours) || kept.RumoursTick != before.RumoursTick {
		t.Errorf("a Work view changed the rumours kept: tick %d, %d rumours, want tick %d, %d", kept.RumoursTick, len(kept.Rumours), before.RumoursTick, len(before.Rumours))
	}
	r, err := economysim.RestoreWorld(cfg, mustCheckpoint(t, w), economysim.WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	after, err := r.Agent(id)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("restored Agent() = %+v, %v, want %+v", after, err, before)
	}
	for _, x := range []*economysim.World{w, r} {
		if err := x.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	if !bytes.Equal(slotBytes(t, w), slotBytes(t, r)) {
		t.Error("the restored world's checkpoint differs from the original's one step on")
	}
	if again, _ := r.Agent(id); again.RumoursTick != r.Tick()-1 {
		t.Errorf("after a step the view tick is %d, want %d", again.RumoursTick, r.Tick()-1)
	}
}

// tag22Body builds a body of tag 22: the rows are agent, view tick and the rumours
// as (region, commodity, price, mean, lag, source age).
func tag22Body(rows ...tag22Row) []byte {
	b := binary.AppendUvarint(nil, uint64(len(rows)))
	for _, r := range rows {
		b = binary.AppendUvarint(b, r.agent)
		b = binary.AppendUvarint(b, uint64(r.tick)<<1)
		b = binary.AppendUvarint(b, uint64(len(r.rumours)))
		for _, ru := range r.rumours {
			for _, v := range ru {
				b = binary.AppendUvarint(b, v)
			}
		}
	}
	return b
}

// tag22Row is a row of tag 22.
type tag22Row struct {
	agent   uint64
	tick    int64
	rumours [][6]uint64
}

// TestRestore_Tag22Corrupt verifies that tag 22 refuses what a world cannot have
// written: an agent that is not an external agent of an informed role, rows out of
// order, a view tick the run has not made, a rumour that cannot be one, and a count
// above the bytes left. The good body restores, so each case fails by its fault.
//
// Requirement: FR-INT-049, FR-CHK-022
func TestRestore_Tag22Corrupt(t *testing.T) {
	t.Parallel()
	w, cfg, id := informedPlayer(t, 14)
	raw := mustMarshal(t, mustCheckpoint(t, w))
	tick := w.Tick() - 1
	ru := func(region, commodity, price, lag uint64) [6]uint64 {
		return [6]uint64{region, commodity, price, 0, lag, 0}
	}
	good := tag22Row{agent: uint64(id), tick: tick, rumours: [][6]uint64{ru(1, 0, 300, 3), ru(1, 1, 300, 3)}}
	other := economysim.AgentID(len(cfg.Regions))
	with := func(f func(r *tag22Row)) []byte {
		r := good
		r.rumours = slices.Clone(good.rumours)
		f(&r)
		return tag22Body(r)
	}
	tests := []struct {
		name string
		body []byte
	}{
		{"an agent that is not external", tag22Body(tag22Row{agent: uint64(other), tick: tick})},
		{"an agent never created", tag22Body(tag22Row{agent: 9_999, tick: tick})},
		{"rows out of order", tag22Body(good, tag22Row{agent: uint64(id) - 1, tick: tick})},
		{"a row repeated", tag22Body(good, good)},
		{"a tick the run has not made", with(func(r *tag22Row) { r.tick = w.Tick() })},
		{"a price of 0", with(func(r *tag22Row) { r.rumours[0][2] = 0 })},
		{"a lag below 2", with(func(r *tag22Row) { r.rumours[0][4] = 1 })},
		{"a region out of range", with(func(r *tag22Row) { r.rumours[0][0] = 99 })},
		{"rumours out of order", with(func(r *tag22Row) { r.rumours[0], r.rumours[1] = r.rumours[1], r.rumours[0] })},
		{"a rumour repeated", with(func(r *tag22Row) { r.rumours[1] = r.rumours[0] })},
		{"a count above the bytes left", append([]byte{50}, tag22Body(good)[1:]...)},
		{"bytes after the last row", append(tag22Body(good), 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, func(s []section) []section { return editSection(s, 22, func([]byte) []byte { return tt.body }) })
			_, err := economysim.RestoreWorld(cfg, c)
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) || !strings.Contains(err.Error(), "section 22") {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt naming section 22", tt.name, err)
			}
		})
	}
	c := reframe(t, raw, func(s []section) []section { return editSection(s, 22, func([]byte) []byte { return tag22Body(good) }) })
	if _, err := economysim.RestoreWorld(cfg, c); err != nil {
		t.Errorf("RestoreWorld(the good body) error = %v, want nil", err)
	}
}

// TestFork_BeforeFirstStep verifies that a call made in the construction phase is
// saved and restored: the player's money is in the construction bucket as initial,
// and the restored world runs to the same bytes.
//
// Requirement: FR-INT-011, FR-CHK-050
func TestFork_BeforeFirstStep(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packTwoTransport...)
	forkAtN(t, forkCase{
		Name: "before the first step", Cfg: cfg, Seed: 7, N: 0, M: 12,
		Before: func(w *economysim.World) {
			playerCalls(t, cfg, w)
			snap, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() = %v, want nil", err)
			}
			if !slices.ContainsFunc(snap.Construction, func(f economysim.FlowTotal) bool {
				return f.Reason == "initial" && f.Commodity == economysim.NoCommodity && f.Total >= 51_000
			}) {
				t.Errorf("Snapshot().Construction = %+v, want the money of the calls by initial", snap.Construction)
			}
		},
	})
}

// callScript is the call sequence of INT-S3: at each tick it makes the calls of that
// tick, between steps, on the pack of two regions with transport. It returns the
// world, which the caller reads.
func callScript(t *testing.T, cfg economysim.Config, seed uint64, swap bool, opts ...economysim.Option) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(seed), economysim.WithAudit()}, opts...)...)
	if err != nil {
		t.Fatalf("seed %d: NewWorld() = %v, want nil", seed, err)
	}
	grower := roleByID(t, cfg, "grower")
	var id economysim.AgentID
	for tick := range 24 {
		switch tick {
		case 3:
			if id, err = w.AddAgent(economysim.NewAgent{Region: slotNorth, Role: grower, Money: 20_000, Stock: []economysim.StockItem{{Commodity: slotFuel, Quantity: 5}}}); err != nil {
				t.Fatalf("seed %d: AddAgent() = %v, want nil", seed, err)
			}
		case 5:
			err = w.Direct(id, economysim.Directive{Sticky: true, Hold: []economysim.CommodityID{slotFuel}})
		case 8:
			// Two keys of one call and the same keys in two calls, in either order, are one state.
			a, b := economysim.YieldShock{Region: slotNorth, Recipe: slotGrowStaple, BP: 7_000}, economysim.YieldShock{Region: slotOther, Recipe: slotGrowStaple, BP: 4_000}
			if swap {
				a, b = b, a
			}
			if err = w.SetYieldShocks([]economysim.YieldShock{a}); err == nil {
				err = w.SetYieldShocks([]economysim.YieldShock{b})
			}
		case 12:
			err = w.SetEdgeStatuses([]economysim.EdgeStatus{{From: slotNorth, To: slotOther, Closed: true, FactorBP: 10_000}})
		case 16:
			err = w.Relocate(id, slotOther)
		case 20:
			err = w.RemoveAgent(id)
		}
		if err != nil {
			t.Fatalf("seed %d: the calls of tick %d = %v, want nil", seed, tick, err)
		}
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() at tick %d = %v, want nil", seed, tick, err)
		}
	}
	return w
}

// TestScenarioS81_CallSequence is INT-S3: one call script run in two worlds gives the
// same state and the same events (checkpoint bytes and events are equal), a script
// whose calls on different keys come in the other order ends in the same state, the
// money of AddAgent shows in the BoundaryFlow of its tick as initial, and the audit
// passes after every step (the worlds run with WithAudit).
//
// Requirement: FR-INT-008, FR-INT-009, FR-INT-050
func TestScenarioS81_CallSequence(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packTwoTransport...)
	for seed := range scenarioSeedCount() {
		t.Run(fmt.Sprintf("seed %d", seed+1), func(t *testing.T) {
			t.Parallel()
			s := uint64(seed + 1)
			var r1, r2, r3 recorder
			w1 := callScript(t, cfg, s, false, economysim.WithObserver(&r1))
			w2 := callScript(t, cfg, s, false, economysim.WithObserver(&r2))
			w3 := callScript(t, cfg, s, true, economysim.WithObserver(&r3))
			if !reflect.DeepEqual(r1.events, r2.events) || !bytes.Equal(slotBytes(t, w1), slotBytes(t, w2)) {
				t.Errorf("seed %d: two runs of one call script differ in their events or their state", s)
			}
			if !bytes.Equal(slotBytes(t, w1), slotBytes(t, w3)) {
				t.Errorf("seed %d: the calls on different keys in the other order end in another state", s)
			}
			initial := slices.ContainsFunc(r1.events, func(e economysim.Event) bool {
				f, ok := e.(economysim.BoundaryFlow)
				return ok && f.Reason == economysim.ReasonInitial && f.Kind == economysim.KindFaucet && f.Commodity == economysim.NoCommodity && f.Total == 20_000
			})
			if !initial {
				t.Errorf("seed %d: no BoundaryFlow of 20 000 by initial after AddAgent", s)
			}
		})
	}
}

// TestScenarioS79_YieldShockCalls verifies, by exact checks only, the yield shock
// through the public call: three calls (3 000 at tick 0, 3 000 again, then back to
// 10 000 after 60 ticks) emit one YieldShocked each time the factor changes, Slots
// lists the shock while it holds and is empty after it is cleared. The bands on the
// recovery are plan 19's.
//
// Requirement: FR-INT-040
func TestScenarioS79_YieldShockCalls(t *testing.T) {
	t.Parallel()
	var rec recorder
	cfg := loadValidPack(t, packOne...)
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(3), economysim.WithObserver(&rec))
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	shock := func(bp int64) {
		t.Helper()
		if err := w.SetYieldShocks([]economysim.YieldShock{{Region: slotNorth, Recipe: slotGrowStaple, BP: bp}}); err != nil {
			t.Fatalf("SetYieldShocks(%d) = %v, want nil", bp, err)
		}
	}
	run := func(n int) {
		t.Helper()
		for range n {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
		}
	}
	shock(3_000)
	run(60)
	shock(3_000) // no change, no event
	if s, _ := w.Slots(); len(s.YieldShocks) != 1 || s.YieldShocks[0].BP != 3_000 {
		t.Errorf("Slots().YieldShocks = %+v, want the one shock of 3 000", s.YieldShocks)
	}
	run(60)
	shock(10_000)
	if s, _ := w.Slots(); len(s.YieldShocks) != 0 {
		t.Errorf("Slots().YieldShocks = %+v after the factor returned to 10 000, want none", s.YieldShocks)
	}
	run(5)
	var got []economysim.YieldShocked
	for _, e := range rec.events {
		if y, ok := e.(economysim.YieldShocked); ok {
			got = append(got, y)
		}
	}
	want := []economysim.YieldShocked{
		{Tick: 0, Region: slotNorth, Recipe: slotGrowStaple, BeforeBP: 10_000, AfterBP: 3_000},
		{Tick: 120, Region: slotNorth, Recipe: slotGrowStaple, BeforeBP: 3_000, AfterBP: 10_000},
	}
	if !slices.Equal(got, want) {
		t.Errorf("the YieldShocked events = %+v, want %+v", got, want)
	}
}

// TestObservers_DoNotChangeCalls verifies that a recording observer and a capturing
// logger change nothing about a world that is called: the call script ends in the
// state and the snapshot of a world that has neither.
//
// Requirement: FR-INT-050, FR-INT-051, FR-INT-052
func TestObservers_DoNotChangeCalls(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, packTwoTransport...)
	var rec recorder
	watched := callScript(t, cfg, 5, false, economysim.WithObserver(&rec), economysim.WithLogger(slog.New(newCaptureHandler())))
	plain := callScript(t, cfg, 5, false)
	if !bytes.Equal(slotBytes(t, watched), slotBytes(t, plain)) {
		t.Error("a world with an observer ends in another state than one without")
	}
	ws, _ := watched.Snapshot()
	ps, _ := plain.Snapshot()
	if !reflect.DeepEqual(ws, ps) {
		t.Error("a world with an observer has another snapshot than one without")
	}
}

// TestRestore_ExternalStrategyName verifies that the strategy name of an external
// agent is saved with it and resolved at the restore: a registered name comes back
// as the agent's strategy, and a restore that lacks the registration is refused with
// ErrCheckpointExtension and ErrUnknownStrategy. A record with the flags of a
// household that name another strategy than its role's is corrupt.
//
// Requirement: FR-CHK-022, FR-INT-049
func TestRestore_ExternalStrategyName(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, packOne...)
	reg := economysim.WithStrategy("mine", economysim.DefaultStrategy{})
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(2), reg)
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	id, err := w.AddAgent(economysim.NewAgent{Region: slotNorth, Role: 0, Strategy: "mine", Money: 100, Mortal: true})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	ck := mustCheckpoint(t, w)
	r, err := economysim.RestoreWorld(cfg, ck, reg)
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	if st, err := r.Agent(id); err != nil || st.Strategy != "mine" || !st.External || !st.Mortal {
		t.Errorf("restored Agent(%d) = %+v, %v, want the external mortal agent with strategy mine", id, st, err)
	}
	if _, err := economysim.RestoreWorld(cfg, ck); !errors.Is(err, economysim.ErrCheckpointExtension) || !errors.Is(err, economysim.ErrUnknownStrategy) {
		t.Errorf("RestoreWorld() without the registration = %v, want ErrCheckpointExtension and ErrUnknownStrategy", err)
	}
}
