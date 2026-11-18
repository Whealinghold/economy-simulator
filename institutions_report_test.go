package economysim_test

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of what the institutions module shows and saves (spec-institutions.md 11, 12):
// the events of P7.2b, the snapshot rows, the logs, tag 31 and the fork at N. They build
// the world of institutionsConfig with 40 places for the guild instead of 60, so that 15 of
// the 55 makers of north wait for a place every tick.

// orgWaitingConfig is institutionsConfig with the guild's places lowered to 40. North's 55
// makers meet 40 places: 40 admitted at the first round, 15 skipped and waiting in every
// round after it.
func orgWaitingConfig() economysim.Config {
	c := institutionsConfig()
	c.Modules.Institutions.Bodies[0].Places = 40
	return c
}

// orgNorth is the RegionID of north in institutionsConfig: the six regions are in the
// order of their IDs (capital, farm, harbour, hills, mill, north).
const orgNorth economysim.RegionID = 5

// membershipEvents returns the MembershipChanged events of the recorder, in order.
func membershipEvents(rec *recorder) []economysim.MembershipChanged {
	var out []economysim.MembershipChanged
	for _, e := range rec.events {
		if m, ok := e.(economysim.MembershipChanged); ok {
			out = append(out, m)
		}
	}
	return out
}

// TestEvent_MembershipChangedFieldsAndGate verifies the event of the module: its fields at
// the first round and the rounds after it, that it is emitted only with an observer and
// the module on, and that a world with the module off emits none.
//
// Arithmetic: 55 makers, 40 places: tick 0 admits 40 and skips 15, leaving 15 waiting; the
// next ticks admit nobody and the 15 stay skipped and waiting.
//
// Requirement: FR-ORG-039, FR-ORG-001
func TestEvent_MembershipChangedFieldsAndGate(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := newPackWorld(t, orgWaitingConfig(), 1, economysim.WithObserver(rec))
	step(t, w, 3)
	want := []economysim.MembershipChanged{
		{Tick: 0, Region: orgNorth, Body: 0, Members: 40, Places: 40, Eligible: 55, Admitted: 40, Skipped: 15, Waiting: 15},
		{Tick: 1, Region: orgNorth, Body: 0, Members: 40, Places: 40, Eligible: 55, Skipped: 15, Waiting: 15},
		{Tick: 2, Region: orgNorth, Body: 0, Members: 40, Places: 40, Eligible: 55, Skipped: 15, Waiting: 15},
	}
	if got := membershipEvents(rec); !slices.Equal(got, want) {
		t.Errorf("MembershipChanged events = %+v, want %+v", got, want)
	}
	off := &recorder{}
	step(t, newPackWorld(t, referenceConfig(), 1, economysim.WithObserver(off)), 3)
	if got := membershipEvents(off); len(got) != 0 {
		t.Errorf("MembershipChanged events of a world with the module off = %v, want none", got)
	}
}

// TestEvents_P72bAfterSettlementBeforePhase8 verifies the place of the event in the tick:
// after the market's clearing and before TickEnded, once per tick for the pair.
//
// Requirement: FR-ORG-043
func TestEvents_P72bAfterSettlementBeforePhase8(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := newPackWorld(t, orgWaitingConfig(), 1, economysim.WithObserver(rec))
	rec.forget()
	step(t, w, 1)
	lastCleared, member, ended := -1, -1, -1
	for i, e := range rec.events {
		switch e.(type) {
		case economysim.MarketCleared:
			lastCleared = i
		case economysim.MembershipChanged:
			member = i
		case economysim.TickEnded:
			ended = i
		}
	}
	if !(lastCleared >= 0 && lastCleared < member && member < ended) {
		t.Errorf("event positions: last MarketCleared %d, MembershipChanged %d, TickEnded %d, want them in that order", lastCleared, member, ended)
	}
}

// TestSnapshot_BodiesLegalLevelsSuspended verifies the rows of the snapshot: one row for
// each (region, body) of the scope, the households as they are now, and no level or
// suspension while none is set.
//
// Arithmetic: before the first step nobody is a member and the 55 makers are eligible; after
// it 40 are members.
//
// Requirement: FR-ORG-044, FR-ORG-038
func TestSnapshot_BodiesLegalLevelsSuspended(t *testing.T) {
	t.Parallel()
	w := newPackWorld(t, orgWaitingConfig(), 1)
	for _, c := range []struct {
		ticks int
		want  economysim.BodyRow
	}{
		{0, economysim.BodyRow{Region: orgNorth, Body: 0, Members: 0, Eligible: 55, Places: 40}},
		{1, economysim.BodyRow{Region: orgNorth, Body: 0, Members: 40, Eligible: 55, Places: 40}},
	} {
		step(t, w, c.ticks)
		s := snapshotOf(t, w)
		if !slices.Equal(s.Bodies, []economysim.BodyRow{c.want}) {
			t.Errorf("Snapshot.Bodies at tick %d = %+v, want %+v", s.Tick, s.Bodies, c.want)
		}
		if s.LegalLevels != nil || s.Suspended != nil {
			t.Errorf("Snapshot.LegalLevels, Suspended = %v, %v, want none while no slot is set", s.LegalLevels, s.Suspended)
		}
	}
	if s := snapshotOf(t, newPackWorld(t, referenceConfig(), 1)); s.Bodies != nil {
		t.Errorf("Snapshot.Bodies with the module off = %v, want none", s.Bodies)
	}
}

// TestNames_BodiesRightsControls verifies the names table of the snapshot and of
// WorldCreated: the string ID of each body, right and control, and nothing with the module off.
//
// Requirement: FR-ORG-044, FR-ORG-029
func TestNames_BodiesRightsControls(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := newPackWorld(t, institutionsConfig(), 1, economysim.WithObserver(rec))
	s := snapshotOf(t, w)
	for _, n := range []struct {
		name      string
		got, want []string
	}{
		{"Bodies", s.Names.Bodies, []string{"guild"}},
		{"Rights", s.Names.Rights, []string{"make_tools", "sell_tools"}},
		{"Controls", s.Names.Controls, []string{"assize", "usury"}},
	} {
		if !slices.Equal(n.got, n.want) {
			t.Errorf("Snapshot.Names.%s = %v, want %v", n.name, n.got, n.want)
		}
	}
	created, ok := rec.events[0].(economysim.WorldCreated)
	if !ok || !slices.Equal(created.Names.Bodies, []string{"guild"}) {
		t.Errorf("WorldCreated.Names.Bodies = %v, want [guild]", created.Names.Bodies)
	}
	if off := snapshotOf(t, newPackWorld(t, referenceConfig(), 1)); off.Names.Bodies != nil || off.Names.Rights != nil || off.Names.Controls != nil {
		t.Errorf("names with the module off = %v, %v, %v, want none", off.Names.Bodies, off.Names.Rights, off.Names.Controls)
	}
}

// logsOf returns the records of the handler with the message.
func logsOf(h *captureHandler, msg string) []logLine {
	var out []logLine
	for _, l := range h.records() {
		if l.msg == msg {
			out = append(out, l)
		}
	}
	return out
}

// TestLogs_InstitutionsConstructionLine verifies that the one Info line of the module is the
// existing "world created" line, which gains the counts of bodies, rights and controls and
// the module in its list, and that the module logs no other Info.
//
// Arithmetic: one body, two rights, and the controls count the control and the rate limit: 2.
//
// Requirement: FR-ORG-046
func TestLogs_InstitutionsConstructionLine(t *testing.T) {
	t.Parallel()
	h := newCaptureHandler()
	w := newPackWorld(t, institutionsConfig(), 1, economysim.WithLogger(slog.New(h)))
	step(t, w, 3)
	lines := logsOf(h, "world created")
	if len(lines) != 1 {
		t.Fatalf("\"world created\" lines = %d, want 1", len(lines))
	}
	for key, want := range map[string]int64{"bodies": 1, "rights": 2, "controls": 2} {
		if v, ok := lines[0].attr(key); !ok || v.Int64() != want {
			t.Errorf("\"world created\" attribute %s = %v, %v, want %d", key, v, ok, want)
		}
	}
	if v, ok := lines[0].attr("modules"); !ok || !strings.Contains(fmt.Sprint(v.Any()), "institutions") {
		t.Errorf("\"world created\" attribute modules = %v, want institutions in it", v)
	}
	if got := infoMessages(h); !slices.Equal(got, []string{"warm start applied", "world created"}) {
		t.Errorf("Info lines of three ticks = %v, want only the two lines of construction (the warm start and the world), none of the module's", got)
	}
}

// infoMessages returns the messages of the Info records of the handler, in order.
func infoMessages(h *captureHandler) []string {
	var out []string
	for _, l := range h.records() {
		if l.level == slog.LevelInfo {
			out = append(out, l.msg)
		}
	}
	return out
}

// TestLogs_BodyWaitingWarn verifies the Warn of a body with waiting households: one for the
// pair at the tenth tick that has waiting households and at every tenth after, with the
// region, the body, the waiting households and the places, and none before; and that its
// counterpart is MembershipChanged.Waiting.
//
// Arithmetic: the run counts from tick 0, so the tenth tick is tick 9 and the next tick 19.
//
// Requirement: FR-ORG-045
func TestLogs_BodyWaitingWarn(t *testing.T) {
	t.Parallel()
	h := newCaptureHandler()
	rec := &recorder{}
	w := newPackWorld(t, orgWaitingConfig(), 1, economysim.WithLogger(slog.New(h)), economysim.WithObserver(rec))
	step(t, w, 9)
	if got := logsOf(h, "body has waiting households"); len(got) != 0 {
		t.Fatalf("Warn lines after 9 ticks = %d, want none", len(got))
	}
	step(t, w, 1)
	got := logsOf(h, "body has waiting households")
	if len(got) != 1 || got[0].level != slog.LevelWarn {
		t.Fatalf("Warn lines after 10 ticks = %v, want one at Warn", got)
	}
	for key, want := range map[string]int64{"region": int64(orgNorth), "body": 0, "waiting": 15, "places": 40} {
		if v, ok := got[0].attr(key); !ok || v.Int64() != want {
			t.Errorf("Warn attribute %s = %v, %v, want %d", key, v, ok, want)
		}
	}
	step(t, w, 10)
	if got := logsOf(h, "body has waiting households"); len(got) != 2 {
		t.Errorf("Warn lines after 20 ticks = %d, want 2 (ticks 9 and 19)", len(got))
	}
	if last := membershipEvents(rec); last[len(last)-1].Waiting != 15 {
		t.Errorf("MembershipChanged.Waiting of the last tick = %d, want 15, the counterpart of the Warn", last[len(last)-1].Waiting)
	}
}

// orgSection returns the body of tag 31 of a checkpoint.
func orgSection(t *testing.T, raw []byte) []byte {
	t.Helper()
	for _, s := range decodeFrame(t, raw).sections {
		if s.tag == 31 {
			return s.body
		}
	}
	t.Fatalf("the checkpoint has no tag 31")
	return nil
}

// TestCheckpoint_InstitutionsSectionIsDenseAndAbsentWhenOff verifies tag 31: a world with the
// module on writes the three lists, with zero counts for the levels and the suspensions that
// nothing set, and the sparse run of the Warn, and a world with the module off writes no
// section.
//
// Arithmetic: after 3 ticks the pair (north, guild) has waited 3 ticks: no level (0), no
// suspension (0), one run row (1) of region 5, body 0, run 3.
//
// Requirement: FR-ORG-033, FR-ORG-038, FR-CHK-023
func TestCheckpoint_InstitutionsSectionIsDenseAndAbsentWhenOff(t *testing.T) {
	t.Parallel()
	on := newPackWorld(t, orgWaitingConfig(), 1)
	step(t, on, 3)
	if got, want := orgSection(t, mustMarshal(t, mustCheckpoint(t, on))), []byte{0, 0, 1, 5, 0, 3}; !slices.Equal(got, want) {
		t.Errorf("tag 31 = %v, want %v", got, want)
	}
	off := newPackWorld(t, referenceConfig(), 1)
	step(t, off, 3)
	for _, s := range decodeFrame(t, mustMarshal(t, mustCheckpoint(t, off))).sections {
		if s.tag == 31 {
			t.Errorf("tag 31 of the world with the module off = %v, want no section", s.body)
		}
	}
}

// TestCheckpoint_InstitutionsRefusesCorrupt verifies that the decoder of tag 31 refuses each
// way the section can be wrong, as ErrCheckpointCorrupt naming section 31, and that the good
// body restores: a level for a control that equals the configured one, one out of bound or
// for a control that is not defined, a right that is not defined, rows out of order, a run
// of a pair outside the scope or above the tick, a section that ends early or has bytes
// after its last row, a section missing and a section present with the module off.
//
// Arithmetic: at tick 3 the largest run is 3; there are two controls (0 and 1), two rights
// (0 and 1) and six regions (the only pair in scope is region 5, body 0); the configured levels
// of control 0 (assize) are 12 000 and 8 000, and a cap above MaxBasisPoints (1 000 000) is
// out of bound.
//
// Requirement: FR-ORG-050, FR-CHK-021
func TestCheckpoint_InstitutionsRefusesCorrupt(t *testing.T) {
	t.Parallel()
	cfg := orgWaitingConfig()
	w := newPackWorld(t, cfg, 1)
	step(t, w, 3)
	raw := mustMarshal(t, mustCheckpoint(t, w))
	tests := []struct {
		name string
		body []byte
	}{
		{"a level equal to the configured one", []byte{1, 0, 0xE0, 0x5D, 0xC0, 0x3E, 0, 1, 5, 0, 3}},
		{"a level above the bound", []byte{1, 0, 0xC1, 0x84, 0x3D, 0, 0, 1, 5, 0, 3}},
		{"a control that is not defined", []byte{1, 9, 1, 0, 0, 1, 5, 0, 3}},
		{"a rate limit with a floor", []byte{1, 1, 1, 1, 0, 1, 5, 0, 3}},
		{"a right that is not defined", []byte{0, 1, 9, 1, 5, 0, 3}},
		{"rights out of order", []byte{0, 2, 1, 0, 1, 5, 0, 3}},
		{"a run of a pair outside the scope", []byte{0, 0, 1, 4, 0, 3}},
		{"a run above the tick", []byte{0, 0, 1, 5, 0, 4}},
		{"a run of 0", []byte{0, 0, 1, 5, 0, 0}},
		{"a section that ends early", []byte{0, 0, 1, 5, 0}},
		{"an empty section", nil},
		{"bytes after the last row", []byte{0, 0, 1, 5, 0, 3, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, func(s []section) []section {
				return editSection(s, 31, func([]byte) []byte { return tt.body })
			})
			_, err := economysim.RestoreWorld(cfg, c)
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) || !strings.Contains(err.Error(), "section 31") {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt naming section 31", tt.name, err)
			}
		})
	}
	t.Run("a section missing", func(t *testing.T) {
		t.Parallel()
		c := reframe(t, raw, func(s []section) []section {
			return slices.DeleteFunc(s, func(x section) bool { return x.tag == 31 })
		})
		if _, err := economysim.RestoreWorld(cfg, c); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
			t.Errorf("RestoreWorld(no section 31) error = %v, want ErrCheckpointCorrupt", err)
		}
	})
	t.Run("a section with the module off", func(t *testing.T) {
		t.Parallel()
		off := newPackWorld(t, referenceConfig(), 1)
		step(t, off, 3)
		c := reframe(t, mustMarshal(t, mustCheckpoint(t, off)), func(s []section) []section {
			i := slices.IndexFunc(s, func(x section) bool { return x.tag > 31 })
			return slices.Insert(s, i, section{tag: 31, body: []byte{0, 0, 0}})
		})
		if _, err := economysim.RestoreWorld(referenceConfig(), c); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
			t.Errorf("RestoreWorld(a section 31 with the module off) error = %v, want ErrCheckpointCorrupt", err)
		}
	})
	good := reframe(t, raw, func(s []section) []section {
		return editSection(s, 31, func([]byte) []byte { return []byte{0, 0, 1, 5, 0, 2} })
	})
	if _, err := economysim.RestoreWorld(cfg, good); err != nil {
		t.Errorf("RestoreWorld(the body with a run of 2) error = %v, want nil", err)
	}
}

// FuzzDecodeInstitutions feeds the decoder of tag 31 arbitrary bodies in place of the
// section and checks that the restore never panics and answers with nothing or with
// ErrCheckpointCorrupt, and that an allocation bound holds: a count above the bytes remaining
// is refused before any slice is made.
//
// Requirement: FR-ORG-050
func FuzzDecodeInstitutions(f *testing.F) {
	cfg := orgWaitingConfig()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		f.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for range 3 {
		if err := w.Step(); err != nil {
			f.Fatalf("Step() error = %v, want nil", err)
		}
	}
	ck, err := w.Checkpoint()
	if err != nil {
		f.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	raw, err := ck.MarshalBinary()
	if err != nil {
		f.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	for _, seed := range [][]byte{{0, 0, 1, 5, 0, 3}, {0, 0, 0}, {1, 0, 1, 0, 0, 0, 0}, {0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}, nil} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		c := reframe(t, raw, func(s []section) []section { return editSection(s, 31, func([]byte) []byte { return body }) })
		if _, err := economysim.RestoreWorld(cfg, c); err != nil && !errors.Is(err, economysim.ErrCheckpointCorrupt) {
			t.Errorf("RestoreWorld(tag 31 = %v) error = %v, want nil or ErrCheckpointCorrupt", body, err)
		}
	})
}

// TestFork_Institutions verifies the fork at N of a world with the module on: at each N of the
// cases the original and two restores give the same events, snapshot, audit, checkpoint bytes
// and logs for 30 ticks, so a restored world admits, lapses and warns as the uninterrupted
// one. The cases are a world before its first round, one after it, one with a membership
// made stale by a move that the next round lapses, and one with a Warn run in progress.
//
// Arithmetic: the Warn fires at the tenth waiting tick; a fork at N 7 saves a run of 7, and
// the next 30 ticks include the warnings of ticks 9 and 19 and 29.
//
// Requirement: FR-ORG-033, FR-CHK-050
//
// Guard: the membership is in the agent record and the run is in tag 31; a restore that lost
// either would admit again, lapse late or warn at the wrong tick.
func TestFork_Institutions(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := orgWaitingConfig()
	move := func(w *economysim.World) {
		snap, err := w.Snapshot()
		if err != nil {
			panic("Snapshot: " + err.Error()) // the world of the test is whole; Before has no *testing.T
		}
		maker := economysim.RoleID(slices.Index(snap.Names.Roles, "maker"))
		ids, err := w.Agents(orgNorth)
		if err != nil {
			panic("Agents: " + err.Error())
		}
		for _, id := range ids {
			if st, err := w.Agent(id); err == nil && st.Role == maker {
				if err := w.Relocate(id, 0); err != nil {
					panic("Relocate: " + err.Error())
				}
				return
			}
		}
		panic("no maker found in north")
	}
	for _, c := range []struct {
		name   string
		n      int
		before func(*economysim.World)
	}{
		{"before the first round", 0, nil},
		{"after the first round", 1, nil},
		{"a membership made stale by a move", 3, move},
		{"a Warn run in progress", 7, nil},
	} {
		forkAtN(t, forkCase{Name: "institutions " + c.name, Cfg: cfg, Seed: 1, N: c.n, M: forkM, Logs: true, Before: c.before})
	}
}

// TestDeterminism_InstitutionsNonInterference verifies that a logger and an observer change
// nothing: the same seed and configuration end in the same snapshot and state hash with
// and without them, and the events of the module are the same on a second run.
//
// Requirement: NFR-ORG-003, NFR-ORG-004, FR-ORG-053
func TestDeterminism_InstitutionsNonInterference(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := orgWaitingConfig()
	bare := newPackWorld(t, cfg, 2)
	rec := &recorder{}
	watched := newPackWorld(t, cfg, 2, economysim.WithObserver(rec), economysim.WithLogger(slog.New(newCaptureHandler())))
	step(t, bare, 40)
	step(t, watched, 40)
	if !reflect.DeepEqual(snapshotOf(t, bare), snapshotOf(t, watched)) || economysim.HashWorld(bare) != economysim.HashWorld(watched) {
		t.Errorf("the world with a logger and an observer differs from the bare one after 40 ticks")
	}
	again := &recorder{}
	step(t, newPackWorld(t, cfg, 2, economysim.WithObserver(again)), 40)
	if !slices.Equal(membershipEvents(rec), membershipEvents(again)) {
		t.Errorf("the MembershipChanged events of two runs of one seed differ")
	}
}

// TestEmptyModule_EqualsOff verifies that the module present with nothing declared leaves no
// trace: the world runs as the world without the module, snapshot for snapshot, apart from
// the hash and the list of modules, and emits no event of its own.
//
// Requirement: FR-ORG-001
//
// Guard: the first of the module's outputs that leaked into a world with no body would fail here.
func TestEmptyModule_EqualsOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, seed := range seedList(invariantSeeds(2)) {
		off := referenceConfig()
		on := referenceConfig()
		on.Modules.Institutions = &economysim.Institutions{}
		rec := &recorder{}
		a, b := newPackWorld(t, off, seed), newPackWorld(t, on, seed, economysim.WithObserver(rec))
		step(t, a, 30)
		step(t, b, 30)
		sa, sb := snapshotOf(t, a), snapshotOf(t, b)
		sa.ConfigHash, sb.ConfigHash, sa.Modules, sb.Modules = [32]byte{}, [32]byte{}, nil, nil
		if !reflect.DeepEqual(sa, sb) {
			t.Errorf("seed %d: the world with the empty module differs from the world without it", seed)
		}
		if got := membershipEvents(rec); len(got) != 0 {
			t.Errorf("seed %d: MembershipChanged events of an empty module = %v, want none", seed, got)
		}
	}
}

// TestEvents_MetricsDerivable verifies that the instruments of spec-institutions.md 11.6 that
// the module feeds so far are derivable: the members of a body from the event and the
// snapshot agree, and the waiting households are the eligible less the members, as the
// instrument folds them.
//
// Requirement: FR-ORG-055
func TestEvents_MetricsDerivable(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := newPackWorld(t, orgWaitingConfig(), 1, economysim.WithObserver(rec))
	step(t, w, 2)
	events := membershipEvents(rec)
	last := events[len(events)-1]
	s := snapshotOf(t, w)
	if len(s.Bodies) != 1 || s.Bodies[0].Members != last.Members {
		t.Fatalf("Snapshot.Bodies = %+v, event members %d, want the same members", s.Bodies, last.Members)
	}
	if got := s.Bodies[0].Eligible - s.Bodies[0].Members; got != 15 || last.Waiting != 15 {
		t.Errorf("waiting from the snapshot = %d and from the event = %d, want 15 and 15", got, last.Waiting)
	}
}
