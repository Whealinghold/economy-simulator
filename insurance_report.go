package economysim

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/insure"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the reporting of the insurance module (spec-insurance.md 15, 16; the
// continuation of insurance.go, architecture.md 2.6): the events of the cover round, their
// Debug lines and the counts of the construction summary, the snapshot rows and the group
// of AgentState, and the codec of tag 29 of the checkpoint. Events are constructed only when
// an observer was supplied (observability-and-logging.md 4.5). There is no Warn in the tick:
// a policy and a refusal are per entity, and an underwriter's arrears are CRD's.

// reportCoverFormed reports a policy formed (spec-insurance.md 7.4 step 5, 15.1, 15.2): the
// Debug line "cover formed" and, with an observer, CoverFormed in formation order. The
// request names the lane of a route policy.
func (w *World) reportCoverFormed(p *coverPolicy, rate int64, rq *admittedRequest, underRegion RegionID) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "cover formed", slog.Int64(logkeys.CoverPolicy, p.id),
			slog.Int64(logkeys.Agent, int64(p.insured)), slog.Int64(logkeys.Region, int64(rq.region)),
			slog.Int64(logkeys.Sum, p.sum), slog.Int64(logkeys.Premium, p.premium))
	}
	if !w.obs.supplied {
		return
	}
	ev := CoverFormed{
		Tick: p.formed, Policy: p.id, Kind: p.kind, Insured: p.insured, Underwriter: p.underwriter,
		Region: rq.region, UnderwriterRegion: underRegion, Shipment: p.shipment, Mode: NoMode, To: NoRegion,
		Sum: Money(p.sum), MaxPayout: Money(p.maxPayout), Premium: Money(p.premium), RateBP: rate, CoverBP: p.coverBP,
	}
	if p.kind == CoverRoute {
		ev.Mode, ev.To = rq.req.lane.Mode, rq.req.lane.To
	}
	w.obs.deliver(ev)
}

// reportCoverRound delivers CoverRound for each region, in ascending RegionID, that has a
// count other than 0, after the last CoverFormed (spec-insurance.md 15.1).
func (w *World) reportCoverRound() {
	if !w.obs.supplied {
		return
	}
	t := w.Tick()
	for g := range w.insurance.round.counts {
		c := &w.insurance.round.counts[g]
		if c.quiet() {
			continue
		}
		w.obs.deliver(CoverRound{
			Tick: t, Region: RegionID(g), Requests: c.requests, Offers: c.offers,
			RequestsRefused: c.requestsRefused, OffersRefused: c.offersRefused, InArrears: c.inArrears, Exposure: c.exposure,
			Candidates: c.candidates, Formed: c.formed, Sum: Money(c.sum), Premium: Money(c.premium), Unfilled: Money(c.unfilled),
			NoShipment: c.noShipment, NoRisk: c.noRisk, NoOffer: c.noOffer, BelowMinimum: c.belowMinimum, Funds: c.funds,
			TooMany: c.tooMany, Overflow: c.overflow, ValueCapped: c.valueCapped,
		})
	}
}

// coverAgents counts the live agents whose role has each block, for the construction
// summary (spec-insurance.md 15.2).
func (w *World) coverAgents() (underwriters, insured int64) {
	for i := w.firstHousehold; i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 {
			continue
		}
		if t := w.cfg.Roles[a.role].Cover; t != nil {
			if t.Underwrite != nil {
				underwriters++
			}
			if t.Insure != nil {
				insured++
			}
		}
	}
	return underwriters, insured
}

// insuranceSummary appends the counts of the module to the construction summary.
func (w *World) insuranceSummary(attrs []slog.Attr) []slog.Attr {
	if w.insurance == nil {
		return attrs
	}
	u, i := w.coverAgents()
	return append(attrs, slog.Int64(logkeys.Underwriters, u), slog.Int64(logkeys.Insured, i))
}

// reportCoverEnded reports a policy ended (spec-insurance.md 15.1, 15.2): the Debug line
// "cover ended" and, with an observer, CoverEnded in ascending policy ID. quiet says the
// event is not emitted: a store policy that ended with no loss is only counted (15.1.1).
func (w *World) reportCoverEnded(it *coverItem, quiet bool) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "cover ended", slog.Int64(logkeys.CoverPolicy, it.p.id),
			slog.Int64(logkeys.Agent, int64(it.p.insured)), slog.String(logkeys.Cause, string(it.cause)),
			slog.Int64(logkeys.Payout, it.paid))
	}
	if quiet || !w.obs.supplied {
		return
	}
	w.obs.deliver(CoverEnded{
		Tick: w.Tick(), Policy: it.p.id, Kind: it.p.kind, Insured: it.p.insured, Underwriter: it.p.underwriter,
		Region: it.region, UnderwriterRegion: it.underRegion, Cause: it.cause,
		LossValue: Money(it.lossValue), Due: Money(it.due), Paid: Money(it.paid), Short: Money(it.short), Claim: ClaimID(it.claim),
		Dropped: Money(it.dropped), Cut: Money(it.cut), Capped: it.capped,
	})
}

// reportCoverSettled delivers CoverSettled of a position for each region, in ascending
// RegionID, that has a count or a sum, after the CoverEnded events of the position
// (spec-insurance.md 15.1).
func (w *World) reportCoverSettled(stage CoverStage) {
	if !w.obs.supplied {
		return
	}
	t := w.Tick()
	for g, r := range w.insurance.st.rows {
		if r.quiet() {
			continue
		}
		w.obs.deliver(CoverSettled{
			Tick: t, Region: RegionID(g), Stage: stage, Ended: r.ended, Claimed: r.claimed,
			LossValue: Money(r.lossValue), Due: Money(r.due), Paid: Money(r.paid), Short: Money(r.short),
			Dropped: Money(r.dropped), Cut: Money(r.cut), Capped: r.capped, ValueCapped: r.valueCapped, Shielded: Money(r.shielded),
		})
	}
}

// CoverPolicyRow is the snapshot row of an open route policy, by policy ID (spec-insurance.md
// 15.4): what was fixed at formation. Region is the insured's region at the boundary. A store
// policy opens and ends inside one tick, so none is a row.
type CoverPolicyRow struct {
	Policy               int64
	Insured, Underwriter AgentID
	Region               RegionID
	Shipment             ShipmentID
	Sum, MaxPayout       Money
	CoverBP              int64
	Premium              Money
	Formed               int64
}

// InsuranceTotals are the cumulative totals of the insurance module in the snapshot
// (spec-insurance.md 9.3, 15.4): the policies formed, the premiums paid, the sum due, the part
// paid at once, the shortfall formed as payout claims (the value of Credit.Awarded, read from
// the ledger and not saved twice), the part dropped and the part cut at a headroom. It is the
// zero value with the module off.
type InsuranceTotals struct {
	Formed                     int64
	Premiums, Due, Paid, Short Money
	Dropped, Cut               Money
}

// CoverState is the insurance group of AgentState (spec-insurance.md 16.3.1, a read at a
// boundary, not the strategy's view): Policies is the number of the agent's open route
// policies, as the insured or as an underwriter, and Open the sum of the maximum payouts of
// the policies it underwrites, its open liability. Both are 0 with the module off. The field
// is AgentState.Insurance, since AgentState.Cover is the subsistence ticks of upkeep.
type CoverState struct {
	Policies int64
	Open     Money
}

// snapshotInsurance sets the Policies and Insurance of a snapshot: the open route policies by
// ID and the totals, with the shortfall read from the ledger. It does nothing with the module off.
func (w *World) snapshotInsurance(s *Snapshot) {
	in := w.insurance
	if in == nil {
		return
	}
	s.Policies = make([]CoverPolicyRow, len(in.open))
	for i := range in.open {
		p := &in.open[i]
		s.Policies[i] = CoverPolicyRow{
			Policy: p.id, Insured: p.insured, Underwriter: p.underwriter, Region: w.regionOf(&w.agents[p.insured]),
			Shipment: p.shipment, Sum: Money(p.sum), MaxPayout: Money(p.maxPayout), CoverBP: p.coverBP,
			Premium: Money(p.premium), Formed: p.formed,
		}
	}
	t := in.tot
	s.Insurance = InsuranceTotals{
		Formed: t.formed, Premiums: Money(t.premiums), Due: Money(t.due), Paid: Money(t.paid),
		Short: Money(w.ledger.ClaimTotals().Awarded), Dropped: Money(t.dropped), Cut: Money(t.cut),
	}
}

// fillInsuranceState sets the insurance group of the state of agent a (spec-insurance.md
// 16.3.1): the count of the open route policies it holds as the insured or underwrites, and
// its open liability. It reads the lists and sets nothing else.
func (w *World) fillInsuranceState(st *AgentState, a *agent) {
	in := w.insurance
	if in == nil {
		return
	}
	n := in.policiesOf(a.id)
	for i := range in.open {
		if in.open[i].underwriter == a.id {
			n++
		}
	}
	st.Insurance = CoverState{Policies: n, Open: Money(in.liabilityOf(a.id))}
}

// saveInsurance is tag 29 (spec-checkpoint.md 4.2, spec-insurance.md 16.1): the next policy
// ID, the open route policies in ascending ID, each as (ID, insured, underwriter, shipment,
// Sum, MaxPayout, CoverBP, Premium, Formed), then the six totals of 9.3. A store policy is
// formed and ended inside one tick and the shortfall is the ledger's Awarded, so neither is
// saved; the indexes, the open liability and the scratch are derived.
func (w *World) saveInsurance(cw *codec.Writer) error {
	in := w.insurance
	cw.U(unsigned(in.next))
	cw.Count(len(in.open))
	for i := range in.open {
		p := &in.open[i]
		for _, v := range [...]int64{p.id, int64(p.insured), int64(p.underwriter), int64(p.shipment), p.sum, p.maxPayout, p.coverBP, p.premium, p.formed} {
			cw.U(unsigned(v))
		}
	}
	t := in.tot
	for _, v := range [...]int64{t.formed, t.premiums, t.due, t.paid, t.dropped, t.cut} {
		cw.U(unsigned(v))
	}
	return nil
}

// loadInsurance reads tag 29 and rebuilds the indexes (spec-insurance.md 16.3, 16.5): the
// policies strictly ascending and below the next ID, each in range and passing policyFault,
// at most MaxPolicies of them, and the totals within 0 to MaxMoney with Formed equal to the
// next ID. The identity of the due over the ledger's Awarded is checked by the audit that
// follows the restore. Any other record is ErrCheckpointCorrupt naming the policy.
func (w *World) loadInsurance(rc *restoreCtx, r *codec.Reader) error {
	in := w.insurance
	in.next = r.Bounded("next policy ID", MaxMoney)
	n := r.Count(9)
	if int64(n) > in.cfg.MaxPolicies {
		r.Failf("%d open policies, above MaxPolicies %d", n, in.cfg.MaxPolicies)
	}
	for i := 0; i < n && r.Err() == nil; i++ {
		p := w.readPolicy(rc, r)
		if r.Err() == nil {
			w.acceptPolicy(r, p, i > 0 && p.id <= in.open[i-1].id)
		}
	}
	in.readTotals(r)
	return nil
}

// acceptPolicy checks one policy read at position i and adds it, or fails the reader: it must
// be above the one before it (notAbove) and below the next ID, cover a shipment no other
// policy covers, pass policyFault and keep its underwriter's open liability within MaxMoney.
func (w *World) acceptPolicy(r *codec.Reader, p coverPolicy, notAbove bool) {
	in := w.insurance
	if notAbove || p.id >= in.next {
		r.Failf("policy %d is not above the one before it or not below the next ID %d", p.id, in.next)
		return
	}
	if in.covered(p.shipment) {
		r.Failf("policy %d covers shipment %d, which another policy covers", p.id, p.shipment)
		return
	}
	if why := w.policyFault(&p); why != "" {
		r.Failf("policy %d %s", p.id, why)
		return
	}
	if sum, ok := checked.Add(in.liabilityOf(p.underwriter), p.maxPayout); !ok || sum > MaxMoney {
		r.Failf("policy %d takes the open liability of underwriter %d above MaxMoney", p.id, p.underwriter)
		return
	}
	in.add(p)
}

// readPolicy reads one policy record with the range of each field checked: the agent IDs and
// the shipment ID below the next issued, the sums within 1 to MaxMoney, the share within 1 to
// 10 000, the premium within 0 to MaxMoney and the tick of formation not after the checkpoint.
func (w *World) readPolicy(rc *restoreCtx, r *codec.Reader) coverPolicy {
	p := coverPolicy{kind: CoverRoute}
	p.id = r.Bounded("policy ID", MaxMoney)
	p.insured, p.underwriter = AgentID(r.Index(uint64(w.state.nextAgent))), AgentID(r.Index(uint64(w.state.nextAgent)))
	ships := uint64(0)
	if w.transport != nil {
		ships = uint64(w.transport.next)
	}
	p.shipment = ShipmentID(r.Index(ships))
	p.sum, p.maxPayout = r.Bounded("sum insured", MaxMoney), r.Bounded("maximum payout", MaxMoney)
	p.coverBP, p.premium = r.Bounded("cover share", wholeBP), r.Bounded("premium", MaxMoney)
	p.formed = r.Bounded("tick of formation", rc.tick)
	if r.Err() == nil && (p.sum < 1 || p.coverBP < 1) {
		r.Failf("policy %d has a sum %d and a cover share %d, each at least 1", p.id, p.sum, p.coverBP)
	}
	return p
}

// readTotals reads the six totals of 9.3, each within 0 to MaxMoney, and checks that the
// policies formed are the next ID, the premiums at least those of the open policies and the
// formed at least the open ones.
func (in *insurance) readTotals(r *codec.Reader) {
	for _, v := range [...]*int64{&in.tot.formed, &in.tot.premiums, &in.tot.due, &in.tot.paid, &in.tot.dropped, &in.tot.cut} {
		*v = r.Bounded("insurance total", MaxMoney)
	}
	if r.Err() == nil && in.tot.formed != in.next {
		r.Failf("%d policies formed, want the next ID %d", in.tot.formed, in.next)
	}
	if r.Err() == nil {
		if err := in.checkOpenTotals(); err != nil {
			r.Failf("%v", err)
		}
	}
}

// policyFault says what is wrong with an open route policy, or "" when it is sound
// (spec-insurance.md 9.4.4, 16.5): its insured and underwriter are distinct live agents with
// open Agent accounts, its shipment is in flight with the insured as its shipper, and its
// maximum payout is halfEven(CoverBP * Sum / 10 000).
func (w *World) policyFault(p *coverPolicy) string {
	if mp, ok := insure.MaxPayout(p.coverBP, p.sum); !ok || mp != p.maxPayout {
		return fmt.Sprintf("has maximum payout %d, want halfEven(%d * %d / 10000) = %d", p.maxPayout, p.coverBP, p.sum, mp)
	}
	if p.insured == p.underwriter {
		return fmt.Sprintf("has agent %d as its insured and its underwriter", p.insured)
	}
	for _, id := range [...]AgentID{p.insured, p.underwriter} {
		if !w.openAgentAccount(id) {
			return fmt.Sprintf("names agent %d, which is no live agent with an open Agent account", id)
		}
	}
	s, ok := w.shipmentInFlight(p.shipment)
	switch {
	case !ok:
		return fmt.Sprintf("covers shipment %d, which is not in flight", p.shipment)
	case s.shipper != p.insured:
		return fmt.Sprintf("covers shipment %d, whose shipper is agent %d, not its insured %d", p.shipment, s.shipper, p.insured)
	}
	return ""
}
