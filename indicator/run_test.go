package indicator_test

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/indicator"
)

// runPacks are the packs the run-level tests drive: pack one warm and the second,
// unrelated setting (D576, Risk 7: `five` is plan 24's).
var runPacks = []struct {
	name  string
	files []string
}{
	{"one", []string{"base.toml", "one.toml", "warm-one.toml"}},
	{"second", []string{"second/second.toml", "second/warm-second.toml"}},
}

const (
	runWindowTicks = 10
	runWindowCount = 3
	runSeeds       = 4
)

// loadPack loads a pack of testdata/chains.
func loadPack(t testing.TB, files ...string) economysim.Config {
	t.Helper()
	cfg, err := config.LoadFS(os.DirFS("../testdata/chains"), files...)
	if err != nil {
		t.Fatalf("LoadFS(%v) error = %v, want nil", files, err)
	}
	return cfg
}

// recorder is the observer of a run: it feeds the tally, keeps the units of the
// relocations it saw and, when asked, every event.
type recorder struct {
	tl     *indicator.Tally
	reloc  int64
	keep   bool
	events []economysim.Event
}

func (r *recorder) Observe(e economysim.Event) {
	r.tl.Observe(e)
	if x, ok := e.(economysim.AgentRelocated); ok {
		r.reloc += x.Units
	}
	if r.keep {
		r.events = append(r.events, e)
	}
}

// runWin is one closed window of a run with the snapshots that bound it and the
// units of the AgentRelocated events seen inside it.
type runWin struct {
	w             indicator.Window
	open, closing economysim.Snapshot
	reloc         int64
}

// stepWindows runs a world for n windows of length ticks, opening and closing the
// tally at each boundary, and calls each (if not nil) after every closed window.
func stepWindows(t testing.TB, w *economysim.World, rec *recorder, length int64, n int, each func(runWin)) []runWin {
	t.Helper()
	var out []runWin
	for i := 0; i < n; i++ {
		open, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		if err := rec.tl.Open(open); err != nil {
			t.Fatalf("Open() at tick %d error = %v, want nil", open.Tick, err)
		}
		rec.reloc = 0
		if _, err := w.Run(context.Background(), length); err != nil {
			t.Fatalf("Run() from tick %d error = %v, want nil", open.Tick, err)
		}
		closing, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		win, err := rec.tl.Close(closing)
		if err != nil {
			t.Fatalf("Close() at tick %d error = %v, want nil", closing.Tick, err)
		}
		rw := runWin{w: win, open: open, closing: closing, reloc: rec.reloc}
		out = append(out, rw)
		if each != nil {
			each(rw)
		}
	}
	return out
}

// newRun returns a world of cfg observed by a recorder.
func newRun(t testing.TB, cfg economysim.Config, seed uint64, keep bool) (*economysim.World, *recorder) {
	t.Helper()
	rec := &recorder{tl: indicator.NewTally(), keep: keep}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	return w, rec
}

// runCase is the windows of one (pack, seed).
type runCase struct {
	pack string
	seed uint64
	cfg  economysim.Config
	wins []runWin
}

// collectRuns drives every pack for the seeds of this build; the tests of the run
// level read the same runs (sharedRuns).
func collectRuns(t *testing.T) []runCase {
	t.Helper()
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	var out []runCase
	for _, p := range runPacks {
		cfg := loadPack(t, p.files...)
		for s := 1; s <= seedCount(runSeeds); s++ {
			w, rec := newRun(t, cfg, uint64(s), false)
			out = append(out, runCase{pack: p.name, seed: uint64(s), cfg: w.Config(), wins: stepWindows(t, w, rec, runWindowTicks, runWindowCount, nil)})
		}
	}
	return out
}

// runsOnce memoises collectRuns across the tests of the package.
var (
	runsMu   sync.Mutex
	runsDone []runCase
)

func sharedRuns(t *testing.T) []runCase {
	t.Helper()
	runsMu.Lock()
	defer runsMu.Unlock()
	if runsDone == nil {
		runsDone = collectRuns(t)
	}
	return runsDone
}

// TestRun_UnitsConserved verifies the identity of the ledger in units (7.1.3) over
// real runs: for every window, commodity, seed and pack, the change of the units
// agents and estates hold (plus 0 in transit: no shipment exists before plan 2-12)
// equals the corrected faucets less the corrected sinks of the commodity.
//
// Requirement: FR-IND-016
func TestRun_UnitsConserved(t *testing.T) {
	moved := 0
	for _, rc := range sharedRuns(t) {
		for i, rw := range rc.wins {
			flows, err := rw.w.Corrected()
			if err != nil {
				t.Fatalf("%s seed %d window %d: Corrected() error = %v, want nil", rc.pack, rc.seed, i, err)
			}
			net := make([]int64, len(rc.cfg.Commodities))
			for _, f := range flows {
				if f.Commodity == economysim.NoCommodity {
					continue
				}
				if f.Kind == economysim.KindFaucet {
					net[f.Commodity] += f.Total
				} else {
					net[f.Commodity] -= f.Total
				}
			}
			held := func(s economysim.Snapshot) []int64 {
				out := make([]int64, len(net))
				for _, m := range s.Markets {
					out[m.State.Commodity] += int64(m.HeldByAgents) + int64(m.HeldByEstate)
				}
				return out
			}
			o, c := held(rw.open), held(rw.closing)
			for k := range net {
				if net[k] != 0 {
					moved++
				}
				if c[k]-o[k] != net[k] {
					t.Errorf("%s seed %d window %d commodity %d: dHeld = %d, faucets - sinks = %d, want equal", rc.pack, rc.seed, i, k, c[k]-o[k], net[k])
				}
			}
		}
	}
	if moved == 0 {
		t.Error("no commodity changed in any window: the identity was checked on nothing")
	}
}

// TestRun_GDPIdentityHolds verifies that production equals expenditure at one
// price table in every window of every run, region by region and for the realm.
//
// Requirement: FR-IND-017
func TestRun_GDPIdentityHolds(t *testing.T) {
	for _, rc := range sharedRuns(t) {
		for i, rw := range rc.wins {
			p, err := indicator.PricesOfWindow(rw.w)
			if err != nil {
				t.Fatalf("%s seed %d window %d: PricesOfWindow() error = %v, want nil", rc.pack, rc.seed, i, err)
			}
			g, err := indicator.GDP(rw.w, rw.open, rw.closing, p)
			if err != nil {
				t.Fatalf("%s seed %d window %d: GDP() error = %v, want nil", rc.pack, rc.seed, i, err)
			}
			if g.Realm.Residual != 0 || g.TradeValuation != 0 {
				t.Errorf("%s seed %d window %d: realm residual = %d, trade valuation = %d, want 0 at one table", rc.pack, rc.seed, i, g.Realm.Residual, g.TradeValuation)
			}
			for _, r := range g.Regions {
				if r.Residual != 0 {
					t.Errorf("%s seed %d window %d region %d: residual = %d, want 0", rc.pack, rc.seed, i, r.Region, r.Residual)
				}
			}
		}
	}
}

// TestRun_NetWorthIdentity verifies the closing of 7.5.3 in every window: the
// change of the sectors' net worth is the money, the goods and the work in
// progress they gained, with no residual.
//
// Requirement: FR-IND-018
func TestRun_NetWorthIdentity(t *testing.T) {
	for _, rc := range sharedRuns(t) {
		for i, rw := range rc.wins {
			p, _ := indicator.PricesOfWindow(rw.w)
			s, err := indicator.Sectors(rw.w, rw.open, rw.closing, p)
			if err != nil {
				t.Fatalf("%s seed %d window %d: Sectors() error = %v, want nil", rc.pack, rc.seed, i, err)
			}
			if s.Closing.Residual != 0 {
				t.Errorf("%s seed %d window %d: net worth closing = %+v, want residual 0", rc.pack, rc.seed, i, s.Closing)
			}
		}
	}
}

// TestRun_RegionalMoneyIdentity verifies the money account of every region and of
// the realm in every window, and that the units of the relocation item are the
// Units of the AgentRelocated events (the terms that exist before plans 2-12,
// 2-15 and the fiscal module: relocation, faucets and sinks).
//
// Requirement: FR-IND-019
func TestRun_RegionalMoneyIdentity(t *testing.T) {
	for _, rc := range sharedRuns(t) {
		for i, rw := range rc.wins {
			p, _ := indicator.PricesOfWindow(rw.w)
			s, err := indicator.Sectors(rw.w, rw.open, rw.closing, p)
			if err != nil {
				t.Fatalf("%s seed %d window %d: Sectors() error = %v, want nil", rc.pack, rc.seed, i, err)
			}
			for _, r := range append(append([]indicator.SectorRow(nil), s.Regions...), s.Realm) {
				if r.Account.Residual != 0 {
					t.Errorf("%s seed %d window %d region %d: money account = %+v, want residual 0", rc.pack, rc.seed, i, r.Region, r.Account)
				}
			}
			var units int64
			for _, x := range rw.w.Relocations {
				units += x.Units
			}
			if units != rw.reloc {
				t.Errorf("%s seed %d window %d: relocated units in the window = %d, events = %d, want equal", rc.pack, rc.seed, i, units, rw.reloc)
			}
		}
	}
}

// TestRun_NonInterference verifies that no indicator changes a run (spec 3.1.2):
// the same seed, run once with no consumer and once with a consumer that closes a
// window and computes every preset after every step, ends in equal checkpoint
// bytes, the same event stream and equal snapshots.
//
// Requirement: FR-IND-003
func TestRun_NonInterference(t *testing.T) {
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	const ticks = 12
	cfg := loadPack(t, runPacks[0].files...)

	plain, plainRec := newRun(t, cfg, 3, true)
	for i := 0; i < ticks; i++ { // the same calls as the busy run: one Run of one tick each
		if _, err := plain.Run(context.Background(), 1); err != nil {
			t.Fatalf("Run() without a consumer error = %v, want nil", err)
		}
	}

	busy, busyRec := newRun(t, cfg, 3, true)
	var base indicator.Base
	computed := 0
	stepWindows(t, busy, busyRec, 1, ticks, func(rw runWin) {
		if computed == 0 {
			p, err := indicator.PricesOfWindow(rw.w)
			if err == nil {
				base, err = indicator.BaseOf(rw.w, p)
			}
			if err != nil {
				t.Fatalf("BaseOf() error = %v, want nil", err)
			}
		}
		presetsOf(t, busy.Config(), busy.Graph(), base, rw)
		computed++
	})
	if computed != ticks {
		t.Fatalf("the consumer ran %d times, want %d", computed, ticks)
	}

	a, errA := plain.Checkpoint()
	b, errB := busy.Checkpoint()
	if errA != nil || errB != nil {
		t.Fatalf("Checkpoint() errors = %v, %v, want nil", errA, errB)
	}
	ab, errA := a.MarshalBinary()
	bb, errB := b.MarshalBinary()
	if errA != nil || errB != nil || !bytes.Equal(ab, bb) {
		t.Errorf("checkpoint bytes differ (%d and %d bytes, errors %v, %v), want equal", len(ab), len(bb), errA, errB)
	}
	if !reflect.DeepEqual(plainRec.events, busyRec.events) {
		t.Errorf("event streams differ (%d and %d events), want equal", len(plainRec.events), len(busyRec.events))
	}
	sa, errA := plain.Snapshot()
	sb, errB := busy.Snapshot()
	if errA != nil || errB != nil || !reflect.DeepEqual(sa, sb) {
		t.Errorf("final snapshots differ (errors %v, %v), want equal", errA, errB)
	}
}

// TestRun_CostIndependentOfAgents verifies that the presets of cost class 1 and 2
// allocate the same at two scales of one configuration: pack one with 200
// households in every role at scale 1 and at scale 4 has four times fewer agents
// for the same households, and the same regions, commodities and roles.
//
// Requirement: NFR-IND-001
func TestRun_CostIndependentOfAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	allocs := func(scale int64) (agents int64, n float64) {
		cfg := loadPack(t, "base.toml", "one.toml", "warm-one.toml", "overlay/h200.toml")
		cfg.World.Scale = scale
		w, rec := newRun(t, cfg, 1, false)
		wins := stepWindows(t, w, rec, 5, 1, nil)
		for _, r := range wins[0].closing.Roles {
			agents += r.Agents
		}
		p, err := indicator.PricesOfWindow(wins[0].w)
		if err != nil {
			t.Fatalf("PricesOfWindow() error = %v, want nil", err)
		}
		b, err := indicator.BaseOf(wins[0].w, p)
		if err != nil {
			t.Fatalf("BaseOf() error = %v, want nil", err)
		}
		return agents, testing.AllocsPerRun(3, func() { presetsOf(t, w.Config(), w.Graph(), b, wins[0]) })
	}
	a1, n1 := allocs(1)
	a4, n4 := allocs(4)
	if a4 >= a1 {
		t.Fatalf("agents at scale 4 = %d, at scale 1 = %d, want fewer: the two worlds are not at different scales", a4, a1)
	}
	if n1 != n4 {
		t.Errorf("allocations of the presets = %v at %d agents and %v at %d agents, want equal: the cost is a function of the configuration", n1, a1, n4, a4)
	}
}
