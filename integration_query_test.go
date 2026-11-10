package economysim_test

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The queries of spec-integration.md 6 are tested through the public API on the
// packs of the modules whose rows they carry. What is tested here is the query: the
// guards, that it changes nothing, and that it agrees with the snapshot and the
// ledger it reads; the mechanism behind each row is its owner's.

// queryPacks are the packs the queries run on: the reference chain, the external
// sector (an outside node), transport, resources with population, demand and
// migration.
func queryPacks() map[string][]string {
	return map[string][]string{
		"one":       packOne,
		"external":  packExternal(),
		"transport": packTwoTransport,
		"resources": slotsEndow,
		"demand":    packOneDemand,
		"migration": append(packInformation("two"), "overlay/migration.toml"),
	}
}

// queryWorld builds the world of a pack and steps it n times.
func queryWorld(t testing.TB, layers []string, n int, opts ...economysim.Option) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(loadValidPack(t, layers...), append([]economysim.Option{economysim.WithSeed(7), economysim.WithAudit()}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld(%q) = %v, want nil", layers, err)
	}
	for i := range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step %d of %q = %v, want nil", i, layers, err)
		}
	}
	return w
}

// callQueries calls every query of INT 6 once on w, for the tests that call them
// all.
func callQueries(t testing.TB, w *economysim.World) {
	t.Helper()
	_, _ = w.Slots()
	_ = w.Warnings()
	_ = w.Calendar()
	_, _ = w.Market(0, 0)
	_, _ = w.Region(0)
	ids, err := w.Agents(0)
	if err != nil {
		t.Fatalf("Agents(0) = %v, want nil", err)
	}
	for _, id := range ids[:min(len(ids), 3)] {
		if _, err := w.Agent(id); err != nil {
			t.Fatalf("Agent(%d) = %v, want nil", id, err)
		}
	}
	_, _, _ = w.Tick(), w.Err(), w.ConfigHash()
}

// TestQueries_NoDrawNoStateNoEvent verifies that no query draws, changes state or
// emits an event: the checkpoint bytes are equal before and after every query and
// the world, run on, ends in the state of a twin that was never queried.
//
// Requirement: FR-INT-060, FR-WLD-010
func TestQueries_NoDrawNoStateNoEvent(t *testing.T) {
	t.Parallel()
	for name, layers := range queryPacks() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var events int
			count := economysim.ObserverFunc(func(economysim.Event) { events++ })
			w := queryWorld(t, layers, 12, economysim.WithObserver(count))
			twin := queryWorld(t, layers, 12)
			before, seen := slotBytes(t, w), events
			callQueries(t, w)
			if !bytes.Equal(before, slotBytes(t, w)) {
				t.Error("the checkpoint changed after the queries")
			}
			if events != seen {
				t.Errorf("the queries emitted %d events, want 0", events-seen)
			}
			for _, x := range []*economysim.World{w, twin} {
				if err := x.Step(); err != nil {
					t.Fatalf("Step() = %v, want nil", err)
				}
			}
			if !bytes.Equal(slotBytes(t, w), slotBytes(t, twin)) {
				t.Error("a step after the queries differs from the step of a twin that was not queried")
			}
		})
	}
}

// TestQueries_Guards verifies the guards of spec-integration.md 6.1.2: the queries
// with an error result return ErrInStep from inside a step, the readers with none
// return their value there, and every query is legal in the construction phase.
//
// Requirement: FR-INT-060
func TestQueries_Guards(t *testing.T) {
	t.Parallel()
	errs := map[string]error{}
	var tick int64 = -1
	var w *economysim.World
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		if _, ok := e.(economysim.TickStarted); !ok || len(errs) > 0 {
			return
		}
		_, errs["Agent"] = w.Agent(economysim.AgentID(w.Tick()))
		_, errs["Agents"] = w.Agents(0)
		_, errs["Market"] = w.Market(0, 0)
		_, errs["Region"] = w.Region(0)
		tick = w.Tick()
		_, _, _ = w.Err(), w.ConfigHash(), w.Calendar()
		_ = w.Warnings()
	})
	w, err := economysim.NewWorld(loadValidPack(t, packOne...), economysim.WithSeed(7), economysim.WithObserver(obs))
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	callQueries(t, w) // legal before the first step
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	if len(errs) != 4 {
		t.Fatalf("the observer made %d query calls, want 4", len(errs))
	}
	for name, err := range errs {
		if !errors.Is(err, economysim.ErrInStep) {
			t.Errorf("%s from inside a step = %v, want ErrInStep", name, err)
		}
	}
	if tick != 0 {
		t.Errorf("Tick() inside the first step = %d, want 0", tick)
	}
}

// TestAgent_State verifies AgentState against the ledger, the beliefs and the job
// of a stepped world: an external agent's money, stock, mailbox and strategy are
// what the call set, the belief bounds agree with the snapshot's counts of beliefs
// at a bound, the jobs agree with the snapshot's job rows, the result is a deep
// copy, and an unknown, removed or pseudo-agent ID is refused.
//
// Requirement: FR-INT-033, FR-INT-030
func TestAgent_State(t *testing.T) {
	t.Parallel()
	w := queryWorld(t, packOne, 40)
	id, err := w.AddAgent(economysim.NewAgent{
		Region: slotNorth, Role: 0, Money: 5_000, Stock: []economysim.StockItem{{Commodity: slotStaple, Quantity: 7}},
	})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	dir := economysim.Directive{Sticky: true, Hold: []economysim.CommodityID{slotFuel}}
	if err := w.Direct(id, dir); err != nil {
		t.Fatalf("Direct() = %v, want nil", err)
	}
	st, err := w.Agent(id)
	if err != nil {
		t.Fatalf("Agent(%d) = %v, want nil", id, err)
	}
	if st.ID != id || st.Region != slotNorth || !st.External || st.Mortal || st.Money != 5_000 || st.Strategy != "directed" || st.RoleSince != w.Tick() {
		t.Errorf("Agent(%d) identity = %+v, want the agent as added at tick %d", id, st, w.Tick())
	}
	if want := []economysim.HoldingState{{Commodity: slotStaple, Quantity: 7}}; !reflect.DeepEqual(st.Holdings, want) {
		t.Errorf("Agent(%d).Holdings = %v, want %v (its stock, none expiring)", id, st.Holdings, want)
	}
	if !st.HasDirective || !slices.Equal(st.Directive.Hold, dir.Hold) || !st.Directive.Sticky || st.RumoursTick != -1 || len(st.Rumours) != 0 {
		t.Errorf("Agent(%d) mailbox = %v %+v, rumours tick %d, want the directive set and no rumours", id, st.HasDirective, st.Directive, st.RumoursTick)
	}
	st.Directive.Hold[0], st.Holdings[0].Quantity = slotStaple, 99
	if again, _ := w.Agent(id); again.Directive.Hold[0] != slotFuel || again.Holdings[0].Quantity != 7 {
		t.Errorf("changing the result changed the next result: %+v", again)
	}
	checkBeliefsAndJobs(t, w)
	if err := w.RemoveAgent(id); err != nil {
		t.Fatalf("RemoveAgent() = %v, want nil", err)
	}
	for _, tt := range []struct {
		name string
		id   economysim.AgentID
		want error
	}{
		{"removed", id, economysim.ErrAgentRemoved},
		{"never issued", 9_999, economysim.ErrUnknownAgent},
		{"negative", -1, economysim.ErrUnknownAgent},
		{"the estate pseudo-agent", 0, economysim.ErrUnknownAgent},
	} {
		if _, err := w.Agent(tt.id); !errors.Is(err, tt.want) {
			t.Errorf("Agent(%d) (%s) = %v, want %v", tt.id, tt.name, err, tt.want)
		}
	}
}

// checkBeliefsAndJobs compares the households' AgentState with the snapshot: the
// beliefs on a bound of their range count to the snapshot's AtFloor and AtCeiling,
// the beliefs lie in their range, and the jobs sum to the job rows.
func checkBeliefsAndJobs(t *testing.T, w *economysim.World) {
	t.Helper()
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	type key struct{ c economysim.CommodityID }
	floor, ceil := map[key]int64{}, map[key]int64{}
	execs := map[economysim.RecipeID]int64{}
	ids, _ := w.Agents(slotNorth)
	for _, id := range ids {
		st, err := w.Agent(id)
		if err != nil {
			t.Fatalf("Agent(%d) = %v, want nil", id, err)
		}
		if st.HasJob {
			execs[st.Recipe] += st.Executions
			if len(st.Pending) == 0 {
				t.Errorf("Agent(%d) has a job and no pending output", id)
			}
		}
		for _, b := range st.Beliefs {
			if b.Centre < b.Floor || b.Centre > b.Ceiling {
				t.Errorf("Agent(%d) belief %+v lies outside its range", id, b)
			}
			floor[key{b.Commodity}] += btoi(b.Centre == b.Floor)
			ceil[key{b.Commodity}] += btoi(b.Centre == b.Ceiling)
		}
	}
	for _, m := range snap.Markets {
		if m.State.Region == slotNorth && (floor[key{m.State.Commodity}] != m.AtFloor || ceil[key{m.State.Commodity}] != m.AtCeiling) {
			t.Errorf("commodity %d: beliefs at the floor %d and ceiling %d, snapshot says %d and %d",
				m.State.Commodity, floor[key{m.State.Commodity}], ceil[key{m.State.Commodity}], m.AtFloor, m.AtCeiling)
		}
	}
	for _, j := range snap.Jobs {
		if j.Region == slotNorth && execs[j.Recipe] != j.Executions {
			t.Errorf("recipe %d: executions %d from the agents, snapshot says %d", j.Recipe, execs[j.Recipe], j.Executions)
		}
	}
}

// btoi is 1 for true.
func btoi(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// TestAgents_Region verifies that Agents lists the live agents of a region in
// ascending ID, no pseudo-agent and no removed agent, and that the count agrees with
// the snapshot.
//
// Requirement: FR-INT-061
func TestAgents_Region(t *testing.T) {
	t.Parallel()
	w := queryWorld(t, packOne, 10)
	added, err := w.AddAgent(economysim.NewAgent{Region: slotNorth, Role: 0})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	ids, err := w.Agents(slotNorth)
	if err != nil {
		t.Fatalf("Agents() = %v, want nil", err)
	}
	snap, _ := w.Snapshot()
	if int64(len(ids)) != snap.Regions[slotNorth].Agents || !slices.IsSorted(ids) || !slices.Contains(ids, added) {
		t.Errorf("Agents(north) = %v, want %d ascending IDs with %d", ids, snap.Regions[slotNorth].Agents, added)
	}
	if err := w.RemoveAgent(added); err != nil {
		t.Fatalf("RemoveAgent() = %v, want nil", err)
	}
	if after, _ := w.Agents(slotNorth); slices.Contains(after, added) || len(after) != len(ids)-1 {
		t.Errorf("Agents(north) after the removal = %v, want without %d", after, added)
	}
	if _, err := w.Agents(99); !errors.Is(err, economysim.ErrUnknownRegion) {
		t.Errorf("Agents(99) = %v, want ErrUnknownRegion", err)
	}
	ext := queryWorld(t, packExternal(), 3)
	if got, err := ext.Agents(slotOther); err != nil || len(got) != 0 {
		t.Errorf("Agents(outside node) = %v, %v, want an empty list", got, err)
	}
}

// TestMarket_Fields verifies that Market equals the snapshot's MarketState for
// every market of every pack, the seed price with age 0 before the first step, and
// the errors.
//
// Requirement: FR-INT-053, FR-OBS-045
func TestMarket_Fields(t *testing.T) {
	t.Parallel()
	for name, layers := range queryPacks() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := queryWorld(t, layers, 20)
			snap, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() = %v, want nil", err)
			}
			for _, m := range snap.Markets {
				got, err := w.Market(m.State.Region, m.State.Commodity)
				if err != nil || got != m.State {
					t.Fatalf("Market(%d, %d) = %+v, %v, want the snapshot's %+v", m.State.Region, m.State.Commodity, got, err, m.State)
				}
			}
		})
	}
	fresh := queryWorld(t, packOne, 0)
	snap, _ := fresh.Snapshot()
	if m, err := fresh.Market(slotNorth, slotFuel); err != nil || m.Age != 0 || m.Price != snap.Markets[slotFuel].State.Price || m.Price <= 0 {
		t.Errorf("Market before the first step = %+v, %v, want the seed price with age 0", m, err)
	}
	ext := queryWorld(t, packExternal(), 1)
	for _, tt := range []struct {
		name string
		w    *economysim.World
		g    economysim.RegionID
		c    economysim.CommodityID
		want error
	}{
		{"unknown region", fresh, 99, 0, economysim.ErrUnknownRegion},
		{"negative region", fresh, -1, 0, economysim.ErrUnknownRegion},
		{"unknown commodity", fresh, 0, 99, economysim.ErrUnknownCommodity},
		{"outside node", ext, slotOther, 0, economysim.ErrOutsideNode},
	} {
		if _, err := tt.w.Market(tt.g, tt.c); !errors.Is(err, tt.want) {
			t.Errorf("Market(%d, %d) (%s) = %v, want %v", tt.g, tt.c, tt.name, err, tt.want)
		}
	}
}

// nz is nil for an empty slice, so that an empty row set and an absent one compare
// equal.
func nz[T any](s []T) []T {
	if len(s) == 0 {
		return nil
	}
	return s
}

// TestRegion_EqualsSnapshotFilter verifies that Region(g) equals the rows of the
// full Snapshot with that region, for every region of every pack, so that the module
// rows (endowments, shipments, journeys) are in it when the module is on.
//
// Requirement: FR-INT-054, FR-OBS-044
func TestRegion_EqualsSnapshotFilter(t *testing.T) {
	t.Parallel()
	for name, layers := range queryPacks() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := queryWorld(t, layers, 25)
			snap, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() = %v, want nil", err)
			}
			for g := range snap.Regions {
				got, err := w.Region(economysim.RegionID(g))
				if err != nil {
					t.Fatalf("Region(%d) = %v, want nil", g, err)
				}
				id := economysim.RegionID(g)
				want := economysim.RegionState{
					Region:     snap.Regions[g],
					Roles:      filter(snap.Roles, func(r economysim.RoleRow) bool { return r.Region == id }),
					Markets:    filter(snap.Markets, func(m economysim.MarketRow) bool { return m.State.Region == id }),
					Jobs:       filter(snap.Jobs, func(j economysim.JobRow) bool { return j.Region == id }),
					Endowments: filter(snap.Endowments, func(e economysim.EndowmentRow) bool { return e.Region == id }),
					Shipments:  filter(snap.Shipments, func(s economysim.ShipmentRow) bool { return s.Region == id || s.To == id }),
					Journeys:   filter(snap.Journeys, func(j economysim.JourneyRow) bool { return j.From == id || j.To == id }),
				}
				got.Roles, got.Markets, got.Jobs = nz(got.Roles), nz(got.Markets), nz(got.Jobs)
				got.Endowments, got.Shipments, got.Journeys = nz(got.Endowments), nz(got.Shipments), nz(got.Journeys)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("Region(%d) = %+v, want the snapshot's rows %+v", g, got, want)
				}
			}
			if _, err := w.Region(99); !errors.Is(err, economysim.ErrUnknownRegion) {
				t.Errorf("Region(99) = %v, want ErrUnknownRegion", err)
			}
		})
	}
}

// filter returns the elements of rows for which keep is true, nil if none.
func filter[T any](rows []T, keep func(T) bool) []T {
	var out []T
	for _, r := range rows {
		if keep(r) {
			out = append(out, r)
		}
	}
	return out
}

// TestWarnings_SortedAndRestored verifies that Warnings returns what
// Config.Warnings returned, sorted by path then rule, that the result is a deep copy
// and that a restored world holds the same list.
//
// Requirement: FR-INT-062
func TestWarnings_SortedAndRestored(t *testing.T) {
	t.Parallel()
	var cfg economysim.Config
	var want []economysim.Warning
	for _, layers := range queryPacks() {
		if cfg = loadValidPack(t, layers...); len(cfg.Warnings()) > 1 {
			want = cfg.Warnings()
			break
		}
	}
	if len(want) < 2 {
		t.Fatal("no pack of the suite raises two warnings, so there is no order to check")
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(7))
	if err != nil {
		t.Fatalf("NewWorld() = %v, want nil", err)
	}
	if got := w.Warnings(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Warnings() = %v, want %v", got, want)
	}
	got := w.Warnings()
	got[0].Path[0] = "changed"
	got[0].Rule = "changed"
	if again := w.Warnings(); !reflect.DeepEqual(again, want) {
		t.Errorf("changing the result changed the next result: %v", again)
	}
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	r, err := economysim.RestoreWorld(cfg, cp)
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	if got := r.Warnings(); !reflect.DeepEqual(got, want) {
		t.Errorf("restored Warnings() = %v, want %v", got, want)
	}
}

// TestCalendar_DeepCopy verifies that Calendar equals the configuration's and that
// changing the result changes nothing in the world.
//
// Requirement: FR-INT-060
func TestCalendar_DeepCopy(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, packOne...)
	w := queryWorld(t, packOne, 0)
	got := w.Calendar()
	if !reflect.DeepEqual(got, w.Config().Calendar) || !reflect.DeepEqual(got.Months, cfg.Calendar.Months) {
		t.Fatalf("Calendar() = %+v, want the configuration's %+v", got, cfg.Calendar)
	}
	got.Months[0].ID = "changed"
	if again := w.Calendar(); again.Months[0].ID == "changed" {
		t.Error("changing the result changed the world's calendar")
	}
}

// TestQueryAllocations verifies that the readers of the first class allocate
// nothing (spec-integration.md 6.1.3).
//
// Requirement: NFR-INT-001
func TestQueryAllocations(t *testing.T) {
	w := queryWorld(t, packOne, 3)
	var sink int64
	for name, f := range map[string]func(){
		"Tick":       func() { sink += w.Tick() },
		"Err":        func() { _ = w.Err() },
		"ConfigHash": func() { sink += int64(w.ConfigHash()[0]) },
	} {
		if got := testing.AllocsPerRun(50, f); got != 0 {
			t.Errorf("%s allocates %v per call, want 0", name, got)
		}
	}
	_ = sink
}
