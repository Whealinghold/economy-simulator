package economysim_test

import (
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// insuranceBoundRow is one bounded field of the module or of a block of
// insuranceConfig: its path, its bounds and the setter that writes a value to it.
type insuranceBoundRow struct {
	path   string
	lo, hi int64
	set    func(c *economysim.Config, v int64)
}

// insuranceBoundRows are the sixteen bounded fields of spec-insurance.md 13 that a
// number can break, with their bounds, over insuranceConfig.
func insuranceBoundRows() []insuranceBoundRow {
	mod := func(f func(m *economysim.Insurance, v int64)) func(c *economysim.Config, v int64) {
		return func(c *economysim.Config, v int64) { f(c.Modules.Insurance, v) }
	}
	under := func(f func(u *economysim.UnderwriteTerms, v int64)) func(c *economysim.Config, v int64) {
		return func(c *economysim.Config, v int64) { f(insuranceRole(c, "cook").Cover.Underwrite, v) }
	}
	insure := func(f func(s *economysim.InsureTerms, v int64)) func(c *economysim.Config, v int64) {
		return func(c *economysim.Config, v int64) { f(insuranceRole(c, "grower").Cover.Insure, v) }
	}
	return []insuranceBoundRow{
		{"modules.insurance.value_cap_bp", 1, economysim.MaxValueCapBP, mod(func(m *economysim.Insurance, v int64) { m.ValueCapBP = v })},
		{"modules.insurance.payout_ticks", 1, economysim.MaxCreditTerm, mod(func(m *economysim.Insurance, v int64) { m.PayoutTicks = v })},
		{"modules.insurance.min_sum", 1, economysim.MaxMoney, mod(func(m *economysim.Insurance, v int64) { m.MinSum = v })},
		{"modules.insurance.max_policies", 1, economysim.MaxOpenPolicies, mod(func(m *economysim.Insurance, v int64) { m.MaxPolicies = v })},
		{"modules.insurance.max_cover_requests", 1, economysim.MaxCoverRequestsBound, mod(func(m *economysim.Insurance, v int64) { m.MaxCoverRequests = v })},
		{"roles.cook.cover.underwrite.loading_bp", 0, economysim.MaxLoadingBound, under(func(u *economysim.UnderwriteTerms, v int64) { u.LoadingBP = v })},
		{"roles.cook.cover.underwrite.store_rate_bp", 0, economysim.MaxStoreRateBound, under(func(u *economysim.UnderwriteTerms, v int64) { u.StoreRateBP = v })},
		{"roles.cook.cover.underwrite.cover_bp", 1, 10_000, under(func(u *economysim.UnderwriteTerms, v int64) { u.CoverBP = v })},
		{"roles.cook.cover.underwrite.cap_bp", 1, economysim.MaxCapBound, under(func(u *economysim.UnderwriteTerms, v int64) { u.CapBP = v })},
		{"roles.cook.cover.underwrite.share_bp", 1, 10_000, under(func(u *economysim.UnderwriteTerms, v int64) { u.ShareBP = v })},
		{"roles.grower.cover.insure.min_loss_bp", 0, 10_000, insure(func(s *economysim.InsureTerms, v int64) { s.MinLossBP = v })},
		{"roles.grower.cover.insure.ruin_share_bp", 0, economysim.MaxRuinBound, insure(func(s *economysim.InsureTerms, v int64) { s.RuinShareBP = v })},
		{"roles.grower.cover.insure.max_loading_bp", 0, economysim.MaxLoadingBound, insure(func(s *economysim.InsureTerms, v int64) { s.MaxLoadingBP = v })},
		{"roles.grower.cover.insure.min_cover_bp", 1, 10_000, insure(func(s *economysim.InsureTerms, v int64) { s.MinCoverBP = v })},
		{"roles.grower.cover.insure.store_share_bp", 0, 10_000, insure(func(s *economysim.InsureTerms, v int64) { s.StoreShareBP = v })},
		{"roles.grower.cover.insure.max_store_rate_bp", 0, economysim.MaxStoreRateBound, insure(func(s *economysim.InsureTerms, v int64) { s.MaxStoreRateBP = v })},
	}
}

// TestValidateInsurance_BoundEdges verifies both edges of every bounded field of the
// module and its blocks: the smallest and the largest valid value validate, and the
// value one step outside refuses with ErrOutOfBounds at the field's path. The field
// cases of TestValidateInsurance_Errors hold one failing side of each field; this
// holds the other side and the accepted edges, so an off-by-one in a bound is caught.
//
// Requirement: FR-INS-010
func TestValidateInsurance_BoundEdges(t *testing.T) {
	t.Parallel()
	for _, row := range insuranceBoundRows() {
		for _, tc := range []struct {
			name string
			v    int64
			ok   bool
		}{{"lo", row.lo, true}, {"hi", row.hi, true}, {"below lo", row.lo - 1, false}, {"above hi", row.hi + 1, false}} {
			t.Run(row.path+" "+tc.name, func(t *testing.T) {
				t.Parallel()
				c := insuranceConfig()
				row.set(&c, tc.v)
				err := c.Validate()
				if tc.ok && err != nil {
					t.Fatalf("Validate(%s = %d) = %v, want nil", row.path, tc.v, err)
				}
				if !tc.ok && !hasInsuranceBoundError(err, row.path) {
					t.Errorf("Validate(%s = %d) = %v, want ErrOutOfBounds at %s", row.path, tc.v, err, row.path)
				}
			})
		}
	}
}

// hasInsuranceBoundError reports whether err holds an ErrOutOfBounds at the path.
func hasInsuranceBoundError(err error, p string) bool {
	for _, e := range unwrapAll(err) {
		var ve *economysim.ValidationError
		if errors.As(e, &ve) && errors.Is(e, economysim.ErrOutOfBounds) && strings.Join(ve.Path, ".") == p {
			return true
		}
	}
	return false
}

// insuranceFuzzConfig sets the sixteen bounded fields of insuranceConfig from the
// bytes b, eight bytes each in the order of insuranceBoundRows; a short b is padded
// with 0.
func insuranceFuzzConfig(b []byte) economysim.Config {
	rows := insuranceBoundRows()
	b = append(b, make([]byte, 8*len(rows))...)
	c := insuranceConfig()
	for i, row := range rows {
		row.set(&c, int64(binary.BigEndian.Uint64(b[8*i:])))
	}
	return c
}

// FuzzValidateInsurance verifies, for any values of the module and of an underwriter
// and an insured block, that Validate and Warnings never panic, that an accepted
// configuration is within every bound of spec-insurance.md 13 and that its hash is
// stable.
//
// Requirement: FR-INS-010, FR-INS-011
func FuzzValidateInsurance(f *testing.F) {
	seed := func(vs ...int64) []byte {
		var b []byte
		for _, v := range vs {
			b = binary.BigEndian.AppendUint64(b, uint64(v))
		}
		return b
	}
	f.Add(seed(20_000, 1, 100, 1_000, 4, 12_000, 50, 8_000, 5_000, 5_000, 1_000, 3_000, 15_000, 2_000, 5_000, 200))
	f.Add(seed(0, -1, 1<<62, 0, -1, 0, 0, 0, -1, 0, -1, 1<<62, 0, 0, -1, 0))
	f.Fuzz(func(t *testing.T, b []byte) {
		c := insuranceFuzzConfig(b)
		err := c.Validate()
		_ = c.Warnings()
		if err == nil {
			assertInsuranceInBounds(t, c)
		}
		if first, second := economysim.HashConfig(c), economysim.HashConfig(c); first != second {
			t.Fatalf("HashConfig is not stable")
		}
	})
}

// assertInsuranceInBounds fails when an accepted configuration holds a field of
// insuranceBoundRows outside its bound, by reading the fields back.
func assertInsuranceInBounds(t *testing.T, c economysim.Config) {
	t.Helper()
	m, u, s := c.Modules.Insurance, insuranceRole(&c, "cook").Cover.Underwrite, insuranceRole(&c, "grower").Cover.Insure
	got := []int64{m.ValueCapBP, m.PayoutTicks, m.MinSum, m.MaxPolicies, m.MaxCoverRequests,
		u.LoadingBP, u.StoreRateBP, u.CoverBP, u.CapBP, u.ShareBP,
		s.MinLossBP, s.RuinShareBP, s.MaxLoadingBP, s.MinCoverBP, s.StoreShareBP, s.MaxStoreRateBP}
	for i, row := range insuranceBoundRows() {
		if got[i] < row.lo || got[i] > row.hi {
			t.Fatalf("Validate accepted %s = %d outside %d to %d", row.path, got[i], row.lo, row.hi)
		}
	}
}
