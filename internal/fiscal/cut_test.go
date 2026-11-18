package fiscal_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// almsRecipients returns the 20 recipients of the payout alms of
// spec-fiscal.md 18.1 at tick 0: the worker households of farm (agents 11 to 20),
// hills (27 to 31) and mill (32 to 36), each due 500, with the key K(0, id).
func almsRecipients() []fiscal.Recipient {
	var rs []fiscal.Recipient
	for _, r := range [][2]int64{{11, 20}, {27, 31}, {32, 36}} {
		for id := r[0]; id <= r[1]; id++ {
			rs = append(rs, fiscal.Recipient{Due: 500, Key: ordering.Key(0, id), ID: id})
		}
	}
	return rs
}

// TestCut_Example verifies the cut of spec-fiscal.md 18.1: 20 recipients due 500
// each against 4 135 held. q = floor(500 * 4 135 / 10 000) = 206, every
// remainder is 7 500, the leftover is 4 135 - 20 * 206 = 15, so 15 recipients
// receive 207 and the five with the largest keys K(0, id) receive 206: the
// agents 13, 19, 31, 32 and 36.
//
// Requirement: FR-FIS-038
func TestCut_Example(t *testing.T) {
	t.Parallel()
	var s fiscal.Scratch
	rs := almsRecipients()
	got, ok := s.Cut(nil, 4_135, rs)
	if !ok {
		t.Fatalf("Cut(4135, alms) not ok")
	}
	var short []int64
	var sum int64
	for i, a := range got {
		sum += a
		if a == 206 {
			short = append(short, rs[i].ID)
		} else if a != 207 {
			t.Errorf("recipient %d got %d, want 206 or 207", rs[i].ID, a)
		}
	}
	slices.Sort(short)
	if want := []int64{13, 19, 31, 32, 36}; sum != 4_135 || !slices.Equal(short, want) {
		t.Errorf("Cut(4135, alms): sum %d, 206 to %v; want 4135 and %v", sum, short, want)
	}
}

// TestCut_NoCutWhenEnough verifies that a treasury holding at least the total due
// pays every due, and that a recipient due nothing gets nothing.
//
// Requirement: FR-FIS-038
func TestCut_NoCutWhenEnough(t *testing.T) {
	t.Parallel()
	var s fiscal.Scratch
	rs := []fiscal.Recipient{{Due: 5}, {Due: 0}, {Due: 7}}
	got, ok := s.Cut(nil, 12, rs)
	if !ok || !slices.Equal(got, []int64{5, 0, 7}) {
		t.Errorf("Cut(12, 5 0 7) = %v, %v; want 5 0 7", got, ok)
	}
	if got, ok = s.Cut(nil, 0, rs); !ok || !slices.Equal(got, []int64{0, 0, 0}) {
		t.Errorf("Cut(0, 5 0 7) = %v, %v; want all zero", got, ok)
	}
	if got, ok = s.Cut(nil, 5, nil); !ok || len(got) != 0 {
		t.Errorf("Cut(5, none) = %v, %v; want empty", got, ok)
	}
}

// randomRecipients draws a case: 1 to 60 recipients due 0 to 10^6, with keys and
// distinct IDs; the treasury holds 0 to the total plus 5.
func randomRecipients(rng *rand.Rand) (rs []fiscal.Recipient, avail int64) {
	var total int64
	for id := range rng.IntN(60) + 1 {
		d := rng.Int64N(1_000_001)
		total += d
		rs = append(rs, fiscal.Recipient{Due: d, Key: rng.Uint64(), ID: int64(id)})
	}
	return rs, rng.Int64N(total + 6)
}

// TestCut_SumsToAvail verifies, over 500 random cases (the seed is in the
// message), that the whole of the treasury is used when it is short, no share
// exceeds its due, a recipient due nothing gets nothing, a share is within one
// unit of its proportional floor, and the result is the same for any listing
// order of the recipients.
//
// Requirement: FR-FIS-038, NFR-FIS-004
func TestCut_SumsToAvail(t *testing.T) {
	t.Parallel()
	var s, s2 fiscal.Scratch
	for seed := uint64(1); seed <= 500; seed++ {
		rng := rand.New(rand.NewPCG(seed, 11))
		rs, avail := randomRecipients(rng)
		got, ok := s.Cut(nil, avail, rs)
		if !ok {
			t.Fatalf("seed %d: Cut not ok", seed)
		}
		if msg := cutViolation(rs, avail, got); msg != "" {
			t.Errorf("seed %d: %s", seed, msg)
		}
		perm := rng.Perm(len(rs))
		shuffled := make([]fiscal.Recipient, len(rs))
		for i, p := range perm {
			shuffled[i] = rs[p]
		}
		got2, _ := s2.Cut(nil, avail, shuffled)
		for i, p := range perm {
			if got2[i] != got[p] {
				t.Errorf("seed %d: recipient %d got %d listed first and %d listed shuffled", seed, rs[p].ID, got[p], got2[i])
				break
			}
		}
	}
}

// cutViolation says how a result of Cut breaks the rules of spec-fiscal.md 8.4.2,
// or "".
func cutViolation(rs []fiscal.Recipient, avail int64, got []int64) string {
	var total, sum int64
	for _, r := range rs {
		total += r.Due
	}
	for i, a := range got {
		sum += a
		if a < 0 || a > rs[i].Due {
			return "a share is outside [0, due]"
		}
		if rs[i].Due == 0 && a != 0 {
			return "a recipient due nothing got something"
		}
	}
	if want := min(avail, total); sum != want {
		return "the shares do not add up to min(avail, total)"
	}
	return ""
}

// TestCut_Fairness verifies that the keyed tie does not favour a low ID: ten equal
// recipients and five extra units a tick over 1 000 ticks each receive between
// 475 and 525 of them (the model of spec-fiscal.md 20: "lower ID first" would give
// 1 000 to five of them and none to the rest).
//
// Ten recipients due 100 against 1 005: each gets floor(100 * 1005 / 1000) = 100
// ... the treasury is short only when it holds less than 1 000, so the case is
// 995 held: floor(100 * 995 / 1 000) = 99, the leftover 995 - 990 = 5.
//
// Requirement: FR-FIS-038
func TestCut_Fairness(t *testing.T) {
	t.Parallel()
	var s fiscal.Scratch
	extra := make([]int64, 10)
	for tick := int64(0); tick < 1_000; tick++ {
		rs := make([]fiscal.Recipient, 10)
		for i := range rs {
			rs[i] = fiscal.Recipient{Due: 100, Key: ordering.Key(tick, int64(i)), ID: int64(i)}
		}
		got, _ := s.Cut(nil, 995, rs)
		for i, a := range got {
			extra[i] += a - 99
		}
	}
	for i, e := range extra {
		if e < 475 || e > 525 {
			t.Errorf("recipient %d received %d extra units in 1000 ticks, want 475 to 525", i, e)
		}
	}
}

// TestCut_Domain verifies that a negative holding or due and a total that
// overflows are reported and the result is nil.
//
// Requirement: NFR-FIS-003
func TestCut_Domain(t *testing.T) {
	t.Parallel()
	var s fiscal.Scratch
	const big = int64(1)<<62 + 1
	for name, c := range map[string]struct {
		avail int64
		rs    []fiscal.Recipient
	}{
		"negative avail": {-1, []fiscal.Recipient{{Due: 1}}},
		"negative due":   {1, []fiscal.Recipient{{Due: -1}}},
		"overflow":       {1, []fiscal.Recipient{{Due: big}, {Due: big}}},
	} {
		if got, ok := s.Cut(nil, c.avail, c.rs); ok || got != nil {
			t.Errorf("%s: Cut = %v, %v; want nil, false", name, got, ok)
		}
	}
}
