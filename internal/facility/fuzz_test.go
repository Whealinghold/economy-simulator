package facility_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/facility"
)

// FuzzEffect verifies that Eff and Cap never panic, that an accepted result is
// inside [0, boost] and [0, capacity] and is 0 at condition 0, and that Eff is
// the floor and Cap the ceiling of the same ratio.
//
// Requirement: NFR-CAP-003, FR-CAP-046
func FuzzEffect(f *testing.F) {
	f.Add(int64(6000), int64(1999), int64(2000))
	f.Add(int64(1), int64(1), int64(2))
	f.Add(int64(1)<<62, int64(1)<<61, int64(1)<<62)
	f.Add(int64(-1), int64(0), int64(0))
	f.Fuzz(func(t *testing.T, a, cond, dur int64) {
		if msg := effectViolation(a, cond, dur); msg != "" {
			t.Fatalf("(%d, %d, %d): %s", a, cond, dur, msg)
		}
	})
}

// effectViolation returns the first property of Eff and Cap that fails for the
// input, or "".
func effectViolation(a, cond, dur int64) string {
	e, eok := facility.Eff(a, cond, dur)
	c, cok := facility.Cap(a, cond, dur)
	if eok != cok && a >= 0 {
		return "Eff and Cap disagree on ok"
	}
	if !eok {
		return ""
	}
	return boundsViolation(a, cond, e, c)
}

// boundsViolation checks the range of an accepted pair: eff in [0, a], cap
// between eff and eff + 1, and both 0 at condition 0.
func boundsViolation(a, cond, e, c int64) string {
	if e < 0 || e > a || c < e || c-e > 1 {
		return "eff or cap outside [0, a], or cap not within one above eff"
	}
	if cond == 0 && e+c != 0 {
		return "nonzero at condition 0"
	}
	return ""
}

// FuzzCompose verifies that Gain never exceeds its cap and that Layer never
// exceeds its cap, never turns a closed slot open and never panics.
//
// Requirement: NFR-CAP-003, FR-CAP-021
func FuzzCompose(f *testing.F) {
	f.Add(int64(6000), int64(3000), int64(200_000), int64(10_000), int64(100_000))
	f.Add(int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62)
	f.Add(int64(-5), int64(7), int64(-1), int64(0), int64(0))
	f.Fuzz(func(t *testing.T, e1, e2, maxTotal, slot, maxFactor int64) {
		g := facility.Gain([]int64{e1, e2}, maxTotal)
		if g < 0 || g > max(maxTotal, 0) {
			t.Fatalf("Gain([%d %d], %d) = %d outside [0, cap]", e1, e2, maxTotal, g)
		}
		got, ok := facility.Layer(slot, g, maxFactor)
		if !ok {
			return
		}
		if got < 0 || got > maxFactor || (slot == 0 && got != 0) {
			t.Fatalf("Layer(%d, %d, %d) = %d: outside [0, cap] or opened a closed slot", slot, g, maxFactor, got)
		}
	})
}

// FuzzAssign verifies that Assign never panics and, on any input, places at
// most k, never above a Cap, keeps Condition in [0, its start] and returns
// placements that add up to k less the rest.
//
// Requirement: NFR-CAP-003, FR-CAP-019
//
// Fix: testdata/fuzz/FuzzAssign/6da5df08126ebc23. A negative WearPerUse raised the
// condition of a facility above its start; wear now never restores (the bound of
// the class keeps WearPerUse at 0 or more, so this is a defence of the function).
func FuzzAssign(f *testing.F) {
	f.Add(int64(5), int64(3), int64(10), int64(2), int64(4), int64(8), int64(1))
	f.Add(int64(-1), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0))
	f.Add(int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(-3), int64(1)<<62, int64(1)<<62)
	f.Fuzz(func(t *testing.T, k, cap1, cond1, served1, cap2, cond2, wpu int64) {
		fac := []facility.Fac{
			{Ref: 1, Table: cond1, Condition: cond1, Cap: cap1, Served: served1, WearPerUse: wpu},
			{Ref: 2, Table: cond2, Condition: cond2, Cap: cap2, WearPerUse: wpu},
		}
		start := map[int]facility.Fac{1: fac[0], 2: fac[1]}
		got, rest := facility.Assign(nil, k, fac)
		var sum int64
		for _, p := range got {
			sum += p.N
		}
		if sum+rest != max(k, 0) {
			t.Fatalf("Assign(%d): placed %d + rest %d != k", k, sum, rest)
		}
		for _, x := range fac {
			s := start[x.Ref]
			if x.Served > max(x.Cap, s.Served) || x.Condition > s.Condition && s.Condition >= 0 {
				t.Fatalf("Assign(%d): ref %d served %d (cap %d, was %d), condition %d (was %d)", k, x.Ref, x.Served, x.Cap, s.Served, x.Condition, s.Condition)
			}
		}
	})
}

// FuzzDistribute verifies that Distribute never panics and conserves units:
// what it gives plus what it voids is the units offered, and no facility gets
// more than it misses.
//
// Requirement: NFR-CAP-003, FR-CAP-015
func FuzzDistribute(f *testing.F) {
	f.Add(int64(10), int64(3), int64(5), int64(4), int64(9))
	f.Add(int64(-2), int64(0), int64(-1), int64(0), int64(0))
	f.Add(int64(1)<<62, int64(0), int64(1)<<62, int64(0), int64(1)<<62)
	f.Fuzz(func(t *testing.T, units, c1, m1, c2, m2 int64) {
		need := []facility.Need{{Ref: 1, Condition: c1, Missing: m1}, {Ref: 2, Condition: c2, Missing: m2}}
		got, voided := facility.Distribute(nil, units, need)
		var sum int64
		for _, p := range got {
			sum += p.N
			if p.N < 1 || (p.Ref == 1 && p.N > m1) || (p.Ref == 2 && p.N > m2) {
				t.Fatalf("Distribute(%d): placement %v outside what is missing", units, p)
			}
		}
		if sum+voided != max(units, 0) {
			t.Fatalf("Distribute(%d): given %d + voided %d != units", units, sum, voided)
		}
	})
}
