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

// The module-alone goldens of the risk module (plan 3-rsk phase 8, testing.md 7.1.1): two sets,
// one for the route side and one for the store side, both recorded once and never regenerated to
// absorb a failure. The packs are the ones the plan names (Q8): pack two with the trade and
// transport overlays and overlay/risk.toml, driven by the scripted shipper of the transport
// goldens (no default strategy ships, so without it no shipment is dispatched and no row is read),
// and pack one with overlay/storage.toml and overlay/risk-store.toml.

// riskTwoPosLines and riskStorePosLines are the positions that ran per tick: the 22 of the off
// set, P1.3 and P3.1 of the transport module (two only) or P3.6a of the storage module (store
// only), and P3.1a and P8.1a of the risk module.
const (
	riskTwoPosLines   = 26
	riskStorePosLines = 25
)

// packTwoRisk is pack two with the trade, transport and risk overlays (overlay/risk.toml needs
// the edge and the modes of the first two).
var packTwoRisk = append(slices.Clone(packTwoTransport), "overlay/risk.toml")

// packOneRiskStore is pack one with the storage and the risk-store overlays.
var packOneRiskStore = append(slices.Clone(packOneStorage), "overlay/risk-store.toml")

// riskGolden is one set of the module-alone goldens: its name inside testdata/golden/risk, the
// pack, the options of its run and the events its events file must hold.
type riskGolden struct {
	name   string
	cfg    economysim.Config
	opts   []economysim.Option
	posN   int
	solve  solveReport
	events []string
}

// riskGoldenTwo returns the set of the route side: pack two, the shipper and the route rows.
func riskGoldenTwo(t testing.TB) riskGolden {
	t.Helper()
	cfg, opts := shipperPackOn(t, packTwoRisk)
	return riskGolden{name: "two", cfg: cfg, opts: opts, posN: riskTwoPosLines, solve: solveTransport(t),
		events: []string{"ShipmentDispatched", "ShipmentArrived"}}
}

// riskGoldenStore returns the set of the store side: pack one, the barns and the store row.
func riskGoldenStore(t testing.TB) riskGolden {
	t.Helper()
	return riskGolden{name: "one-store", cfg: loadValidPack(t, packOneRiskStore...), posN: riskStorePosLines, solve: solveCold(t),
		events: []string{"StoreLoss"}}
}

// check runs the three goldens of the set: the rendering of a Step run with the recorder and the
// audit, the events of a Run (which must end where the Step loop ends and must hold the events the
// set exists for) and the per-position hash of ticks 0 to 29, against testdata/golden/risk.
func (g riskGolden) check(t *testing.T) {
	t.Helper()
	res := runPack(t, g.cfg, goldenSpec(g.opts))
	got := renderGolden(g.name, goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), g.solve, g.cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("risk", g.name+".txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("%s.txt differs from the golden: %s (testing.md 7.1.2)", g.name, g.positionsNote(t))
	}
	checkModuleGolden(t, "risk", g.name+".txt", got)

	ev := runPack(t, g.cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true, Extra: g.opts})
	step := runPack(t, g.cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Extra: g.opts})
	if !reflect.DeepEqual(ev.Final, step.Final) {
		t.Errorf("%s: final snapshot of Run(400) differs from the Step loop's (seed %d)", g.name, goldenSeed)
	}
	events := renderEvents(ev)
	for _, w := range g.events {
		if !bytes.Contains(events, []byte(w)) {
			t.Errorf("%s: the events hold no %s, want the module's run in the golden", g.name, w)
		}
	}
	checkModuleGolden(t, "risk", "events-"+g.name+".txt", events)

	lines := positionLines(t, g.cfg, goldenSeed, posTicks, g.opts...)
	if want := posTicks * g.posN; len(lines) != want {
		t.Errorf("%s: recorded %d position lines, want %d (%d positions per tick)", g.name, len(lines), want, g.posN)
	}
	checkModuleGolden(t, "risk", g.name+".positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// positionsNote replays the scripted run against the recorded positions file of the set, so a
// failing golden says where the state first moved.
func (g riskGolden) positionsNote(t testing.TB) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("risk", g.name+".positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, g.cfg, goldenSeed, posTicks, g.opts...)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with %s.positions.txt: the divergence is later", posTicks-1, g.name)
}

// TestGolden_RiskAloneRoutes verifies the module-alone golden of the route side: pack two with
// the trade, transport and risk overlays (testing.md 7.1.1), the scripted shipper, seed 1, 400
// ticks, a Step loop with the recorder and WithAudit, against testdata/golden/risk/two.txt, the
// events of a Run against events-two.txt and the per-position state hash of ticks 0 to 29
// against two.positions.txt. The events hold the dispatches with their loss share and the
// arrivals with the units lost.
//
// Requirement: FR-REF-025, FR-REF-026, FR-RSK-001, FR-RSK-009, FR-RSK-011, FR-RSK-015, FR-OBS-080, FR-WLD-004
//
// Guard: a behaviour change of the module moves the files; they are regenerated with -update
// only after the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN: the files are new,
// no existing golden moves.
func TestGolden_RiskAloneRoutes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	riskGoldenTwo(t).check(t)
}

// TestGolden_RiskAloneStores verifies the module-alone golden of the store side: pack one with
// the storage and the risk-store overlays, seed 1, 400 ticks, against
// testdata/golden/risk/one-store.txt, events-one-store.txt and one-store.positions.txt. The
// events hold the StoreLoss of the barns that a strike took goods from.
//
// Requirement: FR-REF-025, FR-REF-026, FR-RSK-001, FR-RSK-018, FR-RSK-021, FR-OBS-080, FR-WLD-004
//
// Guard: a behaviour change of the module moves the files; they are regenerated with -update
// only after the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN: the files are new,
// no existing golden moves.
func TestGolden_RiskAloneStores(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	riskGoldenStore(t).check(t)
}

// strikesOf counts what a run of events shows of the module: the shipments dispatched with a
// loss share, the units the arrivals lost and the StoreLoss events.
func strikesOf(events []economysim.Event) (struck, lost, stores int64) {
	for _, e := range events {
		switch e := e.(type) {
		case economysim.ShipmentDispatched:
			if e.LossBP > 0 {
				struck++
			}
		case economysim.StoreLoss:
			stores++
		}
	}
	return struck, lostOf(events), stores
}

// TestGolden_RiskProbeExercisesStrikes verifies that the packs of the two module-alone goldens
// are not empty for the module: over seeds 1 to 4 and 400 ticks, the route pack dispatches a
// shipment with a loss share above 0 and has an arrival that lost units, and the store pack
// emits a StoreLoss. Packs one, two and three collapse (N-352), so a golden may end before a
// strike matters; the probe fails the golden's pack, not the golden, and the plan's fallback is
// pack five with the trade and transport overlays of five.
//
// Requirement: FR-REF-025, FR-RSK-009, FR-RSK-015, FR-RSK-021
//
// Guard: an overlay whose numbers drift so that no strike lands would leave a golden that sees
// nothing of the module and still passes.
func TestGolden_RiskProbeExercisesStrikes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	two, store := riskGoldenTwo(t), riskGoldenStore(t)
	for _, seed := range seedList(4) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			res := runPack(t, two.cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Extra: two.opts})
			if struck, lost, _ := strikesOf(res.Events); struck == 0 || lost == 0 {
				t.Errorf("route pack, seed %d: %d shipments struck and %d units lost in %d ticks, want both above 0", seed, struck, lost, goldenTicks)
			}
			res = runPack(t, store.cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true})
			if _, _, stores := strikesOf(res.Events); stores == 0 {
				t.Errorf("store pack, seed %d: no StoreLoss in %d ticks, want at least one", seed, goldenTicks)
			}
		})
	}
}
