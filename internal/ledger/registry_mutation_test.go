package ledger

import "testing"

// TestKindSet_HasIsBoundedByNumKinds verifies that a kind at or past numKinds
// is never a member, even of a set with every bit on.
//
// Guard: the bound k < numKinds must be strict; with k <= numKinds, bit 7 of
// a full set would read as a kind that does not exist.
//
// Equivalent mutants of registry.go (not killable): `k < numKinds` to
// `k != numKinds` (a shift of 8 or more on a uint8 set is 0), `!= 0` to
// `> 0` (unsigned), `1 << KindAgent` to `1 >> KindAgent` (KindAgent is 0), and
// `setAgent | setEstate` to `^` (disjoint bits).
func TestKindSet_HasIsBoundedByNumKinds(t *testing.T) {
	full := kindSet(0xFF)
	for k := KindAgent; k < numKinds; k++ {
		if !full.has(k) {
			t.Errorf("full set has(%d) = false, want true", k)
		}
	}
	for _, k := range []Kind{numKinds, numKinds + 1, 200} {
		if full.has(k) {
			t.Errorf("full set has(%d) = true, want false", k)
		}
	}
	if setHolder.has(KindBoundary) {
		t.Error("setHolder has the boundary kind, want every kind but it")
	}
	if !setHolder.has(KindTreasury) || !setAE.has(KindEstate) || setAE.has(KindEscrow) {
		t.Error("setHolder/setAE membership is wrong")
	}
}
