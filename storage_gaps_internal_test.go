package economysim

import (
	"errors"
	"reflect"
	"testing"
)

// TestComplete_CutRoomIsSharedAcrossOutputLines verifies that the room a cut leaves is
// shared by the lines of one completion in commodity order: the first line takes what
// it fits, and the next line sees the room that is left, so a recipe with two outputs
// never credits more than RoomLeft in total and disposes of the rest of each line.
//
// Arithmetic: as TestComplete_AfterCutDisposesRemainderInCommodityOrder, the room left at
// the completion is (3, 2). The recipe makes 3 flour (3, 2) and 2 perish (1, 1). Flour
// is first in commodity order and fits 1 unit, which takes all of (3, 2); perish then
// has (0, 0) left and fits none. Read alone, the perish line would fit 2 units, and
// crediting them would take the load to (5, 4), above the room (36, 24).
//
// Requirement: FR-STO-013, FR-STO-007
//
// Guard: the loop over the lines already cut in fitRoom is the only thing that keeps a
// second output out of the room the first took; a recipe with a single output, which the
// other cut tests use, never reaches it.
func TestComplete_CutRoomIsSharedAcrossOutputLines(t *testing.T) {
	t.Parallel()
	w, rec := stoMillWorld(t, 36, 24, stoMiller{}, func(c *Config) {
		c.Recipes[0].Outputs[0].CostShare = 6_000
		c.Recipes[0].Outputs = append(c.Recipes[0].Outputs, Output{Commodity: "perish", Quantity: 2, CostShare: 4_000})
	})
	stoAdd(t, w, shipHome, 0)
	step(t, w, 1)
	stoGrain(t, w, shipHome, 5)
	if _, err := w.removeStore(&w.agents[shipHome], 0, CauseFacility, true); err != nil {
		t.Fatalf("removeStore(forced) = %v, want nil", err)
	}
	step(t, w, 2)
	flour, perish := commodityOf(t, w, "flour"), commodityOf(t, w, "perish")
	var got RecipeExecuted
	for _, e := range of[RecipeExecuted](rec) {
		if e.Agent == shipHome {
			got = e
		}
	}
	wantDisposed := []Amount{{Commodity: flour, Quantity: 2}, {Commodity: perish, Quantity: 2}}
	if len(got.Disposed) != len(wantDisposed) || got.Disposed[0] != wantDisposed[0] || got.Disposed[1] != wantDisposed[1] {
		t.Errorf("RecipeExecuted.Disposed of the cut completion = %+v, want %+v (2 flour, 2 perish)", got.Disposed, wantDisposed)
	}
	account := w.agents[shipHome].account
	if held, _, _ := w.ledger.Holding(account, int(flour)); held != 1 {
		t.Errorf("flour held after the completion = %d, want 1", held)
	}
	if held, _, _ := w.ledger.Holding(account, int(perish)); held != 0 {
		t.Errorf("perish held after the completion = %d, want 0: the flour took the room left", held)
	}
	if st := roomOfAgent(t, w, shipHome); st.Load.Mass > st.Room.Mass || st.Load.Bulk > st.Room.Bulk {
		t.Errorf("load %+v above room %+v after the completion", st.Load, st.Room)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestRemoveStore_UnknownClass verifies the refusal of a class that is not in the
// configuration, below 0 or past the last: ErrUnknownStoreClass, and the room group of
// the agent is unchanged.
//
// Requirement: FR-STO-033
func TestRemoveStore_UnknownClass(t *testing.T) {
	t.Parallel()
	for _, class := range []StoreClassID{-1, 2, 99} {
		w, _ := stoWorld(t, nil)
		stoAdd(t, w, shipHome, 0)
		before := roomStateAt(t, w, shipHome)
		if err := w.RemoveStore(shipHome, class); !errors.Is(err, ErrUnknownStoreClass) {
			t.Errorf("RemoveStore(%d, class %d) = %v, want ErrUnknownStoreClass", shipHome, class, err)
		}
		if after := roomStateAt(t, w, shipHome); !reflect.DeepEqual(after, before) {
			t.Errorf("a refused RemoveStore(class %d) changed the room group %+v to %+v", class, before, after)
		}
	}
}

// TestAddAgent_RoleRowStoresCountTowardsTheRoom verifies that the goods of a new agent
// are tested against the room it would have: the base room plus the stores its role's
// row gives it at creation, so a stock above the base room alone is accepted, one above
// the whole is ErrNoRoom, and the agent that is created owns the store.
//
// Arithmetic: the base room is (36, 24) and the barn (100, 120): (136, 144). Grain is
// (3, 2): 45 grain are (135, 90), within it; 46 are (138, 92), 2 mass above it.
//
// Requirement: FR-STO-019, FR-STO-034
func TestAddAgent_RoleRowStoresCountTowardsTheRoom(t *testing.T) {
	t.Parallel()
	w, _ := stoMillWorld(t, 36, 24, stoMiller{}, func(c *Config) {
		c.Modules.Storage.Roles = []RoleRoom{{Role: c.Roles[0].ID, Base: &Capacity{Mass: 36, Bulk: 24}, Stores: []string{"barn"}}}
		c.Modules.Storage = c.Modules.Storage.canonical()
	})
	grain := commodityOf(t, w, "grain")
	n := NewAgent{Region: regHome, Role: 0, Strategy: "script", Money: 5, Stock: []StockItem{{Commodity: grain, Quantity: 46}}}
	if id, err := w.AddAgent(n); !errors.Is(err, ErrNoRoom) || id != NoAgent {
		t.Errorf("AddAgent(46 grain) = %d, %v, want NoAgent and ErrNoRoom", id, err)
	}
	n.Stock[0].Quantity = 45
	id, err := w.AddAgent(n)
	if err != nil {
		t.Fatalf("AddAgent(45 grain) = %v, want nil: the barn of the role's row holds it", err)
	}
	if st := roomOfAgent(t, w, id); st.Room != (Capacity{Mass: 136, Bulk: 144}) || st.Load != (Capacity{Mass: 135, Bulk: 90}) {
		t.Errorf("room group of the new agent = %+v, want room (136, 144) and load (135, 90)", st)
	}
}

// TestWeightReduction_ReweighedJobShrinksThePledge verifies that a reduction of weight
// below the executions of a running job scales the job and its pledge down together: the
// pledge is the load of the output that is left, not of the output the job started with,
// so the room of the executions that ended is free again at once.
//
// Arithmetic: a household of weight 4 starts 4 executions of the mill (3 flour of
// (3, 2) each), a pledge of 4 * (9, 6) = (36, 24); a reduction to weight 2 leaves
// min(4, 2) = 2 executions and a pledge of (18, 12).
//
// Requirement: FR-STO-017, FR-STO-020
//
// Guard: TestWeightReduction_LoadFitsPledgeMayExceed keeps one execution below the new
// weight, where the job is not scaled and the pledge is not touched; only a job that is
// scaled shows a reduction that forgets to derive the pledge again, which the audit
// would name only at the next change of the job.
func TestWeightReduction_ReweighedJobShrinksThePledge(t *testing.T) {
	t.Parallel()
	w, _ := stoMillWorld(t, 400, 300, stoMiller{max: 4}, nil)
	a := &w.agents[shipHome]
	a.weight = 4
	w.weightChangedStorage(a)
	step(t, w, 1)
	if st := roomOfAgent(t, w, shipHome); st.Pledge != (Capacity{Mass: 36, Bulk: 24}) {
		t.Fatalf("pledge of 4 executions = %+v, want (36, 24)", st.Pledge)
	}
	if err := w.execReduce(regHome, a, Decision{Agent: shipHome, Households: 2, Cause: CauseStarved}); err != nil {
		t.Fatalf("execReduce() = %v, want nil", err)
	}
	if st := roomOfAgent(t, w, shipHome); st.Pledge != (Capacity{Mass: 18, Bulk: 12}) {
		t.Errorf("pledge after the job was scaled to 2 executions = %+v, want (18, 12)", st.Pledge)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// TestArrival_LinesShareTheRoomLeftInLineOrder verifies that the lines of one cargo are
// credited in line order from the one room that is left: the first line takes what it
// fits, the next sees what remains, the part that does not fit is diverted line by
// line, and the pledge falls by the load of both whole lines.
//
// Arithmetic: as TestArrival_DivertsWhatDoesNotFit, the room left at the arrival is
// (12, 8). The cargo is 5 grain (3, 2) and 4 hay (2, 3), a pledge of (15, 10) + (8, 12)
// = (23, 22), within the free room (106, 124) less the upkeep room (3, 2). Grain comes
// first (commodity order) and fits 4 (12 / 3 by mass, 8 / 2 by bulk), taking all of
// (12, 8); hay then fits 0.
//
// Requirement: FR-STO-015
func TestArrival_LinesShareTheRoomLeftInLineOrder(t *testing.T) {
	t.Parallel()
	w, rec := stoWorld(t, nil)
	stoAdd(t, w, shipFar, 0)
	stoGrain(t, w, shipFar, 8)
	grain, hay := commodityOf(t, w, "grain"), commodityOf(t, w, "hay")
	give(t, w, shipHome, commodityOf(t, w, "haul"), 4)
	give(t, w, shipHome, hay, 4)
	lines := []Line{{Commodity: grain, Quantity: 5}, {Commodity: hay, Quantity: 4}}
	if _, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: stoCoach, To: regFar, Consignee: shipFar, Lines: lines}); reason != "" {
		t.Fatalf("dispatch = %q, want accepted", reason)
	}
	if st := roomOfAgent(t, w, shipFar); st.Pledge != (Capacity{Mass: 23, Bulk: 22}) {
		t.Fatalf("pledge after the dispatch = %+v, want (23, 22)", st.Pledge)
	}
	if _, err := w.removeStore(&w.agents[shipFar], 0, CauseFacility, true); err != nil {
		t.Fatalf("removeStore(forced) = %v, want nil", err)
	}
	step(t, w, 3)
	arr := of[ShipmentArrived](rec)
	if len(arr) != 1 || len(arr[0].Lines) != 2 {
		t.Fatalf("ShipmentArrived = %+v, want one arrival with two lines", arr)
	}
	if l := arr[0].Lines[0]; l.Commodity != grain || l.Delivered != 4 || l.Diverted != 1 {
		t.Errorf("first line = %+v, want grain, 4 delivered, 1 diverted", l)
	}
	if l := arr[0].Lines[1]; l.Commodity != hay || l.Delivered != 0 || l.Diverted != 4 {
		t.Errorf("second line = %+v, want hay, 0 delivered, 4 diverted", l)
	}
	if st := roomOfAgent(t, w, shipFar); st.Pledge != (Capacity{}) {
		t.Errorf("pledge after the arrival = %+v, want (0, 0): the whole of both lines", st.Pledge)
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}
