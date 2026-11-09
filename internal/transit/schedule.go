package transit

// Schedule is the arrival schedule of the shipments in flight (spec-transport.md
// 14, NFR-TRN-002): a min-heap of (tick, ID) pairs, so that the shipments due in a
// tick come out in ascending ID and a tick with no arrival costs one comparison, whatever
// is in flight. The zero value is empty and ready.
type Schedule struct {
	h     []entry
	steps int64
}

type entry struct{ at, id int64 }

// less orders by tick and then by ID, so that ties come out in shipment order.
func (a entry) less(b entry) bool { return a.at < b.at || (a.at == b.at && a.id < b.id) }

// Push adds a shipment id that arrives at tick at.
func (s *Schedule) Push(at, id int64) {
	s.h = append(s.h, entry{at, id})
	for i := len(s.h) - 1; i > 0; {
		p := (i - 1) / 2
		s.steps++
		if !s.h[i].less(s.h[p]) {
			break
		}
		s.h[i], s.h[p] = s.h[p], s.h[i]
		i = p
	}
}

// Due removes and returns the entry with the lowest (tick, ID) if its tick is at
// most tick, or ok false when none is due. A tick below the one asked for is not
// expected (an arrival is processed in its tick or held to the next), but it comes
// out first if it is there.
func (s *Schedule) Due(tick int64) (id int64, ok bool) {
	s.steps++
	if len(s.h) == 0 || s.h[0].at > tick {
		return 0, false
	}
	id = s.h[0].id
	last := len(s.h) - 1
	s.h[0] = s.h[last]
	s.h = s.h[:last]
	for i := 0; ; {
		l, r, m := 2*i+1, 2*i+2, i
		if l < len(s.h) && s.h[l].less(s.h[m]) {
			m = l
		}
		if r < len(s.h) && s.h[r].less(s.h[m]) {
			m = r
		}
		s.steps += 2
		if m == i {
			break
		}
		s.h[i], s.h[m] = s.h[m], s.h[i]
		i = m
	}
	return id, true
}

// Len returns the number of entries, stale ones included.
func (s *Schedule) Len() int { return len(s.h) }

// Reset empties the schedule, keeping its storage.
func (s *Schedule) Reset() { s.h = s.h[:0] }

// Steps returns the number of comparisons the schedule has made: the measure of
// the cost of arrivals that NFR-TRN-002 asks to follow the arrivals.
func (s *Schedule) Steps() int64 { return s.steps }
