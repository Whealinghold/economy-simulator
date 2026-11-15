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
			a, b := &firstSpawn{}, &firstSpawn{}
			wa := mustWorld(t, free, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(a))
			wb := mustWorld(t, gov, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(b))
			for i := 0; i < 200; i++ {
				if err := wa.Step(); err != nil {
					t.Fatalf("seed %d: Step() of the ungoverned world error = %v, want nil", seed, err)
				}
				if err := wb.Step(); err != nil {
					t.Fatalf("seed %d: Step() of the governed world error = %v, want nil", seed, err)
				}
				checkTreasuryHeld(t, seed, wb)
			}
			if b.id != a.id+1 {
				t.Errorf("seed %d: first household ID = %d with a treasury, want %d (one more than without)", seed, b.id, a.id+1)
			}
			sa, _ := wa.Snapshot()
			sb, _ := wb.Snapshot()
			t.Logf("seed %d: households %d (ungoverned) and %d (governed), money %d and %d: reported, not compared (POL-A1)",
				seed, sa.Households, sb.Households, sa.Money, sb.Money)
		})
	}
}

// firstSpawn is an observer that keeps the ID of the first AgentSpawned only, so the
// ten parallel runs hold no event log.
type firstSpawn struct {
	id   economysim.AgentID
	seen bool
}

// Observe keeps the first spawned agent's ID.
func (f *firstSpawn) Observe(e economysim.Event) {
	if s, ok := e.(economysim.AgentSpawned); ok && !f.seen {
		f.id, f.seen = s.Agent, true
	}
}

// mustWorld builds a world of cfg or fails the test.
func mustWorld(t testing.TB, cfg economysim.Config, opts ...economysim.Option) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	return w
}

// checkTreasuryHeld checks that the treasury row and the seat's region money equal the
// StartMoney at the boundary of w.
func checkTreasuryHeld(t *testing.T, seed uint64, w *economysim.World) {
	t.Helper()
	const start, seat = 50_000, 0 // overlay/polity-governed.toml: seat "centre", the first region by ID
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
	}
	if len(s.Jurisdictions) != 1 || s.Jurisdictions[0].Money != start || s.Jurisdictions[0].Seat != seat {
		t.Fatalf("seed %d tick %d: Jurisdictions = %+v, want one row holding %d in the seat", seed, s.Tick, s.Jurisdictions, start)
	}
	if got := s.Regions[seat].MoneyTreasury; got != start {
		t.Fatalf("seed %d tick %d: MoneyTreasury = %d, want %d", seed, s.Tick, got, start)
	}
}
