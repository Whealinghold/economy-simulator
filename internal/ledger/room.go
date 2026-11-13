package ledger

import (
	"fmt"
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Size is the mass and the bulk of one unit of a commodity, as primitives
// (spec-storage.md 4.1.1): the ledger knows no commodity, only this table,
// given once with SetSizes like the shelf life and the durability.
type Size struct {
	Mass, Bulk int64
}

// Room is the room of an account and what it keeps beside its lots
// (spec-storage.md 4.1.1). Limited is false for an account with no bound, and
// then none of the rest is read. Load is maintained by every operation that
// changes a lot; the pledge and the reserved room are set by the world.
type Room struct {
	Limited      bool
	Mass, Bulk   int64
	LoadMass     int64
	LoadBulk     int64
	PledgeMass   int64
	PledgeBulk   int64
	ReservedMass int64
	ReservedBulk int64
}

// Credit is the form of a credit of goods to an account (spec-storage.md
// 4.1.3), as a debit of money may be marked "from reserved". It is orthogonal
// to the earmark of the source: a settlement is a move of earmarked goods
// credited from the buyer's reserved room.
type Credit uint8

// The forms of a credit.
const (
	// CreditFree is the unmarked form: the load must fit the free room, else
	// ErrNoRoom. It is the zero value, and the only form for an account that is
	// not limited.
	CreditFree Credit = iota
	// CreditReserved draws on the reserved room: the reserved room falls by
	// what the load rises by, so the free room does not change. More than is
	// reserved is ErrReservationExceeds.
	CreditReserved
	// CreditPledge draws on the pledge, bounded by it (ErrReservationExceeds)
	// and by RoomLeft (ErrNoRoom): the pledge falls by what the load rises by.
	CreditPledge
)

// SetSizes gives the table of unit sizes, one per commodity in index order. It
// is called once, before any account is limited; a table of another length, a
// negative size, or a call after an account has a room is ErrInvalidConfig. A
// ledger with no table gives every commodity the size 0, which takes no room.
func (l *Ledger) SetSizes(sizes []Size) error {
	if len(sizes) != len(l.comms) {
		return fmt.Errorf("%w: %d sizes for %d commodities", ErrInvalidConfig, len(sizes), len(l.comms))
	}
	for i, s := range sizes {
		if s.Mass < 0 || s.Bulk < 0 {
			return fmt.Errorf("%w: commodity %d has a negative size", ErrInvalidConfig, i)
		}
	}
	for i := range l.accounts {
		if l.accounts[i].room != nil {
			return fmt.Errorf("%w: sizes are set after an account was limited", ErrInvalidConfig)
		}
	}
	l.sizes = append([]Size(nil), sizes...)
	return nil
}

// sizeOf returns the size of one unit of commodity c.
func (l *Ledger) sizeOf(c int) Size {
	if l.sizes == nil {
		return Size{}
	}
	return l.sizes[c]
}

// roomAccount returns the open account id, which must be limited.
func (l *Ledger) roomAccount(id int) (*account, error) {
	a, err := l.open(id)
	if err != nil {
		return nil, err
	}
	if a.room == nil {
		return nil, ErrNotLimited
	}
	return a, nil
}

// RoomOf returns a copy of the room of account id. An account that is not
// limited, a closed one and a boundary return the zero Room, with Limited
// false; an id outside the table is ErrUnknownAccount.
func (l *Ledger) RoomOf(id int) (Room, error) {
	a, err := l.at(id)
	if err != nil {
		return Room{}, err
	}
	if a.room == nil {
		return Room{}, nil
	}
	return *a.room, nil
}

// SetRoom sets the room of an agent account, or removes it with limited false
// (the pledge and the reserved room go with it). Turning a room on computes
// the load from the lots. The room must hold what is there: if the load plus
// the reserved room exceeds mass or bulk it is ErrNoRoom, and nothing changes.
// Setting the room of an account that is already limited resizes it: the load,
// the reserved room and the pledge stay as they are. The pledge is not tested, because a cut of the room may leave it above what
// the room can hold (spec-storage.md 3.4.4, 4.1.2). A mass or bulk below 0 is
// ErrInvalidAmount, an account that is not an agent's is ErrInvalidKind, and a
// load that does not fit int64 is ErrBalanceOverflow.
func (l *Ledger) SetRoom(id int, limited bool, mass, bulk int64) error {
	a, err := l.open(id)
	if err != nil {
		return err
	}
	if a.kind != KindAgent {
		return ErrInvalidKind
	}
	if mass < 0 || bulk < 0 {
		return ErrInvalidAmount
	}
	if !limited {
		a.room = nil
		return nil
	}
	r := a.room
	if r == nil {
		r = &Room{}
		if r.LoadMass, r.LoadBulk, err = l.loadOfLots(a); err != nil {
			return err
		}
	}
	if r.LoadMass+r.ReservedMass > mass || r.LoadBulk+r.ReservedBulk > bulk {
		return ErrNoRoom
	}
	r.Limited, r.Mass, r.Bulk = true, mass, bulk
	a.room = r
	return nil
}

// loadOfLots sums quantity times size over the lots of a, per dimension.
func (l *Ledger) loadOfLots(a *account) (mass, bulk int64, err error) {
	var m, b checked.Acc
	for _, lt := range a.lots {
		s := l.sizeOf(lt.Commodity)
		if !m.AddMul(lt.Quantity, s.Mass) || !b.AddMul(lt.Quantity, s.Bulk) {
			return 0, 0, ErrBalanceOverflow
		}
	}
	mass, okm := m.DivHalfEven(1)
	bulk, okb := b.DivHalfEven(1)
	if !okm || !okb {
		return 0, 0, ErrBalanceOverflow
	}
	return mass, bulk, nil
}

// SetPledge sets the pledge of a limited account, absolute (spec-storage.md
// 4.1.2). The value is not tested against the room: the callers test before a
// commitment, and the recompute after a cut, the warm start and a restore set
// what the job and the shipments imply. A value below 0 is ErrInvalidAmount and
// an account that is not limited ErrNotLimited.
func (l *Ledger) SetPledge(id int, mass, bulk int64) error {
	a, err := l.roomAccount(id)
	if err != nil {
		return err
	}
	if mass < 0 || bulk < 0 {
		return ErrInvalidAmount
	}
	a.room.PledgeMass, a.room.PledgeBulk = mass, bulk
	return nil
}

// ReserveRoom adds to the reserved room of a limited account: the room of a
// bid admitted this tick. It is ErrNoRoom if the amount exceeds the free room
// (a free room below 0 counts as 0), ErrInvalidAmount below 0, ErrNotLimited
// for an account with no room. A pair of 0 and 0 is a valid no-op, the room of
// a commodity of size 0.
func (l *Ledger) ReserveRoom(id int, mass, bulk int64) error {
	a, err := l.roomAccount(id)
	if err != nil {
		return err
	}
	if mass < 0 || bulk < 0 {
		return ErrInvalidAmount
	}
	fm, fb := a.room.free()
	if mass > fm || bulk > fb {
		return ErrNoRoom
	}
	a.room.ReservedMass += mass
	a.room.ReservedBulk += bulk
	return nil
}

// ReleaseRoom takes from the reserved room of a limited account. More than is
// reserved is ErrReservationExceeds.
func (l *Ledger) ReleaseRoom(id int, mass, bulk int64) error {
	a, err := l.roomAccount(id)
	if err != nil {
		return err
	}
	if mass < 0 || bulk < 0 {
		return ErrInvalidAmount
	}
	if mass > a.room.ReservedMass || bulk > a.room.ReservedBulk {
		return ErrReservationExceeds
	}
	a.room.ReservedMass -= mass
	a.room.ReservedBulk -= bulk
	return nil
}

// FreeRoom returns the free room of a limited account, the room less the load,
// the pledge and the reserved room, 0 where it would be negative. It reads and
// changes nothing and allocates nothing.
func (l *Ledger) FreeRoom(id int) (mass, bulk int64, err error) {
	a, err := l.roomAccount(id)
	if err != nil {
		return 0, 0, err
	}
	mass, bulk = a.room.free()
	return mass, bulk, nil
}

// RoomLeft returns the room less the load and the reserved room, 0 where it
// would be negative: the room a credit from the pledge may use (spec-storage.md
// 4.1.3). It reads and changes nothing and allocates nothing.
func (l *Ledger) RoomLeft(id int) (mass, bulk int64, err error) {
	a, err := l.roomAccount(id)
	if err != nil {
		return 0, 0, err
	}
	mass, bulk = a.room.left()
	return mass, bulk, nil
}

// free is Room - Load - Pledge - Reserved per dimension, floored at 0. The
// values are bounded far below int64 by the invariant, and a result that does
// not fit is read as 0, the safe side.
func (r *Room) free() (mass, bulk int64) {
	return headroom(r.Mass, r.LoadMass, r.PledgeMass, r.ReservedMass), headroom(r.Bulk, r.LoadBulk, r.PledgeBulk, r.ReservedBulk)
}

// left is Room - Load - Reserved per dimension, floored at 0.
func (r *Room) left() (mass, bulk int64) {
	return headroom(r.Mass, r.LoadMass, 0, r.ReservedMass), headroom(r.Bulk, r.LoadBulk, 0, r.ReservedBulk)
}

// headroom returns room less the three uses, or 0 when it is negative.
func headroom(room, a, b, c int64) int64 {
	v, ok := checked.Sub(room, a)
	if ok {
		v, ok = checked.Sub(v, b)
	}
	if ok {
		v, ok = checked.Sub(v, c)
	}
	if !ok {
		return 0
	}
	return max(v, 0)
}

// unitLoad returns the load of qty units of commodity c. ok is false when it
// does not fit int64.
func (l *Ledger) unitLoad(c int, qty int64) (mass, bulk int64, ok bool) {
	s := l.sizeOf(c)
	mass, okm := checked.Mul(qty, s.Mass)
	bulk, okb := checked.Mul(qty, s.Bulk)
	return mass, bulk, okm && okb
}

// loadAdd raises the load of a limited account by qty units of c. A credit is
// checked before it is applied, so an overflow here is a programmer error.
func (l *Ledger) loadAdd(a *account, c int, qty int64) {
	m, b, ok := l.unitLoad(c, qty)
	lm, okm := checked.Add(a.room.LoadMass, m)
	lb, okb := checked.Add(a.room.LoadBulk, b)
	if !ok || !okm || !okb {
		invariantViolated("a credit overflowed the load of a limited account")
	}
	a.room.LoadMass, a.room.LoadBulk = lm, lb
}

// loadSub lowers the load of a limited account by qty units of c.
func (l *Ledger) loadSub(a *account, c int, qty int64) {
	m, b, ok := l.unitLoad(c, qty)
	if !ok || a.room.LoadMass < m || a.room.LoadBulk < b {
		invariantViolated("a debit took more load than a limited account carries")
	}
	a.room.LoadMass -= m
	a.room.LoadBulk -= b
}

// checkRoomCredit tests a credit of d to a against its form. freed is the load
// that the same operation frees first (the inputs of a Transform). An account
// that is not limited takes any credit in the free form and refuses the others
// with ErrNotLimited. The order is ErrReservationExceeds, then ErrNoRoom.
func checkRoomCredit(a *account, d, freed Size, form Credit) error {
	r := a.room
	if r == nil {
		if form != CreditFree {
			return ErrNotLimited
		}
		return nil
	}
	switch form {
	case CreditReserved:
		return above(d, Size{r.ReservedMass, r.ReservedBulk}, ErrReservationExceeds)
	case CreditPledge:
		if err := above(d, Size{r.PledgeMass, r.PledgeBulk}, ErrReservationExceeds); err != nil {
			return err
		}
		lm, lb := r.left()
		return above(d, Size{plus(lm, freed.Mass), plus(lb, freed.Bulk)}, ErrNoRoom)
	}
	fm, fb := r.free()
	return above(d, Size{plus(fm, freed.Mass), plus(fb, freed.Bulk)}, ErrNoRoom)
}

// plus is a + b for two non-negative values, saturating at the largest int64
// so that a limit never wraps below the credit it is tested against.
func plus(a, b int64) int64 {
	if v, ok := checked.Add(a, b); ok {
		return v
	}
	return math.MaxInt64
}

// above returns err if d exceeds limit in either dimension, else nil.
func above(d, limit Size, err error) error {
	if d.Mass > limit.Mass || d.Bulk > limit.Bulk {
		return err
	}
	return nil
}

// settleCredit draws a credit of dm mass and db bulk on the reserved room or
// the pledge, after the checks passed.
func settleCredit(a *account, dm, db int64, form Credit) {
	switch {
	case a.room == nil:
	case form == CreditReserved:
		a.room.ReservedMass -= dm
		a.room.ReservedBulk -= db
	case form == CreditPledge:
		a.room.PledgeMass -= dm
		a.room.PledgeBulk -= db
	}
}

// roomReserved reports whether the account holds reserved room.
func (a *account) roomReserved() bool {
	return a.room != nil && (a.room.ReservedMass != 0 || a.room.ReservedBulk != 0)
}
