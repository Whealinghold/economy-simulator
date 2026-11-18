package economysim

import (
	"slices"
	"testing"
)

// TestCost_TenureEventNoopAllocs verifies NFR-LND-001 for the rent step: with the default
// logger and the no-op observer, P8.3a of a world that charged rent, refused requests and
// moved the posted rent allocates nothing once the scratch has grown, because the event, its
// money rows and the log attributes are built only when an observer or an enabled logger
// asks for them (observability-and-logging.md 4.5, 8.2).
//
// Requirement: NFR-LND-001
func TestCost_TenureEventNoopAllocs(t *testing.T) {
	// No t.Parallel: AllocsPerRun changes GOMAXPROCS while it measures.
	w := newMill(t, stepConfig(10, 5_000, false), WithSeed(1))
	toStart(t, w) // grows the scratch of the tick
	c := &w.tenure.rt.counts[0]
	c.noTitle, c.rentFunds = 2, 1
	if c.paid == 0 {
		t.Fatalf("paid = 0 after the start, want rent paid so the Debug line is on the measured path")
	}
	got := testing.AllocsPerRun(20, func() { w.p83aRent() })
	if got != 0 {
		t.Errorf("allocs of P8.3a with the no-op observer and logger = %v, want 0", got)
	}
}

// BenchmarkApportionWorld measures the entitlements of P2.3 of a world with one titled
// endowment and ten households (NFR-LND-004), with b.ReportAllocs() for the trend beside
// the allocation test.
func BenchmarkApportionWorld(b *testing.B) {
	w := newMill(b, rationConfig(10, 5_000, false), WithSeed(1))
	toSnapshot(b, w)
	b.ReportAllocs()
	for b.Loop() {
		w.tenure.entitlements()
	}
}

// BenchmarkTiersWorld measures the four tiers of P2.6 for the one titled endowment of the
// same world (NFR-LND-004), with b.ReportAllocs().
func BenchmarkTiersWorld(b *testing.B) {
	w := newMill(b, rationConfig(10, 5_000, false), WithSeed(1))
	toStart(b, w)
	order := slices.Clone(w.res.order)
	b.ReportAllocs()
	for b.Loop() {
		w.tenure.entitlements()
		w.tenure.ration(order)
	}
}
