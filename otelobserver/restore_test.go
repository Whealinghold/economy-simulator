package otelobserver_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// levelMetrics are the instruments that two adapters must agree on from the
// first complete step after a restore: the up-down counters and the gauges fed
// every tick (spec-observability.md 6.4.2).
var levelMetrics = []string{"agent.count", "household.count", "household.unfed", "market.clearing_price"}

// TestRestore_AdapterAttachedAtRestoreAgrees verifies that an adapter attached
// at a restore, seeded from WorldRestored, shows the same levels as the one
// attached at NewWorld after the first complete step: the world is run on both
// for the same ticks, restoring at tick 0 and mid-run.
//
// The checkpoint is taken at tick n; both worlds then run 5 more ticks (the
// second complete step is the first with no seeded state left to correct).
//
// Requirement: FR-OBS-054
func TestRestore_AdapterAttachedAtRestoreAgrees(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, n := range []int{0, 12, 25} {
		a, b, wa, wb := restorePair(t, n)
		for i := 0; i < 5; i++ {
			for _, w := range []*economysim.World{wa, wb} {
				if err := w.Step(); err != nil {
					t.Fatalf("Step() n=%d error = %v", n, err)
				}
			}
		}
		pa, pb := a.collect(), b.collect()
		for _, name := range levelMetrics {
			sameLevels(t, n, name, pa["economysim."+name], pb["economysim."+name])
		}
	}
}

// restorePair runs a world with an adapter from NewWorld for n ticks, restores
// its checkpoint with a second adapter, and returns both with their worlds.
func restorePair(t *testing.T, n int) (a, b *harness, wa, wb *economysim.World) {
	t.Helper()
	a, b = newHarness(t), newHarness(t)
	cfg := miniConfig()
	wa, err := economysim.NewWorld(cfg, economysim.WithSeed(7), economysim.WithObserver(a.obs))
	if err != nil {
		t.Fatalf("NewWorld() error = %v", err)
	}
	for i := 0; i < n; i++ {
		if err := wa.Step(); err != nil {
			t.Fatalf("Step() n=%d error = %v", n, err)
		}
	}
	ckpt, err := wa.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() n=%d error = %v", n, err)
	}
	wb, err = economysim.RestoreWorld(cfg, ckpt, economysim.WithObserver(b.obs))
	if err != nil {
		t.Fatalf("RestoreWorld() n=%d error = %v", n, err)
	}
	return a, b, wa, wb
}

// sameLevels reports a series the restored adapter has and the original lacks,
// and a series whose value differs. A series the original has at 0 may be
// absent on the restored one: it was never moved there (spec-observability.md
// 6.4.2).
func sameLevels(t *testing.T, n int, name string, orig, restored map[string]int64) {
	t.Helper()
	if len(orig) == 0 {
		t.Errorf("n=%d: %s has no series on the original adapter", n, name)
	}
	for k := range restored {
		if _, ok := orig[k]; !ok {
			t.Errorf("n=%d: %s: restored has series %q the original lacks", n, name, k)
		}
	}
	for k, v := range orig {
		if restored[k] != v {
			t.Errorf("n=%d: %s{%s}: restored %d, want original %d", n, name, k, restored[k], v)
		}
	}
}

// TestRestore_FailedRestoreSeedsNothing verifies that a restore that fails
// emits no WorldRestored, so the adapter given to it holds no series.
//
// Requirement: FR-OBS-054
func TestRestore_FailedRestoreSeedsNothing(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	a := newHarness(t)
	cfg := miniConfig()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(7))
	if err != nil {
		t.Fatalf("NewWorld() error = %v", err)
	}
	ckpt, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v", err)
	}
	cfg.Groups[0].Households++ // another configuration hash
	if _, err := economysim.RestoreWorld(cfg, ckpt, economysim.WithObserver(a.obs)); err == nil {
		t.Fatal("RestoreWorld() with another configuration error = nil, want an error")
	}
	if got := a.collect(); len(got) != 0 {
		t.Errorf("points after a failed restore = %v, want none", got)
	}
}
