package economysim_test

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The module-alone goldens of the insurance module (plan 3-ins phase 8, testing.md 7.1.1): two sets,
// one for the route kind and one for the store kind, both recorded once and never regenerated to
// absorb a failure. The packs are the ones the plan names (Q9): pack two with the trade, transport,
// risk, credit and cover overlays, driven by the scripted shipper of the transport goldens (no
// default strategy ships, so without it no shipment is dispatched and no lane is asked about), and
// pack one with the storage, risk-store, credit and cover-store overlays.

// insTwoPosLines and insStorePosLines are the positions that ran per tick: the 22 of the off set,
// P1.3 and P3.1 of the transport module (two only) or P3.6a of the storage module (store only), P3.1a and
// P8.1a of the risk module, P1.7a and P3.3b of the credit module, and P1.3a, P3.3c and P8.1b of the
// insurance module (P8.1b runs in both sets: a store policy of the tick ends there, spec-insurance.md 7.1.1).
const (
	insTwoPosLines   = 31
	insStorePosLines = 30
)

// packTwoCover is pack two with the trade, transport, risk, credit and cover overlays (overlay/cover.toml
// needs the lanes of the first three and the claims of the fourth).
var packTwoCover = append(slices.Clone(packTwoRisk), "overlay/credit.toml", "overlay/cover.toml")

// packOneCoverStore is pack one with the storage, risk-store, credit and cover-store overlays.
var packOneCoverStore = append(slices.Clone(packOneRiskStore), "overlay/credit.toml", "overlay/cover-store.toml")

// coverShipper is the scripted shipper with the cover of the default strategy asked after its
// shipments: the default Trade has already run DefaultCover on a plan that had no shipment yet, so
// the cover it made is dropped and made again over the plan with the shipment.
type coverShipper struct{ shipper }

func (s coverShipper) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	s.shipper.Trade(v, rng, p)
	p.Cover, p.Store, p.Offer = p.Cover[:0], p.Store[:0], p.Offer[:0]
	economysim.DefaultCover(v, p)
}

// insGolden is one set of the module-alone goldens: its name inside testdata/golden/insurance, the
// pack, the options of its run and the events its events file must hold.
type insGolden struct {
	name   string
	cfg    economysim.Config
	opts   []economysim.Option
	posN   int
	solve  solveReport
	events []string
}

// insGoldenTwo returns the set of the route kind: pack two, the shipper and the cover of its lanes.
func insGoldenTwo(t testing.TB) insGolden {
	t.Helper()
	cfg := loadValidPack(t, packTwoCover...)
	cfg, s := shipperFor(t, cfg)
	return insGolden{name: "two", cfg: cfg, opts: []economysim.Option{economysim.WithStrategy("ship", coverShipper{s})}, posN: insTwoPosLines,
		solve: solveTransport(t), events: []string{"CoverFormed", "CoverEnded", "CoverSettled"}}
}

// insGoldenStore returns the set of the store kind: pack one, the barns, the store row and the cover.
func insGoldenStore(t testing.TB) insGolden {
	t.Helper()
	return insGolden{name: "one-store", cfg: loadValidPack(t, packOneCoverStore...), posN: insStorePosLines, solve: solveCold(t),
		events: []string{"CoverFormed", "CoverEnded", "CoverSettled"}}
}

// check runs the three goldens of the set: the rendering of a Step run with the recorder and the
// audit, the events of a Run (which must end where the Step loop ends and must hold the events the
// set exists for) and the per-position hash of ticks 0 to 29, against testdata/golden/insurance.
func (g insGolden) check(t *testing.T) {
	t.Helper()
	res := runPack(t, g.cfg, goldenSpec(g.opts))
	got := renderGolden(g.name, goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), g.solve, g.cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("insurance", g.name+".txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("%s.txt differs from the golden: %s (testing.md 7.1.2)", g.name, g.positionsNote(t))
	}
	checkModuleGolden(t, "insurance", g.name+".txt", got)

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
	checkModuleGolden(t, "insurance", "events-"+g.name+".txt", events)

	lines := positionLines(t, g.cfg, goldenSeed, posTicks, g.opts...)
	if want := posTicks * g.posN; len(lines) != want {
		t.Errorf("%s: recorded %d position lines, want %d (%d positions per tick)", g.name, len(lines), want, g.posN)
	}
	checkModuleGolden(t, "insurance", g.name+".positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// positionsNote replays the scripted run against the recorded positions file of the set, so a
// failing golden says where the state first moved.
func (g insGolden) positionsNote(t testing.TB) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("insurance", g.name+".positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, g.cfg, goldenSeed, posTicks, g.opts...)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with %s.positions.txt: the divergence is later", posTicks-1, g.name)
}

// TestGolden_InsuranceAloneRoutes verifies the module-alone golden of the route kind: pack two with
// the trade, transport, risk, credit and cover overlays (testing.md 7.1.1), the scripted shipper
// asking for cover of its lanes, seed 1, 400 ticks, a Step loop with the recorder and WithAudit,
// against testdata/golden/insurance/two.txt, the events of a Run against events-two.txt and the
// per-position state hash of ticks 0 to 29 against two.positions.txt. The events hold the policies
// formed, ended and settled.
//
// Requirement: FR-REF-025, FR-REF-026, FR-INS-001, FR-INS-030, FR-INS-040, FR-OBS-080, FR-WLD-004
//
// Guard: a behaviour change of the module moves the files; they are regenerated with -update
// only after the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN: the files are new,
// no existing golden moves.
func TestGolden_InsuranceAloneRoutes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	insGoldenTwo(t).check(t)
}

// TestGolden_InsuranceAloneStores verifies the module-alone golden of the store kind: pack one with
// the storage, risk-store, credit and cover-store overlays, seed 1, 400 ticks, against
// testdata/golden/insurance/one-store.txt, events-one-store.txt and one-store.positions.txt. The
// events hold the store policies formed and settled at P8.1b.
//
// Requirement: FR-REF-025, FR-REF-026, FR-INS-001, FR-INS-031, FR-INS-040, FR-OBS-080, FR-WLD-004
//
// Guard: a behaviour change of the module moves the files; they are regenerated with -update
// only after the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN: the files are new,
// no existing golden moves.
func TestGolden_InsuranceAloneStores(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	insGoldenStore(t).check(t)
}

// insTally is what a run of events shows of the module.
type insTally struct {
	formed, routeFormed, storeFormed int64
	ended, paid, shortfalls          int64 // policies ended, with a payout above 0, and with a shortfall
	due                              int64
}

// add tallies the cover events of a run.
func (r *insTally) add(events []economysim.Event) {
	for _, e := range events {
		switch e := e.(type) {
		case economysim.CoverFormed:
			r.formed++
			if e.Kind == economysim.CoverRoute {
				r.routeFormed++
			} else {
				r.storeFormed++
			}
		case economysim.CoverEnded:
			r.ended++
			if e.Paid > 0 {
				r.paid++
				r.due += int64(e.Due)
			}
			if e.Short > 0 {
				r.shortfalls++
			}
		}
	}
}

// TestGolden_InsuranceProbeExercisesPayouts verifies that the packs of the two module-alone goldens
// are not empty for the module: over seeds 1 to 4 and 400 ticks, the route pack forms a route
// policy and ends one with a payout above 0, and the store pack forms a store policy and ends one
// with a payout above 0. Packs one, two and three collapse (N-352), so a golden may end before a
// strike matters; the probe fails the golden's pack, not the golden, and the plan's fallback is
// pack five with the trade and transport overlays of five.
//
// Requirement: FR-REF-025, FR-INS-030, FR-INS-040
//
// Guard: an overlay whose numbers drift so that no policy forms or none pays would leave a golden
// that sees nothing of the module and still passes.
func TestGolden_InsuranceProbeExercisesPayouts(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	two, store := insGoldenTwo(t), insGoldenStore(t)
	for _, seed := range seedList(4) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			var tl insTally
			res := runPack(t, two.cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Extra: two.opts})
			tl.add(res.Events)
			if tl.routeFormed == 0 || tl.paid == 0 {
				t.Errorf("route pack, seed %d: %d route policies formed and %d paid in %d ticks, want both above 0", seed, tl.routeFormed, tl.paid, goldenTicks)
			}
			tl = insTally{}
			res = runPack(t, store.cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true})
			tl.add(res.Events)
			if tl.storeFormed == 0 || tl.paid == 0 {
				t.Errorf("store pack, seed %d: %d store policies formed and %d paid in %d ticks, want both above 0", seed, tl.storeFormed, tl.paid, goldenTicks)
			}
		})
	}
}
