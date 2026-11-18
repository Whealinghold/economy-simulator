package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// institutionsPackDoc is the institutions tables of a consumer's file, layered after pack
// one: a body of the makers of north, a produce right and a sell right of that body, and a
// price control on the staple of north. It names no rate limit, which needs the credit module.
const institutionsPackDoc = `
[modules.institutions]

[modules.institutions.body.guild]
roles = ["maker"]
regions = ["north"]
places = 60
max_trainees = 2

[modules.institutions.right.make_tools]
body = "guild"
activity = "produce"
subject = "make_tool_a"
region = "north"

[modules.institutions.right.sell_tools]
body = "guild"
activity = "sell"
subject = "tool_a"
region = "north"

[modules.institutions.control.assize]
commodity = "staple"
region = "north"
cap_bp = 12000
floor_bp = 8000
`

// institutionsWorldSeeds is the full seed count of the end-to-end test below, reduced by
// invariantSeeds under -race.
const institutionsWorldSeeds = 2

// TestConfigOnly_InstitutionsFileBuildsAndRuns verifies, as a consumer would use it, that
// a world built from files alone (pack one and a document of the institutions tables, no
// Go code of the module) loads, validates, is accepted by NewWorld and runs detTicks ticks
// under the audit; that the module is on with its rows in the world's configuration and
// in the configuration hash; that two builds of one seed agree; and that the run is the
// run of pack one without the module, snapshot for snapshot, since the module has no
// behaviour yet (plan 3-org, PR A: the readers come in the next pull requests, whose tests
// compare their results with this run).
//
// Requirement: FR-ORG-001, FR-ORG-051, FR-CFG-094
//
// Guard: every other test of this change stops at Validate or builds the Go fixture, so a
// module that loads and validates but breaks construction, the audit or the off-world
// results would fail only here.
func TestConfigOnly_InstitutionsFileBuildsAndRuns(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	off := loadValidPack(t, packOne...)
	on := loadPackBytes(t, []byte(institutionsPackDoc), packOne...)
	m := on.Modules.Institutions
	if m == nil || len(m.Bodies) != 1 || len(m.Rights) != 2 || len(m.Controls) != 1 {
		t.Fatalf("Modules.Institutions of the file = %+v, want the module on with one body, two rights and one control", m)
	}
	if economysim.HashConfig(on) == economysim.HashConfig(off) {
		t.Errorf("HashConfig(pack one with institutions) equals that of pack one alone, want them to differ")
	}
	for _, seed := range seedList(invariantSeeds(institutionsWorldSeeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			spec := detStep(runSpec{Seed: seed, Ticks: detTicks, Audit: true})
			a := runPack(t, on, spec)
			b := runPack(t, on, spec)
			base := runPack(t, off, spec)
			if got := a.World.Config().Modules.Institutions; got == nil || got.Bodies[0].ID != "guild" {
				t.Errorf("seed %d: World.Config().Modules.Institutions = %+v, want the body guild", seed, got)
			}
			if a.Final.Tick != detTicks {
				t.Errorf("seed %d: final tick = %d, want %d", seed, a.Final.Tick, detTicks)
			}
			if !reflect.DeepEqual(a.Final, b.Final) || economysim.HashWorld(a.World) != economysim.HashWorld(b.World) {
				t.Errorf("seed %d: two builds of the institutions world differ", seed)
			}
			if !reflect.DeepEqual(withoutConfigHash(a.Final), withoutConfigHash(base.Final)) {
				t.Errorf("seed %d: the final snapshot with the module on differs from the one without it, want the same apart from the configuration hash: the module has no behaviour yet", seed)
			}
			for tick, s := range base.Samples {
				if !reflect.DeepEqual(withoutConfigHash(a.Samples[tick]), withoutConfigHash(s)) {
					t.Errorf("seed %d: the snapshot after tick %d differs with the module on, want the same apart from the configuration hash", seed, tick)
				}
			}
		})
	}
}

// withoutConfigHash returns the snapshot without its configuration hash, which differs
// between a world and the same world with a module added that does nothing.
func withoutConfigHash(s economysim.Snapshot) economysim.Snapshot {
	s.ConfigHash = [32]byte{}
	return s
}
