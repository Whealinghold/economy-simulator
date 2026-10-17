package welfare

import (
	"errors"
	"fmt"
	"math"
)

// ErrRecords is the one refusal of the package: a window or a bound that is
// out of range, more records than the window holds, or a record with a field
// beyond the clamp. The checkpoint row maps it to its own corrupt-checkpoint
// error (architecture.md 7.7).
var ErrRecords = errors.New("welfare: invalid ring or records")

// Record is the welfare record of one tick of one agent (spec-welfare-and-
// wages.md 13, FR-WEL-040): signed minor units, and household-ticks. It holds
// no region and no role, so a move cannot reset it (FR-WEL-006).
type Record struct {
	Income     int64 // income_t, relief and the unearned sum included
	Cost       int64 // cost_t
	Relief     int64 // F_t
	Unearned   int64 // G_t
	OwnReceipt int64 // K^own_t
	Working    bool  // a working tick (4.3)
	WorkHH     int64 // working household-ticks
	OwnHH      int64 // own-use household-ticks, part of WorkHH
}

// fields lists the numeric fields of a record, in a fixed order.
func (r *Record) fields() [7]*int64 {
	return [7]*int64{&r.Income, &r.Cost, &r.Relief, &r.Unearned, &r.OwnReceipt, &r.WorkHH, &r.OwnHH}
}

// clampTo limits every field to plus or minus c and reports whether it changed one.
func (r *Record) clampTo(c int64) (changed bool) {
	for _, p := range r.fields() {
		if *p > c {
			*p, changed = c, true
		} else if *p < -c {
			*p, changed = -c, true
		}
	}
	return changed
}

// outside reports a field beyond plus or minus c.
func (r *Record) outside(c int64) bool {
	for _, p := range r.fields() {
		if *p > c || *p < -c {
			return true
		}
	}
	return false
}

// atBound reports a field exactly at plus or minus c.
func (r *Record) atBound(c int64) bool {
	for _, p := range r.fields() {
		if *p == c || *p == -c {
			return true
		}
	}
	return false
}

// sums are the running sums of a ring. Each record field is within plus or
// minus the clamp and a ring is at most maxWindow long, so no sum overflows.
type sums struct {
	income, cost, relief, unearned, ownReceipt, workHH, ownHH int64
	working                                                   int64 // working ticks
	atBound                                                   int64 // records holding a field at the clamp
}

// Ring holds the last window records of one agent, oldest first, with running
// sums, so a read costs the same whatever the window (NFR-WEL-001). The zero
// Ring holds nothing; make one with NewRing or FromRecords.
type Ring struct {
	recs      []Record // len is the window
	start, n  int
	maxWindow int64
	clamp     int64 // MaxInt64 / maxWindow
	s         sums
}

func validBounds(window, maxWindow int64) error {
	if maxWindow < 1 || window < 1 || window > maxWindow {
		return fmt.Errorf("%w: window %d, maximum %d", ErrRecords, window, maxWindow)
	}
	return nil
}

// NewRing returns an empty ring of the given window. maxWindow is the root's
// MaxWelfareWindow, which the package cannot import; the window must be in
// [1, maxWindow]. The full capacity is allocated here and never at a push.
func NewRing(window, maxWindow int64) (Ring, error) {
	if err := validBounds(window, maxWindow); err != nil {
		return Ring{}, err
	}
	return Ring{recs: make([]Record, window), maxWindow: maxWindow, clamp: math.MaxInt64 / maxWindow}, nil
}

// FromRecords rebuilds a ring and its sums from records ordered oldest first
// (the restore path). It refuses, with ErrRecords and no ring, a bad window or
// bound, more records than the window, and a record with a field beyond plus
// or minus MaxInt64 / maxWindow: only Push stores records, and it clamps.
func FromRecords(records []Record, window, maxWindow int64) (Ring, error) {
	r, err := NewRing(window, maxWindow)
	if err != nil {
		return Ring{}, err
	}
	if int64(len(records)) > window {
		return Ring{}, fmt.Errorf("%w: %d records for a window of %d", ErrRecords, len(records), window)
	}
	for i := range records {
		if records[i].outside(r.clamp) {
			return Ring{}, fmt.Errorf("%w: record %d has a field beyond the clamp", ErrRecords, i)
		}
	}
	for _, rec := range records {
		r.store(rec)
	}
	return r, nil
}

func (r *Ring) apply(rec *Record, sign int64) {
	r.s.income += sign * rec.Income
	r.s.cost += sign * rec.Cost
	r.s.relief += sign * rec.Relief
	r.s.unearned += sign * rec.Unearned
	r.s.ownReceipt += sign * rec.OwnReceipt
	r.s.workHH += sign * rec.WorkHH
	r.s.ownHH += sign * rec.OwnHH
	if rec.Working {
		r.s.working += sign
	}
	if rec.atBound(r.clamp) {
		r.s.atBound += sign
	}
}

// store appends an already clamped record, dropping the oldest when full.
func (r *Ring) store(rec Record) {
	w := len(r.recs)
	if r.n == w {
		r.apply(&r.recs[r.start], -1)
		r.start = (r.start + 1) % w
		r.n--
	}
	r.recs[(r.start+r.n)%w] = rec
	r.n++
	r.apply(&rec, 1)
}

// Push stores a record, dropping the oldest when the ring is full. Each field
// is clamped to plus or minus MaxInt64 / maxWindow and Push reports whether it
// clamped (2.3.2). It never panics and allocates nothing.
func (r *Ring) Push(rec Record) (clamped bool) {
	if len(r.recs) == 0 {
		return false
	}
	clamped = rec.clampTo(r.clamp)
	r.store(rec)
	return clamped
}

// Resize changes the window to t and keeps the newest min(held, t) records
// (D64). It allocates only when growing past the capacity it ever had. A t
// below 1 or above maxWindow is refused with ErrRecords and the ring is left
// unchanged, because the no-overflow proof would not cover it.
func (r *Ring) Resize(t int64) error {
	if err := validBounds(t, r.maxWindow); err != nil {
		return err
	}
	rotateLeft(r.recs, r.start)
	r.start = 0
	nt := int(t)
	if drop := r.n - nt; drop > 0 {
		for i := 0; i < drop; i++ {
			r.apply(&r.recs[i], -1)
		}
		copy(r.recs, r.recs[drop:r.n])
		r.n = nt
	}
	if nt <= cap(r.recs) {
		r.recs = r.recs[:nt]
		return nil
	}
	grown := make([]Record, nt)
	copy(grown, r.recs[:r.n])
	r.recs = grown
	return nil
}

// rotateLeft moves the element at index k to index 0, in place.
func rotateLeft(s []Record, k int) {
	if k == 0 {
		return
	}
	reverse(s[:k])
	reverse(s[k:])
	reverse(s)
}

func reverse(s []Record) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// Len is the number of records held, at most the window.
func (r *Ring) Len() int { return r.n }

// Window is the window the ring was sized to.
func (r *Ring) Window() int64 { return int64(len(r.recs)) }

// AppendRecords appends the records held to dst, oldest first, for the
// checkpoint row.
func (r *Ring) AppendRecords(dst []Record) []Record {
	w := len(r.recs)
	for i := 0; i < r.n; i++ {
		dst = append(dst, r.recs[(r.start+i)%w])
	}
	return dst
}

// AtBound reports whether a record of the ring holds a field at the clamp: a
// constant-cost read of the count kept at push, eviction and resize (D63).
func (r *Ring) AtBound() bool { return r.s.atBound > 0 }

// WorkingTicks is the number of working ticks held, for the floor (5.3).
func (r *Ring) WorkingTicks() int64 { return r.s.working }

// Ratio is the welfare ratio of the ring (2.1, 2.2): the sums with the ticks
// the ring lacks padded at defaultBP and the current upkeep. It reads only the
// running sums and the window.
func (r *Ring) Ratio(upkeep, defaultBP int64) (ratioBP int64, present bool) {
	return Ratio(r.s.income, r.s.cost, r.Window()-int64(r.n), upkeep, defaultBP)
}

// EarnedRatio is the earned ratio of the ring (2.1.7).
func (r *Ring) EarnedRatio(upkeep, defaultBP int64) (ratioBP int64, present bool) {
	return EarnedRatio(r.s.income, r.s.relief, r.s.cost, r.Window()-int64(r.n), upkeep, defaultBP)
}

// WageParts returns the numerator and denominator of the observed wage of the
// agent (4.3): income - relief - unearned - own-use receipt, taken in that
// order with saturating differences, over the working household-ticks less the
// own-use household-ticks.
func (r *Ring) WageParts() (num, den int64) {
	num = satSub(satSub(satSub(r.s.income, r.s.relief), r.s.unearned), r.s.ownReceipt)
	return num, satSub(r.s.workHH, r.s.ownHH)
}

// Wage is the observed wage of the agent, truncated toward zero, present
// whenever the denominator is above 0 and of either sign (FR-WEL-018).
func (r *Ring) Wage() (wage int64, present bool) {
	return quotient(r.WageParts())
}

func quotient(num, den int64) (int64, bool) {
	if den <= 0 {
		return 0, false
	}
	return num / den, true
}
