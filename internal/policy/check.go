package policy

import "slices"

// Outcome is the one result each entry of a returned list gets (spec-policy.md
// 6.2.1), the first check that holds, in this order.
type Outcome int

const (
	// NotHeld: the lever index is outside the actor's levers.
	NotHeld Outcome = iota
	// Duplicate: the lever already had an entry in this list.
	Duplicate
	// NoClose: the entry would close a key the actor may not close.
	NoClose
	// Unchanged: the result equals what every key of the lever already has.
	Unchanged
	// Applied: some key of the lever would change.
	Applied
)

// Lever is the state of one lever the checks read: its limits and the value it
// has when the checks begin. Current is the number of the first key of a group;
// Mixed says some key of the group has another number; AnyOpen and AnyClosed say
// some key is open and some key is closed (a lever that is not an edge has no
// closed key, so AnyOpen is true and AnyClosed false).
type Lever struct {
	Min, Max  int64
	MaxStep   int64
	MayClose  bool
	Current   int64
	Mixed     bool
	AnyOpen   bool
	AnyClosed bool
}

// Entry is one setting a policy returned: the lever index, the number and the
// closed flag.
type Entry struct {
	Lever  int
	BP     int64
	Closed bool
}

// Result is what the checks decided for one entry. Value and Closed are what
// every key of the lever takes when Outcome is Applied (and what it already has
// when Unchanged); Clamped and Stepped say the first and the second cut moved the
// number.
type Result struct {
	Outcome Outcome
	Value   int64
	Closed  bool
	Clamped bool
	Stepped bool
}

// Tally counts the outcomes of one list. Entries is the sum of Applied,
// Unchanged, NotHeld, Duplicate and NoClose; Clamped and Stepped count the
// entries that reached the cuts and were moved by one.
type Tally struct {
	Entries   int64
	Applied   int64
	Unchanged int64
	NotHeld   int64
	Duplicate int64
	NoClose   int64
	Clamped   int64
	Stepped   int64
}

// Scratch holds the slices Check reuses, so a call allocates nothing once they
// have grown to the sizes of the actor. The zero value is ready.
type Scratch struct {
	// Results has one row per entry of the last call, in the order of the entries.
	Results []Result
	seen    []bool
}

// Check gives each entry one outcome in the order of spec-policy.md 6.2.1 and
// computes the value the lever would take, with the two cuts of 6.2.2: the
// lever's bounds first, then Current plus or minus MaxStep when MaxStep is above
// 0. When Current is outside the bounds the bounds cut wins and the step is taken
// from Current toward the bound, so the lever comes back inside by at most MaxStep
// a decision. Nothing is written: the caller applies the Applied rows. The rows
// are in s.Results, valid until the next call.
func Check(levers []Lever, entries []Entry, s *Scratch) Tally {
	s.Results = slices.Grow(s.Results[:0], len(entries))[:len(entries)]
	clear(s.Results)
	s.seen = slices.Grow(s.seen[:0], len(levers))[:len(levers)]
	clear(s.seen)
	var t Tally
	t.Entries = int64(len(entries))
	for i, e := range entries {
		r := &s.Results[i]
		if e.Lever < 0 || e.Lever >= len(levers) {
			r.Outcome = NotHeld
			t.NotHeld++
			continue
		}
		if s.seen[e.Lever] {
			r.Outcome = Duplicate
			t.Duplicate++
			continue
		}
		s.seen[e.Lever] = true
		checkOne(&levers[e.Lever], e, r)
		t.count(r)
	}
	return t
}

// checkOne is checks 3 and 4 of 6.2.1 for an entry whose lever exists and has no
// earlier entry.
func checkOne(l *Lever, e Entry, r *Result) {
	if e.Closed && !l.MayClose && l.AnyOpen {
		r.Outcome = NoClose
		return
	}
	r.Value, r.Clamped, r.Stepped = cut(l, e.BP)
	r.Closed = e.Closed
	numberChanges := r.Value != l.Current || l.Mixed
	flagChanges := (e.Closed && l.AnyOpen) || (!e.Closed && l.AnyClosed)
	r.Outcome = Unchanged
	if numberChanges || flagChanges {
		r.Outcome = Applied
	}
}

// cut applies the two cuts of 6.2.2 and reports which moved the value.
func cut(l *Lever, bp int64) (v int64, clamped, stepped bool) {
	v = min(max(bp, l.Min), l.Max)
	clamped = v != bp
	if l.MaxStep > 0 {
		lo, hi := SatAdd(l.Current, -l.MaxStep), SatAdd(l.Current, l.MaxStep)
		w := min(max(v, lo), hi)
		stepped = w != v
		v = w
	}
	return v, clamped, stepped
}

// count adds one row to the tally.
func (t *Tally) count(r *Result) {
	switch r.Outcome {
	case NoClose:
		t.NoClose++
		return
	case Unchanged:
		t.Unchanged++
	case Applied:
		t.Applied++
	}
	if r.Clamped {
		t.Clamped++
	}
	if r.Stepped {
		t.Stepped++
	}
}
