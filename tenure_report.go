package economysim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	tnr "github.com/Whealinghold/economy-sim/internal/tenure"
)

// This file is the reporting and the saved state of the tenure module (spec-land.md
// 13.4, 14): the tenure fields of the snapshot row of an endowment and the checkpoint
// section of tag 25. The events of the calls are made in tenure_transfer.go.

// fillRow sets the tenure fields of the snapshot row of the endowment at a dense index
// (spec-land.md 13.4): the number of titles, the pool's share, the commons share, the
// largest share of an agent, the posted rent, the scale, whether it is closed and the two
// runs. Every field is a function of saved state.
func (t *tenure) fillRow(row *EndowmentRow, cell int) {
	row.Holders = int64(len(t.holds[cell]))
	var top int64
	for _, x := range t.holds[cell] {
		if x.holder < AgentID(t.regions) {
			row.PoolBP = x.share
		} else {
			top = max(top, x.share)
		}
	}
	row.TopBP, row.CommonsBP = top, wholeBP-sumTitles(t.holds[cell])
	row.Posted, row.ScaleBP, row.Closed = t.posted[cell], t.scale[cell], t.closed[cell]
	row.NoTitleRun, row.RentFundsRun = t.noTitleRun[cell], t.rentFundsRun[cell]
}

// postedSaved says the posted rent of an endowment is written to the checkpoint: the
// step is on and the endowment has a Terms row that is not closed (spec-land.md 14.1).
func (t *tenure) postedSaved(cell int) bool {
	return t.stepping && t.hasTerm[cell] && !t.closed[cell]
}

// saveTenure is tag 25 (spec-land.md 14.1): the endowments with at least one title, each
// as its key, the number of titles and (holder, share) in ascending holder ID; the
// posted rent of each endowment whose step is on, dense; the rent scales that are not
// 10 000, sparse; and the two runs of each endowment with one that is not 0, sparse. All
// are in ascending (region, resource).
func (w *World) saveTenure(cw *codec.Writer) error {
	t := w.tenure
	n := 0
	for _, list := range t.holds {
		if len(list) > 0 {
			n++
		}
	}
	cw.Count(n)
	for cell, list := range t.holds {
		if len(list) == 0 {
			continue
		}
		cw.U(uint64(t.regionOf(cell)))
		cw.U(uint64(t.resourceOf(cell)))
		cw.Count(len(list))
		for _, x := range list {
			cw.U(uint64(x.holder))
			cw.U(uint64(x.share))
		}
	}
	for cell, p := range t.posted {
		if t.postedSaved(cell) {
			cw.U(uint64(p))
		}
	}
	t.saveSparse(cw, func(cell int) bool { return t.scale[cell] != wholeBP }, func(cell int) []int64 { return []int64{t.scale[cell]} })
	t.saveSparse(cw, func(cell int) bool { return t.noTitleRun[cell] != 0 || t.rentFundsRun[cell] != 0 },
		func(cell int) []int64 { return []int64{t.noTitleRun[cell], t.rentFundsRun[cell]} })
	return nil
}

// saveSparse writes the count of the endowments that has says, then for each in order its
// key and the values.
func (t *tenure) saveSparse(cw *codec.Writer, has func(cell int) bool, values func(cell int) []int64) {
	n := 0
	for cell := range t.holds {
		if has(cell) {
			n++
		}
	}
	cw.Count(n)
	for cell := range t.holds {
		if !has(cell) {
			continue
		}
		cw.U(uint64(t.regionOf(cell)))
		cw.U(uint64(t.resourceOf(cell)))
		for _, v := range values(cell) {
			cw.U(uint64(v))
		}
	}
}

// loadTenure reads tag 25 into the state a world with the module has built, which holds
// no title, the configured rent and the default scales, and applies the checks of
// spec-land.md 14.2 in the order of spec-checkpoint.md 6.2: each key is an endowment of
// the configuration (an unknown one is ErrCheckpointConfig) of an ordinary region, keys
// are strictly ascending, each holder is a live agent of the agent section or the pool of
// the endowment's own region, each share is 1 to 10 000, a holder appears once, the
// shares sum to at most 10 000, an agent is not above MaxHolderShareBP, the posted rent
// lies within the band of the configured rent, a scale is not the default and at most
// MaxShockBP, and a run is positive and at most the ticks elapsed.
func (w *World) loadTenure(rs *restoreCtx, r *codec.Reader) error {
	t := w.tenure
	prev := -1
	for range r.Count(4) {
		cell, err := t.readKey(w, r, prev)
		if err != nil || r.Err() != nil {
			return err
		}
		prev = cell
		if err := t.readTitles(w, r, cell); err != nil {
			return err
		}
	}
	if err := t.readPosted(r); err != nil {
		return err
	}
	t.readSparse(w, r, 1, func(cell int, v []int64) {
		if v[0] == wholeBP || v[0] > MaxShockBP {
			r.Failf("rent scale of endowment %s is %d, which is the default or above MaxShockBP", w.endowmentName(cell), v[0])
			return
		}
		t.scale[cell] = v[0]
	})
	t.readSparse(w, r, 2, func(cell int, v []int64) {
		if v[0] > rs.tick || v[1] > rs.tick || v[0]+v[1] == 0 {
			r.Failf("runs of endowment %s are %d and %d, want each at most the tick %d and one above 0", w.endowmentName(cell), v[0], v[1], rs.tick)
			return
		}
		t.noTitleRun[cell], t.rentFundsRun[cell] = v[0], v[1]
	})
	return nil
}

// readKey reads the (region, resource) key of a record and returns its dense index. A
// key outside the configuration is ErrCheckpointConfig, one of an outside node or not
// above the previous key is corrupt.
func (t *tenure) readKey(w *World, r *codec.Reader, prev int) (int, error) {
	region, resource := r.U(), r.U()
	if r.Err() != nil {
		return 0, nil
	}
	if region >= uint64(t.regions) || resource >= uint64(t.resources) {
		return 0, fmt.Errorf("%w: a tenure record names region %d, resource %d, which the configuration does not have", ErrCheckpointConfig, region, resource)
	}
	cell := int(region)*t.resources + int(resource)
	switch {
	case w.isOutsideNode(RegionID(region)):
		r.Failf("a title is written for endowment %s of an outside node", w.endowmentName(cell))
	case cell <= prev:
		r.Failf("endowment %s is not above endowment %d", w.endowmentName(cell), prev)
	}
	return cell, nil
}

// readTitles reads the titles of one endowment and checks each.
func (t *tenure) readTitles(w *World, r *codec.Reader, cell int) error {
	n := r.Count(2)
	if n == 0 && r.Err() == nil {
		r.Failf("endowment %s is written with no title", w.endowmentName(cell))
	}
	var sum int64
	for i := range n {
		holder, share := AgentID(r.Index(uint64(w.state.nextAgent))), r.Bounded("title share", wholeBP)
		if r.Err() != nil {
			return nil
		}
		t.holds[cell] = append(t.holds[cell], title{holder: holder, share: share})
		if err := t.checkTitle(cell, i, t.holds[cell]); err != nil {
			r.Failf("endowment %s: %v", w.endowmentName(cell), err)
			return nil
		}
		sum += share
	}
	if sum > wholeBP {
		r.Failf("the titles of endowment %s sum to %d, above %d", w.endowmentName(cell), sum, wholeBP)
	}
	return nil
}

// readPosted reads the posted rent of each endowment whose step is on and checks it lies
// within [floor, ceiling] of the configured rent, both in basis points of it.
func (t *tenure) readPosted(r *codec.Reader) error {
	m := t.world.cfg.Modules.Tenure
	for cell := range t.posted {
		if !t.postedSaved(cell) {
			continue
		}
		v := r.Bounded("posted rent", MaxPrice)
		lo, ok1 := checked.MulDiv(t.base[cell], m.RentFloorBP, tnr.BasisPoints)
		hi, ok2 := checked.MulDiv(t.base[cell], m.RentCeilingBP, tnr.BasisPoints)
		if r.Err() == nil && (!ok1 || !ok2 || v < lo || v > hi) {
			r.Failf("posted rent %d of endowment %s is outside [%d, %d]", v, t.world.endowmentName(cell), lo, hi)
		}
		t.posted[cell] = v
	}
	return nil
}

// readSparse reads a sparse list: the count, then each record as a key in strictly
// ascending order and k numbers, which set receives.
func (t *tenure) readSparse(w *World, r *codec.Reader, k int, set func(cell int, v []int64)) {
	prev := -1
	v := make([]int64, k)
	for range r.Count(2 + k) {
		region, resource := r.Index(uint64(t.regions)), r.Index(uint64(t.resources))
		for i := range v {
			v[i] = r.Bounded("tenure value", math.MaxInt64)
		}
		cell := int(region)*t.resources + int(resource)
		switch {
		case r.Err() != nil:
			return
		case cell <= prev || w.isOutsideNode(RegionID(region)):
			r.Failf("endowment %s is not above endowment %d, or is of an outside node", w.endowmentName(cell), prev)
			return
		}
		prev = cell
		set(cell, v)
		if r.Err() != nil {
			return
		}
	}
}

// rentsInForce is the rent in force of every endowment, dense by region and resource,
// for the cost anchor of P1.2 (spec-land.md 5.7.1). The slice is the module's scratch.
func (t *tenure) rentsInForce() []int64 {
	if t.rt.rents == nil {
		t.rt.rents = make([]int64, len(t.holds))
	}
	for cell := range t.rt.rents {
		t.rt.rents[cell] = t.rentInForce(cell)
	}
	return t.rt.rents
}

// shortAdjusted is the units short that the Warn "endowment exhausted" reads with
// tenure data: Short less the units of unused entitlement that a closure held and that
// were let and not drawn, which are in the stock (spec-land.md 13.1.4). An endowment
// with no title is as RES says.
func (t *tenure) shortAdjusted(cell int, short int64) int64 {
	if !t.titled(cell) || cell >= len(t.rt.counts) {
		return short
	}
	c := &t.rt.counts[cell]
	return short - min(short, c.held+c.idleUnits())
}

// fillTenure sets what the view carries of tenure (spec-land.md 4.6): per endowment the
// rent in force, the agent's own share and entitlement and whether it is closed, and
// per recipe the rent of one execution and the bound the title puts on MaxExecutions,
// with the reason no_title when a closed tenure leaves no unit open to the agent
// though the stock holds one. It carries no other holder's share.
func (t *tenure) fillTenure(v *AgentView, a *agent) {
	for i := range v.Endowments {
		e := &v.Endowments[i]
		cell := int(v.Region)*t.resources + int(e.Resource)
		e.Posted, e.Closed = t.rentInForce(cell), t.closed[cell]
		e.ShareBP, e.Entitled = t.shareOf(cell, a.id), t.entitled(cell, a.id)
	}
	for i := range v.Recipes {
		t.fillRecipe(&v.Recipes[i], v.Region, a)
	}
}

// fillRecipe sets the rent and the title bound of one recipe of a view.
func (t *tenure) fillRecipe(r *RecipeView, region RegionID, a *agent) {
	res := t.world.res
	for _, d := range res.drawsOf[r.Recipe] {
		cell := int(region)*t.resources + int(d.resource)
		rent, ok := checked.Mul(d.quantity, t.rentInForce(cell))
		sum, added := checked.Add(r.Rent, rent)
		if r.Rent = sum; !ok || !added {
			r.Rent = math.MaxInt64
		}
		if !t.titled(cell) || len(t.rt.ents) == 0 || r.MaxExecutions == 0 {
			continue
		}
		if b := t.open(cell, a.id) / d.quantity; b < r.MaxExecutions {
			r.MaxExecutions = b
			if b == 0 && res.snapStock[cell]/d.quantity > 0 {
				r.Reason = ReasonNoTitle
			}
		}
	}
}

// noteFlow keeps one transfer of rent for the money rows of RentPaid, when an observer is
// supplied: the region of the payer's account, and the lessor's, or the endowment's region
// for the pool.
func (t *tenure) noteFlow(x rentStart, lessor AgentID, paid int64) {
	w, rt := t.world, &t.rt
	if !w.obs.supplied {
		return
	}
	f := rentFlow{cell: x.cell, from: w.regionOf(&w.agents[x.payer]), to: t.regionOf(x.cell), pool: lessor < AgentID(t.regions), amount: paid}
	if !f.pool {
		f.to = w.regionOf(&w.agents[lessor])
	}
	rt.flows = append(rt.flows, f)
}

// p83aRent is P8.3a (spec-land.md 5.5, 8.1): for every endowment that has a title, in
// (RegionID, ResourceID) order, the rent step on the counts of the tick, the runs of the
// refusals and their Warn, the Debug line of the rent charged and the event RentPaid. The
// new posted rent applies from the next tick (P1.2 reads it). The counts are the scratch of
// P2.3 to P2.7; a world with no tick yet has none.
func (w *World) p83aRent() {
	t := w.tenure
	if len(t.rt.counts) == 0 {
		return
	}
	flow := 0
	for cell := range t.holds {
		if !t.titled(cell) {
			continue
		}
		flow = t.rentCell(cell, flow)
	}
}

// rentCell is P8.3a for one endowment; flow is the index of its first transfer of rent in
// the scratch, which is in ascending endowment order, and the result is the index after the
// last.
func (t *tenure) rentCell(cell, flow int) int {
	w := t.world
	c := &t.rt.counts[cell]
	rent, before := t.rentInForce(cell), t.posted[cell]
	after, ok := tnr.Step(tnr.StepIn{Posted: before, Base: t.base[cell], StepBP: t.stepBP, FloorBP: t.floorBP, CeilBP: t.ceilBP,
		Unserved: c.unserved, Idle: c.idleUnits(), Cut: c.cutUnits, Closed: t.closed[cell]})
	if !ok {
		invariant("the rent step of an endowment is out of range after validation")
	}
	t.posted[cell] = after
	t.noTitleRun[cell] = nextRun(t.noTitleRun[cell], c.noTitle)
	t.rentFundsRun[cell] = nextRun(t.rentFundsRun[cell], c.rentFunds)
	t.warnRefusal(cell, ReasonNoTitle, c.noTitle, t.noTitleRun[cell], rent)
	t.warnRefusal(cell, ReasonRentFunds, c.rentFunds, t.rentFundsRun[cell], rent)
	if c.paid > 0 {
		t.debugCharged(cell, rent, c.letUnits)
	}
	if !w.obs.supplied {
		return flow
	}
	e := RentPaid{Tick: w.Tick(), Region: t.regionOf(cell), Resource: t.resourceOf(cell), Rent: Money(rent), Entitled: c.entitle,
		OwnUnits: c.ownUnits, CommonsUnits: c.commonsUnits, LetUnits: c.letUnits, Offered: c.offered, Idle: c.idleUnits(),
		Held: c.held, Unserved: c.unserved, PricedOut: c.pricedOut, Tenants: c.tenants, Lessors: c.lessors,
		Paid: Money(c.paid), Forfeited: Money(c.forfeited), NoTitle: c.noTitle, RentFunds: c.rentFunds,
		CutUnits: c.cutUnits, Exempt: c.exemptUnits, RentAfter: Money(after)}
	flow = t.moneyRows(&e, cell, flow)
	if e.quiet() && after == before {
		return flow
	}
	w.obs.deliver(e)
	return flow
}

// quiet reports that none of the counts that make a RentPaid due is non-zero
// (spec-land.md 13.1.1); the posted rent is tested by the caller.
func (e *RentPaid) quiet() bool {
	return e.OwnUnits|e.CommonsUnits|e.LetUnits|e.Idle|e.Held|e.Unserved|e.PricedOut|e.NoTitle|e.RentFunds|e.CutUnits|
		int64(e.Paid)|int64(e.Forfeited) == 0
}

// moneyRows sets ToPool, From and To of the event from the transfers of the endowment
// that start at flow, and returns the index after them. A row is made for each region with
// a non-zero amount, in ascending RegionID.
func (t *tenure) moneyRows(e *RentPaid, cell, flow int) int {
	fs := t.rt.flows
	for ; flow < len(fs) && fs[flow].cell == cell; flow++ {
		f := fs[flow]
		e.From = addRegionMoney(e.From, f.from, f.amount)
		if f.pool {
			e.ToPool += Money(f.amount)
		} else {
			e.To = addRegionMoney(e.To, f.to, f.amount)
		}
	}
	return flow
}

// addRegionMoney adds an amount to the row of a region in a list kept in ascending RegionID.
func addRegionMoney(list []RegionMoney, region RegionID, amount int64) []RegionMoney {
	i, found := slices.BinarySearchFunc(list, region, func(x RegionMoney, r RegionID) int { return cmp.Compare(x.Region, r) })
	if found {
		list[i].Money += Money(amount)
		return list
	}
	return slices.Insert(list, i, RegionMoney{Region: region, Money: Money(amount)})
}

// nextRun is the run of consecutive ticks with a refusal after a tick with n refusals.
func nextRun(run, n int64) int64 {
	if n > 0 {
		return run + 1
	}
	return 0
}

// warnRefusal writes the Warn "tenure refusal" for an endowment and a reason in a tick with
// n requests refused: at the first tick of a run and at every tenth after it (spec-land.md
// 13.2). Its volume is bounded by regions times resources times two.
func (t *tenure) warnRefusal(cell int, reason Reason, n, run, rent int64) {
	w := t.world
	if n == 0 || (run != 1 && run%10 != 0) {
		return
	}
	ctx := context.Background()
	if !w.log.Enabled(ctx, slog.LevelWarn) {
		return
	}
	w.log.LogAttrs(ctx, slog.LevelWarn, "tenure refusal", slog.Int64(logkeys.Tick, w.Tick()),
		slog.Int(logkeys.Region, int(t.regionOf(cell))), slog.Int(logkeys.Resource, int(t.resourceOf(cell))),
		slog.Int64(logkeys.Rent, rent), slog.Int64(logkeys.Agents, n), slog.String(logkeys.Reason, string(reason)))
}

// debugCharged writes the Debug line "rent charged" for an endowment on which rent was
// paid this tick (spec-land.md 13.2).
func (t *tenure) debugCharged(cell int, rent, units int64) {
	w := t.world
	ctx := context.Background()
	if !w.log.Enabled(ctx, slog.LevelDebug) {
		return
	}
	w.log.LogAttrs(ctx, slog.LevelDebug, "rent charged", slog.Int64(logkeys.Tick, w.Tick()),
		slog.Int(logkeys.Region, int(t.regionOf(cell))), slog.Int(logkeys.Resource, int(t.resourceOf(cell))),
		slog.Int64(logkeys.Rent, rent), slog.Int64(logkeys.Quantity, units))
}
