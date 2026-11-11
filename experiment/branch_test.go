package experiment_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/experiment"
)

// branched is a short pack-one experiment with a branch at tick 2 of 4: the
// tests change what they test.
func branched(t testing.TB, variants int, conts int) experiment.Spec {
	t.Helper()
	e := packSpec(t)
	e.Ticks = 4
	e.Branch = &experiment.Branch{At: 2, Variants: make([]experiment.Variant, variants), Continuations: conts, Salt0: 5}
	return e
}

// unbranched returns e without its branch: the uninterrupted run of the same key.
func unbranched(e experiment.Spec) experiment.Spec {
	e.Branch = nil
	return e
}

// TestBranch_BurnInOnceRestoreEach verifies that Options is called once for the
// burn-in, with the key of branch (0, 0), and once for each restore with the
// key of that branch: 1 + V*C calls for V variants and C continuations.
//
// Requirement: FR-EXP-030, FR-EXP-024
func TestBranch_BurnInOnceRestoreEach(t *testing.T) {
	short(t)
	e := branched(t, 2, 3)
	var keys []experiment.Key
	e.Options = func(k experiment.Key) []economysim.Option { keys = append(keys, k); return nil }
	recs := runSweep(t, e)
	if len(recs) != 6 || len(keys) != 7 {
		t.Fatalf("records %d, Options calls %d, want 6 and 1 + 2*3 = 7", len(recs), len(keys))
	}
	want := []experiment.Key{{}, {}, {Cont: 1}, {Cont: 2}, {Variant: 1}, {Variant: 1, Cont: 1}, {Variant: 1, Cont: 2}}
	if !slices.Equal(keys, want) {
		t.Errorf("Options keys = %v, want %v: the burn-in with the key of (0, 0), then each restore", keys, want)
	}
}

// TestBranch_NoStepsNoReseedEqualsUninterrupted verifies the equality that makes
// branching safe (spec-experiments.md 6.4): with no steps and no reseed every
// branch has the digest, the readings and the series of the same key run from
// tick 0, also with a window straddling the branch point and with script steps
// below and at it. Cites FR-CHK-005, FR-CHK-050 and FR-WLD-063.
//
// Requirement: FR-EXP-031, FR-EXP-014
func TestBranch_NoStepsNoReseedEqualsUninterrupted(t *testing.T) {
	short(t)
	tests := []struct {
		name  string
		every int64
		steps []int64
	}{
		{"one window", 0, nil},
		{"window ends at the branch point", 2, nil},
		{"window straddles the branch point", 3, nil},
		{"steps below and at the branch point", 3, []int64{1, 2}},
	}
	for _, tt := range tests {
		for _, conts := range []int{0, 1} {
			e := branched(t, 2, conts)
			e.Every, e.Keep.Series = tt.every, true
			for _, tick := range tt.steps {
				e.Script = append(e.Script, experiment.Step{Tick: tick, Do: func(*economysim.World) error { return nil }})
			}
			want := runSweep(t, unbranched(e))[0]
			for _, r := range runSweep(t, e) {
				if r.Digest != want.Digest || !reflect.DeepEqual(r.Final, want.Final) || !reflect.DeepEqual(r.Series, want.Series) || !slices.Equal(r.Ticks, want.Ticks) {
					t.Errorf("%s, %d continuations, key %+v: branch %+v, uninterrupted %+v", tt.name, conts, r.Key, r, want)
				}
				if r.HasSalt {
					t.Errorf("%s: key %+v has a salt with %d continuations, want none", tt.name, r.Key, conts)
				}
			}
		}
	}
}

// TestBranch_ReseedByContinuation verifies the salts: with 3 continuations the
// salts are Salt0 to Salt0+2, the same for every variant, the digests differ
// across continuations and are equal across the two no-step variants of one
// continuation; with 2 continuations the salts are Salt0 and Salt0+1.
//
// Requirement: FR-EXP-032
func TestBranch_ReseedByContinuation(t *testing.T) {
	short(t)
	recs := runSweep(t, branched(t, 2, 3))
	for _, r := range recs {
		if !r.HasSalt || r.Salt != 5+uint64(r.Key.Cont) {
			t.Errorf("key %+v: salt %d (has %v), want %d", r.Key, r.Salt, r.HasSalt, 5+r.Key.Cont)
		}
	}
	for c := 0; c < 3; c++ {
		a, b := recs[c], recs[3+c]
		if a.Digest != b.Digest {
			t.Errorf("continuation %d: the two no-step variants have digests %x, %x, want equal", c, a.Digest, b.Digest)
		}
	}
	if recs[0].Digest == recs[1].Digest || recs[1].Digest == recs[2].Digest || recs[0].Digest == recs[2].Digest {
		t.Errorf("continuations share a digest: %x %x %x", recs[0].Digest, recs[1].Digest, recs[2].Digest)
	}
	two := runSweep(t, branched(t, 1, 2))
	if two[0].Salt != 5 || two[1].Salt != 6 || !two[1].HasSalt {
		t.Errorf("2 continuations salts = %d, %d, want 5, 6", two[0].Salt, two[1].Salt)
	}
}

// TestBranch_BurnInFailureInherited verifies that a world that fails before the
// branch point fails every key of the unit with the burn-in's status and tick,
// the hash of the world, and that no restore is made.
//
// Requirement: FR-EXP-033
func TestBranch_BurnInFailureInherited(t *testing.T) {
	short(t)
	on := false
	e := branched(t, 2, 2)
	e.Ticks, e.Every, e.Keep.Series = 6, 1, true
	e.Branch.At = 4
	e.Base = append(slices.Clone(e.Base), faultyDoc)
	built := 0
	inner := faultyOn(&on)
	e.Options = func(k experiment.Key) []economysim.Option { built++; return inner(k) }
	e.Script = []experiment.Step{{Tick: 1, Do: func(*economysim.World) error { on = true; return nil }}}
	recs := runSweep(t, e)
	if built != 1 || len(recs) != 4 {
		t.Fatalf("worlds built %d, records %d, want 1 and 4", built, len(recs))
	}
	for _, r := range recs {
		if r.Status != experiment.StatusInvalidTrade || r.FailTick != 1 || r.Hash == [32]byte{} || r.Digest != [32]byte{} || !slices.Equal(r.Ticks, []int64{1}) {
			t.Errorf("key %+v: %+v, want invalid_trade at tick 1 with the hash and the first window", r.Key, r)
		}
	}
}

// TestBranch_VariantStepBelowAtIsErrBadBranch verifies that a variant step
// below the branch point is refused before any run.
//
// Requirement: FR-EXP-034
func TestBranch_VariantStepBelowAtIsErrBadBranch(t *testing.T) {
	t.Parallel()
	e := branched(t, 1, 0)
	e.Branch.Variants[0].Steps = []experiment.Step{step(1)}
	if _, err := experiment.Plan(e); !errors.Is(err, experiment.ErrBadBranch) {
		t.Errorf("Plan() = %v, want ErrBadBranch", err)
	}
}

// TestBranch_OptionsWithSeedGivesWorld verifies that an Options that returns
// WithSeed fails every branch with world and ErrOptionContext in the detail
// (a restore refuses it), and that the burn-in's hash is kept.
//
// Requirement: FR-EXP-024
func TestBranch_OptionsWithSeedGivesWorld(t *testing.T) {
	short(t)
	e := branched(t, 2, 0)
	e.Options = func(experiment.Key) []economysim.Option { return []economysim.Option{economysim.WithSeed(7)} }
	for _, r := range runSweep(t, e) {
		if r.Status != experiment.StatusWorld || !strings.Contains(r.Detail, economysim.ErrOptionContext.Error()) || r.Hash == [32]byte{} || r.Digest != [32]byte{} {
			t.Errorf("key %+v: %+v, want world with ErrOptionContext", r.Key, r)
		}
	}
}

// TestBranch_ConfigCannotChange verifies that a restore under another
// configuration is refused: variant 1's Options changes WithTargetSpread (a datum
// of the configuration hash), so only that branch fails, as world with
// ErrCheckpointConfig, and variant 0 runs.
//
// Requirement: FR-EXP-030
func TestBranch_ConfigCannotChange(t *testing.T) {
	short(t)
	e := branched(t, 2, 0)
	e.Options = func(k experiment.Key) []economysim.Option {
		if k.Variant == 1 {
			return []economysim.Option{economysim.WithTargetSpread(300)}
		}
		return nil
	}
	recs := runSweep(t, e)
	if recs[0].Status != experiment.StatusOK {
		t.Errorf("variant 0 = %+v, want ok", recs[0])
	}
	if r := recs[1]; r.Status != experiment.StatusWorld || !strings.Contains(r.Detail, economysim.ErrCheckpointConfig.Error()) {
		t.Errorf("variant 1 = %+v, want world with ErrCheckpointConfig", r)
	}
}

// TestBranch_ShockVariantDiffersFromNone verifies that a variant that applies a
// yield shock at the branch point reaches another digest than the none variant,
// which equals the uninterrupted run.
//
// Requirement: FR-EXP-030, FR-EXP-031
func TestBranch_ShockVariantDiffersFromNone(t *testing.T) {
	short(t)
	e := branched(t, 2, 0)
	e.Branch.Variants[1].Steps = []experiment.Step{{Tick: 2, Label: "shock", Do: func(w *economysim.World) error {
		return w.SetYieldShocks([]economysim.YieldShock{{Region: 0, Recipe: 0, BP: 3_000}})
	}}}
	recs := runSweep(t, e)
	want := runSweep(t, unbranched(e))[0]
	if recs[0].Digest != want.Digest || recs[1].Digest == want.Digest || recs[1].Status != experiment.StatusOK {
		t.Errorf("digests none %x variant %x uninterrupted %x, want none equal and the variant different", recs[0].Digest, recs[1].Digest, want.Digest)
	}
}

// TestBranch_SeriesAndFork verifies the hooks across a branch: the series holds
// the burn-in's windows first and equals the uninterrupted run's; Fork is called
// for every branch including (0, 0) with equal keys, after that branch's Options
// and before its first window call; a window open at the branch point is not
// opened again; and one world is held at a time (the Fn calls of the burn-in
// come first, then those of each branch in turn, never interleaved).
//
// Requirement: FR-EXP-016, FR-EXP-030, NFR-EXP-004
func TestBranch_SeriesAndFork(t *testing.T) {
	short(t)
	var log []string
	e := branched(t, 2, 0)
	e.Every, e.Keep.Series = 3, true // windows [0,3) and [3,4): the first straddles the branch point at 2
	e.Options = func(k experiment.Key) []economysim.Option {
		log = append(log, fmt.Sprintf("options %+v", k))
		return nil
	}
	e.Measures = []experiment.Measure{{
		Name: "m",
		Begin: func(k experiment.Key, _ economysim.Snapshot, from int64, span bool) {
			log = append(log, fmt.Sprintf("begin %+v %d %v", k, from, span))
		},
		Fork: func(from, to experiment.Key) { log = append(log, fmt.Sprintf("fork %+v>%+v", from, to)) },
		Fn: func(s experiment.Sample) (int64, bool, error) {
			log = append(log, fmt.Sprintf("fn %+v %d-%d %v", s.Key, s.From, s.To, s.Span))
			return s.Close.Households, true, nil
		},
	}}
	recs := runSweep(t, e)
	want := runSweep(t, unbranched(e))[0]
	for _, r := range recs {
		if !reflect.DeepEqual(r.Series, want.Series) || !slices.Equal(r.Ticks, []int64{3, 4}) {
			t.Errorf("key %+v: series %v ticks %v, want the uninterrupted %v %v", r.Key, r.Series, r.Ticks, want.Series, want.Ticks)
		}
	}
	k0, k1 := "{Point:0 Replica:0 Variant:0 Cont:0}", "{Point:0 Replica:0 Variant:1 Cont:0}"
	wantLog := []string{
		"options " + k0,
		"begin " + k0 + " 0 false", "begin " + k0 + " 0 true",
		"options " + k0, "fork " + k0 + ">" + k0,
		"fn " + k0 + " 0-3 false", "begin " + k0 + " 3 false", "fn " + k0 + " 3-4 false", "fn " + k0 + " 0-4 true",
		"options " + k1, "fork " + k0 + ">" + k1,
		"fn " + k1 + " 0-3 false", "begin " + k1 + " 3 false", "fn " + k1 + " 3-4 false", "fn " + k1 + " 0-4 true",
	}
	// The uninterrupted run is in the log too, after the branches: cut it off.
	if len(log) < len(wantLog) || !slices.Equal(log[:len(wantLog)], wantLog) {
		t.Errorf("hook calls =\n%s\nwant it to begin\n%s", strings.Join(log, "\n"), strings.Join(wantLog, "\n"))
	}
}

// TestBranch_StepAtTheBranchPoint verifies that, with a window ending at the
// branch point, the burn-in closes it and the branch then applies the steps of
// that tick before it opens the next window, as an uninterrupted run does
// (close, steps, open: FR-EXP-019).
//
// Requirement: FR-EXP-019, FR-EXP-030
func TestBranch_StepAtTheBranchPoint(t *testing.T) {
	short(t)
	var log []call
	e := branched(t, 1, 0)
	e.Every = 2
	m := recorder(&log)
	e.Measures = []experiment.Measure{m}
	do := func(*economysim.World) error { log = append(log, call{kind: "do"}); return nil }
	e.Script = []experiment.Step{{Tick: 2, Do: do}}
	e.Branch.Variants[0].Steps = []experiment.Step{{Tick: 2, Do: do}}
	runSweep(t, e)
	var kinds []string
	for _, c := range log {
		if c.kind == "do" || (c.kind == "fn" && c.to == 2 && !c.span) || (c.kind == "begin" && c.from == 2 && !c.span) {
			kinds = append(kinds, c.kind)
		}
	}
	if want := []string{"fn", "do", "do", "begin"}; !slices.Equal(kinds, want) {
		t.Errorf("hooks at the branch point = %v, want %v: close, the script's step and the variant's, open", kinds, want)
	}
}

// TestBranch_FailuresAreEachKeys verifies the status of the failures around a
// branch: a world that cannot be built, and a checkpoint that cannot be taken at
// the branch point, are the failure of every key of the unit; a step error and a
// failing tick of one variant are that key's failure alone, the other variants
// run, and the failing variant keeps the burn-in's windows with Keep.Series.
//
// Requirement: FR-EXP-033, FR-EXP-050
func TestBranch_FailuresAreEachKeys(t *testing.T) {
	short(t)
	nilClearing := branched(t, 2, 0)
	nilClearing.Base = append(slices.Clone(nilClearing.Base), []byte("[world]\nclearing = \"nobody\"\n"))
	nilClearing.Options = func(experiment.Key) []economysim.Option {
		return []economysim.Option{economysim.WithClearing("nobody", nil)}
	}
	for _, r := range runSweep(t, nilClearing) {
		if r.Status != experiment.StatusWorld || r.FailTick != 0 {
			t.Errorf("unbuildable world, key %+v: %+v, want world at tick 0", r.Key, r)
		}
	}

	noSave := branched(t, 2, 0)
	noSave.Base = append(slices.Clone(noSave.Base), []byte("[world]\nclearing = \"stateful\"\n"))
	noSave.Options = func(experiment.Key) []economysim.Option {
		return []economysim.Option{economysim.WithClearing("stateful", failingSave{})}
	}
	for _, r := range runSweep(t, noSave) {
		if r.Status != experiment.StatusCheckpoint || r.FailTick != 2 || r.Hash == [32]byte{} {
			t.Errorf("checkpoint at the branch point, key %+v: %+v, want checkpoint at tick 2 with the hash", r.Key, r)
		}
	}

	on := false
	e := branched(t, 3, 0)
	e.Every, e.Keep.Series = 1, true
	e.Base = append(slices.Clone(e.Base), faultyDoc)
	e.Options = faultyOn(&on)
	e.Branch.Variants[1].Steps = []experiment.Step{{Tick: 3, Do: func(*economysim.World) error { return errors.New("variant step refuses") }}}
	e.Branch.Variants[2].Steps = []experiment.Step{{Tick: 3, Do: func(*economysim.World) error { on = true; return nil }}}
	recs := runSweep(t, e)
	if recs[0].Status != experiment.StatusOK {
		t.Errorf("variant 0 = %+v, want ok", recs[0])
	}
	if r := recs[1]; r.Status != experiment.StatusScript || r.FailTick != 3 || !slices.Equal(r.Ticks, []int64{1, 2, 3}) {
		t.Errorf("variant 1 = %+v, want script at tick 3 with the windows 1, 2 and 3 closed before", r)
	}
	if r := recs[2]; r.Status != experiment.StatusInvalidTrade || r.FailTick != 3 || r.Digest != [32]byte{} {
		t.Errorf("variant 2 = %+v, want invalid_trade at tick 3", r)
	}
}

// TestBranch_CancelDiscardsTheUnit verifies that a context cancelled during the
// burn-in, or during a branch, discards the whole unit: no records and the
// wrapped context error, never a record cut at the tick a signal arrived.
//
// Requirement: FR-EXP-052
func TestBranch_CancelDiscardsTheUnit(t *testing.T) {
	short(t)
	for _, inBranch := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		e := branched(t, 2, 0)
		e.Every = 1
		e.Measures = []experiment.Measure{{Name: "m", Fn: func(s experiment.Sample) (int64, bool, error) {
			if (s.Key.Variant == 1) == inBranch && s.To > 2 == inBranch {
				cancel()
			}
			return 1, true, nil
		}}}
		units, err := experiment.Plan(e)
		if err != nil {
			t.Fatal(err)
		}
		recs, err := experiment.RunUnit(ctx, e, units[0])
		if recs != nil || !errors.Is(err, context.Canceled) {
			t.Errorf("cancel in the branch %v: RunUnit() = %v, %v, want no records and context.Canceled", inBranch, recs, err)
		}
		cancel()
	}
}

// TestRunUnit_KeepSnapshotAndInvalidConfig verifies Keep.Snapshot (the final
// snapshot is the record's, at the last tick), and that ExecuteUnit refuses a
// Config that fails validation as invalid_config with no world.
//
// Requirement: FR-EXP-001, FR-EXP-050
func TestRunUnit_KeepSnapshotAndInvalidConfig(t *testing.T) {
	short(t)
	e := packSpec(t)
	e.Keep.Snapshot = true
	r := runSweep(t, e)[0]
	if !r.HasSnapshot || r.Snapshot.Tick != e.Ticks || r.Snapshot.Households != 600 {
		t.Errorf("snapshot of the record = tick %d households %d (has %v), want tick %d and 600", r.Snapshot.Tick, r.Snapshot.Households, r.HasSnapshot, e.Ticks)
	}
	units, _ := experiment.Plan(e)
	recs, err := experiment.ExecuteUnit(context.Background(), economysim.Config{}, e, units[0])
	if err != nil || len(recs) != 1 || recs[0].Status != experiment.StatusInvalidConfig || recs[0].Hash != [32]byte{} {
		t.Errorf("ExecuteUnit(Config{}) = %+v, %v, want invalid_config with no world", recs, err)
	}
	bad := units[0]
	bad.Keys = nil
	if _, err := experiment.RunUnit(context.Background(), e, bad); !errors.Is(err, experiment.ErrUnknownRun) {
		t.Errorf("RunUnit() of a unit with no keys = %v, want ErrUnknownRun", err)
	}
	if _, err := experiment.RunUnit(context.Background(), experiment.Spec{}, units[0]); !errors.Is(err, experiment.ErrBadSchedule) {
		t.Errorf("RunUnit() of an invalid experiment = %v, want its Validate error", err)
	}
}
