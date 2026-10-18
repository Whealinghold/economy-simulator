package household_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// params are the defaults of AGT 12 that the worked examples use.
var params = household.Params{
	CoverTicks: 3, InputJobs: 1, OutputCoverTicks: 1, FavourFloor: 2000,
	ReserveTicks: 3, SpendMaxShare: 5000, SpendHalfTicks: 10, ProduceHaltRatio: 5000,
}

// TestComputeTargets_Examples verifies the reserve, use target, target and keep
// of AGT 16.1, a tool, a want and the output target.
//
// Requirement: FR-AGT-038, FR-AGT-046
func TestComputeTargets_Examples(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		need household.Need
		want household.Targets
	}{
		// staple: U 3, use 3 * 3 = 9, keep 3 + 9 = 12.
		{"staple", household.Need{Upkeep: 3}, household.Targets{Reserve: 3, UseTgt: 9, Target: 9, Keep: 12}},
		// fuel: U 1, use 1 * 3 = 3, keep 1 + 3 = 4.
		{"fuel", household.Need{Upkeep: 1}, household.Targets{Reserve: 1, UseTgt: 3, Target: 3, Keep: 4}},
		// ore: no upkeep, Job 1 * 1 * 2 = 2 so use 2, keep 2.
		{"ore input", household.Need{Job: 2}, household.Targets{UseTgt: 2, Target: 2, Keep: 2}},
		// item: no use, OutTgt = ceil(1 * 1) = 1 is the target and nothing is kept.
		{"item output", household.Need{OutTgt: 1}, household.Targets{Target: 1}},
		// A commodity that is an input (InputJobs 1 * k 2 * q 3 = 6) and a tool (k 2): Job 8.
		{"input and tool", household.Need{Job: 8}, household.Targets{UseTgt: 8, Target: 8, Keep: 8}},
		// Use wins over the output target when both exist: use 9, not OutTgt 5.
		{"use beats output", household.Need{Upkeep: 3, OutTgt: 5}, household.Targets{Reserve: 3, UseTgt: 9, Target: 9, Keep: 12}},
		// A want alone makes the target: use 0, want 4.
		{"want alone", household.Need{Want: 4, OutTgt: 5}, household.Targets{Target: 4, Keep: 4}},
		// Want on top of use: use 9 + 4 = 13, keep 3 + 9 + 4 = 16.
		{"use and want", household.Need{Upkeep: 3, Want: 4}, household.Targets{Reserve: 3, UseTgt: 9, Target: 13, Keep: 16}},
		{"negative fields count as 0", household.Need{Upkeep: -3, Job: -1, OutTgt: -2, Want: -1}, household.Targets{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.ComputeTargets(tt.need, params)
			if !ok || got != tt.want {
				t.Errorf("ComputeTargets(%+v) = %+v, %v, want %+v, true", tt.need, got, ok, tt.want)
			}
		})
	}
}

// TestComputeTargets_Overflow verifies that each checked step reports ok false.
//
// Requirement: NFR-AGT-004
func TestComputeTargets_Overflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		need household.Need
	}{
		{"upkeep times cover", household.Need{Upkeep: math.MaxInt64}},
		{"use plus job", household.Need{Job: math.MaxInt64, Upkeep: 1}},
		{"use plus want", household.Need{Upkeep: 1, Want: math.MaxInt64}},
		{"keep", household.Need{Upkeep: math.MaxInt64 / 3, Want: math.MaxInt64 - 3*(math.MaxInt64/3)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := household.ComputeTargets(tt.need, params); ok || got != (household.Targets{}) {
				t.Errorf("ComputeTargets(%+v) = %+v, %v, want zero, false", tt.need, got, ok)
			}
		})
	}
}

// TestComputeTargets_LinearInTheWeight verifies that the targets of a cohort of
// weight w are w times those of one household when the totals scale with w,
// given an already-formed output target.
//
// Requirement: FR-AGT-052, FR-AGT-046
func TestComputeTargets_LinearInTheWeight(t *testing.T) {
	t.Parallel()
	for _, w := range []int64{1, 2, 7, 40, 1000} {
		for _, n := range []household.Need{{Upkeep: 3}, {Upkeep: 1, Job: 2}, {Job: 5, Want: 1}, {OutTgt: 4}} {
			one, ok1 := household.ComputeTargets(n, params)
			scaled := household.Need{Upkeep: w * n.Upkeep, Job: w * n.Job, OutTgt: w * n.OutTgt, Want: w * n.Want}
			many, ok2 := household.ComputeTargets(scaled, params)
			want := household.Targets{Reserve: w * one.Reserve, UseTgt: w * one.UseTgt, Target: w * one.Target, Keep: w * one.Keep}
			if !ok1 || !ok2 || many != want {
				t.Errorf("weight %d, need %+v: ComputeTargets = %+v, %v, want %+v", w, n, many, ok2, want)
			}
		}
	}
}

// TestTiers_Examples verifies the gap, surplus and tiers of AGT 16.1.
//
// Requirement: FR-AGT-040
func TestTiers_Examples(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                              string
		keep, stock, reserve, jobwant     int64
		gap, surplus, tier1, tier2, tier3 int64
	}{
		// staple: keep 12, stock 4: gap 8; T1 = max(0, 3 - 4) = 0; T2 = 0; T3 = 8.
		{"staple", 12, 4, 3, 0, 8, 0, 0, 0, 8},
		// fuel: keep 4, stock 0: gap 4; T1 = 1; T2 = 0; T3 = 3.
		{"fuel", 4, 0, 1, 0, 4, 0, 1, 0, 3},
		// ore: keep 2, stock 0, jobwant 2: gap 2; T1 0; left 0, T2 = min(2, 2) = 2; T3 0.
		{"ore", 2, 0, 0, 2, 2, 0, 0, 2, 0},
		// item: keep 0, stock 1: surplus 1, no bid.
		{"item", 0, 1, 0, 0, 0, 1, 0, 0, 0},
		{"stock equals keep", 5, 5, 2, 3, 0, 0, 0, 0, 0},
		{"stock above keep", 5, 9, 2, 3, 0, 4, 0, 0, 0},
		// Stock 4 holds the reserve 3 and leaves 1 of the job want 3: T2 = min(6, 2) = 2.
		{"stock covers the reserve, job needs more", 10, 4, 3, 3, 6, 0, 0, 2, 4},
		// Stock 1 is below the reserve 3: T1 = 2, left 0, T2 = min(4, 3) = 3, T3 = 1.
		{"reserve above stock", 6, 1, 3, 3, 5, 0, 2, 3, 0},
		// The job want is met by the stock above the reserve: left 5, T2 = 0.
		{"job met by the stock", 20, 8, 3, 5, 12, 0, 0, 0, 12},
		{"negative arguments count as 0", -4, -1, -2, -3, 0, 0, 0, 0, 0},
		// keep below reserve: T1 is cut to the gap so the tiers still add up.
		{"keep below reserve", 1, 0, 3, 0, 1, 0, 1, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gap, surplus, t1, t2, t3 := household.Tiers(tt.keep, tt.stock, tt.reserve, tt.jobwant)
			if gap != tt.gap || surplus != tt.surplus || t1 != tt.tier1 || t2 != tt.tier2 || t3 != tt.tier3 {
				t.Errorf("Tiers(%d, %d, %d, %d) = %d, %d, %d, %d, %d, want %d, %d, %d, %d, %d",
					tt.keep, tt.stock, tt.reserve, tt.jobwant, gap, surplus, t1, t2, t3,
					tt.gap, tt.surplus, tt.tier1, tt.tier2, tt.tier3)
			}
		})
	}
}

// TestTiers_AddToTheGap verifies over a grid, including keep below reserve and
// negative arguments, that the three tiers are not negative and add to the gap,
// and that an ask never accompanies a bid.
//
// Requirement: FR-AGT-040
func TestTiers_AddToTheGap(t *testing.T) {
	t.Parallel()
	vals := []int64{-1, 0, 1, 2, 3, 5, 8, 13}
	for _, keep := range vals {
		for _, stock := range vals {
			for _, reserve := range vals {
				for _, job := range vals {
					gap, surplus, t1, t2, t3 := household.Tiers(keep, stock, reserve, job)
					if t1 < 0 || t2 < 0 || t3 < 0 || t1+t2+t3 != gap || (gap > 0 && surplus > 0) {
						t.Fatalf("Tiers(%d, %d, %d, %d) = %d, %d, %d, %d, %d: tiers must be non-negative and add to the gap",
							keep, stock, reserve, job, gap, surplus, t1, t2, t3)
					}
				}
			}
		}
	}
}

// TestTiers_LinearInTheWeight verifies that the wants of a cohort are w times
// those of a household when keep, stock, reserve and jobwant are, the case
// FR-AGT-052 allows (ample money is not needed: no money enters the tiers).
//
// Requirement: FR-AGT-052, FR-AGT-046, FR-AGT-069
func TestTiers_LinearInTheWeight(t *testing.T) {
	t.Parallel()
	vals := []int64{0, 1, 2, 3, 5, 8}
	for _, w := range []int64{2, 7, 40} {
		for _, keep := range vals {
			for _, stock := range vals {
				for _, reserve := range vals {
					for _, job := range vals {
						g1, s1, a1, b1, c1 := household.Tiers(keep, stock, reserve, job)
						g, s, a, b, c := household.Tiers(w*keep, w*stock, w*reserve, w*job)
						if g != w*g1 || s != w*s1 || a != w*a1 || b != w*b1 || c != w*c1 {
							t.Fatalf("weight %d, Tiers(%d, %d, %d, %d): got %d,%d,%d,%d,%d, want %d times %d,%d,%d,%d,%d",
								w, keep, stock, reserve, job, g, s, a, b, c, w, g1, s1, a1, b1, c1)
						}
					}
				}
			}
		}
	}
}

// TestTargetsAndTiers_AllocateNothing verifies that the target and tier
// functions allocate nothing.
//
// Requirement: NFR-AGT-001
func TestTargetsAndTiers_AllocateNothing(t *testing.T) {
	var sink int64
	allocs := testing.AllocsPerRun(100, func() {
		tg, _ := household.ComputeTargets(household.Need{Upkeep: 3, Job: 2}, params)
		g, s, a, b, c := household.Tiers(tg.Keep, 4, tg.Reserve, 2)
		sink += g + s + a + b + c
	})
	if allocs != 0 {
		t.Errorf("target functions allocated %v times per run, want 0 (sink %d)", allocs, sink)
	}
}

func BenchmarkComputeTargets(b *testing.B) {
	b.ReportAllocs()
	n := household.Need{Upkeep: 3, Job: 2, OutTgt: 1}
	for b.Loop() {
		household.ComputeTargets(n, params)
	}
}

func BenchmarkTiers(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		household.Tiers(12, 4, 3, 2)
	}
}

func BenchmarkFed(b *testing.B) {
	b.ReportAllocs()
	consumed, per := []int64{100, 40}, []int64{3, 1}
	for b.Loop() {
		household.Fed(consumed, per, 40)
	}
}
