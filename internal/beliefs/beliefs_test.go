package beliefs_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
)

const maxPrice = 1_000_000_000

// defaults are the BEL 11 defaults the worked example uses: reference band
// 1 000 to 100 000 basis points, cost band 7 000 to 30 000, width 500 to 5 000
// (init 2 000, below 10 000).
func defaults() beliefs.Params {
	return beliefs.Params{
		WidthInit: 2000, WidthMin: 500, WidthMax: 5000, WidthStep: 250,
		CostFloor: 7000, CostCeiling: 30000,
		RefBandLow: 1000, RefBandHigh: 100000,
		MaxPrice: maxPrice,
	}
}

// TestLimits_Bounds verifies the reference band, the cost band, the cost
// floor's precedence and the degenerate inputs.
//
// Requirement: FR-BEL-004, FR-BEL-005, FR-BEL-006
func TestLimits_Bounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		ref, cost int64
		want      beliefs.Bounds
	}{
		// BEL 15: refLow = 1 000 * 1 000 / 10 000 = 100, refHigh = 1 000 *
		// 100 000 / 10 000 = 10 000; no cost.
		{name: "no cost", ref: 1000, cost: 0, want: beliefs.Bounds{Floor: 100, Ceiling: 10000}},
		// costLow = 1 600 * 7 000 / 10 000 = 1 120; costHigh = 1 600 * 30 000
		// / 10 000 = 4 800; Floor = max(100, 1 120), Ceiling = max(1 120, min(10 000, 4 800)).
		{name: "cost band", ref: 1000, cost: 1600, want: beliefs.Bounds{Floor: 1120, Ceiling: 4800}},
		// costLow = 200 000 * 0.7 = 140 000 > refHigh 10 000: the floor wins,
		// Ceiling = max(140 000, min(10 000, 600 000)) = 140 000.
		{name: "cost floor above band wins", ref: 1000, cost: 200000, want: beliefs.Bounds{Floor: 140000, Ceiling: 140000, Conflict: true}},
		// ref 1: refLow = max(1, 1 * 1 000 / 10 000 = 0) = 1; refHigh = 10.
		{name: "ref 1", ref: 1, cost: 0, want: beliefs.Bounds{Floor: 1, Ceiling: 10}},
		// A9: ref 0 and negative read as 1, a negative cost as 0.
		{name: "ref 0 is 1", ref: 0, cost: 0, want: beliefs.Bounds{Floor: 1, Ceiling: 10}},
		{name: "negative ref is 1", ref: -5, cost: -3, want: beliefs.Bounds{Floor: 1, Ceiling: 10}},
		// A6: cost 10: costLow 7, costHigh 30; Floor = max(100, 7) = 100,
		// Ceiling = max(100, min(10 000, 30)) = 100; no conflict (7 < 10 000).
		{name: "floor equals ceiling without conflict", ref: 1000, cost: 10, want: beliefs.Bounds{Floor: 100, Ceiling: 100}},
		// cost 2e9 * 7 000 / 10 000 = 1.4e9 > MaxPrice 1e9: Floor capped.
		{name: "max price caps floor", ref: 1000, cost: 2_000_000_000, want: beliefs.Bounds{Floor: maxPrice, Ceiling: maxPrice, Conflict: true}},
		// MaxInt64 * 7 000 / 10 000 fits (it is below MaxInt64) but exceeds
		// MaxPrice; MaxInt64 * 30 000 / 10 000 = 3 * MaxInt64 overflows and
		// reads as MaxPrice.
		{name: "overflow reads max price", ref: 1000, cost: math.MaxInt64, want: beliefs.Bounds{Floor: maxPrice, Ceiling: maxPrice, Conflict: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := beliefs.Limits(tt.ref, tt.cost, defaults()); got != tt.want {
				t.Errorf("Limits(%d, %d) = %+v, want %+v", tt.ref, tt.cost, got, tt.want)
			}
		})
	}
}

// TestLimits_FloorOverflowReadsMaxPrice verifies that a unit cost whose
// product with CostFloor overflows int64 gives a floor of MaxPrice and no
// panic (BEL 3.2.5).
//
// Requirement: FR-BEL-006
func TestLimits_FloorOverflowReadsMaxPrice(t *testing.T) {
	t.Parallel()
	p := defaults()
	p.CostFloor = 1_000_000 // MaxBasisPoints: MaxInt64 * 100 overflows
	got := beliefs.Limits(1000, math.MaxInt64, p)
	if got.Floor != maxPrice || got.Ceiling != maxPrice {
		t.Errorf("Limits(1000, MaxInt64) = %+v, want floor and ceiling %d", got, maxPrice)
	}
}

// TestClamp verifies that Clamp moves the centre and the width into their
// limits, reports a change, and leaves an inside belief alone.
//
// Requirement: FR-BEL-002, FR-BEL-026
func TestClamp(t *testing.T) {
	t.Parallel()
	bd := beliefs.Bounds{Floor: 100, Ceiling: 500}
	tests := []struct {
		name string
		in   beliefs.Belief
		want beliefs.Belief
		chg  bool
	}{
		{"inside", beliefs.Belief{Centre: 300, Width: 2000}, beliefs.Belief{Centre: 300, Width: 2000}, false},
		{"on the bounds", beliefs.Belief{Centre: 100, Width: 500}, beliefs.Belief{Centre: 100, Width: 500}, false},
		{"centre below", beliefs.Belief{Centre: 99, Width: 2000}, beliefs.Belief{Centre: 100, Width: 2000}, true},
		{"centre above", beliefs.Belief{Centre: 501, Width: 2000}, beliefs.Belief{Centre: 500, Width: 2000}, true},
		{"width below", beliefs.Belief{Centre: 300, Width: 499}, beliefs.Belief{Centre: 300, Width: 500}, true},
		{"width above", beliefs.Belief{Centre: 300, Width: 5001}, beliefs.Belief{Centre: 300, Width: 5000}, true},
		{"both", beliefs.Belief{Centre: 9, Width: 99999}, beliefs.Belief{Centre: 100, Width: 5000}, true},
		{"int64 extremes low", beliefs.Belief{Centre: math.MinInt64, Width: math.MinInt64}, beliefs.Belief{Centre: 100, Width: 500}, true},
		{"int64 extremes high", beliefs.Belief{Centre: math.MaxInt64, Width: math.MaxInt64}, beliefs.Belief{Centre: 500, Width: 5000}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, chg := beliefs.Clamp(tt.in, bd, defaults())
			if got != tt.want || chg != tt.chg {
				t.Errorf("Clamp(%+v) = %+v, %v, want %+v, %v", tt.in, got, chg, tt.want, tt.chg)
			}
		})
	}
}

// TestDraw_Range verifies the range of BEL 15 case A, the two ends of the
// word, a one-price range, and the re-clamp of a stored belief.
//
// Requirement: FR-BEL-002, FR-BEL-003, FR-BEL-019
func TestDraw_Range(t *testing.T) {
	t.Parallel()
	wide := beliefs.Bounds{Floor: 100, Ceiling: 10000}
	tests := []struct {
		name string
		b    beliefs.Belief
		bd   beliefs.Bounds
		u    uint64
		want int64
	}{
		// half = 1 030 * 2 250 / 10 000 = 231 (231.75 truncated); Low = 799,
		// High = 1 261.
		{"u zero gives low", beliefs.Belief{Centre: 1030, Width: 2250}, wide, 0, 799},
		{"u max gives high", beliefs.Belief{Centre: 1030, Width: 2250}, wide, math.MaxUint64, 1261},
		// n = 463; u = 2^63 gives hi = 463/2 = 231; 799 + 231 = 1 030.
		{"u half gives middle", beliefs.Belief{Centre: 1030, Width: 2250}, wide, 1 << 63, 1030},
		// Floor = Ceiling = 100: n = 1, hi = 0 whatever u is.
		{"one price", beliefs.Belief{Centre: 100, Width: 2000}, beliefs.Bounds{Floor: 100, Ceiling: 100}, math.MaxUint64, 100},
		// Centre 50 re-clamped to 100; width 2 000: half = 20; Low =
		// max(100, 80) = 100.
		{"centre below floor", beliefs.Belief{Centre: 50, Width: 2000}, wide, 0, 100},
		// Centre 20 000 re-clamped to 10 000; half = 2 000; High =
		// min(10 000, 12 000) = 10 000.
		{"centre above ceiling", beliefs.Belief{Centre: 20000, Width: 2000}, wide, math.MaxUint64, 10000},
		// Width -7 re-clamped to 500: half = 1 000 * 500 / 10 000 = 50;
		// Low = 950.
		{"negative width", beliefs.Belief{Centre: 1000, Width: -7}, wide, 0, 950},
		// Width 20 000 re-clamped to 5 000: half = 500; High = 1 500.
		{"width above max", beliefs.Belief{Centre: 1000, Width: 20000}, wide, math.MaxUint64, 1500},
		// Width 9 999 is above WidthMax 5 000 too: half = 500, Low = 500.
		{"width 9999", beliefs.Belief{Centre: 1000, Width: 9999}, wide, 0, 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := beliefs.Draw(tt.b, tt.bd, defaults(), tt.u); got != tt.want {
				t.Errorf("Draw(%+v, %+v, u=%d) = %d, want %d", tt.b, tt.bd, tt.u, got, tt.want)
			}
		})
	}
}

// TestDraw_PropertyInRange verifies over a fixed seed range that a drawn
// price lies in [Floor, Ceiling], is at least 1 and is deterministic, for
// beliefs anywhere, including far outside their bounds.
//
// Requirement: FR-BEL-003, FR-BEL-019
func TestDraw_PropertyInRange(t *testing.T) {
	t.Parallel()
	p := defaults()
	// splitmix64 gives a spread of words and inputs from the loop seed.
	next := func(s *uint64) uint64 {
		*s += 0x9e3779b97f4a7c15
		z := *s
		z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
		z = (z ^ z>>27) * 0x94d049bb133111eb
		return z ^ z>>31
	}
	for seed := uint64(1); seed <= 5000; seed++ {
		s := seed
		ref := int64(next(&s)%2_000_000) + 1
		cost := int64(next(&s) % 3_000_000)
		bd := beliefs.Limits(ref, cost, p)
		b := beliefs.Belief{Centre: int64(next(&s)), Width: int64(next(&s))}
		u := next(&s)
		got := beliefs.Draw(b, bd, p, u)
		if got < bd.Floor || got > bd.Ceiling || got < 1 {
			t.Fatalf("seed %d: Draw(%+v, %+v) = %d, outside [%d, %d]", seed, b, bd, got, bd.Floor, bd.Ceiling)
		}
		if again := beliefs.Draw(b, bd, p, u); again != got {
			t.Fatalf("seed %d: Draw not deterministic: %d then %d", seed, got, again)
		}
	}
}

// TestPackage_NoMapFuncFloatAndImports verifies that the package has no
// struct field of map, func or float type, no float identifier at all, and
// imports only internal/checked and the standard library.
//
// Requirement: FR-BEL-017
//
// Guard: the rule must stay a pure integer function; a map, a function value
// or a float would break determinism (architecture.md 3.5, 7.5).
func TestPackage_NoMapFuncFloatAndImports(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", name, err)
		}
		for _, im := range f.Imports {
			path, _ := strconv.Unquote(im.Path.Value)
			if strings.Contains(path, ".") && path != "github.com/Whealinghold/economy-sim/internal/checked" {
				t.Errorf("%s imports %q, want internal/checked and the standard library only", name, path)
			}
			if path == "math/rand" || path == "math/rand/v2" || path == "time" {
				t.Errorf("%s imports %q, want no random source or clock", name, path)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.Ident:
				if x.Name == "float32" || x.Name == "float64" {
					t.Errorf("%s: float type %s at %v", name, x.Name, fset.Position(x.Pos()))
				}
			case *ast.StructType:
				for _, fl := range x.Fields.List {
					switch fl.Type.(type) {
					case *ast.MapType, *ast.FuncType:
						t.Errorf("%s: struct field of map or func type at %v", name, fset.Position(fl.Pos()))
					}
				}
			}
			return true
		})
	}
}

// TestLimitsClampDraw_NoAllocs verifies that the per-outcome path allocates
// nothing.
//
// Requirement: NFR-BEL-001
func TestLimitsClampDraw_NoAllocs(t *testing.T) {
	p := defaults()
	bd := beliefs.Limits(1000, 1600, p)
	b := beliefs.Belief{Centre: 2000, Width: 2000}
	var sink int64
	tests := []struct {
		name string
		fn   func()
	}{
		{"Limits", func() { sink += beliefs.Limits(1000, 1600, p).Floor }},
		{"Clamp", func() { c, _ := beliefs.Clamp(b, bd, p); sink += c.Centre }},
		{"Draw", func() { sink += beliefs.Draw(b, bd, p, 12345) }},
	}
	for _, tt := range tests {
		if got := testing.AllocsPerRun(100, tt.fn); got != 0 {
			t.Errorf("%s allocs = %v, want 0", tt.name, got)
		}
	}
	_ = sink
}
