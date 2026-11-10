package calibration

import (
	"slices"
	"strings"
	"testing"
)

// checkKnown is the registry the minimal record and the record of 19.5 are
// checked against: rows of the classes the rules tell apart.
func checkKnown() Known {
	return Known{
		Knobs: map[string]KnobInfo{
			"K-BEL07": {Class: 'K', Key: "World.BeliefStep", Stage: "C4", Owner: "BEL"},
			"K-BEL12": {Class: 'K', Key: "World.CostFloor", Stage: "C4", Owner: "BEL", Moves: map[string]MoveSign{"P02/Pinned.Share": {'+', 'S'}}},
			"K-BEL14": {Class: 'K', Key: "World.BeliefLow", Stage: "C4", Owner: "BEL"},
			"K-BEL15": {Class: 'K', Key: "World.BeliefHigh", Stage: "C4", Owner: "BEL"},
			"K-MON01": {Class: 'K', Key: "World.Faucet", Stage: "C8", Owner: "MON", Z: true},
			"K-OLD01": {Class: 'K', Key: "World.Legacy", Stage: "C8", Owner: "MON", V1x: true},
			"K-SPN05": {Class: 'D', Key: "Spawner.Cooldown", Stage: "C4", Owner: "SPN", Off: Off{Has: true, Value: 0}},
			"K-AGT01": {Class: 'K', Key: "Role.TargetRatio", Stage: "C4", Owner: "AGT"},
			"S-PRO01": {Class: 'S', Key: "Role.ExpectedWorkTicks", Stage: "C2", Owner: "PRO"},
		},
		Facts:     map[string]byte{"P02": 'B', "P04": 'B', "S01": 'X'},
		Detectors: map[string]bool{"R5a": true},
		Paths:     map[string]string{"role.*.target_ratio": "Role.TargetRatio"},
	}
}

func specKnown() Known { return checkKnown() }

func minimalRecord(t *testing.T) Record {
	t.Helper()
	r, err := Parse(readFixture(t, "valid/minimal.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestCheckRecord_Valid verifies that the minimal record has no problem.
//
// Requirement: FR-CLB-039, FR-CLB-060, FR-CLB-062, FR-CLB-063, FR-CLB-071, FR-CLB-100
func TestCheckRecord_Valid(t *testing.T) {
	t.Parallel()
	if ps := CheckRecord(minimalRecord(t), checkKnown()); len(ps) != 0 {
		t.Errorf("CheckRecord(minimal) = %+v, want none", ps)
	}
}

// stab is a stability line of K-BEL12 on pack one.
func stab(point, set string, first, last uint64, inSample bool, v string, edge string) Stability {
	return Stability{Knob: "K-BEL12", Pack: "one", Point: point, Value: 7000, Set: set, First: first, Last: last, InSample: inSample, CBP: v, CNC: v, CWS: v, Edge: edge}
}

// TestCheckRecord_Rules verifies each rule of 11.2 on the minimal record changed in
// one way: the rule that fails, and the keyword of the line it points at.
//
// Requirement: FR-CLB-039, FR-CLB-060, FR-CLB-062, FR-CLB-063, FR-CLB-071, FR-CLB-100, FR-STY-092
func TestCheckRecord_Rules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*Record)
		rule   string
		kw     string
	}{
		{"unsearched knob unknown", func(r *Record) { r.Unsearched[0].Knobs = []string{"K-NOPE"} }, RuleKnobUnknown, "unsearched"},
		{"screen knob unknown", func(r *Record) { r.Screens[0].Knob = "K-NOPE" }, RuleKnobUnknown, "screen"},
		{"screened_in knob unknown", func(r *Record) { r.ScreenedIn[0] = "K-NOPE" }, RuleKnobUnknown, "screened_in"},
		{"dormant knob unknown", func(r *Record) { r.Dormant[0] = "K-NOPE" }, RuleKnobUnknown, "dormant"},
		{"finding knob unknown", func(r *Record) { r.Findings[0].Knob = "K-NOPE" }, RuleKnobUnknown, "finding"},
		{"stability knob unknown", func(r *Record) { r.Stability[1].Knob = "K-NOPE" }, RuleKnobUnknown, "stability"},
		{"undetermined claim without a default line", func(r *Record) { r.Stability[0].Point = pointLower }, RuleClaim, "claim"},
		{"claim knob unknown", func(r *Record) {
			r.Claims[0].Knob = "K-NOPE"
			r.Stability[0].Knob, r.Stability[1].Knob = "K-NOPE", "K-NOPE"
		}, RuleKnobUnknown, "stability"},
		{"v1x names a row without the flag", func(r *Record) { r.Unsearched[1].Knobs = []string{"K-BEL07"} }, RuleUnsearchedV1x, "unsearched"},
		{"at_off names a row without z", func(r *Record) { r.Unsearched[0].Knobs = []string{"K-BEL07"} }, RuleUnsearchedAtOff, "unsearched"},
		{"fact unknown", func(r *Record) { r.Facts[0].ID = "STY-P99" }, RuleFactUnknown, "fact"},
		{"fact of another class", func(r *Record) { r.Facts[0].Class = 'X' }, RuleFactClass, "fact"},
		{"measure fact unknown", func(r *Record) { r.Measures[0].Fact = "P99" }, RuleFactUnknown, "measure"},
		{"measure detector unknown", func(r *Record) { r.Measures[2].Fact = "R9" }, RuleDetectorUnknown, "measure"},
		{"finding fact unknown", func(r *Record) { r.Findings[0].Fact = "P99" }, RuleFactUnknown, "finding"},
		{"unmet fact unknown", func(r *Record) { r.Unmet[0].Fact = "P99" }, RuleFactUnknown, "unmet"},
		{"detector unknown", func(r *Record) { r.Detectors[0].ID = "R9" }, RuleDetectorUnknown, "detector"},
		{"set of an S row", func(r *Record) { r.Sets[0].Key = "Role.ExpectedWorkTicks" }, RuleSetKey, "set"},
		{"set of an unknown key", func(r *Record) { r.Sets[0].Key = "World.Nope" }, RuleSetKey, "set"},
		{"set of a dormant knob", func(r *Record) { r.Dormant = []string{"K-BEL12", "K-BEL15"}; r.ScreenedIn = []string{"K-BEL07"} }, RuleSetDormant, "set"},
		{"set of a D knob at its off value", func(r *Record) {
			r.Sets = []Setting{{"Spawner.Cooldown", 0}}
			r.Notes = nil
		}, RuleSetOff, "set"},
		{"feasible n above N", func(r *Record) { r.Feasible = Feasible{Has: true, N: 21, Of: 20} }, RuleFeasibleCount, "feasible"},
		{"chosen above the points", func(r *Record) { r.Pick.Index = 20 }, RuleChosenRange, "chosen"},
		{"screened_in and dormant overlap", func(r *Record) { r.Dormant = []string{"K-BEL07"} }, RuleScreenedDormant, "screened_in"},
		{"more screened_in than measures", func(r *Record) {
			r.ScreenedIn = []string{"K-BEL07", "K-BEL12", "K-BEL14", "K-BEL15"}
			r.Dormant = nil
			r.Factors = r.Factors[:1] // no composite
		}, RuleScreenedMeasure, "screened_in"},
		{"screen of an unnamed measure", func(r *Record) { r.Screens[0].Measure = "nope" }, RuleScreenMeasure, "screen"},
		{"screen sign differs and no finding", func(r *Record) {
			r.Screens[0].ElUpper = Opt{Has: true, Value: -5}
			r.Findings = nil
		}, RuleScreenSign, "screen"},
		{"stop with a failed fact", func(r *Record) { r.Facts[1].Result = "fail" }, RuleStopFact, "fact"},
		{"hold-out on the wrong seeds", func(r *Record) { r.Stability[1].First = 50 }, RuleHoldout, "stability"},
		{"hold-out in sample", func(r *Record) { r.Stability[1].InSample = true }, RuleInSample, "stability"},
		{"gate not in sample", func(r *Record) { r.Stability[0].InSample = false }, RuleInSample, "stability"},
		{"claim with no stability line", func(r *Record) { r.Claims[0].Pack = "two" }, RuleClaim, "claim"},
		{"stable with two lines", func(r *Record) { r.Claims[0].Word = "stable" }, RuleClaim, "claim"},
		{"not_stable with no failing verdict", func(r *Record) { r.Claims[0].Word = "not_stable" }, RuleClaim, "claim"},
		{"one_neighbour with no edge", func(r *Record) { r.Claims[0].Word = "one_neighbour" }, RuleClaim, "claim"},
		{"default-only undetermined with two lines", func(r *Record) {
			r.Stability[1].Point = "default"
			r.Stability[1].Set, r.Stability[1].First, r.Stability[1].Last, r.Stability[1].InSample = "gate", 1, 10, true
		}, RuleClaim, "claim"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := minimalRecord(t)
			tt.mutate(&r)
			ps := CheckRecord(r, checkKnown())
			if got := slices.Compact(rules(ps)); !slices.Equal(got, []string{tt.rule}) {
				t.Fatalf("CheckRecord(%s) = %+v, want only rule %s", tt.name, ps, tt.rule)
			}
			// The line it points at is a line of the rendered record that starts with the keyword.
			text, err := Render(r)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(string(text), "\n")
			if ps[0].Line < 1 || ps[0].Line > len(lines) || !strings.HasPrefix(lines[ps[0].Line-1], tt.kw+" ") {
				t.Errorf("CheckRecord(%s) points at line %d, want a %q line of:\n%s", tt.name, ps[0].Line, tt.kw, text)
			}
		})
	}
}

// TestCheckRecord_Accepts verifies the cases that look like a failure and are not:
// a path that resolves to a K row, a D row set to a value off its off value or to
// it with a note, a dormant knob set under a revert, a sign that differs with a
// finding, the registry signs that state no direction, and composite factors.
//
// Requirement: FR-CLB-060, FR-CLB-062, FR-CLB-063, FR-CLB-071
func TestCheckRecord_Accepts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*Record)
	}{
		{"a path of a table keyed by an ID", func(r *Record) { r.Sets = []Setting{{"role.baker.target_ratio", 5}} }},
		{"a D row off its off value", func(r *Record) { r.Sets = []Setting{{"Spawner.Cooldown", 4}}; r.Notes = nil }},
		{"a D row at its off value with a note", func(r *Record) { r.Sets = []Setting{{"Spawner.Cooldown", 0}} }},
		{"a dormant knob under a revert", func(r *Record) {
			r.Dormant = []string{"K-BEL12", "K-BEL15"}
			r.ScreenedIn = []string{"K-BEL07"}
			r.Decision = "revert"
		}},
		{"a sign that differs with a finding", func(r *Record) { r.Screens[0].ElUpper = Opt{Has: true, Value: -5} }},
		{"a sign read at the lower neighbour", func(r *Record) { r.Screens[0].ElUpper = Opt{}; r.Screens[0].ElLower = Opt{Has: true, Value: 3} }},
		{"a screen with no elasticity", func(r *Record) { r.Screens[0].ElUpper, r.Screens[0].ElLower = Opt{}, Opt{} }},
		{"a measure with no token", func(r *Record) { r.Measures[0].Token = "" }},
		{"more screened_in than measures with a composite", func(r *Record) {
			r.ScreenedIn = []string{"K-BEL07", "K-BEL12", "K-BEL14", "K-BEL15"}
			r.Dormant = nil
		}},
		{"a claim word the parser owns", func(r *Record) { r.Claims[0].Word = "maybe" }},
		{"a chosen none", func(r *Record) { r.Pick = Pick{Has: true, None: true} }},
		{"a chosen index checked against the explore", func(r *Record) { r.Feasible = Feasible{}; r.Pick.Index = 19 }},
		{"a chosen index with no point count", func(r *Record) { r.Feasible = Feasible{}; r.Experiments = nil; r.Pick.Index = 99 }},
		{"a stable claim", func(r *Record) {
			r.Stability = []Stability{stab("default", "gate", 1, 10, true, "STABLE", "none"), stab("lower", "gate", 1, 10, true, "STABLE", "none"), stab("upper", "gate", 1, 10, true, "STABLE", "none")}
			r.Claims[0].Word = "stable"
		}},
		{"a one_neighbour claim", func(r *Record) {
			r.Stability = []Stability{stab("default", "gate", 1, 10, true, "STABLE", "above:bound"), stab("lower", "gate", 1, 10, true, "STABLE", "none")}
			r.Claims[0].Word = "one_neighbour"
		}},
		{"a not_stable claim", func(r *Record) {
			r.Stability = []Stability{stab("default", "gate", 1, 10, true, "NOT_STABLE", "none")}
			r.Claims[0].Word = "not_stable"
		}},
		{"a default-only undetermined claim", func(r *Record) {
			r.Stability = []Stability{stab("default", "gate", 1, 10, true, "UNDETERMINED", "none")}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := minimalRecord(t)
			tt.mutate(&r)
			if ps := CheckRecord(r, checkKnown()); len(ps) != 0 {
				t.Errorf("CheckRecord(%s) = %+v, want none", tt.name, ps)
			}
		})
	}
}

// TestCheckRecord_StabilityAndClaim verifies the shape of a stable claim: three
// lines, all STABLE and no edge, and that a claim is judged against the lines of
// its own knob and pack only.
//
// Requirement: FR-CLB-039, FR-CLB-100
func TestCheckRecord_StabilityAndClaim(t *testing.T) {
	t.Parallel()
	three := func(edge, verdict string) []Stability {
		return []Stability{stab("default", "gate", 1, 10, true, "STABLE", "none"), stab("lower", "gate", 1, 10, true, "STABLE", "none"), stab("upper", "gate", 1, 10, true, verdict, edge)}
	}
	tests := []struct {
		name  string
		lines []Stability
		word  string
		bad   bool
	}{
		{"stable ok", three("none", "STABLE"), "stable", false},
		{"stable with an edge", three("above:bound", "STABLE"), "stable", true},
		{"stable with a verdict that is not STABLE", three("none", "UNDETERMINED"), "stable", true},
		{"stable with a failing verdict", three("none", "NOT_STABLE"), "stable", true},
		{"stable with a repeated point", []Stability{stab("default", "gate", 1, 10, true, "STABLE", "none"), stab("default", "gate", 1, 10, true, "STABLE", "none"), stab("upper", "gate", 1, 10, true, "STABLE", "none")}, "stable", true},
		{"one_neighbour with three lines", three("above:bound", "STABLE"), "one_neighbour", true},
		{"one_neighbour with a failing verdict", []Stability{stab("default", "gate", 1, 10, true, "NOT_STABLE", "below:rule"), stab("upper", "gate", 1, 10, true, "STABLE", "none")}, "one_neighbour", true},
		{"one_neighbour with two edges", []Stability{stab("default", "gate", 1, 10, true, "STABLE", "below:rule"), stab("upper", "gate", 1, 10, true, "STABLE", "above:bound")}, "one_neighbour", true},
		{"undetermined default-only with one line", three("none", "STABLE")[:1], "undetermined", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := minimalRecord(t)
			other := stab("default", "gate", 1, 10, true, "NOT_STABLE", "above:bound")
			other.Knob = "K-BEL07" // another knob's line must not count
			r.Stability = append(slices.Clone(tt.lines), other)
			r.Claims = []Claim{{"K-BEL12", "one", tt.word}}
			got := rules(CheckRecord(r, checkKnown()))
			if tt.bad != slices.Equal(got, []string{RuleClaim}) || (!tt.bad && len(got) != 0) {
				t.Errorf("CheckRecord(%s) rules = %v, want claim problem: %v", tt.name, got, tt.bad)
			}
		})
	}
}

// TestCheckRecord_SignsThatStateNoDirection verifies that a registry sign of ?, ~ or
// a missing token is never compared with the measured sign, and that 0 is: a
// measured effect where the row expects none is a finding.
//
// Requirement: FR-CLB-060, FR-STY-092
func TestCheckRecord_SignsThatStateNoDirection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		sign byte
		el   int64
		bad  bool
	}{
		{'?', -5, false}, {'~', -5, false}, {'+', 5, false}, {'+', -5, true}, {'-', 5, true}, {'-', -5, false},
		{'0', 0, false}, {'0', 5, true}, {'+', 0, true},
	}
	for _, tt := range tests {
		r := minimalRecord(t)
		r.Screens[0].ElUpper = Opt{Has: true, Value: tt.el}
		r.Findings = nil
		k := checkKnown()
		info := k.Knobs["K-BEL12"]
		info.Moves = map[string]MoveSign{"P02/Pinned.Share": {tt.sign, 'S'}}
		k.Knobs["K-BEL12"] = info
		if got := slices.Equal(rules(CheckRecord(r, k)), []string{RuleScreenSign}); got != tt.bad {
			t.Errorf("CheckRecord(registry %c, elasticity %d) screen_sign problem = %v, want %v", tt.sign, tt.el, got, tt.bad)
		}
	}
}
