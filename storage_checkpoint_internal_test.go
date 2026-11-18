package economysim

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/room"
)

// The world of the checkpoint tests is the mill of the storage tests (stoMillWorld):
// households 3 and 4 in home, 5 and 6 in far, a base room of (60, 40) each, a role
// "monk" with no base room, and the classes barn (100, 120), index 0, and shed
// (10, 10), index 1. The stores are saved once in the agent record (tag 3), after the
// agent list, for each live agent in ascending ID: a count of classes and the (class
// index, count) pairs ascending (spec-storage.md 15.1).

// stoCkptWorld is the world of the checkpoint tests, run for ticks steps.
func stoCkptWorld(t *testing.T, mass, bulk int64, strat Strategy, mutate func(c *Config)) *World {
	t.Helper()
	w, _ := stoMillWorld(t, mass, bulk, strat, func(c *Config) {
		monk := DefaultRole()
		monk.ID, monk.Trades = "monk", []string{"grain", "flour"}
		c.Roles = append(c.Roles, monk)
		c.Modules.Storage.Roles = []RoleRoom{{Role: "monk"}}
		c.Modules.Storage.Classes = append(c.Modules.Storage.Classes, StoreClass{ID: "shed", Mass: 10, Bulk: 10})
		c.Modules.Storage = c.Modules.Storage.canonical()
		if mutate != nil {
			mutate(c)
		}
	})
	return w
}

// storesTail is the bytes saveStores writes for w.
func storesTail(w *World) []byte {
	cw := codec.NewWriter(nil)
	w.saveStores(cw)
	return cw.Out()
}

// tamperedTail returns the checkpoint of w with the bytes of its stores block edited in
// place by edit (which must keep the length), its trailer recomputed.
func tamperedTail(t *testing.T, w *World, edit func(tail []byte)) Checkpoint {
	t.Helper()
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	raw, err := ck.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	tail := storesTail(w)
	if n := bytes.Count(raw, tail); n != 1 {
		t.Fatalf("the stores block % x occurs %d times in the checkpoint, want once", tail, n)
	}
	at := bytes.Index(raw, tail)
	edit(raw[at : at+len(tail)])
	body := raw[:len(raw)-sha256.Size]
	sum := sha256.Sum256(body)
	var out Checkpoint
	if err := out.UnmarshalBinary(append(body, sum[:]...)); err != nil {
		t.Fatalf("UnmarshalBinary() of the edited checkpoint error = %v, want nil", err)
	}
	return out
}

// sameStores fails if the stores, the room group and the pledge of the households of r
// differ from w's.
func sameStores(t *testing.T, w, r *World) {
	t.Helper()
	for _, id := range []AgentID{shipHome, shipHome2, shipFar} {
		if got, want := roomOfAgent(t, r, id), roomOfAgent(t, w, id); !reflect.DeepEqual(got, want) {
			t.Errorf("room of agent %d after the restore = %+v, want %+v", id, got, want)
		}
		if !reflect.DeepEqual(r.sto.stores[id], w.sto.stores[id]) {
			t.Errorf("stores of agent %d after the restore = %v, want %v", id, r.sto.stores[id], w.sto.stores[id])
		}
	}
}

// TestCheckpoint_StoresRoundTrip verifies the block of the stores in tag 3: sparse, one
// spelling (an agent with no store writes the count 0 and nothing else), the pairs
// ascending by class index, and a restore that gives back the stores, the room and the
// bytes of a second checkpoint, and again for a restore of a restore.
//
// Arithmetic: household 3 owns two barns and a shed, household 5 a shed, 4 and 6 none:
// 02 00 02 01 01 | 00 | 01 01 01 | 00 (the count of classes, then class and count).
//
// Requirement: FR-STO-039, FR-STO-004, FR-CHK-050
//
// Guard: a restore that dropped the stores would hand back the room of the role row
// alone, and only a world that owns a store the row did not give shows it.
func TestCheckpoint_StoresRoundTrip(t *testing.T) {
	t.Parallel()
	w := stoCkptWorld(t, 60, 40, stoMiller{max: 1}, nil)
	stoAdd(t, w, shipHome, 0)
	stoAdd(t, w, shipHome, 0)
	stoAdd(t, w, shipHome, 1)
	stoAdd(t, w, shipFar, 1)
	step(t, w, 2)
	want := []byte{2, 0, 2, 1, 1, 0, 1, 1, 1, 0}
	if got := storesTail(w); !bytes.Equal(got, want) {
		t.Fatalf("saveStores = % x, want % x", got, want)
	}
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	r, err := RestoreWorld(w.Config(), ck, WithStrategy("script", stoMiller{max: 1}), WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	sameStores(t, w, r)
	if got := storesTail(r); !bytes.Equal(got, want) {
		t.Errorf("saveStores of the restored world = % x, want % x", got, want)
	}
	again, err := r.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() of the restored world error = %v, want nil", err)
	}
	a, _ := ck.MarshalBinary()
	b, _ := again.MarshalBinary()
	if !bytes.Equal(a, b) {
		t.Error("the checkpoint of the restored world differs from the one it was restored from")
	}
}

// TestCheckpoint_RestoreRejects_StoreBlock verifies that a stores block that breaks a
// rule of spec-storage.md 15.3 is ErrCheckpointCorrupt, and that nothing is returned: a
// class index at or above the number of classes, a class that is not above the one
// before it, a count of 0 for a class, a count above MaxStoresPerAgent, and a store
// owned by an agent of a role with no base room.
//
// Requirement: FR-STO-039
//
// Guard: the block is read before any of its meaning is checked, so each rule is a
// separate branch of the reader; a missing one would build a world with a store it
// cannot price.
func TestCheckpoint_RestoreRejects_StoreBlock(t *testing.T) {
	t.Parallel()
	// the block of the world below is 02 00 02 01 01 | 00 | 01 01 01 | 00
	tests := []struct {
		name string
		edit func(tail []byte)
	}{
		{"class index above the classes", func(b []byte) { b[1] = 2 }},
		{"classes not ascending", func(b []byte) { b[1], b[3] = 1, 0 }},
		{"classes equal", func(b []byte) { b[3] = 0 }},
		{"a count of 0", func(b []byte) { b[2] = 0 }},
		{"above MaxStoresPerAgent", func(b []byte) { b[2] = MaxStoresPerAgent + 1 }},
		{"above MaxStoresPerAgent in all", func(b []byte) { b[2], b[4] = MaxStoresPerAgent, 1 }},
		{"a count of classes above the bytes", func(b []byte) { b[0] = 0x7f }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := stoCkptWorld(t, 60, 40, stoMiller{max: 1}, nil)
			stoAdd(t, w, shipHome, 0)
			stoAdd(t, w, shipHome, 0)
			stoAdd(t, w, shipHome, 1)
			stoAdd(t, w, shipFar, 1)
			step(t, w, 2) // a world at tick 0 is refused for its ledger, whatever the block says
			ck := tamperedTail(t, w, tt.edit)
			got, err := RestoreWorld(w.Config(), ck, WithStrategy("script", stoMiller{max: 1}))
			if !errors.Is(err, ErrCheckpointCorrupt) || got != nil {
				t.Errorf("RestoreWorld() = (%v, %v), want (nil, ErrCheckpointCorrupt)", got, err)
			}
		})
	}
}

// TestCheckpoint_RestoreRejects_StoreOnUnlimitedRole verifies that a world whose agent of
// a role with no base room owns a store (a state the library never reaches, a role change
// ends the stores) is refused by the restore.
//
// Requirement: FR-STO-039
func TestCheckpoint_RestoreRejects_StoreOnUnlimitedRole(t *testing.T) {
	t.Parallel()
	w := stoCkptWorld(t, 60, 40, stoMiller{max: 1}, nil)
	step(t, w, 2) // a world at tick 0 is refused for its ledger, whatever the block says
	stoAdd(t, w, shipHome, 0)
	w.agents[shipHome].role = RoleID(len(w.cfg.Roles) - 1) // the monk, no base room
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	got, err := RestoreWorld(w.Config(), ck, WithStrategy("script", stoMiller{max: 1}))
	if !errors.Is(err, ErrCheckpointCorrupt) || got != nil {
		t.Errorf("RestoreWorld() = (%v, %v), want (nil, ErrCheckpointCorrupt)", got, err)
	}
}

// TestCheckpoint_RestoreRejects_LoadOverRoom verifies that a checkpoint whose lots do not
// fit the room its stores give is ErrCheckpointCorrupt, in the restore and not in the
// first tick after it.
//
// Arithmetic: household 3 holds 50 grain, (150, 100), in a room of (60, 40) plus a barn
// (100, 120): (160, 160). The block is edited from the barn to the shed (10, 10): the
// room is (70, 50), below the load in both dimensions.
//
// Requirement: FR-STO-039
func TestCheckpoint_RestoreRejects_LoadOverRoom(t *testing.T) {
	t.Parallel()
	w := stoCkptWorld(t, 60, 40, stoMiller{max: 1}, nil)
	step(t, w, 2) // a world at tick 0 is refused for its ledger, whatever the block says
	stoAdd(t, w, shipHome, 0)
	stoGrain(t, w, shipHome, 40) // 10 of the start and 40: 50 grain
	ck := tamperedTail(t, w, func(b []byte) { b[1] = 1 })
	got, err := RestoreWorld(w.Config(), ck, WithStrategy("script", stoMiller{max: 1}))
	if !errors.Is(err, ErrCheckpointCorrupt) || got != nil {
		t.Errorf("RestoreWorld() = (%v, %v), want (nil, ErrCheckpointCorrupt)", got, err)
	}
}

// TestCheckpoint_PledgeAboveRoomAccepted verifies that the checkpoint of a world after a
// cut of the room, whose pledge is above the room (spec-storage.md 5.5.3, 18.4), is
// restored with no refusal, the pledge is rebuilt from the job, and the restored world
// goes on to the same completion as the one that never stopped.
//
// Arithmetic: as TestWeightReduction_LoadFitsPledgeMayExceed: the job of weight 10 pledges
// (300, 200) and a reduction to weight 5 leaves the room (180, 120).
//
// Requirement: FR-STO-017, FR-STO-020, FR-STO-039
//
// Guard: the pledge is derived and never tested against the room, so a restore that
// tested it would refuse the checkpoint of an ordinary world.
func TestCheckpoint_PledgeAboveRoomAccepted(t *testing.T) {
	t.Parallel()
	w := stoCkptWorld(t, 36, 24, stoMiller{max: 1}, func(c *Config) { c.Recipes[0].Outputs[0].Quantity = 100 })
	a := &w.agents[shipHome]
	a.weight = 10
	w.weightChangedStorage(a)
	step(t, w, 1)
	if err := w.execReduce(regHome, a, Decision{Agent: shipHome, Households: 5, Cause: CauseStarved}); err != nil {
		t.Fatalf("execReduce() = %v, want nil", err)
	}
	before := roomOfAgent(t, w, shipHome)
	if before.Pledge.Mass <= before.Room.Mass {
		t.Fatalf("pledge %+v is not above room %+v, want the cut case", before.Pledge, before.Room)
	}
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	r, err := RestoreWorld(w.Config(), ck, WithStrategy("script", stoMiller{max: 1}), WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld() of a pledge above the room error = %v, want nil", err)
	}
	if got := roomOfAgent(t, r, shipHome); !reflect.DeepEqual(got, before) {
		t.Errorf("room group after the restore = %+v, want %+v", got, before)
	}
	step(t, w, 2)
	step(t, r, 2)
	x, _ := w.Checkpoint()
	y, _ := r.Checkpoint()
	xb, _ := x.MarshalBinary()
	yb, _ := y.MarshalBinary()
	if !bytes.Equal(xb, yb) {
		t.Error("the restored world's checkpoint after the completion differs from the original's")
	}
}

// TestCheckpoint_RestoreRejects_PledgeOverflow verifies that a checkpoint whose jobs
// pledge a load that does not fit int64 is ErrCheckpointCorrupt from rebuildStorage, and
// that pledgedByJobs reports it as an error: syncJob and repledge raise an invariant on
// such a sum, and a checkpoint is input.
//
// Arithmetic: with big the larger dimension of the output's size, the first case pends
// MaxInt64 units, whose load is MaxInt64*big and overflows when big is above 1; the second
// gives two jobs of one principal MaxInt64/big units each, which fit alone, and 2*(MaxInt64/big)*big
// is above MaxInt64.
//
// Requirement: FR-STO-039, FR-STO-017
//
// Guard: without the check a crafted checkpoint panics the restore instead of returning
// an error (the fix of the commit that added it).
func TestCheckpoint_RestoreRejects_PledgeOverflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		units func(per int64) []int64 // pending units of the jobs of shipHome and shipHome2
	}{
		{"one job above int64", func(int64) []int64 { return []int64{math.MaxInt64, 0} }},
		{"two jobs of one principal sum above int64", func(per int64) []int64 { return []int64{per, per} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := stoCkptWorld(t, 60, 40, stoMiller{max: 1}, nil)
			out := w.sto.recipes.row(0).outputs[0]
			sz := w.sto.sizes[out.commodity]
			big := max(sz.Mass, sz.Bulk)
			if big < 1 {
				t.Fatalf("size of the first output = %+v, want a positive dimension", sz)
			}
			units := tt.units(math.MaxInt64 / big)
			for i, id := range []AgentID{shipHome, shipHome2} {
				if units[i] == 0 {
					continue
				}
				a := &w.agents[id]
				pending := make([]int64, len(w.sto.recipes.row(0).outputs))
				pending[0] = units[i]
				a.job = job{recipe: 0, remaining: 1, executions: 1, principal: shipHome, pending: pending}
			}
			if _, err := w.sto.pledgedByJobs(w); err == nil {
				t.Fatal("pledgedByJobs() error = nil, want the overflow")
			}
			if err := w.rebuildStorage(); !errors.Is(err, ErrCheckpointCorrupt) {
				t.Errorf("rebuildStorage() = %v, want ErrCheckpointCorrupt", err)
			}
		})
	}
}

// TestCheckpoint_RestoreRejects_CargoOverflow verifies that cargo in flight whose load
// does not fit int64 is ErrCheckpointCorrupt from rebuildStorage, and that a shipment
// with no line or a consignee outside the table adds nothing to the cargo.
//
// Arithmetic: a line of MaxInt64 units of grain, whose size is above 1 in a dimension
// (shipConfig: mass 3, bulk 2), has the load MaxInt64*3, which overflows.
//
// Requirement: FR-STO-039, FR-STO-043
//
// Guard: the cargo is rebuilt from the shipments of the checkpoint, which are input; an
// unchecked sum would wrap or raise an invariant instead of refusing the checkpoint.
func TestCheckpoint_RestoreRejects_CargoOverflow(t *testing.T) {
	t.Parallel()
	w := stoCkptWorld(t, 60, 40, stoMiller{max: 1}, nil)
	step(t, w, 2)
	grain := commodityOf(t, w, "grain")
	w.transport.ships = append(w.transport.ships,
		shipment{consignee: shipHome}, // no line: skipped
		shipment{consignee: AgentID(1 << 20), lines: []Line{{Commodity: grain, Quantity: 1}}}, // outside the table: skipped
	)
	got, err := w.sto.cargoInFlight(w)
	if err != nil {
		t.Fatalf("cargoInFlight() with skipped shipments error = %v, want nil", err)
	}
	for id, p := range got {
		if p != (room.Pair{}) {
			t.Errorf("cargoInFlight()[%d] = %+v, want the zero load", id, p)
		}
	}
	w.transport.ships = append(w.transport.ships, shipment{consignee: shipHome, lines: []Line{{Commodity: grain, Quantity: math.MaxInt64}}})
	if _, err := w.sto.cargoInFlight(w); err == nil {
		t.Error("cargoInFlight() of a line of MaxInt64 units error = nil, want the overflow")
	}
	if err := w.rebuildStorage(); !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("rebuildStorage() = %v, want ErrCheckpointCorrupt", err)
	}
}

// TestCheckpoint_RestoreRejects_PledgePlusCargoOverflow verifies that a pledge of a job
// and a cargo in flight that each fit int64 but whose sum does not are refused by
// rebuildStorage as ErrCheckpointCorrupt.
//
// Arithmetic: grain and the first output weigh the same per unit, so MaxInt64/size units
// of each fit alone and together load twice what int64 holds in the larger dimension.
//
// Requirement: FR-STO-039, FR-STO-020
//
// Guard: the pledge of an account is its jobs' load plus its cargo; syncJob and repledge
// raise an invariant on an overflowing sum, so the restore checks it first.
func TestCheckpoint_RestoreRejects_PledgePlusCargoOverflow(t *testing.T) {
	t.Parallel()
	w := stoCkptWorld(t, 60, 40, stoMiller{max: 1}, nil)
	step(t, w, 2)
	grain := commodityOf(t, w, "grain")
	out := w.sto.recipes.row(0).outputs[0]
	sz := w.sto.sizes[out.commodity]
	if sz != w.sto.sizes[grain] {
		t.Fatalf("size of the first output = %+v, want grain's %+v", sz, w.sto.sizes[grain])
	}
	per := math.MaxInt64 / max(sz.Mass, sz.Bulk)
	pending := make([]int64, len(w.sto.recipes.row(0).outputs))
	pending[0] = per
	w.agents[shipHome].job = job{recipe: 0, remaining: 1, executions: 1, principal: shipHome, pending: pending}
	w.transport.ships = append(w.transport.ships, shipment{consignee: shipHome, lines: []Line{{Commodity: grain, Quantity: Quantity(per)}}})
	if err := w.rebuildStorage(); !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("rebuildStorage() = %v, want ErrCheckpointCorrupt", err)
	}
}

// TestCheckpoint_LoadStoresEmptyAgentTable verifies that the stores block of a world
// with no agent table is read as empty without growing the module's tables.
//
// Guard: loadStores grows the tables to the last agent ID; with no agents that ID is -1,
// which as an AgentID would grow the tables without end.
func TestCheckpoint_LoadStoresEmptyAgentTable(t *testing.T) {
	t.Parallel()
	w := stoCkptWorld(t, 60, 40, stoMiller{max: 1}, nil)
	w.agents, w.firstHousehold = nil, 0
	before := len(w.sto.stores)
	w.loadStores(codec.NewReader(nil))
	if got := len(w.sto.stores); got != before {
		t.Errorf("loadStores on an empty agent table: len(stores) = %d, want %d", got, before)
	}
}

// FuzzDecodeAgentRecordStores feeds arbitrary bytes to the reader of the stores block:
// it never panics, a rejected block leaves the reader at an error, and an accepted one
// is read to its end with every count in bounds (NFR-CHK-003). The seeds are the block
// of a world with stores and the edits the tests above make.
//
// Requirement: FR-STO-039, NFR-CHK-003
func FuzzDecodeAgentRecordStores(f *testing.F) {
	for _, s := range [][]byte{
		{0, 0, 0, 0}, {2, 0, 2, 1, 1, 0, 1, 1, 1, 0}, {1, 2, 1, 0, 0, 0}, {1, 1, 0, 0, 0, 0}, {0x7f}, {},
		{1, 0, 9, 0, 0, 0}, {2, 1, 1, 0, 1, 0, 0, 0}, {0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01},
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		w := stoCkptWorld(t, 60, 40, stoMiller{max: 1}, nil)
		r := codec.NewReader(b)
		w.loadStores(r)
		if r.Err() != nil {
			return
		}
		for id := range w.sto.stores {
			if n := w.sto.owned(AgentID(id)); n > MaxStoresPerAgent {
				t.Fatalf("agent %d owns %d stores after an accepted block, want at most %d", id, n, MaxStoresPerAgent)
			}
		}
	})
}
