package economysim

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

// The world of the room tests is shipConfig of the transport tests with the storage
// module: the millers (two households of weight 1 in home, agents 3 and 4, and two in
// far, agents 5 and 6; the estates are 0 to 2) are limited to a base room of (60, 40)
// per household, the sizes are grain (3, 2), flour (3, 2), hay (2, 3), perish (1, 1),
// wagon (10, 10) and haul (0, 0), the upkeep is 1 grain, so one tick's upkeep room of
// a household is (3, 2), and a miller holds 10 grain at the start, a load of (30, 20).
// The one store class is a barn of (100, 120).
const (
	stoBaseMass, stoBaseBulk = 60, 40
	stoBarnMass, stoBarnBulk = 100, 120
)

// stoStorage is the storage module of the room tests: the millers limited, a second
// role "monk" with no base room, and the class "barn".
func stoStorage() *Storage {
	return &Storage{
		Base:    &Capacity{Mass: stoBaseMass, Bulk: stoBaseBulk},
		Roles:   []RoleRoom{{Role: "monk"}},
		Classes: []StoreClass{{ID: "barn", Mass: stoBarnMass, Bulk: stoBarnBulk}, {ID: "shed", Mass: 10, Bulk: 10}},
	}
}

// stoWorld is the world of the room tests, changed by mut after the storage module
// is set: it has a second role "monk" (an unlimited role) so that a role can change.
func stoWorld(t *testing.T, mut func(c *Config), opts ...Option) (*World, *evLog) {
	t.Helper()
	return newShipWorld(t, func(c *Config) {
		monk := DefaultRole()
		monk.ID, monk.Trades = "monk", []string{"grain", "flour"}
		c.Roles = append(c.Roles, monk)
		c.Modules.Storage = stoStorage()
		if mut != nil {
			mut(c)
		}
	}, nil, opts...)
}

// roomOfAgent returns the room group of the agent query, failing the test on an
// error.
func roomOfAgent(t *testing.T, w *World, id AgentID) RoomState {
	t.Helper()
	st, err := w.Agent(id)
	if err != nil {
		t.Fatalf("Agent(%d) = %v, want nil", id, err)
	}
	return st.Room
}

// TestRoom_Formula verifies the room of a limited account: weight times the base
// room plus the capacity of each store, a store's capacity not multiplied by the
// weight, and the pair kept apart (a mass is never compared with a bulk).
//
// Arithmetic: weight 3 gives 3 * (60, 40) = (180, 120); a barn adds (100, 120) once,
// not three times: (280, 240); a shed adds (10, 10): (290, 250).
//
// Requirement: FR-STO-004
func TestRoom_Formula(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	a := &w.agents[shipHome]
	a.weight = 3
	steps := []struct {
		name  string
		class StoreClassID
		want  Capacity
	}{
		{"no store", -1, Capacity{Mass: 180, Bulk: 120}},
		{"a barn", 0, Capacity{Mass: 280, Bulk: 240}},
		{"a shed", 1, Capacity{Mass: 290, Bulk: 250}},
	}
	for _, s := range steps {
		if s.class >= 0 {
			if err := w.addStore(a, s.class, CauseConsumer); err != nil {
				t.Fatalf("addStore(%s) = %v, want nil", s.name, err)
			}
		}
		if err := w.sto.apply(w.ledger, a); err != nil {
			t.Fatalf("apply() after %s = %v, want nil", s.name, err)
		}
		if got := roomOfAgent(t, w, shipHome).Room; got != s.want {
			t.Errorf("room after %s = %+v, want %+v", s.name, got, s.want)
		}
	}
}

// TestLimited_IffBaseRoom_PseudoAgentNever verifies that an agent is limited if and
// only if its role has a base room, and that an estate pseudo-agent never is.
//
// Requirement: FR-STO-003
func TestLimited_IffBaseRoom_PseudoAgentNever(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	monk, err := w.AddAgent(NewAgent{Region: regHome, Role: 1, Money: 10})
	if err != nil {
		t.Fatalf("AddAgent(monk) = %v, want nil", err)
	}
	if !roomOfAgent(t, w, shipHome).Limited || roomOfAgent(t, w, monk).Limited {
		t.Errorf("limited: miller %t, monk %t, want true and false", roomOfAgent(t, w, shipHome).Limited, roomOfAgent(t, w, monk).Limited)
	}
	for id := range w.firstHousehold {
		rm, err := w.ledger.RoomOf(w.estates.accounts[id])
		if err != nil || rm.Limited {
			t.Errorf("RoomOf(estate %d) = %+v, %v, want an account that is not limited", id, rm, err)
		}
	}
}

// TestAddStore_Errors verifies each refusal of the table of spec-storage.md 3.6.2
// for AddStore, in its order, that a refused call changes nothing (the room group of
// the agent and no event) and that the success path adds one store.
//
// Requirement: FR-STO-032
func TestAddStore_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T, w *World)
		agent AgentID
		class StoreClassID
		want  error
	}{
		{"pseudo-agent", nil, 0, 0, ErrUnknownAgent},
		{"never issued", nil, 99, 0, ErrUnknownAgent},
		{"unknown class below", nil, shipHome, -1, ErrUnknownStoreClass},
		{"unknown class above", nil, shipHome, 2, ErrUnknownStoreClass},
		{"unlimited role", func(t *testing.T, w *World) {
			if _, err := w.AddAgent(NewAgent{Region: regHome, Role: 1}); err != nil {
				t.Fatalf("AddAgent(monk) = %v", err)
			}
		}, 7, 0, ErrNotLimited},
		{"away on a trip", func(t *testing.T, w *World) { w.transport.setAway(shipHome, true) }, shipHome, 0, ErrAgentAway},
		{"too many stores", func(t *testing.T, w *World) {
			for range MaxStoresPerAgent {
				if err := w.AddStore(shipHome, 1); err != nil {
					t.Fatalf("AddStore() = %v, want nil", err)
				}
			}
		}, shipHome, 0, ErrOutOfBounds},
		{"class locked in the region", func(t *testing.T, w *World) { w.sto.gate = func(RegionID, int) bool { return false } }, shipHome, 0, ErrStoreLocked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := stoWorld(t, nil)
			if tt.setup != nil {
				tt.setup(t, w)
			}
			before, events := roomStateAt(t, w, tt.agent), len(rec.events)
			if err := w.AddStore(tt.agent, tt.class); !errors.Is(err, tt.want) {
				t.Errorf("AddStore(%d, %d) = %v, want %v", tt.agent, tt.class, err, tt.want)
			}
			if after := roomStateAt(t, w, tt.agent); !reflect.DeepEqual(after, before) || len(rec.events) != events {
				t.Errorf("a refused AddStore changed the room group %+v to %+v or emitted %d events, want nothing", before, after, len(rec.events)-events)
			}
		})
	}
}

// roomStateAt is the room group of an agent, or the zero value for an ID the query
// refuses (a pseudo-agent, an ID never issued), so a refused call can be compared.
func roomStateAt(t *testing.T, w *World, id AgentID) RoomState {
	t.Helper()
	st, err := w.Agent(id)
	if err != nil {
		return RoomState{}
	}
	return st.Room
}

// TestAddStore_ModuleOffAndRemoved verifies the two refusals that need another
// world: a world with the module off refuses with ErrModuleOff, and an agent that
// was removed with ErrAgentRemoved.
//
// Requirement: FR-STO-032, FR-STO-033
func TestAddStore_ModuleOffAndRemoved(t *testing.T) {
	t.Parallel()
	off, _ := newShipWorld(t, nil, nil)
	if err := off.AddStore(shipHome, 0); !errors.Is(err, ErrModuleOff) {
		t.Errorf("AddStore() with the module off = %v, want ErrModuleOff", err)
	}
	if err := off.RemoveStore(shipHome, 0); !errors.Is(err, ErrModuleOff) {
		t.Errorf("RemoveStore() with the module off = %v, want ErrModuleOff", err)
	}
	w, _ := stoWorld(t, nil)
	ext, err := w.AddAgent(NewAgent{Region: regHome, Role: 0, Money: 5})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	if err := w.RemoveAgent(ext); err != nil {
		t.Fatalf("RemoveAgent() = %v, want nil", err)
	}
	if err := w.AddStore(ext, 0); !errors.Is(err, ErrAgentRemoved) {
		t.Errorf("AddStore(removed agent) = %v, want ErrAgentRemoved", err)
	}
	if err := w.RemoveStore(ext, 0); !errors.Is(err, ErrAgentRemoved) {
		t.Errorf("RemoveStore(removed agent) = %v, want ErrAgentRemoved", err)
	}
}

// TestRemoveStore_Errors verifies the refusals of RemoveStore: the agent owns none of
// the class (ErrOutOfBounds), the stores of the class would fall below the installed
// store facilities (ErrStoreInUse, through the seam of D9), and the account's load
// plus pledge, or its pledge plus its upkeep room, would exceed the room
// (ErrNoRoom); and that the call is allowed for an agent that is away.
//
// Arithmetic: a miller of room (60, 40) and a barn (100, 120) has (160, 160); 140 mass
// of goods fit the full room but not (60, 40) once the barn is gone. A pledge of (30,
// 0) with a load of (30, 0) is 60 mass, which (60, 40) holds, but the upkeep room (3,
// 2) added to the pledge, 33, is within it too: only the load plus the pledge binds.
//
// Requirement: FR-STO-033
func TestRemoveStore_Errors(t *testing.T) {
	t.Parallel()
	give := func(w *World, mass int64) { // units of hay (2, 3) would bind bulk; grain (3, 2) binds mass
		clearGoods(t, w, shipHome, commodityOf(t, w, "grain"))
		stoGrain(t, w, shipHome, mass/3)
	}
	tests := []struct {
		name  string
		setup func(t *testing.T, w *World)
		want  error
	}{
		{"owns none", nil, ErrOutOfBounds},
		{"facility holds it", func(t *testing.T, w *World) {
			stoAdd(t, w, shipHome, 0)
			w.sto.facilityCount = func(*agent, StoreClassID) int { return 1 }
		}, ErrStoreInUse},
		{"load would exceed the room", func(t *testing.T, w *World) {
			stoAdd(t, w, shipHome, 0)
			give(w, 120) // 40 grain, load (120, 80): within (160, 160), above (60, 40)
		}, ErrNoRoom},
		{"pledge would exceed the room", func(t *testing.T, w *World) {
			stoAdd(t, w, shipHome, 0)
			give(w, 30)
			if err := w.ledger.SetPledge(w.agents[shipHome].account, 50, 0); err != nil {
				t.Fatalf("SetPledge() = %v, want nil", err)
			}
		}, ErrNoRoom},
		{"fits and succeeds", func(t *testing.T, w *World) { stoAdd(t, w, shipHome, 0) }, nil},
		{"away is allowed", func(t *testing.T, w *World) {
			stoAdd(t, w, shipHome, 0)
			w.transport.setAway(shipHome, true)
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := stoWorld(t, nil)
			if tt.setup != nil {
				tt.setup(t, w)
			}
			before := roomStateAt(t, w, shipHome)
			err := w.RemoveStore(shipHome, 0)
			if !errors.Is(err, tt.want) && !(tt.want == nil && err == nil) {
				t.Fatalf("RemoveStore() = %v, want %v", err, tt.want)
			}
			if after := roomStateAt(t, w, shipHome); tt.want != nil && !reflect.DeepEqual(after, before) {
				t.Errorf("a refused RemoveStore changed the room group %+v to %+v, want nothing", before, after)
			}
		})
	}
}

// stoAdd gives the agent a store of the class, failing the test on an error.
func stoAdd(t *testing.T, w *World, id AgentID, class StoreClassID) {
	t.Helper()
	if err := w.AddStore(id, class); err != nil {
		t.Fatalf("AddStore(%d, %d) = %v, want nil", id, class, err)
	}
}

// stoGrain gives an agent n units of grain from the initial boundary.
func stoGrain(t *testing.T, w *World, id AgentID, n int64) {
	t.Helper()
	give(t, w, id, commodityOf(t, w, "grain"), n)
}

// TestStoreChanged_Fields_NoMap_OnlyAtCall verifies the event of a call: one
// StoreChanged for AddStore and one for RemoveStore, with the agent, region, role,
// class, kind, cause consumer, the count of the class after and the room and load of
// the account after; no step emits one while no call is made; and the event type has
// no map, pointer or slice field.
//
// Arithmetic: the miller holds 10 grain (3, 2), a load of (30, 20), and has the room
// (60, 40); with a barn the room is (160, 160).
//
// Requirement: FR-STO-035
func TestStoreChanged_Fields_NoMap_OnlyAtCall(t *testing.T) {
	t.Parallel()
	w, rec := stoWorld(t, nil)
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	if got := of[StoreChanged](rec); len(got) != 0 {
		t.Fatalf("StoreChanged events after a step with no call = %v, want none", got)
	}
	stoAdd(t, w, shipHome, 0)
	if err := w.RemoveStore(shipHome, 0); err != nil {
		t.Fatalf("RemoveStore() = %v, want nil", err)
	}
	got := of[StoreChanged](rec)
	load := w.agents[shipHome].account
	_ = load
	want := []StoreChanged{
		{Tick: w.Tick(), Agent: shipHome, Region: regHome, Role: 0, Class: 0, Kind: KindAdded, Cause: CauseConsumer, Stores: 1,
			Room: Capacity{Mass: 160, Bulk: 160}, Load: got[0].Load},
		{Tick: w.Tick(), Agent: shipHome, Region: regHome, Role: 0, Class: 0, Kind: KindRemoved, Cause: CauseConsumer, Stores: 0,
			Room: Capacity{Mass: 60, Bulk: 40}, Load: got[1].Load},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("StoreChanged events = %+v, want %+v", got, want)
	}
	if got[0].Load.Mass <= 0 {
		t.Errorf("StoreChanged.Load = %+v, want the load of the account (the start stock)", got[0].Load)
	}
	typ := reflect.TypeOf(StoreChanged{})
	for i := range typ.NumField() {
		switch k := typ.Field(i).Type.Kind(); k {
		case reflect.Map, reflect.Pointer, reflect.Slice, reflect.Func, reflect.Interface:
			t.Errorf("StoreChanged.%s has kind %v, want a plain value (observability-and-logging.md 4.1)", typ.Field(i).Name, k)
		}
	}
}

// TestCreation_GivesRoleRowStores_NoEvent verifies that an agent created in a role
// whose row lists stores owns them from its creation, in class order with repeats,
// that no StoreChanged is emitted for them, and that AgentSpawned reports no locked
// store (the gate is open until the settlements module exists).
//
// Arithmetic: two barns and a shed give (60, 40) + 2 * (100, 120) + (10, 10) =
// (270, 290) to a household of weight 1.
//
// Requirement: FR-STO-034
func TestCreation_GivesRoleRowStores_NoEvent(t *testing.T) {
	t.Parallel()
	w, rec := stoWorld(t, func(c *Config) {
		c.Modules.Storage.Roles = append(c.Modules.Storage.Roles, RoleRoom{Role: "miller", Base: &Capacity{Mass: stoBaseMass, Bulk: stoBaseBulk}, Stores: []string{"shed", "barn", "barn"}})
		c.Modules.Storage = c.Modules.Storage.canonical()
	})
	st := roomOfAgent(t, w, shipHome)
	if want := (Capacity{Mass: 270, Bulk: 290}); st.Room != want {
		t.Errorf("room of a created miller = %+v, want %+v", st.Room, want)
	}
	if want := []StoreCount{{Class: 0, Count: 2}, {Class: 1, Count: 1}}; !slices.Equal(st.Stores, want) {
		t.Errorf("stores of a created miller = %+v, want %+v (class order)", st.Stores, want)
	}
	if got := of[StoreChanged](rec); len(got) != 0 {
		t.Errorf("StoreChanged events at creation = %+v, want none", got)
	}
	for _, e := range of[AgentSpawned](rec) {
		if e.StoresLocked != 0 {
			t.Errorf("AgentSpawned.StoresLocked = %d, want 0 with the gate open", e.StoresLocked)
		}
	}
}

// TestRoleChange_ToUnlimitedEndsStoresInClassOrderAfterRoleChanged verifies that a
// role change to a role with no base room ends the agent's stores in class order,
// each with a StoreChanged of kind removed and cause role_changed after the
// RoleChanged, and leaves the account unlimited, holding its goods.
//
// Requirement: FR-STO-034
func TestRoleChange_ToUnlimitedEndsStoresInClassOrderAfterRoleChanged(t *testing.T) {
	t.Parallel()
	w, rec := stoWorld(t, nil)
	stoAdd(t, w, shipHome, 1)
	stoAdd(t, w, shipHome, 0)
	stoAdd(t, w, shipHome, 0)
	rec.events = nil
	a := &w.agents[shipHome]
	if err := w.execSwitch(regHome, a, Decision{Role: 1, Cause: CauseReview}); err != nil {
		t.Fatalf("execSwitch() = %v, want nil", err)
	}
	var kinds []string
	var ended []StoreChanged
	for _, e := range rec.events {
		switch x := e.(type) {
		case RoleChanged:
			kinds = append(kinds, "role")
		case StoreChanged:
			kinds = append(kinds, "store")
			ended = append(ended, x)
		}
	}
	if want := []string{"role", "store", "store", "store"}; !slices.Equal(kinds, want) {
		t.Fatalf("events of the change = %v, want %v (the stores after the RoleChanged)", kinds, want)
	}
	classes, counts := make([]StoreClassID, len(ended)), make([]int64, len(ended))
	for i, e := range ended {
		classes[i], counts[i] = e.Class, e.Stores
		if e.Kind != KindRemoved || e.Cause != CauseRoleChanged || e.Role != 1 {
			t.Errorf("StoreChanged[%d] = %+v, want kind removed, cause role_changed, role 1 (the new one)", i, e)
		}
	}
	if !slices.Equal(classes, []StoreClassID{0, 0, 1}) || !slices.Equal(counts, []int64{1, 0, 0}) {
		t.Errorf("ended stores by class %v with counts %v, want classes [0 0 1] (class order) and counts [1 0 0]", classes, counts)
	}
	if st := roomOfAgent(t, w, shipHome); st.Limited || len(st.Stores) != 0 {
		t.Errorf("room group of a monk = %+v, want unlimited with no store", st)
	}
	if held, _, _ := w.ledger.Holding(a.account, int(commodityOf(t, w, "grain"))); held != 10 {
		t.Errorf("grain held after the change = %d, want its 10 (the goods stay)", held)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestRoleChange_ToLimitedKeepsStores verifies that a role change to another limited
// role leaves the stores with the agent and re-evaluates the room.
//
// Requirement: FR-STO-034
func TestRoleChange_ToLimitedKeepsStores(t *testing.T) {
	t.Parallel()
	w, rec := stoWorld(t, func(c *Config) {
		c.Modules.Storage.Roles = nil // every role limited: monk too
	})
	stoAdd(t, w, shipHome, 0)
	rec.events = nil
	if err := w.execSwitch(regHome, &w.agents[shipHome], Decision{Role: 1, Cause: CauseReview}); err != nil {
		t.Fatalf("execSwitch() = %v, want nil", err)
	}
	if got := of[StoreChanged](rec); len(got) != 0 {
		t.Errorf("StoreChanged events of a change between limited roles = %+v, want none", got)
	}
	if st := roomOfAgent(t, w, shipHome); !st.Limited || len(st.Stores) != 1 || st.Room != (Capacity{Mass: 160, Bulk: 160}) {
		t.Errorf("room group after the change = %+v, want limited with the barn and room (160, 160)", st)
	}
}

// TestRelocation_RefusedHasStore verifies that an agent that owns a store cannot be
// relocated, with the region_bound refusal has_store and no instance key, and can
// once the store is removed; the refusal is registered only with the module on.
//
// Requirement: FR-STO-030
func TestRelocation_RefusedHasStore(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	if reason, err := w.RelocationRefusal(shipHome, regFar); err != nil {
		t.Fatalf("RelocationRefusal() with no store = %q, %v, want none", reason, err)
	}
	stoAdd(t, w, shipHome, 0)
	reason, err := w.RelocationRefusal(shipHome, regFar)
	var rb *RegionBoundError
	if !errors.As(err, &rb) || rb.Obligation != "has_store" || reason != string(ReasonRegionBound) {
		t.Errorf("RelocationRefusal() with a store = %q, %v, want region_bound naming has_store", reason, err)
	}
	if reason, err := w.RelocationRefusal(shipHome, regFar, "has_store"); err != nil {
		t.Errorf("RelocationRefusal(skip has_store) = %q, %v, want none: the obligation is named by its bare name", reason, err)
	}
	if err := w.RemoveStore(shipHome, 0); err != nil {
		t.Fatalf("RemoveStore() = %v, want nil", err)
	}
	if err := w.Relocate(shipHome, regFar); err != nil {
		t.Errorf("Relocate() after the store is removed = %v, want nil", err)
	}
	off, _ := newShipWorld(t, nil, nil)
	if len(off.bounds.rows) != 0 && slices.ContainsFunc(off.bounds.rows, func(r regionBound) bool { return r.err.Obligation == "has_store" }) {
		t.Errorf("a world with the module off registers has_store, want it only with the module on")
	}
}

// TestRelocation_ForcedMoveCarriesBaseRoomOnly verifies the closing of the leak of
// note N-120: a limited agent that moves carries its goods, which are within its
// base room times its weight because it owns no store (a store forbids the move), so
// its room after the move is the same base room.
//
// Requirement: FR-STO-030
func TestRelocation_ForcedMoveCarriesBaseRoomOnly(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	before := roomOfAgent(t, w, shipHome)
	if err := w.Relocate(shipHome, regFar); err != nil {
		t.Fatalf("Relocate() = %v, want nil", err)
	}
	after := roomOfAgent(t, w, shipHome)
	if after.Room != before.Room || after.Load.Mass > after.Room.Mass || after.Load.Bulk > after.Room.Bulk {
		t.Errorf("room group after a move = %+v, before %+v, want the same room and a load within it", after, before)
	}
}

// TestAddAgent_NoRoom verifies that AddAgent refuses with ErrNoRoom, and creates
// nothing, when the money-and-goods it would create do not fit the room its role
// gives, counting the stores of the role's row, and accepts the same goods in an
// unlimited role.
//
// Arithmetic: 21 grain is (63, 42), above (60, 40) in both dimensions; 20 grain is
// (60, 40) and fits exactly.
//
// Requirement: FR-STO-019
func TestAddAgent_NoRoom(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	grainID := commodityOf(t, w, "grain")
	next := w.state.nextAgent
	n := NewAgent{Region: regHome, Role: 0, Money: 5, Stock: []StockItem{{Commodity: grainID, Quantity: 21}}}
	if id, err := w.AddAgent(n); !errors.Is(err, ErrNoRoom) || id != NoAgent || w.state.nextAgent != next {
		t.Errorf("AddAgent(21 grain) = %d, %v, next ID %d, want NoAgent, ErrNoRoom and no ID issued", id, err, w.state.nextAgent)
	}
	n.Stock[0].Quantity = 20
	if _, err := w.AddAgent(n); err != nil {
		t.Errorf("AddAgent(20 grain) = %v, want nil: the goods fit the room exactly", err)
	}
	n.Role, n.Stock[0].Quantity = 1, 500
	if _, err := w.AddAgent(n); err != nil {
		t.Errorf("AddAgent(monk, 500 grain) = %v, want nil: an unlimited role has no room to test", err)
	}
}

// TestSnapshot_RoomFields verifies the rows of the snapshot: the role row sums the
// households, room and load of the limited agents of the role in the region, with
// the weight folded in, and the store rows count the stores by (region, class) and
// leave out a class with none.
//
// Arithmetic: home has two millers, each room (60, 40) and load (30, 20): rows
// 120, 80 and 60, 40; with one barn on agent 3 the room is 120 + 100 = 220 and 80 +
// 120 = 200.
//
// Requirement: FR-STO-004
func TestSnapshot_RoomFields(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	stoAdd(t, w, shipHome, 0)
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	nr := len(w.cfg.Roles)
	row := snap.Roles[int(regHome)*nr+0]
	if row.LimitedHouseholds != 2 || row.RoomMass != 220 || row.RoomBulk != 200 || row.LoadMass != 60 || row.LoadBulk != 40 {
		t.Errorf("RoleRow of the millers in home = %+v, want 2 households, room (220, 200), load (60, 40)", row)
	}
	if want := []StoreRow{{Region: regHome, Class: 0, Stores: 1}}; !slices.Equal(snap.Stores, want) {
		t.Errorf("Snapshot.Stores = %+v, want %+v", snap.Stores, want)
	}
}

// TestAgentState_RoomGroup verifies the room group of the agent query for a limited
// agent: room, load, pledge, free and stores; and that it is the zero value for an
// unlimited agent.
//
// Arithmetic: free = room - load - pledge = (60, 40) - (30, 20) - (0, 0) = (30, 20).
//
// Requirement: FR-STO-004
func TestAgentState_RoomGroup(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	got := roomOfAgent(t, w, shipHome)
	want := RoomState{Limited: true, Room: Capacity{Mass: 60, Bulk: 40}, Load: Capacity{Mass: 30, Bulk: 20}, Free: Capacity{Mass: 30, Bulk: 20}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("room group = %+v, want %+v", got, want)
	}
	monk, err := w.AddAgent(NewAgent{Region: regHome, Role: 1})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	if got := roomOfAgent(t, w, monk); !reflect.DeepEqual(got, RoomState{}) {
		t.Errorf("room group of a monk = %+v, want the zero value", got)
	}
}

// TestConstruction_StartGoodsAboveRoom verifies that construction fails with
// ErrRoomShape, never the ledger's refusal, when the goods a household starts with do
// not fit its room, which only a warm start or a start stock the validator did not
// see can cause here.
//
// Requirement: FR-STO-045
func TestConstruction_StartGoodsAboveRoom(t *testing.T) {
	t.Parallel()
	c := shipConfig()
	c.Roles[0].Strategy = DefaultRole().Strategy
	c.Modules.Storage = &Storage{Base: &Capacity{Mass: 100, Bulk: 100}}
	c.WarmStart = &WarmStart{Groups: []WarmGroup{{Region: "a_home", Role: "miller", Stock: []Item{{Commodity: "grain", Quantity: 500}}}}}
	if _, err := NewWorld(c); !errors.Is(err, ErrRoomShape) || errors.Is(err, ErrNoRoom) {
		t.Errorf("NewWorld(warm stock of 500 grain over a room of (100, 100)) = %v, want ErrRoomShape and not ErrNoRoom", err)
	}
}

// TestAudit_StorageNamesAWrongPledge verifies that the audit of the module finds a
// pledge that nothing explains and names the agent.
//
// Requirement: FR-STO-040
func TestAudit_StorageNamesAWrongPledge(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	if err := w.Audit(); err != nil {
		t.Fatalf("Audit() on a fresh world = %v, want nil", err)
	}
	if err := w.ledger.SetPledge(w.agents[shipHome].account, 5, 5); err != nil {
		t.Fatalf("SetPledge() = %v, want nil", err)
	}
	err := w.Audit()
	if !errors.Is(err, ErrAuditFailed) {
		t.Fatalf("Audit() with a pledge nothing explains = %v, want ErrAuditFailed", err)
	}
}

// TestStoredLoad_OfAnAccount verifies the stored load read of an agent: the part of
// the load above the base room, up to the capacity of the stores.
//
// Arithmetic: base room (60, 40) and a barn (100, 120): a load of (130, 90) is above
// the base by (70, 50), within the barn: (70, 50); a load of (30, 20) is below the
// base room: (0, 0).
//
// Requirement: FR-STO-042
func TestStoredLoad_OfAnAccount(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	stoAdd(t, w, shipHome, 0)
	a := &w.agents[shipHome]
	if got := w.storedLoad(a); got.Mass != 0 || got.Bulk != 0 {
		t.Errorf("storedLoad at load (30, 20) = %+v, want (0, 0)", got)
	}
	stoGrain(t, w, shipHome, 33) // 43 grain: load (129, 86)
	if got := w.storedLoad(a); got.Mass != 69 || got.Bulk != 46 {
		t.Errorf("storedLoad at load (129, 86) = %+v, want (69, 46)", got)
	}
}
