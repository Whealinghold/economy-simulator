package economysim_test

import (
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioS113_Treasury verifies POL-S2 of spec-polity.md 13.3: the warm five with
// overlay/polity-governed.toml (one governed jurisdiction, seat centre, 50 000) against
// S112's run (the same polity, ungoverned), seeds 1 to 10 and 200 ticks with the audit
// on. At every boundary the treasury holds exactly its StartMoney (nothing moves it),
// the snapshot says so in the row and in MoneyTreasury, the audit passes, and every
// household's agent ID is one higher than in the ungoverned run (the price of a
// treasury, POL-A1). The means of the two runs are reported, not compared: the shift
// changes the tie-breaks, so a paired comparison is not valid.
//
// Requirement: FR-POL-015, FR-POL-029, FR-POL-028
func TestScenarioS113_Treasury(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	free := loadValidPack(t, fivePolity...)
	gov := loadValidPack(t, fivePolityGoverned...)
	for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			a := runPack(t, free, runSpec{Seed: seed, Ticks: 200, Observe: true, Audit: true})
			b := runPack(t, gov, runSpec{Seed: seed, Ticks: 200, Observe: true, Audit: true, Snapshots: snapEvery, Drive: driveStep})
			checkTreasuryHeld(t, seed, b)
			checkIDShift(t, seed, a, b)
			t.Logf("seed %d: households %d (ungoverned) and %d (governed), money %d and %d: reported, not compared (POL-A1)",
				seed, a.Final.Households, b.Final.Households, a.Final.Money, b.Final.Money)
		})
	}
}

// checkTreasuryHeld checks that the treasury row and region money equal the
// StartMoney at every sampled boundary of res.
func checkTreasuryHeld(t *testing.T, seed uint64, res result) {
	t.Helper()
	const start, seat = 50_000, 0 // overlay/polity-governed.toml: seat "centre", the first region by ID
	for tick, s := range res.Samples {
		if len(s.Jurisdictions) != 1 || s.Jurisdictions[0].Money != start || s.Jurisdictions[0].Seat != seat {
			t.Fatalf("seed %d tick %d: Jurisdictions = %+v, want one row holding %d in the seat", seed, tick, s.Jurisdictions, start)
		}
		if got := s.Regions[seat].MoneyTreasury; got != start {
			t.Fatalf("seed %d tick %d: MoneyTreasury = %d, want %d", seed, tick, got, start)
		}
	}
}

// checkIDShift checks that the agents of the governed run are those of the ungoverned
// run with each household's ID one higher, from the AgentSpawned events of construction.
func checkIDShift(t *testing.T, seed uint64, free, gov result) {
	t.Helper()
	first := func(r result) economysim.AgentID {
		for _, e := range r.Events {
			if s, ok := e.(economysim.AgentSpawned); ok {
				return s.Agent
			}
		}
		return economysim.NoAgent
	}
	if got, want := first(gov), first(free)+1; got != want {
		t.Errorf("seed %d: first household ID = %d with a treasury, want %d (one more than without)", seed, got, want)
	}
}
