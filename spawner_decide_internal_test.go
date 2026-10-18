package economysim

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// The wages used below, with upkeep 1 000, ExpectedWorkTicks 250 and 365 ticks
// a year (E = wage * 250 * 10 000 / (1 000 * 365), V = wage * 10 000 / 1 000):
//
//	wage  400: E = 1 000 000 000 / 365 000 = 2 739   V =  4 000 (not viable at 8 000)
//	wage  600: E = 1 500 000 000 / 365 000 = 4 109   V =  6 000
//	wage 1 200: E = 3 000 000 000 / 365 000 = 8 219  V = 12 000 (SPN 15.1)
//	wage 1 250: E = 3 125 000 000 / 365 000 = 8 561  V = 12 500
//	wage 1 300: E = 3 250 000 000 / 365 000 = 8 904  V = 13 000
//	wage 1 319: E = 3 297 500 000 / 365 000 = 9 034  (below 9 041, the band of 15.1)
//	wage 1 320: E = 3 300 000 000 / 365 000 = 9 041  V = 13 200 (the band of 15.1)
const (
	wLow, wMid, wOwn, w1250, w1300, w1319, w1320 = 400, 600, 1200, 1250, 1300, 1319, 1320
)

// spwView is a region at tick 100 with the defaults of the tests: review every
// tick, EntryShare 100 %, a band of 1 000, cooldown 5, EntryRatio 8 000, rung 1
// at privation 5 and rung 3 at 10, full renewal.
func spwView() *RegionView {
	return &RegionView{
		Tick: 100, TicksPerYear: 365, ReviewTicks: 1, EntryShare: 10000, RoleHysteresis: 1000,
		RoleCooldown: 5, EntryRatio: 8000, ChangePrivation: 5, RemovePrivation: 10, DeathStep: 1000,
		ExitShare: 10000, Renewal: 10000, EntryPoolShare: 5000,
	}
}

func spwRole(id RoleID, wage Money, hh int64) RoleView {
	return RoleView{Role: id, Households: hh, Agents: hh, Wage: wage, HasWage: true, Upkeep: 1000, ExpectedWorkTicks: 250}
}

func spwAgent(id AgentID, role RoleID, w int64) RegionAgent {
	return RegionAgent{Agent: id, Role: role, Weight: w, Money: 5000, Upkeep: 1000}
}

func spwPriv(a RegionAgent, privation int64) RegionAgent { a.Privation = privation; return a }

func spwUnfed(a RegionAgent, privation, unfed int64) RegionAgent {
	a.Privation, a.Unfed = privation, unfed
	return a
}

// spwDecide runs the default spawner on a fresh plan.
func spwDecide(v *RegionView) *Decisions {
	var d Decisions
	d.reset()
	DefaultSpawner{}.Decide(v, nil, &d)
	return &d
}

func spwSwitch(agent AgentID, role RoleID, cause Cause) Decision {
	return Decision{Kind: DecideSwitch, Agent: agent, Role: role, Cause: cause}
}

func spwEnter(role RoleID, n int64) Decision {
	return Decision{Kind: DecideEnter, Role: role, Households: n, Cause: CauseRenewal}
}

// TestDecide_Exits verifies rung 3: a weight-1 agent is removed whole, a cohort
// of 40 with 7 unfed loses 4 (SPN 15.4), the allowance cuts the drop and the
// agent that cannot be served waits (counted in ExitCap), ExitShare 0 turns the
// rung off, and an agent with Unfed 0, an external agent, a privation below
// RemovePrivation, an agent created this tick or one still inside its
// RemovePrivation window is never touched.
//
// Requirement: FR-SPN-013, FR-SPN-014, FR-SPN-015, FR-SPN-016, FR-SPN-050, FR-SPN-059
func TestDecide_Exits(t *testing.T) {
	t.Parallel()
	cohort := func(money Money, upkeep Money) RegionAgent {
		a := spwUnfed(spwAgent(1, 1, 40), 12, 7)
		a.Money, a.Upkeep = money, upkeep
		return a
	}
	ext := spwUnfed(spwAgent(1, 1, 1), 10, 1)
	ext.External = true
	fresh := spwUnfed(spwAgent(1, 1, 1), 10, 1)
	fresh.RoleSince = 100
	young := spwUnfed(spwAgent(1, 1, 1), 10, 1)
	young.RoleSince = 95
	tests := []struct {
		name        string
		share       int64
		agents      []RegionAgent
		want        []Decision
		wantExitCap int64
		otherHouseh int64
	}{
		// allowance = ceil(1 * 10 000 / 10 000) = 1; drop = min(1, max(1, ceil(1 * 1 000 / 10 000))) = 1 = weight.
		{"weight 1 removed whole", 10000, []RegionAgent{spwUnfed(spwAgent(1, 1, 1), 10, 1)},
			[]Decision{{Kind: DecideRemove, Agent: 1, Households: 1, Cause: CauseStarved}}, 0, 0},
		// drop = min(7, max(1, ceil(40 * 1 000 / 10 000))) = min(7, 4) = 4; money 12 000 < 1 000 * 40 so insolvent.
		{"cohort of 40 loses 4, insolvent", 10000, []RegionAgent{cohort(12000, 1000)},
			[]Decision{{Kind: DecideReduce, Agent: 1, Households: 4, Cause: CauseInsolvent}}, 0, 0},
		// money 12 000 >= 100 * 40 = 4 000 so starved.
		{"cohort of 40 loses 4, starved", 10000, []RegionAgent{cohort(12000, 100)},
			[]Decision{{Kind: DecideReduce, Agent: 1, Households: 4, Cause: CauseStarved}}, 0, 0},
		// 1 000 households, ExitShare 50: allowance ceil(1 000 * 50 / 10 000) = 5; the cohort (privation 12) drops 4, the weight-1 agent (privation 10) drops 1.
		{"allowance 5 serves both", 50, []RegionAgent{cohort(12000, 100), spwUnfed(spwAgent(2, 1, 1), 10, 1)},
			[]Decision{{Kind: DecideReduce, Agent: 1, Households: 4, Cause: CauseStarved}, {Kind: DecideRemove, Agent: 2, Households: 1, Cause: CauseStarved}}, 0, 960 - 1},
		// ExitShare 20: allowance ceil(1 000 * 20 / 10 000) = 2; the cohort drops min(4, 2) = 2 (cut 2) and the weight-1 agent waits (cut 1): ExitCap 3.
		{"allowance 2 cuts and waits", 20, []RegionAgent{cohort(12000, 100), spwUnfed(spwAgent(2, 1, 1), 10, 1)},
			[]Decision{{Kind: DecideReduce, Agent: 1, Households: 2, Cause: CauseStarved}}, 3, 960 - 1},
		{"ExitShare 0 is off and counts nothing", 0, []RegionAgent{spwUnfed(spwAgent(1, 1, 1), 10, 1)}, nil, 0, 0},
		{"Unfed 0 gets no reduction", 10000, []RegionAgent{spwUnfed(spwAgent(1, 1, 1), 10, 0)}, nil, 0, 0},
		{"privation below RemovePrivation", 10000, []RegionAgent{spwUnfed(spwAgent(1, 1, 1), 9, 1)}, nil, 0, 0},
		{"external agent", 10000, []RegionAgent{ext}, nil, 0, 0},
		{"created this tick", 10000, []RegionAgent{fresh}, nil, 0, 0},
		// tick - RoleSince = 5 < RemovePrivation 10.
		{"inside the RemovePrivation window", 10000, []RegionAgent{young}, nil, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := spwView()
			var hh int64
			for _, a := range tt.agents {
				hh += a.Weight
			}
			v.ExitShare = tt.share
			v.Roles = []RoleView{spwRole(1, wOwn, hh+tt.otherHouseh)}
			v.Agents = tt.agents
			v.Renewal = 0
			d := spwDecide(v)
			if !slices.Equal(d.List, tt.want) {
				t.Errorf("Decide plan = %+v, want %+v", d.List, tt.want)
			}
			if d.exitCap != tt.wantExitCap {
				t.Errorf("Decide exitCap = %d, want %d", d.exitCap, tt.wantExitCap)
			}
		})
	}
}

// TestDecide_PrivationTrace verifies the privation of SPN 15.6 as
// household.NextPrivation produces it and that the ladder reads it: the unfed
// pattern gives 1, 2, 3, 2, 3, 4, 5, 4, 5, 6, 7, 8, 9, 10, and the agent is
// removed only at the 14th tick, when privation reaches RemovePrivation 10.
//
// Requirement: FR-SPN-012, FR-SPN-013
func TestDecide_PrivationTrace(t *testing.T) {
	t.Parallel()
	unfed := []int64{1, 1, 1, 0, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1}
	want := []int64{1, 2, 3, 2, 3, 4, 5, 4, 5, 6, 7, 8, 9, 10}
	var p int64
	for i, u := range unfed {
		p = household.NextPrivation(p, u)
		if p != want[i] {
			t.Fatalf("tick %d: NextPrivation = %d, want %d", i+1, p, want[i])
		}
		v := spwView()
		v.Roles = []RoleView{spwRole(1, wOwn, 1)}
		v.Agents = []RegionAgent{spwUnfed(spwAgent(1, 1, 1), p, u)}
		v.Renewal = 0
		got := spwDecide(v).List
		if last := i == len(unfed)-1; last != (len(got) == 1 && got[0].Kind == DecideRemove) {
			t.Errorf("tick %d with privation %d: plan %+v, removal expected only at privation 10", i+1, p, got)
		}
	}
}

// TestDecide_Review verifies the review: only on the rotation of SPN 15.2 (id
// 17, period 10: ticks 0, 10, 20), not inside the cooldown (counted in
// Cooldown), only through the band of SPN 15.1 (9 041 passes, 9 034 fails and is
// counted in Hysteresis when observed), with no viability term, to the highest
// E with ties to the lower role ID, and never to a closed role, a role with
// training or a role with no wage; an own role with no wage counts as E_o 0.
//
// Requirement: FR-SPN-017, FR-SPN-018, FR-SPN-019, FR-SPN-041
func TestDecide_Review(t *testing.T) {
	t.Parallel()
	closed := spwRole(2, w1320, 50)
	closed.Closed = true
	training := spwRole(2, w1320, 50)
	training.TrainingTicks = 10
	noWage := spwRole(2, w1320, 50)
	noWage.HasWage = false
	noUpkeep := spwRole(2, w1320, 50)
	noUpkeep.Upkeep = 0
	ownNoWage := spwRole(1, wOwn, 50)
	ownNoWage.HasWage = false
	tests := []struct {
		name         string
		reviewTicks  int64
		tick         int64
		roleSince    int64
		own, dest    RoleView
		extra        []RoleView
		entryRatio   int64
		want         []Decision
		wantHyst     int64
		wantCooldown int64
	}{
		{"band met: 9 041 >= ceil(8 219 * 11 000 / 10 000)", 1, 100, 0, spwRole(1, wOwn, 50), spwRole(2, w1320, 50), nil, 8000,
			[]Decision{spwSwitch(17, 2, CauseReview)}, 0, 0},
		{"band missed: 9 034 < 9 041", 1, 100, 0, spwRole(1, wOwn, 50), spwRole(2, w1319, 50), nil, 8000, nil, 1, 0},
		{"no viability term for a reviewer", 1, 100, 0, spwRole(1, wOwn, 50), spwRole(2, w1320, 50), nil, 20000,
			[]Decision{spwSwitch(17, 2, CauseReview)}, 0, 0},
		{"on its tick (100 mod 10 = 0, bucket 0)", 10, 100, 0, spwRole(1, wOwn, 50), spwRole(2, w1320, 50), nil, 8000,
			[]Decision{spwSwitch(17, 2, CauseReview)}, 0, 0},
		{"off its tick (105 mod 10 = 5)", 10, 105, 0, spwRole(1, wOwn, 50), spwRole(2, w1320, 50), nil, 8000, nil, 0, 0},
		{"inside the cooldown (100 - 97 = 3 < 5)", 1, 100, 97, spwRole(1, wOwn, 50), spwRole(2, w1320, 50), nil, 8000, nil, 0, 1},
		{"created this tick", 1, 100, 100, spwRole(1, wOwn, 50), spwRole(2, w1320, 50), nil, 8000, nil, 0, 1},
		{"closed destination", 1, 100, 0, spwRole(1, wOwn, 50), closed, nil, 8000, nil, 0, 0},
		{"destination with training", 1, 100, 0, spwRole(1, wOwn, 50), training, nil, 8000, nil, 0, 0},
		{"destination with no wage", 1, 100, 0, spwRole(1, wOwn, 50), noWage, nil, 8000, nil, 0, 0},
		{"destination with no upkeep", 1, 100, 0, spwRole(1, wOwn, 50), noUpkeep, nil, 8000, nil, 0, 0},
		{"own role with no wage counts as E_o 0", 1, 100, 0, ownNoWage, spwRole(2, wOwn, 50), nil, 8000,
			[]Decision{spwSwitch(17, 2, CauseReview)}, 0, 0},
		{"highest E wins", 1, 100, 0, spwRole(1, wOwn, 50), spwRole(2, w1320, 50), []RoleView{spwRole(3, w1300+100, 50)}, 8000,
			[]Decision{spwSwitch(17, 3, CauseReview)}, 0, 0},
		{"equal E goes to the lower role ID", 1, 100, 0, spwRole(1, wOwn, 50), spwRole(5, w1320, 50), []RoleView{spwRole(3, w1320, 50)}, 8000,
			[]Decision{spwSwitch(17, 3, CauseReview)}, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := spwView()
			v.ReviewTicks, v.Tick, v.EntryRatio, v.observed = tt.reviewTicks, tt.tick, tt.entryRatio, true
			a := spwAgent(17, 1, 1)
			a.RoleSince = tt.roleSince
			v.Roles = append([]RoleView{tt.own, tt.dest}, tt.extra...)
			v.Agents = []RegionAgent{a}
			d := spwDecide(v)
			if !slices.Equal(d.List, tt.want) {
				t.Errorf("Decide plan = %+v, want %+v", d.List, tt.want)
			}
			if d.hysteresis != tt.wantHyst || d.cooldown != tt.wantCooldown {
				t.Errorf("Decide hysteresis, cooldown = %d, %d, want %d, %d", d.hysteresis, d.cooldown, tt.wantHyst, tt.wantCooldown)
			}
		})
	}
}

// TestDecide_ReviewHysteresisNeedsObserver verifies that the Hysteresis count is
// built only when the view says an observer is attached (observed), and that the
// plan is the same either way.
//
// Requirement: FR-SPN-019
func TestDecide_ReviewHysteresisNeedsObserver(t *testing.T) {
	t.Parallel()
	for _, observed := range []bool{false, true} {
		v := spwView()
		v.observed = observed
		v.Roles = []RoleView{spwRole(1, wOwn, 50), spwRole(2, w1319, 50)}
		v.Agents = []RegionAgent{spwAgent(1, 1, 3)}
		d := spwDecide(v)
		want := int64(0)
		if observed {
			want = 3
		}
		if len(d.List) != 0 || d.hysteresis != want {
			t.Errorf("observed %v: plan %+v hysteresis %d, want an empty plan and %d", observed, d.List, d.hysteresis, want)
		}
	}
}

// TestDecide_Rung1 verifies rung 1: an agent at ChangePrivation takes the best
// viable role that beats its own with no band (8 561 < 9 041 still moves), not a
// role that is not viable or does not beat its own, and, when it is also due
// for a review, is judged by the rung-1 rule only: with no viable destination
// it stays and is not re-judged by the band, which a reviewer next to it
// passes.
//
// Requirement: FR-SPN-019, FR-SPN-023, FR-SPN-059, FR-SPN-060
func TestDecide_Rung1(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		dest       Money
		entryRatio int64
		want       []Decision
	}{
		// E_d 8 561 > E_o 8 219, V 12 500 >= 8 000: moves although 8 561 < 9 041.
		{"beats with no band", w1250, 8000, []Decision{spwSwitch(1, 2, CauseDistress)}},
		// V 12 500 < 13 000: not viable.
		{"not viable stays", w1250, 13000, nil},
		// E_d = E_o = 8 219: not strictly better.
		{"equal E stays", wOwn, 8000, nil},
		// E_d 2 739 < E_o: worse.
		{"worse stays", wLow, 2000, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := spwView()
			v.EntryRatio = tt.entryRatio
			v.Roles = []RoleView{spwRole(1, wOwn, 50), spwRole(2, tt.dest, 50)}
			v.Agents = []RegionAgent{spwPriv(spwAgent(1, 1, 1), 5)}
			if got := spwDecide(v).List; !slices.Equal(got, tt.want) {
				t.Errorf("Decide plan = %+v, want %+v", got, tt.want)
			}
		})
	}
	// Agent 1 (rung 1) and agent 2 (privation 0) are both due. The destination
	// passes the band (9 041) but V 13 200 < 14 000: rung 1 stays, the reviewer
	// goes.
	v := spwView()
	v.EntryRatio, v.observed = 14000, true
	v.Roles = []RoleView{spwRole(1, wOwn, 50), spwRole(2, w1320, 50)}
	v.Agents = []RegionAgent{spwPriv(spwAgent(1, 1, 1), 5), spwAgent(2, 1, 1)}
	d := spwDecide(v)
	if want := []Decision{spwSwitch(2, 2, CauseReview)}; !slices.Equal(d.List, want) || d.hysteresis != 0 {
		t.Errorf("rung 1 and reviewer: plan %+v hysteresis %d, want %+v and 0", d.List, d.hysteresis, want)
	}
}

// TestDecide_Stranded verifies that a household at rung 1 that has no role to
// move to is counted in Stranded, in households: a destination that is not
// viable, not better or paid for with a tool the agent cannot afford leaves a
// weight-3 agent stranded 3, while a destination it can take, a reviewer with
// no destination (it is not at rung 1), an external agent and an agent held by
// the cooldown (counted in Cooldown) leave Stranded at 0.
//
// Guard: without the count a household that starves for want of a role to move
// to is invisible until household.exits{starved} rises, rung-3 ticks later.
//
// Requirement: FR-SPN-035, FR-SPN-059, FR-SPN-061
func TestDecide_Stranded(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		dest       Money
		entryRatio int64
		tool       Money
		agent      func(RegionAgent) RegionAgent
		want       int64
	}{
		{"moves, not stranded", w1250, 8000, 0, func(a RegionAgent) RegionAgent { return a }, 0},
		// V 12 500 < 13 000: the destination is not viable.
		{"not viable", w1250, 13000, 0, func(a RegionAgent) RegionAgent { return a }, 3},
		// E_d = E_o: not strictly better.
		{"not better", wOwn, 8000, 0, func(a RegionAgent) RegionAgent { return a }, 3},
		// A tool of 600 000 000 against money 5 000: the capital test fails.
		{"tool unaffordable", w1250, 8000, 600_000_000, func(a RegionAgent) RegionAgent { return a }, 3},
		// Privation 0 is a reviewer, not rung 1; it is held by the band, not stranded.
		{"reviewer", wOwn, 8000, 0, func(a RegionAgent) RegionAgent { a.Privation = 0; return a }, 0},
		{"external", wOwn, 8000, 0, func(a RegionAgent) RegionAgent { a.External = true; return a }, 0},
		// RoleSince 98 at tick 100 with RoleCooldown 5: held by the cooldown.
		{"cooldown", wOwn, 8000, 0, func(a RegionAgent) RegionAgent { a.RoleSince = 98; return a }, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := spwView()
			v.EntryRatio = tt.entryRatio
			dest := spwRole(2, tt.dest, 50)
			dest.ToolPrice, dest.CapitalReturn = tt.tool, 3
			v.Roles = []RoleView{spwRole(1, wOwn, 50), dest}
			v.Agents = []RegionAgent{tt.agent(spwPriv(spwAgent(1, 1, 3), 5))}
			if got := spwDecide(v).stranded; got != tt.want {
				t.Errorf("Decide stranded = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestDecide_Caps verifies the entry limit of SPN 15.3: a region of 1 000
// households with EntryShare 50 admits 5 changes and a destination of 450 admits
// ceil(450 * 50 / 10 000) = 3, in the order of need (privation descending), the
// next best taking the overflow, the counts in households, and EntryShare 0
// counting every candidate in RegionCap.
//
// Requirement: FR-SPN-016, FR-SPN-021, FR-SPN-060
func TestDecide_Caps(t *testing.T) {
	t.Parallel()
	// Ten weight-1 agents of role 1 at privations 5 to 14 (id i has 4 + i),
	// Unfed 0 so rung 3 is quiet.
	agents := make([]RegionAgent, 0, 10)
	for i := AgentID(1); i <= 10; i++ {
		agents = append(agents, spwPriv(spwAgent(i, 1, 1), 4+int64(i)))
	}
	tests := []struct {
		name                    string
		share                   int64
		hh                      [3]int64
		want                    []Decision
		wantRegionCap, wantRole int64
	}{
		// Region cap ceil(1 000 * 50 / 10 000) = 5; role caps ceil(450 * 50 / 10 000) = 3.
		// Ids 10, 9, 8 fill role 2 (E 8 904); id 7 takes role 3 (E 8 561), id 6 too; ids 5..1 are held by the region cap.
		{"region cap binds", 50, [3]int64{100, 450, 450}, []Decision{
			spwSwitch(10, 2, CauseDistress), spwSwitch(9, 2, CauseDistress), spwSwitch(8, 2, CauseDistress),
			spwSwitch(7, 3, CauseDistress), spwSwitch(6, 3, CauseDistress)}, 5, 0},
		// Role caps ceil(400 * 50 / 10 000) = 2 and ceil(50 * 50 / 10 000) = 1: three admitted, seven find every destination full.
		{"role caps bind", 50, [3]int64{550, 400, 50}, []Decision{
			spwSwitch(10, 2, CauseDistress), spwSwitch(9, 2, CauseDistress), spwSwitch(8, 3, CauseDistress)}, 0, 7},
		{"EntryShare 0 counts every candidate in RegionCap", 0, [3]int64{550, 400, 50}, nil, 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := spwView()
			v.EntryShare = tt.share
			v.Roles = []RoleView{spwRole(1, wOwn, tt.hh[0]), spwRole(2, w1300, tt.hh[1]), spwRole(3, w1250, tt.hh[2])}
			v.Agents = agents
			d := spwDecide(v)
			if !slices.Equal(d.List, tt.want) || d.regionCap != tt.wantRegionCap || d.roleCap != tt.wantRole {
				t.Errorf("Decide plan %+v regionCap %d roleCap %d, want %+v, %d, %d", d.List, d.regionCap, d.roleCap, tt.want, tt.wantRegionCap, tt.wantRole)
			}
		})
	}
}

// TestDecide_Tool verifies a role that needs a tool: it is entered only when the
// money covers ToolPrice and the capital test holds. Moving from E_o 4 109 to
// E_d 8 219 is diff 4 110; ut = 1 000 * 365 = 365 000; diff * ut = 1 500 150 000;
// with CapitalReturn 3 a price of 400 000 000 gives 1 200 000 000 and passes, a
// price of 600 000 000 gives 1 800 000 000 and fails.
//
// Requirement: FR-SPN-020, FR-SPN-058
func TestDecide_Tool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		price Money
		money Money
		want  []Decision
	}{
		{"no tool", 0, 0, []Decision{spwSwitch(1, 2, CauseDistress)}},
		{"tool affordable and paid back", 400_000_000, 700_000_000, []Decision{spwSwitch(1, 2, CauseDistress)}},
		{"money one short of the price", 400_000_000, 399_999_999, nil},
		{"money exactly the price", 400_000_000, 400_000_000, []Decision{spwSwitch(1, 2, CauseDistress)}},
		{"price too high for the gain", 600_000_000, 700_000_000, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := spwView()
			dest := spwRole(2, wOwn, 50)
			dest.ToolPrice, dest.CapitalReturn = tt.price, 3
			v.Roles = []RoleView{spwRole(1, wMid, 50), dest}
			a := spwPriv(spwAgent(1, 1, 1), 5)
			a.Money = tt.money
			v.Agents = []RegionAgent{a}
			if got := spwDecide(v).List; !slices.Equal(got, tt.want) {
				t.Errorf("Decide plan = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestDecide_RenewalCarry verifies SPN 15.7: with Renewal 5 000 and one removal
// per tick, the carry fed back by the world gives entries 0, 1, 0, 1 into the
// removed agent's role, and the carry is the view's: Decide does not keep it.
//
// Requirement: FR-SPN-024, FR-SPN-025
func TestDecide_RenewalCarry(t *testing.T) {
	t.Parallel()
	var carry int64
	for i, want := range []int64{0, 1, 0, 1} {
		v := spwView()
		v.Renewal = 5000
		r := spwRole(1, wOwn, 1)
		r.RenewalCarry = carry
		v.Roles = []RoleView{r}
		v.Agents = []RegionAgent{spwUnfed(spwAgent(1, 1, 1), 10, 1)}
		d := spwDecide(v)
		var got int64
		for _, x := range d.List {
			if x.Kind == DecideEnter {
				got += x.Households
			}
		}
		if got != want {
			t.Errorf("tick %d with carry %d: entries = %d, want %d", i, carry, got, want)
		}
		// carry + 1 removed * 5 000, mod 10 000: 5 000, 0, 5 000, 0.
		_, carry, _ = occupation.Renewal(carry, 1, 5000)
	}
}

// TestDecide_Replacement verifies where the replacements go (SPN 7.2): to the
// leaver's own role when it is open and viable (uncapped, even with EntryShare
// 0), else to the viable open role of highest E under the caps, else nowhere,
// counted in NoViableRole; and the entries come per leaver role in role order,
// like for like before the fallback, so a short budget funds the earlier role
// first (the test applies the plan in order: budget 15 000, StartMoney 10 000).
//
// Requirement: FR-SPN-022, FR-SPN-025, FR-SPN-031, FR-SPN-041
func TestDecide_Replacement(t *testing.T) {
	t.Parallel()
	gone := func(id AgentID, role RoleID) RegionAgent { return spwUnfed(spwAgent(id, role, 1), 10, 1) }
	closedOwn := spwRole(1, wOwn, 10)
	closedOwn.Closed = true
	tests := []struct {
		name         string
		share        int64
		roles        []RoleView
		agents       []RegionAgent
		want         []Decision
		wantRegion   int64
		wantNoViable int64
	}{
		{"like for like is uncapped", 0, []RoleView{spwRole(1, wOwn, 10), spwRole(2, w1300, 10)}, []RegionAgent{gone(1, 1)},
			[]Decision{{Kind: DecideRemove, Agent: 1, Households: 1, Cause: CauseStarved}, spwEnter(1, 1)}, 0, 0},
		{"fallback goes to the highest E", 10000, []RoleView{spwRole(1, wLow, 10), spwRole(2, wOwn, 10), spwRole(3, w1300, 10)}, []RegionAgent{gone(1, 1)},
			[]Decision{{Kind: DecideRemove, Agent: 1, Households: 1, Cause: CauseStarved}, spwEnter(3, 1)}, 0, 0},
		{"fallback is capped", 0, []RoleView{spwRole(1, wLow, 10), spwRole(2, w1300, 10)}, []RegionAgent{gone(1, 1)},
			[]Decision{{Kind: DecideRemove, Agent: 1, Households: 1, Cause: CauseStarved}}, 1, 0},
		{"no viable role drops the replacement", 10000, []RoleView{spwRole(1, wLow, 10), spwRole(2, wLow, 10)}, []RegionAgent{gone(1, 1)},
			[]Decision{{Kind: DecideRemove, Agent: 1, Households: 1, Cause: CauseStarved}}, 0, 1},
		{"a closed own role is not like for like", 10000, []RoleView{closedOwn, spwRole(2, w1300, 10)}, []RegionAgent{gone(1, 1)},
			[]Decision{{Kind: DecideRemove, Agent: 1, Households: 1, Cause: CauseStarved}, spwEnter(2, 1)}, 0, 0},
		{"entries per leaver role in role order", 10000, []RoleView{spwRole(1, wLow, 10), spwRole(2, wOwn, 10), spwRole(3, w1300, 10)}, []RegionAgent{gone(1, 1), gone(2, 2)},
			[]Decision{
				{Kind: DecideRemove, Agent: 1, Households: 1, Cause: CauseStarved},
				{Kind: DecideRemove, Agent: 2, Households: 1, Cause: CauseStarved},
				spwEnter(3, 1), spwEnter(2, 1)}, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := spwView()
			v.EntryShare = tt.share
			v.Roles, v.Agents = tt.roles, tt.agents
			d := spwDecide(v)
			if !slices.Equal(d.List, tt.want) || d.regionCap != tt.wantRegion || d.noViableRole != tt.wantNoViable {
				t.Errorf("Decide plan %+v regionCap %d noViable %d, want %+v, %d, %d", d.List, d.regionCap, d.noViableRole, tt.want, tt.wantRegion, tt.wantNoViable)
			}
			if tt.wantNoViable > 0 && d.noViableByRole[0] != tt.wantNoViable {
				t.Errorf("noViableByRole = %v, want %d for the leaver's role", d.noViableByRole, tt.wantNoViable)
			}
		})
	}
	// The short budget: the entries of the last case, applied in plan order with
	// the funded number min(n, floor(budget / StartMoney)), fund role 3 and not
	// role 2 (15 000 / 10 000 = 1 household).
	budget, start := int64(15000), int64(10000)
	funded := map[RoleID]int64{}
	for _, x := range tests[len(tests)-1].want {
		if x.Kind != DecideEnter {
			continue
		}
		n := occupation.Funded(x.Households, budget, start)
		funded[x.Role] = n
		budget -= n * start
	}
	if funded[3] != 1 || funded[2] != 0 {
		t.Errorf("funding in plan order = %v, want role 3 funded and role 2 not", funded)
	}
}

// TestDecide_BatchSpansDestinations verifies a fallback batch cut by the caps
// across destinations: five households replace in a region of 1 000 households
// whose EntryShare 50 gives role caps of 1 (50 households) and 2 (400): the batch
// takes 2 into the better role (E 8 904, cap 2), 1 into the next (cap 1) and
// the last 2 are counted in RoleCap, exactly as five one-at-a-time candidates
// would be.
//
// Requirement: FR-SPN-021, FR-SPN-025, FR-SPN-060
func TestDecide_BatchSpansDestinations(t *testing.T) {
	t.Parallel()
	v := spwView()
	v.EntryShare = 50
	v.Roles = []RoleView{spwRole(1, wLow, 500), spwRole(2, w1300, 400), spwRole(3, w1250, 50)}
	for i := AgentID(1); i <= 5; i++ {
		v.Agents = append(v.Agents, spwUnfed(spwAgent(i, 1, 1), 10, 1))
	}
	d := spwDecide(v)
	var enters []Decision
	for _, x := range d.List {
		if x.Kind == DecideEnter {
			enters = append(enters, x)
		}
	}
	if want := []Decision{spwEnter(2, 2), spwEnter(3, 1)}; !slices.Equal(enters, want) || d.roleCap != 2 || d.regionCap != 0 {
		t.Errorf("Decide entries %+v roleCap %d regionCap %d, want %+v, 2, 0", enters, d.roleCap, d.regionCap, want)
	}
}

// TestDecide_PlanOrder verifies the order of the plan (SPN 10.2): the reduction
// first, then the switches, then the renewal entries, and that Decide takes no
// budget (the view's Budget is advisory: a zero budget changes nothing).
//
// FR-SPN-026 is covered in part: Decide takes no budget; the world takes it once
// per region (row 11).
//
// Requirement: FR-SPN-026, FR-SPN-031
func TestDecide_PlanOrder(t *testing.T) {
	t.Parallel()
	v := spwView()
	v.Roles = []RoleView{spwRole(1, wOwn, 50), spwRole(2, w1300, 50)}
	v.Agents = []RegionAgent{spwUnfed(spwAgent(1, 1, 1), 10, 1), spwPriv(spwAgent(2, 1, 1), 5)}
	want := []Decision{
		{Kind: DecideRemove, Agent: 1, Households: 1, Cause: CauseStarved},
		spwSwitch(2, 2, CauseDistress),
		spwEnter(1, 1),
	}
	v.Budget = 0
	if got := spwDecide(v).List; !slices.Equal(got, want) {
		t.Errorf("Decide plan = %+v, want %+v", got, want)
	}
}

// TestDecide_Empty verifies that an empty region, a region of roles with no
// agents and a view with no roles give an empty plan without a panic.
//
// Requirement: FR-SPN-042
func TestDecide_Empty(t *testing.T) {
	t.Parallel()
	roles := []RoleView{spwRole(1, wOwn, 0), spwRole(2, w1300, 0)}
	tests := []struct {
		name   string
		roles  []RoleView
		agents []RegionAgent
	}{
		{"nothing", nil, nil},
		{"roles with no agents", roles, nil},
		{"agents with no roles", nil, []RegionAgent{spwUnfed(spwAgent(1, 1, 1), 10, 1), spwPriv(spwAgent(2, 1, 1), 7)}},
	}
	for _, tt := range tests {
		v := spwView()
		v.Roles, v.Agents = tt.roles, tt.agents
		if got := spwDecide(v); len(got.List) != 0 {
			t.Errorf("Decide(%s) plan = %+v, want empty", tt.name, got.List)
		}
	}
}

// spwGen builds a random view from a seed: up to nRoles roles and nAgents
// agents with small ranges, so that ties in privation, weight and E are common.
func spwGen(seed uint64, nAgents, nRoles int) *RegionView {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	v := spwView()
	v.Tick = 20 + int64(rng.IntN(20))
	v.ReviewTicks = 1 + int64(rng.IntN(4))
	v.EntryShare = int64(rng.IntN(4)) * 500
	v.RoleCooldown = int64(rng.IntN(4))
	v.Renewal = int64(rng.IntN(4)) * 4000
	v.ExitShare = int64(rng.IntN(4)) * 500
	v.EntryRatio = 6000 + int64(rng.IntN(4))*2000
	v.observed = rng.IntN(2) == 0
	nr := 1 + nRoles%5
	wages := []Money{wLow, wMid, wOwn, w1250, w1300, w1320}
	for i := range nr {
		r := spwRole(RoleID(2*i+1), wages[rng.IntN(len(wages))], 0)
		r.HasWage = rng.IntN(6) != 0
		r.Closed = rng.IntN(6) == 0
		r.RenewalCarry = int64(rng.IntN(10000))
		if rng.IntN(8) == 0 {
			r.TrainingTicks = 5
		}
		if rng.IntN(5) == 0 {
			r.ToolPrice, r.CapitalReturn = Money(rng.IntN(500))*1_000_000, int64(1+rng.IntN(5))
		}
		v.Roles = append(v.Roles, r)
	}
	for i := range nAgents % 40 {
		w := int64(1 + rng.IntN(3))
		a := spwAgent(AgentID(3*i+1), v.Roles[rng.IntN(nr)].Role, w)
		a.Privation = int64(rng.IntN(13))
		a.Unfed = int64(rng.IntN(int(w) + 1))
		a.RoleSince = int64(rng.IntN(int(v.Tick) + 2))
		a.External = rng.IntN(10) == 0
		a.Money = Money(rng.IntN(800)) * 1_000_000
		v.Agents = append(v.Agents, a)
		for j := range v.Roles {
			if v.Roles[j].Role == a.Role {
				v.Roles[j].Households += w
				v.Roles[j].Agents++
			}
		}
	}
	return v
}

// planCheck accumulates what spwCheckPlan learns from a plan.
type planCheck struct {
	agents                       map[AgentID]*RegionAgent
	roles                        map[RoleID]*RoleView
	acted                        map[AgentID]bool
	removed, into                map[RoleID]int64
	regionIn, entered, maxWeight int64
}

// decision checks one decision of a plan and books it.
func (p *planCheck) decision(t *testing.T, v *RegionView, x Decision) {
	t.Helper()
	if x.Kind == DecideEnter {
		if p.roles[x.Role] == nil || x.Households < 1 {
			t.Fatalf("entry %+v: unknown role or fewer than 1 household", x)
		}
		p.entered += x.Households
		return
	}
	a := p.agents[x.Agent]
	if a == nil || a.External || a.RoleSince >= v.Tick || p.acted[x.Agent] {
		t.Fatalf("decision %+v: unknown, external, created this tick or repeated agent", x)
	}
	p.acted[x.Agent] = true
	if x.Kind == DecideSwitch {
		p.switched(t, x, a)
		return
	}
	p.shed(t, x, a)
}

// shed checks and books one reduction or removal.
func (p *planCheck) shed(t *testing.T, x Decision, a *RegionAgent) {
	t.Helper()
	if x.Households < 1 || x.Households > a.Weight || x.Kind == DecideRemove && x.Households != a.Weight {
		t.Fatalf("decision %+v of weight %d: bad households", x, a.Weight)
	}
	p.removed[a.Role] += x.Households
}

// switched checks and books one switch.
func (p *planCheck) switched(t *testing.T, x Decision, a *RegionAgent) {
	t.Helper()
	r := p.roles[x.Role]
	if r == nil || r.Closed || r.TrainingTicks > 0 || !r.HasWage || x.Role == a.Role {
		t.Fatalf("switch %+v: bad destination %+v", x, r)
	}
	p.into[x.Role] += a.Weight
	p.regionIn += a.Weight
}

// caps checks that no cap is exceeded by more than one weight.
func (p *planCheck) caps(t *testing.T, v *RegionView) {
	t.Helper()
	var hh int64
	for i := range v.Roles {
		hh += v.Roles[i].Households
	}
	if capR := satAllowance(hh, v.EntryShare); p.regionIn > 0 && (p.regionIn > capR-1+p.maxWeight || capR == 0) {
		t.Fatalf("region admitted %d households, cap %d, max weight %d", p.regionIn, capR, p.maxWeight)
	}
	for id, n := range p.into {
		if capD := satAllowance(p.roles[id].Households, v.EntryShare); n > capD-1+p.maxWeight || capD == 0 {
			t.Fatalf("role %d admitted %d households, cap %d, max weight %d", id, n, capD, p.maxWeight)
		}
	}
}

// spwCheckPlan verifies the invariants of any plan of the default on v: every
// agent and role exists, no agent is acted on twice or when external or created
// this tick, weights and entries are positive and within the households, no cap
// is exceeded by more than one weight, and the entries plus the dropped
// replacements never exceed what the carries allow.
func spwCheckPlan(t *testing.T, v *RegionView, d *Decisions) {
	t.Helper()
	p := planCheck{
		agents: map[AgentID]*RegionAgent{}, roles: map[RoleID]*RoleView{}, acted: map[AgentID]bool{},
		removed: map[RoleID]int64{}, into: map[RoleID]int64{},
	}
	for i := range v.Agents {
		p.agents[v.Agents[i].Agent] = &v.Agents[i]
		p.maxWeight = max(p.maxWeight, v.Agents[i].Weight)
	}
	for i := range v.Roles {
		p.roles[v.Roles[i].Role] = &v.Roles[i]
	}
	for _, x := range d.List {
		p.decision(t, v, x)
	}
	p.caps(t, v)
	var due int64
	for i := range v.Roles {
		if n, _, ok := occupation.Renewal(v.Roles[i].RenewalCarry, p.removed[v.Roles[i].Role], v.Renewal); ok {
			due += n
		}
	}
	if p.entered+d.noViableRole > due {
		t.Fatalf("entries %d + dropped %d exceed the %d households due", p.entered, d.noViableRole, due)
	}
}

// TestDecide_Properties verifies, over 200 seeds and reporting the failing seed,
// that a plan satisfies the invariants of spwCheckPlan, that the same view gives
// the same plan twice and that permuting the Agents slice (IDs fixed) leaves the
// plan and the counts unchanged because every tie ends in the agent ID. The
// roles are not permuted: their order fixes the renewal entry order by design.
//
// Requirement: FR-SPN-021, FR-SPN-029, FR-SPN-030, FR-SPN-050, FR-SPN-059
func TestDecide_Properties(t *testing.T) {
	t.Parallel()
	for seed := range uint64(200) {
		v := spwGen(seed, 8+int(seed%30), int(seed))
		d1, d2 := spwDecide(v), spwDecide(v)
		if !reflect.DeepEqual(d1, d2) {
			t.Fatalf("seed %d: Decide twice gave %+v and %+v", seed, d1, d2)
		}
		spwCheckPlan(t, v, d1)
		p := *v
		p.Agents = slices.Clone(v.Agents)
		rand.New(rand.NewPCG(seed, 7)).Shuffle(len(p.Agents), func(i, j int) { p.Agents[i], p.Agents[j] = p.Agents[j], p.Agents[i] })
		if dp := spwDecide(&p); !reflect.DeepEqual(d1, dp) {
			t.Fatalf("seed %d: permuting Agents changed the plan: %+v, want %+v", seed, dp, d1)
		}
	}
}

// TestDecide_TiesEndInID verifies that equal-privation, equal-gap candidates are
// ordered by agent ID whatever their order in the view: with a region cap of 1,
// the lowest ID of four identical agents is admitted.
//
// Requirement: FR-SPN-021, FR-SPN-030
func TestDecide_TiesEndInID(t *testing.T) {
	t.Parallel()
	v := spwView()
	v.EntryShare = 25 // ceil(400 * 25 / 10 000) = 1
	v.Roles = []RoleView{spwRole(1, wOwn, 4), spwRole(2, w1300, 396)}
	v.Agents = []RegionAgent{spwPriv(spwAgent(9, 1, 1), 5), spwPriv(spwAgent(3, 1, 1), 5), spwPriv(spwAgent(7, 1, 1), 5), spwPriv(spwAgent(5, 1, 1), 5)}
	if got, want := spwDecide(v).List, []Decision{spwSwitch(3, 2, CauseDistress)}; !slices.Equal(got, want) {
		t.Errorf("Decide plan = %+v, want %+v", got, want)
	}
}

// TestDecide_NoAllocation verifies that, with the world's scratch and a plan
// with room, a call allocates nothing.
//
// Requirement: NFR-SPN-001
func TestDecide_NoAllocation(t *testing.T) {
	v := spwGen(11, 60, 4)
	v.scratch = new(decideScratch)
	d := Decisions{List: make([]Decision, 0, 256)}
	run := func() {
		d.reset()
		DefaultSpawner{}.Decide(v, nil, &d)
	}
	run()
	if len(d.List) == 0 {
		t.Fatal("seed 11 gives an empty plan; pick a seed that exercises the code")
	}
	if n := testing.AllocsPerRun(50, run); n != 0 {
		t.Errorf("Decide allocated %v times per call, want 0", n)
	}
}

// TestRoleSignals verifies the one signal function Decide uses: the values of
// SPN 15.1 (E 8 219, V 12 000, viable at 8 000), no signal without a wage or an
// upkeep, and a viability gate that excludes.
//
// Requirement: FR-SPN-017
func TestRoleSignals(t *testing.T) {
	t.Parallel()
	noWage := spwRole(1, wOwn, 1)
	noWage.HasWage = false
	noUpkeep := spwRole(1, wOwn, 1)
	noUpkeep.Upkeep = 0
	own := spwRole(1, wOwn, 1)
	tests := []struct {
		name        string
		r           RoleView
		ratio       int64
		e, v        int64
		has, viable bool
	}{
		{"SPN 15.1", own, 8000, 8219, 12000, true, true},
		{"not viable above V", own, 12001, 8219, 12000, true, false},
		{"no wage", noWage, 0, 0, 0, false, false},
		{"no upkeep", noUpkeep, 0, 0, 0, false, false},
	}
	for _, tt := range tests {
		e, v, has, viable := roleSignals(&tt.r, 365, tt.ratio)
		if e != tt.e || v != tt.v || has != tt.has || viable != tt.viable {
			t.Errorf("roleSignals(%s) = %d, %d, %v, %v, want %d, %d, %v, %v", tt.name, e, v, has, viable, tt.e, tt.v, tt.has, tt.viable)
		}
	}
}

// FuzzDecide verifies, for any seed, that Decide does not panic and that its
// plan satisfies the invariants of spwCheckPlan: every agent and role is in the
// view and every Households is at least 1.
//
// Requirement: FR-SPN-004, FR-SPN-029
func FuzzDecide(f *testing.F) {
	f.Add(uint64(1), uint8(10), uint8(3))
	f.Add(uint64(0), uint8(0), uint8(0))
	f.Add(uint64(99), uint8(39), uint8(4))
	f.Fuzz(func(t *testing.T, seed uint64, agents, roles uint8) {
		v := spwGen(seed, int(agents), int(roles))
		spwCheckPlan(t, v, spwDecide(v))
	})
}

// TestSpawnerSaturation verifies the two saturating helpers of Decide: a gap
// E_d - E_o beyond int64 is the extreme of its sign, and an allowance whose
// product leaves int64 is MaxInt64 (a cap that never binds).
//
// Guard: a wrapped gap or allowance would reorder candidates or close a cap
// silently; without this test only a hand-built view reaches the branches.
func TestSpawnerSaturation(t *testing.T) {
	t.Parallel()
	const maxI, minI = int64(1<<63 - 1), int64(-1 << 63)
	for _, tt := range []struct{ e, eo, want int64 }{{5, 3, 2}, {maxI, -1, maxI}, {minI, 1, minI}} {
		if got := gapOf(tt.e, tt.eo); got != tt.want {
			t.Errorf("gapOf(%d, %d) = %d, want %d", tt.e, tt.eo, got, tt.want)
		}
	}
	if got := satAllowance(maxI, 10000); got != maxI {
		t.Errorf("satAllowance(MaxInt64, 10 000) = %d, want MaxInt64", got)
	}
	if got := satAllowance(1000, 50); got != 5 {
		t.Errorf("satAllowance(1 000, 50) = %d, want ceil(1 000 * 50 / 10 000) = 5", got)
	}
	// satAdd: MaxInt64 + 1 saturates to MaxInt64 (not wrapped to MinInt64),
	// MinInt64 + -1 to MinInt64, and 2 + 3 is 5.
	for _, tt := range []struct{ p, n, want int64 }{{2, 3, 5}, {maxI, 1, maxI}, {minI, -1, minI}} {
		if got := satAdd(tt.p, tt.n); got != tt.want {
			t.Errorf("satAdd(%d, %d) = %d, want %d", tt.p, tt.n, got, tt.want)
		}
	}
}

// TestDecide_HouseholdSumSaturates verifies that two roles holding MaxInt64
// households give a region total of MaxInt64, not a wrapped negative, so the
// region cap stays positive: the plan is the same as with one such role.
//
// Guard: a wrapped sum is -2, making the region cap non-positive and blocking
// every switch.
//
// Requirement: FR-SPN-021
func TestDecide_HouseholdSumSaturates(t *testing.T) {
	t.Parallel()
	const maxI = int64(1<<63 - 1)
	v := spwView()
	v.Roles = []RoleView{spwRole(1, wOwn, maxI), spwRole(2, w1320, maxI)}
	v.Agents = []RegionAgent{spwPriv(spwAgent(1, 1, 1), 5)}
	if got, want := spwDecide(v).List, []Decision{spwSwitch(1, 2, CauseDistress)}; !slices.Equal(got, want) {
		t.Errorf("Decide plan = %+v, want %+v", got, want)
	}
}

// TestDecide_GapOfBestViable verifies that the gap that orders candidates is
// E_d - E_o of the best destination that qualifies, not of a higher-E role the
// agent was refused. Role 2 has E 10 849 (1 320 * 300 * 10 000 / (1 200 * 365))
// but V 11 000 < EntryRatio 12 000; role 3 has E 8 561 and V 12 500. Agent 1
// (role 1, E_o 8 219) has gap 342; agent 2 (role 4, E_o 2 739) has gap 5 822, so
// with one place agent 2 goes first although agent 1 has the lower ID.
//
// Guard: ordering by a wrong gap (0 for both) would fall back to the ID and
// serve the less needy agent first.
//
// Requirement: FR-SPN-021, FR-SPN-023, FR-SPN-060
func TestDecide_GapOfBestViable(t *testing.T) {
	t.Parallel()
	v := spwView()
	v.EntryShare, v.EntryRatio = 25, 12000 // ceil(400 * 25 / 10 000) = 1 and ceil(100 * 25 / 10 000) = 1
	r2 := spwRole(2, w1320, 100)
	r2.Upkeep, r2.ExpectedWorkTicks = 1200, 300
	v.Roles = []RoleView{spwRole(1, wOwn, 100), r2, spwRole(3, w1250, 100), spwRole(4, wLow, 100)}
	v.Agents = []RegionAgent{spwPriv(spwAgent(1, 1, 1), 5), spwPriv(spwAgent(2, 4, 1), 5)}
	if got, want := spwDecide(v).List, []Decision{spwSwitch(2, 3, CauseDistress)}; !slices.Equal(got, want) {
		t.Errorf("Decide plan = %+v, want %+v", got, want)
	}
}
