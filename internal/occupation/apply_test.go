package occupation_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// TestApplyOK_QueueDiscountAndFloor verifies the queue-discounted capital test of
// spec-skill-barriers.md 4.5: both forms, equal sides, one either side, the
// bracket floored at 0, and the denials on bad input.
//
// The base case has diff = 5 000 and ut = 8 * 250 = 2 000, so diff * ut =
// 10 000 000 = p_d * D_work * 10 000. With a capital of 10 000 and a return of
// 1 000 the right side is 10 000 * 1 000 * 10 000 = 10^11 = 10^7 * 10 000, the
// left side with no queue, so the sides are equal. With a wait of 100 ticks the
// discount is 1 000 * 100 / 250 = 400, the bracket 9 600, and the capital that
// still clears is 9 600.
//
// Requirement: FR-SKL-005
func TestApplyOK_QueueDiscountAndFloor(t *testing.T) {
	t.Parallel()
	base := occupation.ApplyParams{ED: 5000, EO: 0, CapitalReturn: 1000, Upkeep: 8, Year: 250}
	with := func(f func(*occupation.ApplyParams)) occupation.ApplyParams { x := base; f(&x); return x }
	tests := []struct {
		name string
		p    occupation.ApplyParams
		want bool
	}{
		{"free and no queue", base, true},
		{"equal sides, no queue", with(func(p *occupation.ApplyParams) { p.ToolPrice = 4000; p.CTrain = 6000 }), true},
		{"one above equal, no queue", with(func(p *occupation.ApplyParams) { p.ToolPrice = 4000; p.CTrain = 6001 }), false},
		{"equal sides after a wait of 100", with(func(p *occupation.ApplyParams) { p.ToolPrice = 3600; p.CTrain = 6000; p.QWait = 100 }), true},
		{"one above equal after a wait", with(func(p *occupation.ApplyParams) { p.ToolPrice = 3601; p.CTrain = 6000; p.QWait = 100 }), false},
		{"the discount truncates toward zero", with(func(p *occupation.ApplyParams) { p.ToolPrice = 10_000; p.QWait = 1 }), false},
		{"bracket 0 denies a free role", with(func(p *occupation.ApplyParams) { p.QWait = 2500 }), false},
		{"bracket below 0 is floored and denies", with(func(p *occupation.ApplyParams) { p.QWait = 9999 }), false},
		{"bracket 1 still clears a free role", with(func(p *occupation.ApplyParams) { p.QWait = 2499 }), true},
		{"destination below own role", with(func(p *occupation.ApplyParams) { p.ED = -1 }), false},
		{"destination equal to own role and free", with(func(p *occupation.ApplyParams) { p.ED = 0 }), true},
		{"destination equal to own role and costly", with(func(p *occupation.ApplyParams) { p.ED = 0; p.ToolPrice = 1 }), false},
		{"no upkeep", with(func(p *occupation.ApplyParams) { p.Upkeep = 0 }), false},
		{"no year", with(func(p *occupation.ApplyParams) { p.Year = 0 }), false},
		{"negative tool price", with(func(p *occupation.ApplyParams) { p.ToolPrice = -1 }), false},
		{"negative training cost", with(func(p *occupation.ApplyParams) { p.CTrain = -1 }), false},
		{"negative return", with(func(p *occupation.ApplyParams) { p.CapitalReturn = -1 }), false},
		{"negative wait", with(func(p *occupation.ApplyParams) { p.QWait = -1 }), false},
		{"only a negative tool price", with(func(p *occupation.ApplyParams) { p.ToolPrice = -1; p.CTrain = 1 }), false},
		{"only a negative training cost", with(func(p *occupation.ApplyParams) { p.ToolPrice = 1; p.CTrain = -1 }), false},
		{"negative upkeep", with(func(p *occupation.ApplyParams) { p.Upkeep = -8; p.Year = -250 }), false},
		{"negative year with positive product", with(func(p *occupation.ApplyParams) { p.Upkeep = -8; p.Year = 250 }), false},
		{"signals shifted by the same amount", with(func(p *occupation.ApplyParams) { p.ED, p.EO = 7000, 2000 }), true},
		{"shifted signals, one above equal", with(func(p *occupation.ApplyParams) { p.ED, p.EO = 7000, 2000; p.ToolPrice = 10_001 }), false},
		{"upkeep times year overflows", with(func(p *occupation.ApplyParams) { p.Upkeep = math.MaxInt64; p.Year = 2 }), false},
		{"capital overflows", with(func(p *occupation.ApplyParams) { p.ToolPrice = math.MaxInt64; p.CTrain = 1 }), false},
		{"discount overflows", with(func(p *occupation.ApplyParams) { p.CapitalReturn = math.MaxInt64; p.QWait = math.MaxInt64 }), false},
		// ED - EO needs 64 bits: the exact difference 2^64 - 2 is formed unwrapped.
		{"difference beyond int64", occupation.ApplyParams{ED: math.MaxInt64, EO: math.MinInt64, ToolPrice: math.MaxInt64, CapitalReturn: 1, Upkeep: 1, Year: math.MaxInt64}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.ApplyOK(tt.p); got != tt.want {
				t.Errorf("ApplyOK(%+v) = %v, want %v", tt.p, got, tt.want)
			}
		})
	}
}

// TestApplyOK_EqualsCapitalOKWithoutQueue verifies that with no wait and no
// training cost ApplyOK agrees with the tool half CapitalOK wherever the tool
// half does not pass for a free tool.
//
// Requirement: FR-SKL-005
func TestApplyOK_EqualsCapitalOKWithoutQueue(t *testing.T) {
	t.Parallel()
	for _, price := range []int64{1, 999, 1000, 1001, 5_000_000} {
		for _, diff := range []int64{0, 1, 500, 5000} {
			want := occupation.CapitalOK(diff, 0, price, 1000, 8, 250)
			got := occupation.ApplyOK(occupation.ApplyParams{ED: diff, ToolPrice: price, CapitalReturn: 1000, Upkeep: 8, Year: 250})
			if got != want {
				t.Errorf("ApplyOK(diff %d, price %d) = %v, CapitalOK = %v", diff, price, got, want)
			}
		}
	}
}

// TestRefusals_OrderOfTests verifies the refusals appended with the skill
// barriers: one failing check each, the order of the table of
// spec-spawner.md 4 when several fail, and that a removal and a reduction are
// never refused for them.
//
// Requirement: FR-SPN-009, FR-SPN-011, FR-SKL-010, FR-SKL-012
func TestRefusals_OrderOfTests(t *testing.T) {
	t.Parallel()
	ok := occupation.Facts{AgentKnown: true, RoleKnown: true, CauseSupported: true, Households: 2, Weight: 1}
	with := func(f func(*occupation.Facts)) occupation.Facts { x := ok; f(&x); return x }
	tests := []struct {
		name string
		kind occupation.Kind
		f    occupation.Facts
		want occupation.Refusal
	}{
		{"apply passes", occupation.KindApply, ok, occupation.RefusalNone},
		{"apply of weight 0 is not refused for its weight", occupation.KindApply, with(func(f *occupation.Facts) { f.Weight = 0 }), occupation.RefusalNone},
		{"apply of a trainee", occupation.KindApply, with(func(f *occupation.Facts) { f.InTraineeship = true }), occupation.InTraineeship},
		{"apply needs weight one", occupation.KindApply, with(func(f *occupation.Facts) { f.Weight = 2 }), occupation.NeedsWeightOne},
		{"apply already queued", occupation.KindApply, with(func(f *occupation.Facts) { f.Queued = true }), occupation.AlreadyQueued},
		{"apply on a journey", occupation.KindApply, with(func(f *occupation.Facts) { f.OnJourney = true }), occupation.OnJourney},
		{"apply of an external agent", occupation.KindApply, with(func(f *occupation.Facts) { f.External = true }), occupation.ExternalAgent},
		{"apply to an unknown role", occupation.KindApply, with(func(f *occupation.Facts) { f.RoleKnown = false }), occupation.UnknownRole},
		{"apply to an unknown agent", occupation.KindApply, with(func(f *occupation.Facts) { f.AgentKnown = false }), occupation.UnknownAgent},
		{"apply to the own role is not same_role", occupation.KindApply, with(func(f *occupation.Facts) { f.SameRole = true }), occupation.RefusalNone},
		{"order: in_traineeship before on_journey", occupation.KindApply, with(func(f *occupation.Facts) { f.InTraineeship, f.OnJourney = true, true }), occupation.InTraineeship},
		{"order: on_journey before needs_weight_one", occupation.KindApply, with(func(f *occupation.Facts) { f.OnJourney, f.Weight = true, 3 }), occupation.OnJourney},
		{"order: needs_weight_one before already_queued", occupation.KindApply, with(func(f *occupation.Facts) { f.Weight, f.Queued = 3, true }), occupation.NeedsWeightOne},
		{"switch of a trainee", occupation.KindSwitch, with(func(f *occupation.Facts) { f.InTraineeship = true }), occupation.InTraineeship},
		{"switch: same_role before in_traineeship", occupation.KindSwitch, with(func(f *occupation.Facts) { f.SameRole, f.InTraineeship = true, true }), occupation.SameRole},
		{"switch: in_traineeship before on_journey", occupation.KindSwitch, with(func(f *occupation.Facts) { f.InTraineeship, f.OnJourney = true, true }), occupation.InTraineeship},
		{"switch into a training role", occupation.KindSwitch, with(func(f *occupation.Facts) { f.TrainingRole = true }), occupation.NeedsTraineeship},
		{"switch: on_journey before needs_traineeship", occupation.KindSwitch, with(func(f *occupation.Facts) { f.OnJourney, f.TrainingRole = true, true }), occupation.OnJourney},
		{"enter into a training role", occupation.KindEnter, with(func(f *occupation.Facts) { f.TrainingRole = true }), occupation.NeedsTraineeship},
		{"enter: needs_traineeship before bad households", occupation.KindEnter, with(func(f *occupation.Facts) { f.TrainingRole, f.Households = true, 0 }), occupation.NeedsTraineeship},
		{"remove is never refused for a traineeship", occupation.KindRemove, with(func(f *occupation.Facts) { f.InTraineeship, f.Queued, f.TrainingRole = true, true, true }), occupation.RefusalNone},
		{"reduce is never refused for a traineeship", occupation.KindReduce, with(func(f *occupation.Facts) { f.InTraineeship, f.Queued, f.TrainingRole, f.Weight = true, true, true, 2 }), occupation.RefusalNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.Refuse(tt.kind, tt.f); got != tt.want {
				t.Errorf("Refuse(%d, %+v) = %d, want %d", tt.kind, tt.f, got, tt.want)
			}
		})
	}
}

// exactApplyOK is the inequality of ApplyOK in math/big, for the inputs where
// every step fits int64.
func exactApplyOK(p occupation.ApplyParams) (want, valid bool) {
	bi := big.NewInt
	if p.ToolPrice < 0 || p.CTrain < 0 || p.CapitalReturn < 0 || p.QWait < 0 || p.ED < p.EO || p.Year <= 0 || p.Upkeep <= 0 {
		return false, true
	}
	ut := new(big.Int).Mul(bi(p.Upkeep), bi(p.Year))
	capital := new(big.Int).Add(bi(p.ToolPrice), bi(p.CTrain))
	disc := new(big.Int).Quo(new(big.Int).Mul(bi(p.CapitalReturn), bi(p.QWait)), bi(p.Year))
	if !ut.IsInt64() || !capital.IsInt64() || !disc.IsInt64() {
		return false, true
	}
	bracket := new(big.Int).Sub(bi(10_000), disc)
	if bracket.Sign() <= 0 {
		return false, true
	}
	left := new(big.Int).Mul(new(big.Int).Sub(bi(p.ED), bi(p.EO)), ut)
	left.Mul(left, bracket)
	right := new(big.Int).Mul(capital, bi(p.CapitalReturn))
	right.Mul(right, bi(10_000))
	return left.Cmp(right) >= 0, true
}

// TestApplyOK_MatchesExactProductsAtTheExtremes verifies ApplyOK against the
// exact inequality in math/big over a grid of extreme values, where the 192-bit
// product has carries between its words.
//
// Requirement: FR-SKL-005
func TestApplyOK_MatchesExactProductsAtTheExtremes(t *testing.T) {
	t.Parallel()
	signals := [][2]int64{{math.MaxInt64, math.MinInt64}, {math.MaxInt64, 0}, {1 << 62, -(1 << 62)}, {5000, 0}}
	factors := []int64{1, 1 << 31, 1<<31 + 1, math.MaxInt32, 1 << 62}
	capitals := []int64{0, 1, 1 << 40, 1<<62 + 12345, math.MaxInt64 - 1}
	n := 0
	for _, s := range signals {
		for _, f := range factors {
			for _, c := range capitals {
				for _, cr := range []int64{1, 1000, 1<<62 + 3} {
					p := occupation.ApplyParams{ED: s[0], EO: s[1], ToolPrice: c, CapitalReturn: cr, Upkeep: f, Year: min(f, 1<<31+7), QWait: 0}
					want, _ := exactApplyOK(p)
					if got := occupation.ApplyOK(p); got != want {
						t.Errorf("ApplyOK(%+v) = %v, want %v", p, got, want)
					}
					n++
				}
			}
		}
	}
	if n == 0 {
		t.Fatal("empty grid")
	}
}

// FuzzApplyOK verifies, for any int64 inputs, that ApplyOK never panics and
// equals the exact inequality diff * ut * bracket >= (P + C) * CR * 10 000 in
// math/big wherever every step fits int64, and denies where a step does not.
//
// Requirement: FR-SKL-005
func FuzzApplyOK(f *testing.F) {
	f.Add(int64(5000), int64(0), int64(3600), int64(6000), int64(1000), int64(8), int64(250), int64(100))
	f.Add(int64(math.MaxInt64), int64(math.MinInt64), int64(1), int64(1), int64(1), int64(1), int64(1), int64(0))
	f.Add(int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0))
	f.Fuzz(func(t *testing.T, ed, eo, price, ctrain, cr, upkeep, year, q int64) {
		got := occupation.ApplyOK(occupation.ApplyParams{ED: ed, EO: eo, ToolPrice: price, CTrain: ctrain, CapitalReturn: cr, Upkeep: upkeep, Year: year, QWait: q})
		bi := big.NewInt
		if price < 0 || ctrain < 0 || cr < 0 || q < 0 || ed < eo || year <= 0 || upkeep <= 0 {
			if got {
				t.Fatalf("ApplyOK accepted an invalid input ed=%d eo=%d price=%d ctrain=%d cr=%d upkeep=%d year=%d q=%d", ed, eo, price, ctrain, cr, upkeep, year, q)
			}
			return
		}
		ut := new(big.Int).Mul(bi(upkeep), bi(year))
		capital := new(big.Int).Add(bi(price), bi(ctrain))
		disc := new(big.Int).Quo(new(big.Int).Mul(bi(cr), bi(q)), bi(year))
		if !ut.IsInt64() || !capital.IsInt64() || !disc.IsInt64() {
			if got {
				t.Fatalf("ApplyOK accepted a step beyond int64")
			}
			return
		}
		bracket := new(big.Int).Sub(bi(10_000), disc)
		if bracket.Sign() <= 0 {
			if got {
				t.Fatalf("ApplyOK accepted a bracket of %v", bracket)
			}
			return
		}
		left := new(big.Int).Mul(new(big.Int).Sub(bi(ed), bi(eo)), ut)
		left.Mul(left, bracket)
		right := new(big.Int).Mul(capital, bi(cr))
		right.Mul(right, bi(10_000))
		if want := left.Cmp(right) >= 0; got != want {
			t.Fatalf("ApplyOK = %v, want %v (left %v, right %v)", got, want, left, right)
		}
	})
}
