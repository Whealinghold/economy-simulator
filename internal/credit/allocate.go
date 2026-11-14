package credit

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Scratch is the working memory of Allocate and Waterfall. The zero value is
// ready; it grows to the largest call and is reused, so a pass allocates nothing
// once it has grown (NFR-CRD-001). A Scratch is not safe for concurrent use.
type Scratch struct {
	rem []remainder
}

// remainder is the fractional part of one claim's exact share, with its tie keys.
type remainder struct {
	frac uint64 // pay * owed_i mod total
	key  uint64
	idx  int
}

// Allocate shares pay among the claims that owe owed, pro rata by what each owes,
// with the largest remainder, and returns the shares appended to dst[:0]
// (spec-credit.md 8.3, FR-CRD-032): share_i = floor(pay * owed_i / total) with a
// 128-bit intermediate, and the pay - sum(share_i) units left go one each to the
// claims in descending order of the fractional part pay * owed_i mod total, equal
// parts by tie ascending and then by index. tie has one key per claim. No share
// exceeds owed_i and the shares sum to pay.
//
// ok is false, and the shares are nil, for a negative pay or owed, a tie of a
// different length from owed, a total of the owed that overflows, or a pay above
// the total. A pay of 0 or no claim gives all-zero shares.
func (s *Scratch) Allocate(dst []int64, pay int64, owed []int64, tie []uint64) (shares []int64, ok bool) {
	total, ok := sumOwed(owed)
	if !ok || len(tie) != len(owed) || pay < 0 || pay > total {
		return nil, false
	}
	shares = slices.Grow(dst[:0], len(owed))[:len(owed)]
	if total == 0 {
		clear(shares) // nothing is owed, so nothing is paid (pay <= total)
		return shares, true
	}
	s.rem = s.rem[:0]
	var given int64
	for i, o := range owed {
		q, r, _ := checked.MulDivRem(pay, o, total) // 0 <= pay <= total and o <= total: the quotient fits
		shares[i] = q
		given += q // at most pay
		// A zero remainder is appended too: it sorts last and is never reached,
		// because the units left are fewer than the claims with a positive one.
		s.rem = append(s.rem, remainder{frac: uint64(r), key: tie[i], idx: i})
	}
	slices.SortFunc(s.rem, func(a, b remainder) int {
		return cmp.Or(cmp.Compare(b.frac, a.frac), cmp.Compare(a.key, b.key), cmp.Compare(a.idx, b.idx))
	})
	for _, r := range s.rem[:pay-given] { // pay - given < len(s.rem): see the doc comment
		shares[r.idx]++
	}
	return shares, true
}

// sumOwed returns the sum of owed, ok false for a negative element or an overflow;
// the total of a refused sum is partial and a caller must not use it.
func sumOwed(owed []int64) (total int64, ok bool) {
	for _, o := range owed {
		if o < 0 {
			return total, false
		}
		if total, ok = checked.Add(total, o); !ok {
			return total, false
		}
	}
	return total, true
}

// Split divides a share of a claim into the interest paid and the principal
// paid, interest first: ip = min(share, interestOwed), pp = share - ip
// (spec-credit.md 8.3). A negative share or interestOwed gives (0, 0, false).
func Split(share, interestOwed int64) (interest, principal int64, ok bool) {
	if share < 0 || interestOwed < 0 {
		return 0, 0, false
	}
	ip := min(share, interestOwed)
	return ip, share - ip, true
}

// Fit cuts a payment of interest ip and principal pp to the room the creditor
// has below MaxMoney, interest first (spec-credit.md 8.4, FR-CRD-033), and
// returns the parts kept and the amount cut. A negative part or room gives
// (0, 0, 0, false); the sum of the parts must not overflow.
func Fit(ip, pp, room int64) (interest, principal, cut int64, ok bool) {
	sum, ok := checked.Add(ip, pp)
	if !ok || ip < 0 || pp < 0 || room < 0 {
		return 0, 0, 0, false
	}
	if sum <= room {
		return ip, pp, 0, true
	}
	ipOut := min(ip, room)
	ppOut := room - ipOut // room < sum <= ip + pp, so ppOut < pp
	return ipOut, ppOut, sum - room, true
}
