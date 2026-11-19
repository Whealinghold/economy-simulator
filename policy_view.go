package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/policy"
)

// This file is what an actor reads (spec-policy.md 5; plan 3-plc Phase 5, D6): the
// current value of each lever from its owner's slot, the treasury, the region states of
// the covered regions, and the view that holds them. A view is built fresh for each due
// actor and is a function of the saved state, the tick and the configuration, so a
// restored world builds the same one and an observer changes nothing (5.2.1).

// leverReading is what the world reads of one lever's keys: the number of the first key
// (3.4.4), whether some key has another number, and whether some key is open and some is
// closed (the inputs of the checks of 6.2).
type leverReading struct {
	bp        int64
	closed    bool // the first key is closed
	mixed     bool
	anyOpen   bool
	anyClosed bool
}

// readLever reads the keys of lever l from their owners.
func (w *World) readLever(l *policyLever) leverReading {
	switch l.kind {
	case LeverFlow:
		return leverReading{bp: w.flowScaleOf(l.flow), anyOpen: true}
	case LeverTransit:
		return leverReading{bp: w.transport.factorOf(l.from, l.to, l.mode), anyOpen: true}
	}
	return w.readEdges(l.edges)
}

// readEdges reads the keys of an edge lever. A group with no edge reads the default and
// has no open and no closed key.
func (w *World) readEdges(edges []EdgeID) leverReading {
	r := leverReading{bp: policyDefault}
	for i, id := range edges {
		e := w.geo.g.view(int(id))
		if i == 0 {
			r.bp, r.closed = e.FactorBP, e.Closed
		}
		r.mixed = r.mixed || e.FactorBP != r.bp
		r.anyOpen = r.anyOpen || !e.Closed
		r.anyClosed = r.anyClosed || e.Closed
	}
	return r
}

// flowScaleOf returns the scale of the flow with the ID, a money flow or a fiscal flow, as
// setScaleOf resolves it. The ID was found at validation.
func (w *World) flowScaleOf(id string) int64 {
	if at := slices.IndexFunc(w.moneyFlowList(), func(f MoneyFlow) bool { return f.ID == id }); at >= 0 {
		return w.money.flows[at].scale
	}
	if w.fiscal != nil {
		if at := slices.IndexFunc(w.fiscal.flows, func(f fiscalFlowState) bool { return f.id == id }); at >= 0 {
			return w.fiscal.flows[at].scale
		}
	}
	invariant("a policy lever names a flow that is neither a money flow nor a fiscal flow: " + id)
	return 0
}

// policyTreasury returns the balance of the treasury of jurisdiction j at the boundary,
// and false for a grouping.
func (w *World) policyTreasury(j JurisdictionID) (Money, bool) {
	p := w.polity
	if p == nil || p.byJ[j] < 0 {
		return 0, false
	}
	v, err := w.ledger.Account(p.tr[p.byJ[j]].account)
	if err != nil {
		invariant("the treasury account of a policy actor's jurisdiction cannot be read: " + err.Error())
	}
	return Money(v.Balance), true
}

// policyPass is what the actors of one P0.3a share (plan 3-plc D6): the moment of the
// tick, found once, and whether the cost anchors were prepared. The anchors read the
// prices, wages, yield shocks and tools, none of which an actor's write changes (a flow
// scale, an edge status or a transit factor), so the first actor that reads regions
// prepares them for the others.
type policyPass struct {
	moment   Moment
	haveMoms bool
	anchored bool
}

// policyMoment returns the moment of the tick, false when the calendar cannot place it
// (P1.1 then fails the step with that error and no actor decides).
func (w *World) policyMoment(pass *policyPass) bool {
	if !pass.haveMoms {
		m, err := w.cfg.Calendar.At(w.Tick())
		if err != nil {
			return false
		}
		pass.moment, pass.haveMoms = m, true
	}
	return true
}

// policyAnchors prepares the cost anchors once for the pass, before the first region
// state is read, and reports false when it cannot (the calendar: P1.1 fails the step).
func (w *World) policyAnchors(pass *policyPass) bool {
	if pass.anchored {
		return true
	}
	m, err := w.prepareAnchors()
	if err != nil {
		return false
	}
	pass.moment, pass.haveMoms, pass.anchored = m, true, true
	return true
}

// policyView builds the view of actor id: the levers by LeverID with the limits and
// the current value of each, the treasury, and with ReadsRegions the state of each
// covered region. The levers' rows and the inputs of the checks are written to the
// scratch in one pass (s.levers, s.state). Levers is that scratch, not a copy: only the
// built-in policies are decided here and none keeps the view, so D16's zero allocation holds;
// the change that decides a consumer's policy must hand it a copy (FR-PLC-025). It reports false when the view cannot be built.
func (w *World) policyView(id ActorID, a *policyActor, pass *policyPass) (PolicyView, bool) {
	if !w.policyMoment(pass) {
		return PolicyView{}, false
	}
	s := &w.policy.scratch
	s.levers, s.state = s.levers[:0], s.state[:0]
	for i := range a.levers {
		l := &a.levers[i]
		r := w.readLever(l)
		s.levers = append(s.levers, LeverView{
			Lever: LeverID(i), Kind: l.kind, BP: r.bp, Closed: r.closed, Default: policyDefault,
			Min: l.min, Max: l.max, MaxStep: l.maxStep, MayClose: l.mayClose,
		})
		s.state = append(s.state, policy.Lever{
			Min: l.min, Max: l.max, MaxStep: l.maxStep, MayClose: l.mayClose,
			Current: r.bp, Mixed: r.mixed, AnyOpen: r.anyOpen, AnyClosed: r.anyClosed,
		})
	}
	v := PolicyView{Tick: w.Tick(), Moment: pass.moment, Actor: id, Jurisdiction: a.j, Seat: a.seat, Levers: s.levers}
	v.Treasury, v.HasTreasury = w.policyTreasury(a.j)
	if a.reads && !w.policyRegions(&v, a, pass) {
		return PolicyView{}, false
	}
	return v, true
}

// policyRegions fills v.Regions with the state of each covered region that is not an
// outside node, in ascending RegionID: the rows of a snapshot filtered by region
// (FR-OBS-044). The cost is one snapshot's scan per region (NFR-PLC-002).
func (w *World) policyRegions(v *PolicyView, a *policyActor, pass *policyPass) bool {
	if !w.policyAnchors(pass) {
		return false
	}
	v.Moment = pass.moment
	v.Regions = make([]RegionState, 0, len(a.covered))
	for _, g := range a.covered {
		rs, err := w.regionStateOf(g)
		if err != nil {
			invariant("the state of a region a policy actor covers cannot be read: " + err.Error())
		}
		v.Regions = append(v.Regions, rs)
	}
	return true
}
