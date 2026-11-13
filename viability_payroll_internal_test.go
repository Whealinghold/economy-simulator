package economysim

import (
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// fracEq reports whether f is exactly num/den.
func fracEq(f steady.Frac, num, den int64) bool { return f.Cmp(steady.Rat(num, den)) == 0 }

// TestPayrollRegions verifies that a region is listed for the payroll only when
// it has both a payer (a share above 0) and a Hireable group, and that the
// regions come in region order.
//
// Requirement: FR-VIA-068
func TestPayrollRegions(t *testing.T) {
	t.Parallel()
	payer := steady.PayrollGroup{BP: 2500}
	worker := steady.PayrollGroup{Hireable: true}
	tests := []struct {
		name   string
		groups []steady.PayrollGroup // Region is set from the index in regionOf
		region []int
		want   []int
	}{
		{"payer and worker in one region", []steady.PayrollGroup{payer, worker}, []int{0, 0}, []int{0}},
		{"payer only", []steady.PayrollGroup{payer, {}}, []int{0, 0}, nil},
		{"worker only", []steady.PayrollGroup{worker, {}}, []int{0, 0}, nil},
		{"share 0 is not a payer", []steady.PayrollGroup{{BP: 0}, worker}, []int{0, 0}, nil},
		{"payer and worker in different regions", []steady.PayrollGroup{payer, worker}, []int{0, 1}, nil},
		{"second region only", []steady.PayrollGroup{payer, {}, payer, worker}, []int{0, 0, 1, 1}, []int{1}},
		{"both regions, in order", []steady.PayrollGroup{worker, payer, payer, worker}, []int{0, 0, 1, 1}, []int{0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := steady.PayrollInput{}
			for i, g := range tt.groups {
				g.Region = tt.region[i]
				in.Groups = append(in.Groups, g)
			}
			p := &viaProgramme{regions: make([]RegionID, 2)}
			if got := payrollRegions(p, in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("payrollRegions(%+v) = %v, want %v", tt.groups, got, tt.want)
			}
		})
	}
}

// TestPayrollPerExecution verifies the pay per execution that moves the
// net-wage equation: d * Out / Labour for an activity of a payer group, minus
// d * In / Labour for one of a Hireable group, 0 for an activity of no group and
// for a group with no labour.
//
// Requirement: FR-VIA-067
func TestPayrollPerExecution(t *testing.T) {
	t.Parallel()
	// Four activities of durations 2, 3, 5 and 7 in groups 0 (payer), 1 (worker), -1 and 0 again with no labour.
	// Group 0 pays 6 over a labour of 4: 3/2 a labour unit; group 1 receives 10 over a labour of 4: 5/2.
	// Activity 0: 2 * 6/4 = 3. Activity 1: -(3 * 10/4) = -15/2. Activity 2: no group, 0.
	p := &viaProgramme{md: steady.Model{Activities: []steady.Activity{{Duration: 2}, {Duration: 3}, {Duration: 5}, {Duration: 7}}}}
	in := steady.PayrollInput{
		Groups: []steady.PayrollGroup{{BP: 2500}, {Hireable: true}},
		Group:  []int{0, 1, -1, 0},
	}
	res := steady.PayrollResult{
		Out:    []steady.Frac{steady.Rat(6, 1), steady.Rat(0, 1)},
		In:     []steady.Frac{steady.Rat(0, 1), steady.Rat(10, 1)},
		Labour: []steady.Frac{steady.Rat(4, 1), steady.Rat(4, 1)},
	}
	got, _ := payrollPerExecution(p, in, res)
	for at, want := range [][2]int64{{3, 1}, {-15, 2}, {0, 1}, {21, 2}} {
		if at == 3 {
			continue // checked below with a labour of 0
		}
		if !fracEq(got[at], want[0], want[1]) {
			t.Errorf("payrollPerExecution()[%d] = %v/%v, want %d/%d", at, got[at].Num, got[at].Den, want[0], want[1])
		}
	}
	res.Labour[0] = steady.Rat(0, 1)
	if got, _ := payrollPerExecution(p, in, res); !fracEq(got[0], 0, 1) || !fracEq(got[3], 0, 1) {
		t.Errorf("payrollPerExecution(labour 0) = %v/%v, %v/%v, want 0 (no division by a zero labour)", got[0].Num, got[0].Den, got[3].Num, got[3].Den)
	}
}

// TestRegionCtx_WithPay verifies that the residual target of an activity moves
// by the pay per execution times the price scale, and is returned unchanged for
// a programme that kept its payroll-free duals.
//
// Requirement: FR-VIA-067
func TestRegionCtx_WithPay(t *testing.T) {
	t.Parallel()
	target := steady.Rat(100, 1)
	rc := &regionCtx{p: &viaProgramme{}, scale: steady.Rat(3, 1)}
	if got := rc.withPay(target, 0); !fracEq(got, 100, 1) {
		t.Errorf("withPay(nil rows) = %v/%v, want 100 unchanged", got.Num, got.Den)
	}
	// 100 + (-5/2) * 3 = 185/2 for activity 1; 100 + 4 * 3 = 112 for activity 0.
	rc.p.pay = &payrollRows{perExec: []steady.Frac{steady.Rat(4, 1), steady.Rat(-5, 2)}}
	if got := rc.withPay(target, 0); !fracEq(got, 112, 1) {
		t.Errorf("withPay(activity 0) = %v/%v, want 112", got.Num, got.Den)
	}
	if got := rc.withPay(target, 1); !fracEq(got, 185, 2) {
		t.Errorf("withPay(activity 1) = %v/%v, want 185/2", got.Num, got.Den)
	}
}

// TestRegionCtx_PayrollBP verifies the pay a household pays and receives, in
// basis points of its upkeep cost: pay * scale * 10 000 / (households * upkeep),
// rounded half to even, and 0 for each case the field documents (no payroll rows,
// an upkeep commodity with no seed price, no households, a figure that does not
// fit int64), reading the group at the base of its own region.
//
// Requirement: FR-VIA-069
func TestRegionCtx_PayrollBP(t *testing.T) {
	t.Parallel()
	// Upkeep of one household: 2 of commodity 0 at 50 and 1 of commodity 1 at 25 = 125.
	// Ten households: den = 1250, so a pay of x is 8x bp. Out 5/8 is 5 bp exactly; the halves are
	// x = 1/16 (0.5 bp, to 0), 5/16 (2.5, to 2), 3/16 (1.5, to 2) and 7/16 (3.5, to 4).
	group := func(households int64) groupCtx {
		return groupCtx{g: &Group{Households: households}, up: []int64{2, 1}}
	}
	tests := []struct {
		name        string
		pay         *payrollRows
		scale       steady.Frac
		price       []int64
		households  int64
		k, gi       int
		wantOut, in int64
	}{
		{"no rows", nil, steady.Rat(1, 1), []int64{50, 25}, 10, 0, 0, 0, 0},
		{"exact", &payrollRows{base: []int{0}, out: []steady.Frac{steady.Rat(5, 8)}, in: []steady.Frac{steady.Rat(5, 4)}}, steady.Rat(1, 1), []int64{50, 25}, 10, 0, 0, 5, 10},
		{"scale applies", &payrollRows{base: []int{0}, out: []steady.Frac{steady.Rat(5, 8)}, in: []steady.Frac{steady.Rat(0, 1)}}, steady.Rat(3, 1), []int64{50, 25}, 10, 0, 0, 15, 0},
		{"half rounds to even down", &payrollRows{base: []int{0}, out: []steady.Frac{steady.Rat(1, 16)}, in: []steady.Frac{steady.Rat(5, 16)}}, steady.Rat(1, 1), []int64{50, 25}, 10, 0, 0, 0, 2},
		{"half rounds to even up", &payrollRows{base: []int{0}, out: []steady.Frac{steady.Rat(3, 16)}, in: []steady.Frac{steady.Rat(7, 16)}}, steady.Rat(1, 1), []int64{50, 25}, 10, 0, 0, 2, 4},
		{"second region reads its own base", &payrollRows{base: []int{0, 2}, out: []steady.Frac{steady.Rat(0, 1), steady.Rat(0, 1), steady.Rat(5, 8)}, in: []steady.Frac{steady.Rat(0, 1), steady.Rat(0, 1), steady.Rat(0, 1)}}, steady.Rat(1, 1), []int64{50, 25}, 10, 1, 0, 5, 0},
		{"upkeep commodity with no seed price", &payrollRows{base: []int{0}, out: []steady.Frac{steady.Rat(5, 8)}, in: []steady.Frac{steady.Rat(5, 4)}}, steady.Rat(1, 1), []int64{50, 0}, 10, 0, 0, 0, 0},
		{"no households", &payrollRows{base: []int{0}, out: []steady.Frac{steady.Rat(5, 8)}, in: []steady.Frac{steady.Rat(5, 4)}}, steady.Rat(1, 1), []int64{50, 25}, 0, 0, 0, 0, 0},
		{"denominator overflows", &payrollRows{base: []int{0}, out: []steady.Frac{steady.Rat(5, 8)}, in: []steady.Frac{steady.Rat(5, 4)}}, steady.Rat(1, 1), []int64{50, 25}, math.MaxInt64 / 100, 0, 0, 0, 0},
		{"figure above int64 is 0", &payrollRows{base: []int{0}, out: []steady.Frac{steady.Rat(math.MaxInt64, 1)}, in: []steady.Frac{steady.Rat(5, 4)}}, steady.Rat(1, 1), []int64{50, 25}, 10, 0, 0, 0, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gc := group(tt.households)
			rc := &regionCtx{p: &viaProgramme{pay: tt.pay}, k: tt.k, scale: tt.scale, price: tt.price, groups: []groupCtx{gc}}
			out, in := rc.payrollBP(tt.gi)
			if out != tt.wantOut || in != tt.in {
				t.Errorf("payrollBP() = %d, %d, want %d, %d", out, in, tt.wantOut, tt.in)
			}
		})
	}
}
