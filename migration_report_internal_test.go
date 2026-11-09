package economysim

import (
	"context"
	"log/slog"
	"reflect"
	"slices"
	"testing"

	mig "github.com/Whealinghold/economy-sim/internal/migration"
)

// The tests of the reporting side of the migration module (plan 2-14 Phase 7): the
// events, the logs, the snapshot rows and the checkpoint. They use the fixture of the
// pass tests (migration_pass_internal_test.go): three regions on a line, two millers
// in a and in b and two carriers in c, whose households trade at fixed prices.

// slowJourneys makes a journey from a to b take 2 ticks (40 at a speed of 20), so a
// household is on the way at a tick boundary.
func slowJourneys(c *Config) { c.Modules.Migration.MoveSpeed = 20 }

// TestEvents_MigrationStartedFields verifies FR-MIG-035: one MigrationStarted per
// departure in the order of admission, after the RecipeAbandoned of the departures,
// with the fields of spec-migration.md 13.1 (the arrival tick t + 1 + k, the graph's
// distance, k, the push, the home figure, the estimate of 69 611 and 78 561 of the
// worked fixture, and whether a job was abandoned), and no map.
//
// Requirement: FR-MIG-035, FR-MIG-026
func TestEvents_MigrationStartedFields(t *testing.T) {
	t.Parallel()
	w, rec := passWorld(t, 6, nil)
	open(w)
	for _, id := range []AgentID{3, 4} {
		w.agents[id].job = job{recipe: 0, executions: 1, remaining: 2}
	}
	rec.events = nil
	tick := w.Tick()
	w.p72Migration()
	started := of[MigrationStarted](rec)
	if len(started) != 2 || len(started) != len(w.migration.pass.started) {
		t.Fatalf("MigrationStarted events = %d, departures %d, want two of each", len(started), len(w.migration.pass.started))
	}
	for i, e := range started {
		rec := w.migration.pass.started[i]
		want := MigrationStarted{
			Tick: tick, Agent: rec.agent, Role: w.agents[rec.agent].role, From: rec.from, To: rec.to, Weight: 1,
			Arrives: tick + 2, Distance: 40, Journey: 1, EarnedRatio: rec.earned, Home: rec.home,
			Estimate: map[RegionID]int64{infoA: 69_611, infoB: 78_561}[rec.from], JobAbandoned: rec.from == infoA,
		}
		if e != want {
			t.Errorf("MigrationStarted %d = %+v, want %+v", i, e, want)
		}
	}
	// After the RecipeAbandoned of the departures, before every MigrationReviewed.
	lastAbandon, firstStart, lastStart, firstReview := -1, -1, -1, len(rec.events)
	for i, e := range rec.events {
		switch e.(type) {
		case RecipeAbandoned:
			lastAbandon = i
		case MigrationStarted:
			firstStart, lastStart = cmpFirst(firstStart, i), i
		case MigrationReviewed:
			firstReview = min(firstReview, i)
		}
	}
	if !(lastAbandon < firstStart && lastStart < firstReview) {
		t.Errorf("event positions: last RecipeAbandoned %d, starts %d to %d, first review %d, want abandoned < started < reviewed", lastAbandon, firstStart, lastStart, firstReview)
	}
	assertNoMapOrPointer(t, reflect.TypeFor[MigrationStarted]())
	assertNoMapOrPointer(t, reflect.TypeFor[MigrationReviewed]())
}

// cmpFirst is the first index seen: -1 means none yet.
func cmpFirst(first, i int) int {
	if first < 0 {
		return i
	}
	return first
}

// assertNoMapOrPointer fails when a field of the event type is a map, a pointer, a
// slice or an interface: events are plain data (observability-and-logging.md 4.1).
func assertNoMapOrPointer(t *testing.T, ty reflect.Type) {
	t.Helper()
	for i := range ty.NumField() {
		switch k := ty.Field(i).Type.Kind(); k {
		case reflect.Map, reflect.Pointer, reflect.Slice, reflect.Interface:
			t.Errorf("%s.%s is a %v, want plain data", ty.Name(), ty.Field(i).Name, k)
		}
	}
}

// TestEvents_MigrationReviewedPartition verifies FR-MIG-025, FR-MIG-036: one
// MigrationReviewed per region in ascending order, after the starts, only in a region
// where a household was pushed, held back by the settling rule, held at arrival or
// ended; the eleven outcomes add up to Pushed; the counts are households (an agent of
// weight 3 counts 3); and with every trigger zero nothing is emitted.
//
// Requirement: FR-MIG-025, FR-MIG-036, FR-MIG-043
func TestEvents_MigrationReviewedPartition(t *testing.T) {
	t.Parallel()
	w, rec := passWorld(t, 6, nil)
	// Shut gate: nobody is pushed, nobody unsettled, nothing held, nothing ended.
	rec.events = nil
	step(t, w, 3)
	if got := of[MigrationReviewed](rec); len(got) != 0 {
		t.Errorf("with the gate shut MigrationReviewed = %+v, want none", got)
	}
	// Open gate; the first miller of a stands for 3 households.
	open(w)
	w.agents[3].weight = 3
	w.agents[3].roleSince = -1_000
	rec.events = nil
	w.p72Migration()
	got := of[MigrationReviewed](rec)
	if len(got) != 3 {
		t.Fatalf("MigrationReviewed = %d events, want one per region with pushed households (3)", len(got))
	}
	for i, e := range got {
		if e.Region != RegionID(i) {
			t.Errorf("event %d is for region %d, want ascending regions", i, e.Region)
		}
		sum := e.Started + e.NoEstimate + e.NoDestination + e.NoRumour + e.TooFar + e.NoViable + e.NoGain +
			e.Bound + e.OutCap + e.Unprovisioned + e.InCap
		if sum != e.Pushed || e.Pushed != e.Reviewed {
			t.Errorf("region %d: outcomes add to %d, Pushed %d, Reviewed %d, want all equal", i, sum, e.Pushed, e.Reviewed)
		}
	}
	// Region a: the miller of weight 3 and the one of weight 1 are pushed: 4 households;
	// the cap of the outflow admits the poorer first and then stops (7.3).
	if got[0].Pushed != 4 {
		t.Errorf("region a Pushed = %d, want 4 households (agents of weight 3 and 1)", got[0].Pushed)
	}
	// Region c: the two carriers have no recipe: two households, no estimate.
	if got[2].Pushed != 2 || got[2].NoEstimate != 2 {
		t.Errorf("region c = %+v, want Pushed 2 and NoEstimate 2", got[2])
	}
	// An unsettled household alone triggers the event.
	w2, rec2 := passWorld(t, 6, nil)
	open(w2)
	w2.agents[3].roleSince = w2.Tick()
	rec2.events = nil
	w2.p72Migration()
	if r := of[MigrationReviewed](rec2); len(r) == 0 || r[0].Region != infoA || r[0].Unsettled != 1 {
		t.Errorf("MigrationReviewed with one unsettled miller = %+v, want region a first with Unsettled 1", r)
	}
	// A journey that ended alone triggers it too, and is reported once.
	w3, rec3 := passWorld(t, 6, nil)
	w3.migration.depart(3, infoB, 1_000)
	w3.migration.cancelJourney(&w3.agents[3], infoA)
	rec3.events = nil
	w3.p72Migration()
	w3.p72Migration()
	if r := of[MigrationReviewed](rec3); len(r) != 1 || r[0].Region != infoA || r[0].Ended != 1 {
		t.Errorf("MigrationReviewed after a cancelled journey = %+v, want one event, region a, Ended 1 (reported once)", r)
	}
}

// TestEvents_RebuildOnJourney verifies FR-MIG-037: the households on a journey are
// rebuilt each tick from MigrationStarted (+Weight), AgentRelocated with the cause
// migration (-Weight) and the Ended count (-), and equal the snapshot's rows, as does
// the household count of each region rebuilt from the creation and relocation events.
//
// Requirement: FR-MIG-037, FR-OBS-033
func TestEvents_RebuildOnJourney(t *testing.T) {
	t.Parallel()
	w, rec := passWorld(t, 6, slowJourneys)
	open(w)
	seen, onWay := 0, int64(0)
	sawFlight := false
	for range 10 {
		step(t, w, 1)
		if w.Tick() == 8 { // a journey is on the way: cancel one, as Relocate would
			for _, id := range w.migration.active {
				w.migration.cancelJourney(&w.agents[id], w.regionOf(&w.agents[id]))
				break
			}
		}
		for ; seen < len(rec.events); seen++ {
			switch e := rec.events[seen].(type) {
			case MigrationStarted:
				onWay += e.Weight
			case AgentRelocated:
				if e.Cause == CauseMigration {
					onWay -= e.Weight
				}
			case MigrationReviewed:
				onWay -= e.Ended
			}
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		var rows, from, to int64
		for _, j := range s.Journeys {
			rows += j.Weight
		}
		for _, r := range s.Regions {
			from, to = from+r.OnJourneyFrom, to+r.OnJourneyTo
		}
		sawFlight = sawFlight || rows > 0
		// The cancel at tick 8 is reported by the pass of the next step, so compare after it.
		if w.Tick() != 8 && (onWay != rows || from != rows || to != rows) {
			t.Errorf("tick %d: events rebuild %d households on the way, snapshot rows %d, from %d, to %d, want all equal", w.Tick(), onWay, rows, from, to)
		}
	}
	if !sawFlight {
		t.Errorf("no household was ever on a journey, want the rebuild to be exercised")
	}
	households := map[RegionID]int64{}
	for _, e := range rec.events {
		switch e := e.(type) {
		case AgentSpawned:
			households[e.Region] += e.Weight
		case AgentRelocated:
			households[e.From] -= e.Weight
			households[e.To] += e.Weight
		}
	}
	s, _ := w.Snapshot()
	for g, r := range s.Regions {
		if households[RegionID(g)] != r.Households {
			t.Errorf("region %d: events rebuild %d households, snapshot %d", g, households[RegionID(g)], r.Households)
		}
	}
}

// TestSnapshot_Journeys verifies FR-OBS-033 (spec-migration.md 13.4): the rows depend
// on the journeys in flight, ascending by agent, and not on the agents; the counts per
// region are households; the module off gives none.
//
// Requirement: FR-OBS-033, FR-MIG-028
func TestSnapshot_Journeys(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 2, nil)
	w.agents[6].weight = 4
	w.migration.depart(6, infoA, 99)
	w.migration.depart(3, infoC, 50)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	want := []JourneyRow{
		{Agent: 3, Role: w.agents[3].role, From: infoA, To: infoC, Arrives: 50, Weight: 1},
		{Agent: 6, Role: w.agents[6].role, From: infoB, To: infoA, Arrives: 99, Weight: 4},
	}
	if !slices.Equal(s.Journeys, want) {
		t.Errorf("Journeys = %+v, want %+v", s.Journeys, want)
	}
	from, to := []int64{1, 4, 0}, []int64{4, 0, 1}
	for g := range s.Regions {
		if s.Regions[g].OnJourneyFrom != from[g] || s.Regions[g].OnJourneyTo != to[g] {
			t.Errorf("region %d OnJourney from/to = %d/%d, want %d/%d", g, s.Regions[g].OnJourneyFrom, s.Regions[g].OnJourneyTo, from[g], to[g])
		}
	}
	// The account stays in its origin: the region's household count is unchanged.
	if s.Regions[infoB].Households != 4+2-1 {
		t.Errorf("region b Households = %d, want 5 (the migrant of weight 4 is still counted in its origin)", s.Regions[infoB].Households)
	}
	off := newMill(t, infoConfig(), WithStrategy("script", pairScript{}))
	if so, _ := off.Snapshot(); so.Journeys != nil {
		t.Errorf("Journeys with the module off = %+v, want nil", so.Journeys)
	}
}

// TestLogs_MigrationWarnAndDebug verifies FR-MIG-038: the bounded Warn lines at the
// tenth consecutive tick and every tenth after, with constant messages and the keys of
// 13.2, the Debug lines per agent, nothing at Info inside the tick, and no call of the
// handler when the level is not enabled.
//
// Requirement: FR-MIG-038
func TestLogs_MigrationWarnAndDebug(t *testing.T) {
	t.Parallel()
	lc := newCapture()
	w, _ := passWorld(t, 3, slowJourneys, WithLogger(slog.New(lc)))
	*lc.lines = nil
	// Agent 5 is on its way to a and its arrival is held by an obligation of the test.
	w.bounds.add("test_hold", func(a *agent, _ []string) bool { return a.id == 5 })
	w.migration.depart(5, infoA, w.Tick())
	open(w)
	step(t, w, 10)
	var blockedC, held []logRec
	var declined, started, heldDebug int
	for _, l := range *lc.lines {
		switch {
		case l.level == slog.LevelInfo:
			t.Errorf("Info line %q inside the tick, want none", l.msg)
		case l.msg == "migration blocked" && l.attrs["region"] == int64(infoC):
			blockedC = append(blockedC, l)
		case l.msg == "arrivals held":
			held = append(held, l)
		case l.msg == "migration declined":
			declined++
		case l.msg == "migration started":
			started++
			for _, k := range []string{"agent", "from_region", "to_region", "households", "lag"} {
				if _, ok := l.attrs[k]; !ok {
					t.Errorf("migration started has no %q key: %+v", k, l.attrs)
				}
			}
		case l.msg == "arrival held":
			heldDebug++
			if l.strs["reason"] == "" || l.attrs["agent"] != 5 {
				t.Errorf("arrival held = %+v, want agent 5 and a reason", l)
			}
		}
	}
	if len(blockedC) != 1 || blockedC[0].level != slog.LevelWarn || blockedC[0].attrs["households"] != 2 || blockedC[0].strs["reason"] != string(ReasonNoEstimate) {
		t.Errorf("migration blocked lines of region c = %+v, want one Warn at the tenth tick with households 2 and reason no_estimate", blockedC)
	}
	if len(held) != 1 || held[0].level != slog.LevelWarn || held[0].attrs["region"] != int64(infoB) || held[0].attrs["households"] != 1 {
		t.Errorf("arrivals held lines = %+v, want one Warn for region b with households 1", held)
	}
	if declined == 0 || started == 0 || heldDebug != 10 {
		t.Errorf("Debug lines: declined %d, started %d, arrival held %d, want some, some and 10", declined, started, heldDebug)
	}
	// The next ten ticks log the next lines, and only those.
	*lc.lines = nil
	step(t, w, 10)
	n := 0
	for _, l := range *lc.lines {
		if l.msg == "migration blocked" && l.attrs["region"] == int64(infoC) || l.msg == "arrivals held" {
			n++
		}
	}
	if n != 2 {
		t.Errorf("ticks 11 to 20 logged %d blocked or held Warns for regions c and b, want 2 (every tenth)", n)
	}
}

// failHandler is a slog handler that enables no level and fails the test if it is
// called.
type failHandler struct{ t *testing.T }

func (failHandler) Enabled(context.Context, slog.Level) bool { return false }
func (h failHandler) Handle(context.Context, slog.Record) error {
	h.t.Errorf("the handler was called although no level is enabled")
	return nil
}
func (h failHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h failHandler) WithGroup(string) slog.Handler      { return h }

// TestLogs_MigrationEnabledGuards verifies that no line is built or handled when the
// level is off, and that the world's results do not depend on whether anything
// watches (spec-migration.md 13.5, plan Observability).
//
// Requirement: FR-MIG-038, NFR-MIG-001
func TestLogs_MigrationEnabledGuards(t *testing.T) {
	t.Parallel()
	run := func(opts ...Option) Snapshot {
		w, _ := passWorld(t, 3, slowJourneys, opts...)
		w.bounds.add("test_hold", func(a *agent, _ []string) bool { return a.id == 5 })
		w.migration.depart(5, infoA, w.Tick())
		open(w)
		step(t, w, 24)
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		return s
	}
	quiet := run(WithLogger(slog.New(failHandler{t})))
	loud := run(WithLogger(slog.New(newCapture())))
	if !reflect.DeepEqual(quiet, loud) {
		t.Errorf("a run with a logger that logs everything differs from one that logs nothing, want the same snapshot")
	}
}

// TestCheckpoint_MigrationRoundTrip verifies FR-MIG-041, FR-CHK-041: a world with
// households on a journey, runs of the two Warns and a pending end count, saved
// and restored, has the same journeys (tag 3), runs and pending ends (tag 20), and a
// restore that continues makes the same decisions as the original.
//
// Requirement: FR-MIG-041, FR-CHK-041
func TestCheckpoint_MigrationRoundTrip(t *testing.T) {
	t.Parallel()
	cfg := passConfig(func(c *Config) {
		slowJourneys(c)
		c.Modules.Migration.MoveCooldown, c.Modules.Migration.MoveBelow = 0, MaxBasisPoints
	})
	opts := func() []Option { return []Option{WithStrategy("script", pairScript{}), WithAudit()} }
	a, err := NewWorld(cfg, opts()...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	step(t, a, 29) // a role-since of -1 000 cannot be saved: the agents settle at the welfare window, 30
	step(t, a, 4)
	a.migration.ended[infoC] = 7
	ck, err := a.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	b, err := RestoreWorld(cfg, ck, opts()...)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if len(a.migration.active) == 0 {
		t.Fatalf("no journey in the original at the boundary, want some")
	}
	ma, mb := a.migration, b.migration
	if !slices.Equal(ma.active, mb.active) || !slices.Equal(ma.journey, mb.journey) ||
		!slices.Equal(ma.blockedRun, mb.blockedRun) || !slices.Equal(ma.heldRun, mb.heldRun) || !slices.Equal(ma.ended, mb.ended) {
		t.Errorf("restored migration state differs: active %v/%v journeys %v/%v runs %v/%v %v/%v ended %v/%v",
			ma.active, mb.active, ma.journey, mb.journey, ma.blockedRun, mb.blockedRun, ma.heldRun, mb.heldRun, ma.ended, mb.ended)
	}
	if ma.blockedRun[infoC] == 0 {
		t.Errorf("blocked run of region c = 0, want a run to have been saved")
	}
	// Both continue: the same snapshot after more ticks, journeys arriving included.
	step(t, a, 12)
	step(t, b, 12)
	sa, _ := a.Snapshot()
	sb, _ := b.Snapshot()
	if !reflect.DeepEqual(sa, sb) {
		t.Errorf("the restored world's snapshot after 12 ticks differs from the original's")
	}
	// The checkpoint is canonical: saving the restored world at the same tick gives the same bytes.
	ckb, err := b.Checkpoint()
	cka, _ := a.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() of the restored world error = %v, want nil", err)
	}
	ba, _ := cka.MarshalBinary()
	bb, _ := ckb.MarshalBinary()
	if !slices.Equal(ba, bb) {
		t.Errorf("the checkpoints of the original and the restored world differ after the same ticks, want identical bytes")
	}
}

// TestFork_MigrationMidJourney verifies FR-MIG-041, the fork-at-N property for a
// household on a journey at the boundary: for several N, a world that runs on and one
// restored from a checkpoint at N deliver the same events and end in the same state.
//
// Requirement: FR-MIG-041, FR-CHK-041
func TestFork_MigrationMidJourney(t *testing.T) {
	t.Parallel()
	cfg := passConfig(func(c *Config) {
		slowJourneys(c)
		c.Modules.Migration.MoveCooldown, c.Modules.Migration.MoveBelow = 0, MaxBasisPoints
	})
	opts := func(extra ...Option) []Option {
		return append([]Option{WithStrategy("script", pairScript{}), WithAudit()}, extra...)
	}
	for _, n := range []int{0, 1, 2, 3, 12} {
		refRec := &evLog{}
		ref, err := NewWorld(cfg, opts(WithObserver(refRec))...)
		if err != nil {
			t.Fatalf("NewWorld() error = %v, want nil", err)
		}
		step(t, ref, 29+n)
		ck, err := ref.Checkpoint()
		if err != nil {
			t.Fatalf("N %d: Checkpoint() error = %v, want nil", n, err)
		}
		fork := &evLog{}
		got, err := RestoreWorld(cfg, ck, opts(WithObserver(fork))...)
		if err != nil {
			t.Fatalf("N %d: RestoreWorld() error = %v, want nil", n, err)
		}
		inFlight := len(ref.migration.active)
		mark := len(refRec.events)
		step(t, ref, 14)
		step(t, got, 14)
		after := &evLog{events: refRec.events[mark:]}
		want := of[MigrationStarted](after)
		have := of[MigrationStarted](fork)
		if !slices.Equal(want, have) {
			t.Errorf("N %d: MigrationStarted after the boundary: original %+v, restored %+v", n, want, have)
		}
		wr, hr := of[MigrationReviewed](after), of[MigrationReviewed](fork)
		if !slices.Equal(wr, hr) {
			t.Errorf("N %d: MigrationReviewed after the boundary: original %+v, restored %+v", n, wr, hr)
		}
		if n == 2 && inFlight == 0 {
			t.Errorf("N %d: no journey in flight at the boundary, want the cell to be inside a journey", n)
		}
		sa, _ := ref.Snapshot()
		sb, _ := got.Snapshot()
		if !reflect.DeepEqual(sa.Journeys, sb.Journeys) || !reflect.DeepEqual(sa.Regions, sb.Regions) {
			t.Errorf("N %d: the snapshots after the boundary differ", n)
		}
	}
}

// TestAlloc_PassNoOp verifies NFR-MIG-001: with the default logger and no observer a
// pass that starts nobody allocates nothing once the scratch has grown, with the gate
// shut and with it open.
//
// Requirement: NFR-MIG-001
func TestAlloc_PassNoOp(t *testing.T) {
	// Not parallel: AllocsPerRun reads the process-wide allocation count.
	for _, gate := range []string{"shut", "open"} {
		w := newMill(t, passConfig(func(c *Config) { c.Modules.Migration.MoveOutShare = 0 }), WithStrategy("script", pairScript{}))
		for i := len(w.cfg.Regions); i < len(w.agents); i++ {
			w.agents[i].roleSince = -1_000
		}
		step(t, w, 6)
		if gate == "open" {
			open(w)
		}
		w.p72Migration()
		if n := testing.AllocsPerRun(20, w.p72Migration); n != 0 {
			t.Errorf("gate %s: the pass allocated %v times per run, want 0", gate, n)
		}
	}
}

// TestMigration_CardinalityOfAttributes verifies the cardinality rule of the
// metrics (observability-and-logging.md 5.4.1): the reason of an outcome is one of
// the ten closed values, the regions of the events are valid RegionIDs of the
// configuration, and the eleven outcomes map to distinct strings that are the limit
// attribute's values.
//
// Requirement: FR-MIG-038, NFR-OBS-004
func TestMigration_CardinalityOfAttributes(t *testing.T) {
	t.Parallel()
	closed := []Reason{
		ReasonStarted, ReasonNoEstimate, ReasonNoDestination, ReasonNoRumour, ReasonTooFar,
		ReasonNoViable, ReasonNoGain, ReasonBound, ReasonOutCap, ReasonUnprovisioned, ReasonInCap,
	}
	if !slices.Equal(outcomeReason[:], closed) {
		t.Errorf("outcomeReason = %v, want the eleven values of 7.6 in order %v", outcomeReason, closed)
	}
	seen := map[Reason]bool{}
	for _, r := range outcomeReason {
		if seen[r] {
			t.Errorf("reason %q is mapped twice", r)
		}
		seen[r] = true
	}
	w, rec := passWorld(t, 6, slowJourneys)
	open(w)
	step(t, w, 12)
	regions := RegionID(len(w.cfg.Regions))
	for _, e := range of[MigrationStarted](rec) {
		if e.From >= regions || e.To >= regions || e.From == e.To {
			t.Errorf("MigrationStarted from %d to %d, want two different regions below %d", e.From, e.To, regions)
		}
	}
	if int(mig.InCap)+1 != len(closed) {
		t.Errorf("the package has %d outcomes, the closed set has %d", int(mig.InCap)+1, len(closed))
	}
}

// TestCheckpoint_MigrationJourneyBounds verifies that a journey that cannot be one is
// corrupt: to the region the account is in, or arriving before tick 1 or beyond the
// horizon.
//
// Requirement: FR-MIG-041
func TestCheckpoint_MigrationJourneyBounds(t *testing.T) {
	t.Parallel()
	w, _ := passWorld(t, 2, nil)
	rs := &restoreCtx{tick: 10}
	h := w.migration.cfg.MoveHorizon
	tests := []struct {
		name string
		rec  agentRec
		want bool
	}{
		{"to its own region", agentRec{jto: uint64(infoA), jarrives: 5}, true},
		{"before tick 1", agentRec{jto: uint64(infoB), jarrives: 0}, true},
		{"beyond the horizon", agentRec{jto: uint64(infoB), jarrives: 10 + h + 2}, true},
		{"the farthest allowed", agentRec{jto: uint64(infoB), jarrives: 10 + h + 1}, false},
		{"held past its arrival", agentRec{jto: uint64(infoB), jarrives: 3}, false},
	}
	for _, tt := range tests {
		if got := w.badJourney(rs, &tt.rec, int(infoA)); got != tt.want {
			t.Errorf("%s: badJourney = %v, want %v", tt.name, got, tt.want)
		}
	}
}
