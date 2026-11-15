package settlement

// MaxBits is the most flags, and the most limits, a table holds: one word per
// class (spec-settlements.md 8.1).
const MaxBits = 64

// Limit is a limit a class carries: its index in the table and its value.
type Limit struct {
	// Index is the position of the limit name among the module's limit names,
	// 0 to MaxBits-1.
	Index int
	// Value is the count, 0 or more.
	Value int64
}

// Class is what one class carries: flag indexes and limits.
type Class struct {
	// Flags are indexes of flag names, each 0 to MaxBits-1, a flag at most once.
	Flags []int
	// Limits are the limits of the class, an index at most once.
	Limits []Limit
}

// Table is the dense form of the flags and limits of the classes
// (spec-settlements.md 3.3, 6.1): one word of flags and one word of which limits
// exist per class, and the values in a flat slice. It is built once and read
// without allocating.
type Table struct {
	flags  []uint64
	has    []uint64
	values []int64
	limits int
}

// NewTable builds the table of classes for limits distinct limit names. ok is
// false for a limit count outside 0 to MaxBits, a flag or limit index outside
// 0 to MaxBits-1 (or the limit count), a negative value or a limit twice in one
// class. A flag twice in one class is the same bit and is accepted.
func NewTable(limits int, classes []Class) (t *Table, ok bool) {
	if limits < 0 || limits > MaxBits {
		return nil, false
	}
	t = &Table{
		flags:  make([]uint64, len(classes)),
		has:    make([]uint64, len(classes)),
		values: make([]int64, len(classes)*limits),
		limits: limits,
	}
	for k, c := range classes {
		if !t.fill(k, c) {
			return nil, false
		}
	}
	return t, true
}

// fill writes the class k into the table and reports whether it is valid.
func (t *Table) fill(k int, c Class) bool {
	for _, f := range c.Flags {
		if f < 0 || f >= MaxBits {
			return false
		}
		t.flags[k] |= 1 << uint(f)
	}
	for _, lim := range c.Limits {
		if lim.Index < 0 || lim.Index >= t.limits || lim.Value < 0 {
			return false
		}
		bit := uint64(1) << uint(lim.Index)
		if t.has[k]&bit != 0 {
			return false
		}
		t.has[k] |= bit
		t.values[k*t.limits+lim.Index] = lim.Value
	}
	return true
}

// Has reports whether the class carries the flag; false for an index out of range.
func (t *Table) Has(class, flag int) bool {
	if class < 0 || class >= len(t.flags) || flag < 0 || flag >= MaxBits {
		return false
	}
	return t.flags[class]>>uint(flag)&1 == 1
}

// Limit returns the value of the limit for the class. ok is false when the class
// has no row for it: no limit, which is not the same as a value of 0, a closed
// gate (spec-settlements.md 3.3.5).
func (t *Table) Limit(class, limit int) (v int64, ok bool) {
	if class < 0 || class >= len(t.has) || limit < 0 || limit >= t.limits {
		return 0, false
	}
	if t.has[class]>>uint(limit)&1 == 0 {
		return 0, false
	}
	return t.values[class*t.limits+limit], true
}

// Classes returns the number of classes of the table.
func (t *Table) Classes() int { return len(t.flags) }
