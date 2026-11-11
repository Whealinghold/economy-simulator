package economysim_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// This file holds the dynamic parts of X1 and X2 of spec-reference-chain.md 7.4 (plan 2-26
// D701, D702, D706). They run report-only: the warm merchants pack (packMerchantsWarm) loses
// its households near tick 730 with or without the merchants (N-314), so the tests assert
// the exact facts that hold whatever the economy does (the audit every tick, the escrow
// identity, the books) and report the rest, with "absent" where a series has no data. A band
// is asserted only where bandIn finds a value (FR-REF-038, FR-REF-024). When the households
// reach 0 the run logs FINDING N-314 and reports on; it does not fail.

const (
	xTicks = 1_095
	xYear  = 365
)

// xObs counts the contracts of a run.
type xObs struct{ formed, settled int64 }

func (o *xObs) Observe(e economysim.Event) {
	switch e.(type) {
	case economysim.ContractFormed:
		o.formed++
	case economysim.ContractSettled:
		o.settled++
	}
}

// xRun is one run of X1 or X2: the snapshots at the ticks the report reads.
type xRun struct {
	// snaps maps a tick (0, 100, 365, 730, 1095) to the snapshot there.
	snaps map[int64]economysim.Snapshot
	// priced is, per year end, whether both fuel markets had a price.
	priced map[int64]bool
	fuel   *trnObs
	cts    *xObs
	// extinct is the first tick with 0 households, or 0.
	extinct int64
}

// xReadAt are the ticks whose snapshot is kept; the year ends are among them.
var xReadAt = map[int64]bool{0: true, 100: true, 365: true, 730: true, 1_095: true}

// xSimulate runs the pack for xTicks with the audit, the merchant strategy and the escrow
// identity checked after every step (Escrow is the sum of the open contracts' escrow).
func xSimulate(t *testing.T, cfg economysim.Config, seed uint64) xRun {
	t.Helper()
	scenSem <- struct{}{}
	defer func() { <-scenSem }()
	run := xRun{snaps: map[int64]economysim.Snapshot{}, priced: map[int64]bool{}, cts: &xObs{}}
	run.fuel = newTrnObs(merCommodity(t, cfg, "fuel"), merCommodity(t, cfg, "haul"))
	opts := append(strategy.Options(), economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(run.fuel), economysim.WithObserver(run.cts))
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	for tick := int64(0); tick <= xTicks; tick++ {
		if tick > 0 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step at tick %d (seed %d) error = %v, want nil (the audit runs every tick)", w.Tick(), seed, err)
			}
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot at tick %d (seed %d) error = %v, want nil", tick, seed, err)
		}
		xCheckEscrow(t, s, seed, tick)
		if s.Households == 0 && run.extinct == 0 {
			run.extinct = tick
		}
		if xReadAt[tick] {
			run.snaps[tick] = s
			run.priced[tick] = run.fuel.price[0] > 0 && run.fuel.price[1] > 0
		}
	}
	return run
}

// xCheckEscrow fails the test if the escrow is not the sum held by the open contracts.
func xCheckEscrow(t *testing.T, s economysim.Snapshot, seed uint64, tick int64) {
	t.Helper()
	var open economysim.Money
	for _, c := range s.Contracts {
		open += c.Escrowed
	}
	if s.Escrow != open {
		t.Fatalf("seed %d tick %d: Escrow = %d, open contracts hold %d, want equal", seed, tick, s.Escrow, open)
	}
}

// xCommon records what X1 and X2 share: the households at the read ticks, the contracts
// and the exact books; it logs FINDING N-314 when the households reached 0.
func xCommon(t *testing.T, rep *scenReport, name, pack string, seed uint64, r xRun) {
	t.Helper()
	for _, tick := range []int64{100, 365, 730, 1_095} {
		rep.num(seed, fmt.Sprintf("%s.households_t%d", name, tick), r.snaps[tick].Households)
	}
	rep.num(seed, name+".contracts_formed", r.cts.formed)
	rep.num(seed, name+".contracts_settled", r.cts.settled)
	end := r.snaps[xTicks]
	books := r.fuel.dispatched == r.fuel.arrived+int64(len(end.Shipments))
	checkExact(t, rep, name, pack, seed, name+".books", books, "dispatched %d, arrived %d, in the snapshot %d", r.fuel.dispatched, r.fuel.arrived, len(end.Shipments))
	checkExact(t, rep, name, pack, seed, name+".lines_add_up", r.fuel.lineBroken == 0, "%d arrived lines do not add up", r.fuel.lineBroken)
	if r.extinct > 0 {
		t.Logf("FINDING N-314: %s pack %s seed %d: the households reached 0 at tick %d; the gaps and shares below are reported as they are", name, pack, seed, r.extinct)
	}
}

// x1GapBand reports whether the gaps of ticks from.. are within the band of X1: the mean at
// most margin and the largest at most worst. shortcut: the transport cost of the edge is
// taken as 0, because no band has a value; whoever sets x1.margin_bp (STY 8.4) states the
// term with it.
func x1GapBand(gaps []int64, from int, margin, worst int64) bool {
	if from >= len(gaps) {
		return true
	}
	var sum, peak int64
	for _, g := range gaps[from:] {
		sum += g
		peak = max(peak, g)
	}
	return sum/int64(len(gaps)-from) <= margin && peak <= worst
}

// TestScenarioX1_Arbitrage runs the dynamic part of X1 (REF 7.4) on the warm merchants pack
// two for 1 095 ticks, seeds 1 and 2: the fuel price in the two regions with an edge, a
// carrier and the merchants. It asserts the exact facts (the invariants every tick through
// the audit, the escrow back to the open contracts' sum after every step, the shipments in
// the books) and reports the households at ticks 100, 365, 730 and 1 095, the contracts
// formed and settled, and the mean and the largest fuel gap per year in basis points of the
// cheap price (absent while a market has no price or the year ends with no households, since the last prices of a dead economy are frozen). The convergence band (the gap at most the
// transport cost plus x1.margin_bp and x1.worst_bp after x1.warmup, in the last year) is
// asserted only when all three keys have a value. Arithmetic: years end at ticks 365, 730
// and 1 095.
//
// Requirement: FR-VIA-043, FR-VIA-044, FR-VIA-045, FR-CAR-003, FR-TRN-005, FR-REF-038
func TestScenarioX1_Arbitrage(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const bands = "testdata/chains/bands/two.toml"
	rep := newPackScenReport("two", "X1 two regions, arbitrage (dynamic, report-only)")
	cfg := loadValidPack(t, packMerchantsWarm("two")...)
	for _, seed := range []uint64{1, 2} {
		t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) {
			t.Parallel()
			r := xSimulate(t, cfg, seed)
			xCommon(t, rep, "x1", "two", seed, r)
			gaps := r.fuel.gapBP
			for year := int64(1); year <= 3; year++ {
				from, to := (year-1)*xYear, year*xYear
				name := fmt.Sprintf("x1.y%d", year)
				if !r.priced[to] || r.snaps[to].Households == 0 || to > int64(len(gaps)) {
					rep.absent(seed, name+".mean_gap_bp")
					rep.absent(seed, name+".largest_gap_bp")
					continue
				}
				mean, _ := windowMean(gaps, from, to)
				rep.num(seed, name+".mean_gap_bp", mean)
				rep.num(seed, name+".largest_gap_bp", slicesMax(gaps[from:to]))
			}
			margin, okM := bandIn(t, bands, "x1.margin_bp")
			worst, okW := bandIn(t, bands, "x1.worst_bp")
			warm, okT := bandIn(t, bands, "x1.warmup")
			if okM && okW && okT && r.priced[xTicks] && r.snaps[xTicks].Households > 0 {
				checkExact(t, rep, "X1", "two", seed, "band x1.gap", x1GapBand(gaps, max(int(warm), xTicks-xYear), margin, worst),
					"the gap of the last year against margin %d and worst %d", margin, worst)
			}
		})
	}
	t.Cleanup(func() { rep.flush(t) })
}

// slicesMax is the largest value of a non-empty slice.
func slicesMax(s []int64) int64 {
	m := s[0]
	for _, v := range s {
		m = max(m, v)
	}
	return m
}

// xShares returns, per region, the share in basis points of each role among the region's
// households, and the pooled share of each role over all regions (the pooled census, A3);
// both are indexed [region][role] and [role].
func xShares(s economysim.Snapshot, regions, roles int) (per [][]int64, pooled []int64) {
	hh := make([][]int64, regions)
	tot := make([]int64, regions)
	all := make([]int64, roles)
	var sum int64
	for _, r := range s.Roles {
		if int(r.Region) >= regions {
			continue
		}
		if hh[r.Region] == nil {
			hh[r.Region] = make([]int64, roles)
		}
		hh[r.Region][r.Role] += r.Households
		tot[r.Region] += r.Households
		all[r.Role] += r.Households
		sum += r.Households
	}
	per = make([][]int64, regions)
	pooled = make([]int64, roles)
	for i := range per {
		per[i] = make([]int64, roles)
		for j := range roles {
			if tot[i] > 0 && hh[i] != nil {
				per[i][j] = hh[i][j] * 10_000 / tot[i]
			}
		}
	}
	for j := range roles {
		if sum > 0 {
			pooled[j] = all[j] * 10_000 / sum
		}
	}
	return per, pooled
}

// xDistance is the largest over the regions of the sum over the roles of the absolute
// difference between the region's share and the pooled share, in basis points.
func xDistance(per [][]int64, pooled []int64) int64 {
	var worst int64
	for _, row := range per {
		var d int64
		for j, v := range row {
			d += absInt(v - pooled[j])
		}
		worst = max(worst, d)
	}
	return worst
}

// TestScenarioX2_ComparativeAdvantage runs the dynamic part of X2 (REF 7.4) on the warm
// merchants pack three for 1 095 ticks, seeds 1 and 2. It asserts the same exact facts as X1
// and reports, at the end of each year, the distance between the regions' occupation shares
// and the pooled census (the largest over the regions of the sum of the absolute share
// differences, in basis points) and the share of the start distance left (absent while the
// start distance or the households are 0). x2.share_band_bp is asserted on the share left at
// the end only when it has a value. Arithmetic: a region of 40% staple growers against a
// pooled 30% and 60% against 70% is a distance of 1 000 + 1 000 = 2 000.
//
// Requirement: FR-REF-038, FR-REF-024, FR-CAR-003, FR-TRN-005
func TestScenarioX2_ComparativeAdvantage(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const bands = "testdata/chains/bands/three.toml"
	rep := newPackScenReport("three", "X2 three regions, comparative advantage (dynamic, report-only)")
	cfg := loadValidPack(t, packMerchantsWarm("three")...)
	for _, seed := range []uint64{1, 2} {
		t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) {
			t.Parallel()
			r := xSimulate(t, cfg, seed)
			xCommon(t, rep, "x2", "three", seed, r)
			per, pooled := xShares(r.snaps[0], len(cfg.Regions), len(cfg.Roles))
			start := xDistance(per, pooled)
			rep.num(seed, "x2.start_distance_bp", start)
			for year := int64(1); year <= 3; year++ {
				name := fmt.Sprintf("x2.y%d", year)
				s := r.snaps[year*xYear]
				if s.Households == 0 || start == 0 {
					rep.absent(seed, name+".share_left_bp")
					continue
				}
				per, pooled := xShares(s, len(cfg.Regions), len(cfg.Roles))
				d := xDistance(per, pooled)
				rep.num(seed, name+".distance_bp", d)
				rep.num(seed, name+".share_left_bp", d*10_000/start)
				if band, ok := bandIn(t, bands, "x2.share_band_bp"); ok && year == 3 {
					checkExact(t, rep, "X2", "three", seed, "band x2.share_band_bp", d*10_000/start <= band, "share left %d against %d", d*10_000/start, band)
				}
			}
		})
	}
	t.Cleanup(func() { rep.flush(t) })
}

var xSetByRe = regexp.MustCompile(`(?m)^\s*set_by\s*=`)

// TestScenarioX_BandsStayUnset fails if one of the keys X1 and X2 read has a value without a
// set_by line, so the report-only state cannot become an unreviewed assertion.
//
// Requirement: FR-REF-024
//
// Guard: a value typed into a bands file by hand would turn a report into an assertion with
// no reviewed run behind it (STY 8.4).
func TestScenarioX_BandsStayUnset(t *testing.T) {
	t.Parallel()
	for _, k := range []struct{ file, key string }{
		{"two", "x1.margin_bp"}, {"two", "x1.worst_bp"}, {"two", "x1.warmup"}, {"three", "x2.share_band_bp"},
	} {
		path := "testdata/chains/bands/" + k.file + ".toml"
		if _, set := bandIn(t, path, k.key); !set {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		table := string(b)
		i := strings.Index(table, "["+k.key+"]")
		rest := table[i+1:]
		if j := strings.Index(rest, "\n["); j >= 0 {
			rest = rest[:j]
		}
		if !xSetByRe.MatchString(rest) {
			t.Errorf("%s [%s] has a value and no set_by line, want both (FR-REF-024)", path, k.key)
		}
	}
}

// TestX1GapBand verifies the band predicate on a hand-made series: gaps 100, 200, 300 from
// index 0 have the mean 200 and the largest 300.
//
// Requirement: FR-REF-024
func TestX1GapBand(t *testing.T) {
	t.Parallel()
	g := []int64{100, 200, 300}
	for _, tc := range []struct {
		name          string
		from          int
		margin, worst int64
		want          bool
	}{
		{"mean and peak inside", 0, 200, 300, true},
		{"mean over", 0, 199, 300, false},
		{"peak over", 0, 200, 299, false},
		{"window past the end", 3, 0, 0, true},
		{"last two: mean 250", 1, 250, 300, true},
	} {
		if got := x1GapBand(g, tc.from, tc.margin, tc.worst); got != tc.want {
			t.Errorf("%s: x1GapBand(from %d, margin %d, worst %d) = %t, want %t", tc.name, tc.from, tc.margin, tc.worst, got, tc.want)
		}
	}
}
