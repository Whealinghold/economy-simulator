package ledger

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

// TestDiscard_Int64OverflowRefusesBeforeAnyWrite verifies that a discard whose
// recipe_output boundary, disposal boundary or either flow total would leave
// int64 is refused with ErrBalanceOverflow and writes nothing, so the other
// boundary and the other flow are unchanged too.
//
// Each case breaks one limit only: the rig's balances and totals are small, and
// the one under test is set at the edge from inside the package.
//
// Requirement: FR-LED-005, FR-LED-007, FR-LED-066
func TestDiscard_Int64OverflowRefusesBeforeAnyWrite(t *testing.T) {
	t.Parallel()
	const commodity = 2 // plain
	tests := []struct {
		name string
		set  func(g *rig, a *account)
	}{
		{"recipe_output boundary at the minimum", func(g *rig, _ *account) {
			g.l.accounts[g.bnd[RecipeOutput][commodity]].balance = math.MinInt64
		}},
		{"disposal boundary at the maximum", func(g *rig, _ *account) {
			g.l.accounts[g.bnd[Disposal][commodity]].balance = math.MaxInt64
		}},
		{"recipe_output flow total at the maximum", func(g *rig, a *account) {
			g.l.flows()[g.l.slot(a, RecipeOutput, commodity)] = math.MaxInt64
		}},
		{"disposal flow total at the maximum", func(g *rig, a *account) {
			g.l.flows()[g.l.slot(a, Disposal, commodity)] = math.MaxInt64
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := newRig(t, scriptMaxMoney)
			id := g.holders[0]
			a := &g.l.accounts[id]
			tc.set(g, a)
			before := fingerprint(g.l)
			if err := g.l.Discard(id, commodity, 1); !errors.Is(err, ErrBalanceOverflow) {
				t.Errorf("Discard(%d, %d, 1) = %v, want ErrBalanceOverflow", id, commodity, err)
			}
			if after := fingerprint(g.l); !reflect.DeepEqual(after, before) {
				t.Errorf("Discard(%d, %d, 1) refused but changed the ledger", id, commodity)
			}
		})
	}
}
