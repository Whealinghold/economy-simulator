package economysim

import (
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
)

// TestBeliefParams_FieldByField verifies that the parameters of DefaultWorld
// reach the fifteen fields of beliefs.Params and MaxPrice, that changing one
// World field changes exactly one field of the result (no two swapped), and
// that PriceWindow changes none.
//
// Requirement: FR-BEL-004
//
// Guard: two parameters of the same type are easy to swap in a literal; the
// compiler cannot see it, so each field is moved on its own.
func TestBeliefParams_FieldByField(t *testing.T) {
	t.Parallel()
	d := DefaultWorld()
	base := beliefParams(d)
	// The defaults of BEL 11 as literals, not read back from the World.
	want := beliefs.Params{
		Step: 200, AdjustProb: 5_000, Significance: 500, EvidenceShare: 5_000,
		StockLow: 5_000, StockHigh: 15_000, MaxAge: 3,
		WidthInit: 2_000, WidthMin: 200, WidthMax: 5_000, WidthStep: 250,
		CostFloor: 7_000, CostCeiling: 30_000, RefBandLow: 1_000, RefBandHigh: 100_000,
		MaxPrice: 1_000_000_000,
	}
	if base != want {
		t.Fatalf("beliefParams(DefaultWorld) = %+v, want %+v", base, want)
	}
	fields := []string{
		"BeliefStep", "BeliefAdjustProb", "BeliefSignificance", "BeliefStockLow", "BeliefStockHigh",
		"BeliefMaxAge", "BeliefWidthInit", "BeliefWidthMin", "BeliefWidthMax", "BeliefWidthStep",
		"BeliefEvidenceShare", "CostFloor", "CostCeiling", "RefBandLow", "RefBandHigh",
	}
	for _, f := range fields {
		w := DefaultWorld()
		reflect.ValueOf(&w).Elem().FieldByName(f).SetInt(7_777)
		got, bv := beliefParams(w), reflect.ValueOf(base)
		diff := 0
		for i := range bv.NumField() {
			if reflect.ValueOf(got).Field(i).Int() != bv.Field(i).Int() {
				diff++
			}
		}
		if diff != 1 {
			t.Errorf("changing %s changed %d fields of Params, want exactly 1", f, diff)
		}
	}
	w := DefaultWorld()
	w.PriceWindow = 7_777
	if got := beliefParams(w); got != base {
		t.Errorf("changing PriceWindow changed Params: %+v, want %+v", got, base)
	}
}

// TestBeliefReason_Precedence verifies the reason of one outcome over a Change:
// one case per value and the precedence of spec-beliefs.md 12.2.
//
// Requirement: FR-BEL-030, FR-OBS-006
func TestBeliefReason_Precedence(t *testing.T) {
	t.Parallel()
	bd := beliefs.Bounds{Floor: 100, Ceiling: 200}
	bl := func(c int64) beliefs.Belief { return beliefs.Belief{Centre: c, Width: 1000} }
	tests := []struct {
		name string
		c    beliefs.Change
		want Reason
	}{
		{"gate closed, unchanged", beliefs.Change{Before: bl(150), After: bl(150), Bounds: bd}, ReasonGateClosed},
		{"gate closed on the floor", beliefs.Change{Before: bl(100), After: bl(100), Bounds: bd}, ReasonGateClosed},
		{"gate closed, re-clamped centre", beliefs.Change{Before: bl(90), After: bl(100), Bounds: bd}, ReasonGateClosed},
		{"open, moved inside", beliefs.Change{Before: bl(150), After: bl(160), Bounds: bd, Open: true}, ReasonMoved},
		{"open, unchanged inside", beliefs.Change{Before: bl(150), After: bl(150), Bounds: bd, Open: true}, ReasonHeld},
		{"open, moved onto the floor", beliefs.Change{Before: bl(110), After: bl(100), Bounds: bd, Open: true}, ReasonAtFloor},
		{"open, held on the floor", beliefs.Change{Before: bl(100), After: bl(100), Bounds: bd, Open: true}, ReasonAtFloor},
		{"open, moved onto the ceiling", beliefs.Change{Before: bl(190), After: bl(200), Bounds: bd, Open: true}, ReasonAtCeiling},
		{"floor equals ceiling: floor wins", beliefs.Change{Before: bl(100), After: bl(100), Bounds: beliefs.Bounds{Floor: 100, Ceiling: 100}, Open: true}, ReasonAtFloor},
		// A belief no Update produced (a custom strategy's) still has a reason.
		{"custom belief, moved", beliefs.Change{Before: bl(150), After: beliefs.Belief{Centre: 151, Width: 5}, Bounds: bd, Open: true, Clamped: true}, ReasonMoved},
	}
	for _, tc := range tests {
		if got := beliefReason(tc.c); got != tc.want {
			t.Errorf("%s: beliefReason = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestBeliefsEvent_MapsTally verifies the mapping of a tally to the event.
//
// Requirement: FR-BEL-030
func TestBeliefsEvent_MapsTally(t *testing.T) {
	t.Parallel()
	bd := beliefs.Bounds{Floor: 100, Ceiling: 200, Conflict: true}
	bl := func(c int64) beliefs.Belief { return beliefs.Belief{Centre: c} }
	var tl beliefs.Tally
	// Centres after: 150, 100, 101 -> mean (150+100+101)/3 = 117.
	tl.Add(beliefs.Change{Before: bl(140), After: bl(150), Bounds: bd, Open: true})                // raised
	tl.Add(beliefs.Change{Before: bl(110), After: bl(100), Bounds: bd, Open: true, Clamped: true}) // lowered, floor
	tl.Add(beliefs.Change{Before: bl(101), After: bl(101), Bounds: bd})                            // closed
	got := beliefsEvent(9, 2, 3, &tl)
	want := BeliefsUpdated{
		Tick: 9, Region: 2, Commodity: 3, Outcomes: 3, Adjusted: 2, Raised: 1, Lowered: 1,
		AtFloor: 1, AtCeiling: 0, Conflicts: 3, Clamped: 1,
		MeanCentre: 117, LowCentre: 100, HighCentre: 150,
	}
	if got != want {
		t.Errorf("beliefsEvent = %+v, want %+v", got, want)
	}
}

// TestBeliefTypes_Fields verifies the field types of Belief and Bounds, which
// are a compatibility promise, and that the cast between them and the internal
// structs is exact at the extremes.
//
// Guard: architecture.md 3.3 makes the named type a MUST; a change of Centre to
// int64 would compile at every internal caller and break only the consumers.
func TestBeliefTypes_Fields(t *testing.T) {
	t.Parallel()
	priceT, int64T := reflect.TypeFor[Price](), reflect.TypeFor[int64]()
	for _, c := range []struct {
		typ         reflect.Type
		field       string
		want        reflect.Type
		wantNameFor string
	}{
		{reflect.TypeFor[Belief](), "Centre", priceT, "Price"},
		{reflect.TypeFor[Belief](), "Width", int64T, "int64"},
		{reflect.TypeFor[Bounds](), "Floor", priceT, "Price"},
		{reflect.TypeFor[Bounds](), "Ceiling", priceT, "Price"},
	} {
		f, ok := c.typ.FieldByName(c.field)
		if !ok || f.Type != c.want {
			t.Errorf("%s.%s has type %v, want %s", c.typ.Name(), c.field, f.Type, c.wantNameFor)
		}
	}
	// Every public field must exist, same name and order, in the internal struct
	// (which may carry more: Bounds.Conflict is reported through the event).
	for _, pair := range [][2]reflect.Type{
		{reflect.TypeFor[Belief](), reflect.TypeFor[beliefs.Belief]()},
		{reflect.TypeFor[Bounds](), reflect.TypeFor[beliefs.Bounds]()},
	} {
		if pair[0].NumField() > pair[1].NumField() {
			t.Fatalf("%s has %d fields, internal only %d", pair[0].Name(), pair[0].NumField(), pair[1].NumField())
		}
		for i := range pair[0].NumField() {
			if pair[0].Field(i).Name != pair[1].Field(i).Name {
				t.Errorf("%s field %d is %s, internal %s", pair[0].Name(), i, pair[0].Field(i).Name, pair[1].Field(i).Name)
			}
		}
	}
	// Internal to public, as the root does at the call boundary: values carry over
	// unscaled, so each public field equals the internal one.
	for _, v := range []int64{1, MaxPrice, math.MaxInt64, math.MinInt64} {
		in := beliefs.Belief{Centre: v, Width: v + 1}
		b := Belief{Centre: Price(in.Centre), Width: in.Width}
		if int64(b.Centre) != in.Centre || b.Width != in.Width {
			t.Errorf("Belief from %+v = %+v, want the same values", in, b)
		}
		bi := beliefs.Bounds{Floor: v, Ceiling: v - 1}
		bd := Bounds{Floor: Price(bi.Floor), Ceiling: Price(bi.Ceiling)}
		if int64(bd.Floor) != bi.Floor || int64(bd.Ceiling) != bi.Ceiling {
			t.Errorf("Bounds from %+v = %+v, want the same values", bi, bd)
		}
	}
}
