package experiment_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/experiment"
	"github.com/Whealinghold/economy-sim/internal/calibration"
)

// The dry run of the calibration procedure (spec-calibration.md 11.4): the
// procedure of CLB, end to end on pack one with a tiny experiment, to show that it
// is a procedure and not a prose. It asserts the shape of the procedure (the
// plan, the choice, the seeds, the digests, the record) and no number of the
// economy: every criterion reads a count of runs or a value the test itself set.
//
// Declared test constants (CLB 11.4, FR-REF-025; they are outside REF 8.2 and are
// listed in the description of the change that added this test): search block 1
// (seeds 101 to 103), confirm seeds 1 to 3, three replicas, 400 ticks, transient
// 100, windows of 100 for leg A and for the two drivers of leg B that run a pack;
// one replica and 20 ticks (transient 5, windows of 5) for the explore of leg B (e),
// whose criterion reads the setting of the knob and not the economy.
const (
	dryTicks      = 400
	dryTransient  = 100
	dryEvery      = 100
	dryReplicas   = 3
	drySearch0    = 101 // first seed of search block 1 (CLB 7.1)
	dryConfirm0   = 1   // first confirm seed: a prefix of the confirm range 1 to 50
	dryExploreTk  = 20
	dryExploreTr  = 5
	dryLineage    = "5a1c0e4477d3b210"
	dryRecordPath = "testdata/calibration/one/c4a-1.txt"
)

// dryKnob is a knob of the dry run: its registry ID and key, its path in a
// configuration file and its grid. The levels of its factor are the base and the
// neighbours it has, in that order (the first level of a factor is its baseline).
type dryKnob struct {
	id, name, key, path string
	grid                calibration.Grid
	word                string // the grid word of the record: geo or lin<step>
}

// rel gives, for each level of the factor, the position relative to the base:
// 0 for the base, -1 for the lower neighbour, 1 for the upper.
func (k dryKnob) rel() []int {
	out := []int{0}
	if k.grid.Lower.Present {
		out = append(out, -1)
	}
	if k.grid.Upper.Present {
		out = append(out, 1)
	}
	return out
}

// factor is the factor of the knob: a level for the base and one for each neighbour.
func (k dryKnob) factor() experiment.Factor {
	level := func(label string, v int64) experiment.Level {
		return experiment.Level{Label: label, Changes: []experiment.Change{{Path: k.path, Value: v}}}
	}
	f := experiment.Factor{Name: k.key, Levels: []experiment.Level{level("base", k.grid.Levels[k.grid.Base])}}
	if k.grid.Lower.Present {
		f.Levels = append(f.Levels, level("lower", k.grid.Lower.Level))
	}
	if k.grid.Upper.Present {
		f.Levels = append(f.Levels, level("upper", k.grid.Upper.Level))
	}
	return f
}

// line is the factor line of the record.
func (k dryKnob) line() calibration.Factor {
	opt := func(s calibration.Side) calibration.Opt { return calibration.Opt{Has: s.Present, Value: s.Level} }
	return calibration.Factor{
		Name: k.name, Key: k.key, Grid: k.word,
		Base: k.grid.Levels[k.grid.Base], Lower: opt(k.grid.Lower), Upper: opt(k.grid.Upper), Edge: "none",
	}
}

// dryKnobs builds the grids of the two knobs of leg A (spec-calibration.md 19.1):
// World.CostFloor (geo, base 7 000: neighbours 3 500 and 14 000) and
// World.BeliefEvidenceShare (lin1000, base 5 000: neighbours 4 000 and 6 000).
func dryKnobs(t *testing.T) (cost, share dryKnob) {
	t.Helper()
	g, err := calibration.Build(calibration.Rule{Kind: calibration.Geo}, 7000, calibration.Bounds{Lo: 1, Hi: 1_000_000},
		calibration.Off{Has: true, Value: 1}, nil)
	if err != nil {
		t.Fatalf("Build(CostFloor) error = %v, want nil", err)
	}
	cost = dryKnob{id: "K-BEL12", name: "cost_floor", key: "World.CostFloor", path: "world.cost_floor", grid: g, word: "geo"}
	g, err = calibration.Build(calibration.Rule{Kind: calibration.Lin, Step: 1000}, 5000, calibration.Bounds{Lo: 1, Hi: 10_000},
		calibration.Off{}, nil)
	if err != nil {
		t.Fatalf("Build(BeliefEvidenceShare) error = %v, want nil", err)
	}
	share = dryKnob{id: "K-BEL07", name: "belief_evidence_share", key: "World.BeliefEvidenceShare", path: "world.belief_evidence_share", grid: g, word: "lin1000"}
	return cost, share
}

// dryKnown is the small registry the checks of the dry run read: the rows of the
// knobs it moves, three facts of STY and a detector.
func dryKnown() calibration.Known {
	return calibration.Known{
		Knobs: map[string]calibration.KnobInfo{
			"K-BEL12": {Class: 'D', Key: "World.CostFloor", Stage: "C4", Owner: "BEL", Off: calibration.Off{Has: true, Value: 1}},
			"K-BEL07": {Class: 'D', Key: "World.BeliefEvidenceShare", Stage: "C4", Owner: "BEL"},
			"K-BEL01": {Class: 'D', Key: "World.BeliefStep", Stage: "C4", Owner: "BEL"},
			"K-WEL06": {Class: 'S', Key: "Role.ExpectedWorkTicks", Stage: "C2", Owner: "WEL"},
		},
		Facts:     map[string]byte{"P01": 'X', "P02": 'B', "P03": 'B'},
		Detectors: map[string]bool{"R5a": true},
		Paths:     map[string]string{"role.*.expected_work_ticks": "Role.ExpectedWorkTicks"},
	}
}

// dryClass is the class function of Overlay, read from dryKnown.
func dryClass(key string) byte {
	for _, k := range dryKnown().Knobs {
		if k.Key == key {
			return k.Class
		}
	}
	return 0
}

// Measures and criteria: a count of runs and the setting of the knob, never a
// reading of the economy. Criterion 0 is met by every point, 1 by none, 2 only by
// a point whose CostFloor is below its base.
var (
	dryRan       = experiment.Measure{Name: "ran", Fn: func(experiment.Sample) (int64, bool, error) { return 1, true, nil }}
	dryCostFloor = experiment.Measure{Name: "cost_floor", Fn: func(s experiment.Sample) (int64, bool, error) {
		return s.Config.World.CostFloor, true, nil
	}}
)

const (
	critAll = iota
	critNone
	critLower
)

// dryOps spells the operators of a criterion as the record does.
var dryOps = [...]string{experiment.AtMost: "AtMost", experiment.AtLeast: "AtLeast", experiment.Between: "Between"}

func dryCriteria() []experiment.Criterion {
	return []experiment.Criterion{
		{Name: "all", Measure: "ran", Op: experiment.AtLeast, A: 1},
		{Name: "none", Measure: "ran", Op: experiment.AtLeast, A: 2},
		{Name: "lower", Measure: "cost_floor", Op: experiment.AtMost, A: 6999},
	}
}

// drySpec is the screen of leg A: OneAtATime over the two knobs.
func drySpec(t *testing.T, cost, share dryKnob) experiment.Spec {
	t.Helper()
	return experiment.Spec{
		Name: "dry-screen", Base: onePack(t), Factors: []experiment.Factor{cost.factor(), share.factor()},
		Design: experiment.OneAtATime, Replicas: dryReplicas, Seed0: drySearch0,
		Ticks: dryTicks, Transient: dryTransient, Every: dryEvery,
		Measures: []experiment.Measure{dryRan, dryCostFloor}, Criteria: dryCriteria(),
	}
}

// dryRun runs a spec and keeps its records, so the result can be assembled again.
func dryRun(t *testing.T, e experiment.Spec) (experiment.Result, []experiment.Record) {
	t.Helper()
	var recs []experiment.Record
	res, err := experiment.Run(context.Background(), e, experiment.WithRecordFunc(func(r experiment.Record) { recs = append(recs, r) }))
	if err != nil {
		t.Fatalf("Run(%s) error = %v, want nil", e.Name, err)
	}
	return res, recs
}

// dryPoints reads the result as the choice does: the levels of each point relative
// to the base and whether every run of it passed the criterion.
func dryPoints(res experiment.Result, rel [][]int, crit int) []calibration.Point {
	pts := make([]calibration.Point, len(res.Points))
	for i, p := range res.Points {
		pt := calibration.Point{Index: p.Index, Levels: make([]int, len(p.Levels))}
		for f, l := range p.Levels {
			pt.Levels[f] = rel[f][l]
		}
		ps := p.Variants[0].Pass[crit]
		pt.ExactPass = p.Status == experiment.StatusOK && ps.Planned > 0 && ps.Passes == ps.Planned
		pts[i] = pt
	}
	return pts
}

// hex16 is the first eight bytes of a hash as 16 hex digits.
func hex16(h [32]byte) string { return fmt.Sprintf("%x", h[:8]) }

// dryExperiment is the experiment line of a step: the plain data of its spec and
// the digest of the rendering of the stored result (FR-CLB-040).
func dryExperiment(step, design string, e experiment.Spec, res experiment.Result, points int) calibration.Experiment {
	return calibration.Experiment{
		Step: step, Design: design, Points: points, Replicas: e.Replicas, Seed0: e.Seed0, Seeding: "same",
		Ticks: e.Ticks, Transient: e.Transient, Every: e.Every, Render: calibration.Digest([]byte(experiment.Render(res))),
	}
}

// dryDocuments are the documents of the pack, named and digested as the record
// names them.
func dryDocuments(t *testing.T) []calibration.Document {
	t.Helper()
	var out []calibration.Document
	for i, name := range []string{"base.toml", "one.toml", "warm-one.toml"} {
		out = append(out, calibration.Document{Name: name, Digest: calibration.Digest(onePack(t)[i])})
	}
	return out
}

// dryRecord is the record of one stage attempt of the dry run: the header, the
// screen and its factors, the choice and, for a chosen point, its set and its
// confirm. The facts and the detector stand for the verdicts of STY: they are
// declared constants of the dry run, which asserts no number of the economy.
func dryRecord(t *testing.T, screen calibration.Experiment, factors []calibration.Factor, c calibration.Choice, crit experiment.Criterion) calibration.Record {
	t.Helper()
	r := calibration.Record{
		Pack: "one", Lineage: dryLineage, Stage: "C4a", Attempt: 1, Block: 1, Commit: "dry-run",
		Documents:   dryDocuments(t),
		Experiments: []calibration.Experiment{screen},
		Factors:     factors,
		Measures: []calibration.Measure{
			{Name: "ran", Fact: "none", Class: 'R', Kind: "counter"},
			{Name: "cost_floor", Fact: "none", Class: 'R', Kind: "counter"},
		},
		Criteria: []calibration.Criterion{{Name: crit.Name, Measure: crit.Measure, Op: dryOps[crit.Op], A: crit.A, B: crit.B, Target: "report"}},
		Feasible: calibration.Feasible{Has: true, N: len(c.Feasible), Of: screen.Points},
		Pick:     calibration.Pick{Has: true, None: c.None, Index: c.Index, Distance: c.Distance},
		Pareto:   c.Pareto,
		Facts: []calibration.Fact{
			{ID: "P01", Class: 'X', Result: "pass", Target: "none"},
			{ID: "P02", Class: 'B', Target: "set", Result: "pass"},
			{ID: "P03", Class: 'B', Target: "working", Result: "pass"},
		},
		Detectors: []calibration.Detector{{ID: "R5a", Verdict: "none", Working: true}},
		Decision:  "stop",
	}
	if c.None {
		r.Decision = "revert"
	}
	return r
}

// dryOut is what one run of the procedure leaves: the stored records of the screen
// and of the confirm, the three choices and their records and overlays.
type dryOut struct {
	cost, share dryKnob
	screen      experiment.Spec
	confirm     experiment.Spec
	screenRecs  []experiment.Record
	confirmRecs []experiment.Record
	screenRes   experiment.Result
	confirmRes  experiment.Result
	choice      map[int]calibration.Choice // by criterion
	rec         map[int]calibration.Record
	text        map[int][]byte // the rendered records
	overlay     map[int][]byte // the overlay of each choice
	chosenPoint int
}

// dryProcedure runs the procedure once: the screen of five points, the choice
// under the three criteria sets, the overlay of each choice and, for the third, a
// confirm of the chosen point against the base on seeds 1 to 3, then the records.
func dryProcedure(t *testing.T) dryOut {
	t.Helper()
	o := dryOut{choice: map[int]calibration.Choice{}, rec: map[int]calibration.Record{}, text: map[int][]byte{}, overlay: map[int][]byte{}}
	o.cost, o.share = dryKnobs(t)
	o.screen = drySpec(t, o.cost, o.share)
	o.screenRes, o.screenRecs = dryRun(t, o.screen)
	rel := [][]int{o.cost.rel(), o.share.rel()}
	factors := []calibration.Factor{o.cost.line(), o.share.line()}
	screenLine := dryExperiment("screen", "oat", o.screen, o.screenRes, len(o.screenRes.Points))
	for crit, cr := range o.screen.Criteria {
		c := calibration.Choose(dryPoints(o.screenRes, rel, crit), 0, dryReplicas)
		o.choice[crit] = c
		o.rec[crit] = dryRecord(t, screenLine, factors, c, cr)
	}
	o.chosenPoint = o.choice[critLower].Index
	o.confirmSpec(t)
	for crit := range o.rec {
		o.finish(t, crit)
	}
	return o
}

// confirmSpec runs the confirm: the chosen point and the base as the control, on
// seeds 1 to 3, with the measures, ticks and windows of the screen.
func (o *dryOut) confirmSpec(t *testing.T) {
	t.Helper()
	chosen := o.cost.factor() // the lower neighbour is level 1 of the cost factor
	e := o.screen
	e.Name, e.Seed0, e.Design = "dry-confirm", dryConfirm0, experiment.Grid
	e.Factors = []experiment.Factor{{Name: "arm", Levels: []experiment.Level{chosen.Levels[1], chosen.Levels[0]}}}
	o.confirm = e
	o.confirmRes, o.confirmRecs = dryRun(t, e)
}

// finish adds to the record of a criterion what the choice says: the set and the
// confirm of a chosen point, the overlay of a choice, and the rendered text.
func (o *dryOut) finish(t *testing.T, crit int) {
	t.Helper()
	r := o.rec[crit]
	c := o.choice[crit]
	if !c.None && c.Index != 0 {
		r.Sets = []calibration.Setting{{Key: o.cost.key, Value: o.cost.grid.Lower.Level}}
		r.Experiments = append(r.Experiments, dryExperiment("confirm", "grid", o.confirm, o.confirmRes, len(o.confirmRes.Points)))
		r.Confirm = calibration.ConfirmLine{Has: true, First: dryConfirm0, Last: dryConfirm0 + dryReplicas - 1, Control: true}
		r.Chosen = hex16(o.screenRes.Points[c.Index].Hash)
	} else if crit == critLower {
		t.Fatalf("Choose(lower) = %+v, want a point other than the base", c)
	}
	r.Base = hex16(o.screenRes.Points[0].Hash)
	if r.Chosen == "" {
		r.Chosen = "same"
	}
	var changes []calibration.Change
	for _, s := range r.Sets {
		changes = append(changes, calibration.Change{Key: s.Key, Path: o.cost.path, Value: s.Value})
	}
	ov, err := calibration.Overlay(calibration.Header{Stage: r.Stage, Attempt: r.Attempt, Lineage: r.Lineage, Record: dryRecordPath}, changes, dryClass)
	if err != nil {
		t.Fatalf("Overlay(%v) error = %v, want nil", changes, err)
	}
	o.overlay[crit] = ov
	text, err := calibration.Render(r)
	if err != nil {
		t.Fatalf("Render(record %d) error = %v, want nil", crit, err)
	}
	o.rec[crit], o.text[crit] = r, text
}

var (
	dryOnce sync.Once
	dryRes  dryOut
	dryDone bool
)

// dryFirst runs the procedure once for the tests of leg A and of leg B (d).
func dryFirst(t *testing.T) dryOut {
	t.Helper()
	dryOnce.Do(func() { dryRes = dryProcedure(t); dryDone = true })
	if !dryDone { // the first caller failed inside the Once: say so, not a panic on the zero value
		t.Fatal("the dry procedure failed in an earlier test; see its failure")
	}
	return dryRes
}

// TestCalibrationDry_LegA verifies the eight asserts of spec-calibration.md 11.4 on
// the screen of two knobs of pack one: the plan, the choice as a function of the
// stored result, the empty overlay of a base that passes, `chosen none` and no stop
// when no point passes, the seeds of each step, the same record bytes from a second
// run, the digest of each step and the control of the confirm.
//
// Requirement: FR-CLB-012, FR-CLB-020, FR-CLB-021, FR-CLB-022, FR-CLB-025, FR-CLB-031, FR-CLB-033, FR-CLB-034, FR-CLB-038, FR-CLB-040, FR-CLB-041, FR-CLB-091
func TestCalibrationDry_LegA(t *testing.T) {
	short(t)
	o := dryFirst(t)
	dryAssertPlan(t, o)
	dryAssertChoices(t, o)
	dryAssertSeeds(t, o)
	dryAssertDigests(t, o)
	dryAssertControl(t, o)
	if raceEnabled {
		// The bytes of a second run are proven by the run without -race (gate F): no
		// goroutine is involved, so -race would only repeat it about ten times slower.
		return
	}
	again := dryProcedure(t)
	for crit, want := range o.text {
		if !bytes.Equal(again.text[crit], want) || !bytes.Equal(again.overlay[crit], o.overlay[crit]) {
			t.Errorf("second run, criterion %d: record or overlay differs from the first run (FR-CLB-041)", crit)
		}
	}
}

// dryAssertPlan checks assert (1): the plan is the base and the two neighbours of
// each knob, five points, and Check resolves them without a tick.
func dryAssertPlan(t *testing.T, o dryOut) {
	t.Helper()
	if got := calibration.ScreenPoints(o.cost.grid, o.share.grid); got != 5 {
		t.Errorf("ScreenPoints = %d, want 5 (the base and two neighbours of two knobs)", got)
	}
	if got, want := calibration.ScreenLevels(o.cost.grid), []int64{3500, 7000, 14000}; !slices.Equal(got, want) {
		t.Errorf("ScreenLevels(CostFloor) = %v, want %v", got, want)
	}
	if got, want := calibration.ScreenLevels(o.share.grid), []int64{4000, 5000, 6000}; !slices.Equal(got, want) {
		t.Errorf("ScreenLevels(BeliefEvidenceShare) = %v, want %v", got, want)
	}
	checks, err := experiment.Check(context.Background(), o.screen)
	if err != nil {
		t.Fatalf("Check(screen) error = %v, want nil", err)
	}
	if len(checks) != 5 {
		t.Fatalf("Check(screen) = %d points, want 5", len(checks))
	}
	for _, pc := range checks {
		if pc.Status != experiment.StatusOK {
			t.Errorf("Check(screen) point %d status = %v (%s), want ok", pc.Point, pc.Status, pc.Detail)
		}
	}
	wantLevels := [][]int{{0, 0}, {1, 0}, {2, 0}, {0, 1}, {0, 2}}
	for i, p := range o.screenRes.Points {
		if !slices.Equal(p.Levels, wantLevels[i]) {
			t.Errorf("screen point %d levels = %v, want %v", i, p.Levels, wantLevels[i])
		}
	}
}

// dryAssertChoices checks asserts (2) to (4): the choice is a function of the
// stored result, a base that passes is chosen with an empty overlay, and no point
// passing gives `chosen none` and a decision other than stop.
func dryAssertChoices(t *testing.T, o dryOut) {
	t.Helper()
	rel := [][]int{o.cost.rel(), o.share.rel()}
	stored, err := experiment.Assemble(o.screen, o.screenRecs)
	if err != nil {
		t.Fatalf("Assemble(stored records) error = %v, want nil", err)
	}
	for crit, c := range o.choice {
		again := calibration.Choose(dryPoints(stored, rel, crit), 0, dryReplicas)
		if !slices.Equal(again.Feasible, c.Feasible) || again.Index != c.Index || again.None != c.None {
			t.Errorf("Choose(criterion %d) on the stored result = %+v, want %+v (FR-CLB-034)", crit, again, c)
		}
	}
	dryAssertBaseChosen(t, o)
	dryAssertNoneChosen(t, o)
	if c := o.choice[critLower]; c.Index != o.chosenPoint || o.chosenPoint != 1 || c.Distance != 1 {
		t.Errorf("Choose(lower) = %+v, want point 1 (CostFloor lower) at distance 1", c)
	}
}

// dryAssertBaseChosen checks assert (3): with criteria every point meets, the base
// is chosen and the overlay holds the header comment alone.
func dryAssertBaseChosen(t *testing.T, o dryOut) {
	t.Helper()
	if c := o.choice[critAll]; c.None || c.Index != 0 || len(c.Feasible) != 5 {
		t.Errorf("Choose(all pass) = %+v, want the base (0) of 5 feasible points (FR-CLB-031)", c)
	}
	ov := string(o.overlay[critAll])
	if strings.Contains(ov, "[") || !strings.HasPrefix(ov, "# calibration C4a attempt 1 of lineage 5a1c0e44") {
		t.Errorf("Overlay(base chosen) = %q, want the header comment alone", ov)
	}
}

// dryAssertNoneChosen checks assert (4): with criteria no point meets, the result
// is `chosen none` and the decision is not a stop.
func dryAssertNoneChosen(t *testing.T, o dryOut) {
	t.Helper()
	if c := o.choice[critNone]; !c.None || len(c.Feasible) != 0 {
		t.Errorf("Choose(none pass) = %+v, want none (FR-CLB-033)", c)
	}
	if r := o.rec[critNone]; r.Decision == "stop" || !r.Pick.None {
		t.Errorf("record(none pass) decision = %q, chosen none = %v, want a decision other than stop and chosen none (FR-CLB-038)", r.Decision, r.Pick.None)
	}
}

// drySeedsOut reports the runs whose seed is outside [lo, hi].
func drySeedsOut(recs []experiment.Record, lo, hi uint64) []experiment.Key {
	var out []experiment.Key
	for _, r := range recs {
		if r.Seed < lo || r.Seed > hi {
			out = append(out, r.Key)
		}
	}
	return out
}

// dryAssertSeeds checks assert (5): the screen ran on block 1, the confirm on seeds
// 1 to 3, and no search seed lies in 1 to 60.
func dryAssertSeeds(t *testing.T, o dryOut) {
	t.Helper()
	first, last, err := calibration.Block(1)
	if err != nil {
		t.Fatalf("Block(1) error = %v, want nil", err)
	}
	if out := drySeedsOut(o.screenRecs, first, last); len(out) != 0 {
		t.Errorf("screen runs %v are outside block 1 [%d, %d]", out, first, last)
	}
	for _, r := range o.screenRecs {
		if calibration.InSTYRange(r.Seed) {
			t.Errorf("screen run %+v seed = %d, want no search seed in 1 to 60 (FR-CLB-021)", r.Key, r.Seed)
		}
	}
	cFirst, cLast := calibration.ConfirmSeeds(false)
	if out := drySeedsOut(o.confirmRecs, cFirst, cLast); len(out) != 0 {
		t.Errorf("confirm runs %v are outside the confirm range [%d, %d]", out, cFirst, cLast)
	}
	if out := drySeedsOut(o.confirmRecs, 0, first-1); len(out) != 0 {
		t.Errorf("confirm runs %v share a seed with the search block", out)
	}
	if got := len(o.screenRecs); got != 5*dryReplicas {
		t.Errorf("screen ran %d runs, want %d", got, 5*dryReplicas)
	}
}

// dryAssertDigests checks assert (7): the render digest of each step of the record
// is the digest of Render of the stored result, and the record reads back.
func dryAssertDigests(t *testing.T, o dryOut) {
	t.Helper()
	r := o.rec[critLower]
	stored, err := experiment.Assemble(o.confirm, o.confirmRecs)
	if err != nil {
		t.Fatalf("Assemble(confirm) error = %v, want nil", err)
	}
	if want := calibration.Digest([]byte(experiment.Render(stored))); r.Experiments[1].Render != want {
		t.Errorf("confirm render digest = %s, want %s: the digest of Render of the stored result (FR-CLB-040)", r.Experiments[1].Render, want)
	}
	for crit, text := range o.text {
		back, err := calibration.Parse(text)
		if err != nil {
			t.Fatalf("Parse(record %d) error = %v, want nil", crit, err)
		}
		if again, err := calibration.Render(back); err != nil || !bytes.Equal(again, text) {
			t.Errorf("Render(Parse(record %d)) differs from the record (FR-CLB-041), error = %v", crit, err)
		}
	}
}

// dryAssertControl checks assert (8): the control of the confirm is the base, on the
// same seeds, measures and windows as the chosen point.
func dryAssertControl(t *testing.T, o dryOut) {
	t.Helper()
	plan := calibration.ConfirmPlan("chosen", "base", false)
	if plan.Arms != [2]string{"chosen", "base"} || plan.First != dryConfirm0 {
		t.Errorf("ConfirmPlan = %+v, want the chosen configuration and the base as control from seed %d", plan, dryConfirm0)
	}
	if len(o.confirmRes.Points) != 2 {
		t.Fatalf("confirm has %d points, want 2 (the chosen point and the control)", len(o.confirmRes.Points))
	}
	if got, want := o.confirmRes.Points[1].Hash, o.screenRes.Points[0].Hash; got != want {
		t.Errorf("confirm control hash = %x, want the base's %x", got[:8], want[:8])
	}
	if got, want := o.confirmRes.Points[0].Hash, o.screenRes.Points[o.chosenPoint].Hash; got != want {
		t.Errorf("confirm chosen hash = %x, want the screen's chosen point %x", got[:8], want[:8])
	}
	dryAssertSameFrame(t, o)
}

// dryAssertSameFrame checks that the confirm runs on the seeds, measures, ticks and
// windows of the screen (FR-CLB-025).
func dryAssertSameFrame(t *testing.T, o dryOut) {
	t.Helper()
	s, c := o.screen, o.confirm
	if s.Ticks != c.Ticks || s.Transient != c.Transient || s.Every != c.Every || len(s.Measures) != len(c.Measures) {
		t.Errorf("confirm ticks, windows or measures differ from the screen's (FR-CLB-025)")
	}
	for _, rc := range o.confirmRecs {
		if rc.Seed != dryConfirm0+uint64(rc.Key.Replica) {
			t.Errorf("confirm run %+v seed = %d, want the same seeds for both arms", rc.Key, rc.Seed)
		}
	}
}

var reHex16 = regexp.MustCompile(`\b[0-9a-f]{16}\b`)

// maskShape replaces every 16-digit hex digest by <hex16>: what is left is the
// shape of the procedure (D637).
func maskShape(b []byte) string { return reHex16.ReplaceAllString(string(b), "<hex16>") }

// TestCalibrationDry_Shape verifies that the three records of the dry run, with
// every digest masked, equal testdata/calibration/dryrun.shape.txt. The file holds
// the keywords, the order, the points, the replicas, the seeds, the blocks and the
// decisions of the procedure and no number of the economy, so it moves when the
// procedure changes and not when the economy does.
//
// Guard: a change of the record grammar, of the plan or of the choice that alters a
// stage record shows here before it reaches a calibration. Regenerate it with
// -update only after reading testing.md 7.2.
//
// Requirement: FR-CLB-041, FR-CLB-091
func TestCalibrationDry_Shape(t *testing.T) {
	short(t)
	o := dryFirst(t)
	var parts []string
	for crit := range 3 {
		parts = append(parts, maskShape(o.text[crit])+maskShape(o.overlay[crit]))
	}
	got := strings.Join(parts, "---\n")
	path := filepath.Join("..", "testdata", "calibration", "dryrun.shape.txt")
	if *updateFlag {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v, want nil", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v; record it with -update after reading testing.md 7.2", path, err)
	}
	if string(want) != got {
		t.Errorf("masked dry-run records differ from %s:\n got:\n%s\nwant:\n%s", path, got, want)
	}
}

// TestCalibrationDry_ChosenOverlay verifies that the overlay of a chosen
// neighbour is one key that the loader accepts, that the configuration passes
// Validate, and that the world built from the base and the overlay has the
// ConfigHash the record names as `config chosen`.
//
// Requirement: FR-CLB-032, FR-CLB-050, FR-CLB-051
func TestCalibrationDry_ChosenOverlay(t *testing.T) {
	short(t)
	o := dryFirst(t)
	ov := string(o.overlay[critLower])
	if want := "# calibration C4a attempt 1 of lineage 5a1c0e44: see " + dryRecordPath + "\n[world]\ncost_floor = 3500\n"; ov != want {
		t.Fatalf("Overlay(chosen) = %q, want %q", ov, want)
	}
	cfg, err := config.Parse(append(onePack(t), o.overlay[critLower])...)
	if err != nil {
		t.Fatalf("Parse(base, overlay) error = %v, want nil", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(base + overlay) error = %v, want nil", err)
	}
	w, err := economysim.NewWorld(cfg)
	if err != nil {
		t.Fatalf("NewWorld(base + overlay) error = %v, want nil", err)
	}
	if got, want := hex16(w.ConfigHash()), o.rec[critLower].Chosen; got != want {
		t.Errorf("ConfigHash(base + overlay) = %s, want the record's config chosen %s", got, want)
	}
	if known := dryKnown(); len(calibration.CheckRecord(o.rec[critLower], known)) != 0 {
		t.Errorf("CheckRecord(chosen) = %v, want none", calibration.CheckRecord(o.rec[critLower], known))
	}
}

// dryShortSpec is a driver of leg B that runs a pack for a few ticks: the
// criterion or the check it exists for reads the setting or the status of a point,
// not the economy.
func dryShortSpec(t *testing.T, base [][]byte, name string, f experiment.Factor) experiment.Spec {
	t.Helper()
	return experiment.Spec{
		Name: name, Base: base, Factors: []experiment.Factor{f}, Design: experiment.Grid,
		Replicas: 1, Seed0: drySearch0, Ticks: dryExploreTk, Transient: dryExploreTr, Every: dryExploreTr,
		Measures: []experiment.Measure{dryRan, dryCostFloor}, Criteria: dryCriteria(),
	}
}

// dryTicksOf reads the expected work ticks of a role of a resolved configuration.
func dryTicksOf(cfg economysim.Config, role string) int64 {
	for _, r := range cfg.Roles {
		if r.ID == role {
			return int64(r.ExpectedWorkTicks)
		}
	}
	return -1
}

// TestCalibrationDry_LegBTable verifies leg B (b): a pack quantity with the flag
// `in` (Role.ExpectedWorkTicks, K-WEL06) as a grid of three values regenerates the
// warm-start table for each point with Prepare, after every other document, and the
// record names the table of each point by the digest of the world's ConfigHash.
//
// Requirement: FR-CLB-037, FR-CLB-052
func TestCalibrationDry_LegBTable(t *testing.T) {
	short(t)
	g, err := calibration.Build(calibration.Rule{Kind: calibration.Lin, Step: 25}, 250, calibration.Bounds{Lo: 1, Hi: 6000}, calibration.Off{}, nil)
	if err != nil {
		t.Fatalf("Build(ExpectedWorkTicks) error = %v, want nil", err)
	}
	knob := dryKnob{id: "K-WEL06", name: "expected_work_ticks", key: "role.cook.expected_work_ticks", path: "role.cook.expected_work_ticks", grid: g, word: "lin25"}
	if got, want := calibration.ScreenLevels(g), []int64{225, 250, 275}; !slices.Equal(got, want) {
		t.Fatalf("ScreenLevels(ExpectedWorkTicks) = %v, want %v", got, want)
	}
	var seen []int64 // the setting each Prepare saw: the overlays are already applied
	e := dryShortSpec(t, onePack(t), "dry-table", knob.factor())
	e.Prepare = func(ctx context.Context, cfg economysim.Config) (economysim.Config, error) {
		seen = append(seen, dryTicksOf(cfg, "cook"))
		return dryRegenerate(ctx, cfg)
	}
	res, _ := dryRun(t, e)
	if want := []int64{250, 225, 275}; !slices.Equal(seen, want) {
		t.Errorf("Prepare saw ExpectedWorkTicks %v, want %v: once per point, after the overlay of the point", seen, want)
	}
	rec := dryTableRecord(t, res, knob)
	text, err := calibration.Render(rec)
	if err != nil {
		t.Fatalf("Render(table record) error = %v, want nil", err)
	}
	if !bytes.Contains(text, []byte("document warm-table-2 "+hex16(res.Points[2].Hash))) {
		t.Errorf("record = %q, want the digest of the table of each point", text)
	}
	if ps := calibration.CheckRecord(rec, dryKnown()); len(ps) != 0 {
		t.Errorf("CheckRecord(table) = %v, want none: the path of a role resolves to its registry row", ps)
	}
}

// dryRegenerate is the Prepare of a pack quantity with the flag `in`: it solves the
// configuration of the point and installs the warm-start table of the solution.
func dryRegenerate(ctx context.Context, cfg economysim.Config) (economysim.Config, error) {
	ss, err := economysim.SolveSteadyState(ctx, cfg)
	if err != nil {
		return cfg, err
	}
	cfg.WarmStart = &ss.Table
	return cfg, nil
}

// dryTableRecord is the record of the table driver: one document line for the table
// of each point, named by the digest of the ConfigHash of its world (D636), and
// the failures of a point that did not build a world.
func dryTableRecord(t *testing.T, res experiment.Result, knob dryKnob) calibration.Record {
	t.Helper()
	rec := calibration.Record{
		Pack: "one", Lineage: dryLineage, Stage: "C2", Attempt: 1, Commit: "dry-run", Base: hex16(res.Points[0].Hash), Chosen: "same",
		Documents: dryDocuments(t), Factors: []calibration.Factor{knob.line()}, Decision: "revert",
	}
	for _, p := range res.Points {
		if p.Status != experiment.StatusOK || p.Hash == ([32]byte{}) {
			t.Errorf("table point %d status = %v, hash %x, want ok and a world built", p.Index, p.Status, p.Hash[:4])
		}
		for q := range p.Index {
			if p.Hash == res.Points[q].Hash {
				t.Errorf("table points %d and %d have the same ConfigHash, want one table for each point", q, p.Index)
			}
		}
		rec.Documents = append(rec.Documents, calibration.Document{Name: fmt.Sprintf("warm-table-%d", p.Index), Digest: hex16(p.Hash)})
	}
	return rec
}

// TestCalibrationDry_LegBInvalidLevel verifies leg B (c): World.BeliefStep on a
// test-local overlay that sets the reference price of the cheapest tradable
// commodity to 96 has an invalid lower level, which the grid skips on the verdict
// of Check, the edge is `below:rule`, and a run that does include the level records
// it as invalid_config and goes on with the others.
//
// Requirement: FR-CLB-012, FR-CLB-108, FR-EXP-050
func TestCalibrationDry_LegBInvalidLevel(t *testing.T) {
	short(t)
	// The reference price of the staple is 96 in pack one today (spec-calibration.md
	// 19.1); the overlay states it so that the test does not depend on a rebalance.
	base := append(onePack(t), []byte("[commodity.staple]\nreference_price = 96\n"))
	level := func(v int64) experiment.Level {
		return experiment.Level{Changes: []experiment.Change{{Path: "world.belief_step", Value: v}}}
	}
	valid := func(v int64) bool {
		pcs, err := experiment.Check(context.Background(), dryShortSpec(t, base, "dry-valid",
			experiment.Factor{Levels: []experiment.Level{level(v), level(200)}}))
		return err == nil && pcs[0].Status == experiment.StatusOK
	}
	g, err := calibration.Build(calibration.Rule{Kind: calibration.Geo}, 200, calibration.Bounds{Lo: 1, Hi: 2500}, calibration.Off{}, valid)
	if err != nil {
		t.Fatalf("Build(BeliefStep) error = %v, want nil", err)
	}
	if g.Lower.Present || g.Lower.Edge != calibration.RuleEdge || !g.Upper.Present || g.Upper.Level != 400 {
		t.Errorf("BeliefStep grid = lower %+v upper %+v, want no lower level (edge rule) and the upper neighbour 400", g.Lower, g.Upper)
	}
	res, _ := dryRun(t, dryShortSpec(t, base, "dry-invalid", experiment.Factor{Levels: []experiment.Level{level(200), level(100), level(400)}}))
	if got := []experiment.Status{res.Points[0].Status, res.Points[1].Status, res.Points[2].Status}; !slices.Equal(got,
		[]experiment.Status{experiment.StatusOK, experiment.StatusInvalidConfig, experiment.StatusOK}) {
		t.Errorf("statuses of BeliefStep 200, 100, 400 = %v, want ok, invalid_config, ok: the run goes on", got)
	}
	line := calibration.Factor{Name: "belief_step", Key: "World.BeliefStep", Grid: "geo", Base: 200, Upper: calibration.Opt{Has: true, Value: 400}, Edge: "below:rule"}
	rec := calibration.Record{Pack: "one", Lineage: dryLineage, Stage: "C4a", Attempt: 1, Commit: "dry-run", Chosen: "same",
		Base: hex16(res.Points[0].Hash), Documents: dryDocuments(t), Factors: []calibration.Factor{line}, Decision: "revert"}
	if _, err := calibration.Render(rec); err != nil {
		t.Errorf("Render(edge below:rule) error = %v, want nil", err)
	}
	if ps := calibration.CheckRecord(rec, dryKnown()); len(ps) != 0 {
		t.Errorf("CheckRecord(edge below:rule) = %v, want none", ps)
	}
}

// dryExplore runs an explore of one factor over the levels of a span of the grid of
// CostFloor, on its own: a grid design of one replica for a few ticks, whose only
// feasible point is the top level of the span (the criterion reads the setting).
func dryExplore(t *testing.T, cost dryKnob, s calibration.Span, seed0 uint64) experiment.Result {
	t.Helper()
	f := experiment.Factor{}
	for _, v := range cost.grid.Levels[s.Lo : s.Hi+1] {
		f.Levels = append(f.Levels, experiment.Level{Changes: []experiment.Change{{Path: cost.path, Value: v}}})
	}
	e := dryShortSpec(t, onePack(t), "dry-explore", f)
	e.Seed0 = seed0
	e.Criteria = []experiment.Criterion{{Name: "top", Measure: "cost_floor", Op: experiment.AtLeast, A: cost.grid.Levels[s.Hi]}}
	res, _ := dryRun(t, e)
	return res
}

// dryWiden widens the upper end of an explore by two levels and checks the rule.
func dryWiden(t *testing.T, g calibration.Grid, span calibration.Span) calibration.Span {
	t.Helper()
	if calibration.AtEdge(g, span, true) {
		t.Fatalf("AtEdge(%+v, upper) = true, want an end that can be widened", span)
	}
	wide, ok := calibration.Widen(g, span, true)
	if !ok || wide.Hi != span.Hi+2 || wide.Lo != span.Lo {
		t.Fatalf("Widen(%+v, upper) = %+v, %v, want the upper end two levels further", span, wide, ok)
	}
	return wide
}

// TestCalibrationDry_LegBWiden verifies leg B (e): an explore whose feasible points
// touch its end is widened by two levels on a new block, and the new block is the
// next one, 201 to 300.
//
// Requirement: FR-CLB-023, FR-CLB-026
func TestCalibrationDry_LegBWiden(t *testing.T) {
	short(t)
	cost, _ := dryKnobs(t)
	span := calibration.Explore(cost.grid)
	res := dryExplore(t, cost, span, drySearch0)
	if top := res.Points[len(res.Points)-1]; top.Variants[0].Pass[0].Passes != 1 || span.Hi-span.Lo != 4 {
		t.Fatalf("explore of %+v: top point passes = %d, want 1 over five levels", span, top.Variants[0].Pass[0].Passes)
	}
	wide := dryWiden(t, cost.grid, span)
	first, last, err := calibration.Block(2)
	if err != nil || first != 201 || last != 300 {
		t.Fatalf("Block(2) = %d, %d, %v, want 201, 300", first, last, err)
	}
	res = dryExplore(t, cost, wide, first)
	if got, want := len(res.Points), 7; got != want {
		t.Errorf("widened explore has %d points, want %d", got, want)
	}
	for _, p := range res.Points {
		if out := drySeedsOut(p.Variants[0].Runs, first, last); len(out) != 0 {
			t.Errorf("widened explore point %d runs %v are outside block 2 [%d, %d]", p.Index, out, first, last)
		}
	}
}

func ruleNames(ps []calibration.Problem) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Rule
	}
	return out
}

// dryAttempt returns the record of attempt n of a stage, on search block n.
func dryAttempt(r calibration.Record, n int, decision string) calibration.Record {
	r.Attempt, r.Block, r.Decision = n, n, decision
	r.Experiments = slices.Clone(r.Experiments)
	r.Experiments[0].Seed0 = uint64(100*n + 1)
	return r
}

// TestCalibrationDry_LegBProcess verifies leg B (f) to (i), which are pure checks on
// stored records and run no tick: three failed attempts stop the stage until a
// review, going back to a stage needs a reentry, a failing X fact closes the search,
// and a stop needs all seven conditions.
//
// Requirement: FR-CLB-027, FR-CLB-035, FR-CLB-036, FR-CLB-038
func TestCalibrationDry_LegBProcess(t *testing.T) {
	short(t)
	o := dryFirst(t)
	k := dryKnown()
	failed, good := o.rec[critNone], o.rec[critLower]
	stage := func(stage string, attempt int, decision string, facts ...calibration.Fact) calibration.Record {
		return calibration.Record{Pack: "one", Lineage: dryLineage, Stage: stage, Attempt: attempt, Decision: decision, Facts: facts}
	}
	x := func(result string) calibration.Fact { return calibration.Fact{ID: "P01", Class: 'X', Result: result} }
	again := dryAttempt(good, 1, "stop")
	again.Block, again.Experiments = 2, slices.Clone(good.Experiments)
	again.Experiments[0].Seed0 = 201
	tests := []struct {
		name string
		rs   []calibration.Record
		want []string
	}{
		{"f three attempts then a review", []calibration.Record{dryAttempt(failed, 1, "revert"), dryAttempt(failed, 2, "revert"), dryAttempt(failed, 3, "review"), dryAttempt(failed, 4, "revert")}, nil},
		{"f a fourth attempt with no review", []calibration.Record{dryAttempt(failed, 1, "revert"), dryAttempt(failed, 2, "revert"), dryAttempt(failed, 3, "revert"), dryAttempt(failed, 4, "revert")}, []string{calibration.RuleAttempts}},
		{"g reentry to the first stage", []calibration.Record{good, stage("C5", 1, "reentry C4a"), again}, nil},
		{"g back with no reentry", []calibration.Record{good, stage("C5", 1, "stop"), again}, []string{calibration.RuleStageOrder}},
		{"h a failing X fact then a search", []calibration.Record{stage("C0", 1, "review", x("fail")), dryAttempt(failed, 1, "revert")}, []string{calibration.RuleOpenExact, calibration.RuleStageOrder}}, // C0 asked for a review, which is no stop
		{"h the fact passes first", []calibration.Record{stage("C0", 1, "review", x("fail")), stage("C0", 2, "stop", x("pass")), dryAttempt(failed, 1, "revert")}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ruleNames(calibration.CheckSequence(tt.rs, k, experiment.MaxFactors)); !slices.Equal(got, tt.want) {
				t.Errorf("CheckSequence(%s) rules = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
	if got := calibration.CheckSequence([]calibration.Record{good}, k, experiment.MaxFactors); len(got) != 0 {
		t.Fatalf("CheckSequence(the chosen record) = %v, want none: a stop that meets the seven conditions", got)
	}
	dryAssertSevenConditions(t, good, k)
}

// dryAssertSevenConditions checks leg B (i): breaking any one of the seven
// conditions of a stop is reported as that condition, and as nothing under revert.
func dryAssertSevenConditions(t *testing.T, good calibration.Record, k calibration.Known) {
	t.Helper()
	tests := []struct {
		name   string
		mutate func(*calibration.Record)
		want   string
	}{
		{"1 an X fact fails", func(r *calibration.Record) { r.Facts[0].Result = "fail" }, calibration.RuleStopExact},
		{"2 a detector fires", func(r *calibration.Record) { r.Detectors[0].Verdict = "detected" }, calibration.RuleStopDetector},
		{"3 a B fact with a set key fails", func(r *calibration.Record) { r.Facts[1].Result = "fail" }, calibration.RuleStopBSet},
		{"4 a B fact with a working target fails", func(r *calibration.Record) { r.Facts[2].Result = "fail" }, calibration.RuleStopBWorking},
		{"5 a finding is open", func(r *calibration.Record) {
			r.Findings = []calibration.Finding{{Knob: "K-BEL12", Fact: "P02", Measure: "m", Status: "open"}}
		}, calibration.RuleStopFinding},
		{"6 a moved D knob stayed dormant", func(r *calibration.Record) { r.Dormant = []string{"K-BEL12"} }, calibration.RuleStopDormant},
		{"7 no control in the confirm", func(r *calibration.Record) { r.Confirm.Control = false }, calibration.RuleStopControl},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := good
			r.Facts, r.Detectors = slices.Clone(good.Facts), slices.Clone(good.Detectors)
			tt.mutate(&r)
			if got := ruleNames(calibration.CheckSequence([]calibration.Record{r}, k, experiment.MaxFactors)); !slices.Equal(got, []string{tt.want}) {
				t.Errorf("CheckSequence(stop, %s) rules = %v, want [%s]", tt.name, got, tt.want)
			}
			r.Decision = "revert"
			if got := calibration.CheckSequence([]calibration.Record{r}, k, experiment.MaxFactors); len(got) != 0 {
				t.Errorf("CheckSequence(revert, %s) = %v, want none: only a stop is bound by the conditions", tt.name, got)
			}
		})
	}
}
