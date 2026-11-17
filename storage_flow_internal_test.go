package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// The cost of a store is a money flow of the target kind stores (spec-storage.md 7,
// spec-money-supply.md 4.3.4). The world of these tests is the miller chain of the
// money tests with four households of weight 1 and the storage module on, two store
// classes (bin, shed), and a flow of the kind levy to the owners.

// storesFlow is a levy of amount per tick on the stores of a class ("" for every
// class).
func storesFlow(id, kind string, amount int64, class string) MoneyFlow {
	return MoneyFlow{ID: id, Kind: kind, Amount: amount, Target: FlowTarget{Kind: "stores", Store: class}}
}

// storesWorld builds the world with the flows and gives agent ids the stores in the
// order listed (a class index each), the audit on.
func storesWorld(t *testing.T, owned map[AgentID][]StoreClassID, flows ...MoneyFlow) (*World, *obsRec) {
	t.Helper()
	c := millConfig(1, 4)
	c.Roles[0].StartMoney = 1_000
	c.Modules.MoneySupply = &MoneySupply{Flows: flows}
	c.Modules.Storage = &Storage{
		Base:    &Capacity{Mass: 100, Bulk: 100},
		Classes: []StoreClass{{ID: "bin", Mass: 10, Bulk: 10}, {ID: "shed", Mass: 20, Bulk: 20}},
	}
	rec := &obsRec{}
	w := newMill(t, c, WithSeed(1), WithAudit(), WithObserver(rec))
	for id, classes := range owned {
		for _, k := range classes {
			stoAdd(t, w, id, k)
		}
	}
	return w, rec
}

// TestStoresFlow_AmountPerStore verifies that the amount of a stores levy is per store
// and not per household: a household with 2 bins is due twice the amount, one with a
// bin and a shed is due once for the class bin and twice for every class, one with
// none is due nothing, and the carry is per account. The first world levies the bin
// at 10 a tick, the second every class.
//
// Requirement: FR-STO-026, FR-MON-008
func TestStoresFlow_AmountPerStore(t *testing.T) {
	t.Parallel()
	owned := map[AgentID][]StoreClassID{1: {0, 0}, 2: {0, 1}}
	tests := []struct {
		name  string
		class string
		paid  map[AgentID]int64
	}{
		{"one class", "bin", map[AgentID]int64{1: 20, 2: 10, 3: 0, 4: 0}},
		{"every class", "", map[AgentID]int64{1: 20, 2: 20, 3: 0, 4: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := storesWorld(t, owned, storesFlow("upkeep", FlowLevy, 10, tt.class))
			before := map[AgentID]int64{}
			for id := range tt.paid {
				before[id] = holds(t, w, id)
			}
			w.p17Flows()
			for id, want := range tt.paid {
				if got := before[id] - holds(t, w, id); got != want {
					t.Errorf("agent %d paid %d, want %d", id, got, want)
				}
			}
		})
	}
}

// TestStoresFlow_ExemptIsPerAccount verifies that Exempt is an amount per account,
// not per store: an owner of 2 bins with 1 000 and an Exempt of 995 may be charged
// the 5 above it, where a count of 2 would leave nothing; and an owner at or below
// the Exempt is charged nothing.
//
// Requirement: FR-STO-026, FR-MON-008
func TestStoresFlow_ExemptIsPerAccount(t *testing.T) {
	t.Parallel()
	f := storesFlow("upkeep", FlowLevy, 10, "bin")
	f.Exempt = 995
	w, _ := storesWorld(t, map[AgentID][]StoreClassID{1: {0, 0}}, f)
	w.p17Flows()
	if got := 1_000 - holds(t, w, 1); got != 5 {
		t.Errorf("owner of 2 bins above an Exempt of 995 paid %d, want 5, the balance above the Exempt", got)
	}
	f.Exempt = 1_000
	w, _ = storesWorld(t, map[AgentID][]StoreClassID{1: {0, 0}}, f)
	w.p17Flows()
	if got := holds(t, w, 1); got != 1_000 {
		t.Errorf("owner at its Exempt holds %d, want 1000, nothing charged", got)
	}
}

// TestStoresFlow_MembershipIsReadAtP17 verifies that a store added or removed counts
// from the next P1.7, and that the grant form pays per store too.
//
// Requirement: FR-STO-026, FR-MON-008
func TestStoresFlow_MembershipIsReadAtP17(t *testing.T) {
	t.Parallel()
	w, _ := storesWorld(t, nil, storesFlow("aid", FlowGrant, 7, "shed"))
	w.p17Flows()
	if got := holds(t, w, 1); got != 1_000 {
		t.Fatalf("agent 1 holds %d with no store, want 1000", got)
	}
	stoAdd(t, w, 1, 1)
	stoAdd(t, w, 1, 1)
	w.p17Flows()
	if got := holds(t, w, 1); got != 1_014 {
		t.Errorf("agent 1 holds %d after a grant of 7 per shed on 2 sheds, want 1014", got)
	}
	if err := w.RemoveStore(1, 1); err != nil {
		t.Fatalf("RemoveStore() = %v, want nil", err)
	}
	w.p17Flows()
	if got := holds(t, w, 1); got != 1_021 {
		t.Errorf("agent 1 holds %d after a store was removed, want 1021: one shed pays 7", got)
	}
}

// TestStoresFlow_NoNewReason verifies FR-STO-027: the boundary accounts of a world
// with a stores levy are those of the same world with a household levy under the same
// flow ID, so the module added no reason, faucet or sink.
//
// Requirement: FR-STO-027
func TestStoresFlow_NoNewReason(t *testing.T) {
	t.Parallel()
	reasons := func(w *World) []Reason {
		var out []Reason
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		for _, b := range snap.Boundary {
			out = append(out, b.Reason)
		}
		return out
	}
	w, _ := storesWorld(t, nil, storesFlow("upkeep", FlowLevy, 10, "bin"))
	c := millConfig(1, 4)
	c.Roles[0].StartMoney = 1_000
	c.Modules.MoneySupply = &MoneySupply{Flows: []MoneyFlow{millerFlow("upkeep", FlowLevy, 10)}}
	ref := newMill(t, c, WithSeed(1), WithAudit())
	if got, want := reasons(w), reasons(ref); !slices.Equal(got, want) {
		t.Errorf("boundary reasons with a stores levy = %v, want %v, those of a household levy", got, want)
	}
}

// TestStoresFlow_CarryRestored verifies tag 13 for a stores target: the carries of the
// owners are saved as (account, carry) pairs, as for households, in ascending account
// ID, and loading the section back gives the same carry table, so the flow pays the
// same afterwards. A fractional per-year amount makes the carry non-zero. (A restore of
// a whole world with the module on waits for the stores section, Phase 8.)
//
// Requirement: FR-STO-026, FR-MON-040
func TestStoresFlow_CarryRestored(t *testing.T) {
	t.Parallel()
	f := storesFlow("upkeep", FlowLevy, 1_000, "bin")
	f.Per = PerYear
	w, _ := storesWorld(t, map[AgentID][]StoreClassID{1: {0, 0}, 2: {0}}, f)
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	fs := &w.money.flows[0]
	want := slices.Clone(fs.carry)
	started := [4]bool{want[1] != carryUnset, want[2] != carryUnset, want[3] != carryUnset, want[4] != carryUnset}
	if started != [4]bool{true, true, false, false} {
		t.Fatalf("carries after 3 ticks = %v, want the two owners started and the other households not", want)
	}
	cw := codec.NewWriter(nil)
	if err := w.saveMoneySupply(cw); err != nil {
		t.Fatalf("saveMoneySupply() = %v, want nil", err)
	}
	fs.carry = slices.Repeat([]int64{carryUnset}, len(want))
	r := codec.NewReader(cw.Out())
	if err := w.loadMoneySupply(nil, r); err != nil {
		t.Fatalf("loadMoneySupply() = %v, want nil", err)
	}
	if r.Err() != nil {
		t.Fatalf("reading tag 13 = %v, want nil", r.Err())
	}
	if !slices.Equal(fs.carry, want) {
		t.Errorf("loaded carries = %v, want %v", fs.carry, want)
	}
}

// TestStoresFlow_Accepted verifies that a stores target that meets every rule is
// accepted now that the step is built, and that the shape rules stay: Level, RateBP and
// a role are ErrFlowShape (the failing cases are in TestValidateStorage_Errors).
//
// Requirement: FR-STO-026
func TestStoresFlow_Accepted(t *testing.T) {
	t.Parallel()
	c := millConfig(1, 4)
	c.Modules.Storage = &Storage{Base: &Capacity{Mass: 100, Bulk: 100}, Classes: []StoreClass{{ID: "bin", Mass: 10, Bulk: 10}}}
	c.Modules.MoneySupply = &MoneySupply{Flows: []MoneyFlow{storesFlow("upkeep", FlowLevy, 10, "bin")}}
	if err := c.Validate(); err != nil {
		t.Errorf("Config.Validate(a stores levy) = %v, want nil", err)
	}
}

// TestStoresTarget_UnresolvedClassIsAnInvariant verifies that a stores target whose
// class is not in the configuration stops the world's construction with the invariant
// panic instead of reading as "every class" (the index -1 of a failed lookup).
//
// Guard: the validator rejects such a class (ErrUnknownStoreClass), so only a broken
// caller reaches it; without the panic a levy meant for one class would be charged on
// every store silently.
func TestStoresTarget_UnresolvedClassIsAnInvariant(t *testing.T) {
	t.Parallel()
	c := &Config{Modules: Modules{Storage: &Storage{Classes: []StoreClass{{ID: "bin"}}}}}
	for _, tt := range []struct{ name, class string }{{"unknown class", "shed"}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if msg := invariantMessage(func() { storesTarget(c, FlowTarget{Kind: "stores", Store: tt.class}) }); msg == "" {
				t.Errorf("storesTarget(class %q) did not panic, want the invariant panic", tt.class)
			}
		})
	}
	if stores, class := storesTarget(c, FlowTarget{Kind: "stores", Store: "bin"}); !stores || class != 0 {
		t.Errorf("storesTarget(bin) = %v, %d, want true, 0", stores, class)
	}
	if stores, class := storesTarget(c, FlowTarget{Kind: "stores"}); !stores || class != -1 {
		t.Errorf("storesTarget(every class) = %v, %d, want true, -1", stores, class)
	}
}

// TestStoresTarget_Branches verifies the reading of a flow target for newFlowState: a
// target that is not stores is not a stores flow (and carries the zero class, so the
// hashed state of the flow is as it was before the module), a stores target of a class
// names the index of the class, and a stores target that names a class with the module
// off stops the construction with the invariant panic, as validation (ErrModuleDependency)
// makes it unreachable.
//
// Requirement: FR-STO-026
//
// Guard: the class -1 means every class and 0 means the first; a target that read the
// wrong one would charge a levy on the wrong stores with no error.
func TestStoresTarget_Branches(t *testing.T) {
	t.Parallel()
	on := &Config{Modules: Modules{Storage: &Storage{Classes: []StoreClass{{ID: "bin"}, {ID: "shed"}}}}}
	off := &Config{}
	tests := []struct {
		name       string
		c          *Config
		target     FlowTarget
		wantStores bool
		wantClass  int
	}{
		{"households are not stores", on, FlowTarget{Kind: "households", Role: "miller"}, false, 0},
		{"the second class", on, FlowTarget{Kind: "stores", Store: "shed"}, true, 1},
		{"every class", on, FlowTarget{Kind: "stores"}, true, -1},
		{"every class with the module off", off, FlowTarget{Kind: "stores"}, true, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if stores, class := storesTarget(tt.c, tt.target); stores != tt.wantStores || class != tt.wantClass {
				t.Errorf("storesTarget(%+v) = %v, %d, want %v, %d", tt.target, stores, class, tt.wantStores, tt.wantClass)
			}
		})
	}
	if msg := invariantMessage(func() { storesTarget(off, FlowTarget{Kind: "stores", Store: "bin"}) }); msg == "" {
		t.Errorf("storesTarget(a class, module off) did not panic, want the invariant panic")
	}
}
