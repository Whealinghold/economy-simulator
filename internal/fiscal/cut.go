package fiscal

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Recipient is one due of the payouts of an authority and a resource in one tick
// (spec-fiscal.md 8.4.1).
type Recipient struct {
	Due int64  // what the recipient is due, never negative
	Key uint64 // K(t, id) of the account's owner, computed by the caller (spec-fiscal.md 8.4.2)
	ID  int64  // the account ID, the last tie-break
}

// Scratch is the working memory of Cut. The zero value is ready; it grows to the
// largest call and is reused, so a pass allocates nothing once it has grown
// (NFR-FIS-001). A Scratch is not safe for concurrent use.
type Scratch struct {
	rem []remainder
}

type remainder struct {
	frac uint64 // due * avail mod total
	key  uint64
	id   int64
	idx  int
}

// Cut shares avail among the recipients pro rata by what each is due, with the
// largest remainder, and returns the applied amounts appended to dst[:0]
// (spec-fiscal.md 8.4.2). If the dues sum to no more than avail every recipient
// is paid its due. Otherwise applied_i = floor(due_i * avail / total), and the
// avail - sum(applied_i) units left go one each to the recipients in descending
// order of the remainder due_i * avail mod total, ties by ascending Key, then
// ascending ID, then listing index (the last only for two equal accounts, which
// do not occur). The result depends on no listing order, uses the whole of avail,
// gives no recipient more than its due, and gives a recipient due nothing
// nothing. It draws no random number and keeps no state.
//
// ok is false, and the result is nil, for a negative avail or due or a total that
// overflows int64.
func (s *Scratch) Cut(dst []int64, avail int64, rs []Recipient) (applied []int64, ok bool) {
	total, ok := totalDue(rs)
	if !ok || avail < 0 {
		return nil, false
	}
	applied = slices.Grow(dst[:0], len(rs))[:len(rs)]
	if total <= avail {
		for i, r := range rs {
			applied[i] = r.Due
		}
		return applied, true
	}
	s.rem = s.rem[:0]
	var given int64
	for i, r := range rs {
		q, rem, _ := checked.MulDivRem(r.Due, avail, total) // due <= total and avail < total: the quotient fits
		applied[i] = q
		given += q // at most avail
		s.rem = append(s.rem, remainder{frac: uint64(rem), key: r.Key, id: r.ID, idx: i})
	}
	if given == avail {
		return applied, true // nothing left to place: skip the sort
	}
	slices.SortFunc(s.rem, func(a, b remainder) int {
		return cmp.Or(cmp.Compare(b.frac, a.frac), cmp.Compare(a.key, b.key),
			cmp.Compare(a.id, b.id), cmp.Compare(a.idx, b.idx))
	})
	for _, r := range s.rem[:avail-given] { // avail - given < the recipients with a positive remainder
		applied[r.idx]++
	}
	return applied, true
}

// totalDue sums the dues, ok false for a negative due or an overflow.
func totalDue(rs []Recipient) (total int64, ok bool) {
	for _, r := range rs {
		if r.Due < 0 {
			return 0, false
		}
		if total, ok = checked.Add(total, r.Due); !ok {
			return 0, false
		}
	}
	return total, true
}
