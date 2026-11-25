package economysim

import (
	"bytes"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

// savingAt is the Work view of agent m at the boundary of w, built as the query builds it:
// the anchors and the moment of the next step are derived first.
func savingAt(t testing.TB, w *World, m AgentID) AgentView {
	t.Helper()
	if _, err := w.prepareAnchors(); err != nil {
		t.Fatalf("prepareAnchors() error = %v, want nil", err)
	}
	return savView(t, w, m, viewWork)
}

// TestQuery_SavingState verifies that the query returns what the exported rules return for
// the same view with tier 2 read as 0: the floor, the precaution, the goal and the project
// chosen (FR-SAV-037, FR-SAV-042), and the drawdown of the agent's ring. At tick 2 of the
// fixture the miller has no job and its next mill passes (benefit 39 742 440, cost 717 060,
// outlay 25 480 at the beliefs of the pack).
//
// Requirement: FR-SAV-037, FR-SAV-042
func TestQuery_SavingState(t *testing.T) {
	t.Parallel()
	w, m := richMiller(t, nil)
	facStep(t, w)
	facStep(t, w)
	got, err := w.Saving(m)
	if err != nil {
		t.Fatalf("Saving(%d) error = %v, want nil", m, err)
	}
	v := savingAt(t, w, m)
	res := SavingTarget(&v, 0)
	pc, ok := ChooseProject(&v)
	reserve, _ := v.savingBasis()
	want := SavingState{
		Reserve: Money(reserve), Floor: Money(res.Floor), Precaution: Money(res.Precaution),
		Drawdown: Money(v.Saving.Drawdown), Goal: Money(res.Goal), Idle: Money(max(0, int64(v.Money)-res.Target)),
		Project: pc, Chosen: ok, Building: res.Building, Bar: res.Bar,
	}
	if got != want {
		t.Errorf("Saving(%d) = %+v, want %+v", m, got, want)
	}
	if !got.Chosen || got.Project.Benefit != 39_742_440 || got.Project.Cost != 717_060 || got.Project.Outlay != 25_480 {
		t.Errorf("Saving(%d).Project = %+v (chosen %t), want benefit 39742440, cost 717060, outlay 25480", m, got.Project, got.Chosen)
	}
	if got.Drawdown <= 0 || got.Floor < got.Reserve || got.Floor < got.Precaution {
		t.Errorf("Saving(%d) = %+v, want a drawdown above 0 and a floor of at least the reserve and the precaution", m, got)
	}
}

// TestQuery_KeepOfAJobInProgress verifies that for a household whose job is a project in
// progress no project is chosen, Building is set and the goal is the keep of FR-SAV-046,
// UpkeepCost * (Remaining + 1 + SafetyTicks).
//
// Arithmetic: after one tick the miller has the build of its first mill in progress with one
// tick to go; the module's SafetyTicks is 2, so the keep is the upkeep cost of 4 ticks.
//
// Requirement: FR-SAV-037, FR-SAV-046
func TestQuery_KeepOfAJobInProgress(t *testing.T) {
	t.Parallel()
	w, m := richMiller(t, nil)
	facStep(t, w)
	got, err := w.Saving(m)
	if err != nil {
		t.Fatalf("Saving(%d) error = %v, want nil", m, err)
	}
	v := savingAt(t, w, m)
	cost, _ := v.upkeepCost()
	if want := Money(cost * (v.Remaining + 1 + 2)); !got.Building || got.Chosen || got.Goal != want {
		t.Errorf("Saving(%d) = %+v, want Building with the keep %d (upkeep %d, remaining %d) and no project", m, got, want, cost, v.Remaining)
	}
}

// TestQuery_Errors verifies the errors of the query in the order of spec-integration.md
// 6.7.1: the module off, an agent never issued, a pseudo-agent, then ErrNotSaver with the zero
// value for the weaver, whose role has no block.
//
// Requirement: FR-SAV-037
func TestQuery_Errors(t *testing.T) {
	t.Parallel()
	w, _ := richMiller(t, nil)
	off, om := richMiller(t, func(c *Config) { c.Modules.Saving, c.Roles[0].Saver = nil, nil })
	weaver := facAgents(w, 0, "weaver")[0]
	tests := []struct {
		name string
		w    *World
		id   AgentID
		want error
	}{
		{"module off", off, om, ErrModuleOff},
		{"never issued", w, 10_000, ErrUnknownAgent},
		{"negative", w, -1, ErrUnknownAgent},
		{"pseudo-agent", w, 0, ErrUnknownAgent},
		{"role with no block", w, weaver, ErrNotSaver},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.w.Saving(tt.id)
			if !errors.Is(err, tt.want) {
				t.Errorf("Saving(%d) error = %v, want %v", tt.id, err, tt.want)
			}
			if got != (SavingState{}) {
				t.Errorf("Saving(%d) = %+v with an error, want the zero value", tt.id, got)
			}
		})
	}
}

// TestQuery_ChangesNothing verifies that the query and the snapshot, which build views between
// two steps, draw nothing and change no saved state: a world queried and snapshotted at every
// boundary equals a twin that was not, tick for tick.
//
// Requirement: FR-SAV-037, FR-SAV-038
func TestQuery_ChangesNothing(t *testing.T) {
	t.Parallel()
	a, ma := richMiller(t, nil)
	b, _ := richMiller(t, nil)
	for range 8 {
		facStep(t, a)
		facStep(t, b)
		if _, err := a.Saving(ma); err != nil {
			t.Fatalf("Saving(%d) error = %v, want nil", ma, err)
		}
		if _, err := a.Snapshot(); err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
	}
	sa, errA := a.Snapshot()
	sb, errB := b.Snapshot()
	if errA != nil || errB != nil {
		t.Fatalf("Snapshot() errors = %v, %v, want nil", errA, errB)
	}
	if !reflect.DeepEqual(sa, sb) {
		t.Errorf("snapshot of the queried world differs from the twin's after 8 ticks (seed %d)", facSeed)
	}
}

// TestSnapshot_SavingRows verifies the four rows of 8.4.2 against the query: the row of a role
// with a block is the sum over its agents of the query's goal, of the excess of the floor over
// the reserve and of the idle money, and its households those of the agents with a goal above
// 0; the weaver's row, which has no block, holds 0 in all four.
//
// Requirement: FR-SAV-031
func TestSnapshot_SavingRows(t *testing.T) {
	t.Parallel()
	w, _ := richMiller(t, nil)
	for range 4 {
		facStep(t, w)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	nr := len(w.cfg.Roles)
	sawGoal := false
	for g := range w.cfg.Regions {
		want, saw := savingRowWant(t, w, RegionID(g))
		sawGoal = sawGoal || saw
		row := s.Roles[g*nr]
		if got := (RoleRow{SavingHouseholds: row.SavingHouseholds, GoalMoney: row.GoalMoney, BufferMoney: row.BufferMoney, IdleMoney: row.IdleMoney}); got != want {
			t.Errorf("rows of miller in region %d = %+v, want %+v (the sum of the queries)", g, got, want)
		}
		if wv := s.Roles[g*nr+1]; savingFieldsSet(wv) {
			t.Errorf("row of the weaver in region %d = %+v, want the four saving fields 0 (no block)", g, wv)
		}
	}
	if !sawGoal {
		t.Errorf("no agent had a goal above 0 at tick 4: the test would pass with the rows 0")
	}
}

// savingFieldsSet reports whether any of the four saving fields of a role row is non-zero.
func savingFieldsSet(r RoleRow) bool {
	return r.SavingHouseholds != 0 || r.GoalMoney != 0 || r.BufferMoney != 0 || r.IdleMoney != 0
}

// savingRowWant sums the query over the millers of region g into the four saving fields of
// a role row, and reports whether any agent had a goal above 0.
func savingRowWant(t *testing.T, w *World, g RegionID) (RoleRow, bool) {
	t.Helper()
	var want RoleRow
	saw := false
	for _, id := range facAgents(w, g, "miller") {
		st, err := w.Saving(id)
		if err != nil {
			t.Fatalf("Saving(%d) error = %v, want nil", id, err)
		}
		if st.Goal > 0 {
			want.SavingHouseholds += w.agents[id].weight
			saw = true
		}
		want.GoalMoney += st.Goal
		want.BufferMoney += max(0, st.Floor-st.Reserve)
		want.IdleMoney += st.Idle
	}
	return want, saw
}

// TestSnapshot_ModuleOffRowsAreZero verifies that a world with the module off, or on with no
// block, reports 0 in the four fields and pays for no view: the rows are omitted at their
// off value (architecture.md 2.10.5).
//
// Requirement: FR-SAV-001, FR-SAV-031
func TestSnapshot_ModuleOffRowsAreZero(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mut  func(c *Config)
	}{
		{"module off", func(c *Config) { c.Modules.Saving, c.Roles[0].Saver = nil, nil }},
		{"module on, no block", func(c *Config) { c.Roles[0].Saver = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := richMiller(t, tt.mut)
			facStep(t, w)
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v, want nil", err)
			}
			for _, r := range s.Roles {
				if r.SavingHouseholds != 0 || r.GoalMoney != 0 || r.BufferMoney != 0 || r.IdleMoney != 0 {
					t.Errorf("row of role %d in region %d = %+v, want the four saving fields 0", r.Role, r.Region, r)
				}
			}
		})
	}
}

// TestSnapshot_RestoredEqualsOriginal verifies FR-SAV-040: a world restored from a checkpoint
// taken while the miller is mid-build answers the query and the four rows as the original, and
// goes on to equal it for five more ticks.
//
// Requirement: FR-SAV-031, FR-SAV-037, FR-SAV-040
func TestSnapshot_RestoredEqualsOriginal(t *testing.T) {
	t.Parallel()
	w, m := richMiller(t, nil)
	facStep(t, w) // the first build is in progress
	r, err := restoreOf(t, w)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	for i := range 6 {
		wst, errW := w.Saving(m)
		rst, errR := r.Saving(m)
		if errW != nil || errR != nil {
			t.Fatalf("Saving(%d) errors = %v, %v, want nil", m, errW, errR)
		}
		if wst != rst {
			t.Errorf("tick %d: Saving(%d) = %+v on the restored world, want %+v", w.Tick(), m, rst, wst)
		}
		ws, _ := w.Snapshot()
		rs, _ := r.Snapshot()
		if !reflect.DeepEqual(ws.Roles, rs.Roles) {
			t.Errorf("tick %d: role rows of the restored world differ from the original's", w.Tick())
		}
		if i == 0 && !wst.Building {
			t.Errorf("Saving(%d) at the checkpoint = %+v, want a job in progress (Building)", m, wst)
		}
		facStep(t, w)
		facStep(t, r)
	}
}

// TestIdle_MaxZeroMoneyMinusTarget verifies the idle money of 8.1.1: max(0, Money - S*), over
// the run, with both a positive idle money (tick 3 of the fixture holds 3 691 over its target)
// and a household with none.
//
// Requirement: FR-SAV-032
func TestIdle_MaxZeroMoneyMinusTarget(t *testing.T) {
	t.Parallel()
	w, m := richMiller(t, nil)
	var positive, zero bool
	for range 8 {
		facStep(t, w)
		st, err := w.Saving(m)
		if err != nil {
			t.Fatalf("Saving(%d) error = %v, want nil", m, err)
		}
		v := savingAt(t, w, m)
		if want := Money(max(0, int64(v.Money)-SavingTarget(&v, 0).Target)); st.Idle != want {
			t.Errorf("tick %d: Idle = %d, want max(0, %d - S*) = %d", w.Tick(), st.Idle, v.Money, want)
		}
		positive, zero = positive || st.Idle > 0, zero || st.Idle == 0
	}
	if !positive || !zero {
		t.Errorf("idle money over 8 ticks had a positive value %t and a zero %t, want both", positive, zero)
	}
}

// TestLog_SavingDebugGuarded verifies that a logger that discards Debug sees none of the
// module's lines and that none of them is at Info or above: the lines are guarded by Enabled
// and "no Warn inside the tick" holds (FR-SAV-039).
//
// Requirement: FR-SAV-039
func TestLog_SavingDebugGuarded(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	w, _ := richMiller(t, nil, WithLogger(log))
	mark := buf.Len()
	for range 8 {
		facStep(t, w)
	}
	if got := buf.String()[mark:]; strings.Contains(got, "project") {
		t.Errorf("log lines of 8 ticks at Info = %q, want none of the module's Debug lines", got)
	}
}

// TestAlloc_SavingNoObserver verifies the cost with the default observer and logger
// (observability-and-logging.md 8.2): the Trade-view fill with its count call, the request
// count, P3.6b and the clear at P8.6 allocate nothing once warm.
//
// Requirement: FR-SAV-038, NFR-SAV-002
func TestAlloc_SavingNoObserver(t *testing.T) {
	w, m := richMiller(t, nil)
	w.refreshViewCtx()
	a := &w.agents[m]
	req := JobRequest{Recipe: 0, Executions: 1}
	run := func() {
		if err := w.fillAgentView(&w.tk.view, a, viewTrade); err != nil {
			t.Fatalf("fillAgentView() error = %v, want nil", err)
		}
		w.saving.noteRequest(a, 0, &req)
		w.p36bSaving()
		w.saving.clearTick()
	}
	run()
	if got := testing.AllocsPerRun(20, run); got != 0 {
		t.Errorf("AllocsPerRun(Trade view fill, noteRequest, P3.6b, clearTick) = %v, want 0", got)
	}
}

// TestSaving_NoteRequest verifies the request count at the lowest level: a build request of
// an agent with a block counts as a build, a repair request as a repair, an inert request, a
// request of a recipe out of range and a request of an agent with no block count nothing.
//
// Requirement: FR-SAV-030
func TestSaving_NoteRequest(t *testing.T) {
	t.Parallel()
	ev := &evLog{}
	w, m := richMiller(t, nil, WithObserver(ev))
	weaver := facAgents(w, 0, "weaver")[0]
	build, repair := bldRecipe(t, w, "zz_build"), bldRecipe(t, w, "zz_fix")
	tests := []struct {
		name string
		a    AgentID
		req  JobRequest
		want savingCount
	}{
		{"a build", m, JobRequest{Recipe: build, Executions: 1}, savingCount{builds: 1}},
		{"a repair", m, JobRequest{Recipe: repair, Executions: 3}, savingCount{repairs: 1}},
		{"no job", m, JobRequest{NoJob: true, Recipe: build}, savingCount{}},
		{"a recipe out of range", m, JobRequest{Recipe: 999, Executions: 1}, savingCount{}},
		{"a negative recipe", m, JobRequest{Recipe: -1, Executions: 1}, savingCount{}},
		{"a role with no block", weaver, JobRequest{Recipe: build, Executions: 1}, savingCount{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w.saving.clearTick()
			a := &w.agents[tt.a]
			w.saving.noteRequest(a, 0, &tt.req)
			if got := w.saving.counts[int(0)*len(w.cfg.Roles)+int(a.role)]; got != tt.want {
				t.Errorf("noteRequest(%+v) counts = %+v, want %+v", tt.req, got, tt.want)
			}
		})
	}
}

// TestCardinality_SavingBarsAndProjects verifies that the attributes economysim.bar and
// economysim.project that SavingPlanned feeds take only their closed sets, and that its region
// and role are IDs of the configuration: the four bar constants are the four values of the rule
// file and the event has one count for each, and the two projects are its Builds and Repairs.
//
// Requirement: FR-SAV-029
//
// Guard: a fifth bar constant, or a free-text bar, would make the attribute unbounded.
func TestCardinality_SavingBarsAndProjects(t *testing.T) {
	t.Parallel()
	got := []SavingBar{BarNoReturn, BarStanding, BarUnreachable, BarSurvival}
	want := []string{"no_return", "standing", "unreachable", "survival"}
	for i, b := range got {
		if string(b) != want[i] {
			t.Errorf("bar constant %d = %q, want %q", i, b, want[i])
		}
	}
	evs, w, _ := savingRun(t, 6)
	for _, e := range evs {
		if int(e.Region) >= len(w.cfg.Regions) || int(e.Role) >= len(w.cfg.Roles) || e.Region < 0 || e.Role < 0 {
			t.Errorf("SavingPlanned %+v names a region or role outside the configuration", e)
		}
	}
	if n := reflect.TypeOf(SavingPlanned{}).NumField(); n != 18 {
		t.Errorf("SavingPlanned has %d fields, want 18: a new count needs its row in the metric feeds", n)
	}
}
