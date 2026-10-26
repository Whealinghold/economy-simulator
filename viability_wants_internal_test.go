package economysim

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// viaWanting returns viaToy with the demand module on: the farmers want q units
// of grain every period ticks, so ten households ask for 10 * q / period a tick
// at the reference. viaToy makes 40 grain a tick at full labour and the basket
// takes 10 a tick.
func viaWanting(q, period int64) Config {
	c := viaToy()
	w := DefaultWant()
	w.ID, w.Quantity, w.Period = "treat", q, period
	w.Items = []WantItem{DefaultWantItem()}
	w.Items[0].Commodity = "grain"
	ta := DefaultTaste()
	ta.Role, ta.Want = "farmer", "treat"
	c.Modules.Demand = &Demand{Wants: []Want{w}, Tastes: []Taste{ta}}
	return c
}

// viaWantsExceed runs the check wants_exceed_labour on a model of c with the
// size gate maxCells (0 keeps the default) and returns its outcome.
func viaWantsExceed(c Config, maxCells int64) viabilityOutcome {
	cc := c.canonical()
	m := newViaModel(&cc)
	if maxCells > 0 {
		m.maxCells = maxCells
	}
	var o viabilityOutcome
	m.wantsExceed(&o)
	return o
}

// TestWantsTerm_ExactFractions verifies that the fixed term is the exact
// fraction Households * taste * region * quantity / (10^8 * period) split by the
// item weights, not a rounded integer, and that it is nil with the module off.
//
// Requirement: FR-VIA-055
func TestWantsTerm_ExactFractions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  func() Config
		want []steady.Ratio
	}{
		{"module off is nil", viaToy, nil},
		// 10 households * 1 / 3 = 10/3 a tick, not 3.
		{"period 3", func() Config { return viaWanting(1, 3) }, []steady.Ratio{{Num: 10, Den: 3}}},
		{"quantity over period", func() Config { return viaWanting(2, 1) }, []steady.Ratio{{Num: 20, Den: 1}}},
		{"half taste", func() Config {
			c := viaWanting(2, 1)
			c.Modules.Demand.Tastes[0].Scale = 5_000
			return c
		}, []steady.Ratio{{Num: 10, Den: 1}}},
		{"zero taste removes the want", func() Config {
			c := viaWanting(2, 1)
			c.Modules.Demand.Tastes[0].Scale = 0
			return c
		}, []steady.Ratio{{Num: 0, Den: 1}}},
		{"region scale", func() Config {
			c := viaWanting(2, 1)
			c.Modules.Demand.RegionTastes = []RegionTaste{{Region: "north", Want: "treat", Scale: 15_000}}
			return c
		}, []steady.Ratio{{Num: 30, Den: 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.cfg()
			cc := c.canonical()
			got, err := wantsTerm(&cc, []RegionID{0})
			if err != nil {
				t.Fatalf("wantsTerm error = %v, want nil", err)
			}
			if len(got) != len(tt.want) || (len(got) == 1 && got[0] != tt.want[0]) {
				t.Errorf("wantsTerm = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSolveSteadyState_WithWants verifies the surplus factor with and without
// the term: viaToy has lambda* 4 alone; ten households wanting 20 grain a tick
// leave 10 lambda <= 40 - 20, so lambda* 2 and SurplusWantsBP 20 000; a term of
// 10/3 gives (40 - 10/3) / 10 = 11/3, floor 36 666; wants above the whole output
// are infeasible, fall back to the upkeep-only programme and leave
// SurplusWantsBP absent (FR-VIA-057).
//
// Requirement: FR-VIA-055, FR-VIA-056
func TestSolveSteadyState_WithWants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfg      Config
		surplus  int64
		wantsBP  int64
		hasWants bool
	}{
		{"module off", viaToy(), 40_000, 0, false},
		{"zero term unchanged", func() Config { c := viaWanting(2, 1); c.Modules.Demand.Tastes[0].Scale = 0; return c }(), 40_000, 40_000, true},
		{"wants of 20", viaWanting(2, 1), 20_000, 20_000, true},
		{"fractional term", viaWanting(1, 3), 36_666, 36_666, true},
		{"infeasible falls back", viaWanting(15, 1), 40_000, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st, err := SolveSteadyState(context.Background(), tt.cfg)
			if err != nil {
				t.Fatalf("SolveSteadyState error = %v, want nil", err)
			}
			if got := st.Programmes[0].SurplusBP; got != tt.surplus {
				t.Errorf("Programmes[0].SurplusBP = %d, want %d", got, tt.surplus)
			}
			if r := st.Report; r.HasSurplusWants != tt.hasWants || r.SurplusWantsBP != tt.wantsBP {
				t.Errorf("Report.SurplusWantsBP = %d (has %v), want %d (has %v)", r.SurplusWantsBP, r.HasSurplusWants, tt.wantsBP, tt.hasWants)
			}
		})
	}
}

// TestLabourShort_IgnoresWants verifies that a pack is not rejected for wants it
// cannot afford: with wants above the whole output Validate passes (no
// labour_short) and only warns wants_exceed_labour.
//
// Requirement: FR-VIA-056, FR-VIA-057
func TestLabourShort_IgnoresWants(t *testing.T) {
	t.Parallel()
	errs, warns := viaRun(t, viaWanting(15, 1))
	if len(errs) != 0 {
		t.Errorf("Validate(wants above the output) viability errors = %v, want none", errs)
	}
	if !viaWarned(warns, "wants_exceed_labour", "regions") {
		t.Errorf("Warnings = %v, want wants_exceed_labour at regions", warns)
	}
}

// TestWantsExceedLabour_Warning verifies the warning's conditions: lambda* below
// 1 with the wants and 1 or more without; the same for an infeasible programme;
// no warning when the wants fit, when the module is off, or when the upkeep-only
// programme is already below 1; and viability_check_skipped past the size gate.
// Wants of 35 leave (40 - 35) / 10 = 0.5; upkeep 5 a household leaves 40 / 50.
//
// Requirement: FR-VIA-057, FR-CFG-065
func TestWantsExceedLabour_Warning(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfg      Config
		maxCells int64
		rule     string
		fires    bool
	}{
		{"surplus below 1", viaWanting(7, 2), 0, "wants_exceed_labour", true},
		{"infeasible", viaWanting(15, 1), 0, "wants_exceed_labour", true},
		{"wants fit", viaWanting(2, 1), 0, "wants_exceed_labour", false},
		{"module off", viaToy(), 0, "wants_exceed_labour", false},
		{"upkeep only already below 1", func() Config {
			c := viaWanting(7, 2)
			c.World.Upkeep = []Item{{Commodity: "grain", Quantity: 5}}
			return c
		}(), 0, "wants_exceed_labour", false},
		{"size gate", viaWanting(7, 2), 1, "viability_check_skipped", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o := viaWantsExceed(tt.cfg, tt.maxCells)
			got := false
			for _, w := range o.warnings {
				got = got || w.Rule == tt.rule
			}
			if got != tt.fires || len(o.errs) != 0 {
				t.Errorf("wantsExceed warnings = %v, errors = %v, want %s fires %v and no error", o.warnings, o.errs, tt.rule, tt.fires)
			}
		})
	}
}

// TestWantsModel_OverflowIsErrOverflow verifies that a wants term whose sum of
// fractions does not fit int64 is ErrOverflow from the model: four wants with
// coprime periods near 10^5 have a common denominator near 10^20.
//
// Requirement: FR-VIA-055
func TestWantsModel_OverflowIsErrOverflow(t *testing.T) {
	t.Parallel()
	c := viaWanting(1, 99_991)
	for i, period := range []int64{99_989, 99_971, 99_961} {
		w := c.Modules.Demand.Wants[0]
		w.ID, w.Period = "treat"+string(rune('b'+i)), period
		c.Modules.Demand.Wants = append(c.Modules.Demand.Wants, w)
		c.Modules.Demand.Tastes = append(c.Modules.Demand.Tastes, Taste{Role: "farmer", Want: w.ID, Scale: 10_000})
	}
	c.Groups[0].Households = 7
	cc := c.canonical()
	m := newViaModel(&cc)
	if _, err := m.wantsModel([]RegionID{0}, steady.Ratio{Num: 1, Den: 1}); !errors.Is(err, ErrOverflow) {
		t.Errorf("wantsModel error = %v, want ErrOverflow", err)
	}
}
