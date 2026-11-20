package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds the scenarios ORG-S1 to ORG-S5 of spec-institutions.md 14.3 (S137 to S141 of
// spec-reference-chain.md 7.12). They assert the invariants of the spec and report the measures
// the spec names; they assert no band (every default of the module is a placeholder, N-015,
// N-027), so they write no line of expected/*.scenarios.txt. A multi-seed scenario skips under
// -race and carries the gateFPrefixes prefix TestScenarioORG.

// orgStaplePrice is the reference price of the staple of pack one in the region north
// (spec-institutions.md 13.2), the base of the cap of ORG-S1.
const orgStaplePrice = 96

// orgLiftTick is the tick at which ORG-S1 lifts the control, and orgS1Ticks the run's length.
const (
	orgLiftTick = 30
	orgS1Ticks  = 70
)

// capRun is what ORG-S1 keeps of one run of the staple market with a cap lifted at a tick:
// the trades before and after the lift, the unmatched bid units while the cap is in force
// and the prices of the trades.
type capRun struct {
	before, after []economysim.TradeExecuted
	unmatchedBid  int64
	clamped       int64
}

// runCapped runs pack one with a control of capBP on the staple in north for orgS1Ticks
// ticks under the audit, and lifts it at orgLiftTick by SetLegalLevels (a cap of 0 is
// none). It returns what the run recorded about the staple, commodity 5 of pack one.
func runCapped(t testing.TB, capBP int64, seed uint64) capRun {
	t.Helper()
	var out capRun
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		switch e := e.(type) {
		case economysim.TradeExecuted:
			if e.Commodity != 5 {
				return
			}
			if e.Tick < orgLiftTick {
				out.before = append(out.before, e)
			} else {
				out.after = append(out.after, e)
			}
		case economysim.MarketCleared:
			if e.Commodity == 5 && e.Tick < orgLiftTick {
				out.unmatchedBid += int64(e.UnmatchedBid)
			}
		case economysim.PriceControlled:
			out.clamped += e.Bids + e.Asks
		}
	})
	w, err := economysim.NewWorld(stapleControl(t, capBP, 0), economysim.WithSeed(seed), economysim.WithObserver(obs), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(cap %d, seed %d) error = %v, want nil", capBP, seed, err)
	}
	for tick := range orgS1Ticks {
		if tick == orgLiftTick {
			if err := w.SetLegalLevels([]economysim.LegalLevel{{Control: 0}}); err != nil {
				t.Fatalf("SetLegalLevels() error = %v, want nil", err)
			}
		}
		if err := w.Step(); err != nil {
			t.Fatalf("Step() tick %d, cap %d, seed %d error = %v, want nil (the audit failed)", tick, capBP, seed, err)
		}
	}
	return out
}

// meanPrice is the mean of the prices of the trades (0 for none), reported by ORG-S1.
func meanPrice(trades []economysim.TradeExecuted) int64 {
	if len(trades) == 0 {
		return 0
	}
	var sum int64
	for _, e := range trades {
		sum += int64(e.Price)
	}
	return sum / int64(len(trades))
}

// TestScenarioORG1_PriceCapLifted verifies ORG-S1 (S137) on pack one: a cap on the staple at
// 80, 100 and 120 % of the reference price, lifted at tick 30, over seeds 1 to
// scenarioSeedCount(), with the audit on. Asserted: every trade before the lift is at or below
// the cap, the run conserves (the audit would fail the step) and the cap, once lifted, binds
// nothing (no PriceControlled after the lift is not asserted: the count is reported). Reported
// by Logf: the trades and their mean price before and after, the unmatched bid units under the
// cap and the orders clamped.
//
// Arithmetic: the reference price is 96, so the caps are 96 * 8 000 / 10 000 = 76, 96 and
// 96 * 12 000 / 10 000 = 115 (truncated).
//
// Requirement: FR-ORG-030, FR-ORG-031, FR-ORG-036, FR-ORG-042
//
// Guard: a clamp that let a trade through above the cap, or a lift that did not take effect,
// fails here; the shortage the cap makes is reported and not asserted (spec 6.5).
func TestScenarioORG1_PriceCapLifted(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceEnabled {
		t.Skip("multi-seed scenario: runs in gate F, not in the short or race runs (testing.md 9.1)")
	}
	for _, capBP := range []int64{8_000, 10_000, 12_000} {
		limit := economysim.Price(orgStaplePrice * capBP / 10_000)
		for _, seed := range seedList(scenarioSeedCount()) {
			run := runCapped(t, capBP, seed)
			t.Logf("cap %d bp (%d), seed %d: %d trades at a mean of %d before the lift, %d at a mean of %d after, %d unmatched bid units, %d orders clamped",
				capBP, limit, seed, len(run.before), meanPrice(run.before), len(run.after), meanPrice(run.after), run.unmatchedBid, run.clamped)
			for _, e := range run.before {
				if e.Price > limit {
					t.Errorf("cap %d bp, seed %d: trade at tick %d at price %d, want at most the cap %d", capBP, seed, e.Tick, e.Price, limit)
				}
			}
			if len(run.before) == 0 {
				t.Errorf("cap %d bp, seed %d: no trade of the staple before the lift, want a market the cap acts on", capBP, seed)
			}
		}
	}
}

// orgBodyTally is what ORG-S2 and ORG-S3 keep of the MembershipChanged events of a run: the
// rows, the largest waiting count and, for each (region, body), whether the change of the
// members is what the round says it is.
type orgBodyTally struct {
	rows, maxWaiting, maxHeld int64
	overPlaces, unexplained   int64
}

// tallyMembership folds the events: a row whose members exceed a body's places (the places
// of the body are those of the row), and a row whose members exceed the last row's members
// plus the admitted (a member from nowhere). A fall of the members is allowed: a household that
// leaves the world or the region between two rows is not removed for the places, and a
// household removed for the places would show as a fall with no lapse in the row, which the
// round never does (FR-ORG-006).
func tallyMembership(events []economysim.Event) orgBodyTally {
	var out orgBodyTally
	last := map[[2]int64]int64{}
	for _, e := range events {
		x, ok := e.(economysim.MembershipChanged)
		if !ok {
			continue
		}
		out.rows++
		out.maxWaiting = max(out.maxWaiting, x.Waiting)
		out.maxHeld = max(out.maxHeld, x.Held)
		if x.Places > 0 && x.Members > x.Places {
			out.overPlaces++
		}
		key := [2]int64{int64(x.Region), int64(x.Body)}
		if prev, seen := last[key]; seen && x.Members > prev+x.Admitted {
			out.unexplained++
		}
		last[key] = x.Members
	}
	return out
}

// runOrg runs the world of cfg for ticks from the seed with the recorder and the audit and
// returns the recorded events.
func runOrg(t testing.TB, cfg economysim.Config, seed uint64, ticks int64) []economysim.Event {
	t.Helper()
	return runPack(t, cfg, runSpec{Seed: seed, Ticks: ticks, Drive: driveRun(ticks), Observe: true, Audit: true}).Events
}

// fiveMigrationInstitutions is the five with the trade, information and migration overlays,
// then the polity and institutions overlays: ORG-S3's world, where households move between
// the regions (spec-migration.md 15.3).
var fiveMigrationInstitutions = append(slices.Clone(packInformation("five")), "overlay/migration.toml", "overlay/polity.toml", "overlay/institutions.toml")

// orgS2Ticks and orgS3Ticks are the lengths of ORG-S2 and ORG-S3: the course of the overlay
// is 120 ticks, so the first completions come near tick 125.
const (
	orgS2Ticks = 300
	orgS3Ticks = 200
)

// TestScenarioORG2_BodyOverlayLongRun verifies ORG-S2 (S138): the body overlay on the five
// over a long run. Asserted: the members of a body never exceed its places, nobody is removed
// for the places (every change of the members is the admitted less the lapsed), and the
// households held at a completion stay within the places the trainers of the body offer
// (members times the body's trainee limit of 2). The observed premium is not reported: the
// code has no premium check (PR D, spec 5.7.5).
//
// Requirement: FR-ORG-006, FR-ORG-011, FR-ORG-019, FR-ORG-025, FR-ORG-027
//
// Guard: a body that admitted above its places, or removed a member to make room, fails.
func TestScenarioORG2_BodyOverlayLongRun(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceEnabled {
		t.Skip("multi-seed scenario: runs in gate F, not in the short or race runs (testing.md 9.1)")
	}
	cfg := loadValidPack(t, fiveInstitutions...)
	const traineesPerTrainer = 2
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		tally := tallyMembership(runOrg(t, cfg, seed, orgS2Ticks))
		t.Logf("seed %d: %d rows, largest waiting %d, largest held %d", seed, tally.rows, tally.maxWaiting, tally.maxHeld)
		if tally.rows == 0 || tally.maxWaiting == 0 {
			t.Errorf("seed %d: %d rows, waiting %d, want the body to have outsiders", seed, tally.rows, tally.maxWaiting)
		}
		if tally.overPlaces != 0 || tally.unexplained != 0 {
			t.Errorf("seed %d: %d rows over the places, %d changes of the members that the round does not explain, want 0", seed, tally.overPlaces, tally.unexplained)
		}
		if limit := int64(174 * traineesPerTrainer); tally.maxHeld > limit {
			t.Errorf("seed %d: %d households held, want at most %d (the trainers' places)", seed, tally.maxHeld, limit)
		}
	}
}

// TestScenarioORG3_StartAbovePlacesWithMigration verifies ORG-S3 (S139): a body whose places
// are below the households of its role, on the five with migration on. The start has 218
// cooks in the centre and 174 places, so 44 wait. Asserted: the first row shows the 44
// outsiders, the members never exceed the places and the members never grow but by admission. The largest waiting count is reported.
//
// Requirement: FR-ORG-006, FR-ORG-008, FR-ORG-011
//
// Guard: a round that dropped a member to admit a waiting household, or that let the waiting
// count exceed the eligible, fails.
func TestScenarioORG3_StartAbovePlacesWithMigration(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceEnabled {
		t.Skip("multi-seed scenario: runs in gate F, not in the short or race runs (testing.md 9.1)")
	}
	cfg := loadValidPack(t, fiveMigrationInstitutions...)
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		events := runOrg(t, cfg, seed, orgS3Ticks)
		tally := tallyMembership(events)
		first := firstMembership(events)
		t.Logf("seed %d: %d rows, largest waiting %d, first row %+v", seed, tally.rows, tally.maxWaiting, first)
		if first.Waiting != 218-174 {
			t.Errorf("seed %d: first row waiting = %d, want 44 (218 cooks, 174 places)", seed, first.Waiting)
		}
		if tally.overPlaces != 0 || tally.unexplained != 0 {
			t.Errorf("seed %d: %d rows over the places, %d unexplained changes of the members, want 0", seed, tally.overPlaces, tally.unexplained)
		}
	}
}

// firstMembership returns the first MembershipChanged of a run (the zero value for none).
func firstMembership(events []economysim.Event) economysim.MembershipChanged {
	for _, e := range events {
		if x, ok := e.(economysim.MembershipChanged); ok {
			return x
		}
	}
	return economysim.MembershipChanged{}
}

// ownUseRightDoc is a prohibition (a right with no body) of the own-use recipe that makes the
// staple of pack one in north: nobody may start it where a household is not exempt.
const ownUseRightDoc = `
[modules.institutions]
own_use_bound = %v

[modules.institutions.right.no_staple]
activity = "produce"
subject = "own_staple"
region = "north"
`

// TestScenarioORG4_OwnUseExemptUnlessBound verifies ORG-S4 (S140): pack one with
// overlay/own-use.toml and a prohibition of the recipe that makes the staple for own use, with
// OwnUseBound false and true. Asserted: with it false the households that enter the mode of
// subsistence are those of the world without the module (the exemption) and nothing is
// refused; with it true the right refuses starts, counted in RightRefused.Produce.
//
// Requirement: FR-ORG-016, FR-ORG-022, FR-SUB-023
//
// Guard: a prohibition that starved an own-use household with OwnUseBound false, or one that
// bound nobody with it true, fails.
func TestScenarioORG4_OwnUseExemptUnlessBound(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceEnabled {
		t.Skip("multi-seed scenario: runs in gate F, not in the short or race runs (testing.md 9.1)")
	}
	base := loadValidPack(t, packOneSubsistence...)
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		off := subsistenceEntered(runOrg(t, base, seed, 120))
		for _, bound := range []bool{false, true} {
			cfg := loadPackBytes(t, fmt.Appendf(nil, ownUseRightDoc, bound), packOneSubsistence...)
			events := runOrg(t, cfg, seed, 120)
			entered, refused := subsistenceEntered(events), refusedProduce(events)
			t.Logf("seed %d, bound %v: %d households entered the mode (%d without the module), %d starts refused", seed, bound, entered, off, refused)
			if !bound && (entered != off || refused != 0) {
				t.Errorf("seed %d, OwnUseBound false: entered %d (want %d), refused %d (want 0)", seed, entered, off, refused)
			}
			if bound && refused == 0 {
				t.Errorf("seed %d, OwnUseBound true: refused = 0, want the prohibition to bind own use", seed)
			}
		}
	}
}

// refusedProduce is the sum of RightRefused.Produce of a run.
func refusedProduce(events []economysim.Event) (n int64) {
	for _, e := range events {
		if x, ok := e.(economysim.RightRefused); ok {
			n += x.Produce
		}
	}
	return n
}
