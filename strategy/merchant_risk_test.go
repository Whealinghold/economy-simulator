package strategy_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestMerchant_NetOfRouteLoss verifies spec-risk.md 8.4.1 and FR-RSK-027: the revenue
// per unit is net of the expected loss of the route after the consignee's cut,
// Rnet * (10 000 - LossBP) / 10 000 truncated. In the valley of example A the rumour
// gives 305 and the cut of 500 gives 305 * 9 500 / 10 000 = 289; at a loss of 1 000 the
// revenue is 289 * 9 000 / 10 000 = 260, above the local 200, and the candidate is
// planned; at 5 000 it is 144, below the local price, and is counted NoMargin.
//
// Requirement: FR-RSK-027
func TestMerchant_NetOfRouteLoss(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		loss int64
		want economysim.PlanNotes
	}{
		{"no loss", 0, economysim.PlanNotes{Considered: 1, Planned: 1}},
		{"a loss of 1 000", 1_000, economysim.PlanNotes{Considered: 1, Planned: 1}},
		{"a loss of 5 000", 5_000, economysim.PlanNotes{Considered: 1, NoMargin: 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := valley()
			v.Routes[0].LossBP = tt.loss
			p, _ := trade(t, merchantStrategy, v)
			if p.Notes.Considered != tt.want.Considered || p.Notes.NoMargin != tt.want.NoMargin || p.Notes.Planned != tt.want.Planned {
				t.Errorf("notes = %+v, want considered %d, no margin %d, planned %d", p.Notes, tt.want.Considered, tt.want.NoMargin, tt.want.Planned)
			}
		})
	}
}
