package calibration

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

const testMaxFactors = 16

// procKnown is a small registry: BEL, AGT and SPN rows of C4 (SPN05 is a D row),
// a C8 row and an S row of C2.
func procKnown() Known {
	return Known{Knobs: map[string]KnobInfo{
		"K-BEL01": {Class: 'K', Key: "World.CostFloor", Stage: "C4", Owner: "BEL"},
		"K-BEL02": {Class: 'K', Key: "World.BeliefStep", Stage: "C4", Owner: "BEL"},
		"K-AGT01": {Class: 'K', Key: "World.Reserve", Stage: "C4", Owner: "AGT"},
		"K-SPN05": {Class: 'D', Key: "World.Cooldown", Stage: "C4", Owner: "SPN"},
		"K-MON01": {Class: 'K', Key: "World.Faucet", Stage: "C8", Owner: "MON"},
		"S-PRO01": {Class: 'S', Key: "Role.ExpectedWorkTicks", Stage: "C2", Owner: "PRO"},
	}}
}

// goodSearch is a valid stage record that searches on block b and stops: one
// factor on CostFloor, a screen and an explore on the block, a confirm on 1 to 3
// with a control, and one passing fact of each class.
func goodSearch(stage string, attempt, block int) Record {
	first := uint64(100*block + 1)
	return Record{
		Pack: "one", Lineage: "5a1c0e4477d3b210", Stage: stage, Attempt: attempt, Block: block,
		Documents: []Document{{"base.toml", "3b9c11aa5e07f4d2"}, {"warm-one.toml", "0be4c0319d7a52f6"}},
		Experiments: []Experiment{
			{Step: "screen", Design: "oat", Points: 3, Replicas: 10, Seed0: first, Seeding: "same"},
			{Step: "explore", Design: "spread", Points: 16, Replicas: 10, Seed0: first, Seeding: "same"},
			{Step: "confirm", Design: "grid", Points: 2, Replicas: 3, Seed0: 1, Seeding: "same"},
		},
		Factors:   []Factor{{Name: "cost_floor", Key: "World.CostFloor"}},
		Confirm:   ConfirmLine{Has: true, First: 1, Last: 3, Control: true},
		Facts:     []Fact{{ID: "P01", Class: 'X', Result: "pass"}, {ID: "P02", Class: 'B', Target: "set", Result: "pass"}, {ID: "P03", Class: 'B', Target: "working", Result: "pass"}, {ID: "P04", Class: 'R', Result: "report"}},
		Detectors: []Detector{{ID: "R5a", Verdict: "none"}},
		Decision:  "stop",
	}
}

// stageOnly is a record of a stage with no search.
func stageOnly(stage string, attempt int, decision string) Record {
	return Record{Pack: "one", Lineage: "5a1c0e4477d3b210", Stage: stage, Attempt: attempt, Decision: decision}
}

func rules(ps []Problem) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Rule
	}
	return out
}

// TestProcess_StageOrderAndReentry verifies that stages run in the order C0 to C9,
// that a stage is left only after a stop, that a stopped stage is not run again,
// and that going back needs a reentry decision naming the stage.
//
// Requirement: FR-CLB-035
func TestProcess_StageOrderAndReentry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rs   []Record
		want []string
	}{
		{"in order with a skip", []Record{stageOnly("C1", 1, "stop"), stageOnly("C3", 1, "stop"), stageOnly("C4a", 1, "stop")}, nil},
		{"back with no reentry", []Record{stageOnly("C2", 1, "stop"), stageOnly("C1", 1, "stop")}, []string{RuleStageOrder}},
		{"left a stage that did not stop", []Record{stageOnly("C1", 1, "revert"), stageOnly("C2", 1, "stop")}, []string{RuleStageOrder}},
		{"a stopped stage is run again", []Record{stageOnly("C2", 1, "stop"), stageOnly("C2", 1, "stop")}, []string{RuleStageOrder}},
		{"reentry to the named stage", []Record{stageOnly("C3", 1, "reentry C2"), stageOnly("C2", 1, "stop"), stageOnly("C3", 1, "stop")}, nil},
		{"reentry to another stage", []Record{stageOnly("C3", 1, "reentry C2"), stageOnly("C3", 1, "stop")}, []string{RuleStageOrder}},
		{"an unknown stage", []Record{stageOnly("C10", 1, "stop")}, []string{RuleStageName}},
		{"a retry of the same stage", []Record{stageOnly("C2", 1, "revert"), stageOnly("C2", 2, "stop")}, nil},
		{"sub-stages in order", []Record{stageOnly("C4a", 1, "stop"), stageOnly("C4b", 1, "stop"), stageOnly("C4c", 1, "stop")}, nil},
		{"sub-stage back", []Record{stageOnly("C4b", 1, "stop"), stageOnly("C4a", 1, "stop")}, []string{RuleStageOrder}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := rules(CheckSequence(tt.rs, procKnown(), testMaxFactors)); !slices.Equal(got, tt.want) {
				t.Errorf("CheckSequence(%s) rules = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
	// An unknown stage after a known one is reported once and does not cascade.
	rs := []Record{stageOnly("C2", 1, "stop"), stageOnly("X", 1, "stop")}
	if got := rules(CheckSequence(rs, procKnown(), testMaxFactors)); !slices.Equal(got, []string{RuleStageName}) {
		t.Errorf("CheckSequence(C2, X) rules = %v, want only stage_name", got)
	}
}

// TestProcess_SplitByOwner verifies the cap on factors, that a factor belongs to
// its stage, and that an owner is searched in one sub-stage only.
//
// Requirement: FR-CLB-035
func TestProcess_SplitByOwner(t *testing.T) {
	t.Parallel()
	many := goodSearch("C4a", 1, 1)
	many.Factors = slices.Repeat(many.Factors, 3)
	wrong := goodSearch("C4a", 1, 1)
	wrong.Factors = []Factor{{Key: "World.Faucet"}} // a C8 knob in a C4 stage
	a := goodSearch("C4a", 1, 1)
	b := goodSearch("C4b", 1, 2)
	b.Factors = []Factor{{Key: "World.BeliefStep"}} // BEL again, in C4b
	c := goodSearch("C4b", 1, 2)
	c.Factors = []Factor{{Key: "World.Reserve"}} // AGT: another owner
	tests := []struct {
		name string
		rs   []Record
		max  int
		want []string
	}{
		{"at the cap", []Record{many}, 3, nil},
		{"over the cap", []Record{many}, 2, []string{RuleMaxFactors}},
		{"knob of another stage", []Record{wrong}, testMaxFactors, []string{RuleStageKnob}},
		{"an owner split across sub-stages", []Record{a, b}, testMaxFactors, []string{RuleOwnerSplit}},
		{"owners kept apart", []Record{a, c}, testMaxFactors, nil},
		{"an unknown key is not this check's", []Record{func() Record { r := a; r.Factors = []Factor{{Key: "role.baker.expected_work_ticks"}}; return r }()}, testMaxFactors, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := rules(CheckSequence(tt.rs, procKnown(), tt.max)); !slices.Equal(got, tt.want) {
				t.Errorf("CheckSequence(%s) rules = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestProcess_ThreeAttempts verifies that a stage has three attempts before a
// review, that a review allows three more, and that attempts are numbered in
// order from 1.
//
// Requirement: FR-CLB-027
func TestProcess_ThreeAttempts(t *testing.T) {
	t.Parallel()
	at := func(attempt int, decision string) Record { return stageOnly("C2", attempt, decision) }
	tests := []struct {
		name string
		rs   []Record
		want []string
	}{
		{"three attempts", []Record{at(1, "revert"), at(2, "revert"), at(3, "stop")}, nil},
		{"a fourth with no review", []Record{at(1, "revert"), at(2, "revert"), at(3, "revert"), at(4, "stop")}, []string{RuleAttempts}},
		{"a fourth after a review", []Record{at(1, "revert"), at(2, "revert"), at(3, "review"), at(4, "stop")}, nil},
		{"a seventh after one review", []Record{at(1, "revert"), at(2, "revert"), at(3, "review"), at(4, "revert"), at(5, "revert"), at(6, "revert"), at(7, "stop")}, []string{RuleAttempts}},
		{"a sixth after one review", []Record{at(1, "revert"), at(2, "revert"), at(3, "review"), at(4, "revert"), at(5, "revert"), at(6, "stop")}, nil},
		{"a skipped number", []Record{at(1, "revert"), at(3, "stop")}, []string{RuleAttemptNumber}},
		{"the first is not 2", []Record{at(2, "stop")}, []string{RuleAttemptNumber}},
		{"escalate counts as a failed attempt", []Record{at(1, "escalate"), at(2, "escalate"), at(3, "escalate"), at(4, "stop")}, []string{RuleAttempts}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := rules(CheckSequence(tt.rs, procKnown(), testMaxFactors)); !slices.Equal(got, tt.want) {
				t.Errorf("CheckSequence(%s) rules = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestProcess_OpenXFact verifies that no knob is searched while an X or O fact of
// the pack fails, that the fact is closed by a later pass, and that C0 is exempt.
//
// Requirement: FR-CLB-036
func TestProcess_OpenXFact(t *testing.T) {
	t.Parallel()
	failing := func(class byte) Record {
		r := stageOnly("C0", 1, "review")
		r.Facts = []Fact{{ID: "P01", Class: class, Result: "fail"}}
		return r
	}
	passing := func() Record {
		r := stageOnly("C0", 2, "stop")
		r.Facts = []Fact{{ID: "P01", Class: 'X', Result: "pass"}}
		return r
	}
	search := func(stage string, attempt int, block int) Record {
		r := goodSearch(stage, attempt, block)
		r.Decision = "revert"
		return r
	}
	tests := []struct {
		name string
		rs   []Record
		want []string
	}{
		{"an X fact fails then a search", []Record{failing('X'), search("C4a", 1, 1)}, []string{RuleOpenExact}},
		{"an O fact fails then a search", []Record{failing('O'), search("C4a", 1, 1)}, []string{RuleOpenExact}},
		{"a B fact failing is not exact", []Record{failing('B'), search("C4a", 1, 1)}, nil},
		{"closed by a pass", []Record{failing('X'), passing(), search("C4a", 1, 1)}, nil},
		{"C0 itself may fail twice", []Record{failing('X'), func() Record { r := failing('X'); r.Attempt = 2; return r }()}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// The failing C0 record asks for a review, which is not a stop: stage order
			// is not what is under test, so its problems are filtered out.
			var got []string
			for _, p := range CheckSequence(tt.rs, procKnown(), testMaxFactors) {
				if p.Rule == RuleOpenExact {
					got = append(got, p.Rule)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("CheckSequence(%s) open-fact problems = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestProcess_SevenStopConditions verifies each of the seven conditions of 7.4.1 on
// a valid stopping record: exactly the rule of the broken condition is reported.
//
// Requirement: FR-CLB-038
func TestProcess_SevenStopConditions(t *testing.T) {
	t.Parallel()
	if got := CheckSequence([]Record{goodSearch("C4a", 1, 1)}, procKnown(), testMaxFactors); len(got) != 0 {
		t.Fatalf("CheckSequence(valid stop) = %v, want none", got)
	}
	tests := []struct {
		name   string
		mutate func(*Record)
		want   string
	}{
		{"1 an X fact fails", func(r *Record) { r.Facts[0].Result = "fail" }, RuleStopExact},
		{"1 an O fact is only reported", func(r *Record) { r.Facts[0].Class, r.Facts[0].Result = 'O', "report" }, RuleStopExact},
		{"2 a detector fires", func(r *Record) { r.Detectors[0].Verdict = "detected" }, RuleStopDetector},
		{"2 a detector unknown with no working threshold", func(r *Record) { r.Detectors[0].Verdict = "unknown" }, RuleStopDetector},
		{"3 a B fact with a set key fails", func(r *Record) { r.Facts[1].Result = "fail" }, RuleStopBSet},
		{"4 a B fact with a working target fails", func(r *Record) { r.Facts[2].Result = "fail" }, RuleStopBWorking},
		{"5 a finding is open", func(r *Record) { r.Findings = []Finding{{Knob: "K-BEL01", Fact: "P02", Measure: "m", Status: "open"}} }, RuleStopFinding},
		{"6 a moved D knob stayed dormant", func(r *Record) {
			r.Sets = []Setting{{Key: "World.Cooldown", Value: 5}}
			r.Dormant = []string{"K-SPN05"}
		}, RuleStopDormant},
		{"7 no confirm", func(r *Record) { r.Confirm = ConfirmLine{} }, RuleStopControl},
		{"7 no control", func(r *Record) { r.Confirm.Control = false }, RuleStopControl},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := goodSearch("C4a", 1, 1)
			r.Facts = slices.Clone(r.Facts)
			r.Detectors = slices.Clone(r.Detectors)
			tt.mutate(&r)
			if got := rules(CheckSequence([]Record{r}, procKnown(), testMaxFactors)); !slices.Equal(got, []string{tt.want}) {
				t.Errorf("CheckSequence(stop, %s) rules = %v, want [%s]", tt.name, got, tt.want)
			}
			// The same record is no stop when it says revert: the conditions bind a stop only.
			r.Decision = "revert"
			if got := CheckSequence([]Record{r}, procKnown(), testMaxFactors); len(got) != 0 {
				t.Errorf("CheckSequence(revert, %s) = %v, want none", tt.name, got)
			}
		})
	}
	// Condition 2 accepts a detector that is unknown only at a working threshold, and
	// conditions 3 and 4 accept an unavailable fact (na).
	ok := goodSearch("C4a", 1, 1)
	ok.Detectors = []Detector{{ID: "R5a", Verdict: "unknown", Working: true}}
	ok.Facts = []Fact{{ID: "P02", Class: 'B', Target: "set", Result: "na"}}
	if got := CheckSequence([]Record{ok}, procKnown(), testMaxFactors); len(got) != 0 {
		t.Errorf("CheckSequence(working unknown, na fact) = %v, want none", got)
	}
}

// TestProcess_BlocksAndSeeds verifies that a block serves one choice of a pack, that
// a search is on its block and outside 1 to 60, and that a confirm is on 1 to 50 and
// shares no seed with a search.
//
// Requirement: FR-CLB-020, FR-CLB-021
func TestProcess_BlocksAndSeeds(t *testing.T) {
	t.Parallel()
	mod := func(f func(*Record)) Record {
		r := goodSearch("C4a", 1, 1)
		r.Experiments = slices.Clone(r.Experiments)
		f(&r)
		return r
	}
	failed := func(r Record) Record { r.Decision = "revert"; return r }
	tests := []struct {
		name string
		rs   []Record
		want []string
	}{
		{"block 1 then block 2", []Record{failed(goodSearch("C4a", 1, 1)), goodSearch("C4a", 2, 2)}, nil},
		{"block reused", []Record{failed(goodSearch("C4a", 1, 1)), goodSearch("C4a", 2, 1)}, []string{RuleBlockReused}},
		{"another pack may use block 1", []Record{failed(goodSearch("C4a", 1, 1)), func() Record { r := goodSearch("C4a", 1, 1); r.Pack = "two"; return r }()}, nil},
		{"block 0", []Record{mod(func(r *Record) { r.Block = 0 })}, []string{RuleBlockRange}},
		{"search seeds in 1 to 60", []Record{mod(func(r *Record) { r.Experiments[0].Seed0 = 55 })}, []string{RuleSearchSeed, RuleSearchSeed}},
		{"search seeds outside the block", []Record{mod(func(r *Record) { r.Experiments[1].Seed0 = 195 })}, []string{RuleSearchSeed}},
		{"search seeds of the next block", []Record{mod(func(r *Record) { r.Experiments[1].Seed0 = 201 })}, []string{RuleSearchSeed}},
		{"confirm above 50", []Record{mod(func(r *Record) { r.Experiments[2].Seed0, r.Experiments[2].Replicas = 48, 5 })}, []string{RuleConfirmSeed}},
		{"confirm on a search block", []Record{mod(func(r *Record) { r.Experiments[2].Seed0 = 101 })}, []string{RuleConfirmSeed, RuleConfirmSeed, RuleConfirmSeed}}, // outside 1..50, and sharing 101..103 with the screen and the explore
		{"confirm line above 50", []Record{mod(func(r *Record) { r.Confirm.Last = 51 })}, []string{RuleConfirmSeed}},
		{"confirm line reversed", []Record{mod(func(r *Record) { r.Confirm.First, r.Confirm.Last = 3, 1 })}, []string{RuleConfirmSeed}},
		// Search 40..49 is in 1..60 and off the block (two problems); the confirm 45..50 is
		// inside 1..50 and shares 45..49 with it.
		{"confirm shares a search seed", []Record{mod(func(r *Record) { r.Experiments[0].Seed0, r.Experiments[2].Seed0, r.Experiments[2].Replicas = 40, 45, 6 })}, []string{RuleConfirmSeed, RuleSearchSeed, RuleSearchSeed}},
		{"an experiment with no replicas has no seeds", []Record{mod(func(r *Record) { r.Experiments[0].Replicas = 0 })}, nil},
		{"a seed range that leaves int64 is outside the block", []Record{mod(func(r *Record) { r.Experiments[0].Seed0 = 1<<63 - 1; r.Experiments[0].Replicas = 5 })}, []string{RuleSearchSeed}},
		{"a seed range that leaves uint64 is outside 1..50", []Record{mod(func(r *Record) { r.Experiments[2].Seed0 = 1<<64 - 2; r.Experiments[2].Replicas = 5 })}, []string{RuleConfirmSeed}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, p := range CheckSequence(tt.rs, procKnown(), testMaxFactors) {
				if strings.HasPrefix(p.Rule, "block") || strings.HasSuffix(p.Rule, "_seed") {
					got = append(got, p.Rule)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("CheckSequence(%s) seed problems = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestProcess_Lineage verifies that the first record of a new lineage names its
// predecessor's lineage as parent, restarts at C1, carries no set, and carries no
// calibration overlay or warm-start table of an earlier lineage, and that records
// of different lineages are not comparable.
//
// Requirement: FR-CLB-053, FR-CLB-054
func TestProcess_Lineage(t *testing.T) {
	t.Parallel()
	old := goodSearch("C4a", 1, 1)
	old.Documents = append(slices.Clone(old.Documents), Document{"calibrated/c4a-1.toml", "aaaa000000000001"})
	old.Sets = []Setting{{Key: "World.CostFloor", Value: 3500}}
	next := func(f func(*Record)) Record {
		r := stageOnly("C1", 1, "stop")
		r.Lineage, r.Parent = "9999999999999999", old.Lineage
		r.Documents = []Document{{"base.toml", "3b9c11aa5e07f4d2"}, {"warm-one.toml", "ffff000000000001"}}
		f(&r)
		return r
	}
	tests := []struct {
		name string
		rs   []Record
		want []string
	}{
		{"a clean new lineage", []Record{old, next(func(*Record) {})}, nil},
		{"the parent is missing", []Record{old, next(func(r *Record) { r.Parent = "" })}, []string{RuleLineageParent}},
		{"the parent is another lineage", []Record{old, next(func(r *Record) { r.Parent = "1234567890abcdef" })}, []string{RuleLineageParent}},
		{"it does not restart at C1", []Record{old, next(func(r *Record) { r.Stage = "C2" })}, []string{RuleLineageRestart}},
		{"it carries a set", []Record{old, next(func(r *Record) { r.Sets = old.Sets })}, []string{RuleLineageSet}},
		{"it carries the old overlay", []Record{old, next(func(r *Record) { r.Documents = append(r.Documents, old.Documents[2]) })}, []string{RuleLineageDoc}},
		{"it carries the old warm table", []Record{old, next(func(r *Record) { r.Documents[1].Digest = old.Documents[1].Digest })}, []string{RuleLineageDoc}},
		{"a structure document may repeat", []Record{old, next(func(r *Record) { r.Documents[0].Digest = old.Documents[0].Digest })}, nil},
		{"a later record still may not carry it", []Record{old, next(func(*Record) {}), func() Record {
			r := stageOnly("C2", 1, "stop")
			r.Lineage = "9999999999999999"
			r.Documents = []Document{{"calibrated/c4a-1.toml", "aaaa000000000001"}}
			return r
		}()}, []string{RuleLineageDoc}},
		{"the same lineage needs no parent", []Record{old, func() Record { r := stageOnly("C4a", 2, "stop"); return r }()}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, p := range CheckSequence(tt.rs, procKnown(), testMaxFactors) {
				if strings.HasPrefix(p.Rule, "lineage") {
					got = append(got, p.Rule)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("CheckSequence(%s) lineage problems = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
	a, b := stageOnly("C1", 1, "stop"), stageOnly("C1", 1, "stop")
	if err := Comparable(a, b); err != nil {
		t.Errorf("Comparable(same lineage) = %v, want nil", err)
	}
	b.Lineage = "0000000000000001"
	if err := Comparable(a, b); !errors.Is(err, ErrLineage) {
		t.Errorf("Comparable(different lineage) = %v, want ErrLineage", err)
	}
	b.Lineage, b.Pack = a.Lineage, "two"
	if err := Comparable(a, b); !errors.Is(err, ErrLineage) {
		t.Errorf("Comparable(different pack) = %v, want ErrLineage", err)
	}
}

// TestProcess_PacksInterleaved verifies that records of two packs are checked
// each in its own order and that the problems come back sorted.
//
// Requirement: FR-CLB-035
func TestProcess_PacksInterleaved(t *testing.T) {
	t.Parallel()
	b2 := stageOnly("C2", 1, "stop")
	b2.Pack = "two"
	rs := []Record{stageOnly("C3", 1, "stop"), b2, stageOnly("C2", 1, "stop"), stageOnly("X", 1, "stop")}
	got := CheckSequence(rs, procKnown(), testMaxFactors)
	want := []Problem{
		{Record: 2, Rule: RuleStageOrder, Detail: "stage C2 follows C3 with no reentry decision"},
		{Record: 3, Rule: RuleStageName, Detail: `stage "X" is not one of C0 C1 C2 C3 C4a C4b C4c C5 C6 C7a C7b C8 C9`},
	}
	if !slices.Equal(got, want) {
		t.Errorf("CheckSequence(interleaved) = %+v, want %+v", got, want)
	}
}
