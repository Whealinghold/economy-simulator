package indicator_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// lossStorePack is pack one with the storage module on and a store risk on the
// grower's barn. The growers' base room is cut to 100 and the barn raised to 5 000
// so that their 714 units of warm stock lie in the barn (the stored part, which a
// strike can take) and the pledged stock still fits. The row strikes the barn every
// tick (10 000) and takes a fifth of what it holds (2 000), so every window sees a
// loss while the stock lasts.
func lossStorePack(t *testing.T) economysim.Config {
	t.Helper()
	cfg := loadPack(t, "base.toml", "one.toml", "warm-one.toml", "overlay/storage.toml")
	for i := range cfg.Modules.Storage.Roles {
		cfg.Modules.Storage.Roles[i].Base = &economysim.Capacity{Mass: 100, Bulk: 100}
	}
	for i := range cfg.Modules.Storage.Classes {
		cfg.Modules.Storage.Classes[i].Mass, cfg.Modules.Storage.Classes[i].Bulk = 5_000, 5_000
	}
	cfg.Modules.Risk = &economysim.Risk{Stores: []economysim.StoreRisk{
		{Region: "north", Class: "barn", StrikeBP: 10_000, SeverityBP: 2_000},
	}}
	return cfg
}

// TestRun_StoreLossReadByLossesIndicator verifies, end to end over a real run of pack
// one with storage and risk on, that a struck store reaches the consumer: StoreLoss
// events are emitted, the Losses preset of every closed window reads exactly the units
// those events carry (by source store, none by route) with a Gap of 0, the loss
// boundary of the closing snapshot holds goods only (a sink of units, never money) and
// equals the sum of the events, and the run is the same for the same seed.
//
// Requirement: FR-IND-059, FR-RSK-021, FR-RSK-022, FR-RSK-023
func TestRun_StoreLossReadByLossesIndicator(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := lossStorePack(t)
	events, wins := runStoreLoss(t, cfg)
	if len(events) == 0 {
		t.Fatal("run emitted no StoreLoss, want the certain strike to take stock")
	}
	var total, fromEvents int64
	for _, rw := range wins {
		inWindow := unitsLostIn(events, rw.w.From, rw.w.To)
		fromEvents += inWindow
		requireStoreLosses(t, rw.w, inWindow)
		total = lossBoundaryGoods(t, rw.closing)
	}
	if fromEvents == 0 {
		t.Fatal("no window read a loss, want units lost")
	}
	if total != fromEvents {
		t.Errorf("loss boundary of the last snapshot = %d units, StoreLoss events sum to %d, want equal", total, fromEvents)
	}
	if again, _ := runStoreLoss(t, cfg); len(again) != len(events) {
		t.Errorf("second run with seed 1 emitted %d StoreLoss, first emitted %d, want equal", len(again), len(events))
	}
}

// runStoreLoss runs cfg from seed 1 for four windows of five ticks and returns the
// StoreLoss events and the windows.
func runStoreLoss(t *testing.T, cfg economysim.Config) (events []economysim.StoreLoss, wins []runWin) {
	t.Helper()
	w, rec := newRun(t, cfg, 1, true)
	wins = stepWindows(t, w, rec, 5, 4, nil)
	for _, e := range rec.events {
		if s, ok := e.(economysim.StoreLoss); ok {
			events = append(events, s)
		}
	}
	return events, wins
}

// unitsLostIn sums the units the events of ticks [from, to) lost.
func unitsLostIn(events []economysim.StoreLoss, from, to int64) (n int64) {
	for _, e := range events {
		if e.Tick < from || e.Tick >= to {
			continue
		}
		for _, a := range e.Lost {
			n += int64(a.Quantity)
		}
	}
	return n
}

// requireStoreLosses fails the test unless Losses over w, at a zero price table, reads
// want units, all of source store, with the boundary flow equal and a Gap of 0.
func requireStoreLosses(t *testing.T, w indicator.Window, want int64) {
	t.Helper()
	cells := make([]economysim.Price, w.Regions*w.Commodities)
	p, err := indicator.NewPrices(w.ConfigHash, w.From, w.To, w.Regions, w.Commodities, cells)
	if err != nil {
		t.Fatalf("NewPrices() error = %v, want nil", err)
	}
	got, err := indicator.Losses(w, p)
	if err != nil {
		t.Fatalf("Losses(window [%d, %d)) error = %v, want nil", w.From, w.To, err)
	}
	if got.Units != want || got.Gap != 0 || got.Boundary != want {
		t.Errorf("Losses(window [%d, %d)) = (units %d, boundary %d, gap %d), want all %d from the StoreLoss events", w.From, w.To, got.Units, got.Boundary, got.Gap, want)
	}
	for _, r := range got.Rows {
		if r.Source != indicator.SourceStore {
			t.Errorf("Losses row %+v has source %v, want store only: no route is configured", r, r.Source)
		}
	}
}

// lossBoundaryGoods returns the units the loss boundary holds in a snapshot and fails
// the test if the reason holds any money: a loss is a sink of goods only (FR-RSK-022).
func lossBoundaryGoods(t *testing.T, s economysim.Snapshot) (units int64) {
	t.Helper()
	for _, b := range s.Boundary {
		if b.Reason != economysim.ReasonLoss {
			continue
		}
		if b.Commodity == economysim.NoCommodity {
			if b.Balance != 0 {
				t.Errorf("loss boundary holds money %d, want 0", b.Balance)
			}
			continue
		}
		units += b.Balance
	}
	return units
}
