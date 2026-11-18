package institution_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/institution"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// TestLapse_ListAndScope verifies the lapse of a membership: off the list is
// LapsedList (tested first), out of the scope or in another place than the one of the
// admission is LapsedScope, whatever the new place is, and otherwise the membership
// is kept.
//
// Requirement: FR-ORG-011
func TestLapse_ListAndScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		onList, inScope bool
		admittedAt, at  int32
		want            institution.Lapsed
	}{
		{"eligible and in place", true, true, 2, 2, institution.Keep},
		{"off the list", false, true, 2, 2, institution.LapsedList},
		{"off the list and out of scope counts as the list", false, false, 2, 3, institution.LapsedList},
		{"out of the scope", true, false, 2, 3, institution.LapsedScope},
		{"moved inside the scope", true, true, 2, 3, institution.LapsedScope},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := institution.Lapse(tt.onList, tt.inScope, tt.admittedAt, tt.at); got != tt.want {
				t.Errorf("Lapse(%v, %v, %d, %d) = %d, want %d", tt.onList, tt.inScope, tt.admittedAt, tt.at, got, tt.want)
			}
		})
	}
}

// TestOutcome_LapsedCounts verifies that a lapse adds its household weight to the
// count of its reason and that a kept membership counts nothing.
//
// Requirement: FR-ORG-011
func TestOutcome_LapsedCounts(t *testing.T) {
	t.Parallel()
	var o institution.Outcome
	o.Lapsed(institution.LapsedList, 2)
	o.Lapsed(institution.LapsedScope, 3)
	o.Lapsed(institution.Keep, 5)
	if want := (institution.Outcome{LapsedList: 2, LapsedScope: 3}); o != want {
		t.Errorf("Outcome after three lapses = %+v, want %+v", o, want)
	}
}

// founders returns n candidates of weight 1 and Since 0 with the stateless key of tick t,
// as the round builds them for the founders of a body (spec-institutions.md 13.1).
func founders(n int, tick int64) []institution.Candidate {
	out := make([]institution.Candidate, n)
	for i := range out {
		out[i] = institution.Candidate{ID: int32(i), Weight: 1, Key: ordering.Key(tick, int64(i))}
	}
	return out
}

// TestAdmit_ExampleA verifies example A of spec-institutions.md 13.1: fourteen
// households of the role, twelve places and twelve founders (since 0) and two arrivals
// (since 5 and 9) admit the twelve founders and skip the two; then, one member removed,
// the arrival since 5 is admitted and the one since 9 still waits.
//
// Requirement: FR-ORG-006, FR-ORG-011, FR-ORG-012
func TestAdmit_ExampleA(t *testing.T) {
	t.Parallel()
	cands := founders(14, 0)
	cands[12].Since, cands[13].Since = 5, 9
	free, limited := institution.Headroom(12, 0)
	got, out := institution.Admit(nil, cands, free, !limited, false)
	// Tick 0: "lapsed 0 admitted 12 skipped 2 waiting 2" (admit.out of the spec's model).
	if want := (institution.Outcome{Admitted: 12, Skipped: 2, Waiting: 2}); out != want {
		t.Errorf("tick 0 outcome = %+v, want %+v", out, want)
	}
	if slices.Contains(got, 12) || slices.Contains(got, 13) || len(got) != 12 {
		t.Errorf("tick 0 admitted %v, want the twelve founders and neither arrival", got)
	}
	// Tick 4: eleven members stay after one removal, one place is free, the two arrivals wait.
	left := []institution.Candidate{cands[12], cands[13]}
	free, _ = institution.Headroom(12, 11)
	got, out = institution.Admit(nil, left, free, true, false)
	if want := (institution.Outcome{Admitted: 1, Skipped: 1, Waiting: 1}); out != want {
		t.Errorf("tick 4 outcome = %+v, want %+v", out, want)
	}
	if !slices.Equal(got, []int32{12}) {
		t.Errorf("tick 4 admitted %v, want [12] (the household since 5 before the one since 9)", got)
	}
}

// TestAdmit_Order verifies the total order of admission: Since ascending, then key
// ascending, then the agent ID, so equal keys are separated by the ID.
//
// Requirement: FR-ORG-011
func TestAdmit_Order(t *testing.T) {
	t.Parallel()
	cands := []institution.Candidate{
		{ID: 4, Weight: 1, Since: 3, Key: 1},
		{ID: 3, Weight: 1, Since: 1, Key: 9},
		{ID: 2, Weight: 1, Since: 1, Key: 5},
		{ID: 1, Weight: 1, Since: 1, Key: 5}, // same Since and key as ID 2: the lower ID first
	}
	got, _ := institution.Admit(nil, cands, 0, false, false)
	if want := []int32{1, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("Admit order = %v, want %v (Since, then key, then ID)", got, want)
	}
}

// TestAdmit_FirstFitSkipsCounted verifies that a cohort that does not fit is skipped
// and counted and a smaller one behind it may be admitted: five free places, cohorts
// of 4, 3 and 1 in that order admit the first and the third.
//
// Requirement: FR-ORG-012
func TestAdmit_FirstFitSkipsCounted(t *testing.T) {
	t.Parallel()
	cands := []institution.Candidate{
		{ID: 1, Weight: 4, Since: 0, Key: 1},
		{ID: 2, Weight: 3, Since: 0, Key: 2},
		{ID: 3, Weight: 1, Since: 0, Key: 3},
	}
	got, out := institution.Admit(nil, cands, 5, true, false)
	// 5 - 4 = 1 free; the cohort of 3 does not fit and is skipped; the cohort of 1 fits.
	if want := (institution.Outcome{Admitted: 5, Skipped: 3, Waiting: 3}); out != want || !slices.Equal(got, []int32{1, 3}) {
		t.Errorf("Admit = %v, %+v, want [1 3], %+v", got, out, want)
	}
}

// TestAdmit_BarredAndNoLimit verifies that a place whose need is not met admits
// nobody and counts every candidate Barred and Waiting, and that with no limit
// every candidate is admitted whatever free says.
//
// Requirement: FR-ORG-008, FR-ORG-006
func TestAdmit_BarredAndNoLimit(t *testing.T) {
	t.Parallel()
	got, out := institution.Admit(nil, founders(3, 1), 10, true, true)
	if want := (institution.Outcome{Barred: 3, Waiting: 3}); out != want || len(got) != 0 {
		t.Errorf("barred Admit = %v, %+v, want none, %+v", got, out, want)
	}
	got, out = institution.Admit(nil, founders(3, 1), 0, false, false)
	if want := (institution.Outcome{Admitted: 3}); out != want || len(got) != 3 {
		t.Errorf("unlimited Admit = %v, %+v, want three admitted, %+v", got, out, want)
	}
}

// TestAdmit_NeverAbovePlaces verifies, over every small population, that the
// households admitted never exceed the free places, that the counts add up, that a
// second round with the admitted ones counted as used admits nobody (idempotent), and
// that the result does not depend on the order of the candidates in the slice. It is
// the exhaustive loop of the spec's model (admit.py A1 to A3), not a sample.
//
// Requirement: FR-ORG-006, FR-ORG-019
func TestAdmit_NeverAbovePlaces(t *testing.T) {
	t.Parallel()
	// n candidates, each of weight 1 to maxW, Since 0 or 1, a key that is one of the n!
	// orders, and free places 0 to maxFree.
	for n := 0; n <= 4; n++ {
		exhaust(t, n, 3, 6)
	}
	exhaust(t, 5, 1, 6)
}

// exhaust runs the properties of TestAdmit_NeverAbovePlaces for every population of n
// candidates.
func exhaust(t *testing.T, n int, maxW int64, maxFree int64) {
	t.Helper()
	perms := permutations(n)
	weights := make([]int64, n)
	var each func(i int)
	each = func(i int) {
		if i < n {
			for w := int64(1); w <= maxW; w++ {
				weights[i] = w
				each(i + 1)
			}
			return
		}
		for since := 0; since < 1<<n; since++ {
			for _, perm := range perms {
				cands := make([]institution.Candidate, n)
				for k := range cands {
					cands[k] = institution.Candidate{ID: int32(k), Weight: weights[k], Since: int64(since >> k & 1), Key: uint64(perm[k])}
				}
				for free := int64(0); free <= maxFree; free++ {
					checkProperties(t, cands, free)
				}
			}
		}
	}
	each(0)
}

// checkProperties asserts the properties of one population and one count of free places.
func checkProperties(t *testing.T, cands []institution.Candidate, free int64) {
	t.Helper()
	got, out := institution.Admit(nil, slices.Clone(cands), free, true, false)
	weight := checkCounts(t, cands, free, got, out)
	checkIdempotent(t, cands, free-weight, got)
	checkOrderFree(t, cands, free, got, out)
}

// checkCounts asserts that the admitted weight is at most the free places and that
// the counts add up, and returns the admitted weight.
func checkCounts(t *testing.T, cands []institution.Candidate, free int64, got []int32, out institution.Outcome) int64 {
	t.Helper()
	var total, weight int64
	for _, c := range cands {
		total += c.Weight
		if slices.Contains(got, c.ID) {
			weight += c.Weight
		}
	}
	if weight > free || weight != out.Admitted || out.Admitted+out.Waiting != total || out.Skipped != out.Waiting {
		t.Fatalf("Admit(%v, free %d) = %v, %+v: admitted weight %d, want at most free and counts adding up to %d", cands, free, got, out, weight, total)
	}
	return weight
}

// checkIdempotent asserts that a second round, with the admitted ones now counted as
// used places, admits nobody: a candidate that did not fit cannot fit in fewer places.
func checkIdempotent(t *testing.T, cands []institution.Candidate, left int64, got []int32) {
	t.Helper()
	var rest []institution.Candidate
	for _, c := range cands {
		if !slices.Contains(got, c.ID) {
			rest = append(rest, c)
		}
	}
	if again, o2 := institution.Admit(nil, rest, left, true, false); len(again) != 0 || o2.Admitted != 0 {
		t.Fatalf("second round of %v with %d free admitted %v, want nobody", rest, left, again)
	}
}

// checkOrderFree asserts that the reversed slice and a rotated one admit the same set
// with the same counts.
func checkOrderFree(t *testing.T, cands []institution.Candidate, free int64, got []int32, out institution.Outcome) {
	t.Helper()
	rev := slices.Clone(cands)
	slices.Reverse(rev)
	rot := slices.Clone(cands)
	if len(rot) > 0 {
		rot = append(rot[1:], rot[0])
	}
	for _, other := range [][]institution.Candidate{rev, rot} {
		g2, o2 := institution.Admit(nil, other, free, true, false)
		if !slices.Equal(g2, got) || o2 != out {
			t.Fatalf("Admit(%v, free %d) = %v, %+v differs from %v, %+v of the original order", other, free, g2, o2, got, out)
		}
	}
}

// permutations returns every permutation of 0 to n-1 (one empty permutation for n 0).
func permutations(n int) [][]int {
	if n == 0 {
		return [][]int{{}}
	}
	var out [][]int
	for _, p := range permutations(n - 1) {
		for at := 0; at <= len(p); at++ {
			q := append(append(slices.Clone(p[:at]), n-1), p[at:]...)
			out = append(out, q)
		}
	}
	return out
}

// TestAdmit_OrderOfListIrrelevant verifies, for seeded populations larger than the
// exhaustive loop reaches, that shuffling the candidates changes neither the admitted
// set nor the counts.
//
// Requirement: FR-ORG-019
func TestAdmit_OrderOfListIrrelevant(t *testing.T) {
	t.Parallel()
	for seed := int64(1); seed <= 20; seed++ {
		cands := make([]institution.Candidate, 40)
		for i := range cands {
			k := ordering.Key(seed, int64(i))
			cands[i] = institution.Candidate{ID: int32(i), Weight: int64(k%3) + 1, Since: int64(k >> 8 % 5), Key: ordering.Key(seed, int64(i)+1000)}
		}
		want, wantOut := institution.Admit(nil, slices.Clone(cands), 30, true, false)
		shuffled := slices.Clone(cands)
		for i := range shuffled { // a deterministic permutation: swap with the key-chosen slot
			j := int(ordering.Key(seed, int64(i)+7) % uint64(len(shuffled)))
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		}
		got, gotOut := institution.Admit(nil, shuffled, 30, true, false)
		if !slices.Equal(got, want) || gotOut != wantOut {
			t.Fatalf("seed %d: shuffled Admit = %v, %+v, want %v, %+v", seed, got, gotOut, want, wantOut)
		}
	}
}

// FuzzAdmit checks that Admit never panics, never admits more weight than the free
// places when limited, and counts every candidate once.
func FuzzAdmit(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6}, int64(3), true, false)
	f.Add([]byte{}, int64(0), false, true)
	f.Fuzz(func(t *testing.T, data []byte, free int64, limited, barred bool) {
		var cands []institution.Candidate
		var total int64
		for i := 0; i+2 < len(data); i += 3 {
			w := int64(data[i]%4) + 1
			total += w
			cands = append(cands, institution.Candidate{ID: int32(i / 3), Weight: w, Since: int64(data[i+1] % 3), Key: uint64(data[i+2])})
		}
		got, out := institution.Admit(nil, cands, free, limited, barred)
		if out.Admitted+out.Waiting != total {
			t.Errorf("Admit counts %+v do not add up to %d households", out, total)
		}
		if limited && !barred && out.Admitted > max(free, 0) {
			t.Errorf("Admit admitted %d households with %d free places: %v", out.Admitted, free, got)
		}
	})
}

// TestAdmit_NonPositiveWeightIsNoCandidate verifies that a candidate of weight 0 or less
// is neither admitted nor counted, and does not change the free places of the ones behind.
//
// Guard: a negative weight would raise the free places and the admitted count, and break
// the idempotence of a second round; Headroom and Tally clamp a negative likewise.
func TestAdmit_NonPositiveWeightIsNoCandidate(t *testing.T) {
	t.Parallel()
	cands := []institution.Candidate{
		{ID: 1, Weight: -3, Since: 0, Key: 1},
		{ID: 2, Weight: 0, Since: 0, Key: 2},
		{ID: 3, Weight: 2, Since: 0, Key: 3},
		{ID: 4, Weight: 1, Since: 0, Key: 4},
	}
	got, out := institution.Admit(nil, cands, 2, true, false)
	// Only 3 (2 households) fits in the 2 free places; 4 (1) is skipped, 1 and 2 are ignored.
	if want := (institution.Outcome{Admitted: 2, Skipped: 1, Waiting: 1}); out != want || !slices.Equal(got, []int32{3}) {
		t.Errorf("Admit = %v, %+v, want [3], %+v", got, out, want)
	}
	// A barred place counts the same candidates: 2 + 1 = 3, not 2 + 1 - 3 + 0 = 0.
	_, out = institution.Admit(nil, cands, 2, true, true)
	if want := (institution.Outcome{Barred: 3, Waiting: 3}); out != want {
		t.Errorf("Admit(barred) = %+v, want %+v (weights -3, 0, 2, 1 count 2 + 1)", out, want)
	}
}
