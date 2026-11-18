package economysim

import (
	"reflect"
	"slices"
	"testing"

	fcy "github.com/Whealinghold/economy-sim/internal/facility"
)

// facRows is the initial rows of the tests of the records: three mills held by the
// millers of home (3 over 5 agents), eight drains held by them too (8 over 5), a drain of
// the estate of home, a road from home to yonder and one that leaves yonder, and two barns
// held by the millers of yonder.
func facRows(m *Infrastructure) {
	m.Initial = []InitialFacility{
		{Region: "home", Class: "mill", Role: "miller", Count: 3},
		{Region: "home", Class: "drain", Role: "miller", Count: 8},
		{Region: "home", Class: "drain", Count: 1},
		{Region: "home", Class: "road", Toward: "yonder", Count: 1},
		{Region: "yonder", Class: "road", Count: 1},
		{Region: "yonder", Class: "barn", Role: "miller", Count: 2},
	}
}

// TestFacility_Records verifies that the initial rows make records with IDs from 1 in
// the order of the rows, ascending and never reused, that the next ID follows the last,
// that every record is at its full durability, and that a facility is not a good: the
// ledger of the world has the accounts and lots of the same world with the module off.
//
// Requirement: FR-CAP-005
func TestFacility_Records(t *testing.T) {
	t.Parallel()
	w := facWorld(t, facRows)
	n := w.infra
	if got, want := len(n.fac), 3+8+1+1+1+2; got != want {
		t.Fatalf("seed %d: %d records, want %d", facSeed, got, want)
	}
	for i, f := range n.fac {
		if f.id != FacilityID(i+1) {
			t.Errorf("record %d has ID %d, want %d", i, f.id, i+1)
		}
		if f.condition != n.cls[f.class].durability || f.built != 0 {
			t.Errorf("record %d condition %d, built %d, want the full durability %d at tick 0", f.id, f.condition, f.built, n.cls[f.class].durability)
		}
	}
	if n.next != FacilityID(len(n.fac)+1) {
		t.Errorf("next ID = %d, want %d", n.next, len(n.fac)+1)
	}
	off := facConfig(facRows)
	off.Modules.Infrastructure = nil
	wo := newMill(t, off, WithSeed(facSeed), WithAudit())
	if w.ledger.Len() != wo.ledger.Len() {
		t.Errorf("accounts with the module = %d, off = %d, want the same: a facility is not an account", w.ledger.Len(), wo.ledger.Len())
	}
	if err := w.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
}

// facHeld is the number of facilities of the class that the owner holds in the region.
func facHeld(w *World, class string, owner AgentID, region RegionID) int {
	ci := slices.IndexFunc(w.cfg.Modules.Infrastructure.Classes, func(c FacilityClass) bool { return c.ID == class })
	var k int
	for _, f := range w.infra.fac {
		if f.owner == owner && int(f.class) == ci && f.region == region {
			k++
		}
	}
	return k
}

// TestInitial_RoundRobin verifies the holders of a row (spec-infrastructure.md 3.4.2):
// the facilities of a row go round the agents of the group in ascending agent ID, so a row
// of 3 over 5 agents gives the first three one each and a row of 8 over 5 gives the first
// three two each, as the pure round robin says; a row with no role is the estate's; and
// the store rows are held by the agents of their role, who then have the store.
//
// Arithmetic: 3 over 5 is 3 mod 5 = 3 agents with one more, base 0, so [1 1 1 0 0]; 8 over
// 5 is base 1 and 8 mod 5 = 3 agents with one more, so [2 2 2 1 1]; each miller of home
// therefore holds 1+2, 1+2, 1+2, 0+1 and 0+1 of the two classes.
//
// Requirement: FR-CAP-009
func TestInitial_RoundRobin(t *testing.T) {
	t.Parallel()
	w := facWorld(t, facRows)
	millers := facAgents(w, 0, "miller")
	if len(millers) != 5 {
		t.Fatalf("seed %d: %d millers in home, want 5", facSeed, len(millers))
	}
	count := func(class string, owner AgentID) int { return facHeld(w, class, owner, 0) }
	for _, c := range []struct {
		class string
		n     int64
	}{{"mill", 3}, {"drain", 8}} {
		want := fcy.RoundRobin(c.n, len(millers))
		for i, a := range millers {
			if got := count(c.class, a); int64(got) != want[i] {
				t.Errorf("seed %d: miller %d (%d of 5) holds %d %s, want %d", facSeed, a, i, got, c.class, want[i])
			}
		}
	}
	if got := count("drain", 0); got != 1 {
		t.Errorf("the estate of home holds %d drains, want 1", got)
	}
	checkBarns(t, w)
}

// checkBarns checks that each miller of yonder holds one installed barn and its store.
func checkBarns(t *testing.T, w *World) {
	t.Helper()
	for _, a := range facAgents(w, 1, "miller") {
		if got := w.sto.owned(a); got != 1 {
			t.Errorf("miller %d of yonder has %d stores, want 1 (a barn each)", a, got)
		}
		st, err := w.Agent(a)
		if err != nil || len(st.Facilities) != 1 || !st.Facilities[0].Installed {
			t.Errorf("Agent(%d).Facilities = %+v, %v, want one installed barn", a, st.Facilities, err)
		}
	}
}

// TestInitial_Events verifies that the initial facilities are announced after the last
// AgentSpawned and before the first step, in ascending ID, each with the kind added, the
// cause initial, a condition from 0 to the full durability and the effect of the class
// (cap of a site, eff of a route or ground, 0 for a store).
//
// Requirement: FR-CAP-009, FR-CAP-038
func TestInitial_Events(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := facWorld(t, facRows, WithObserver(obs))
	lastSpawn, first := -1, -1
	var got []FacilityChanged
	for i, e := range obs.events {
		switch v := e.(type) {
		case AgentSpawned:
			lastSpawn = i
		case FacilityChanged:
			if first < 0 {
				first = i
			}
			got = append(got, v)
		}
	}
	if len(got) != len(w.infra.fac) || first <= lastSpawn {
		t.Fatalf("seed %d: %d events from index %d, last AgentSpawned at %d; want %d events after it", facSeed, len(got), first, lastSpawn, len(w.infra.fac))
	}
	for i, e := range got {
		f := w.infra.fac[i]
		want := FacilityChanged{Tick: 0, Facility: f.id, Class: f.class, Region: f.region, To: f.to, Owner: f.owner,
			Kind: FacilityAdded, Cause: CauseInitial, Before: 0, After: f.condition, Effect: w.infra.eff(&f)}
		if !reflect.DeepEqual(e, want) {
			t.Errorf("event %d = %+v, want %+v", i, e, want)
		}
	}
	// A mill at full condition serves its capacity, a road gives its whole gain. The rows
	// are in canonical order (region, class, toward, role), so the IDs do not follow the
	// order of facRows: find the events by class.
	effect := map[FacilityClassID]int64{}
	for _, e := range got {
		effect[e.Class] = e.Effect
	}
	if effect[2] != 2 || effect[3] != 6_000 || effect[1] != 3_000 || effect[0] != 0 {
		t.Errorf("effects by class (barn, drain, mill, road) = %d, %d, %d, %d, want 0, 3000, 2 and 6000", effect[0], effect[1], effect[2], effect[3])
	}
}

// TestFacility_NoEventsWithoutObserver verifies that a world with no observer builds no
// FacilityChanged: the announcement delivers nothing and allocates nothing.
//
// Requirement: FR-CAP-039
func TestFacility_NoEventsWithoutObserver(t *testing.T) {
	w := facWorld(t, facRows)
	if allocs := testing.AllocsPerRun(20, func() { w.infra.announce(w) }); allocs != 0 {
		t.Errorf("announce with no observer allocates %v times, want 0", allocs)
	}
}

// TestTables_RebuildEqual verifies that the effect tables are a function of the records
// and the configuration alone (spec-infrastructure.md 6.4): the tables of the world equal
// those of a fresh state given the same records, after a change of a condition and a
// rebuild; and the values are the capped sums of the rule.
//
// Arithmetic: the first miller holds one of the three mills (3 over 5), which at full
// condition serves ceil(2 * 1000 / 1000) = 2 and at condition 1 serves ceil(2 * 1 / 1000)
// = 1 (a site that is not ruined serves at least one); a road of 6 000 at its full
// condition gives 6 000, so the road with a target gives 6 000 to the edge home to yonder,
// and the road back and the road that leaves yonder give 6 000 + 6 000 = 12 000 to the
// edge yonder to home; nine drains of 3 000 give 27 000, far below MaxTotalBoostBP.
//
// Requirement: FR-CAP-029, FR-CAP-006
func TestTables_RebuildEqual(t *testing.T) {
	t.Parallel()
	w := facWorld(t, facRows)
	n := w.infra
	home, yonder := RegionID(0), RegionID(1)
	if got := n.route[int(home)*n.regions*n.modes+int(yonder)*n.modes]; got != 6_000 {
		t.Errorf("route gain home to yonder = %d, want 6000 (a road with a target)", got)
	}
	if got := n.route[int(yonder)*n.regions*n.modes+int(home)*n.modes]; got != 6_000+6_000 {
		t.Errorf("route gain yonder to home = %d, want 12000 (the road back and the road that leaves yonder)", got)
	}
	if got := n.ground[0]; got != 9*3_000 {
		t.Errorf("ground gain of home = %d, want 27000 (nine drains)", got)
	}
	m := facAgents(w, 0, "miller")[0]
	if got := n.site[siteKey{m, home, 2}]; got != 2 {
		t.Errorf("site capacity of the first miller = %d, want 2 (one mill at full condition)", got)
	}
	for i := range n.fac {
		if n.cls[n.fac[i].class].kind == kindSite {
			n.fac[i].condition = 1
		}
	}
	n.rebuild()
	checkRebuilt(t, w, siteKey{m, home, 2})
}

// checkRebuilt checks the capacity of a site at condition 1 after a rebuild, and that the
// tables equal those of a fresh state given the same records.
func checkRebuilt(t *testing.T, w *World, k siteKey) {
	t.Helper()
	n := w.infra
	if got := n.site[k]; got != 1 {
		t.Errorf("site capacity at condition 1 = %d, want 1 (a site that is not ruined serves one)", got)
	}
	fresh := newInfrastructure(&w.cfg, w.set)
	fresh.world, fresh.fac = w, slices.Clone(n.fac)
	fresh.rebuild()
	if !reflect.DeepEqual(fresh.site, n.site) || !slices.Equal(fresh.route, n.route) || !slices.Equal(fresh.ground, n.ground) {
		t.Errorf("seed %d: the tables of the world differ from a rebuild of its records", facSeed)
	}
}

// TestInfrastructure_AllocsPerRun verifies that the rebuild of the effect tables
// allocates nothing beyond the tables sized at construction (NFR-CAP-002).
//
// Requirement: NFR-CAP-002
func TestInfrastructure_AllocsPerRun(t *testing.T) {
	w := facWorld(t, facRows)
	if allocs := testing.AllocsPerRun(50, w.infra.rebuild); allocs != 0 {
		t.Errorf("rebuild allocates %v times, want 0", allocs)
	}
}

// TestFacility_Snapshot_Rows verifies the rows of the snapshot: one for each (region,
// class) with a facility, in ascending order, with the count, the condition in basis
// points of the durability, the effect and the pending stores.
//
// Requirement: FR-CAP-037
func TestFacility_Snapshot_Rows(t *testing.T) {
	t.Parallel()
	w := facWorld(t, func(m *Infrastructure) {
		m.Initial = []InitialFacility{
			{Region: "yonder", Class: "road", Count: 2},
			{Region: "home", Class: "mill", Role: "miller", Count: 1},
		}
	})
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	want := []FacilityRow{
		{Region: 0, Class: 2, Facilities: 1, ConditionBP: 10_000, Effect: 2},
		{Region: 1, Class: 3, Facilities: 2, ConditionBP: 20_000, Effect: 12_000},
	}
	if !reflect.DeepEqual(s.Facilities, want) {
		t.Errorf("Snapshot().Facilities = %+v, want %+v", s.Facilities, want)
	}
}

// TestTables_GainIsCappedSum verifies that the gains that apply to one endowment add and
// are capped, never multiplied (spec-infrastructure.md 5.3.2, CAP-D3): three drains of
// 100 000 on the same ground give min(MaxTotalBoostBP, 300 000) = 200 000.
//
// Requirement: FR-CAP-024, FR-CAP-006
func TestTables_GainIsCappedSum(t *testing.T) {
	t.Parallel()
	w := facWorld(t, func(m *Infrastructure) {
		m.Classes[1].BoostBP = MaxBoostBP
		m.Initial = []InitialFacility{{Region: "home", Class: "drain", Count: 3}}
	})
	if got := w.infra.ground[0]; got != MaxTotalBoostBP {
		t.Errorf("ground gain of three drains of %d = %d, want the cap %d", MaxBoostBP, got, MaxTotalBoostBP)
	}
}
