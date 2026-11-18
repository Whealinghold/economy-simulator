package economysim

import (
	"context"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the observation and the checkpoint of the institutions module
// (spec-institutions.md 11, 12): the events of P7.2b, the Warn of a body with waiting
// households, the snapshot rows, and the encoders and decoders of the membership tail
// of tag 3 and of tag 31. It is a continuation file of institutions.go (architecture.md
// 2.6).

// p72bInstitutions is P7.2b: the aggregate events of the institutions module, in
// ascending (RegionID, BodyID) order, after SettlementClassChanged and before phase 8
// (spec-institutions.md 7.3.3). It also advances the run of the Warn of each pair,
// which is a diagnostic and runs whether or not an observer was supplied. It reads the
// counts of P1.9a and writes nothing else.
func (w *World) p72bInstitutions() {
	o := w.org
	for g := range o.nr {
		for b := range o.nb {
			if o.inScope[b*o.nr+g] {
				o.reportCell(w, g, b)
			}
		}
	}
}

// reportCell reports one (region g, body b): the run of the Warn and, with an observer
// and a count that is not zero, the event.
func (o *institutions) reportCell(w *World, g, b int) {
	cell := g*o.nb + b
	out := &o.rd.out[cell]
	var warn bool
	o.run[cell], warn = advanceRun(o.run[cell], out.Waiting > 0, false)
	if warn {
		w.warnWaiting(g, b, out.Waiting)
	}
	if !w.obs.supplied || (out.Admitted == 0 && out.LapsedList == 0 && out.LapsedScope == 0 && out.Skipped == 0 &&
		out.Barred == 0 && out.Waiting == 0) {
		return
	}
	w.obs.deliver(MembershipChanged{
		Tick: w.Tick(), Region: RegionID(g), Body: BodyID(b),
		Members: o.rd.members[cell], Places: o.places[b], Eligible: o.rd.eligible[cell],
		Admitted: out.Admitted, LapsedRole: out.LapsedList, LapsedRegion: out.LapsedScope,
		Skipped: out.Skipped, Barred: out.Barred, Waiting: out.Waiting,
	})
}

// warnWaiting logs the Warn "body has waiting households" for (region g, body b): the
// pair has had households waiting for each of the last tenth ticks, and at every tenth
// after (spec-institutions.md 11.2, pattern B of spec-observability.md 7.3). Its
// counterpart is MembershipChanged.Waiting. It is once per pair, never per household.
func (w *World) warnWaiting(g, b int, waiting int64) {
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelWarn) {
		w.log.LogAttrs(ctx, slog.LevelWarn, "body has waiting households", slog.Int64(logkeys.Region, int64(g)),
			slog.Int64(logkeys.Body, int64(b)), slog.Int64(logkeys.Waiting, waiting), slog.Int64(logkeys.Places, w.org.places[b]))
	}
}

// orgSummary appends the counts of the institutions module to the attributes of the
// construction line (observability-and-logging.md 3.3.2: bodies, rights, controls), and
// nothing with the module off. It is a hook of the construction summary of construct.go.
func (w *World) orgSummary(attrs []slog.Attr) []slog.Attr {
	m := w.cfg.Modules.Institutions
	if m == nil {
		return attrs
	}
	return append(attrs, slog.Int(logkeys.Bodies, len(m.Bodies)), slog.Int(logkeys.Rights, len(m.Rights)),
		slog.Int(logkeys.Controls, len(m.Controls)+len(m.RateLimits)))
}

// snapshotBodies fills Snapshot.Bodies, one row for each (region, body) of the scope in
// ascending (RegionID, BodyID) order, from the households as they are now: the
// members as isMember tests them and the households of the body's roles. A snapshot is
// a function of the saved state, so it does not read the counts of the last round
// (spec-institutions.md 11.5.1, N-125). A nil module fills nothing.
func (w *World) snapshotBodies(s *Snapshot) {
	o := w.org
	if o == nil {
		return
	}
	members, eligible := make([]int64, o.nr*o.nb), make([]int64, o.nr*o.nb)
	for i := w.firstHousehold; i < len(w.agents); i++ {
		a := &w.agents[i]
		g, ok := w.householdRegion(a)
		if !ok {
			continue
		}
		b := o.bodyOf[int(g)*o.nroles+int(a.role)]
		if b < 0 {
			continue
		}
		cell := int(g)*o.nb + int(b)
		eligible[cell] = satAdd64(eligible[cell], a.weight)
		if o.isMember(i, a.role, g, int(b)) {
			members[cell] = satAdd64(members[cell], a.weight)
		}
	}
	for g := range o.nr {
		for b := range o.nb {
			if o.inScope[b*o.nr+g] {
				cell := g*o.nb + b
				s.Bodies = append(s.Bodies, BodyRow{Region: RegionID(g), Body: BodyID(b), Members: members[cell], Eligible: eligible[cell], Places: o.places[b]})
			}
		}
	}
	s.LegalLevels, s.Suspended = o.legalRows(), o.suspendedRows()
}

// legalRows returns the control and rate limit levels that differ from the configured
// ones, by ControlID (spec-institutions.md 11.5.2); nil when none does.
func (o *institutions) legalRows() []LegalRow {
	var rows []LegalRow
	l := &o.legal
	for k := range l.ids {
		if l.capBP[k] != l.baseCap[k] || l.floorBP[k] != l.baseFloor[k] {
			rows = append(rows, LegalRow{Control: ControlID(k), CapBP: l.capBP[k], FloorBP: l.floorBP[k]})
		}
	}
	return rows
}

// suspendedRows returns the suspended rights, by RightID; nil when none is.
func (o *institutions) suspendedRows() []RightID {
	var out []RightID
	for i, s := range o.rights.suspended {
		if s {
			out = append(out, RightID(i))
		}
	}
	return out
}

// saveMembers is the block of tag 3 that follows the agent list when the institutions
// module is on (spec-institutions.md 12.1): for each live agent in ascending ID, the
// body and the region of its admission as BodyID + 1 and RegionID + 1, 0 and 0 for an
// agent with no membership. The field is written for every agent, so the shape of the
// record depends on the module and on nothing in the run's history.
func (w *World) saveMembers(cw *codec.Writer) {
	o := w.org
	w.eachLive(func(a *agent) {
		var b, g int32
		if i := int(a.id); i < len(o.member) {
			b, g = o.member[i], o.memberRegion[i]
		}
		cw.U(uint64(b))
		cw.U(uint64(g))
	})
}

// loadMembers reads the block saveMembers wrote, for each live agent in ascending ID.
// A pair is 0 and 0, or the body and the region (each plus 1) of a defined body and a
// region of its scope; one of the two 0 is corrupt. The membership of a household that
// is no longer eligible is legal: the round of the next tick lapses it
// (spec-institutions.md 12.3).
func (w *World) loadMembers(r *codec.Reader) {
	o := w.org
	o.grow(len(w.agents))
	w.eachLive(func(a *agent) {
		b, g := r.U(), r.U()
		switch {
		case r.Err() != nil:
			return
		case b == 0 && g == 0:
			return
		case b == 0 || g == 0:
			r.Failf("agent %d has the membership (%d, %d): one of the two is 0", a.id, b, g)
		case b > uint64(o.nb) || g > uint64(o.nr) || !o.inScope[int(b-1)*o.nr+int(g-1)]:
			r.Failf("agent %d is a member of body %d in region %d, which is not a body or not in its scope", a.id, int64(b)-1, int64(g)-1)
		default:
			o.member[a.id], o.memberRegion[a.id] = int32(b), int32(g)
		}
	})
}

// saveInstitutions is tag 31 (spec-institutions.md 12.2): the levels that differ from the
// configured ones as (ControlID, CapBP, FloorBP) in ascending ControlID, the suspended
// RightIDs in ascending order, and the runs of the Warn as (region, body, run), sparse.
// With nothing to save it writes three counts of 0, so its shape does not depend on a
// run's history.
func (w *World) saveInstitutions(cw *codec.Writer) error {
	o := w.org
	rows := o.legalRows()
	cw.Count(len(rows))
	for _, row := range rows {
		cw.U(uint64(row.Control))
		cw.U(uint64(row.CapBP))
		cw.U(uint64(row.FloorBP))
	}
	susp := o.suspendedRows()
	cw.Count(len(susp))
	for _, id := range susp {
		cw.U(uint64(id))
	}
	writeSparse(cw, o.run, o.nb)
	return nil
}

// loadInstitutions reads tag 31 into the state built for the configuration and derives
// the tables again. A control is defined and its levels are in bound and differ from the
// configured ones, a right is defined, the IDs ascend, and a run is of a pair of the
// scope and at most the checkpoint's tick (spec-institutions.md 12.3).
func (w *World) loadInstitutions(rs *restoreCtx, r *codec.Reader) error {
	o := w.org
	o.initLevels()
	o.readLevels(r)
	o.readSuspended(r)
	readSparse(r, o.run, o.nr, o.nb, rs.tick)
	for cell, run := range o.run {
		if r.Err() == nil && run != 0 && !o.inScope[(cell%o.nb)*o.nr+cell/o.nb] {
			r.Failf("a Warn run for region %d, body %d, which is not in the body's scope", cell/o.nb, cell%o.nb)
		}
	}
	if r.Err() == nil {
		o.resolve()
	}
	return nil
}

// readLevels reads the levels of tag 31 into the levels in force.
func (o *institutions) readLevels(r *codec.Reader) {
	l := &o.legal
	prev := uint64(0)
	for i := range r.Count(3) {
		k := r.Strict(prev, i > 0)
		capBP, floorBP := r.U(), r.U()
		switch {
		case r.Err() != nil:
			return
		case k >= uint64(len(l.ids)):
			r.Failf("control %d of %d controls", k, len(l.ids))
			return
		case !o.levelsFit(int(k), capBP, floorBP):
			r.Failf("control %d has the levels (%d, %d), out of bound or equal to the configured ones", k, capBP, floorBP)
			return
		}
		l.capBP[k], l.floorBP[k], prev = int64(capBP), int64(floorBP), k
	}
}

// levelsFit reports whether the levels of control k are in bound and differ from the
// configured ones: a control has a cap and a floor of at most MaxBasisPoints with the
// floor at most the cap when a cap is set, a rate limit a cap of at most MaxCreditRateBP
// and no floor.
func (o *institutions) levelsFit(k int, capBP, floorBP uint64) bool {
	l := &o.legal
	if int64(capBP) == l.baseCap[k] && int64(floorBP) == l.baseFloor[k] {
		return false
	}
	if l.isRate[k] {
		return floorBP == 0 && capBP <= MaxCreditRateBP
	}
	return capBP <= MaxBasisPoints && floorBP <= MaxBasisPoints && (capBP == 0 || floorBP <= capBP)
}

// readSuspended reads the suspended rights of tag 31.
func (o *institutions) readSuspended(r *codec.Reader) {
	prev := uint64(0)
	for i := range r.Count(1) {
		k := r.Strict(prev, i > 0)
		if r.Err() != nil {
			return
		}
		if k >= uint64(len(o.rights.suspended)) {
			r.Failf("right %d of %d rights", k, len(o.rights.suspended))
			return
		}
		o.rights.suspended[k], prev = true, k
	}
}
