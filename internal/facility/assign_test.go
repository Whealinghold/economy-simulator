package facility_test

import (
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/facility"
)

// byRef returns the Fac with the given Ref.
func byRef(fac []facility.Fac, ref int) facility.Fac {
	for _, f := range fac {
		if f.Ref == ref {
			return f
		}
	}
	return facility.Fac{Ref: -1}
}

// TestAssign_MostWornFirst verifies the order of the assignment: ascending
// table condition, ties to the lower Ref, each facility at most its Cap, and
// the wear taken from the running condition to a floor of 0.
//
// Requirement: FR-CAP-019
func TestAssign_MostWornFirst(t *testing.T) {
	t.Parallel()
	fac := []facility.Fac{
		{Ref: 3, Table: 800, Condition: 800, Cap: 2, WearPerUse: 5},
		{Ref: 1, Table: 900, Condition: 900, Cap: 2, WearPerUse: 5},
		{Ref: 2, Table: 800, Condition: 800, Cap: 2, WearPerUse: 5},
	}
	// Order: Ref 2 and 3 tie at 800 (2 first), then Ref 1 at 900. 5 executions:
	// 2 on Ref 2, 2 on Ref 3, 1 on Ref 1.
	got, rest := facility.Assign(nil, 5, fac)
	want := []facility.Placement{{Ref: 2, N: 2}, {Ref: 3, N: 2}, {Ref: 1, N: 1}}
	if !slices.Equal(got, want) || rest != 0 {
		t.Fatalf("Assign(5) = %v, %d, want %v, 0", got, rest, want)
	}
	for ref, wantCond := range map[int]int64{1: 895, 2: 790, 3: 790} {
		if c := byRef(fac, ref).Condition; c != wantCond {
			t.Errorf("Assign: condition of ref %d = %d, want %d", ref, c, wantCond)
		}
	}
}

// TestAssign_CapPerFacility verifies that no facility serves above its Cap and
// that what does not fit is returned as the rest, and that wear floors at 0.
//
// Requirement: FR-CAP-019
func TestAssign_CapPerFacility(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		k        int64
		fac      []facility.Fac
		want     []facility.Placement
		wantRest int64
		wantCond map[int]int64
	}{
		{"over capacity", 9, []facility.Fac{{Ref: 1, Table: 10, Condition: 10, Cap: 3, WearPerUse: 2}, {Ref: 2, Table: 20, Condition: 20, Cap: 3, WearPerUse: 2}},
			[]facility.Placement{{Ref: 1, N: 3}, {Ref: 2, N: 3}}, 3, map[int]int64{1: 4, 2: 14}},
		{"served counts", 3, []facility.Fac{{Ref: 1, Table: 10, Condition: 10, Cap: 3, Served: 2, WearPerUse: 1}, {Ref: 2, Table: 20, Condition: 20, Cap: 3, WearPerUse: 1}},
			[]facility.Placement{{Ref: 1, N: 1}, {Ref: 2, N: 2}}, 0, map[int]int64{1: 9, 2: 18}},
		// 3 executions at 5 each = 15 above the condition 4: floors at 0.
		{"wear floors at zero", 3, []facility.Fac{{Ref: 1, Table: 4, Condition: 4, Cap: 3, WearPerUse: 5}},
			[]facility.Placement{{Ref: 1, N: 3}}, 0, map[int]int64{1: 0}},
		{"negative wear never restores", 2, []facility.Fac{{Ref: 1, Table: 4, Condition: 4, Cap: 3, WearPerUse: -7}},
			[]facility.Placement{{Ref: 1, N: 2}}, 0, map[int]int64{1: 4}},
		{"no wear", 2, []facility.Fac{{Ref: 1, Table: 4, Condition: 4, Cap: 3}},
			[]facility.Placement{{Ref: 1, N: 2}}, 0, map[int]int64{1: 4}},
		{"nothing asked", 0, []facility.Fac{{Ref: 1, Table: 4, Condition: 4, Cap: 3}}, nil, 0, map[int]int64{1: 4}},
		{"negative asked", -4, []facility.Fac{{Ref: 1, Table: 4, Condition: 4, Cap: 3}}, nil, 0, map[int]int64{1: 4}},
		{"no facility", 2, nil, nil, 2, nil},
		{"wear product overflows", 2, []facility.Fac{{Ref: 1, Table: 4, Condition: 4, Cap: 3, WearPerUse: 1 << 62}},
			[]facility.Placement{{Ref: 1, N: 2}}, 0, map[int]int64{1: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, rest := facility.Assign(nil, tt.k, tt.fac)
			if !slices.Equal(got, tt.want) || rest != tt.wantRest {
				t.Errorf("Assign(%d) = %v, %d, want %v, %d", tt.k, got, rest, tt.want, tt.wantRest)
			}
			for ref, c := range tt.wantCond {
				if g := byRef(tt.fac, ref).Condition; g != c {
					t.Errorf("Assign(%d): condition of ref %d = %d, want %d", tt.k, ref, g, c)
				}
			}
		})
	}
}

// TestAssign_ReusesDst verifies that the placements are written into the
// caller's slice, so a start allocates nothing.
//
// Requirement: NFR-CAP-002
func TestAssign_ReusesDst(t *testing.T) {
	t.Parallel()
	dst := make([]facility.Placement, 0, 4)
	fac := []facility.Fac{{Ref: 1, Table: 5, Condition: 5, Cap: 2}}
	got, _ := facility.Assign(dst, 1, fac)
	if len(got) != 1 || &got[0] != &dst[:1][0] {
		t.Errorf("Assign did not reuse dst: %v", got)
	}
}

// TestAssign_SplitInvariant verifies the property of 5.1.3: two starts of one
// tick leave the services and the conditions that one start of their sum
// leaves, over 5 000 random lists (the model of worked.py part 3). A failing
// case prints its seed.
//
// Requirement: FR-CAP-019
func TestAssign_SplitInvariant(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 5000; seed++ {
		r := rand.New(rand.NewPCG(seed, 12))
		fac, total := randomFacs(r)
		n := r.Int64N(total + 1)
		n1 := r.Int64N(n + 1)
		split, one := slices.Clone(fac), slices.Clone(fac)
		_, r1 := facility.Assign(nil, n1, split)
		_, r2 := facility.Assign(nil, n-n1, split)
		_, r3 := facility.Assign(nil, n, one)
		if r1 != 0 || r2 != 0 || r3 != 0 {
			t.Fatalf("seed %d: rest %d, %d, %d for n %d within the room %d", seed, r1, r2, r3, n, total)
		}
		checkSplit(t, seed, fac, split, one)
	}
}

// checkSplit fails the test when the two ways of assigning differ, or when a
// facility serves above its Cap or its condition rises.
func checkSplit(t *testing.T, seed uint64, fac, split, one []facility.Fac) {
	t.Helper()
	for _, f := range fac {
		a, b := byRef(split, f.Ref), byRef(one, f.Ref)
		if a.Served != b.Served || a.Condition != b.Condition {
			t.Fatalf("seed %d: ref %d split served %d cond %d, one start served %d cond %d", seed, f.Ref, a.Served, a.Condition, b.Served, b.Condition)
		}
		if a.Served > f.Cap || a.Condition < 0 || a.Condition > f.Condition {
			t.Fatalf("seed %d: ref %d served %d above cap %d or condition %d outside [0, %d]", seed, f.Ref, a.Served, f.Cap, a.Condition, f.Condition)
		}
	}
}

// randomFacs builds one to six facilities with random conditions and wear, and
// returns them with the sum of their Cap (the most a start can place).
func randomFacs(r *rand.Rand) ([]facility.Fac, int64) {
	n := 1 + r.IntN(6)
	dur := []int64{10, 100, 1000}[r.IntN(3)]
	fac := make([]facility.Fac, n)
	var total int64
	wpu := r.Int64N(dur + 1)
	for i := range fac {
		cond := 1 + r.Int64N(dur)
		c, _ := facility.Cap(1+r.Int64N(8), cond, dur)
		fac[i] = facility.Fac{Ref: i + 1, Table: cond, Condition: cond, Cap: c, WearPerUse: wpu}
		total += c
	}
	r.Shuffle(n, func(i, j int) { fac[i], fac[j] = fac[j], fac[i] })
	return fac, total
}

// TestAssign_TwoMills verifies worked.py part 3c: two mills of cap 3 at table
// conditions 900 and 950, WearPerUse 2, an own start of 3 and a hire of 3 serve
// 3 each and end at 894 and 944.
//
// Requirement: FR-CAP-019
func TestAssign_TwoMills(t *testing.T) {
	t.Parallel()
	fac := []facility.Fac{
		{Ref: 1, Table: 900, Condition: 900, Cap: 3, WearPerUse: 2},
		{Ref: 2, Table: 950, Condition: 950, Cap: 3, WearPerUse: 2},
	}
	for i := 0; i < 2; i++ {
		if _, rest := facility.Assign(nil, 3, fac); rest != 0 {
			t.Fatalf("start %d: rest %d, want 0", i, rest)
		}
	}
	a, b := byRef(fac, 1), byRef(fac, 2)
	if a.Served != 3 || b.Served != 3 || a.Condition != 894 || b.Condition != 944 {
		t.Errorf("two starts: served %d and %d, conditions %d and %d, want 3, 3, 894, 944", a.Served, b.Served, a.Condition, b.Condition)
	}
}

// TestAssign_OracleModel compares Assign with a plain re-expression of the
// model of worked.py (selection of the least table condition, then lower Ref)
// over 1 000 random cases with several starts.
//
// Requirement: FR-CAP-019
func TestAssign_OracleModel(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 1000; seed++ {
		t.Run(strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			r := rand.New(rand.NewPCG(seed, 77))
			fac, total := randomFacs(r)
			model := slices.Clone(fac)
			for start := 0; start < 3; start++ {
				k := r.Int64N(total/2 + 2)
				_, rest := facility.Assign(nil, k, fac)
				if want := modelAssign(model, k); rest != want {
					t.Fatalf("seed %d start %d: rest %d, model %d", seed, start, rest, want)
				}
				for _, m := range model {
					if g := byRef(fac, m.Ref); g.Served != m.Served || g.Condition != m.Condition {
						t.Fatalf("seed %d start %d ref %d: served %d cond %d, model %d %d", seed, start, m.Ref, g.Served, g.Condition, m.Served, m.Condition)
					}
				}
			}
		})
	}
}

// modelAssign is the straightforward form of the model: repeat k times, pick the
// facility with the least table condition (then the lower Ref) that has room.
func modelAssign(fac []facility.Fac, k int64) (rest int64) {
	for ; k > 0; k-- {
		best := -1
		for i, f := range fac {
			if f.Served >= f.Cap {
				continue
			}
			if best < 0 || f.Table < fac[best].Table || (f.Table == fac[best].Table && f.Ref < fac[best].Ref) {
				best = i
			}
		}
		if best < 0 {
			return k
		}
		fac[best].Served++
		fac[best].Condition = max(fac[best].Condition-fac[best].WearPerUse, 0)
	}
	return 0
}
