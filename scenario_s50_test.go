package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s50 sizes: the edge of 100 at an information speed of 50 gives a lag of 3 for every
// rumour (spec-information.md 12.1); the swing's period is 8, 4 and 2 lags, so the
// lag is an eighth, a quarter and half of the period. The run is long enough for
// whole periods of the longest after a warm-up of one.
const (
	s50Lag    = 3
	s50From   = 24
	s50Ticks  = 72
	s50Seeds  = 3
	s50Centre = 96
	s50Step   = 2
)

// s50Wave is a triangle wave of the given period in ticks, in price units around
// s50Centre: odd harmonics only, so a shift of a quarter period correlates to zero
// and a shift of half a period is the opposite wave (the argument of INF-A5).
func s50Wave(period, tick int64) int64 {
	s := tick % period
	return s50Step * (abs64((4*s+period)%(4*period)-2*period) - period)
}

func abs64(x int64) int64 { return max(x, -x) }

// TestScenarioS50_SeasonalPhase runs scenario S50 (INF-S5, STB-L23, plan 2-13 D478) on
// pack two with the information overlay and no error: two growers of the north quote
// staple at a triangle wave of period 24, 12 and 6 ticks, and the readers of the south
// are shown the rumours of it at a lag of 3, an eighth, a quarter and half of the
// period. It asserts that every rumour is the price the market published a lag ago, and
// records the correlation of the rumoured series with the true one in basis points of
// the true series' variance: it is clearly positive at an eighth, nil at a quarter and
// -10 000 at half a period, where the sign of the gap is reversed. It is a measurement
// of the signal, not of the economy.
//
// Requirement: FR-INF-016, FR-INF-018
//
// Guard: the signal a shipper acts on is shifted by the lag; if a rumour read the
// current price or the wrong vintage the phase would not follow the lag.
func TestScenarioS50_SeasonalPhase(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rep := newPackScenReport("two", "S50 a seasonal swing at the source (INF-S5, a measurement of the signal)")
	for _, period := range []int64{8 * s50Lag, 4 * s50Lag, 2 * s50Lag} {
		name := map[int64]string{8 * s50Lag: "eighth", 4 * s50Lag: "quarter", 2 * s50Lag: "half"}[period]
		for seed := uint64(1); seed <= s50Seeds; seed++ {
			quote := func(tick int64) (economysim.Price, bool) {
				return economysim.Price(s50Centre + s50Wave(period, tick)), true
			}
			r, staple := thinWorld(t, seed, 0, quote)
			r.run(t, s50Ticks)
			ticks, rumours := r.strat.rumoursAbout(r.region["south"], r.region["north"], staple)
			series := map[int64]int64{} // tick -> rumoured price less the centre
			var wrong int64
			for i, ru := range rumours {
				if ticks[i] < s50From {
					continue
				}
				if ru.Lag != s50Lag || int64(ru.Price) != s50Centre+s50Wave(period, ticks[i]-ru.Lag) {
					wrong++
				}
				series[ticks[i]] = int64(ru.Price) - s50Centre
			}
			var cross, variance int64
			for tick := int64(s50From); tick < s50Ticks; tick++ {
				w := s50Wave(period, tick)
				cross += w * series[tick]
				variance += w * w
			}
			corr := cross * 10_000 / variance
			key := "s50." + name
			rep.num(seed, key+".correlation_bp", corr)
			infoCheck(t, rep, seed, key+".rumour_is_the_price_a_lag_ago", wrong == 0 && len(series) == s50Ticks-s50From, fmt.Sprintf("%d rumours off the vintage, %d ticks read", wrong, len(series)))
			switch period {
			case 8 * s50Lag:
				infoCheck(t, rep, seed, key+".phase_follows_lag", corr > 5_000 && corr < 10_000, fmt.Sprintf("correlation %d bp, want between 5000 and 10000", corr))
			case 4 * s50Lag:
				infoCheck(t, rep, seed, key+".phase_follows_lag", corr > -500 && corr < 500, fmt.Sprintf("correlation %d bp, want about 0", corr))
			default:
				infoCheck(t, rep, seed, key+".phase_follows_lag", corr == -10_000, fmt.Sprintf("correlation %d bp, want -10000 (the sign reversed)", corr))
			}
		}
	}
	rep.flush(t)
}
