package economysim_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiscalPackDoc is a fiscal overlay for the governed warm five, written as a file
// layer: a head tax and a payout on the cooks of the seat region, a duty on the sales
// of one commodity, and a purchase and a release of it. Every authority is the root,
// which governs the seat region centre. The module has no step yet (PR 2 is the
// configuration surface), so the overlay changes the configuration and nothing else.
const fiscalPackDoc = `
[modules.fiscal.tax.poll]
authority = "root"
target = { kind = "households", region = "centre", role = "cook" }
amount = 1

[modules.fiscal.duty.sales]
authority = "root"
region = "centre"
commodity = "good_a"
payer = "seller"
rate_bp = 100

[modules.fiscal.payout.alms]
authority = "root"
target = { kind = "households", region = "centre", role = "cook" }
amount = 1

[modules.fiscal.purchase.buy]
authority = "root"
commodity = "good_a"
quantity = 1
limit_bp = 8000
hold = 10

[modules.fiscal.release.sell]
authority = "root"
commodity = "good_a"
quantity = 1
limit_bp = 12000
`

// fiscalPackConfig loads the governed warm five with the fiscal overlay and validates
// it; with off set it clears the module, so a test can compare the two.
func fiscalPackConfig(t *testing.T, off bool) economysim.Config {
	t.Helper()
	cfg := loadPackBytes(t, []byte(fiscalPackDoc), fivePolityGoverned...)
	if cfg.Modules.Fiscal == nil {
		t.Fatal("loadPackBytes(five + polity-governed + fiscal).Modules.Fiscal is nil, want the module on")
	}
	if off {
		cfg.Modules.Fiscal = nil
	}
	return cfg
}

// TestFiscal_PackModuleOnChangesNothingYet verifies, on files alone (testing.md 6.4), that
// the warm five with the governed polity and the fiscal overlay builds, runs 30 audited
// steps and ends in the state of the same pack with the module off, and that the
// configuration hash tells the two apart. The ledger declares the module's reasons and
// no step uses them, so any difference in the snapshot is a leak of the module into the
// run (FR-FIS-001: off leaves no trace; here the reverse, on does nothing yet).
//
// Requirement: FR-FIS-001, FR-FIS-002, FR-FIS-005
//
// Guard: the module is configuration only in this change. A later change that adds a
// step moves this test on purpose; one that moves it by accident has shifted an agent
// ID or drawn a word of the stream.
func TestFiscal_PackModuleOnChangesNothingYet(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	const seed, ticks = 1, 30
	on, off := fiscalPackConfig(t, false), fiscalPackConfig(t, true)
	if economysim.HashConfig(on) == economysim.HashConfig(off) {
		t.Error("HashConfig(module on) = HashConfig(module off), want different: the module is part of the configuration")
	}
	ron := runPack(t, on, runSpec{Seed: seed, Ticks: ticks, Audit: true, Drive: driveStep, Snapshots: snapNone})
	roff := runPack(t, off, runSpec{Seed: seed, Ticks: ticks, Audit: true, Drive: driveStep, Snapshots: snapNone})
	// The snapshot carries the configuration hash, which differs by design.
	ron.Final.ConfigHash, roff.Final.ConfigHash = [32]byte{}, [32]byte{}
	if !reflect.DeepEqual(ron.Final, roff.Final) {
		t.Errorf("final snapshot after %d ticks (seed %d) with the module on differs from the module off, want equal", ticks, seed)
	}
}

// TestFiscal_PackCheckpointRoundTrip verifies that a world of the pack with the module
// on, checkpointed after 10 steps and restored under the same configuration, steps on
// to the same snapshot as the world that was not interrupted, so the module's ledger
// reasons survive a save by position and the restore needs no section of the module.
//
// Requirement: FR-FIS-002, FR-CHK-001
//
// Guard: a restore that mapped the fiscal flow reasons to other positions would pass the
// audit and diverge here; a section tag claimed by a later change must not make a
// checkpoint of a module-on world unreadable.
func TestFiscal_PackCheckpointRoundTrip(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := fiscalPackConfig(t, false)
	w := mustWorld(t, cfg, economysim.WithSeed(1), economysim.WithAudit())
	stepTo := func(w *economysim.World, n int) {
		t.Helper()
		for range n {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
		}
	}
	stepTo(w, 10)
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	restored, err := economysim.RestoreWorld(cfg, cp, economysim.WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld(module on) error = %v, want nil", err)
	}
	stepTo(w, 10)
	stepTo(restored, 10)
	want, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	got, err := restored.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() of the restored world error = %v, want nil", err)
	}
	if !reflect.DeepEqual(got, want) { // same configuration, so the hash matches too
		t.Error("snapshot of the restored world after 10 more steps differs from the uninterrupted world, want equal")
	}
}
