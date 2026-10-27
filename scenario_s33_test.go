package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/moneyflow"
)

// TestScenarioS33_PairDriftsByItsDifference verifies MON-S1 (spec-money-supply.md
// 16, 17.3), the pair half: a fixed grant to 50 households of one role and a fixed
// levy on 450 of another, the amounts of 17.3 (300 and 33 a household-tick), run
// 400 ticks with the audit on. Exact: the boundary balance of the grant is minus
// 15 000 a tick, that of the levy plus 14 850 a tick, their net is minus 150 a
// tick times the ticks, and the money held grew by exactly that; the nominal
// totals differ by 1% so the warning pair_unbalanced is not raised at the
// tolerance of 1 000 basis points. The pair is built in Go: no overlay of it
// exists (plan 2-6 Cross-plan contract).
//
// Requirement: FR-MON-026, FR-MON-012, FR-MON-039, FR-REF-023
//
// Guard: a pair with unequal amounts drifts the money by its difference, which is
// the whole of what a pair promises; a mirror rule that evened it out would be a
// control law the module does not have (FR-MON-024).
func TestScenarioS33_PairDriftsByItsDifference(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const ticks = 400
	grant := flowTo("stipend", economysim.FlowGrant, "farmer")
	grant.Amount = 300
	levy := flowTo("due", economysim.FlowLevy, "baker")
	levy.Amount = 33
	c := moneyWorldConfig(50, 450, grant, levy)
	c.Roles[1].StartMoney = 20_000 // 33 * 400 = 13 200 a payer, so no payer is ever short
	for _, w := range c.Warnings() {
		if w.Rule == "pair_unbalanced" {
			t.Errorf("warning pair_unbalanced raised for totals 1%% apart, want none at the tolerance of 1 000 basis points")
		}
	}
	for _, seed := range seedList(min(3, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			w, err := economysim.NewWorld(c, economysim.WithSeed(seed), economysim.WithAudit())
			if err != nil {
				t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
			}
			before := roleMoney(t, w)
			step(t, w, ticks)
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot error = %v, want nil", seed, err)
			}
			g, l := flowRow(t, w, "stipend").Total, flowRow(t, w, "due").Total
			if g != 300*50*ticks || l != 33*450*ticks {
				t.Errorf("seed %d: totals %d and %d, want %d and %d", seed, g, l, 300*50*ticks, 33*450*ticks)
			}
			if net, want := g-l, economysim.Money(150*ticks); net != want || s.Money-before != want {
				t.Errorf("seed %d: net of the pair %d and money grew by %d, want both %d = 150 x %d ticks", seed, net, s.Money-before, want, ticks)
			}
		})
	}
}

// TestScenarioS33_HoldingChargeStaysBelowItsBound verifies MON-S1, the holding
// charge half, on the arithmetic of the module as the model does (spec-money-supply.md
// 9.3, 18.2): a household that never spends, a grant of 300 a tick and a rate levy
// of 5 000 basis points a year over 365 ticks with no exemption, has the balance
// m(t+1) = m(t) + g - rho * (m(t) + g) in the order of the step (the grant first,
// then the levy on what the grant gave). Exact: it never reaches the bound
// m* = E + g / rho = 219 000, rises every tick, and equals the model's
// recorded 28 016 at tick 100, 86 098 at tick 365 and 189 143 at tick 1 460. A household that trades can
// pass the bound, so the bound is checked here and not in a world (plan 2-6
// Phase 5).
//
// Requirement: FR-MON-015, FR-MON-012, FR-MON-013, FR-MON-020
//
// Guard: a levy that took the carry as a debt, or took past the balance, would
// move the curve off the model.
func TestScenarioS33_HoldingChargeStaysBelowItsBound(t *testing.T) {
	t.Parallel()
	const (
		period   = 365
		rateBP   = 5_000
		grant    = 300
		bound    = 219_000 // E + g / rho = 0 + 300 * 730
		maxMoney = economysim.MaxMoney
		scale    = 10_000
	)
	var m, gCarry, lCarry int64
	at := map[int]int64{}
	for tick := 1; tick <= 1_460; tick++ {
		due, c, ok := moneyflow.Spread(1, grant, scale, gCarry, 1, maxMoney)
		if !ok {
			t.Fatalf("Spread at tick %d: not ok", tick)
		}
		m, gCarry = m+due, c
		take, c, ok := moneyflow.RateLevy(m, rateBP, scale, lCarry, period, maxMoney)
		if !ok {
			t.Fatalf("RateLevy at tick %d: not ok", tick)
		}
		applied, _ := moneyflow.Clamp(take, m, maxMoney)
		m, lCarry = m-applied, c
		if m >= bound {
			t.Fatalf("balance %d at tick %d, want below the bound %d", m, tick, bound)
		}
		at[tick] = m
	}
	t.Logf("balances at ticks 100, 365, 1460: %d %d %d", at[100], at[365], at[1_460])
	for _, want := range []struct {
		tick   int
		figure int64
	}{{100, 28_016}, {365, 86_098}, {1_460, 189_143}} {
		if got := at[want.tick]; got != want.figure {
			t.Errorf("balance at tick %d = %d, want %d (the model's figure, spec-money-supply.md 18.2)", want.tick, got, want.figure)
		}
	}
}
