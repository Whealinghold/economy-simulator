package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// FuzzFiscalValidate verifies, for random fields of a tax, a duty, a toll, a payout, a
// purchase and a release laid over a valid configuration, that Validate and Warnings
// never panic, that a configuration Validate accepts hashes the same twice and survives
// the canonical order of its lists, and that a rejected one reports an error.
//
// Requirement: NFR-FIS-003, FR-FIS-012
func FuzzFiscalValidate(f *testing.F) {
	f.Add("poll", "crown", int64(3), int64(300), int64(8_000), "north", "seller")
	f.Add("", "", int64(-1), int64(-1), int64(-1), "", "")
	f.Add("x", "nowhere", int64(1)<<62, int64(1)<<62, int64(1)<<62, "nowhere", "both")
	f.Fuzz(func(t *testing.T, id, authority string, amount, rate, limit int64, region, payer string) {
		c := fiscalTolled()
		m := c.Modules.Fiscal
		m.Taxes[0].ID, m.Taxes[0].Authority, m.Taxes[0].Amount, m.Taxes[0].RateBP, m.Taxes[0].Per = id, authority, amount, rate, payer
		m.Duties[0].Authority, m.Duties[0].RateBP, m.Duties[0].Region, m.Duties[0].Payer = authority, rate, region, payer
		m.Tolls[0].Authority, m.Tolls[0].Amount = authority, amount
		m.Payouts[0].Authority, m.Payouts[0].Amount, m.Payouts[0].Level = authority, amount, rate
		m.Purchases[0].Quantity, m.Purchases[0].LimitBP = amount, limit
		m.Releases[0].Quantity, m.Releases[0].LimitBP = amount, limit
		err := c.Validate()
		_ = c.Warnings() // must not panic, valid or not
		if err != nil {
			return
		}
		first, second := economysim.HashConfig(c), economysim.HashConfig(c)
		if first != second {
			t.Fatalf("HashConfig of an accepted configuration is not stable: %+v", m)
		}
		m.Taxes[0], m.Taxes[2] = m.Taxes[2], m.Taxes[0]
		if c.Validate() != nil {
			t.Fatalf("Validate rejected an accepted configuration with its taxes in another order: %+v", m)
		}
	})
}
