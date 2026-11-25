package economysim

import (
	"context"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/insure"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the strategy side of the insurance module (spec-insurance.md 5.1, 10): the
// requests and the offer a plan carries, the group of the view, the filler of that group,
// the check of the plan and the default rules of the insured and of the underwriter,
// exported so that a wrapper strategy can call them (architecture.md 4.7).

// LaneCover asks for a route policy on every shipment of the agent dispatched this tick in
// a mode to a destination. It names no shipment: the round finds them (spec-insurance.md
// 5.1.2). Both limits are bounds the round checks.
type LaneCover struct {
	// Mode is the mode of carriage and To the destination region of the shipments.
	Mode ModeID
	To   RegionID
	// MinCoverBP is the least share of a loss value it accepts, 1 to 10 000.
	MinCoverBP int64
	// MaxLoadingBP is the highest loading it accepts, 0 to MaxLoadingBound.
	MaxLoadingBP int64
}

// StoreCover asks for a store policy on this tick: a share of the value of the goods the
// agent holds, which the world computes (spec-insurance.md 6.3.3).
type StoreCover struct {
	// ShareBP is the share of its held value to insure, 1 to 10 000.
	ShareBP int64
	// MinCoverBP is the least share of a loss value it accepts, 1 to 10 000.
	MinCoverBP int64
	// MaxStoreRateBP is the highest rate per tick it accepts, 0 to MaxStoreRateBound.
	MaxStoreRateBP int64
}

// CoverOffer offers capacity at posted terms. MaxLiability is money: the most the agent
// will add to its open liability this tick (spec-insurance.md 5.1).
type CoverOffer struct {
	// MaxLiability is at least the module's MinSum.
	MaxLiability int64
	// LoadingBP is the loading of a route policy, 0 to MaxLoadingBound.
	LoadingBP int64
	// StoreRateBP is the rate of a store policy, 0 to MaxStoreRateBound.
	StoreRateBP int64
	// CoverBP is the share of a loss value the offer pays, 1 to 10 000.
	CoverBP int64
}

// CoverView is the insurance group of the view of an agent whose role has a cover block
// (spec-insurance.md 10.1): plain values valid for the call only. It holds no other
// agent's policy, no offer, no loading, no count of underwriters and no figure of the
// market for cover. It is AgentView.Insurance: the name Cover of the view is the
// subsistence group's ticks of upkeep.
type CoverView struct {
	// Underwrite and Insure are a copy of the role's block; one of them is nil.
	Underwrite *UnderwriteTerms
	Insure     *InsureTerms
	// MinSum and MaxCoverRequests are the module's, and ValueCapBP its cap on a unit's
	// value, so a strategy need not know the configuration.
	MinSum, MaxCoverRequests, ValueCapBP int64
	// Open is, for an underwriter, the sum of the maximum payouts of its open policies.
	Open int64
	// Money is the agent's whole balance at the start of phase 3: no reservation exists
	// yet.
	Money int64
	// Capacity is, for an underwriter, max(0, CapBP * Money / 10 000 - Open), saturated at
	// MaxMoney.
	Capacity int64
	// Owed is the principal of payout claims the agent owes as debtor, 0 when none.
	Owed int64
	// Policies is, for an insured, the number of its open route policies.
	Policies int64
	// LaneLoss is, for an insured, the expected loss of each row of Routes, in the same
	// order: Routes[i].LossBP.
	LaneLoss []int64
	// UnitValue is, for an insured, the unit value of each commodity in its region
	// (spec-insurance.md 6.2.1), read at the start of phase 3, by commodity ID.
	UnitValue []int64
}

// coverScratch is the storage of the group of the view: the copies of the role's block
// and the group itself, reused by every agent (a view is valid for its call).
type coverScratch struct {
	view      CoverView
	uw        UnderwriteTerms
	ins       InsureTerms
	laneLoss  []int64
	unitValue []int64
}

// fillCoverView sets the insurance group of the view passed to Trade for an agent whose
// role has a cover block (hook H-8 of plan 3-ins). It is nil-safe, so the fan-in holds no
// branch on the module, and draws nothing.
func (w *World) fillCoverView(v *AgentView, a *agent) {
	in := w.insurance
	if in == nil {
		return
	}
	t := w.cfg.Roles[a.role].Cover
	if t == nil {
		return
	}
	s := &in.vs
	s.view = CoverView{
		MinSum: in.cfg.MinSum, MaxCoverRequests: in.cfg.MaxCoverRequests, ValueCapBP: in.cfg.ValueCapBP,
		Money: v.Money, Owed: w.payoutOwed(a.account),
	}
	if u := t.Underwrite; u != nil {
		s.uw, s.view.Underwrite = *u, &s.uw
		s.view.Open = in.liabilityOf(a.id)
		free, ok := insure.Free(u.CapBP, v.Money, s.view.Open, MaxMoney)
		if !ok {
			invariant("a negative figure in the capacity of an underwriter")
		}
		s.view.Capacity = free
	}
	if i := t.Insure; i != nil {
		s.ins, s.view.Insure = *i, &s.ins
		s.view.Policies = in.policiesOf(a.id)
		s.laneLoss = s.laneLoss[:0]
		for j := range v.Routes {
			s.laneLoss = append(s.laneLoss, v.Routes[j].LossBP)
		}
		s.view.LaneLoss = s.laneLoss
		s.unitValue = w.unitValues(w.regionOf(a), s.unitValue[:0], nil)
		s.view.UnitValue = s.unitValue
	}
	v.Insurance = &s.view
}

// publishedPrice is the price the market of a region published at the last clearing
// before the position that reads it, the reference price while the market has never
// traded (spec-insurance.md 6.2.1): the price left by P4.1 of the previous tick at P3.3c
// and P1.3a, by P4.1 of this tick at P8.1b. It reads the market report the welfare
// valuation reads, and allocates nothing.
func (w *World) publishedPrice(g RegionID, c CommodityID) Price {
	i := int(g)*len(w.cfg.Commodities) + int(c)
	if w.markets.windows[i].Len() > 0 {
		return w.markets.reports[i].Price
	}
	return w.refs[i]
}

// unitValue is the value of a unit of commodity c in region g: the published price capped
// at ValueCapBP of the effective reference price (spec-insurance.md 6.2.1). capped says
// the cap was the smaller, which CoverRound and CoverSettled count in units.
func (w *World) unitValue(g RegionID, c CommodityID) (v int64, capped bool) {
	pub := int64(w.publishedPrice(g, c))
	v, ok := insure.UnitValue(pub, int64(w.refPrice(g, c)), w.insurance.cfg.ValueCapBP)
	if !ok {
		invariant("a negative or overflowing unit value of a validated configuration")
	}
	return v, v < pub
}

// unitValues appends the unit value of every commodity of region g, by commodity ID, to
// dst. When capped is not nil it is set, by commodity, to whether the cap bound.
func (w *World) unitValues(g RegionID, dst []int64, capped []bool) []int64 {
	for c := range w.cfg.Commodities {
		v, cap := w.unitValue(g, CommodityID(c))
		dst = append(dst, v)
		if capped != nil {
			capped[c] = cap
		}
	}
	return dst
}

// checkCoverPlan is the plan check's part for the requests and the offer of agent a (the
// checkTradePlan fan-in, hook H-8): with the module off each is refused at once as
// insurance_off, a Debug line and no event, else it is kept for the round of P3.3c. It is
// nil-safe, so the fan-in holds no branch on the module.
func (w *World) checkCoverPlan(a *agent, p *TradePlan) {
	if len(p.Cover) == 0 && len(p.Store) == 0 && len(p.Offer) == 0 {
		return
	}
	in := w.insurance
	if in == nil {
		for n := len(p.Cover) + len(p.Store) + len(p.Offer); n > 0; n-- {
			w.logCoverRefused(a.id, ReasonInsuranceOff)
		}
		return
	}
	rd := &in.round
	seq := 0 // the number of the request in this plan, lanes then the store request
	for _, l := range p.Cover {
		rd.reqs = append(rd.reqs, coverRequest{agent: a.id, seq: seq, lane: l})
		seq++
	}
	for _, s := range p.Store {
		rd.reqs = append(rd.reqs, coverRequest{agent: a.id, seq: seq, store: true, sc: s})
		seq++
	}
	for _, o := range p.Offer {
		rd.offs = append(rd.offs, coverOffer{agent: a.id, o: o})
	}
}

// logCoverRefused is the Debug line "cover refused" of a request or an offer refused at
// admission (spec-insurance.md 15.2).
func (w *World) logCoverRefused(agent AgentID, reason Reason) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "cover refused", slog.Int64(logkeys.Agent, int64(agent)), slog.String(logkeys.Reason, string(reason)))
	}
}

// DefaultCover appends to the plan what the default strategy makes for the cover group of
// the view, as DefaultCredit does for credit: the requests of the default insured
// (spec-insurance.md 10.2) and then the offer of the default underwriter (10.3, see
// DefaultOffer), from the bids the plan holds. The requests are a route request for each lane on which the
// plan has a shipment or a delivery offer, when the lane's expected loss is at least
// MinLossBP and the value of the plan's goods on it is at least RuinShareBP of the agent's
// money, and a store request when StoreShareBP is above 0 and the agent owns a store. The
// rule is ruin-based and not price-based. It takes no random word and keeps no state, and
// a view with no cover group gets nothing. FR-INS-061, FR-INS-063.
func DefaultCover(v *AgentView, p *TradePlan) {
	cv := v.Insurance
	if cv == nil {
		return
	}
	if o, ok := DefaultOffer(v, planBids(p)); ok {
		p.Offer = append(p.Offer, o)
	}
	i := cv.Insure
	if i == nil {
		return
	}
	if i.MinLossBP > 0 {
		defaultLanes(v, p)
	}
	if i.StoreShareBP > 0 && ownsStore(v) {
		p.Store = append(p.Store, StoreCover{ShareBP: i.StoreShareBP, MinCoverBP: i.MinCoverBP, MaxStoreRateBP: i.MaxStoreRateBP})
	}
}

// ownsStore says the agent owns at least one store, from the room group of the view.
func ownsStore(v *AgentView) bool {
	for _, s := range v.Room.Stores {
		if s.Count > 0 {
			return true
		}
	}
	return false
}

// defaultLanes appends a request for each lane of the plan, in the order its first
// shipment or offer names it, that passes the two tests of 10.2.
func defaultLanes(v *AgentView, p *TradePlan) {
	i := v.Insurance.Insure
	for k := range p.Shipments {
		s := &p.Shipments[k]
		if firstOfLane(p, k, s.Mode, s.To) && laneInsurable(v, p, s.Mode, s.To) {
			p.Cover = append(p.Cover, LaneCover{Mode: s.Mode, To: s.To, MinCoverBP: i.MinCoverBP, MaxLoadingBP: i.MaxLoadingBP})
		}
	}
	for k := range p.Offers {
		o := &p.Offers[k]
		if firstOfLane(p, len(p.Shipments)+k, o.Mode, o.To) && laneInsurable(v, p, o.Mode, o.To) {
			p.Cover = append(p.Cover, LaneCover{Mode: o.Mode, To: o.To, MinCoverBP: i.MinCoverBP, MaxLoadingBP: i.MaxLoadingBP})
		}
	}
}

// laneAt returns the lane of the n-th item of the plan, the shipments then the delivery
// offers.
func laneAt(p *TradePlan, n int) (ModeID, RegionID) {
	if n < len(p.Shipments) {
		return p.Shipments[n].Mode, p.Shipments[n].To
	}
	o := &p.Offers[n-len(p.Shipments)]
	return o.Mode, o.To
}

// firstOfLane says no earlier item of the plan names the lane (mode, to).
func firstOfLane(p *TradePlan, n int, mode ModeID, to RegionID) bool {
	for k := range n {
		if m, t := laneAt(p, k); m == mode && t == to {
			return false
		}
	}
	return true
}

// laneInsurable is the two tests of spec-insurance.md 10.2 for a lane: its expected loss
// is at least MinLossBP and the plan's goods on it are at least RuinShareBP of the money.
func laneInsurable(v *AgentView, p *TradePlan, mode ModeID, to RegionID) bool {
	i := v.Insurance.Insure
	loss, ok := laneLoss(v, mode, to)
	if !ok || loss < i.MinLossBP {
		return false
	}
	return insure.AtRisk(laneValue(v, p, mode, to), i.RuinShareBP, v.Insurance.Money)
}

// laneLoss is the expected loss of the route (mode, to) from the agent's region.
func laneLoss(v *AgentView, mode ModeID, to RegionID) (int64, bool) {
	for k := range v.Routes {
		if r := &v.Routes[k]; r.Mode == mode && r.To == to {
			return r.LossBP, r.Has
		}
	}
	return 0, false
}

// laneValue is the value of the plan's goods on a lane at the unit values of the view:
// the lines of a shipment and the quantity of a delivery offer. It saturates at MaxMoney.
func laneValue(v *AgentView, p *TradePlan, mode ModeID, to RegionID) int64 {
	var s sum
	for k := range p.Shipments {
		if r := &p.Shipments[k]; r.Mode == mode && r.To == to {
			for _, ln := range r.Lines {
				s.add(checked.Mul(int64(ln.Quantity), unitAt(v, ln.Commodity)))
			}
		}
	}
	for k := range p.Offers {
		if o := &p.Offers[k]; o.Mode == mode && o.To == to {
			s.add(checked.Mul(int64(o.Quantity), unitAt(v, o.Commodity)))
		}
	}
	if s.bad {
		return MaxMoney
	}
	return s.v
}

// unitAt is the unit value of a commodity in the view, 0 for a commodity the view holds no
// value of (a hand-built view).
func unitAt(v *AgentView, c CommodityID) int64 {
	if c < 0 || int(c) >= len(v.Insurance.UnitValue) {
		return 0
	}
	return v.Insurance.UnitValue[c]
}

// DefaultOffer is the default underwriter of spec-insurance.md 10.3: a share of the room
// its own bids left, at its block's posted terms. bids is the sum of limit times quantity
// of the bids of the plan it has just built. It offers nothing while it owes a payout
// claim or when the offer is below MinSum. FR-INS-062, FR-INS-063.
func DefaultOffer(v *AgentView, bids int64) (CoverOffer, bool) {
	cv := v.Insurance
	if cv == nil || cv.Underwrite == nil || cv.Owed > 0 {
		return CoverOffer{}, false
	}
	u := cv.Underwrite
	avail, ok := checked.Sub(cv.Money, bids)
	if !ok {
		return CoverOffer{}, false // a hand-built view whose figures wrap: nothing is offered
	}
	free, ok := insure.Free(u.CapBP, max(0, avail), cv.Open, MaxMoney)
	if !ok {
		return CoverOffer{}, false
	}
	offer, ok := checked.MulDiv(u.ShareBP, free, wholeBP)
	if !ok || offer < cv.MinSum {
		return CoverOffer{}, false
	}
	return CoverOffer{MaxLiability: min(offer, MaxMoney), LoadingBP: u.LoadingBP, StoreRateBP: u.StoreRateBP, CoverBP: u.CoverBP}, true
}
