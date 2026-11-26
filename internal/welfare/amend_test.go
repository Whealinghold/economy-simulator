package welfare_test

import (
	"math"
	"math/rand/v2" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used -- a seeded source for a test
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// ringOf builds a ring of the window holding recs, with the sums of FromRecords.
func ringOf(t *testing.T, window int64, recs ...welfare.Record) welfare.Ring {
	t.Helper()
	r, err := welfare.FromRecords(recs, window, maxW)
	if err != nil {
		t.Fatalf("FromRecords: %v", err)
	}
	return r
}

// TestRing_AmendLast verifies that Amend changes the income and the unearned sum of the
// newest record only, with the running sums following, whatever the position of the start
// of a full ring.
//
// Requirement: FR-INS-064
func TestRing_AmendLast(t *testing.T) {
	t.Parallel()
	recs := []welfare.Record{{Income: 10, Unearned: 1, Cost: 4}, {Income: 20, Unearned: 2, Cost: 5}, {Income: 30, Unearned: 3, Cost: 6}}
	r := ringOf(t, 3, recs...)
	// Push once more so the ring is full and its start has moved: the newest record is not at the end of the slice.
	r.Push(welfare.Record{Income: 40, Unearned: 4, Cost: 7})
	amended, clamped := r.Amend(-15)
	if !amended || clamped {
		t.Fatalf("Amend = %v, %v, want true, false", amended, clamped)
	}
	got := r.AppendRecords(nil)
	want := []welfare.Record{{Income: 20, Unearned: 2, Cost: 5}, {Income: 30, Unearned: 3, Cost: 6}, {Income: 25, Unearned: -11, Cost: 7}}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("record %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	// The running sums equal a ring rebuilt from the records held.
	same := ringOf(t, 3, got...)
	a1, a2 := r.WageParts()
	b1, b2 := same.WageParts()
	if a1 != b1 || a2 != b2 {
		t.Errorf("WageParts after Amend = %d, %d, rebuilt %d, %d", a1, a2, b1, b2)
	}
	ra, _ := r.Ratio(1, 10_000)
	rb, _ := same.Ratio(1, 10_000)
	if ra != rb {
		t.Errorf("Ratio after Amend = %d, rebuilt %d", ra, rb)
	}
}

// TestRing_AmendEmptyRing verifies a ring with no record is not amended.
//
// Requirement: FR-INS-064
func TestRing_AmendEmptyRing(t *testing.T) {
	t.Parallel()
	r := ringOf(t, 5)
	if amended, clamped := r.Amend(100); amended || clamped {
		t.Errorf("Amend of an empty ring = %v, %v, want false, false", amended, clamped)
	}
	if r.Len() != 0 {
		t.Errorf("Len = %d, want 0 (no record written)", r.Len())
	}
	var zero welfare.Ring
	if amended, _ := zero.Amend(1); amended {
		t.Errorf("Amend of the zero ring amended")
	}
}

// TestRing_AmendClampsAndCountsTheBound verifies the amended fields are clamped as Push
// clamps, the count of records at the bound follows both ways, and a saturating sum stays
// inside int64.
//
// Requirement: FR-INS-064
func TestRing_AmendClampsAndCountsTheBound(t *testing.T) {
	t.Parallel()
	r := ringOf(t, 4, welfare.Record{Income: c - 5})
	steps := []struct {
		name        string
		delta       int64
		clamped     bool
		income, unr int64
		atBound     bool
	}{
		{"past the clamp", 100, true, c, 100, true},
		{"below minus the clamp", math.MinInt64, true, -c, -c, true},
		{"back inside", c, false, 0, 0, false},
	}
	for _, st := range steps {
		amended, clamped := r.Amend(st.delta)
		got := r.AppendRecords(nil)[0]
		if !amended || clamped != st.clamped || got.Income != st.income || got.Unearned != st.unr || r.AtBound() != st.atBound {
			t.Errorf("%s: amended %v clamped %v record %+v atBound %v, want clamped %v income %d unearned %d atBound %v",
				st.name, amended, clamped, got, r.AtBound(), st.clamped, st.income, st.unr, st.atBound)
		}
	}
}

// TestRing_AmendEqualsRecomputation verifies over random amendments that the running sums
// of the ring equal the sums of a ring rebuilt from the records it holds (the risk R2 of
// plan 3-ins).
//
// Requirement: FR-INS-064
func TestRing_AmendEqualsRecomputation(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(7, 11)) // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used -- a seeded source for a test
	r := ringOf(t, 6)
	for i := range 200 {
		r.Push(welfare.Record{Income: rng.Int64N(2000) - 1000, Cost: rng.Int64N(500), Unearned: rng.Int64N(100) - 50, Working: rng.IntN(2) == 0})
		if rng.IntN(2) == 0 {
			r.Amend(rng.Int64N(2000) - 1000)
		}
		rebuilt := ringOf(t, 6, r.AppendRecords(nil)...)
		ra, _ := r.Ratio(3, 10_000)
		rb, _ := rebuilt.Ratio(3, 10_000)
		wn, wd := r.WageParts()
		bn, bd := rebuilt.WageParts()
		if ra != rb || wn != bn || wd != bd || r.WorkingTicks() != rebuilt.WorkingTicks() || r.AtBound() != rebuilt.AtBound() {
			t.Fatalf("step %d: ring reads (%d, %d/%d) differ from the rebuilt ring (%d, %d/%d)", i, ra, wn, wd, rb, bn, bd)
		}
	}
}
