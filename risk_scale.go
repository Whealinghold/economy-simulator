package economysim

import (
	"cmp"
	"fmt"
	"slices"
)

// This file is the slots of the risk module (spec-risk.md 9): the scale of the strike
// probability of a route row, set between ticks, held until another call sets it,
// saved in a checkpoint and listed in SlotState.

// RouteRiskScale scales the strike probability of the route row (From, To, Mode): BP
// is 0 to MaxShockBP, 10 000 when none is set (the configured probability), and 0
// makes the row strike never and take no word (SetRouteRiskScales).
type RouteRiskScale struct {
	From, To RegionID
	Mode     ModeID
	BP       int64
}

// routeScaleRec is one saved route scale: the key of a row and its scale.
type routeScaleRec struct {
	routeKey
	bp int64
}

// compareRoute orders two route keys by (From, To, Mode).
func compareRoute(a, b routeKey) int {
	return cmp.Or(cmp.Compare(a.from, b.from), cmp.Compare(a.to, b.to), cmp.Compare(a.mode, b.mode))
}

// SetRouteRiskScales sets the scale of the strike probability of route rows (the risk
// module, spec-risk.md 9): the probability in force is min(10 000, rhe(StrikeBP *
// scale, 10 000)), read at P3.1a and at the record of a dispatch, never by a shipment
// already on the road, which keeps the share it was given. A scale of 10 000 clears the
// key, and it never creates a risk where there is no row. It returns ErrInStep,
// ErrWorldFailed, ErrModuleOff, then for the first entry that fails ErrUnknownRegion,
// ErrUnknownMode, ErrUnknownRiskRow, ErrOutOfBounds or ErrDuplicateID; a refused call
// changes nothing. It emits RouteRiskScaled for each changed key in (From, To, Mode)
// order, after the whole call is applied, and rebuilds the table of expected losses.
func (w *World) SetRouteRiskScales(scales []RouteRiskScale) error {
	if err := w.enter("SetRouteRiskScales", "risk", w.risk != nil); err != nil {
		return err
	}
	r := w.risk
	modes := int64(0)
	if w.transport != nil {
		modes = int64(len(w.transport.modes))
	}
	regions := int64(len(w.cfg.Regions))
	keys, err := slotKeys(scales, func(i int, c RouteRiskScale) (int64, error) {
		for _, f := range [...]struct {
			region RegionID
			field  string
		}{{c.From, "from"}, {c.To, "to"}} {
			if err := w.slotRegion(i, f.region, f.field, false); err != nil {
				return 0, err
			}
		}
		switch {
		case c.Mode < 0 || int64(c.Mode) >= modes:
			return 0, argError(ErrUnknownMode, itoa(int(c.Mode)), "", entry("changes", i), "mode")
		case r.find(c.From, c.To, c.Mode) == nil:
			return 0, argError(ErrUnknownRiskRow, fmt.Sprintf("%d->%d mode %d", c.From, c.To, c.Mode), "", entry("changes", i), "from")
		}
		return (int64(c.From)*regions+int64(c.To))*modes + int64(c.Mode), slotBP(i, c.BP, 0, MaxShockBP, "MaxShockBP")
	}, "from")
	if err != nil {
		return fmt.Errorf("SetRouteRiskScales: %w", err)
	}
	var events []RouteRiskScaled
	for _, i := range inKeyOrder(keys) {
		c := scales[i]
		if before, changed := r.setRouteScale(routeKey{c.From, c.To, c.Mode}, c.BP); changed {
			events = append(events, RouteRiskScaled{Tick: w.Tick(), From: c.From, To: c.To, Mode: c.Mode, BeforeBP: before, AfterBP: c.BP})
		}
	}
	if len(events) > 0 {
		r.rebuild(w.geo.g, len(w.cfg.Regions))
	}
	slotNote(w, "route risk scales set", events)
	return nil
}

// setRouteScale sets the scale of the row of a key, which exists, keeps the sparse
// list of the scales that differ from 10 000 in key order and recomputes the row's
// probability. It returns the scale before and whether it changed.
func (r *risk) setRouteScale(k routeKey, bp int64) (before int64, changed bool) {
	row := r.find(k.from, k.to, k.mode)
	if before = row.scale; before == bp {
		return before, false
	}
	row.scale, row.p = bp, effectiveP(row.strikeBP, bp)
	i, found := slices.BinarySearchFunc(r.scales, k, func(s routeScaleRec, k routeKey) int { return compareRoute(s.routeKey, k) })
	switch {
	case bp == wholeBP:
		r.scales = slices.Delete(r.scales, i, i+1)
	case found:
		r.scales[i].bp = bp
	default:
		r.scales = slices.Insert(r.scales, i, routeScaleRec{k, bp})
	}
	return before, true
}

// riskSlots adds the route scales that are not 10 000 to the slot list, in (From, To,
// Mode) order; it is empty with the module off.
func (w *World) riskSlots(s *SlotState) {
	if w.risk == nil {
		return
	}
	for _, sc := range w.risk.scales {
		s.RouteRiskScales = append(s.RouteRiskScales, RouteRiskScale{From: sc.from, To: sc.to, Mode: sc.mode, BP: sc.bp})
	}
}
