package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// colonyRuns returns the seeds and the ticks of a run: four seeds of 400 ticks, and
// under the race detector one seed of 100 (the full set runs without it).
func colonyRuns() (seeds []uint64, ticks int64) {
	if raceEnabled {
		return []uint64{1}, 100
	}
	return []uint64{1, 2, 3, 4}, 400
}

// TestColony_RunsFromTheEmbeddedPack verifies that the embedded pack builds and runs
// with the audit on for every seed: no step fails, the books balance at the end, and
// households remain. The bands of X4 are asserted by the root module, which reads
// bands/second.toml.
//
// Requirement: FR-REF-031, FR-REF-028
func TestColony_RunsFromTheEmbeddedPack(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	seeds, ticks := colonyRuns()
	for _, seed := range seeds {
		t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) {
			t.Parallel()
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit())
			if err != nil {
				t.Fatalf("seed %d: NewWorld() = %v, want nil", seed, err)
			}
			if _, err := w.Run(t.Context(), ticks); err != nil {
				t.Fatalf("seed %d: Run(%d) = %v, want nil", seed, ticks, err)
			}
			if err := w.Audit(); err != nil {
				t.Errorf("seed %d: Audit() at tick %d = %v, want nil", seed, w.Tick(), err)
			}
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() = %v, want nil", seed, err)
			}
			if s.Households <= 0 {
				t.Errorf("seed %d: households at tick %d = %d, want above zero", seed, s.Tick, s.Households)
			}
		})
	}
}

// extraLayer adds a commodity, a recipe that makes it and a role that runs the recipe.
const extraLayer = `
[commodity.zz_good]
reference_price = 100
raw = true
[recipe.make_zz_good]
sited = true
outputs = [ { commodity = "zz_good", quantity = 1, cost_share = 10000 } ]
[role.zz_maker]
recipes = [ "make_zz_good" ]
`

// TestColony_ExtraLayerNeedsOnlyConfiguration verifies that a new commodity, a recipe
// and a role added by one more layer, served beside the embedded pack from an
// in-memory file system, validate, build and run 100 ticks, and the new good has a
// market: the consumer-side twin of the root test of a configuration-only addition.
//
// Requirement: FR-CFG-004
// Guard: architecture.md 9.4. Go code that branched on the colony's identifiers
// would not know the new ones, and the layer would fail to run.
func TestColony_ExtraLayerNeedsOnlyConfiguration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	fsys := fstest.MapFS{"extra.toml": {Data: []byte(extraLayer)}}
	for _, name := range packLayers {
		b, err := fs.ReadFile(packFS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		fsys[name] = &fstest.MapFile{Data: b}
	}
	cfg, err := config.LoadFS(fsys, append(append([]string{}, packLayers...), "extra.toml")...)
	if err != nil {
		t.Fatalf("LoadFS(pack and extra layer) = %v, want nil", err)
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(pack and extra layer) = %v, want nil", err)
	}
	if _, err := w.Run(t.Context(), 100); err != nil {
		t.Fatalf("Run(100) = %v, want nil", err)
	}
	var found bool
	for i, c := range w.Config().Commodities {
		if c.ID != "zz_good" {
			continue
		}
		found = true
		if _, err := w.Market(0, economysim.CommodityID(i)); err != nil {
			t.Errorf("Market(crater_rim, zz_good) = %v, want a market", err)
		}
	}
	if !found {
		t.Error("the new commodity zz_good is not in the world's configuration")
	}
}

// TestRun_PrintsOneLinePerWindow verifies the command's output shape: the tick 0
// line and one line per window, each with the households and every commodity priced
// in the one-denomination preset; and that a bad argument is refused with exit code 2.
//
// Requirement: FR-CUR-018
func TestRun_PrintsOneLinePerWindow(t *testing.T) {
	t.Parallel()
	var out, errs bytes.Buffer
	if code := Run([]string{"-seed", "2", "-ticks", "30", "-every", "10", "-audit"}, &out, &errs); code != 0 {
		t.Fatalf("Run() = %d, stderr %q, want 0", code, errs.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("Run() printed %d lines, want 4 (ticks 0, 10, 20, 30):\n%s", len(lines), out.String())
	}
	for i, l := range lines {
		if want := fmt.Sprintf("tick %d households ", i*10); !strings.HasPrefix(l, want) || !strings.Contains(l, "ice ") || !strings.Contains(l, " cr") {
			t.Errorf("line %d = %q, want it to start %q and price ice in cr", i, l, want)
		}
	}
	for _, args := range [][]string{{"-every", "0"}, {"-ticks", "-1"}, {"-nosuchflag"}} {
		if code := Run(args, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
			t.Errorf("Run(%v) = %d, want 2", args, code)
		}
	}
}

// ExampleRun runs the colony for 10 ticks and prints the first line, the world as it
// is built.
func ExampleRun() {
	var out bytes.Buffer
	Run([]string{"-seed", "1", "-ticks", "10", "-every", "10"}, &out, &bytes.Buffer{})
	first, _, _ := strings.Cut(out.String(), "\n")
	fmt.Println(first)
	// Output: tick 0 households 1000 prices biomass 327 cr, board 12364 cr, drill 14330 cr, fabricator 41155 cr, ice 112 cr, regolith 66 cr, slag 50 cr, solder 2097 cr, wafer 2163 cr, water 346 cr
}
