package clearing_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
)

// TestAdmit_NotPermittedRow verifies the place of the not_permitted row of the
// institutions module: right after not_tradable and before every check of the market, so
// an order a right bars is refused as not permitted even when its price is invalid, and an
// order that fails an identity row reports that row.
//
// Requirement: FR-ORG-024, FR-CLR-002
func TestAdmit_NotPermittedRow(t *testing.T) {
	t.Parallel()
	mut := func(base clearing.Facts, f func(*clearing.Facts)) clearing.Facts { f(&base); return base }
	barred := func(f *clearing.Facts) { f.NotPermitted = true }
	tests := []struct {
		name string
		in   clearing.Facts
		want clearing.Code
	}{
		{"bid barred", mut(okBid(), barred), clearing.NotPermitted},
		{"ask barred", mut(okAsk(), barred), clearing.NotPermitted},
		{"not tradable comes first", mut(okBid(), func(f *clearing.Facts) { barred(f); f.Tradable = false }), clearing.NotTradable},
		{"unknown agent comes first", mut(okBid(), func(f *clearing.Facts) { barred(f); f.AgentKnown = false }), clearing.UnknownAgent},
		{"invalid price comes after", mut(okBid(), func(f *clearing.Facts) { barred(f); f.Price = 0 }), clearing.NotPermitted},
		{"insufficient funds comes after", mut(okBid(), func(f *clearing.Facts) { barred(f); f.Available = 0 }), clearing.NotPermitted},
		{"permitted is admitted", okBid(), clearing.Admitted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := clearing.Admit(tt.in); got != tt.want {
				t.Errorf("Admit(%+v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

// TestAdmitClamp_Rows verifies the legal clamp of spec-institutions.md 6.3.1: a bid above
// the cap is admitted at the cap, an ask below the floor at the floor, an order inside the
// range or on the other side of a bound is unchanged, and the quantity is never touched.
//
// Arithmetic: okBid is 5 units at 10, so with a cap of 8 the bid is admitted at 8 and its
// product is 40; okAsk is 5 units at 10, so with a floor of 12 it is admitted at 12.
//
// Requirement: FR-ORG-030, FR-ORG-032
func TestAdmitClamp_Rows(t *testing.T) {
	t.Parallel()
	mut := func(base clearing.Facts, f func(*clearing.Facts)) clearing.Facts { f(&base); return base }
	tests := []struct {
		name        string
		in          clearing.Facts
		wantPrice   int64
		wantChanged bool
	}{
		{"bid above the cap is lowered", mut(okBid(), func(f *clearing.Facts) { f.Cap = 8 }), 8, true},
		{"bid at the cap is unchanged", mut(okBid(), func(f *clearing.Facts) { f.Cap = 10 }), 10, false},
		{"bid below the cap is unchanged", mut(okBid(), func(f *clearing.Facts) { f.Cap = 12 }), 10, false},
		{"bid ignores the floor", mut(okBid(), func(f *clearing.Facts) { f.Floor = 12 }), 10, false},
		{"ask below the floor is raised", mut(okAsk(), func(f *clearing.Facts) { f.Floor = 12 }), 12, true},
		{"ask at the floor is unchanged", mut(okAsk(), func(f *clearing.Facts) { f.Floor = 10 }), 10, false},
		{"ask ignores the cap", mut(okAsk(), func(f *clearing.Facts) { f.Cap = 8 }), 10, false},
		{"no range is unchanged", okBid(), 10, false},
		{"cap and floor leave a bid at the cap", mut(okBid(), func(f *clearing.Facts) { f.Cap, f.Floor = 8, 6 }), 8, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, price, changed := clearing.AdmitClamp(tt.in)
			if code != clearing.Admitted || price != tt.wantPrice || changed != tt.wantChanged {
				t.Errorf("AdmitClamp(%+v) = %d, %d, %v, want admitted, %d, %v", tt.in, code, price, changed, tt.wantPrice, tt.wantChanged)
			}
		})
	}
}

// TestAdmitClamp_OrderOfChecks verifies that the clamp sits after overflow and the settlement
// bounds and before the funds row (spec-institutions.md 6.3.1): a cap does not rescue an
// order the earlier rows refuse, the funds are tested at the replaced limit, and a refused
// order reports its own limit and no change.
//
// Arithmetic: a bid of 5 units with 30 available fails funds at 10 (50 > 30) and is
// admitted when the cap is 6 (30 <= 30). An ask raised to a floor of 1<<62 for 5 units has
// a product that does not fit int64, so it is refused as an overflow after the clamp.
//
// Requirement: FR-ORG-030
func TestAdmitClamp_OrderOfChecks(t *testing.T) {
	t.Parallel()
	mut := func(base clearing.Facts, f func(*clearing.Facts)) clearing.Facts { f(&base); return base }
	tests := []struct {
		name      string
		in        clearing.Facts
		want      clearing.Code
		wantPrice int64
	}{
		{"invalid price is not rescued by a cap", mut(okBid(), func(f *clearing.Facts) { f.Price, f.Cap = maxPrice+1, 5 }), clearing.InvalidPrice, maxPrice + 1},
		{"overflow is not rescued by a cap", mut(okBid(), func(f *clearing.Facts) {
			f.MaxPrice, f.MaxQuantity, f.Price, f.Qty, f.Cap = 1<<62, 1<<62, 1<<62, 4, 5
		}), clearing.Overflow, 1 << 62},
		{"holding limit is not rescued by a cap", mut(okBid(), func(f *clearing.Facts) { f.Held, f.Cap = maxQty-4, 5 }), clearing.HoldingLimit, 10},
		{"funds fail at the original limit", okBid2(30), clearing.InsufficientFunds, 10},
		{"funds pass at the replaced limit", mut(okBid2(30), func(f *clearing.Facts) { f.Cap = 6 }), clearing.Admitted, 6},
		{"funds still fail below the cap", mut(okBid2(20), func(f *clearing.Facts) { f.Cap = 6 }), clearing.InsufficientFunds, 10},
		{"a floor that overflows is refused", mut(okAsk(), func(f *clearing.Facts) { f.Floor = 1 << 62 }), clearing.Overflow, 10},
		{"not permitted precedes the clamp", mut(okBid(), func(f *clearing.Facts) { f.NotPermitted, f.Cap = true, 5 }), clearing.NotPermitted, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, price, changed := clearing.AdmitClamp(tt.in)
			if code != tt.want || price != tt.wantPrice || (code != clearing.Admitted && changed) {
				t.Errorf("AdmitClamp(%+v) = %d, %d, %v, want %d, %d and no change for a refusal", tt.in, code, price, changed, tt.want, tt.wantPrice)
			}
		})
	}
}

// okBid2 is okBid with the given available money.
func okBid2(available int64) clearing.Facts {
	f := okBid()
	f.Available = available
	return f
}

// TestAdmit_IsAdmitClampWithoutThePrice verifies that Admit returns the code of AdmitClamp
// for facts with a range, so no existing caller of Admit sees a different code.
//
// Guard: Admit is the wrapper every caller and table test uses; if the two diverged, a
// world without the module would be unaffected but one with it would be refused by a row the
// tests of Admit never ran.
func TestAdmit_IsAdmitClampWithoutThePrice(t *testing.T) {
	t.Parallel()
	for _, f := range []clearing.Facts{okBid(), okAsk(), okBid2(30)} {
		for _, rng := range [][2]int64{{0, 0}, {6, 0}, {0, 12}, {6, 12}} {
			f.Cap, f.Floor = rng[0], rng[1]
			want, _, _ := clearing.AdmitClamp(f)
			if got := clearing.Admit(f); got != want {
				t.Errorf("Admit(%+v) = %d, want %d (AdmitClamp's code)", f, got, want)
			}
		}
	}
}
