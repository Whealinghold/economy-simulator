package economysim

import (
	"slices"
	"testing"
)

// TestDecide_RoleKeep verifies the role floor of spec-spawner.md 6.6: a rung-3 cut
// and a Switch that would leave the role under RoleKeep households are cut or not
// made, in ascending agent ID, and RoleKeep 0 changes nothing.
//
// Requirement: FR-SPN-062, FR-SPN-063
func TestDecide_RoleKeep(t *testing.T) {
	t.Parallel()
	// role 1 holds 2 + 1 households, role 2 is the better-paid destination.
	exits := []RegionAgent{spwUnfed(spwAgent(1, 1, 2), 12, 2), spwUnfed(spwAgent(2, 1, 1), 12, 1)}
	reviewers := []RegionAgent{spwAgent(1, 1, 1), spwAgent(2, 1, 1), spwAgent(3, 1, 1)}
	tests := []struct {
		name   string
		keep   int64
		agents []RegionAgent
		hh     int64
		want   []Decision
	}{
		// weight 2 drops min(2, max(1, ceil(2*1000/10000))) = 1; agent 2 drops 1.
		{"off removes", 0, exits, 3, []Decision{
			{Kind: DecideReduce, Agent: 1, Households: 1, Cause: CauseStarved},
			{Kind: DecideRemove, Agent: 2, Households: 1, Cause: CauseStarved}}},
		{"keep 2 cuts the second exit to 0", 2, exits, 3, []Decision{
			{Kind: DecideReduce, Agent: 1, Households: 1, Cause: CauseStarved}}},
		{"keep 3 cuts both", 3, exits, 3, nil},
		{"switch off", 0, reviewers, 3, []Decision{spwSwitch(1, 2, CauseReview), spwSwitch(2, 2, CauseReview), spwSwitch(3, 2, CauseReview)}},
		{"switch keep 1 lets two go", 1, reviewers, 3, []Decision{spwSwitch(1, 2, CauseReview), spwSwitch(2, 2, CauseReview)}},
		{"switch keep 3 lets none go", 3, reviewers, 3, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := spwView()
			v.RoleKeep = tt.keep
			v.Renewal = 0
			v.Roles = []RoleView{spwRole(1, wOwn, tt.hh), spwRole(2, w1320, 50)}
			v.Agents = slices.Clone(tt.agents)
			d := spwDecide(v)
			// the destination role has no agents of this test: only role 1 matters
			var got []Decision
			for _, x := range d.List {
				if x.Kind != DecideEnter {
					got = append(got, x)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Decide plan = %+v, want %+v", got, tt.want)
			}
		})
	}
}
