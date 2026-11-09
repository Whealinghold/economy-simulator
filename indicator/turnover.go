package indicator

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// TurnoverRow is the trade of one market over the window: the value
// sum(Price * Volume), the volume in units (not value, N-201) and the ticks with
// volume out of the ticks it cleared (Active, Ticks), so that a thin market is
// flagged.
type TurnoverRow struct {
	Region        economysim.RegionID
	Commodity     economysim.CommodityID
	Value, Volume int64
	Active, Ticks int64
}

// TurnoverResult is the value and volume of trade of the selected markets, and
// the velocity of the money: Velocity = rdiv(10 000 * Value, M) per window and
// Annualised = rdiv(10 000 * Value * TicksPerYear, M * n) from the exact sums
// (spec-indicators.md 9.1, row 8). M is the money of the selected regions' sectors
// at the close.
type TurnoverResult struct {
	ConfigHash    [32]byte
	From, To      int64
	Rows          []TurnoverRow
	Value, Volume int64
	Money         int64
	Velocity      Index
	Annualised    Index
	Warnings      []Warning
}

// Turnover returns the trade of the markets in sel and the velocity of the money
// of their regions at the closing snapshot, annualised with ticksPerYear
// (ErrRange below 1). WarnThin marks a result with a market that never traded.
// A velocity with a falling household money and a rising pool is a different
// story from a falling turnover; the money of each sector is Sectors'.
func Turnover(w Window, closing economysim.Snapshot, ticksPerYear int64, sel CellSel) (TurnoverResult, error) {
	if ticksPerYear < 1 {
		return TurnoverResult{}, wrap("Turnover", "ticks per year", ErrRange)
	}
	if closing.ConfigHash != w.ConfigHash || closing.Tick != w.To {
		return TurnoverResult{}, wrap("Turnover", "snapshot", ErrWindowOrder)
	}
	if len(closing.Regions) != w.Regions {
		return TurnoverResult{}, wrap("Turnover", "regions", ErrRange)
	}
	res := TurnoverResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}
	var value, volume acc
	thin := false
	err := forMarkets(w, sel, func(i int, r economysim.RegionID, c economysim.CommodityID) error {
		m := w.Markets[i]
		pv, ok := measure.FromUint128(m.SumPV.u())
		if !ok {
			return wrap("Turnover", "value", ErrOverflow)
		}
		row := TurnoverRow{Region: r, Commodity: c, Volume: m.SumV, Active: m.Active, Ticks: m.Ticks}
		v, fits := measure.ToInt64(pv)
		if !fits {
			return wrap("Turnover", "value", ErrOverflow)
		}
		row.Value = v
		value.addInt(pv)
		volume.addInt(measure.FromInt64(m.SumV))
		thin = thin || (m.Ticks > 0 && m.Active == 0)
		res.Rows = append(res.Rows, row)
		return nil
	})
	if err != nil {
		return TurnoverResult{}, err
	}
	if err := res.velocity(w, closing, ticksPerYear, sel, value, volume); err != nil {
		return TurnoverResult{}, err
	}
	if thin {
		res.Warnings = []Warning{WarnThin}
	}
	return res, nil
}

// velocity sets the totals and the two velocities.
func (res *TurnoverResult) velocity(w Window, closing economysim.Snapshot, ticksPerYear int64, sel CellSel, value, volume acc) error {
	var err error
	if res.Value, err = value.int64("Turnover", "value"); err != nil {
		return err
	}
	if res.Volume, err = volume.int64("Turnover", "volume"); err != nil {
		return err
	}
	var money acc
	for g, r := range closing.Regions {
		if sel.Regions == nil || containsID(sel.Regions, economysim.RegionID(g)) {
			money.addInt(measure.FromInt64(int64(r.MoneyHouseholds) + int64(r.MoneyPool)))
		}
	}
	if res.Money, err = money.int64("Turnover", "money"); err != nil {
		return err
	}
	if res.Velocity, err = ratioWide("Turnover", "velocity", value.v, money.v); err != nil {
		return err
	}
	// Value and Money fit int64 (checked above), so the products are exact 128-bit.
	res.Annualised, err = ratioWide("Turnover", "annualised", measure.MulInt64(res.Value, ticksPerYear), measure.MulInt64(res.Money, w.Steps))
	return err
}
