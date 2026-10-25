package ledger

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Moved is one row of the relocation totals of the current tick
// (spec-regions.md 6.4): the money, or the units of one commodity, that moved
// with accounts from one region to another. Commodity is Money for money.
type Moved struct {
	From, To  int
	Commodity int
	Total     int64
}

// CheckRelocate is the dry run of Relocate: it returns the error Relocate would
// return and changes nothing. The checks run in this order: unknown account,
// closed, a kind other than Agent (ErrNotRelocatable), a region outside the
// ledger (ErrUnknownRegion), the region the account is in (ErrSameRegion), a
// reservation of money or an earmark of goods (ErrOpenReservation), and a total
// of the pending relocation totals that would overflow (ErrBalanceOverflow).
func (l *Ledger) CheckRelocate(id, region int) error {
	_, err := l.relocatable(id, region)
	return err
}

// Relocate moves the account of an agent to a region (spec-regions.md 6.1,
// 6.2). It is not a transfer: no balance, lot, reservation, flow total or
// boundary account changes, only the region the account is attributed to, and
// the relocation totals gain the balance and the units of each commodity the
// account holds under (from, to). A refusal, the errors of CheckRelocate,
// changes nothing.
func (l *Ledger) Relocate(id, region int) error {
	a, err := l.relocatable(id, region)
	if err != nil {
		return err
	}
	from := a.region
	a.eachHolding(func(thing int, amount int64) {
		i, found := slices.BinarySearchFunc(l.relocs, [3]int{from, region, thing}, cmpReloc)
		if found {
			l.relocs[i].total += amount
			return
		}
		l.relocs = slices.Insert(l.relocs, i, reloc{from, region, thing, amount})
	})
	a.region = region
	return nil
}

// relocatable returns the account if it can be moved to the region, or the
// first refusal of CheckRelocate.
func (l *Ledger) relocatable(id, region int) (*account, error) {
	a, err := l.open(id)
	switch {
	case err != nil:
		return nil, err
	case a.kind != KindAgent:
		return nil, ErrNotRelocatable
	case region < 0 || region >= l.regions:
		return nil, ErrUnknownRegion
	case region == a.region:
		return nil, ErrSameRegion
	case a.reserved != 0 || a.earmarked():
		return nil, ErrOpenReservation
	}
	ok := true
	a.eachHolding(func(thing int, amount int64) {
		if i, found := slices.BinarySearchFunc(l.relocs, [3]int{a.region, region, thing}, cmpReloc); found {
			_, fits := checked.Add(l.relocs[i].total, amount)
			ok = ok && fits
		}
	})
	if !ok {
		return nil, ErrBalanceOverflow
	}
	return a, nil
}

// eachHolding calls fn for the balance of the account (thing 0), if it is above
// 0, and then for the units of each commodity it holds (thing is the commodity
// plus 1), in commodity order. The lots are in commodity order already
// (spec-ledger.md 5.1.3), and a held quantity is bounded by MaxQuantity, so the
// sum of a commodity cannot overflow.
func (a *account) eachHolding(fn func(thing int, amount int64)) {
	if a.balance > 0 {
		fn(0, a.balance)
	}
	for i := 0; i < len(a.lots); {
		c, n := a.lots[i].Commodity, int64(0)
		for ; i < len(a.lots) && a.lots[i].Commodity == c; i++ {
			n += a.lots[i].Quantity
		}
		fn(c+1, n)
	}
}

// cmpReloc orders a relocation row against a (from, to, thing) key.
func cmpReloc(r reloc, k [3]int) int {
	return cmp.Or(cmp.Compare(r.from, k[0]), cmp.Compare(r.to, k[1]), cmp.Compare(r.thing, k[2]))
}

// Units returns the units of goods the account holds, earmarked ones included,
// so the world can report them in the event of a relocation. It is
// ErrBalanceOverflow if the sum does not fit in int64.
func (l *Ledger) Units(id int) (int64, error) {
	a, err := l.at(id)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, lt := range a.lots {
		var ok bool
		if total, ok = checked.Add(total, lt.Quantity); !ok {
			return 0, ErrBalanceOverflow
		}
	}
	return total, nil
}

// AppendRelocations appends the relocation totals of the current tick to dst
// and returns it: ascending from, then to, then money before the commodities in
// commodity order (spec-regions.md 6.4.1). It allocates only when dst has no
// room. The totals are read-only and enter no identity of Audit, because a
// relocation moves nothing between accounts (6.3).
func (l *Ledger) AppendRelocations(dst []Moved) []Moved {
	for _, r := range l.relocs {
		dst = append(dst, Moved{From: r.from, To: r.to, Commodity: r.thing - 1, Total: r.total})
	}
	return dst
}
