package economysim

import (
	"errors"
	"math/big"
	"slices"
	"testing"
)

// fuzzShares decodes the shares of a group's jobs from bytes and clamps them to
// the valid range: each byte b is a share of b * 39 basis points (at most 9 945)
// and a share that would take the sum past 10 000 is cut to what is left, so
// every input is a table that Validate accepts.
func fuzzShares(raw []byte) []int64 {
	var shares []int64
	var sum int64
	for _, b := range raw[:min(len(raw), 8)] {
		s := min(int64(b)*39, wholeBP-sum)
		shares = append(shares, s)
		sum += s
	}
	return shares
}

// apportionOracle is the apportionment of apportion written as the spec words
// it and made independent of the order of the entries: the busy count of an
// entry is the floor of its quota plus one when it is among the entries the
// remaining agents go to, which are taken one at a time from the largest
// remainder, a tie to the entry with the smaller key (the idle share has the
// largest key). The caller gives the keys, so a shuffled copy of the entries with
// their original indexes as keys must give the same counts.
func apportionOracle(shares []int64, keys []int, n int64) []int64 {
	all := append(slices.Clone(shares), wholeBP-sumOf(shares))
	allKeys := append(slices.Clone(keys), len(keys))
	got, rem := make([]int64, len(all)), make([]int64, len(all))
	left := n
	for i, s := range all {
		q := new(big.Int).Mul(big.NewInt(s), big.NewInt(n))
		quo, mod := new(big.Int).QuoRem(q, big.NewInt(wholeBP), new(big.Int))
		got[i], rem[i] = quo.Int64(), mod.Int64()
		left -= got[i]
	}
	for ; left > 0; left-- {
		best := -1
		for i := range rem {
			if rem[i] < 0 {
				continue
			}
			if best < 0 || rem[i] > rem[best] || (rem[i] == rem[best] && allKeys[i] < allKeys[best]) {
				best = i
			}
		}
		got[best]++
		rem[best] = -1
	}
	return got[:len(shares)]
}

func sumOf(v []int64) (s int64) {
	for _, x := range v {
		s += x
	}
	return s
}

// FuzzApportion verifies, for shares decoded from bytes and any number of agents
// up to 65 535, that apportion never panics and returns no error for a valid
// table; that the result has one entry per share, none negative; that each count
// is within [floor(quota), floor(quota) + 1]; that the sum never exceeds n, so a
// group never has more busy agents than agents; that a second call gives the same
// vector; and that the vector does not depend on how the entries are listed: the
// oracle run over a shuffled copy of the entries, ties broken by the original
// index (a tie goes to the earlier entry), gives the same counts permuted back
// (FR-VIA-034, O7). A product that does not fit is an error and never a wrap
// (architecture.md 3.5), which valid shares cannot reach.
//
// Requirement: NFR-VIA-004, FR-VIA-034
func FuzzApportion(f *testing.F) {
	f.Add([]byte{131, 11}, uint16(20)) // 5 113 and 418 basis points over 20 agents
	f.Add([]byte{}, uint16(5))
	f.Add([]byte{128, 128}, uint16(1))
	f.Add([]byte{255, 255, 255}, uint16(65535))
	f.Add([]byte{1}, uint16(0))
	f.Fuzz(func(t *testing.T, raw []byte, nRaw uint16) {
		shares, n := fuzzShares(raw), int64(nRaw)
		got, err := apportion(shares, n)
		if err != nil || len(got) != len(shares) {
			t.Fatalf("apportion(%v, %d) = %v, %v, want one count per share and no error", shares, n, got, err)
		}
		var total int64
		for i, c := range got {
			floor := shares[i] * n / wholeBP
			if c < floor || c > floor+1 {
				t.Fatalf("apportion(%v, %d)[%d] = %d, want within [%d, %d] of the quota %d * %d / 10 000", shares, n, i, c, floor, floor+1, shares[i], n)
			}
			total += c
		}
		if total > n {
			t.Fatalf("apportion(%v, %d) = %v, sum %d above the %d agents", shares, n, got, total, n)
		}
		if again, _ := apportion(shares, n); !slices.Equal(got, again) {
			t.Fatalf("apportion(%v, %d) gave %v and then %v, want the same", shares, n, got, again)
		}
		// A deterministic shuffle from the bytes: the entries sorted by a byte key.
		order := make([]int, len(shares))
		for i := range order {
			order[i] = i
		}
		slices.SortStableFunc(order, func(a, b int) int { return int(raw[b]%7) - int(raw[a]%7) })
		shuffled, keys := make([]int64, len(order)), make([]int, len(order))
		for k, i := range order {
			shuffled[k], keys[k] = shares[i], i
		}
		back := make([]int64, len(shares))
		for k, c := range apportionOracle(shuffled, keys, n) {
			back[order[k]] = c
		}
		if !slices.Equal(back, got) {
			t.Fatalf("apportion(%v, %d) = %v, but the oracle over the entries in the order %v gives %v: the result depends on the order of the entries", shares, n, got, order, back)
		}
	})
}

// FuzzStagger verifies, for any agent i of n (n up to 100 000), any period p up
// to 2^32 and any set of start offsets, that phase never panics, lies in [0, p)
// (0 when p is 0), is 0 at i = 0, does not decrease with i and equals floor(i * p
// / n) computed with math/big; that it depends only on (i, n, p), so evaluating
// the agents of a group in a shuffled order gives the same value for each; and
// that for a sorted offset set V the jth of busy agents takes V[floor(j * |V| /
// busy)], a member of V, so the offsets do not decrease with j and two calls give
// the same offsets, with no random number (FR-VIA-034, 037, 040). An empty set is
// ErrWarmStartJob.
//
// Requirement: NFR-VIA-004, FR-VIA-037, FR-VIA-034, FR-VIA-040
func FuzzStagger(f *testing.F) {
	f.Add(uint32(2), uint32(4), uint32(5), []byte{0, 1, 2, 3}, uint8(2))
	f.Add(uint32(0), uint32(1), uint32(0), []byte{}, uint8(1))
	f.Add(uint32(99999), uint32(100000), uint32(4294967295), []byte{7}, uint8(255))
	f.Add(uint32(3), uint32(4), uint32(10), []byte{5, 5, 9, 1}, uint8(3))
	f.Fuzz(func(t *testing.T, iRaw, nRaw, pRaw uint32, offsets []byte, busyRaw uint8) {
		n := 1 + int64(nRaw)%100_000
		i, p := int64(iRaw)%n, int64(pRaw)
		got, err := phase(i, n, p)
		want := new(big.Int).Div(new(big.Int).Mul(big.NewInt(i), big.NewInt(p)), big.NewInt(n))
		if err != nil || got != want.Int64() || got < 0 || (p > 0 && got >= p) || (p == 0 && got != 0) {
			t.Fatalf("phase(%d, %d, %d) = %d, %v, want floor(i * p / n) = %v in [0, p)", i, n, p, got, err, want)
		}
		if z, _ := phase(0, n, p); z != 0 {
			t.Fatalf("phase(0, %d, %d) = %d, want 0", n, p, z)
		}
		// The first agents forward, then the same agents in reverse order: each value
		// is a function of (k, n, p) alone, so the order of evaluation changes nothing,
		// and the forward values do not decrease.
		m := min(n, 64)
		fwd := make([]int64, m)
		for k := range m {
			fwd[k], _ = phase(k, n, p)
			if k > 0 && fwd[k] < fwd[k-1] {
				t.Fatalf("phase(%d, %d, %d) = %d below phase(%d) = %d, want non-decreasing", k, n, p, fwd[k], k-1, fwd[k-1])
			}
		}
		for k := m - 1; k >= 0; k-- {
			if again, _ := phase(k, n, p); again != fwd[k] {
				t.Fatalf("phase(%d, %d, %d) gave %d and then %d in the reverse order, want the same", k, n, p, fwd[k], again)
			}
		}
		fuzzPick(t, offsets, 1+int64(busyRaw)%50)
	})
}

// fuzzPick checks the offsets the jth of busy agents take from the sorted set of
// the distinct bytes.
func fuzzPick(t *testing.T, raw []byte, busy int64) {
	t.Helper()
	var v []int64
	for _, b := range raw {
		v = append(v, int64(b))
	}
	slices.Sort(v)
	v = slices.Compact(v)
	if len(v) == 0 {
		if _, err := pickOffset(v, 0, busy); !errors.Is(err, ErrWarmStartJob) {
			t.Fatalf("pickOffset(empty set) error = %v, want ErrWarmStartJob", err)
		}
		return
	}
	prev := int64(-1)
	for j := range busy {
		o, err := pickOffset(v, j, busy)
		idx := new(big.Int).Div(new(big.Int).Mul(big.NewInt(j), big.NewInt(int64(len(v)))), big.NewInt(busy)).Int64()
		again, _ := pickOffset(v, j, busy)
		if err != nil || o != v[idx] || o != again || o < prev {
			t.Fatalf("pickOffset(%v, %d, %d) = %d, %v, want V[%d] = %d, non-decreasing, and the same twice", v, j, busy, o, err, idx, v[idx])
		}
		prev = o
	}
}
