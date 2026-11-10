package verdict

import "testing"

// TestCmpMul_Exact verifies that the 128-bit comparison of two products is
// exact, including signs, zero and the extremes of int64, where a plain
// multiply would wrap.
//
// Guard: the detectors compare series values with basis-point factors by
// cmpMul; a wrap would turn a regime into none (architecture.md 3.5).
func TestCmpMul_Exact(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a, b, c, d int64
		want       int
	}{
		{2, 3, 6, 1, 0},
		{2, 3, 7, 1, -1},
		{4, 3, 7, 1, 1},
		{-2, 3, 5, -1, -1}, // -6 < -5
		{-2, 3, 2, 1, -1},
		{2, 3, -7, 1, 1},
		{0, minInt64, 0, 5, 0},
		{0, 5, -1, 1, 1},
		{minInt64, 2, maxInt64, 2, -1},
		{maxInt64, maxInt64, maxInt64, maxInt64, 0},
		{maxInt64, maxInt64, maxInt64, maxInt64 - 1, 1},
		{minInt64, minInt64, maxInt64, maxInt64, 1}, // 2^126 > (2^63-1)^2
		{minInt64, maxInt64, minInt64, maxInt64 - 1, -1},
	} {
		if got := cmpMul(tc.a, tc.b, tc.c, tc.d); got != tc.want {
			t.Errorf("cmpMul(%d, %d, %d, %d) = %d, want %d", tc.a, tc.b, tc.c, tc.d, got, tc.want)
		}
	}
}

// TestMulDivFloor_Range verifies the floor quotient and its refusals.
func TestMulDivFloor_Range(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a, b, d int64
		want    int64
		ok      bool
	}{
		{1000, 9000, 10000, 900, true},
		{7, 3, 2, 10, true},
		{-1, 3, 2, 0, false},
		{1, 3, 0, 0, false},
		{maxInt64, maxInt64, 1, 0, false},
		{maxInt64, 2, 2, maxInt64, true},
	} {
		if got, ok := mulDivFloor(tc.a, tc.b, tc.d); ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("mulDivFloor(%d, %d, %d) = %d, %v, want %d, %v", tc.a, tc.b, tc.d, got, ok, tc.want, tc.ok)
		}
	}
}

// TestMidpoints verifies the overflow-free midpoint and mean helpers on
// small values against the plain formula, and at the extremes.
func TestMidpoints(t *testing.T) {
	t.Parallel()
	for a := int64(-5); a <= 5; a++ {
		for b := int64(-5); b <= 5; b++ {
			// floor((a+b+1)/2) by arithmetic shift of the plain sum.
			if got, want := midUp(a, b), (a+b+1)>>1; got != want {
				t.Errorf("midUp(%d, %d) = %d, want %d", a, b, got, want)
			}
			// the mean half to even: a+b even is exact; odd goes to the even neighbour.
			s := a + b
			want := s >> 1
			if s%2 != 0 && want%2 != 0 {
				want++
			}
			if got := meanHalfEven(a, b); got != want {
				t.Errorf("meanHalfEven(%d, %d) = %d, want %d", a, b, got, want)
			}
		}
	}
	if got := midUp(maxInt64, maxInt64); got != maxInt64 {
		t.Errorf("midUp(max, max) = %d, want max", got)
	}
	if got := meanHalfEven(minInt64, minInt64); got != minInt64 {
		t.Errorf("meanHalfEven(min, min) = %d, want min", got)
	}
}

// TestRoundTicksUp verifies the multiple-of-5 rounding at the top of the range.
func TestRoundTicksUp(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in, want int64
		ok       bool
	}{{0, 0, true}, {1, 5, true}, {5, 5, true}, {-1, 0, true}, {-6, -5, true}, {maxInt64, 0, false}, {maxInt64 - 1, 0, false}, {maxInt64 - 2, maxInt64 - 2, true}} {
		got, ok := roundTicksUp(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("roundTicksUp(%d) = %d, %v, want %d, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// TestSaturating verifies that the saturating add and subtract clamp at the
// limits of int64 and never wrap.
//
// Guard: a link read as "value minus 10 000" must not wrap for an extreme value
// and turn a regime into none.
func TestSaturating(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ a, b, add, sub int64 }{
		{1, 2, 3, -1},
		{maxInt64, 1, maxInt64, maxInt64 - 1},
		{minInt64, -1, minInt64, minInt64 + 1},
		{maxInt64, -1, maxInt64 - 1, maxInt64},
		{minInt64, 1, minInt64 + 1, minInt64},
		{0, minInt64, minInt64, maxInt64},
		{-1, minInt64, minInt64, maxInt64 - 0},
		{-5, minInt64, minInt64, maxInt64 - 4},
	} {
		if got := satAdd(tc.a, tc.b); got != tc.add {
			t.Errorf("satAdd(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.add)
		}
		if got := satSub(tc.a, tc.b); got != tc.sub {
			t.Errorf("satSub(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.sub)
		}
	}
}

// TestDetect_UnknownDetectorPanics verifies the programmer-error panic of
// Detect for a detector that is not in Detectors.
func TestDetect_UnknownDetectorPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Errorf("Detect(\"R0\") did not panic")
		}
	}()
	Detect("R0", Input{}, NewThresholds())
}
