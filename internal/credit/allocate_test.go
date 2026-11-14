package credit_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/credit"
)

// sum returns the sum of a slice, in big integers so that a test of overflow does
// not itself wrap.
func sum(v []int64) *big.Int {
	t := new(big.Int)
	for _, x := range v {
		t.Add(t, big.NewInt(x))
	}
	return t
}

// TestAllocate_ExampleB verifies the shares of spec-credit.md 20.2: a pot of
// 1 001 over claims of 600 and 900 is 400 and 601, a pot of 1 000 is 400 and
// 600, and a payment of 800 over 500 and 700 is 333 and 467.
//
// Requirement: FR-CRD-032
func TestAllocate_ExampleB(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		pay  int64
		owed []int64
		want []int64
	}{
		{"pot 1001", 1_001, []int64{600, 900}, []int64{400, 601}},
		{"pot 1000", 1_000, []int64{600, 900}, []int64{400, 600}},
		{"removal of 800", 800, []int64{500, 700}, []int64{333, 467}},
		{"paid in full", 1_200, []int64{500, 700}, []int64{500, 700}},
		{"nothing paid", 0, []int64{500, 700}, []int64{0, 0}},
		{"no claim", 0, nil, []int64{}},
		{"a claim of nothing among others", 5, []int64{0, 10, 0}, []int64{0, 5, 0}},
		{"nothing owed at all", 0, []int64{0, 0}, []int64{0, 0}},
	}
	var s credit.Scratch
	for _, tt := range tests {
		got, ok := s.Allocate(nil, tt.pay, tt.owed, make([]uint64, len(tt.owed)))
		if !ok || !slices.Equal(got, tt.want) {
			t.Errorf("%s: Allocate(%d, %v) = %v, %v, want %v", tt.name, tt.pay, tt.owed, got, ok, tt.want)
		}
	}
}

// TestAllocate_Ties verifies that equal fractional parts are broken by the key
// ascending and then by the index: three claims of 1 share a payment of 1 and the
// unit goes to the smallest key, and to the smaller index at equal keys.
//
// Requirement: FR-CRD-032
func TestAllocate_Ties(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tie  []uint64
		want []int64
	}{
		{"the smallest key", []uint64{30, 10, 20}, []int64{0, 1, 0}},
		{"equal keys go by index", []uint64{5, 5, 5}, []int64{1, 0, 0}},
		{"key before index", []uint64{9, 7, 7}, []int64{0, 1, 0}},
	}
	var s credit.Scratch
	for _, tt := range tests {
		got, ok := s.Allocate(nil, 1, []int64{1, 1, 1}, tt.tie)
		if !ok || !slices.Equal(got, tt.want) {
			t.Errorf("%s: Allocate with keys %v = %v, %v, want %v", tt.name, tt.tie, got, ok, tt.want)
		}
	}
}

// TestAllocate_Refused verifies the inputs outside the domain: a negative pay or
// owed, a pay above the total, a tie of the wrong length and a sum that overflows.
//
// Requirement: FR-CRD-032, NFR-CRD-005
func TestAllocate_Refused(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		pay  int64
		owed []int64
		tie  []uint64
	}{
		{"negative pay", -1, []int64{5}, []uint64{0}},
		{"negative owed", 1, []int64{5, -1}, []uint64{0, 0}},
		{"pay above the total", 6, []int64{5}, []uint64{0}},
		{"short tie", 1, []int64{5, 5}, []uint64{0}},
		{"long tie", 1, []int64{5, 5}, []uint64{0, 0, 0}},
		{"sum overflows", 1, []int64{math.MaxInt64, 1}, []uint64{0, 0}},
	}
	var s credit.Scratch
	for _, tt := range tests {
		if got, ok := s.Allocate(nil, tt.pay, tt.owed, tt.tie); ok || got != nil {
			t.Errorf("%s: Allocate = %v, %v, want nil, false", tt.name, got, ok)
		}
	}
}

// checkShare reports the first violation of the contract for one share: it is in
// [0, owed], and it is the floor of the exact share or one above it when the exact
// share has a positive remainder.
func checkShare(sh, owed, pay int64, total *big.Int) string {
	if total.Sign() == 0 {
		if sh != 0 {
			return "a share of nothing owed is not 0"
		}
		return ""
	}
	if sh < 0 || sh > owed {
		return "a share is outside [0, owed]"
	}
	exact := new(big.Int).Mul(big.NewInt(pay), big.NewInt(owed))
	fl, m := new(big.Int).DivMod(exact, total, new(big.Int))
	if d := new(big.Int).Sub(big.NewInt(sh), fl).Int64(); d < 0 || d > 1 || (d == 1 && m.Sign() == 0) {
		return "a share is not the floor or the floor plus one of a positive remainder"
	}
	return ""
}

// checkShares reports the first violation of the allocation's contract for one
// random case: the shares sum to the pay and each satisfies checkShare.
func checkShares(pay int64, owed []int64, shares []int64) string {
	if len(shares) != len(owed) {
		return "wrong length"
	}
	if sum(shares).Cmp(big.NewInt(pay)) != 0 {
		return "shares do not sum to the pay"
	}
	total := sum(owed)
	for i, sh := range shares {
		if msg := checkShare(sh, owed[i], pay, total); msg != "" {
			return msg
		}
	}
	return ""
}

// TestAllocate_SumsToPay verifies, over 500 random cases with a seed of their
// own (the failing seed is in the message), that the shares sum to the pay, that
// none exceeds what its claim owes and that each is the exact floor or one unit
// above it.
//
// Requirement: FR-CRD-032, NFR-CRD-005
func TestAllocate_SumsToPay(t *testing.T) {
	t.Parallel()
	var s credit.Scratch
	for seed := uint64(1); seed <= 500; seed++ {
		r := rand.New(rand.NewPCG(seed, 7))
		owed := make([]int64, 1+r.IntN(8))
		tie := make([]uint64, len(owed))
		var total int64
		for i := range owed {
			owed[i] = r.Int64N(1_000_000_000)
			tie[i] = r.Uint64N(4)
			total += owed[i]
		}
		pay := r.Int64N(total + 1)
		shares, ok := s.Allocate(nil, pay, owed, tie)
		if !ok {
			t.Fatalf("seed %d: Allocate(%d, %v) refused", seed, pay, owed)
		}
		if msg := checkShares(pay, owed, shares); msg != "" {
			t.Fatalf("seed %d: Allocate(%d, %v) = %v: %s", seed, pay, owed, shares, msg)
		}
	}
}

// TestAllocate_LargeAmounts verifies the 128-bit intermediate: a pay and a claim
// near MaxMoney give exact shares, not wrapped ones.
//
// Requirement: FR-CRD-032, NFR-CRD-005
func TestAllocate_LargeAmounts(t *testing.T) {
	t.Parallel()
	owed := []int64{maxMoney / 3, maxMoney / 3, maxMoney / 3}
	pay := maxMoney/3*3 - 1
	var s credit.Scratch
	shares, ok := s.Allocate(nil, pay, owed, []uint64{3, 2, 1})
	if !ok {
		t.Fatalf("Allocate refused near MaxMoney")
	}
	if msg := checkShares(pay, owed, shares); msg != "" {
		t.Errorf("Allocate near MaxMoney = %v: %s", shares, msg)
	}
}

// TestAllocate_ReusesDst verifies that the result is appended to dst[:0] and that
// a second call with a grown Scratch allocates nothing.
//
// Requirement: NFR-CRD-001
func TestAllocate_ReusesDst(t *testing.T) {
	var s credit.Scratch
	owed := []int64{600, 900, 7}
	tie := []uint64{1, 2, 3}
	dst := make([]int64, 0, 8)
	got, _ := s.Allocate(dst, 1_001, owed, tie)
	if &got[0] != &dst[:1][0] {
		t.Errorf("Allocate did not append to dst")
	}
	if n := testing.AllocsPerRun(100, func() { _, _ = s.Allocate(dst, 1_001, owed, tie) }); n != 0 {
		t.Errorf("Allocate allocates %v times per run with a grown Scratch, want 0", n)
	}
}

// TestAllocate_StaleDst verifies that the shares of a payment of nothing are 0 even
// when dst holds old values in its capacity, so a reused buffer never leaks a share,
// and that a single claim of 1 is paid in full.
//
// Requirement: FR-CRD-032
func TestAllocate_StaleDst(t *testing.T) {
	t.Parallel()
	var s credit.Scratch
	stale := []int64{9, 9, 9}
	got, ok := s.Allocate(stale[:0], 0, []int64{0, 0}, []uint64{0, 0})
	if !ok || !slices.Equal(got, []int64{0, 0}) {
		t.Errorf("Allocate over a stale buffer = %v, %v, want [0 0]", got, ok)
	}
	if got, ok := s.Allocate(nil, 1, []int64{1}, []uint64{0}); !ok || !slices.Equal(got, []int64{1}) {
		t.Errorf("Allocate(1, [1]) = %v, %v, want [1]", got, ok)
	}
}

// TestAllocate_DstIsOwed verifies that dst may be the very slice of owed: the shares
// are the same as into a separate buffer, owed ends up holding them, and a refusal
// leaves owed untouched.
//
// Requirement: FR-CRD-032
//
// Guard: the caller of a settlement pass may reuse the owed buffer for the shares. The
// total is summed before the first share is written and element i is read before it
// is written, so the call is safe; a change that wrote a share early or read owed
// again after the loop would corrupt every later share.
func TestAllocate_DstIsOwed(t *testing.T) {
	t.Parallel()
	var s credit.Scratch
	tie := []uint64{3, 1, 2, 0}
	want, ok := s.Allocate(nil, 1_001, []int64{600, 900, 7, 400}, tie)
	if !ok {
		t.Fatalf("Allocate into a fresh buffer refused")
	}
	owed := []int64{600, 900, 7, 400}
	got, ok := s.Allocate(owed, 1_001, owed, tie)
	if !ok || !slices.Equal(got, want) || !slices.Equal(owed, want) {
		t.Errorf("Allocate(dst = owed) = %v, %v with owed %v, want %v in both", got, ok, owed, want)
	}
	owed = []int64{5, 5}
	if got, ok := s.Allocate(owed, 11, owed, []uint64{0, 0}); ok || got != nil || !slices.Equal(owed, []int64{5, 5}) {
		t.Errorf("refused Allocate(dst = owed) = %v, %v with owed %v, want nil, false and owed unchanged", got, ok, owed)
	}
}

// TestSplit_InterestFirst verifies the split of a share: interest first, then
// principal, and the refusal of a negative operand.
//
// Requirement: FR-CRD-032
func TestSplit_InterestFirst(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		share, owedI int64
		ip, pp       int64
		ok           bool
	}{
		{"all interest", 40, 50, 40, 0, true},
		{"interest then principal", 333, 50, 50, 283, true},
		{"exactly the interest", 50, 50, 50, 0, true},
		{"nothing", 0, 50, 0, 0, true},
		{"no interest", 10, 0, 0, 10, true},
		{"negative share", -1, 5, 0, 0, false},
		{"negative interest owed", 1, -5, 0, 0, false},
	}
	for _, tt := range tests {
		ip, pp, ok := credit.Split(tt.share, tt.owedI)
		if ip != tt.ip || pp != tt.pp || ok != tt.ok {
			t.Errorf("%s: Split(%d, %d) = %d, %d, %v, want %d, %d, %v", tt.name, tt.share, tt.owedI, ip, pp, ok, tt.ip, tt.pp, tt.ok)
		}
	}
}

// TestFit_Headroom verifies the cut of a payment to the creditor's headroom:
// interest is kept first, the cut is reported, and overflow and negatives are
// refused.
//
// Requirement: FR-CRD-033
func TestFit_Headroom(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		ip, pp, room          int64
		wantIP, wantPP, wantC int64
		ok                    bool
	}{
		{"fits", 10, 90, 100, 10, 90, 0, true},
		{"fits with room to spare", 10, 90, 500, 10, 90, 0, true},
		{"cuts principal first", 10, 90, 60, 10, 50, 40, true},
		{"cuts into interest", 10, 90, 4, 4, 0, 96, true},
		{"no room", 10, 90, 0, 0, 0, 100, true},
		{"nothing to pay", 0, 0, 0, 0, 0, 0, true},
		{"negative room", 1, 1, -1, 0, 0, 0, false},
		{"negative part", -1, 1, 5, 0, 0, 0, false},
		{"sum overflows", math.MaxInt64, 1, 5, 0, 0, 0, false},
	}
	for _, tt := range tests {
		ip, pp, cut, ok := credit.Fit(tt.ip, tt.pp, tt.room)
		if ip != tt.wantIP || pp != tt.wantPP || cut != tt.wantC || ok != tt.ok {
			t.Errorf("%s: Fit(%d, %d, %d) = %d, %d, %d, %v, want %d, %d, %d, %v", tt.name, tt.ip, tt.pp, tt.room, ip, pp, cut, ok, tt.wantIP, tt.wantPP, tt.wantC, tt.ok)
		}
	}
}
