package experiment_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/experiment"
)

// valid returns a small experiment that passes Validate: the tests break one
// rule of it at a time.
func valid() experiment.Spec {
	return experiment.Spec{
		Name:     "t",
		Base:     [][]byte{[]byte("base")},
		Replicas: 2,
		Seed0:    1,
		Ticks:    10,
		Measures: []experiment.Measure{{Name: "m", Fn: func(experiment.Sample) (int64, bool, error) { return 1, true, nil }}},
	}
}

func step(tick int64) experiment.Step {
	return experiment.Step{Tick: tick, Do: func(*economysim.World) error { return nil }}
}

func level(paths ...string) experiment.Level {
	var l experiment.Level
	for _, p := range paths {
		l.Changes = append(l.Changes, experiment.Change{Path: p, Value: 1})
	}
	return l
}

// twoLevels is an explicit factor over one path.
func twoLevels(path string) experiment.Factor {
	return experiment.Factor{Levels: []experiment.Level{level(path), level(path)}}
}

// onePack returns the layers of pack one, the reference economy of the root's
// tests (support_test.go packOne), read from the shared testdata by relative
// path (testing.md 10.4).
func onePack(t testing.TB) [][]byte {
	t.Helper()
	var docs [][]byte
	for _, name := range []string{"base.toml", "one.toml", "warm-one.toml"} {
		b, err := os.ReadFile(filepath.Join("..", "testdata", "chains", name))
		if err != nil {
			t.Fatalf("read pack layer %s: %v", name, err)
		}
		docs = append(docs, b)
	}
	return docs
}

// households and money are snapshot-only measures: the closing snapshot's
// households and money.
var households = experiment.Measure{Name: "households", Fn: func(s experiment.Sample) (int64, bool, error) {
	return s.Close.Households, true, nil
}}

var money = experiment.Measure{Name: "money", Fn: func(s experiment.Sample) (int64, bool, error) {
	return int64(s.Close.Money), true, nil
}}

// packSpec is a short experiment over pack one: three ticks, one replica, the
// two snapshot measures. The tests change what they test.
func packSpec(t testing.TB) experiment.Spec {
	t.Helper()
	return experiment.Spec{
		Name:     "pack",
		Base:     onePack(t),
		Replicas: 1,
		Seed0:    1,
		Ticks:    3,
		Measures: []experiment.Measure{households, money},
	}
}

// runSweep runs every unit of Plan in order on this goroutine, as Run does, and
// returns the records.
func runSweep(t testing.TB, e experiment.Spec) []experiment.Record {
	t.Helper()
	units, err := experiment.Plan(e)
	if err != nil {
		t.Fatalf("Plan() error = %v, want nil", err)
	}
	var out []experiment.Record
	for _, u := range units {
		recs, err := experiment.RunUnit(context.Background(), e, u)
		if err != nil {
			t.Fatalf("RunUnit(%+v) error = %v, want nil", u, err)
		}
		out = append(out, recs...)
	}
	return out
}

// parseDocs loads documents with the loader, as the harness does.
func parseDocs(t testing.TB, docs ...[]byte) economysim.Config {
	t.Helper()
	cfg, err := config.Parse(docs...)
	if err != nil {
		t.Fatalf("config.Parse() error = %v, want nil", err)
	}
	return cfg
}

// digestOf is the digest of a world as the harness takes it: the last 32 bytes
// of its checkpoint's encoding.
func digestOf(t testing.TB, w *economysim.World) [32]byte {
	t.Helper()
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	b, err := cp.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	var d [32]byte
	copy(d[:], b[len(b)-32:])
	return d
}
