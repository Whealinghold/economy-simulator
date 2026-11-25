package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The scenarios of the fiscal module (spec-fiscal.md 19, spec-reference-chain.md 7.12, plan 3-fis
// Phase 10) run on the pack of the golden, the warm five with overlay/polity-governed.toml and
// overlay/fiscal-governed.toml (S135 on `one` and S136 on `three` build their own governance in a
// document). They assert what is exact (the accounting of each flow, the audit, the equality of two
// worlds) and report, never assert, what the bands of STY will judge (N-027). A scenario keeps counters,
// not per-seed snapshots (plan 3-fis, gate economics; the S113 lesson).

// fiveFiscal is the pack of the module-alone golden: the governed warm five with the two fiscal overlays
// in the layering order of spec-reference-chain.md 4.5.
var fiveFiscal = append(slices.Clone(fivePolityGoverned), "overlay/fiscal.toml", "overlay/fiscal-governed.toml")

// fiscalFlowIDs are the IDs of every flow of the module of cfg, in the order of its six lists.
func fiscalFlowIDs(cfg economysim.Config) []string {
	f := cfg.Modules.Fiscal
	var ids []string
	for _, x := range f.Taxes {
		ids = append(ids, x.ID)
	}
	for _, x := range f.Duties {
		ids = append(ids, x.ID)
	}
	for _, x := range f.Tolls {
		ids = append(ids, x.ID)
	}
	for _, x := range f.Payouts {
		ids = append(ids, x.ID)
	}
	for _, x := range f.Purchases {
		ids = append(ids, x.ID)
	}
	for _, x := range f.Releases {
		ids = append(ids, x.ID)
	}
	return ids
}

// fiscalKept is what a scenario run keeps of the events: the sums of the module's flow events by kind and
// flow, the module's other events by count, and the money of every flow for the rebuild of a treasury.
type fiscalKept struct {
	applied map[string]economysim.Money // by flow ID: the sum of Applied
	due     map[string]economysim.Money // by flow ID: the sum of Due
	short   map[string]int64            // by flow ID: the sum of Short
	cut     map[string]int64            // by flow ID: the sum of Cut
	exempt  map[string]int64            // by flow ID: the sum of Exempt
	events  map[string]int              // by flow ID: the number of FiscalFlowApplied
	kinds   map[string]economysim.Kind  // by flow ID
	ordered int                         // TreasuryOrdered
	scaled  int                         // MoneyFlowScaled
	toll    economysim.Money            // the sum of ShipmentDispatched.Toll
	refused int                         // ShipmentRejected with the reason toll_unaffordable
	others  map[string]int
}

func newFiscalKept() *fiscalKept {
	return &fiscalKept{
		applied: map[string]economysim.Money{}, due: map[string]economysim.Money{}, short: map[string]int64{},
		cut: map[string]int64{}, exempt: map[string]int64{}, events: map[string]int{}, kinds: map[string]economysim.Kind{},
		others: map[string]int{},
	}
}

// Observe keeps what the scenarios read of e.
func (k *fiscalKept) Observe(e economysim.Event) {
	switch v := e.(type) {
	case economysim.FiscalFlowApplied:
		k.applied[v.Flow] += v.Applied
		k.due[v.Flow] += v.Due
		k.short[v.Flow] += v.Short
		k.cut[v.Flow] += v.Cut
		k.exempt[v.Flow] += v.Exempt
		k.events[v.Flow]++
		k.kinds[v.Flow] = v.Kind
	case economysim.TreasuryOrdered:
		k.ordered++
	case economysim.MoneyFlowScaled:
		k.scaled++
	case economysim.ShipmentDispatched:
		k.toll += v.Toll
	case economysim.ShipmentRejected:
		if v.Reason == economysim.ReasonTollUnaffordable {
			k.refused++
		}
	default:
		k.others[fmt.Sprintf("%T", e)]++
	}
}

// sumKind is the sum of Applied over the flows of kind.
func (k *fiscalKept) sumKind(kind economysim.Kind) economysim.Money {
	var s economysim.Money
	for f, a := range k.applied {
		if k.kinds[f] == kind {
			s += a
		}
	}
	return s
}

// fiscalWorld builds the world of cfg for seed with the audit on and a fresh fiscalKept observer.
func fiscalWorld(t testing.TB, cfg economysim.Config, seed uint64, opts ...economysim.Option) (*economysim.World, *fiscalKept) {
	t.Helper()
	k := newFiscalKept()
	all := append([]economysim.Option{economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(k)}, opts...)
	return mustWorld(t, cfg, all...), k
}

// fiscalStepTo steps w until its tick is n.
func fiscalStepTo(t testing.TB, w *economysim.World, n int64) {
	t.Helper()
	for w.Tick() < n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", w.Tick(), err)
		}
	}
}

// treasurySum is the money of the treasuries of w.
func treasurySum(t testing.TB, w *economysim.World) economysim.Money {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	var sum economysim.Money
	for _, j := range s.Jurisdictions {
		sum += j.Money
	}
	return sum
}

// fiscalScenario runs fn for the seeds of a scenario as parallel subtests and records the block under
// title in the file of pack.
func fiscalScenario(t *testing.T, pack, title string, seeds int, fn func(t *testing.T, rep *scenReport, seed uint64)) {
	t.Helper()
	rep := newPackScenReport(pack, fmt.Sprintf("%s seeds 1-%d", title, seeds))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(seeds) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				fn(t, rep, seed)
			})
		}
	})
	rep.flush(t)
}

// fiscalChecker is the exact check of a fiscal scenario for one seed: it records the verdict and fails the
// test (the shape of s155Check).
type fiscalChecker struct {
	t        testing.TB
	rep      *scenReport
	scenario string
	pack     string
	seed     uint64
}

// check records the verdict of the exact check name and fails the test with the detail when it is false.
func (c fiscalChecker) check(name string, ok bool, format string, args ...any) {
	c.t.Helper()
	checkExact(c.t, c.rep, c.scenario, c.pack, c.seed, name, ok, format, args...)
}

// treasuryMoneyRows is the money of each treasury of w, in ascending jurisdiction ID.
func treasuryMoneyRows(t testing.TB, w *economysim.World) []economysim.Money {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	out := make([]economysim.Money, len(s.Jurisdictions))
	for i, j := range s.Jurisdictions {
		out[i] = j.Money
	}
	return out
}
