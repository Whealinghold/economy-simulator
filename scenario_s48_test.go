package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s48 sizes: pack three is a chain north - south - far_south of edges of 100, and the
// information speed is 50. The two edges between south and far_south are closed from
// tick 20 to 39, stretched to twice their length from 40 to 59 and back to normal
// from 60; the edges between north and south never change.
const (
	s48Ticks   = 80
	s48Close   = 20
	s48Stretch = 40
	s48Back    = 60
	s48Speed   = 50
	s48Error   = 500
)

// s48Lag is the lag of a rumour whose shortest walk is dist long: 1 + ceil(dist / 50).
func s48Lag(dist int64) int64 { return 1 + (dist+s48Speed-1)/s48Speed }

// s48Edge is the length of the edge between south and far_south at a tick; the one
// between north and south is always 100.
func s48Edge(tick int64) int64 {
	if tick >= s48Stretch && tick < s48Back {
		return 200
	}
	return 100
}

// s48Want is the lag the rumour from source to reader must have at tick, and false
// when the closure cuts the pair (any walk to or from far_south).
func s48Want(tick int64, reader, source string) (int64, bool) {
	if (reader == "far_south" || source == "far_south") && tick >= s48Close && tick < s48Stretch {
		return 0, false
	}
	if reader != "far_south" && source != "far_south" {
		return s48Lag(100), true
	}
	if reader == "south" || source == "south" {
		return s48Lag(s48Edge(tick)), true
	}
	return s48Lag(100 + s48Edge(tick)), true
}

// TestScenarioS48_ClosureAndReopening runs scenario S48 (INF-S3, STB-L95, plan 2-13
// D478) on pack three with the information overlay, seeds 1 to scenarioSeedCount(),
// with the default trading and a recording strategy that logs the rumours it is
// shown. The edges between south and far_south are closed, stretched and restored
// between steps with the unexported setter of plan 3. It asserts that during the
// closure no view of far_south holds a rumour and no view elsewhere holds one about
// far_south, that RumoursServed counts the pairs as NoPath for exactly those ticks
// and EdgeStatusChanged names the changes, that every rumour has the lag the
// shortest walk of the moment gives (the vintage steps by the change of lag), that
// its price is the market's price at its vintage tick within the error, and that
// after the reopening the rumours return with the old lags.
//
// Requirement: FR-INF-010, FR-INF-016, FR-INF-026
//
// Guard: a lag that was not derived from the current graph, or a rumour built before a change that
// moved with it, would show a lag or a presence that the walk of the tick does not
// give (STB-L95).
func TestScenarioS48_ClosureAndReopening(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rep := newPackScenReport("three", "S48 a closure and a reopening in a chain (INF-S3)")
	for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
		s := &recordingStrategy{}
		r := newInfoRun(t, packInformation("three"), seed, func(c *economysim.Config) { c.Modules.Information.RumourErrorBP = s48Error }, s)
		names := map[economysim.RegionID]string{}
		for n, id := range r.region {
			names[id] = n
		}
		south, far := r.region["south"], r.region["far_south"]
		set := func(factor int64, closed bool) {
			t.Helper()
			ch := []economysim.EdgeStatus{{From: south, To: far, Closed: closed, FactorBP: factor}, {From: far, To: south, Closed: closed, FactorBP: factor}}
			if err := economysim.SetEdgeStatusesForTest(r.w, ch, economysim.CauseConsumer); err != nil {
				t.Fatalf("seed %d: setEdgeStatuses() at tick %d error = %v, want nil", seed, r.w.Tick(), err)
			}
		}
		for tick := range s48Ticks {
			switch tick {
			case s48Close:
				set(10_000, true)
			case s48Stretch:
				set(20_000, false)
			case s48Back:
				set(10_000, false)
			}
			r.run(t, 1)
		}
		cleared := map[[3]int64]int64{} // tick, region, commodity -> published price
		var changes []economysim.EdgeStatusChanged
		served := map[int64]economysim.RumoursServed{} // tick -> far_south's event
		for _, e := range r.rec.events {
			switch x := e.(type) {
			case economysim.MarketCleared:
				cleared[[3]int64{x.Tick, int64(x.Region), int64(x.Commodity)}] = int64(x.Price)
			case economysim.EdgeStatusChanged:
				changes = append(changes, x)
			case economysim.RumoursServed:
				if x.Region == far {
					served[x.Tick] = x
				}
			}
		}
		var offered, wrongLag, wrongPrice, leaked, missing, before int64
		for _, v := range r.strat.seen {
			from := map[economysim.RegionID]bool{}
			for _, ru := range v.rumours {
				from[ru.Region] = true
				want, open := s48Want(v.tick, names[v.region], names[ru.Region])
				offered++
				switch {
				case !open:
					leaked++
				case ru.Lag != want:
					wrongLag++
				}
				p := cleared[[3]int64{v.tick - ru.Lag, int64(ru.Region), int64(ru.Commodity)}]
				slack := (p*s48Error + 9_999) / 10_000
				if d := int64(ru.Price) - p; d > slack || -d > slack {
					wrongPrice++
				}
			}
			if v.tick >= s48Stretch && v.region == far && len(v.rumours) == 0 {
				missing++
			}
			if v.tick >= 10 && v.tick < s48Close && v.region == far && len(v.rumours) > 0 {
				before++
			}
		}
		var cutTicks, badServed int64
		for tick := int64(0); tick < s48Ticks; tick++ {
			ev, ok := served[tick]
			cut := tick >= s48Close && tick < s48Stretch
			if cut {
				cutTicks++
			}
			// Far_south has informed readers every tick, so the event exists; a tick before
			// the first rumour can reach is NotYet, never NoPath.
			if !ok || (ev.NoPath > 0) != cut || cut && ev.Offered != 0 {
				badServed++
			}
		}
		wantChanges := []int64{s48Close, s48Close, s48Stretch, s48Stretch, s48Back, s48Back}
		okChanges := len(changes) == len(wantChanges)
		for i := 0; okChanges && i < len(changes); i++ {
			okChanges = changes[i].Tick == wantChanges[i] && changes[i].Closed == (wantChanges[i] == s48Close)
		}
		rep.num(seed, "s48.rumours_seen", offered)
		rep.num(seed, "s48.no_path_ticks_of_far_south", cutTicks)
		infoCheck(t, rep, seed, "s48.cut_pairs_absent", leaked == 0 && before > 0, fmt.Sprintf("%d rumours of a closed pair, %d views of far_south with one before it", leaked, before))
		infoCheck(t, rep, seed, "s48.lag_follows_the_walk", wrongLag == 0 && offered > 0, fmt.Sprintf("%d of %d rumours with a lag other than the walk's", wrongLag, offered))
		infoCheck(t, rep, seed, "s48.price_is_the_vintage", wrongPrice == 0, fmt.Sprintf("%d rumours whose price is not the market's at the vintage", wrongPrice))
		infoCheck(t, rep, seed, "s48.rumours_return_after_reopening", missing == 0, fmt.Sprintf("%d views of far_south with no rumour after the reopening", missing))
		infoCheck(t, rep, seed, "s48.served_agrees_with_closure", badServed == 0 && cutTicks > 0, fmt.Sprintf("%d RumoursServed disagree with the closure", badServed))
		infoCheck(t, rep, seed, "s48.edge_status_events", okChanges, fmt.Sprintf("EdgeStatusChanged = %+v, want two at each of ticks %d, %d, %d", changes, s48Close, s48Stretch, s48Back))
	}
	rep.flush(t)
}
