package economysim_test

import (
	"errors"
	"log/slog"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The end-to-end tests of the saving module's observability (spec-saving-and-investment.md
// 8.4, 12, FR-SAV-027, 029 to 032, 037 to 040), through the public API only: a world is built,
// stepped, observed, queried and snapshotted, and the three reports are checked against each
// other.

// savingWorld builds the world of savingConfig at seed with the options and steps it ticks
// times.
func savingWorld(t *testing.T, seed uint64, ticks int, opts ...economysim.Option) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(savingConfig(), append([]economysim.Option{economysim.WithSeed(seed)}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	step(t, w, ticks)
	return w
}

// savingAgents returns the state of every agent of w by ID.
func savingAgents(t *testing.T, w *economysim.World) []economysim.AgentState {
	t.Helper()
	var out []economysim.AgentState
	for id := economysim.AgentID(0); ; id++ {
		a, err := w.Agent(id)
		if err != nil {
			return out
		}
		out = append(out, a)
	}
}

// savingSum is the sum of the queries of one (region, role) cell.
type savingSum struct {
	households int64
	goal       economysim.Money
}

// checkPlanned fails the test for a SavingPlanned that breaks its bounds.
func checkPlanned(t *testing.T, seed uint64, ev economysim.SavingPlanned, saverRole int) {
	t.Helper()
	if int(ev.Role) != saverRole || ev.Households <= 0 {
		t.Errorf("seed %d: SavingPlanned %+v, want role %d and households above 0", seed, ev, saverRole)
	}
	if bars := ev.Passed + ev.NoReturn + ev.Standing + ev.Unreachable + ev.Survival; bars > ev.Households {
		t.Errorf("seed %d: SavingPlanned %+v has the bars summing to %d, want at most %d households", seed, ev, bars, ev.Households)
	}
	if ev.Building > ev.Households || ev.Saving > ev.Households || ev.Funded > ev.Passed || ev.Committed > ev.Passed {
		t.Errorf("seed %d: SavingPlanned %+v has a count above its bound", seed, ev)
	}
}

// querySums asks the query of every agent: a role with no block must answer ErrNotSaver, a
// saver's answers are summed by (region, role).
func querySums(t *testing.T, seed uint64, w *economysim.World, saverRole int) map[[2]int]savingSum {
	t.Helper()
	got := map[[2]int]savingSum{}
	for _, a := range savingAgents(t, w) {
		st, err := w.Saving(a.ID)
		if int(a.Role) != saverRole {
			if !errors.Is(err, economysim.ErrNotSaver) {
				t.Errorf("seed %d: Saving(%d) of a role with no block error = %v, want ErrNotSaver", seed, a.ID, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("seed %d: Saving(%d) error = %v, want nil", seed, a.ID, err)
		}
		k := [2]int{int(a.Region), int(a.Role)}
		s := got[k]
		if st.Goal > 0 {
			s.households += a.Weight
		}
		s.goal += st.Goal
		got[k] = s
	}
	return got
}

// TestSaving_ReportsAgree verifies that over seeds 1 to 3 the three reports of the module
// tell one story: every SavingPlanned names the role with a Saver block, keeps its counts
// within its households and its bars exclusive, and at the boundary the snapshot rows equal
// the sums of the per-agent queries (the households with a goal above 0, the goals) while a
// role with no block has rows of 0 and its agents answer ErrNotSaver.
//
// Requirement: FR-SAV-027, FR-SAV-029, FR-SAV-037, FR-SAV-038, FR-SAV-040
//
// Guard: a row filled from a different view than the query's, or an event that counted an
// agent of a role with no block, would break the equality at once.
func TestSaving_ReportsAgree(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	saverRole := slices.IndexFunc(savingConfig().Roles, func(r economysim.Role) bool { return r.Saver != nil })
	if saverRole < 0 {
		t.Fatal("savingConfig has no role with a Saver block")
	}
	for seed := uint64(1); seed <= 3; seed++ {
		rec := &recorder{}
		w := savingWorld(t, seed, 30, economysim.WithObserver(rec))
		planned := 0
		for _, e := range rec.events {
			if ev, ok := e.(economysim.SavingPlanned); ok {
				planned++
				checkPlanned(t, seed, ev, saverRole)
			}
		}
		if planned == 0 {
			t.Errorf("seed %d: no SavingPlanned in 30 ticks, want at least one while a saver holds a project", seed)
		}
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
		}
		got := querySums(t, seed, w, saverRole)
		for _, r := range snap.Roles {
			want := got[[2]int{int(r.Region), int(r.Role)}]
			if r.SavingHouseholds != want.households || r.GoalMoney != want.goal {
				t.Errorf("seed %d: row (%d,%d) saving %d goal %d, want the sum of the queries %d and %d",
					seed, r.Region, r.Role, r.SavingHouseholds, r.GoalMoney, want.households, want.goal)
			}
			if int(r.Role) != saverRole && (r.BufferMoney != 0 || r.IdleMoney != 0) {
				t.Errorf("seed %d: row (%d,%d) of a role with no block has buffer %d idle %d, want 0", seed, r.Region, r.Role, r.BufferMoney, r.IdleMoney)
			}
		}
	}
}

// TestDeterminism_SavingNonInterference verifies that an observer and a Debug logger change
// nothing the module does: the same seed gives the same snapshot and the same agents' queries
// with and without them, and the observer's SavingPlanned events repeat from run to run.
//
// Requirement: FR-SAV-031, FR-SAV-039, FR-SAV-040
func TestDeterminism_SavingNonInterference(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	const seed, ticks = 2, 25
	bare := savingWorld(t, seed, ticks)
	rec1, rec2 := &recorder{}, &recorder{}
	lg := slog.New(newCaptureHandler())
	obs := savingWorld(t, seed, ticks, economysim.WithObserver(rec1), economysim.WithLogger(lg))
	savingWorld(t, seed, ticks, economysim.WithObserver(rec2))
	sb, errB := bare.Snapshot()
	so, errO := obs.Snapshot()
	if errB != nil || errO != nil {
		t.Fatalf("Snapshot() = %v, %v, want nil", errB, errO)
	}
	if !reflect.DeepEqual(sb, so) {
		t.Errorf("snapshot differs with an observer and a logger attached, want identical")
	}
	for _, a := range savingAgents(t, bare) {
		x, errX := bare.Saving(a.ID)
		y, errY := obs.Saving(a.ID)
		if x != y || (errX == nil) != (errY == nil) {
			t.Errorf("Saving(%d) = %+v, %v bare and %+v, %v observed, want equal", a.ID, x, errX, y, errY)
		}
	}
	pick := func(r *recorder) []economysim.SavingPlanned {
		var out []economysim.SavingPlanned
		for _, e := range r.events {
			if ev, ok := e.(economysim.SavingPlanned); ok {
				out = append(out, ev)
			}
		}
		return out
	}
	if a, b := pick(rec1), pick(rec2); !slices.Equal(a, b) {
		t.Errorf("SavingPlanned events differ between two runs of seed %d (%d against %d), want equal", seed, len(a), len(b))
	}
}
