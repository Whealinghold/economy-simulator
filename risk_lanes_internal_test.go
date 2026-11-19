package economysim

import (
	"errors"
	"testing"
)

// TestTransportLanes_UnvalidatedRiskRowIsAnError verifies that TransportLanes, which
// takes a configuration that no validator has seen, returns an error for a risk row
// whose numbers are outside 0 to 10 000 instead of panicking through invariant.
//
// Fix: dp3-rsk-2 stage 4. configuredLoss called effectiveP and expectedOf, whose
// invariant panics are for validated rows, so a negative strike or a severity above
// 10 000 in a public call crashed the caller (architecture.md 8.1).
//
// Requirement: FR-RSK-026
func TestTransportLanes_UnvalidatedRiskRowIsAnError(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name               string
		strikeBP, severity int64
	}{
		{"negative strike", -1, 2_500},
		{"negative severity", 3_000, -1},
		{"severity above 10000", 3_000, 10_001},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := shipConfig()
			riskShip(riskCoach("a_home", "b_far", tt.strikeBP, tt.severity, false))(&c)
			lanes, err := c.TransportLanes()
			if !errors.Is(err, ErrOutOfBounds) || lanes != nil {
				t.Errorf("TransportLanes() = (%d lanes, %v), want no lanes and ErrOutOfBounds", len(lanes), err)
			}
		})
	}
}
