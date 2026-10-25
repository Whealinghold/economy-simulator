package economysim

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the dynamic part of the region graph: the status of an edge, set
// between ticks (spec-regions.md 7), and what the world shows and saves of it.

// EdgeStatus is the status a shock gives an edge. The zero Closed and FactorBP
// 10 000 are the default; the zero EdgeStatus is not valid, because a factor of 0
// is below 1.
type EdgeStatus struct {
	From, To RegionID
	Closed   bool
	FactorBP int64 // 1 to MaxEdgeFactorBP
}

// statusOf returns the status of the edge at index i: the default while the
// world has no other.
func (r *regionState) statusOf(i int) edgeState {
	if r.status == nil {
		return edgeState{factorBP: defaultFactorBP}
	}
	return r.status[i]
}

// setEdgeStatuses applies the changes to the world's graph, all or none
// (spec-regions.md 7.3): an edge that is not configured is ErrUnknownEdge, a
// factor outside [1, MaxEdgeFactorBP] ErrOutOfBounds and an edge twice
// ErrDuplicateID. The tables are rebuilt once for the batch, then
// EdgeStatusChanged is emitted for each edge whose status changed, in ascending
// EdgeID; a change equal to the status in force is accepted and silent. The caller
// (INT's call between ticks, or a policy actor at P0.3a) owns the guards of the
// world and the cause.
func (w *World) setEdgeStatuses(changes []EdgeStatus, cause Cause) error {
	next, err := w.nextEdgeStatuses(changes)
	if err != nil {
		return err
	}
	var changed []int
	for i, s := range next {
		if s != w.geo.statusOf(i) {
			changed = append(changed, i)
		}
	}
	if len(changed) == 0 {
		return nil
	}
	g, err := newGraph(&w.cfg, next)
	if err != nil {
		return fmt.Errorf("edge statuses: %w", err)
	}
	w.geo.g = g
	w.geo.status = next
	if !slices.ContainsFunc(next, func(s edgeState) bool { return !s.isDefault() }) {
		w.geo.status = nil
	}
	w.geo.rebuilds++
	for _, i := range changed {
		w.edgeStatusChanged(g.view(i), next[i], cause)
	}
	return nil
}

// nextEdgeStatuses returns the statuses by EdgeID that the changes would give,
// without changing the world, or the first change that is refused.
func (w *World) nextEdgeStatuses(changes []EdgeStatus) ([]edgeState, error) {
	m := len(w.cfg.Edges)
	next := make([]edgeState, m)
	for i := range next {
		next[i] = w.geo.statusOf(i)
	}
	touched := make([]bool, m)
	for i, c := range changes {
		e, ok := w.geo.g.Edge(c.From, c.To)
		switch {
		case !ok:
			return nil, fmt.Errorf("edge status %d, region %d to %d: %w", i, c.From, c.To, ErrUnknownEdge)
		case c.FactorBP < 1 || c.FactorBP > MaxEdgeFactorBP:
			return nil, fmt.Errorf("edge status %d, factor %d: %w", i, c.FactorBP, ErrOutOfBounds)
		case touched[e.ID]:
			return nil, fmt.Errorf("edge status %d, region %d to %d: %w", i, c.From, c.To, ErrDuplicateID)
		}
		touched[e.ID] = true
		next[e.ID] = edgeState{closed: c.Closed, factorBP: c.FactorBP}
	}
	return next, nil
}

// edgeStatusChanged emits the event and the Debug line of one changed edge. The
// event is built only when an observer was supplied.
func (w *World) edgeStatusChanged(e EdgeView, s edgeState, cause Cause) {
	if w.obs.supplied {
		w.obs.deliver(EdgeStatusChanged{Tick: w.Tick(), From: e.From, To: e.To, Closed: s.closed, FactorBP: s.factorBP, Cause: cause})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "edge status changed",
			slog.Int64(logkeys.FromRegion, int64(e.From)), slog.Int64(logkeys.ToRegion, int64(e.To)),
			slog.Bool(logkeys.Closed, s.closed), slog.Int64(logkeys.FactorBP, s.factorBP), slog.String(logkeys.Cause, string(cause)))
	}
}

// isDefault reports whether the status is the default: open at 10 000.
func (s edgeState) isDefault() bool { return s == edgeState{factorBP: defaultFactorBP} }

// edgeRows returns the edges whose status is not the default, by EdgeID: the rows
// of the snapshot and of tag 11. It is nil while every status is the default.
func (w *World) edgeRows() []EdgeRow {
	var rows []EdgeRow
	for i, s := range w.geo.status {
		if !s.isDefault() {
			e := w.geo.g.edges[i]
			rows = append(rows, EdgeRow{From: e.from, To: e.to, Closed: s.closed, FactorBP: s.factorBP})
		}
	}
	return rows
}

// componentCount is Snapshot.Components: the components of the graph as it is now,
// and 0 for a world of one region (spec-regions.md 10.5).
func (w *World) componentCount() int64 {
	if len(w.cfg.Regions) < 2 {
		return 0
	}
	return int64(w.geo.components())
}

// saveRegions is tag 11 (spec-checkpoint.md 4.2): a count, then the edges whose
// status is not the default in ascending EdgeID, each as From, To, Closed and
// FactorBP. A world with none writes the count 0, the layout's prefix, so the
// bytes of a world with no status did not change when the rows were added (plan
// 2-2 D244).
func (w *World) saveRegions(cw *codec.Writer) error {
	rows := w.edgeRows()
	cw.Count(len(rows))
	for _, e := range rows {
		cw.U(uint64(e.From))
		cw.U(uint64(e.To))
		cw.B(e.Closed)
		cw.U(uint64(e.FactorBP))
	}
	return nil
}

// loadRegions is the reader of tag 11. It runs only on a world just built from
// the configuration, whose statuses are the defaults, so a count of 0 changes
// nothing. It refuses a row that names no configured
// edge, one not above the one before it in EdgeID order, one at the default
// status and a factor outside [1, MaxEdgeFactorBP]; then it builds the world's
// tables from the configuration and the statuses, as a world that had them set
// would hold them (spec-regions.md 11).
func (w *World) loadRegions(_ *restoreCtx, r *codec.Reader) error {
	count := r.Count(4)
	status := make([]edgeState, len(w.cfg.Edges))
	for i := range status {
		status[i] = edgeState{factorBP: defaultFactorBP}
	}
	prev := EdgeID(-1)
	for range count {
		id, s := w.readEdgeStatus(r, prev)
		if r.Err() != nil {
			return nil
		}
		prev, status[id] = id, s
	}
	if count == 0 {
		return nil
	}
	g, err := newGraph(&w.cfg, status)
	if err != nil {
		return err
	}
	w.geo.g, w.geo.status = g, status
	return nil
}

// readEdgeStatus reads one row of tag 11 and checks it: it must name a configured
// edge (a check that needs the configuration only, so it is made here and not in
// checkpointCross), come after the edge before it, and not be the default.
func (w *World) readEdgeStatus(r *codec.Reader, prev EdgeID) (EdgeID, edgeState) {
	n := uint64(len(w.cfg.Regions))
	from, to := RegionID(r.Index(n)), RegionID(r.Index(n))
	s := edgeState{closed: r.B(), factorBP: r.Bounded("edge factor", MaxEdgeFactorBP)}
	if r.Err() != nil {
		return 0, s
	}
	e, ok := w.geo.g.Edge(from, to)
	switch {
	case !ok:
		r.Failf("edge status row for regions %d to %d, which is not a configured edge", from, to)
	case e.ID <= prev:
		r.Failf("edge status row for edge %d is not above the one before it", e.ID)
	case s.factorBP < 1:
		r.Failf("edge status row for edge %d has the factor %d, below 1", e.ID, s.factorBP)
	case s.isDefault():
		r.Failf("edge status row for edge %d holds the default status", e.ID)
	}
	return e.ID, s
}
