package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// hashOf returns the hash of c.
func hashOf(t *testing.T, c economysim.Config) [32]byte {
	t.Helper()
	return economysim.HashConfig(c)
}

// TestHash_PolicyOnOff verifies FR-PLC-001 and FR-CFG-104 for the module: it is off when
// nil and contributes nothing, on when present even if empty and then changes the hash,
// and an empty module differs from one with an actor.
//
// Requirement: FR-PLC-001, FR-PLC-003
func TestHash_PolicyOnOff(t *testing.T) {
	t.Parallel()
	off := fiscalConfig()
	on := fiscalConfig()
	on.Modules.Policy = &economysim.PolicyConfig{}
	withActor := policyConfig()
	if hashOf(t, off) == hashOf(t, on) {
		t.Error("an empty policy module does not change the hash of a configuration without it")
	}
	if hashOf(t, on) == hashOf(t, withActor) {
		t.Error("a policy module with actors hashes as an empty one")
	}
	again := fiscalConfig()
	if hashOf(t, off) != hashOf(t, again) {
		t.Error("two equal configurations with the module off hash differently")
	}
}

// TestHash_PolicyLabelNotHashed verifies that the label of an actor is presentation
// only: it changes no hash (FR-PLC-003).
//
// Requirement: FR-PLC-003
func TestHash_PolicyLabelNotHashed(t *testing.T) {
	t.Parallel()
	a, b := policyConfig(), policyConfig()
	b.Modules.Policy.Actors[0].Label = "The crown's council"
	if hashOf(t, a) != hashOf(t, b) {
		t.Error("a label moved the hash")
	}
}

// TestHash_PolicyByNameOnly verifies spec-policy.md 3.1.3: the policy is hashed by its
// name, so two names differ and the same name hashes alike, and a spelling of the lists
// in another order hashes alike.
//
// Requirement: FR-PLC-003
func TestHash_PolicyByNameOnly(t *testing.T) {
	t.Parallel()
	a, b := policyConfig(), policyConfig()
	b.Modules.Policy.Actors[1].Policy = "other"
	if hashOf(t, a) == hashOf(t, b) {
		t.Error("a policy name did not change the hash")
	}
	c := policyConfig()
	m := c.Modules.Policy
	m.Actors[0], m.Actors[2] = m.Actors[2], m.Actors[0]
	m.Actors[2].Levers[0], m.Actors[2].Levers[1] = m.Actors[2].Levers[1], m.Actors[2].Levers[0]
	m.Actors[2].Schedule[0], m.Actors[2].Schedule[2] = m.Actors[2].Schedule[2], m.Actors[2].Schedule[0]
	if hashOf(t, a) != hashOf(t, c) {
		t.Error("two spellings of one module hash differently")
	}
}
