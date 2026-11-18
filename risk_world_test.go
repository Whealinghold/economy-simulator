package economysim_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of what a world does with the risk module on before the module has
// a behaviour (plan 3-rsk, PR A: phases 1 to 3 build rows, bounds, hash and
// loader, and no position reads them). Plan 3-rsk phase 4 replaces them with the
// neutrality test at probability 0 (FR-RSK-005) and the golden of the module.

// riskRun runs a world of c for ticks steps from seed and returns every event
// after WorldCreated, in order, and the final snapshot with its configuration
// hash zeroed: WorldCreated and the hash carry the module's rows, which differ by
// construction.
func riskRun(t *testing.T, c economysim.Config, seed uint64, ticks int64) ([]economysim.Event, economysim.Snapshot) {
	t.Helper()
	var events []economysim.Event
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		if _, created := e.(economysim.WorldCreated); !created {
			events = append(events, e)
		}
	})
	w, err := economysim.NewWorld(c, economysim.WithObserver(obs), economysim.WithSeed(seed))
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	if _, err := w.Run(context.Background(), ticks); err != nil {
		t.Fatalf("Run(seed %d, %d ticks) error = %v, want nil", seed, ticks, err)
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot(seed %d) error = %v, want nil", seed, err)
	}
	snap.ConfigHash = [32]byte{}
	return events, snap
}

// withoutLoss returns snap with the boundary rows of the reason loss taken out,
// and fails the test if one of them holds a balance: the module on opens those
// accounts (spec-world.md 5.2 row 3a), which a module that is off does not have,
// and a run with no strike leaves them at 0 (FR-RSK-005).
func withoutLoss(t *testing.T, snap economysim.Snapshot) economysim.Snapshot {
	t.Helper()
	var rows []economysim.BoundaryRow
	for _, b := range snap.Boundary {
		if b.Reason != economysim.ReasonLoss {
			rows = append(rows, b)
		} else if b.Balance != 0 {
			t.Errorf("boundary account loss of commodity %d holds %d, want 0 in a run with no strike", b.Commodity, b.Balance)
		}
	}
	snap.Boundary = rows
	return snap
}

// TestRisk_OnWorldBuildsAndRunsLikeOff verifies, over three seeds, that a world
// built from a configuration with the risk module on (two route rows, two store
// rows, one of each common) is accepted by NewWorld, runs, hashes differently
// from the same configuration with the module off, and emits the same events and
// ends in the same snapshot as that configuration but for the boundary accounts of loss, because no position reads the
// rows yet.
//
// Guard: PR A lands rows that nothing reads. If a later change made them read
// before the module's positions exist, or let a row leak into the core's
// draws, a configuration with the module on would change a run that the module
// is not meant to touch. The events and the snapshot are compared with
// reflect.DeepEqual, and the test fails for nothing if the run is empty, so it
// checks the run had events.
//
// Requirement: FR-RSK-001
func TestRisk_OnWorldBuildsAndRunsLikeOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	on := riskConfig()
	off := riskConfig()
	off.Modules.Risk = nil
	if economysim.HashConfig(on) == economysim.HashConfig(off) {
		t.Fatal("HashConfig(risk on) = HashConfig(risk off), want the module hashed")
	}
	for _, seed := range []uint64{1, 7, 42} {
		onEvents, onSnap := riskRun(t, on, seed, 25)
		offEvents, offSnap := riskRun(t, off, seed, 25)
		if len(onEvents) == 0 {
			t.Fatalf("seed %d: no event after WorldCreated in 25 ticks, the comparison would pass for nothing", seed)
		}
		if !reflect.DeepEqual(onEvents, offEvents) {
			t.Errorf("seed %d: %d events with the module on, %d with it off, want the same events", seed, len(onEvents), len(offEvents))
		}
		if !reflect.DeepEqual(withoutLoss(t, onSnap), offSnap) {
			t.Errorf("seed %d: snapshots differ with the module on and off, want equal", seed)
		}
	}
}

// TestRisk_OnWorldRejectsABrokenRowAtConstruction verifies that NewWorld refuses
// a configuration whose risk row breaks a rule of spec-risk.md 13.1, with the
// rule's sentinel, so a running simulation never meets a bad row (architecture.md
// 8.5).
//
// Requirement: FR-RSK-003
func TestRisk_OnWorldRejectsABrokenRowAtConstruction(t *testing.T) {
	t.Parallel()
	c := riskConfig()
	c.Modules.Risk.Routes[0].StrikeBP = 10_001
	_, err := economysim.NewWorld(c)
	if !errors.Is(err, economysim.ErrRiskShape) {
		t.Fatalf("NewWorld(strike_bp 10001) error = %v, want ErrRiskShape", err)
	}
}

// riskPackExtra is the risk module as a consumer writes it in a file layered on
// pack two with its trade and transport overlays: a unit and a common route row
// over the edges between north and south (the modes foot and coach) and a store
// row of the class barn, which the storage overlay of pack one defines.
const riskPackExtra = `
[modules.risk.route.north.south.foot]
strike_bp = 2000
severity_bp = 3000

[modules.risk.route.south.north.coach]
strike_bp = 500
severity_bp = 8000
common = true

[modules.risk.store.north.barn]
strike_bp = 300
severity_bp = 5000
`

// TestScenario_RiskFileLoadsValidatesAndRuns verifies, from the file layers
// alone and with no Go code specific to the module, that a risk table layered on
// pack two with its trade, transport and storage overlays loads, validates with
// no risk warning, builds a world, runs 40 ticks and ends where the pack without
// the table ends, on the events and the snapshot (the module reads nothing in PR
// A), while its hash differs and the rows are in the loaded configuration.
//
// This is the end-to-end path of the feature as it stands: file, loader,
// validator, hash and constructor. The behaviour at dispatch and arrival is
// phases 4 to 8 of plan 3-rsk, and they replace the comparison with the module's
// golden. A row naming a mode or class the layers lack would fail the load
// here, so the test also pins that the shipped overlays carry what a risk file
// needs.
//
// Requirement: FR-RSK-001, FR-RSK-003, FR-RSK-004, FR-CFG-089
func TestScenario_RiskFileLoadsValidatesAndRuns(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	layers := append(withTrade("two"), "overlay/transport-two.toml", "overlay/storage.toml")
	off := loadPack(t, layers...)
	on := loadPackBytes(t, []byte(riskPackExtra), layers...)
	requireRiskFile(t, on, off)
	if economysim.HashConfig(on) == economysim.HashConfig(off) {
		t.Error("HashConfig(pack with risk) = HashConfig(pack), want the rows hashed")
	}
	onEvents, onSnap := riskRun(t, on, 5, 40)
	offEvents, offSnap := riskRun(t, off, 5, 40)
	if len(onEvents) == 0 {
		t.Fatal("no event after WorldCreated in 40 ticks, the comparison would pass for nothing")
	}
	if !reflect.DeepEqual(onEvents, offEvents) || !reflect.DeepEqual(withoutLoss(t, onSnap), offSnap) {
		t.Errorf("pack with the risk file: %d events, snapshot equal %v; pack alone: %d events, want the same run", len(onEvents), reflect.DeepEqual(withoutLoss(t, onSnap), offSnap), len(offEvents))
	}
}

// requireRiskFile fails the test unless on holds the rows of riskPackExtra, off
// holds none, on validates and on raises no risk warning.
func requireRiskFile(t *testing.T, on, off economysim.Config) {
	t.Helper()
	if on.Modules.Risk == nil || len(on.Modules.Risk.Routes) != 2 || len(on.Modules.Risk.Stores) != 1 {
		t.Fatalf("loaded Modules.Risk = %+v, want two route rows and one store row", on.Modules.Risk)
	}
	if off.Modules.Risk != nil {
		t.Fatalf("the pack alone loaded Modules.Risk = %+v, want nil", off.Modules.Risk)
	}
	if err := on.Validate(); err != nil {
		t.Fatalf("Validate(pack with risk) error = %v, want nil", err)
	}
	for _, w := range on.Warnings() {
		switch w.Rule {
		case "no_risk_rows", "risk_edge_off_path", "risk_store_class_unowned", "risk_certain":
			t.Errorf("Warnings() has %s at %v, want no risk warning on a consistent file", w.Rule, w.Path)
		}
	}
}
