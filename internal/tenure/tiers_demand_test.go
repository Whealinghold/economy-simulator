package tenure_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// TestTiers_DemandBeyondInt64IsRefused verifies that requests whose units wanted
// sum beyond int64 are refused with every output 0, whether the sum builds up
// among the requests of one holder (the own tier) or among the residual demands
// of the commons tier, and not wrapped into a small demand that would be served.
//
// Guard: the own and the commons tiers each ration through their own call, so a
// refusal of one is a separate branch; without this test a change that ignored
// the failure of either would serve a wrapped demand and the other tests of the
// package would not notice.
//
// Requirement: FR-LND-007, NFR-LND-003
func TestTiers_DemandBeyondInt64IsRefused(t *testing.T) {
	t.Parallel()
	big := int64(math.MaxInt64/2 + 1) // two of them are 2^63
	tests := []struct {
		name    string
		p       tenure.Params
		holders []tenure.Holder
		reqs    []tenure.Request
	}{
		{
			name:    "two requests of one holder",
			p:       tenure.Params{Stock: 10},
			holders: []tenure.Holder{{ID: 1, Entitled: 10}},
			reqs: []tenure.Request{
				{Ref: 0, ID: 1, Principal: 1, Executions: big, Draw: 1},
				{Ref: 1, ID: 2, Principal: 1, Executions: big, Draw: 1},
			},
		},
		{
			name: "two requests with no own entitlement",
			p:    tenure.Params{Stock: 10, Commons: 10},
			reqs: []tenure.Request{
				{Ref: 0, ID: 1, Principal: 1, Executions: big, Draw: 1},
				{Ref: 1, ID: 2, Principal: 2, Executions: big, Draw: 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var s tenure.Scratch
			res, ok := s.Tiers(tt.p, tt.holders, tt.reqs)
			if ok || res != (tenure.Result{}) {
				t.Errorf("Tiers(%+v) = %+v, %v, want refused with a zero result", tt.p, res, ok)
			}
			for _, r := range tt.reqs {
				if r.Own != 0 || r.Commons != 0 || r.Let != 0 || r.PricedOut != 0 {
					t.Errorf("Tiers(%+v): request %d = %+v, want every output 0", tt.p, r.ID, r)
				}
			}
			for _, h := range tt.holders {
				if h.Unused != 0 {
					t.Errorf("Tiers(%+v): holder %d unused = %d, want 0", tt.p, h.ID, h.Unused)
				}
			}
		})
	}
}
