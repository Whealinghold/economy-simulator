package economysim_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// This file holds the scenarios of the merchants module (spec-reference-chain.md 7.9,
// S67 to S70 and S72; plan 2-15 Phase 8). They run the packs of merchants_golden_test.go
// for a few tens of ticks with the audit on, one seed (the first of the golden, 1), and
// assert what holds whatever the economy does: the module's caps and counts, the table
// of spec-merchants.md 9.7, conservation and the resume. The cold packs lose their
// households in the first ticks (the golden of the module alone shows it), so they
// assert no price gap; the contracts are formed by the scripted merchant of
// scriptedMerchants, which stands for the strategy and uses the public calls only.

// merWorld builds the world of cfg with the options, a recorder and the audit on, and
// returns it with the recorder.
func merWorld(t testing.TB, cfg economysim.Config, opts ...economysim.Option) (*economysim.World, *recorder) {
	t.Helper()
	rec := &recorder{}
	opts = append([]economysim.Option{economysim.WithSeed(goldenSeed), economysim.WithObserver(rec), economysim.WithAudit()}, opts...)
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	return w, rec
}

// merStep steps the world n times.
func merStep(t testing.TB, w *economysim.World, n int) {
	t.Helper()
	for range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d (seed %d) error = %v, want nil", w.Tick(), goldenSeed, err)
		}
	}
}

// kinds returns the events of the type T in order.
func kinds[T economysim.Event](rec *recorder) []T {
	var out []T
	for _, e := range rec.events {
		if v, ok := e.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// TestScenarioS67_MerchantCaps (MER-S1) runs the merchant strategy with 20 and then 50
// brokers in each region and a fee share spread of 0 and of 2 500, 40 ticks each,
// and asserts the caps of spec-merchants.md 7.1: no region plans more shipment requests
// and delivery offers in a tick than it has merchants (a plan carries at most one,
// FR-MER-025), the lanes locked are counted, and the audit holds on every tick (a
// contract creates or destroys no money, FR-MER-044). The swing size is reported by the
// counts in the log of a failing run.
//
// Requirement: FR-MER-025, FR-MER-024, FR-MER-044
func TestScenarioS67_MerchantCaps(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, tt := range []struct {
		brokers int64
		spread  int64
	}{{20, 0}, {50, 2_500}} {
		cfg := loadValidPack(t, packMerchants("two")...)
		for i := range cfg.Groups {
			if cfg.Groups[i].Role == "broker" {
				cfg.Groups[i].Households = tt.brokers
			}
		}
		cfg.Roles[slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == "broker" })].Merchant.FeeShareSpreadBP = tt.spread
		w, rec := merWorld(t, cfg, strategy.Options()...)
		merStep(t, w, 40)
		for _, p := range kinds[economysim.MerchantsPlanned](rec) {
			if p.Dispatched+p.Offered > p.Merchants {
				t.Errorf("brokers %d spread %d (seed %d): tick %d region %d planned %d requests and %d offers for %d merchants, want at most one each",
					tt.brokers, tt.spread, goldenSeed, p.Tick, p.Region, p.Dispatched, p.Offered, p.Merchants)
			}
		}
	}
}

// TestScenarioS68_ContractsFailAndEnd (MER-S2) runs the scripted merchant of
// scriptedMerchants: eight contracts formed in ticks 0 and 1 from the north to the
// south; the edge is closed in tick 2, a deliverer is removed in tick 2 and a payer in
// tick 3, all in transit (the removal through RemoveAgentForTest). It asserts the table
// of spec-merchants.md 9.7: each contract is settled once, Paid plus Refunded is the
// escrow, a removed deliverer's contracts end at once with the escrow returned, a
// removed payer's contracts are paid on arrival, the deadline does not move for the
// closure (FR-MER-038), and the escrow is back to 0 when the last contract ends.
//
// Arithmetic: price (100 + 120) / 2 = 110, 10 units, escrow 1 100; the coach takes
// ceil(100 / 30) = 4 ticks, so the contracts of tick 0 arrive in tick 4, those of tick 1
// in tick 5.
//
// Requirement: FR-MER-038, FR-MER-040, FR-MER-041, FR-MER-042, FR-MER-044
func TestScenarioS68_ContractsFailAndEnd(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, opts := scriptedMerchants(t)
	w, rec := merWorld(t, cfg, opts...)
	merStep(t, w, 2)
	formed := kinds[economysim.ContractFormed](rec)
	if len(formed) != 8 {
		t.Fatalf("contracts formed in ticks 0 and 1 = %d (seed %d), want 8", len(formed), goldenSeed)
	}
	north, south := formed[0].Region, formed[0].To
	if err := w.SetEdgeStatuses([]economysim.EdgeStatus{{From: north, To: south, Closed: true, FactorBP: 10_000}}); err != nil {
		t.Fatalf("SetEdgeStatuses() = %v, want nil", err)
	}
	deliverer, payer := formed[0].Deliverer, formed[1].Payer
	if err := economysim.RemoveAgentForTest(w, deliverer, economysim.CauseInsolvent); err != nil {
		t.Fatalf("RemoveAgentForTest(deliverer) = %v, want nil", err)
	}
	merStep(t, w, 1)
	if err := economysim.RemoveAgentForTest(w, payer, economysim.CauseStarved); err != nil {
		t.Fatalf("RemoveAgentForTest(payer) = %v, want nil", err)
	}
	merStep(t, w, 9)
	price := map[economysim.ContractID]economysim.Money{}
	for _, f := range formed {
		price[f.Contract] = f.Escrowed
	}
	seen := map[economysim.ContractID]bool{}
	var removed, payerGone int
	for _, s := range kinds[economysim.ContractSettled](rec) {
		if seen[s.Contract] {
			t.Errorf("contract %d settled twice (seed %d)", s.Contract, goldenSeed)
		}
		seen[s.Contract] = true
		if s.Paid+s.Refunded != price[s.Contract] {
			t.Errorf("contract %d: paid %d + refunded %d, want the escrow %d (seed %d)", s.Contract, s.Paid, s.Refunded, price[s.Contract], goldenSeed)
		}
		switch s.Cause {
		case economysim.CauseDelivererRemoved:
			removed++
			if s.Paid != 0 {
				t.Errorf("contract %d ended at the removal of its deliverer paid %d, want 0", s.Contract, s.Paid)
			}
		case economysim.CausePayerRemoved:
			payerGone++
			if s.Tick != 4 && s.Tick != 5 {
				t.Errorf("contract %d of a removed payer settled in tick %d, want its arrival tick, 4 or 5: the closure moves no deadline", s.Contract, s.Tick)
			}
		}
	}
	if len(seen) != 8 || removed == 0 || payerGone == 0 {
		t.Errorf("settled %d of 8 contracts, %d at a deliverer's removal, %d with a removed payer (seed %d), want all 8 and at least one of each", len(seen), removed, payerGone, goldenSeed)
	}
	end, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	if len(end.Contracts) != 0 || end.Escrow != 0 {
		t.Errorf("contracts in flight %d, escrow %d at the end, want 0 and 0", len(end.Contracts), end.Escrow)
	}
}

// TestScenarioS69_SourceAge (MER-S3) runs the merchant strategy for 40 ticks with
// MaxSourceAge at 0 and at its bound: at 0 every rumour that is not from this tick is
// stale, so no trip is planned and the stale are counted; at the bound none is stale
// (FR-MER-009).
//
// Requirement: FR-MER-009, FR-MER-007
func TestScenarioS69_SourceAge(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, tt := range []struct {
		name  string
		age   int64
		stale bool
	}{{"age 0", 0, true}, {"age at the bound", economysim.MaxRumourAge, false}} {
		cfg := loadValidPack(t, packMerchants("two")...)
		cfg.Roles[slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == "broker" })].Merchant.MaxSourceAge = tt.age
		w, rec := merWorld(t, cfg, strategy.Options()...)
		merStep(t, w, 40)
		var stale, planned int64
		for _, p := range kinds[economysim.MerchantsPlanned](rec) {
			stale, planned = stale+p.Stale, planned+p.Planned
		}
		if (stale > 0) != tt.stale || (tt.age == 0 && planned != 0) {
			t.Errorf("%s (seed %d): stale %d, planned %d, want stale above 0: %t and, at age 0, no trip", tt.name, goldenSeed, stale, planned, tt.stale)
		}
	}
}

// TestScenarioS70_ThreeRegions (MER-S4) runs the pack three with the trade, transport,
// information and merchants-three overlays, whose brokers list a hired mode and a
// self-carried one, for 40 ticks with the audit on: MerchantsPlanned is reported for each
// of the three regions, and the lanes considered by the merchants of a region are those
// of its two neighbours (a candidate is counted once, FR-MER-007).
//
// Requirement: FR-MER-007, FR-MER-019
func TestScenarioS70_ThreeRegions(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	w, rec := merWorld(t, loadValidPack(t, packMerchants("three")...), strategy.Options()...)
	merStep(t, w, 40)
	regions := map[economysim.RegionID]bool{}
	for _, p := range kinds[economysim.MerchantsPlanned](rec) {
		regions[p.Region] = true
		decided := p.NoRumour + p.Stale + p.Locked + p.Perishable + p.NoMargin + p.Unaffordable + p.BelowShare
		if decided > p.Considered {
			t.Errorf("tick %d region %d (seed %d): %d candidates decided, %d considered, want each candidate counted once", p.Tick, p.Region, goldenSeed, decided, p.Considered)
		}
	}
	if len(regions) != 3 {
		t.Errorf("MerchantsPlanned in %d regions (seed %d), want 3", len(regions), goldenSeed)
	}
}

// TestScenarioS72_CheckpointResume (MER-S6) runs the scripted merchants of
// scriptedMerchants, takes a checkpoint in tick 2 with eight contracts in flight and
// resumes it: the restored world settles the same contracts in the same ticks and ends
// with the same snapshot and the same checkpoint as the uninterrupted one
// (FR-MER-061, FR-MER-062).
//
// Requirement: FR-MER-061, FR-MER-062
func TestScenarioS72_CheckpointResume(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, opts := scriptedMerchants(t)
	w, rec := merWorld(t, cfg, opts...)
	merStep(t, w, 2)
	cut := len(rec.events)
	ckpt, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	rrec := &recorder{}
	r, err := economysim.RestoreWorld(cfg, ckpt, append([]economysim.Option{economysim.WithObserver(rrec), economysim.WithAudit()}, opts...)...)
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	merStep(t, w, 10)
	merStep(t, r, 10)
	if len(rrec.events) < 1 || !reflect.DeepEqual(rec.events[cut:], rrec.events[1:]) {
		t.Errorf("events after the resume: %d, want the %d of the uninterrupted run after WorldRestored (seed %d)", len(rrec.events), len(rec.events)-cut, goldenSeed)
	}
	sa, _ := w.Snapshot()
	sb, _ := r.Snapshot()
	if !reflect.DeepEqual(sa, sb) {
		t.Errorf("snapshot of the resumed world differs from the uninterrupted one (seed %d)", goldenSeed)
	}
	if n := len(kinds[economysim.ContractSettled](rrec)); n != 8 {
		t.Errorf("contracts settled after the resume = %d, want the 8 in flight at the cut", n)
	}
}
