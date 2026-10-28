package endowment_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/curve"
	"github.com/Whealinghold/economy-sim/internal/endowment"
)

// TestOvershoot_RefillAndLogistic verifies resource_overshoot on the worked
// check of spec-resources.md 8.3.1: r* 12 000, dR 400, dX 8 000, a plan share of
// 6 000 and a curve through the origin (eps 10 000). The right side is
// 4 * 12 000 * 400 * 10 000 * 4 000 = 7.68e14. For a refill the left side is
// rate * 10^4 * 8 000 * 6 000, so 5 000 bp gives 2.4e15 (no warning), 1 600 bp
// gives 7.68e14 (equal, no warning) and 1 599 bp is below it (warning). For a
// logistic stock the left is rate * 2 000^2 * 8 000 = rate * 3.2e10, equal at
// 24 000.
//
// Requirement: FR-RES-044
func TestOvershoot_RefillAndLogistic(t *testing.T) {
	t.Parallel()
	plan := func(shape endowment.Shape, rate, share int64) endowment.Plan {
		return endowment.Plan{Shape: shape, RateBPEff: rate, ShareBP: share, ElasticityBP: 10_000, RStar: 12_000, DR: 400, DX: 8000}
	}
	tests := []struct {
		name string
		p    endowment.Plan
		want bool
	}{
		{"refill well above the threshold", plan(endowment.Refill, 5000, 6000), false},
		{"refill at equality", plan(endowment.Refill, 1600, 6000), false},
		{"refill just below", plan(endowment.Refill, 1599, 6000), true},
		{"logistic at equality", plan(endowment.Logistic, 24_000, 6000), false},
		{"logistic just below", plan(endowment.Logistic, 23_999, 6000), true},
		{"logistic at half is the other warning", plan(endowment.Logistic, 1, 5000), false},
		{"constant never overshoots", plan(endowment.Constant, 1, 6000), false},
		{"flat curve has no feedback", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1, ShareBP: 6000, RStar: 12_000, DR: 400, DX: 8000}, false},
		// With every other field valid and the warning true, a negative field alone must stop it.
		{"negative rate", endowment.Plan{Shape: endowment.Refill, RateBPEff: -1, ShareBP: 6000, ElasticityBP: 10_000, RStar: 12_000, DR: 400, DX: 8000}, false},
		{"negative elasticity", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1, ShareBP: 6000, ElasticityBP: -1, RStar: 12_000, DR: 400, DX: 8000}, false},
		{"negative r*", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1, ShareBP: 6000, ElasticityBP: 10_000, RStar: -1, DR: 400, DX: 8000}, false},
		{"negative dR", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1, ShareBP: 6000, ElasticityBP: 10_000, RStar: 12_000, DR: -1, DX: 8000}, false},
		{"negative dX", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1, ShareBP: 6000, ElasticityBP: 10_000, RStar: 12_000, DR: 400, DX: -1}, false},
		{"negative share", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1, ShareBP: -1, ElasticityBP: 10_000, RStar: 12_000, DR: 400, DX: 8000}, false},
		{"share beyond the whole", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1, ShareBP: 10_001, ElasticityBP: 10_000, RStar: 12_000, DR: 400, DX: 8000}, false},
		// Zero is a value, not an error: a refill that does not refill, a population that does
		// not respond, and a plan share of 0 all make the left side 0 against a positive right.
		{"zero rate overshoots", endowment.Plan{Shape: endowment.Refill, RateBPEff: 0, ShareBP: 6000, ElasticityBP: 10_000, RStar: 12_000, DR: 400, DX: 8000}, true},
		{"zero dX overshoots", endowment.Plan{Shape: endowment.Refill, RateBPEff: 5000, ShareBP: 6000, ElasticityBP: 10_000, RStar: 12_000, DR: 400, DX: 0}, true},
		{"zero share overshoots", endowment.Plan{Shape: endowment.Refill, RateBPEff: 5000, ShareBP: 0, ElasticityBP: 10_000, RStar: 12_000, DR: 400, DX: 8000}, true},
		{"logistic below half is not an overshoot", endowment.Plan{Shape: endowment.Logistic, RateBPEff: 1, ShareBP: 4000, ElasticityBP: 10_000, RStar: 12_000, DR: 400, DX: 8000}, false},
		// A right side beyond 128 bits is above any left side: 2^63^4 * 4 needs 254 bits.
		{"right beyond 128 bits", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1, ShareBP: 5000, ElasticityBP: 1 << 62, RStar: 1 << 62, DR: 1 << 62, DX: 1}, true},
		// 1e9 * 10^4 * 1 * 5 000 = 5e16 against 4 * 1 * 1 * 1 * 5 000 = 20 000: the high words differ.
		{"left above right in the high word", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1_000_000_000_000, ShareBP: 5000, ElasticityBP: 1, RStar: 1, DR: 1, DX: 1_000_000_000_000}, false},
		{"right above left in the high word", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1, ShareBP: 5000, ElasticityBP: 10_000, RStar: 1_000_000_000_000, DR: 1_000_000_000_000, DX: 1}, true},
		{"left beyond 128 bits", endowment.Plan{Shape: endowment.Refill, RateBPEff: 1 << 62, ShareBP: 5000, ElasticityBP: 1, RStar: 1, DR: 1, DX: 1 << 62}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := endowment.Overshoot(tt.p); got != tt.want {
				t.Errorf("Overshoot(%+v) = %v, want %v", tt.p, got, tt.want)
			}
		})
	}
}

// TestUnstablePlan_Shapes verifies resource_unstable_plan: a logistic stock at
// or below half its capacity, and a constant flow, with an elasticity above 0;
// never a refill, never a flat curve.
//
// Requirement: FR-RES-044
func TestUnstablePlan_Shapes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		p    endowment.Plan
		want bool
	}{
		{"logistic at half", endowment.Plan{Shape: endowment.Logistic, ShareBP: 5000, ElasticityBP: 1}, true},
		{"logistic below half", endowment.Plan{Shape: endowment.Logistic, ShareBP: 4000, ElasticityBP: 1}, true},
		{"logistic with a negative field", endowment.Plan{Shape: endowment.Logistic, ShareBP: 4000, ElasticityBP: 1, DX: -1}, false},
		{"logistic above half", endowment.Plan{Shape: endowment.Logistic, ShareBP: 5001, ElasticityBP: 1}, false},
		{"logistic flat curve", endowment.Plan{Shape: endowment.Logistic, ShareBP: 4000}, false},
		{"constant with feedback", endowment.Plan{Shape: endowment.Constant, ShareBP: 9000, ElasticityBP: 5}, true},
		{"constant with a rate of 0", endowment.Plan{Shape: endowment.Constant, ShareBP: 9000, ElasticityBP: 5}, true},
		{"constant at the whole share", endowment.Plan{Shape: endowment.Constant, ShareBP: 10_000, ElasticityBP: 5}, true},
		{"share beyond the whole", endowment.Plan{Shape: endowment.Constant, ShareBP: 10_001, ElasticityBP: 5}, false},
		{"constant flat curve", endowment.Plan{Shape: endowment.Constant, ShareBP: 9000}, false},
		{"refill never", endowment.Plan{Shape: endowment.Refill, ShareBP: 1000, ElasticityBP: 9999}, false},
		{"negative elasticity", endowment.Plan{Shape: endowment.Constant, ElasticityBP: -1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := endowment.UnstablePlan(tt.p); got != tt.want {
				t.Errorf("UnstablePlan(%+v) = %v, want %v", tt.p, got, tt.want)
			}
		})
	}
}

// TestElasticity_FromTheSegment verifies eps_bp = 10^4 * sig * slope / a(sig)
// with the slope of the segment that contains the share, rounded half to even,
// and 0 for a flat curve, a share outside the knots, or a factor of 0.
//
// Requirement: FR-RES-044
func TestElasticity_FromTheSegment(t *testing.T) {
	t.Parallel()
	origin := []curve.Knot{{X: 0, Y: 0}, {X: 10_000, Y: 10_000}}
	bent := []curve.Knot{{X: 0, Y: 2000}, {X: 2000, Y: 8000}, {X: 10_000, Y: 10_000}}
	tests := []struct {
		name  string
		knots []curve.Knot
		share int64
		want  int64
	}{
		// A curve through the origin has an elasticity of 1: 10^4 * 6 000 * 10 000 / (10 000 * 6 000).
		{"through the origin", origin, 6000, 10_000},
		// a(5 000) = 8 750 on the segment (2 000, 8 000)-(10 000, 10 000):
		// 10^4 * 5 000 * 2 000 / (8 000 * 8 750) = 1 428.57, rounded to 1 429.
		{"bent curve", bent, 5000, 1429},
		{"flat segment", []curve.Knot{{X: 0, Y: 5000}, {X: 10_000, Y: 5000}}, 5000, 0},
		{"at the last knot", origin, 10_000, 0},
		{"below the first knot", []curve.Knot{{X: 1000, Y: 0}, {X: 2000, Y: 100}}, 500, 0},
		{"factor 0 is not evaluated", origin, 0, 0},
		// a(5 000) = 7 500 on (0, 10 000)-(10 000, 5 000): 10^4 * 5 000 * (-5 000) / (10 000 * 7 500)
		// = -3 333.33, a falling curve has a negative elasticity, rounded to -3 333.
		{"falling curve", []curve.Knot{{X: 0, Y: 10_000}, {X: 10_000, Y: 5000}}, 5000, -3333},
		// a(4 999) = 1 + trunc(2 * 4 999 / 10 000) = 1: 10^4 * 4 999 * 2 / (10 000 * 1) = 9 998.
		{"factor 1 is evaluated", []curve.Knot{{X: 0, Y: 1}, {X: 10_000, Y: 3}}, 4999, 9998},
		{"no knots", nil, 5000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := endowment.Elasticity(tt.knots, tt.share)
			if !ok || got != tt.want {
				t.Errorf("Elasticity(%v, %d) = %d, %v, want %d, true", tt.knots, tt.share, got, ok, tt.want)
			}
		})
	}
}

// TestElasticity_OverflowReported verifies each product of the elasticity that
// can leave int64 is reported with ok false and never wrapped.
//
// Requirement: NFR-RES-003
func TestElasticity_OverflowReported(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		knots []curve.Knot
		share int64
	}{
		// a(50) = trunc(MaxInt64 * 50 / 100) = 4.6e18 and dx * a = 100 * 4.6e18 leaves int64.
		{"divisor", []curve.Knot{{X: 0, Y: 0}, {X: 100, Y: math.MaxInt64}}, 50},
		// share * 10^4 = 10^19 leaves int64.
		{"numerator", []curve.Knot{{X: 0, Y: 0}, {X: 2_000_000_000_000_000, Y: 10}}, 1_000_000_000_000_000},
		// dy leaves int64, which Eval reports first.
		{"curve", []curve.Knot{{X: 0, Y: -5}, {X: 10, Y: math.MaxInt64}}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := endowment.Elasticity(tt.knots, tt.share); ok {
				t.Errorf("Elasticity(%v, %d) = %d, true, want not ok", tt.knots, tt.share, got)
			}
		})
	}
}
