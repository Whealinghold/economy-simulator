package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// policyMoney returns policyConfig with three money flows, so a flow lever can name
// one: "aid" pays the cooks of north, "dole" the cooks of every region and "farm_aid" the
// cooks of farm.
func policyMoney() economysim.Config {
	c := policyConfig()
	c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{
		{ID: "aid", Kind: economysim.FlowGrant, Amount: 1, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Region: "north", Role: "cook"}},
		{ID: "dole", Kind: economysim.FlowGrant, Amount: 1, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "cook"}},
		{ID: "farm_aid", Kind: economysim.FlowGrant, Amount: 1, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Region: "farm", Role: "cook"}},
	}}
	return c
}

// polLevers sets the levers of an actor of a configuration to one lever.
func polLever(c *economysim.Config, actor int, l economysim.Lever) {
	c.Modules.Policy.Actors[actor].Levers = []economysim.Lever{l}
	c.Modules.Policy.Actors[actor].Schedule, c.Modules.Policy.Actors[actor].Rules = nil, nil
	if c.Modules.Policy.Actors[actor].Policy != "directed" {
		c.Modules.Policy.Actors[actor].Policy = "hold"
	}
}

// TestValidate_Policy_ModuleDependency verifies FR-PLC-002: the module needs the polity
// module with a jurisdiction, a flow lever needs the money supply or the fiscal module
// and a transit lever the transport module.
//
// Requirement: FR-PLC-002
func TestValidate_Policy_ModuleDependency(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		c    func() economysim.Config
		path []string
	}{
		{"no polity module", func() economysim.Config {
			c := policyConfig()
			c.Modules.Polity, c.Modules.Fiscal = nil, nil
			return c
		}, []string{"modules", "policy"}},
		{"a polity with no jurisdiction", func() economysim.Config {
			c := policyConfig()
			c.Modules.Polity = &economysim.PolityConfig{}
			c.Modules.Fiscal = nil
			return c
		}, []string{"modules", "policy"}},
		{"a flow lever with neither the money supply nor the fiscal module", func() economysim.Config {
			c := policyConfig()
			c.Modules.Fiscal = nil
			return c
		}, polPath(0, "levers[0]", "kind")},
		{"a transit lever with transport off", func() economysim.Config {
			c := policyConfig()
			c.Modules.Transport = nil
			return c
		}, polPath(0, "levers[1]", "kind")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.c()
			vs := policyViolations(t, c, economysim.ErrModuleDependency)
			if len(vs) == 0 || !slices.Equal(vs[0].Path, tt.path) {
				t.Errorf("Validate paths = %v, want ErrModuleDependency at %v", vs, tt.path)
			}
		})
	}
}

// policyViolations validates c, requires the rule among the errors and returns the
// violations that are of the rule.
func policyViolations(t *testing.T, c economysim.Config, rule error) []*economysim.ValidationError {
	t.Helper()
	err := c.Validate()
	if !errors.Is(err, rule) {
		t.Fatalf("Validate() = %v, want %v", err, rule)
	}
	vs, _ := violations(err)
	var out []*economysim.ValidationError
	for _, v := range vs {
		if errors.Is(v, rule) {
			out = append(out, v)
		}
	}
	return out
}

// TestValidate_Policy_NotInReach verifies FR-PLC-007: one failing case for each row of
// the table of spec-policy.md 3.5.1, with the path of the lever, the key as the value and
// the jurisdiction as the bound.
//
// Requirement: FR-PLC-007
func TestValidate_Policy_NotInReach(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		actor int // the actor whose lever is replaced
		lever economysim.Lever
		want  string
	}{
		{"a fiscal flow of another authority", 2, economysim.Lever{ID: "x", Kind: economysim.LeverFlow, Flow: "poll", Min: 0, Max: 20_000}, "poll"},
		{"a money flow with a target region outside", 1, economysim.Lever{ID: "x", Kind: economysim.LeverFlow, Flow: "aid", Min: 0, Max: 20_000}, "aid"},
		{"a money flow with no region and a jurisdiction that covers fewer", 2, economysim.Lever{ID: "x", Kind: economysim.LeverFlow, Flow: "dole", Min: 0, Max: 20_000}, "dole"},
		{"a fiscal flow on the treasury of a jurisdiction above", 2, economysim.Lever{ID: "x", Kind: economysim.LeverFlow, Flow: "remit", Min: 0, Max: 20_000}, "remit"},
		{"an edge neither of whose ends is covered", 1, economysim.Lever{ID: "x", Kind: economysim.LeverEdge, From: "capital", To: "mill", Min: 10_000, Max: 20_000}, "capital>mill"},
		{"a transit neither of whose ends is covered", 1, economysim.Lever{ID: "x", Kind: economysim.LeverTransit, From: "capital", To: "mill", Mode: "foot", Min: 0, Max: 20_000}, "capital>mill:foot"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := policyMoney()
			polLever(&c, tt.actor, tt.lever)
			vs := policyViolations(t, c, economysim.ErrLeverNotInReach)
			if len(vs) != 1 || vs[0].Value != tt.want || !slices.Equal(vs[0].Path, polPath(tt.actor, "levers[0]", "kind")) {
				t.Errorf("Validate violations = %v, want one at actor %d with value %q", vs, tt.actor, tt.want)
			}
		})
	}
}

// TestValidate_Policy_InReach verifies the other side of 3.5.1: a parent may set a
// child's rate, a jurisdiction that covers the whole target reaches a money flow, a
// treasury flow of a descendant is in reach of its ancestor, and an edge or a transit with
// one end covered is in reach.
//
// Requirement: FR-PLC-007
func TestValidate_Policy_InReach(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		actor int
		lever economysim.Lever
	}{
		{"a parent sets a child's fiscal flow", 0, economysim.Lever{ID: "x", Kind: economysim.LeverFlow, Flow: "stipend", Min: 0, Max: 20_000}},
		{"the crown reaches a money flow on every region", 0, economysim.Lever{ID: "x", Kind: economysim.LeverFlow, Flow: "dole", Min: 0, Max: 20_000}},
		{"the crown reaches a fiscal flow on a treasury", 0, economysim.Lever{ID: "x", Kind: economysim.LeverFlow, Flow: "remit", Min: 0, Max: 20_000}},
		{"the inland province reaches a money flow on one of its regions", 2, economysim.Lever{ID: "x", Kind: economysim.LeverFlow, Flow: "farm_aid", Min: 0, Max: 20_000}},
		{"an edge with one end covered", 1, economysim.Lever{ID: "x", Kind: economysim.LeverEdge, From: "harbour", To: "mill", Min: 10_000, Max: 20_000}},
		{"a transit with one end covered", 1, economysim.Lever{ID: "x", Kind: economysim.LeverTransit, From: "harbour", To: "mill", Mode: "foot", Min: 0, Max: 20_000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := policyMoney()
			polLever(&c, tt.actor, tt.lever)
			if tt.actor == 0 && tt.lever.Flow == "stipend" {
				// the inland actor leaves the stipend to the crown
				polLever(&c, 2, economysim.Lever{ID: "wall", Kind: economysim.LeverEdge, From: "capital", To: "mill", Min: 10_000, Max: 20_000})
			}
			if err := c.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

// TestValidate_Policy_LeverSharedAfterExpansion verifies FR-PLC-009 and the key rule of
// 3.5.2: a key named by two levers of one actor is ErrDuplicateID, a key held by levers of
// two actors is ErrLeverShared with the first holder as the bound, and an Against group
// shares its expanded edges (the edges 1 and 3 of the fixture), so a pair lever of
// another actor on one of them is shared.
//
// Requirement: FR-PLC-004, FR-PLC-008, FR-PLC-009
func TestValidate_Policy_LeverSharedAfterExpansion(t *testing.T) {
	t.Parallel()
	pair := economysim.Lever{ID: "twin", Kind: economysim.LeverEdge, From: "harbour", To: "mill", Min: 10_000, Max: 20_000}
	t.Run("two levers of one actor on one flow", func(t *testing.T) {
		t.Parallel()
		c := policyConfig()
		l := c.Modules.Policy.Actors[0].Levers[0]
		l.ID = "poll2"
		c.Modules.Policy.Actors[0].Levers = append(c.Modules.Policy.Actors[0].Levers, l)
		vs := policyViolations(t, c, economysim.ErrDuplicateID)
		if len(vs) != 1 || vs[0].Value != "flow:poll" || !slices.Equal(vs[0].Path, polPath(0, "levers[2]", "id")) {
			t.Errorf("violations = %v, want one at levers[2] for flow:poll", vs)
		}
	})
	t.Run("an Against group and a pair lever of the same actor", func(t *testing.T) {
		t.Parallel()
		c := policyConfig()
		c.Modules.Policy.Actors[1].Levers = append(c.Modules.Policy.Actors[1].Levers, pair)
		if vs := policyViolations(t, c, economysim.ErrDuplicateID); len(vs) != 1 || vs[0].Value != "edge:1" {
			t.Errorf("violations = %v, want one for edge:1", vs)
		}
	})
	t.Run("two actors on one flow", func(t *testing.T) {
		t.Parallel()
		c := policyConfig()
		polLever(&c, 0, economysim.Lever{ID: "x", Kind: economysim.LeverFlow, Flow: "stipend", Min: 0, Max: 20_000})
		vs := policyViolations(t, c, economysim.ErrLeverShared)
		if len(vs) != 1 || vs[0].Value != "flow:stipend" || vs[0].Bound != "crown_council" {
			t.Errorf("violations = %v, want flow:stipend shared with crown_council", vs)
		}
	})
	t.Run("an Against group and another actor's pair on an expanded edge", func(t *testing.T) {
		t.Parallel()
		c := policyConfig()
		c.Modules.Policy.Actors[2].Levers = append(c.Modules.Policy.Actors[2].Levers, economysim.Lever{ID: "twin", Kind: economysim.LeverEdge, From: "mill", To: "harbour", Min: 10_000, Max: 20_000})
		vs := policyViolations(t, c, economysim.ErrLeverShared)
		if len(vs) != 1 || vs[0].Value != "edge:3" || vs[0].Bound != "coast_council" {
			t.Errorf("violations = %v, want edge:3 shared with coast_council", vs)
		}
	})
}

// TestAgainst_KeysAreBothBetweens verifies spec-policy.md 3.4.3 on the fixture: the
// group of coast against inland is Between(coast, inland) and Between(inland, coast), the
// edges harbour>mill and mill>harbour (the edges 1 and 3 in the order of the sorted
// regions), the analogue of edges 4 and 8 of the worked example 14.4, and no edge between
// nested jurisdictions.
//
// Requirement: FR-PLC-008
func TestAgainst_KeysAreBothBetweens(t *testing.T) {
	t.Parallel()
	c := policyConfig()
	g, err := economysim.NewGraph(c)
	if err != nil {
		t.Fatalf("NewGraph() = %v", err)
	}
	pol, err := economysim.NewPolity(c)
	if err != nil {
		t.Fatalf("NewPolity() = %v", err)
	}
	coast, _ := pol.Lookup("coast")
	inland, _ := pol.Lookup("inland")
	crown, _ := pol.Lookup("crown")
	if got := pol.Between(g, coast, inland); !slices.Equal(got, []economysim.EdgeID{1}) {
		t.Errorf("Between(coast, inland) = %v, want [1] (harbour>mill)", got)
	}
	if got := pol.Between(g, inland, coast); !slices.Equal(got, []economysim.EdgeID{3}) {
		t.Errorf("Between(inland, coast) = %v, want [3] (mill>harbour)", got)
	}
	if got := pol.Between(g, crown, inland); len(got) != 0 {
		t.Errorf("Between(crown, inland) = %v, want none: the regions are nested", got)
	}
}

// TestPolicy_KindsAreExactlyThree verifies FR-PLC-046 and the reserved kinds of 6.3.2:
// flow, edge and transit are offered, and every other kind, the three reserved ones and
// the shocks among them, is ErrOutOfBounds at .kind.
//
// Requirement: FR-PLC-046, FR-PLC-006
func TestPolicy_KindsAreExactlyThree(t *testing.T) {
	t.Parallel()
	for _, kind := range []economysim.LeverKind{"rate", "rent", "legal", "yield", "route_risk", "", "Flow"} {
		t.Run("refuses "+string(kind), func(t *testing.T) {
			t.Parallel()
			c := policyConfig()
			c.Modules.Policy.Actors[0].Levers[0].Kind = kind
			vs := policyViolations(t, c, economysim.ErrOutOfBounds)
			if len(vs) != 1 || !slices.Equal(vs[0].Path, polPath(0, "levers[0]", "kind")) {
				t.Errorf("violations = %v, want one at the lever's kind", vs)
			}
		})
	}
	if got := []economysim.LeverKind{economysim.LeverFlow, economysim.LeverEdge, economysim.LeverTransit}; !slices.Equal(got, []economysim.LeverKind{"flow", "edge", "transit"}) {
		t.Errorf("the kinds are %v, want flow, edge, transit", got)
	}
}

// TestValidate_Policy_FirstErrorOrder verifies that the rules run in the order of 11.1 and
// a phase that finds an error ends the run: a configuration that breaks the count rule and
// a later one reports only the count.
//
// Requirement: FR-PLC-013
func TestValidate_Policy_FirstErrorOrder(t *testing.T) {
	t.Parallel()
	c := policyConfig()
	c.Modules.Policy.Actors[0].Every = 0                                                                  // rule 6, bounds
	c.Modules.Policy.Actors[1].Jurisdiction = "nowhere"                                                   // rule 4, jurisdictions
	c.Modules.Policy.Actors[2].Levers[0].Flow, c.Modules.Policy.Actors[2].Levers[0].Mode = "poll", "foot" // rule 8
	err := c.Validate()
	vs, _ := violations(err)
	var got []error
	for _, v := range vs {
		if slices.Contains(v.Path, "policy") {
			got = append(got, v.Rule)
		}
	}
	if len(got) != 1 || !errors.Is(got[0], economysim.ErrUnknownJurisdiction) {
		t.Errorf("policy errors = %v, want only ErrUnknownJurisdiction (the first rule that fails)", got)
	}
}

// TestSentinels_Policy verifies that the six sentinels of the module are distinct and
// non-nil, match themselves through errors.Is and are none of the sentinels they sit
// beside.
//
// Requirement: FR-PLC-006
func TestSentinels_Policy(t *testing.T) {
	t.Parallel()
	all := []error{
		economysim.ErrPolicyShape, economysim.ErrLeverNotInReach, economysim.ErrLeverShared,
		economysim.ErrUnknownPolicy, economysim.ErrUnknownActor, economysim.ErrUnknownLever,
	}
	for i, a := range all {
		if a == nil || a.Error() == "" {
			t.Fatalf("sentinel %d is nil or empty", i)
		}
		for j, b := range all {
			if (i == j) != errors.Is(a, b) {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", a, b, errors.Is(a, b), i == j)
			}
		}
		if errors.Is(a, economysim.ErrOutOfBounds) || errors.Is(a, economysim.ErrNotDirected) {
			t.Errorf("%v matches a sentinel it is not", a)
		}
	}
}
