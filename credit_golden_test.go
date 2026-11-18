package economysim_test

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packOneCredit is pack one with the credit module alone on: overlay/credit.toml (plan
// 3-crd D10), the module-alone golden's configuration (spec-reference-chain.md 4.5).
var packOneCredit = append(slices.Clone(packOne), "overlay/credit.toml")

// creditPosLines is the number of positions that ran per tick with the module on: the
// 22 of the off set plus P1.7a and P3.3b, the two positions of the module.
const creditPosLines = 24

// TestGolden_Credit verifies the module-alone golden of the credit module: pack one +
// overlay/credit.toml, seed 1, 400 ticks, run as a Step loop with the recorder and
// WithAudit (so the books and the claim identities are audited every tick), against
// testdata/golden/credit/one.txt.
//
// Requirement: FR-REF-025, FR-REF-026, FR-CRD-001, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_Credit(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneCredit...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveCold(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("credit", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", creditPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "credit", "one.txt", got)
}

// TestGolden_CreditEvents verifies the golden rendering of the events of pack one with
// the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/credit/events-one.txt, and that the Run and a bare Step loop end in
// the same snapshot. The file holds the loans formed and ended, the rounds and the
// settlements of the run.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-CRD-070, FR-CRD-071, FR-CRD-073
//
// Guard: the events of the module are the run's record of every claim; a rule of the
// round or of the settlement that changed moves the file.
func TestGolden_CreditEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneCredit...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	checkModuleGolden(t, "credit", "events-one.txt", renderEvents(res))
}

// TestPositionHash_CreditAlone verifies the per-position state hash of pack one with
// the module on, seed 1, ticks 0 to 29, against testdata/golden/credit/one.positions.txt:
// one line per position that ran, 24 per tick.
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-CRD-001
//
// Guard: when the golden fails, the first differing line names the tick and position at
// which the module's state diverged.
func TestPositionHash_CreditAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneCredit...), goldenSeed, posTicks)
	if want := posTicks * creditPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, creditPosLines)
	}
	checkModuleGolden(t, "credit", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// TestGolden_CreditOffUnmoved verifies the proof of testing.md 7.1.1 for the credit
// module: the three files of the off set of pack one are byte-identical to the ones the
// module found, the set holds no other file than before, and the hash of the all-off
// reference configuration is the pinned one.
//
// Requirement: FR-CRD-001, FR-REF-029
//
// Guard: an off file regenerated to absorb the module's change, or a hash that moved,
// fails here; a diff in golden/off is a failure of the change, not a golden to
// regenerate (architecture.md 2.10.5, N-217).
func TestGolden_CreditOffUnmoved(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]string{"one.txt": offOneTxtSha256, "events-one.txt": eventsOneSha256, "one.positions.txt": positionsOneSha256} {
		b, err := os.ReadFile(goldenPath(name))
		if err != nil {
			t.Fatalf("read off/%s: %v", name, err)
		}
		if got := sha256Hex(b); got != want {
			t.Errorf("sha256(off/%s) = %s, want %s", name, got, want)
		}
	}
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want the pinned %s", got, pinnedReferenceHash)
	}
	entries, err := os.ReadDir(moduleGoldenPath("credit", "."))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if want := []string{"events-one.txt", "one.positions.txt", "one.txt"}; !slices.Equal(names, want) {
		t.Errorf("files of golden/credit = %v, want %v", names, want)
	}
}

// creditPositionsNote replays the scripted run on cfg against the recorded positions
// file of the module, so a failing golden says where the state first moved.
func creditPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("credit", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}

// TestDeterminism_Credit verifies that a run of the credit pack is the same run however
// it is observed: for each seed, a run with a logger, an observer or the audit, and a run
// split into Run calls (the reduced case list, seeds 1 and 2), end in the snapshot and the
// state hash of the bare Step run, the events of the observed runs are equal, and a
// different seed gives a different run.
//
// Requirement: FR-CRD-062, FR-CRD-075, NFR-CRD-004
//
// Guard: the settlement and the round read state (claims, balances, scales) and draw no
// word; a logger, an observer or the audit that changed what the module counted would
// change the run, and only a run with claims open can show it.
func TestDeterminism_Credit(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneCredit...)
	cases := detCases(true)
	seeds := seedList(2)
	finals := make([]economysim.Snapshot, len(seeds))
	for i, seed := range seeds {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
			finals[i] = base.Final
			ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
			refEvents := renderEvents(withoutRunBrackets(ref))
			if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
				t.Fatalf("seed %d: observed Step run differs from the bare one at %s", seed, d)
			}
			for _, tc := range cases {
				spec := tc.spec()
				spec.Seed, spec.Ticks = seed, detTicks
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("seed %d, case %q: final snapshot differs from the bare run", seed, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("seed %d, case %q: state hash of the world differs from the bare run", seed, tc.name)
				}
				if spec.Observe {
					if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
						t.Errorf("seed %d, case %q: events differ from the reference at %s", seed, tc.name, d)
					}
				}
			}
		})
	}
	if len(seeds) > 1 && reflect.DeepEqual(finals[0], finals[1]) {
		t.Errorf("seeds %d and %d end in the same snapshot, want different runs", seeds[0], seeds[1])
	}
}
