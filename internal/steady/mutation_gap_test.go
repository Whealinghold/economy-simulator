package steady

import (
	"slices"
	"testing"
)

// TestNewProducerMutation pins when a producer is open every tick and when the
// greedy sequence is read instead.
//
// Guard: gomu survivors gap.go:25, 26, 30.
func TestNewProducerMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		year      []Run
		wantEvery bool
	}{
		// The total is the sum of the lengths 5 + 0, not the last length.
		{"total sums the runs", []Run{{Length: 5, FactorBP: 10000}, {Length: 0, FactorBP: 10000}}, true},
		// A closed run of one tick closes the year.
		{"one closed tick", []Run{{Length: 1, FactorBP: 0}, {Length: 3, FactorBP: 10000}}, false},
		// A zero-length closed run closes nothing.
		{"empty closed run", []Run{{Length: 0, FactorBP: 0}, {Length: 3, FactorBP: 10000}}, true},
		// A one-tick year is the smallest that is open every tick.
		{"one open tick", []Run{{Length: 1, FactorBP: 10000}}, true},
		{"no ticks", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NewProducer(2, tc.year); got.Every != tc.wantEvery {
				t.Errorf("Every = %v, want %v", got.Every, tc.wantEvery)
			}
		})
	}
}

// TestGapMutation pins the cyclic gap across the year end and inside it.
//
// Guard: gomu survivors gap.go:43, 61, 71.
func TestGapMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		year int64
		ps   []Producer
		want int64
	}{
		// One-tick year with nothing completing: the whole year, 1.
		{"year of one tick", 1, nil, 1},
		{"no ticks", 0, nil, 0},
		// Ticks 0 and 10 of 12: across the end 0 + 12 - 10 - 1 = 1; inside 10 - 0 - 1 = 9.
		{"inside gap is the longest", 12, []Producer{{Completions: []int64{0, 10}}}, 9},
		// Completions past the year end wrap: 13 mod 12 = 1, 24 mod 12 = 0.
		{"wrap", 12, []Producer{{Completions: []int64{13, 24}}}, 10},
		{"every", 12, []Producer{{Completions: []int64{3}}, {Every: true}}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Gap(tc.year, tc.ps); got != tc.want {
				t.Errorf("Gap = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestExceedsShelfMutation pins the two edges of the shelf-life test.
//
// Guard: gomu survivors gap.go:71.
func TestExceedsShelfMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		gap, shelf int64
		want       bool
	}{
		{5, 1, true}, {5, 0, false}, {5, -1, false}, {5, 5, false}, {5, 4, true}, {5, 6, false},
	}
	for _, tc := range tests {
		if got := ExceedsShelf(tc.gap, tc.shelf); got != tc.want {
			t.Errorf("ExceedsShelf(%d, %d) = %v, want %v", tc.gap, tc.shelf, got, tc.want)
		}
	}
}

// TestSpansMutation pins which stretches a source produces in.
//
// Guard: gomu survivors gap.go:98, 102, 112.
func TestSpansMutation(t *testing.T) {
	t.Parallel()
	one := Rat(1, 1)
	tests := []struct {
		name string
		src  Source
		year int64
		want []span
	}{
		// Factor 1 bp is open: 1 open tick, base = 1 * 4 * 1 / 1 = 4, rate 4 / 10 000.
		{"one basis point is open", Source{Duration: 1, Flow: one, Yield: 1, Year: []Run{{Length: 1, FactorBP: 1}}}, 4, []span{{lo: 0, hi: 1}}},
		{"all closed", Source{Duration: 1, Flow: one, Yield: 1, Year: []Run{{Length: 3, FactorBP: 0}}}, 4, nil},
		{"zero yield", Source{Duration: 1, Flow: one, Yield: 0, Year: []Run{{Length: 3, FactorBP: 10000}}}, 4, nil},
		{"zero flow", Source{Duration: 1, Flow: Rat(0, 1), Yield: 1, Year: []Run{{Length: 3, FactorBP: 10000}}}, 4, nil},
		{"empty run", Source{Duration: 1, Flow: one, Yield: 1, Year: []Run{{Length: 0, FactorBP: 10000}, {Length: 2, FactorBP: 10000}}}, 4, []span{{lo: 0, hi: 2}}},
		// Completion exactly at the year end, tick 4: nothing credited.
		{"completes at year end", Source{Duration: 5, Flow: one, Yield: 1, Year: []Run{{Length: 4, FactorBP: 10000}}}, 4, nil},
		{"completes past year end", Source{Duration: 9, Flow: one, Yield: 1, Year: []Run{{Length: 4, FactorBP: 10000}}}, 4, nil},
		// Closed run first: the span after it starts at tick 2 + 1 - 1.
		{"closed run skipped", Source{Duration: 1, Flow: one, Yield: 1, Year: []Run{{Length: 2, FactorBP: 0}, {Length: 2, FactorBP: 10000}}}, 4, []span{{lo: 2, hi: 4}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.src.spans(tc.year)
			if len(got) != len(tc.want) {
				t.Fatalf("spans = %v, want %d of them", got, len(tc.want))
			}
			for i := range got {
				if got[i].lo != tc.want[i].lo || got[i].hi != tc.want[i].hi {
					t.Errorf("span %d = [%d, %d), want [%d, %d)", i, got[i].lo, got[i].hi, tc.want[i].lo, tc.want[i].hi)
				}
			}
		})
	}
}

// TestReserveMutation checks the lean reserve on overlapping stretches worked
// by hand.
//
// Guard: gomu survivors gap.go:139-150 (coverage test of a stretch over a cut
// interval, accumulation).
func TestReserveMutation(t *testing.T) {
	t.Parallel()
	// Year 12, consumption 3 per tick. Source A produces 3/2 per tick on
	// [2, 10): 8 open ticks, base 1 * 12 * 1 / 8 = 3/2. Source B produces 6 per
	// tick on [4, 6): 2 open ticks, base 12 / 2 = 6. Cuts 0 2 4 6 10 12.
	//   [0,2):  3 * 2           =  6  level  6
	//   [2,4):  (3 - 3/2) * 2   =  3  level  9
	//   [4,6):  (3 - 3/2 - 6) * 2 = -9  level  0
	//   [6,10): (3 - 3/2) * 4   =  6  level  6
	//   [10,12): 3 * 2          =  6  level 12   the peak
	a := Source{Duration: 1, Flow: Rat(1, 1), Yield: 1, Year: []Run{{Length: 2, FactorBP: 0}, {Length: 8, FactorBP: 10000}, {Length: 2, FactorBP: 0}}}
	b := Source{Duration: 1, Flow: Rat(1, 1), Yield: 1, Year: []Run{{Length: 4, FactorBP: 0}, {Length: 2, FactorBP: 10000}, {Length: 6, FactorBP: 0}}}
	if got := Reserve(12, Rat(3, 1), []Source{a, b}); got.Cmp(Rat(12, 1)) != 0 {
		t.Errorf("Reserve = %v/%v, want 12", got.Num, got.Den)
	}
	// No source: the whole year's consumption, 1 * 5.
	if got := Reserve(5, Rat(1, 1), nil); got.Cmp(Rat(5, 1)) != 0 {
		t.Errorf("Reserve without sources = %v/%v, want 5", got.Num, got.Den)
	}
}

// TestProducibleMutation pins the realm vector for a realm with no regions and
// the pooled reading of makers in a second region.
//
// Guard: gomu survivors producible.go:40, 41, 48.
func TestProducibleMutation(t *testing.T) {
	t.Parallel()
	t.Run("no regions", func(t *testing.T) {
		t.Parallel()
		r := Realm{Commodities: 2, Regions: 0, Sourced: []bool{true, false}, Makers: nil}
		got := Producible(r)
		if want := []bool{true, false}; !slices.Equal(got.Realm, want) {
			t.Errorf("Realm = %v, want %v", got.Realm, want)
		}
	})
	t.Run("pooled over regions", func(t *testing.T) {
		t.Parallel()
		// Region 1 makes commodity 1 from the sourced 0; region 0 makes 2 from 1.
		r := Realm{Commodities: 3, Regions: 2, Sourced: []bool{true, false, false}, Makers: []Maker{
			{Region: 0, Needs: []int{1}, Makes: []int{2}},
			{Region: 1, Needs: []int{0}, Makes: []int{1}},
		}}
		got := Producible(r)
		if want := []bool{true, true, true}; !slices.Equal(got.Realm, want) {
			t.Errorf("Realm = %v, want %v", got.Realm, want)
		}
		if want := []bool{true, false, false}; !slices.Equal(got.Region[0], want) {
			t.Errorf("Region[0] = %v, want %v", got.Region[0], want)
		}
		// Region 1 alone makes 1 and cannot make 2 (that maker is in region 0).
		if want := []bool{true, true, false}; !slices.Equal(got.Region[1], want) {
			t.Errorf("Region[1] = %v, want %v", got.Region[1], want)
		}
	})
}

// TestUnproducibleMutation pins that By names the lowest listed maker.
//
// Guard: gomu survivors producible.go:113.
func TestUnproducibleMutation(t *testing.T) {
	t.Parallel()
	r := Realm{Commodities: 3, Regions: 1, Sourced: []bool{false, false, false}, Makers: []Maker{
		{Needs: []int{2}, Makes: []int{1}},               // unlisted: does not name commodity 2
		{Needs: []int{2}, Makes: []int{0}, Listed: true}, // the lowest listed maker
		{Needs: []int{2}, Makes: []int{0}, Listed: true},
	}}
	got := Unproducible(r, nil)
	if want := []Missing{{Commodity: 2, By: 1}}; !slices.Equal(got, want) {
		t.Errorf("Unproducible = %v, want %v", got, want)
	}
	r.Makers = r.Makers[1:]
	if got, want := Unproducible(r, nil), []Missing{{Commodity: 2, By: 0}}; !slices.Equal(got, want) {
		t.Errorf("Unproducible = %v, want %v", got, want)
	}
}

// TestCannotWorkMutation pins that a role with no recipes does not end the scan.
//
// Guard: gomu survivor producible.go:150 (continue -> break).
func TestCannotWorkMutation(t *testing.T) {
	t.Parallel()
	roles := []RoleUse{{Regions: []int{0}}, {Regions: []int{0}, Recipes: []int{0}}}
	got := CannotWork(roles, [][]bool{{false}})
	if want := []Stuck{{Role: 1, Region: 0}}; !slices.Equal(got, want) {
		t.Errorf("CannotWork = %v, want %v", got, want)
	}
}
