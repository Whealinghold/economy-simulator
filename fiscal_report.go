package economysim

import (
	"context"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the reporting of the fiscal module (spec-fiscal.md 16; a
// continuation file of fiscal.go, architecture.md 2.6): the counters of the
// aggregate event FiscalFlowApplied, the event and its Debug line. An event is built
// only when an observer was supplied (observability-and-logging.md 4.5).

// count adds payer k of the flow at position pos to the counters of its region:
// what it was due and paid, and whether it paid less than it was due or was cut at
// the treasury's headroom (spec-fiscal.md 16.1.1).
func (s *fiscalState) count(pos, k int) {
	p := s.payers[k]
	g := &s.agg[pos*s.regions+int(p.region)]
	g.due, g.applied = satAdd(g.due, s.dues[k]), satAdd(g.applied, s.applied[k])
	if s.dues[k] > 0 {
		g.parties = satAdd(g.parties, p.weight)
	}
	if s.take[k] < s.dues[k] {
		g.short = satAdd(g.short, p.weight)
	}
	if s.applied[k] < s.take[k] {
		g.headrm = satAdd(g.headrm, p.weight)
	}
}

// reportFiscal emits, per flow in the order it ran and per region ascending, the
// FiscalFlowApplied of each region where a due was computed; never when nothing was
// due and never for a flow at scale 0, which did not run (FR-FIS-044).
func (w *World) reportFiscal(s *fiscalState, tick int64) {
	for _, pos := range s.ran {
		f := &s.flows[pos]
		for g := range s.regions {
			if a := s.agg[pos*s.regions+g]; a.due != 0 {
				w.flowApplied(f, tick, RegionID(g), a)
			}
		}
	}
}

// flowApplied reports one flow in one region.
func (w *World) flowApplied(f *fiscalFlowState, tick int64, region RegionID, a fiscalAgg) {
	commodity := NoCommodity
	if f.resource >= 0 {
		commodity = CommodityID(f.resource)
	}
	if w.obs.supplied {
		w.obs.deliver(FiscalFlowApplied{Tick: tick, Flow: f.id, Kind: f.kind.event(), Authority: f.authority, Region: region,
			Class: FiscalClass(f.target.kind), Commodity: commodity, Due: Money(a.due), Applied: Money(a.applied),
			Parties: a.parties, Short: a.short, Cut: a.cut, Headroom: a.headrm})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "fiscal flow applied", slog.String(logkeys.Flow, f.id),
			slog.Int64(logkeys.Region, int64(region)), slog.Int64(logkeys.Commodity, int64(commodity)),
			slog.Int64(logkeys.Due, a.due), slog.Int64(logkeys.Applied, a.applied),
			slog.Int64(logkeys.Cut, a.cut), slog.Int64(logkeys.Headroom, a.headrm))
	}
}

// event is the Kind of a flow kind in the event.
func (k fiscalKind) event() Kind {
	switch k {
	case fkDuty:
		return KindDuty
	case fkToll:
		return KindToll
	case fkPayout:
		return KindPayout
	}
	return KindTax
}
