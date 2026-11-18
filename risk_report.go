package economysim

import (
	"cmp"
	"context"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the reporting of the risk module (spec-risk.md 14, 15): the Debug
// lines and the checkpoint section, tag 28, with the loss share the shipment record
// of tag 17 carries when the module is on. The logs go through the injected logger
// only and never influence a result.

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

// saveRisk is tag 28 (spec-checkpoint.md 4.2, spec-risk.md 15.1): the route scales
// that differ from 10 000, ascending by key, each as (From, To, Mode, BP), then the
// store scales, ascending by key, each as (Region, Class, BP). The rows, the
// probabilities, the outcomes of the common rows and the table of expected losses are
// derived and not saved.
func (w *World) saveRisk(cw *codec.Writer) error {
	r := w.risk
	cw.Count(len(r.scales))
	for _, s := range r.scales {
		cw.U(unsigned(s.from))
		cw.U(unsigned(s.to))
		cw.U(unsigned(s.mode))
		cw.U(unsigned(s.bp))
	}
	cw.Count(len(r.storeScales))
	for _, s := range r.storeScales {
		cw.U(unsigned(s.region))
		cw.U(unsigned(s.class))
		cw.U(unsigned(s.bp))
	}
	return nil
}

// loadRisk reads tag 28: the route scales and the store scales, each list strictly
// ascending by key, each key a configured row and each scale within 0 to MaxShockBP
// and not 10 000 (spec-risk.md 15.3). Any other record is ErrCheckpointCorrupt. The
// table of expected losses is rebuilt from the rows, the scales and the graph.
func (w *World) loadRisk(_ *restoreCtx, r *codec.Reader) error {
	rk := w.risk
	regions, modes := uint64(len(w.cfg.Regions)), uint64(0)
	if w.transport != nil {
		modes = uint64(len(w.transport.modes))
	}
	var prev routeKey
	for i := range r.Count(4) {
		k := routeKey{RegionID(r.Index(regions)), RegionID(r.Index(regions)), ModeID(r.Index(modes))}
		bp := r.Bounded("route risk scale", MaxShockBP)
		switch {
		case r.Err() != nil:
			return nil
		case bp == wholeBP:
			r.Failf("route risk scale %d of (%d, %d, %d) is the default, which is not saved", i, k.from, k.to, k.mode)
		case i > 0 && compareRoute(prev, k) >= 0:
			r.Failf("route risk scale %d is not above the one before it", i)
		case rk.find(k.from, k.to, k.mode) == nil:
			r.Failf("route risk scale %d names (%d, %d, %d), which is no route row", i, k.from, k.to, k.mode)
		}
		if r.Err() != nil {
			return nil
		}
		rk.setRouteScale(k, bp)
		prev = k
	}
	rk.loadStoreScales(w, r)
	rk.rebuild(w.geo.g, len(w.cfg.Regions))
	return nil
}

// loadStoreScales reads the store scales of tag 28 into the state, with the checks of
// the route scales over the store rows.
func (r *risk) loadStoreScales(w *World, cr *codec.Reader) {
	classes := uint64(0)
	if m := w.cfg.Modules.Storage; m != nil {
		classes = uint64(len(m.Classes))
	}
	var prev storeScaleRec
	for i := range cr.Count(3) {
		s := storeScaleRec{region: RegionID(cr.Index(uint64(len(w.cfg.Regions)))), class: StoreClassID(cr.Index(classes))}
		s.bp = cr.Bounded("store risk scale", MaxShockBP)
		switch {
		case cr.Err() != nil:
			return
		case s.bp == wholeBP:
			cr.Failf("store risk scale %d of (%d, %d) is the default, which is not saved", i, s.region, s.class)
		case i > 0 && compareStoreKey(prev, s) >= 0:
			cr.Failf("store risk scale %d is not above the one before it", i)
		case !slices.ContainsFunc(r.stores, func(k storeScaleRec) bool { return compareStoreKey(k, s) == 0 }):
			cr.Failf("store risk scale %d names (%d, %d), which is no store row", i, s.region, s.class)
		}
		if cr.Err() != nil {
			return
		}
		r.storeScales = append(r.storeScales, s)
		prev = s
	}
}

// compareStoreKey orders two store keys by (Region, Class).
func compareStoreKey(a, b storeScaleRec) int {
	return cmp.Or(cmp.Compare(a.region, b.region), cmp.Compare(a.class, b.class))
}

// saveLoss writes the loss share of a shipment record, after its lines, when the
// module is on (spec-risk.md 15.1): a nil receiver, the module off, writes nothing, so
// the record of a world with the module off is the record without it.
func (r *risk) saveLoss(cw *codec.Writer, s *shipment) {
	if r != nil {
		cw.U(unsigned(s.loss))
	}
}

// readLoss reads the loss share of a shipment record, 0 to 10 000, when the module is
// on, and reads nothing with it off (spec-risk.md 15.3).
func (r *risk) readLoss(cr *codec.Reader, s *shipment) {
	if r != nil {
		s.loss = cr.Bounded("shipment loss share", wholeBP)
	}
}
