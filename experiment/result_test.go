package experiment_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/experiment"
)

// TestAssemble_OrderInsensitive verifies that the result of a set of records is
// the same whatever order they arrive in, so a sweep run on eight goroutines or
// three machines renders as a sequential one.
//
// Requirement: FR-EXP-022
func TestAssemble_OrderInsensitive(t *testing.T) {
	t.Parallel()
	e := toySpec(3, experiment.Grid, knob(1, 3, 3))
	var recs []experiment.Record
	for p := range 3 {
		for r := range 3 {
			recs = append(recs, okRec(p, r, uint64(1+r), int64(10000+100*p+r)))
		}
	}
	want := assemble(t, e, recs...)
	shuffled := slices.Clone(recs)
	slices.Reverse(shuffled)
	shuffled[0], shuffled[4] = shuffled[4], shuffled[0]
	got := assemble(t, e, shuffled...)
	if !reflect.DeepEqual(got, want) || experiment.Render(got) != experiment.Render(want) {
		t.Errorf("Assemble(shuffled) differs from Assemble(in order):\n got %s\nwant %s", experiment.Render(got), experiment.Render(want))
	}
}

// TestAssemble_UnknownDuplicateMissing verifies the three errors and the one
// acceptance of Assemble: a key outside the plan, a key twice with different
// contents, an identical duplicate, and an absent key reported as missing.
//
// Requirement: FR-EXP-022
func TestAssemble_UnknownDuplicateMissing(t *testing.T) {
	t.Parallel()
	e := toySpec(2, experiment.Grid, knob(1, 2, 2))
	good := okRec(0, 0, 1, 10100)
	outside := func(mut func(*experiment.Record)) experiment.Record {
		r := okRec(0, 0, 1, 10100)
		mut(&r)
		return r
	}
	tests := []struct {
		name string
		recs []experiment.Record
		want error
	}{
		{"point past the plan", []experiment.Record{outside(func(r *experiment.Record) { r.Key.Point = 2 })}, experiment.ErrUnknownRun},
		{"negative replica", []experiment.Record{outside(func(r *experiment.Record) { r.Key.Replica = -1 })}, experiment.ErrUnknownRun},
		{"variant of an unbranched experiment", []experiment.Record{outside(func(r *experiment.Record) { r.Key.Variant = 1 })}, experiment.ErrUnknownRun},
		{"same key with another digest", []experiment.Record{good, outside(func(r *experiment.Record) { r.Digest[0] = 7 })}, experiment.ErrDuplicateRun},
		{"same key with another reading", []experiment.Record{good, okRec(0, 0, 1, 10200)}, experiment.ErrDuplicateRun},
		{"identical duplicate", []experiment.Record{good, good}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := experiment.Assemble(e, tt.recs)
			if !errors.Is(err, tt.want) || (tt.want == nil && err != nil) {
				t.Errorf("Assemble(%s) error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
	res := assemble(t, e, good)
	runs := res.Points[1].Variants[0].Runs
	if len(runs) != 2 || runs[0].Status != experiment.StatusMissing || runs[0].FailTick != -1 {
		t.Errorf("absent point runs = %+v, want two records with status missing and fail tick -1", runs)
	}
	red := res.Points[0].Variants[0].Reductions[0]
	if red.Planned != 2 || red.Used != 1 || red.Missing != 1 {
		t.Errorf("reduction of one run present of two = %+v, want planned 2 used 1 missing 1", red)
	}
	if _, err := experiment.Assemble(experiment.Spec{}, nil); err == nil {
		t.Errorf("Assemble(zero Spec) error = nil, want the error of Validate")
	}
}

// TestAssemble_SameAsAndDuplicatePoint verifies that a point whose worlds hash like
// an earlier point's carries SameAs and the duplicate_point warning, and that a
// point that built no world (hash zero) matches nothing.
//
// Requirement: FR-EXP-063, FR-EXP-071
func TestAssemble_SameAsAndDuplicatePoint(t *testing.T) {
	t.Parallel()
	e := toySpec(1, experiment.Grid, knob(1, 4, 4))
	mk := func(p int, h byte) experiment.Record {
		r := okRec(p, 0, 1, 10000)
		r.Hash = [32]byte{h}
		return r
	}
	res := assemble(t, e, mk(0, 5), mk(1, 6), mk(2, 5), mk(3, 0))
	var sameAs []int
	for _, p := range res.Points {
		sameAs = append(sameAs, p.SameAs)
	}
	if want := []int{-1, -1, 0, -1}; !slices.Equal(sameAs, want) {
		t.Errorf("SameAs of the points = %v, want %v (point 2 hashes like point 0; point 3 has no hash)", sameAs, want)
	}
	if got := res.Points[2].Warnings; !reflect.DeepEqual(got, []experiment.Warning{{Rule: "duplicate_point", Path: "2"}}) {
		t.Errorf("Warnings of point 2 = %+v, want one duplicate_point at path 2", got)
	}
	if !strings.Contains(experiment.Render(res), "warning duplicate_point 2\n") {
		t.Errorf("Render lacks the duplicate_point line:\n%s", experiment.Render(res))
	}
}

// TestAssemble_PointStatusAndHash verifies the derivation of a point's Status
// (the first record in key order that is unresolved or invalid_config, else ok)
// and Hash (the first non-zero one).
//
// Requirement: FR-EXP-063
func TestAssemble_PointStatusAndHash(t *testing.T) {
	t.Parallel()
	e := toySpec(2, experiment.Grid, knob(1, 2, 2))
	bad := failedRec(0, 1, 2, experiment.StatusInvalidConfig)
	bad.Hash = [32]byte{}
	first := failedRec(0, 0, 1, experiment.StatusWorld)
	first.Hash = [32]byte{}
	res := assemble(t, e, first, bad, failedRec(1, 0, 1, experiment.StatusWorld), okRec(1, 1, 2, 1))
	if got := res.Points[0]; got.Status != experiment.StatusInvalidConfig || got.Hash != ([32]byte{}) {
		t.Errorf("point 0 status, hash = %s, %x, want invalid_config and zero (no record built a world)", got.Status, got.Hash)
	}
	if got := res.Points[1]; got.Status != experiment.StatusOK || got.Hash != ([32]byte{9, 1}) {
		t.Errorf("point 1 status, hash = %s, %x, want ok and the hash of its first record", got.Status, got.Hash)
	}
}

// worked199 is the result of spec-experiments.md 19.9 with the invented hashes
// and digests of the example.
func worked199(t testing.TB) experiment.Result {
	t.Helper()
	e := toySpec(2, experiment.Grid, knob(7000, 9000, 2))
	e.Name = "example"
	e.Measures[0].Name = "price_range_bp"
	e.Criteria[0].Measure = "price_range_bp"
	rec := func(p, r int, v int64, hash, digest string) experiment.Record {
		x := okRec(p, r, uint64(1+r), v)
		x.Hash, x.Digest = hash8(t, hash), hash8(t, digest)
		return x
	}
	failed := failedRec(1, 1, 2, experiment.StatusInvalidTrade)
	failed.Hash = hash8(t, "c7f20b19e84a5d06")
	failed.FailTick = 212
	failed.Detail = "step 212: a clearing returned an invalid list of trades for a market"
	return assemble(t, e,
		rec(0, 0, 10400, "5a1c0e44b27d9f31", "91d3a7c20f4be816"),
		rec(0, 1, 10900, "5a1c0e44b27d9f31", "0be4417f6c93d2a5"),
		rec(1, 0, 11800, "c7f20b19e84a5d06", "4e9d02b6a17c35f9"),
		failed)
}

// TestRender_Worked199 verifies that Render of the result of spec-experiments.md
// 19.9 is that section's text exactly: the lines, their order, the first 8 bytes of
// a hash in hex, the dashes of a failed run, the reductions of one and two runs.
//
// Requirement: FR-EXP-060
func TestRender_Worked199(t *testing.T) {
	t.Parallel()
	want := `experiment example format 1 points 2 measures 1 criteria 1
measure 0 price_range_bp
point 0 hash 5a1c0e44b27d9f31 status ok same_as -1 levels 0 values 7000
variant 0
run replica 0 cont 0 seed 1 salt - status ok fail_tick -1 digest 91d3a7c20f4be816
final price_range_bp=10400
verdict calm=pass
run replica 1 cont 0 seed 2 salt - status ok fail_tick -1 digest 0be4417f6c93d2a5
final price_range_bp=10900
verdict calm=pass
reduce price_range_bp planned 2 used 2 failed 0 undefined 0 missing 0 min 10400 max 10900 mean 10650 std 354 q1000=10400 q2500=10400 q5000=10400 q7500=10900 q9000=10900
pass calm passes 2 planned 2 share 10000
point 1 hash c7f20b19e84a5d06 status ok same_as -1 levels 1 values 9000
variant 0
run replica 0 cont 0 seed 1 salt - status ok fail_tick -1 digest 4e9d02b6a17c35f9
final price_range_bp=11800
verdict calm=fail
run replica 1 cont 0 seed 2 salt - status invalid_trade fail_tick 212 digest -
detail step 212: a clearing returned an invalid list of trades for a market
final price_range_bp=-
verdict calm=fail
reduce price_range_bp planned 2 used 1 failed 1 undefined 0 missing 0 min 11800 max 11800 mean 11800 std - q1000=11800 q2500=11800 q5000=11800 q7500=11800 q9000=11800
pass calm passes 0 planned 2 share 0
`
	if got := experiment.Render(worked199(t)); got != want {
		t.Errorf("Render(worked 19.9) =\n%s\nwant\n%s", got, want)
	}
}

// TestRender_DashRules verifies the dash rules and the small forms of Render: a
// dash for the statistics of an empty group, for std below two runs, for a zero
// hash and digest, for an undefined reading and for an empty list of levels; a
// salt in decimal; a detail line with its line breaks replaced; the window lines
// of a kept series.
//
// Requirement: FR-EXP-060
func TestRender_DashRules(t *testing.T) {
	t.Parallel()
	e := toySpec(2, experiment.Grid)
	e.Keep.Series = true
	e.Branch = &experiment.Branch{At: 5, Variants: []experiment.Variant{{Label: "none"}}, Continuations: 2, Salt0: 100}
	mk := func(r, c int, mut func(*experiment.Record)) experiment.Record {
		x := okRec(0, r, uint64(1+r), 10100)
		x.Key.Cont, x.Salt, x.HasSalt = c, 100+uint64(c), true
		mut(&x)
		return x
	}
	undefined := func(x *experiment.Record) { x.Final[0] = experiment.Reading{}; x.Pass[0] = false }
	windows := func(x *experiment.Record) {
		x.Ticks, x.Series = []int64{5, 10}, [][]experiment.Reading{{{Value: 7, OK: true}}, {{}}}
	}
	res := assemble(t, e,
		mk(0, 0, windows), mk(0, 1, undefined),
		mk(1, 0, func(x *experiment.Record) {
			x.Status, x.Detail, x.Digest, x.Hash = experiment.StatusStep, "a\nb\r\nc", [32]byte{}, [32]byte{}
		}),
		mk(1, 1, undefined))
	got := experiment.Render(res)
	for _, want := range []string{
		"point 0 hash 0900000000000000 status ok same_as -1 levels - values -\n",
		"run replica 0 cont 1 seed 1 salt 101 status ok fail_tick -1 digest 0100000000000000\nfinal m=-\nverdict calm=fail\n",
		"run replica 1 cont 0 seed 2 salt 100 status step fail_tick -1 digest -\ndetail a b c\n",
		"final m=10100\nverdict calm=pass\nwindow 5 m=7\nwindow 10 m=-\n",
		"reduce m planned 4 used 1 failed 1 undefined 2 missing 0 min 10100 max 10100 mean 10100 std - q1000=10100",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Render lacks %q:\n%s", want, got)
		}
	}
	none := assemble(t, toySpec(1, experiment.Grid), failedRec(0, 0, 1, experiment.StatusWorld))
	if want := "reduce m planned 1 used 0 failed 1 undefined 0 missing 0 min - max - mean - std - q1000=- q2500=- q5000=- q7500=- q9000=-\n"; !strings.Contains(experiment.Render(none), want) {
		t.Errorf("Render of a group with no used run lacks %q:\n%s", want, experiment.Render(none))
	}
	if experiment.Render(res) != got {
		t.Errorf("Render twice differs")
	}
}

// TestResult_PlainData verifies, by reflection over the type of Result, that
// nothing it holds is a map, pointer, function, channel, interface or float: it is
// plain data a consumer can marshal on another goroutine (architecture.md 12.4).
//
// Requirement: FR-EXP-061
func TestResult_PlainData(t *testing.T) {
	t.Parallel()
	seen := map[reflect.Type]bool{}
	var walk func(path string, ty reflect.Type)
	walk = func(path string, ty reflect.Type) {
		if seen[ty] {
			return
		}
		seen[ty] = true
		switch ty.Kind() {
		case reflect.Map, reflect.Pointer, reflect.Func, reflect.Chan, reflect.Interface, reflect.UnsafePointer,
			reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
			t.Errorf("%s has kind %s, want plain data", path, ty.Kind())
		case reflect.Slice, reflect.Array:
			walk(path+"[]", ty.Elem())
		case reflect.Struct:
			for i := range ty.NumField() {
				f := ty.Field(i)
				walk(path+"."+f.Name, f.Type)
			}
		}
	}
	walk("Result", reflect.TypeFor[experiment.Result]())
	if len(seen) < 10 {
		t.Errorf("walked %d types, want the whole tree", len(seen))
	}
}
