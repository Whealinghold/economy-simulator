package production_test

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/production"
)

const maxYieldBP = 100_000 // MaxYieldBP, passed by the test only

// The Earth default season indexes (spec-calendar.md 7.4).
const (
	spring = iota
	summer
	autumn
	winter
)

// TestSite_Rules verifies each rule of PRO 6.2 in order, and the two climates
// of PRO 12.1 (the expected values come from the spec's example, not from the
// code).
//
// Requirement: FR-PRO-020
func TestSite_Rules(t *testing.T) {
	t.Parallel()
	north := &production.Profile{Seasons: []production.SeasonBP{{spring, 10000}}}
	south := &production.Profile{Seasons: []production.SeasonBP{{spring, 12000}, {autumn, 7000}}}
	tests := []struct {
		name    string
		profile *production.Profile
		sited   bool
		season  int
		want    int64
	}{
		{"season entry wins over All", &production.Profile{HasAll: true, All: 5000, Seasons: []production.SeasonBP{{summer, 8000}}}, true, summer, 8000},
		{"All when no season entry", &production.Profile{HasAll: true, All: 5000, Seasons: []production.SeasonBP{{summer, 8000}}}, true, winter, 5000},
		{"seasonal profile without the season is 0", &production.Profile{Seasons: []production.SeasonBP{{spring, 9000}}}, false, summer, 0},
		{"no profile, not sited", nil, false, spring, 10000},
		{"no profile, sited", nil, true, spring, 0},
		{"season -1 with All", &production.Profile{HasAll: true, All: 6000}, true, -1, 6000},
		{"season -1 never matches an entry", &production.Profile{Seasons: []production.SeasonBP{{0, 9000}}}, false, -1, 0},
		{"only All = 0", &production.Profile{HasAll: true, All: 0}, false, spring, 0},
		{"empty profile reads as absent, not sited", &production.Profile{}, false, spring, 10000},
		{"empty profile reads as absent, sited", &production.Profile{}, true, spring, 0},
		{"north spring", north, true, spring, 10000},
		{"north summer", north, true, summer, 0},
		{"south spring", south, true, spring, 12000},
		{"south summer", south, true, summer, 0},
		{"south autumn", south, true, autumn, 7000},
		{"south winter", south, true, winter, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := production.Site(tt.profile, tt.sited, tt.season); got != tt.want {
				t.Errorf("Site(%+v, sited %v, season %d) = %d, want %d", tt.profile, tt.sited, tt.season, got, tt.want)
			}
		})
	}
}

// TestEffective_Chain verifies the order and the truncation of each step.
//
// Requirement: FR-PRO-022, FR-PRO-034
func TestEffective_Chain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                 string
		site, shock, tech, trainee, abundant int64
		want                                 int64
		wantOK                               bool
	}{
		// 8 500 * 10 000 / 10 000 = 8 500; * 10 000 / 10 000 = 8 500;
		// 8 500 * 5 000 / 10 000 = 4 250; * 10 000 / 10 000 = 4 250.
		{"trainee halves", 8500, 10000, 10000, 5000, 10000, 4250, true},
		// 10 000 * 3 000 / 10 000 = 3 000 and the rest are neutral.
		{"shock of 3 000", 10000, 3000, 10000, 10000, 10000, 3000, true},
		// 3 * 5 000 / 10 000 = 1 (1.5 truncated); 1 * 10 000 / 10 000 = 1;
		// 1 * 15 000 / 10 000 = 1 (1.5 truncated); the untruncated chain is 2.25.
		{"truncated at each step", 3, 5000, 10000, 15000, 10000, 1, true},
		{"zero site", 0, 10000, 10000, 10000, 10000, 0, true},
		{"zero shock", 8500, 0, 10000, 10000, 10000, 0, true},
		{"zero tech", 8500, 10000, 0, 10000, 10000, 0, true},
		{"zero trainee", 8500, 10000, 10000, 0, 10000, 0, true},
		{"zero abundance", 8500, 10000, 10000, 10000, 0, 0, true},
		// Steps: 1e5 * 1e5 / 1e4 = 1e6; 1e6 * 1e5 / 1e4 = 1e7; 1e8; 1e9.
		{"largest validated", maxYieldBP, maxYieldBP, maxYieldBP, maxYieldBP, maxYieldBP, 1_000_000_000, true},
		// MaxInt64 * 20 000 / 10 000 = 2 * MaxInt64, which does not fit.
		{"overflow", math.MaxInt64, 20000, 10000, 10000, 10000, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := production.Effective(tt.site, tt.shock, tt.tech, tt.trainee, tt.abundant)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Effective(%d, %d, %d, %d, %d) = %d, %v, want %d, %v",
					tt.site, tt.shock, tt.tech, tt.trainee, tt.abundant, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestOutput_Cases verifies the worked example of PRO 6.6 row by row, the
// neutral factor and the boundaries.
//
// Requirement: FR-PRO-024
func TestOutput_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		k, q, e, residue  int64
		wantN, wantResidu int64
		wantOK            bool
	}{
		// 3 * 1 * 8 500 + 2 000 = 27 500: 2 units, residue 7 500.
		{"example 1", 3, 1, 8500, 2000, 2, 7500, true},
		// 25 500 + 7 500 = 33 000: 3 units, residue 3 000.
		{"example 2", 3, 1, 8500, 7500, 3, 3000, true},
		// 25 500 + 3 000 = 28 500: 2 units, residue 8 500.
		{"example 3", 3, 1, 8500, 3000, 2, 8500, true},
		// 25 500 + 8 500 = 34 000: 3 units, residue 4 000.
		{"example 4", 3, 1, 8500, 8500, 3, 4000, true},
		// E = 10 000: k * q = 12 units, residue unchanged (6.6.1).
		{"neutral factor", 4, 3, 10000, 1234, 12, 1234, true},
		// 1 * 1 * 1 + 0 = 1: 0 units, residue 1.
		{"k 1 small", 1, 1, 1, 0, 0, 1, true},
		// 3 * 1 * 3 333 = 9 999 + 9 999 = 19 998: 1 unit, residue 9 998.
		{"residue 9 999 carries", 3, 1, 3333, 9999, 1, 9998, true},
		// 2 * 5 * 10 000 = 100 000 is a multiple of 10 000: 10 units, residue
		// 9 999 stays 9 999 (the remainder part is 0, no carry).
		{"multiple with residue 9 999", 2, 5, 10000, 9999, 10, 9999, true},
		// 1e6 * 100 * 1e5 / 1e4 = 1e9 = MaxQuantity exactly.
		{"largest that fits MaxQuantity", 1_000_000, 100, maxYieldBP, 0, 1_000_000_000, 0, true},
		// One above: 1e6 * 101 * 1e5 / 1e4 = 1.01e9 is still an int64 here (the
		// MaxQuantity refusal is validation's, PRO 10.3).
		{"one above MaxQuantity still computes", 1_000_000, 101, maxYieldBP, 0, 1_010_000_000, 0, true},
		// MaxInt64 * 10 000 / 10 000 = MaxInt64.
		{"MaxInt64 fits", 1, math.MaxInt64, 10000, 0, math.MaxInt64, 0, true},
		// MaxInt64 * 20 000 / 10 000 = 2 * MaxInt64.
		{"product overflows", 1, math.MaxInt64, 20000, 0, 0, 0, false},
		// k * q overflows.
		{"k times q overflows", math.MaxInt64, 2, 10000, 0, 0, 0, false},
		// kq * 10 001 = MaxInt64 * 10 000 + 8 249: Q = MaxInt64, R = 8 249.
		// Residue 1 750 gives 9 999, no carry; 1 751 gives 10 000, a carry that
		// does not fit.
		{"quotient is MaxInt64, no carry", 1, 9222449791875588249, 10001, 1750, math.MaxInt64, 9999, true},
		{"carry overflows", 1, 9222449791875588249, 10001, 1751, 0, 0, false},
		{"negative residue", 1, 1, 10000, -1, 0, 0, false},
		{"residue 10 000", 1, 1, 10000, 10000, 0, 0, false},
		{"negative k", -1, 1, 10000, 0, 0, 0, false},
		{"negative q", 1, -1, 10000, 0, 0, 0, false},
		{"negative e", 1, 1, -1, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n, r, ok := production.Output(tt.k, tt.q, tt.e, tt.residue)
			if n != tt.wantN || r != tt.wantResidu || ok != tt.wantOK {
				t.Errorf("Output(%d, %d, %d, %d) = %d, %d, %v, want %d, %d, %v",
					tt.k, tt.q, tt.e, tt.residue, n, r, ok, tt.wantN, tt.wantResidu, tt.wantOK)
			}
		})
	}
}

// TestOutput_LongRun verifies that the carried residue makes the total exact
// to less than a unit, at weight 1 and at weight 1 000.
//
// Requirement: FR-PRO-025
//
// Guard: dropping the residue (a zero start every job) loses up to a unit per
// job, 1 000 jobs at E = 8 500 and k = 1 lose 150 units, far over the bound.
func TestOutput_LongRun(t *testing.T) {
	t.Parallel()
	const jobs = 1000
	for _, k := range []int64{1, 1000} {
		for seed := int64(0); seed < 50; seed++ {
			e := 1 + (seed*1237)%maxYieldBP // a varied E per seed
			start := (seed * 7919) % 10000
			residue := start
			var total int64
			for range jobs {
				n, r, ok := production.Output(k, 3, e, residue)
				if !ok {
					t.Fatalf("seed %d weight %d: Output not ok", seed, k)
				}
				total += n
				residue = r
			}
			// total * 10 000 + residue = jobs * k * 3 * e + start, exactly.
			if got, want := total*10000+residue, int64(jobs)*k*3*e+start; got != want {
				t.Errorf("seed %d weight %d: total*10000+residue = %d, want %d", seed, k, got, want)
			}
			// So total differs from jobs*k*3*e/10 000 by less than one unit.
			diff := int64(jobs)*k*3*e - total*10000
			if diff > 10000 || diff < -10000 {
				t.Errorf("seed %d weight %d: total %d is %d ten-thousandths from exact", seed, k, total, diff)
			}
		}
	}
}

// TestRescale_Cases verifies PRO 5.4: a fall scales the pending quantities by
// the new executions over the old, rounded down; a same weight or a rise
// changes nothing.
//
// Requirement: FR-PRO-019
func TestRescale_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		k, w        int64
		pending     []int64
		wantK       int64
		wantPending []int64
		wantOK      bool
	}{
		// 7 * 4 / 10 = 2 (2.8), 12 * 4 / 10 = 4 (4.8), 1 * 4 / 10 = 0 (0.4).
		{"fall 10 to 4", 10, 4, []int64{7, 12, 1}, 4, []int64{2, 4, 0}, true},
		{"same weight", 10, 10, []int64{7, 12, 1}, 10, []int64{7, 12, 1}, true},
		{"rise", 10, 25, []int64{7, 12, 1}, 10, []int64{7, 12, 1}, true},
		// 5 * 0 / 3 = 0.
		{"fall to 0", 3, 0, []int64{5}, 0, []int64{0}, true},
		{"no pending", 10, 4, nil, 4, nil, true},
		// Largest quantity: 1e9 * 999 999 / 1 000 000 = 999 999 000 (the
		// product 9.99999e14 fits, and the helper would hold it anyway).
		{"large", 1_000_000, 999_999, []int64{1_000_000_000}, 999_999, []int64{999_999_000}, true},
		{"k 0", 0, 0, []int64{1}, 0, []int64{1}, false},
		{"negative weight", 5, -1, []int64{1}, 5, []int64{1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := production.Rescale(tt.k, tt.w, tt.pending)
			if got != tt.wantK || ok != tt.wantOK || !slices.Equal(tt.pending, tt.wantPending) {
				t.Errorf("Rescale(%d, %d) = %d, %v with pending %v, want %d, %v with %v",
					tt.k, tt.w, got, ok, tt.pending, tt.wantK, tt.wantOK, tt.wantPending)
			}
			if ok && got > tt.k {
				t.Errorf("Rescale(%d, %d) = %d exceeds k", tt.k, tt.w, got)
			}
		})
	}
}

// TestProduction_NoAllocs verifies that no function allocates.
//
// Requirement: NFR-PRO-001
func TestProduction_NoAllocs(t *testing.T) {
	prof := &production.Profile{HasAll: true, All: 5000, Seasons: []production.SeasonBP{{0, 1}, {2, 3}}}
	pending := []int64{70, 120, 10}
	var sink int64
	tests := []struct {
		name string
		fn   func()
	}{
		{"Site", func() { sink += production.Site(prof, true, 2) }},
		{"Site nil", func() { sink += production.Site(nil, false, -1) }},
		{"Effective", func() { e, _ := production.Effective(8500, 10000, 10000, 5000, 10000); sink += e }},
		{"Output", func() { n, r, _ := production.Output(3, 1, 8500, 2000); sink += n + r }},
		{"Rescale", func() { n, _ := production.Rescale(10, 4, pending); sink += n }},
	}
	for _, tt := range tests {
		if got := testing.AllocsPerRun(100, tt.fn); got != 0 {
			t.Errorf("%s allocs = %v, want 0", tt.name, got)
		}
	}
	_ = sink
}

func encode(vals ...int64) []byte {
	b := make([]byte, 0, 8*len(vals))
	for _, v := range vals {
		b = binary.LittleEndian.AppendUint64(b, uint64(v))
	}
	return b
}

func word(data []byte, i int) int64 {
	if len(data) < 8*(i+1) {
		return 0
	}
	return int64(binary.LittleEndian.Uint64(data[8*i:]))
}

// FuzzSite decodes a profile and a season from bytes, made valid (entries in
// [0, MaxYieldBP], seasons sorted), and verifies no panic and a factor in
// [0, MaxYieldBP].
//
// Requirement: NFR-PRO-004, FR-PRO-020
func FuzzSite(f *testing.F) {
	// Fields: flags (1 profile, 2 sited, 4 has all), all, season, then season
	// entries as (gap, bp) pairs.
	f.Add(encode(1|2, 0, 0, 0, 10000))
	f.Add(encode(1|2, 0, 2, 0, 12000, 1, 7000))
	f.Add(encode(1|4, 5000, -1))
	f.Add(encode(0, 0, 0))
	f.Add(encode(1, math.MaxInt64, math.MinInt64, math.MaxInt64, math.MinInt64))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		flags := word(data, 0)
		var p production.Profile
		p.HasAll = flags&4 != 0
		p.All = int64(uint64(word(data, 1)) % (maxYieldBP + 1))
		season := int(word(data, 2)%8) - 1
		idx := 0
		for i := 3; i+1 < len(data)/8 && len(p.Seasons) < 8; i += 2 {
			idx += int(uint64(word(data, i)) % 3)
			p.Seasons = append(p.Seasons, production.SeasonBP{Season: idx, BP: int64(uint64(word(data, i+1)) % (maxYieldBP + 1))})
			idx++
		}
		prof := &p
		if flags&1 == 0 {
			prof = nil
		}
		if got := production.Site(prof, flags&2 != 0, season); got < 0 || got > maxYieldBP {
			t.Fatalf("Site(%+v, %v, %d) = %d outside [0, %d]", prof, flags&2 != 0, season, got, maxYieldBP)
		}
	})
}

// FuzzEffective verifies no panic for any operands, a result in [0, 1e9] for
// operands within [0, MaxYieldBP], and a result that never exceeds the
// operands' exact product bound when ok.
//
// Requirement: NFR-PRO-004, FR-PRO-022
func FuzzEffective(f *testing.F) {
	f.Add(encode(8500, 10000, 10000, 5000, 10000))
	f.Add(encode(maxYieldBP, maxYieldBP, maxYieldBP, maxYieldBP, maxYieldBP))
	f.Add(encode(math.MaxInt64, 20000, 10000, 10000, 10000))
	f.Add(encode(math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64, math.MinInt64))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		var w [5]int64
		for i := range w {
			w[i] = word(data, i)
		}
		// Any operands: must not panic.
		production.Effective(w[0], w[1], w[2], w[3], w[4])
		var v [5]int64
		for i := range v {
			v[i] = int64(uint64(w[i]) % (maxYieldBP + 1))
		}
		got, ok := production.Effective(v[0], v[1], v[2], v[3], v[4])
		if !ok || got < 0 || got > 1_000_000_000 {
			t.Fatalf("Effective(%v) = %d, %v, want ok within [0, 1e9]", v, got, ok)
		}
	})
}
