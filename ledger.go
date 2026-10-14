package economysim

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// constructionTick is the Tick of the BoundaryFlow rows of the construction
// bucket (spec-ledger.md 7.4.1, 12.1).
const constructionTick = -1

// numBuiltInReasons is the count of the built-in reasons of internal/ledger,
// which are numbered from 0 (spec-ledger.md 7.2.2). The declared reasons (the
// loan rows, appended with the credit module) follow them.
const numBuiltInReasons = int(ledger.LoanOut)

// reasonName returns the root's Reason of a ledger reason, and false for a
// reason the root has no name for. The number is internal and never saved raw
// (a checkpoint saves the name), so this switch is the one place the two meet.
// A module that declares a reason adds its case with its own change
// (architecture.md 2.10.5).
func reasonName(r ledger.Reason) (Reason, bool) {
	switch r {
	case ledger.Trade:
		return ReasonTrade, true
	case ledger.Initial:
		return ReasonInitial, true
	case ledger.RecipeInput:
		return ReasonRecipeInput, true
	case ledger.RecipeOutput:
		return ReasonRecipeOutput, true
	case ledger.Disposal:
		return ReasonDisposal, true
	case ledger.Consumption:
		return ReasonConsumption, true
	case ledger.Spoilage:
		return ReasonSpoilage, true
	case ledger.Wear:
		return ReasonWear, true
	case ledger.EstateMoney:
		return ReasonEstateMoney, true
	case ledger.EstateGoods:
		return ReasonEstateGoods, true
	case ledger.StartMoney:
		return ReasonStartMoney, true
	case ledger.Split:
		return ReasonSplit, true
	case ledger.Merge:
		return ReasonMerge, true
	case ledger.Births:
		return ReasonBirths, true
	case ledger.Starvation:
		return ReasonStarvation, true
	}
	return "", false
}

// ledgerReason returns the ledger's reason of a root Reason, and false for a
// Reason that is not a ledger reason (a rejection reason, for example). The
// table has fewer than twenty rows, so a scan is simpler than a map and keeps
// the order fixed (architecture.md 7.5).
func ledgerReason(r Reason) (ledger.Reason, bool) {
	for i := range numBuiltInReasons {
		if name, _ := reasonName(ledger.Reason(i)); name == r {
			return ledger.Reason(i), true
		}
	}
	return 0, false
}

// newLedger builds the ledger of a configuration in canonical order
// (Config.canonical): the commodity table from ShelfLife and Durability in
// commodity-ID order, the bounds of bounds.go, and the boundary accounts of the
// built-in group in the creation order of spec-ledger.md 3.4.2, so that the
// first account of an estate or an agent has the ID that section counts. It
// opens no other account. The claims limit is 0 and no reason is declared:
// the credit module sets both when it is on (plan-phase-1-1.md A1).
//
// The order is money first, then commodity-major: for each commodity, the
// goods reasons of the built-in group that are a faucet or a sink, in the
// registry order. It reads the classes from the ledger's table, so no reason
// or commodity is named here (architecture.md 9.1).
func newLedger(cfg Config) (*ledger.Ledger, error) {
	comms := make([]ledger.Commodity, len(cfg.Commodities))
	for i, c := range cfg.Commodities {
		comms[i] = ledger.Commodity{ShelfLife: c.ShelfLife, Durability: c.Durability}
	}
	l, err := ledger.New(ledger.Config{
		Commodities: comms,
		Regions:     len(cfg.Regions),
		MaxMoney:    MaxMoney,
		MaxQuantity: MaxQuantity,
	})
	if err != nil {
		return nil, fmt.Errorf("build ledger: %w", err)
	}
	open := func(r ledger.Reason, thing int) error {
		if _, err := l.OpenBoundary(r, thing); err != nil {
			return fmt.Errorf("open boundary account %d of thing %d: %w", r, thing, err)
		}
		return nil
	}
	for r := ledger.Reason(0); int(r) < numBuiltInReasons; r++ {
		if class, ok := ledger.ClassOf(r, false); ok && class != ledger.Internal {
			if err := open(r, ledger.Money); err != nil {
				return nil, err
			}
		}
	}
	for c := range comms {
		for r := ledger.Reason(0); int(r) < numBuiltInReasons; r++ {
			if class, ok := ledger.ClassOf(r, true); ok && class != ledger.Internal {
				if err := open(r, c); err != nil {
					return nil, err
				}
			}
		}
	}
	return l, nil
}

// boundaryFlows appends to dst the BoundaryFlow rows for flows, which are the
// totals of one tick or the construction bucket as Ledger.AppendFlows returns
// them, and returns the extended slice. tick is the tick of the rows, or
// constructionTick for the bucket. Only faucet and sink reasons are reported,
// and a zero total is not in flows. The rows follow the order of flows:
// ascending region, then reason in the order of spec-ledger.md 7.2, money
// before goods, then ascending commodity (spec-observability.md 4.4.1.2).
//
// It allocates only when dst has no room, so the world reuses one slice. A
// faucet or a sink with no root name is a programmer error: its module
// declared the reason and did not add its case to reasonName.
func boundaryFlows(dst []BoundaryFlow, flows []ledger.Flow, tick int64) []BoundaryFlow {
	for _, f := range flows {
		goods := f.Commodity != ledger.Money
		class, ok := ledger.ClassOf(f.Reason, goods)
		if !ok || class == ledger.Internal {
			continue
		}
		name, ok := reasonName(f.Reason)
		if !ok {
			panic(fmt.Sprintf("economysim: invariant violated (programmer error): boundary flow of ledger reason %d has no Reason name", f.Reason))
		}
		kind, commodity := KindFaucet, NoCommodity
		if class == ledger.Sink {
			kind = KindSink
		}
		if goods {
			commodity = CommodityID(f.Commodity)
		}
		dst = append(dst, BoundaryFlow{
			Tick: tick, Region: RegionID(f.Region), Reason: name,
			Kind: kind, Commodity: commodity, Total: f.Total,
		})
	}
	return dst
}

// wearApplied appends to dst the WearApplied rows of l: for each of the
// commodities commodity indices, in ascending order, one row with the wear
// units of the tick, or with bucket true those of the construction phase, when
// they are above 0. tick is the tick of the rows, or constructionTick for the
// bucket (spec-ledger.md 12.1.1). The wear units are per commodity and have no
// region (spec-ledger.md 7.4).
//
// It allocates only when dst has no room, so the world reuses one slice. A
// commodity index the ledger does not know is a programmer error: commodities
// is the length of the table the ledger was built from.
func wearApplied(dst []WearApplied, l *ledger.Ledger, commodities int, tick int64, bucket bool) []WearApplied {
	for c := 0; c < commodities; c++ {
		units, err := l.WearUnits(c, bucket)
		if err != nil {
			panic(fmt.Sprintf("economysim: invariant violated (programmer error): wear units of commodity %d: %v", c, err))
		}
		if units > 0 {
			dst = append(dst, WearApplied{Tick: tick, Commodity: CommodityID(c), Units: units})
		}
	}
	return dst
}
