package checked_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Equivalent mutants of checked.go found by gomu (no test can kill them):
//   - Add/Sub l35,l46: `b >= 0` -> `b > 0` and Sub `d >= 0` -> `d > 0`; b == 0
//     never overflows and a-b == 0 is impossible when the signs differ.
//   - `hi != 0`, `ur != 0`, `carry != 0` -> `> 0`: unsigned operands.
//   - `ur == rest` -> `>=` (already covered by `ur > rest ||`) and `uq&1 == 1`
//     -> `>= 1` (the value is 0 or 1).
//   - `a < 0` -> `a <= 0` in Mul/muldiv/abs: a zero operand gives a zero
//     magnitude and a zero product whatever the sign.
//   - return_zero_value mutants replace 0 with 0 (identical source).

// TestZeroOperandSigns verifies that a zero operand of either sign yields zero
// with a zero remainder, which is why the `<` -> `<=` mutants are equivalent.
//
// Requirement: FR-CUR-003
func TestZeroOperandSigns(t *testing.T) {
	t.Parallel()
	for _, v := range []int64{-5, 5} {
		if p, ok := checked.Mul(0, v); !ok || p != 0 {
			t.Errorf("Mul(0, %d) = %d, %v, want 0, true", v, p, ok)
		}
		if p, ok := checked.Mul(v, 0); !ok || p != 0 {
			t.Errorf("Mul(%d, 0) = %d, %v, want 0, true", v, p, ok)
		}
		if q, r, ok := checked.MulDivRem(0, v, -3); !ok || q != 0 || r != 0 {
			t.Errorf("MulDivRem(0, %d, -3) = %d, %d, %v, want 0, 0, true", v, q, r, ok)
		}
	}
}
