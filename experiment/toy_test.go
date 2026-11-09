package experiment_test

import (
	"encoding/hex"
	"testing"

	"github.com/Whealinghold/economy-sim/experiment"
)

// toyMeasure is the one measure of the hand-built results below. Its function is
// never called: the tests assemble records by hand, so no world is built.
const toyMeasure = "m"

// toySpec is an experiment of the given design over the given factors with one
// measure "m" and, when band is true, the criterion "calm" (at most 11 000).
func toySpec(replicas int, d experiment.Design, factors ...experiment.Factor) experiment.Spec {
	e := experiment.Spec{
		Name:     "toy",
		Base:     [][]byte{[]byte("base")},
		Factors:  factors,
		Design:   d,
		Replicas: replicas,
		Seed0:    1,
		Ticks:    10,
		Measures: []experiment.Measure{{Name: toyMeasure, Fn: func(experiment.Sample) (int64, bool, error) { return 0, false, nil }}},
		Criteria: []experiment.Criterion{{Name: "calm", Measure: toyMeasure, Op: experiment.AtMost, A: 11000}},
	}
	return e
}

// knob is a range factor of steps values from min to max over the path world.knob.
func knob(min, max int64, steps int) experiment.Factor {
	return experiment.Factor{Name: "knob", Path: "world.knob", Min: min, Max: max, Steps: steps}
}

// levels is an explicit factor of n levels, each a change of its own path.
func levels(path string, n int) experiment.Factor {
	f := experiment.Factor{Name: path}
	for range n {
		f.Levels = append(f.Levels, level(path))
	}
	return f
}

// okRec is a finished record of key (p, r) with seed s and one reading.
func okRec(p, r int, seed uint64, v int64) experiment.Record {
	rec := experiment.Record{
		Key:      experiment.Key{Point: p, Replica: r},
		Seed:     seed,
		Status:   experiment.StatusOK,
		FailTick: -1,
		Final:    []experiment.Reading{{Value: v, OK: true}},
		Pass:     []bool{v <= 11000},
		Digest:   [32]byte{1, byte(p), byte(r)},
		Hash:     [32]byte{9, byte(p)},
	}
	return rec
}

// failedRec is a record of key (p, r) that failed with the status.
func failedRec(p, r int, seed uint64, st experiment.Status) experiment.Record {
	return experiment.Record{
		Key:      experiment.Key{Point: p, Replica: r},
		Seed:     seed,
		Status:   st,
		FailTick: 5,
		Final:    []experiment.Reading{{}},
		Pass:     []bool{false},
		Hash:     [32]byte{9, byte(p)},
	}
}

// assemble assembles the records of e, failing the test on an error.
func assemble(t testing.TB, e experiment.Spec, recs ...experiment.Record) experiment.Result {
	t.Helper()
	res, err := experiment.Assemble(e, recs)
	if err != nil {
		t.Fatalf("Assemble() error = %v, want nil", err)
	}
	return res
}

// column assembles one record per (point, replica) from values[p][r], seeded by
// the SameSeeds rule (Seed0 + r), and returns the result.
func column(t testing.TB, e experiment.Spec, values [][]int64) experiment.Result {
	t.Helper()
	var recs []experiment.Record
	for p, row := range values {
		for r, v := range row {
			recs = append(recs, okRec(p, r, e.Seed0+uint64(r), v))
		}
	}
	return assemble(t, e, recs...)
}

// hash8 is a 32-byte hash whose first 8 bytes are the 16 hex digits.
func hash8(t testing.TB, digits string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(digits)
	if err != nil || len(b) != 8 {
		t.Fatalf("hash8(%q): want 16 hex digits, got %v", digits, err)
	}
	var h [32]byte
	copy(h[:], b)
	return h
}
