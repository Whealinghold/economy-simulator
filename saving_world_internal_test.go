package economysim

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// richMiller is the fixture of the end-to-end tests: the world of savWorld in which the grind
// recipe makes 60 flour (so the mill is worth building at the beliefs of the pack), with a
// ring of 30 records of a surplus of 19 000 for the weight-5 miller, so that every project is
// reachable. The numbers are placeholders (N-015); the pack is a hand pack with a site recipe
// and a build recipe, as plan 3-sav Phase 5 asks.
func richMiller(t testing.TB, mut func(c *Config), opts ...Option) (*World, AgentID) {
	t.Helper()
	w := savWorld(t, func(c *Config) {
		for i := range c.Recipes {
			if c.Recipes[i].ID == "zz_grind" {
				c.Recipes[i].Outputs[0].Quantity = 60
			}
		}
		if mut != nil {
			mut(c)
		}
	}, opts...)
	m := bldMiller(w, 0)
	for range 30 {
		w.agents[m].history.Push(welfare.Record{Income: 20_000, Cost: 1_000})
	}
	return w, m
}

// ownedMills is the number of mills of the agent.
func ownedMills(w *World, id AgentID) int {
	n := 0
	for _, f := range w.infra.fac {
		if f.owner == id && f.class == clsMill {
			n++
		}
	}
	return n
}

// TestSaving_BuildsItsOwnMill verifies the module end to end through the default strategy
// alone, with no script: a saver miller of weight 5 asks for the build of a mill each time its
// project passes and stops when the capacity reaches its weight (dJ is 0 at the weight,
// 5.1.3), so it ends with ceil(5 / 2) = 3 mills of capacity 2; the same world with the module
// off builds none, because the default builds and repairs nothing for a role with no block
// (FR-CAP-018). The audit is on in both.
//
// Requirement: FR-SAV-021, FR-SAV-025, FR-SAV-041
func TestSaving_BuildsItsOwnMill(t *testing.T) {
	t.Parallel()
	w, m := richMiller(t, nil)
	for range 14 {
		facStep(t, w)
	}
	if got := ownedMills(w, m); got != 3 {
		t.Errorf("mills of the saver after 14 ticks = %d, want 3 (capacity 2 each, weight 5) with seed %d", got, facSeed)
	}
	off, om := richMiller(t, func(c *Config) { c.Modules.Saving, c.Roles[0].Saver = nil, nil })
	for range 14 {
		facStep(t, off)
	}
	if got := ownedMills(off, om); got != 0 {
		t.Errorf("mills of a role with no block after 14 ticks = %d, want 0", got)
	}
}

// savingLog runs the world of richMiller for 10 ticks with a Debug logger and returns the
// lines the module wrote, as records, with the world and the miller.
func savingLog(t testing.TB) ([]slog.Record, *World, AgentID) {
	t.Helper()
	var recs []slog.Record
	w, m := richMiller(t, nil, WithLogger(slog.New(mkLogs{&recs})))
	for range 10 {
		facStep(t, w)
	}
	return recs, w, m
}

// TestLog_SavingProjectChosen verifies the Debug line "project chosen" of 12.2: it carries the
// tick, the agent, the recipe, the goal and the buffer, at Debug.
//
// Arithmetic: the weight-5 miller at tick 0 has the build chosen, its goal 25 000 (the outlay
// less the inputs it holds).
//
// Requirement: FR-SAV-039
func TestLog_SavingProjectChosen(t *testing.T) {
	t.Parallel()
	recs, w, m := savingLog(t)
	var chosen int
	for _, r := range recs {
		if r.Message != "project chosen" {
			continue
		}
		chosen++
		a := logAttrs(r)
		if a[logkeys.Agent].Int64() != int64(m) || a[logkeys.Goal].Int64() <= 0 || a[logkeys.Recipe].Int64() != int64(bldRecipe(t, w, "zz_build")) {
			t.Errorf("project chosen line = %+v, want the agent %d, a goal above 0 and the recipe zz_build", a, m)
		}
		if _, ok := a[logkeys.Buffer]; !ok || r.Level != slog.LevelDebug {
			t.Errorf("project chosen line has buffer %v at level %v, want the key at Debug", ok, r.Level)
		}
	}
	if chosen == 0 {
		t.Errorf("lines %q over 10 ticks = 0, want at least one while the mills are built", "project chosen")
	}
}

// TestLog_SavingProjectDeclined verifies the Debug line "project declined": the reason is the
// bar, no_return for the miller that has its three mills and a build that passes no more, and
// unreachable for the miller of the other region, whose ring has no surplus, so its project,
// which passes, is not reachable. No line is at Info or above.
//
// Requirement: FR-SAV-039
func TestLog_SavingProjectDeclined(t *testing.T) {
	t.Parallel()
	recs, _, m := savingLog(t)
	var own int
	for _, r := range recs {
		if r.Level >= slog.LevelInfo && r.Message == "project declined" {
			t.Errorf("project declined line at %v, want Debug", r.Level)
		}
		if r.Message != "project declined" {
			continue
		}
		a := logAttrs(r)
		want := BarUnreachable
		if a[logkeys.Agent].Int64() == int64(m) {
			own++
			want = BarNoReturn
		}
		if got := SavingBar(a[logkeys.Reason].String()); got != want {
			t.Errorf("project declined line of agent %d: reason = %q, want %q", a[logkeys.Agent].Int64(), got, want)
		}
	}
	if own == 0 {
		t.Errorf("project declined lines of the miller over 10 ticks = 0, want at least one once its mills are built")
	}
}

// TestSaving_NonInterference verifies that a logger and an observer change nothing (FR-SAV-038):
// two worlds of the same seed, one with a Debug logger and an observer, give the same snapshot
// after 12 ticks.
//
// Requirement: FR-SAV-038
func TestSaving_NonInterference(t *testing.T) {
	t.Parallel()
	var recs []slog.Record
	a, _ := richMiller(t, nil)
	b, _ := richMiller(t, nil, WithLogger(slog.New(mkLogs{&recs})), WithObserver(&evLog{}))
	for range 12 {
		facStep(t, a)
		facStep(t, b)
	}
	sa, err := a.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	sb, err := b.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(sa, sb) {
		t.Errorf("snapshots after 12 ticks differ with a logger and an observer (seed %d)", facSeed)
	}
	if len(recs) == 0 {
		t.Errorf("the logger saw no line: the test would pass with the lines removed")
	}
}

// TestCost_SavingWorkAndTradeZeroAllocs verifies that the default Work and Trade of a saver
// allocate nothing once warm: the evaluation of the projects uses stack arrays and the scratch
// of the view (NFR-SAV-002).
//
// Requirement: NFR-SAV-002
func TestCost_SavingWorkAndTradeZeroAllocs(t *testing.T) {
	w, m := richMiller(t, nil)
	w.refreshViewCtx()
	a := &w.agents[m]
	if err := w.fillAgentView(&w.tk.view, a, viewTrade); err != nil {
		t.Fatalf("fillAgentView() error = %v, want nil", err)
	}
	p := w.views.tradePlan()
	run := func() {
		_ = (DefaultStrategy{}).Work(&w.tk.view, w.rnd.rnd)
		p = w.views.tradePlan()
		(DefaultStrategy{}).Trade(&w.tk.view, w.rnd.rnd, p)
	}
	run() // grows the scratch of the plan
	if got := testing.AllocsPerRun(20, run); got != 0 {
		t.Errorf("AllocsPerRun(Work and Trade of a saver) = %v, want 0", got)
	}
}

// TestLog_SavingOneLinePerAgentAndTick verifies that the Debug lines of the module are
// logged once per decision: the rule runs three times in a tick (the Work view, the Trade
// view and the default's Trade), but only the fill of the Trade view logs, so an agent has at
// most one of "project chosen" and "project declined" in a tick.
//
// Requirement: FR-SAV-039
func TestLog_SavingOneLinePerAgentAndTick(t *testing.T) {
	t.Parallel()
	recs, _, _ := savingLog(t)
	type key struct{ tick, agent int64 }
	seen := map[key]int{}
	for _, r := range recs {
		if r.Message != "project chosen" && r.Message != "project declined" {
			continue
		}
		a := logAttrs(r)
		seen[key{a[logkeys.Tick].Int64(), a[logkeys.Agent].Int64()}]++
	}
	if len(seen) == 0 {
		t.Fatalf("project lines over 10 ticks = 0, want at least one")
	}
	for k, n := range seen {
		if n != 1 {
			t.Errorf("agent %d at tick %d has %d project lines, want 1", k.agent, k.tick, n)
		}
	}
}
