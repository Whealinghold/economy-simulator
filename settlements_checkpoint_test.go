package economysim_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// settlementsRunConfig is referenceConfig with the module on and four classes by
// Enter (hamlet 0, village 300, town 500, city 700), a dwell of 10 and no floor. The
// region north has 600 households, so it is a town at construction (class 2).
func settlementsRunConfig() economysim.Config {
	c := referenceConfig()
	m := economysim.DefaultSettlements()
	m.Dwell = 10
	m.Classes = []economysim.SettlementClass{
		{ID: "hamlet", Enter: 0},
		{ID: "village", Enter: 300, Flags: []string{"fair"}},
		{ID: "town", Enter: 500, Flags: []string{"fair"}, Limits: []economysim.ClassLimit{{Name: "workers", Value: 900}}},
		{ID: "city", Enter: 700, Flags: []string{"fair"}},
	}
	m.Regions = []economysim.RegionSettlement{economysim.DefaultRegionSettlement("north")}
	c.Modules.Settlements = &m
	return c
}

// addExternals adds n external agents of weight 1 to region 0, as a call between
// ticks does: the households of the region rise by n and its class does not move until
// the next review.
func addExternals(n int) func(*economysim.World) {
	return func(w *economysim.World) {
		for range n {
			if _, err := w.AddAgent(economysim.NewAgent{Region: 0, Role: 0, Money: 100}); err != nil {
				panic("AddAgent: " + err.Error()) // the world of the test accepts it; Before has no *testing.T
			}
		}
	}
}

// settlementsSection returns the body of tag 23 of a checkpoint and whether it has one.
func settlementsSection(t *testing.T, raw []byte) (body []byte, ok bool) {
	t.Helper()
	for _, s := range decodeFrame(t, raw).sections {
		if s.tag == 23 {
			return s.body, true
		}
	}
	return nil, false
}

// TestCheckpoint_SettlementsSectionIsDenseAndAbsentWhenOff verifies tag 23: a world with
// the module on writes the class and Since of each region as two unsigned varints, and
// a world with the module off writes no section and no byte of any other changes its
// layout.
//
// Arithmetic: north is a town, class 2, with Since 0 at tick 5, so the body is the two
// bytes 2 and 0.
//
// Requirement: FR-SET-029, FR-CHK-023
func TestCheckpoint_SettlementsSectionIsDenseAndAbsentWhenOff(t *testing.T) {
	t.Parallel()
	on := newPackWorld(t, settlementsRunConfig(), 1)
	step(t, on, 5)
	body, ok := settlementsSection(t, mustMarshal(t, mustCheckpoint(t, on)))
	if !ok || !slices.Equal(body, []byte{2, 0}) {
		t.Errorf("tag 23 of the world with the module on = %v, %v, want [2 0], true", body, ok)
	}
	off := newPackWorld(t, referenceConfig(), 1)
	step(t, off, 5)
	if body, ok := settlementsSection(t, mustMarshal(t, mustCheckpoint(t, off))); ok {
		t.Errorf("tag 23 of the world with the module off = %v, want no section", body)
	}
}

// TestCheckpoint_SettlementsRefusesCorrupt verifies that the decoder of tag 23 refuses
// each way the section can be wrong, as ErrCheckpointCorrupt naming section 23: a class
// not below the number of classes, a class below the floor of its region, a Since above
// the tick of the checkpoint less 1, a section that ends early or has bytes after its
// last row, a section missing with the module on and a section present with the module
// off; and that the good body restores.
//
// Arithmetic: at tick 5 the largest Since is 4; there are four classes, so the largest
// class is 3; with the floor village (1) a class of 0 is below it.
//
// Requirement: FR-SET-029, FR-CHK-021
func TestCheckpoint_SettlementsRefusesCorrupt(t *testing.T) {
	t.Parallel()
	cfg := settlementsRunConfig()
	cfg.Modules.Settlements.Regions[0].Floor = "village"
	w := newPackWorld(t, cfg, 1)
	step(t, w, 5)
	raw := mustMarshal(t, mustCheckpoint(t, w))
	tests := []struct {
		name string
		body []byte
	}{
		{"a class not below the number of classes", []byte{4, 0}},
		{"a class below the floor", []byte{0, 0}},
		{"a since above the tick less 1", []byte{2, 5}},
		{"a section that ends early", []byte{2}},
		{"an empty section", nil},
		{"bytes after the last row", []byte{2, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, func(s []section) []section {
				return editSection(s, 23, func([]byte) []byte { return tt.body })
			})
			_, err := economysim.RestoreWorld(cfg, c)
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) || !strings.Contains(err.Error(), "section 23") {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt naming section 23", tt.name, err)
			}
		})
	}
	t.Run("a section missing", func(t *testing.T) {
		t.Parallel()
		c := reframe(t, raw, func(s []section) []section {
			return slices.DeleteFunc(s, func(x section) bool { return x.tag == 23 })
		})
		if _, err := economysim.RestoreWorld(cfg, c); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
			t.Errorf("RestoreWorld(no section 23) error = %v, want ErrCheckpointCorrupt", err)
		}
	})
	t.Run("a section with the module off", func(t *testing.T) {
		t.Parallel()
		off := newPackWorld(t, referenceConfig(), 1)
		step(t, off, 5)
		c := reframe(t, mustMarshal(t, mustCheckpoint(t, off)), func(s []section) []section {
			i := slices.IndexFunc(s, func(x section) bool { return x.tag > 23 })
			return slices.Insert(s, i, section{tag: 23, body: []byte{2, 0}})
		})
		if _, err := economysim.RestoreWorld(referenceConfig(), c); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
			t.Errorf("RestoreWorld(a section 23 with the module off) error = %v, want ErrCheckpointCorrupt", err)
		}
	})
	good := reframe(t, raw, func(s []section) []section { return editSection(s, 23, func([]byte) []byte { return []byte{2, 4} }) })
	if _, err := economysim.RestoreWorld(cfg, good); err != nil {
		t.Errorf("RestoreWorld(the body [2 4]) error = %v, want nil", err)
	}
}

// TestFork_Settlements verifies the fork-at-N of a world with the module on and a dwell
// of 10: at each N of the cases the original and two restores give the same events,
// snapshot, audit and checkpoint bytes for 30 ticks, so the restored world reviews as the
// uninterrupted one would (spec-settlements.md 4.5.2, SET-S5). The cases are a count
// moved by a call before the first step, a hold in progress, the tick of the change and
// a boundary with no change due.
//
// Arithmetic: north has 600 households, a town (enter 500); 150 external agents make
// 750 >= 700, the enter of city. With Since 0 and Dwell 10 the change is held in ticks
// 0 to 9 and applies at tick 10, so a fork at N 5 saves the class with a hold ahead, and
// a fork at N 10 has the change in its first step.
//
// Requirement: FR-SET-029, FR-CHK-050, FR-SET-011
func TestFork_Settlements(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := settlementsRunConfig()
	has := func(want ...string) func(*testing.T, []economysim.Event, []economysim.Event) {
		return func(t *testing.T, _, since []economysim.Event) {
			t.Helper()
			var got []string
			for _, e := range since {
				switch e.(type) {
				case economysim.SettlementHeld:
					got = append(got, "held")
				case economysim.SettlementClassChanged:
					got = append(got, "changed")
				}
			}
			for _, w := range want {
				if !slices.Contains(got, w) {
					t.Errorf("events after the boundary hold no %s event, want one", w)
				}
			}
		}
	}
	for _, c := range []struct {
		name    string
		n       int
		before  func(*economysim.World)
		inspect func(*testing.T, []economysim.Event, []economysim.Event)
	}{
		{"count moved before the first step", 0, addExternals(150), has("held", "changed")},
		{"inside a hold", 5, addExternals(150), has("held", "changed")},
		{"at the tick of the change", 10, addExternals(150), has("changed")},
		{"no change due", 20, nil, nil},
	} {
		forkAtN(t, forkCase{Name: "settlements " + c.name, Cfg: cfg, Seed: 1, N: c.n, M: forkM, Logs: true, Before: c.before, Inspect: c.inspect})
	}
}
