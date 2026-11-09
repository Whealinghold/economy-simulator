package rumour_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/rumour"
)

// obsOf is the observation of source s at tick t in these tests: distinct for
// every pair, so a read of the wrong cell is a different value.
func obsOf(s int, t int64) rumour.Obs {
	return rumour.Obs{Price: 100*int64(s) + t + 1, Mean: 10*int64(s) + t, Age: t % 3}
}

// run writes ticks 0 to last of every source of r.
func run(r *rumour.Record, last int64) {
	for tick := int64(0); tick <= last; tick++ {
		for s := 0; s < r.Sources(); s++ {
			r.Put(s, tick, obsOf(s, tick))
		}
		r.Seal(tick)
	}
}

// TestRecord_EmptyAtStart verifies that a new record holds nothing and refuses
// every read: it is not filled from seed prices (spec-information.md 4.3.3).
//
// Requirement: FR-INF-007
func TestRecord_EmptyAtStart(t *testing.T) {
	t.Parallel()
	r := rumour.NewRecord(3, 4)
	if r.Held() != 0 || r.Depth() != 4 || r.Sources() != 3 {
		t.Errorf("NewRecord(3, 4) holds %d, depth %d, sources %d, want 0, 4, 3", r.Held(), r.Depth(), r.Sources())
	}
	for s := 0; s < 3; s++ {
		for _, v := range []int64{-1, 0, 1} {
			if o, ok := r.At(s, v); ok {
				t.Errorf("At(%d, %d) on an empty record = %v, true, want ok false", s, v, o)
			}
		}
		if got := r.Entries(nil, s); len(got) != 0 {
			t.Errorf("Entries(nil, %d) on an empty record = %v, want none", s, got)
		}
	}
}

// TestRecord_RingKeepsLastD verifies that after more than depth ticks a ring
// holds exactly the last depth ticks of each source, oldest first, and has
// overwritten the older ones: ticks 0 to 6 at depth 3 hold 4, 5 and 6.
//
// Requirement: FR-INF-007
func TestRecord_RingKeepsLastD(t *testing.T) {
	t.Parallel()
	r := rumour.NewRecord(2, 3)
	run(r, 6)
	if r.Held() != 3 {
		t.Fatalf("Held() after 7 ticks at depth 3 = %d, want 3", r.Held())
	}
	for s := 0; s < 2; s++ {
		want := []rumour.Obs{obsOf(s, 4), obsOf(s, 5), obsOf(s, 6)}
		if got := r.Entries(nil, s); !slices.Equal(got, want) {
			t.Errorf("Entries(nil, %d) = %v, want %v", s, got, want)
		}
	}
	// Fewer ticks than the depth: held counts the ticks run.
	short := rumour.NewRecord(1, 5)
	run(short, 1)
	if short.Held() != 2 {
		t.Errorf("Held() after 2 ticks at depth 5 = %d, want 2", short.Held())
	}
}

// TestRecord_VintageAt verifies the read of a vintage: the newest tick and the
// oldest held (a lag of the depth read one tick later) are returned, the tick
// before it was overwritten and is refused, a tick not yet written and a
// negative one are refused, and a source out of range is refused.
//
// Requirement: FR-INF-017
func TestRecord_VintageAt(t *testing.T) {
	t.Parallel()
	const depth, last = 4, 9
	r := rumour.NewRecord(2, depth)
	run(r, last)
	tests := []struct {
		name    string
		source  int
		vintage int64
		want    bool
	}{
		{"newest tick", 1, last, true},
		{"oldest held, the vintage of lag D read at tick last+1", 1, last - depth + 1, true},
		{"overwritten cell is refused", 1, last - depth, false},
		{"not yet written", 0, last + 1, false},
		{"first source reads", 0, last, true},
		{"negative vintage", 0, -1, false},
		{"source out of range", 2, last, false},
		{"negative source", -1, last, false},
		{"source far out of range", 5, last, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := r.At(tt.source, tt.vintage)
			if ok != tt.want || ok && got != obsOf(tt.source, tt.vintage) {
				t.Errorf("At(%d, %d) = %v, %v, want %v, %v", tt.source, tt.vintage, got, ok, obsOf(tt.source, tt.vintage), tt.want)
			}
		})
	}
}

// TestRecord_RestoreOrder verifies the checkpoint round trip: the entries saved
// oldest first and loaded as of tick T place entry i of n at T - n + i, the
// restored record reads like the original, and a record that has run for fewer
// ticks than the depth (n < depth) restores to the same state.
//
// Requirement: FR-INF-007, FR-INF-017
func TestRecord_RestoreOrder(t *testing.T) {
	t.Parallel()
	for _, ran := range []int64{2, 3, 7} { // ticks run: below, at and above the depth 4
		orig := rumour.NewRecord(2, 4)
		run(orig, ran-1)
		got := rumour.NewRecord(2, 4)
		for s := 0; s < 2; s++ {
			if !got.Load(s, ran, orig.Entries(nil, s)) {
				t.Fatalf("ran %d: Load(%d) refused its own entries", ran, s)
			}
		}
		if got.Held() != orig.Held() {
			t.Errorf("ran %d: restored Held() = %d, want %d", ran, got.Held(), orig.Held())
		}
		for s := 0; s < 2; s++ {
			for v := int64(-1); v <= ran; v++ {
				wo, wok := orig.At(s, v)
				go_, gok := got.At(s, v)
				if go_ != wo || gok != wok {
					t.Errorf("ran %d: restored At(%d, %d) = %v, %v, want %v, %v", ran, s, v, go_, gok, wo, wok)
				}
			}
		}
		// The restored record goes on like the original.
		for s := 0; s < 2; s++ {
			orig.Put(s, ran, obsOf(s, ran))
			got.Put(s, ran, obsOf(s, ran))
		}
		orig.Seal(ran)
		got.Seal(ran)
		for s := 0; s < 2; s++ {
			if a, b := orig.Entries(nil, s), got.Entries(nil, s); !slices.Equal(a, b) {
				t.Errorf("ran %d: after one more tick Entries(nil, %d) = %v, want %v", ran, s, b, a)
			}
		}
	}
}

// TestRecord_LoadRefusals verifies that Load refuses, and changes nothing for,
// a source out of range, more entries than the depth, more entries than ticks
// run, a count that differs from an earlier Load, a different tick, and a record
// that has already run.
//
// Requirement: FR-INF-007
func TestRecord_LoadRefusals(t *testing.T) {
	t.Parallel()
	three := []rumour.Obs{obsOf(0, 1), obsOf(0, 2), obsOf(0, 3)}
	tests := []struct {
		name    string
		prepare func(r *rumour.Record)
		source  int
		next    int64
		entries []rumour.Obs
	}{
		{"source out of range", func(*rumour.Record) {}, 2, 4, three},
		{"negative source", func(*rumour.Record) {}, -1, 4, three},
		{"more entries than the depth", func(*rumour.Record) {}, 0, 9, append(slices.Clone(three), obsOf(0, 4), obsOf(0, 5))},
		{"more entries than ticks run", func(*rumour.Record) {}, 0, 2, three},
		{"negative tick", func(*rumour.Record) {}, 0, -1, nil},
		{"count differs from an earlier load", func(r *rumour.Record) { r.Load(0, 4, three) }, 1, 4, three[:2]},
		{"tick differs from an earlier load", func(r *rumour.Record) { r.Load(0, 4, three) }, 1, 5, three},
		{"record already ran", func(r *rumour.Record) { run(r, 3) }, 0, 4, three},
		{"earlier tick than an earlier load, same count", func(r *rumour.Record) { r.Load(0, 4, three) }, 1, 3, three},
		{"source far out of range", func(*rumour.Record) {}, 9, 4, three},
		{"more entries than an earlier load", func(r *rumour.Record) { r.Load(0, 4, three) }, 1, 4, append(slices.Clone(three), obsOf(0, 0))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := rumour.NewRecord(2, 4)
			tt.prepare(r)
			held := r.Held()
			before := r.Entries(nil, tt.source)
			if r.Load(tt.source, tt.next, tt.entries) {
				t.Errorf("Load(%d, %d, %d entries) = true, want false", tt.source, tt.next, len(tt.entries))
			}
			if r.Held() != held || !slices.Equal(r.Entries(nil, tt.source), before) {
				t.Errorf("a refused Load changed the record: held %d -> %d", held, r.Held())
			}
		})
	}
}

// TestObs_Valid verifies the bounds a restore tests each loaded observation
// against: a price from 1 to the bound, a mean from 0 to the bound, an age of
// at least 0.
//
// Requirement: FR-INF-005
func TestObs_Valid(t *testing.T) {
	t.Parallel()
	const maxPrice = 1_000
	tests := []struct {
		name string
		o    rumour.Obs
		want bool
	}{
		{"ordinary", rumour.Obs{Price: 370, Mean: 366, Age: 0}, true},
		{"no mean", rumour.Obs{Price: 1, Mean: 0, Age: 7}, true},
		{"price and mean at the bound", rumour.Obs{Price: maxPrice, Mean: maxPrice}, true},
		{"price 0", rumour.Obs{Price: 0, Mean: 5}, false},
		{"price above the bound", rumour.Obs{Price: maxPrice + 1}, false},
		{"mean above the bound", rumour.Obs{Price: 5, Mean: maxPrice + 1}, false},
		{"negative mean", rumour.Obs{Price: 5, Mean: -1}, false},
		{"negative age", rumour.Obs{Price: 5, Age: -1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.o.Valid(maxPrice); got != tt.want {
				t.Errorf("%v.Valid(%d) = %v, want %v", tt.o, maxPrice, got, tt.want)
			}
		})
	}
}

// TestRecord_Invariants verifies that a write out of order or out of range is a
// programmer error that panics with "invariant", and that a record needs a depth
// of at least 1.
//
// Guard: a gap in the ticks written would shift every vintage silently; the
// panic is the only thing that would point at it.
func TestRecord_Invariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		do   func()
	}{
		{"Put for a tick that is not the next", func() { rumour.NewRecord(1, 2).Put(0, 1, rumour.Obs{}) }},
		{"Put for a source out of range", func() { rumour.NewRecord(1, 2).Put(1, 0, rumour.Obs{}) }},
		{"Seal for a tick that is not the next", func() { rumour.NewRecord(1, 2).Seal(1) }},
		{"Put for a tick already sealed", func() {
			r := rumour.NewRecord(1, 2)
			r.Seal(0)
			r.Put(0, 0, rumour.Obs{})
		}},
		{"Put for a source far out of range", func() { rumour.NewRecord(1, 2).Put(5, 0, rumour.Obs{}) }},
		{"Put for a negative source", func() { rumour.NewRecord(1, 2).Put(-1, 0, rumour.Obs{}) }},
		{"Seal for a tick already sealed", func() {
			r := rumour.NewRecord(1, 2)
			r.Seal(0)
			r.Seal(0)
		}},
		{"depth 0", func() { rumour.NewRecord(1, 0) }},
		{"negative sources", func() { rumour.NewRecord(-1, 2) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				v, _ := recover().(string)
				if !strings.Contains(v, "invariant") {
					t.Errorf("%s: recovered %q, want a panic that says invariant", tt.name, v)
				}
			}()
			tt.do()
		})
	}
}

// TestRecord_ReadWriteAllocate verifies that writing a tick, sealing it, reading
// a vintage and saving the entries into a prepared slice allocate nothing.
//
// Requirement: NFR-INF-001
func TestRecord_ReadWriteAllocate(t *testing.T) {
	r := rumour.NewRecord(8, 6)
	run(r, 8)
	dst := make([]rumour.Obs, 0, 6)
	tick := int64(9)
	var sink rumour.Obs
	got := testing.AllocsPerRun(100, func() {
		for s := 0; s < 8; s++ {
			r.Put(s, tick, rumour.Obs{Price: 5})
		}
		r.Seal(tick)
		tick++
		for s := 0; s < 8; s++ {
			sink, _ = r.At(s, tick-3)
		}
		dst = r.Entries(dst[:0], 3)
	})
	if got != 0 {
		t.Errorf("a tick of Put, Seal, At and Entries allocated %v times, want 0 (last read %v)", got, sink)
	}
}

// TestRecord_Edges verifies the smallest records: no source at all (a module on
// where nothing is a source), a depth of 1 that keeps only the newest tick, a
// load of nothing at tick 0, and Entries for a source out of range, which
// returns the slice it was given.
//
// Requirement: FR-INF-007
func TestRecord_Edges(t *testing.T) {
	t.Parallel()
	none := rumour.NewRecord(0, 3)
	run(none, 4)
	if none.Held() != 3 || none.Sources() != 0 {
		t.Errorf("no sources: Held() = %d, Sources() = %d, want 3, 0", none.Held(), none.Sources())
	}
	one := rumour.NewRecord(1, 1)
	run(one, 3)
	if o, ok := one.At(0, 3); !ok || o != obsOf(0, 3) {
		t.Errorf("depth 1: At(0, 3) = %v, %v, want %v, true", o, ok, obsOf(0, 3))
	}
	if o, ok := one.At(0, 2); ok {
		t.Errorf("depth 1: At(0, 2) = %v, true, want ok false (overwritten)", o)
	}
	fresh := rumour.NewRecord(2, 3)
	if !fresh.Load(0, 0, nil) || !fresh.Load(1, 0, nil) {
		t.Errorf("Load(s, 0, nil) = false, want true: nothing has run, nothing is held")
	}
	run2 := rumour.NewRecord(2, 3)
	run(run2, 1)
	prefix := []rumour.Obs{{Price: 7}}
	for _, s := range []int{-1, 2, 9} {
		if got := run2.Entries(prefix, s); !slices.Equal(got, prefix) {
			t.Errorf("Entries(prefix, %d) = %v, want %v unchanged", s, got, prefix)
		}
	}
}

// TestRecord_Resume verifies that a record with no source is resumed as of a
// checkpoint tick: it then holds the ticks run up to the depth and goes on from
// the next tick, and Resume refuses a record that has sources, one that has
// already run or been restored, and a negative tick.
//
// Requirement: FR-INF-029
func TestRecord_Resume(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		sources  int
		prepare  func(r *rumour.Record)
		next     int64
		wantOK   bool
		wantHeld int
	}{
		{"before the depth", 0, func(*rumour.Record) {}, 2, true, 2},
		{"past the depth", 0, func(*rumour.Record) {}, 9, true, 4},
		{"at tick zero", 0, func(*rumour.Record) {}, 0, true, 0},
		{"negative tick", 0, func(*rumour.Record) {}, -1, false, 0},
		{"has sources", 1, func(*rumour.Record) {}, 3, false, 0},
		{"already run", 0, func(r *rumour.Record) { r.Seal(0) }, 3, false, 1},
		{"already restored", 0, func(r *rumour.Record) { r.Resume(2) }, 3, false, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := rumour.NewRecord(tt.sources, 4)
			tt.prepare(r)
			if ok := r.Resume(tt.next); ok != tt.wantOK || r.Held() != tt.wantHeld {
				t.Errorf("Resume(%d) = %v with %d held, want %v with %d", tt.next, ok, r.Held(), tt.wantOK, tt.wantHeld)
			}
		})
	}
	r := rumour.NewRecord(0, 4)
	r.Resume(6)
	r.Seal(6) // goes on from the tick after the checkpoint's last
	if r.Held() != 4 {
		t.Errorf("after Resume(6) and Seal(6), Held() = %d, want 4", r.Held())
	}
}
