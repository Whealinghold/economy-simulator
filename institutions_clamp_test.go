package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The legal prices of the institutions module over a whole run of pack one (the staple of
// the reference price 96 in the region north, spec-institutions.md 6.3, 6.5): the range
// holds for every trade, and a cap that binds shows as a shortage.

// stapleControl is pack one with a control on the staple in north of the levels given.
func stapleControl(t testing.TB, capBP, floorBP int64) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, packOne...)
	cfg.Modules.Institutions = &economysim.Institutions{Controls: []economysim.Control{
		{ID: "assize", Commodity: "staple", Region: "north", CapBP: capBP, FloorBP: floorBP},
	}}
	return cfg
}

// runStaple runs the world of cfg for ticks from the seed and returns the events of the
// staple, the commodity with index 5 of pack one.
func runStaple(t testing.TB, cfg economysim.Config, seed uint64, ticks int) (trades []economysim.TradeExecuted, cleared []economysim.MarketCleared, controlled []economysim.PriceControlled) {
	t.Helper()
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		switch e := e.(type) {
		case economysim.TradeExecuted:
			if e.Commodity == 5 {
				trades = append(trades, e)
			}
		case economysim.MarketCleared:
			if e.Commodity == 5 {
				cleared = append(cleared, e)
			}
		case economysim.PriceControlled:
			controlled = append(controlled, e)
		}
	})
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(obs), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() seed %d error = %v, want nil", seed, err)
		}
	}
	return trades, cleared, controlled
}

// TestClamp_EveryTradeInsideRange verifies that every trade of a controlled market is at a
// price inside the legal range, whatever the seed (spec-institutions.md 6.3.2, FR-ORG-031),
// with the audit on: no trade is lost to the clamp and the books balance (conservation).
//
// Arithmetic: the reference price of the staple is 96. A cap of 9 000 bp is 96 * 9 000 / 10 000
// = 86 (truncated) and a floor of 11 000 bp is 96 * 11 000 / 10 000 = 105. The market runs
// near 95, so each control binds. A control that never bound would pass vacuously, so the
// run needs trades and clamps.
//
// Requirement: FR-ORG-031, FR-ORG-041
func TestClamp_EveryTradeInsideRange(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, tc := range []struct {
		name          string
		capBP, flrBP  int64
		lo, hi        economysim.Price
		wantBidsAsks  func(economysim.PriceControlled) int64
		wantSideLabel string
	}{
		{"cap", 9_000, 0, 1, 86, func(e economysim.PriceControlled) int64 { return e.Bids }, "bids lowered"},
		{"floor", 0, 11_000, 105, economysim.MaxPrice, func(e economysim.PriceControlled) int64 { return e.Asks }, "asks raised"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for seed := uint64(1); seed <= 3; seed++ {
				trades, _, controlled := runStaple(t, stapleControl(t, tc.capBP, tc.flrBP), seed, 40)
				var clamped int64
				for _, e := range controlled {
					clamped += tc.wantBidsAsks(e)
				}
				if len(trades) == 0 || clamped == 0 {
					t.Errorf("seed %d: %d trades, %d %s, want both above 0 so the control binds", seed, len(trades), clamped, tc.wantSideLabel)
				}
				for _, e := range trades {
					if e.Price < tc.lo || e.Price > tc.hi {
						t.Errorf("seed %d: trade at tick %d at price %d, want it in [%d, %d]", seed, e.Tick, e.Price, tc.lo, tc.hi)
					}
				}
			}
		})
	}
}

// TestMarketCleared_ShortageSignal verifies the signal of a binding cap (spec-institutions.md
// 6.5.1, 11.1.1): with the cap of 86 below the price at which the staple is offered, bid
// units go unmatched and the best unmatched ask is above the cap, and no trade is above it.
//
// Requirement: FR-ORG-031, FR-ORG-041
func TestMarketCleared_ShortageSignal(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	trades, cleared, controlled := runStaple(t, stapleControl(t, 8_000, 0), 1, 40)
	var short bool
	for _, e := range cleared {
		short = short || (e.UnmatchedBid > 0 && e.HasBestAsk && int64(e.BestAsk) > 76)
	}
	if !short {
		t.Errorf("no MarketCleared of the staple has unmatched bid units and a best unmatched ask above the cap of 76 (96 * 8 000 / 10 000), want the shortage signal")
	}
	for _, e := range trades {
		if e.Price > 76 {
			t.Errorf("trade at tick %d at price %d, want at most the cap of 76", e.Tick, e.Price)
		}
	}
	if len(controlled) == 0 {
		t.Errorf("PriceControlled events = 0, want the clamp told")
	}
}
