package insure_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/insure"
)

// TestFree_RoomOfAnUnderwriter verifies the room an underwriter has for new open
// liability: trunc(capBP * money / 10 000) less the open liability, never below 0, and
// saturated at the limit when the 128-bit quotient passes int64.
//
// Worked cases: 20 000 bp of 1 000 is 2 000, less 500 open is 1 500; 5 000 bp of 999 is
// trunc(499.5) = 499, and 600 open leaves 0; 100 000 bp of 10^18 is 10^19, above int64,
// so it saturates at the limit.
//
// Requirement: FR-INS-012, FR-INS-060
func TestFree_RoomOfAnUnderwriter(t *testing.T) {
	t.Parallel()
	const limit = 1_000_000_000_000_000_000
	tests := []struct {
		name                    string
		capBP, money, open, lim int64
		want                    int64
		ok                      bool
	}{
		{"cap above the whole", 20_000, 1_000, 500, limit, 1_500, true},
		{"truncated", 5_000, 999, 0, limit, 499, true},
		{"open above the cap", 5_000, 999, 600, limit, 0, true},
		{"quotient passes int64 saturates", 100_000, limit, 0, limit, limit, true},
		{"saturates at the limit before the open", 100_000, limit, 5, limit, limit - 5, true},
		{"negative cap", -1, 1, 0, limit, 0, false},
		{"negative money", 1, -1, 0, limit, 0, false},
		{"negative open", 1, 1, -1, limit, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := insure.Free(tt.capBP, tt.money, tt.open, tt.lim)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Free(%d, %d, %d, %d) = %d, %v, want %d, %v", tt.capBP, tt.money, tt.open, tt.lim, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestAtRisk_RuinTestIn128Bits verifies the default insured's test that a cargo is
// large against the household's money: 10 000 * value >= ruinShareBP * money, exact
// at the equality and in 128 bits where the products pass int64.
//
// Worked: value 100 against 50% of 200 is 10^6 against 10^6, at risk; value 99 is not.
// MaxInt64 against 10 000 bp of MaxInt64 is equal in 128 bits, so at risk.
//
// Requirement: FR-INS-061
func TestAtRisk_RuinTestIn128Bits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                     string
		value, ruinShareBP, mony int64
		want                     bool
	}{
		{"equal", 100, 5_000, 200, true},
		{"just below", 99, 5_000, 200, false},
		{"128 bit equal", math.MaxInt64, 10_000, math.MaxInt64, true},
		{"128 bit below", math.MaxInt64 - 1, 10_000, math.MaxInt64, false},
		{"zero share is always at risk", 0, 0, 5, true},
		{"negative value", -1, 0, 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := insure.AtRisk(tt.value, tt.ruinShareBP, tt.mony); got != tt.want {
				t.Errorf("AtRisk(%d, %d, %d) = %v, want %v", tt.value, tt.ruinShareBP, tt.mony, got, tt.want)
			}
		})
	}
}

// TestMatch_ZeroMaximumPayoutIsNotAFill verifies that a pair whose maximum payout
// rounds to 0 is skipped: a policy that can pay nothing is not formed.
//
// Sum 1 at a cover share of 4 000 bp is halfEven(4 000 / 10 000) = 0, so the offer is
// not eligible though its room is 0 or more; the next offer, at 6 000 bp, rounds to 1
// (halfEven(0.6)) and forms.
//
// Requirement: FR-INS-034
func TestMatch_ZeroMaximumPayoutIsNotAFill(t *testing.T) {
	t.Parallel()
	offers := []insure.Offer{
		{Ref: 0, Party: 2, Loading: 100, CoverBP: 4_000, Room: 10},
		{Ref: 1, Party: 3, Loading: 110, CoverBP: 6_000, Room: 10},
	}
	cands := []insure.Candidate{{Ref: 7, Party: 1, Limit: 20_000, MinCover: 1, Sum: 1}}
	var calls []fill
	halted := insure.Match(cands, offers, false, scriptedForm(nil, &calls))
	if halted || len(calls) != 1 || calls[0] != (fill{7, 1}) {
		t.Errorf("Match = halted %v, calls %v, want false, [{7 1}]", halted, calls)
	}
}
