package economysim_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// roleIndex returns the RoleID of a role name of a snapshot's names table.
func roleIndex(t *testing.T, s economysim.Snapshot, name string) int {
	t.Helper()
	for i, n := range s.Names.Roles {
		if n == name {
			return i
		}
	}
	t.Fatalf("role %q not in the names table %q", name, s.Names.Roles)
	return -1
}

// snapshotOf takes a snapshot and fails the test on an error.
func snapshotOf(t *testing.T, w *economysim.World) economysim.Snapshot {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	return s
}

// TestSnapshot_DeepCopyAndNoRace verifies that a snapshot owns what it carries:
// mutating every slice of it changes nothing in the world or in the next
// snapshot, a snapshot taken and then a world stepped is unchanged, and
// marshalling a snapshot on another goroutine while the world steps has no race
// under -race.
//
// Arithmetic: two snapshots taken at one tick are equal (the same state); after
// 3 more steps the tick is 3 higher and the first one still reads its own tick.
//
// Requirement: FR-OBS-030, FR-WLD-010
func TestSnapshot_DeepCopyAndNoRace(t *testing.T) {
	t.Parallel()
	w, err := economysim.NewWorld(loadPack(t, "base.toml", "one.toml"), economysim.WithSeed(7))
	if err != nil {
		t.Fatalf("NewWorld(one) error = %v, want nil", err)
	}
	if _, err := w.Run(context.Background(), 2); err != nil {
		t.Fatalf("Run(2) error = %v, want nil", err)
	}
	first, ref := snapshotOf(t, w), snapshotOf(t, w)
	if !reflect.DeepEqual(first, ref) {
		t.Fatalf("two snapshots of tick %d differ", first.Tick)
	}
	scribble(&first)
	if again := snapshotOf(t, w); !reflect.DeepEqual(again, ref) {
		t.Errorf("mutating a snapshot changed the next snapshot of the world")
	}
	kept := snapshotOf(t, w)
	done := make(chan error, 1)
	go func() { _, err := json.Marshal(kept); done <- err }()
	if _, err := w.Run(context.Background(), 3); err != nil {
		t.Fatalf("Run(3) error = %v, want nil", err)
	}
	if err := <-done; err != nil {
		t.Errorf("json.Marshal(snapshot) error = %v, want nil", err)
	}
	if !reflect.DeepEqual(kept, ref) {
		t.Errorf("a snapshot changed after the world was stepped: tick %d, want %d", kept.Tick, ref.Tick)
	}
}

// scribble overwrites every slice of s, so a shared backing array would show.
func scribble(s *economysim.Snapshot) {
	s.Names.Roles[0], s.Names.Regions[0], s.Names.Commodities[0] = "x", "x", "x"
	s.Regions[0].Agents, s.Regions[0].MoneyPool = -1, -1
	s.Roles[0].Agents, s.Markets[0].Lots, s.Markets[0].State.Price = -1, -1, -1
	for i := range s.Boundary {
		s.Boundary[i].Balance = -1
	}
	s.Modules = append(s.Modules[:0], "x")
}

// TestSnapshot_Guards verifies that Snapshot returns ErrInStep when called from
// an observer inside a step, and ErrWorldFailed (wrapping the cause) on a failed
// world.
//
// Requirement: FR-OBS-031
func TestSnapshot_Guards(t *testing.T) {
	t.Parallel()
	t.Run("in step", func(t *testing.T) {
		t.Parallel()
		var w *economysim.World
		var inner error
		probe := economysim.ObserverFunc(func(e economysim.Event) {
			if _, ok := e.(economysim.TickStarted); ok {
				_, inner = w.Snapshot()
			}
		})
		w = shellWorld(t, economysim.WithObserver(probe))
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
		if !errors.Is(inner, economysim.ErrInStep) {
			t.Errorf("Snapshot() from an observer = %v, want ErrInStep", inner)
		}
	})
	t.Run("failed", func(t *testing.T) {
		t.Parallel()
		w := shellWorld(t, economysim.WithObserver(panicOnTickStarted()))
		_ = recoverPanic(func() { _ = w.Step() })
		s, err := w.Snapshot()
		if !errors.Is(err, economysim.ErrWorldFailed) || !errors.Is(err, w.Err()) || !reflect.DeepEqual(s, economysim.Snapshot{}) {
			t.Errorf("Snapshot() on a failed world = (%+v, %v), want the zero Snapshot and ErrWorldFailed wrapping %v", s, err, w.Err())
		}
	})
}

// TestSnapshot_CountsAndScale verifies, on the pack `one` at scale 1 and at
// scale 40, that Agents and Households are different fields (the households are
// the sum of the weights), that the sum over the role rows and over the region
// rows equals the header, that the estates are in neither count, that a group
// of 40 households at scale 40 is one agent of weight 40 (1 agent, 40
// households), and that the number of rows is the same at both scales.
//
// Arithmetic: at scale 40 with MinAgents 1 the group `cook` of 40 households is
// ceil(40 / 40) = 1 agent; the other groups of the pack keep their households,
// so the header households are 600 - 136 + 40 = 504 at both scales.
//
// Requirement: FR-OBS-032, FR-OBS-033, FR-SCL-015
func TestSnapshot_CountsAndScale(t *testing.T) {
	t.Parallel()
	var rows [2][4]int
	for k, scale := range []int64{1, 40} {
		cfg := loadPack(t, "base.toml", "one.toml")
		cfg.World.Scale, cfg.World.MinAgents = scale, 1
		for i := range cfg.Groups {
			if cfg.Groups[i].Role == "cook" {
				cfg.Groups[i].Households, cfg.Groups[i].Scale = 40, 0
			}
		}
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(3))
		if err != nil {
			t.Fatalf("scale %d: NewWorld error = %v, want nil", scale, err)
		}
		s := snapshotOf(t, w)
		var agents, hh, ragents, rhh int64
		for _, r := range s.Regions {
			ragents, rhh = ragents+r.Agents, rhh+r.Households
		}
		for _, r := range s.Roles {
			agents, hh = agents+r.Agents, hh+r.Households
		}
		if agents != s.Agents || ragents != s.Agents || hh != s.Households || rhh != s.Households {
			t.Errorf("scale %d: roles %d/%d, regions %d/%d agents/households, header %d/%d, want all equal", scale, agents, hh, ragents, rhh, s.Agents, s.Households)
		}
		if s.Households != 504 {
			t.Errorf("scale %d: header households = %d, want 504", scale, s.Households)
		}
		if scale == 40 {
			if s.Agents == s.Households {
				t.Errorf("scale 40: agents = households = %d, want them different fields", s.Agents)
			}
			cook := s.Roles[roleIndex(t, s, "cook")]
			if cook.Agents != 1 || cook.Households != 40 {
				t.Errorf("scale 40: cook row = %d agents, %d households, want 1 and 40", cook.Agents, cook.Households)
			}
		}
		rows[k] = [4]int{len(s.Regions), len(s.Roles), len(s.Markets), len(s.Boundary)}
	}
	if rows[0] != rows[1] {
		t.Errorf("row counts at scale 1 = %v, at scale 40 = %v, want equal", rows[0], rows[1])
	}
}

// TestSnapshot_AllocationsDoNotGrowWithScale verifies that the allocation count
// of Snapshot at scale 1 is the one at scale 40 for one configuration: the scans
// reuse the world's scratch and the rows are sized by the configuration.
//
// Requirement: NFR-OBS-004
func TestSnapshot_AllocationsDoNotGrowWithScale(t *testing.T) {
	var got [2]float64
	for k, scale := range []int64{1, 40} {
		cfg := loadPack(t, "base.toml", "one.toml")
		cfg.World.Scale = scale
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(3))
		if err != nil {
			t.Fatalf("scale %d: NewWorld error = %v, want nil", scale, err)
		}
		if err := w.Step(); err != nil {
			t.Fatalf("scale %d: Step error = %v, want nil", scale, err)
		}
		got[k] = testing.AllocsPerRun(5, func() {
			if _, err := w.Snapshot(); err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
		})
	}
	if got[0] != got[1] {
		t.Errorf("allocations per Snapshot = %v at scale 1 and %v at scale 40, want equal", got[0], got[1])
	}
}

// TestSnapshot_ChangesNothing verifies that taking snapshots draws no random
// number and changes no state: two worlds of one seed, one snapshotted at every
// boundary and one not, have equal snapshots after 6 ticks.
//
// Requirement: FR-OBS-030, FR-WLD-010
func TestSnapshot_ChangesNothing(t *testing.T) {
	t.Parallel()
	var end [2]economysim.Snapshot
	for k, look := range []bool{true, false} {
		w, err := economysim.NewWorld(loadPack(t, "base.toml", "one.toml"), economysim.WithSeed(11))
		if err != nil {
			t.Fatalf("NewWorld(one) error = %v, want nil", err)
		}
		for range 6 {
			if look {
				snapshotOf(t, w)
			}
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
		}
		end[k] = snapshotOf(t, w)
	}
	if !reflect.DeepEqual(end[0], end[1]) {
		t.Errorf("snapshots at tick %d differ between a world that was looked at and one that was not (seed 11)", end[0].Tick)
	}
}
