package economysim

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/endowment"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// This file is the resources module at run time (spec-resources.md 5, 6;
// architecture.md 2.10.1): the snapshot of P2.3, the requests that P2.5 records,
// the ration of P2.6, the cap and the take of the start at P2.7, the read of a
// stock and the agent's view of the endowments of its region, then the
// regeneration of P8.3 with its event and its Warn, the shock and adjustment
// helpers and the audit of the identity of an endowment. The state and the derived
// tables are in resources.go.

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
		w.logRationed(r.reqs[r.order[lo]].cell)
	}
}

// logRationed writes the Debug line "draw rationed" for an endowment whose requests
// the ration had to cut (spec-resources.md 14.2): once per endowment per tick.
func (w *World) logRationed(cell int) {
	r := w.res
	short := r.requested[cell] - r.granted[cell]
	if short == 0 {
		return
	}
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "draw rationed", slog.Int64(logkeys.Tick, w.Tick()),
			slog.Int(logkeys.Region, cell/r.resources), slog.Int(logkeys.Resource, cell%r.resources),
			slog.Int64(logkeys.Quantity, r.requested[cell]), slog.Int64(logkeys.Short, short))
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

// p83Regen is P8.3: every endowment, in (RegionID, ResourceID) order, regenerates
// once with the season of the tick, from the stock the draws left (spec-resources.md
// 4.1). It then advances the run of ticks with units short, writes the Warn
// "endowment exhausted" and emits EndowmentChanged. The counts stay until P8.6, so
// the audit of P8.4 reads them.
func (w *World) p83Regen() {
	r := w.res
	if r == nil {
		return
	}
	season := w.tk.moment.Season
	for cell := range r.stock {
		r.regen(cell, season)
		w.reportEndowment(cell)
	}
}

// regen applies one tick of regeneration to an endowment (internal/endowment.Regen)
// and records the units kept.
func (r *resourcesState) regen(cell, season int) {
	res := cell % r.resources
	m, ok := endowment.EffectiveScale(r.regenRow[cell], r.regenScale[cell])
	if !ok {
		invariant("the regeneration scale of an endowment does not fit after validation")
	}
	factor := int64(wholeBP)
	if sb := r.seasonBP[res]; season >= 0 && season < len(sb) {
		factor = sb[season]
	}
	out, ok := endowment.Regen(
		endowment.Params{Shape: r.shape[res], Rate: r.rate[res], Refuge: r.refuge[res],
			Capacity: r.effectiveCapacity(cell), ScaleBP: m, TicksPerYear: r.ticksPerYear},
		endowment.Step{Stock: r.stock[cell], Carry: r.carry[cell], SeasonBP: factor})
	if !ok {
		invariant("the regeneration of an endowment is out of range after validation")
	}
	r.stock[cell], r.carry[cell], r.regenerated[cell] = out.Stock, out.Carry, out.Added
}

// reportEndowment advances the run of an endowment and reports its tick: the Warn at
// the first tick with units short and every tenth consecutive one, and the event if
// any of the units drawn, regenerated, short or lost is non-zero (spec-resources.md
// 14.1, 14.2). Without tenure data the Warn's condition is Short above 0.
func (w *World) reportEndowment(cell int) {
	r := w.res
	short := r.requested[cell] - r.granted[cell]
	if short > 0 {
		r.exhaustedRun[cell]++
	} else {
		r.exhaustedRun[cell] = 0
	}
	if run := r.exhaustedRun[cell]; short > 0 && (run == 1 || run%10 == 0) {
		ctx := context.Background()
		if w.log.Enabled(ctx, slog.LevelWarn) {
			w.log.LogAttrs(ctx, slog.LevelWarn, "endowment exhausted", slog.Int64(logkeys.Tick, w.Tick()),
				slog.Int(logkeys.Region, cell/r.resources), slog.Int(logkeys.Resource, cell%r.resources),
				slog.Int64(logkeys.Stock, r.stock[cell]), slog.Int64(logkeys.Short, short))
		}
	}
	if !w.obs.supplied || r.drawn[cell]+r.regenerated[cell]+short+r.lost[cell] == 0 {
		return
	}
	w.obs.deliver(EndowmentChanged{
		Tick: w.Tick(), Region: RegionID(cell / r.resources), Resource: ResourceID(cell % r.resources),
		Stock: r.stock[cell], Capacity: r.effectiveCapacity(cell), AbundanceBP: r.snapAbundance[cell],
		Requested: r.requested[cell], Granted: r.granted[cell], Drawn: r.drawn[cell], Short: short,
		Regenerated: r.regenerated[cell], Lost: r.lost[cell], AgentsDenied: r.denied[cell], AgentsReduced: r.reduced[cell],
	})
}

// setCapacityScales sets the capacity scale of the shock slot of one endowment (the
// helper that plan 17's SetCapacityScales wraps; the caller has checked the key
// and the range 0 to MaxShockBP). The effective capacity is recomputed; a stock
// above it is clamped and the excess counted as lost. Nothing happens, and
// nothing is emitted, if the scale is the one in force.
func (w *World) setCapacityScales(region RegionID, resource ResourceID, bp int64) {
	r := w.res
	cell := r.cellOf(region, resource)
	if r.capScale[cell] == bp {
		return
	}
	before := r.stock[cell]
	r.capScale[cell] = bp
	var lost int64
	if c := r.effectiveCapacity(cell); before > c {
		lost, r.stock[cell] = before-c, c
		r.lost[cell] += lost
	}
	w.deliverAdjusted(cell, EndowmentAdjusted{Kind: AdjustKindShock, StockBefore: before, Lost: lost})
}

// setRegenScales sets the regeneration scale of the shock slot of one endowment,
// as setCapacityScales does for the capacity; the stock is not touched.
func (w *World) setRegenScales(region RegionID, resource ResourceID, bp int64) {
	r := w.res
	cell := r.cellOf(region, resource)
	if r.regenScale[cell] == bp {
		return
	}
	r.regenScale[cell] = bp
	w.deliverAdjusted(cell, EndowmentAdjusted{Kind: AdjustKindShock, StockBefore: r.stock[cell]})
}

// adjustEndowments changes the stock of one endowment by a signed delta clamped to
// [0, effective capacity] (spec-resources.md 6.3.2; the caller has checked the key
// and that delta is at most MaxEndowment in magnitude). The change that took
// effect is counted as adjusted and reported as Delta; a delta with no effect
// emits nothing.
func (w *World) adjustEndowments(region RegionID, resource ResourceID, delta int64) {
	r := w.res
	cell := r.cellOf(region, resource)
	before := r.stock[cell]
	sum, ok := checked.Add(before, delta)
	if !ok {
		invariant("an endowment adjustment does not fit after validation")
	}
	after := min(max(sum, 0), r.effectiveCapacity(cell))
	if after == before {
		return
	}
	r.stock[cell] = after
	r.adjusted[cell] += after - before
	w.deliverAdjusted(cell, EndowmentAdjusted{Kind: AdjustKindAdjust, Delta: after - before, StockBefore: before})
}

// deliverAdjusted completes and delivers the EndowmentAdjusted of a call on one endowment:
// the key, the tick the next step runs, the scales and the stock after.
func (w *World) deliverAdjusted(cell int, e EndowmentAdjusted) {
	if !w.obs.supplied {
		return
	}
	r := w.res
	e.Tick, e.Region, e.Resource = w.Tick(), RegionID(cell/r.resources), ResourceID(cell%r.resources)
	e.CapacityBP, e.RegenBP, e.StockAfter = r.capScale[cell], r.regenScale[cell], r.stock[cell]
	w.obs.deliver(e)
}

// auditEndowments is the endowment part of P8.4 (spec-resources.md 6.4, FR-RES-025,
// FR-RES-005): every stock lies in [0, effective capacity] and equals the stock at
// the last boundary less the units drawn plus those regenerated, less those lost
// plus those adjusted. The first endowment that disagrees is named with the tick and
// the difference.
func (w *World) auditEndowments() error {
	r := w.res
	if r == nil {
		return nil
	}
	for cell, got := range r.stock {
		want := r.base[cell] - r.drawn[cell] + r.regenerated[cell] - r.lost[cell] + r.adjusted[cell]
		if got != want || got < 0 || got > r.effectiveCapacity(cell) {
			return auditFailure(w.Tick(), fmt.Errorf("endowment %q in %q: stock %d, want %d from %d at the last boundary, %d drawn, %d regenerated, %d lost and %d adjusted (difference %d), capacity %d",
				w.cfg.Modules.Resources.Resources[cell%r.resources].ID, w.cfg.Regions[cell/r.resources].ID,
				got, want, r.base[cell], r.drawn[cell], r.regenerated[cell], r.lost[cell], r.adjusted[cell], got-want, r.effectiveCapacity(cell)))
		}
	}
	return nil
}
