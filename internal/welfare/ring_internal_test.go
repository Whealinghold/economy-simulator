package welfare

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

// checkRebuild asserts the invariant that makes "not saved" safe: the running
// sums and the count at the clamp equal those of a ring rebuilt from the
// ordered read (FR-WEL-034 in part).
func checkRebuild(t *testing.T, step string, r *Ring) {
	t.Helper()
	recs := r.AppendRecords(nil)
	if len(recs) != r.Len() {
		t.Fatalf("%s: ordered read has %d records, Len is %d", step, len(recs), r.Len())
	}
	b, err := FromRecords(recs, r.Window(), r.maxWindow)
	if err != nil {
		t.Fatalf("%s: FromRecords refused the ordered read of a live ring: %v", step, err)
	}
	if b.s != r.s {
		t.Fatalf("%s: sums %+v, rebuilt %+v", step, r.s, b.s)
	}
}

func rec(income, cost int64, working bool) Record {
	r := Record{Income: income, Cost: cost, Relief: income / 10, Unearned: 1, OwnReceipt: 2, Working: working}
	if working {
		r.WorkHH, r.OwnHH = 3, 1
	}
	return r
}

// TestRing_SumsEqualRebuildAfterEveryStep verifies a scripted sequence of
// pushes, evictions, resizes and clamped pushes keeps the sums equal to a
// rebuild from the ordered read after every step.
//
// Requirement: FR-WEL-034, FR-WEL-042, FR-WEL-010
func TestRing_SumsEqualRebuildAfterEveryStep(t *testing.T) {
	t.Parallel()
	const maxW = 10
	c := int64(math.MaxInt64 / maxW)
	r, err := NewRing(4, maxW)
	if err != nil {
		t.Fatal(err)
	}
	step := func(name string, f func()) {
		f()
		checkRebuild(t, name, &r)
	}
	for i := int64(1); i <= 6; i++ { // 6 pushes into 4: two evictions
		step("push", func() { r.Push(rec(i*100, i*10, i%2 == 0)) })
	}
	if got := r.AppendRecords(nil); len(got) != 4 || got[0].Income != 300 || got[3].Income != 600 {
		t.Errorf("after 6 pushes into 4, ordered read = %+v, want incomes 300..600", got)
	}
	step("resize down", func() {
		if err := r.Resize(2); err != nil {
			t.Fatal(err)
		}
	})
	if got := r.AppendRecords(nil); len(got) != 2 || got[0].Income != 500 || got[1].Income != 600 {
		t.Errorf("resize down to 2 kept %+v, want the newest two (500, 600)", got)
	}
	step("push after shrink", func() { r.Push(rec(700, 70, true)) })
	step("resize up", func() {
		if err := r.Resize(6); err != nil {
			t.Fatal(err)
		}
	})
	if r.Len() != 2 || r.Window() != 6 {
		t.Errorf("after resize up Len %d Window %d, want 2 and 6", r.Len(), r.Window())
	}
	step("clamped push", func() {
		if !r.Push(Record{Income: math.MaxInt64, Cost: math.MinInt64, Working: true, WorkHH: 1}) {
			t.Error("Push of MaxInt64 and MinInt64 did not report a clamp")
		}
	})
	if got := r.AppendRecords(nil); got[len(got)-1].Income != c || got[len(got)-1].Cost != -c {
		t.Errorf("stored record = %+v, want fields at +-%d", got[len(got)-1], c)
	}
	if !r.AtBound() {
		t.Error("AtBound false with a record at the clamp")
	}
	step("exactly at the bound is not a clamp", func() {
		if r.Push(Record{Income: c, Cost: -c}) {
			t.Error("a field exactly at the bound was reported as clamped")
		}
	})
	for i := 0; i < 6; i++ { // evict every record at the bound
		step("evict", func() { r.Push(rec(1, 1, false)) })
	}
	if r.AtBound() {
		t.Error("AtBound true after every record at the bound was evicted")
	}
}

// ringState is the whole observable state of a ring: the ordered records, the
// window, Len, every running sum and the at-bound count (inside sums).
type ringState struct {
	recs   []Record
	window int64
	n      int
	s      sums
}

func stateOf(r *Ring) ringState {
	return ringState{recs: r.AppendRecords(nil), window: r.Window(), n: r.Len(), s: r.s}
}

// TestRing_ResizeRefusalsLeaveTheRingUnchanged verifies a window of 0, below 0
// or above maxWindow is refused with ErrRecords and the ordered records,
// Window, Len, every running sum and the at-bound count are untouched (on a
// wrapped ring holding a record at the clamp), and that a resize to exactly
// maxWindow stays inside the proof with every field at C.
//
// Requirement: FR-WEL-010, FR-WEL-034
func TestRing_ResizeRefusalsLeaveTheRingUnchanged(t *testing.T) {
	t.Parallel()
	const maxW = 5
	c := int64(math.MaxInt64 / maxW)
	r, _ := NewRing(3, maxW)
	r.Push(rec(1, 1, true))
	r.Push(rec(10, 5, true))
	r.Push(Record{Income: c, Cost: 2}) // at the clamp
	r.Push(rec(20, 5, false))          // evicts the first: start is now 1
	if !r.AtBound() || r.start == 0 {
		t.Fatalf("setup: AtBound %v start %d, want a wrapped ring with a bound record", r.AtBound(), r.start)
	}
	before := stateOf(&r)
	for _, bad := range []int64{0, -1, math.MinInt64, maxW + 1, math.MaxInt64} {
		if err := r.Resize(bad); !errors.Is(err, ErrRecords) {
			t.Errorf("Resize(%d) = %v, want ErrRecords", bad, err)
		}
		if got := stateOf(&r); !reflect.DeepEqual(got, before) {
			t.Errorf("Resize(%d) changed the ring: %+v, want %+v", bad, got, before)
		}
	}
	if err := r.Resize(maxW); err != nil {
		t.Fatalf("Resize(maxWindow) = %v", err)
	}
	for i := 0; i < maxW+2; i++ {
		r.Push(Record{Income: c, Cost: -c, Relief: c, Unearned: -c, OwnReceipt: c, WorkHH: c, OwnHH: -c, Working: true})
	}
	checkRebuild(t, "full at the clamp", &r)
	if want := int64(maxW) * c; r.s.income != want || r.s.cost != -want {
		t.Errorf("sums income %d cost %d, want +-%d", r.s.income, r.s.cost, want)
	}
}

// TestRing_ResizeWithinCapacityAllocatesNothing verifies a shrink, a grow back
// within the capacity the ring already had, and a no-op resize allocate
// nothing. Growing past the capacity necessarily allocates and is not asserted.
//
// Requirement: NFR-WEL-001
func TestRing_ResizeWithinCapacityAllocatesNothing(t *testing.T) {
	// No t.Parallel: AllocsPerRun must not overlap other tests' allocations.
	r, _ := NewRing(10, 20)
	for i := int64(1); i <= 13; i++ { // wrapped: start is 3
		r.Push(rec(i, i, true))
	}
	if n := testing.AllocsPerRun(100, func() {
		_ = r.Resize(4)  // shrink
		_ = r.Resize(10) // grow back within the capacity of 10
		_ = r.Resize(10) // no-op
	}); n != 0 {
		t.Errorf("Resize within capacity allocated %v times per run, want 0", n)
	}
}

// ringFields names every numeric field of a record with a setter and a getter,
// written here and not taken from Record.fields so the table is independent.
var ringFields = []struct {
	name string
	set  func(*Record, int64)
	get  func(Record) int64
}{
	{"Income", func(r *Record, v int64) { r.Income = v }, func(r Record) int64 { return r.Income }},
	{"Cost", func(r *Record, v int64) { r.Cost = v }, func(r Record) int64 { return r.Cost }},
	{"Relief", func(r *Record, v int64) { r.Relief = v }, func(r Record) int64 { return r.Relief }},
	{"Unearned", func(r *Record, v int64) { r.Unearned = v }, func(r Record) int64 { return r.Unearned }},
	{"OwnReceipt", func(r *Record, v int64) { r.OwnReceipt = v }, func(r Record) int64 { return r.OwnReceipt }},
	{"WorkHH", func(r *Record, v int64) { r.WorkHH = v }, func(r Record) int64 { return r.WorkHH }},
	{"OwnHH", func(r *Record, v int64) { r.OwnHH = v }, func(r Record) int64 { return r.OwnHH }},
}

// TestRing_PushClampAndRestoreRefusalPerField verifies, for each of the seven
// fields: Push stores a value beyond +-C clamped to +-C and reports it, a value
// exactly at +-C is stored as is and not reported, both set AtBound, the sums
// equal a rebuild; and FromRecords refuses C+1 and -(C+1) with ErrRecords and
// accepts exactly +-C. C = MaxInt64 / maxWindow = 9223372036854775807 / 10.
//
// Requirement: FR-WEL-010, FR-WEL-034, FR-WEL-042
func TestRing_PushClampAndRestoreRefusalPerField(t *testing.T) {
	t.Parallel()
	const maxW = 10
	const c = int64(math.MaxInt64 / maxW)
	for _, f := range ringFields {
		for _, tc := range []struct {
			name    string
			in      int64
			stored  int64
			clamped bool
		}{
			{"MaxInt64", math.MaxInt64, c, true},
			{"MinInt64", math.MinInt64, -c, true},
			{"C+1", c + 1, c, true},
			{"-(C+1)", -(c + 1), -c, true},
			{"C", c, c, false},
			{"-C", -c, -c, false},
		} {
			t.Run(f.name+" push "+tc.name, func(t *testing.T) {
				t.Parallel()
				r, _ := NewRing(3, maxW)
				var in Record
				f.set(&in, tc.in)
				if got := r.Push(in); got != tc.clamped {
					t.Errorf("Push(%s=%d) reported %v, want %v", f.name, tc.in, got, tc.clamped)
				}
				got := r.AppendRecords(nil)
				if len(got) != 1 || f.get(got[0]) != tc.stored {
					t.Errorf("Push(%s=%d) stored %+v, want %s=%d", f.name, tc.in, got, f.name, tc.stored)
				}
				if !r.AtBound() {
					t.Errorf("AtBound false after %s=%d (stored %d)", f.name, tc.in, tc.stored)
				}
				checkRebuild(t, "push", &r)
			})
		}
		for _, tc := range []struct {
			name    string
			in      int64
			refused bool
		}{
			{"C+1", c + 1, true},
			{"-(C+1)", -(c + 1), true},
			{"C", c, false},
			{"-C", -c, false},
		} {
			t.Run(f.name+" restore "+tc.name, func(t *testing.T) {
				t.Parallel()
				var in Record
				f.set(&in, tc.in)
				r, err := FromRecords([]Record{{}, in}, 3, maxW)
				if tc.refused {
					if !errors.Is(err, ErrRecords) || !reflect.DeepEqual(r, Ring{}) {
						t.Errorf("FromRecords(%s=%d) = %+v, %v, want no ring and ErrRecords", f.name, tc.in, r, err)
					}
				} else if err != nil || !r.AtBound() {
					t.Errorf("FromRecords(%s=%d) = %v, AtBound %v, want accepted at the bound", f.name, tc.in, err, r.AtBound())
				}
			})
		}
	}
}

// TestRing_AtBoundCountAfterEvictionAndResize verifies the at-bound count falls
// when a record at the clamp leaves the ring by a Push or by a Resize shrink,
// and stays when the shrink keeps it.
//
// Requirement: FR-WEL-034, FR-WEL-042
func TestRing_AtBoundCountAfterEvictionAndResize(t *testing.T) {
	t.Parallel()
	const maxW = 10
	const c = int64(math.MaxInt64 / maxW)
	bound := Record{Cost: -c}
	plain := Record{Income: 1}
	fill := func(t *testing.T, window int64, recs ...Record) Ring {
		t.Helper()
		r, err := NewRing(window, maxW)
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range recs {
			r.Push(x)
		}
		return r
	}
	tests := []struct {
		name  string
		build func(t *testing.T) Ring
		act   func(r *Ring)
		want  int64
	}{
		{"push evicts the only bound record", func(t *testing.T) Ring { return fill(t, 3, bound, plain, plain) },
			func(r *Ring) { r.Push(plain) }, 0},
		{"push evicts one of two bound records", func(t *testing.T) Ring { return fill(t, 3, bound, bound, plain) },
			func(r *Ring) { r.Push(plain) }, 1},
		{"push evicts a plain record, bound stays", func(t *testing.T) Ring { return fill(t, 3, plain, bound, plain) },
			func(r *Ring) { r.Push(plain) }, 1},
		{"shrink evicts the bound record", func(t *testing.T) Ring { return fill(t, 4, plain, bound, plain, plain) },
			func(r *Ring) { _ = r.Resize(2) }, 0},
		{"shrink keeps the bound record", func(t *testing.T) Ring { return fill(t, 4, plain, bound, plain, plain) },
			func(r *Ring) { _ = r.Resize(3) }, 1},
		{"shrink evicts one of two bound records", func(t *testing.T) Ring { return fill(t, 4, bound, plain, bound, plain) },
			func(r *Ring) { _ = r.Resize(3) }, 1},
		{"shrink of a wrapped ring evicts the bound record", func(t *testing.T) Ring {
			return fill(t, 3, plain, plain, plain, bound, plain) // start is 2; bound is second oldest
		}, func(r *Ring) { _ = r.Resize(1) }, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := tc.build(t)
			tc.act(&r)
			if r.s.atBound != tc.want || r.AtBound() != (tc.want > 0) {
				t.Errorf("at-bound count = %d, AtBound %v, want %d", r.s.atBound, r.AtBound(), tc.want)
			}
			checkRebuild(t, tc.name, &r)
		})
	}
}

// TestNewRingAndFromRecords_RefuseBadBounds verifies a window of 0 or below, a
// window above maxWindow, a maxWindow of 0 or below, and more records than a
// valid window are each refused by errors.Is ErrRecords with no ring.
//
// Requirement: FR-WEL-034, FR-WEL-010
func TestNewRingAndFromRecords_RefuseBadBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		recs           int
		window, maxWin int64
	}{
		{"window 0", 0, 0, 10},
		{"negative window", 0, -1, 10},
		{"MinInt64 window", 0, math.MinInt64, 10},
		{"window maxWindow+1", 0, 11, 10},
		{"maxWindow 0", 0, 1, 0},
		{"maxWindow negative", 0, 1, -1},
		{"window and maxWindow both 0", 0, 0, 0},
		{"window and maxWindow both negative", 0, -2, -2},
		{"more records than a valid window", 4, 3, 10},
		{"one record, window 1 holds none of two", 2, 1, 10},
		{"more records than window = maxWindow", 11, 10, 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.recs == 0 {
				r, err := NewRing(tc.window, tc.maxWin)
				if !errors.Is(err, ErrRecords) || !reflect.DeepEqual(r, Ring{}) {
					t.Errorf("NewRing(%d, %d) = %+v, %v, want no ring and ErrRecords", tc.window, tc.maxWin, r, err)
				}
			}
			r, err := FromRecords(make([]Record, tc.recs), tc.window, tc.maxWin)
			if !errors.Is(err, ErrRecords) || !reflect.DeepEqual(r, Ring{}) {
				t.Errorf("FromRecords(%d records, %d, %d) = %+v, %v, want no ring and ErrRecords", tc.recs, tc.window, tc.maxWin, r, err)
			}
		})
	}
}

// TestRing_RatioOfAPartlyFilledRing verifies the padding of the ticks the ring
// lacks, from the window, and how a Resize moves it. Ring of window 10 holding
// (income 100, cost 50) and (60, 30): sums 160 and 80, Len 2; upkeep 5,
// DefaultRatio 15 000.
//
// Requirement: FR-WEL-010, FR-WEL-042
func TestRing_RatioOfAPartlyFilledRing(t *testing.T) {
	t.Parallel()
	r, _ := NewRing(10, 20)
	r.Push(Record{Income: 100, Cost: 50})
	r.Push(Record{Income: 60, Cost: 30})
	tests := []struct {
		name   string
		window int64
		want   int64
	}{
		// n 8: pad_cost 40, pad_income 40*15000/10000 = 60; (160+60)*10000/(80+40) = 2200000/120 = 18333.
		{"window 10", 10, 18_333},
		// n 12: pad_cost 60, pad_income 90; 250*10000/140 = 17857.
		{"resize up to 14", 14, 17_857},
		// n 2: pad_cost 10, pad_income 15; 175*10000/90 = 19444.
		{"resize down to 4", 4, 19_444},
		// n 0: no padding; 160*10000/80 = 20000.
		{"resize down to 2", 2, 20_000},
		// the oldest leaves: sums 60 and 30, n 0; 60*10000/30 = 20000.
		{"resize down to 1", 1, 20_000},
	}
	for _, tc := range tests { // sequential: each step resizes the same ring
		if err := r.Resize(tc.window); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got, ok := r.Ratio(5, 15_000); !ok || got != tc.want {
			t.Errorf("%s: Ratio(5, 15000) = %d, %v, want %d, true", tc.name, got, ok, tc.want)
		}
	}
}

// TestRing_NegativeWageDenominatorIsAbsent verifies own-use household-ticks
// above the working household-ticks give a negative denominator, which Wage
// reports as absent while WageParts still returns the raw pair.
//
// Requirement: FR-WEL-018
func TestRing_NegativeWageDenominatorIsAbsent(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		workHH, ownH int64
		wantDen      int64
	}{
		{"own above work", 1, 3, -2}, // 1 - 3
		{"own equals work", 2, 2, 0}, // 2 - 2
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r, _ := NewRing(5, 10)
			r.Push(Record{Income: 100, Working: true, WorkHH: tc.workHH, OwnHH: tc.ownH})
			if num, den := r.WageParts(); num != 100 || den != tc.wantDen {
				t.Errorf("WageParts = %d, %d, want 100, %d", num, den, tc.wantDen)
			}
			if w, ok := r.Wage(); ok || w != 0 {
				t.Errorf("Wage = %d, %v, want 0, false", w, ok)
			}
		})
	}
}

// TestRing_GuardNoRegionOrRole is a Guard: the record and the ring have no
// region or role field, so nothing in them can be reset by a move.
//
// Requirement: FR-WEL-006
func TestRing_GuardNoRegionOrRole(t *testing.T) {
	t.Parallel()
	for _, typ := range []reflect.Type{reflect.TypeOf(Record{}), reflect.TypeOf(Ring{}), reflect.TypeOf(sums{})} {
		for i := 0; i < typ.NumField(); i++ {
			n := strings.ToLower(typ.Field(i).Name)
			if strings.Contains(n, "region") || strings.Contains(n, "role") {
				t.Errorf("%s has field %s", typ, typ.Field(i).Name)
			}
		}
	}
}

// TestRing_CostOfARatioIsConstantInTheWindow verifies the ratio read takes only
// the running sums, Len and the window: two rings of T=30 and T=3650, neither
// full, with equal sums and equal Len differ only through n = T - Len, and read
// what the formula gives; and the read has no slice or map parameter.
// Both hold 10 records of (income 1000, cost 100): sums 10 000 and 1 000;
// upkeep 5, DefaultRatio 15 000.
//
// Requirement: NFR-WEL-001
func TestRing_CostOfARatioIsConstantInTheWindow(t *testing.T) {
	t.Parallel()
	mk := func(window int64) Ring {
		r, _ := NewRing(window, 3650)
		for i := 0; i < 10; i++ {
			r.Push(Record{Income: 1000, Cost: 100})
		}
		return r
	}
	a, b := mk(30), mk(3650)
	if a.s != b.s || a.Len() != 10 || b.Len() != 10 {
		t.Fatalf("setup: sums %+v vs %+v, Len %d vs %d", a.s, b.s, a.Len(), b.Len())
	}
	tests := []struct {
		name string
		r    *Ring
		want int64
	}{
		// n 20: pad_cost 100, pad_income 100*15000/10000 = 150; (10000+150)*10000/(1000+100) = 101500000/1100 = 92272.
		{"T=30", &a, 92_272},
		// n 3640: pad_cost 18200, pad_income 27300; 37300*10000/19200 = 373000000/19200 = 19427.
		{"T=3650", &b, 19_427},
	}
	for _, tc := range tests {
		if got, ok := tc.r.Ratio(5, 15_000); !ok || got != tc.want {
			t.Errorf("%s: Ratio(5, 15000) = %d, %v, want %d, true", tc.name, got, ok, tc.want)
		}
	}
	m, _ := reflect.TypeOf(&a).MethodByName("Ratio")
	for i := 0; i < m.Type.NumIn(); i++ {
		if k := m.Type.In(i).Kind(); k == reflect.Slice || k == reflect.Map {
			t.Errorf("Ratio parameter %d is a %v", i, k)
		}
	}
}

// TestRing_HotPathAllocatesNothing verifies Push, the ratio, the wage and the
// work share allocate nothing.
//
// Requirement: NFR-WEL-001
func TestRing_HotPathAllocatesNothing(t *testing.T) {
	// No t.Parallel: AllocsPerRun must not overlap other tests' allocations.
	r, _ := NewRing(30, 3650)
	rc := rec(100, 50, true)
	if n := testing.AllocsPerRun(200, func() {
		r.Push(rc)
		r.Ratio(10, 15_000)
		r.EarnedRatio(10, 15_000)
		r.Wage()
		r.WorkingTicks()
		r.AtBound()
		WorkShare(15_300, 15_000, 500)
	}); n != 0 {
		t.Errorf("Push and the reads allocated %v times per run, want 0", n)
	}
}

// FuzzRing decodes bytes into pushes, resizes and rebuilds, with field values
// drawn from the extremes as well, and asserts the sum invariant, the refusals
// and that nothing panics.
//
// Requirement: NFR-WEL-002, FR-WEL-034
func FuzzRing(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 1, 9, 2, 0, 5, 3, 4, 4, 7})
	f.Add([]byte{0, 255, 255, 255, 1, 0, 2, 200, 0, 254, 3, 2})
	f.Fuzz(func(t *testing.T, data []byte) {
		const maxW = 16
		c := int64(math.MaxInt64 / maxW)
		vals := []int64{0, 1, -1, 7, c, -c, c + 1, -c - 1, math.MaxInt64, math.MinInt64}
		r, err := NewRing(4, maxW)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i+1 < len(data); i += 2 {
			op, arg := data[i]%4, int(data[i+1])
			switch op {
			case 0, 1:
				v := func(k int) int64 { return vals[(arg+k*3)%len(vals)] }
				r.Push(Record{Income: v(0), Cost: v(1), Relief: v(2), Unearned: v(3), OwnReceipt: v(4),
					Working: arg%2 == 0, WorkHH: v(5), OwnHH: v(6)})
			case 2:
				t2 := int64(arg%20) - 1 // -1..18: both sides of the bounds
				was := stateOf(&r)
				err := r.Resize(t2)
				if (t2 < 1 || t2 > maxW) != (err != nil) || (err != nil && !errors.Is(err, ErrRecords)) {
					t.Fatalf("Resize(%d) error = %v", t2, err)
				}
				if got := stateOf(&r); err != nil && !reflect.DeepEqual(got, was) {
					t.Fatalf("refused Resize(%d) changed the ring: %+v, was %+v", t2, got, was)
				}
			case 3:
				recs := r.AppendRecords(nil)
				if arg%3 == 0 && len(recs) > 0 {
					recs[0].Income = vals[arg%len(vals)]
				}
				w := int64(arg%20) - 1
				b, err := FromRecords(recs, w, maxW)
				// The oracle is written here from c = MaxInt64 / maxW, not from outside().
				beyond := func(v int64) bool { return v > c || v < -c }
				bad := w < 1 || w > maxW || int64(len(recs)) > w
				for _, x := range recs {
					for _, v := range []int64{x.Income, x.Cost, x.Relief, x.Unearned, x.OwnReceipt, x.WorkHH, x.OwnHH} {
						bad = bad || beyond(v)
					}
				}
				if bad != (err != nil) || (err != nil && !errors.Is(err, ErrRecords)) {
					t.Fatalf("FromRecords(%d records, window %d) error = %v, want refusal %v", len(recs), w, err, bad)
				}
				if err != nil && !reflect.DeepEqual(b, Ring{}) {
					t.Fatalf("a refused FromRecords returned a ring: %+v", b)
				}
				if err == nil {
					r = b
				}
			}
			checkRebuild(t, "fuzz", &r)
			r.Ratio(5, 15_000)
			r.Wage()
		}
	})
}

// TestRing_ResizeKeepsTheNewestInOrderAtEveryStart verifies that a shrink, a
// no-op and a grow keep the newest records in order whatever the start index
// of the wrapped ring: with a window of 4 and 4+s pushes the start index is s
// (0 to 3), including the rotation by exactly one position.
//
// Guard: the rotation that straightens a wrapped ring is three reversals
// of which a start of 1 is the shortest; a shortcut for that start that skipped
// the rotation would reorder the records and the running sums would still
// match, so the ordered read is compared with the pushed incomes.
//
// Requirement: FR-WEL-034, FR-WEL-010
func TestRing_ResizeKeepsTheNewestInOrderAtEveryStart(t *testing.T) {
	t.Parallel()
	for start := range 4 {
		for _, to := range []int64{2, 4, 6} {
			t.Run(fmt.Sprintf("start %d resize %d", start, to), func(t *testing.T) {
				t.Parallel()
				r, err := NewRing(4, 10)
				if err != nil {
					t.Fatal(err)
				}
				pushed := 4 + start
				for i := 1; i <= pushed; i++ {
					r.Push(rec(int64(i)*100, int64(i)*10, i%2 == 0))
				}
				if r.start != start {
					t.Fatalf("start index = %d after %d pushes into 4, want %d", r.start, pushed, start)
				}
				if err := r.Resize(to); err != nil {
					t.Fatalf("Resize(%d) = %v, want nil", to, err)
				}
				kept := min(int(to), 4) // the ring held 4, the newest min(to, 4) stay
				got := r.AppendRecords(nil)
				if len(got) != kept {
					t.Fatalf("Resize(%d) kept %d records, want %d", to, len(got), kept)
				}
				for i, g := range got {
					want := int64(pushed-kept+i+1) * 100 // incomes are 100 * the push number
					if g.Income != want {
						t.Errorf("Resize(%d) record %d has income %d, want %d", to, i, g.Income, want)
					}
				}
				checkRebuild(t, "after resize", &r)
			})
		}
	}
}

// TestRing_ZeroValuePushIsANoOp verifies that Push on the zero Ring (no
// window) reports no clamp and stores nothing, even for a record far beyond the
// bound, and does not panic.
//
// Guard: Push is documented never to panic and a zero Ring has a window of 0,
// where the store would divide by zero; the early return is the only thing
// between a restored-before-sized ring and a panic.
//
// Requirement: FR-WEL-042
func TestRing_ZeroValuePushIsANoOp(t *testing.T) {
	t.Parallel()
	var r Ring
	if r.Push(Record{Income: math.MaxInt64, Cost: math.MinInt64}) {
		t.Error("Push on a zero Ring reported a clamp, want false")
	}
	if r.Len() != 0 || r.Window() != 0 || r.AtBound() {
		t.Errorf("zero Ring after Push: Len %d, Window %d, AtBound %v, want 0, 0, false", r.Len(), r.Window(), r.AtBound())
	}
}
