package steady_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// gainStrings renders gains as "path=num/den" for comparison.
func gainStrings(gs []steady.Gain) []string {
	var out []string
	for _, g := range gs {
		out = append(out, fmtPath(g.Commodities)+"="+g.Ratio.Num.String()+"/"+g.Ratio.Den.String())
	}
	return out
}

// fmtPath renders a path of commodities as "0>1>2".
func fmtPath(p []int) string {
	s := ""
	for i, c := range p {
		if i > 0 {
			s += ">"
		}
		s += string(rune('0' + c))
	}
	return s
}

// TestCycleGains verifies the enumeration of the cycles with one maker per
// commodity and their ratios.
//
// Requirement: FR-CFG-044
func TestCycleGains(t *testing.T) {
	t.Parallel()
	pair := func(c, q int64) [][2]int64 { return [][2]int64{{c, q}} }
	// The seed cycle of `one`: 0 seed, 1 staple. gather_seed is a source;
	// save_seed 20 staple -> 20 seed; grow_staple 40 seed -> 1 800 staple.
	// (40 / 1 800) * (20 / 20) = 1/45.
	seedCycle := []steady.Candidate{
		{Recipe: 0, Activity: act(nil, pair(0, 5))},
		{Recipe: 1, Activity: act(pair(1, 20), pair(0, 20))},
		{Recipe: 2, Activity: act(pair(0, 40), pair(1, 1800))},
	}
	// Two makers of commodity 0 (recipes 0 and 1, both with inputs): not enumerated.
	twoMakers := []steady.Candidate{
		{Recipe: 0, Activity: act(pair(0, 1), pair(0, 2))},
		{Recipe: 1, Activity: act(pair(1, 1), pair(0, 2))},
		{Recipe: 2, Activity: act(pair(0, 1), pair(1, 1))},
	}
	// 0 -> 1 -> 0 and 0 -> 2 -> 0: recipe 1 makes 1 from 0 (3 in, 1 out), recipe 2
	// makes 2 from 0 (1 in, 1 out), recipe 3 makes 0 from 1 and 2 (2 of each in, 4 out).
	// Cycle 0>1: (3/1) * (2/4) = 3/2. Cycle 0>2: (1/1) * (2/4) = 1/2.
	// Recipe 3 is the one maker of 0, so both cycles are enumerated.
	diamond := []steady.Candidate{
		{Recipe: 1, Activity: act(pair(0, 3), pair(1, 1))},
		{Recipe: 2, Activity: act(pair(0, 1), pair(2, 1))},
		{Recipe: 3, Activity: act([][2]int64{{1, 2}, {2, 2}}, pair(0, 4))},
	}
	// A hand cycle with q_in / q_out = 4/5 and a tool: 4 in, 5 out; the tool
	// (wear 1 of 2) is a second need of commodity 1, ratio (1/2) / 5.
	tool := act(pair(0, 4), pair(0, 5))
	tool.Tools = []steady.Tool{{Commodity: 1, Wear: 1, Durability: 2}}
	selfLoop := []steady.Candidate{{Recipe: 0, Activity: tool}, {Recipe: 1, Activity: act(nil, pair(1, 1))}}
	tests := []struct {
		name string
		comp []int
		cs   []steady.Candidate
		want []string
	}{
		{"seed cycle of one", []int{0, 1}, seedCycle, []string{"0>1=1/45"}},
		{"two makers are not enumerated", []int{0, 1}, twoMakers, nil},
		{"two cycles", []int{0, 1, 2}, diamond, []string{"0>1=3/2", "0>2=1/2"}},
		{"self loop 4/5", []int{0}, selfLoop[:1], []string{"0=4/5"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := steady.CycleGains(tt.comp, tt.cs, 1000)
			if !ok {
				t.Fatalf("CycleGains(%s) ok = false, want true", tt.name)
			}
			gs := gainStrings(got)
			if len(gs) != len(tt.want) {
				t.Fatalf("CycleGains(%s) = %v, want %v", tt.name, gs, tt.want)
			}
			for i := range gs {
				if gs[i] != tt.want[i] {
					t.Errorf("CycleGains(%s)[%d] = %s, want %s", tt.name, i, gs[i], tt.want[i])
				}
			}
		})
	}
	t.Run("tool counts as a need", func(t *testing.T) {
		t.Parallel()
		// Commodity 0 made from 4 of 0 and a tool 1: the tool is a need but 1 has no
		// maker here, so only the self loop on 0 is a cycle: 4/5.
		got, ok := steady.CycleGains([]int{0, 1}, selfLoop, 1000)
		gs := gainStrings(got)
		if !ok || len(gs) != 1 || gs[0] != "0=4/5" {
			t.Errorf("CycleGains(tool) = %v, %v, want [0=4/5]", gs, ok)
		}
	})
	t.Run("budget exceeded is skipped", func(t *testing.T) {
		t.Parallel()
		got, ok := steady.CycleGains([]int{0, 1, 2}, diamond, 1)
		if ok || got != nil {
			t.Errorf("CycleGains(budget 1) = %v, %v, want nil, false", got, ok)
		}
	})
}
