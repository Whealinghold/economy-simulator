package verdict_test

import (
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// held returns every conjunct a check needs as Held.
func held(check string) []verdict.Conjunct {
	var out []verdict.Conjunct
	for _, n := range verdict.Required(check) {
		out = append(out, verdict.Conjunct{Name: n, Outcome: verdict.Held})
	}
	return out
}

// TestCheck_ThreeResults verifies the three results of a check (STY 7.2): the
// result is STABLE only when every conjunct is asserted and held, NOT STABLE
// lists the conjunct and the seed, UNDETERMINED covers an unset key, an
// unknown detector, a blind-spot field outside its bound and a missing
// overlay or conjunct; the state of today, with every key unset, is
// UNDETERMINED for all three checks.
//
// Requirement: FR-STY-095, FR-STY-007
//
// Guard: a check that said STABLE on a conjunct it did not assert would let a
// default be called by a word the procedure has not earned.
func TestCheck_ThreeResults(t *testing.T) {
	t.Parallel()
	for _, check := range verdict.Checks() {
		if got := verdict.Check(check, held(check)); got.Result != verdict.Stable || len(got.Failed)+len(got.Open) != 0 {
			t.Errorf("Check(%s, all held) = %+v, want STABLE with nothing listed", check, got)
		}
		// Today: every B key unset, every detector unknown.
		var today []verdict.Conjunct
		for _, n := range verdict.Required(check) {
			today = append(today, verdict.Conjunct{Name: n, Reason: "band unset"})
		}
		if got := verdict.Check(check, today); got.Result != verdict.Undetermined || len(got.Open) != len(today) {
			t.Errorf("Check(%s, all open) = %+v, want UNDETERMINED with %d open", check, got, len(today))
		}
		// One failed conjunct, one open: NOT STABLE names the failed one and its seed.
		mixed := held(check)
		mixed[0] = verdict.Conjunct{Name: mixed[0].Name, Outcome: verdict.Failed, Seed: 7}
		mixed[1] = verdict.Conjunct{Name: mixed[1].Name, Reason: "unknown (windows)"}
		got := verdict.Check(check, mixed)
		if got.Result != verdict.NotStable || len(got.Failed) != 1 || got.Failed[0].Seed != 7 || got.Failed[0].Name != mixed[0].Name || len(got.Open) != 1 {
			t.Errorf("Check(%s, one failed one open) = %+v, want NOT STABLE listing %s seed 7", check, got, mixed[0].Name)
		}
		// One conjunct open only.
		open := held(check)
		open[len(open)-1].Outcome, open[len(open)-1].Reason = verdict.Open, "overlay missing"
		if got := verdict.Check(check, open); got.Result != verdict.Undetermined || len(got.Open) != 1 || got.Open[0].Reason != "overlay missing" {
			t.Errorf("Check(%s, one open) = %+v, want UNDETERMINED with the reason", check, got)
		}
		// A required conjunct not given is open, "not run".
		if got := verdict.Check(check, held(check)[1:]); got.Result != verdict.Undetermined || len(got.Open) != 1 || got.Open[0].Reason != "not run" {
			t.Errorf("Check(%s, one conjunct missing) = %+v, want UNDETERMINED, not run", check, got)
		}
		// Extra conjuncts (the lane part of C-NC) count.
		extra := append(held(check), verdict.Conjunct{Name: "STY-P06", Outcome: verdict.Failed, Seed: 3})
		if got := verdict.Check(check, extra); got.Result != verdict.NotStable {
			t.Errorf("Check(%s, extra failed) = %+v, want NOT STABLE", check, got)
		}
	}
	for _, tc := range []struct{ name, check string }{{"no conjunct", verdict.CheckBP}, {"unknown check", "C-XX"}} {
		in := held(verdict.CheckBP)
		if tc.name == "no conjunct" {
			in = nil
		}
		if got := verdict.Check(tc.check, in); got.Result != verdict.Undetermined {
			t.Errorf("Check(%s) = %+v, want UNDETERMINED", tc.name, got)
		}
	}
	if got := verdict.Required("C-XX"); got != nil {
		t.Errorf("Required(C-XX) = %v, want nil", got)
	}
}

// TestCheck_OfVerdict verifies that a detector verdict becomes a conjunct: none
// held, detected failed at the seed, unknown open with its reason.
//
// Requirement: FR-STY-095, FR-STY-060
func TestCheck_OfVerdict(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		v    verdict.Verdict
		want verdict.Conjunct
	}{
		{verdict.Verdict{Detector: verdict.R5a, Class: verdict.None}, verdict.Conjunct{Name: "R5a", Outcome: verdict.Held, Seed: 4}},
		{verdict.Verdict{Detector: verdict.R6, Class: verdict.Detected}, verdict.Conjunct{Name: "R6", Outcome: verdict.Failed, Seed: 4}},
		{verdict.Verdict{Detector: verdict.R7, Class: verdict.Unknown, Reason: verdict.ReasonThresholdUnset}, verdict.Conjunct{Name: "R7", Seed: 4, Reason: "unknown (threshold unset)"}},
	} {
		if got := verdict.OfVerdict(tc.v, 4); got != tc.want {
			t.Errorf("OfVerdict(%v) = %+v, want %+v", tc.v, got, tc.want)
		}
	}
}

// TestResult_Words verifies that the three words are spelled as the record
// spells them, and that they are the only spellings.
//
// Requirement: FR-STY-095, FR-STY-096
func TestResult_Words(t *testing.T) {
	t.Parallel()
	for r, want := range map[verdict.Result]string{verdict.Stable: "STABLE", verdict.NotStable: "NOT STABLE", verdict.Undetermined: "UNDETERMINED"} {
		if got := r.String(); got != want {
			t.Errorf("Result(%d).String() = %q, want %q", r, got, want)
		}
	}
}

// TestAcceptanceSet verifies that X and O facts are asserted whether or not a
// key exists, B and P facts only when every key has a value, and R facts never;
// and that the five classes are the closed set.
//
// Requirement: FR-STY-011
//
// Guard: a banded fact asserted on an unset key would fail the build on a
// prior nobody reviewed (REF 7.1.5).
func TestAcceptanceSet(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		c    verdict.FactClass
		keys bool
		want bool
	}{
		{verdict.ClassX, false, true}, {verdict.ClassX, true, true},
		{verdict.ClassO, false, true}, {verdict.ClassO, true, true},
		{verdict.ClassB, false, false}, {verdict.ClassB, true, true},
		{verdict.ClassP, false, false}, {verdict.ClassP, true, true},
		{verdict.ClassR, false, false}, {verdict.ClassR, true, false},
		{"Z", true, false},
	} {
		if got := verdict.Asserted(tc.c, tc.keys); got != tc.want {
			t.Errorf("Asserted(%s, keys set %v) = %v, want %v", tc.c, tc.keys, got, tc.want)
		}
	}
	for _, c := range []verdict.FactClass{"X", "O", "B", "R", "P"} {
		if !c.Valid() {
			t.Errorf("FactClass(%q).Valid() = false, want true", c)
		}
	}
	if verdict.FactClass("Z").Valid() || verdict.FactClass("").Valid() {
		t.Errorf("an unknown class is valid, want the closed set X, O, B, R, P")
	}
}

// sampleRecord is the record of STY 8.5's example, with a not-applicable row, a
// per-region row and an observed set band.
func sampleRecord() verdict.Record {
	return verdict.Record{
		Pack: "one", Seeds: "1..10", Commit: "abc123", Config: "9f8e",
		Scenarios: []verdict.ScenarioRecord{{
			Scenario: "S01",
			Facts: []verdict.FactRow{
				{ID: "STY-S01", Class: verdict.ClassO, Statistic: "sign", N: 10},
				{ID: "STY-S01", Class: verdict.ClassB, Statistic: "s01.recover_ticks", N: 10, Outside: []int64{7}, HasPrior: true, Prior: 90, HasValues: true, Min: 42, Median: 58, Max: 117},
				{ID: "STY-W07", Class: verdict.ClassR, Statistic: "employment", NotApplicable: true, Reason: "module employment absent"},
			},
			Regimes: []verdict.Verdict{
				{Detector: verdict.R9, Class: verdict.Unknown, Reason: verdict.ReasonWindows},
				{Detector: verdict.R5a, Class: verdict.None}, {Detector: verdict.R5b, Class: verdict.None}, {Detector: verdict.R6, Class: verdict.None},
			},
		}},
	}
}

// TestRenderVerdict_Bytes verifies the rendering: the example of STY 8.5 (the
// rows and words; the spec's column padding is typographic and is not kept),
// the seeds outside listed, `band unset prior 90`, `not applicable` rows, the
// order by scenario, fact, region and role whatever the order of the input,
// and no float, no map order and no wall-clock text.
//
// Requirement: FR-STY-085, FR-STY-009, FR-STY-011, NFR-STY-002
//
// Guard: the record is a golden file; a rendering that depended on input order
// would move it with no change of the code.
func TestRenderVerdict_Bytes(t *testing.T) {
	t.Parallel()
	want := `pack one  scenario S01  seeds 1..10  commit abc123  config 9f8e
fact STY-S01  class B  s01.recover_ticks  band unset  prior 90
                       values min 42 median 58 max 117  outside-prior 1 (seed 7)
fact STY-S01  class O  sign  ok 10/10
fact STY-W07  class R  employment  not applicable (module employment absent)
regime R5a none   R5b none   R6 none   R9 unknown (windows)
`
	if got := verdict.RenderVerdict(sampleRecord()); got != want {
		t.Errorf("RenderVerdict(sample) =\n%s\nwant\n%s", got, want)
	}
	// The order of the input does not matter.
	rng := rand.New(rand.NewPCG(1, 2))
	for range 20 {
		r := sampleRecord()
		rng.Shuffle(len(r.Scenarios[0].Facts), func(i, j int) {
			r.Scenarios[0].Facts[i], r.Scenarios[0].Facts[j] = r.Scenarios[0].Facts[j], r.Scenarios[0].Facts[i]
		})
		rng.Shuffle(len(r.Scenarios[0].Regimes), func(i, j int) {
			r.Scenarios[0].Regimes[i], r.Scenarios[0].Regimes[j] = r.Scenarios[0].Regimes[j], r.Scenarios[0].Regimes[i]
		})
		if got := verdict.RenderVerdict(r); got != want {
			t.Fatalf("RenderVerdict(shuffled) =\n%s\nwant the same bytes", got)
		}
	}
	if floatRe.MatchString(want) || strings.Contains(want, "T00") || strings.Contains(want, "UTC") {
		t.Errorf("the record holds a float or a clock: %q", want)
	}
}

var floatRe = regexp.MustCompile(`[0-9]\.[0-9]`)

// TestRenderVerdict_Rows verifies the other row forms: a set band with its
// outside seeds, an exact fact with failures, rows per region and role in
// ascending order, scenarios in ascending order, a detected verdict, and a row
// with neither band nor prior.
//
// Requirement: FR-STY-085, FR-STY-084
func TestRenderVerdict_Rows(t *testing.T) {
	t.Parallel()
	r := verdict.Record{Pack: "second", Seeds: "1..4", Commit: "c", Config: "h", Scenarios: []verdict.ScenarioRecord{
		{Scenario: "X4", Facts: []verdict.FactRow{
			{ID: "STY-P02", Class: verdict.ClassB, Statistic: "x4.pinned_share_bp", Region: "south", Role: "cook", N: 4, BandSet: true, Band: 1000, Outside: []int64{2, 3}, HasValues: true, Min: 5, Median: 9, Max: 1500},
			{ID: "STY-P02", Class: verdict.ClassB, Statistic: "x4.pinned_share_bp", Region: "north", Role: "grower", N: 4, BandSet: true, Band: 1000, HasValues: true, Min: 1, Median: 2, Max: 3},
			{ID: "STY-P01", Class: verdict.ClassX, Statistic: "inside bands", N: 4, Outside: []int64{1}},
			{ID: "STY-D02", Class: verdict.ClassR, Statistic: "gini", HasValues: true, Min: 1, Median: 2, Max: 3},
			{ID: "STY-P08", Class: verdict.ClassP, Statistic: "x4.trade_share_bp", N: 4, HasValues: true, Min: 1, Median: 1, Max: 1},
			{ID: "STY-P08", Class: verdict.ClassR, Statistic: "x4.trade_share_bp", N: 4},
			{ID: "STY-P08", Class: verdict.ClassP, Statistic: "mean_volume", N: 4, BandSet: true, Band: 1},
		}, Regimes: []verdict.Verdict{{Detector: verdict.R3, Class: verdict.Detected, Onset: 700, Value: 2100, Threshold: 2000}}},
		{Scenario: "S01"},
	}}
	want := `pack second  scenario S01  seeds 1..4  commit c  config h
pack second  scenario X4  seeds 1..4  commit c  config h
fact STY-D02  class R  gini  report
                       values min 1 median 2 max 3
fact STY-P01  class X  inside bands  ok 3/4
fact STY-P02  class B  x4.pinned_share_bp  region north  role grower  band 1000  ok 4/4
                       values min 1 median 2 max 3  outside 0
fact STY-P02  class B  x4.pinned_share_bp  region south  role cook  band 1000  ok 2/4
                       values min 5 median 9 max 1500  outside 2 (seeds 2, 3)
fact STY-P08  class P  mean_volume  band 1  ok 4/4
fact STY-P08  class P  x4.trade_share_bp  band unset  prior none
                       values min 1 median 1 max 1
fact STY-P08  class R  x4.trade_share_bp  report
regime R3 detected onset 700 value 2100 threshold 2000
`
	if got := verdict.RenderVerdict(r); got != want {
		t.Errorf("RenderVerdict(rows) =\n%s\nwant\n%s", got, want)
	}
}

// TestRenderStability_Bytes verifies the stability record of STY 7.3.3: the
// default point alone carries the claim `undetermined`; the claim follows the
// points (stable with both neighbours, one_neighbour with one, not_stable if a
// check failed, undetermined if one is open); the same input gives the same
// bytes.
//
// Requirement: FR-STY-085, FR-STY-097, NFR-STY-002
//
// Guard: no default is called stable before the neighbour rule has run.
func TestRenderStability_Bytes(t *testing.T) {
	t.Parallel()
	open := [3]verdict.Result{verdict.Undetermined, verdict.Undetermined, verdict.Undetermined}
	all := [3]verdict.Result{verdict.Stable, verdict.Stable, verdict.Stable}
	s := verdict.Stability{Pack: "one", Seeds: "1..10", Commit: "abc123", Config: "9f8e", Points: []verdict.StabilityPoint{{Name: "default", Checks: open}}}
	want := `stability pack one  seeds 1..10  commit abc123  config 9f8e
point default  C-BP UNDETERMINED  C-NC UNDETERMINED  C-WS UNDETERMINED
claim undetermined
`
	if got := verdict.RenderStability(s); got != want {
		t.Errorf("RenderStability(default only) =\n%s\nwant\n%s", got, want)
	}
	failed := [3]verdict.Result{verdict.Stable, verdict.NotStable, verdict.Stable}
	for _, tc := range []struct {
		name   string
		points []verdict.StabilityPoint
		want   string
	}{
		{"default only, all stable", []verdict.StabilityPoint{{Name: "default", Checks: all}}, verdict.ClaimUndetermined},
		{"one neighbour", []verdict.StabilityPoint{{Name: "default", Checks: all}, {Name: "upper", Checks: all}}, verdict.ClaimOneNeighbour},
		{"both neighbours", []verdict.StabilityPoint{{Name: "default", Checks: all}, {Name: "lower", Checks: all}, {Name: "upper", Checks: all}}, verdict.ClaimStable},
		{"a check failed at a neighbour", []verdict.StabilityPoint{{Name: "default", Checks: all}, {Name: "lower", Checks: failed}, {Name: "upper", Checks: all}}, verdict.ClaimNotStable},
		{"a failure beats an open check", []verdict.StabilityPoint{{Name: "default", Checks: open}, {Name: "upper", Checks: failed}}, verdict.ClaimNotStable},
		{"an open check", []verdict.StabilityPoint{{Name: "default", Checks: all}, {Name: "lower", Checks: open}, {Name: "upper", Checks: all}}, verdict.ClaimUndetermined},
		{"no default point", []verdict.StabilityPoint{{Name: "lower", Checks: all}, {Name: "upper", Checks: all}}, verdict.ClaimUndetermined},
		{"the same neighbour twice", []verdict.StabilityPoint{{Name: "default", Checks: all}, {Name: "lower", Checks: all}, {Name: "lower", Checks: all}}, verdict.ClaimOneNeighbour},
		{"no point", nil, verdict.ClaimUndetermined},
	} {
		if got := verdict.ClaimOf(tc.points); got != tc.want {
			t.Errorf("ClaimOf(%s) = %q, want %q", tc.name, got, tc.want)
		}
	}
	s.Points = []verdict.StabilityPoint{{Name: "default", Checks: all}, {Name: "lower", Checks: failed}, {Name: "upper", Checks: all}}
	got := verdict.RenderStability(s)
	if !strings.Contains(got, "point lower  C-BP STABLE  C-NC NOT STABLE  C-WS STABLE\n") || !strings.HasSuffix(got, "claim not_stable\n") || got != verdict.RenderStability(s) {
		t.Errorf("RenderStability(three points) =\n%s\nwant the lower point's checks and claim not_stable", got)
	}
}

// TestCheck_RequiredConjuncts verifies the conjuncts of the three checks as
// STY 7.1 lists them, and the names of the checks.
//
// Requirement: FR-STY-095
func TestCheck_RequiredConjuncts(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"C-BP": {"STY-P01", "STY-P02", "R5a", "R5b"},
		"C-NC": {"STY-P03", "STY-D03", "R6"},
		"C-WS": {"STY-H01", "STY-H02", "STY-H03", "STY-H04", "STY-P04", "R7", "R8"},
	}
	if got := verdict.Checks(); len(got) != 3 || got[0] != "C-BP" || got[1] != "C-NC" || got[2] != "C-WS" {
		t.Errorf("Checks() = %v, want [C-BP C-NC C-WS]", got)
	}
	for check, names := range want {
		got := verdict.Required(check)
		if strings.Join(got, ",") != strings.Join(names, ",") {
			t.Errorf("Required(%s) = %v, want %v", check, got, names)
		}
		got[0] = "changed"
		if verdict.Required(check)[0] == "changed" {
			t.Errorf("Required(%s) returned its own slice, want a copy", check)
		}
	}
}

// TestRender_DoesNotChangeItsInput verifies that rendering sorts a copy: the
// scenarios, rows and regimes of the caller keep their order.
//
// Requirement: NFR-STY-002
func TestRender_DoesNotChangeItsInput(t *testing.T) {
	t.Parallel()
	r := verdict.Record{Pack: "p", Seeds: "1..2", Commit: "c", Config: "h", Scenarios: []verdict.ScenarioRecord{
		{Scenario: "S2", Facts: []verdict.FactRow{{ID: "B", Class: verdict.ClassX}, {ID: "A", Class: verdict.ClassX}},
			Regimes: []verdict.Verdict{{Detector: verdict.R9}, {Detector: verdict.R1}}},
		{Scenario: "S1"},
	}}
	verdict.RenderVerdict(r)
	if r.Scenarios[0].Scenario != "S2" || r.Scenarios[0].Facts[0].ID != "B" || r.Scenarios[0].Regimes[0].Detector != verdict.R9 {
		t.Errorf("RenderVerdict sorted its input: %+v", r)
	}
	if verdict.ClaimStable != "stable" || verdict.ClaimOneNeighbour != "one_neighbour" || verdict.ClaimNotStable != "not_stable" || verdict.ClaimUndetermined != "undetermined" {
		t.Errorf("claim words = %q %q %q %q, want the words of CLB 8.2", verdict.ClaimStable, verdict.ClaimNotStable, verdict.ClaimOneNeighbour, verdict.ClaimUndetermined)
	}
}

// TestRenderVerdict_IdenticalRows verifies that rows equal in every sort key
// are all kept and keep their input order (the sort is stable), so the
// rendering is a function of the record's row order.
//
// Requirement: FR-STY-085
//
// Guard: the rendering is the golden of a record; a sort that merged equal
// rows or reordered them would move the bytes with no change of the facts.
func TestRenderVerdict_IdenticalRows(t *testing.T) {
	t.Parallel()
	row := func(min int64) verdict.FactRow {
		return verdict.FactRow{ID: "STY-P02", Class: verdict.ClassB, Statistic: "s", N: 4, BandSet: true, Band: 1, HasValues: true, Min: min, Median: min, Max: min}
	}
	render := func(rows ...verdict.FactRow) string {
		return verdict.RenderVerdict(verdict.Record{Pack: "one", Scenarios: []verdict.ScenarioRecord{{Scenario: "S", Facts: rows}}})
	}
	got := render(row(7), row(7), row(5))
	if n := strings.Count(got, "\nfact STY-P02"); n != 3 {
		t.Errorf("RenderVerdict(7, 7, 5) has %d rows, want 3:\n%s", n, got)
	}
	if i, j := strings.Index(got, "min 7"), strings.Index(got, "min 5"); i < 0 || j < i {
		t.Errorf("RenderVerdict(7, 7, 5) =\n%s\nwant the rows in input order (min 7 before min 5)", got)
	}
}
