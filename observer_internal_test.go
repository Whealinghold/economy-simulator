package economysim

import (
	"slices"
	"testing"
)

// recorder returns an observer that appends label to *log for each event it
// receives.
func recorder(log *[]string, label string) Observer {
	return ObserverFunc(func(Event) { *log = append(*log, label) })
}

// TestNewObservers_SuppliedFlag verifies that the flag the world reads before
// it builds an event is false with no observer and with only nil ones, and
// true as soon as one real observer is given, so that the no-op default builds
// nothing.
//
// Requirement: FR-OBS-090, FR-WLD-003
func TestNewObservers_SuppliedFlag(t *testing.T) {
	t.Parallel()
	var log []string
	tests := []struct {
		name  string
		given []Observer
		want  bool
		n     int
	}{
		{"none", nil, false, 0},
		{"empty", []Observer{}, false, 0},
		{"nil interface", []Observer{nil}, false, 0},
		{"nil func", []Observer{ObserverFunc(nil)}, false, 0},
		{"nils only", []Observer{nil, ObserverFunc(nil), nil}, false, 0},
		{"one", []Observer{recorder(&log, "a")}, true, 1},
		{"nil among real", []Observer{nil, recorder(&log, "a"), ObserverFunc(nil), recorder(&log, "b")}, true, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := newObservers(tt.given)
			if got.supplied != tt.want || len(got.list) != tt.n {
				t.Errorf("newObservers(%d given) = supplied %v with %d observers, want %v with %d",
					len(tt.given), got.supplied, len(got.list), tt.want, tt.n)
			}
		})
	}
}

// TestObservers_DeliverInOrder verifies that every event reaches each observer
// in the order the observers were given, once, and that nil observers among
// them are skipped without disturbing the order.
//
// Requirement: FR-OBS-024, FR-WLD-003
func TestObservers_DeliverInOrder(t *testing.T) {
	t.Parallel()
	var log []string
	o := newObservers([]Observer{
		recorder(&log, "a"), nil, recorder(&log, "b"), ObserverFunc(nil), recorder(&log, "c"),
	})
	o.deliver(TickStarted{Tick: 1})
	o.deliver(TickEnded{Tick: 1})
	want := []string{"a", "b", "c", "a", "b", "c"}
	if !slices.Equal(log, want) {
		t.Errorf("delivery order of two events to observers a, b, c = %v, want %v", log, want)
	}
}

// TestObservers_ReceiveTheEvent verifies that an observer receives the event
// value that was delivered, whole, so that a type switch on it works.
//
// Requirement: FR-OBS-024
func TestObservers_ReceiveTheEvent(t *testing.T) {
	t.Parallel()
	var got []Event
	o := newObservers([]Observer{ObserverFunc(func(e Event) { got = append(got, e) })})
	want := TradeExecuted{Tick: 3, Region: 1, Commodity: 2, Buyer: 4, Seller: 5, Price: 6, Quantity: 7}
	o.deliver(want)
	if len(got) != 1 || got[0] != Event(want) {
		t.Errorf("deliver(%+v) delivered %+v, want the same event once", want, got)
	}
	tr, ok := got[0].(TradeExecuted)
	if !ok || tr.Price != 6 {
		t.Errorf("delivered event type-switches to %+v, %v, want TradeExecuted with Price 6", tr, ok)
	}
}

// TestNewObservers_CopiesGivenSlice verifies that changing the slice given to
// newObservers afterwards does not change who is observing, because the world
// records its observers once and never changes them.
//
// Guard: an observers value that kept the caller's slice would let a later
// edit of the options change delivery, and the flag would disagree with the
// list.
func TestNewObservers_CopiesGivenSlice(t *testing.T) {
	t.Parallel()
	var log []string
	given := []Observer{recorder(&log, "a"), recorder(&log, "b")}
	o := newObservers(given)
	given[0] = recorder(&log, "x")
	given[1] = nil
	o.deliver(TickStarted{})
	if want := []string{"a", "b"}; !slices.Equal(log, want) {
		t.Errorf("delivery after the given slice was edited = %v, want %v", log, want)
	}
}

// TestObservers_PanicPropagates verifies that the library does not recover a
// panicking observer: the panic reaches the caller of deliver, the observers
// before it have the event and the ones after it do not.
//
// Requirement: FR-OBS-024
func TestObservers_PanicPropagates(t *testing.T) {
	t.Parallel()
	var log []string
	o := newObservers([]Observer{
		recorder(&log, "a"),
		ObserverFunc(func(Event) { panic("observer failed") }),
		recorder(&log, "c"),
	})
	var got any
	func() {
		defer func() { got = recover() }()
		o.deliver(TickStarted{})
	}()
	if got != "observer failed" {
		t.Errorf("deliver with a panicking observer recovered %v, want the observer's panic", got)
	}
	if want := []string{"a"}; !slices.Equal(log, want) {
		t.Errorf("observers reached before the panic = %v, want %v", log, want)
	}
}

// TestDiscardObserver_DiscardsEverything verifies that the default observer
// accepts every event and does nothing, as an Observer.
func TestDiscardObserver_DiscardsEverything(t *testing.T) {
	t.Parallel()
	var o Observer = discardObserver{}
	o.Observe(TickStarted{})
	o.Observe(BoundaryFlow{Tick: -1})
}

// TestWarnRun_IsTen verifies the rule constant of the bounded Warn: the run
// length is 10, the figure the pattern bounds of spec-observability.md 7.3
// are written for.
//
// Requirement: FR-OBS-072
func TestWarnRun_IsTen(t *testing.T) {
	t.Parallel()
	if warnRun != 10 {
		t.Errorf("warnRun = %d, want 10", warnRun)
	}
}

// TestObservers_DeliverAllocs verifies the no-op path of
// observability-and-logging.md 4.5 and 8.2: with no observer supplied, the
// delivery loop allocates nothing and a guarded site that would build an event
// allocates nothing, because the event is neither built nor boxed; and with
// observers supplied, delivering an event already boxed allocates nothing.
//
// Requirement: NFR-OBS-001
func TestObservers_DeliverAllocs(t *testing.T) {
	// No t.Parallel: AllocsPerRun changes GOMAXPROCS while it measures.
	none := newObservers(nil)
	var count int
	some := newObservers([]Observer{
		ObserverFunc(func(Event) { count++ }),
		discardObserver{},
	})
	boxed := Event(TickStarted{Tick: 7})
	tests := []struct {
		name string
		f    func()
	}{
		{"deliver with none", func() { none.deliver(boxed) }},
		{"guarded site with none", func() {
			if none.supplied {
				none.deliver(TradeExecuted{Tick: 7, Price: 3, Quantity: 2})
			}
		}},
		{"deliver boxed event to observers", func() { some.deliver(boxed) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(100, tt.f); got != 0 {
				t.Errorf("AllocsPerRun(%s) = %v, want 0", tt.name, got)
			}
		})
	}
	if count == 0 {
		t.Errorf("observer was never called, count = %d, want > 0", count)
	}
}

// BenchmarkObserversDeliver reports the cost and allocations of delivery with
// no observer, and with three, beside the allocation test.
func BenchmarkObserversDeliver(b *testing.B) {
	boxed := Event(TickStarted{Tick: 7})
	var n int
	three := newObservers([]Observer{
		ObserverFunc(func(Event) { n++ }),
		ObserverFunc(func(Event) { n++ }),
		ObserverFunc(func(Event) { n++ }),
	})
	b.Run("none", func(b *testing.B) {
		none := newObservers(nil)
		b.ReportAllocs()
		for b.Loop() {
			if none.supplied {
				none.deliver(TradeExecuted{Tick: 7, Price: 3, Quantity: 2})
			}
			none.deliver(boxed)
		}
	})
	b.Run("three", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			three.deliver(boxed)
		}
	})
	_ = n
}
