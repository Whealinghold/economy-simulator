package economysim_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// migrationPosLines is the number of positions that ran per tick with the module
// (and the information module it needs) on: the 24 of the information set plus P1.5
// and P7.2.
const migrationPosLines = 26

// packMigration is the module pack of the variant: the cold pack, its trade overlay,
// its information overlay (the module needs the information module) and the migration
// overlay (spec-reference-chain.md 4.6; plan 2-14 Phase 8).
func packMigration(variant string) []string {
	return append(packInformation(variant), "overlay/migration.toml")
}

// TestPacks_MigrationOverlayLoads verifies that the migration overlay loads over the
// variants two and three with their trade and information overlays, validates, sets a
// speed of 100 (a journey over an edge of 100 takes 1 tick) and raises none of the
// module's seven warnings except the ones the cold packs have of their own: none
// of migration_without_routes, migration_no_mobile_role, migration_provision,
// migration_far or migration_uncapped.
//
// Requirement: FR-MIG-001, FR-MIG-039, FR-REF-025
//
// Guard: the golden and the scenarios read these files; a pack that stopped being
// valid or lost a role would change them without a message.
func TestPacks_MigrationOverlayLoads(t *testing.T) {
	t.Parallel()
	for _, variant := range []string{"two", "three"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, packMigration(variant)...)
			if m := cfg.Modules.Migration; m == nil || m.MoveSpeed != 100 {
				t.Fatalf("pack %s: Modules.Migration = %+v, want the module on at a speed of 100", variant, m)
			}
			for _, w := range cfg.Warnings() {
				switch w.Rule {
				case "migration_without_routes", "migration_no_mobile_role", "migration_provision", "migration_far", "migration_uncapped":
					t.Errorf("pack %s: warning %+v, want none of the module's", variant, w)
				}
			}
		})
	}
}

// TestGolden_MigrationAlone verifies the module-alone golden of the migration module:
// pack two with the trade, information and migration overlays (testing.md 7.1.1), the
// default strategies, seed 1, 400 ticks, as a Step loop with the recorder and WithAudit,
// against testdata/golden/migration/two.txt; the events of a Run against events-two.txt,
// which holds MigrationReviewed; and the per-position state hash of ticks 0 to 29
// against two.positions.txt, with P1.5 and P7.2 in it. The households of the cold pack
// are unprovisioned for a journey or below the viability gate, so the golden holds the
// module's counts and no departure: the departures are the scenarios'. A run without
// the observer ends in the same snapshot.
//
// Requirement: FR-MIG-034, FR-MIG-042, FR-REF-025, FR-REF-026, FR-OBS-023, FR-WLD-004
//
// Guard: a behaviour change of the module moves the files; they are regenerated with
// -update only after the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN:
// the files are new, no existing golden moves.
func TestGolden_MigrationAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packMigration("two")...)
	res := runPack(t, cfg, goldenSpec(nil))
	got := renderGolden("two", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveMigration(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("migration", "two.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("two.txt differs from the golden: %s (testing.md 7.1.2)", migrationPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "migration", "two.txt", got)

	ev := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(ev.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) observed differs from the bare Step loop's (seed %d)", goldenSeed)
	}
	events := renderEvents(ev)
	if !bytes.Contains(events, []byte("MigrationReviewed")) {
		t.Errorf("the events hold no MigrationReviewed, want the module's counts in the golden")
	}
	checkModuleGolden(t, "migration", "events-two.txt", events)

	lines := positionLines(t, cfg, goldenSeed, posTicks)
	if want := posTicks * migrationPosLines; len(lines) != want {
		t.Errorf("recorded %d position lines, want %d (%d positions per tick)", len(lines), want, migrationPosLines)
	}
	for _, p := range []string{" P1.5 ", " P7.2 "} {
		if !slices.ContainsFunc(lines, func(l string) bool { return strings.Contains(l, p) }) {
			t.Errorf("the positions file has no line for %q, want the module's two positions", strings.TrimSpace(p))
		}
	}
	checkModuleGolden(t, "migration", "two.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// solveMigration solves the steady state of the module-alone pack, the programme the
// golden reports.
func solveMigration(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packMigration("two")...))
	if err != nil {
		t.Fatalf("SolveSteadyState(pack two with migration) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// migrationPositionsNote replays the scripted run against the recorded positions
// file, so a failing golden says where the state first moved.
func migrationPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("migration", "two.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with two.positions.txt: the divergence is later", posTicks-1)
}

// TestGolden_OffUnchangedByMigration verifies that the three files of the off set are
// byte for byte what they were before the migration module (the sums of
// offGoldenSums), and that the pinned hash of the reference configuration did not
// move: a module that is off leaves no trace (architecture.md 2.10.5).
//
// Requirement: FR-MIG-044, FR-REF-025
//
// Guard: a regenerated off golden would hide a non-interference failure of the module
// (testing.md 7.1.1).
func TestGolden_OffUnchangedByMigration(t *testing.T) {
	t.Parallel()
	for name, want := range offGoldenSums {
		b, err := os.ReadFile(goldenPath(name))
		if err != nil {
			t.Fatalf("read off golden %s: %v", name, err)
		}
		sum := sha256.Sum256(b)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("sha256(testdata/golden/off/%s) = %s, want %s: the off set moved (testing.md 7.1.1)", name, got, want)
		}
	}
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want the pin %s", got, pinnedReferenceHash)
	}
}
