package economysim_test

import (
	"maps"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// policyNeutralTicks is the length of S159's runs and policyNeutralEvery the ticks between two compared snapshots.
const (
	policyNeutralTicks = 200
	policyNeutralEvery = 50
)

// policyNeutral is what S159 compares of one world: its snapshots at the sample ticks with the module's
// own fields cleared, and the number of events of every type other than the module's.
type policyNeutral struct {
	samples []economysim.Snapshot
	others  map[string]int
	decided []economysim.PolicyDecided
}

// runPolicyNeutral runs cfg for seed and samples it every policyNeutralEvery ticks.
func runPolicyNeutral(t testing.TB, cfg economysim.Config, seed uint64) policyNeutral {
	t.Helper()
	w, k := policyWorld(t, cfg, seed)
	var out policyNeutral
	sample := func() {
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		s.Actors, s.Modules, s.ConfigHash = nil, nil, [32]byte{} // what the module adds to a snapshot is not state of the world
		out.samples = append(out.samples, s)
	}
	for w.Tick() < policyNeutralTicks {
		policyStepTo(t, w, min(w.Tick()+policyNeutralEvery, policyNeutralTicks), nil)
		sample()
	}
	out.others, out.decided = maps.Clone(k.others), k.decided
	delete(out.others, "economysim.WorldCreated") // the construction event names the modules
	return out
}

// restBand is cfg with council_a on hold and council_b a rule whose rest band covers every reading:
// target 5 000, deadband 5 000 of 10 000 basis points.
func restBand(cfg economysim.Config) economysim.Config {
	c := holdAll(cfg)
	m := *c.Modules.Policy
	m.Actors = slices.Clone(m.Actors)
	b := &m.Actors[1]
	b.Policy, b.ReadsRegions = "rule", true
	b.Rules = []economysim.Reaction{{Lever: "lane", Gauge: economysim.GaugeUnfedShare, Target: 5_000, Deadband: 5_000, Span: 500, Move: 2_000, Raise: true}}
	c.Modules.Policy = &m
	return c
}

// TestScenarioS159_NeutralityOnTheFive reports PLC-S5 of spec-policy.md 14 on the pack of the golden,
// seeds 1 to 4, 200 ticks: a world with the module on and every actor on hold, and one with a rule
// that never leaves its rest band, end with the snapshots and the event counts of the same world
// without the module (FR-PLC-001); the rule is called at every one of its decisions and holds each
// one (InBand), and no actor of either writes a slot. Exact; a harness test outside the band rule.
//
// Requirement: FR-PLC-001, FR-PLC-060, FR-PLC-061, NFR-PLC-005
//
// Guard: a module that perturbed a world it did not move, by a draw, an ID or an event, fails here.
func TestScenarioS159_NeutralityOnTheFive(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolicy...)
	off := cfg
	off.Modules.Policy = nil
	policyScenario(t, "S159 five+policy neutrality of hold and of a rule at rest, 200 ticks", min(scenarioSeedCount(), 4), func(t *testing.T, rep *scenReport, seed uint64) {
		base := runPolicyNeutral(t, off, seed)
		for _, c := range []struct {
			name string
			cfg  economysim.Config
		}{{"every actor on hold", holdAll(cfg)}, {"a rule that never leaves its band", restBand(cfg)}} {
			got := runPolicyNeutral(t, c.cfg, seed)
			checkExact(t, rep, "S159", "five", seed, c.name+": the snapshots equal the world without the module", reflect.DeepEqual(got.samples, base.samples), "a snapshot differs")
			checkExact(t, rep, "S159", "five", seed, c.name+": the other events equal in number", maps.Equal(got.others, base.others), "event counts differ")
			held := 0
			for _, d := range got.decided {
				held += int(d.InBand)
				if d.Applied != 0 || d.Entries != 0 {
					held = -1 << 30
				}
			}
			if c.name == "every actor on hold" {
				checkExact(t, rep, "S159", "five", seed, "a hold actor emits no decision", len(got.decided) == 0, "%d decisions", len(got.decided))
			} else {
				checkExact(t, rep, "S159", "five", seed, "the rule holds every decision", held > 0 && len(got.decided) == held, "%d decisions, %d InBand", len(got.decided), held)
			}
		}
	})
}
