package economysim

import (
	"cmp"
	"fmt"
	"slices"

	tnr "github.com/Whealinghold/economy-sim/internal/tenure"
)

// This file is the tenure module (spec-land.md; architecture.md 2.10.1): its state
// (the titles of each endowment, the posted rent, the rent scale and the two
// diagnostic runs, kept in slices indexed by the dense endowment index of the
// resources module, N-244), the construction of the titles from the holdings of the
// configuration (step 8a of spec-world.md 5.1), the share arithmetic the calls and the
// removal share and the audit of the shares. The calls and the queries and the step of
// a removal are in tenure_transfer.go, the events, the snapshot rows and the checkpoint
// section in tenure_report.go and the validator in tenure_validate.go. Outside these
// files the module appears only in the catalogue files (config.go, hash.go, bounds.go,
// defaults.go, errors.go, event.go, validate.go) and at the named hooks of ADR 0010: the
// field of World, the construction call, the step of a removal in removeAgent, the audit,
// the section row of the checkpoint, the snapshot rows and the slot list.

// TitleState is one title: the right of Holder to ShareBP basis points of the snapshot
// stock of the endowment of Resource in Region (spec-land.md 3.1, 9.3).
type TitleState struct {
	Region   RegionID
	Resource ResourceID
	// Holder is the agent that holds the title, or the estate pseudo-agent of Region
	// for the pool, whose ID is its RegionID.
	Holder  AgentID
	ShareBP int64
}

// HoldingsState is the tenure of one endowment at the boundary (spec-land.md 9.3).
type HoldingsState struct {
	// CommonsBP is 10 000 less the sum of the titles.
	CommonsBP int64
	// Closed says the unused entitlement of a holder is not let.
	Closed bool
	// Posted is the posted rent before the scale, money per unit let.
	Posted int64
	// ScaleBP is the rent scale, 10 000 by default.
	ScaleBP int64
	// Titles are the titles in ascending holder ID.
	Titles []TitleState
}

// TitleTransfer moves a share of one endowment between two holders (spec-land.md 6.3).
type TitleTransfer struct {
	Region   RegionID
	Resource ResourceID
	// From and To are the holders; NoAgent names the commons: From NoAgent is an
	// enclosure, To NoAgent a release.
	From, To AgentID
	// ShareBP is the share that moves, 1 to 10 000.
	ShareBP int64
	// Price is the money To pays From with the reason title_sale; 0 is a gift.
	Price Money
}

// title is one title of an endowment: the holder and the share in basis points.
type title struct {
	holder AgentID
	share  int64
}

// tenure is the state of the module, nil when it is off. Everything but the titles, the
// posted rent (while the step is on), the scales and the runs is derived from the
// configuration and never saved (spec-land.md 14.1).
type tenure struct {
	world *World
	// regions and resources are the dimensions of the resources module; the dense index
	// of an endowment is region * resources + resource.
	regions, resources int
	// holds is, per endowment, its titles in ascending holder ID: the pool, whose ID is
	// a RegionID, comes before every agent. Saved.
	holds [][]title
	// base is the configured rent of each endowment, hasTerm says it has a Terms row and
	// closed says its unused entitlement is not let. Derived.
	base    []int64
	hasTerm []bool
	closed  []bool
	// posted is the posted rent of each endowment: the configured rent until the step of
	// the rent moves it (spec-land.md 5.2.1). Saved while stepping is on.
	posted []int64
	// scale is the rent scale of each endowment, 10 000 by default. Saved when not.
	scale []int64
	// noTitleRun and rentFundsRun are the consecutive ticks with requests refused for
	// each reason (spec-land.md 13.2). Saved when not 0.
	noTitleRun, rentFundsRun []int64
	// capBP is MaxHolderShareBP, the most one agent may hold; the pool is exempt.
	capBP int64
	// stepping says the posted rent is state: RentStepBP is above 0.
	stepping bool
}

// newTenure builds the state of the module from a canonical configuration that passed
// Validate, with no title, or returns nil when the module is off. The titles are made
// by found once the agents exist.
func newTenure(c *Config) *tenure {
	m := c.Modules.Tenure
	if m == nil || c.Modules.Resources == nil {
		return nil
	}
	t := &tenure{regions: len(c.Regions), resources: len(c.Modules.Resources.Resources),
		capBP: cmp.Or(m.MaxHolderShareBP, DefaultMaxHolderShareBP), stepping: m.RentStepBP > 0}
	n := t.regions * t.resources
	t.holds, t.base, t.hasTerm, t.closed = make([][]title, n), make([]int64, n), make([]bool, n), make([]bool, n)
	t.posted, t.scale = make([]int64, n), make([]int64, n)
	t.noTitleRun, t.rentFundsRun = make([]int64, n), make([]int64, n)
	for i := range t.scale {
		t.scale[i] = wholeBP
	}
	for i := range m.Terms {
		tm := &m.Terms[i]
		cell := t.cellOfNames(c, tm.Region, tm.Resource)
		if cell < 0 || t.hasTerm[cell] {
			continue
		}
		t.hasTerm[cell], t.closed[cell] = true, tm.HasClosed && tm.Closed
		if tm.HasRent {
			t.base[cell] = tm.Rent
		}
		t.posted[cell] = t.base[cell]
	}
	return t
}

// cellOfNames is the dense index of the endowment of a region and a resource by their
// IDs, or -1 for an ID the configuration does not have.
func (t *tenure) cellOfNames(c *Config, region, resource string) int {
	g := slices.IndexFunc(c.Regions, func(r Region) bool { return r.ID == region })
	r := slices.IndexFunc(c.Modules.Resources.Resources, func(x Resource) bool { return x.ID == resource })
	if g < 0 || r < 0 {
		return -1
	}
	return g*t.resources + r
}

// regionOf and resourceOf are the reverse of the dense index.
func (t *tenure) regionOf(cell int) RegionID     { return RegionID(cell / t.resources) }
func (t *tenure) resourceOf(cell int) ResourceID { return ResourceID(cell % t.resources) }

// found makes the titles of the holdings of the configuration, in the order of the
// rows: for each row a role holder gives every household agent of the role whose
// account is in the row's region, in ascending agent ID, a title of weight times
// ShareBP, merged with one it already holds; the pool holder gives the estate
// pseudo-agent of the region a title of ShareBP. It emits no event, draws nothing and
// charges no rent (spec-land.md 6.1). The configuration passed Validate, so a bound
// that fails here is a violated invariant.
func (t *tenure) found(w *World) {
	m := &w.cfg.Modules.Tenure.Holdings
	for i := range *m {
		h := &(*m)[i]
		cell := t.cellOfNames(&w.cfg, h.Region, h.Resource)
		if cell < 0 {
			invariant("a holding names an endowment the configuration does not have")
		}
		if h.Holder.Kind == HolderKindPool {
			t.give(cell, AgentID(t.regionOf(cell)), h.ShareBP)
			continue
		}
		role := slices.IndexFunc(w.cfg.Roles, func(r Role) bool { return r.ID == h.Holder.Role })
		for k := w.firstHousehold; k < len(w.agents); k++ {
			if a := &w.agents[k]; int(a.role) == role && w.regionOf(a) == t.regionOf(cell) {
				share, ok := tnr.HolderShare(a.weight, h.ShareBP)
				if !ok {
					invariant("the title of a household overflows after validation")
				}
				t.give(cell, a.id, share)
			}
		}
	}
}

// give adds share to the title of holder on the endowment, making it when there is none,
// and keeps the order. A sum above 10 000 or an agent above the cap is a violated
// invariant: validation computes both from the census.
func (t *tenure) give(cell int, holder AgentID, share int64) {
	list := t.holds[cell]
	i, found := slices.BinarySearchFunc(list, holder, func(x title, h AgentID) int { return cmp.Compare(x.holder, h) })
	if found {
		list[i].share += share
	} else {
		t.holds[cell] = slices.Insert(list, i, title{holder: holder, share: share})
		i = slices.IndexFunc(t.holds[cell], func(x title) bool { return x.holder == holder })
	}
	got := t.holds[cell][i].share
	if t.sum(cell) > wholeBP || (holder >= AgentID(t.regions) && got > t.capBP) {
		invariant("a title passes the sum of 10 000 or the cap of the holder after validation")
	}
}

// sum is the sum of the titles of an endowment, never above 10 000.
func (t *tenure) sum(cell int) int64 {
	var s int64
	for _, x := range t.holds[cell] {
		s += x.share
	}
	return s
}

// shareOf is the share a holder has of an endowment, 0 for none.
func (t *tenure) shareOf(cell int, holder AgentID) int64 {
	return shareIn(t.holds[cell], holder)
}

// shareIn is the share of a holder in a sorted list of titles, 0 for none.
func shareIn(list []title, holder AgentID) int64 {
	i, found := slices.BinarySearchFunc(list, holder, func(x title, h AgentID) int { return cmp.Compare(x.holder, h) })
	if !found {
		return 0
	}
	return list[i].share
}

// setShare writes the share of a holder in a sorted list of titles, removing the title
// when it is 0, and returns the list.
func setShare(list []title, holder AgentID, share int64) []title {
	i, found := slices.BinarySearchFunc(list, holder, func(x title, h AgentID) int { return cmp.Compare(x.holder, h) })
	switch {
	case found && share == 0:
		return slices.Delete(list, i, i+1)
	case found:
		list[i].share = share
	case share > 0:
		return slices.Insert(list, i, title{holder: holder, share: share})
	}
	return list
}

// rentInForce is the rent of an endowment in a tick: the posted rent times the scale
// (spec-land.md 5.2.2).
func (t *tenure) rentInForce(cell int) int64 {
	r, ok := tnr.RentInForce(t.posted[cell], t.scale[cell])
	if !ok {
		invariant("the rent in force overflows after validation")
	}
	return r
}

// holderLive reports whether a holder ID is a title holder of the endowment now: the
// estate pseudo-agent of the endowment's own region, or a live agent that is not a
// pseudo-agent.
func (t *tenure) holderLive(cell int, holder AgentID) bool {
	if holder < AgentID(t.regions) {
		return holder == AgentID(t.regionOf(cell)) && !t.world.isOutsideNode(RegionID(holder))
	}
	return t.world.liveAgent(holder)
}

// auditTenure is the tenure part of the audit of P8.4 and of a restore (spec-land.md
// 14.2): on every endowment the shares are 1 to 10 000, the titles are in strictly
// ascending holder order, their sum is at most 10 000, each holder is live, and no agent
// is above the cap of MaxHolderShareBP.
func (w *World) auditTenure() error {
	t := w.tenure
	if t == nil {
		return nil
	}
	for cell, list := range t.holds {
		var sum int64
		for i, x := range list {
			if err := t.checkTitle(cell, i, list); err != nil {
				return fmt.Errorf("tenure: endowment %s: %w", w.endowmentName(cell), err)
			}
			sum += x.share
		}
		if sum > wholeBP {
			return fmt.Errorf("tenure: endowment %s: the shares sum to %d, above %d", w.endowmentName(cell), sum, wholeBP)
		}
	}
	return nil
}

// checkTitle checks the title i of an endowment's list on its own: its share, its place
// in the order, its holder and the cap.
func (t *tenure) checkTitle(cell, i int, list []title) error {
	x := list[i]
	switch {
	case x.share < 1 || x.share > wholeBP:
		return fmt.Errorf("holder %d has share %d, outside 1 to %d", x.holder, x.share, wholeBP)
	case i > 0 && list[i-1].holder >= x.holder:
		return fmt.Errorf("holder %d is not above holder %d", x.holder, list[i-1].holder)
	case !t.holderLive(cell, x.holder):
		return fmt.Errorf("holder %d is not a live holder", x.holder)
	case x.holder >= AgentID(t.regions) && x.share > t.capBP:
		return fmt.Errorf("holder %d has share %d, above the cap %d", x.holder, x.share, t.capBP)
	}
	return nil
}
