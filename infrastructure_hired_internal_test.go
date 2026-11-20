package economysim

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// hiredBuildScript is a Strategy for the hired build: the miller posts one position of the
// recipe zz_build at a bid of 100 and the weaver offers its household at a reserve of 10.
type hiredBuildScript struct {
	recipe RecipeID
	miller RoleID
}

func (s *hiredBuildScript) Work(v *AgentView, _ *rand.Rand) JobRequest {
	if v.Role == s.miller {
		return JobRequest{NoJob: true, Hire: HireRequest{Recipe: s.recipe, Positions: 1, Bid: 100}}
	}
	return JobRequest{NoJob: true, Offer: LabourOffer{Households: 1, Reserve: 10}}
}
func (*hiredBuildScript) Trade(*AgentView, *rand.Rand, *TradePlan)   {}
func (*hiredBuildScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = (*hiredBuildScript)(nil)

// TestComplete_HiredBuildIsThePrincipals verifies 4.3.1 with the hire form of EMP: a household
// of the Hireable role that runs the build recipe for an employer makes the facility for the
// employer, not for itself. The miller posts one position of zz_build (two ticks), the weaver is
// hired at P2.5a, the job completes at P2.1 of the second tick after its start, and the one mill
// built is owned by the miller, in the miller's region, while the weaver owns no facility.
//
// Requirement: FR-CAP-014
func TestComplete_HiredBuildIsThePrincipals(t *testing.T) {
	t.Parallel()
	script, rec := &hiredBuildScript{}, &evLog{}
	w := newMill(t, bldConfig(func(c *Config) {
		c.Modules.Employment = &Employment{Hire: true, MaxOpen: 100}
		miller := &c.Roles[0]
		miller.Employer = &Employer{MaxHires: 4, WorkerShareBP: 5_000, HireYieldBP: 10_000, Recipes: []string{"zz_build"}}
		for i := range c.Groups {
			if c.Groups[i].Role == "miller" {
				c.Groups[i].Households, c.Groups[i].Scale = 1, 1 // an employer has a weight of 1
			}
		}
		for i := range c.Roles {
			if c.Roles[i].ID == "weaver" {
				c.Roles[i].Hireable, c.Roles[i].Strategy = true, "script"
			}
		}
	}), WithSeed(facSeed), WithAudit(), WithObserver(rec), WithStrategy("script", script))
	script.recipe, script.miller = bldRecipe(t, w, "zz_build"), 0
	m, weaver := bldMiller(w, 0), facAgents(w, 0, "weaver")[0]
	for range 3 {
		facStep(t, w)
	}
	formed := of[HireFormed](rec)
	if len(formed) == 0 || formed[0].Employer != m || formed[0].Worker != weaver {
		t.Fatalf("HireFormed = %+v, want a first hire of employer %d and worker %d (seed %d)", formed, m, weaver, facSeed)
	}
	// Yonder's miller hires in the same way, so the home region's events are the ones read.
	built := slices.DeleteFunc(of[FacilityChanged](rec), func(e FacilityChanged) bool { return e.Kind != FacilityBuilt || e.Region != 0 })
	if len(built) != 1 {
		t.Fatalf("FacilityChanged built events of the home region = %d after three ticks of a hired two-tick build (seed %d), want 1 (one household hired)", len(built), facSeed)
	}
	if e := built[0]; e.Owner != m || e.Class != clsMill {
		t.Errorf("built[0] = %+v, want owner %d (the employer, not the worker %d) and class %d", e, m, weaver, clsMill)
	}
	if got := w.infra.ownedBy(weaver); len(got) != 0 {
		t.Errorf("facilities of the hired worker = %v, want none", got)
	}
}
