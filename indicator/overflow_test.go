package indicator_test

import (
	"errors"
	"math"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// TestPresets_MalformedAndExtremeInputs verifies that the account presets turn a
// window or a snapshot whose figures do not fit, or whose rows lie outside the
// names table, into ErrOverflow or ErrRange and leave the presets that do not
// read the bad figure untouched. The base is the worked window of 18.1; each
// case damages one place of it (an arrival row, a flow total, a money balance, a
// stock, a household count).
//
// Requirement: FR-IND-009, FR-IND-012
// Guard: every sum in the accounts is checked. If one were unchecked, a figure
// past int64 would wrap and an identity would report a small wrong residual.
func TestPresets_MalformedAndExtremeInputs(t *testing.T) {
	t.Parallel()
	big := int64(math.MaxInt64)
	arrival := indicator.ArrivalTally{Origin: alpha, Destination: beta, Commodity: grain, Delivered: big}
	// ok means the preset must succeed; the others name the sentinel it returns.
	type want struct{ gdp, sectors, cover, households, turnover, valueAdded error }
	tests := []struct {
		name   string
		damage func(w *indicator.Window, c *economysim.Snapshot)
		want   want
	}{
		{"arrival twice the largest quantity", func(w *indicator.Window, _ *economysim.Snapshot) {
			w.Arrivals = []indicator.ArrivalTally{arrival, arrival}
		}, want{gdp: indicator.ErrOverflow, sectors: indicator.ErrOverflow, valueAdded: indicator.ErrOverflow}},
		{"arrival from a region outside the names", func(w *indicator.Window, _ *economysim.Snapshot) {
			w.Arrivals = []indicator.ArrivalTally{{Origin: 9, Destination: beta, Commodity: grain}}
		}, want{gdp: indicator.ErrRange, sectors: indicator.ErrRange, valueAdded: indicator.ErrRange}},
		{"arrival of a commodity outside the names", func(w *indicator.Window, _ *economysim.Snapshot) {
			w.Arrivals = []indicator.ArrivalTally{{Origin: alpha, Destination: beta, Commodity: 9}}
		}, want{gdp: indicator.ErrRange, sectors: indicator.ErrRange, valueAdded: indicator.ErrRange}},
		{"output flow of the largest quantity", func(w *indicator.Window, _ *economysim.Snapshot) {
			addFlow(w, goodsFlow(10, alpha, economysim.ReasonRecipeOutput, grain, big))
		}, want{sectors: indicator.ErrOverflow}},
		{"money balances past int64", func(_ *indicator.Window, c *economysim.Snapshot) {
			c.Regions[0].MoneyHouseholds, c.Regions[1].MoneyHouseholds = economysim.Money(big), economysim.Money(big)
			c.Regions[0].MoneyPool = economysim.Money(big)
		}, want{sectors: indicator.ErrOverflow, turnover: indicator.ErrOverflow}},
		{"stock of the largest quantity", func(_ *indicator.Window, c *economysim.Snapshot) {
			setStock(c, alpha, grain, big, big, 0)
		}, want{gdp: indicator.ErrOverflow, sectors: indicator.ErrOverflow, cover: indicator.ErrOverflow, valueAdded: indicator.ErrOverflow}},
		{"households past int64", func(_ *indicator.Window, c *economysim.Snapshot) {
			c.Regions[0].Households, c.Regions[1].Households = big, big
		}, want{sectors: indicator.ErrOverflow}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, open, closing := workedWindow(t)
			closing.Regions = append([]economysim.RegionRow(nil), closing.Regions...)
			closing.Markets = append([]economysim.MarketRow(nil), closing.Markets...)
			tt.damage(&w, &closing)
			p := valleyTable(t, w, alphaP, betaP)
			cl := indicator.NewClasses(valleyConfig())
			_, gdp := indicator.GDP(w, open, closing, p)
			_, sectors := indicator.Sectors(w, open, closing, p)
			_, cover := indicator.StockCover(w, closing, indicator.CellSel{})
			_, households := indicator.Households(w, open, closing, 365)
			_, turnover := indicator.Turnover(w, closing, 365, indicator.CellSel{})
			_, valueAdded := indicator.ValueAdded(w, valleyConfig(), open, closing, p, cl)
			for _, c := range []struct {
				preset    string
				got, want error
			}{
				{"GDP", gdp, tt.want.gdp}, {"Sectors", sectors, tt.want.sectors}, {"StockCover", cover, tt.want.cover},
				{"Households", households, tt.want.households}, {"Turnover", turnover, tt.want.turnover},
				{"ValueAdded", valueAdded, tt.want.valueAdded},
			} {
				if c.want == nil && c.got != nil || c.want != nil && !errors.Is(c.got, c.want) {
					t.Errorf("%s(%s) error = %v, want %v", c.preset, tt.name, c.got, c.want)
				}
			}
		})
	}
}

// TestPeoplePricesAndDistributions_ExtremeInputs verifies that the household,
// price and distribution presets report ErrOverflow, ErrRange or
// ErrNegativeValue for counts and rows that do not fit or are not valid, and a
// window that only a hand could build (a negative volume sum, a price table of
// another world) is refused rather than read.
//
// Requirement: FR-IND-009, FR-IND-012, FR-IND-014
// Guard: a hand-built Window is the consumer's, so the presets cannot assume a
// Tally made it.
func TestPeoplePricesAndDistributions_ExtremeInputs(t *testing.T) {
	t.Parallel()
	big := int64(math.MaxInt64)
	w, _, closing := peopleWindow(t)
	huge := func(mut func(w *indicator.Window)) indicator.Window {
		c := w
		c.OpenHouseholds = append([]int64(nil), w.OpenHouseholds...)
		c.CloseHouseholds = append([]int64(nil), w.CloseHouseholds...)
		c.People = append([]indicator.PeopleTally(nil), w.People...)
		c.Markets = append([]indicator.MarketTally(nil), w.Markets...)
		mut(&c)
		return c
	}
	other := [32]byte{5}
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{"role shares with households past int64", func() error {
			_, err := indicator.RoleShares(huge(func(c *indicator.Window) { c.OpenHouseholds[0], c.OpenHouseholds[1] = big, big }))
			return err
		}, indicator.ErrOverflow},
		{"role shares with a closing count past int64", func() error {
			_, err := indicator.RoleShares(huge(func(c *indicator.Window) { c.CloseHouseholds[0], c.CloseHouseholds[1] = big, big }))
			return err
		}, indicator.ErrOverflow},
		{"unfed with household-ticks past int64", func() error {
			_, err := indicator.Unfed(huge(func(c *indicator.Window) { c.People[0].HouseholdTicks, c.People[1].HouseholdTicks = big, big }), closing)
			return err
		}, indicator.ErrOverflow},
		{"unfed with unfed past int64", func() error {
			_, err := indicator.Unfed(huge(func(c *indicator.Window) { c.People[0].Unfed, c.People[1].Unfed = big, big }), closing)
			return err
		}, indicator.ErrOverflow},
		{"prices of a negative volume sum", func() error {
			_, err := indicator.PricesOfWindow(huge(func(c *indicator.Window) { c.Markets[0].SumV = -1 }))
			return err
		}, indicator.ErrRange},
		{"prices past int64", func() error {
			_, err := indicator.PricesOfWindow(huge(func(c *indicator.Window) {
				c.Markets[0].SumV, c.Markets[0].SumPV = 1, indicator.Wide{Hi: 1}
			}))
			return err
		}, indicator.ErrOverflow},
		{"base of a table of another world", func() error {
			p, err := indicator.NewPrices(other, w.From, w.To, w.Regions, w.Commodities, make([]economysim.Price, w.Regions*w.Commodities))
			if err != nil {
				return err
			}
			_, err = indicator.BaseOf(w, p)
			return err
		}, indicator.ErrConfigMismatch},
		{"base of a table of another shape", func() error {
			p, err := indicator.NewPrices(w.ConfigHash, w.From, w.To, 1, 1, make([]economysim.Price, 1))
			if err != nil {
				return err
			}
			_, err = indicator.BaseOf(w, p)
			return err
		}, indicator.ErrRange},
		{"inequality of a negative value", func() error {
			_, err := indicator.Inequality(indicator.RowSet{Rows: []indicator.Household{{Weight: 1, Total: -1}}})
			return err
		}, indicator.ErrNegativeValue},
		{"inequality of sums past int64", func() error {
			_, err := indicator.Inequality(indicator.RowSet{Rows: []indicator.Household{{Weight: big, Total: big}, {Weight: big, Total: big}}})
			return err
		}, indicator.ErrOverflow},
		{"welfare of weights past int64", func() error {
			set := indicator.RowSet{Rows: []indicator.Household{{Weight: big, Total: 1}, {Weight: big, Total: 1}}}
			_, err := indicator.WelfareDistribution(set, indicator.RowSet{}, 10_000)
			return err
		}, indicator.ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Errorf("%s: error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

// TestTradeBalance_Misuse verifies that TradeBalance refuses a price table of
// another world and a GDP result of another window, so a balance is never valued
// at prices or openness of a different span.
//
// Requirement: FR-IND-042
func TestTradeBalance_Misuse(t *testing.T) {
	t.Parallel()
	w, open, closing := workedWindow(t)
	p := valleyTable(t, w, alphaP, betaP)
	gdp, err := indicator.GDP(w, open, closing, p)
	if err != nil {
		t.Fatalf("GDP() error = %v, want nil", err)
	}
	later := gdp
	later.To++
	foreign := p
	foreign.ConfigHash = [32]byte{8}
	tests := []struct {
		name string
		p    indicator.Prices
		gdp  indicator.GDPResult
		want error
	}{
		{"table of another world", foreign, gdp, indicator.ErrConfigMismatch},
		{"gdp of another window", p, later, indicator.ErrWindowOrder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := indicator.TradeBalance(w, tt.p, tt.gdp); !errors.Is(err, tt.want) {
				t.Errorf("TradeBalance(%s) error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

// TestCalendarWindows_TickPastTheCalendar verifies that a year or a season run
// whose ticks the calendar cannot read is ErrOverflow, not a wrapped tick.
//
// Requirement: FR-IND-052
func TestCalendarWindows_TickPastTheCalendar(t *testing.T) {
	t.Parallel()
	cal := calendar365()
	if _, err := indicator.YearWindow(cal, math.MaxInt64/2); !errors.Is(err, indicator.ErrOverflow) && !errors.Is(err, indicator.ErrRange) {
		t.Errorf("YearWindow(year near int64 max) error = %v, want ErrOverflow or ErrRange", err)
	}
	if _, err := indicator.SeasonWindows(cal, math.MaxInt64-10, math.MaxInt64, 0); err == nil {
		t.Errorf("SeasonWindows(ticks at int64 max) error = nil, want an error")
	}
}
