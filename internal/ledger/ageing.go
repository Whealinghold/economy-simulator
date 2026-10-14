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
	nc := len(l.comms)
	due := false
	// Pass 1 reads only: the expired units per (region, commodity).
	for i := range l.accounts {
		a := &l.accounts[i]
		if !a.hasHint || a.hint > l.tick {
			continue
		}
		for _, lt := range a.lots {
			s := l.comms[lt.Commodity].ShelfLife
			if s == 0 || expiry(lt.Born, s) > l.tick {
				continue
			}
			slot := a.region*nc + lt.Commodity
			v, ok := checked.Add(l.spoil[slot], lt.Quantity)
			if !ok {
				clear(l.spoil)
				return ErrBalanceOverflow
			}
			l.spoil[slot] = v
			due = true
		}
	}
	flows := l.flows()
	if due {
		if err := l.checkSpoilage(flows); err != nil {
			clear(l.spoil)
			return err
		}
		clear(l.spoil)
	}
	// Pass 2 applies. Accounts are in ID order and lots in canonical order.
	for i := range l.accounts {
		a := &l.accounts[i]
		if !a.hasHint || a.hint > l.tick {
			continue
		}
		kept := a.lots[:0]
		for _, lt := range a.lots {
			s := l.comms[lt.Commodity].ShelfLife
			if s == 0 || expiry(lt.Born, s) > l.tick {
				kept = append(kept, lt)
				continue
			}
			flows[a.region*l.stride+l.off[Spoilage][thingGoods]+lt.Commodity] += lt.Quantity
			l.accounts[l.boundary[l.off[Spoilage][thingGoods]+lt.Commodity]].balance += lt.Quantity
		}
		a.lots = kept
		l.rehint(a)
	}
	return nil
}

// checkSpoilage checks, from the per-(region, commodity) sums in l.spoil, that
// every Spoilage boundary exists and that no balance or flow total overflows.
func (l *Ledger) checkSpoilage(flows []int64) error {
	nc := len(l.comms)
	for c := 0; c < nc; c++ {
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
			if _, ok := checked.Add(flows[r*l.stride+l.off[Spoilage][thingGoods]+c], v); !ok {
				return ErrBalanceOverflow
			}
		}
		if total == 0 {
			continue
		}
		id := l.boundary[l.off[Spoilage][thingGoods]+c]
		if id < 0 {
			return ErrUnknownAccount
		}
		if _, ok := checked.Add(l.accounts[id].balance, total); !ok {
			return ErrBalanceOverflow
		}
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
	if s == 0 || tick < l.tick || !a.hasHint || a.hint > tick {
		return 0, nil
	}
	var n int64
	lo, hi := a.span(commodity)
	for _, lt := range a.lots[lo:hi] {
		if expiry(lt.Born, s) > tick {
			break
		}
		n += lt.Quantity - lt.Reserved
	}
	return n, nil
}

// compact removes the lots of quantity 0, in place.
func (a *account) compact() {
	a.lots = slices.DeleteFunc(a.lots, func(lt Lot) bool { return lt.Quantity == 0 })
}
