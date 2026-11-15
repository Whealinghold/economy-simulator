package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestTechnologyHash verifies the configuration hash of the technology module
// (spec-technology.md 12.3, spec-config.md 8.2.2): the module off leaves the hash of
// the reference configuration as it was, a Label is not hashed, the fields and the
// recipes hash alike in whatever order they are written, a module that is on and
// empty differs from one that is off, and a change of any number of a field changes
// the hash.
//
// Requirement: FR-TEC-001, FR-CFG-104
func TestTechnologyHash(t *testing.T) {
	t.Parallel()
	off := referenceConfig()
	if got := hexHash(off); got != pinnedReferenceHash {
		t.Errorf("HashConfig(reference) = %s, want the pin %s: a module that is off must write nothing", got, pinnedReferenceHash)
	}
	base := hexHash(technologyConfig())
	if base == hexHash(off) {
		t.Errorf("HashConfig with the module on equals the off hash %s", base)
	}
	empty := referenceConfig()
	empty.Modules.Technology = &economysim.Technology{}
	if hexHash(empty) == hexHash(off) {
		t.Errorf("HashConfig of an empty module equals that of the module off: both would be written as nothing")
	}
	relabelled := technologyConfig()
	relabelled.Modules.Technology.Fields[0].Label = "Husbandry"
	if got := hexHash(relabelled); got != base {
		t.Errorf("HashConfig after a Label change = %s, want %s: the label is presentation only", got, base)
	}
	shuffled := technologyConfig()
	f := shuffled.Modules.Technology.Fields
	f[0], f[1] = f[1], f[0]
	f[1].Recipes = []string{"grow_staple", "gather_seed"}
	if got := hexHash(shuffled); got != base {
		t.Errorf("HashConfig with the fields and recipes reordered = %s, want %s: the hash is of the canonical order", got, base)
	}
}

// TestTechnologyHash_EveryFieldIsHashed verifies that a change of each field of a
// technology field changes the hash, so no field is left out of the pairs, and that
// the ten fields are written under distinct ids (a swap of two values changes it).
//
// Requirement: FR-CFG-104
func TestTechnologyHash_EveryFieldIsHashed(t *testing.T) {
	t.Parallel()
	mutations := map[string]func(f *economysim.TechField){
		"ID":          func(f *economysim.TechField) { f.ID = "tillage" },
		"Recipes":     func(f *economysim.TechField) { f.Recipes = []string{"grow_staple"} },
		"CapBP":       func(f *economysim.TechField) { f.CapBP++ },
		"Base":        func(f *economysim.TechField) { f.Base++ },
		"DoublingBP":  func(f *economysim.TechField) { f.DoublingBP++ },
		"TrendBP":     func(f *economysim.TechField) { f.TrendBP++ },
		"DiscoveryBP": func(f *economysim.TechField) { f.DiscoveryBP++ },
		"StepBP":      func(f *economysim.TechField) { f.StepBP++ },
		"CatchUpBP":   func(f *economysim.TechField) { f.CatchUpBP++ },
		"Reach":       func(f *economysim.TechField) { f.Reach++ },
	}
	base := hexHash(technologyConfig())
	seen := map[string]string{base: "the base"}
	for name, mutate := range mutations {
		c := technologyConfig()
		mutate(&c.Modules.Technology.Fields[0])
		got := hexHash(c)
		if prev, dup := seen[got]; dup {
			t.Errorf("HashConfig after a change of %s = %s, the same as %s", name, got, prev)
		}
		seen[got] = name
	}
	a, b := technologyConfig(), technologyConfig()
	a.Modules.Technology.Fields[0].TrendBP, a.Modules.Technology.Fields[0].DiscoveryBP = 7, 9
	b.Modules.Technology.Fields[0].TrendBP, b.Modules.Technology.Fields[0].DiscoveryBP = 9, 7
	if hexHash(a) == hexHash(b) {
		t.Errorf("swapping TrendBP and DiscoveryBP leaves the hash unchanged: the two share an id")
	}
}
