package economysim

import (
	"slices"
	"testing"
)

// The tests of the readers of the institutions module in the contract round of P3.3
// (spec-merchants.md 9.3, spec-institutions.md 5.2.1, 6.7): row 5a, the destination's sell
// and buy rights, and the clamp of a legal price (Phase 7). The fixture is merchConfig:
// brokerHome of a_home and brokerFar of b_far, each the only household of the role
// broker in its region.

// merOrg adds the institutions module to merchConfig: the body carriers of the brokers of
// the regions given, with unlimited places, and a right in b_far on grain, of the activity.
// With the brokers' region in the scope the broker is a member from the round of tick 0.
func merOrg(activity string, scope ...string) func(c *Config) {
	return func(c *Config) {
		c.Modules.Institutions = &Institutions{
			Bodies: []Body{{ID: "carriers", Roles: []string{"broker"}, Regions: scope}},
			Rights: []Right{{ID: "grain_far", Body: "carriers", Activity: activity, Subject: "grain", Region: "b_far"}},
		}
	}
}

// TestContract_DestinationRightsBind verifies that the sell right of the destination binds
// an offer and the buy right of the destination a take (spec-institutions.md 5.2.1): with
// the carriers body over the destination only, the home broker is not a member and its
// offer is refused not_permitted (row 5a, before the amounts), the far broker is a member
// and its take is admitted; with the body over the origin only the far broker's take is
// refused and the home broker's offer is admitted.
//
// Requirement: FR-ORG-035, FR-MER-031
func TestContract_DestinationRightsBind(t *testing.T) {
	t.Parallel()
	offer := DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 10, Limit: 100}
	take := DeliveryTake{Commodity: cGrain, Quantity: 10, Limit: 100}
	tests := []struct {
		name     string
		activity string
		scope    []string
		plan     roundPlan
		offers   []string
		takes    []string
	}{
		{"offer by a non-member", ActivitySell, []string{"b_far"}, roundPlan{offers: map[AgentID][]DeliveryOffer{brokerHome: {offer}}}, []string{"not_permitted"}, nil},
		{"offer by a member", ActivitySell, []string{"a_home", "b_far"}, roundPlan{offers: map[AgentID][]DeliveryOffer{brokerHome: {offer}}}, nil, nil},
		{"take by a non-member", ActivityBuy, []string{"a_home"}, roundPlan{takes: map[AgentID][]DeliveryTake{brokerFar: {take}}}, nil, []string{"not_permitted"}},
		{"take by a member", ActivityBuy, []string{"b_far"}, roundPlan{takes: map[AgentID][]DeliveryTake{brokerFar: {take}}}, nil, nil},
		{"a sell right does not bind a take", ActivitySell, []string{"a_home"}, roundPlan{takes: map[AgentID][]DeliveryTake{brokerFar: {take}}}, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec, cp := roundWorld(t, merOrg(tt.activity, tt.scope...), tt.plan, brokerHome)
			step(t, w, 1)
			offers, takes := refusals(cp)
			if !slices.Equal(offers, tt.offers) || !slices.Equal(takes, tt.takes) {
				t.Errorf("refusals = offers %v, takes %v, want offers %v, takes %v", offers, takes, tt.offers, tt.takes)
			}
			got := of[RightRefused](rec)
			if want := len(tt.offers) + len(tt.takes); (len(got) == 1) != (want == 1) || (len(got) == 1 && got[0].Region != regFar) {
				t.Errorf("RightRefused = %+v, want one event of the destination b_far exactly when a refusal is expected (%d)", got, want)
			}
		})
	}
}

// TestContract_RightRowIsAfterGoodsBeforeAmounts verifies the place of row 5a: an offer
// that fails row 5 (a commodity outside the tradable set) is refused for it, and one that
// fails both the right and row 6 (a quantity of 0) is refused not_permitted.
//
// Requirement: FR-MER-031
func TestContract_RightRowIsAfterGoodsBeforeAmounts(t *testing.T) {
	t.Parallel()
	bad := DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 0, Limit: 100}
	w, _, cp := roundWorld(t, merOrg(ActivitySell, "b_far"), roundPlan{offers: map[AgentID][]DeliveryOffer{brokerHome: {bad}}}, brokerHome)
	step(t, w, 1)
	if offers, _ := refusals(cp); !slices.Equal(offers, []string{"not_permitted"}) {
		t.Errorf("refusals of an offer that fails the right and the amount = %v, want [not_permitted]", offers)
	}
}
