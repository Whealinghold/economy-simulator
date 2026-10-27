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
// (a checkpoint saves the name), so this table is the one place the two meet.
// A module that declares a reason adds its row with its own change
// (architecture.md 2.10.5).
func reasonName(r ledger.Reason) (Reason, bool) {
	if int(r) >= len(reasonNames) || reasonNames[r] == "" {
		return "", false
	}
	return reasonNames[r], true
}

// reasonNames is the table reasonName reads, never mutated. The declared loan
// reasons have no row: they are not built in.
var reasonNames = [...]Reason{
	ledger.Trade:        ReasonTrade,
	ledger.Initial:      ReasonInitial,
	ledger.RecipeInput:  ReasonRecipeInput,
	ledger.RecipeOutput: ReasonRecipeOutput,
	ledger.Disposal:     ReasonDisposal,
	ledger.Consumption:  ReasonConsumption,
	ledger.Spoilage:     ReasonSpoilage,
	ledger.Wear:         ReasonWear,
	ledger.EstateMoney:  ReasonEstateMoney,
	ledger.EstateGoods:  ReasonEstateGoods,
	ledger.StartMoney:   ReasonStartMoney,
	ledger.Split:        ReasonSplit,
	ledger.Merge:        ReasonMerge,
	ledger.Births:       ReasonBirths,
	ledger.Starvation:   ReasonStarvation,
	// The declared module rows with a name: the loan rows have none yet.
	ledger.WantConsumption: ReasonWantConsumption,
	ledger.Relief:          ReasonRelief,
}

// ledgerReason returns the ledger's reason of a root Reason, and false for a
// Reason that is not a ledger reason (a rejection reason, for example). The
// table has fewer than twenty rows, so a scan is simpler than a map and keeps
// the order fixed (architecture.md 7.5).
func ledgerReason(r Reason) (ledger.Reason, bool) {
	for i := range reasonNames {
		if reasonNames[i] != "" && reasonNames[i] == r {
			return ledger.Reason(i), true
		}
	}
	return 0, false
}

// newLedger builds the ledger of a configuration in canonical order
// (Config.canonical): the commodity table from ShelfLife and Durability in
// commodity-ID order, the bounds of bounds.go, the reasons the modules that are
// on declare (moduleReasons), and the boundary accounts in the four groups of
// openBoundaries (spec-ledger.md 3.4.2), so that the first account of an estate
// or an agent has the ID that section counts. It opens no other account. The
// claims limit is 0: the credit module sets it when it is on
// (plan-phase-1-1.md A1).
func newLedger(cfg Config) (*ledger.Ledger, error) {
	comms := make([]ledger.Commodity, len(cfg.Commodities))
	for i, c := range cfg.Commodities {
		comms[i] = ledger.Commodity{ShelfLife: c.ShelfLife, Durability: c.Durability}
	}
	modules := moduleReasons(&cfg)
	var declared []ledger.Reason
	for _, m := range modules {
		declared = append(declared, m...)
	}
	classes := flowClasses(&cfg)
	l, err := ledger.New(ledger.Config{
		Commodities: comms,
		Regions:     len(cfg.Regions),
		MaxMoney:    MaxMoney,
		MaxQuantity: MaxQuantity,
		Declared:    declared,
		Flows:       classes,
	})
	if err != nil {
		return nil, fmt.Errorf("build ledger: %w", err)
	}
	for _, g := range boundaryGroups(len(comms), modules, len(classes)) {
		if err := g.open(l); err != nil {
			return nil, fmt.Errorf("open the %s boundary accounts: %w", g.name, err)
		}
	}
	return l, nil
}

// moduleReasons returns the ledger reasons each module that is on declares, one
// list per module in the order of the Modules pointers (spec-config.md 3.1). A
// module's change appends its own case (architecture.md 2.10.5).
func moduleReasons(c *Config) [][]ledger.Reason {
	var m [][]ledger.Reason
	if c.Modules.Demand != nil {
		m = append(m, demandReasons())
	}
	if c.Modules.Estates != nil {
		m = append(m, estatesReasons())
	}
	return m
}

// boundaryGroup is one of the four groups of boundary accounts of
// spec-ledger.md 3.4.2: its name, for an error, and the function that opens it.
type boundaryGroup struct {
	name string
	open func(*ledger.Ledger) error
}

// boundaryGroups returns the four groups in the order the accounts are created:
// (1) the built-in group; (2) the declared money-flow reasons, one money
// boundary for each of the flows; (3) the six outside_* reasons; (4) the other
// reasons of the modules that are on, in module order. Group 3 is an empty hook
// until plan 16 fills it; each module's change fills its own group and no other.
func boundaryGroups(commodities int, modules [][]ledger.Reason, flows int) [4]boundaryGroup {
	return [4]boundaryGroup{
		{"built-in", func(l *ledger.Ledger) error { return openBuiltInBoundaries(l, commodities) }},
		{"money-flow", func(l *ledger.Ledger) error { return openFlowBoundaries(l, flows) }},
		{"outside", func(*ledger.Ledger) error { return nil }},
		{"module", func(l *ledger.Ledger) error { return openModuleBoundaries(l, commodities, modules) }},
	}
}

// openBuiltInBoundaries opens the boundary accounts of the built-in group: money
// first, then commodity-major, the reasons in registry order. It reads the
// classes from the ledger's table, so no reason or commodity is named here
// (architecture.md 9.1).
func openBuiltInBoundaries(l *ledger.Ledger, commodities int) error {
	for thing := ledger.Money; thing < commodities; thing++ {
		for r := ledger.Reason(0); int(r) < numBuiltInReasons; r++ {
			if err := openBoundary(l, r, thing); err != nil {
				return err
			}
		}
	}
	return nil
}

// openFlowBoundaries opens the money boundary account of each of the n money
// flows, in flow-ID order (spec-money-supply.md 4.5.1, plan 2-6 D324).
func openFlowBoundaries(l *ledger.Ledger, n int) error {
	for i := range n {
		if _, err := l.OpenBoundary(ledger.FlowReason(i), ledger.Money); err != nil {
			return fmt.Errorf("open the boundary account of money flow %d: %w", i, err)
		}
	}
	return nil
}

// openModuleBoundaries opens the boundary accounts of group 4: module by module,
// and within a module as the built-in group is ordered (money first, then
// commodity-major, the module's reasons in the order it lists them).
func openModuleBoundaries(l *ledger.Ledger, commodities int, modules [][]ledger.Reason) error {
	for _, reasons := range modules {
		for thing := ledger.Money; thing < commodities; thing++ {
			for _, r := range reasons {
				if err := openBoundary(l, r, thing); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// openBoundary opens the boundary account of reason r for a thing (Money or a
// commodity index) when the reason is a faucet or a sink for it, and nothing
// otherwise.
func openBoundary(l *ledger.Ledger, r ledger.Reason, thing int) error {
	class, ok := ledger.ClassOf(r, thing != ledger.Money)
	if !ok || class == ledger.Internal {
		return nil
	}
	if _, err := l.OpenBoundary(r, thing); err != nil {
		return fmt.Errorf("open boundary account %d of thing %d: %w", r, thing, err)
	}
	return nil
}

// boundaryFlows appends to dst the BoundaryFlow rows for flows, which are the
// totals of one tick or the construction bucket as Ledger.AppendFlows returns
// them, and returns the extended slice. tick is the tick of the rows, or
// constructionTick for the bucket. Only faucet and sink reasons are reported,
// and a zero total is not in flows. The rows are in ascending region, then in
// the group order of spec-observability.md 4.4.1.2: the built-in reasons in the
// order of flows (the order of spec-ledger.md 7.2), then, where the money flows
// and the outside_* reasons will go, the declared module rows by the rank of
// their registry row (not by the number of the reason), each with money before
// goods and then ascending commodity.
//
// mf is the money flows of the configuration in ID order (nil with the module
// off): the reasons NumReasons+i of the ledger are their flows, named by the ID
// and classed by the kind (spec-money-supply.md 4.5.1).
//
// It allocates only when dst has no room, so the world reuses one slice. A
// faucet or a sink with no root name is a programmer error: its module
// declared the reason and did not add its case to reasonName.
func boundaryFlows(dst []BoundaryFlow, flows []ledger.Flow, tick int64, mf []MoneyFlow) []BoundaryFlow {
	var scratch [ledger.NumReasons]ledger.Reason
	module := ledger.DeclaredByRank(scratch[:0])
	for lo := 0; lo < len(flows); {
		hi := lo + 1
		for hi < len(flows) && flows[hi].Region == flows[lo].Region {
			hi++
		}
		dst = appendGroup(dst, flows[lo:hi], tick, 0, false, mf)
		for _, r := range module {
			dst = appendGroup(dst, flows[lo:hi], tick, r, true, mf)
		}
		lo = hi
	}
	return dst
}

// appendGroup appends the rows of the faucet and sink flows of one region that
// belong to a group: the flows of reason only when declared is true, else the
// flows of every reason that is not declared.
func appendGroup(dst []BoundaryFlow, flows []ledger.Flow, tick int64, only ledger.Reason, declared bool, mf []MoneyFlow) []BoundaryFlow {
	for _, f := range flows {
		if _, d := ledger.Rank(f.Reason); d != declared || (declared && f.Reason != only) {
			continue
		}
		goods := f.Commodity != ledger.Money
		class, name, ok := classAndName(f.Reason, goods, mf)
		if !ok {
			continue
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

// classAndName returns the class and the root name of a faucet or a sink reason
// of the ledger for money or goods, and false for a reason that is neither (an
// internal one, or one with no row for the thing). A flow reason is classed by
// the kind of its flow and named by its ID; a built-in or declared one by the
// ledger's table and reasonName.
func classAndName(r ledger.Reason, goods bool, mf []MoneyFlow) (ledger.Class, Reason, bool) {
	if r >= ledger.NumReasons {
		i := int(r - ledger.NumReasons)
		if goods || i >= len(mf) {
			panic(fmt.Sprintf("economysim: invariant violated (programmer error): boundary flow of ledger reason %d has no Reason name", r))
		}
		return flowClass(mf[i]), Reason(mf[i].ID), true
	}
	class, ok := ledger.ClassOf(r, goods)
	if !ok || class == ledger.Internal {
		return 0, "", false
	}
	name, ok := reasonName(r)
	if !ok {
		panic(fmt.Sprintf("economysim: invariant violated (programmer error): boundary flow of ledger reason %d has no Reason name", r))
	}
	return class, name, true
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
