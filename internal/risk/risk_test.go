package risk_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/risk"
)

// TestProb_ClampAndRhe verifies the effective probability: the product of the
// row and its scale over 10 000 rounded half to even, clamped at 10 000, and 0
// for a scale of 0 (a row that takes no word).
//
// Requirement: FR-RSK-006, FR-RSK-040
func TestProb_ClampAndRhe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		strike, scale   int64
		want            int64
		wantOK          bool
		arithmeticNotes string
	}{
		{"scale one", 3_000, 10_000, 3_000, true, "3 000 * 10 000 / 10 000"},
		{"scale zero", 3_000, 0, 0, true, "0"},
		{"half scale", 3_000, 5_000, 1_500, true, "1 500"},
		{"tie to even down", 1, 5_000, 0, true, "0.5 goes to 0"},
		{"tie to even up", 3, 5_000, 2, true, "1.5 goes to 2"},
		{"above one rounds up", 1, 5_001, 1, true, "0.5001"},
		{"clamped at the whole", 6_000, 100_000, 10_000, true, "60 000 clamped"},
		{"largest case", 10_000, 100_000, 10_000, true, "10^9 / 10^4 = 100 000 clamped"},
		{"negative strike", -1, 10_000, 0, false, "refused"},
		{"negative scale", 1, -1, 0, false, "refused"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := risk.Prob(tt.strike, tt.scale)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Prob(%d, %d) = %d, %v, want %d, %v (%s)", tt.strike, tt.scale, got, ok, tt.want, tt.wantOK, tt.arithmeticNotes)
			}
		})
	}
}

// TestStruck_BelowP verifies the strike test is a strict comparison: a draw
// equal to P spares, a draw one below strikes, and P of 10 000 strikes every
// draw of [0, 10 000).
//
// Requirement: FR-RSK-006
func TestStruck_BelowP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		draw, p int64
		want    bool
	}{
		{"one below", 2_999, 3_000, true},
		{"equal spares", 3_000, 3_000, false},
		{"above spares", 3_001, 3_000, false},
		{"certain strikes the top draw", 9_999, 10_000, true},
		{"zero never strikes", 0, 0, false},
		{"lowest draw strikes any p above 0", 0, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := risk.Struck(tt.draw, tt.p); got != tt.want {
				t.Errorf("Struck(%d, %d) = %v, want %v", tt.draw, tt.p, got, tt.want)
			}
		})
	}
}

// TestFold_TwoRowsExample verifies the fold of the kept share on the example of
// spec-risk.md 4.3.1: severities 2 500 and 4 000 both struck keep rhe(10 000 *
// 7 500, 10 000) = 7 500 then rhe(7 500 * 6 000, 10 000) = 4 500, a loss of
// 5 500; and that the order fixes the rounding, a severity of 10 000 is a total
// loss and no struck row is no loss.
//
// Requirement: FR-RSK-009
func TestFold_TwoRowsExample(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		sev  []int64
		want int64
	}{
		{"none", nil, 0},
		{"one row", []int64{2_500}, 2_500},
		{"two rows", []int64{2_500, 4_000}, 5_500},
		{"two rows reversed", []int64{4_000, 2_500}, 5_500},
		{"total", []int64{10_000}, 10_000},
		{"total after a partial", []int64{1_234, 10_000}, 10_000},
		{"rounding step", []int64{3, 3}, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := risk.Combine(tt.sev)
			if !ok || got != tt.want {
				t.Errorf("Combine(%v) = %d, %v, want %d, true", tt.sev, got, ok, tt.want)
			}
		})
	}
}

// TestFold_SeverityTenThousandIsTotal verifies that a severity of 10 000 leaves
// nothing kept and that an out-of-range input is refused.
//
// Requirement: FR-RSK-009
func TestFold_SeverityTenThousandIsTotal(t *testing.T) {
	t.Parallel()
	if got, ok := risk.Fold(10_000, 10_000); !ok || got != 0 {
		t.Errorf("Fold(10000, 10000) = %d, %v, want 0, true", got, ok)
	}
	for _, in := range [][2]int64{{-1, 0}, {10_001, 0}, {5_000, -1}, {5_000, 10_001}} {
		if got, ok := risk.Fold(in[0], in[1]); ok || got != 0 {
			t.Errorf("Fold(%d, %d) = %d, %v, want 0, false", in[0], in[1], got, ok)
		}
	}
	if got, ok := risk.Combine([]int64{500, 10_001}); ok || got != 0 {
		t.Errorf("Combine with a severity above 10 000 = %d, %v, want 0, false", got, ok)
	}
}

// TestLossLine_RheTies verifies the loss of a line on the rows of
// spec-risk.md 18.1 at a share of 2 500: 37 units lose rhe(9.25) = 9, 5 lose
// rhe(1.25) = 1, 2 lose rhe(0.5) = 0 (a tie to the even 0) and 6 lose rhe(1.5)
// = 2 (a tie to the even 2); and the one-unit rule of 5.3.2.
//
// Requirement: FR-RSK-012, FR-RSK-040
func TestLossLine_RheTies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		held, lossBP int64
		want         int64
	}{
		{"grain", 37, 2_500, 9},
		{"cloth", 5, 2_500, 1},
		{"rope tie down", 2, 2_500, 0},
		{"tar tie up", 6, 2_500, 2},
		{"one unit at the half", 1, 5_000, 0},
		{"one unit above the half", 1, 5_001, 1},
		{"three units at 2 500", 3, 2_500, 1},
		{"three units at 1 500", 3, 1_500, 0},
		{"total loss", 40, 10_000, 40},
		{"no loss", 40, 0, 0},
		{"empty line", 0, 7_000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := risk.Line(tt.held, tt.lossBP)
			if !ok || got != tt.want {
				t.Errorf("Line(%d, %d) = %d, %v, want %d, true", tt.held, tt.lossBP, got, ok, tt.want)
			}
		})
	}
	if got, ok := risk.Line(1, 10_001); ok || got != 0 {
		t.Errorf("Line(1, 10001) = %d, %v, want 0, false", got, ok)
	}
	if got, ok := risk.Line(-1, 10); ok || got != 0 {
		t.Errorf("Line(-1, 10) = %d, %v, want 0, false", got, ok)
	}
}

// TestLossLine_MeanErrorOfSmallLines verifies the bias stated in
// spec-risk.md 5.3.2: over lines of 1 to 20 units at a share of 2 500 the mean
// absolute error against the exact q * share / 10 000 is at most 0.025 units.
//
// Requirement: FR-RSK-012
func TestLossLine_MeanErrorOfSmallLines(t *testing.T) {
	t.Parallel()
	var signed int64 // sum of (lost * 10 000 - q * share), in units of 1/10 000
	for q := int64(1); q <= 20; q++ {
		lost, ok := risk.Line(q, 2_500)
		if !ok {
			t.Fatalf("Line(%d, 2500) not ok", q)
		}
		signed += lost*10_000 - q*2_500
	}
	// The mean signed error is signed / (20 * 10 000) units; 0.025 units is 500 / 20 000.
	if got, limit := math.Abs(float64(signed)), 0.025*20*10_000; got > limit {
		t.Errorf("sum of errors = %v, want at most %v (mean 0.025 units over 20 lines)", got, limit)
	}
}

// TestAccount_ExceptVehicle verifies the trip account of spec-risk.md 18.3: 12
// grain and 3 cloth at 2 500 lose 3 and 1, and the 2 cart at the index skipped
// lose none, where a rule over every commodity would have taken rhe(0.5) = 0
// and, above 5 000, one.
//
// Requirement: FR-RSK-013
func TestAccount_ExceptVehicle(t *testing.T) {
	t.Parallel()
	held := []int64{12, 3, 2} // grain, cloth, cart
	tests := []struct {
		name   string
		skip   int
		lossBP int64
		want   []int64
	}{
		{"cart skipped", 2, 2_500, []int64{3, 1, 0}},
		{"cart skipped at a high share", 2, 9_000, []int64{11, 3, 0}},
		{"no vehicle", -1, 2_500, []int64{3, 1, 0}},
		{"no vehicle, high share", -1, 9_000, []int64{11, 3, 2}},
		{"skip beyond the account", 7, 10_000, []int64{12, 3, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := risk.Account(nil, held, tt.skip, tt.lossBP)
			if !ok || !slices.Equal(got, tt.want) {
				t.Errorf("Account(%v, skip %d, %d) = %v, %v, want %v, true", held, tt.skip, tt.lossBP, got, ok, tt.want)
			}
		})
	}
	if got, ok := risk.Account(make([]int64, 0, 4), []int64{1, -1}, -1, 100); ok || got != nil {
		t.Errorf("Account with a negative count = %v, %v, want nil, false", got, ok)
	}
	buf := make([]int64, 0, 8)
	if got, _ := risk.Account(buf, held, 2, 2_500); &got[0] != &buf[:1][0] {
		t.Errorf("Account did not reuse dst: got a new backing array")
	}
}

// TestPhi_Example18_2 verifies the stored fractions of spec-risk.md 18.2: the
// bin has x_mass = floor(450 * 200 / 500) = 180, phi_mass = floor(180 * 10 000
// / 550) = 3 272, x_bulk = floor(320 * 100 / 400) = 80, phi_bulk = 2 000, so
// phi = 3 272; the pit has 1 636 and 4 000, so phi = 4 000.
//
// Requirement: FR-RSK-017
func TestPhi_Example18_2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                       string
		stored, sumCap, capK, load int64
		want                       int64
	}{
		{"bin mass", 450, 500, 200, 550, 3_272},
		{"bin bulk", 320, 400, 100, 400, 2_000},
		{"pit mass", 450, 500, 100, 550, 1_636},
		{"pit bulk", 320, 400, 200, 400, 4_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := risk.Phi(tt.stored, tt.sumCap, tt.capK, tt.load)
			if !ok || got != tt.want {
				t.Errorf("Phi(%d, %d, %d, %d) = %d, %v, want %d, true", tt.stored, tt.sumCap, tt.capK, tt.load, got, ok, tt.want)
			}
		})
	}
}

// TestPhi_LargerOfTwoDimensions verifies that the fraction of a class is the
// larger of its mass and bulk fractions (spec-risk.md 6.1.3).
//
// Requirement: FR-RSK-017
func TestPhi_LargerOfTwoDimensions(t *testing.T) {
	t.Parallel()
	if got := risk.Larger(3_272, 2_000); got != 3_272 {
		t.Errorf("Larger(3272, 2000) = %d, want 3272", got)
	}
	if got := risk.Larger(1_636, 4_000); got != 4_000 {
		t.Errorf("Larger(1636, 4000) = %d, want 4000", got)
	}
	if got := risk.Larger(7, 7); got != 7 {
		t.Errorf("Larger(7, 7) = %d, want 7", got)
	}
}

// TestPhi_ZeroSumCapZeroLoad verifies the two zero divisors give 0 and not a
// refusal (spec-risk.md 6.1.2) and that a negative input is refused.
//
// Requirement: FR-RSK-017
func TestPhi_ZeroSumCapZeroLoad(t *testing.T) {
	t.Parallel()
	if got, ok := risk.Phi(10, 0, 5, 10); !ok || got != 0 {
		t.Errorf("Phi with SumCap 0 = %d, %v, want 0, true", got, ok)
	}
	if got, ok := risk.Phi(10, 10, 5, 0); !ok || got != 0 {
		t.Errorf("Phi with Load 0 = %d, %v, want 0, true", got, ok)
	}
	for _, in := range [][4]int64{{-1, 1, 1, 1}, {1, -1, 1, 1}, {1, 1, -1, 1}, {1, 1, 1, -1}} {
		if got, ok := risk.Phi(in[0], in[1], in[2], in[3]); ok || got != 0 {
			t.Errorf("Phi(%v) = %d, %v, want 0, false", in, got, ok)
		}
	}
}

// TestPhi_Product128 verifies the first product is held in 128 bits: stored
// and capacity near 2^62 over a sum of the same size lose nothing, where a
// 64-bit product would wrap.
//
// Requirement: FR-RSK-017, FR-RSK-040
func TestPhi_Product128(t *testing.T) {
	t.Parallel()
	big := int64(1) << 62
	// x = floor(big * big / big) = big; phi = floor(big * 10 000 / big) does not
	// fit as a product in 64 bits but the quotient is 10 000.
	if got, ok := risk.Phi(big, big, big, big); !ok || got != 10_000 {
		t.Errorf("Phi(2^62, 2^62, 2^62, 2^62) = %d, %v, want 10000, true", got, ok)
	}
	// x = 2^62 and load 1 would give a quotient of 10 000 * 2^62: it does not fit.
	if got, ok := risk.Phi(big, big, big, 1); ok || got != 0 {
		t.Errorf("Phi with a quotient above int64 = %d, %v, want 0, false", got, ok)
	}
}

// TestShare_Example18_2 verifies the shares and losses of spec-risk.md 18.2:
// bin 1 * rhe(3 272 * 4 000, 10 000) = 1 309, pit 1 * rhe(4 000 * 10 000, 10
// 000) = 4 000, and from 250 grain and 50 cloth the losses 33 + 100 = 133 and 7
// + 20 = 27.
//
// Requirement: FR-RSK-019
func TestShare_Example18_2(t *testing.T) {
	t.Parallel()
	bin, ok := risk.Share(1, 3_272, 4_000)
	if !ok || bin != 1_309 {
		t.Fatalf("Share(1, 3272, 4000) = %d, %v, want 1309, true (rhe(1308.8))", bin, ok)
	}
	pit, ok := risk.Share(1, 4_000, 10_000)
	if !ok || pit != 4_000 {
		t.Fatalf("Share(1, 4000, 10000) = %d, %v, want 4000, true", pit, ok)
	}
	tests := []struct {
		name     string
		q0       int64
		wantBin  int64
		wantPit  int64
		wantLeft int64
	}{
		{"grain", 250, 33, 100, 117}, // rhe(32.725) = 33, rhe(100.0) = 100
		{"cloth", 50, 7, 20, 23},     // rhe(6.545) = 7, rhe(20.0) = 20
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b, _ := risk.Apply(tt.q0, bin)
			p, _ := risk.Apply(tt.q0, pit)
			if b != tt.wantBin || p != tt.wantPit {
				t.Errorf("Apply(%d, ...) = %d and %d, want %d and %d", tt.q0, b, p, tt.wantBin, tt.wantPit)
			}
			left := tt.q0 - risk.Capped(tt.q0, 0, b) - risk.Capped(tt.q0, b, p)
			if left != tt.wantLeft {
				t.Errorf("left of %d = %d, want %d", tt.q0, left, tt.wantLeft)
			}
		})
	}
}

// TestShare_CapAtTenThousand verifies the share of a class is capped at 10 000
// by the count of struck stores and that a product past int64 is the cap too.
//
// Requirement: FR-RSK-019
func TestShare_CapAtTenThousand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		n, phi, sev  int64
		want         int64
		wantOK       bool
		arithmeticIs string
	}{
		{"three stores", 3, 5_000, 10_000, 10_000, true, "3 * 5 000 = 15 000 capped"},
		{"under the cap", 2, 1_000, 2_500, 500, true, "2 * rhe(250) = 500"},
		{"no store struck", 0, 5_000, 5_000, 0, true, "0"},
		{"product past int64", math.MaxInt64, 10_000, 10_000, 10_000, true, "overflow means above the cap"},
		{"negative count", -1, 1, 1, 0, false, "refused"},
		{"negative phi", 1, -1, 1, 0, false, "refused"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := risk.Share(tt.n, tt.phi, tt.sev)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Share(%d, %d, %d) = %d, %v, want %d, %v (%s)", tt.n, tt.phi, tt.sev, got, ok, tt.want, tt.wantOK, tt.arithmeticIs)
			}
		})
	}
}

// TestApply_CappedAtStartQuantity verifies that the total taken of a commodity
// never exceeds the quantity at the start of the position, however many
// classes strike, and that a share above 10 000 is refused.
//
// Requirement: FR-RSK-019
func TestApply_CappedAtStartQuantity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		q0, taken, loss int64
		want            int64
	}{
		{"room left", 100, 30, 50, 50},
		{"capped", 100, 80, 50, 20},
		{"all taken", 100, 100, 5, 0},
		{"over taken", 100, 120, 5, 0},
		{"nothing to take", 100, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := risk.Capped(tt.q0, tt.taken, tt.loss); got != tt.want {
				t.Errorf("Capped(%d, %d, %d) = %d, want %d", tt.q0, tt.taken, tt.loss, got, tt.want)
			}
		})
	}
	if got, ok := risk.Apply(10, 10_001); ok || got != 0 {
		t.Errorf("Apply(10, 10001) = %d, %v, want 0, false", got, ok)
	}
}

// TestExpected_Example18_1 verifies the expected loss of spec-risk.md 18.1:
// rows (3 000, 2 500) and (1 000, 6 000) give f1 = 10 000 - rhe(750) = 9 250,
// f2 = 10 000 - rhe(600) = 9 400, K = 9 250 then rhe(9 250 * 9 400 / 10 000) =
// 8 695, so LossBP 1 305; and no row gives 0.
//
// Requirement: FR-RSK-024
func TestExpected_Example18_1(t *testing.T) {
	t.Parallel()
	got, ok := risk.Expected([]risk.Edge{{P: 3_000, SeverityBP: 2_500}, {P: 1_000, SeverityBP: 6_000}})
	if !ok || got != 1_305 {
		t.Errorf("Expected(example A) = %d, %v, want 1305, true", got, ok)
	}
	if got, ok := risk.Expected(nil); !ok || got != 0 {
		t.Errorf("Expected(nil) = %d, %v, want 0, true", got, ok)
	}
	if got, ok := risk.Expected([]risk.Edge{{P: 10_000, SeverityBP: 10_000}}); !ok || got != 10_000 {
		t.Errorf("Expected(certain total) = %d, %v, want 10000, true", got, ok)
	}
	for _, bad := range []risk.Edge{{P: -1, SeverityBP: 1}, {P: 10_001, SeverityBP: 1}, {P: 1, SeverityBP: -1}, {P: 1, SeverityBP: 10_001}} {
		if got, ok := risk.Expected([]risk.Edge{bad}); ok || got != 0 {
			t.Errorf("Expected(%+v) = %d, %v, want 0, false", bad, got, ok)
		}
	}
}

// TestRhe_LargestCaseNoOverflow verifies the largest products of the module do
// not overflow: 10 000 * MaxShockBP (100 000) for a probability and
// MaxQuantity (10^9) * 10 000 for a line.
//
// Requirement: FR-RSK-040
func TestRhe_LargestCaseNoOverflow(t *testing.T) {
	t.Parallel()
	if got, ok := risk.Prob(10_000, 100_000); !ok || got != 10_000 {
		t.Errorf("Prob(10000, 100000) = %d, %v, want 10000, true", got, ok)
	}
	if got, ok := risk.Line(1_000_000_000, 10_000); !ok || got != 1_000_000_000 {
		t.Errorf("Line(10^9, 10000) = %d, %v, want 10^9, true", got, ok)
	}
	if got, ok := risk.Line(math.MaxInt64, 10_000); !ok || got != math.MaxInt64 {
		t.Errorf("Line(MaxInt64, 10000) = %d, %v, want MaxInt64, true (128-bit product)", got, ok)
	}
}
