package economysim_test

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// spawnerView is a region at tick 100 with two roles and the defaults of the
// spawner tests: review every tick, EntryShare 100 %, a band of 1 000, cooldown
// 5, EntryRatio 8 000, rung 1 at privation 5, rung 3 at 10, full renewal.
//
// Role 1 pays 400 against an upkeep of 1 000: V = 400 * 10 000 / 1 000 = 4 000,
// below the EntryRatio of 8 000, so it is not viable. Role 2 pays 1 300:
// V = 13 000, and E = 1 300 * 250 * 10 000 / (1 000 * 365) = 8 904.
func spawnerView(agents ...economysim.RegionAgent) *economysim.RegionView {
	role := func(id economysim.RoleID, wage economysim.Money, households int64) economysim.RoleView {
		return economysim.RoleView{Role: id, Households: households, Agents: households, Wage: wage, HasWage: true, Upkeep: 1000, ExpectedWorkTicks: 250}
	}
	return &economysim.RegionView{
		Tick: 100, TicksPerYear: 365, ReviewTicks: 1, EntryShare: 10000, RoleHysteresis: 1000,
		RoleCooldown: 5, EntryRatio: 8000, ChangePrivation: 5, RemovePrivation: 10, DeathStep: 1000,
		ExitShare: 10000, Renewal: 10000, EntryPoolShare: 5000,
		Roles:  []economysim.RoleView{role(1, 400, 1), role(2, 1300, 9)},
		Agents: agents,
	}
}

// TestDefaultSpawner_PlansThroughThePublicAPI verifies, through the exported
// surface only, that the default spawner removes a starved weight-1 household
// and renews it into the best viable role, and that it plans nothing for a fed
// household already in the best role.
//
// Requirement: FR-SPN-013, FR-SPN-014, FR-SPN-023, FR-SPN-025, FR-SPN-031
func TestDefaultSpawner_PlansThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	starved := economysim.RegionAgent{Agent: 1, Role: 1, Weight: 1, Money: 5000, Upkeep: 1000, Unfed: 1, Privation: 10}
	fed := economysim.RegionAgent{Agent: 1, Role: 2, Weight: 1, Money: 5000, Upkeep: 1000}
	tests := []struct {
		name string
		view *economysim.RegionView
		want []economysim.Decision
	}{
		// Privation 10 reaches RemovePrivation, the allowance is ceil(1 * 10 000 / 10 000) = 1
		// and money 5 000 >= upkeep 1 000 * weight 1, so the cause is starved. The one
		// household lost is renewed into role 2, the only viable open role (V 13 000 >= 8 000).
		{"starved household is removed and renewed", spawnerView(starved), []economysim.Decision{
			{Kind: economysim.DecideRemove, Agent: 1, Households: 1, Cause: economysim.CauseStarved},
			{Kind: economysim.DecideEnter, Role: 2, Households: 1, Cause: economysim.CauseRenewal},
		}},
		// A fed agent already in the best role (E 8 904, the highest) has no better destination.
		{"fed agent in the best role plans nothing", spawnerView(fed), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var d economysim.Decisions
			economysim.DefaultSpawner{}.Decide(tt.view, rand.New(rand.NewPCG(1, 2)), &d)
			if !reflect.DeepEqual(d.List, tt.want) && (len(d.List) != 0 || len(tt.want) != 0) {
				t.Errorf("Decide() plan = %+v, want %+v", d.List, tt.want)
			}
		})
	}
}

// TestDefaultSpawner_DrawsNothing verifies that the default spawner reads no
// word of the random source it is given, so a nil source and a seeded one give
// the same plan.
//
// Requirement: FR-SPN-003
//
// Guard: the doc comment of DefaultSpawner promises no draw; a draw would move
// every later draw of the world's keyed stream.
func TestDefaultSpawner_DrawsNothing(t *testing.T) {
	t.Parallel()
	agent := economysim.RegionAgent{Agent: 1, Role: 1, Weight: 1, Money: 5000, Upkeep: 1000, Unfed: 1, Privation: 10}
	var a, b economysim.Decisions
	economysim.DefaultSpawner{}.Decide(spawnerView(agent), nil, &a)
	economysim.DefaultSpawner{}.Decide(spawnerView(agent), rand.New(rand.NewPCG(7, 7)), &b)
	if !reflect.DeepEqual(a.List, b.List) {
		t.Errorf("Decide() with nil source = %+v, with a seeded one = %+v, want equal", a.List, b.List)
	}
}

// removeAll is a consumer's spawner: it removes every agent it is shown.
type removeAll struct{}

func (removeAll) Decide(v *economysim.RegionView, _ *rand.Rand, d *economysim.Decisions) {
	for _, a := range v.Agents {
		d.List = append(d.List, economysim.Decision{Kind: economysim.DecideRemove, Agent: a.Agent, Households: a.Weight, Cause: economysim.CauseConsumer})
	}
}

var _ economysim.Spawner = removeAll{}

// TestSpawner_ConsumerImplementation verifies that a consumer can implement
// Spawner with the exported types alone and append to Decisions.List.
//
// Requirement: FR-SPN-001, FR-SPN-003
func TestSpawner_ConsumerImplementation(t *testing.T) {
	t.Parallel()
	var d economysim.Decisions
	var s economysim.Spawner = removeAll{}
	s.Decide(spawnerView(economysim.RegionAgent{Agent: 4, Role: 2, Weight: 3}), nil, &d)
	want := []economysim.Decision{{Kind: economysim.DecideRemove, Agent: 4, Households: 3, Cause: economysim.CauseConsumer}}
	if !reflect.DeepEqual(d.List, want) {
		t.Errorf("Decide() plan = %+v, want %+v", d.List, want)
	}
}

// DefaultSpawner removes a household that has starved and renews it into the
// best viable role. A consumer's own Spawner appends to the same plan.
func ExampleDefaultSpawner() {
	view := spawnerView(economysim.RegionAgent{Agent: 1, Role: 1, Weight: 1, Money: 5000, Upkeep: 1000, Unfed: 1, Privation: 10})
	var plan economysim.Decisions
	economysim.DefaultSpawner{}.Decide(view, nil, &plan)
	for _, d := range plan.List {
		fmt.Println(d.Kind == economysim.DecideRemove, d.Kind == economysim.DecideEnter, d.Agent, d.Role, d.Households, d.Cause)
	}
	// Output:
	// true false 1 0 1 starved
	// false true 0 2 1 renewal
}
