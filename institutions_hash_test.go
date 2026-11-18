package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestInstitutionsHash verifies the configuration hash of the institutions module
// (spec-institutions.md 12.4, spec-config.md 8.2.2): the module off leaves the hash of
// the reference configuration as it was, a Label is not hashed, the lists hash alike in
// whatever order they are written, the default Who of a produce right and its spelling
// hash alike, and a module that is on and empty differs from one that is off.
//
// Requirement: FR-ORG-051, FR-CFG-104
func TestInstitutionsHash(t *testing.T) {
	t.Parallel()
	off := referenceConfig()
	if got := hexHash(off); got != pinnedReferenceHash {
		t.Errorf("HashConfig(reference) = %s, want the pin %s: a module that is off must write nothing", got, pinnedReferenceHash)
	}
	base := hexHash(institutionsConfig())
	empty := referenceConfig()
	empty.Modules.Institutions = &economysim.Institutions{}
	if hexHash(empty) == hexHash(off) {
		t.Errorf("HashConfig of an empty module equals that of the module off: both would be written as nothing")
	}
	relabelled := institutionsConfig()
	m := relabelled.Modules.Institutions
	m.Bodies[0].Label, m.Rights[0].Label, m.Controls[0].Label, m.RateLimits[0].Label = "a", "b", "c", "d"
	if got := hexHash(relabelled); got != base {
		t.Errorf("HashConfig after a Label change = %s, want %s: labels are presentation only", got, base)
	}
	shuffled := institutionsConfig()
	s := shuffled.Modules.Institutions
	s.Rights[0], s.Rights[1] = s.Rights[1], s.Rights[0]
	if got := hexHash(shuffled); got != base {
		t.Errorf("HashConfig with the rights reordered = %s, want %s: the hash is of the canonical order", got, base)
	}
	spelled := institutionsConfig()
	spelled.Modules.Institutions.Rights[0].Who = ""                      // the default of a produce right, written out in the base
	spelled.Modules.Institutions.Rights[1].Who = economysim.WhoPrincipal // the default of any other right, empty in the base
	if got := hexHash(spelled); got != base {
		t.Errorf("HashConfig with the default Who spelled the other way = %s, want %s", got, base)
	}
}

// TestInstitutionsHash_EveryFieldIsHashed verifies that a change of each field of the
// module changes the hash, so no field is left out of the pairs, and that no two
// fields share an id (a change of each gives a different hash).
//
// Requirement: FR-ORG-051
func TestInstitutionsHash_EveryFieldIsHashed(t *testing.T) {
	t.Parallel()
	mutations := map[string]func(m *economysim.Institutions){
		"Body.ID":                  func(m *economysim.Institutions) { m.Bodies[0].ID = "other" },
		"Body.Roles":               func(m *economysim.Institutions) { m.Bodies[0].Roles = []string{"cook"} },
		"Body.Regions":             func(m *economysim.Institutions) { m.Bodies[0].Regions = []string{"farm"} },
		"Body.Jurisdictions":       func(m *economysim.Institutions) { m.Bodies[0].Jurisdictions = []string{"crown"} },
		"Body.Places":              func(m *economysim.Institutions) { m.Bodies[0].Places++ },
		"Body.MaxTrainees":         func(m *economysim.Institutions) { m.Bodies[0].MaxTrainees++ },
		"Body.Needs":               func(m *economysim.Institutions) { m.Bodies[0].Needs = "market" },
		"Right.ID":                 func(m *economysim.Institutions) { m.Rights[0].ID = "other" },
		"Right.Body":               func(m *economysim.Institutions) { m.Rights[0].Body = "" },
		"Right.Activity":           func(m *economysim.Institutions) { m.Rights[0].Activity = economysim.ActivityBuy },
		"Right.Subject":            func(m *economysim.Institutions) { m.Rights[0].Subject = "tool_b" },
		"Right.Region":             func(m *economysim.Institutions) { m.Rights[0].Region = "farm" },
		"Right.Jurisdiction":       func(m *economysim.Institutions) { m.Rights[0].Jurisdiction = "crown" },
		"Right.Who":                func(m *economysim.Institutions) { m.Rights[0].Who = economysim.WhoBoth },
		"Control.ID":               func(m *economysim.Institutions) { m.Controls[0].ID = "other" },
		"Control.Commodity":        func(m *economysim.Institutions) { m.Controls[0].Commodity = "fuel" },
		"Control.Region":           func(m *economysim.Institutions) { m.Controls[0].Region = "north" },
		"Control.Jurisdiction":     func(m *economysim.Institutions) { m.Controls[0].Jurisdiction = "coast" },
		"Control.CapBP":            func(m *economysim.Institutions) { m.Controls[0].CapBP++ },
		"Control.FloorBP":          func(m *economysim.Institutions) { m.Controls[0].FloorBP++ },
		"RateLimit.ID":             func(m *economysim.Institutions) { m.RateLimits[0].ID = "other" },
		"RateLimit.Region":         func(m *economysim.Institutions) { m.RateLimits[0].Region = "north" },
		"RateLimit.Jurisdiction":   func(m *economysim.Institutions) { m.RateLimits[0].Jurisdiction = "coast" },
		"RateLimit.CapBP":          func(m *economysim.Institutions) { m.RateLimits[0].CapBP++ },
		"Institutions.OwnUseBound": func(m *economysim.Institutions) { m.OwnUseBound = true },
		"Institutions.Bodies":      func(m *economysim.Institutions) { m.Bodies = nil },
		"Institutions.Rights":      func(m *economysim.Institutions) { m.Rights = nil },
		"Institutions.Controls":    func(m *economysim.Institutions) { m.Controls = nil },
		"Institutions.RateLimits":  func(m *economysim.Institutions) { m.RateLimits = nil },
	}
	base := hexHash(institutionsConfig())
	seen := map[string]string{base: "the base"}
	for name, mutate := range mutations {
		c := institutionsConfig()
		mutate(c.Modules.Institutions)
		got := hexHash(c)
		if prev, dup := seen[got]; dup {
			t.Errorf("HashConfig after a change of %s = %s, the same as %s", name, got, prev)
		}
		seen[got] = name
	}
}
