package economysim

import (
	"context"
	"fmt"
	"log/slog"

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
	if !anyNonDefaultStatus(next) {
		w.geo.status = nil
	}
	w.geo.rebuilds++
	for _, i := range changed {
		w.edgeStatusChanged(g.view(i), next[i], cause)
	}
	return nil
}

// anyNonDefaultStatus reports whether some status is not the default.
func anyNonDefaultStatus(s []edgeState) bool {
	for _, e := range s {
		if e != (edgeState{factorBP: defaultFactorBP}) {
			return true
		}
	}
	return false
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
			slog.Bool(logkeys.Closed, s.closed), slog.String(logkeys.Cause, string(cause)))
	}
}
