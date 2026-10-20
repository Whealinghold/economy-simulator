package economysim

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// ratioOne is the full available work share, 1/1.
var ratioOne = steady.Ratio{Num: 1, Den: 1}

// This file tests the entry point of SolveSteadyState on configuration literals
// (plan 1-10 A9): its options, its errors, its purity and the shape of its
// exported surface. The table is viability_table_internal_test.go's, the report
// viability_report_internal_test.go's, the packs viability_solve_pack_test.go's.
// Every case starts from viaToy (viability_internal_test.go): one region of ten
// households, 4 grain a tick each against 1 eaten.

// solveToy solves a configuration and fails the test on an error.
func solveToy(t *testing.T, c Config, opts ...SolveOption) SteadyState {
	t.Helper()
	st, err := SolveSteadyState(context.Background(), c, opts...)
	if err != nil {
		t.Fatalf("SolveSteadyState = %v, want nil", err)
	}
	return st
}

// TestSolveSteadyState_ToyTable verifies the whole table of the smallest chain
// by hand: the programme, the price, the wage, the money, the stock and the
// absence of jobs and pools, and that the table passes Validate.
//
// Requirement: FR-VIA-003, FR-VIA-012, FR-VIA-023, FR-VIA-024, FR-VIA-030, FR-VIA-050, FR-VIA-051
func TestSolveSteadyState_ToyTable(t *testing.T) {
	t.Parallel()
	c := viaToy()
	st := solveToy(t, c)
	// 10 households make 4 grain a tick each and eat 1: lambda 4, SurplusBP 40 000,
	// labour share ceil(10^8 / 40 000) = 2 500, 2 pivots.
	wantProg := []ProgrammeResult{{Regions: []string{"north"}, SurplusBP: 40_000, LabourShareBP: 2_500, HasLabourShare: true, NominalSurplusBP: 27_397, Pivots: 2}}
	if !reflect.DeepEqual(st.Programmes, wantProg) {
		t.Errorf("Programmes = %+v, want %+v (nominal: 4 * 250 / 365 = 2.7397)", st.Programmes, wantProg)
	}
	// The price p of grain has 10 * p = 1 (the normalisation), so p = 0.1; the farm
	// adds 4 p = 0.4 in one tick, so the wage is 0.4. The scale is H * B_ref /
	// (D * p) = 10 * 100 / 1 = 1 000: price 100, wage 400.
	// Money = inputs 0 + U * (d_max + buffer) = 100 * (1 + 10) = 1 100. Stock =
	// cover 5 * 1; the farm runs all year, so there is no supply gap and no reserve.
	want := WarmStart{
		Prices: []SeedPrice{{Region: "north", Commodity: "grain", Price: 100}},
		Wages:  []SeedWage{{Region: "north", Wage: 400}},
		Groups: []WarmGroup{{Region: "north", Role: "farmer", Money: 1_100, Stock: []Item{{Commodity: "grain", Quantity: 5}}}},
	}
	if !reflect.DeepEqual(st.Table, want) {
		t.Errorf("Table = %+v, want %+v", st.Table, want)
	}
	c.WarmStart = &st.Table
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(toy with its table) = %v, want nil", err)
	}
}

// TestSolveSteadyState_StaleTableIsIgnored verifies that the solve validates a
// copy whose WarmStart is nil: a table that Validate would reject does not make
// the solve that replaces it fail, and the result is the cold one.
//
// Requirement: FR-REF-007, FR-VIA-020
func TestSolveSteadyState_StaleTableIsIgnored(t *testing.T) {
	t.Parallel()
	cold := solveToy(t, viaToy())
	c := viaToy()
	c.WarmStart = &WarmStart{Prices: []SeedPrice{{Region: "north", Commodity: "grain", Price: 0}}} // price 0: ErrWarmStartPrice
	if err := c.Validate(); !errors.Is(err, ErrWarmStartPrice) {
		t.Fatalf("Validate(stale) = %v, want ErrWarmStartPrice", err)
	}
	if got := solveToy(t, c); !reflect.DeepEqual(got, cold) {
		t.Errorf("SolveSteadyState(stale table) = %+v, want the cold result %+v", got, cold)
	}
}

// TestSolveSteadyState_Options verifies each option: it changes the table where
// its field enters it, a nil option is ignored, the last of a kind wins, and a
// value outside its bound is ErrOutOfBounds before any work.
//
// Requirement: FR-CFG-047, FR-VIA-023
func TestSolveSteadyState_Options(t *testing.T) {
	t.Parallel()
	moneyOf := func(st SteadyState) Money { return st.Table.Groups[0].Money }
	stockOf := func(st SteadyState) Quantity { return st.Table.Groups[0].Stock[0].Quantity }
	t.Run("stock cover", func(t *testing.T) {
		t.Parallel()
		// Cover 7 ticks of an upkeep of 1: 7 grain; the last of the kind wins.
		got := stockOf(solveToy(t, viaToy(), WithStockCoverTicks(1), nil, WithStockCoverTicks(7)))
		if got != 7 {
			t.Errorf("stock with WithStockCoverTicks(1), nil, (7) = %d, want 7", got)
		}
	})
	t.Run("money buffer", func(t *testing.T) {
		t.Parallel()
		// U * (d_max + buffer) = 100 * (1 + 0) = 100.
		if got := moneyOf(solveToy(t, viaToy(), WithMoneyBufferTicks(0))); got != 100 {
			t.Errorf("money with WithMoneyBufferTicks(0) = %d, want 100", got)
		}
	})
	bounds := []struct {
		name string
		opt  SolveOption
		path string
	}{
		{"stock cover below", WithStockCoverTicks(-1), "stock_cover_ticks"},
		{"stock cover above", WithStockCoverTicks(maxStockCoverTicks + 1), "stock_cover_ticks"},
		{"money buffer below", WithMoneyBufferTicks(-1), "money_buffer_ticks"},
		{"money buffer above", WithMoneyBufferTicks(maxMoneyBufferTicks + 1), "money_buffer_ticks"},
		{"residual below", WithResidualWarnBP(-1), "residual_warn_bp"},
		{"residual above", WithResidualWarnBP(maxResidualWarnBP + 1), "residual_warn_bp"},
		{"pivots below", WithMaxPivots(0), "max_pivots"},
		{"pivots above", WithMaxPivots(MaxViabilityPivots + 1), "max_pivots"},
	}
	for _, tt := range bounds {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st, err := SolveSteadyState(context.Background(), viaToy(), tt.opt)
			var ve *ValidationError
			if !errors.Is(err, ErrOutOfBounds) || !errors.As(err, &ve) || !slices.Equal(ve.Path, []string{"solve_options", tt.path}) {
				t.Errorf("SolveSteadyState(%s) = %v, want ErrOutOfBounds at solve_options/%s", tt.name, err, tt.path)
			}
			if !reflect.DeepEqual(st, SteadyState{}) {
				t.Errorf("SolveSteadyState(%s) returned a result with an error", tt.name)
			}
		})
	}
	edges := []SolveOption{
		WithStockCoverTicks(0), WithStockCoverTicks(maxStockCoverTicks), WithMoneyBufferTicks(maxMoneyBufferTicks),
		WithResidualWarnBP(0), WithResidualWarnBP(maxResidualWarnBP), WithMaxPivots(1_000), WithMaxPivots(MaxViabilityPivots),
	}
	for i, opt := range edges {
		if _, err := newSolveOptions([]SolveOption{opt}); err != nil {
			t.Errorf("newSolveOptions(edge %d) = %v, want nil: a bound is inclusive", i, err)
		}
	}
}

// TestSolveSteadyState_FreeTrade verifies that WithFreeTrade turns two autarky
// programmes into one pooled programme with one scale, and that without it each
// region is solved alone.
//
// Requirement: FR-VIA-043, FR-VIA-044
func TestSolveSteadyState_FreeTrade(t *testing.T) {
	t.Parallel()
	c := viaToy()
	south := DefaultRegion()
	south.ID, south.ReferenceBasket = "south", 100
	c.Regions = append(c.Regions, south)
	c.Groups = append(c.Groups, Group{Region: "south", Role: "farmer", Households: 10})
	alone := solveToy(t, c)
	if len(alone.Programmes) != 2 || !slices.Equal(alone.Programmes[1].Regions, []string{"south"}) {
		t.Errorf("Programmes without WithFreeTrade = %+v, want two, one per region", alone.Programmes)
	}
	pooled := solveToy(t, c, WithFreeTrade())
	if len(pooled.Programmes) != 1 || !slices.Equal(pooled.Programmes[0].Regions, []string{"north", "south"}) {
		t.Fatalf("Programmes with WithFreeTrade = %+v, want one over north and south", pooled.Programmes)
	}
	// Both regions are the same chain: the pooled factor is the autarky one, 4.
	if got := pooled.Programmes[0].SurplusBP; got != 40_000 {
		t.Errorf("pooled SurplusBP = %d, want 40 000", got)
	}
	want := []RegionSurplus{{Region: "north", AutarkyBP: 40_000, PooledBP: 40_000, HasPooled: true}, {Region: "south", AutarkyBP: 40_000, PooledBP: 40_000, HasPooled: true}}
	if !reflect.DeepEqual(pooled.Report.RegionSurplusBP, want) {
		t.Errorf("RegionSurplusBP = %+v, want %+v", pooled.Report.RegionSurplusBP, want)
	}
}

// TestSolveSteadyState_WagesOnePerRegionSorted verifies that the table holds at
// most one SeedWage per region, sorted by region, whatever the order of the
// configuration's regions, autarky or pooled.
//
// Requirement: FR-VIA-049
func TestSolveSteadyState_WagesOnePerRegionSorted(t *testing.T) {
	t.Parallel()
	c := viaToy()
	south := DefaultRegion()
	south.ID, south.ReferenceBasket = "south", 100
	// south first: a table built in input order would list south before north.
	c.Regions = []Region{south, c.Regions[0]}
	c.Groups = []Group{{Region: "south", Role: "farmer", Households: 10}, c.Groups[0]}
	for name, opts := range map[string][]SolveOption{"autarky": nil, "pooled": {WithFreeTrade()}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := solveToy(t, c, opts...).Table.Wages
			regions := make([]string, len(got))
			for i, w := range got {
				regions[i] = w.Region
			}
			if want := []string{"north", "south"}; !slices.Equal(regions, want) {
				t.Errorf("Table.Wages regions = %v, want %v (one per region, sorted)", regions, want)
			}
		})
	}
}

// TestSolveSteadyState_Errors verifies each failure of the call: a
// configuration that is not viable wraps ErrNotViable, a basket-less one is
// ErrNoUpkeep, the pivot cap is ErrSolveLimit with the counts, the cell gate is
// the same error, and a cancelled context is its wrapped error; none returns a
// table.
//
// Requirement: FR-VIA-010, FR-VIA-011, FR-VIA-017, FR-VIA-018, FR-CFG-033
func TestSolveSteadyState_Errors(t *testing.T) {
	t.Parallel()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	notViable := viaToy()
	notViable.Recipes[0].Duration, notViable.Recipes[0].Outputs[0].Quantity = 2, 1 // 0.5 grain a tick against 1 eaten
	noUpkeep := viaToy()
	noUpkeep.World.Upkeep, noUpkeep.Regions[0].ReferenceBasket = nil, 0
	tests := []struct {
		name string
		ctx  context.Context
		cfg  Config
		opts []SolveOption
		want error
	}{
		{"not viable", context.Background(), notViable, nil, ErrNotViable},
		{"no upkeep", context.Background(), noUpkeep, nil, ErrNoUpkeep},
		{"pivot cap", context.Background(), viaToy(), []SolveOption{WithMaxPivots(1)}, ErrSolveLimit},
		{"cancelled", cancelled, viaToy(), nil, context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st, err := SolveSteadyState(tt.ctx, tt.cfg, tt.opts...)
			if !errors.Is(err, tt.want) {
				t.Errorf("SolveSteadyState(%s) = %v, want %v", tt.name, err, tt.want)
			}
			if !reflect.DeepEqual(st, SteadyState{}) {
				t.Errorf("SolveSteadyState(%s) returned a table with its error", tt.name)
			}
		})
	}
	// One pivot of the two the toy needs: the cap is reached with 1 pivot made on a
	// tableau of 2 rows (grain, north's labour) and 4 columns (farm, lambda, two
	// slacks).
	_, err := SolveSteadyState(context.Background(), viaToy(), WithMaxPivots(1))
	var se *SolveError
	if !errors.As(err, &se) || *se != (SolveError{Pivots: 1, Rows: 2, Columns: 4}) {
		t.Errorf("SolveSteadyState(WithMaxPivots(1)) error = %v, want a SolveError of 1 pivot, 2 rows, 4 columns", err)
	}
}

// TestSolveStrict_GateIsErrSolveLimit verifies that a tableau over the cell gate
// is ErrSolveLimit with the size and no pivots, where Validate would skip the
// part with a warning.
//
// Requirement: FR-VIA-010
func TestSolveStrict_GateIsErrSolveLimit(t *testing.T) {
	t.Parallel()
	cc := viaToy().canonical()
	m := newViaModel(&cc)
	m.maxCells = 7 // the toy is 2 rows by 4 columns = 8 cells
	_, err := m.solveStrict(context.Background(), m.model(m.withHouseholds, ratioOne))
	var se *SolveError
	if !errors.As(err, &se) || *se != (SolveError{Rows: 2, Columns: 4}) || !errors.Is(err, ErrSolveLimit) {
		t.Errorf("solveStrict over the gate = %v, want a SolveError of 0 pivots, 2 rows, 4 columns", err)
	}
	m.maxCells = 8
	if _, err := m.solveStrict(context.Background(), m.model(m.withHouseholds, ratioOne)); err != nil {
		t.Errorf("solveStrict at the gate = %v, want nil: the gate is inclusive", err)
	}
}

// TestSolveSteadyState_IsPure verifies that the same call twice gives equal
// results, that cfg is not changed, and that the call does not need the
// configuration to be canonical.
//
// Requirement: FR-VIA-020, FR-VIA-022
func TestSolveSteadyState_IsPure(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Commodities = append(c.Commodities, viaCommodity("water", 100, nil))
	c.World.Upkeep = []Item{{Commodity: "water", Quantity: 1}, {Commodity: "grain", Quantity: 1}} // reversed on purpose
	c.Regions[0].ReferenceBasket = 200
	c.Recipes = append(c.Recipes, viaRecipe("well", nil, []Output{viaOut("water", 4)}, nil))
	c.Roles[0].Recipes = []string{"well", "farm"}
	c.Commodities[0], c.Commodities[1] = c.Commodities[1], c.Commodities[0]
	before := c
	before.Commodities = slices.Clone(c.Commodities)
	first := solveToy(t, c)
	second := solveToy(t, c)
	if !reflect.DeepEqual(first, second) {
		t.Errorf("two solves differ: %+v and %+v", first, second)
	}
	if !reflect.DeepEqual(c.Commodities, before.Commodities) || c.WarmStart != nil {
		t.Errorf("SolveSteadyState changed its argument")
	}
	sorted := c.canonical()
	if got := solveToy(t, sorted); !reflect.DeepEqual(got, first) {
		t.Errorf("solve of the canonical order = %+v, want the same as the given order %+v", got, first)
	}
}

// TestSolveFiles_NoTimeRandomOrLogging verifies by text that the solver's files
// call nothing of time, rand or slog.
//
// Guard: the solver must be a pure function of its arguments (FR-VIA-020). The
// determinism guard bans the calls module-wide; this pins that the solver's own
// files never import the packages, so a later edit cannot reach for them.
func TestSolveFiles_NoTimeRandomOrLogging(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"viability.go", "viability_check.go", "viability_table.go"} {
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			for _, banned := range []string{`"time"`, `"math/rand"`, `"math/rand/v2"`, `"crypto/rand"`, `"log/slog"`, `"log"`, `"os"`} {
				if imp.Path.Value == banned {
					t.Errorf("%s imports %s, want a solver that uses no time, randomness, logging or files", name, banned)
				}
			}
		}
	}
}

// TestSolveOptions_ExactlyFive verifies that the exported With* functions of
// viability.go are the five of D127 and that no Lane type is published.
//
// Guard: WithLanes, Lane and MaxLanes come with the transport module (D127,
// N-279). A lane cannot be expressed in Stage 1, so no rejection code exists; a
// sixth option added here would be one the solver cannot honour.
func TestSolveOptions_ExactlyFive(t *testing.T) {
	t.Parallel()
	f, err := parser.ParseFile(token.NewFileSet(), "viability.go", nil, 0)
	if err != nil {
		t.Fatalf("parse viability.go: %v", err)
	}
	var withs, types []string
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && strings.HasPrefix(d.Name.Name, "With") && d.Name.IsExported() {
				withs = append(withs, d.Name.Name)
			}
		case *ast.GenDecl:
			for _, s := range d.Specs {
				if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.IsExported() {
					types = append(types, ts.Name.Name)
				}
			}
		}
	}
	slices.Sort(withs)
	want := []string{"WithFreeTrade", "WithMaxPivots", "WithMoneyBufferTicks", "WithResidualWarnBP", "WithStockCoverTicks"}
	if !slices.Equal(withs, want) {
		t.Errorf("exported With* functions of viability.go = %v, want %v", withs, want)
	}
	if slices.Contains(types, "Lane") {
		t.Errorf("viability.go publishes the type Lane, want none in Stage 1")
	}
}

// TestMapSolveErr verifies the mapping of the errors of internal/steady to the
// root's sentinels: the pivot cap carries its counts, an empty basket is
// ErrNoUpkeep, an overflow is ErrOverflow, a cancelled context stays itself, a
// failed certificate is ErrSolveInconsistent, and anything else is too, since no
// input can cause it.
//
// Requirement: FR-VIA-010, FR-VIA-014, FR-VIA-017, FR-VIA-018
func TestMapSolveErr(t *testing.T) {
	t.Parallel()
	wrapped := fmt.Errorf("solve cancelled after 3 pivots: %w", context.DeadlineExceeded)
	tests := []struct {
		name string
		in   error
		is   error
	}{
		{"none", nil, nil},
		{"limit", &steady.LimitError{Pivots: 7, Rows: 3, Cols: 9}, ErrSolveLimit},
		{"unbounded", steady.ErrUnbounded, ErrNoUpkeep},
		{"overflow", steady.ErrOverflow, ErrOverflow},
		{"deadline", wrapped, context.DeadlineExceeded},
		{"certificate", steady.ErrInconsistent, ErrSolveInconsistent},
		{"malformed", steady.ErrBadProgramme, ErrSolveInconsistent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mapSolveErr(tt.in)
			if tt.is == nil {
				if got != nil {
					t.Errorf("mapSolveErr(nil) = %v, want nil", got)
				}
				return
			}
			if !errors.Is(got, tt.is) {
				t.Errorf("mapSolveErr(%v) = %v, want %v", tt.in, got, tt.is)
			}
		})
	}
	var se *SolveError
	if err := mapSolveErr(&steady.LimitError{Pivots: 7, Rows: 3, Cols: 9}); !errors.As(err, &se) || *se != (SolveError{Pivots: 7, Rows: 3, Columns: 9}) {
		t.Errorf("mapSolveErr(LimitError) = %v, want a SolveError of 7 pivots, 3 rows, 9 columns", err)
	}
}

// TestSolveSteadyState_TableOverBoundIsAnError verifies that a table that breaks
// a bound is the error of the rule and never returned: 10^9 households each
// holding 1.1 * 10^10 of money is 1.1 * 10^19, above MaxMoney.
//
// Requirement: FR-VIA-030, FR-VIA-032
func TestSolveSteadyState_TableOverBoundIsAnError(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Commodities[0].ReferencePrice = MaxPrice
	c.Regions[0].ReferenceBasket = MaxPrice
	c.Groups[0].Households = MaxHouseholds
	c.Roles[0].StartMoney = 0
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(huge toy) = %v, want nil: only the table is over the bound", err)
	}
	st, err := SolveSteadyState(context.Background(), c)
	if !errors.Is(err, ErrOverflow) || !reflect.DeepEqual(st, SteadyState{}) {
		t.Errorf("SolveSteadyState(huge toy) = %+v, %v, want ErrOverflow and no table", st, err)
	}
}
