package economysim

import (
	"errors"
	"testing"
)

// This file tests the seam the settlements module fills in the storage module
// (spec-settlements.md 6.3 H5, plan 3-set SET11 a): the gate of a store class
// (storage.gate) and StoresLocked.

// storeSeamWorld is the world of the storage room tests with the module on: the barn
// needs the flag market, which the class hamlet, where the region is, lacks.
func storeSeamWorld(t *testing.T) (*World, *evLog) {
	t.Helper()
	return stoWorld(t, func(c *Config) {
		c.Modules.Storage.Roles = append(c.Modules.Storage.Roles, RoleRoom{Role: "miller", Base: &Capacity{Mass: stoBaseMass, Bulk: stoBaseBulk}, Stores: []string{"shed", "barn", "barn"}})
		c.Modules.Storage = c.Modules.Storage.canonical()
		c.Modules.Storage.Classes[0].Needs = "market" // barn
		m := DefaultSettlements()
		m.Classes = []SettlementClass{{ID: "hamlet"}, {ID: "town", Enter: 1_000_000, Flags: []string{"market"}}}
		c.Modules.Settlements = &m
	})
}

// TestStoreClassNeeds_GatesStore verifies H5: a store class whose Needs flag the region's
// class lacks is not given at the creation of an agent, is counted in StoresLocked of its
// AgentSpawned, is refused by AddStore with ErrStoreLocked, and opens when the class
// carries the flag. Stores already owned stay.
//
// Arithmetic: the row lists shed, barn and barn; the barn is locked in a hamlet, so one
// shed is given and two stores are locked.
//
// Requirement: FR-STO-044, FR-SET-021
func TestStoreClassNeeds_GatesStore(t *testing.T) {
	t.Parallel()
	w, rec := storeSeamWorld(t)
	st := roomOfAgent(t, w, shipHome)
	if want := []StoreCount{{Class: 1, Count: 1}}; len(st.Stores) != 1 || st.Stores[0] != want[0] {
		t.Errorf("stores of a created miller in a hamlet = %+v, want %+v (the barns locked)", st.Stores, want)
	}
	var locked int64
	for _, e := range of[AgentSpawned](rec) {
		if e.Agent == shipHome {
			locked = e.StoresLocked
		}
	}
	if locked != 2 {
		t.Errorf("AgentSpawned.StoresLocked of agent %d = %d, want 2", shipHome, locked)
	}
	if err := w.AddStore(shipHome, 0); !errors.Is(err, ErrStoreLocked) {
		t.Errorf("AddStore(barn) in a hamlet = %v, want ErrStoreLocked", err)
	}
	w.set.class[regHome] = 1
	if err := w.AddStore(shipHome, 0); err != nil {
		t.Errorf("AddStore(barn) in a town = %v, want nil", err)
	}
	if err := w.AddStore(shipHome, 1); err != nil {
		t.Errorf("AddStore(shed), which has no Needs = %v, want nil", err)
	}
}

// TestStoreClassNeeds_GatesRuntimeCreation verifies that an agent created after the start
// is given the stores the class allows and its AgentSpawned counts the others, and that a
// gate with no Needs anywhere is nil.
//
// Requirement: FR-STO-044
func TestStoreClassNeeds_GatesRuntimeCreation(t *testing.T) {
	t.Parallel()
	w, rec := storeSeamWorld(t)
	rec.events = nil
	id, err := w.AddAgent(NewAgent{Region: regHome, Role: 0, Money: 10})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	var locked int64 = -1
	for _, e := range of[AgentSpawned](rec) {
		if e.Agent == id {
			locked = e.StoresLocked
		}
	}
	if locked != 2 || w.sto.count(id, 0) != 0 || w.sto.count(id, 1) != 1 {
		t.Errorf("created miller: StoresLocked %d, barns %d, sheds %d, want 2, 0, 1", locked, w.sto.count(id, 0), w.sto.count(id, 1))
	}
	open, _ := stoWorld(t, nil)
	if open.sto.gate != nil {
		t.Error("a world with no Needs has a store gate, want nil")
	}
}
