package economysim

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// churnWorld builds the world of churnConfig with the pool seeded, observed by a
// recorder only when asked, and steps it n ticks.
func churnWorld(t *testing.T, n int) (*World, Config) {
	t.Helper()
	c := churnConfig()
	w := newMill(t, c, WithSeed(popSeed), WithAudit())
	if err := w.estates.seedPool(w.ledger, 0, 500_000); err != nil {
		t.Fatalf("seed %d: seedPool() error = %v, want nil", popSeed, err)
	}
	step(t, w, n)
	return w, c
}

// TestCheckpoint_PopulationRoundTrip verifies that the death carry of every agent
// (in the agent record), the birth carries, a scale a shock set and the three runs
// are saved and restored exactly, and that the restored world makes the same births
// and deaths as the original afterwards.
//
// Requirement: FR-POP-035, FR-POP-024
func TestCheckpoint_PopulationRoundTrip(t *testing.T) {
	t.Parallel()
	w, c := churnWorld(t, 60)
	if _, ok := w.setPopulationScale(0, ScaleDeath, 20_000); !ok {
		t.Fatalf("seed %d: setPopulationScale() = false, want true", popSeed)
	}
	w.pop.unfundedRun[0], w.pop.heldRun[1], w.pop.extinctRun[0] = 4, 3, 2
	if !slices.ContainsFunc(w.pop.birthCarry, func(c int64) bool { return c > 0 }) {
		t.Fatalf("seed %d: no birth carry is non-zero after 60 ticks, want the round trip to carry one", popSeed)
	}
	got := restoreWith(t, w, c)
	same := func(when string) {
		t.Helper()
		for id := len(w.cfg.Regions); id < len(w.agents); id++ {
			if w.agents[id].weight >= 1 && got.pop.carryOf(AgentID(id)) != w.pop.carryOf(AgentID(id)) {
				t.Errorf("seed %d, %s: death carry of agent %d = %d, want %d", popSeed, when, id, got.pop.carryOf(AgentID(id)), w.pop.carryOf(AgentID(id)))
				return
			}
		}
		if !slices.Equal(got.pop.birthCarry, w.pop.birthCarry) || !slices.Equal(got.pop.effDeath, w.pop.effDeath) || !slices.Equal(got.pop.effBirth, w.pop.effBirth) ||
			!slices.Equal(got.pop.unfundedRun, w.pop.unfundedRun) || !slices.Equal(got.pop.heldRun, w.pop.heldRun) || !slices.Equal(got.pop.extinctRun, w.pop.extinctRun) {
			t.Errorf("seed %d, %s: restored state differs: carries %v vs %v, scales %v vs %v", popSeed, when, got.pop.birthCarry, w.pop.birthCarry, got.pop.effDeath, w.pop.effDeath)
		}
	}
	same("at the boundary")
	step(t, got, 20)
	step(t, w, 20)
	same("20 ticks after")
	if got.live.households != w.live.households {
		t.Errorf("seed %d: households %d after the restore, want the original's %d", popSeed, got.live.households, w.live.households)
	}
}

// TestCheckpoint_PopulationRejectsCorrupt verifies that tag 14 refuses a birth
// carry at D, a scale equal to the configured one, rows out of order, a run at 0 or
// above the ticks, a trailing byte, and (cross-check) a birth carry of a closed role;
// and that tag 3 refuses a death carry outside [0, D).
//
// Requirement: FR-POP-035
func TestCheckpoint_PopulationRejectsCorrupt(t *testing.T) {
	t.Parallel()
	w := popWorld(t, DefaultPopulation())
	d := uint64(w.pop.denom)
	body := func(birth [][3]uint64, scales [][3]uint64, tail ...int) []byte {
		cw := codec.NewWriter(nil)
		cw.Count(len(birth))
		for _, r := range birth {
			cw.U(r[0])
			cw.U(r[1])
			cw.U(r[2])
		}
		cw.Count(len(scales))
		for _, r := range scales {
			cw.U(r[0])
			cw.U(r[1])
			cw.U(r[2])
		}
		for range 3 {
			cw.Count(0)
		}
		out := cw.Out()
		for _, b := range tail {
			out = append(out, byte(b))
		}
		return out
	}
	for _, tt := range []struct {
		name string
		body []byte
	}{
		{"birth carry at D", body([][3]uint64{{0, 0, d}}, nil)},
		{"birth carry at 0", body([][3]uint64{{0, 0, 0}}, nil)},
		{"rows out of order", body([][3]uint64{{0, 1, 5}, {0, 0, 5}}, nil)},
		{"unknown role", body([][3]uint64{{0, 9, 5}}, nil)},
		{"scale equal to the row", body(nil, [][3]uint64{{0, 0, 10_000}})},
		{"scales out of order", body(nil, [][3]uint64{{0, 1, 1}, {0, 0, 1}})},
		{"unknown scale kind", body(nil, [][3]uint64{{0, 2, 1}})},
		{"trailing byte", body(nil, nil, 0)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := popWorld(t, DefaultPopulation())
			r := codec.NewReader(tt.body)
			err := w.loadPopulation(&restoreCtx{tick: 9}, r)
			if err == nil {
				err = r.Done()
			}
			if err == nil && r.Err() == nil {
				t.Errorf("loadPopulation(%v) = no error, want one", tt.body)
			}
		})
	}
	if ok := body(nil, [][3]uint64{{0, 0, 1}}); func() error {
		r := codec.NewReader(ok)
		if err := w.loadPopulation(&restoreCtx{tick: 9}, r); err != nil {
			return err
		}
		return r.Done()
	}() != nil || w.pop.effBirth[0] != 1 {
		t.Errorf("loadPopulation(a scale of 1) refused or not applied, want it accepted")
	}
	w = popWorld(t, DefaultPopulation())
	w.pop.birthCarry[1] = 5 // role 1, the weaver, is closed
	if err := w.populationCross(); !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("populationCross() with a carry in a closed role error = %v, want ErrCheckpointCorrupt", err)
	}
	rec := agentRec{id: 1, weight: 1}
	for carry, want := range map[int64]bool{-1: true, 0: false, w.pop.denom - 1: false, w.pop.denom: true} {
		if got := w.badDeathCarry(AgentID(rec.id), carry); got != want {
			t.Errorf("badDeathCarry(1, %d) = %v, want %v", carry, got, want)
		}
	}
}
