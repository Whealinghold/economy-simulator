package economysim

import (
	"log/slog"
	"math/rand/v2"
	"reflect"
	"testing"
)

// The tests of what an actor reads and sets at P0.3a (spec-policy.md 5, 6, 12; plan 3-plc
// Phase 5): the view, the checks and their counts, the application in the order of 6.4.2,
// the event and the log lines, the schedule and the rule. The arithmetic of the checks, the
// schedule and the rule is tested in internal/policy.

// scriptedPolicy is a policy the test controls: it returns set at every call and keeps the
// last view it was given.
type scriptedPolicy struct {
	set  Settings
	view PolicyView
}

func (p *scriptedPolicy) Decide(v PolicyView, _ *rand.Rand) Settings {
	p.view = v
	return p.set
}

// scheduledCrown is the crown actor on a schedule that sets all three levers at tick 0.
func scheduledCrown(every int64) PolicyActor {
	a := crownActor("schedule", every)
	a.Schedule = []ScheduleStep{
		{Lever: "gate", At: 0, BP: 15_000},
		{Lever: "poll", At: 0, BP: 12_000},
		{Lever: "road", At: 0, BP: 5_000},
	}
	return a
}

// policyEvents returns the types of the events after the first TickStarted of the log, up to
// and including the first PolicyDecided, or all of them when there is none.
func policyEvents(rec *evLog) []string {
	var out []string
	started := false
	for _, e := range rec.events {
		switch e.(type) {
		case TickStarted:
			started = true
		case PolicyDecided:
			return append(out, reflect.TypeOf(e).Name())
		default:
			if started {
				out = append(out, reflect.TypeOf(e).Name())
			}
		}
	}
	return out
}

// TestP03a_AppliesInTheOrderOfKindsAfterTickStarted verifies the order of 6.4.2 and 12.1.3:
// after TickStarted and before any event of phase 1, the edge event, the transit event, the
// flow event, then PolicyDecided, each slot event with the cause policy.
//
// Requirement: FR-PLC-043, FR-PLC-044, FR-PLC-080, FR-PLC-084
func TestP03a_AppliesInTheOrderOfKindsAfterTickStarted(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, policyMillConfig(scheduledCrown(10)), WithSeed(3), WithObserver(rec))
	rec.events = nil
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	got := policyEvents(rec)
	want := []string{"EdgeStatusChanged", "TransitFactorChanged", "MoneyFlowScaled", "PolicyDecided"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events after TickStarted = %v, want %v", got, want)
	}
	checkScheduledSlotEvents(t, rec)
}

// checkScheduledSlotEvents checks the three slot events and PolicyDecided of the first decision
// of scheduledCrown.
func checkScheduledSlotEvents(t *testing.T, rec *evLog) {
	t.Helper()
	if got, want := of[EdgeStatusChanged](rec)[0], (EdgeStatusChanged{From: 0, To: 1, FactorBP: 15_000, Cause: CausePolicy}); got != want {
		t.Errorf("EdgeStatusChanged = %+v, want %+v", got, want)
	}
	if got, want := of[TransitFactorChanged](rec)[0], (TransitFactorChanged{From: 0, To: 1, BeforeBP: 10_000, AfterBP: 5_000, Cause: CausePolicy}); got != want {
		t.Errorf("TransitFactorChanged = %+v, want %+v", got, want)
	}
	if got, want := of[MoneyFlowScaled](rec)[0], (MoneyFlowScaled{Flow: "poll", From: 10_000, To: 12_000, Cause: CausePolicy}); got != want {
		t.Errorf("MoneyFlowScaled = %+v, want %+v", got, want)
	}
	if got, want := of[PolicyDecided](rec)[0], (PolicyDecided{Entries: 3, Applied: 3}); got != want {
		t.Errorf("PolicyDecided = %+v, want %+v", got, want)
	}
}

// TestApply_IdleDecisionIsSilentSlotsButCounted verifies 6.2.4 and 8.2.2: a schedule that
// returns the value every lever already has gets unchanged entries, no write and no slot
// event, and a PolicyDecided that says so.
//
// Requirement: FR-PLC-031, FR-PLC-042
func TestApply_IdleDecisionIsSilentSlotsButCounted(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, policyMillConfig(scheduledCrown(1)), WithSeed(3), WithObserver(rec))
	for range 2 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if n := len(of[MoneyFlowScaled](rec)) + len(of[EdgeStatusChanged](rec)) + len(of[TransitFactorChanged](rec)); n != 3 {
		t.Errorf("slot events after two decisions = %d, want 3 (the second decision wrote nothing)", n)
	}
	got := of[PolicyDecided](rec)
	if len(got) != 2 || got[1].Unchanged != 3 || got[1].Applied != 0 || got[1].Entries != 3 {
		t.Errorf("PolicyDecided = %+v, want a second one with 3 unchanged entries", got)
	}
}

// TestApply_ChecksCountEveryOutcome verifies the outcomes of 6.2.1 in the world with a
// scripted policy: an unknown lever, a repeated lever, a closing the lever may not make, a
// value cut by the bounds and a value cut by the step; the counts, the sum of Entries and
// what was written.
//
// Requirement: FR-PLC-024, FR-PLC-026, FR-PLC-027, FR-PLC-028, FR-PLC-029, FR-PLC-030, FR-PLC-082
func TestApply_ChecksCountEveryOutcome(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, policyMillConfig(crownActor("schedule", 1)), WithSeed(3), WithObserver(rec))
	p := &scriptedPolicy{set: Settings{Levers: []LeverSetting{
		{Lever: 7, BP: 1},                   // not held
		{Lever: 1, BP: 99_999},              // poll: bounds cut to 20 000, then the step cut to 13 000
		{Lever: 1, BP: 11_000},              // duplicate
		{Lever: 2, BP: 4_000, Closed: true}, // road is not an edge: no_close
		{Lever: 0, BP: 1_000, Closed: true}, // gate: closed, factor cut to its minimum 5 000
	}}}
	w.policy.actors[0].builtin = p
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	d := of[PolicyDecided](rec)[0]
	want := PolicyDecided{Tick: 0, Entries: 5, Applied: 2, NotHeld: 1, Duplicate: 1, NoClose: 1, Clamped: 2, Stepped: 1}
	if d != want {
		t.Errorf("PolicyDecided = %+v, want %+v", d, want)
	}
	if e := of[MoneyFlowScaled](rec)[0]; e.To != 13_000 {
		t.Errorf("flow poll set to %d, want 13 000 (cut by the bounds, then by the step)", e.To)
	}
	if e := of[EdgeStatusChanged](rec)[0]; !e.Closed || e.FactorBP != 5_000 || e.Cause != CausePolicy {
		t.Errorf("EdgeStatusChanged = %+v, want closed at 5 000 by the policy", e)
	}
	if len(of[TransitFactorChanged](rec)) != 0 {
		t.Errorf("the refused entry for the road wrote a transit factor")
	}
}

// TestView_RegionsEqualTheRegionStateOfTheBoundary verifies FR-OBS-044 and D6 in the world:
// two actors that read regions at one pass, the second sharing the cost anchors the first
// prepared, get exactly the states Region gives for the covered regions at the boundary, the
// treasury, the seat and the levers' current values.
//
// Requirement: FR-PLC-021, FR-PLC-022, FR-PLC-025, FR-PLC-032
func TestView_RegionsEqualTheRegionStateOfTheBoundary(t *testing.T) {
	t.Parallel()
	crown, shire := crownActor("schedule", 1), shireActor("schedule", 1)
	crown.ReadsRegions, shire.ReadsRegions = true, true
	w := newMill(t, policyMillConfig(crown, shire), WithSeed(5))
	var want [2]RegionState
	for g := range want {
		var err error
		if want[g], err = w.Region(RegionID(g)); err != nil {
			t.Fatalf("Region(%d) error = %v, want nil", g, err)
		}
	}
	a, b := &scriptedPolicy{}, &scriptedPolicy{}
	w.policy.actors[0].builtin, w.policy.actors[1].builtin = a, b
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(a.view.Regions, []RegionState{want[0], want[1]}) || !reflect.DeepEqual(b.view.Regions, []RegionState{want[1]}) {
		t.Errorf("views' regions differ from Region(g) of the boundary")
	}
	checkCrownView(t, a.view)
}

// checkCrownView checks the fields of the crown's view at tick 0 other than its regions.
func checkCrownView(t *testing.T, v PolicyView) {
	t.Helper()
	type head struct {
		tick, actor, seat int64
		levers            int
		governed          bool
		treasury          Money
	}
	got := head{v.Tick, int64(v.Actor), int64(v.Seat), len(v.Levers), v.HasTreasury, v.Treasury}
	if want := (head{0, 0, 0, 3, true, 1_000}); got != want {
		t.Errorf("view = %+v, want %+v", got, want)
	}
	want := LeverView{Lever: 1, Kind: LeverFlow, BP: 10_000, Default: 10_000, Max: 20_000, MaxStep: 3_000}
	if v.Levers[1] != want {
		t.Errorf("lever poll view = %+v, want %+v", v.Levers[1], want)
	}
}

// TestRule_MovesALeverByTheTreasury verifies the rule in the world: with the treasury
// gauge, a reading above the target and Raise, the lever rises by the step in proportion to
// the error, inside its limits, and the decision is counted; a reading in the band is
// counted and writes nothing.
//
// Requirement: FR-PLC-060, FR-PLC-061, FR-PLC-062, FR-PLC-063
func TestRule_MovesALeverByTheTreasury(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name         string
		target, band int64
		wantTo       int64
		inBand       int64
	}{
		{"above the band", 400, 10, 13_000, 0}, // error 600, move 6 000 per 1 000: +3 600, cut to the step 3 000
		{"in the band", 1_000, 10, 0, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := crownActor("rule", 1)
			a.Rules = []Reaction{{Lever: "poll", Gauge: GaugeTreasury, Target: tt.target, Deadband: tt.band, Span: 1_000, Move: 6_000, Raise: true}}
			rec := &evLog{}
			w := newMill(t, policyMillConfig(a), WithSeed(3), WithObserver(rec))
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
			moved := of[MoneyFlowScaled](rec)
			d := of[PolicyDecided](rec)[0]
			if tt.wantTo == 0 {
				if len(moved) != 0 || d.InBand != tt.inBand || d.Entries != 0 {
					t.Errorf("in band: scaled %v, PolicyDecided %+v, want no write and one reaction in band", moved, d)
				}
				return
			}
			if len(moved) != 1 || moved[0].To != tt.wantTo || d.Applied != 1 || d.Stepped != 1 {
				t.Errorf("above band: scaled %v, PolicyDecided %+v, want 13 000, one applied and stepped", moved, d)
			}
		})
	}
}

// debugLines returns the captured lines with the message.
func debugLines(c capture, msg string) []logRec {
	var out []logRec
	for _, l := range *c.lines {
		if l.msg == msg {
			out = append(out, l)
		}
	}
	return out
}

// TestLog_DecisionLinesAtDebug verifies 12.2.1: the Debug lines "policy decided" and "policy
// entry refused" with their keys, and no line above Debug inside the tick.
//
// Requirement: FR-PLC-085
func TestLog_DecisionLinesAtDebug(t *testing.T) {
	t.Parallel()
	logs := newCapture()
	w := newMill(t, policyMillConfig(crownActor("schedule", 1)), WithSeed(3), WithLogger(slog.New(logs)))
	w.policy.actors[0].builtin = &scriptedPolicy{set: Settings{Levers: []LeverSetting{{Lever: 1, BP: 12_000}, {Lever: 9, BP: 1}}}}
	*logs.lines = nil
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	for _, l := range *logs.lines {
		if l.level > slog.LevelDebug {
			t.Errorf("line %q at level %v inside the tick, want Debug only", l.msg, l.level)
		}
	}
	checkDecisionLines(t, logs)
}

// checkDecisionLines checks the two lines of the scripted decision of TestLog_DecisionLinesAtDebug.
func checkDecisionLines(t *testing.T, logs capture) {
	t.Helper()
	type line struct {
		actor, reason    string
		entries, changed int64
	}
	var got []line
	for _, l := range debugLines(logs, "policy decided") {
		got = append(got, line{actor: l.strs["actor"], entries: l.attrs["entries"], changed: l.attrs["changed"]})
	}
	for _, l := range debugLines(logs, "policy entry refused") {
		got = append(got, line{actor: l.strs["actor"], reason: l.strs["reason"]})
	}
	want := []line{{actor: "crown_ctl", entries: 2, changed: 1}, {actor: "crown_ctl", reason: "not_held"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("decision lines = %+v, want %+v", got, want)
	}
}

// TestLog_RuleMovedLever verifies the Debug line "rule moved lever" with the reading and the
// value before and after.
//
// Requirement: FR-PLC-085
func TestLog_RuleMovedLever(t *testing.T) {
	t.Parallel()
	a := crownActor("rule", 1)
	a.Rules = []Reaction{{Lever: "poll", Gauge: GaugeTreasury, Target: 400, Span: 1_000, Move: 1_000, Raise: true}}
	logs := newCapture()
	w := newMill(t, policyMillConfig(a), WithSeed(3), WithLogger(slog.New(logs)))
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	got := debugLines(logs, "rule moved lever")
	if len(got) != 1 || got[0].strs["lever"] != "poll" || got[0].attrs["reading"] != 1_000 || got[0].attrs["from"] != 10_000 || got[0].attrs["to"] != 10_600 {
		t.Errorf("rule moved lever lines = %+v, want poll read 1 000 moved from 10 000 to 10 600", got)
	}
}

// TestDeterminism_PolicyNonInterference verifies FR-PLC-033: a world with an observer and an
// enabled logger and a world with neither reach the same snapshot and the same slots after
// 40 ticks of a schedule and a rule that read regions.
//
// Requirement: FR-PLC-033
func TestDeterminism_PolicyNonInterference(t *testing.T) {
	t.Parallel()
	crown, shire := scheduledCrown(7), shireActor("rule", 5)
	shire.ReadsRegions = true
	shire.Rules = []Reaction{{Lever: "back", Gauge: GaugeMeanRatio, Target: 20_000, Span: 1_000, Move: 500, Raise: false}}
	cfg := policyMillConfig(crown, shire)
	quiet := newMill(t, cfg, WithSeed(11), WithAudit())
	loud := newMill(t, cfg, WithSeed(11), WithAudit(), WithObserver(&evLog{}), WithLogger(slog.New(newCapture())))
	for tick := range 40 {
		for _, w := range []*World{quiet, loud} {
			if err := w.Step(); err != nil {
				t.Fatalf("tick %d: Step() error = %v, want nil", tick, err)
			}
		}
	}
	a, errA := quiet.Snapshot()
	b, errB := loud.Snapshot()
	if errA != nil || errB != nil || !reflect.DeepEqual(a, b) {
		t.Errorf("snapshots differ with and without an observer and logger (errors %v, %v)", errA, errB)
	}
	sa, _ := quiet.Slots()
	sb, _ := loud.Slots()
	if !reflect.DeepEqual(sa, sb) || len(sa.EdgeStatuses) == 0 {
		t.Errorf("slots = %+v and %+v, want equal and the schedule's edge status among them", sa, sb)
	}
}

// TestApply_AddsNoReasonNoAccount verifies FR-PLC-047 and D14: a world whose actor writes a
// flow scale has the accounts and ledger reasons of the same world with the module off.
//
// Requirement: FR-PLC-047
func TestApply_AddsNoReasonNoAccount(t *testing.T) {
	t.Parallel()
	on := policyMillConfig(scheduledCrown(1))
	off := on
	off.Modules.Policy = nil
	wa := newMill(t, on, WithSeed(2), WithAudit())
	wb := newMill(t, off, WithSeed(2), WithAudit())
	for range 5 {
		if err := wa.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if wa.ledger.Len() != wb.ledger.Len() {
		t.Errorf("accounts = %d with the module, %d without, want equal", wa.ledger.Len(), wb.ledger.Len())
	}
}

// TestInvariants_PolicyRunsAudited verifies that a world with a schedule and a rule that
// move all three kinds of lever passes the audit of every tick for 60 ticks.
//
// Requirement: FR-PLC-046, FR-PLC-047
func TestInvariants_PolicyRunsAudited(t *testing.T) {
	t.Parallel()
	crown, shire := scheduledCrown(3), shireActor("rule", 2)
	shire.ReadsRegions = true
	shire.Rules = []Reaction{{Lever: "back", Gauge: GaugeUnfedShare, Target: 0, Deadband: 0, Span: 100, Move: 1_000, Raise: false}}
	w := newMill(t, policyMillConfig(crown, shire), WithSeed(4), WithAudit())
	for tick := range 60 {
		if err := w.Step(); err != nil {
			t.Fatalf("tick %d: Step() error = %v, want nil", tick, err)
		}
	}
}

// TestCost_PolicyDueBuiltinsZeroAllocs verifies NFR-PLC-001 for a due schedule that finds
// every lever as it wants it: with the default logger and observer the decision allocates
// nothing once the scratch has grown.
//
// Requirement: NFR-PLC-001
func TestCost_PolicyDueBuiltinsZeroAllocs(t *testing.T) {
	// No t.Parallel: AllocsPerRun changes GOMAXPROCS while it measures.
	a := crownActor("schedule", 1)
	a.Schedule = []ScheduleStep{{Lever: "gate", At: 0, BP: 10_000}, {Lever: "poll", At: 0, BP: 10_000}, {Lever: "road", At: 0, BP: 10_000}}
	w := newMill(t, policyMillConfig(a), WithSeed(1))
	if got := testing.AllocsPerRun(50, w.p03aPolicy); got != 0 {
		t.Errorf("allocs of P0.3a with a due schedule that changes nothing = %v, want 0", got)
	}
}
