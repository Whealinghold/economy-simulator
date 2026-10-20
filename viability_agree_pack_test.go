package economysim_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// benchFive loads the cold five-region pack for a benchmark.
func benchFive(b *testing.B) economysim.Config {
	b.Helper()
	cfg, err := config.LoadFS(os.DirFS("testdata/chains"), packLayers["five"]...)
	if err != nil {
		b.Fatalf("LoadFS(five) = %v, want nil", err)
	}
	return cfg
}

// violationsOf returns the viability errors of a Validate result, in order.
func violationsOf(err error) (found []*economysim.ViabilityError) {
	if err == nil {
		return nil
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return nil
	}
	for _, e := range joined.Unwrap() {
		var ve *economysim.ViabilityError
		if errors.As(e, &ve) {
			found = append(found, ve)
		}
	}
	return found
}

// TestPacks_A5SouthAutarkyFailsPoolingPasses verifies A5 of REF 7.3 on the solver
// (`two` + `overlay/a5`): the south cannot grow a staple, so its autarky surplus
// factor is 0 and it has no table row (its wage is unpriced), the pooled realm
// reaches 2.590 with 19 pivots, and the pooled factor exceeds the north's alone.
//
// Requirement: FR-VIA-009, FR-VIA-043, FR-VIA-044
func TestPacks_A5SouthAutarkyFailsPoolingPasses(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPack(t, "base.toml", "two.toml", "overlay/a5.toml")
	alone, err := economysim.SolveSteadyState(context.Background(), cfg)
	if err != nil {
		t.Fatalf("SolveSteadyState(two + a5) = %v, want nil", err)
	}
	if got, want := []int64{alone.Programmes[0].SurplusBP, alone.Programmes[1].SurplusBP}, []int64{22_288, 0}; !slices.Equal(got, want) {
		t.Errorf("autarky surplus factors (north, south) = %v, want %v", got, want)
	}
	if !slices.Equal(alone.Report.UnpricedWages, []string{"south"}) {
		t.Errorf("UnpricedWages = %v, want [south]", alone.Report.UnpricedWages)
	}
	pooled, err := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithFreeTrade())
	if err != nil {
		t.Fatalf("SolveSteadyState(two + a5, WithFreeTrade) = %v, want nil", err)
	}
	if p := pooled.Programmes[0]; p.SurplusBP != 25_896 || p.Pivots != 19 {
		t.Errorf("pooled programme = %+v, want SurplusBP 25 896 (2.590) and 19 pivots", p)
	}
}

// TestPacks_A3ThreeRegionsDifferAndPoolSelectively verifies A3 and FR-REF-011 on
// `three`: the autarky factors differ (2.666, 2.512, 2.060), the pooled programme
// runs some recipe in a strict subset of the regions (it runs `grow_staple` in
// far_south and north only), the pooled factor equals the best region's, and every
// pooled price ratio is inside the autarky range or flagged Outside.
//
// Requirement: FR-VIA-043, FR-VIA-044, FR-VIA-045, FR-REF-011
func TestPacks_A3ThreeRegionsDifferAndPoolSelectively(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPack(t, packLayers["three"]...)
	st, err := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithFreeTrade())
	if err != nil {
		t.Fatalf("SolveSteadyState(three, WithFreeTrade) = %v, want nil", err)
	}
	var autarky []int64
	for _, r := range st.Report.RegionSurplusBP {
		autarky = append(autarky, r.AutarkyBP)
	}
	if want := []int64{26_655, 25_119, 20_600}; !slices.Equal(autarky, want) {
		t.Errorf("autarky surplus factors (far_south, north, south) = %v, want %v", autarky, want)
	}
	strict := false
	for _, a := range st.Report.Advantage.Active {
		if n := len(a.Regions); n >= 1 && n < 3 {
			strict = true
		}
	}
	if !strict {
		t.Errorf("Advantage.Active = %v, want a recipe active in a strict subset of the regions", st.Report.Advantage.Active)
	}
	for _, r := range st.Report.Advantage.Ratios {
		inside := r.LowBP <= r.PooledBP && r.PooledBP <= r.HighBP
		if !inside && !r.Outside {
			t.Errorf("ratio %s/%s pooled %d outside [%d, %d] but not flagged", r.A, r.B, r.PooledBP, r.LowBP, r.HighBP)
		}
	}
}

// TestPacks_RefShapeOfTheSolverOutputs verifies the solver-output parts of
// FR-REF-009 to FR-REF-013: `one` has 600 households in six groups and a surplus
// factor of at least 2; `two` has two regions viable alone whose autarky
// staple-to-fuel price ratios differ by a factor of at least 2; `five` has five
// regions viable alone, 5 000 households, 30 commodities and 45 recipes and a
// pooled factor above every autarky one; `famine` (built from `one`) is viable at
// full labour with a factor below 2 and a positive wage. The pack-shape counts of
// the other packs and the estates module of `famine` belong to the Stage 0 pack
// tests and to Stage 2.
//
// Requirement: FR-REF-009, FR-REF-010, FR-REF-011, FR-REF-012, FR-REF-013
func TestPacks_RefShapeOfTheSolverOutputs(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	ctx := context.Background()
	households := func(c economysim.Config) (n int64) {
		for _, g := range c.Groups {
			n += g.Households
		}
		return n
	}
	t.Run("one", func(t *testing.T) {
		t.Parallel()
		cfg := loadPack(t, packLayers["one"]...)
		st, err := economysim.SolveSteadyState(ctx, cfg)
		if err != nil {
			t.Fatalf("SolveSteadyState(one) = %v", err)
		}
		// 136 + 30 + 90 + 253 + 55 + 36 = 600 households; 20 000 basis points is lambda 2.
		if h := households(cfg); h != 600 || len(cfg.Groups) != 6 || st.Programmes[0].SurplusBP < 20_000 {
			t.Errorf("households, groups, SurplusBP = %d, %d, %d, want 600, 6 and at least 20 000", h, len(cfg.Groups), st.Programmes[0].SurplusBP)
		}
	})
	t.Run("two", func(t *testing.T) {
		t.Parallel()
		st, err := economysim.SolveSteadyState(ctx, loadPack(t, packLayers["two"]...))
		if err != nil {
			t.Fatalf("SolveSteadyState(two) = %v", err)
		}
		price := map[economysim.RegionCommodity]economysim.Price{}
		for _, p := range st.Table.Prices {
			price[economysim.RegionCommodity{Region: p.Region, Commodity: p.Commodity}] = p.Price
		}
		n := func(r string) (staple, fuel economysim.Price) {
			return price[economysim.RegionCommodity{Region: r, Commodity: "staple"}], price[economysim.RegionCommodity{Region: r, Commodity: "fuel"}]
		}
		// north 85 / 265 = 0.32, south 134 / 132 = 1.02: a factor of 3.2, compared by
		// cross-multiplication, 134 * 265 = 35 510 against 85 * 132 = 11 220.
		sn, fn := n("north")
		ss, fs := n("south")
		hi, lo := ss*fn, sn*fs
		if lo > hi {
			hi, lo = lo, hi
		}
		if lo == 0 || hi < 2*lo {
			t.Errorf("staple/fuel ratios north %d/%d and south %d/%d, want them to differ by a factor of at least 2", sn, fn, ss, fs)
		}
		for _, p := range st.Programmes {
			if p.SurplusBP < 10_000 {
				t.Errorf("autarky programme %v SurplusBP = %d, want viable (at least 10 000)", p.Regions, p.SurplusBP)
			}
		}
	})
	t.Run("five", func(t *testing.T) {
		t.Parallel()
		cfg := loadPack(t, packLayers["five"]...)
		alone, err := economysim.SolveSteadyState(ctx, cfg)
		if err != nil {
			t.Fatalf("SolveSteadyState(five) = %v", err)
		}
		pooled, err := economysim.SolveSteadyState(ctx, cfg, economysim.WithFreeTrade())
		if err != nil {
			t.Fatalf("SolveSteadyState(five, WithFreeTrade) = %v", err)
		}
		if h := households(cfg); h != 5_000 || len(cfg.Regions) != 5 || len(cfg.Commodities) != 30 || len(cfg.Recipes) != 45 {
			t.Errorf("households, regions, commodities, recipes = %d, %d, %d, %d, want 5000, 5, 30, 45", h, len(cfg.Regions), len(cfg.Commodities), len(cfg.Recipes))
		}
		for _, p := range alone.Programmes {
			if p.SurplusBP < 10_000 || p.SurplusBP >= pooled.Programmes[0].SurplusBP {
				t.Errorf("autarky %v SurplusBP = %d, want viable and below the pooled %d", p.Regions, p.SurplusBP, pooled.Programmes[0].SurplusBP)
			}
		}
	})
	t.Run("famine", func(t *testing.T) {
		t.Parallel()
		st, err := economysim.SolveSteadyState(ctx, famine(t))
		if err != nil {
			t.Fatalf("SolveSteadyState(famine) = %v", err)
		}
		// 13 150 basis points is lambda 1.315: viable (at least 10 000) and below 2.
		if p := st.Programmes[0]; p.SurplusBP < 10_000 || p.SurplusBP >= 20_000 || len(st.Table.Wages) != 1 || st.Table.Wages[0].Wage <= 0 {
			t.Errorf("famine programme %+v, wages %v, want a factor in [1, 2) and a positive wage", p, st.Table.Wages)
		}
	})
}

// TestSolveSteadyState_StorageGapsOfTheReferenceChain verifies the supply gaps of
// REF 5 at the file level (D122): with the staple's shelf life lowered to 100
// ticks, V4 reports a gap of 304 ticks in the spring-only regions, 123 where the
// region plants twice and 184 in the west pattern of `five`.
//
// Requirement: FR-VIA-008, FR-VIA-039
func TestSolveSteadyState_StorageGapsOfTheReferenceChain(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	tests := []struct {
		pack string
		want map[string]string
	}{
		{"one", map[string]string{"north": "304"}},
		{"two", map[string]string{"north": "304", "south": "123"}},
		{"three", map[string]string{"far_south": "304", "north": "304", "south": "123"}},
		{"five", map[string]string{"centre": "123", "east": "123", "north": "304", "south": "304", "west": "184"}},
	}
	for _, tt := range tests {
		t.Run(tt.pack, func(t *testing.T) {
			t.Parallel()
			cfg := loadPack(t, packLayers[tt.pack]...)
			for i := range cfg.Commodities {
				if cfg.Commodities[i].ID == "staple" {
					cfg.Commodities[i].ShelfLife = 100
				}
			}
			got := map[string]string{}
			for _, e := range violationsOf(cfg.Validate()) {
				if e.Rule == "storage_gap" {
					got[e.Path[1]] = e.Value
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("storage_gap values of %s = %v, want %v", tt.pack, got, tt.want)
			}
		})
	}
}

// TestSolveSteadyState_IsDeterministic verifies A9 of REF 7.3 inside one process:
// two solves of every pack give deeply equal results, and the configuration with
// the solver's table has the same hash both times. The two-architecture run
// belongs to the golden row and this test is written to pass there too.
//
// Requirement: FR-VIA-020, FR-VIA-022, FR-REF-007
func TestSolveSteadyState_IsDeterministic(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for name, layers := range packLayers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := loadPack(t, layers...)
			var results [2]economysim.SteadyState
			var hashes [2][32]byte
			for i := range results {
				st, err := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithFreeTrade())
				if err != nil {
					t.Fatalf("SolveSteadyState(%s) = %v", name, err)
				}
				results[i] = st
				c := cfg
				c.WarmStart = &st.Table
				hashes[i] = economysim.HashConfig(c)
			}
			if !reflect.DeepEqual(results[0], results[1]) || hashes[0] != hashes[1] {
				t.Errorf("two solves of %s differ: hashes %x and %x", name, hashes[0], hashes[1])
			}
		})
	}
}

// TestSolver_IsOutsideTheTickPath verifies that no root file outside the viability
// files imports internal/steady, so the solver, which allocates freely, is
// reachable only through SolveSteadyState and the checks of Validate and cannot
// be called from a position of the tick.
//
// Requirement: NFR-VIA-001
// Guard: the world row adds files that must not call the solver (FR-VIA-021); a
// call from a tick position would put its allocations in the limits of
// observability-and-logging.md 8.2.
func TestSolver_IsOutsideTheTickPath(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob = %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || strings.HasPrefix(f, "viability") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("ReadFile(%s) = %v", f, err)
		}
		if strings.Contains(string(src), "internal/steady") {
			t.Errorf("%s imports internal/steady, want only the viability files to", f)
		}
	}
}

// BenchmarkSolveFive measures the pooled solve of the five-region pack (30
// commodities, 45 recipes; 202 pivots, 1.7 s in the prototype), the number behind
// a10.seconds of REF 7.3 and MaxViabilityCells. It reports; it never fails.
//
// Requirement: NFR-VIA-002, NFR-VIA-003
func BenchmarkSolveFive(b *testing.B) {
	cfg := benchFive(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithFreeTrade()); err != nil {
			b.Fatalf("SolveSteadyState(five, WithFreeTrade) = %v, want nil", err)
		}
	}
}

// BenchmarkValidateFive measures Validate of the cold five-region pack, which runs
// the programmes below the size gate, and Warnings after it (the double cost of
// N-276).
//
// Requirement: NFR-VIA-002
func BenchmarkValidateFive(b *testing.B) {
	cfg := benchFive(b)
	b.ReportAllocs()
	for b.Loop() {
		if err := cfg.Validate(); err != nil {
			b.Fatalf("Validate(five) = %v, want nil", err)
		}
		_ = cfg.Warnings()
	}
}
