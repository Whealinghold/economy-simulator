package economysim_test

import (
	"math/rand/v2"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// cutSpawner is the default spawner with two additions that cut the room of a household:
// every cutEvery ticks it halves the weight of every agent of weight 2 or more of the
// region (a reduction, with the cause starved), and every cutEvery ticks, half a period
// later, it switches the first agent of the region to the next role (a role change). The
// default's own plan comes first, so the additions come after its removals.
type cutSpawner struct {
	economysim.DefaultSpawner
	cutEvery int64
	roles    int
}

func (s cutSpawner) Decide(v *economysim.RegionView, rng *rand.Rand, d *economysim.Decisions) {
	s.DefaultSpawner.Decide(v, rng, d)
	switch {
	case v.Tick > 0 && v.Tick%s.cutEvery == 0:
		for _, a := range v.Agents {
			if a.Weight >= 2 && !a.External {
				d.List = append(d.List, economysim.Decision{Kind: economysim.DecideReduce, Agent: a.Agent, Households: a.Weight / 2, Cause: economysim.CauseStarved})
			}
		}
	case v.Tick > 0 && v.Tick%s.cutEvery == s.cutEvery/2 && len(v.Agents) > 0:
		a := v.Agents[0]
		d.List = append(d.List, economysim.Decision{Kind: economysim.DecideSwitch, Agent: a.Agent, Role: economysim.RoleID((int(a.Role) + 1) % s.roles)})
	}
}

// TestScenarioS96_CutOfTheRoom is STO-S6 (spec-storage.md 16, 17 part 4 as a run): pack one
// with the storage module and a spawner that, every 40 ticks, halves the weight of every
// cohort and, 20 ticks later, changes the role of the first agent of the region, 400 ticks,
// seeds 1 to 10, audit on. Exact: the audit holds every tick (so the pledge, derived and
// allowed above the room after a cut, is what the job and the cargo imply, and a credit
// never met a room that was not committed), and the load of every role is within its room
// after every tick. Recorded, not asserted (spec-storage.md open questions 11 and 14): the
// role changes that moved goods to the estate, and the completions that disposed of part
// of their output.
//
// Requirement: FR-STO-017, FR-STO-018, FR-STO-013, FR-STO-020, FR-STO-040
//
// Guard: a reduction that was refused for room, or a cut that left the load above the new
// room, would stop the ladder or break the invariant; neither may.
func TestScenarioS96_CutOfTheRoom(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packOneStorage...)
	cfg.World.Spawner = "cut"
	opts := []economysim.Option{economysim.WithSpawner("cut", cutSpawner{cutEvery: 40, roles: len(cfg.Roles)})}
	stoScenario(t, "one", "S96 one+storage cut of the room by reductions and role changes, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		res := runPack(t, cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapEvery, Extra: opts})
		broke := ""
		for _, s := range res.Samples {
			for _, r := range s.Roles {
				if broke == "" && (r.LoadMass > r.RoomMass || r.LoadBulk > r.RoomBulk) {
					broke = "a role row is over its room"
				}
			}
		}
		checkExact(t, rep, "S96", "one", seed, "the load of every role is within its room after every tick", broke == "", "%s", broke)
		var changes, listed, cutCompletions int64
		for _, e := range res.Events {
			switch e := e.(type) {
			case economysim.RoleChanged:
				changes++
				listed += e.GoodsListed
			case economysim.RecipeExecuted:
				if len(e.Disposed) > 0 {
					cutCompletions++
				}
			}
		}
		rep.num(seed, "role changes", changes)
		rep.num(seed, "goods lots moved to the estate by role changes", listed)
		rep.num(seed, "completions that disposed of part of their output", cutCompletions)
		reportRoomCuts(rep, seed, addRoomCuts(res.Events))
	})
}
