package economysim

import (
	"slices"
	"testing"
)

// This file tests the reader of the role ceiling in the default spawner
// (spec-spawner.md 6.3.1, spec-settlements.md 6.4.2, FR-SPN-056). The numbers of the
// signals are those of spawner_decide_internal_test.go: role 1 at wage 1 200 is the
// agents' own and role 2 at wage 1 300 the destination (E 8 904 against 8 219: viable
// and better), with EntryShare 100 % so no cap binds before the ceiling.

// ceilView is a region of ten weight-1 agents of role 1 at privations 5 to 14 (id i
// has 4 + i, so rung 1 and ordered by need), role 1 with 10 households and role 2 with
// 9, whose ceiling leaves the headroom.
func ceilView(headroom int64, limited bool, agents ...RegionAgent) *RegionView {
	v := spwView()
	dest := spwRole(2, w1300, 9)
	dest.Headroom, dest.NoLimit = headroom, !limited
	v.Roles = []RoleView{spwRole(1, wOwn, 10), dest}
	v.Agents = agents
	return v
}

func ceilAgents(n int, weight int64) []RegionAgent {
	var out []RegionAgent
	for i := AgentID(1); i <= AgentID(n); i++ {
		out = append(out, spwPriv(spwAgent(i, 1, weight), 4+int64(i)))
	}
	return out
}

// TestCeiling_HeadroomAndRefusalCounted verifies the case of spec-settlements.md 12.1:
// a role with 9 households and a limit of 12 has a headroom of 3; four entrants ask,
// three are admitted in the order of need (the three highest privations) and one is
// refused and counted in RoleCeiling, not in RoleCap.
//
// Arithmetic: ceil(9 * 10 000 / 10 000) = 9 households is the role's own cap, which
// four do not reach; the ceiling room is 12 - 9 = 3; of the agents 4, 3, 2 and 1 at
// privations 8, 7, 6 and 5 the first three fit, the fourth does not (1 > 0 left).
//
// Requirement: FR-SET-023, FR-SPN-056
func TestCeiling_HeadroomAndRefusalCounted(t *testing.T) {
	t.Parallel()
	d := spwDecide(ceilView(3, true, ceilAgents(4, 1)...))
	want := []Decision{spwSwitch(4, 2, CauseDistress), spwSwitch(3, 2, CauseDistress), spwSwitch(2, 2, CauseDistress)}
	if !slices.Equal(d.List, want) || d.roleCeiling != 1 || d.roleCap != 0 {
		t.Errorf("Decide plan %+v roleCeiling %d roleCap %d, want %+v, 1, 0", d.List, d.roleCeiling, d.roleCap, want)
	}
}

// TestCeiling_NoLimitAdmitsAll verifies that a role with no ceiling is bounded by the
// caps of Stage 1 alone and counts no refusal.
//
// Requirement: FR-SET-023
func TestCeiling_NoLimitAdmitsAll(t *testing.T) {
	t.Parallel()
	d := spwDecide(ceilView(0, false, ceilAgents(4, 1)...))
	if len(d.List) != 4 || d.roleCeiling != 0 {
		t.Errorf("Decide plan %+v roleCeiling %d, want 4 switches and 0", d.List, d.roleCeiling)
	}
}

// TestCeiling_WholeCandidateOnly verifies that an agent of weight 2 is not admitted
// into a headroom of 1, that its two households are counted, and that the weight-1
// agent behind it takes the place (spec-settlements.md 6.4.2: only while the weight of
// the agent fits).
//
// Arithmetic: privation 14 (weight 2) goes first; 2 > 1 left, refused, RoleCeiling 2;
// the agent at privation 5 (weight 1) fits, 1 <= 1.
//
// Requirement: FR-SET-023, FR-SPN-056
func TestCeiling_WholeCandidateOnly(t *testing.T) {
	t.Parallel()
	agents := []RegionAgent{spwPriv(spwAgent(1, 1, 2), 14), spwPriv(spwAgent(2, 1, 1), 5)}
	d := spwDecide(ceilView(1, true, agents...))
	if want := []Decision{spwSwitch(2, 2, CauseDistress)}; !slices.Equal(d.List, want) || d.roleCeiling != 2 {
		t.Errorf("Decide plan %+v roleCeiling %d, want %+v and 2", d.List, d.roleCeiling, want)
	}
}

// TestCeiling_FirstComeIsPlanOrder verifies SET-A5: when a ceiling binds the entrants
// are served in the order of the plan, need first and then agent ID, so the agents at
// the highest privation get the places, and equal privation goes to the lower ID.
//
// Requirement: FR-SET-023
func TestCeiling_FirstComeIsPlanOrder(t *testing.T) {
	t.Parallel()
	agents := []RegionAgent{spwPriv(spwAgent(5, 1, 1), 9), spwPriv(spwAgent(3, 1, 1), 9), spwPriv(spwAgent(4, 1, 1), 12)}
	d := spwDecide(ceilView(2, true, agents...))
	want := []Decision{spwSwitch(4, 2, CauseDistress), spwSwitch(3, 2, CauseDistress)}
	if !slices.Equal(d.List, want) || d.roleCeiling != 1 {
		t.Errorf("Decide plan %+v roleCeiling %d, want %+v and 1 (privation 12, then ID 3 before 5)", d.List, d.roleCeiling, want)
	}
}

// TestCeiling_NobodyRemoved verifies that a role above its ceiling (headroom 0, as a
// fall of the limit leaves it) loses no household: with no one starving the plan is
// empty, and the households of the role are untouched (spec-settlements.md 6.4.2).
//
// Requirement: FR-SET-023
func TestCeiling_NobodyRemoved(t *testing.T) {
	t.Parallel()
	v := ceilView(0, true, spwAgent(1, 2, 3), spwAgent(2, 2, 1))
	if d := spwDecide(v); len(d.List) != 0 || d.roleCeiling != 0 {
		t.Errorf("Decide plan %+v roleCeiling %d, want none and 0", d.List, d.roleCeiling)
	}
}

// TestCeiling_ReplacementLikeForLike verifies that the like-for-like replacement of a
// leaver is an entrant too: the removed household of role 2 is replaced only if the
// ceiling has room, else the replacement is refused and counted.
//
// Arithmetic: one household of role 2 starves and is removed; the carry gives one
// entry. Headroom 1: the entry is admitted. Headroom 0: refused, RoleCeiling 1, no
// Enter in the plan.
//
// Requirement: FR-SET-023, FR-SPN-056
func TestCeiling_ReplacementLikeForLike(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		headroom, wantCeiling int64
		wantEnter             bool
	}{{1, 0, true}, {0, 1, false}} {
		v := ceilView(tt.headroom, true, spwUnfed(spwAgent(1, 2, 1), 10, 1))
		d := spwDecide(v)
		var enters int64
		for _, x := range d.List {
			if x.Kind == DecideEnter {
				enters += x.Households
			}
		}
		if (enters == 1) != tt.wantEnter || d.roleCeiling != tt.wantCeiling {
			t.Errorf("headroom %d: entries %d, roleCeiling %d, want entry %v and %d", tt.headroom, enters, d.roleCeiling, tt.wantEnter, tt.wantCeiling)
		}
	}
}

// TestCeiling_BirthsShareTheRoom verifies that the births the population module asks
// for take the room an admission left, and that the rest is counted in RoleCeiling.
//
// Arithmetic: headroom 3, one switcher admitted (room 2), PopBirths 3: two births enter
// and one is counted.
//
// Requirement: FR-SET-023, FR-POP-041
func TestCeiling_BirthsShareTheRoom(t *testing.T) {
	t.Parallel()
	v := ceilView(3, true, ceilAgents(1, 1)...)
	v.Roles[1].PopBirths = 3
	d := spwDecide(v)
	var births int64
	for _, x := range d.List {
		if x.Cause == CauseBirths {
			births += x.Households
		}
	}
	if births != 2 || d.roleCeiling != 1 {
		t.Errorf("Decide plan %+v: births %d roleCeiling %d, want 2 and 1", d.List, births, d.roleCeiling)
	}
}
