package otelobserver

import (
	"math"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestRatioBP_TruncatesAndDoesNotWrap verifies the basis-point ratio: n * 10 000
// / d truncated toward zero, computed in 128 bits so that a large n does not
// wrap, and 0 where the ratio is undefined.
//
// Requirement: FR-OBS-098
func TestRatioBP_TruncatesAndDoesNotWrap(t *testing.T) {
	tests := []struct {
		name string
		n, d int64
		want int64
	}{
		{"one third truncates", 1, 3, 3333},
		{"all is 10 000", 8, 8, 10000},
		{"zero numerator", 0, 5, 0},
		{"zero denominator", 5, 0, 0},
		{"negative denominator", 5, -5, 0},
		{"high word equal to the denominator saturates", 1_000_000_000_000_000_000, 542, math.MaxInt64},
		{"negative numerator", -1, 5, 0},
		{"large numerator does not wrap (just under a half)", math.MaxInt64 / 2, math.MaxInt64, 4999},
		{"ratio beyond int64 saturates", math.MaxInt64, 1, math.MaxInt64},
		{"quotient beyond int64 saturates", 1_000_000_000_000_000_000, 1000, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ratioBP(tt.n, tt.d); got != tt.want {
				t.Errorf("ratioBP(%d, %d) = %d, want %d", tt.n, tt.d, got, tt.want)
			}
		})
	}
}

// TestIx_RejectsNegativeAndHuge verifies that an ID outside the dense range maps
// to -1, which no table holds, so the sample is dropped.
//
// Requirement: FR-OBS-052
func TestIx_RejectsNegativeAndHuge(t *testing.T) {
	if got := ix(int64(-3)); got != -1 {
		t.Errorf("ix(-3) = %d, want -1", got)
	}
	if got := ix(int64(1) << 40); got != -1 {
		t.Errorf("ix(1<<40) = %d, want -1", got)
	}
	if got := ix(int64(1) << 30); got != 1<<30 {
		t.Errorf("ix(1<<30) = %d, want %d: the largest index", got, 1<<30)
	}
	if got := ix(int64(1)<<30 + 1); got != -1 {
		t.Errorf("ix(1<<30+1) = %d, want -1", got)
	}
	if got := ix(int32(0)); got != 0 {
		t.Errorf("ix(0) = %d, want 0", got)
	}
	if got := ix(int32(7)); got != 7 {
		t.Errorf("ix(7) = %d, want 7", got)
	}
}

// TestClosed_SetsAreThoseOfTheSpec verifies the closed sets of attribute values
// against spec-observability.md 6.5.1, in the order the handlers index them, and
// that a constant outside a set has no place.
//
// Guard: a handler writes a place by position, so a reorder or a typo here moves
// a count under another label without any other test noticing.
func TestClosed_SetsAreThoseOfTheSpec(t *testing.T) {
	want := map[string][]string{
		keyBound:       {"floor", "ceiling"},
		keyBeliefLimit: {"conflict", "clamped"},
		keyWorkDecline: {"rest", "no_recipe"},
		keyCause: {"insolvent", "starved", "consumer", "population", "transport", "migration", "delivered", "short",
			"payer_removed", "deliverer_removed"},
		keyEntryLimit: {"unfunded", "no_viable_role", "region_cap", "role_cap", "role_ceiling", "cooldown",
			"hysteresis", "exit_cap", "refused", "stranded"},
		keyKind:    {"faucet", "sink"},
		keySide:    {"bid", "ask"},
		keyOutcome: {"offered", "no_path", "too_old", "not_yet"},
		keyLimit: {"no_estimate", "no_destination", "no_rumour", "too_far", "no_viable", "no_gain", "bound",
			"out_cap", "unprovisioned", "in_cap", "held", "ended"},
		keyDecision: {"no_rumour", "stale", "locked", "perishable", "no_margin", "unaffordable", "below_share",
			"planned", "rested", "waiting", "lot_capped"},
		keyHolder:          {"agents", "estate", "treasury"},
		keySector:          {"households", "pool", "contract", "treasury"},
		keyPopulationLimit: {"deaths_cut", "deaths_held", "births_capped", "births_unfunded", "rate_clamped"},
		keyFlowLimit:       {"short", "headroom", "limit"},
		keyOwnUseChange:    {"entered", "left"},
		keyExternalCut:     {"closed", "cap", "price"},
		keyContractFailure: {"no_taker", "payer_funds", "too_many", "dispatch_refused"},
		keyContractRequest: {"offer", "take"},
		keyDirection:       {"from", "to"},
	}
	for key, w := range want {
		if !slices.Equal(closed[key], w) {
			t.Errorf("closed[%s] = %v, want %v", key, closed[key], w)
		}
	}
	if len(closed) != len(want) {
		t.Errorf("closed has %d sets, want %d", len(closed), len(want))
	}
	if got := closedIndex(keyCause, "nonsense"); got != -1 {
		t.Errorf("closedIndex(cause, nonsense) = %d, want -1", got)
	}
}

// TestIntern_CapOfOpenReasons verifies the production cap of the open reasons
// (money flow IDs and the like) is maxInterned.
//
// Guard: the cap is what keeps a flood of distinct strings from growing the series.
func TestIntern_CapOfOpenReasons(t *testing.T) {
	n := newNames(economysim.Names{})
	if n.other.max != 1024 || maxInterned != 1024 {
		t.Errorf("open reasons cap = %d, maxInterned = %d, want 1024", n.other.max, maxInterned)
	}
}
