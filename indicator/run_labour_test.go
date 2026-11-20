package indicator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// hireRunPack is pack one-demand with the employment module in its hire form (the cooks hire,
// the diggers and the foresters are hired by the default strategy).
var hireRunPack = []string{"base.toml", "demand.toml", "one-demand.toml", "warm-one-demand.toml", "overlay/employment-hire.toml"}

// labourSums is what the LabourRound events of the ticks [from, to) add up to.
type labourSums struct{ offered, hired, noVacancy, householdTicks, paid, refunded int64 }

func sumLabour(events []economysim.Event, from, to int64) (s labourSums) {
	for _, e := range events {
		if r, ok := e.(economysim.LabourRound); ok && r.Tick >= from && r.Tick < to {
			s.offered += r.Offered
			s.hired += r.Hired
			s.noVacancy += r.NoVacancy
			s.householdTicks += r.HouseholdTicks
			s.paid += int64(r.Paid)
			s.refunded += int64(r.Refunded)
		}
	}
	return s
}

// hireWindows runs the hire pack for two windows of 100 ticks, seed 1 (the pack forms hires until
// about tick 250, N-352) and returns the world, the recorder and the windows.
func hireWindows(t *testing.T) (*economysim.World, *recorder, []runWin) {
	t.Helper()
	w, rec := newRun(t, loadPack(t, hireRunPack...), 1, true)
	return w, rec, stepWindows(t, w, rec, 100, 2, nil)
}

// labourPresets returns the three presets of a window of the hire pack.
func labourPresets(t *testing.T, w *economysim.World, rw runWin) (indicator.UnemploymentResult, indicator.WageBillResult, indicator.EmploymentResult) {
	t.Helper()
	p, err := indicator.PricesOfWindow(rw.w)
	if err != nil {
		t.Fatalf("PricesOfWindow() error = %v, want nil", err)
	}
	g, err := indicator.GDP(rw.w, rw.open, rw.closing, p)
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	u, err := indicator.Unemployment(rw.w)
	if err != nil {
		t.Fatalf("Unemployment() error = %v, want nil", err)
	}
	b, err := indicator.WageBill(rw.w, g)
	if err != nil {
		t.Fatalf("WageBill() error = %v, want nil", err)
	}
	e, err := indicator.Employment(rw.w, w.Config(), rw.closing)
	if err != nil {
		t.Fatalf("Employment() error = %v, want nil", err)
	}
	return u, b, e
}

// TestRun_LabourPresetsOverRealWindows verifies the three presets of the employment module against
// the LabourRound events of two real windows of the hire pack: the households offered, hired and
// unmatched of Unemployment are the sums of the events and the offered are the hired and the
// unmatched; the wages of WageBill are the money the events paid, no start was cut so nothing is
// refunded, and the share of the value added is defined; the paid household-ticks of Employment
// are those of the events, over the running households of the hireable roles, and no hire is open at
// the close (the jobs of the pack take one tick).
//
// Requirement: FR-IND-056, FR-IND-057, FR-IND-058
//
// Guard: a tally row that dropped a region or double-counted a round would differ from the events.
func TestRun_LabourPresetsOverRealWindows(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	w, rec, wins := hireWindows(t)
	for i, rw := range wins {
		want := sumLabour(rec.events, rw.w.From, rw.w.To)
		if want.offered == 0 || want.hired == 0 {
			t.Fatalf("window %d: %d households offered and %d hired, want hires in the window", i, want.offered, want.hired)
		}
		u, b, e := labourPresets(t, w, rw)
		checkUnemployment(t, i, u, want)
		checkWageBill(t, i, b, want)
		checkEmployment(t, i, e, want)
	}
}

// checkUnemployment compares the realm row of Unemployment with the sums of the events.
func checkUnemployment(t *testing.T, i int, u indicator.UnemploymentResult, want labourSums) {
	t.Helper()
	tot := u.Total
	if tot.Offered != want.offered || tot.Hired != want.hired || tot.NoVacancy != want.noVacancy || !tot.Rate.OK {
		t.Errorf("window %d: Unemployment().Total = %+v, want offered %d, hired %d, unmatched %d and a rate", i, tot, want.offered, want.hired, want.noVacancy)
	}
	if tot.Offered != tot.Hired+tot.NoVacancy {
		t.Errorf("window %d: Unemployment().Total = %+v, want offered = hired + unmatched", i, tot)
	}
}

// checkWageBill compares the realm row of WageBill with the money the events paid.
func checkWageBill(t *testing.T, i int, b indicator.WageBillResult, want labourSums) {
	t.Helper()
	if int64(b.Realm.Paid) != want.paid || int64(b.Realm.Refunded) != want.refunded || b.Realm.Paid <= 0 {
		t.Errorf("window %d: WageBill().Realm = %+v, want paid %d and refunded %d", i, b.Realm, want.paid, want.refunded)
	}
	if !b.Realm.Share.OK || !b.Realm.ShareCash.OK {
		t.Errorf("window %d: WageBill().Realm = %+v, want both shares defined", i, b.Realm)
	}
}

// checkEmployment compares the window form of Employment with the events and reads the close.
func checkEmployment(t *testing.T, i int, e indicator.EmploymentResult, want labourSums) {
	t.Helper()
	if e.Total.HouseholdTicks != want.householdTicks || e.Total.Running <= 0 || !e.Total.Rate.OK {
		t.Errorf("window %d: Employment().Total = %+v, want %d paid household-ticks over the running households", i, e.Total, want.householdTicks)
	}
	for _, r := range e.Rows {
		if r.Employed != 0 {
			t.Errorf("window %d: Employment() row %+v, want no household on a hired job at the close of a one-tick job", i, r)
		}
	}
}

// labourGoldenHeader says what the file is and how it is recorded.
const labourGoldenHeader = `# Presets 22 to 24 of spec-indicators.md 9.1 for the second window of 100 ticks of the pack
# one-demand with the employment module in its hire form, warm (seed 1, 200 ticks). Integers
# only; a field that is 0, false or empty is not written. It records no world state, so
# testdata/golden/off/ does not move (plan 3-emp Phase 8). The pack collapses after about tick 250
# (N-352), so the window is read before it.
`

// TestGolden_IndicatorLabourPresetsOnTheHirePack verifies the rendering of presets 22 to 24 for the
// second window of the hire pack, seed 1, against testdata/golden/indicator/employment_hire.txt. A
// change of a preset or of the pack data moves the file; regenerate it with -update only after
// reading testing.md 7.2 and understanding the diff.
//
// Requirement: FR-IND-056, FR-IND-057, FR-IND-058
func TestGolden_IndicatorLabourPresetsOnTheHirePack(t *testing.T) {
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	w, _, wins := hireWindows(t)
	u, b, e := labourPresets(t, w, wins[1])
	got := append([]byte(labourGoldenHeader), render(t, []named{{"Unemployment", u}, {"WageBill", b}, {"Employment", e}})...)
	path := filepath.Join("..", "testdata", "golden", "indicator", "employment_hire.txt")
	if *updateFlag {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v, want nil", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v; record it with -update after reading testing.md 7.2", path, err)
	}
	if string(want) != string(got) {
		wl, gl := strings.Split(string(want), "\n"), strings.Split(string(got), "\n")
		for i := 0; i < max(len(wl), len(gl)); i++ {
			var a, c string
			if i < len(wl) {
				a = wl[i]
			}
			if i < len(gl) {
				c = gl[i]
			}
			if a != c {
				t.Fatalf("golden %s differs at line %d:\n want %q\n  got %q\n(regenerate with -update only after reading testing.md 7.2)", path, i+1, a, c)
			}
		}
	}
}
