package economysim

import (
	"fmt"
	"slices"
	"testing"
)

// runBeforeWelfare runs the first step of w up to, and not including, P6.3 by the rows of
// the replay, so the flows of the tick are still in the welfare scratch. The step is
// left open: a test reads the scratch and the rings and does not step again.
func runBeforeWelfare(t *testing.T, w *World) {
	t.Helper()
	i := slices.IndexFunc(replayRows, func(r replayRow) bool { return r.id == "P6.3" })
	if i < 0 {
		t.Fatalf("replayRows has no row P6.3")
	}
	if err := runRows(w, w.Tick(), replayRows[:i], func(string) {}); err != nil {
		t.Fatalf("positions up to P6.3: %v", err)
	}
}

// TestWelfare_HiredTickIsWorkingTickPrincipalFloorNoAdd verifies the welfare rules of a
// hired tick (spec-employment.md 8.1 to 8.3): the worker's wage is its income (Y_t) and
// its tick is a working tick of k household-ticks though its role has no recipe; the
// employer's income is less the wage of the tick (V_t), not the escrow, and its tick is a
// working tick for the floor but adds no working household-tick to the wage it observes.
//
// Setup: one employer, one hand agent of weight 2 (a role with no recipe); bid 100,
// reserve 0, so the wage is 0 + 100 * 5 000 / 10 000 = 50 a household-tick, 100 a tick
// for k = 2; the job takes 3 ticks, so the escrow is 300, of which only the 100 of the
// tick is paid by P2.7.
//
// Requirement: FR-EMP-045, FR-EMP-046
func TestWelfare_HiredTickIsWorkingTickPrincipalFloorNoAdd(t *testing.T) {
	t.Parallel()
	o := defaultHire(hireOpts{hands: 2, handScale: 2})
	w := hireWorld(t, o, byRole(
		JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: 2, Bid: 100}},
		JobRequest{NoJob: true, Offer: LabourOffer{Households: 2}}))
	farm, hand := slotsOf(w, "farm")[0], slotsOf(w, "hand")[0]
	if got := w.agents[hand].weight; got != 2 {
		t.Fatalf("weight of the hand agent = %d, want 2", got)
	}
	runBeforeWelfare(t, w)
	hf, ff := w.welfare.at(hand), w.welfare.at(farm)
	if got, want := fmt.Sprintf("%d %d %v", hf.earned, hf.k, hf.jobSet), "100 2 true"; got != want {
		t.Errorf("hand flows (earned, k, jobSet) = %s, want %s", got, want)
	}
	// The farm paid the wage of the tick, 100, and not the escrow of 300.
	if got, want := fmt.Sprintf("%d %v", ff.wagePaid, ff.principal), "100 true"; got != want {
		t.Errorf("farm flows (wagePaid, principal) = %s, want %s", got, want)
	}
	w.p63Welfare()
	hr, fr := w.agents[hand].history, w.agents[farm].history
	hn, hd := hr.WageParts()
	fn, fd := fr.WageParts()
	if got, want := fmt.Sprintf("%d/%d/%d", hn, hd, hr.WorkingTicks()), "100/2/1"; got != want {
		t.Errorf("hand ring (wage parts, working ticks) = %s, want %s", got, want)
	}
	if got, want := fmt.Sprintf("%d/%d/%d", fn, fd, fr.WorkingTicks()), "-100/0/1"; got != want {
		t.Errorf("farm ring (wage parts, working ticks) = %s, want %s: a working tick for the floor, no household-tick", got, want)
	}
}

// TestWelfare_HireableNoRecipeWorksByJobOnly verifies the clause of
// spec-welfare-and-wages.md 4.3 that the hire form turns off for a Hireable role
// (spec-employment.md 8.3.1): a role with no recipe works by trading, so a worker that
// bought its upkeep would count its whole weight as working household-ticks; in the hire
// form a tick of a Hireable role works only by a job, and not in the payroll form.
//
// Requirement: FR-EMP-046
func TestWelfare_HireableNoRecipeWorksByJobOnly(t *testing.T) {
	t.Parallel()
	hire := &employment{hire: true, hireable: []bool{false, true}}
	payroll := &employment{hireable: []bool{false, true}}
	for _, tt := range []struct {
		name string
		e    *employment
		role RoleID
		want bool
	}{
		{"hire form, Hireable", hire, 1, true},
		{"hire form, not Hireable", hire, 0, false},
		{"payroll form keeps the clause", payroll, 1, false},
		{"module off keeps the clause", nil, 1, false},
	} {
		if got := tt.e.worksByJob(tt.role); got != tt.want {
			t.Errorf("worksByJob(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
	// A weight of 5 that traded and had no job: the clause makes it a working tick of 5,
	// and the job rule makes it none.
	if working, workHH, _ := workingTick(false, 0, false, true, false, 5); !working || workHH != 5 {
		t.Errorf("workingTick(no recipes, traded, weight 5) = %v, %d, want true, 5", working, workHH)
	}
	if working, workHH, _ := workingTick(true, 0, false, true, false, 5); working || workHH != 0 {
		t.Errorf("workingTick(by job, traded, no job) = %v, %d, want false, 0", working, workHH)
	}
}

// TestWelfare_PrincipalPaysWear verifies that the wear of the tools a hired job uses is
// the employer's flow, not the worker's (spec-employment.md 8.1): the tool is the
// employer's capital.
//
// Requirement: FR-EMP-049
func TestWelfare_PrincipalPaysWear(t *testing.T) {
	t.Parallel()
	o := defaultHire(hireOpts{hands: 2, tool: true, farmTools: 5})
	w := hireWorld(t, o, byRole(
		JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: 2, Bid: 100}},
		JobRequest{NoJob: true, Offer: LabourOffer{Households: 1}}))
	farm, hand := slotsOf(w, "farm")[0], slotsOf(w, "hand")[0]
	runBeforeWelfare(t, w)
	if f := w.welfare.at(farm); f.nWear != 1 || f.wear[0].executions != 2 {
		t.Errorf("farm wear rows = %d (first %+v), want 1 row of 2 executions", f.nWear, f.wear[0])
	}
	if f := w.welfare.at(hand); f.nWear != 0 {
		t.Errorf("hand wear rows = %d, want 0: the tool is not the worker's", f.nWear)
	}
}

// TestWelfare_HireDoesNotLengthenWindow verifies that a hire adds nothing to the window
// of a worker (spec-employment.md 8.5): the window of a role is a function of its own
// recipes, so a hand with no recipe keeps WelfareWindow though the job it is hired
// for lasts longer, while the employer's window covers its hire list.
//
// Setup: the recipe lasts 60 ticks; WelfareWindow is 30 and WindowCycles at its default
// times 60 is above it for the farm.
//
// Requirement: FR-EMP-048
func TestWelfare_HireDoesNotLengthenWindow(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{duration: 60}), farmWants(1, 100, 0))
	hand, farm := w.roles.row(1).window, w.roles.row(0).window
	if want := w.cfg.World.WelfareWindow; hand != want {
		t.Errorf("window of the hand role = %d, want WelfareWindow %d", hand, want)
	}
	if farm < 60 {
		t.Errorf("window of the farm role = %d, want at least the 60 ticks of its hire list", farm)
	}
}

// TestWelfare_PooledWageExcludesEmployerRoles verifies that the pooled regional wage
// leaves the employer roles out in the hire form as in the payroll form, from the
// configuration alone and with no state (spec-employment.md 8.4).
//
// Requirement: FR-EMP-047
func TestWelfare_PooledWageExcludesEmployerRoles(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{}), farmWants(1, 100, 0))
	if !w.employment.isPayer(0) || w.employment.isPayer(1) {
		t.Errorf("isPayer(farm), isPayer(hand) = %v, %v, want true, false", w.employment.isPayer(0), w.employment.isPayer(1))
	}
}

// TestDemand_HireableNoClassWithoutIncomeWhenEmployerExists verifies that the warning
// class_without_income is not raised for a Hireable role in a world with the employment
// module and an employer role: its income is wages (spec-demand.md 10.2).
//
// Requirement: FR-EMP-001
func TestDemand_HireableNoClassWithoutIncomeWhenEmployerExists(t *testing.T) {
	t.Parallel()
	hand := DefaultRole()
	hand.ID, hand.Hireable = "hand", true
	farm := DefaultRole()
	farm.ID, farm.Employer = "farm", &Employer{}
	cfg := &Config{World: DefaultWorldParams(), Calendar: DefaultCalendar(), Roles: []Role{farm, hand}}
	cfg.Modules.Employment = &Employment{Hire: true}
	ro := &cfg.Roles[1]
	// 3 ticks of an upkeep of 1 000 plus a year of wants at 100 is above a start of 0.
	if !classWithoutIncome(&Config{World: cfg.World, Calendar: cfg.Calendar, Roles: []Role{hand}}, &hand, 1_000, 100) {
		t.Fatalf("classWithoutIncome(Hireable, no employer role) = false, want true: the fixture must raise the warning")
	}
	if classWithoutIncome(cfg, ro, 1_000, 100) {
		t.Errorf("classWithoutIncome(Hireable, employer role in the world) = true, want false")
	}
	cfg.Modules.Employment = nil
	if !classWithoutIncome(cfg, ro, 1_000, 100) {
		t.Errorf("classWithoutIncome(Hireable, module off) = false, want true")
	}
}
