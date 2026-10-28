package economysim

import (
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/endowment"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// This file is the resources module at run time (spec-resources.md 5, 6;
// architecture.md 2.10.1): the snapshot of P2.3, the requests that P2.5 records,
// the ration of P2.6, the cap and the take of the start at P2.7, the read of a
// stock and the agent's view of the endowments of its region. The state and the
// derived tables are in resources.go.

// Endowment returns the units of the resource available in the region now
// (spec-resources.md 5.1). It returns 0 for an unknown region or resource, for a
// pair with no capacity and while the module is off. It is a read, safe between
// steps and on a failed world.
func (w *World) Endowment(region RegionID, resource ResourceID) int64 {
	r := w.res
	if r == nil || region < 0 || int(region) >= r.regions || resource < 0 || int(resource) >= r.resources {
		return 0
	}
	return r.stock[r.cellOf(region, resource)]
}

// takeEndowment takes n units of an endowment whole, or none: it returns
// ErrInsufficientEndowment when n exceeds the stock (spec-resources.md 5.1).
// Production alone calls it, during the start of a job. n is at least 1.
func (r *resourcesState) takeEndowment(cell int, n int64) error {
	if n < 1 || n > r.stock[cell] {
		return fmt.Errorf("take %d units of endowment %d holding %d: %w", n, cell, r.stock[cell], ErrInsufficientEndowment)
	}
	r.stock[cell] -= n
	r.drawn[cell] += n
	return nil
}

// p23Snapshot is P2.3: for every endowment, in (RegionID, ResourceID) order, the
// stock and the abundance are recorded, and they are the same for every agent in
// the tick whatever the order of the starts (spec-resources.md 5.2 step 1, 4.7.3,
// FR-RES-015). It reads the stocks the last regeneration left, so a call between
// ticks is in it. The requests of the last tick are dropped.
func (w *World) p23Snapshot() {
	r := w.res
	if r == nil {
		return
	}
	r.reqs = r.reqs[:0]
	for cell := range r.stock {
		r.snapStock[cell] = r.stock[cell]
		r.snapAbundance[cell] = r.abundanceOf(cell, r.stock[cell])
	}
}

// recordDraws is the record of P2.5 for an admitted request: one request on the
// endowment of each draw of the recipe, for the k0 executions the agent could
// start ignoring draws, in the order P2.5 walks the rotation (spec-resources.md
// 5.2 step 2, D386). It returns the index of the first record and their count,
// which the start at P2.7 reads; a recipe with no draw records none.
func (w *World) recordDraws(a *agent, region RegionID, recipe RecipeID, k0 int64) (first, n int) {
	r := w.res
	if r == nil {
		return 0, 0
	}
	first = len(r.reqs)
	for _, d := range r.drawsOf[recipe] {
		r.reqs = append(r.reqs, resRequest{agent: a.id, cell: r.cellOf(region, d.resource), executions: k0, draw: d.quantity})
	}
	return first, len(r.reqs) - first
}

// p26Ration is P2.6: for each endowment some request draws on, in order, the
// requests are rationed against the snapshot of the stock by the rule of
// spec-resources.md 5.3 (internal/endowment), each in agent-ID order and keyed by
// the stateless hash of the tick and the agent. If the stock covers them all every
// request is granted in full, in one sum. The grant of each request, and the
// counts the event of 14.1 reads, are recorded. No random word is drawn.
func (w *World) p26Ration() {
	r := w.res
	if r == nil || len(r.reqs) == 0 {
		return
	}
	for len(r.grants) < len(w.agents) {
		r.grants = append(r.grants, 0)
	}
	r.order = r.order[:0]
	for i := range r.reqs {
		r.order = append(r.order, i)
	}
	// The order is total: an agent makes at most one request on an endowment.
	slices.SortFunc(r.order, func(a, b int) int {
		x, y := &r.reqs[a], &r.reqs[b]
		if x.cell != y.cell {
			return x.cell - y.cell
		}
		return int(x.agent) - int(y.agent)
	})
	for lo, hi := 0, 0; lo < len(r.order); lo = hi {
		hi = r.runEnd(lo)
		r.rationRun(lo, hi, w.Tick())
	}
}

// runEnd is the end of the run of requests on one endowment that starts at lo in
// the sorted order.
func (r *resourcesState) runEnd(lo int) int {
	cell, hi := r.reqs[r.order[lo]].cell, lo
	for hi < len(r.order) && r.reqs[r.order[hi]].cell == cell {
		hi++
	}
	return hi
}

// rationRun rations the requests order[lo:hi], all on one endowment, against its
// snapshot, records the grant of each and counts the units asked and granted and
// the requests granted nothing and less than they asked.
func (r *resourcesState) rationRun(lo, hi int, tick int64) {
	cell := r.reqs[r.order[lo]].cell
	r.ration = r.ration[:0]
	for _, i := range r.order[lo:hi] {
		q := &r.reqs[i]
		// The Refs are the agent IDs, distinct because a recipe draws on a resource
		// once and an agent makes one request a tick (internal/endowment.Ration).
		if n := len(r.ration); n > 0 && r.ration[n-1].Ref == int(q.agent) {
			invariant("an agent made two requests on one endowment in a tick")
		}
		r.ration = append(r.ration, endowment.Request{
			Ref: int(q.agent), Executions: q.executions, Draw: q.draw, Key: ordering.Key(tick, int64(q.agent)),
		})
	}
	if !endowment.Ration(r.snapStock[cell], r.ration, r.grants) {
		invariant("the ration of an endowment refused its requests after validation")
	}
	for _, i := range r.order[lo:hi] {
		q := &r.reqs[i]
		q.grant = r.grants[q.agent]
		r.requested[cell] += q.executions * q.draw
		r.granted[cell] += q.grant * q.draw
		switch {
		case q.grant == 0:
			r.denied[cell]++
		case q.grant < q.executions:
			r.reduced[cell]++
		}
	}
}

// capDraws sets the cap of a start request from the grants of the draws of the
// admitted request at records [first, first+n): the least over them of the grant
// and the executions the live stock covers (spec-resources.md 5.2 step 4, 5.4.1).
// A request with no draw is left as it is. Clipping to the live stock makes the
// take whole or nothing: a stock lowered since P2.6 refuses with no_endowment.
func (w *World) capDraws(q *startReq, first, n int) {
	if n == 0 {
		return
	}
	r := w.res
	q.capped, q.drawCap = true, q.weight
	for _, d := range r.reqs[first : first+n] {
		q.drawCap = min(q.drawCap, d.grant, r.stock[d.cell]/d.draw)
	}
}

// takeDraws takes k times each draw of a started job from the live stock, one
// call per draw (spec-resources.md 5.4.1). The cap of the start made each take
// fit, so a refusal is a violated invariant.
func (w *World) takeDraws(first, n int, k int64) {
	r := w.res
	for _, d := range r.reqs[first : first+n] {
		units, ok := checked.Mul(k, d.draw)
		if !ok {
			invariant("the units drawn by a start do not fit after validation")
		}
		if err := r.takeEndowment(d.cell, units); err != nil {
			invariant("a start took more than the endowment holds: " + err.Error())
		}
	}
}

// viewCap is the cap the draws put on the executions a view reports: for each
// draw, floor(snapshot stock / draw), a bound that ignores the competition of the
// ration (spec-resources.md 5.5.1). has is false for a recipe with no draw.
func (r *resourcesState) viewCap(region RegionID, recipe RecipeID) (limit int64, has bool) {
	ds := r.drawsOf[recipe]
	if len(ds) == 0 {
		return 0, false
	}
	limit = r.snapStock[r.cellOf(region, ds[0].resource)] / ds[0].quantity
	for _, d := range ds[1:] {
		limit = min(limit, r.snapStock[r.cellOf(region, d.resource)]/d.quantity)
	}
	return limit, true
}

// fillEndowments sets the Endowments group of a view: the endowments of the
// agent's own region that its role's recipes draw on, sorted by resource, with the
// stock now, the effective capacity and the abundance of the tick
// (spec-resources.md 5.5.1). It carries no endowment of another region.
func (w *World) fillEndowments(v *AgentView, a *agent) {
	r, sc := w.res, v.scratch
	rows := sc.endowments[:0]
	for _, id := range r.roleRes[a.role] {
		cell := r.cellOf(v.Region, id)
		rows = append(rows, EndowmentView{
			Resource: id, Stock: r.stock[cell], Capacity: r.effectiveCapacity(cell), AbundanceBP: r.snapAbundance[cell],
		})
	}
	sc.endowments = rows
	v.Endowments = rows[:len(rows):len(rows)]
}
