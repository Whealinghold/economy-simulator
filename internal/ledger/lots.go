package ledger

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Lot is goods of one commodity in an account that share a born tick and a
// remaining durability (spec-ledger.md 5.1.1). The first four fields are the
// key, and Reserved is the units earmarked by a reservation (6.4), at most
// Quantity and not part of the key.
type Lot struct {
	// Commodity is the commodity index.
	Commodity int
	// Quantity is at least 1.
	Quantity int64
	// Born is the tick of creation, signed: 0 for a commodity that does not
	// spoil, negative only for a lot created at construction (5.2.4).
	Born int64
	// Durability is the remaining durability, 0 for a commodity that does not
	// wear.
	Durability int64
	// Reserved is the units earmarked, 0 between ticks.
	Reserved int64
}

// Key names a lot: commodity, born tick and remaining durability (6.7).
type Key struct {
	Commodity  int
	Born       int64
	Durability int64
}

// cmpLot orders a lot against a key by (commodity, born, durability), the
// canonical order of 5.1.3.
func cmpLot(lt Lot, k Key) int {
	if c := cmp.Compare(lt.Commodity, k.Commodity); c != 0 {
		return c
	}
	if c := cmp.Compare(lt.Born, k.Born); c != 0 {
		return c
	}
	return cmp.Compare(lt.Durability, k.Durability)
}

func (lt Lot) key() Key { return Key{lt.Commodity, lt.Born, lt.Durability} }

// find returns where the lot with key k is or would be inserted.
func (a *account) find(k Key) (int, bool) { return slices.BinarySearchFunc(a.lots, k, cmpLot) }

// span returns the half-open range of the lots of commodity c. They are
// contiguous, and in canonical order within the range: oldest first, then
// lowest durability (5.2.1).
func (a *account) span(c int) (lo, hi int) {
	lo, _ = slices.BinarySearchFunc(a.lots, c, func(lt Lot, c int) int { return cmp.Compare(lt.Commodity, c) })
	for hi = lo; hi < len(a.lots) && a.lots[hi].Commodity == c; hi++ {
	}
	return lo, hi
}

// held is the total quantity of commodity c in a, earmarked units included.
// A non-boundary account holds at most MaxQuantity of it, enforced on every
// credit, so the sum cannot overflow.
func (a *account) held(c int) (n int64) {
	lo, hi := a.span(c)
	for _, lt := range a.lots[lo:hi] {
		n += lt.Quantity
	}
	return n
}

// earmarkedOf is the earmarked part of held.

func (a *account) earmarkedOf(c int) (n int64) {
	lo, hi := a.span(c)
	for _, lt := range a.lots[lo:hi] {
		n += lt.Reserved
	}
	return n
}

// earmarked reports whether any lot of the account has an earmark.
func (a *account) earmarked() bool {
	for _, lt := range a.lots {
		if lt.Reserved != 0 {
			return true
		}
	}
	return false
}

// add credits n units under key k to a, merging into an equal key (5.1.3), and
// lowers the account's expiry hint if the lot expires before it (5.3.2). Every
// operation that adds a lot goes through it.
func (l *Ledger) add(a *account, k Key, n int64) {
	i, ok := a.find(k)
	if ok {
		a.lots[i].Quantity += n
	} else {
		a.lots = slices.Insert(a.lots, i, Lot{Commodity: k.Commodity, Quantity: n, Born: k.Born, Durability: k.Durability})
	}
	if s := l.comms[k.Commodity].ShelfLife; s > 0 {
		a.noteExpiry(expiry(k.Born, s))
	}
}

// take removes n units from lot i, from its earmark if marked, and returns the
// key. The lot may be left at 0; the caller compacts.
func (a *account) take(i int, n int64, marked bool) Key {
	lt := &a.lots[i]
	lt.Quantity -= n
	if marked {
		lt.Reserved -= n
	}
	return lt.key()
}

// AppendLots appends a copy of the lots of account id, in canonical order, to
// dst and returns it. A closed account and a boundary have none.
func (l *Ledger) AppendLots(dst []Lot, id int) ([]Lot, error) {
	a, err := l.at(id)
	if err != nil {
		return dst, err
	}
	return append(dst, a.lots...), nil
}

// goodsMove is one Move, in its four forms.
type goodsMove struct {
	from, to  int
	commodity int
	qty       int64
	r         Reason
	// marked draws on earmarks (4.3, 6.3).
	marked bool
	// named takes exactly the lot key (6.7).
	named bool
	// stated credits under key instead of the creation defaults (5.2.4).
	stated bool
	key    Key
}

// Move moves qty units of a commodity from one account to another under a
// reason, taking lots in canonical order: oldest first, then lowest remaining
// durability; the moved lots keep their born tick and durability (spec-ledger.md
// 5.2.1). It takes only unearmarked units. A move from a boundary is a
// creation: the new lot is born in the current tick if the commodity spoils and
// has full durability if it wears, and the boundary goes more negative (5.2.3).
//
// The checks run in a fixed order, and the first that holds is returned; a
// refusal changes nothing (2.6):
//
//  1. ErrInvalidAmount, if qty < 1;
//  2. ErrUnknownAccount, then ErrClosedAccount, for from and then to;
//  3. ErrUnknownCommodity;
//  4. ErrReasonMismatch, as for Transfer but for goods;
//  5. ErrInsufficientStock;
//  6. ErrBalanceOverflow, in to (above MaxQuantity, or int64 for a boundary),
//     the debited boundary, or the flow total.
//
// The flow is attributed as Transfer's is and counted in units.
func (l *Ledger) Move(from, to, commodity int, qty int64, r Reason) error {
	return l.move(goodsMove{from: from, to: to, commodity: commodity, qty: qty, r: r})
}

// MoveReserved is Move marked "from reserved" (4.3, 6.3): it takes earmarked
// units, in canonical order, lowering the earmark and the holding together. It
// is ErrBoundaryReserve for a boundary source, and ErrReservationExceeds, in
// place of ErrInsufficientStock, when fewer units are earmarked than qty.
func (l *Ledger) MoveReserved(from, to, commodity int, qty int64, r Reason) error {
	return l.move(goodsMove{from: from, to: to, commodity: commodity, qty: qty, r: r, marked: true})
}

// MoveLot is Move of exactly the named lot (6.7). It is ErrUnknownLot if the
// account holds no such lot, and ErrInsufficientStock if the lot has fewer
// than qty unearmarked units. A boundary holds no lot.
func (l *Ledger) MoveLot(from, to int, key Key, qty int64, r Reason) error {
	return l.move(goodsMove{from: from, to: to, commodity: key.Commodity, qty: qty, r: r, named: true, key: key})
}

// MoveLotReserved is MoveLot marked from reserved: ErrReservationExceeds when
// the lot has fewer than qty earmarked units.
func (l *Ledger) MoveLotReserved(from, to int, key Key, qty int64, r Reason) error {
	return l.move(goodsMove{from: from, to: to, commodity: key.Commodity, qty: qty, r: r, marked: true, named: true, key: key})
}

// MoveStated is a creation under Initial that states the lot key (5.2.4):
// from must be the boundary account of Initial for the key's commodity. After
// the checks of Move up to the reason, which also refuses any other reason or
// source with ErrReasonMismatch, it returns ErrNotConstruction outside the
// construction phase and then ErrInvalidLot for a key outside the ranges of
// FR-LED-043: for a spoiling commodity Born from -(ShelfLife-1) to 0, for a
// durable a durability from 1 to the full durability, and 0 for the field of a
// commodity that does not spoil or wear.
func (l *Ledger) MoveStated(from, to int, key Key, qty int64, r Reason) error {
	return l.move(goodsMove{from: from, to: to, commodity: key.Commodity, qty: qty, r: r, stated: true, key: key})
}

// validStated reports whether a stated key is inside the ranges of 5.2.4.
func (l *Ledger) validStated(k Key) bool {
	c := l.comms[k.Commodity]
	if c.ShelfLife > 0 {
		if k.Born > 0 || k.Born < -(c.ShelfLife-1) {
			return false
		}
	} else if k.Born != 0 {
		return false
	}
	if c.Durability > 0 {
		return k.Durability >= 1 && k.Durability <= c.Durability
	}
	return k.Durability == 0
}

// creationKey is the key of a lot created in the current tick (5.2.3).
func (l *Ledger) creationKey(c int) Key {
	k := Key{Commodity: c, Durability: l.comms[c].Durability}
	if l.comms[c].ShelfLife > 0 {
		k.Born = l.tick
	}
	return k
}

// move runs a goods move: the checks in order, then the flow total, then the
// effects, which cannot fail.
func (l *Ledger) move(m goodsMove) error {
	f, t, err := l.checkMove(m)
	if err != nil {
		return err
	}
	newFrom, err := debitCheck(f, m)
	if err != nil {
		return err
	}
	newTo, err := l.creditCheck(t, m)
	if err != nil {
		return err
	}
	slot := l.flowSlot(f, t, l.off[m.r][thingGoods]+m.commodity)
	flows := l.flows()
	total, ok := checked.Add(flows[slot], m.qty)
	if !ok {
		return ErrBalanceOverflow
	}

	// Nothing below can fail.
	flows[slot] = total
	if f.kind == KindBoundary {
		f.balance = newFrom
		l.add(t, l.moveKey(m), m.qty)
		return nil
	}
	if t.kind == KindBoundary {
		t.balance = newTo
	}
	l.takeLots(f, t, m)
	return nil
}

// moveKey is the key of the lot a boundary source creates: the stated key, or
// the key of the current tick.
func (l *Ledger) moveKey(m goodsMove) Key {
	if m.stated {
		return m.key
	}
	return l.creationKey(m.commodity)
}

// checkMove runs the argument, account, commodity, reason and stated-key
// checks of move, in order.
func (l *Ledger) checkMove(m goodsMove) (f, t *account, err error) {
	if m.qty < 1 {
		return nil, nil, ErrInvalidAmount
	}
	if f, t, err = l.parties(m.from, m.to); err != nil {
		return nil, nil, err
	}
	if m.commodity < 0 || m.commodity >= len(l.comms) {
		return nil, nil, ErrUnknownCommodity
	}
	if err = l.checkReason(m, f, t); err != nil {
		return nil, nil, err
	}
	if m.stated {
		if err = l.checkStated(m, f); err != nil {
			return nil, nil, err
		}
	}
	return f, t, nil
}

// checkReason refuses a reason that does not fit the accounts (4.2).
func (l *Ledger) checkReason(m goodsMove, f, t *account) error {
	rw, ok := rowOf(m.r, true)
	if !ok || m.from == m.to || !rw.fits(f.kind, t.kind) {
		return ErrReasonMismatch
	}
	if !boundariesFit(f, t, m.r, m.commodity) {
		return ErrReasonMismatch
	}
	return nil
}

// checkStated runs the extra checks of MoveStated.
func (l *Ledger) checkStated(m goodsMove, f *account) error {
	switch {
	case f.kind != KindBoundary || m.r != Initial:
		return ErrReasonMismatch
	case !l.construction:
		return ErrNotConstruction
	case !l.validStated(m.key):
		return ErrInvalidLot
	}
	return nil
}

// debitCheck checks the source of a move has the units and returns the new
// balance of a boundary source.
func debitCheck(f *account, m goodsMove) (newFrom int64, err error) {
	switch {
	case f.kind == KindBoundary:
		switch {
		case m.marked:
			return 0, ErrBoundaryReserve
		case m.named:
			return 0, ErrUnknownLot
		}
		n, ok := checked.Sub(f.balance, m.qty)
		if !ok {
			return 0, ErrBalanceOverflow
		}
		return n, nil
	case m.named:
		i, found := f.find(m.key)
		if !found {
			return 0, ErrUnknownLot
		}
		return 0, lotShort(f.lots[i].Quantity, f.lots[i].Reserved, m.qty, m.marked)
	}
	return 0, lotShort(f.held(m.commodity), f.earmarkedOf(m.commodity), m.qty, m.marked)
}

// creditCheck checks the destination of a move can take the units and returns
// the new balance of a boundary destination.
func (l *Ledger) creditCheck(t *account, m goodsMove) (newTo int64, err error) {
	if t.kind == KindBoundary {
		n, ok := checked.Add(t.balance, m.qty)
		if !ok {
			return 0, ErrBalanceOverflow
		}
		return n, nil
	}
	if n, ok := checked.Add(t.held(m.commodity), m.qty); !ok || n > l.maxQty {
		return 0, ErrBalanceOverflow
	}
	return 0, nil
}

// takeLots moves the units out of the source lots in canonical order and
// credits the destination, then drops emptied lots.
func (l *Ledger) takeLots(f, t *account, m goodsMove) {
	remaining := m.qty
	emptied := false
	lo, hi := f.takeSpan(m)
	for i := lo; i < hi && remaining > 0; i++ {
		n := min(f.lots[i].takeable(m.marked), remaining)
		if n == 0 {
			continue
		}
		key := f.take(i, n, m.marked)
		remaining -= n
		emptied = emptied || f.lots[i].Quantity == 0
		if t.kind != KindBoundary {
			l.add(t, key, n)
		}
	}
	if emptied {
		f.lots = slices.DeleteFunc(f.lots, func(lt Lot) bool { return lt.Quantity == 0 })
	}
}

// takeSpan is the range of lots a move takes from: the named lot, or every lot
// of the commodity.
func (a *account) takeSpan(m goodsMove) (lo, hi int) {
	if m.named {
		lo, _ = a.find(m.key)
		return lo, lo + 1
	}
	return a.span(m.commodity)
}

// takeable is the units of the lot a take can use: the earmarked units for a
// marked take, else the unearmarked ones.
func (lt Lot) takeable(marked bool) int64 {
	if marked {
		return lt.Reserved
	}
	return lt.Quantity - lt.Reserved
}

// lotShort returns the refusal for taking qty from a holding of held units of
// which reserved are earmarked, or nil. An unmarked take is limited by the
// unearmarked units, a marked one by the earmark.
func lotShort(held, reserved, qty int64, marked bool) error {
	switch {
	case marked && reserved < qty:
		return ErrReservationExceeds
	case !marked && held-reserved < qty:
		return ErrInsufficientStock
	}
	return nil
}

// ReserveGoods earmarks qty units of a commodity in an account, in canonical
// order, oldest first, so that the units a later marked move takes are the
// ones earmarked (spec-ledger.md 6.1, 6.4). It is refused with
// ErrInvalidAmount, ErrUnknownAccount or ErrClosedAccount, ErrUnknownCommodity,
// ErrBoundaryReserve, and ErrInsufficientStock when more than the available
// (unearmarked) units are asked.
func (l *Ledger) ReserveGoods(id, commodity int, qty int64) error {
	return l.reserve(reservation{id: id, commodity: commodity, qty: qty})
}

// ReleaseGoods returns qty earmarked units of a commodity to available (6.2).
// The units released are those earmarked last in canonical order, so a
// reserve of n followed by a release of m leaves what a reserve of n-m would
// have. More than is earmarked is ErrReservationExceeds.
func (l *Ledger) ReleaseGoods(id, commodity int, qty int64) error {
	return l.reserve(reservation{id: id, commodity: commodity, qty: qty, release: true})
}

// ReserveLot is ReserveGoods on exactly the named lot (6.7): ErrUnknownLot if
// there is no such lot, ErrInsufficientStock if it has too few available.
func (l *Ledger) ReserveLot(id int, key Key, qty int64) error {
	return l.reserve(reservation{id: id, commodity: key.Commodity, qty: qty, named: true, key: key})
}

// ReleaseLot is ReleaseGoods on exactly the named lot: ErrUnknownLot if there
// is no such lot, ErrReservationExceeds if it has fewer units earmarked.
func (l *Ledger) ReleaseLot(id int, key Key, qty int64) error {
	return l.reserve(reservation{id: id, commodity: key.Commodity, qty: qty, named: true, key: key, release: true})
}

// reservation is one ReserveGoods, ReleaseGoods, ReserveLot or ReleaseLot:
// qty units of commodity in account id, on the named lot key if named, and
// released rather than earmarked if release.
type reservation struct {
	id, commodity int
	qty           int64
	key           Key
	named         bool
	release       bool
}

// reserve runs a reservation: the checks in order, then the earmark.
func (l *Ledger) reserve(r reservation) error {
	a, err := l.checkReserve(r)
	if err != nil {
		return err
	}
	if r.named {
		return reserveLot(a, r.key, r.qty, r.release)
	}
	lo, hi := a.span(r.commodity)
	if err := lotShort(a.held(r.commodity), a.earmarkedOf(r.commodity), r.qty, r.release); err != nil {
		return err
	}
	a.earmark(lo, hi, r.qty, r.release)
	return nil
}

// checkReserve runs the amount, account, commodity and boundary checks of a
// reservation, in order, and returns the account.
func (l *Ledger) checkReserve(r reservation) (*account, error) {
	if r.qty < 1 {
		return nil, ErrInvalidAmount
	}
	a, err := l.open(r.id)
	if err != nil {
		return nil, err
	}
	if r.commodity < 0 || r.commodity >= len(l.comms) {
		return nil, ErrUnknownCommodity
	}
	if a.kind == KindBoundary && !r.release {
		return nil, ErrBoundaryReserve
	}
	return a, nil
}

// earmark earmarks qty units over lots lo..hi, oldest first, or releases them
// newest first.
func (a *account) earmark(lo, hi int, qty int64, release bool) {
	remaining := qty
	if release {
		for i := hi - 1; i >= lo && remaining > 0; i-- {
			n := min(a.lots[i].Reserved, remaining)
			a.lots[i].Reserved -= n
			remaining -= n
		}
		return
	}
	for i := lo; i < hi && remaining > 0; i++ {
		n := min(a.lots[i].Quantity-a.lots[i].Reserved, remaining)
		a.lots[i].Reserved += n
		remaining -= n
	}
}

// reserveLot earmarks or releases qty units on exactly the named lot.
func reserveLot(a *account, key Key, qty int64, release bool) error {
	i, found := a.find(key)
	if !found {
		return ErrUnknownLot
	}
	lt := &a.lots[i]
	if err := lotShort(lt.Quantity, lt.Reserved, qty, release); err != nil {
		return err
	}
	if release {
		qty = -qty
	}
	lt.Reserved += qty
	return nil
}
