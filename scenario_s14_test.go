package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

const (
	s14Ticks     int64 = 1800
	s14ShockFrom int64 = 424
	s14ShockTo   int64 = 516
	s14Sample    int64 = 12 // households are read every this many ticks
)

// s14Depths are the depths m of the harvest failure, in basis points of the
// staple yield, one script each (scripts/s14-<m>.toml).
var s14Depths = []int64{7000, 5000, 2500, 0}

// s14Obs keeps what S14 reports besides the streaming observer: the relief paid
// and the households removed by cause.
type s14Obs struct {
	relief  int64
	removed map[economysim.Cause]int64
}

// Observe sums the relief and the weight of each removal by cause.
func (o *s14Obs) Observe(e economysim.Event) {
	switch e := e.(type) {
	case economysim.ReliefPaid:
		o.relief += int64(e.Paid)
	case economysim.AgentBankrupted:
		o.removed[e.Cause] += e.Weight
	}
}

// TestScenarioS14_RegionLosesItsIncome runs scenario S14 of spec-reference-chain.md
// 7.2 on pack famine (cold layer set, the estates module on): 1 800 ticks, four
// depths of a harvest failure of grow_staple from tick 424 to 515, seeds 1 to 10
// (the first under -race), with the audit on. Exact: no published price leaves its
// belief interval except an estate sale (FR-STB-008), and the audit (goods and
// money conserved, no balance or stock negative, the pool never overdrawn) holds
// at every tick. Reported: the households as a share of the start (basis points)
// at ticks 423, 515, 695, 1 007 and 1 799 and the lowest share sampled every 12
// ticks, the unfed household-ticks and the longest run of ticks with any unfed,
// the relief paid and the pool at the end, the households removed by cause and,
// when s14.recover_band_bp has a value, the tick of recovery. Asserted only when
// its key has a value (bands/famine.toml): s14.recover_ticks and s14.fall_bp. The
// outcomes are written to testdata/chains/expected/famine.scenarios.txt with
// -update. The ladder-order check (FR-SPN-013 to 015) is not asserted: s14.fall_bp
// is unset and no event tells a rung from another (plan 2-7 O6).
//
// Requirement: FR-EST-014, FR-EST-015, FR-EST-016, FR-EST-017, FR-EST-018, FR-EST-019, FR-EST-020, FR-EST-022,
// FR-STB-005, FR-STB-008, FR-AGT-013, FR-AGT-017, FR-SPN-012, FR-SPN-016, FR-SPN-024, FR-SPN-026,
// FR-REF-020, FR-REF-024, FR-REF-026, FR-REF-027
//
// Guard: a region that loses its income may starve, and it must do so with the
// books balanced and the prices inside their bounds.
func TestScenarioS14_RegionLosesItsIncome(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packLayers["famine"]...)
	for _, m := range s14Depths {
		script := loadScript(t, fmt.Sprintf("s14-%d.toml", m))
		rep := newPackScenReport("famine", fmt.Sprintf("S14 depth %d seeds 1-%d, %d ticks", m, scenarioSeedCount(), s14Ticks))
		t.Run(fmt.Sprintf("depth %d", m), func(t *testing.T) {
			for _, seed := range seedList(scenarioSeedCount()) {
				t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
					t.Parallel()
					s14Seed(t, cfg, script, seed, rep)
				})
			}
		})
		t.Cleanup(func() { rep.flush(t) })
	}
}

// s14Seed runs one seed of one depth of S14.
func s14Seed(t *testing.T, cfg economysim.Config, script []shockStep, seed uint64, rep *scenReport) {
	const scenario, pack = "S14", "famine"
	extra := &s14Obs{removed: map[economysim.Cause]int64{}}
	shares := map[int64]int64{}
	var start, lowest int64 = 0, 10_000
	var series []int64 // the share at ticks s14ShockTo-1+k*s14Sample, from the shock's end
	run := runScenario(t, cfg, scenarioSpec{
		Seed: seed, Ticks: s14Ticks, Script: script,
		Options: []economysim.Option{economysim.WithAudit(), economysim.WithObserver(extra)},
		Snap: func(tick int64) bool {
			return tick%s14Sample == s14Sample-1 || tick == s14ShockFrom-1 || tick == s14ShockTo-1
		},
		OnSnap: func(tick int64, s economysim.Snapshot) {
			if start == 0 {
				start = s.Households // the first sample is at tick 11, before any shock
			}
			share, _ := shareBP(s.Households, start)
			shares[tick] = share
			lowest = min(lowest, share)
			if tick >= s14ShockTo-1 {
				series = append(series, share)
			}
		},
	})
	o := run.Obs
	checkExact(t, rep, scenario, pack, seed, "market series in tick order", o.badSeq == "", "%s", o.badSeq)
	br := boundsViolations(cfg, run.Names, o, 3)
	checkExact(t, rep, scenario, pack, seed, "price within belief bounds (FR-STB-008)", len(br.Violations) == 0, "measure price bounds: %v", br.Violations)
	for _, tick := range []int64{s14ShockFrom - 1, s14ShockTo - 1, 695, 1007, s14Ticks - 1} {
		if v, ok := shares[tick]; ok {
			rep.num(seed, fmt.Sprintf("households share bp at tick %d", tick), v)
		} else {
			rep.absent(seed, fmt.Sprintf("households share bp at tick %d", tick))
		}
	}
	rep.num(seed, "lowest households share bp sampled", lowest)
	var unfed int64
	flags := make([]bool, len(o.unfed))
	for i, u := range o.unfed {
		unfed += u
		flags[i] = u > 0
	}
	rep.num(seed, "unfed household-ticks", unfed)
	rep.num(seed, "longest run of ticks with any unfed", longestRun(flags))
	rep.num(seed, "relief paid", extra.relief)
	for _, c := range []economysim.Cause{economysim.CauseInsolvent, economysim.CauseStarved} {
		rep.num(seed, "households removed, "+string(c), extra.removed[c])
	}
	rep.num(seed, "pool at the end", int64(poolOf(run.Final)))
	rep.num(seed, "households at the end", run.Final.Households)
	if band, set := bandIn(t, "testdata/chains/bands/famine.toml", "s14.recover_band_bp"); set {
		recovered := int64(-1)
		for k, v := range series {
			if v >= shares[s14ShockFrom-1]-band {
				recovered = int64(k) * s14Sample
				break
			}
		}
		if recovered >= 0 {
			rep.num(seed, "ticks from the shock's end to recovery", recovered)
		} else {
			rep.absent(seed, "ticks from the shock's end to recovery")
		}
		if bound, set := bandIn(t, "testdata/chains/bands/famine.toml", "s14.recover_ticks"); set && recovered > bound {
			rep.check(seed, "band s14.recover_ticks", false, fmt.Sprintf("measure %d against %d", recovered, bound))
			t.Errorf("S14 seed %d: recovered after %d ticks, band s14.recover_ticks is %d", seed, recovered, bound)
		}
	}
	if bound, set := bandIn(t, "testdata/chains/bands/famine.toml", "s14.fall_bp"); set && 10_000-lowest > bound {
		rep.check(seed, "band s14.fall_bp", false, fmt.Sprintf("fall %d against %d", 10_000-lowest, bound))
		t.Errorf("S14 seed %d: households fell by %d bp, band s14.fall_bp is %d", seed, 10_000-lowest, bound)
	}
}

// poolOf sums the pool of every region of a snapshot.
func poolOf(s economysim.Snapshot) economysim.Money {
	var m economysim.Money
	for _, r := range s.Regions {
		m += r.MoneyPool
	}
	return m
}
