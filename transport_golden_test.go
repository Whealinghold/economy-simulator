package economysim_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// transportPosLines is the number of positions that ran per tick with the module on:
// the 22 of the off set and P1.3 and P3.1.
const transportPosLines = 24

// solveTransport solves the steady state of the module-alone pack, the programme the
// golden reports.
func solveTransport(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packTwoTransport...))
	if err != nil {
		t.Fatalf("SolveSteadyState(pack two with transport) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// goldenSpec is the run of every golden of the module: seed 1, 400 ticks, a Step
// loop with the recorder and the audit.
func goldenSpec(extra []economysim.Option) runSpec {
	return runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true, Extra: extra}
}

// checkTransportGolden runs the three goldens of one variant (name is "two" or
// "two-shipper"): the rendering of the run, the events of a Run and the per-position
// hash, against testdata/golden/transport/<name>.txt, events-<name>.txt and
// <name>.positions.txt. wantEvent is an event type the events file must hold, so that
// the golden sees the module.
func checkTransportGolden(t *testing.T, name string, cfg economysim.Config, extra []economysim.Option, wantEvents ...string) {
	t.Helper()
	res := runPack(t, cfg, goldenSpec(extra))
	got := renderGolden(name, goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveTransport(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("transport", name+".txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("%s.txt differs from the golden: %s (testing.md 7.1.2)", name, transportPositionsNote(t, name, cfg, extra))
	}
	checkModuleGolden(t, "transport", name+".txt", got)

	ev := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true, Extra: extra})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Extra: extra})
	if !reflect.DeepEqual(ev.Final, step.Final) {
		t.Errorf("%s: final snapshot of Run(400) differs from the Step loop's (seed %d)", name, goldenSeed)
	}
	events := renderEvents(ev)
	for _, w := range wantEvents {
		if !bytes.Contains(events, []byte(w)) {
			t.Errorf("%s: the events hold no %s, want the module's event in the golden", name, w)
		}
	}
	checkModuleGolden(t, "transport", "events-"+name+".txt", events)

	lines := positionLines(t, cfg, goldenSeed, posTicks, extra...)
	if want := posTicks * transportPosLines; len(lines) != want {
		t.Errorf("%s: recorded %d position lines, want %d (%d positions per tick)", name, len(lines), want, transportPosLines)
	}
	checkModuleGolden(t, "transport", name+".positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// TestGolden_TransportAlone verifies the module-alone golden of the transport module:
// pack two with the trade and the transport overlays (testing.md 7.1.1), the default
// strategies, seed 1, 400 ticks, as a Step loop with the recorder and WithAudit,
// against testdata/golden/transport/two.txt, and the events of a Run against
// events-two.txt and the per-position state hash of ticks 0 to 29 against
// two.positions.txt. No default strategy ships, so the module is on and idle: the
// golden sees the world the module leaves untouched, the boundary rows it adds and the
// two positions it runs.
//
// Requirement: FR-REF-025, FR-REF-026, FR-TRN-004, FR-TRN-044, FR-OBS-023, FR-OBS-080, FR-WLD-004
//
// Guard: a behaviour change of the module moves the files; they are regenerated with
// -update only after the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN:
// the files are new, no existing golden moves.
func TestGolden_TransportAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	checkTransportGolden(t, "two", loadValidPack(t, packTwoTransport...), nil)
}

// TestGolden_TransportShipper verifies the same goldens with a scripted shipper
// (shipper above) that sends cargo on foot between the two regions, so the files hold
// the dispatch, the journey, the arrival and the relocation of the travellers:
// testdata/golden/transport/two-shipper.txt, events-two-shipper.txt and
// two-shipper.positions.txt.
//
// Requirement: FR-TRN-024, FR-TRN-035, FR-TRN-044, FR-OBS-080, FR-WLD-004
//
// Guard: without a shipment the module-alone golden would not see the arrival path;
// the events file must hold ShipmentDispatched and ShipmentArrived. DELIBERATE NEW
// GOLDEN: the files are new, no existing golden moves.
func TestGolden_TransportShipper(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, opts := shipperPack(t)
	checkTransportGolden(t, "two-shipper", cfg, opts, "ShipmentDispatched", "ShipmentArrived", "AgentRelocated")
}

// transportPositionsNote replays the scripted run against the recorded positions file
// of the variant, so a failing golden says where the state first moved.
func transportPositionsNote(t testing.TB, name string, cfg economysim.Config, extra []economysim.Option) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("transport", name+".positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks, extra...)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with %s.positions.txt: the divergence is later", posTicks-1, name)
}
