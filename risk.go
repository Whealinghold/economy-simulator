package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	rsk "github.com/Whealinghold/economy-sim/internal/risk"
)

// This file is the risk module (spec-risk.md): the loss of cargo in transit and of
// stored goods as data. The pure rules are in internal/risk, imported as rsk
// because the root's type is risk. The file holds, in the order of architecture.md
// 2.10.1: the ledger reason the module declares, its state and its construction,
// the common route rows of P3.1a, the strike of a dispatched shipment (step 4a of
// spec-transport.md 5.5.1) and the table of expected losses. The configuration
// validator is in risk_validate.go, the events and logs in risk_report.go.

// riskReasons returns the ledger reasons the risk module declares: the one goods
// sink, loss (spec-risk.md 7.1). Its boundary accounts are opened goods
// commodity-major in the order of the Modules pointers (spec-world.md 5.2 row 3a).
func riskReasons() []ledger.Reason {
	return []ledger.Reason{ledger.Loss}
}

// routeRow is one route row resolved to indices (spec-risk.md 3.2.1): the key, the
// two numbers and the flag, the scale of its slot (10 000 until a call sets it), the
// effective probability the scale gives and, for a common row, the outcome P3.1a
// drew for the tick.
type routeRow struct {
	from, to   RegionID
	mode       ModeID
	strikeBP   int64
	severityBP int64
	common     bool
	scale      int64
	p          int64
	struck     bool
}

// routeKey is the key of a route row.
type routeKey struct {
	from, to RegionID
	mode     ModeID
}

// expectRow is one entry of the table of expected losses (spec-risk.md 8.1.3): a
// route with a loss above 0. A route that is not in the table has no loss.
type expectRow struct {
	routeKey
	lossBP int64
}

// risk is the state of the risk module, nil when it is off (spec-risk.md 15). The
// rows and the table are derived and never saved; the scales, the saved state, join
// with the calls that set them.
type risk struct {
	// rows is the count of route and store rows, for the construction summary.
	rows int
	// routes are the route rows in canonical order (From, To, Mode as indices).
	routes []routeRow
	// expect is the table of expected losses, ascending by key, rebuilt at
	// construction and at every call that changes an edge status or a scale. edges
	// and modes are the scratch of the rebuild.
	expect []expectRow
	edges  []rsk.Edge
	modes  []ModeID
}

// newRisk builds the module from the canonical configuration, or nil with the module
// off: the rows resolved to indices at the neutral scale, and the table of expected
// losses over the graph g. A row of a validated configuration names a region and a
// mode that exist.
func newRisk(c *Config, g Graph) *risk {
	m := c.Modules.Risk
	if m == nil {
		return nil
	}
	region := make(map[string]RegionID, len(c.Regions)) // lookup only
	for i := range c.Regions {
		region[c.Regions[i].ID] = RegionID(i)
	}
	mode := map[string]ModeID{} // lookup only
	if t := c.Modules.Transport; t != nil {
		for i := range t.Modes {
			mode[t.Modes[i].ID] = ModeID(i)
		}
	}
	r := &risk{rows: len(m.Routes) + len(m.Stores), routes: make([]routeRow, 0, len(m.Routes))}
	for i := range m.Routes {
		row := &m.Routes[i]
		rr := routeRow{
			from: region[row.From], to: region[row.To], mode: mode[row.Mode],
			strikeBP: row.StrikeBP, severityBP: row.SeverityBP, common: row.Common, scale: wholeBP,
		}
		rr.p = effectiveP(rr.strikeBP, rr.scale)
		r.routes = append(r.routes, rr)
	}
	r.rebuild(g, len(c.Regions))
	return r
}

// effectiveP is the effective probability of a row, P = min(10 000, rhe(StrikeBP *
// Scale, 10 000)) (spec-risk.md 3.4.1). Its arguments come from a validated row and
// a checked scale, so the result always exists.
func effectiveP(strikeBP, scale int64) int64 {
	p, ok := rsk.Prob(strikeBP, scale)
	if !ok {
		invariant("the effective probability of a risk row did not fit after validation")
	}
	return p
}

// find returns the route row of an edge and a mode, or nil. It allocates nothing.
func (r *risk) find(a, b RegionID, m ModeID) *routeRow {
	i, ok := slices.BinarySearchFunc(r.routes, routeKey{a, b, m}, func(row routeRow, k routeKey) int {
		return cmp.Or(cmp.Compare(row.from, k.from), cmp.Compare(row.to, k.to), cmp.Compare(row.mode, k.mode))
	})
	if !ok {
		return nil
	}
	return &r.routes[i]
}

// p31aCommon is P3.1a (spec-risk.md 4.2): one word, the first of the stream keyed
// by the tick, the From region and To * MaxModes + Mode (D13), for each common route
// row with a probability above 0, in canonical order. The outcome is kept in the row
// for this tick only and is never saved. A common row draws in a tick with nothing
// on the road (3.3.3); a row with a probability of 0 takes no word (3.4.3).
func (w *World) p31aCommon() {
	r := w.risk
	tick := w.Tick()
	for i := range r.routes {
		row := &r.routes[i]
		row.struck = false
		if !row.common || row.p == 0 {
			continue
		}
		w.rnd.keyedPair(siteRiskCommon, tick, int64(row.from), int64(row.to)*MaxModes+int64(row.mode))
		row.struck = rsk.Struck(w.rnd.draw(wholeBP), row.p)
	}
}

// riskStrike is step 4a of a dispatch (spec-risk.md 4.1.1): it walks the path of the
// shipment over the world's own next-hop table and, for each edge with a route row
// of the shipment's mode and a probability above 0, takes the outcome of the row: a
// common row's outcome of P3.1a, or one word of the stream keyed by the tick and the
// shipment (D14), taken in path order and keyed only if some unit row needs a word.
// It returns LossBP, 10 000 less the kept share after the struck rows' severities
// fold in path order. A path with no such row takes no word and gives 0.
func (w *World) riskStrike(s *shipment) int64 {
	r := w.risk
	kept := int64(wholeBP)
	keyed := false
	for a, hops := s.from, 0; a != s.to && hops < len(w.cfg.Regions); hops++ {
		b, ok := w.geo.next(a, s.to)
		if !ok {
			break
		}
		row := r.find(a, b, s.mode)
		a = b
		if row == nil || row.p == 0 {
			continue
		}
		hit := row.struck
		if !row.common {
			if !keyed {
				w.rnd.keyed(siteRiskDispatch, w.Tick(), int64(s.id))
				keyed = true
			}
			hit = rsk.Struck(w.rnd.draw(wholeBP), row.p)
		}
		if hit {
			kept = foldKept(kept, row.severityBP)
		}
	}
	loss := wholeBP - kept
	if loss > 0 {
		w.logRouteStruck(s, loss)
	}
	return loss
}

// foldKept folds one struck row's severity into the kept share (spec-risk.md 4.1.1).
// Its arguments come from a share the module keeps within 0 to 10 000 and a
// validated row.
func foldKept(kept, severityBP int64) int64 {
	k, ok := rsk.Fold(kept, severityBP)
	if !ok {
		invariant("the kept share of a shipment left 0 to 10 000")
	}
	return k
}

// expectedLoss is the expected loss in basis points of a route of a mode: the value
// of the table rebuilt at the last call that changed an edge status or a scale
// (spec-risk.md 8.1.3), 0 for a route with no loss, with the module off (a nil
// receiver) or with no path. It allocates nothing.
func (r *risk) expectedLoss(from, to RegionID, mode ModeID) int64 {
	if r == nil {
		return 0
	}
	k := routeKey{from, to, mode}
	i, ok := slices.BinarySearchFunc(r.expect, k, func(e expectRow, k routeKey) int {
		return cmp.Or(cmp.Compare(e.from, k.from), cmp.Compare(e.to, k.to), cmp.Compare(e.mode, k.mode))
	})
	if !ok {
		return 0
	}
	return r.expect[i].lossBP
}

// viewLoss is the expected loss a RouteRow shows: the table value of a route that
// exists, 0 with the module off or for a route that does not.
func (r *risk) viewLoss(from, to RegionID, mode ModeID, has bool) int64 {
	if !has {
		return 0
	}
	return r.expectedLoss(from, to, mode)
}

// rebuild recomputes the table of expected losses over the graph g from the rows at
// their scales (spec-risk.md 8.1.1): for each ordered pair of regions with a path and
// each mode that has a row with a probability above 0, the loss of the rows met along
// the path. It runs at construction and at the calls that change an edge status or
// a scale, never in a tick (R6).
func (r *risk) rebuild(g Graph, regions int) {
	r.expect = r.expect[:0]
	r.modes = r.modes[:0]
	for i := range r.routes {
		if row := &r.routes[i]; row.p > 0 && !slices.Contains(r.modes, row.mode) {
			r.modes = append(r.modes, row.mode)
		}
	}
	slices.Sort(r.modes)
	for from := range regions {
		for to := range regions {
			if from == to {
				continue
			}
			for _, m := range r.modes {
				k := routeKey{RegionID(from), RegionID(to), m}
				if loss := r.pathLoss(g, k); loss > 0 {
					r.expect = append(r.expect, expectRow{k, loss})
				}
			}
		}
	}
}

// pathLoss is the expected loss of one route (spec-risk.md 8.1.1): the rows with a
// probability above 0 on the edges of the path of g, in path order, through
// rsk.Expected. No path gives 0.
func (r *risk) pathLoss(g Graph, k routeKey) int64 {
	r.edges = r.edges[:0]
	for a, hops := k.from, 0; a != k.to; hops++ {
		b, ok := g.NextHop(a, k.to)
		if !ok || hops > len(g.names) {
			return 0
		}
		if row := r.find(a, b, k.mode); row != nil && row.p > 0 {
			r.edges = append(r.edges, rsk.Edge{P: row.p, SeverityBP: row.severityBP})
		}
		a = b
	}
	return expectedOf(r.edges)
}

// expectedOf is rsk.Expected for rows the module built within 0 to 10 000.
func expectedOf(edges []rsk.Edge) int64 {
	loss, ok := rsk.Expected(edges)
	if !ok {
		invariant("the expected loss of a route left 0 to 10 000")
	}
	return loss
}

// configuredLoss is the expected loss of a lane of a mode from the configuration
// alone (spec-risk.md 8.3): the rows at the scale of 10 000 over the path of the
// configured graph g with every edge open. c is canonical. It reads no world.
func configuredLoss(c *Config, g Graph, from, to RegionID, mode string) int64 {
	m := c.Modules.Risk
	if m == nil {
		return 0
	}
	var edges []rsk.Edge
	for a, hops := from, 0; a != to; hops++ {
		b, ok := g.NextHop(a, to)
		if !ok || hops > len(c.Regions) {
			return 0
		}
		i, found := slices.BinarySearchFunc(m.Routes, [3]string{c.Regions[a].ID, c.Regions[b].ID, mode}, func(row RouteRisk, k [3]string) int {
			return cmp.Or(cmp.Compare(row.From, k[0]), cmp.Compare(row.To, k[1]), cmp.Compare(row.Mode, k[2]))
		})
		if found {
			if p := effectiveP(m.Routes[i].StrikeBP, wholeBP); p > 0 {
				edges = append(edges, rsk.Edge{P: p, SeverityBP: m.Routes[i].SeverityBP})
			}
		}
		a = b
	}
	return expectedOf(edges)
}
