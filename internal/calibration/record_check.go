package calibration

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// The rule words of CheckRecord (spec-calibration.md 11.2).
const (
	RuleKnobUnknown     = "knob_unknown"
	RuleUnsearchedV1x   = "unsearched_v1x"
	RuleUnsearchedAtOff = "unsearched_at_off"
	RuleFactUnknown     = "fact_unknown"
	RuleFactClass       = "fact_class"
	RuleDetectorUnknown = "detector_unknown"
	RuleSetKey          = "set_key"
	RuleSetDormant      = "set_dormant"
	RuleSetOff          = "set_off"
	RuleFeasibleCount   = "feasible_count"
	RuleChosenRange     = "chosen_range"
	RuleScreenedDormant = "screened_dormant"
	RuleScreenedMeasure = "screened_measures"
	RuleScreenMeasure   = "screen_measure"
	RuleScreenSign      = "screen_sign"
	RuleClaim           = "claim"
	RuleHoldout         = "holdout"
	RuleInSample        = "in_sample"
	RuleStopFact        = "stop_fact"
)

const (
	// factPrefix is the optional prefix of a fact ID in a record (8.2).
	factPrefix = "STY-"
	// holdoutFirst and holdoutLast are the hold-out seeds of REF 8.2.
	holdoutFirst, holdoutLast = 51, 60
	// decRevert is the decision under which a dormant knob's set may stay (FR-CLB-062).
	decRevert = "revert"
	// The words of a claim (8.2) and of a stability point and verdict.
	claimStable, claimNot, claimOne, claimUndetermined = "stable", "not_stable", "one_neighbour", "undetermined"
	pointDefault, pointLower, pointUpper               = "default", "lower", "upper"
	setHoldout, stableWord, notStableWord              = "holdout", "STABLE", "NOT_STABLE"
	edgeNone                                           = "none"
	// The 3 stability lines of a stable claim and the 2 of a one-neighbour claim.
	linesStable, linesOneNeighbour = 3, 2
)

// recChecker holds one record under CheckRecord.
type recChecker struct {
	r      Record
	k      Known
	idx    index
	starts map[string]int
	out    []Problem
}

// CheckRecord checks one record against the registry and the facts of STY
// (spec-calibration.md 11.2, plan 2-21 D634) and returns the problems sorted by
// line and rule (FR-CLB-039, 060, 062, 063, 071, 100; FR-STY-092). The closed
// keywords are the parser's; this check resolves what they name. It is pure.
func CheckRecord(r Record, k Known) []Problem {
	c := recChecker{r: r, k: k, idx: newIndex(k), starts: lineStarts(r)}
	c.knobs()
	c.facts()
	c.sets()
	c.counts()
	c.screens()
	c.stability()
	c.claims()
	c.stop()
	slices.SortFunc(c.out, func(a, b Problem) int {
		return cmp.Or(cmp.Compare(a.Line, b.Line), cmp.Compare(a.Rule, b.Rule), cmp.Compare(a.Detail, b.Detail))
	})
	return c.out
}

// lineStarts gives the line of the first row of each keyword in the rendered
// record: the lines come in the order of 8.2, after the four header lines.
func lineStarts(r Record) map[string]int {
	counts := []struct {
		kw string
		n  int
	}{
		{"document", len(r.Documents)}, {"experiment", len(r.Experiments)}, {"factor", len(r.Factors)},
		{"unsearched", len(r.Unsearched)}, {"measure", len(r.Measures)}, {"criterion", len(r.Criteria)},
		{"screen", len(r.Screens)}, {"screened_in", oneIf(len(r.ScreenedIn) > 0)}, {"dormant", oneIf(len(r.Dormant) > 0)},
		{"finding", len(r.Findings)}, {"feasible", oneIf(r.Feasible.Has)}, {"chosen", oneIf(r.Pick.Has)},
		{"pareto", oneIf(len(r.Pareto) > 0)}, {"set", len(r.Sets)}, {"confirm", oneIf(r.Confirm.Has)},
		{"fact", len(r.Facts)}, {"detector", len(r.Detectors)}, {"stability", len(r.Stability)},
		{"claim", len(r.Claims)}, {"unmet", len(r.Unmet)},
	}
	starts := map[string]int{}
	line := 5 // the lines calibration, pack, commit and config come first
	for _, c := range counts {
		starts[c.kw] = line
		line += c.n
	}
	starts["decision"] = line
	starts["note"] = line + 1
	return starts
}

func oneIf(b bool) int {
	if b {
		return 1
	}
	return 0
}

// add records a problem at the i-th row of a keyword.
func (c *recChecker) add(rule, kw string, i int, format string, args ...any) {
	c.out = append(c.out, Problem{Line: c.starts[kw] + i, Rule: rule, Detail: fmt.Sprintf(format, args...)})
}

// factID strips the optional prefix of a fact ID.
func factID(s string) string { return strings.TrimPrefix(s, factPrefix) }

// knobs checks that every knob token resolves, and that `unsearched` names the
// right kind of row.
func (c *recChecker) knobs() {
	check := func(kw string, i int, id string) {
		if _, ok := c.k.Knobs[id]; !ok {
			c.add(RuleKnobUnknown, kw, i, "%s is not a registry ID", id)
		}
	}
	for i, u := range c.r.Unsearched {
		for _, id := range u.Knobs {
			check("unsearched", i, id)
			c.unsearched(i, u.Why, id)
		}
	}
	for i, s := range c.r.Screens {
		check("screen", i, s.Knob)
	}
	for _, id := range c.r.ScreenedIn {
		check("screened_in", 0, id)
	}
	for _, id := range c.r.Dormant {
		check("dormant", 0, id)
	}
	for i, f := range c.r.Findings {
		check("finding", i, f.Knob)
	}
	for i, s := range c.r.Stability {
		check("stability", i, s.Knob)
	}
	for i, cl := range c.r.Claims {
		check("claim", i, cl.Knob)
	}
	for i, f := range c.r.Factors {
		if _, ok := c.idx.find(f.Key); f.Key != "" && !ok { // a composite factor names no single key
			c.add(RuleKnobUnknown, "factor", i, "%s is not a registry key", f.Key)
		}
	}
}

// unsearched checks the reason of an `unsearched` line against the row: `v1x` names
// a row with the flag v1x, `at_off` a row with the flag z.
func (c *recChecker) unsearched(i int, why, id string) {
	info, ok := c.k.Knobs[id]
	switch {
	case !ok:
	case why == "v1x" && !info.V1x:
		c.add(RuleUnsearchedV1x, "unsearched", i, "%s has no flag v1x", id)
	case why == "at_off" && !info.Z:
		c.add(RuleUnsearchedAtOff, "unsearched", i, "%s has no flag z", id)
	}
}

// facts checks that every fact and detector is one of STY's and that a fact line
// has its class.
func (c *recChecker) facts() {
	for i, m := range c.r.Measures {
		c.measureFact(i, m)
	}
	for i, f := range c.r.Facts {
		class, ok := c.k.Facts[factID(f.ID)]
		switch {
		case !ok:
			c.add(RuleFactUnknown, "fact", i, "%s is not a fact of STY", f.ID)
		case class != f.Class:
			c.add(RuleFactClass, "fact", i, "%s is class %c in STY, %c in the record", f.ID, class, f.Class)
		}
	}
	c.references()
}

// references checks the facts and regimes that findings, unmet lines and
// detectors name.
func (c *recChecker) references() {
	for i, f := range c.r.Findings {
		if _, ok := c.k.Facts[factID(f.Fact)]; !ok {
			c.add(RuleFactUnknown, "finding", i, "%s is not a fact of STY", f.Fact)
		}
	}
	for i, u := range c.r.Unmet {
		if _, ok := c.k.Facts[factID(u.Fact)]; !ok {
			c.add(RuleFactUnknown, "unmet", i, "%s is not a fact of STY", u.Fact)
		}
	}
	for i, d := range c.r.Detectors {
		if !c.k.Detectors[factID(d.ID)] {
			c.add(RuleDetectorUnknown, "detector", i, "%s is not a regime of STY", d.ID)
		}
	}
}

// measureFact checks the fact or detector a `measure` line names.
func (c *recChecker) measureFact(i int, m Measure) {
	switch m.Kind {
	case "fact":
		if _, ok := c.k.Facts[factID(m.Fact)]; !ok {
			c.add(RuleFactUnknown, "measure", i, "%s is not a fact of STY", m.Fact)
		}
	case "detector":
		if !c.k.Detectors[factID(m.Fact)] {
			c.add(RuleDetectorUnknown, "measure", i, "%s is not a regime of STY", m.Fact)
		}
	}
}

// sets checks that a `set` names a K or D row, not a dormant one unless the
// decision reverts, and not the off value of a D row without a note.
func (c *recChecker) sets() {
	for i, s := range c.r.Sets {
		ref, ok := c.idx.find(s.Key)
		if !ok || (ref.Class != 'K' && ref.Class != 'D') {
			c.add(RuleSetKey, "set", i, "%s is not a K or D row", s.Key)
			continue
		}
		c.setRow(i, s, ref)
	}
}

// setRow checks one set of a K or D row.
func (c *recChecker) setRow(i int, s Setting, ref knobRef) {
	if slices.Contains(c.r.Dormant, ref.ID) && c.r.Decision != decRevert {
		c.add(RuleSetDormant, "set", i, "%s is dormant and the decision is %q, not %s", ref.ID, c.r.Decision, decRevert)
	}
	if ref.Class == 'D' && ref.Off.Has && ref.Off.Value == s.Value && len(c.r.Notes) == 0 {
		c.add(RuleSetOff, "set", i, "%s is set to its off value %d with no note", ref.ID, s.Value)
	}
}

// counts checks `feasible n of N`, the chosen index, the disjoint sets and the
// number of screened-in knobs against the measures.
func (c *recChecker) counts() {
	r := c.r
	if r.Feasible.Has && r.Feasible.N > r.Feasible.Of {
		c.add(RuleFeasibleCount, "feasible", 0, "%d feasible of %d", r.Feasible.N, r.Feasible.Of)
	}
	c.chosen()
	for _, id := range r.ScreenedIn {
		if slices.Contains(r.Dormant, id) {
			c.add(RuleScreenedDormant, "screened_in", 0, "%s is screened in and dormant", id)
		}
	}
	if len(r.ScreenedIn) > len(r.Measures) && !c.composite() {
		c.add(RuleScreenedMeasure, "screened_in", 0, "%d screened-in knobs and %d measures, and no composite factor", len(r.ScreenedIn), len(r.Measures))
	}
}

// chosen checks that the chosen index is below the number of points.
func (c *recChecker) chosen() {
	p := c.r.Pick
	if points, ok := c.points(); ok && p.Has && !p.None && p.Index >= points {
		c.add(RuleChosenRange, "chosen", 0, "point %d of %d", p.Index, points)
	}
}

// points is the number of points the chosen index is below: N of `feasible`, or
// the points of the explore.
func (c *recChecker) points() (int, bool) {
	if c.r.Feasible.Has {
		return c.r.Feasible.Of, true
	}
	for _, e := range c.r.Experiments {
		if e.Step == stepExplore {
			return e.Points, true
		}
	}
	return 0, false
}

// composite reports whether a factor is a composite: it carries several changes
// and so names no single registry key, written `-` (8.2).
func (c *recChecker) composite() bool {
	for _, f := range c.r.Factors {
		if f.Key == "" {
			return true
		}
	}
	return false
}

// screens checks that a screen whose measured sign differs from the registry's has
// a finding (FR-CLB-060, FR-STY-092).
func (c *recChecker) screens() {
	byName := map[string]Measure{}
	for _, m := range c.r.Measures {
		byName[m.Name] = m
	}
	for i, s := range c.r.Screens {
		m, ok := byName[s.Measure]
		if !ok {
			c.add(RuleScreenMeasure, "screen", i, "no measure line named %s", s.Measure)
			continue
		}
		if m.Token == "" {
			continue
		}
		token := factID(m.Fact) + "/" + m.Token
		want, has := c.k.Knobs[s.Knob].Moves[token]
		got, measured := measuredSign(s)
		if has && measured && isSigned(want.Sign) && want.Sign != got && !c.hasFinding(s.Knob, factID(m.Fact), m.Token) {
			c.add(RuleScreenSign, "screen", i, "%s on %s measured %c, registry %c, and no finding", s.Knob, token, got, want.Sign)
		}
	}
}

// isSigned reports whether a registry sign states a direction to compare: + - or 0.
func isSigned(b byte) bool { return b == '+' || b == '-' || b == '0' }

// measuredSign is the sign of the elasticity at the upper neighbour, or at the
// lower one when there is no upper; ok is false when the screen has neither.
func measuredSign(s Screen) (sign byte, ok bool) {
	e := s.ElUpper
	if !e.Has {
		e = s.ElLower
	}
	switch {
	case !e.Has:
		return 0, false
	case e.Value > 0:
		return '+', true
	case e.Value < 0:
		return '-', true
	}
	return '0', true
}

// hasFinding reports whether a finding names the knob, the fact and the token.
func (c *recChecker) hasFinding(knob, fact, token string) bool {
	for _, f := range c.r.Findings {
		if f.Knob == knob && factID(f.Fact) == fact && f.Measure == token {
			return true
		}
	}
	return false
}

// stability checks the seed set and the in_sample word of each stability line
// (FR-CLB-100).
func (c *recChecker) stability() {
	for i, s := range c.r.Stability {
		hold := s.Set == setHoldout
		if hold && (s.First != holdoutFirst || s.Last != holdoutLast) {
			c.add(RuleHoldout, "stability", i, "a hold-out line names seeds %d..%d, want %d..%d", s.First, s.Last, holdoutFirst, holdoutLast)
		}
		if hold && s.InSample {
			c.add(RuleInSample, "stability", i, "a hold-out line says in_sample yes")
		}
		if !hold && !s.InSample {
			c.add(RuleInSample, "stability", i, "a %s line says in_sample no", s.Set)
		}
	}
}

// claims checks each claim against the stability lines of its knob and pack
// (FR-CLB-039): stable has the three points, all STABLE and no edge;
// one_neighbour has two lines and one edge; undetermined for a default-only
// record has exactly one line.
func (c *recChecker) claims() {
	for i, cl := range c.r.Claims {
		var lines []Stability // the gate lines: the hold-out repeats the points and is not counted
		var holdout []Stability
		for _, s := range c.r.Stability {
			switch {
			case s.Knob != cl.Knob || s.Pack != cl.Pack:
			case s.Set == setHoldout:
				holdout = append(holdout, s)
			default:
				lines = append(lines, s)
			}
		}
		if msg := claimProblem(cl.Word, lines, holdout); msg != "" {
			c.add(RuleClaim, "claim", i, "claim %s %s %s: %s", cl.Knob, cl.Pack, cl.Word, msg)
		}
	}
}

// claimProblem says what is wrong with a claim given its stability lines, or "".
func claimProblem(word string, lines, holdout []Stability) string {
	if len(lines) == 0 && len(holdout) == 0 {
		return "no stability line"
	}
	t := tallyOf(lines)
	switch word {
	case claimStable:
		return stableProblem(lines, holdout, t)
	case claimNot:
		return notStableProblem(tallyOf(append(slices.Clone(lines), holdout...)))
	case claimOne:
		return oneNeighbourProblem(lines, t)
	case claimUndetermined:
		return undeterminedProblem(lines, t)
	}
	return ""
}

// tallyOf counts the points, the edges and the failing lines.
func tallyOf(lines []Stability) tally {
	t := tally{points: map[string]int{}}
	for _, s := range lines {
		t.points[s.Point]++
		t.edges += oneIf(s.Edge != edgeNone)
		t.notStable += oneIf(s.CBP == notStableWord || s.CNC == notStableWord || s.CWS == notStableWord)
	}
	return t
}

// notStableProblem checks that a not_stable claim has a failing verdict.
func notStableProblem(t tally) string {
	if t.notStable == 0 {
		return "no verdict is NOT_STABLE"
	}
	return ""
}

// oneNeighbourProblem checks the two lines and the one edge of a one_neighbour claim.
func oneNeighbourProblem(lines []Stability, t tally) string {
	if len(lines) != linesOneNeighbour || t.edges != 1 || t.notStable > 0 {
		return fmt.Sprintf("%d lines, %d edges and %d NOT_STABLE, want 2, 1 and 0", len(lines), t.edges, t.notStable)
	}
	return ""
}

// undeterminedProblem checks that an undetermined claim has a line at the default,
// and exactly one line when the record is default-only (spec-calibration.md 11.2,
// FR-CLB-039).
func undeterminedProblem(lines []Stability, t tally) string {
	if t.points[pointDefault] == 0 {
		return "an undetermined claim has no line at the default"
	}
	if t.points[pointDefault] == len(lines) && len(lines) != 1 {
		return fmt.Sprintf("a default-only record has %d lines, want 1", len(lines))
	}
	return ""
}

// tally counts the points, the edges and the failing lines of a claim's stability lines.
type tally struct {
	points           map[string]int
	edges, notStable int
}

// stableProblem checks the three lines of a stable claim.
func stableProblem(lines, holdout []Stability, t tally) string {
	if len(lines) != linesStable || t.points[pointDefault] != 1 || t.points[pointLower] != 1 || t.points[pointUpper] != 1 {
		return "want one default, one lower and one upper line"
	}
	if t.edges > 0 {
		return "a line has an edge"
	}
	for _, s := range append(slices.Clone(lines), holdout...) {
		if s.CBP != stableWord || s.CNC != stableWord || s.CWS != stableWord {
			return "a verdict is not STABLE"
		}
	}
	return ""
}

// stop checks that a stage that ends `stop` has no failed fact.
func (c *recChecker) stop() {
	if c.r.Decision != decStop {
		return
	}
	for i, f := range c.r.Facts {
		if f.Result == resFail {
			c.add(RuleStopFact, "fact", i, "%s fails and the decision is stop", f.ID)
		}
	}
}
