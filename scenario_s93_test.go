package economysim_test

import (
	"slices"
	"testing"
)

// TestScenarioS93_ProducerGlut is STO-S3 (spec-storage.md 16): pack one with the storage
// module, the growers' barn taken away so that their room is the base room just above one
// job's pledge, 400 ticks, seeds 1 to 10, audit on. A grower whose room is full cannot start
// a job (the start is cut or refused for room) until it has sold. Exact: the audit holds
// every tick and the load of every role is within its room. Recorded, not asserted (the pack
// collapses under its own rules, N-352): the starts the room cut or refused, the units the
// growers made and the households at the end, beside the same run with the barn.
//
// Requirement: FR-STO-011, FR-STO-013, FR-STO-040
//
// Guard: a producer that kept starting jobs in a full room would credit its output nowhere
// and lose it at the completion.
func TestScenarioS93_ProducerGlut(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	with := loadValidPack(t, packOneStorage...)
	without := loadValidPack(t, packOneStorage...)
	m := *without.Modules.Storage
	m.Roles = slices.Clone(m.Roles)
	for i := range m.Roles {
		if m.Roles[i].Role == "grower" {
			m.Roles[i].Stores = nil
		}
	}
	without.Modules.Storage = &m
	stoScenario(t, "one", "S93 one+storage producer glut without the barn, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		run := runPack(t, without, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapEvery})
		broke := ""
		for _, s := range run.Samples {
			for _, r := range s.Roles {
				if broke == "" && (r.LoadMass > r.RoomMass || r.LoadBulk > r.RoomBulk) {
					broke = "a role row is over its room"
				}
			}
		}
		checkExact(t, rep, "S93", "one", seed, "the load of every role is within its room after every tick", broke == "", "%s", broke)
		reportRoomCuts(rep, seed, addRoomCuts(run.Events))
		rep.num(seed, "households at end, no barn", run.Final.Households)
		rep.num(seed, "households at end, barn", runPack(t, with, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep}).Final.Households)
	})
}
