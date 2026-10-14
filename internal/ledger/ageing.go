package ledger

import (
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// expiry is the last tick at which a lot born at born with shelf life shelf is
// still held: it spoils at the end of it (spec-ledger.md 5.1.2). A sum that does
// not fit int64 means the lot never expires.
func expiry(born, shelf int64) int64 {
	e, ok := checked.Add(born, shelf-1)
	if !ok {
		return math.MaxInt64
	}
	return e
}

// noteExpiry lowers the account's hint to e if it is earlier.
func (a *account) noteExpiry(e int64) {
	if !a.hasHint || e < a.hint {
		a.hint, a.hasHint = e, true
	}
}

// rehint recomputes the hint of one account from its lots, exactly.
func (l *Ledger) rehint(a *account) {
	a.hint, a.hasHint = 0, false
	for _, lt := range a.lots {
		if s := l.comms[lt.Commodity].ShelfLife; s > 0 {
			a.noteExpiry(expiry(lt.Born, s))
		}
	}
}

// RebuildHints recomputes the earliest-expiry hint of every account from its
// lots. A restore calls it, because the hints are derived and not saved
// (spec-ledger.md 5.3.2, FR-LED-030).
func (l *Ledger) RebuildHints() {
	for i := range l.accounts {
		l.rehint(&l.accounts[i])
	}
}

// Age runs the ageing of the current tick (spec-ledger.md 5.3.1): every lot
// whose Born + ShelfLife - 1 is at most the current tick moves whole to the
// Spoilage boundary of its commodity, in the region of its account. The world
// calls it at the end of the tick, before AdvanceTick (P8.1).
//
// Only accounts whose hint has been reached are visited (5.3.2), so the work is
// proportional to them and to the lots that expire, plus one hint comparison
// per account. A visited account's hint is recomputed exactly.
//
// It is atomic (2.6). It returns ErrUnknownAccount if a spoiling commodity with
// an expiring lot has no Spoilage boundary account, and ErrBalanceOverflow if
// a boundary balance or a flow total would leave int64; both are found before
// anything is moved.
func (l *Ledger) Age() error {
	due, err := l.sumExpired()
	if err != nil {
		clear(l.spoil)
		return err
	}
	flows := l.flows()
	if due {
		err := l.checkSpoilage(flows)
		clear(l.spoil)
		if err != nil {
			return err
		}
	}
	l.moveExpired(flows)
	return nil
}

// expired reports whether lot lt has spoiled by the current tick.
func (l *Ledger) expired(lt Lot) bool {
	s := l.comms[lt.Commodity].ShelfLife
	return s != 0 && expiry(lt.Born, s) <= l.tick
}

// sumExpired is pass 1 of Age: it reads only, summing the expired units per
// (region, commodity) into l.spoil, and reports whether any lot is due.
func (l *Ledger) sumExpired() (bool, error) {
	nc := len(l.comms)
	due := false
	for i := range l.accounts {
		a := &l.accounts[i]
		if !a.hasHint || a.hint > l.tick {
			continue
		}
		for _, lt := range a.lots {
			if !l.expired(lt) {
				continue
			}
			slot := a.region*nc + lt.Commodity
			v, ok := checked.Add(l.spoil[slot], lt.Quantity)
			if !ok {
				return false, ErrBalanceOverflow
			}
			l.spoil[slot] = v
			due = true
		}
	}
	return due, nil
}

// moveExpired is pass 2 of Age: it applies the moves. Accounts are in ID order
// and lots in canonical order.
func (l *Ledger) moveExpired(flows []int64) {
	for i := range l.accounts {
		a := &l.accounts[i]
		if !a.hasHint || a.hint > l.tick {
			continue
		}
		kept := a.lots[:0]
		for _, lt := range a.lots {
			if !l.expired(lt) {
				kept = append(kept, lt)
				continue
			}
			flows[a.region*l.stride+l.off[Spoilage][thingGoods]+lt.Commodity] += lt.Quantity
			l.accounts[l.boundary[l.off[Spoilage][thingGoods]+lt.Commodity]].balance += lt.Quantity
		}
		a.lots = kept
		l.rehint(a)
	}
}

// checkSpoilage checks, from the per-(region, commodity) sums in l.spoil, that
// every Spoilage boundary exists and that no balance or flow total overflows.
func (l *Ledger) checkSpoilage(flows []int64) error {
	for c := range l.comms {
		if err := l.checkSpoilageOf(c, flows); err != nil {
			return err
		}
	}
	return nil
}

// checkSpoilageOf is checkSpoilage for commodity c.
func (l *Ledger) checkSpoilageOf(c int, flows []int64) error {
	nc := len(l.comms)
	var total int64
	for r := 0; r < l.regions; r++ {
		v := l.spoil[r*nc+c]
		if v == 0 {
			continue
		}
		t, ok := checked.Add(total, v)
		if !ok {
			return ErrBalanceOverflow
		}
		total = t
		if !addFits(flows[r*l.stride+l.off[Spoilage][thingGoods]+c], v) {
			return ErrBalanceOverflow
		}
	}
	if total == 0 {
		return nil
	}
	id := l.boundary[l.off[Spoilage][thingGoods]+c]
	if id < 0 {
		return ErrUnknownAccount
	}
	if !addFits(l.accounts[id].balance, total) {
		return ErrBalanceOverflow
	}
	return nil
}

// EndTick closes the current tick (P8.6): the tick counter advances, then the
// flow totals and the wear units are cleared. Moves a caller makes after it are
// in the next tick's totals (WLD 8.1.3).
func (l *Ledger) EndTick() {
	l.AdvanceTick()
	l.ClearTotals()
}

// Expiring returns the units of a commodity in an account that are not
// earmarked and spoil by the end of tick (spec-ledger.md 5.3.3): the lots with
// Born + ShelfLife - 1 at most tick. For the current tick that is what Age will
// move. It is 0 for a commodity that does not spoil and for a tick before the
// current one, and it reads no lot when the account's hint is after tick. It is
// read-only and allocates nothing. It returns ErrUnknownAccount,
// ErrClosedAccount or ErrUnknownCommodity.
func (l *Ledger) Expiring(id, commodity int, tick int64) (int64, error) {
	a, err := l.open(id)
	if err != nil {
		return 0, err
	}
	if commodity < 0 || commodity >= len(l.comms) {
		return 0, ErrUnknownCommodity
	}
	s := l.comms[commodity].ShelfLife
	if s == 0 || tick < l.tick || !a.mayExpireBy(tick) {
		return 0, nil
	}
	return a.expiringUnits(commodity, s, tick), nil
}

// expiringUnits sums the unearmarked units of the commodity's lots that expire
// by tick, given its shelf life.
func (a *account) expiringUnits(commodity int, shelfLife, tick int64) int64 {
	var n int64
	lo, hi := a.span(commodity)
	for _, lt := range a.lots[lo:hi] {
		if expiry(lt.Born, shelfLife) > tick {
			break
		}
		n += lt.Quantity - lt.Reserved
	}
	return n
}

// mayExpireBy reports whether any lot can expire by tick: the account has a
// hint and it is not after tick.
func (a *account) mayExpireBy(tick int64) bool {
	return a.hasHint && a.hint <= tick
}

// compact removes the lots of quantity 0, in place.
func (a *account) compact() {
	a.lots = slices.DeleteFunc(a.lots, func(lt Lot) bool { return lt.Quantity == 0 })
}
