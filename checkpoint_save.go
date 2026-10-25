package economysim

import (
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
)

// This file is the part of the write path that is not a concept's own: the
// sections whose owner has no code yet (tags 10 and 11), the ledger's, the
// extensions' (tag 34) and the helpers the concept files' saveX share
// (spec-checkpoint.md 4.2, 4.1.4; plan 2-2 D244, D247, D255).

// saveLedger is tag 2 (spec-checkpoint.md 4.2): the ledger writes its own
// section, over the same writer.
func (w *World) saveLedger(cw *codec.Writer) error {
	w.ledger.WriteState(cw)
	return nil
}

// saveSkills is tag 10 as a zero-row skeleton: four counts of 0, for the trainees,
// the trainers, the rows of the queue and the rows of the Warn run, in the order
// of spec-checkpoint.md 4.2.8. The skills module keeps this as the prefix of its
// layout, so a world with no row keeps its bytes (plan 2-2 D244).
func (w *World) saveSkills(cw *codec.Writer) error {
	for range 4 {
		cw.Count(0)
	}
	return nil
}

// eachLive calls f for every live agent in ascending ID: the household agents
// whose slot is not the zero agent of an estate or of a removed agent.
func (w *World) eachLive(f func(a *agent)) {
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 {
			f(a)
		}
	}
}

// strategyName is the name an agent's strategy is saved under (spec-checkpoint.md
// 4.2, tag 3). In the slice an agent's strategy is its role's, so the name is the
// role's configured one; an external agent's own name replaces this one place
// with the integration module.
func (w *World) strategyName(a *agent) string { return w.cfg.Roles[a.role].Strategy }

// deathCarry is the start of the death carry of an agent, offset(id) of
// spec-population.md 3.8.3: ((id * 2 654 435 769) mod 2^32) * D / 2^32, with D the
// ticks of ten thousand years. The population module owns the field and is not
// built; the agent record always carries its value, so that the bytes of tag 3 do
// not change when the module lands (spec-checkpoint.md R10, N-217). With the
// module off the carry never moves from it.
func (w *World) deathCarry(id AgentID) int64 {
	// Excluded from coverage (testing.md 9.5): validation bounds the calendar, so
	// neither product overflows and the two checks only name a violated invariant.
	d, ok := checked.Mul(wholeBP, w.cfg.Calendar.TicksPerYear())
	if !ok {
		invariant("the death carry bound overflows after validation")
	}
	v, ok := checked.MulDiv(int64(uint64(id)*2_654_435_769%(1<<32)), d, 1<<32)
	if !ok {
		invariant("the death carry offset overflows after validation")
	}
	return v
}

// writeSparse writes the rows of vals, indexed by region * roles + role, that
// differ from 0, as a count and then (region, role, value) in ascending key
// order (spec-checkpoint.md 4.1.4): a row at its default is not written.
func writeSparse(cw *codec.Writer, vals []int64, roles int) {
	writeSparseRows(cw, vals, func(i int) {
		cw.U(uint64(i / roles))
		cw.U(uint64(i % roles))
	})
}

// writeSparseByRegion is writeSparse for a run kept per region: a count and then
// (region, value).
func writeSparseByRegion(cw *codec.Writer, vals []int64) {
	writeSparseRows(cw, vals, func(i int) { cw.U(uint64(i)) })
}

// writeSparseRows is the loop of both: a count of the non-zero rows, then for each
// the key written by writeKey and the value.
func writeSparseRows(cw *codec.Writer, vals []int64, writeKey func(i int)) {
	n := 0
	for _, v := range vals {
		if v != 0 {
			n++
		}
	}
	cw.Count(n)
	for i, v := range vals {
		if v != 0 {
			writeKey(i)
			cw.U(unsigned(v))
		}
	}
}

// The kinds of an extension entry, the enumeration of spec-checkpoint.md 4.2.4. A
// policy is 3 and has no registry yet, so it has no entry (plan 2-2 D255).
const (
	extStrategy uint64 = iota
	extSpawner
	extClearing
)

// extEntry is one entry of tag 34: a registered extension that implements
// Stateful.
type extEntry struct {
	kind uint64
	name string
	s    Stateful
}

// statefulEntries returns the registered extensions that implement Stateful, used
// or not, plus the built-in strategy "directed", in the order of (kind, name)
// with the names as byte strings (spec-checkpoint.md 7.3.1, 7.5).
func (w *World) statefulEntries() []extEntry {
	// This must use the world's resolved strategy once DefaultStrategy gains state.
	out := []extEntry{{extStrategy, directedStrategyName, Directed{Fallback: DefaultStrategy{}}}}
	for i, s := range w.reg.strategies.vals {
		if st, ok := s.(Stateful); ok {
			out = append(out, extEntry{extStrategy, w.reg.strategies.names[i], st})
		}
	}
	for i, s := range w.reg.spawners.vals {
		if st, ok := s.(Stateful); ok {
			out = append(out, extEntry{extSpawner, w.reg.spawners.names[i], st})
		}
	}
	for i, c := range w.reg.clearings.vals {
		if st, ok := c.(Stateful); ok {
			out = append(out, extEntry{extClearing, w.reg.clearings.names[i], st})
		}
	}
	slices.SortFunc(out, func(a, b extEntry) int {
		return compareExt(a.kind, a.name, b.kind, b.name)
	})
	return out
}

// saveExtensions is tag 34 (spec-checkpoint.md 4.2, 7): a count and, for each
// entry, its kind, its name, the version and the state bytes SaveState returned,
// called once per entry. A SaveState error is ErrCheckpointExtension wrapping it.
func (w *World) saveExtensions(cw *codec.Writer) error {
	entries := w.statefulEntries()
	cw.Count(len(entries))
	for _, e := range entries {
		version, state, err := e.s.SaveState()
		if err != nil {
			return fmt.Errorf("%w: SaveState of %q (kind %d): %w", ErrCheckpointExtension, e.name, e.kind, err)
		}
		cw.U(e.kind)
		cw.Bytes([]byte(e.name))
		cw.U(uint64(version))
		cw.Bytes(state)
	}
	return nil
}

// unsigned returns v for a field the checkpoint writes unsigned. A negative v
// would wrap to a number the restore refuses, so it is a violated invariant, found
// when the checkpoint is made and not when it is restored.
func unsigned[T ~int | ~int32 | ~int64](v T) uint64 {
	if v < 0 {
		invariant("a negative value in a field a checkpoint writes unsigned")
	}
	return uint64(v)
}
