package economysim_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// externalPosLines is the number of positions that ran per tick with the external
// module on over the off set: the 22 of the off set and P1.8, P3.5 and P4.2.
const externalPosLines = 25

// packExternal is the module-alone pack of the variant one: the cold pack and the
// external overlay (spec-reference-chain.md 4.5; plan 2-16 D538). The module needs the
// region graph, which the overlay's edges give (ErrModuleDependency).
func packExternal() []string { return append(slices.Clone(packLayers["one"]), "overlay/external.toml") }

// TestPacks_ExternalOverlayLoads verifies that the external overlay loads over pack one
// and over the S71 stack of pack three, validates, puts the module on with the node
// `outer` and its two lines, and that the node is a region with no group.
//
// Requirement: FR-EXT-002, FR-REF-025
//
// Guard: the golden and the scenarios read these files; a pack that stopped being valid
// would change them without a message.
func TestPacks_ExternalOverlayLoads(t *testing.T) {
	t.Parallel()
	for name, layers := range map[string][]string{"one": packExternal(), "three": packExternalThree()} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, layers...)
			m := cfg.Modules.External
			if m == nil || len(m.Nodes) != 1 || m.Nodes[0].Region != "outer" || len(m.Nodes[0].Lines) != 2 {
				t.Fatalf("pack %s: module %+v, want one node outer with two lines", name, m)
			}
			for _, g := range cfg.Groups {
				if g.Region == "outer" {
					t.Errorf("pack %s: group %+v in the node, want none", name, g)
				}
			}
		})
	}
}

// packExternalThree is the S71 stack: pack three with the trade, transport, information
// and merchants overlays and the external overlay on top (plan 2-16 D538, D539).
func packExternalThree() []string {
	return append(packMerchants("three"), "overlay/external-three.toml")
}

// solveExternal solves the steady state of the module-alone pack, the programme the
// golden reports; a pack the solver refuses reports nothing, because the open layer is
// not written (N-114).
func solveExternal(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packExternal()...))
	if err != nil {
		return solveReport{}
	}
	return newSolveReport(ss)
}

// TestGolden_ExternalAlone verifies the module-alone golden of the external module:
// pack one with the external overlay (testing.md 7.1.1), the default strategies, seed 1,
// 400 ticks, as a Step loop with the recorder and WithAudit, against
// testdata/golden/external/one.txt; the events of a Run against events-one.txt, which
// holds OutsideTraded; and the per-position state hash of ticks 0 to 29 against
// one.positions.txt, with P1.8, P3.5 and P4.2 in it. A run without the observer ends in
// the same snapshot.
//
// Requirement: FR-REF-029, FR-EXT-036, FR-OBS-023, FR-WLD-004
//
// Guard: a behaviour change of the module moves the files; they are regenerated with
// -update only after the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN:
// the files are new, no existing golden moves.
func TestGolden_ExternalAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packExternal()...)
	res := runPack(t, cfg, goldenSpec(nil))
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveExternal(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("external", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", externalPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "external", "one.txt", got)

	ev := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(ev.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) observed differs from the bare Step loop's (seed %d)", goldenSeed)
	}
	events := renderEvents(ev)
	if !bytes.Contains(events, []byte("OutsideTraded")) {
		t.Errorf("the events hold no OutsideTraded, want the module's event in the golden")
	}
	checkModuleGolden(t, "external", "events-one.txt", events)

	lines := positionLines(t, cfg, goldenSeed, posTicks)
	if want := posTicks * externalPosLines; len(lines) != want {
		t.Errorf("recorded %d position lines, want %d (%d positions per tick)", len(lines), want, externalPosLines)
	}
	for _, p := range []string{" P1.8 ", " P3.5 ", " P4.2 "} {
		if !slices.ContainsFunc(lines, func(l string) bool { return strings.Contains(l, p) }) {
			t.Errorf("the positions file has no line for %q, want the module's three positions", strings.TrimSpace(p))
		}
	}
	checkModuleGolden(t, "external", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// externalPositionsNote replays the run against the recorded positions file, so a
// failing golden says where the state first moved.
func externalPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("external", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}
