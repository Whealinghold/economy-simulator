package economysim_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestValidateFiscal_TreasuryTargetIsAProperDescendant verifies the edge of the rule
// that a remittance goes to the treasury of a governed jurisdiction below its authority:
// a child and a grandchild are accepted, the authority itself, its parent and a sibling
// are not (ErrNotCovered). The tree is crown > inland > uplands, crown > coast.
//
// Requirement: FR-FIS-007, FR-FIS-011
//
// Guard: the walk up the parents stops at the root; a rule that tested only the direct
// parent would reject the grandchild, one that included the authority would take its own
// treasury as the target of its own remittance.
func TestValidateFiscal_TreasuryTargetIsAProperDescendant(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		authority string
		target    string
		want      error
	}{
		{"a child", "crown", "inland", nil},
		{"a grandchild", "crown", "uplands", nil},
		{"the authority itself", "inland", "inland", economysim.ErrNotCovered},
		{"the parent of the authority", "inland", "crown", economysim.ErrNotCovered},
		{"a sibling of the authority", "inland", "coast", economysim.ErrNotCovered},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := fiscalConfig()
			// uplands and coast need a seat to be governed; seats are regions of their own.
			c.Modules.Polity.Jurisdictions[3].Seat = "hills"
			c.Modules.Polity.Jurisdictions[1].Seat = "harbour"
			c.Modules.Fiscal.Taxes = []economysim.Tax{{
				ID: "remit", Authority: tt.authority,
				Target: economysim.FlowTarget{Kind: economysim.TargetTreasury, Jurisdiction: tt.target}, RateBP: 100,
			}}
			err := c.Validate()
			if tt.want == nil && err != nil {
				t.Fatalf("Validate() with the target %s of authority %s error = %v, want nil", tt.target, tt.authority, err)
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Errorf("Validate() with the target %s of authority %s error = %v, want %v", tt.target, tt.authority, err, tt.want)
			}
		})
	}
}
