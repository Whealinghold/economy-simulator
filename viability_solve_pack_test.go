package economysim_test

import (
	"context"
	"os"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// loadWarm loads the layers of a pack with its warm document and returns the
// canonical table of the document, which is not validated: two of the documents
// of the repository are the solver's stale output (D133).
func loadWarm(t *testing.T, layers ...string) economysim.WarmStart {
	t.Helper()
	c := economysim.Canonical(loadPack(t, layers...))
	if c.WarmStart == nil {
		t.Fatalf("%v loaded no warm start", layers)
	}
	return *c.WarmStart
}

// setJob returns the table with the in-progress share of one job row replaced,
// the edit that names a difference from a warm document.
func setJob(ws economysim.WarmStart, region, role, recipe string, bp int64) economysim.WarmStart {
	ws.Groups = slices.Clone(ws.Groups)
	for i, g := range ws.Groups {
		if g.Region != region || g.Role != role {
			continue
		}
		ws.Groups[i].Jobs = slices.Clone(g.Jobs)
		for j, job := range ws.Groups[i].Jobs {
			if job.Recipe == recipe {
				ws.Groups[i].Jobs[j].InProgressBP = bp
			}
		}
	}
	return ws
}

// TestSolveSteadyState_Packs verifies, on the reference packs through the public
// API, the programmes of REF 7.3 (the surplus factors), that the table passes
// Validate, and that it equals the loaded warm document of the pack field for
// field (D133, FR-REF-007). The documents of `two`, `three`, `five` and `second`
// were regenerated in the agreement phase of plan 1-10: the seasonal job rows
// follow spec-viability.md 8.5.1 as written (D134 (9): `two` south 3 110 for the
// 3 341 of the old document, `three` south 3 224 and far_south 3 250 for 3 464 and
// 1 736), the one `grow_algae` row of `second` is 5 232 for 5 163 (D134 (8)), and
// `five` is the whole table with the two free goods lifted into the band (O11).
//
// Requirement: FR-VIA-003, FR-VIA-012, FR-VIA-024, FR-VIA-030, FR-VIA-048, FR-VIA-050, FR-REF-007
func TestSolveSteadyState_Packs(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	type prog struct {
		region    string
		surplusBP int64
	}
	tests := []struct {
		name   string
		layers []string
		warm   []string
		progs  []prog
	}{
		// REF 7.3: one 2.512; the table is the document to the unit.
		{"one", packLayers["one"], []string{"base.toml", "one.toml", "warm-one.toml"}, []prog{{"north", 25_119}}},
		// two 2.229 and 2.216.
		{"two", packLayers["two"], []string{"base.toml", "two.toml", "warm-two.toml"}, []prog{{"north", 22_288}, {"south", 22_159}}},
		// three 2.666, 2.512, 2.060 (far_south, north, south).
		{"three", packLayers["three"], []string{"base.toml", "three.toml", "warm-three.toml"}, []prog{{"far_south", 26_655}, {"north", 25_119}, {"south", 20_600}}},
		// five 1.904, 1.704, 1.915, 1.642, 1.805 (centre, east, north, south, west).
		{"five", packLayers["five"], []string{"base.toml", "lines.toml", "five.toml", "warm-five.toml"},
			[]prog{{"centre", 19_038}, {"east", 17_038}, {"north", 19_145}, {"south", 16_415}, {"west", 18_054}}},
		// second 1.966, wage 1 966 (VIA 15).
		{"second", packLayers["second"], []string{"second/second.toml", "second/warm-second.toml"}, []prog{{"crater_rim", 19_663}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadPack(t, tt.layers...)
			st, err := economysim.SolveSteadyState(context.Background(), cfg)
			if err != nil {
				t.Fatalf("SolveSteadyState(%s) = %v, want nil", tt.name, err)
			}
			var got []prog
			for _, p := range st.Programmes {
				got = append(got, prog{p.Regions[0], p.SurplusBP})
			}
			if !reflect.DeepEqual(got, tt.progs) {
				t.Errorf("surplus factors of %s = %v, want %v", tt.name, got, tt.progs)
			}
			if want := loadWarm(t, tt.warm...); !reflect.DeepEqual(st.Table, want) {
				t.Errorf("table of %s differs from its document\ngot  %+v\nwant %+v", tt.name, st.Table, want)
			}
			cfg.WarmStart = &st.Table
			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate(%s with its table) = %v, want nil", tt.name, err)
			}
			doc := loadPack(t, tt.warm...)
			if err := doc.Validate(); err != nil {
				t.Errorf("Validate(%s with its document) = %v, want nil", tt.name, err)
			}
		})
	}
}

// TestSolveSteadyState_One verifies the numbers the golden header reads for
// `one` (REF 8.5 item 1): SurplusBP 25 119 (lambda 29568/11771), LabourShareBP
// 3 982 (ceil of 10^8 / 25 119, not of the exact lambda) and 10 pivots, nothing
// unpriced, and the balanced census of REF 4.4.1 (each group its own balanced
// figure, 600 households).
//
// Requirement: FR-VIA-012, FR-VIA-019, FR-VIA-028, FR-VIA-048
func TestSolveSteadyState_One(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	st, err := economysim.SolveSteadyState(context.Background(), loadPack(t, packLayers["one"]...))
	if err != nil {
		t.Fatalf("SolveSteadyState(one) = %v, want nil", err)
	}
	want := economysim.ProgrammeResult{Regions: []string{"north"}, SurplusBP: 25_119, LabourShareBP: 3_982, HasLabourShare: true, NominalSurplusBP: 17_205, Pivots: 10}
	if len(st.Programmes) != 1 || !reflect.DeepEqual(st.Programmes[0], want) {
		t.Errorf("Programmes = %+v, want [%+v]", st.Programmes, want)
	}
	if st.Report.UnpricedCommodities != nil || st.Report.UnpricedWages != nil || !st.Report.Degenerate {
		t.Errorf("UnpricedCommodities, UnpricedWages, Degenerate = %v, %v, %v, want none, none, true", st.Report.UnpricedCommodities, st.Report.UnpricedWages, st.Report.Degenerate)
	}
	// Households 136, 30, 90, 253, 55, 36 of the pack, each balanced (VIA 5.2).
	var got []int64
	for _, row := range st.Report.Census {
		if row.Imbalance != 0 || row.Role == "" {
			t.Errorf("Census row %+v, want a group row with no imbalance", row)
		}
		got = append(got, row.Balanced)
	}
	if want := []int64{136, 30, 90, 253, 55, 36}; !slices.Equal(got, want) {
		t.Errorf("balanced households = %v, want %v", got, want)
	}
	// The grower's staple: the lean reserve of 644 (1 380.68 a tick * 118 ticks / 253
	// growers, rounded up) plus the 20 of one `save_seed`, VIA 8.4.1.
	for _, g := range st.Table.Groups {
		for _, it := range g.Stock {
			if g.Role == "grower" && it.Commodity == "staple" && it.Quantity != 664 {
				t.Errorf("grower staple = %d, want 664 = 644 + 20", it.Quantity)
			}
		}
	}
	// The prepared cover of 5 ticks is cut to its 4-tick shelf life (8.4.2).
	if want := []economysim.RegionCommodity{{Region: "north", Commodity: "prepared"}}; !reflect.DeepEqual(st.Report.CapShortened, want) {
		t.Errorf("CapShortened = %v, want %v", st.Report.CapShortened, want)
	}
}

// TestSolveSteadyState_Five verifies the pack with five regions: the surplus
// factors of REF 7.3 (1.915, 1.704, 1.642, 1.805, 1.904), that a solve with the
// loaded `warm-five.toml` gives the same table as the cold pack (the solve
// ignores a loaded table), that the document validates (it was regenerated with
// the two free goods lifted into the reference band, O11, 51 and 52 for the 50 of
// the old document), and that the lifts are listed in Floored and
// UnpricedCommodities.
//
// Requirement: FR-VIA-023, FR-VIA-026 (dual-0 half; non-unique half N-311), FR-VIA-030, FR-REF-007
func TestSolveSteadyState_Five(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cold := loadPack(t, packLayers["five"]...)
	st, err := economysim.SolveSteadyState(context.Background(), cold)
	if err != nil {
		t.Fatalf("SolveSteadyState(five) = %v, want nil", err)
	}
	var got []int64
	for _, p := range st.Programmes {
		got = append(got, p.SurplusBP)
	}
	// centre, east, north, south, west: 1.904, 1.704, 1.915, 1.642, 1.805.
	if want := []int64{19_038, 17_038, 19_145, 16_415, 18_054}; !slices.Equal(got, want) {
		t.Errorf("surplus factors = %v, want %v", got, want)
	}
	warm := loadPack(t, "base.toml", "lines.toml", "five.toml", "warm-five.toml")
	if err := warm.Validate(); err != nil {
		t.Fatalf("Validate(five with warm-five.toml) = %v, want nil", err)
	}
	again, err := economysim.SolveSteadyState(context.Background(), warm)
	if err != nil || !reflect.DeepEqual(again, st) {
		t.Errorf("SolveSteadyState(five with its document) = %v, want the cold result", err)
	}
	for _, rc := range []economysim.RegionCommodity{{Region: "east", Commodity: "semi_a"}, {Region: "east", Commodity: "semi_d"}} {
		if !slices.Contains(st.Report.Floored, rc) || !slices.Contains(st.Report.UnpricedCommodities, rc) {
			t.Errorf("%v is missing from Floored or UnpricedCommodities", rc)
		}
	}
	prices := map[economysim.RegionCommodity]economysim.Price{}
	for _, p := range st.Table.Prices {
		prices[economysim.RegionCommodity{Region: p.Region, Commodity: p.Commodity}] = p.Price
	}
	// The band of ref 510 (semi_a) starts at 510 * 1 000 / 10 000 = 51, and semi_d's at 52.
	if a, d := prices[economysim.RegionCommodity{Region: "east", Commodity: "semi_a"}], prices[economysim.RegionCommodity{Region: "east", Commodity: "semi_d"}]; a != 51 || d != 52 {
		t.Errorf("east semi_a, semi_d = %d, %d, want 51, 52", a, d)
	}
}

// TestSolveSteadyState_FreeTradePivots verifies the pooled programmes of REF 7.3
// and their pivot counts, the exact lists of the first pack that has a pooled
// ratio outside the autarky range, and that every pooled table passes Validate.
//
// Requirement: FR-VIA-043, FR-VIA-044, FR-VIA-045, FR-VIA-048
func TestSolveSteadyState_FreeTradePivots(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	tests := []struct {
		name    string
		layers  []string
		pivots  int64
		pooled  int64
		outside int
	}{
		{"one", packLayers["one"], 10, 25_119, 0},
		{"two", packLayers["two"], 19, 26_264, 1},
		{"three", packLayers["three"], 30, 26_655, 0},
		{"five", packLayers["five"], 202, 20_387, -1},
		{"second", packLayers["second"], 10, 19_663, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadPack(t, tt.layers...)
			st, err := economysim.SolveSteadyState(context.Background(), cfg, economysim.WithFreeTrade())
			if err != nil {
				t.Fatalf("SolveSteadyState(%s, WithFreeTrade) = %v, want nil", tt.name, err)
			}
			if len(st.Programmes) != 1 || st.Programmes[0].Pivots != tt.pivots || st.Programmes[0].SurplusBP != tt.pooled {
				t.Errorf("pooled programme of %s = %+v, want %d pivots and SurplusBP %d", tt.name, st.Programmes, tt.pivots, tt.pooled)
			}
			outside := 0
			for _, r := range st.Report.Advantage.Ratios {
				if r.Outside {
					outside++
				}
			}
			if tt.outside >= 0 && outside != tt.outside {
				t.Errorf("ratios outside the autarky range of %s = %d, want %d", tt.name, outside, tt.outside)
			}
			cfg.WarmStart = &st.Table
			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate(%s with its pooled table) = %v, want nil", tt.name, err)
			}
		})
	}
}

// famine loads the pack of REF 4.4.5, its cold layer set: `one` with poor land
// (the staple yield in spring 3 000), the groups of REF 4.4.5 and the estates
// module on.
func famine(t *testing.T) economysim.Config {
	t.Helper()
	return loadPack(t, packLayers["famine"]...)
}

// TestSolveSteadyState_Famine verifies the nominal fallback on the famine
// chain (spec-viability.md 5.3.4, FR-VIA-058): at full labour the capacity row of
// `grow_staple` is tight and the wage is 0, so the prices and the wage come from
// the nominal programme (wage 1 624, the value of N-085), the region is listed in
// NominalSeed, the labour share is absent, and the flows (the reserve 393 = 383 +
// 10, the jobs) are those of the full-labour programme at lambda 1.
//
// Requirement: FR-VIA-012, FR-VIA-058, FR-VIA-028
func TestSolveSteadyState_Famine(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	st, err := economysim.SolveSteadyState(context.Background(), famine(t))
	if err != nil {
		t.Fatalf("SolveSteadyState(famine) = %v, want nil", err)
	}
	// lambda 96/73 = 1.31506 at full labour: SurplusBP 13 150; the nominal 1.11247.
	p := st.Programmes[0]
	if p.SurplusBP != 13_150 || p.HasLabourShare || p.NominalSurplusBP != 11_124 || p.Pivots != 9 {
		t.Errorf("Programme = %+v, want SurplusBP 13 150, no labour share, nominal 11 124 and 9 pivots", p)
	}
	if !slices.Equal(st.Report.NominalSeed, []string{"north"}) {
		t.Errorf("NominalSeed = %v, want [north]", st.Report.NominalSeed)
	}
	if want := []economysim.SeedWage{{Region: "north", Wage: 1_624}}; !reflect.DeepEqual(st.Table.Wages, want) {
		t.Errorf("Wages = %v, want %v", st.Table.Wages, want)
	}
	doc := loadWarm(t, "base.toml", "famine.toml", "overlay/estates.toml", "warm-famine.toml")
	// The document holds the maker's two job rows unscaled, 2 272 + 8 522 = 10 794
	// (Validate rejects shares above 10 000); the solver scales them down in
	// proportion, each rounded down: 2 272 * 10 000 / 10 794 = 2 104 and
	// 8 522 * 10 000 / 10 794 = 7 895 (D121). Every other row is the document's.
	want := setJob(setJob(doc, "north", "maker", "make_tool_a", 2_104), "north", "maker", "make_tool_b", 7_895)
	if !reflect.DeepEqual(st.Table, want) {
		t.Errorf("table of famine differs from warm-famine.toml apart from the maker's jobs\ngot  %+v\nwant %+v", st.Table, want)
	}
	for _, row := range st.Report.Census {
		if row.Role == "maker" && !row.Short {
			t.Errorf("Census row of the maker = %+v, want Short: its shares were scaled down", row)
		}
	}
	if want := []economysim.RegionRecipe{{Region: "north", Recipe: "grow_staple"}}; !reflect.DeepEqual(st.Report.CapacityTight, want) {
		t.Errorf("CapacityTight = %v, want %v", st.Report.CapacityTight, want)
	}
}

// TestSolveSteadyState_OverlaysSolve verifies that every layered file under
// overlay/ that validates on `one` solves, and that the two that do not (they
// name the estates module, or carry a warm start of their own) are the two
// named: the loops that skip a pack that stops validating cannot hide a pack the
// solver rejects, and a third file that stopped validating is not skipped
// silently.
//
// Guard: role_pack_test.go and welfare_pack_test.go skip a pack whose Validate
// fails; a change that made SolveSteadyState reject a valid pack, or an overlay
// stop validating, would not be seen there.
func TestSolveSteadyState_OverlaysSolve(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	notOnOne := []string{"overhang.toml"}
	entries, err := os.ReadDir("testdata/chains/overlay")
	if err != nil {
		t.Fatalf("ReadDir(overlay) = %v", err)
	}
	for _, e := range entries {
		t.Run(e.Name(), func(t *testing.T) {
			t.Parallel()
			cfg, err := config.LoadFS(os.DirFS("testdata/chains"), "base.toml", "one.toml", "overlay/"+e.Name())
			if err == nil {
				err = cfg.Validate()
			}
			if slices.Contains(notOnOne, e.Name()) {
				if err == nil {
					t.Errorf("overlay %s validates on `one`, want it in the solver's list of the ones that do not", e.Name())
				}
				return
			}
			if err != nil {
				t.Fatalf("overlay %s on `one` = %v, want it to load and validate", e.Name(), err)
			}
			if _, err := economysim.SolveSteadyState(context.Background(), cfg); err != nil {
				t.Errorf("SolveSteadyState(one with overlay %s) = %v, want nil: it validates", e.Name(), err)
			}
		})
	}
}

// TestSolveSteadyState_SecondMatchesVIA15 verifies the digits of chain B in
// spec-viability.md 15 (rescaled, one region, scaled to its basket): the wage
// 1 966 and the nine seed prices, that the seed sits at its full unit cost (no
// seed outside the cost band) and that the value added at the integer prices is
// within 1% of the wage times the duration (FR-VIA-054: the prototype gives 0.991
// to 1.000).
//
// Requirement: FR-VIA-027, FR-VIA-048, FR-VIA-054
func TestSolveSteadyState_SecondMatchesVIA15(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	st, err := economysim.SolveSteadyState(context.Background(), loadPack(t, packLayers["second"]...))
	if err != nil {
		t.Fatalf("SolveSteadyState(second) = %v, want nil", err)
	}
	if want := []economysim.SeedWage{{Region: "crater_rim", Wage: 1_966}}; !reflect.DeepEqual(st.Table.Wages, want) {
		t.Errorf("Wages = %v, want %v", st.Table.Wages, want)
	}
	want := map[string]economysim.Price{
		"biomass": 327, "board": 12_364, "drill": 14_330, "fabricator": 41_155, "ice": 112,
		"regolith": 66, "solder": 2_097, "wafer": 2_163, "water": 346,
	}
	got := map[string]economysim.Price{}
	for _, p := range st.Table.Prices {
		got[p.Commodity] = p.Price
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("seed prices = %v, want %v", got, want)
	}
	if r := st.Report; r.SeedBelowCostFloor != nil || r.SeedAboveCostCeiling != nil || r.ResidualBP > 100 {
		t.Errorf("below, above, residual = %v, %v, %d, want none, none and at most 100 basis points", r.SeedBelowCostFloor, r.SeedAboveCostCeiling, r.ResidualBP)
	}
}
