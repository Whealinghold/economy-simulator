package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// FuzzValidateTenure verifies, for random numbers in the rows and the tunables of the
// tenure module of tenureConfig, that Validate and Warnings never panic, that Validate
// is repeatable, and that a configuration it accepts has shares in 1 to 10 000, a census
// sum of the titles of the endowment within 10 000 (spec-land.md 3.1.2) and a band whose
// ceiling is not below its floor.
//
// Requirement: FR-LND-038, FR-LND-039
func FuzzValidateTenure(f *testing.F) {
	f.Add(int64(20), int64(1_000), int64(3), int64(500), int64(2_000), int64(15_000), int64(5_000))
	f.Add(int64(0), int64(0), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1))
	f.Add(int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62)
	f.Add(int64(10_000), int64(1), int64(economysim.MaxPrice), int64(economysim.MaxRentStepBP), int64(10_000), int64(economysim.MaxRentCeilingBP), int64(1))
	f.Fuzz(func(t *testing.T, role, pool, rent, step, floor, ceiling, maxShare int64) {
		c := tenureConfig()
		m := c.Modules.Tenure
		m.Holdings[0].ShareBP, m.Holdings[1].ShareBP = role, pool
		m.Terms[0].Rent, m.RentStepBP, m.RentFloorBP, m.RentCeilingBP, m.MaxHolderShareBP = rent, step, floor, ceiling, maxShare
		_ = c.Warnings()
		err := c.Validate()
		if err != nil {
			return
		}
		if again := c.Validate(); again != nil {
			t.Fatalf("Validate is not repeatable: first nil, then %v, for %+v", again, m)
		}
		for _, h := range m.Holdings {
			if h.ShareBP < 1 || h.ShareBP > 10_000 {
				t.Fatalf("Validate accepted the holding %+v, want a share of 1 to 10 000", h)
			}
		}
		if sum := 253*role + pool; sum > 10_000 {
			t.Fatalf("Validate accepted titles of %d (253 growers at %d and the pool at %d), want at most 10 000", sum, role, pool)
		}
		if m.RentCeilingBP != 0 && m.RentCeilingBP < m.RentFloorBP {
			t.Fatalf("Validate accepted a ceiling %d below the floor %d", m.RentCeilingBP, m.RentFloorBP)
		}
	})
}
