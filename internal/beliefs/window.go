package beliefs

import "github.com/Whealinghold/economy-sim/internal/checked"

// Window is the price window of one market (spec-beliefs.md 7): the market
// prices of its last capacity ticks in which units traded. It is a ring whose
// slice is allocated once, so a push allocates nothing, and it caches its mean,
// low and high at each push and restore. The write position is not part of its
// meaning: two windows with the same prices, oldest first, behave identically.
type Window struct {
	buf       []int64
	start, n  int
	maxPrice  int64
	mean      int64
	low, high int64
}

// NewWindow returns an empty window of the given capacity (at least 1) that
// stores prices in [1, maxPrice].
func NewWindow(capacity int, maxPrice int64) Window {
	return Window{buf: make([]int64, max(capacity, 1)), maxPrice: max(maxPrice, 1)}
}

// Push adds a market price, overwriting the oldest when the window is full. It
// has no error result: a price below 1 is stored as 1 and one above the maximum
// price as the maximum, and takes its slot like any price (spec-beliefs.md 7.5).
func (w *Window) Push(price int64) {
	price = min(max(price, 1), w.maxPrice)
	if w.n < len(w.buf) {
		w.buf[(w.start+w.n)%len(w.buf)] = price
		w.n++
	} else {
		w.buf[w.start] = price
		w.start = (w.start + 1) % len(w.buf)
	}
	w.refresh()
}

// refresh recomputes the cached statistics. The sum is 128 bits, so it cannot
// overflow, and the mean is rounded half to even (spec-beliefs.md 7.2).
//
// shortcut: O(capacity) per push, not incremental. checked.Acc is add-only (no
// subtraction of the evicted price), and a running int64 sum could overflow at a
// large maxPrice; upgrade if a profile shows PriceWindow large enough to matter.
func (w *Window) refresh() {
	if w.n == 0 {
		// An empty window reads as a new one: the statistics are 0 and unread.
		w.mean, w.low, w.high = 0, 0, 0
		return
	}
	var sum checked.Acc
	w.low, w.high = w.maxPrice, 1
	for i := range w.n {
		v := w.buf[(w.start+i)%len(w.buf)]
		sum.AddMul(v, 1) // v is at least 1: cannot fail
		w.low, w.high = min(w.low, v), max(w.high, v)
	}
	w.mean, _ = sum.DivHalfEven(int64(w.n)) // a mean of values <= maxPrice fits
}

// Len returns the number of prices held.
func (w *Window) Len() int { return w.n }

// Mean returns the mean of the prices, rounded half to even, and false while the
// window is empty.
func (w *Window) Mean() (int64, bool) { return w.mean, w.n > 0 }

// Low returns the lowest price and false while the window is empty.
func (w *Window) Low() (int64, bool) { return w.low, w.n > 0 }

// High returns the highest price and false while the window is empty.
func (w *Window) High() (int64, bool) { return w.high, w.n > 0 }

// Values appends the prices, oldest first, to dst and returns it.
func (w *Window) Values(dst []int64) []int64 {
	for i := range w.n {
		dst = append(dst, w.buf[(w.start+i)%len(w.buf)])
	}
	return dst
}

// Restore replaces the content with vals, oldest first. It refuses, leaving the
// window unchanged, more values than the capacity or a price outside
// [1, maxPrice], and reports whether it restored.
func (w *Window) Restore(vals []int64) bool {
	if len(vals) > len(w.buf) {
		return false
	}
	for _, v := range vals {
		if v < 1 || v > w.maxPrice {
			return false
		}
	}
	w.start, w.n = 0, copy(w.buf, vals)
	w.refresh()
	return true
}

// Position returns where price lies between low and high, in basis points:
// (price - low) * 10 000 / (high - low), truncated, clamped to [0, 10 000], and
// 5 000 when high equals low (spec-beliefs.md 7.3).
func Position(price, low, high int64) int64 {
	switch {
	case high <= low:
		return bpDen / 2
	case price <= low:
		return 0
	case price >= high:
		return bpDen
	}
	// low < price < high: both differences are positive and the ratio is below 10 000.
	pos, ok := checked.MulDiv(price-low, bpDen, high-low)
	if !ok {
		return bpDen
	}
	return pos
}
