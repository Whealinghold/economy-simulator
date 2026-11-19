package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// The tests of the membership round and of the tables of the institutions module at the
// level of the world (spec-institutions.md 3, 4). The rules themselves are tested in
// internal/institution; these build a world from orgToy, whose households are agents of
// weight 1, and read the round's counts, the events and the membership fields.

// The regions and roles of orgToy, by ID.
const (
	orgA RegionID = iota
	orgB
	orgC
)

const (
	orgFarmer RoleID = iota
	orgSmith
)

// orgToy is a viable world of three regions and two roles whose households are agents of
// weight 1: region a has 6 farmers and 4 smiths, b 3 farmers and 8 smiths, c 5 smiths.
// The body guild holds the smiths of a and b with 5 places in each region, and the body
// lodge the farmers of a with no limit; so a has 4 smiths and 6 farmers to admit, all of
// them (4 <= 5, no limit), and b has 8 smiths for 5 places: 5 admitted, 3 skipped. The
// smiths of c are in no body. The bodies are numbered in the order of their IDs, guild
// (BodyID 0) and lodge (1); orgSettlementToy adds hall between them, so lodge is then 2.
func orgToy() Config {
	c := viaToy()
	a, b, cc := DefaultRegion(), DefaultRegion(), DefaultRegion()
	a.ID, b.ID, cc.ID = "a", "b", "c"
	for _, r := range []*Region{&a, &b, &cc} {
		r.ReferenceBasket = 100
	}
	c.Regions = []Region{a, b, cc}
	smith := DefaultRole()
	smith.ID, smith.Recipes = "smith", []string{"farm"}
	c.Roles = append(c.Roles, smith)
	c.Groups = []Group{
		{Region: "a", Role: "farmer", Households: 6}, {Region: "a", Role: "smith", Households: 4},
		{Region: "b", Role: "farmer", Households: 3}, {Region: "b", Role: "smith", Households: 8},
		{Region: "c", Role: "smith", Households: 5},
	}
	c.Modules.Institutions = &Institutions{
		Bodies: []Body{
			{ID: "guild", Roles: []string{"smith"}, Regions: []string{"a", "b"}, Places: 5},
			{ID: "lodge", Roles: []string{"farmer"}, Regions: []string{"a"}},
		},
	}
	return c
}

// orgWorld builds the world of c with an event log.
func orgWorld(t *testing.T, c Config) (*World, *evLog) {
	t.Helper()
	rec := &evLog{}
	return newMill(t, c, WithObserver(rec)), rec
}

// orgAgents returns the IDs of the agents of the role in the region, ascending.
func orgAgents(w *World, g RegionID, r RoleID) []int {
	var ids []int
	for i := w.firstHousehold; i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 && a.role == r && w.regionOf(a) == g {
			ids = append(ids, i)
		}
	}
	return ids
}

// orgMembers returns the IDs of the agents that are members of body b, ascending.
func orgMembers(w *World, b int) []int {
	var ids []int
	for i := w.firstHousehold; i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 && w.org.isMember(i, a.role, w.regionOf(a), b) {
			ids = append(ids, i)
		}
	}
	return ids
}

// orgCell returns the event of the (region, body) pair, and false when none was emitted.
func orgCell(rec *evLog, tick int64, g RegionID, b BodyID) (MembershipChanged, bool) {
	for _, e := range of[MembershipChanged](rec) {
		if e.Tick == tick && e.Region == g && e.Body == b {
			return e, true
		}
	}
	return MembershipChanged{}, false
}

// TestConstruction_TablesResolved verifies the tables derived at construction: the body
// of each (role, region), the list and scope of each body and the scope regions, ascending.
//
// Arithmetic: guild is body 0 over smith in a and b, lodge body 1 over farmer in a; the
// pair (smith, c) and the farmers of b and c are in no body (-1).
//
// Requirement: FR-ORG-003, FR-ORG-004, FR-ORG-005
func TestConstruction_TablesResolved(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgToy())
	o := w.org
	cases := []struct {
		name   string
		region RegionID
		role   RoleID
		want   int32
	}{
		{"smith of a", orgA, orgSmith, 0}, {"smith of b", orgB, orgSmith, 0}, {"smith of c", orgC, orgSmith, -1},
		{"farmer of a", orgA, orgFarmer, 1}, {"farmer of b", orgB, orgFarmer, -1}, {"farmer of c", orgC, orgFarmer, -1},
	}
	for _, tc := range cases {
		if got := o.bodyOf[int(tc.region)*o.nroles+int(tc.role)]; got != tc.want {
			t.Errorf("bodyOf(%s) = %d, want %d", tc.name, got, tc.want)
		}
	}
	if want := []RegionID{orgA, orgB}; !slices.Equal(o.scope[0], want) {
		t.Errorf("scope of guild = %v, want %v", o.scope[0], want)
	}
	if want := []RegionID{orgA}; !slices.Equal(o.scope[1], want) {
		t.Errorf("scope of lodge = %v, want %v", o.scope[1], want)
	}
	if got := o.eligibleBody(orgSmith, orgB); got != 0 {
		t.Errorf("eligibleBody(smith, b) = %d, want 0", got)
	}
	if got := o.eligibleBody(orgFarmer, orgC); got != NoBody {
		t.Errorf("eligibleBody(farmer, c) = %d, want NoBody", got)
	}
}

// TestRound_NoneAtConstruction verifies that no household is a member at construction and
// no MembershipChanged is emitted before the first step: the first round is P1.9a of the
// first step (spec-institutions.md 4.1.3).
//
// Requirement: FR-ORG-013
func TestRound_NoneAtConstruction(t *testing.T) {
	t.Parallel()
	w, rec := orgWorld(t, orgToy())
	for b := range w.org.nb {
		if got := orgMembers(w, b); len(got) != 0 {
			t.Errorf("members of body %d at construction = %v, want none", b, got)
		}
	}
	if got := of[MembershipChanged](rec); len(got) != 0 {
		t.Errorf("MembershipChanged events at construction = %v, want none", got)
	}
}

// TestRound_FirstRoundAdmitsAndReports verifies the first round and its events.
//
// Arithmetic: in a the 4 smiths fit the 5 places of guild (4 admitted, none waiting) and
// the 6 farmers have no limit in lodge (6 admitted); in b the 8 smiths meet 5 places, 5
// admitted and 3 skipped and waiting. From the second tick the pair (a, guild) has
// nothing to report and the pair (b, guild) reports the 3 waiting again.
//
// Requirement: FR-ORG-006, FR-ORG-011, FR-ORG-012, FR-ORG-039, FR-ORG-043
func TestRound_FirstRoundAdmitsAndReports(t *testing.T) {
	t.Parallel()
	w, rec := orgWorld(t, orgToy())
	step(t, w, 2)
	want := []MembershipChanged{
		{Tick: 0, Region: orgA, Body: 0, Members: 4, Places: 5, Eligible: 4, Admitted: 4},
		{Tick: 0, Region: orgA, Body: 1, Members: 6, Places: 0, Eligible: 6, Admitted: 6},
		{Tick: 0, Region: orgB, Body: 0, Members: 5, Places: 5, Eligible: 8, Admitted: 5, Skipped: 3, Waiting: 3},
		{Tick: 1, Region: orgB, Body: 0, Members: 5, Places: 5, Eligible: 8, Skipped: 3, Waiting: 3},
	}
	if got := of[MembershipChanged](rec); !slices.Equal(got, want) {
		t.Errorf("MembershipChanged events = %+v, want %+v", got, want)
	}
}

// TestRound_OrderRoleSinceKeyID verifies the order of admission: the household that has
// held its role the longest first, then the lower key of the tick, then the lower ID.
//
// Arithmetic: b has 8 smiths for 5 places. With every RoleSince 0 the order is the key
// ordering.Key(0, id): the five lowest keys are admitted. With the first three smiths made
// juniors (RoleSince 5) the five older ones fill the places whatever their keys.
//
// Requirement: FR-ORG-011
func TestRound_OrderRoleSinceKeyID(t *testing.T) {
	t.Parallel()
	ids := func(w *World) []int { return orgAgents(w, orgB, orgSmith) }
	t.Run("by key when all are as senior", func(t *testing.T) {
		t.Parallel()
		w, _ := orgWorld(t, orgToy())
		all := ids(w)
		byKey := slices.Clone(all)
		slices.SortFunc(byKey, func(x, y int) int {
			kx, ky := ordering.Key(0, int64(x)), ordering.Key(0, int64(y))
			switch {
			case kx < ky:
				return -1
			case kx > ky:
				return 1
			}
			return x - y
		})
		want := slices.Clone(byKey[:5])
		slices.Sort(want)
		step(t, w, 1)
		var got []int
		for _, i := range orgMembers(w, 0) {
			if w.regionOf(&w.agents[i]) == orgB {
				got = append(got, i)
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("members of guild in b = %v, want the five of the lowest keys %v", got, want)
		}
	})
	t.Run("by seniority first", func(t *testing.T) {
		t.Parallel()
		w, _ := orgWorld(t, orgToy())
		all := ids(w)
		for _, i := range all[:3] {
			w.agents[i].roleSince = 5
		}
		step(t, w, 1)
		var got []int
		for _, i := range orgMembers(w, 0) {
			if w.regionOf(&w.agents[i]) == orgB {
				got = append(got, i)
			}
		}
		if !slices.Equal(got, all[3:]) {
			t.Errorf("members of guild in b = %v, want the five senior smiths %v", got, all[3:])
		}
	})
}

// TestRound_SecondRoundChangesNothing verifies that a second round with no change in
// between admits and lapses no one and leaves every membership as it was (FR-ORG-019).
//
// Requirement: FR-ORG-019
func TestRound_SecondRoundChangesNothing(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgToy())
	step(t, w, 1)
	before := slices.Clone(w.org.member)
	w.org.round(w)
	if !slices.Equal(before, w.org.member) {
		t.Errorf("membership after a second round = %v, want %v", w.org.member, before)
	}
	for cell, out := range w.org.rd.out {
		if out.Admitted != 0 || out.LapsedList != 0 || out.LapsedScope != 0 {
			t.Errorf("second round, cell %d: counts = %+v, want nobody admitted or lapsed", cell, out)
		}
	}
}

// TestRound_NobodyRemovedForPlaces verifies that a household above the places is not
// removed: an arrival into a full body waits, and the members it finds stay members.
//
// Arithmetic: b has 5 members of 8 smiths and 5 places; 2 more smiths arrive and wait, so
// b has 10 smiths, 5 members and 5 waiting; a member's weight raised to 3 puts the used
// places at 7 above 5, and the next round removes nobody.
//
// Requirement: FR-ORG-006
func TestRound_NobodyRemovedForPlaces(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgToy())
	step(t, w, 1)
	members := orgMembers(w, 0)
	for range 2 {
		if _, err := w.AddAgent(NewAgent{Region: orgB, Role: orgSmith, Money: 100}); err != nil {
			t.Fatalf("AddAgent() error = %v, want nil", err)
		}
	}
	for _, i := range members {
		if w.regionOf(&w.agents[i]) == orgB {
			w.agents[i].weight = 3
			break
		}
	}
	w.org.round(w)
	for _, i := range members {
		if !w.org.isMember(i, w.agents[i].role, w.regionOf(&w.agents[i]), 0) {
			t.Errorf("agent %d, a member before the round, is not a member after it, want nobody removed for the places", i)
		}
	}
	cell := int(orgB)*w.org.nb + 0
	if got := w.org.rd.out[cell]; got.Waiting != 5 || got.Admitted != 0 {
		t.Errorf("round counts of (b, guild) = %+v, want Waiting 5 (3 + 2 arrivals) and Admitted 0", got)
	}
}

// orgStale builds the world of orgToy after one round and makes two memberships stale: a
// smith of a that became a farmer, and a smith of a that moved to c. It returns the world
// and the two slots.
func orgStale(t *testing.T) (w *World, changed, moved int) {
	t.Helper()
	w, _ = orgWorld(t, orgToy())
	step(t, w, 1)
	changed, moved = orgAgents(w, orgA, orgSmith)[0], orgAgents(w, orgA, orgSmith)[1]
	w.agents[changed].role = orgFarmer
	if err := w.Relocate(AgentID(moved), orgC); err != nil {
		t.Fatalf("Relocate() error = %v, want nil", err)
	}
	return w, changed, moved
}

// TestMember_StaleFieldIsNotMembership verifies that a membership field that a role change
// or a move has made stale is not a membership at a read, though the field is still set.
//
// Requirement: FR-ORG-010
func TestMember_StaleFieldIsNotMembership(t *testing.T) {
	t.Parallel()
	w, changed, moved := orgStale(t)
	o := w.org
	if o.member[changed] == 0 || o.member[moved] == 0 {
		t.Fatalf("the fields of the two agents = %d and %d before the round, want them stale and still set", o.member[changed], o.member[moved])
	}
	if o.isMember(changed, w.agents[changed].role, orgA, 0) {
		t.Errorf("isMember(role changed) = true, want false: a stale field is not a membership")
	}
	if o.isMember(moved, w.agents[moved].role, orgC, 0) {
		t.Errorf("isMember(moved) = true, want false: a stale field is not a membership")
	}
}

// TestRound_LapsesAStaleMembership verifies that the next round clears a stale field and
// counts the lapse: LapsedRole for the smith that became a farmer, LapsedRegion for the one
// that moved, in the cell of the place they were admitted in; and that the former, eligible
// for lodge in a, is a candidate there in the same round and admitted.
//
// Arithmetic: guild is BodyID 0 and lodge 1, stored as 1 and 2; a is region 0, stored as 1.
// The one that moved to c, where no body has smiths, has no membership.
//
// Requirement: FR-ORG-011
func TestRound_LapsesAStaleMembership(t *testing.T) {
	t.Parallel()
	w, changed, moved := orgStale(t)
	o := w.org
	o.round(w)
	if o.member[changed] != 2 || o.memberRegion[changed] != int32(orgA)+1 || o.member[moved] != 0 {
		t.Errorf("the fields after the round = %d (region %d) and %d, want lodge in a (2, 1) and none", o.member[changed], o.memberRegion[changed], o.member[moved])
	}
	if got := o.rd.out[int(orgA)*o.nb+0]; got.LapsedList != 1 || got.LapsedScope != 1 {
		t.Errorf("(a, guild) counts = %+v, want one LapsedList and one LapsedScope", got)
	}
}

// TestEligible_ExternalAgent verifies that an external agent is eligible like any
// household, and is admitted at the next round, and that the estates (the pseudo-agents,
// below firstHousehold) are never members.
//
// Requirement: FR-ORG-005
func TestEligible_ExternalAgent(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgToy())
	step(t, w, 1)
	id, err := w.AddAgent(NewAgent{Region: orgA, Role: orgSmith, Money: 100})
	if err != nil {
		t.Fatalf("AddAgent() error = %v, want nil", err)
	}
	w.org.round(w)
	if !w.org.isMember(int(id), orgSmith, orgA, 0) {
		t.Errorf("the external agent %d is not a member after the round, want admitted (5 places, 4 + 1 smiths)", id)
	}
	for i := range w.firstHousehold {
		if i < len(w.org.member) && w.org.member[i] != 0 {
			t.Errorf("pseudo-agent %d has the membership field %d, want none", i, w.org.member[i])
		}
	}
}

// orgHall is the BodyID of hall in orgSettlementToy: guild 0, hall 1, lodge 2.
const orgHall = 1

// orgSettlementToy is orgToy with the settlements module and a third body: hall holds the
// smiths of c and needs the flag fair, which only the class town carries. Region c has 5
// households, a hamlet (enter 0 below the town's 10), a and b are towns.
func orgSettlementToy() Config {
	c := orgToy()
	m := DefaultSettlements()
	m.Classes = []SettlementClass{{ID: "hamlet", Enter: 0}, {ID: "town", Enter: 10, Flags: []string{"fair"}}}
	c.Modules.Settlements = &m
	c.Modules.Institutions.Bodies = append(c.Modules.Institutions.Bodies,
		Body{ID: "hall", Roles: []string{"smith"}, Regions: []string{"c"}, Needs: "fair"})
	return c
}

// TestRound_NeedsFlagMissingBarsAndCounts verifies that a body whose need the class of the
// region lacks admits no one and counts its eligible non-members Barred and Waiting.
//
// Arithmetic: c is a hamlet (5 households under the town's 10), so hall admits none of its
// 5 smiths: Barred 5, Waiting 5, Members 0.
//
// Requirement: FR-ORG-008
func TestRound_NeedsFlagMissingBarsAndCounts(t *testing.T) {
	t.Parallel()
	w, rec := orgWorld(t, orgSettlementToy())
	step(t, w, 1)
	want := MembershipChanged{Tick: 0, Region: orgC, Body: orgHall, Members: 0, Places: 0, Eligible: 5, Barred: 5, Waiting: 5}
	if got, ok := orgCell(rec, 0, orgC, orgHall); !ok || got != want {
		t.Errorf("MembershipChanged(c, hall) = %+v, %v, want %+v", got, ok, want)
	}
}

// TestRound_NeedsKeepsMembers verifies that a flag gained admits the households, and that a
// flag lost afterwards removes none: the members stay and only a new arrival is barred.
//
// Requirement: FR-ORG-008
func TestRound_NeedsKeepsMembers(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgSettlementToy())
	step(t, w, 1)
	w.set.class[orgC] = 1 // town, which carries fair
	w.org.round(w)
	if got := len(orgMembers(w, orgHall)); got != 5 {
		t.Fatalf("members of hall after the flag was gained = %d, want 5", got)
	}
	w.set.class[orgC] = 0 // hamlet again
	if _, err := w.AddAgent(NewAgent{Region: orgC, Role: orgSmith, Money: 100}); err != nil {
		t.Fatalf("AddAgent() error = %v, want nil", err)
	}
	w.org.round(w)
	if got := len(orgMembers(w, orgHall)); got != 5 {
		t.Errorf("members of hall after the flag was lost = %d, want the 5 still members", got)
	}
	if got := w.org.rd.out[int(orgC)*w.org.nb+orgHall]; got.Barred != 1 || got.Admitted != 0 {
		t.Errorf("round counts of (c, hall) = %+v, want the arrival Barred 1 and Admitted 0", got)
	}
}

// TestInstitutions_WorkIsCounted verifies the work of a round: the household slots it
// visits are the households of the world, once, whatever the bodies, so the cost is
// linear in the agents (NFR-ORG-002).
//
// Requirement: NFR-ORG-002
func TestInstitutions_WorkIsCounted(t *testing.T) {
	t.Parallel()
	w, _ := orgWorld(t, orgToy())
	step(t, w, 1)
	if got, want := w.org.rd.visited, int64(26); got != want {
		t.Errorf("households visited by the round = %d, want %d (6 + 4 + 3 + 8 + 5)", got, want)
	}
}

// TestAlloc_InstitutionsNoOp verifies that a round and the report of P7.2b allocate nothing
// once the scratch has grown, with no observer and the discarded logger (observability
// rule 8.2, NFR-ORG-001).
//
// Requirement: NFR-ORG-001
func TestAlloc_InstitutionsNoOp(t *testing.T) {
	w := newMill(t, orgToy())
	step(t, w, 2)
	if n := testing.AllocsPerRun(20, func() { w.org.round(w) }); n != 0 {
		t.Errorf("AllocsPerRun(round) = %v, want 0", n)
	}
	if n := testing.AllocsPerRun(20, w.p72bInstitutions); n != 0 {
		t.Errorf("AllocsPerRun(p72bInstitutions) = %v, want 0", n)
	}
}
