package indicator_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// Unit tests of the measures that the shock, pack and trade fact tests build on
// (plan 2-19 Phase 7). They run under -short: the run-level tests skip, these do
// not, so the arithmetic of a fact is checked on built series.

// TestStylised_ShockMeasures verifies the readings of S01 and P05 on built price
// series: the mean over a window, the first excursion out of a band, the tick a
// price settles for good, the tick the ten-tick mean returns, and the order of
// excursions along a chain.
//
// Requirement: FR-STY-024, FR-STY-030
func TestStylised_ShockMeasures(t *testing.T) {
	t.Parallel()
	// A flat price of 100 with a shock of 150 from tick 5 to 9 and 100 again.
	s := []int64{100, 100, 100, 100, 100, 150, 150, 150, 150, 150, 100, 100, 100, 100, 100}
	if got, ok := meanOver(s, 5, 10); !ok || got != 150 {
		t.Errorf("meanOver(s, 5, 10) = %d, %v, want 150, true", got, ok)
	}
	if got, ok := meanOver([]int64{1, 2}, 0, 2); !ok || got != 2 {
		t.Errorf("meanOver([1 2], 0, 2) = %d, %v, want 2, true (1.5 rounds up)", got, ok)
	}
	if _, ok := meanOver(s, 3, 3); ok {
		t.Error("meanOver(s, 3, 3) ok = true, want false for an empty range")
	}
	if tick, above, ok := firstOutside(s, 0, 15, 100, 1000); !ok || tick != 5 || !above {
		t.Errorf("firstOutside(s, band 1000) = %d, above %v, %v, want tick 5, above, true", tick, above, ok)
	}
	if _, _, ok := firstOutside(s, 0, 5, 100, 1000); ok {
		t.Error("firstOutside(s[:5]) ok = true, want false (the price never leaves its band)")
	}
	// 110 is exactly 1000 bp of 100: inside, since only a strictly larger distance is outside.
	if outsideBand(110, 100, 1000) || !outsideBand(111, 100, 1000) {
		t.Errorf("outsideBand(110/111, 100, 1000) = %v, %v, want false, true", outsideBand(110, 100, 1000), outsideBand(111, 100, 1000))
	}
	if at, ok := settledAt(s, 10, 100, 1000); !ok || at != 10 {
		t.Errorf("settledAt(s, from 10) = %d, %v, want 10, true", at, ok)
	}
	if at, ok := settledAt(append(slices.Clone(s), 150), 10, 100, 1000); ok {
		t.Errorf("settledAt(s + 150) = %d, true, want false (the last price is outside)", at)
	}
	// After the shock the price is 100. The ten-tick mean ending at tick t holds the
	// shocked ticks 5 to 9 that are in [t-9, t]: it is 120 at tick 15 (ticks 6 to 15:
	// four of 150), 110 at tick 17 (two of 150, the band edge, inside) and stays so.
	long := append(slices.Clone(s), slices.Repeat([]int64{100}, 10)...)
	if at, ok := recoveredAt(long, 10, 100, 1000); !ok || at != 17 {
		t.Errorf("recoveredAt(long, from 10) = %d, %v, want 17, true", at, ok)
	}
	first := map[string]int64{"mineral": 10, "refined": 12, "tool": 11}
	depth := map[string]int{"mineral": 0, "refined": 1, "tool": 2}
	if got := excursionOrder(first, depth); got == "" {
		t.Error("excursionOrder(tool at 11 before refined at 12) = none, want an order error")
	}
	first["tool"] = 14
	if got := excursionOrder(first, depth); got != "" {
		t.Errorf("excursionOrder(in order) = %q, want none", got)
	}
}

// TestStylised_ChainDepths verifies that the chain of a shocked recipe is read
// from the recipes, with no identifier named: the output at depth 0, what is made
// from it at depth 1 and 2.
//
// Requirement: FR-STY-024
func TestStylised_ChainDepths(t *testing.T) {
	t.Parallel()
	cfg := loadPack(t, "base.toml", "one.toml")
	got := chainDepths(cfg, "dig_mineral")
	want := map[string]int{"mineral": 0, "refined": 1, "tool_a": 2, "tool_b": 2}
	for c, d := range want {
		if got[c] != d {
			t.Errorf("chainDepths(dig_mineral)[%s] = %d, want %d", c, got[c], d)
		}
	}
	if _, ok := got["waste"]; ok {
		t.Errorf("chainDepths(dig_mineral) holds waste, want the discard output left out")
	}
	if out := outputOf(cfg, "cut_fuel"); !slices.Equal(out, []string{"fuel"}) {
		t.Errorf("outputOf(cut_fuel) = %v, want [fuel]", out)
	}
}

// TestStylised_Scripts verifies the reader of the scripts: the shocks of S08 and
// S01 leg (a), the buyer of leg (b), and the span of a shock.
//
// Requirement: FR-STY-024, FR-STY-030, FR-REF-026
func TestStylised_Scripts(t *testing.T) {
	t.Parallel()
	from, to, region, recipe := shockSpan(readScript(t, "s08.toml"))
	if from != 400 || to != 520 || region != "north" || recipe != "dig_mineral" {
		t.Errorf("shockSpan(s08) = %d, %d, %s, %s, want 400, 520, north, dig_mineral", from, to, region, recipe)
	}
	b := readScript(t, "s01b.toml")
	if len(b) != 1 || !b[0].Buyer || b[0].Tick != 400 || b[0].Money != 2_000_000 || b[0].Commodity != "fuel" || b[0].ShareBP != 500 || b[0].Ticks != 60 {
		t.Errorf("readScript(s01b) = %+v, want one buyer of fuel at tick 400, 500 bp of the volume for 60 ticks", b)
	}
	var r stylisedRun
	r.cfg = loadPack(t, "base.toml", "one.toml")
	if f, e, reg, c := s01Window(&r, b); f != 400 || e != 460 || reg != "north" || c != "fuel" {
		t.Errorf("s01Window(buyer) = %d, %d, %s, %s, want 400, 460, north, fuel", f, e, reg, c)
	}
	a := readScript(t, "s01.toml")
	if f, e, reg, c := s01Window(&r, a); f != 400 || e != 460 || reg != "north" || c != "fuel" {
		t.Errorf("s01Window(shock) = %d, %d, %s, %s, want 400, 460, north, fuel (the output of cut_fuel)", f, e, reg, c)
	}
}

// TestStylised_FamineRead verifies the households read of S02 on a built series:
// the level before the shock, the trough, the tick the band is regained, and a run
// with no household before the shock.
//
// Requirement: FR-STY-031, FR-STY-036
func TestStylised_FamineRead(t *testing.T) {
	t.Parallel()
	hh := slices.Repeat([]int64{600}, int(famineShockAt))
	hh = append(hh, 500, 300, 200, 250, 400, 530, 560, 600)
	r := &stylisedRun{households: hh}
	// Index 423 is 600 (the level); 200 is at 426; 530 at 429 is under 90% of 600 (540),
	// 560 at 430 is the first inside.
	st, ok := famineRead(r, 1000)
	if !ok || st.level != 600 || st.trough != 200 || st.troughAt != famineShockAt+2 || st.recoverAt != famineShockAt+6 {
		t.Errorf("famineRead = %+v, %v, want level 600, trough 200 at %d, recovered at %d", st, ok, famineShockAt+2, famineShockAt+6)
	}
	if _, ok := famineRead(&stylisedRun{households: slices.Repeat([]int64{0}, int(famineShockAt)+5)}, 1000); ok {
		t.Error("famineRead(no household before the shock) ok = true, want false")
	}
}

// TestStylised_ProvokedStatistic verifies the statistic of a detector by the
// search of its separated key: the longest pinned run of a built series for R5a,
// and a series the detector cannot read.
//
// Requirement: FR-STY-071, FR-STY-065
func TestStylised_ProvokedStatistic(t *testing.T) {
	t.Parallel()
	var pinned []verdict.Point
	for i := int64(0); i < 200; i++ {
		share := int64(0)
		if i >= 50 && i < 150 {
			share = 10_000
		}
		pinned = append(pinned, verdict.Point{Tick: i, Value: share})
	}
	in := verdict.Input{Pinned: [][]verdict.Point{pinned}, Outcomes: 200}
	if got, ok := statisticOf(in, verdict.R5a); !ok || got != 100 {
		t.Errorf("statisticOf(R5a, a pinned run of 100 ticks) = %d, %v, want 100, true", got, ok)
	}
	if got, ok := statisticOf(verdict.Input{}, verdict.R5a); ok {
		t.Errorf("statisticOf(R5a, no series) = %d, true, want false", got)
	}
	for _, d := range verdict.Detectors() {
		k, has := separated[d]
		if !has {
			t.Errorf("separated has no key for %s", d)
			continue
		}
		if _, ok := workingThresholds().Get(k.name); !ok {
			t.Errorf("separated key %s of %s is not a threshold of 6.2", k.name, d)
		}
	}
	if extreme([]int64{3, 9, 5}, false) != 9 || extreme([]int64{3, 9, 5}, true) != 3 || extreme(nil, true) != 0 {
		t.Error("extreme: want the largest, the smallest, and 0 for none")
	}
}

// TestStylised_TradeMeasures verifies the readings of the trade facts on built
// data: the layers of a variant, the events of a closure, the spread of a pair, the
// share of ticks near a quote, and the ladder order of relief.
//
// Requirement: FR-STY-025, FR-STY-037, FR-STY-038
func TestStylised_TradeMeasures(t *testing.T) {
	t.Parallel()
	if got := tradeLayers("three", true); got[len(got)-1] != "overlay/merchants-three.toml" || len(got) != 4 {
		t.Errorf("tradeLayers(three, merchants) = %v, want the edges, transport, information and merchants-three", got)
	}
	if got := tradeLayers("two", false); len(got) != 3 {
		t.Errorf("tradeLayers(two, control) = %v, want three layers and no merchants", got)
	}
	both := []economysim.EdgeStatusChanged{{Closed: true}, {Closed: true}, {Closed: false}, {Closed: false}}
	if got := edgeEvents(both); got != "" {
		t.Errorf("edgeEvents(2 closing, 2 reopening) = %q, want none", got)
	}
	if got := edgeEvents(both[:3]); got == "" {
		t.Error("edgeEvents(2 closing, 1 reopening) = none, want an error")
	}
	// Two regions, one commodity: prices 100 and 150 at tick 0, so the spread is
	// 50 / mean 125 = 4 000 bp.
	r := &stylisedRun{names: economysim.Names{Regions: []string{"a", "b"}, Commodities: []string{"x"}}, obs: newRunObs()}
	r.obs.markets[[2]int{0, 0}] = &marketLog{Price: []int64{100}}
	r.obs.markets[[2]int{1, 0}] = &marketLog{Price: []int64{150}}
	if got, ok := pairSpread(r, "a", "b", 0, 1); !ok || got != 4000 {
		t.Errorf("pairSpread(100, 150) = %d, %v, want 4000, true", got, ok)
	}
	if _, ok := pairSpread(r, "a", "b", 1, 2); ok {
		t.Error("pairSpread(no price at the tick) ok = true, want false")
	}
	// A quote of 100 for four ticks; the price is 100, 105, 120, 100: three of four within 1 000 bp.
	r.obs.quote[[2]int{0, 0}] = []int64{100, 100, 100, 100}
	r.obs.markets[[2]int{0, 0}] = &marketLog{Price: []int64{100, 105, 120, 100}}
	if got := quoteShare(r); got.Why != "" || got.Value != 7500 {
		t.Errorf("quoteShare = %+v, want 7500 (3 of 4 ticks within 1 000 bp of the quote)", got)
	}
	o := newRunObs()
	o.relief = []int64{0, 0, 5}
	o.removed = []removal{{Tick: 1, Agent: 7, Cause: economysim.CauseStarved}}
	if got := ladderOrder(o); got == "" {
		t.Error("ladderOrder(removal at 1, relief at 2) = none, want an order error")
	}
	o.removed[0].Tick = 3
	if got := ladderOrder(o); got != "" {
		t.Errorf("ladderOrder(relief before removal) = %q, want none", got)
	}
}

// TestTicks_GateConstants verifies the run lengths of the tiers (D595, REF 8.2): a
// gate run is at most 1 800 ticks, the nightly run is the gate run, and the only
// file that names the long run is the one behind the stylised_long tag, so the
// default build cannot compile a run of a hundred years.
//
// Requirement: FR-STY-083, FR-STY-081
//
// Guard: a long run in an untagged file would put a 24-minute test in the merge
// gate.
func TestTicks_GateConstants(t *testing.T) {
	t.Parallel()
	if gateTicks > 1800 || nightlyTicks != gateTicks {
		t.Errorf("gateTicks = %d, nightlyTicks = %d, want gateTicks at most 1800 and nightlyTicks equal to it", gateTicks, nightlyTicks)
	}
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var tagged []string
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), "_test.go") {
			continue
		}
		b, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		src := string(b)
		switch {
		case strings.HasPrefix(src, "//go:build stylised_long\n"):
			tagged = append(tagged, f.Name())
		case strings.Contains(src, "longTicks") && f.Name() != "stylised_measures_test.go":
			t.Errorf("%s names longTicks and has no //go:build stylised_long line, want only the tagged file to", f.Name())
		}
	}
	if !slices.Equal(tagged, []string{"stylised_long_test.go"}) {
		t.Errorf("files with the stylised_long tag = %v, want [stylised_long_test.go]", tagged)
	}
}
