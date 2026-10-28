package economysim_test

import (
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// x4Band is the value of the band key key in bands/second.toml; ok is false when
// the key has no value line (FR-REF-022: the measure is then reported only).
func x4Band(t testing.TB, key string) (v float64, ok bool) {
	t.Helper()
	for _, tb := range plainTOML(t, chainsDir+"/bands/second.toml") {
		if tb.name != "x4."+key {
			continue
		}
		s, has := tb.kv["value"]
		if !has {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatalf("bands/second.toml [x4.%s] value %q: %v", key, s, err)
		}
		return f, true
	}
	t.Fatalf("bands/second.toml has no table [x4.%s]", key)
	return 0, false
}

// x4BandThresholds reads the four keys of the predicate from bands/second.toml,
// each with the Set flag its value line gives.
func x4BandThresholds(t testing.TB) x4Thresholds {
	t.Helper()
	var th x4Thresholds
	th.TradeShareBP.V, th.TradeShareBP.Set = x4Band(t, "trade_share_bp")
	th.PinnedShareBP.V, th.PinnedShareBP.Set = x4Band(t, "pinned_share_bp")
	th.HouseholdsBP.V, th.HouseholdsBP.Set = x4Band(t, "households_bp")
	th.RangeFactor.V, th.RangeFactor.Set = x4Band(t, "range_factor")
	return th
}

// TestConfigOnly_SecondRunsToTheStablePredicate verifies X4, the
// configuration-only chain test: pack second, loaded from testdata/chains/second
// alone, builds a world and runs 1 095 ticks for seeds 1 to 4 with no Go specific
// to the pack, and over the last 365 ticks the stable-state predicate holds. The
// test names no commodity, recipe or role of the pack: every identifier comes from
// the configuration and the snapshot's Names (architecture.md 9.4).
//
// Asserted: (a) the audit passes every tick and the conservation identities hold
// exactly, (c) the mean volume per active market is at least one unit a tick (the
// paper's number, spec-beliefs.md 19) and the Go-built secondConfig has the file's
// hash (FR-CFG-004). Reported, and asserted only once bands/second.toml gives the
// key a value (FR-REF-022): (b) x4.trade_share_bp, (d) x4.pinned_share_bp, (e)
// x4.households_bp, (f) x4.range_factor.
//
// Requirement: FR-REF-031, FR-REF-017, FR-REF-001, FR-CFG-004, FR-PRO-046
//
// Guard: a branch on a commodity, recipe or role identifier anywhere in the
// library would fail here on a setting it never heard of.
func TestConfigOnly_SecondRunsToTheStablePredicate(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test (1 095 ticks, four seeds), skipped with -short")
	}
	cfg := loadValidPack(t, packSecond...)
	if economysim.HashConfig(cfg) != economysim.HashConfig(secondConfig()) {
		t.Fatalf("HashConfig(file) != HashConfig(secondConfig()): the file and the Go build disagree")
	}
	for _, seed := range x4Seeds {
		t.Run("seed"+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			res := runPack(t, cfg, runSpec{Seed: seed, Ticks: x4Ticks, Drive: driveStep, Audit: true, Snapshots: snapEvery})
			crit := stablePredicate(newStableRun(cfg, res), x4BandThresholds(t))
			var traj []int64
			for _, tick := range []int64{0, 50, 100, 200, 365, 730, x4Ticks - 1} {
				traj = append(traj, res.Samples[tick].Households)
			}
			t.Logf("X4 second seed %d: households at ticks 0, 50, 100, 200, 365, 730, %d = %v (start %d)", seed, x4Ticks-1, traj, res.Samples[0].Households)
			for _, c := range crit {
				t.Logf("X4 second seed %d: (%s) %s: %s", seed, c.Key, c.Name, c.Detail)
				if !c.Pass {
					t.Errorf("X4 (%s) second seed %d: %s", c.Key, seed, c.Detail)
				}
			}
		})
	}
}
