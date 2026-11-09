package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s47 sizes: a source market that trades every fourth tick (the thin market of
// INF-S2) read from the other region over an edge of 100 at an information speed
// of 50, so every lag is 1 + ceil(100 / 50) = 3 (spec-information.md 12.1).
const (
	s47Ticks = 60
	s47Every = 4
	s47Lag   = 3
	s47Error = 500
	// s47From is the first vintage the checks read: the record is full of trades.
	s47From = 8
)

// s47Quote is the price the thin market clears at on a trade tick: it changes at
// each trade, so a rumour of the carried-forward price names the trade it carries.
func s47Quote(tick int64) (economysim.Price, bool) {
	if tick%s47Every != 0 {
		return 0, false
	}
	return economysim.Price(90 + 3*(tick/s47Every%8)), true
}

// TestScenarioS47_ThinSourceAge runs the first assertion of scenario S47 (INF-S2,
// STB-L94, plan 2-13 D478) on pack two with the information overlay, seeds 1 to
// scenarioSeedCount(): two growers of the north quote staple only on every fourth
// tick, so the market there publishes its carried-forward price with a growing age,
// and the readers of the south are shown rumours of it. It asserts that every
// rumour of the thin market carries the source's age at the vintage tick, that
// the lag is the one the distance gives whatever the age, and that the price is the
// last trade's within the error (the age and the lag are never blurred). The
// second assertion of S47 (a merchant that ignores SourceAge ships, one that
// reads it does not) waits for the merchants module (plan 15) and is recorded as
// waiting.
//
// Requirement: FR-INF-016, FR-INF-017, FR-INF-050
//
// Guard: without the age in the rumour a strategy cannot tell a quiet market from a
// stable price (STB-L94); this is the one assertion INF owes the scenario.
func TestScenarioS47_ThinSourceAge(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rep := newPackScenReport("two", "S47 a thin source market (INF-S2, first assertion; the merchant assertion waits for plan 15)")
	for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
		r, staple := thinWorld(t, seed, s47Error, s47Quote)
		r.run(t, s47Ticks)
		north, south := r.region["north"], r.region["south"]
		ticks, rumours := r.strat.rumoursAbout(south, north, staple)
		var aged, bad int64
		var maxAge int64
		for i, ru := range rumours {
			v := ticks[i] - ru.Lag
			if v < s47From {
				continue
			}
			last := v - v%s47Every
			p, _ := s47Quote(last)
			slack := (int64(p)*s47Error + 9_999) / 10_000
			d := int64(ru.Price) - int64(p)
			if ru.Lag != s47Lag || ru.SourceAge != v-last || d > slack || -d > slack {
				bad++
			}
			if ru.SourceAge > 0 {
				aged++
			}
			maxAge = max(maxAge, ru.SourceAge)
		}
		rep.num(seed, "s47.rumours", int64(len(rumours)))
		rep.num(seed, "s47.aged_rumours", aged)
		rep.num(seed, "s47.max_source_age", maxAge)
		infoCheck(t, rep, seed, "s47.age_lag_price_of_the_vintage", bad == 0 && len(rumours) > 0, fmt.Sprintf("%d of %d rumours do not carry the age, lag and price of their vintage", bad, len(rumours)))
		infoCheck(t, rep, seed, "s47.thin_market_is_aged", aged > 0 && maxAge == s47Every-1, fmt.Sprintf("%d aged rumours, largest age %d, want some and %d", aged, maxAge, s47Every-1))
	}
	// The second assertion needs a merchant that ships: recorded as waiting (plan 2-13 D478).
	rep.absent(0, "s47.second_assertion_waits_for_mer_plan_15")
	rep.flush(t)
	waiting := newPackScenReport("two", "S46 a shipper on a rumoured gap (INF-S1, waits for MER plan 15 and the dispatch cap of plan 12)")
	waiting.absent(0, "s46.waits_for_mer_plan_15")
	waiting.flush(t)
}
