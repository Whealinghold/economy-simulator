package household

import "github.com/Whealinghold/economy-sim/internal/checked"

// AskFloor raises the limit of an ask to at least trunc(ref * bp / 10 000), the
// floor of World.AskFloorBP. bp 0 is off and returns limit. The product is
// through the 128-bit helper; a floor that does not fit leaves the limit as it
// is (spec-agents.md 7.3.5, FR-AGT-087).
func AskFloor(limit, ref, bp int64) int64 {
	if bp < 1 {
		return limit
	}
	floor, ok := checked.MulDiv(ref, bp, Basis)
	if !ok {
		return limit
	}
	return max(limit, floor)
}
