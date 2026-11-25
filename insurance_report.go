package economysim

import (
	"context"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the reporting of the insurance module (spec-insurance.md 15; the
// continuation of insurance.go, architecture.md 2.6): the events of the cover round, their
// Debug lines and the counts of the construction summary. Events are constructed only when
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
