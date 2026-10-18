package economysim

import (
	"strings"
	"testing"
)

// The tests in this file pin edges of strategy_view.go that a mutation run
// (gomu, inside Docker) showed the other tests did not reach: the scratch
// starting empty and the wording of the errors fillCore returns.

// TestNewViewScratch_StartsEmpty verifies that every slice of a new scratch has
// length 0, so that a fill appends from the start and a plan starts empty.
//
// Requirement: NFR-AGT-001
//
// Guard: a scratch built with a non-zero length would show a zero row first.
func TestNewViewScratch_StartsEmpty(t *testing.T) {
	t.Parallel()
	cfg := viewConfig().canonical()
	rt, rc := newRoleTable(&cfg), newRecipeTable(&cfg)
	sc := newViewScratch(&cfg, &rt, &rc)
	lens := map[string]int{
		"trade.rows": len(sc.trade.rows), "commodities": len(sc.commodities), "recipes": len(sc.recipes),
		"upkeep": len(sc.upkeep), "inputs": len(sc.inputs), "outputs": len(sc.outputs), "tools": len(sc.tools),
		"costQ": len(sc.costQ), "costP": len(sc.costP), "plan.Intents": len(sc.plan.Intents),
	}
	for name, n := range lens {
		if n != 0 {
			t.Errorf("len(%s) of a new scratch = %d, want 0", name, n)
		}
	}
}

// TestFillCore_ErrorsNameTheirStep verifies that each refusal of fillCore is
// returned from its own step, with that step's wording, so that a log line says
// what failed.
//
// Requirement: FR-AGT-071
//
// Guard: an error dropped or replaced by a later step would change or lose the
// prefix asserted here.
func TestFillCore_ErrorsNameTheirStep(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		role   string
		break_ func(h *ViewHarness, i int)
		want   string
	}{
		{"account", "farmer", func(h *ViewHarness, i int) { h.agents[i].account = 1 << 20 },
			"fill view of agent 0: account 1048576: "},
		{"executions", "baker", func(h *ViewHarness, i int) {
			h.rt.row(0).inputs[0].commodity = 99
		}, "fill view of agent 0: executions of recipe 0: "},
		{"beliefs", "farmer", func(h *ViewHarness, i int) { h.agents[i].beliefs = h.agents[i].beliefs[:1] },
			"fill view of agent 0: 1 beliefs for 3 tradable commodities"},
		{"stock", "farmer", func(h *ViewHarness, i int) { h.roles.row(h.agents[i].role).tradable[0] = 99 },
			"fill view of agent 0: stock of commodity 99: "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewViewHarness(t, viewConfig())
			i := h.Spawn(t, tt.role, "north", 1)
			tt.break_(h, i)
			_, err := h.Fill(i)
			if err == nil || !strings.HasPrefix(err.Error(), tt.want) {
				t.Errorf("fillCore error = %v, want prefix %q", err, tt.want)
			}
		})
	}
}
