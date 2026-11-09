package indicator

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// TradeLine is the trade with the outside in one commodity at one gate region:
// the units imported (Offer less Unsold) and exported (Purch) and their value at
// the price table (spec-indicators.md 7.1.1, 9.1 row 13).
type TradeLine struct {
	Gate      economysim.RegionID
	Commodity economysim.CommodityID
	Imp, Exp  int64
	// ImpValue and ExpValue are the units at the table of the call.
	ImpValue, ExpValue int64
}

// TradeGate is the money of the trade with the outside at one gate region:
// ImportValue is what the outside received for what it sold (outside_receipts),
// ExportValue what it paid for what it bought (outside_budget less
// outside_unspent), and Net their difference (exports less imports). The boundary
// flows of the outside are gross; the net is derived (N-115).
type TradeGate struct {
	Gate                          economysim.RegionID
	ImportValue, ExportValue, Net int64
	Binding                       BindingCounts
}

// TradeBalanceResult is the trade with the outside by gate and commodity, gross
// and net, in units and in money, and the openness of the realm. The price
// improvement of the midpoint rule is a transfer, so ImportValue and ExportValue
// differ from the units at the table; neither is production. By-node rows need
// the OutsideTraded event, which the tally does not read until the outside
// module is in the tree.
type TradeBalanceResult struct {
	ConfigHash [32]byte
	From, To   int64
	Lines      []TradeLine
	Gates      []TradeGate
	// Total is the sum of the gates.
	Total TradeGate
	// Openness is (ExportValue + ImportValue) / GDP, in basis points, defined where
	// OpennessOK (a value added above 0).
	Openness   BP
	OpennessOK bool
	Warnings   []Warning
}

// TradeBalance returns the trade with the outside of a window, valued at p, and
// its openness against the realm's value added in gdp (the GDP of the same
// window and table). A gate with no outside flow has no row.
func TradeBalance(w Window, p Prices, gdp GDPResult) (TradeBalanceResult, error) {
	if err := w.checkTable("TradeBalance", p); err != nil {
		return TradeBalanceResult{}, err
	}
	if err := w.checkRegional("TradeBalance"); err != nil {
		return TradeBalanceResult{}, err
	}
	if gdp.ConfigHash != w.ConfigHash || gdp.From != w.From || gdp.To != w.To {
		return TradeBalanceResult{}, wrap("TradeBalance", "gdp", ErrWindowOrder)
	}
	f, err := flowsOf(w)
	if err != nil {
		return TradeBalanceResult{}, err
	}
	res := TradeBalanceResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To, Warnings: withWarnings(w)}
	if res.Lines, err = tradeLines(w, f, p); err != nil {
		return TradeBalanceResult{}, err
	}
	if err = res.gates(w, f); err != nil {
		return TradeBalanceResult{}, err
	}
	err = res.openness(gdp)
	return res, err
}

// tradeLines returns the units imported and exported by gate and commodity.
func tradeLines(w Window, f flowSet, p Prices) ([]TradeLine, error) {
	var tables [3][]int64
	for k, reason := range [...]economysim.Reason{reasonOutsideOffer, reasonOutsideUnsold, reasonOutsidePurchase} {
		var err error
		if tables[k], err = f.units(reason); err != nil {
			return nil, err
		}
	}
	var lines []TradeLine
	for i := range tables[0] {
		imp, err := net("TradeBalance", "imp", []int64{tables[0][i]}, tables[1][i])
		if err != nil {
			return nil, err
		}
		exp := tables[2][i]
		if imp == 0 && exp == 0 {
			continue
		}
		r, c := economysim.RegionID(i/w.Commodities), economysim.CommodityID(i%w.Commodities)
		var iv, ev acc
		iv.add(imp, p.At(r, c))
		ev.add(exp, p.At(r, c))
		line := TradeLine{Gate: r, Commodity: c, Imp: imp, Exp: exp}
		if line.ImpValue, err = iv.int64("TradeBalance", "imp_value"); err != nil {
			return nil, err
		}
		if line.ExpValue, err = ev.int64("TradeBalance", "exp_value"); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// gates fills the money of each gate from the six outside reasons.
func (res *TradeBalanceResult) gates(w Window, f flowSet) error {
	budget, unspent, receipts := f.money(reasonOutsideBudget), f.money(reasonOutsideUnspent), f.money(reasonOutsideReceipts)
	const name = "TradeBalance"
	var imports, exports, nets []int64
	for r := 0; r < w.Regions; r++ {
		if budget[r] == 0 && unspent[r] == 0 && receipts[r] == 0 {
			continue
		}
		g := TradeGate{Gate: economysim.RegionID(r), ImportValue: receipts[r], Binding: w.Regional[r].Binding}
		var err error
		if g.ExportValue, err = net(name, "export_value", []int64{budget[r]}, unspent[r]); err != nil {
			return err
		}
		if g.Net, err = net(name, "net", []int64{g.ExportValue}, g.ImportValue); err != nil {
			return err
		}
		res.Gates = append(res.Gates, g)
		imports, exports, nets = append(imports, g.ImportValue), append(exports, g.ExportValue), append(nets, g.Net)
	}
	var err error
	if res.Total.ImportValue, err = net(name, "import_value", imports); err != nil {
		return err
	}
	if res.Total.ExportValue, err = net(name, "export_value", exports); err != nil {
		return err
	}
	res.Total.Net, err = net(name, "net", nets)
	return err
}

// openness sets (X + M) / GDP in basis points. A sum past int64 is ErrOverflow.
func (res *TradeBalanceResult) openness(gdp GDPResult) error {
	if gdp.Realm.ValueAdded <= 0 {
		return nil
	}
	xm, err := net("TradeBalance", "openness", []int64{res.Total.ExportValue, res.Total.ImportValue})
	if err != nil {
		return err
	}
	n := measure.MulInt64(10_000, xm)
	q, ok := measure.RoundDiv(n, measure.FromInt64(gdp.Realm.ValueAdded))
	if !ok {
		return nil
	}
	v, clamped := measure.ClampBP(q) // FR-IND-023: held at MaxIndexBP and flagged
	res.Openness, res.OpennessOK = BP(v), true
	res.Warnings = clampWarning(res.Warnings, Index{Clamped: clamped})
	return nil
}
