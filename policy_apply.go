package economysim

import (
	"cmp"
	"context"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/policy"
)

// This file is what an actor sets (spec-policy.md 6; plan 3-plc Phase 5, D4): the checks
// of the returned settings, the three change lists, their application through the
// owners' internal forms in the order of 6.4.2, the event PolicyDecided and the Debug
// lines. Every check is complete before anything is written, and none can fail after
// (6.1.2), so an actor's application is complete or absent.

// transitWrite is one transit key an application writes, flowWrite one flow.
type transitWrite struct {
	from, to RegionID
	mode     ModeID
	bp       int64
}

type flowWrite struct {
	id string
	bp int64
}

// policyScratch is what a decision reuses, so that a due actor allocates nothing beyond
// it once it has grown (plan 3-plc D16): the rows of the view, the inputs and outputs of
// the checks, and the three change lists.
type policyScratch struct {
	levers   []LeverView
	state    []policy.Lever
	entries  []policy.Entry
	check    policy.Scratch
	edges    []EdgeStatus
	transits []transitWrite
	flows    []flowWrite
}

// policyDecide is the decision of one due actor: the view, the policy, the checks and the
// application. A hold actor was skipped by the caller. An actor of a consumer's policy or of
// a directed one has no decision until the change that adds them (the consumer's draws
// and the mailbox), so it does nothing here.
func (w *World) policyDecide(id ActorID, a *policyActor, pass *policyPass) {
	if a.builtin == nil {
		return
	}
	v, ok := w.policyView(id, a, pass)
	if !ok {
		return
	}
	set := a.builtin.Decide(v, nil)
	tally := w.policyCheck(set)
	w.policyWrite(a)
	hold := a.heldCounts()
	w.policyReport(id, a, tally, hold)
}

// policyCheck converts the returned settings to entries, runs the checks against the
// levers' state read for the view, and returns the tally (6.2).
func (w *World) policyCheck(set Settings) policy.Tally {
	s := &w.policy.scratch
	s.entries = s.entries[:0]
	for _, e := range set.Levers {
		s.entries = append(s.entries, policy.Entry{Lever: int(e.Lever), BP: e.BP, Closed: e.Closed})
	}
	return policy.Check(s.state, s.entries, &s.check)
}

// policyWrite builds the three change lists from the entries the checks applied and writes
// them in the order of 6.4.2: the edges in one call (the graph is rebuilt once), then each
// transit key in ascending (from, to, mode), then each flow in ascending flow ID. The
// owners' setters cannot refuse a value that passed validation and the checks, so a refusal
// is a programmer error, as in the public setters (6.1.2).
func (w *World) policyWrite(a *policyActor) {
	s := &w.policy.scratch
	s.edges, s.transits, s.flows = s.edges[:0], s.transits[:0], s.flows[:0]
	for i, e := range s.entries {
		r := &s.check.Results[i]
		if r.Outcome == policy.Applied {
			w.policyChange(&a.levers[e.Lever], r)
		}
	}
	if len(s.edges) > 0 {
		if err := w.setEdgeStatuses(s.edges, CausePolicy); err != nil {
			invariant("validated edge statuses of a policy actor were refused: " + err.Error())
		}
	}
	slices.SortFunc(s.transits, func(x, y transitWrite) int {
		return cmp.Or(cmp.Compare(x.from, y.from), cmp.Compare(x.to, y.to), cmp.Compare(x.mode, y.mode))
	})
	for _, t := range s.transits {
		if err := w.setTransitFactor(t.from, t.to, t.mode, t.bp, CausePolicy); err != nil {
			invariant("a validated transit factor of a policy actor was refused: " + err.Error())
		}
	}
	slices.SortFunc(s.flows, func(x, y flowWrite) int { return cmp.Compare(x.id, y.id) })
	for _, f := range s.flows {
		if err := w.setScaleOf(f.id, f.bp, CausePolicy); err != nil {
			invariant("a validated flow scale of a policy actor was refused: " + err.Error())
		}
	}
}

// policyChange adds the keys of lever l, set to the result r, to the list of its kind. An
// edge group takes the whole value on every key: the number and the closed flag together.
func (w *World) policyChange(l *policyLever, r *policy.Result) {
	s := &w.policy.scratch
	switch l.kind {
	case LeverFlow:
		s.flows = append(s.flows, flowWrite{id: l.flow, bp: r.Value})
	case LeverTransit:
		s.transits = append(s.transits, transitWrite{from: l.from, to: l.to, mode: l.mode, bp: r.Value})
	default:
		for _, id := range l.edges {
			e := w.geo.g.view(int(id))
			s.edges = append(s.edges, EdgeStatus{From: e.From, To: e.To, Closed: r.Closed, FactorBP: r.Value})
		}
	}
}

// policyReport emits PolicyDecided and the Debug lines of one decision, after the owners'
// slot events (12.1.3). Nothing is emitted for a decision with no entry and no reaction held
// for a reason, and the event is built only when an observer was supplied.
func (w *World) policyReport(id ActorID, a *policyActor, t policy.Tally, hold heldCounts) {
	if t.Entries == 0 && hold.inBand == 0 && hold.noReading == 0 {
		return
	}
	if w.obs.supplied {
		w.obs.deliver(PolicyDecided{
			Tick: w.Tick(), Actor: id, Jurisdiction: a.j, Entries: t.Entries, Applied: t.Applied, Unchanged: t.Unchanged,
			NotHeld: t.NotHeld, Duplicate: t.Duplicate, NoClose: t.NoClose, Clamped: t.Clamped, Stepped: t.Stepped,
			InBand: hold.inBand, NoReading: hold.noReading,
		})
	}
	ctx := context.Background()
	if !w.log.Enabled(ctx, slog.LevelDebug) {
		return
	}
	w.log.LogAttrs(ctx, slog.LevelDebug, "policy decided", slog.String(logkeys.Actor, a.id),
		slog.Int64(logkeys.Entries, t.Entries), slog.Int64(logkeys.Changed, t.Applied))
	w.policyDebugLines(a)
}

// policyDebugLines writes the Debug lines of the refused entries and of the levers a rule
// moved.
func (w *World) policyDebugLines(a *policyActor) {
	ctx := context.Background()
	s := &w.policy.scratch
	for i, e := range s.entries {
		if reason := policyRefusal(s.check.Results[i].Outcome); reason != "" {
			w.log.LogAttrs(ctx, slog.LevelDebug, "policy entry refused", slog.String(logkeys.Actor, a.id),
				slog.String(logkeys.Lever, a.leverID(e.Lever)), slog.String(logkeys.Reason, reason))
		}
	}
	if a.rule == nil {
		return
	}
	for _, m := range a.rule.moves {
		w.log.LogAttrs(ctx, slog.LevelDebug, "rule moved lever", slog.String(logkeys.Actor, a.id),
			slog.String(logkeys.Lever, a.leverID(int(m.lever))), slog.Int64(logkeys.Reading, m.reading),
			slog.Int64(logkeys.From, m.from), slog.Int64(logkeys.To, m.to))
	}
}

// policyRefusal is the reason of the Debug line of a refused entry, "" for an entry that was
// not refused.
func policyRefusal(o policy.Outcome) string {
	switch o {
	case policy.NotHeld:
		return "not_held"
	case policy.Duplicate:
		return "duplicate"
	case policy.NoClose:
		return "no_close"
	}
	return ""
}

// heldCounts are the reactions a rule left at rest or with no reading in one decision.
type heldCounts struct{ inBand, noReading int64 }

// heldCounts returns what the actor's rule held in its last decision, and zero for an actor
// of another policy.
func (a *policyActor) heldCounts() heldCounts {
	if a.rule == nil {
		return heldCounts{}
	}
	return heldCounts{inBand: a.rule.inBand, noReading: a.rule.noReading}
}

// leverID is the ID string of lever i for a log line; a lever outside the list is written
// as its index.
func (a *policyActor) leverID(i int) string {
	if i < 0 || i >= len(a.levers) {
		return itoa(i)
	}
	return a.levers[i].id
}
