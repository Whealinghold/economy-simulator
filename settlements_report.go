package economysim

import (
	"context"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/settlement"
)

// This file is the observation and the checkpoint of the settlements module
// (spec-settlements.md 10, 11): the two events, the two Debug lines, the snapshot
// fields, the names table and the encoder and decoder of tag 23. It is the
// continuation file of settlements.go (architecture.md 2.6).

// settlementNames returns the string ID of each settlement class, by
// SettlementClassID, or nil with the module off (Names.SettlementClasses).
func settlementNames(c *Config) []string {
	m := c.Modules.Settlements
	if m == nil {
		return nil
	}
	out := make([]string, len(m.Classes))
	for k := range m.Classes {
		out[k] = m.Classes[k].ID
	}
	return out
}

// announce is the founding of the classes in the events and logs: one
// SettlementClassChanged per region that is not an outside node, in ascending
// RegionID, with From NoSettlementClass and the cause founded, after the AgentSpawned
// events of the initial agents (spec-settlements.md 5.4.1). A nil module reports
// nothing. The household counts are those found() took: nothing between the two adds
// or reweights an agent (the warm start moves goods and money only).
func (s *settlements) announce(w *World) {
	if s == nil {
		return
	}
	for g := range s.class {
		if !w.isOutsideNode(RegionID(g)) {
			w.reportClassChanged(RegionID(g), NoSettlementClass, SettlementClassID(s.class[g]), s.hh[g], CauseFounded)
		}
	}
}

// reportClassChanged emits SettlementClassChanged, only when an observer was
// supplied, and logs the Debug line "settlement class changed" when Debug is enabled
// (spec-settlements.md 10.1, 10.2). Both come after the class is set.
func (w *World) reportClassChanged(g RegionID, from, to SettlementClassID, households int64, cause Cause) {
	if w.obs.supplied {
		w.obs.deliver(SettlementClassChanged{Tick: w.Tick(), Region: g, From: from, To: to, Households: households, Cause: cause})
	}
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "settlement class changed",
			slog.Int64(logkeys.Region, int64(g)), slog.Int64(logkeys.FromSettlementClass, int64(from)),
			slog.Int64(logkeys.ToSettlementClass, int64(to)), slog.Int64(logkeys.Households, households),
			slog.String(logkeys.Cause, string(cause)))
	}
}

// reportClassHeld emits SettlementHeld, only when an observer was supplied, and logs
// the Debug line "settlement class held" when Debug is enabled.
func (w *World) reportClassHeld(g RegionID, class, target SettlementClassID, households, ticksLeft int64) {
	if w.obs.supplied {
		w.obs.deliver(SettlementHeld{Tick: w.Tick(), Region: g, Class: class, Target: target, Households: households, TicksLeft: ticksLeft})
	}
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "settlement class held",
			slog.Int64(logkeys.Region, int64(g)), slog.Int64(logkeys.SettlementClass, int64(class)),
			slog.Int64(logkeys.TargetSettlementClass, int64(target)), slog.Int64(logkeys.Households, households))
	}
}

// fillRegionClass sets the four class fields of the row of region g, whose
// Households the row already holds: the saved class and Since, and the derived class
// a review at the current count would give, ignoring the dwell, and whether the
// region is at its floor (spec-settlements.md 10.5).
func (w *World) fillRegionClass(rr *RegionRow, g RegionID) {
	s := w.set
	if s == nil {
		return
	}
	target, atFloor := settlement.Target(s.class[g], s.floor[g], rr.Households, s.e[g], s.l[g])
	rr.Class, rr.ClassSince = SettlementClassID(s.class[g]), s.since[g]
	rr.ClassTarget, rr.ClassAtFloor = SettlementClassID(target), atFloor
}

// saveSettlements writes tag 23 (spec-checkpoint.md 4.2, spec-settlements.md 11):
// for each region that is not an outside node, in ascending RegionID, the class and
// the tick of its last change as unsigned varints.
func (w *World) saveSettlements(cw *codec.Writer) error {
	s := w.set
	for g := range s.class {
		if !w.isOutsideNode(RegionID(g)) {
			cw.U(uint64(s.class[g]))
			cw.U(uint64(s.since[g]))
		}
	}
	return nil
}

// loadSettlements reads tag 23 into the state built for the configuration. A class
// must be below the number of classes and not below the floor of its region, and
// Since at most the checkpoint's tick less 1 (0 at tick 0): the last change ran in a
// tick that has ended (spec-settlements.md 11). It runs no review and emits nothing
// (spec-settlements.md 4.5.2).
func (w *World) loadSettlements(rs *restoreCtx, r *codec.Reader) error {
	s := w.set
	for g := range s.class {
		if w.isOutsideNode(RegionID(g)) {
			continue
		}
		at := r.Offset()
		c := int(r.Index(uint64(s.n)))
		if r.Err() == nil && c < s.floor[g] {
			r.Failf("byte %d: class %d of region %d is below its floor %d", at, c, g, s.floor[g])
		}
		s.class[g], s.since[g] = c, r.Bounded("settlement since", max(rs.tick-1, 0))
	}
	return nil
}
