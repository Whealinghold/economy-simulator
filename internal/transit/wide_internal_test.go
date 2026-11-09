package transit

import (
	"math"
	"testing"
)

// TestCeilDiv128_Boundaries verifies the rounding and the int64 limit of the
// 128-bit division behind Labour: the ceiling at the largest result, a result one
// past it, a quotient that does not fit 64 bits, and the comparison of two
// products across the 64-bit word.
//
// Requirement: FR-TRN-041
func TestCeilDiv128_Boundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		a, b, d int64
		want    int64
		ok      bool
	}{
		{"largest result, exact", math.MaxInt64, 1, 1, math.MaxInt64, true},
		{"largest result rounded up", math.MaxInt64, 3, 3, math.MaxInt64, true},
		// (2^32 - 1)(2^32 + 1) = 2^64 - 1 = 2 * (2^63 - 1) + 1: the ceiling is 2^63.
		{"ceiling one past int64", 1<<32 - 1, 1<<32 + 1, 2, 0, false},
		// 2^32 * 2^31 = 2^63 exactly, one past int64.
		{"exact one past int64", 1 << 32, 1 << 31, 1, 0, false},
		// 2^62 * 4 = 2^64: the high word equals the divisor, so the quotient has 65 bits.
		{"quotient past 64 bits", 1 << 62, 4, 1, 0, false},
		{"small rounded up", 7, 7, 5, 10, true}, // 49 / 5 = 9.8
		{"zero numerator", 0, 5, 3, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ceilDiv128(tt.a, tt.b, tt.d)
			if got != tt.want || ok != tt.ok {
				t.Errorf("ceilDiv128(%d, %d, %d) = %d, %v, want %d, %v", tt.a, tt.b, tt.d, got, ok, tt.want, tt.ok)
			}
		})
	}
	if mulCmp(1<<63, 2, 1, 1<<63) <= 0 || mulCmp(1, 1<<63, 1<<63, 2) >= 0 || mulCmp(3, 4, 6, 2) != 0 {
		t.Errorf("mulCmp does not order products across the word: want 2^64 > 2^63, 2^63 < 2^64, 12 == 12")
	}
}
