package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestHash_TenureOnOff verifies the configuration hash of the tenure module (spec-land.md
// 3.3, spec-config.md 8.2.2): the module off leaves the hash of the reference
// configuration as it was, the module on changes it, a module that is on and empty
// differs from one that is off, and the order of the terms and of the holdings does not
// matter.
//
// Requirement: FR-LND-001, FR-CFG-104
func TestHash_TenureOnOff(t *testing.T) {
	t.Parallel()
	off := referenceConfig()
	if got := hexHash(off); got != pinnedReferenceHash {
		t.Errorf("HashConfig(reference) = %s, want the pin %s: a module that is off must write nothing", got, pinnedReferenceHash)
	}
	base := hexHash(tenureConfig())
	resourcesOnly := tenureConfig()
	resourcesOnly.Modules.Tenure = nil
	if base == hexHash(resourcesOnly) {
		t.Errorf("HashConfig with the module on equals the hash with it off: %s", base)
	}
	empty := tenureConfig()
	empty.Modules.Tenure = &economysim.TenureConfig{}
	if hexHash(empty) == hexHash(resourcesOnly) {
		t.Errorf("HashConfig of an empty module equals that of the module off: both would be written as nothing")
	}
}

// TestHash_TenureListOrderIrrelevant verifies that two spellings of one configuration
// hash alike: the terms and the holdings in another order, the pool before or after the
// roles, and a ceiling and a cap spelled as their defaults or as 0 (spec-config.md 3.4).
//
// Requirement: FR-CFG-104
func TestHash_TenureListOrderIrrelevant(t *testing.T) {
	t.Parallel()
	a := tenureConfig()
	a.Modules.Tenure.Terms = append(a.Modules.Tenure.Terms, economysim.TenureTerm{Region: "north", Resource: "stand", HasClosed: true, Closed: true})
	b := tenureConfig()
	b.Modules.Tenure.Terms = append([]economysim.TenureTerm{{Region: "north", Resource: "stand", HasClosed: true, Closed: true}}, b.Modules.Tenure.Terms...)
	h := b.Modules.Tenure.Holdings
	h[0], h[1] = h[1], h[0]
	if hexHash(a) != hexHash(b) {
		t.Errorf("HashConfig after the lists were reordered = %s, want %s", hexHash(b), hexHash(a))
	}
	zero, spelled := tenureConfig(), tenureConfig()
	zero.Modules.Tenure.RentCeilingBP, zero.Modules.Tenure.MaxHolderShareBP = 0, 0
	spelled.Modules.Tenure.RentCeilingBP, spelled.Modules.Tenure.MaxHolderShareBP = economysim.DefaultRentCeilingBP, economysim.DefaultMaxHolderShareBP
	if hexHash(zero) != hexHash(spelled) {
		t.Errorf("HashConfig of the defaults spelled as 0 = %s, spelled out = %s, want equal", hexHash(zero), hexHash(spelled))
	}
}

// TestHash_TenureEveryFieldIsHashed verifies that a change of each field of the module
// and of its rows changes the hash, so no field is left out of the pairs, and that the
// fields are written under distinct ids (two changes never give one hash).
//
// Requirement: FR-CFG-104
func TestHash_TenureEveryFieldIsHashed(t *testing.T) {
	t.Parallel()
	mutations := map[string]func(m *economysim.TenureConfig){
		"RentStepBP":       func(m *economysim.TenureConfig) { m.RentStepBP++ },
		"RentFloorBP":      func(m *economysim.TenureConfig) { m.RentFloorBP++ },
		"RentCeilingBP":    func(m *economysim.TenureConfig) { m.RentCeilingBP++ },
		"OwnUseExempt":     func(m *economysim.TenureConfig) { m.OwnUseExempt = true },
		"MaxHolderShareBP": func(m *economysim.TenureConfig) { m.MaxHolderShareBP = 5_000 },
		"Term.Region":      func(m *economysim.TenureConfig) { m.Terms[0].Region = "south" },
		"Term.Resource":    func(m *economysim.TenureConfig) { m.Terms[0].Resource = "stand" },
		"Term.HasRent":     func(m *economysim.TenureConfig) { m.Terms[0].HasRent = false },
		"Term.Rent":        func(m *economysim.TenureConfig) { m.Terms[0].Rent++ },
		"Term.HasClosed":   func(m *economysim.TenureConfig) { m.Terms[0].HasClosed = true },
		"Term.Closed":      func(m *economysim.TenureConfig) { m.Terms[0].HasClosed, m.Terms[0].Closed = true, true },
		"Holding.Region":   func(m *economysim.TenureConfig) { m.Holdings[0].Region = "south" },
		"Holding.Resource": func(m *economysim.TenureConfig) { m.Holdings[0].Resource = "stand" },
		"Holding.Role":     func(m *economysim.TenureConfig) { m.Holdings[0].Holder.Role = "cook" },
		"Holding.Kind": func(m *economysim.TenureConfig) {
			m.Holdings[0].Holder = economysim.HolderRef{Kind: economysim.HolderKindPool}
		},
		"Holding.ShareBP": func(m *economysim.TenureConfig) { m.Holdings[0].ShareBP++ },
		"A holding more":  func(m *economysim.TenureConfig) { m.Holdings = m.Holdings[:1] },
		"A term fewer":    func(m *economysim.TenureConfig) { m.Terms = nil },
	}
	base := hexHash(tenureConfig())
	seen := map[string]string{base: "the base"}
	for name, mutate := range mutations {
		c := tenureConfig()
		mutate(c.Modules.Tenure)
		got := hexHash(c)
		if prev, dup := seen[got]; dup {
			t.Errorf("HashConfig after a change of %s = %s, the same as %s", name, got, prev)
		}
		seen[got] = name
	}
	a, b := tenureConfig(), tenureConfig()
	a.Modules.Tenure.RentStepBP, a.Modules.Tenure.RentFloorBP = 7, 9
	b.Modules.Tenure.RentStepBP, b.Modules.Tenure.RentFloorBP = 9, 7
	if hexHash(a) == hexHash(b) {
		t.Errorf("swapping RentStepBP and RentFloorBP leaves the hash unchanged: the two share an id")
	}
}
