package indicator

import (
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
)

// maxInt64 is the largest int64; indicator imports no math (FR-IND-008).
const maxInt64 = 1<<63 - 1

// Tally accumulates the running sums of one window from the events of its steps
// and nothing else (spec-indicators.md 4.2). It implements economysim.Observer:
// it ignores every event type it does not use, every event while no window is
// open, and the per-order and per-trade classes (OrderPlaced, OrderRejected,
// TradeExecuted). It holds no world and so cannot call one.
//
// Its size is a function of the configuration: the dense tables are sized at
// Open from the names table of the snapshot, and the sparse ones (flows,
// outputs, relocations) by the distinct keys, never by agents, orders, trades or
// ticks (FR-IND-004). A Tally is the consumer's value, not world state, and is
// not saved in a checkpoint. It is not safe for concurrent use.
type Tally struct {
	open bool
	w    Window
	// steps counts TickEnded; last is the tick of the previous one.
	steps, last int64
	// maxStep is the highest tick of a step event and maxCall the highest of a
	// call event; Close checks them against its snapshot.
	maxStep, maxCall int64
	// stray, failed and over are the faults Close reports: an event outside the
	// window or with an ID outside the names table; a failed step; a sum past
	// int64.
	stray, failed, over bool
}

var _ economysim.Observer = (*Tally)(nil)

// NewTally returns an empty tally with no window open.
func NewTally() *Tally { return &Tally{} }

// Open starts a window at s.Tick: it takes the households by (region, role)
// and the pending totals from s and clears every sum, so nothing of an earlier
// window survives (spec-indicators.md 4.2.5). Take s after the calls you want
// inside the opening state (4.1.4). It returns ErrRange, and leaves the tally as
// it was, for a role row outside the names table.
func (t *Tally) Open(s economysim.Snapshot) error {
	nr, nc, nro, nrec := len(s.Names.Regions), len(s.Names.Commodities), len(s.Names.Roles), len(s.Names.Recipes)
	w := Window{
		From: s.Tick, To: s.Tick, ConfigHash: s.ConfigHash,
		Regions: nr, Commodities: nc, Roles: nro, Recipes: nrec,
		OpenHouseholds: make([]int64, nr*nro), CloseHouseholds: nil,
		OpenPending: slices.Clone(s.Pending),
		Markets:     make([]MarketTally, nr*nc), Beliefs: make([]BeliefTally, nr*nc),
		Recipe: make([]RecipeTally, nr*nrec), People: make([]PeopleTally, nr*nro),
		Regional: make([]RegionTally, nr),
	}
	for _, r := range s.Roles {
		i, ok := w.cell(r.Region, r.Role)
		if !ok {
			return wrap("Tally.Open", "roles", ErrRange)
		}
		w.OpenHouseholds[i], w.People[i].Households = r.Households, r.Households
	}
	*t = Tally{open: true, w: w, maxStep: s.Tick - 1, maxCall: s.Tick - 1}
	return nil
}

// Close ends the window at s and returns it, or an error and leaves the window
// open. s.Tick must be above the opening tick and the tally must have seen
// exactly the TickEnded of the steps in between, none failed, and no event
// outside them (ErrWindowOrder); the snapshots must share a configuration hash
// (ErrConfigMismatch); a sum must fit int64 (ErrOverflow). The returned Window
// shares nothing with the tally, which may be opened again at once
// (spec-indicators.md 4.2.5, FR-IND-006).
func (t *Tally) Close(s economysim.Snapshot) (Window, error) {
	switch {
	case !t.open:
		return Window{}, wrap("Tally.Close", "window", ErrWindowOrder)
	case s.ConfigHash != t.w.ConfigHash:
		return Window{}, wrap("Tally.Close", "config_hash", ErrConfigMismatch)
	case s.Tick <= t.w.From, t.stray, t.failed, t.steps != s.Tick-t.w.From, t.maxStep >= s.Tick, t.maxCall > s.Tick:
		return Window{}, wrap("Tally.Close", "ticks", ErrWindowOrder)
	case t.over:
		return Window{}, wrap("Tally.Close", "sums", ErrOverflow)
	}
	w := t.w
	w.To, w.Steps = s.Tick, t.steps
	w.CloseHouseholds = make([]int64, len(w.OpenHouseholds))
	for _, r := range s.Roles {
		i, ok := w.cell(r.Region, r.Role)
		if !ok {
			return Window{}, wrap("Tally.Close", "roles", ErrRange)
		}
		w.CloseHouseholds[i] = r.Households
	}
	w.ClosePending = slices.Clone(s.Pending)
	*t = Tally{}
	return w, nil
}

// cell returns the index of the (region, role) cell of the people tables.
func (w *Window) cell(r economysim.RegionID, role economysim.RoleID) (int, bool) {
	if r < 0 || int(r) >= w.Regions || role < 0 || int(role) >= w.Roles {
		return 0, false
	}
	return int(r)*w.Roles + int(role), true
}

// market returns the index of the (region, commodity) cell of the market tables.
func (w *Window) market(r economysim.RegionID, c economysim.CommodityID) (int, bool) {
	if r < 0 || int(r) >= w.Regions || c < 0 || int(c) >= w.Commodities {
		return 0, false
	}
	return int(r)*w.Commodities + int(c), true
}

// add adds v to *dst and records an overflow, so that Close reports it.
func (t *Tally) add(dst *int64, v int64) {
	if (v > 0 && *dst > maxInt64-v) || (v < 0 && *dst < -maxInt64-1-v) {
		t.over = true
		return
	}
	*dst += v
}

// step reports whether a step event at tick belongs after the window's start,
// and notes the highest tick for Close.
func (t *Tally) step(tick int64) bool {
	if tick < t.w.From {
		t.stray = true
		return false
	}
	t.maxStep = max(t.maxStep, tick)
	return true
}

// call is step for the events of a call between ticks, which carry the tick of
// the next step and so may be at the window's end (spec-indicators.md 4.1.4).
func (t *Tally) call(tick int64) bool {
	if tick < t.w.From {
		t.stray = true
		return false
	}
	t.maxCall = max(t.maxCall, tick)
	return true
}
