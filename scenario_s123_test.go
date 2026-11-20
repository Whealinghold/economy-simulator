package economysim_test

import (
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// capDrainTicks is the length of S123: the drain lasts 200 ticks (Durability 200, Decay 1), so
// 300 ticks show the capacity raised, then back at the facility-less value.
const capDrainTicks = 300

// capDrainLayer is the extra configuration of S123: a ground class on the arable resource of
// overlay/resources-binding.toml and one initial drain on the north's land, held by the estate.
// The numbers are placeholders (N-018, N-027).
const capDrainLayer = `
[modules.infrastructure.class.drain]
kind = "ground"
durability = 200
decay = 1
boost_bp = 3000
resource = "arable"

[[modules.infrastructure.initial]]
region = "north"
class = "drain"
count = 1
`

// arable is the endowment row of the north's arable land of a snapshot.
func arable(t *testing.T, cfg economysim.Config, s economysim.Snapshot) economysim.EndowmentRow {
	t.Helper()
	i := slices.IndexFunc(cfg.Modules.Resources.Resources, func(r economysim.Resource) bool { return r.ID == "arable" })
	for _, r := range s.Endowments {
		if int(r.Resource) == i && r.Region == capRegion(cfg, "north") {
			return r
		}
	}
	t.Fatalf("no endowment row for the arable land of the north in %d rows", len(s.Endowments))
	return economysim.EndowmentRow{}
}

// drainRun runs the pack with or without the drain and records the north's arable row after
// each tick and the households at the end.
func drainRun(t *testing.T, layers []string, drain bool, seed uint64) (rows []economysim.EndowmentRow, households int64) {
	t.Helper()
	var cfg economysim.Config
	if drain {
		cfg = loadPackBytes(t, []byte(capDrainLayer), layers...)
	} else {
		cfg = loadValidPack(t, layers...)
	}
	w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit())
	for range capDrainTicks {
		capStep(t, w)
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		rows, households = append(rows, arable(t, cfg, s)), s.Households
	}
	return rows, households
}

// TestScenarioS123_Drain verifies CAP-S5 of spec-infrastructure.md 16.2: a drain on the arable
// endowment of pack one with the binding resources overlay, population on and off, seeds 1 to
// the seed count, 300 ticks, the audit on, against the same pack without the module. Exact: the
// effective capacity of the drained land is the base capacity plus the effect of the drain
// (3 000 basis points times its condition over 200, floored) while it stands, and the base capacity again after the drain is ruined (200
// ticks); the stock never exceeds the capacity at the end of a tick (the clamp of FR-CAP-025).
// Reported, not compared: the households at the end with and without the drain, with the
// population module on and off (the model's order: with it off the capacity and the draw rise
// and stay, with it on the households rise to a larger fixed point).
//
// Requirement: FR-CAP-024, FR-CAP-025
//
// Guard: a ground layer that did not reach the capacity, or a ruin that left the raised
// capacity behind, or a stock left above a lowered capacity, would each fail one assertion.
func TestScenarioS123_Drain(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	for _, v := range []struct {
		name   string
		layers []string
	}{
		{"population off", append(slices.Clone(packOne), "overlay/resources-binding.toml")},
		{"population on", packOnePopResources},
	} {
		for _, seed := range seedList(min(scenarioSeedCount(), 3)) {
			t.Run(v.name+" seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
				t.Parallel()
				base, baseHouseholds := drainRun(t, v.layers, false, seed)
				drained, drainedHouseholds := drainRun(t, v.layers, true, seed)
				c0 := base[0].Capacity
				// After the second step the drain has lost 2 of its 200 units: eff = floor(3 000 * 198 / 200).
				if want := c0 + c0*(3000*198/200)/10_000; drained[1].Capacity != want {
					t.Errorf("capacity with the drain after tick 2 = %d, want %d (base %d + 29.7%%)", drained[1].Capacity, want, c0)
				}
				if got := drained[capDrainTicks-1].Capacity; got != base[capDrainTicks-1].Capacity {
					t.Errorf("capacity after the drain is ruined = %d, want the facility-less %d", got, base[capDrainTicks-1].Capacity)
				}
				for i, r := range drained {
					if r.Stock > r.Capacity {
						t.Fatalf("tick %d: stock %d above capacity %d, want the clamp to hold it at or below", i, r.Stock, r.Capacity)
					}
				}
				t.Logf("seed %d %s: households at the end %d with the drain, %d without", seed, v.name, drainedHouseholds, baseHouseholds)
			})
		}
	}
}
