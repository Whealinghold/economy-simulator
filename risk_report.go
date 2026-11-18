package economysim

import (
	"context"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the reporting of the risk module (spec-risk.md 14): the Debug lines
// and, with the module's later parts, the snapshot rows and the checkpoint section.
// The logs go through the injected logger only and never influence a result.

// logRouteStruck writes the Debug line of a shipment dispatched with a loss share
// above 0 (spec-risk.md 14.3). No Info and no Warn is written in a tick: a strike is
// a fact of an entity and is an event, not an anomaly.
func (w *World) logRouteStruck(s *shipment, shareBP int64) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "route struck",
			slog.Int64(logkeys.Tick, w.Tick()), slog.Int64(logkeys.Shipment, int64(s.id)), slog.Int64(logkeys.Mode, int64(s.mode)),
			slog.Int64(logkeys.Region, int64(s.from)), slog.Int64(logkeys.Dest, int64(s.to)), slog.Int64(logkeys.Share, shareBP))
	}
}
