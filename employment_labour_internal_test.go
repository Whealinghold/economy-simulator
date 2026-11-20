package economysim

import (
	"log/slog"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	payroll "github.com/Whealinghold/economy-sim/internal/employment"
)

// labourLog records the events a test reads: every event in order, and the LabourRound of
// each tick by region.
type labourLog struct {
	hireLog
	rounds []LabourRound
}

func (l *labourLog) Observe(e Event) {
	l.hireLog.Observe(e)
	if r, ok := e.(LabourRound); ok {
		l.rounds = append(l.rounds, r)
	}
}

// reserveByAgent returns a plan in which a farm posts positions at bid and the hand of agent
// ID n offers one household at reserves[n], hands with no entry not offering.
func reserveByAgent(positions int64, bid Price, reserves map[AgentID]Price) func(v *AgentView) JobRequest {
	return func(v *AgentView) JobRequest {
		if v.Role == 0 {
			return JobRequest{NoJob: true, Hire: HireRequest{Recipe: 0, Positions: positions, Bid: bid}}
		}
		if r, ok := reserves[v.Agent]; ok {
			return JobRequest{NoJob: true, Offer: LabourOffer{Households: 1, Reserve: r}}
		}
		return JobRequest{NoJob: true}
	}
}

// TestLabourRound_Counts verifies the counts of one round (spec-employment.md 15.1): three
// hands offer one household each, two at a reserve of 0 and one at 200, to a farm that posts
// two positions at a bid of 100. The two cheap hands are hired at a wage of 0 + 100 * 5 000
// / 10 000 = 50 and the dear one has no vacancy it can take (its reserve is above the bid).
// The wage of the tick is released at P2.7 for two households, 2 * 50 = 100, and the
// rows are by role: the hands are role 1, the farm role 0.
//
// Requirement: FR-EMP-053, FR-EMP-060
func TestLabourRound_Counts(t *testing.T) {
	t.Parallel()
	log := &labourLog{}
	w := hireWorld(t, defaultHire(hireOpts{hands: 3}), reserveByAgent(2, 100, map[AgentID]Price{2: 0, 3: 0, 4: 200}), WithObserver(log))
	stepN(t, w, 1)
	want := LabourRound{Tick: 0, Region: 0, Offered: 3, Hired: 2, NoVacancy: 1, Posted: 2, Filled: 2, Matched: 2,
		HouseholdTicks: 2, Paid: 100, Workers: []LabourRow{{Role: 1, Offered: 3, Hired: 2, NoVacancy: 1}},
		Employers: []VacancyRow{{Role: 0, Posted: 2, Filled: 2}}}
	if len(log.rounds) != 1 {
		t.Fatalf("LabourRound events after one step = %d, want 1", len(log.rounds))
	}
	got := log.rounds[0]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LabourRound = %+v, want %+v", got, want)
	}
}

// slicesEqual compares two slices of comparable rows.
func slicesEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestLabourRound_OncePerRegionOnlyWhenCountedAscending verifies that a LabourRound is
// emitted once per region in ascending RegionID in each tick in which a count was made, and
// none in a tick in which nothing was offered, posted or paid (spec-employment.md 15.1,
// 15.1.2).
//
// Setup: two regions, each with one farm and one hand. The plan acts at tick 0 only: a hire
// of 3 ticks forms in each region and is paid at ticks 0, 1 and 2 (P2.7, then P2.1 twice),
// so the three ticks have a round per region, and tick 3 has none.
//
// Requirement: FR-EMP-053
func TestLabourRound_OncePerRegionOnlyWhenCountedAscending(t *testing.T) {
	t.Parallel()
	log := &labourLog{}
	plan := func(v *AgentView) JobRequest {
		if v.Tick.Tick > 0 {
			return JobRequest{NoJob: true}
		}
		return farmWants(1, 100, 0)(v)
	}
	w := hireWorld(t, defaultHire(hireOpts{regions: 2}), plan, WithObserver(log))
	stepN(t, w, 4)
	var got [][2]int64
	for _, r := range log.rounds {
		got = append(got, [2]int64{r.Tick, int64(r.Region)})
	}
	want := [][2]int64{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 0}, {2, 1}}
	if !slicesEqual(got, want) {
		t.Errorf("LabourRound (tick, region) after 4 steps = %v, want %v: ascending regions, none at tick 3", got, want)
	}
	for _, r := range log.rounds[2:] {
		if r.Offered != 0 || r.HouseholdTicks != 1 || r.Paid != 50 {
			t.Errorf("LabourRound(tick %d, region %d) = offered %d, household-ticks %d, paid %d, want 0, 1, 50: a wage of a hire in progress",
				r.Tick, r.Region, r.Offered, r.HouseholdTicks, r.Paid)
		}
	}
}

// TestLabourRound_RefusalFieldsByReason verifies that the positions a failed formation check
// closed are counted in the field of its reason, and that the fields of the modules not
// built stay 0 (spec-employment.md 5.5, 15.1).
//
// Setup: the farm posts 2 positions and one hand offers. With 100 money against a bill of 50
// * 3 = 150 the check of the funds fails (employer_funds); with the recipe needing 1 grain
// that the farm lacks the check of the inputs fails (no_input), first in the order.
//
// Requirement: FR-EMP-023, FR-EMP-060
func TestLabourRound_RefusalFieldsByReason(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		o     hireOpts
		field string
	}{
		{"employer funds", hireOpts{farmMoney: 100}, "EmployerFunds"},
		{"no input", hireOpts{input: true}, "NoInput"},
		{"no tool", hireOpts{tool: true}, "NoTool"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			log := &labourLog{}
			w := hireWorld(t, defaultHire(tt.o), farmWants(2, 100, 0), WithObserver(log))
			stepN(t, w, 1)
			if len(log.rounds) != 1 {
				t.Fatalf("LabourRound events = %d, want 1", len(log.rounds))
			}
			r := log.rounds[0]
			if got := reflect.ValueOf(r).FieldByName(tt.field).Int(); got != 2 || r.Unfilled != 2 || r.Matched != 0 || r.NoVacancy != 1 {
				t.Errorf("LabourRound = %+v, want 2 positions closed in %s, 2 unfilled, none matched, 1 household unmatched", r, tt.field)
			}
			if got := r.NotPermitted + r.NoFacility + r.NotUnlocked + r.FacilityCap + r.NoTarget + r.StoreRefused + r.NothingToRepair; got != 0 {
				t.Errorf("LabourRound fields of the modules not built = %d, want 0: a later module's fill must be a visible change", got)
			}
		})
	}
}

// labourWarnFault says what is wrong with a Warn line of the module, or "": the keys of
// spec-employment.md 15.2.2 with the values of the scenario of its test.
func labourWarnFault(r logRec) string {
	switch r.msg {
	case "labour unmatched":
		if r.attrs["households"] != 1 || r.attrs["role"] != 1 {
			return "want households 1 and role 1 (the hands)"
		}
	case "vacancies unfilled":
		if r.attrs["vacancies"] != 1 || r.attrs["role"] != 0 || r.strs["reason"] != "no_labour" {
			return "want vacancies 1, role 0 and reason no_labour"
		}
	default:
		return "want only the two lines of the module"
	}
	return ""
}

// TestWarn_LabourUnmatchedAndVacanciesUnfilled verifies the two bounded Warn lines of
// spec-employment.md 15.2.2: "labour unmatched" and "vacancies unfilled" are logged at the
// tenth tick of an uninterrupted run and at every tenth after, once per (region, role), with
// the keys of the spec, and a tick in which a hire forms resets the run.
//
// Setup: two regions, a farm that posts one position at a bid of 100 and a hand that offers
// its household at a reserve of 500 every tick: no offer ever reaches the vacancy, so each
// tick the hand is unmatched and the position unfilled for no_labour. The run reaches 10 at
// tick 9 and 20 at tick 19; in the quiet world, where the reserve is 0, there is no line.
//
// Requirement: FR-EMP-056
func TestWarn_LabourUnmatchedAndVacanciesUnfilled(t *testing.T) {
	t.Parallel()
	type line struct {
		tick int64
		rec  logRec
	}
	run := func(reserve Price) []line {
		c := newCapture()
		w := hireWorld(t, defaultHire(hireOpts{regions: 2}), farmWants(1, 100, reserve), WithLogger(slog.New(c)))
		*c.lines = nil // the warnings of the configuration, logged at construction
		var out []line
		for range 21 {
			stepN(t, w, 1)
			for _, l := range *c.lines {
				if l.level >= slog.LevelWarn {
					out = append(out, line{w.Tick() - 1, l})
				}
			}
			*c.lines = nil
		}
		return out
	}
	got := run(500)
	counts := map[string]int{}
	for _, l := range got {
		counts[l.rec.msg]++
		if l.tick != 9 && l.tick != 19 {
			t.Errorf("Warn %q at tick %d, want only the ticks 9 and 19 (the tenth and the twentieth of the run)", l.rec.msg, l.tick)
		}
		if why := labourWarnFault(l.rec); why != "" {
			t.Errorf("Warn %q: %s", l.rec.msg, why)
		}
	}
	if counts["labour unmatched"] != 4 || counts["vacancies unfilled"] != 4 {
		t.Errorf("Warn lines = %v, want 4 of each: 2 regions at 2 ticks", counts)
	}
	if quiet := run(0); len(quiet) != 0 {
		t.Errorf("Warn lines with the hires forming = %d, want 0", len(quiet))
	}
}

// sectorLog rebuilds the money of the household sector of region 0 from the events, as an
// observer that keeps no ledger does (spec-observability.md 4.3.3, spec-employment.md
// 15.1.1): the four items of the hire form and what leaves with a removed agent.
type sectorLog struct {
	money  Money
	causes []Cause // the cause of each HireEnded, so a test can tell the scenario ran
}

func (s *sectorLog) Observe(e Event) {
	switch e := e.(type) {
	case HireFormed:
		s.money -= e.Escrowed
	case LabourRound:
		s.money += e.Paid + e.Refunded
	case HireEnded:
		s.money += e.Refunded
		s.causes = append(s.causes, e.Cause)
	case AgentBankrupted:
		s.money -= e.Money
	}
}

// TestEvents_RebuildMoneyFourItems verifies that the money of the household sector, rebuilt
// from the escrow of HireFormed, the wages and the refunds of LabourRound, the remainder of
// HireEnded and the money of a removed agent, equals the snapshot at the end of every tick
// (spec-employment.md 15.1.1, FR-EMP-054).
//
// Setup: one hire of 3 ticks at a wage of 50, whose worker is removed after the first tick
// (the remainder of 100 returns to the farm, the worker's 50 goes to the estate), and a
// second world where the ration of a scarce endowment cuts both hires to nothing (the
// whole escrow of each returns at once, cause not_started).
//
// Requirement: FR-EMP-054
func TestEvents_RebuildMoneyFourItems(t *testing.T) {
	t.Parallel()
	check := func(t *testing.T, w *World, s *sectorLog, ticks int, afterTick func(i int)) {
		t.Helper()
		for i := range ticks {
			stepN(t, w, 1)
			if afterTick != nil {
				afterTick(i)
			}
			snap, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() = %v, want nil", err)
			}
			if got := snap.Regions[0].MoneyHouseholds; got != s.money {
				t.Fatalf("tick %d: MoneyHouseholds = %d, rebuilt from the events %d", i, got, s.money)
			}
		}
	}
	t.Run("a worker removed mid-job", func(t *testing.T) {
		t.Parallel()
		s := &sectorLog{}
		w := hireWorld(t, defaultHire(hireOpts{}), farmWants(1, 100, 0), WithObserver(s))
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		s.money = snap.Regions[0].MoneyHouseholds
		check(t, w, s, 4, func(i int) {
			if i == 0 {
				if err := w.removeAgent(&w.agents[slotsOf(w, "hand")[0]], CauseConsumer); err != nil {
					t.Fatalf("removeAgent() = %v, want nil", err)
				}
			}
		})
		if !slices.Equal(s.causes, []Cause{CauseWorkerLeft}) {
			t.Errorf("HireEnded causes = %v, want [worker_left]: the scenario must end the hire", s.causes)
		}
	})
	t.Run("a ration that cuts a hire to nothing", func(t *testing.T) {
		t.Parallel()
		s := &sectorLog{}
		// 4 ore for two requests of 3 each: the ration leaves one of them under one execution.
		w := hireWorld(t, defaultHire(hireOpts{hands: 2, ore: 4}), farmWants(2, 100, 0), WithObserver(s))
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		s.money = snap.Regions[0].MoneyHouseholds
		check(t, w, s, 2, nil)
		if !slices.Contains(s.causes, CauseNotStarted) {
			t.Errorf("HireEnded causes = %v, want a not_started: the scenario must cut a hire to nothing", s.causes)
		}
	})
}

// TestCost_EmploymentNoopAllocs verifies NFR-EMP-001 for the aggregate: with the default
// logger and the no-op observer, P2.8b of a region that counted offers, vacancies and
// refusals, and whose two runs are at the ninth tick (so both Warn lines are due), allocates
// nothing, because the event, its rows and the log attributes are built only when an
// observer or an enabled logger asks for them (observability-and-logging.md 4.5, 8.2).
//
// Requirement: NFR-EMP-001
func TestCost_EmploymentNoopAllocs(t *testing.T) {
	// No t.Parallel: AllocsPerRun changes GOMAXPROCS while it measures.
	w := hireWorld(t, defaultHire(hireOpts{hands: 2}), farmWants(2, 100, 0))
	stepN(t, w, 1) // grows the scratch of the tick
	e := w.employment
	nr := len(w.cfg.Roles)
	got := testing.AllocsPerRun(20, func() {
		rl := &e.tallyOf(0, w.Tick()).roles[1]
		*rl = roleLabour{offered: 2, noVacancy: 2, posted: 1, unfilled: 1}
		e.unmatchedRun[1], e.unfilledRun[0] = warnRun-1, warnRun-1
		w.p28bLabour()
	})
	if got != 0 {
		t.Errorf("allocs of P2.8b with the no-op observer and logger = %v, want 0 (roles %d)", got, nr)
	}
}

// TestWork_IsCounted verifies NFR-EMP-002: the labour round keeps and admits only the
// offers and vacancies that Work returned, so its work is proportional to them and not to
// the agents. With one farm posting and one hand offering, a world of 200 hands and a world
// of 2 keep the same 2 requests at P2.5, admit them and form the same one hire.
//
// Requirement: NFR-EMP-002
func TestWork_IsCounted(t *testing.T) {
	t.Parallel()
	for _, hands := range []int64{2, 200} {
		plan := func(v *AgentView) JobRequest {
			if v.Role == 0 || v.Agent == 2 { // the farm and one hand act, the rest idle
				return farmWants(1, 100, 0)(v)
			}
			return JobRequest{NoJob: true}
		}
		w := hireWorld(t, defaultHire(hireOpts{hands: hands}), plan)
		at := slices.IndexFunc(replayRows, func(r replayRow) bool { return r.id == "P2.5a" })
		if err := runRows(w, w.Tick(), replayRows[:at], func(string) {}); err != nil {
			t.Fatalf("positions up to P2.5: %v", err)
		}
		if got := len(w.employment.reqs); got != 2 {
			t.Errorf("requests kept at P2.5 with %d hands = %d, want 2: the farm's and one hand's", hands, got)
		}
		w.p25aLabour()
		if got := len(w.employment.hires); got != 1 {
			t.Errorf("hires after the round with %d hands = %d, want 1", hands, got)
		}
		if got := len(w.employment.lanes[0].offers) + len(w.employment.lanes[0].vacs); got != 0 {
			t.Errorf("scratch lanes after the round with %d hands = %d entries, want 0 (cleared)", hands, got)
		}
	}
}

// TestCardinality_LabourReasons verifies that the formation refusals the labour round
// counts are a closed set of eleven reasons, each a Reason constant of event.go and each
// named in the cardinality rule of observability-and-logging.md 5.4.1, which bounds the
// attribute economysim.reason of the instrument economysim.vacancy.unfilled
// (spec-employment.md 15.3).
//
// Requirement: FR-EMP-057
func TestCardinality_LabourReasons(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("doc/rules/observability-and-logging.md")
	if err != nil {
		t.Fatalf("read the rules file: %v", err)
	}
	text := string(raw)
	seen := map[Reason]bool{}
	for code := payroll.ReasonTooMany; code < numLabourReasons; code++ {
		r := labourReasons[code]
		if r == "" || seen[r] {
			t.Errorf("labourReasons[%d] = %q, want a distinct reason", code, r)
		}
		seen[r] = true
		if !strings.Contains(text, "`"+string(r)+"`") {
			t.Errorf("reason %q is not named in observability-and-logging.md", r)
		}
	}
	if len(seen) != 11 {
		t.Errorf("labour reasons = %d, want 11", len(seen))
	}
}

// TestMoreThanHalf_Boundaries verifies the condition of the Warn "labour unmatched": strictly
// more than half of the households offered were left unmatched, so exactly half is not,
// and a round with no offer has no half (spec-employment.md 15.2.2).
//
// Requirement: FR-EMP-056
func TestMoreThanHalf_Boundaries(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		part, whole int64
		want        bool
	}{{0, 0, false}, {1, 1, true}, {1, 2, false}, {2, 3, true}, {2, 4, false}, {3, 5, true}, {2, 5, false}} {
		if got := moreThanHalf(tt.part, tt.whole); got != tt.want {
			t.Errorf("moreThanHalf(%d, %d) = %v, want %v", tt.part, tt.whole, got, tt.want)
		}
	}
}
