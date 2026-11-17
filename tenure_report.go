package economysim

import (
	"fmt"
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
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
