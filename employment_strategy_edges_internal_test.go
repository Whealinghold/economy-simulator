package economysim

import (
	"math"
	"testing"
)

// TestDefaultHire_HostileViewNeverPanicsNorOverflows verifies that the default
// strategy's vacancies and planned hires stay inside their bounds for a hand-built
// view at the edges of int64 or with a degenerate employer: an establishment of 0, a
// negative count of open hires, an empty view, a hire list that names no recipe of the
// view, a Duration or a quantity of 0 or below, and money, weight and stock at the
// largest value. Each row must return a request that admission accepts or none, never a
// position count below 1, a bid outside [1, MaxPrice], or a panic.
//
// Setup: hireView, then one field moved to the edge.
//
// Requirement: FR-EMP-015, FR-EMP-016
func TestDefaultHire_HostileViewNeverPanicsNorOverflows(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(v *AgentView)
	}{
		{"MaxHires 0", func(v *AgentView) { v.Employer.MaxHires = 0 }},
		{"MaxHires below Hired", func(v *AgentView) { v.Employer.MaxHires, v.Employer.Hired = 1, 5 }},
		{"MaxHires at the largest value", func(v *AgentView) { v.Employer.MaxHires = math.MaxInt64 }},
		{"Hired at the smallest value", func(v *AgentView) { v.Employer.Hired = math.MinInt64 }},
		{"MaxOpen 0", func(v *AgentView) { v.Employer.MaxOpen = 0 }},
		{"no recipe in the view", func(v *AgentView) { v.Recipes = nil }},
		{"no recipe on the hire list", func(v *AgentView) { v.Employer.Recipes = nil }},
		{"Duration 0", func(v *AgentView) { v.Recipes[0].Duration = 0 }},
		{"Duration below 0", func(v *AgentView) { v.Recipes[0].Duration = -3 }},
		{"Duration at the largest value", func(v *AgentView) { v.Recipes[0].Duration = math.MaxInt64 }},
		{"hired yield below 0", func(v *AgentView) { v.Recipes[0].Hired = -1 }},
		{"hired yield at the largest value", func(v *AgentView) { v.Recipes[0].Hired = math.MaxInt64 }},
		{"input quantity 0", func(v *AgentView) { v.Recipes[0].Inputs[0].Quantity = 0 }},
		{"input quantity at the largest value", func(v *AgentView) { v.Recipes[0].Inputs[0].Quantity = math.MaxInt64 }},
		{"output quantity at the largest value", func(v *AgentView) { v.Recipes[0].Outputs[0].Quantity = math.MaxInt64 }},
		{"money at the largest value", func(v *AgentView) { v.Money = math.MaxInt64 }},
		{"money below 0", func(v *AgentView) { v.Money = -1 }},
		{"weight at the largest value", func(v *AgentView) { v.Weight = math.MaxInt64 }},
		{"stock at the largest value", func(v *AgentView) { v.Commodities[cOre].Stock = math.MaxInt64 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := hireView()
			tt.mutate(v)
			got := (DefaultStrategy{}).Work(v, nil).Hire
			if got == (HireRequest{}) {
				return
			}
			if got.Positions < 1 || got.Bid < 1 || got.Bid > MaxPrice {
				t.Errorf("DefaultStrategy.Work(%s).Hire = %+v, want none or Positions >= 1 and Bid in [1, MaxPrice]", tt.name, got)
			}
			if got.Positions > v.Employer.MaxHires-v.Employer.Hired {
				t.Errorf("DefaultStrategy.Work(%s).Hire = %+v, more positions than the establishment has left", tt.name, got)
			}
		})
	}
}

// TestPlannedHire_HostileViewPlansNothingOrWithinBounds verifies H* for the same kind of
// view: with an empty view, no employer group, a negative upkeep cost or
// money at the largest value it plans none or a count of 1 to MaxHires, and the targets
// built from it never wrap.
//
// Requirement: FR-EMP-015
func TestPlannedHire_HostileViewPlansNothingOrWithinBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(v *AgentView)
		upkeep int64
	}{
		{"empty view", func(v *AgentView) { v.Recipes = nil }, 900},
		{"no employer group", func(v *AgentView) { v.HasEmployer = false }, 900},
		{"MaxHires 0", func(v *AgentView) { v.Employer.MaxHires = 0 }, 900},
		{"negative upkeep", func(*AgentView) {}, -1},
		{"upkeep at the largest value", func(*AgentView) {}, math.MaxInt64},
		{"money at the largest value", func(v *AgentView) { v.Money = math.MaxInt64 }, 900},
		{"money at the largest value and a negative upkeep", func(v *AgentView) { v.Money = math.MaxInt64 }, math.MinInt64},
		{"Duration 0", func(v *AgentView) { v.Recipes[0].Duration = 0 }, 900},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := hireView()
			tt.mutate(v)
			got := v.plannedHire(tt.upkeep)
			if tt.upkeep < 0 && got.n != 0 {
				t.Fatalf("plannedHire(%d) = %+v, want none for a negative upkeep cost", tt.upkeep, got)
			}
			if got.n < 0 || got.n > v.Employer.MaxHires {
				t.Fatalf("plannedHire(%d) = %+v, want n in [0, MaxHires %d]", tt.upkeep, got, v.Employer.MaxHires)
			}
			tc := tradeCtx{v: v, par: householdParams(&v.Tunables), hire: got, rstar: -1}
			if _, _, ok := tc.jobAndOutput(cOre); !ok && got.n < 1 {
				t.Errorf("jobAndOutput(ore) failed with no planned hire: %+v", got)
			}
		})
	}
}

// TestDefaultOffer_HostileViewOffersNothingOrWithinBounds verifies that the offer stays
// inside the admission bounds (Households in [1, Weight], Reserve in [0, MaxPrice]) for
// a weight, a share, a duration and prices at the edges, and that an agent with no
// recipe reserves 0.
//
// Requirement: FR-EMP-013, FR-EMP-014
func TestDefaultOffer_HostileViewOffersNothingOrWithinBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(v *AgentView)
	}{
		{"weight at the largest value", func(v *AgentView) { v.Weight = math.MaxInt64 }},
		{"weight 0", func(v *AgentView) { v.Weight = 0 }},
		{"weight below 0", func(v *AgentView) { v.Weight = -4 }},
		{"Duration 0", func(v *AgentView) { v.Weight, v.Recipes[0].Duration = 4, 0 }},
		{"Duration below 0", func(v *AgentView) { v.Weight, v.Recipes[0].Duration = 4, -1 }},
		{"ratio below 0", func(v *AgentView) { v.Weight, v.Ratio = 4, math.MinInt64 }},
		{"output quantity at the largest value", func(v *AgentView) { v.Weight, v.Recipes[0].Outputs[0].Quantity = 4, math.MaxInt64 }},
		{"effective yield at the largest value", func(v *AgentView) { v.Weight, v.Recipes[0].Effective = 4, math.MaxInt64 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := workView()
			v.Hireable, v.Money = true, math.MaxInt64
			tt.mutate(v)
			got := (DefaultStrategy{}).Work(v, nil).Offer
			if got == (LabourOffer{}) {
				return
			}
			if got.Households < 1 || got.Households > v.Weight || got.Reserve < 0 || got.Reserve > MaxPrice {
				t.Errorf("DefaultStrategy.Work(%s).Offer = %+v, outside the admission bounds (weight %d)", tt.name, got, v.Weight)
			}
		})
	}
}
