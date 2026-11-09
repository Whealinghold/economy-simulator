package experiment

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
)

// ResultFormat is the Format of a Result and of its rendering (spec-experiments.md
// 8.2.1). Nothing released has rendered another.
const ResultFormat = 1

// A Result is the experiment's runs as a tree of plain data in canonical order:
// no map, no pointer, no function, no float and no time (spec-experiments.md
// 8.2.1, FR-EXP-061).
type Result struct {
	// Format is ResultFormat.
	Format int
	// Name is the experiment's name.
	Name string
	// Design is the design of the experiment. Render does not print it: the
	// points say what it was, and MainEffects and Elasticity read it to know
	// whether Levels are levels or cells of a spread.
	Design Design
	// Measures are the measure names in order.
	Measures []string
	// Criteria are the criterion names in order.
	Criteria []string
	// Points are the points of the design in point order.
	Points []PointResult

	// first is, per factor with explicit levels, the value of each level's first
	// change (0 for a level with none), which Elasticity reads as x
	// (spec-experiments.md 8.5.3); nil for a range factor.
	first [][]int64
}

// A PointResult is one point of the design and the runs of every variant of it.
type PointResult struct {
	// Index is the point's index in the design.
	Index int
	// Levels is, per factor, the level index, or the cell of a spread.
	Levels []int
	// Values is, per factor, the value of a range factor, else the level index.
	Values []int64
	// Hash is the configuration hash of the first of its runs that built a world,
	// else zero.
	Hash [32]byte
	// Status is unresolved or invalid_config when the first record, in key order,
	// that has either has it, else ok.
	Status Status
	// SameAs is the first earlier point with the same non-zero hash, else -1.
	SameAs int
	// Warnings are the rules of 14.4 that applied to this point.
	Warnings []Warning
	// Variants are the variants of the point; one when the experiment has no branch.
	Variants []VariantResult
}

// A VariantResult is the runs of one variant of one point, over replicas and
// continuations, with their reductions.
type VariantResult struct {
	// Variant is the variant index.
	Variant int
	// Runs are the records in key order: replica, then continuation. A record the
	// assembly did not have is a record with status missing.
	Runs []Record
	// Reductions has one entry per measure, in measure order.
	Reductions []Reduction
	// Pass has one entry per criterion, in criterion order.
	Pass []PassShare
}

// Run validates the experiment, plans it, runs every unit on the caller's
// goroutine, delivers each record to the WithRecordFunc callback and assembles the
// result (spec-experiments.md 5.2.1). It starts no goroutine and holds one world
// at a time. A cancelled Run returns the Result of the complete units before the
// cancellation and the context's error wrapped; the keys not run are missing
// (5.2.3). An invalid experiment returns an empty Result and the error of Validate.
func Run(ctx context.Context, e Spec, opts ...RunOption) (Result, error) {
	recs, err := e.runAll(ctx, opts...)
	if err != nil && ctx.Err() == nil {
		return Result{}, err
	}
	res, aerr := Assemble(e, recs)
	if aerr != nil {
		return Result{}, aerr
	}
	return res, err
}

// Assemble builds the Result of an experiment from its records, in any order
// (spec-experiments.md 5.3.4, FR-EXP-022). A key outside the plan is
// ErrUnknownRun, a key twice with different contents is ErrDuplicateRun, an
// identical duplicate is accepted, and an absent key is a record with status
// missing. A record is trusted: Assemble does not rerun it.
func Assemble(e Spec, recs []Record) (Result, error) {
	if err := e.Validate(); err != nil {
		return Result{}, err
	}
	byKey, err := e.indexRecords(recs)
	if err != nil {
		return Result{}, err
	}
	points, _ := e.pointCount() // Validate has checked it
	res := Result{Format: ResultFormat, Name: e.Name, Design: e.Design, Points: make([]PointResult, points)}
	for _, m := range e.Measures {
		res.Measures = append(res.Measures, m.Name)
	}
	for _, c := range e.Criteria {
		res.Criteria = append(res.Criteria, c.Name)
	}
	res.first = e.firstChanges()
	firstOf := map[[32]byte]int{} // membership and lookup only: never ranged over
	for p := range points {
		res.Points[p] = e.assemblePoint(p, byKey, firstOf)
	}
	return res, nil
}

// firstChanges is Result.first of the experiment.
func (e Spec) firstChanges() [][]int64 {
	out := make([][]int64, len(e.Factors))
	for f, fac := range e.Factors {
		if fac.Path != "" {
			continue
		}
		out[f] = make([]int64, len(fac.Levels))
		for l, lv := range fac.Levels {
			if len(lv.Changes) > 0 {
				out[f][l] = lv.Changes[0].Value
			}
		}
	}
	return out
}

// indexRecords maps the records by key, refusing a key outside the plan and a key
// twice with different contents.
func (e Spec) indexRecords(recs []Record) (map[Key]Record, error) {
	byKey := make(map[Key]Record, len(recs)) // looked up by key, never ranged over
	for _, r := range recs {
		k := r.Key
		if !e.inPlan(k) {
			return nil, fmt.Errorf("%w: point %d replica %d variant %d continuation %d", ErrUnknownRun, k.Point, k.Replica, k.Variant, k.Cont)
		}
		if old, dup := byKey[k]; dup && !equalRecord(old, r) {
			return nil, fmt.Errorf("%w: point %d replica %d variant %d continuation %d", ErrDuplicateRun, k.Point, k.Replica, k.Variant, k.Cont)
		}
		byKey[k] = r
	}
	return byKey, nil
}

// inPlan reports whether k is a key of the plan.
func (e Spec) inPlan(k Key) bool {
	points, _ := e.pointCount() // Validate has checked it
	variants, conts := e.branchShape()
	return k.Point >= 0 && k.Point < points && k.Replica >= 0 && k.Replica < e.Replicas &&
		k.Variant >= 0 && k.Variant < variants && k.Cont >= 0 && k.Cont < conts
}

// assemblePoint builds point p from the records; firstOf maps a hash to the first
// earlier point that has it, and gains this point's hash.
func (e Spec) assemblePoint(p int, byKey map[Key]Record, firstOf map[[32]byte]int) PointResult {
	levels, values, _ := e.point(p) // Validate has checked it
	pr := PointResult{Index: p, Levels: levels, Values: values, Status: StatusOK, SameAs: -1}
	variants, conts := e.branchShape()
	for v := range variants {
		vr := VariantResult{Variant: v}
		for r := range e.Replicas {
			for c := range conts {
				k := Key{Point: p, Replica: r, Variant: v, Cont: c}
				rec, ok := byKey[k]
				if !ok {
					rec = e.baseRecord(k)
					rec.Status = StatusMissing
				}
				vr.Runs = append(vr.Runs, rec)
			}
		}
		pr.Variants = append(pr.Variants, vr)
	}
	pr.settle(firstOf, conts)
	for v := range pr.Variants {
		pr.Variants[v].Reductions, pr.Variants[v].Pass = e.reduceRuns(pr.Variants[v].Runs)
	}
	return pr
}

// settle sets the point's Hash, Status, SameAs and warnings from its runs, taken in
// key order (replica, variant, continuation) so that the choice does not depend on
// the order the records came in.
func (pr *PointResult) settle(firstOf map[[32]byte]int, conts int) {
	for _, r := range pr.inKeyOrder(conts) {
		if pr.Hash == ([32]byte{}) {
			pr.Hash = r.Hash
		}
		if pr.Status == StatusOK && (r.Status == StatusUnresolved || r.Status == StatusInvalidConfig) {
			pr.Status = r.Status
		}
	}
	if pr.Hash == ([32]byte{}) {
		return
	}
	if q, dup := firstOf[pr.Hash]; dup {
		pr.SameAs = q
		pr.Warnings = append(pr.Warnings, Warning{Rule: "duplicate_point", Path: strconv.Itoa(pr.Index)})
		return
	}
	firstOf[pr.Hash] = pr.Index
}

// equalRecord reports whether two records are the same field for field. A
// Snapshot is plain data, so reflect.DeepEqual compares it exactly.
func equalRecord(a, b Record) bool {
	return equalHeader(a, b) && equalReadings(a, b) &&
		a.HasSnapshot == b.HasSnapshot && reflect.DeepEqual(a.Snapshot, b.Snapshot)
}

// equalHeader compares the scalar fields of two records.
func equalHeader(a, b Record) bool {
	return a.Key == b.Key && a.Seed == b.Seed && a.Salt == b.Salt && a.HasSalt == b.HasSalt &&
		a.Hash == b.Hash && a.Digest == b.Digest && a.Status == b.Status && a.FailTick == b.FailTick &&
		a.Detail == b.Detail
}

// equalReadings compares the slices of two records.
func equalReadings(a, b Record) bool {
	return slices.Equal(a.Ticks, b.Ticks) && slices.Equal(a.Final, b.Final) && slices.Equal(a.Pass, b.Pass) &&
		slices.EqualFunc(a.Series, b.Series, slices.Equal[[]Reading])
}

// reading is the final reading of measure m of a record; a record that is shorter
// than the experiment says (a forged one) has an undefined reading.
func reading(r Record, m int) Reading {
	if m < len(r.Final) {
		return r.Final[m]
	}
	return Reading{}
}

// inKeyOrder returns the point's records in key order: replica, then variant, then
// continuation (the order of Key's fields).
func (pr PointResult) inKeyOrder(conts int) []Record {
	if len(pr.Variants) == 0 {
		return nil
	}
	n := len(pr.Variants[0].Runs)
	out := make([]Record, 0, n*len(pr.Variants))
	for i := 0; i < n; i += conts {
		for _, vr := range pr.Variants {
			out = append(out, vr.Runs[i:i+conts]...)
		}
	}
	return out
}
