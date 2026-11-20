package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

// This file is the tolls of the fiscal module (spec-fiscal.md 7, 10.1; a continuation
// file of fiscal.go, architecture.md 2.6): a charge per carrying unit on a shipment
// for each edge of its shortest path that crosses a border of the authority. The toll
// is a step of the dispatch, not a position: the check of row 22a refuses the
// shipment whole if the shipper cannot pay the total (ckToll), step 3a pays it
// (chargeToll), and P3.6c reports the tolls of the tick (p36cTolls). Nothing is drawn
// (FR-FIS-062).

// fiscalToll is what a toll derives from its configuration once: whether it acts on
// leaving or on entering its authority, the mode it charges (-1 is every mode) and its
// amount. The regions the authority covers are the flow's target.in.
type fiscalToll struct {
	exit   bool
	mode   int
	amount int64
}

// tollCharge is the toll of one crossing: the flow that charges it and the amount.
type tollCharge struct {
	flow int
	due  int64
}

// newToll resolves a toll's side and mode. A mode that is not defined is a violated
// invariant: the validator rejects it, and -1 would charge every mode in its place.
func newToll(c *Config, t *Toll) fiscalToll {
	out := fiscalToll{exit: t.On == TollExit, mode: -1, amount: t.Amount}
	if t.Mode != "" {
		out.mode = slices.IndexFunc(c.Modules.Transport.Modes, func(m Mode) bool { return m.ID == t.Mode })
		if out.mode < 0 {
			invariant("a toll names a mode that is not defined: " + t.ID)
		}
	}
	return out
}

// acts reports whether the toll charges the edge from a to b for a shipment of the
// mode: the crossing of the edge has the authority in its Exit (the toll is on exit)
// or its Enter, which is the authority covering the one end of the edge and not the
// other (spec-fiscal.md 7.2.1, spec-polity.md 4.1.1).
func (f *fiscalFlowState) acts(a, b RegionID, mode ModeID) bool {
	if f.toll.mode >= 0 && f.toll.mode != int(mode) {
		return false
	}
	in := f.target.in
	if f.toll.exit {
		return in[a] && !in[b]
	}
	return in[b] && !in[a]
}

// charges appends to dst the tolls of a shipment of n carrying units from one region
// to another by a mode, for each edge of the shortest path in path order and each toll
// in ascending ID that acts on it at the tick, with the due of each above 0 (7.2, 7.3):
// floor(Amount * S * max(1, n) / 10^4), one truncation per crossing, and their total.
// ok is false when a due or the total leaves int64, which TRN reports as the refusal
// overflow.
func (s *fiscalState) charges(w *World, dst []tollCharge, from, to RegionID, mode ModeID, n int64) (out []tollCharge, total int64, ok bool) {
	tick, season := w.Tick(), w.tk.moment.Season
	for cur := from; cur != to; {
		next, found := w.geo.next(cur, to)
		if !found {
			return dst, total, true // a route that has no path owes nothing: row 7 refuses it
		}
		for i := s.tolls[0]; i < s.tolls[1]; i++ {
			f := &s.flows[i]
			if !f.active(tick, season) || !f.acts(cur, next, mode) {
				continue
			}
			due, dok := fiscal.Toll(f.toll.amount, f.scale, n)
			if dok && due > 0 {
				total, dok = checked.Add(total, due)
				dst = append(dst, tollCharge{flow: i, due: due})
			}
			if !dok {
				return dst, 0, false
			}
		}
		cur = next
	}
	return dst, total, true
}

// ckToll is row 22a of spec-transport.md 5.4, toll_unaffordable: the fiscal module is
// on and the shipper's available money, its balance less what is reserved, is below
// the total toll of the path. It also refuses as overflow a toll or a total that leaves
// int64 or that would take a treasury above MaxMoney, which the transfer of step 3a
// would refuse. It leaves the tolls of the request in the module's scratch and their
// total in c.toll. It is empty while the module is off.
func ckToll(c *shipCtx) Reason {
	s := c.w.fiscal
	if s == nil || s.tolls[0] == s.tolls[1] {
		return ""
	}
	var ok bool
	s.paid, c.toll, ok = s.charges(c.w, s.paid[:0], c.region, c.r.To, c.r.Mode, c.n)
	switch {
	case !ok:
		return ReasonOverflow
	case c.toll == 0:
		return ""
	case c.toll > c.w.available(c.a.account):
		return ReasonTollUnaffordable
	case !c.w.tollsFit(s.paid):
		return ReasonOverflow
	}
	return ""
}

// tollsFit reports whether every treasury the tolls pay can take them: its balance and
// the tolls it is paid together stay within MaxMoney.
func (w *World) tollsFit(paid []tollCharge) bool {
	s := w.fiscal
	for _, ch := range paid {
		tr := s.flows[ch.flow].treasury
		var sum int64
		for _, other := range paid {
			if s.flows[other.flow].treasury == tr {
				sum = satAdd(sum, other.due)
			}
		}
		if sum > fiscal.Headroom(w.balance(s.pol.tr[tr].account), MaxMoney) {
			return false
		}
	}
	return true
}

// chargeToll is step 3a of spec-transport.md 5.5.1: for each crossing in path order and
// each toll in ascending ID with a due above 0, the shipper pays the treasury of the
// authority under the toll's reason, nets it from its income of the tick (X_t) and the
// counters of P3.6c take it. The check of row 22a ran, so the ledger cannot refuse.
func (w *World) chargeToll(c *shipCtx) {
	s := w.fiscal
	for k, ch := range s.paid {
		f := &s.flows[ch.flow]
		if err := w.ledger.Transfer(c.a.account, s.pol.tr[f.treasury].account, ch.due, f.reason); err != nil {
			invariant("the ledger refused a checked toll: " + err.Error())
		}
		w.netCharge(f, fiscalPayer{owner: c.a.id}, ch.due)
		g := &s.agg[ch.flow*s.regions+int(c.region)]
		g.due, g.applied = satAdd(g.due, ch.due), satAdd(g.applied, ch.due)
		if !s.chargedBefore(k) {
			g.parties = satAdd(g.parties, c.a.weight)
		}
	}
}

// chargedBefore reports whether an earlier toll of the same shipment was of the same
// flow as toll k: a shipper is a party to a flow once for each shipment.
func (s *fiscalState) chargedBefore(k int) bool {
	for _, ch := range s.paid[:k] {
		if ch.flow == s.paid[k].flow {
			return true
		}
	}
	return false
}

// viewToll is the toll a RouteRow shows: the money one carrying unit pays in tolls on
// the shortest path from one region to another by a mode, at the scales and in the
// windows of the tick, a pure read of the configuration, the scales and the graph
// (spec-fiscal.md 7.5). The route table is built at P3.1 for the tick, so it is the
// cache. It is 0 with the module off, with no toll and for a route that does not exist.
func (w *World) viewToll(from, to RegionID, mode ModeID, has bool) int64 {
	s := w.fiscal
	if !has || s == nil || s.tolls[0] == s.tolls[1] {
		return 0
	}
	var total int64
	var ok bool
	s.view, total, ok = s.charges(w, s.view[:0], from, to, mode, 1)
	if !ok {
		return 0
	}
	return total
}

// p36cTolls is P3.6c (spec-world.md 6, spec-fiscal.md 10.1): after the aggregates of
// phase 3, the tolls charged at dispatch in P3.2(c) and P3.3, per flow in ascending ID
// and region ascending, as FiscalFlowApplied. It then clears the counters of the tolls.
func (w *World) p36cTolls() {
	s := w.fiscal
	s.ran = s.ran[:0]
	for i := s.tolls[0]; i < s.tolls[1]; i++ {
		s.ran = append(s.ran, i)
	}
	w.reportFiscal(s, w.Tick())
	for _, pos := range s.ran {
		clear(s.agg[pos*s.regions : (pos+1)*s.regions])
	}
}
