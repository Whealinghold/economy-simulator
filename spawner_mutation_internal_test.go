package economysim

import (
	"reflect"
	"slices"
	"testing"
)

// Equivalent mutants of spawner.go that no test can kill (gomu, one-line why each):
//   - prepare `slices.Grow(s.dests[:0], na*nr+nr)` (304, all four): only the
//     capacity hint; dests grows by append, and consider slices it after the append.
//   - prepare `c.households = 0` (305): the decider is built fresh per call.
//   - exits `len(s.exits) == 0` return and its variants (359): with no exits the
//     loop below it does nothing.
//   - exits `a.Unfed > 0 -> >=, !=` (349): occupation.Exits skips Unfed <= 0.
//   - switches `len(s.cands) == 0` return and its variants (399): Admit on no
//     candidate gives no admission and zero counts.
//   - eligible `return 0, false -> return 1, false` (429, 432, 435, 439): ri is
//     ignored when ok is false.
//   - consider `s.sig[ri].has` -> true (454): a role with no signal has e 0.
//   - consider `len(s.dests) == start -> <=` (460) and destinations (506 `<=`):
//     dests never shrinks below start.
//   - gapOf `e > eo -> >=` (538): equal values never overflow.
//   - affordable `ToolPrice > 0 -> !=` (526): a negative price is denied by
//     CapitalOK anyway.
//   - renewals `len(s.cands) > 0` variants (559): Admit on no candidate is a no-op.
//   - renewals `n > 0 -> !=` (566): lfl is only ever set above 0.
//   - renewals `p < len(s.adm) -> !=` (569) and `Ref == r -> <=` (569): p never
//     passes the length, and admissions come in ascending Ref, so none is left behind.
//   - size `!ok || n <= 0` variants (584) and `len(s.dests) == 0 -> <=` (587): a
//     zero-household batch admits nothing and counts nothing; n is never negative.

// TestRoleSignals_ViableAtRatio verifies that V == EntryRatio is viable.
//
// Arithmetic: wage 800, upkeep 1 000 gives V = 800 * 10 000 / 1 000 = 8 000.
//
// Guard: viable is V >= ratio, so > would refuse the exact ratio.
//
// Requirement: FR-SPN-017
func TestRoleSignals_ViableAtRatio(t *testing.T) {
	t.Parallel()
	r := spwRole(1, 800, 1)
	if _, v, has, viable := roleSignals(&r, 365, 8000); v != 8000 || !has || !viable {
		t.Errorf("roleSignals(wage 800, ratio 8000) V = %d, has = %v, viable = %v, want 8000, true, true", v, has, viable)
	}
}

// TestDecider_RoleIdx verifies the lookup by exact id: a missing id is -1 even
// when a larger id is present.
//
// Guard: >= would find the first larger id, and -2 is not -1.
//
// Requirement: FR-SPN-021
func TestDecider_RoleIdx(t *testing.T) {
	t.Parallel()
	v := spwView()
	v.Roles = []RoleView{spwRole(2, wOwn, 1), spwRole(5, wOwn, 1), spwRole(9, wOwn, 1)}
	c := decider{v: v}
	for id, want := range map[RoleID]int{2: 0, 5: 1, 9: 2, 1: -1, 3: -1, 10: -1} {
		if got := c.roleIdx(id); got != want {
			t.Errorf("roleIdx(%d) = %d, want %d", id, got, want)
		}
	}
}

// TestDecide_TrainingClosesEntry verifies that any positive training closes a
// role to the default, and a negative one does not.
//
// Guard: the test is TrainingTicks <= 0, so <= 1 would open a role of 1 tick and
// == 0 would close a role of -1.
//
// Requirement: FR-SPN-018
func TestDecide_TrainingClosesEntry(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		training int64
		want     []Decision
	}{
		{"1 tick of training is closed", 1, nil},
		{"negative training is open", -1, []Decision{spwSwitch(1, 2, CauseDistress)}},
	} {
		v := spwView()
		v.Renewal = 0
		r2 := spwRole(2, w1320, 100)
		r2.TrainingTicks = tt.training
		v.Roles = []RoleView{spwRole(1, wOwn, 100), r2}
		v.Agents = []RegionAgent{spwPriv(spwAgent(1, 1, 1), 5)}
		if got := spwDecide(v).List; !slices.Equal(got, tt.want) {
			t.Errorf("%s: plan = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// TestDecide_ExitShareEdges verifies that a negative ExitShare is off (nothing
// is cut, nothing counted) and that ExitShare 1 is on.
//
// Arithmetic: 10 000 households at 1 basis point give an allowance of 1.
//
// Guard: the test is ExitShare <= 0, so == 0 would run rung 3 with a negative
// allowance and count the leaver in ExitCap, and <= 1 would turn rung 3 off.
//
// Requirement: FR-SPN-013, FR-SPN-014
func TestDecide_ExitShareEdges(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		share   int64
		want    []Decision
		wantCap int64
	}{
		{-1, nil, 0},
		{1, []Decision{{Kind: DecideRemove, Agent: 1, Households: 1, Cause: CauseStarved}}, 0},
	} {
		v := spwView()
		v.Renewal, v.ExitShare = 0, tt.share
		v.Roles = []RoleView{spwRole(1, wOwn, 10000)}
		v.Agents = []RegionAgent{spwUnfed(spwAgent(1, 1, 1), 10, 1)}
		d := spwDecide(v)
		if !slices.Equal(d.List, tt.want) || d.exitCap != tt.wantCap {
			t.Errorf("ExitShare %d: plan = %+v, exitCap = %d, want %+v, %d", tt.share, d.List, d.exitCap, tt.want, tt.wantCap)
		}
	}
}

// TestDecide_ExitOfUnlistedRole verifies that an agent whose role is not in the
// view is still removed, and books no role.
//
// Guard: the role index is used only when found; true would index -1.
//
// Requirement: FR-SPN-013
func TestDecide_ExitOfUnlistedRole(t *testing.T) {
	t.Parallel()
	v := spwView()
	v.Roles = []RoleView{spwRole(1, wOwn, 1)}
	v.Agents = []RegionAgent{spwUnfed(spwAgent(1, 99, 1), 10, 1)}
	want := []Decision{{Kind: DecideRemove, Agent: 1, Households: 1, Cause: CauseStarved}}
	if got := spwDecide(v).List; !slices.Equal(got, want) {
		t.Errorf("plan = %+v, want %+v", got, want)
	}
}

// TestDecide_NoSwitchForNonPositiveWeight verifies that an agent of weight 0 or
// below is never a candidate.
//
// Guard: the test is Weight <= 0, so < would admit weight 0 and == would admit -1.
//
// Requirement: FR-SPN-021
func TestDecide_NoSwitchForNonPositiveWeight(t *testing.T) {
	t.Parallel()
	for _, w := range []int64{0, -1} {
		v := spwView()
		v.Renewal = 0
		v.Roles = []RoleView{spwRole(1, wOwn, 100), spwRole(2, w1320, 100)}
		v.Agents = []RegionAgent{spwPriv(spwAgent(1, 1, w), 5)}
		if got := spwDecide(v).List; len(got) != 0 {
			t.Errorf("weight %d: plan = %+v, want empty", w, got)
		}
	}
}

// TestDecide_Rung1AtAndAboveChange verifies that privation equal to, and above,
// ChangePrivation is rung 1 without a review being due (ReviewTicks 0 never
// reviews).
//
// Guard: rung 1 is Privation >= ChangePrivation, so <= in the guard would need
// a review at equality and != would need one above it.
//
// Requirement: FR-SPN-020
func TestDecide_Rung1AtAndAboveChange(t *testing.T) {
	t.Parallel()
	for _, p := range []int64{5, 6} {
		v := spwView()
		v.Renewal, v.ReviewTicks = 0, 0
		v.Roles = []RoleView{spwRole(1, wOwn, 100), spwRole(2, w1320, 100)}
		v.Agents = []RegionAgent{spwPriv(spwAgent(1, 1, 1), p)}
		want := []Decision{spwSwitch(1, 2, CauseDistress)}
		if got := spwDecide(v).List; !slices.Equal(got, want) {
			t.Errorf("privation %d: plan = %+v, want %+v", p, got, want)
		}
	}
}

// TestDecide_UnaffordableBestFallsToNext verifies that a best destination the
// agent cannot afford is skipped, not the end of the search.
//
// Guard: continue -> break would strand the agent.
//
// Requirement: FR-SPN-058
func TestDecide_UnaffordableBestFallsToNext(t *testing.T) {
	t.Parallel()
	v := spwView()
	v.Renewal = 0
	r2 := spwRole(2, w1320, 100)
	r2.ToolPrice, r2.CapitalReturn = 1_000_000, 1
	v.Roles = []RoleView{spwRole(1, wOwn, 100), r2, spwRole(3, w1300, 100)}
	v.Agents = []RegionAgent{spwPriv(spwAgent(1, 1, 1), 5)}
	want := []Decision{spwSwitch(1, 3, CauseDistress)}
	if got := spwDecide(v).List; !slices.Equal(got, want) {
		t.Errorf("plan = %+v, want %+v", got, want)
	}
}

// TestDecide_ToolMoneyEdges verifies the tool test: a tool the agent's money
// does not cover refuses (even a price of 1), and a role with no tool is open to
// an agent with negative money.
//
// Guard: the test is ToolPrice > 0 && Money < ToolPrice, so > 1 would pass a
// price of 1 on to the payback test, and >= 0 would refuse a role without tools.
//
// Requirement: FR-SPN-058
func TestDecide_ToolMoneyEdges(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name         string
		price, money Money
		want         []Decision
	}{
		{"price 1, money 0", 1, 0, nil},
		{"no tool, money -1", 0, -1, []Decision{spwSwitch(1, 2, CauseDistress)}},
	} {
		v := spwView()
		v.Renewal = 0
		r2 := spwRole(2, w1320, 100)
		r2.ToolPrice, r2.CapitalReturn = tt.price, 1
		v.Roles = []RoleView{spwRole(1, wOwn, 100), r2}
		a := spwPriv(spwAgent(1, 1, 1), 5)
		a.Money = tt.money
		v.Agents = []RegionAgent{a}
		if got := spwDecide(v).List; !slices.Equal(got, tt.want) {
			t.Errorf("%s: plan = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// TestDecide_HysteresisCounts verifies the Hysteresis count: only the best
// candidate role is judged, and only when it is better than the agent's own.
//
// Arithmetic: own role E 8 219; roles at E 8 904 and 8 561 are better but under
// the band (9 041), so 3 households count once (not 6); a role of equal E
// (8 219) or lower (2 739) counts nothing.
//
// Guard: `first = false` removed counts every failing role; `>` as >= or != counts
// an equal or a worse role.
//
// Requirement: FR-SPN-060
func TestDecide_HysteresisCounts(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		others []RoleView
		want   int64
	}{
		{"two better, one count", []RoleView{spwRole(2, w1300, 100), spwRole(3, w1250, 100)}, 3},
		{"equal", []RoleView{spwRole(2, wOwn, 100)}, 0},
		{"worse", []RoleView{spwRole(2, wLow, 100)}, 0},
	} {
		v := spwView()
		v.Renewal, v.observed = 0, true
		v.Roles = append([]RoleView{spwRole(1, wOwn, 100)}, tt.others...)
		v.Agents = []RegionAgent{spwAgent(1, 1, 3)}
		d := spwDecide(v)
		if d.hysteresis != tt.want || len(d.List) != 0 {
			t.Errorf("%s: hysteresis = %d, plan = %+v, want %d and no plan", tt.name, d.hysteresis, d.List, tt.want)
		}
	}
}

// TestDecide_GapIsOfTheBestDestination verifies that a candidate with several
// qualifying roles is ordered by the gap to its best one.
//
// Arithmetic: E = ET * 27 (ET expected work ticks, wage 1 000). Agent 1 (E 2 739)
// qualifies for 8 219, 4 109 and agent 2's role (3 287): gap 5 480 to the best,
// 548 to the last. Agent 2 (E 3 287) qualifies for 8 219 and 4 109: gap 4 932 to
// the best, 822 to the last. With one place agent 1 goes first; ordered by the
// last destination, agent 2 would.
//
// Requirement: FR-SPN-021, FR-SPN-023
func TestDecide_GapIsOfTheBestDestination(t *testing.T) {
	t.Parallel()
	v := spwView()
	v.EntryShare = 25
	v.Renewal = 0
	v.Roles = []RoleView{spwEt(1, 100), spwEt(2, 120), spwEt(3, 300), spwEt(4, 150)}
	v.Agents = []RegionAgent{spwAgent(1, 1, 1), spwAgent(2, 2, 1)}
	want := []Decision{spwSwitch(1, 3, CauseReview)}
	if got := spwDecide(v).List; !slices.Equal(got, want) {
		t.Errorf("plan = %+v, want %+v", got, want)
	}
}

// spwEt is a role of wage 1 000 whose E is 27.4 * et.
func spwEt(id RoleID, et int64) RoleView {
	r := spwRole(id, 1000, 100)
	r.ExpectedWorkTicks = et
	return r
}

// TestDecide_ReviewersIgnorePrivation verifies that reviewers are ordered by gap
// alone: a reviewer with some privation below rung 1 does not go before one with
// none and a larger gap.
//
// Arithmetic: agent 1 (E 2 739, privation 0) has gap 5 480 to role 3; agent 2
// (E 5 479, privation 3) has gap 2 740.
//
// Guard: carrying the privation of a non-rung-1 agent into the key would put
// agent 2 first.
//
// Requirement: FR-SPN-021
func TestDecide_ReviewersIgnorePrivation(t *testing.T) {
	t.Parallel()
	v := spwView()
	v.EntryShare = 25
	v.Renewal = 0
	v.Roles = []RoleView{spwEt(1, 100), spwEt(2, 200), spwEt(3, 300)}
	v.Agents = []RegionAgent{spwAgent(1, 1, 1), spwPriv(spwAgent(2, 2, 1), 3)}
	want := []Decision{spwSwitch(1, 3, CauseReview)}
	if got := spwDecide(v).List; !slices.Equal(got, want) {
		t.Errorf("plan = %+v, want %+v", got, want)
	}
}

// TestDecide_RenewalIgnoresSwitchDestinations verifies that the destinations a
// review found (a role that beats the band without being viable) are not
// destinations of a replacement.
//
// Arithmetic: roles 1 (E 2 739) and 2 (E 4 109, V 6 000 under EntryRatio 8 000):
// agent 1 reviews into role 2; role 1 has a carry of one household and no viable
// role exists, so it is counted in NoViableRole, not entered into role 2.
//
// Guard: a destination list not emptied before the renewals would enter it.
//
// Requirement: FR-SPN-027, FR-SPN-028
func TestDecide_RenewalIgnoresSwitchDestinations(t *testing.T) {
	t.Parallel()
	v := spwView()
	r1 := spwRole(1, wLow, 100)
	r1.RenewalCarry = 10000
	v.Roles = []RoleView{r1, spwRole(2, wMid, 100)}
	v.Agents = []RegionAgent{spwAgent(1, 1, 1)}
	d := spwDecide(v)
	want := []Decision{spwSwitch(1, 2, CauseReview)}
	if !slices.Equal(d.List, want) || d.noViableRole != 1 {
		t.Errorf("plan = %+v, noViableRole = %d, want %+v and 1", d.List, d.noViableRole, want)
	}
}

// TestDecide_RenewalOrder verifies the order of the renewal entries: per role in
// role order, a like-for-like entry before the batch of a later role.
//
// Arithmetic: roles 1 (E 8 561), 2 (E 8 219, carry 10 000) and 3 (not viable,
// carry 10 000); role 2 enters itself, role 3's household goes to the best role 1.
//
// Guard: consuming the admissions at the first role would put the batch first.
//
// Requirement: FR-SPN-027
func TestDecide_RenewalOrder(t *testing.T) {
	t.Parallel()
	v := spwView()
	r2 := spwRole(2, w1250, 100)
	r2.RenewalCarry = 10000
	r3 := spwRole(3, wLow, 100)
	r3.RenewalCarry = 10000
	v.Roles = []RoleView{spwRole(1, w1320, 100), r2, r3}
	want := []Decision{spwEnter(2, 1), spwEnter(1, 1)}
	if got := spwDecide(v).List; !slices.Equal(got, want) {
		t.Errorf("plan = %+v, want %+v", got, want)
	}
}

// TestDecide_ScratchReuseIsClean verifies that one scratch reused over many
// regions gives the plan and the counts of a fresh scratch each time.
//
// Guard: a buffer not reset between calls (the clear of regrow, the region
// count, the rank, the leavers, the id order, the candidates, the destinations)
// leaks one region's state into the next.
//
// Requirement: NFR-SPN-001, FR-SPN-029
func TestDecide_ScratchReuseIsClean(t *testing.T) {
	t.Parallel()
	shared := new(decideScratch)
	var dShared Decisions
	for seed := uint64(1); seed <= 120; seed++ {
		v := spwGen(seed, 40+int(seed%3), 4)
		fresh := spwDecide(v)
		v.scratch = shared
		dShared.reset()
		DefaultSpawner{}.Decide(v, nil, &dShared)
		if !slices.Equal(dShared.List, fresh.List) || !reflect.DeepEqual(counts(&dShared), counts(fresh)) {
			t.Fatalf("seed %d: reused scratch gave %+v %v, fresh gave %+v %v", seed, dShared.List, counts(&dShared), fresh.List, counts(fresh))
		}
	}
}

func counts(d *Decisions) []int64 {
	return append([]int64{d.regionCap, d.roleCap, d.cooldown, d.hysteresis, d.exitCap, d.noViableRole, d.stranded}, d.noViableByRole...)
}
