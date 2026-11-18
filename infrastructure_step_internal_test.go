package economysim

import (
	"log/slog"
	"reflect"
	"slices"
	"testing"
)

// facStep runs one step and fails the test on an error.
func facStep(t testing.TB, w *World) {
	t.Helper()
	if err := w.Step(); err != nil {
		t.Fatalf("Step() at tick %d with seed %d error = %v, want nil", w.Tick(), facSeed, err)
	}
}

// facPlant puts a facility record in the list the way a completion does: it creates the
// record and does not rebuild the tables. It returns the ID.
func facPlant(w *World, class FacilityClassID, region RegionID, owner AgentID, condition int64) FacilityID {
	return w.infra.create(class, region, NoRegion, owner, condition, w.Tick()).id
}

// facDecaying is the mutation of facConfig that gives the barn a decay of 1.
func facDecaying(m *Infrastructure) {
	for i := range m.Classes {
		if m.Classes[i].ID == "barn" {
			m.Classes[i].Decay = 1
		}
	}
}

// TestStep_Decay verifies step 1 of P8.3b: each facility loses its class's Decay a tick
// whether used or not, to a floor of 0, and a facility added between ticks decays at the
// end of the first tick it lives in. A class with a Decay of 0 keeps its condition.
//
// Requirement: FR-CAP-007
func TestStep_Decay(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	m := facMiller(w)
	ids, err := w.AddFacilities([]NewFacility{
		{Class: clsMill, Region: 0, Toward: NoRegion, Owner: m},
		{Class: clsMill, Region: 0, Toward: NoRegion, Owner: m, Condition: 10},
		{Class: clsRoad, Region: 0, Toward: 1, Owner: m},
	})
	if err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	for tick := range 3 {
		facStep(t, w)
		got := []int64{facRecord(w, ids[0]).condition, facRecord(w, ids[1]).condition, facRecord(w, ids[2]).condition}
		// The mill has Decay 1 and Durability 1 000, the road Decay 0: after tick+1 steps the
		// first mill is at 1 000-(tick+1), the second at 10-(tick+1), the road untouched.
		want := []int64{1_000 - int64(tick+1), 10 - int64(tick+1), 2_000}
		if !slices.Equal(got, want) {
			t.Fatalf("conditions after %d steps = %v, want %v (seed %d)", tick+1, got, want, facSeed)
		}
	}
}

// TestStep_RuinCauses verifies step 2 of P8.3b: a facility whose fall of the step takes it
// to 0 is ruined with the cause decay, one that was already at 0 (worn out by the starts of
// the tick) with the cause use, the record is removed, the ID is not reused and the effect
// leaves the tables in the same step.
//
// Requirement: FR-CAP-008
func TestStep_RuinCauses(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, nil, WithObserver(obs))
	m := facMiller(w)
	worn := facPlant(w, clsMill, 0, m, 0)  // wore out at a start of this tick
	last := facPlant(w, clsMill, 0, m, 1)  // one unit left: the fall of 1 takes it to 0
	kept := facPlant(w, clsMill, 0, m, 50) // stays
	w.infra.rebuild()
	next := w.infra.next
	facStep(t, w)
	got := map[FacilityID]Cause{}
	for _, e := range of[FacilityChanged](obs) {
		if e.Kind == FacilityRuined {
			got[e.Facility] = e.Cause
		}
	}
	want := map[FacilityID]Cause{worn: CauseUse, last: CauseDecay}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ruin causes = %v, want %v (seed %d)", got, want, facSeed)
	}
	if w.infra.index(worn) >= 0 || w.infra.index(last) >= 0 || w.infra.index(kept) < 0 {
		t.Errorf("records after the step: worn %d last %d kept %d, want the two ruined removed and the other kept",
			w.infra.index(worn), w.infra.index(last), w.infra.index(kept))
	}
	if cap := w.infra.site[siteKey{m, 0, clsMill}]; cap != 1 {
		t.Errorf("site capacity after the ruins = %d, want 1: only the mill at 49 of 1 000 serves, ceil(2*49/1000) = 1", cap)
	}
	if w.infra.next != next {
		t.Errorf("next facility ID = %d after the ruins, want %d: an ID is never reused", w.infra.next, next)
	}
	for _, e := range of[FacilityChanged](obs) {
		if e.Kind == FacilityRuined && (e.After != 0 || e.Effect != 0) {
			t.Errorf("FacilityChanged %+v, want After 0 and Effect 0 for a ruin", e)
		}
	}
	if ruined := of[FacilityChanged](obs); len(ruined) != 2 || ruined[0].Facility != worn || ruined[1].Facility != last {
		t.Errorf("ruin events = %+v, want worn then last in ascending ID", ruined)
	}
}

// TestStep_Commissioning verifies that a facility created at the completion of a job, which
// does not rebuild the tables, is in the list when P8.3b runs, so it decays by one tick at
// the end of the tick it was made in and enters the tables at the step: its effect is read
// from the next tick (spec-infrastructure.md 6.3.3, FR-CAP-016).
//
// Requirement: FR-CAP-016
func TestStep_Commissioning(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	m := facMiller(w)
	id := facPlant(w, clsMill, 0, m, 1_000)
	if got := w.infra.site[siteKey{m, 0, clsMill}]; got != 0 {
		t.Fatalf("site capacity before the step = %d, want 0: the record is not in the tables yet", got)
	}
	facStep(t, w)
	if c := facRecord(w, id).condition; c != 999 {
		t.Errorf("condition after its first step = %d, want 999: it decays in the tick it was made in", c)
	}
	// cap = ceil(2 * 999 / 1000) = 2.
	if got := w.infra.site[siteKey{m, 0, clsMill}]; got != 2 {
		t.Errorf("site capacity after the step = %d, want 2: the effect enters the tables at the step", got)
	}
}

// facEvents lists, in order, a short tag of each facility and store event of the log.
func facEvents(obs *evLog) []string {
	var out []string
	for _, e := range obs.events {
		switch v := e.(type) {
		case FacilityChanged:
			out = append(out, "F:"+string(v.Kind))
		case StoreChanged:
			out = append(out, "S:"+string(v.Kind)+":"+string(v.Cause))
		case FacilityLimited:
			out = append(out, "L")
		case TickEnded:
			out = append(out, "T")
		}
	}
	return out
}

// TestStep_Order verifies the order of P8.3b: the ruins in ascending ID, each store
// facility's storage StoreChanged before its FacilityChanged, then the installs in ascending
// ID in the same form, then FacilityLimited, then TickEnded.
//
// Requirement: FR-CAP-028, FR-CAP-047
func TestStep_Order(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, facDecaying, WithObserver(obs))
	ms := facAgents(w, 0, "miller")
	weaver := facAgents(w, 0, "weaver")[0]
	if _, err := w.AddFacilities([]NewFacility{{Class: clsBarn, Region: 0, Toward: NoRegion, Owner: ms[0], Condition: 1}}); err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	facPlant(w, clsBarn, 0, ms[1], 800)  // installs
	facPlant(w, clsBarn, 0, weaver, 800) // no base room: pending
	facPlant(w, clsMill, 0, ms[2], 1)    // ruined by decay
	obs.events = nil
	facStep(t, w)
	var got []string
	for _, tag := range facEvents(obs) {
		if tag != "T" || len(got) > 0 {
			got = append(got, tag)
		}
	}
	// IDs: barn 1 (condition 1: ruined, its store first), barn 2 (installs), barn 3 (pending),
	// mill 4 (condition 1: ruined). The ruins come first in ascending ID, then the install,
	// then the aggregate for the pending barn, then the end of the tick.
	want := []string{"S:removed:facility", "F:ruined", "F:ruined", "S:added:facility", "F:installed", "L", "T"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("events of the step = %v, want %v (seed %d)", got, want, facSeed)
	}
}

// TestStoreFacility_Install verifies step 3 of P8.3b: a store facility with no store asks
// storage for it, the owner holds the store from then on and the installed count and the
// ErrStoreInUse seam follow, and the facility is not pending afterwards.
//
// Requirement: FR-CAP-026, FR-CAP-047
func TestStoreFacility_Install(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	m := facMiller(w)
	id := facPlant(w, clsBarn, 0, m, 800)
	before := w.sto.count(m, 0)
	facStep(t, w)
	if !facRecord(w, id).installed || w.sto.count(m, 0) != before+1 {
		t.Errorf("after the step installed = %v, stores of the class = %d, want true and %d", facRecord(w, id).installed, w.sto.count(m, 0), before+1)
	}
	if got := w.infra.installedOf(w.agentOf(m), 0); got != 1 {
		t.Errorf("installedOf = %d, want 1", got)
	}
	if rows := w.facilityRows(); len(rows) != 1 || rows[0].Pending != 0 {
		t.Errorf("rows = %+v, want one row with no pending facility", rows)
	}
}

// TestStoreFacility_Pending verifies that a refusal of the install leaves the facility
// pending, counted in FacilityLimited and in the snapshot row, gives no room, and is retried
// at every P8.3b: an owner with no base room, and an owner whose account is not in the
// facility's region. When the cause goes the facility installs at the next step.
//
// Requirement: FR-CAP-026, FR-CAP-047
func TestStoreFacility_Pending(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		place func(w *World) (region RegionID, owner AgentID)
		heal  func(w *World, owner AgentID)
	}{
		{"owner with no base room", func(w *World) (RegionID, AgentID) { return 0, facAgents(w, 0, "weaver")[0] }, nil},
		{"owner in another region", func(w *World) (RegionID, AgentID) { return 1, facMiller(w) },
			func(w *World, owner AgentID) {
				if err := w.Relocate(owner, 1); err != nil {
					t.Fatalf("Relocate error = %v, want nil", err)
				}
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			obs := &evLog{}
			w := facWorld(t, nil, WithObserver(obs))
			region, owner := tt.place(w)
			id := facPlant(w, clsBarn, region, owner, 800)
			stores := w.sto.count(owner, 0)
			for range 2 {
				facStep(t, w)
			}
			if facRecord(w, id).installed || w.sto.count(owner, 0) != stores {
				t.Fatalf("installed = %v, stores = %d, want pending and %d stores (seed %d)", facRecord(w, id).installed, w.sto.count(owner, 0), stores, facSeed)
			}
			lim := of[FacilityLimited](obs)
			if len(lim) != 2 || lim[0].Pending != 1 || lim[0].Region != region {
				t.Errorf("FacilityLimited = %+v, want one a tick with Pending 1 in region %d", lim, region)
			}
			if rows := w.facilityRows(); len(rows) != 1 || rows[0].Pending != 1 {
				t.Errorf("rows = %+v, want Pending 1", rows)
			}
			if tt.heal == nil {
				return
			}
			tt.heal(w, owner)
			facStep(t, w)
			if !facRecord(w, id).installed {
				t.Errorf("after the cause went the facility is still pending, want it installed at the next step")
			}
		})
	}
}

// TestStoreFacility_Ruin verifies that the ruin of an installed store facility removes its
// store by the forced form of storage's removal: StoreChanged (removed, cause facility)
// first, then FacilityChanged (ruined, decay), the owner holds one store less and the
// installed count falls.
//
// Requirement: FR-CAP-026, FR-CAP-047
func TestStoreFacility_Ruin(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, facDecaying, WithObserver(obs))
	m := facMiller(w)
	if _, err := w.AddFacilities([]NewFacility{{Class: clsBarn, Region: 0, Toward: NoRegion, Owner: m, Condition: 1}}); err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	stores := w.sto.count(m, 0)
	obs.events = nil
	facStep(t, w)
	got := slices.DeleteFunc(facEvents(obs), func(s string) bool { return s == "T" })
	if want := []string{"S:removed:facility", "F:ruined"}; !reflect.DeepEqual(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
	if w.sto.count(m, 0) != stores-1 || w.infra.installedOf(w.agentOf(m), 0) != 0 {
		t.Errorf("stores = %d, installed = %d, want %d and 0", w.sto.count(m, 0), w.infra.installedOf(w.agentOf(m), 0), stores-1)
	}
}

// TestEvents_FacilityLimited verifies the aggregate: one event per (region, class) with a
// non-zero count, in ascending region then class, never when every count is 0, and no
// counter is kept when no observer was supplied.
//
// Requirement: FR-CAP-038, FR-CAP-039
func TestEvents_FacilityLimited(t *testing.T) {
	t.Parallel()
	setup := func(w *World) {
		facPlant(w, clsBarn, 1, facAgents(w, 1, "weaver")[0], 800)
		facPlant(w, clsBarn, 0, facAgents(w, 0, "weaver")[0], 800)
	}
	obs := &evLog{}
	w := facWorld(t, nil, WithObserver(obs))
	facStep(t, w)
	if n := len(of[FacilityLimited](obs)); n != 0 {
		t.Fatalf("FacilityLimited with every count 0 = %d events, want none", n)
	}
	setup(w)
	facStep(t, w)
	want := []FacilityLimited{{Tick: 1, Region: 0, Class: clsBarn, Pending: 1}, {Tick: 1, Region: 1, Class: clsBarn, Pending: 1}}
	if got := of[FacilityLimited](obs); !reflect.DeepEqual(got, want) {
		t.Errorf("FacilityLimited = %+v, want %+v (seed %d)", got, want, facSeed)
	}
	if clear := w.infra.limits; slices.ContainsFunc(clear, func(l facilityLimit) bool { return l != facilityLimit{} }) {
		t.Errorf("counters after the step = %+v, want them cleared at P8.6", clear)
	}
	quiet := facWorld(t, nil)
	setup(quiet)
	facStep(t, quiet)
	if slices.ContainsFunc(quiet.infra.limits, func(l facilityLimit) bool { return l != facilityLimit{} }) {
		t.Errorf("counters with no observer = %+v, want none kept", quiet.infra.limits)
	}
}

// TestEvents_P83bAfterRent verifies the position of P8.3b in the tick: after P8.3 (and, once
// the rent step P8.3a exists, after it) and before the audit, and that the events of the
// step come after EndowmentChanged of the regeneration and before TickEnded.
//
// Requirement: FR-CAP-048
func TestEvents_P83bAfterRent(t *testing.T) {
	t.Parallel()
	at := func(id string) int {
		return slices.IndexFunc(positionList[:], func(p position) bool { return p.id == id })
	}
	if !(at("P8.3") < at("P8.3b") && at("P8.3b") < at("P8.4") && at("P8.3b") < at("P8.3c")) {
		t.Errorf("positions P8.3 %d, P8.3b %d, P8.3c %d, P8.4 %d, want them in this order", at("P8.3"), at("P8.3b"), at("P8.3c"), at("P8.4"))
	}
	obs := &evLog{}
	w := facWorld(t, facDecaying, WithObserver(obs))
	if _, err := w.AddFacilities([]NewFacility{{Class: clsBarn, Region: 0, Toward: NoRegion, Owner: facMiller(w), Condition: 1}}); err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	obs.events = nil
	facStep(t, w)
	last := func(name string) int {
		idx := -1
		for i, e := range obs.events {
			if reflect.TypeOf(e).Name() == name {
				idx = i
			}
		}
		return idx
	}
	first := slices.IndexFunc(obs.events, func(e Event) bool { _, ok := e.(FacilityChanged); return ok })
	if end := last("TickEnded"); first < last("EndowmentChanged") || first > end {
		t.Errorf("first FacilityChanged at %d, last EndowmentChanged at %d, TickEnded at %d, want it between them", first, last("EndowmentChanged"), end)
	}
}

// TestFacility_Conservation verifies that a ruin moves no money and no goods: the audit of
// every tick (the ledger identities) passes across steps in which facilities decay and are
// ruined, and no BoundaryFlow row carries a facility reason (spec-infrastructure.md 6.2.3).
//
// Requirement: FR-CAP-042, FR-CAP-044
func TestFacility_Conservation(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, facDecaying, WithObserver(obs))
	m := facMiller(w)
	for _, c := range []int64{1, 2, 3} {
		facPlant(w, clsMill, 0, m, c)
		facPlant(w, clsBarn, 0, facAgents(w, 0, "miller")[int(c)], c)
	}
	for range 5 {
		facStep(t, w) // the audit runs every tick (WithAudit) and fails the step on a broken identity
	}
	if len(w.infra.fac) != 0 {
		t.Fatalf("%d facilities left, want all ruined after 4 steps", len(w.infra.fac))
	}
	for _, e := range of[BoundaryFlow](obs) {
		if len(e.Reason) >= 8 && e.Reason[:8] == "facility" {
			t.Errorf("BoundaryFlow %+v, want no facility reason: a ruin is not a flow", e)
		}
	}
}

// TestFacility_NoInfoOrWarnInTick verifies FR-CAP-040: the steps of a run with ruins, installs
// and pending facilities log nothing above Debug; the module's Warns are at construction.
//
// Requirement: FR-CAP-040
func TestFacility_NoInfoOrWarnInTick(t *testing.T) {
	t.Parallel()
	var recs []slog.Record
	w := facWorld(t, facDecaying, WithLogger(slog.New(mkLogs{&recs})))
	facPlant(w, clsMill, 0, facMiller(w), 1)
	facPlant(w, clsBarn, 0, facAgents(w, 0, "weaver")[0], 800)
	facPlant(w, clsBarn, 0, facAgents(w, 0, "miller")[1], 800)
	recs = nil
	for range 3 {
		facStep(t, w)
	}
	var debug int
	for _, r := range recs {
		if r.Level > slog.LevelDebug && r.Message != "tick ended" {
			t.Errorf("logged %q at level %v inside a tick, want only Debug", r.Message, r.Level)
		}
		if r.Level == slog.LevelDebug && (r.Message == "facility ruined" || r.Message == "facility installed") {
			debug++
		}
	}
	if debug != 2 {
		t.Errorf("Debug lines of a ruin and an install = %d, want 2", debug)
	}
}

// TestInfrastructure_StepAllocsPerRun verifies NFR-CAP-002 for the step: with the default
// logger and no observer, P8.3b over facilities that neither decay out nor install
// allocates nothing. Not parallel: AllocsPerRun panics in a parallel test.
//
// Requirement: NFR-CAP-002
func TestInfrastructure_StepAllocsPerRun(t *testing.T) {
	w := facWorld(t, nil)
	m := facMiller(w)
	facPlant(w, clsMill, 0, m, 900)
	facPlant(w, clsRoad, 0, m, 900)
	facPlant(w, clsDrain, 0, m, 400)
	facPlant(w, clsBarn, 0, facAgents(w, 0, "weaver")[0], 800) // stays pending
	w.infra.rebuild()
	w.p83bFacilities() // the first run sizes nothing, but settle any lazy state
	if got := testing.AllocsPerRun(50, w.p83bFacilities); got != 0 {
		t.Errorf("allocations of P8.3b = %v, want 0", got)
	}
}

// TestStoreFacility_RoleChangeRuin verifies the step of a role change (spec-infrastructure.md
// 5.4.3): an agent that moves to a role with no base room loses its stores, so its installed
// store facilities are ruined in ascending ID with cause role_changed after RoleChanged and
// storage's StoreChanged events, a pending facility stays pending, and the same happens with
// no observer. A change to a role that keeps a room ruins nothing.
//
// Requirement: FR-CAP-027, FR-CAP-047
func TestStoreFacility_RoleChangeRuin(t *testing.T) {
	t.Parallel()
	for _, observed := range []bool{true, false} {
		obs := &evLog{}
		var opts []Option
		if observed {
			opts = append(opts, WithObserver(obs))
		}
		w := facWorld(t, nil, opts...)
		m := facMiller(w)
		ids, err := w.AddFacilities([]NewFacility{
			{Class: clsBarn, Region: 0, Toward: NoRegion, Owner: m},
			{Class: clsBarn, Region: 0, Toward: NoRegion, Owner: m},
			{Class: clsMill, Region: 0, Toward: NoRegion, Owner: m},
		})
		if err != nil {
			t.Fatalf("AddFacilities error = %v, want nil", err)
		}
		pending := facPlant(w, clsBarn, 0, m, 800)
		obs.events = nil
		weaver, _ := w.cfg.roleID("weaver")
		if err := w.changeRole(0, &w.agents[m], Decision{Role: weaver, Cause: CauseReview}, false); err != nil {
			t.Fatalf("changeRole error = %v, want nil", err)
		}
		for _, id := range ids[:2] {
			if w.infra.index(id) >= 0 {
				t.Errorf("observed %v: facility %d still exists, want the store facility ruined", observed, id)
			}
		}
		if w.infra.index(ids[2]) < 0 || w.infra.index(pending) < 0 || facRecord(w, pending).installed {
			t.Errorf("observed %v: the mill and the pending barn must stay, the pending one pending", observed)
		}
		if got := w.infra.installedOf(&w.agents[m], 0); got != 0 {
			t.Errorf("observed %v: installed count = %d, want 0", observed, got)
		}
		if err := w.Audit(); err != nil {
			t.Errorf("observed %v: Audit() = %v, want nil", observed, err)
		}
		if !observed {
			continue
		}
		var got []string
		for _, e := range obs.events {
			switch v := e.(type) {
			case RoleChanged:
				got = append(got, "role")
			case StoreChanged:
				got = append(got, "store")
			case FacilityChanged:
				got = append(got, string(v.Kind)+":"+string(v.Cause))
			}
		}
		// Storage ends one store a barn (two StoreChanged events, after RoleChanged), then
		// the two ruins follow in ascending ID.
		if want := []string{"role", "store", "store", "ruined:role_changed", "ruined:role_changed"}; !slices.Equal(got, want) {
			t.Errorf("events of the change = %v, want %v", got, want)
		}
	}
}
