package economysim

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// The observability of the storage module (spec-storage.md 14, FR-STO-036, FR-STO-041,
// NFR-STO-002): RoomLimited at P3.6a, the counters that feed it, and the cost with the
// no-op observer and logger.

// TestRoomLimited_OnlyWhenNonZero_AscendingRegionThenRole_NoMap verifies FR-STO-036: the
// event is emitted once per (region, role) in ascending region then role, only when one
// of the six triggers is non-zero; the households given a view ride along and trigger
// nothing; and the event carries no map, pointer or slice.
//
// Arithmetic: the world has two regions (home 0, far 1) and two roles (miller 0, monk 1),
// so the cells in order are (0,0) (0,1) (1,0) (1,1). Cell (0,0) has only households (5
// limited, 1 full) and is silent; (0,1) has one shipment refused; (1,0) has 3 bids
// rejected and 2 cut; (1,1) has 7 discarded units.
//
// Requirement: FR-STO-036
func TestRoomLimited_OnlyWhenNonZero_AscendingRegionThenRole_NoMap(t *testing.T) {
	t.Parallel()
	w, rec := stoWorld(t, nil)
	cell := func(g, r int) *roomCount { return &w.sto.counts[g*len(w.cfg.Roles)+r] }
	*cell(0, 0) = roomCount{limitedHouseholds: 5, fullHouseholds: 1}
	*cell(0, 1) = roomCount{shipmentsRefused: 1, limitedHouseholds: 2}
	*cell(1, 0) = roomCount{bidsRejected: 3, bidsCut: 2, startsCut: 4, startsRefused: 1, limitedHouseholds: 6, fullHouseholds: 2}
	*cell(1, 1) = roomCount{discarded: 7}
	w.p36aRoomLimited()
	got := of[RoomLimited](rec)
	want := []RoomLimited{
		{Region: 0, Role: 1, LimitedHouseholds: 2, ShipmentsRefused: 1},
		{Region: 1, Role: 0, LimitedHouseholds: 6, FullHouseholds: 2, StartsCut: 4, StartsRefused: 1, BidsCut: 2, BidsRejected: 3},
		{Region: 1, Role: 1, Discarded: 7},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RoomLimited events = %+v, want %+v", got, want)
	}
	for _, typ := range []reflect.Type{reflect.TypeOf(RoomLimited{}), reflect.TypeOf(StoreChanged{})} {
		for i := range typ.NumField() {
			switch k := typ.Field(i).Type.Kind(); k {
			case reflect.Map, reflect.Pointer, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
				t.Errorf("%s.%s is a %v, want plain data with no map (FR-STO-035, FR-STO-036)", typ.Name(), typ.Field(i).Name, k)
			}
		}
	}
}

// TestRoomLimited_CountersClearedAtP86 verifies that no counter survives P8.6: a step
// reports the counts it was given and the next tick starts at zero.
//
// Requirement: FR-STO-036
func TestRoomLimited_CountersClearedAtP86(t *testing.T) {
	t.Parallel()
	w, rec := stoWorld(t, nil)
	w.sto.counts[0] = roomCount{bidsRejected: 1, bidsCut: 1, discarded: 1, startsCut: 1, startsRefused: 1, shipmentsRefused: 1, limitedHouseholds: 1, fullHouseholds: 1}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	if n := len(of[RoomLimited](rec)); n == 0 {
		t.Fatalf("no RoomLimited in the step, want the counts it was given reported at P3.6a")
	}
	for i, c := range w.sto.counts {
		if c != (roomCount{}) {
			t.Errorf("counts[%d] = %+v after the step, want zero: none survives P8.6", i, c)
		}
	}
}

// TestRoomLimited_CountsAreMadeWithoutAnObserver verifies that the counters are plain
// adds made always (the sizing needs them with no observer) and that no event is built
// without one: a bid rejected for room is counted in a world with no observer.
//
// Requirement: FR-STO-041
func TestRoomLimited_CountsAreMadeWithoutAnObserver(t *testing.T) {
	t.Parallel()
	w := newMill(t, storageMillConfig(), WithSeed(1))
	w.sto.bidRejected(0, ReasonNoRoom)
	w.sto.bidRejected(0, ReasonInsufficientFunds)
	w.sto.refusedStart(0, ReasonNoRoom)
	if c := w.sto.counts[0]; c.bidsRejected != 1 {
		t.Errorf("bidsRejected = %d, want 1: only no_room counts", c.bidsRejected)
	}
	if c := w.sto.counts[0]; c.startsRefused != 1 {
		t.Errorf("startsRefused = %d, want 1", c.startsRefused)
	}
}

// storageMillConfig is the miller chain with the storage module and no observer.
func storageMillConfig() Config {
	c := millConfig(1, 4)
	c.Modules.Storage = &Storage{Base: &Capacity{Mass: 100, Bulk: 100}, Classes: []StoreClass{{ID: "bin", Mass: 10, Bulk: 10}}}
	return c
}

// TestAllocs_StorageEventNoop verifies FR-STO-041 and NFR-STO-002: with the no-op observer
// and logger, P3.6a with non-zero counts, the count of a Trade view, the check of a
// plan with cuts and the Debug lines of a store allocate nothing: the event is neither
// built nor boxed and the log call is not made.
//
// Requirement: FR-STO-041, NFR-STO-002
func TestAllocs_StorageEventNoop(t *testing.T) {
	w := newMill(t, storageMillConfig(), WithSeed(1))
	a := &w.agents[1]
	p := &TradePlan{Cuts: []RoomCut{{Commodity: 0, Quantity: 1}}, Discards: []Discard{{Commodity: 0, Quantity: 1}}}
	if n := testing.AllocsPerRun(50, func() {
		w.sto.counts[0] = roomCount{bidsCut: 1, startsCut: 1}
		w.p36aRoomLimited()
		w.sto.viewed(0, 1, w.sto.roomFor(a))
		w.checkRoomPlan(a, p)
		w.logStore(a, 0, KindAdded, CauseConsumer)
		w.logRefusedJob(a, 0, ReasonNoRoom)
	}); n != 0 {
		t.Errorf("allocations per P3.6a, view count, plan check and log call with the no-op observer = %v, want 0", n)
	}
}

// TestDebug_JobRefusedForRoom verifies the Debug line "job refused for room" of 14.3:
// a start whose room leaves no upkeep room is refused with no_room, and the line
// carries the tick, the agent, the recipe and the reason, written with no observer
// needed.
//
// Arithmetic: as TestStart_KIsMinK0KRoom, a base room of (33, 22) has a free room (3, 2)
// that holds the growth of one execution but leaves no upkeep room.
//
// Requirement: FR-STO-041, FR-STO-046
func TestDebug_JobRefusedForRoom(t *testing.T) {
	t.Parallel()
	c := newCapture()
	w, _ := stoMillWorld(t, 33, 22, stoMiller{max: 1}, nil, WithLogger(slog.New(c)))
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	var got []logRec
	for _, l := range *c.lines {
		if l.msg == "job refused for room" {
			got = append(got, l)
		}
	}
	if len(got) == 0 {
		t.Fatalf("no Debug line \"job refused for room\", want one for each refused start")
	}
	l := got[0]
	if l.level != slog.LevelDebug || l.strs[logkeys.Reason] != string(ReasonNoRoom) {
		t.Errorf("line = %+v, want Debug with the reason no_room", l)
	}
	for _, k := range []string{logkeys.Tick, logkeys.Agent, logkeys.Recipe} {
		if _, ok := l.attrs[k]; !ok {
			t.Errorf("line has no %q attribute, want tick, agent and recipe", k)
		}
	}
}

// TestRoomLimited_DiscardedCountsOnlyTheDefaultsDiscard verifies that the units of
// RoomLimited.Discarded are the default strategy's discard of 6.3.2 and no other: a
// plan whose Discards were added by a directive or a consumer's strategy leaves the
// count at 0.
//
// Requirement: FR-STO-036
//
// Fix: review of STO PR 4. checkRoomPlan summed every Discard of a limited agent's plan,
// so a directive's discard raised Discarded and could trigger a RoomLimited by itself.
func TestRoomLimited_DiscardedCountsOnlyTheDefaultsDiscard(t *testing.T) {
	t.Parallel()
	w := newMill(t, storageMillConfig(), WithSeed(1))
	a := &w.agents[1]
	cell := w.cellOf(w.regionOf(a), a)
	w.checkRoomPlan(a, &TradePlan{Discards: []Discard{{Commodity: 0, Quantity: 5}}})
	if got := w.sto.counts[cell].discarded; got != 0 {
		t.Errorf("discarded after a plan with a foreign discard = %d, want 0", got)
	}
	// Two units are the default's (roomDiscarded) and 5 more were added after them.
	w.checkRoomPlan(a, &TradePlan{Discards: []Discard{{Commodity: 0, Quantity: 2}, {Commodity: 0, Quantity: 5}}, roomDiscarded: 2})
	if got := w.sto.counts[cell].discarded; got != 2 {
		t.Errorf("discarded after the default's 2 units and a foreign 5 = %d, want 2", got)
	}
}
