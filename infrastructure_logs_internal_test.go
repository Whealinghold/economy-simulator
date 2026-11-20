package economysim

import (
	"log/slog"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// logAttrs returns the attributes of a record by key, as strings or ints in a small struct.
func logAttrs(r slog.Record) map[string]slog.Value {
	m := map[string]slog.Value{}
	r.Attrs(func(a slog.Attr) bool { m[a.Key] = a.Value; return true })
	return m
}

// TestComplete_VoidedIsLoggedWithoutAnObserver verifies that a build that loses executions at
// its completion says so in a Debug line with the class, the region and the count, though no
// observer is supplied and so FacilityLimited.Voided is not counted.
//
// Arithmetic: the build starts for 5, two mills of room are left at the completion, so
// 5 - 2 = 3 executions are voided (as TestComplete_Voided).
//
// Guard: without the line a run with only a logger cannot tell that the inputs of a build
// were spent for nothing.
//
// Requirement: FR-CAP-017, FR-CAP-039
func TestComplete_VoidedIsLoggedWithoutAnObserver(t *testing.T) {
	t.Parallel()
	var recs []slog.Record
	s := &bldScript{}
	w := newMill(t, bldConfig(nil), WithSeed(facSeed), WithStrategy("script", s), WithLogger(slog.New(mkLogs{&recs})))
	m := bldMiller(w, 0)
	bldAsk(t, w, s, "zz_build", 5)
	facStep(t, w)
	bldAdd(t, w, clsMill, m, MaxFacilitiesPerAgent-2)
	facStep(t, w)
	var found int
	for _, r := range recs {
		if r.Message != "facility executions voided" {
			continue
		}
		found++
		a := logAttrs(r)
		if got := a[logkeys.Executions].Int64(); got != 3 {
			t.Errorf("voided line %q = %d, want 3 (5 executions, room for 2)", logkeys.Executions, got)
		}
		if got := a[logkeys.FacilityClass].String(); got != w.cfg.Modules.Infrastructure.Classes[clsMill].ID {
			t.Errorf("voided line %q = %q, want the class of the build", logkeys.FacilityClass, got)
		}
		if r.Level != slog.LevelDebug {
			t.Errorf("voided line level = %v, want Debug", r.Level)
		}
	}
	if found != 1 {
		t.Errorf("lines %q = %d, want 1", "facility executions voided", found)
	}
}

// TestFacility_RuinLineCarriesTheCause verifies that the Debug line of a ruin names its cause:
// a facility at condition 1 with a decay above 1 is ruined by decay, as the event says.
//
// Guard: decay and use ruins point at different faults and the event is absent without an
// observer; the line is the only trace then.
//
// Requirement: FR-CAP-039
func TestFacility_RuinLineCarriesTheCause(t *testing.T) {
	t.Parallel()
	var recs []slog.Record
	w := facWorld(t, facDecaying, WithLogger(slog.New(mkLogs{&recs})))
	facPlant(w, clsBarn, 0, facAgents(w, 0, "weaver")[0], 1)
	recs = nil
	facStep(t, w)
	var got string
	for _, r := range recs {
		if r.Message == "facility ruined" {
			got = logAttrs(r)[logkeys.Cause].String()
		}
	}
	if got != string(CauseDecay) {
		t.Errorf("cause of the ruin line = %q, want %q", got, CauseDecay)
	}
}

// TestStart_RefusalIsLoggedWithoutAnObserver verifies that a build refused whole for the
// owner's cap writes the Debug line "facility start refused" with the tick, the agent, the
// recipe and the reason, though no observer is supplied and so Refused is not counted.
//
// Arithmetic: the miller owns MaxFacilitiesPerAgent mills, so the room is 0 and the build is
// refused facility_cap (as TestStart_Limits_Order); it holds grain, so no_input does not come
// first.
//
// Guard: without the line a run with only a logger cannot tell why a build never started.
//
// Requirement: FR-CAP-012, FR-CAP-039
func TestStart_RefusalIsLoggedWithoutAnObserver(t *testing.T) {
	t.Parallel()
	var recs []slog.Record
	s := &bldScript{}
	c := bldConfig(func(c *Config) { c.Roles[1].StartStock = []Item{{Commodity: "grain", Quantity: 10}} })
	w := newMill(t, c, WithSeed(facSeed), WithStrategy("script", s), WithLogger(slog.New(mkLogs{&recs})))
	m := bldMiller(w, 0)
	bldAdd(t, w, clsMill, m, MaxFacilitiesPerAgent)
	bldAsk(t, w, s, "zz_build", 5)
	facStep(t, w)
	var found int
	for _, r := range recs {
		if r.Message != "facility start refused" {
			continue
		}
		found++
		a := logAttrs(r)
		if got := Reason(a[logkeys.Reason].String()); got != ReasonFacilityCap {
			t.Errorf("refusal line %q = %q, want %q", logkeys.Reason, got, ReasonFacilityCap)
		}
		if got := AgentID(a[logkeys.Agent].Int64()); got != m {
			t.Errorf("refusal line %q = %d, want the miller %d", logkeys.Agent, got, m)
		}
		if r.Level != slog.LevelDebug {
			t.Errorf("refusal line level = %v, want Debug", r.Level)
		}
	}
	if found < 1 {
		t.Errorf("lines %q = %d, want at least 1 (seed %d)", "facility start refused", found, facSeed)
	}
}
