package rumour

// Obs is what one source market published in one tick: the unit of the price
// record (spec-information.md 4.2). It holds no reader and no error.
type Obs struct {
	// Price is the published market price, at least 1.
	Price int64
	// Mean is the price window's mean after that tick's push, 0 when the window
	// was empty.
	Mean int64
	// Age is the price age published that tick, at least 0.
	Age int64
}

// Valid reports whether the observation is one a market can publish: a price
// from 1 to maxPrice, a mean from 0 to maxPrice and an age of at least 0. A
// restore uses it to refuse corrupt bytes.
func (o Obs) Valid(maxPrice int64) bool {
	return o.Price >= 1 && o.Price <= maxPrice && o.Mean >= 0 && o.Mean <= maxPrice && o.Age >= 0
}

// Record is a ring of depth observations for each of a fixed number of sources,
// in one slice: the cell of a source and a tick is source*depth + tick%depth
// (spec-information.md 4.4.1). Every source is written every tick, so all hold
// the same number of entries and the record keeps one count. The zero value is
// not usable; build one with NewRecord.
type Record struct {
	depth   int64
	sources int
	cells   []Obs
	// last is the newest tick sealed, -1 while none; held is the number of ticks
	// held, the ticks last-held+1 to last.
	last int64
	held int64
	// restored is set by the first Load; the later ones must agree with it.
	restored bool
}

// NewRecord returns an empty record of sources rings of depth cells. depth is
// at least 1 and sources at least 0, else it panics (a violated invariant: the
// root validates both before it builds one).
func NewRecord(sources, depth int) *Record {
	if sources < 0 || depth < 1 {
		panic("rumour: invariant violated: record needs sources >= 0 and depth >= 1")
	}
	return &Record{
		depth:   int64(depth),
		sources: sources,
		cells:   make([]Obs, sources*depth),
		last:    -1,
	}
}

// Depth returns the number of ticks a ring holds.
func (r *Record) Depth() int { return int(r.depth) }

// Sources returns the number of rings.
func (r *Record) Sources() int { return r.sources }

// Held returns the number of ticks held, 0 to the depth.
func (r *Record) Held() int { return int(r.held) }

// Put writes the observation of source for tick, overwriting the cell of the
// tick one depth earlier. It does not count the tick as held: Seal does, once
// every source is written. tick must be the next tick to seal and source in
// range, else it panics (a violated invariant: one caller writes the record, in
// the position that owns it).
func (r *Record) Put(source int, tick int64, o Obs) {
	if source < 0 || source >= r.sources || tick != r.last+1 {
		panic("rumour: invariant violated: Put out of range or out of order")
	}
	r.cells[int64(source)*r.depth+tick%r.depth] = o
}

// Seal counts tick as held, after Put has written every source for it. tick
// must be the next tick, else it panics (a violated invariant).
func (r *Record) Seal(tick int64) {
	if tick != r.last+1 {
		panic("rumour: invariant violated: Seal out of order")
	}
	r.last = tick
	r.held = min(r.held+1, r.depth)
}

// At returns the observation of source for the vintage tick. ok is false when
// the vintage is outside the ticks held: before the first entry, overwritten
// (a lag above the depth) or not yet written (spec-information.md 5.4.1).
func (r *Record) At(source int, vintage int64) (o Obs, ok bool) {
	if source < 0 || source >= r.sources || vintage > r.last || vintage <= r.last-r.held {
		return Obs{}, false
	}
	return r.cells[int64(source)*r.depth+vintage%r.depth], true
}

// Entries appends the held observations of source to dst, oldest first, and
// returns it, the order a checkpoint saves them (spec-information.md 14). It
// returns dst unchanged for a source out of range.
func (r *Record) Entries(dst []Obs, source int) []Obs {
	if source < 0 || source >= r.sources {
		return dst
	}
	for v := r.last - r.held + 1; v <= r.last; v++ {
		dst = append(dst, r.cells[int64(source)*r.depth+v%r.depth])
	}
	return dst
}

// Load restores source from entries, oldest first, as a checkpoint taken before
// the tick next saved them: entry i of n is the observation of tick
// next-n+i, so the newest is the tick before next (spec-information.md 14). ok
// is false, changing nothing, when source is out of range, next is negative,
// there are more entries than the depth or than next ticks have run, or the
// count differs from the one an earlier Load set (every source holds the same
// number). The caller validates each Obs with Valid.
func (r *Record) Load(source int, next int64, entries []Obs) (ok bool) {
	n := int64(len(entries))
	if !r.canLoad(source, next, n) {
		return false
	}
	for i, o := range entries {
		r.cells[int64(source)*r.depth+(next-n+int64(i))%r.depth] = o
	}
	r.last, r.held, r.restored = next-1, n, true
	return true
}

// Resume sets the ticks held of a record with no source, as a checkpoint taken
// before tick next: the ticks run, at most the depth. It is Load for the record
// that has no source to load, and it fails like Load: ok is false, changing
// nothing, when next is negative, the record has sources (they are loaded one by
// one) or it has already run or been restored.
func (r *Record) Resume(next int64) (ok bool) {
	if r.sources != 0 || next < 0 || r.restored || r.last != -1 {
		return false
	}
	r.last, r.held, r.restored = next-1, min(next, r.depth), true
	return true
}

// canLoad reports whether n entries for source, as of tick next, fit the
// record: the arguments are in range, and they agree with an earlier Load or
// the record has not run.
func (r *Record) canLoad(source int, next, n int64) bool {
	if source < 0 || source >= r.sources || next < 0 || n > r.depth || n > next {
		return false
	}
	if r.restored {
		return r.last == next-1 && r.held == n
	}
	return r.last == -1
}
