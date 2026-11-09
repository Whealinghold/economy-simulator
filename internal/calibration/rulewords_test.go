package calibration

import "testing"

// TestRuleWords_Spelling verifies that every rule word of the process and record
// checks is the snake-case word the spec and the records use, and that no two
// are the same.
//
// Guard: a rule word is read by tools that filter a list of problems; a word
// changed to another spelling, or emptied, would make two rules look alike or
// hide one. Every test compares a rule with its constant, so only this test
// pins the spelling.
func TestRuleWords_Spelling(t *testing.T) {
	t.Parallel()
	want := map[string]string{
		RuleStageName: "stage_name", RuleStageOrder: "stage_order", RuleAttemptNumber: "attempt_number",
		RuleAttempts: "attempts", RuleOpenExact: "open_exact_fact", RuleBlockRange: "block_range",
		RuleBlockReused: "block_reused", RuleSearchSeed: "search_seed", RuleConfirmSeed: "confirm_seed",
		RuleMaxFactors: "max_factors", RuleStageKnob: "stage_knob", RuleOwnerSplit: "owner_split",
		RuleLineageParent: "lineage_parent", RuleLineageRestart: "lineage_restart", RuleLineageSet: "lineage_set",
		RuleLineageDoc: "lineage_document", RuleStopExact: "stop_exact", RuleStopDetector: "stop_detector",
		RuleStopBSet: "stop_b_set", RuleStopBWorking: "stop_b_working", RuleStopFinding: "stop_finding",
		RuleStopDormant: "stop_dormant", RuleStopControl: "stop_control",
		RuleKnobUnknown: "knob_unknown", RuleUnsearchedV1x: "unsearched_v1x", RuleUnsearchedAtOff: "unsearched_at_off",
		RuleFactUnknown: "fact_unknown", RuleFactClass: "fact_class", RuleDetectorUnknown: "detector_unknown",
		RuleSetKey: "set_key", RuleSetDormant: "set_dormant", RuleSetOff: "set_off", RuleFeasibleCount: "feasible_count",
		RuleChosenRange: "chosen_range", RuleScreenedDormant: "screened_dormant", RuleScreenedMeasure: "screened_measures",
		RuleScreenMeasure: "screen_measure", RuleScreenSign: "screen_sign", RuleClaim: "claim", RuleHoldout: "holdout",
		RuleInSample: "in_sample", RuleStopFact: "stop_fact",
	}
	if len(want) != 42 {
		t.Fatalf("table has %d distinct rule constants, want 42 (23 of the process, 19 of the record)", len(want))
	}
	for got, spelled := range want {
		if got != spelled {
			t.Errorf("rule word = %q, want %q", got, spelled)
		}
	}
}
