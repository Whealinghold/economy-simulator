package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestValidate_Policy_ScheduleCycleWrap verifies the wrap of rule schedule_step: with a
// cycle the first step is compared with the last value, so a schedule whose last value
// is further than MaxStep from its first is refused at the first step, and the same
// steps without a cycle are accepted.
//
// Requirement: FR-PLC-011
func TestValidate_Policy_ScheduleCycleWrap(t *testing.T) {
	t.Parallel()
	// The lever "poll" starts at its default 10 000 and has MaxStep 5 000. Each step is
	// within 5 000 of the one before: 12 000 (+2 000), 16 000 (+4 000), 20 000 (+4 000).
	// With a cycle the first step follows the last: 20 000 to 12 000 is 8 000 > 5 000.
	steps := []economysim.ScheduleStep{
		{Lever: "poll", At: 0, BP: 12_000}, {Lever: "poll", At: 50, BP: 16_000}, {Lever: "poll", At: 100, BP: 20_000},
	}
	tests := []struct {
		name  string
		cycle int64
		want  bool // rule schedule_step is raised
	}{
		{"a cycle whose wrap is too long", 200, true},
		{"the same steps without a cycle", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := policyConfig()
			a := &c.Modules.Policy.Actors[0]
			a.Cycle, a.Schedule = tt.cycle, steps
			err := c.Validate()
			if !tt.want {
				if err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
				return
			}
			vs := policyViolations(t, c, economysim.ErrPolicyShape)
			if len(vs) != 1 || vs[0].Bound != "schedule_step" || !slices.Equal(vs[0].Path, polPath(0, "schedule[0]", "bp")) {
				t.Errorf("Validate violations = %v, want one schedule_step at schedule[0].bp", vs)
			}
		})
	}
}

// TestValidate_Policy_UnsoundBaseSkipsRules verifies that a configuration whose edges are
// not sound reports the edge error and no policy error, and does not panic: the rules that
// need the polity and the graph do not run on a base that is not built.
//
// Guard: the reach and key rules read the built graph; if they ran on a failed build they
// would dereference a nil graph.
func TestValidate_Policy_UnsoundBaseSkipsRules(t *testing.T) {
	t.Parallel()
	c := policyConfig()
	c.Edges = append(c.Edges, economysim.Edge{From: "nowhere", To: "mill", Distance: 40})
	err := c.Validate()
	if !errors.Is(err, economysim.ErrUnknownRegion) {
		t.Fatalf("Validate() = %v, want ErrUnknownRegion", err)
	}
	vs, _ := violations(err)
	for _, v := range vs {
		if len(v.Path) > 1 && v.Path[0] == "modules" && v.Path[1] == "policy" {
			t.Errorf("Validate reported %v on the policy module of an unsound base, want none", v)
		}
	}
}
