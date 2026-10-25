package economysim

import (
	"math"
	"slices"
	"strings"
	"testing"
)

// TestGroupWeights verifies the pure derivation of the agents and weights of a
// group: ceil(H / S) agents, raised to min(K, H), weights that sum to H and
// differ by at most 1, the first H mod a agents taking the extra household
// (spec-scale.md 3.2). The expected weights are worked out in each row.
//
// Requirement: FR-SCL-001, FR-SCL-002, FR-SCL-003, FR-SCL-004, FR-SCL-005
func TestGroupWeights(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		households, scale int64
		minAgents         int64
		want              []int64
	}{
		// H 1, S 1: one agent of weight 1 (the floor min(K, H) is 1).
		{"one household", 1, 1, 10, []int64{1}},
		// H 10, S 4: ceil(10 / 4) = 3 agents; 10 = 3 * 3 + 1, so the first agent takes
		// the extra household: 4, 3, 3.
		{"ten over scale four", 10, 4, 1, []int64{4, 3, 3}},
		// H 2 below K 3: ceil(2 / 100) = 1, raised to min(3, 2) = 2 agents of 1.
		{"fewer households than the floor", 2, 100, 3, []int64{1, 1}},
		// H 40, S 40, K 1: one agent holding all forty households.
		{"one agent holds the group", 40, 40, 1, []int64{40}},
		// H 7, S 100, K 3: ceil(7 / 100) = 1, raised to min(3, 7) = 3 agents; 7 = 3 * 2 + 1:
		// 3, 2, 2.
		{"floor raises the count", 7, 100, 3, []int64{3, 2, 2}},
		// H 100, S 1, K 10: ceil(100 / 1) = 100 agents of weight 1 (the floor is below).
		{"scale one gives one household each", 100, 1, 10, slices.Repeat([]int64{1}, 100)},
		// H 10, S 3, K 1: ceil(10 / 3) = 4 agents; 10 = 4 * 2 + 2: 3, 3, 2, 2.
		{"two extras", 10, 3, 1, []int64{3, 3, 2, 2}},
		// No household or no scale is a group validation rejects: no agents.
		{"no household", 0, 1, 1, nil},
		{"no scale", 5, 0, 1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := groupWeights(tt.households, tt.scale, tt.minAgents)
			if !slices.Equal(got, tt.want) {
				t.Errorf("groupWeights(%d, %d, %d) = %v, want %v", tt.households, tt.scale, tt.minAgents, got, tt.want)
			}
			if tt.want == nil {
				return
			}
			var sum int64
			for _, w := range got {
				sum += w
			}
			if lo, hi := slices.Min(got), slices.Max(got); sum != tt.households || hi-lo > 1 || lo < 1 {
				t.Errorf("groupWeights(%d, %d, %d) = %v: sum %d, spread %d, min %d, want sum %d, spread at most 1, min at least 1",
					tt.households, tt.scale, tt.minAgents, got, sum, hi-lo, lo, tt.households)
			}
		})
	}
}

// TestAgentCount verifies the one formula of the number of agents of a group
// (spec-scale.md 3.2): ceil(H / S), raised to min(K, H), and 0 for a group
// validation rejects. The expected counts are worked out in each row.
//
// Requirement: FR-SCL-002, FR-SCL-020
func TestAgentCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                         string
		households, scale, minAgents int64
		want                         int64
	}{
		{"no household", 0, 1, 1, 0},
		{"no scale", 5, 0, 1, 0},
		{"negative scale", 5, -1, 1, 0},
		// H 10, S 5: 10 / 5 = 2 exactly, no remainder, so no extra agent.
		{"exact division", 10, 5, 1, 2},
		// H 11, S 5: 11 / 5 = 2 remainder 1, so ceil = 3.
		{"remainder adds one", 11, 5, 1, 3},
		// H 100, S 50, K 10: ceil = 2, below the floor, so min(10, 100) = 10.
		{"floor wins", 100, 50, 10, 10},
		// H 4, S 100, K 10: the floor is above the households, so min(10, 4) = 4
		// agents of weight 1 (legal and silent, FR-SCL-020).
		{"floor above households", 4, 100, 10, 4},
		{"one household", 1, 1, 10, 1},
		// H MaxHouseholds at S MaxHouseholds: ceil = 1, raised to the floor 10.
		{"MaxHouseholds at the largest scale", MaxHouseholds, MaxHouseholds, 10, 10},
		{"MaxHouseholds at scale 1 and 3", MaxHouseholds, 3, 1, 333_333_334},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := agentCount(tt.households, tt.scale, tt.minAgents); got != tt.want {
				t.Errorf("agentCount(%d, %d, %d) = %d, want %d", tt.households, tt.scale, tt.minAgents, got, tt.want)
			}
		})
	}
}

// TestGroupShape_AgreesWithGroupWeights verifies that the shape validation reads
// (agents and largest weight) is the length and the maximum of the weights the
// construction builds, over a grid of households, scales and floors.
//
// Requirement: FR-SCL-002, FR-SCL-003
// Guard: the formula was written twice (construct.go and validate.go); two copies
// that drift apart would let validation bound a weight that construction does not
// build.
func TestGroupShape_AgreesWithGroupWeights(t *testing.T) {
	t.Parallel()
	for h := int64(1); h <= 400; h += 3 {
		for s := int64(1); s <= 50; s++ {
			for k := int64(1); k <= 20; k++ {
				agents, maxWeight, scale, ok := groupShape(WorldParams{MinAgents: k}, Group{Households: h, Scale: s})
				w := groupWeights(h, s, k)
				if !ok || scale != s || agents != int64(len(w)) || maxWeight != slices.Max(w) {
					t.Fatalf("groupShape(K %d, H %d, S %d) = %d agents, max %d, scale %d, ok %t; groupWeights has %d agents, max %d",
						k, h, s, agents, maxWeight, scale, ok, len(w), slices.Max(w))
				}
			}
		}
	}
}

// TestGroupShape_Bounds verifies the edges of groupShape: the largest group at
// the largest scale keeps the agent count at the floor, the world's scale applies
// when the group gives none, and a household count, scale or floor outside its
// bound has no shape.
//
// Requirement: FR-SCL-002, FR-SCL-011
func TestGroupShape_Bounds(t *testing.T) {
	t.Parallel()
	agents, maxWeight, _, ok := groupShape(WorldParams{MinAgents: 10}, Group{Households: MaxHouseholds, Scale: MaxHouseholds})
	if !ok || agents != 10 || maxWeight != MaxHouseholds/10 {
		t.Errorf("groupShape(MaxHouseholds at MaxHouseholds, K 10) = %d agents, max %d, ok %t, want 10 agents of %d", agents, maxWeight, ok, MaxHouseholds/10)
	}
	if _, _, scale, ok := groupShape(WorldParams{Scale: 4, MinAgents: 1}, Group{Households: 40}); !ok || scale != 4 {
		t.Errorf("groupShape(world scale 4, group scale 0) scale = %d, ok %t, want 4", scale, ok)
	}
	for _, g := range []Group{{Households: 0, Scale: 1}, {Households: MaxHouseholds + 1, Scale: 1}, {Households: 5, Scale: -1}, {Households: 5, Scale: MaxHouseholds + 1}} {
		if _, _, _, ok := groupShape(WorldParams{Scale: 1, MinAgents: 1}, g); ok {
			t.Errorf("groupShape(%+v) ok = true, want false", g)
		}
	}
	if _, _, _, ok := groupShape(WorldParams{Scale: 1, MinAgents: 0}, Group{Households: 5}); ok {
		t.Errorf("groupShape(floor 0) ok = true, want false")
	}
}

// TestGroupWeights_RejectsWhatValidationRejects verifies that a group with no
// household or with a scale below 1, which validation rejects, gets no weights.
//
// Requirement: FR-SCL-001
func TestGroupWeights_RejectsWhatValidationRejects(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ households, scale int64 }{{0, 1}, {-3, 1}, {5, 0}, {5, -2}, {0, 0}} {
		if got := groupWeights(tt.households, tt.scale, 10); got != nil {
			t.Errorf("groupWeights(%d, %d, 10) = %v, want nil", tt.households, tt.scale, got)
		}
	}
}

// TestGroupWeights_DrawsNothing verifies that the weights of the groups of a
// configuration do not depend on the seed: worlds of seeds 1 and 2 plan the same
// weights, and the plan is built before the creation draws.
//
// Requirement: FR-SCL-004
func TestGroupWeights_DrawsNothing(t *testing.T) {
	t.Parallel()
	c := removalConfig()
	a, b := newMill(t, c, WithSeed(1)), newMill(t, c, WithSeed(2))
	pa, pb := groupPlans(&a.cfg), groupPlans(&b.cfg)
	if len(pa) == 0 || len(pa) != len(pb) {
		t.Fatalf("groupPlans of seeds 1 and 2 = %d and %d plans, want equal and non-zero", len(pa), len(pb))
	}
	for i := range pa {
		if !slices.Equal(pa[i].weights, pb[i].weights) {
			t.Errorf("group %d weights = %v at seed 1 and %v at seed 2, want equal", i, pa[i].weights, pb[i].weights)
		}
	}
}

// FuzzGroupWeights verifies the properties of the derivation over arbitrary
// inputs inside the bounds: the weights sum to the households, differ by at most
// 1, are each at least 1, their count is agentCount, and nothing panics. The
// inputs are mapped into the bounds, and a draw that would allocate more than
// 100 000 agents is skipped so the fuzzer cannot allocate the 10^9 agents of a
// group at scale 1 (plan 2-4 Risks, item 1).
//
// Requirement: FR-SCL-001, FR-SCL-002, FR-SCL-003
func FuzzGroupWeights(f *testing.F) {
	f.Add(uint64(2000), uint64(40), uint64(10))
	f.Add(uint64(105), uint64(40), uint64(10))
	f.Add(uint64(1), uint64(1), uint64(1))
	f.Add(uint64(math.MaxUint64), uint64(math.MaxUint64), uint64(math.MaxUint64))
	f.Fuzz(func(t *testing.T, rh, rs, rk uint64) {
		h, s, k := int64(1+rh%MaxHouseholds), int64(1+rs%MaxHouseholds), int64(1+rk%MaxHouseholds)
		n := agentCount(h, s, k)
		if n > 100_000 {
			return // a bounded input space, not a skipped case
		}
		w := groupWeights(h, s, k)
		var sum int64
		for _, x := range w {
			sum += x
		}
		if int64(len(w)) != n || sum != h || slices.Max(w)-slices.Min(w) > 1 || slices.Min(w) < 1 {
			t.Fatalf("groupWeights(%d, %d, %d): %d weights, sum %d, spread %d, min %d, want %d weights summing to %d, spread at most 1, min at least 1",
				h, s, k, len(w), sum, slices.Max(w)-slices.Min(w), slices.Min(w), n, h)
		}
	})
}

// TestCheckCensus_PanicsOnABrokenPlan verifies the census invariant: a plan whose
// weights are a household short, spread by 2, hold a weight of 0, or sit under
// the floor of a group of weight above 1, and totals that differ from the live
// counters by one, each panic with a violated-invariant message, and a good
// plan returns.
//
// Arithmetic: group A is 10 households (4, 3, 3: 3 agents, spread 1); group B is
// 4 households at floor 3 (2, 1, 1: 3 agents). The totals are 6 agents and 14
// households.
//
// Requirement: FR-SCL-003, FR-SCL-005, FR-SCL-019
// Guard: the construction accepts a census that does not add up, and every later
// count (the snapshot, the metrics, the spawner) inherits it.
func TestCheckCensus_PanicsOnABrokenPlan(t *testing.T) {
	t.Parallel()
	cfg := &Config{World: WorldParams{MinAgents: 3}, Groups: []Group{{Households: 10}, {Households: 4}}}
	good := func() []groupPlan {
		return []groupPlan{{weights: []int64{4, 3, 3}}, {weights: []int64{2, 1, 1}}}
	}
	tests := []struct {
		name               string
		edit               func([]groupPlan)
		agents, households int64
		want               string // "" for a plan that passes
	}{
		{"a good plan", func([]groupPlan) {}, 6, 14, ""},
		{"a sum one short", func(p []groupPlan) { p[0].weights = []int64{4, 3, 2} }, 6, 14, "do not sum"},
		{"a sum one over", func(p []groupPlan) { p[1].weights = []int64{2, 2, 1} }, 6, 14, "do not sum"},
		{"a spread of 2", func(p []groupPlan) { p[0].weights = []int64{5, 3, 2} }, 6, 14, "differ by more than 1"},
		{"a weight of 0", func(p []groupPlan) { p[0].weights = []int64{5, 5, 0} }, 6, 14, "weight below 1"},
		// Group B, 4 households, floor 3: two agents of weight 2 are under min(3, 4).
		{"a group of weight above 1 under its floor", func(p []groupPlan) { p[1].weights = []int64{2, 2} }, 5, 14, "fewer agents than its floor"},
		{"agents off by one", func([]groupPlan) {}, 7, 14, "differ from the census"},
		{"households off by one", func([]groupPlan) {}, 6, 13, "differ from the census"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plans := good()
			tt.edit(plans)
			msg := invariantMessage(func() { checkCensus(plans, cfg, tt.agents, tt.households) })
			if tt.want == "" {
				if msg != "" {
					t.Errorf("checkCensus(good plan) panicked with %q, want no panic", msg)
				}
				return
			}
			if !strings.Contains(msg, "invariant") || !strings.Contains(msg, tt.want) {
				t.Errorf("checkCensus(%s) panicked with %q, want an invariant panic containing %q", tt.name, msg, tt.want)
			}
		})
	}
}

// TestScale_BankruptcyCarriesTheWeight verifies that removing a cohort emits
// AgentBankrupted with the weight of the agent, and that the household count
// falls by that weight.
//
// Arithmetic: the miller group is 7 households at scale 7, so agent 1 is one
// agent of weight 7 holding 7 * 7 000 = 49 000 money.
//
// Requirement: FR-SCL-018
func TestScale_BankruptcyCarriesTheWeight(t *testing.T) {
	t.Parallel()
	c := execConfig()
	c.Groups[0].Households, c.Groups[0].Scale = 7, 7
	d := Decision{Kind: DecideRemove, Agent: 1, Cause: CauseStarved}
	w, rec, _ := execWorld(t, c, 25_000, &scriptSpawner{plan: regionPlans([]Decision{d})})
	if w.agents[1].weight != 7 {
		t.Fatalf("seed %d: agent 1 weight = %d, want 7", execSeed, w.agents[1].weight)
	}
	agents0, hh0 := w.live.agents, w.live.households
	runEntry(t, w, rec)
	var got []AgentBankrupted
	for _, e := range rec.events {
		if b, ok := e.(AgentBankrupted); ok {
			got = append(got, b)
		}
	}
	if len(got) != 1 || got[0].Weight != 7 || got[0].Money != 49_000 {
		t.Errorf("seed %d: AgentBankrupted events = %+v, want one of weight 7 and money 49 000", execSeed, got)
	}
	if w.live.agents != agents0-1 || w.live.households != hh0-7 {
		t.Errorf("seed %d: live agents %d and households %d, want %d and %d", execSeed, w.live.agents, w.live.households, agents0-1, hh0-7)
	}
}

// TestScale_FloorAboveHouseholdsIsSilent verifies that a group with fewer
// households than the floor builds one agent of weight 1 per household, validates,
// and raises neither a configuration warning nor the run-time warning "group
// below floor" (which is for a group of weight above 1).
//
// Arithmetic: 4 households at MinAgents 10 is min(10, 4) = 4 agents of weight 1.
//
// Requirement: FR-SCL-020, FR-SCL-005
func TestScale_FloorAboveHouseholdsIsSilent(t *testing.T) {
	t.Parallel()
	c := execConfig()
	c.World.MinAgents = 10
	c.Groups[0].Households, c.Groups[0].Scale = 4, 0
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(4 households at floor 10) error = %v, want nil", err)
	}
	for _, w := range c.Warnings() {
		if strings.Contains(w.Rule, "group_below_floor") {
			t.Errorf("Warnings() holds %+v, want no group_below_floor", w)
		}
	}
	w, _, lg := execWorld(t, c, 0, &scriptSpawner{})
	if got := groupWeights(4, 1, 10); !slices.Equal(got, []int64{1, 1, 1, 1}) {
		t.Errorf("groupWeights(4, 1, 10) = %v, want four agents of weight 1", got)
	}
	if w.live.agents != 5 || w.live.households != 5 {
		t.Errorf("live agents %d, households %d, want 5 and 5 (4 millers and the weaver)", w.live.agents, w.live.households)
	}
	// Only construction is asserted: at the first Step the run-time Warn "group
	// below floor" does fire for this group, because the entry step reads the
	// households of the group (4) as its weight (phase_exit.go entryWarns); the
	// spec says it is silent (spec-scale.md 5.2). Reported in the plan 2-4 report.
	for _, l := range *lg.lines {
		if l.msg == "group below floor" {
			t.Errorf("a %q line was logged at construction for a group of 4 households at floor 10, want none", l.msg)
		}
	}
}

// TestScale_DefaultsAndGroupOverride verifies that the default world scale is 1
// and that a group's own scale replaces the world's: with the world at 5, a group
// that gives none has ceil(20 / 5) = 4 agents and one that gives scale 2 has
// ceil(20 / 2) = 10.
//
// Requirement: FR-SCL-006
func TestScale_DefaultsAndGroupOverride(t *testing.T) {
	t.Parallel()
	if got := DefaultWorldParams().Scale; got != 1 {
		t.Errorf("DefaultWorldParams().Scale = %d, want 1", got)
	}
	c := execConfig()
	c.World.Scale, c.World.MinAgents = 5, 1
	c.Groups[0].Households, c.Groups[0].Scale = 20, 0
	c.Groups[1].Households, c.Groups[1].Scale = 20, 2
	c.Roles[1].TrainingTicks = 0 // the traineeship rule needs weight 1 for a scaled group
	plans := groupPlans(&c)
	if len(plans[0].weights) != 4 || len(plans[1].weights) != 10 {
		t.Errorf("groupPlans agents = %d and %d, want 4 (world scale 5) and 10 (group scale 2)", len(plans[0].weights), len(plans[1].weights))
	}
}
